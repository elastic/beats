package kubernetes

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/kubernetes"
)

func init() {
	kubernetes.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)
	cfg := common.NewConfig()

	//Add a container indexer config by default.
	kubernetes.Indexing.AddDefaultIndexerConfig(kubernetes.ContainerIndexerName, *cfg)

	//Add a log path matcher which can extract container ID from the "source" field.
	kubernetes.Indexing.AddDefaultMatcherConfig(LogPathMatcherName, *cfg)
}

const LogPathMatcherName = "logs_path"

type LogPathMatcher struct {
	LogsPath string
}

func newLogsPathMatcher(cfg common.Config) (kubernetes.Matcher, error) {
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

	return &LogPathMatcher{LogsPath: logPath}, nil
}

func (f *LogPathMatcher) MetadataIndex(event common.MapStr) string {

	if value, ok := event["source"]; ok {
		source := value.(string)
		logp.Debug("kubernetes", "Incoming source value: ", source)
		cid := ""
		if strings.Contains(source, f.LogsPath) {
			//Docker container is 64 chars in length
			cid = source[len(f.LogsPath) : len(f.LogsPath)+64]
		}
		logp.Debug("kubernetes", "Using container id: ", cid)

		if cid != "" {
			return cid
		}
	}

	return ""
}
