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
// dpiSodaDb.c
//   Implementation of SODA database methods.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiSodaDb__checkConnected() [INTERNAL]
//   Check to see that the connection to the database is available for use.
//-----------------------------------------------------------------------------
static int dpiSodaDb__checkConnected(dpiSodaDb *db, const char *fnName,
        dpiError *error)
{
    if (dpiGen__startPublicFn(db, DPI_HTYPE_SODA_DB, fnName, 1, error) < 0)
        return DPI_FAILURE;
    if (!db->conn->handle || db->conn->closing)
        return dpiError__set(error, "check connection", DPI_ERR_NOT_CONNECTED);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDb__getCollectionNames() [PUBLIC]
//   Internal method used for getting all collection names from the database.
// The provided cursor handle is iterated until either the limit is reached
// or there are no more collections to find.
//-----------------------------------------------------------------------------
static int dpiSodaDb__getCollectionNames(dpiSodaDb *db, void *cursorHandle,
        uint32_t limit, dpiSodaCollNames *names, char **namesBuffer,
        dpiError *error)
{
    uint32_t numAllocatedNames, namesBufferUsed, namesBufferAllocated;
    uint32_t i, nameLength, *tempNameLengths;
    char *name, *tempNamesBuffer, *ptr;
    void *collectionHandle;

    ptr = *namesBuffer;
    namesBufferUsed = namesBufferAllocated = numAllocatedNames = 0;
    while (names->numNames < limit || limit == 0) {

        // get next collection from cursor
        if (dpiOci__sodaCollGetNext(db->conn, cursorHandle, &collectionHandle,
                DPI_OCI_DEFAULT, error) < 0)
            return DPI_FAILURE;
        if (!collectionHandle)
            break;

        // get name from collection
        if (dpiOci__attrGet(collectionHandle, DPI_OCI_HTYPE_SODA_COLLECTION,
                (void*) &name, &nameLength, DPI_OCI_ATTR_SODA_COLL_NAME,
                "get collection name", error) < 0) {
            dpiOci__handleFree(collectionHandle,
                    DPI_OCI_HTYPE_SODA_COLLECTION);
            return DPI_FAILURE;
        }

        // allocate additional space for the lengths array, if needed
        if (numAllocatedNames <= names->numNames) {
            numAllocatedNames += 256;
            if (dpiUtils__allocateMemory(numAllocatedNames, sizeof(uint32_t),
                    0, "allocate lengths array", (void**) &tempNameLengths,
                    error) < 0) {
                dpiOci__handleFree(collectionHandle,
                        DPI_OCI_HTYPE_SODA_COLLECTION);
                return DPI_FAILURE;
            }
            if (names->nameLengths) {
                memcpy(tempNameLengths, names->nameLengths,
                        names->numNames * sizeof(uint32_t));
                dpiUtils__freeMemory(names->nameLengths);
            }
            names->nameLengths = tempNameLengths;
        }

        // allocate additional space for the names buffer, if needed
        if (namesBufferUsed + nameLength > namesBufferAllocated) {
            namesBufferAllocated += 32768;
            if (dpiUtils__allocateMemory(namesBufferAllocated, 1, 0,
                    "allocate names buffer", (void**) &tempNamesBuffer,
                    error) < 0) {
                dpiOci__handleFree(collectionHandle,
                        DPI_OCI_HTYPE_SODA_COLLECTION);
                return DPI_FAILURE;
            }
            if (*namesBuffer) {
                memcpy(tempNamesBuffer, *namesBuffer, namesBufferUsed);
                dpiUtils__freeMemory(*namesBuffer);
            }
            *namesBuffer = tempNamesBuffer;
            ptr = *namesBuffer + namesBufferUsed;
        }

        // store name in buffer and length in array
        // the names array itself is created and populated afterwards in order
        // to avoid unnecessary copying
        memcpy(ptr, name, nameLength);
        namesBufferUsed += nameLength;
        names->nameLengths[names->numNames] = nameLength;
        names->numNames++;
        ptr += nameLength;

        // free collection now that we have processed it successfully
        dpiOci__handleFree(collectionHandle, DPI_OCI_HTYPE_SODA_COLLECTION);

    }

    // now that all of the names have been determined, populate names array
    if (names->numNames > 0) {
        if (dpiUtils__allocateMemory(names->numNames, sizeof(char*), 0,
                "allocate names array", (void**) &names->names, error) < 0)
            return DPI_FAILURE;
        ptr = *namesBuffer;
        for (i = 0; i < names->numNames; i++) {
            names->names[i] = ptr;
            ptr += names->nameLengths[i];
        }
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiSodaDb__free() [INTERNAL]
//   Free the memory for a SODA database.
//-----------------------------------------------------------------------------
void dpiSodaDb__free(dpiSodaDb *db, dpiError *error)
{
    if (db->conn) {
        dpiGen__setRefCount(db->conn, error, -1);
        db->conn = NULL;
    }
    dpiUtils__freeMemory(db);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_addRef() [PUBLIC]
//   Add a reference to the SODA database.
//-----------------------------------------------------------------------------
int dpiSodaDb_addRef(dpiSodaDb *db)
{
    return dpiGen__addRef(db, DPI_HTYPE_SODA_DB, __func__);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_createCollection() [PUBLIC]
//   Create a new SODA collection with the given name and metadata.
//-----------------------------------------------------------------------------
int dpiSodaDb_createCollection(dpiSodaDb *db, const char *name,
        uint32_t nameLength, const char *metadata, uint32_t metadataLength,
        uint32_t flags, dpiSodaColl **coll)
{
    dpiError error;
    uint32_t mode;
    void *handle;

    // validate parameters
    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(db, name)
    DPI_CHECK_PTR_AND_LENGTH(db, metadata)
    DPI_CHECK_PTR_NOT_NULL(db, coll)

    // determine OCI mode to use
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;
    if (flags & DPI_SODA_FLAGS_CREATE_COLL_MAP)
        mode |= DPI_OCI_SODA_COLL_CREATE_MAP;

    // create collection
    if (dpiOci__sodaCollCreateWithMetadata(db, name, nameLength, metadata,
            metadataLength, mode, &handle, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    if (dpiSodaColl__allocate(db, handle, coll, &error) < 0) {
        dpiOci__handleFree(handle, DPI_OCI_HTYPE_SODA_COLLECTION);
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    }
    return dpiGen__endPublicFn(db, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_createDocument() [PUBLIC]
//   Create a SODA document that can be inserted in the collection or can be
// used to replace and existing document in the collection.
//-----------------------------------------------------------------------------
int dpiSodaDb_createDocument(dpiSodaDb *db, const char *key,
        uint32_t keyLength, const char *content, uint32_t contentLength,
        const char *mediaType, uint32_t mediaTypeLength, uint32_t flags,
        dpiSodaDoc **doc)
{
    int detectEncoding;
    void *docHandle;
    dpiError error;

    // validate parameters
    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(db, key)
    DPI_CHECK_PTR_AND_LENGTH(db, content)
    DPI_CHECK_PTR_AND_LENGTH(db, mediaType)
    DPI_CHECK_PTR_NOT_NULL(db, doc)

    // allocate SODA document handle
    if (dpiOci__handleAlloc(db->env->handle, &docHandle,
            DPI_OCI_HTYPE_SODA_DOCUMENT, "allocate SODA document handle",
            &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);

    // set key, if applicable
    if (key && keyLength > 0) {
        if (dpiOci__attrSet(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT,
                (void*) key, keyLength, DPI_OCI_ATTR_SODA_KEY, "set key",
                &error) < 0) {
            dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
            return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
        }
    }

    // set content, if applicable
    if (content && contentLength > 0) {
        detectEncoding = 1;
        if (dpiOci__attrSet(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT,
                (void*) &detectEncoding, 0, DPI_OCI_ATTR_SODA_DETECT_JSON_ENC,
                "set detect encoding", &error) < 0) {
            dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
            return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
        }
        if (dpiOci__attrSet(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT,
                (void*) content, contentLength, DPI_OCI_ATTR_SODA_CONTENT,
                "set content", &error) < 0) {
            dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
            return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
        }
    }

    // set media type, if applicable
    if (mediaType && mediaTypeLength > 0) {
        if (dpiOci__attrSet(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT,
                (void*) mediaType, mediaTypeLength,
                DPI_OCI_ATTR_SODA_MEDIA_TYPE, "set media type", &error) < 0) {
            dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
            return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
        }
    }

    // allocate the ODPI-C document that will be returned
    if (dpiSodaDoc__allocate(db, docHandle, doc, &error) < 0) {
        dpiOci__handleFree(docHandle, DPI_OCI_HTYPE_SODA_DOCUMENT);
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    }
    (*doc)->binaryContent = 1;

    return dpiGen__endPublicFn(db, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_freeCollectionNames() [PUBLIC]
//   Free the names of the collections allocated earlier with a call to
// dpiSodaDb_getCollectionNames().
//-----------------------------------------------------------------------------
int dpiSodaDb_freeCollectionNames(dpiSodaDb *db, dpiSodaCollNames *names)
{
    dpiError error;

    // validate parameters
    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(db, names)

    // perform frees; note that the memory for the names themselves is stored
    // in one contiguous block pointed to by the first name
    if (names->names) {
        dpiUtils__freeMemory((void*) names->names[0]);
        dpiUtils__freeMemory((void*) names->names);
        names->names = NULL;
    }
    if (names->nameLengths) {
        dpiUtils__freeMemory(names->nameLengths);
        names->nameLengths = NULL;
    }
    names->numNames = 0;

    return dpiGen__endPublicFn(db, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_getCollections() [PUBLIC]
//   Return a cursor to iterate over the SODA collections in the database.
//-----------------------------------------------------------------------------
int dpiSodaDb_getCollections(dpiSodaDb *db, const char *startName,
        uint32_t startNameLength, uint32_t flags, dpiSodaCollCursor **cursor)
{
    dpiError error;
    uint32_t mode;
    void *handle;

    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(db, startName)
    DPI_CHECK_PTR_NOT_NULL(db, cursor)
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;
    if (dpiOci__sodaCollList(db, startName, startNameLength, &handle, mode,
            &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    if (dpiSodaCollCursor__allocate(db, handle, cursor, &error) < 0) {
        dpiOci__handleFree(handle, DPI_OCI_HTYPE_SODA_COLL_CURSOR);
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    }
    return dpiGen__endPublicFn(db, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_getCollectionNames() [PUBLIC]
//   Return the names of all collections in the provided array.
//-----------------------------------------------------------------------------
int dpiSodaDb_getCollectionNames(dpiSodaDb *db, const char *startName,
        uint32_t startNameLength, uint32_t limit, uint32_t flags,
        dpiSodaCollNames *names)
{
    char *namesBuffer;
    dpiError error;
    uint32_t mode;
    void *handle;
    int status;

    // validate parameters
    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(db, startName)
    DPI_CHECK_PTR_NOT_NULL(db, names)

    // initialize output structure
    names->numNames = 0;
    names->names = NULL;
    names->nameLengths = NULL;

    // determine OCI mode to use
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;

    // acquire collection cursor
    if (dpiOci__sodaCollList(db, startName, startNameLength, &handle, mode,
            &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);

    // iterate over cursor to acquire collection names
    namesBuffer = NULL;
    status = dpiSodaDb__getCollectionNames(db, handle, limit, names,
            &namesBuffer, &error);
    dpiOci__handleFree(handle, DPI_OCI_HTYPE_SODA_COLL_CURSOR);
    if (status < 0) {
        names->numNames = 0;
        if (namesBuffer) {
            dpiUtils__freeMemory(namesBuffer);
            namesBuffer = NULL;
        }
        if (names->names) {
            dpiUtils__freeMemory((void*) names->names);
            names->names = NULL;
        }
        if (names->nameLengths) {
            dpiUtils__freeMemory(names->nameLengths);
            names->nameLengths = NULL;
        }
    }
    return dpiGen__endPublicFn(db, status, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_openCollection() [PUBLIC]
//   Open an existing SODA collection and return a handle to it.
//-----------------------------------------------------------------------------
int dpiSodaDb_openCollection(dpiSodaDb *db, const char *name,
        uint32_t nameLength, uint32_t flags, dpiSodaColl **coll)
{
    dpiError error;
    uint32_t mode;
    void *handle;

    if (dpiSodaDb__checkConnected(db, __func__, &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(db, name)
    DPI_CHECK_PTR_NOT_NULL(db, coll)
    mode = DPI_OCI_DEFAULT;
    if (flags & DPI_SODA_FLAGS_ATOMIC_COMMIT)
        mode |= DPI_OCI_SODA_ATOMIC_COMMIT;
    if (dpiOci__sodaCollOpen(db, name, nameLength, mode, &handle,
            &error) < 0)
        return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
    *coll = NULL;
    if (handle) {
        if (dpiSodaColl__allocate(db, handle, coll, &error) < 0) {
            dpiOci__handleFree(handle, DPI_OCI_HTYPE_SODA_COLLECTION);
            return dpiGen__endPublicFn(db, DPI_FAILURE, &error);
        }
    }
    return dpiGen__endPublicFn(db, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiSodaDb_release() [PUBLIC]
//   Release a reference to the SODA database.
//-----------------------------------------------------------------------------
int dpiSodaDb_release(dpiSodaDb *db)
{
    return dpiGen__release(db, DPI_HTYPE_SODA_DB, __func__);
}

