// +build freebsd,!cgo openbsd,!cgo

package numcpu

// getCPU is the fallback for unimplemented platforms
func getCPU() (int, bool, error) {

	return -1, false, nil
}
