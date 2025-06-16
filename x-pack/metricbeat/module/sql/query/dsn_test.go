// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package query

import (
	"os"
	"testing"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockCA   = "-----BEGIN CERTIFICATE-----\nMIIDITCCAgmgAwIBAgIUK5BTuk98yrDnFcOM0JiBh74FEQ8wDQYJKoZIhvcNAQEL\nBQAwFDESMBAGA1UEAwwJbG9jYWxob3N0MCAXDTI1MDYxNjEzMDI0MVoYDzIxMjQw\nMTA5MTMwMjQxWjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQCo8aaCj86L+gQuLeAbhV/VvPqKrOs21lZ1NsnRc1rV\njoany8jq4GcRiLEbbQfLQJ2gGq1KjBnfojo8yVI5JGL5yZn8PAGz9ZFJBymsfwFt\nCumh2fnWWjeg2P5+62dDR50KFjoSFkEU0Mk14g2Gq0RNmHcWuRYq18jDOomL6jbc\nkVw//1eriyZZ49K25ddGZZeSjXw8tjWDsJyok48PXKCA5mU8XLzshtoEa49z01fW\nrfMICT/lGbHUVa7xqx+oIoreCTjed0c8OS2bKCH9JAlk09Iqu6eDGkTTETpmp5qo\nHfCRoXSVRBqJub3ISGjYrMMXQfakYPDfzDj3D/GlNySZAgMBAAGjaTBnMB0GA1Ud\nDgQWBBSZsH+4+i3NgY/an9Xaa4878jw1+jAfBgNVHSMEGDAWgBSZsH+4+i3NgY/a\nn9Xaa4878jw1+jAPBgNVHRMBAf8EBTADAQH/MBQGA1UdEQQNMAuCCWxvY2FsaG9z\ndDANBgkqhkiG9w0BAQsFAAOCAQEAXPbUKy7KVnoN8/nU2qaRSy+GZRJTl9UTX8B+\nJJPGUpo/QmKZkDdZIQO2KTjtjH2j58ThSu3MWMfDA2s8rssPrCoLNmgtX+7N3F22\nCl34Tn41wFv9VUVj03eCr5q0PYnkhFMhIjsj6AkwF9uh9uISaBFYc7WXQrzhGa0y\n6oK4rp8oxrmhBF5qccwz7dmhMW9TsC0/B6e5MuOTqY+Bkr0jV5FiT0ccUJG+SQ8d\nsKgyna9L/WeIorE6QN98TOnksbeRt28tmUp1nVAD+6vGCwfea2eCqwOA7uehgXSJ\n7pOpdUBbvnalEhdR+K0q0ZaAUINePrZppa6dwLgbjAIFsDhPmA==\n-----END CERTIFICATE-----\n"
	mockCert = "-----BEGIN CERTIFICATE-----\nMIIC/TCCAeWgAwIBAgIBATANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAlsb2Nh\nbGhvc3QwIBcNMjUwNjE2MTMwMzEzWhgPMjEyNDAxMDkxMzAzMTNaMBQxEjAQBgNV\nBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBANg+\nvGjQ8nvHLknVzbfKJT+v5cBwpl0KeLf4FKCk/o8WUamd5Bww3+1wvep3VXj+Ik2/\nj41NQEBo1J4n6/hR6A+OEoZs+magUcwM26FiNv2OZoYb6Oy1Moj0oxJZq16js627\nVOWOOwlPtmmHvhmg5r40sQS+MZhs9oFZCH7NFwqmsi4CVWtwdwS9vR+/tOrdi/X4\nDtTGvfJk+MRDctbGrFn1NgvlKGvQm4uccfWS2V35L4pH5VMW+PsoYogHIex1hmvl\nJxZT+HAX2qA4B1HX3zNh398VMcRA7NG52B06symCEaA3Qpw3DVE1Fa3qH2dskyk/\nmuil/1wz/ObfrKJm3E0CAwEAAaNYMFYwFAYDVR0RBA0wC4IJbG9jYWxob3N0MB0G\nA1UdDgQWBBQ1DNW2Jqwxnl9PwouJo6xEX2pi8zAfBgNVHSMEGDAWgBSZsH+4+i3N\ngY/an9Xaa4878jw1+jANBgkqhkiG9w0BAQsFAAOCAQEAGBxCXhnCt1l1eHfhhlAr\nP4r+vqxM8X2SM+e/md1LbQDet4iXelsbpmVCLT6mzS2zzBubHTaSua3O4qYtIGDt\n7KiKla/jo/WEcaIq1TkFvOPoFuwNtycxODHtBe7jPTk8cjnGehM3JDQdCRRtI2aa\nc8MXXFafkWNPgJo93+7OQd3EWb8bJ8Th62BB6gcRpAVExb314CUtjRydhwQW9Xoo\nAjhf1NhxpBdvnq3UJHJsGyS70dRAnKQuq6TIcBkVZ5Z5ExmemytI1aKQWhpFAule\nnJQQz75wxkGSvX8fW7q10tJOOmAfapM7Y14dv5FyHy0b0zvdYERXng7jTsJdrLdZ\n3g==\n-----END CERTIFICATE-----\n"
	//gitguardian:ignore
	mockKey = "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDYPrxo0PJ7xy5J\n1c23yiU/r+XAcKZdCni3+BSgpP6PFlGpneQcMN/tcL3qd1V4/iJNv4+NTUBAaNSe\nJ+v4UegPjhKGbPpmoFHMDNuhYjb9jmaGG+jstTKI9KMSWateo7Otu1TljjsJT7Zp\nh74ZoOa+NLEEvjGYbPaBWQh+zRcKprIuAlVrcHcEvb0fv7Tq3Yv1+A7Uxr3yZPjE\nQ3LWxqxZ9TYL5Shr0JuLnHH1ktld+S+KR+VTFvj7KGKIByHsdYZr5ScWU/hwF9qg\nOAdR198zYd/fFTHEQOzRudgdOrMpghGgN0KcNw1RNRWt6h9nbJMpP5ropf9cM/zm\n36yiZtxNAgMBAAECggEAM2u3obUN9CEJAMW2hV2sPdi16WzgIn+69QQo44pYfe3w\nvUSuWYXFudB1WKvHx12nCpXirNcR0D8dT/5uPj470HcYMJ75bC3zRXJJR7bzHJgg\nCQPZ/2+W9Lo3jMWF2ptSvp0tMuj/YNdzqOR+b9mzBMfC0D3pzTUb6OYi/wQF1qId\nDrFjOgE1NrujJqJsqSiTDbveA/ipmPR5h4Ivm9ibHT70p07Yk1j2t+NKszLqws5R\nxRuPRCMNrlF8BFqCcWdl/T//AUQdwH4ffh+PoNZqrCNIIQ9dOxMC5h5X60Rt1Zgl\nmisjkW+NJ9+xyNt/BRVlEgq3Nw1QITDzNV0RpEJkgQKBgQDw6lCXLHsiy3ceOTIf\n5mBr0sH17fTSY1SlaJvXrq31Y03SNOr/nigVxFX8N912NRoSlYdEyArGN3PXXtA7\nA0g6cMs6ltURke7ghRva7ahTEsM+SMyL08gEs3gHBWtfz2t2EUaE8i1+0aOB3Vrt\nfgJGvjeVe6NrTP21oU0EH8bMRwKBgQDlyPn9qh8h5F3ZvO8KDROHU9pwKkIwqFPV\nQ4tdr5VyRIzQBjrRgayQA9NL2AfzXIAnrQrv7IVVsuXJW074lnxJzUUXDm0Llryw\nkvpOBwsWVgIs0UxbH4P6oSC7Yf0sBmiwz1KNW+yrBnQ+hB6SoAMZ1PTx8nSHuxZn\nO9fuZvYgywKBgBv8TCJTg3ZWRl8Xa9Ay1c6QrAFihAcQjNuuHDRg0UppH7gkd4v8\nFlH4/bgP0UUTBBVWk2EVD9NYy7cgB3Zjejd3tNP4g4XH+wTP0Z2L7/q+ejm5ATHZ\nByosouvF4GQ/1w7fEN8OtuQ9fA3w5cgi1Cbdn91YgHJNfkdkFms9Ob2vAoGATxpl\nvQZwmzlDea6J17ryqxaZzx0tFhUMbxFGi+TjHKgulXpfizoJzrYSajyfWA7S61Wt\nuzSAHiVs52lwgTFE7h8lFq/XqDKnGF4wnuXb0j+flhAjKgdqZsBLRVaRUjOOnLdy\nYslvatzY7aCL6cv95UmjXRsrNIKaTsWSKzb0qgsCgYALNEng4BEnezblkq10Scci\nPBi4QLKAxae7s22AVvVvD+Wk1dNA4N1cGJC4MKWDZNZrJn59QjWZ0NrEcICRdjuU\nWQ8OTkBq4a2wyEfH7lv6MAIZ8E36B3jP456HbjblgUtM+Xc1Vm9YI5gv2TMViYDw\n8XSCt63jIOxnyMT4aMgRgg==\n-----END PRIVATE KEY-----\n"

	caPath   = "./ca.pem"
	certPath = "./cert.pem"
	keyPath  = "./key.pem"
)

