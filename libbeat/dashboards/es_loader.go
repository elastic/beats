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

var KibanaApiStartingWith = "6.0.0-alpha2"

type ElasticsearchLoader struct {
	client       *elasticsearch.Client
	config       *DashboardsConfig
	version      string
	msgOutputter *MessageOutputter
}

func NewElasticsearchLoader(cfg *common.Config, dashboardsConfig *DashboardsConfig, msgOutputter *MessageOutputter) (*ElasticsearchLoader, error) {

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

	// initialize the Kibana index
	if err := loader.createKibanaIndex(); err != nil {
		return nil, fmt.Errorf("fail to create the kibana index: %v", err)
	}

	return &loader, nil
}

// CreateKibanaIndex creates the kibana index if it doesn't exists and sets
// some index properties which are needed as a workaround for:
// https://github.com/elastic/beats-dashboards/issues/94
func (loader ElasticsearchLoader) createKibanaIndex() error {
	status, err := loader.client.IndexExists(loader.config.KibanaIndex)

	if err != nil {
		if status != 404 {
			return err
		} else {
			var settings common.MapStr
			// XXX: this can be removed when the dashboard loaded will no longer need to support 6.0,
			// because the Kibana API is used instead
			if strings.HasPrefix(loader.client.GetVersion(), "6.") {
				settings = common.MapStr{
					"settings": common.MapStr{
						"index.mapping.single_type": false,
					},
				}
			} else {
				settings = nil
			}
			_, _, err = loader.client.CreateIndex(loader.config.KibanaIndex, settings)
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
	}

	return nil
}

func (loader ElasticsearchLoader) ImportIndex(file string) error {
	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	var indexContent common.MapStr
	json.Unmarshal(reader, &indexContent)

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
	json.Unmarshal(reader, &jsonContent)
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
	json.Unmarshal(reader, &jsonContent)

	var widgets []panel
	json.Unmarshal([]byte(jsonContent.PanelsJSON), &widgets)

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
	if err := loader.importJSONFile("visualization", file); err != nil {
		return err
	}

	err := loader.importSearchFromVisualization(file)
	if err != nil {
		return err
	}
	return nil
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
		return fmt.Errorf("Failed to unmarshal search content %s: %v", searchName, err)
	}

	if loader.config.Index != "" {

		// change index pattern name
		if savedObject, ok := searchContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
			if source, ok := savedObject["searchSourceJSON"].(string); ok {
				var record common.MapStr
				err = json.Unmarshal([]byte(source), &record)
				if err != nil {
					return fmt.Errorf("Failed to unmarshal searchSourceJSON from search %s: %v", searchName, err)
				}

				if _, ok := record["index"]; ok {
					record["index"] = loader.config.Index
				}
				searchSourceJSON, err := json.Marshal(record)
				if err != nil {
					return fmt.Errorf("Failed to marshal searchSourceJSON: %v", err)
				}

				savedObject["searchSourceJSON"] = string(searchSourceJSON)
			}
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
	json.Unmarshal(reader, &jsonContent)
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
	err = loader.importPanelsFromDashboard(file)
	if err != nil {
		return err
	}

	return nil
}

func (loader ElasticsearchLoader) Close() error {
	return loader.client.Close()
}

func (loader ElasticsearchLoader) statusMsg(msg string, a ...interface{}) {
	if loader.msgOutputter != nil {
		(*loader.msgOutputter)(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}

}
