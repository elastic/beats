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
	"strings"

	"github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Initialize initializes all the options for the `add_kubernetes_metadata` process for filebeat.
//
// Must be called from the settings `InitFunc`.
func Initialize() {
	add_kubernetes_metadata.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)
	cfg := conf.NewConfig()

	// Add a container indexer config by default.
	add_kubernetes_metadata.Indexing.AddDefaultIndexerConfig(add_kubernetes_metadata.ContainerIndexerName, *cfg)

	// Add a log path matcher which can extract container ID from the "source" field.
	add_kubernetes_metadata.Indexing.AddDefaultMatcherConfig(LogPathMatcherName, *cfg)
}

const (
	LogPathMatcherName = "logs_path"
	pathSeparator      = string(os.PathSeparator)
)

type LogPathMatcher struct {
	LogsPath     string
	ResourceType string
	logger       *logp.Logger
}

func newLogsPathMatcher(cfg conf.C) (add_kubernetes_metadata.Matcher, error) {
	config := struct {
		LogsPath     string `config:"logs_path"`
		ResourceType string `config:"resource_type"`
	}{
		LogsPath:     defaultLogPath(),
		ResourceType: "container",
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return nil, fmt.Errorf("fail to unpack the `logs_path` configuration: %w", err)
	}

	logPath := config.LogsPath
	if logPath[len(logPath)-1:] != pathSeparator {
		logPath = logPath + pathSeparator
	}
	resourceType := config.ResourceType

	log := logp.NewLogger("kubernetes")
	log.Debugf("logs_path matcher log path: %s", logPath)
	log.Debugf("logs_path matcher resource type: %s", resourceType)

	return &LogPathMatcher{LogsPath: logPath, ResourceType: resourceType, logger: log}, nil
}

// Docker container ID is a 64-character-long hexadecimal string
const containerIdLen = 64

func (f *LogPathMatcher) MetadataIndex(event mapstr.M) string {
	value, err := event.GetValue("log.file.path")
	if err != nil {
		f.logger.Debugf("Error extracting log.file.path from the event: %s.", event)
		return ""
	}

	source := value.(string)
	f.logger.Debugf("Incoming log.file.path value: %s", source)

	if !strings.Contains(source, f.LogsPath) {
		f.logger.Debugf("log.file.path value does not contain matcher's logs_path '%s', skipping...", f.LogsPath)
		return ""
	}

	sourceLen := len(source)
	logsPathLen := len(f.LogsPath)

	if f.ResourceType == "pod" {
		// Pod resource type will extract only the pod UID, which offers less granularity of metadata when compared to the container ID
		if strings.Contains(source, ".log") && !strings.HasSuffix(source, ".gz") {
			// Specify a pod resource type when writing logs into manually mounted log volume,
			// those logs apper under under "/var/lib/kubelet/pods/<pod_id>/volumes/..."
			if strings.HasPrefix(f.LogsPath, podKubeletLogsPath()) {
				pathDirs := strings.Split(source, pathSeparator)
				podUIDPos := 5
				if len(pathDirs) > podUIDPos {
					podUID := strings.Split(source, pathSeparator)[podUIDPos]
					f.logger.Debugf("Using pod uid: %s", podUID)
					return podUID
				}
			}
			// In case of the Kubernetes log path "/var/log/pods/",
			// the pod ID will be extracted from the directory name,
			// file name example: "/var/log/pods/'<namespace>_<pod_name>_<pod_uid>'/container_name/0.log".
			if strings.HasPrefix(f.LogsPath, podLogsPath()) {
				pathDirs := strings.Split(source, pathSeparator)
				podUIDPos := 4
				if len(pathDirs) > podUIDPos {
					podUID := strings.Split(pathDirs[podUIDPos], "_")
					if len(podUID) > 2 {
						f.logger.Debugf("Using pod uid: %s", podUID[2])
						return podUID[2]
					}
				}
			}

			f.logger.Error("Error extracting pod uid - source value does not contains matcher's logs_path")
			return ""
		}
	} else {
		// In case of the Kubernetes log path "/var/log/containers/",
		// the container ID will be located right before the ".log" extension.
		// file name example: /var/log/containers/<pod_name>_<namespace>_<container_name>-<continer_id>.log
		if strings.HasPrefix(f.LogsPath, containerLogsPath()) && strings.HasSuffix(source, ".log") && sourceLen >= containerIdLen+4 {
			containerIDEnd := sourceLen - 4
			cid := source[containerIDEnd-containerIdLen : containerIDEnd]
			f.logger.Debugf("Using container id: %s", cid)
			return cid
		}

		// In any other case, we assume the container ID will follow right after the log path.
		// However we need to check the length to prevent "slice bound out of range" runtime errors.
		// for the default log path /var/lib/docker/containers/ container ID will follow right after the log path.
		// file name example: /var/lib/docker/containers/<container_id>/<container_id>-json.log
		if sourceLen >= logsPathLen+containerIdLen {
			cid := source[logsPathLen : logsPathLen+containerIdLen]
			f.logger.Debugf("Using container id: %s", cid)
			return cid
		}
	}
	f.logger.Error("Error extracting container id - source value contains matcher's logs_path, however it is too short to contain a Docker container ID.")
	return ""
}

func defaultLogPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\ProgramData\\Docker\\containers"
	}
	return "/var/lib/docker/containers/"
}

func podKubeletLogsPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\var\\lib\\kubelet\\pods\\"
	}
	return "/var/lib/kubelet/pods/"
}

func podLogsPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\var\\log\\pods\\"
	}
	return "/var/log/pods/"
}

func containerLogsPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\var\\log\\containers\\"
	}
	return "/var/log/containers/"
}
