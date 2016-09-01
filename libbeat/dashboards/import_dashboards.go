package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	lbeat "github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

var usage = fmt.Sprintf(`
Usage: ./import_dashboards [options]

Kibana dashboards are stored in a special index in Elasticsearch together with the searches, visualizations, and indexes that they use.

You can import the dashboards, visualizations, searches, and the index pattern for a single Beat (eg. Metricbeat):
  1. from a local directory:
	./import_dashboards -dir kibana/metricbeat
  2. from a local zip archive containing dashboards of multiple Beats:
	./import_dashboards -beat metricbeat -file beats-dashboards-%s.zip
  3. from the official zip archive available under http://download.elastic.co/beats/dashboards/beats-dashboards-%s.zip:
	./import_dashboards -beat metricbeat
  4. from any zip archive available online:
    ./import_dashboards -beat metricbeat -url https://github.com/monicasarbu/metricbeat-dashboards/archive/1.1.zip

To import only the index-pattern for a single Beat (eg. Metricbeat) use:
	./import_dashboards -only-index -beat metricbeat

To import only the dashboards together with visualizations and searches for a single Beat (eg. Metricbeat) use:
	./import_dashboards -only-dashboards -beat metricbeat

Options:
`, lbeat.GetDefaultVersion(), lbeat.GetDefaultVersion())

var beat string

type Options struct {
	KibanaIndex    string
	ES             string
	Index          string
	Dir            string
	File           string
	Beat           string
	Url            string
	User           string
	Pass           string
	OnlyDashboards bool
	OnlyIndex      bool
}

type CommandLine struct {
	flagSet *flag.FlagSet
	opt     Options
}

type Importer struct {
	cl     *CommandLine
	client *elasticsearch.Client
}

func DefineCommandLine() (*CommandLine, error) {
	var cl CommandLine

	cl.flagSet = flag.NewFlagSet("import", flag.ContinueOnError)

	cl.flagSet.Usage = func() {

		os.Stderr.WriteString(usage)
		cl.flagSet.PrintDefaults()
	}

	cl.flagSet.StringVar(&cl.opt.KibanaIndex, "k", ".kibana", "Kibana index")
	cl.flagSet.StringVar(&cl.opt.ES, "es", "http://127.0.0.1:9200", "Elasticsearch URL")
	cl.flagSet.StringVar(&cl.opt.User, "user", "", "Username to connect to Elasticsearch. By default no username is passed.")
	cl.flagSet.StringVar(&cl.opt.Pass, "pass", "", "Password to connect to Elasticsearch. By default no password is passed.")
	cl.flagSet.StringVar(&cl.opt.Index, "i", "", "The Elasticsearch index name. This overwrites the index name defined in the dashboards and index pattern. Example: metricbeat-*")
	cl.flagSet.StringVar(&cl.opt.Dir, "dir", "", "Directory containing the subdirectories: dashboard, visualization, search, index-pattern. Example: etc/kibana/")
	cl.flagSet.StringVar(&cl.opt.File, "file", "", "Zip archive file containing the Beats dashboards. The archive contains a directory for each Beat.")
	cl.flagSet.StringVar(&cl.opt.Url, "url",
		fmt.Sprintf("https://download.elastic.co/beats/dashboards/beats-dashboards-%s.zip", lbeat.GetDefaultVersion()),
		"URL to the zip archive containing the Beats dashboards")
	cl.flagSet.StringVar(&cl.opt.Beat, "beat", beat, "The Beat name, in case a zip archive is passed as input")
	cl.flagSet.BoolVar(&cl.opt.OnlyDashboards, "only-dashboards", false, "Import only dashboards together with visualizations and searches. By default import both, dashboards and the index-pattern.")
	cl.flagSet.BoolVar(&cl.opt.OnlyIndex, "only-index", false, "Import only the index-pattern. By default imports both, dashboards and the index pattern.")

	return &cl, nil
}

func (cl *CommandLine) ParseCommandLine() error {

	cl.opt.Beat = beat

	if err := cl.flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	if cl.opt.Url == "" && cl.opt.File == "" && cl.opt.Dir == "" {
		return errors.New("ERROR: Missing input. Please specify one of the options -file, -url or -dir")
	}

	return nil
}

