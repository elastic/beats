package healthcheck

import (
	//"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
	"reflect"
	"strings"
)

func eventsMapping(containersList []dc.APIContainers, m *MetricSet) []common.MapStr {
	myEvents := []common.MapStr{}
	// Set an empty map in order to detect empty healthcheck event
	emptyEvent := common.MapStr{}
	for _, container := range containersList {
		returnevent := eventMapping(&container, m)
		// Compare event to empty event
		if !reflect.DeepEqual(emptyEvent, returnevent) {
			myEvents = append(myEvents, eventMapping(&container, m))
		}
	}
	return myEvents
}

func eventMapping(cont *dc.APIContainers, m *MetricSet) common.MapStr {
	event := common.MapStr{}
	// Detect if healthcheck is available for container
	if strings.Contains(cont.Status, "(") && strings.Contains(cont.Status, ")") {
		container, _ := m.dockerClient.InspectContainer(cont.ID)
		last_event := len(container.State.Health.Log) - 1
		// Detect if an healthcheck already occured
		if last_event >= 0 {
			event = common.MapStr{
				mb.ModuleData: common.MapStr{
					"container": common.MapStr{
						"name": docker.ExtractContainerName(cont.Names),
					},
				},
				"status":           container.State.Health.Status,
				"failingstreak":    container.State.Health.FailingStreak,
				"event_start_date": common.Time(container.State.Health.Log[last_event].Start),
				"event_end_date":   common.Time(container.State.Health.Log[last_event].End),
				"event_exit_code":  container.State.Health.Log[last_event].ExitCode,
				"event_output":     container.State.Health.Log[last_event].Output,
			}
		}
	}

	return event
}
