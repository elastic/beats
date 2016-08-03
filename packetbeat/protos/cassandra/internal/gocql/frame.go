// Copyright (c) 2012 The gocql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cassandra

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
	"sync"
)

var (
	ErrFrameLength = errors.New("frame body length can not be less than 0")
	ErrFrameTooBig = errors.New("frame length is bigger than the maximum allowed")
	isDebug        = false
)

type frameHeader struct {
	Version       protoVersion
	Flags         byte
	Stream        int
	Op            FrameOp
	Length        int
	CustomPayload map[string][]byte
}

func (f frameHeader) ToMap() map[string]interface{} {
	data := make(map[string]interface{})
	data["version"] = fmt.Sprintf("%d", f.Version.version())
	data["flags"] = f.Flags
	data["stream"] = f.Stream
	data["op"] = f.Op.String()
	data["length"] = f.Length
	return data
}

func (f frameHeader) String() string {
	return fmt.Sprintf("[header version=%s flags=0x%x stream=%d op=%s length=%d]", f.Version.String(), f.Flags, f.Stream, f.Op.String(), f.Length)
}

func (f frameHeader) Header() frameHeader {
	return f
}

const defaultBufSize = 128

var framerPool = sync.Pool{
	New: func() interface{} {
		return &framer{
			readBuffer: make([]byte, defaultBufSize),
		}
	},
}

// a framer is responsible for reading, writing and parsing frames on a single stream
type framer struct {
	proto byte

	// flags are for outgoing flags, enabling compression and tracing etc
	flags byte

	compres Compressor

	isCompressed bool

	headSize int
	// if this frame was read then the header will be here
	header *frameHeader

	// holds a ref to the whole byte slice for rbuf so that it can be reset to
	// 0 after a read.
	readBuffer []byte

	r *streambuf.Buffer

	decoder Decoder
}

func NewFramer(r *streambuf.Buffer, compressor Compressor) *framer {

	f := framerPool.Get().(*framer)
	f.compres = compressor
	f.r = r

	return f
}

// read header frame from stream
func (f *framer) ReadHeader() (head *frameHeader, err error) {
	v, err := f.r.ReadByte()
	if err != nil {
		return nil, err
	}
	version := v & protoVersionMask

	if version < protoVersion1 || version > protoVersion4 {
		return nil, fmt.Errorf("unsupported response version: %d", version)
	}
	//fmt.Printf("Version Byte: %x \n",v)

	head = &frameHeader{}

	head.Version = protoVersion(v)

	head.Flags, err = f.r.ReadByte()

	if version > protoVersion2 {
		stream, err := f.r.ReadNetUint16()
		if err != nil {
			return nil, err
		}
		head.Stream = int(stream)

		b, err := f.r.ReadByte()
		if err != nil {
			return nil, err
		}

		head.Op = FrameOp(b)
		l, err := f.r.ReadNetUint32()
		if err != nil {
			return nil, err
		}
		head.Length = int(l)
	} else {
		stream, err := f.r.ReadNetUint8()
		if err != nil {
			return nil, err
		}
		head.Stream = int(stream)

		b, err := f.r.ReadByte()
		if err != nil {
			return nil, err
		}

		head.Op = FrameOp(b)
		l, err := f.r.ReadNetUint32()
		if err != nil {
			return nil, err
		}
		head.Length = int(l)
	}

	if head.Length < 0 {
		return nil, fmt.Errorf("frame body length can not be less than 0: %d", head.Length)
	} else if head.Length > maxFrameSize {
		// need to free up the connection to be used again
		logp.Err("head length is too large")
		return nil, ErrFrameTooBig
	}

	if !f.r.Avail(head.Length) {
		return nil, errors.New(fmt.Sprintf("frame length is not enough as expected length: %v", head.Length))
	}

	logp.Debug("cassandra", "header: %v", head)
	f.header = head
	return head, nil
}

