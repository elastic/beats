package outputs

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/joeshaw/multierror"
)

var (
	// ErrNotACertificate indicates a PEM file to be loaded not being a valid
	// PEM file or certificate.
	ErrNotACertificate = errors.New("file is not a certificate")

	// ErrCertificateNoKey indicate a configuration error with missing key file
	ErrCertificateNoKey = errors.New("key file not configured")

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrKeyNoCertificate = errors.New("certificate file not configured")
)

// TLSConfig defines config file options for TLS clients.
type TLSConfig struct {
	Enabled          *bool                         `config:"enabled"`
	VerificationMode transport.TLSVerificationMode `config:"verification_mode"` // one of 'none', 'full'
	Versions         []transport.TLSVersion        `config:"supported_protocols"`
	CipherSuites     []tlsCipherSuite              `config:"cipher_suites"`
	CAs              []string                      `config:"certificate_authorities"`
	Certificate      CertificateConfig             `config:",inline"`
	CurveTypes       []tlsCurveType                `config:"curve_types"`
}

type CertificateConfig struct {
	Certificate string `config:"certificate"`
	Key         string `config:"key"`
	Passphrase  string `config:"key_passphrase"`
}

type tlsCipherSuite uint16

type tlsCurveType tls.CurveID

var tlsCipherSuites = map[string]tlsCipherSuite{
	"ECDHE-ECDSA-AES-128-CBC-SHA":    tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA),
	"ECDHE-ECDSA-AES-128-GCM-SHA256": tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-ECDSA-AES-256-CBC-SHA":    tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA),
	"ECDHE-ECDSA-AES-256-GCM-SHA384": tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-ECDSA-RC4-128-SHA":        tlsCipherSuite(tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA),
	"ECDHE-RSA-3DES-CBC3-SHA":        tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA),
	"ECDHE-RSA-AES-128-CBC-SHA":      tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA),
	"ECDHE-RSA-AES-128-GCM-SHA256":   tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256),
	"ECDHE-RSA-AES-256-CBC-SHA":      tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA),
	"ECDHE-RSA-AES-256-GCM-SHA384":   tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384),
	"ECDHE-RSA-RC4-128-SHA":          tlsCipherSuite(tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA),
	"RSA-3DES-CBC3-SHA":              tlsCipherSuite(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA),
	"RSA-AES-128-CBC-SHA":            tlsCipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	"RSA-AES-128-GCM-SHA256":         tlsCipherSuite(tls.TLS_RSA_WITH_AES_128_GCM_SHA256),
	"RSA-AES-256-CBC-SHA":            tlsCipherSuite(tls.TLS_RSA_WITH_AES_256_CBC_SHA),
	"RSA-AES-256-GCM-SHA384":         tlsCipherSuite(tls.TLS_RSA_WITH_AES_256_GCM_SHA384),
	"RSA-RC4-128-SHA":                tlsCipherSuite(tls.TLS_RSA_WITH_RC4_128_SHA),
}

var tlsCurveTypes = map[string]tlsCurveType{
	"P-256": tlsCurveType(tls.CurveP256),
	"P-384": tlsCurveType(tls.CurveP384),
	"P-521": tlsCurveType(tls.CurveP521),
}

func (c *TLSConfig) Validate() error {
	hasCertificate := c.Certificate.Certificate != ""
	hasKey := c.Certificate.Key != ""

	switch {
	case hasCertificate && !hasKey:
		return ErrCertificateNoKey
	case !hasCertificate && hasKey:
		return ErrKeyNoCertificate
	}

	return nil
}

