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
	"path"

	"github.com/elastic/beats/filebeat/scripts/generator"
)

func generateFileset(module, fileset, modulesPath, beatsPath string) error {
	filesetPath := path.Join(modulesPath, "module", module, fileset)
	if generator.DirExists(filesetPath) {
		return fmt.Errorf("fileset already exists: %s", fileset)
	}

	err := generator.CreateDirectories(filesetPath, []string{"", "_meta", "test", "config", "ingest"})
	if err != nil {
		return err
	}

	replace := map[string]string{"module": module, "fileset": fileset}
	templatesPath := path.Join(beatsPath, "scripts", "fileset")
	filesToCopy := []string{path.Join("config", "config.yml"), path.Join("ingest", "pipeline.json"), "manifest.yml"}
	err = generator.CopyTemplates(templatesPath, filesetPath, filesToCopy, replace)
	if err != nil {
		return err
	}
	err = generator.RenameConfigYml(modulesPath, module, fileset)
	if err != nil {
		return err
	}

	return addFilesetDashboard(module, fileset, modulesPath, templatesPath)
}

func addFilesetDashboard(module, fileset, modulesPath, templatesPath string) error {
	template := path.Join(templatesPath, "module-fileset.yml")
	dest := path.Join(modulesPath, "module", module, "module.yml")
	replacement := map[string]string{"module": module, "fileset": fileset}
	return generator.AppendTemplate(template, dest, replacement)
}

func main() {
	module := flag.String("module", "", "Name of the module")
	fileset := flag.String("fileset", "", "Name of the fileset")
	modulesPath := flag.String("path", ".", "Path to the generated fileset")
	beatsPath := flag.String("beats_path", ".", "Path to elastic/beats")
	flag.Parse()

	if *module == "" {
		fmt.Println("Missing parameter: module")
		os.Exit(1)
	}

	if *fileset == "" {
		fmt.Println("Missing parameter: fileset")
		os.Exit(1)
	}

	modulePath := path.Join(*modulesPath, "module", *module)
	if !generator.DirExists(modulePath) {
		fmt.Print("Cannot generate fileset: module not exists, please create module first by create-module command\n")
		os.Exit(2)
	}

	err := generateFileset(*module, *fileset, *modulesPath, *beatsPath)
	if err != nil {
		fmt.Printf("Cannot generate fileset: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("New fileset was generated, please check that module.yml file have proper fileset dashboard settings. After setting up Grok pattern in pipeline.json, please generate fields.yml")
}
