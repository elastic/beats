//-----------------------------------------------------------------------------
// Copyright (c) 2017, 2019, Oracle and/or its affiliates. All rights reserved.
// This program is free software: you can modify it and/or redistribute it
// under the terms of:
//
// (i)  the Universal Permissive License v 1.0 or at your option, any
//      later version (http://oss.oracle.com/licenses/upl); and/or
//
// (ii) the Apache License v 2.0. (http://www.apache.org/licenses/LICENSE-2.0)
//-----------------------------------------------------------------------------

//-----------------------------------------------------------------------------
// dpiOci.c
//   Link to OCI using dynamic linking. The OCI library (11.2+) is loaded
// dynamically and a function table kept for the functions that are used by
// DPI. This function table is populated as functions are used and permits use
// of all versions of OCI from one driver.
//-----------------------------------------------------------------------------

#include "dpiImpl.h"

// forward declarations of internal functions only used in this file
static void *dpiOci__allocateMem(void *unused, size_t size);
static void dpiOci__freeMem(void *unused, void *ptr);
static int dpiOci__loadLib(dpiError *error);
static int dpiOci__loadLibValidate(dpiError *error);
static int dpiOci__loadSymbol(const char *symbolName, void **symbol,
        dpiError *error);
static void *dpiOci__reallocMem(void *unused, void *ptr, size_t newSize);


// macro to simplify code for loading each symbol
#define DPI_OCI_LOAD_SYMBOL(symbolName, symbol) \
    if (!symbol && dpiOci__loadSymbol(symbolName, (void**) &symbol, \
            error) < 0) \
        return DPI_FAILURE;


// typedefs for all OCI functions used by ODPI-C
typedef int (*dpiOciFnType__aqDeq)(void *svchp, void *errhp,
        const char *queue_name, void *deqopt, void *msgprop, void *payload_tdo,
        void **payload, void **payload_ind, void **msgid, uint32_t flags);
typedef int (*dpiOciFnType__aqEnq)(void *svchp, void *errhp,
        const char *queue_name, void *enqopt, void *msgprop, void *payload_tdo,
        void **payload, void **payload_ind, void **msgid, uint32_t flags);
typedef int (*dpiOciFnType__arrayDescriptorAlloc)(const void *parenth,
        void **descpp, const uint32_t type, uint32_t array_size,
        const size_t xtramem_sz, void **usrmempp);
typedef int (*dpiOciFnType__arrayDescriptorFree)(void **descp,
        const uint32_t type);
typedef int (*dpiOciFnType__attrGet)(const void  *trgthndlp,
        uint32_t trghndltyp, void *attributep, uint32_t *sizep,
        uint32_t attrtype, void *errhp);
typedef int (*dpiOciFnType__attrSet)(void *trgthndlp, uint32_t trghndltyp,
        void *attributep, uint32_t size, uint32_t attrtype, void *errhp);
typedef int (*dpiOciFnType__bindByName)(void *stmtp, void **bindp, void *errhp,
        const char *placeholder, int32_t placeh_len, void *valuep,
        int32_t value_sz, uint16_t dty, void *indp, uint16_t *alenp,
        uint16_t *rcodep, uint32_t maxarr_len, uint32_t *curelep,
        uint32_t mode);
typedef int (*dpiOciFnType__bindByName2)(void *stmtp, void **bindp,
        void *errhp, const char *placeholder, int32_t placeh_len, void *valuep,
        int64_t value_sz, uint16_t dty, void *indp, uint32_t *alenp,
        uint16_t *rcodep, uint32_t maxarr_len, uint32_t *curelep,
        uint32_t mode);
typedef int (*dpiOciFnType__bindByPos)(void *stmtp, void **bindp, void *errhp,
        uint32_t position, void *valuep, int32_t value_sz, uint16_t dty,
        void *indp, uint16_t *alenp, uint16_t *rcodep, uint32_t maxarr_len,
        uint32_t *curelep, uint32_t mode);
typedef int (*dpiOciFnType__bindByPos2)(void *stmtp, void **bindp, void *errhp,
        uint32_t position, void *valuep, int64_t value_sz, uint16_t dty,
        void *indp, uint32_t *alenp, uint16_t *rcodep, uint32_t maxarr_len,
        uint32_t *curelep, uint32_t mode);
typedef int (*dpiOciFnType__bindDynamic)(void *bindp, void *errhp, void *ictxp,
        void *icbfp, void *octxp, void *ocbfp);
typedef int (*dpiOciFnType__bindObject)(void *bindp, void *errhp,
        const void *type, void **pgvpp, uint32_t *pvszsp, void **indpp,
        uint32_t *indszp);
typedef int (*dpiOciFnType__break)(void *hndlp, void *errhp);
typedef void (*dpiOciFnType__clientVersion)(int *major_version,
        int *minor_version, int *update_num, int *patch_num,
        int *port_update_num);
typedef int (*dpiOciFnType__collAppend)(void *env, void *err, const void *elem,
        const void *elemind, void *coll);
typedef int (*dpiOciFnType__collAssignElem)(void *env, void *err,
        int32_t index, const void *elem, const void *elemind, void *coll);
typedef int (*dpiOciFnType__collGetElem)(void *env, void *err,
        const void *coll, int32_t index, int *exists, void **elem,
        void **elemind);
typedef int (*dpiOciFnType__collSize)(void *env, void *err, const void *coll,
        int32_t *size);
typedef int (*dpiOciFnType__collTrim)(void *env, void *err, int32_t trim_num,
        void *coll);
typedef int (*dpiOciFnType__contextGetValue)(void *hdl, void *err,
        const char *key, uint8_t keylen, void **ctx_value);
typedef int (*dpiOciFnType__contextSetValue)(void *hdl, void *err,
        uint16_t duration, const char *key, uint8_t keylen, void *ctx_value);
typedef int (*dpiOciFnType__dateTimeConstruct)(void *hndl, void *err,
        void *datetime, int16_t yr, uint8_t mnth, uint8_t dy, uint8_t hr,
        uint8_t mm, uint8_t ss, uint32_t fsec, const char *tz,
        size_t tzLength);
typedef int (*dpiOciFnType__dateTimeGetDate)(void *hndl, void *err,
        const void *date, int16_t *yr, uint8_t *mnth, uint8_t *dy);
typedef int (*dpiOciFnType__dateTimeGetTime)(void *hndl, void *err,
        void *datetime, uint8_t *hr, uint8_t *mm, uint8_t *ss, uint32_t *fsec);
typedef int (*dpiOciFnType__dateTimeGetTimeZoneOffset)(void *hndl, void *err,
        const void *datetime, int8_t *hr, int8_t *mm);
typedef int (*dpiOciFnType__dateTimeIntervalAdd)(void *hndl, void *err,
        void *datetime, void *inter, void *outdatetime);
typedef int (*dpiOciFnType__dateTimeSubtract)(void *hndl, void *err,
        void *indate1, void *indate2, void *inter);
typedef int (*dpiOciFnType__dbShutdown)(void *svchp, void *errhp, void *admhp,
        uint32_t mode);
typedef int (*dpiOciFnType__dbStartup)(void *svchp, void *errhp, void *admhp,
        uint32_t mode, uint32_t flags);
typedef int (*dpiOciFnType__defineByPos)(void *stmtp, void **defnp,
        void *errhp, uint32_t position, void *valuep, int32_t value_sz,
        uint16_t dty, void *indp, uint16_t *rlenp, uint16_t *rcodep,
        uint32_t mode);
typedef int (*dpiOciFnType__defineByPos2)(void *stmtp, void **defnp,
        void *errhp, uint32_t position, void *valuep, uint64_t value_sz,
        uint16_t dty, void *indp, uint32_t *rlenp, uint16_t *rcodep,
        uint32_t mode);
typedef int (*dpiOciFnType__defineDynamic)(void *defnp, void *errhp,
        void *octxp, void *ocbfp);
typedef int (*dpiOciFnType__defineObject)(void *defnp, void *errhp,
        const void *type, void **pgvpp, uint32_t *pvszsp, void **indpp,
        uint32_t *indszp);
typedef int (*dpiOciFnType__describeAny)(void *svchp, void *errhp,
        void *objptr, uint32_t objnm_len, uint8_t objptr_typ,
        uint8_t info_level, uint8_t objtyp, void *dschp);
typedef int (*dpiOciFnType__descriptorAlloc)(const void *parenth,
        void **descpp, const uint32_t type, const size_t xtramem_sz,
        void **usrmempp);
typedef int (*dpiOciFnType__descriptorFree)(void *descp, const uint32_t type);
typedef int (*dpiOciFnType__envNlsCreate)(void **envp, uint32_t mode,
        void *ctxp, void *malocfp, void *ralocfp, void *mfreefp,
        size_t xtramem_sz, void **usrmempp, uint16_t charset,
        uint16_t ncharset);
typedef int (*dpiOciFnType__errorGet)(void *hndlp, uint32_t recordno,
        char *sqlstate, int32_t *errcodep, char *bufp, uint32_t bufsiz,
        uint32_t type);
typedef int (*dpiOciFnType__handleAlloc)(const void *parenth, void **hndlpp,
        const uint32_t type, const size_t xtramem_sz, void **usrmempp);
typedef int (*dpiOciFnType__handleFree)(void *hndlp, const uint32_t type);
typedef int (*dpiOciFnType__intervalGetDaySecond)(void *hndl, void *err,
        int32_t *dy, int32_t *hr, int32_t *mm, int32_t *ss, int32_t *fsec,
        const void *result);
typedef int (*dpiOciFnType__intervalGetYearMonth)(void *hndl, void *err,
        int32_t *yr, int32_t *mnth, const void *result);
typedef int (*dpiOciFnType__intervalSetDaySecond)(void *hndl, void *err,
        int32_t dy, int32_t hr, int32_t mm, int32_t ss, int32_t fsec,
        void *result);
typedef int (*dpiOciFnType__intervalSetYearMonth)(void *hndl, void *err,
        int32_t yr, int32_t mnth, void *result);
typedef int (*dpiOciFnType__lobClose)(void *svchp, void *errhp, void *locp);
typedef int (*dpiOciFnType__lobCreateTemporary)(void *svchp, void *errhp,
        void *locp, uint16_t csid, uint8_t csfrm, uint8_t lobtype, int cache,
        uint16_t duration);
typedef int (*dpiOciFnType__lobFileExists)(void *svchp, void *errhp,
        void *filep, int *flag);
typedef int (*dpiOciFnType__lobFileGetName)(void *envhp, void *errhp,
        const void *filep, char *dir_alias, uint16_t *d_length, char *filename,
        uint16_t *f_length);
typedef int (*dpiOciFnType__lobFileSetName)(void *envhp, void *errhp,
        void **filepp, const char *dir_alias, uint16_t d_length,
        const char *filename, uint16_t f_length);
typedef int (*dpiOciFnType__lobFreeTemporary)(void *svchp, void *errhp,
        void *locp);
typedef int (*dpiOciFnType__lobGetChunkSize)(void *svchp, void *errhp,
        void *locp, uint32_t *chunksizep);
typedef int (*dpiOciFnType__lobGetLength2)(void *svchp, void *errhp,
        void *locp, uint64_t *lenp);
typedef int (*dpiOciFnType__lobIsOpen)(void *svchp, void *errhp, void *locp,
        int *flag);
typedef int (*dpiOciFnType__lobIsTemporary)(void *envp, void *errhp,
        void *locp, int *is_temporary);
typedef int (*dpiOciFnType__lobLocatorAssign)(void *svchp, void *errhp,
        const void *src_locp, void **dst_locpp);
typedef int (*dpiOciFnType__lobOpen)(void *svchp, void *errhp, void *locp,
        uint8_t mode);
typedef int (*dpiOciFnType__lobRead2)(void *svchp, void *errhp, void *locp,
        uint64_t *byte_amtp, uint64_t *char_amtp, uint64_t offset, void *bufp,
        uint64_t bufl, uint8_t piece, void *ctxp, void *cbfp, uint16_t csid,
        uint8_t csfrm);
typedef int (*dpiOciFnType__lobTrim2)(void *svchp, void *errhp, void *locp,
        uint64_t newlen);
typedef int (*dpiOciFnType__lobWrite2)(void *svchp, void *errhp, void *locp,
        uint64_t *byte_amtp, uint64_t *char_amtp, uint64_t offset, void *bufp,
        uint64_t buflen, uint8_t piece, void *ctxp, void *cbfp, uint16_t csid,
        uint8_t csfrm);
typedef int (*dpiOciFnType__memoryAlloc)(void *hdl, void *err, void **mem,
        uint16_t dur, uint32_t size, uint32_t flags);
typedef int (*dpiOciFnType__memoryFree)(void *hdl, void *err, void *mem);
typedef int (*dpiOciFnType__nlsCharSetConvert)(void *envhp, void *errhp,
        uint16_t dstid, void  *dstp, size_t dstlen, uint16_t srcid,
        const void *srcp, size_t srclen, size_t *rsize);
typedef int (*dpiOciFnType__nlsCharSetIdToName)(void *envhp, char *buf,
        size_t buflen, uint16_t id);
typedef uint16_t (*dpiOciFnType__nlsCharSetNameToId)(void *envhp,
        const char *name);
typedef int (*dpiOciFnType__nlsEnvironmentVariableGet)(void *val, size_t size,
        uint16_t item, uint16_t charset, size_t *rsize);
typedef int (*dpiOciFnType__nlsNameMap)(void *envhp, char *buf, size_t buflen,
        const char *srcbuf, uint32_t flag);
typedef int (*dpiOciFnType__nlsNumericInfoGet)(void *envhp, void *errhp,
        int32_t *val, uint16_t item);
typedef int (*dpiOciFnType__numberFromInt)(void *err, const void *inum,
        unsigned int inum_length, unsigned int inum_s_flag, void *number);
typedef int (*dpiOciFnType__numberFromReal)(void *err, const void *number,
        unsigned int rsl_length, void *rsl);
typedef int (*dpiOciFnType__numberToInt)(void *err, const void *number,
        unsigned int rsl_length, unsigned int rsl_flag, void *rsl);
typedef int (*dpiOciFnType__numberToReal)(void *err, const void *number,
        unsigned int rsl_length, void *rsl);
typedef int (*dpiOciFnType__objectCopy)(void *env, void *err, const void *svc,
        void *source, void *null_source, void *target, void *null_target,
        void *tdo, uint16_t duration, uint8_t option);
typedef int (*dpiOciFnType__objectFree)(void *env, void *err, void *instance,
        uint16_t flags);
typedef int (*dpiOciFnType__objectGetAttr)(void *env, void *err,
        void *instance, void *null_struct, void *tdo, const char **names,
        const uint32_t *lengths, const uint32_t name_count,
        const uint32_t *indexes, const uint32_t index_count,
        int16_t *attr_null_status, void **attr_null_struct, void **attr_value,
        void **attr_tdo);
typedef int (*dpiOciFnType__objectGetInd)(void *env, void *err, void *instance,
        void **null_struct);
typedef int (*dpiOciFnType__objectNew)(void *env, void *err, const void *svc,
        uint16_t typecode, void *tdo, void *table, uint16_t duration,
        int value, void **instance);
typedef int (*dpiOciFnType__objectPin)(void *env, void *err, void *object_ref,
        void *corhdl, int pin_option, uint16_t pin_duration, int lock_option,
        void **object);
typedef int (*dpiOciFnType__objectSetAttr)(void *env, void *err,
        void *instance, void *null_struct, void *tdo, const char **names,
        const uint32_t *lengths, const uint32_t name_count,
        const uint32_t *indexes, const uint32_t index_count,
        const int16_t null_status, const void *attr_null_struct,
        const void *attr_value);
typedef int (*dpiOciFnType__paramGet)(const void *hndlp, uint32_t htype,
        void *errhp, void **parmdpp, uint32_t pos);
typedef int (*dpiOciFnType__passwordChange)(void *svchp, void *errhp,
        const char *user_name, uint32_t usernm_len, const char *opasswd,
        uint32_t opasswd_len, const char *npasswd, uint32_t npasswd_len,
        uint32_t mode);
typedef int (*dpiOciFnType__ping)(void *svchp, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__rawAssignBytes)(void *env, void *err,
        const char *rhs, uint32_t rhs_len, void **lhs);
typedef void *(*dpiOciFnType__rawPtr)(void *env, const void *raw);
typedef int (*dpiOciFnType__rawResize)(void *env, void *err, uint32_t new_size,
        void **raw);
typedef uint32_t (*dpiOciFnType__rawSize)(void * env, const void *raw);
typedef int (*dpiOciFnType__rowidToChar)(void *rowidDesc, char *outbfp,
        uint16_t *outbflp, void *errhp);
typedef int (*dpiOciFnType__serverAttach)(void *srvhp, void *errhp,
        const char *dblink, int32_t dblink_len, uint32_t mode);
typedef int (*dpiOciFnType__serverDetach)(void *srvhp, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__serverRelease)(void *hndlp, void *errhp,
        char *bufp, uint32_t bufsz, uint8_t hndltype, uint32_t *version);
typedef int (*dpiOciFnType__serverRelease2)(void *hndlp, void *errhp,
        char *bufp, uint32_t bufsz, uint8_t hndltype, uint32_t *version,
        uint32_t mode);
typedef int (*dpiOciFnType__sessionBegin)(void *svchp, void *errhp,
        void *usrhp, uint32_t credt, uint32_t mode);
