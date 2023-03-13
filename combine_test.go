package parse_test

import (
	"github.com/orkes-io/go-parse"
	"github.com/orkes-io/go-parse/bools"
	"github.com/orkes-io/go-parse/comp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBoolComp(t *testing.T) {

	b, err := bools.NewParser()
	require.NoError(t, err)
	c, err := comp.NewParser()
	require.NoError(t, err)

	tests := []struct {
		input  string
		output parse.AST
	}{
		{
			"x > 3 AND y == 5 OR z != 3",
			and(gt(un("x"), un("3")), or(eq(un("y"), un("5")), neq(un("z"), un("3")))),
		},
		{
			"${foo.bar.var1} > ${foo.baz.var2} OR ${bar.foo.var2} == 3",
			or(gt(un("${foo.bar.var1}"), un("${foo.baz.var2}")), eq(un("${bar.foo.var2}"), un("3"))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ast, err := b.ParseStr(tt.input)
			require.NoError(t, err)
			err = ast.Parse(c)
			assert.NoError(t, err)

			assert.EqualValues(t, tt.output, ast)
		})
	}

}

func eq(a, b parse.AST) parse.AST {
	return &comp.EqualExpr{LHS: a, RHS: b, Op: comp.OpEqual}
}

func neq(a, b parse.AST) parse.AST {
	return &comp.EqualExpr{LHS: a, RHS: b, Op: comp.OpNotEqual}
}

func gt(a, b parse.AST) parse.AST {
	return &comp.OrdinalExpr{LHS: a, RHS: b, Op: comp.OpGreater}
}

func lt(a, b parse.AST) parse.AST {
	return &comp.OrdinalExpr{LHS: a, RHS: b, Op: comp.OpLess}
}
func gte(a, b parse.AST) parse.AST {
	return &comp.OrdinalExpr{LHS: a, RHS: b, Op: comp.OpGreaterOrEqual}
}
func lte(a, b parse.AST) parse.AST {
	return &comp.OrdinalExpr{LHS: a, RHS: b, Op: comp.OpLessOrEqual}
}

func or(lhs parse.AST, rhs parse.AST) parse.AST {
	return &bools.BinExpr{LHS: lhs, RHS: rhs, Op: bools.OpOr}
}

func and(lhs parse.AST, rhs parse.AST) parse.AST {
	return &bools.BinExpr{LHS: lhs, RHS: rhs, Op: bools.OpAnd}
}

func not(inside parse.AST) parse.AST {
	return &bools.UnaryExpr{Expr: inside, Op: bools.OpNot}
}

// un stands for unparsed and returns a Unparsed
func un(tokens ...string) parse.AST {
	return parse.Unparsed{Contents: tokens}
}
