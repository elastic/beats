package file_integrity

import (
	"os"
	"sync"
	"time"

	"github.com/google/flatbuffers/go"
	"github.com/pkg/errors"

	"github.com/elastic/beats/auditbeat/module/file_integrity/schema"
)

// Requires the Google flatbuffer compiler.
//go:generate flatc --go schema.fbs

var actionMap = map[Action]byte{
	AttributesModified: schema.ActionAttributesModified,
	Created:            schema.ActionCreated,
	Deleted:            schema.ActionDeleted,
	Updated:            schema.ActionUpdated,
	Moved:              schema.ActionMoved,
	ConfigChange:       schema.ActionConfigChanged,
}

var bufferPool sync.Pool

func init() {
	bufferPool.New = func() interface{} {
		return flatbuffers.NewBuilder(1024)
	}
}

// fbGetBuilder returns a Builder that can be used for encoding data. The builder
// should be released by invoking the release function after the encoded bytes
// are no longer in used (i.e. a copy of b.FinishedBytes() has been made).
func fbGetBuilder() (b *flatbuffers.Builder, release func()) {
	b = bufferPool.Get().(*flatbuffers.Builder)
	b.Reset()
	return b, func() { bufferPool.Put(b) }
}

// fbEncodeEvent encodes the given Event to a flatbuffer. The returned bytes
// are a pointer into the Builder's memory.
func fbEncodeEvent(b *flatbuffers.Builder, e *Event) []byte {
	if e == nil {
		return nil
	}

	offset := fbWriteEvent(b, e)
	b.Finish(offset)
	return b.FinishedBytes()
}

func fbWriteHash(b *flatbuffers.Builder, hashes map[HashType]Digest) flatbuffers.UOffsetT {
	if len(hashes) == 0 {
		return 0
	}

	offsets := make(map[HashType]flatbuffers.UOffsetT, len(hashes))
	for name, value := range hashes {
		offsets[name] = b.CreateByteVector(value)
	}

	schema.HashStart(b)
	for hashType, offset := range offsets {
		switch hashType {
		case BLAKE2B_256:
			schema.HashAddBlake2b256(b, offset)
		case BLAKE2B_384:
			schema.HashAddBlake2b384(b, offset)
		case BLAKE2B_512:
			schema.HashAddBlake2b512(b, offset)
		case MD5:
			schema.HashAddMd5(b, offset)
		case SHA1:
			schema.HashAddSha1(b, offset)
		case SHA224:
			schema.HashAddSha224(b, offset)
		case SHA256:
			schema.HashAddSha256(b, offset)
		case SHA384:
			schema.HashAddSha384(b, offset)
		case SHA3_224:
			schema.HashAddSha3224(b, offset)
		case SHA3_256:
			schema.HashAddSha3256(b, offset)
		case SHA3_384:
			schema.HashAddSha3384(b, offset)
		case SHA3_512:
			schema.HashAddSha3512(b, offset)
		case SHA512:
			schema.HashAddSha512(b, offset)
		case SHA512_224:
			schema.HashAddSha512224(b, offset)
		case SHA512_256:
			schema.HashAddSha512256(b, offset)
		}
	}
	return schema.HashEnd(b)
}

func fbWriteMetadata(b *flatbuffers.Builder, m *Metadata) flatbuffers.UOffsetT {
	if m == nil {
		return 0
	}

	var sidOffset flatbuffers.UOffsetT
	if m.SID != "" {
		sidOffset = b.CreateString(m.SID)
	}

	schema.MetadataStart(b)
	schema.MetadataAddInode(b, m.Inode)
	schema.MetadataAddUid(b, m.UID)
	schema.MetadataAddGid(b, m.GID)
	if sidOffset > 0 {
		schema.MetadataAddSid(b, sidOffset)
	}
	mode := m.Mode
	if m.SetUID {
		mode |= os.ModeSetuid
	}
	if m.SetGID {
		mode |= os.ModeSetgid
	}
	schema.MetadataAddMode(b, uint32(mode))
	switch m.Type {
	case UnknownType:
		schema.MetadataAddType(b, schema.TypeUnknown)
	case FileType:
		schema.MetadataAddType(b, schema.TypeFile)

		// This info is only used for files.
		schema.MetadataAddSize(b, m.Size)
		schema.MetadataAddMtimeNs(b, m.MTime.UnixNano())
		schema.MetadataAddCtimeNs(b, m.CTime.UnixNano())
	case DirType:
		schema.MetadataAddType(b, schema.TypeDir)
	case SymlinkType:
		schema.MetadataAddType(b, schema.TypeSymlink)
	}
	return schema.MetadataEnd(b)
}

