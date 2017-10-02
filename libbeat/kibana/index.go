package kibana

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
)

type Index struct {
	Version   string
	IndexName string
	BeatDir   string
	BeatName  string

	fieldsYaml       string
	targetDirDefault string
	targetDir5x      string
	targetFilename   string
}

// Create the Index-Pattern for Kibana for 5.x and default.
func (i *Index) Create() ([]string, error) {
	indices := []string{}

	err := i.init()
	if err != nil {
		return nil, err
	}

	commonFields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return nil, err
	}
	transformed := TransformFields("@timestamp", i.IndexName, commonFields)

	if fieldsBytes, err := json.Marshal(transformed["fields"]); err != nil {
		return nil, err
	} else {
		transformed["fields"] = string(fieldsBytes)
	}
	if fieldFormatBytes, err := json.Marshal(transformed["fieldFormatMap"]); err != nil {
		return nil, err
	} else {
		transformed["fieldFormatMap"] = string(fieldFormatBytes)
	}

	file5x := filepath.Join(i.targetDir5x, i.targetFilename)
	err = dumpToFile(file5x, transformed)
	if err != nil {
		return nil, err
	}
	indices = append(indices, file5x)

	out := common.MapStr{
		"version": i.Version,
		"objects": []common.MapStr{
			common.MapStr{
				"type":       "index-pattern",
				"id":         i.IndexName,
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

func (i *Index) init() error {
	if i.Version == "" || i.IndexName == "" || i.BeatDir == "" || i.BeatName == "" {
		return errors.New("RequiredParams: Version, IndexName, BeatDir and BeatName")
	}
	i.BeatName = clean(i.BeatName)

	i.fieldsYaml = filepath.Join(i.BeatDir, "fields.yml")
	if _, err := os.Stat(i.fieldsYaml); err != nil {
		return err
	}

	i.targetDirDefault = createTargetDir(i.BeatDir, "default")
	i.targetDir5x = createTargetDir(i.BeatDir, "5.x")
	i.targetFilename = i.BeatName + ".json"

	return nil
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
