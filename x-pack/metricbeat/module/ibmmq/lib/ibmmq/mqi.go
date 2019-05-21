/*
Package ibmmq provides a wrapper to a the IBM MQ procedural interface (the MQI).

The verbs are given mixed case names without MQ - Open instead
of MQOPEN etc.

For more information on the MQI, including detailed descriptions of the functions,
constants and structures, see the MQ Knowledge Center
at https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.dev.doc/q023720_.htm#q023720_

If an MQI call returns MQCC_FAILED or MQCC_WARNING, a custom error
type is returned containing the MQCC/MQRC values as
a formatted string. Use mqreturn:= err(*ibmmq.MQReturn) to access
the particular MQRC or MQCC values.

The build directives assume the default MQ installation path
which is in /opt/mqm (Linux) and c:\Program Files\IBM\MQ (Windows).
If you use a non-default path for the installation, you can set
environment variables CGO_CFLAGS and CGO_LDFLAGS to reference those
directories.
*/
package ibmmqi

/*
  Copyright (c) IBM Corporation 2016, 2018

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
#cgo !windows CFLAGS: -I/opt/mqm/inc -D_REENTRANT
#cgo  windows CFLAGS:  -I"C:/Program Files/IBM/MQ/Tools/c/include"
#cgo !windows,!darwin LDFLAGS: -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo darwin   LDFLAGS:         -L/opt/mqm/lib64 -lmqm_r -Wl,-rpath,/opt/mqm/lib64 -Wl,-rpath,/usr/lib64
#cgo windows LDFLAGS: -L "C:/Program Files/IBM/MQ/bin64" -lmqm

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>

*/
import "C"

import (
	"encoding/binary"
	"io"
	"strings"
	"unsafe"
)

/*
   This file contains the C wrappers, calling out to structure-specific
   functions where necessary.

   Define some basic types to hold the
   references to MQ objects - hconn, hobj - and
   a simple way to pass the combination of MQCC/MQRC
   returned from MQI verbs

   The object name is copied into the structures only
   for convenience. It's not really needed, but
   it can sometimes be nice to print which queue an hObj
   refers to during debug.
*/

/*
MQQueueManager contains the connection to the queue manager
*/
type MQQueueManager struct {
	hConn C.MQHCONN
	Name  string
}

/*
MQObject contains a reference to an open object and the associated
queue manager
*/
type MQObject struct {
	hObj C.MQHOBJ
	qMgr *MQQueueManager
	Name string
}

/*
 * MQMessageHandle is a wrapper for the C message handle
 * type. Unlike the C MQI, a valid hConn is required to create
 * the message handle.
 */
type MQMessageHandle struct {
	hMsg C.MQHMSG
	qMgr *MQQueueManager
}

/*
MQReturn holds the MQRC and MQCC values returned from an MQI verb. It
implements the Error() function so is returned as the specific error
from the verbs. See the sample programs for how to access the
MQRC/MQCC values in this returned error.
*/
type MQReturn struct {
	MQCC int32
	MQRC int32
	verb string
}

func (e *MQReturn) Error() string {
	return mqstrerror(e.verb, C.MQLONG(e.MQCC), C.MQLONG(e.MQRC))
}

var endian binary.ByteOrder // Used by structure formatters such as MQCFH

/*
 * Copy a Go string in "strings"
 * to a fixed-size C char array such as MQCHAR12
 * Once the string has been copied, it can be immediately freed
 * Empty strings have first char set to 0 in MQI structures
 */
func setMQIString(a *C.char, v string, l int) {
	if len(v) > 0 {
		p := C.CString(v)
		C.strncpy(a, p, (C.size_t)(l))
		C.free(unsafe.Pointer(p))
	} else {
		*a = 0
	}
}

/*
 * The C.GoStringN function can return strings that include
 * NUL characters (which is not really what is expected for a C string-related
 * function). So we have a utility function to remove any trailing nulls and spaces
 */
func trimStringN(c *C.char, l C.int) string {
	var rc string
	s := C.GoStringN(c, l)
	i := strings.IndexByte(s, 0)
	if i == -1 {
		rc = s
	} else {
		rc = s[0:i]
	}
	return strings.TrimSpace(rc)
}

