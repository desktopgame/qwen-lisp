// Package lisp implements a small Scheme-style Lisp interpreter:
// lexer, parser, tree-walking evaluator with lexical closures.
package lisp

import (
	"strconv"
	"strings"
)

// Object is the type of every Lisp value. Concrete types are
// Integer, Boolean, Symbol, NilType, *Pair, *Builtin and *Closure.
type Object interface {
	String() string
}

// Integer is a 64-bit signed integer.
type Integer int64

func (i Integer) String() string { return strconv.FormatInt(int64(i), 10) }

// Boolean is #t or #f.
type Boolean bool

func (b Boolean) String() string {
	if b {
		return "#t"
	}
	return "#f"
}

// Symbol is an interned name.
type Symbol string

func (s Symbol) String() string { return string(s) }

// NilType is the empty list ().
type NilType struct{}

// Nil is the single empty-list value.
var Nil = NilType{}

func (NilType) String() string { return "()" }

// Pair is a cons cell; lists are chains of pairs terminated by Nil.
type Pair struct {
	Car Object
	Cdr Object
}

func (p *Pair) String() string {
	var sb strings.Builder
	sb.WriteByte('(')
	cur := p
	for {
		sb.WriteString(cur.Car.String())
		next, ok := cur.Cdr.(*Pair)
		if !ok {
			if !isNil(cur.Cdr) {
				sb.WriteString(" . ")
				sb.WriteString(cur.Cdr.String())
			}
			break
		}
		sb.WriteByte(' ')
		cur = next
	}
	sb.WriteByte(')')
	return sb.String()
}

// Builtin is a primitive procedure implemented in Go.
type Builtin struct {
	Name string
	Fn   func(args []Object) (Object, error)
}

func (b *Builtin) String() string { return "#<builtin " + b.Name + ">" }

// Closure is a user-defined procedure capturing its defining environment.
type Closure struct {
	Name   string   // "" when anonymous
	Params []Symbol // fixed parameters
	Rest   Symbol   // variadic tail name, "" when none
	Body   []Object
	Env    *Env
}

func (c *Closure) String() string {
	name := c.Name
	if name == "" {
		name = "lambda"
	}
	return "#<procedure " + name + ">"
}

func isNil(o Object) bool {
	_, ok := o.(NilType)
	return ok
}

// typeName returns the human-readable type of an object.
func typeName(o Object) string {
	switch o.(type) {
	case Integer:
		return "integer"
	case Boolean:
		return "boolean"
	case Symbol:
		return "symbol"
	case NilType:
		return "empty list"
	case *Pair:
		return "list"
	case *Builtin, *Closure:
		return "procedure"
	}
	return "object"
}

// describe formats "type value" for error messages.
func describe(o Object) string { return typeName(o) + " " + o.String() }

// listFrom builds a list from items, terminated by tail.
func listFrom(items []Object, tail Object) Object {
	result := tail
	for i := len(items) - 1; i >= 0; i-- {
		result = &Pair{Car: items[i], Cdr: result}
	}
	return result
}

// listSlice walks a proper list and returns its elements.
// ok is false when o is not a proper list.
func listSlice(o Object) (items []Object, ok bool) {
	cur := o
	for !isNil(cur) {
		p, isPair := cur.(*Pair)
		if !isPair {
			return nil, false
		}
		items = append(items, p.Car)
		cur = p.Cdr
	}
	return items, true
}

// equal compares two objects deeply (lists element-wise, procedures by identity).
func equal(a, b Object) bool {
	switch x := a.(type) {
	case Integer:
		y, ok := b.(Integer)
		return ok && x == y
	case Boolean:
		y, ok := b.(Boolean)
		return ok && x == y
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case NilType:
		return isNil(b)
	case *Pair:
		y, ok := b.(*Pair)
		if !ok {
			return false
		}
		if !equal(x.Car, y.Car) {
			return false
		}
		return equal(x.Cdr, y.Cdr)
	default:
		return a == b
	}
}

// eqv compares numbers, booleans, symbols and nil by value,
// everything else by identity.
func eqv(a, b Object) bool {
	switch x := a.(type) {
	case Integer:
		y, ok := b.(Integer)
		return ok && x == y
	case Boolean:
		y, ok := b.(Boolean)
		return ok && x == y
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case NilType:
		return isNil(b)
	default:
		return a == b
	}
}