typedef int (*dpiOciFnType__sessionEnd)(void *svchp, void *errhp, void *usrhp,
        uint32_t mode);
typedef int (*dpiOciFnType__sessionGet)(void *envhp, void *errhp, void **svchp,
        void *authhp, const char *poolName, uint32_t poolName_len,
        const char *tagInfo, uint32_t tagInfo_len, const char **retTagInfo,
        uint32_t *retTagInfo_len, int *found, uint32_t mode);
typedef int (*dpiOciFnType__sessionPoolCreate)(void *envhp, void *errhp,
        void *spoolhp, char **poolName, uint32_t *poolNameLen,
        const char *connStr, uint32_t connStrLen, uint32_t sessMin,
        uint32_t sessMax, uint32_t sessIncr, const char *userid,
        uint32_t useridLen, const char *password, uint32_t passwordLen,
        uint32_t mode);
typedef int (*dpiOciFnType__sessionPoolDestroy)(void *spoolhp, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__sessionRelease)(void *svchp, void *errhp,
        const char *tag, uint32_t tag_len, uint32_t mode);
typedef int (*dpiOciFnType__shardingKeyColumnAdd)(void *shardingKey,
        void *errhp, void *col, uint32_t colLen, uint16_t colType,
        uint32_t mode);
typedef int (*dpiOciFnType__sodaCollCreateWithMetadata)(void *svchp,
        const char *collname, uint32_t collnamelen, const char *metadata,
        uint32_t metadatalen, void **collection, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaCollDrop)(void *svchp, void *coll,
        int *isDropped, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaCollGetNext)(void *svchp, const void *cur,
        void **coll, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaCollList)(void *svchp, const char *startname,
        uint32_t stnamelen, void **cur, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaCollOpen)(void *svchp, const char *collname,
        uint32_t collnamelen, void **coll, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaDataGuideGet)(void *svchp,
        const void *collection, uint32_t docFlags, void **doc, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__sodaDocCount)(void *svchp, const void *coll,
        const void *optns, uint64_t *numdocs, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaDocGetNext)(void *svchp, const void *cur,
        void **doc, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaFind)(void *svchp, const void *coll,
        const void *findOptions, uint32_t docFlags, void **cursor,
        void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaFindOne)(void *svchp, const void *coll,
        const void *findOptions, uint32_t docFlags, void **doc, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__sodaIndexCreate)(void *svchp, const void *coll,
        const char *indexspec, uint32_t speclen, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaIndexDrop)(void *svchp, const char *indexname,
        uint32_t indexnamelen, int *isDropped, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaInsert)(void *svchp, void *collection,
        void *document, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaInsertAndGet)(void *svchp, void *collection,
        void **document, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaOperKeysSet)(const void *operhp,
        const char **keysArray, uint32_t *lengthsArray, uint32_t count,
        void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaRemove)(void *svchp, const void *coll,
        const void *optns, uint64_t *removeCount, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__sodaReplOne)(void *svchp, const void *coll,
        const void *optns, void *document, int *isReplaced, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__sodaReplOneAndGet)(void *svchp, const void *coll,
        const void *optns, void **document, int *isReplaced, void *errhp,
        uint32_t mode);
typedef int (*dpiOciFnType__stmtExecute)(void *svchp, void *stmtp, void *errhp,
        uint32_t iters, uint32_t rowoff, const void *snap_in, void *snap_out,
        uint32_t mode);
typedef int (*dpiOciFnType__stmtFetch2)(void *stmtp, void *errhp,
        uint32_t nrows, uint16_t orientation, int32_t scrollOffset,
        uint32_t mode);
typedef int (*dpiOciFnType__stmtGetBindInfo)(void *stmtp, void *errhp,
        uint32_t size, uint32_t startloc, int32_t *found, char *bvnp[],
        uint8_t bvnl[], char *invp[], uint8_t inpl[], uint8_t dupl[],
        void **hndl);
typedef int (*dpiOciFnType__stmtGetNextResult)(void *stmthp, void *errhp,
        void **result, uint32_t *rtype, uint32_t mode);
typedef int (*dpiOciFnType__stmtPrepare2)(void *svchp, void **stmtp,
        void *errhp, const char *stmt, uint32_t stmt_len, const char *key,
        uint32_t key_len, uint32_t language, uint32_t mode);
typedef int (*dpiOciFnType__stmtRelease)(void *stmtp, void *errhp,
        const char *key, uint32_t key_len, uint32_t mode);
typedef int (*dpiOciFnType__stringAssignText)(void *env, void *err,
        const char *rhs, uint32_t rhs_len, void **lhs);
typedef char *(*dpiOciFnType__stringPtr)(void *env, const void *vs);
typedef int (*dpiOciFnType__stringResize)(void *env, void *err,
        uint32_t new_size, void **str);
typedef uint32_t (*dpiOciFnType__stringSize)(void *env, const void *vs);
typedef int (*dpiOciFnType__subscriptionRegister)(void *svchp,
        void **subscrhpp, uint16_t count, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__subscriptionUnRegister)(void *svchp,
        void *subscrhp, void *errhp, uint32_t mode);
typedef int (*dpiOciFnType__tableDelete)(void *env, void *err, int32_t index,
        void *tbl);
typedef int (*dpiOciFnType__tableExists)(void *env, void *err, const void *tbl,
        int32_t index, int *exists);
typedef int (*dpiOciFnType__tableFirst)(void *env, void *err, const void *tbl,
        int32_t *index);
typedef int (*dpiOciFnType__tableLast)(void *env, void *err, const void *tbl,
        int32_t *index);
typedef int (*dpiOciFnType__tableNext)(void *env, void *err, int32_t index,
        const void *tbl, int32_t *next_index, int *exists);
typedef int (*dpiOciFnType__tablePrev)(void *env, void *err, int32_t index,
        const void *tbl, int32_t *prev_index, int *exists);
typedef int (*dpiOciFnType__tableSize)(void *env, void *err, const void *tbl,
        int32_t *size);
typedef int (*dpiOciFnType__threadKeyDestroy)(void *hndl, void *err,
        void **key);
typedef int (*dpiOciFnType__threadKeyGet)(void *hndl, void *err, void *key,
        void **pValue);
typedef int (*dpiOciFnType__threadKeyInit)(void *hndl, void *err, void **key,
        void *destFn);
typedef int (*dpiOciFnType__threadKeySet)(void *hndl, void *err, void *key,
        void *value);
typedef void (*dpiOciFnType__threadProcessInit)(void);
typedef int (*dpiOciFnType__transCommit)(void *svchp, void *errhp,
        uint32_t flags);
typedef int (*dpiOciFnType__transPrepare)(void *svchp, void *errhp,
        uint32_t flags);
typedef int (*dpiOciFnType__transRollback)(void *svchp, void *errhp,
        uint32_t flags);
typedef int (*dpiOciFnType__transStart)(void *svchp, void *errhp,
        unsigned int timeout, uint32_t flags);
typedef int (*dpiOciFnType__typeByFullName)(void *env, void *err,
        const void *svc, const char *full_type_name,
        uint32_t full_type_name_length, const char *version_name,
        uint32_t version_name_length, uint16_t pin_duration, int get_option,
        void **tdo);


// library handle for dynamically loaded OCI library
static void *dpiOciLibHandle = NULL;

// library names to search
static const char *dpiOciLibNames[] = {
#if defined _WIN32 || defined __CYGWIN__
    "oci.dll",
#elif __APPLE__
    "libclntsh.dylib",
    "libclntsh.dylib.18.1",
    "libclntsh.dylib.12.1",
    "libclntsh.dylib.11.1",
    "libclntsh.dylib.19.1",
#else
    "libclntsh.so",
    "libclntsh.so.18.1",
    "libclntsh.so.12.1",
    "libclntsh.so.11.1",
    "libclntsh.so.19.1",
#endif
    NULL
};

// URL fragment to use in load library exception
#if defined _WIN32 || defined __CYGWIN__
    #define DPI_ERR_LOAD_URL_FRAGMENT   "windows"
#elif __APPLE__
    #define DPI_ERR_LOAD_URL_FRAGMENT   "macos"
#else
    #define DPI_ERR_LOAD_URL_FRAGMENT   "linux"
#endif

// version information for loaded OCI library
static dpiVersionInfo dpiOciLibVersionInfo;

// all OCI symbols used by ODPI-C
static struct {
    dpiOciFnType__aqDeq fnAqDeq;
    dpiOciFnType__aqEnq fnAqEnq;
    dpiOciFnType__arrayDescriptorAlloc fnArrayDescriptorAlloc;
    dpiOciFnType__arrayDescriptorFree fnArrayDescriptorFree;
    dpiOciFnType__attrGet fnAttrGet;
    dpiOciFnType__attrSet fnAttrSet;
    dpiOciFnType__bindByName fnBindByName;
    dpiOciFnType__bindByName2 fnBindByName2;
    dpiOciFnType__bindByPos fnBindByPos;
    dpiOciFnType__bindByPos2 fnBindByPos2;
    dpiOciFnType__bindDynamic fnBindDynamic;
    dpiOciFnType__bindObject fnBindObject;
    dpiOciFnType__break fnBreak;
    dpiOciFnType__clientVersion fnClientVersion;
    dpiOciFnType__collAppend fnCollAppend;
    dpiOciFnType__collAssignElem fnCollAssignElem;
    dpiOciFnType__collGetElem fnCollGetElem;
    dpiOciFnType__collSize fnCollSize;
    dpiOciFnType__collTrim fnCollTrim;
    dpiOciFnType__contextGetValue fnContextGetValue;
    dpiOciFnType__contextSetValue fnContextSetValue;
    dpiOciFnType__dateTimeConstruct fnDateTimeConstruct;
    dpiOciFnType__dateTimeGetDate fnDateTimeGetDate;
    dpiOciFnType__dateTimeGetTime fnDateTimeGetTime;
    dpiOciFnType__dateTimeGetTimeZoneOffset fnDateTimeGetTimeZoneOffset;
    dpiOciFnType__dateTimeIntervalAdd fnDateTimeIntervalAdd;
    dpiOciFnType__dateTimeSubtract fnDateTimeSubtract;
    dpiOciFnType__dbShutdown fnDbShutdown;
    dpiOciFnType__dbStartup fnDbStartup;
    dpiOciFnType__defineByPos fnDefineByPos;
    dpiOciFnType__defineByPos2 fnDefineByPos2;
    dpiOciFnType__defineDynamic fnDefineDynamic;
    dpiOciFnType__defineObject fnDefineObject;
    dpiOciFnType__describeAny fnDescribeAny;
    dpiOciFnType__descriptorAlloc fnDescriptorAlloc;
    dpiOciFnType__descriptorFree fnDescriptorFree;
    dpiOciFnType__envNlsCreate fnEnvNlsCreate;
    dpiOciFnType__errorGet fnErrorGet;
    dpiOciFnType__handleAlloc fnHandleAlloc;
    dpiOciFnType__handleFree fnHandleFree;
    dpiOciFnType__intervalGetDaySecond fnIntervalGetDaySecond;
    dpiOciFnType__intervalGetYearMonth fnIntervalGetYearMonth;
    dpiOciFnType__intervalSetDaySecond fnIntervalSetDaySecond;
    dpiOciFnType__intervalSetYearMonth fnIntervalSetYearMonth;
    dpiOciFnType__lobClose fnLobClose;
    dpiOciFnType__lobCreateTemporary fnLobCreateTemporary;
    dpiOciFnType__lobFileExists fnLobFileExists;
    dpiOciFnType__lobFileGetName fnLobFileGetName;
    dpiOciFnType__lobFileSetName fnLobFileSetName;
    dpiOciFnType__lobFreeTemporary fnLobFreeTemporary;
    dpiOciFnType__lobGetChunkSize fnLobGetChunkSize;
    dpiOciFnType__lobGetLength2 fnLobGetLength2;
    dpiOciFnType__lobIsOpen fnLobIsOpen;
    dpiOciFnType__lobIsTemporary fnLobIsTemporary;
    dpiOciFnType__lobLocatorAssign fnLobLocatorAssign;
    dpiOciFnType__lobOpen fnLobOpen;
    dpiOciFnType__lobRead2 fnLobRead2;
    dpiOciFnType__lobTrim2 fnLobTrim2;
    dpiOciFnType__lobWrite2 fnLobWrite2;
    dpiOciFnType__memoryAlloc fnMemoryAlloc;
    dpiOciFnType__memoryFree fnMemoryFree;
    dpiOciFnType__nlsCharSetConvert fnNlsCharSetConvert;
    dpiOciFnType__nlsCharSetIdToName fnNlsCharSetIdToName;
    dpiOciFnType__nlsCharSetNameToId fnNlsCharSetNameToId;
    dpiOciFnType__nlsEnvironmentVariableGet fnNlsEnvironmentVariableGet;
    dpiOciFnType__nlsNameMap fnNlsNameMap;
    dpiOciFnType__nlsNumericInfoGet fnNlsNumericInfoGet;
    dpiOciFnType__numberFromInt fnNumberFromInt;
    dpiOciFnType__numberFromReal fnNumberFromReal;
    dpiOciFnType__numberToInt fnNumberToInt;
    dpiOciFnType__numberToReal fnNumberToReal;
    dpiOciFnType__objectCopy fnObjectCopy;
    dpiOciFnType__objectFree fnObjectFree;
    dpiOciFnType__objectGetAttr fnObjectGetAttr;
    dpiOciFnType__objectGetInd fnObjectGetInd;
    dpiOciFnType__objectNew fnObjectNew;
    dpiOciFnType__objectPin fnObjectPin;
    dpiOciFnType__objectSetAttr fnObjectSetAttr;
    dpiOciFnType__paramGet fnParamGet;
    dpiOciFnType__passwordChange fnPasswordChange;
    dpiOciFnType__ping fnPing;
    dpiOciFnType__rawAssignBytes fnRawAssignBytes;
    dpiOciFnType__rawPtr fnRawPtr;
    dpiOciFnType__rawResize fnRawResize;
    dpiOciFnType__rawSize fnRawSize;
    dpiOciFnType__rowidToChar fnRowidToChar;
    dpiOciFnType__serverAttach fnServerAttach;
    dpiOciFnType__serverDetach fnServerDetach;
    dpiOciFnType__serverRelease fnServerRelease;
    dpiOciFnType__serverRelease2 fnServerRelease2;
    dpiOciFnType__sessionBegin fnSessionBegin;
    dpiOciFnType__sessionEnd fnSessionEnd;
    dpiOciFnType__sessionGet fnSessionGet;
    dpiOciFnType__sessionPoolCreate fnSessionPoolCreate;
    dpiOciFnType__sessionPoolDestroy fnSessionPoolDestroy;
    dpiOciFnType__sessionRelease fnSessionRelease;
    dpiOciFnType__shardingKeyColumnAdd fnShardingKeyColumnAdd;
    dpiOciFnType__stmtExecute fnStmtExecute;
    dpiOciFnType__sodaCollCreateWithMetadata fnSodaCollCreateWithMetadata;
    dpiOciFnType__sodaCollDrop fnSodaCollDrop;
    dpiOciFnType__sodaCollGetNext fnSodaCollGetNext;
    dpiOciFnType__sodaCollList fnSodaCollList;
    dpiOciFnType__sodaCollOpen fnSodaCollOpen;
    dpiOciFnType__sodaDataGuideGet fnSodaDataGuideGet;
    dpiOciFnType__sodaDocCount fnSodaDocCount;
    dpiOciFnType__sodaDocGetNext fnSodaDocGetNext;
    dpiOciFnType__sodaFind fnSodaFind;
    dpiOciFnType__sodaFindOne fnSodaFindOne;
    dpiOciFnType__sodaIndexCreate fnSodaIndexCreate;
    dpiOciFnType__sodaIndexDrop fnSodaIndexDrop;
    dpiOciFnType__sodaInsert fnSodaInsert;
    dpiOciFnType__sodaInsertAndGet fnSodaInsertAndGet;
    dpiOciFnType__sodaOperKeysSet fnSodaOperKeysSet;
    dpiOciFnType__sodaRemove fnSodaRemove;
    dpiOciFnType__sodaReplOne fnSodaReplOne;
    dpiOciFnType__sodaReplOneAndGet fnSodaReplOneAndGet;
    dpiOciFnType__stmtFetch2 fnStmtFetch2;
    dpiOciFnType__stmtGetBindInfo fnStmtGetBindInfo;
    dpiOciFnType__stmtGetNextResult fnStmtGetNextResult;
    dpiOciFnType__stmtPrepare2 fnStmtPrepare2;
    dpiOciFnType__stmtRelease fnStmtRelease;
    dpiOciFnType__stringAssignText fnStringAssignText;
    dpiOciFnType__stringPtr fnStringPtr;
    dpiOciFnType__stringResize fnStringResize;
    dpiOciFnType__stringSize fnStringSize;
    dpiOciFnType__subscriptionRegister fnSubscriptionRegister;
    dpiOciFnType__subscriptionUnRegister fnSubscriptionUnRegister;
    dpiOciFnType__tableDelete fnTableDelete;
    dpiOciFnType__tableExists fnTableExists;
    dpiOciFnType__tableFirst fnTableFirst;
    dpiOciFnType__tableLast fnTableLast;
    dpiOciFnType__tableNext fnTableNext;
    dpiOciFnType__tablePrev fnTablePrev;
    dpiOciFnType__tableSize fnTableSize;
    dpiOciFnType__threadKeyDestroy fnThreadKeyDestroy;
    dpiOciFnType__threadKeyGet fnThreadKeyGet;
    dpiOciFnType__threadKeyInit fnThreadKeyInit;
    dpiOciFnType__threadKeySet fnThreadKeySet;
    dpiOciFnType__threadProcessInit fnThreadProcessInit;
    dpiOciFnType__transCommit fnTransCommit;
    dpiOciFnType__transPrepare fnTransPrepare;
    dpiOciFnType__transRollback fnTransRollback;
    dpiOciFnType__transStart fnTransStart;
    dpiOciFnType__typeByFullName fnTypeByFullName;
} dpiOciSymbols;


