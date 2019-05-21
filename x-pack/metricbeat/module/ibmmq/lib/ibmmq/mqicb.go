/*
  Copyright (c) IBM Corporation 2018

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
This file deals with asynchronous delivery of MQ messages via the MQCTL/MQCB verbs.
*/
package ibmmqi

/*
#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

extern void MQCALLBACK_Go(MQHCONN, MQMD *, MQGMO *, PMQVOID, MQCBC *);
extern void MQCALLBACK_C(MQHCONN hc,MQMD *md,MQGMO *gmo,PMQVOID buf,MQCBC *cbc);
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"
)

// The user's callback function must match this signature
type MQCB_FUNCTION func(*MQObject, *MQMD, *MQGMO, []byte, *MQCBC, *MQReturn)

// Need to keep references to the user's callback function and some other
// structure elements which do not map to the C functions, or do not need to
// be passed onwards
type cbInfo struct {
	hObj             *MQObject
	callbackFunction MQCB_FUNCTION
	callbackArea     interface{}
	connectionArea   interface{}
}

// This map is indexed by a combination of the hConn and hObj values
var cbMap = make(map[string]*cbInfo)

/*
MQCALLBACK_Go is a wrapper callback function that will invoke the user-supplied callback
after converting the C structures into the corresponding Go format.

The "export" directive makes the function available through the CGo processing to be
accessible from a C function. See mqicb_c.go for the proxy/gateway C function that in turn calls this one
*/
//export MQCALLBACK_Go
func MQCALLBACK_Go(hConn C.MQHCONN, mqmd *C.MQMD, mqgmo *C.MQGMO, mqBuffer C.PMQVOID, mqcbc *C.MQCBC) {

	var cbHObj *MQObject

	// Find the real callback function and invoke it
	// Invoked function should match signature of the MQCB_FUNCTION type
	gogmo := NewMQGMO()
	gomd := NewMQMD()
	gocbc := NewMQCBC()

	// For EVENT callbacks, the GMO and MD may be NULL
	if mqgmo != (C.PMQGMO)(C.NULL) {
		copyGMOfromC(mqgmo, gogmo)
	}

	if mqmd != (C.PMQMD)(C.NULL) {
		copyMDfromC(mqmd, gomd)
	}

	// This should never be NULL
	copyCBCfromC(mqcbc, gocbc)

	mqreturn := &MQReturn{MQCC: int32(mqcbc.CompCode),
		MQRC: int32(mqcbc.Reason),
		verb: "MQCALLBACK",
	}

	key := makeKey(hConn, mqcbc.Hobj)
	info, ok := cbMap[key]

	// The MQ Client libraries seem to sometimes call us with an EVENT
	// even if it's not been registered. And therefore the cbMap does not
	// contain a matching callback function with the hObj.  It has
	// been seen with a 2033 return (see issue #75).
	//
	// This feels like wrong behaviour from the client, but we need to find a
	// way to deal with it even if it gets fixed in future.
	// The way I've chosen is to find the first entry in
	// the map associated with the hConn and call its registered function with
	// a dummy hObj.
	if !ok {
		if gocbc.CallType == MQCBCT_EVENT_CALL && mqcbc.Hobj == 0 {
			key = makePartialKey(hConn)
			for k, i := range cbMap {
				if strings.HasPrefix(k, key) {
					ok = true
					info = i
					cbHObj = &MQObject{qMgr: info.hObj.qMgr, Name: ""}
					// Only care about finding one match in the table
					break
				}
			}
		}
	} else {
		cbHObj = info.hObj
	}

	if ok {

		gocbc.CallbackArea = info.callbackArea
		gocbc.ConnectionArea = info.connectionArea

		// Get the data
		b := C.GoBytes(unsafe.Pointer(mqBuffer), C.int(mqcbc.DataLength))
		// And finally call the user function
		info.callbackFunction(cbHObj, gomd, gogmo, b, gocbc, mqreturn)
	}
}

/*
CB is the function to register/unregister a callback function for a queue, based on
criteria in the message descriptor and get-message-options
*/
func (object *MQObject) CB(goOperation int32, gocbd *MQCBD, gomd *MQMD, gogmo *MQGMO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqOperation C.MQLONG
	var mqcbd C.MQCBD
	var mqmd C.MQMD
	var mqgmo C.MQGMO

	mqOperation = C.MQLONG(goOperation)
	copyCBDtoC(&mqcbd, gocbd)
	copyMDtoC(&mqmd, gomd)
	copyGMOtoC(&mqgmo, gogmo)

	key := makeKey(object.qMgr.hConn, object.hObj)

	// The callback function is a C function that is a proxy for the MQCALLBACK_Go function
	// defined here. And that in turn will call the user's callback function
	mqcbd.CallbackFunction = (C.MQPTR)(unsafe.Pointer(C.MQCALLBACK_C))

	C.MQCB(object.qMgr.hConn, mqOperation, (C.PMQVOID)(unsafe.Pointer(&mqcbd)),
		object.hObj,
		(C.PMQVOID)(unsafe.Pointer(&mqmd)), (C.PMQVOID)(unsafe.Pointer(&mqgmo)),
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCB",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	// Add or remove the control information in the map used by the callback routines
	switch mqOperation {
	case C.MQOP_DEREGISTER:
		delete(cbMap, key)
	case C.MQOP_REGISTER:
		// Stash the hObj and real function to be called
		info := &cbInfo{hObj: object,
			callbackFunction: gocbd.CallbackFunction,
			connectionArea:   nil,
			callbackArea:     gocbd.CallbackArea}
		cbMap[key] = info
	default: // Other values leave the map alone
	}

	return nil
}

/*
Ctl is the function that starts/stops invocation of a registered callback.
*/
func (x *MQQueueManager) Ctl(goOperation int32, goctlo *MQCTLO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqOperation C.MQLONG
	var mqctlo C.MQCTLO

	mqOperation = C.MQLONG(goOperation)
	copyCTLOtoC(&mqctlo, goctlo)

	// Need to make sure control information is available before the callback
	// is enabled. So this gets setup even if the MQCTL fails.
	key := makePartialKey(x.hConn)
	for k, info := range cbMap {
		if strings.HasPrefix(k, key) {
			info.connectionArea = goctlo.ConnectionArea
		}
	}

	C.MQCTL(x.hConn, mqOperation, (C.PMQVOID)(unsafe.Pointer(&mqctlo)), &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCTL",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

// Functions below here manage the map of objects and control information so that
// the Go variables can be saved/restored from invocations to the C layer
func makeKey(hConn C.MQHCONN, hObj C.MQHOBJ) string {
	key := fmt.Sprintf("%d/%d", hConn, hObj)
	return key
}

func makePartialKey(hConn C.MQHCONN) string {
	key := fmt.Sprintf("%d/", hConn)
	return key
}

// Functions to delete any structures used to map C elements to Go
func cbRemoveConnection(hConn C.MQHCONN) {
	// Remove all of the hObj values for this hconn
	key := makePartialKey(hConn)
	for k, _ := range cbMap {
		if strings.HasPrefix(k, key) {
			delete(cbMap, k)
		}
	}
}

func cbRemoveHandle(hConn C.MQHCONN, hObj C.MQHOBJ) {
	key := makeKey(hConn, hObj)
	delete(cbMap, key)
}
