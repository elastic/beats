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

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"errors"
)

// ReadInput shows the text and ask the user to provide input.
func ReadInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	return input(reader, os.Stdout, prompt)
}

func input(r io.Reader, out io.Writer, prompt string) (string, error) {
	reader := bufio.NewScanner(r)
	fmt.Fprintf(out, prompt+" ")

	if !reader.Scan() {
		return "", errors.New("error reading user input")
	}
	return reader.Text(), nil
}
