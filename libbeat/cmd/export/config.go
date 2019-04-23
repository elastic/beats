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

package export

import (
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
)

// GenExportConfigCmd write to stdout the current configuration in the YAML format.
func GenExportConfigCmd(settings instance.Settings) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Export current config to stdout",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			return exportConfig(settings)
		}),
	}
}

func exportConfig(settings instance.Settings) error {
	settings.DisableConfigResolver = true
	b, err := instance.NewInitializedBeat(settings)
	if err != nil {
		fatalfInitCmd(err)
	}

	var config map[string]interface{}
	err = b.RawConfig.Unpack(&config)
	if err != nil {
		fatalf("Error unpacking config: %+v.", err)
	}
	res, err := yaml.Marshal(config)
	if err != nil {
		fatalf("Error converting config to YAML format: %+v.", err)
	}

	os.Stdout.Write(res)
	return nil
}
