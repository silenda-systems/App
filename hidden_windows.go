// +build windows

package main

import (
	"path/filepath"
	"syscall"
)

func isFileHidden(dir, name string) bool {
	ptr, err := syscall.UTF16PtrFromString(
		filepath.Join(dir, name),
	)

	if err != nil {
		return true
	}

	attrs, err := syscall.GetFileAttributes(ptr)
	if err != nil {
		return true
	}

	if attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
		return true
	}

	return false
}
