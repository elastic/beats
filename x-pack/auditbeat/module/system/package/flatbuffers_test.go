package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testPackage() []*Package {
	return []*Package{
		&Package{
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
	}
}

func TestFBEncodeDecode(t *testing.T) {
	p := testPackage()
	builder, release := fbGetBuilder()
	defer release()
	data := encodePackages(builder, p)
	t.Log("encoded length:", len(data))

	out := decodePackagesFromContainer(data, nil)
	if out == nil {
		t.Fatal("decode returned nil")
	}

	assert.Equal(t, len(p), len(out))
	for i := 0; i < len(p); i++ {
		assert.Equal(t, p[i], out[i])
	}
}