/*
Conn is the function to connect to a queue manager
*/
func Conn(goQMgrName string) (MQQueueManager, error) {
	return Connx(goQMgrName, nil)
}

/*
Connx is the extended function to connect to a queue manager.
*/
func Connx(goQMgrName string, gocno *MQCNO) (MQQueueManager, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqcno C.MQCNO

	if (C.MQENC_NATIVE % 2) == 0 { // May be needed for conversion later
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	qMgr := MQQueueManager{}
	mqQMgrName := unsafe.Pointer(C.CString(goQMgrName))
	defer C.free(mqQMgrName)

	// Set up a default CNO if not provided.
	if gocno == nil {
		// Because Go programs are always threaded, and we cannot
		// tell on which thread we might get dispatched, allow handles always to
		// be shareable.
		gocno = NewMQCNO()
		gocno.Options = MQCNO_HANDLE_SHARE_NO_BLOCK
	} else {
		if (gocno.Options & (MQCNO_HANDLE_SHARE_NO_BLOCK |
			MQCNO_HANDLE_SHARE_BLOCK)) == 0 {
			gocno.Options |= MQCNO_HANDLE_SHARE_NO_BLOCK
		}
	}
	copyCNOtoC(&mqcno, gocno)

	C.MQCONNX((*C.MQCHAR)(mqQMgrName), &mqcno, &qMgr.hConn, &mqcc, &mqrc)

	if gocno != nil {
		copyCNOfromC(&mqcno, gocno)
	}

	mqreturn := &MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCONNX",
	}

	if mqcc != C.MQCC_OK {
		return qMgr, mqreturn
	}

	qMgr.Name = goQMgrName

	return qMgr, nil
}

/*
Disc is the function to disconnect from the queue manager
*/
func (x *MQQueueManager) Disc() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	savedConn := x.hConn
	C.MQDISC(&x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDISC",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	cbRemoveConnection(savedConn)

	return nil
}

/*
Open an object such as a queue or topic
*/
func (x *MQQueueManager) Open(good *MQOD, goOpenOptions int32) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqod C.MQOD
	var mqOpenOptions C.MQLONG

	object := MQObject{
		Name: good.ObjectName,
		qMgr: x,
	}

	copyODtoC(&mqod, good)
	mqOpenOptions = C.MQLONG(goOpenOptions) | C.MQOO_FAIL_IF_QUIESCING

	C.MQOPEN(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqod)),
		mqOpenOptions,
		&object.hObj,
		&mqcc,
		&mqrc)

	copyODfromC(&mqod, good)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQOPEN",
	}

	if mqcc != C.MQCC_OK {
		return object, &mqreturn
	}

	// ObjectName may have changed because it's a model queue
	object.Name = good.ObjectName
	if good.ObjectType == C.MQOT_TOPIC {
		object.Name = good.ObjectString
	}

	return object, nil

}

/*
Close the object
*/
func (object *MQObject) Close(goCloseOptions int32) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCloseOptions C.MQLONG

	mqCloseOptions = C.MQLONG(goCloseOptions)

	savedHConn := object.qMgr.hConn
	savedHObj := object.hObj

	C.MQCLOSE(object.qMgr.hConn, &object.hObj, mqCloseOptions, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCLOSE",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	cbRemoveHandle(savedHConn, savedHObj)
	return nil

}

/*
Sub is the function to subscribe to a topic
*/
func (x *MQQueueManager) Sub(gosd *MQSD, qObject *MQObject) (MQObject, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqsd C.MQSD

	subObject := MQObject{
		Name: gosd.ObjectName,
		qMgr: x,
	}

	copySDtoC(&mqsd, gosd)

	C.MQSUB(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqsd)),
		&qObject.hObj,
		&subObject.hObj,
		&mqcc,
		&mqrc)

	copySDfromC(&mqsd, gosd)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSUB",
	}

	if mqcc != C.MQCC_OK {
		return subObject, &mqreturn
	}

	qObject.qMgr = x // Force the correct hConn for managed objects

	return subObject, nil

}

