package azure

import (
	"github.com/elastic/beats/metricbeat/mb"
)

// Config options
type Config struct {
	ClientId    string `config:"client_id"    validate:"required"`
	ClientSecret     string `config:"client_secret"`
	TenantId string `config:"tenant_id" validate:"required"`
}

func init() {
	// Register the ModuleFactory function for the "azure" module.
	if err := mb.Registry.AddModule("azure", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// mssql module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}
	return &base, nil
}

// NewMetricSet creates a base metricset for default configurations optons and auth in the future
func GetConfig(base mb.BaseMetricSet) (Config, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

