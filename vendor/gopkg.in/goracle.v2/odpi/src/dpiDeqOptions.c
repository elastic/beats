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
// dpiDeqOptions.c
//   Implementation of AQ dequeue options.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiDeqOptions__create() [INTERNAL]
//   Create a new subscription structure and return it. In case of error NULL
// is returned.
//-----------------------------------------------------------------------------
int dpiDeqOptions__create(dpiDeqOptions *options, dpiConn *conn,
        dpiError *error)
{
    dpiGen__setRefCount(conn, error, 1);
    options->conn = conn;
    return dpiOci__descriptorAlloc(conn->env->handle, &options->handle,
            DPI_OCI_DTYPE_AQDEQ_OPTIONS, "allocate descriptor", error);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions__free() [INTERNAL]
//   Free the memory for a dequeue options structure.
//-----------------------------------------------------------------------------
void dpiDeqOptions__free(dpiDeqOptions *options, dpiError *error)
{
    if (options->handle) {
        dpiOci__descriptorFree(options->handle, DPI_OCI_DTYPE_AQDEQ_OPTIONS);
        options->handle = NULL;
    }
    if (options->conn) {
        dpiGen__setRefCount(options->conn, error, -1);
        options->conn = NULL;
    }
    dpiUtils__freeMemory(options);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions__getAttrValue() [INTERNAL]
//   Get the attribute value in OCI.
//-----------------------------------------------------------------------------
static int dpiDeqOptions__getAttrValue(dpiDeqOptions *options,
        uint32_t attribute, const char *fnName, void *value,
        uint32_t *valueLength)
{
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(options, DPI_HTYPE_DEQ_OPTIONS, fnName,
            &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(options, value)
    DPI_CHECK_PTR_NOT_NULL(options, valueLength)
    status = dpiOci__attrGet(options->handle, DPI_OCI_DTYPE_AQDEQ_OPTIONS,
            value, valueLength, attribute, "get attribute value", &error);
    return dpiGen__endPublicFn(options, status, &error);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions__setAttrValue() [INTERNAL]
//   Set the attribute value in OCI.
//-----------------------------------------------------------------------------
static int dpiDeqOptions__setAttrValue(dpiDeqOptions *options,
        uint32_t attribute, const char *fnName, const void *value,
        uint32_t valueLength)
{
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(options, DPI_HTYPE_DEQ_OPTIONS, fnName,
            &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(options, value)
    status = dpiOci__attrSet(options->handle, DPI_OCI_DTYPE_AQDEQ_OPTIONS,
            (void*) value, valueLength, attribute, "set attribute value",
            &error);
    return dpiGen__endPublicFn(options, status, &error);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_addRef() [PUBLIC]
//   Add a reference to the dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_addRef(dpiDeqOptions *options)
{
    return dpiGen__addRef(options, DPI_HTYPE_DEQ_OPTIONS, __func__);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getCondition() [PUBLIC]
//   Return condition associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getCondition(dpiDeqOptions *options, const char **value,
        uint32_t *valueLength)
{
    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_DEQCOND, __func__,
            (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getConsumerName() [PUBLIC]
//   Return consumer name associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getConsumerName(dpiDeqOptions *options, const char **value,
        uint32_t *valueLength)
{
    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_CONSUMER_NAME,
            __func__, (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getCorrelation() [PUBLIC]
//   Return correlation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getCorrelation(dpiDeqOptions *options, const char **value,
        uint32_t *valueLength)
{
    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_CORRELATION,
            __func__, (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getMode() [PUBLIC]
//   Return mode associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getMode(dpiDeqOptions *options, dpiDeqMode *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_DEQ_MODE,
            __func__, value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getMsgId() [PUBLIC]
//   Return message id associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getMsgId(dpiDeqOptions *options, const char **value,
        uint32_t *valueLength)
{
    dpiError error;
    void *rawValue;

    if (dpiGen__startPublicFn(options, DPI_HTYPE_DEQ_OPTIONS, __func__,
            &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(options, value)
    DPI_CHECK_PTR_NOT_NULL(options, valueLength)
    if (dpiOci__attrGet(options->handle, DPI_OCI_DTYPE_AQDEQ_OPTIONS,
            &rawValue, NULL, DPI_OCI_ATTR_DEQ_MSGID, "get attribute value",
            &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    dpiOci__rawPtr(options->env->handle, rawValue, (void**) value);
    dpiOci__rawSize(options->env->handle, rawValue, valueLength);
    return dpiGen__endPublicFn(options, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getNavigation() [PUBLIC]
//   Return navigation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getNavigation(dpiDeqOptions *options,
        dpiDeqNavigation *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_NAVIGATION,
            __func__, value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getTransformation() [PUBLIC]
//   Return transformation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getTransformation(dpiDeqOptions *options, const char **value,
        uint32_t *valueLength)
{
    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_TRANSFORMATION,
            __func__, (void*) value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getVisibility() [PUBLIC]
//   Return visibility associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getVisibility(dpiDeqOptions *options, dpiVisibility *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_VISIBILITY,
            __func__, value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_getWait() [PUBLIC]
//   Return the number of seconds to wait for a message when dequeuing.
//-----------------------------------------------------------------------------
int dpiDeqOptions_getWait(dpiDeqOptions *options, uint32_t *value)
{
    uint32_t valueLength = sizeof(uint32_t);

    return dpiDeqOptions__getAttrValue(options, DPI_OCI_ATTR_WAIT, __func__,
            value, &valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_release() [PUBLIC]
//   Release a reference to the dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_release(dpiDeqOptions *options)
{
    return dpiGen__release(options, DPI_HTYPE_DEQ_OPTIONS, __func__);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setCondition() [PUBLIC]
//   Set condition associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setCondition(dpiDeqOptions *options, const char *value,
        uint32_t valueLength)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_DEQCOND, __func__,
            value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setConsumerName() [PUBLIC]
//   Set consumer name associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setConsumerName(dpiDeqOptions *options, const char *value,
        uint32_t valueLength)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_CONSUMER_NAME,
            __func__, value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setCorrelation() [PUBLIC]
//   Set correlation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setCorrelation(dpiDeqOptions *options, const char *value,
        uint32_t valueLength)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_CORRELATION,
            __func__, value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setDeliveryMode() [PUBLIC]
//   Set the delivery mode associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setDeliveryMode(dpiDeqOptions *options,
        dpiMessageDeliveryMode value)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_MSG_DELIVERY_MODE,
            __func__, &value, 0);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setMode() [PUBLIC]
//   Set the mode associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setMode(dpiDeqOptions *options, dpiDeqMode value)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_DEQ_MODE,
            __func__, &value, 0);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setMsgId() [PUBLIC]
//   Set the message id associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setMsgId(dpiDeqOptions *options, const char *value,
        uint32_t valueLength)
{
    void *rawValue = NULL;
    dpiError error;
    int status;

    if (dpiGen__startPublicFn(options, DPI_HTYPE_DEQ_OPTIONS, __func__,
            &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(options, value)
    if (dpiOci__rawAssignBytes(options->env->handle, value, valueLength,
            &rawValue, &error) < 0)
        return dpiGen__endPublicFn(options, DPI_FAILURE, &error);
    status = dpiOci__attrSet(options->handle, DPI_OCI_DTYPE_AQDEQ_OPTIONS,
            (void*) &rawValue, valueLength, DPI_OCI_ATTR_DEQ_MSGID,
            "set value", &error);
    dpiOci__rawResize(options->env->handle, &rawValue, 0, &error);
    return dpiGen__endPublicFn(options, status, &error);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setNavigation() [PUBLIC]
//   Set navigation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setNavigation(dpiDeqOptions *options, dpiDeqNavigation value)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_NAVIGATION,
            __func__, &value, 0);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setTransformation() [PUBLIC]
//   Set transformation associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setTransformation(dpiDeqOptions *options, const char *value,
        uint32_t valueLength)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_TRANSFORMATION,
            __func__, value, valueLength);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setVisibility() [PUBLIC]
//   Set visibility associated with dequeue options.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setVisibility(dpiDeqOptions *options, dpiVisibility value)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_VISIBILITY,
            __func__, &value, 0);
}


//-----------------------------------------------------------------------------
// dpiDeqOptions_setWait() [PUBLIC]
//   Set the number of seconds to wait for a message when dequeuing.
//-----------------------------------------------------------------------------
int dpiDeqOptions_setWait(dpiDeqOptions *options, uint32_t value)
{
    return dpiDeqOptions__setAttrValue(options, DPI_OCI_ATTR_WAIT, __func__,
            &value, 0);
}
