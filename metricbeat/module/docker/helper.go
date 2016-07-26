package docker

import (
	"strings"
	"github.com/elastic/beats/libbeat/common"
	"github.com/fsouza/go-dockerclient"
)

type Container struct{
	Id string
	Name string
	Labels []common.MapStr
	//Socket *string
}

func InitCurrentContainer(container *docker.APIContainers) *Container{
	return &Container{
		Id: container.ID,
		Name: extractContainerName(container.Names),
		Labels: buildLabelArray(container.Labels),
		//Socket: d.Socket,
	}
}
func extractContainerName(names []string) string {
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
func buildLabelArray(labels map[string]string) []common.MapStr {

	output_labels := make([]common.MapStr, len(labels))

	i := 0
	for k, v := range labels {
		label := strings.Replace(k, ".", "_", -1)
		output_labels[i] = common.MapStr{
			"key":   label,
			"value": v,
		}
		i++
	}
	return output_labels
}
