package lisp

import "fmt"

// specialForms maps special-form keywords to their handlers. Each handler
// receives the (unevaluated) argument list and the current environment.
// Special forms cannot be redefined by user code. Initialized in init() to
// avoid a static initialization cycle (handlers call back into Eval).
var specialForms map[Symbol]func(in *Interpreter, args Object, env *Env) (Object, error)

func init() {
	specialForms = map[Symbol]func(in *Interpreter, args Object, env *Env) (Object, error){
		"quote":  specQuote,
		"define": specDefine,
		"set!":   specSet,
		"if":     specIf,
		"lambda": specLambda,
		"let":    specLet,
		"begin":  specBegin,
		"cond":   specCond,
		"and":    specAnd,
		"or":     specOr,
	}
}

// formArgs returns the argument list of a special form as a slice,
// requiring a proper list with at least min entries.
func formArgs(name string, args Object, min int) ([]Object, error) {
	items, ok := listSlice(args)
	if !ok {
		return nil, fmt.Errorf("%s: malformed special form (dotted argument list)", name)
	}
	if len(items) < min {
		return nil, fmt.Errorf("%s: expected at least %d argument(s), got %d", name, min, len(items))
	}
	return items, nil
}

func specQuote(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("quote", args, 1)
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("quote: expected exactly 1 argument, got %d", len(items))
	}
	return items[0], nil
}

func specDefine(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("define", args, 2)
	if err != nil {
		return nil, err
	}
	switch target := items[0].(type) {
	case Symbol:
		if len(items) != 2 {
			return nil, fmt.Errorf("define: expected exactly one value, got %d", len(items)-1)
		}
		v, err := in.Eval(items[1], env)
		if err != nil {
			return nil, err
		}
		env.Define(string(target), v)
		return v, nil
	case *Pair:
		name, ok := target.Car.(Symbol)
		if !ok {
			return nil, fmt.Errorf("define: expected a symbol as the function name, got %s", describe(target.Car))
		}
		params, rest, err := parseParams(target.Cdr)
		if err != nil {
			return nil, fmt.Errorf("define: %w", err)
		}
		cl := &Closure{Name: string(name), Params: params, Rest: rest, Body: items[1:], Env: env}
		env.Define(string(name), cl)
		return cl, nil
	default:
		return nil, fmt.Errorf("define: expected a symbol or (name params...), got %s", describe(items[0]))
	}
}

func specSet(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("set!", args, 2)
	if err != nil {
		return nil, err
	}
	if len(items) != 2 {
		return nil, fmt.Errorf("set!: expected exactly 2 arguments, got %d", len(items))
	}
	name, ok := items[0].(Symbol)
	if !ok {
		return nil, fmt.Errorf("set!: expected a symbol, got %s", describe(items[0]))
	}
	v, err := in.Eval(items[1], env)
	if err != nil {
		return nil, err
	}
	if !env.Set(string(name), v) {
		return nil, fmt.Errorf("set!: unbound symbol: %s", name)
	}
	return v, nil
}

func specIf(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("if", args, 2)
	if err != nil {
		return nil, err
	}
	if len(items) > 3 {
		return nil, fmt.Errorf("if: expected at most 3 arguments, got %d", len(items))
	}
	test, err := in.Eval(items[0], env)
	if err != nil {
		return nil, err
	}
	if truthy(test) {
		return in.Eval(items[1], env)
	}
	if len(items) == 3 {
		return in.Eval(items[2], env)
	}
	return Nil, nil
}

func specLambda(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("lambda", args, 2)
	if err != nil {
		return nil, err
	}
	params, rest, err := parseParams(items[0])
	if err != nil {
		return nil, fmt.Errorf("lambda: %w", err)
	}
	return &Closure{Params: params, Rest: rest, Body: items[1:], Env: env}, nil
}

func specLet(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("let", args, 1)
	if err != nil {
		return nil, err
	}
	bindings, ok := listSlice(items[0])
	if !ok {
		return nil, fmt.Errorf("let: malformed binding list")
	}
	inner := NewEnv(env)
	for _, b := range bindings {
		pair, ok := listSlice(b)
		if !ok || len(pair) != 2 {
			return nil, fmt.Errorf("let: expected a binding of the form (name value), got %s", describe(b))
		}
		name, ok := pair[0].(Symbol)
		if !ok {
			return nil, fmt.Errorf("let: expected a symbol in a binding, got %s", describe(pair[0]))
		}
		// Values are evaluated in the OUTER environment.
		v, err := in.Eval(pair[1], env)
		if err != nil {
			return nil, err
		}
		inner.Define(string(name), v)
	}
	return in.evalSequence(items[1:], inner)
}

func specBegin(in *Interpreter, args Object, env *Env) (Object, error) {
	items, err := formArgs("begin", args, 0)
	if err != nil {
		return nil, err
	}
	return in.evalSequence(items, env)
}

func specCond(in *Interpreter, args Object, env *Env) (Object, error) {
	clauses, ok := listSlice(args)
	if !ok {
		return nil, fmt.Errorf("cond: malformed clause list")
	}
	for _, clause := range clauses {
		parts, ok := listSlice(clause)
		if !ok || len(parts) == 0 {
			return nil, fmt.Errorf("cond: malformed clause %s", describe(clause))
		}
		if sym, isSym := parts[0].(Symbol); isSym && sym == "else" {
			return in.evalSequence(parts[1:], env)
		}
		test, err := in.Eval(parts[0], env)
		if err != nil {
			return nil, err
		}
		if truthy(test) {
			if len(parts) == 1 {
				return test, nil
			}
			return in.evalSequence(parts[1:], env)
		}
	}
	return Nil, nil
}

func specAnd(in *Interpreter, args Object, env *Env) (Object, error) {
	items, ok := listSlice(args)
	if !ok {
		return nil, fmt.Errorf("and: malformed argument list")
	}
	result := Object(Boolean(true))
	for _, item := range items {
		v, err := in.Eval(item, env)
		if err != nil {
			return nil, err
		}
		result = v
		if !truthy(v) {
			return Boolean(false), nil
		}
	}
	return result, nil
}

func specOr(in *Interpreter, args Object, env *Env) (Object, error) {
	items, ok := listSlice(args)
	if !ok {
		return nil, fmt.Errorf("or: malformed argument list")
	}
	for _, item := range items {
		v, err := in.Eval(item, env)
		if err != nil {
			return nil, err
		}
		if truthy(v) {
			return v, nil
		}
	}
	return Boolean(false), nil
}
