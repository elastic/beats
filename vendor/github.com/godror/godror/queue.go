// Copyright 2019 Tamás Gulácsi
//
//
// SPDX-License-Identifier: UPL-1.0 OR Apache-2.0

package godror

/*
#include <stdlib.h>
#include "dpiImpl.h"
*/
import "C"
import (
	"context"
	"sync"
	"time"
	"unsafe"

	errors "golang.org/x/xerrors"
)

const MsgIDLength = 16

var zeroMsgID [MsgIDLength]byte

// DefaultEnqOptions is the default set for NewQueue.
var DefaultEnqOptions = EnqOptions{
	Visibility:   VisibleImmediate,
	DeliveryMode: DeliverPersistent,
}

// DefaultDeqOptions is the default set for NewQueue.
var DefaultDeqOptions = DeqOptions{
	Mode:         DeqRemove,
	DeliveryMode: DeliverPersistent,
	Navigation:   NavFirst,
	Visibility:   VisibleImmediate,
	Wait:         30,
}

// Queue represents an Oracle Advanced Queue.
type Queue struct {
	PayloadObjectType ObjectType
	props             []*C.dpiMsgProps
	name              string
	conn              *conn
	dpiQueue          *C.dpiQueue

	mu sync.Mutex
}

// NewQueue creates a new Queue.
//
// WARNING: the connection given to it must not be closed before the Queue is closed!
// So use an sql.Conn for it.
func NewQueue(ctx context.Context, execer Execer, name string, payloadObjectTypeName string) (*Queue, error) {
	cx, err := DriverConn(ctx, execer)
	if err != nil {
		return nil, err
	}
	Q := Queue{conn: cx.(*conn), name: name}

	var payloadType *C.dpiObjectType
	if payloadObjectTypeName != "" {
		if Q.PayloadObjectType, err = Q.conn.GetObjectType(payloadObjectTypeName); err != nil {
			return nil, err
		} else {
			payloadType = Q.PayloadObjectType.dpiObjectType
		}
	}
	value := C.CString(name)
	if C.dpiConn_newQueue(Q.conn.dpiConn, value, C.uint(len(name)), payloadType, &Q.dpiQueue) == C.DPI_FAILURE {
		err = errors.Errorf("newQueue %q: %w", name, Q.conn.drv.getError())
	}
	C.free(unsafe.Pointer(value))
	if err != nil {
		cx.Close()
		return nil, err
	}
	if err = Q.SetEnqOptions(DefaultEnqOptions); err != nil {
		cx.Close()
		Q.Close()
		return nil, err
	}
	if err = Q.SetDeqOptions(DefaultDeqOptions); err != nil {
		cx.Close()
		Q.Close()
		return nil, err
	}
	return &Q, nil
}

// Close the queue.
func (Q *Queue) Close() error {
	c, q := Q.conn, Q.dpiQueue
	Q.conn, Q.dpiQueue = nil, nil
	if q == nil {
		return nil
	}
	if C.dpiQueue_release(q) == C.DPI_FAILURE {
		return errors.Errorf("release: %w", c.getError())
	}
	return nil
}

// Name of the queue.
func (Q *Queue) Name() string { return Q.name }

// EnqOptions returns the queue's enqueue options in effect.
func (Q *Queue) EnqOptions() (EnqOptions, error) {
	var E EnqOptions
	var opts *C.dpiEnqOptions
	if C.dpiQueue_getEnqOptions(Q.dpiQueue, &opts) == C.DPI_FAILURE {
		return E, errors.Errorf("getEnqOptions: %w", Q.conn.drv.getError())
	}
	err := E.fromOra(Q.conn.drv, opts)
	return E, err
}

// DeqOptions returns the queue's dequeue options in effect.
func (Q *Queue) DeqOptions() (DeqOptions, error) {
	var D DeqOptions
	var opts *C.dpiDeqOptions
	if C.dpiQueue_getDeqOptions(Q.dpiQueue, &opts) == C.DPI_FAILURE {
		return D, errors.Errorf("getDeqOptions: %w", Q.conn.drv.getError())
	}
	err := D.fromOra(Q.conn.drv, opts)
	return D, err
}

