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
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cespare/xxhash"
	"github.com/dustin/go-humanize"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"golang.org/x/time/rate"

	"github.com/elastic/beats/libbeat/common/file"
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

var validHashes = map[HashType](func() hash.Hash){
	BLAKE2B_256: func() hash.Hash {
		h, _ := blake2b.New256(nil)
		return h
	},
	BLAKE2B_384: func() hash.Hash {
		h, _ := blake2b.New384(nil)
		return h
	},
	BLAKE2B_512: func() hash.Hash {
		h, _ := blake2b.New512(nil)
		return h
	},
	MD5:        md5.New,
	SHA1:       sha1.New,
	SHA224:     sha256.New224,
	SHA256:     sha256.New,
	SHA384:     sha512.New384,
	SHA512:     sha512.New,
	SHA512_224: sha512.New512_224,
	SHA512_256: sha512.New512_256,
	SHA3_224:   sha3.New224,
	SHA3_256:   sha3.New256,
	SHA3_384:   sha3.New384,
	SHA3_512:   sha3.New512,
	XXH64: func() hash.Hash {
		return xxhash.New()
	},
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
	return fmt.Sprintf("hasher: file size %d exceeds max file size", e.fileSize)
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
	var errs multierror.Errors

	for _, ht := range c.HashTypes {
		if !ht.IsValid() {
			errs = append(errs, errors.Errorf("invalid hash_types value '%v'", ht))
		}
	}

	var err error

	c.MaxFileSizeBytes, err = humanize.ParseBytes(c.MaxFileSize)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid max_file_size value"))
	} else if c.MaxFileSizeBytes <= 0 {
		errs = append(errs, errors.Errorf("max_file_size value (%v) must be positive", c.MaxFileSize))
	}

	c.ScanRateBytesPerSec, err = humanize.ParseBytes(c.ScanRatePerSec)
	if err != nil {
		errs = append(errs, errors.Wrap(err, "invalid scan_rate_per_sec value"))
	}

	return errs.Err()
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
	return &FileHasher{
		config: c,
		limiter: rate.NewLimiter(
			rate.Limit(c.ScanRateBytesPerSec), // Rate
			int(c.MaxFileSizeBytes),           // Burst
		),
		done: done,
	}, nil
}

// HashFile hashes the contents of a file.
func (hasher *FileHasher) HashFile(path string) (map[HashType]Digest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat file %v", path)
	}

	// Throttle reading and hashing rate.
	if len(hasher.config.HashTypes) > 0 {
		err = hasher.throttle(info.Size())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to hash file %v", path)
		}
	}

	var hashes []hash.Hash
	for _, hashType := range hasher.config.HashTypes {
		h, valid := validHashes[hashType]
		if !valid {
			return nil, errors.Errorf("unknown hash type '%v'", hashType)
		}

		hashes = append(hashes, h())
	}

	if len(hashes) > 0 {
		f, err := file.ReadOpen(path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open file for hashing")
		}
		defer f.Close()

		hashWriter := multiWriter(hashes)
		if _, err := io.Copy(hashWriter, f); err != nil {
			return nil, errors.Wrap(err, "failed to calculate file hashes")
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
