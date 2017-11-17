package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapStrPointer(t *testing.T) {
	data := MapStr{
		"foo": "bar",
	}

	p := NewMapStrPointer(data)
	assert.Equal(t, p.Get(), data)

	newData := MapStr{
		"new": "data",
	}
	p.Set(newData)
	assert.Equal(t, p.Get(), newData)
}

func BenchmarkMapStrPointer(b *testing.B) {
	p := NewMapStrPointer(MapStr{"counter": 0})
	go func() {
		counter := 0
		for {
			counter++
			p.Set(MapStr{"counter": counter})
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for n := 0; n < b.N; n++ {
		_ = p.Get()
	}
}