// Dequeues messages into the given slice.
// Returns the number of messages filled in the given slice.
func (Q *Queue) Dequeue(messages []Message) (int, error) {
	Q.mu.Lock()
	defer Q.mu.Unlock()
	var props []*C.dpiMsgProps
	if cap(Q.props) >= len(messages) {
		props = Q.props[:len(messages)]
	} else {
		props = make([]*C.dpiMsgProps, len(messages))
	}
	Q.props = props

	var ok C.int
	num := C.uint(len(props))
	if num == 1 {
		ok = C.dpiQueue_deqOne(Q.dpiQueue, &props[0])
	} else {
		ok = C.dpiQueue_deqMany(Q.dpiQueue, &num, &props[0])
	}
	if ok == C.DPI_FAILURE {
		err := Q.conn.getError()
		if code := err.(interface{ Code() int }).Code(); code == 3156 {
			return 0, context.DeadlineExceeded
		}
		return 0, errors.Errorf("dequeue: %w", err)
	}
	var firstErr error
	for i, p := range props[:int(num)] {
		if err := messages[i].fromOra(Q.conn, p, &Q.PayloadObjectType); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
		C.dpiMsgProps_release(p)
	}
	return int(num), firstErr
}

// Enqueue all the messages given.
//
// WARNING: calling this function in parallel on different connections acquired from the same pool may fail due to Oracle bug 29928074. Ensure that this function is not run in parallel, use standalone connections or connections from different pools, or make multiple calls to Queue.enqOne() instead. The function Queue.Dequeue() call is not affected.
func (Q *Queue) Enqueue(messages []Message) error {
	Q.mu.Lock()
	defer Q.mu.Unlock()
	var props []*C.dpiMsgProps
	if cap(Q.props) >= len(messages) {
		props = Q.props[:len(messages)]
	} else {
		props = make([]*C.dpiMsgProps, len(messages))
	}
	Q.props = props
	defer func() {
		for _, p := range props {
			if p != nil {
				C.dpiMsgProps_release(p)
			}
		}
	}()
	for i, m := range messages {
		if C.dpiConn_newMsgProps(Q.conn.dpiConn, &props[i]) == C.DPI_FAILURE {
			return errors.Errorf("newMsgProps: %w", Q.conn.getError())
		}
		if err := m.toOra(Q.conn.drv, props[i]); err != nil {
			return err
		}
	}

	var ok C.int
	if len(messages) == 1 {
		ok = C.dpiQueue_enqOne(Q.dpiQueue, props[0])
	} else {
		ok = C.dpiQueue_enqMany(Q.dpiQueue, C.uint(len(props)), &props[0])
	}
	if ok == C.DPI_FAILURE {
		return errors.Errorf("enqueue %#v: %w", messages, Q.conn.getError())
	}

	return nil
}

// Message is a message - either received or being sent.
type Message struct {
	Correlation, ExceptionQ string
	Enqueued                time.Time
	MsgID, OriginalMsgID    [16]byte
	Raw                     []byte
	Delay, Expiration       int32
	Priority, NumAttempts   int32
	Object                  *Object
	DeliveryMode            DeliveryMode
	State                   MessageState
}

