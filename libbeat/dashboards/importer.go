package dashboards

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// MessageOutputter is a function type for injecting status logging
// into this module.
type MessageOutputter func(msg string, a ...interface{})

type Importer struct {
	cfg          *DashboardsConfig
	client       DashboardLoader
	msgOutputter *MessageOutputter
}

func NewImporter(cfg *DashboardsConfig, client DashboardLoader, msgOutputter *MessageOutputter) (*Importer, error) {
	return &Importer{
		cfg:          cfg,
		client:       client,
		msgOutputter: msgOutputter,
	}, nil
}

func (imp Importer) statusMsg(msg string, a ...interface{}) {
	if imp.msgOutputter != nil {
		(*imp.msgOutputter)(msg, a...)
	} else {
		logp.Debug("dashboards", msg, a...)
	}
}

// Import imports the Kibana dashboards according to the configuration options.
func (imp Importer) Import() error {

	err := imp.CreateKibanaIndex()
	if err != nil {
		return fmt.Errorf("Error creating Kibana index: %v", err)
	}

	if imp.cfg.Dir != "" {
		err = imp.ImportKibana(imp.cfg.Dir)
		if err != nil {
			return fmt.Errorf("Error importing directory %s: %v", imp.cfg.Dir, err)
		}
	} else {
		if imp.cfg.URL != "" || imp.cfg.Snapshot || imp.cfg.File != "" {
			err = imp.ImportArchive()
			if err != nil {
				return fmt.Errorf("Error importing URL/file: %v", err)
			}
		} else {
			return fmt.Errorf("No URL and no file specify. Nothing to import")
		}
	}
	return nil
}

// CreateKibanaIndex creates the kibana index if it doesn't exists and sets
// some index properties which are needed as a workaround for:
// https://github.com/elastic/beats-dashboards/issues/94
func (imp Importer) CreateKibanaIndex() error {
	imp.client.CreateIndex(imp.cfg.KibanaIndex,
		common.MapStr{
			"settings": common.MapStr{
				"index.mapping.single_type": false,
			},
		})
	_, _, err := imp.client.CreateIndex(imp.cfg.KibanaIndex+"/_mapping/search",
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
		imp.statusMsg("Failed to set the mapping: %v", err)
	}
	return nil
}

func (imp Importer) ImportJSONFile(fileType string, file string) error {

	path := "/" + imp.cfg.KibanaIndex + "/" + fileType

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Failed to read %s. Error: %s", file, err)
	}
	var jsonContent map[string]interface{}
	json.Unmarshal(reader, &jsonContent)
	fileBase := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	body, err := imp.client.LoadJSON(path+"/"+fileBase, jsonContent)
	if err != nil {
		return fmt.Errorf("Failed to load %s under %s/%s: %s. Response body: %s", file, path, fileBase, err, body)
	}

	return nil
}

func (imp Importer) ImportDashboard(file string) error {

	imp.statusMsg("Import dashboard %s", file)

	/* load dashboard */
	err := imp.ImportJSONFile("dashboard", file)
	if err != nil {
		return err
	}

	/* load the visualizations and searches that depend on the dashboard */
	err = imp.importPanelsFromDashboard(file)
	if err != nil {
		return err
	}

	return nil
}

func (imp Importer) importPanelsFromDashboard(file string) (err error) {

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
			err = imp.ImportVisualization(path.Join(mainDir, "visualization", widget.ID+".json"))
			if err != nil {
				return err
			}
		} else if widget.Type == "search" {
			err = imp.ImportSearch(path.Join(mainDir, "search", widget.ID+".json"))
			if err != nil {
				return err
			}
		} else {
			imp.statusMsg("Widgets: %v", widgets)
			return fmt.Errorf("Unknown panel type %s in %s", widget.Type, file)
		}
	}
	return
}

func (imp Importer) importSearchFromVisualization(file string) error {
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
		if err := imp.ImportSearch(searchFile); err != nil {
			return err
		}
	}
	return nil
}

func (imp Importer) ImportVisualization(file string) error {

	imp.statusMsg("Import visualization %s", file)
	if err := imp.ImportJSONFile("visualization", file); err != nil {
		return err
	}

	err := imp.importSearchFromVisualization(file)
	if err != nil {
		return err
	}
	return nil
}

func (imp Importer) ImportSearch(file string) error {

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

	if imp.cfg.Index != "" {

		// change index pattern name
		if savedObject, ok := searchContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
			if source, ok := savedObject["searchSourceJSON"].(string); ok {
				var record common.MapStr
				err = json.Unmarshal([]byte(source), &record)
				if err != nil {
					return fmt.Errorf("Failed to unmarshal searchSourceJSON from search %s: %v", searchName, err)
				}

				if _, ok := record["index"]; ok {
					record["index"] = imp.cfg.Index
				}
				searchSourceJSON, err := json.Marshal(record)
				if err != nil {
					return fmt.Errorf("Failed to marshal searchSourceJSON: %v", err)
				}

				savedObject["searchSourceJSON"] = string(searchSourceJSON)
			}
		}

	}

	path := "/" + imp.cfg.KibanaIndex + "/search/" + searchName
	imp.statusMsg("Import search %s", file)

	if _, err = imp.client.LoadJSON(path, searchContent); err != nil {
		return err
	}

	return nil
}

