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

//go:build windows && !requirefips

package translate_ldap_attribute

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"

	"github.com/alexbrainman/sspi"
	"github.com/alexbrainman/sspi/kerberos"
	"github.com/go-ldap/ldap/v3"
	"github.com/go-ldap/ldap/v3/gssapi"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Windows LDAP channel binding application data:
// "tls-server-end-point:" + RFC 5929 hash of the TLS server cert.
// This is the application data expected by AD when LDAP channel binding is enforced.
// See: https://gary-nebbett.blogspot.com/2020/01/ldap-channel-binding.html
const tlsServerEndPointPrefix = "tls-server-end-point:"

// SECBUFFER_CHANNEL_BINDINGS is the SSPI buffer type for channel bindings (0x0E).
// Defined in sspi.h as SECBUFFER_CHANNEL_BINDINGS = 14.
const secBufferChannelBindings = 14

// secChannelBindingsHeader is SEC_CHANNEL_BINDINGS (Windows SSPI).
// https://learn.microsoft.com/en-us/windows/win32/api/sspi/ns-sspi-sec_channel_bindings
// Header size is 8 uint32 fields (8 * 4 bytes = 32).
const secChannelBindingsHeaderSize = 32

// tlsServerEndpointChannelBindingData builds the LDAP CBT application data.
// The returned bytes are appended to the SEC_CHANNEL_BINDINGS header.
func tlsServerEndpointChannelBindingData(cs *tls.ConnectionState) ([]byte, error) {
	if cs == nil {
		return nil, errors.New("nil TLS connection state")
	}
	if len(cs.PeerCertificates) == 0 {
		return nil, errors.New("no peer TLS certificates for channel binding")
	}
	certDER := cs.PeerCertificates[0].Raw
	if len(certDER) == 0 {
		return nil, errors.New("empty peer certificate DER")
	}
	hash, err := tlsServerEndpointHash(certDER, cs.PeerCertificates[0].SignatureAlgorithm)
	if err != nil {
		return nil, err
	}
	app := make([]byte, 0, len(tlsServerEndPointPrefix)+len(hash))
	app = append(app, tlsServerEndPointPrefix...)
	app = append(app, hash...)
	return app, nil
}

// tlsServerEndpointHash implements RFC 5929 for "tls-server-end-point".
// Use the cert's signature hash (except MD5/SHA1 -> SHA-256).
func tlsServerEndpointHash(certDER []byte, alg x509.SignatureAlgorithm) ([]byte, error) {
	switch alg {
	case x509.MD5WithRSA, x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
		sum := sha256.Sum256(certDER)
		return sum[:], nil
	case x509.DSAWithSHA256, x509.ECDSAWithSHA256, x509.SHA256WithRSA, x509.SHA256WithRSAPSS:
		sum := sha256.Sum256(certDER)
		return sum[:], nil
	case x509.ECDSAWithSHA384, x509.SHA384WithRSA, x509.SHA384WithRSAPSS:
		sum := sha512.Sum384(certDER)
		return sum[:], nil
	case x509.ECDSAWithSHA512, x509.SHA512WithRSA, x509.SHA512WithRSAPSS:
		sum := sha512.Sum512(certDER)
		return sum[:], nil
	case x509.PureEd25519, x509.UnknownSignatureAlgorithm:
		return nil, fmt.Errorf("unsupported signature algorithm for tls-server-end-point: %s", alg)
	default:
		return nil, fmt.Errorf("unsupported signature algorithm for tls-server-end-point: %s", alg)
	}
}

// marshalSecChannelBindings packs application data into SEC_CHANNEL_BINDINGS.
// Offsets are from the start of the structure (MSDN).
// marshalSecChannelBindings packs application data into the SEC_CHANNEL_BINDINGS layout
// expected by SSPI when passed as SECBUFFER_CHANNEL_BINDINGS.
func marshalSecChannelBindings(applicationData []byte) []byte {
	total := secChannelBindingsHeaderSize + len(applicationData)
	b := make([]byte, total)
	// Initiator / acceptor address fields are unused for TLS endpoint binding.
	// Offsets 24 and 28 map to cbApplicationDataLength and dwApplicationDataOffset.
	binary.LittleEndian.PutUint32(b[24:28], uint32(len(applicationData)))
	binary.LittleEndian.PutUint32(b[28:32], secChannelBindingsHeaderSize)
	copy(b[secChannelBindingsHeaderSize:], applicationData)
	return b
}

