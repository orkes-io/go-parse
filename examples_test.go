package parse_test

import (
	"fmt"
	"github.com/orkes-io/go-parse"
	"github.com/orkes-io/go-parse/bools"
	"github.com/orkes-io/go-parse/comp"
	"strings"
)

func Example() {
	bParser, _ := bools.NewParser()
	cParser, _ := comp.NewParser()

	// parse boolean expression
	ast, err := bParser.ParseStr("x >= 5 AND NOT(y < 7 OR z != 3)")
	if err != nil {
		fmt.Printf("error parsing boolean expression: %v\n", err)
	}

	// parse comparisons
	err = ast.Parse(cParser)
	if err != nil {
		fmt.Printf("error parsing comparison: %v\n", err)
	}

	bfsPrint(ast)
	// Output:
	// x >= 5 AND  NOT(y < 7 OR z != 3)
}

func bfsPrint(ast parse.AST) {
	switch ast := (ast).(type) {
	case *bools.BinExpr:
		bfsPrint(ast.LHS)
		fmt.Printf(" %s ", ast.Op.String())
		bfsPrint(ast.RHS)
	case *bools.UnaryExpr:
		fmt.Print(" NOT(")
		bfsPrint(ast.Expr)
		fmt.Print(") ")
	case *comp.EqualExpr:
		bfsPrint(ast.LHS)
		fmt.Printf(" %s ", ast.Op.String())
		bfsPrint(ast.RHS)
	case *comp.OrdinalExpr:
		bfsPrint(ast.LHS)
		fmt.Printf(" %s ", ast.Op.String())
		bfsPrint(ast.RHS)
	case parse.Unparsed:
		fmt.Print(strings.Join(ast.Contents, " "))
	}
}
