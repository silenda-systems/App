package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var MARKER uint64 = binary.LittleEndian.Uint64([]byte{
	0xd7, 0x21, 0xa6, 0xfa,
	0xa1, 0x15, 0x2d, 0xc8,
})

type Encoder struct {
	token []byte
	block cipher.Block
}

func NewEncoder(token [16]byte) *Encoder {
	block, err := aes.NewCipher(token[:])
	if err != nil {
		return nil
	}

	return &Encoder{
		token: token[:],
		block: block,
	}
}

func (e *Encoder) BlockSize() uint64 {
	return uint64(e.block.BlockSize())
}

func (e *Encoder) encode(in io.Reader, out io.Writer) error {
	var err error

	iv := make(
		[]byte, e.BlockSize(),
	)

	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return err
	}

	if _, err = out.Write(iv); err != nil {
		return err
	}

	buf := make([]byte, 1024)
	ctr := cipher.NewCTR(e.block, iv)

	for {
		count, err := in.Read(buf)

		if count > 0 {
			ctr.XORKeyStream(buf, buf[:count])
			out.Write(buf[:count])
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Encoder) decode(size int64, in io.Reader, out io.Writer) error {
	var err error

	iv := make(
		[]byte, e.BlockSize(),
	)

	_, err = in.Read(iv)
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	ctr := cipher.NewCTR(e.block, iv)

	if size > 0 {
		in = io.LimitReader(in, size)
	}

	for {
		n, err := in.Read(buf)

		if n > 0 {
			ctr.XORKeyStream(buf, buf[:n])
			out.Write(buf[:n])
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Encoder) WipeFile(file *os.File) error {
	_, err := file.Seek(0, io.SeekStart)

	if err != nil {
		return err
	}

	info, err := file.Stat()

	if err != nil {
		return err
	}

	var size int64 = info.Size()
	const chunk = 65536

	parts := uint64(
		math.Ceil(float64(size) / float64(chunk)),
	)

	pos := 0

	for i := uint64(0); i < parts; i++ {
		bufsize := int(math.Min(
			chunk, float64(size-int64(i*chunk))),
		)

		zeros := make([]byte, bufsize)
		copy(zeros[:], "0")

		_, err := file.WriteAt(
			[]byte(zeros), int64(pos),
		)

		if err != nil {
			return err
		}

		pos = pos + bufsize
	}

	err = file.Truncate(1)

	if err != nil {
		return err
	}

	err = file.Sync()

	if err != nil {
		return err
	}

	return nil
}

func (e *Encoder) EncodeFile(path string) (string, error) {
	var name = bytes.Buffer{}
	var head [10]byte

	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return "", err
	}

	defer file.Close()

	temp, err := ioutil.TempFile(
		filepath.Dir(path), "*.sl")

	if err != nil {
		return "", err
	}

	defer temp.Close()

	err = e.encode(
		strings.NewReader(filepath.Base(path)),
		&name,
	)

	if err != nil {
		return "", err
	}

	binary.LittleEndian.PutUint64(
		head[:8], MARKER,
	)

	binary.LittleEndian.PutUint16(
		head[8:], uint16(name.Len()),
	)

	_, err = temp.Write(head[:])
	if err != nil {
		return "", err
	}

	_, err = temp.Write(name.Bytes())
	if err != nil {
		return "", err
	}

	err = e.encode(file, temp)
	if err != nil {
		return "", err
	}

	if err = e.WipeFile(file); err != nil {
		return "", err
	}

	file.Close()

	err = os.Remove(file.Name())
	if err != nil {
		return "", err
	}

	return temp.Name(), nil
}

func (e *Encoder) DecodeFile(path string) (string, error) {
	var head [10]byte

	file, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.ReadFull(file, head[:])
	if err != nil {
		return "", err
	}

	if binary.LittleEndian.Uint64(head[:8]) != MARKER {
		return "", err
	}

	size := binary.LittleEndian.Uint16(head[8:])
	name := bytes.Buffer{}

	err = e.decode(
		int64(uint64(size)-e.BlockSize()),
		file, &name,
	)

	if err != nil {
		return "", err
	}

	real_path := filepath.Join(
		filepath.Dir(path),
		name.String(),
	)

	real, err := os.OpenFile(
		real_path,
		os.O_CREATE|os.O_WRONLY,
		0600,
	)

	if err != nil {
		return "", err
	}

	defer real.Close()

	err = e.decode(
		0, file, real,
	)

	if err != nil {
		return "", err
	}

	if err = e.WipeFile(file); err != nil {
		return "", err
	}

	file.Close()

	err = os.Remove(file.Name())
	if err != nil {
		return "", err
	}

	return name.String(), nil
}

func (e *Encoder) DecodeFileName(path string) (string, error) {
	var head [10]byte

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.ReadFull(file, head[:])
	if err != nil {
		return "", err
	}

	if binary.LittleEndian.Uint64(head[:8]) != MARKER {
		return "", err
	}

	size := binary.LittleEndian.Uint16(head[8:])
	name := bytes.Buffer{}

	err = e.decode(
		int64(uint64(size)-e.BlockSize()),
		file, &name,
	)

	if err != nil {
		return "", err
	}

	return name.String(), nil
}
