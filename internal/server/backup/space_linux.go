package backup

import "syscall"

func freeBytes(directory string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(directory, &stat)
	return stat.Bavail * uint64(stat.Bsize), err
}
