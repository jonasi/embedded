package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

func main() {
	emb := &embedder{}

	cmd := &cobra.Command{
		Use: "embedgen",
	}

	cmd.RunE = func(_ *cobra.Command, args []string) error {
		emb.paths = args
		return emb.parse()
	}

	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

type embedder struct {
	paths []string
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
		fns     = []func() error{}
	)

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.ImportSpec:
			if v.Path.Value == `"github.com/jonasi/embedded"` {
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

			var fn func(string) func() error

			switch s.Sel.Name {
			case "Dir":
				fn = embedDir
			case "MustDir":
				fn = embedDir
			case "File":
				fn = embedFile
			case "MustFile":
				fn = embedFile
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
				return false
			}

			path, err = filepath.Abs(filepath.Join(filepath.Dir(path), val))
			if err != nil {
				return false
			}

			fns = append(fns, fn(path))
		}
		return true
	})

	if err != nil {
		return err
	}

	return err
}

func embedDir(path string) func() error {
	return func() error {
		return nil
	}
}

func embedFile(path string) func() error {
	return func() error {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return fmt.Errorf("%s is a directory", path)
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return nil
	}
}
