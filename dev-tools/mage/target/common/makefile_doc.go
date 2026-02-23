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

package common

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var reTargetDoc = regexp.MustCompile(
	`^((?P<name>[-_\w]+)|(\$\{(?P<varname>[-_\w]+)\}))\s*:.*##\s*@(?P<category>\w+)\s+(?P<doc>.*)`)

var reVarHelp = regexp.MustCompile(
	`^(?P<name>\w+)\s*\??=\s*(?P<default>[^#]+?)\s*(##\s*@(?P<category>\w+):?\s+(?P<doc>.*))?$`)

type makefileDocEntry struct {
	name       string
	doc        string
	isVariable bool
	defVal     string
}

// GenerateMakefileDoc parses annotated Makefile targets and prints help.
// Replaces libbeat/scripts/generate_makefile_doc.py.
// Set MAKEFILE_LIST env var to a space-separated list of Makefile paths.
// Set VARIABLES=true to show variables instead of targets.
func GenerateMakefileDoc() error {
	fileList := os.Getenv("MAKEFILE_LIST")
	files := strings.Fields(fileList)
	showVariables := os.Getenv("VARIABLES") == "true"

	targetCats := make(map[string][]makefileDocEntry)
	varCats := make(map[string][]makefileDocEntry)
	var targetCatOrder, varCatOrder []string
	variables := make(map[string]string)

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			parseMakeTarget(line, targetCats, &targetCatOrder)
			name, def := parseMakeVariable(line, varCats, &varCatOrder)
			if name != "" {
				if _, ok := variables[name]; !ok {
					variables[name] = def
				}
			}
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("scanning %s: %w", path, err)
		}
	}

	for _, entries := range targetCats {
		for i := range entries {
			if entries[i].isVariable {
				if v, ok := variables[entries[i].name]; ok {
					entries[i].name = v
				}
				entries[i].isVariable = false
			}
		}
	}

	if !showVariables {
		fmt.Println("Usage: make [target] [VARIABLE=value]")
		printMakefileHelp(targetCats, targetCatOrder)
	} else {
		printMakefileHelp(varCats, varCatOrder)
	}
	return nil
}

func parseMakeTarget(line string, cats map[string][]makefileDocEntry, order *[]string) {
	m := reTargetDoc.FindStringSubmatch(line)
	if m == nil {
		return
	}
	name := regexpGroup(reTargetDoc, m, "varname")
	isVar := name != ""
	if !isVar {
		name = regexpGroup(reTargetDoc, m, "name")
	}
	if name == "" {
		return
	}
	cat := regexpGroup(reTargetDoc, m, "category")
	if cat == "" {
		return
	}
	cat = strings.ReplaceAll(cat, "_", " ")
	cat = capitalizeFirstRune(cat)
	doc := strings.TrimRight(regexpGroup(reTargetDoc, m, "doc"), ". ")
	doc = capitalizeFirstRune(doc)

	if _, ok := cats[cat]; !ok {
		*order = append(*order, cat)
	}
	cats[cat] = append(cats[cat], makefileDocEntry{name: name, doc: doc, isVariable: isVar})
}

func parseMakeVariable(line string, cats map[string][]makefileDocEntry, order *[]string) (string, string) {
	m := reVarHelp.FindStringSubmatch(line)
	if m == nil {
		return "", ""
	}
	name := regexpGroup(reVarHelp, m, "name")
	def := strings.TrimSpace(regexpGroup(reVarHelp, m, "default"))
	cat := regexpGroup(reVarHelp, m, "category")
	doc := regexpGroup(reVarHelp, m, "doc")

	if cat != "" && doc != "" {
		cat = strings.ReplaceAll(cat, "_", " ")
		cat = capitalizeFirstRune(cat)
		doc = strings.TrimRight(doc, ". ")
		doc = capitalizeFirstRune(doc)

		if _, ok := cats[cat]; !ok {
			*order = append(*order, cat)
		}
		cats[cat] = append(cats[cat], makefileDocEntry{name: name, doc: doc, isVariable: true, defVal: def})
	}

	return name, def
}

func printMakefileHelp(cats map[string][]makefileDocEntry, order []string) {
	maxName := 0
	for _, cat := range order {
		for _, e := range cats[cat] {
			if len(e.name) > maxName {
				maxName = len(e.name)
			}
		}
	}

	for _, cat := range order {
		fmt.Printf("\n%s:\n", cat)
		for _, e := range cats[cat] {
			defStr := ""
			if e.defVal != "" {
				defStr = fmt.Sprintf(" Default: %s", e.defVal)
			}
			fmt.Printf("\t%-*s\t%s.%s\n", maxName, e.name, e.doc, defStr)
		}
	}
}

func regexpGroup(re *regexp.Regexp, m []string, name string) string {
	for i, gn := range re.SubexpNames() {
		if gn == name && i < len(m) {
			return m[i]
		}
	}
	return ""
}

func capitalizeFirstRune(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
