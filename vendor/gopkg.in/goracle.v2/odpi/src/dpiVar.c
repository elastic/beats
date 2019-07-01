//-----------------------------------------------------------------------------
// Copyright (c) 2016, 2019, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiVar.c
//   Implementation of variables.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// forward declarations of internal functions only used in this file
static int dpiVar__initBuffer(dpiVar *var, dpiVarBuffer *buffer,
        dpiError *error);
static int dpiVar__setBytesFromDynamicBytes(dpiBytes *bytes,
        dpiDynamicBytes *dynBytes, dpiError *error);
static int dpiVar__setBytesFromLob(dpiBytes *bytes, dpiDynamicBytes *dynBytes,
        dpiLob *lob, dpiError *error);
static int dpiVar__setFromBytes(dpiVar *var, uint32_t pos, const char *value,
        uint32_t valueLength, dpiError *error);
static int dpiVar__setFromLob(dpiVar *var, uint32_t pos, dpiLob *lob,
        dpiError *error);
static int dpiVar__setFromObject(dpiVar *var, uint32_t pos, dpiObject *obj,
        dpiError *error);
static int dpiVar__setFromRowid(dpiVar *var, uint32_t pos, dpiRowid *rowid,
        dpiError *error);
static int dpiVar__setFromStmt(dpiVar *var, uint32_t pos, dpiStmt *stmt,
        dpiError *error);
static int dpiVar__validateTypes(const dpiOracleType *oracleType,
        dpiNativeTypeNum nativeTypeNum, dpiError *error);


