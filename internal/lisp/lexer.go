package lisp

import (
	"errors"
	"fmt"
	"strconv"
)

type tokenKind int

const (
	tokLParen tokenKind = iota
	tokRParen
	tokQuote
	tokDot
	tokInt
	tokSymbol
)

type token struct {
	kind tokenKind
	text string
	val  int64
	line int
}

func (t token) describe() string {
	switch t.kind {
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokQuote:
		return "'"
	case tokDot:
		return "'.'"
	default:
		return strconv.Quote(t.text)
	}
}

func isDelim(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '(', ')', '\'', ';':
		return true
	}
	return false
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// looksLikeInt reports whether s is an optionally signed run of digits.
func looksLikeInt(s string) bool {
	if s != "" && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigit(s[i]) {
			return false
		}
	}
	return true
}

// lex converts source into a token slice. It reports lexical errors with
// 1-based line numbers.
func lex(src string) ([]token, error) {
	var toks []token
	line := 1
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == ';': // line comment
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '(':
			toks = append(toks, token{kind: tokLParen, text: "(", line: line})
			i++
		case c == ')':
			toks = append(toks, token{kind: tokRParen, text: ")", line: line})
			i++
		case c == '\'':
			toks = append(toks, token{kind: tokQuote, text: "'", line: line})
			i++
		case c == '.' && (i+1 >= len(src) || isDelim(src[i+1])):
			toks = append(toks, token{kind: tokDot, text: ".", line: line})
			i++
		default:
			j := i
			for j < len(src) && !isDelim(src[j]) {
				j++
			}
			word := src[i:j]
			i = j
			n, err := strconv.ParseInt(word, 10, 64)
			switch {
			case err == nil:
				toks = append(toks, token{kind: tokInt, text: word, val: n, line: line})
			case errors.Is(err, strconv.ErrRange) && looksLikeInt(word):
				return nil, fmt.Errorf("line %d: integer literal out of range: %s", line, word)
			default:
				if _, ferr := strconv.ParseFloat(word, 64); ferr == nil {
					return nil, fmt.Errorf("line %d: floating point literals are not supported: %s", line, word)
				}
				toks = append(toks, token{kind: tokSymbol, text: word, line: line})
			}
		}
	}
	return toks, nil
}
