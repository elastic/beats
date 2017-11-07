// Copyright (c) 2012 The gocql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cassandra

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/elastic/beats/libbeat/common/streambuf"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	ErrFrameTooBig = errors.New("frame length is bigger than the maximum allowed")
	debugf         = logp.MakeDebug("cassandra")
)

type frameHeader struct {
	Version       protoVersion
	Flags         byte
	Stream        int
	Op            FrameOp
	BodyLength    int
	HeadLength    int
	CustomPayload map[string][]byte
}

func (f frameHeader) ToMap() map[string]interface{} {
	data := make(map[string]interface{})
	data["version"] = fmt.Sprintf("%d", f.Version.version())
	data["flags"] = getHeadFlagString(f.Flags)
	data["stream"] = f.Stream
	data["op"] = f.Op.String()
	data["length"] = f.BodyLength
	return data
}

func (f frameHeader) String() string {
	return fmt.Sprintf("version:%s, flags: %s, steam: %v, OP: %v, length: %v", f.Version.String(), getHeadFlagString(f.Flags), f.Stream, f.Op.String(), f.BodyLength)
}

var framerPool = sync.Pool{
	New: func() interface{} {
		return &Framer{compres: nil, isCompressed: false, Header: nil, r: nil, decoder: nil}
	},
}

// a framer is responsible for reading, writing and parsing frames on a single stream
type Framer struct {
	proto byte

	compres Compressor

	isCompressed bool

	// if this frame was read then the header will be here
	Header *frameHeader

	r *streambuf.Buffer

	decoder Decoder
}

func NewFramer(r *streambuf.Buffer, compressor Compressor) *Framer {
	f := framerPool.Get().(*Framer)
	f.compres = compressor
	f.r = r

	return f
}

// read header frame from stream
func (f *Framer) ReadHeader() (head *frameHeader, err error) {
	v, err := f.r.ReadByte()
	if err != nil {
		return nil, err
	}
	version := v & protoVersionMask

	if version < protoVersion1 || version > protoVersion4 {
		return nil, fmt.Errorf("unsupported version: %x ", v)
	}

	f.proto = version

	head = &frameHeader{}

	head.Version = protoVersion(v)

	flag, err := f.r.ReadByte()
	if err != nil {
		return nil, err
	}
	head.Flags = flag

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
		head.BodyLength = int(l)
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
		head.BodyLength = int(l)
	}

	if head.BodyLength < 0 {
		return nil, fmt.Errorf("frame body length can not be less than 0: %d", head.BodyLength)
	} else if head.BodyLength > maxFrameSize {
		// need to free up the connection to be used again
		logp.Err("head length is too large")
		return nil, ErrFrameTooBig
	}

	headSize := f.r.BufferConsumed()
	head.HeadLength = headSize

	debugf("header: %v", head)

	f.Header = head
	return head, nil
}

// reads a frame form the wire into the framers buffer
func (f *Framer) ReadFrame() (data map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	decoder := &StreamDecoder{}
	decoder.r = f.r
	f.decoder = decoder

	data = make(map[string]interface{})

	//Only QUERY, PREPARE and EXECUTE queries support tracing
	//If a response frame has the tracing flag set, its body contains
	//a tracing ID. The tracing ID is a [uuid] and is the first thing in
	//the frame body. The rest of the body will then be the usual body
	//corresponding to the response opcode.
	if f.Header.Flags&flagTracing == flagTracing && (f.Header.Op&opQuery == opQuery || f.Header.Op&opExecute == opExecute || f.Header.Op&opPrepare == opPrepare) {

		debugf("tracing enabled")

		//seems no UUID to read, protocol incorrect?
		//uid := decoder.ReadUUID()
		//data["trace_id"] = uid.String()
	}

	if f.Header.Flags&flagWarning == flagWarning {
		debugf("hit warning flags")

		warnings := decoder.ReadStringList()
		// dealing with warnings
		data["warnings"] = warnings
	}

	if f.Header.Flags&flagCustomPayload == flagCustomPayload {
		debugf("hit custom payload flags")

		f.Header.CustomPayload = decoder.ReadBytesMap()
	}

	if f.Header.Flags&flagCompress == flagCompress {
		//decompress data and switch to use bytearray decoder
		if f.compres == nil {
			logp.Err("hit compress flag, but compressor was not set")
			panic(errors.New("hit compress flag, but compressor was not set"))
		}

		decoder := &ByteArrayDecoder{}
		buf := make([]byte, f.Header.BodyLength)
		f.r.Read(buf)
		dec, err := f.compres.Decode(buf)
		if err != nil {
			return nil, err
		}

		decoder.Data = &dec
		f.decoder = decoder

		debugf("hit compress flags")
	}

	// assumes that the frame body has been read into rbuf
	switch f.Header.Op {

	//below ops are requests
	case opStartup, opAuthResponse, opOptions, opPrepare, opExecute, opBatch, opRegister:
	//ignored
	case opQuery:
		data = f.parseQueryFrame()

	//below ops are responses
	case opError:
		data["error"] = f.parseErrorFrame()
	case opResult:
		data["result"] = f.parseResultFrame()
	case opSupported:
		data = f.parseSupportedFrame()
	case opAuthenticate:
		data["authentication"] = f.parseAuthenticateFrame()
	case opAuthChallenge:
		data["authentication"] = f.parseAuthChallengeFrame()
	case opAuthSuccess:
		data["authentication"] = f.parseAuthSuccessFrame()
	case opEvent:
		data["event"] = f.parseEventFrame()
	case opReady:
		// the body should be empty

	default:
		//ignore
		debugf("unknow ops, not processed, %v", f.Header)

	}

	return data, nil
}

