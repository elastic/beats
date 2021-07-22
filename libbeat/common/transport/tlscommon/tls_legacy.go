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

// +build !go1.15

package tlscommon

import (
	"io/ioutil"
	"os"
	"strings"
)

// ResolveCipherSuite takes the integer representation and return the cipher name.
func ResolveCipherSuite(cipher uint16) string {
	return tlsCipherSuite(cipher).String()
}

// NewPEMReader returns a new PEMReader.
func NewPEMReader(certificate string) (*PEMReader, error) {
	if IsPEMString(certificate) {
		// Take a substring of the certificate so we do not leak the whole certificate or private key in the log.
		debugStr := certificate[0:256] + "..."
		return &PEMReader{reader: ioutil.NopCloser(strings.NewReader(certificate)), debugStr: debugStr}, nil
	}

	r, err := os.Open(certificate)
	if err != nil {
		return nil, err
	}
	return &PEMReader{reader: r, debugStr: certificate}, nil
}
