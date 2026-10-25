package lisp

import (
	"strings"
	"testing"
)

func TestParseAtoms(t *testing.T) {
	for src, want := range map[string]Object{
		"42":      Integer(42),
		"-7":      Integer(-7),
		"foo":     Symbol("foo"),
		"#t":      Symbol("#t"), // parsed as symbol; bound in the global env
		"a-b*?+1": Symbol("a-b*?+1"),
	} {
		got, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if !equal(got, want) {
			t.Errorf("Parse(%q) = %s (%T), want %s (%T)", src, got, got, want, want)
		}
	}
}

func TestParseEmptyList(t *testing.T) {
	got, err := Parse("()")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !isNil(got) {
		t.Errorf("got %s, want ()", got)
	}
}

func TestParseNested(t *testing.T) {
	e, err := Parse("(+ 1 (* 2 3))")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &Pair{Car: Symbol("+"), Cdr: &Pair{Car: Integer(1), Cdr: &Pair{
		Car: &Pair{Car: Symbol("*"), Cdr: &Pair{Car: Integer(2), Cdr: &Pair{Car: Integer(3), Cdr: Nil}}},
		Cdr: Nil,
	}}}
	if !equal(e, want) {
		t.Errorf("got %s, want %s", e, want)
	}
}

func TestParseQuoteAbbrev(t *testing.T) {
	e, err := Parse("''a")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &Pair{Car: Symbol("quote"), Cdr: &Pair{
		Car: &Pair{Car: Symbol("quote"), Cdr: &Pair{Car: Symbol("a"), Cdr: Nil}},
		Cdr: Nil,
	}}
	if !equal(e, want) {
		t.Errorf("got %s, want %s", e, want)
	}
}

func TestParseDottedPair(t *testing.T) {
	e, err := Parse("(1 2 . 3)")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &Pair{Car: Integer(1), Cdr: &Pair{Car: Integer(2), Cdr: Integer(3)}}
	if !equal(e, want) {
		t.Errorf("got %s, want %s", e, want)
	}
}

func TestParseAllMultiple(t *testing.T) {
	forms, err := ParseAll("1 (+ 1 2)\n'x")
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if len(forms) != 3 {
		t.Fatalf("got %d forms, want 3", len(forms))
	}
	if !equal(forms[2], &Pair{Car: Symbol("quote"), Cdr: &Pair{Car: Symbol("x"), Cdr: Nil}}) {
		t.Errorf("third form = %s", forms[2])
	}
}

func TestParseAllEmpty(t *testing.T) {
	forms, err := ParseAll("  ; nothing here\n\n")
	if err != nil {
		t.Fatalf("ParseAll: %v", err)
	}
	if len(forms) != 0 {
		t.Errorf("got %d forms, want 0", len(forms))
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"(1 2", "unclosed"},
		{"(1 2))", "unexpected ')'"},
		{"(+ . )", "unexpected ')'"},
		{"(a . b c)", "after '.'"},
		{".", "unexpected '.'"},
		{"(+ 1\n  ", "unexpected end of input"},
		{"1 )", "after expression"},
	}
	for _, c := range cases {
		_, err := Parse(c.src)
		if err == nil {
			t.Errorf("Parse(%q): expected error", c.src)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("Parse(%q) error = %q, want it to contain %q", c.src, err.Error(), c.want)
		}
	}
}

func TestParseErrorLineNumbers(t *testing.T) {
	_, err := Parse("\n\n(1 2")
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("expected line 3 in error, got %v", err)
	}
}

func TestParseDepthLimit(t *testing.T) {
	deep := strings.Repeat("(", 5000) + "1" + strings.Repeat(")", 5000)
	_, err := Parse(deep)
	if err == nil || !strings.Contains(err.Error(), "nests too deeply") {
		t.Errorf("expected nesting error, got %v", err)
	}
}

func TestPairString(t *testing.T) {
	cases := []struct {
		o    Object
		want string
	}{
		{listFrom([]Object{Integer(1), Integer(2)}, Nil), "(1 2)"},
		{listFrom([]Object{Integer(1)}, Integer(2)), "(1 . 2)"},
		{Nil, "()"},
		{listFrom([]Object{Symbol("quote"), Symbol("a")}, Nil), "(quote a)"},
	}
	for _, c := range cases {
		if got := c.o.String(); got != c.want {
			t.Errorf("String() = %q, want %q", got, c.want)
		}
	}
}
