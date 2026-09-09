package attachmentname

import (
	"path/filepath"
	"strings"
)

// Normalize applies the filename rules shared by uploads and local downloads.
func Normalize(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	cleanFilename := filepath.Base(strings.TrimSpace(filename))
	cleanFilename = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, cleanFilename)
	if cleanFilename == "" || cleanFilename == "." || cleanFilename == string(filepath.Separator) {
		return "attachment"
	}
	return cleanFilename
}
