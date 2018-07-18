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

// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"

	"github.com/elastic/beats/libbeat/asset"
)

var pkg *string

func init() {
	pkg = flag.String("pkg", "", "Package name")
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "File path must be set")
		os.Exit(1)
	}

	file := args[0]
	beatName := args[1]

	data, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid file path: %s", args[0])
		os.Exit(1)
	}

	encData, err := asset.EncodeData(string(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error encoding the data: %s", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	asset.Template.Execute(&buf, asset.Data{
		Beat:    beatName,
		Name:    file,
		Data:    encData,
		Package: *pkg,
	})

	bs, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(bs)
}
