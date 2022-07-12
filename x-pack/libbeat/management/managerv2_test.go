package management

import (
	"testing"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/require"
)

func TestV2InputTranspile(t *testing.T) {
	var beatInput = `
id: system/metrics-system-default-system
type: system/metrics
data_stream.namespace: default
use_output: default
streams:
  - metricset: cpu
    data_stream.dataset: system.cpu
    cpu.metrics:
      - percentages
      - normalized_percentages
  - metricset: memory
    data_stream.dataset: system.memory
  - metricset: network
    data_stream.dataset: system.network
  - metricset: filesystem
    data_stream.dataset: system.filesystem
`
	confIn, err := conf.NewConfigWithYAML([]byte(beatInput), "test")
	require.NoError(t, err, "NewConfigWithYAML()")
	confMap := UnitInput{}
	err = confIn.Unpack(&confMap)
	require.NoError(t, err, "Unpack()")
	t.Logf("Config namespace: %#v", confMap)
	confMap.Init()
	// step 1
	err = confMap.InjectIndex("metrics")
	require.NoError(t, err, "InjectIndex")
	t.Logf("rendered: %#v", confMap.renderedCfg)
	// 2
}
