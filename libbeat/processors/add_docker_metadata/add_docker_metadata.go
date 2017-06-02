package add_docker_metadata

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

func init() {
	processors.RegisterPlugin("add_docker_metadata", newDockerMetadataProcessor)
}

type addDockerMetadata struct {
	watcher Watcher
	fields  []string
}

func newDockerMetadataProcessor(cfg common.Config) (processors.Processor, error) {
	return buildDockerMetadataProcessor(cfg, NewWatcher)
}

func buildDockerMetadataProcessor(cfg common.Config, watcherConstructor WatcherConstructor) (processors.Processor, error) {
	logp.Beta("The add_docker_metadata processor is beta")

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

	return &addDockerMetadata{
		watcher: watcher,
		fields:  config.Fields,
	}, nil
}

func (d *addDockerMetadata) Run(event common.MapStr) (common.MapStr, error) {
	var cid string
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
		metaIface, ok := event["docker"]
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
		event["docker"] = meta
	} else {
		logp.Debug("docker", "Container not found: %s", cid)
	}

	return event, nil
}

func (d *addDockerMetadata) String() string {
	return "add_docker_metadata=[fields=" + strings.Join(d.fields, ", ") + "]"
}
