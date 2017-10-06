package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/libbeat/version"
)

func main() {
	index := flag.String("index", "", "The name of the index pattern. (required)")
	beatName := flag.String("beat-name", "", "The name of the beat. (required)")
	beatDir := flag.String("beat-dir", "", "The local beat directory. (required)")
	beatVersion := flag.String("version", version.GetDefaultVersion(), "The beat version.")
	flag.Parse()

	if *index == "" {
		fmt.Fprint(os.Stderr, "The name of the index pattern msut be set.")
		os.Exit(1)
	}

	if *beatName == "" {
		fmt.Fprint(os.Stderr, "The name of the beat must be set.")
		os.Exit(1)
	}

	if *beatDir == "" {
		fmt.Fprint(os.Stderr, "The beat directory must be set.")
		os.Exit(1)
	}

	version5, _ := common.NewVersion("5.0.0")
	version6, _ := common.NewVersion("6.0.0")
	versions := []*common.Version{version5, version6}
	for _, version := range versions {

		indexPatternGenerator, err := kibana.NewGenerator(*index, *beatName, *beatDir, *beatVersion, *version)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		pattern, err := indexPatternGenerator.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "-- The index pattern was created under %v\n", pattern)
	}
}
