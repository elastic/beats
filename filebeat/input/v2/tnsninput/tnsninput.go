// Package tnsninput implements the v2 Filebeat input API for transient inputs
// that keep all state in memory and don't need any additional features.
package tnsninput

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
)

// Loader creates inputs from a configuration by finding and calling the right
// plugin.
type Loader struct {
	typeField string
	reg       *Registry
}

// Registry is used to lookup available plugins.
type Registry v2.RegistryTree

// Plugin to be added to a registry. The Plugin will be used to create an
// Input.
type Plugin struct {
	Name       string
	Stability  feature.Stability
	Deprecated bool
	Info       string
	Doc        string
	Create     func(*common.Config) (v2.Input, error)
}

// Extension types can be Registry or Plugin. It is used to combine plugins and
// registry into an even bigger registry.
// for Example:
//    r1, _ := NewRegistry(...)
//    r2, _ := NewRegistry(...)
//    r, err := NewRegistry(r1, r2,
//        &Plugin{...},
//        &Plugin{...},
//    )
//    // -> r can be used to load plugins from all registries and plugins
//    //    added.
type Extension interface {
	addToRegistry(reg *Registry) error
}

var _ v2.Loader = (*Loader)(nil)
var _ v2.Plugin = (*Plugin)(nil)
var _ v2.Registry = (*Registry)(nil)
var _ Extension = (*Plugin)(nil)
var _ Extension = (*Registry)(nil)

// NewLoader creates a new Loader for a registry.
func NewLoader(typeField string, reg *Registry) *Loader {
	if typeField == "" {
		typeField = "type"
	}
	return &Loader{typeField, reg}
}

// Configure looks for a plugin matching the 'type' name in the configuration
// and creates a new Input.  Configure fails if the type is not known, or if
// the plugin can not apply the configuration.
func (l *Loader) Configure(cfg *common.Config) (v2.Input, error) {
	name, err := cfg.String(l.typeField, -1)
	if err != nil {
		return v2.Input{}, err
	}

	plugin, ok := l.reg.findPlugin(name)
	if !ok {
		return v2.Input{}, &v2.LoaderError{Name: name, Reason: v2.ErrUnknown}
	}

	return plugin.Create(cfg)
}

// NewRegistry creates a new Registry from the list of Registrations and Plugins.
func NewRegistry(exts ...Extension) (*Registry, error) {
	r := &Registry{}
	for _, ext := range exts {
		if err := r.Add(ext); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Add adds another registry or plugin to the current registry.
func (r *Registry) Add(ext Extension) error {
	return ext.addToRegistry(r)
}

func (r *Registry) addToRegistry(parent *Registry) error {
	return (*v2.RegistryTree)(parent).AddRegistry(r)
}

func (p *Plugin) addToRegistry(parent *Registry) error {
	return (*v2.RegistryTree)(parent).AddPlugin(p)
}

// Each iterates over all known plugins accessible using this registry.
// The iteration stops when fn return false.
func (r *Registry) Each(fn func(v2.Plugin) bool) {
	(*v2.RegistryTree)(r).Each(fn)
}

// Find returns a Plugin based on it's name.
func (r *Registry) Find(name string) (plugin v2.Plugin, ok bool) {
	return (*v2.RegistryTree)(r).Find(name)
}

func (r *Registry) findPlugin(name string) (*Plugin, bool) {
	p, ok := r.Find(name)
	if !ok {
		return nil, false
	}
	return p.(*Plugin), ok
}

// Details returns common feature information about the plugin the and input
// type it can generate.
func (p *Plugin) Details() feature.Details {
	return feature.Details{
		Name:       p.Name,
		Stability:  p.Stability,
		Deprecated: p.Deprecated,
		Info:       p.Info,
		Doc:        p.Doc,
	}
}
