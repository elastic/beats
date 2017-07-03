package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

var exportAPI = "/api/kibana/dashboards/export"

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

func Export(client *http.Client, conn string, dashboards []string, out string) error {

	params := url.Values{}

	for _, dashboard := range dashboards {
		params.Add("dashboard", dashboard)
	}

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

	fmt.Printf("The dashboards were exported under the %s file\n", out)
	return err
}

func main() {

	kibanaURL := flag.String("kibana", "http://localhost:5601", "Kibana URL")
	fileOutput := flag.String("output", "output.json", "Output file")

	flag.Parse()

	args := flag.Args()

	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ignore expired SSL certificates
	}

	client := &http.Client{Transport: transCfg}

	err := Export(client, *kibanaURL, args, *fileOutput)
	if err != nil {
		fmt.Printf("ERROR: fail to export the dashboards: %s\n", err)
	}
}
