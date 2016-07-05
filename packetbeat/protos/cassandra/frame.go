/**
https://github.com/apache/cassandra/blob/trunk/doc/native_protocol_v4.spec

  The CQL binary protocol is a frame based protocol. Frames are defined as:

      0         8        16        24        32         40
      +---------+---------+---------+---------+---------+
      | version |  flags  |      stream       | opcode  |
      +---------+---------+---------+---------+---------+
      |                length                 |
      +---------+---------+---------+---------+
      |                                       |
      .            ...  body ...              .
      .                                       .
      .                                       .
      +----------------------------------------

  The protocol is big-endian (network byte order).

  some code derived from https://github.com/gocql/gocql

*/
package cassandra

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"sync"
)

var (
	ErrFrameTooBig = errors.New("frame length is bigger than the maximum allowed")
)

type frameHeader struct {
	version       protoVersion
	flags         byte
	stream        int
	op            frameOp
	length        int
	customPayload map[string][]byte
}

func (f frameHeader) toMap() map[string]interface{} {
	data := make(map[string]interface{})
	data["version"] = fmt.Sprintf("%d", f.version.version())
	data["flags"] = f.flags
	data["stream"] = f.stream
	data["op"] = f.op.String()
	data["length"] = f.length
	return data
}

func (f frameHeader) String() string {
	return fmt.Sprintf("[header version=%s flags=0x%x stream=%d op=%s length=%d]", f.version.String(), f.flags, f.stream, f.op.String(), f.length)
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
	r io.Reader

	proto byte
	// flags are for outgoing flags, enabling compression and tracing etc
	flags    byte
	compres  Compressor
	headSize int
	// if this frame was read then the header will be here
	header *frameHeader

	// if tracing flag is set this is not nil
	traceID []byte

	// holds a ref to the whole byte slice for rbuf so that it can be reset to
	// 0 after a read.
	readBuffer []byte

	rbuf []byte
}

func newFramer(r io.Reader, compressor Compressor, version byte) *framer {
	f := framerPool.Get().(*framer)
	var flags byte
	if compressor != nil {
		flags |= flagCompress
	}

	version &= protoVersionMask

	headSize := 8
	if version > protoVersion2 {
		headSize = 9
	}

	f.compres = compressor
	f.proto = version
	f.flags = flags
	f.headSize = headSize

	f.r = r
	f.rbuf = f.readBuffer[:0]

	f.header = nil
	f.traceID = nil

	return f
}

type frame interface {
	Header() frameHeader
}

func readHeader(r io.Reader, p []byte) (head frameHeader, err error) {
	_, err = io.ReadFull(r, p[:1])
	if err != nil {
		return frameHeader{}, err
	}

	version := p[0] & protoVersionMask

	if version < protoVersion1 || version > protoVersion4 {
		return frameHeader{}, fmt.Errorf("unsupported response version: %d", version)
	}

	headSize := 9
	if version < protoVersion3 {
		headSize = 8
	}

	_, err = io.ReadFull(r, p[1:headSize])
	if err != nil {
		return frameHeader{}, err
	}

	p = p[:headSize]

	v := p[0]

	head.version = protoVersion(v)

	head.flags = p[1]

	if version > protoVersion2 {
		if len(p) != 9 {
			return frameHeader{}, fmt.Errorf("not enough bytes to read header require 9 got: %d", len(p))
		}

		head.stream = int(int16(p[2])<<8 | int16(p[3]))
		head.op = frameOp(p[4])
		head.length = int(readInt(p[5:]))
	} else {
		if len(p) != 8 {
			return frameHeader{}, fmt.Errorf("not enough bytes to read header require 8 got: %d", len(p))
		}

		head.stream = int(int8(p[2]))
		head.op = frameOp(p[3])
		head.length = int(readInt(p[4:]))
	}

	return head, nil
}

// reads a frame form the wire into the framers buffer
func (f *framer) readFrame(head *frameHeader) error {
	if head.length < 0 {
		return fmt.Errorf("frame body length can not be less than 0: %d", head.length)
	} else if head.length > maxFrameSize {
		// need to free up the connection to be used again
		_, err := io.CopyN(ioutil.Discard, f.r, int64(head.length))
		if err != nil {
			return fmt.Errorf("error whilst trying to discard frame with invalid length: %v", err)
		}
		return ErrFrameTooBig
	}

	if cap(f.readBuffer) >= head.length {
		f.rbuf = f.readBuffer[:head.length]
	} else {
		f.readBuffer = make([]byte, head.length)
		f.rbuf = f.readBuffer
	}

	// assume the underlying reader takes care of timeouts and retries
	n, err := io.ReadFull(f.r, f.rbuf)
	if err != nil {
		return fmt.Errorf("unable to read frame body: read %d/%d bytes: %v", n, head.length, err)
	}

	// dealing with compressed frame body
	if head.flags&flagCompress == flagCompress {
		if f.compres == nil {
			return errors.New("no compressor available with compressed frame body")
		}

		f.rbuf, err = f.compres.Decode(f.rbuf)
		if err != nil {
			return err
		}
	}

	f.header = head
	return nil
}

