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

var Matcher Match = &FieldMatch{};

type Match interface {
	InitMatcher(cfg common.Config) error
	MetadataIndex(event common.MapStr) string
}

type NoopMatch struct {
	MatchFields []string
}

type FieldMatch struct {
	MatchFields []string
}

func (match *FieldMatch) MetadataIndex(event common.MapStr) string{
	for _, field := range match.MatchFields {
		keyIface, err := event.GetValue(field)
		if err == nil {
			key, ok := keyIface.(string)
			if ok {
				return key
			}
		}
	}

	return ""
}

func (match *NoopMatch) InitMatcher(cfg common.Config) error {
	return nil
}

func (match *NoopMatch) MetadataIndex(event common.MapStr) string{
	return ""
}

func (match *FieldMatch) InitMatcher(cfg common.Config) error {
	config := struct {
		Fields []string `config:"match_fields"`
	}{}
	err := cfg.Unpack(&config)
	if err != nil || len(config.Fields) ==0 {
		return err
	}
	match.MatchFields = config.Fields
	return nil
}



type addDockerMetadata struct {
	watcher Watcher
	fields  []string
	matcher   Match
}

func newDockerMetadataProcessor(cfg common.Config) (processors.Processor, error) {
	return BuildDockerMetadataProcessor(cfg, NewWatcher)
}

func BuildDockerMetadataProcessor(cfg common.Config, watcherConstructor WatcherConstructor) (processors.Processor, error) {
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

	if err = Matcher.InitMatcher(cfg); err != nil {
	      Matcher = &NoopMatch{}
	}

	return &addDockerMetadata{
		watcher: watcher,
		fields:  config.Fields,
		matcher: Matcher,
	}, nil
}

func (d *addDockerMetadata) Run(event common.MapStr) (common.MapStr, error) {

	cid := d.matcher.MetadataIndex(event)

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
