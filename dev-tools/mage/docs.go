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

package mage

import (
	"log"

	"github.com/magefile/mage/sh"
)

type docsBuilder struct{}

// Docs holds the utilities for building documentation.
var Docs = docsBuilder{}

// FieldDocs generates docs/fields.asciidoc from the specified fields.yml file.
func (b docsBuilder) FieldDocs(fieldsYML string) error {
	// Run the docs_collector.py script.
	ve, err := PythonVirtualenv()
	if err != nil {
		return err
	}

	python, err := LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}

	esBeats, err := ElasticBeatsDir()
	if err != nil {
		return err
	}

	log.Println(">> Generating docs/fields.asciidoc for", BeatName)
	return sh.Run(python, LibbeatDir("scripts/generate_fields_docs.py"),
		fieldsYML,                     // Path to fields.yml.
		BeatName,                      // Beat title.
		esBeats,                       // Path to general beats folder.
		"--output_path", OSSBeatDir()) // It writes to {output_path}/docs/fields.asciidoc.
}
