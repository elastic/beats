package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/libbeat/generator/fields"
)

func main() {
	beatsPath := flag.String("beats_path", "..", "Path to elastic/beats")
	name := flag.String("beat_name", "", "Name of the Beat")
	flag.Parse()

	beatFieldsPath := flag.Args()
	if len(beatFieldsPath) == 0 {
		fmt.Println("No field files to collect")
		return
	}

	if *name == "" {
		fmt.Println("Beat_name cannot be empty")
		os.Exit(1)
	}

	pathToModules := filepath.Join(*beatsPath, *name, beatFieldsPath[0])
	fieldFiles, err := fields.CollectModuleFiles(pathToModules)
	if err != nil {
		fmt.Printf("Cannot collect fields.yml files: %v\n", err)
		os.Exit(2)

	}

	err = fields.Generate(*beatsPath, *name, fieldFiles)
	if err != nil {
		fmt.Printf("Cannot generate global fields.yml file: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("Generated global fields.yml under _meta/fields.generated.yml")
}
