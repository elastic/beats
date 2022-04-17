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
	"os"

	"github.com/spf13/cobra"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/cfgfile"
	"github.com/menderesk/beats/v7/libbeat/cmd/instance"
)

// ModulesManager interface provides all actions needed to implement modules command
// (to list, enable & disable modules)
type ModulesManager interface {
	ListEnabled() []*cfgfile.CfgFile
	ListDisabled() []*cfgfile.CfgFile
	Exists(name string) bool
	Enabled(name string) bool
	Enable(name string) error
	Disable(name string) error
}

// modulesManagerFactory builds and return a ModulesManager for the given Beat
type modulesManagerFactory func(beat *beat.Beat) (ModulesManager, error)

// GenModulesCmd initializes a command to manage a modules.d folder, it offers
// list, enable and siable actions
func GenModulesCmd(name, version string, modulesFactory modulesManagerFactory) *cobra.Command {
	modulesCmd := cobra.Command{
		Use:   "modules",
		Short: "Manage configured modules",
	}
	settings := instance.Settings{Name: name, Version: version}

	modulesCmd.AddCommand(genListModulesCmd(settings, modulesFactory))
	modulesCmd.AddCommand(genEnableModulesCmd(settings, modulesFactory))
	modulesCmd.AddCommand(genDisableModulesCmd(settings, modulesFactory))

	return &modulesCmd
}

// Instantiate a modules manager or die trying
func getModules(settings instance.Settings, modulesFactory modulesManagerFactory) ModulesManager {
	b, err := instance.NewInitializedBeat(settings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
		os.Exit(1)
	}

	manager, err := modulesFactory(&b.Beat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in modules manager: %s\n", err)
		os.Exit(1)
	}

	return manager
}

func genListModulesCmd(settings instance.Settings, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(settings, modulesFactory)

			fmt.Println("Enabled:")
			for _, module := range modules.ListEnabled() {
				fmt.Println(module.Name)
			}

			fmt.Println("\nDisabled:")
			for _, module := range modules.ListDisabled() {
				fmt.Println(module.Name)
			}
		},
	}
}

func genEnableModulesCmd(settings instance.Settings, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "enable MODULE...",
		Short: "Enable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(settings, modulesFactory)

			for _, module := range args {
				if !modules.Exists(module) {
					fmt.Printf("Module %s doesn't exist!\n", module)
					os.Exit(1)
				}

				if modules.Enabled(module) {
					fmt.Printf("Module %s is already enabled\n", module)
					continue
				}

				if err := modules.Enable(module); err != nil {
					fmt.Fprintf(os.Stderr, "There was an error enabling module %s: %s\n", module, err)
					os.Exit(1)
				}

				fmt.Printf("Enabled %s\n", module)
			}
		},
	}
}

func genDisableModulesCmd(settings instance.Settings, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "disable MODULE...",
		Short: "Disable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(settings, modulesFactory)

			for _, module := range args {
				if !modules.Exists(module) {
					fmt.Fprintf(os.Stderr, "Module %s doesn't exist!\n", module)
					os.Exit(1)
				}

				if !modules.Enabled(module) {
					fmt.Fprintf(os.Stderr, "Module %s is already disabled\n", module)
					continue
				}

				if err := modules.Disable(module); err != nil {
					fmt.Fprintf(os.Stderr, "There was an error disabling module %s: %s\n", module, err)
					os.Exit(1)
				}

				fmt.Printf("Disabled %s\n", module)
			}
		},
	}
}
