package monitoring

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Registry to store variables and sub-registries.
// When adding or retrieving variables, all names are split on the `.`-symbol and
// intermediate registries will be generated.
type Registry struct {
	mu sync.RWMutex

	name    string
	entries map[string]entry

	opts *options
}

type entry struct {
	Var
	Mode
}

// Var interface required for every metric to implement.
type Var interface {
	Visit(Mode, Visitor)
}

// NewRegistry create a new empty unregistered registry
func NewRegistry(opts ...Option) *Registry {
	return &Registry{
		opts:    applyOpts(nil, opts),
		entries: map[string]entry{},
	}
}

func (r *Registry) Do(mode Mode, f func(string, interface{})) {
	r.doVisit(mode, NewKeyValueVisitor(f))
}

// Visit uses the Visitor interface to iterate the complete metrics hieararchie.
// In case of the visitor reporting an error, Visit will return immediately,
// reporting the very same error.
func (r *Registry) Visit(mode Mode, vs Visitor) {
	r.doVisit(mode, vs)
}

func (r *Registry) doVisit(mode Mode, vs Visitor) {
	vs.OnRegistryStart()
	defer vs.OnRegistryFinished()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for key, v := range r.entries {
		if _, isReg := v.Var.(*Registry); !isReg {
			if v.Mode > mode {
				continue
			}
		}

		vs.OnKey(key)
		v.Var.Visit(mode, vs)
	}
}

// NewRegistry creates and register a new registry
func (r *Registry) NewRegistry(name string, opts ...Option) *Registry {
	v := &Registry{
		name:    fullName(r, name),
		opts:    applyOpts(r.opts, opts),
		entries: map[string]entry{},
	}
	r.Add(name, v, v.opts.mode)
	return v
}

// Get tries to find a registered variable by name.
func (r *Registry) Get(name string) Var {
	v, err := r.find(name)
	if err != nil {
		return nil
	}
	return v.Var
}

// GetRegistry tries to find a sub-registry by name.
func (r *Registry) GetRegistry(name string) *Registry {
	e, err := r.find(name)
	if err != nil {
		return nil
	}

	v := e.Var
	if v == nil {
		return nil
	}

	reg, ok := v.(*Registry)
	if !ok {
		return nil
	}

	return reg
}

// Remove removes a variable or a sub-registry by name
func (r *Registry) Remove(name string) {
	r.removeNames(strings.Split(name, "."))
}

// Clear removes all entries from the current registry
func (r *Registry) Clear() error {
	r.mu.Lock()
	r.mu.Unlock()

	if r.opts.publishExpvar {
		return errors.New("Can not clear registry with metrics being exported via expvar")
	}

	r.entries = map[string]entry{}
	return nil
}

// Add adds a new variable to the registry. The method panics if the variables
// name is already in use.
func (r *Registry) Add(name string, v Var, m Mode) {
	opts := r.opts
	if m != opts.mode {
		tmp := *r.opts
		tmp.mode = m
		opts = &tmp
	}

	panicErr(r.addNames(strings.Split(name, "."), v, opts))
}

func (r *Registry) doAdd(name string, v Var, opts *options) {
	panicErr(r.addNames(strings.Split(name, "."), v, opts))
}

func (r *Registry) addNames(names []string, v Var, opts *options) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := names[0]
	if len(names) == 1 {
		if _, found := r.entries[name]; found {
			return fmt.Errorf("name %v already used", name)
		}

		r.entries[name] = entry{v, opts.mode}
		return nil
	}

	if tmp, found := r.entries[name]; found {
		reg, ok := tmp.Var.(*Registry)
		if !ok {
			return fmt.Errorf("name %v already used", name)
		}

		return reg.addNames(names[1:], v, opts)
	}

	sub := NewRegistry()
	sub.opts = opts
	if err := sub.addNames(names[1:], v, opts); err != nil {
		return err
	}

	r.entries[name] = entry{sub, sub.opts.mode}
	return nil
}

func (r *Registry) find(name string) (entry, error) {
	return r.findNames(strings.Split(name, "."))
}

func (r *Registry) findNames(names []string) (entry, error) {
	switch len(names) {
	case 0:
		return entry{r, r.opts.mode}, nil
	case 1:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.entries[names[0]], nil
	}

	r.mu.RLock()
	next, exist := r.entries[names[0]]
	r.mu.RUnlock()

	if !exist {
		return entry{}, errNotFound
	}

	if reg, ok := next.Var.(*Registry); ok {
		return reg.findNames(names[1:])
	}
	return entry{}, errInvalidName
}

func (r *Registry) removeNames(names []string) {
	switch len(names) {
	case 0:
		return
	case 1:
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.entries, names[0])
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	next, exists := r.entries[names[0]]

	// if name does not exist => don't remove anything
	if !exists {
		return
	}

	sub, ok := next.Var.(*Registry)
	if ok {
		sub.removeNames(names[1:])
		sub.mu.RLock()
		sub.mu.RUnlock()

		if len(sub.entries) == 0 {
			delete(r.entries, names[0])
		}
	}
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