/*
Subrq is the function to request retained publications
*/
func (subObject *MQObject) Subrq(gosro *MQSRO, action int32) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqsro C.MQSRO

	copySROtoC(&mqsro, gosro)

	C.MQSUBRQ(subObject.qMgr.hConn,
		subObject.hObj,
		C.MQLONG(action),
		(C.PMQVOID)(unsafe.Pointer(&mqsro)),
		&mqcc,
		&mqrc)

	copySROfromC(&mqsro, gosro)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSUBRQ",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
Begin is the function to start a two-phase XA transaction coordinated by MQ
*/
func (x *MQQueueManager) Begin(gobo *MQBO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqbo C.MQBO

	copyBOtoC(&mqbo, gobo)

	C.MQBEGIN(x.hConn, (C.PMQVOID)(unsafe.Pointer(&mqbo)), &mqcc, &mqrc)

	copyBOfromC(&mqbo, gobo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQBEGIN",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Cmit is the function to commit an in-flight transaction
*/
func (x *MQQueueManager) Cmit() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	C.MQCMIT(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCMIT",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Back is the function to backout an in-flight transaction
*/
func (x *MQQueueManager) Back() error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	C.MQBACK(x.hConn, &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQBACK",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Stat is the function to check the status after using the asynchronous put
across a client channel
*/
func (x *MQQueueManager) Stat(statusType int32, gosts *MQSTS) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqsts C.MQSTS

	copySTStoC(&mqsts, gosts)

	C.MQSTAT(x.hConn, C.MQLONG(statusType), (C.PMQVOID)(unsafe.Pointer(&mqsts)), &mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSTAT",
	}

	copySTSfromC(&mqsts, gosts)

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Put a message to a queue or publish to a topic
*/
func (object MQObject) Put(gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var ptr C.PMQVOID

	bufflen := len(buffer)

	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
Put1 puts a single messsage to a queue or topic. Typically used for one-shot
replies where it can be cheaper than multiple Open/Put/Close
sequences
*/
func (x *MQQueueManager) Put1(good *MQOD, gomd *MQMD,
	gopmo *MQPMO, buffer []byte) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqpmo C.MQPMO
	var mqod C.MQOD
	var ptr C.PMQVOID

	copyODtoC(&mqod, good)
	copyMDtoC(&mqmd, gomd)
	copyPMOtoC(&mqpmo, gopmo)

	bufflen := len(buffer)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQPUT1(x.hConn, (C.PMQVOID)(unsafe.Pointer(&mqod)),
		(C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqpmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&mqcc, &mqrc)

	copyODfromC(&mqod, good)
	copyMDfromC(&mqmd, gomd)
	copyPMOfromC(&mqpmo, gopmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQPUT1",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil

}

/*
Get a message from a queue
The length of the retrieved message is returned.
*/
func (object MQObject) Get(gomd *MQMD,
	gogmo *MQGMO, buffer []byte) (int, error) {

	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqmd C.MQMD
	var mqgmo C.MQGMO
	var datalen C.MQLONG
	var ptr C.PMQVOID

	bufflen := len(buffer)

	copyMDtoC(&mqmd, gomd)
	copyGMOtoC(&mqgmo, gogmo)

	if bufflen > 0 {
		ptr = (C.PMQVOID)(unsafe.Pointer(&buffer[0]))
	} else {
		ptr = nil
	}

	C.MQGET(object.qMgr.hConn, object.hObj, (C.PMQVOID)(unsafe.Pointer(&mqmd)),
		(C.PMQVOID)(unsafe.Pointer(&mqgmo)),
		(C.MQLONG)(bufflen),
		ptr,
		&datalen,
		&mqcc, &mqrc)

	godatalen := int(datalen)
	copyMDfromC(&mqmd, gomd)
	copyGMOfromC(&mqgmo, gogmo)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQGET",
	}

	if mqcc != C.MQCC_OK {
		return 0, &mqreturn
	}

	return godatalen, nil

}

/*
Inq is the function to inquire on an attribute of an object

Slices are returned containing the integer attributes, and all the
strings concatenated into a single buffer - the caller needs to know
how long each field in that buffer will be.

The caller passes in how many integer selectors are expected to be
returned, as well as the maximum length of the char buffer to be returned

Deprecated: This function is a direct mapping of the MQI C function. It should be considered
deprecated. In preference, use the InqMap function which provides a more convenient
API. In a future version of this package, Inq will be replaced by InqMap
*/
func (object MQObject) Inq(goSelectors []int32, intAttrCount int, charAttrLen int) ([]int32,
	[]byte, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCharAttrs C.PMQCHAR
	var goCharAttrs []byte
	var goIntAttrs []int32
	var ptr C.PMQLONG

	if intAttrCount > 0 {
		goIntAttrs = make([]int32, intAttrCount)
		ptr = (C.PMQLONG)(unsafe.Pointer(&goIntAttrs[0]))
	} else {
		ptr = nil
	}
	if charAttrLen > 0 {
		mqCharAttrs = (C.PMQCHAR)(C.malloc(C.size_t(charAttrLen)))
		defer C.free(unsafe.Pointer(mqCharAttrs))
	} else {
		mqCharAttrs = nil
	}

	// Pass in the selectors directly
	C.MQINQ(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(goSelectors)),
		C.PMQLONG(unsafe.Pointer(&goSelectors[0])),
		C.MQLONG(intAttrCount),
		ptr,
		C.MQLONG(charAttrLen),
		mqCharAttrs,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQ",
	}

	if mqcc != C.MQCC_OK {
		return nil, nil, &mqreturn
	}

	if charAttrLen > 0 {
		goCharAttrs = C.GoBytes(unsafe.Pointer(mqCharAttrs), C.int(charAttrLen))
	}

	return goIntAttrs, goCharAttrs, nil
}

/*
 * InqMap should be considered the replacement for the Inq() function as it
 * has a much simpler API. Simply pass in the list of selectors for the object
 * and the return value consists of a map whose elements are
 * a) accessed via the selector
 * b) varying datatype (integer, string, string array) based on the selector
 * In a future breaking update, this function will become the default Inq()
 * implementation.
 */
func (object MQObject) InqMap(goSelectors []int32) (map[int32]interface{}, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG
	var mqCharAttrs C.PMQCHAR
	var goCharAttrs []byte
	var goIntAttrs []int32
	var ptr C.PMQLONG
	var charOffset int
	var charLength int

	intAttrCount, _, charAttrLen := getAttrInfo(goSelectors)

	if intAttrCount > 0 {
		goIntAttrs = make([]int32, intAttrCount)
		ptr = (C.PMQLONG)(unsafe.Pointer(&goIntAttrs[0]))
	} else {
		ptr = nil
	}
	if charAttrLen > 0 {
		mqCharAttrs = (C.PMQCHAR)(C.malloc(C.size_t(charAttrLen)))
		defer C.free(unsafe.Pointer(mqCharAttrs))
	} else {
		mqCharAttrs = nil
	}

	// Pass in the selectors
	C.MQINQ(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(goSelectors)),
		C.PMQLONG(unsafe.Pointer(&goSelectors[0])),
		C.MQLONG(intAttrCount),
		ptr,
		C.MQLONG(charAttrLen),
		mqCharAttrs,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQ",
	}

	if mqcc != C.MQCC_OK {
		return nil, &mqreturn
	}

	// Create a map of the selectors to the returned values
	returnedMap := make(map[int32]interface{})

	// Get access to the returned character data
	if charAttrLen > 0 {
		goCharAttrs = C.GoBytes(unsafe.Pointer(mqCharAttrs), C.int(charAttrLen))
	}

	// Walk through the returned data to build a map of responses. Go through
	// the integers first to ensure that the map includes MQIA_NAME_COUNT if that
	// had been requested
	intAttr := 0
	for i := 0; i < len(goSelectors); i++ {
		s := goSelectors[i]
		if s >= C.MQIA_FIRST && s <= C.MQIA_LAST {
			returnedMap[s] = goIntAttrs[intAttr]
			intAttr++
		}
	}

	// Now we can walk through the list again for the character attributes
	// and build the map entries. Getting the list of NAMES from a NAMELIST
	// is a bit complicated ...
	charLength = 0
	charOffset = 0
	for i := 0; i < len(goSelectors); i++ {
		s := goSelectors[i]
		if s >= C.MQCA_FIRST && s <= C.MQCA_LAST {
			if s == C.MQCA_NAMES {
				count, ok := returnedMap[C.MQIA_NAME_COUNT]
				if ok {
					c := int(count.(int32))
					charLength = C.MQ_OBJECT_NAME_LENGTH
					names := make([]string, c)
					for j := 0; j < c; j++ {
						name := string(goCharAttrs[charOffset : charOffset+charLength])
						idx := strings.IndexByte(name, 0)
						if idx != -1 {
							name = name[0:idx]
						}
						names[j] = strings.TrimSpace(name)
						charOffset += charLength
					}
					returnedMap[s] = names
				} else {
					charLength = 0
				}
			} else {
				charLength = getAttrLength(s)
				name := string(goCharAttrs[charOffset : charOffset+charLength])
				idx := strings.IndexByte(name, 0)
				if idx != -1 {
					name = name[0:idx]
				}

				returnedMap[s] = strings.TrimSpace(name)
				charOffset += charLength
			}
		}
	}

	return returnedMap, nil
}

/*
 * Set is the function that wraps MQSET. The single parameter is a map whose
 * elements contain an MQIA/MQCA selector with either a string or an int32 for
 * the value.
 */
func (object MQObject) Set(goSelectors map[int32]interface{}) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var charAttrs []byte
	var charAttrsPtr C.PMQCHAR
	var intAttrs []int32
	var intAttrsPtr C.PMQLONG
	var charOffset int
	var charLength int

	// Pass through the map twice. First time lets us
	// create an array of selector names from map keys which is then
	// used to calculate the character buffer that's needed
	selectors := make([]int32, len(goSelectors))
	i := 0
	for k, _ := range goSelectors {
		selectors[i] = k
		i++
	}

	intAttrCount, _, charAttrLen := getAttrInfo(selectors)

	// Create the areas to be used for the separate char and int values
	if intAttrCount > 0 {
		intAttrs = make([]int32, intAttrCount)
		intAttrsPtr = (C.PMQLONG)(unsafe.Pointer(&intAttrs[0]))
	} else {
		intAttrsPtr = nil
	}

	if charAttrLen > 0 {
		charAttrs = make([]byte, charAttrLen)
		charAttrsPtr = (C.PMQCHAR)(unsafe.Pointer(&charAttrs[0]))
	} else {
		charAttrsPtr = nil
	}

	// Walk through the map a second time
	charOffset = 0
	intAttr := 0
	for i := 0; i < len(selectors); i++ {
		s := selectors[i]
		if s >= C.MQCA_FIRST && s <= C.MQCA_LAST {
			// The character processing is a bit OTT since there is in reality
			// only a single attribute that can ever be SET. But a general purpose
			// function looks more like the MQINQ operation
			v := goSelectors[s].(string)
			charLength = getAttrLength(s)
			vBytes := []byte(v)
			b := byte(0)
			for j := 0; j < charLength; j++ {
				if j < len(vBytes) {
					b = vBytes[j]
				} else {
					b = 0
				}
				charAttrs[charOffset+j] = b
			}
			charOffset += charLength
		} else if s >= C.MQIA_FIRST && s <= C.MQIA_LAST {
			vv := int32(0)
			v := goSelectors[s]
			// Force the returned value from the map to be int32 because we
			// can't check it at compile time.
			if _, ok := v.(int32); ok {
				vv = v.(int32)
			} else if _, ok := v.(int); ok {
				vv = int32(v.(int))
			}
			intAttrs[intAttr] = vv
			intAttr++
		}
	}

	// Pass in the selectors
	C.MQSET(object.qMgr.hConn, object.hObj,
		C.MQLONG(len(selectors)),
		C.PMQLONG(unsafe.Pointer(&selectors[0])),
		C.MQLONG(intAttrCount),
		intAttrsPtr,
		C.MQLONG(charAttrLen),
		charAttrsPtr,
		&mqcc, &mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSET",
	}

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*********** Message Handles and Properties  ****************/

/*
CrtMH is the function to create a message handle for holding properties
*/
func (x *MQQueueManager) CrtMH(gocmho *MQCMHO) (MQMessageHandle, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqcmho C.MQCMHO
	var mqhmsg C.MQHMSG

	copyCMHOtoC(&mqcmho, gocmho)

	C.MQCRTMH(x.hConn,
		(C.PMQVOID)(unsafe.Pointer(&mqcmho)),
		(C.PMQHMSG)(unsafe.Pointer(&mqhmsg)),
		&mqcc,
		&mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQCRTMH",
	}

	copyCMHOfromC(&mqcmho, gocmho)
	msgHandle := MQMessageHandle{hMsg: mqhmsg, qMgr: x}

	if mqcc != C.MQCC_OK {
		return msgHandle, &mqreturn
	}

	return msgHandle, nil

}

/*
DltMH is the function to delete a message handle holding properties
*/
func (handle *MQMessageHandle) DltMH(godmho *MQDMHO) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqdmho C.MQDMHO

	copyDMHOtoC(&mqdmho, godmho)

	C.MQDLTMH(handle.qMgr.hConn,
		(C.PMQHMSG)(unsafe.Pointer(&handle.hMsg)),
		(C.PMQVOID)(unsafe.Pointer(&mqdmho)),
		&mqcc,
		&mqrc)

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDLTMH",
	}

	copyDMHOfromC(&mqdmho, godmho)

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	handle.hMsg = C.MQHM_NONE
	return nil
}

/*
SetMP is the function to set a message property. This function allows the
property value to be (almost) any basic datatype - string, int32, int64, []byte
and converts it into the appropriate format for the C MQI.
*/
func (handle *MQMessageHandle) SetMP(gosmpo *MQSMPO, name string, gopd *MQPD, value interface{}) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqsmpo C.MQSMPO
	var mqpd C.MQPD
	var mqName C.MQCHARV

	var propertyLength C.MQLONG
	var propertyType C.MQLONG
	var propertyPtr C.PMQVOID

	var propertyInt32 C.MQLONG
	var propertyInt64 C.MQINT64
	var propertyBool C.MQLONG
	var propertyInt8 C.MQINT8
	var propertyInt16 C.MQINT16
	var propertyFloat32 C.MQFLOAT32
	var propertyFloat64 C.MQFLOAT64

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	}

	propertyType = -1
	if v, ok := value.(int32); ok {
		propertyInt32 = (C.MQLONG)(v)
		propertyType = C.MQTYPE_INT32
		propertyLength = 4
		propertyPtr = (C.PMQVOID)(&propertyInt32)
	} else if v, ok := value.(int64); ok {
		propertyInt64 = (C.MQINT64)(v)
		propertyType = C.MQTYPE_INT64
		propertyLength = 8
		propertyPtr = (C.PMQVOID)(&propertyInt64)
	} else if v, ok := value.(int); ok {
		propertyInt64 = (C.MQINT64)(v)
		propertyType = C.MQTYPE_INT64
		propertyLength = 8
		propertyPtr = (C.PMQVOID)(&propertyInt64)
	} else if v, ok := value.(int8); ok {
		propertyInt8 = (C.MQINT8)(v)
		propertyType = C.MQTYPE_INT8
		propertyLength = 1
		propertyPtr = (C.PMQVOID)(&propertyInt8)
	} else if v, ok := value.(byte); ok { // Separate for int8 and byte (alias uint8)
		propertyInt8 = (C.MQINT8)(v)
		propertyType = C.MQTYPE_INT8
		propertyLength = 1
		propertyPtr = (C.PMQVOID)(&propertyInt8)
	} else if v, ok := value.(int16); ok {
		propertyInt16 = (C.MQINT16)(v)
		propertyType = C.MQTYPE_INT16
		propertyLength = 2
		propertyPtr = (C.PMQVOID)(&propertyInt16)
	} else if v, ok := value.(float32); ok {
		propertyFloat32 = (C.MQFLOAT32)(v)
		propertyType = C.MQTYPE_FLOAT32
		propertyLength = C.sizeof_MQFLOAT32
		propertyPtr = (C.PMQVOID)(&propertyFloat32)
	} else if v, ok := value.(float64); ok {
		propertyFloat64 = (C.MQFLOAT64)(v)
		propertyType = C.MQTYPE_FLOAT64
		propertyLength = C.sizeof_MQFLOAT64
		propertyPtr = (C.PMQVOID)(&propertyFloat64)
	} else if v, ok := value.(string); ok {
		propertyType = C.MQTYPE_STRING
		propertyLength = (C.MQLONG)(len(v))
		propertyPtr = (C.PMQVOID)(C.CString(v))
	} else if v, ok := value.(bool); ok {
		propertyType = C.MQTYPE_BOOLEAN
		propertyLength = 4
		if v {
			propertyBool = 1
		} else {
			propertyBool = 0
		}
		propertyPtr = (C.PMQVOID)(&propertyBool)
	} else if v, ok := value.([]byte); ok {
		propertyType = C.MQTYPE_BYTE_STRING
		propertyLength = (C.MQLONG)(len(v))
		propertyPtr = (C.PMQVOID)(C.malloc(C.size_t(len(v))))
		copy((*[1 << 31]byte)(propertyPtr)[0:len(v)], v)
	} else if v == nil {
		propertyType = C.MQTYPE_NULL
		propertyLength = 0
		propertyPtr = (C.PMQVOID)(C.NULL)
	} else {
		// Unknown datatype - return an error immediately
		mqreturn := MQReturn{MQCC: C.MQCC_FAILED,
			MQRC: C.MQRC_PROPERTY_TYPE_ERROR,
			verb: "MQSETMP",
		}
		return &mqreturn
	}

	copySMPOtoC(&mqsmpo, gosmpo)
	copyPDtoC(&mqpd, gopd)

	C.MQSETMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqsmpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		(C.PMQVOID)(unsafe.Pointer(&mqpd)),
		propertyType,
		propertyLength,
		propertyPtr,
		&mqcc,
		&mqrc)

	if len(name) > 0 {
		C.free(unsafe.Pointer(mqName.VSPtr))
	}

	if propertyType == C.MQTYPE_STRING || propertyType == C.MQTYPE_BYTE_STRING {
		C.free(unsafe.Pointer(propertyPtr))
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQSETMP",
	}

	copySMPOfromC(&mqsmpo, gosmpo)
	copyPDfromC(&mqpd, gopd)

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
DltMP is the function to remove a message property.
*/
func (handle *MQMessageHandle) DltMP(godmpo *MQDMPO, name string) error {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqdmpo C.MQDMPO
	var mqName C.MQCHARV

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	}

	copyDMPOtoC(&mqdmpo, godmpo)

	C.MQDLTMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqdmpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		&mqcc,
		&mqrc)

	if len(name) > 0 {
		C.free(unsafe.Pointer(mqName.VSPtr))
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQDLTMP",
	}

	copyDMPOfromC(&mqdmpo, godmpo)

	if mqcc != C.MQCC_OK {
		return &mqreturn
	}

	return nil
}

