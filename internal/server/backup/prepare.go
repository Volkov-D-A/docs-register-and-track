package backup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PreparedRestore contains only validated members in a private staging directory.
type PreparedRestore struct {
	Manifest  Manifest
	Directory string
}

func PrepareRestore(ctx context.Context, pg PostgreSQL, archive, directory string, maxBytes int64, schema int) (PreparedRestore, error) {
	var m Manifest
	var err error
	if copyFormat(strings.TrimSuffix(filepath.Base(archive), ".tar.gz")) == 2 {
		m, err = UnpackV2(ctx, archive, directory, maxBytes)
	} else {
		m, err = Unpack(ctx, archive, directory, maxBytes)
	}
	if err != nil {
		return PreparedRestore{}, err
	}
	if m.Schema > schema {
		return PreparedRestore{}, fmt.Errorf("backup requires a newer application")
	}
	if err = pg.ValidateDump(ctx, filepath.Join(directory, "database.dump")); err != nil {
		return PreparedRestore{}, err
	}
	dumpSchema, err := pg.DumpSchema(ctx, filepath.Join(directory, "database.dump"))
	if err != nil {
		return PreparedRestore{}, err
	}
	if dumpSchema > schema || (m.Schema != 0 && m.Schema != dumpSchema) {
		return PreparedRestore{}, fmt.Errorf("dump schema is incompatible with manifest or application")
	}
	m.Schema = dumpSchema
	return PreparedRestore{Manifest: m, Directory: directory}, nil
}

// downloadSelected is called with the common SMB lock held. The expected
// manifest is captured before queueing, so replacement on the NAS is rejected.
func downloadSelected(ctx context.Context, remote remoteReader, expected RemoteCopy, directory string, maxBytes int64) (string, error) {
	actual, err := readCopyMarker(ctx, remote, expected.ID)
	if err != nil {
		return "", err
	}
	if !sameCopy(actual, expected) {
		return "", fmt.Errorf("выбранная копия изменилась на SMB; обновите каталог")
	}
	if actual.Size > maxBytes {
		return "", fmt.Errorf("archive exceeds staging limit")
	}
	archive := filepath.Join(directory, actual.ID+".tar.gz")
	in, err := remote.Open(ctx, actual.ID+".tar.gz")
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(in, actual.Size+1))
	syncErr, closeErr := out.Sync(), out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if syncErr != nil {
		return "", syncErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n != actual.Size || fmt.Sprintf("%x", hash.Sum(nil)) != actual.SHA256 {
		return "", fmt.Errorf("remote archive checksum mismatch")
	}
	if actual.Format == 2 {
		in, err := remote.Open(ctx, markerName(actual.ID))
		if err != nil {
			return "", err
		}
		defer in.Close()
		raw, err := io.ReadAll(io.LimitReader(in, 8193))
		if err != nil {
			return "", err
		}
		marker, err := parseCopyMarker(actual.ID, raw)
		if err != nil || !sameCopy(marker, actual) {
			return "", fmt.Errorf("legacy manifest changed")
		}
		if err = durableMarker(archive+".manifest", raw); err != nil {
			return "", err
		}
	}
	return archive, syncDirectory(directory)
}

func sameCopy(a, b RemoteCopy) bool {
	return a.ID == b.ID && a.Format == b.Format && a.Size == b.Size && a.SHA256 == b.SHA256 && a.CreatedAt.Equal(b.CreatedAt)
}