func (M *Message) toOra(d *drv, props *C.dpiMsgProps) error {
	var firstErr error
	OK := func(ok C.int, name string) {
		if ok == C.DPI_SUCCESS {
			return
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", name, d.getError())
		}
	}
	if M.Correlation != "" {
		value := C.CString(M.Correlation)
		OK(C.dpiMsgProps_setCorrelation(props, value, C.uint(len(M.Correlation))), "setCorrelation")
		C.free(unsafe.Pointer(value))
	}

	if M.Delay != 0 {
		OK(C.dpiMsgProps_setDelay(props, C.int(M.Delay)), "setDelay")
	}

	if M.ExceptionQ != "" {
		value := C.CString(M.ExceptionQ)
		OK(C.dpiMsgProps_setExceptionQ(props, value, C.uint(len(M.ExceptionQ))), "setExceptionQ")
		C.free(unsafe.Pointer(value))
	}

	if M.Expiration != 0 {
		OK(C.dpiMsgProps_setExpiration(props, C.int(M.Expiration)), "setExpiration")
	}

	if M.OriginalMsgID != zeroMsgID {
		OK(C.dpiMsgProps_setOriginalMsgId(props, (*C.char)(unsafe.Pointer(&M.OriginalMsgID[0])), MsgIDLength), "setMsgOriginalId")
	}

	OK(C.dpiMsgProps_setPriority(props, C.int(M.Priority)), "setPriority")

	if M.Object == nil {
		OK(C.dpiMsgProps_setPayloadBytes(props, (*C.char)(unsafe.Pointer(&M.Raw[0])), C.uint(len(M.Raw))), "setPayloadBytes")
	} else {
		OK(C.dpiMsgProps_setPayloadObject(props, M.Object.dpiObject), "setPayloadObject")
	}

	return firstErr
}

func (M *Message) fromOra(c *conn, props *C.dpiMsgProps, objType *ObjectType) error {
	var firstErr error
	OK := func(ok C.int, name string) bool {
		if ok == C.DPI_SUCCESS {
			return true
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", name, c.getError())
		}
		return false
	}
	M.NumAttempts = 0
	var cint C.int
	if OK(C.dpiMsgProps_getNumAttempts(props, &cint), "getNumAttempts") {
		M.NumAttempts = int32(cint)
	}
	var value *C.char
	var length C.uint
	M.Correlation = ""
	if OK(C.dpiMsgProps_getCorrelation(props, &value, &length), "getCorrelation") {
		M.Correlation = C.GoStringN(value, C.int(length))
	}

	M.Delay = 0
	if OK(C.dpiMsgProps_getDelay(props, &cint), "getDelay") {
		M.Delay = int32(cint)
	}

	M.DeliveryMode = DeliverPersistent
	var mode C.dpiMessageDeliveryMode
	if OK(C.dpiMsgProps_getDeliveryMode(props, &mode), "getDeliveryMode") {
		M.DeliveryMode = DeliveryMode(mode)
	}

	M.ExceptionQ = ""
	if OK(C.dpiMsgProps_getExceptionQ(props, &value, &length), "getExceptionQ") {
		M.ExceptionQ = C.GoStringN(value, C.int(length))
	}

	var ts C.dpiTimestamp
	M.Enqueued = time.Time{}
	if OK(C.dpiMsgProps_getEnqTime(props, &ts), "getEnqTime") {
		tz := c.timeZone
		if ts.tzHourOffset != 0 || ts.tzMinuteOffset != 0 {
			tz = timeZoneFor(ts.tzHourOffset, ts.tzMinuteOffset)
		}
		if tz == nil {
			tz = time.Local
		}
		M.Enqueued = time.Date(
			int(ts.year), time.Month(ts.month), int(ts.day),
			int(ts.hour), int(ts.minute), int(ts.second), int(ts.fsecond),
			tz,
		)
	}

	M.Expiration = 0
	if OK(C.dpiMsgProps_getExpiration(props, &cint), "getExpiration") {
		M.Expiration = int32(cint)
	}

	M.MsgID = zeroMsgID
	if OK(C.dpiMsgProps_getMsgId(props, &value, &length), "getMsgId") {
		n := C.int(length)
		if n > MsgIDLength {
			n = MsgIDLength
		}
		copy(M.MsgID[:], C.GoBytes(unsafe.Pointer(value), n))
	}

	M.OriginalMsgID = zeroMsgID
	if OK(C.dpiMsgProps_getOriginalMsgId(props, &value, &length), "getMsgOriginalId") {
		n := C.int(length)
		if n > MsgIDLength {
			n = MsgIDLength
		}
		copy(M.OriginalMsgID[:], C.GoBytes(unsafe.Pointer(value), n))
	}

	M.Priority = 0
	if OK(C.dpiMsgProps_getPriority(props, &cint), "getPriority") {
		M.Priority = int32(cint)
	}

	M.State = 0
	var state C.dpiMessageState
	if OK(C.dpiMsgProps_getState(props, &state), "getState") {
		M.State = MessageState(state)
	}

	M.Raw = nil
	M.Object = nil
	var obj *C.dpiObject
	if OK(C.dpiMsgProps_getPayload(props, &obj, &value, &length), "getPayload") {
		if obj == nil {
			M.Raw = C.GoBytes(unsafe.Pointer(value), C.int(length))
		} else {
			if C.dpiObject_addRef(obj) == C.DPI_FAILURE {
				return objType.getError()
			}
			M.Object = &Object{dpiObject: obj, ObjectType: *objType}
		}
	}
	return nil
}

