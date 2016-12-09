package docker

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"

	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	ID     string
	Name   string
	Labels common.MapStr
}

func (c *Container) ToMapStr() common.MapStr {
	m := common.MapStr{
		"id":   c.ID,
		"name": c.Name,
	}

	if len(c.Labels) > 0 {
		m["labels"] = c.Labels
	}
	return m
}

func NewContainer(container *docker.APIContainers) *Container {
	return &Container{
		ID:     container.ID,
		Name:   ExtractContainerName(container.Names),
		Labels: DeDotLabels(container.Labels),
	}
}

func ExtractContainerName(names []string) string {
	output := names[0]

	if len(names) > 1 {
		for _, name := range names {
			if strings.Count(output, "/") > strings.Count(name, "/") {
				output = name
			}
		}
	}
	return strings.Trim(output, "/")
}

// DeDotLabels returns a new common.MapStr containing a copy of the labels
// where the dots in each label name have been changed to an underscore.
func DeDotLabels(labels map[string]string) common.MapStr {
	outputLabels := common.MapStr{}
	for k, v := range labels {
		// This is necessary so that ES does not interpret '.' fields as new
		// nested JSON objects, and also makes this compatible with ES 2.x.
		label := strings.Replace(k, ".", "_", -1)
		outputLabels.Put(label, v)
	}

	return outputLabels
}
