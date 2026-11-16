package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"

	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_docker_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
)

func TestGenerateProcessorList(t *testing.T) {
	inputCfg := []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{"add_kubernetes_metadata": nil},
	}

	plugins, err := processors.NewPluginConfigFromList(inputCfg)
	require.NoError(t, err)

	processors, err := processors.New(plugins, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	// make sure the processor init got the config formatted in a way it expected
	require.Len(t, processors.List, 4)

	require.NoError(t, processors.Close())
}
