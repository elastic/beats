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

package diskqueue

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// KeySize is 128-bit
	KeySize = 16
)

// EncryptionReader allows reading from a AES-128-CTR stream
type EncryptionReader struct {
	src        io.ReadCloser
	stream     cipher.Stream
	block      cipher.Block
	iv         []byte
	ciphertext []byte
}

// NewEncryptionReader returns a new AES-128-CTR decrypter
func NewEncryptionReader(r io.ReadCloser, key []byte) (*EncryptionReader, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("key must be %d bytes long", KeySize)
	}

	er := &EncryptionReader{}
	er.src = r

	// turn key into block & save
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	er.block = block

	// read IV from the io.ReadCloser
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(er.src, iv); err != nil {
		return nil, err
	}
	er.iv = iv

	// create Stream
	er.stream = cipher.NewCTR(block, iv)

	return er, nil
}

func (er *EncryptionReader) Read(buf []byte) (int, error) {
	if cap(er.ciphertext) >= len(buf) {
		er.ciphertext = er.ciphertext[:len(buf)]
	} else {
		er.ciphertext = make([]byte, len(buf))
	}
	n, err := er.src.Read(er.ciphertext)
	if err != nil {
		return n, err
	}
	er.stream.XORKeyStream(buf, er.ciphertext)
	return n, nil
}

func (er *EncryptionReader) Close() error {
	return er.src.Close()
}

// Reset Sets up stream again, assumes that caller has already set the
// src to the iv
func (er *EncryptionReader) Reset() error {
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(er.src, iv); err != nil {
		return err
	}
	if !bytes.Equal(iv, er.iv) {
		return fmt.Errorf("different iv, something is wrong")
	}

	// create Stream
	er.stream = cipher.NewCTR(er.block, iv)
	return nil
}

// EncryptionWriter allows writing to a AES-128-CTR stream
type EncryptionWriter struct {
	dst        WriteCloseSyncer
	stream     cipher.Stream
	ciphertext []byte
}

// NewEncryptionWriter returns a new AES-128-CTR stream encryptor
func NewEncryptionWriter(w WriteCloseSyncer, key []byte) (*EncryptionWriter, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("key must be %d bytes long", KeySize)
	}

	ew := &EncryptionWriter{}

	// turn key into block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// create random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// create stream
	stream := cipher.NewCTR(block, iv)

	//write IV
	n, err := w.Write(iv)
	if err != nil {
		return nil, err
	}
	if n != len(iv) {
		return nil, io.ErrShortWrite
	}

	ew.dst = w
	ew.stream = stream
	return ew, nil
}

func (ew *EncryptionWriter) Write(buf []byte) (int, error) {
	if cap(ew.ciphertext) >= len(buf) {
		ew.ciphertext = ew.ciphertext[:len(buf)]
	} else {
		ew.ciphertext = make([]byte, len(buf))
	}
	ew.stream.XORKeyStream(ew.ciphertext, buf)
	return ew.dst.Write(ew.ciphertext)
}

func (ew *EncryptionWriter) Close() error {
	return ew.dst.Close()
}

func (ew *EncryptionWriter) Sync() error {
	return ew.dst.Sync()
}
