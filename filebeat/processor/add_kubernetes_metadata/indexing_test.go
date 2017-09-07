package add_kubernetes_metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

// A random container ID that we use for our tests
const cid = "0069869de9adf97f574c62029aeba65d1ecd85a2a112e87fbc28afe4dec2b843"

func TestLogsPathMatcher_InvalidSource1(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	source := "/var/log/messages"
	expectedResult := ""
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_InvalidSource2(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	source := "/var/lib/docker/containers/01234567/89abcdef-json.log"
	expectedResult := ""
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_InvalidSource3(t *testing.T) {
	cfgLogsPath := "/var/log/containers/"
	source := "/var/log/containers/pod_ns_container_01234567.log"
	expectedResult := ""
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_VarLibDockerContainers(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	source := fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", cid, cid)
	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_VarLogContainers(t *testing.T) {
	cfgLogsPath := "/var/log/containers/"
	source := fmt.Sprintf("/var/log/containers/kube-proxy-4d7nt_kube-system_kube-proxy-%s.log", cid)
	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_AnotherLogDir(t *testing.T) {
	cfgLogsPath := "/var/log/other/"
	source := fmt.Sprintf("/var/log/other/%s.log", cid)
	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func executeTest(t *testing.T, cfgLogsPath string, source string, expectedResult string) {
	var testConfig = common.NewConfig()
	if cfgLogsPath != "" {
		testConfig.SetString("logs_path", -1, cfgLogsPath)
	}

	logMatcher, err := newLogsPathMatcher(*testConfig)
	assert.Nil(t, err)

	input := common.MapStr{
		"source": source,
	}
	output := logMatcher.MetadataIndex(input)
	assert.Equal(t, output, expectedResult)
}
