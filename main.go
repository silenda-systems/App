package main

import (
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
)

var debug = false
var index = "this://app/index.html"

var (
	basedir string
	workdir string
	storage *Storage = nil
	encoder *Encoder = nil
)

var (
	unkn_error  = "unknown error"
	perm_error  = "permission denied"
	pass_error  = "invalid password"
	pass_same   = "real and fake passwords are the same"
	pass_simple = "password is too simple"
	no_device   = "silenda device not found"
)

func Ok(val interface{}) *sciter.Value {
	r := sciter.NewValue()

	r.Set("err", sciter.NothingValue())
	r.Set("val", val)

	return r
}

func Error(err string) *sciter.Value {
	r := sciter.NewValue()

	r.Set("val", sciter.NothingValue())
	r.Set("err", err)

	return r
}

func makeWindow() *window.Window {
	rect := sciter.NewRect(100, 100, 338, 562)

	sciter.SetOption(
		sciter.SCITER_SET_SCRIPT_RUNTIME_FEATURES,
		sciter.ALLOW_FILE_IO|
			sciter.ALLOW_SOCKET_IO|
			sciter.ALLOW_EVAL|
			sciter.ALLOW_SYSINFO,
	)

	win, err := window.New(
		sciter.SW_MAIN|
			sciter.SW_TITLEBAR|
			sciter.SW_CONTROLS,
		rect,
	)

	if err != nil {
		log.Fatal(err)
	}

	win.SetTitle("Silenda Flash")
	return win
}

func normalizePath(path string) string {
	path, _ = url.PathUnescape(path)

	path = strings.TrimPrefix(
		path, "file://",
	)

	path = filepath.Join(
		path, "/")

	return path
}

func hasUSB(opt ...*sciter.Value) *sciter.Value {
	if err := usbRead(); err != nil {
		return Error(no_device)
	}

	return Ok(true)
}

func appReady(opt ...*sciter.Value) *sciter.Value {
	return Ok(storage.Ready())
}

func appLogged(opt ...*sciter.Value) *sciter.Value {
	return Ok(storage.IsLogged() && encoder != nil)
}

func baseDir(opt ...*sciter.Value) *sciter.Value {
	return Ok(basedir)
}

func baseVol(opt ...*sciter.Value) *sciter.Value {
	return Ok(filepath.VolumeName(basedir))
}

func parentDir(opt ...*sciter.Value) *sciter.Value {
	if len(opt) != 1 {
		return Error(unkn_error)
	}

	path := normalizePath(
		opt[0].String(),
	)

	return Ok(filepath.Dir(path))
}

func buildPath(opt ...*sciter.Value) *sciter.Value {
	if len(opt) != 2 {
		return Error(unkn_error)
	}

	path := normalizePath(
		opt[0].String(),
	)

	return Ok(filepath.Join(
		path, opt[1].String()),
	)
}

func listDir(opt ...*sciter.Value) *sciter.Value {
	var list = sciter.NewValue()

	if len(opt) != 1 {
		return Error(unkn_error)
	}

	path := normalizePath(
		opt[0].String(),
	)

	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	var dirs, files = sciter.NewValue(), sciter.NewValue()

	for _, entry := range entries {
		name := entry.Name()

		if isFileHidden(path, name) {
			continue
		}

		if entry.IsDir() {
			dirs.Append(name)
			continue
		}

		if entry.Type().IsRegular() {
			files.Append(name)
		}
	}

	list.Set("dirs", dirs)
	list.Set("files", files)

	return Ok(list)
}

func setFakePass(opt ...*sciter.Value) *sciter.Value {
	if len(opt) != 1 {
		return Error(unkn_error)
	}

	err := storage.pass.Real.Compare(
		opt[0].String(),
	)

	if err == nil {
		return Error(pass_same)
	}

	err = storage.pass.Fake.Make(
		opt[0].String(),
	)

	if err != nil {
		return Error(unkn_error)
	}

	err = storage.Update()
	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	return Ok(true)
}

