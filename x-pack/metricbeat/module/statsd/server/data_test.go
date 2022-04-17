// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/metricbeat/helper/server"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func init() {
	mb.Registry.SetSecondarySource(mb.NewLightModulesSource("../../../module"))
}

func TestEventMapping(t *testing.T) {
	mappingsYml := `
      - metric: '<job_name>_start'
        labels:
          - attr: job_name
            field: job_name
        value:
          field: started
      - metric: '<job_name>_end'
        labels:
          - attr: job_name
            field: job_name
        value:
          field: ended
      - metric: <job_name>_heartbeat_failure
        labels:
          - attr: job_name
            field: job_name
        value:
          field: heartbeat_failure
      - metric: 'operator_failures_<operator_name>'
        labels:
          - attr: operator_name
            field: operator_name
        value:
          field: failures
      - metric: 'operator_successes_<operator_name>'
        labels:
          - attr: operator_name
            field: operator_name
        value:
          field: successes
      - metric: 'ti_failures'
        value:
          field: task_failures
      - metric: 'ti_successes'
        value:
          field: task_successes
      - metric: 'previously_succeeded'
        value:
          field: previously_succeeded
      - metric: 'zombies_killed'
        value:
          field: zombies_killed
      - metric: 'scheduler_heartbeat'
        value:
          field: scheduler_heartbeat
      - metric: 'dag_processing.manager_stalls'
        value:
          field: dag_file_processor_manager_stalls
      - metric: 'dag_file_refresh_error'
        value:
          field: dag_file_refresh_error
      - metric: 'dag_processing.processes'
        value:
          field: dag_processes
      - metric: 'scheduler.tasks.killed_externally'
        value:
          field: task_killed_externally
      - metric: 'scheduler.tasks.running'
        value:
          field: task_running
      - metric: 'scheduler.tasks.starving'
        value:
          field: task_starving
      - metric: 'scheduler.orphaned_tasks.cleared'
        value:
          field: task_orphaned_cleared
      - metric: 'scheduler.orphaned_tasks.adopted'
        value:
          field: task_orphaned_adopted
      - metric: 'scheduler.critical_section_busy'
        value:
          field: scheduler_critical_section_busy
      - metric: 'sla_email_notification_failure'
        value:
          field: sla_email_notification_failure
      - metric: 'ti.start.<dagid>.<taskid>'
        labels:
          - attr: dagid
            field: dag_id
          - attr: taskid
            field: task_id
        value:
          field: task_started
      - metric: 'ti.finish.<dagid>.<taskid>.<status>'
        labels:
          - attr: dagid
            field: dag_id
          - attr: taskid
            field: task_id
          - attr: status
            field: status
        value:
          field: task_finished
      - metric: 'dag.callback_exceptions'
        value:
          field: dag_callback_exceptions
      - metric: 'celery.task_timeout_error'
        value:
          field: task_celery_timeout_error
      - metric: 'task_removed_from_dag.<dagid>'
        labels:
          - attr: dagid
            field: dag_id
        value:
          field: task_removed
      - metric: 'task_restored_to_dag.<dagid>'
        labels:
          - attr: dagid
            field: dag_id
        value:
          field: task_restored
      - metric: 'task_instance_created-<operator_name>'
        labels:
          - attr: operator_name
            field: operator_name
        value:
          field: task_created
      - metric: 'dagbag_size'
        value:
          field: dag_bag_size
      - metric: 'dag_processing.import_errors'
        value:
          field: dag_import_errors
      - metric: 'dag_processing.total_parse_time'
        value:
          field: dag_total_parse_time
      - metric: 'dag_processing.last_runtime.<dag_file>'
        labels:
          - attr: dag_file
            field: dag_file
        value:
          field: dag_last_runtime
      - metric: 'dag_processing.last_run.seconds_ago.<dag_file>'
        labels:
          - attr: dag_file
            field: dag_file
        value:
          field: dag_last_run_seconds_ago
      - metric: 'dag_processing.processor_timeouts'
        value:
          field: processor_timeouts
      - metric: 'scheduler.tasks.without_dagrun'
        value:
          field: task_without_dagrun
      - metric: 'scheduler.tasks.running'
        value:
          field: task_running
      - metric: 'scheduler.tasks.starving'
        value:
          field: task_starving
      - metric: 'scheduler.tasks.executable'
        value:
          field: task_executable
      - metric: 'executor.open_slots'
        value:
          field: executor_open_slots
      - metric: 'executor.queued_tasks'
        value:
          field: executor_queued_tasks
      - metric: 'executor.running_tasks'
        value:
          field: executor_running_tasks
      - metric: 'pool.open_slots.<pool_name>'
        labels:
          - attr: pool_name
            field: pool_name
        value:
          field: pool_open_slots
      - metric: 'pool.queued_slots.<pool_name>'
        labels:
          - attr: pool_name
            field: pool_name
        value:
          field: pool_queued_slots
      - metric: 'pool.running_slots.<pool_name>'
        labels:
          - attr: pool_name
            field: pool_name
        value:
          field: pool_running_slots
      - metric: 'pool.starving_tasks.<pool_name>'
        labels:
          - attr: pool_name
            field: pool_name
        value:
          field: pool_starving_tasks
      - metric: 'smart_sensor_operator.poked_tasks'
        value:
          field: smart_sensor_operator_poked_tasks
      - metric: 'smart_sensor_operator.poked_success'
        value:
          field: smart_sensor_operator_poked_success
      - metric: 'smart_sensor_operator.poked_exception'
        value:
          field: smart_sensor_operator_poked_exception
      - metric: 'smart_sensor_operator.exception_failures'
        value:
          field: smart_sensor_operator_exception_failures
      - metric: 'smart_sensor_operator.infra_failures'
        value:
          field: smart_sensor_operator_infra_failures
      - metric: 'dagrun.dependency-check.<dag_id>'
        labels:
          - attr: dag_id
            field: dag_id
        value:
          field: dag_dependency_check
      - metric: 'dag.<dag_id>.<task_id>.duration'
        labels:
          - attr: dag_id
            field: dag_id
          - attr: task_id
            field: task_id
        value:
          field: task_duration
      - metric: 'dag_processing.last_duration.<dag_file>'
        labels:
          - attr: dag_file
            field: dag_file
        value:
          field: dag_last_duration
      - metric: 'dagrun.duration.success.<dag_id>'
        labels:
          - attr: dag_id
            field: dag_id
        value:
          field: success_dag_duration
      - metric: 'dagrun.duration.failed.<dag_id>'
        labels:
          - attr: dag_id
            field: dag_id
        value:
          field: failed_dag_duration
      - metric: 'dagrun.schedule_delay.<dag_id>'
        labels:
          - attr: dag_id
            field: dag_id
        value:
          field: dag_schedule_delay
      - metric: 'scheduler.critical_section_duration'
        value:
          field: scheduler_critical_section_duration
      - metric: 'dagrun.<dag_id>.first_task_scheduling_delay'
        labels:
          - attr: dag_id
            field: dag_id
        value:
          field: dag_first_task_scheduling_delay
    `

	var mappings []StatsdMapping
	_ = yaml.Unmarshal([]byte(mappingsYml), &mappings)

	countValue := map[string]interface{}{"count": 4}
	timerValue := map[string]interface{}{
		"stddev":    0,
		"p99_9":     100,
		"mean_rate": 0.2689038235718098,
		"max":       100,
		"mean":      100,
		"p95":       100,
		"min":       100,
		"median":    100,
		"p75":       100,
		"p99":       100,
		"5m_rate":   0.2,
		"count":     1,
		"1m_rate":   0.2,
		"15m_rate":  0.2,
	}

	gaugeValue := map[string]interface{}{"value": 2}

	for _, test := range []struct {
		metricName  string
		metricValue interface{}
		expected    common.MapStr
	}{
		{
			metricName:  "a_job_name_start",
			metricValue: countValue,
			expected: common.MapStr{
				"job_name": "a_job_name",
				"started":  countValue,
			},
		},
		{
			metricName:  "a_job_name_end",
			metricValue: countValue,
			expected: common.MapStr{
				"job_name": "a_job_name",
				"ended":    countValue,
			},
		},
		{
			metricName:  "a_job_name_heartbeat_failure",
			metricValue: countValue,
			expected: common.MapStr{
				"job_name":          "a_job_name",
				"heartbeat_failure": countValue,
			},
		},
		{
			metricName:  "operator_failures_an_operator_name",
			metricValue: countValue,
			expected: common.MapStr{
				"operator_name": "an_operator_name",
				"failures":      countValue,
			},
		},
		{
			metricName:  "operator_successes_an_operator_name",
			metricValue: countValue,
			expected: common.MapStr{
				"operator_name": "an_operator_name",
				"successes":     countValue,
			},
		},
		{
			metricName:  "ti_failures",
			metricValue: countValue,
			expected: common.MapStr{
				"task_failures": countValue,
			},
		},
		{
			metricName:  "ti_successes",
			metricValue: countValue,
			expected: common.MapStr{
				"task_successes": countValue,
			},
		},
		{
			metricName:  "previously_succeeded",
			metricValue: countValue,
			expected: common.MapStr{
				"previously_succeeded": countValue,
			},
		},
		{
			metricName:  "zombies_killed",
			metricValue: countValue,
			expected: common.MapStr{
				"zombies_killed": countValue,
			},
		},
		{
			metricName:  "scheduler_heartbeat",
			metricValue: countValue,
			expected: common.MapStr{
				"scheduler_heartbeat": countValue,
			},
		},
		{
			metricName:  "dag_processing.processes",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_processes": countValue,
			},
		},
		{
			metricName:  "dag_processing.manager_stalls",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_file_processor_manager_stalls": countValue,
			},
		},
		{
			metricName:  "dag_file_refresh_error",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_file_refresh_error": countValue,
			},
		},
		{
			metricName:  "scheduler.tasks.killed_externally",
			metricValue: countValue,
			expected: common.MapStr{
				"task_killed_externally": countValue,
			},
		},
		{
			metricName:  "scheduler.tasks.running",
			metricValue: countValue,
			expected: common.MapStr{
				"task_running": countValue,
			},
		},
		{
			metricName:  "scheduler.tasks.starving",
			metricValue: countValue,
			expected: common.MapStr{
				"task_starving": countValue,
			},
		},
		{
			metricName:  "scheduler.orphaned_tasks.cleared",
			metricValue: countValue,
			expected: common.MapStr{
				"task_orphaned_cleared": countValue,
			},
		},
		{
			metricName:  "scheduler.orphaned_tasks.adopted",
			metricValue: countValue,
			expected: common.MapStr{
				"task_orphaned_adopted": countValue,
			},
		},
		{
			metricName:  "scheduler.critical_section_busy",
			metricValue: countValue,
			expected: common.MapStr{
				"scheduler_critical_section_busy": countValue,
			},
		},
		{
			metricName:  "sla_email_notification_failure",
			metricValue: countValue,
			expected: common.MapStr{
				"sla_email_notification_failure": countValue,
			},
		},
		{
			metricName:  "ti.start.a_dagid.a_taskid",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_id":       "a_dagid",
				"task_id":      "a_taskid",
				"task_started": countValue,
			},
		},
		{
			metricName:  "ti.finish.a_dagid.a_taskid.a_status",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_id":        "a_dagid",
				"task_id":       "a_taskid",
				"status":        "a_status",
				"task_finished": countValue,
			},
		},
		{
			metricName:  "dag.callback_exceptions",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_callback_exceptions": countValue,
			},
		},
		{
			metricName:  "celery.task_timeout_error",
			metricValue: countValue,
			expected: common.MapStr{
				"task_celery_timeout_error": countValue,
			},
		},
		{
			metricName:  "task_removed_from_dag.a_dagid",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_id":       "a_dagid",
				"task_removed": countValue,
			},
		},
		{
			metricName:  "task_restored_to_dag.a_dagid",
			metricValue: countValue,
			expected: common.MapStr{
				"dag_id":        "a_dagid",
				"task_restored": countValue,
			},
		},
		{
			metricName:  "task_instance_created-an_operator_name",
			metricValue: countValue,
			expected: common.MapStr{
				"operator_name": "an_operator_name",
				"task_created":  countValue,
			},
		},
		{
			metricName:  "dagbag_size",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"dag_bag_size": gaugeValue,
			},
		},
		{
			metricName:  "dag_processing.import_errors",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"dag_import_errors": gaugeValue,
			},
		},
		{
			metricName:  "dag_processing.total_parse_time",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"dag_total_parse_time": gaugeValue,
			},
		},
		{
			metricName:  "dag_processing.last_runtime.a_dag_file",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"dag_file":         "a_dag_file",
				"dag_last_runtime": gaugeValue,
			},
		},
		{
			metricName:  "dag_processing.last_run.seconds_ago.a_dag_file",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"dag_file":                 "a_dag_file",
				"dag_last_run_seconds_ago": gaugeValue,
			},
		},
		{
			metricName:  "dag_processing.processor_timeouts",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"processor_timeouts": gaugeValue,
			},
		},
		{
			metricName:  "scheduler.tasks.without_dagrun",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"task_without_dagrun": gaugeValue,
			},
		},
		{
			metricName:  "scheduler.tasks.running",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"task_running": gaugeValue,
			},
		},
		{
			metricName:  "scheduler.tasks.starving",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"task_starving": gaugeValue,
			},
		},
		{
			metricName:  "scheduler.tasks.executable",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"task_executable": gaugeValue,
			},
		},
		{
			metricName:  "executor.open_slots",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"executor_open_slots": gaugeValue,
			},
		},
		{
			metricName:  "executor.queued_tasks",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"executor_queued_tasks": gaugeValue,
			},
		},
		{
			metricName:  "executor.running_tasks",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"executor_running_tasks": gaugeValue,
			},
		},
		{
			metricName:  "pool.open_slots.a_pool_name",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"pool_name":       "a_pool_name",
				"pool_open_slots": gaugeValue,
			},
		},
		{
			metricName:  "pool.queued_slots.a_pool_name",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"pool_name":         "a_pool_name",
				"pool_queued_slots": gaugeValue,
			},
		},
		{
			metricName:  "pool.running_slots.a_pool_name",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"pool_name":          "a_pool_name",
				"pool_running_slots": gaugeValue,
			},
		},
		{
			metricName:  "pool.starving_tasks.a_pool_name",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"pool_name":           "a_pool_name",
				"pool_starving_tasks": gaugeValue,
			},
		},
		{
			metricName:  "smart_sensor_operator.poked_tasks",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"smart_sensor_operator_poked_tasks": gaugeValue,
			},
		},
		{
			metricName:  "smart_sensor_operator.poked_success",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"smart_sensor_operator_poked_success": gaugeValue,
			},
		},
		{
			metricName:  "smart_sensor_operator.poked_exception",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"smart_sensor_operator_poked_exception": gaugeValue,
			},
		},
		{
			metricName:  "smart_sensor_operator.exception_failures",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"smart_sensor_operator_exception_failures": gaugeValue,
			},
		},
		{
			metricName:  "smart_sensor_operator.infra_failures",
			metricValue: gaugeValue,
			expected: common.MapStr{
				"smart_sensor_operator_infra_failures": gaugeValue,
			},
		},
		{
			metricName:  "dagrun.dependency-check.a_dag_id",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":               "a_dag_id",
				"dag_dependency_check": timerValue,
			},
		},
		{
			metricName:  "dag.a_dag_id.a_task_id.duration",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":        "a_dag_id",
				"task_id":       "a_task_id",
				"task_duration": timerValue,
			},
		},
		{
			metricName:  "dag_processing.last_duration.a_dag_file",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_file":          "a_dag_file",
				"dag_last_duration": timerValue,
			},
		},
		{
			metricName:  "dagrun.duration.success.a_dag_id",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":               "a_dag_id",
				"success_dag_duration": timerValue,
			},
		},
		{
			metricName:  "dagrun.duration.failed.a_dag_id",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":              "a_dag_id",
				"failed_dag_duration": timerValue,
			},
		},
		{
			metricName:  "dagrun.schedule_delay.a_dag_id",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":             "a_dag_id",
				"dag_schedule_delay": timerValue,
			},
		},
		{
			metricName:  "scheduler.critical_section_duration",
			metricValue: timerValue,
			expected: common.MapStr{
				"scheduler_critical_section_duration": timerValue,
			},
		},
		{
			metricName:  "dagrun.a_dag_id.first_task_scheduling_delay",
			metricValue: timerValue,
			expected: common.MapStr{
				"dag_id":                          "a_dag_id",
				"dag_first_task_scheduling_delay": timerValue,
			},
		},
		{
			metricName:  "not_mapped_metric",
			metricValue: timerValue,
			expected:    common.MapStr{},
		},
	} {
		t.Run(test.metricName, func(t *testing.T) {
			metricSetFields := common.MapStr{}
			builtMappings, _ := buildMappings(mappings)
			eventMapping(test.metricName, test.metricValue, metricSetFields, builtMappings)

			assert.Equal(t, test.expected, metricSetFields)
		})
	}
}

