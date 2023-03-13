package bool

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
		output []token
	}{
		{
			"", nil,
		},
		{
			"abc AND def", []token{"abc", "AND", "def"},
		},
		{
			"isToken = 123 AND x > 13", []token{"isToken", "=", "123", "AND", "x", ">", "13"},
		},
		{
			"abc AND NOT(def OR xyz)", []token{"abc", "AND", "NOT", "(", "def", "OR", "xyz", ")"},
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
			"abc AND def",
			and(un("abc"), un("def")),
		},
		{
			"abc AND def OR xyz",
			and(un("abc"), or(un("def"), un("xyz"))),
		},
		{
			"abc AND NOT(def OR xyz)",
			and(un("abc"), not(or(un("def"), un("xyz")))),
		},
		{
			"isToken > 45 OR goodBye < 15",
			or(un("isToken", ">", "45"), un("goodBye", "<", "15")),
		},
		{
			"hello > 5 AND goodbye < 14 AND isAnything == 45",
			and(un("hello", ">", "5"), and(un("goodbye", "<", "14"), un("isAnything", "==", "45"))),
		},
		{
			"a AND b AND c AND d OR x OR y OR z",
			and(un("a"), and(un("b"), and(un("c"), or(un("d"), or(un("x"), or(un("y"), un("z"))))))),
		},
		{
			"xyzNOT OR abc",
			or(un("xyzNOT"), un("abc")),
		},
		{
			"x OR y AND z OR w",
			and(or(un("x"), un("y")), or(un("z"), un("w"))),
		},
		{
			"x OR (y AND z OR w)",
			or(un("x"), and(un("y"), or(un("z"), un("w")))),
		},
		{
			"x AND y OR z AND w",
			and(un("x"), and(or(un("y"), un("z")), un("w"))),
		},
		{
			"xyz == 5",
			un("xyz", "==", "5"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := NewParser()
			require.NoError(t, err)
			ast, err := p.Parse(tt.input)
			require.NoError(t, err)
			assert.EqualValues(t, tt.output, ast)
		})
	}
}

func TestParser_ParseError(t *testing.T) {
	tests := []string{
		"abc AND",
		"abc OR    \t\n",
		"NOT",
		"NOT (a AND b AND c",
		"((((((x > 5))))",
		"()",
		"AND 7",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			p, err := NewParser()
			require.NoError(t, err)
			ast, err := p.Parse(tt)
			assert.ErrorIs(t, err, parse.ErrParse, "ast was: %#v", ast)
		})
	}
}

func TestWithTokens(t *testing.T) {
	p, err := NewParser(WithTokens(map[Token]string{
		And:        "&&",
		Or:         "||",
		Not:        "~",
		OpenParen:  "(",
		CloseParen: ")",
	}))
	require.NoError(t, err)

	tests := []struct {
		input  string
		output parse.AST
	}{
		{
			"15 != 3 && 7 == 5",
			and(un("15", "!=", "3"), un("7", "==", "5")),
		},
		{
			"isTrue == true && ~isFalse",
			and(un("isTrue", "==", "true"), not(un("isFalse"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ast, err := p.Parse(tt.input)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.output, ast)
		})
	}

	testWithErrors := []map[Token]string{
		{And: "&&", Or: "&&", Not: "!", OpenParen: "[", CloseParen: "]"},
		{And: "&&", Or: "||", Not: "!", OpenParen: "::", CloseParen: "::"},
		{And: "&&", Or: "||", Not: "!", OpenParen: "_", CloseParen: "_"},
	}
	for idx, config := range testWithErrors {
		t.Run(fmt.Sprintf("error case %d", idx), func(t *testing.T) {
			_, err := NewParser(WithTokens(config))
			assert.ErrorIs(t, err, parse.ErrConfig)
		})
	}
}

func TestWithCaseSensitive(t *testing.T) {
	p, err := NewParser(WithCaseSensitive(false))
	require.NoError(t, err)

	tests := []struct {
		input  string
		output parse.AST
	}{
		{
			"x == 6 and y == 4",
			and(un("x", "==", "6"), un("y", "==", "4")),
		},
		{
			"x == 6 AND y == 4",
			and(un("x", "==", "6"), un("y", "==", "4")),
		},
		{
			"x OR y and z OR w",
			and(or(un("x"), un("y")), or(un("z"), un("w"))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ast, err := p.Parse(tt.input)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.output, ast)
		})
	}
}

func or(lhs parse.AST, rhs parse.AST) parse.AST {
	return BinExpr{LHS: lhs, RHS: rhs, Op: OpOr}
}

func and(lhs parse.AST, rhs parse.AST) parse.AST {
	return BinExpr{LHS: lhs, RHS: rhs, Op: OpAnd}
}

func not(inside parse.AST) parse.AST {
	return UnaryExpr{Expr: inside, Op: OpNot}
}

// un stands for unparsed and returns a parse.Unparsed
func un(tokens ...string) parse.AST {
	return parse.Unparsed{Contents: tokens}
}
