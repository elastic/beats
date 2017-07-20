package dashboards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/setup/kibana"
	"github.com/pkg/errors"
)

var importAPI = "/api/kibana/dashboards/import"
var exportAPI = "/api/kibana/dashboards/export"

type KibanaLoader struct {
	client       *kibana.Client
	config       *Config
	version      string
	msgOutputter MessageOutputter
}

func NewKibanaLoader(cfg *common.Config, dashboardsConfig *Config, msgOutputter MessageOutputter) (*KibanaLoader, error) {
	client, err := kibana.NewKibanaClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("Error creating Kibana client: %v", err)
	}

	loader := KibanaLoader{
		client:       client,
		config:       dashboardsConfig,
		version:      client.GetVersion(),
		msgOutputter: msgOutputter,
	}

	loader.statusMsg("Initialize the Kibana %s loader", client.GetVersion())

	return &loader, nil
}

func (loader KibanaLoader) ImportIndex(file string) error {
	params := url.Values{}
	params.Set("force", "true") //overwrite the existing dashboards

	// read json file
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read index-pattern: %v", err)
	}

	return loader.client.ImportJSON(importAPI, params, bytes.NewBuffer(content))
}

func (loader KibanaLoader) ImportDashboard(file string) error {
	params := url.Values{}
	params.Set("force", "true")            //overwrite the existing dashboards
	params.Add("exclude", "index-pattern") //don't import the index pattern from the dashboards

	// read json file
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "fail to read index-pattern")
	}

	return loader.client.ImportJSON(importAPI, params, bytes.NewBuffer(content))
}

func (loader KibanaLoader) Close() error {
	return loader.client.Close()
}

func (loader KibanaLoader) ExportDashboard(dashboard string, out io.Writer) error {
	params := url.Values{}
	params.Set("dashboard", dashboard)

	status, body, err := loader.client.Request("GET", exportAPI, params, nil)
	if err != nil {
		return errors.Wrap(err, "error exporting dashboard")
	}

	if status != 200 {
		return fmt.Errorf("HTTP GET failed with %d, %s", status, body)
	}

	body, err = extractIndexPattern(body)
	if err != nil {
		return fmt.Errorf("fail to extract the index pattern: %v", err)
	}

	_, err = out.Write(body)
	return err
}

func (loader KibanaLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		loader.msgOutputter(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}

func extractIndexPattern(body []byte) ([]byte, error) {
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
		return nil, errors.Wrap(err, "error mashaling")
	}

	return newBody, nil
}
