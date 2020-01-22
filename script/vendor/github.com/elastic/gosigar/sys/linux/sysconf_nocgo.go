// +build !cgo !linux

package linux

// GetClockTicks returns the number of click ticks in one jiffie.
func GetClockTicks() int {
	return 100
}
