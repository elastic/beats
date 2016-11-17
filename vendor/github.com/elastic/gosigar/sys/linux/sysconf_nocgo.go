// +build !cgo !linux

package linux

func GetClockTicks() int {
	return 100
}
