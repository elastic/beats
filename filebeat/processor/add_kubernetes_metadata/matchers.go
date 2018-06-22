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
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

func init() {
	add_kubernetes_metadata.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)
	cfg := common.NewConfig()

	//Add a container indexer config by default.
	add_kubernetes_metadata.Indexing.AddDefaultIndexerConfig(add_kubernetes_metadata.ContainerIndexerName, *cfg)

	//Add a log path matcher which can extract container ID from the "source" field.
	add_kubernetes_metadata.Indexing.AddDefaultMatcherConfig(LogPathMatcherName, *cfg)
}

const LogPathMatcherName = "logs_path"

type LogPathMatcher struct {
	LogsPath     string
	ResourceType string
}

func newLogsPathMatcher(cfg common.Config) (add_kubernetes_metadata.Matcher, error) {
	config := struct {
		LogsPath     string `config:"logs_path"`
		ResourceType string `config:"resource_type"`
	}{
		LogsPath:     "/var/lib/docker/containers/",
		ResourceType: "container",
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return nil, fmt.Errorf("fail to unpack the `logs_path` configuration: %s", err)
	}

	logPath := config.LogsPath
	if logPath[len(logPath)-1:] != "/" {
		logPath = logPath + "/"
	}
	resourceType := config.ResourceType

	logp.Debug("kubernetes", "logs_path matcher log path: %s", logPath)
	logp.Debug("kubernetes", "logs_path matcher resource type: %s", resourceType)

	return &LogPathMatcher{LogsPath: logPath, ResourceType: resourceType}, nil
}

// Docker container ID is a 64-character-long hexadecimal string
const containerIdLen = 64

// Pod UID is on the 5th index of the path directories
const podUIDPos = 5

func (f *LogPathMatcher) MetadataIndex(event common.MapStr) string {
	if value, ok := event["source"]; ok {
		source := value.(string)
		logp.Debug("kubernetes", "Incoming source value: %s", source)

		if !strings.Contains(source, f.LogsPath) {
			logp.Debug("kubernetes", "Error extracting container id - source value does not contain matcher's logs_path '%s'.", f.LogsPath)
			return ""
		}

		sourceLen := len(source)
		logsPathLen := len(f.LogsPath)

		if f.ResourceType == "pod" {
			// Specify a pod resource type when manually mounting log volumes and they end up under "/var/lib/kubelet/pods/"
			// This will extract only the pod UID, which offers less granularity of metadata when compared to the container ID
			if strings.HasPrefix(f.LogsPath, "/var/lib/kubelet/pods/") && strings.HasSuffix(source, ".log") {
				pathDirs := strings.Split(source, "/")
				if len(pathDirs) > podUIDPos {
					podUID := strings.Split(source, "/")[podUIDPos]

					logp.Debug("kubernetes", "Using pod uid: %s", podUID)
					return podUID
				}

				logp.Debug("kubernetes", "Error extracting pod uid - source value contains matcher's logs_path, however it is too short to contain a Pod UID.")
			}
		} else {
			// In case of the Kubernetes log path "/var/log/containers/",
			// the container ID will be located right before the ".log" extension.
			if strings.HasPrefix(f.LogsPath, "/var/log/containers/") && strings.HasSuffix(source, ".log") && sourceLen >= containerIdLen+4 {
				containerIDEnd := sourceLen - 4
				cid := source[containerIDEnd-containerIdLen : containerIDEnd]
				logp.Debug("kubernetes", "Using container id: %s", cid)
				return cid
			}

			// In any other case, we assume the container ID will follow right after the log path.
			// However we need to check the length to prevent "slice bound out of range" runtime errors.
			if sourceLen >= logsPathLen+containerIdLen {
				cid := source[logsPathLen : logsPathLen+containerIdLen]
				logp.Debug("kubernetes", "Using container id: %s", cid)
				return cid
			}

			logp.Debug("kubernetes", "Error extracting container id - source value contains matcher's logs_path, however it is too short to contain a Docker container ID.")
		}
	}

	return ""
}
