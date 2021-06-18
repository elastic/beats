// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package modifiers

import (
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func TestMonitoringInjection(t *testing.T) {
	agentInfo, err := info.NewAgentInfo(true)
	if err != nil {
		t.Fatal(err)
	}
	ast, err := transpiler.NewAST(inputConfigMap)
	if err != nil {
		t.Fatal(err)
	}

	programsToRun, err := program.Programs(agentInfo, ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRun) != 1 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 1))
	}

GROUPLOOP:
	for group, ptr := range programsToRun {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, ast, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOP
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOP
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
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

func TestMonitoringInjectionDefaults(t *testing.T) {
	agentInfo, err := info.NewAgentInfo(true)
	if err != nil {
		t.Fatal(err)
	}
	ast, err := transpiler.NewAST(inputConfigMapDefaults)
	if err != nil {
		t.Fatal(err)
	}

	programsToRun, err := program.Programs(agentInfo, ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRun) != 1 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 1))
	}

GROUPLOOP:
	for group, ptr := range programsToRun {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, ast, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOP
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOP
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
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
			} else if uname != "xxx" {
				t.Errorf("output.elasticsearch.username has incorrect value expected '%s', got '%s for %s", "monitoring-uname", uname, group)
				continue GROUPLOOP
			}
		}
	}
}

func TestMonitoringToLogstashInjection(t *testing.T) {
	agentInfo, err := info.NewAgentInfo(true)
	if err != nil {
		t.Fatal(err)
	}
	ast, err := transpiler.NewAST(inputConfigLS)
	if err != nil {
		t.Fatal(err)
	}

	programsToRun, err := program.Programs(agentInfo, ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRun) != 1 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 1))
	}

GROUPLOOP:
	for group, ptr := range programsToRun {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, ast, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOP
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOP
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
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

			esCfg, found := outputMap["logstash"]
			if !found {
				t.Errorf("logstash output not found for '%s' %v", group, outputMap)
				continue GROUPLOOP
			}

			esMap, ok := esCfg.(map[string]interface{})
			if !ok {
				t.Errorf("output.logstash is not a map  for '%s'", group)
				continue GROUPLOOP
			}

			if uname, found := esMap["hosts"]; !found {
				t.Errorf("output.logstash.hosts output not found for '%s'", group)
				continue GROUPLOOP
			} else if uname != "192.168.1.2" {
				t.Errorf("output.logstash.hosts has incorrect value expected '%s', got '%s for %s", "monitoring-uname", uname, group)
				continue GROUPLOOP
			}
		}
	}
}

func TestMonitoringInjectionDisabled(t *testing.T) {
	agentInfo, err := info.NewAgentInfo(true)
	if err != nil {
		t.Fatal(err)
	}
	ast, err := transpiler.NewAST(inputConfigMapDisabled)
	if err != nil {
		t.Fatal(err)
	}

	programsToRun, err := program.Programs(agentInfo, ast)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRun) != 2 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 2))
	}

GROUPLOOP:
	for group, ptr := range programsToRun {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, ast, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOP
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOP
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
				continue
			}

			cm, err := p.Config.Map()
			if err != nil {
				t.Error(err)
				continue GROUPLOOP
			}

			// is enabled set
			agentObj, found := cm["agent"]
			if !found {
				t.Errorf("settings not found for '%s(%s)': %v", group, p.Spec.Name, cm)
				continue GROUPLOOP
			}

			agentMap, ok := agentObj.(map[string]interface{})
			if !ok {
				t.Errorf("settings not a map for '%s(%s)': %v", group, p.Spec.Name, cm)
				continue GROUPLOOP
			}

			monitoringObj, found := agentMap["monitoring"]
			if !found {
				t.Errorf("agent.monitoring not found for '%s(%s)': %v", group, p.Spec.Name, cm)
				continue GROUPLOOP
			}

			monitoringMap, ok := monitoringObj.(map[string]interface{})
			if !ok {
				t.Errorf("agent.monitoring not a map for '%s(%s)': %v", group, p.Spec.Name, cm)
				continue GROUPLOOP
			}

			enabledVal, found := monitoringMap["enabled"]
			if !found {
				t.Errorf("monitoring.enabled not found for '%s(%s)': %v", group, p.Spec.Name, cm)
				continue GROUPLOOP
			}

			monitoringEnabled, ok := enabledVal.(bool)
			if !ok {
				t.Errorf("agent.monitoring.enabled is not a bool for '%s'", group)
				continue GROUPLOOP
			}

			if monitoringEnabled {
				t.Errorf("agent.monitoring.enabled is enabled, should be disabled for '%s'", group)
				continue GROUPLOOP
			}
		}
	}
}

