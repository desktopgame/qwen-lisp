// Command qwen-lisp is a REPL for the small Scheme-style Lisp interpreter
// in internal/lisp.
//
// Usage:
//
//	go run .                        # start the REPL
//	go run . examples/demo.scm      # run script file(s)
//	go run . -e "(+ 1 2)"           # evaluate an expression and exit
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"qwen-lisp/internal/lisp"
)

func main() {
	expr := flag.String("e", "", "evaluate the given expression(s) and exit")
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "A small Scheme-style Lisp interpreter.\n\n")
		fmt.Fprintf(out, "Usage:\n  %s [options] [file ...]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	in := lisp.New()

	switch {
	case *expr != "":
		if err := printResults(in, *expr); err != nil {
			fail(err)
		}
	case flag.NArg() > 0:
		for _, path := range flag.Args() {
			data, err := os.ReadFile(path)
			if err != nil {
				fail(err)
			}
			if _, err := in.EvalString(string(data)); err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", path, err)
				os.Exit(1)
			}
		}
	default:
		repl(in)
	}
}

// printResults evaluates src and prints the value of every form
// (used by -e, where seeing results is the point).
func printResults(in *lisp.Interpreter, src string) error {
	results, err := in.EvalString(src)
	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Println(r)
	}
	return nil
}

func repl(in *lisp.Interpreter) {
	if interactive(os.Stdin) {
		fmt.Println("qwen-lisp — a small Scheme-style Lisp interpreter. Press Ctrl-D to exit.")
	}
	scanner := bufio.NewScanner(os.Stdin)
	var buf strings.Builder
	prompt := "lisp> "
	for {
		fmt.Print(prompt)
		if !scanner.Scan() {
			// EOF: report any incomplete input instead of dropping it silently.
			if strings.TrimSpace(buf.String()) != "" {
				if _, err := in.EvalString(buf.String()); err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
				}
			}
			break
		}
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
		if !balanced(buf.String()) {
			prompt = "  ... "
			continue
		}
		src := buf.String()
		buf.Reset()
		prompt = "lisp> "
		if strings.TrimSpace(src) == "" {
			continue
		}
		results, err := in.EvalString(src)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		for _, r := range results {
			fmt.Println(r)
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintln(os.Stderr, "error:", scanner.Err())
		os.Exit(1)
	}
}

// balanced reports whether every '(' in s has a matching ')', ignoring
// line comments. A negative depth (extra ')') also counts as "ready" so
// the parser can report the error instead of the REPL hanging.
func balanced(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case ';':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth < 0 {
			return true
		}
	}
	return depth == 0
}

func interactive(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
