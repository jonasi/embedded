package embedded

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Errors
var (
	ErrCallerInfo = errors.New("Could not retrieve caller information")
)

// FileFunc returns an "Open" func
func FileFunc(path string) func() (http.File, error) {
	file, err := filename(2)
	return func() (http.File, error) {
		if err != nil {
			return nil, err
		}

		fpath := filepath.Join(filepath.Dir(file), path)
		return os.Open(fpath)
	}
}

// MustFileContents calls FileContents and panics if there is an error
func MustFileContents(path string) []byte {
	f, err := fileContents(path, 2)
	if err != nil {
		panic(err)
	}
	return f
}

// MustFile calls file and panics if there is an error
func MustFile(path string) http.File {
	f, err := file(path, 2)
	if err != nil {
		panic(err)
	}
	return f
}

// MustDir calls Dir and panics if there is an error
func MustDir(path string) http.Dir {
	d, err := dir(path, 2)
	if err != nil {
		panic(err)
	}
	return d
}

// FileContents returns the byte contents of the file
// provided at path.
func FileContents(path string) ([]byte, error) {
	return fileContents(path, 2)
}

// File returns an http.File provided at path.
func File(path string) (http.File, error) {
	return file(path, 2)
}

// Dir returns an http.Dir provided at dir.
func Dir(path string) (http.Dir, error) {
	return dir(path, 2)
}

func fileContents(path string, depth int) ([]byte, error) {
	file, err := filename(depth + 1)
	if err != nil {
		return nil, err
	}

	path = filepath.Join(filepath.Dir(file), path)
	return ioutil.ReadFile(path)
}

func file(path string, depth int) (http.File, error) {
	file, err := filename(depth + 1)
	if err != nil {
		return nil, err
	}

	path = filepath.Join(filepath.Dir(file), path)
	return os.Open(path)
}

func dir(path string, depth int) (http.Dir, error) {
	file, err := filename(depth + 1)
	if err != nil {
		return http.Dir(""), err
	}

	path = filepath.Join(filepath.Dir(file), path)
	return http.Dir(path), nil
}

func filename(depth int) (string, error) {
	_, file, _, ok := runtime.Caller(depth)
	if !ok {
		return "", ErrCallerInfo
	}
	return file, nil
}
