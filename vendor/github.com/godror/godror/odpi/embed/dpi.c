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
// dpi.c
//   Include this file in your project in order to embed ODPI-C source without
// having to compile files individually. Only the definitions in the file
// include/dpi.h are intended to be used publicly. Each file can also be
// compiled independently if that is preferable.
//-----------------------------------------------------------------------------

#include "../src/dpiConn.c"
#include "../src/dpiContext.c"
#include "../src/dpiData.c"
#include "../src/dpiDebug.c"
#include "../src/dpiDeqOptions.c"
#include "../src/dpiEnqOptions.c"
#include "../src/dpiEnv.c"
#include "../src/dpiError.c"
#include "../src/dpiGen.c"
#include "../src/dpiGlobal.c"
#include "../src/dpiHandleList.c"
#include "../src/dpiHandlePool.c"
#include "../src/dpiLob.c"
#include "../src/dpiMsgProps.c"
#include "../src/dpiObjectAttr.c"
#include "../src/dpiObject.c"
#include "../src/dpiObjectType.c"
#include "../src/dpiOci.c"
#include "../src/dpiOracleType.c"
#include "../src/dpiPool.c"
#include "../src/dpiQueue.c"
#include "../src/dpiRowid.c"
#include "../src/dpiSodaColl.c"
#include "../src/dpiSodaCollCursor.c"
#include "../src/dpiSodaDb.c"
#include "../src/dpiSodaDoc.c"
#include "../src/dpiSodaDocCursor.c"
#include "../src/dpiStmt.c"
#include "../src/dpiSubscr.c"
#include "../src/dpiUtils.c"
#include "../src/dpiVar.c"
