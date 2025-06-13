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

package backoff

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryer_NoRetries(t *testing.T) {
	r := NewRetryer(0, 0, 0)
	count := 0

	err := r.Retry(context.Background(), func() error {
		count++
		return fmt.Errorf("foo")
	})

	assert.Error(t, err)
	assert.Equal(t, 1, count, "expected function to be called once")
}

func TestRetryer_SuccessfulRetry(t *testing.T) {
	r := NewRetryer(10, 0, 0)
	count := 0

	err := r.Retry(context.Background(), func() error {
		count++
		if count == 2 {
			return nil
		}
		return fmt.Errorf("foo")
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, count, "expected function to be called twice")
}

func TestRetryer_MaxRetries(t *testing.T) {
	r := NewRetryer(2, 0, 0)
	count := 0

	err := r.Retry(context.Background(), func() error {
		count++
		return fmt.Errorf("foo")
	})

	assert.Error(t, err)
	assert.Equal(t, 3, count, "expected 2 retries plus the initial call")
}

func TestRetryer_ContextCancelled(t *testing.T) {
	r := NewRetryer(10, time.Second, 0)
	count := 0

	ctx, cancel := context.WithCancel(context.Background())

	err := r.Retry(ctx, func() error {
		count++

		// cancel on the first call - should stop further retries even though we return an error
		cancel()
		return fmt.Errorf("foo")
	})
	assert.Error(t, err)
	assert.Equal(t, 1, count, "expected 1 retry before context cancellation")
}