//-----------------------------------------------------------------------------
// dpiOci__allocateMem() [INTERNAL]
//   Wrapper for OCI allocation of memory, only used when debugging memory
// allocation.
//-----------------------------------------------------------------------------
static void *dpiOci__allocateMem(void *unused, size_t size)
{
    void *ptr;

    ptr = malloc(size);
    dpiDebug__print("OCI allocated %u bytes at %p\n", size, ptr);
    return ptr;
}


//-----------------------------------------------------------------------------
// dpiOci__aqDeq() [INTERNAL]
//   Wrapper for OCIAQDeq().
//-----------------------------------------------------------------------------
int dpiOci__aqDeq(dpiConn *conn, const char *queueName, void *options,
        void *msgProps, void *payloadType, void **payload, void **payloadInd,
        void **msgId, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIAQDeq", dpiOciSymbols.fnAqDeq)
    status = (*dpiOciSymbols.fnAqDeq)(conn->handle, error->handle, queueName,
            options, msgProps, payloadType, payload, payloadInd, msgId,
            DPI_OCI_DEFAULT);
    return dpiError__check(error, status, conn, "dequeue message");
}


//-----------------------------------------------------------------------------
// dpiOci__aqEnq() [INTERNAL]
//   Wrapper for OCIAQEnq().
//-----------------------------------------------------------------------------
int dpiOci__aqEnq(dpiConn *conn, const char *queueName, void *options,
        void *msgProps, void *payloadType, void **payload, void **payloadInd,
        void **msgId, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIAQEnq", dpiOciSymbols.fnAqEnq)
    status = (*dpiOciSymbols.fnAqEnq)(conn->handle, error->handle, queueName,
            options, msgProps, payloadType, payload, payloadInd, msgId,
            DPI_OCI_DEFAULT);
    return dpiError__check(error, status, conn, "enqueue message");
}


//-----------------------------------------------------------------------------
// dpiOci__arrayDescriptorAlloc() [INTERNAL]
//   Wrapper for OCIArrayDescriptorAlloc().
//-----------------------------------------------------------------------------
int dpiOci__arrayDescriptorAlloc(void *envHandle, void **handle,
        uint32_t handleType, uint32_t arraySize, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIArrayDescriptorAlloc",
            dpiOciSymbols.fnArrayDescriptorAlloc)
    status = (*dpiOciSymbols.fnArrayDescriptorAlloc)(envHandle, handle,
            handleType, arraySize, 0, NULL);
    return dpiError__check(error, status, NULL, "allocate descriptors");
}


//-----------------------------------------------------------------------------
// dpiOci__arrayDescriptorFree() [INTERNAL]
//   Wrapper for OCIArrayDescriptorFree().
//-----------------------------------------------------------------------------
int dpiOci__arrayDescriptorFree(void **handle, uint32_t handleType)
{
    dpiError *error = NULL;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIArrayDescriptorFree",
            dpiOciSymbols.fnArrayDescriptorFree)
    status = (*dpiOciSymbols.fnArrayDescriptorFree)(handle, handleType);
    if (status != DPI_OCI_SUCCESS && dpiDebugLevel & DPI_DEBUG_LEVEL_FREES)
        dpiDebug__print("free array descriptors %p, handleType %d failed\n",
                handle, handleType);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__attrGet() [INTERNAL]
//   Wrapper for OCIAttrGet().
//-----------------------------------------------------------------------------
int dpiOci__attrGet(const void *handle, uint32_t handleType, void *ptr,
        uint32_t *size, uint32_t attribute, const char *action,
        dpiError *error)
{
    int status;

    status = (*dpiOciSymbols.fnAttrGet)(handle, handleType, ptr, size,
            attribute, error->handle);
    if (action)
        return dpiError__check(error, status, NULL, action);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__attrSet() [INTERNAL]
//   Wrapper for OCIAttrSet().
//-----------------------------------------------------------------------------
int dpiOci__attrSet(void *handle, uint32_t handleType, void *ptr,
        uint32_t size, uint32_t attribute, const char *action, dpiError *error)
{
    int status;

    status = (*dpiOciSymbols.fnAttrSet)(handle, handleType, ptr, size,
            attribute, error->handle);
    if (action)
        return dpiError__check(error, status, NULL, action);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__bindByName() [INTERNAL]
//   Wrapper for OCIBindByName().
//-----------------------------------------------------------------------------
int dpiOci__bindByName(dpiStmt *stmt, void **bindHandle, const char *name,
        int32_t nameLength, int dynamicBind, dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindByName", dpiOciSymbols.fnBindByName)
    status = (*dpiOciSymbols.fnBindByName)(stmt->handle, bindHandle,
            error->handle, name, nameLength,
            (dynamicBind) ? NULL : var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : (int32_t) var->sizeInBytes,
            var->type->oracleType, (dynamicBind) ? NULL :
                    var->buffer.indicator,
            (dynamicBind || var->type->sizeInBytes) ? NULL :
                    var->buffer.actualLength16,
            (dynamicBind) ? NULL : var->buffer.returnCode,
            (var->isArray) ? var->buffer.maxArraySize : 0,
            (var->isArray) ? &var->buffer.actualArraySize : NULL,
            (dynamicBind) ? DPI_OCI_DATA_AT_EXEC : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "bind by name");
}


//-----------------------------------------------------------------------------
// dpiOci__bindByName2() [INTERNAL]
//   Wrapper for OCIBindByName2().
//-----------------------------------------------------------------------------
int dpiOci__bindByName2(dpiStmt *stmt, void **bindHandle, const char *name,
        int32_t nameLength, int dynamicBind, dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindByName2", dpiOciSymbols.fnBindByName2)
    status = (*dpiOciSymbols.fnBindByName2)(stmt->handle, bindHandle,
            error->handle, name, nameLength,
            (dynamicBind) ? NULL : var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : var->sizeInBytes,
            var->type->oracleType, (dynamicBind) ? NULL :
                    var->buffer.indicator,
            (dynamicBind || var->type->sizeInBytes) ? NULL :
                    var->buffer.actualLength32,
            (dynamicBind) ? NULL : var->buffer.returnCode,
            (var->isArray) ? var->buffer.maxArraySize : 0,
            (var->isArray) ? &var->buffer.actualArraySize : NULL,
            (dynamicBind) ? DPI_OCI_DATA_AT_EXEC : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "bind by name");
}


//-----------------------------------------------------------------------------
// dpiOci__bindByPos() [INTERNAL]
//   Wrapper for OCIBindByPos().
//-----------------------------------------------------------------------------
int dpiOci__bindByPos(dpiStmt *stmt, void **bindHandle, uint32_t pos,
        int dynamicBind, dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindByPos", dpiOciSymbols.fnBindByPos)
    status = (*dpiOciSymbols.fnBindByPos)(stmt->handle, bindHandle,
            error->handle, pos, (dynamicBind) ? NULL : var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : (int32_t) var->sizeInBytes,
            var->type->oracleType, (dynamicBind) ? NULL :
                    var->buffer.indicator,
            (dynamicBind || var->type->sizeInBytes) ? NULL :
                    var->buffer.actualLength16,
            (dynamicBind) ? NULL : var->buffer.returnCode,
            (var->isArray) ? var->buffer.maxArraySize : 0,
            (var->isArray) ? &var->buffer.actualArraySize : NULL,
            (dynamicBind) ? DPI_OCI_DATA_AT_EXEC : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "bind by position");
}


//-----------------------------------------------------------------------------
// dpiOci__bindByPos2() [INTERNAL]
//   Wrapper for OCIBindByPos2().
//-----------------------------------------------------------------------------
int dpiOci__bindByPos2(dpiStmt *stmt, void **bindHandle, uint32_t pos,
        int dynamicBind, dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindByPos2", dpiOciSymbols.fnBindByPos2)
    status = (*dpiOciSymbols.fnBindByPos2)(stmt->handle, bindHandle,
            error->handle, pos, (dynamicBind) ? NULL : var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : var->sizeInBytes,
            var->type->oracleType, (dynamicBind) ? NULL :
                    var->buffer.indicator,
            (dynamicBind || var->type->sizeInBytes) ? NULL :
                    var->buffer.actualLength32,
            (dynamicBind) ? NULL : var->buffer.returnCode,
            (var->isArray) ? var->buffer.maxArraySize : 0,
            (var->isArray) ? &var->buffer.actualArraySize : NULL,
            (dynamicBind) ? DPI_OCI_DATA_AT_EXEC : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "bind by position");
}


//-----------------------------------------------------------------------------
// dpiOci__bindDynamic() [INTERNAL]
//   Wrapper for OCIBindDynamic().
//-----------------------------------------------------------------------------
int dpiOci__bindDynamic(dpiVar *var, void *bindHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindDynamic", dpiOciSymbols.fnBindDynamic)
    status = (*dpiOciSymbols.fnBindDynamic)(bindHandle, error->handle, var,
            (void*) dpiVar__inBindCallback, var,
            (void*) dpiVar__outBindCallback);
    return dpiError__check(error, status, var->conn, "bind dynamic");
}


//-----------------------------------------------------------------------------
// dpiOci__bindObject() [INTERNAL]
//   Wrapper for OCIBindObject().
//-----------------------------------------------------------------------------
int dpiOci__bindObject(dpiVar *var, void *bindHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBindObject", dpiOciSymbols.fnBindObject)
    status = (*dpiOciSymbols.fnBindObject)(bindHandle, error->handle,
            var->objectType->tdo, (void**) var->buffer.data.asRaw, 0,
            var->buffer.objectIndicator, 0);
    return dpiError__check(error, status, var->conn, "bind object");
}


//-----------------------------------------------------------------------------
// dpiOci__break() [INTERNAL]
//   Wrapper for OCIBreak().
//-----------------------------------------------------------------------------
int dpiOci__break(dpiConn *conn, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIBreak", dpiOciSymbols.fnBreak)
    status = (*dpiOciSymbols.fnBreak)(conn->handle, error->handle);
    return dpiError__check(error, status, conn, "break execution");
}


//-----------------------------------------------------------------------------
// dpiOci__clientVersion() [INTERNAL]
//   Set the version information in the context to the OCI client version
// information that was discovered when the OCI library was loaded.
//-----------------------------------------------------------------------------
void dpiOci__clientVersion(dpiContext *context)
{
    context->versionInfo = &dpiOciLibVersionInfo;
}


//-----------------------------------------------------------------------------
// dpiOci__collAppend() [INTERNAL]
//   Wrapper for OCICollAppend().
//-----------------------------------------------------------------------------
int dpiOci__collAppend(dpiConn *conn, const void *elem, const void *elemInd,
        void *coll, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCICollAppend", dpiOciSymbols.fnCollAppend)
    status = (*dpiOciSymbols.fnCollAppend)(conn->env->handle, error->handle,
            elem, elemInd, coll);
    return dpiError__check(error, status, conn, "append element");
}


//-----------------------------------------------------------------------------
// dpiOci__collAssignElem() [INTERNAL]
//   Wrapper for OCICollAssignElem().
//-----------------------------------------------------------------------------
int dpiOci__collAssignElem(dpiConn *conn, int32_t index, const void *elem,
        const void *elemInd, void *coll, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCICollAssignElem", dpiOciSymbols.fnCollAssignElem)
    status = (*dpiOciSymbols.fnCollAssignElem)(conn->env->handle,
            error->handle, index, elem, elemInd, coll);
    return dpiError__check(error, status, conn, "assign element");
}


//-----------------------------------------------------------------------------
// dpiOci__collGetElem() [INTERNAL]
//   Wrapper for OCICollGetElem().
//-----------------------------------------------------------------------------
int dpiOci__collGetElem(dpiConn *conn, void *coll, int32_t index, int *exists,
        void **elem, void **elemInd, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCICollGetElem", dpiOciSymbols.fnCollGetElem)
    status = (*dpiOciSymbols.fnCollGetElem)(conn->env->handle, error->handle,
            coll, index, exists, elem, elemInd);
    return dpiError__check(error, status, conn, "get element");
}


//-----------------------------------------------------------------------------
// dpiOci__collSize() [INTERNAL]
//   Wrapper for OCICollSize().
//-----------------------------------------------------------------------------
int dpiOci__collSize(dpiConn *conn, void *coll, int32_t *size, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCICollSize", dpiOciSymbols.fnCollSize)
    status = (*dpiOciSymbols.fnCollSize)(conn->env->handle, error->handle,
            coll, size);
    return dpiError__check(error, status, conn, "get size");
}


//-----------------------------------------------------------------------------
// dpiOci__collTrim() [INTERNAL]
//   Wrapper for OCICollTrim().
//-----------------------------------------------------------------------------
int dpiOci__collTrim(dpiConn *conn, uint32_t numToTrim, void *coll,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCICollTrim", dpiOciSymbols.fnCollTrim)
    status = (*dpiOciSymbols.fnCollTrim)(conn->env->handle, error->handle,
            (int32_t) numToTrim, coll);
    return dpiError__check(error, status, conn, "trim");
}


//-----------------------------------------------------------------------------
// dpiOci__contextGetValue() [INTERNAL]
//   Wrapper for OCIContextGetValue().
//-----------------------------------------------------------------------------
int dpiOci__contextGetValue(dpiConn *conn, const char *key, uint32_t keyLength,
        void **value, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIContextGetValue", dpiOciSymbols.fnContextGetValue)
    status = (*dpiOciSymbols.fnContextGetValue)(conn->sessionHandle,
            error->handle, key, (uint8_t) keyLength, value);
    if (checkError)
        return dpiError__check(error, status, conn, "get context value");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__contextSetValue() [INTERNAL]
//   Wrapper for OCIContextSetValue().
//-----------------------------------------------------------------------------
int dpiOci__contextSetValue(dpiConn *conn, const char *key, uint32_t keyLength,
        void *value, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIContextSetValue", dpiOciSymbols.fnContextSetValue)
    status = (*dpiOciSymbols.fnContextSetValue)(conn->sessionHandle,
            error->handle, DPI_OCI_DURATION_SESSION, key, (uint8_t) keyLength,
            value);
    if (checkError)
        return dpiError__check(error, status, conn, "set context value");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeConstruct() [INTERNAL]
//   Wrapper for OCIDateTimeConstruct().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeConstruct(void *envHandle, void *handle, int16_t year,
        uint8_t month, uint8_t day, uint8_t hour, uint8_t minute,
        uint8_t second, uint32_t fsecond, const char *tz, size_t tzLength,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeConstruct",
            dpiOciSymbols.fnDateTimeConstruct)
    status = (*dpiOciSymbols.fnDateTimeConstruct)(envHandle, error->handle,
            handle, year, month, day, hour, minute, second, fsecond, tz,
            tzLength);
    return dpiError__check(error, status, NULL, "construct date");
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeGetDate() [INTERNAL]
//   Wrapper for OCIDateTimeGetDate().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeGetDate(void *envHandle, void *handle, int16_t *year,
        uint8_t *month, uint8_t *day, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeGetDate", dpiOciSymbols.fnDateTimeGetDate)
    status = (*dpiOciSymbols.fnDateTimeGetDate)(envHandle, error->handle,
            handle, year, month, day);
    return dpiError__check(error, status, NULL, "get date portion");
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeGetTime() [INTERNAL]
//   Wrapper for OCIDateTimeGetTime().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeGetTime(void *envHandle, void *handle, uint8_t *hour,
        uint8_t *minute, uint8_t *second, uint32_t *fsecond, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeGetTime", dpiOciSymbols.fnDateTimeGetTime)
    status = (*dpiOciSymbols.fnDateTimeGetTime)(envHandle, error->handle,
            handle, hour, minute, second, fsecond);
    return dpiError__check(error, status, NULL, "get time portion");
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeGetTimeZoneOffset() [INTERNAL]
//   Wrapper for OCIDateTimeGetTimeZoneOffset().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeGetTimeZoneOffset(void *envHandle, void *handle,
        int8_t *tzHourOffset, int8_t *tzMinuteOffset, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeGetTimeZoneOffset",
            dpiOciSymbols.fnDateTimeGetTimeZoneOffset)
    status = (*dpiOciSymbols.fnDateTimeGetTimeZoneOffset)(envHandle,
            error->handle, handle, tzHourOffset, tzMinuteOffset);
    return dpiError__check(error, status, NULL, "get time zone portion");
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeIntervalAdd() [INTERNAL]
//   Wrapper for OCIDateTimeIntervalAdd().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeIntervalAdd(void *envHandle, void *handle, void *interval,
        void *outHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeIntervalAdd",
            dpiOciSymbols.fnDateTimeIntervalAdd)
    status = (*dpiOciSymbols.fnDateTimeIntervalAdd)(envHandle, error->handle,
            handle, interval, outHandle);
    return dpiError__check(error, status, NULL, "add interval to date");
}


