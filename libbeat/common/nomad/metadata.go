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
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/imdario/mergo"
)

// MetaGenerator builds metadata objects for allocations
type MetaGenerator interface {
	// ResourceMetadata generates metadata for the given allocation
	ResourceMetadata(obj Resource) common.MapStr

	// AllocationNodeName returns the name of the node where the Task is allocated
	AllocationNodeName(id string) (string, error)
}

// MetaGeneratorConfig settings
type MetaGeneratorConfig struct {
	IncludeLabels      []string `config:"include_labels"`
	ExcludeLabels      []string `config:"exclude_labels"`
	IncludeAnnotations []string `config:"include_annotations"`

	// Undocumented settings, to be deprecated in favor of `drop_fields` processor:
	IncludeCreatorMetadata bool `config:"include_creator_metadata"`
	LabelsDedot            bool `config:"labels.dedot"`
	AnnotationsDedot       bool `config:"annotations.dedot"`
	client                 *Client
	nodesCache             map[string]string
}

type metaGenerator = MetaGeneratorConfig

// NewMetaGenerator initializes and returns a new kubernetes metadata generator
func NewMetaGenerator(cfg *common.Config, c *Client) (MetaGenerator, error) {
	// default settings:
	generator := metaGenerator{
		IncludeCreatorMetadata: true,
		LabelsDedot:            true,
		AnnotationsDedot:       true,
		client:                 c,
	}

	err := cfg.Unpack(&generator)
	return &generator, err
}

// NewMetaGeneratorFromConfig initializes and returns a new kubernetes metadata generator
func NewMetaGeneratorFromConfig(cfg *MetaGeneratorConfig) MetaGenerator {
	return cfg
}

// ResourceMetadata generates metadata for the *given Nomad allocation*
func (g *metaGenerator) ResourceMetadata(obj Resource) common.MapStr {
	tasksMeta := g.groupMeta(obj.Job)
	for idx, task := range tasksMeta {
		labelMap := common.MapStr{}

		if len(g.IncludeLabels) == 0 {
			for k, v := range task {
				if g.LabelsDedot {
					label := common.DeDot(k)
					labelMap.Put(label, v)
				} else {
					safemapstr.Put(labelMap, k, v)
				}
			}
		} else {
			labelMap = generateMapSubset(task, g.IncludeLabels, g.LabelsDedot)
		}

		// Exclude any labels that are present in the exclude_labels config
		for _, label := range g.ExcludeLabels {
			labelMap.Delete(label)
		}

		tasksMeta[idx] = labelMap
	}

	// default labels that we expose / filter with `IncludeLabels`
	meta := common.MapStr{
		"name":        obj.Name,
		"job":         *obj.Job.Name,
		"namespace":   obj.Namespace,
		"datacenters": obj.Job.Datacenters,
		"region":      *obj.Job.Region,
		"type":        *obj.Job.Type,
		"uuid":        obj.ID,
		"tasks":       tasksMeta, // contains metadata for all tasks in the allocation
	}

	return meta
}

// returns a tuple of maps (group/task name to metadata)
func (g *metaGenerator) groupMeta(job *Job) []common.MapStr {
	tasksMeta := []common.MapStr{}

	for _, group := range job.TaskGroups {
		meta := job.Meta
		mergo.Merge(&meta, group.Meta, mergo.WithOverride)
		group.Meta = meta

		tasks := g.tasksMeta(group)
		tasksMeta = append(tasksMeta, tasks...)
	}

	return tasksMeta
}

// returns a map of task name to metadata
func (g *metaGenerator) tasksMeta(group *TaskGroup) []common.MapStr {
	taskMap := common.MapStr{}
	tasks := []common.MapStr{}
	for _, task := range group.Tasks {
		svcMeta := common.MapStr{
			"name":        []string{},
			"tags":        []string{},
			"canary_tags": []string{},
		}

		for _, service := range task.Services {
			svcMeta["name"] = append(svcMeta["name"].([]string), service.Name)
			svcMeta["tags"] = append(svcMeta["name"].([]string), service.Tags...)
			svcMeta["canary_tags"] = append(svcMeta["name"].([]string), service.CanaryTags...)
		}

		joinMeta := group.Meta
		mergo.Merge(&joinMeta, task.Meta, mergo.WithOverride)

		meta := common.MapStr{
			"name":    task.Name,
			"service": svcMeta,
		}

		for k, v := range joinMeta {
			meta.Put(k, v)
		}

		tasks = append(tasks, meta)
	}

	taskMap.Put("tasks", tasks)
	return tasks
}

func generateMapSubset(input common.MapStr, keys []string, dedot bool) common.MapStr {
	output := common.MapStr{}
	if input == nil {
		return output
	}

	for _, key := range keys {
		value, ok := input[key]
		if ok {
			if dedot {
				dedotKey := common.DeDot(key)
				output.Put(dedotKey, value)
			} else {
				safemapstr.Put(output, key, value)
			}
		}
	}

	return output
}

// AllocationNodeName returns Name of the node where the task is allocated. It
// does one additional API call to circunvent the empty NodeName property of
// older Nomad versions
func (g *metaGenerator) AllocationNodeName(id string) (string, error) {
	if name, ok := g.nodesCache[id]; ok {
		return name, nil
	}

	node, _, err := g.client.Nodes().Info(id, nil)
	if err != nil {
		return "", err
	}

	g.nodesCache[id] = node.Name

	return node.Name, nil
}
