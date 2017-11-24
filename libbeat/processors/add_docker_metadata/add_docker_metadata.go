package add_docker_metadata

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
)

func init() {
	processors.RegisterPlugin("add_docker_metadata", newDockerMetadataProcessor)
}

type addDockerMetadata struct {
	watcher         docker.Watcher
	fields          []string
	sourceProcessor processors.Processor
}

func newDockerMetadataProcessor(cfg *common.Config) (processors.Processor, error) {
	return buildDockerMetadataProcessor(cfg, docker.NewWatcher)
}

func buildDockerMetadataProcessor(cfg *common.Config, watcherConstructor docker.WatcherConstructor) (processors.Processor, error) {
	cfgwarn.Beta("The add_docker_metadata processor is beta")

	config := defaultConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_docker_metadata configuration: %s", err)
	}

	watcher, err := watcherConstructor(config.Host, config.TLS)
	if err != nil {
		return nil, err
	}

	if err = watcher.Start(); err != nil {
		return nil, err
	}

	// Use extract_field processor to get container id from source file path
	var sourceProcessor processors.Processor
	if config.MatchSource {
		var procConf, _ = common.NewConfigFrom(map[string]interface{}{
			"field":     "source",
			"separator": "/",
			"index":     config.SourceIndex,
			"target":    "docker.container.id",
		})
		sourceProcessor, err = actions.NewExtractField(procConf)
		if err != nil {
			return nil, err
		}

		// Ensure `docker.container.id` is matched:
		config.Fields = append(config.Fields, "docker.container.id")
	}

	return &addDockerMetadata{
		watcher:         watcher,
		fields:          config.Fields,
		sourceProcessor: sourceProcessor,
	}, nil
}

func (d *addDockerMetadata) Run(event *beat.Event) (*beat.Event, error) {
	var cid string
	var err error

	// Process source field
	if d.sourceProcessor != nil {
		if event.Fields["source"] != nil {
			event, err = d.sourceProcessor.Run(event)
			if err != nil {
				logp.Debug("docker", "Error while extracting container ID from source path: %v", err)
				return event, nil
			}
		}
	}

	for _, field := range d.fields {
		value, err := event.GetValue(field)
		if err != nil {
			continue
		}

		if strValue, ok := value.(string); ok {
			cid = strValue
		}
	}

	if cid == "" {
		return event, nil
	}

	container := d.watcher.Container(cid)
	if container != nil {
		meta := common.MapStr{}
		metaIface, ok := event.Fields["docker"]
		if ok {
			meta = metaIface.(common.MapStr)
		}

		if len(container.Labels) > 0 {
			labels := common.MapStr{}
			for k, v := range container.Labels {
				labels.Put(k, v)
			}
			meta.Put("container.labels", labels)
		}

		meta.Put("container.id", container.ID)
		meta.Put("container.image", container.Image)
		meta.Put("container.name", container.Name)
		event.Fields["docker"] = meta
	} else {
		logp.Debug("docker", "Container not found: %s", cid)
	}

	return event, nil
}

func (d *addDockerMetadata) String() string {
	return "add_docker_metadata=[fields=" + strings.Join(d.fields, ", ") + "]"
}
