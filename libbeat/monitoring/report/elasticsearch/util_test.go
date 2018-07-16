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

package elasticsearch

import (
	"testing"
	"time"
)

func TestStopper(t *testing.T) {
	runPar := func(name string, f func(*testing.T)) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f(t)
		})
	}

	st := newStopper()
	runPar("wait on channel stop", func(*testing.T) { <-st.C() })
	runPar("use wait", func(*testing.T) { st.Wait() })
	runPar("use dowait", func(t *testing.T) {
		i := 0
		st.DoWait(func() { i = 1 })
		if i != 1 {
			t.Error("callback did not run")
		}
	})

	// unblock all waiters
	time.Sleep(10 * time.Millisecond)
	st.Stop()

	// test either blocks or returns as stopper as been stopped
	t.Run("wait after stop", func(t *testing.T) { st.Wait() })

	// check subsequent stop does not panic
	st.Stop()
	st.Stop()
}
