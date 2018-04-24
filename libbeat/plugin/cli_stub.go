//+build linux,!go1.8 darwin,!go1.10 !linux,!darwin !cgo

package plugin

func Initialize() error {
	return nil
}
