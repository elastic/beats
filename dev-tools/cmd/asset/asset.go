// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/v7/libbeat/asset"
	"github.com/elastic/beats/v7/licenses"
)

var (
	pkg      string
	input    string
	output   string
	name     string
	priority string
	license  = "ASL2"
)

func init() {
	flag.StringVar(&pkg, "pkg", "", "Package name")
	flag.StringVar(&input, "in", "-", "Source of input. \"-\" means reading from stdin")
	flag.StringVar(&output, "out", "-", "Output path. \"-\" means writing to stdout")
	flag.StringVar(&license, "license", "ASL2", "License header for generated file.")
	flag.StringVar(&name, "name", "", "Asset name")
	flag.StringVar(&priority, "priority", "asset.BeatFieldsPri", "Priority name")
}

func main() {
	flag.Parse()
	args := flag.Args()

	var (
		file, beatName string
		data           []byte
		err            error
	)
	if input == "-" {
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "File path must be set")
			os.Exit(1)
		}
		file = args[0]
		beatName = args[1]

		r := bufio.NewReader(os.Stdin)
		data, err = ioutil.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading from stdin: %v\n", err)
			os.Exit(1)
		}
	} else {
		file = input
		beatName = args[0]
		data, err = ioutil.ReadFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid file path: %s\n", input)
			os.Exit(1)
		}
	}

	licenseHeader, err := licenses.Find(license)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid license: %s\n", err)
		os.Exit(1)
	}
	if name == "" {
		name = file
	}

	bs, err := asset.CreateAsset(licenseHeader, beatName, name, pkg, data, priority, file)
	if err != nil {
		panic(err)
	}

	if output == "-" {
		os.Stdout.Write(bs)
	} else {
		ioutil.WriteFile(output, bs, 0640)
	}
}
