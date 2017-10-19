package dashboards

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

type ElasticsearchLoader struct {
	client       *elasticsearch.Client
	config       *Config
	version      string
	msgOutputter MessageOutputter
}

func NewElasticsearchLoader(cfg *common.Config, dashboardsConfig *Config, msgOutputter MessageOutputter) (*ElasticsearchLoader, error) {
	if cfg == nil || !cfg.Enabled() {
		return nil, fmt.Errorf("Elasticsearch output is not configured/enabled")
	}

	esClient, err := elasticsearch.NewConnectedClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("Error creating Elasticsearch client: %v", err)
	}

	version := esClient.GetVersion()

	loader := ElasticsearchLoader{
		client:       esClient,
		config:       dashboardsConfig,
		version:      version,
		msgOutputter: msgOutputter,
	}

	loader.statusMsg("Initialize the Elasticsearch %s loader", version)

	return &loader, nil
}

// CreateKibanaIndex creates the kibana index if it doesn't exists and sets
// some index properties which are needed as a workaround for:
// https://github.com/elastic/beats-dashboards/issues/94
func (loader ElasticsearchLoader) CreateKibanaIndex() error {
	status, err := loader.client.IndexExists(loader.config.KibanaIndex)

	if err != nil {
		if status != 404 {
			return err
		}

		_, _, err = loader.client.CreateIndex(loader.config.KibanaIndex, nil)
		if err != nil {
			return fmt.Errorf("Failed to create index: %v", err)
		}

		_, _, err = loader.client.CreateIndex(loader.config.KibanaIndex+"/_mapping/search",
			common.MapStr{
				"search": common.MapStr{
					"properties": common.MapStr{
						"hits": common.MapStr{
							"type": "integer",
						},
						"version": common.MapStr{
							"type": "integer",
						},
					},
				},
			})
		if err != nil {
			return fmt.Errorf("Failed to set the mapping: %v", err)
		}
	}

	return nil
}

func (loader ElasticsearchLoader) ImportIndex(file string) error {
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	var indexContent common.MapStr
	err = json.Unmarshal(reader, &indexContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal index content: %v", err)
	}

	indexName, ok := indexContent["title"].(string)
	if !ok {
		return fmt.Errorf("Missing title in the index-pattern file at %s", file)
	}

	if loader.config.Index != "" {
		// change index pattern name
		loader.statusMsg("Change index in index-pattern %s", indexName)
		indexContent["title"] = loader.config.Index
	}

	path := "/" + loader.config.KibanaIndex + "/index-pattern/" + indexName

	if _, err = loader.client.LoadJSON(path, indexContent); err != nil {
		return err
	}

	return nil
}

func (loader ElasticsearchLoader) importJSONFile(fileType string, file string) error {
	path := "/" + loader.config.KibanaIndex + "/" + fileType

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Failed to read %s. Error: %s", file, err)
	}
	var jsonContent map[string]interface{}
	err = json.Unmarshal(reader, &jsonContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal json file: %v", err)
	}

	fileBase := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	body, err := loader.client.LoadJSON(path+"/"+fileBase, jsonContent)
	if err != nil {
		return fmt.Errorf("Failed to load %s under %s/%s: %s. Response body: %s", file, path, fileBase, err, body)
	}

	return nil
}

func (loader ElasticsearchLoader) importPanelsFromDashboard(file string) (err error) {
	// directory with the dashboards
	dir := filepath.Dir(file)

	// main directory with dashboard, search, visualizations directories
	mainDir := filepath.Dir(dir)

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	type record struct {
		Title      string `json:"title"`
		PanelsJSON string `json:"panelsJSON"`
	}
	type panel struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}

	var jsonContent record
	err = json.Unmarshal(reader, &jsonContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal json content: %v", err)
	}

	var widgets []panel
	err = json.Unmarshal([]byte(jsonContent.PanelsJSON), &widgets)
	if err != nil {
		return fmt.Errorf("fail to unmarshal panels content: %v", err)
	}

	for _, widget := range widgets {
		if widget.Type == "visualization" {
			err = loader.importVisualization(path.Join(mainDir, "visualization", widget.ID+".json"))
			if err != nil {
				return err
			}
		} else if widget.Type == "search" {
			err = loader.importSearch(path.Join(mainDir, "search", widget.ID+".json"))
			if err != nil {
				return err
			}
		} else {
			loader.statusMsg("Widgets: %v", widgets)
			return fmt.Errorf("Unknown panel type %s in %s", widget.Type, file)
		}
	}
	return
}

func (loader ElasticsearchLoader) importVisualization(file string) error {
	loader.statusMsg("Import visualization %s", file)
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	var vizContent common.MapStr
	err = json.Unmarshal(reader, &vizContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal visualization content %s: %v", file, err)
	}

	if loader.config.Index != "" {
		if savedObject, ok := vizContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {

			vizContent["kibanaSavedObjectMeta"] = ReplaceIndexInSavedObject(loader.config.Index, savedObject)
		}
	}

	vizName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	path := "/" + loader.config.KibanaIndex + "/visualization/" + vizName
	if _, err := loader.client.LoadJSON(path, vizContent); err != nil {
		return err
	}

	return loader.importSearchFromVisualization(file)
}

func (loader ElasticsearchLoader) importSearch(file string) error {
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	searchName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	var searchContent common.MapStr
	err = json.Unmarshal(reader, &searchContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal search content %s: %v", searchName, err)
	}

	if loader.config.Index != "" {

		// change index pattern name
		if savedObject, ok := searchContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {

			searchContent["kibanaSavedObjectMeta"] = ReplaceIndexInSavedObject(loader.config.Index, savedObject)

		}
	}

	path := "/" + loader.config.KibanaIndex + "/search/" + searchName
	loader.statusMsg("Import search %s", file)

	if _, err = loader.client.LoadJSON(path, searchContent); err != nil {
		return err
	}

	return nil
}

func (loader ElasticsearchLoader) importSearchFromVisualization(file string) error {
	type record struct {
		Title         string `json:"title"`
		SavedSearchID string `json:"savedSearchId"`
	}

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	var jsonContent record
	err = json.Unmarshal(reader, &jsonContent)
	if err != nil {
		return fmt.Errorf("fail to unmarshal the search content: %v", err)
	}

	id := jsonContent.SavedSearchID
	if len(id) == 0 {
		// no search used
		return nil
	}

	// directory with the visualizations
	dir := filepath.Dir(file)

	// main directory
	mainDir := filepath.Dir(dir)

	searchFile := path.Join(mainDir, "search", id+".json")

	if searchFile != "" {
		// visualization depends on search
		if err := loader.importSearch(searchFile); err != nil {
			return err
		}
	}
	return nil
}

func (loader ElasticsearchLoader) ImportDashboard(file string) error {
	/* load dashboard */
	err := loader.importJSONFile("dashboard", file)
	if err != nil {
		return err
	}

	/* load the visualizations and searches that depend on the dashboard */
	return loader.importPanelsFromDashboard(file)
}

func (loader ElasticsearchLoader) Close() error {
	return loader.client.Close()
}

func (loader ElasticsearchLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		loader.msgOutputter(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}
