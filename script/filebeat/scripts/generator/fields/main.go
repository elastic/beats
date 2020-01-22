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

	"github.com/elastic/beats/filebeat/generator/fields"
)

func main() {
	module := flag.String("module", "", "Name of the module")
	fileset := flag.String("fileset", "", "Name of the fileset")
	beatsPath := flag.String("beats_path", ".", "Path to elastic/beats")
	noDoc := flag.Bool("nodoc", false, "Generate documentation for fields")
	flag.Parse()

	if *module == "" {
		fmt.Println("Missing parameter: module")
		os.Exit(1)
	}

	if *fileset == "" {
		fmt.Println("Missing parameter: fileset")
		os.Exit(1)
	}

	err := fields.Generate(*beatsPath, *module, *fileset, *noDoc)
	if err != nil {
		fmt.Printf("Error while generating fields.yml: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("Fields.yml generated for %s/%s\n", *module, *fileset)
}
