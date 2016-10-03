package docker

import (
	"sort"
	"strings"

	"github.com/elastic/beats/libbeat/common"

	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	Id     string
	Name   string
	Labels []common.MapStr
	Socket *string
}

func (c *Container) ToMapStr() common.MapStr {
	m := common.MapStr{
		"id":   c.Id,
		"name": c.Name,
		// TODO: Is this really needed
		"socket": GetSocket(),
	}

	// Only add labels array if not 0
	if len(c.Labels) > 0 {
		m["labels"] = c.Labels
	}
	return m
}

func NewContainer(container *docker.APIContainers) *Container {
	return &Container{
		Id:     container.ID,
		Name:   ExtractContainerName(container.Names),
		Labels: BuildLabelArray(container.Labels),
	}
}

func ExtractContainerName(names []string) string {
	output := names[0]

	if cap(names) > 1 {
		for _, name := range names {
			if strings.Count(output, "/") > strings.Count(name, "/") {
				output = name
			}
		}
	}
	return strings.Trim(output, "/")
}

func BuildLabelArray(labels map[string]string) []common.MapStr {

	outputLabels := make([]common.MapStr, len(labels))
	i := 0
	var keys []string
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, k := range keys {
		// Replace all . in the labels by _
		// TODO: WHY?
		label := strings.Replace(k, ".", "_", -1)
		outputLabels[i] = common.MapStr{
			"key":   label,
			"value": labels[k],
		}
		i++
	}
	return outputLabels
}
