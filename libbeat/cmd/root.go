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
	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
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

// GenRootCmd returns the root command to use for your beat. It takes the beat name, version,
// and run command, which will be called if no args are given (for backwards compatibility).
//
// Deprecated: Use GenRootCmdWithSettings instead.
func GenRootCmd(name, version string, beatCreator beat.Creator) *BeatsRootCmd {
	return GenRootCmdWithRunFlags(name, version, beatCreator, nil)
}

// GenRootCmdWithRunFlags returns the root command to use for your beat. It takes
// beat name, version, run command, and runFlags. runFlags parameter must the flagset used by
// run command.
//
// Deprecated: Use GenRootCmdWithSettings instead.
func GenRootCmdWithRunFlags(name, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *BeatsRootCmd {
	return GenRootCmdWithIndexPrefixWithRunFlags(name, name, version, beatCreator, runFlags)
}

// GenRootCmdWithIndexPrefixWithRunFlags returns the root command to use for your beat. It takes
// beat name, index prefix, version, run command, and runFlags. runFlags parameter must the flagset used by
// run command.
//
// Deprecated: Use GenRootCmdWithSettings instead.
func GenRootCmdWithIndexPrefixWithRunFlags(name, indexPrefix, version string, beatCreator beat.Creator, runFlags *pflag.FlagSet) *BeatsRootCmd {
	settings := instance.Settings{
		Name:        name,
		IndexPrefix: indexPrefix,
		Version:     version,
		RunFlags:    runFlags,
	}
	return GenRootCmdWithSettings(beatCreator, settings)
}

// GenRootCmdWithSettings returns the root command to use for your beat. It take the
// run command, which will be called if no args are given (for backwards compatibility),
// and beat settings
func GenRootCmdWithSettings(beatCreator beat.Creator, settings instance.Settings) *BeatsRootCmd {
	if settings.IndexPrefix == "" {
		settings.IndexPrefix = settings.Name
	}

	name := settings.Name
	version := settings.Version
	indexPrefix := settings.IndexPrefix
	runFlags := settings.RunFlags

	rootCmd := &BeatsRootCmd{}
	rootCmd.Use = name

	// Due to a dependence upon the beat name, the default config file path
	err := cfgfile.ChangeDefaultCfgfileFlag(name)
	if err != nil {
		panic(fmt.Errorf("failed to set default config file path: %v", err))
	}

	// must be updated prior to CLI flag handling.

	rootCmd.RunCmd = genRunCmd(settings, beatCreator, runFlags)
	rootCmd.SetupCmd = genSetupCmd(name, indexPrefix, version, beatCreator)
	rootCmd.VersionCmd = genVersionCmd(name, version)
	rootCmd.CompletionCmd = genCompletionCmd(name, version, rootCmd)
	rootCmd.ExportCmd = genExportCmd(settings, name, indexPrefix, version)
	rootCmd.TestCmd = genTestCmd(name, version, beatCreator)
	rootCmd.KeystoreCmd = genKeystoreCmd(name, indexPrefix, version, runFlags)

	// Root command is an alias for run
	rootCmd.Run = rootCmd.RunCmd.Run

	// Persistent flags, common across all subcommands
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("E"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("c"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	rootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
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
