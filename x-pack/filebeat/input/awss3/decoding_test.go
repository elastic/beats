// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// all test files are read from the "testdata" directory
const testDataPath = "testdata"

func TestParquetDecoding(t *testing.T) {
	testCases := []struct {
		name          string
		file          string
		contentType   string
		numEvents     int
		assertAgainst string
		config        *readerConfig
	}{
		{
			name:      "test decoding of a parquet file and compare the number of events with batch size 1",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoderConfig{
					Codec: "parquet",
					Parquet: parquetCodecConfig{
						ProcessParallel: true,
						BatchSize:       1,
					},
				},
			},
		},
		{
			name:      "test decoding of a parquet file and compare the number of events with batch size 100",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoderConfig{
					Codec: "parquet",
					Parquet: parquetCodecConfig{
						ProcessParallel: true,
						BatchSize:       100,
					},
				},
			},
		},
		{
			name:      "test decoding of a parquet file and compare the number of events with default parquet config",
			file:      "vpc-flow.gz.parquet",
			numEvents: 1304,
			config: &readerConfig{
				Decoding: decoderConfig{
					Codec: "parquet",
				},
			},
		},
		{
			name:          "test decoding of a parquet file and compare the number of events along with the content",
			file:          "cloudtrail.parquet",
			numEvents:     1,
			assertAgainst: "cloudtrail.ndjson",
			config: &readerConfig{
				Decoding: decoderConfig{
					Codec: "parquet",
					Parquet: parquetCodecConfig{
						ProcessParallel: true,
						BatchSize:       1,
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
			// if assertAgainst is not empty, then compare the events with the target file
			// there is a chance for this comparison to become flaky if number of events > 1 as
			// the order of events are not guaranteed by beats
			if tc.assertAgainst != "" {
				targetFile, err := os.Open(filepath.Join(testDataPath, tc.assertAgainst))
				assert.NoError(t, err)
				defer targetFile.Close()
				targetData := readJSONFromFile(t, targetFile)
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
func readJSONFromFile(t *testing.T, file *os.File) []string {
	var data []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("failed to read ndjson file: %v", err)
	}

	return data
}
