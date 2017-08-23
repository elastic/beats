package add_kubernetes_metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestLogsPathMatcher(t *testing.T) {
	var testConfig = common.NewConfig()

	logMatcher, err := newLogsPathMatcher(*testConfig)
	assert.Nil(t, err)

	cid := "0069869de9adf97f574c62029aeba65d1ecd85a2a112e87fbc28afe4dec2b843"
	logPath := fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", cid, cid)

	input := common.MapStr{
		"source": "/var/log/messages",
	}

	output := logMatcher.MetadataIndex(input)
	assert.Equal(t, output, "")

	input["source"] = logPath
	output = logMatcher.MetadataIndex(input)

	assert.Equal(t, output, cid)
}

func TestLogsPathMatcherVarLogContainers(t *testing.T) {
	var testConfig = common.NewConfig()
	testConfig.SetString("logs_path", -1, "/var/log/containers/")

	logMatcher, err := newLogsPathMatcher(*testConfig)
	assert.Nil(t, err)

	cid := "0069869de9adf97f574c62029aeba65d1ecd85a2a112e87fbc28afe4dec2b843"
	logPath := fmt.Sprintf("/var/log/containers/kube-proxy-4d7nt_kube-system_kube-proxy-%s.log", cid)

	input := common.MapStr{
		"source": logPath,
	}

	output := logMatcher.MetadataIndex(input)

	assert.Equal(t, output, cid)
}
