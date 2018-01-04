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
	indexName      string
	beatVersion    string
	fieldsYaml     string
	version        common.Version
	targetDir      string
	targetFilename string
}

// Create an instance of the Kibana Index Pattern Generator
func NewGenerator(indexName, beatName, beatDir, beatVersion string, version common.Version) (*IndexPatternGenerator, error) {
	beatName = clean(beatName)

	fieldsYaml := filepath.Join(beatDir, "fields.yml")
	if _, err := os.Stat(fieldsYaml); err != nil {
		return nil, err
	}

	return &IndexPatternGenerator{
		indexName:      indexName,
		fieldsYaml:     fieldsYaml,
		beatVersion:    beatVersion,
		version:        version,
		targetDir:      createTargetDir(beatDir, version),
		targetFilename: beatName + ".json",
	}, nil
}

// Create the Index-Pattern for Kibana for 5.x and default.
func (i *IndexPatternGenerator) Generate() (string, error) {
	commonFields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return "", err
	}

	transformed, err := generate(i.indexName, &i.version, commonFields)
	if err != nil {
		return "", err
	}

	if i.version.Major >= 6 {
		transformed = i.generateMinVersion6(transformed)
	}

	file := filepath.Join(i.targetDir, i.targetFilename)
	err = dumpToFile(file, transformed)

	return file, err
}

func (i *IndexPatternGenerator) generateMinVersion6(transformed common.MapStr) common.MapStr {
	out := common.MapStr{
		"version": i.beatVersion,
		"objects": []common.MapStr{
			common.MapStr{
				"type":       "index-pattern",
				"id":         i.indexName,
				"version":    1,
				"attributes": transformed,
			},
		},
	}

	return out

}

func generate(indexName string, version *common.Version, f common.Fields) (common.MapStr, error) {
	transformer, err := newTransformer("@timestamp", indexName, version, f)

	if err != nil {
		return nil, err
	}
	transformed, err := transformer.transformFields()
	if err != nil {
		return nil, err
	}

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

func createTargetDir(baseDir string, version common.Version) string {
	targetDir := filepath.Join(baseDir, "_meta", "kibana", getVersionPath(version), "index-pattern")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0777)
	}
	return targetDir
}

func getVersionPath(version common.Version) string {
	versionPath := "6"
	if version.Major == 5 {
		versionPath = "5"
	}
	return versionPath
}
