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
	"path"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/libbeat/generator/fields"
	"github.com/elastic/beats/licenses"
)

func main() {

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Module path must be set")
		os.Exit(1)
	}

	dir := args[0]

	modules, err := fields.GetModules(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching modules: %s\n", err)
		os.Exit(1)
	}

	for _, module := range modules {
		files, err := fields.CollectFiles(module, dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching files for module %s: %s\n", module, err)
			os.Exit(1)
		}

		data, err := fields.GenerateFieldsYml(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching files for module %s: %s\n", module, err)
			os.Exit(1)
		}

		bs, err := asset.CreateAsset(licenses.ASL2, "metricbeat", module, module, data, "asset.ModuleFieldsPri", dir+"/"+module)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating golang file from template: %s\n", err)
			os.Exit(1)
		}

		err = ioutil.WriteFile(path.Join(dir, module, "fields.go"), bs, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing fields.go: %s\n", err)
			os.Exit(1)
		}
	}
}
