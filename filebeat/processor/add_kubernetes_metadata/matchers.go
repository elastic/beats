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
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// InitializeModule initializes this module.
func InitializeModule() {
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

func newLogsPathMatcher(cfg conf.C, log *logp.Logger) (add_kubernetes_metadata.Matcher, error) {
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

	log.Debugf("logs_path matcher log path: %s", logPath)
	log.Debugf("logs_path matcher resource type: %s", resourceType)

	return &LogPathMatcher{LogsPath: logPath, ResourceType: resourceType, logger: log}, nil
}

// Docker container ID is a 64-character-long hexadecimal string
const containerIdLen = 64

func (f *LogPathMatcher) MetadataIndexCandidates(event mapstr.M) []string {
	value, err := event.GetValue("log.file.path")
	if err != nil {
		f.logger.Debugf("Error extracting log.file.path from the event: %s.", event)
		return nil
	}

	source, ok := value.(string)
	if !ok {
		f.logger.Debugf("Error extracting log.file.path from the event: value is not a string.")
		return nil
	}
	f.logger.Debugf("Incoming log.file.path value: %s", source)

	if !strings.HasPrefix(source, f.LogsPath) {
		f.logger.Debugf("log.file.path value does not have matcher's logs_path '%s' as a prefix, skipping...", f.LogsPath)
		return nil
	}

	sourceLen := len(source)
	logsPathLen := len(f.LogsPath)

	if f.ResourceType == "pod" {
		// Check the basename only — a namespace like "corp.logging" would otherwise trigger
		// this guard. strings.Contains (not HasSuffix) handles rotated names like 0.log.20220221.
		basename := source[strings.LastIndex(source, pathSeparator)+1:]
		if strings.Contains(basename, ".log") {
			// Specify a pod resource type when writing logs into manually mounted log volume,
			// those logs appear under "/var/lib/kubelet/pods/<pod_id>/volumes/..."
			if strings.HasPrefix(f.LogsPath, podKubeletLogsPath()) {
				pathDirs := strings.Split(source, pathSeparator)
				podUIDPos := 5
				if len(pathDirs) > podUIDPos {
					podUID := pathDirs[podUIDPos]
					f.logger.Debugf("Using pod uid: %s", podUID)
					return []string{podUID}
				}
			}
			// In case of the Kubernetes log path "/var/log/pods/",
			// the pod UID is extracted from the directory name and the container name and
			// restart count from the subsequent path segments.
			// file name example: "/var/log/pods/<namespace>_<pod_name>_<pod_uid>/<container_name>/<restart_count>.log"
			if strings.HasPrefix(f.LogsPath, podLogsPath()) {
				pathDirs := strings.Split(source, pathSeparator)
				// pathDirs: ["", "var", "log", "pods", "<ns>_<pod>_<uid>", "<container>", "<n>.log"]
				//                  0     1     2      3          4               5              6
				podUIDPos := 4
				if len(pathDirs) > podUIDPos {
					uidParts := strings.Split(pathDirs[podUIDPos], "_")
					if len(uidParts) > 2 {
						podUID := uidParts[len(uidParts)-1]
						f.logger.Debugf("Using pod uid: %s", podUID)

						if len(pathDirs) > podUIDPos+1 {
							containerName := pathDirs[podUIDPos+1]
							containerIndex := podUID + "/" + containerName
							if len(pathDirs) > podUIDPos+2 {
								basename := pathDirs[podUIDPos+2]
								// strip .gz so compressed rotated logs (e.g. 0.log.gz) are parsed like 0.log
								basename = strings.TrimSuffix(basename, ".gz")
								// guard against digit directory segments (e.g. container/3/real.log)
								if logName, ok := strings.CutSuffix(basename, ".log"); ok {
									if _, err := strconv.Atoi(logName); err == nil {
										f.logger.Debugf("Using pod uid/container/restart: %s/%s/%s", podUID, containerName, logName)
										return []string{
											containerIndex + "/" + logName,
											containerIndex,
											podUID,
										}
									}
								}
							}
							f.logger.Debugf("Using pod uid/container: %s/%s", podUID, containerName)
							return []string{containerIndex, podUID}
						}
						return []string{podUID}
					}
				}
			}

			f.logger.Errorf("Error extracting pod UID from '%s': configured logs_path '%s' does not match a known Kubernetes pod log directory (/var/log/pods/ or /var/lib/kubelet/pods/)", source, f.LogsPath)
			return nil
		}
		// Source filename has no .log extension — silently drop.
		return nil
	} else {
		// In case of the Kubernetes log path "/var/log/containers/",
		// the container ID will be located right before the ".log" extension.
		// file name example: /var/log/containers/<pod_name>_<namespace>_<container_name>-<continer_id>.log
		if strings.HasPrefix(f.LogsPath, containerLogsPath()) && strings.HasSuffix(source, ".log") && sourceLen >= containerIdLen+4 {
			containerIDEnd := sourceLen - 4
			cid := source[containerIDEnd-containerIdLen : containerIDEnd]
			f.logger.Debugf("Using container id: %s", cid)
			return []string{cid}
		}

		// In any other case, we assume the container ID will follow right after the log path.
		// However we need to check the length to prevent "slice bound out of range" runtime errors.
		// for the default log path /var/lib/docker/containers/ container ID will follow right after the log path.
		// file name example: /var/lib/docker/containers/<container_id>/<container_id>-json.log
		if sourceLen >= logsPathLen+containerIdLen {
			cid := source[logsPathLen : logsPathLen+containerIdLen]
			f.logger.Debugf("Using container id: %s", cid)
			return []string{cid}
		}
	}
	f.logger.Error("Error extracting container id - source value contains matcher's logs_path, however it is too short to contain a Docker container ID.")
	return nil
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
