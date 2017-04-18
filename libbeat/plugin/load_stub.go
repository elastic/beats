//+build !linux !go1.8 !cgo

package plugin

import "errors"

var errNotSupported = errors.New("plugins not supported")

func loadPlugins(path string) error {
	return errNotSupported
}
