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
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

type method func(m string) (string, error)

var methods = map[string]method{
	"stdin": stdin,
	"env":   env,
}

// ReadPassword allows to read a password passed as a command line parameter.
// It offers several ways to read the password so it is not directly passed as a plain text argument:
//   stdin - Will prompt the user to input the password
//   env:VAR_NAME - Will read the password from the given env variable
//
func ReadPassword(def string) (string, error) {
	if len(def) == 0 {
		return "", errors.New("empty password definition")
	}

	var method, params string
	parts := strings.SplitN(def, ":", 2)
	method = strings.ToLower(parts[0])

	if len(parts) == 2 {
		params = parts[1]
	}

	m := methods[method]
	if m == nil {
		return "", errors.New("unknown password source, use stdin or env:VAR_NAME")
	}

	return m(params)
}

func stdin(p string) (string, error) {
	fmt.Print("Enter password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", errors.Wrap(err, "reading password input")
	}
	fmt.Println()
	return string(bytePassword), nil
}

func env(p string) (string, error) {
	if len(p) == 0 {
		return "", errors.New("environment variable name is needed when using env: password method")
	}

	v, ok := os.LookupEnv(p)
	if !ok {
		return "", fmt.Errorf("environment variable %s does not exist", p)
	}

	return v, nil
}
