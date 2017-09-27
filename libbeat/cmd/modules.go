package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/cmd/instance"
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

	modulesCmd.AddCommand(genListModulesCmd(name, version, modulesFactory))
	modulesCmd.AddCommand(genEnableModulesCmd(name, version, modulesFactory))
	modulesCmd.AddCommand(genDisableModulesCmd(name, version, modulesFactory))

	return &modulesCmd
}

// Instantiate a modules manager or die trying
func getModules(name, version string, modulesFactory modulesManagerFactory) ModulesManager {
	b, err := instance.NewBeat(name, "", version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
		os.Exit(1)
	}

	if err = b.Init(); err != nil {
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

func genListModulesCmd(name, version string, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(name, version, modulesFactory)

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

func genEnableModulesCmd(name, version string, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "enable MODULE...",
		Short: "Enable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(name, version, modulesFactory)

			for _, module := range args {
				if !modules.Exists(module) {
					fmt.Printf("Module %s doesn't exists!\n", module)
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

func genDisableModulesCmd(name, version string, modulesFactory modulesManagerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "disable MODULE...",
		Short: "Disable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
			modules := getModules(name, version, modulesFactory)

			for _, module := range args {
				if !modules.Exists(module) {
					fmt.Fprintf(os.Stderr, "Module %s doesn't exists!\n", module)
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
