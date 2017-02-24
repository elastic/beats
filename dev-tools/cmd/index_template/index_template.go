package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/template"
)

// main generates index templates for the beats
func main() {

	beatName := flag.String("beat.name", "", ": Base index name. Normally {beat_name} (required)")
	output := flag.String("output", "", "Required: Full path to the output file (required)")
	version := flag.String("es.version", beat.GetDefaultVersion(), "Elasticsearch version")

	flag.Parse()

	var existingFiles []string
	files := flag.Args()

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No fields.yml files provided. At least one file must be added.")
		os.Exit(1)
	}

	if *beatName == "" {
		fmt.Fprintf(os.Stderr, "beat.name is empty. It must be set.")
		os.Exit(1)
	}

	// Skip some of the passed files, as not all beats have the same files
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			fmt.Printf("Skipping file because it does not exist: %s", f)
			continue
		}
		existingFiles = append(existingFiles, f)
	}

	// Make it compatible with the sem versioning
	if *version == "2x" {
		*version = "2.0.0"
	}

	templateString, err := template.GetTemplate(*version, *beatName, existingFiles)
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
