// This generates the rollup job config for a metricSet.
// It requires that for this metricSet the rollup configs are set for each field.
// For the system/network metricset this looks as following:
//
//  go run ../dev-tools/cmd/rollup/rollup.go -metricbeatPath=$GOPATH/src/github.com/elastic/beats/metricbeat -module=system -metricet=network > job.json
//
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/fields/rollup"
)

func main() {
	metricbeatPath := flag.String("metricbeatPath", "", "The path to the Metricbeat directory. (required)")
	module := flag.String("module", "", "The name of the Metricbeat module. (required)")
	metricSet := flag.String("metricset", "", "The name of the Metricbeat metricset. (required)")

	flag.Parse()

	if *metricbeatPath == "" {
		fmt.Fprint(os.Stderr, "The metricbeatPath must be set.")
		os.Exit(1)
	}

	if *module == "" {
		fmt.Fprint(os.Stderr, "The module directory must be set.")
		os.Exit(1)
	}

	if *metricSet == "" {
		fmt.Fprint(os.Stderr, "The metricset directory must be set.")
		os.Exit(1)
	}

	fieldsPath := path.Join(*metricbeatPath, "module", *module, *metricSet, "_meta", "fields.yml")

	fields, err := common.LoadFieldsYamlNoKeys(fieldsPath)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error loading fields.yml: %s, %s", fields, err)
		os.Exit(1)
	}

	processor := rollup.Processor{}
	err = processor.Process(fields, *module)
	if err != nil {
		fmt.Fprint(os.Stderr, "Error processing fields: %s", err)
		os.Exit(1)
	}

	fmt.Printf("%s", processor.Generate().StringToPrint())
}
