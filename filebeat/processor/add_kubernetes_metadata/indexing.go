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
	LogsPath string
}

func newLogsPathMatcher(cfg common.Config) (add_kubernetes_metadata.Matcher, error) {
	config := struct {
		LogsPath string `config:"logs_path"`
	}{
		LogsPath: "/var/lib/docker/containers/",
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return nil, fmt.Errorf("fail to unpack the `logs_path` configuration: %s", err)
	}

	logPath := config.LogsPath
	if logPath[len(logPath)-1:] != "/" {
		logPath = logPath + "/"
	}

	logp.Debug("kubernetes", "logs_path matcher log path: %s", logPath)

	return &LogPathMatcher{LogsPath: logPath}, nil
}

func (f *LogPathMatcher) MetadataIndex(event common.MapStr) string {
	if value, ok := event["source"]; ok {
		source := value.(string)
		logp.Debug("kubernetes", "Incoming source value: %s", source)
		cid := ""
		if strings.Contains(source, f.LogsPath) {
			if f.LogsPath == "/var/log/containers/" && strings.HasSuffix(source, ".log") {
				// In case of the Kubernetes log path "/var/log/containers/",
				// the container ID will be located right before the ".log" ending.
				sourceLen := len(source)
				cid = source[sourceLen-68 : sourceLen-4]
			} else {
				// In any other case, we assume the container ID will follow right after the log path.
				//Docker container is 64 chars in length
				cid = source[len(f.LogsPath) : len(f.LogsPath)+64]
			}
			logp.Debug("kubernetes", "Using container id: %s", cid)
		} else {
			logp.Debug("kubernetes", "Error extracting container id - source value does not contain log path.")
		}

		if cid != "" {
			return cid
		}
	}

	return ""
}
