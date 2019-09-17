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

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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
	err = genNewBeat(cfg)
	if err != nil {
		return err
	}

	err = os.Chdir(filepath.Join(build.Default.GOPATH, "src", cfg["beat_path"]))
	if err != nil {
		return err
	}

	mg.Deps(CopyVendor)
	mg.Deps(RunSetup)
	mg.Deps(GitInit)

	if cfg["type"] == "metricbeat" {
		err = sh.RunV("make", "create-metricset")
		if err != nil {
			return errors.Wrap(err, "error running create-metricset")
		}
	}

	mg.Deps(Update)
	mg.Deps(GitAdd)

	return nil
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
