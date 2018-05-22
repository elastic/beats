package outputs

import (
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// Managing TLS option with the outputs package is deprecated move your code to use the tlscommon
// package.
var (
	// ErrNotACertificate indicates a PEM file to be loaded not being a valid
	// PEM file or certificate.
	ErrNotACertificate = tlscommon.ErrNotACertificate

	// ErrCertificateNoKey indicate a configuration error with missing key file
	ErrCertificateNoKey = tlscommon.ErrCertificateNoKey

	// ErrKeyNoCertificate indicate a configuration error with missing certificate file
	ErrKeyNoCertificate = tlscommon.ErrKeyNoCertificate
)

// TLSConfig defines config file options for TLS clients.
type TLSConfig = tlscommon.Config

// CertificateConfig define a common set of fields for a certificate.
type CertificateConfig = tlscommon.CertificateConfig

// LoadTLSConfig will load a certificate from config with all TLS based keys
// defined. If Certificate and CertificateKey are configured, client authentication
// will be configured. If no CAs are configured, the host CA will be used by go
// built-in TLS support.
var LoadTLSConfig = tlscommon.LoadTLSConfig

// LoadCertificate will load a certificate from disk and return a tls.Certificate or error
var LoadCertificate = tlscommon.LoadCertificate

// ReadPEMFile reads a PEM format file on disk and decrypt it with the privided password and
// return the raw content.
var ReadPEMFile = tlscommon.ReadPEMFile

// LoadCertificateAuthorities read the slice of CAcert and return a Certpool.
var LoadCertificateAuthorities = tlscommon.LoadCertificateAuthorities
