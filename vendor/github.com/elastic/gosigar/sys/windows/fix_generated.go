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

//+build ignore

package main

import (
	"flag"
	"io/ioutil"
	"log"
	"regexp"
)

func main() {
	var filename string

	log.SetFlags(0)
	flag.StringVar(&filename, "input", "", "name of generated source file to fix")
	flag.Parse()

	if filename == "" {
		log.Fatal("Name of generated file must be specified with -input flag")
	}

	if err := fixGeneratedCode(filename); err != nil {
		log.Fatal(err)
	}
}

var lazySystemRegex = regexp.MustCompile(`(?m)\sNewLazySystemDLL`)
var unsafeImportRegex = regexp.MustCompile(`(?m)"unsafe"`)

// fixGeneratedCode adds "windows." to locations in the generated source code
// that reference "NewLazySystemDLL" without the package name.
func fixGeneratedCode(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	data = lazySystemRegex.ReplaceAll(data, []byte(" windows.NewLazySystemDLL"))
	data = unsafeImportRegex.ReplaceAll(data, []byte(`"unsafe"`+"\n\n\t"+`"golang.org/x/sys/windows"`))

	return ioutil.WriteFile(filename, data, 0644)
}
