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

package file

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"

	"github.com/elastic/beats/libbeat/common/file"
)

// HashType identifies a cryptographic algorithm.
type HashType string

// Unpack unpacks a string to a HashType for config parsing.
func (t *HashType) Unpack(v string) error {
	*t = HashType(v)
	return nil
}

var validHashes = map[HashType]struct{}{
	BLAKE2B_256: {},
	BLAKE2B_384: {},
	BLAKE2B_512: {},
	MD5:         {},
	SHA1:        {},
	SHA224:      {},
	SHA256:      {},
	SHA384:      {},
	SHA512:      {},
	SHA512_224:  {},
	SHA512_256:  {},
	SHA3_224:    {},
	SHA3_256:    {},
	SHA3_384:    {},
	SHA3_512:    {},
	XXH64:       {},
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

type FileHasher struct {
	hashTypes []HashType
}

func NewFileHasher(hashTypes []HashType) (*FileHasher, error) {
	hasher := FileHasher{}

	// Check hash types are valid
	for _, hashType := range hashTypes {
		ht := HashType(strings.ToLower(string(hashType)))
		if _, valid := validHashes[ht]; !valid {
			return nil, errors.Errorf("invalid hash type '%v'", ht)
		}

		hasher.hashTypes = append(hasher.hashTypes, ht)
	}

	return &hasher, nil
}

func (hasher *FileHasher) HashFile(name string) (map[HashType]Digest, error) {
	var hashes []hash.Hash
	for _, name := range hasher.hashTypes {
		switch name {
		case BLAKE2B_256:
			h, _ := blake2b.New256(nil)
			hashes = append(hashes, h)
		case BLAKE2B_384:
			h, _ := blake2b.New384(nil)
			hashes = append(hashes, h)
		case BLAKE2B_512:
			h, _ := blake2b.New512(nil)
			hashes = append(hashes, h)
		case MD5:
			hashes = append(hashes, md5.New())
		case SHA1:
			hashes = append(hashes, sha1.New())
		case SHA224:
			hashes = append(hashes, sha256.New224())
		case SHA256:
			hashes = append(hashes, sha256.New())
		case SHA384:
			hashes = append(hashes, sha512.New384())
		case SHA3_224:
			hashes = append(hashes, sha3.New224())
		case SHA3_256:
			hashes = append(hashes, sha3.New256())
		case SHA3_384:
			hashes = append(hashes, sha3.New384())
		case SHA3_512:
			hashes = append(hashes, sha3.New512())
		case SHA512:
			hashes = append(hashes, sha512.New())
		case SHA512_224:
			hashes = append(hashes, sha512.New512_224())
		case SHA512_256:
			hashes = append(hashes, sha512.New512_256())
		case XXH64:
			hashes = append(hashes, xxhash.New64())
		default:
			return nil, errors.Errorf("unknown hash type '%v'", name)
		}
	}

	f, err := file.ReadOpen(name)
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
		nameToHash[hasher.hashTypes[i]] = h.Sum(nil)
	}

	return nameToHash, nil
}

func multiWriter(hash []hash.Hash) io.Writer {
	writers := make([]io.Writer, 0, len(hash))
	for _, h := range hash {
		writers = append(writers, h)
	}
	return io.MultiWriter(writers...)
}
