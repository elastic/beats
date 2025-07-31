package encoding

import (
	"fmt"
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type binaryEncoding struct {
	data []byte
}

func BinaryEncoding(in io.Reader) (Encoding, error) {
	// Binary encoding does not require any transformation
	fmt.Println("Creating Binary Encoding")

	buff := make([]byte, 2048)
	n, err := in.Read(buff)

	fmt.Println("4 bytes from input:", buff[:4])

	fmt.Println("Read bytes from input:", n, "Error:", err)
	if err != nil {
		fmt.Println("Error reading from input:", err)
	}

	if len(buff) == 0 {
		return nil, fmt.Errorf("binary encoding requires non-empty input")
	}
	return NewBinaryEncoding(buff[:n]), nil
}

func NewBinaryEncoding(data []byte) Encoding {
	fmt.Println("Creating new Binary Encoding")
	return &binaryEncoding{
		data: data,
	}
}

// func (b *binaryEncoding) Transform(in io.Reader) (io.Reader, error) {
// 	fmt.Println("Transforming input with Binary Encoding")
// 	if len(b.data) == 0 {
// 		return nil, fmt.Errorf("binary encoding requires non-empty data")
// 	}
// 	// No transformation needed for binary encoding, just return the input reader
// 	fmt.Println("Returning input reader without transformation")
// 	if in == nil {
// 		return nil, fmt.Errorf("input reader cannot be nil for binary encoding")
// 	}
// 	// Return the input reader as is, since binary encoding does not change the data
// 	fmt.Println("Returning input reader as is for binary encoding")
// 	if _, err := in.Read(b.data); err != nil && err != io.EOF {
// 		return nil, fmt.Errorf("error reading from input reader: %w", err)
// 	}

// 	reader := io.NopCloser(transform.NewReader(in, transform.Nop))
// 	return reader, nil
// }

func (b *binaryEncoding) NewDecoder() *encoding.Decoder {
	fmt.Println("Creating new Binary Decoder")
	return &encoding.Decoder{
		Transformer: transform.Nop,
	}
}

func (b *binaryEncoding) NewEncoder() *encoding.Encoder {
	fmt.Println("Creating new Binary Encoder")
	return &encoding.Encoder{
		Transformer: transform.Nop,
	}
}
