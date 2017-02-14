package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/elastic/beats/libbeat/template"
)

// Generates index templates for the beats
//
// The command which can be used in the Makefile looks as following
//
//   go run ${ES_BEATS}/dev-tools/cmd/index_template/index_template.go -files ${PWD}/${ES_BEATS}/libbeat/_meta/fields.generated.yml,${BEAT_GOPATH}/src/${BEAT_PATH}/_meta/fields.yml,${BEAT_GOPATH}/src/${BEAT_PATH}/_meta/fields.generated.yml -es-version 5.0.0 -beatname ${BEAT_NAME} -output ${BEAT_GOPATH}/src/${BEAT_PATH}/${BEAT_NAME}.template.json

func main() {

	version := flag.String("es-version", "", "Elasticsearch version")
	inputFiles := flag.String("files", "", "List of files, comma seperated. This files must be passed with the full path.")
	beatName := flag.String("beatname", "", "Base index name. Normally {beatname}")
	output := flag.String("output", "", "Full path to the output file.")

	flag.Parse()

	var existingFiles []string
	files := strings.Split(*inputFiles, ",")

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
		fmt.Printf("Error generating template: %+v", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile(*output, []byte(templateString.StringToPrint()), 0644)
	if err != nil {
		fmt.Printf("Error writing output: %s", err)
		os.Exit(1)
	}
}
