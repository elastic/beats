package decoder

type ipfixDecoder struct {
}

func (d *ipfixDecoder) Decode() ([]byte, error) {
	// Implement IPFIX decoding logic here
	return []byte{}, nil
}

func (d *ipfixDecoder) Close() error {
	// Implement any necessary cleanup logic here
	return nil
}

func (d *ipfixDecoder) More() bool {
	// Implement logic to determine if there are more events to decode
	return false
}

func (d *ipfixDecoder) Next() bool {
	// Implement logic to move to the next event
	return false
}

func NewIPFIXDecoder() (Decoder, error) {
	return &ipfixDecoder{}, nil
}
