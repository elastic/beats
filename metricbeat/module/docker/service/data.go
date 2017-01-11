package service

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
	"github.com/fsouza/go-dockerclient/external/github.com/docker/api/types/swarm"
	"reflect"
)

func eventsMapping(serviceList []swarm.Service) []common.MapStr {
	myEvents := []common.MapStr{}

	for _, service := range serviceList {
		myEvents = append(myEvents, eventMapping(&service))
	}

	return myEvents
}

func eventMapping(service *swarm.Service) common.MapStr {

	event := common.MapStr{
		"id": service.ID,
		"version": service.Meta.Version.Index,
		"createdat": service.Meta.CreatedAt,
		"updatedat": service.Meta.UpdatedAt,
		"name": service.Spec.Annotations.Name,
	}

	if service.UpdateStatus.Message != "" {
		updatestatus := common.MapStr{
			"state": service.UpdateStatus.State,
			"startedat": service.UpdateStatus.StartedAt,
			"completedat": service.UpdateStatus.CompletedAt,
			"message": service.UpdateStatus.Message,
		}
		event["updatestatus"] = updatestatus
	}

	if service.Spec.Mode.Replicated != nil {
		event["mode"] = "Replicated"
		event["replicas"] = service.Spec.Mode.Replicated.Replicas
	} else {
		event["mode"] = "Global"
	}

	previousspec_labels := common.MapStr{}
	spec_labels := common.MapStr{}

	if service.PreviousSpec != nil && service.PreviousSpec.Annotations.Labels != nil {
		previousspec_labels = docker.DeDotLabels(service.PreviousSpec.Annotations.Labels)
	}

	if service.Spec.Annotations.Labels != nil {
		spec_labels = docker.DeDotLabels(service.Spec.Annotations.Labels)
	}

	if len(spec_labels) != 0 || len(previousspec_labels) != 0 {
		if len(spec_labels) != 0 && len(previousspec_labels) != 0 {
			if reflect.DeepEqual(spec_labels, previousspec_labels) {
				event["labels"] = spec_labels
			} else {
				event["previousspec.labels"] = previousspec_labels
				event["labels"] = spec_labels
			}
		} else if len(spec_labels) == 0 && len(previousspec_labels) != 0 {
			event["previousspec.labels"] = previousspec_labels
		} else {
			event["labels"] = spec_labels
		}
	}

	if service.Spec.TaskTemplate.ContainerSpec.Healthcheck != nil {
		event["healtcheck_enabled"] = true
	} else {
		event["healtcheck_enabled"] = false
	}

	return event
}
