package backup

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

var legacyCopyID = regexp.MustCompile(`^backup_[0-9]{8}_[0-9]{6}(_[0-9]{9})?$`)

func copyFormat(id string) int {
	if parsed, err := uuid.Parse(id); err == nil && parsed.String() == id {
		return 3
	}
	if legacyCopyID.MatchString(id) {
		return 2
	}
	return 0
}

type remoteReader interface {
	List(context.Context) ([]os.FileInfo, error)
	Open(context.Context, string) (io.ReadCloser, error)
}

func markerName(id string) string {
	if copyFormat(id) == 2 {
		return id + ".tar.gz.manifest"
	}
	return id + ".manifest.json"
}

func parseCopyMarker(id string, raw []byte) (RemoteCopy, error) {
	m := RemoteCopy{ID: id, Format: copyFormat(id)}
	if m.Format == 0 || len(raw) > 8192 {
		return m, fmt.Errorf("invalid copy manifest")
	}
	if m.Format == 3 {
		if len(raw) > 4096 {
			return m, fmt.Errorf("oversized remote manifest")
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return m, err
		}
		if m.ID != id || m.Format != 3 {
			return m, fmt.Errorf("invalid remote manifest identity")
		}
	} else {
		fields := map[string]string{}
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if _, duplicate := fields[key]; !ok || duplicate {
				return m, fmt.Errorf("invalid legacy manifest")
			}
			fields[key] = value
		}
		if fields["format_version"] != "2" || fields["archive"] != id+".tar.gz" {
			return m, fmt.Errorf("invalid legacy manifest identity")
		}
		var err error
		m.Size, err = strconv.ParseInt(fields["size_bytes"], 10, 64)
		if err != nil {
			return m, err
		}
		m.SHA256 = fields["sha256"]
		m.CreatedAt, err = time.Parse("20060102_150405", strings.TrimPrefix(id, "backup_")[:15])
		if err != nil {
			return m, err
		}
	}
	digest, err := hex.DecodeString(m.SHA256)
	if err != nil || len(digest) != 32 || m.Size < 0 || m.Size >= 1<<60 || m.CreatedAt.IsZero() {
		return m, fmt.Errorf("invalid remote manifest metadata")
	}
	return m, nil
}

func readCopyMarker(ctx context.Context, remote remoteReader, id string) (RemoteCopy, error) {
	if copyFormat(id) == 0 {
		return RemoteCopy{}, fmt.Errorf("invalid copy identifier")
	}
	f, err := remote.Open(ctx, markerName(id))
	if err != nil {
		return RemoteCopy{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil {
		return RemoteCopy{}, err
	}
	return parseCopyMarker(id, raw)
}

// Catalog never treats a manifest or a matching size as a full verification.
// Orphan archives and invalid manifests remain visible to the administrator.
func Catalog(ctx context.Context, remote remoteReader) ([]models.BackupCopy, error) {
	entries, err := remote.List(ctx)
	if err != nil {
		return nil, err
	}
	files := map[string]os.FileInfo{}
	ids := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files[entry.Name()] = entry
		for _, suffix := range []string{".tar.gz.manifest", ".manifest.json", ".delete.json", ".tar.gz"} {
			if strings.HasSuffix(entry.Name(), suffix) {
				id := strings.TrimSuffix(entry.Name(), suffix)
				if copyFormat(id) != 0 {
					ids[id] = true
				}
				break
			}
		}
	}
	out := make([]models.BackupCopy, 0, len(ids))
	for id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		copy := models.BackupCopy{ID: id, Format: copyFormat(id), Verification: "incomplete"}
		archive, exists := files[id+".tar.gz"]
		if exists {
			copy.Size = archive.Size()
			copy.CreatedAt = archive.ModTime()
		}
		m, err := readCopyMarker(ctx, remote, id)
		switch {
		case err != nil:
			copy.Issue = "Manifest отсутствует или повреждён"
		case !exists:
			copy.CreatedAt = m.CreatedAt
			copy.Size = m.Size
			copy.SHA256 = m.SHA256
			copy.Issue = "Архив отсутствует"
		case archive.Size() != m.Size:
			copy.Issue = "Размер архива не совпадает с manifest"
		default:
			copy.CreatedAt = m.CreatedAt
			copy.Size = m.Size
			copy.SHA256 = m.SHA256
			copy.Verification = "unverified"
			copy.CanDelete = copy.Format == 3
			if copy.Format == 2 {
				copy.Issue = "Удаление копий v2 через панель не поддерживается"
			}
		}
		if _, deleting := files[id+".delete.json"]; deleting {
			copy.Verification = "deleting"
			copy.CanDelete = copy.Format == 3
			copy.Issue = "Удаление не завершено; повторите операцию"
			if m, err := readDeletion(ctx, remote, id); err == nil {
				copy.CreatedAt = m.CreatedAt
				copy.Size = m.Size
				copy.SHA256 = m.SHA256
			}
		}
		out = append(out, copy)
		if m, err := readCopyMarker(ctx, remote, id); err == nil {
			out[len(out)-1].RestoreConfirmation = operationConfirmation("restore", m)
			out[len(out)-1].DeleteConfirmation = operationConfirmation("delete", m)
		} else if m, err := readDeletion(ctx, remote, id); err == nil {
			out[len(out)-1].DeleteConfirmation = operationConfirmation("delete", m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Service) Catalog(ctx context.Context) ([]models.BackupCopy, error) {
	if err := s.Ready(); err != nil {
		return nil, err
	}
	cfg, err := (SettingsRepository{s.DB}).Load(ctx)
	if err != nil {
		return nil, err
	}
	password, err := s.Box.Decrypt(cfg.Secret)
	if err != nil {
		return nil, err
	}
	client, err := smb.Open(ctx, smb.Config(cfg.Settings.SMB), password)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	copies, err := Catalog(ctx, client)
	if err != nil {
		return nil, err
	}
	ops, err := s.operations()
	if err != nil {
		return nil, err
	}
	for i := range copies {
		copy := &copies[i]
		if copy.Verification != "unverified" {
			continue
		}
		for _, op := range ops {
			if op.Kind != "verify" || op.CopyID != copy.ID || op.Target.Settings.SMB != cfg.Settings.SMB || op.Copy.SHA256 != copy.SHA256 || op.Copy.Size != copy.Size || !op.Copy.CreatedAt.Equal(copy.CreatedAt) {
				continue
			}
			if op.State == "completed" {
				copy.Verification = "verified"
				copy.Issue = strings.TrimSpace(copy.Issue + " Последняя полная проверка: " + op.UpdatedAt.UTC().Format(time.RFC3339))
			}
			break
		}
	}
	return copies, nil
}
