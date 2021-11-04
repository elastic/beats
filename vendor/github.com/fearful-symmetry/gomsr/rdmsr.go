package gomsr

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

//Read reads a given MSR on the CPU and returns the uint64
func (d MSRDev) Read(msr int64) (uint64, error) {
	regBuf := make([]byte, 8)

	rc, err := syscall.Pread(d.fd, regBuf, msr)

	if err != nil {
		return 0, err
	}

	if rc != 8 {
		return 0, fmt.Errorf("Read wrong count of bytes: %d", rc)
	}

	//I'm gonna go ahead and assume an x86 processor will be little endian
	msrOut := binary.LittleEndian.Uint64(regBuf)

	return msrOut, nil
}

//ReadMSRWithLocation is like ReadMSR(), but takes a custom location, for use with testing or 3rd party utilities like llnl/msr-safe
//It takes a string that has a `%d` format specifier for the cpu. For example: /dev/cpu/%d/msr_safe
func ReadMSRWithLocation(cpu int, msr int64, fmtStr string) (uint64, error) {

	m, err := MSRWithLocation(cpu, fmtStr)
	if err != nil {
		return 0, err
	}

	msrD, err := m.Read(msr)
	if err != nil {
		return 0, err
	}

	return msrD, m.Close()

}

//ReadMSR reads the MSR on the given CPU as a one-time operation
func ReadMSR(cpu int, msr int64) (uint64, error) {
	m, err := MSR(cpu)
	if err != nil {
		return 0, err
	}

	msrD, err := m.Read(msr)
	if err != nil {
		return 0, err
	}

	return msrD, m.Close()

}
