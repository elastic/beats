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

package fingerprint

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

var errMethodUnknown = errors.New("unknown method")

type Method uint8

const (
	MethodSHA1 Method = iota
	MethodSHA256
)

// Unpack creates the Method enumeration value from the given string
func (m *Method) Unpack(str string) error {
	str = strings.ToLower(str)

	switch str {
	case "sha1":
		*m = MethodSHA1
	case "sha256":
		*m = MethodSHA256
	default:
		return errMethodUnknown
	}

	return nil
}

type fingerprinter func(string) (string, error)

func (m *Method) factory() (fingerprinter, error) {
	var f fingerprinter
	switch *m {
	case MethodSHA1:
		f = sha1Fingerprinter
	case MethodSHA256:
		f = sha256Fingerprinter
	default:
		return nil, errMethodUnknown
	}

	return f, nil
}

func sha1Fingerprinter(in string) (string, error) {
	return in, nil
}

func sha256Fingerprinter(in string) (string, error) {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(in))), nil

}
