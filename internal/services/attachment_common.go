package services

import (
	"path/filepath"
	"reflect"
	"strings"
)

// Both HTTP upload metadata and local downloads use the same filename rules.
func safeDownloadFilename(filename string) string {
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

// Interface dependencies must also reject a typed nil pointer.
func attachmentDependencyMissing(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
