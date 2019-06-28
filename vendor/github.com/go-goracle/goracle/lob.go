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
#include "dpiImpl.h"
*/
import "C"
import (
	//"fmt"
	"io"
	"unicode/utf8"
	"unsafe"

	"github.com/pkg/errors"
)

// Lob is for reading/writing a LOB.
type Lob struct {
	io.Reader
	IsClob bool
}

// Hijack the underlying lob reader/writer, and
// return a DirectLob for reading/writing the lob directly.
//
// After this, the Lob is unusable!
func (lob *Lob) Hijack() (*DirectLob, error) {
	if lob == nil || lob.Reader == nil {
		return nil, errors.New("lob is nil")
	}
	lr, ok := lob.Reader.(*dpiLobReader)
	if !ok {
		return nil, errors.Errorf("Lob.Reader is %T, not *dpiLobReader", lob.Reader)
	}
	lob.Reader = nil
	return &DirectLob{conn: lr.conn, dpiLob: lr.dpiLob}, nil
}

// Scan assigns a value from a database driver.
//
// The src value will be of one of the following types:
//
//    int64
//    float64
//    bool
//    []byte
//    string
//    time.Time
//    nil - for NULL values
//
// An error should be returned if the value cannot be stored
// without loss of information.
func (dlr *dpiLobReader) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return errors.Errorf("cannot convert LOB to %T", src)
	}
	_ = b
	return nil
}

var _ = io.Reader((*dpiLobReader)(nil))

type dpiLobReader struct {
	*conn
	dpiLob              *C.dpiLob
	offset, sizePlusOne C.uint64_t
	finished            bool
	IsClob              bool
}

