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

	as "github.com/aerospike/aerospike-client-go/v7"
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

func pointer[T any](d T) *T {
	return &d
}

func TestParseClientPolicy(t *testing.T) {
	sampleClusterName := "TestCluster"
	sampleUser := "TestUser"
	samplePassword := "MySecretPassword"

	TLSPolicy := as.NewClientPolicy()
	tlsconfig, _ := tlscommon.LoadTLSConfig(&tlscommon.Config{Enabled: pointer(true)})
	TLSPolicy.TlsConfig = tlsconfig.ToConfig()

	ClusterNamePolicy := as.NewClientPolicy()
	ClusterNamePolicy.ClusterName = sampleClusterName

	UserPasswordClientPolicy := as.NewClientPolicy()
	UserPasswordClientPolicy.User = sampleUser
	UserPasswordClientPolicy.Password = samplePassword

	UserPasswordTLSPolicy := as.NewClientPolicy()
	UserPasswordTLSPolicy.User = sampleUser
	UserPasswordTLSPolicy.Password = samplePassword
	UserPasswordTLSPolicy.TlsConfig = tlsconfig.ToConfig()

	ExternalAuthModePolicy := as.NewClientPolicy()
	ExternalAuthModePolicy.AuthMode = as.AuthModeExternal

	PKIAuthModePolicy := as.NewClientPolicy()
	PKIAuthModePolicy.AuthMode = as.AuthModePKI

	InternalAuthModePolicy := as.NewClientPolicy()
	InternalAuthModePolicy.AuthMode = as.AuthModeInternal

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
			Name: "TLS Declaration",
			Config: Config{
				TLS: &tlscommon.Config{
					Enabled: pointer(true),
				},
			},
			expectedClientPolicy: TLSPolicy,
			expectedErr:          nil,
		},
		{
			Name: "Cluster Name Setting",
			Config: Config{
				ClusterName: sampleClusterName,
			},
			expectedClientPolicy: ClusterNamePolicy,
			expectedErr:          nil,
		},
		{
			Name: "Username and password are honored",
			Config: Config{
				User:     sampleUser,
				Password: samplePassword,
			},
			expectedClientPolicy: UserPasswordClientPolicy,
			expectedErr:          nil,
		},
		{
			Name: "Username is set and password is not set",
			Config: Config{
				User: sampleUser,
			},
			expectedClientPolicy: as.NewClientPolicy(),
			expectedErr:          fmt.Errorf("if username is set, password should be set too"),
		},
		{
			Name: "Password is set and user is not set",
			Config: Config{
				Password: samplePassword,
			},
			expectedClientPolicy: as.NewClientPolicy(),
			expectedErr:          fmt.Errorf("if password is set, username should be set too"),
		},
		{
			Name: "TLS and Basic Auth",
			Config: Config{
				TLS: &tlscommon.Config{
					Enabled: pointer(true),
				},
				User:     sampleUser,
				Password: samplePassword,
			},
			expectedClientPolicy: UserPasswordTLSPolicy,
			expectedErr:          nil,
		},
		{
			Name: "Unsupported Authentication Mode",
			Config: Config{
				AuthMode: "doesnotexist",
			},
			expectedErr: fmt.Errorf("unknown authentication mode 'doesnotexist'"),
		},
		{
			Name: "External Authentication Mode",
			Config: Config{
				AuthMode: AUTH_MODE_EXTERNAL,
			},
			expectedClientPolicy: ExternalAuthModePolicy,
			expectedErr:          fmt.Errorf("unknown authentication mode 'doesnotexist'"),
		},
		{
			Name: "Internal Authentication Mode",
			Config: Config{
				AuthMode: AUTH_MODE_INTERNAL,
			},
			expectedClientPolicy: InternalAuthModePolicy,
			expectedErr:          fmt.Errorf("unknown authentication mode 'doesnotexist'"),
		},
	}

	for _, test := range tests {
		result, err := ParseClientPolicy(test.Config)
		if err != nil {
			if test.expectedErr != nil {
				assert.Equalf(t, test.expectedErr.Error(), err.Error(),
					"Aerospike policy the error produced is not the one expected: got '%s', expected '%s'", err.Error(), test.expectedErr.Error())
				continue
			}
			t.Error(err)
			continue
		}
		assert.Equalf(t, test.expectedClientPolicy.ClusterName, result.ClusterName,
			"Aerospike policy cluster name is wrong. Got '%s' expected '%s'", result.ClusterName, test.expectedClientPolicy.ClusterName)

		assert.Equalf(t, test.expectedClientPolicy.User, result.User,
			"Aerospike policy username is wrong. Got '%s' expected '%s'", result.User, test.expectedClientPolicy.User)

		assert.Equalf(t, test.expectedClientPolicy.Password, result.Password,
			"Aerospike policy password is wrong. Got '%s' expected '%s'", result.Password, test.expectedClientPolicy.Password)

		assert.Equalf(t, test.expectedClientPolicy.AuthMode, result.AuthMode,
			"Aerospike policy authentication mode is wrong. Got '%s' expected '%s'", result.AuthMode, test.expectedClientPolicy.AuthMode)

		if test.Config.TLS.IsEnabled() {
			assert.NotNil(t, result.TlsConfig, "Aerospike policy: TLS is not set even though TLS is specified in the configuration")
		}
	}
}
