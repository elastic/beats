// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package decoder

import "github.com/google/gopacket"

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
