package backup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/google/uuid"
)

type RemoteCopy struct {
	Format    int       `json:"format"`
	ID        string    `json:"id"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"createdAt"`
}

func digestReader(reader io.Reader, limit int64) (string, int64, error) {
	if limit < 0 || limit >= 1<<60 {
		return "", 0, fmt.Errorf("invalid file size")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(reader, limit+1))
	return fmt.Sprintf("%x", h.Sum(nil)), n, err
}
func ReadMarker(ctx context.Context, client *smb.Client, id string) (RemoteCopy, error) {
	if copyFormat(id) != 3 {
		return RemoteCopy{}, fmt.Errorf("invalid v3 copy identifier")
	}
	return readCopyMarker(ctx, client, id)
}
func RemoteCopies(ctx context.Context, client *smb.Client) ([]RemoteCopy, error) {
	entries, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []RemoteCopy{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".manifest.json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".manifest.json")
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		m, err := ReadMarker(ctx, client, id)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func DownloadCopy(ctx context.Context, client *smb.Client, id, directory string, maxBytes int64) (string, error) {
	release, err := client.Lock(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	lock := "." + id + ".restore-lock"
	if _, err := uuid.Parse(id); err != nil {
		return "", err
	}
	if err := client.Write(ctx, lock, strings.NewReader("download")); err != nil {
		return "", fmt.Errorf("backup is busy: %w", err)
	}
	defer client.Remove(context.WithoutCancel(ctx), lock)
	marker, err := ReadMarker(ctx, client, id)
	if err != nil {
		return "", err
	}
	if marker.Size > maxBytes {
		return "", fmt.Errorf("archive exceeds staging limit")
	}
	input, err := client.Open(ctx, id+".tar.gz")
	if err != nil {
		return "", err
	}
	defer input.Close()
	name := filepath.Join(directory, id+".tar.gz")
	file, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(file, h), io.LimitReader(input, marker.Size+1))
	syncErr := file.Sync()
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n != marker.Size || fmt.Sprintf("%x", h.Sum(nil)) != marker.SHA256 {
		return "", fmt.Errorf("remote archive checksum mismatch")
	}
	return name, nil
}
