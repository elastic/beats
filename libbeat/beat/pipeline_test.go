// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package beat

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ OutputListener = (*NoopOutputListener)(nil)
var _ OutputListener = (*CountOutputListener)(nil)
var _ OutputListener = (*CombinedOutputListener)(nil)

func TestCombinedOutputListener(t *testing.T) {
	a := &CountOutputListener{}
	b := &CountOutputListener{}

	combined := &CombinedOutputListener{A: a, B: b}
	combined.NewEvent()
	combined.Acked()
	combined.DeadLetter()
	combined.Dropped()
	combined.DuplicateEvents()
	combined.ErrTooMany()
	combined.RetryableError()

	want := int64(1)

	assert.Equal(t, want, a.NewLoad(),
		fmt.Sprintf("a.NewLoad() should be %d", want))
	assert.Equal(t, want, a.AckedLoad(),
		fmt.Sprintf("a.AckedLoad() should be %d", want))
	assert.Equal(t, want, a.DeadLetterLoad(),
		fmt.Sprintf("a.DeadLetterLoad() should be %d", want))
	assert.Equal(t, want, a.DroppedLoad(),
		fmt.Sprintf("a.DroppedLoad() should be %d", want))
	assert.Equal(t, want, a.DuplicateEventsLoad(),
		fmt.Sprintf("a.DuplicateEventsLoad() should be %d", want))
	assert.Equal(t, want, a.ErrTooManyLoad(),
		fmt.Sprintf("a.ErrTooManyLoad() should be %d", want))
	assert.Equal(t, want, a.RetryableErrorsLoad(),
		fmt.Sprintf("a.RetryableErrorsLoad() should be %d", want))

	assert.Equal(t, want, b.NewLoad(),
		fmt.Sprintf("b.NewLoad() should be %d", want))
	assert.Equal(t, want, b.AckedLoad(),
		fmt.Sprintf("b.AckedLoad() should be %d", want))
	assert.Equal(t, want, b.DeadLetterLoad(),
		fmt.Sprintf("b.DeadLetterLoad() should be %d", want))
	assert.Equal(t, want, b.DroppedLoad(),
		fmt.Sprintf("b.DroppedLoad() should be %d", want))
	assert.Equal(t, want, b.DuplicateEventsLoad(),
		fmt.Sprintf("b.DuplicateEventsLoad() should be %d", want))
	assert.Equal(t, want, b.ErrTooManyLoad(),
		fmt.Sprintf("b.ErrTooManyLoad() should be %d", want))
	assert.Equal(t, want, b.RetryableErrorsLoad(),
		fmt.Sprintf("b.RetryableErrorsLoad() should be %d", want))
}
