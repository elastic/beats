package lutool

import (
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Table using fine-grained locking to reduce congestion.
type cache struct {
	mutex   sync.Mutex
	bins    map[uint64]*bin
	janitor *janitor
}

// janitor removes expired and unused entries from cache.
//
// Collect and remove old entries in 3 steps:
// 1. create snapshot of bins table using table mutex.
//    After creating snapshot, table can be accessed concurrently
// 2. Search and remove expired entries for each single bin. Bin will be locked, but
//    Other bins can be used concurrently.
// 3. Probe and remove empty bins from table.
type janitor struct {
	ticks chan time.Time
	cache *cache
}

type bin struct {
	mutex   sync.Mutex
	entries []*binEntry
}

type binEntry struct {
	ts    time.Time
	exec  execOnce
	key   Key
	value binValue
}

type binValue interface {
	error() error
	lastRun() time.Time
	backoff() time.Duration
	value() common.MapStr
}

type binError struct {
	ts time.Time
	d  time.Duration
	e  error
}

type binLookupValue common.MapStr

var errConvertString = errors.New("can not convert to string")

func newCache(tick, timeout time.Duration) *cache {
	t := &cache{
		mutex: sync.Mutex{},
		bins:  map[uint64]*bin{},
	}

	if timeout > 0 {
		// TODO: how to stop the go-routine? No processor shutdown logic
		j := newJanitor(t)
		go j.run(tick, timeout)
		t.janitor = j
	}

	return t
}

func (t *cache) Close() {
	t.janitor.stop()
}

func (t *cache) getEntry(ts time.Time, k Key) (*binEntry, error) {
	e, err := t.doGetEntry(k)
	t.janitor.signalTS(ts)
	return e, err
}

func (t *cache) doGetEntry(k Key) (*binEntry, error) {
	hash := fnv.New64()
	if err := k.Hash(hash); err != nil {
		return nil, err
	}

	bin := t.getBin(hash.Sum64())
	defer bin.mutex.Unlock()

	for _, entry := range bin.entries {
		if entry.key.Equals(&k) {
			entry.ts = time.Now()
			return entry, nil
		}
	}

	debugf("Create new table entry for key: %v", k)
	entry := &binEntry{ts: time.Now(), key: k}
	bin.entries = append(bin.entries, entry)
	return entry, nil
}

func (t *cache) getBin(hash uint64) *bin {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	b := t.bins[hash]
	if b == nil {
		b = &bin{}
		t.bins[hash] = b
	}
	b.mutex.Lock()
	return b
}

func newJanitor(c *cache) *janitor {
	return &janitor{
		cache: c,
		ticks: make(chan time.Time, 1),
	}
}

func (j *janitor) stop() {
	close(j.ticks)
}

func (j *janitor) signalTS(ts time.Time) {
	// try to push current timestamp
	select {
	case j.ticks <- ts:
	default:
	}
}

func (j *janitor) run(tick, timeout time.Duration) {
	last := time.Time{}
	for ts := range j.ticks {
		if ts.Before(last) {
			continue
		}
		if ts.Sub(last) < tick {
			continue
		}

		j.collect(ts, timeout)
		last = ts
	}
}

func (j *janitor) collect(ts time.Time, timeout time.Duration) {
	debugf("collect old entries from table")
	hashes, bins := j.makeSnapshot()
	dropped := hashes[:0] // reuse 'hashes' slice to collect hashes to be dropped
	for i := range bins {
		drop := j.collectBin(ts, timeout, bins[i])
		if drop {
			dropped = append(dropped, hashes[i])
		}
	}

	// try to drop bins without any entries
	for _, hash := range dropped {
		j.collectEmptyBin(hash)
	}
}

func (j *janitor) makeSnapshot() ([]uint64, []*bin) {
	t := j.cache

	// take snapshot of main table
	t.mutex.Lock()
	defer t.mutex.Unlock()

	hashes := make([]uint64, 0, len(t.bins))
	bins := make([]*bin, 0, len(t.bins))
	for hash, bin := range t.bins {
		hashes = append(hashes, hash)
		bins = append(bins, bin)
	}
	return hashes, bins
}

func (j *janitor) collectBin(now time.Time, timeout time.Duration, b *bin) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	entries := b.entries[:0]
	for _, entry := range b.entries {
		if now.Sub(entry.ts) > timeout {
			debugf("delete entry for key: ", entry.key)
			continue
		}

		entries = append(entries, entry)
	}
	b.entries = entries
	return len(entries) == 0
}

func (j *janitor) collectEmptyBin(hash uint64) {
	t := j.cache
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Safe to get and drop bin here, as bin can only be acquired when table
	// mutex is locked and before updating entries, the bins mutex is locked using
	// hand-over-hand locking. In case bin is locked, it will be updated, before
	// cleanup gets lock to check if bin has been invalidated.
	b := t.bins[hash]
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if len(b.entries) == 0 {
		delete(t.bins, hash)
	}
}

func (e binError) error() error           { return e.e }
func (e binError) lastRun() time.Time     { return e.ts }
func (e binError) backoff() time.Duration { return e.d }
func (e binError) value() common.MapStr   { return nil }

func (v binLookupValue) error() error           { return nil }
func (v binLookupValue) lastRun() time.Time     { return time.Time{} }
func (v binLookupValue) backoff() time.Duration { return 0 }
func (v binLookupValue) value() common.MapStr   { return common.MapStr(v) }
