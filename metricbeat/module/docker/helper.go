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

// NewContainer converts Docker API container to an internal structure, it applies
// dedot to container labels if dedot is true, or stores them in a nested way if it's
// false
func NewContainer(container *types.Container, dedot bool) *Container {
	return &Container{
		ID:     container.ID,
		Name:   ExtractContainerName(container.Names),
		Labels: DeDotLabels(container.Labels, dedot),
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
func DeDotLabels(labels map[string]string, dedot bool) common.MapStr {
	outputLabels := common.MapStr{}
	for k, v := range labels {
		if dedot {
			// This is necessary so that ES does not interpret '.' fields as new
			// nested JSON objects, and also makes this compatible with ES 2.x.
			label := common.DeDot(k)
			outputLabels.Put(label, v)
		} else {
			// If we don't dedot we ensure there are no mapping errors with safemapstr
			safemapstr.Put(outputLabels, k, v)
		}
	}

	return outputLabels
}