//-----------------------------------------------------------------------------
// dpiVar__allocate() [INTERNAL]
//   Create a new variable object and return it. In case of error NULL is
// returned.
//-----------------------------------------------------------------------------
int dpiVar__allocate(dpiConn *conn, dpiOracleTypeNum oracleTypeNum,
        dpiNativeTypeNum nativeTypeNum, uint32_t maxArraySize, uint32_t size,
        int sizeIsBytes, int isArray, dpiObjectType *objType, dpiVar **var,
        dpiData **data, dpiError *error)
{
    const dpiOracleType *type;
    uint32_t sizeInBytes;
    dpiVar *tempVar;

    // validate arguments
    *var = NULL;
    type = dpiOracleType__getFromNum(oracleTypeNum, error);
    if (!type)
        return DPI_FAILURE;
    if (maxArraySize == 0)
        return dpiError__set(error, "check max array size",
                DPI_ERR_ARRAY_SIZE_ZERO);
    if (isArray && !type->canBeInArray)
        return dpiError__set(error, "check can be in array",
                DPI_ERR_NOT_SUPPORTED);
    if (oracleTypeNum == DPI_ORACLE_TYPE_BOOLEAN &&
            dpiUtils__checkClientVersion(conn->env->versionInfo, 12, 1,
                    error) < 0)
        return DPI_FAILURE;
    if (nativeTypeNum != type->defaultNativeTypeNum) {
        if (dpiVar__validateTypes(type, nativeTypeNum, error) < 0)
            return DPI_FAILURE;
    }

    // calculate size in bytes
    if (size == 0)
        size = 1;
    if (type->sizeInBytes)
        sizeInBytes = type->sizeInBytes;
    else if (sizeIsBytes || !type->isCharacterData)
        sizeInBytes = size;
    else if (type->charsetForm == DPI_SQLCS_IMPLICIT)
        sizeInBytes = size * conn->env->maxBytesPerCharacter;
    else sizeInBytes = size * conn->env->nmaxBytesPerCharacter;

    // allocate memory for variable type
    if (dpiGen__allocate(DPI_HTYPE_VAR, conn->env, (void**) &tempVar,
            error) < 0)
        return DPI_FAILURE;

    // basic initialization
    tempVar->buffer.maxArraySize = maxArraySize;
    if (!isArray)
        tempVar->buffer.actualArraySize = maxArraySize;
    tempVar->sizeInBytes = sizeInBytes;
    if (sizeInBytes > DPI_MAX_BASIC_BUFFER_SIZE) {
        tempVar->sizeInBytes = 0;
        tempVar->isDynamic = 1;
        tempVar->requiresPreFetch = 1;
    }
    tempVar->type = type;
    tempVar->nativeTypeNum = nativeTypeNum;
    tempVar->isArray = isArray;
    dpiGen__setRefCount(conn, error, 1);
    tempVar->conn = conn;
    if (objType) {
        if (dpiGen__checkHandle(objType, DPI_HTYPE_OBJECT_TYPE,
                "check object type", error) < 0) {
            dpiVar__free(tempVar, error);
            return DPI_FAILURE;
        }
        dpiGen__setRefCount(objType, error, 1);
        tempVar->objectType = objType;
    }

    // allocate the data for the variable
    if (dpiVar__initBuffer(tempVar, &tempVar->buffer, error) < 0) {
        dpiVar__free(tempVar, error);
        return DPI_FAILURE;
    }

    *var = tempVar;
    *data = tempVar->buffer.externalData;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__allocateChunks() [INTERNAL]
//   Allocate more chunks for handling dynamic bytes.
//-----------------------------------------------------------------------------
static int dpiVar__allocateChunks(dpiDynamicBytes *dynBytes, dpiError *error)
{
    dpiDynamicBytesChunk *chunks;
    uint32_t allocatedChunks;

    allocatedChunks = dynBytes->allocatedChunks + 8;
    if (dpiUtils__allocateMemory(allocatedChunks, sizeof(dpiDynamicBytesChunk),
            1, "allocate chunks", (void**) &chunks, error) < 0)
        return DPI_FAILURE;
    if (dynBytes->chunks) {
        memcpy(chunks, dynBytes->chunks,
                dynBytes->numChunks * sizeof(dpiDynamicBytesChunk));
        dpiUtils__freeMemory(dynBytes->chunks);
    }
    dynBytes->chunks = chunks;
    dynBytes->allocatedChunks = allocatedChunks;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__allocateDynamicBytes() [INTERNAL]
//   Allocate space in the dynamic bytes structure for the specified number of
// bytes. When complete, there will be exactly one allocated chunk of the
// specified size or greater in the dynamic bytes structure.
//-----------------------------------------------------------------------------
static int dpiVar__allocateDynamicBytes(dpiDynamicBytes *dynBytes,
        uint32_t size, dpiError *error)
{
    // if an error occurs, none of the original space is valid
    dynBytes->numChunks = 0;

    // if there are no chunks at all, make sure some exist
    if (dynBytes->allocatedChunks == 0 &&
            dpiVar__allocateChunks(dynBytes, error) < 0)
        return DPI_FAILURE;

    // at this point there should be 0 or 1 chunks as any retrieval that
    // resulted in multiple chunks would have been consolidated already
    // make sure that chunk has enough space in it
    if (size > dynBytes->chunks->allocatedLength) {
        if (dynBytes->chunks->ptr)
            dpiUtils__freeMemory(dynBytes->chunks->ptr);
        dynBytes->chunks->allocatedLength =
                (size + DPI_DYNAMIC_BYTES_CHUNK_SIZE - 1) &
                        ~(DPI_DYNAMIC_BYTES_CHUNK_SIZE - 1);
        if (dpiUtils__allocateMemory(1, dynBytes->chunks->allocatedLength, 0,
                "allocate chunk", (void**) &dynBytes->chunks->ptr, error) < 0)
            return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__assignCallbackBuffer() [INTERNAL]
//   Assign callback pointers during OCI statement execution. This is used with
// the callack functions used for dynamic binding during DML returning
// statement execution.
//-----------------------------------------------------------------------------
static void dpiVar__assignCallbackBuffer(dpiVar *var, dpiVarBuffer *buffer,
        uint32_t index, void **bufpp)
{
    switch (var->type->oracleTypeNum) {
        case DPI_ORACLE_TYPE_TIMESTAMP:
        case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
        case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
            *bufpp = buffer->data.asTimestamp[index];
            break;
        case DPI_ORACLE_TYPE_INTERVAL_DS:
        case DPI_ORACLE_TYPE_INTERVAL_YM:
            *bufpp = buffer->data.asInterval[index];
            break;
        case DPI_ORACLE_TYPE_CLOB:
        case DPI_ORACLE_TYPE_BLOB:
        case DPI_ORACLE_TYPE_NCLOB:
        case DPI_ORACLE_TYPE_BFILE:
            *bufpp = buffer->data.asLobLocator[index];
            break;
        case DPI_ORACLE_TYPE_ROWID:
            *bufpp = buffer->data.asRowid[index];
            break;
        case DPI_ORACLE_TYPE_STMT:
            *bufpp = buffer->data.asStmt[index];
            break;
        default:
            *bufpp = buffer->data.asBytes + index * var->sizeInBytes;
            break;
    }
}


//-----------------------------------------------------------------------------
// dpiVar__checkArraySize() [INTERNAL]
//   Verifies that the array size has not been exceeded.
//-----------------------------------------------------------------------------
static int dpiVar__checkArraySize(dpiVar *var, uint32_t pos,
        const char *fnName, int needErrorHandle, dpiError *error)
{
    if (dpiGen__startPublicFn(var, DPI_HTYPE_VAR, fnName, needErrorHandle,
            error) < 0)
        return DPI_FAILURE;
    if (pos >= var->buffer.maxArraySize)
        return dpiError__set(error, "check array size",
                DPI_ERR_INVALID_ARRAY_POSITION, pos,
                var->buffer.maxArraySize);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__convertToLob() [INTERNAL]
//   Convert the variable from using dynamic bytes for a long string to using a
// LOB instead. This is needed for PL/SQL which cannot handle more than 32K
// without the use of a LOB.
//-----------------------------------------------------------------------------
int dpiVar__convertToLob(dpiVar *var, dpiError *error)
{
    dpiDynamicBytes *dynBytes;
    dpiLob *lob;
    uint32_t i;

    // change type based on the original Oracle type
    if (var->type->oracleTypeNum == DPI_ORACLE_TYPE_RAW ||
            var->type->oracleTypeNum == DPI_ORACLE_TYPE_LONG_RAW)
        var->type = dpiOracleType__getFromNum(DPI_ORACLE_TYPE_BLOB, error);
    else if (var->type->oracleTypeNum == DPI_ORACLE_TYPE_NCHAR)
        var->type = dpiOracleType__getFromNum(DPI_ORACLE_TYPE_NCLOB,
                error);
    else var->type = dpiOracleType__getFromNum(DPI_ORACLE_TYPE_CLOB,
            error);

    // adjust attributes and re-initialize buffers
    // the dynamic bytes structures will not be removed
    var->sizeInBytes = var->type->sizeInBytes;
    var->isDynamic = 0;
    if (dpiVar__initBuffer(var, &var->buffer, error) < 0)
        return DPI_FAILURE;

    // copy any values already set
    for (i = 0; i < var->buffer.maxArraySize; i++) {
        dynBytes = &var->buffer.dynamicBytes[i];
        lob = var->buffer.references[i].asLOB;
        if (dynBytes->numChunks == 0)
            continue;
        if (dpiLob__setFromBytes(lob, dynBytes->chunks->ptr,
                dynBytes->chunks->length, error) < 0)
            return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__copyData() [INTERNAL]
//   Copy the data from the source to the target variable at the given array
// position.
//-----------------------------------------------------------------------------
int dpiVar__copyData(dpiVar *var, uint32_t pos, dpiData *sourceData,
        dpiError *error)
{
    dpiData *targetData = &var->buffer.externalData[pos];

    // handle null case
    targetData->isNull = sourceData->isNull;
    if (sourceData->isNull)
        return DPI_SUCCESS;

    // handle copying of value from source to target
    switch (var->nativeTypeNum) {
        case DPI_NATIVE_TYPE_BYTES:
            return dpiVar__setFromBytes(var, pos,
                    sourceData->value.asBytes.ptr,
                    sourceData->value.asBytes.length, error);
        case DPI_NATIVE_TYPE_LOB:
            return dpiVar__setFromLob(var, pos, sourceData->value.asLOB,
                    error);
        case DPI_NATIVE_TYPE_OBJECT:
            return dpiVar__setFromObject(var, pos, sourceData->value.asObject,
                    error);
        case DPI_NATIVE_TYPE_STMT:
            return dpiVar__setFromStmt(var, pos, sourceData->value.asStmt,
                    error);
        case DPI_NATIVE_TYPE_ROWID:
            return dpiVar__setFromRowid(var, pos, sourceData->value.asRowid,
                    error);
        default:
            memcpy(targetData, sourceData, sizeof(dpiData));
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__defineCallback() [INTERNAL]
//   Callback which runs during OCI statement execution and allocates the
// buffers required as well as provides that information to the OCI. This is
// intended for handling string and raw columns for which the size is unknown.
// These include LONG, LONG RAW and retrieving CLOB and BLOB as bytes, rather
// than use the LOB API.
//-----------------------------------------------------------------------------
int32_t dpiVar__defineCallback(dpiVar *var, UNUSED void *defnp, uint32_t iter,
        void **bufpp, uint32_t **alenpp, UNUSED uint8_t *piecep, void **indpp,
        uint16_t **rcodepp)
{
    dpiDynamicBytesChunk *chunk;
    dpiDynamicBytes *bytes;

    // allocate more chunks, if necessary
    bytes = &var->buffer.dynamicBytes[iter];
    if (bytes->numChunks == bytes->allocatedChunks &&
            dpiVar__allocateChunks(bytes, var->error) < 0)
        return DPI_OCI_ERROR;

    // allocate memory for the chunk, if needed
    chunk = &bytes->chunks[bytes->numChunks];
    if (!chunk->ptr) {
        chunk->allocatedLength = DPI_DYNAMIC_BYTES_CHUNK_SIZE;
        if (dpiUtils__allocateMemory(1, chunk->allocatedLength, 0,
                "allocate chunk", (void**) &chunk->ptr, var->error) < 0)
            return DPI_OCI_ERROR;
    }

    // return chunk to OCI
    bytes->numChunks++;
    chunk->length = chunk->allocatedLength;
    *bufpp = chunk->ptr;
    *alenpp = &chunk->length;
    *indpp = &(var->buffer.indicator[iter]);
    *rcodepp = NULL;
    return DPI_OCI_CONTINUE;
}


//-----------------------------------------------------------------------------
// dpiVar__extendedPreFetch() [INTERNAL]
//   Perform any necessary actions prior to fetching data.
//-----------------------------------------------------------------------------
int dpiVar__extendedPreFetch(dpiVar *var, dpiVarBuffer *buffer,
        dpiError *error)
{
    dpiRowid *rowid;
    dpiData *data;
    dpiStmt *stmt;
    dpiLob *lob;
    uint32_t i;

    if (var->isDynamic) {
        for (i = 0; i < buffer->maxArraySize; i++)
            buffer->dynamicBytes[i].numChunks = 0;
        return DPI_SUCCESS;
    }

    switch (var->type->oracleTypeNum) {
        case DPI_ORACLE_TYPE_STMT:
            for (i = 0; i < buffer->maxArraySize; i++) {
                data = &buffer->externalData[i];
                if (buffer->references[i].asStmt) {
                    dpiGen__setRefCount(buffer->references[i].asStmt,
                            error, -1);
                    buffer->references[i].asStmt = NULL;
                }
                buffer->data.asStmt[i] = NULL;
                data->value.asStmt = NULL;
                if (dpiStmt__allocate(var->conn, 0, &stmt, error) < 0)
                    return DPI_FAILURE;
                if (dpiOci__handleAlloc(var->env->handle, &stmt->handle,
                        DPI_OCI_HTYPE_STMT, "allocate statement", error) < 0) {
                    dpiStmt__free(stmt, error);
                    return DPI_FAILURE;
                }
                if (dpiHandleList__addHandle(var->conn->openStmts, stmt,
                        &stmt->openSlotNum, error) < 0) {
                    dpiOci__handleFree(stmt->handle, DPI_OCI_HTYPE_STMT);
                    stmt->handle = NULL;
                    dpiStmt__free(stmt, error);
                    return DPI_FAILURE;
                }
                buffer->references[i].asStmt = stmt;
                stmt->isOwned = 1;
                buffer->data.asStmt[i] = stmt->handle;
                data->value.asStmt = stmt;
            }
            break;
        case DPI_ORACLE_TYPE_CLOB:
        case DPI_ORACLE_TYPE_BLOB:
        case DPI_ORACLE_TYPE_NCLOB:
        case DPI_ORACLE_TYPE_BFILE:
            for (i = 0; i < buffer->maxArraySize; i++) {
                data = &buffer->externalData[i];
                if (buffer->references[i].asLOB) {
                    dpiGen__setRefCount(buffer->references[i].asLOB,
                            error, -1);
                    buffer->references[i].asLOB = NULL;
                }
                buffer->data.asLobLocator[i] = NULL;
                data->value.asLOB = NULL;
                if (dpiLob__allocate(var->conn, var->type, &lob, error) < 0)
                    return DPI_FAILURE;
                buffer->references[i].asLOB = lob;
                buffer->data.asLobLocator[i] = lob->locator;
                data->value.asLOB = lob;
                if (buffer->dynamicBytes &&
                        dpiOci__lobCreateTemporary(lob, error) < 0)
                    return DPI_FAILURE;
            }
            break;
        case DPI_ORACLE_TYPE_ROWID:
            for (i = 0; i < buffer->maxArraySize; i++) {
                data = &buffer->externalData[i];
                if (buffer->references[i].asRowid) {
                    dpiGen__setRefCount(buffer->references[i].asRowid,
                            error, -1);
                    buffer->references[i].asRowid = NULL;
                }
                buffer->data.asRowid[i] = NULL;
                data->value.asRowid = NULL;
                if (dpiRowid__allocate(var->conn, &rowid, error) < 0)
                    return DPI_FAILURE;
                buffer->references[i].asRowid = rowid;
                buffer->data.asRowid[i] = rowid->handle;
                data->value.asRowid = rowid;
            }
            break;
        case DPI_ORACLE_TYPE_OBJECT:
            for (i = 0; i < buffer->maxArraySize; i++) {
                data = &buffer->externalData[i];
                if (buffer->references[i].asObject) {
                    dpiGen__setRefCount(buffer->references[i].asObject,
                            error, -1);
                    buffer->references[i].asObject = NULL;
                }
                buffer->data.asObject[i] = NULL;
                buffer->objectIndicator[i] = NULL;
                data->value.asObject = NULL;
            }
            break;
        default:
            break;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__finalizeBuffer() [INTERNAL]
//   Finalize buffer used for passing data to/from Oracle.
//-----------------------------------------------------------------------------
static void dpiVar__finalizeBuffer(dpiVar *var, dpiVarBuffer *buffer,
        dpiError *error)
{
    dpiDynamicBytes *dynBytes;
    uint32_t i, j;

    // free any descriptors that were created
    switch (var->type->oracleTypeNum) {
        case DPI_ORACLE_TYPE_TIMESTAMP:
            dpiOci__arrayDescriptorFree(&buffer->data.asTimestamp[0],
                    DPI_OCI_DTYPE_TIMESTAMP);
            break;
        case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
            dpiOci__arrayDescriptorFree(&buffer->data.asTimestamp[0],
                    DPI_OCI_DTYPE_TIMESTAMP_TZ);
            break;
        case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
            dpiOci__arrayDescriptorFree(&buffer->data.asTimestamp[0],
                    DPI_OCI_DTYPE_TIMESTAMP_LTZ);
            break;
        case DPI_ORACLE_TYPE_INTERVAL_DS:
            dpiOci__arrayDescriptorFree(&buffer->data.asInterval[0],
                    DPI_OCI_DTYPE_INTERVAL_DS);
            break;
        case DPI_ORACLE_TYPE_INTERVAL_YM:
            dpiOci__arrayDescriptorFree(&buffer->data.asInterval[0],
                    DPI_OCI_DTYPE_INTERVAL_YM);
            break;
        default:
            break;
    }

    // release any references that were created
    if (buffer->references) {
        for (i = 0; i < buffer->maxArraySize; i++) {
            if (buffer->references[i].asHandle) {
                dpiGen__setRefCount(buffer->references[i].asHandle, error, -1);
                buffer->references[i].asHandle = NULL;
            }
        }
        dpiUtils__freeMemory(buffer->references);
        buffer->references = NULL;
    }

    // free any dynamic buffers
    if (buffer->dynamicBytes) {
        for (i = 0; i < buffer->maxArraySize; i++) {
            dynBytes = &buffer->dynamicBytes[i];
            if (dynBytes->allocatedChunks > 0) {
                for (j = 0; j < dynBytes->allocatedChunks; j++) {
                    if (dynBytes->chunks[j].ptr) {
                        dpiUtils__freeMemory(dynBytes->chunks[j].ptr);
                        dynBytes->chunks[j].ptr = NULL;
                    }
                }
                dpiUtils__freeMemory(dynBytes->chunks);
                dynBytes->allocatedChunks = 0;
                dynBytes->chunks = NULL;
            }
        }
        dpiUtils__freeMemory(buffer->dynamicBytes);
        buffer->dynamicBytes = NULL;
    }

    // free other memory allocated
    if (buffer->indicator) {
        dpiUtils__freeMemory(buffer->indicator);
        buffer->indicator = NULL;
    }
    if (buffer->returnCode) {
        dpiUtils__freeMemory(buffer->returnCode);
        buffer->returnCode = NULL;
    }
    if (buffer->actualLength16) {
        dpiUtils__freeMemory(buffer->actualLength16);
        buffer->actualLength16 = NULL;
    }
    if (buffer->actualLength32) {
        dpiUtils__freeMemory(buffer->actualLength32);
        buffer->actualLength32 = NULL;
    }
    if (buffer->externalData) {
        dpiUtils__freeMemory(buffer->externalData);
        buffer->externalData = NULL;
    }
    if (buffer->data.asRaw) {
        dpiUtils__freeMemory(buffer->data.asRaw);
        buffer->data.asRaw = NULL;
    }
    if (buffer->objectIndicator) {
        dpiUtils__freeMemory(buffer->objectIndicator);
        buffer->objectIndicator = NULL;
    }
    if (buffer->tempBuffer) {
        dpiUtils__freeMemory(buffer->tempBuffer);
        buffer->tempBuffer = NULL;
    }
}


//-----------------------------------------------------------------------------
// dpiVar__free() [INTERNAL]
//   Free the memory associated with the variable.
//-----------------------------------------------------------------------------
void dpiVar__free(dpiVar *var, dpiError *error)
{
    uint32_t i;

    dpiVar__finalizeBuffer(var, &var->buffer, error);
    if (var->dynBindBuffers) {
        for (i = 0; i < var->buffer.maxArraySize; i++)
            dpiVar__finalizeBuffer(var, &var->dynBindBuffers[i], error);
        dpiUtils__freeMemory(var->dynBindBuffers);
        var->dynBindBuffers = NULL;
    }
    if (var->objectType) {
        dpiGen__setRefCount(var->objectType, error, -1);
        var->objectType = NULL;
    }
    if (var->conn) {
        dpiGen__setRefCount(var->conn, error, -1);
        var->conn = NULL;
    }
    dpiUtils__freeMemory(var);
}


//-----------------------------------------------------------------------------
// dpiVar__getValue() [PRIVATE]
//   Returns the contents of the variable in the type specified, if possible.
//-----------------------------------------------------------------------------
int dpiVar__getValue(dpiVar *var, dpiVarBuffer *buffer, uint32_t pos,
        int inFetch, dpiError *error)
{
    dpiOracleTypeNum oracleTypeNum;
    dpiBytes *bytes;
    dpiData *data;
    uint32_t i;

    // check for dynamic binds first; if they exist, process them instead
    if (var->dynBindBuffers && buffer == &var->buffer) {
        buffer = &var->dynBindBuffers[pos];
        for (i = 0; i < buffer->maxArraySize; i++) {
            if (dpiVar__getValue(var, buffer, i, inFetch, error) < 0)
                return DPI_FAILURE;
        }
        return DPI_SUCCESS;
    }


    // check for a NULL value; for objects the indicator is elsewhere
    data = &buffer->externalData[pos];
    if (!buffer->objectIndicator)
        data->isNull = (buffer->indicator[pos] == DPI_OCI_IND_NULL);
    else if (buffer->objectIndicator[pos])
        data->isNull = (*((int16_t*) buffer->objectIndicator[pos]) ==
                DPI_OCI_IND_NULL);
    else data->isNull = 1;
    if (data->isNull)
        return DPI_SUCCESS;

    // check return code for variable length data
    if (buffer->returnCode) {
        if (buffer->returnCode[pos] != 0) {
            dpiError__set(error, "check return code", DPI_ERR_COLUMN_FETCH,
                    pos, buffer->returnCode[pos]);
            error->buffer->code = buffer->returnCode[pos];
            return DPI_FAILURE;
        }
    }

    // for 11g, dynamic lengths are 32-bit whereas static lengths are 16-bit
    if (buffer->actualLength16 && buffer->actualLength32)
        buffer->actualLength16[pos] = (uint16_t) buffer->actualLength32[pos];

    // transform the various types
    oracleTypeNum = var->type->oracleTypeNum;
    switch (var->nativeTypeNum) {
        case DPI_NATIVE_TYPE_INT64:
        case DPI_NATIVE_TYPE_UINT64:
            switch (oracleTypeNum) {
                case DPI_ORACLE_TYPE_NATIVE_INT:
                    data->value.asInt64 = buffer->data.asInt64[pos];
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_NATIVE_UINT:
                    data->value.asUint64 = buffer->data.asUint64[pos];
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_NUMBER:
                    if (var->nativeTypeNum == DPI_NATIVE_TYPE_INT64)
                        return dpiDataBuffer__fromOracleNumberAsInteger(
                                &data->value, error,
                                &buffer->data.asNumber[pos]);
                    return dpiDataBuffer__fromOracleNumberAsUnsignedInteger(
                            &data->value, error, &buffer->data.asNumber[pos]);
                default:
                    break;
            }
            break;
        case DPI_NATIVE_TYPE_DOUBLE:
            switch (oracleTypeNum) {
                case DPI_ORACLE_TYPE_NUMBER:
                    return dpiDataBuffer__fromOracleNumberAsDouble(
                            &data->value, error, &buffer->data.asNumber[pos]);
                case DPI_ORACLE_TYPE_NATIVE_DOUBLE:
                    data->value.asDouble = buffer->data.asDouble[pos];
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_TIMESTAMP:
                case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
                case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
                    return dpiDataBuffer__fromOracleTimestampAsDouble(
                            &data->value, var->env, error,
                            buffer->data.asTimestamp[pos]);
                default:
                    break;
            }
            break;
        case DPI_NATIVE_TYPE_BYTES:
            bytes = &data->value.asBytes;
            switch (oracleTypeNum) {
                case DPI_ORACLE_TYPE_VARCHAR:
                case DPI_ORACLE_TYPE_NVARCHAR:
                case DPI_ORACLE_TYPE_CHAR:
                case DPI_ORACLE_TYPE_NCHAR:
                case DPI_ORACLE_TYPE_ROWID:
                case DPI_ORACLE_TYPE_RAW:
                case DPI_ORACLE_TYPE_LONG_VARCHAR:
                case DPI_ORACLE_TYPE_LONG_RAW:
                    if (buffer->dynamicBytes)
                        return dpiVar__setBytesFromDynamicBytes(bytes,
                                &buffer->dynamicBytes[pos], error);
                    if (buffer->actualLength16)
                        bytes->length = buffer->actualLength16[pos];
                    else bytes->length = buffer->actualLength32[pos];
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_CLOB:
                case DPI_ORACLE_TYPE_NCLOB:
                case DPI_ORACLE_TYPE_BLOB:
                case DPI_ORACLE_TYPE_BFILE:
                    return dpiVar__setBytesFromLob(bytes,
                            &buffer->dynamicBytes[pos],
                            buffer->references[pos].asLOB, error);
                case DPI_ORACLE_TYPE_NUMBER:
                    bytes->length = DPI_NUMBER_AS_TEXT_CHARS;
                    if (var->env->charsetId == DPI_CHARSET_ID_UTF16)
                        bytes->length *= 2;
                    return dpiDataBuffer__fromOracleNumberAsText(&data->value,
                            var->env, error, &buffer->data.asNumber[pos]);
                default:
                    break;
            }
            break;
        case DPI_NATIVE_TYPE_FLOAT:
            data->value.asFloat = buffer->data.asFloat[pos];
            break;
        case DPI_NATIVE_TYPE_TIMESTAMP:
            if (oracleTypeNum == DPI_ORACLE_TYPE_DATE)
                return dpiDataBuffer__fromOracleDate(&data->value,
                        &buffer->data.asDate[pos]);
            return dpiDataBuffer__fromOracleTimestamp(&data->value, var->env,
                    error, buffer->data.asTimestamp[pos],
                    oracleTypeNum != DPI_ORACLE_TYPE_TIMESTAMP);
            break;
        case DPI_NATIVE_TYPE_INTERVAL_DS:
            return dpiDataBuffer__fromOracleIntervalDS(&data->value, var->env,
                    error, buffer->data.asInterval[pos]);
        case DPI_NATIVE_TYPE_INTERVAL_YM:
            return dpiDataBuffer__fromOracleIntervalYM(&data->value, var->env,
                    error, buffer->data.asInterval[pos]);
        case DPI_NATIVE_TYPE_OBJECT:
            data->value.asObject = NULL;
            if (!buffer->references[pos].asObject) {
                if (dpiObject__allocate(var->objectType,
                        buffer->data.asObject[pos],
                        buffer->objectIndicator[pos], NULL,
                        &buffer->references[pos].asObject, error) < 0)
                    return DPI_FAILURE;
                if (inFetch && var->objectType->isCollection)
                    buffer->references[pos].asObject->freeIndicator = 1;
            }
            data->value.asObject = buffer->references[pos].asObject;
            break;
        case DPI_NATIVE_TYPE_STMT:
            data->value.asStmt = buffer->references[pos].asStmt;
            break;
        case DPI_NATIVE_TYPE_BOOLEAN:
            data->value.asBoolean = buffer->data.asBoolean[pos];
            break;
        default:
            break;
    }
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__inBindCallback() [INTERNAL]
//   Callback which runs during OCI statement execution and provides buffers to
// OCI for binding data IN. This is not used with DML returning so this method
// does nothing useful except satisfy OCI requirements.
//-----------------------------------------------------------------------------
int32_t dpiVar__inBindCallback(dpiVar *var, UNUSED void *bindp,
        UNUSED uint32_t iter, uint32_t index, void **bufpp, uint32_t *alenp,
        uint8_t *piecep, void **indpp)
{
    dpiDynamicBytes *dynBytes;

    if (var->isDynamic) {
        dynBytes = &var->buffer.dynamicBytes[iter];
        if (dynBytes->allocatedChunks == 0) {
            *bufpp = NULL;
            *alenp = 0;
        } else {
            *bufpp = dynBytes->chunks->ptr;
            *alenp = dynBytes->chunks->length;
        }
    } else {
        dpiVar__assignCallbackBuffer(var, &var->buffer, iter, bufpp);
        if (var->buffer.actualLength16)
            *alenp = var->buffer.actualLength16[iter];
        else if (var->buffer.actualLength32)
            *alenp = var->buffer.actualLength32[iter];
        else *alenp = var->type->sizeInBytes;
    }
    *piecep = DPI_OCI_ONE_PIECE;
    if (var->buffer.objectIndicator)
        *indpp = var->buffer.objectIndicator[iter];
    else *indpp = &var->buffer.indicator[iter];
    return DPI_OCI_CONTINUE;
}


//-----------------------------------------------------------------------------
// dpiVar__initBuffer() [INTERNAL]
//   Initialize buffers necessary for passing data to/from Oracle.
//-----------------------------------------------------------------------------
static int dpiVar__initBuffer(dpiVar *var, dpiVarBuffer *buffer,
        dpiError *error)
{
    uint32_t i, tempBufferSize = 0;
    unsigned long long dataLength;
    dpiBytes *bytes;

    // initialize dynamic buffers for dynamic variables
    if (var->isDynamic) {
        if (dpiUtils__allocateMemory(buffer->maxArraySize,
                sizeof(dpiDynamicBytes), 1, "allocate dynamic bytes",
                (void**) &buffer->dynamicBytes, error) < 0)
            return DPI_FAILURE;

    // for all other variables, validate length and allocate buffers
    } else {
        dataLength = (unsigned long long) buffer->maxArraySize *
                (unsigned long long) var->sizeInBytes;
        if (dataLength > INT_MAX)
            return dpiError__set(error, "check max array size",
                    DPI_ERR_ARRAY_SIZE_TOO_BIG, buffer->maxArraySize);
        if (dpiUtils__allocateMemory(1, (size_t) dataLength, 0,
                "allocate buffer", (void**) &buffer->data.asRaw, error) < 0)
            return DPI_FAILURE;
    }

    // allocate the indicator for the variable
    // ensure all values start out as null
    if (!buffer->indicator) {
        if (dpiUtils__allocateMemory(buffer->maxArraySize, sizeof(int16_t), 0,
                "allocate indicator", (void**) &buffer->indicator, error) < 0)
            return DPI_FAILURE;
        for (i = 0; i < buffer->maxArraySize; i++)
            buffer->indicator[i] = DPI_OCI_IND_NULL;
    }

    // allocate the actual length buffers for all but dynamic bytes which are
    // handled differently; ensure actual length starts out as maximum value
    if (!var->isDynamic && !buffer->actualLength16 &&
            !buffer->actualLength32) {
        if (var->env->versionInfo->versionNum < 12 && buffer == &var->buffer) {
            if (dpiUtils__allocateMemory(buffer->maxArraySize,
                    sizeof(uint16_t), 0, "allocate actual length",
                    (void**) &buffer->actualLength16, error) < 0)
                return DPI_FAILURE;
            for (i = 0; i < buffer->maxArraySize; i++)
                buffer->actualLength16[i] = (uint16_t) var->sizeInBytes;
        } else {
            if (dpiUtils__allocateMemory(buffer->maxArraySize,
                    sizeof(uint32_t), 0, "allocate actual length",
                    (void**) &buffer->actualLength32, error) < 0)
                return DPI_FAILURE;
            for (i = 0; i < buffer->maxArraySize; i++)
                buffer->actualLength32[i] = var->sizeInBytes;
        }
    }

    // for variable length data, also allocate the return code array
    if (var->type->defaultNativeTypeNum == DPI_NATIVE_TYPE_BYTES &&
            !var->isDynamic && !buffer->returnCode) {
        if (dpiUtils__allocateMemory(buffer->maxArraySize, sizeof(uint16_t), 0,
                "allocate return code", (void**) &buffer->returnCode,
                error) < 0)
            return DPI_FAILURE;
    }

    // for numbers transferred to/from Oracle as bytes, allocate an additional
    // set of buffers
    if (var->type->oracleTypeNum == DPI_ORACLE_TYPE_NUMBER &&
            var->nativeTypeNum == DPI_NATIVE_TYPE_BYTES) {
        tempBufferSize = DPI_NUMBER_AS_TEXT_CHARS;
        if (var->env->charsetId == DPI_CHARSET_ID_UTF16)
            tempBufferSize *= 2;
        if (!buffer->tempBuffer) {
            if (dpiUtils__allocateMemory(buffer->maxArraySize, tempBufferSize,
                    0, "allocate temp buffer", (void**) &buffer->tempBuffer,
                    error) < 0)
                return DPI_FAILURE;
        }
    }

    // allocate the external data array, if needed
    if (!buffer->externalData) {
        if (dpiUtils__allocateMemory(buffer->maxArraySize, sizeof(dpiData), 1,
                "allocate external data", (void**) &buffer->externalData,
                error) < 0)
            return DPI_FAILURE;
        for (i = 0; i < buffer->maxArraySize; i++)
            buffer->externalData[i].isNull = 1;
    }

    // for bytes transfers, set encoding and pointers for small strings
    if (var->nativeTypeNum == DPI_NATIVE_TYPE_BYTES) {
        for (i = 0; i < buffer->maxArraySize; i++) {
            bytes = &buffer->externalData[i].value.asBytes;
            if (var->type->charsetForm == DPI_SQLCS_IMPLICIT)
                bytes->encoding = var->env->encoding;
            else bytes->encoding = var->env->nencoding;
            if (buffer->tempBuffer)
                bytes->ptr = buffer->tempBuffer + i * tempBufferSize;
            else if (!var->isDynamic && !buffer->dynamicBytes)
                bytes->ptr = buffer->data.asBytes + i * var->sizeInBytes;
        }
    }

    // create array of references, if applicable
    if (var->type->requiresPreFetch && !var->isDynamic) {
        if (dpiUtils__allocateMemory(buffer->maxArraySize,
                sizeof(dpiReferenceBuffer), 1, "allocate references",
                (void**) &buffer->references, error) < 0)
            return DPI_FAILURE;
    }

    // perform variable specific initialization
    switch (var->type->oracleTypeNum) {
        case DPI_ORACLE_TYPE_TIMESTAMP:
            return dpiOci__arrayDescriptorAlloc(var->env->handle,
                    &buffer->data.asTimestamp[0], DPI_OCI_DTYPE_TIMESTAMP,
                    buffer->maxArraySize, error);
        case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
            return dpiOci__arrayDescriptorAlloc(var->env->handle,
                    &buffer->data.asTimestamp[0], DPI_OCI_DTYPE_TIMESTAMP_TZ,
                    buffer->maxArraySize, error);
        case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
            return dpiOci__arrayDescriptorAlloc(var->env->handle,
                    &buffer->data.asTimestamp[0], DPI_OCI_DTYPE_TIMESTAMP_LTZ,
                    buffer->maxArraySize, error);
        case DPI_ORACLE_TYPE_INTERVAL_DS:
            return dpiOci__arrayDescriptorAlloc(var->env->handle,
                    &buffer->data.asInterval[0], DPI_OCI_DTYPE_INTERVAL_DS,
                    buffer->maxArraySize, error);
        case DPI_ORACLE_TYPE_INTERVAL_YM:
            return dpiOci__arrayDescriptorAlloc(var->env->handle,
                    &buffer->data.asInterval[0], DPI_OCI_DTYPE_INTERVAL_YM,
                    buffer->maxArraySize, error);
            break;
        case DPI_ORACLE_TYPE_CLOB:
        case DPI_ORACLE_TYPE_BLOB:
        case DPI_ORACLE_TYPE_NCLOB:
        case DPI_ORACLE_TYPE_BFILE:
        case DPI_ORACLE_TYPE_STMT:
        case DPI_ORACLE_TYPE_ROWID:
            return dpiVar__extendedPreFetch(var, buffer, error);
        case DPI_ORACLE_TYPE_OBJECT:
            if (!var->objectType)
                return dpiError__set(error, "check object type",
                        DPI_ERR_NO_OBJECT_TYPE);
            if (dpiUtils__allocateMemory(buffer->maxArraySize, sizeof(void*),
                    0, "allocate object indicator",
                    (void**) &buffer->objectIndicator, error) < 0)
                return DPI_FAILURE;
            return dpiVar__extendedPreFetch(var, buffer, error);
        default:
            break;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__outBindCallback() [INTERNAL]
//   Callback which runs during OCI statement execution and allocates the
// buffers required as well as provides that information to the OCI. This is
// intended for use with DML returning only.
//-----------------------------------------------------------------------------
int32_t dpiVar__outBindCallback(dpiVar *var, void *bindp, UNUSED uint32_t iter,
        uint32_t index, void **bufpp, uint32_t **alenpp, uint8_t *piecep,
        void **indpp, uint16_t **rcodepp)
{
    dpiDynamicBytesChunk *chunk;
    uint32_t numRowsReturned;
    dpiDynamicBytes *bytes;
    dpiVarBuffer *buffer;

    // determine which variable buffer to use
    if (!var->dynBindBuffers) {
        if (dpiUtils__allocateMemory(var->buffer.maxArraySize,
                sizeof(dpiVarBuffer), 1, "allocate DML returning buffers",
                (void**) &var->dynBindBuffers, var->error) < 0)
            return DPI_FAILURE;
    }
    buffer = &var->dynBindBuffers[iter];

    // special processing during first value returned for each iteration
    if (index == 0) {

        // determine number of rows returned
        if (dpiOci__attrGet(bindp, DPI_OCI_HTYPE_BIND, &numRowsReturned, 0,
                DPI_OCI_ATTR_ROWS_RETURNED, "get rows returned",
                var->error) < 0)
            return DPI_OCI_ERROR;

        // reallocate buffers, if needed
        if (numRowsReturned > buffer->maxArraySize) {
            dpiVar__finalizeBuffer(var, buffer, var->error);
            buffer->maxArraySize = numRowsReturned;
            if (dpiVar__initBuffer(var, buffer, var->error) < 0)
                return DPI_OCI_ERROR;
        }

        // set actual array size to number of rows returned
        buffer->actualArraySize = numRowsReturned;

    }

    // handle dynamically allocated strings (multiple piece)
    // index is the current index into the chunks
    if (var->isDynamic) {

        // allocate more chunks, if necessary
        bytes = &buffer->dynamicBytes[index];
        if (*piecep == DPI_OCI_ONE_PIECE)
            bytes->numChunks = 0;
        if (bytes->numChunks == bytes->allocatedChunks &&
                dpiVar__allocateChunks(bytes, var->error) < 0)
            return DPI_OCI_ERROR;

        // allocate memory for the chunk, if needed
        chunk = &bytes->chunks[bytes->numChunks];
        if (!chunk->ptr) {
            chunk->allocatedLength = DPI_DYNAMIC_BYTES_CHUNK_SIZE;
            if (dpiUtils__allocateMemory(1, chunk->allocatedLength, 0,
                    "allocate chunk", (void**) &chunk->ptr, var->error) < 0)
                return DPI_OCI_ERROR;
        }

        // return chunk to OCI
        bytes->numChunks++;
        chunk->length = chunk->allocatedLength;
        *bufpp = chunk->ptr;
        *alenpp = &chunk->length;
        *indpp = &(buffer->indicator[index]);
        *rcodepp = NULL;

    // handle normally allocated variables (one piece)
    } else {

        *piecep = DPI_OCI_ONE_PIECE;
        if (dpiVar__setValue(var, buffer, index, &buffer->externalData[index],
                var->error) < 0)
            return DPI_OCI_ERROR;
        dpiVar__assignCallbackBuffer(var, buffer, index, bufpp);
        if (buffer->actualLength32 || buffer->actualLength16) {
            if (!buffer->actualLength32) {
                if (dpiUtils__allocateMemory(buffer->maxArraySize,
                        sizeof(uint32_t), 1, "allocate 11g lengths",
                        (void**) &buffer->actualLength32, var->error) < 0)
                    return DPI_OCI_ERROR;
            }
            buffer->actualLength32[index] = var->sizeInBytes;
            *alenpp = &(buffer->actualLength32[index]);
        } else if (*alenpp && var->type->sizeInBytes)
            **alenpp = var->type->sizeInBytes;
        if (buffer->objectIndicator)
            *indpp = buffer->objectIndicator[index];
        else *indpp = &(buffer->indicator[index]);
        if (buffer->returnCode)
            *rcodepp = &buffer->returnCode[index];

    }

    return DPI_OCI_CONTINUE;
}


//-----------------------------------------------------------------------------
// dpiVar__setBytesFromDynamicBytes() [PRIVATE]
//   Set the pointer and length in the dpiBytes structure to the values
// retrieved from the database. At this point, if multiple chunks exist, they
// are combined into one.
//-----------------------------------------------------------------------------
static int dpiVar__setBytesFromDynamicBytes(dpiBytes *bytes,
        dpiDynamicBytes *dynBytes, dpiError *error)
{
    uint32_t i, totalAllocatedLength;

    // if only one chunk is available, make use of it
    if (dynBytes->numChunks == 1) {
        bytes->ptr = dynBytes->chunks->ptr;
        bytes->length = dynBytes->chunks->length;
        return DPI_SUCCESS;
    }

    // determine total allocated size of all chunks
    totalAllocatedLength = 0;
    for (i = 0; i < dynBytes->numChunks; i++)
        totalAllocatedLength += dynBytes->chunks[i].allocatedLength;

    // allocate new memory consolidating all of the chunks
    if (dpiUtils__allocateMemory(1, totalAllocatedLength, 0,
            "allocate consolidated chunk", (void**) &bytes->ptr, error) < 0)
        return DPI_FAILURE;

    // copy memory from chunks to consolidated chunk
    bytes->length = 0;
    for (i = 0; i < dynBytes->numChunks; i++) {
        memcpy(bytes->ptr + bytes->length, dynBytes->chunks[i].ptr,
                dynBytes->chunks[i].length);
        bytes->length += dynBytes->chunks[i].length;
        dpiUtils__freeMemory(dynBytes->chunks[i].ptr);
        dynBytes->chunks[i].ptr = NULL;
        dynBytes->chunks[i].length = 0;
        dynBytes->chunks[i].allocatedLength = 0;
    }

    // populate first chunk with consolidated information
    dynBytes->numChunks = 1;
    dynBytes->chunks->ptr = bytes->ptr;
    dynBytes->chunks->length = bytes->length;
    dynBytes->chunks->allocatedLength = totalAllocatedLength;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setBytesFromLob() [PRIVATE]
//   Populate the dynamic bytes structure with the data from the LOB and then
// populate the bytes structure.
//-----------------------------------------------------------------------------
static int dpiVar__setBytesFromLob(dpiBytes *bytes, dpiDynamicBytes *dynBytes,
        dpiLob *lob, dpiError *error)
{
    uint64_t length, lengthInBytes, lengthReadInBytes;

    // determine length of LOB in bytes
    if (dpiOci__lobGetLength2(lob, &length, error) < 0)
        return DPI_FAILURE;
    if (lob->type->oracleTypeNum == DPI_ORACLE_TYPE_CLOB)
        lengthInBytes = length * lob->env->maxBytesPerCharacter;
    else if (lob->type->oracleTypeNum == DPI_ORACLE_TYPE_NCLOB)
        lengthInBytes = length * lob->env->nmaxBytesPerCharacter;
    else lengthInBytes = length;

    // ensure there is enough space to store the entire LOB value
    if (lengthInBytes > UINT_MAX)
        return dpiError__set(error, "check max length", DPI_ERR_NOT_SUPPORTED);
    if (dpiVar__allocateDynamicBytes(dynBytes, (uint32_t) lengthInBytes,
            error) < 0)
        return DPI_FAILURE;

    // read data from the LOB
    lengthReadInBytes = lengthInBytes;
    if (length > 0 && dpiLob__readBytes(lob, 1, length, dynBytes->chunks->ptr,
            &lengthReadInBytes, error) < 0)
        return DPI_FAILURE;

    dynBytes->chunks->length = (uint32_t) lengthReadInBytes;
    bytes->ptr = dynBytes->chunks->ptr;
    bytes->length = dynBytes->chunks->length;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setFromBytes() [PRIVATE]
//   Set the value of the variable at the given array position from a byte
// string. The byte string is not retained in any way. A copy will be made into
// buffers allocated by ODPI-C.
//-----------------------------------------------------------------------------
static int dpiVar__setFromBytes(dpiVar *var, uint32_t pos, const char *value,
        uint32_t valueLength, dpiError *error)
{
    dpiData *data = &var->buffer.externalData[pos];
    dpiDynamicBytes *dynBytes;
    dpiBytes *bytes;

    // for internally used LOBs, write the data directly
    if (var->buffer.references) {
        data->isNull = 0;
        return dpiLob__setFromBytes(var->buffer.references[pos].asLOB, value,
                valueLength, error);
    }

    // validate the target can accept the input
    if ((var->buffer.tempBuffer &&
                    var->env->charsetId == DPI_CHARSET_ID_UTF16 &&
                    valueLength > DPI_NUMBER_AS_TEXT_CHARS * 2) ||
            (var->buffer.tempBuffer &&
                    var->env->charsetId != DPI_CHARSET_ID_UTF16 &&
                    valueLength > DPI_NUMBER_AS_TEXT_CHARS) ||
            (!var->buffer.dynamicBytes && !var->buffer.tempBuffer &&
                    valueLength > var->sizeInBytes))
        return dpiError__set(error, "check source length",
                DPI_ERR_BUFFER_SIZE_TOO_SMALL, var->sizeInBytes);

    // for dynamic bytes, allocate space as needed
    bytes = &data->value.asBytes;
    if (var->buffer.dynamicBytes) {
        dynBytes = &var->buffer.dynamicBytes[pos];
        if (dpiVar__allocateDynamicBytes(dynBytes, valueLength, error) < 0)
            return DPI_FAILURE;
        memcpy(dynBytes->chunks->ptr, value, valueLength);
        dynBytes->numChunks = 1;
        dynBytes->chunks->length = valueLength;
        bytes->ptr = dynBytes->chunks->ptr;
        bytes->length = valueLength;

    // for everything else, space has already been allocated
    } else {
        bytes->length = valueLength;
        if (valueLength > 0)
            memcpy(bytes->ptr, value, valueLength);
        if (var->type->sizeInBytes == 0) {
            if (var->buffer.actualLength32)
                var->buffer.actualLength32[pos] = valueLength;
            else if (var->buffer.actualLength16)
                var->buffer.actualLength16[pos] = (uint16_t) valueLength;
        }
        if (var->buffer.returnCode)
            var->buffer.returnCode[pos] = 0;
    }
    data->isNull = 0;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setFromLob() [PRIVATE]
//   Set the value of the variable at the given array position from a LOB.
// A reference to the LOB is retained by the variable.
//-----------------------------------------------------------------------------
static int dpiVar__setFromLob(dpiVar *var, uint32_t pos, dpiLob *lob,
        dpiError *error)
{
    dpiData *data;

    // validate the LOB object
    if (dpiGen__checkHandle(lob, DPI_HTYPE_LOB, "check LOB", error) < 0)
        return DPI_FAILURE;

    // mark the value as not null
    data = &var->buffer.externalData[pos];
    data->isNull = 0;

    // if values are the same, nothing to do
    if (var->buffer.references[pos].asLOB == lob)
        return DPI_SUCCESS;

    // clear original value, if needed
    if (var->buffer.references[pos].asLOB) {
        dpiGen__setRefCount(var->buffer.references[pos].asLOB, error, -1);
        var->buffer.references[pos].asLOB = NULL;
    }

    // add reference to passed object
    dpiGen__setRefCount(lob, error, 1);
    var->buffer.references[pos].asLOB = lob;
    var->buffer.data.asLobLocator[pos] = lob->locator;
    data->value.asLOB = lob;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setFromObject() [PRIVATE]
//   Set the value of the variable at the given array position from an object.
// The variable and position are assumed to be valid at this point. A reference
// to the object is retained by the variable.
//-----------------------------------------------------------------------------
static int dpiVar__setFromObject(dpiVar *var, uint32_t pos, dpiObject *obj,
        dpiError *error)
{
    dpiData *data;

    // validate the object
    if (dpiGen__checkHandle(obj, DPI_HTYPE_OBJECT, "check obj", error) < 0)
        return DPI_FAILURE;
    if (obj->type->tdo != var->objectType->tdo)
        return dpiError__set(error, "check type", DPI_ERR_WRONG_TYPE,
                obj->type->schemaLength, obj->type->schema,
                obj->type->nameLength, obj->type->name,
                var->objectType->schemaLength, var->objectType->schema,
                var->objectType->nameLength, var->objectType->name);

    // mark the value as not null
    data = &var->buffer.externalData[pos];
    data->isNull = 0;

    // if values are the same, nothing to do
    if (var->buffer.references[pos].asObject == obj)
        return DPI_SUCCESS;

    // clear original value, if needed
    if (var->buffer.references[pos].asObject) {
        dpiGen__setRefCount(var->buffer.references[pos].asObject, error, -1);
        var->buffer.references[pos].asObject = NULL;
    }

    // add reference to passed object
    dpiGen__setRefCount(obj, error, 1);
    var->buffer.references[pos].asObject = obj;
    var->buffer.data.asObject[pos] = obj->instance;
    var->buffer.objectIndicator[pos] = obj->indicator;
    data->value.asObject = obj;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setFromRowid() [PRIVATE]
//   Set the value of the variable at the given array position from a rowid.
// A reference to the rowid is retained by the variable.
//-----------------------------------------------------------------------------
static int dpiVar__setFromRowid(dpiVar *var, uint32_t pos, dpiRowid *rowid,
        dpiError *error)
{
    dpiData *data;

    // validate the rowid
    if (dpiGen__checkHandle(rowid, DPI_HTYPE_ROWID, "check rowid", error) < 0)
        return DPI_FAILURE;

    // mark the value as not null
    data = &var->buffer.externalData[pos];
    data->isNull = 0;

    // if values are the same, nothing to do
    if (var->buffer.references[pos].asRowid == rowid)
        return DPI_SUCCESS;

    // clear original value, if needed
    if (var->buffer.references[pos].asRowid) {
        dpiGen__setRefCount(var->buffer.references[pos].asRowid, error, -1);
        var->buffer.references[pos].asRowid = NULL;
    }

    // add reference to passed object
    dpiGen__setRefCount(rowid, error, 1);
    var->buffer.references[pos].asRowid = rowid;
    var->buffer.data.asRowid[pos] = rowid->handle;
    data->value.asRowid = rowid;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setFromStmt() [PRIVATE]
//   Set the value of the variable at the given array position from a
// statement. A reference to the statement is retained by the variable.
//-----------------------------------------------------------------------------
static int dpiVar__setFromStmt(dpiVar *var, uint32_t pos, dpiStmt *stmt,
        dpiError *error)
{
    dpiData *data;
    uint32_t i;

    // validate the statement
    if (dpiGen__checkHandle(stmt, DPI_HTYPE_STMT, "check stmt", error) < 0)
        return DPI_FAILURE;

    // prevent attempts to bind a statement to itself
    for (i = 0; i < stmt->numBindVars; i++) {
        if (stmt->bindVars[i].var == var)
            return dpiError__set(error, "bind to self", DPI_ERR_NOT_SUPPORTED);
    }

    // mark the value as not null
    data = &var->buffer.externalData[pos];
    data->isNull = 0;

    // if values are the same, nothing to do
    if (var->buffer.references[pos].asStmt == stmt)
        return DPI_SUCCESS;

    // clear original value, if needed
    if (var->buffer.references[pos].asStmt) {
        dpiGen__setRefCount(var->buffer.references[pos].asStmt, error, -1);
        var->buffer.references[pos].asStmt = NULL;
    }

    // add reference to passed object
    dpiGen__setRefCount(stmt, error, 1);
    var->buffer.references[pos].asStmt = stmt;
    var->buffer.data.asStmt[pos] = stmt->handle;
    data->value.asStmt = stmt;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__setValue() [PRIVATE]
//   Sets the contents of the variable using the type specified, if possible.
//-----------------------------------------------------------------------------
int dpiVar__setValue(dpiVar *var, dpiVarBuffer *buffer, uint32_t pos,
        dpiData *data, dpiError *error)
{
    dpiOracleTypeNum oracleTypeNum;
    dpiObject *obj;

    // if value is null, no need to proceed further
    // however, when binding objects a value MUST be present or OCI will
    // segfault!
    if (data->isNull) {
        buffer->indicator[pos] = DPI_OCI_IND_NULL;
        if (buffer->objectIndicator && !buffer->data.asObject[pos]) {
            if (dpiObject__allocate(var->objectType, NULL, NULL, NULL, &obj,
                    error) < 0)
                return DPI_FAILURE;
            buffer->references[pos].asObject = obj;
            data->value.asObject = obj;
            buffer->data.asObject[pos] = obj->instance;
            buffer->objectIndicator[pos] = obj->indicator;
            if (buffer->objectIndicator[pos])
                *((int16_t*) buffer->objectIndicator[pos]) = DPI_OCI_IND_NULL;
        }
        return DPI_SUCCESS;
    }

    // transform the various types
    buffer->indicator[pos] = DPI_OCI_IND_NOTNULL;
    oracleTypeNum = var->type->oracleTypeNum;
    switch (var->nativeTypeNum) {
        case DPI_NATIVE_TYPE_INT64:
        case DPI_NATIVE_TYPE_UINT64:
            switch (oracleTypeNum) {
                case DPI_ORACLE_TYPE_NATIVE_INT:
                    buffer->data.asInt64[pos] = data->value.asInt64;
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_NATIVE_UINT:
                    buffer->data.asUint64[pos] = data->value.asUint64;
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_NUMBER:
                    if (var->nativeTypeNum == DPI_NATIVE_TYPE_INT64)
                        return dpiDataBuffer__toOracleNumberFromInteger(
                                &data->value, error,
                                &buffer->data.asNumber[pos]);
                    return dpiDataBuffer__toOracleNumberFromUnsignedInteger(
                            &data->value, error, &buffer->data.asNumber[pos]);
                default:
                    break;
            }
            break;
        case DPI_NATIVE_TYPE_FLOAT:
            buffer->data.asFloat[pos] = data->value.asFloat;
            return DPI_SUCCESS;
        case DPI_NATIVE_TYPE_DOUBLE:
            switch (oracleTypeNum) {
                case DPI_ORACLE_TYPE_NATIVE_DOUBLE:
                    buffer->data.asDouble[pos] = data->value.asDouble;
                    return DPI_SUCCESS;
                case DPI_ORACLE_TYPE_NUMBER:
                    return dpiDataBuffer__toOracleNumberFromDouble(
                            &data->value, error, &buffer->data.asNumber[pos]);
                case DPI_ORACLE_TYPE_TIMESTAMP:
                case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
                case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
                    return dpiDataBuffer__toOracleTimestampFromDouble(
                            &data->value, var->env, error,
                            buffer->data.asTimestamp[pos]);
                default:
                    break;
            }
            break;
        case DPI_NATIVE_TYPE_BYTES:
            if (oracleTypeNum == DPI_ORACLE_TYPE_NUMBER)
                return dpiDataBuffer__toOracleNumberFromText(&data->value,
                        var->env, error, &buffer->data.asNumber[pos]);
            if (buffer->actualLength32)
                buffer->actualLength32[pos] = data->value.asBytes.length;
            else if (buffer->actualLength16)
                buffer->actualLength16[pos] =
                        (uint16_t) data->value.asBytes.length;
            if (buffer->returnCode)
                buffer->returnCode[pos] = 0;
            break;
        case DPI_NATIVE_TYPE_TIMESTAMP:
            if (oracleTypeNum == DPI_ORACLE_TYPE_DATE)
                return dpiDataBuffer__toOracleDate(&data->value,
                        &buffer->data.asDate[pos]);
            else if (oracleTypeNum == DPI_ORACLE_TYPE_TIMESTAMP)
                return dpiDataBuffer__toOracleTimestamp(&data->value,
                        var->env, error, buffer->data.asTimestamp[pos], 0);
            else if (oracleTypeNum == DPI_ORACLE_TYPE_TIMESTAMP_TZ ||
                    oracleTypeNum == DPI_ORACLE_TYPE_TIMESTAMP_LTZ)
                return dpiDataBuffer__toOracleTimestamp(&data->value,
                        var->env, error, buffer->data.asTimestamp[pos], 1);
            break;
        case DPI_NATIVE_TYPE_INTERVAL_DS:
            return dpiDataBuffer__toOracleIntervalDS(&data->value, var->env,
                    error, buffer->data.asInterval[pos]);
        case DPI_NATIVE_TYPE_INTERVAL_YM:
            return dpiDataBuffer__toOracleIntervalYM(&data->value, var->env,
                    error, buffer->data.asInterval[pos]);
        case DPI_NATIVE_TYPE_BOOLEAN:
            buffer->data.asBoolean[pos] = data->value.asBoolean;
            return DPI_SUCCESS;
        default:
            break;
    }
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiVar__validateTypes() [PRIVATE]
//   Validate that the Oracle type and the native type are compatible with
// each other when the native type is not already the default native type.
//-----------------------------------------------------------------------------
static int dpiVar__validateTypes(const dpiOracleType *oracleType,
        dpiNativeTypeNum nativeTypeNum, dpiError *error)
{
    switch (oracleType->oracleTypeNum) {
        case DPI_ORACLE_TYPE_TIMESTAMP:
        case DPI_ORACLE_TYPE_TIMESTAMP_TZ:
        case DPI_ORACLE_TYPE_TIMESTAMP_LTZ:
            if (nativeTypeNum == DPI_NATIVE_TYPE_DOUBLE)
                return DPI_SUCCESS;
            break;
        case DPI_ORACLE_TYPE_NUMBER:
            if (nativeTypeNum == DPI_NATIVE_TYPE_INT64 ||
                    nativeTypeNum == DPI_NATIVE_TYPE_UINT64 ||
                    nativeTypeNum == DPI_NATIVE_TYPE_BYTES)
                return DPI_SUCCESS;
            break;
        default:
            break;
    }
    return dpiError__set(error, "validate types", DPI_ERR_UNHANDLED_CONVERSION,
            oracleType->oracleTypeNum, nativeTypeNum);
}


//-----------------------------------------------------------------------------
// dpiVar_addRef() [PUBLIC]
//   Add a reference to the variable.
//-----------------------------------------------------------------------------
int dpiVar_addRef(dpiVar *var)
{
    return dpiGen__addRef(var, DPI_HTYPE_VAR, __func__);
}


//-----------------------------------------------------------------------------
// dpiVar_copyData() [PUBLIC]
//   Copy the data from the source variable to the target variable at the given
// array position. The variables must use the same native type. If the
// variables contain variable length data, the source length must not exceed
// the target allocated memory.
//-----------------------------------------------------------------------------
int dpiVar_copyData(dpiVar *var, uint32_t pos, dpiVar *sourceVar,
        uint32_t sourcePos)
{
    dpiData *sourceData;
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (dpiGen__checkHandle(sourceVar, DPI_HTYPE_VAR, "check source var",
            &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (sourcePos >= sourceVar->buffer.maxArraySize) {
        dpiError__set(&error, "check source size",
                DPI_ERR_INVALID_ARRAY_POSITION, sourcePos,
                sourceVar->buffer.maxArraySize);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    if (var->nativeTypeNum != sourceVar->nativeTypeNum) {
        dpiError__set(&error, "check types match", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    sourceData = &sourceVar->buffer.externalData[sourcePos];
    status = dpiVar__copyData(var, pos, sourceData, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_getNumElementsInArray() [PUBLIC]
//   Return the actual number of elements in the array. This value is only
// relevant if the variable is bound as an array.
//-----------------------------------------------------------------------------
int dpiVar_getNumElementsInArray(dpiVar *var, uint32_t *numElements)
{
    dpiError error;

    if (dpiGen__startPublicFn(var, DPI_HTYPE_VAR, __func__, 0, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(var, numElements)
    if (var->dynBindBuffers)
        *numElements = var->dynBindBuffers->actualArraySize;
    else *numElements = var->buffer.actualArraySize;
    return dpiGen__endPublicFn(var, DPI_SUCCESS, &error);
}

//-----------------------------------------------------------------------------
// dpiVar_getReturnedData() [PUBLIC]
//   Return a pointer to the array of dpiData structures allocated for the
// given row that have been returned by a DML returning statement. The number
// of returned rows is also provided. If the bind variable had no data
// returned, the number of rows returned will be 0 and the pointer to the array
// of dpiData structures will be NULL. This will also be the case if the
// variable was only bound IN or was not bound to a DML returning statement.
// There is no way to differentiate between the two.
//-----------------------------------------------------------------------------
int dpiVar_getReturnedData(dpiVar *var, uint32_t pos, uint32_t *numElements,
        dpiData **data)
{
    dpiError error;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(var, numElements)
    DPI_CHECK_PTR_NOT_NULL(var, data)
    if (var->dynBindBuffers) {
        *numElements = var->dynBindBuffers[pos].actualArraySize;
        *data = var->dynBindBuffers[pos].externalData;
    } else {
        *numElements = 0;
        *data = NULL;
    }
    return dpiGen__endPublicFn(var, DPI_SUCCESS, &error);
}



//-----------------------------------------------------------------------------
// dpiVar_getSizeInBytes() [PUBLIC]
//   Returns the size in bytes of the buffer allocated for the variable.
//-----------------------------------------------------------------------------
int dpiVar_getSizeInBytes(dpiVar *var, uint32_t *sizeInBytes)
{
    dpiError error;

    if (dpiGen__startPublicFn(var, DPI_HTYPE_VAR, __func__, 0, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(var, sizeInBytes)
    *sizeInBytes = var->sizeInBytes;
    return dpiGen__endPublicFn(var, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_release() [PUBLIC]
//   Release a reference to the variable.
//-----------------------------------------------------------------------------
int dpiVar_release(dpiVar *var)
{
    return dpiGen__release(var, DPI_HTYPE_VAR, __func__);
}


//-----------------------------------------------------------------------------
// dpiVar_setFromBytes() [PUBLIC]
//   Set the value of the variable at the given array position from a byte
// string. Checks on the array position, the size of the string and the type of
// variable will be made. The byte string is not retained in any way. A copy
// will be made into buffers allocated by ODPI-C.
//-----------------------------------------------------------------------------
int dpiVar_setFromBytes(dpiVar *var, uint32_t pos, const char *value,
        uint32_t valueLength)
{
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(var, value)
    if (var->nativeTypeNum != DPI_NATIVE_TYPE_BYTES &&
            var->nativeTypeNum != DPI_NATIVE_TYPE_LOB) {
        dpiError__set(&error, "native type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    if (valueLength > DPI_MAX_VAR_BUFFER_SIZE) {
        dpiError__set(&error, "check buffer", DPI_ERR_BUFFER_SIZE_TOO_LARGE,
                valueLength, DPI_MAX_VAR_BUFFER_SIZE);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    status = dpiVar__setFromBytes(var, pos, value, valueLength, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_setFromLob() [PUBLIC]
//   Set the value of the variable at the given array position from a LOB.
// Checks on the array position and the validity of the passed handle. A
// reference to the LOB is retained by the variable.
//-----------------------------------------------------------------------------
int dpiVar_setFromLob(dpiVar *var, uint32_t pos, dpiLob *lob)
{
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (var->nativeTypeNum != DPI_NATIVE_TYPE_LOB) {
        dpiError__set(&error, "native type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    status = dpiVar__setFromLob(var, pos, lob, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_setFromObject() [PUBLIC]
//   Set the value of the variable at the given array position from an object.
// Checks on the array position and the validity of the passed handle. A
// reference to the object is retained by the variable.
//-----------------------------------------------------------------------------
int dpiVar_setFromObject(dpiVar *var, uint32_t pos, dpiObject *obj)
{
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (var->nativeTypeNum != DPI_NATIVE_TYPE_OBJECT) {
        dpiError__set(&error, "native type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    status = dpiVar__setFromObject(var, pos, obj, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_setFromRowid() [PUBLIC]
//   Set the value of the variable at the given array position from a rowid.
// Checks on the array position and the validity of the passed handle. A
// reference to the rowid is retained by the variable.
//-----------------------------------------------------------------------------
int dpiVar_setFromRowid(dpiVar *var, uint32_t pos, dpiRowid *rowid)
{
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (var->nativeTypeNum != DPI_NATIVE_TYPE_ROWID) {
        dpiError__set(&error, "native type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    status = dpiVar__setFromRowid(var, pos, rowid, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_setFromStmt() [PUBLIC]
//   Set the value of the variable at the given array position from a
// statement. Checks on the array position and the validity of the passed
// handle. A reference to the statement is retained by the variable.
//-----------------------------------------------------------------------------
int dpiVar_setFromStmt(dpiVar *var, uint32_t pos, dpiStmt *stmt)
{
    dpiError error;
    int status;

    if (dpiVar__checkArraySize(var, pos, __func__, 1, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (var->nativeTypeNum != DPI_NATIVE_TYPE_STMT) {
        dpiError__set(&error, "native type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    status = dpiVar__setFromStmt(var, pos, stmt, &error);
    return dpiGen__endPublicFn(var, status, &error);
}


//-----------------------------------------------------------------------------
// dpiVar_setNumElementsInArray() [PUBLIC]
//   Set the number of elements in the array (different from the number of
// allocated elements).
//-----------------------------------------------------------------------------
int dpiVar_setNumElementsInArray(dpiVar *var, uint32_t numElements)
{
    dpiError error;

    if (dpiGen__startPublicFn(var, DPI_HTYPE_VAR, __func__, 0, &error) < 0)
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    if (numElements > var->buffer.maxArraySize) {
        dpiError__set(&error, "check num elements",
                DPI_ERR_ARRAY_SIZE_TOO_SMALL, var->buffer.maxArraySize);
        return dpiGen__endPublicFn(var, DPI_FAILURE, &error);
    }
    var->buffer.actualArraySize = numElements;
    return dpiGen__endPublicFn(var, DPI_SUCCESS, &error);
}

