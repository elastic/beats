// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composed

import (
	"errors"
	"testing"

	"github.com/elastic/fleet/x-pack/pkg/artifact/download"
	"github.com/stretchr/testify/assert"
)

type FailingDownloader struct {
	called bool
}

func (d *FailingDownloader) Download(a, b string) (string, error) {
	d.called = true
	return "", errors.New("failing")
}

func (d *FailingDownloader) Called() bool { return d.called }

type SuccDownloader struct {
	called bool
}

func (d *SuccDownloader) Download(a, b string) (string, error) {
	d.called = true
	return "succ", nil
}
func (d *SuccDownloader) Called() bool { return d.called }

func TestComposed(t *testing.T) {
	testCases := []testCase{
		testCase{
			downloaders:    []CheckableDownloader{&FailingDownloader{}, &SuccDownloader{}},
			checkFunc:      func(d []CheckableDownloader) bool { return d[0].Called() && d[1].Called() },
			expectedResult: true,
		}, testCase{
			downloaders:    []CheckableDownloader{&SuccDownloader{}, &SuccDownloader{}},
			checkFunc:      func(d []CheckableDownloader) bool { return d[0].Called() && !d[1].Called() },
			expectedResult: true,
		}, testCase{
			downloaders:    []CheckableDownloader{&SuccDownloader{}, &FailingDownloader{}},
			checkFunc:      func(d []CheckableDownloader) bool { return d[0].Called() && !d[1].Called() },
			expectedResult: true,
		}, testCase{
			downloaders:    []CheckableDownloader{&FailingDownloader{}, &FailingDownloader{}},
			checkFunc:      func(d []CheckableDownloader) bool { return d[0].Called() && d[1].Called() },
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		d := NewDownloader(tc.downloaders[0], tc.downloaders[1])
		r, _ := d.Download("a", "b")

		assert.Equal(t, tc.expectedResult, r == "succ")

		assert.True(t, tc.checkFunc(tc.downloaders))
	}
}

type CheckableDownloader interface {
	download.Downloader
	Called() bool
}

type testCase struct {
	downloaders    []CheckableDownloader
	checkFunc      func(downloaders []CheckableDownloader) bool
	expectedResult bool
}
