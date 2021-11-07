package wasmtime

// #include "shims.h"
import "C"
import (
	"errors"
	"reflect"
	"runtime"
	"unsafe"
)

// Func is a function instance, which is the runtime representation of a function.
// It effectively is a closure of the original function over the runtime module instance of its originating module.
// The module instance is used to resolve references to other definitions during execution of the function.
// Read more in [spec](https://webassembly.github.io/spec/core/exec/runtime.html#function-instances)
type Func struct {
	val C.wasmtime_func_t
}

// Caller is provided to host-defined functions when they're invoked from
// WebAssembly.
//
// A `Caller` can be used for `Storelike` arguments to allow recursive execution
// or creation of wasm objects. Additionally `Caller` can be used to learn about
// the exports of the calling instance.
type Caller struct {
	// Note that unlike other structures in these bindings this is named `ptr`
	// instead of `_ptr` because no finalizer is configured with `Caller` so it's
	// ok to access this raw value.
	ptr *C.wasmtime_caller_t
}

// NewFunc creates a new `Func` with the given `ty` which, when called, will call `f`
//
// The `ty` given is the wasm type signature of the `Func` to create. When called
// the `f` callback receives two arguments. The first is a `Caller` to learn
// information about the calling context and the second is a list of arguments
// represented as a `Val`. The parameters are guaranteed to match the parameters
// types specified in `ty`.
//
// The `f` callback is expected to produce one of two values. Results can be
// returned as an array of `[]Val`. The number and types of these results much
// match the `ty` given, otherwise the program will panic. The `f` callback can
// also produce a trap which will trigger trap unwinding in wasm, and the trap
// will be returned to the original caller.
//
// If the `f` callback panics then the panic will be propagated to the caller
// as well.
func NewFunc(
	store Storelike,
	ty *FuncType,
	f func(*Caller, []Val) ([]Val, *Trap),
) *Func {
	idx := insertFuncNew(getDataInStore(store), ty, f)

	ret := C.wasmtime_func_t{}
	C.go_func_new(
		store.Context(),
		ty.ptr(),
		C.size_t(idx),
		0,
		&ret,
	)
	runtime.KeepAlive(store)
	runtime.KeepAlive(ty)

	return mkFunc(ret)
}

//export goTrampolineNew
func goTrampolineNew(
	callerPtr *C.wasmtime_caller_t,
	env C.size_t,
	argsPtr *C.wasmtime_val_t,
	argsNum C.size_t,
	resultsPtr *C.wasmtime_val_t,
	resultsNum C.size_t,
) *C.wasm_trap_t {
	caller := &Caller{ptr: callerPtr}
	defer func() { caller.ptr = nil }()
	data := getDataInStore(caller)
	entry := data.getFuncNew(int(env))

	params := make([]Val, int(argsNum))
	var val C.wasmtime_val_t
	base := unsafe.Pointer(argsPtr)
	for i := 0; i < len(params); i++ {
		ptr := (*C.wasmtime_val_t)(unsafe.Pointer(uintptr(base) + uintptr(i)*unsafe.Sizeof(val)))
		params[i] = mkVal(ptr)
	}

	var results []Val
	var trap *Trap
	var lastPanic interface{}
	func() {
		defer func() { lastPanic = recover() }()
		results, trap = entry.callback(caller, params)
		if trap != nil {
			if trap._ptr == nil {
				panic("returned an already-returned trap")
			}
			return
		}
		if len(results) != len(entry.results) {
			panic("callback didn't produce the correct number of results")
		}
		for i, ty := range entry.results {
			if results[i].Kind() != ty.Kind() {
				panic("callback produced wrong type of result")
			}
		}
	}()
	if trap == nil && lastPanic != nil {
		data.lastPanic = lastPanic
		return nil
	}
	if trap != nil {
		runtime.SetFinalizer(trap, nil)
		ret := trap.ptr()
		trap._ptr = nil
		return ret
	}

	base = unsafe.Pointer(resultsPtr)
	for i := 0; i < len(results); i++ {
		ptr := (*C.wasmtime_val_t)(unsafe.Pointer(uintptr(base) + uintptr(i)*unsafe.Sizeof(val)))
		C.wasmtime_val_copy(ptr, results[i].ptr())
	}
	runtime.KeepAlive(results)
	return nil
}

