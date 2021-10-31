#include <wasm.h>
#include <wasmtime.h>

wasmtime_store_t *go_store_new(wasm_engine_t *engine, size_t env);
void go_func_new(wasmtime_context_t *context, wasm_functype_t *ty, size_t env, int wrap,  wasmtime_func_t *ret);
wasmtime_error_t *go_linker_define_func(
    wasmtime_linker_t *linker,
    const char *module,
    size_t module_len,
    const char *name,
    size_t name_len,
    const wasm_functype_t *ty,
    int wrap,
    size_t env
);
wasmtime_externref_t *go_externref_new(size_t env);

#define EACH_UNION_ACCESSOR(name) \
  UNION_ACCESSOR(wasmtime_val, i32, int32_t) \
  UNION_ACCESSOR(wasmtime_val, i64, int64_t) \
  UNION_ACCESSOR(wasmtime_val, f32, float) \
  UNION_ACCESSOR(wasmtime_val, f64, double) \
  UNION_ACCESSOR(wasmtime_val, externref, wasmtime_externref_t*) \
  UNION_ACCESSOR(wasmtime_val, funcref, wasmtime_func_t) \
  \
  UNION_ACCESSOR(wasmtime_extern, func, wasmtime_func_t) \
  UNION_ACCESSOR(wasmtime_extern, memory, wasmtime_memory_t) \
  UNION_ACCESSOR(wasmtime_extern, instance, wasmtime_instance_t) \
  UNION_ACCESSOR(wasmtime_extern, table, wasmtime_table_t) \
  UNION_ACCESSOR(wasmtime_extern, global, wasmtime_global_t) \
  UNION_ACCESSOR(wasmtime_extern, module, wasmtime_module_t*)

#define UNION_ACCESSOR(name, field, ty) \
  ty go_##name##_##field##_get(const name##_t *val); \
  void go_##name##_##field##_set(name##_t *val, ty i);

EACH_UNION_ACCESSOR(UNION_ACCESSOR)

#undef UNION_ACCESSOR
