package gomsr

import (
	"encoding/binary"
	"fmt"
	"syscall"
)

//Write writes a given value to the provided register
func (d MSRDev) Write(regno int64, value uint64) error {

	buf := make([]byte, 8)

	binary.LittleEndian.PutUint64(buf, value)

	count, err := syscall.Pwrite(d.fd, buf, regno)
	if err != nil {
		return err
	}
	if count != 8 {
		return fmt.Errorf("Write count not a uint64: %d", count)
	}

	return nil
}

//WriteMSRWithLocation is like WriteMSR(), but takes a custom location, for use with testing or 3rd party utilities like llnl/msr-safe
//It takes a string that has a `%d` format specifier for the cpu. For example: /dev/cpu/%d/msr_safe
func WriteMSRWithLocation(cpu int, msr int64, value uint64, fmtStr string) error {

	m, err := MSRWithLocation(cpu, fmtStr)
	if err != nil {
		return err
	}

	err = m.Write(msr, value)
	if err != nil {
		return err
	}

	return m.Close()

}

//WriteMSR writes the MSR on the given CPU as a one-time operation
func WriteMSR(cpu int, msr int64, value uint64) error {
	m, err := MSR(cpu)
	if err != nil {
		return err
	}

	err = m.Write(msr, value)
	if err != nil {
		return err
	}

	return m.Close()

}
