package dashboards

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// MessageOutputter is a function type for injecting status logging
// into this module.
type MessageOutputter func(msg string, a ...interface{})

type Importer struct {
	cfg     *Config
	version common.Version

	loader Loader
}

type Loader interface {
	ImportIndex(file string) error
	ImportDashboard(file string) error
	statusMsg(msg string, a ...interface{})
	Close() error
}

func NewImporter(version common.Version, cfg *Config, loader Loader) (*Importer, error) {

	// Current max version is 6
	if version.Major > 6 {
		version.Major = 6
	}

	return &Importer{
		cfg:     cfg,
		version: version,
		loader:  loader,
	}, nil
}

// Import imports the Kibana dashboards according to the configuration options.
func (imp Importer) Import() error {
	if imp.cfg.URL != "" || imp.cfg.File != "" {
		err := imp.ImportArchive()
		if err != nil {
			return fmt.Errorf("Error importing URL/file: %v", err)
		}
	} else {
		err := imp.ImportKibanaDir(imp.cfg.Dir)
		if err != nil {
			return fmt.Errorf("Error importing directory %s: %v", imp.cfg.Dir, err)
		}
	}
	return nil
}

func (imp Importer) ImportDashboard(file string) error {
	imp.loader.statusMsg("Import dashboard %s", file)

	return imp.loader.ImportDashboard(file)
}

func (imp Importer) ImportFile(fileType string, file string) error {
	imp.loader.statusMsg("Import %s from %s", fileType, file)

	if fileType == "dashboard" {
		return imp.loader.ImportDashboard(file)
	} else if fileType == "index-pattern" {
		return imp.loader.ImportIndex(file)
	}
	return fmt.Errorf("Unexpected file type %s", fileType)
}

func (imp Importer) ImportDir(dirType string, dir string) error {
	imp.loader.statusMsg("Import directory %s", dir)

	dir = path.Join(dir, dirType)
	var errors []string

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
	imp.loader.statusMsg("Unzip archive %s", target)

	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	// Closure to close the files on each iteration
	unzipFile := func(file *zip.File) error {
		filePath := filepath.Join(target, file.Name)

		if file.FileInfo().IsDir() {
			return os.MkdirAll(filePath, file.Mode())
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

	imp.loader.statusMsg("Created temporary directory %s", target)
	if imp.cfg.File != "" {
		archive = imp.cfg.File
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
		imp.loader.statusMsg("Importing Kibana from %s", dir)
		if imp.cfg.Beat == "" || filepath.Base(dir) == imp.cfg.Beat {
			err = imp.ImportKibanaDir(dir)
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
	imp.loader.statusMsg("Downloading %s", url)

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
	if resp.StatusCode != 200 {
		return targetPath, fmt.Errorf("Server returned: %s", resp.Status)
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
func (imp Importer) ImportKibanaDir(dir string) error {
	var err error

	versionPath := strconv.Itoa(imp.version.Major)

	dir = path.Join(dir, versionPath)

	imp.loader.statusMsg("Importing directory %v", dir)

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

	importDashboards := false
	for _, t := range types {
		err = imp.ImportDir(t, dir)
		if err != nil {
			return fmt.Errorf("Failed to import %s: %v", t, err)
		}

		if t == "dashboard" {
			importDashboards = true
		}
	}

	if !importDashboards {
		return fmt.Errorf("No dashboards to import. Please make sure the %s directory contains a dashboard directory.",
			dir)
	}
	return nil
}

func (imp Importer) subdirExists(parent string, child string) bool {
	if _, err := os.Stat(path.Join(parent, child)); err != nil {
		return false
	}
	return true
}
