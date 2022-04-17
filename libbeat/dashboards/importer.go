// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	"strings"

	errw "github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// ErrNotFound returned when we cannot find any dashboard to import.
type ErrNotFound struct {
	ErrorString string
}

// Error returns the human readable error.
func (e *ErrNotFound) Error() string { return e.ErrorString }

func newErrNotFound(s string, a ...interface{}) *ErrNotFound {
	return &ErrNotFound{fmt.Sprintf(s, a...)}
}

// MessageOutputter is a function type for injecting status logging
// into this module.
type MessageOutputter func(msg string, a ...interface{})

// Importer is a type to import dashboards
type Importer struct {
	cfg     *Config
	version common.Version

	loader KibanaLoader
	fields common.MapStr
}

// NewImporter creates a new dashboard importer
func NewImporter(version common.Version, cfg *Config, loader KibanaLoader, fields common.MapStr) (*Importer, error) {

	// Current max version is 7
	if version.Major > 6 {
		version.Major = 7
	}

	return &Importer{
		cfg:     cfg,
		version: version,
		loader:  loader,
		fields:  fields,
	}, nil
}

// Import imports the Kibana dashboards according to the configuration options.
func (imp Importer) Import() error {
	if imp.cfg.URL != "" || imp.cfg.File != "" {
		err := imp.ImportArchive()
		if err != nil {
			return errw.Wrap(err, "Error importing URL/file")
		}
	} else {
		err := imp.ImportKibanaDir(imp.cfg.Dir)
		if err != nil {
			return errw.Wrapf(err, "Error importing directory %s", imp.cfg.Dir)
		}
	}
	return nil
}

// ImportDashboard imports a dashboard
func (imp Importer) ImportDashboard(file string) error {
	imp.loader.statusMsg("Import dashboard %s", file)

	return imp.loader.ImportDashboard(file)
}

// ImportFile imports a file
func (imp Importer) ImportFile(fileType string, file string) error {
	imp.loader.statusMsg("Import %s from %s", fileType, file)

	if fileType == "dashboard" {
		return imp.loader.ImportDashboard(file)
	} else if fileType == "index-pattern" {
		return imp.loader.ImportIndexFile(file)
	}
	return fmt.Errorf("Unexpected file type %s", fileType)
}

// ImportDir imports a directory
func (imp Importer) ImportDir(dirType string, dir string) error {
	imp.loader.statusMsg("Import directory %s", dir)

	var errors []string

	files, err := filepath.Glob(path.Join(dir, dirType, "*.json"))
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
	defer reader.Close()

	// Closure to close the files on each iteration
	unzipFile := func(file *zip.File) error {
		filePath := filepath.Join(target, file.Name)

		// check that the resulting file path is indeed under target
		// Note that Rel calls Clean.
		relPath, err := filepath.Rel(target, filePath)
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.ToSlash(relPath), "../") {
			return fmt.Errorf("Zip file contains files outside of the target directory: %s", relPath)
		}

		if file.FileInfo().IsDir() {
			return os.MkdirAll(filePath, file.Mode())
		}

		if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed making directory for file %v: %v", filePath, err)
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

// ImportArchive imports a zip archive
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
		} else {
			imp.loader.statusMsg("Skipping import of %s directory. Beat name: %s, base dir name: %s.", dir, imp.cfg.Beat, filepath.Base(dir))
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

// ImportKibanaDir imports dashboards and index-pattern or only one of these
func (imp Importer) ImportKibanaDir(dir string) error {
	var err error

	versionPath := "7"

	// Loads the internal index pattern
	if imp.fields != nil {
		if err = imp.loader.ImportIndex(imp.fields); err != nil {
			return errw.Wrap(err, "failed to import Kibana index pattern")
		}
	}

	dir = path.Join(dir, versionPath)

	imp.loader.statusMsg("Importing directory %v", dir)

	if _, err := os.Stat(dir); err != nil {
		return newErrNotFound("No directory %s", dir)
	}
	check := []string{}
	if !imp.cfg.OnlyDashboards {
		check = append(check, "index-pattern")
	}
	wantDashboards := false
	if !imp.cfg.OnlyIndex {
		check = append(check, "dashboard")
		wantDashboards = true
	}

	types := []string{}
	for _, c := range check {
		if imp.subdirExists(dir, c) {
			types = append(types, c)
		}
	}

	if len(types) == 0 {
		return newErrNotFound("The directory %s does not contain the %s subdirectory."+
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

	if wantDashboards && !importDashboards {
		return newErrNotFound("No dashboards to import. Please make sure the %s directory "+
			"contains a dashboard directory.", dir)
	}
	return nil
}

func (imp Importer) subdirExists(parent string, child string) bool {
	if _, err := os.Stat(path.Join(parent, child)); err != nil {
		return false
	}
	return true
}
