package helper

import (
	"github.com/elastic/beats/libbeat/beat"
)

// Global register for modules and metrics
// Each module name must be unique
// Each module-metric name combination must be unique

// TODO: Global variables should be prevent.
// 	This should be moved into the metricbeat object but can't because the init()
//	functions in each metricset are called before the beater object exists.
var Registry = Register{}

type Register map[string]*Module

// StartModules starts all configured modules
func StartModules(b *beat.Beat) {
	for _, module := range Registry {
		go module.Start(b)
	}
}

// AddModule registers the given module with the registry
func (r Register) AddModule(m *Module) {
	r[m.Name] = m
}
