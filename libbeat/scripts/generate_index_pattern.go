package main

import (
	"os"

	"github.com/elastic/beats/libbeat/kibana"
)

func main() {
	index := kibana.Index{
		IndexName: os.Args[1],
		BeatDir:   os.Args[3],
		BeatName:  os.Args[2],
		Version:   os.Args[4],
	}
	err := index.Create()
	if err != nil {
		panic(err)
	}
}
