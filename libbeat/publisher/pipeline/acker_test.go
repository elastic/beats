package pipeline

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCountACK(t *testing.T) {
	dummyPipeline := &Pipeline{}
	dummyFunc := func(total, acked int) {}
	cAck := newCountACK(dummyPipeline, dummyFunc)
	assert.Equal(t, dummyPipeline, cAck.pipeline)
	assert.Equal(t, reflect.ValueOf(dummyFunc).Pointer(), reflect.ValueOf(cAck.fn).Pointer())
}

func TestMakeCountACK(t *testing.T) {
	dummyPipeline := &Pipeline{}
	dummyFunc := func(total, acked int) {}
	dummySema := &sema{}
	tests := []struct {
		canDrop            bool
		sema               *sema
		fn                 func(total, acked int)
		pipeline           *Pipeline
		expectedOutputType reflect.Value
	}{
		{canDrop: false, sema: dummySema, fn: dummyFunc, pipeline: dummyPipeline, expectedOutputType: reflect.ValueOf(&countACK{})},
		{canDrop: true, sema: dummySema, fn: dummyFunc, pipeline: dummyPipeline, expectedOutputType: reflect.ValueOf(&boundGapCountACK{})},
	}
	for _, test := range tests {
		output := makeCountACK(test.pipeline, test.canDrop, test.sema, test.fn)
		assert.Equal(t, test.expectedOutputType.String(), reflect.ValueOf(output).String())
	}
}
