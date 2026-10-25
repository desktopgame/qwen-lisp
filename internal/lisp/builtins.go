package lisp

import (
	"fmt"
	"math"
)

// installBuiltins populates the global environment with primitives.
func (in *Interpreter) installBuiltins() {
	e := in.Env
	e.Define("#t", Boolean(true))
	e.Define("#f", Boolean(false))
	e.Define("true", Boolean(true))
	e.Define("false", Boolean(false))

	def := func(name string, fn func(args []Object) (Object, error)) {
		e.Define(name, &Builtin{Name: name, Fn: fn})
	}

	// --- arithmetic ---
	def("+", func(args []Object) (Object, error) {
		sum := int64(0)
		for _, a := range args {
			n, err := requireInt("+", a)
			if err != nil {
				return nil, err
			}
			var ok bool
			sum, ok = addInt(sum, n)
			if !ok {
				return nil, fmt.Errorf("+: integer overflow")
			}
		}
		return Integer(sum), nil
	})

	def("-", func(args []Object) (Object, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("-: expected at least 1 argument, got 0")
		}
		first, err := requireInt("-", args[0])
		if err != nil {
			return nil, err
		}
		if len(args) == 1 {
			if first == math.MinInt64 {
				return nil, fmt.Errorf("-: integer overflow")
			}
			return Integer(-first), nil
		}
		acc := first
		for _, a := range args[1:] {
			n, err := requireInt("-", a)
			if err != nil {
				return nil, err
			}
			var ok bool
			acc, ok = subInt(acc, n)
			if !ok {
				return nil, fmt.Errorf("-: integer overflow")
			}
		}
		return Integer(acc), nil
	})

	def("*", func(args []Object) (Object, error) {
		prod := int64(1)
		for _, a := range args {
			n, err := requireInt("*", a)
			if err != nil {
				return nil, err
			}
			var ok bool
			prod, ok = mulInt(prod, n)
			if !ok {
				return nil, fmt.Errorf("*: integer overflow")
			}
		}
		return Integer(prod), nil
	})

	def("/", func(args []Object) (Object, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("/: expected at least 2 arguments, got %d", len(args))
		}
		acc, err := requireInt("/", args[0])
		if err != nil {
			return nil, err
		}
		for _, a := range args[1:] {
			n, err := requireInt("/", a)
			if err != nil {
				return nil, err
			}
			if n == 0 {
				return nil, fmt.Errorf("/: division by zero")
			}
			if acc == math.MinInt64 && n == -1 {
				return nil, fmt.Errorf("/: integer overflow")
			}
			acc /= n
		}
		return Integer(acc), nil
	})

	// --- comparisons ---
	chain := func(name string, cmp func(a, b int64) bool) func([]Object) (Object, error) {
		return func(args []Object) (Object, error) {
			if len(args) < 2 {
				return nil, fmt.Errorf("%s: expected at least 2 arguments, got %d", name, len(args))
			}
			nums := make([]int64, len(args))
			for i, a := range args {
				n, err := requireInt(name, a)
				if err != nil {
					return nil, err
				}
				nums[i] = n
			}
			for i := 0; i+1 < len(nums); i++ {
				if !cmp(nums[i], nums[i+1]) {
					return Boolean(false), nil
				}
			}
			return Boolean(true), nil
		}
	}
	def("=", chain("=", func(a, b int64) bool { return a == b }))
	def("<", chain("<", func(a, b int64) bool { return a < b }))
	def(">", chain(">", func(a, b int64) bool { return a > b }))
	def("<=", chain("<=", func(a, b int64) bool { return a <= b }))
	def(">=", chain(">=", func(a, b int64) bool { return a >= b }))

	def("eq?", binary("eq?", func(a, b Object) (Object, error) { return Boolean(eqv(a, b)), nil }))
	def("equal?", binary("equal?", func(a, b Object) (Object, error) { return Boolean(equal(a, b)), nil }))

	// --- logic ---
	def("not", func(args []Object) (Object, error) {
		if len(args) != 1 {
			return nil, arityErr("not", 1, args)
		}
		return Boolean(!truthy(args[0])), nil
	})

	// --- pairs and lists ---
	def("cons", binary("cons", func(a, b Object) (Object, error) {
		return &Pair{Car: a, Cdr: b}, nil
	}))

	def("car", unary("car", func(a Object) (Object, error) {
		p, ok := a.(*Pair)
		if !ok {
			return nil, fmt.Errorf("car: expected a pair, got %s", describe(a))
		}
		return p.Car, nil
	}))

	def("cdr", unary("cdr", func(a Object) (Object, error) {
		p, ok := a.(*Pair)
		if !ok {
			return nil, fmt.Errorf("cdr: expected a pair, got %s", describe(a))
		}
		return p.Cdr, nil
	}))

	def("list", func(args []Object) (Object, error) {
		return listFrom(args, Nil), nil
	})

	pred := func(name string, f func(Object) bool) func([]Object) (Object, error) {
		return func(args []Object) (Object, error) {
			if len(args) != 1 {
				return nil, arityErr(name, 1, args)
			}
			return Boolean(f(args[0])), nil
		}
	}
	def("null?", pred("null?", isNil))
	def("pair?", pred("pair?", func(o Object) bool { _, ok := o.(*Pair); return ok }))
	def("list?", pred("list?", func(o Object) bool { _, ok := listSlice(o); return ok }))
	def("symbol?", pred("symbol?", func(o Object) bool { _, ok := o.(Symbol); return ok }))
	def("integer?", pred("integer?", func(o Object) bool { _, ok := o.(Integer); return ok }))
	def("procedure?", pred("procedure?", func(o Object) bool {
		switch o.(type) {
		case *Builtin, *Closure:
			return true
		}
		return false
	}))

	def("length", func(args []Object) (Object, error) {
		if len(args) != 1 {
			return nil, arityErr("length", 1, args)
		}
		items, ok := listSlice(args[0])
		if !ok {
			return nil, fmt.Errorf("length: expected a list, got %s", describe(args[0]))
		}
		return Integer(len(items)), nil
	})

	def("append", func(args []Object) (Object, error) {
		var all []Object
		for _, a := range args {
			items, ok := listSlice(a)
			if !ok {
				return nil, fmt.Errorf("append: expected a list, got %s", describe(a))
			}
			all = append(all, items...)
		}
		return listFrom(all, Nil), nil
	})

	// --- output ---
	def("display", func(args []Object) (Object, error) {
		if len(args) != 1 {
			return nil, arityErr("display", 1, args)
		}
		fmt.Fprint(in.Out, args[0].String())
		return Nil, nil
	})
	def("newline", func(args []Object) (Object, error) {
		if len(args) != 0 {
			return nil, arityErr("newline", 0, args)
		}
		fmt.Fprintln(in.Out)
		return Nil, nil
	})
}

