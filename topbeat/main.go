package main

import (
	topbeat "github.com/dr-toboggan/beats/topbeat/beat"

	"github.com/elastic/beats/libbeat/beat"
)

var Name = "topbeat"

func main() {
	beat.Run(Name, "", topbeat.New())
}
