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

package stdfields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"
)

func TestUnnestStream(t *testing.T) {
	type testCase struct {
		name string
		cfg  mapstr.M
		v    validator.Validator
	}
	tests := []testCase{
		{
			name: "simple",
			cfg: mapstr.M{
				"id": "myuuid",
				"streams": []mapstr.M{
					{
						"type":     "montype",
						"streamid": "mystreamid",
						"data_stream": mapstr.M{
							"namespace": "mynamespace",
							"dataset":   "mydataset",
							"type":      "mytype",
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":   "myuuid",
				"type": "montype",
				"data_stream": mapstr.M{
					"namespace": "mynamespace",
					"dataset":   "mydataset",
					"type":      "mytype",
				},
			}),
		},
		{
			name: "split data stream",
			cfg: mapstr.M{
				"id":   "myuuid",
				"type": "montype",
				"data_stream": mapstr.M{
					"namespace": "mynamespace",
				},
				"streams": []mapstr.M{
					{
						"type": "montype",
						"data_stream": mapstr.M{
							"type":    "mytype",
							"dataset": "mydataset",
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":   "myuuid",
				"type": "montype",
				"data_stream": mapstr.M{
					"namespace": "mynamespace",
					"dataset":   "mydataset",
					"type":      "mytype",
				},
			}),
		},
		{
			name: "base is last, not first stream",
			cfg: mapstr.M{
				"id": "myuuid",
				"data_stream": mapstr.M{
					"namespace": "parentnamespace",
				},
				"streams": []mapstr.M{
					{
						"data_stream": mapstr.M{
							// Intentionally missing `type` since
							// this is not the base dataset.
							// There is only one stream with `type`
							"dataset": "notbasedataset",
						},
					},
					{
						"type": "montype",
						"data_stream": mapstr.M{
							"type":    "basetype",
							"dataset": "basedataset",
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":   "myuuid",
				"type": "montype",
				"data_stream": mapstr.M{
					"namespace": "parentnamespace",
					"type":      "basetype",
					"dataset":   "basedataset",
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src, err := common.NewConfigFrom(test.cfg)
			require.NoError(t, err)

			unnested, err := UnnestStream(src)
			require.NoError(t, err)

			unpacked := mapstr.M{}
			err = unnested.Unpack(unpacked)
			require.NoError(t, err)
			testslike.Test(t, test.v, unpacked)
		})
	}
}