func (f *framer) parseFrame(msg *message) (data map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	data = make(map[string]interface{})

	data["request_type"] = f.header.op.String()

	if f.header.flags&flagTracing == flagTracing {
		uuid, err := f.readUUID()
		if err != nil {
			f.traceID = uuid.Bytes()
			// dealing with traceId
			data["trace_id"] = uuid.String()
		}

	}

	if f.header.flags&flagWarning == flagWarning {
		warnings := f.readStringList()
		// dealing with warnings
		data["warnings"] = warnings
	}

	if f.header.flags&flagCustomPayload == flagCustomPayload {
		f.header.customPayload = f.readBytesMap()
	}

	// assumes that the frame body has been read into rbuf
	switch f.header.op {
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

	code := f.readInt()
	msg := f.readString()

	errT := ErrType(code)

	data = make(map[string]interface{})
	data["err_code"] = code
	data["err_msg"] = msg
	data["err_type"] = errT.String()

	switch errT {
	case errUnavailable:
		cl := f.readConsistency()
		required := f.readInt()
		alive := f.readInt()
		data["read_consistency"] = cl.String()
		data["required"] = required
		data["alive"] = alive

	case errWriteTimeout:
		cl := f.readConsistency()
		received := f.readInt()
		blockfor := f.readInt()
		writeType := f.readString()

		data["read_consistency"] = cl.String()
		data["received"] = received
		data["blockfor"] = blockfor
		data["write_type"] = writeType

	case errReadTimeout:
		cl := f.readConsistency()
		received := f.readInt()
		blockfor := f.readInt()
		dataPresent := f.readByte()

		data["read_consistency"] = cl.String()
		data["received"] = received
		data["blockfor"] = blockfor
		data["data_present"] = dataPresent

	case errAlreadyExists:
		ks := f.readString()
		table := f.readString()

		data["keyspace"] = ks
		data["table"] = table

	case errUnprepared:
		stmtId := f.readShortBytes()

		data["stmt_id"] = copyBytes(stmtId)

	case errReadFailure:
		data["read_consistency"] = f.readConsistency().String()
		data["received"] = f.readInt()
		data["blockfor"] = f.readInt()
		data["data_present"] = f.readByte() != 0

	case errWriteFailure:
		data["read_consistency"] = f.readConsistency().String()
		data["received"] = f.readInt()
		data["blockfor"] = f.readInt()
		data["num_failures"] = f.readInt()
		data["write_type"] = f.readString()

	case errFunctionFailure:
		data["keyspace"] = f.readString()
		data["function"] = f.readString()
		data["arg_types"] = f.readStringList()

	case errInvalid, errBootstrapping, errConfig, errCredentials, errOverloaded,
		errProtocol, errServer, errSyntax, errTruncate, errUnauthorized:
	default:
		logp.Err("unknown error code: 0x%x", code)
	}
	return data
}

func (f *framer) parseSupportedFrame() (data map[string]interface{}) {

	data = make(map[string]interface{})
	data["supported"] = f.readStringMultiMap()
	return data
}

func (f *framer) parseResultMetadata(getPKinfo bool) map[string]interface{} {

	meta := make(map[string]interface{})
	flags := f.readInt()
	meta["flags"] = flags
	colCount := f.readInt()
	meta["col_count"] = colCount

	if getPKinfo {

		//only for prepared result
		if f.proto >= protoVersion4 {
			pkeyCount := f.readInt()
			pkeys := make([]int, pkeyCount)
			for i := 0; i < pkeyCount; i++ {
				pkeys[i] = int(f.readShort())
			}
			meta["pkey_columns"] = pkeys
		}
	}

	if flags&flagHasMorePages == flagHasMorePages {
		meta["paging_state"] = fmt.Sprintf("%X", f.readBytes())
	}

	if flags&flagNoMetaData == flagNoMetaData {
		return meta
	}

	var keyspace, table string
	globalSpec := flags&flagGlobalTableSpec == flagGlobalTableSpec
	if globalSpec {
		keyspace = f.readString()
		table = f.readString()
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
	data["query"] = string(f.readBytes())
	return data
}

func (f *framer) parseResultFrame() (data map[string]interface{}) {

	kind := f.readInt()

	data = make(map[string]interface{})
	switch kind {
	case resultKindVoid:
		data["result_type"] = "void"
	case resultKindRows:
		data["result_type"] = "rows"
		data["rows"] = f.parseResultRows()
	case resultKindSetKeyspace:
		data["result_type"] = "set_keyspace"
		data["keyspace"] = f.readString()
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
	result["num_rows"] = f.readInt()

	return result
}

func (f *framer) parseResultPrepared() map[string]interface{} {

	result := make(map[string]interface{})

	result["prepared_id"] = string(f.readShortBytes())
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
		change := f.readString()
		keyspace := f.readString()
		table := f.readString()

		data["change"] = change
		data["keyspace"] = keyspace
		data["table"] = table
	} else {
		change := f.readString()
		target := f.readString()

		data["change"] = change
		data["type"] = target

		switch target {
		case "KEYSPACE":
			data["keyspace"] = f.readString()

		case "TABLE", "TYPE":
			data["keyspace"] = f.readString()
			data["object"] = f.readString()

		case "FUNCTION", "AGGREGATE":
			data["keyspace"] = f.readString()
			data["name"] = f.readString()
			data["args"] = f.readStringList()

		default:
			logp.Warn("unknown SCHEMA_CHANGE target: %q change: %q", target, change)
		}
	}
	return data
}

func (f *framer) parseAuthenticateFrame() (data map[string]interface{}) {
	data = make(map[string]interface{})
	data["class"] = f.readString()
	return data
}

func (f *framer) parseAuthSuccessFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", f.readBytes())
	return data
}

func (f *framer) parseAuthChallengeFrame() (data map[string]interface{}) {
	data = make((map[string]interface{}))
	data["data"] = fmt.Sprintf("%q", f.readBytes())
	return data
}

func (f *framer) parseEventFrame() (data map[string]interface{}) {

	data = make((map[string]interface{}))

	eventType := f.readString()
	data["event_type"] = eventType

	switch eventType {
	case "TOPOLOGY_CHANGE":
		data["change"] = f.readString()
		host, port := f.readInet()
		data["host"] = host
		data["port"] = port

	case "STATUS_CHANGE":
		data["change"] = f.readString()
		host, port := f.readInet()
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
		col.Keyspace = f.readString()
		col.Table = f.readString()
	} else {
		col.Keyspace = keyspace
		col.Table = table
	}

	col.Name = f.readString()
	col.TypeInfo = f.readTypeInfo()
}

func (f *framer) readTypeInfo() TypeInfo {

	id := f.readShort()

	simple := NativeType{
		proto: f.proto,
		typ:   Type(id),
	}

	if simple.typ == TypeCustom {
		simple.custom = f.readString()
		if cassType := getApacheCassandraType(simple.custom); cassType != TypeCustom {
			simple.typ = cassType
		}
	}

	switch simple.typ {
	case TypeTuple:
		n := f.readShort()
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
		udt.KeySpace = f.readString()
		udt.Name = f.readString()

		n := f.readShort()
		udt.Elements = make([]UDTField, n)
		for i := 0; i < int(n); i++ {
			field := &udt.Elements[i]
			field.Name = f.readString()
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

func (f *framer) readByte() byte {
	if len(f.rbuf) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to read byte require 1 got: %d", len(f.rbuf)))
	}

	b := f.rbuf[0]
	f.rbuf = f.rbuf[1:]
	return b
}

func (f *framer) readInt() (n int) {
	if len(f.rbuf) < 4 {
		panic(fmt.Errorf("not enough bytes in buffer to read int require 4 got: %d", len(f.rbuf)))
	}

	n = int(int32(f.rbuf[0])<<24 | int32(f.rbuf[1])<<16 | int32(f.rbuf[2])<<8 | int32(f.rbuf[3]))
	f.rbuf = f.rbuf[4:]
	return
}

func (f *framer) readShort() (n uint16) {
	if len(f.rbuf) < 2 {
		panic(fmt.Errorf("not enough bytes in buffer to read short require 2 got: %d", len(f.rbuf)))
	}
	n = uint16(f.rbuf[0])<<8 | uint16(f.rbuf[1])
	f.rbuf = f.rbuf[2:]
	return
}

func (f *framer) readLong() (n int64) {
	if len(f.rbuf) < 8 {
		panic(fmt.Errorf("not enough bytes in buffer to read long require 8 got: %d", len(f.rbuf)))
	}
	n = int64(f.rbuf[0])<<56 | int64(f.rbuf[1])<<48 | int64(f.rbuf[2])<<40 | int64(f.rbuf[3])<<32 |
		int64(f.rbuf[4])<<24 | int64(f.rbuf[5])<<16 | int64(f.rbuf[6])<<8 | int64(f.rbuf[7])
	f.rbuf = f.rbuf[8:]
	return
}

func (f *framer) readString() (s string) {
	size := f.readShort()

	if len(f.rbuf) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to read string require %d got: %d", size, len(f.rbuf)))
	}

	s = string(f.rbuf[:size])
	f.rbuf = f.rbuf[size:]
	return
}

func (f *framer) readLongString() (s string) {
	size := f.readInt()

	if len(f.rbuf) < size {
		panic(fmt.Errorf("not enough bytes in buffer to read long string require %d got: %d", size, len(f.rbuf)))
	}

	s = string(f.rbuf[:size])
	f.rbuf = f.rbuf[size:]
	return
}

func (f *framer) readUUID() (*UUID, error) {
	if len(f.rbuf) < 16 {
		return nil, fmt.Errorf("not enough bytes in buffer to read uuid require %d got: %d", 16, len(f.rbuf))
	}

	u, _ := UUIDFromBytes(f.rbuf[:16])
	f.rbuf = f.rbuf[16:]
	return &u, nil
}

func (f *framer) readStringList() []string {
	size := f.readShort()

	l := make([]string, size)
	for i := 0; i < int(size); i++ {
		l[i] = f.readString()
	}

	return l
}

func (f *framer) readBytesInternal() ([]byte, error) {
	size := f.readInt()
	if size < 0 {
		return nil, nil
	}

	if len(f.rbuf) < size {
		return nil, fmt.Errorf("not enough bytes in buffer to read bytes require %d got: %d", size, len(f.rbuf))
	}

	l := f.rbuf[:size]
	f.rbuf = f.rbuf[size:]

	return l, nil
}

func (f *framer) readBytes() []byte {
	l, err := f.readBytesInternal()
	if err != nil {
		panic(err)
	}

	return l
}

func (f *framer) readShortBytes() []byte {
	size := f.readShort()
	if len(f.rbuf) < int(size) {
		panic(fmt.Errorf("not enough bytes in buffer to read short bytes: require %d got %d", size, len(f.rbuf)))
	}

	l := f.rbuf[:size]
	f.rbuf = f.rbuf[size:]

	return l
}

func (f *framer) readInet() (net.IP, int) {
	if len(f.rbuf) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to read inet size require %d got: %d", 1, len(f.rbuf)))
	}

	size := f.rbuf[0]
	f.rbuf = f.rbuf[1:]

	if !(size == 4 || size == 16) {
		panic(fmt.Errorf("invalid IP size: %d", size))
	}

	if len(f.rbuf) < 1 {
		panic(fmt.Errorf("not enough bytes in buffer to read inet require %d got: %d", size, len(f.rbuf)))
	}

	ip := make([]byte, size)
	copy(ip, f.rbuf[:size])
	f.rbuf = f.rbuf[size:]

	port := f.readInt()
	return net.IP(ip), port
}

func (f *framer) readConsistency() Consistency {
	return Consistency(f.readShort())
}

func (f *framer) readStringMap() map[string]string {
	size := f.readShort()
	m := make(map[string]string)

	for i := 0; i < int(size); i++ {
		k := f.readString()
		v := f.readString()
		m[k] = v
	}

	return m
}

func (f *framer) readBytesMap() map[string][]byte {
	size := f.readShort()
	m := make(map[string][]byte)

	for i := 0; i < int(size); i++ {
		k := f.readString()
		v := f.readBytes()
		m[k] = v
	}

	return m
}

func (f *framer) readStringMultiMap() map[string][]string {
	size := f.readShort()
	m := make(map[string][]string)

	for i := 0; i < int(size); i++ {
		k := f.readString()
		v := f.readStringList()
		m[k] = v
	}
	return m
}

func readInt(p []byte) int32 {
	return int32(p[0])<<24 | int32(p[1])<<16 | int32(p[2])<<8 | int32(p[3])
}

func readShort(p []byte) uint16 {
	return uint16(p[0])<<8 | uint16(p[1])
}
