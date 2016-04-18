package system

import (
	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	helper.Registry.AddModuler("system", New)
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m Moduler) Setup(md *helper.Module) error {
	return nil
}
