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
// dpiMsgProps.c
//   Implementation of AQ message properties.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiMsgProps__create() [INTERNAL]
//   Create a new subscription structure and return it. In case of error NULL
// is returned.
//-----------------------------------------------------------------------------
int dpiMsgProps__create(dpiMsgProps *options, dpiConn *conn, dpiError *error)
{
    dpiGen__setRefCount(conn, error, 1);
    options->conn = conn;
    return dpiOci__descriptorAlloc(conn->env->handle, &options->handle,
            DPI_OCI_DTYPE_AQMSG_PROPERTIES, "allocate descriptor", error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps__extractMsgId() [INTERNAL]
//   Extract bytes from the OCIRaw value containing the message id and store
// them in allocated memory on the message properties instance. Then resize the
// OCIRaw value so the memory can be reclaimed.
//-----------------------------------------------------------------------------
int dpiMsgProps__extractMsgId(dpiMsgProps *props, void *ociRaw,
        const char **msgId, uint32_t *msgIdLength, dpiError *error)
{
    const char *rawPtr;

    dpiOci__rawPtr(props->env->handle, ociRaw, (void**) &rawPtr);
    dpiOci__rawSize(props->env->handle, ociRaw, msgIdLength);
    if (*msgIdLength > props->bufferLength) {
        if (props->buffer) {
            dpiUtils__freeMemory(props->buffer);
            props->buffer = NULL;
        }
        if (dpiUtils__allocateMemory(1, *msgIdLength, 0,
                "allocate msgid buffer", (void**) &props->buffer, error) < 0)
            return DPI_FAILURE;
    }
    memcpy(props->buffer, rawPtr, *msgIdLength);
    *msgId = props->buffer;
    dpiOci__rawResize(props->env->handle, &ociRaw, 0, error);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiMsgProps__free() [INTERNAL]
//   Free the memory for a message properties structure.
//-----------------------------------------------------------------------------
void dpiMsgProps__free(dpiMsgProps *props, dpiError *error)
{
    if (props->handle) {
        dpiOci__descriptorFree(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES);
        props->handle = NULL;
    }
    if (props->conn) {
        dpiGen__setRefCount(props->conn, error, -1);
        props->conn = NULL;
    }
    if (props->buffer) {
        dpiUtils__freeMemory(props->buffer);
        props->buffer = NULL;
    }
    dpiUtils__freeMemory(props);
}


//-----------------------------------------------------------------------------
// dpiMsgProps__getAttrValue() [INTERNAL]
//   Get the attribute value in OCI.
//-----------------------------------------------------------------------------
static int dpiMsgProps__getAttrValue(dpiMsgProps *props, uint32_t attribute,
        const char *fnName, void *value, uint32_t *valueLength)
{
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(props, DPI_HTYPE_MSG_PROPS, fnName, 1,
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(props, value)
    DPI_CHECK_PTR_NOT_NULL(props, valueLength)
    status = dpiOci__attrGet(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES,
            value, valueLength, attribute, "get attribute value", &error);
    return dpiGen__endPublicFn(props, status, &error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps__setAttrValue() [INTERNAL]
//   Set the attribute value in OCI.
//-----------------------------------------------------------------------------
static int dpiMsgProps__setAttrValue(dpiMsgProps *props, uint32_t attribute,
        const char *fnName, const void *value, uint32_t valueLength)
{
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(props, DPI_HTYPE_MSG_PROPS, fnName, 1,
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(props, value)
    status = dpiOci__attrSet(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES,
            (void*) value, valueLength, attribute, "set attribute value",
            &error);
    return dpiGen__endPublicFn(props, status, &error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_addRef() [PUBLIC]
//   Add a reference to the message properties.
//-----------------------------------------------------------------------------
int dpiMsgProps_addRef(dpiMsgProps *props)
{
    return dpiGen__addRef(props, DPI_HTYPE_MSG_PROPS, __func__);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getCorrelation() [PUBLIC]
//   Return correlation associated with the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getCorrelation(dpiMsgProps *props, const char **value,
        uint32_t *valueLength)
{
    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_CORRELATION, __func__,
            (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getDelay() [PUBLIC]
//   Return the number of seconds the message was delayed.
//-----------------------------------------------------------------------------
int dpiMsgProps_getDelay(dpiMsgProps *props, int32_t *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_DELAY, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getDeliveryMode() [PUBLIC]
//   Return the mode used for delivering the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getDeliveryMode(dpiMsgProps *props,
        dpiMessageDeliveryMode *value)
{
    uint32_t valueLength = sizeof(uint16_t);

    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_MSG_DELIVERY_MODE,
            __func__, value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getEnqTime() [PUBLIC]
//   Return the time the message was enqueued.
//-----------------------------------------------------------------------------
int dpiMsgProps_getEnqTime(dpiMsgProps *props, dpiTimestamp *value)
{
    dpiOciDate ociValue;
    dpiError error;

    if (dpiGen__startPublicFn(props, DPI_HTYPE_MSG_PROPS, __func__, 1,
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(props, value)
    if (dpiOci__attrGet(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES,
            &ociValue, NULL, DPI_OCI_ATTR_ENQ_TIME, "get attribute value",
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    value->year = ociValue.year;
    value->month = ociValue.month;
    value->day = ociValue.day;
    value->hour = ociValue.hour;
    value->minute = ociValue.minute;
    value->second = ociValue.second;
    value->fsecond = 0;
    value->tzHourOffset = 0;
    value->tzMinuteOffset = 0;
    return dpiGen__endPublicFn(props, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getExceptionQ() [PUBLIC]
//   Return the name of the exception queue associated with the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getExceptionQ(dpiMsgProps *props, const char **value,
        uint32_t *valueLength)
{
    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_EXCEPTION_QUEUE,
            __func__, (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getExpiration() [PUBLIC]
//   Return the number of seconds until the message expires.
//-----------------------------------------------------------------------------
int dpiMsgProps_getExpiration(dpiMsgProps *props, int32_t *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_EXPIRATION, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getNumAttempts() [PUBLIC]
//   Return the number of attempts made to deliver the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getNumAttempts(dpiMsgProps *props, int32_t *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_ATTEMPTS, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getOriginalMsgId() [PUBLIC]
//   Return the original message id for the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getOriginalMsgId(dpiMsgProps *props, const char **value,
        uint32_t *valueLength)
{
    dpiError error;
    void *rawValue;

    if (dpiGen__startPublicFn(props, DPI_HTYPE_MSG_PROPS, __func__, 1,
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(props, value)
    DPI_CHECK_PTR_NOT_NULL(props, valueLength)
    if (dpiOci__attrGet(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES,
            &rawValue, NULL, DPI_OCI_ATTR_ORIGINAL_MSGID,
            "get attribute value", &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    dpiOci__rawPtr(props->env->handle, rawValue, (void**) value);
    dpiOci__rawSize(props->env->handle, rawValue, valueLength);
    return dpiGen__endPublicFn(props, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getPriority() [PUBLIC]
//   Return the priority of the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getPriority(dpiMsgProps *props, int32_t *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_PRIORITY, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_getState() [PUBLIC]
//   Return the state of the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_getState(dpiMsgProps *props, dpiMessageState *value)
{
    uint32_t valueLength = sizeof(uint32_t);


    return dpiMsgProps__getAttrValue(props, DPI_OCI_ATTR_MSG_STATE, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_release() [PUBLIC]
//   Release a reference to the message properties.
//-----------------------------------------------------------------------------
int dpiMsgProps_release(dpiMsgProps *props)
{
    return dpiGen__release(props, DPI_HTYPE_MSG_PROPS, __func__);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setCorrelation() [PUBLIC]
//   Set correlation associated with the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_setCorrelation(dpiMsgProps *props, const char *value,
        uint32_t valueLength)
{
    return dpiMsgProps__setAttrValue(props, DPI_OCI_ATTR_CORRELATION, __func__,
            value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setDelay() [PUBLIC]
//   Set the number of seconds to delay the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_setDelay(dpiMsgProps *props, int32_t value)
{
    return dpiMsgProps__setAttrValue(props, DPI_OCI_ATTR_DELAY, __func__,
            &value, 0);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setExceptionQ() [PUBLIC]
//   Set the name of the exception queue associated with the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_setExceptionQ(dpiMsgProps *props, const char *value,
        uint32_t valueLength)
{
    return dpiMsgProps__setAttrValue(props, DPI_OCI_ATTR_EXCEPTION_QUEUE,
            __func__, value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setExpiration() [PUBLIC]
//   Set the number of seconds until the message expires.
//-----------------------------------------------------------------------------
int dpiMsgProps_setExpiration(dpiMsgProps *props, int32_t value)
{
    return dpiMsgProps__setAttrValue(props, DPI_OCI_ATTR_EXPIRATION, __func__,
            &value, 0);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setOriginalMsgId() [PUBLIC]
//   Set the original message id for the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_setOriginalMsgId(dpiMsgProps *props, const char *value,
        uint32_t valueLength)
{
    void *rawValue = NULL;
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(props, DPI_HTYPE_MSG_PROPS, __func__, 1,
            &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(props, value)
    if (dpiOci__rawAssignBytes(props->env->handle, value, valueLength,
            &rawValue, &error) < 0)
        return dpiGen__endPublicFn(props, DPI_FAILURE, &error);
    status = dpiOci__attrSet(props->handle, DPI_OCI_DTYPE_AQMSG_PROPERTIES,
            (void*) rawValue, 0, DPI_OCI_ATTR_ORIGINAL_MSGID, "set value",
            &error);
    dpiOci__rawResize(props->env->handle, &rawValue, 0, &error);
    return dpiGen__endPublicFn(props, status, &error);
}


//-----------------------------------------------------------------------------
// dpiMsgProps_setPriority() [PUBLIC]
//   Set the priority of the message.
//-----------------------------------------------------------------------------
int dpiMsgProps_setPriority(dpiMsgProps *props, int32_t value)
{
    return dpiMsgProps__setAttrValue(props, DPI_OCI_ATTR_PRIORITY, __func__,
            &value, 0);
}

