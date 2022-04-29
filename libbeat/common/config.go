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

package common

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/file"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"
	"github.com/elastic/go-ucfg/yaml"
)

var flagStrictPerms = flag.Bool("strict.perms", true, "Strict permission checking on config files")

// IsStrictPerms returns true if strict permission checking on config files is
// enabled.
func IsStrictPerms() bool {
	if !*flagStrictPerms || os.Getenv("BEAT_STRICT_PERMS") == "false" {
		return false
	}
	return true
}

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

const (
	selectorConfig             = "config"
	selectorConfigWithPassword = "config-with-passwords"
)

// make hasSelector and configDebugf available for unit testing
var hasSelector = logp.HasSelector
var configDebugf = logp.Debug

func PrintConfigDebugf(c *config.C, msg string, params ...interface{}) {
	selector := selectorConfigWithPassword
	filtered := false
	if !hasSelector(selector) {
		selector = selectorConfig
		filtered = true

		if !hasSelector(selector) {
			return
		}
	}

	debugStr := config.DebugString(c, filtered)
	if debugStr != "" {
		configDebugf(selector, "%s\n%s", fmt.Sprintf(msg, params...), debugStr)
	}
}

func LoadFile(path string) (*config.C, error) {
	if IsStrictPerms() {
		if err := OwnerHasExclusiveWritePerms(path); err != nil {
			return nil, err
		}
	}

	c, err := yaml.NewConfigWithFile(path, configOpts...)
	if err != nil {
		return nil, err
	}

	cfg := fromConfig(c)
	PrintConfigDebugf(cfg, "load config file '%v' =>", path)
	return cfg, err
}

func LoadFiles(paths ...string) (*config.C, error) {
	merger := cfgutil.NewCollector(nil, configOpts...)
	for _, path := range paths {
		cfg, err := LoadFile(path)
		if err := merger.Add(access(cfg), err); err != nil {
			return nil, err
		}
	}
	return fromConfig(merger.Config()), nil
}

func fromConfig(in *ucfg.Config) *config.C {
	return (*config.C)(in)
}

func access(c *config.C) *ucfg.Config {
	return (*ucfg.Config)(c)
}

// OwnerHasExclusiveWritePerms asserts that the current user or root is the
// owner of the config file and that the config file is (at most) writable by
// the owner or root (e.g. group and other cannot have write access).
func OwnerHasExclusiveWritePerms(name string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := file.Stat(name)
	if err != nil {
		return err
	}

	euid := os.Geteuid()
	fileUID, _ := info.UID()
	perm := info.Mode().Perm()

	if fileUID != 0 && euid != fileUID {
		return fmt.Errorf(`config file ("%v") must be owned by the user identifier `+
			`(uid=%v) or root`, name, euid)
	}

	// Test if group or other have write permissions.
	if perm&0022 > 0 {
		nameAbs, err := filepath.Abs(name)
		if err != nil {
			nameAbs = name
		}
		return fmt.Errorf(`config file ("%v") can only be writable by the `+
			`owner but the permissions are "%v" (to fix the permissions use: `+
			`'chmod go-w %v')`,
			name, perm, nameAbs)
	}

	return nil
}
