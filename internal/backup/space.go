package backup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (s *Service) checkStagingSpace() error {
	if err := os.MkdirAll(s.Directory, 0700); err != nil {
		return err
	}
	var used int64
	err := filepath.WalkDir(s.Directory, func(_ string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed in staging")
		}
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				return err
			}
			if info.Size() > s.MaxBytes-used {
				return fmt.Errorf("staging limit exceeded")
			}
			used += info.Size()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if used > s.MaxBytes/3 {
		return fmt.Errorf("недостаточно места в лимите staging; сохранённые архивы требуют проверки оператором")
	}
	free, err := freeBytes(s.Directory)
	if err != nil {
		return err
	}
	if free < uint64(s.MaxBytes*2/3) {
		return fmt.Errorf("недостаточно свободного места для источника и страховочной копии")
	}
	return nil
}
