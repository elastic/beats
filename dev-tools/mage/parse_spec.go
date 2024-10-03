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

type command struct {
	Name string
	Args []string
}

// ParseSpec parses agent.beat.spec.yml and generates test command
func ParseSpec() {
	specPath := os.Getenv("AGENTBEAT_SPEC")
	if specPath == "" {
		log.Fatal("AGENTBEAT_SPEC is not defined")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM is not defined")
	}

	spec, err := parseToObj()
	if err != nil {
		log.Fatalf("Error parsing agentbeat.spec.yml: %v", err)
	}

	inputList := filter(spec.Inputs, func(input input) bool {
		return contains(input.Platforms, platform)
	})

	log.Print(inputList)
}

func parseToObj() (spec, error) {
	specFile, err := os.ReadFile("../agentbeat.spec.yml")
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
