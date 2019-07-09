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
// dpiEnv.c
//   Implementation of environment.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiEnv__free() [INTERNAL]
//   Free the memory associated with the environment.
//-----------------------------------------------------------------------------
void dpiEnv__free(dpiEnv *env, dpiError *error)
{
    if (env->threaded)
        dpiMutex__destroy(env->mutex);
    if (env->handle) {
        dpiOci__handleFree(env->handle, DPI_OCI_HTYPE_ENV);
        env->handle = NULL;
    }
    if (env->errorHandles) {
        dpiHandlePool__free(env->errorHandles);
        env->errorHandles = NULL;
        error->handle = NULL;
    }
    dpiUtils__freeMemory(env);
}


//-----------------------------------------------------------------------------
// dpiEnv__getCharacterSetIdAndName() [INTERNAL]
//   Retrieve and store the IANA character set name for the attribute.
//-----------------------------------------------------------------------------
static int dpiEnv__getCharacterSetIdAndName(dpiEnv *env, uint16_t attribute,
        uint16_t *charsetId, char *encoding, dpiError *error)
{
    *charsetId = 0;
    dpiOci__attrGet(env->handle, DPI_OCI_HTYPE_ENV, charsetId, NULL, attribute,
            "get environment", error);
    return dpiGlobal__lookupEncoding(*charsetId, encoding, error);
}


//-----------------------------------------------------------------------------
// dpiEnv__getEncodingInfo() [INTERNAL]
//   Populate the structure with the encoding info.
//-----------------------------------------------------------------------------
int dpiEnv__getEncodingInfo(dpiEnv *env, dpiEncodingInfo *info)
{
    info->encoding = env->encoding;
    info->maxBytesPerCharacter = env->maxBytesPerCharacter;
    info->nencoding = env->nencoding;
    info->nmaxBytesPerCharacter = env->nmaxBytesPerCharacter;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiEnv__init() [INTERNAL]
//   Initialize the environment structure by creating the OCI environment and
// populating information about the environment.
//-----------------------------------------------------------------------------
int dpiEnv__init(dpiEnv *env, const dpiContext *context,
        const dpiCommonCreateParams *params, dpiError *error)
{
    char timezoneBuffer[20];
    size_t timezoneLength;

    // lookup encoding
    if (params->encoding && dpiGlobal__lookupCharSet(params->encoding,
            &env->charsetId, error) < 0)
        return DPI_FAILURE;

    // check for identical encoding before performing lookup
    if (params->nencoding && params->encoding &&
            strcmp(params->nencoding, params->encoding) == 0)
        env->ncharsetId = env->charsetId;
    else if (params->nencoding && dpiGlobal__lookupCharSet(params->nencoding,
            &env->ncharsetId, error) < 0)
        return DPI_FAILURE;

    // both charsetId and ncharsetId must be zero or both must be non-zero
    // use NLS routine to look up missing value, if needed
    if (env->charsetId && !env->ncharsetId) {
        if (dpiOci__nlsEnvironmentVariableGet(DPI_OCI_NLS_NCHARSET_ID,
                &env->ncharsetId, error) < 0)
            return DPI_FAILURE;
    } else if (!env->charsetId && env->ncharsetId) {
        if (dpiOci__nlsEnvironmentVariableGet(DPI_OCI_NLS_CHARSET_ID,
                &env->charsetId, error) < 0)
            return DPI_FAILURE;
    }

    // create the new environment handle
    env->context = context;
    env->versionInfo = context->versionInfo;
    if (dpiOci__envNlsCreate(&env->handle, params->createMode | DPI_OCI_OBJECT,
            env->charsetId, env->ncharsetId, error) < 0)
        return DPI_FAILURE;

    // create the error handle pool and acquire the first error handle
    if (dpiHandlePool__create(&env->errorHandles, error) < 0)
        return DPI_FAILURE;
    if (dpiEnv__initError(env, error) < 0)
        return DPI_FAILURE;

    // if threaded, create mutex for reference counts
    if (params->createMode & DPI_OCI_THREADED)
        dpiMutex__initialize(env->mutex);

    // determine encodings in use
    if (dpiEnv__getCharacterSetIdAndName(env, DPI_OCI_ATTR_CHARSET_ID,
            &env->charsetId, env->encoding, error) < 0)
        return DPI_FAILURE;
    if (dpiEnv__getCharacterSetIdAndName(env, DPI_OCI_ATTR_NCHARSET_ID,
            &env->ncharsetId, env->nencoding, error) < 0)
        return DPI_FAILURE;

    // acquire max bytes per character
    if (dpiOci__nlsNumericInfoGet(env->handle, &env->maxBytesPerCharacter,
            DPI_OCI_NLS_CHARSET_MAXBYTESZ, error) < 0)
        return DPI_FAILURE;

    // for NCHAR we have no idea of how many so we simply take the worst case
    // unless the charsets are identical
    if (env->ncharsetId == env->charsetId)
        env->nmaxBytesPerCharacter = env->maxBytesPerCharacter;
    else env->nmaxBytesPerCharacter = 4;

    // allocate base date descriptor (for converting to/from time_t)
    if (dpiOci__descriptorAlloc(env->handle, &env->baseDate,
            DPI_OCI_DTYPE_TIMESTAMP_LTZ, "alloc base date descriptor",
            error) < 0)
        return DPI_FAILURE;

    // populate base date with January 1, 1970
    if (dpiOci__nlsCharSetConvert(env->handle, env->charsetId, timezoneBuffer,
            sizeof(timezoneBuffer), DPI_CHARSET_ID_ASCII, "+00:00", 6,
            &timezoneLength, error) < 0)
        return DPI_FAILURE;
    if (dpiOci__dateTimeConstruct(env->handle, env->baseDate, 1970, 1, 1, 0, 0,
            0, 0, timezoneBuffer, timezoneLength, error) < 0)
        return DPI_FAILURE;

    // set whether or not we are threaded
    if (params->createMode & DPI_MODE_CREATE_THREADED)
        env->threaded = 1;

    // set whether or not events mode has been set
    if (params->createMode & DPI_MODE_CREATE_EVENTS)
        env->events = 1;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiEnv__initError() [INTERNAL]
//   Retrieve the OCI error handle to use for error handling, from a pool of
// error handles common to the environment handle. The environment that was
// used to create the error handle is stored in the error structure so that
// the encoding and character set can be retrieved in the event of an OCI
// error (which uses the CHAR encoding of the environment).
//-----------------------------------------------------------------------------
int dpiEnv__initError(dpiEnv *env, dpiError *error)
{
    error->env = env;
    if (dpiHandlePool__acquire(env->errorHandles, &error->handle, error) < 0)
        return DPI_FAILURE;

    if (!error->handle) {
        if (dpiOci__handleAlloc(env->handle, &error->handle,
                DPI_OCI_HTYPE_ERROR, "allocate OCI error", error) < 0)
            return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}

