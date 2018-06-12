package registry

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
)

// Goal:
// - Unify plugin handling
// - Keep type safety
// - Allow external plugin
// - different type of plugin:
//		- processors
//    - queue
//    - autodiscover provider
//    - component:
//					- default logging configuration
// 					- enable secomp check
// 					- enable autodiscover
// 					- enabled metrics
// 					- ML
// 					- Kibana
// 					- keystore
// - reduce the global variable
// - allow scopped logger
// - threadsafety?
// - Add logger

type order int

// Control insert ordering.
const (
	Before order = -1
	After  order = 1
)

// PluginTypeKey represents a unique kind of plugins and will acts as a namespace.
// (example: Processors, autodiscover provider)
type PluginTypeKey string

// PluginID is a unique human readable identifier for a plugin.
type PluginID string

type pluginType struct {
	builder BuilderFunc
	plugins *orderedSet
}

func newPluginType(builder BuilderFunc) *pluginType {
	return &pluginType{builder: builder, plugins: newOrderedSet()}
}

// Registry keeps tracks of all the registred plugins in libbeat and provides way to access them.
type Registry struct {
	types map[PluginTypeKey]*pluginType
	log   *logp.Logger
}

// BuilderFunc methods user to builder the plugin.
type BuilderFunc func(interface{}) (interface{}, error)

// Plugin is the actual plugin.
type Plugin interface{}

// New creates a new plugins registry that will contains all the possible libbeat plugins.
func New(log *logp.Logger) *Registry {
	return &Registry{types: make(map[PluginTypeKey]*pluginType), log: log}
}

// MustRegisterType register a new plugin type in the registry and will raises a panic on failure.
func (r *Registry) MustRegisterType(t PluginTypeKey, builder BuilderFunc) {
	err := r.RegisterType(t, builder)
	if err != nil {
		panic("could not register plugin")
	}
}

// RegisterType register a new plugin type in the registry.
func (r *Registry) RegisterType(t PluginTypeKey, builder BuilderFunc) error {
	_, found := r.types[t]
	if found {
		return fmt.Errorf("could not add plugin type %s because it already exist in the registry", t)
	}
	r.types[t] = newPluginType(builder)
	return nil
}

// RemoveType removes a plugin type and all the registered plugins.
func (r *Registry) RemoveType(t PluginTypeKey) error {
	_, found := r.types[t]
	if !found {
		return fmt.Errorf("could not remove plugin type %s because it doesn't exist in the registry", t)
	}
	delete(r.types, t)
	return nil
}

// MustRegisterPlugin adds a new plugin for an existing plugin type and will panic on failure.
func (r *Registry) MustRegisterPlugin(t PluginTypeKey, id PluginID, plugin Plugin) {
	err := r.RegisterPlugin(t, id, plugin)
	if err != nil {
		panic("could not registry plugin")
	}
}

// RegisterPlugin a new plugin for an existing plugin type.
func (r *Registry) RegisterPlugin(t PluginTypeKey, id PluginID, plugin Plugin) error {
	pluginType, err := r.pluginType(t)
	if err != nil {
		return err
	}

	return pluginType.plugins.add(id, plugin)
}

// RemovePlugin removes an existing plugin
func (r *Registry) RemovePlugin(t PluginTypeKey, id PluginID) error {
	pluginType, err := r.pluginType(t)
	if err != nil {
		return err
	}
	return pluginType.plugins.remove(id)
}

func (r *Registry) pluginType(t PluginTypeKey) (*pluginType, error) {
	pluginType, found := r.types[t]
	if !found {
		return nil, fmt.Errorf("could not find plugin type '%s'", t)
	}
	return pluginType, nil
}

// Plugin returns the builder and the plugin.
func (r Registry) Plugin(t PluginTypeKey, name PluginID) (BuilderFunc, Plugin, error) {
	pluginType, err := r.pluginType(t)
	if err != nil {
		return nil, nil, err
	}

	plugin, err := pluginType.plugins.get(name)
	if err != nil {
		return nil, nil, err
	}

	return pluginType.builder, plugin, nil
}

// Plugins returns a list of ordered plugin
func (r *Registry) Plugins(t PluginTypeKey) (BuilderFunc, []Plugin, error) {
	pluginType, err := r.pluginType(t)
	if err != nil {
		return nil, nil, err
	}

	return pluginType.builder, pluginType.plugins.list(), nil
}

// OrderedRegisterPlugin allows to register at a relative position from another plugin.
func (r *Registry) OrderedRegisterPlugin(
	t PluginTypeKey,
	o order,
	targetID, name PluginID,
	plugin Plugin,
) error {
	pluginType, err := r.pluginType(t)
	if err != nil {
		return err
	}

	return pluginType.plugins.insert(o, targetID, name, plugin)
}
