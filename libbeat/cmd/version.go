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

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v8/libbeat/cmd/instance"
	"github.com/elastic/beats/v8/libbeat/common/cli"
	"github.com/elastic/beats/v8/libbeat/version"
)

// GenVersionCmd generates the command version for a Beat.
func GenVersionCmd(settings instance.Settings) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show current version info",
		Run: cli.RunWith(
			func(_ *cobra.Command, args []string) error {
				beat, err := instance.NewBeat(settings.Name, settings.IndexPrefix, settings.Version, settings.ElasticLicensed)
				if err != nil {
					return fmt.Errorf("error initializing beat: %s", err)
				}

				buildTime := "unknown"
				if bt := version.BuildTime(); !bt.IsZero() {
					buildTime = bt.String()
				}
				fmt.Printf("%s version %s (%s), libbeat %s [%s built %s]\n",
					beat.Info.Beat, beat.Info.Version, runtime.GOARCH, version.GetDefaultVersion(),
					version.Commit(), buildTime)
				return nil
			}),
	}
}