func (f *Framer) parseErrorFrame() (data map[string]interface{}) {
	decoder := f.decoder
	code := decoder.ReadInt()
	msg := decoder.ReadString()

	errT := ErrType(code)

	data = make(map[string]interface{})
	data["code"] = code
	data["msg"] = msg
	data["type"] = errT.String()
	detail := map[string]interface{}{}
	switch errT {
	case errUnavailable:
		cl := decoder.ReadConsistency()
		required := decoder.ReadInt()
		alive := decoder.ReadInt()
		detail["read_consistency"] = cl.String()
		detail["required"] = required
		detail["alive"] = alive

	case errWriteTimeout:
		cl := decoder.ReadConsistency()
		received := decoder.ReadInt()
		blockfor := decoder.ReadInt()
		writeType := decoder.ReadString()

		detail["read_consistency"] = cl.String()
		detail["received"] = received
		detail["blockfor"] = blockfor
		detail["write_type"] = writeType

	case errReadTimeout:
		cl := decoder.ReadConsistency()
		received := decoder.ReadInt()
		blockfor := decoder.ReadInt()
		dataPresent, err := decoder.ReadByte()
		if err != nil {
			panic(err)
		}

		detail["read_consistency"] = cl.String()
		detail["received"] = received
		detail["blockfor"] = blockfor
		detail["data_present"] = dataPresent != 0

	case errAlreadyExists:
		ks := decoder.ReadString()
		table := decoder.ReadString()

		detail["keyspace"] = ks
		detail["table"] = table

	case errUnprepared:
		stmtID := decoder.ReadShortBytes()
		detail["stmt_id"] = stmtID

	case errReadFailure:
		detail["read_consistency"] = decoder.ReadConsistency().String()
		detail["received"] = decoder.ReadInt()
		detail["blockfor"] = decoder.ReadInt()
		b, err := decoder.ReadByte()
		if err != nil {
			panic(err)
		}
		detail["data_present"] = b != 0

	case errWriteFailure:
		detail["read_consistency"] = decoder.ReadConsistency().String()
		detail["received"] = decoder.ReadInt()
		detail["blockfor"] = decoder.ReadInt()
		detail["num_failures"] = decoder.ReadInt()
		detail["write_type"] = decoder.ReadString()

	case errFunctionFailure:
		detail["keyspace"] = decoder.ReadString()
		detail["function"] = decoder.ReadString()
		detail["arg_types"] = decoder.ReadStringList()

	case errInvalid, errBootstrapping, errConfig, errCredentials, errOverloaded,
		errProtocol, errServer, errSyntax, errTruncate, errUnauthorized:
		//ignored
	default:
		logp.Err("unknown error code: 0x%x", code)
	}

	if len(detail) > 0 {
		data["details"] = detail
	}

	return data
}

func (f *Framer) parseSupportedFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["supported"] = (f.decoder).ReadStringMultiMap()
	return data
}

