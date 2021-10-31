package rpm

import (
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"io"
	"os"
)

// KeyRing reads a openpgp.KeyRing from the given io.Reader which may then be
// used to validate GPG keys in RPM packages.
func KeyRing(r io.Reader) (openpgp.KeyRing, error) {
	// decode gpgkey file
	p, err := armor.Decode(r)
	if err != nil {
		return nil, err
	}

	// extract keys
	return openpgp.ReadKeyRing(p.Body)
}

// KeyRingFromFile reads a openpgp.KeyRing from the given file path which may
// then be used to validate GPG keys in RPM packages.
func KeyRingFromFile(path string) (openpgp.KeyRing, error) {
	// open gpgkey file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// read keyring
	return KeyRing(f)
}

// KeyRingFromFiles reads a openpgp.KeyRing from the given file paths which may
// then be used to validate GPG keys in RPM packages.
//
// This function might typically be used to read all keys in /etc/pki/rpm-gpg.
func KeyRingFromFiles(files []string) (openpgp.KeyRing, error) {
	keyring := make(openpgp.EntityList, 0)
	for _, path := range files {
		// read keyring in file
		el, err := KeyRingFromFile(path)
		if err != nil {
			return nil, err
		}

		// append keyring
		keyring = append(keyring, el.(openpgp.EntityList)...)
	}

	return keyring, nil
}
