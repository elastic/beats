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
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/generator/fields"
)

func main() {
	esBeatsPath := flag.String("es_beats_path", "..", "Path to elastic/beats")
	beatPath := flag.String("beat_path", ".", "Path to your Beat")
	flag.Parse()

	beatFieldsPath := flag.Args()
	name := filepath.Base(*beatPath)

	err := os.MkdirAll(filepath.Join(*beatPath, "_meta"), 0744)
	if err != nil {
		fmt.Printf("Cannot creata _meta dir for %s: %v\n", name, err)
		os.Exit(1)
	}

	if len(beatFieldsPath) == 0 {
		fmt.Println("No field files to collect")
		err = fields.AppendFromLibbeat(*esBeatsPath, *beatPath)
		if err != nil {
			fmt.Printf("Cannot generate global fields.yml for %s: %v\n", name, err)
			os.Exit(2)
		}
		return
	}

	if *beatPath == "" {
		fmt.Println("beat_path cannot be empty")
		os.Exit(1)
	}

	pathToModules := filepath.Join(*beatPath, beatFieldsPath[0])
	fieldFiles, err := fields.CollectModuleFiles(pathToModules)
	if err != nil {
		fmt.Printf("Cannot collect fields.yml files: %v\n", err)
		os.Exit(2)

	}

	err = fields.Generate(*esBeatsPath, *beatPath, fieldFiles)
	if err != nil {
		fmt.Printf("Cannot generate global fields.yml file for %s: %v\n", name, err)
		os.Exit(3)
	}

	fmt.Printf("Generated fields.yml for %s\n", name)
}
