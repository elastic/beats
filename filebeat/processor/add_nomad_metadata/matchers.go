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

package add_nomad_metadata

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/add_nomad_metadata"
)

// LogPathMatcherName is the name of LogPathMatcher
const LogPathMatcherName = "logs_path"
const pathSeparator = string(os.PathSeparator)
const allocIDRegex = "[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}"

// const allocIDTypeRegex = "([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}).*(stdout|stderr)"

func init() {
	add_nomad_metadata.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)
	cfg := common.NewConfig()

	//Add a container indexer config by default.
	add_nomad_metadata.Indexing.AddDefaultIndexerConfig(add_nomad_metadata.ContainerIndexerName, *cfg)

	//Add a log path matcher which can extract container ID from the "source" field.
	add_nomad_metadata.Indexing.AddDefaultMatcherConfig(LogPathMatcherName, *cfg)
}

// LogPathMatcher matches an event by the UUID in the path
type LogPathMatcher struct {
	LogsPath     string
	allocIDRegex *regexp.Regexp
}

func newLogsPathMatcher(cfg common.Config) (add_nomad_metadata.Matcher, error) {
	config := struct {
		LogsPath string `config:"path"`
	}{
		LogsPath: defaultLogPath(),
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return nil, fmt.Errorf("fail to unpack the `logs_path` configuration: %s", err)
	}

	logPath := config.LogsPath
	if logPath[len(logPath)-1:] != pathSeparator {
		logPath = logPath + pathSeparator
	}

	logp.Debug("nomad", "logs_path matcher log path: %s", logPath)

	return &LogPathMatcher{
		LogsPath:     logPath,
		allocIDRegex: regexp.MustCompile(allocIDRegex),
	}, nil
}

// MetadataIndex returns the index key to be used for enriching the event with the proper metadata
// which is the allocation id from the event `log.file.path` field
func (m *LogPathMatcher) MetadataIndex(event common.MapStr) string {
	value, err := event.GetValue("log.file.path")

	if err == nil {
		path := value.(string)
		logp.Debug("nomad", "Incoming log.file.path value: %s", path)

		if !strings.Contains(path, m.LogsPath) {
			logp.Debug("nomad", "Error extracting allocation id - source value does not contain matcher's logs_path '%s'.", m.LogsPath)
			return ""
		}

		// `log.file.path` looks something like:
		// /appdata/nomad/alloc/389d1bc4-fae4-6956-9f66-6df59a0f11f0/alloc/logs/app-name.stderr.0
		// /appdata/nomad/alloc/18e5cd07-03bb-be76-35e5-39c799d369e6/alloc/logs/app-name.stdout.0

		if !m.allocIDRegex.MatchString(path) {
			logp.Debug("nomad", "Error extracting allocation id - source value doesn't contain a valid UUID '%s'.", path)
			return ""
		}

		return m.allocIDRegex.FindString(path)
	}

	return ""
}

func defaultLogPath() string {
	return "/var/lib/nomad"
}
