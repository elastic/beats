package kibana

import (
	"encoding/json"
	"errors"
	"fmt"
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

func (i *Index) Create() error {
	err := i.init()
	if err != nil {
		return err
	}

	commonFields, err := common.LoadFieldsYaml(i.fieldsYaml)
	if err != nil {
		return err
	}
	fields := TransformFields("@timestamp", i.IndexName, commonFields)

	dumpToFile(i.targetDir5x, i.targetFilename, fields)

	out := common.MapStr{
		"version": i.Version,
		"objects": []common.MapStr{
			common.MapStr{
				"type":       "index-pattern",
				"id":         i.IndexName,
				"version":    1,
				"attributes": fields,
			},
		},
	}
	dumpToFile(i.targetDirDefault, i.targetFilename, out)
	return nil
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

func dumpToFile(targetDir string, file string, pattern common.MapStr) error {
	patternIndent, err := json.MarshalIndent(pattern, "", "  ")
	if err != nil {
		return err
	}
	f := filepath.Join(targetDir, file)
	err = ioutil.WriteFile(f, patternIndent, 0644)
	if err != nil {
		return err
	}
	fmt.Println("-- The index pattern was created under ", f)
	return nil
}

func createTargetDir(baseDir string, version string) string {
	targetDir := filepath.Join(baseDir, "_meta", "kibana", version, "index-pattern")
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0777)
	}
	return targetDir
}
