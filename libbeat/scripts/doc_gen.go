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

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/thehivecorporation/log"
)

type command struct {
	Name         string
	variables    map[string]*struct{}
	Variables    []string
	Dependencies []string
}

func main() {
	f, _ := os.Open("Makefile")

	reader := bufio.NewReader(f)

	phonyString := ".PHONY: "
	isPhoneReg, err := regexp.Compile(phonyString)
	if err != nil {
		log.WithError(err).Fatal("Could not compile regexp")
	}

	commands := make([]*command, 0)

	var c *command
	var isPreviousLinePhony bool
	var finish bool

	// Iterate lines of the Makefile
	for {
		l, _, err := reader.ReadLine()
		if err != nil {
			finish = true
		}

		line := lineWithoutComments(string(l))

		if isPhoneReg.Match(l) {
			if c != nil {
				commands = finishCommandAndAdd(commands, c)
			}

			// New phony
			c = &command{
				Name:      strings.Replace(line, phonyString, "", -1),
				variables: make(map[string]*struct{}),
			}

			isPreviousLinePhony = true
			continue
		} else if c != nil {
			variablesFromLine(line, c.variables)
		}

		if isPreviousLinePhony {
			// Get dependent commands
			res := strings.TrimLeft(line, fmt.Sprintf("%s:", c.Name))
			c.Dependencies = cleanStringsArray(strings.Split(res, " "))
		}

		isPreviousLinePhony = false

		if finish {
			commands = finishCommandAndAdd(commands, c)
			break
		}
	}

	// Print a nice JSON, still for debugging
	byt, _ := json.MarshalIndent(commands, "", "  ")
	fmt.Println(string(byt))
}

func finishCommandAndAdd(cs []*command, c *command) []*command {
	c.Variables = mapStructToStringArray(c.variables)
	cs = append(cs, c)
	return cs
}

func ensureCorrectVariableCatching(s string) string {
	res := strings.Split(s, "$")
	if len(res) == 1 {
		return res[0]
	}

	return res[1]
}

func mapStructToStringArray(in map[string]*struct{}) []string {
	res := make([]string, 0)

	for k := range in {
		res = append(res, k)
	}

	return res
}

func cleanStringsArray(ss []string) []string {
	result := make([]string, 0)

	for _, s := range ss {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			result = append(result, s)
		}
	}

	return result
}

func lineWithoutComments(l string) string {
	line := strings.SplitAfter(l, "#")[0]
	return strings.TrimSpace(strings.TrimRight(line, "#"))
}

func doNothing(s string) {}

func variablesFromLine(l string, targetMap map[string]*struct{}) {
	var curVar string
	var capture bool
	var prev rune
	for _, char := range string(l) {
		sChar := string(char)
		if char == '$' {
			doNothing(sChar)
		}

		if (char == '{' || char == '(') && prev == '$' {
			capture = true
		} else if char == '}' || char == ')' {
			if curVar != "" {
				targetMap[ensureCorrectVariableCatching(curVar)] = &struct{}{}
			}

			curVar = ""
			capture = false
		} else if capture == true {
			curVar = fmt.Sprintf("%s%c", curVar, char)
		}

		prev = char
	}
}
