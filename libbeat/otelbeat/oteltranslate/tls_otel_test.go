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

package oteltranslate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestTLSCommonToOTel(t *testing.T) {

	t.Run("when ssl.enabled = false", func(t *testing.T) {
		b := false
		input := &tlscommon.Config{
			Enabled: &b,
		}
		got, err := TLSCommonToOTel(input)
		require.NoError(t, err)
		want := map[string]any{
			"insecure": true,
		}

		assert.Equal(t, want, got)
	})

	tests := []struct {
		name  string
		input *tlscommon.Config
		want  map[string]any
		err   bool
	}{
		{
			name: "when unsupported configuration is passed",
			input: &tlscommon.Config{
				CATrustedFingerprint: "a3:5f:bf:93:12:8f:bc:5c:ab:14:6d:bf:e4:2a:7f:98:9d:2f:16:92:76:c4:12:ab:67:89:fc:56:4b:8e:0c:43",
			},
			want: nil,
			err:  true,
		},
		{
			name: "when ca, cert, key and key_passphrase is provided",
			input: &tlscommon.Config{
				CAs: []string{
					"testdata/certs/rootCA.crt",
					`-----BEGIN CERTIFICATE-----
MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
/D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
sxSmbIUfc2SGJGCJD4I=
-----END CERTIFICATE-----`},
				Certificate: tlscommon.CertificateConfig{
					Certificate: "testdata/certs/client.crt",
					Key:         "testdata/certs/client.key",
					Passphrase:  "changeme",
				},
			},
			want: map[string]any{
				"ca_pem": `-----BEGIN CERTIFICATE-----
MIIDzTCCArWgAwIBAgIURzQp9/bWT37HJX71yfGEO0xrs2wwDQYJKoZIhvcNAQEL
BQAwdjELMAkGA1UEBhMCVVMxDzANBgNVBAgMBkFsYXNrYTETMBEGA1UEBwwKR2xl
bm5hbGxlbjEQMA4GA1UECgwHRWxhc3RpYzEQMA4GA1UEAwwHRWxhc3RpYzEdMBsG
CSqGSIb3DQEJARYOZm9vQGVsYXN0aWMuY28wHhcNMjQwOTI1MTg1NDQ0WhcNMjkw
OTI0MTg1NDQ0WjB2MQswCQYDVQQGEwJVUzEPMA0GA1UECAwGQWxhc2thMRMwEQYD
VQQHDApHbGVubmFsbGVuMRAwDgYDVQQKDAdFbGFzdGljMRAwDgYDVQQDDAdFbGFz
dGljMR0wGwYJKoZIhvcNAQkBFg5mb29AZWxhc3RpYy5jbzCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAM4FpucFZ9WGZ1uysSYEclFII49BNw6kTAc8H79S
rZBbKCxOz2SoDof0ZAGzcactMFOHT2EuimWoO3crQMfP1RDbWNMt03LauK/zqcUv
8dwPhflbHIMS67Vse5b2wpnKclcfh5bIgCl2QADcdJcGTaQ+7a3akokRPe0GFdo1
r/JgdPoZ026A7Icve+PB7hbOn4NA9IWOp3GZGfPMC3t6MOZsFPDn7HU2YiKVK6rz
h+Z1yhpBzqP/C0/mXg68GZNzlEOuBfEvAVFuXIzAZ8ePZz+NJOmpZEdrUVTTiYWX
L25RHOqLcf1lD1MAN+WCOW27gPSAxSGLjf1r/75kOpF9RxECAwEAAaNTMFEwHQYD
VR0OBBYEFPF9yKBFHPCdOlQYjuCQl55grgxQMB8GA1UdIwQYMBaAFPF9yKBFHPCd
OlQYjuCQl55grgxQMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEB
AFAUX7H/NCNh9EyxQTneNQFOnxFBiwtaluneu0QLMhnIrRXN3RVowJ3+GI+nkRAl
48y8UeBm1urKGCpAygEblzzin5FB8TdtP7Cl4sQ98HVUbAcBJt93HR39SWTJzQIq
qkVEFLrYe9NfYEoFr9LxY2bkb1Ga7BB0guRm/Tyw4NdDbNk+nzMYbYUSUE124IFh
KPQA/X2ZeRIrZJwFhTMdMy+JkkkZGThB4lXLWaLAX8CIcERtGy2CwugAFdT7PHGr
2DQY+NmOje59N9BPezJDY9OvbuZz8NPRRzB3Iv3jDCDD/eCVgrGtG6Y9tTdYztFQ
CkBAlZultTt5DPEE+4400mU=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
/D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
sxSmbIUfc2SGJGCJD4I=
-----END CERTIFICATE-----
`,
				"cert_pem": `-----BEGIN CERTIFICATE-----
MIID2jCCAsKgAwIBAgIUXSGhi1rVH7ftDmJ6TlavLsY/74MwDQYJKoZIhvcNAQEL
BQAwgY8xCzAJBgNVBAYTAlVTMRAwDgYDVQQIDAdGbG9yaWRhMRAwDgYDVQQHDAdP
cmxhbmRvMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQxFzAVBgNV
BAMMDkVsYXN0aWMgQ2xpZW50MSAwHgYJKoZIhvcNAQkBFhFjbGllbnRAZWxhc3Rp
Yy5jbzAeFw0yNDA5MjUxOTAzNDZaFw0zNDA5MjMxOTAzNDZaMIGPMQswCQYDVQQG
EwJVUzEQMA4GA1UECAwHRmxvcmlkYTEQMA4GA1UEBwwHT3JsYW5kbzEhMB8GA1UE
CgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMRcwFQYDVQQDDA5FbGFzdGljIENs
aWVudDEgMB4GCSqGSIb3DQEJARYRY2xpZW50QGVsYXN0aWMuY28wggEiMA0GCSqG
SIb3DQEBAQUAA4IBDwAwggEKAoIBAQDAVGPlx4U2BpWfQlyMNraLMjdJAo4PjO2G
rDbwg2cAO4QFbMECEiNHakvuJ3zVVDO+HsBdkLWr8nO4iXmZDokfDrOrANJuqq16
p022soC8pJQz9uIBWTnxDGd/wdofi4H+V5uaMhw961sgB7GREyBWNRBQzhcFyQEP
XkR1/G52PcuzM5H9cnOSy7jc62g8Pkk8c2eZu3ADmvgWSH0b5pFUIvKsq068QjKP
qoHYXn38d/SSeCX57tzKsj+mBzp0cr1f9jeXmKeu68wPYG14aj9WmmY6ICAPvqPF
BKNLhXn2xPlZzv93zjiUR5bnitenVxvsmwjn5XvlgH56/fh3Y3aPAgMBAAGjLDAq
MAkGA1UdEwQCMAAwHQYDVR0OBBYEFJlsmqi3qid9YoWj4N7GQvAywRzbMA0GCSqG
SIb3DQEBCwUAA4IBAQCAtBwyiRYAGeAcN/UuMEcnMXP8QNrnCO/unoCbyFsByFQT
TcwMrS441hGPp/cAa8Fx0cP+oqrO99G1YHCzhprYVqIi/W9MsvRnR7Nh8SSS2/ld
0Gv9g+DU89NMzE5hlMCt5V0ydKbRj+ChKDsKlgQSopbrArjxHQv4Hb234HSZAR5N
OkJ1rNCF7wMD+xlNzEWZAHl7qjHuG8C4xWP207dXGYuY3064rBqv9hypLxj7RuZn
qesVBabxXBCL6Y1foh5OLLHyEWw28yfK/PnVdqU0lLrBhW9VJ6mQ9XCwZxf/tlSk
B3FafTQk4ZtU+4bVJuiAiQI7DeqpIFU6Lczds2gG
-----END CERTIFICATE-----
`, // the following key pem is decrypted value using key_passphrse
				"key_pem": `-----BEGIN RSA PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDAVGPlx4U2BpWf
QlyMNraLMjdJAo4PjO2GrDbwg2cAO4QFbMECEiNHakvuJ3zVVDO+HsBdkLWr8nO4
iXmZDokfDrOrANJuqq16p022soC8pJQz9uIBWTnxDGd/wdofi4H+V5uaMhw961sg
B7GREyBWNRBQzhcFyQEPXkR1/G52PcuzM5H9cnOSy7jc62g8Pkk8c2eZu3ADmvgW
SH0b5pFUIvKsq068QjKPqoHYXn38d/SSeCX57tzKsj+mBzp0cr1f9jeXmKeu68wP
YG14aj9WmmY6ICAPvqPFBKNLhXn2xPlZzv93zjiUR5bnitenVxvsmwjn5XvlgH56
/fh3Y3aPAgMBAAECggEAKbvmNX03BcMmAnn29SIGOGw8HOamBu/QtvF1tnj9B8ri
Wf8AXr+q0htZwKLm7q+nzrCDk4oMMfSZcci7DyBdVtTs3cV+5C67GCtnrKZNUyHv
WttOrXY8IXdMmidpeoDeQ1+lTy9ie3kvu+KPgGiDEtHO6Yne6w1z4m7VMjkFizh6
Y5sVPGTV1anoiO5tpW/D/Pshh/u+ZVsaeIE0iKV2NACJoMC0opsXBslldBDvXIaV
ZhpEnr5v3CO8i3roq2qp4Bplfyy8g89KXerg6G4JxM0NSafuWwGC/13chHSw3ke1
5odqnnwUyJ+MNvzyH0CPzF56vE3hFe6byeG3rD2euQKBgQD+MPtpvYGZfvMdVKNu
zms4s9CLcoM6lJXD72xL73x4MUcPQLOfk2YpCLDuVadCA8jBvL7d0OjulYbjJNCX
XZH2CITtTPzDEn+hfxKcy4Jd2M53yMQvpr55nmysEp+juuhkAo/xlXgGOcL3nq7o
hBLEAMy2NKwkszt1VLyZaD9SPQKBgQDBsrmb9INYJsCboA/VydKtzMB7Nnrck4S6
1BOsEp5JkepHPC92yolPCI94hJlTdISbt69nbcCGDvD87j2Z2UVP4zOfxT/SGHmF
LgLXgqzVCYcgvpBXZey9rPWvEPwQG/0ODgva7GxHnmVbP0LrQ4ynnX6FMsWaBYFd
K/0eEwY0uwKBgQCieGpqBr+sfbEk4TFpJLTx1DUKvJHWQpyLVSAyVQuIw254+FEX
QR5+QdjdLZAvqL2L33lbzCjmPlquGpzc8ujVilJ0Xs38XXmInvElmQpls6sccw26
q2h50eICBhFVlKTvL5gTwQarbAYLQbjoU2qvLxepqncRKiJp91Ro9XHrvQKBgG9K
mFiiEcFZartAKTkF4BXaGhHxSIBqBg4ugisQ+3975icNzpurXV9apMxzK4GG5hZu
YMrFhaPA+/fnjt9RtgBjo6q986BsTY4W1K0suM8izVAkDd0Zg/+rW/I9iQZcfnZP
3cHoq4Iu4T+fRnzUcAFyfVpcxKptVVnKR4G7Hoq3AoGBAPrrQtt4QvmAxPknKFgO
mDke0HqyWowuMkAHoVLL3VUaH8hrI1FA+oWEXpor4VqguB8D1SWb0qAJRF7pIxoE
m8TtceUhSOnXNrrO5agyMRmL0aYf8D425ot/uwTiSkOd4bdFeEaYs0ahHosxHq2N
me1zqwZ6EX7XHaa6j1mx9tcX
-----END RSA PRIVATE KEY-----
`,
				"cipher_suites":                []string{},
				"insecure_skip_verify":         false,
				"include_system_ca_certs_pool": false,
			},
			err: false,
			// TODO: Add  more  scenarios
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := TLSCommonToOTel(test.input)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, got, "beats to otel ssl mapping")
			}

		})
	}
}
