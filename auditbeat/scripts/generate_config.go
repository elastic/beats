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

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const defaultGlob = "module/*/_meta/config*.yml.tmpl"

var (
	goos      = flag.String("os", runtime.GOOS, "generate config specific to the specified operating system")
	goarch    = flag.String("arch", runtime.GOARCH, "generate config specific to the specified CPU architecture")
	reference = flag.Bool("ref", false, "generate a reference config")
	concat    = flag.Bool("concat", false, "concatenate all configs instead writing individual files")
)

func findConfigFiles(globs []string) ([]string, error) {
	var configFiles []string
	for _, glob := range globs {
		files, err := filepath.Glob(glob)
		if err != nil {
			return nil, errors.Wrapf(err, "failed on glob %v", glob)
		}
		configFiles = append(configFiles, files...)
	}
	return configFiles, nil
}

// archBits returns the number of bit width of the GOARCH architecture value.
// This function is used by the auditd module configuration templates to
// generate architecture specific audit rules.
func archBits(goarch string) int {
	switch goarch {
	case "386", "arm":
		return 32
	default:
		return 64
	}
}

func getConfig(file string) ([]byte, error) {
	tpl, err := template.ParseFiles(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading %v", file)
	}

	data := map[string]interface{}{
		"GOARCH":    *goarch,
		"GOOS":      *goos,
		"Reference": *reference,
		"ArchBits":  archBits,
	}
	buf := new(bytes.Buffer)
	if err = tpl.Execute(buf, data); err != nil {
		return nil, errors.Wrapf(err, "failed executing template %v", file)
	}

	return buf.Bytes(), nil
}

func output(content []byte, file string) error {
	if file == "-" {
		fmt.Println(string(content))
		return nil
	}

	if err := ioutil.WriteFile(file, content, 0640); err != nil {
		return errors.Wrapf(err, "failed writing output to %v", file)
	}

	return nil
}

func logAndExit(err error) {
	fmt.Fprintf(os.Stderr, "%+v\n", err)
	os.Exit(1)
}

func main() {
	flag.Parse()

	globs := os.Args
	if len(os.Args) > 0 {
		path, err := filepath.Abs(defaultGlob)
		if err != nil {
			logAndExit(err)
		}
		globs = []string{path}
	}

	files, err := findConfigFiles(globs)
	if err != nil {
		logAndExit(err)
	}

	var segments [][]byte
	for _, file := range files {
		segment, err := getConfig(file)
		if err != nil {
			logAndExit(err)
		}

		if *concat {
			segments = append(segments, segment)
		} else {
			output(segment, strings.TrimSuffix(file, ".tmpl"))
		}
	}

	if *concat {
		if err := output(bytes.Join(segments, []byte{'\n'}), "-"); err != nil {
			logAndExit(err)
		}
	}
}
