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
// dpiSodaDoc.c
//   Implementation of SODA documents.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiSodaDoc__allocate() [INTERNAL]
//   Allocate and initialize a SODA document structure.
//-----------------------------------------------------------------------------
int dpiSodaDoc__allocate(dpiSodaDb *db, void *handle, dpiSodaDoc **doc,
        dpiError *error)
{
    dpiSodaDoc *tempDoc;

    if (dpiGen__allocate(DPI_HTYPE_SODA_DOC, db->env, (void**) &tempDoc,
            error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(db, error, 1);
    tempDoc->db = db;
    tempDoc->handle = handle;
    *doc = tempDoc;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDoc__check() [INTERNAL]
//   Determine if the SODA document is available to use.
//-----------------------------------------------------------------------------
static int dpiSodaDoc__check(dpiSodaDoc *doc, const char *fnName,
        dpiError *error)
{
    if (dpiGen__startPublicFn(doc, DPI_HTYPE_SODA_DOC, fnName, 1, error) < 0)
        return DPI_FAILURE;
    if (!doc->db->conn->handle || doc->db->conn->closing)
        return dpiError__set(error, "check connection", DPI_ERR_NOT_CONNECTED);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDoc__free() [INTERNAL]
//   Free the memory for a SODA document. Note that the reference to the
// database must remain until after the handle is freed; otherwise, a segfault
// can take place.
//-----------------------------------------------------------------------------
void dpiSodaDoc__free(dpiSodaDoc *doc, dpiError *error)
{
    if (doc->handle) {
        dpiOci__handleFree(doc->handle, DPI_OCI_HTYPE_SODA_DOCUMENT);
        doc->handle = NULL;
    }
    if (doc->db) {
        dpiGen__setRefCount(doc->db, error, -1);
        doc->db = NULL;
    }
    dpiUtils__freeMemory(doc);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc__getAttributeText() [INTERNAL]
//   Get the value of the OCI attribute as a text string.
//-----------------------------------------------------------------------------
static int dpiSodaDoc__getAttributeText(dpiSodaDoc *doc, uint32_t attribute,
        const char **value, uint32_t *valueLength, const char *fnName)
{
    dpiError error;
    int status;

    // validate parameters
    if (dpiSodaDoc__check(doc, fnName, &error) < 0)
        return dpiGen__endPublicFn(doc, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(doc, value)
    DPI_CHECK_PTR_NOT_NULL(doc, valueLength)

    // get attribute value
    status = dpiOci__attrGet(doc->handle, DPI_OCI_HTYPE_SODA_DOCUMENT,
            (void*) value, valueLength, attribute, "get value", &error);
    return dpiGen__endPublicFn(doc, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_addRef() [PUBLIC]
//   Add a reference to the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_addRef(dpiSodaDoc *doc)
{
    return dpiGen__addRef(doc, DPI_HTYPE_SODA_DOC, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getContent() [PUBLIC]
//   Return the content of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getContent(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength, const char **encoding)
{
    uint16_t charsetId;
    dpiError error;

    // validate parameters
    if (dpiSodaDoc__check(doc, __func__, &error) < 0)
        return dpiGen__endPublicFn(doc, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(doc, value)
    DPI_CHECK_PTR_NOT_NULL(doc, valueLength)
    DPI_CHECK_PTR_NOT_NULL(doc, encoding)

    // get content
    if (dpiOci__attrGet(doc->handle, DPI_OCI_HTYPE_SODA_DOCUMENT,
            (void*) value, valueLength, DPI_OCI_ATTR_SODA_CONTENT,
            "get content", &error) < 0)
        return dpiGen__endPublicFn(doc, DPI_FAILURE, &error);

    // if content is not in binary form, always use UTF-8
    if (!doc->binaryContent)
        *encoding = DPI_CHARSET_NAME_UTF8;

    // otherwise, determine the encoding from OCI
    else {
        if (dpiOci__attrGet(doc->handle, DPI_OCI_HTYPE_SODA_DOCUMENT,
                (void*) &charsetId, 0, DPI_OCI_ATTR_SODA_JSON_CHARSET_ID,
                "get charset", &error) < 0)
            return dpiGen__endPublicFn(doc, DPI_FAILURE, &error);
        switch (charsetId) {
            case 0:
                *encoding = NULL;
                break;
            case DPI_CHARSET_ID_UTF8:
                *encoding = DPI_CHARSET_NAME_UTF8;
                break;
            case DPI_CHARSET_ID_UTF16BE:
                *encoding = DPI_CHARSET_NAME_UTF16BE;
                break;
            case DPI_CHARSET_ID_UTF16LE:
                *encoding = DPI_CHARSET_NAME_UTF16LE;
                break;
            default:
                dpiError__set(&error, "check charset",
                        DPI_ERR_INVALID_CHARSET_ID, charsetId);
                return dpiGen__endPublicFn(doc, DPI_FAILURE, &error);
        }
    }

    return dpiGen__endPublicFn(doc, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getCreatedOn() [PUBLIC]
//   Return the created timestamp of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getCreatedOn(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength)
{
    return dpiSodaDoc__getAttributeText(doc,
            DPI_OCI_ATTR_SODA_CREATE_TIMESTAMP, value, valueLength, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getKey() [PUBLIC]
//   Return the key of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getKey(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength)
{
    return dpiSodaDoc__getAttributeText(doc, DPI_OCI_ATTR_SODA_KEY, value,
            valueLength, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getLastModified() [PUBLIC]
//   Return the last modified timestamp of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getLastModified(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength)
{
    return dpiSodaDoc__getAttributeText(doc,
            DPI_OCI_ATTR_SODA_LASTMOD_TIMESTAMP, value, valueLength, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getMediaType() [PUBLIC]
//   Return the media type of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getMediaType(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength)
{
    return dpiSodaDoc__getAttributeText(doc, DPI_OCI_ATTR_SODA_MEDIA_TYPE,
            value, valueLength, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_getVersion() [PUBLIC]
//   Return the version of the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_getVersion(dpiSodaDoc *doc, const char **value,
        uint32_t *valueLength)
{
    return dpiSodaDoc__getAttributeText(doc, DPI_OCI_ATTR_SODA_VERSION,
            value, valueLength, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDoc_release() [PUBLIC]
//   Release a reference to the SODA document.
//-----------------------------------------------------------------------------
int dpiSodaDoc_release(dpiSodaDoc *doc)
{
    return dpiGen__release(doc, DPI_HTYPE_SODA_DOC, __func__);
}

