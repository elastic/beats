//-----------------------------------------------------------------------------
// Copyright (c) 2016, 2018, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiGen.c
//   Generic routines for managing the types available through public APIs.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// definition of handle types
//-----------------------------------------------------------------------------
static const dpiTypeDef dpiAllTypeDefs[DPI_HTYPE_MAX - DPI_HTYPE_NONE - 1] = {
    {
        "dpiConn",                      // name
        sizeof(dpiConn),                // size of structure
        0x49DC600C,                     // check integer
        (dpiTypeFreeProc) dpiConn__free
    },
    {
        "dpiPool",                      // name
        sizeof(dpiPool),                // size of structure
        0x18E1AA4B,                     // check integer
        (dpiTypeFreeProc) dpiPool__free
    },
    {
        "dpiStmt",                      // name
        sizeof(dpiStmt),                // size of structure
        0x31B02B2E,                     // check integer
        (dpiTypeFreeProc) dpiStmt__free
    },
    {
        "dpiVar",                       // name
        sizeof(dpiVar),                 // size of structure
        0x2AE8C6DC,                     // check integer
        (dpiTypeFreeProc) dpiVar__free
    },
    {
        "dpiLob",                       // name
        sizeof(dpiLob),                 // size of structure
        0xD8F31746,                     // check integer
        (dpiTypeFreeProc) dpiLob__free
    },
    {
        "dpiObject",                    // name
        sizeof(dpiObject),              // size of structure
        0x38616080,                     // check integer
        (dpiTypeFreeProc) dpiObject__free
    },
    {
        "dpiObjectType",                // name
        sizeof(dpiObjectType),          // size of structure
        0x86036059,                     // check integer
        (dpiTypeFreeProc) dpiObjectType__free
    },
    {
        "dpiObjectAttr",                // name
        sizeof(dpiObjectAttr),          // size of structure
        0xea6d5dde,                     // check integer
        (dpiTypeFreeProc) dpiObjectAttr__free
    },
    {
        "dpiSubscr",                    // name
        sizeof(dpiSubscr),              // size of structure
        0xa415a1c0,                     // check integer
        (dpiTypeFreeProc) dpiSubscr__free
    },
    {
        "dpiDeqOptions",                // name
        sizeof(dpiDeqOptions),          // size of structure
        0x70ee498d,                     // check integer
        (dpiTypeFreeProc) dpiDeqOptions__free
    },
    {
        "dpiEnqOptions",                // name
        sizeof(dpiEnqOptions),          // size of structure
        0x682f3946,                     // check integer
        (dpiTypeFreeProc) dpiEnqOptions__free
    },
    {
        "dpiMsgProps",                  // name
        sizeof(dpiMsgProps),            // size of structure
        0xa2b75506,                     // check integer
        (dpiTypeFreeProc) dpiMsgProps__free
    },
    {
        "dpiRowid",                     // name
        sizeof(dpiRowid),               // size of structure
        0x6204fa04,                     // check integer
        (dpiTypeFreeProc) dpiRowid__free
    },
    {
        "dpiContext",                   // name
        sizeof(dpiContext),             // size of structure
        0xd81b9181,                     // check integer
        NULL
    },
    {
        "dpiSodaColl",                  // name
        sizeof(dpiSodaColl),            // size of structure
        0x3684db22,                     // check integer
        (dpiTypeFreeProc) dpiSodaColl__free
    },
    {
        "dpiSodaCollCursor",            // name
        sizeof(dpiSodaCollCursor),      // size of structure
        0xcdc73b86,                     // check integer
        (dpiTypeFreeProc) dpiSodaCollCursor__free
    },
    {
        "dpiSodaDb",                    // name
        sizeof(dpiSodaDb),              // size of structure
        0x1f386121,                     // check integer
        (dpiTypeFreeProc) dpiSodaDb__free
    },
    {
        "dpiSodaDoc",                   // name
        sizeof(dpiSodaDoc),             // size of structure
        0xaffd950a,                     // check integer
        (dpiTypeFreeProc) dpiSodaDoc__free
    },
    {
        "dpiSodaDocCursor",             // name
        sizeof(dpiSodaDocCursor),       // size of structure
        0x80ceb83b,                     // check integer
        (dpiTypeFreeProc) dpiSodaDocCursor__free
    },
    {
        "dpiQueue",                     // name
        sizeof(dpiQueue),               // size of structure
        0x54904ba2,                     // check integer
        (dpiTypeFreeProc) dpiQueue__free
    }
};


