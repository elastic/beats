// +build !cgo !linux

package util

func GetClockTicks() int {
	return 100
}
