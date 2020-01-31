package msgpack // import "gopkg.in/vmihailenco/msgpack.v2"

// Deprecated. Use CustomEncoder.
type Marshaler interface {
	MarshalMsgpack() ([]byte, error)
}

// Deprecated. Use CustomDecoder.
type Unmarshaler interface {
	UnmarshalMsgpack([]byte) error
}

type CustomEncoder interface {
	EncodeMsgpack(*Encoder) error
}

type CustomDecoder interface {
	DecodeMsgpack(*Decoder) error
}
