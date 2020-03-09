// Copyright 2017 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package godror

/*
#include <stdlib.h>
#include <stdio.h>
#include "dpiImpl.h"

void CallbackSubscrDebug(void *context, dpiSubscrMessage *message);

*/
import "C"

import (
	"log"
	"strings"
	"sync"
	"unsafe"

	errors "golang.org/x/xerrors"
)

// Cannot pass *Subscription to C, so pass an uint64 that points to this map entry
var (
	subscriptionsMu sync.Mutex
	subscriptions   = make(map[uint64]*Subscription)
	subscriptionsID uint64
)

// CallbackSubscr is the callback for C code on subscription event.
//export CallbackSubscr
func CallbackSubscr(ctx unsafe.Pointer, message *C.dpiSubscrMessage) {
	log.Printf("CB %p %+v", ctx, message)
	if ctx == nil {
		return
	}
	subscriptionsMu.Lock()
	subscr := subscriptions[*((*uint64)(ctx))]
	subscriptionsMu.Unlock()

	getRows := func(rws *C.dpiSubscrMessageRow, rwsNum C.uint32_t) []RowEvent {
		if rwsNum == 0 {
			return nil
		}
		cRws := (*((*[maxArraySize]C.dpiSubscrMessageRow)(unsafe.Pointer(rws))))[:int(rwsNum)]
		rows := make([]RowEvent, len(cRws))
		for i, row := range cRws {
			rows[i] = RowEvent{
				Operation: Operation(row.operation),
				Rowid:     C.GoStringN(row.rowid, C.int(row.rowidLength)),
			}
		}
		return rows
	}
	getTables := func(tbls *C.dpiSubscrMessageTable, tblsNum C.uint32_t) []TableEvent {
		if tblsNum == 0 {
			return nil
		}
		cTbls := (*((*[maxArraySize]C.dpiSubscrMessageTable)(unsafe.Pointer(tbls))))[:int(tblsNum)]
		tables := make([]TableEvent, len(cTbls))
		for i, tbl := range cTbls {
			tables[i] = TableEvent{
				Operation: Operation(tbl.operation),
				Name:      C.GoStringN(tbl.name, C.int(tbl.nameLength)),
				Rows:      getRows(tbl.rows, tbl.numRows),
			}
		}
		return tables
	}
	getQueries := func(qrys *C.dpiSubscrMessageQuery, qrysNum C.uint32_t) []QueryEvent {
		if qrysNum == 0 {
			return nil
		}
		cQrys := (*((*[maxArraySize]C.dpiSubscrMessageQuery)(unsafe.Pointer(qrys))))[:int(qrysNum)]
		queries := make([]QueryEvent, len(cQrys))
		for i, qry := range cQrys {
			queries[i] = QueryEvent{
				ID:        uint64(qry.id),
				Operation: Operation(qry.operation),
				Tables:    getTables(qry.tables, qry.numTables),
			}
		}
		return queries
	}
	var err error
	if message.errorInfo != nil {
		err = fromErrorInfo(*message.errorInfo)
	}

	subscr.callback(Event{
		Err:     err,
		Type:    EventType(message.eventType),
		DB:      C.GoStringN(message.dbName, C.int(message.dbNameLength)),
		Tables:  getTables(message.tables, message.numTables),
		Queries: getQueries(message.queries, message.numQueries),
	})
}

// Event for a subscription.
type Event struct {
	Tables  []TableEvent
	Queries []QueryEvent
	DB      string
	Err     error
	Type    EventType
}

// QueryEvent is an event of a Query.
type QueryEvent struct {
	Tables []TableEvent
	ID     uint64
	Operation
}

// TableEvent is for a Table-related event.
type TableEvent struct {
	Rows []RowEvent
	Name string
	Operation
}

// RowEvent is for row-related event.
type RowEvent struct {
	Rowid string
	Operation
}

// Subscription for events in the DB.
type Subscription struct {
	conn      *conn
	dpiSubscr *C.dpiSubscr
	callback  func(Event)
	ID        uint64
}

func (s *Subscription) getError() error { return s.conn.getError() }

