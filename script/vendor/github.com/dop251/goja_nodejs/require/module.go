package require

import (
	js "github.com/dop251/goja"

	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
)

type ModuleLoader func(*js.Runtime, *js.Object)
type SourceLoader func(path string) ([]byte, error)

var (
	InvalidModuleError     = errors.New("Invalid module")
	IllegalModuleNameError = errors.New("Illegal module name")
)

var native map[string]ModuleLoader

// Registry contains a cache of compiled modules which can be used by multiple Runtimes
type Registry struct {
	sync.Mutex
	compiled map[string]*js.Program

	srcLoader SourceLoader
}

type RequireModule struct {
	r       *Registry
	runtime *js.Runtime
	modules map[string]*js.Object
}

func NewRegistryWithLoader(srcLoader SourceLoader) *Registry {
	return &Registry{
		srcLoader: srcLoader,
	}
}

// Enable adds the require() function to the specified runtime.
func (r *Registry) Enable(runtime *js.Runtime) *RequireModule {
	rrt := &RequireModule{
		r:       r,
		runtime: runtime,
		modules: make(map[string]*js.Object),
	}

	runtime.Set("require", rrt.require)
	return rrt
}

func (r *Registry) getCompiledSource(p string) (prg *js.Program, err error) {
	r.Lock()
	defer r.Unlock()

	prg = r.compiled[p]
	if prg == nil {
		srcLoader := r.srcLoader
		if srcLoader == nil {
			srcLoader = ioutil.ReadFile
		}
		if s, err1 := srcLoader(p); err1 == nil {
			source := "(function(module, exports) {" + string(s) + "\n})"
			prg, err = js.Compile(p, source, false)
			if err == nil {
				if r.compiled == nil {
					r.compiled = make(map[string]*js.Program)
				}
				r.compiled[p] = prg
			}
		} else {
			err = err1
		}
	}
	return
}

func (r *RequireModule) loadModule(path string, jsModule *js.Object) error {

	if ldr, exists := native[path]; exists {
		ldr(r.runtime, jsModule)
		return nil
	}

	prg, err := r.r.getCompiledSource(path)

	if err != nil {
		return err
	}

	f, err := r.runtime.RunProgram(prg)
	if err != nil {
		return err
	}

	if call, ok := js.AssertFunction(f); ok {
		jsExports := jsModule.Get("exports")

		// Run the module source, with "jsModule" as the "module" variable, "jsExports" as "this"(Nodejs capable).
		_, err = call(jsExports, jsModule, jsExports)
		if err != nil {
			return err
		}
	} else {
		return InvalidModuleError
	}

	return nil
}

func (r *RequireModule) require(call js.FunctionCall) js.Value {
	ret, err := r.Require(call.Argument(0).String())
	if err != nil {
		panic(r.runtime.NewGoError(err))
	}
	return ret
}

// Require can be used to import modules from Go source (similar to JS require() function).
func (r *RequireModule) Require(p string) (ret js.Value, err error) {
	p = filepath.Clean(p)
	if p == "" {
		err = IllegalModuleNameError
		return
	}
	module := r.modules[p]
	if module == nil {
		module = r.runtime.NewObject()
		module.Set("exports", r.runtime.NewObject())
		r.modules[p] = module
		err = r.loadModule(p, module)
		if err != nil {
			delete(r.modules, p)
			err = fmt.Errorf("Could not load module '%s': %v", p, err)
			return
		}
	}
	ret = module.Get("exports")
	return
}

func Require(runtime *js.Runtime, name string) js.Value {
	if r, ok := js.AssertFunction(runtime.Get("require")); ok {
		mod, err := r(js.Undefined(), runtime.ToValue(name))
		if err != nil {
			panic(err)
		}
		return mod
	}
	panic(runtime.NewTypeError("Please enable require for this runtime using new(require.Require).Enable(runtime)"))
}

func RegisterNativeModule(name string, loader ModuleLoader) {
	if native == nil {
		native = make(map[string]ModuleLoader)
	}
	native[name] = loader
}
