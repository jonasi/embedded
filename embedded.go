package embedded

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

// Errors
var (
	ErrCallerInfo = errors.New("Could not retrieve caller information")
)

type Dir interface {
	http.FileSystem
	Read(path ...string) ([]os.FileInfo, error)
	File(path ...string) File
}

type File interface {
	Contents() ([]byte, error)
	MustContents() []byte
}

// FileFunc returns an "Open" func
func FileFunc(path string) func() (http.File, error) {
	file, err := frameFile(2)
	return func() (http.File, error) {
		if err != nil {
			return nil, err
		}

		fpath := filepath.Join(filepath.Dir(file), path)
		return os.Open(fpath)
	}
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
