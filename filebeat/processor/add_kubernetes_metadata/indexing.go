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

	return &LogPathMatcher{LogsPath: logPath}, nil
}

// Docker container ID is a 64-character-long hexadecimal string
const containerIdLen = 64

// Minimum `source` lengths are calculated in order to prevent "slice bound out of range"
// runtime errors in case `source` is shorter than expected.
const kubernetesLogPath = "/var/log/containers/"
const kubernetesMinSourceLen = len(kubernetesLogPath) + containerIdLen + 4

const dockerLogPath = "/var/lib/docker/containers/"
const dockerLogPathLen = len(dockerLogPath)
const dockerMinSourceLen = dockerLogPathLen + containerIdLen

func (f *LogPathMatcher) MetadataIndex(event common.MapStr) string {
	if value, ok := event["source"]; ok {
		source := value.(string)
		logp.Debug("kubernetes", "Incoming source value: %s", source)

		sourceLen := len(source)

		// Variant 1: Kubernetes log path "/var/log/containers/...-${cid}.log"
		if strings.HasPrefix(source, kubernetesLogPath) &&
		   strings.HasSuffix(source, ".log") &&
		   sourceLen >= kubernetesMinSourceLen {
			containerIdEnd := sourceLen - 4
			cid := source[containerIdEnd - containerIdLen : containerIdEnd]
			logp.Debug("kubernetes", "Using container id: %s", cid)
			return cid;
		}

		// Variant 2: Docker log path "/var/lib/docker/containers/${cid}/${cid}-json.log"
		if strings.HasPrefix(source, dockerLogPath) &&
		   sourceLen >= dockerMinSourceLen {
			cid := source[dockerLogPathLen : dockerLogPathLen + containerIdLen]
			logp.Debug("kubernetes", "Using container id: %s", cid)
			return cid;
		}

		logp.Debug("kubernetes", "No container id found in source.")
	}

	return ""
}
