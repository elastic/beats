package add_kubernetes_metadata

import (
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
)

var regexpKubernetes *regexp.Regexp
var regexpDocker *regexp.Regexp
var regexpGeneric *regexp.Regexp

func init() {
	// Regular expressions used in the path matcher
	regexpKubernetes = regexp.MustCompile("\\/var\\/log\\/containers\\/.+-([A-Fa-f0-9]{64})\\.log")
	regexpDocker = regexp.MustCompile("\\/var\\/lib\\/docker\\/containers\\/([A-Fa-f0-9]{64})\\/.+\\.log")
	regexpGeneric = regexp.MustCompile("[A-Fa-f0-9]{64}")

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

	return &LogPathMatcher{LogsPath: logPath}, nil
}

func (f *LogPathMatcher) MetadataIndex(event common.MapStr) string {
	if value, ok := event["source"]; ok {
		source := value.(string)
		logp.Debug("kubernetes", "Incoming source value: %s", source)

		// Variant 1: Kubernetes log path "/var/log/containers/"
		matches := regexpKubernetes.FindStringSubmatch(source)
		if matches != nil {
			cid := matches[1]
			logp.Debug("kubernetes", "Using container id: %s", cid)
			return cid;
		}

		// Variant 2: Docker log path "/var/lib/docker/containers/"
		matches = regexpDocker.FindStringSubmatch(source)
		if matches != nil {
			cid := matches[1]
			logp.Debug("kubernetes", "Using container id: %s", cid)
			return cid;
		}

		// Variant 3: Generic fallback
		cid := regexpGeneric.FindString(source)
		if cid != "" {
			logp.Debug("kubernetes", "Using container id: %s", cid)
			return cid;
		}

		logp.Debug("kubernetes", "No container id found in source.")
	}

	return ""
}