func prepare(t *testing.T) {
	err := os.WriteFile(caPath, []byte(mockCA), 0644)
	require.NoError(t, err)

	err = os.WriteFile(keyPath, []byte(mockKey), 0600)
	require.NoError(t, err)

	err = os.WriteFile(certPath, []byte(mockCert), 0644)
	require.NoError(t, err)
}

func cleanup(t *testing.T) {
	err := os.Remove(caPath)
	require.NoError(t, err)

	err = os.Remove(keyPath)
	require.NoError(t, err)

	err = os.Remove(certPath)
	require.NoError(t, err)
}

func TestMysqlParseDSN(t *testing.T) {
	var tlsEnabled = true

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
		prepare(t)
		defer cleanup(t)

		config := ConnectionDetails{
			TLS: &tlscommon.Config{
				Enabled:          &tlsEnabled,
				VerificationMode: tlscommon.VerifyFull,
				CAs:              []string{caPath},
				Certificate: tlscommon.CertificateConfig{
					Certificate: certPath,
					Key:         keyPath,
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
}

func TestPostgresParseDSN(t *testing.T) {
	var tlsEnabled = true

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
		prepare(t)
		defer cleanup(t)

		config := ConnectionDetails{
			TLS: &tlscommon.Config{
				Enabled:          &tlsEnabled,
				VerificationMode: tlscommon.VerifyFull,
				CAs:              []string{caPath},
				Certificate: tlscommon.CertificateConfig{
					Certificate: certPath,
					Key:         keyPath,
				},
			},
		}
		host := "postgres://localhost:5432/mydb"

		hostData, err := postgresParseDSN(config, host)
		require.NoError(t, err)

		assert.Equal(t, "postgres://localhost:5432/mydb?sslcert=.%2Fcert.pem&sslkey=.%2Fkey.pem&sslmode=verify-full&sslrootcert=.%2Fca.pem", hostData.URI)
		assert.Equal(t, "localhost:5432", hostData.SanitizedURI)
		assert.Equal(t, "localhost:5432", hostData.Host)
	})

	t.Run("TLS enabled with multiple CA certificates", func(t *testing.T) {
		prepare(t)
		defer cleanup(t)

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
}

func TestMssqlParseDSN(t *testing.T) {
	var tlsEnabled = true

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
		prepare(t)
		defer cleanup(t)

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
		prepare(t)
		defer cleanup(t)

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
		prepare(t)
		defer cleanup(t)

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
}
