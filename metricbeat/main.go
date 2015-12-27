package main

import (
	"os"

	"github.com/elastic/beats/metricbeat/beater"
	_ "github.com/elastic/beats/metricbeat/include"

	"github.com/elastic/beats/libbeat/beat"
)

var Name = "metricbeat"

func main() {
	if err := beat.Run(Name, "", beater.New()); err != nil {
		os.Exit(1)
	}
}
