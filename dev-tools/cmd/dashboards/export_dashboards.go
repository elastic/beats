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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
)

var exportAPI = "/api/kibana/dashboards/export"

type manifest struct {
	Dashboards []map[string]string `config:"dashboards"`
}

func makeURL(url, path string, params url.Values) string {
	if len(params) == 0 {
		return url + path
	}

	return strings.Join([]string{url, path, "?", params.Encode()}, "")
}

func Export(client *http.Client, conn string, dashboard string, out string) error {
	params := url.Values{}

	params.Add("dashboard", dashboard)

	fullURL := makeURL(conn, exportAPI, params)
	if !quiet {
		fmt.Printf("Calling HTTP GET %v\n", fullURL)
	}

	req, err := http.NewRequest("GET", fullURL, nil)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET HTTP request fails with: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fail to read response %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP GET %s fails with %s, %s", fullURL, resp.Status, body)
	}

	data, err := kibana.RemoveIndexPattern(body)
	if err != nil {
		return fmt.Errorf("fail to extract the index pattern: %v", err)
	}

	objects := data["objects"].([]interface{})
	for _, obj := range objects {
		o := obj.(common.MapStr)

		decodeValue(o, "attributes.uiStateJSON")
		decodeValue(o, "attributes.visState")
		decodeValue(o, "attributes.optionsJSON")
		decodeValue(o, "attributes.panelsJSON")
		decodeValue(o, "attributes.kibanaSavedObjectMeta.searchSourceJSON")
	}

	data["objects"] = objects
	err = ioutil.WriteFile(out, []byte(data.StringToPrint()), 0666)
	if !quiet {
		fmt.Printf("The dashboard %s was exported under the %s file\n", dashboard, out)
	}
	return err
}

func decodeValue(data common.MapStr, key string) {
	v, err := data.GetValue(key)
	if err != nil {
		return
	}
	s := v.(string)
	var d common.MapStr
	json.Unmarshal([]byte(s), &d)

	data.Put(key, d)
}

func ReadManifest(file string) ([]map[string]string, error) {
	cfg, err := common.LoadFile(file)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest file: %v", err)
	}

	var manifest manifest
	err = cfg.Unpack(&manifest)
	if err != nil {
		return nil, fmt.Errorf("error unpacking manifest: %v", err)
	}
	return manifest.Dashboards, nil
}

var indexPattern = false
var quiet = false

func main() {
	kibanaURL := flag.String("kibana", "http://localhost:5601", "Kibana URL")
	dashboard := flag.String("dashboard", "", "Dashboard ID")
	fileOutput := flag.String("output", "output.json", "Output file")
	ymlFile := flag.String("yml", "", "Path to the module.yml file containing the dashboards")
	flag.BoolVar(&indexPattern, "indexPattern", false, "include index-pattern in output")
	flag.BoolVar(&quiet, "quiet", false, "be quiet")

	flag.Parse()

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}

	client := &http.Client{Transport: transCfg}

	if len(*ymlFile) == 0 && len(*dashboard) == 0 {
		fmt.Printf("Please specify a dashboard ID (-dashboard) or a manifest file (-yml)\n\n")
		flag.Usage()
		os.Exit(0)
	}

	if len(*ymlFile) > 0 {
		dashboards, err := ReadManifest(*ymlFile)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}

		for _, dashboard := range dashboards {
			fmt.Printf("id=%s, name=%s\n", dashboard["id"], dashboard["file"])
			directory := path.Join(path.Dir(*ymlFile), "_meta/kibana/6/dashboard")
			err := os.MkdirAll(directory, 0755)
			if err != nil {
				fmt.Printf("ERROR: fail to create directory %s: %v", directory, err)
			}
			err = Export(client, *kibanaURL, dashboard["id"], path.Join(directory, dashboard["file"]))
			if err != nil {
				fmt.Printf("ERROR: fail to export the dashboards: %s\n", err)
			}
		}
		os.Exit(0)
	}

	if len(*dashboard) > 0 {
		err := Export(client, *kibanaURL, *dashboard, *fileOutput)
		if err != nil {
			fmt.Printf("ERROR: fail to export the dashboards: %s\n", err)
		}
	}
}