func TestBuildMappings(t *testing.T) {
	for _, test := range []struct {
		title    string
		input    string
		err      error
		expected map[string]StatsdMapping
	}{
		{
			title: "no err",
			input: `
      - metric: '<job_name>_start'
        labels:
          - attr: job_name
            field: job_name
        value:
          field: started
`,
			err: nil,
			expected: map[string]StatsdMapping{
				"<job_name>_start": {
					Metric: "<job_name>_start",
					Labels: []Label{
						{Attr: "job_name", Field: "job_name"},
					},
					Value: Value{Field: "started"},
				},
			},
		},
		{
			title: "not matching label",
			input: `
      - metric: '<job_name>_start'
        labels:
          - attr: not_matching
            field: job_name
        value:
          field: started
`,
			err: errInvalidMapping{
				metricLabels: []string{"job_name"},
				attrLabels:   []string{"not_matching"},
			},
			expected: nil,
		},
		{
			title: "not existing label",
			input: `
      - metric: '<job_name>_start'
        labels:
          - attr: job_name
            field: job_name
          - attr: not_existing
            field: not_existing
        value:
          field: started
`,
			err: errInvalidMapping{
				metricLabels: []string{"job_name"},
				attrLabels:   []string{"job_name", "not_existing"},
			},
			expected: nil,
		},
		{
			title: "repeated label",
			input: `
      - metric: '<job_name>_<dagid>_start'
        labels:
          - attr: job_name
            field: repeated_label_field
          - attr: job_name
            field: repeated_label_field
        value:
          field: started
`,
			err:      fmt.Errorf(`repeated label fields "repeated_label_field"`),
			expected: nil,
		},
		{
			title: "colliding field",
			input: `
      - metric: '<job_name>_start'
        labels:
          - attr: job_name
            field: colliding_field
        value:
          field: colliding_field
`,
			err:      fmt.Errorf(`collision between label field "colliding_field" and value field "colliding_field"`),
			expected: nil,
		},
	} {
		t.Run(test.title, func(t *testing.T) {
			var mappings []StatsdMapping
			err := yaml.Unmarshal([]byte(test.input), &mappings)
			actual, err := buildMappings(mappings)
			for k, v := range actual {
				v.regex = nil
				actual[k] = v
			}
			assert.Equal(t, test.err, err, test.input)
			assert.Equal(t, test.expected, actual, test.input)
		})
	}
}

