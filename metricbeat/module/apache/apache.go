package apache

import (
	//"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/helper"

	"os"
)

func init() {
	Module.Register()
}

var Module = helper.NewModule("apache", Apache{})

var Config = &ApacheModuleConfig{}

type ApacheModuleConfig struct {
	Metrics map[string]interface{}
	Hosts   []string
}

type Apache struct {
	Name   string
	Config ApacheModuleConfig
}

func (r Apache) Setup() error {

	// Loads module config
	// This is module specific config object
	Module.LoadConfig(&Config)
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
