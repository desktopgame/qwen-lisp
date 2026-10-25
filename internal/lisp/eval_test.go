package lisp

import (
	"bytes"
	"strings"
	"testing"
)

// expect evaluates src in a fresh interpreter and compares the value of
// the last form with want.
func expect(t *testing.T, src string, want Object) {
	t.Helper()
	in := New()
	results, err := in.EvalString(src)
	if err != nil {
		t.Fatalf("EvalString(%q): %v", src, err)
	}
	if len(results) == 0 {
		t.Fatalf("EvalString(%q): no forms evaluated", src)
	}
	got := results[len(results)-1]
	if !equal(got, want) {
		t.Errorf("%s\n  got  %s\n  want %s", src, got, want)
	}
}

// expectErr evaluates src and requires an error containing wantSub.
func expectErr(t *testing.T, src, wantSub string) {
	t.Helper()
	in := New()
	_, err := in.EvalString(src)
	if err == nil {
		t.Fatalf("%s: expected an error", src)
	}
	if !strings.Contains(err.Error(), wantSub) {
		t.Errorf("%s\n  error %q does not contain %q", src, err.Error(), wantSub)
	}
}

func TestArithmetic(t *testing.T) {
	for src, want := range map[string]Object{
		"(+ 1 2)":              Integer(3),
		"(+ 1 2 3)":            Integer(6),
		"(+)":                  Integer(0),
		"(- 10 3)":             Integer(7),
		"(- 5)":                Integer(-5),
		"(- 10 3 2)":           Integer(5),
		"(* 2 3 4)":            Integer(24),
		"(*)":                  Integer(1),
		"(/ 7 2)":              Integer(3),
		"(/ -7 2)":             Integer(-3),
		"(/ 100 5 2)":          Integer(10),
		"(+ (* 2 3) (- 10 4))": Integer(12),
		"(+ (if #t 1 2) 1)":    Integer(2),
	} {
		expect(t, src, want)
	}
}

func TestArithmeticErrors(t *testing.T) {
	expectErr(t, "(/ 1 0)", "division by zero")
	expectErr(t, "(+ 1 #t)", "expected an integer")
	expectErr(t, "(-)", "at least 1")
	expectErr(t, "(/ 1)", "at least 2")
}

func TestOverflow(t *testing.T) {
	expectErr(t, "(+ 9223372036854775807 1)", "overflow")
	expectErr(t, "(* 9223372036854775807 2)", "overflow")
	expectErr(t, "(- -9223372036854775808)", "overflow")
}

func TestComparisons(t *testing.T) {
	for src, want := range map[string]Object{
		"(= 2 2)":   Boolean(true),
		"(= 1 2)":   Boolean(false),
		"(= 1 1 1)": Boolean(true),
		"(< 1 2 3)": Boolean(true),
		"(< 1 3 2)": Boolean(false),
		"(> 3 2)":   Boolean(true),
		"(<= 1 1)":  Boolean(true),
		"(>= 1 2)":  Boolean(false),
	} {
		expect(t, src, want)
	}
	expect(t, "(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (= (fact 5) 120)", Boolean(true))
}

func TestDefineAndLookup(t *testing.T) {
	expect(t, "(define x 5) x", Integer(5))
	expect(t, "(define x 5) (define y (+ x 1)) y", Integer(6))
	expect(t, "(define x 5) (set! x 10) x", Integer(10))
	expectErr(t, "y", "unbound symbol")
	expectErr(t, "(set! zz 1)", "unbound symbol")
}

func TestIf(t *testing.T) {
	expect(t, "(if #t 1 2)", Integer(1))
	expect(t, "(if #f 1 2)", Integer(2))
	expect(t, "(if (= 1 1) 'yes 'no)", Symbol("yes"))
	expect(t, "(if #f 1)", Nil)
	expect(t, `(if '() 'truthy 'falsy)`, Symbol("truthy")) // Scheme: only #f is false
	expectErr(t, "(if)", "if: expected at least")
	expectErr(t, "(if #t 1 2 3)", "at most 3")
}

func TestLambda(t *testing.T) {
	expect(t, "((lambda (x) (* x x)) 5)", Integer(25))
	expect(t, "((lambda () 42))", Integer(42))
	expect(t, "(define square (lambda (x) (* x x))) (square 7)", Integer(49))
	expectErr(t, "((lambda (x) x))", "at least 1")
	expectErr(t, "((lambda (x) x) 1 2)", "expected 1 argument")
	expectErr(t, "((lambda (x 'y) x) 1 2)", "parameter name")
}

func TestClosures(t *testing.T) {
	expect(t, `
	  (define c (let ((n 0)) (lambda () (set! n (+ n 1)) n)))
	  (list (c) (c) (c))`,
		listFrom([]Object{Integer(1), Integer(2), Integer(3)}, Nil))

	// Two closures keep independent state.
	expect(t, `
	  (define a (let ((n 0)) (lambda () (set! n (+ n 1)) n)))
	  (define b (let ((n 100)) (lambda () (set! n (+ n 1)) n)))
	  (list (a) (a) (b))`,
		listFrom([]Object{Integer(1), Integer(2), Integer(101)}, Nil))

	// Captured environment, not dynamic scope.
	expect(t, `
	  (define x 1)
	  (define get-x (lambda () x))
	  (define call-with-2 (lambda (f) (define x 2) (f)))
	  (call-with-2 get-x)`,
		Integer(1))
}

