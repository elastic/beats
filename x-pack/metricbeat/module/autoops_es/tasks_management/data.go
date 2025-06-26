// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tasks_management

import (
	"errors"
	"fmt"

	e "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"golang.org/x/exp/maps"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const TASK_RUNTIME_THRESHOLD_IN_SECONDS_NAME = "TASK_RUNTIME_THRESHOLD_IN_SECONDS"

var taskSchema = s.Schema{
	"id":                 c.Int("id", s.Required),
	"node":               c.Str("node", s.Required),
	"taskType":           c.Str("type", s.Required),
	"action":             c.Str("action", s.Required),
	"startTimeInMillis":  c.Int("start_time_in_millis", s.Required),
	"runningTimeInNanos": c.Int("running_time_in_nanos", s.Required),
	"description":        c.Str("description", s.Optional),
	"cancellable":        c.Bool("cancellable", s.Required),
	"headers":            c.Ifc("headers", s.IgnoreAllErrors),
	"children":           c.Ifc("children", s.IgnoreAllErrors),
}

type GroupedTasks struct {
	Tasks map[string]map[string]interface{} `json:"tasks"`
}

// Get the value from the environment for the TASK_RUNTIME_THRESHOLD_IN_SECONDS_NAME field.
func getTaskRuntimeThresholdInSeconds() int64 {
	return int64(utils.GetIntEnvParam(TASK_RUNTIME_THRESHOLD_IN_SECONDS_NAME, 60))
}

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, nodeTasks *GroupedTasks) error {
	var errs []error
	var tasks []mapstr.M

	taskRuntimeThresholdInSeconds := getTaskRuntimeThresholdInSeconds()

	for taskId, taskData := range nodeTasks.Tasks {
		task, err := taskSchema.Apply(taskData)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed applying task schema for %v: %w", taskId, err))
			continue
		}

		// the schema validated that the value exists as an integer
		runningTimeInNanos, _ := task.GetValue("runningTimeInNanos")

		// skip anything not running long enough
		if nanos, ok := runningTimeInNanos.(int64); !ok || nanos/1000000000 < taskRuntimeThresholdInSeconds {
			continue
		}

		// the schema validated that the value exists as a string
		nodeValue, _ := task.GetValue("node")
		node, ok := nodeValue.(string)

		if !ok {
			continue
		}

		// child tasks are optional
		children, ok := task["children"]

		if !ok || children == nil {
			task["node"] = [1]string{node}
		} else {
			innerChildren, ok := children.([]any)

			nodeMap := parseChildNodes(innerChildren, ok)
			// guarantee the parent node is in the list (it may not be)
			nodeMap[node] = true

			task["node"] = maps.Keys(nodeMap)

			// remove the children node
			delete(task, "children")
		}

		// note: the task ID is not a part of the payload, so we have to add it
		task["taskId"] = taskId

		tasks = append(tasks, mapstr.M{"task": task})
	}

	transactionId := utils.NewUUIDV4()
	e.CreateAndReportEvents(r, info, tasks, transactionId)

	err := errors.Join(errs...)

	if err != nil {
		e.SendErrorEvent(err, info, r, TasksMetricSet, TasksPath, transactionId)
	}

	return err
}
