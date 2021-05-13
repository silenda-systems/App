package main

import (
	"context"
	_ "embed"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var binName = "usb.*"

//go:embed hid/windows
var hid_windows []byte

//go:embed hid/linux
var hid_linux []byte

//go:embed hid/darwin
var hid_darwin []byte

func getHID() []byte {
	if runtime.GOOS == "windows" {
		return hid_windows
	}

	if runtime.GOOS == "linux" {
		return hid_linux
	}

	if runtime.GOOS == "darwin" {
		return hid_darwin
	}

	return nil
}

func usbCommand(opt ...string) ([]byte, error) {
	temp, err := ioutil.TempFile(
		workdir, binName,
	)

	if err != nil {
		return nil, err
	}

	fullPath, err := filepath.Abs(temp.Name())
	if err != nil {
		return nil, err
	}

	if err = temp.Chmod(0777); err != nil {
		return nil, err
	}

	defer temp.Close()
	defer os.Remove(fullPath)

	hid := getHID()
	if hid == nil {
		return nil, errors.New("unsupported platform")
	}

	_, err = temp.Write(hid)
	if err != nil {
		return nil, err
	}

	temp.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)

	defer cancel()

	cmd := exec.CommandContext(
		ctx, fullPath, opt...)

	out, err := cmd.Output()
	if err != nil {
		return nil, errors.New("cannot read silenda")
	}

	return out, err
}

func usbRead() error {
	out, err := usbCommand("read")
	if err != nil {
		return err
	}

	if len(out) <= 0 {
		return errors.New("unknown device")
	}

	return nil
}

func usbWrite() error {
	_, err := usbCommand("write", "0xac")
	if err != nil {
		return err
	}

	return nil
}
