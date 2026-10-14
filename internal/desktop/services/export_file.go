package services

import (
	"errors"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/attachmentname"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func writeExportFile(downloadDir, filename string, write func(*os.File) error) (string, error) {
	cleanFilename := attachmentname.Normalize(filename)
	ext := filepath.Ext(cleanFilename)
	base := strings.TrimSuffix(cleanFilename, ext)
	for i := 0; i < 1000; i++ {
		candidate := cleanFilename
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		fullPath := filepath.Join(downloadDir, candidate)
		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := write(file); err != nil {
			_ = file.Close()
			_ = os.Remove(fullPath)
			return "", err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(fullPath)
			return "", err
		}
		return fullPath, nil
	}
	return "", fmt.Errorf("failed to choose unique download filename for %q", cleanFilename)
}

func defaultDownloadDir() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	return filepath.Join(currentUser.HomeDir, "Downloads"), nil
}
