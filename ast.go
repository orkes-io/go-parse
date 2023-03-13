package parse

import "errors"

var ErrConfig = errors.New("config error")
var ErrParse = errors.New("error parsing")

type AST interface {
	IsAST() // marker interface
}

// Unparsed represents a list of unparsed tokens in an expression.
type Unparsed struct {
	Contents []string
}

func (u Unparsed) IsAST() {} // marker interface
