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

package x509util

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var certPem = `-----BEGIN CERTIFICATE-----
MIIFTDCCAzSgAwIBAgIRAOAMlgVxz4G+Zj/EtBTvpg4wDQYJKoZIhvcNAQENBQAw
LzELMAkGA1UEBhMCVVMxEDAOBgNVBAoTB2VsYXN0aWMxDjAMBgNVBAsTBWJlYXRz
MB4XDTE3MDUxODIwMzI1MVoXDTI3MDUxODIwMzI1MVowLzELMAkGA1UEBhMCVVMx
EDAOBgNVBAoTB2VsYXN0aWMxDjAMBgNVBAsTBWJlYXRzMIICIjANBgkqhkiG9w0B
AQEFAAOCAg8AMIICCgKCAgEAv8IiJDAIDl+roQOWe+oSq46Nyuu9R+Iis0V1i6M7
zA6QijbxCSZ64cCFYQfKheRYQSZRstHPHSUM1gSvUih/sqZqsiNMYDbb9j7geMDv
ls4c7rsHx7xImD7nCrEVWkiapGIhkW6SOtVo18Zmw89FUuDFhoRmMHcQ+7AtM4uU
NPkSqKcXvzG093SU0oNdIBdw5PzoQlvBh5DL0iRYC6y22cwJyjWTUEB5vTjOTDxi
FzsovRtjpdjzSZACXyW68b99icLzmxzLvsZ7w8tFJ8uOPQAVxwg6SmMUorURv48s
BjfVfN487OjH3d+51ozNJjP1MmKoN2BoE8pWq0jdhOWhDQH+pRiRjfMuL+yvcIJ2
pxdOv0F3KBkng7qEgEUA8cqaFnawDA7O3a20SeDFWSQtN6LsFjT7EDMzNkML1pJj
bGK24QFCIOOvCJtaccuREN1OfbN1yhTz3VErbJttwO6j2KueasPHXU3qLu2FKOls
XbPy1XMuLYZgv8Zprcbs4KhQ3/A7/RO1cakxWlRwta63mUIM2xLIMIgRSR+DSZ5d
JaDNO6i49eIGQXRxDb9dxA2hoCcoTv7PJKyOpNb5vyxMXJGY7H5j1jEEcqEeuI5u
vuUwugQGtsl1eFLXIeQLerOHEQoS6wMv0fHBtZOVCHu8CCrnt/ag7kn39nkwNofL
ovECAwEAAaNjMGEwDgYDVR0PAQH/BAQDAgKkMB0GA1UdJQQWMBQGCCsGAQUFBwMC
BggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MA4GA1UdDgQHBAUxMjM0NTAPBgNV
HREECDAGhwR/AAABMA0GCSqGSIb3DQEBDQUAA4ICAQBjeGIfFqXuwHiClMytJNZL
cRyjeZ6PJIAQtqh8Vi+XD2JiDTkwJ/g4R0FbgqE/icGkm/hsJ6BEwp8ep5eXevjS
Hb8tVbM5Uc31yyIKcJMgnfS8O0eIXi5PxgFWPcUXxrsjwHyQREqj96HImmzOm99O
MJhifWT3YP8OEMyl1KpioPaXafhc4ATEiRVZizHM9z+phyINBNghH3OaN91ZnsKJ
El7mvOLjRi7fuSxBWJntKVAZAwXK+nH+z/Ay4AZFA9HgFHo3PGpKUaLOYCIsGxAq
GP4V/WsOtEJ9rP5TR92pOvcj49T47FmwSYaRtoXHDVuoun0fdwT4DxWJdksqdWzG
ieRls2IrZIvR2FT/A/XdQG3kZ79WA/K3OAGDgxv0PCpw6ssAMvgjR03TjEXpwMmN
SNcrx1H6l8DHFHJN9f7SofO/J0hkA+fRZUFxP5R+P2BPU0hV14H9iSie/bxhSWIW
ieAh0K1SNRbffXeYUvAgrjEvG5x40TktnvjHb20lxc1F1gqB+855kfZdiJeUeizi
syq6OnCEp+RSBdK7J3scm7t6Nt3GRndJMO9hNDprogTqHxQbZ0jficntGd7Lbp+C
CBegkhOzD6cp2rGlyYI+MmvdXFaHbsUJj2tfjHQdo2YjQ1s8r2pw219LTzPvO/Dz
morZ618ezCBBqxHsDF6DCA==
-----END CERTIFICATE-----
`

func TestCertToPEMString(t *testing.T) {
	block, _ := pem.Decode([]byte(certPem))
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	assert.Equal(t, certPem, CertToPEMString(cert))
}
