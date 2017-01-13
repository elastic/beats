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
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

var usage = fmt.Sprintf(`
Usage: ./import_dashboards [options]

Kibana dashboards are stored in a special index in Elasticsearch together with the searches, visualizations, and indexes that they use.

To import the official Kibana dashboards for your Beat version into a local Elasticsearch instance, use:

	./import_dashboards

To import the official Kibana dashboards for your Beat version into a remote Elasticsearch instance with Shield, use:

	./import_dashboards -es https://xyz.found.io -user user -pass password

For more details, check https://www.elastic.co/guide/en/beats/libbeat/5.0/import-dashboards.html.

`)

var beat string

type Options struct {
	KibanaIndex          string
	ES                   string
	Index                string
	Dir                  string
	File                 string
	Beat                 string
	URL                  string
	User                 string
	Pass                 string
	Certificate          string
	CertificateKey       string
	CertificateAuthority string
	Insecure             bool // Allow insecure SSL connections.
	OnlyDashboards       bool
	OnlyIndex            bool
	Snapshot             bool
	Quiet                bool
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
	cl.flagSet.StringVar(&cl.opt.URL, "url",
		fmt.Sprintf("https://artifacts.elastic.co/downloads/beats/beats-dashboards/beats-dashboards-%s.zip", lbeat.GetDefaultVersion()),
		"URL to the zip archive containing the Beats dashboards")
	cl.flagSet.StringVar(&cl.opt.Beat, "beat", beat, "The Beat name that is used to select what dashboards to install from a zip. An empty string selects all.")
	cl.flagSet.BoolVar(&cl.opt.OnlyDashboards, "only-dashboards", false, "Import only dashboards together with visualizations and searches. By default import both, dashboards and the index-pattern.")
	cl.flagSet.BoolVar(&cl.opt.OnlyIndex, "only-index", false, "Import only the index-pattern. By default imports both, dashboards and the index pattern.")
	cl.flagSet.BoolVar(&cl.opt.Snapshot, "snapshot", false, "Import dashboards from snapshot builds.")
	cl.flagSet.StringVar(&cl.opt.CertificateAuthority, "cacert", "", "Certificate Authority for server verification")
	cl.flagSet.StringVar(&cl.opt.Certificate, "cert", "", "Certificate for SSL client authentication in PEM format.")
	cl.flagSet.StringVar(&cl.opt.CertificateKey, "key", "", "Client Certificate Key in PEM format.")
	cl.flagSet.BoolVar(&cl.opt.Insecure, "insecure", false, `Allows "insecure" SSL connections`)
	cl.flagSet.BoolVar(&cl.opt.Quiet, "quiet", false, "Suppresses all status messages. Error messages are still printed to stderr.")

	return &cl, nil
}

func (cl *CommandLine) ParseCommandLine() error {

	cl.opt.Beat = beat

	if err := cl.flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	if cl.opt.URL == "" && cl.opt.File == "" && cl.opt.Dir == "" {
		return errors.New("Missing input. Please specify one of the options -file, -url or -dir")
	}

	if cl.opt.Certificate != "" && cl.opt.CertificateKey == "" {
		return errors.New("A certificate key needs to be passed as well by using the -key option.")
	}

	if cl.opt.CertificateKey != "" && cl.opt.Certificate == "" {
		return errors.New("A certificate needs to be passed as well by using the -cert option.")
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
		return nil, fmt.Errorf("Failed to build the Elasticsearch index pattern: %s", err)
	}
	indexSel := outil.MakeSelector(outil.FmtSelectorExpr(fmtstr, ""))

	var tlsConfig outputs.TLSConfig
	var tls *transport.TLSConfig

	if cl.opt.Insecure {
		tlsConfig.VerificationMode = transport.VerifyNone
	}

	if len(cl.opt.Certificate) > 0 && len(cl.opt.CertificateKey) > 0 {
		tlsConfig.Certificate = outputs.CertificateConfig{
			Certificate: cl.opt.Certificate,
			Key:         cl.opt.CertificateKey,
		}
	}

	if len(cl.opt.CertificateAuthority) > 0 {
		tlsConfig.CAs = []string{cl.opt.CertificateAuthority}
	}

	tls, err = outputs.LoadTLSConfig(&tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to load the SSL certificate: %s", err)
	}

	/* connect to Elasticsearch */
	client, err := elasticsearch.NewClient(
		elasticsearch.ClientSettings{
			URL:      cl.opt.ES,
			Index:    indexSel,
			TLS:      tls,
			Username: cl.opt.User,
			Password: cl.opt.Pass,
			Timeout:  60 * time.Second,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to Elasticsearch: %s", err)
	}
	importer.client = client

	return &importer, nil

}

func (imp Importer) statusMsg(msg string, a ...interface{}) {
	if imp.cl.opt.Quiet {
		return
	}

	if len(a) == 0 {
		fmt.Println(msg)
	} else {
		fmt.Println(fmt.Sprintf(msg, a...))
	}
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
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Failed to set the mapping - %s", err))
	}
	return nil
}