func New() (*Importer, error) {
	importer := Importer{}

	/* define the command line arguments */
	cl, err := DefineCommandLine()
	if err != nil {
		cl.flagSet.Usage()
		return nil, err
	}
	/* parse command line arguments */
	err = cl.ParseCommandLine()
	if err != nil {
		return nil, err
	}
	importer.cl = cl

	/* prepare the Elasticsearch index pattern */
	fmtstr, err := fmtstr.CompileEvent(cl.opt.Index)
	if err != nil {
		return nil, fmt.Errorf("fail to build the Elasticsearch index pattern: %s", err)
	}
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	/* connect to Elasticsearch */
	client, err := elasticsearch.NewClient(
		elasticsearch.ClientSettings{
			URL:      cl.opt.ES,
			Index:    indexSel,
			Username: cl.opt.User,
			Password: cl.opt.Pass,
			Timeout:  60 * time.Second,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("fail to connect to Elasticsearch: %s", err)
	}
	importer.client = client

	return &importer, nil

}

func (imp Importer) CreateIndex() error {
	imp.client.CreateIndex(imp.cl.opt.KibanaIndex, nil)
	_, _, err := imp.client.CreateIndex(imp.cl.opt.KibanaIndex+"/_mapping/search",
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
		fmt.Printf("fail to set the mapping. Error: %s\n", err)
	}
	return nil
}

func (imp Importer) ImportJsonFile(fileType string, file string) error {

	path := "/" + imp.cl.opt.KibanaIndex + "/" + fileType

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("fail to read %s. Error: %s", file, err)
	}
	var jsonContent map[string]interface{}
	json.Unmarshal(reader, &jsonContent)
	fileBase := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	err = imp.client.LoadJson(path+"/"+fileBase, jsonContent)
	if err != nil {
		return fmt.Errorf("fail to load %s under %s/%s: %s", file, path, fileBase, err)
	}

	return nil
}

func (imp Importer) ImportDashboard(file string) error {

	fmt.Println("Import dashboard ", file)

	/* load dashboard */
	err := imp.ImportJsonFile("dashboard", file)
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
		Id   string `json:"id"`
		Type string `json:"type"`
	}

	var json_content record
	json.Unmarshal(reader, &json_content)

	var widgets []panel
	json.Unmarshal([]byte(json_content.PanelsJSON), &widgets)

	for _, widget := range widgets {

		if widget.Type == "visualization" {
			err = imp.ImportVisualization(path.Join(mainDir, "visualization", widget.Id+".json"))
			if err != nil {
				return err
			}
		} else if widget.Type == "search" {
			err = imp.ImportSearch(path.Join(mainDir, "search", widget.Id+".json"))
			if err != nil {
				return err
			}
		} else {
			fmt.Println(widgets)
			return fmt.Errorf("unknown panel type %s in %s", widget.Type, file)
		}
	}
	return
}

func (imp Importer) importSearchFromVisualization(file string) error {
	type record struct {
		Title         string `json:"title"`
		SavedSearchId string `json:"savedSearchId"`
	}

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	var json_content record
	json.Unmarshal(reader, &json_content)
	id := json_content.SavedSearchId
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

	fmt.Println("Import vizualization ", file)
	if err := imp.ImportJsonFile("visualization", file); err != nil {
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
		return fmt.Errorf("fail to unmarshal search content %s: %v", searchName, err)
	}

	if imp.cl.opt.Index != "" {

		// change index pattern name
		if savedObject, ok := searchContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
			if source, ok := savedObject["searchSourceJSON"].(string); ok {
				var record common.MapStr
				err = json.Unmarshal([]byte(source), &record)
				if err != nil {
					return fmt.Errorf("fail to unmarshal searchSourceJSON from search %s: %v", searchName, err)
				}

				if _, ok := record["index"]; ok {
					record["index"] = imp.cl.opt.Index
				}
				searchSourceJSON, err := json.Marshal(record)
				if err != nil {
					return fmt.Errorf("fail to marshal searchSourceJSON: %v", err)
				}

				savedObject["searchSourceJSON"] = string(searchSourceJSON)
			}
		}

	}

	path := "/" + imp.cl.opt.KibanaIndex + "/search/" + searchName
	fmt.Println("Import search ", file)

	if err = imp.client.LoadJson(path, searchContent); err != nil {
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
		return errors.New("missing title in the index-pattern file")
	}

	if imp.cl.opt.Index != "" {
		// change index pattern name
		fmt.Println("Change index in index-pattern ", indexName)
		indexContent["title"] = imp.cl.opt.Index
	}

	path := "/" + imp.cl.opt.KibanaIndex + "/index-pattern/" + indexName
	fmt.Printf("Import index to %s from %s\n", path, file)

	if err = imp.client.LoadJson(path, indexContent); err != nil {
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
	return fmt.Errorf("unexpected file type %s", fileType)
}

