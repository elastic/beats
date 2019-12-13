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
// dpiContext.c
//   Implementation of context. Each context uses a specific version of the
// ODPI-C library, which is checked for compatibility before allowing its use.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// maintain major and minor versions compiled into the library
static const unsigned int dpiMajorVersion = DPI_MAJOR_VERSION;
static const unsigned int dpiMinorVersion = DPI_MINOR_VERSION;


//-----------------------------------------------------------------------------
// dpiContext__create() [INTERNAL]
//   Create a new context for interaction with the library. The major versions
// must match and the minor version of the caller must be less than or equal to
// the minor version compiled into the library.
//-----------------------------------------------------------------------------
static int dpiContext__create(const char *fnName, unsigned int majorVersion,
        unsigned int minorVersion, dpiContext **context, dpiError *error)
{
    dpiContext *tempContext;

    // get error structure first (populates global environment if needed)
    if (dpiGlobal__initError(fnName, error) < 0)
        return DPI_FAILURE;

    // validate context handle
    if (!context)
        return dpiError__set(error, "check context handle",
                DPI_ERR_NULL_POINTER_PARAMETER, "context");

    // verify that the supplied version is supported by the library
    if (dpiMajorVersion != majorVersion || minorVersion > dpiMinorVersion)
        return dpiError__set(error, "check version",
                DPI_ERR_VERSION_NOT_SUPPORTED, majorVersion, majorVersion,
                minorVersion, dpiMajorVersion, dpiMinorVersion);

    // allocate context and initialize it
    if (dpiGen__allocate(DPI_HTYPE_CONTEXT, NULL, (void**) &tempContext,
            error) < 0)
        return DPI_FAILURE;
    tempContext->dpiMinorVersion = (uint8_t) minorVersion;
    dpiOci__clientVersion(tempContext);

    *context = tempContext;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiContext__initCommonCreateParams() [INTERNAL]
//   Initialize the common connection/pool creation parameters to default
// values.
//-----------------------------------------------------------------------------
void dpiContext__initCommonCreateParams(dpiCommonCreateParams *params)
{
    memset(params, 0, sizeof(dpiCommonCreateParams));
}


//-----------------------------------------------------------------------------
// dpiContext__initConnCreateParams() [INTERNAL]
//   Initialize the connection creation parameters to default values. Return
// the structure size as a convenience for calling functions which may have to
// differentiate between different ODPI-C application versions.
//-----------------------------------------------------------------------------
void dpiContext__initConnCreateParams(dpiConnCreateParams *params)
{
    memset(params, 0, sizeof(dpiConnCreateParams));
}


//-----------------------------------------------------------------------------
// dpiContext__initPoolCreateParams() [INTERNAL]
//   Initialize the pool creation parameters to default values.
//-----------------------------------------------------------------------------
void dpiContext__initPoolCreateParams(dpiPoolCreateParams *params)
{
    memset(params, 0, sizeof(dpiPoolCreateParams));
    params->minSessions = 1;
    params->maxSessions = 1;
    params->sessionIncrement = 0;
    params->homogeneous = 1;
    params->getMode = DPI_MODE_POOL_GET_NOWAIT;
    params->pingInterval = DPI_DEFAULT_PING_INTERVAL;
    params->pingTimeout = DPI_DEFAULT_PING_TIMEOUT;
}


//-----------------------------------------------------------------------------
// dpiContext__initSodaOperOptions() [INTERNAL]
//   Initialize the SODA operation options to default values.
//-----------------------------------------------------------------------------
void dpiContext__initSodaOperOptions(dpiSodaOperOptions *options)
{
    memset(options, 0, sizeof(dpiSodaOperOptions));
}


//-----------------------------------------------------------------------------
// dpiContext__initSubscrCreateParams() [INTERNAL]
//   Initialize the subscription creation parameters to default values.
//-----------------------------------------------------------------------------
void dpiContext__initSubscrCreateParams(dpiSubscrCreateParams *params)
{
    memset(params, 0, sizeof(dpiSubscrCreateParams));
    params->subscrNamespace = DPI_SUBSCR_NAMESPACE_DBCHANGE;
    params->groupingType = DPI_SUBSCR_GROUPING_TYPE_SUMMARY;
}


//-----------------------------------------------------------------------------
// dpiContext_create() [PUBLIC]
//   Create a new context for interaction with the library. The major versions
// must match and the minor version of the caller must be less than or equal to
// the minor version compiled into the library.
//-----------------------------------------------------------------------------
int dpiContext_create(unsigned int majorVersion, unsigned int minorVersion,
        dpiContext **context, dpiErrorInfo *errorInfo)
{
    dpiError error;
    int status;

    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        dpiDebug__print("fn start %s\n", __func__);
    status = dpiContext__create(__func__, majorVersion, minorVersion, context,
            &error);
    if (status < 0)
        dpiError__getInfo(&error, errorInfo);
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        dpiDebug__print("fn end %s -> %d\n", __func__, status);
    return status;
}


//-----------------------------------------------------------------------------
// dpiContext_destroy() [PUBLIC]
//   Destroy an existing context. The structure will be checked for validity
// first.
//-----------------------------------------------------------------------------
int dpiContext_destroy(dpiContext *context)
{
    char message[80];
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    dpiUtils__clearMemory(&context->checkInt, sizeof(context->checkInt));
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_REFS)
        dpiDebug__print("ref %p (%s) -> 0\n", context, context->typeDef->name);
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        (void) sprintf(message, "fn end %s(%p) -> %d", __func__, context,
                DPI_SUCCESS);
    dpiUtils__freeMemory(context);
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_FNS)
        dpiDebug__print("%s\n", message);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiContext_getClientVersion() [PUBLIC]
