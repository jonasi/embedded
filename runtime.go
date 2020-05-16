package embedded

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func newRuntimeDir(path string) (Dir, error) {
	path, err := relpath(path, 3)
	if err != nil {
		return nil, err
	}

	return &runtimeDir{
		path: path,
	}, nil
}

type runtimeDir struct {
	path string
}

func (r *runtimeDir) Open(path string) (http.File, error) {
	return http.Dir(r.path).Open(path)
}

func (r *runtimeDir) Read(path ...string) ([]os.FileInfo, error) {
	p := filepath.Join(path...)
	return ioutil.ReadDir(filepath.Join(r.path, p))
}

func (r *runtimeDir) File(path ...string) File {
	p := filepath.Join(path...)
	return &runtimeFile{
		path: filepath.Join(r.path, p),
	}
}

func newRuntimeFile(path string) (File, error) {
	path, err := relpath(path, 3)
	if err != nil {
		return nil, err
	}

	return &runtimeFile{
		path: path,
	}, nil
}

type runtimeFile struct {
	path string
}

func (r *runtimeFile) Contents() ([]byte, error) {
	return ioutil.ReadFile(r.path)
}

func (r *runtimeFile) MustContents() []byte {
	b, err := r.Contents()
	if err != nil {
		panic(err)
	}

	return b
}

func relpath(path string, depth int) (string, error) {
	file, err := frameFile(depth + 1)
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(file), path), nil
}

func frameFile(depth int) (string, error) {
	_, file, _, ok := runtime.Caller(depth)
	if !ok {
		return "", ErrCallerInfo
	}
	return file, nil
}
