package cmd

import (
	"flag"
	"fmt"
	"path"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/filebeat/generator"
	"github.com/elastic/beats/filebeat/generator/fields"
	"github.com/elastic/beats/filebeat/generator/fileset"
	"github.com/elastic/beats/filebeat/generator/module"
	libcmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/common/cli"
)

var (
	noDoc = flag.Bool("no-doc", false, "Do not add documentation templates to fields")
)

func genGenerateModule() *cobra.Command {
	return &cobra.Command{
		Use:   "module",
		Short: "Generate a new Filebeat module",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("not enough arguments")
			}
			return generateModule(args[0], args[1], *libcmd.BeatsPath)
		}),
	}
}

func genGenerateFileset() *cobra.Command {
	return &cobra.Command{
		Use:   "fileset",
		Short: "Generate a new Filebeat fileset",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("not enough arguments")
			}
			return generateFileset(args[0], args[1], args[2], *libcmd.BeatsPath)
		}),
	}
}

func genGenerateFieldsYml() *cobra.Command {
	fieldsCmd := &cobra.Command{
		Use:   "fields",
		Short: "Generate fields.yml for a fileset",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return fmt.Errorf("not enough arguments")
			}
			return generateFieldsYml(args[0], args[1], args[2], *libcmd.BeatsPath, *noDoc)
		}),
	}
	fieldsCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("no-doc"))

	return fieldsCmd
}

func generateModule(name, modulesPath, beatsPath string) error {
	if name == "" {
		return fmt.Errorf("missing parameter: module")
	}

	err := module.Generate(name, modulesPath, beatsPath)
	if err != nil {
		return fmt.Errorf("cannot generate module: %v", err)
	}

	fmt.Println("New module was generated, now you can start creating filesets by create-fileset command.")
	return nil
}

func generateFileset(moduleName, filesetName, modulesPath, beatsPath string) error {
	if moduleName == "" {
		return fmt.Errorf("missing parameter: module")
	}

	if filesetName == "" {
		return fmt.Errorf("missing parameter: fileset")
	}

	modulePath := path.Join(modulesPath, "module", moduleName)
	if !generator.DirExists(modulePath) {
		return fmt.Errorf("cannot generate fileset: module not exists, please create module first by create-module command\n")
	}

	err := fileset.Generate(moduleName, filesetName, modulesPath, beatsPath)
	if err != nil {
		return fmt.Errorf("cannot generate fileset: %v", err)
	}

	fmt.Println("New fileset was generated, please check that module.yml file have proper fileset dashboard settings. After setting up Grok pattern in pipeline.json, please generate fields.yml")

	return nil
}

func generateFieldsYml(moduleName, filesetName, modulesPath, beatsPath string, noDoc bool) error {
	if moduleName == "" {
		return fmt.Errorf("missing parameter: module")
	}

	if filesetName == "" {
		return fmt.Errorf("missing parameter: fileset")
	}

	err := fields.Generate(moduleName, filesetName, beatsPath, noDoc)
	if err != nil {
		return fmt.Errorf("cannot generate fields.yml for %s/%s: %v", moduleName, filesetName, err)
	}

	fmt.Printf("Fields.yml generated for %s/%s\n", moduleName, filesetName)

	return nil
}