func setRealPass(opt ...*sciter.Value) *sciter.Value {
	if len(opt) != 1 {
		return Error(unkn_error)
	}

	var pass = opt[0].String()

	err := storage.pass.Fake.Compare(
		pass,
	)

	if err == nil {
		return Error(pass_same)
	}

	if len(pass) < 6 {
		return Error(pass_simple)
	}

	var upp, low, num bool
	var cnt = 0

	for _, char := range pass {
		cnt++

		switch {
		case unicode.IsUpper(char):
			upp = true
		case unicode.IsLower(char):
			low = true
		case unicode.IsNumber(char):
			num = true
		default:
		}
	}

	if !upp || !low || !num || cnt < 6 {
		return Error(pass_simple)
	}

	err = storage.pass.Real.Make(
		pass,
	)

	if err != nil {
		return Error(unkn_error)
	}

	err = storage.Update()

	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	return Ok(true)
}

func login(opt ...*sciter.Value) *sciter.Value {
	if len(opt) != 1 {
		return Error(unkn_error)
	}

	logged := storage.Login(opt[0].String())
	if !logged {
		return Error(pass_error)
	}

	encoder = NewEncoder(storage.token)
	return Ok(true)
}

func encodeFile(opt ...*sciter.Value) *sciter.Value {
	path := buildPath(opt...)

	if !path.Get("err").IsUndefined() {
		return path
	}

	_, err := encoder.EncodeFile(
		path.Get("val").String(),
	)

	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	return Ok(true)
}

func decodeFile(opt ...*sciter.Value) *sciter.Value {
	path := buildPath(opt...)

	if !path.Get("err").IsUndefined() {
		return path
	}

	_, err := encoder.DecodeFile(
		path.Get("val").String(),
	)

	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	return Ok(true)
}

func decodeFileName(opt ...*sciter.Value) *sciter.Value {
	path := buildPath(opt...)

	if !path.Get("err").IsUndefined() {
		return path
	}

	name, err := encoder.DecodeFileName(
		path.Get("val").String(),
	)

	if err != nil {
		if os.IsPermission(err) {
			return Error(perm_error)
		}

		return Error(unkn_error)
	}

	if !utf8.ValidString(name) {
		return Error(unkn_error)
	}

	return Ok(name)
}

func hasPerm(opt ...*sciter.Value) *sciter.Value {
	if runtime.GOOS != "linux" {
		return Ok(true)
	}

	user, err := user.Current()
	if err != nil {
		return Error(unkn_error)
	}

	if user.Gid != "0" {
		return Error(perm_error)
	}

	return Ok(true)
}

func pathSeparator(opt ...*sciter.Value) *sciter.Value {
	return Ok(string(os.PathSeparator))
}

func main() {
	var err error

	basedir, err = os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	workdir = filepath.Dir(os.Args[0])

	if runtime.GOOS == "darwin" {
		os.Setenv("DYLD_LIBRARY_PATH", workdir)
	}

	if runtime.GOOS == "linux" {
		os.Setenv("LD_LIBRARY_PATH", workdir)
	}

	if !debug {
		os.Stderr.Close()
	}

	storage, err = NewStorage()
	if err != nil {
		log.Fatalln(err)
	}

	defer storage.Close()
	var win = makeWindow()

	win.DefineFunction(
		"hasUSB", hasUSB,
	)

	win.DefineFunction(
		"appReady", appReady,
	)

	win.DefineFunction(
		"appLogged", appLogged,
	)

	win.DefineFunction(
		"baseDir", baseDir,
	)

	win.DefineFunction(
		"baseVol", baseVol,
	)

	win.DefineFunction(
		"listDir", listDir,
	)

	win.DefineFunction(
		"parentDir", parentDir,
	)

	win.DefineFunction(
		"buildPath", buildPath,
	)

	win.DefineFunction(
		"setFakePass", setFakePass,
	)

	win.DefineFunction(
		"setRealPass", setRealPass,
	)

	win.DefineFunction(
		"login", login,
	)

	win.DefineFunction(
		"encodeFile", encodeFile,
	)

	win.DefineFunction(
		"decodeFile", decodeFile,
	)

	win.DefineFunction(
		"decodeFileName", decodeFileName,
	)

	win.DefineFunction(
		"hasPerm", hasPerm,
	)

	win.DefineFunction(
		"pathSeparator", pathSeparator,
	)

	if !debug {
		win.SetResourceArchive(resources)
	}

	if debug {
		index, err = filepath.Abs("pages/index.html")
		if err != nil {
			log.Fatalln(err)
		}
	}

	if err := win.LoadFile(index); err != nil {
		log.Fatalln(err)
	}

	win.Show()
	win.Run()
}
