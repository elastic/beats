// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/transpiler"
)

func TestMonitoringInjection(t *testing.T) {
	ast, err := transpiler.NewAST(inputConfigMap)
	if err != nil {
		t.Fatal(err)
	}

	programsToRun, err := program.Programs(ast)
	if err != nil {
		t.Fatal(err)
	}

GROUPLOOP:
	for group, ptr := range programsToRun {
		programsCount := len(ptr)
		newPtr, err := injectMonitoring(group, ast, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOP
		}

		if programsCount == len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOP
		}

		for _, p := range newPtr {
			if p.Spec.Name != monitoringName {
				continue
			}

			cm, err := p.Config.Map()
			if err != nil {
				t.Error(err)
				continue GROUPLOOP
			}

			outputCfg, found := cm[outputKey]
			if !found {
				t.Errorf("output not found for '%s'", group)
				continue GROUPLOOP
			}

			outputMap, ok := outputCfg.(map[string]interface{})
			if !ok {
				t.Errorf("output is not a map  for '%s'", group)
				continue GROUPLOOP
			}

			esCfg, found := outputMap["elasticsearch"]
			if !found {
				t.Errorf("elasticsearch output not found for '%s'", group)
				continue GROUPLOOP
			}

			esMap, ok := esCfg.(map[string]interface{})
			if !ok {
				t.Errorf("output.elasticsearch is not a map  for '%s'", group)
				continue GROUPLOOP
			}

			if uname, found := esMap["username"]; !found {
				t.Errorf("output.elasticsearch.username output not found for '%s'", group)
				continue GROUPLOOP
			} else if uname != "monitoring-uname" {
				t.Errorf("output.elasticsearch.username has incorrect value expected '%s', got '%s for %s", "monitoring-uname", uname, group)
				continue GROUPLOOP
			}
		}
	}
}

var inputConfigMap = map[string]interface{}{
	"settings.monitoring": map[string]interface{}{
		"enabled":    true,
		"logs":       true,
		"metrics":    true,
		"use_output": "monitoring",
	},
	"outputs": map[string]interface{}{
		"default": map[string]interface{}{
			"index_name": "general",
			"pass":       "xxx",
			"type":       "elasticsearch",
			"url":        "xxxxx",
			"username":   "xxx",
		},
		"infosec1": map[string]interface{}{
			"pass": "xxx",
			"spool": map[string]interface{}{
				"file": "${path.data}/spool.dat",
			},
			"type":     "elasticsearch",
			"url":      "xxxxx",
			"username": "xxx",
		},
		"monitoring": map[string]interface{}{
			"type":       "elasticsearch",
			"index_name": "general",
			"pass":       "xxx",
			"url":        "xxxxx",
			"username":   "monitoring-uname",
		},
	},
	"datasources": []map[string]interface{}{
		map[string]interface{}{
			"inputs": []map[string]interface{}{
				map[string]interface{}{
					"type": "log",
					"streams": []map[string]interface{}{
						map[string]interface{}{"paths": "/xxxx"},
					},
					"processors": []interface{}{
						map[string]interface{}{
							"dissect": map[string]interface{}{
								"tokenizer": "---",
							},
						},
					},
				},
			},
		},
		map[string]interface{}{
			"inputs": []map[string]interface{}{
				map[string]interface{}{
					"type": "metrics/system",
				},
			},
			"use_output": "infosec1",
		},
	},
}

// const inputConfig = `outputs:
//   default:
//     index_name: general
//     pass: xxx
//     type: es
//     url: xxxxx
//     username: xxx
//   infosec1:
//     pass: xxx
//     spool:
//       file: "${path.data}/spool.dat"
//     type: es
//     url: xxxxx
//     username: xxx
// streams:
//   -
//     output:
//       override:
//         index_name: my_service_logs
//         ingest_pipeline: process_logs
//     path: /xxxx
//     processors:
//       -
//         dissect:
//           tokenizer: "---"
//     type: log
//   -
//     output:
//       index_name: mysql_access_logs
//     path: /xxxx
//     type: log
//   -
//     output:
//       index_name: mysql_metrics
//       use_output: infosec1
//     pass: yyy
//     type: metrics/system
//     username: xxxx
//   `
