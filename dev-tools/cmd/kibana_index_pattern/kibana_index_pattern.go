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
	"log"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/libbeat/version"
)

var usageText = `
Usage: kibana_index_pattern [flags]
  kibana_index_pattern generates Kibana index patterns from the Beat's
  fields.yml file. It will create a index pattern file that is usable with both
  Kibana 6.x and 7.x.
Options:
`[1:]

var (
	beatName       string
	beatVersion    string
	indexPattern   string
	fieldsYAMLFile string
	outputDir      string
)

func init() {
	flag.StringVar(&beatName, "beat", "", "Name of the beat. (Required)")
	flag.StringVar(&beatVersion, "version", version.GetDefaultVersion(), "Beat version. (Required)")
	flag.StringVar(&indexPattern, "index", "", "Kibana index pattern. (Required)")
	flag.StringVar(&fieldsYAMLFile, "fields", "fields.yml", "fields.yml file containing all fields used by the Beat.")
	flag.StringVar(&outputDir, "out", "build/kibana", "Output dir.")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if beatName == "" {
		log.Fatal("Name of the Beat must be set (-beat).")
	}

	if beatVersion == "" {
		log.Fatal("Beat version must be set (-version).")
	}

	if indexPattern == "" {
		log.Fatal("Index pattern must be set (-index).")
	}

	versions := []string{
		"6.0.0",
	}
	for _, version := range versions {
		version, _ := common.NewVersion(version)
		indexPattern, err := kibana.NewGenerator(indexPattern, beatName, fieldsYAMLFile, outputDir, beatVersion, *version)
		if err != nil {
			log.Fatal(err)
		}

		file, err := indexPattern.Generate()
		if err != nil {
			log.Fatal(err)
		}

		// Log output file location.
		absFile, err := filepath.Abs(file)
		if err != nil {
			absFile = file
		}
		log.Printf(">> The index pattern was created under %v", absFile)
	}
}
