//+build linux,!go1.8 darwin,!go1.10 !linux,!darwin !cgo

package plugin

import "errors"

var errNotSupported = errors.New("plugins not supported")

func loadPlugins(path string) error {
	return errNotSupported
}