func TestRecursion(t *testing.T) {
	expect(t, `(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (fact 10)`, Integer(3628800))
	expect(t, `(define (fib n) (if (< n 2) n (+ (fib (- n 1)) (fib (- n 2))))) (fib 15)`, Integer(610))
	// Recursion well within the depth budget.
	expect(t, `(define (sum n acc) (if (= n 0) acc (sum (- n 1) (+ acc n)))) (sum 2000 0)`,
		Integer(2001000))
	// Runaway recursion produces an error instead of crashing.
	expectErr(t, "(define (loop) (loop)) (loop)", "maximum recursion depth")
}

func TestLet(t *testing.T) {
	expect(t, "(let ((x 2) (y 3)) (+ x y))", Integer(5))
	expect(t, "(let () 7)", Integer(7))
	expect(t, "(let ((x 2)) (let ((x 3) (y x)) y))", Integer(2)) // values use outer scope
	expect(t, "(define x 10) (let ((x 1)) x)", Integer(1))       // shadowing
	expect(t, "(define x 10) (let ((x 1)) x) x", Integer(10))    // outer unaffected
	expectErr(t, "(let ((x)) x)", "binding of the form")
	expectErr(t, "(let ((1 2)) 3)", "symbol in a binding")
}

func TestQuoteAndPairs(t *testing.T) {
	expect(t, "(quote (1 2 3))", listFrom([]Object{Integer(1), Integer(2), Integer(3)}, Nil))
	expect(t, "'a", Symbol("a"))
	expect(t, "(car '(1 2 3))", Integer(1))
	expect(t, "(cdr '(1 2 3))", listFrom([]Object{Integer(2), Integer(3)}, Nil))
	expect(t, "(cons 1 '(2 3))", listFrom([]Object{Integer(1), Integer(2), Integer(3)}, Nil))
	expect(t, "(cons 1 2)", &Pair{Car: Integer(1), Cdr: Integer(2)})
	expect(t, "(car ''a)", Symbol("quote"))
	expectErr(t, "(car 1)", "expected a pair")
	expectErr(t, "(cdr 1)", "expected a pair")
	expectErr(t, "(quote 1 2)", "exactly 1")
}

func TestListBuiltins(t *testing.T) {
	expect(t, "(list 1 2 3)", listFrom([]Object{Integer(1), Integer(2), Integer(3)}, Nil))
	expect(t, "(length '(1 2 3))", Integer(3))
	expect(t, "(length '())", Integer(0))
	expect(t, "(null? '())", Boolean(true))
	expect(t, "(null? '(1))", Boolean(false))
	expect(t, "(pair? '(1))", Boolean(true))
	expect(t, "(pair? '())", Boolean(false))
	expect(t, "(list? '(1 . 2))", Boolean(false))
	expect(t, "(append '(1 2) '(3) '())", listFrom([]Object{Integer(1), Integer(2), Integer(3)}, Nil))
	expect(t, "(eq? 'a 'a)", Boolean(true))
	expect(t, "(eq? '(1) '(1))", Boolean(false)) // identity for pairs
	expect(t, "(equal? '(1 (2 3)) '(1 (2 3)))", Boolean(true))
	expect(t, "(not #f)", Boolean(true))
	expect(t, "(not 0)", Boolean(false))
	expectErr(t, "(length '(1 . 2))", "expected a list")
}

func TestVariadicProcedures(t *testing.T) {
	expect(t, "((lambda args (length args)) 1 2 3)", Integer(3))
	expect(t, "((lambda args args))", Nil)
	expect(t, "(define (f a . b) b) (f 1 2 3)", listFrom([]Object{Integer(2), Integer(3)}, Nil))
	expect(t, "(define (g . xs) (length xs)) (g)", Integer(0))
}

func TestAndOrCondBegin(t *testing.T) {
	expect(t, "(and 1 2 3)", Integer(3))
	expect(t, "(and 1 #f 3)", Boolean(false))
	expect(t, "(and)", Boolean(true))
	expect(t, "(or)", Boolean(false))
	expect(t, "(or #f 2 3)", Integer(2))
	expect(t, "(or 1 undefined-symbol-error)", Integer(1)) // short-circuit
	expect(t, "(and #f (car 42))", Boolean(false))         // short-circuit
	expect(t, `(cond (#f 1) (#t 2) (else 3))`, Integer(2))
	expect(t, `(cond (#f 1) (else 'fallback))`, Symbol("fallback"))
	expect(t, `(cond ((= 1 2)) (else 9))`, Integer(9))
	expect(t, "(begin 1 2 3)", Integer(3))
	expect(t, "(define x 0) (begin (set! x 1) (set! x (+ x 1))) x", Integer(2))
}

func TestCallErrors(t *testing.T) {
	expectErr(t, "(1 2)", "not a procedure")
	expectErr(t, "(undefined-fn 1)", "unbound symbol")
	expectErr(t, "((lambda (a b) a) 1)", "at least 2")
	expectErr(t, "(cons)", "expected 2 argument")
}

func TestDisplayToOut(t *testing.T) {
	in := New()
	var buf bytes.Buffer
	in.Out = &buf
	if _, err := in.EvalString(`(display 'hi) (newline) (display '(1 2))`); err != nil {
		t.Fatalf("EvalString: %v", err)
	}
	if got := buf.String(); got != "hi\n(1 2)" {
		t.Errorf("Out = %q, want %q", got, "hi\n(1 2)")
	}
}

func TestBooleans(t *testing.T) {
	expect(t, "#t", Boolean(true))
	expect(t, "#f", Boolean(false))
	expect(t, "true", Boolean(true))
	expect(t, "false", Boolean(false))
}
