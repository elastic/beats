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

package keystore

// SecureString Initial implementation for a SecureString representation in
// beats, currently we keep the password into a Bytes array, we need to implement a way
// to safely clean that array.
//
// Investigate memguard: https://github.com/awnumar/memguard
type SecureString struct {
	value []byte
}

// NewSecureString return a struct representing a secrets string.
func NewSecureString(value []byte) *SecureString {
	return &SecureString{
		value: value,
	}
}

// Get returns the byte value of the secret, or an error if we cannot return it.
func (s *SecureString) Get() ([]byte, error) {
	return s.value, nil
}

// String custom string implementation to make sure we don't bleed this struct into a string.
func (s SecureString) String() string {
	return "<SecureString>"
}

// GoString implements the GoStringer interface to hide the secret value.
func (s SecureString) GoString() string {
	return s.String()
}
