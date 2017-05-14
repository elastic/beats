package main

import (
	"github.com/elastic/beats/libbeat/beat"
	"{{cookiecutter.beat_path}}/beater"
)

func main() {
	beat.Run("{{cookiecutter.beat}}", "", beater.New())
}