func (c *TLSConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

// LoadTLSConfig will load a certificate from config with all TLS based keys
// defined. If Certificate and CertificateKey are configured, client authentication
// will be configured. If no CAs are configured, the host CA will be used by go
// built-in TLS support.
func LoadTLSConfig(config *TLSConfig) (*transport.TLSConfig, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	fail := multierror.Errors{}
	logFail := func(es ...error) {
		for _, e := range es {
			if e != nil {
				fail = append(fail, e)
			}
		}
	}

	var cipherSuites []uint16
	for _, suite := range config.CipherSuites {
		cipherSuites = append(cipherSuites, uint16(suite))
	}

	var curves []tls.CurveID
	for _, id := range config.CurveTypes {
		curves = append(curves, tls.CurveID(id))
	}

	cert, err := loadCertificate(&config.Certificate)
	logFail(err)

	cas, errs := loadCertificateAuthorities(config.CAs)
	logFail(errs...)

	// fail, if any error occurred when loading certificate files
	if err = fail.Err(); err != nil {
		return nil, err
	}

	var certs []tls.Certificate
	if cert != nil {
		certs = []tls.Certificate{*cert}
	}

	// return config if no error occurred
	return &transport.TLSConfig{
		Versions:         config.Versions,
		Verification:     config.VerificationMode,
		Certificates:     certs,
		RootCAs:          cas,
		CipherSuites:     cipherSuites,
		CurvePreferences: curves,
	}, nil
}

func loadCertificate(config *CertificateConfig) (*tls.Certificate, error) {
	certificate := config.Certificate
	key := config.Key

	hasCertificate := certificate != ""
	hasKey := key != ""

	switch {
	case hasCertificate && !hasKey:
		return nil, ErrCertificateNoKey
	case !hasCertificate && hasKey:
		return nil, ErrKeyNoCertificate
	case !hasCertificate && !hasKey:
		return nil, nil
	}

	certPEM, err := readPEMFile(certificate, config.Passphrase)
	if err != nil {
		logp.Critical("Failed reading certificate file %v: %v", certificate, err)
		return nil, fmt.Errorf("%v %v", err, certificate)
	}

	keyPEM, err := readPEMFile(key, config.Passphrase)
	if err != nil {
		logp.Critical("Failed reading key file %v: %v", key, err)
		return nil, fmt.Errorf("%v %v", err, key)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		logp.Critical("Failed loading client certificate", err)
		return nil, err
	}

	return &cert, nil
}

func readPEMFile(path, passphrase string) ([]byte, error) {
	pass := []byte(passphrase)
	var blocks []*pem.Block

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for len(content) > 0 {
		var block *pem.Block

		block, content = pem.Decode(content)
		if block == nil {
			if len(blocks) == 0 {
				return nil, errors.New("no pem file")
			}
			break
		}

		if x509.IsEncryptedPEMBlock(block) {
			var buffer []byte
			var err error
			if len(pass) == 0 {
				err = errors.New("No passphrase available")
			} else {
				// Note, decrypting pem might succeed even with wrong password, but
				// only noise will be stored in buffer in this case.
				buffer, err = x509.DecryptPEMBlock(block, pass)
			}

			if err != nil {
				logp.Err("Dropping encrypted pem '%v' block read from %v. %v",
					block.Type, path, err)
				continue
			}

			// DEK-Info contains encryption info. Remove header to mark block as
			// unencrypted.
			delete(block.Headers, "DEK-Info")
			block.Bytes = buffer
		}
		blocks = append(blocks, block)
	}

	if len(blocks) == 0 {
		return nil, errors.New("no PEM blocks")
	}

	// re-encode available, decrypted blocks
	buffer := bytes.NewBuffer(nil)
	for _, block := range blocks {
		err := pem.Encode(buffer, block)
		if err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func loadCertificateAuthorities(CAs []string) (*x509.CertPool, []error) {
	errors := []error{}

	if len(CAs) == 0 {
		return nil, nil
	}

	roots := x509.NewCertPool()
	for _, path := range CAs {
		pemData, err := ioutil.ReadFile(path)
		if err != nil {
			logp.Critical("Failed reading CA certificate: %v", err)
			errors = append(errors, fmt.Errorf("%v reading %v", err, path))
			continue
		}

		if ok := roots.AppendCertsFromPEM(pemData); !ok {
			logp.Critical("Failed reading CA certificate: %v", err)
			errors = append(errors, fmt.Errorf("%v adding %v", ErrNotACertificate, path))
			continue
		}
	}

	return roots, errors
}

func (cs *tlsCipherSuite) Unpack(in interface{}) error {
	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("tls cipher suite must be an identifier")
	}

	suite, found := tlsCipherSuites[s]
	if !found {
		return fmt.Errorf("invalid tls cipher suite '%v'", s)
	}

	*cs = suite
	return nil
}

func (ct *tlsCurveType) Unpack(in interface{}) error {
	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("tls curve type must be an identifier")
	}

	t, found := tlsCurveTypes[s]
	if !found {
		return fmt.Errorf("invalid tls curve type '%v'", s)
	}

	*ct = t
	return nil

}
