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

//go:build integration

package mysql

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	_ "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestNewDB(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")

	db, err := NewDB(GetMySQLEnvDSN(service.Host()), nil)
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
}

func loadTLSConfig(caCertPath, clientCertPath, clientKeyPath string) (*tls.Config, error) {
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}

	return tlsConfig, nil
}

func TestNewDBWithSSL(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")

	tlsConfig, err := loadTLSConfig("_meta/certs/root-ca.pem", "_meta/certs/client-cert.pem", "_meta/certs/client-key.pem")
	tlsConfig.InsecureSkipVerify = true
	assert.NoError(t, err)

	db, err := NewDB(GetMySQLEnvDSN(service.Host())+"?tls=custom", tlsConfig)
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)

	// Check if the current connection is using SSL
	var sslCipher, variableName, value string
	err = db.QueryRow(`show status like 'Ssl_cipher'`).Scan(&variableName, &sslCipher)
	assert.NoError(t, err)

	// If sslCipher is not empty, then SSL is being used for the connection
	assert.NotEmpty(t, variableName)
	assert.NotEmpty(t, sslCipher)

	err = db.QueryRow(`show variables like 'have_ssl'`).Scan(&variableName, &value)
	assert.NoError(t, err)
	assert.Equal(t, "YES", value)
}
