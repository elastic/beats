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

package beatgen

import (
	"bufio"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/generator/common/beatgen/setup"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

// ConfigItem represents a value that must be configured for the custom beat
type ConfigItem struct {
	Key     string
	Default func(map[string]string) string
	Help    string
}

// required user config for a custom beat
// These are specified in env variables with newbeat_*
var configList = []ConfigItem{
	{
		Key:  "project_name",
		Help: "Enter the beat name",
		Default: func(cfg map[string]string) string {
			return "examplebeat"
		},
	},
	{
		Key:  "github_name",
		Help: "Enter your github name",
		Default: func(cfg map[string]string) string {
			return "your-github-name"
		},
	},
	{
		Key:  "beat_path",
		Help: "Enter the beat path",
		Default: func(cfg map[string]string) string {
			ghName, _ := cfg["github_name"]
			beatName, _ := cfg["project_name"]
			return "github.com/" + ghName + "/" + strings.ToLower(beatName)
		},
	},
	{
		Key:  "full_name",
		Help: "Enter your full name",
		Default: func(cfg map[string]string) string {
			return "Firstname Lastname"
		},
	},
	{
		Key:  "type",
		Help: "Enter the beat type",
		Default: func(cfg map[string]string) string {
			return "beat"
		},
	},
}

// Generate generates a new custom beat
func Generate() error {
	cfg, err := getConfig()
	if err != nil {
		return errors.Wrap(err, "Error getting config")
	}
	err = setup.GenNewBeat(cfg)
	if err != nil {
		return errors.Wrap(err, "Error generating new beat")
	}

	absBeatPath := filepath.Join(build.Default.GOPATH, "src", cfg["beat_path"])

	err = os.Chdir(absBeatPath)
	if err != nil {
		return errors.Wrap(err, "error changing directory")
	}

	mg.Deps(setup.CopyVendor)
	mg.Deps(setup.RunSetup)
	mg.Deps(setup.GitInit)

	if cfg["type"] == "metricbeat" {
		//This is runV because it'll ask for user input, so we need stdout.
		err = sh.RunV("make", "create-metricset")
		if err != nil {
			return errors.Wrap(err, "error running create-metricset")
		}
	}

	mg.Deps(setup.Update)
	mg.Deps(setup.GitAdd)

	fmt.Printf("=======================\n")
	fmt.Printf("Your custom beat is now available as %s\n", absBeatPath)
	fmt.Printf("=======================\n")

	return nil
}

// VendorUpdate updates the beat vendor directory
func VendorUpdate() error {
	err := sh.Rm("./vendor/github.com/elastic/beats")
	if err != nil {
		return errors.Wrap(err, "error removing vendor dir")
	}

	devtools.SetElasticBeatsDir(getAbsoluteBeatsPath())
	return setup.CopyVendor()
}

// returns a "compleated" config object with everything we need
func getConfig() (map[string]string, error) {
	userCfg := make(map[string]string)
	for _, cfgVal := range configList {
		var cfgKey string
		var err error
		cfgKey, isSet := getEnvConfig(cfgVal.Key)
		if !isSet {
			cfgKey, err = getValFromUser(cfgVal.Help, cfgVal.Default(userCfg))
			if err != nil {
				return userCfg, err
			}
		}
		userCfg[cfgVal.Key] = cfgKey
	}

	return userCfg, nil

}

func getEnvConfig(cfgKey string) (string, bool) {
	EnvKey := fmt.Sprintf("%s_%s", setup.CfgPrefix, strings.ToUpper(cfgKey))

	envKey := os.Getenv(EnvKey)

	if envKey == "" {
		return envKey, false
	}
	return envKey, true
}

// getValFromUser returns a config object from the user. If they don't enter one, fallback to the default
func getValFromUser(help, def string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Human-readable prompt
	fmt.Printf("%s [%s]: ", help, def)
	str, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if str == "\n" {
		return def, nil
	}
	return strings.TrimSpace(str), nil

}

// getAbsoluteBeatsPath tries to infer the "real" non-vendor beats path
func getAbsoluteBeatsPath() string {
	beatsImportPath := "github.com/elastic/beats"
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		return filepath.Join(gopath, "src", beatsImportPath)
	}
	return filepath.Join(build.Default.GOPATH, "src", beatsImportPath)

}
