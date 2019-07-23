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
// dpiStmt.c
//   Implementation of statements (cursors).
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// forward declarations of internal functions only used in this file
static int dpiStmt__getQueryInfo(dpiStmt *stmt, uint32_t pos,
        dpiQueryInfo *info, dpiError *error);
static int dpiStmt__getQueryInfoFromParam(dpiStmt *stmt, void *param,
        dpiQueryInfo *info, dpiError *error);
static int dpiStmt__postFetch(dpiStmt *stmt, dpiError *error);
static int dpiStmt__beforeFetch(dpiStmt *stmt, dpiError *error);
static int dpiStmt__reExecute(dpiStmt *stmt, uint32_t numIters,
        uint32_t mode, dpiError *error);


//-----------------------------------------------------------------------------
// dpiStmt__allocate() [INTERNAL]
//   Create a new statement object and return it. In case of error NULL is
// returned.
//-----------------------------------------------------------------------------
int dpiStmt__allocate(dpiConn *conn, int scrollable, dpiStmt **stmt,
        dpiError *error)
{
    dpiStmt *tempStmt;

    *stmt = NULL;
    if (dpiGen__allocate(DPI_HTYPE_STMT, conn->env, (void**) &tempStmt,
            error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(conn, error, 1);
    tempStmt->conn = conn;
    tempStmt->fetchArraySize = DPI_DEFAULT_FETCH_ARRAY_SIZE;
    tempStmt->scrollable = scrollable;
    *stmt = tempStmt;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__bind() [INTERNAL]
//   Bind the variable to the statement using either a position or a name. A
// reference to the variable will be retained.
//-----------------------------------------------------------------------------
static int dpiStmt__bind(dpiStmt *stmt, dpiVar *var, int addReference,
        uint32_t pos, const char *name, uint32_t nameLength, dpiError *error)
{
    dpiBindVar *bindVars, *entry = NULL;
    int found, dynamicBind, status;
    void *bindHandle = NULL;
    uint32_t i;

    // a zero length name is not supported
    if (pos == 0 && nameLength == 0)
        return dpiError__set(error, "bind zero length name",
                DPI_ERR_NOT_SUPPORTED);

    // prevent attempts to bind a statement to itself
    if (var->type->oracleTypeNum == DPI_ORACLE_TYPE_STMT) {
        for (i = 0; i < var->buffer.maxArraySize; i++) {
            if (var->buffer.externalData[i].value.asStmt == stmt) {
                return dpiError__set(error, "bind to self",
                        DPI_ERR_NOT_SUPPORTED);
            }
        }
    }

    // check to see if the bind position or name has already been bound
    found = 0;
    for (i = 0; i < stmt->numBindVars; i++) {
        entry = &stmt->bindVars[i];
        if (entry->pos == pos && entry->nameLength == nameLength) {
            if (nameLength > 0 && strncmp(entry->name, name, nameLength) != 0)
                continue;
            found = 1;
            break;
        }
    }

    // if already found, use that entry
    if (found) {

        // if already bound, no need to bind a second time
        if (entry->var == var)
            return DPI_SUCCESS;

        // otherwise, release previously bound variable, if applicable
        else if (entry->var) {
            dpiGen__setRefCount(entry->var, error, -1);
            entry->var = NULL;
        }

    // if not found, add to the list of bind variables
    } else {

        // allocate memory for additional bind variables, if needed
        if (stmt->numBindVars == stmt->allocatedBindVars) {
            if (dpiUtils__allocateMemory(stmt->allocatedBindVars + 8,
                    sizeof(dpiBindVar), 1, "allocate bind vars",
                    (void**) &bindVars, error) < 0)
                return DPI_FAILURE;
            if (stmt->bindVars) {
                for (i = 0; i < stmt->numBindVars; i++)
                    bindVars[i] = stmt->bindVars[i];
                dpiUtils__freeMemory(stmt->bindVars);
            }
            stmt->bindVars = bindVars;
            stmt->allocatedBindVars += 8;
        }

        // add to the list of bind variables
        entry = &stmt->bindVars[stmt->numBindVars];
        entry->var = NULL;
        entry->pos = pos;
        if (name) {
            if (dpiUtils__allocateMemory(1, nameLength, 0,
                    "allocate memory for name", (void**) &entry->name,
                    error) < 0)
                return DPI_FAILURE;
            entry->nameLength = nameLength;
            memcpy( (void*) entry->name, name, nameLength);
        }
        stmt->numBindVars++;

    }

    // for PL/SQL where the maxSize is greater than 32K, adjust the variable
    // so that LOBs are used internally
    if (var->isDynamic && (stmt->statementType == DPI_STMT_TYPE_BEGIN ||
            stmt->statementType == DPI_STMT_TYPE_DECLARE ||
            stmt->statementType == DPI_STMT_TYPE_CALL)) {
        if (dpiVar__convertToLob(var, error) < 0)
            return DPI_FAILURE;
    }

    // perform actual bind
    if (addReference)
        dpiGen__setRefCount(var, error, 1);
    entry->var = var;
    dynamicBind = stmt->isReturning || var->isDynamic;
    if (pos > 0) {
        if (stmt->env->versionInfo->versionNum < 12)
            status = dpiOci__bindByPos(stmt, &bindHandle, pos, dynamicBind,
                    var, error);
        else status = dpiOci__bindByPos2(stmt, &bindHandle, pos, dynamicBind,
                var, error);
    } else {
        if (stmt->env->versionInfo->versionNum < 12)
            status = dpiOci__bindByName(stmt, &bindHandle, name,
                    (int32_t) nameLength, dynamicBind, var, error);
        else status = dpiOci__bindByName2(stmt, &bindHandle, name,
                (int32_t) nameLength, dynamicBind, var, error);
    }

    // attempt to improve message "ORA-01036: illegal variable name/number"
    if (status < 0) {
        if (error->buffer->code == 1036) {
            if (stmt->statementType == DPI_STMT_TYPE_CREATE ||
                    stmt->statementType == DPI_STMT_TYPE_DROP ||
                    stmt->statementType == DPI_STMT_TYPE_ALTER)
                dpiError__set(error, error->buffer->action,
                        DPI_ERR_NO_BIND_VARS_IN_DDL);
        }
        return DPI_FAILURE;
    }

    // set the charset form if applicable
    if (var->type->charsetForm != DPI_SQLCS_IMPLICIT) {
        if (dpiOci__attrSet(bindHandle, DPI_OCI_HTYPE_BIND,
                (void*) &var->type->charsetForm, 0, DPI_OCI_ATTR_CHARSET_FORM,
                "set charset form", error) < 0)
            return DPI_FAILURE;
    }

    // set the max data size, if applicable
    if (var->type->sizeInBytes == 0 && !var->isDynamic) {
        if (dpiOci__attrSet(bindHandle, DPI_OCI_HTYPE_BIND,
                (void*) &var->sizeInBytes, 0, DPI_OCI_ATTR_MAXDATA_SIZE,
                "set max data size", error) < 0)
            return DPI_FAILURE;
    }

    // bind object, if applicable
    if (var->buffer.objectIndicator &&
            dpiOci__bindObject(var, bindHandle, error) < 0)
        return DPI_FAILURE;

    // setup dynamic bind, if applicable
    if (dynamicBind && dpiOci__bindDynamic(var, bindHandle, error) < 0)
        return DPI_FAILURE;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__check() [INTERNAL]
//   Determine if the statement is open and available for use.
//-----------------------------------------------------------------------------
static int dpiStmt__check(dpiStmt *stmt, const char *fnName, dpiError *error)
{
    if (dpiGen__startPublicFn(stmt, DPI_HTYPE_STMT, fnName, 1, error) < 0)
        return DPI_FAILURE;
    if (!stmt->handle)
        return dpiError__set(error, "check closed", DPI_ERR_STMT_CLOSED);
    if (dpiConn__checkConnected(stmt->conn, error) < 0)
        return DPI_FAILURE;
    if (stmt->statementType == 0 && dpiStmt__init(stmt, error) < 0)
        return DPI_FAILURE;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__clearBatchErrors() [INTERNAL]
//   Clear the batch errors associated with the statement.
//-----------------------------------------------------------------------------
static void dpiStmt__clearBatchErrors(dpiStmt *stmt)
{
    if (stmt->batchErrors) {
        dpiUtils__freeMemory(stmt->batchErrors);
        stmt->batchErrors = NULL;
    }
    stmt->numBatchErrors = 0;
}


//-----------------------------------------------------------------------------
// dpiStmt__clearBindVars() [INTERNAL]
//   Clear the bind variables associated with the statement.
//-----------------------------------------------------------------------------
static void dpiStmt__clearBindVars(dpiStmt *stmt, dpiError *error)
{
    uint32_t i;

    if (stmt->bindVars) {
        for (i = 0; i < stmt->numBindVars; i++) {
            dpiGen__setRefCount(stmt->bindVars[i].var, error, -1);
            if (stmt->bindVars[i].name)
                dpiUtils__freeMemory( (void*) stmt->bindVars[i].name);
        }
        dpiUtils__freeMemory(stmt->bindVars);
        stmt->bindVars = NULL;
    }
    stmt->numBindVars = 0;
    stmt->allocatedBindVars = 0;
}


//-----------------------------------------------------------------------------
// dpiStmt__clearQueryVars() [INTERNAL]
//   Clear the query variables associated with the statement.
//-----------------------------------------------------------------------------
static void dpiStmt__clearQueryVars(dpiStmt *stmt, dpiError *error)
{
    uint32_t i;

    if (stmt->queryVars) {
        for (i = 0; i < stmt->numQueryVars; i++) {
            if (stmt->queryVars[i]) {
                dpiGen__setRefCount(stmt->queryVars[i], error, -1);
                stmt->queryVars[i] = NULL;
            }
            if (stmt->queryInfo[i].typeInfo.objectType) {
                dpiGen__setRefCount(stmt->queryInfo[i].typeInfo.objectType,
                        error, -1);
                stmt->queryInfo[i].typeInfo.objectType = NULL;
            }
        }
        dpiUtils__freeMemory(stmt->queryVars);
        stmt->queryVars = NULL;
    }
    if (stmt->queryInfo) {
        dpiUtils__freeMemory(stmt->queryInfo);
        stmt->queryInfo = NULL;
    }
    stmt->numQueryVars = 0;
}


//-----------------------------------------------------------------------------
// dpiStmt__close() [INTERNAL]
//   Internal method used for closing the statement. If the statement is marked
// as needing to be dropped from the statement cache that is done as well. This
// is called from dpiStmt_close() where errors are expected to be propagated
// and from dpiStmt__free() where errors are ignored.
//-----------------------------------------------------------------------------
int dpiStmt__close(dpiStmt *stmt, const char *tag, uint32_t tagLength,
        int propagateErrors, dpiError *error)
{
    int closing, status = DPI_SUCCESS;

    // determine whether statement is already being closed and if not, mark
    // statement as being closed; this MUST be done while holding the lock (if
    // in threaded mode) to avoid race conditions!
    if (stmt->env->threaded)
        dpiMutex__acquire(stmt->env->mutex);
    closing = stmt->closing;
    stmt->closing = 1;
    if (stmt->env->threaded)
        dpiMutex__release(stmt->env->mutex);

    // if statement is already being closed, nothing needs to be done
    if (closing)
        return DPI_SUCCESS;

    // perform actual work of closing statement
    dpiStmt__clearBatchErrors(stmt);
    dpiStmt__clearBindVars(stmt, error);
    dpiStmt__clearQueryVars(stmt, error);
    if (stmt->handle) {
        if (!stmt->conn->deadSession && stmt->conn->handle) {
            if (stmt->isOwned)
                dpiOci__handleFree(stmt->handle, DPI_OCI_HTYPE_STMT);
            else status = dpiOci__stmtRelease(stmt, tag, tagLength,
                    propagateErrors, error);
        }
        if (!stmt->conn->closing)
            dpiHandleList__removeHandle(stmt->conn->openStmts,
                    stmt->openSlotNum);
        stmt->handle = NULL;
    }

    // if actual close fails, reset closing flag; again, this must be done
    // while holding the lock (if in threaded mode) in order to avoid race
    // conditions!
    if (status < 0) {
        if (stmt->env->threaded)
            dpiMutex__acquire(stmt->env->mutex);
        stmt->closing = 0;
        if (stmt->env->threaded)
            dpiMutex__release(stmt->env->mutex);
    }

    return status;
}


//-----------------------------------------------------------------------------
// dpiStmt__createBindVar() [INTERNAL]
//   Create a bind variable given a value to bind.
//-----------------------------------------------------------------------------
static int dpiStmt__createBindVar(dpiStmt *stmt,
        dpiNativeTypeNum nativeTypeNum, dpiData *data, dpiVar **var,
        uint32_t pos, const char *name, uint32_t nameLength, dpiError *error)
{
    dpiOracleTypeNum oracleTypeNum;
    dpiObjectType *objType;
    dpiData *varData;
    dpiVar *tempVar;
    uint32_t size;

    // determine the type (and size) of bind variable to create
    size = 0;
    objType = NULL;
    switch (nativeTypeNum) {
        case DPI_NATIVE_TYPE_INT64:
        case DPI_NATIVE_TYPE_UINT64:
        case DPI_NATIVE_TYPE_FLOAT:
        case DPI_NATIVE_TYPE_DOUBLE:
            oracleTypeNum = DPI_ORACLE_TYPE_NUMBER;
            break;
        case DPI_NATIVE_TYPE_BYTES:
            oracleTypeNum = DPI_ORACLE_TYPE_VARCHAR;
            size = data->value.asBytes.length;
            break;
        case DPI_NATIVE_TYPE_TIMESTAMP:
            oracleTypeNum = DPI_ORACLE_TYPE_TIMESTAMP;
            break;
        case DPI_NATIVE_TYPE_INTERVAL_DS:
            oracleTypeNum = DPI_ORACLE_TYPE_INTERVAL_DS;
            break;
        case DPI_NATIVE_TYPE_INTERVAL_YM:
            oracleTypeNum = DPI_ORACLE_TYPE_INTERVAL_YM;
            break;
        case DPI_NATIVE_TYPE_OBJECT:
            oracleTypeNum = DPI_ORACLE_TYPE_OBJECT;
            if (data->value.asObject)
                objType = data->value.asObject->type;
            break;
        case DPI_NATIVE_TYPE_ROWID:
            oracleTypeNum = DPI_ORACLE_TYPE_ROWID;
            break;
        case DPI_NATIVE_TYPE_BOOLEAN:
            oracleTypeNum = DPI_ORACLE_TYPE_BOOLEAN;
            break;
        default:
            return dpiError__set(error, "create bind var",
                    DPI_ERR_UNHANDLED_CONVERSION, 0, nativeTypeNum);
    }

    // create the variable and set its value
    if (dpiVar__allocate(stmt->conn, oracleTypeNum, nativeTypeNum, 1, size, 1,
            0, objType, &tempVar, &varData, error) < 0)
        return DPI_FAILURE;

    // copy value from source to target data
    if (dpiVar__copyData(tempVar, 0, data, error) < 0)
        return DPI_FAILURE;

    // bind variable to statement
    if (dpiStmt__bind(stmt, tempVar, 0, pos, name, nameLength, error) < 0) {
        dpiVar__free(tempVar, error);
        return DPI_FAILURE;
    }

    *var = tempVar;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__createQueryVars() [INTERNAL]
//   Create space for the number of query variables required to support the
// query.
//-----------------------------------------------------------------------------
static int dpiStmt__createQueryVars(dpiStmt *stmt, dpiError *error)
{
    uint32_t numQueryVars, i;

    // determine number of query variables
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            (void*) &numQueryVars, 0, DPI_OCI_ATTR_PARAM_COUNT,
            "get parameter count", error) < 0)
        return DPI_FAILURE;

    // clear the previous query vars if the number has changed
    if (stmt->numQueryVars > 0 && stmt->numQueryVars != numQueryVars)
        dpiStmt__clearQueryVars(stmt, error);

    // allocate space for the query vars, if needed
    if (numQueryVars != stmt->numQueryVars) {
        if (dpiUtils__allocateMemory(numQueryVars, sizeof(dpiVar*), 1,
                "allocate query vars", (void**) &stmt->queryVars, error) < 0)
            return DPI_FAILURE;
        if (dpiUtils__allocateMemory(numQueryVars, sizeof(dpiQueryInfo), 1,
                "allocate query info", (void**) &stmt->queryInfo, error) < 0) {
            dpiStmt__clearQueryVars(stmt, error);
            return DPI_FAILURE;
        }
        stmt->numQueryVars = numQueryVars;
        for (i = 0; i < numQueryVars; i++) {
            if (dpiStmt__getQueryInfo(stmt, i + 1, &stmt->queryInfo[i],
                    error) < 0) {
                dpiStmt__clearQueryVars(stmt, error);
                return DPI_FAILURE;
            }
        }
    }

    // indicate start of fetch
    stmt->bufferRowIndex = stmt->fetchArraySize;
    stmt->hasRowsToFetch = 1;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__define() [INTERNAL]
//   Define the variable that will accept output from the statement in the
// specified column. At this point the statement, position and variable are all
// assumed to be valid.
//-----------------------------------------------------------------------------
static int dpiStmt__define(dpiStmt *stmt, uint32_t pos, dpiVar *var,
        dpiError *error)
{
    void *defineHandle = NULL;
    dpiQueryInfo *queryInfo;

    // no need to perform define if variable is unchanged
    if (stmt->queryVars[pos - 1] == var)
        return DPI_SUCCESS;

    // for objects, the type specified must match the type in the database
    queryInfo = &stmt->queryInfo[pos - 1];
    if (var->objectType && queryInfo->typeInfo.objectType &&
            var->objectType->tdo != queryInfo->typeInfo.objectType->tdo)
        return dpiError__set(error, "check type", DPI_ERR_WRONG_TYPE,
                var->objectType->schemaLength, var->objectType->schema,
                var->objectType->nameLength, var->objectType->name,
                queryInfo->typeInfo.objectType->schemaLength,
                queryInfo->typeInfo.objectType->schema,
                queryInfo->typeInfo.objectType->nameLength,
                queryInfo->typeInfo.objectType->name);

    // perform the define
    if (stmt->env->versionInfo->versionNum < 12) {
        if (dpiOci__defineByPos(stmt, &defineHandle, pos, var, error) < 0)
            return DPI_FAILURE;
    } else {
        if (dpiOci__defineByPos2(stmt, &defineHandle, pos, var, error) < 0)
            return DPI_FAILURE;
    }

    // set the charset form if applicable
    if (var->type->charsetForm != DPI_SQLCS_IMPLICIT) {
        if (dpiOci__attrSet(defineHandle, DPI_OCI_HTYPE_DEFINE,
                (void*) &var->type->charsetForm, 0, DPI_OCI_ATTR_CHARSET_FORM,
                "set charset form", error) < 0)
            return DPI_FAILURE;
    }

    // define objects, if applicable
    if (var->buffer.objectIndicator && dpiOci__defineObject(var, defineHandle,
            error) < 0)
        return DPI_FAILURE;

    // register callback for dynamic defines
    if (var->isDynamic && dpiOci__defineDynamic(var, defineHandle, error) < 0)
        return DPI_FAILURE;

    // remove previous variable and retain new one
    if (stmt->queryVars[pos - 1])
        dpiGen__setRefCount(stmt->queryVars[pos - 1], error, -1);
    dpiGen__setRefCount(var, error, 1);
    stmt->queryVars[pos - 1] = var;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__execute() [INTERNAL]
//   Internal execution of statement.
//-----------------------------------------------------------------------------
static int dpiStmt__execute(dpiStmt *stmt, uint32_t numIters,
        uint32_t mode, int reExecute, dpiError *error)
{
    uint32_t prefetchSize, i, j;
    dpiData *data;
    dpiVar *var;

    // for all bound variables, transfer data from dpiData structure to Oracle
    // buffer structures
    for (i = 0; i < stmt->numBindVars; i++) {
        var = stmt->bindVars[i].var;
        if (var->isArray && numIters > 1)
            return dpiError__set(error, "bind array var",
                    DPI_ERR_ARRAY_VAR_NOT_SUPPORTED);
        for (j = 0; j < var->buffer.maxArraySize; j++) {
            data = &var->buffer.externalData[j];
            if (dpiVar__setValue(var, &var->buffer, j, data, error) < 0)
                return DPI_FAILURE;
            if (var->dynBindBuffers)
                var->dynBindBuffers[j].actualArraySize = 0;
        }
        if (stmt->isReturning || var->isDynamic)
            var->error = error;
    }

    // for queries, set the OCI prefetch to a fixed value; this prevents an
    // additional round trip for single row fetches while avoiding the overhead
    // of copying from the OCI prefetch buffer to our own buffers for larger
    // fetches
    if (stmt->statementType == DPI_STMT_TYPE_SELECT) {
        prefetchSize = DPI_PREFETCH_ROWS_DEFAULT;
        if (dpiOci__attrSet(stmt->handle, DPI_OCI_HTYPE_STMT, &prefetchSize,
                sizeof(prefetchSize), DPI_OCI_ATTR_PREFETCH_ROWS,
                "set prefetch rows", error) < 0)
            return DPI_FAILURE;
    }

    // clear batch errors from any previous execution
    dpiStmt__clearBatchErrors(stmt);

    // adjust mode for scrollable cursors
    if (stmt->scrollable)
        mode |= DPI_OCI_STMT_SCROLLABLE_READONLY;

    // perform execution
    // re-execute statement for ORA-01007: variable not in select list
    // drop statement from cache for all errors (except those which are due to
    // invalid data which may be fixed in subsequent execution)
    if (dpiOci__stmtExecute(stmt, numIters, mode, error) < 0) {
        dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
                &error->buffer->offset, 0, DPI_OCI_ATTR_PARSE_ERROR_OFFSET,
                "set parse offset", error);
        switch (error->buffer->code) {
            case 1007:
                if (reExecute)
                    return dpiStmt__reExecute(stmt, numIters, mode, error);
                stmt->deleteFromCache = 1;
                break;
            case 1:
            case 1400:
            case 1438:
            case 1461:
            case 2290:
            case 2291:
            case 2292:
            case 21525:
                break;
            default:
                stmt->deleteFromCache = 1;
        }
        return DPI_FAILURE;
    }

    // for all bound variables, transfer data from Oracle buffer structures to
    // dpiData structures; OCI doesn't provide a way of knowing if a variable
    // is an out variable so do this for all of them when this is a possibility
    if (stmt->isReturning || stmt->statementType == DPI_STMT_TYPE_BEGIN ||
            stmt->statementType == DPI_STMT_TYPE_DECLARE ||
            stmt->statementType == DPI_STMT_TYPE_CALL) {
        for (i = 0; i < stmt->numBindVars; i++) {
            var = stmt->bindVars[i].var;
            for (j = 0; j < var->buffer.maxArraySize; j++) {
                if (dpiVar__getValue(var, &var->buffer, j, 0, error) < 0)
                    return DPI_FAILURE;
            }
            var->error = NULL;
        }
    }

    // create query variables (if applicable) and reset row count to zero
    if (stmt->statementType == DPI_STMT_TYPE_SELECT) {
        stmt->rowCount = 0;
        if (!(mode & DPI_MODE_EXEC_PARSE_ONLY) &&
                dpiStmt__createQueryVars(stmt, error) < 0)
            return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__fetch() [INTERNAL]
//   Performs the actual fetch from Oracle.
//-----------------------------------------------------------------------------
static int dpiStmt__fetch(dpiStmt *stmt, dpiError *error)
{
    // perform any pre-fetch activities required
    if (dpiStmt__beforeFetch(stmt, error) < 0)
        return DPI_FAILURE;

    // perform fetch
    if (dpiOci__stmtFetch2(stmt, stmt->fetchArraySize, DPI_MODE_FETCH_NEXT, 0,
            error) < 0)
        return DPI_FAILURE;

    // determine the number of rows fetched into buffers
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            &stmt->bufferRowCount, 0, DPI_OCI_ATTR_ROWS_FETCHED,
            "get rows fetched", error) < 0)
        return DPI_FAILURE;

    // set buffer row info
    stmt->bufferMinRow = stmt->rowCount + 1;
    stmt->bufferRowIndex = 0;

    // perform post-fetch activities required
    if (dpiStmt__postFetch(stmt, error) < 0)
        return DPI_FAILURE;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__free() [INTERNAL]
//   Free the memory associated with the statement.
//-----------------------------------------------------------------------------
void dpiStmt__free(dpiStmt *stmt, dpiError *error)
{
    dpiStmt__close(stmt, NULL, 0, 0, error);
    if (stmt->conn) {
        dpiGen__setRefCount(stmt->conn, error, -1);
        stmt->conn = NULL;
    }
    dpiUtils__freeMemory(stmt);
}


//-----------------------------------------------------------------------------
// dpiStmt__getBatchErrors() [INTERNAL]
//   Get batch errors after statement executed with batch errors enabled.
//-----------------------------------------------------------------------------
static int dpiStmt__getBatchErrors(dpiStmt *stmt, dpiError *error)
{
    void *batchErrorHandle, *localErrorHandle;
    dpiError localError;
    int overallStatus;
    int32_t rowOffset;
    uint32_t i;

    // determine the number of batch errors that were found
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            &stmt->numBatchErrors, 0, DPI_OCI_ATTR_NUM_DML_ERRORS,
            "get batch error count", error) < 0)
        return DPI_FAILURE;

    // allocate memory for the batch errors
    if (dpiUtils__allocateMemory(stmt->numBatchErrors, sizeof(dpiErrorBuffer),
            1, "allocate errors", (void**) &stmt->batchErrors, error) < 0) {
        stmt->numBatchErrors = 0;
        return DPI_FAILURE;
    }

    // allocate error handle used for OCIParamGet()
    if (dpiOci__handleAlloc(stmt->env->handle, &localErrorHandle,
            DPI_OCI_HTYPE_ERROR, "allocate parameter error handle",
            error) < 0) {
        dpiStmt__clearBatchErrors(stmt);
        return DPI_FAILURE;
    }

    // allocate error handle used for batch errors
    if (dpiOci__handleAlloc(stmt->env->handle, &batchErrorHandle,
            DPI_OCI_HTYPE_ERROR, "allocate batch error handle", error) < 0) {
        dpiStmt__clearBatchErrors(stmt);
        dpiOci__handleFree(localErrorHandle, DPI_OCI_HTYPE_ERROR);
        return DPI_FAILURE;
    }

    // process each error
    overallStatus = DPI_SUCCESS;
    localError.buffer = error->buffer;
    localError.env = error->env;
    for (i = 0; i < stmt->numBatchErrors; i++) {

        // get error handle for iteration
        if (dpiOci__paramGet(error->handle, DPI_OCI_HTYPE_ERROR,
                &batchErrorHandle, i, "get batch error", error) < 0) {
            overallStatus = dpiError__set(error, "get batch error",
                    DPI_ERR_INVALID_INDEX, i);
            break;
        }

        // determine row offset
        localError.handle = localErrorHandle;
        if (dpiOci__attrGet(batchErrorHandle, DPI_OCI_HTYPE_ERROR, &rowOffset,
                0, DPI_OCI_ATTR_DML_ROW_OFFSET, "get row offset",
                &localError) < 0) {
            overallStatus = dpiError__set(error, "get row offset",
                    DPI_ERR_CANNOT_GET_ROW_OFFSET);
            break;
        }

        // get error message
        localError.buffer = &stmt->batchErrors[i];
        localError.handle = batchErrorHandle;
        dpiError__check(&localError, DPI_OCI_ERROR, stmt->conn,
                "get batch error");
        if (error->buffer->errorNum) {
            overallStatus = DPI_FAILURE;
            break;
        }
        localError.buffer->fnName = error->buffer->fnName;
        localError.buffer->offset = (uint16_t) rowOffset;

    }

    // cleanup
    dpiOci__handleFree(localErrorHandle, DPI_OCI_HTYPE_ERROR);
    dpiOci__handleFree(batchErrorHandle, DPI_OCI_HTYPE_ERROR);
    if (overallStatus < 0)
        dpiStmt__clearBatchErrors(stmt);
    return overallStatus;
}


//-----------------------------------------------------------------------------
// dpiStmt__getQueryInfo() [INTERNAL]
//   Get query information for the position in question.
//-----------------------------------------------------------------------------
static int dpiStmt__getQueryInfo(dpiStmt *stmt, uint32_t pos,
        dpiQueryInfo *info, dpiError *error)
{
    void *param;
    int status;

    // acquire parameter descriptor
    if (dpiOci__paramGet(stmt->handle, DPI_OCI_HTYPE_STMT, &param, pos,
            "get parameter", error) < 0)
        return DPI_FAILURE;

    // acquire information from the parameter descriptor
    status = dpiStmt__getQueryInfoFromParam(stmt, param, info, error);
    dpiOci__descriptorFree(param, DPI_OCI_DTYPE_PARAM);
    return status;
}


//-----------------------------------------------------------------------------
// dpiStmt__getQueryInfoFromParam() [INTERNAL]
//   Get query information from the parameter.
//-----------------------------------------------------------------------------
static int dpiStmt__getQueryInfoFromParam(dpiStmt *stmt, void *param,
        dpiQueryInfo *info, dpiError *error)
{
    uint8_t ociNullOk;

    // aquire name of item
    if (dpiOci__attrGet(param, DPI_OCI_HTYPE_DESCRIBE, (void*) &info->name,
            &info->nameLength, DPI_OCI_ATTR_NAME, "get name", error) < 0)
        return DPI_FAILURE;

    // acquire type information
    if (dpiOracleType__populateTypeInfo(stmt->conn, param,
            DPI_OCI_HTYPE_DESCRIBE, &info->typeInfo, error) < 0)
        return DPI_FAILURE;

    // acquire if column is permitted to be null
    if (dpiOci__attrGet(param, DPI_OCI_HTYPE_DESCRIBE, (void*) &ociNullOk, 0,
            DPI_OCI_ATTR_IS_NULL, "get null ok", error) < 0)
        return DPI_FAILURE;
    info->nullOk = ociNullOk;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__init() [INTERNAL]
//   Initialize the statement for use. This is needed when preparing a
// statement for use and when returning a REF cursor.
//-----------------------------------------------------------------------------
int dpiStmt__init(dpiStmt *stmt, dpiError *error)
{
    // get statement type
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            (void*) &stmt->statementType, 0, DPI_OCI_ATTR_STMT_TYPE,
            "get statement type", error) < 0)
        return DPI_FAILURE;

    // for queries, mark statement as having rows to fetch
    if (stmt->statementType == DPI_STMT_TYPE_SELECT)
        stmt->hasRowsToFetch = 1;

    // otherwise, check if this is a RETURNING statement
    else if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            (void*) &stmt->isReturning, 0, DPI_OCI_ATTR_STMT_IS_RETURNING,
            "get is returning", error) < 0)
        return DPI_FAILURE;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__postFetch() [INTERNAL]
//   Performs the transformations required to convert Oracle data values into
// C data values.
//-----------------------------------------------------------------------------
static int dpiStmt__postFetch(dpiStmt *stmt, dpiError *error)
{
    uint32_t i, j;
    dpiVar *var;

    for (i = 0; i < stmt->numQueryVars; i++) {
        var = stmt->queryVars[i];
        for (j = 0; j < stmt->bufferRowCount; j++) {
            if (dpiVar__getValue(var, &var->buffer, j, 1, error) < 0)
                return DPI_FAILURE;
            if (var->type->requiresPreFetch)
                var->requiresPreFetch = 1;
        }
        var->error = NULL;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__beforeFetch() [INTERNAL]
//   Performs work that needs to be done prior to fetch for each variable. In
// addition, variables are created if they do not already exist. A check is
// also made to ensure that the variable has enough space to support a fetch
// of the requested size.
//-----------------------------------------------------------------------------
static int dpiStmt__beforeFetch(dpiStmt *stmt, dpiError *error)
{
    dpiQueryInfo *queryInfo;
    dpiData *data;
    dpiVar *var;
    uint32_t i;

    if (!stmt->queryInfo && dpiStmt__createQueryVars(stmt, error) < 0)
        return DPI_FAILURE;
    for (i = 0; i < stmt->numQueryVars; i++) {
        var = stmt->queryVars[i];
        if (!var) {
            queryInfo = &stmt->queryInfo[i];
            if (dpiVar__allocate(stmt->conn, queryInfo->typeInfo.oracleTypeNum,
                    queryInfo->typeInfo.defaultNativeTypeNum,
                    stmt->fetchArraySize,
                    queryInfo->typeInfo.clientSizeInBytes, 1, 0,
                    queryInfo->typeInfo.objectType, &var, &data, error) < 0)
                return DPI_FAILURE;
            if (dpiStmt__define(stmt, i + 1, var, error) < 0)
                return DPI_FAILURE;
            dpiGen__setRefCount(var, error, -1);
        }
        var->error = error;
        if (stmt->fetchArraySize > var->buffer.maxArraySize)
            return dpiError__set(error, "check array size",
                    DPI_ERR_ARRAY_SIZE_TOO_SMALL, var->buffer.maxArraySize);
        if (var->requiresPreFetch && dpiVar__extendedPreFetch(var,
                &var->buffer, error) < 0)
            return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiStmt__prepare() [INTERNAL]
//   Prepare a statement for execution.
//-----------------------------------------------------------------------------
int dpiStmt__prepare(dpiStmt *stmt, const char *sql, uint32_t sqlLength,
        const char *tag, uint32_t tagLength, dpiError *error)
{
    if (sql && dpiDebugLevel & DPI_DEBUG_LEVEL_SQL)
        dpiDebug__print("SQL %.*s\n", sqlLength, sql);
    if (dpiOci__stmtPrepare2(stmt, sql, sqlLength, tag, tagLength, error) < 0)
        return DPI_FAILURE;
    if (dpiHandleList__addHandle(stmt->conn->openStmts, stmt,
            &stmt->openSlotNum, error) < 0) {
        dpiOci__stmtRelease(stmt, NULL, 0, 0, error);
        stmt->handle = NULL;
        return DPI_FAILURE;
    }

    return dpiStmt__init(stmt, error);
}


//-----------------------------------------------------------------------------
// dpiStmt__reExecute() [INTERNAL]
//   Re-execute the statement after receiving the error ORA-01007: variable not
// in select list. This takes place when one of the columns in a query is
// dropped, but the original metadata is still being used because the query
// statement was found in the statement cache.
//-----------------------------------------------------------------------------
static int dpiStmt__reExecute(dpiStmt *stmt, uint32_t numIters,
        uint32_t mode, dpiError *error)
{
    void *origHandle, *newHandle;
    uint32_t sqlLength, i;
    dpiError localError;
    dpiBindVar *bindVar;
    dpiVar *var;
    int status;
    char *sql;

    // acquire the statement that was previously prepared; if this cannot be
    // determined, let the original error propagate
    localError.buffer = error->buffer;
    localError.env = error->env;
    localError.handle = error->handle;
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, (void*) &sql,
            &sqlLength, DPI_OCI_ATTR_STATEMENT, "get statement",
            &localError) < 0)
        return DPI_FAILURE;

    // prepare statement a second time before releasing the original statement;
    // release the original statement and delete it from the statement cache
    // so that it does not return with the invalid metadata; again, if this
    // cannot be done, let the original error propagate
    origHandle = stmt->handle;
    status = dpiOci__stmtPrepare2(stmt, sql, sqlLength, NULL, 0, &localError);
    newHandle = stmt->handle;
    stmt->handle = origHandle;
    stmt->deleteFromCache = 1;
    if (dpiOci__stmtRelease(stmt, NULL, 0, 1, &localError) < 0 || status < 0)
        return DPI_FAILURE;
    stmt->handle = newHandle;
    dpiStmt__clearBatchErrors(stmt);
    dpiStmt__clearQueryVars(stmt, error);

    // perform binds
    for (i = 0; i < stmt->numBindVars; i++) {
        bindVar = &stmt->bindVars[i];
        if (!bindVar->var)
            continue;
        var = bindVar->var;
        bindVar->var = NULL;
        if (dpiStmt__bind(stmt, var, 0, bindVar->pos, bindVar->name,
                bindVar->nameLength, error) < 0) {
            dpiGen__setRefCount(var, error, -1);
            return DPI_FAILURE;
        }
    }

    // now re-execute the statement
    return dpiStmt__execute(stmt, numIters, mode, 0, error);
}


//-----------------------------------------------------------------------------
// dpiStmt_addRef() [PUBLIC]
//   Add a reference to the statement.
//-----------------------------------------------------------------------------
int dpiStmt_addRef(dpiStmt *stmt)
{
    return dpiGen__addRef(stmt, DPI_HTYPE_STMT, __func__);
}


//-----------------------------------------------------------------------------
// dpiStmt_bindByName() [PUBLIC]
//   Bind the variable by name.
//-----------------------------------------------------------------------------
int dpiStmt_bindByName(dpiStmt *stmt, const char *name, uint32_t nameLength,
        dpiVar *var)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, name)
    if (dpiGen__checkHandle(var, DPI_HTYPE_VAR, "bind by name", &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    status = dpiStmt__bind(stmt, var, 1, 0, name, nameLength, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_bindByPos() [PUBLIC]
//   Bind the variable by position.
//-----------------------------------------------------------------------------
int dpiStmt_bindByPos(dpiStmt *stmt, uint32_t pos, dpiVar *var)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (dpiGen__checkHandle(var, DPI_HTYPE_VAR, "bind by pos", &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    status = dpiStmt__bind(stmt, var, 1, pos, NULL, 0, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_bindValueByName() [PUBLIC]
//   Create a variable and bind it by name.
//-----------------------------------------------------------------------------
int dpiStmt_bindValueByName(dpiStmt *stmt, const char *name,
        uint32_t nameLength, dpiNativeTypeNum nativeTypeNum, dpiData *data)
{
    dpiVar *var = NULL;
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, name)
    DPI_CHECK_PTR_NOT_NULL(stmt, data)
    if (dpiStmt__createBindVar(stmt, nativeTypeNum, data, &var, 0, name,
            nameLength, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    status = dpiStmt__bind(stmt, var, 1, 0, name, nameLength, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_bindValueByPos() [PUBLIC]
//   Create a variable and bind it by position.
//-----------------------------------------------------------------------------
int dpiStmt_bindValueByPos(dpiStmt *stmt, uint32_t pos,
        dpiNativeTypeNum nativeTypeNum, dpiData *data)
{
    dpiVar *var = NULL;
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, data)
    if (dpiStmt__createBindVar(stmt, nativeTypeNum, data, &var, pos, NULL, 0,
            &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    status = dpiStmt__bind(stmt, var, 1, pos, NULL, 0, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_close() [PUBLIC]
//   Close the statement so that it is no longer usable and all resources have
// been released.
//-----------------------------------------------------------------------------
int dpiStmt_close(dpiStmt *stmt, const char *tag, uint32_t tagLength)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_AND_LENGTH(stmt, tag)
    status = dpiStmt__close(stmt, tag, tagLength, 1, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_define() [PUBLIC]
//   Define the variable that will accept output from the cursor in the
// specified column.
//-----------------------------------------------------------------------------
int dpiStmt_define(dpiStmt *stmt, uint32_t pos, dpiVar *var)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (!stmt->queryInfo && dpiStmt__createQueryVars(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (pos == 0 || pos > stmt->numQueryVars) {
        dpiError__set(&error, "check query position",
                DPI_ERR_QUERY_POSITION_INVALID, pos);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    if (dpiGen__checkHandle(var, DPI_HTYPE_VAR, "check variable", &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    status = dpiStmt__define(stmt, pos, var, &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_defineValue() [PUBLIC]
//   Define the type of data to use for output from the cursor in the specified
// column. This implicitly creates a variable of the specified type and is
// intended for subsequent use by dpiStmt_getQueryValue(), which makes use of
// implicitly created variables.
//-----------------------------------------------------------------------------
int dpiStmt_defineValue(dpiStmt *stmt, uint32_t pos,
        dpiOracleTypeNum oracleTypeNum, dpiNativeTypeNum nativeTypeNum,
        uint32_t size, int sizeIsBytes, dpiObjectType *objType)
{
    dpiError error;
    dpiData *data;
    dpiVar *var;

    // verify parameters
    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (!stmt->queryInfo && dpiStmt__createQueryVars(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (pos == 0 || pos > stmt->numQueryVars) {
        dpiError__set(&error, "check query position",
                DPI_ERR_QUERY_POSITION_INVALID, pos);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    // create a new variable of the specified type
    if (dpiVar__allocate(stmt->conn, oracleTypeNum, nativeTypeNum,
            stmt->fetchArraySize, size, sizeIsBytes, 0, objType, &var, &data,
            &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (dpiStmt__define(stmt, pos, var, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    dpiGen__setRefCount(var, &error, -1);
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_execute() [PUBLIC]
//   Execute a statement. If the statement has been executed before, however,
// and this is a query, the describe information is already available so defer
// execution until the first fetch.
//-----------------------------------------------------------------------------
int dpiStmt_execute(dpiStmt *stmt, dpiExecMode mode, uint32_t *numQueryColumns)
{
    uint32_t numIters;
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    numIters = (stmt->statementType == DPI_STMT_TYPE_SELECT) ? 0 : 1;
    if (dpiStmt__execute(stmt, numIters, mode, 1, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (numQueryColumns)
        *numQueryColumns = stmt->numQueryVars;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_executeMany() [PUBLIC]
//   Execute a statement multiple times. Queries are not supported. The bind
// variables are checked to ensure that their maxArraySize is sufficient to
// support this.
//-----------------------------------------------------------------------------
int dpiStmt_executeMany(dpiStmt *stmt, dpiExecMode mode, uint32_t numIters)
{
    dpiError error;
    uint32_t i;

    // verify statement is open
    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // queries are not supported
    if (stmt->statementType == DPI_STMT_TYPE_SELECT) {
        dpiError__set(&error, "check statement type", DPI_ERR_NOT_SUPPORTED);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    // batch errors and array DML row counts are only supported with DML
    // statements (insert, update, delete and merge)
    if ((mode & DPI_MODE_EXEC_BATCH_ERRORS ||
                mode & DPI_MODE_EXEC_ARRAY_DML_ROWCOUNTS) &&
            stmt->statementType != DPI_STMT_TYPE_INSERT &&
            stmt->statementType != DPI_STMT_TYPE_UPDATE &&
            stmt->statementType != DPI_STMT_TYPE_DELETE &&
            stmt->statementType != DPI_STMT_TYPE_MERGE) {
        dpiError__set(&error, "check mode", DPI_ERR_EXEC_MODE_ONLY_FOR_DML);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    // ensure that all bind variables have a big enough maxArraySize to
    // support this operation
    for (i = 0; i < stmt->numBindVars; i++) {
        if (stmt->bindVars[i].var->buffer.maxArraySize < numIters) {
            dpiError__set(&error, "check array size",
                    DPI_ERR_ARRAY_SIZE_TOO_SMALL,
                    stmt->bindVars[i].var->buffer.maxArraySize);
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        }
    }

    // perform execution
    dpiStmt__clearBatchErrors(stmt);
    if (dpiStmt__execute(stmt, numIters, mode, 0, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // handle batch errors if mode was specified
    if (mode & DPI_MODE_EXEC_BATCH_ERRORS) {
        if (dpiStmt__getBatchErrors(stmt, &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_fetch() [PUBLIC]
//   Fetch a row from the database.
//-----------------------------------------------------------------------------
int dpiStmt_fetch(dpiStmt *stmt, int *found, uint32_t *bufferRowIndex)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, found)
    DPI_CHECK_PTR_NOT_NULL(stmt, bufferRowIndex)
    if (stmt->bufferRowIndex >= stmt->bufferRowCount) {
        if (stmt->hasRowsToFetch && dpiStmt__fetch(stmt, &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        if (stmt->bufferRowIndex >= stmt->bufferRowCount) {
            *found = 0;
            return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
        }
    }
    *found = 1;
    *bufferRowIndex = stmt->bufferRowIndex;
    stmt->bufferRowIndex++;
    stmt->rowCount++;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_fetchRows() [PUBLIC]
//   Fetch rows into buffers and return the number of rows that were so
// fetched. If there are still rows available in the buffer, no additional
// fetch will take place.
//-----------------------------------------------------------------------------
int dpiStmt_fetchRows(dpiStmt *stmt, uint32_t maxRows,
        uint32_t *bufferRowIndex, uint32_t *numRowsFetched, int *moreRows)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, bufferRowIndex)
    DPI_CHECK_PTR_NOT_NULL(stmt, numRowsFetched)
    DPI_CHECK_PTR_NOT_NULL(stmt, moreRows)
    if (stmt->bufferRowIndex >= stmt->bufferRowCount) {
        if (stmt->hasRowsToFetch && dpiStmt__fetch(stmt, &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        if (stmt->bufferRowIndex >= stmt->bufferRowCount) {
            *moreRows = 0;
            *bufferRowIndex = 0;
            *numRowsFetched = 0;
            return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
        }
    }
    *bufferRowIndex = stmt->bufferRowIndex;
    *numRowsFetched = stmt->bufferRowCount - stmt->bufferRowIndex;
    *moreRows = stmt->hasRowsToFetch;
    if (*numRowsFetched > maxRows) {
        *numRowsFetched = maxRows;
        *moreRows = 1;
    }
    stmt->bufferRowIndex += *numRowsFetched;
    stmt->rowCount += *numRowsFetched;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getBatchErrorCount() [PUBLIC]
//   Return the number of batch errors that took place during the last
// execution of the statement.
//-----------------------------------------------------------------------------
int dpiStmt_getBatchErrorCount(dpiStmt *stmt, uint32_t *count)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, count)
    *count = stmt->numBatchErrors;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getBatchErrors() [PUBLIC]
//   Return the batch errors that took place during the last execution of the
// statement.
//-----------------------------------------------------------------------------
int dpiStmt_getBatchErrors(dpiStmt *stmt, uint32_t numErrors,
        dpiErrorInfo *errors)
{
    dpiError error, tempError;
    uint32_t i;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, errors)
    if (numErrors < stmt->numBatchErrors) {
        dpiError__set(&error, "check num errors", DPI_ERR_ARRAY_SIZE_TOO_SMALL,
                numErrors);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    for (i = 0; i < stmt->numBatchErrors; i++) {
        tempError.buffer = &stmt->batchErrors[i];
        dpiError__getInfo(&tempError, &errors[i]);
    }
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getBindCount() [PUBLIC]
//   Return the number of bind variables referenced in the prepared SQL. In
// SQL statements this counts all bind variables but in PL/SQL statements
// this counts only uniquely named bind variables.
//-----------------------------------------------------------------------------
int dpiStmt_getBindCount(dpiStmt *stmt, uint32_t *count)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, count)
    status = dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, (void*) count,
            0, DPI_OCI_ATTR_BIND_COUNT, "get bind count", &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getBindNames() [PUBLIC]
//   Return the unique names of the bind variables referenced in the prepared
// SQL.
//-----------------------------------------------------------------------------
int dpiStmt_getBindNames(dpiStmt *stmt, uint32_t *numBindNames,
        const char **bindNames, uint32_t *bindNameLengths)
{
    uint8_t bindNameLengthsBuffer[8], indNameLengthsBuffer[8], isDuplicate[8];
    uint32_t startLoc, i, numThisPass, numActualBindNames;
    char *bindNamesBuffer[8], *indNamesBuffer[8];
    void *bindHandles[8];
    int32_t numFound;
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, numBindNames)
    DPI_CHECK_PTR_NOT_NULL(stmt, bindNames)
    DPI_CHECK_PTR_NOT_NULL(stmt, bindNameLengths)
    startLoc = 1;
    numActualBindNames = 0;
    while (1) {
        if (dpiOci__stmtGetBindInfo(stmt, 8, startLoc, &numFound,
                bindNamesBuffer, bindNameLengthsBuffer, indNamesBuffer,
                indNameLengthsBuffer, isDuplicate, bindHandles, &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        if (numFound == 0)
            break;
        numThisPass = abs(numFound) - startLoc + 1;
        if (numThisPass > 8)
            numThisPass = 8;
        for (i = 0; i < numThisPass; i++) {
            startLoc++;
            if (isDuplicate[i])
                continue;
            if (numActualBindNames == *numBindNames) {
                dpiError__set(&error, "check num bind names",
                        DPI_ERR_ARRAY_SIZE_TOO_SMALL, *numBindNames);
                return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
            }
            bindNames[numActualBindNames] = bindNamesBuffer[i];
            bindNameLengths[numActualBindNames] = bindNameLengthsBuffer[i];
            numActualBindNames++;
        }
        if (numFound > 0)
            break;
    }
    *numBindNames = numActualBindNames;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getFetchArraySize() [PUBLIC]
//   Get the array size used for fetches.
//-----------------------------------------------------------------------------
int dpiStmt_getFetchArraySize(dpiStmt *stmt, uint32_t *arraySize)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, arraySize)
    *arraySize = stmt->fetchArraySize;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getImplicitResult() [PUBLIC]
//   Return the next implicit result from the previously executed statement. If
// no more implicit results exist, NULL is returned.
//-----------------------------------------------------------------------------
int dpiStmt_getImplicitResult(dpiStmt *stmt, dpiStmt **implicitResult)
{
    dpiStmt *tempStmt;
    dpiError error;
    void *handle;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, implicitResult)
    if (dpiUtils__checkClientVersion(stmt->env->versionInfo, 12, 1,
            &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (dpiOci__stmtGetNextResult(stmt, &handle, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    *implicitResult = NULL;
    if (handle) {
        if (dpiStmt__allocate(stmt->conn, 0, &tempStmt, &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        tempStmt->handle = handle;
        if (dpiStmt__createQueryVars(tempStmt, &error) < 0) {
            dpiStmt__free(tempStmt, &error);
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        }
        *implicitResult = tempStmt;
    }
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getInfo() [PUBLIC]
//   Return information about the statement in the provided structure.
//-----------------------------------------------------------------------------
int dpiStmt_getInfo(dpiStmt *stmt, dpiStmtInfo *info)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, info)
    info->isQuery = (stmt->statementType == DPI_STMT_TYPE_SELECT);
    info->isPLSQL = (stmt->statementType == DPI_STMT_TYPE_BEGIN ||
            stmt->statementType == DPI_STMT_TYPE_DECLARE ||
            stmt->statementType == DPI_STMT_TYPE_CALL);
    info->isDDL = (stmt->statementType == DPI_STMT_TYPE_CREATE ||
            stmt->statementType == DPI_STMT_TYPE_DROP ||
            stmt->statementType == DPI_STMT_TYPE_ALTER);
    info->isDML = (stmt->statementType == DPI_STMT_TYPE_INSERT ||
            stmt->statementType == DPI_STMT_TYPE_UPDATE ||
            stmt->statementType == DPI_STMT_TYPE_DELETE ||
            stmt->statementType == DPI_STMT_TYPE_MERGE);
    info->statementType = stmt->statementType;
    info->isReturning = stmt->isReturning;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getNumQueryColumns() [PUBLIC]
//   Returns the number of query columns associated with a statement. If the
// statement does not refer to a query, 0 is returned.
//-----------------------------------------------------------------------------
int dpiStmt_getNumQueryColumns(dpiStmt *stmt, uint32_t *numQueryColumns)
{
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, numQueryColumns)
    if (stmt->statementType == DPI_STMT_TYPE_SELECT &&
            stmt->numQueryVars == 0 &&
            dpiStmt__createQueryVars(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    *numQueryColumns = stmt->numQueryVars;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getQueryInfo() [PUBLIC]
//   Get query information for the position in question.
//-----------------------------------------------------------------------------
int dpiStmt_getQueryInfo(dpiStmt *stmt, uint32_t pos, dpiQueryInfo *info)
{
    dpiError error;

    // validate parameters
    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, info)
    if (!stmt->queryInfo && dpiStmt__createQueryVars(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (pos == 0 || pos > stmt->numQueryVars) {
        dpiError__set(&error, "check query position",
                DPI_ERR_QUERY_POSITION_INVALID, pos);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    // copy query information from internal cache
    memcpy(info, &stmt->queryInfo[pos - 1], sizeof(dpiQueryInfo));
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getQueryValue() [PUBLIC]
//   Get value from query at specified position.
//-----------------------------------------------------------------------------
int dpiStmt_getQueryValue(dpiStmt *stmt, uint32_t pos,
        dpiNativeTypeNum *nativeTypeNum, dpiData **data)
{
    dpiError error;
    dpiVar *var;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, nativeTypeNum)
    DPI_CHECK_PTR_NOT_NULL(stmt, data)
    if (!stmt->queryVars) {
        dpiError__set(&error, "check query vars", DPI_ERR_QUERY_NOT_EXECUTED);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    if (pos == 0 || pos > stmt->numQueryVars) {
        dpiError__set(&error, "check query position",
                DPI_ERR_QUERY_POSITION_INVALID, pos);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    var = stmt->queryVars[pos - 1];
    if (!var || stmt->bufferRowIndex == 0 ||
            stmt->bufferRowIndex > stmt->bufferRowCount) {
        dpiError__set(&error, "check fetched row", DPI_ERR_NO_ROW_FETCHED);
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    *nativeTypeNum = var->nativeTypeNum;
    *data = &var->buffer.externalData[stmt->bufferRowIndex - 1];
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getRowCount() [PUBLIC]
//   Return the number of rows affected by the last DML executed (for insert,
// update, delete and merge) or the number of rows fetched (for queries). In
// all other cases, 0 is returned.
//-----------------------------------------------------------------------------
int dpiStmt_getRowCount(dpiStmt *stmt, uint64_t *count)
{
    uint32_t rowCount32;
    dpiError error;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, count)
    if (stmt->statementType == DPI_STMT_TYPE_SELECT)
        *count = stmt->rowCount;
    else if (stmt->statementType != DPI_STMT_TYPE_INSERT &&
            stmt->statementType != DPI_STMT_TYPE_UPDATE &&
            stmt->statementType != DPI_STMT_TYPE_DELETE &&
            stmt->statementType != DPI_STMT_TYPE_MERGE &&
            stmt->statementType != DPI_STMT_TYPE_CALL &&
            stmt->statementType != DPI_STMT_TYPE_BEGIN &&
            stmt->statementType != DPI_STMT_TYPE_DECLARE) {
        *count = 0;
    } else if (stmt->env->versionInfo->versionNum < 12) {
        if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, &rowCount32, 0,
                DPI_OCI_ATTR_ROW_COUNT, "get row count", &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        *count = rowCount32;
    } else {
        if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, count, 0,
                DPI_OCI_ATTR_UB8_ROW_COUNT, "get row count", &error) < 0)
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getRowCounts() [PUBLIC]
//   Return the number of rows affected by each of the iterations executed
// using dpiStmt_executeMany().
//-----------------------------------------------------------------------------
int dpiStmt_getRowCounts(dpiStmt *stmt, uint32_t *numRowCounts,
        uint64_t **rowCounts)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, numRowCounts)
    DPI_CHECK_PTR_NOT_NULL(stmt, rowCounts)
    if (dpiUtils__checkClientVersion(stmt->env->versionInfo, 12, 1,
            &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    status = dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, rowCounts,
            numRowCounts, DPI_OCI_ATTR_DML_ROW_COUNT_ARRAY, "get row counts",
            &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_getSubscrQueryId() [PUBLIC]
//   Return the query id for a query registered using this statement.
//-----------------------------------------------------------------------------
int dpiStmt_getSubscrQueryId(dpiStmt *stmt, uint64_t *queryId)
{
    dpiError error;
    int status;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(stmt, queryId)
    status = dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, queryId, 0,
            DPI_OCI_ATTR_CQ_QUERYID, "get query id", &error);
    return dpiGen__endPublicFn(stmt, status, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_release() [PUBLIC]
//   Release a reference to the statement.
//-----------------------------------------------------------------------------
int dpiStmt_release(dpiStmt *stmt)
{
    return dpiGen__release(stmt, DPI_HTYPE_STMT, __func__);
}


//-----------------------------------------------------------------------------
// dpiStmt_scroll() [PUBLIC]
//   Scroll to the specified location in the cursor.
//-----------------------------------------------------------------------------
int dpiStmt_scroll(dpiStmt *stmt, dpiFetchMode mode, int32_t offset,
        int32_t rowCountOffset)
{
    uint32_t numRows, currentPosition;
    uint64_t desiredRow = 0;
    dpiError error;

    // make sure the cursor is open
    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // validate mode; determine desired row to fetch
    switch (mode) {
        case DPI_MODE_FETCH_NEXT:
            desiredRow = stmt->rowCount + rowCountOffset + 1;
            break;
        case DPI_MODE_FETCH_PRIOR:
            desiredRow = stmt->rowCount + rowCountOffset - 1;
            break;
        case DPI_MODE_FETCH_FIRST:
            desiredRow = 1;
            break;
        case DPI_MODE_FETCH_LAST:
            break;
        case DPI_MODE_FETCH_ABSOLUTE:
            desiredRow = (uint64_t) offset;
            break;
        case DPI_MODE_FETCH_RELATIVE:
            desiredRow = stmt->rowCount + rowCountOffset + offset;
            offset = (int32_t) (desiredRow -
                    (stmt->bufferMinRow + stmt->bufferRowCount - 1));
            break;
        default:
            dpiError__set(&error, "scroll mode", DPI_ERR_NOT_SUPPORTED);
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    }

    // determine if a fetch is actually required; "last" is always fetched
    if (mode != DPI_MODE_FETCH_LAST && desiredRow >= stmt->bufferMinRow &&
            desiredRow < stmt->bufferMinRow + stmt->bufferRowCount) {
        stmt->bufferRowIndex = (uint32_t) (desiredRow - stmt->bufferMinRow);
        stmt->rowCount = desiredRow - 1;
        return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
    }

    // perform any pre-fetch activities required
    if (dpiStmt__beforeFetch(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // perform fetch; when fetching the last row, only fetch a single row
    numRows = (mode == DPI_MODE_FETCH_LAST) ? 1 : stmt->fetchArraySize;
    if (dpiOci__stmtFetch2(stmt, numRows, mode, offset, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // determine the number of rows actually fetched
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT,
            &stmt->bufferRowCount, 0, DPI_OCI_ATTR_ROWS_FETCHED,
            "get rows fetched", &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // check that we haven't gone outside of the result set
    if (stmt->bufferRowCount == 0) {
        if (mode != DPI_MODE_FETCH_FIRST && mode != DPI_MODE_FETCH_LAST) {
            dpiError__set(&error, "check result set bounds",
                    DPI_ERR_SCROLL_OUT_OF_RS);
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        }
        stmt->hasRowsToFetch = 0;
        stmt->rowCount = 0;
        stmt->bufferRowIndex = 0;
        stmt->bufferMinRow = 0;
        return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
    }

    // determine the current position of the cursor
    if (dpiOci__attrGet(stmt->handle, DPI_OCI_HTYPE_STMT, &currentPosition, 0,
            DPI_OCI_ATTR_CURRENT_POSITION, "get current pos", &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    // reset buffer row index and row count
    stmt->rowCount = currentPosition - stmt->bufferRowCount;
    stmt->bufferMinRow = stmt->rowCount + 1;
    stmt->bufferRowIndex = 0;

    // perform post-fetch activities required
    if (dpiStmt__postFetch(stmt, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);

    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiStmt_setFetchArraySize() [PUBLIC]
//   Set the array size used for fetches. Using a value of zero will select the
// default value. A check is made to ensure that all defined variables have
// sufficient space to support the array size.
//-----------------------------------------------------------------------------
int dpiStmt_setFetchArraySize(dpiStmt *stmt, uint32_t arraySize)
{
    dpiError error;
    dpiVar *var;
    uint32_t i;

    if (dpiStmt__check(stmt, __func__, &error) < 0)
        return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
    if (arraySize == 0)
        arraySize = DPI_DEFAULT_FETCH_ARRAY_SIZE;
    for (i = 0; i < stmt->numQueryVars; i++) {
        var = stmt->queryVars[i];
        if (var && var->buffer.maxArraySize < arraySize) {
            dpiError__set(&error, "check array size",
                    DPI_ERR_ARRAY_SIZE_TOO_BIG, arraySize);
            return dpiGen__endPublicFn(stmt, DPI_FAILURE, &error);
        }
    }
    stmt->fetchArraySize = arraySize;
    return dpiGen__endPublicFn(stmt, DPI_SUCCESS, &error);
}

