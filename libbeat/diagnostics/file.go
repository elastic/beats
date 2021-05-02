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

package diagnostics

import (
	"fmt"
	"io"
	"os"
	"time"
)

var files = [4]string{"metrics.json", "host.json", "manifest.json", "beat.json"}

func (d *Diagnostics) createFolderAndFiles() (foldername string) {
	foldername = fmt.Sprintf("/tmp/beat-diagnostics-%s", time.Now().Format("20060102150405"))
	fmt.Fprintf(os.Stdout, "Creating diagnostic files at: %s\n", foldername)
	os.Mkdir(foldername, 0755)
	for _, filename := range files {
		f, err := os.Create(fmt.Sprintf("%s/%s", foldername, filename))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create diagnostic file\n")
		}
		defer f.Close()

	}
	return foldername
}

// TODO better error
func (d *Diagnostics) writeToFile(folder string, filename string, data []byte) {
	path := fmt.Sprintf("%s/%s", folder, filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write to file\n")
	}
	f.Write(data)
	f.WriteString("\n")
	defer f.Close()
}

func (d *Diagnostics) copyFiles(src string, dst string) {
	srcf, err := os.OpenFile(src, os.O_RDONLY, os.ModeAppend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %s\n", err)
	}
	defer srcf.Close()

	dstf, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file: %s\n", err)
	}
	defer dstf.Close()

	_, err = io.Copy(dstf, srcf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy files: %s\n", err)
	}
}