//-----------------------------------------------------------------------------
// dpiOci__dateTimeSubtract() [INTERNAL]
//   Wrapper for OCIDateTimeSubtract().
//-----------------------------------------------------------------------------
int dpiOci__dateTimeSubtract(void *envHandle, void *handle1, void *handle2,
        void *interval, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDateTimeSubtract",
            dpiOciSymbols.fnDateTimeSubtract)
    status = (*dpiOciSymbols.fnDateTimeSubtract)(envHandle, error->handle,
            handle1, handle2, interval);
    return dpiError__check(error, status, NULL, "subtract date");
}


//-----------------------------------------------------------------------------
// dpiOci__dbShutdown() [INTERNAL]
//   Wrapper for OCIDBShutdown().
//-----------------------------------------------------------------------------
int dpiOci__dbShutdown(dpiConn *conn, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDBShutdown", dpiOciSymbols.fnDbShutdown)
    status = (*dpiOciSymbols.fnDbShutdown)(conn->handle, error->handle, NULL,
            mode);
    return dpiError__check(error, status, NULL, "shutdown database");
}


//-----------------------------------------------------------------------------
// dpiOci__dbStartup() [INTERNAL]
//   Wrapper for OCIDBStartup().
//-----------------------------------------------------------------------------
int dpiOci__dbStartup(dpiConn *conn, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDBStartup", dpiOciSymbols.fnDbStartup)
    status = (*dpiOciSymbols.fnDbStartup)(conn->handle, error->handle, NULL,
            DPI_OCI_DEFAULT, mode);
    return dpiError__check(error, status, NULL, "startup database");
}


//-----------------------------------------------------------------------------
// dpiOci__defineByPos() [INTERNAL]
//   Wrapper for OCIDefineByPos().
//-----------------------------------------------------------------------------
int dpiOci__defineByPos(dpiStmt *stmt, void **defineHandle, uint32_t pos,
        dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDefineByPos", dpiOciSymbols.fnDefineByPos)
    status = (*dpiOciSymbols.fnDefineByPos)(stmt->handle, defineHandle,
            error->handle, pos, (var->isDynamic) ? NULL :
                    var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : (int32_t) var->sizeInBytes,
            var->type->oracleType, (var->isDynamic) ? NULL :
                    var->buffer.indicator,
            (var->isDynamic) ? NULL : var->buffer.actualLength16,
            (var->isDynamic) ? NULL : var->buffer.returnCode,
            (var->isDynamic) ? DPI_OCI_DYNAMIC_FETCH : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "define");
}


//-----------------------------------------------------------------------------
// dpiOci__defineByPos2() [INTERNAL]
//   Wrapper for OCIDefineByPos2().
//-----------------------------------------------------------------------------
int dpiOci__defineByPos2(dpiStmt *stmt, void **defineHandle, uint32_t pos,
        dpiVar *var, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDefineByPos2", dpiOciSymbols.fnDefineByPos2)
    status = (*dpiOciSymbols.fnDefineByPos2)(stmt->handle, defineHandle,
            error->handle, pos, (var->isDynamic) ? NULL :
                    var->buffer.data.asRaw,
            (var->isDynamic) ? INT_MAX : var->sizeInBytes,
            var->type->oracleType, (var->isDynamic) ? NULL :
                    var->buffer.indicator,
            (var->isDynamic) ? NULL : var->buffer.actualLength32,
            (var->isDynamic) ? NULL : var->buffer.returnCode,
            (var->isDynamic) ? DPI_OCI_DYNAMIC_FETCH : DPI_OCI_DEFAULT);
    return dpiError__check(error, status, stmt->conn, "define");
}


//-----------------------------------------------------------------------------
// dpiOci__defineDynamic() [INTERNAL]
//   Wrapper for OCIDefineDynamic().
//-----------------------------------------------------------------------------
int dpiOci__defineDynamic(dpiVar *var, void *defineHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDefineDynamic", dpiOciSymbols.fnDefineDynamic)
    status = (*dpiOciSymbols.fnDefineDynamic)(defineHandle, error->handle, var,
            (void*) dpiVar__defineCallback);
    return dpiError__check(error, status, var->conn, "define dynamic");
}


//-----------------------------------------------------------------------------
// dpiOci__defineObject() [INTERNAL]
//   Wrapper for OCIDefineObject().
//-----------------------------------------------------------------------------
int dpiOci__defineObject(dpiVar *var, void *defineHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDefineObject", dpiOciSymbols.fnDefineObject)
    status = (*dpiOciSymbols.fnDefineObject)(defineHandle, error->handle,
            var->objectType->tdo, (void**) var->buffer.data.asRaw, 0,
            var->buffer.objectIndicator, 0);
    return dpiError__check(error, status, var->conn, "define object");
}


//-----------------------------------------------------------------------------
// dpiOci__describeAny() [INTERNAL]
//   Wrapper for OCIDescribeAny().
//-----------------------------------------------------------------------------
int dpiOci__describeAny(dpiConn *conn, void *obj, uint32_t objLength,
        uint8_t objType, void *describeHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDescribeAny", dpiOciSymbols.fnDescribeAny)
    status = (*dpiOciSymbols.fnDescribeAny)(conn->handle, error->handle, obj,
            objLength, objType, 0, DPI_OCI_PTYPE_TYPE, describeHandle);
    return dpiError__check(error, status, conn, "describe type");
}


//-----------------------------------------------------------------------------
// dpiOci__descriptorAlloc() [INTERNAL]
//   Wrapper for OCIDescriptorAlloc().
//-----------------------------------------------------------------------------
int dpiOci__descriptorAlloc(void *envHandle, void **handle,
        const uint32_t handleType, const char *action, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDescriptorAlloc", dpiOciSymbols.fnDescriptorAlloc)
    status = (*dpiOciSymbols.fnDescriptorAlloc)(envHandle, handle, handleType,
            0, NULL);
    return dpiError__check(error, status, NULL, action);
}


//-----------------------------------------------------------------------------
// dpiOci__descriptorFree() [INTERNAL]
//   Wrapper for OCIDescriptorFree().
//-----------------------------------------------------------------------------
int dpiOci__descriptorFree(void *handle, uint32_t handleType)
{
    dpiError *error = NULL;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIDescriptorFree", dpiOciSymbols.fnDescriptorFree)
    status = (*dpiOciSymbols.fnDescriptorFree)(handle, handleType);
    if (status != DPI_OCI_SUCCESS && dpiDebugLevel & DPI_DEBUG_LEVEL_FREES)
        dpiDebug__print("free descriptor %p, type %d failed\n", handle,
                handleType);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__envNlsCreate() [INTERNAL]
//   Wrapper for OCIEnvNlsCreate().
//-----------------------------------------------------------------------------
int dpiOci__envNlsCreate(void **envHandle, uint32_t mode, uint16_t charsetId,
        uint16_t ncharsetId, dpiError *error)
{
    void *mallocFn = NULL, *reallocFn = NULL, *freeFn = NULL;
    int status;

    *envHandle = NULL;
    DPI_OCI_LOAD_SYMBOL("OCIEnvNlsCreate", dpiOciSymbols.fnEnvNlsCreate)
    if (dpiDebugLevel & DPI_DEBUG_LEVEL_MEM) {
        mallocFn = (void*) dpiOci__allocateMem;
        reallocFn = (void*) dpiOci__reallocMem;
        freeFn = (void*) dpiOci__freeMem;
    }
    status = (*dpiOciSymbols.fnEnvNlsCreate)(envHandle, mode, NULL, mallocFn,
            reallocFn, freeFn, 0, NULL, charsetId, ncharsetId);
    if (*envHandle) {
        if (status == DPI_OCI_SUCCESS || status == DPI_OCI_SUCCESS_WITH_INFO)
            return DPI_SUCCESS;
        if (dpiOci__errorGet(*envHandle, DPI_OCI_HTYPE_ENV, charsetId,
                "create env", error) == 0)
            return DPI_FAILURE;
    }
    return dpiError__set(error, "create env", DPI_ERR_CREATE_ENV);
}


//-----------------------------------------------------------------------------
// dpiOci__errorGet() [INTERNAL]
//   Wrapper for OCIErrorGet().
//-----------------------------------------------------------------------------
int dpiOci__errorGet(void *handle, uint32_t handleType, uint16_t charsetId,
        const char *action, dpiError *error)
{
    uint32_t i, numChars, bufferChars;
    uint16_t *utf16chars;
    int status;
    char *ptr;

    DPI_OCI_LOAD_SYMBOL("OCIErrorGet", dpiOciSymbols.fnErrorGet)
    status = (*dpiOciSymbols.fnErrorGet)(handle, 1, NULL, &error->buffer->code,
            error->buffer->message, sizeof(error->buffer->message),
            handleType);
    if (status != DPI_OCI_SUCCESS)
        return dpiError__set(error, action, DPI_ERR_GET_FAILED);
    error->buffer->action = action;

    // determine length of message since OCI does not provide this information;
    // all encodings except UTF-16 can use normal string processing; cannot use
    // type whar_t for processing UTF-16, though, as its size may be 4 on some
    // platforms, not 2; also strip trailing whitespace from error messages
    if (charsetId == DPI_CHARSET_ID_UTF16) {
        numChars = 0;
        utf16chars = (uint16_t*) error->buffer->message;
        bufferChars = sizeof(error->buffer->message) / 2;
        for (i = 0; i < bufferChars; i++) {
            if (utf16chars[i] == 0)
                break;
            if (utf16chars[i] > 127 || !isspace(utf16chars[i]))
                numChars = i + 1;
        }
        error->buffer->messageLength = numChars * 2;
    } else {
        error->buffer->messageLength =
                (uint32_t) strlen(error->buffer->message);
        ptr = error->buffer->message + error->buffer->messageLength - 1;
        while (ptr > error->buffer->message && isspace((uint8_t) *ptr--))
            error->buffer->messageLength--;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__freeMem() [INTERNAL]
//   Wrapper for OCI allocation of memory, only used when debugging memory
// allocation.
//-----------------------------------------------------------------------------
static void dpiOci__freeMem(void *unused, void *ptr)
{
    char message[40];

    (void) sprintf(message, "OCI freed ptr at %p", ptr);
    free(ptr);
    dpiDebug__print("%s\n", message);
}


//-----------------------------------------------------------------------------
// dpiOci__handleAlloc() [INTERNAL]
//   Wrapper for OCIHandleAlloc().
//-----------------------------------------------------------------------------
int dpiOci__handleAlloc(void *envHandle, void **handle, uint32_t handleType,
        const char *action, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIHandleAlloc", dpiOciSymbols.fnHandleAlloc)
    status = (*dpiOciSymbols.fnHandleAlloc)(envHandle, handle, handleType, 0,
            NULL);
    if (handleType == DPI_OCI_HTYPE_ERROR && status != DPI_OCI_SUCCESS)
        return dpiError__set(error, action, DPI_ERR_NO_MEMORY);
    return dpiError__check(error, status, NULL, action);
}


//-----------------------------------------------------------------------------
// dpiOci__handleFree() [INTERNAL]
//   Wrapper for OCIHandleFree().
//-----------------------------------------------------------------------------
int dpiOci__handleFree(void *handle, uint32_t handleType)
{
    dpiError *error = NULL;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIHandleFree", dpiOciSymbols.fnHandleFree)
    status = (*dpiOciSymbols.fnHandleFree)(handle, handleType);
    if (status != DPI_OCI_SUCCESS && dpiDebugLevel & DPI_DEBUG_LEVEL_FREES)
        dpiDebug__print("free handle %p, handleType %d failed\n", handle,
                handleType);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__intervalGetDaySecond() [INTERNAL]
//   Wrapper for OCIIntervalGetDaySecond().
//-----------------------------------------------------------------------------
int dpiOci__intervalGetDaySecond(void *envHandle, int32_t *day, int32_t *hour,
        int32_t *minute, int32_t *second, int32_t *fsecond,
        const void *interval, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIIntervalGetDaySecond",
            dpiOciSymbols.fnIntervalGetDaySecond)
    status = (*dpiOciSymbols.fnIntervalGetDaySecond)(envHandle,
            error->handle, day, hour, minute, second, fsecond, interval);
    return dpiError__check(error, status, NULL, "get interval components");
}


//-----------------------------------------------------------------------------
// dpiOci__intervalGetYearMonth() [INTERNAL]
//   Wrapper for OCIIntervalGetYearMonth().
//-----------------------------------------------------------------------------
int dpiOci__intervalGetYearMonth(void *envHandle, int32_t *year,
        int32_t *month, const void *interval, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIIntervalGetYearMonth",
            dpiOciSymbols.fnIntervalGetYearMonth)
    status = (*dpiOciSymbols.fnIntervalGetYearMonth)(envHandle, error->handle,
            year, month, interval);
    return dpiError__check(error, status, NULL, "get interval components");
}


//-----------------------------------------------------------------------------
// dpiOci__intervalSetDaySecond() [INTERNAL]
//   Wrapper for OCIIntervalSetDaySecond().
//-----------------------------------------------------------------------------
int dpiOci__intervalSetDaySecond(void *envHandle, int32_t day, int32_t hour,
        int32_t minute, int32_t second, int32_t fsecond, void *interval,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIIntervalSetDaySecond",
            dpiOciSymbols.fnIntervalSetDaySecond)
    status = (*dpiOciSymbols.fnIntervalSetDaySecond)(envHandle, error->handle,
            day, hour, minute, second, fsecond, interval);
    return dpiError__check(error, status, NULL, "set interval components");
}


//-----------------------------------------------------------------------------
// dpiOci__intervalSetYearMonth() [INTERNAL]
//   Wrapper for OCIIntervalSetYearMonth().
//-----------------------------------------------------------------------------
int dpiOci__intervalSetYearMonth(void *envHandle, int32_t year, int32_t month,
        void *interval, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIIntervalSetYearMonth",
            dpiOciSymbols.fnIntervalSetYearMonth)
    status = (*dpiOciSymbols.fnIntervalSetYearMonth)(envHandle, error->handle,
            year, month, interval);
    return dpiError__check(error, status, NULL, "set interval components");
}


#ifdef _WIN32
//-----------------------------------------------------------------------------
// dpiOci__checkDllArchitecture() [INTERNAL]
//   Check the architecture of the specified DLL name and check that it
// matches the expected architecture. Returns -1 if the DLL architecture could
// not be determined, 0 if it does not match and 1 if it matches.
//-----------------------------------------------------------------------------
static int dpiOci__checkDllArchitecture(const char *name)
{
    IMAGE_DOS_HEADER dosHeader;
    IMAGE_NT_HEADERS ntHeaders;
    FILE *fp;

    fp = fopen(name, "rb");
    if (!fp)
        return -1;
    fread(&dosHeader, sizeof(dosHeader), 1, fp);
    if (dosHeader.e_magic != IMAGE_DOS_SIGNATURE) {
        fclose(fp);
        return -1;
    }
    fseek(fp, dosHeader.e_lfanew, SEEK_SET);
    fread(&ntHeaders, sizeof(ntHeaders), 1, fp);
    fclose(fp);
    if (ntHeaders.Signature != IMAGE_NT_SIGNATURE)
        return -1;
#if defined _M_AMD64
    if (ntHeaders.FileHeader.Machine == IMAGE_FILE_MACHINE_AMD64)
        return 1;
#elif defined _M_IX86
    if (ntHeaders.FileHeader.Machine == IMAGE_FILE_MACHINE_I386)
        return 1;
#endif
    return 0;
}


//-----------------------------------------------------------------------------
// dpiOci__findAndCheckDllArchitecture() [INTERNAL]
//   Attempt to find the specified DLL name using the standard search path and
// if the DLL can be found but is of the wrong architecture, include the full
// name of the DLL in the load error. Return -1 if such a DLL could not be
// found and 0 if the load error was changed.
//-----------------------------------------------------------------------------
static int dpiOci__findAndCheckDllArchitecture(const char *dllName,
        char *loadError, size_t loadErrorLength)
{
    char fullName[_MAX_PATH + 1], *path, *temp;
    size_t length;
    int found = 0;

    // first search executable directory
    if (GetModuleFileName(NULL, fullName, sizeof(fullName)) != 0) {
        temp = strrchr(fullName, '\\');
        if (temp) {
            *(temp + 1) = '\0';
            strncat(fullName, dllName,
                    sizeof(fullName) - strlen(fullName) - 1);
            if (dpiOci__checkDllArchitecture(fullName) == 0)
                found = 1;
        }
    }

    // check current directory
    if (!found && GetCurrentDirectory(sizeof(fullName), fullName) != 0) {
        temp = fullName + strlen(fullName);
        snprintf(temp, sizeof(fullName) - strlen(fullName), "\\%s", dllName);
        if (dpiOci__checkDllArchitecture(fullName) == 0)
            found = 1;
    }

    // search PATH
    path = getenv("PATH");
    if (path) {
        while (!found) {
            temp = strchr(path, ';');
            if (!temp)
                length = strlen(path);
            else length = temp - path;
            if (length <= _MAX_DIR) {
                snprintf(fullName, sizeof(fullName), "%.*s\\%s", (int) length,
                        path, dllName);
                if (dpiOci__checkDllArchitecture(fullName) == 0) {
                    found = 1;
                    break;
                }
            }
            if (!temp)
                break;
            path = temp + 1;
        }
    }

    // if found, adjust the load error
    if (found) {
        snprintf(loadError, loadErrorLength,
                "%s is not the correct architecture", fullName);
        loadError[loadErrorLength - 1] = '\0';
        return 0;
    }

    return -1;
}


