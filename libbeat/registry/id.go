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