func (imp Importer) ImportIndex(file string) error {

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	var indexContent common.MapStr
	json.Unmarshal(reader, &indexContent)

	indexName, ok := indexContent["title"].(string)
	if !ok {
		return errors.New(fmt.Sprintf("Missing title in the index-pattern file at %s", file))
	}

	if imp.cfg.Index != "" {
		// change index pattern name
		imp.statusMsg("Change index in index-pattern %s", indexName)
		indexContent["title"] = imp.cfg.Index
	}

	path := "/" + imp.cfg.KibanaIndex + "/index-pattern/" + indexName
	imp.statusMsg("Import index to %s from %s\n", path, file)

	if _, err = imp.client.LoadJSON(path, indexContent); err != nil {
		return err
	}
	return nil

}

func (imp Importer) ImportFile(fileType string, file string) error {

	if fileType == "dashboard" {
		return imp.ImportDashboard(file)
	} else if fileType == "index-pattern" {
		return imp.ImportIndex(file)
	}
	return fmt.Errorf("Unexpected file type %s", fileType)
}

func (imp Importer) ImportDir(dirType string, dir string) error {

	dir = path.Join(dir, dirType)

	imp.statusMsg("Import directory %s", dir)
	errors := []string{}

	files, err := filepath.Glob(path.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("Failed to read directory %s. Error: %s", dir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("The directory %s is empty, nothing to import", dir)
	}
	for _, file := range files {

		err = imp.ImportFile(dirType, file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("  error loading %s: %s", file, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("Failed to load directory %s:\n%s", dir, strings.Join(errors, "\n"))
	}
	return nil

}

func (imp Importer) unzip(archive, target string) error {

	imp.statusMsg("Unzip archive %s", target)

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	// Closure to close the files on each iteration
	unzipFile := func(file *zip.File) error {
		filePath := filepath.Join(target, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			return nil
		}
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
		return nil
	}

	for _, file := range reader.File {
		err := unzipFile(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func (imp Importer) ImportArchive() error {

	var archive string

	target, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return fmt.Errorf("Failed to generate a temporary directory name: %v", err)
	}

	if err = os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("Failed to create a temporary directory %s: %v", target, err)
	}

	defer os.RemoveAll(target) // clean up

	imp.statusMsg("Created temporary directory %s", target)
	if imp.cfg.File != "" {
		archive = imp.cfg.File
	} else if imp.cfg.Snapshot {
		// In case snapshot is set, snapshot version is fetched
		url := imp.cfg.SnapshotURL
		archive, err = imp.downloadFile(url, target)
		if err != nil {
			return fmt.Errorf("Failed to download snapshot file: %s. Error: %v", url, err)
		}
	} else if imp.cfg.URL != "" {
		archive, err = imp.downloadFile(imp.cfg.URL, target)
		if err != nil {
			return fmt.Errorf("Failed to download file: %s. Error: %v", imp.cfg.URL, err)
		}
	} else {
		return errors.New("No archive file or URL is set - please use -file or -url option")
	}

	err = imp.unzip(archive, target)
	if err != nil {
		return fmt.Errorf("Failed to unzip the archive: %s: %v", archive, err)
	}
	dirs, err := getDirectories(target)
	if err != nil {
		return err
	}
	if len(dirs) != 1 {
		return fmt.Errorf("Too many directories under %s", target)
	}

	dirs, err = getDirectories(dirs[0])
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		imp.statusMsg("Importing Kibana from %s", dir)
		if imp.cfg.Beat == "" || filepath.Base(dir) == imp.cfg.Beat {
			err = imp.ImportKibana(dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getDirectories(target string) ([]string, error) {

	files, err := ioutil.ReadDir(target)
	if err != nil {
		return nil, err
	}
	var dirs []string

	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, filepath.Join(target, file.Name()))
		}
	}
	return dirs, nil
}

func (imp Importer) downloadFile(url string, target string) (string, error) {

	fileName := filepath.Base(url)
	targetPath := path.Join(target, fileName)
	imp.statusMsg("Downloading %s", url)

	// Create the file
	out, err := os.Create(targetPath)
	if err != nil {
		return targetPath, err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return targetPath, err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return targetPath, err
	}

	return targetPath, nil
}

// import Kibana dashboards and index-pattern or only one of these
func (imp Importer) ImportKibana(dir string) error {

	var err error

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("No directory %s", dir)
	}

	check := []string{}
	if !imp.cfg.OnlyDashboards {
		check = append(check, "index-pattern")
	}
	if !imp.cfg.OnlyIndex {
		check = append(check, "dashboard")
	}

	types := []string{}
	for _, c := range check {
		if imp.subdirExists(dir, c) {
			types = append(types, c)
		}
	}

	if len(types) == 0 {
		return fmt.Errorf("The directory %s does not contain the %s subdirectory."+
			" There is nothing to import into Kibana.", dir, strings.Join(check, " or "))
	}

	for _, t := range types {
		err = imp.ImportDir(t, dir)
		if err != nil {
			return fmt.Errorf("Failed to import %s: %v", t, err)
		}
	}
	return nil
}

func (imp Importer) subdirExists(parent string, child string) bool {
	if _, err := os.Stat(path.Join(parent, child)); err != nil {
		return false
	}
	return true
}
