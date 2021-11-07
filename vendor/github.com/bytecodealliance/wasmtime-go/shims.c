#include "_cgo_export.h"
#include "shims.h"

wasmtime_store_t *go_store_new(wasm_engine_t *engine, size_t env) {
  return wasmtime_store_new(engine, (void*) env, goFinalizeStore);
}

static wasm_trap_t* trampoline(
   void *env,
   wasmtime_caller_t *caller,
   const wasmtime_val_t *args,
   size_t nargs,
   wasmtime_val_t *results,
   size_t nresults
) {
    return goTrampolineNew(caller, (size_t) env,
        (wasmtime_val_t*) args, nargs,
        results, nresults);
}

static wasm_trap_t* wrap_trampoline(
   void *env,
   wasmtime_caller_t *caller,
   const wasmtime_val_t *args,
   size_t nargs,
   wasmtime_val_t *results,
   size_t nresults
) {
    return goTrampolineWrap(caller, (size_t) env,
        (wasmtime_val_t*) args, nargs,
        results, nresults);
}

void go_func_new(
    wasmtime_context_t *store,
    wasm_functype_t *ty,
    size_t env,
    int wrap,
    wasmtime_func_t *ret
) {
  wasmtime_func_callback_t callback = trampoline;
  if (wrap)
    callback = wrap_trampoline;
  return wasmtime_func_new(store, ty, callback, (void*) env, NULL, ret);
}

wasmtime_error_t *go_linker_define_func(
    wasmtime_linker_t *linker,
    const char *module,
    size_t module_len,
    const char *name,
    size_t name_len,
    const wasm_functype_t *ty,
    int wrap,
    size_t env
) {
  wasmtime_func_callback_t cb = trampoline;
  void(*finalizer)(void*) = goFinalizeFuncNew;
  if (wrap) {
    cb = wrap_trampoline;
    finalizer = goFinalizeFuncWrap;
  }
  return wasmtime_linker_define_func(linker, module, module_len, name, name_len, ty, cb, (void*) env, finalizer);
}

wasmtime_externref_t *go_externref_new(size_t env) {
  return wasmtime_externref_new((void*) env, goFinalizeExternref);
}

#define UNION_ACCESSOR(name, field, ty) \
  ty go_##name##_##field##_get(const name##_t *val) { return val->of.field; } \
  void go_##name##_##field##_set(name##_t *val, ty i) { val->of.field = i; }

EACH_UNION_ACCESSOR(UNION_ACCESSOR)
