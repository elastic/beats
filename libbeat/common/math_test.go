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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound(t *testing.T) {
	assert.EqualValues(t, 0.5, Round(0.5, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 0.5, Round(0.50004, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 0.5001, Round(0.50005, DefaultDecimalPlacesCount))

	assert.EqualValues(t, 1234.5, Round(1234.5, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 1234.5, Round(1234.50004, DefaultDecimalPlacesCount))
	assert.EqualValues(t, 1234.5001, Round(1234.50005, DefaultDecimalPlacesCount))
}
