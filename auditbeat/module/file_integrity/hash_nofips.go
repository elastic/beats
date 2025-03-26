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

//go:build !requirefips

package file_integrity

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"hash"

	"github.com/cespare/xxhash/v2"
	"golang.org/x/crypto/blake2b"
)

var (
	validHashes = []HashType{
		BLAKE2B_256, BLAKE2B_384, BLAKE2B_512,
		MD5,
		SHA1,
		SHA224, SHA256, SHA384, SHA512, SHA512_224, SHA512_256,
		SHA3_224, SHA3_256, SHA3_384, SHA3_512,
		XXH64,
	}

	hashTypes = map[HashType]func() hash.Hash{
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
		MD5:    md5.New,
		SHA1:   sha1.New,
		SHA224: sha256.New224,
		SHA256: sha256.New,
		SHA384: sha512.New384,
		SHA3_224: func() hash.Hash {
			return sha3.New224()
		},
		SHA3_256: func() hash.Hash {
			return sha3.New256()
		},
		SHA3_384: func() hash.Hash {
			return sha3.New384()
		},
		SHA3_512: func() hash.Hash {
			return sha3.New512()
		},
		SHA512:     sha512.New,
		SHA512_224: sha512.New512_224,
		SHA512_256: sha512.New512_256,
		XXH64: func() hash.Hash {
			return xxhash.New()
		},
	}
)
