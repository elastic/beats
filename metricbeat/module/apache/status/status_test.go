// +build integration

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urso/ucfg"

	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/apache"
)

func TestConnect(t *testing.T) {

	config, _ := getApacheModuleConfig()

	module, mErr := helper.NewModule(config, apache.New)
	assert.NoError(t, mErr)
	ms, msErr := helper.NewMetricSet("status", New, module)
	assert.NoError(t, msErr)

	// Setup metricset and metricseter
	err := ms.Setup()
	assert.NoError(t, err)
	err = ms.MetricSeter.Setup(ms)
	assert.NoError(t, err)

	// Check that host is correctly set
	assert.Equal(t, apache.GetApacheEnvHost(), ms.Config.Hosts[0])

	data, err := ms.MetricSeter.Fetch(ms, ms.Config.Hosts[0])
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 13, len(data))
}

type ApacheModuleConfig struct {
	Hosts  []string `config:"hosts"`
	Module string   `config:"module"`
}

func getApacheModuleConfig() (*ucfg.Config, error) {
	return ucfg.NewFrom(ApacheModuleConfig{
		Module: "apache",
		Hosts:  []string{apache.GetApacheEnvHost()},
	})
}
