// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package program

import (
	"regexp"

	"github.com/elastic/fleet/x-pack/pkg/agent/transpiler"
)

// Spec represents a specific program specification, it contains information about how to run the
// program and also the rules to apply to the single configuration to create a specific program
// configuration.
//
// NOTE: Current spec are build at compile time, we may want to revisit that to allow other program
// to register their spec via the UI. We will need to assert the risks.
type Spec struct {
	Name  string
	Cmd   string
	Rules transpiler.Rule
	When  func(ast *transpiler.AST) bool
}

// Filebeat specifications contains all the information for the Agent to start Filebeat.
var Filebeat = Spec{
	Name: "Filebeat",
	Cmd:  "filebeat",
	Rules: transpiler.NewRuleList(
		transpiler.Map("inputs",
			transpiler.Translate("type", []transpiler.TranslateKV{
				transpiler.TranslateKV{K: "event/file", V: "log"},
				transpiler.TranslateKV{K: "event/stdin", V: "stdin"},
				transpiler.TranslateKV{K: "event/udp", V: "udp"},
				transpiler.TranslateKV{K: "event/tcp", V: "tcp"},
				transpiler.TranslateKV{K: "log/docker", V: "docker"},
				transpiler.TranslateKV{K: "log/redis_slowlog", V: "redis"},
				transpiler.TranslateKV{K: "log/syslog", V: "syslog"},
			})),
		transpiler.FilterValues(
			"inputs",
			"type",
			"log",
			"stdin",
			"udp",
			"tcp",
			"docker",
			"redis",
			"syslog",
		),
		transpiler.Copy("inputs", "filebeat"),
		transpiler.Filter("filebeat", "output", "keystore"),
	),
	When: func(ast *transpiler.AST) bool {
		return transpiler.CountComp(ast, "filebeat.inputs", func(a int) bool { return a > 0 })
	},
}

// Metricbeat specifications contains all the information for the Agent to start Metricbeat
var Metricbeat = Spec{
	Name: "Metricbeat",
	Cmd:  "metricbeat",
	Rules: transpiler.NewRuleList(
		transpiler.FilterValuesWithRegexp("inputs", "type", regexp.MustCompile("^metric/.+")),
		transpiler.Map("inputs",
			transpiler.TranslateWithRegexp("type", regexp.MustCompile("^metric/(.+)"), "$1"),
		),
		transpiler.Copy("inputs", "metricbeat"),
		transpiler.Filter("metricbeat", "output", "keystore"),
	),
	When: func(ast *transpiler.AST) bool {
		return transpiler.CountComp(ast, "metricbeat.inputs", func(a int) bool { return a > 0 })
	},
}

// Auditbeat specifications contains all the information for the Agent to start auditbeat.
var Auditbeat = Spec{
	Name: "Auditbeat",
	Cmd:  "auditbeat",
	Rules: transpiler.NewRuleList(
		transpiler.FilterValuesWithRegexp("inputs", "type", regexp.MustCompile("^audit/.+")),
		transpiler.Map("inputs",
			transpiler.TranslateWithRegexp("type", regexp.MustCompile("^audit/(.+)"), "$1"),
		),

		transpiler.Copy("inputs", "auditbeat"),
		transpiler.Map("auditbeat.inputs",
			transpiler.Rename("type", "module"),
		),
		transpiler.Rename("auditbeat.inputs", "modules"),
		transpiler.Filter("auditbeat", "output", "keystore"),
	),
	When: func(ast *transpiler.AST) bool {
		return transpiler.CountComp(ast, "auditbeat.modules", func(a int) bool { return a > 0 })
	},
}

// Journalbeat specifications contains all the information for the Agent to start journalbeat.
var Journalbeat = Spec{
	Name: "Journalbeat",
	Cmd:  "journalbeat",
	Rules: transpiler.NewRuleList(
		transpiler.FilterValues(
			"inputs",
			"type",
			"log/journal",
		),
		transpiler.Copy("inputs", "journalbeat"),
		transpiler.Filter("journalbeat", "output", "keystore"),
	),
	When: func(ast *transpiler.AST) bool {
		return transpiler.CountComp(ast, "journalbeat.inputs", func(a int) bool { return a > 0 })
	},
}

// Heartbeat specifications contains all the information for the Agent to start hearthbeat.
var Heartbeat = Spec{
	Name: "Heartbeat",
	Cmd:  "heartbeat",
	Rules: transpiler.NewRuleList(
		transpiler.FilterValuesWithRegexp("inputs", "type", regexp.MustCompile("^monitor/.+")),
		transpiler.Map("inputs",
			transpiler.TranslateWithRegexp("type", regexp.MustCompile("^monitor/(.+)"), "$1"),
		),
		transpiler.Copy("inputs", "heartbeat"),
		transpiler.Filter("heartbeat", "output", "keystore"),
	),
	When: func(ast *transpiler.AST) bool {
		return transpiler.CountComp(ast, "heartbeat.inputs", func(a int) bool { return a > 0 })
	},
}

// Supported is the list of programs currently supported by the Agent.
var Supported = []Spec{
	Filebeat,
	Metricbeat,
	Auditbeat,
	Journalbeat,
	Heartbeat,
}

// Program represents a program that must be started or must run.
type Program struct {
	Spec   Spec
	Config *transpiler.AST
}

// Programs take a Tree representation of the main configuration and apply all the different
// programs rules and generate individual configuration from the rules.
func Programs(singleConfig *transpiler.AST) ([]Program, error) {
	programs := make([]Program, 0)
	for _, spec := range Supported {
		// TODO: better error handling here.
		specificAST := singleConfig.Clone()
		err := spec.Rules.Apply(specificAST)
		if err != nil {
			return nil, err
		}

		if !spec.When(specificAST) {
			continue
		}

		// TODO(ph): Add executes conditions, ie: len(inputs) > 0
		program := Program{
			Spec:   spec,
			Config: specificAST,
		}
		programs = append(programs, program)
	}
	return programs, nil
}

// KnownProgramNames returns a list of runnable programs by the agent.
func KnownProgramNames() []string {
	names := make([]string, len(Supported))
	for idx, program := range Supported {
		names[idx] = program.Name
	}
	return names
}
