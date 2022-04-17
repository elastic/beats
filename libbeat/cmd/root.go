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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/cfgfile"
	"github.com/menderesk/beats/v7/libbeat/cmd/instance"
	"github.com/menderesk/beats/v7/libbeat/cmd/platformcheck"
	"github.com/menderesk/beats/v7/libbeat/licenser"
	"github.com/menderesk/beats/v7/libbeat/outputs/elasticsearch"
)

func init() {
	// backwards compatibility workaround, convert -flags to --flags:
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 2 {
			os.Args[1+i] = "-" + arg
		}
	}
}

// BeatsRootCmd handles all application command line interface, parses user
// flags and runs subcommands
type BeatsRootCmd struct {
	cobra.Command
	RunCmd        *cobra.Command
	SetupCmd      *cobra.Command
	VersionCmd    *cobra.Command
	CompletionCmd *cobra.Command
	ExportCmd     *cobra.Command
	TestCmd       *cobra.Command
	KeystoreCmd   *cobra.Command
}

// GenRootCmdWithSettings returns the root command to use for your beat. It take the
// run command, which will be called if no args are given (for backwards compatibility),
// and beat settings
func GenRootCmdWithSettings(beatCreator beat.Creator, settings instance.Settings) *BeatsRootCmd {
	// Add global Elasticsearch license endpoint check.
	// Check we are actually talking with Elasticsearch, to ensure that used features actually exist.
	elasticsearch.RegisterGlobalCallback(licenser.FetchAndVerify)

	if err := platformcheck.CheckNativePlatformCompat(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
		os.Exit(1)
	}

	if settings.IndexPrefix == "" {
		settings.IndexPrefix = settings.Name
	}

	rootCmd := &BeatsRootCmd{}
	rootCmd.Use = settings.Name

	// Due to a dependence upon the beat name, the default config file path
	err := cfgfile.ChangeDefaultCfgfileFlag(settings.Name)
	if err != nil {
		panic(fmt.Errorf("failed to set default config file path: %v", err))
	}

	// must be updated prior to CLI flag handling.

	rootCmd.RunCmd = genRunCmd(settings, beatCreator)
	rootCmd.ExportCmd = genExportCmd(settings)
	rootCmd.TestCmd = genTestCmd(settings, beatCreator)
	rootCmd.SetupCmd = genSetupCmd(settings, beatCreator)
	rootCmd.KeystoreCmd = genKeystoreCmd(settings)
	rootCmd.VersionCmd = GenVersionCmd(settings)
	rootCmd.CompletionCmd = genCompletionCmd(settings, rootCmd)

	// Root command is an alias for run
	rootCmd.Run = rootCmd.RunCmd.Run

	// Persistent flags, common across all subcommands
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("E"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("environment"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.data"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.logs"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("strict.perms"))
	if f := flag.CommandLine.Lookup("plugin"); f != nil {
		rootCmd.PersistentFlags().AddGoFlag(f)
	}

	// Inherit root flags from run command
	// TODO deprecate when root command no longer executes run (7.0)
	rootCmd.Flags().AddFlagSet(rootCmd.RunCmd.Flags())

	// Register subcommands common to all beats
	rootCmd.AddCommand(rootCmd.RunCmd)
	rootCmd.AddCommand(rootCmd.SetupCmd)
	rootCmd.AddCommand(rootCmd.VersionCmd)
	rootCmd.AddCommand(rootCmd.CompletionCmd)
	rootCmd.AddCommand(rootCmd.ExportCmd)
	rootCmd.AddCommand(rootCmd.TestCmd)
	rootCmd.AddCommand(rootCmd.KeystoreCmd)

	return rootCmd
}
