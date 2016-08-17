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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

const usage = `
Usage: ./import_dashboards [options]

Kibana dashboards are stored in a special index in Elasticsearch together with the searches, visualizations, and indexes that they use.

You can import the dashboards, visualizations, searches, and the index pattern for any Beat:
  1. from a local directory:
	./import_dashboards -dir etc/kibana
  2. from a directory of a local zip archive:
	./import_dashboards -dir metricbeat -file beats-dashboards-1.2.3.zip
  3. from a directory of zip archive available online:
	./import_dashboards -dir metricbeat -url http://download.elastic.co/beats/dashboards/beats-dashboards-1.2.3.zip

Options:
`

type Options struct {
	KibanaIndex string
	ES          string
	Index       string
	Dir         string
	File        string
	Beat        string
	Url         string
	User        string
	Pass        string
}

type CommandLine struct {
	flagSet *flag.FlagSet
	opt     Options
}

type Importer struct {
	cl     *CommandLine
	client *elasticsearch.Client
}

func ParseCommandLine() (*CommandLine, error) {
	var cl CommandLine

	cl.flagSet = flag.NewFlagSet("import", flag.ContinueOnError)

	cl.flagSet.Usage = func() {

		os.Stderr.WriteString(usage)
		cl.flagSet.PrintDefaults()
	}

	cl.flagSet.StringVar(&cl.opt.KibanaIndex, "k", ".kibana", "Kibana index")
	cl.flagSet.StringVar(&cl.opt.ES, "es", "http://127.0.0.1:9200", "Elasticsearch URL")
	cl.flagSet.StringVar(&cl.opt.User, "user", "", "Username to connect to Elasticsearch")
	cl.flagSet.StringVar(&cl.opt.Pass, "pass", "", "Password to connect to Elasticsearch")
	cl.flagSet.StringVar(&cl.opt.Index, "i", "", "Overwrites the Elasticsearch index name. For example you can replaces metricbeat-* with custombeat-*")
	cl.flagSet.StringVar(&cl.opt.Dir, "dir", "", "Directory containing the subdirectories: dashboard, visualization, search, index-pattern. eg. kibana/")
	cl.flagSet.StringVar(&cl.opt.File, "file", "", "Zip archive file containing the Beats dashboards.")
	cl.flagSet.StringVar(&cl.opt.Url, "url", "", "URL to the zip archive containing the Beats dashboards")

	return &cl, nil
}

func (cl *CommandLine) Read() error {

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
	cl, err := ParseCommandLine()
	if err != nil {
		cl.flagSet.Usage()
		return nil, err
	}
	/* parse command line arguments */
	err = cl.Read()
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
	var indexName string
	json.Unmarshal(reader, &indexContent)

	if imp.cl.opt.Index != "" {
		// change index pattern name
		indexName = strings.Trim(imp.cl.opt.Index, "-*")
		if _, ok := indexContent["title"]; ok {
			fmt.Println("Change index in index-pattern ", indexContent["title"])
			indexContent["title"] = imp.cl.opt.Index
		}
	} else {
		// keep the index pattern file name
		indexName = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
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
		return err
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

func unzip(archive, target string) (string, error) {

	archiveName := filepath.Base(archive)
	dirName := archiveName[:len(archiveName)-len(filepath.Ext(archiveName))]
	dir := path.Join(target, dirName)
	fmt.Println("Unzip archive to ", dir)

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return dir, err
	}

	for _, file := range reader.File {
		filePath := filepath.Join(target, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}
		fileReader, err := file.Open()
		if err != nil {
			return dir, err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return dir, err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return dir, err
		}
	}
	return dir, nil
}

func downloadFile(url string, target string) (string, error) {

	fileName := filepath.Base(url)
	targetPath := path.Join(target, fileName)
	fmt.Println("Download file ", fileName, " to ", targetPath)

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

func (imp Importer) ImportArchive() (err error) {

	var archive string

	target, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return
	}

	if err = os.MkdirAll(target, 0755); err != nil {
		return
	}

	//defer os.RemoveAll(target) // clean up

	if imp.cl.opt.Url != "" {
		// it's an URL
		archive, err = downloadFile(imp.cl.opt.Url, target)
		if err != nil {
			return
		}
	} else if imp.cl.opt.File != "" {
		archive = imp.cl.opt.File
	} else {
		return errors.New("No archive file or URL is set. Please use -file or -url option.")
	}

	dir, err := unzip(archive, target)
	if err != nil {
		return
	}
	err = imp.ImportDir("index-pattern", path.Join(dir, imp.cl.opt.Dir))
	if err != nil {
		return
	}
	err = imp.ImportDir("dashboard", path.Join(dir, imp.cl.opt.Dir))
	if err != nil {
		return
	}
	return
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

	fmt.Println(importer.cl.opt.Index)

	if importer.cl.opt.Url != "" || importer.cl.opt.File != "" {
		if err = importer.ImportArchive(); err != nil {
			fmt.Println(err)
		}

	} else if importer.cl.opt.Dir != "." {
		if err = importer.ImportDir("index-pattern", importer.cl.opt.Dir); err != nil {
			fmt.Println(err)
		}
		if err = importer.ImportDir("dashboard", importer.cl.opt.Dir); err != nil {
			fmt.Println(err)
		}
	}

}
