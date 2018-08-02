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

package file_integrity

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/match"
)

// HashType identifies a cryptographic algorithm.
type HashType string

// Unpack unpacks a string to a HashType for config parsing.
func (t *HashType) Unpack(v string) error {
	*t = HashType(v)
	return nil
}

var validHashes = []HashType{
	BLAKE2B_256, BLAKE2B_384, BLAKE2B_512,
	MD5,
	SHA1,
	SHA224, SHA256, SHA384, SHA512, SHA512_224, SHA512_256,
	SHA3_224, SHA3_256, SHA3_384, SHA3_512,
	XXH64,
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

// Config contains the configuration parameters for the file integrity
// metricset.
type Config struct {
	Paths               []string        `config:"paths" validate:"required"`
	HashTypes           []HashType      `config:"hash_types"`
	MaxFileSize         string          `config:"max_file_size"`
	MaxFileSizeBytes    uint64          `config:",ignore"`
	ScanAtStart         bool            `config:"scan_at_start"`
	ScanRatePerSec      string          `config:"scan_rate_per_sec"`
	ScanRateBytesPerSec uint64          `config:",ignore"`
	Recursive           bool            `config:"recursive"` // Recursive enables recursive monitoring of directories.
	ExcludeFiles        []match.Matcher `config:"exclude_files"`
	IncludeFiles        []match.Matcher `config:"include_files"`
}

// Validate validates the config data and return an error explaining all the
// problems with the config. This method modifies the given config.
func (c *Config) Validate() error {
	// Resolve symlinks.
	for i, p := range c.Paths {
		if evalPath, err := filepath.EvalSymlinks(p); err == nil {
			c.Paths[i] = evalPath
		}
	}
	// Sort and deduplicate.
	sort.Strings(c.Paths)
	c.Paths = deduplicate(c.Paths)

	var errs multierror.Errors
	var err error

nextHash:
	for _, ht := range c.HashTypes {
		ht = HashType(strings.ToLower(string(ht)))
		for _, validHash := range validHashes {
			if ht == validHash {
				continue nextHash
			}
		}
		errs = append(errs, errors.Errorf("invalid hash_types value '%v'", ht))
	}

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

// deduplicate deduplicates the given sorted string slice. The returned slice
// reuses the same backing array as in (so don't use in after calling this).
func deduplicate(in []string) []string {
	var lastValue string
	out := in[:0]
	for _, value := range in {
		if value == lastValue {
			continue
		}
		out = append(out, value)
		lastValue = value
	}
	return out
}

// IsExcludedPath checks if a path matches the exclude_files regular expressions.
func (c *Config) IsExcludedPath(path string) bool {
	for _, matcher := range c.ExcludeFiles {
		if matcher.MatchString(path) {
			return true
		}
	}
	return false
}

// IsIncludedPath checks if a path matches the include_files regular expressions.
func (c *Config) IsIncludedPath(path string) bool {
	if len(c.IncludeFiles) == 0 {
		return true
	}

	for _, matcher := range c.IncludeFiles {
		if matcher.MatchString(path) {
			return true
		}
	}
	return false
}

var defaultConfig = Config{
	HashTypes:        []HashType{SHA1},
	MaxFileSize:      "100 MiB",
	MaxFileSizeBytes: 100 * 1024 * 1024,
	ScanAtStart:      true,
	ScanRatePerSec:   "50 MiB",
}
