package apache

import (
	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	if err := helper.Registry.AddModuler("apache", New); err != nil {
		panic(err)
	}
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m *Moduler) Setup(mo *helper.Module) error {
	return nil
}
