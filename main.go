// This is the main package
// for the goat tool.
//
// Goat allows users to add new comments to exisiting code by finding all declarations that do not already have comments and interactively collecting comments to be added to the declarations.
//
// New comments can be entered as multi-line text and will automatically have slashes added.
//
// Package comments can be shared across multiple files making it easier to add preamble style comments such as copyright notices.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ajz01/goat/read"
	"github.com/ajz01/goat/walk"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sync"
)

// Entry point for goat tool
func main() {
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}

	dch := make(chan read.Decl)
	var n sync.WaitGroup
	for _, root := range roots {
		n.Add(1)
		go walk.WalkDir(root, &n, dch)
	}

	go func() {
		n.Wait()
		close(dch)
	}()

	ld := []read.Decl{}
	scanner := bufio.NewScanner(os.Stdin)
DeclLoop:
	for d := range dch {
		if d.Dtype == "package" {
			for _, decl := range ld {
				if decl.PackageName == d.PackageName {
					fmt.Printf("using existing comment filename: %s %s\n", d.FileName, decl.Comment)
					d.Comment = decl.Comment
					ld = append(ld, d)
					continue DeclLoop
				}
			}
		}
		fmt.Println("\nAdd comments for the following declaration. Multi-line allowed type q alone on a line to quit.")
		fmt.Printf("Type: %s Filename: %s Package: %s Name: %s\n", d.Dtype, d.FileName, d.PackageName, d.Name)
		for scanner.Scan() {
			if scanner.Text() == "q" {
				break
			}
			d.Comment = append(d.Comment, scanner.Text())
		}
		ld = append(ld, d)
	}

	type fileAst struct {
		file		*ast.File
		fileSet		*token.FileSet
		decls		[]read.Decl
		comments	[]*ast.Comment
	}

	m := make(map[string]*fileAst)
	for _, d := range ld {
		if m[d.FileName] == nil {
			var f fileAst
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, d.FileName, nil, parser.ParseComments)
			if err != nil {
				fmt.Printf("parse: %s", err)
				continue
			}
			f.file = node
			f.fileSet = fset
			m[d.FileName] = &f
		}
		m[d.FileName].decls = append(m[d.FileName].decls, d)
	}

	for _, f := range m {
		comments := []*ast.CommentGroup{}
		for _, d := range f.decls {
			if d.Dtype == "package" {
				for _, c := range d.Comment {
					comments = append(comments, &ast.CommentGroup{[]*ast.Comment{&ast.Comment{token.Pos(1), `// ` + c}}})
				}

				f.file.Package++
			}
		}
		ast.Inspect(f.file, func(n ast.Node) bool {
			c, ok := n.(*ast.CommentGroup)
			if ok {
				comments = append(comments, c)
				for _, comment := range c.List {
					fmt.Printf("existing comment: %d %s\n", comment.Slash, comment.Text)
				}
			}
			switch v := n.(type) {
			case *ast.FuncDecl:
				for _, d := range f.decls {
					if int(v.Pos()) == d.Pos {
						if v.Doc != nil {
							fmt.Println("comments not empty")
						}
						cg := ast.CommentGroup{}
						for _, c := range d.Comment {
							cg.List = append(cg.List, &ast.Comment{token.Pos(d.Pos - 1), `// ` + c})
						}
						v.Doc = &cg
					}
				}
			}
			return true
		})
		f.file.Comments = comments
	}

	for f, a := range m {
		file, err := os.Create(f)
		if err != nil {
			fmt.Printf("create file: %s\n", err)
		}
		defer file.Close()
		if err := printer.Fprint(file, a.fileSet, a.file); err != nil {
			fmt.Printf("writing ast: %s\n", err)
		}
	}
}
