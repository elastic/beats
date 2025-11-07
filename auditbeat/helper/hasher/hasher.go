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

package hasher

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/time/rate"

	"github.com/elastic/beats/v7/libbeat/common/file"
)

// HashType identifies a cryptographic algorithm.
type HashType string

// Unpack unpacks a string to a HashType for config parsing.
func (t *HashType) Unpack(v string) error {
	*t = HashType(strings.ToLower(v))
	return nil
}

// IsValid checks if the hash type is valid.
func (t *HashType) IsValid() bool {
	_, valid := validHashes[*t]
	return valid
}

// Enum of hash types.
const (
	BLAKE2B_256 HashType = "blake2b_256"
	BLAKE2B_384 HashType = "blake2b_384"
	BLAKE2B_512 HashType = "blake2b_512"
	MD5         HashType = "md5"
	SHA1        HashType = "sha1"
	SHA224      HashType = "sha224"
	SHA256      HashType = "sha256"
	SHA384      HashType = "sha384"
	SHA3_224    HashType = "sha3_224"
	SHA3_256    HashType = "sha3_256"
	SHA3_384    HashType = "sha3_384"
	SHA3_512    HashType = "sha3_512"
	SHA512      HashType = "sha512"
	SHA512_224  HashType = "sha512_224"
	SHA512_256  HashType = "sha512_256"
	XXH64       HashType = "xxh64"
)

// Digest is a output of a hash function.
type Digest []byte

// String returns the digest value in lower-case hexadecimal form.
func (d Digest) String() string {
	return hex.EncodeToString(d)
}

// MarshalText encodes the digest to a hexadecimal representation of itself.
func (d Digest) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// FileTooLargeError is the error that occurs when a file that
// exceeds the max file size is attempting to be hashed.
type FileTooLargeError struct {
	fileSize int64
}

// Error returns the error message for FileTooLargeError.
func (e FileTooLargeError) Error() string {
	return fmt.Sprintf("size %d exceeds max file size", e.fileSize)
}

// Config contains the configuration of a FileHasher.
type Config struct {
	HashTypes           []HashType `config:"hash_types,replace"`
	MaxFileSize         string     `config:"max_file_size"`
	MaxFileSizeBytes    uint64     `config:",ignore"`
	ScanRatePerSec      string     `config:"scan_rate_per_sec"`
	ScanRateBytesPerSec uint64     `config:",ignore"`
}

// Validate validates the config.
func (c *Config) Validate() error {
	var errs []error

	for _, ht := range c.HashTypes {
		if !ht.IsValid() {
			errs = append(errs, fmt.Errorf("invalid hash_types value '%v'", ht))
		}
	}

	var err error

	c.MaxFileSizeBytes, err = humanize.ParseBytes(c.MaxFileSize)
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid max_file_size value: %w", err))
	} else if c.MaxFileSizeBytes <= 0 {
		errs = append(errs, fmt.Errorf("max_file_size value (%v) must be positive", c.MaxFileSize))
	}

	c.ScanRateBytesPerSec, err = humanize.ParseBytes(c.ScanRatePerSec)
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid scan_rate_per_sec value: %w", err))
	}

	return errors.Join(errs...)
}

// FileHasher hashes the contents of files.
type FileHasher struct {
	config  Config
	limiter *rate.Limiter

	// To cancel hashing
	done <-chan struct{}
}

// NewFileHasher creates a new FileHasher.
func NewFileHasher(c Config, done <-chan struct{}) (*FileHasher, error) {
	var limit rate.Limit

	if c.ScanRateBytesPerSec == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Limit(c.ScanRateBytesPerSec)
	}

	return &FileHasher{
		config: c,
		limiter: rate.NewLimiter(
			limit,                   // Rate
			int(c.MaxFileSizeBytes), // Burst
		),
		done: done,
	}, nil
}

// HashFile hashes the contents of a file.
func (hasher *FileHasher) HashFile(path string) (map[HashType]Digest, error) {
	f, err := file.ReadOpen(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")

	}

	// Throttle reading and hashing rate.
	if len(hasher.config.HashTypes) > 0 {
		err = hasher.throttle(info.Size())
		if err != nil {
			return nil, err
		}
	}

	var hashes []hash.Hash //nolint:prealloc // Preallocating doesn't bring improvements.
	for _, hashType := range hasher.config.HashTypes {
		h, valid := validHashes[hashType]
		if !valid {
			return nil, fmt.Errorf("unknown hash type '%v'", hashType)
		}

		hashes = append(hashes, h())
	}

	if len(hashes) > 0 {
		hashWriter := multiWriter(hashes)
		if _, err := io.Copy(hashWriter, f); err != nil {
			return nil, err
		}

		nameToHash := make(map[HashType]Digest, len(hashes))
		for i, h := range hashes {
			nameToHash[hasher.config.HashTypes[i]] = h.Sum(nil)
		}

		return nameToHash, nil
	}

	return nil, nil
}

func (hasher *FileHasher) throttle(fileSize int64) error {
	// Burst is ignored if limit is infinite, so check it manually
	if hasher.limiter.Limit() == rate.Inf && int(fileSize) > hasher.limiter.Burst() {
		return FileTooLargeError{fileSize}
	}
	reservation := hasher.limiter.ReserveN(time.Now(), int(fileSize))
	if !reservation.OK() {
		// File is bigger than the max file size
		return FileTooLargeError{fileSize}
	}

	delay := reservation.Delay()
	if delay == 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-hasher.done:
	case <-timer.C:
	}

	return nil
}

func multiWriter(hash []hash.Hash) io.Writer {
	writers := make([]io.Writer, 0, len(hash))
	for _, h := range hash {
		writers = append(writers, h)
	}
	return io.MultiWriter(writers...)
}
