package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/libbeat/template"
	"github.com/elastic/beats/libbeat/version"
)

// main generates index templates for the beats
func main() {
	beatVersion := version.GetDefaultVersion()
	index := flag.String("index", "", "Base index name. Normally {beat_name} (required)")
	output := flag.String("output", "", "Required: Full path to the output file (required)")
	version := flag.String("es.version", beatVersion, "Elasticsearch version")
	file := flag.String("file", "", "Path to fields.yml file")

	flag.Parse()

	if len(*file) == 0 {
		fmt.Fprintf(os.Stderr, "File path cannot be empty")
		os.Exit(1)
	}

	if *index == "" {
		fmt.Fprintf(os.Stderr, "index is empty. It must be set.")
		os.Exit(1)
	}

	if _, err := os.Stat(*file); err != nil {
		fmt.Fprintf(os.Stderr, "Error during loading -file %s with error: %s", *file, err)
		os.Exit(1)
	}

	// Make it compatible with the sem versioning
	if *version == "2x" {
		*version = "2.0.0"
	}

	tmpl, err := template.New(beatVersion, *index, *version, template.TemplateConfig{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
		os.Exit(1)
	}

	templateString, err := tmpl.Load(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*output, []byte(templateString.StringToPrint()), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %s", err)
		os.Exit(1)
	}
}
