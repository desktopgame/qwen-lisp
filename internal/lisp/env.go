package lisp

// Env is a lexical environment: a set of bindings plus a parent link.
type Env struct {
	parent *Env
	vars   map[string]Object
}

// NewEnv creates an environment nested inside parent (nil for global).
func NewEnv(parent *Env) *Env {
	return &Env{parent: parent, vars: make(map[string]Object)}
}

// Define binds name in this environment, shadowing outer bindings.
func (e *Env) Define(name string, val Object) {
	e.vars[name] = val
}

// Get looks name up in this environment and then in its parents.
func (e *Env) Get(name string) (Object, bool) {
	for cur := e; cur != nil; cur = cur.parent {
		if v, ok := cur.vars[name]; ok {
			return v, true
		}
	}
	return nil, false
}

// Set assigns to the innermost existing binding of name.
// It reports whether such a binding was found.
func (e *Env) Set(name string, val Object) bool {
	for cur := e; cur != nil; cur = cur.parent {
		if _, ok := cur.vars[name]; ok {
			cur.vars[name] = val
			return true
		}
	}
	return false
}
