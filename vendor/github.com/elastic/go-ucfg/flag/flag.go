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

package flag

import (
	goflag "flag"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/json"
	"github.com/elastic/go-ucfg/yaml"
)

func ConfigVar(
	set *goflag.FlagSet,
	def *ucfg.Config,
	name string,
	usage string,
	opts ...ucfg.Option,
) *ucfg.Config {
	v := NewFlagKeyValue(def, true, opts...)
	registerFlag(set, v, name, usage)
	return v.Config()
}

func Config(
	set *goflag.FlagSet,
	name string,
	usage string,
	opts ...ucfg.Option,
) *ucfg.Config {
	return ConfigVar(set, nil, name, usage, opts...)
}

func ConfigFilesVar(
	set *goflag.FlagSet,
	def *ucfg.Config,
	name string,
	usage string,
	extensions map[string]FileLoader,
	opts ...ucfg.Option,
) *FlagValue {
	v := NewFlagFiles(def, extensions, opts...)
	registerFlag(set, v, name, usage)
	return v
}

func ConfigFiles(
	set *goflag.FlagSet,
	name string,
	usage string,
	extensions map[string]FileLoader,
	opts ...ucfg.Option,
) *FlagValue {
	return ConfigFilesVar(set, nil, name, usage, extensions, opts...)
}

func ConfigYAMLFilesVar(
	set *goflag.FlagSet,
	def *ucfg.Config,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	exts := map[string]FileLoader{"": yaml.NewConfigWithFile}
	return ConfigFilesVar(set, def, name, usage, exts, opts...)
}

func ConfigYAMLFiles(
	set *goflag.FlagSet,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	return ConfigYAMLFilesVar(set, nil, name, usage, opts...)
}

func ConfigJSONFilesVar(
	set *goflag.FlagSet,
	def *ucfg.Config,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	exts := map[string]FileLoader{"": json.NewConfigWithFile}
	return ConfigFilesVar(set, def, name, usage, exts, opts...)
}

func ConfigJSONFiles(
	set *goflag.FlagSet,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	return ConfigJSONFilesVar(set, nil, name, usage, opts...)
}

func ConfigFilesExtsVar(
	set *goflag.FlagSet,
	def *ucfg.Config,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	exts := map[string]FileLoader{
		".yaml": yaml.NewConfigWithFile,
		".yml":  yaml.NewConfigWithFile,
		".json": json.NewConfigWithFile,
	}
	return ConfigFilesVar(set, def, name, usage, exts, opts...)
}

func ConfigFilesExts(
	set *goflag.FlagSet,
	name string,
	usage string,
	opts ...ucfg.Option,
) *FlagValue {
	return ConfigFilesExtsVar(set, nil, name, usage, opts...)
}

func registerFlag(set *goflag.FlagSet, v goflag.Value, name, usage string) {
	if set != nil {
		set.Var(v, name, usage)
	} else {
		goflag.Var(v, name, usage)
	}
}
