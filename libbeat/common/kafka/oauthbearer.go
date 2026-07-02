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

package kafka

import (
	"fmt"
	"os"
	"strings"

	"github.com/elastic/sarama"
)

// fileTokenProvider implements sarama.AccessTokenProvider for SASL/OAUTHBEARER
// authentication. It reads a JWT from a file on each Token() call so that
// credential rotations are picked up automatically without restarting the beat.
type fileTokenProvider struct {
	credentialsPath string
	extensions      map[string]string
}

func newFileTokenProvider(credentialsPath string, extensions map[string]string) (*fileTokenProvider, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("sasl.credentials_path is required for OAUTHBEARER")
	}
	return &fileTokenProvider{
		credentialsPath: credentialsPath,
		extensions:      extensions,
	}, nil
}

// Token reads the JWT from the credentials file and returns it as a sarama
// AccessToken. Re-reading the file on each call ensures the provider picks up
// rotated credentials without restarting the beat.
func (p *fileTokenProvider) Token() (*sarama.AccessToken, error) {
	tokenBytes, err := os.ReadFile(p.credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("reading OAUTHBEARER credentials file %q: %w", p.credentialsPath, err)
	}
	return &sarama.AccessToken{
		Token:      strings.TrimSpace(string(tokenBytes)),
		Extensions: p.extensions,
	}, nil
}
