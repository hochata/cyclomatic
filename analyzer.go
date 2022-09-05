package cyclomatic

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"reflect"

	"golang.org/x/tools/go/analysis"

	"github.com/ichiban/prodinspect"
)

// Analyzer checks cyclomatic complexity of defined functions and reports if it exceeded the given limit.
var Analyzer = &analysis.Analyzer{
	Name:       "cyclomatic",
	Doc:        `check cyclomatic complexity of functions.`,
	Requires:   []*analysis.Analyzer{prodinspect.Analyzer},
	ResultType: reflect.TypeOf(new(map[*ast.Ident]Complexity)),
	Run:        run,
	FactTypes:  []analysis.Fact{new(Complexity)},
}

var limit int // -limit flag

func init() {
	prefix := ""
	if os.Args[0] != "cyclomatic" {
		prefix = "cyclomatic-"
	}
	Analyzer.Flags.IntVar(&limit, prefix+"limit", 10, "limit of cyclomatic complexity")
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[prodinspect.Analyzer].(*prodinspect.Inspector)

	acc := make(map[*ast.Ident]Complexity)
	var c Complexity
	inspect.Nodes([]ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.IfStmt)(nil),
		(*ast.ForStmt)(nil),
		(*ast.CaseClause)(nil),
		(*ast.CommClause)(nil),
		(*ast.BinaryExpr)(nil),
	}, func(n ast.Node, push bool) bool {
		if push {
			c.add(n)
			return true
		}

		c.report(n, pass, acc)
		return true
	})
	return &acc, nil
}

type Complexity int

func (c *Complexity) AFact()         {}
func (c *Complexity) String() string { return fmt.Sprintf("complexity(%d)", *c) }

func (c *Complexity) add(n ast.Node) {
	switch n := n.(type) {
	case *ast.FuncDecl:
		*c = 1
	case *ast.IfStmt, *ast.ForStmt, *ast.CommClause:
		*c++
	case *ast.CaseClause:
		*c += Complexity(len(n.List))
	case *ast.BinaryExpr:
		switch n.Op {
		case token.LAND, token.LOR:
			*c++
		}
	}
}

func (c *Complexity) report(n ast.Node, pass *analysis.Pass, acc map[*ast.Ident]Complexity) {
	fd, ok := n.(*ast.FuncDecl)
	if !ok {
		return
	}

	acc[fd.Name] = *c

	o := pass.TypesInfo.Defs[fd.Name]
	f := o.(*types.Func)

	d := *c
	pass.ExportObjectFact(f, &d)

	if d > Complexity(limit) {
		pass.Reportf(n.Pos(), "cyclomatic complexity of %s exceeded limit %d > %d", f.Name(), d, limit)
	}
}
