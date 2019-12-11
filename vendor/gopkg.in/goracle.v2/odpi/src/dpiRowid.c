//-----------------------------------------------------------------------------
// Copyright (c) 2016, 2017, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiRowid.c
//   Implementation of rowids.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiRowid__allocate() [INTERNAL]
//   Allocate and initialize a rowid object.
//-----------------------------------------------------------------------------
int dpiRowid__allocate(dpiConn *conn, dpiRowid **rowid, dpiError *error)
{
    dpiRowid *tempRowid;

    if (dpiGen__allocate(DPI_HTYPE_ROWID, conn->env, (void**) &tempRowid,
            error) < 0)
        return DPI_FAILURE;
    if (dpiOci__descriptorAlloc(conn->env->handle, &tempRowid->handle,
            DPI_OCI_DTYPE_ROWID, "allocate descriptor", error) < 0) {
        dpiRowid__free(tempRowid, error);
        return DPI_FAILURE;
    }

    *rowid = tempRowid;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiRowid__free() [INTERNAL]
//   Free the memory for a rowid.
//-----------------------------------------------------------------------------
void dpiRowid__free(dpiRowid *rowid, UNUSED dpiError *error)
{
    if (rowid->handle) {
        dpiOci__descriptorFree(rowid->handle, DPI_OCI_DTYPE_ROWID);
        rowid->handle = NULL;
    }
    if (rowid->buffer) {
        dpiUtils__freeMemory(rowid->buffer);
        rowid->buffer = NULL;
    }
    dpiUtils__freeMemory(rowid);
}


//-----------------------------------------------------------------------------
// dpiRowid_addRef() [PUBLIC]
//   Add a reference to the rowid.
//-----------------------------------------------------------------------------
int dpiRowid_addRef(dpiRowid *rowid)
{
    return dpiGen__addRef(rowid, DPI_HTYPE_ROWID, __func__);
}


//-----------------------------------------------------------------------------
// dpiRowid_getStringValue() [PUBLIC]
//   Get the string representation of the rowid.
//-----------------------------------------------------------------------------
int dpiRowid_getStringValue(dpiRowid *rowid, const char **value,
        uint32_t *valueLength)
{
    char temp, *adjustedBuffer, *sourcePtr;
    uint16_t *targetPtr;
    dpiError error;
    uint16_t i;

    if (dpiGen__startPublicFn(rowid, DPI_HTYPE_ROWID, __func__, 1,
            &error) < 0)
        return dpiGen__endPublicFn(rowid, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(rowid, value)
    DPI_CHECK_PTR_NOT_NULL(rowid, valueLength)
    if (!rowid->buffer) {

        // determine length of rowid
        rowid->bufferLength = 0;
        dpiOci__rowidToChar(rowid, &temp, &rowid->bufferLength, &error);

        // allocate and populate buffer containing string representation
        if (dpiUtils__allocateMemory(1, rowid->bufferLength, 0,
                "allocate rowid buffer", (void**) &rowid->buffer, &error) < 0)
            return dpiGen__endPublicFn(rowid, DPI_FAILURE, &error);
        if (dpiOci__rowidToChar(rowid, rowid->buffer, &rowid->bufferLength,
                &error) < 0)
            return dpiGen__endPublicFn(rowid, DPI_FAILURE, &error);

        // UTF-16 is not handled properly (data is returned as ASCII instead)
        // adjust the buffer to use the correct encoding
        if (rowid->env->charsetId == DPI_CHARSET_ID_UTF16) {
            if (dpiUtils__allocateMemory(2, rowid->bufferLength, 0,
                    "allocate rowid buffer", (void**) &adjustedBuffer,
                    &error) < 0) {
                dpiUtils__freeMemory(rowid->buffer);
                rowid->bufferLength = 0;
                rowid->buffer = NULL;
                return dpiGen__endPublicFn(rowid, DPI_FAILURE, &error);
            }
            sourcePtr = rowid->buffer;
            targetPtr = (uint16_t*) adjustedBuffer;
            for (i = 0; i < rowid->bufferLength; i++)
                *targetPtr++ = (uint16_t) *sourcePtr++;
            dpiUtils__freeMemory(rowid->buffer);
            rowid->buffer = adjustedBuffer;
            rowid->bufferLength *= 2;
        }

    }

    *value = rowid->buffer;
    *valueLength = rowid->bufferLength;
    return dpiGen__endPublicFn(rowid, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiRowid_release() [PUBLIC]
//   Release a reference to the rowid.
//-----------------------------------------------------------------------------
int dpiRowid_release(dpiRowid *rowid)
{
    return dpiGen__release(rowid, DPI_HTYPE_ROWID, __func__);
}

