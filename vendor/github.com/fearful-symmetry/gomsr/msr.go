package gomsr

import (
	"fmt"
	"syscall"
)

const defaultFmtStr = "/dev/cpu/%d/msr"

//MSRDev represents a handler for frequent read/write operations
//for one-off MSR read/writes, gomsr provides {Read,Write}MSR*() functions
type MSRDev struct {
	fd int
}

//Close closes the connection to the MSR
func (d MSRDev) Close() error {
	return syscall.Close(d.fd)
}

//MSR provides an interface for reoccurring access to a given CPU's MSR interface
func MSR(cpu int) (MSRDev, error) {
	cpuDir := fmt.Sprintf(defaultFmtStr, cpu)
	fd, err := syscall.Open(cpuDir, syscall.O_RDWR, 777)
	if err != nil {
		return MSRDev{}, err
	}
	return MSRDev{fd: fd}, nil
}

//MSRWithLocation is the same as MSR(), but takes a custom location, for use with testing or 3rd party utilities like llnl/msr-safe
//It takes a string that has a `%d` format specifier for the cpu. For example: /dev/cpu/%d/msr_safe
func MSRWithLocation(cpu int, fmtString string) (MSRDev, error) {
	cpuDir := fmt.Sprintf(fmtString, cpu)
	fd, err := syscall.Open(cpuDir, syscall.O_RDWR, 777)
	if err != nil {
		return MSRDev{}, err
	}
	return MSRDev{fd: fd}, nil
}
