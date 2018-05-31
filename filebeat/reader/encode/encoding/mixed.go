package encoding

import "golang.org/x/text/encoding"

// mixed encoder is a copy of encoding.Replacement
// The difference between the two is that for the Decoder the Encoder is used
// The reasons is that during decoding UTF-8 we want to have the behaviour of the encoding,
// means copying all and replacing invalid UTF-8 chars.
type mixed struct{}

func (mixed) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: encoding.Replacement.NewEncoder().Transformer}
}

func (mixed) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: encoding.Replacement.NewEncoder().Transformer}
}
