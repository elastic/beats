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

package tlscommon

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// This does not actually test that it ignores the server name because no part of the func even consumes the server name
func Test_verifyCertificateExceptServerName(t *testing.T) {

	tests := []struct {
		name    string
		ca      string
		chain   string
		cert    string
		time    func() time.Time
		wantErr bool
	}{
		{
			name: "happy path",
			// a CA for morello.ovh valid from August 9 2019 to 2029
			ca: "ca.crt",
			// a cert signed by morello.ovh that expired in nov 2019
			cert: "tls.crt",
			time: func() time.Time {
				layout := "2006-01-02"
				t, _ := time.Parse(layout, "2019-10-01")
				return t
			},
			wantErr: false,
		},
		{
			name: "cert not signed by CA",
			ca:   "ca.crt",
			// a self-signed cert for www.example.com valid from July 23 2020 to 2030
			cert: "unsigned_tls.crt",
			time: func() time.Time {
				layout := "2006-01-02"
				t, _ := time.Parse(layout, "2020-07-24")
				return t
			},
			wantErr: true,
		},
		{
			name:    "cert expired",
			ca:      "ca.crt",
			cert:    "tls.crt",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &TLSConfig{time: tc.time}
			// load the CA
			if tc.ca != "" {
				ca := loadFileBytes(tc.ca)
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(ca)
				cfg.RootCAs = caCertPool
			}

			// load the cert
			rawCerts := [][]byte{}
			if tc.cert != "" {
				pemCert := loadFileBytes(tc.cert)
				block, _ := pem.Decode(pemCert)
				rawCerts = append(rawCerts, block.Bytes)
			}

			_, _, got := verifyCertificateExceptServerName(rawCerts, cfg)
			if tc.wantErr {
				assert.Error(t, got)
			} else {
				assert.NoError(t, got)
			}
		})
	}
}

func loadFileBytes(fileName string) []byte {
	contents, err := ioutil.ReadFile(filepath.Join("testdata", fileName))
	if err != nil {
		panic(err)
	}
	return contents
}