//-----------------------------------------------------------------------------
// dpiOci__getLoadErrorOnWindows() [INTERNAL]
//   Get the error message for a load failure on Windows.
//-----------------------------------------------------------------------------
static void dpiOci__getLoadErrorOnWindows(const char *dllName,
        char *loadError, size_t loadErrorLength)
{
    DWORD length = 0, errorNum, status;
    wchar_t *wLoadError = NULL;

    // if DLL is of the wrong architecture, attempt to locate the DLL that was
    // loaded and use that information if it can be found
    errorNum = GetLastError();
    if (errorNum == ERROR_BAD_EXE_FORMAT &&
            dpiOci__findAndCheckDllArchitecture(dllName, loadError,
                    loadErrorLength) == 0)
        return;

    // get error message in Unicode first
    // use English unless English error messages aren't available
    status = FormatMessageW(FORMAT_MESSAGE_FROM_SYSTEM |
            FORMAT_MESSAGE_IGNORE_INSERTS | FORMAT_MESSAGE_ALLOCATE_BUFFER,
            NULL, errorNum, MAKELANGID(LANG_ENGLISH, SUBLANG_ENGLISH_US),
            (LPWSTR) &wLoadError, 0, NULL);
    if (!status && GetLastError() == ERROR_MUI_FILE_NOT_FOUND)
        FormatMessageW(FORMAT_MESSAGE_FROM_SYSTEM |
                FORMAT_MESSAGE_IGNORE_INSERTS | FORMAT_MESSAGE_ALLOCATE_BUFFER,
                NULL, errorNum, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
                (LPWSTR) &wLoadError, 0, NULL);

    if (wLoadError) {

        // strip trailing period and carriage return from message, if needed
        length = (DWORD) wcslen(wLoadError);
        while (length > 0) {
            if (wLoadError[length - 1] > 127 ||
                    (wLoadError[length - 1] != L'.' &&
                    !isspace(wLoadError[length - 1])))
                break;
            length--;
        }
        wLoadError[length] = L'\0';

        // convert to a multi-byte string in UTF-8 encoding
        if (length > 0)
            length = WideCharToMultiByte(CP_UTF8, 0, wLoadError, -1, loadError,
                    (int) loadErrorLength, NULL, NULL);
        LocalFree(wLoadError);

    }

    // fallback in case message cannot be determined
    if (length == 0)
        sprintf(loadError, "DLL load failed: Windows Error %d", errorNum);
}


//-----------------------------------------------------------------------------
// dpiOci__loadLibOnWindows() [INTERNAL]
//   Load the library on the Windows platform. First an attempt is made to
// determine the location of the module containing ODPI-C. If found, an attempt
// is made to load oci.dll from that location; otherwise a standard Windows
// search is made for oci.dll.
//-----------------------------------------------------------------------------
static void dpiOci__loadLibOnWindows(const char *dllName)
{
    char moduleName[MAX_PATH + 1], *temp;
    HMODULE module = NULL;

    // attempt to determine the location of the module containing ODPI-C;
    // errors in this code are ignored and the normal loading mechanism is
    // used instead
    if (GetModuleHandleEx(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
            (LPCSTR) dpiOci__loadLibOnWindows, &module)) {
        if (GetModuleFileName(module, moduleName, sizeof(moduleName)) > 0) {
            temp = strrchr(moduleName, '\\');
            if (temp) {
                *(temp + 1) = '\0';
                strncat(moduleName, dllName,
                        sizeof(moduleName) - strlen(moduleName) - 1);
                dpiOciLibHandle = LoadLibrary(moduleName);
            }
        }
        FreeLibrary(module);
    }

    // if library was not loaded in the same location as ODPI-C, use the
    // standard Windows search to locate oci.dll instead
    if (!dpiOciLibHandle)
        dpiOciLibHandle = LoadLibrary(dllName);
}
#endif


