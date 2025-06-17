// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	mockCA   = "-----BEGIN CERTIFICATE-----\nMIIDITCCAgmgAwIBAgIUK5BTuk98yrDnFcOM0JiBh74FEQ8wDQYJKoZIhvcNAQEL\nBQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI1MDYxNjEzMDI0MVoYDzIxMjQw\nMTA5MTMwMjQxWjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQCo8aaCj86L+gQuLeAbhV/VvPqKrOs21lZ1NsnRc1rV\njoany8jq4GcRiLEbbQfLQJ2gGq1KjBnfojo8yVI5JGL5yZn8PAGz9ZFJBymsfwFt\nCumh2fnWWjeg2P5+62dDR50KFjoSFkEU0Mk14g2Gq0RNmHcWuRYq18jDOomL6jbc\nkVw//1eriyZZ49K25ddGZZeSjXw8tjWDsJyok48PXKCA5mU8XLzshtoEa49z01fW\nrfMICT/lGbHUVa7xqx+oIoreCTjed0c8OS2bKCH9JAlk09Iqu6eDGkTTETpmp5qo\nHfCRoXSVRBqJub3ISGjYrMMXQfakYPDfzDj3D/GlNySZAgMBAAGjaTBnMB0GA1Ud\nDgQWBBSZsH+4+i3NgY/an9Xaa4878jw1+jAfBgNVHSMEGDAWgBSZsH+4+i3NgY/a\nn9Xaa4878jw1+jAPBgNVHRMBAf8EBTADAQH/MBQGA1UdEQQNMAuCCWxvY2FsaG9z\ndDANBgkqhkiG9w0BAQsFAAOCAQEAXPbUKy7KVnoN8/nU2qaRSy+GZRJTl9UTX8B+\nJJPGUpo/QmKZkDdZIQO2KTjtjH2j58ThSu3MWMfDA2s8rssPrCoLNmgtX+7N3F22\nCl34Tn41wFv9VUVj03eCr5q0PYnkhFMhIjsj6AkwF9uh9uISaBFYc7WXQrzhGa0y\n6oK4rp8oxrmhBF5qccwz7dmhMW9TsC0/B6e5MuOTqY+Bkr0jV5FiT0ccUJG+SQ8d\nsKgyna9L/WeIorE6QN98TOnksbeRt28tmUp1nVAD+6vGCwfea2eCqwOA7uehgXSJ\n7pOpdUBbvnalEhdR+K0q0ZaAUINePrZppa6dwLgbjAIFsDhPmA==\n-----END CERTIFICATE-----\n"
	mockCert = "-----BEGIN CERTIFICATE-----\nMIIC/TCCAeWgAwIBAgIBATANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAlsb2Nh\nbGhvc3QwIBcNMjUwNjE3MDgwNTQzWhgPMjEyNDAxMTAwODA1NDNaMBQxEjAQBgNV\nBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAIqe\n6Qd/+4MS0vyz004fqh/rhqSodyRxHDz3HHHEI1xYB1Iz/RzkusgSqRtsa1HTL0pN\nXOq/8JFLLg4x1IrEn6Fp0dWf1qnxD7OatKe5HDgAdb5Wh1j4dU4ipnNMdfMg0VYv\nZ2KHR9TvIvpU0fzcFBmkBBRHrrsXAm4IFqSX5xW4oc2Thlhv+tOUH6kUpWGUkSRV\nqBqLMzuFtwVVOQXkgOguDoL5PC4MVXzmdx8Bwut2yj8gB2vqTuB5LDoSHu8xwx7J\nqngNroJM7jhpI0raS6Eek/gMznARXRevWLjcvJHRR2BKkPvF7g+UDVhoupgyezac\nny/OLkKeDEoWKEhnMHMCAwEAAaNYMFYwFAYDVR0RBA0wC4IJbG9jYWxob3N0MB0G\nA1UdDgQWBBRu3N763N/O3YMoEtLOeHcru8lGqTAfBgNVHSMEGDAWgBSZsH+4+i3N\ngY/an9Xaa4878jw1+jANBgkqhkiG9w0BAQsFAAOCAQEAj8CDiMZOJzL2SuD87iUo\n8nr08w0SvHm/qLB0KXJXFXPAI7GO6GRA+tNj0N31Dza4n2ex6/hwbxdpNzlvPOdz\nMCKTyV07G45kiJ34wA7YfVObdgFGvtwDnqy4aca6eG7nsBVaAmPTbGvG4Nmidcir\nqkMMb1C2OkOc03EHp7kCtgzllCG1GK89/LdtJQE2VSCtPNIwmcWNwE87w3WnoLYL\nBosa4ijRwhOx4lB8osdXgxHkC/u9F1uLUFNo4xb7fkNdfP63T3UdBkPORL2PpQA9\nr6D85scEbCi68pKLNpEXGCB9hnaXN0aPh+RjaChmZjMVam6QCYuuCvT0DsLoC/W4\nEw==\n-----END CERTIFICATE-----\n"
	//gitguardian:ignore
	mockKey         = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCKnukHf/uDEtL8\ns9NOH6of64akqHckcRw89xxxxCNcWAdSM/0c5LrIEqkbbGtR0y9KTVzqv/CRSy4O\nMdSKxJ+hadHVn9ap8Q+zmrSnuRw4AHW+VodY+HVOIqZzTHXzINFWL2dih0fU7yL6\nVNH83BQZpAQUR667FwJuCBakl+cVuKHNk4ZYb/rTlB+pFKVhlJEkVagaizM7hbcF\nVTkF5IDoLg6C+TwuDFV85ncfAcLrdso/IAdr6k7geSw6Eh7vMcMeyap4Da6CTO44\naSNK2kuhHpP4DM5wEV0Xr1i43LyR0UdgSpD7xe4PlA1YaLqYMns2nJ8vzi5CngxK\nFihIZzBzAgMBAAECggEAB5uzNIsstcP5uo7wIRCR2NCngjAQ4fonT51MfV4DhtT6\nCeP6l3RiYArOJ0grF8GcjdpzKBtCy+axb2wCu18RV92j+7KbKJangvcRxUbeqqAz\n1i+PnC1+2rwCIL/olWCOvMk7RmggZCp/4/d10wgNPl8HLknE6FXZ90oQXBZOQ53a\n7LJPbBy2mf7n1vg50tKwNSeQ6Y3STpSYmDz4isKoE/9HyT7GXzYOdHmWJzzUCUkZ\npxeO/002y2PBZEJAUlHVnDDuZae2cXWASejXu7MDeWzwukSgj3/0FFKs6RTKyu/n\n6Ilowe5V55c5PjzY1968r1KPb5K9BjI89WWZ5OtfKQKBgQDAVgdNs9Jv6g6qMPYJ\nJdMtMDIpOxqoc+ImSBI3+MfEEFp5XQmpGYT0QiLDQdFwJ/loabWmNWMf7QM5kpL1\ngQtO+3Gb8RX/r89y8TvCncirK2UxN86RqpTGWVl9Cc42JTd7A7X3HECqf6iRYbxl\nn5Tp7JkmpvoxBaLg0Ctvei2abwKBgQC4gTXP2aCxyr1W3/PaAYBo4CfnRXIeJSns\nNIDLsUnMz1TqKNBbo7F3HLpDbWGf+bZAcOhqi9iqrGS9fSz9aAS5UFs3sXLM/HwT\ne3po0nOLgScw2HyqgdaUBPQyGh5DQjRkNTNcgADT1HKmEEnBUArqvwASZTsAX9YC\nAoY1Y07cPQKBgQCej01+FVzKvl5QmAR9Dh3GBxGTRBJ6BO7POGMsmX+2dvTfUIAC\nU/NzmoImDkCnAY1vMpZ561FIpJAgCmH02umDt261bE8CduHClHT7wDAKTMAjjypQ\nlBwKWOaZWlgR8ySF2U1N5pC4/nztPXGfJawSHOc1Ijrn5wmb5IGqaULnKQKBgHUc\nHonleuge5XtE/0T6+wSWcv2KyNp1gFybHr0rtMo5N47BhS8Fgdk29MtjnDmsiI/y\nmrM2PLpoXjEgSPQ3l/gAF0YMbe/Kuv6qu5HZMtnzimqonsijTQ367vz2MwtB9Hs+\ngXFPFjdee78IS6hWI/fIcEU81+xu6CmybHlqpV2JAoGARij6b/bWgWnEYzPsBHWS\nF4vH+x2uMXWeZrPYVtge8XBinCvPOX6QgsozKIEO2Bkh9H6i6FRa6m0PHXxud/Pg\nkeUy/jJ2vWdymqRVA21E7PWzUzHyvjY9JsviPVMx8aDW9wcGBvi7CAnKUphoIvmu\nbKhz9e1dN19sx1DIYrvDSGs=\n-----END PRIVATE KEY-----\n"
	mockKeyPassword = "test"
	caPath          = "./ca.pem"
	certPath        = "./cert.pem"
	keyPath         = "./key.pem"
)

