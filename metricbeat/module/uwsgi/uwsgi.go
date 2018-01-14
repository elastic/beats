package uwsgi

import (
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// HostParser is used for parsing the configured uWSGI hosts.
var HostParser = parse.URLHostParserBuilder{DefaultScheme: "tcp"}.Build()

func init() {
	if err := mb.Registry.AddModule("uwsgi", NewModule); err != nil {
		panic(err)
	}
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}
