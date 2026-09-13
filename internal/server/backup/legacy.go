package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UnpackV2 reads existing script archives, requiring their original sidecar.
func UnpackV2(ctx context.Context, archive, directory string, maxBytes int64) (Manifest, error) {
	m := Manifest{Format: 2, ID: strings.TrimSuffix(filepath.Base(archive), ".tar.gz"), CreatedAt: time.Now().UTC(), Objects: []Object{}}
	if !regexp.MustCompile(`^backup_[0-9]{8}_[0-9]{6}(_[0-9]{9})?$`).MatchString(m.ID) {
		return m, fmt.Errorf("invalid legacy archive name")
	}
	file, err := os.Open(archive + ".manifest")
	if err != nil {
		return m, err
	}
	raw, err := io.ReadAll(io.LimitReader(file, 8193))
	file.Close()
	if err != nil {
		return m, err
	}
	if len(raw) > 8192 {
		return m, fmt.Errorf("oversized legacy manifest")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || fields[key] != "" {
			return m, fmt.Errorf("invalid legacy manifest")
		}
		fields[key] = value
	}
	if fields["format_version"] != "2" || fields["archive"] != filepath.Base(archive) {
		return m, fmt.Errorf("invalid legacy manifest identity")
	}
	size, err := strconv.ParseInt(fields["size_bytes"], 10, 64)
	if err != nil || size < 0 || size > maxBytes {
		return m, fmt.Errorf("invalid legacy archive size")
	}
	checksum, actual, err := FileDigest(archive)
	if err != nil {
		return m, err
	}
	if actual != size || checksum != fields["sha256"] {
		return m, fmt.Errorf("legacy archive checksum mismatch")
	}
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
	var used int64
	seen := map[string]bool{}
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
		name := strings.TrimSuffix(header.Name, "/")
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\:\x00") || (name != "database.dump" && name != "objects" && !strings.HasPrefix(name, "objects/")) {
			return m, fmt.Errorf("unsafe legacy archive path")
		}
		if header.Typeflag == tar.TypeDir {
			if name == "database.dump" {
				return m, fmt.Errorf("invalid dump member")
			}
			if err = root.MkdirAll(name, 0700); err != nil {
				return m, err
			}
			continue
		}
		if len(seen) >= 100002 || header.Typeflag != tar.TypeReg || seen[name] || header.Size < 0 || header.Size > maxBytes-used {
			return m, fmt.Errorf("invalid legacy archive member")
		}
		seen[name] = true
		used += header.Size
		if err = root.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return m, err
		}
		out, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return m, err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return m, copyErr
		}
		if closeErr != nil {
			return m, closeErr
		}
	}
	if err = drainTarPadding(gz); err != nil {
		return m, err
	}
	m.DatabaseSHA256, m.DatabaseSize, err = FileDigest(filepath.Join(directory, "database.dump"))
	if err != nil {
		return m, err
	}
	err = fs.WalkDir(root.FS(), "objects", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		digest, size, err := FileDigest(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		var head [512]byte
		n, e := file.Read(head[:])
		file.Close()
		if e != nil && e != io.EOF {
			return e
		}
		m.Objects = append(m.Objects, Object{Key: strings.TrimPrefix(name, "objects/"), File: name, Size: size, SHA256: digest, ContentType: http.DetectContentType(head[:n])})
		return nil
	})
	return m, err
}
