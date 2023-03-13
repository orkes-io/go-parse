package parse

import (
	"errors"
	"fmt"
	"unicode"
)

// ErrConfig is returned when an error occurs configuring a Parser.
var ErrConfig = errors.New("config error")

// ErrParse is returned when an error occurs during parsing.
var ErrParse = errors.New("error parsing")

// ErrEval is returned when an error occurs during evaluation.
var ErrEval = errors.New("eval error")

// ErrUnknownAST is returned by Interpreters to signal that they are unprepared to evaluate nodes of unknown type.
var ErrUnknownAST = errors.New("unknown AST node")

// An AST is a node in an abstract syntax tree. The ASTs provided by this package are extensible. Care must be taken to
// ensure that no parsing ambiguities are introduced.
type AST interface {
	// Parse recursively parses and replaces all Unparsed nodes found in this portion of the AST.
	Parse(Parser) error
}

// Unparsed represents a list of unparsed tokens in an expression.
type Unparsed struct {
	Contents []string
}

func (u Unparsed) Parse(p Parser) error {
	// Parse calls should never make it to an Unparsed node.
	return fmt.Errorf("%w: attempted to parse Unparsed node", ErrParse)
}

// A Parser knows how to turn a slice of tokens into AST nodes.
type Parser interface {
	Parse(tokens []string) (AST, error)
}

// An Interpreter provides a way to interpret an AST, producing a value of type T. If an Interpreter ever
// finds a node with an unrecognized type, it must return ErrUnknownAST.
type Interpreter[T any] func(AST) (T, error)

// WithFallback uses the provided interpreter as a fallback, in case this interpreter finds an AST node it doesn't know
// how to interpret, the provided fallback will be used.
func (i Interpreter[T]) WithFallback(b Interpreter[T]) Interpreter[T] {
	return func(ast AST) (T, error) {
		a, err := i(ast)
		if errors.Is(err, ErrUnknownAST) {
			return b(ast)
		}
		return a, err
	}
}

// Tokenize is a general-purpose expression tokenizer which handles keywords according to the isKeyword func passed.
// Open and close braces must be single runes and are handled according to the provided runes.
func Tokenize(str string, open, close rune, isKeyword func(string) bool) []string {
	runes := []rune(str)
	var substr []rune
	var result []string
	for i := range runes {
		if runes[i] == open || runes[i] == close {
			if len(substr) != 0 {
				result = append(result, string(substr))
				substr = nil
			}
			result = append(result, string(runes[i]))
			continue
		}
		if unicode.IsSpace(runes[i]) {
			if len(substr) > 0 {
				result = append(result, string(substr))
				substr = nil
			}
			continue
		}
		substr = append(substr, runes[i])
		if isKeyword(string(substr)) {
			result = append(result, string(substr))
			substr = nil
			continue
		}
	}
	if len(substr) > 0 {
		result = append(result, string(substr))
	}
	return result
}
