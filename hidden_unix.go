// +build darwin linux

package main

func isFileHidden(dir, name string) bool {
	return name[0] == '.'
}
