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

//go:build linux

package hasher

import (
	"errors"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sys/unix"

	"github.com/elastic/elastic-agent-libs/logp"
)

// CachedHasher is a metadata aware FileHasher with a LRU cache on top of it.
type CachedHasher struct {
	hasher   *FileHasher
	hashLRU  *lru.Cache[string, hashEntry]
	hasStatx bool
	stats    CachedHasherStats
	log      *logp.Logger
}

// CachedHasherStats are basics statistics for debugging and testing.
type CachedHasherStats struct {
	Hits          uint64
	Misses        uint64
	Invalidations uint64
	Evictions     uint64
}

// hashEntry is an entry in the LRU cache.
type hashEntry struct {
	statx  unix.Statx_t
	hashes map[HashType]Digest
}

// NewFileHasherWithCache creates a CachedHasher with space up to size elements.
func NewFileHasherWithCache(c Config, size int) (*CachedHasher, error) {
	// We don't rate limit our hashes, we cache
	c.ScanRateBytesPerSec = 0
	hasher, err := NewFileHasher(c, nil)
	if err != nil {
		return nil, err
	}
	hashLRU, err := lru.New[string, hashEntry](size)
	if err != nil {
		return nil, err
	}
	var nada unix.Statx_t
	hasStatx := unix.Statx(-1, "/", 0, unix.STATX_ALL|unix.STATX_MNT_ID, &nada) != unix.ENOSYS

	return &CachedHasher{
		hasher:   hasher,
		hashLRU:  hashLRU,
		hasStatx: hasStatx,
		log:      logp.NewLogger("cached_hasher"),
	}, nil
}

// HashFile looks up a hashEntry in the cache, if the lookup fails,
// the hash is computed, inserted in the cache, and returned. If the
// lookup succeeds but the file metadata changed, the entry is evicted
// and refreshed.
func (ch *CachedHasher) HashFile(path string) (map[HashType]Digest, error) {
	var x time.Time
	if logp.IsDebug("cached_hasher") {
		x = time.Now()
	}

	// See if we have it stored
	if entry, ok := ch.hashLRU.Get(path); ok {
		statx, err := ch.statxFromPath(path)
		if err != nil {
			// No point in keeping an entry if we can't compare
			if !ch.hashLRU.Remove(path) {
				err := errors.New("can't remove existing entry, this is a bug")
				ch.log.Error(err)
			}
			return nil, err
		}
		// If metadata didn't change, this is a good entry, if not fall and rehash
		if statx == entry.statx {
			ch.log.Debugf("hit (%s) took %v", path, time.Since(x))
			ch.stats.Hits++
			return entry.hashes, nil
		}
		// Zap from lru
		if !ch.hashLRU.Remove(path) {
			err := errors.New("can't remove existing entry, this is a bug")
			ch.log.Error(err)
			return nil, err
		} else {
			ch.stats.Invalidations++
			ch.log.Debugf("invalidate (%s)", path)
		}
	}
	// Nah, so do the hard work
	hashes, err := ch.hasher.HashFile(path)
	if err != nil {
		return nil, err
	}
	// Fetch metadata
	statx, err := ch.statxFromPath(path)
	if err != nil {
		return nil, err
	}
	// Insert
	entry := hashEntry{hashes: hashes, statx: statx}
	if ch.hashLRU.Add(path, entry) {
		ch.stats.Evictions++
		ch.log.Debugf("evict (%s)", path)
	}

	ch.log.Debugf("miss (%s) took %v", path, time.Since(x))
	ch.stats.Misses++

	return entry.hashes, nil
}

// Close releases all resources
func (ch *CachedHasher) Close() {
	ch.hashLRU.Purge()
}

// Stats returns basic stats suitable for debugging and testing
func (ch *CachedHasher) Stats() CachedHasherStats {
	return ch.stats
}

// statxFromPath returns the metadata (unix.Statx_t) of path. In case
// the system doesn't support statx(2), it uses stat(2) and fills the
// corresponding members of unix.Statx_t, leaving the remaining members
// with a zero value.
func (ch *CachedHasher) statxFromPath(path string) (unix.Statx_t, error) {
	if ch.hasStatx {
		var tmpstx unix.Statx_t
		err := unix.Statx(-1, path, 0, unix.STATX_ALL|unix.STATX_MNT_ID, &tmpstx)
		if err != nil {
			return unix.Statx_t{}, err
		}

		// This might look stupid, but it guarantees we only compare
		// the members we are really interested, unix.Statx_t grows
		// with time, so if they ever add a member that changes all
		// the time, we don't introduce a bug where we compare things
		// we don't want to.
		return unix.Statx_t{
			Mask:            tmpstx.Mask,
			Blksize:         tmpstx.Blksize,
			Attributes:      tmpstx.Attributes,
			Nlink:           tmpstx.Nlink,
			Uid:             tmpstx.Uid,
			Gid:             tmpstx.Gid,
			Mode:            tmpstx.Mode,
			Ino:             tmpstx.Ino,
			Size:            tmpstx.Size,
			Blocks:          tmpstx.Blocks,
			Attributes_mask: tmpstx.Attributes_mask,
			Btime:           tmpstx.Btime,
			Ctime:           tmpstx.Ctime,
			Mtime:           tmpstx.Mtime,
			Rdev_minor:      tmpstx.Rdev_minor,
			Rdev_major:      tmpstx.Rdev_major,
			// no Atime
			// no Dio_mem_align
			// no Dio_offset_align
			// no Subvol
			// no Atomic_write_unit_min
			// no Atomic_write_unit_max
			// no Atomic_write_segments_max
		}, nil
	}

	// No statx(2), fallback to stat(2)
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return unix.Statx_t{}, err
	}

	return unix.Statx_t{
		Dev_major:  unix.Major(st.Dev),
		Dev_minor:  unix.Minor(st.Dev),
		Ino:        st.Ino,
		Nlink:      uint32(st.Nlink),
		Mode:       uint16(st.Mode),
		Uid:        st.Uid,
		Gid:        st.Gid,
		Rdev_major: unix.Major(st.Rdev),
		Rdev_minor: unix.Minor(st.Rdev),
		Size:       uint64(st.Size),
		Blksize:    uint32(st.Blksize),
		Blocks:     uint64(st.Blocks),
		Mtime: unix.StatxTimestamp{
			Nsec: uint32(st.Mtim.Nsec),
			Sec:  st.Mtim.Sec,
		},
		Ctime: unix.StatxTimestamp{
			Nsec: uint32(st.Ctim.Nsec),
			Sec:  st.Ctim.Sec,
		},
		// no Atime
	}, nil
}
