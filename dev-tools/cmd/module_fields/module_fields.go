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
	"log"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v8/libbeat/asset"
	"github.com/elastic/beats/v8/libbeat/generator/fields"
	"github.com/elastic/beats/v8/licenses"
)

var usageText = `
Usage: module_fields [flags] [module-dir]
  module_fields generates a fields.go file containing a copy of the module's
  field.yml data in a format that can be embedded in Beat's binary. module-dir
  should be the directory containing modules (e.g. filebeat/module).
Options:
`[1:]

var (
	beatName string
	license  string
)

func init() {
	flag.StringVar(&beatName, "beat", "", "Name of the beat. (Required)")
	flag.StringVar(&license, "license", "ASL2", "License header for generated file.")
	flag.Usage = usageFlag
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if beatName == "" {
		log.Fatal("You must use -beat to specify the beat name.")
	}

	license, err := licenses.Find(license)
	if err != nil {
		log.Fatalf("Invalid license specifier: %v", err)
	}

	args := flag.Args()
	if len(args) != 1 {
		log.Fatal("module-dir must be passed as an argument.")
	}
	dir := args[0]

	modules, err := fields.GetModules(dir)
	if err != nil {
		log.Fatalf("Error fetching modules: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to determine working directory: %v", err)
	}

	for _, module := range modules {
		files, err := fields.CollectFiles(module, dir)
		if err != nil {
			log.Fatalf("Error fetching files for module %v: %v", module, err)
		}
		if len(files) == 0 {
			// This can happen on moved modules
			log.Printf("No fields files for module %v", module)
			continue
		}

		data, err := fields.GenerateFieldsYml(files)
		if err != nil {
			log.Fatalf("error fetching files for package %v: %v", module, err)
		}

		p, err := filepath.Rel(wd, filepath.Join(dir, module))
		if err != nil {
			log.Fatal(err)
		}

		bs, err := asset.CreateAsset(license, beatName, module, module, data, "asset.ModuleFieldsPri", filepath.ToSlash(p))
		if err != nil {
			log.Fatalf("Error creating golang file from template: %v", err)
		}

		err = ioutil.WriteFile(filepath.Join(dir, module, "fields.go"), bs, 0644)
		if err != nil {
			log.Fatalf("Error writing fields.go: %v", err)
		}
	}
}

func usageFlag() {
	fmt.Fprintf(os.Stderr, usageText)
	flag.PrintDefaults()
}
