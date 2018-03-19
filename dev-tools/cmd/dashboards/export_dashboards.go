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

func ExtractIndexPattern(body []byte) ([]byte, error) {
	var contents common.MapStr

	err := json.Unmarshal(body, &contents)
	if err != nil {
		return nil, err
	}

	objects, ok := contents["objects"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Key objects not found or wrong type")
	}

	var result []interface{}
	for _, obj := range objects {
		_type, ok := obj.(map[string]interface{})["type"].(string)
		if !ok {
			return nil, fmt.Errorf("type key not found or not string")
		}
		if _type != "index-pattern" {
			result = append(result, obj)
		}
	}
	contents["objects"] = result

	newBody, err := json.MarshalIndent(contents, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Error mashaling: %v", err)
	}

	return newBody, nil
}

func Export(client *http.Client, conn string, dashboard string, out string) error {
	params := url.Values{}

	params.Add("dashboard", dashboard)

	fullURL := makeURL(conn, exportAPI, params)
	fmt.Printf("Calling HTTP GET %v\n", fullURL)

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

	body, err = ExtractIndexPattern(body)
	if err != nil {
		return fmt.Errorf("fail to extract the index pattern: %v", err)
	}

	err = ioutil.WriteFile(out, body, 0666)

	fmt.Printf("The dashboard %s was exported under the %s file\n", dashboard, out)
	return err
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

func main() {
	kibanaURL := flag.String("kibana", "http://localhost:5601", "Kibana URL")
	dashboard := flag.String("dashboard", "", "Dashboard ID")
	fileOutput := flag.String("output", "output.json", "Output file")
	ymlFile := flag.String("yml", "", "Path to the module.yml file containing the dashboards")

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
