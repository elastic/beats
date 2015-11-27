package encoding

import "golang.org/x/text/transform"

type mixedEncoding struct {
	decoder func() transform.Transformer
	encoder func() transform.Transformer
}

func (m mixedEncoding) NewDecoder() transform.Transformer {
	return m.decoder()
}

func (m mixedEncoding) NewEncoder() transform.Transformer {
	return m.encoder()
}
