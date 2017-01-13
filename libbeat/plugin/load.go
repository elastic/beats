//+build linux,go1.8,cgo

package plugin

import (
	"errors"
	goplugin "plugin"
)

func loadPlugins(path string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}

	sym, err := p.Lookup("Bundle")
	if err != nil {
		return err
	}

	ptr, ok := sym.(*map[string][]interface{})
	if !ok {
		return errors.New("invalid bundle type")
	}

	bundle := *ptr
	for name, plugins := range bundle {
		loader := registry[name]
		if loader == nil {
			continue
		}

		for _, plugin := range plugins {
			if err := loader(plugin); err != nil {
				return err
			}
		}
	}

	return nil
}
