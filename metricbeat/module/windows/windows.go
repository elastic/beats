// +build windows

package windows

import (
	"sync"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

var once sync.Once

func init() {
	// Register the ModuleFactory function for the "windows" module.
	if err := mb.Registry.AddModule("windows", NewModule); err != nil {
		panic(err)
	}
}

func initModule() {
	if err := helper.CheckAndEnableSeDebugPrivilege(); err != nil {
		logp.Warn("%v", err)
	}
}

type Module struct {
	mb.BaseModule
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	once.Do(func() {
		initModule()
	})

	return &Module{BaseModule: base}, nil
}
