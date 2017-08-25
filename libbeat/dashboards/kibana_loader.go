package dashboards

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/setup/kibana"
)

var importAPI = "/api/kibana/dashboards/import"

type KibanaLoader struct {
	client       *kibana.Client
	config       *Config
	version      string
	msgOutputter MessageOutputter
}

func NewKibanaLoader(cfg *common.Config, dashboardsConfig *Config, msgOutputter MessageOutputter) (*KibanaLoader, error) {

	if cfg == nil || !cfg.Enabled() {
		return nil, fmt.Errorf("Kibana is not configured or enabled")
	}

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
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read index-pattern: %v", err)
	}

	var indexContent common.MapStr
	err = json.Unmarshal(reader, &indexContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the index content: %v", err)
	}

	indexContent = ReplaceIndexInIndexPattern(loader.config.Index, indexContent)

	return loader.client.ImportJSON(importAPI, params, indexContent)
}

func (loader KibanaLoader) ImportDashboard(file string) error {
	params := url.Values{}
	params.Set("force", "true")            //overwrite the existing dashboards
	params.Add("exclude", "index-pattern") //don't import the index pattern from the dashboards

	// read json file
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read index-pattern: %v", err)
	}
	var content common.MapStr
	err = json.Unmarshal(reader, &content)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the index content: %v", err)
	}

	content = ReplaceIndexInDashboardObject(loader.config.Index, content)

	return loader.client.ImportJSON(importAPI, params, content)
}

func (loader KibanaLoader) Close() error {
	return loader.client.Close()
}

func (loader KibanaLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		loader.msgOutputter(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}