// updateKerberosContextWithCBT mirrors sspi/internal/common.UpdateContext, but
// adds a SECBUFFER_CHANNEL_BINDINGS input buffer to carry LDAP CBT.
func updateKerberosContextWithCBT(c *sspi.Context, dst, src []byte, targetName *uint16, cbt []byte) (authCompleted bool, n int, err error) {
	var inBuf [2]sspi.SecBuffer
	inBuf[0].Set(sspi.SECBUFFER_TOKEN, src)
	inBuf[1].Set(secBufferChannelBindings, cbt)
	inBufs := &sspi.SecBufferDesc{
		Version:      sspi.SECBUFFER_VERSION,
		BuffersCount: 2,
		Buffers:      &inBuf[0],
	}

	var outBuf [1]sspi.SecBuffer
	outBuf[0].Set(sspi.SECBUFFER_TOKEN, dst)
	outBufs := &sspi.SecBufferDesc{
		Version:      sspi.SECBUFFER_VERSION,
		BuffersCount: 1,
		Buffers:      &outBuf[0],
	}

	ret := c.Update(targetName, outBufs, inBufs)
	switch ret {
	case sspi.SEC_E_OK:
		return true, int(outBuf[0].BufferSize), nil
	case sspi.SEC_I_COMPLETE_NEEDED, sspi.SEC_I_COMPLETE_AND_CONTINUE:
		ret = sspi.CompleteAuthToken(c.Handle, outBufs)
		if ret != sspi.SEC_E_OK {
			return false, 0, ret
		}
	case sspi.SEC_I_CONTINUE_NEEDED:
	default:
		return false, 0, ret
	}
	return false, int(outBuf[0].BufferSize), nil
}

// kerberosClientContextCBT is a Kerberos client SSPI context that supplies
// LDAP TLS channel bindings on every InitializeSecurityContext call.
type kerberosClientContextCBT struct {
	sctxt      *sspi.Context
	targetName *uint16
	cbt        []byte // SEC_CHANNEL_BINDINGS blob; must stay allocated for handshake
}

func newKerberosClientContextCBT(cred *sspi.Credentials, targetName string, flags uint32, cbt []byte) (*kerberosClientContextCBT, bool, []byte, error) {
	var tname *uint16
	if len(targetName) > 0 {
		p, err := syscall.UTF16FromString(targetName)
		if err != nil {
			return nil, false, nil, err
		}
		if len(p) > 0 {
			tname = &p[0]
		}
	}
	otoken := make([]byte, kerberos.PackageInfo.MaxToken)
	c := sspi.NewClientContext(cred, flags)

	authCompleted, n, err := updateKerberosContextWithCBT(c, otoken, nil, tname, cbt)
	if err != nil {
		_ = c.Release()
		return nil, false, nil, err
	}
	if n == 0 {
		_ = c.Release()
		return nil, false, nil, errors.New("kerberos token should not be empty")
	}
	otoken = otoken[:n]
	return &kerberosClientContextCBT{sctxt: c, targetName: tname, cbt: cbt}, authCompleted, otoken, nil
}

func (c *kerberosClientContextCBT) Update(token []byte) (bool, []byte, error) {
	otoken := make([]byte, kerberos.PackageInfo.MaxToken)
	authDone, n, err := updateKerberosContextWithCBT(c.sctxt, otoken, token, c.targetName, c.cbt)
	if err != nil {
		return false, nil, err
	}
	if n == 0 && !authDone {
		return false, nil, errors.New("kerberos token should not be empty")
	}
	otoken = otoken[:n]
	return authDone, otoken, nil
}

func (c *kerberosClientContextCBT) Release() error {
	if c == nil {
		return nil
	}
	return c.sctxt.Release()
}