// reads a frame form the wire into the framers buffer
func (f *framer) ReadFrame() (data map[string]interface{}, err error) {

	//defer func() {
	//	if r := recover(); r != nil {
	//		if _, ok := r.(runtime.Error); ok {
	//			panic(r)
	//		}
	//		err = r.(error)
	//	}
	//}()

	if f.header.Length < 0 {
		return nil, ErrFrameLength
	} else if f.header.Length > maxFrameSize {
		return nil, ErrFrameTooBig
	}

	var flags byte
	version := byte(f.header.Version)

	if f.compres != nil {
		flags |= flagCompress
	}

	version &= protoVersionMask

	headSize := 8
	if version > protoVersion2 {
		headSize = 9
	}

	f.proto = version
	f.flags = flags
	f.headSize = headSize

	decoder := &StreamDecoder{}
	decoder.r = f.r
	f.decoder = decoder

	data = make(map[string]interface{})

	//Only QUERY, PREPARE and EXECUTE queries support tracing
	//If a response frame has the tracing flag set, its body contains
	//a tracing ID. The tracing ID is a [uuid] and is the first thing in
	//the frame body. The rest of the body will then be the usual body
	//corresponding to the response opcode.
	if f.header.Flags&flagTracing == flagTracing && (f.header.Op == opQuery || f.header.Op == opExecute || f.header.Op == opPrepare) {

		logp.Debug("cassandra", "tracing enabled")

		uid := f.decoder.ReadUUID()

		logp.Debug("cassandra", uid.String())

		data["trace_id"] = uid.String()
	}

	if f.header.Flags&flagWarning == flagWarning {
		warnings := f.decoder.ReadStringList()
		// dealing with warnings
		data["warnings"] = warnings
	}

	if f.header.Flags&flagCustomPayload == flagCustomPayload {
		f.header.CustomPayload = f.decoder.ReadBytesMap()
	}

	data["request_type"] = f.header.Op.String()

	// assumes that the frame body has been read into rbuf
	switch f.header.Op {
	case opError:
		data = f.parseErrorFrame()
	case opReady:
		// the body should be empty
	case opQuery:
		data = f.parseQueryFrame()
	case opResult:
		data = f.parseResultFrame()
	case opSupported:
		data = f.parseSupportedFrame()
	case opAuthenticate:
		data = f.parseAuthenticateFrame()
	case opAuthChallenge:
		data = f.parseAuthChallengeFrame()
	case opAuthSuccess:
		data = f.parseAuthSuccessFrame()
	case opEvent:
		data = f.parseEventFrame()
	case opOptions:
	case opStartup:
	default:
		return nil, errors.New(fmt.Sprintf("unknown op and not parsed,%s", f.header))
	}

	return data, nil
}

func (f *framer) parseErrorFrame() (data map[string]interface{}) {

	code := f.decoder.ReadInt()
	msg := f.decoder.ReadString()

	errT := ErrType(code)

	data = make(map[string]interface{})
	data["err_code"] = code
	data["err_msg"] = msg
	data["err_type"] = errT.String()

	switch errT {
	case errUnavailable:
		cl := f.decoder.ReadConsistency()
		required := f.decoder.ReadInt()
		alive := f.decoder.ReadInt()
		data["read_consistency"] = cl.String()
		data["required"] = required
		data["alive"] = alive

	case errWriteTimeout:
		cl := f.decoder.ReadConsistency()
		received := f.decoder.ReadInt()
		blockfor := f.decoder.ReadInt()
		writeType := f.decoder.ReadString()

		data["read_consistency"] = cl.String()
		data["received"] = received
		data["blockfor"] = blockfor
		data["write_type"] = writeType

	case errReadTimeout:
		cl := f.decoder.ReadConsistency()
		received := f.decoder.ReadInt()
		blockfor := f.decoder.ReadInt()
		dataPresent := f.decoder.ReadByte()

		data["read_consistency"] = cl.String()
		data["received"] = received
		data["blockfor"] = blockfor
		data["data_present"] = dataPresent

	case errAlreadyExists:
		ks := f.decoder.ReadString()
		table := f.decoder.ReadString()

		data["keyspace"] = ks
		data["table"] = table

	case errUnprepared:
		stmtId := f.decoder.ReadShortBytes()

		data["stmt_id"] = copyBytes(stmtId)

	case errReadFailure:
		data["read_consistency"] = f.decoder.ReadConsistency().String()
		data["received"] = f.decoder.ReadInt()
		data["blockfor"] = f.decoder.ReadInt()
		data["data_present"] = f.decoder.ReadByte() != 0

	case errWriteFailure:
		data["read_consistency"] = f.decoder.ReadConsistency().String()
		data["received"] = f.decoder.ReadInt()
		data["blockfor"] = f.decoder.ReadInt()
		data["num_failures"] = f.decoder.ReadInt()
		data["write_type"] = f.decoder.ReadString()

	case errFunctionFailure:
		data["keyspace"] = f.decoder.ReadString()
		data["function"] = f.decoder.ReadString()
		data["arg_types"] = f.decoder.ReadStringList()

	case errInvalid, errBootstrapping, errConfig, errCredentials, errOverloaded,
		errProtocol, errServer, errSyntax, errTruncate, errUnauthorized:
	default:
		logp.Err("unknown error code: 0x%x", code)
	}
	return data
}

