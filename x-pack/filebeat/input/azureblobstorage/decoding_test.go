// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// all test files are read from the "testdata" directory
const testDataPath = "testdata"

func TestDecoding(t *testing.T) {
	logp.TestingSetup()
	log := logp.L()

	testCases := []struct {
		name          string
		file          string
		content       string
		contentType   string
		numEvents     int
		assertAgainst string
		config        decoderConfig
	}{
		{
			name:          "gzip_csv",
			file:          "txn.csv.gz",
			content:       "text/csv",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: decoderConfig{
				Codec: &codecConfig{
					CSV: &csvCodecConfig{
						Enabled: true,
						Comma:   ptr[configRune](' '),
					},
				},
			},
		},
		{
			name:          "csv",
			file:          "txn.csv",
			content:       "text/csv",
			numEvents:     4,
			assertAgainst: "txn.json",
			config: decoderConfig{
				Codec: &codecConfig{
					CSV: &csvCodecConfig{
						Enabled: true,
						Comma:   ptr[configRune](' '),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(testDataPath, tc.file)
			if tc.contentType == "" {
				tc.contentType = "application/octet-stream"
			}
			f, err := os.Open(file)
			if err != nil {
				t.Fatalf("failed to open test data: %v", err)
			}
			defer f.Close()
			p := &pub{t: t}
			item := &azcontainer.BlobItem{
				Name: ptr("test_blob"),
				Properties: &azcontainer.BlobProperties{
					ContentType:  ptr(tc.content),
					LastModified: &time.Time{},
				},
			}
			j := newJob(&blob.Client{}, item, "https://foo.blob.core.windows.net/", newState(), &Source{}, p, nil, log)
			j.src.ReaderConfig.Decoding = tc.config
			err = j.decode(context.Background(), f, "test")
			if err != nil {
				t.Errorf("unexpected error calling decode: %v", err)
			}

			events := p.events
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

type pub struct {
	t      *testing.T
	events []beat.Event
}

func (p *pub) Publish(e beat.Event, _cursor interface{}) error {
	p.t.Logf("%v\n", e.Fields)
	p.events = append(p.events, e)
	return nil
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
	want    decoderConfig
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
		want: decoderConfig{&codecConfig{
			CSV: &csvCodecConfig{
				Enabled: true,
				Comma:   ptr[configRune](' '),
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
		want: decoderConfig{&codecConfig{
			CSV: &csvCodecConfig{
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
		want: decoderConfig{&codecConfig{
			CSV: &csvCodecConfig{
				Enabled: true,
				Comma:   ptr[configRune]('\x00'),
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
}

func TestCodecConfig(t *testing.T) {
	for _, test := range codecConfigTests {
		t.Run(test.name, func(t *testing.T) {
			c, err := conf.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Fatalf("unexpected error unmarshaling config: %v", err)
			}

			var got decoderConfig
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
