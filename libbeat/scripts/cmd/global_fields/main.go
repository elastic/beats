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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/menderesk/beats/v7/libbeat/generator/fields"
	"github.com/menderesk/beats/v7/libbeat/mapping"
)

func main() {
	var (
		esBeatsPath string
		beatPath    string
		output      string
	)
	flag.StringVar(&esBeatsPath, "es_beats_path", "..", "Path to menderesk/beats")
	flag.StringVar(&beatPath, "beat_path", ".", "Path to your Beat")
	flag.StringVar(&output, "out", "-", "Path to output. Default: stdout")
	flag.Parse()

	beatFieldsPaths := flag.Args()
	name := filepath.Base(beatPath)

	if beatPath == "" {
		fmt.Fprintf(os.Stderr, "beat_path cannot be empty")
		os.Exit(1)
	}

	esBeats, err := os.Open(esBeatsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening menderesk/beats: %+v\n", err)
		os.Exit(1)
	}
	beat, err := os.Open(beatPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening target Beat: %+v\n", err)
		os.Exit(1)
	}
	esBeatsInfo, err := esBeats.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info of menderesk/beats: %+v\n", err)
		os.Exit(1)
	}
	beatInfo, err := beat.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info of target Beat: %+v\n", err)
		os.Exit(1)
	}
	beat.Close()
	esBeats.Close()

	// If a community Beat does not have its own fields.yml file, it still requires
	// the fields coming from libbeat to generate e.g assets. In case of Elastic Beats,
	// it's not a problem because all of them has unique fields.yml files somewhere.
	if len(beatFieldsPaths) == 0 && os.SameFile(esBeatsInfo, beatInfo) {
		if output != "-" {
			fmt.Fprintln(os.Stderr, "No field files to collect")
		}
		return
	}

	var fieldsFiles []*fields.YmlFile
	for _, fieldsFilePath := range beatFieldsPaths {
		fieldsFile, err := fields.CollectModuleFiles(fieldsFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot collect fields.yml files: %+v\n", err)
			os.Exit(2)
		}

		fieldsFiles = append(fieldsFiles, fieldsFile...)
	}

	var buffer bytes.Buffer
	err = fields.Generate(esBeatsPath, beatPath, fieldsFiles, &buffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot generate global fields.yml file for %s: %+v\n", name, err)
		os.Exit(3)
	}

	_, err = mapping.LoadFields(buffer.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Generated global fields.yml file for %s is invalid: %+v\n", name, err)
		os.Exit(3)
	}

	if output == "-" {
		fmt.Print(buffer.String())
		return
	}

	err = ioutil.WriteFile(output, buffer.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write global fields.yml file for %s: %v", name, err)
	}

	outputPath, _ := filepath.Abs(output)
	fmt.Fprintf(os.Stderr, "Generated fields.yml for %s to %s\n", name, outputPath)
}
