// Package to read go files and find declarations with
// no comments
package read

import (
	"fmt"
	"os"
	"bytes"
	"go/token"
	"go/parser"
	"go/ast"
	"go/printer"
)

// declaration info
type Decl struct {
	Dtype		string
	Pos		int
	FileName	string
	PackageName	string
	Name		string
	Line		int
	Body		string
	Comment		[]string
}

// Read go file and find declarations with no comments
// and
// return through channel
func ReadDecl(file string) ([]Decl, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	d := []Decl{}
	if node.Package == 1 {
		d = append(d, Decl{"package", int(node.Pos()), file, node.Name.Name, node.Name.Name, int(node.Pos()), "", []string{}})
	}
	ast.Inspect(node, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			if v.Doc.Text() == "" {
				var b []byte
				body := bytes.NewBuffer(b)
				err := printer.Fprint(body, fset, n)
				if err != nil {
					fmt.Fprintf(os.Stderr, "read print: %v\n", err)
					return false
				}
				d = append(d, Decl{"function", int(n.Pos()), file, node.Name.Name, v.Name.Name, int(v.Pos()), body.String(), []string{}})
			}
		case *ast.GenDecl:
			if v.Tok == token.TYPE {
				name := ""
				for _, t := range v.Specs {
					if a, ok := t.(*ast.TypeSpec); ok {
						if v.Doc.Text() == "" {
							name = a.Name.Name
							d = append(d, Decl{"type", int(n.Pos()), file, node.Name.Name, name, int(a.Pos()), "", []string{}})
						}
					}
				}
			}
		}
		return true
	})
	return d, nil
}
