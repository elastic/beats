// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"io"
	"io/ioutil"
	"log"
	"os"

	"go.elastic.co/go-licence-detector/dependency"
	"go.elastic.co/go-licence-detector/detector"
	"go.elastic.co/go-licence-detector/render"
	"go.elastic.co/go-licence-detector/validate"
)

var (
	depsTemplateFlag    = flag.String("depsTemplate", "example/templates/dependencies.asciidoc.tmpl", "Path to the dependency list template file.")
	depsOutFlag         = flag.String("depsOut", "", "Path to output the dependency list.")
	inFlag              = flag.String("in", "-", "Dependency list (output from go list -m -json all).")
	includeIndirectFlag = flag.Bool("includeIndirect", false, "Include indirect dependencies.")
	licenceDataFlag     = flag.String("licenceData", "", "Path to the licence database. Uses embedded database if empty.")
	noticeTemplateFlag  = flag.String("noticeTemplate", "example/templates/NOTICE.txt.tmpl", "Path to the NOTICE template file.")
	noticeOutFlag       = flag.String("noticeOut", "", "Path to output the notice.")
	overridesFlag       = flag.String("overrides", "", "Path to the file containing override directives.")
	rulesFlag           = flag.String("rules", "", "Path to file containing rules regarding licence types. Uses embedded rules if empty.")
	validateFlag        = flag.Bool("validate", false, "Validate results (slow).")
)

func main() {
	flag.Parse()

	// create reader for dependency information
	depInput, err := mkReader(*inFlag)
	if err != nil {
		log.Fatalf("Failed to create reader for %s: %v", *inFlag, err)
	}
	defer depInput.Close()

	// create licence classifier
	classifier, err := detector.NewClassifier(*licenceDataFlag)
	if err != nil {
		log.Fatalf("Failed to create licence classifier: %v", err)
	}

	// load overrides
	overrides, err := dependency.LoadOverrides(*overridesFlag)
	if err != nil {
		log.Fatalf("Failed to load overrides: %v", err)
	}

	// load rules
	rules, err := detector.LoadRules(*rulesFlag)
	if err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}

	// detect dependencies
	dependencies, err := detector.Detect(depInput, classifier, rules, overrides, *includeIndirectFlag)
	if err != nil {
		log.Fatalf("Failed to detect licences: %v", err)
	}

	if *validateFlag {
		if err := validate.Validate(dependencies); err != nil {
			log.Fatalf("Validation failed: %v", err)
		}
	}

	// only generate notice file if the output path is provided
	if *noticeOutFlag != "" {
		if err := render.Template(dependencies, *noticeTemplateFlag, *noticeOutFlag); err != nil {
			log.Fatalf("Failed to render notice: %v", err)
		}
	}

	// only generate dependency listing if the output path is provided
	if *depsOutFlag != "" {
		if err := render.Template(dependencies, *depsTemplateFlag, *depsOutFlag); err != nil {
			log.Fatalf("Failed to render dependency list: %v", err)
		}
	}
}

func mkReader(path string) (io.ReadCloser, error) {
	if path == "-" {
		return ioutil.NopCloser(os.Stdin), nil
	}

	return os.Open(path)
}