// EnqOptions are the options used to enqueue a message.
type EnqOptions struct {
	Transformation string
	Visibility     Visibility
	DeliveryMode   DeliveryMode
}

func (E *EnqOptions) fromOra(d *drv, opts *C.dpiEnqOptions) error {
	var firstErr error
	OK := func(ok C.int, msg string) bool {
		if ok == C.DPI_SUCCESS {
			return true
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", msg, d.getError())
		}
		return false
	}

	E.DeliveryMode = DeliverPersistent

	var value *C.char
	var length C.uint
	if OK(C.dpiEnqOptions_getTransformation(opts, &value, &length), "getTransformation") {
		E.Transformation = C.GoStringN(value, C.int(length))
	}

	var vis C.dpiVisibility
	if OK(C.dpiEnqOptions_getVisibility(opts, &vis), "getVisibility") {
		E.Visibility = Visibility(vis)
	}

	return firstErr
}

func (E EnqOptions) toOra(d *drv, opts *C.dpiEnqOptions) error {
	var firstErr error
	OK := func(ok C.int, msg string) bool {
		if ok == C.DPI_SUCCESS {
			return true
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", msg, d.getError())
		}
		return false
	}

	OK(C.dpiEnqOptions_setDeliveryMode(opts, C.dpiMessageDeliveryMode(E.DeliveryMode)), "setDeliveryMode")
	cs := C.CString(E.Transformation)
	OK(C.dpiEnqOptions_setTransformation(opts, cs, C.uint(len(E.Transformation))), "setTransformation")
	C.free(unsafe.Pointer(cs))
	OK(C.dpiEnqOptions_setVisibility(opts, C.uint(E.Visibility)), "setVisibility")
	return firstErr
}

// SetEnqOptions sets all the enqueue options
func (Q *Queue) SetEnqOptions(E EnqOptions) error {
	var opts *C.dpiEnqOptions
	if C.dpiQueue_getEnqOptions(Q.dpiQueue, &opts) == C.DPI_FAILURE {
		return errors.Errorf("getEnqOptions: %w", Q.conn.drv.getError())
	}
	return E.toOra(Q.conn.drv, opts)
}

// DeqOptions are the options used to dequeue a message.
type DeqOptions struct {
	Condition, Consumer, Correlation string
	MsgID, Transformation            string
	Mode                             DeqMode
	DeliveryMode                     DeliveryMode
	Navigation                       DeqNavigation
	Visibility                       Visibility
	Wait                             uint32
}

