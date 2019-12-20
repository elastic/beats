//-----------------------------------------------------------------------------
// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiHandleList.c
//   Implementation of a list of handles which are managed in a thread-safe
// manner. The references to these handles are assumed to be held by other
// structures. No references are held by the list of handles defined here.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiHandleList__addHandle() [INTERNAL]
//   Add a handle to the list. The list is expanded in sets of 8 handles as
// needed. A current position is maintained to reduce the number of scans of
// the list are required. An empty slot is designated by a NULL pointer.
//-----------------------------------------------------------------------------
int dpiHandleList__addHandle(dpiHandleList *list, void *handle,
        uint32_t *slotNum, dpiError *error)
{
    uint32_t numSlots, i;
    void **tempHandles;

    dpiMutex__acquire(list->mutex);
    if (list->numUsedSlots == list->numSlots) {
        numSlots = list->numSlots + 8;
        if (dpiUtils__allocateMemory(numSlots, sizeof(void*), 1,
                "allocate slots", (void**) &tempHandles, error) < 0) {
            dpiMutex__release(list->mutex);
            return DPI_FAILURE;
        }
        memcpy(tempHandles, list->handles, list->numSlots * sizeof(void*));
        dpiUtils__freeMemory(list->handles);
        list->handles = tempHandles;
        list->numSlots = numSlots;
        *slotNum = list->numUsedSlots++;
        list->currentPos = list->numUsedSlots + 1;
    } else {
        for (i = 0; i < list->numSlots; i++) {
            if (!list->handles[list->currentPos])
                break;
            list->currentPos++;
            if (list->currentPos == list->numSlots)
                list->currentPos = 0;
        }
        list->numUsedSlots++;
        *slotNum = list->currentPos++;
        if (list->currentPos == list->numSlots)
            list->currentPos = 0;
    }
    list->handles[*slotNum] = handle;
    dpiMutex__release(list->mutex);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiHandleList__create() [INTERNAL]
//   Create a new (empty) list of handles.
//-----------------------------------------------------------------------------
int dpiHandleList__create(dpiHandleList **list, dpiError *error)
{
    dpiHandleList *tempList;

    if (dpiUtils__allocateMemory(1, sizeof(dpiHandleList), 0,
            "allocate handle list", (void**) &tempList, error) < 0)
        return DPI_FAILURE;
    tempList->numSlots = 8;
    tempList->numUsedSlots = 0;
    if (dpiUtils__allocateMemory(tempList->numSlots, sizeof(void*), 1,
            "allocate handle list slots", (void**) &tempList->handles,
            error) < 0) {
        dpiUtils__freeMemory(tempList);
        return DPI_FAILURE;
    }
    dpiMutex__initialize(tempList->mutex);
    tempList->currentPos = 0;
    *list = tempList;
    return DPI_SUCCESS;
}

//-----------------------------------------------------------------------------
// dpiHandleList__free() [INTERNAL]
//   Free the memory associated with the handle list.
//-----------------------------------------------------------------------------
void dpiHandleList__free(dpiHandleList *list)
{
    if (list->handles) {
        dpiUtils__freeMemory(list->handles);
        list->handles = NULL;
    }
    dpiMutex__destroy(list->mutex);
    dpiUtils__freeMemory(list);
}


//-----------------------------------------------------------------------------
// dpiHandleList__removeHandle() [INTERNAL]
//   Remove the handle at the specified location from the list.
//-----------------------------------------------------------------------------
void dpiHandleList__removeHandle(dpiHandleList *list, uint32_t slotNum)
{
    dpiMutex__acquire(list->mutex);
    list->handles[slotNum] = NULL;
    list->numUsedSlots--;
    dpiMutex__release(list->mutex);
}

