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

package registry

import (
	"math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid"
)

type idGen rand.Rand

var idPool = sync.Pool{
	New: func() interface{} {
		seed := time.Now().UnixNano()
		rng := rand.New(rand.NewSource(seed))
		return (*idGen)(rng)
	},
}

func newIDGen() *idGen {
	return idPool.Get().(*idGen)
}

func (g *idGen) close() {
	idPool.Put(g)
}

func (g *idGen) Make() Key {
	ts := uint64(time.Now().Unix())
	id := ulid.MustNew(ts, (*rand.Rand)(g))
	k, err := id.MarshalText()
	if err != nil {
		panic(err)
	}
	return k
}
