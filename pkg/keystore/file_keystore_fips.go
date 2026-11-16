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

//go:build requirefips

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
)

// Version of the keystore format, will be added at the beginning of the file.
var version = []byte("v2")

// Encrypt the data payload using a derived keys and the AES-256-GCM algorithm.
func (k *FileKeystore) encrypt(reader io.Reader) (io.Reader, error) {
	salt, err := randomBytes(saltLength)
	if err != nil {
		return nil, err
	}

	// Stretch the user provided key
	password, _ := k.password.Get()
	passwordBytes, err := k.hashPassword(string(password), salt)
	if err != nil {
		return nil, fmt.Errorf("could not hash password, error: %w", err)
	}

	// Select AES-256: because len(passwordBytes) == 32 bytes
	block, err := aes.NewCipher(passwordBytes)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to encrypt, error: %w", err)
	}

	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to encrypt, error: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read unencrypted data, error: %w", err)
	}

	encodedBytes := aesgcm.Seal(nil, nil, data, nil)

	// Generate the payload with all the additional information required to decrypt the
	// output format of the document: VERSION|SALT|PAYLOAD
	buf := bytes.NewBuffer(salt)
	buf.Write(encodedBytes)

	return buf, nil
}

// should receive an io.reader...
func (k *FileKeystore) decrypt(reader io.Reader) (io.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read all the data from the encrypted file, error: %w", err)
	}

	if len(data) < saltLength+1 {
		return nil, fmt.Errorf("missing information in the file for decrypting the keystore")
	}

	// extract the necessary information to decrypt the data from the data payload
	salt := data[0:saltLength]
	encodedBytes := data[saltLength:]

	password, _ := k.password.Get()
	passwordBytes, err := k.hashPassword(string(password), salt)
	if err != nil {
		return nil, fmt.Errorf("could not hash password, error: %w", err)
	}

	block, err := aes.NewCipher(passwordBytes)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to decrypt the data: %w", err)
	}

	aesgcm, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("could not create the keystore cipher to decrypt the data: %w", err)
	}

	decodedBytes, err := aesgcm.Open(nil, nil, encodedBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt keystore data: %w", err)
	}

	return bytes.NewReader(decodedBytes), nil
}