func (c *kerberosClientContextCBT) VerifyFlags() error {
	return c.sctxt.VerifyFlags()
}

// sspiEncryptMessage and sspiDecryptMessage mirror github.com/alexbrainman/sspi/internal/common
// (that package is not importable from outside sspi).
func sspiEncryptMessage(c *sspi.Context, msg []byte, qop, seqno uint32) ([]byte, error) {
	_, maxSignature, cBlockSize, cSecurityTrailer, err := c.Sizes()
	if err != nil {
		return nil, err
	}
	if maxSignature == 0 {
		return nil, errors.New("integrity services are not requested or unavailable")
	}
	var b [3]sspi.SecBuffer
	b[0].Set(sspi.SECBUFFER_TOKEN, make([]byte, cSecurityTrailer))
	b[1].Set(sspi.SECBUFFER_DATA, msg)
	b[2].Set(sspi.SECBUFFER_PADDING, make([]byte, cBlockSize))
	ret := sspi.EncryptMessage(c.Handle, qop, sspi.NewSecBufferDesc(b[:]), seqno)
	if ret != sspi.SEC_E_OK {
		return nil, ret
	}
	r0, r1, r2 := b[0].Bytes(), b[1].Bytes(), b[2].Bytes()
	res := make([]byte, 0, len(r0)+len(r1)+len(r2))
	res = append(res, r0...)
	res = append(res, r1...)
	res = append(res, r2...)
	return res, nil
}

func sspiDecryptMessage(c *sspi.Context, msg []byte, seqno uint32) (uint32, []byte, error) {
	var b [2]sspi.SecBuffer
	b[0].Set(sspi.SECBUFFER_STREAM, msg)
	b[1].Set(sspi.SECBUFFER_DATA, []byte{})
	var qop uint32
	ret := sspi.DecryptMessage(c.Handle, sspi.NewSecBufferDesc(b[:]), seqno, &qop)
	if ret != sspi.SEC_E_OK {
		return qop, nil, ret
	}
	return qop, b[1].Bytes(), nil
}

func handshakePayload(secLayer byte, maxSize uint32, authzid []byte) []byte {
	var selectedSecurity byte = secLayer
	var truncatedSize uint32
	if selectedSecurity != 0 {
		// Only 3 bytes available for max size, set to 0x00FFFFFF per RFC 4752.
		truncatedSize = 0b00000000_11111111_11111111_11111111
		if truncatedSize > maxSize {
			truncatedSize = maxSize
		}
	}
	payload := make([]byte, 4, 4+len(authzid))
	binary.BigEndian.PutUint32(payload, truncatedSize)
	payload[0] = selectedSecurity
	payload = append(payload, authzid...)
	return payload
}

// ldapGSSAPIClientCBT implements ldap.GSSAPIClient using SSPI with channel bindings.
// It is used only when the LDAP connection is protected by TLS (LDAPS/StartTLS).
type ldapGSSAPIClientCBT struct {
	creds *sspi.Credentials
	ctx   *kerberosClientContextCBT
	cbt   []byte
}

func newLDAPGSSAPIClientWithCBT(secChannelBindings []byte) (*ldapGSSAPIClientCBT, error) {
	creds, err := kerberos.AcquireCurrentUserCredentials()
	if err != nil {
		return nil, err
	}
	return &ldapGSSAPIClientCBT{creds: creds, cbt: secChannelBindings}, nil
}

func (c *ldapGSSAPIClientCBT) InitSecContext(target string, token []byte) ([]byte, bool, error) {
	sspiFlags := uint32(sspi.ISC_REQ_INTEGRITY | sspi.ISC_REQ_CONFIDENTIALITY | sspi.ISC_REQ_MUTUAL_AUTH)

	switch token {
	case nil:
		ctx, completed, output, err := newKerberosClientContextCBT(c.creds, target, sspiFlags, c.cbt)
		if err != nil {
			return nil, false, err
		}
		c.ctx = ctx
		return output, !completed, nil
	default:
		if c.ctx == nil {
			return nil, false, errors.New("kerberos client context does not exist")
		}
		completed, output, err := c.ctx.Update(token)
		if err != nil {
			return nil, false, err
		}
		if err := c.ctx.VerifyFlags(); err != nil {
			return nil, false, fmt.Errorf("error verifying flags: %v", err)
		}
		return output, !completed, nil
	}
}

