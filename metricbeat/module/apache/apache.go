package apache

import (
	"os"

	"github.com/elastic/beats/metricbeat/helper"
)

func init() {
	helper.Registry.AddModuler("apache", Moduler{})
}

type Moduler struct{}

func (r Moduler) Setup() error {
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
