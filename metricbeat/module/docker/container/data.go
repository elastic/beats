package container

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"

	"strings"
	dc "github.com/fsouza/go-dockerclient"
)

func eventsMapping(containersList []dc.APIContainers, m *MetricSet) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, container := range containersList {
		myEvents = append(myEvents, eventMapping(&container, m))
	}
	return myEvents
}

func eventMapping(cont *dc.APIContainers, m *MetricSet) common.MapStr {
	event := common.MapStr{
		"created": common.Time(time.Unix(cont.Created, 0)),
		"id":      cont.ID,
		"name":    docker.ExtractContainerName(cont.Names),
		"command": cont.Command,
		"image":   cont.Image,
		"size": common.MapStr{
			"root_fs": cont.SizeRootFs,
			"rw":      cont.SizeRw,
		},
		"status": cont.Status,
	}

if strings.Contains(cont.Status, "(") && strings.Contains(cont.Status, ")") {
	container, _ := m.dockerClient.InspectContainer(cont.ID)

	last_event :=  len(container.State.Health.Log)-1
	if last_event >= 0 {
		health := common.MapStr{
			"status": container.State.Health.Status,
			"failingstreak": container.State.Health.FailingStreak,
			"event_start_date": container.State.Health.Log[last_event].Start,
			"event_end_date": container.State.Health.Log[last_event].End,
			"event_exit_code": container.State.Health.Log[last_event].ExitCode,
			"event_output": container.State.Health.Log[last_event].Output,
		}
		event["health"] = health
	}
}

	labels := docker.DeDotLabels(cont.Labels)

	if len(labels) > 0 {
		event["labels"] = labels
	}

	return event
}
