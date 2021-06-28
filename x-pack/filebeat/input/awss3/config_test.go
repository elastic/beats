// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
)

func TestConfig(t *testing.T) {
	const queueURL = "https://example.com"
	makeConfig := func() config {
		// Have a separate copy of defaults in the test to make it clear when
		// anyone changes the defaults.
		return config{
			QueueURL:            queueURL,
			APITimeout:          120 * time.Second,
			VisibilityTimeout:   300 * time.Second,
			FIPSEnabled:         false,
			MaxNumberOfMessages: 5,
			ReaderConfig: readerConfig{
				BufferSize:     16 * humanize.KiByte,
				MaxBytes:       10 * humanize.MiByte,
				LineTerminator: readfile.AutoLineTerminator,
			},
		}
	}

	testCases := []struct {
		name        string
		config      common.MapStr
		expectedErr string
		expectedCfg func() config
	}{
		{
			"input with defaults",
			common.MapStr{
				"queue_url": queueURL,
			},
			"",
			makeConfig,
		},
		{
			"input with file_selectors",
			common.MapStr{
				"queue_url": queueURL,
				"file_selectors": []common.MapStr{
					{
						"regex": "/CloudTrail/",
					},
				},
			},
			"",
			func() config {
				c := makeConfig()
				regex := match.MustCompile("/CloudTrail/")
				c.FileSelectors = []fileSelectorConfig{
					{
						Regex:        &regex,
						ReaderConfig: c.ReaderConfig,
					},
				}
				return c
			},
		},
		{
			"error on api_timeout == 0",
			common.MapStr{
				"queue_url":   queueURL,
				"api_timeout": "0",
			},
			"api_timeout <0s> must be greater than 0 and less than 1/2 of the visibility_timeout (2m30s)",
			nil,
		},
		{
			"error on api_timeout less than visibility_timeout/2",
			common.MapStr{
				"queue_url":   queueURL,
				"api_timeout": "3m",
			},
			"api_timeout <3m0s> must be greater than 0 and less than 1/2 of the visibility_timeout (2m30s)",
			nil,
		},
		{
			"error on visibility_timeout == 0",
			common.MapStr{
				"queue_url":          queueURL,
				"visibility_timeout": "0",
			},
			"visibility_timeout <0s> must be greater than 0 and less than or equal to 12h",
			nil,
		},
		{
			"error on visibility_timeout > 12h",
			common.MapStr{
				"queue_url":          queueURL,
				"visibility_timeout": "12h1ns",
			},
			"visibility_timeout <12h0m0.000000001s> must be greater than 0 and less than or equal to 12h",
			nil,
		},
		{
			"error on max_number_of_messages == 0",
			common.MapStr{
				"queue_url":              queueURL,
				"max_number_of_messages": "0",
			},
			"max_number_of_messages <0> must be greater than 0 and less than or equal to 10",
			nil,
		},
		{
			"error on max_number_of_messages > 10",
			common.MapStr{
				"queue_url":              queueURL,
				"max_number_of_messages": "11",
			},
			"max_number_of_messages <11> must be greater than 0 and less than or equal to 10",
			nil,
		},
		{
			"error on buffer_size == 0 ",
			common.MapStr{
				"queue_url":   queueURL,
				"buffer_size": "0",
			},
			"buffer_size <0> must be greater than 0",
			nil,
		},
		{
			"error on max_bytes == 0 ",
			common.MapStr{
				"queue_url": queueURL,
				"max_bytes": "0",
			},
			"max_bytes <0> must be greater than 0",
			nil,
		},
		{
			"error on expand_event_list_from_field and content_type != application/json ",
			common.MapStr{
				"queue_url":                    queueURL,
				"expand_event_list_from_field": "Records",
				"content_type":                 "text/plain",
			},
			"content_type must be `application/json` when expand_event_list_from_field is used",
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in := common.MustNewConfigFrom(tc.config)

			c := defaultConfig()
			if err := in.Unpack(&c); err != nil {
				if tc.expectedErr != "" {
					assert.Contains(t, err.Error(), tc.expectedErr)
					return
				}
				t.Fatal(err)
			}

			if tc.expectedCfg == nil {
				t.Fatal("missing expected config in test case")
			}
			assert.EqualValues(t, tc.expectedCfg(), c)
		})
	}
}
