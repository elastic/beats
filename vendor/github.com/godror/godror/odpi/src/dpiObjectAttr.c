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
// dpiObjectAttr.c
//   Implementation of object attributes.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

//-----------------------------------------------------------------------------
// dpiObjectAttr__allocate() [INTERNAL]
//   Allocate and initialize an object attribute structure.
//-----------------------------------------------------------------------------
int dpiObjectAttr__allocate(dpiObjectType *objType, void *param,
        dpiObjectAttr **attr, dpiError *error)
{
    dpiObjectAttr *tempAttr;

    // allocate and assign main reference to the type this attribute belongs to
    *attr = NULL;
    if (dpiGen__allocate(DPI_HTYPE_OBJECT_ATTR, objType->env,
            (void**) &tempAttr, error) < 0)
        return DPI_FAILURE;
    dpiGen__setRefCount(objType, error, 1);
    tempAttr->belongsToType = objType;

    // determine the name of the attribute
    if (dpiUtils__getAttrStringWithDup("get name", param, DPI_OCI_DTYPE_PARAM,
            DPI_OCI_ATTR_NAME, &tempAttr->name, &tempAttr->nameLength,
            error) < 0) {
        dpiObjectAttr__free(tempAttr, error);
        return DPI_FAILURE;
    }

    // determine type information of the attribute
    if (dpiOracleType__populateTypeInfo(objType->conn, param,
            DPI_OCI_DTYPE_PARAM, &tempAttr->typeInfo, error) < 0) {
        dpiObjectAttr__free(tempAttr, error);
        return DPI_FAILURE;
    }

    *attr = tempAttr;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiObjectAttr__free() [INTERNAL]
//   Free the memory for an object attribute.
//-----------------------------------------------------------------------------
void dpiObjectAttr__free(dpiObjectAttr *attr, dpiError *error)
{
    if (attr->belongsToType) {
        dpiGen__setRefCount(attr->belongsToType, error, -1);
        attr->belongsToType = NULL;
    }
    if (attr->typeInfo.objectType) {
        dpiGen__setRefCount(attr->typeInfo.objectType, error, -1);
        attr->typeInfo.objectType = NULL;
    }
    if (attr->name) {
        dpiUtils__freeMemory((void*) attr->name);
        attr->name = NULL;
    }
    dpiUtils__freeMemory(attr);
}


//-----------------------------------------------------------------------------
// dpiObjectAttr_addRef() [PUBLIC]
//   Add a reference to the object attribute.
//-----------------------------------------------------------------------------
int dpiObjectAttr_addRef(dpiObjectAttr *attr)
{
    return dpiGen__addRef(attr, DPI_HTYPE_OBJECT_ATTR, __func__);
}


//-----------------------------------------------------------------------------
// dpiObjectAttr_getInfo() [PUBLIC]
//   Return information about the attribute to the caller.
//-----------------------------------------------------------------------------
int dpiObjectAttr_getInfo(dpiObjectAttr *attr, dpiObjectAttrInfo *info)
{
    dpiError error;

    if (dpiGen__startPublicFn(attr, DPI_HTYPE_OBJECT_ATTR, __func__,
            &error) < 0)
        return dpiGen__endPublicFn(attr, DPI_FAILURE, &error);
    DPI_CHECK_PTR_NOT_NULL(attr, info)
    info->name = attr->name;
    info->nameLength = attr->nameLength;
    info->typeInfo = attr->typeInfo;
    return dpiGen__endPublicFn(attr, DPI_SUCCESS, &error);
}


//-----------------------------------------------------------------------------
// dpiObjectAttr_release() [PUBLIC]
//   Release a reference to the object attribute.
//-----------------------------------------------------------------------------
int dpiObjectAttr_release(dpiObjectAttr *attr)
{
    return dpiGen__release(attr, DPI_HTYPE_OBJECT_ATTR, __func__);
}
