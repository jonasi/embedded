package embedded

import (
	"errors"
	"net/http"
	"os"
)

// Errors
var (
	ErrCallerInfo = errors.New("Could not retrieve caller information")
)

type Dir interface {
	http.FileSystem
	Read(path ...string) ([]os.FileInfo, error)
	File(path ...string) File
	Add(string, File) Dir
}

type File interface {
	Contents() ([]byte, error)
	MustContents() []byte
}

// MustFile calls file and panics if there is an error
func MustFile(path string) File {
	f, err := newRuntimeFile(path)
	if err != nil {
		panic(err)
	}
	return f
}

// File returns an http.File provided at path.
func NewFile(path string) (File, error) {
	return newRuntimeFile(path)
}

// MustDir calls Dir and panics if there is an error
func MustDir(path string) Dir {
	d, err := newRuntimeDir(path)
	if err != nil {
		panic(err)
	}

	return d
}

// NewDir returns an http.Dir provided at dir.
func NewDir(path string) (Dir, error) {
	return newRuntimeDir(path)
}
