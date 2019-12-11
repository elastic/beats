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
// dpiError.c
//   Implementation of error.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"
#include "dpiErrorMessages.h"

//-----------------------------------------------------------------------------
// dpiError__check() [INTERNAL]
//   Checks to see if the status of the last call resulted in an error
// condition. If so, the error is populated. Note that trailing newlines and
// spaces are truncated from the message if they exist. If the connection is
// not NULL a check is made to see if the connection is no longer viable.
//-----------------------------------------------------------------------------
int dpiError__check(dpiError *error, int status, dpiConn *conn,
        const char *action)
{
    uint32_t callTimeout;

    // no error has taken place
    if (status == DPI_OCI_SUCCESS || status == DPI_OCI_SUCCESS_WITH_INFO)
        return DPI_SUCCESS;

    // special error cases
    if (status == DPI_OCI_INVALID_HANDLE)
        return dpiError__set(error, action, DPI_ERR_INVALID_HANDLE, "OCI");
    else if (!error)
        return DPI_FAILURE;
    else if (!error->handle)
        return dpiError__set(error, action, DPI_ERR_ERR_NOT_INITIALIZED);
    else if (status != DPI_OCI_ERROR && status != DPI_OCI_NO_DATA)
        return dpiError__set(error, action,
                DPI_ERR_UNEXPECTED_OCI_RETURN_VALUE, status,
                error->buffer->fnName);

    // fetch OCI error
    error->buffer->action = action;
    strcpy(error->buffer->encoding, error->env->encoding);
    if (dpiOci__errorGet(error->handle, DPI_OCI_HTYPE_ERROR,
            error->env->charsetId, action, error) < 0)
        return DPI_FAILURE;
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_ERRORS)
        dpiDebug__print("OCI error %.*s (%s / %s)\n",
                error->buffer->messageLength, error->buffer->message,
                error->buffer->fnName, action);

    // determine if error is recoverable (Transaction Guard)
    // if the attribute cannot be read properly, simply leave it as false;
    // otherwise, that error will mask the one that we really want to see
    error->buffer->isRecoverable = 0;
    dpiOci__attrGet(error->handle, DPI_OCI_HTYPE_ERROR,
            (void*) &error->buffer->isRecoverable, 0,
            DPI_OCI_ATTR_ERROR_IS_RECOVERABLE, NULL, error);

    // check for certain errors which indicate that the session is dead and
    // should be dropped from the session pool (if a session pool was used)
    // also check for call timeout and raise unified message instead
    if (conn && !conn->deadSession) {
        switch (error->buffer->code) {
            case    22: // invalid session ID; access denied
            case    28: // your session has been killed
            case    31: // your session has been marked for kill
            case    45: // your session has been terminated with no replay
            case   378: // buffer pools cannot be created as specified
            case   602: // internal programming exception
            case   603: // ORACLE server session terminated by fatal error
            case   609: // could not attach to incoming connection
            case  1012: // not logged on
            case  1041: // internal error. hostdef extension doesn't exist
            case  1043: // user side memory corruption
            case  1089: // immediate shutdown or close in progress
            case  1092: // ORACLE instance terminated. Disconnection forced
            case  2396: // exceeded maximum idle time, please connect again
            case  3113: // end-of-file on communication channel
            case  3114: // not connected to ORACLE
            case  3122: // attempt to close ORACLE-side window on user side
            case  3135: // connection lost contact
            case 12153: // TNS:not connected
            case 12537: // TNS:connection closed
            case 12547: // TNS:lost contact
            case 12570: // TNS:packet reader failure
            case 12583: // TNS:no reader
            case 27146: // post/wait initialization failed
            case 28511: // lost RPC connection
            case 56600: // an illegal OCI function call was issued
                conn->deadSession = 1;
                break;
            case  3136: // inbound connection timed out
            case 12161: // TNS:internal error: partial data received
                callTimeout = 0;
                if (conn->env->versionInfo->versionNum >= 18)
                    dpiOci__attrGet(conn->handle, DPI_OCI_HTYPE_SVCCTX,
                            (void*) &callTimeout, 0, DPI_OCI_ATTR_CALL_TIMEOUT,
                            NULL, error);
                if (callTimeout > 0) {
                    dpiError__set(error, action, DPI_ERR_CALL_TIMEOUT,
                            callTimeout, error->buffer->code);
                    error->buffer->code = 0;
                }
                break;
        }
    }

    return DPI_FAILURE;
}


