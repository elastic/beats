package main

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/go-yaml/yaml"
	"os"
)


func main() {
	f, err := os.Open("fields.yml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fieldValues := make([]common.Field, 0)
	err = yaml.NewDecoder(f).Decode(&fieldValues)
	if err != nil {
		panic(err)
	}

	for _, fieldV := range fieldValues {
		printField("",fieldV)
	}
}

func printField(prefix string, v common.Field) {
	if v.Type == "group" {
		for _, fields := range v.Fields {
			printField(v.Name, fields)
		}
	} else {
		fmt.Printf("* *%s.%v*: %v\n", prefix, v.Name, v.Description)
	}
}
