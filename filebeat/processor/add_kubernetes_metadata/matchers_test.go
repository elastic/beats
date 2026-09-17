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
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// A random container ID that we use for our tests
const cid = "0069869de9adf97f574c62029aeba65d1ecd85a2a112e87fbc28afe4dec2b843"

// A random pod UID that we use for our tests
const puid = "005f3b90-4b9d-12f8-acf0-31020a840133"

func TestMain(m *testing.M) {
	InitializeModule()

	os.Exit(m.Run())
}

func TestLogsPathMatcher_InvalidSource1(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	source := "/var/log/messages"
	executeTest(t, cfgLogsPath, source, nil)
}

func TestLogsPathMatcher_InvalidSource2(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration
	source := "/var/lib/docker/containers/01234567/89abcdef-json.log"
	executeTest(t, cfgLogsPath, source, nil)
}

func TestLogsPathMatcher_InvalidSource3(t *testing.T) {
	cfgLogsPath := "/var/log/containers/"
	source := "/var/log/containers/pod_ns_container_01234567.log"
	executeTest(t, cfgLogsPath, source, nil)
}

func TestLogsPathMatcher_VarLibDockerContainers(t *testing.T) {
	cfgLogsPath := "" // use the default matcher configuration

	path := "/var/lib/docker/containers/%s/%s-json.log"
	if runtime.GOOS == "windows" {
		path = "C:\\ProgramData\\Docker\\containers\\%s\\%s-json.log"
	}

	source := fmt.Sprintf(path, cid, cid)
	executeTest(t, cfgLogsPath, source, []string{cid})
}

func TestLogsPathMatcher_VarLogContainers(t *testing.T) {
	cfgLogsPath := "/var/log/containers/"
	sourcePath := "/var/log/containers/kube-proxy-4d7nt_kube-system_kube-proxy-%s.log"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\containers\\"
		sourcePath = "C:\\var\\log\\containers\\kube-proxy-4d7nt_kube-system_kube-proxy-%s.log"
	}

	source := fmt.Sprintf(sourcePath, cid)
	executeTest(t, cfgLogsPath, source, []string{cid})
}

func TestLogsPathMatcher_AnotherLogDir(t *testing.T) {
	cfgLogsPath := "/var/log/other/"
	sourcePath := "/var/log/other/%s.log"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\othere\\"
		sourcePath = "C:\\var\\log\\othere\\%s.log"
	}

	source := fmt.Sprintf(sourcePath, cid)
	executeTest(t, cfgLogsPath, source, []string{cid})
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
	// kubelet path: no container segment — returns bare UID only
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, []string{puid})
}

