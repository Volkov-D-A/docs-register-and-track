package backup

import "golang.org/x/sys/windows"

func freeBytes(directory string) (uint64, error) {
	name, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return 0, err
	}
	var free uint64
	err = windows.GetDiskFreeSpaceEx(name, &free, nil, nil)
	return free, err
}
