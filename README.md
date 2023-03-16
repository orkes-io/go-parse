# go-parse

![GitHub](https://img.shields.io/github/license/orkes-io/go-parse)
![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/orkes-io/go-parse)

Simple expression parsers in Go.

### Installation
```
go get github.com/orkes-io/go-parse
```

## Packages

All parsers implemented in this package perform tokenization and produce an Abstract Syntax Tree (AST)
of their results, which can be consumed by other functions.

### bools

Supports parsing boolean expressions using `AND`, `OR`, and `NOT`, according to the following grammar.
The syntax used for each operator can be configured at runtime.

```
    expr     -> and
    and      -> or 'AND' and | or
    or       -> not 'OR' or | not
    not      -> 'NOT' parens | parens
    parens   -> '(' expr ')' | unparsed
    unparsed -> '.*
```

### comp

Supports parsing comparison expressions using equality and comparison operators, according to the
following grammar. The syntax used for each operator can be configured at runtime.

```
    expr    -> equal
    equal   -> ordinal ( '!=' | '==' ) ordinal | ordinal
    ordinal -> term ( '>=' | '>' | '<' | '<=' ) term | term
    term    -> '(' expr ')' | unparsed
    unparsed -> .*
```