func TestParseMetrics(t *testing.T) {
	for _, test := range []struct {
		input    string
		err      error
		expected []statsdMetric
	}{
		{
			input: "gauge1:1.0|g",
			expected: []statsdMetric{{
				name:       "gauge1",
				metricType: "g",
				value:      "1.0",
			}},
		},
		{
			input: "counter1:11|c",
			expected: []statsdMetric{{
				name:       "counter1",
				metricType: "c",
				value:      "11",
			}},
		},
		{
			input: "counter2:15|c|@0.1",
			expected: []statsdMetric{{
				name:       "counter2",
				metricType: "c",
				value:      "15",
				sampleRate: "0.1",
			}},
		},
		{
			input: "decrement-counter:-15|c",
			expected: []statsdMetric{{
				name:       "decrement-counter",
				metricType: "c",
				value:      "-15",
			}},
		},
		{
			input: "timer1:1.2|ms",
			expected: []statsdMetric{{
				name:       "timer1",
				metricType: "ms",
				value:      "1.2",
			}},
		},
		{
			input: "histogram1:3|h",
			expected: []statsdMetric{{
				name:       "histogram1",
				metricType: "h",
				value:      "3",
			}},
		},
		{
			input: "meter1:1.4|m",
			expected: []statsdMetric{{
				name:       "meter1",
				metricType: "m",
				value:      "1.4",
			}},
		},
		{
			input: "set1:hello|s",
			expected: []statsdMetric{{
				name:       "set1",
				metricType: "s",
				value:      "hello",
			}},
		},
		{
			input: "lf-ended-meter1:1.5|m\n",
			expected: []statsdMetric{{
				name:       "lf-ended-meter1",
				metricType: "m",
				value:      "1.5",
			}},
		},
		{
			input: "counter2.1:1|c|@0.01\ncounter2.2:2|c|@0.01",
			expected: []statsdMetric{
				{
					name:       "counter2.1",
					metricType: "c",
					value:      "1",
					sampleRate: "0.01",
				},
				{
					name:       "counter2.2",
					metricType: "c",
					value:      "2",
					sampleRate: "0.01",
				},
			},
		},
		/// tags
		{
			input: "tags1:1|c|#k1:v1,k2:v2",
			expected: []statsdMetric{
				{
					name:       "tags1",
					metricType: "c",
					value:      "1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		{
			input: "tags2:2|m|@0.1|#k1:v1,k2:v2",
			expected: []statsdMetric{
				{
					name:       "tags2",
					metricType: "m",
					value:      "2",
					sampleRate: "0.1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		{ // Influx Statsd tags
			input: "tags2,k1=v1,k2=v2:1|c",
			expected: []statsdMetric{
				{
					name:       "tags2",
					metricType: "c",
					value:      "1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		/// errors
		{
			input:    "meter1-1.4|m",
			expected: []statsdMetric{},
			err:      errInvalidPacket,
		},
		{
			input:    "meter1:1.4-m",
			expected: []statsdMetric{},
			err:      errInvalidPacket,
		},
	} {
		actual, err := parse([]byte(test.input))
		assert.Equal(t, test.err, err, test.input)
		assert.Equal(t, test.expected, actual, test.input)

		processor := newMetricProcessor(time.Second)
		for _, e := range actual {
			err := processor.processSingle(e)

			assert.NoError(t, err)
		}
	}
}

type testUDPEvent struct {
	event common.MapStr
	meta  server.Meta
}

func (u *testUDPEvent) GetEvent() common.MapStr {
	return u.event
}

func (u *testUDPEvent) GetMeta() server.Meta {
	return u.meta
}

func process(packets []string, ms *MetricSet) error {
	for _, d := range packets {
		udpEvent := &testUDPEvent{
			event: common.MapStr{
				server.EventDataKey: []byte(d),
			},
			meta: server.Meta{
				"client_ip": "127.0.0.1",
			},
		}
		if err := ms.processor.Process(udpEvent); err != nil {
			return err
		}
	}
	return nil
}

func TestTagsGrouping(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric1:1.0|g|#k1:v1,k2:v2",
		"metric2:2|c|#k1:v1,k2:v2",

		"metric3:3|c|@0.1|#k1:v2,k2:v3",
		"metric4:4|ms|#k1:v2,k2:v3",
	}

	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 2)

	actualTags := []common.MapStr{}
	for _, e := range events {
		actualTags = append(actualTags, e.RootFields)
	}

	expectedTags := []common.MapStr{
		common.MapStr{
			"labels": common.MapStr{
				"k1": "v1",
				"k2": "v2",
			},
		},
		common.MapStr{
			"labels": common.MapStr{
				"k1": "v2",
				"k2": "v3",
			},
		},
	}

	assert.ElementsMatch(t, expectedTags, actualTags)
}

func TestTagsCleanup(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd", "ttl": "1s"}).(*MetricSet)
	testData := []string{
		"metric1:1|g|#k1:v1,k2:v2",

		"metric2:3|ms|#k1:v2,k2:v3",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	// they will be reported at least once
	assert.Len(t, ms.getEvents(), 2)

	testData = []string{
		"metric1:+2|g|#k1:v1,k2:v2",
	}
	// refresh metrics1
	err = process(testData, ms)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// metrics2 should be out now
	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{"metric1": map[string]interface{}{"value": float64(3)}})
}

func TestSetReset(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd", "ttl": "1s"}).(*MetricSet)
	testData := []string{
		"metric1:hello|s|#k1:v1,k2:v2",
		"metric1:again|s|#k1:v1,k2:v2",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	require.Len(t, events, 1)

	assert.Equal(t, 2, events[0].MetricSetFields["metric1"].(map[string]interface{})["count"])

	events = ms.getEvents()
	assert.Equal(t, 0, events[0].MetricSetFields["metric1"].(map[string]interface{})["count"])
}

func TestData(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1.0|g|#k1:v1,k2:v2",
		"metric02:2|c|#k1:v1,k2:v2",
		"metric03:3|c|@0.1|#k1:v1,k2:v2",
		"metric04:4|ms|#k1:v1,k2:v2",
		"metric05:5|h|#k1:v1,k2:v2",
		"metric06:6|h|#k1:v1,k2:v2",
		"metric07:7|ms|#k1:v1,k2:v2",
		"metric08:seven|s|#k1:v1,k2:v2",
		"metric09,k1=v1,k2=v2:8|h",
		"metric10.with.dots,k1=v1,k2=v2:9|h",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	mbevent := mbtest.StandardizeEvent(ms, *events[0])
	mbtest.WriteEventToDataJSON(t, mbevent, "")
}

func TestGaugeDeltas(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1.0|g|#k1:v1,k2:v2",
		"metric01:-2.0|g|#k1:v1,k2:v2",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{
		"metric01": map[string]interface{}{"value": -1.0},
	})

	// same value reported again
	events = ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{
		"metric01": map[string]interface{}{"value": -1.0},
	})
}
func TestCounter(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1|c|#k1:v1,k2:v2",
		"metric01:2|c|#k1:v1,k2:v2",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{
		"metric01": map[string]interface{}{"count": int64(3)},
	})

	// reset
	events = ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{
		"metric01": map[string]interface{}{"count": int64(0)},
	})
}

func TestCounterSampled(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1|c|@0.1",
		"metric01:2|c|@0.2",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{
		"metric01": map[string]interface{}{"count": int64(20)},
	})
}

func TestCounterSampledZero(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1|c|@0.0",
	}
	err := process(testData, ms)
	assert.Error(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 0)
}

