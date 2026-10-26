package backup

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/server/backup/smb"
	"github.com/google/uuid"
)

type RemoteCopy struct {
	Format    int       `json:"format"`
	ID        string    `json:"id"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"createdAt"`
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
		m, err := readCopyMarker(ctx, client, id)
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
	marker, err := readCopyMarker(ctx, client, id)
	if err != nil {
		return "", err
	}
	return downloadSelected(ctx, client, marker, directory, maxBytes)
}
