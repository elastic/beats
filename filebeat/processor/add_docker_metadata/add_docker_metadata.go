package add_docker_metadata

import (
"strings"

"github.com/elastic/beats/libbeat/common"
"github.com/elastic/beats/libbeat/logp"
"github.com/elastic/beats/libbeat/processors/add_docker_metadata"
)
const LogPathMatcherName = "logs_path"


func init() {
	add_docker_metadata.Matcher=&FileMatch{}
}

type FileMatch struct {
	LogsPath string
}

func (match *FileMatch) MetadataIndex(event common.MapStr) string{
	if value, ok := event["source"]; ok {
		source := value.(string)
		cid := ""
		if strings.Contains(source, match.LogsPath) {
			//Docker container is 64 chars in length
			cid = source[len(match.LogsPath) : len(match.LogsPath)+64]
		}
		if cid != "" {
			return cid
		}
	}

	return ""
}

func (match *FileMatch) InitMatcher(cfg common.Config) error{
	config := struct {
		LogsPath string `config:"logs_path"`
	}{
		LogsPath: "/var/lib/docker/containers/",
	}

	err := cfg.Unpack(&config)
	if err != nil || config.LogsPath == "" {
		return err
	}

	match.LogsPath = config.LogsPath
	return nil
}




