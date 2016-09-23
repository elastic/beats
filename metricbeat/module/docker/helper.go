package docker

import (
	"strings"

	"github.com/fsouza/go-dockerclient"

	"github.com/elastic/beats/libbeat/common"
	"sort"
)

type Container struct {
	Id     string
	Name   string
	Labels []common.MapStr
	Socket *string
}

func InitCurrentContainer(container *docker.APIContainers) *Container {
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

	output_labels := make([]common.MapStr, len(labels))
	i := 0
	var keys []string
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, k := range keys {
		label := strings.Replace(k, ".", "_", -1)
		output_labels[i] = common.MapStr{
			"key":   label,
			"value": labels[k],
		}
		i++
	}
	return output_labels
}

func ConvertContainerPorts(ports *[]docker.APIPort) []map[string]interface{} {
	var outputPorts = []map[string]interface{}{}
	for _, port := range *ports {
		outputPort := common.MapStr{
			"ip":          port.IP,
			"privatePort": port.PrivatePort,
			"publicPort":  port.PublicPort,
			"type":        port.Type,
		}
		outputPorts = append(outputPorts, outputPort)
	}

	return outputPorts
}
