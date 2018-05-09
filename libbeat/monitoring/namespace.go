package monitoring

var namespaces = map[string]*Namespace{}

// Namespace contains the name of the namespace and it's registry
type Namespace struct {
	name     string
	registry *Registry
}

func newNamespace(name string) *Namespace {
	n := &Namespace{
		name: name,
	}
	namespaces[name] = n
	return n
}

// GetNamespace gets the namespace with the given name.
// If the namespace does not exist yet, a new one is created.
func GetNamespace(name string) *Namespace {
	if n, ok := namespaces[name]; ok {
		return n
	}
	return newNamespace(name)
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
