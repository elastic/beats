package wasmtime

// #include <wasm.h>
import "C"
import (
	"runtime"
)

// Engine is an instance of a wasmtime engine which is used to create a `Store`.
//
// Engines are a form of global configuration for wasm compilations and modules
// and such.
type Engine struct {
	_ptr *C.wasm_engine_t
}

// NewEngine creates a new `Engine` with default configuration.
func NewEngine() *Engine {
	engine := &Engine{_ptr: C.wasm_engine_new()}
	runtime.SetFinalizer(engine, func(engine *Engine) {
		C.wasm_engine_delete(engine._ptr)
	})
	return engine
}

// NewEngineWithConfig creates a new `Engine` with the `Config` provided
//
// Note that once a `Config` is passed to this method it cannot be used again.
func NewEngineWithConfig(config *Config) *Engine {
	if config.ptr() == nil {
		panic("config already used")
	}
	engine := &Engine{_ptr: C.wasm_engine_new_with_config(config.ptr())}
	runtime.SetFinalizer(config, nil)
	config._ptr = nil
	runtime.SetFinalizer(engine, func(engine *Engine) {
		C.wasm_engine_delete(engine._ptr)
	})
	return engine
}

func (engine *Engine) ptr() *C.wasm_engine_t {
	ret := engine._ptr
	maybeGC()
	return ret
}
