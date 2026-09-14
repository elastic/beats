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

// https://github.com/Shopify/sarama/blob/master/examples/sasl_scram_client/scram_client.go
package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/xdg-go/scram"

	"github.com/elastic/sarama"
)

// SHA256 and SHA512 are the hash generators used with SCRAM-SHA-256 and SCRAM-SHA-512 respectively.
// xdg-go/scram v1.2.0 uses stdlib crypto/pbkdf2 on go1.24+ (pbkdf2_go124.go), which is inside
// the certified Go Cryptographic Module boundary (GOFIPS140=v1.0.0, CMVP #5247). The legacy
// xdg-go/pbkdf2 path (pbkdf2_legacy.go, //go:build !go1.24) is never compiled because go.mod
// declares a minimum Go version of 1.26, which is a hard floor since Go 1.21.
var SHA256 scram.HashGeneratorFcn = func() hash.Hash { return sha256.New() }
var SHA512 scram.HashGeneratorFcn = func() hash.Hash { return sha512.New() }

type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.NewConversation()
	return nil
}

func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return response, err
}

func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

func scramClient(mechanism string) func() sarama.SCRAMClient {
	if mechanism == saslTypeSCRAMSHA512 {
		return func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
	}
	return func() sarama.SCRAMClient {
		return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
	}
}
