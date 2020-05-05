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

package apmschema

import (
	"log"
	"path"
	"path/filepath"
	"runtime"

	"github.com/santhosh-tekuri/jsonschema"
)

var (
	// Error is the compiled JSON Schema for an error.
	Error *jsonschema.Schema

	// Metadata is the compiled JSON Schema for metadata.
	Metadata *jsonschema.Schema

	// MetricSet is the compiled JSON Schema for a set of metrics.
	MetricSet *jsonschema.Schema

	// Span is the compiled JSON Schema for a span.
	Span *jsonschema.Schema

	// Transaction is the compiled JSON Schema for a transaction.
	Transaction *jsonschema.Schema
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("source line info not available")
	}
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft4
	schemaDir := path.Join(filepath.ToSlash(filepath.Dir(filename)), "jsonschema")
	if runtime.GOOS == "windows" {
		schemaDir = "/" + schemaDir
	}
	compile := func(filepath string, out **jsonschema.Schema) {
		schema, err := compiler.Compile("file://" + path.Join(schemaDir, filepath))
		if err != nil {
			log.Fatal(err)
		}
		*out = schema
	}
	compile("errors/error.json", &Error)
	compile("metadata.json", &Metadata)
	compile("metricsets/metricset.json", &MetricSet)
	compile("spans/span.json", &Span)
	compile("transactions/transaction.json", &Transaction)
}
