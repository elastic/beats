package common

type NetString []byte

// implement encoding.TextMarshaller interface to treat []byte as raw string
// by other encoders/serializers (e.g. JSON)

func (n NetString) MarshalText() ([]byte, error) {
	return n, nil
}
