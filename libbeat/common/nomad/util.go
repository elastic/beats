// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package nomad

import (
	"regexp"
	"strconv"
	"strings"

	nomad "github.com/hashicorp/nomad/api"
)

var indexRegex = regexp.MustCompile(`\[(?P<index>[0-9]+)\]`)

// FetchProperties returns a map with all the properties of the allocation
func FetchProperties(alloc *nomad.Allocation) map[string]interface{} {
	properties := map[string]interface{}{
		"region":     *alloc.Job.Region,
		"namespace":  alloc.Namespace,
		"job":        alloc.JobID,
		"group":      alloc.TaskGroup,
		"allocation": alloc.ID,
	}

	if matches := indexRegex.FindStringSubmatch(alloc.Name); len(matches) == 2 {
		index, _ := strconv.Atoi(matches[1])
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

// FetchMetadata returns a map with the metadata that starts with prefix
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

// IsTerminal returns if the allocation is terminal
func IsTerminal(alloc *nomad.Allocation) bool {
	switch alloc.ClientStatus {
	case nomad.AllocClientStatusComplete, nomad.AllocClientStatusFailed, nomad.AllocClientStatusLost:
		return true
	default:
		return false
	}
}
