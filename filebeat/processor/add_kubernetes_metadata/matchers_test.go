// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package add_kubernetes_metadata

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

// A random container ID that we use for our tests
const cid = "0069869de9adf97f574c62029aeba65d1ecd85a2a112e87fbc28afe4dec2b843"

// A random pod UID that we use for our tests
const puid = "005f3b90-4b9d-12f8-acf0-31020a840133"

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

	path := "/var/lib/docker/containers/%s/%s-json.log"
	if runtime.GOOS == "windows" {
		path = "C:\\ProgramData\\Docker\\containers\\%s\\%s-json.log"
	}

	source := fmt.Sprintf(path, cid, cid)

	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_VarLogContainers(t *testing.T) {
	cfgLogsPath := "/var/log/containers/"
	sourcePath := "/var/log/containers/kube-proxy-4d7nt_kube-system_kube-proxy-%s.log"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\containers\\"
		sourcePath = "C:\\var\\log\\containers\\kube-proxy-4d7nt_kube-system_kube-proxy-%s.log"
	}

	source := fmt.Sprintf(sourcePath, cid)
	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_AnotherLogDir(t *testing.T) {
	cfgLogsPath := "/var/log/other/"
	sourcePath := "/var/log/other/%s.log"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\othere\\"
		sourcePath = "C:\\var\\log\\othere\\%s.log"
	}

	source := fmt.Sprintf(sourcePath, cid)
	expectedResult := cid
	executeTest(t, cfgLogsPath, source, expectedResult)
}

func TestLogsPathMatcher_VarLibKubeletPods(t *testing.T) {
	cfgLogsPath := "/var/lib/kubelet/pods/"
	sourcePath := "/var/lib/kubelet/pods/%s/volumes/kubernetes.io~empty-dir/applogs/server.log"
	cfgResourceType := "pod"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\lib\\kubelet\\pods\\"
		sourcePath = "C:\\var\\lib\\kubelet\\pods\\%s\\volumes\\kubernetes.io~empty-dir\\applogs\\server.log"
	}

	source := fmt.Sprintf(sourcePath, puid)
	expectedResult := puid
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_InvalidSource4(t *testing.T) {
	cfgLogsPath := "/var/lib/kubelet/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/invalid/dir/%s/volumes/kubernetes.io~empty-dir/applogs/server.log", puid)
	expectedResult := ""
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_InvalidVarLogPodSource(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/invalid/dir/namespace_pod-name_%s/container/0.log", puid)
	expectedResult := ""
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_ValidVarLogPodSource(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log.20220221-210912"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log.20220221-210912"
	}
	source := fmt.Sprintf(sourcePath, puid)
	expectedResult := puid
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_InvalidVarLogPodSource2(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/var/log/pods/namespace_pod-name_%s/container/0.log.20220221-210526.gz", puid)
	expectedResult := ""
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_InvalidVarLogPodIDFormat(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/var/log/pods/%s/container/0.log", puid)
	expectedResult := ""
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func TestLogsPathMatcher_ValidVarLogPod(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log"
	}
	source := fmt.Sprintf(sourcePath, puid)
	expectedResult := puid
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, expectedResult)
}

func executeTest(t *testing.T, cfgLogsPath string, source string, expectedResult string) {
	executeTestWithResourceType(t, cfgLogsPath, "", source, expectedResult)
}

func executeTestWithResourceType(t *testing.T, cfgLogsPath string, cfgResourceType string, source string, expectedResult string) {
	var testConfig = common.NewConfig()
	if cfgLogsPath != "" {
		testConfig.SetString("logs_path", -1, cfgLogsPath)
	}

	if cfgResourceType != "" {
		testConfig.SetString("resource_type", -1, cfgResourceType)
	}

	logMatcher, err := newLogsPathMatcher(*testConfig)
	assert.NoError(t, err)

	input := common.MapStr{
		"log": common.MapStr{
			"file": common.MapStr{
				"path": source,
			},
		},
	}
	output := logMatcher.MetadataIndex(input)
	assert.Equal(t, expectedResult, output)
}