func TestLogsPathMatcher_InvalidSource4(t *testing.T) {
	cfgLogsPath := "/var/lib/kubelet/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/invalid/dir/%s/volumes/kubernetes.io~empty-dir/applogs/server.log", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

func TestLogsPathMatcher_InvalidVarLogPodSource(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/invalid/dir/namespace_pod-name_%s/container/0.log", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

// ValidVarLogPodSource uses a rotated log filename with a timestamp suffix; the
// basename is not a plain integer so only the <uid>/<container> and <uid> candidates
// are returned.
func TestLogsPathMatcher_ValidVarLogPodSource(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log.20220221-210912"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log.20220221-210912"
	}
	source := fmt.Sprintf(sourcePath, puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/container", puid})
}

// ValidVarLogPodSource_GzRotated verifies that a .gz-compressed rotated log (with a timestamp
// suffix like 0.log.20220221-210526.gz) returns uid/container and uid candidates. After
// stripping .gz, the basename is "0.log.20220221-210526" which does not end in ".log", so no
// restart-count candidate is produced.
func TestLogsPathMatcher_ValidVarLogPodSource_GzRotated(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log.20220221-210526.gz"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log.20220221-210526.gz"
	}
	source := fmt.Sprintf(sourcePath, puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/container", puid})
}

// ValidVarLogPodSource_GzDirect verifies that a directly-compressed log (0.log.gz) is treated
// as restart 0 and returns all three candidates, same as its uncompressed counterpart 0.log.
func TestLogsPathMatcher_ValidVarLogPodSource_GzDirect(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log.gz"
	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log.gz"
	}
	source := fmt.Sprintf(sourcePath, puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/container/0", puid + "/container", puid})
}

func TestLogsPathMatcher_InvalidVarLogPodIDFormat(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	// Pod dir has no underscores — UID extraction fails
	source := fmt.Sprintf("/var/log/pods/%s/container/0.log", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

// ValidVarLogPod is the primary success case: a plain <n>.log basename yields the
// three-candidate list: <uid>/<container>/<n>, <uid>/<container>, <uid>.
func TestLogsPathMatcher_ValidVarLogPod(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/container/0.log"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\container\\0.log"
	}
	source := fmt.Sprintf(sourcePath, puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/container/0", puid + "/container", puid})
}

// ValidVarLogPod_Restart verifies that a non-zero restart count log file (e.g. 3.log) also
// produces the correct three-candidate list.
func TestLogsPathMatcher_ValidVarLogPod_Restart(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	sourcePath := "/var/log/pods/namespace_pod-name_%s/mycontainer/3.log"

	if runtime.GOOS == "windows" {
		cfgLogsPath = "C:\\var\\log\\pods\\"
		sourcePath = "C:\\var\\log\\pods\\namespace_pod-name_%s\\mycontainer\\3.log"
	}
	source := fmt.Sprintf(sourcePath, puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/mycontainer/3", puid + "/mycontainer", puid})
}

// ValidVarLogPod_NoLogExtension covers paths that match /var/log/pods/ but do not contain
// ".log" in their source — they are rejected and return nil.
func TestLogsPathMatcher_ValidVarLogPod_NoLogExtension(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	source := fmt.Sprintf("/var/log/pods/namespace_pod-name_%s/container/0", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

// LogsPathNotAPrefix verifies that a source where logs_path appears as an interior substring
// (not a prefix) is rejected. Previously, strings.Contains was used, which would have accepted
// /mnt/host/var/log/pods/… and parsed it at the wrong segment offsets.
func TestLogsPathMatcher_LogsPathNotAPrefix(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	// logs_path is a substring (interior), not a prefix of the source.
	source := fmt.Sprintf("/mnt/host/var/log/pods/namespace_pod-name_%s/container/0.log", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

// DigitDirectoryNotRestartCount verifies that an intermediate directory named with a digit
// is not treated as a restart-count filename. Only a basename that ends in ".log" and whose
// stem is a pure integer is accepted as a restart count.
func TestLogsPathMatcher_DigitDirectoryNotRestartCount(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	// pathDirs[6] = "3" (a directory name, not a "<n>.log" filename).
	// Must fall back to uid/container only — no restart-count candidate.
	source := fmt.Sprintf("/var/log/pods/namespace_pod-name_%s/container/3/real.log", puid)
	// This path has an extra segment; source HasPrefix check still passes for the pod dir,
	// but pathDirs[podUIDPos+2]="3" is not "3.log", so no restart candidate is produced.
	// The outer .log guard passes (real.log), so we expect uid/container + uid fallbacks.
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source,
		[]string{puid + "/container", puid})
}

// DotLogInNamespace verifies that ".log" appearing inside a directory segment
// (e.g. a namespace named "corp.logging") does not cause non-log files to be
// enriched. The outer guard must scope the ".log" check to the basename only.
func TestLogsPathMatcher_DotLogInNamespace_NonLogFile(t *testing.T) {
	cfgLogsPath := "/var/log/pods/"
	cfgResourceType := "pod"
	// Namespace "corp.logging" contains ".log" in the directory name.
	// The basename "datafile" has no ".log" — must return nil.
	source := fmt.Sprintf("/var/log/pods/corp.logging_pod-name_%s/container/datafile", puid)
	executeTestWithResourceType(t, cfgLogsPath, cfgResourceType, source, nil)
}

func executeTest(t *testing.T, cfgLogsPath string, source string, expectedResult []string) {
	executeTestWithResourceType(t, cfgLogsPath, "", source, expectedResult)
}

func executeTestWithResourceType(t *testing.T, cfgLogsPath string, cfgResourceType string, source string, expectedResult []string) {
	testConfig := conf.NewConfig()
	if cfgLogsPath != "" {
		testConfig.SetString("logs_path", -1, cfgLogsPath)
	}

	if cfgResourceType != "" {
		testConfig.SetString("resource_type", -1, cfgResourceType)
	}

	logMatcher, err := newLogsPathMatcher(*testConfig, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)

	input := mapstr.M{
		"log": mapstr.M{
			"file": mapstr.M{
				"path": source,
			},
		},
	}
	output := logMatcher.MetadataIndexCandidates(input)
	assert.Equal(t, expectedResult, output)
}
