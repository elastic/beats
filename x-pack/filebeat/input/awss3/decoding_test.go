// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v9/x-pack/libbeat/reader/decoder"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// all test files are read from the "testdata" directory
const testDataPath = "testdata"

func TestDecoding(t *testing.T) {
	testCases := []struct {
		name          string
		file          string
		contentType   string
		numEvents     int
		assertAgainst string
		config        *readerConfig
	}{
		{
			name:      "parquet_batch_size_1",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						Parquet: &decoder.ParquetCodecConfig{
							ProcessParallel: true,
							BatchSize:       1,
						},
					},
				},
			},
		},
		{
			name:      "parquet_batch_size_100",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						Parquet: &decoder.ParquetCodecConfig{
							ProcessParallel: true,
							BatchSize:       100,
						},
					},
				},
			},
		},
		{
			name:      "parquet_default",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						Parquet: &decoder.ParquetCodecConfig{
							Enabled: true,
						},
					},
				},
			},
		},
		{
			name:          "parquet_default_content_check",
			file:          "cloudtrail.parquet",
			numEvents:     1,
			assertAgainst: "cloudtrail.json",
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						Parquet: &decoder.ParquetCodecConfig{
							Enabled:         true,
							ProcessParallel: true,
							BatchSize:       1,
						},
					},
				},
			},
		},
		{
			name:          "gzip_csv",
			file:          "txn.csv.gz",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						CSV: &decoder.CSVCodecConfig{
							Enabled: true,
							Comma:   ptr[decoder.Rune](' '),
						},
					},
				},
			},
		},
		{
			name:          "csv",
			file:          "txn.csv",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: &readerConfig{
				Decoding: decoder.Config{
					Codec: &decoder.CodecConfig{
						CSV: &decoder.CSVCodecConfig{
							Enabled: true,
							Comma:   ptr[decoder.Rune](' '),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(testDataPath, tc.file)
			sel := fileSelectorConfig{ReaderConfig: *tc.config}
			if tc.contentType == "" {
				tc.contentType = "application/octet-stream"
			}
			// uses the s3_objects test method to perform the test
			events := testProcessS3Object(t, file, tc.contentType, tc.numEvents, sel)
			// If assertAgainst is not empty, then compare the events with the target file.
			if tc.assertAgainst != "" {
				targetData := readJSONFromFile(t, filepath.Join(testDataPath, tc.assertAgainst))
				assert.Equal(t, len(targetData), len(events))

				for i, event := range events {
					msg, err := event.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.JSONEq(t, targetData[i], msg.(string))
				}
			}
		})
	}
}

// readJSONFromFile reads the json file and returns the data as a slice of strings
func readJSONFromFile(t *testing.T, filepath string) []string {
	fileBytes, err := os.ReadFile(filepath)
	assert.NoError(t, err)
	var rawMessages []json.RawMessage
	err = json.Unmarshal(fileBytes, &rawMessages)
	assert.NoError(t, err)
	var data []string

	for _, rawMsg := range rawMessages {
		data = append(data, string(rawMsg))
	}
	return data
}

var codecConfigTests = []struct {
	name    string
	yaml    string
	want    decoder.Config
	wantErr error
}{
	{
		name: "handle_rune",
		yaml: `
codec:
  csv:
    enabled: true
    comma: ' '
    comment: '#'
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
					Comma:   ptr[decoder.Rune](' '),
					Comment: '#',
				},
			}},
	},
	{
		name: "no_comma",
		yaml: `
codec:
  csv:
    enabled: true
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
				},
			}},
	},
	{
		name: "null_comma",
		yaml: `
codec:
  csv:
    enabled: true
    comma: "\u0000"
`,
		want: decoder.Config{
			Codec: &decoder.CodecConfig{
				CSV: &decoder.CSVCodecConfig{
					Enabled: true,
					Comma:   ptr[decoder.Rune]('\x00'),
				},
			}},
	},
	{
		name: "bad_rune",
		yaml: `
codec:
  csv:
    enabled: true
    comma: 'this is too long'
`,
		wantErr: errors.New(`single character option given more than one character: "this is too long" accessing 'codec.csv.comma'`),
	},
	{
		name: "confused",
		yaml: `
codec:
  csv:
    enabled: true
  parquet:
    enabled: true
`,
		wantErr: errors.New(`more than one decoder configured accessing 'codec'`),
	},
}

func TestCodecConfig(t *testing.T) {
	for _, test := range codecConfigTests {
		t.Run(test.name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatalf("unexpected error unmarshaling config: %v", err)
			}

			var got decoder.Config
			err = c.Unpack(&got)
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error unpacking config: got:%v want:%v", err, test.wantErr)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}

func ptr[T any](v T) *T { return &v }
