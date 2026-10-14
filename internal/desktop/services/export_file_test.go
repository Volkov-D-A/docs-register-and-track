package services

import (
	"errors"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestExportFilePreservesExistingFileAndRemovesFailedDownload(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "report.xlsx")
	require.NoError(t, os.WriteFile(original, []byte("existing"), 0600))
	path, err := writeExportFile(dir, "report.xlsx", func(file *os.File) error { _, err := file.WriteString("new report"); return err })
	require.NoError(t, err)
	require.NotEqual(t, original, path)
	bytes, err := os.ReadFile(original)
	require.NoError(t, err)
	require.Equal(t, "existing", string(bytes))
	sentinel := errors.New("interrupted")
	_, err = writeExportFile(dir, "partial.pdf", func(file *os.File) error {
		_, err := file.WriteString("partial")
		require.NoError(t, err)
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	_, err = os.Stat(filepath.Join(dir, "partial.pdf"))
	require.True(t, os.IsNotExist(err))
}
