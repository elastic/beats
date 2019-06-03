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
	"net/url"
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/dashboards"
	"github.com/elastic/beats/libbeat/kibana"
)

var (
	indexPattern = false
	quiet        = false
)

const (
	kibanaTimeout = 90 * time.Second
)

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

	u, err := url.Parse(*kibanaURL)
	if err != nil {
		log.Fatalf("Error parsing Kibana URL: %v", err)
	}

	var user, pass string
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	client, err := kibana.NewClientWithConfig(&kibana.ClientConfig{
		Protocol: u.Scheme,
		Host:     u.Host,
		Username: user,
		Password: pass,
		Path:     u.Path,
		SpaceID:  *spaceID,
		Timeout:  kibanaTimeout,
	})
	if err != nil {
		log.Fatalf("Error while connecting to Kibana: %v", err)
	}

	if len(*ymlFile) == 0 && len(*dashboard) == 0 {
		flag.Usage()
		log.Fatalf("Please specify a dashboard ID (-dashboard) or a manifest file (-yml)")
	}

	if len(*ymlFile) > 0 {
		err = exportDashboardsFromYML(client, *ymlFile)
		if err != nil {
			log.Fatalf("Failed to export dashboards from YML file: %v", err)
		}
		return
	}

	if len(*dashboard) > 0 {
		err = exportSingleDashboard(client, *dashboard, *fileOutput)
		if err != nil {
			log.Fatalf("Failed to export the dashboard: %v", err)
		}
		if !quiet {
			log.Printf("The dashboard %s was exported under '%s'\n", *dashboard, *fileOutput)
		}
		return
	}
}

func exportDashboardsFromYML(client *kibana.Client, ymlFile string) error {
	results, info, err := dashboards.ExportAllFromYml(client, ymlFile)
	if err != nil {
		return err
	}
	for i, r := range results {
		log.Printf("id=%s, name=%s\n", info.Dashboards[i].ID, info.Dashboards[i].File)
		r = dashboards.DecodeExported(r)
		err = dashboards.SaveToFile(r, info.Dashboards[i].File, filepath.Dir(ymlFile), client.GetVersion())
		if err != nil {
			return err
		}
	}
	return nil
}

func exportSingleDashboard(client *kibana.Client, dashboard, output string) error {
	result, err := dashboards.Export(client, dashboard)
	if err != nil {
		return fmt.Errorf("failed to export the dashboard: %+v", err)
	}
	result = dashboards.DecodeExported(result)
	err = ioutil.WriteFile(output, []byte(result.StringToPrint()), dashboards.OutputPermission)
	if err != nil {
		return fmt.Errorf("failed to save the dashboards: %+v", err)
	}
	return nil
}
