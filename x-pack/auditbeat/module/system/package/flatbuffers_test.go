// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package pkg

import (
	"errors"
	"sync"
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
		{
			Name:        "vscode",
			Version:     "1.5.7",
			Release:     "2",
			Arch:        "",
			License:     "",
			InstallTime: time.Time{},
			Size:        0,
			Summary:     "",
			URL:         "",
			Type:        "",
			error:       errors.New("error unmarshalling JSON in /homebrew/Cellar: invalid JSON"),
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

// tests if the bufferPool is being shared in an unintended manner
func TestPoolPoison(t *testing.T) {
	p := testPackage()
	input := [][]*Package{p[:2], p[2:]}

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go poolPoison(input[i%2], t, &wg)
	}
	wg.Wait()
}

func poolPoison(p []*Package, t *testing.T, wg *sync.WaitGroup) {
	builder, release := fbGetBuilder()
	defer release()

	for k := 0; k < 1000; k++ {
		builder.Reset()

		data := encodePackages(builder, p)

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
	wg.Done()
}

func BenchmarkFBEncodePackages(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	p := testPackage()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		builder.Reset()
		encodePackages(builder, p)
	}
}

func BenchmarkFBDecodePackages(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	p := testPackage()
	data := encodePackages(builder, p)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if packages, err := decodePackagesFromContainer(data); err != nil || len(packages) == 0 {
			b.Fatal("failed to decode")
		}
	}
}
