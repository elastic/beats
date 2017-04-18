package kubernetes

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/annotate/kubernetes"
)

func init() {
	kubernetes.Indexing.AddMatcher(LogPathMatcherName, newLogsPathMatcher)

	indexer := kubernetes.Indexing.GetIndexer(kubernetes.ContainerIndexerName)
	//Add a container indexer by default.
	if indexer != nil {
		cfg := common.NewConfig()
		container, err := indexer(*cfg)

		if err == nil {
			kubernetes.Indexing.AddDefaultIndexer(container)
		} else {
			logp.Err("Unable to load indexer plugin due to error: %v", err)
		}
	} else {
		logp.Err("Unable to get indexer plugin %s", kubernetes.ContainerIndexerName)
	}

	//Add a log path matcher which can extract container ID from the "source" field.
	matcher := kubernetes.Indexing.GetMatcher(LogPathMatcherName)

	if matcher != nil {
		cfg := common.NewConfig()
		logsPathMatcher, err := matcher(*cfg)
		if err == nil {
			kubernetes.Indexing.AddDefaultMatcher(logsPathMatcher)
		} else {
			logp.Err("Unable to load matcher plugin due to error: %v", err)
		}
	} else {
		logp.Err("Unable to get matcher plugin %s", LogPathMatcherName)
	}

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