func (imp Importer) ImportJSONFile(fileType string, file string) error {

	path := "/" + imp.cl.opt.KibanaIndex + "/" + fileType

	reader, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Failed to read %s. Error: %s", file, err)
	}
	var jsonContent map[string]interface{}
	json.Unmarshal(reader, &jsonContent)
	fileBase := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	err = imp.client.LoadJSON(path+"/"+fileBase, jsonContent)
	if err != nil {
		return fmt.Errorf("Failed to load %s under %s/%s: %s", file, path, fileBase, err)
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

	if imp.cl.opt.Index != "" {

		// change index pattern name
		if savedObject, ok := searchContent["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
			if source, ok := savedObject["searchSourceJSON"].(string); ok {
				var record common.MapStr
				err = json.Unmarshal([]byte(source), &record)
				if err != nil {
					return fmt.Errorf("Failed to unmarshal searchSourceJSON from search %s: %v", searchName, err)
				}

				if _, ok := record["index"]; ok {
					record["index"] = imp.cl.opt.Index
				}
				searchSourceJSON, err := json.Marshal(record)
				if err != nil {
					return fmt.Errorf("Failed to marshal searchSourceJSON: %v", err)
				}

				savedObject["searchSourceJSON"] = string(searchSourceJSON)
			}
		}

	}

	path := "/" + imp.cl.opt.KibanaIndex + "/search/" + searchName
	imp.statusMsg("Import search %s", file)

	if err = imp.client.LoadJSON(path, searchContent); err != nil {
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

	if imp.cl.opt.Index != "" {
		// change index pattern name
		imp.statusMsg("Change index in index-pattern %s", indexName)
		indexContent["title"] = imp.cl.opt.Index
	}

	path := "/" + imp.cl.opt.KibanaIndex + "/index-pattern/" + indexName
	fmt.Printf("Import index to %s from %s\n", path, file)

	if err = imp.client.LoadJSON(path, indexContent); err != nil {
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
		return "", fmt.Errorf("Too many subdirectories under %s", target)
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

func (imp Importer) ImportArchive() error {

	var archive string

	target, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return errors.New("Failed to generate a temporary directory name")
	}

	if err = os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("Failed to create a temporary directory: %v", target)
	}

	defer os.RemoveAll(target) // clean up

	imp.statusMsg("Create temporary directory %s", target)
	if imp.cl.opt.File != "" {
		archive = imp.cl.opt.File
	} else if imp.cl.opt.Snapshot {
		// In case snapshot is set, snapshot version is fetched
		url := fmt.Sprintf("https://beats-nightlies.s3.amazonaws.com/dashboards/beats-dashboards-%s-SNAPSHOT.zip", lbeat.GetDefaultVersion())
		archive, err = imp.downloadFile(url, target)
		if err != nil {
			return fmt.Errorf("Failed to download snapshot file: %s", url)
		}
	} else if imp.cl.opt.URL != "" {
		archive, err = imp.downloadFile(imp.cl.opt.URL, target)
		if err != nil {
			return fmt.Errorf("Failed to download file: %s", imp.cl.opt.URL)
		}
	} else {
		return errors.New("No archive file or URL is set - please use -file or -url option")
	}

	err = imp.unzip(archive, target)
	if err != nil {
		return fmt.Errorf("Failed to unzip the archive: %s", archive)
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
		if imp.cl.opt.Beat == "" || filepath.Base(dir) == imp.cl.opt.Beat {
			err = imp.ImportKibana(dir)
			if err != nil {
				return err
			}
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

// import Kibana dashboards and index-pattern or only one of these
func (imp Importer) ImportKibana(dir string) error {

	var err error

	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("No directory %s", dir)
	}

	check := []string{}
	if !imp.cl.opt.OnlyDashboards {
		check = append(check, "index-pattern")
	}
	if !imp.cl.opt.OnlyIndex {
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

func main() {

	importer, err := New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Exiting")
		os.Exit(1)
	}
	if err := importer.CreateIndex(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Exiting")
		os.Exit(1)
	}

	if importer.cl.opt.Dir != "" {
		if err = importer.ImportKibana(importer.cl.opt.Dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(os.Stderr, "Exiting")
			os.Exit(1)
		}
	} else {
		if importer.cl.opt.URL != "" || importer.cl.opt.File != "" {
			if err = importer.ImportArchive(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				fmt.Fprintln(os.Stderr, "Exiting")
				os.Exit(1)
			}
		}
	}
}
