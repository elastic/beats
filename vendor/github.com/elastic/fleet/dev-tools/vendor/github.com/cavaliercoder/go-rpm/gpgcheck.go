package rpm

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/errors"
	"io"
)

// Signature storage types (as defined in package lead segment)
const (
	// Signature is stored in the first package header
	RPMSIGTYPE_HEADERSIG = 5
)

// Predefined checksum errors.
var (
	// ErrMD5ValidationFailed indicates that a RPM package failed checksum
	// validation.
	ErrMD5ValidationFailed = fmt.Errorf("MD5 checksum validation failed")

	// ErrGPGValidationFailed indicates that a RPM package failed GPG signature
	// validation.
	ErrGPGValidationFailed = fmt.Errorf("GPG signature validation failed")
)

// rpmReadSigHeader reads the lead and signature header of a rpm package and
// returns a pointer to the signature header.
func rpmReadSigHeader(r io.Reader) (*Header, error) {
	// read package lead
	if lead, err := ReadPackageLead(r); err != nil {
		return nil, err
	} else {
		// check signature type
		if lead.SignatureType != RPMSIGTYPE_HEADERSIG {
			return nil, fmt.Errorf("Unsupported signature type: 0x%x", lead.SignatureType)
		}
	}

	// read signature header
	sigheader, err := ReadPackageHeader(r)
	if err != nil {
		return nil, err
	}

	return sigheader, nil
}

// GPGCheck validates the integrity of a RPM package file read from the given
// io.Reader. Public keys in the given keyring are used to validate the package
// signature.
//
// If validation succeeds, nil is returned. If validation fails,
// ErrGPGValidationFailed is returned.
//
// This function is an expensive operation which reads the entire package file.
func GPGCheck(r io.Reader, keyring openpgp.KeyRing) (string, error) {
	// read signature header
	sigheader, err := rpmReadSigHeader(r)
	if err != nil {
		return "", err
	}

	// get signature bytes
	var sigval []byte = nil
	for _, tag := range []int{RPMSIGTAG_PGP, RPMSIGTAG_PGP5, RPMSIGTAG_GPG} {
		if sigval = sigheader.Indexes.BytesByTag(tag); sigval != nil {
			break
		}
	}

	if sigval == nil {
		return "", fmt.Errorf("Package signature not found")
	}

	// check signature
	signer, err := openpgp.CheckDetachedSignature(keyring, r, bytes.NewReader(sigval))
	if err == errors.ErrUnknownIssuer {
		return "", ErrGPGValidationFailed
	} else if err != nil {
		return "", err
	}

	// get signer identity
	for id, _ := range signer.Identities {
		return id, nil
	}

	return "", fmt.Errorf("No identity found in public key")
}

// MD5Check validates the integrity of a RPM package file read from the given
// io.Reader. An MD5 checksum is computed for the package payload and compared
// with the checksum value specified in the package header.
//
// If validation succeeds, nil is returned. If validation fails,
// ErrMD5ValidationFailed is returned.
//
// This function is an expensive operation which reads the entire package file.
func MD5Check(r io.Reader) error {
	// read signature header
	sigheader, err := rpmReadSigHeader(r)
	if err != nil {
		return err
	}

	// get expected payload size
	payloadSize := sigheader.Indexes.IntByTag(RPMSIGTAG_SIZE)
	if payloadSize == 0 {
		return fmt.Errorf("RPMSIGTAG_SIZE tag not found in signature header")
	}

	// get expected payload md5 sum
	sigmd5 := sigheader.Indexes.BytesByTag(RPMSIGTAG_MD5)
	if sigmd5 == nil {
		return fmt.Errorf("RPMSIGTAG_MD5 tag not found in signature header")
	}

	// compute payload sum
	h := md5.New()
	if n, err := io.Copy(h, r); err != nil {
		return fmt.Errorf("Error reading payload: %v", err)
	} else if n != payloadSize {
		return ErrMD5ValidationFailed
	}

	// compare sums
	payloadmd5 := h.Sum(nil)
	if !bytes.Equal(payloadmd5, sigmd5) {
		return ErrMD5ValidationFailed
	}

	return nil
}
