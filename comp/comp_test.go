package comp

import (
	"fmt"
	"github.com/orkes-io/go-parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParser_tokenize(t *testing.T) {
	tests := []struct {
		input  string
		output []string
	}{
		{
			"", nil,
		},
		{
			"x >= y", []string{"x", ">=", "y"},
		},
		{
			"x < y", []string{"x", "<", "y"},
		},
		{
			"x == y", []string{"x", "==", "y"},
		},
		{
			"abc == (xyz > 3)", []string{"abc", "==", "(", "xyz", ">", "3", ")"},
		},
		{
			"x > 3 > t > 7", []string{"x", ">", "3", ">", "t", ">", "7"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := NewParser()
			require.NoError(t, err)
			assert.EqualValues(t, tt.output, p.tokenize(tt.input))
		})
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		input  string
		output parse.AST
	}{
		{
			"abc >= def",
			gte(un("abc"), un("def")),
		},
		{
			"abc == (x > 3)",
			eq(un("abc"), gt(un("x"), un("3"))),
		},
		{
			"x == (y == 3)",
			eq(un("x"), eq(un("y"), un("3"))),
		},
		{
			"x > (y > 3)",
			gt(un("x"), gt(un("y"), un("3"))),
		},
		{
			"(x > 3) == ((y == 3) != ((z < 8) == (y <= 4)))",
			eq(gt(un("x"), un("3")), neq(eq(un("y"), un("3")), eq(lt(un("z"), un("8")), lte(un("y"), un("4"))))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := NewParser()
			require.NoError(t, err)
			ast, err := p.ParseStr(tt.input)
			require.NoError(t, err)
			assert.EqualValues(t, tt.output, ast)
		})
	}
}

func TestParser_ParseError(t *testing.T) {
	tests := []string{
		"x >",
		"y ==    \t\n",
		"!=",
		"!= > 7",
		"(((x > 5))",
		"(x > (7 == 5) < 12)",
		"==!",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			p, err := NewParser()
			require.NoError(t, err)
			ast, err := p.ParseStr(tt)
			assert.ErrorIs(t, err, parse.ErrParse, "ast was: %#v", ast)
		})
	}
}

func TestWithTokens(t *testing.T) {
	p, err := NewParser(WithTokens(map[Token]string{
		Equal:          "EQ",
		NotEqual:       "NEQ",
		Greater:        "GT",
		GreaterOrEqual: "GE",
		Less:           "LT",
		LessOrEqual:    "LE",
		OpenParen:      "[",
		CloseParen:     "]",
	}))
	require.NoError(t, err)

	tests := []struct {
		input  string
		output parse.AST
	}{
		{
			"5 NEQ 7",
			neq(un("5"), un("7")),
		},
		{
			"false NEQ [7 GT 5]",
			neq(un("false"), gt(un("7"), un("5"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ast, err := p.ParseStr(tt.input)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.output, ast)
		})
	}

	testWithErrors := []map[Token]string{
		{Equal: "=="},
		{Equal: "==", NotEqual: "==", LessOrEqual: "<=", Less: "<", Greater: ">", GreaterOrEqual: ">=", OpenParen: "(", CloseParen: ")"},
		{Equal: "==", NotEqual: "!=", LessOrEqual: "<=", Less: "<", Greater: ">", GreaterOrEqual: ">=", OpenParen: ":", CloseParen: ":"},
	}
	for idx, config := range testWithErrors {
		t.Run(fmt.Sprintf("error case %d", idx), func(t *testing.T) {
			_, err := NewParser(WithTokens(config))
			assert.ErrorIs(t, err, parse.ErrConfig)
		})
	}
}

func eq(a, b parse.AST) parse.AST {
	return &EqualExpr{LHS: a, RHS: b, Op: OpEqual}
}

func neq(a, b parse.AST) parse.AST {
	return &EqualExpr{LHS: a, RHS: b, Op: OpNotEqual}
}

func gt(a, b parse.AST) parse.AST {
	return &OrdinalExpr{LHS: a, RHS: b, Op: OpGreater}
}
func lt(a, b parse.AST) parse.AST {
	return &OrdinalExpr{LHS: a, RHS: b, Op: OpLess}
}
func gte(a, b parse.AST) parse.AST {
	return &OrdinalExpr{LHS: a, RHS: b, Op: OpGreaterOrEqual}
}
func lte(a, b parse.AST) parse.AST {
	return &OrdinalExpr{LHS: a, RHS: b, Op: OpLessOrEqual}
}

// un stands for unparsed and returns a parse.Unparsed
func un(tokens ...string) parse.AST {
	return parse.Unparsed{Contents: tokens}
}