// WrapFunc wraps a native Go function, `f`, as a wasm `Func`.
//
// This function differs from `NewFunc` in that it will determine the type
// signature of the wasm function given the input value `f`. The `f` value
// provided must be a Go function. It may take any number of the following
// types as arguments:
//
// `int32` - a wasm `i32`
//
// `int64` - a wasm `i64`
//
// `float32` - a wasm `f32`
//
// `float64` - a wasm `f64`
//
// `*Caller` - information about the caller's instance
//
// `*Func` - a wasm `funcref`
//
// anything else - a wasm `externref`
//
// The Go function may return any number of values. It can return any number of
// primitive wasm values (integers/floats), and the last return value may
// optionally be `*Trap`. If a `*Trap` returned is `nil` then the other values
// are returned from the wasm function. Otherwise the `*Trap` is returned and
// it's considered as if the host function trapped.
//
// If the function `f` panics then the panic will be propagated to the caller.
func WrapFunc(
	store Storelike,
	f interface{},
) *Func {
	val := reflect.ValueOf(f)
	wasmTy := inferFuncType(val)
	idx := insertFuncWrap(getDataInStore(store), val)

	ret := C.wasmtime_func_t{}
	C.go_func_new(
		store.Context(),
		wasmTy.ptr(),
		C.size_t(idx),
		1, // this is `WrapFunc`, not `NewFunc`
		&ret,
	)
	runtime.KeepAlive(store)
	runtime.KeepAlive(wasmTy)
	return mkFunc(ret)
}

func inferFuncType(val reflect.Value) *FuncType {
	// Make sure the `interface{}` passed in was indeed a function
	ty := val.Type()
	if ty.Kind() != reflect.Func {
		panic("callback provided must be a `func`")
	}

	// infer the parameter types, and `*Caller` type is special in the
	// parameters so be sure to case on that as well.
	params := make([]*ValType, 0, ty.NumIn())
	var caller *Caller
	for i := 0; i < ty.NumIn(); i++ {
		paramTy := ty.In(i)
		if paramTy != reflect.TypeOf(caller) {
			params = append(params, typeToValType(paramTy))
		}
	}

	// Then infer the result types, where a final `*Trap` result value is
	// also special.
	results := make([]*ValType, 0, ty.NumOut())
	var trap *Trap
	for i := 0; i < ty.NumOut(); i++ {
		resultTy := ty.Out(i)
		if i == ty.NumOut()-1 && resultTy == reflect.TypeOf(trap) {
			continue
		}
		results = append(results, typeToValType(resultTy))
	}
	return NewFuncType(params, results)
}

func typeToValType(ty reflect.Type) *ValType {
	var a int32
	if ty == reflect.TypeOf(a) {
		return NewValType(KindI32)
	}
	var b int64
	if ty == reflect.TypeOf(b) {
		return NewValType(KindI64)
	}
	var c float32
	if ty == reflect.TypeOf(c) {
		return NewValType(KindF32)
	}
	var d float64
	if ty == reflect.TypeOf(d) {
		return NewValType(KindF64)
	}
	var f *Func
	if ty == reflect.TypeOf(f) {
		return NewValType(KindFuncref)
	}
	return NewValType(KindExternref)
}