/*
InqMP is the function to inquire about the value of a message property.
*/

func (handle *MQMessageHandle) InqMP(goimpo *MQIMPO, gopd *MQPD, name string) (string, interface{}, error) {
	var mqrc C.MQLONG
	var mqcc C.MQLONG

	var mqimpo C.MQIMPO
	var mqpd C.MQPD
	var mqName C.MQCHARV

	var propertyLength C.MQLONG
	var propertyType C.MQLONG
	var propertyPtr C.PMQVOID
	var propertyValue interface{}

	const namebufsize = 1024
	const propbufsize = 10240

	mqName.VSLength = (C.MQLONG)(len(name))
	mqName.VSCCSID = C.MQCCSI_APPL
	if mqName.VSLength > 0 {
		mqName.VSPtr = (C.MQPTR)(C.CString(name))
		mqName.VSBufSize = mqName.VSLength
	} else {
		mqName.VSPtr = (C.MQPTR)(C.malloc(namebufsize))
		mqName.VSBufSize = namebufsize
	}

	copyIMPOtoC(&mqimpo, goimpo)
	copyPDtoC(&mqpd, gopd)

	propertyPtr = C.PMQVOID(C.malloc(propbufsize))
	bufferLength := C.MQLONG(namebufsize)

	C.MQINQMP(handle.qMgr.hConn,
		handle.hMsg,
		(C.PMQVOID)(unsafe.Pointer(&mqimpo)),
		(C.PMQVOID)(unsafe.Pointer(&mqName)),
		(C.PMQVOID)(unsafe.Pointer(&mqpd)),
		(C.PMQLONG)(unsafe.Pointer(&propertyType)),
		bufferLength,
		propertyPtr,
		(C.PMQLONG)(unsafe.Pointer(&propertyLength)),
		&mqcc,
		&mqrc)

	if len(name) > 0 {
		C.free(unsafe.Pointer(mqName.VSPtr))
	}

	mqreturn := MQReturn{MQCC: int32(mqcc),
		MQRC: int32(mqrc),
		verb: "MQINQMP",
	}

	copyIMPOfromC(&mqimpo, goimpo)
	copyPDfromC(&mqpd, gopd)

	if mqcc != C.MQCC_OK {
		return "", nil, &mqreturn
	}

	switch propertyType {
	case C.MQTYPE_INT8:
		p := (*C.MQBYTE)(propertyPtr)
		propertyValue = (int8)(*p)
	case C.MQTYPE_INT16:
		p := (*C.MQBYTE)(propertyPtr)
		propertyValue = (int16)(*p)
	case C.MQTYPE_INT32:
		p := (*C.MQINT16)(propertyPtr)
		propertyValue = (int16)(*p)
	case C.MQTYPE_INT64:
		p := (*C.MQINT64)(propertyPtr)
		propertyValue = (int64)(*p)
	case C.MQTYPE_FLOAT32:
		p := (*C.MQFLOAT32)(propertyPtr)
		propertyValue = (float32)(*p)
	case C.MQTYPE_FLOAT64:
		p := (*C.MQFLOAT64)(propertyPtr)
		propertyValue = (float64)(*p)
	case C.MQTYPE_BOOLEAN:
		p := (*C.MQLONG)(propertyPtr)
		b := (int32)(*p)
		if b == 0 {
			propertyValue = false
		} else {
			propertyValue = true
		}
	case C.MQTYPE_STRING:
		propertyValue = C.GoStringN((*C.char)(propertyPtr), (C.int)(propertyLength))
	case C.MQTYPE_BYTE_STRING:
		ba := make([]byte, propertyLength)
		p := (*C.MQBYTE)(propertyPtr)
		copy(ba[:], C.GoBytes(unsafe.Pointer(p), (C.int)(propertyLength)))
		propertyValue = ba
	case C.MQTYPE_NULL:
		propertyValue = nil
	}

	return goimpo.ReturnedName, propertyValue, nil
}

/*
GetHeader returns a structure containing a parsed-out version of an MQI
message header such as the MQDLH (which is currently the only structure
supported). Other structures like the RFH2 could follow.

The caller of this function needs to cast the returned structure to the
specific type in order to reference the fields.
*/
func GetHeader(md *MQMD, buf []byte) (interface{}, int, error) {
	switch md.Format {
	case MQFMT_DEAD_LETTER_HEADER:
		return getHeaderDLH(md, buf)
	}

	mqreturn := &MQReturn{MQCC: int32(MQCC_FAILED),
		MQRC: int32(MQRC_FORMAT_NOT_SUPPORTED),
	}

	return nil, 0, mqreturn
}

func readStringFromFixedBuffer(r io.Reader, l int32) string {
	tmpBuf := make([]byte, l)
	binary.Read(r, endian, tmpBuf)
	return strings.TrimSpace(string(tmpBuf))
}