func (D *DeqOptions) fromOra(d *drv, opts *C.dpiDeqOptions) error {
	var firstErr error
	OK := func(ok C.int, msg string) bool {
		if ok == C.DPI_SUCCESS {
			return true
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", msg, d.getError())
		}
		return false
	}

	var value *C.char
	var length C.uint
	D.Transformation = ""
	if OK(C.dpiDeqOptions_getTransformation(opts, &value, &length), "getTransformation") {
		D.Transformation = C.GoStringN(value, C.int(length))
	}
	D.Condition = ""
	if OK(C.dpiDeqOptions_getCondition(opts, &value, &length), "getCondifion") {
		D.Condition = C.GoStringN(value, C.int(length))
	}
	D.Consumer = ""
	if OK(C.dpiDeqOptions_getConsumerName(opts, &value, &length), "getConsumer") {
		D.Consumer = C.GoStringN(value, C.int(length))
	}
	D.Correlation = ""
	if OK(C.dpiDeqOptions_getCorrelation(opts, &value, &length), "getCorrelation") {
		D.Correlation = C.GoStringN(value, C.int(length))
	}
	D.DeliveryMode = DeliverPersistent
	var mode C.dpiDeqMode
	if OK(C.dpiDeqOptions_getMode(opts, &mode), "getMode") {
		D.Mode = DeqMode(mode)
	}
	D.MsgID = ""
	if OK(C.dpiDeqOptions_getMsgId(opts, &value, &length), "getMsgId") {
		D.MsgID = C.GoStringN(value, C.int(length))
	}
	var nav C.dpiDeqNavigation
	if OK(C.dpiDeqOptions_getNavigation(opts, &nav), "getNavigation") {
		D.Navigation = DeqNavigation(nav)
	}
	var vis C.dpiVisibility
	if OK(C.dpiDeqOptions_getVisibility(opts, &vis), "getVisibility") {
		D.Visibility = Visibility(vis)
	}
	D.Wait = 0
	var u32 C.uint
	if OK(C.dpiDeqOptions_getWait(opts, &u32), "getWait") {
		D.Wait = uint32(u32)
	}
	return firstErr
}

func (D DeqOptions) toOra(d *drv, opts *C.dpiDeqOptions) error {
	var firstErr error
	OK := func(ok C.int, msg string) bool {
		if ok == C.DPI_SUCCESS {
			return true
		}
		if firstErr == nil {
			firstErr = errors.Errorf("%s: %w", msg, d.getError())
		}
		return false
	}

	cs := C.CString(D.Transformation)
	OK(C.dpiDeqOptions_setTransformation(opts, cs, C.uint(len(D.Transformation))), "setTransformation")
	C.free(unsafe.Pointer(cs))

	cs = C.CString(D.Condition)
	OK(C.dpiDeqOptions_setCondition(opts, cs, C.uint(len(D.Condition))), "setCondifion")
	C.free(unsafe.Pointer(cs))

	cs = C.CString(D.Consumer)
	OK(C.dpiDeqOptions_setConsumerName(opts, cs, C.uint(len(D.Consumer))), "setConsumer")
	C.free(unsafe.Pointer(cs))

	cs = C.CString(D.Correlation)
	OK(C.dpiDeqOptions_setCorrelation(opts, cs, C.uint(len(D.Correlation))), "setCorrelation")
	C.free(unsafe.Pointer(cs))

	OK(C.dpiDeqOptions_setDeliveryMode(opts, C.dpiMessageDeliveryMode(D.DeliveryMode)), "setDeliveryMode")
	OK(C.dpiDeqOptions_setMode(opts, C.dpiDeqMode(D.Mode)), "setMode")

	cs = C.CString(D.MsgID)
	OK(C.dpiDeqOptions_setMsgId(opts, cs, C.uint(len(D.MsgID))), "setMsgId")
	C.free(unsafe.Pointer(cs))

	OK(C.dpiDeqOptions_setNavigation(opts, C.dpiDeqNavigation(D.Navigation)), "setNavigation")

	OK(C.dpiDeqOptions_setVisibility(opts, C.dpiVisibility(D.Visibility)), "setVisibility")

	OK(C.dpiDeqOptions_setWait(opts, C.uint(D.Wait)), "setWait")

	return firstErr
}

// SetDeqOptions sets all the dequeue options
func (Q *Queue) SetDeqOptions(D DeqOptions) error {
	var opts *C.dpiDeqOptions
	if C.dpiQueue_getDeqOptions(Q.dpiQueue, &opts) == C.DPI_FAILURE {
		return errors.Errorf("getDeqOptions: %w", Q.conn.drv.getError())
	}
	return D.toOra(Q.conn.drv, opts)
}

