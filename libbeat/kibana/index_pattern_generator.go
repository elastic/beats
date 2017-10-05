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
	commonFields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return nil, err
	}

	index5xPath, err := i.generate5x(commonFields)
	if err != nil {
		return nil, err
	}

	index6xPath, err := i.generate6x(commonFields)
	if err != nil {
		return nil, err
	}

	return []string{index5xPath, index6xPath}, nil
}

func (i *IndexPatternGenerator) generate5x(fields common.Fields) (string, error) {
	version, _ := common.NewVersion("5.0.0")
	transformed, err := generate(i.indexName, version, fields)
	if err != nil {
		return "", err
	}

	file5x := filepath.Join(i.targetDir5x, i.targetFilename)
	err = dumpToFile(file5x, transformed)
	return file5x, err
}

func (i *IndexPatternGenerator) generate6x(fields common.Fields) (string, error) {
	version, _ := common.NewVersion("6.0.0")
	transformed, err := generate(i.indexName, version, fields)
	if err != nil {
		return "", err
	}
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
	file6x := filepath.Join(i.targetDirDefault, i.targetFilename)
	err = dumpToFile(file6x, out)
	return file6x, err
}

func generate(indexName string, version *common.Version, f common.Fields) (common.MapStr, error) {
	transformer, err := NewTransformer("@timestamp", indexName, version, f)
	if err != nil {
		return nil, err
	}
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
	return transformed, nil
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
