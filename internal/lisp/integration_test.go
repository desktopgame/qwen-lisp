package lisp

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestDemoScript evaluates examples/demo.scm end to end and checks the
// output produced through display/newline.
func TestDemoScript(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "demo.scm")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	in := New()
	var buf bytes.Buffer
	in.Out = &buf
	if _, err := in.EvalString(string(src)); err != nil {
		t.Fatalf("evaluating demo.scm: %v", err)
	}
	want := `3628800
6765
(0 5 10 100 101)
(hello world)
(1 2 3 4 5 6 7)
(0 1 2 3 4 5)
(1 4 9 16 25)
(negative zero positive)
5
`
	if got := buf.String(); got != want {
		t.Errorf("demo output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
