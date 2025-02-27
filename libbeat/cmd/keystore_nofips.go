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

//go:build !requirefips

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

// genKeystoreCmd initialize the Keystore command to manage the Keystore
// with the following subcommands:
//   - create
//   - add
//   - remove
//   - list
func genKeystoreCmd(settings instance.Settings) *cobra.Command {
	keystoreCmd := cobra.Command{
		Use:   "keystore",
		Short: "Manage secrets keystore",
	}

	keystoreCmd.AddCommand(genCreateKeystoreCmd(settings))
	keystoreCmd.AddCommand(genAddKeystoreCmd(settings))
	keystoreCmd.AddCommand(genRemoveKeystoreCmd(settings))
	keystoreCmd.AddCommand(genListKeystoreCmd(settings))

	return &keystoreCmd
}
