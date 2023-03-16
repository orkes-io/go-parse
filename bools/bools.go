// Package bools implements a recursive-descent parser for boolean expressions according to the following grammar.
//
//		expr     -> and
//		and      -> or 'AND' and | or
//		or       -> not 'OR' or | not
//	    not      -> 'NOT' parens | parens
//		parens   -> '(' expr ')' | unparsed
//		unparsed -> '.*
//
// It leaves unparsed portions of the expression in parse.Unparsed nodes, for later consumption by other parsers.
//
// The syntax used by this parser is configurable at runtime, see NewParser for details. By default, this parser
// provides a case-sensitive variety of ANSI SQL syntax.
//
// Care must be taken when selecting a NOT operator, since the parser provided by this package is not aware of
// the expression language in use. For instance, selecting '!' as the NOT operator may result in conflicts when used
// with expressions containing '!=', due to parsing ambiguity.
package bools

import (
	"fmt"
	"github.com/orkes-io/go-parse"
	"strings"
)

// VarInterpreter provides an interpreter which looks up the value of every parsed.Unparsed node with a single token in
// the provided map.
func VarInterpreter(variables map[string]bool) parse.Interpreter[bool] {
	return func(ast parse.AST) (bool, error) {
		if variables == nil {
			return false, fmt.Errorf("%w: nil map", parse.ErrEval)
		}
		switch ast := ast.(type) {
		case parse.Unparsed:
			if len(ast.Contents) > 1 {
				return false, fmt.Errorf("%w: cannot evaluate multi-word variables; found '%v'", parse.ErrEval, strings.Join(ast.Contents, " "))
			}
			val, ok := variables[ast.Contents[0]]
			if !ok {
				return false, fmt.Errorf("%w: unknown variable '%s'", parse.ErrEval, ast.Contents[0])
			}
			return val, nil
		default:
			return false, fmt.Errorf("%w: unknown AST node: %v", parse.ErrUnknownAST, ast)
		}
	}
}

// Eval evaluates the provided AST node using the provided Interpreter, which must be capable of interpreting any nodes
// not found in the bools package.
func Eval(expr parse.AST, interpreter parse.Interpreter[bool]) (bool, error) {
	if interpreter == nil {
		return false, fmt.Errorf("%w: nil Interpreter", parse.ErrEval)
	}
	if expr == nil {
		return false, fmt.Errorf("%w: nil expression", parse.ErrEval)
	}
	switch expr := expr.(type) {
	case *BinExpr:
		if expr.Op != OpAnd && expr.Op != OpOr {
			return false, fmt.Errorf("unexpected binary boolean operator: %v", expr.Op)
		}
		rhs, err := Eval(expr.RHS, interpreter)
		if err != nil {
			return false, err
		}
		lhs, err := Eval(expr.LHS, interpreter)
		if err != nil {
			return false, err
		}
		switch expr.Op {
		case OpAnd:
			return rhs && lhs, nil
		case OpOr:
			return rhs || lhs, nil
		}
	case *UnaryExpr:
		if expr.Op != OpNot {
			return false, fmt.Errorf("unexpected boolean unary operator: %v", expr.Op)
		}
		val, err := Eval(expr.Expr, interpreter)
		if err != nil {
			return false, err
		}
		return !val, nil
	default:
		return interpreter(expr)
	}
	// unreachable
	return false, fmt.Errorf("unexpected expression")
}

// BinExpr represents a boolean expression consisting of clauses of one boolean operator.
type BinExpr struct {
	LHS parse.AST // LHS is the left-hand side
	RHS parse.AST // RHS is the right-hand side
	Op  Op
}

// Parse runs the provided parse.Parser on all the unparsed nodes in this AST.
func (b *BinExpr) Parse(p parse.Parser) error {
	if unparsed, ok := b.LHS.(parse.Unparsed); ok {
		newLHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		b.LHS = newLHS
	} else if err := b.LHS.Parse(p); err != nil {
		return err
	}
	if unparsed, ok := b.RHS.(parse.Unparsed); ok {
		newRHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		b.RHS = newRHS
	} else if err := b.RHS.Parse(p); err != nil {
		return err
	}
	return nil
}

// UnaryExpr represents a unary boolean expression.
type UnaryExpr struct {
	Op   Op
	Expr parse.AST
}

// Parse runs the provided parse.Parser on all unparsed nodes in this AST.
func (u *UnaryExpr) Parse(p parse.Parser) error {
	if unparsed, ok := u.Expr.(parse.Unparsed); ok {
		newExpr, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		u.Expr = newExpr
	} else if err := u.Expr.Parse(p); err != nil {
		return err
	}
	return nil
}

// Op represents a boolean operation recognized by this grammar.
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
	And        Token = iota + 1 // And represents boolean and.
	Or                          // Or represents boolean or.
	Not                         // Not represents boolean not.
	OpenParen                   // OpenParen represents the start of a sub-expression.
	CloseParen                  // CloseParen represents the end of a sub-expression.
)

type ParserOpt func(*Parser)

type Parser struct {
	config          map[Token]string
	caseInsensitive bool
	matcher         *parse.KeywordTrie

	tokens []string
	curr   int
}

// WithTokens configures the syntax used by this parser using the provided token mapping. The provided map must contain
// distinct entries for each Token provided in this package: And, Or, Not, OpenParen, and CloseParen.
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
		matcher: &parse.KeywordTrie{},
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
	for _, str := range p.config {
		p.matcher.Add(str)
	}
	if p.matcher.Count() != 5 {
		return fmt.Errorf("%w: token collision detected; at least two of the configured tokens are identical", parse.ErrConfig)
	}
	return nil
}

// ParseStr tokenizes and parses the provided string.
func (p *Parser) ParseStr(str string) (parse.AST, error) {
	return p.Parse(p.tokenize(str))
}

// Parse parses the provided list of tokens, producing a parse.AST. An error is returned if the tokens provided cannot
// be parsed.
func (p *Parser) Parse(tokens []string) (parse.AST, error) {
	p.curr = 0
	p.tokens = tokens
	ast, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.curr != len(p.tokens) {
		return nil, fmt.Errorf("%w: expected end of expression, found '%s'", parse.ErrParse, p.tokens[p.curr])
	}
	return ast, nil
}

func (p *Parser) tokenize(str string) []string {
	openP, closeP := []rune(p.config[OpenParen])[0], []rune(p.config[CloseParen])[0]
	return parse.Tokenize(str, openP, closeP, p.matcher)
}

func (p *Parser) match(token Token) bool {
	if p.curr == len(p.tokens) {
		return false
	}
	curr := p.tokens[p.curr]
	if p.caseInsensitive {
		curr = strings.ToLower(curr)
	}
	if curr == p.config[token] {
		p.curr++
		return true
	}
	return false
}

func (p *Parser) peek() string {
	return p.tokens[p.curr]
}

func (p *Parser) isKeyword(str string) bool {
	if p.caseInsensitive {
		str = strings.ToLower(str)
	}
	return p.matcher.Contains(str)
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
		return &BinExpr{LHS: lhs, RHS: rhs, Op: OpAnd}, nil
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
		return &BinExpr{LHS: lhs, RHS: rhs, Op: OpOr}, nil
	}
	return lhs, nil
}

func (p *Parser) parseNot() (parse.AST, error) {
	if p.match(Not) {
		rest, err := p.parseParens()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Expr: rest, Op: OpNot}, nil
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
