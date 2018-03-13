package docker

import (
	"strings"

	"github.com/docker/docker/api/types"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/safemapstr"
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

func NewContainer(container *types.Container) *Container {
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
// where the dots have been converted into nested structure, avoiding
// possible mapping errors
func DeDotLabels(labels map[string]string) common.MapStr {
	outputLabels := common.MapStr{}
	for k, v := range labels {
		safemapstr.Put(outputLabels, k, v)
	}

	return outputLabels
}
