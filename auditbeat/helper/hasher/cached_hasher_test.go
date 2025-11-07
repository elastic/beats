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
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type pattern struct {
	text     []byte
	sha256   string
	sha512   string
	sha3_384 string
}

var patternA = pattern{
	text:     []byte("Rather than love, than money, than fame, give me truth.\n"),
	sha256:   "19c76b22dd0bf97b0bf064e6587961938ba9f4ab73d034b0edac6c2c2829c0cd",
	sha512:   "e339322ed81208f930047e8b94db504f40a3e8bb2af75511925e3469488104edcd8eb8c613ea7fd0b08199a4d7061690512a05f66b50b4427470d6c8cf2d74a3",
	sha3_384: "9961640983a079920f74f2503feb5ce63325d6a6cd0138905e9419c4307043fa324217587062ac8648cbf43138a33034",
}

var patternB = pattern{
	text:     []byte("From womb to tomb, in kindness and crime.\n"),
	sha256:   "67606f88f25357b2b101e94bd02fc5da8dd2993391b88596c15bea77780a6a77",
	sha512:   "23c3779d7c6a8d4be2ca7a0bf412a2c99ea2f8a95ac21f56e3b9cb1bd0c0427bf2db91bbb484128f53ef48fbbfc97e525b328e1c4c0f8d24dd8a3f438c449736",
	sha3_384: "2034d02ad7b46831b9f2bf09b2eaa77bfcf70ebd136f29b95e6723cc6bf94d0fb7aae972dd2297b5507bb568cb65563b",
}

var config = Config{
	HashTypes:        []HashType{SHA256, SHA512, SHA3_384},
	MaxFileSize:      "1 KiB",
	MaxFileSizeBytes: 1024,
}

func TestCachedHasher(t *testing.T) {
	ch, err := NewFileHasherWithCache(config, 1)
	require.NoError(t, err)
	doTestCachedHasher(t, ch)
}

func TestCachedHasherWithStat(t *testing.T) {
	ch, err := NewFileHasherWithCache(config, 1)
	require.NoError(t, err)
	ch.hasStatx = false
	doTestCachedHasher(t, ch)
}

func doTestCachedHasher(t *testing.T, ch *CachedHasher) {
	// Create a file
	file := mkTemp(t)
	defer file.Close()

	// Write patternA and confirm first hash is a miss
	writePattern(t, file, patternA)
	ch.checkState(t, file.Name(), patternA, CachedHasherStats{Misses: 1})

	// Prove a subsequent hash hits the cache
	ch.checkState(t, file.Name(), patternA, CachedHasherStats{Misses: 1, Hits: 1})

	// Prove changing access time still causes a hit.
	// Note: we can't use os.Chtimes() to change _only_ atime, it
	// might end up modifying mtime since it can round/truncate
	// value we would get from file.Stat().ModTime()
	time.Sleep(time.Millisecond * 2)
	_, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	ch.checkState(t, file.Name(), patternA, CachedHasherStats{Misses: 1, Hits: 2})

	// Prove changing mtime invalides the entry, and causes a miss
	ostat, err := file.Stat()
	require.NoError(t, err)
	mtime := ostat.ModTime().Add(time.Hour)
	require.NoError(t, os.Chtimes(file.Name(), mtime, mtime))
	ch.checkState(t, file.Name(), patternA, CachedHasherStats{Misses: 2, Hits: 2, Invalidations: 1})

	// Write the second pattern, prove it's a miss
	writePattern(t, file, patternB)
	ch.checkState(t, file.Name(), patternB, CachedHasherStats{Misses: 3, Hits: 2, Invalidations: 2})

	// Hash something else, prove first one is evicted
	file2 := mkTemp(t)
	defer file2.Close()
	writePattern(t, file2, patternA)
	ch.checkState(t, file2.Name(), patternA, CachedHasherStats{Misses: 4, Hits: 2, Invalidations: 2, Evictions: 1})

	// If we go back and lookup the original path, prove we should evict again and it's a miss
	ch.checkState(t, file.Name(), patternB, CachedHasherStats{Misses: 5, Hits: 2, Invalidations: 2, Evictions: 2})

	// If we close, prove we purge
	require.Equal(t, ch.hashLRU.Len(), 1)
	ch.Close()
	require.Equal(t, ch.hashLRU.Len(), 0)
}

func mkTemp(t *testing.T) *os.File {
	file, err := os.CreateTemp(t.TempDir(), "cached_hasher_test_*")
	require.NoError(t, err)

	return file
}

func writePattern(t *testing.T, file *os.File, p pattern) {
	err := file.Truncate(0)
	require.NoError(t, err)
	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	n, err := file.Write(p.text)
	require.NoError(t, err)
	require.Equal(t, n, len(p.text))
}

func (ch *CachedHasher) checkState(t *testing.T, path string, p pattern, stats CachedHasherStats) {
	hashes, err := ch.HashFile(path)
	require.NoError(t, err)
	require.Len(t, hashes, 3)
	require.Equal(t, p.sha256, hashes["sha256"].String())
	require.Equal(t, p.sha512, hashes["sha512"].String())
	require.Equal(t, p.sha3_384, hashes["sha3_384"].String())
	require.Equal(t, stats, ch.Stats())
}
