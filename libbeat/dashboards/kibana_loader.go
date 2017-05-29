package dashboards

import (
	"bytes"
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
	config       *DashboardsConfig
	version      string
	msgOutputter *MessageOutputter
}

func NewKibanaLoader(cfg *common.Config, dashboardsConfig *DashboardsConfig, msgOutputter *MessageOutputter) (*KibanaLoader, error) {

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
		return fmt.Errorf("fail to read index-pattern: %v", err)
	}

	return loader.client.ImportJSON(importAPI, params, bytes.NewBuffer(content))
}

func (loader KibanaLoader) Close() error {
	return loader.client.Close()
}

func (loader KibanaLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		(*loader.msgOutputter)(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}