func (imp Importer) ImportDir(dirType string, dir string) error {

	dir = path.Join(dir, dirType)

	// check if the directory exists
	if _, err := os.Stat(dir); err != nil {
		// nothing to import
		fmt.Println("No directory", dir)
		return nil
	}

	fmt.Println("Import directory ", dir)
	errors := []string{}

	files, err := filepath.Glob(path.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("fail to read directory %s. Error: %s", dir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("empty directory %s", dir)
	}
	for _, file := range files {

		err = imp.ImportFile(dirType, file)
		if err != nil {
			fmt.Println("ERROR: ", err)
			errors = append(errors, fmt.Sprintf("error loading %s: %s\n", file, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("fail to load directory %s: %s", dir, strings.Join(errors, ", "))
	}
	return nil

}

func unzip(archive, target string) error {

	fmt.Println("Unzip archive ", target)

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		filePath := filepath.Join(target, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
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
	}
	return nil
}

func getMainDir(target string) (string, error) {

	files, err := ioutil.ReadDir(target)
	if err != nil {
		return "", err
	}
	var dirs []string

	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("too many subdirectories under %s", target)
	}
	return filepath.Join(target, dirs[0]), nil
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

func downloadFile(url string, target string) (string, error) {

	fileName := filepath.Base(url)
	targetPath := path.Join(target, fileName)
	fmt.Println("Downloading", url)

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

func (imp Importer) ImportArchive() error {

	var archive string

	target, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return errors.New("fail to generate the temporary directory")
	}

	if err = os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("fail to create the temporary directory: %v", target)
	}

	defer os.RemoveAll(target) // clean up

	fmt.Println("Create temporary directory", target)
	if imp.cl.opt.File != "" {
		archive = imp.cl.opt.File
	} else if imp.cl.opt.Url != "" {
		// it's an URL
		archive, err = downloadFile(imp.cl.opt.Url, target)
		if err != nil {
			return fmt.Errorf("fail to download file: %s", imp.cl.opt.Url)
		}
	} else {
		return errors.New("No archive file or URL is set. Please use -file or -url option.")
	}

	err = unzip(archive, target)
	if err != nil {
		return fmt.Errorf("fail to unzip the archive: %s", archive)
	}
	dirs, err := getDirectories(target)
	if err != nil {
		return err
	}
	if len(dirs) != 1 {
		return fmt.Errorf("too many directories under %s", target)
	}

	dirs, err = getDirectories(dirs[0])
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		fmt.Println(dir)
		if imp.cl.opt.Beat == "" || filepath.Base(dir) == imp.cl.opt.Beat {
			err = imp.ImportKibana(dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// import Kibana dashboards and index-pattern or only one of these
func (imp Importer) ImportKibana(dir string) error {

	var err error

	if !imp.cl.opt.OnlyDashboards {
		err = imp.ImportDir("index-pattern", dir)
		if err != nil {
			return fmt.Errorf("fail to import index-pattern: %v", err)
		}
	}
	if !imp.cl.opt.OnlyIndex {
		err = imp.ImportDir("dashboard", dir)
		if err != nil {
			return fmt.Errorf("fail to import dashboards: %v", err)
		}
	}
	return nil

}

func main() {

	importer, err := New()
	if err != nil {
		fmt.Println(err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}
	if err := importer.CreateIndex(); err != nil {
		fmt.Println(err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}

	if importer.cl.opt.Dir != "" {
		if err = importer.ImportKibana(importer.cl.opt.Dir); err != nil {
			fmt.Println(err)
		}
	} else {
		if importer.cl.opt.Url != "" || importer.cl.opt.File != "" {
			if err = importer.ImportArchive(); err != nil {
				fmt.Println(err)
			}
		}
	}
}
