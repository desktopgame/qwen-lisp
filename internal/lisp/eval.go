package lisp

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// maxDepth bounds evaluator recursion so runaway programs produce an
// error instead of crashing the process.
const maxDepth = 10000

// Interpreter holds the global environment and evaluation state.
type Interpreter struct {
	Env   *Env
	Out   io.Writer // destination of display/newline
	depth int
}

// New creates an interpreter with all builtins installed.
func New() *Interpreter {
	in := &Interpreter{Out: os.Stdout}
	in.Env = NewEnv(nil)
	in.installBuiltins()
	return in
}

var errTooDeep = errors.New("maximum recursion depth exceeded")

// EvalString parses src and evaluates every form in the global environment.
func (in *Interpreter) EvalString(src string) ([]Object, error) {
	forms, err := ParseAll(src)
	if err != nil {
		return nil, err
	}
	results := make([]Object, 0, len(forms))
	for _, f := range forms {
		v, err := in.Eval(f, in.Env)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}

// Eval evaluates one expression in env.
func (in *Interpreter) Eval(exp Object, env *Env) (Object, error) {
	in.depth++
	defer func() { in.depth-- }()
	if in.depth > maxDepth {
		return nil, errTooDeep
	}
	switch e := exp.(type) {
	case Integer, Boolean, NilType, *Builtin, *Closure:
		return e, nil
	case Symbol:
		if v, ok := env.Get(string(e)); ok {
			return v, nil
		}
		return nil, fmt.Errorf("unbound symbol: %s", e)
	case *Pair:
		return in.evalList(e, env)
	}
	return nil, fmt.Errorf("cannot evaluate %s", describe(exp))
}

// truthy implements Scheme truthiness: only #f is false.
func truthy(o Object) bool {
	if b, ok := o.(Boolean); ok {
		return bool(b)
	}
	return true
}

func (in *Interpreter) evalSequence(exps []Object, env *Env) (Object, error) {
	var result Object = Nil
	for _, e := range exps {
		v, err := in.Eval(e, env)
		if err != nil {
			return nil, err
		}
		result = v
	}
	return result, nil
}

func (in *Interpreter) evalList(p *Pair, env *Env) (Object, error) {
	if sym, ok := p.Car.(Symbol); ok {
		if form, ok := specialForms[sym]; ok {
			return form(in, p.Cdr, env)
		}
	}
	fn, err := in.Eval(p.Car, env)
	if err != nil {
		return nil, err
	}
	args, err := in.evalArgs(p.Cdr, env)
	if err != nil {
		return nil, err
	}
	return in.applyFn(fn, args)
}

func (in *Interpreter) evalArgs(o Object, env *Env) ([]Object, error) {
	var args []Object
	cur := o
	for !isNil(cur) {
		pair, ok := cur.(*Pair)
		if !ok {
			return nil, fmt.Errorf("malformed argument list (dotted tail %s)", describe(cur))
		}
		v, err := in.Eval(pair.Car, env)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
		cur = pair.Cdr
	}
	return args, nil
}

func (in *Interpreter) applyFn(fn Object, args []Object) (Object, error) {
	switch f := fn.(type) {
	case *Builtin:
		return f.Fn(args)
	case *Closure:
		cenv := NewEnv(f.Env)
		if err := bindParams(f, args, cenv); err != nil {
			if f.Name != "" {
				return nil, fmt.Errorf("%s: %w", f.Name, err)
			}
			return nil, err
		}
		return in.evalSequence(f.Body, cenv)
	}
	return nil, fmt.Errorf("cannot call %s (not a procedure)", describe(fn))
}

// parseParams converts a lambda parameter spec into fixed names plus an
// optional variadic tail name.
func parseParams(spec Object) ([]Symbol, Symbol, error) {
	var names []Symbol
	for {
		switch s := spec.(type) {
		case NilType:
			return names, "", nil
		case Symbol:
			return names, s, nil
		case *Pair:
			sym, ok := s.Car.(Symbol)
			if !ok {
				return nil, "", fmt.Errorf("expected a parameter name, got %s", describe(s.Car))
			}
			names = append(names, sym)
			spec = s.Cdr
		default:
			return nil, "", fmt.Errorf("malformed parameter list (got %s)", describe(spec))
		}
	}
}

func bindParams(c *Closure, args []Object, env *Env) error {
	i := 0
	for ; i < len(c.Params); i++ {
		if i >= len(args) {
			return fmt.Errorf("expected at least %d argument(s), got %d", len(c.Params), len(args))
		}
		env.Define(string(c.Params[i]), args[i])
	}
	remaining := args[i:]
	if c.Rest != "" {
		env.Define(string(c.Rest), listFrom(remaining, Nil))
	} else if len(remaining) > 0 {
		return fmt.Errorf("expected %d argument(s), got %d", len(c.Params), len(args))
	}
	return nil
}
