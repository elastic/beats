//-----------------------------------------------------------------------------
// Copyright (c) 2019, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiQueue.c
//   Implementation of AQ queues.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// forward declarations of internal functions only used in this file
static int dpiQueue__allocateBuffer(dpiQueue *queue, uint32_t numElements,
        dpiError *error);
static int dpiQueue__deq(dpiQueue *queue, uint32_t *numProps,
        dpiMsgProps **props, dpiError *error);
static void dpiQueue__freeBuffer(dpiQueue *queue, dpiError *error);
static int dpiQueue__getPayloadTDO(dpiQueue *queue, void **tdo,
        dpiError *error);


//-----------------------------------------------------------------------------
// dpiQueue__allocate() [INTERNAL]
//   Allocate and initialize a queue.
//-----------------------------------------------------------------------------
int dpiQueue__allocate(dpiConn *conn, const char *name, uint32_t nameLength,
        dpiObjectType *payloadType, dpiQueue **queue, dpiError *error)
{
    dpiQueue *tempQueue;
    char *buffer;

    // allocate handle; store reference to the connection that created it
    if (dpiGen__allocate(DPI_HTYPE_QUEUE, conn->env, (void**) &tempQueue,
            error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(conn, error, 1);
    tempQueue->conn = conn;

    // store payload type, which is either an object type or NULL (meaning that
    // RAW payloads are being enqueued and dequeued)
    if (payloadType) {
        dpiGen__setRefCount(payloadType, error, 1);
        tempQueue->payloadType = payloadType;
    }

    // allocate space for the name of the queue; OCI requires a NULL-terminated
    // string so allocate enough space to store the NULL terminator; UTF-16
    // encoded strings are not currently supported
    if (dpiUtils__allocateMemory(1, nameLength + 1, 0, "queue name",
            (void**) &buffer, error) < 0) {
        dpiQueue__free(tempQueue, error);
        return DPI_FAILURE;
    }
    memcpy(buffer, name, nameLength);
    buffer[nameLength] = '\0';
    tempQueue->name = buffer;

    *queue = tempQueue;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__allocateBuffer() [INTERNAL]
//   Ensure there is enough space in the buffer for the specified number of
// elements.
//-----------------------------------------------------------------------------
static int dpiQueue__allocateBuffer(dpiQueue *queue, uint32_t numElements,
        dpiError *error)
{
    dpiQueue__freeBuffer(queue, error);
    queue->buffer.numElements = numElements;
    if (dpiUtils__allocateMemory(numElements, sizeof(dpiMsgProps*), 1,
            "allocate msg props array", (void**) &queue->buffer.props,
            error) < 0)
        return DPI_FAILURE;
    if (dpiUtils__allocateMemory(numElements, sizeof(void*), 1,
            "allocate OCI handles array", (void**) &queue->buffer.handles,
            error) < 0)
        return DPI_FAILURE;
    if (dpiUtils__allocateMemory(numElements, sizeof(void*), 1,
            "allocate OCI instances array", (void**) &queue->buffer.instances,
            error) < 0)
        return DPI_FAILURE;
    if (dpiUtils__allocateMemory(numElements, sizeof(void*), 1,
            "allocate OCI indicators array",
            (void**) &queue->buffer.indicators, error) < 0)
        return DPI_FAILURE;
    if (!queue->payloadType) {
        if (dpiUtils__allocateMemory(numElements, sizeof(int16_t), 1,
                "allocate OCI raw indicators array",
                (void**) &queue->buffer.rawIndicators, error) < 0)
            return DPI_FAILURE;
    }
    if (dpiUtils__allocateMemory(numElements, sizeof(void*), 1,
            "allocate message ids array", (void**) &queue->buffer.msgIds,
            error) < 0)
        return DPI_FAILURE;

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__check() [INTERNAL]
//   Determine if the queue is available to use.
//-----------------------------------------------------------------------------
static int dpiQueue__check(dpiQueue *queue, const char *fnName,
        dpiError *error)
{
    if (dpiGen__startPublicFn(queue, DPI_HTYPE_QUEUE, fnName, error) < 0)
        return DPI_FAILURE;
    if (!queue->conn->handle || queue->conn->closing)
        return dpiError__set(error, "check connection", DPI_ERR_NOT_CONNECTED);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__createDeqOptions() [INTERNAL]
//   Create the dequeue options object that will be used for performing
// dequeues against the queue.
//-----------------------------------------------------------------------------
static int dpiQueue__createDeqOptions(dpiQueue *queue, dpiError *error)
{
    dpiDeqOptions *tempOptions;

    if (dpiGen__allocate(DPI_HTYPE_DEQ_OPTIONS, queue->env,
            (void**) &tempOptions, error) < 0)
        return DPI_FAILURE;
    if (dpiDeqOptions__create(tempOptions, queue->conn, error) < 0) {
        dpiDeqOptions__free(tempOptions, error);
        return DPI_FAILURE;
    }

    queue->deqOptions = tempOptions;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__createEnqOptions() [INTERNAL]
//   Create the dequeue options object that will be used for performing
// dequeues against the queue.
//-----------------------------------------------------------------------------
static int dpiQueue__createEnqOptions(dpiQueue *queue, dpiError *error)
{
    dpiEnqOptions *tempOptions;

    if (dpiGen__allocate(DPI_HTYPE_ENQ_OPTIONS, queue->env,
            (void**) &tempOptions, error) < 0)
        return DPI_FAILURE;
    if (dpiEnqOptions__create(tempOptions, queue->conn, error) < 0) {
        dpiEnqOptions__free(tempOptions, error);
        return DPI_FAILURE;
    }

    queue->enqOptions = tempOptions;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__deq() [INTERNAL]
//   Perform a dequeue of up to the specified number of properties.
//-----------------------------------------------------------------------------
static int dpiQueue__deq(dpiQueue *queue, uint32_t *numProps,
        dpiMsgProps **props, dpiError *error)
{
    dpiMsgProps *prop;
    void *payloadTDO;
    uint32_t i;

    // create dequeue options, if necessary
    if (!queue->deqOptions && dpiQueue__createDeqOptions(queue, error) < 0)
        return DPI_FAILURE;

    // allocate buffer, if necessary
    if (queue->buffer.numElements < *numProps &&
            dpiQueue__allocateBuffer(queue, *numProps, error) < 0)
        return DPI_FAILURE;

    // populate buffer
    for (i = 0; i < *numProps; i++) {
        prop = queue->buffer.props[i];

        // create new message properties, if applicable
        if (!prop) {
            if (dpiMsgProps__allocate(queue->conn, &prop, error) < 0)
                return DPI_FAILURE;
            queue->buffer.props[i] = prop;
        }

        // create payload object, if applicable
        if (queue->payloadType && !prop->payloadObj &&
                dpiObject__allocate(queue->payloadType, NULL, NULL, NULL,
                &prop->payloadObj, error) < 0)
            return DPI_FAILURE;

        // set OCI arrays
        queue->buffer.handles[i] = prop->handle;
        if (queue->payloadType) {
            queue->buffer.instances[i] = prop->payloadObj->instance;
            queue->buffer.indicators[i] = prop->payloadObj->indicator;
        } else {
            queue->buffer.instances[i] = prop->payloadRaw;
            queue->buffer.indicators[i] = &queue->buffer.rawIndicators[i];
        }
        queue->buffer.msgIds[i] = prop->msgIdRaw;

    }

    // perform dequeue
    if (dpiQueue__getPayloadTDO(queue, &payloadTDO, error) < 0)
        return DPI_FAILURE;
    if (dpiOci__aqDeqArray(queue->conn, queue->name, queue->deqOptions->handle,
            numProps, queue->buffer.handles, payloadTDO,
            queue->buffer.instances, queue->buffer.indicators,
            queue->buffer.msgIds, error) < 0) {
        if (error->buffer->code != 25228)
            return DPI_FAILURE;
        error->buffer->offset = (uint16_t) *numProps;
    }

    // transfer message properties to destination array
    for (i = 0; i < *numProps; i++) {
        props[i] = queue->buffer.props[i];
        queue->buffer.props[i] = NULL;
        if (!queue->payloadType)
            props[i]->payloadRaw = queue->buffer.instances[i];
        props[i]->msgIdRaw = queue->buffer.msgIds[i];
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__enq() [INTERNAL]
//   Perform an enqueue of the specified properties.
//-----------------------------------------------------------------------------
static int dpiQueue__enq(dpiQueue *queue, uint32_t numProps,
        dpiMsgProps **props, dpiError *error)
{
    void *payloadTDO;
    uint32_t i;

    // if no messages are being enqueued, nothing to do!
    if (numProps == 0)
        return DPI_SUCCESS;

    // create enqueue options, if necessary
    if (!queue->enqOptions && dpiQueue__createEnqOptions(queue, error) < 0)
        return DPI_FAILURE;

    // allocate buffer, if necessary
    if (queue->buffer.numElements < numProps &&
            dpiQueue__allocateBuffer(queue, numProps, error) < 0)
        return DPI_FAILURE;

    // populate buffer
    for (i = 0; i < numProps; i++) {

        // perform checks
        if (!props[i]->payloadObj && !props[i]->payloadRaw)
            return dpiError__set(error, "check payload",
                    DPI_ERR_QUEUE_NO_PAYLOAD);
        if ((queue->payloadType && !props[i]->payloadObj) ||
                (!queue->payloadType && props[i]->payloadObj))
            return dpiError__set(error, "check payload",
                    DPI_ERR_QUEUE_WRONG_PAYLOAD_TYPE);
        if (queue->payloadType && props[i]->payloadObj &&
                queue->payloadType->tdo != props[i]->payloadObj->type->tdo)
            return dpiError__set(error, "check payload",
                    DPI_ERR_WRONG_TYPE,
                    props[i]->payloadObj->type->schemaLength,
                    props[i]->payloadObj->type->schema,
                    props[i]->payloadObj->type->nameLength,
                    props[i]->payloadObj->type->name,
                    queue->payloadType->schemaLength,
                    queue->payloadType->schema,
                    queue->payloadType->nameLength,
                    queue->payloadType->name);

        // set OCI arrays
        queue->buffer.handles[i] = props[i]->handle;
        if (queue->payloadType) {
            queue->buffer.instances[i] = props[i]->payloadObj->instance;
            queue->buffer.indicators[i] = props[i]->payloadObj->indicator;
        } else {
            queue->buffer.instances[i] = props[i]->payloadRaw;
            queue->buffer.indicators[i] = &queue->buffer.rawIndicators[i];
        }
        queue->buffer.msgIds[i] = props[i]->msgIdRaw;

    }

    // perform enqueue
    if (dpiQueue__getPayloadTDO(queue, &payloadTDO, error) < 0)
        return DPI_FAILURE;
    if (numProps == 1) {
        if (dpiOci__aqEnq(queue->conn, queue->name, queue->enqOptions->handle,
                queue->buffer.handles[0], payloadTDO, queue->buffer.instances,
                queue->buffer.indicators, queue->buffer.msgIds, error) < 0)
            return DPI_FAILURE;
    } else {
        if (dpiOci__aqEnqArray(queue->conn, queue->name,
                queue->enqOptions->handle, &numProps, queue->buffer.handles,
                payloadTDO, queue->buffer.instances, queue->buffer.indicators,
                queue->buffer.msgIds, error) < 0) {
            error->buffer->offset = (uint16_t) numProps;
            return DPI_FAILURE;
        }
    }

    // transfer message ids back to message properties
    for (i = 0; i < numProps; i++)
        props[i]->msgIdRaw = queue->buffer.msgIds[i];

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue__free() [INTERNAL]
//   Free the memory for a queue.
//-----------------------------------------------------------------------------
void dpiQueue__free(dpiQueue *queue, dpiError *error)
{
    if (queue->conn) {
        dpiGen__setRefCount(queue->conn, error, -1);
        queue->conn = NULL;
    }
    if (queue->payloadType) {
        dpiGen__setRefCount(queue->payloadType, error, -1);
        queue->payloadType = NULL;
    }
    if (queue->name) {
        dpiUtils__freeMemory((void*) queue->name);
        queue->name = NULL;
    }
    if (queue->deqOptions) {
        dpiGen__setRefCount(queue->deqOptions, error, -1);
        queue->deqOptions = NULL;
    }
    if (queue->enqOptions) {
        dpiGen__setRefCount(queue->enqOptions, error, -1);
        queue->enqOptions = NULL;
    }
    dpiQueue__freeBuffer(queue, error);
    dpiUtils__freeMemory(queue);
}


//-----------------------------------------------------------------------------
// dpiQueue__freeBuffer() [INTERNAL]
//   Free the memory areas in the queue buffer.
//-----------------------------------------------------------------------------
static void dpiQueue__freeBuffer(dpiQueue *queue, dpiError *error)
{
    dpiQueueBuffer *buffer = &queue->buffer;
    uint32_t i;

    if (buffer->props) {
        for (i = 0; i < buffer->numElements; i++) {
            if (buffer->props[i]) {
                dpiGen__setRefCount(buffer->props[i], error, -1);
                buffer->props[i] = NULL;
            }
        }
        dpiUtils__freeMemory(buffer->props);
        buffer->props = NULL;
    }
    if (buffer->handles) {
        dpiUtils__freeMemory(buffer->handles);
        buffer->handles = NULL;
    }
    if (buffer->instances) {
        dpiUtils__freeMemory(buffer->instances);
        buffer->instances = NULL;
    }
    if (buffer->indicators) {
        dpiUtils__freeMemory(buffer->indicators);
        buffer->indicators = NULL;
    }
    if (buffer->rawIndicators) {
        dpiUtils__freeMemory(buffer->rawIndicators);
        buffer->rawIndicators = NULL;
    }
    if (buffer->msgIds) {
        dpiUtils__freeMemory(buffer->msgIds);
        buffer->msgIds = NULL;
    }
}


//-----------------------------------------------------------------------------
// dpiQueue__getPayloadTDO() [INTERNAL]
//   Acquire the TDO to use for the payload. This will either be the TDO of the
// object type (if one was specified when the queue was created) or it will be
// the RAW TDO cached on the connection.
//-----------------------------------------------------------------------------
static int dpiQueue__getPayloadTDO(dpiQueue *queue, void **tdo,
        dpiError *error)
{
    if (queue->payloadType) {
        *tdo = queue->payloadType->tdo;
    } else {
        if (dpiConn__getRawTDO(queue->conn, error) < 0)
            return DPI_FAILURE;
        *tdo = queue->conn->rawTDO;
    }
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiQueue_addRef() [PUBLIC]
//   Add a reference to the queue.
//-----------------------------------------------------------------------------
int dpiQueue_addRef(dpiQueue *queue)
{
    return dpiGen__addRef(queue, DPI_HTYPE_QUEUE, __func__);
}


//-----------------------------------------------------------------------------
// dpiQueue_deqMany() [PUBLIC]
//   Dequeue multiple messages from the queue.
//-----------------------------------------------------------------------------
int dpiQueue_deqMany(dpiQueue *queue, uint32_t *numProps, dpiMsgProps **props)
{
    dpiError error;
    int status;

    if (dpiQueue__check(queue, __func__, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(queue, numProps)
    DPI_CHECK_PTR_NOT_NULL(queue, props)
    status = dpiQueue__deq(queue, numProps, props, &error);
    return dpiGen__endPublicFn(queue, status, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_deqOne() [PUBLIC]
//   Dequeue a single message from the queue.
//-----------------------------------------------------------------------------
int dpiQueue_deqOne(dpiQueue *queue, dpiMsgProps **props)
{
    uint32_t numProps = 1;
    dpiError error;

    if (dpiQueue__check(queue, __func__, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(queue, props)
    if (dpiQueue__deq(queue, &numProps, props, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    if (numProps == 0)
        *props = NULL;
    return dpiGen__endPublicFn(queue, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_enqMany() [PUBLIC]
//   Enqueue multiple message to the queue.
//-----------------------------------------------------------------------------
int dpiQueue_enqMany(dpiQueue *queue, uint32_t numProps, dpiMsgProps **props)
{
    dpiError error;
    uint32_t i;
    int status;

    // validate parameters
    if (dpiQueue__check(queue, __func__, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(queue, props)
    for (i = 0; i < numProps; i++) {
        if (dpiGen__checkHandle(props[i], DPI_HTYPE_MSG_PROPS,
                "check message properties", &error) < 0)
            return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    }
    status = dpiQueue__enq(queue, numProps, props, &error);
    return dpiGen__endPublicFn(queue, status, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_enqOne() [PUBLIC]
//   Enqueue a single message to the queue.
//-----------------------------------------------------------------------------
int dpiQueue_enqOne(dpiQueue *queue, dpiMsgProps *props)
{
    dpiError error;
    int status;

    if (dpiQueue__check(queue, __func__, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    if (dpiGen__checkHandle(props, DPI_HTYPE_MSG_PROPS,
            "check message properties", &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    status = dpiQueue__enq(queue, 1, &props, &error);
    return dpiGen__endPublicFn(queue, status, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_getDeqOptions() [PUBLIC]
//   Return the dequeue options associated with the queue. If no dequeue
// options are currently associated with the queue, create them first.
//-----------------------------------------------------------------------------
int dpiQueue_getDeqOptions(dpiQueue *queue, dpiDeqOptions **options)
{
    dpiError error;

    if (dpiGen__startPublicFn(queue, DPI_HTYPE_QUEUE, __func__, &error) < 0)
        return DPI_FAILURE;
    DPI_CHECK_PTR_NOT_NULL(queue, options)
    if (!queue->deqOptions && dpiQueue__createDeqOptions(queue, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    *options = queue->deqOptions;
    return dpiGen__endPublicFn(queue, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_getEnqOptions() [PUBLIC]
//   Return the enqueue options associated with the queue. If no enqueue
// options are currently associated with the queue, create them first.
//-----------------------------------------------------------------------------
int dpiQueue_getEnqOptions(dpiQueue *queue, dpiEnqOptions **options)
{
    dpiError error;

    if (dpiGen__startPublicFn(queue, DPI_HTYPE_QUEUE, __func__, &error) < 0)
        return DPI_FAILURE;
    DPI_CHECK_PTR_NOT_NULL(queue, options)
    if (!queue->enqOptions && dpiQueue__createEnqOptions(queue, &error) < 0)
        return dpiGen__endPublicFn(queue, DPI_FAILURE, &error);
    *options = queue->enqOptions;
    return dpiGen__endPublicFn(queue, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiQueue_release() [PUBLIC]
//   Release a reference to the queue.
//-----------------------------------------------------------------------------
int dpiQueue_release(dpiQueue *queue)
{
    return dpiGen__release(queue, DPI_HTYPE_QUEUE, __func__);
}
