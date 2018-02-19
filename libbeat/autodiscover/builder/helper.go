package builder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

func GetContainerID(container common.MapStr) string {
	id, _ := container["id"].(string)
	return id
}

func GetContainerName(container common.MapStr) string {
	name, _ := container["name"].(string)
	return name
}

func GetAnnotationAsString(annotations map[string]string, prefix, key string) string {
	value, _ := annotations[fmt.Sprintf("%s/%s", prefix, key)]
	return value
}

func GetContainerAnnotationAsString(annotations map[string]string, prefix, container, key string) string {
	if value := GetAnnotationAsString(annotations, fmt.Sprintf("%s.%s", prefix, container), key); value != "" {
		return value
	}
	return GetAnnotationAsString(annotations, prefix, key)
}

func GetAnnotationsAsList(annotations map[string]string, prefix, key string) []string {
	value := GetAnnotationAsString(annotations, prefix, key)
	if value == "" {
		return []string{}
	}
	list := strings.Split(value, ",")

	for i := 0; i < len(list); i++ {
		list[i] = strings.TrimSpace(list[i])
	}

	return list
}

func GetContainerAnnotationsAsList(annotations map[string]string, prefix, container, key string) []string {
	if values := GetAnnotationsAsList(annotations, fmt.Sprintf("%s.%s", prefix, container), key); len(values) != 0 {
		return values
	}
	return GetAnnotationsAsList(annotations, prefix, key)
}

func IsNoOp(annotations map[string]string, prefix string) bool {
	value := GetAnnotationAsString(annotations, prefix, "disable")
	noop, _ := strconv.ParseBool(value)

	return noop
}

func IsContainerNoOp(annotations map[string]string, prefix, container string) bool {
	if IsNoOp(annotations, prefix) == true {
		return true
	}
	return IsNoOp(annotations, fmt.Sprintf("%s.%s", prefix, container))
}

func GetAnnotationsWithPrefix(annotations map[string]string, prefix, key string) map[string]string {
	result := map[string]string{}

	pref := fmt.Sprintf("%s/%s.", prefix, key)
	for k, v := range annotations {
		if strings.Index(k, pref) == 0 {
			parts := strings.Split(k, "/")
			if len(parts) == 2 {
				result[parts[1]] = v
			}
		}
	}
	return result
}

func GetContainerAnnotationsWithPrefix(annotations map[string]string, prefix, container, key string) map[string]string {
	pref := fmt.Sprintf("%s.%s", prefix, container)
	return GetAnnotationsWithPrefix(annotations, pref, key)
}