func (f *Framer) parseResultMetadata(getPKinfo bool) map[string]interface{} {
	decoder := f.decoder
	meta := make(map[string]interface{})
	flags := decoder.ReadInt()
	meta["flags"] = getRowFlagString(flags)
	colCount := decoder.ReadInt()
	meta["col_count"] = colCount

	if getPKinfo {

		//only for prepared result
		if f.proto >= protoVersion4 {
			pkeyCount := decoder.ReadInt()
			pkeys := make([]int, pkeyCount)
			for i := 0; i < pkeyCount; i++ {
				pkeys[i] = int(decoder.ReadShort())
			}
			meta["pkey_columns"] = pkeys
		}
	}

	if flags&flagHasMorePages == flagHasMorePages {
		meta["paging_state"] = fmt.Sprintf("%X", decoder.ReadBytes())
		return meta
	}

	if flags&flagNoMetaData == flagNoMetaData {
		return meta
	}

	var keyspace, table string
	globalSpec := flags&flagGlobalTableSpec == flagGlobalTableSpec
	if globalSpec {
		keyspace = decoder.ReadString()
		table = decoder.ReadString()
		meta["keyspace"] = keyspace
		meta["table"] = table
	}

	return meta
}

func (f *Framer) parseQueryFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["query"] = (f.decoder).ReadLongString()
	return data
}

func (f *Framer) parseResultFrame() (data map[string]interface{}) {
	kind := (f.decoder).ReadInt()

	data = make(map[string]interface{})
	switch kind {
	case resultKindVoid:
		data["type"] = "void"
	case resultKindRows:
		data["type"] = "rows"
		data["rows"] = f.parseResultRows()
	case resultKindSetKeyspace:
		data["type"] = "set_keyspace"
		data["keyspace"] = (f.decoder).ReadString()
	case resultKindPrepared:
		data["type"] = "prepared"
		data["prepared"] = f.parseResultPrepared()
	case resultKindSchemaChanged:
		data["type"] = "schemaChanged"
		data["schema_change"] = f.parseResultSchemaChange()
	}

	return data
}

func (f *Framer) parseResultRows() map[string]interface{} {
	result := make(map[string]interface{})
	result["meta"] = f.parseResultMetadata(false)
	result["num_rows"] = (f.decoder).ReadInt()

	return result
}

func (f *Framer) parseResultPrepared() map[string]interface{} {
	result := make(map[string]interface{})

	uuid, err := UUIDFromBytes((f.decoder).ReadShortBytes())

	if err != nil {
		logp.Err("Error in parsing UUID")
	}

	result["prepared_id"] = uuid.String()

	result["req_meta"] = f.parseResultMetadata(true)

	if f.proto < protoVersion2 {
		return result
	}

	result["resp_meta"] = f.parseResultMetadata(false)

	return result
}

func (f *Framer) parseResultSchemaChange() (data map[string]interface{}) {
	data = make(map[string]interface{})
	decoder := f.decoder
	if f.proto <= protoVersion2 {
		change := decoder.ReadString()
		keyspace := decoder.ReadString()
		table := decoder.ReadString()

		data["change"] = change
		data["keyspace"] = keyspace
		data["table"] = table
	} else {
		change := decoder.ReadString()
		target := decoder.ReadString()

		data["change"] = change
		data["target"] = target

		switch target {
		case "KEYSPACE":
			data["keyspace"] = decoder.ReadString()

		case "TABLE", "TYPE":
			data["keyspace"] = decoder.ReadString()
			data["object"] = decoder.ReadString()

		case "FUNCTION", "AGGREGATE":
			data["keyspace"] = decoder.ReadString()
			data["name"] = decoder.ReadString()
			data["args"] = decoder.ReadStringList()

		default:
			logp.Warn("unknown SCHEMA_CHANGE target: %q change: %q", target, change)
		}
	}
	return data
}

func (f *Framer) parseAuthenticateFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["class"] = (f.decoder).ReadString()
	return data
}

func (f *Framer) parseAuthSuccessFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", (f.decoder).ReadBytes())
	return data
}

func (f *Framer) parseAuthChallengeFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", (f.decoder).ReadBytes())
	return data
}

func (f *Framer) parseEventFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	decoder := f.decoder
	eventType := decoder.ReadString()
	data["type"] = eventType

	switch eventType {
	case "TOPOLOGY_CHANGE":
		data["change"] = decoder.ReadString()
		host, port := decoder.ReadInet()
		data["host"] = host
		data["port"] = port

	case "STATUS_CHANGE":
		data["change"] = decoder.ReadString()
		host, port := decoder.ReadInet()
		data["host"] = host
		data["port"] = port

	case "SCHEMA_CHANGE":
		// this should work for all versions
		data["schema_change"] = f.parseResultSchemaChange()
	default:
		logp.Err("unknown event type: %q", eventType)
	}

	return data
}
