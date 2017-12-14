package file_integrity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFBEncodeDecode(t *testing.T) {
	e := testEvent()

	builder, release := fbGetBuilder()
	defer release()
	data := fbEncodeEvent(builder, e)
	t.Log("encoded length:", len(data))

	out := fbDecodeEvent(e.Path, data)
	if out == nil {
		t.Fatal("decode returned nil")
	}

	assert.Equal(t, *e.Info, *out.Info)
	e.Info, out.Info = nil, nil
	assert.Equal(t, e, out)
}

func BenchmarkFBEncodeEvent(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	e := testEvent()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		builder.Reset()
		fbEncodeEvent(builder, e)
	}
}

func BenchmarkFBEventDecode(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	e := testEvent()
	data := fbEncodeEvent(builder, e)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if event := fbDecodeEvent(e.Path, data); event == nil {
			b.Fatal("failed to decode")
		}
	}
}

// JSON benchmarks for comparisons.

func BenchmarkJSONEventEncoding(b *testing.B) {
	e := testEvent()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(e)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONEventDecode(b *testing.B) {
	e := testEvent()
	data, err := json.Marshal(e)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var e *Event
		err := json.Unmarshal(data, &e)
		if err != nil {
			b.Fatal(err)
		}
	}
}
