package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/elastic/beats/metricbeat/scripts/msetlists/gather"

	_ "github.com/elastic/beats/x-pack/metricbeat/include"
)

func main() {

	msList := gather.DefaultMetricsets()

	raw, err := json.MarshalIndent(msList, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error Marshalling json: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", string(raw))
}
