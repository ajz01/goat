// Read an existing go file and extract declarations that are missing comments
// pass declaration info back through channel
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

// Read existing go files and extract declarations that are missing comments
// pass declaration info back through channel
func ReadDecl(file string) ([]Decl, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	d := []Decl{}
	d = append(d, Decl{"package", int(node.Pos()), file, node.Name.Name, node.Name.Name, int(node.Pos()), "", []string{}})
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
		}
		return true
	})
	return d, nil
}
