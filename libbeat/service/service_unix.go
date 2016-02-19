// +build !windows

package service

// ProcessWindowsControlEvents is not used on non-windows platforms.
func ProcessWindowsControlEvents(stopCallback func()) {
}
