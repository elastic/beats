// +build debug

package wasmtime

// See `ffi.go` documentation about `ptr()` for what's going on here.

import "runtime"

func maybeGC() {
	runtime.GC()
}
