package kibana

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
)

type IndexPatternGenerator struct {
	indexName        string
	version          string
	fieldsYaml       string
	targetDirDefault string
	targetDir5x      string
	targetFilename   string
}

// Create an instance of the Kibana Index Pattern Generator
func NewGenerator(indexName, beatName, beatDir, version string) (*IndexPatternGenerator, error) {
	beatName = clean(beatName)

	fieldsYaml := filepath.Join(beatDir, "fields.yml")
	if _, err := os.Stat(fieldsYaml); err != nil {
		return nil, err
	}

	return &IndexPatternGenerator{
		indexName:        indexName,
		version:          version,
		fieldsYaml:       fieldsYaml,
		targetDirDefault: createTargetDir(beatDir, "default"),
		targetDir5x:      createTargetDir(beatDir, "5.x"),
		targetFilename:   beatName + ".json",
	}, nil
}

// Create the Index-Pattern for Kibana for 5.x and default.
func (i *IndexPatternGenerator) Generate() ([]string, error) {
	var indices []string

	commonFields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return nil, err
	}
	transformer := NewTransformer("@timestamp", i.indexName, commonFields)
	transformed := transformer.TransformFields()

	fieldsBytes, err := json.Marshal(transformed["fields"])
	if err != nil {
		return nil, err
	}
	transformed["fields"] = string(fieldsBytes)

	fieldFormatBytes, err := json.Marshal(transformed["fieldFormatMap"])
	if err != nil {
		return nil, err
	}
	transformed["fieldFormatMap"] = string(fieldFormatBytes)

	file5x := filepath.Join(i.targetDir5x, i.targetFilename)
	err = dumpToFile(file5x, transformed)
	if err != nil {
		return nil, err
	}
	indices = append(indices, file5x)

	out := common.MapStr{
		"version": i.version,
		"objects": []common.MapStr{
			common.MapStr{
				"type":       "index-pattern",
				"id":         i.indexName,
				"version":    1,
				"attributes": transformed,
			},
		},
	}

	fileDefault := filepath.Join(i.targetDirDefault, i.targetFilename)
	err = dumpToFile(fileDefault, out)
	if err != nil {
		return indices, err
	}
	indices = append(indices, fileDefault)
	return indices, nil
}

func clean(name string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	return reg.ReplaceAllString(name, "")
}

func dumpToFile(f string, pattern common.MapStr) error {
	patternIndent, err := json.MarshalIndent(pattern, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(f, patternIndent, 0644)
	if err != nil {
		return err
	}
	return nil
}

func createTargetDir(baseDir string, version string) string {
	targetDir := filepath.Join(baseDir, "_meta", "kibana", version, "index-pattern")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0777)
	}
	return targetDir
}
