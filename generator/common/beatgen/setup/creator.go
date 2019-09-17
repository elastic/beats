package setup

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/pkg/errors"
)

// CfgPrefix specifies the env variable prefix used to configure the beat
var CfgPrefix = "NEWBEAT"

// GenNewBeat generates a new custom beat
// We assume our config object is populated and valid here
func GenNewBeat(config map[string]string) error {
	if config["type"] != "beat" && config["type"] != "metricbeat" {
		return fmt.Errorf("%s is not a valid custom beat type. Valid types are 'beat' and 'metricbeat'", config["type"])
	}

	genPath := devtools.OSSBeatDir("generator", config["type"], "{beat}")
	err := filepath.Walk(genPath, func(path string, info os.FileInfo, err error) error {
		newBase := filepath.Join(build.Default.GOPATH, "src", config["beat_path"])
		replacePath := strings.Replace(path, genPath, newBase, -1)

		writePath := strings.Replace(replacePath, "{beat}", config["project_name"], -1)
		writePath = strings.Replace(writePath, ".go.tmpl", ".go", -1)
		if info.IsDir() {
			err := os.MkdirAll(writePath, 0755)
			if err != nil {
				return errors.Wrap(err, "error creating directory")
			}
		} else {

			//dump original source file
			tmplFile, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			newFile := replaceVars(config, string(tmplFile))

			err = ioutil.WriteFile(writePath, []byte(newFile), 0644)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// replaceVars replaces any template vars in a target file
// We're not using the golang template engine as it seems a tad heavy-handed for this use case
// We have a dozen or so files across various languages (go, make, etc) and most just need one or two vars replaced.
func replaceVars(config map[string]string, fileBody string) string {
	var newBody = fileBody
	config["beat"] = strings.ToLower(config["project_name"])
	for tmplName, tmplValue := range config {
		tmplStr := fmt.Sprintf("{%s}", tmplName)
		newBody = strings.ReplaceAll(newBody, tmplStr, tmplValue)
	}

	return newBody
}
