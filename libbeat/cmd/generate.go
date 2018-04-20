package cmd

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/libbeat/kibana"
)

var (
	BeatsPath = flag.String("beats_path", "..", "")
)

func genGenerateCmd(name, idxPrefix, version string, beatCreator beat.Creator) *cobra.Command {
	b, err := instance.NewBeat(name, idxPrefix, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
		os.Exit(1)
	}

	generateCmd := cobra.Command{
		Use:   "generate",
		Short: fmt.Sprintf("Generate files for %s", strings.Title(name)),
	}

	generateCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("beats_path"))
	generateCmd.AddCommand(genGenerateFieldsCmd(b, beatCreator))
	generateCmd.AddCommand(genGenerateKibanaIndexPattern(name, idxPrefix, version))

	return &generateCmd
}

func genGenerateFieldsCmd(b *instance.Beat, beatCreator beat.Creator) *cobra.Command {
	return &cobra.Command{
		Use:   "global-fields",
		Short: fmt.Sprintf("Generate global fields.yml for %s", strings.Title(b.Info.Beat)),
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			return b.GenerateGlobalFields(beatCreator, *BeatsPath)
		}),
	}
}

func genGenerateKibanaIndexPattern(name, idxPrefix, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "kibana-index-pattern",
		Short: fmt.Sprintf("Generate Kibana index pattern for %s", strings.Title(name)),
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			return generateKibanaIndexPattern(name, idxPrefix, version, *BeatsPath)
		}),
	}
}

func generateKibanaIndexPattern(name, idxPrefix, version, beatsPath string) error {
	folders := []string{"5", "6"}
	for _, f := range folders {
		patternPath := path.Join(beatsPath, name, "_meta", "kibana", f, "index-pattern")
		err := os.MkdirAll(patternPath, 0750)
		if err != nil {
			return err
		}
	}

	version5, _ := common.NewVersion("5.0.0")
	version6, _ := common.NewVersion("6.0.0")
	versions := []*common.Version{version5, version6}

	beatPath := path.Join(beatsPath, name)
	for _, v := range versions {
		indexPatternGenerator, err := kibana.NewGenerator(idxPrefix+"-*", name, beatPath, version, *v)
		if err != nil {
			return err
		}
		pattern, err := indexPatternGenerator.Generate()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "-- The index pattern was created under %v\n", pattern)
	}

	return nil
}
