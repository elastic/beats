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

package aerospike

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	as "github.com/aerospike/aerospike-client-go"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		Name         string
		Host         string
		expectedHost *as.Host
		expectedErr  error
	}{
		{
			Name:         "with hostname and port",
			Host:         "localhost:3000",
			expectedHost: as.NewHost("localhost", 3000),
		},
		{
			Name:        "without port",
			Host:        "localhost",
			expectedErr: errors.New("Can't parse host localhost"),
		},
		{
			Name:        "with wrong port",
			Host:        "localhost:wrong",
			expectedErr: errors.New("Can't parse port: strconv.Atoi: parsing \"wrong\": invalid syntax"),
		},
	}

	for _, test := range tests {
		result, err := ParseHost(test.Host)
		if err != nil {
			if test.expectedErr != nil {
				assert.Equal(t, test.expectedErr.Error(), err.Error())
				continue
			}
			t.Error(err)
			continue
		}

		assert.Equal(t, test.expectedHost.String(), result.String(), test.Name)
	}
}

func TestParseInfo(t *testing.T) {
	tests := []struct {
		Name     string
		info     string
		expected map[string]interface{}
	}{
		{
			Name: "with kv",
			info: "key1=value1;key2=value2",
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			Name:     "without kv",
			info:     "wrong result",
			expected: map[string]interface{}{},
		},
		{
			Name:     "mixed",
			info:     "wrong result;key=value",
			expected: map[string]interface{}{"key": "value"},
		},
	}

	for _, test := range tests {
		result := ParseInfo(test.info)
		assert.Equal(t, test.expected, result, test.Name)
	}
}

func getStringPointer(value string) *string {
	return &value
}

func getBoolPointer(value bool) *bool {
	return &value
}

func TestParseClientPolicy(t *testing.T) {
	sampleUser := "Test"
	samplePassword := "MySecretPassword"
	sampleClusterName := "TestCluster"

	UserPasswordClientPolicy := as.NewClientPolicy()
	UserPasswordClientPolicy.User = sampleUser
	UserPasswordClientPolicy.Password = samplePassword

	TLSPolicy := as.NewClientPolicy()
	tlsconfig, _ := tlscommon.LoadTLSConfig(&tlscommon.Config{Enabled: getBoolPointer(true)})
	TLSPolicy.TlsConfig = tlsconfig.ToConfig()

	ClusterNamePolicy := as.NewClientPolicy()
	ClusterNamePolicy.ClusterName = sampleClusterName

	tests := []struct {
		Name                 string
		Config               Config
		expectedClientPolicy *as.ClientPolicy
		expectedErr          error
	}{
		{
			Name:                 "Empty configuration leads to default policy",
			Config:               Config{},
			expectedClientPolicy: as.NewClientPolicy(),
			expectedErr:          nil,
		},
		{
			Name: "Username and password are honored",
			Config: Config{
				User:     getStringPointer(sampleUser),
				Password: getStringPointer(samplePassword),
			},
			expectedClientPolicy: UserPasswordClientPolicy,
			expectedErr:          nil,
		},
		{
			Name: "Username is set but Password is not",
			Config: Config{
				User: getStringPointer(sampleUser),
			},
			expectedClientPolicy: UserPasswordClientPolicy,
			expectedErr:          fmt.Errorf("if username is set, password should be set too"),
		},
		{
			Name: "Password is set but Username is not",
			Config: Config{
				Password: getStringPointer(samplePassword),
			},
			expectedClientPolicy: UserPasswordClientPolicy,
			expectedErr:          fmt.Errorf("if password is set, username should be set too"),
		},
		{
			Name: "TLS Declaration",
			Config: Config{
				TLS: &tlscommon.Config{
					Enabled: getBoolPointer(true),
				},
			},
			expectedClientPolicy: TLSPolicy,
			expectedErr:          nil,
		},
		{
			Name: "Cluster Name Setting",
			Config: Config{
				ClusterName: getStringPointer(sampleClusterName),
			},
			expectedClientPolicy: ClusterNamePolicy,
			expectedErr:          nil,
		},
	}

	for _, test := range tests {
		result, err := ParseClientPolicy(test.Config)
		if err != nil {
			if test.expectedErr != nil {
				assert.Equalf(t, test.expectedErr.Error(), err.Error(),
					"Aerospike policy the error produced is not the one expected. Got '%s' expected '%s'", err.Error(), test.expectedErr.Error())
				continue
			}
			t.Error(err)
			continue
		}
		assert.Equalf(t, test.expectedClientPolicy.User, result.User,
			"Aerospike policy username is wrong. Got '%s' expected '%s'", result.User, test.expectedClientPolicy.User)
		assert.Equalf(t, test.expectedClientPolicy.Password, result.Password,
			"Aerospike policy password is wrong. Got '%s' expected '%s'", result.Password, test.expectedClientPolicy.Password)
		assert.Equalf(t, test.expectedClientPolicy.ClusterName, result.ClusterName,
			"Aerospike policy cluster name is wrong. Got '%s' expected '%s'", result.ClusterName, test.expectedClientPolicy.ClusterName)
		if test.Config.TLS.IsEnabled() {
			assert.NotNil(t, result.TlsConfig, "Aerospike policy: TLS is not set even though TLS is specified in the configuration")
		}
	}
}
