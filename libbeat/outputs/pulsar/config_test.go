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

package pulsar

import (
    "testing"
)

func Test_pulsarConfig_Validate(t *testing.T) {
    type fields struct {
        URL                   string
        UseTLS                bool
        TLSTrustCertsFilePath string
        Topic                 string
    }
    tests := []struct {
        name    string
        fields  fields
        wantErr bool
    }{
        // TODO: Add test cases.
        {
            "test url",
            fields{
                "",
                false,
                "",
                "test",
            },
            true,
        },
        {
            "test topic",
            fields{
                "pulsar://localhost:6650",
                false,
                "",
                "",
            },
            true,
        },
        {
            "test use tls",
            fields{
                "pulsar://localhost:6650",
                true,
                "/go/src/github.com/AmateurEvents/filebeat-ouput-pulsar/certs/ca.cert.pem",
                "persistent://public/default/my-topic1",
            },
            false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := &pulsarConfig{
                URL:                   tt.fields.URL,
                UseTLS:                tt.fields.UseTLS,
                TLSTrustCertsFilePath: tt.fields.TLSTrustCertsFilePath,
                Topic:                 tt.fields.Topic,
            }
            if err := c.Validate(); (err != nil) != tt.wantErr {
                t.Errorf("pulsarConfig.Validate() error = %v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
