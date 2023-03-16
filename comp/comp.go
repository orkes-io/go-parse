// Package comp implements a recursive-descent parser for comparison expressions according to the following grammar.
//
//	expr    -> equal
//	equal   -> ordinal ( '!=' | '==' ) ordinal | ordinal
//	ordinal -> term ( '>=' | '>' | '<' | '<=' ) term | term
//	term    -> '(' expr ')' | unparsed
//	unparsed -> .*
//
// It leaves unparsed portions of the expression in parse.Unparsed nodes, for later consumption. Terms in this grammar
// match
package comp

import (
	"fmt"
	"github.com/orkes-io/go-parse"
	"strings"
)

// EqualExpr represents an equality comparison.
type EqualExpr struct {
	LHS parse.AST
	RHS parse.AST
	Op  Op // Op can only be one of OpEqual or OpNotEqual
}

func (e *EqualExpr) Parse(p parse.Parser) error {
	if unparsed, ok := e.LHS.(parse.Unparsed); ok {
		newLHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		e.LHS = newLHS
	} else if err := e.LHS.Parse(p); err != nil {
		return err
	}
	if unparsed, ok := e.RHS.(parse.Unparsed); ok {
		newRHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		e.RHS = newRHS
	} else if err := e.RHS.Parse(p); err != nil {
		return err
	}
	return nil
}

// OrdinalExpr represents a ordinal expression.
type OrdinalExpr struct {
	LHS parse.AST
	RHS parse.AST
	Op  Op // Op can only be one of OpGreater, OpLess, OpGreaterOrEqual, or OpLessOrEqual
}

func (e *OrdinalExpr) Parse(p parse.Parser) error {
	if unparsed, ok := e.LHS.(parse.Unparsed); ok {
		newLHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		e.LHS = newLHS
	} else if err := e.LHS.Parse(p); err != nil {
		return err
	}
	if unparsed, ok := e.RHS.(parse.Unparsed); ok {
		newRHS, err := p.Parse(unparsed.Contents)
		if err != nil {
			return err
		}
		e.RHS = newRHS
	} else if err := e.RHS.Parse(p); err != nil {
		return err
	}
	return nil
}

// Op represents one of six possible comparison operations recognized by this grammar.
type Op uint8

const (
	OpEqual Op = iota + 1
	OpNotEqual
	OpGreaterOrEqual
	OpGreater
	OpLessOrEqual
	OpLess
)

func (o Op) String() string {
	switch o {
	case OpEqual:
		return "=="
	case OpNotEqual:
		return "!="
	case OpGreater:
		return ">"
	case OpGreaterOrEqual:
		return ">="
	case OpLess:
		return "<"
	case OpLessOrEqual:
		return "<="
	default:
		return "unknown op"
	}
}

// Token is a token required by this grammar.
type Token uint8

const (
	Equal Token = iota + 1
	NotEqual
	GreaterOrEqual
	Greater
	LessOrEqual
	Less
	OpenParen
	CloseParen
)

type ParserOpt func(*Parser)

// Parser parses this grammar.
type Parser struct {
	config          map[Token]string
	caseInsensitive bool

	matcher *parse.KeywordTrie
	tokens  []string
	curr    int
}

// WithTokens configures the syntax used by this parser using the provided token mapping. The provided map must contain
// distinct entries for each Token provided in this package: Equal, NotEqual, Greater, GreaterOrEqual, Less,
// LessOrEqual, OpenParen, and CloseParen.
func WithTokens(config map[Token]string) ParserOpt {
	return func(parser *Parser) {
		parser.config = config
	}
}

// WithCaseSensitive can be used to set whether this parser is case sensitive.
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
			Equal:          "==",
			NotEqual:       "!=",
			Greater:        ">",
			GreaterOrEqual: ">=",
			Less:           "<",
			LessOrEqual:    "<=",
			OpenParen:      "(",
			CloseParen:     ")",
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
	if p.matcher.Count() != 8 {
		return fmt.Errorf("%w: token collision detected; at least two of the provided tokens are identical", parse.ErrConfig)
	}
	return nil
}

// ParseStr tokenizes and parses the provided string. See Parser.Parse for details.
func (p *Parser) ParseStr(str string) (parse.AST, error) {
	return p.Parse(p.tokenize(str))
}

// Parse parses the provided list of tokens, producing a parse.AST. An error is returned if the provided tokens do not
// conform to the grammar specified in this package.
func (p *Parser) Parse(tokens []string) (parse.AST, error) {
	p.curr = 0
	p.tokens = tokens
	ast, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if p.curr != len(p.tokens) {
		return nil, fmt.Errorf("%w: expected end of expression; found '%s'", parse.ErrParse, p.tokens[p.curr])
	}
	return ast, nil
}

func (p *Parser) tokenize(str string) []string {
	openP, closeP := []rune(p.config[OpenParen])[0], []rune(p.config[CloseParen])[0]
	return parse.Tokenize(str, openP, closeP, p.matcher)
}

func (p *Parser) isKeyword(str string) bool {
	if p.caseInsensitive {
		str = strings.ToLower(str)
	}
	return p.matcher.Contains(str)
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

func (p *Parser) parseExpr() (parse.AST, error) {
	return p.parseEqual()
}

func (p *Parser) parseEqual() (parse.AST, error) {
	lhs, err := p.parseOrdinal()
	if err != nil {
		return nil, err
	}
	if op := p.matchOps(Equal, NotEqual); op != 0 {
		rhs, err := p.parseOrdinal()
		if err != nil {
			return nil, err
		}
		return &EqualExpr{LHS: lhs, RHS: rhs, Op: tokenToOp(op)}, nil
	}
	return lhs, nil
}

func (p *Parser) parseOrdinal() (parse.AST, error) {
	lhs, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	if op := p.matchOps(GreaterOrEqual, LessOrEqual, Greater, Less); op != 0 {
		rhs, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		return &OrdinalExpr{LHS: lhs, RHS: rhs, Op: tokenToOp(op)}, nil
	}
	return lhs, nil
}

func (p *Parser) parseTerm() (parse.AST, error) {
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

// matchOps attempts to match all of the provided ops in order, returning the first one matched. If none match, 0 is returned.
func (p *Parser) matchOps(ops ...Token) Token {
	for _, op := range ops {
		if p.match(op) {
			return op
		}
	}
	return 0
}

func tokenToOp(t Token) Op {
	switch t {
	case Equal:
		return OpEqual
	case NotEqual:
		return OpNotEqual
	case Greater:
		return OpGreater
	case GreaterOrEqual:
		return OpGreaterOrEqual
	case Less:
		return OpLess
	case LessOrEqual:
		return OpLessOrEqual
	}
	return 0
}
