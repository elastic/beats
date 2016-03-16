package apache

import (
	"os"

	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	helper.Registry.AddModuler("apache", New)
}

// New creates new instance of Moduler
func New() helper.Moduler {
	return &Moduler{}
}

type Moduler struct{}

func (m *Moduler) Setup(mo *helper.Module) error {
	return nil
}

///*** Helper functions for testing ***///

func GetApacheEnvHost() string {
	host := os.Getenv("APACHE_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}