func (f *framer) parseSupportedFrame() (data map[string]interface{}) {

	data = make(map[string]interface{})
	data["supported"] = f.decoder.ReadStringMultiMap()
	return data
}

func (f *framer) parseResultMetadata(getPKinfo bool) map[string]interface{} {

	meta := make(map[string]interface{})
	flags := f.decoder.ReadInt()
	meta["flags"] = flags
	colCount := f.decoder.ReadInt()
	meta["col_count"] = colCount

	if getPKinfo {

		//only for prepared result
		if f.proto >= protoVersion4 {
			pkeyCount := f.decoder.ReadInt()
			pkeys := make([]int, pkeyCount)
			for i := 0; i < pkeyCount; i++ {
				pkeys[i] = int(f.decoder.ReadShort())
			}
			meta["pkey_columns"] = pkeys
		}
	}

	if flags&flagHasMorePages == flagHasMorePages {
		meta["paging_state"] = fmt.Sprintf("%X", f.decoder.ReadBytes())
	}

	if flags&flagNoMetaData == flagNoMetaData {
		return meta
	}

	var keyspace, table string
	globalSpec := flags&flagGlobalTableSpec == flagGlobalTableSpec
	if globalSpec {
		keyspace = f.decoder.ReadString()
		table = f.decoder.ReadString()
		meta["keyspace"] = keyspace
		meta["table"] = table
	}

	var cols []ColumnInfo
	if colCount < 1000 {
		// preallocate columninfo to avoid excess copying
		cols = make([]ColumnInfo, colCount)
		for i := 0; i < colCount; i++ {
			f.readCol(&cols[i], globalSpec, keyspace, table)
		}

	} else {
		// use append, huge number of columns usually indicates a corrupt frame or
		// just a huge row.
		for i := 0; i < colCount; i++ {
			var col ColumnInfo
			f.readCol(&col, globalSpec, keyspace, table)
			cols = append(cols, col)
		}
	}

	return meta
}

func (f *framer) parseQueryFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["query"] = f.decoder.ReadLongString()
	return data
}

func (f *framer) parseResultFrame() (data map[string]interface{}) {

	kind := f.decoder.ReadInt()

	data = make(map[string]interface{})
	switch kind {
	case resultKindVoid:
		data["result_type"] = "void"
	case resultKindRows:
		data["result_type"] = "rows"
		data["rows"] = f.parseResultRows()
	case resultKindSetKeyspace:
		data["result_type"] = "set_keyspace"
		data["keyspace"] = f.decoder.ReadString()
	case resultKindPrepared:
		data["result_type"] = "prepared"
		data["result"] = f.parseResultPrepared()
	case resultKindSchemaChanged:
		data["result_type"] = "schemaChanged"
		data["result"] = f.parseResultSchemaChange()
	}

	return data
}

func (f *framer) parseResultRows() map[string]interface{} {

	result := make(map[string]interface{})
	result["meta"] = f.parseResultMetadata(false)
	result["num_rows"] = f.decoder.ReadInt()

	return result
}

func (f *framer) parseResultPrepared() map[string]interface{} {

	result := make(map[string]interface{})

	result["prepared_id"] = string(f.decoder.ReadShortBytes())
	result["req_meta"] = f.parseResultMetadata(true)

	if f.proto < protoVersion2 {
		return result
	}

	result["resp_meta"] = f.parseResultMetadata(false)

	return result
}

func (f *framer) parseResultSchemaChange() (data map[string]interface{}) {
	data = make(map[string]interface{})

	if f.proto <= protoVersion2 {
		change := f.decoder.ReadString()
		keyspace := f.decoder.ReadString()
		table := f.decoder.ReadString()

		data["change"] = change
		data["keyspace"] = keyspace
		data["table"] = table
	} else {
		change := f.decoder.ReadString()
		target := f.decoder.ReadString()

		data["change"] = change
		data["type"] = target

		switch target {
		case "KEYSPACE":
			data["keyspace"] = f.decoder.ReadString()

		case "TABLE", "TYPE":
			data["keyspace"] = f.decoder.ReadString()
			data["object"] = f.decoder.ReadString()

		case "FUNCTION", "AGGREGATE":
			data["keyspace"] = f.decoder.ReadString()
			data["name"] = f.decoder.ReadString()
			data["args"] = f.decoder.ReadStringList()

		default:
			logp.Warn("unknown SCHEMA_CHANGE target: %q change: %q", target, change)
		}
	}
	return data
}

