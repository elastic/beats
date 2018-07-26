package converter

import "golang.org/x/text/transform"

func init() {
	err := register("utf-8", newUTF8Converter)
	if err != nil {
		panic(err)
	}
}

type UTF8Converter struct{}

func newUTF8Converter(t transform.Transformer, size int) (Converter, error) {
	return &UTF8Converter{}, nil
}

func (c *UTF8Converter) Convert(in, out []byte) (int, int, error) {
	n := copy(out, in)
	return n, n, nil
}

func (c *UTF8Converter) MsgSize(symlen []uint8, size int) (int, []uint8, error) {
	return size, nil, nil
}

func (c *UTF8Converter) GetSymLen() []uint8 {
	return nil
}

func (c *UTF8Converter) Collect(out []byte) (int, error) {
	return 0, nil
}
