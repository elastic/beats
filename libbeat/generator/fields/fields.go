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

package fields

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// YmlFile holds the info on files and how to write them into the global fields.yml
type YmlFile struct {
	Path   string
	Indent int
}

// NewYmlFile performs some checks and then creates and returns a YmlFile struct
func NewYmlFile(path string, indent int) (*YmlFile, error) {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		// skip
		return nil, nil
	}

	if err != nil {
		// return error
		return nil, err
	}

	// All good, return file
	return &YmlFile{
		Path:   path,
		Indent: indent,
	}, nil
}

func makeYml(indent int, paths ...string) ([]*YmlFile, error) {
	var files []*YmlFile
	for _, path := range paths {
		if ymlFile, err := NewYmlFile(path, indent); err != nil {
			return nil, err
		} else if ymlFile != nil {
			files = append(files, ymlFile)
		}
	}
	return files, nil
}

func collectCommonFiles(esBeatsPath, beatPath string, fieldFiles []*YmlFile) ([]*YmlFile, error) {
	var files []*YmlFile
	var ymls []*YmlFile
	var err error
	if ymls, err = makeYml(0, filepath.Join(esBeatsPath, "libbeat/_meta/fields.ecs.yml")); err != nil {
		return nil, err
	}
	files = append(files, ymls...)

	if !isLibbeat(beatPath) {
		if ymls, err = makeYml(0, filepath.Join(esBeatsPath, "libbeat/_meta/fields.common.yml")); err != nil {
			return nil, err
		}
		files = append(files, ymls...)
		libbeatModulesPaths := []string{
			filepath.Join(esBeatsPath, "libbeat/processors"),
			filepath.Join(esBeatsPath, "libbeat/autodiscover/providers"),
		}
		for _, libbeatModulesPath := range libbeatModulesPaths {
			libbeatFieldFiles, err := CollectModuleFiles(libbeatModulesPath)
			if err != nil {
				return nil, err
			}
			files = append(files, libbeatFieldFiles...)
		}
	}

	// Fields for custom beats last, to enable overriding more generically defined fields
	if ymls, err = makeYml(0, filepath.Join(beatPath, "_meta/fields.common.yml"), filepath.Join(beatPath, "_meta/fields.yml")); err != nil {
		return nil, err
	}
	files = append(files, ymls...)

	return append(files, fieldFiles...), nil
}

func isLibbeat(beatPath string) bool {
	return filepath.Base(beatPath) == "libbeat"
}

func writeGeneratedFieldsYml(fieldFiles []*YmlFile, output string) error {
	data, err := GenerateFieldsYml(fieldFiles)
	if err != nil {
		return err
	}

	if output == "-" {
		fw := bufio.NewWriter(os.Stdout)
		_, err = fw.Write(data)
		if err != nil {
			return err
		}
		return fw.Flush()
	}

	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	fw := bufio.NewWriter(f)
	_, err = fw.Write(data)
	if err != nil {
		return err
	}
	return fw.Flush()
}

// GenerateFieldsYml generates a fields.yml based on the given files
func GenerateFieldsYml(fieldFiles []*YmlFile) ([]byte, error) {
	buf := bytes.NewBufferString("")
	for _, p := range fieldFiles {
		file, err := os.Open(p.Path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		fs := bufio.NewScanner(file)
		for fs.Scan() {
			err = writeIndentedLine(buf, fs.Text()+"\n", p.Indent)
			if err != nil {
				return nil, err
			}
		}
		if err := fs.Err(); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func writeIndentedLine(buf *bytes.Buffer, line string, indent int) error {
	newLine := strings.Repeat(" ", indent) + line
	_, err := buf.WriteString(newLine)
	return err
}

// Generate collects fields.yml files and concatenates them into one global file.
func Generate(esBeatsPath, beatPath string, files []*YmlFile, output string) error {
	files, err := collectCommonFiles(esBeatsPath, beatPath, files)
	if err != nil {
		return err
	}

	return writeGeneratedFieldsYml(files, output)
}