func arityErr(name string, want int, args []Object) error {
	return fmt.Errorf("%s: expected %d argument(s), got %d", name, want, len(args))
}

func requireInt(op string, o Object) (int64, error) {
	if i, ok := o.(Integer); ok {
		return int64(i), nil
	}
	return 0, fmt.Errorf("%s: expected an integer, got %s", op, describe(o))
}

func unary(name string, f func(Object) (Object, error)) func([]Object) (Object, error) {
	return func(args []Object) (Object, error) {
		if len(args) != 1 {
			return nil, arityErr(name, 1, args)
		}
		return f(args[0])
	}
}

func binary(name string, f func(a, b Object) (Object, error)) func([]Object) (Object, error) {
	return func(args []Object) (Object, error) {
		if len(args) != 2 {
			return nil, arityErr(name, 2, args)
		}
		return f(args[0], args[1])
	}
}

// Checked integer arithmetic. Go wraps on overflow; these detect it.
func addInt(a, b int64) (int64, bool) {
	c := a + b
	if (a < 0) == (b < 0) && (c < 0) != (a < 0) {
		return 0, false
	}
	return c, true
}

func subInt(a, b int64) (int64, bool) {
	c := a - b
	if (a < 0) != (b < 0) && (c < 0) != (a < 0) {
		return 0, false
	}
	return c, true
}

func mulInt(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if c/a != b {
		return 0, false
	}
	return c, true
}
