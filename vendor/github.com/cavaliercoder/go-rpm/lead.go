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

const (
	r_LeadLength = 96
)

var (
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
	var buf [r_LeadLength]byte
	_, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
	}
	if 0 != bytes.Compare(buf[:4], []byte{0xED, 0xAB, 0xEE, 0xDB}) {
		return nil, ErrNotRPMFile
	}
	lead := &Lead{
		VersionMajor:    int(buf[4]),
		VersionMinor:    int(buf[5]),
		Type:            int(binary.BigEndian.Uint16(buf[6:8])),
		Architecture:    int(binary.BigEndian.Uint16(buf[8:10])),
		Name:            string(buf[10:76]),
		OperatingSystem: int(binary.BigEndian.Uint16(buf[76:78])),
		SignatureType:   int(binary.BigEndian.Uint16(buf[78:80])),
	}
	if lead.VersionMajor < 3 || lead.VersionMajor > 4 {
		return nil, ErrUnsupportedVersion
	}

	// TODO: validate lead value ranges

	return lead, nil
}