func TestChangeInMonitoringWithChangeInInput(t *testing.T) {
	agentInfo, err := info.NewAgentInfo(true)
	if err != nil {
		t.Fatal(err)
	}

	astBefore, err := transpiler.NewAST(inputChange1)
	if err != nil {
		t.Fatal(err)
	}

	programsToRunBefore, err := program.Programs(agentInfo, astBefore)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRunBefore) != 1 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 1))
	}

	astAfter, err := transpiler.NewAST(inputChange2)
	if err != nil {
		t.Fatal(err)
	}

	programsToRunAfter, err := program.Programs(agentInfo, astAfter)
	if err != nil {
		t.Fatal(err)
	}

	if len(programsToRunAfter) != 1 {
		t.Fatal(fmt.Errorf("programsToRun expected to have %d entries", 1))
	}

	// inject to both
	var hashConfigBefore, hashConfigAfter string
GROUPLOOPBEFORE:
	for group, ptr := range programsToRunBefore {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, astBefore, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOPBEFORE
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOPBEFORE
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
				continue
			}

			hashConfigBefore = p.Config.HashStr()
		}
	}

GROUPLOOPAFTER:
	for group, ptr := range programsToRunAfter {
		programsCount := len(ptr)
		newPtr, err := InjectMonitoring(agentInfo, group, astAfter, ptr)
		if err != nil {
			t.Error(err)
			continue GROUPLOOPAFTER
		}

		if programsCount+1 != len(newPtr) {
			t.Errorf("incorrect programs to run count, expected: %d, got %d", programsCount+1, len(newPtr))
			continue GROUPLOOPAFTER
		}

		for _, p := range newPtr {
			if p.Spec.Name != MonitoringName {
				continue
			}

			hashConfigAfter = p.Config.HashStr()
		}
	}

	if hashConfigAfter == "" || hashConfigBefore == "" {
		t.Fatal("hash configs uninitialized")
	}

	if hashConfigAfter == hashConfigBefore {
		t.Fatal("hash config equal, expected to be different")
	}
}

var inputConfigMap = map[string]interface{}{
	"agent.monitoring": map[string]interface{}{
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
	"inputs": []map[string]interface{}{
		{
			"type":       "log",
			"use_output": "infosec1",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
			},
			"processors": []interface{}{
				map[string]interface{}{
					"dissect": map[string]interface{}{
						"tokenizer": "---",
					},
				},
			},
		},
		{
			"type":       "system/metrics",
			"use_output": "infosec1",
			"streams": []map[string]interface{}{
				{
					"id":      "system/metrics-system.core",
					"enabled": true,
					"dataset": "system.core",
					"period":  "10s",
					"metrics": []string{"percentages"},
				},
			},
		},
	},
}

var inputConfigMapDefaults = map[string]interface{}{
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

	"inputs": []map[string]interface{}{
		{
			"type":       "log",
			"use_output": "infosec1",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
			},
			"processors": []interface{}{
				map[string]interface{}{
					"dissect": map[string]interface{}{
						"tokenizer": "---",
					},
				},
			},
		},
		{
			"type":       "system/metrics",
			"use_output": "infosec1",
			"streams": []map[string]interface{}{
				{
					"id":      "system/metrics-system.core",
					"enabled": true,
					"dataset": "system.core",
					"period":  "10s",
					"metrics": []string{"percentages"},
				},
			},
		},
	},
}

var inputConfigMapDisabled = map[string]interface{}{
	"agent.monitoring": map[string]interface{}{
		"enabled": false,
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

	"inputs": []map[string]interface{}{
		{
			"type": "log",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
			},
			"processors": []interface{}{
				map[string]interface{}{
					"dissect": map[string]interface{}{
						"tokenizer": "---",
					},
				},
			},
		},
		{
			"type":       "system/metrics",
			"use_output": "infosec1",
			"streams": []map[string]interface{}{
				{
					"id":      "system/metrics-system.core",
					"enabled": true,
					"dataset": "system.core",
					"period":  "10s",
					"metrics": []string{"percentages"},
				},
			},
		},
	},
}

var inputChange1 = map[string]interface{}{
	"agent.monitoring": map[string]interface{}{
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
		"monitoring": map[string]interface{}{
			"type":       "elasticsearch",
			"index_name": "general",
			"pass":       "xxx",
			"url":        "xxxxx",
			"username":   "monitoring-uname",
		},
	},
	"inputs": []map[string]interface{}{
		{
			"type": "log",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
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
}

var inputChange2 = map[string]interface{}{
	"agent.monitoring": map[string]interface{}{
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
		"monitoring": map[string]interface{}{
			"type":       "elasticsearch",
			"index_name": "general",
			"pass":       "xxx",
			"url":        "xxxxx",
			"username":   "monitoring-uname",
		},
	},
	"inputs": []map[string]interface{}{
		{
			"type": "log",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
				{"paths": "/yyyy"},
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
}

var inputConfigLS = map[string]interface{}{
	"agent.monitoring": map[string]interface{}{
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
		"monitoring": map[string]interface{}{
			"type":                        "logstash",
			"hosts":                       "192.168.1.2",
			"ssl.certificate_authorities": []string{"/etc/pki.key"},
		},
	},
	"inputs": []map[string]interface{}{
		{
			"type": "log",
			"streams": []map[string]interface{}{
				{"paths": "/xxxx"},
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
}
