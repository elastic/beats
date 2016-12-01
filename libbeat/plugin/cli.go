//+build linux,go1.8,cgo

package plugin

import (
	"flag"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

type pluginList struct {
	paths []string
}

func (p *pluginList) String() string {
	return strings.Join(p.paths, ",")
}

func (p *pluginList) Set(v string) error {
	// TODO: check file exists

	p.paths = append(p.paths, v)
	return nil
}

var plugins = &pluginList{}

func init() {
	flag.Var(plugins, "plugin", "Load additional plugins")
}

func Initialize() error {
	if len(plugins.paths) > 0 {
		logp.Warn("EXPERIMENTAL: loadable plugin support is experimental")
	}

	for _, path := range plugins.paths {
		logp.Info("loading plugin bundle: %v", path)

		if err := LoadPlugins(path); err != nil {
			return err
		}
	}

	return nil
}