// SetDeqCorrelation is a convenience function setting the Correlation DeqOption
func (Q *Queue) SetDeqCorrelation(correlation string) error {
	var opts *C.dpiDeqOptions
	if C.dpiQueue_getDeqOptions(Q.dpiQueue, &opts) == C.DPI_FAILURE {
		return errors.Errorf("getDeqOptions: %w", Q.conn.drv.getError())
	}
	cs := C.CString(correlation)
	ok := C.dpiDeqOptions_setCorrelation(opts, cs, C.uint(len(correlation))) == C.DPI_FAILURE
	C.free(unsafe.Pointer(cs))
	if !ok {
		return errors.Errorf("setCorrelation: %w", Q.conn.drv.getError())
	}
	return nil
}

const (
	NoWait      = uint32(0)
	WaitForever = uint32(1<<31 - 1)
)

// MessageState constants representing message's state.
type MessageState uint32

const (
	// MsgStateReady says that "The message is ready to be processed".
	MsgStateReady = MessageState(C.DPI_MSG_STATE_READY)
	// MsgStateWaiting says that "The message is waiting for the delay time to expire".
	MsgStateWaiting = MessageState(C.DPI_MSG_STATE_WAITING)
	// MsgStateProcessed says that "The message has already been processed and is retained".
	MsgStateProcessed = MessageState(C.DPI_MSG_STATE_PROCESSED)
	// MsgStateExpired says that "The message has been moved to the exception queue".
	MsgStateExpired = MessageState(C.DPI_MSG_STATE_EXPIRED)
)

// DeliveryMode constants for delivery modes.
type DeliveryMode uint32

const (
	// DeliverPersistent is to Dequeue only persistent messages from the queue. This is the default mode.
	DeliverPersistent = DeliveryMode(C.DPI_MODE_MSG_PERSISTENT)
	// DeliverBuffered is to Dequeue only buffered messages from the queue.
	DeliverBuffered = DeliveryMode(C.DPI_MODE_MSG_BUFFERED)
	// DeliverPersistentOrBuffered is to Dequeue both persistent and buffered messages from the queue.
	DeliverPersistentOrBuffered = DeliveryMode(C.DPI_MODE_MSG_PERSISTENT_OR_BUFFERED)
)

// Visibility constants represents visibility.
type Visibility uint32

const (
	// VisibleImmediate means that "The message is not part of the current transaction but constitutes a transaction of its own".
	VisibleImmediate = Visibility(C.DPI_VISIBILITY_IMMEDIATE)
	// VisibleOnCommit means that "The message is part of the current transaction. This is the default value".
	VisibleOnCommit = Visibility(C.DPI_VISIBILITY_ON_COMMIT)
)

// DeqMode constants for dequeue modes.
type DeqMode uint32

const (
	// DeqRemove reads the message and updates or deletes it. This is the default mode. Note that the message may be retained in the queue table based on retention properties.
	DeqRemove = DeqMode(C.DPI_MODE_DEQ_REMOVE)
	// DeqBrows reads the message without acquiring a lock on the message (equivalent to a SELECT statement).
	DeqBrowse = DeqMode(C.DPI_MODE_DEQ_BROWSE)
	// DeqLocked reads the message and obtain a write lock on the message (equivalent to a SELECT FOR UPDATE statement).
	DeqLocked = DeqMode(C.DPI_MODE_DEQ_LOCKED)
	// DeqPeek confirms receipt of the message but does not deliver the actual message content.
	DeqPeek = DeqMode(C.DPI_MODE_DEQ_REMOVE_NO_DATA)
)

// DeqNavigation constants for navigation.
type DeqNavigation uint32

const (
	// NavFirst retrieves the first available message that matches the search criteria. This resets the position to the beginning of the queue.
	NavFirst = DeqNavigation(C.DPI_DEQ_NAV_FIRST_MSG)
	// NavNext skips the remainder of the current transaction group (if any) and retrieves the first message of the next transaction group. This option can only be used if message grouping is enabled for the queue.
	NavNextTran = DeqNavigation(C.DPI_DEQ_NAV_NEXT_TRANSACTION)
	// NavNext  	Retrieves the next available message that matches the search criteria. This is the default method.
	NavNext = DeqNavigation(C.DPI_DEQ_NAV_NEXT_MSG)
)
