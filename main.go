// The main goat package
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
	"unicode"
)

// split lines greater than 50 characters
func splitLine(line string) []string {
	lines := []string{}
	a := []rune(line)
	nl := ""
	for _, r := range a {
		nl += string(r)
		if unicode.IsSpace(r) && len(nl) > 50 {
			lines = append(lines, nl)
			nl = ""
		}
	}
	lines = append(lines, nl)
	return lines
}

// goat cmd main entry point
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

	fmt.Println("WARNING: This command will re-write all files in the working directory and all sub-directories to add the new comments. Since this is an experimental project there is a chance something could go wrong and you could loose files or work. Exit this process unless you have a back-up.")

DeclLoop:
	for d := range dch {

		if d.Dtype == "package" {
			for _, decl := range ld {
				if decl.PackageName == d.PackageName {
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
			line := scanner.Text()
			if len(line) > 50 {
				lines := splitLine(line)
				for _, l := range lines {
					d.Comment = append(d.Comment, l)
				}
			} else {
				d.Comment = append(d.Comment, line)
			}
		}
		ld = append(ld, d)
	}

	// file ast info
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
			if c, ok := n.(*ast.CommentGroup); ok {
				comments = append(comments, c)
			}
			switch v := n.(type) {
			case *ast.FuncDecl:
			case *ast.GenDecl:
				for _, d := range f.decls {
					if int(v.Pos()) == d.Pos {
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
