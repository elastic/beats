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
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/dashboards"
	"github.com/elastic/beats/libbeat/kibana"
)

var indexPattern = false
var quiet = false

func main() {
	kibanaURL := flag.String("kibana", "http://localhost:5601", "Kibana URL")
	spaceID := flag.String("space-id", "", "Space ID")
	dashboard := flag.String("dashboard", "", "Dashboard ID")
	fileOutput := flag.String("output", "output.json", "Output file")
	ymlFile := flag.String("yml", "", "Path to the module.yml file containing the dashboards")
	flag.BoolVar(&indexPattern, "indexPattern", false, "include index-pattern in output")
	flag.BoolVar(&quiet, "quiet", false, "be quiet")

	flag.Parse()
	log.SetFlags(0)

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}

	elements := strings.Split(kibanaURL, "://")
	if len(elements) != 2 {
		log.Fatalf("Invalid Kibana URL format")
	}

	client, err := kibana.NewKibanaClient(kibana.ClientConfig{
		Protocol: elements[0],
		Host:     elements[1],
		SpaceID:  spaceID,
	})

	if len(*ymlFile) == 0 && len(*dashboard) == 0 {
		flag.Usage()
		log.Fatalf("Please specify a dashboard ID (-dashboard) or a manifest file (-yml)")
	}

	if len(*ymlFile) > 0 {
		results, infos, err := dashboards.ExportAllFromYml(client, *ymlFile)
		for i, r := range results {
			log.Printf("id=%s, name=%s\n", info.Dashboards[i].ID, info.Dashboards[i].File)
			err = dashboards.SaveToFile(r, info.Dashboards[i].File, filepath.Dir(*ymlFile), client.GetVersion())
			if err != nil {
				log.Fatalf("failed to export the dashboards: %s", err)
			}
		}
		os.Exit(0)
	}

	if len(*dashboard) > 0 {
		result, err := dashboards.Export(client, *dashboard)
		if err != nil {
			log.Fatalf("Failed to export the dashboard: %s", err)
		}
		bytes, err := json.Marshal(result)
		if err != nil {
			log.Fatalf("Failed to save the dashboard: %s", err)
		}

		err = ioutil.WriteFile(*fileOutput, bytes, 0644)
		if err != nil {
			log.Fatalf("Failed to save the dashboard: %s", err)

		}
		if !*quiet {
			log.Printf("The dashboard %s was exported under the %s file\n", dashboard, out)
		}
	}
}
