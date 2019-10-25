// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

func main() {
	f, _ := os.OpenFile(filepath.Join(os.TempDir(), "testing.out"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	f.WriteString("starting \n")
	if os.Args[1] != "-c" {
		panic(fmt.Errorf("configuration not provided %#v", os.Args))
	}

	if len(os.Args) == 2 {
		panic(errors.New("configuration path not provided"))
	}

	filepath := os.Args[2]
	contentBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	testCfg := &TestConfig{}
	if err := yaml.Unmarshal(contentBytes, &testCfg); err != nil {
		panic(err)
	}

	if testCfg.TestFile != "" {
		panic(errors.New("'TestFile' key not found in config"))
	}

	<-time.After(90 * time.Second)

	f.WriteString("finished \n")
}

// TestConfig is a configuration for testing Config calls
type TestConfig struct {
	TestFile string
}
