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

package test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/cmd/instance"
)

func GenTestConfigCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	configTestCmd := cobra.Command{
		Use:   "config",
		Short: "Test configuration settings",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewBeat(settings.Name, settings.IndexPrefix, settings.Version, settings.ElasticLicensed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			if err = b.TestConfig(settings, beatCreator); err != nil {
				os.Exit(1)
			}
		},
	}

	return &configTestCmd
}