//export goTrampolineWrap
func goTrampolineWrap(
	callerPtr *C.wasmtime_caller_t,
	env C.size_t,
	argsPtr *C.wasmtime_val_t,
	argsNum C.size_t,
	resultsPtr *C.wasmtime_val_t,
	resultsNum C.size_t,
) *C.wasm_trap_t {
	// Convert all our parameters to `[]reflect.Value`, taking special care
	// for `*Caller` but otherwise reading everything through `Val`.
	caller := &Caller{ptr: callerPtr}
	defer func() { caller.ptr = nil }()
	data := getDataInStore(caller)
	entry := data.getFuncWrap(int(env))

	ty := entry.callback.Type()
	params := make([]reflect.Value, ty.NumIn())
	base := unsafe.Pointer(argsPtr)
	var raw C.wasmtime_val_t
	for i := 0; i < len(params); i++ {
		if ty.In(i) == reflect.TypeOf(caller) {
			params[i] = reflect.ValueOf(caller)
		} else {
			ptr := (*C.wasmtime_val_t)(base)
			val := mkVal(ptr)
			params[i] = reflect.ValueOf(val.Get())
			base = unsafe.Pointer(uintptr(base) + unsafe.Sizeof(raw))
		}
	}

	// Invoke the function, catching any panics to propagate later. Panics
	// result in immediately returning a trap.
	var results []reflect.Value
	var lastPanic interface{}
	func() {
		defer func() { lastPanic = recover() }()
		results = entry.callback.Call(params)
	}()
	if lastPanic != nil {
		data.lastPanic = lastPanic
		return nil
	}

	// And now we write all the results into memory depending on the type
	// of value that was returned.
	base = unsafe.Pointer(resultsPtr)
	for _, result := range results {
		ptr := (*C.wasmtime_val_t)(base)
		switch val := result.Interface().(type) {
		case int32:
			*ptr = *ValI32(val).ptr()
		case int64:
			*ptr = *ValI64(val).ptr()
		case float32:
			*ptr = *ValF32(val).ptr()
		case float64:
			*ptr = *ValF64(val).ptr()
		case *Func:
			*ptr = *ValFuncref(val).ptr()
		case *Trap:
			if val != nil {
				runtime.SetFinalizer(val, nil)
				ret := val._ptr
				val._ptr = nil
				if ret == nil {
					data.lastPanic = "cannot return trap twice"
					return nil
				} else {
					return ret
				}
			}
		default:
			raw := ValExternref(val)
			C.wasmtime_val_copy(ptr, raw.ptr())
			runtime.KeepAlive(raw)
		}
		base = unsafe.Pointer(uintptr(base) + unsafe.Sizeof(raw))
	}
	return nil
}

func mkFunc(val C.wasmtime_func_t) *Func {
	return &Func{val}
}

// Type returns the type of this func
func (f *Func) Type(store Storelike) *FuncType {
	ptr := C.wasmtime_func_type(store.Context(), &f.val)
	runtime.KeepAlive(store)
	return mkFuncType(ptr, nil)
}

// Call invokes this function with the provided `args`.
//
// This variadic function must be invoked with the correct number and type of
// `args` as specified by the type of this function. This property is checked
// at runtime. Each `args` may have one of the following types:
//
// `int32` - a wasm `i32`
//
// `int64` - a wasm `i64`
//
// `float32` - a wasm `f32`
//
// `float64` - a wasm `f64`
//
// `Val` - correspond to a wasm value
//
// `*Func` - a wasm `funcref`
//
// anything else - a wasm `externref`
//
// This function will have one of three results:
//
// 1. If the function returns successfully, then the `interface{}` return
// argument will be the result of the function. If there were 0 results then
// this value is `nil`. If there was one result then this is that result.
// Otherwise if there were multiple results then `[]Val` is returned.
//
// 2. If this function invocation traps, then the returned `interface{}` value
// will be `nil` and a non-`nil` `*Trap` will be returned with information
// about the trap that happened.
//
// 3. If a panic in Go ends up happening somewhere, then this function will
// panic.
func (f *Func) Call(store Storelike, args ...interface{}) (interface{}, error) {
	ty := f.Type(store)
	params := ty.Params()
	if len(args) > len(params) {
		return nil, errors.New("too many arguments provided")
	}
	paramVals := make([]C.wasmtime_val_t, len(args))
	var externrefs []Val
	for i, param := range args {
		dst := &paramVals[i]
		switch val := param.(type) {
		case int:
			switch params[i].Kind() {
			case KindI32:
				dst.kind = C.WASMTIME_I32
				C.go_wasmtime_val_i32_set(dst, C.int32_t(val))
			case KindI64:
				dst.kind = C.WASMTIME_I64
				C.go_wasmtime_val_i64_set(dst, C.int64_t(val))
			default:
				return nil, errors.New("integer provided for non-integer argument")
			}
		case int32:
			dst.kind = C.WASMTIME_I32
			C.go_wasmtime_val_i32_set(dst, C.int32_t(val))
		case int64:
			dst.kind = C.WASMTIME_I64
			C.go_wasmtime_val_i64_set(dst, C.int64_t(val))
		case float32:
			dst.kind = C.WASMTIME_F32
			C.go_wasmtime_val_f32_set(dst, C.float(val))
		case float64:
			dst.kind = C.WASMTIME_F64
			C.go_wasmtime_val_f64_set(dst, C.double(val))
		case *Func:
			dst.kind = C.WASMTIME_FUNCREF
			C.go_wasmtime_val_funcref_set(dst, val.val)
		case Val:
			*dst = *val.ptr()

		default:
			externref := ValExternref(val)
			externrefs = append(externrefs, externref)
			*dst = *externref.ptr()
		}

	}

	resultVals := make([]C.wasmtime_val_t, len(ty.Results()))

	err := enterWasm(store, func(trap **C.wasm_trap_t) *C.wasmtime_error_t {
		var paramsPtr *C.wasmtime_val_t
		if len(paramVals) > 0 {
			paramsPtr = (*C.wasmtime_val_t)(unsafe.Pointer(&paramVals[0]))
		}
		var resultsPtr *C.wasmtime_val_t
		if len(resultVals) > 0 {
			resultsPtr = (*C.wasmtime_val_t)(unsafe.Pointer(&resultVals[0]))
		}
		return C.wasmtime_func_call(
			store.Context(),
			&f.val,
			paramsPtr,
			C.size_t(len(paramVals)),
			resultsPtr,
			C.size_t(len(resultVals)),
			trap,
		)
	})
	runtime.KeepAlive(store)
	runtime.KeepAlive(args)
	runtime.KeepAlive(resultVals)
	runtime.KeepAlive(paramVals)
	runtime.KeepAlive(externrefs)

	if err != nil {
		return nil, err
	}

	if len(resultVals) == 0 {
		return nil, nil
	} else if len(resultVals) == 1 {
		ret := takeVal(&resultVals[0]).Get()
		return ret, nil
	} else {
		results := make([]Val, len(resultVals))
		for i := 0; i < len(results); i++ {
			results[i] = takeVal(&resultVals[i])
		}
		return results, nil
	}

}

