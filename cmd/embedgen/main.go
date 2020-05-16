package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

func main() {
	cmd := &cobra.Command{
		Use: "embedgen",
	}

	var (
		outdir = cmd.Flags().String("out", "", "")
	)

	cmd.RunE = func(_ *cobra.Command, args []string) error {
		if *outdir == "" {
			return errors.New("No out provided")
		}

		abspath, err := filepath.Abs(*outdir)
		if err != nil {
			return err
		}
		outpkg, err := packageFromFile(abspath, "pattern")
		if err != nil {
			return err
		}

		emb := &embedder{
			outpkg: outpkg,
			tree:   nodeMap{},
			paths:  args,
		}

		if err := emb.parse(); err != nil {
			return err
		}

		if err := os.Mkdir(*outdir, 0755); err != nil && !os.IsExist(err) {
			return err
		}

		fns, err := os.Create(filepath.Join(*outdir, "assets.go"))
		if err != nil {
			return err
		}

		defer fns.Close()
		err = tmpl.ExecuteTemplate(fns, "functionality", map[string]interface{}{
			"package": filepath.Base(*outdir),
		})
		if err != nil {
			return err
		}

		data, err := os.Create(filepath.Join(*outdir, "data.go"))
		if err != nil {
			return err
		}

		return tmpl.ExecuteTemplate(data, "data", map[string]interface{}{
			"package": filepath.Base(*outdir),
			"data":    emb.tree,
		})
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

type nodeMap map[string]*node

func (n nodeMap) GoString() string {
	if n == nil {
		return ""
	}

	b := &strings.Builder{}
	b.WriteString(`map[string]*node{`)
	for k := range n {
		b.WriteString(fmt.Sprintf("\n%#v: %#v,", k, n[k]))
	}

	b.WriteString("\n}")
	return b.String()
}

type node struct {
	name     string
	content  []byte
	children nodeMap
	fi       os.FileInfo
}

func (n *node) GoString() string {
	b := &strings.Builder{}
	b.WriteString(`&node{`)
	b.WriteString(fmt.Sprintf(`name: %#v`, n.name))
	b.WriteString(fmt.Sprintf(`, fi: &fileInfo{name: %#v, size: %#v, mode: %#v, modtime: time.Unix(0, %#v), isdir: %#v}`, n.name, n.fi.Size(), n.fi.Mode(), n.fi.ModTime().UnixNano(), n.fi.IsDir()))
	if n.children != nil {
		b.WriteString(fmt.Sprintf(`, children: %#v`, n.children))
	}
	if n.content != nil {
		b.WriteString(`, content: []byte("`)
		for _, byt := range n.content {
			b.WriteString(fmt.Sprintf(`\x%02x`, byt))
		}
		b.WriteString(`")`)
	}

	b.WriteString("}")
	return b.String()
}

type embedder struct {
	outpkg string
	paths  []string
	tree   nodeMap
}

func (e *embedder) parse() error {
	for _, p := range e.paths {
		if err := e.parseFile(p); err != nil {
			return err
		}
	}

	return nil
}

func (e *embedder) parseFile(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	var (
		pkgName string
	)

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.ImportSpec:
			if v.Path.Value == `"github.com/jonasi/embedded"` || v.Path.Value == `"`+e.outpkg+`"` {
				if v.Name == nil {
					pkgName = "embedded"
				} else {
					pkgName = v.Name.Name
				}
			}
		case *ast.CallExpr:
			s, ok := v.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}
			x, ok := s.X.(*ast.Ident)
			if !ok {
				break
			}

			if x.Name != pkgName {
				break
			}

			var fn func(string) (*node, error)

			switch s.Sel.Name {
			case "Dir":
				fn = e.embedDir
			case "MustDir":
				fn = e.embedDir
			case "File":
				fn = e.embedFile
			case "MustFile":
				fn = e.embedFile
			default:
				break
			}

			a, ok := v.Args[0].(*ast.BasicLit)
			if !ok {
				err = fmt.Errorf("embedded.%s was not called with a string literal", s.Sel.Name)
				return false
			}

			var val string
			val, err = strconv.Unquote(a.Value)
			if err != nil {
				fmt.Printf("unquote  = %+v\n", err)
				return false
			}

			abspath, err := filepath.Abs(filepath.Join(filepath.Dir(path), val))
			if err != nil {
				fmt.Printf("abspath = %+v\n", err)
				return false
			}

			p, err := packageFromFile(path, "file")
			if err != nil {
				fmt.Printf("packages = %+v\n", err)
				return false
			}

			id := p + "/" + filepath.Base(path) + "|" + val
			node, err := fn(abspath)
			if err != nil {
				fmt.Printf("fn = %+v\n", err)
				return false
			}

			e.tree[id] = node
		}

		return true
	})

	if err != nil {
		return err
	}

	return err
}

func (e *embedder) embedDir(path string) (*node, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	n := &node{
		name:     filepath.Base(path),
		children: map[string]*node{},
		fi:       fi,
	}

	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if fi.IsDir() {
			cn, err := e.embedDir(filepath.Join(path, fi.Name()))
			if err != nil {
				return nil, err
			}

			n.children[fi.Name()] = cn
		} else {
			cn, err := e.embedFile(filepath.Join(path, fi.Name()))
			if err != nil {
				return nil, err
			}

			n.children[fi.Name()] = cn
		}
	}

	return n, nil
}

func (e *embedder) embedFile(path string) (*node, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &node{
		name:    filepath.Base(path),
		content: b,
		fi:      fi,
	}, nil
}

func packageFromFile(path, typ string) (string, error) {
	p, err := packages.Load(&packages.Config{
		Mode: packages.LoadFiles,
	}, typ+"="+path)

	if err != nil {
		return "", err
	}

	return p[0].PkgPath, nil
}
