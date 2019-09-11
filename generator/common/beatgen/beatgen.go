package beatgen

import (
	"bufio"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/pkg/errors"
)

// ConfigItem represents a value that must be configured for the custom beat
type ConfigItem struct {
	Key     string
	Default func(map[string]string) string
}

// required user config for a custom beat
// These are specified in env variables with newbeat_*
var configList = []ConfigItem{
	{
		Key: "project_name",
		Default: func(cfg map[string]string) string {
			return "examplebeat"
		},
	},
	{
		Key: "github_name",
		Default: func(cfg map[string]string) string {
			return "your-github-name"
		},
	},
	{
		Key: "beat_path",
		Default: func(cfg map[string]string) string {
			ghName, _ := cfg["github_name"]
			beatName, _ := cfg["project_name"]
			return "github.com/" + ghName + "/" + strings.ToLower(beatName)
		},
	},
	{
		Key: "full_name",
		Default: func(cfg map[string]string) string {
			return "Firstname Lastname"
		},
	},
	{
		Key: "type",
		Default: func(cfg map[string]string) string {
			return "beat"
		},
	},
}

var cfgPrefix = "NEWBEAT"

// Generate generates a new custom beat
func Generate() error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}
	fmt.Printf("%#v\n", cfg)
	return genNewBeat(cfg)
}

// genNewBeat generates a new custom beat
func genNewBeat(config map[string]string) error {

	genPath := devtools.OSSBeatDir("generator", config["type"], "{beat}")
	fmt.Printf("Attempting to read from gen directory %s\n", genPath)
	err := filepath.Walk(genPath, func(path string, info os.FileInfo, err error) error {
		//fmt.Printf("Got path %s, Fileinfo: %v Name: %s\n", path, info.IsDir(), info.Name())
		newBase := filepath.Join(build.Default.GOPATH, "src", config["beat_path"])
		replacePath := strings.Replace(path, genPath, newBase, -1)
		//fmt.Printf("replacing with path %s\n", replacePath)

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

			fmt.Printf("Attempting to write to %s\n", writePath)
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
	for tmplName, tmplValue := range config {
		tmplStr := fmt.Sprintf("{%s}", tmplName)
		newBody = strings.ReplaceAll(newBody, tmplStr, tmplValue)
	}

	return newBody
}

// returns a "compleated" config object with everything we need
func getConfig() (map[string]string, error) {

	userCfg := make(map[string]string)
	for _, cfgVal := range configList {
		var cfgKey string
		var err error
		cfgKey, isSet := getEnvConfig(cfgVal.Key)
		if !isSet {
			cfgKey, err = getValFromUser(cfgVal.Key, cfgVal.Default(userCfg))
			if err != nil {
				return userCfg, err
			}
		}
		userCfg[cfgVal.Key] = cfgKey
	}

	return userCfg, nil

}

func getEnvConfig(cfgKey string) (string, bool) {
	EnvKey := fmt.Sprintf("%s_%s", cfgPrefix, strings.ToUpper(cfgKey))

	envKey := os.Getenv(EnvKey)

	if envKey == "" {
		return envKey, false
	}
	return envKey, true
}

// getValFromUser returns a config object from the user. If they don't enter one, fallback to the default
func getValFromUser(cfgKey, def string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Human-readable prompt
	fmt.Printf("Enter a %s [%s]: ", strings.Replace(cfgKey, "_", " ", -1), def)
	str, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if str == "\n" {
		return def, nil
	}
	return strings.TrimSpace(str), nil

}
