package monitoring

import (
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Registry to store variables and sub-registries.
// When adding or retrieving variables, all names are split on the `.`-symbol and
// intermediate registries will be generated.
type Registry struct {
	mu sync.RWMutex

	name    string
	entries map[string]Var

	opts *options
}

// Var interface required for every metric to implement.
type Var interface {
	Visit(Visitor) error
}

// NewRegistry create a new empty unregistered registry
func NewRegistry(opts ...Option) *Registry {
	return &Registry{
		opts:    applyOpts(nil, opts),
		entries: map[string]Var{},
	}
}

func (r *Registry) Do(f func(string, interface{}) error) error {
	return r.Visit(NewKeyValueVisitor(f))
}

// Visit uses the Visitor interface to iterate the complete metrics hieararchie.
// In case of the visitor reporting an error, Visit will return immediately,
// reporting the very same error.
func (r *Registry) Visit(vs Visitor) error {
	if err := vs.OnRegistryStart(); err != nil {
		return err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	first := true
	for key, v := range r.entries {
		if first {
			first = false
		} else {
			if err := vs.OnKeyNext(); err != nil {
				return err
			}
		}

		if err := vs.OnKey(key); err != nil {
			return err
		}

		if err := v.Visit(vs); err != nil {
			return err
		}
	}

	return vs.OnRegistryFinished()
}

// NewRegistry creates and register a new registry
func (r *Registry) NewRegistry(name string, opts ...Option) *Registry {
	v := &Registry{
		name:    r.fullName(name),
		opts:    applyOpts(r.opts, opts),
		entries: map[string]Var{},
	}
	r.Add(name, v)
	return v
}

// NewInt creates and registers a new integer variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func (r *Registry) NewInt(name string) *Int {
	v := &Int{}
	r.Add(name, v)
	r.publish(name, makeExpvar(func() string {
		return strconv.FormatInt(v.Get(), 10)
	}))
	return v
}

// NewFloat creates and registers a new float variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func (r *Registry) NewFloat(name string) *Float {
	v := &Float{}
	r.Add(name, v)
	r.publish(name, makeExpvar(func() string {
		return strconv.FormatFloat(v.Get(), 'g', -1, 64)
	}))
	return v
}

// NewString creates and registers a new string variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func (r *Registry) NewString(name string) *String {
	v := &String{}
	r.Add(name, v)
	r.publish(name, makeExpvar(func() string {
		b, _ := json.Marshal(v.Get())
		return string(b)
	}))
	return v
}

// Get tries to find a registered variable by name.
func (r *Registry) Get(name string) interface{} {
	v, err := r.find(name)
	if err != nil {
		return nil
	}
	return v
}

// GetRegistry tries to find a sub-registry by name.
func (r *Registry) GetRegistry(name string) *Registry {
	v, err := r.find(name)
	if err != nil {
		return nil
	}

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

	r.entries = map[string]Var{}
	return nil
}

func (r *Registry) publish(name string, v expvar.Var) {
	if !r.opts.publishExpvar {
		return
	}

	expvar.Publish(r.fullName(name), v)
}

func (r *Registry) fullName(name string) string {
	if r.name == "" {
		return name
	}
	return r.name + "." + name
}

// Add adds a new variable to the registry. The method panics if the variables
// name is already in use.
func (r *Registry) Add(name string, v Var) {
	panicErr(r.addNames(strings.Split(name, "."), v))
}

func (r *Registry) addNames(names []string, v Var) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := names[0]
	if len(names) == 1 {
		if _, found := r.entries[name]; found {
			return fmt.Errorf("name %v already used", name)
		}

		r.entries[name] = v
		return nil
	}

	if tmp, found := r.entries[name]; found {
		reg, ok := tmp.(*Registry)
		if !ok {
			return fmt.Errorf("name %v already used", name)
		}

		return reg.addNames(names[1:], v)
	}

	sub := NewRegistry()
	sub.opts = r.opts
	if err := sub.addNames(names[1:], v); err != nil {
		return err
	}

	r.entries[name] = sub
	return nil
}

func (r *Registry) find(name string) (interface{}, error) {
	return r.findNames(strings.Split(name, "."))
}

func (r *Registry) findNames(names []string) (interface{}, error) {
	switch len(names) {
	case 0:
		return r, nil
	case 1:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.entries[names[0]], nil
	}

	r.mu.RLock()
	next := r.entries[names[0]]
	r.mu.RUnlock()

	if next == nil {
		return nil, errNotFound
	}

	if reg, ok := next.(*Registry); ok {
		return reg.findNames(names[1:])
	}
	return nil, errInvalidName
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
	next := r.entries[names[0]]
	sub, ok := next.(*Registry)

	// if name does not exist => don't remove anything
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
