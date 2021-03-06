package main

import "text/template"

var tmpl = template.Must(template.New("").Parse(`{{ define "functionality" -}}
package {{ .package }}

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jonasi/embedded"
)

var useFs, _ = strconv.ParseBool(os.Getenv("EMBEDDED_USE_FS"))

func newDir(path string) (embedded.Dir, error) {
	d, err := pathData(path)
	if err != nil {
		return nil, err
	}

	return &dir{d}, nil
}

func NewDir(path string) (embedded.Dir, error) {
	if useFs {
		return embedded.NewRuntimeDir(path, 3)
	}

	return newDir(path)
}

func MustDir(path string) embedded.Dir {
	var (
		d   embedded.Dir
		err error
	)

	if useFs {
		d, err = embedded.NewRuntimeDir(path, 3)
	} else {
		d, err = newDir(path)
	}

	if err != nil {
		panic(err)
	}

	return d
}

func NewFile(path string) (embedded.File, error) {
	if useFs {
		return embedded.NewRuntimeFile(path, 3)
	}

	d, err := pathData(path)
	if err != nil {
		return nil, err
	}

	return &file{d, nil}, nil
}

func MustFile(path string) embedded.File {
	var (
		f   embedded.File
		err error
	)

	if useFs {
		f, err = embedded.NewRuntimeFile(path, 3)
	} else {
		f, err = NewFile(path)
	}

	if err != nil {
		panic(err)
	}

	return f
}

type dir struct {
	node *node
}

func (r *dir) Open(path string) (http.File, error) {
	path = strings.TrimSpace(path)
	if len(path) > 1 && path[0] == filepath.Separator {
		path = path[1:]
	} else if len(path) == 1 && path[0] == filepath.Separator {
		return &httpfile{node: r.node}, nil
	}

	parts := strings.Split(path, string(filepath.Separator))
	n := r.node.walk(parts...)
	if n == nil {
		return nil, os.ErrNotExist
	}

	h := &httpfile{node: n}
	if n.content != nil {
		h.ReadSeeker = bytes.NewReader(n.content)
	}

	return h, nil
}

func (r *dir) Read(path ...string) ([]os.FileInfo, error) {
	n := r.node.walk(path...)
	if n == nil {
		return nil, os.ErrNotExist
	}

	if !n.fi.IsDir() {
		return nil, errors.New("Not a directory")
	}

	sl := make([]os.FileInfo, len(n.children))
	i := 0
	for _, n := range n.children {
		sl[i] = n.fi
		i++
	}

	return sl, nil
}

func (r *dir) File(path ...string) embedded.File {
	n := r.node.walk(path...)
	var err error
	if n == nil {
		err = os.ErrNotExist
	} else if n.fi.IsDir() {
		err = errors.New("Not a file")
	}

	return &file{n, err}
}

func (r *dir) Add(path string, f embedded.File) embedded.Dir {
	parts := strings.Split(path, string(filepath.Separator))
	cur := r.node
	for i, p := range parts {
		if i == len(parts)-1 {
			cur.children[p] = f.(*file).node
		} else {
			ch := cur.children[p]
			if ch == nil {
				cur.children[p] = &node{
					name:     p,
					children: map[string]*node{},
					fi: &fileInfo{
						name:  p,
						isdir: true,
					},
				}
				ch = cur.children[p]
			}

			cur = ch
		}
	}

	return r
}

type httpfile struct {
	io.ReadSeeker
	node *node
}

func (h *httpfile) Close() error {
	return nil
}

func (h *httpfile) Readdir(count int) ([]os.FileInfo, error) {
	if count > 0 {
		return nil, errors.New("Readdir with count > 0 not implemented")
	}

	sl := make([]os.FileInfo, len(h.node.children))
	i := 0
	for _, n := range h.node.children {
		sl[i] = n.fi
		i++
	}

	return sl, nil
}

func (h *httpfile) Stat() (os.FileInfo, error) {
	return h.node.fi, nil
}

type file struct {
	node *node
	err  error
}

func (r *file) Contents() ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.node.content, nil
}

func (r *file) MustContents() []byte {
	if r.err != nil {
		panic(r.err)
	}

	return r.node.content
}

func pathData(path string) (*node, error) {
	id, err := callerID(4)
	if err != nil {
		return nil, err
	}

	id = id + "|" + path
	if data[id] == nil {
		return nil, os.ErrNotExist
	}

	return data[id], nil
}

func callerID(depth int) (string, error) {
	pc, file, _, ok := runtime.Caller(depth)
	if !ok {
		return "", embedded.ErrCallerInfo
	}

	var (
		parts       = strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl          = len(parts)
		packageName = ""
	)

	if parts[pl-2][0] == '(' {
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	id := packageName + "/" + filepath.Base(file)
	return id, nil
}

type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modtime time.Time
	isdir   bool
}

func (f *fileInfo) Name() string       { return f.name }
func (f *fileInfo) Size() int64        { return f.size }
func (f *fileInfo) Mode() os.FileMode  { return f.mode }
func (f *fileInfo) ModTime() time.Time { return f.modtime }
func (f *fileInfo) IsDir() bool        { return f.isdir }
func (f *fileInfo) Sys() interface{}   { return nil }

type node struct {
	name     string
	content  []byte
	children map[string]*node
	fi       os.FileInfo
}

func (n *node) walk(parts ...string) *node {
	cur := n
	for _, p := range parts {
		if len(cur.children) == 0 {
			return nil
		}

		cur = cur.children[p]
		if cur == nil {
			return nil
		}
	}

	return cur
}
{{ end }}
{{ define "data" -}}
package {{ .package }}

import (
	"time"
)

var data = {{ .data | printf "%#v" }}
{{ end }}
`))
