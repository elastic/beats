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

package mage

import (
	"gopkg.in/yaml.v2"

	"log"
	"os"
	"strings"
)

type spec struct {
	Inputs []input
}

type input struct {
	Name        string
	Description string
	Platforms   []string
	Command     command
}

func (i *input) GetCommand() string {
	return strings.Join(i.Command.Args, " ")
}

type command struct {
	Name string
	Args []string
}

// SpecCommands parses agent.beat.spec.yml and collects commands for tests
func SpecCommands(specPath string, platform string) []string {
	spec, _ := parseToObj(specPath)

	filteredInputs := filter(spec.Inputs, func(input input) bool {
		return contains(input.Platforms, platform)
	})

	commands := make(map[string]interface{})
	for _, i := range filteredInputs {
		commands[i.GetCommand()] = nil
	}
	keys := make([]string, 0, len(commands))
	for k := range commands {
		keys = append(keys, k)
	}

	return keys
}

func parseToObj(path string) (spec, error) {
	specFile, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error opening agentbeat.spec.yml: %v", err)
		return spec{}, err
	}
	var spec spec
	err = yaml.Unmarshal(specFile, &spec)
	if err != nil {
		log.Fatalf("Error parsing agentbeat.spec.yml: %v", err)
		return spec, err
	}
	return spec, nil
}

func filter[T any](slice []T, condition func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if condition(v) {
			result = append(result, v)
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
