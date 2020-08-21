package registries

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/go-concert/unison"
)

type registryList []v2.Registry

// Combine combines a list of input registries into a single registry.
// When configuring an input the each registry is tried. The first registry
// that returns an input type wins.
// registry in the list should have a type prefix to allow some routing.
//
// The registryList can be used to combine v2 style inputs and old RunnerFactory
// into a single namespace. By listing v2 style inputs first we can shadow older implementations
// without fully replacing them in the Beats code-base.
func Combine(registries ...v2.Registry) v2.Registry {
	return registryList(registries)
}

func (r registryList) Init(grp unison.Group, mode v2.Mode) error {
	for _, sub := range r {
		if err := sub.Init(grp, mode); err != nil {
			return err
		}
	}
	return nil
}

func (r registryList) Find(name string) (v2.Plugin, bool) {
	for _, sub := range r {
		if p, ok := sub.Find(name); ok {
			return p, true
		}
	}
	return v2.Plugin{}, false
}
