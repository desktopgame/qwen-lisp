package lisp

import "fmt"

// maxParseDepth bounds expression nesting so pathological input cannot
// overflow the Go stack.
const maxParseDepth = 1000

type parser struct {
	toks  []token
	pos   int
	depth int
}

// Parse parses exactly one expression from src.
func Parse(src string) (Object, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	if p.eof() {
		return nil, fmt.Errorf("unexpected end of input: no expression")
	}
	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.eof() {
		return nil, fmt.Errorf("line %d: unexpected %s after expression", p.cur().line, p.cur().describe())
	}
	return e, nil
}

// ParseAll parses every expression in src, in order.
func ParseAll(src string) ([]Object, error) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	var forms []Object
	for !p.eof() {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		forms = append(forms, e)
	}
	return forms, nil
}

func (p *parser) eof() bool  { return p.pos >= len(p.toks) }
func (p *parser) cur() token { return p.toks[p.pos] }

func (p *parser) next() token {
	t := p.toks[p.pos]
	p.pos++
	return t
}

func (p *parser) parseExpr() (Object, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxParseDepth {
		return nil, fmt.Errorf("expression nests too deeply (max %d)", maxParseDepth)
	}
	if p.eof() {
		return nil, fmt.Errorf("unexpected end of input")
	}
	t := p.next()
	switch t.kind {
	case tokInt:
		return Integer(t.val), nil
	case tokSymbol:
		return Symbol(t.text), nil
	case tokQuote:
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		return &Pair{Car: Symbol("quote"), Cdr: &Pair{Car: e, Cdr: Nil}}, nil
	case tokLParen:
		return p.parseList(t)
	case tokRParen:
		return nil, fmt.Errorf("line %d: unexpected ')'", t.line)
	case tokDot:
		return nil, fmt.Errorf("line %d: unexpected '.' outside of a list", t.line)
	}
	return nil, fmt.Errorf("line %d: unexpected token", t.line)
}

// parseList parses the remainder of a list whose '(' was just consumed.
func (p *parser) parseList(open token) (Object, error) {
	var items []Object
	var tail Object = Nil
	for {
		if p.eof() {
			return nil, fmt.Errorf("line %d: unclosed '(' (unexpected end of input)", open.line)
		}
		t := p.cur()
		switch t.kind {
		case tokRParen:
			p.pos++
			return listFrom(items, tail), nil
		case tokDot:
			p.pos++
			e, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if p.eof() {
				return nil, fmt.Errorf("line %d: unclosed '(' (unexpected end of input)", open.line)
			}
			if p.cur().kind != tokRParen {
				return nil, fmt.Errorf("line %d: unexpected %s after '.' (expected ')')", p.cur().line, p.cur().describe())
			}
			p.pos++
			return listFrom(items, e), nil
		default:
			e, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			items = append(items, e)
		}
	}
}
