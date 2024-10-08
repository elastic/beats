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

package capabilities

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

func TestEmpty(t *testing.T) {
	sl, err := FromString("0", 16)
	assert.Nil(t, err)
	assert.Equal(t, len(sl), 0)

	sl, err = FromUint64(0)
	assert.Nil(t, err)
	assert.Equal(t, len(sl), 0)

	// assumes non root has no capabilities
	if os.Geteuid() != 0 {
		empty := cap.NewSet()
		self := cap.GetProc()
		d, err := self.Cf(empty)
		assert.Nil(t, err)
		assert.False(t, d.Has(cap.Effective))
		assert.False(t, d.Has(cap.Permitted))
		assert.False(t, d.Has(cap.Inheritable))
	}
}

func TestOverflow(t *testing.T) {
	sl, err := FromUint64(^uint64(0))
	assert.Nil(t, err)
	assert.Equal(t, len(sl), 64)

	for _, cap := range []string{
		"CAP_CHOWN",
		"CAP_DAC_OVERRIDE",
		"CAP_DAC_READ_SEARCH",
		"CAP_FOWNER",
		"CAP_FSETID",
		"CAP_KILL",
		"CAP_SETGID",
		"CAP_SYS_MODULE",
		"CAP_SYS_RAWIO",
		"CAP_IPC_LOCK",
		"CAP_MAC_OVERRIDE",
	} {
		assertHasCap(t, sl, cap)
	}
	if cap.MaxBits() <= 62 {
		assertHasCap(t, sl, "CAP_62")
	}
	if cap.MaxBits() <= 63 {
		assertHasCap(t, sl, "CAP_63")
	}
}

func assertHasCap(t *testing.T, sl []string, s string) {
	var found int

	for _, s2 := range sl {
		if s2 == s {
			found++
		}
	}

	assert.Equal(t, found, 1, s)
}