func (f *framer) parseAuthenticateFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["class"] = f.decoder.ReadString()
	return data
}

func (f *framer) parseAuthSuccessFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", f.decoder.ReadBytes())
	return data
}

func (f *framer) parseAuthChallengeFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", f.decoder.ReadBytes())
	return data
}

func (f *framer) parseEventFrame() (data map[string]interface{}) {

	data = make((map[string]interface{}))

	eventType := f.decoder.ReadString()
	data["event_type"] = eventType

	switch eventType {
	case "TOPOLOGY_CHANGE":
		data["change"] = f.decoder.ReadString()
		host, port := f.decoder.ReadInet()
		data["host"] = host
		data["port"] = port

	case "STATUS_CHANGE":
		data["change"] = f.decoder.ReadString()
		host, port := f.decoder.ReadInet()
		data["host"] = host
		data["port"] = port

	case "SCHEMA_CHANGE":
		// this should work for all versions
		data = f.parseResultSchemaChange()
	default:
		logp.Err("unknown event type: %q", eventType)
	}

	return data
}

// explicitly enables tracing for the framers outgoing requests
func (f *framer) trace() {
	f.flags |= flagTracing
}

type UUID [16]byte

// Bytes returns the raw byte slice for this UUID. A UUID is always 128 bits
// (16 bytes) long.
func (u UUID) Bytes() []byte {
	return u[:]
}

// String returns the UUID in it's canonical form, a 32 digit hexadecimal
// number in the form of xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func (u UUID) String() string {
	var offsets = [...]int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
	const hexString = "0123456789abcdef"
	r := make([]byte, 36)
	for i, b := range u {
		r[offsets[i]] = hexString[b>>4]
		r[offsets[i]+1] = hexString[b&0xF]
	}
	r[8] = '-'
	r[13] = '-'
	r[18] = '-'
	r[23] = '-'
	return string(r)

}

// UUIDFromBytes converts a raw byte slice to an UUID.
func UUIDFromBytes(input []byte) (UUID, error) {
	var u UUID
	if len(input) != 16 {
		return u, errors.New("UUIDs must be exactly 16 bytes long")
	}

	copy(u[:], input)
	return u, nil
}

func copyBytes(p []byte) []byte {
	b := make([]byte, len(p))
	copy(b, p)
	return b
}

type ColumnInfo struct {
	Keyspace string
	Table    string
	Name     string
	TypeInfo TypeInfo
}

func (f *framer) readCol(col *ColumnInfo, globalSpec bool, keyspace, table string) {
	if !globalSpec {
		col.Keyspace = f.decoder.ReadString()
		col.Table = f.decoder.ReadString()
	} else {
		col.Keyspace = keyspace
		col.Table = table
	}

	col.Name = f.decoder.ReadString()
	col.TypeInfo = f.readTypeInfo()
}

func (f *framer) readTypeInfo() TypeInfo {

	id := f.decoder.ReadShort()

	simple := NativeType{
		proto: f.proto,
		typ:   Type(id),
	}

	if simple.typ == TypeCustom {
		simple.custom = f.decoder.ReadString()
		if cassType := getApacheCassandraType(simple.custom); cassType != TypeCustom {
			simple.typ = cassType
		}
	}

	switch simple.typ {
	case TypeTuple:
		n := f.decoder.ReadShort()
		tuple := TupleTypeInfo{
			NativeType: simple,
			Elems:      make([]TypeInfo, n),
		}

		for i := 0; i < int(n); i++ {
			tuple.Elems[i] = f.readTypeInfo()
		}

		return tuple

	case TypeUDT:
		udt := UDTTypeInfo{
			NativeType: simple,
		}
		udt.KeySpace = f.decoder.ReadString()
		udt.Name = f.decoder.ReadString()

		n := f.decoder.ReadShort()
		udt.Elements = make([]UDTField, n)
		for i := 0; i < int(n); i++ {
			field := &udt.Elements[i]
			field.Name = f.decoder.ReadString()
			field.Type = f.readTypeInfo()
		}

		return udt
	case TypeMap, TypeList, TypeSet:
		collection := CollectionType{
			NativeType: simple,
		}

		if simple.typ == TypeMap {
			collection.Key = f.readTypeInfo()
		}

		collection.Elem = f.readTypeInfo()

		return collection
	}

	return simple
}

func readInt(p []byte) int32 {
	return int32(p[0])<<24 | int32(p[1])<<16 | int32(p[2])<<8 | int32(p[3])
}

func readShort(p []byte) uint16 {
	return uint16(p[0])<<8 | uint16(p[1])
}