func prepare(t *testing.T) {
	require.NoError(t, os.WriteFile(caPath, []byte(mockCA), 0644))
	require.NoError(t, os.WriteFile(keyPath, []byte(mockKey), 0600))
	require.NoError(t, os.WriteFile(certPath, []byte(mockCert), 0644))
}

func cleanup(t *testing.T) {
	require.NoError(t, os.Remove(caPath))
	require.NoError(t, os.Remove(keyPath))
	require.NoError(t, os.Remove(certPath))
}

func TestParseDSNfunctions(t *testing.T) {
	prepare(t)
	defer cleanup(t)

	tlsEnabled := true

	t.Run("mysql", func(t *testing.T) {
		t.Run("TLS disabled", func(t *testing.T) {
			config := ConnectionDetails{}
			host := "root:test@tcp(localhost:3306)/"

			hostData, err := mysqlParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, host, hostData.URI)
			assert.Equal(t, "localhost:3306", hostData.SanitizedURI)
			assert.Equal(t, "localhost:3306", hostData.Host)
		})

		t.Run("TLS enabled with valid configuration", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled:          &tlsEnabled,
					VerificationMode: tlscommon.VerifyFull,
					CAs:              []string{caPath},
					Certificate: tlscommon.CertificateConfig{
						Certificate: certPath,
						Key:         keyPath,
						Passphrase:  mockKeyPassword,
					},
				},
			}
			host := "root:test@tcp(localhost:3306)/"

			hostData, err := mysqlParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, "root:test@tcp(localhost:3306)/?tls=custom", hostData.URI)
			assert.Equal(t, "localhost:3306", hostData.SanitizedURI)
			assert.Equal(t, "localhost:3306", hostData.Host)
		})

		t.Run("TLS enabled with invalid CA certificate path", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{"/path/to/invalid/ca.crt"},
				},
			}
			host := "root:test@tcp(localhost:3306)/"

			_, err := mysqlParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "could not load provided TLS configuration")
		})

		t.Run("TLS enabled with PEM-formatted CA certificate", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{mockCA},
				},
			}
			host := "root:test@tcp(localhost:3306)/"

			hostData, err := mysqlParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, "root:test@tcp(localhost:3306)/?tls=custom", hostData.URI)
			assert.Equal(t, "localhost:3306", hostData.SanitizedURI)
			assert.Equal(t, "localhost:3306", hostData.Host)
		})
	})

	t.Run("postgres", func(t *testing.T) {
		t.Run("TLS disabled", func(t *testing.T) {
			config := ConnectionDetails{}
			host := "postgres://localhost:5432/mydb"

			hostData, err := postgresParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, host, hostData.URI)
			assert.Equal(t, "localhost:5432", hostData.SanitizedURI)
			assert.Equal(t, "localhost:5432", hostData.Host)
		})

		t.Run("TLS enabled with valid configuration", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled:          &tlsEnabled,
					VerificationMode: tlscommon.VerifyFull,
					CAs:              []string{caPath},
					Certificate: tlscommon.CertificateConfig{
						Certificate: certPath,
						Key:         keyPath,
						Passphrase:  mockKeyPassword,
					},
				},
			}
			host := "postgres://localhost:5432/mydb"

			hostData, err := postgresParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, "postgres://localhost:5432/mydb?sslcert=.%2Fcert.pem&sslkey=.%2Fkey.pem&sslmode=verify-full&sslpassword=test&sslrootcert=.%2Fca.pem", hostData.URI)
			assert.Equal(t, "localhost:5432", hostData.SanitizedURI)
			assert.Equal(t, "localhost:5432", hostData.Host)
		})

		t.Run("TLS enabled with multiple CA certificates", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{caPath, caPath},
				},
			}
			host := "postgres://localhost:5432/mydb"

			_, err := postgresParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "postgres driver supports only one CA certificate")
		})

		t.Run("TLS enabled with PEM formatted CA certificate", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{mockCA},
				},
			}
			host := "postgres://localhost:5432/mydb"

			_, err := postgresParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "postgres driver supports only certificate file path")
		})
	})

	t.Run("mssql", func(t *testing.T) {
		t.Run("TLS disabled", func(t *testing.T) {
			config := ConnectionDetails{}
			host := "sqlserver://localhost:1433?database=mydb"

			hostData, err := mssqlParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, host, hostData.URI)
			assert.Equal(t, "localhost:1433", hostData.SanitizedURI)
			assert.Equal(t, "localhost:1433", hostData.Host)
		})

		t.Run("TLS enabled with valid configuration", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled:          &tlsEnabled,
					VerificationMode: tlscommon.VerifyFull,
					CAs:              []string{caPath},
				},
			}
			host := "sqlserver://localhost:1433?database=mydb"

			hostData, err := mssqlParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, "sqlserver://localhost:1433?TrustServerCertificate=false&certificate=.%2Fca.pem&database=mydb&encrypt=true", hostData.URI)
			assert.Equal(t, "localhost:1433", hostData.SanitizedURI)
			assert.Equal(t, "localhost:1433", hostData.Host)
		})

		t.Run("TLS enabled with multiple CA certificates", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{caPath, caPath},
				},
			}
			host := "sqlserver://localhost:1433?database=mydb"

			_, err := mssqlParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "mssql driver supports only one CA certificate")
		})

		t.Run("TLS enabled with client key and/or certificate", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					Certificate: tlscommon.CertificateConfig{
						Certificate: certPath,
						Key:         keyPath,
					},
				},
			}
			host := "sqlserver://localhost:1433?database=mydb"

			_, err := mssqlParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "mssql driver supports only CA certificate")
		})

		t.Run("TLS enabled with PEM formatted CA certificate", func(t *testing.T) {
			config := ConnectionDetails{
				TLS: &tlscommon.Config{
					Enabled: &tlsEnabled,
					CAs:     []string{mockCA},
				},
			}
			host := "sqlserver://localhost:1433?database=mydb"

			_, err := mssqlParseDSN(config, host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "mssql driver supports only certificate file path")
		})
	})

	t.Run("defaultParseDSN", func(t *testing.T) {
		t.Run("connection string is URL-formatted", func(t *testing.T) {
			config := ConnectionDetails{}
			host := "postgres://myuser:mypassword@localhost:5432/mydb"

			hostData, err := defaultParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, host, hostData.URI)
			assert.Equal(t, "localhost:5432", hostData.SanitizedURI)
			assert.Equal(t, "localhost:5432", hostData.Host)
		})

		t.Run("connection string is NOT URL-formatted", func(t *testing.T) {
			config := ConnectionDetails{}
			host := "user=myuser password=mypassword dbname=mydb sslmode=disable"

			hostData, err := defaultParseDSN(config, host)
			require.NoError(t, err)

			assert.Equal(t, host, hostData.URI)
			assert.Equal(t, "(redacted)", hostData.SanitizedURI)
			assert.Equal(t, "(redacted)", hostData.Host)
		})

	})
}