func (c *ldapGSSAPIClientCBT) NegotiateSaslAuth(token []byte, authzid string) ([]byte, error) {
	// KERB_WRAP_NO_ENCRYPT (SECQOP_WRAP_NO_ENCRYPT): sign-only, no encryption.
	const kerbWrapNoEncrypt = 0x80000001

	flags, inputPayload, err := sspiDecryptMessage(c.ctx.sctxt, token, 0)
	if err != nil {
		return nil, fmt.Errorf("error decrypting message: %w", err)
	}
	if flags&kerbWrapNoEncrypt == 0 {
		return nil, errors.New("message encrypted")
	}

	if len(inputPayload) != 4 {
		return nil, errors.New("bad server token")
	}
	if inputPayload[0] == 0x0 && !bytes.Equal(inputPayload, []byte{0x0, 0x0, 0x0, 0x0}) {
		return nil, errors.New("bad server token")
	}

	selectedSec := 0
	var maxSecMsgSize uint32
	if selectedSec != 0 {
		maxSecMsgSize, _, _, _, err = c.ctx.sctxt.Sizes()
		if err != nil {
			return nil, fmt.Errorf("error getting security context max message size: %w", err)
		}
	}

	inputPayload, err = sspiEncryptMessage(c.ctx.sctxt, handshakePayload(byte(selectedSec), maxSecMsgSize, []byte(authzid)), kerbWrapNoEncrypt, 0)
	if err != nil {
		return nil, fmt.Errorf("error encrypting message: %w", err)
	}

	return inputPayload, nil
}

func (c *ldapGSSAPIClientCBT) DeleteSecContext() error {
	var err error
	if c.ctx != nil {
		err = c.ctx.Release()
		c.ctx = nil
	}
	if c.creds != nil {
		if releaseErr := c.creds.Release(); releaseErr != nil && err == nil {
			err = releaseErr
		}
		c.creds = nil
	}
	return err
}

// newGSSAPIClientForConn returns an SSPI GSSAPI client. When the LDAP connection
// uses TLS (LDAPS or StartTLS), it supplies LDAP channel bindings so SSPI binds
// succeed against domain controllers that enforce channel binding.
// newGSSAPIClientForConn returns a GSSAPI client for LDAP binds.
// - If TLS is active, it includes CBT so the bind succeeds when AD enforces it.
// - If TLS is not active, it falls back to the standard SSPI client.
func newGSSAPIClientForConn(log *logp.Logger, conn *ldap.Conn) (ldap.GSSAPIClient, error) {
	tlsState, ok := conn.TLSConnectionState()
	if ok && tlsState.HandshakeComplete {
		if len(tlsState.PeerCertificates) > 0 {
			cert := tlsState.PeerCertificates[0]
			log.Debugw("LDAP TLS detected for SSPI bind, preparing CBT",
				"tls_version", tls.VersionName(tlsState.Version),
				"cert_subject", cert.Subject.String(),
				"cert_sig_alg", cert.SignatureAlgorithm.String())
		} else {
			log.Debugw("LDAP TLS detected for SSPI bind, preparing CBT",
				"tls_version", tls.VersionName(tlsState.Version),
				"cert_subject", "none")
		}

		app, err := tlsServerEndpointChannelBindingData(&tlsState)
		if err != nil {
			log.Warnw("LDAP TLS channel binding unavailable, attempting SSPI bind without CBT (may fail if the DC requires channel binding)",
				"error", err)
			return gssapi.NewSSPIClient()
		}
		log.Debugw("using SSPI client with LDAP TLS channel binding (tls-server-end-point)",
			"cbt_app_data_len", len(app),
			"cbt_hash_len", len(app)-len(tlsServerEndPointPrefix))
		return newLDAPGSSAPIClientWithCBT(marshalSecChannelBindings(app))
	}
	return gssapi.NewSSPIClient()
}
