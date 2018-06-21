package feature

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

type mapper map[string]map[string]Featurable

// Registry implements a global registry for any kind of feature in beats.
// feature are grouped by namespace, a namespace is a kind of plugin like outputs, inputs, or queue.
// The feature name must be unique.
type registry struct {
	sync.RWMutex
	namespaces mapper
	log        *logp.Logger
}

// NewRegistry returns a new registry.
func newRegistry() *registry {
	return &registry{
		namespaces: make(mapper),
		log:        logp.NewLogger("registry"),
	}
}

// Register registers a new feature into a specific namespace, namespace are lazy created.
// Feature name must be unique.
func (r *registry) Register(feature Featurable) error {
	r.Lock()
	defer r.Unlock()

	// Lazy create namespaces
	_, found := r.namespaces[feature.Namespace()]
	if !found {
		r.namespaces[feature.Namespace()] = make(map[string]Featurable)
	}

	f, found := r.namespaces[feature.Namespace()][feature.Name()]
	if found {
		if feature.Equal(f) {
			// Allow both old style and new style of plugin to work together.
			r.log.Debugw(
				"ignoring, feature '%s' is already registered in the namespace '%s'",
				feature.Name(),
				feature.Namespace(),
			)
			return nil
		}

		return fmt.Errorf(
			"could not register new feature '%s' in namespace '%s', feature name must be unique",
			feature.Name(),
			feature.Namespace(),
		)
	}

	r.log.Debugw(
		"registering new feature",
		"namespace",
		feature.Namespace(),
		"name",
		feature.Name(),
	)

	r.namespaces[feature.Namespace()][feature.Name()] = feature

	return nil
}

// Unregister removes a feature from the registry.
func (r *registry) Unregister(namespace, name string) error {
	r.Lock()
	defer r.Unlock()

	v, found := r.namespaces[namespace]
	if !found {
		return fmt.Errorf("unknown namespace named '%s'", namespace)
	}

	_, found = v[name]
	if !found {
		return fmt.Errorf("unknown feature '%s' in namespace '%s'", name, namespace)
	}

	delete(r.namespaces[namespace], name)
	return nil
}

// Find returns a specific Find from a namespace or an error if not found.
func (r *registry) Find(namespace, name string) (Featurable, error) {
	r.RLock()
	defer r.RUnlock()

	v, found := r.namespaces[namespace]
	if !found {
		return nil, fmt.Errorf("unknown namespace named '%s'", namespace)
	}

	m, found := v[name]
	if !found {
		return nil, fmt.Errorf("unknown feature '%s' in namespace '%s'", name, namespace)
	}

	return m, nil
}

// FindAll returns all the features for a specific namespace.
func (r *registry) FindAll(namespace string) ([]Featurable, error) {
	r.RLock()
	defer r.RUnlock()

	v, found := r.namespaces[namespace]
	if !found {
		return nil, fmt.Errorf("unknown namespace named '%s'", namespace)
	}

	list := make([]Featurable, len(v))
	c := 0
	for _, feature := range v {
		list[c] = feature
		c++
	}

	return list, nil
}

// Size returns the number of registered features in the registry.
func (r *registry) Size() int {
	r.RLock()
	defer r.RUnlock()

	c := 0
	for _, namespace := range r.namespaces {
		c += len(namespace)
	}

	return c
}
