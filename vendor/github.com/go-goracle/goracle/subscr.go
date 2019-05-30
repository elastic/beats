// Copyright 2017 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package goracle

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
	"unsafe"

	"github.com/pkg/errors"
)

// CallbackSubscr is the callback for C code on subscription event.
//export CallbackSubscr
func CallbackSubscr(ctx unsafe.Pointer, message *C.dpiSubscrMessage) {
	log.Printf("CB %p %+v", ctx, message)
	if ctx == nil {
		return
	}
	subscr := (*Subscription)(ctx)

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
	C.dpiContext_initSubscrCreateParams(c.dpiContext, params)
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
	params.callbackContext = unsafe.Pointer(&subscr)

	dpiSubscr := (*C.dpiSubscr)(C.malloc(C.sizeof_void))

	if C.dpiConn_subscribe(c.dpiConn,
		params,
		(**C.dpiSubscr)(unsafe.Pointer(&dpiSubscr)),
	) == C.DPI_FAILURE {
		C.free(unsafe.Pointer(params))
		C.free(unsafe.Pointer(dpiSubscr))
		err := errors.Wrap(c.getError(), "newSubscription")
		if strings.Contains(errors.Cause(err).Error(), "DPI-1065:") {
			err = errors.WithMessage(err, "specify \"enableEvents=1\" connection parameter on connection to be able to use subscriptions")
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
		return errors.Wrapf(s.getError(), "prepareStmt[%p]", s.dpiSubscr)
	}
	defer func() { C.dpiStmt_release(dpiStmt) }()

	mode := C.dpiExecMode(C.DPI_MODE_EXEC_DEFAULT)
	var qCols C.uint32_t
	if C.dpiStmt_execute(dpiStmt, mode, &qCols) == C.DPI_FAILURE {
		return errors.Wrap(s.getError(), "executeStmt")
	}
	var queryID C.uint64_t
	if C.dpiStmt_getSubscrQueryId(dpiStmt, &queryID) == C.DPI_FAILURE {
		return errors.Wrap(s.getError(), "getSubscrQueryId")
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
	dpiSubscr := s.dpiSubscr
	conn := s.conn
	s.conn = nil
	s.dpiSubscr = nil
	s.callback = nil
	if dpiSubscr == nil || conn == nil || conn.dpiConn == nil {
		return nil
	}
	if C.dpiConn_unsubscribe(conn.dpiConn, dpiSubscr) == C.DPI_FAILURE {
		return errors.Wrap(s.getError(), "close")
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
	EvtDropDB      = EventType(C.DPI_EVENT_DROP_DB)
	EvtDereg       = EventType(C.DPI_EVENT_DEREG)
	EvtObjChange   = EventType(C.DPI_EVENT_OBJCHANGE)
	EvtQueryChange = EventType(C.DPI_EVENT_QUERYCHANGE)
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
