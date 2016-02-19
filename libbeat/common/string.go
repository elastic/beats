package common

// NetString store the byte length of the data that follows, making it easier
// to unambiguously pass text and byte data between programs that could be
// sensitive to values that could be interpreted as delimiters or terminators
// (such as a null character).
type NetString []byte

// MarshalText exists to implement encoding.TextMarshaller interface to
// treat []byte as raw string by other encoders/serializers (e.g. JSON)
func (n NetString) MarshalText() ([]byte, error) {
	return n, nil
}
