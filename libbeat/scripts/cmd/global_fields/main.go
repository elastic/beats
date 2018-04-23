package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/generator/fields"
)

func main() {
	path := flag.String("beats_path", ".", "Path to elastic/beats")
	name := flag.String("beat_name", ".", "Name of the Beat")
	flag.Parse()

	if len(os.Args) != 6 {
		fmt.Println("Not enough parameters to generate fields")
		os.Exit(1)
	}

	fieldFiles, err := fields.CollectModuleFiles(os.Args[5])
	if err != nil {
		fmt.Printf("Cannot collect fields.yml files: %v\n", err)
		os.Exit(2)

	}

	err = fields.Generate(*path, *name, fieldFiles)
	if err != nil {
		fmt.Printf("Cannot generate global fields.yml file: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("Generated global fields.yml under _meta/fields.generated.yml")
}
