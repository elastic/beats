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
	"fmt"
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

func TestAll(t *testing.T) {
	all, err := FromString(fmt.Sprintf("%x", allMask), 16)
	assert.Nil(t, err)
	assert.Equal(t, all[0], "CAP_ALL")

	all, err = FromUint64(allMask)
	assert.Nil(t, err)
	assert.Equal(t, len(all), 1)
	assert.Equal(t, all[0], "CAP_ALL")
}

func TestOverflow(t *testing.T) {
	sl, err := FromUint64(^uint64(0))
	assert.Nil(t, err)
	assert.Equal(t, len(sl), 64)

	assertHasCap(t, sl, "CAP_CHOWN")
	assertHasCap(t, sl, "CAP_DAC_OVERRIDE")
	assertHasCap(t, sl, "CAP_DAC_READ_SEARCH")
	assertHasCap(t, sl, "CAP_FOWNER")
	assertHasCap(t, sl, "CAP_FSETID")
	assertHasCap(t, sl, "CAP_KILL")
	assertHasCap(t, sl, "CAP_SETGID")
	assertHasCap(t, sl, "CAP_SYS_MODULE")
	assertHasCap(t, sl, "CAP_SYS_RAWIO")
	assertHasCap(t, sl, "CAP_IPC_LOCK")
	assertHasCap(t, sl, "CAP_MAC_OVERRIDE")
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

	assert.Equal(t, found, 1)
}
