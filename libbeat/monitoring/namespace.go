package monitoring

import (
	"sync"
)

var namespaces = struct {
	sync.Mutex
	m map[string]*Namespace
}{
	m: make(map[string]*Namespace),
}

// Namespace contains the name of the namespace and it's registry
type Namespace struct {
	name     string
	registry *Registry
}

// GetNamespace gets the namespace with the given name.
// If the namespace does not exist yet, a new one is created.
func GetNamespace(name string) *Namespace {
	namespaces.Lock()
	defer namespaces.Unlock()

	n, ok := namespaces.m[name]
	if !ok {
		n = &Namespace{name: name}
		namespaces.m[name] = n
	}
	return n
}

// SetRegistry sets the registry of the namespace
func (n *Namespace) SetRegistry(r *Registry) {
	n.registry = r
}

// GetRegistry gets the registry of the namespace
func (n *Namespace) GetRegistry() *Registry {
	if n.registry == nil {
		n.registry = NewRegistry()
	}
	return n.registry
}
