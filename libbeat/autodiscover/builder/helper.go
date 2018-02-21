package builder

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// GetContainerID returns the id of a container
func GetContainerID(container common.MapStr) string {
	id, _ := container["id"].(string)
	return id
}

// GetContainerName returns the name of a container
func GetContainerName(container common.MapStr) string {
	name, _ := container["name"].(string)
	return name
}

// GetHintString takes a hint and returns its value as a string
func GetHintString(hints common.MapStr, key, config string) string {
	if iface, err := hints.GetValue(fmt.Sprintf("%s.%s", key, config)); err == nil {
		if str, ok := iface.(string); ok {
			return str
		}
	}

	return ""
}

// GetHintMapStr takes a hint and returns a MapStr
func GetHintMapStr(hints common.MapStr, key, config string) common.MapStr {
	if iface, err := hints.GetValue(fmt.Sprintf("%s.%s", key, config)); err == nil {
		if mapstr, ok := iface.(common.MapStr); ok {
			return mapstr
		}
	}

	return nil
}

// GetHintAsList takes a hint and returns the value as lists.
func GetHintAsList(hints common.MapStr, key, config string) []string {
	if str := GetHintString(hints, key, config); str != "" {
		return getStringAsList(str)
	}

	return nil
}

func getStringAsList(input string) []string {
	if input == "" {
		return []string{}
	}
	list := strings.Split(input, ",")

	for i := 0; i < len(list); i++ {
		list[i] = strings.TrimSpace(list[i])
	}

	return list
}

// IsNoOp is a big red button to prevent spinning up Runners in case of issues.
func IsNoOp(hints common.MapStr, key string) bool {
	if value, err := hints.GetValue(fmt.Sprintf("%s.disable", key)); err == nil {
		noop, _ := strconv.ParseBool(value.(string))
		return noop
	}

	return false
}

// GenerateHints parses annotations based on a prefix and sets up hints that can be picked up by individual Beats.
func GenerateHints(annotations common.MapStr, container, prefix string) common.MapStr {
	hints := common.MapStr{}
	plen := len(prefix)

	for key, value := range annotations {
		// Filter out all annotations which start with the prefix
		if strings.Index(key, prefix) == 0 {
			subKey := key[plen:]
			// Split an annotation by /. Ex co.elastic.metrics/module would split to ["metrics", "module"]
			// part[0] would give the type of config and part[1] would give the config entry
			parts := strings.Split(subKey, "/")
			if len(parts) == 0 || parts[0] == "" {
				continue
			}
			// tc stands for type and container
			// Split part[0] to get the builder type and the container if it exists
			tc := strings.Split(parts[0], ".")
			k := fmt.Sprintf("%s.%s", tc[0], parts[1])
			if len(tc) == 2 && container != "" && tc[1] == container {
				// Container specific properties always carry higher preference.
				// Overwrite properties even if they exist.
				hints.Put(k, value)
			} else {
				// Only insert the config if it doesn't already exist
				if _, err := hints.GetValue(k); err != nil {
					hints.Put(k, value)
				}
			}
		}
	}

	return hints
}