//-----------------------------------------------------------------------------
// dpiGen__addRef() [INTERNAL]
//   Add a reference to the specified handle.
//-----------------------------------------------------------------------------
int dpiGen__addRef(void *ptr, dpiHandleTypeNum typeNum, const char *fnName)
{
    dpiError error;

    if (dpiGen__startPublicFn(ptr, typeNum, fnName, &error) < 0)
        return dpiGen__endPublicFn(ptr, DPI_FAILURE, &error);
    dpiGen__setRefCount(ptr, &error, 1);
    return dpiGen__endPublicFn(ptr, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiGen__allocate() [INTERNAL]
//   Allocate memory for the specified type and initialize the base fields. The
// type specified is assumed to be valid. If the environment is specified, use
// it; otherwise, create a new one. No additional initialization is performed.
//-----------------------------------------------------------------------------
int dpiGen__allocate(dpiHandleTypeNum typeNum, dpiEnv *env, void **handle,
        dpiError *error)
{
    const dpiTypeDef *typeDef;
    dpiBaseType *value;

    typeDef = &dpiAllTypeDefs[typeNum - DPI_HTYPE_NONE - 1];
    if (dpiUtils__allocateMemory(1, typeDef->size, 1, "allocate handle",
            (void**) &value, error) < 0)
        return DPI_FAILURE;
    value->typeDef = typeDef;
    value->checkInt = typeDef->checkInt;
    value->refCount = 1;
    if (!env && typeNum != DPI_HTYPE_CONTEXT) {
        if (dpiUtils__allocateMemory(1, sizeof(dpiEnv), 1, "allocate env",
                (void**) &env, error) < 0) {
            dpiUtils__freeMemory(value);
            return DPI_FAILURE;
        }
    }
    value->env = env;
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_REFS)
        dpiDebug__print("ref %p (%s) -> 1 [NEW]\n", value, typeDef->name);

    *handle = value;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiGen__checkHandle() [INTERNAL]
//   Check that the specific handle is valid, that it matches the type
// requested and that the check integer is still in place.
//-----------------------------------------------------------------------------
int dpiGen__checkHandle(const void *ptr, dpiHandleTypeNum typeNum,
        const char *action, dpiError *error)
{
    dpiBaseType *value = (dpiBaseType*) ptr;
    const dpiTypeDef *typeDef;

    typeDef = &dpiAllTypeDefs[typeNum - DPI_HTYPE_NONE - 1];
    if (!ptr || value->typeDef != typeDef ||
            value->checkInt != typeDef->checkInt)
        return dpiError__set(error, action, DPI_ERR_INVALID_HANDLE,
                typeDef->name);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiGen__endPublicFn() [INTERNAL]
//   This method should be the last call made in any public method using an
// ODPI-C handle (other than dpiContext which is handled differently).
//-----------------------------------------------------------------------------
int dpiGen__endPublicFn(const void *ptr, int returnValue, dpiError *error)
{
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        dpiDebug__print("fn end %s(%p) -> %d\n", error->buffer->fnName, ptr,
                returnValue);
    if (error->handle)
        dpiHandlePool__release(error->env->errorHandles, &error->handle);

    return returnValue;
}


//-----------------------------------------------------------------------------
// dpiGen__release() [INTERNAL]
//   Release a reference to the specified handle. If the reference count
// reaches zero, the resources associated with the handle are released and
// the memory associated with the handle is freed. Any internal references
// held to other handles are also released.
//-----------------------------------------------------------------------------
int dpiGen__release(void *ptr, dpiHandleTypeNum typeNum, const char *fnName)
{
    dpiError error;

    if (dpiGen__startPublicFn(ptr, typeNum, fnName, &error) < 0)
        return dpiGen__endPublicFn(ptr, DPI_FAILURE, &error);
    dpiGen__setRefCount(ptr, &error, -1);
    return dpiGen__endPublicFn(ptr, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiGen__setRefCount() [INTERNAL]
//   Increase or decrease the reference count by the given amount. The handle
// is assumed to be valid at this point. If the environment is in threaded
// mode, acquire the mutex first before making any adjustments to the reference
// count. If the operation sets the reference count to zero, release all
// resources and free the memory associated with the structure.
//-----------------------------------------------------------------------------
void dpiGen__setRefCount(void *ptr, dpiError *error, int increment)
{
    dpiBaseType *value = (dpiBaseType*) ptr;
    unsigned localRefCount;

    // if threaded need to protect modification of the refCount with a mutex;
    // also ensure that if the reference count reaches zero that it is
    // immediately marked invalid in order to avoid race conditions
    if (value->env->threaded)
        dpiMutex__acquire(value->env->mutex);
    value->refCount += increment;
    localRefCount = value->refCount;
    if (localRefCount == 0)
        dpiUtils__clearMemory(&value->checkInt, sizeof(value->checkInt));
    if (value->env->threaded)
        dpiMutex__release(value->env->mutex);

    // reference count debugging
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_REFS)
        dpiDebug__print("ref %p (%s) -> %d\n", ptr, value->typeDef->name,
                localRefCount);

    // if the refCount has reached zero, call the free routine
    if (localRefCount == 0)
        (*value->typeDef->freeProc)(value, error);
}


//-----------------------------------------------------------------------------
// dpiGen__startPublicFn() [INTERNAL]
//   This method should be the first call made in any public method using an
// ODPI-C handle (other than dpiContext which is handled differently). The
// handle is checked for validity and an error handle is acquired for use in
// all subsequent calls.
//-----------------------------------------------------------------------------
int dpiGen__startPublicFn(const void *ptr, dpiHandleTypeNum typeNum,
        const char *fnName, dpiError *error)
{
    dpiBaseType *value = (dpiBaseType*) ptr;

    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        dpiDebug__print("fn start %s(%p)\n", fnName, ptr);
    if (dpiGlobal__initError(fnName, error) < 0)
        return DPI_FAILURE;
    if (dpiGen__checkHandle(ptr, typeNum, "check main handle", error) < 0)
        return DPI_FAILURE;
    error->env = value->env;
    return DPI_SUCCESS;
}
