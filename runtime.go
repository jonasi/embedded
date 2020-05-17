package embedded

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func NewRuntimeDir(path string, depth int) (Dir, error) {
	path, err := relpath(path, depth)
	if err != nil {
		return nil, err
	}

	return &runtimeDir{
		path:  path,
		files: map[string]File{},
	}, nil
}

type runtimeDir struct {
	path  string
	files map[string]File
}

func (r *runtimeDir) Open(path string) (http.File, error) {
	path = strings.TrimSpace(path)
	if len(path) > 0 && path[0] == filepath.Separator {
		path = path[1:]
	}

	if f, ok := r.files[path]; ok {
		return os.Open(f.(*runtimeFile).path)
	}

	return http.Dir(r.path).Open(path)
}

func (r *runtimeDir) Read(path ...string) ([]os.FileInfo, error) {
	p := filepath.Join(path...)
	return ioutil.ReadDir(filepath.Join(r.path, p))
}

func (r *runtimeDir) File(path ...string) File {
	p := filepath.Join(path...)
	if f, ok := r.files[p]; ok {
		return f
	}

	return &runtimeFile{
		path: filepath.Join(r.path, p),
	}
}

func (r *runtimeDir) Add(path string, f File) Dir {
	r.files[path] = f
	return r
}

func NewRuntimeFile(path string, depth int) (File, error) {
	path, err := relpath(path, depth)
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
	_, file, _, ok := runtime.Caller(depth)
	if !ok {
		return "", ErrCallerInfo
	}

	return filepath.Join(filepath.Dir(file), path), nil
}
