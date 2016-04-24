// +build integration

package stubstatus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/module/nginx"
)

func TestConnect(t *testing.T) {

	config, _ := getNginxModuleConfig()

	module, mErr := helper.NewModule(config, nginx.New)
	assert.NoError(t, mErr)
	ms, msErr := helper.NewMetricSet("stubstatus", New, module)
	assert.NoError(t, msErr)

	// Setup metricset and metricseter
	err := ms.Setup()
	assert.NoError(t, err)
	err = ms.MetricSeter.Setup(ms)
	assert.NoError(t, err)

	// Check that host is correctly set
	assert.Equal(t, nginx.GetNginxEnvHost(), ms.Config.Hosts[0])

	data, err := ms.MetricSeter.Fetch(ms, ms.Config.Hosts[0])
	assert.NoError(t, err)

	// Check fields
	assert.Equal(t, 10, len(data))
}

type NginxModuleConfig struct {
	Hosts  []string `config:"hosts"`
	Module string   `config:"module"`
}

func getNginxModuleConfig() (*common.Config, error) {
	return common.NewConfigFrom(NginxModuleConfig{
		Module: "nginx",
		Hosts:  []string{nginx.GetNginxEnvHost()},
	})
}
