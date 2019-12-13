//-----------------------------------------------------------------------------
// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiSodaDocCursor.c
//   Implementation of SODA document cursors.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiSodaDocCursor__allocate() [INTERNAL]
//   Allocate and initialize a SODA document cursor structure.
//-----------------------------------------------------------------------------
int dpiSodaDocCursor__allocate(dpiSodaColl *coll, void *handle,
        dpiSodaDocCursor **cursor, dpiError *error)
{
    dpiSodaDocCursor *tempCursor;

    if (dpiGen__allocate(DPI_HTYPE_SODA_DOC_CURSOR, coll->env,
            (void**) &tempCursor, error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(coll, error, 1);
    tempCursor->coll = coll;
    tempCursor->handle = handle;
    *cursor = tempCursor;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor__check() [INTERNAL]
//   Determine if the SODA document cursor is available to use.
//-----------------------------------------------------------------------------
static int dpiSodaDocCursor__check(dpiSodaDocCursor *cursor,
        const char *fnName, dpiError *error)
{
    if (dpiGen__startPublicFn(cursor, DPI_HTYPE_SODA_DOC_CURSOR, fnName, 1,
            error) < 0)
        return DPI_FAILURE;
    if (!cursor->handle)
        return dpiError__set(error, "check closed",
                DPI_ERR_SODA_CURSOR_CLOSED);
    if (!cursor->coll->db->conn->handle || cursor->coll->db->conn->closing)
        return dpiError__set(error, "check connection", DPI_ERR_NOT_CONNECTED);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor__free() [INTERNAL]
//   Free the memory for a SODA document cursor. Note that the reference to the
// collection must remain until after the handle is freed; otherwise, a
// segfault can take place.
//-----------------------------------------------------------------------------
void dpiSodaDocCursor__free(dpiSodaDocCursor *cursor, dpiError *error)
{
    if (cursor->handle) {
        dpiOci__handleFree(cursor->handle, DPI_OCI_HTYPE_SODA_DOC_CURSOR);
        cursor->handle = NULL;
    }
    if (cursor->coll) {
        dpiGen__setRefCount(cursor->coll, error, -1);
        cursor->coll = NULL;
    }
    dpiUtils__freeMemory(cursor);
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor_addRef() [PUBLIC]
//   Add a reference to the SODA document cursor.
//-----------------------------------------------------------------------------
int dpiSodaDocCursor_addRef(dpiSodaDocCursor *cursor)
{
    return dpiGen__addRef(cursor, DPI_HTYPE_SODA_DOC_CURSOR, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor_close() [PUBLIC]
//   Close the cursor.
//-----------------------------------------------------------------------------
int dpiSodaDocCursor_close(dpiSodaDocCursor *cursor)
{
    dpiError error;

    if (dpiSodaDocCursor__check(cursor, __func__, &error) < 0)
        return dpiGen__endPublicFn(cursor, DPI_FAILURE, &error);
    if (cursor->handle) {
        dpiOci__handleFree(cursor->handle, DPI_OCI_HTYPE_SODA_DOC_CURSOR);
        cursor->handle = NULL;
    }
    return dpiGen__endPublicFn(cursor, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor_getNext() [PUBLIC]
//   Return the next document available from the cursor.
//-----------------------------------------------------------------------------
int dpiSodaDocCursor_getNext(dpiSodaDocCursor *cursor, uint32_t flags,
        dpiSodaDoc **doc)
{
    dpiError error;
    uint32_t mode;
    void *handle;

    if (dpiSodaDocCursor__check(cursor, __func__, &error) < 0)
        return dpiGen__endPublicFn(cursor, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(cursor, doc)
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;
    if (dpiOci__sodaDocGetNext(cursor, &handle, mode, &error) < 0)
        return dpiGen__endPublicFn(cursor, DPI_FAILURE, &error);
    *doc = NULL;
    if (handle) {
        if (dpiSodaDoc__allocate(cursor->coll->db, handle, doc, &error) < 0) {
            dpiOci__handleFree(handle, DPI_OCI_HTYPE_SODA_DOCUMENT);
            return dpiGen__endPublicFn(cursor, DPI_FAILURE, &error);
        }
        (*doc)->binaryContent = cursor->coll->binaryContent;
    }
    return dpiGen__endPublicFn(cursor, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDocCursor_release() [PUBLIC]
//   Release a reference to the SODA document cursor.
//-----------------------------------------------------------------------------
int dpiSodaDocCursor_release(dpiSodaDocCursor *cursor)
{
    return dpiGen__release(cursor, DPI_HTYPE_SODA_DOC_CURSOR, __func__);
}

