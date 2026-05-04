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

package tlscommon

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/youmark/pkcs8"
)

func decryptPKCS1Key(block pem.Block, passphrase []byte) (pem.Block, error) {
	if len(passphrase) == 0 {
		return block, errors.New("no passphrase available")
	}
	decrypted, err := x509.DecryptPEMBlock(&block, passphrase) //nolint:staticcheck // intentional legacy support path
	if err != nil {
		return block, fmt.Errorf("failed to decrypt PKCS#1 key: %w", err)
	}
	block.Bytes = decrypted
	// x509.IsEncryptedPEMBlock checks only for DEK-Info, so removing just
	// that header prevents re-detection. However, x509.EncryptPEMBlock also
	// sets Proc-Type, which would falsely appear as "4,ENCRYPTED" in the
	// re-encoded PEM output if left in place. Nil the whole map to remove both.
	block.Headers = nil
	return block, nil
}

func decryptPKCS8Key(block pem.Block, passphrase []byte) (pem.Block, error) {
	if len(passphrase) == 0 {
		return block, errors.New("no passphrase available")
	}

	key, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, passphrase)
	if err != nil {
		return block, fmt.Errorf("failed to parse key: %w", err)
	}

	switch key.(type) {
	case *rsa.PrivateKey:
		block.Type = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		block.Type = "ECDSA PRIVATE KEY"
	default:
		return block, fmt.Errorf("unknown key type %T", key)
	}

	buffer, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return block, fmt.Errorf("failed to marshal decrypted private key: %w", err)
	}
	block.Bytes = buffer

	return block, nil
}
