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
// dpiGlobal.c
//   Global environment used for managing errors in a thread safe manner as
// well as for looking up encodings.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// cross platform way of defining an initializer that runs at application
// startup (similar to what is done for the constructor calls for static C++
// objects)
#if defined(_MSC_VER)
    #pragma section(".CRT$XCU", read)
    #define DPI_INITIALIZER_HELPER(f, p) \
        static void f(void); \
        __declspec(allocate(".CRT$XCU")) void (*f##_)(void) = f; \
        __pragma(comment(linker,"/include:" p #f "_")) \
        static void f(void)
    #ifdef _WIN64
        #define DPI_INITIALIZER(f) DPI_INITIALIZER_HELPER(f, "")
    #else
        #define DPI_INITIALIZER(f) DPI_INITIALIZER_HELPER(f, "_")
    #endif
#else
    #define DPI_INITIALIZER(f) \
        static void f(void) __attribute__((constructor)); \
        static void f(void)
#endif

// a global OCI environment is used for managing error buffers in a thread-safe
// manner; each thread is given its own error buffer; OCI error handles,
// though, must be created within the OCI environment created for use by
// standalone connections and session pools
static void *dpiGlobalEnvHandle = NULL;
static void *dpiGlobalErrorHandle = NULL;
static void *dpiGlobalThreadKey = NULL;
static dpiErrorBuffer dpiGlobalErrorBuffer;
static int dpiGlobalInitialized = 0;

// a global mutex is used to ensure that only one thread is used to perform
// initialization of ODPI-C
static dpiMutexType dpiGlobalMutex;

//-----------------------------------------------------------------------------
// dpiGlobal__extendedInitialize() [INTERNAL]
//   Create the global environment used for managing error buffers in a
// thread-safe manner. This environment is solely used for implementing thread
// local storage for the error buffers and for looking up encodings given an
// IANA or Oracle character set name.
//-----------------------------------------------------------------------------
static int dpiGlobal__extendedInitialize(dpiError *error)
{
    int status;

    // create threaded OCI environment for storing error buffers and for
    // looking up character sets; use character set AL32UTF8 solely to avoid
    // the overhead of processing the environment variables; no error messages
    // from this environment are ever used (ODPI-C specific error messages are
    // used)
    if (dpiOci__envNlsCreate(&dpiGlobalEnvHandle, DPI_OCI_THREADED,
            DPI_CHARSET_ID_UTF8, DPI_CHARSET_ID_UTF8, error) < 0)
        return DPI_FAILURE;

    // create global error handle
    if (dpiOci__handleAlloc(dpiGlobalEnvHandle, &dpiGlobalErrorHandle,
            DPI_OCI_HTYPE_ERROR, "create global error", error) < 0) {
        dpiOci__handleFree(dpiGlobalEnvHandle, DPI_OCI_HTYPE_ENV);
        return DPI_FAILURE;
    }

    // create global thread key
    status = dpiOci__threadKeyInit(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
            &dpiGlobalThreadKey, (void*) dpiUtils__freeMemory, error);
    if (status < 0) {
        dpiOci__handleFree(dpiGlobalEnvHandle, DPI_OCI_HTYPE_ENV);
        return DPI_FAILURE;
    }

    // mark library as fully initialized
    dpiGlobalInitialized = 1;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiGlobal__finalize() [INTERNAL]
//   Called when the process terminates and ensures that everything is cleaned
// up.
//-----------------------------------------------------------------------------
static void dpiGlobal__finalize(void)
{
    void *errorBuffer = NULL;
    dpiError error;

    dpiMutex__acquire(dpiGlobalMutex);
    dpiGlobalInitialized = 0;
    error.buffer = &dpiGlobalErrorBuffer;
    if (dpiGlobalThreadKey) {
        dpiOci__threadKeyGet(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
                dpiGlobalThreadKey, &errorBuffer, &error);
        if (errorBuffer) {
            dpiOci__threadKeySet(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
                    dpiGlobalThreadKey, NULL, &error);
            dpiUtils__freeMemory(errorBuffer);
        }
        dpiOci__threadKeyDestroy(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
                &dpiGlobalThreadKey, &error);
        dpiGlobalThreadKey = NULL;
    }
    if (dpiGlobalEnvHandle) {
        dpiOci__handleFree(dpiGlobalEnvHandle, DPI_OCI_HTYPE_ENV);
        dpiGlobalEnvHandle = NULL;
    }
    dpiMutex__release(dpiGlobalMutex);
}


//-----------------------------------------------------------------------------
// dpiGlobal__initError() [INTERNAL]
//   Get the thread local error structure for use in all other functions. If
// an error structure cannot be determined for some reason, the global error
// buffer structure is returned instead.
//-----------------------------------------------------------------------------
int dpiGlobal__initError(const char *fnName, dpiError *error)
{
    dpiErrorBuffer *tempErrorBuffer;

    // initialize error buffer output to global error buffer structure; this is
    // the value that is used if an error takes place before the thread local
    // error structure can be returned
    error->handle = NULL;
    error->buffer = &dpiGlobalErrorBuffer;
    if (fnName)
        error->buffer->fnName = fnName;

    // initialize global environment, if necessary
    // this should only ever be done once by the first thread to execute this
    if (!dpiGlobalInitialized) {
        dpiMutex__acquire(dpiGlobalMutex);
        if (!dpiGlobalInitialized)
            dpiGlobal__extendedInitialize(error);
        dpiMutex__release(dpiGlobalMutex);
        if (!dpiGlobalInitialized)
            return DPI_FAILURE;
    }

    // look up the error buffer specific to this thread
    if (dpiOci__threadKeyGet(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
            dpiGlobalThreadKey, (void**) &tempErrorBuffer, error) < 0)
        return DPI_FAILURE;

    // if NULL, key has never been set for this thread, allocate new error
    // and set it
    if (!tempErrorBuffer) {
        if (dpiUtils__allocateMemory(1, sizeof(dpiErrorBuffer), 1,
                "allocate error buffer", (void**) &tempErrorBuffer, error) < 0)
            return DPI_FAILURE;
        if (dpiOci__threadKeySet(dpiGlobalEnvHandle, dpiGlobalErrorHandle,
                dpiGlobalThreadKey, tempErrorBuffer, error) < 0) {
            dpiUtils__freeMemory(tempErrorBuffer);
            return DPI_FAILURE;
        }
    }

    // if a function name has been specified, clear error
    // the only time a function name is not specified is for
    // dpiContext_getError() when the error information is being retrieved
    if (fnName) {
        tempErrorBuffer->code = 0;
        tempErrorBuffer->offset = 0;
        tempErrorBuffer->errorNum = (dpiErrorNum) 0;
        tempErrorBuffer->isRecoverable = 0;
        tempErrorBuffer->messageLength = 0;
        tempErrorBuffer->fnName = fnName;
        tempErrorBuffer->action = "start";
        strcpy(tempErrorBuffer->encoding, DPI_CHARSET_NAME_UTF8);
    }

    error->buffer = tempErrorBuffer;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiGlobal__initialize() [INTERNAL]
//   Initialization function that runs at process startup or when the library
// is first loaded. Some operating systems have limits on what can be run in
// this function, so most work is done in the dpiGlobal__extendedInitialize()
// function that runs when the first call to dpiContext_create() is made.
//-----------------------------------------------------------------------------
DPI_INITIALIZER(dpiGlobal__initialize)
{
    memset(&dpiGlobalErrorBuffer, 0, sizeof(dpiGlobalErrorBuffer));
    strcpy(dpiGlobalErrorBuffer.encoding, DPI_CHARSET_NAME_UTF8);
    dpiMutex__initialize(dpiGlobalMutex);
    dpiDebug__initialize();
    atexit(dpiGlobal__finalize);
}


//-----------------------------------------------------------------------------
// dpiGlobal__lookupCharSet() [INTERNAL]
//   Lookup the character set id that can be used in the call to
// OCINlsEnvCreate().
//-----------------------------------------------------------------------------
int dpiGlobal__lookupCharSet(const char *name, uint16_t *charsetId,
        dpiError *error)
{
    char oraCharsetName[DPI_OCI_NLS_MAXBUFSZ];

    // check for well-known encodings first
    if (strcmp(name, DPI_CHARSET_NAME_UTF8) == 0)
        *charsetId = DPI_CHARSET_ID_UTF8;
    else if (strcmp(name, DPI_CHARSET_NAME_UTF16) == 0)
        *charsetId = DPI_CHARSET_ID_UTF16;
    else if (strcmp(name, DPI_CHARSET_NAME_ASCII) == 0)
        *charsetId = DPI_CHARSET_ID_ASCII;
    else if (strcmp(name, DPI_CHARSET_NAME_UTF16LE) == 0 ||
            strcmp(name, DPI_CHARSET_NAME_UTF16BE) == 0)
        return dpiError__set(error, "check encoding", DPI_ERR_NOT_SUPPORTED);

    // perform lookup; check for the Oracle character set name first and if
    // that fails, lookup using the IANA character set name
    else {
        if (dpiOci__nlsCharSetNameToId(dpiGlobalEnvHandle, name, charsetId,
                error) < 0)
            return DPI_FAILURE;
        if (!*charsetId) {
            if (dpiOci__nlsNameMap(dpiGlobalEnvHandle, oraCharsetName,
                    sizeof(oraCharsetName), name, DPI_OCI_NLS_CS_IANA_TO_ORA,
                    error) < 0)
                return dpiError__set(error, "lookup charset",
                        DPI_ERR_INVALID_CHARSET, name);
            dpiOci__nlsCharSetNameToId(dpiGlobalEnvHandle, oraCharsetName,
                    charsetId, error);
        }
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiGlobal__lookupEncoding() [INTERNAL]
//   Get the IANA character set name (encoding) given the Oracle character set
// id.
//-----------------------------------------------------------------------------
int dpiGlobal__lookupEncoding(uint16_t charsetId, char *encoding,
        dpiError *error)
{
    char oracleName[DPI_OCI_NLS_MAXBUFSZ];

    // check for well-known encodings first
    switch (charsetId) {
        case DPI_CHARSET_ID_UTF8:
            strcpy(encoding, DPI_CHARSET_NAME_UTF8);
            return DPI_SUCCESS;
        case DPI_CHARSET_ID_UTF16:
            strcpy(encoding, DPI_CHARSET_NAME_UTF16);
            return DPI_SUCCESS;
        case DPI_CHARSET_ID_ASCII:
            strcpy(encoding, DPI_CHARSET_NAME_ASCII);
            return DPI_SUCCESS;
    }

    // get character set name
    if (dpiOci__nlsCharSetIdToName(dpiGlobalEnvHandle, oracleName,
            sizeof(oracleName), charsetId, error) < 0)
        return dpiError__set(error, "lookup Oracle character set name",
                DPI_ERR_INVALID_CHARSET_ID, charsetId);

    // get IANA character set name
    if (dpiOci__nlsNameMap(dpiGlobalEnvHandle, encoding, DPI_OCI_NLS_MAXBUFSZ,
            oracleName, DPI_OCI_NLS_CS_ORA_TO_IANA, error) < 0)
        return dpiError__set(error, "lookup IANA name",
                DPI_ERR_INVALID_CHARSET_ID, charsetId);

    return DPI_SUCCESS;
}

