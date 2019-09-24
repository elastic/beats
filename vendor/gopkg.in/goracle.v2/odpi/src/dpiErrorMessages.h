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
// dpiErrorMessages.h
//   Definition of error messages used in ODPI-C.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

static const char* const dpiErrorMessages[DPI_ERR_MAX - DPI_ERR_NO_ERR] = {
    "DPI-1000: no error", // DPI_ERR_NO_ERR
    "DPI-1001: out of memory", // DPI_ERR_NO_MEMORY
    "DPI-1002: invalid %s handle", // DPI_ERR_INVALID_HANDLE
    "DPI-1003: OCI error handle is not initialized", // DPI_ERR_ERR_NOT_INITIALIZED
    "DPI-1004: unable to get error message", // DPI_ERR_GET_FAILED
    "DPI-1005: unable to acquire Oracle environment handle", // DPI_ERR_CREATE_ENV
    "DPI-1006: unable to convert text to session character set", // DPI_ERR_CONVERT_TEXT
    "DPI-1007: no query has been executed", // DPI_ERR_QUERY_NOT_EXECUTED
    "DPI-1008: data type %d is not supported", // DPI_ERR_UNHANDLED_DATA_TYPE
    "DPI-1009: zero-based position %u is not valid with max array size of %u", // DPI_ERR_INVALID_ARRAY_POSITION
    "DPI-1010: not connected", // DPI_ERR_NOT_CONNECTED
    "DPI-1011: connection was not acquired from a session pool", // DPI_ERR_CONN_NOT_IN_POOL
    "DPI-1012: proxy authentication is not possible with homogeneous pools", // DPI_ERR_INVALID_PROXY
    "DPI-1013: not supported", // DPI_ERR_NOT_SUPPORTED
    "DPI-1014: conversion between Oracle type %d and native type %d is not implemented", // DPI_ERR_UNHANDLED_CONVERSION
    "DPI-1015: array size of %u is too large", // DPI_ERR_ARRAY_SIZE_TOO_BIG
    "DPI-1016: invalid date", // DPI_ERR_INVALID_DATE
    "DPI-1017: value is null", // DPI_ERR_VALUE_IS_NULL
    "DPI-1018: array size of %u is too small", // DPI_ERR_ARRAY_SIZE_TOO_SMALL
    "DPI-1019: buffer size of %u is too small", // DPI_ERR_BUFFER_SIZE_TOO_SMALL
    "DPI-1020: application requires ODPI-C %d (min %d.%d) but is using a shared library at version %d.%d", // DPI_ERR_VERSION_NOT_SUPPORTED
    "DPI-1021: Oracle type %u is invalid", // DPI_ERR_INVALID_ORACLE_TYPE
    "DPI-1022: attribute %.*s is not part of object type %.*s.%.*s", // DPI_ERR_WRONG_ATTR
    "DPI-1023: object %.*s.%.*s is not a collection", // DPI_ERR_NOT_COLLECTION
    "DPI-1024: element at index %d does not exist", // DPI_ERR_INVALID_INDEX
    "DPI-1025: no object type specified for object variable", // DPI_ERR_NO_OBJECT_TYPE
    "DPI-1026: invalid character set %s", // DPI_ERR_INVALID_CHARSET
    "DPI-1027: scroll operation would go out of the result set", // DPI_ERR_SCROLL_OUT_OF_RS
    "DPI-1028: query position %u is invalid", // DPI_ERR_QUERY_POSITION_INVALID
    "DPI-1029: no row currently fetched", // DPI_ERR_NO_ROW_FETCHED
    "DPI-1030: unable to get or set error structure for thread local storage", // DPI_ERR_TLS_ERROR
    "DPI-1031: array size cannot be zero", // DPI_ERR_ARRAY_SIZE_ZERO
    "DPI-1032: user name and password cannot be set when using external authentication", // DPI_ERR_EXT_AUTH_WITH_CREDENTIALS
    "DPI-1033: unable to get row offset", // DPI_ERR_CANNOT_GET_ROW_OFFSET
    "DPI-1034: connection created from external handle cannot be closed", // DPI_ERR_CONN_IS_EXTERNAL
    "DPI-1035: size of the transaction ID is %u and cannot exceed %u", // DPI_ERR_TRANS_ID_TOO_LARGE
    "DPI-1036: size of the branch ID is %u and cannot exceed %u", // DPI_ERR_BRANCH_ID_TOO_LARGE
    "DPI-1037: column at array position %u fetched with error %u", // DPI_ERR_COLUMN_FETCH
    "DPI-1039: statement was already closed", // DPI_ERR_STMT_CLOSED
    "DPI-1040: LOB was already closed", // DPI_ERR_LOB_CLOSED
    "DPI-1041: invalid character set id %d", // DPI_ERR_INVALID_CHARSET_ID
    "DPI-1042: invalid OCI number", // DPI_ERR_INVALID_OCI_NUMBER
    "DPI-1043: invalid number", // DPI_ERR_INVALID_NUMBER
    "DPI-1044: value cannot be represented as an Oracle number", // DPI_ERR_NUMBER_NO_REPR
    "DPI-1045: strings converted to numbers can only be up to 172 characters long", // DPI_ERR_NUMBER_STRING_TOO_LONG
    "DPI-1046: parameter %s cannot be a NULL pointer", // DPI_ERR_NULL_POINTER_PARAMETER
    "DPI-1047: Cannot locate a %s-bit Oracle Client library: \"%s\". See https://oracle.github.io/odpi/doc/installation.html#%s for help", // DPI_ERR_LOAD_LIBRARY
    "DPI-1049: symbol %s not found in OCI library", // DPI_ERR_LOAD_SYMBOL
    "DPI-1050: Oracle Client library is at version %d.%d but version %d.%d or higher is needed", // DPI_ERR_ORACLE_CLIENT_TOO_OLD
    "DPI-1052: unable to get NLS environment variable", // DPI_ERR_NLS_ENV_VAR_GET,
    "DPI-1053: parameter %s cannot be a NULL pointer while corresponding length parameter is non-zero", // DPI_ERR_PTR_LENGTH_MISMATCH
    "DPI-1055: value is not a number (NaN) and cannot be used in Oracle numbers", // DPI_ERR_NAN
    "DPI-1056: found object of type %.*s.%.*s when expecting object of type %.*s.%.*s", // DPI_ERR_WRONG_TYPE
    "DPI-1057: buffer size of %u is too large (max %u)", // DPI_ERR_BUFFER_SIZE_TOO_LARGE
    "DPI-1058: edition not supported with connection class", // DPI_ERR_NO_EDITION_WITH_CONN_CLASS
    "DPI-1059: bind variables are not supported in DDL statements", // DPI_ERR_NO_BIND_VARS_IN_DDL
    "DPI-1060: subscription was already closed", // DPI_ERR_SUBSCR_CLOSED
    "DPI-1061: edition is not supported when a new password is specified", // DPI_ERR_NO_EDITION_WITH_NEW_PASSWORD
    "DPI-1062: unexpected OCI return value %d in function %s", // DPI_ERR_UNEXPECTED_OCI_RETURN_VALUE
    "DPI-1063: modes DPI_MODE_EXEC_BATCH_ERRORS and DPI_MODE_EXEC_ARRAY_DML_ROWCOUNTS can only be used with insert, update, delete and merge statements", // DPI_ERR_EXEC_MODE_ONLY_FOR_DML
    "DPI-1064: array variables are not supported with dpiStmt_executeMany()", // DPI_ERR_ARRAY_VAR_NOT_SUPPORTED
    "DPI-1065: events mode is required to subscribe to events in the database", // DPI_ERR_EVENTS_MODE_REQUIRED
    "DPI-1066: Oracle Database is at version %d.%d but version %d.%d or higher is needed", // DPI_ERR_ORACLE_DB_TOO_OLD
    "DPI-1067: call timeout of %u ms exceeded with ORA-%d", // DPI_ERR_CALL_TIMEOUT
    "DPI-1068: SODA cursor was already closed", // DPI_ERR_SODA_CURSOR_CLOSED
    "DPI-1069: proxy user name must be enclosed in [] when using external authentication", // DPI_ERR_EXT_AUTH_INVALID_PROXY
};

