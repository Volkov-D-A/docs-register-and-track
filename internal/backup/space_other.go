//go:build !linux && !windows

package backup

import "fmt"

func freeBytes(string) (uint64, error) {
	return 0, fmt.Errorf("restore space checks require Linux or Windows")
}
