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
	"strings"

	"github.com/pkg/errors"
)

// Confirm shows the confirmation text and ask the user to answer (y/n)
// default will be shown in uppercase and be selected if the user hits enter
// returns true for yes, false for no
func Confirm(prompt string, def bool) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	return confirm(reader, prompt, def)
}

func confirm(r io.Reader, prompt string, def bool) (bool, error) {
	options := " [Y/n]"
	if !def {
		options = " [y/N]"
	}

	reader := bufio.NewScanner(r)
	for {
		fmt.Print(prompt + options + ":")

		if !reader.Scan() {
			break
		}
		switch strings.ToLower(reader.Text()) {
		case "":
			return def, nil
		case "y", "yes", "yeah":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please write 'y' or 'n'")
		}
	}

	return false, errors.New("error reading user input")
}
