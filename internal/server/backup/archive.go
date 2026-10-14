package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const ArchiveFormat = 4

type Manifest struct {
	Format         int       `json:"format"`
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	Version        string    `json:"version"`
	Schema         int       `json:"schema"`
	DatabaseSHA256 string    `json:"databaseSha256"`
	DatabaseSize   int64     `json:"databaseSize"`
}

func FileDigest(name string) (string, int64, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	return fmt.Sprintf("%x", h.Sum(nil)), size, err
}
func writeJSONFile(name string, value any) error {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(value)
	if err == nil {
		err = file.Sync()
	}
	closed := file.Close()
	if err != nil {
		return err
	}
	return closed
}

func Pack(ctx context.Context, directory, archive string, m Manifest) error {
	file, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	names := []string{"manifest.json", "database.dump"}
	for _, name := range names {
		if err = ctx.Err(); err != nil {
			break
		}
		err = func() error {
			source, err := os.Open(filepath.Join(directory, name))
			if err != nil {
				return err
			}
			defer source.Close()
			info, err := source.Stat()
			if err != nil {
				return err
			}
			if err = tw.WriteHeader(&tar.Header{Name: name, Size: info.Size(), Mode: 0600, ModTime: m.CreatedAt}); err != nil {
				return err
			}
			_, err = io.Copy(tw, source)
			return err
		}()
		if err != nil {
			break
		}
	}
	for _, close := range []func() error{tw.Close, gz.Close, file.Sync, file.Close} {
		e := close()
		if err == nil {
			err = e
		}
	}
	return err
}

// Unpack verifies a database-only v4 archive into a new directory, with bounded expanded size.
func Unpack(ctx context.Context, archive, directory string, maxBytes int64) (Manifest, error) {
	var m Manifest
	source, err := os.Open(archive)
	if err != nil {
		return m, err
	}
	defer source.Close()
	gz, err := gzip.NewReader(source)
	if err != nil {
		return m, err
	}
	defer gz.Close()
	if err = os.Mkdir(directory, 0700); err != nil {
		return m, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return m, err
	}
	defer root.Close()
	tr := tar.NewReader(gz)
	seen := map[string]bool{}
	var used int64
	for {
		if err = ctx.Err(); err != nil {
			return m, err
		}
		header, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return m, e
		}
		name := header.Name
		allowed := name == "manifest.json" || name == "database.dump"
		if len(seen) >= 2 || !allowed || header.Typeflag != tar.TypeReg || seen[name] || header.Size < 0 || header.Size > maxBytes-used {
			return m, fmt.Errorf("invalid or oversized archive member")
		}
		if name == "manifest.json" && header.Size > 16<<20 {
			return m, fmt.Errorf("manifest exceeds limit")
		}
		seen[name] = true
		used += header.Size
		if err = root.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return m, err
		}
		file, e := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return m, e
		}
		_, e = io.Copy(file, tr)
		closeErr := file.Close()
		if e != nil {
			return m, e
		}
		if closeErr != nil {
			return m, closeErr
		}
	}
	if err = drainTarPadding(gz); err != nil {
		return m, err
	}
	raw, err := root.ReadFile("manifest.json")
	if err != nil {
		return m, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&m); err != nil {
		return m, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return m, fmt.Errorf("manifest должен содержать один JSON-объект")
	}
	if _, err = uuid.Parse(m.ID); err != nil {
		return m, fmt.Errorf("invalid backup identifier")
	}
	if m.Format != ArchiveFormat || m.ID == "" || m.Schema < 1 || len(seen) != 2 {
		return m, fmt.Errorf("несовместимый архив: требуется формат 4 без объектного хранилища")
	}
	if err = VerifyFiles(directory, m); err != nil {
		return m, err
	}
	return m, nil
}
func VerifyFiles(directory string, m Manifest) error {
	digest, size, err := FileDigest(filepath.Join(directory, "database.dump"))
	if err != nil {
		return err
	}
	if digest != m.DatabaseSHA256 || size != m.DatabaseSize {
		return fmt.Errorf("database dump checksum mismatch")
	}
	return nil
}

func drainTarPadding(reader io.Reader) error {
	raw, err := io.ReadAll(io.LimitReader(reader, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return fmt.Errorf("oversized archive padding")
	}
	for _, value := range raw {
		if value != 0 {
			return fmt.Errorf("unexpected archive trailer")
		}
	}
	return nil
}