//-----------------------------------------------------------------------------
// dpiError__getInfo() [INTERNAL]
//   Get the error state from the error structure. Returns DPI_FAILURE as a
// convenience to the caller.
//-----------------------------------------------------------------------------
int dpiError__getInfo(dpiError *error, dpiErrorInfo *info)
{
    if (!info)
        return DPI_FAILURE;
    info->code = error->buffer->code;
    info->offset = error->buffer->offset;
    info->message = error->buffer->message;
    info->messageLength = error->buffer->messageLength;
    info->fnName = error->buffer->fnName;
    info->action = error->buffer->action;
    info->isRecoverable = error->buffer->isRecoverable;
    info->encoding = error->buffer->encoding;
    switch(info->code) {
        case 12154: // TNS:could not resolve the connect identifier specified
            info->sqlState = "42S02";
            break;
        case    22: // invalid session ID; access denied
        case   378: // buffer pools cannot be created as specified
        case   602: // Internal programming exception
        case   603: // ORACLE server session terminated by fatal error
        case   604: // error occurred at recursive SQL level
        case   609: // could not attach to incoming connection
        case  1012: // not logged on
        case  1033: // ORACLE initialization or shutdown in progress
        case  1041: // internal error. hostdef extension doesn't exist
        case  1043: // user side memory corruption
        case  1089: // immediate shutdown or close in progress
        case  1090: // shutdown in progress
        case  1092: // ORACLE instance terminated. Disconnection forced
        case  3113: // end-of-file on communication channel
        case  3114: // not connected to ORACLE
        case  3122: // attempt to close ORACLE-side window on user side
        case  3135: // connection lost contact
        case 12153: // TNS:not connected
        case 27146: // post/wait initialization failed
        case 28511: // lost RPC connection to heterogeneous remote agent
            info->sqlState = "01002";
            break;
        default:
            if (error->buffer->code == 0 &&
                    error->buffer->errorNum == (dpiErrorNum) 0)
                info->sqlState = "00000";
            else info->sqlState = "HY000";
            break;
    }
    return DPI_FAILURE;
}


//-----------------------------------------------------------------------------
// dpiError__set() [INTERNAL]
//   Set the error buffer to the specified DPI error. Returns DPI_FAILURE as a
// convenience to the caller.
//-----------------------------------------------------------------------------
int dpiError__set(dpiError *error, const char *action, dpiErrorNum errorNum,
        ...)
{
    va_list varArgs;

    if (error) {
        error->buffer->code = 0;
        error->buffer->isRecoverable = 0;
        error->buffer->offset = 0;
        strcpy(error->buffer->encoding, DPI_CHARSET_NAME_UTF8);
        error->buffer->action = action;
        error->buffer->errorNum = errorNum;
        va_start(varArgs, errorNum);
        error->buffer->messageLength =
                (uint32_t) vsnprintf(error->buffer->message,
                sizeof(error->buffer->message),
                dpiErrorMessages[errorNum - DPI_ERR_NO_ERR], varArgs);
        va_end(varArgs);
        if (dpiDebugLevel & DPI_DEBUG_LEVEL_ERRORS)
            dpiDebug__print("internal error %.*s (%s / %s)\n",
                    error->buffer->messageLength, error->buffer->message,
                    error->buffer->fnName, action);
    }
    return DPI_FAILURE;
}

