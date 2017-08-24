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
	source := "/var/log/messages"
	expectedResult := ""
	executeTest(t, source, expectedResult);
}

func TestLogsPathMatcher_InvalidSource2(t *testing.T) {
	source := "/var/lib/docker/containers/01234567/89abcdef-json.log"
	expectedResult := ""
	executeTest(t, source, expectedResult);
}

func TestLogsPathMatcher_InvalidSource3(t *testing.T) {
	source := "/var/log/containers/pod_ns_container_01234567.log"
	expectedResult := ""
	executeTest(t, source, expectedResult);
}

func TestLogsPathMatcher_VarLibDockerContainers(t *testing.T) {
	source := fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", cid, cid)
	expectedResult := cid;
	executeTest(t, source, expectedResult);
}

func TestLogsPathMatcher_VarLogContainers(t *testing.T) {
	source := fmt.Sprintf("/var/log/containers/kube-proxy-4d7nt_kube-system_kube-proxy-%s.log", cid)
	expectedResult := cid;
	executeTest(t, source, expectedResult);
}

func TestLogsPathMatcher_GenericFallback(t *testing.T) {
	source := fmt.Sprintf("/var/log/foo/bar-%s-baz.log", cid)
	expectedResult := cid;
	executeTest(t, source, expectedResult);
}

func executeTest(t *testing.T, source string, expectedResult string) {
	var testConfig = common.NewConfig()
	logMatcher, err := newLogsPathMatcher(*testConfig)
	assert.Nil(t, err)

	input := common.MapStr{
		"source": source,
	}
	output := logMatcher.MetadataIndex(input)
	assert.Equal(t, output, expectedResult)
}
