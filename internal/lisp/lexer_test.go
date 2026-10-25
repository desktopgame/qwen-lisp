package lisp

import (
	"strings"
	"testing"
)

func TestLexBasic(t *testing.T) {
	toks, err := lex("(+ 1 2)")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	want := []tokenKind{tokLParen, tokSymbol, tokInt, tokInt, tokRParen}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(toks), len(want))
	}
	for i, k := range want {
		if toks[i].kind != k {
			t.Errorf("token %d: kind %v, want %v", i, toks[i].kind, k)
		}
	}
	if toks[2].val != 1 || toks[3].val != 2 {
		t.Errorf("integer values wrong: %v %v", toks[2].val, toks[3].val)
	}
}

func TestLexNegativeAndSymbols(t *testing.T) {
	toks, err := lex("-5 - +x a-b*")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if len(toks) != 4 {
		t.Fatalf("got %d tokens, want 4: %+v", len(toks), toks)
	}
	if toks[0].kind != tokInt || toks[0].val != -5 {
		t.Errorf("expected integer -5, got %v %d", toks[0].kind, toks[0].val)
	}
	for _, i := range []int{1, 2, 3} {
		if toks[i].kind != tokSymbol {
			t.Errorf("token %d: expected symbol, got %v", i, toks[i].kind)
		}
	}
}

func TestLexComment(t *testing.T) {
	toks, err := lex("; leading comment\n1 ; trailing\n2")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if len(toks) != 2 {
		t.Fatalf("got %d tokens, want 2: %+v", len(toks), toks)
	}
	if toks[1].line != 3 {
		t.Errorf("second token on line %d, want 3", toks[1].line)
	}
}

func TestLexQuoteAndDot(t *testing.T) {
	toks, err := lex(`'(1 . 2)`)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	want := []tokenKind{tokQuote, tokLParen, tokInt, tokDot, tokInt, tokRParen}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, k := range want {
		if toks[i].kind != k {
			t.Errorf("token %d: kind %v, want %v", i, toks[i].kind, k)
		}
	}
}

func TestLexDotInsideSymbol(t *testing.T) {
	toks, err := lex(`a.b`)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if len(toks) != 1 || toks[0].kind != tokSymbol || toks[0].text != "a.b" {
		t.Errorf("expected one symbol a.b, got %+v", toks)
	}
}

func TestLexFloatError(t *testing.T) {
	_, err := lex("3.14")
	if err == nil || !strings.Contains(err.Error(), "floating point") {
		t.Errorf("expected float error, got %v", err)
	}
}

func TestLexIntOutOfRange(t *testing.T) {
	_, err := lex("99999999999999999999999")
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected out-of-range error, got %v", err)
	}
}
