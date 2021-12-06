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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var files = []string{
	".go-version",
	"auditbeat/Dockerfile",
	"filebeat/Dockerfile",
	"heartbeat/Dockerfile",
	"libbeat/Dockerfile",
	"libbeat/docs/version.asciidoc",
	"metricbeat/Dockerfile",
	"metricbeat/module/http/_meta/Dockerfile",
	"x-pack/functionbeat/Dockerfile",
	"x-pack/libbeat/Dockerfile",
}

func main() {
	currVersion := getGoVersion()
	newVersion := flag.String("newversion", currVersion, "new version of Go")

	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	updateGoVersion(currVersion, *newVersion)
}

func getGoVersion() string {
	version, err := ioutil.ReadFile(".go-version")
	checkErr(err)
	return strings.TrimRight(string(version), "\r\n")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func updateGoVersion(oldVersion, newVersion string) {
	for _, file := range files {
		fmt.Printf("Updating Go version from %s to %s in %s\n", oldVersion, newVersion, file)
		content, err := ioutil.ReadFile(file)
		checkErr(err)
		updatedContent := strings.ReplaceAll(string(content), oldVersion, newVersion)
		err = ioutil.WriteFile(file, []byte(updatedContent), 0644)
		checkErr(err)
	}
}
