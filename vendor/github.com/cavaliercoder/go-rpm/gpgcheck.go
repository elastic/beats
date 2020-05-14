package rpm

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
)

// see: https://github.com/rpm-software-management/rpm/blob/3b1f4b0c6c9407b08620a5756ce422df10f6bd1a/rpmio/rpmpgp.c#L51
var gpgPubkeyTbl = map[packet.PublicKeyAlgorithm]string{
	packet.PubKeyAlgoRSA:            "RSA",
	packet.PubKeyAlgoRSASignOnly:    "RSA(Sign-Only)",
	packet.PubKeyAlgoRSAEncryptOnly: "RSA(Encrypt-Only)",
	packet.PubKeyAlgoElGamal:        "Elgamal",
	packet.PubKeyAlgoDSA:            "DSA",
	packet.PubKeyAlgoECDH:           "Elliptic Curve",
	packet.PubKeyAlgoECDSA:          "ECDSA",
}

// Map Go hashes to rpm info name
// See: https://golang.org/src/crypto/crypto.go?s=#L23
//      https://github.com/rpm-software-management/rpm/blob/3b1f4b0c6c9407b08620a5756ce422df10f6bd1a/rpmio/rpmpgp.c#L88
var gpgHashTbl = []string{
	"Unknown hash algorithm",
	"MD4",
	"MD5",
	"SHA1",
	"SHA224",
	"SHA256",
	"SHA384",
	"SHA512",
	"MD5SHA1",
	"RIPEMD160",
	"SHA3_224",
	"SHA3_256",
	"SHA3_384",
	"SHA3_512",
	"SHA512_224",
	"SHA512_256",
}

// GPGSignature is the raw byte representation of a package's signature.
type GPGSignature []byte

func (b GPGSignature) String() string {
	pkt, err := packet.Read(bytes.NewReader(b))
	if err != nil {
		return ""
	}

	switch sig := pkt.(type) {
	case *packet.SignatureV3:
		algo, ok := gpgPubkeyTbl[sig.PubKeyAlgo]
		if !ok {
			algo = "Unknown public key algorithm"
		}

		hasher := gpgHashTbl[0]
		if int(sig.Hash) < len(gpgHashTbl) {
			hasher = gpgHashTbl[sig.Hash]
		}

		ctime := sig.CreationTime.UTC().Format(RPMDate)
		return fmt.Sprintf("%v/%v, %v, Key ID %x", algo, hasher, ctime, sig.IssuerKeyId)
	}

	return ""
}

// Predefined checksum errors.
var (
	// ErrMD5ValidationFailed indicates that a RPM package failed checksum
	// validation.
	ErrMD5ValidationFailed = fmt.Errorf("MD5 checksum validation failed")

	// ErrGPGValidationFailed indicates that a RPM package failed GPG signature
	// validation.
	ErrGPGValidationFailed = fmt.Errorf("GPG signature validation failed")

	// ErrGPGUnknownSignature indicates that the RPM package signature tag is of
	// an unknown type.
	ErrGPGUnknownSignature = fmt.Errorf("unknown signature type")
)

// rpmReadSigHeader reads the lead and signature header of a rpm package and
// returns a pointer to the signature header.
func rpmReadSigHeader(r io.Reader) (*Header, error) {
	// read package lead
	lead, err := ReadPackageLead(r)
	if err != nil {
		return nil, err
	}

	// check signature type
	if lead.SignatureType != 5 { // RPMSIGTYPE_HEADERSIG
		return nil, fmt.Errorf("Unsupported signature type: 0x%x", lead.SignatureType)
	}

	// read signature header
	hdr, err := ReadPackageHeader(r)
	if err != nil {
		return nil, err
	}

	// pad to next header
	if _, err := io.CopyN(ioutil.Discard, r, int64(8-(hdr.Length%8))%8); err != nil {
		return nil, err
	}

	return hdr, nil
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

	tags := []int{
		1002, // RPMSIGTAG_PGP
		1006, // RPMSIGTAG_PGP5
		1005, // RPMSIGTAG_GPG
	}

	// get signature bytes
	var sigval []byte
	for _, tag := range tags {
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
	for id := range signer.Identities {
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
	payloadSize := sigheader.Indexes.IntByTag(1000) // RPMSIGTAG_SIZE
	if payloadSize == 0 {
		return fmt.Errorf("RPMSIGTAG_SIZE tag not found in signature header")
	}

	// get expected payload md5 sum
	sigmd5 := sigheader.Indexes.BytesByTag(1004) // RPMSIGTAG_MD5
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
