package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/kibana"
	"github.com/elastic/beats/libbeat/version"
)

func main() {
	beatVersion := version.GetDefaultVersion()
	index := flag.String("index", "", "The name of the index pattern. (required)")
	beatName := flag.String("beat-name", "", "The name of the beat. (required)")
	beatDir := flag.String("beat-dir", "", "The local beat directory. (required)")
	version := flag.String("version", beatVersion, "The beat version.")
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

	indexPatternGenerator, err := kibana.NewGenerator(*index, *beatName, *beatDir, *version)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	pattern, err := indexPatternGenerator.Generate()
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
	for _, p := range pattern {
		fmt.Fprintf(os.Stdout, "-- The index pattern was created under %v\n", p)
	}
}
