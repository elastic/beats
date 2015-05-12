// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package layers

import (
	"encoding/binary"
	"github.com/tsg/gopacket"
)

// GRE is a Generic Routing Encapsulation header.
type GRE struct {
	BaseLayer
	ChecksumPresent, RoutingPresent, KeyPresent, SeqPresent, StrictSourceRoute bool
	RecursionControl, Flags, Version                                           uint8
	Protocol                                                                   EthernetType
	Checksum, Offset                                                           uint16
	Key, Seq                                                                   uint32
	*GRERouting
}

// GRERouting is GRE routing information, present if the RoutingPresent flag is
// set.
type GRERouting struct {
	AddressFamily        uint16
	SREOffset, SRELength uint8
	RoutingInformation   []byte
}

// LayerType returns gopacket.LayerTypeGRE.
func (g *GRE) LayerType() gopacket.LayerType { return LayerTypeGRE }

// DecodeFromBytes decodes the given bytes into this layer.
func (g *GRE) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	g.ChecksumPresent = data[0]&0x80 != 0
	g.RoutingPresent = data[0]&0x40 != 0
	g.KeyPresent = data[0]&0x20 != 0
	g.SeqPresent = data[0]&0x10 != 0
	g.StrictSourceRoute = data[0]&0x08 != 0
	g.RecursionControl = data[0] & 0x7
	g.Flags = data[1] >> 3
	g.Version = data[1] & 0x7
	g.Protocol = EthernetType(binary.BigEndian.Uint16(data[2:4]))
	g.Checksum = binary.BigEndian.Uint16(data[4:6])
	g.Offset = binary.BigEndian.Uint16(data[6:8])
	g.Key = binary.BigEndian.Uint32(data[8:12])
	g.Seq = binary.BigEndian.Uint32(data[12:16])
	g.BaseLayer = BaseLayer{data[:16], data[16:]}
	// reset data to point to after the main gre header
	rData := data[16:]
	if g.RoutingPresent {
		g.GRERouting = &GRERouting{
			AddressFamily: binary.BigEndian.Uint16(rData[:2]),
			SREOffset:     rData[2],
			SRELength:     rData[3],
		}
		end := g.SRELength + 4
		g.RoutingInformation = rData[4:end]
		g.Contents = data[:16+end]
		g.Payload = data[16+end:]
	} else {
		g.GRERouting = nil
	}
	return nil
}

// CanDecode returns the set of layer types that this DecodingLayer can decode.
func (g *GRE) CanDecode() gopacket.LayerClass {
	return LayerTypeGRE
}

// NextLayerType returns the layer type contained by this DecodingLayer.
func (g *GRE) NextLayerType() gopacket.LayerType {
	return g.Protocol.LayerType()
}

func decodeGRE(data []byte, p gopacket.PacketBuilder) error {
	g := &GRE{}
	return decodingLayerDecoder(g, data, p)
}
