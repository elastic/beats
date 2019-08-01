package rpm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// A Lead is the deprecated lead section of an RPM file which is used in legacy
// RPM versions to store package metadata.
type Lead struct {
	VersionMajor    int
	VersionMinor    int
	Name            string
	Type            int
	Architecture    int
	OperatingSystem int
	SignatureType   int
}

// Predefined lead section errors.
var (
	// ErrBadLeadLength indicates that the read lead section is not the expected
	// length.
	ErrBadLeadLength = fmt.Errorf("RPM lead section is incorrect length")

	// ErrNotRPMFile indicates that the read file does not start with the
	// expected descriptor.
	ErrNotRPMFile = fmt.Errorf("RPM file descriptor is invalid")

	// ErrUnsupportedVersion indicates that the read lead section version is not
	// currently supported.
	ErrUnsupportedVersion = fmt.Errorf("unsupported RPM package version")
)

// ReadPackageLead reads the deprecated lead section of an RPM file which is
// used in legacy RPM versions to store package metadata.
//
// This function should only be used if you intend to read a package lead in
// isolation.
func ReadPackageLead(r io.Reader) (*Lead, error) {
	// read bytes
	b := make([]byte, 96)
	n, err := r.Read(b)
	if err != nil {
		return nil, err
	}

	// check length
	if n != 96 {
		return nil, ErrBadLeadLength
	}

	// check magic number
	if 0 != bytes.Compare(b[:4], []byte{0xED, 0xAB, 0xEE, 0xDB}) {
		return nil, ErrNotRPMFile
	}

	// decode lead
	lead := &Lead{
		VersionMajor:    int(b[4]),
		VersionMinor:    int(b[5]),
		Type:            int(binary.BigEndian.Uint16(b[6:8])),
		Architecture:    int(binary.BigEndian.Uint16(b[8:10])),
		Name:            string(b[10:76]),
		OperatingSystem: int(binary.BigEndian.Uint16(b[76:78])),
		SignatureType:   int(binary.BigEndian.Uint16(b[78:80])),
	}

	// check version
	if lead.VersionMajor < 3 || lead.VersionMajor > 4 {
		return nil, ErrUnsupportedVersion
	}

	// TODO: validate lead value ranges

	return lead, nil
}