//   Return the version of the Oracle client that is in use.
//-----------------------------------------------------------------------------
int dpiContext_getClientVersion(const dpiContext *context,
        dpiVersionInfo *versionInfo)
{
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, versionInfo)
    memcpy(versionInfo, context->versionInfo, sizeof(dpiVersionInfo));
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiContext_getError() [PUBLIC]
//   Return information about the error that was last populated.
//-----------------------------------------------------------------------------
void dpiContext_getError(const dpiContext *context, dpiErrorInfo *info)
{
    dpiError error;

    dpiGlobal__initError(NULL, &error);
    dpiGen__checkHandle(context, DPI_HTYPE_CONTEXT, "check handle", &error);
    dpiError__getInfo(&error, info);
}


//-----------------------------------------------------------------------------
// dpiContext_initCommonCreateParams() [PUBLIC]
//   Initialize the common connection/pool creation parameters to default
// values.
//-----------------------------------------------------------------------------
int dpiContext_initCommonCreateParams(const dpiContext *context,
        dpiCommonCreateParams *params)
{
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, params)
    dpiContext__initCommonCreateParams(params);
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiContext_initConnCreateParams() [PUBLIC]
//   Initialize the connection creation parameters to default values.
//-----------------------------------------------------------------------------
int dpiContext_initConnCreateParams(const dpiContext *context,
        dpiConnCreateParams *params)
{
    dpiConnCreateParams localParams;
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, params)

    // size changed in version 3.1; can be dropped once version 4 released
    if (context->dpiMinorVersion > 0)
        dpiContext__initConnCreateParams(params);
    else {
        dpiContext__initConnCreateParams(&localParams);
        memcpy(params, &localParams, sizeof(dpiConnCreateParams__v30));
    }
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiContext_initPoolCreateParams() [PUBLIC]
//   Initialize the pool creation parameters to default values.
//-----------------------------------------------------------------------------
int dpiContext_initPoolCreateParams(const dpiContext *context,
        dpiPoolCreateParams *params)
{
    dpiPoolCreateParams localParams;
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, params)

    // size changed in version 3.1; can be dropped once version 4 released
    if (context->dpiMinorVersion > 0)
        dpiContext__initPoolCreateParams(params);
    else {
        dpiContext__initPoolCreateParams(&localParams);
        memcpy(params, &localParams, sizeof(dpiPoolCreateParams__v30));
    }
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiContext_initSodaOperOptions() [PUBLIC]
//   Initialize the SODA operation options to default values.
//-----------------------------------------------------------------------------
int dpiContext_initSodaOperOptions(const dpiContext *context,
        dpiSodaOperOptions *options)
{
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, options)
    dpiContext__initSodaOperOptions(options);
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiContext_initSubscrCreateParams() [PUBLIC]
//   Initialize the subscription creation parameters to default values.
//-----------------------------------------------------------------------------
int dpiContext_initSubscrCreateParams(const dpiContext *context,
        dpiSubscrCreateParams *params)
{
    dpiError error;

    if (dpiGen__startPublicFn(context, DPI_HTYPE_CONTEXT, __func__, 0,
            &error) < 0)
        return dpiGen__endPublicFn(context, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(context, params)
    dpiContext__initSubscrCreateParams(params);
    return dpiGen__endPublicFn(context, DPI_SUCCESS, &error);
}

