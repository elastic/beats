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

func collectCommonFiles(esBeatsPath, beatPath string) ([]*YmlFile, error) {
	var libbeatProcessorFiles []*YmlFile
	var err error
	var ymls []*YmlFile
	var files []*YmlFile

	if ymls, err = makeYml(0, filepath.Join(esBeatsPath, "libbeat/_meta/fields.ecs.yml")); err != nil {
		return nil, err
	}
	files = append(files, ymls...)
	commonFields := []string{filepath.Join(esBeatsPath, "libbeat/_meta/fields.ecs.yml")}
	if !isLibbeat(beatPath) {
		commonFields = append(commonFields,
			filepath.Join(esBeatsPath, "libbeat/_meta/fields.common.yml"),
		)

		libbeatProcessorsPath := filepath.Join(esBeatsPath, "libbeat/processors")
		libbeatProcessorFiles, err = CollectModuleFiles(libbeatProcessorsPath)
		if err != nil {
			return nil, err
		}
	}

	if ymls, err = makeYml(0, filepath.Join(esBeatsPath, "libbeat/_meta/fields.common.yml")); err != nil {
		return nil, err
	}
	files = append(files, ymls...)
	libbeatModulesPath := filepath.Join(esBeatsPath, "libbeat/processors")
	libbeatFieldFiles, err := CollectModuleFiles(libbeatModulesPath)
	if err != nil {
		return nil, err
	}
	files = append(files, libbeatFieldFiles...)

	// Fields for custom beats last, to enable overriding more generically defined fields
	if ymls, err = makeYml(0, filepath.Join(beatPath, "_meta/fields.common.yml"), filepath.Join(beatPath, "_meta/fields.yml")); err != nil {
		return nil, err
	}

	files, err = checkExists(commonFields)
	if err != nil {
		return nil, err
	}

	files = append(files, libbeatProcessorFiles...)

	return files, nil
}

func collectBeatFields(beatPath string) ([]*YmlFile, error) {
	commonFields := []string{}
	// Fields for custom beats last, to enable overriding more generically defined fields
	commonFields = append(commonFields,
		filepath.Join(beatPath, "_meta/fields.common.yml"),
		filepath.Join(beatPath, "_meta/fields.yml"),
	)

	return checkExists(commonFields)
}

func checkExists(fields []string) ([]*YmlFile, error) {
	var files []*YmlFile
	for _, cf := range fields {
		_, err := os.Stat(cf)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		files = append(files, &YmlFile{
			Path:   cf,
			Indent: 0,
		})
	}

	return files, nil
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
func Generate(esBeatsPath, beatPath string, moduleFiles []*YmlFile, output string) error {
	libbeatFiles, err := collectCommonFiles(esBeatsPath, beatPath)
	if err != nil {
		return err
	}

	beatFiles, err := collectBeatFields(beatPath)
	if err != nil {
		return err
	}

	if err = os.MkdirAll("build/fields", 0755); err != nil {
		return err
	}

	// Write separate files into build directory for use by fields.go generation
	err = writeGeneratedFieldsYml(libbeatFiles, filepath.Join(beatPath, "build/fields/libbeat.yml"))
	if err != nil {
		return err
	}
	err = writeGeneratedFieldsYml(beatFiles, filepath.Join(beatPath, "build/fields/beat.yml"))
	if err != nil {
		return err
	}
	if len(moduleFiles) != 0 {
		err = writeGeneratedFieldsYml(moduleFiles, filepath.Join(beatPath, "build/fields/module.yml"))
		if err != nil {
			return err
		}
	}

	// Order matters here: ecs, libbeat, beat, module
	files := append(libbeatFiles, beatFiles...)
	files = append(files, moduleFiles...)

	return writeGeneratedFieldsYml(files, output)
}
