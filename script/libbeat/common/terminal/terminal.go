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

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadInput Capture user input for a question
func ReadInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(input, "\n"), nil
}

// PromptYesNo Returns true if the user has entered Y or YES, capitalization is ignored, we are
// matching elasticsearch behavior
func PromptYesNo(prompt string, defaultAnswer bool) bool {
	var defaultYNprompt string

	if defaultAnswer == true {
		defaultYNprompt = "[Y/n]"
	} else {
		defaultYNprompt = "[y/N]"
	}

	fmt.Printf("%s %s: ", prompt, defaultYNprompt)

	for {
		input, err := ReadInput()
		if err != nil {
			panic("could not read from input")
		}

		response := strings.TrimSpace(input)
		response = strings.ToLower(response)
		if response == "" {
			return defaultAnswer
		} else if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}

		fmt.Printf("Did not understand the answer '%s'\n", input)
	}
}
