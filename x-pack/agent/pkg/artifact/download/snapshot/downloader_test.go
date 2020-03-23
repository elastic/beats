// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
)

type SuccDownloader struct {
	called bool
}

func (d *SuccDownloader) Download(ctx context.Context, a, b string) (string, error) {
	if strings.HasSuffix(b, "-SNAPSHOT") {
		d.called = true
	}
	return "succ", nil
}
func (d *SuccDownloader) Called() bool { return d.called }

func TestSnapshotDownloader(t *testing.T) {
	testCases := []testCase{
		testCase{
			downloader: &SuccDownloader{},
			checkFunc:  func(d CheckableDownloader) bool { return d.Called() },
		},
	}

	for _, tc := range testCases {
		d := NewDownloader(tc.downloader)
		r, _ := d.Download(nil, "a", "b")

		assert.Equal(t, true, r == "succ")

		assert.True(t, tc.checkFunc(tc.downloader))
	}
}

type CheckableDownloader interface {
	download.Downloader
	Called() bool
}

type testCase struct {
	downloader CheckableDownloader
	checkFunc  func(downloader CheckableDownloader) bool
}