func (dlr *dpiLobReader) Read(p []byte) (int, error) {
	if dlr == nil {
		return 0, errors.New("read on nil dpiLobReader")
	}
	if dlr.finished {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	// For CLOB, sizePlusOne and offset counts the CHARACTERS!
	// See https://oracle.github.io/odpi/doc/public_functions/dpiLob.html dpiLob_readBytes
	if dlr.sizePlusOne == 0 {
		// never read size before
		if C.dpiLob_getSize(dlr.dpiLob, &dlr.sizePlusOne) == C.DPI_FAILURE {
			return 0, errors.Wrap(dlr.getError(), "getSize")
		}
		dlr.sizePlusOne++
	}
	n := C.uint64_t(len(p))
	//fmt.Printf("%p.Read offset=%d sizePlusOne=%d n=%d\n", dlr.dpiLob, dlr.offset, dlr.sizePlusOne, n)
	if dlr.offset+1 >= dlr.sizePlusOne {
		return 0, io.EOF
	}
	if C.dpiLob_readBytes(dlr.dpiLob, dlr.offset+1, n, (*C.char)(unsafe.Pointer(&p[0])), &n) == C.DPI_FAILURE {
		err := dlr.getError()
		if dlr.finished = err.(interface{ Code() int }).Code() == 1403; dlr.finished {
			dlr.offset += n
			return int(n), io.EOF
		}
		return int(n), errors.Wrapf(err, "lob=%p offset=%d n=%d", dlr.dpiLob, dlr.offset, len(p))
	}
	//fmt.Printf("read %d\n", n)
	if dlr.IsClob {
		dlr.offset += C.uint64_t(utf8.RuneCount(p[:n]))
	} else {
		dlr.offset += n
	}
	var err error
	if n == 0 || dlr.offset+1 >= dlr.sizePlusOne {
		err = io.EOF
	}
	return int(n), err
}

type dpiLobWriter struct {
	*conn
	dpiLob *C.dpiLob
	offset C.uint64_t
	opened bool
	isClob bool
}

func (dlw *dpiLobWriter) Write(p []byte) (int, error) {
	lob := dlw.dpiLob
	if !dlw.opened {
		//fmt.Printf("open %p\n", lob)
		if C.dpiLob_openResource(lob) == C.DPI_FAILURE {
			return 0, errors.Wrapf(dlw.getError(), "openResources(%p)", lob)
		}
		dlw.opened = true
	}

	n := C.uint64_t(len(p))
	if C.dpiLob_writeBytes(lob, dlw.offset+1, (*C.char)(unsafe.Pointer(&p[0])), n) == C.DPI_FAILURE {
		err := errors.Wrapf(dlw.getError(), "writeBytes(%p, offset=%d, data=%d)", lob, dlw.offset, n)
		dlw.dpiLob = nil
		C.dpiLob_closeResource(lob)
		return 0, err
	}
	//fmt.Printf("written %q into %p@%d\n", p[:n], lob, dlw.offset)
	dlw.offset += n

	return int(n), nil
}

func (dlw *dpiLobWriter) Close() error {
	if dlw == nil || dlw.dpiLob == nil {
		return nil
	}
	lob := dlw.dpiLob
	dlw.dpiLob = nil
	//C.dpiLob_flushBuffer(lob)
	if C.dpiLob_closeResource(lob) == C.DPI_FAILURE {
		err := dlw.getError()
		if ec, ok := err.(interface{ Code() int }); ok && !dlw.opened && ec.Code() == 22289 { // cannot perform %s operation on an unopened file or LOB
			return nil
		}
		return errors.Wrapf(err, "closeResource(%p)", lob)
	}
	return nil
}

// DirectLob holds a Lob and allows direct (Read/WriteAt, not streaming Read/Write) operations on it.
type DirectLob struct {
	conn   *conn
	dpiLob *C.dpiLob
	opened bool
}

var _ = io.ReaderAt((*DirectLob)(nil))
var _ = io.WriterAt((*DirectLob)(nil))

// Close the Lob.
func (dl *DirectLob) Close() error {
	if !dl.opened {
		return nil
	}
	dl.opened = false
	if C.dpiLob_closeResource(dl.dpiLob) == C.DPI_FAILURE {
		return errors.Wrap(dl.conn.getError(), "closeResource")
	}
	return nil
}

// Size returns the size of the LOB.
func (dl *DirectLob) Size() (int64, error) {
	var n C.uint64_t
	if C.dpiLob_getSize(dl.dpiLob, &n) == C.DPI_FAILURE {
		return int64(n), errors.Wrap(dl.conn.getError(), "getSize")
	}
	return int64(n), nil
}

// Trim the LOB to the given size.
func (dl *DirectLob) Trim(size int64) error {
	if C.dpiLob_trim(dl.dpiLob, C.uint64_t(size)) == C.DPI_FAILURE {
		return errors.Wrap(dl.conn.getError(), "trim")
	}
	return nil
}

// Set the contents of the LOB to the given byte slice.
// The LOB is cleared first.
func (dl *DirectLob) Set(p []byte) error {
	if C.dpiLob_setFromBytes(dl.dpiLob, (*C.char)(unsafe.Pointer(&p[0])), C.uint64_t(len(p))) == C.DPI_FAILURE {
		return errors.Wrap(dl.conn.getError(), "setFromBytes")
	}
	return nil
}

// ReadAt reads at most len(p) bytes into p at offset.
func (dl *DirectLob) ReadAt(p []byte, offset int64) (int, error) {
	n := C.uint64_t(len(p))
	if C.dpiLob_readBytes(dl.dpiLob, C.uint64_t(offset)+1, n, (*C.char)(unsafe.Pointer(&p[0])), &n) == C.DPI_FAILURE {
		return int(n), errors.Wrap(dl.conn.getError(), "readBytes")
	}
	return int(n), nil
}

// WriteAt writes p starting at offset.
func (dl *DirectLob) WriteAt(p []byte, offset int64) (int, error) {
	if !dl.opened {
		//fmt.Printf("open %p\n", lob)
		if C.dpiLob_openResource(dl.dpiLob) == C.DPI_FAILURE {
			return 0, errors.Wrapf(dl.conn.getError(), "openResources(%p)", dl.dpiLob)
		}
		dl.opened = true
	}

	n := C.uint64_t(len(p))
	if C.dpiLob_writeBytes(dl.dpiLob, C.uint64_t(offset)+1, (*C.char)(unsafe.Pointer(&p[0])), n) == C.DPI_FAILURE {
		return int(n), errors.Wrap(dl.conn.getError(), "writeBytes")
	}
	return int(n), nil
}
