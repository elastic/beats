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

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"
)

func TestUnnestStream(t *testing.T) {
	testRootId := "rootId"
	testStreamId := "streamId"
	testType := "montype"
	testOrigin := "testOrigin"
	testSched := "@every 10s"
	testNs := "mynamespace"
	testDs := "mydataset"

	type testCase struct {
		name string
		cfg  mapstr.M
		v    validator.Validator
	}
	tests := []testCase{
		{
			name: "no datastream",
			cfg: mapstr.M{
				"id":       testRootId,
				"type":     testType,
				"schedule": testSched,
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":       testRootId,
				"type":     testType,
				"schedule": testSched,
			}),
		},
		{
			name: "simple datastream, with origin uses streamId",
			cfg: mapstr.M{
				"id": testRootId,
				"streams": []mapstr.M{
					{
						"id":       testStreamId,
						"type":     testType,
						"schedule": testSched,
						"origin":   testOrigin,
						"data_stream": mapstr.M{
							"namespace": testNs,
							"dataset":   testDs,
							"type":      testType,
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":       testStreamId,
				"type":     testType,
				"schedule": testSched,
				"origin":   testOrigin,
				"data_stream": mapstr.M{
					"namespace": testNs,
					"dataset":   testDs,
					"type":      testType,
				},
			}),
		},
		{
			name: "simple datastream, no origin, uses rootId",
			cfg: mapstr.M{
				"id": testRootId,
				"streams": []mapstr.M{
					{
						"id":       testStreamId,
						"type":     testType,
						"schedule": testSched,
						"data_stream": mapstr.M{
							"namespace": testNs,
							"dataset":   testDs,
							"type":      testType,
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":       testRootId,
				"type":     testType,
				"schedule": testSched,
				"data_stream": mapstr.M{
					"namespace": testNs,
					"dataset":   testDs,
					"type":      testType,
				},
			}),
		},
		{
			name: "split data stream",
			cfg: mapstr.M{
				"id":   testRootId,
				"type": testType,
				"data_stream": mapstr.M{
					"namespace": testNs,
				},
				"streams": []mapstr.M{
					{
						"id":       testStreamId,
						"origin":   testOrigin,
						"type":     testType,
						"schedule": testSched,
						"data_stream": mapstr.M{
							"type":    testType,
							"dataset": testDs,
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":       testStreamId,
				"type":     testType,
				"schedule": testSched,
				"origin":   testOrigin,
				"data_stream": mapstr.M{
					"namespace": testNs,
					"dataset":   testDs,
					"type":      testType,
				},
			}),
		},
		{
			name: "base is last, not first stream",
			cfg: mapstr.M{
				"id": testRootId,
				"data_stream": mapstr.M{
					"namespace": testNs,
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
						"id":       testStreamId,
						"type":     testType,
						"schedule": testSched,
						"origin":   testOrigin,
						"data_stream": mapstr.M{
							"type":    testType,
							"dataset": testDs,
						},
					},
				},
			},
			v: lookslike.MustCompile(mapstr.M{
				"id":       testStreamId,
				"type":     testType,
				"schedule": testSched,
				"data_stream": mapstr.M{
					"namespace": testNs,
					"type":      testType,
					"dataset":   testDs,
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			src, err := conf.NewConfigFrom(test.cfg)
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
