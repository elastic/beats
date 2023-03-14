// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testPackage() []*Package {
	return []*Package{
		{
			Name:        "foo",
			Version:     "1.2.3",
			Release:     "1",
			Arch:        "amd64",
			License:     "bar",
			InstallTime: time.Unix(1591021924, 0).UTC(),
			Size:        1234,
			Summary:     "Foo stuff",
			URL:         "http://foo.example.com",
			Type:        "rpm",
		},
		{
			Name:        "csv",
			Version:     "1.5.7",
			Release:     "2",
			Arch:        "amd64",
			License:     "bar",
			InstallTime: time.Unix(1591021924, 0).UTC(),
			Size:        2456,
			Summary:     "Csv stuff",
			URL:         "http://csv.example.com",
			Type:        "rpm",
		},
	}
}

func TestFBEncodeDecode(t *testing.T) {
	p := testPackage()
	builder, release := fbGetBuilder()
	defer release()
	data := encodePackages(builder, p)
	t.Log("encoded length:", len(data))

	out, err := decodePackagesFromContainer(data)
	if err != nil {
		t.Error(err)
	}

	// since decoded slice is in reversed order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	assert.Equal(t, len(p), len(out))
	for i := 0; i < len(p); i++ {
		assert.Equal(t, p[i], out[i])
	}
}
