// Package bool implements a recursive-descent parser for boolean expressions according to the following grammar.
//
//	parens   -> ( and ) | and
//	and      -> or AND parens | or
//	or       -> not OR parens | not
//	not      -> NOT parens | unparsed
//	unparsed -> .*
//
// It leaves unparsed portions of the expression in parse.Unparsed nodes, for later consumption by other processes.
//
// The syntax used by this parser is configurable at runtime, see NewParser for details. By default, this parser
// provides a case-sensitive variety of ANSI SQL syntax.
//
// Care must be taken when selecting a NOT operator, since the parser provided by this package is not aware of
// the expression language in use. For instance, selecting '!' as the NOT operator may result in conflicts which used
// with expressions containing '!=', due to parsing ambiguity.
package bool

import (
	"fmt"
	"github.com/orkes-io/go-parse"
	"strings"
	"unicode"
)

// Expr represents a boolean expression.
type Expr interface {
	parse.AST
	BoolExpr() // marker interface
}

// BinExpr represents a boolean expression consisting of clauses of one boolean operator.
type BinExpr struct {
	LHS parse.AST // LHS is the left-hand side
	RHS parse.AST // RHS is the right-hand side
	Op  Op
}

func (b BinExpr) IsAST()    {}
func (b BinExpr) BoolExpr() {} // marker interface

// UnaryExpr represents a unary boolean expression.
type UnaryExpr struct {
	Op   Op
	Expr parse.AST
}

func (u UnaryExpr) IsAST()    {}
func (u UnaryExpr) BoolExpr() {}

// Op represents a boolean operation.
type Op uint8

const (
	OpAnd Op = iota + 1
	OpOr
	OpNot
)

func (o Op) String() string {
	switch o {
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	case OpNot:
		return "NOT"
	default:
		return "unknown op"
	}
}

// Token represents a token in the expression being parsed.
type Token uint8

const (
	And Token = iota + 1
	Or
	Not
	OpenParen
	CloseParen
)

type token string

type ParserOpt func(*Parser)

type Parser struct {
	config          map[Token]string
	keywords        map[string]struct{}
	caseInsensitive bool

	tokens []token
	curr   int
}

// WithTokens configures a parser with the provided token mapping.
func WithTokens(config map[Token]string) ParserOpt {
	return func(parser *Parser) {
		parser.config = config
	}
}

// WithCaseSensitive sets whether the configured parser is case-sensitive.
func WithCaseSensitive(caseSensitive bool) ParserOpt {
	return func(parser *Parser) {
		parser.caseInsensitive = !caseSensitive
	}
}

// NewParser returns a parser configured according to the provided options. If no options are configured, the default
// parser is returned.
func NewParser(opts ...ParserOpt) (*Parser, error) {
	p := &Parser{
		config: map[Token]string{
			And:        "AND",
			Or:         "OR",
			Not:        "NOT",
			OpenParen:  "(",
			CloseParen: ")",
		},
	}
	for _, opt := range opts {
		opt(p)
	}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Parser) init() error {
	if len(p.config[OpenParen]) != 1 || len(p.config[CloseParen]) != 1 {
		return fmt.Errorf("%w: OpenParen and CloseParen must each have length 1", parse.ErrConfig)
	}
	if p.config[OpenParen] == p.config[CloseParen] {
		return fmt.Errorf("%w: OpenParen and CloseParen must each be distinct", parse.ErrConfig)
	}
	if p.caseInsensitive {
		newTokens := make(map[Token]string, len(p.config))
		for token, str := range p.config {
			newTokens[token] = strings.ToLower(str)
		}
		p.config = newTokens
	}
	p.keywords = make(map[string]struct{}, len(p.config))
	for _, str := range p.config {
		p.keywords[str] = struct{}{}
	}
	if len(p.keywords) != 5 {
		return fmt.Errorf("%w: token collision detected; at least two of the configured tokens are identical", parse.ErrConfig)
	}
	return nil
}

func (p *Parser) Parse(str string) (parse.AST, error) {
	p.tokens = p.tokenize(str)
	p.curr = 0
	ast, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.curr != len(p.tokens) {
		return nil, fmt.Errorf("%w: expected end of expression, found '%s'", parse.ErrParse, p.tokens[p.curr])
	}
	return ast, nil
}

func (p *Parser) tokenize(str string) []token {
	if p.caseInsensitive {
		str = strings.ToLower(str)
	}
	fmt.Println("tokenizing", str)
	runes := []rune(str)
	var substr []rune
	var result []token
	for i := range runes {
		if string(runes[i]) == p.config[OpenParen] ||
			string(runes[i]) == p.config[CloseParen] {
			if len(substr) != 0 {
				result = append(result, token(substr))
				substr = nil
			}
			result = append(result, token(runes[i]))
			continue
		}
		if unicode.IsSpace(runes[i]) {
			if len(substr) > 0 {
				result = append(result, token(substr))
				substr = nil
			}
			continue
		}
		substr = append(substr, runes[i])
		if p.isKeyword(string(substr)) {
			result = append(result, token(substr))
			substr = nil
			continue
		}
	}
	if len(substr) > 0 {
		result = append(result, token(substr))
	}
	return result
}

func (p *Parser) match(token Token) bool {
	if p.curr == len(p.tokens) {
		return false
	}
	if string(p.tokens[p.curr]) == p.config[token] {
		p.curr++
		return true
	}
	return false
}

func (p *Parser) peek() string {
	return string(p.tokens[p.curr])
}

func (p *Parser) isKeyword(str string) bool {
	_, ok := p.keywords[str]
	return ok
}

func (p *Parser) parseExpr() (parse.AST, error) {
	return p.parseAnd()
}

func (p *Parser) parseAnd() (parse.AST, error) {
	lhs, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.match(And) {
		rhs, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		return BinExpr{LHS: lhs, RHS: rhs, Op: OpAnd}, nil
	}
	return lhs, nil
}

func (p *Parser) parseOr() (parse.AST, error) {
	lhs, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	if p.match(Or) {
		rhs, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		return BinExpr{LHS: lhs, RHS: rhs, Op: OpOr}, nil
	}
	return lhs, nil
}

func (p *Parser) parseNot() (parse.AST, error) {
	if p.match(Not) {
		rest, err := p.parseParens()
		if err != nil {
			return nil, err
		}
		return UnaryExpr{Expr: rest, Op: OpNot}, nil
	}
	return p.parseParens()
}

// parseParens parses parentheses, which must be correctly matched
func (p *Parser) parseParens() (parse.AST, error) {
	if p.match(OpenParen) {
		ast, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if !p.match(CloseParen) {
			return nil, fmt.Errorf("%w: expected '%s'", parse.ErrParse, p.config[CloseParen])
		}
		return ast, nil
	}
	return p.parseRest()
}

func (p *Parser) parseRest() (parse.AST, error) {
	var result []string
	for p.curr < len(p.tokens) && !p.isKeyword(p.peek()) {
		result = append(result, p.peek())
		p.curr++
	}
	if result == nil {
		return nil, fmt.Errorf("%w: unexpected end of expression", parse.ErrParse)
	}
	return parse.Unparsed{Contents: result}, nil
}
