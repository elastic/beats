package main

import (
	"fmt"
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
	indices, err := index.Create()
	if err != nil {
		panic(err)
	}
	for _, i := range indices {
		fmt.Println("-- The index pattern was created under ", i)
	}
}