func fbWriteEvent(b *flatbuffers.Builder, e *Event) flatbuffers.UOffsetT {
	if e == nil {
		return 0
	}

	hashesOffset := fbWriteHash(b, e.Hashes)
	metadataOffset := fbWriteMetadata(b, e.Info)

	var targetPathOffset flatbuffers.UOffsetT
	if e.TargetPath != "" {
		targetPathOffset = b.CreateString(e.TargetPath)
	}

	schema.EventStart(b)
	schema.EventAddTimestampNs(b, e.Timestamp.UnixNano())

	switch e.Source {
	case SourceFSNotify:
		schema.EventAddSource(b, schema.SourceFSNotify)
	case SourceScan:
		schema.EventAddSource(b, schema.SourceScan)
	}

	if targetPathOffset > 0 {
		schema.EventAddTargetPath(b, targetPathOffset)
	}

	var action byte
	for k, v := range actionMap {
		if 0 != e.Action&k {
			action |= v
		}
	}
	schema.EventAddAction(b, action)

	if metadataOffset > 0 {
		schema.EventAddInfo(b, metadataOffset)
	}
	if hashesOffset > 0 {
		schema.EventAddHashes(b, hashesOffset)
	}

	return schema.EventEnd(b)
}

// fbDecodeEvent decodes flatbuffer event data and copies it into an Event
// object that is returned.
func fbDecodeEvent(path string, buf []byte) *Event {
	e := schema.GetRootAsEvent(buf, 0)

	rtn := &Event{
		Timestamp:  time.Unix(0, e.TimestampNs()).UTC(),
		Path:       path,
		TargetPath: string(e.TargetPath()),
	}

	switch e.Source() {
	case schema.SourceScan:
		rtn.Source = SourceScan
	case schema.SourceFSNotify:
		rtn.Source = SourceFSNotify
	}

	action := e.Action()
	for k, v := range actionMap {
		if 0 != action&v {
			rtn.Action |= k
		}
	}

	rtn.Info = fbDecodeMetadata(e)
	rtn.Hashes = fbDecodeHash(e)

	return rtn
}

func fbDecodeMetadata(e *schema.Event) *Metadata {
	info := e.Info(nil)
	if info == nil {
		return nil
	}
	mode := os.FileMode(info.Mode())
	rtn := &Metadata{
		Inode:  info.Inode(),
		UID:    info.Uid(),
		GID:    info.Gid(),
		SID:    string(info.Sid()),
		Mode:   mode & ^(os.ModeSetuid | os.ModeSetgid),
		Size:   info.Size(),
		MTime:  time.Unix(0, info.MtimeNs()).UTC(),
		CTime:  time.Unix(0, info.CtimeNs()).UTC(),
		SetUID: mode&os.ModeSetuid != 0,
		SetGID: mode&os.ModeSetgid != 0,
	}

	switch info.Type() {
	case schema.TypeFile:
		rtn.Type = FileType
	case schema.TypeDir:
		rtn.Type = DirType
	case schema.TypeSymlink:
		rtn.Type = SymlinkType
	default:
		rtn.Type = UnknownType
	}

	return rtn
}

func fbDecodeHash(e *schema.Event) map[HashType]Digest {
	hash := e.Hashes(nil)
	if hash == nil {
		return nil
	}

	rtn := map[HashType]Digest{}
	for _, hashType := range validHashes {
		var length int
		var producer func(i int) int8

		switch hashType {
		case BLAKE2B_256:
			length = hash.Blake2b256Length()
			producer = hash.Blake2b256
		case BLAKE2B_384:
			length = hash.Blake2b384Length()
			producer = hash.Blake2b384
		case BLAKE2B_512:
			length = hash.Blake2b512Length()
			producer = hash.Blake2b512
		case MD5:
			length = hash.Md5Length()
			producer = hash.Md5
		case SHA1:
			length = hash.Sha1Length()
			producer = hash.Sha1
		case SHA224:
			length = hash.Sha224Length()
			producer = hash.Sha224
		case SHA256:
			length = hash.Sha256Length()
			producer = hash.Sha256
		case SHA384:
			length = hash.Sha384Length()
			producer = hash.Sha384
		case SHA3_224:
			length = hash.Sha3224Length()
			producer = hash.Sha3224
		case SHA3_256:
			length = hash.Sha3256Length()
			producer = hash.Sha3256
		case SHA3_384:
			length = hash.Sha3384Length()
			producer = hash.Sha3384
		case SHA3_512:
			length = hash.Sha3512Length()
			producer = hash.Sha3512
		case SHA512:
			length = hash.Sha512Length()
			producer = hash.Sha512
		case SHA512_224:
			length = hash.Sha512224Length()
			producer = hash.Sha512224
		case SHA512_256:
			length = hash.Sha512256Length()
			producer = hash.Sha512256
		default:
			panic(errors.Errorf("unhandled hash type: %v", hashType))
		}

		if length > 0 {
			hashValue := make([]byte, length)
			for i := 0; i < len(hashValue); i++ {
				hashValue[i] = byte(producer(i))
			}

			rtn[hashType] = hashValue
		}
	}

	return rtn
}

// fbIsEventTimestampBefore returns true if the event's timestamp is before
// the given ts. This convenience function allows you to compare the event's
// timestamp without fully decoding and copying the flatbuffer event data.
func fbIsEventTimestampBefore(buf []byte, ts time.Time) bool {
	e := schema.GetRootAsEvent(buf, 0)
	eventTime := time.Unix(0, e.TimestampNs())
	return eventTime.Before(ts)
}
