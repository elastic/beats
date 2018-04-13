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
	if rawEntries, err := annotations.GetValue(prefix); err == nil {
		if entries, ok := rawEntries.(common.MapStr); ok {
			for key, rawValue := range entries {
				// If there are top level hints like co.elastic.logs/ then just add the values after the /
				// Only consider namespaced annotations
				parts := strings.Split(key, "/")
				if len(parts) == 2 {
					hintKey := fmt.Sprintf("%s.%s", parts[0], parts[1])
					// Insert only if there is no entry already. container level annotations take
					// higher priority.
					if _, err := hints.GetValue(hintKey); err != nil {
						hints.Put(hintKey, rawValue)
					}
				} else if container != "" {
					// Only consider annotations that are of type common.MapStr as we are looking for
					// container level nesting
					builderHints, ok := rawValue.(common.MapStr)
					if !ok {
						continue
					}

					// Check for <containerName>/ prefix
					for hintKey, rawVal := range builderHints {
						if strings.HasPrefix(hintKey, container) {
							// Split the key to get part[1] to be the hint
							parts := strings.Split(hintKey, "/")
							if len(parts) == 2 {
								// key will be the hint type
								hintKey := fmt.Sprintf("%s.%s", key, parts[1])
								hints.Put(hintKey, rawVal)
							}
						}
					}
				}
			}
		}
	}

	return hints
}
