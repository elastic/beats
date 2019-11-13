package nomad

import (
	"regexp"
	"strconv"
	"strings"

	nomad "github.com/hashicorp/nomad/api"
)

var indexRegex = regexp.MustCompile(`\[(?P<index>[0-9]+)\]`)

func FetchProperties(alloc *nomad.Allocation) map[string]interface{} {
	properties := map[string]interface{}{
		"region":     *alloc.Job.Region,
		"namespace":  alloc.Namespace,
		"job":        alloc.JobID,
		"group":      alloc.TaskGroup,
		"allocation": alloc.ID,
	}

	if matchs := indexRegex.FindStringSubmatch(alloc.Name); len(matchs) == 2 {
		index, _ := strconv.Atoi(matchs[1])
		properties["alloc_index"] = index
	}
	return properties
}

func filterMeta(alloc map[string]string, meta map[string]interface{}, prefix string) {
	for k, v := range alloc {
		if strings.HasPrefix(k, prefix) {
			meta[strings.ToLower(strings.TrimPrefix(k, prefix))] = v
		}
	}
}

func FetchMetadata(alloc *nomad.Allocation, task, prefix string) map[string]interface{} {
	meta := make(map[string]interface{})
	filterMeta(alloc.Job.Meta, meta, prefix)
	for _, tg := range alloc.Job.TaskGroups {
		if *tg.Name == alloc.TaskGroup {
			filterMeta(tg.Meta, meta, prefix)
			for _, t := range tg.Tasks {
				if t.Name == task {
					filterMeta(t.Meta, meta, prefix)
				}
			}
		}
	}
	return meta
}

func IsTerminal(alloc *nomad.Allocation) bool {
	switch alloc.ClientStatus {
	case nomad.AllocClientStatusComplete, nomad.AllocClientStatusFailed, nomad.AllocClientStatusLost:
		return true
	default:
		return false
	}
}
