package decoder

import "github.com/tsg/gopacket"

// implement DecodingLayer with support of switching between multiple layers to
// remember outter layer results
type multiLayer struct {
	layers []gopacket.DecodingLayer // all layers must have same type
	i      int
	cnt    int
}

func (m *multiLayer) next() {
	m.i++
	m.cnt++
	if m.i >= len(m.layers) {
		m.i = 0
	}
}

func (m *multiLayer) init(layer ...gopacket.DecodingLayer) {
	m.layers = layer
}

func (m *multiLayer) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	return m.layers[m.i].DecodeFromBytes(data, df)
}

func (m *multiLayer) CanDecode() gopacket.LayerClass {
	return m.layers[m.i].CanDecode()
}

func (m *multiLayer) NextLayerType() gopacket.LayerType {
	return m.layers[m.i].NextLayerType()
}

func (m *multiLayer) LayerPayload() []byte {
	return m.layers[m.i].LayerPayload()
}
