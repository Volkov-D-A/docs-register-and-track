package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type remoteWriter interface {
	remoteReader
	Write(context.Context, string, io.Reader) error
	Remove(context.Context, string) error
}

func (s *Service) deleteOperation(ctx context.Context, client remoteWriter, op *operation) error {
	if err := s.operationPhase(op, "deleting", false); err != nil {
		return err
	}
	if err := s.deleteRemote(ctx, client, op.Copy, op.Target.Settings.KeepCopies); err != nil {
		return err
	}
	if err := s.markDeleted(op.CopyID); err != nil {
		return err
	}
	return s.operationPhase(op, "completed", false)
}

func (s *Service) markDeleted(id string) error {
	jobs, err := s.jobs()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.ID == id {
			job.State = "deleted"
			if err = s.persist(&job); err != nil {
				return err
			}
		}
	}
	return nil
}

func readDeletion(ctx context.Context, client remoteReader, id string) (RemoteCopy, error) {
	f, err := client.Open(ctx, id+".delete.json")
	if err != nil {
		return RemoteCopy{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return RemoteCopy{}, err
	}
	return parseCopyMarker(id, raw)
}

// deleteRemote and retention use the same global SMB lock and tombstone.
// The archive goes first; a crash never leaves it appearing as a healthy set.
func (s *Service) deleteRemote(ctx context.Context, client remoteWriter, copy RemoteCopy, minimum int) error {
	if copyFormat(copy.ID) != 3 || copy.Format != 3 {
		return fmt.Errorf("удаление v2 не поддерживается")
	}
	if minimum < 1 {
		minimum = 1
	}
	tombstone := copy.ID + ".delete.json"
	deleting, err := readDeletion(ctx, client, copy.ID)
	if err == nil {
		if !sameCopy(copy, deleting) {
			return fmt.Errorf("deletion identity mismatch")
		}
	} else {
		if !os.IsNotExist(err) {
			return err
		}
		actual, err := readCopyMarker(ctx, client, copy.ID)
		if err != nil {
			return err
		}
		if !sameCopy(copy, actual) {
			return fmt.Errorf("copy changed before deletion")
		}
		copies, err := Catalog(ctx, client)
		if err != nil {
			return err
		}
		valid := 0
		for _, candidate := range copies {
			if candidate.ID == copy.ID || candidate.Verification == "incomplete" || candidate.Verification == "deleting" {
				continue
			}
			// Check complete archives, including members and dump, rather than
			// counting manifests or relying on history from a previous server.
			m, err := readCopyMarker(ctx, client, candidate.ID)
			if err != nil {
				continue
			}
			work, err := os.MkdirTemp(s.Directory, "retention-check-")
			if err != nil {
				return err
			}
			check := func() error {
				archive, err := downloadSelected(ctx, client, m, work, s.MaxBytes/8)
				if err != nil {
					return err
				}
				_, err = PrepareRestore(ctx, s.PostgreSQL, archive, filepath.Join(work, "contents"), s.MaxBytes/8, s.Schema)
				return err
			}()
			os.RemoveAll(work)
			if check == nil {
				valid++
			}
			if valid >= minimum {
				break
			}
		}
		if valid < minimum {
			return fmt.Errorf("удаление запрещено: требуется сохранить минимум %d проверенных копий, найдены %d других", minimum, valid)
		}
		raw, err := json.Marshal(copy)
		if err != nil {
			return err
		}
		if err = client.Write(ctx, tombstone, bytes.NewReader(raw)); err != nil {
			return err
		}
	}
	for _, name := range []string{copy.ID + ".tar.gz", copy.ID + ".manifest.json"} {
		if err = client.Remove(ctx, name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return client.Remove(ctx, tombstone)
}
