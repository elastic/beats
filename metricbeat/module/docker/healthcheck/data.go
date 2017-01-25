package healthcheck

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"

	dc "github.com/fsouza/go-dockerclient"
	"strings"
)

func eventsMapping(containersList []dc.APIContainers, m *MetricSet) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, container := range containersList {
		returnevent := eventMapping(&container, m)
		// Compare event to empty event
		if returnevent != nil {
			myEvents = append(myEvents, returnevent)
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
						"id":   cont.ID,
					},
				},
				"status":        container.State.Health.Status,
				"failingstreak": container.State.Health.FailingStreak,
				"event": common.MapStr{
					"start_date": common.Time(container.State.Health.Log[last_event].Start),
					"end_date":   common.Time(container.State.Health.Log[last_event].End),
					"exit_code":  container.State.Health.Log[last_event].ExitCode,
					"output":     container.State.Health.Log[last_event].Output,
				},
			}
			return event
		}
	}
	return nil
}