// Implementation of the `AsExtern` interface for `Func`
func (f *Func) AsExtern() C.wasmtime_extern_t {
	ret := C.wasmtime_extern_t{kind: C.WASMTIME_EXTERN_FUNC}
	C.go_wasmtime_extern_func_set(&ret, f.val)
	return ret
}

// GetExport gets an exported item from the caller's module.
//
// May return `nil` if the export doesn't, if it's not a memory, if there isn't
// a caller, etc.
func (c *Caller) GetExport(name string) *Extern {
	if c.ptr == nil {
		return nil
	}
	var ret C.wasmtime_extern_t
	ok := C.wasmtime_caller_export_get(
		c.ptr,
		C._GoStringPtr(name),
		C._GoStringLen(name),
		&ret,
	)
	runtime.KeepAlive(name)
	runtime.KeepAlive(c)
	if ok {
		return mkExtern(&ret)
	}
	return nil

}

// Implementation of the `Storelike` interface for `Caller`.
func (c *Caller) Context() *C.wasmtime_context_t {
	if c.ptr == nil {
		panic("cannot use caller after host function returns")
	}
	return C.wasmtime_caller_context(c.ptr)
}

// Shim function that's expected to wrap any invocations of WebAssembly from Go
// itself.
//
// This is used to handle traps and error returns from any invocation of
// WebAssembly. This will also automatically propagate panics that happen within
// Go from one end back to this original invocation point.
//
// The `store` object is the context being used for the invocation, and `wasm`
// is the closure which will internally execute WebAssembly. A trap pointer is
// provided to the closure and it's expected that the closure returns an error.
func enterWasm(store Storelike, wasm func(**C.wasm_trap_t) *C.wasmtime_error_t) error {
	// Load the internal `storeData` that our `store` references, which is
	// used for handling panics which we are going to use here.
	data := getDataInStore(store)

	var trap *C.wasm_trap_t
	err := wasm(&trap)

	// Take ownership of any returned values to ensure we properly run
	// destructors for them.
	var wrappedTrap *Trap
	var wrappedError error
	if trap != nil {
		wrappedTrap = mkTrap(trap)
	}
	if err != nil {
		wrappedError = mkError(err)
	}

	// Check to see if wasm panicked, and if it did then we need to
	// propagate that. Note that this happens after we take ownership of
	// return values to ensure they're cleaned up properly.
	if data.lastPanic != nil {
		lastPanic := data.lastPanic
		data.lastPanic = nil
		panic(lastPanic)
	}

	// If there wasn't a panic then we determine whether to return the trap
	// or the error.
	if wrappedTrap != nil {
		return wrappedTrap
	}
	return wrappedError
}
