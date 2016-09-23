package container

import (
	"time"

	dc "github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(containersList []dc.APIContainers) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, container := range containersList {
		myEvents = append(myEvents, eventMapping(&container))
	}
	return myEvents
}
func eventMapping(mycontainer *dc.APIContainers) common.MapStr {

	event := common.MapStr{
		"@timestamp": time.Now(),
		"container": common.MapStr{
			"created":      common.Time(time.Unix(mycontainer.Created, 0)),
			"id":           mycontainer.ID,
			"name":         docker.ExtractContainerName(mycontainer.Names),
			"labels":       docker.BuildLabelArray(mycontainer.Labels),
			"command":      mycontainer.Command,
			"image":        mycontainer.Image,
			"ports":        docker.ConvertContainerPorts(&mycontainer.Ports),
			"size_root_fs": mycontainer.SizeRootFs,
			"size_rw":      mycontainer.SizeRw,
			"status":       mycontainer.Status,
		},
		"socket": docker.GetSocket(),
	}

	return event
}
