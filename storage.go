package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

var (
	hash_file = ".silenda"
	max_tries = 3
)

type Hash [60]byte

func (h *Hash) Make(raw string) error {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(raw), bcrypt.DefaultCost,
	)

	if err != nil {
		return err
	}

	copy(h[:], hash)
	return nil
}

func (h *Hash) Compare(raw string) error {
	return bcrypt.CompareHashAndPassword(
		h[:], []byte(raw),
	)
}

func (p *Hash) IsEmpty() bool {
	empty := Hash{}

	return bytes.Equal(
		p[:], empty[:],
	)
}

type Password struct {
	Fake Hash
	Real Hash
}

type Storage struct {
	pass  Password
	file  *os.File
	token [16]byte
	tries int
}

func NewStorage() (*Storage, error) {
	var err error

	path := filepath.Join(
		workdir, hash_file,
	)

	Storage := &Storage{
		pass:  Password{},
		tries: 0,
	}

	var exists = true

	if _, err = os.Stat(path); os.IsNotExist(err) {
		exists = false
	}

	Storage.file, err = os.OpenFile(
		path, os.O_CREATE|os.O_RDWR,
		0600,
	)

	if err != nil {
		return nil, err
	}

	if !exists {
		if err = Storage.Update(); err != nil {
			return nil, err
		}

	} else {
		if err = Storage.Read(); err != nil {
			return nil, err
		}
	}

	return Storage, nil
}

func (s *Storage) Close() {
	s.file.Close()
}

func (s *Storage) Update() error {
	_, err := s.file.Seek(0, 0)

	if err != nil {
		return err
	}

	return binary.Write(
		s.file,
		binary.LittleEndian,
		s.pass,
	)
}

func (s *Storage) Read() error {
	_, err := s.file.Seek(0, 0)

	if err != nil {
		return err
	}

	return binary.Read(
		s.file,
		binary.LittleEndian,
		&s.pass,
	)
}

func (s *Storage) Ready() bool {
	return !s.pass.Real.IsEmpty() &&
		!s.pass.Fake.IsEmpty()
}

func (s *Storage) Kill() {
	usbWrite()
}

func (s *Storage) Login(token string) bool {
	err := s.pass.Fake.Compare(
		token,
	)

	if err == nil {
		s.Kill()
		return false
	}

	err = s.pass.Real.Compare(
		token,
	)

	if err == nil {
		s.token = md5.Sum([]byte(token))
		return true
	}

	s.tries++

	if s.tries >= max_tries {
		s.Kill()
		return false
	}

	return false
}

func (s *Storage) IsLogged() bool {
	var empty [16]byte

	return !bytes.Equal(
		s.token[:], empty[:],
	)
}
