package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type ModulesManager interface {
	ListEnabled() []string
	ListDisabled() []string
	Exists(name string) bool
	Enabled(name string) bool
	Enable(name string) error
	Disable(name string) error
}

// GenModulesCmd initializes a command to manage a modules.d folder, it offers
// list, enable and siable actions
func GenModulesCmd(name, version string, modules ModulesManager) *cobra.Command {
	modulesCmd := cobra.Command{
		Use:   "modules",
		Short: "Manage configured modules",
	}

	modulesCmd.AddCommand(genListModulesCmd(modules))
	modulesCmd.AddCommand(genEnableModulesCmd(modules))
	modulesCmd.AddCommand(genDisableModulesCmd(modules))

	return &modulesCmd
}

func genListModulesCmd(modules ModulesManager) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List modules",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Enabled:")
			for _, module := range modules.ListEnabled() {
				fmt.Println(module)
			}

			fmt.Println("\nDisabled:")
			for _, module := range modules.ListDisabled() {
				fmt.Println(module)
			}
		},
	}
}

func genEnableModulesCmd(modules ModulesManager) *cobra.Command {
	return &cobra.Command{
		Use:   "enable MODULE...",
		Short: "Enable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
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

func genDisableModulesCmd(modules ModulesManager) *cobra.Command {
	return &cobra.Command{
		Use:   "disable MODULE...",
		Short: "Disable one or more given modules",
		Run: func(cmd *cobra.Command, args []string) {
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