//-----------------------------------------------------------------------------
// dpiOci__loadLib() [INTERNAL]
//   Load the OCI library.
//-----------------------------------------------------------------------------
static int dpiOci__loadLib(dpiError *error)
{
    const char *libName;
    char loadError[512];
    unsigned int i;
#ifndef _WIN32
    char *oracleHome, *oracleHomeLibName;
    size_t oracleHomeLibNameLength;
#endif

    // dynamically load the OCI library
    for (i = 0; !dpiOciLibHandle; i++) {
        libName = dpiOciLibNames[i];
        if (!libName)
            break;
#ifdef _WIN32
        dpiOci__loadLibOnWindows(libName);
        if (!dpiOciLibHandle && i == 0)
            dpiOci__getLoadErrorOnWindows(libName, loadError,
                    sizeof(loadError));
#else
        dpiOciLibHandle = dlopen(libName, RTLD_LAZY);
        if (!dpiOciLibHandle && i == 0) {
            strncpy(loadError, dlerror(), sizeof(loadError) - 1);
            loadError[sizeof(loadError) - 1] = '\0';
        }
#endif

    }

#ifndef _WIN32
    // on platforms other than Windows, attempt to use
    // $ORACLE_HOME/lib/libclntsh.so
    if (!dpiOciLibHandle) {
        oracleHome = getenv("ORACLE_HOME");
        if (oracleHome) {
            oracleHomeLibNameLength = strlen(oracleHome) + 6 +
                    strlen(dpiOciLibNames[0]);
            oracleHomeLibName = (char*) malloc(oracleHomeLibNameLength);
            if (oracleHomeLibName) {
                (void) sprintf(oracleHomeLibName, "%s/lib/%s", oracleHome,
                        dpiOciLibNames[0]);
                dpiOciLibHandle = dlopen(oracleHomeLibName, RTLD_LAZY);
                free(oracleHomeLibName);
            }
        }
    }
#endif

    if (!dpiOciLibHandle) {
        const char *bits = (sizeof(void*) == 8) ? "64" : "32";
        return dpiError__set(error, "load library", DPI_ERR_LOAD_LIBRARY,
                bits, loadError, DPI_ERR_LOAD_URL_FRAGMENT);
    }

    // validate library
    if (dpiOci__loadLibValidate(error) < 0) {
#ifdef _WIN32
        FreeLibrary(dpiOciLibHandle);
#else
        dlclose(dpiOciLibHandle);
#endif
        dpiOciLibHandle = NULL;
        memset(&dpiOciSymbols, 0, sizeof(dpiOciSymbols));
        return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__loadLibValidate() [INTERNAL]
//   Validate the OCI library after loading.
//-----------------------------------------------------------------------------
static int dpiOci__loadLibValidate(dpiError *error)
{
    // determine the OCI client version information
    if (dpiOci__loadSymbol("OCIClientVersion",
            (void**) &dpiOciSymbols.fnClientVersion, NULL) < 0)
        return dpiError__set(error, "check Oracle Client version",
                DPI_ERR_ORACLE_CLIENT_TOO_OLD, 0, 0, 11, 2);
    (*dpiOciSymbols.fnClientVersion)(&dpiOciLibVersionInfo.versionNum,
            &dpiOciLibVersionInfo.releaseNum,
            &dpiOciLibVersionInfo.updateNum,
            &dpiOciLibVersionInfo.portReleaseNum,
            &dpiOciLibVersionInfo.portUpdateNum);
    dpiOciLibVersionInfo.fullVersionNum = (uint32_t)
            DPI_ORACLE_VERSION_TO_NUMBER(dpiOciLibVersionInfo.versionNum,
                    dpiOciLibVersionInfo.releaseNum,
                    dpiOciLibVersionInfo.updateNum,
                    dpiOciLibVersionInfo.portReleaseNum,
                    dpiOciLibVersionInfo.portUpdateNum);

    // OCI version must be a minimum of 11.2
    if (dpiUtils__checkClientVersion(&dpiOciLibVersionInfo, 11, 2, error) < 0)
        return DPI_FAILURE;

    // initialize threading capability in the OCI library
    // this must be run prior to any other OCI threading calls
    DPI_OCI_LOAD_SYMBOL("OCIThreadProcessInit",
            dpiOciSymbols.fnThreadProcessInit)
    (*dpiOciSymbols.fnThreadProcessInit)();

    // load symbols for key functions which are called many times
    // this list should be kept as small as possible in order to avoid
    // overhead in looking up symbols at startup
    DPI_OCI_LOAD_SYMBOL("OCIAttrGet", dpiOciSymbols.fnAttrGet)
    DPI_OCI_LOAD_SYMBOL("OCIAttrSet", dpiOciSymbols.fnAttrSet)
    DPI_OCI_LOAD_SYMBOL("OCIThreadKeyGet", dpiOciSymbols.fnThreadKeyGet)

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__loadSymbol() [INTERNAL]
//   Return the symbol for the function that is to be called. The symbol table
// is first consulted. If the symbol is not found there, it is looked up and
// then stored there so the next invocation does not have to perform the
// lookup.
//-----------------------------------------------------------------------------
static int dpiOci__loadSymbol(const char *symbolName, void **symbol,
        dpiError *error)
{
    // if library not already opened, open it
    if (!dpiOciLibHandle && dpiOci__loadLib(error) < 0)
        return DPI_FAILURE;

    // load symbol
#ifdef _WIN32
    *symbol = GetProcAddress(dpiOciLibHandle, symbolName);
#else
    *symbol = dlsym(dpiOciLibHandle, symbolName);
#endif
    if (!*symbol)
        return dpiError__set(error, "get symbol", DPI_ERR_LOAD_SYMBOL,
                symbolName);

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__lobClose() [INTERNAL]
//   Wrapper for OCILobClose().
//-----------------------------------------------------------------------------
int dpiOci__lobClose(dpiLob *lob, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobClose", dpiOciSymbols.fnLobClose)
    status = (*dpiOciSymbols.fnLobClose)(lob->conn->handle, error->handle,
            lob->locator);
    return dpiError__check(error, status, lob->conn, "close LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__lobCreateTemporary() [INTERNAL]
//   Wrapper for OCILobCreateTemporary().
//-----------------------------------------------------------------------------
int dpiOci__lobCreateTemporary(dpiLob *lob, dpiError *error)
{
    uint8_t lobType;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobCreateTemporary",
            dpiOciSymbols.fnLobCreateTemporary)
    if (lob->type->oracleTypeNum == DPI_ORACLE_TYPE_BLOB)
        lobType = DPI_OCI_TEMP_BLOB;
    else lobType = DPI_OCI_TEMP_CLOB;
    status = (*dpiOciSymbols.fnLobCreateTemporary)(lob->conn->handle,
            error->handle, lob->locator, DPI_OCI_DEFAULT,
            lob->type->charsetForm, lobType, 1, DPI_OCI_DURATION_SESSION);
    return dpiError__check(error, status, lob->conn, "create temporary LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__lobFileExists() [INTERNAL]
//   Wrapper for OCILobFileExists().
//-----------------------------------------------------------------------------
int dpiOci__lobFileExists(dpiLob *lob, int *exists, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobFileExists", dpiOciSymbols.fnLobFileExists)
    status = (*dpiOciSymbols.fnLobFileExists)(lob->conn->handle, error->handle,
            lob->locator, exists);
    return dpiError__check(error, status, lob->conn, "get file exists");
}


//-----------------------------------------------------------------------------
// dpiOci__lobFileGetName() [INTERNAL]
//   Wrapper for OCILobFileGetName().
//-----------------------------------------------------------------------------
int dpiOci__lobFileGetName(dpiLob *lob, char *dirAlias,
        uint16_t *dirAliasLength, char *name, uint16_t *nameLength,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobFileGetName", dpiOciSymbols.fnLobFileGetName)
    status = (*dpiOciSymbols.fnLobFileGetName)(lob->env->handle, error->handle,
            lob->locator, dirAlias, dirAliasLength, name, nameLength);
    return dpiError__check(error, status, lob->conn, "get LOB file name");
}


//-----------------------------------------------------------------------------
// dpiOci__lobFileSetName() [INTERNAL]
//   Wrapper for OCILobFileSetName().
//-----------------------------------------------------------------------------
int dpiOci__lobFileSetName(dpiLob *lob, const char *dirAlias,
        uint16_t dirAliasLength, const char *name, uint16_t nameLength,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobFileSetName", dpiOciSymbols.fnLobFileSetName)
    status = (*dpiOciSymbols.fnLobFileSetName)(lob->env->handle, error->handle,
            &lob->locator, dirAlias, dirAliasLength, name, nameLength);
    return dpiError__check(error, status, lob->conn, "set LOB file name");
}


//-----------------------------------------------------------------------------
// dpiOci__lobFreeTemporary() [INTERNAL]
//   Wrapper for OCILobFreeTemporary().
//-----------------------------------------------------------------------------
int dpiOci__lobFreeTemporary(dpiConn *conn, void *lobLocator, int checkError,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobFreeTemporary",
            dpiOciSymbols.fnLobFreeTemporary)
    status = (*dpiOciSymbols.fnLobFreeTemporary)(conn->handle,
            error->handle, lobLocator);
    if (checkError)
        return dpiError__check(error, status, conn, "free temporary LOB");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__lobGetChunkSize() [INTERNAL]
//   Wrapper for OCILobGetChunkSize().
//-----------------------------------------------------------------------------
int dpiOci__lobGetChunkSize(dpiLob *lob, uint32_t *size, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobGetChunkSize", dpiOciSymbols.fnLobGetChunkSize)
    status = (*dpiOciSymbols.fnLobGetChunkSize)(lob->conn->handle,
            error->handle, lob->locator, size);
    return dpiError__check(error, status, lob->conn, "get chunk size");
}


//-----------------------------------------------------------------------------
// dpiOci__lobGetLength2() [INTERNAL]
//   Wrapper for OCILobGetLength2().
//-----------------------------------------------------------------------------
int dpiOci__lobGetLength2(dpiLob *lob, uint64_t *size, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobGetLength2", dpiOciSymbols.fnLobGetLength2)
    status = (*dpiOciSymbols.fnLobGetLength2)(lob->conn->handle, error->handle,
            lob->locator, size);
    return dpiError__check(error, status, lob->conn, "get length");
}


//-----------------------------------------------------------------------------
// dpiOci__lobIsOpen() [INTERNAL]
//   Wrapper for OCILobIsOpen().
//-----------------------------------------------------------------------------
int dpiOci__lobIsOpen(dpiLob *lob, int *isOpen, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobIsOpen", dpiOciSymbols.fnLobIsOpen)
    status = (*dpiOciSymbols.fnLobIsOpen)(lob->conn->handle, error->handle,
            lob->locator, isOpen);
    return dpiError__check(error, status, lob->conn, "check is open");
}


//-----------------------------------------------------------------------------
// dpiOci__lobIsTemporary() [INTERNAL]
//   Wrapper for OCILobIsTemporary().
//-----------------------------------------------------------------------------
int dpiOci__lobIsTemporary(dpiLob *lob, int *isTemporary, int checkError,
        dpiError *error)
{
    int status;

    *isTemporary = 0;
    DPI_OCI_LOAD_SYMBOL("OCILobIsTemporary", dpiOciSymbols.fnLobIsTemporary)
    status = (*dpiOciSymbols.fnLobIsTemporary)(lob->env->handle, error->handle,
            lob->locator, isTemporary);
    if (checkError)
        return dpiError__check(error, status, lob->conn, "check is temporary");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__lobLocatorAssign() [INTERNAL]
//   Wrapper for OCILobLocatorAssign().
//-----------------------------------------------------------------------------
int dpiOci__lobLocatorAssign(dpiLob *lob, void **copiedHandle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobLocatorAssign",
            dpiOciSymbols.fnLobLocatorAssign)
    status = (*dpiOciSymbols.fnLobLocatorAssign)(lob->conn->handle,
            error->handle, lob->locator, copiedHandle);
    return dpiError__check(error, status, lob->conn, "assign locator");
}


//-----------------------------------------------------------------------------
// dpiOci__lobOpen() [INTERNAL]
//   Wrapper for OCILobOpen().
//-----------------------------------------------------------------------------
int dpiOci__lobOpen(dpiLob *lob, dpiError *error)
{
    uint8_t mode;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobOpen", dpiOciSymbols.fnLobOpen)
    mode = (lob->type->oracleTypeNum == DPI_ORACLE_TYPE_BFILE) ?
            DPI_OCI_LOB_READONLY : DPI_OCI_LOB_READWRITE;
    status = (*dpiOciSymbols.fnLobOpen)(lob->conn->handle, error->handle,
            lob->locator, mode);
    return dpiError__check(error, status, lob->conn, "close LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__lobRead2() [INTERNAL]
//   Wrapper for OCILobRead2().
//-----------------------------------------------------------------------------
int dpiOci__lobRead2(dpiLob *lob, uint64_t offset, uint64_t *amountInBytes,
        uint64_t *amountInChars, char *buffer, uint64_t bufferLength,
        dpiError *error)
{
    uint16_t charsetId;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobRead2", dpiOciSymbols.fnLobRead2)
    charsetId = (lob->type->charsetForm == DPI_SQLCS_NCHAR) ?
            lob->env->ncharsetId : lob->env->charsetId;
    status = (*dpiOciSymbols.fnLobRead2)(lob->conn->handle, error->handle,
            lob->locator, amountInBytes, amountInChars, offset, buffer,
            bufferLength, DPI_OCI_ONE_PIECE, NULL, NULL, charsetId,
            lob->type->charsetForm);
    return dpiError__check(error, status, lob->conn, "read from LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__lobTrim2() [INTERNAL]
//   Wrapper for OCILobTrim2().
//-----------------------------------------------------------------------------
int dpiOci__lobTrim2(dpiLob *lob, uint64_t newLength, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobTrim2", dpiOciSymbols.fnLobTrim2)
    status = (*dpiOciSymbols.fnLobTrim2)(lob->conn->handle, error->handle,
            lob->locator, newLength);
    if (status == DPI_OCI_INVALID_HANDLE)
        return dpiOci__lobCreateTemporary(lob, error);
    return dpiError__check(error, status, lob->conn, "trim LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__lobWrite2() [INTERNAL]
//   Wrapper for OCILobWrite2().
//-----------------------------------------------------------------------------
int dpiOci__lobWrite2(dpiLob *lob, uint64_t offset, const char *value,
        uint64_t valueLength, dpiError *error)
{
    uint64_t lengthInBytes = valueLength, lengthInChars = 0;
    uint16_t charsetId;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCILobWrite2", dpiOciSymbols.fnLobWrite2)
    charsetId = (lob->type->charsetForm == DPI_SQLCS_NCHAR) ?
            lob->env->ncharsetId : lob->env->charsetId;
    status = (*dpiOciSymbols.fnLobWrite2)(lob->conn->handle, error->handle,
            lob->locator, &lengthInBytes, &lengthInChars, offset, (void*) value,
            valueLength, DPI_OCI_ONE_PIECE, NULL, NULL, charsetId,
            lob->type->charsetForm);
    return dpiError__check(error, status, lob->conn, "write to LOB");
}


//-----------------------------------------------------------------------------
// dpiOci__memoryAlloc() [INTERNAL]
//   Wrapper for OCIMemoryAlloc().
//-----------------------------------------------------------------------------
int dpiOci__memoryAlloc(dpiConn *conn, void **ptr, uint32_t size,
        int checkError, dpiError *error)
{
    int status;

    *ptr = NULL;
    DPI_OCI_LOAD_SYMBOL("OCIMemoryAlloc", dpiOciSymbols.fnMemoryAlloc)
    status = (*dpiOciSymbols.fnMemoryAlloc)(conn->sessionHandle, error->handle,
            ptr, DPI_OCI_DURATION_SESSION, size, DPI_OCI_MEMORY_CLEARED);
    if (checkError)
        return dpiError__check(error, status, conn, "allocate memory");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__memoryFree() [INTERNAL]
//   Wrapper for OCIMemoryFree().
//-----------------------------------------------------------------------------
int dpiOci__memoryFree(dpiConn *conn, void *ptr, dpiError *error)
{
    DPI_OCI_LOAD_SYMBOL("OCIMemoryFree", dpiOciSymbols.fnMemoryFree)
    (*dpiOciSymbols.fnMemoryFree)(conn->sessionHandle, error->handle, ptr);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__nlsCharSetConvert() [INTERNAL]
//   Wrapper for OCINlsCharSetConvert().
//-----------------------------------------------------------------------------
int dpiOci__nlsCharSetConvert(void *envHandle, uint16_t destCharsetId,
        char *dest, size_t destLength, uint16_t sourceCharsetId,
        const char *source, size_t sourceLength, size_t *resultSize,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINlsCharSetConvert",
            dpiOciSymbols.fnNlsCharSetConvert)
    status = (*dpiOciSymbols.fnNlsCharSetConvert)(envHandle, error->handle,
            destCharsetId, dest, destLength, sourceCharsetId, source,
            sourceLength, resultSize);
    return dpiError__check(error, status, NULL, "convert text");
}


//-----------------------------------------------------------------------------
// dpiOci__nlsCharSetIdToName() [INTERNAL]
//   Wrapper for OCINlsCharSetIdToName().
//-----------------------------------------------------------------------------
int dpiOci__nlsCharSetIdToName(void *envHandle, char *buf, size_t bufLength,
        uint16_t charsetId, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINlsCharSetIdToName",
            dpiOciSymbols.fnNlsCharSetIdToName)
    status = (*dpiOciSymbols.fnNlsCharSetIdToName)(envHandle, buf, bufLength,
            charsetId);
    return (status == DPI_OCI_SUCCESS) ? DPI_SUCCESS : DPI_FAILURE;
}


//-----------------------------------------------------------------------------
// dpiOci__nlsCharSetNameToId() [INTERNAL]
//   Wrapper for OCINlsCharSetNameToId().
//-----------------------------------------------------------------------------
int dpiOci__nlsCharSetNameToId(void *envHandle, const char *name,
        uint16_t *charsetId, dpiError *error)
{
    DPI_OCI_LOAD_SYMBOL("OCINlsCharSetNameToId",
            dpiOciSymbols.fnNlsCharSetNameToId)
    *charsetId = (*dpiOciSymbols.fnNlsCharSetNameToId)(envHandle, name);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__nlsEnvironmentVariableGet() [INTERNAL]
//   Wrapper for OCIEnvironmentVariableGet().
//-----------------------------------------------------------------------------
int dpiOci__nlsEnvironmentVariableGet(uint16_t item, void *value,
        dpiError *error)
{
    size_t ignored;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINlsEnvironmentVariableGet",
            dpiOciSymbols.fnNlsEnvironmentVariableGet)
    status = (*dpiOciSymbols.fnNlsEnvironmentVariableGet)(value, 0, item, 0,
            &ignored);
    if (status != DPI_OCI_SUCCESS)
        return dpiError__set(error, "get NLS environment variable",
                DPI_ERR_NLS_ENV_VAR_GET);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__nlsNameMap() [INTERNAL]
//   Wrapper for OCINlsNameMap().
//-----------------------------------------------------------------------------
int dpiOci__nlsNameMap(void *envHandle, char *buf, size_t bufLength,
        const char *source, uint32_t flag, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINlsNameMap", dpiOciSymbols.fnNlsNameMap)
    status = (*dpiOciSymbols.fnNlsNameMap)(envHandle, buf, bufLength, source,
            flag);
    return (status == DPI_OCI_SUCCESS) ? DPI_SUCCESS : DPI_FAILURE;
}


//-----------------------------------------------------------------------------
// dpiOci__nlsNumericInfoGet() [INTERNAL]
//   Wrapper for OCINlsNumericInfoGet().
//-----------------------------------------------------------------------------
int dpiOci__nlsNumericInfoGet(void *envHandle, int32_t *value, uint16_t item,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINlsNumericInfoGet",
            dpiOciSymbols.fnNlsNumericInfoGet)
    status = (*dpiOciSymbols.fnNlsNumericInfoGet)(envHandle, error->handle,
            value, item);
    return dpiError__check(error, status, NULL, "get NLS info");
}


//-----------------------------------------------------------------------------
// dpiOci__numberFromInt() [INTERNAL]
//   Wrapper for OCINumberFromInt().
//-----------------------------------------------------------------------------
int dpiOci__numberFromInt(const void *value, unsigned int valueLength,
        unsigned int flags, void *number, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINumberFromInt", dpiOciSymbols.fnNumberFromInt)
    status = (*dpiOciSymbols.fnNumberFromInt)(error->handle, value,
            valueLength, flags, number);
    return dpiError__check(error, status, NULL, "number from integer");
}


//-----------------------------------------------------------------------------
// dpiOci__numberFromReal() [INTERNAL]
//   Wrapper for OCINumberFromReal().
//-----------------------------------------------------------------------------
int dpiOci__numberFromReal(const double value, void *number, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINumberFromReal", dpiOciSymbols.fnNumberFromReal)
    status = (*dpiOciSymbols.fnNumberFromReal)(error->handle, &value,
            sizeof(double), number);
    return dpiError__check(error, status, NULL, "number from real");
}


//-----------------------------------------------------------------------------
// dpiOci__numberToInt() [INTERNAL]
//   Wrapper for OCINumberToInt().
//-----------------------------------------------------------------------------
int dpiOci__numberToInt(void *number, void *value, unsigned int valueLength,
        unsigned int flags, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINumberToInt", dpiOciSymbols.fnNumberToInt)
    status = (*dpiOciSymbols.fnNumberToInt)(error->handle, number, valueLength,
            flags, value);
    return dpiError__check(error, status, NULL, "number to integer");
}


//-----------------------------------------------------------------------------
// dpiOci__numberToReal() [INTERNAL]
//   Wrapper for OCINumberToReal().
//-----------------------------------------------------------------------------
int dpiOci__numberToReal(double *value, void *number, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCINumberToReal", dpiOciSymbols.fnNumberToReal)
    status = (*dpiOciSymbols.fnNumberToReal)(error->handle, number,
            sizeof(double), value);
    return dpiError__check(error, status, NULL, "number to real");
}


//-----------------------------------------------------------------------------
// dpiOci__objectCopy() [INTERNAL]
//   Wrapper for OCIObjectCopy().
//-----------------------------------------------------------------------------
int dpiOci__objectCopy(dpiObject *obj, void *sourceInstance,
        void *sourceIndicator, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectCopy", dpiOciSymbols.fnObjectCopy)
    status = (*dpiOciSymbols.fnObjectCopy)(obj->env->handle, error->handle,
            obj->type->conn->handle, sourceInstance, sourceIndicator,
            obj->instance, obj->indicator, obj->type->tdo,
            DPI_OCI_DURATION_SESSION, DPI_OCI_DEFAULT);
    return dpiError__check(error, status, obj->type->conn, "copy object");
}


//-----------------------------------------------------------------------------
// dpiOci__objectFree() [INTERNAL]
//   Wrapper for OCIObjectFree().
//-----------------------------------------------------------------------------
int dpiOci__objectFree(dpiObject *obj, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectFree", dpiOciSymbols.fnObjectFree)
    status = (*dpiOciSymbols.fnObjectFree)(obj->env->handle, error->handle,
            obj->instance, DPI_OCI_DEFAULT);
    if (checkError && dpiError__check(error, status, obj->type->conn,
            "free instance") < 0) {
        // during the attempt to free, PL/SQL records fail with error
        // "ORA-21602: operation does not support the specified typecode", but
        // a subsequent attempt will yield error "OCI-21500: internal error
        // code" and crash the process, so pretend like the free was
        // successful!
        if (error->buffer->code == 21602)
            return DPI_SUCCESS;
        return DPI_FAILURE;
    }
    if (obj->freeIndicator) {
        status = (*dpiOciSymbols.fnObjectFree)(obj->env->handle, error->handle,
                obj->indicator, DPI_OCI_DEFAULT);
        if (checkError && dpiError__check(error, status, obj->type->conn,
                "free indicator") < 0)
            return DPI_FAILURE;
    }
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__objectGetAttr() [INTERNAL]
//   Wrapper for OCIObjectGetAttr().
//-----------------------------------------------------------------------------
int dpiOci__objectGetAttr(dpiObject *obj, dpiObjectAttr *attr,
        int16_t *scalarValueIndicator, void **valueIndicator, void **value,
        void **tdo, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectGetAttr", dpiOciSymbols.fnObjectGetAttr)
    status = (*dpiOciSymbols.fnObjectGetAttr)(obj->env->handle, error->handle,
            obj->instance, obj->indicator, obj->type->tdo, &attr->name,
            &attr->nameLength, 1, 0, 0, scalarValueIndicator, valueIndicator,
            value, tdo);
    return dpiError__check(error, status, obj->type->conn, "get attribute");
}


//-----------------------------------------------------------------------------
// dpiOci__objectGetInd() [INTERNAL]
//   Wrapper for OCIObjectGetInd().
//-----------------------------------------------------------------------------
int dpiOci__objectGetInd(dpiObject *obj, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectGetInd", dpiOciSymbols.fnObjectGetInd)
    status = (*dpiOciSymbols.fnObjectGetInd)(obj->env->handle, error->handle,
            obj->instance, &obj->indicator);
    return dpiError__check(error, status, obj->type->conn, "get indicator");
}


//-----------------------------------------------------------------------------
// dpiOci__objectNew() [INTERNAL]
//   Wrapper for OCIObjectNew().
//-----------------------------------------------------------------------------
int dpiOci__objectNew(dpiObject *obj, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectNew", dpiOciSymbols.fnObjectNew)
    status = (*dpiOciSymbols.fnObjectNew)(obj->env->handle, error->handle,
            obj->type->conn->handle, obj->type->typeCode, obj->type->tdo, NULL,
            DPI_OCI_DURATION_SESSION, 1, &obj->instance);
    return dpiError__check(error, status, obj->type->conn, "create object");
}


//-----------------------------------------------------------------------------
// dpiOci__objectPin() [INTERNAL]
//   Wrapper for OCIObjectPin().
//-----------------------------------------------------------------------------
int dpiOci__objectPin(void *envHandle, void *objRef, void **obj,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectPin", dpiOciSymbols.fnObjectPin)
    status = (*dpiOciSymbols.fnObjectPin)(envHandle, error->handle, objRef,
            NULL, DPI_OCI_PIN_ANY, DPI_OCI_DURATION_SESSION, DPI_OCI_LOCK_NONE,
            obj);
    return dpiError__check(error, status, NULL, "pin reference");
}


//-----------------------------------------------------------------------------
// dpiOci__objectSetAttr() [INTERNAL]
//   Wrapper for OCIObjectSetAttr().
//-----------------------------------------------------------------------------
int dpiOci__objectSetAttr(dpiObject *obj, dpiObjectAttr *attr,
        int16_t scalarValueIndicator, void *valueIndicator, const void *value,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIObjectSetAttr", dpiOciSymbols.fnObjectSetAttr)
    status = (*dpiOciSymbols.fnObjectSetAttr)(obj->env->handle, error->handle,
            obj->instance, obj->indicator, obj->type->tdo, &attr->name,
            &attr->nameLength, 1, NULL, 0, scalarValueIndicator,
            valueIndicator, value);
    return dpiError__check(error, status, obj->type->conn, "set attribute");
}


//-----------------------------------------------------------------------------
// dpiOci__passwordChange() [INTERNAL]
//   Wrapper for OCIPasswordChange().
//-----------------------------------------------------------------------------
int dpiOci__passwordChange(dpiConn *conn, const char *userName,
        uint32_t userNameLength, const char *oldPassword,
        uint32_t oldPasswordLength, const char *newPassword,
        uint32_t newPasswordLength, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIPasswordChange", dpiOciSymbols.fnPasswordChange)
    status = (*dpiOciSymbols.fnPasswordChange)(conn->handle, error->handle,
            userName, userNameLength, oldPassword, oldPasswordLength,
            newPassword, newPasswordLength, mode);
    return dpiError__check(error, status, conn, "change password");
}


//-----------------------------------------------------------------------------
// dpiOci__paramGet() [INTERNAL]
//   Wrapper for OCIParamGet().
//-----------------------------------------------------------------------------
int dpiOci__paramGet(const void *handle, uint32_t handleType, void **parameter,
        uint32_t pos, const char *action, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIParamGet", dpiOciSymbols.fnParamGet)
    status = (*dpiOciSymbols.fnParamGet)(handle, handleType, error->handle,
            parameter, pos);
    return dpiError__check(error, status, NULL, action);
}


//-----------------------------------------------------------------------------
// dpiOci__ping() [INTERNAL]
//   Wrapper for OCIPing().
//-----------------------------------------------------------------------------
int dpiOci__ping(dpiConn *conn, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIPing", dpiOciSymbols.fnPing)
    status = (*dpiOciSymbols.fnPing)(conn->handle, error->handle,
            DPI_OCI_DEFAULT);
    status = dpiError__check(error, status, conn, "ping");

    // attempting to ping a database earlier than 10g will result in error
    // ORA-1010: invalid OCI operation, but that implies a successful ping
    // so ignore that error and treat it as a successful operation
    if (status < 0 && error->buffer->code == 1010)
        return DPI_SUCCESS;

    return status;
}


//-----------------------------------------------------------------------------
// dpiOci__rawAssignBytes() [INTERNAL]
//   Wrapper for OCIRawAssignBytes().
//-----------------------------------------------------------------------------
int dpiOci__rawAssignBytes(void *envHandle, const char *value,
        uint32_t valueLength, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIRawAssignBytes", dpiOciSymbols.fnRawAssignBytes)
    status = (*dpiOciSymbols.fnRawAssignBytes)(envHandle, error->handle, value,
            valueLength, handle);
    return dpiError__check(error, status, NULL, "assign bytes to raw");
}


//-----------------------------------------------------------------------------
// dpiOci__rawPtr() [INTERNAL]
//   Wrapper for OCIRawPtr().
//-----------------------------------------------------------------------------
int dpiOci__rawPtr(void *envHandle, void *handle, void **ptr)
{
    dpiError *error = NULL;

    DPI_OCI_LOAD_SYMBOL("OCIRawPtr", dpiOciSymbols.fnRawPtr)
    *ptr = (*dpiOciSymbols.fnRawPtr)(envHandle, handle);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__rawResize() [INTERNAL]
//   Wrapper for OCIRawResize().
//-----------------------------------------------------------------------------
int dpiOci__rawResize(void *envHandle, void **handle, uint32_t newSize,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIRawResize", dpiOciSymbols.fnRawResize)
    status = (*dpiOciSymbols.fnRawResize)(envHandle, error->handle, newSize,
            handle);
    return dpiError__check(error, status, NULL, "resize raw");
}


//-----------------------------------------------------------------------------
// dpiOci__rawSize() [INTERNAL]
//   Wrapper for OCIRawSize().
//-----------------------------------------------------------------------------
int dpiOci__rawSize(void *envHandle, void *handle, uint32_t *size)
{
    dpiError *error = NULL;

    DPI_OCI_LOAD_SYMBOL("OCIRawSize", dpiOciSymbols.fnRawSize)
    *size = (*dpiOciSymbols.fnRawSize)(envHandle, handle);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__reallocMem() [INTERNAL]
//   Wrapper for OCI allocation of memory, only used when debugging memory
// allocation.
//-----------------------------------------------------------------------------
static void *dpiOci__reallocMem(void *unused, void *ptr, size_t newSize)
{
    char message[80];
    void *newPtr;

    (void) sprintf(message, "OCI reallocated ptr at %p", ptr);
    newPtr = realloc(ptr, newSize);
    dpiDebug__print("%s to %u bytes at %p\n", message, newSize, newPtr);
    return newPtr;
}


//-----------------------------------------------------------------------------
// dpiOci__rowidToChar() [INTERNAL]
//   Wrapper for OCIRowidToChar().
//-----------------------------------------------------------------------------
int dpiOci__rowidToChar(dpiRowid *rowid, char *buffer, uint16_t *bufferSize,
        dpiError *error)
{
    uint16_t origSize;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIRowidToChar", dpiOciSymbols.fnRowidToChar)
    origSize = *bufferSize;
    status = (*dpiOciSymbols.fnRowidToChar)(rowid->handle, buffer, bufferSize,
            error->handle);
    if (origSize == 0)
        return DPI_SUCCESS;
    return dpiError__check(error, status, NULL, "get rowid as string");
}


//-----------------------------------------------------------------------------
// dpiOci__serverAttach() [INTERNAL]
//   Wrapper for OCIServerAttach().
//-----------------------------------------------------------------------------
int dpiOci__serverAttach(dpiConn *conn, const char *connectString,
        uint32_t connectStringLength, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIServerAttach", dpiOciSymbols.fnServerAttach)
    status = (*dpiOciSymbols.fnServerAttach)(conn->serverHandle, error->handle,
            connectString, (int32_t) connectStringLength, DPI_OCI_DEFAULT);
    return dpiError__check(error, status, conn, "server attach");
}


//-----------------------------------------------------------------------------
// dpiOci__serverDetach() [INTERNAL]
//   Wrapper for OCIServerDetach().
//-----------------------------------------------------------------------------
int dpiOci__serverDetach(dpiConn *conn, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIServerDetach", dpiOciSymbols.fnServerDetach)
    status = (*dpiOciSymbols.fnServerDetach)(conn->serverHandle, error->handle,
            DPI_OCI_DEFAULT);
    if (checkError)
        return dpiError__check(error, status, conn, "detatch from server");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__serverRelease() [INTERNAL]
//   Wrapper for OCIServerRelease().
//-----------------------------------------------------------------------------
int dpiOci__serverRelease(dpiConn *conn, char *buffer, uint32_t bufferSize,
        uint32_t *version, dpiError *error)
{
    int status;

    if (conn->env->versionInfo->versionNum < 18) {
        DPI_OCI_LOAD_SYMBOL("OCIServerRelease", dpiOciSymbols.fnServerRelease)
        status = (*dpiOciSymbols.fnServerRelease)(conn->handle, error->handle,
                buffer, bufferSize, DPI_OCI_HTYPE_SVCCTX, version);
    } else {
        DPI_OCI_LOAD_SYMBOL("OCIServerRelease2",
                dpiOciSymbols.fnServerRelease2)
        status = (*dpiOciSymbols.fnServerRelease2)(conn->handle, error->handle,
                buffer, bufferSize, DPI_OCI_HTYPE_SVCCTX, version,
                DPI_OCI_DEFAULT);
    }
    return dpiError__check(error, status, conn, "get server version");
}


//-----------------------------------------------------------------------------
// dpiOci__sessionBegin() [INTERNAL]
//   Wrapper for OCISessionBegin().
//-----------------------------------------------------------------------------
int dpiOci__sessionBegin(dpiConn *conn, uint32_t credentialType,
        uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionBegin", dpiOciSymbols.fnSessionBegin)
    status = (*dpiOciSymbols.fnSessionBegin)(conn->handle, error->handle,
            conn->sessionHandle, credentialType, mode);
    return dpiError__check(error, status, conn, "begin session");
}


//-----------------------------------------------------------------------------
// dpiOci__sessionEnd() [INTERNAL]
//   Wrapper for OCISessionEnd().
//-----------------------------------------------------------------------------
int dpiOci__sessionEnd(dpiConn *conn, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionEnd", dpiOciSymbols.fnSessionEnd)
    status = (*dpiOciSymbols.fnSessionEnd)(conn->handle, error->handle,
            conn->sessionHandle, DPI_OCI_DEFAULT);
    if (checkError)
        return dpiError__check(error, status, conn, "end session");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__sessionGet() [INTERNAL]
//   Wrapper for OCISessionGet().
//-----------------------------------------------------------------------------
int dpiOci__sessionGet(void *envHandle, void **handle, void *authInfo,
        const char *connectString, uint32_t connectStringLength,
        const char *tag, uint32_t tagLength, const char **outTag,
        uint32_t *outTagLength, int *found, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionGet", dpiOciSymbols.fnSessionGet)
    status = (*dpiOciSymbols.fnSessionGet)(envHandle, error->handle, handle,
            authInfo, connectString, connectStringLength, tag, tagLength,
            outTag, outTagLength, found, mode);
    return dpiError__check(error, status, NULL, "get session");
}


//-----------------------------------------------------------------------------
// dpiOci__sessionPoolCreate() [INTERNAL]
//   Wrapper for OCISessionPoolCreate().
//-----------------------------------------------------------------------------
int dpiOci__sessionPoolCreate(dpiPool *pool, const char *connectString,
        uint32_t connectStringLength, uint32_t minSessions,
        uint32_t maxSessions, uint32_t sessionIncrement, const char *userName,
        uint32_t userNameLength, const char *password, uint32_t passwordLength,
        uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionPoolCreate",
            dpiOciSymbols.fnSessionPoolCreate)
    status = (*dpiOciSymbols.fnSessionPoolCreate)(pool->env->handle,
            error->handle, pool->handle, (char**) &pool->name,
            &pool->nameLength, connectString, connectStringLength, minSessions,
            maxSessions, sessionIncrement, userName, userNameLength, password,
            passwordLength, mode);
    return dpiError__check(error, status, NULL, "create pool");
}


//-----------------------------------------------------------------------------
// dpiOci__sessionPoolDestroy() [INTERNAL]
//   Wrapper for OCISessionPoolDestroy().
//-----------------------------------------------------------------------------
int dpiOci__sessionPoolDestroy(dpiPool *pool, uint32_t mode, int checkError,
        dpiError *error)
{
    void *handle;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionPoolDestroy",
            dpiOciSymbols.fnSessionPoolDestroy)

    // clear the pool handle immediately so that no further attempts are made
    // to use the pool while the pool is being closed; if the pool close fails,
    // restore the pool handle afterwards
    handle = pool->handle;
    pool->handle = NULL;
    status = (*dpiOciSymbols.fnSessionPoolDestroy)(handle, error->handle,
            mode);
    if (checkError &&
            dpiError__check(error, status, NULL, "destroy pool") < 0) {
        pool->handle = handle;
        return DPI_FAILURE;
    }
    dpiOci__handleFree(handle, DPI_OCI_HTYPE_SPOOL);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__sessionRelease() [INTERNAL]
//   Wrapper for OCISessionRelease().
//-----------------------------------------------------------------------------
int dpiOci__sessionRelease(dpiConn *conn, const char *tag, uint32_t tagLength,
        uint32_t mode, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISessionRelease", dpiOciSymbols.fnSessionRelease)
    status = (*dpiOciSymbols.fnSessionRelease)(conn->handle, error->handle,
            tag, tagLength, mode);
    if (checkError)
        return dpiError__check(error, status, conn, "release session");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__shardingKeyColumnAdd() [INTERNAL]
//   Wrapper for OCIshardingKeyColumnAdd().
//-----------------------------------------------------------------------------
int dpiOci__shardingKeyColumnAdd(void *shardingKey, void *col, uint32_t colLen,
        uint16_t colType, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIShardingKeyColumnAdd",
            dpiOciSymbols.fnShardingKeyColumnAdd)
    status = (*dpiOciSymbols.fnShardingKeyColumnAdd)(shardingKey,
            error->handle, col, colLen, colType, DPI_OCI_DEFAULT);
    return dpiError__check(error, status, NULL, "add sharding column");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaCollCreateWithMetadata() [INTERNAL]
//   Wrapper for OCISodaCollCreateWithMetadata().
//-----------------------------------------------------------------------------
int dpiOci__sodaCollCreateWithMetadata(dpiSodaDb *db, const char *name,
        uint32_t nameLength, const char *metadata, uint32_t metadataLength,
        uint32_t mode, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaCollCreateWithMetadata",
            dpiOciSymbols.fnSodaCollCreateWithMetadata)
    status = (*dpiOciSymbols.fnSodaCollCreateWithMetadata)(db->conn->handle,
            name, nameLength, metadata, metadataLength, handle, error->handle,
            mode);
    return dpiError__check(error, status, db->conn, "create SODA collection");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaCollDrop() [INTERNAL]
//   Wrapper for OCISodaCollDrop().
//-----------------------------------------------------------------------------
int dpiOci__sodaCollDrop(dpiSodaColl *coll, int *isDropped, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaCollDrop", dpiOciSymbols.fnSodaCollDrop)
    status = (*dpiOciSymbols.fnSodaCollDrop)(coll->db->conn->handle,
            coll->handle, isDropped, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "drop SODA collection");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaCollGetNext() [INTERNAL]
//   Wrapper for OCISodaCollGetNext().
//-----------------------------------------------------------------------------
int dpiOci__sodaCollGetNext(dpiConn *conn, void *cursorHandle,
        void **collectionHandle, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaCollGetNext", dpiOciSymbols.fnSodaCollGetNext)
    status = (*dpiOciSymbols.fnSodaCollGetNext)(conn->handle, cursorHandle,
            collectionHandle, error->handle, mode);
    if (status == DPI_OCI_NO_DATA) {
        *collectionHandle = NULL;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, conn, "get next collection");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaCollList() [INTERNAL]
//   Wrapper for OCISodaCollList().
//-----------------------------------------------------------------------------
int dpiOci__sodaCollList(dpiSodaDb *db, const char *startingName,
        uint32_t startingNameLength, void **handle, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaCollList", dpiOciSymbols.fnSodaCollList)
    status = (*dpiOciSymbols.fnSodaCollList)(db->conn->handle, startingName,
            startingNameLength, handle, error->handle, mode);
    return dpiError__check(error, status, db->conn,
            "get SODA collection cursor");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaCollOpen() [INTERNAL]
//   Wrapper for OCISodaCollOpen().
//-----------------------------------------------------------------------------
int dpiOci__sodaCollOpen(dpiSodaDb *db, const char *name, uint32_t nameLength,
        uint32_t mode, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaCollOpen", dpiOciSymbols.fnSodaCollOpen)
    status = (*dpiOciSymbols.fnSodaCollOpen)(db->conn->handle, name,
            nameLength, handle, error->handle, mode);
    return dpiError__check(error, status, db->conn, "open SODA collection");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaDataGuideGet() [INTERNAL]
//   Wrapper for OCISodaDataGuideGet().
//-----------------------------------------------------------------------------
int dpiOci__sodaDataGuideGet(dpiSodaColl *coll, void **handle, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaDataGuideGet",
            dpiOciSymbols.fnSodaDataGuideGet)
    status = (*dpiOciSymbols.fnSodaDataGuideGet)(coll->db->conn->handle,
            coll->handle, DPI_OCI_DEFAULT, handle, error->handle, mode);
    if (dpiError__check(error, status, coll->db->conn, "get data guide") < 0) {
        if (error->buffer->code != 24801)
            return DPI_FAILURE;
        *handle = NULL;
    }
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__sodaDocCount() [INTERNAL]
//   Wrapper for OCISodaDocCount().
//-----------------------------------------------------------------------------
int dpiOci__sodaDocCount(dpiSodaColl *coll, void *options, uint32_t mode,
        uint64_t *count, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaDocCount", dpiOciSymbols.fnSodaDocCount)
    status = (*dpiOciSymbols.fnSodaDocCount)(coll->db->conn->handle,
            coll->handle, options, count, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "get document count");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaDocGetNext() [INTERNAL]
//   Wrapper for OCISodaDocGetNext().
//-----------------------------------------------------------------------------
int dpiOci__sodaDocGetNext(dpiSodaDocCursor *cursor, void **handle,
        uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaDocGetNext", dpiOciSymbols.fnSodaDocGetNext)
    status = (*dpiOciSymbols.fnSodaDocGetNext)(cursor->coll->db->conn->handle,
            cursor->handle, handle, error->handle, mode);
    if (status == DPI_OCI_NO_DATA) {
        *handle = NULL;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, cursor->coll->db->conn,
            "get next document");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaFind() [INTERNAL]
//   Wrapper for OCISodaFind().
//-----------------------------------------------------------------------------
int dpiOci__sodaFind(dpiSodaColl *coll, const void *options, uint32_t flags,
        uint32_t mode, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaFind", dpiOciSymbols.fnSodaFind)
    status = (*dpiOciSymbols.fnSodaFind)(coll->db->conn->handle,
            coll->handle, options, flags, handle, error->handle, mode);
    if (status == DPI_OCI_NO_DATA) {
        *handle = NULL;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, coll->db->conn,
            "find SODA documents");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaFindOne() [INTERNAL]
//   Wrapper for OCISodaFindOne().
//-----------------------------------------------------------------------------
int dpiOci__sodaFindOne(dpiSodaColl *coll, const void *options, uint32_t flags,
        uint32_t mode, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaFindOne", dpiOciSymbols.fnSodaFindOne)
    status = (*dpiOciSymbols.fnSodaFindOne)(coll->db->conn->handle,
            coll->handle, options, flags, handle, error->handle, mode);
    if (status == DPI_OCI_NO_DATA) {
        *handle = NULL;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, coll->db->conn, "get SODA document");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaIndexCreate() [INTERNAL]
//   Wrapper for OCISodaIndexCreate().
//-----------------------------------------------------------------------------
int dpiOci__sodaIndexCreate(dpiSodaColl *coll, const char *indexSpec,
        uint32_t indexSpecLength, uint32_t mode, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaIndexCreate", dpiOciSymbols.fnSodaIndexCreate)
    status = (*dpiOciSymbols.fnSodaIndexCreate)(coll->db->conn->handle,
            coll->handle, indexSpec, indexSpecLength, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn, "create index");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaIndexDrop() [INTERNAL]
//   Wrapper for OCISodaIndexDrop().
//-----------------------------------------------------------------------------
int dpiOci__sodaIndexDrop(dpiSodaColl *coll, const char *name,
        uint32_t nameLength, uint32_t mode, int *isDropped, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaIndexDrop", dpiOciSymbols.fnSodaIndexDrop)
    status = (*dpiOciSymbols.fnSodaIndexDrop)(coll->db->conn->handle, name,
            nameLength, isDropped, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn, "drop index");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaInsert() [INTERNAL]
//   Wrapper for OCISodaInsert().
//-----------------------------------------------------------------------------
int dpiOci__sodaInsert(dpiSodaColl *coll, void *handle, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaInsert", dpiOciSymbols.fnSodaInsert)
    status = (*dpiOciSymbols.fnSodaInsert)(coll->db->conn->handle,
            coll->handle, handle, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "insert SODA document");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaInsertAndGet() [INTERNAL]
//   Wrapper for OCISodaInsertAndGet().
//-----------------------------------------------------------------------------
int dpiOci__sodaInsertAndGet(dpiSodaColl *coll, void **handle, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaInsertAndGet",
            dpiOciSymbols.fnSodaInsertAndGet)
    status = (*dpiOciSymbols.fnSodaInsertAndGet)(coll->db->conn->handle,
            coll->handle, handle, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "insert and get SODA document");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaOperKeysSet() [INTERNAL]
//   Wrapper for OCISodaOperKeysSet().
//-----------------------------------------------------------------------------
int dpiOci__sodaOperKeysSet(const dpiSodaOperOptions *options, void *handle,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaOperKeysSet", dpiOciSymbols.fnSodaOperKeysSet)
    status = (*dpiOciSymbols.fnSodaOperKeysSet)(handle, options->keys,
            options->keyLengths, options->numKeys, error->handle,
            DPI_OCI_DEFAULT);
    return dpiError__check(error, status, NULL, "set operation options keys");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaRemove() [INTERNAL]
//   Wrapper for OCISodaRemove().
//-----------------------------------------------------------------------------
int dpiOci__sodaRemove(dpiSodaColl *coll, void *options, uint32_t mode,
        uint64_t *count, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaRemove", dpiOciSymbols.fnSodaRemove)
    status = (*dpiOciSymbols.fnSodaRemove)(coll->db->conn->handle,
            coll->handle, options, count, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "remove documents from SODA collection");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaReplOne() [INTERNAL]
//   Wrapper for OCISodaReplOne().
//-----------------------------------------------------------------------------
int dpiOci__sodaReplOne(dpiSodaColl *coll, const void *options, void *handle,
        uint32_t mode, int *isReplaced, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaReplOne", dpiOciSymbols.fnSodaReplOne)
    status = (*dpiOciSymbols.fnSodaReplOne)(coll->db->conn->handle,
            coll->handle, options, handle, isReplaced, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "replace SODA document");
}


//-----------------------------------------------------------------------------
// dpiOci__sodaReplOneAndGet() [INTERNAL]
//   Wrapper for OCISodaReplOneAndGet().
//-----------------------------------------------------------------------------
int dpiOci__sodaReplOneAndGet(dpiSodaColl *coll, const void *options,
        void **handle, uint32_t mode, int *isReplaced, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISodaReplOneAndGet",
            dpiOciSymbols.fnSodaReplOneAndGet)
    status = (*dpiOciSymbols.fnSodaReplOneAndGet)(coll->db->conn->handle,
            coll->handle, options, handle, isReplaced, error->handle, mode);
    return dpiError__check(error, status, coll->db->conn,
            "replace and get SODA document");
}


//-----------------------------------------------------------------------------
// dpiOci__stmtExecute() [INTERNAL]
//   Wrapper for OCIStmtExecute().
//-----------------------------------------------------------------------------
int dpiOci__stmtExecute(dpiStmt *stmt, uint32_t numIters, uint32_t mode,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStmtExecute", dpiOciSymbols.fnStmtExecute)
    status = (*dpiOciSymbols.fnStmtExecute)(stmt->conn->handle, stmt->handle,
            error->handle, numIters, 0, 0, 0, mode);
    return dpiError__check(error, status, stmt->conn, "execute");
}


//-----------------------------------------------------------------------------
// dpiOci__stmtFetch2() [INTERNAL]
//   Wrapper for OCIStmtFetch2().
//-----------------------------------------------------------------------------
int dpiOci__stmtFetch2(dpiStmt *stmt, uint32_t numRows, uint16_t fetchMode,
        int32_t offset, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStmtFetch2", dpiOciSymbols.fnStmtFetch2)
    status = (*dpiOciSymbols.fnStmtFetch2)(stmt->handle, error->handle,
            numRows, fetchMode, offset, DPI_OCI_DEFAULT);
    if (status == DPI_OCI_NO_DATA || fetchMode == DPI_MODE_FETCH_LAST)
        stmt->hasRowsToFetch = 0;
    else if (dpiError__check(error, status, stmt->conn, "fetch") < 0)
        return DPI_FAILURE;
    else stmt->hasRowsToFetch = 1;
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__stmtGetBindInfo() [INTERNAL]
//   Wrapper for OCIStmtGetBindInfo().
//-----------------------------------------------------------------------------
int dpiOci__stmtGetBindInfo(dpiStmt *stmt, uint32_t size, uint32_t startLoc,
        int32_t *numFound, char *names[], uint8_t nameLengths[],
        char *indNames[], uint8_t indNameLengths[], uint8_t isDuplicate[],
        void *bindHandles[], dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStmtGetBindInfo", dpiOciSymbols.fnStmtGetBindInfo)
    status = (*dpiOciSymbols.fnStmtGetBindInfo)(stmt->handle, error->handle,
            size, startLoc, numFound, names, nameLengths, indNames,
            indNameLengths, isDuplicate, bindHandles);
    if (status == DPI_OCI_NO_DATA) {
        *numFound = 0;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, stmt->conn, "get bind info");
}


//-----------------------------------------------------------------------------
// dpiOci__stmtGetNextResult() [INTERNAL]
//   Wrapper for OCIStmtGetNextResult().
//-----------------------------------------------------------------------------
int dpiOci__stmtGetNextResult(dpiStmt *stmt, void **handle, dpiError *error)
{
    uint32_t returnType;
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStmtGetNextResult",
            dpiOciSymbols.fnStmtGetNextResult)
    status = (*dpiOciSymbols.fnStmtGetNextResult)(stmt->handle, error->handle,
            handle, &returnType, DPI_OCI_DEFAULT);
    if (status == DPI_OCI_NO_DATA) {
        *handle = NULL;
        return DPI_SUCCESS;
    }
    return dpiError__check(error, status, stmt->conn, "get next result");
}


//-----------------------------------------------------------------------------
// dpiOci__stmtPrepare2() [INTERNAL]
//   Wrapper for OCIStmtPrepare2().
//-----------------------------------------------------------------------------
int dpiOci__stmtPrepare2(dpiStmt *stmt, const char *sql, uint32_t sqlLength,
        const char *tag, uint32_t tagLength, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStmtPrepare2", dpiOciSymbols.fnStmtPrepare2)
    status = (*dpiOciSymbols.fnStmtPrepare2)(stmt->conn->handle, &stmt->handle,
            error->handle, sql, sqlLength, tag, tagLength, DPI_OCI_NTV_SYNTAX,
            DPI_OCI_DEFAULT);
    if (dpiError__check(error, status, stmt->conn, "prepare SQL") < 0) {
        stmt->handle = NULL;
        return DPI_FAILURE;
    }

    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__stmtRelease() [INTERNAL]
//   Wrapper for OCIStmtRelease().
//-----------------------------------------------------------------------------
int dpiOci__stmtRelease(dpiStmt *stmt, const char *tag, uint32_t tagLength,
        int checkError, dpiError *error)
{
    uint32_t mode = DPI_OCI_DEFAULT;
    uint32_t cacheSize = 0;
    int status;

    // if the statement should be deleted from the cache, first check to see
    // that there actually is a cache currently being used; otherwise, the
    // error "ORA-24300: bad value for mode" will be raised
    if (stmt->deleteFromCache) {
        dpiOci__attrGet(stmt->conn->handle, DPI_OCI_HTYPE_SVCCTX,
                &cacheSize, NULL, DPI_OCI_ATTR_STMTCACHESIZE, NULL, error);
        if (cacheSize > 0)
            mode = DPI_OCI_STRLS_CACHE_DELETE;
    }

    DPI_OCI_LOAD_SYMBOL("OCIStmtRelease", dpiOciSymbols.fnStmtRelease)
    status = (*dpiOciSymbols.fnStmtRelease)(stmt->handle, error->handle, tag,
            tagLength, mode);
    if (checkError)
        return dpiError__check(error, status, stmt->conn, "release statement");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__stringAssignText() [INTERNAL]
//   Wrapper for OCIStringAssignText().
//-----------------------------------------------------------------------------
int dpiOci__stringAssignText(void *envHandle, const char *value,
        uint32_t valueLength, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStringAssignText",
            dpiOciSymbols.fnStringAssignText)
    status = (*dpiOciSymbols.fnStringAssignText)(envHandle, error->handle,
            value, valueLength, handle);
    return dpiError__check(error, status, NULL, "assign to string");
}


//-----------------------------------------------------------------------------
// dpiOci__stringPtr() [INTERNAL]
//   Wrapper for OCIStringPtr().
//-----------------------------------------------------------------------------
int dpiOci__stringPtr(void *envHandle, void *handle, char **ptr)
{
    dpiError *error = NULL;

    DPI_OCI_LOAD_SYMBOL("OCIStringPtr", dpiOciSymbols.fnStringPtr)
    *ptr = (*dpiOciSymbols.fnStringPtr)(envHandle, handle);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__stringResize() [INTERNAL]
//   Wrapper for OCIStringResize().
//-----------------------------------------------------------------------------
int dpiOci__stringResize(void *envHandle, void **handle, uint32_t newSize,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIStringResize", dpiOciSymbols.fnStringResize)
    status = (*dpiOciSymbols.fnStringResize)(envHandle, error->handle, newSize,
            handle);
    return dpiError__check(error, status, NULL, "resize string");
}


//-----------------------------------------------------------------------------
// dpiOci__stringSize() [INTERNAL]
//   Wrapper for OCIStringSize().
//-----------------------------------------------------------------------------
int dpiOci__stringSize(void *envHandle, void *handle, uint32_t *size)
{
    dpiError *error = NULL;

    DPI_OCI_LOAD_SYMBOL("OCIStringSize", dpiOciSymbols.fnStringSize)
    *size = (*dpiOciSymbols.fnStringSize)(envHandle, handle);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__subscriptionRegister() [INTERNAL]
//   Wrapper for OCISubscriptionRegister().
//-----------------------------------------------------------------------------
int dpiOci__subscriptionRegister(dpiConn *conn, void **handle, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISubscriptionRegister",
            dpiOciSymbols.fnSubscriptionRegister)
    status = (*dpiOciSymbols.fnSubscriptionRegister)(conn->handle, handle, 1,
            error->handle, DPI_OCI_DEFAULT);
    return dpiError__check(error, status, conn, "register");
}


//-----------------------------------------------------------------------------
// dpiOci__subscriptionUnRegister() [INTERNAL]
//   Wrapper for OCISubscriptionUnRegister().
//-----------------------------------------------------------------------------
int dpiOci__subscriptionUnRegister(dpiConn *conn, dpiSubscr *subscr,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCISubscriptionUnRegister",
            dpiOciSymbols.fnSubscriptionUnRegister)
    status = (*dpiOciSymbols.fnSubscriptionUnRegister)(conn->handle,
            subscr->handle, error->handle, DPI_OCI_DEFAULT);
    return dpiError__check(error, status, conn, "unregister");
}


//-----------------------------------------------------------------------------
// dpiOci__tableDelete() [INTERNAL]
//   Wrapper for OCITableDelete().
//-----------------------------------------------------------------------------
int dpiOci__tableDelete(dpiObject *obj, int32_t index, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableDelete", dpiOciSymbols.fnTableDelete)
    status = (*dpiOciSymbols.fnTableDelete)(obj->env->handle, error->handle,
            index, obj->instance);
    return dpiError__check(error, status, obj->type->conn, "delete element");
}


//-----------------------------------------------------------------------------
// dpiOci__tableExists() [INTERNAL]
//   Wrapper for OCITableExists().
//-----------------------------------------------------------------------------
int dpiOci__tableExists(dpiObject *obj, int32_t index, int *exists,
        dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableExists", dpiOciSymbols.fnTableExists)
    status = (*dpiOciSymbols.fnTableExists)(obj->env->handle, error->handle,
            obj->instance, index, exists);
    return dpiError__check(error, status, obj->type->conn, "get index exists");
}


//-----------------------------------------------------------------------------
// dpiOci__tableFirst() [INTERNAL]
//   Wrapper for OCITableFirst().
//-----------------------------------------------------------------------------
int dpiOci__tableFirst(dpiObject *obj, int32_t *index, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableFirst", dpiOciSymbols.fnTableFirst)
    status = (*dpiOciSymbols.fnTableFirst)(obj->env->handle, error->handle,
            obj->instance, index);
    return dpiError__check(error, status, obj->type->conn, "get first index");
}


//-----------------------------------------------------------------------------
// dpiOci__tableLast() [INTERNAL]
//   Wrapper for OCITableLast().
//-----------------------------------------------------------------------------
int dpiOci__tableLast(dpiObject *obj, int32_t *index, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableLast", dpiOciSymbols.fnTableLast)
    status = (*dpiOciSymbols.fnTableLast)(obj->env->handle, error->handle,
            obj->instance, index);
    return dpiError__check(error, status, obj->type->conn, "get last index");
}


//-----------------------------------------------------------------------------
// dpiOci__tableNext() [INTERNAL]
//   Wrapper for OCITableNext().
//-----------------------------------------------------------------------------
int dpiOci__tableNext(dpiObject *obj, int32_t index, int32_t *nextIndex,
        int *exists, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableNext", dpiOciSymbols.fnTableNext)
    status = (*dpiOciSymbols.fnTableNext)(obj->env->handle, error->handle,
            index, obj->instance, nextIndex, exists);
    return dpiError__check(error, status, obj->type->conn, "get next index");
}


//-----------------------------------------------------------------------------
// dpiOci__tablePrev() [INTERNAL]
//   Wrapper for OCITablePrev().
//-----------------------------------------------------------------------------
int dpiOci__tablePrev(dpiObject *obj, int32_t index, int32_t *prevIndex,
        int *exists, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITablePrev", dpiOciSymbols.fnTablePrev)
    status = (*dpiOciSymbols.fnTablePrev)(obj->env->handle, error->handle,
            index, obj->instance, prevIndex, exists);
    return dpiError__check(error, status, obj->type->conn, "get prev index");
}


//-----------------------------------------------------------------------------
// dpiOci__tableSize() [INTERNAL]
//   Wrapper for OCITableSize().
//-----------------------------------------------------------------------------
int dpiOci__tableSize(dpiObject *obj, int32_t *size, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITableSize", dpiOciSymbols.fnTableSize)
    status = (*dpiOciSymbols.fnTableSize)(obj->env->handle, error->handle,
            obj->instance, size);
    return dpiError__check(error, status, obj->type->conn, "get size");
}


//-----------------------------------------------------------------------------
// dpiOci__threadKeyDestroy() [INTERNAL]
//   Wrapper for OCIThreadKeyDestroy().
//-----------------------------------------------------------------------------
int dpiOci__threadKeyDestroy(void *envHandle, void *errorHandle, void **key,
        dpiError *error)
{
    DPI_OCI_LOAD_SYMBOL("OCIThreadKeyDestroy",
            dpiOciSymbols.fnThreadKeyDestroy)
    (*dpiOciSymbols.fnThreadKeyDestroy)(envHandle, errorHandle, key);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__threadKeyGet() [INTERNAL]
//   Wrapper for OCIThreadKeyGet().
//-----------------------------------------------------------------------------
int dpiOci__threadKeyGet(void *envHandle, void *errorHandle, void *key,
        void **value, dpiError *error)
{
    int status;

    status = (*dpiOciSymbols.fnThreadKeyGet)(envHandle, errorHandle, key,
            value);
    if (status != DPI_OCI_SUCCESS)
        return dpiError__set(error, "get TLS error", DPI_ERR_TLS_ERROR);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__threadKeyInit() [INTERNAL]
//   Wrapper for OCIThreadKeyInit().
//-----------------------------------------------------------------------------
int dpiOci__threadKeyInit(void *envHandle, void *errorHandle, void **key,
        void *destroyFunc, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIThreadKeyInit", dpiOciSymbols.fnThreadKeyInit)
    status = (*dpiOciSymbols.fnThreadKeyInit)(envHandle, errorHandle, key,
            destroyFunc);
    return dpiError__check(error, status, NULL, "initialize thread key");
}


//-----------------------------------------------------------------------------
// dpiOci__threadKeySet() [INTERNAL]
//   Wrapper for OCIThreadKeySet().
//-----------------------------------------------------------------------------
int dpiOci__threadKeySet(void *envHandle, void *errorHandle, void *key,
        void *value, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCIThreadKeySet", dpiOciSymbols.fnThreadKeySet)
    status = (*dpiOciSymbols.fnThreadKeySet)(envHandle, errorHandle, key,
            value);
    if (status != DPI_OCI_SUCCESS)
        return dpiError__set(error, "set TLS error", DPI_ERR_TLS_ERROR);
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__transCommit() [INTERNAL]
//   Wrapper for OCITransCommit().
//-----------------------------------------------------------------------------
int dpiOci__transCommit(dpiConn *conn, uint32_t flags, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITransCommit", dpiOciSymbols.fnTransCommit)
    status = (*dpiOciSymbols.fnTransCommit)(conn->handle, error->handle,
            flags);
    return dpiError__check(error, status, conn, "commit");
}


//-----------------------------------------------------------------------------
// dpiOci__transPrepare() [INTERNAL]
//   Wrapper for OCITransPrepare().
//-----------------------------------------------------------------------------
int dpiOci__transPrepare(dpiConn *conn, int *commitNeeded, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITransPrepare", dpiOciSymbols.fnTransPrepare)
    status = (*dpiOciSymbols.fnTransPrepare)(conn->handle, error->handle,
            DPI_OCI_DEFAULT);
    *commitNeeded = (status == DPI_OCI_SUCCESS);
    return dpiError__check(error, status, conn, "prepare transaction");
}


//-----------------------------------------------------------------------------
// dpiOci__transRollback() [INTERNAL]
//   Wrapper for OCITransRollback().
//-----------------------------------------------------------------------------
int dpiOci__transRollback(dpiConn *conn, int checkError, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITransRollback", dpiOciSymbols.fnTransRollback)
    status = (*dpiOciSymbols.fnTransRollback)(conn->handle, error->handle,
            DPI_OCI_DEFAULT);
    if (checkError)
        return dpiError__check(error, status, conn, "rollback");
    return DPI_SUCCESS;
}


//-----------------------------------------------------------------------------
// dpiOci__transStart() [INTERNAL]
//   Wrapper for OCITransStart().
//-----------------------------------------------------------------------------
int dpiOci__transStart(dpiConn *conn, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITransStart", dpiOciSymbols.fnTransStart)
    status = (*dpiOciSymbols.fnTransStart)(conn->handle, error->handle, 0,
            DPI_OCI_TRANS_NEW);
    return dpiError__check(error, status, conn, "start transaction");
}


//-----------------------------------------------------------------------------
// dpiOci__typeByFullName() [INTERNAL]
//   Wrapper for OCITypeByFullName().
//-----------------------------------------------------------------------------
int dpiOci__typeByFullName(dpiConn *conn, const char *name,
        uint32_t nameLength, void **tdo, dpiError *error)
{
    int status;

    DPI_OCI_LOAD_SYMBOL("OCITypeByFullName", dpiOciSymbols.fnTypeByFullName)
    status = (*dpiOciSymbols.fnTypeByFullName)(conn->env->handle,
            error->handle, conn->handle, name, nameLength, NULL, 0,
            DPI_OCI_DURATION_SESSION, DPI_OCI_TYPEGET_ALL, tdo);
    return dpiError__check(error, status, conn, "get type by full name");
}

