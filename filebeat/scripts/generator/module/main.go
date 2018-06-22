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

func generateModule(module, modulesPath, beatsPath string) error {
	modulePath := path.Join(modulesPath, "module", module)
	if generator.DirExists(modulePath) {
		return fmt.Errorf("module already exists: %s", module)
	}

	err := generator.CreateDirectories(modulePath, []string{path.Join("_meta", "kibana", "6")})
	if err != nil {
		return err
	}

	replace := map[string]string{"module": module}
	templatesPath := path.Join(beatsPath, "scripts", "module")
	filesToCopy := []string{path.Join("_meta", "fields.yml"), path.Join("_meta", "docs.asciidoc"), path.Join("_meta", "config.yml"), path.Join("module.yml")}
	generator.CopyTemplates(templatesPath, modulePath, filesToCopy, replace)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	module := flag.String("module", "", "Name of the module")
	modulePath := flag.String("path", ".", "Path to the generated fileset")
	beatsPath := flag.String("beats_path", ".", "Path to elastic/beats")
	flag.Parse()

	if *module == "" {
		fmt.Println("Missing parameter: module")
		os.Exit(1)
	}

	err := generateModule(*module, *modulePath, *beatsPath)
	if err != nil {
		fmt.Printf("Cannot generate module: %v\n", err)
		os.Exit(2)
	}

	fmt.Println("New module was generated, now you can start creating filesets by create-fileset command.")
}
