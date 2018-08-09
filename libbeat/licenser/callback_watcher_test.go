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

package licenser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallbackWatcher(t *testing.T) {
	t.Run("when no callback is set do not execute anything", func(t *testing.T) {
		w := &CallbackWatcher{}
		w.OnNewLicense(License{})
		w.OnManagerStopped()
	})

	t.Run("proxy call to callback function", func(t *testing.T) {
		c := 0
		w := &CallbackWatcher{
			New:     func(license License) { c++ },
			Stopped: func() { c++ },
		}
		w.OnNewLicense(License{})
		w.OnManagerStopped()
		assert.Equal(t, 2, c)
	})
}