func TestTimerSampled(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:2|ms|@0.01",
		"metric01:1|ms|@0.1",
		"metric01:2|ms|@0.2",
		"metric01:2|ms",
	}

	// total of 100 + 10 + 5 + 1 = 116 measurements
	err := process(testData, ms)
	require.NoError(t, err)

	// rate gorutine runs every 5 sec
	time.Sleep(time.Second * 6)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	actualMetric01 := events[0].MetricSetFields["metric01"].(map[string]interface{})

	// returns the extrapolated count
	assert.Equal(t, int64(116), actualMetric01["count"])

	// rate numbers are updated by a gorutine periodically, so we cant tell exactly what they should be here
	// we just need to check that the sample rate was applied
	assert.True(t, actualMetric01["1m_rate"].(float64) > 10)
	assert.True(t, actualMetric01["5m_rate"].(float64) > 10)
	assert.True(t, actualMetric01["15m_rate"].(float64) > 10)
}

func TestChangeType(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric01:1|ms",
		"metric01:2|c",
	}
	err := process(testData, ms)
	require.NoError(t, err)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, common.MapStr{
		"metric01": map[string]interface{}{"count": int64(2)},
	}, events[0].MetricSetFields)
}

func BenchmarkIngest(b *testing.B) {
	tests := []string{
		"metric01:1.0|g|#k1:v1,k2:v2",
		"metric02:2|c|#k1:v1,k2:v2",
		"metric03:3|c|@0.1|#k1:v1,k2:v2",
		"metric04:4|ms|#k1:v1,k2:v2",
		"metric05:5|h|#k1:v1,k2:v2",
		"metric06:6|h|#k1:v1,k2:v2",
		"metric07:7|ms|#k1:v1,k2:v2",
		"metric08:seven|s|#k1:v1,k2:v2",
		"metric09,k1=v1,k2=v2:8|h",
		"metric10.with.dots,k1=v1,k2=v2:9|h",
	}

	events := make([]*testUDPEvent, len(tests))
	for i, d := range tests {
		events[i] = &testUDPEvent{
			event: common.MapStr{
				server.EventDataKey: []byte(d),
			},
			meta: server.Meta{
				"client_ip": "127.0.0.1",
			},
		}
	}
	ms := mbtest.NewMetricSet(b, map[string]interface{}{"module": "statsd"}).(*MetricSet)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ms.processor.Process(events[i%len(events)])
		assert.NoError(b, err)
	}

}
