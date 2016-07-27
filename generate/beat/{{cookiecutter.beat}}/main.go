package main

import (
	"os"

	"github.com/elastic/beats/libbeat/beat"

	"{{cookiecutter.beat_path}}/{{cookiecutter.beat}}/beater"
)

func main() {
	err := beat.Run("{{cookiecutter.beat}}", "", beater.New)
	if err != nil {
		os.Exit(1)
	}
}
