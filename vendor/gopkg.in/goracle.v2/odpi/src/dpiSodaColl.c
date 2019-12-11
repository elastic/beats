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
// dpiSodaColl.c
//   Implementation of SODA collections.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiSodaColl__allocate() [INTERNAL]
//   Allocate and initialize a SODA collection structure.
//-----------------------------------------------------------------------------
int dpiSodaColl__allocate(dpiSodaDb *db, void *handle, dpiSodaColl **coll,
        dpiError *error)
{
    uint8_t sqlType, contentType;
    dpiSodaColl *tempColl;

    if (dpiOci__attrGet(handle, DPI_OCI_HTYPE_SODA_COLLECTION,
            (void*) &sqlType, 0, DPI_OCI_ATTR_SODA_CTNT_SQL_TYPE,
            "get content sql type", error) < 0)
        return DPI_FAILURE;
    if (dpiGen__allocate(DPI_HTYPE_SODA_COLL, db->env, (void**) &tempColl,
            error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(db, error, 1);
    tempColl->db = db;
    tempColl->handle = handle;
    if (sqlType == DPI_SQLT_BLOB) {
        tempColl->binaryContent = 1;
        contentType = 0;
        dpiOci__attrGet(handle, DPI_OCI_HTYPE_SODA_COLLECTION,
                (void*) &contentType, 0, DPI_OCI_ATTR_SODA_CTNT_FORMAT,
                    NULL, error);
        if (contentType == DPI_OCI_JSON_FORMAT_OSON)
            tempColl->binaryContent = 0;
    }
    *coll = tempColl;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__check() [INTERNAL]
//   Determine if the SODA collection is available to use.
//-----------------------------------------------------------------------------
static int dpiSodaColl__check(dpiSodaColl *coll, const char *fnName,
        dpiError *error)
{
    if (dpiGen__startPublicFn(coll, DPI_HTYPE_SODA_COLL, fnName, 1, error) < 0)
        return DPI_FAILURE;
    if (!coll->db->conn->handle || coll->db->conn->closing)
        return dpiError__set(error, "check connection", DPI_ERR_NOT_CONNECTED);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__createOperOptions() [INTERNAL]
//   Create a SODA operation options handle with the specified information.
//-----------------------------------------------------------------------------
static int dpiSodaColl__createOperOptions(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, void **handle, dpiError *error)
{
    dpiSodaOperOptions localOptions;

    // if no options specified, use default values
    if (!options) {
        dpiContext__initSodaOperOptions(&localOptions);
        options = &localOptions;
    }

    // allocate new handle
    if (dpiOci__handleAlloc(coll->env->handle, handle,
            DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
            "allocate SODA operation options handle", error) < 0)
        return DPI_FAILURE;

    // set multiple keys, if applicable
    if (options->numKeys > 0) {
        if (dpiOci__sodaOperKeysSet(options, *handle, error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    // set single key, if applicable
    if (options->keyLength > 0) {
        if (dpiOci__attrSet(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
                (void*) options->key, options->keyLength,
                DPI_OCI_ATTR_SODA_KEY, "set key", error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    // set single version, if applicable
    if (options->versionLength > 0) {
        if (dpiOci__attrSet(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
                (void*) options->version, options->versionLength,
                DPI_OCI_ATTR_SODA_VERSION, "set version", error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    // set filter, if applicable
    if (options->filterLength > 0) {
        if (dpiOci__attrSet(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
                (void*) options->filter, options->filterLength,
                DPI_OCI_ATTR_SODA_FILTER, "set filter", error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    // set skip count, if applicable
    if (options->skip > 0) {
        if (dpiOci__attrSet(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
                (void*) &options->skip, 0, DPI_OCI_ATTR_SODA_SKIP,
                "set skip count", error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    // set limit, if applicable
    if (options->limit > 0) {
        if (dpiOci__attrSet(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS,
                (void*) &options->limit, 0, DPI_OCI_ATTR_SODA_LIMIT,
                "set limit", error) < 0) {
            dpiOci__handleFree(*handle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
            return DPI_FAILURE;
        }
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__find() [INTERNAL]
//   Perform a find of SODA documents by creating an operation options handle
// and populating it with the requested options. Once the find is complete,
// return either a cursor or a document.
//-----------------------------------------------------------------------------
static int dpiSodaColl__find(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, uint32_t flags,
        dpiSodaDocCursor **cursor, dpiSodaDoc **doc, dpiError *error)
{
    uint32_t ociMode, returnHandleType, ociFlags;
    void *optionsHandle, *ociReturnHandle;
    int status;

    // determine OCI mode to pass
    ociMode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        ociMode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // create new OCI operation options handle
    if (dpiSodaColl__createOperOptions(coll, options, &optionsHandle,
            error) < 0)
        return DPI_FAILURE;

    // determine OCI flags to use
    ociFlags = (coll->binaryContent) ? DPI_OCI_SODA_AS_STORED :
            DPI_OCI_SODA_AS_AL32UTF8;

    // perform actual find
    if (cursor) {
        *cursor = NULL;
        status = dpiOci__sodaFind(coll, optionsHandle, ociFlags, ociMode,
                &ociReturnHandle, error);
    } else {
        *doc = NULL;
        status = dpiOci__sodaFindOne(coll, optionsHandle, ociFlags, ociMode,
                &ociReturnHandle, error);
    }
    dpiOci__handleFree(optionsHandle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
    if (status < 0)
        return DPI_FAILURE;

    // return cursor or document, as appropriate
    if (cursor) {
        status = dpiSodaDocCursor__allocate(coll, ociReturnHandle, cursor,
                error);
        returnHandleType = DPI_OCI_HTYPE_SODA_DOC_CURSOR;
    } else if (ociReturnHandle) {
        status = dpiSodaDoc__allocate(coll->db, ociReturnHandle, doc, error);
        returnHandleType = DPI_OCI_HTYPE_SODA_DOCUMENT;
    }
    if (status < 0)
        dpiOci__handleFree(ociReturnHandle, returnHandleType);

    return status;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__free() [INTERNAL]
//   Free the memory for a SODA collection. Note that the reference to the
// database must remain until after the handle is freed; otherwise, a segfault
// can take place.
//-----------------------------------------------------------------------------
void dpiSodaColl__free(dpiSodaColl *coll, dpiError *error)
{
    if (coll->handle) {
        dpiOci__handleFree(coll->handle, DPI_OCI_HTYPE_SODA_COLLECTION);
        coll->handle = NULL;
    }
    if (coll->db) {
        dpiGen__setRefCount(coll->db, error, -1);
        coll->db = NULL;
    }
    dpiUtils__freeMemory(coll);
}


//-----------------------------------------------------------------------------
// dpiSodaColl__getDocCount() [INTERNAL]
//   Internal method for getting document count.
//-----------------------------------------------------------------------------
static int dpiSodaColl__getDocCount(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, uint32_t flags, uint64_t *count,
        dpiError *error)
{
    void *optionsHandle;
    uint32_t ociMode;
    int status;

    // determine OCI mode to pass
    ociMode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        ociMode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // create new OCI operation options handle
    if (dpiSodaColl__createOperOptions(coll, options, &optionsHandle,
            error) < 0)
        return DPI_FAILURE;

    // perform actual document count
    status = dpiOci__sodaDocCount(coll, optionsHandle, ociMode, count, error);
    dpiOci__handleFree(optionsHandle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
    return status;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__remove() [INTERNAL]
//   Internal method for removing documents from a collection.
//-----------------------------------------------------------------------------
static int dpiSodaColl__remove(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, uint32_t flags, uint64_t *count,
        dpiError *error)
{
    void *optionsHandle;
    uint32_t mode;
    int status;

    // determine OCI mode to pass
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // create new OCI operation options handle
    if (dpiSodaColl__createOperOptions(coll, options, &optionsHandle,
            error) < 0)
        return DPI_FAILURE;

    // remove documents from collection
    status = dpiOci__sodaRemove(coll, optionsHandle, mode, count, error);
    dpiOci__handleFree(optionsHandle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
    return status;
}


//-----------------------------------------------------------------------------
// dpiSodaColl__replace() [INTERNAL]
//   Internal method for replacing a document in the collection.
//-----------------------------------------------------------------------------
static int dpiSodaColl__replace(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, dpiSodaDoc *doc, uint32_t flags,
        int *replaced, dpiSodaDoc **replacedDoc, dpiError *error)
{
    void *docHandle, *optionsHandle;
    int status, dummyIsReplaced;
    uint32_t mode;

    // use dummy value if the replaced flag is not desired
    if (!replaced)
        replaced = &dummyIsReplaced;

    // determine OCI mode to pass
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // create new OCI operation options handle
    if (dpiSodaColl__createOperOptions(coll, options, &optionsHandle,
            error) < 0)
        return DPI_FAILURE;

    // replace document in collection
    // use "AndGet" variant if the replaced document is requested
    docHandle = doc->handle;
    if (!replacedDoc) {
        status = dpiOci__sodaReplOne(coll, optionsHandle, docHandle, mode,
                replaced, error);
    } else {
        *replacedDoc = NULL;
        status = dpiOci__sodaReplOneAndGet(coll, optionsHandle, &docHandle,
                mode, replaced, error);
        if (status == 0 && docHandle) {
            status = dpiSodaDoc__allocate(coll->db, docHandle, replacedDoc,
                    error);
            if (status < 0)
                dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
        }
    }

    dpiOci__handleFree(optionsHandle, DPI_OCI_HTYPE_SODA_OPER_OPTIONS);
    return status;
}


//-----------------------------------------------------------------------------
// dpiSodaColl_addRef() [PUBLIC]
//   Add a reference to the SODA collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_addRef(dpiSodaColl *coll)
{
    return dpiGen__addRef(coll, DPI_HTYPE_SODA_COLL, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_createIndex() [PUBLIC]
//   Create an index on the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_createIndex(dpiSodaColl *coll, const char *indexSpec,
        uint32_t indexSpecLength, uint32_t flags)
{
    dpiError error;
    uint32_t mode;
    int status;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(coll, indexSpec)

    // determine mode to pass to OCI
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // create index
    status = dpiOci__sodaIndexCreate(coll, indexSpec, indexSpecLength, mode,
            &error);
    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_drop() [PUBLIC]
//   Drop the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_drop(dpiSodaColl *coll, uint32_t flags, int *isDropped)
{
    int status, dummyIsDropped;
    dpiError error;
    uint32_t mode;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    // isDropped is not a mandatory parameter, but it is for OCI
    if (!isDropped)
        isDropped = &dummyIsDropped;

    // determine mode to pass to OCI
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // drop collection
    status = dpiOci__sodaCollDrop(coll, isDropped, mode, &error);
    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_dropIndex() [PUBLIC]
//   Drop the index on the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_dropIndex(dpiSodaColl *coll, const char *name,
        uint32_t nameLength, uint32_t flags, int *isDropped)
{
    int status, dummyIsDropped;
    dpiError error;
    uint32_t mode;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(coll, name)

    // isDropped is not a mandatory parameter, but it is for OCI
    if (!isDropped)
        isDropped = &dummyIsDropped;

    // determine mode to pass to OCI
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;
    if (flags & DPI_SODA_FLAGS_INDEX_DROP_FORCE)
        mode |= DPI_OCI_SODA_INDEX_DROP_FORCE;

    // drop index
    status = dpiOci__sodaIndexDrop(coll, name, nameLength, mode, isDropped,
            &error);
    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_find() [PUBLIC]
//   Find documents in a collection and return a cursor.
//-----------------------------------------------------------------------------
int dpiSodaColl_find(dpiSodaColl *coll, const dpiSodaOperOptions *options,
        uint32_t flags, dpiSodaDocCursor **cursor)
{
    dpiError error;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, cursor)

    // perform find and return a cursor
    if (dpiSodaColl__find(coll, options, flags, cursor, NULL, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    return dpiGen__endPublicFn(coll, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_findOne() [PUBLIC]
//   Find a single document in a collection and return it.
//-----------------------------------------------------------------------------
int dpiSodaColl_findOne(dpiSodaColl *coll, const dpiSodaOperOptions *options,
        uint32_t flags, dpiSodaDoc **doc)
{
    dpiError error;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, doc)

    // perform find and return a document
    if (dpiSodaColl__find(coll, options, flags, NULL, doc, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    return dpiGen__endPublicFn(coll, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_getDataGuide() [PUBLIC]
//   Return the data guide document for the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_getDataGuide(dpiSodaColl *coll, uint32_t flags,
        dpiSodaDoc **doc)
{
    void *docHandle;
    dpiError error;
    uint32_t mode;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, doc)

    // determine mode to pass
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // get data guide
    if (dpiOci__sodaDataGuideGet(coll, &docHandle, mode, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    if (!docHandle) {
        *doc = NULL;
    } else if (dpiSodaDoc__allocate(coll->db, docHandle, doc, &error) < 0) {
        dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    }

    return dpiGen__endPublicFn(coll, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_getDocCount() [PUBLIC]
//   Return the number of documents in the collection that match the specified
// criteria.
//-----------------------------------------------------------------------------
int dpiSodaColl_getDocCount(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, uint32_t flags, uint64_t *count)
{
    dpiError error;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, count)

    // get document count
    if (dpiSodaColl__getDocCount(coll, options, flags, count, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    return dpiGen__endPublicFn(coll, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_getMetadata() [PUBLIC]
//   Return the metadata for the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_getMetadata(dpiSodaColl *coll, const char **value,
        uint32_t *valueLength)
{
    dpiError error;
    int status;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, value)
    DPI_CHECK_PTR_NOT_NULL(coll, valueLength)

    // get attribute value
    status = dpiOci__attrGet(coll->handle, DPI_OCI_HTYPE_SODA_COLLECTION,
            (void*) value, valueLength, DPI_OCI_ATTR_SODA_COLL_DESCRIPTOR,
            "get value", &error);
    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_getName() [PUBLIC]
//   Return the name of the collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_getName(dpiSodaColl *coll, const char **value,
        uint32_t *valueLength)
{
    dpiError error;
    int status;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, value)
    DPI_CHECK_PTR_NOT_NULL(coll, valueLength)

    // get attribute value
    status = dpiOci__attrGet(coll->handle, DPI_OCI_HTYPE_SODA_COLLECTION,
            (void*) value, valueLength, DPI_OCI_ATTR_SODA_COLL_NAME,
            "get value", &error);
    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_insertOne() [PUBLIC]
//   Insert a document into the collection and return a handle to the newly
// created document, if desired.
//-----------------------------------------------------------------------------
int dpiSodaColl_insertOne(dpiSodaColl *coll, dpiSodaDoc *doc, uint32_t flags,
        dpiSodaDoc **insertedDoc)
{
    void *docHandle;
    dpiError error;
    uint32_t mode;
    int status;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    if (dpiGen__checkHandle(doc, DPI_HTYPE_SODA_DOC, "check document",
            &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    // determine OCI mode to use
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // insert document into collection
    // use "AndGet" variant if the inserted document is requested
    docHandle = doc->handle;
    if (!insertedDoc)
        status = dpiOci__sodaInsert(coll, docHandle, mode, &error);
    else {
        status = dpiOci__sodaInsertAndGet(coll, &docHandle, mode, &error);
        if (status == 0) {
            status = dpiSodaDoc__allocate(coll->db, docHandle, insertedDoc,
                    &error);
            if (status < 0)
                dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
        }
    }

    return dpiGen__endPublicFn(coll, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_release() [PUBLIC]
//   Release a reference to the SODA collection.
//-----------------------------------------------------------------------------
int dpiSodaColl_release(dpiSodaColl *coll)
{
    return dpiGen__release(coll, DPI_HTYPE_SODA_COLL, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_remove() [PUBLIC]
//   Remove the documents from the collection that match the given criteria.
//-----------------------------------------------------------------------------
int dpiSodaColl_remove(dpiSodaColl *coll, const dpiSodaOperOptions *options,
        uint32_t flags, uint64_t *count)
{
    dpiError error;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(coll, count)

    // perform removal
    if (dpiSodaColl__remove(coll, options, flags, count, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    return dpiGen__endPublicFn(coll, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaColl_replaceOne() [PUBLIC]
//   Replace the first document in the collection that matches the given
// criteria. Returns a handle to the newly replaced document, if desired.
//-----------------------------------------------------------------------------
int dpiSodaColl_replaceOne(dpiSodaColl *coll,
        const dpiSodaOperOptions *options, dpiSodaDoc *doc, uint32_t flags,
        int *replaced, dpiSodaDoc **replacedDoc)
{
    dpiError error;
    int status;

    // validate parameters
    if (dpiSodaColl__check(coll, __func__, &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);
    if (dpiGen__checkHandle(doc, DPI_HTYPE_SODA_DOC, "check document",
            &error) < 0)
        return dpiGen__endPublicFn(coll, DPI_FAILURE, &error);

    // perform replace
    status = dpiSodaColl__replace(coll, options, doc, flags, replaced,
            replacedDoc, &error);
    return dpiGen__endPublicFn(coll, status, &error);
}

