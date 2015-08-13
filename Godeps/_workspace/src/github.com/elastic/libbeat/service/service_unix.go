// +build !windows

package service

// On non-windows platforms, this function does nothing.
func ProcessWindowsControlEvents(stopCallback func()) {
}
