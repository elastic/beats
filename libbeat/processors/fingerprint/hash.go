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

package fingerprint

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"strings"

	"github.com/cespare/xxhash/v2"
)

type namedHashMethod struct {
	Name string
	Hash hashMethod
}
type hashMethod func() hash.Hash

var hashes = map[string]namedHashMethod{}

func init() {
	for _, h := range []namedHashMethod{
		{Name: "md5", Hash: md5.New},
		{Name: "sha1", Hash: sha1.New},
		{Name: "sha256", Hash: sha256.New},
		{Name: "sha384", Hash: sha512.New384},
		{Name: "sha512", Hash: sha512.New},
		{Name: "xxhash", Hash: newXxHash},
	} {
		hashes[h.Name] = h
	}
}

// Unpack creates the hashMethod from the given string
func (f *namedHashMethod) Unpack(str string) error {
	str = strings.ToLower(str)

	m, found := hashes[str]
	if !found {
		return makeErrUnknownMethod(str)
	}

	*f = m
	return nil
}

// newXxHash returns a hash.Hash instead of the *Digest which implements the same
func newXxHash() hash.Hash {
	return xxhash.New()
}