// NewSubscription creates a new Subscription in the DB.
//
// Make sure your user has CHANGE NOTIFICATION privilege!
//
// This code is EXPERIMENTAL yet!
func (c *conn) NewSubscription(name string, cb func(Event)) (*Subscription, error) {
	if !c.connParams.EnableEvents {
		return nil, errors.New("subscription must be allowed by specifying \"enableEvents=1\" in the connection parameters")
	}
	subscr := Subscription{conn: c, callback: cb}
	params := (*C.dpiSubscrCreateParams)(C.malloc(C.sizeof_dpiSubscrCreateParams))
	//defer func() { C.free(unsafe.Pointer(params)) }()
	C.dpiContext_initSubscrCreateParams(c.drv.dpiContext, params)
	params.subscrNamespace = C.DPI_SUBSCR_NAMESPACE_DBCHANGE
	params.protocol = C.DPI_SUBSCR_PROTO_CALLBACK
	params.qos = C.DPI_SUBSCR_QOS_BEST_EFFORT | C.DPI_SUBSCR_QOS_QUERY | C.DPI_SUBSCR_QOS_ROWIDS
	params.operations = C.DPI_OPCODE_ALL_OPS
	if name != "" {
		params.name = C.CString(name)
		params.nameLength = C.uint32_t(len(name))
	}
	// typedef void (*dpiSubscrCallback)(void* context, dpiSubscrMessage *message);
	params.callback = C.dpiSubscrCallback(C.CallbackSubscrDebug)
	// cannot pass &subscr to C, so pass indirectly
	subscriptionsMu.Lock()
	subscriptionsID++
	subscr.ID = subscriptionsID
	subscriptions[subscr.ID] = &subscr
	subscriptionsMu.Unlock()
	subscrID := (*C.uint64_t)(C.malloc(8))
	*subscrID = C.uint64_t(subscriptionsID)
	params.callbackContext = unsafe.Pointer(subscrID)

	dpiSubscr := (*C.dpiSubscr)(C.malloc(C.sizeof_void))

	if C.dpiConn_subscribe(c.dpiConn,
		params,
		(**C.dpiSubscr)(unsafe.Pointer(&dpiSubscr)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(params))
		C.free(unsafe.Pointer(dpiSubscr))
		err := errors.Errorf("newSubscription: %w", c.getError())
		if strings.Contains(errors.Unwrap(err).Error(), "DPI-1065:") {
			err = errors.Errorf("specify \"enableEvents=1\" connection parameter on connection to be able to use subscriptions: %w", err)
		}
		return nil, err
	}
	subscr.dpiSubscr = dpiSubscr
	return &subscr, nil
}

// Register a query for Change Notification.
//
// This code is EXPERIMENTAL yet!
func (s *Subscription) Register(qry string, params ...interface{}) error {
	cQry := C.CString(qry)
	defer func() { C.free(unsafe.Pointer(cQry)) }()

	var dpiStmt *C.dpiStmt
	if C.dpiSubscr_prepareStmt(s.dpiSubscr, cQry, C.uint32_t(len(qry)), &dpiStmt) == C.DPI_FAILURE {
		return errors.Errorf("prepareStmt[%p]: %w", s.dpiSubscr, s.getError())
	}
	defer func() { C.dpiStmt_release(dpiStmt) }()

	mode := C.dpiExecMode(C.DPI_MODE_EXEC_DEFAULT)
	var qCols C.uint32_t
	if C.dpiStmt_execute(dpiStmt, mode, &qCols) == C.DPI_FAILURE {
		return errors.Errorf("executeStmt: %w", s.getError())
	}
	var queryID C.uint64_t
	if C.dpiStmt_getSubscrQueryId(dpiStmt, &queryID) == C.DPI_FAILURE {
		return errors.Errorf("getSubscrQueryId: %w", s.getError())
	}
	if Log != nil {
		Log("msg", "subscribed", "query", qry, "id", queryID)
	}

	return nil
}

// Close the subscription.
//
// This code is EXPERIMENTAL yet!
func (s *Subscription) Close() error {
	subscriptionsMu.Lock()
	delete(subscriptions, s.ID)
	subscriptionsMu.Unlock()
	dpiSubscr := s.dpiSubscr
	conn := s.conn
	s.conn = nil
	s.dpiSubscr = nil
	s.callback = nil
	if dpiSubscr == nil || conn == nil || conn.dpiConn == nil {
		return nil
	}
	if C.dpiConn_unsubscribe(conn.dpiConn, dpiSubscr) == C.DPI_FAILURE {
		return errors.Errorf("close: %w", s.getError())
	}
	return nil
}

// EventType is the type of an event.
type EventType C.dpiEventType

// Events that can be watched.
const (
	EvtStartup     = EventType(C.DPI_EVENT_STARTUP)
	EvtShutdown    = EventType(C.DPI_EVENT_SHUTDOWN)
	EvtShutdownAny = EventType(C.DPI_EVENT_SHUTDOWN_ANY)
	EvtDereg       = EventType(C.DPI_EVENT_DEREG)
	EvtObjChange   = EventType(C.DPI_EVENT_OBJCHANGE)
	EvtQueryChange = EventType(C.DPI_EVENT_QUERYCHANGE)
	EvtAQ          = EventType(C.DPI_EVENT_AQ)
)

// Operation in the DB.
type Operation C.dpiOpCode

const (
	// OpAll Indicates that notifications should be sent for all operations on the table or query.
	OpAll = Operation(C.DPI_OPCODE_ALL_OPS)
	// OpAllRows Indicates that all rows have been changed in the table or query (or too many rows were changed or row information was not requested).
	OpAllRows = Operation(C.DPI_OPCODE_ALL_ROWS)
	// OpInsert Indicates that an insert operation has taken place in the table or query.
	OpInsert = Operation(C.DPI_OPCODE_INSERT)
	// OpUpdate Indicates that an update operation has taken place in the table or query.
	OpUpdate = Operation(C.DPI_OPCODE_UPDATE)
	// OpDelete Indicates that a delete operation has taken place in the table or query.
	OpDelete = Operation(C.DPI_OPCODE_DELETE)
	// OpAlter Indicates that the registered table or query has been altered.
	OpAlter = Operation(C.DPI_OPCODE_ALTER)
	// OpDrop Indicates that the registered table or query has been dropped.
	OpDrop = Operation(C.DPI_OPCODE_DROP)
	// OpUnknown An unknown operation has taken place.
	OpUnknown = Operation(C.DPI_OPCODE_UNKNOWN)
)
