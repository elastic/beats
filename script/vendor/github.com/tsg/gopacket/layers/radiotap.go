// Copyright 2014 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package layers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/tsg/gopacket"
)

// align calculates the number of bytes needed to align with the width
// on the offset, returning the number of bytes we need to skip to
// align to the offset (width).
func align(offset uint16, width uint16) uint16 {
	return ((((offset) + ((width) - 1)) & (^((width) - 1))) - offset)
}

type RadioTapPresent uint32

const (
	RadioTapPresentTSFT RadioTapPresent = 1 << iota
	RadioTapPresentFlags
	RadioTapPresentRate
	RadioTapPresentChannel
	RadioTapPresentFHSS
	RadioTapPresentDBMAntennaSignal
	RadioTapPresentDBMAntennaNoise
	RadioTapPresentLockQuality
	RadioTapPresentTxAttenuation
	RadioTapPresentDBTxAttenuation
	RadioTapPresentDBMTxPower
	RadioTapPresentAntenna
	RadioTapPresentDBAntennaSignal
	RadioTapPresentDBAntennaNoise
	RadioTapPresentRxFlags
	RadioTapPresentTxFlags
	RadioTapPresentRtsRetries
	RadioTapPresentDataRetries
	RadioTapPresentEXT RadioTapPresent = 1 << 31
)

func (r RadioTapPresent) TSFT() bool {
	return r&RadioTapPresentTSFT != 0
}
func (r RadioTapPresent) Flags() bool {
	return r&RadioTapPresentFlags != 0
}
func (r RadioTapPresent) Rate() bool {
	return r&RadioTapPresentRate != 0
}
func (r RadioTapPresent) Channel() bool {
	return r&RadioTapPresentChannel != 0
}
func (r RadioTapPresent) FHSS() bool {
	return r&RadioTapPresentFHSS != 0
}
func (r RadioTapPresent) DBMAntennaSignal() bool {
	return r&RadioTapPresentDBMAntennaSignal != 0
}
func (r RadioTapPresent) DBMAntennaNoise() bool {
	return r&RadioTapPresentDBMAntennaNoise != 0
}
func (r RadioTapPresent) LockQuality() bool {
	return r&RadioTapPresentLockQuality != 0
}
func (r RadioTapPresent) TxAttenuation() bool {
	return r&RadioTapPresentTxAttenuation != 0
}
func (r RadioTapPresent) DBTxAttenuation() bool {
	return r&RadioTapPresentDBTxAttenuation != 0
}
func (r RadioTapPresent) DBMTxPower() bool {
	return r&RadioTapPresentDBMTxPower != 0
}
func (r RadioTapPresent) Antenna() bool {
	return r&RadioTapPresentAntenna != 0
}
func (r RadioTapPresent) DBAntennaSignal() bool {
	return r&RadioTapPresentDBAntennaSignal != 0
}
func (r RadioTapPresent) DBAntennaNoise() bool {
	return r&RadioTapPresentDBAntennaNoise != 0
}
func (r RadioTapPresent) RxFlags() bool {
	return r&RadioTapPresentRxFlags != 0
}
func (r RadioTapPresent) TxFlags() bool {
	return r&RadioTapPresentTxFlags != 0
}
func (r RadioTapPresent) RtsRetries() bool {
	return r&RadioTapPresentRtsRetries != 0
}
func (r RadioTapPresent) DataRetries() bool {
	return r&RadioTapPresentDataRetries != 0
}
func (r RadioTapPresent) EXT() bool {
	return r&RadioTapPresentEXT != 0
}

type RadioTapChannelFlags uint16

const (
	RadioTapChannelFlagsTurbo   RadioTapChannelFlags = 0x0010 // Turbo channel
	RadioTapChannelFlagsCCK     RadioTapChannelFlags = 0x0020 // CCK channel
	RadioTapChannelFlagsOFDM    RadioTapChannelFlags = 0x0040 // OFDM channel
	RadioTapChannelFlagsGhz2    RadioTapChannelFlags = 0x0080 // 2 GHz spectrum channel.
	RadioTapChannelFlagsGhz5    RadioTapChannelFlags = 0x0100 // 5 GHz spectrum channel
	RadioTapChannelFlagsPassive RadioTapChannelFlags = 0x0200 // Only passive scan allowed
	RadioTapChannelFlagsDynamic RadioTapChannelFlags = 0x0400 // Dynamic CCK-OFDM channel
	RadioTapChannelFlagsGFSK    RadioTapChannelFlags = 0x0800 // GFSK channel (FHSS PHY)
)

func (r RadioTapChannelFlags) Turbo() bool {
	return r&RadioTapChannelFlagsTurbo != 0
}
func (r RadioTapChannelFlags) CCK() bool {
	return r&RadioTapChannelFlagsCCK != 0
}
func (r RadioTapChannelFlags) OFDM() bool {
	return r&RadioTapChannelFlagsOFDM != 0
}
func (r RadioTapChannelFlags) Ghz2() bool {
	return r&RadioTapChannelFlagsGhz2 != 0
}
func (r RadioTapChannelFlags) Ghz5() bool {
	return r&RadioTapChannelFlagsGhz5 != 0
}
func (r RadioTapChannelFlags) Passive() bool {
	return r&RadioTapChannelFlagsPassive != 0
}
func (r RadioTapChannelFlags) Dynamic() bool {
	return r&RadioTapChannelFlagsDynamic != 0
}
func (r RadioTapChannelFlags) GFSK() bool {
	return r&RadioTapChannelFlagsGFSK != 0
}

// String provides a human readable string for RadioTapChannelFlags.
// This string is possibly subject to change over time; if you're storing this
// persistently, you should probably store the RadioTapChannelFlags value, not its string.
func (a RadioTapChannelFlags) String() string {
	var out bytes.Buffer
	if a.Turbo() {
		out.WriteString("Turbo,")
	}
	if a.CCK() {
		out.WriteString("CCK,")
	}
	if a.OFDM() {
		out.WriteString("OFDM,")
	}
	if a.Ghz2() {
		out.WriteString("Ghz2,")
	}
	if a.Ghz5() {
		out.WriteString("Ghz5,")
	}
	if a.Passive() {
		out.WriteString("Passive,")
	}
	if a.Dynamic() {
		out.WriteString("Dynamic,")
	}
	if a.GFSK() {
		out.WriteString("GFSK,")
	}

	if length := out.Len(); length > 0 {
		return string(out.Bytes()[:length-1]) // strip final comma
	}
	return ""
}

type RadioTapFlags uint8

const (
	RadioTapFlagsCFP           RadioTapFlags = 1 << iota // sent/received during CFP
	RadioTapFlagsShortPreamble                           // sent/received * with short * preamble
	RadioTapFlagsWEP                                     // sent/received * with WEP encryption
	RadioTapFlagsFrag                                    // sent/received * with fragmentation
	RadioTapFlagsFCS                                     // frame includes FCS
	RadioTapFlagsDatapad                                 // frame has padding between * 802.11 header and payload * (to 32-bit boundary)
	RadioTapFlagsBadFCS                                  // does not pass FCS check
	RadioTapFlagsShortGI                                 // HT short GI
)

func (r RadioTapFlags) CFP() bool {
	return r&RadioTapFlagsCFP != 0
}
func (r RadioTapFlags) ShortPreamble() bool {
	return r&RadioTapFlagsShortPreamble != 0
}
func (r RadioTapFlags) WEP() bool {
	return r&RadioTapFlagsWEP != 0
}
func (r RadioTapFlags) Frag() bool {
	return r&RadioTapFlagsFrag != 0
}
func (r RadioTapFlags) FCS() bool {
	return r&RadioTapFlagsFCS != 0
}
func (r RadioTapFlags) Datapad() bool {
	return r&RadioTapFlagsDatapad != 0
}
func (r RadioTapFlags) BadFCS() bool {
	return r&RadioTapFlagsBadFCS != 0
}
func (r RadioTapFlags) ShortGI() bool {
	return r&RadioTapFlagsShortGI != 0
}

// String provides a human readable string for RadioTapFlags.
// This string is possibly subject to change over time; if you're storing this
// persistently, you should probably store the RadioTapFlags value, not its string.
func (a RadioTapFlags) String() string {
	var out bytes.Buffer
	if a.CFP() {
		out.WriteString("CFP,")
	}
	if a.ShortPreamble() {
		out.WriteString("SHORT-PREAMBLE,")
	}
	if a.WEP() {
		out.WriteString("WEP,")
	}
	if a.Frag() {
		out.WriteString("FRAG,")
	}
	if a.FCS() {
		out.WriteString("FCS,")
	}
	if a.Datapad() {
		out.WriteString("DATAPAD,")
	}
	if a.ShortGI() {
		out.WriteString("SHORT-GI,")
	}

	if length := out.Len(); length > 0 {
		return string(out.Bytes()[:length-1]) // strip final comma
	}
	return ""
}

type RadioTapRate uint8

func (a RadioTapRate) String() string {
	return fmt.Sprintf("%v Mb/s", 0.5*float32(a))
}

type RadioTapChannelFrequency uint16

func (a RadioTapChannelFrequency) String() string {
	return fmt.Sprintf("%d MHz", a)
}

func decodeRadioTap(data []byte, p gopacket.PacketBuilder) error {
	d := &RadioTap{}
	// TODO: Should we set LinkLayer here? And implement LinkFlow
	return decodingLayerDecoder(d, data, p)
}

type RadioTap struct {
	BaseLayer

	// Version 0. Only increases for drastic changes, introduction of compatible new fields does not count.
	Version uint8
	// Length of the whole header in bytes, including it_version, it_pad, it_len, and data fields.
	Length uint16
	// Present is a bitmap telling which fields are present. Set bit 31 (0x80000000) to extend the bitmap by another 32 bits. Additional extensions are made by setting bit 31.
	Present RadioTapPresent
	// TSFT: value in microseconds of the MAC's 64-bit 802.11 Time Synchronization Function timer when the first bit of the MPDU arrived at the MAC. For received frames, only.
	TSFT  uint64
	Flags RadioTapFlags
	// Rate Tx/Rx data rate
	Rate RadioTapRate
	// ChannelFrequency Tx/Rx frequency in MHz, followed by flags
	ChannelFrequency RadioTapChannelFrequency
	ChannelFlags     RadioTapChannelFlags
	// FHSS For frequency-hopping radios, the hop set (first byte) and pattern (second byte).
	FHSS uint16
	// DBMAntennaSignal RF signal power at the antenna, decibel difference from one milliwatt.
	DBMAntennaSignal int8
	// DBMAntennaNoise RF noise power at the antenna, decibel difference from one milliwatt.
	DBMAntennaNoise int8
	// LockQuality Quality of Barker code lock. Unitless. Monotonically nondecreasing with "better" lock strength. Called "Signal Quality" in datasheets.
	LockQuality uint16
	// TxAttenuation Transmit power expressed as unitless distance from max power set at factory calibration.  0 is max power. Monotonically nondecreasing with lower power levels.
	TxAttenuation uint16
	// DBTxAttenuation Transmit power expressed as decibel distance from max power set at factory calibration.  0 is max power.  Monotonically nondecreasing with lower power levels.
	DBTxAttenuation uint16
	// DBMTxPower Transmit power expressed as dBm (decibels from a 1 milliwatt reference). This is the absolute power level measured at the antenna port.
	DBMTxPower int8
	// Antenna Unitless indication of the Rx/Tx antenna for this packet. The first antenna is antenna 0.
	Antenna uint8
	// DBAntennaSignal RF signal power at the antenna, decibel difference from an arbitrary, fixed reference.
	DBAntennaSignal uint8
	// DBAntennaNoise RF noise power at the antenna, decibel difference from an arbitrary, fixed reference point.
	DBAntennaNoise uint8
}

func (m *RadioTap) LayerType() gopacket.LayerType { return LayerTypeRadioTap }

func (m *RadioTap) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	m.Version = uint8(data[0])
	m.Length = binary.LittleEndian.Uint16(data[2:4])
	m.Present = RadioTapPresent(binary.LittleEndian.Uint32(data[4:8]))

	offset := uint16(4)

	for (binary.LittleEndian.Uint32(data[offset:offset+4]) & 0x80000000) != 0 {
		// Extended bitmap.
		offset += 4
	}

	if m.Present.TSFT() {
		offset += align(offset, 8)
		m.TSFT = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8
	}
	if m.Present.Flags() {
		m.Flags = RadioTapFlags(data[offset])
		offset++
	}
	if m.Present.Rate() {
		m.Rate = RadioTapRate(data[offset])
		offset++
	}
	if m.Present.FHSS() {
		m.FHSS = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
	}
	if m.Present.Channel() {
		m.ChannelFrequency = RadioTapChannelFrequency(binary.LittleEndian.Uint16(data[offset : offset+2]))
		offset += 2
		m.ChannelFlags = RadioTapChannelFlags(data[offset])
		offset++
	}
	if m.Present.DBMAntennaSignal() {
		m.DBMAntennaSignal = int8(data[offset])
		offset++
	}
	if m.Present.DBMAntennaNoise() {
		m.DBMAntennaNoise = int8(data[offset])
		offset++
	}
	if m.Present.LockQuality() {
		offset += align(offset, 2)
		m.LockQuality = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
	}
	if m.Present.TxAttenuation() {
		offset += align(offset, 2)
		m.TxAttenuation = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
	}
	if m.Present.DBTxAttenuation() {
		offset += align(offset, 2)
		m.DBTxAttenuation = binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
	}
	if m.Present.DBMTxPower() {
		m.DBMTxPower = int8(data[offset])
		offset++
	}
	if m.Present.Antenna() {
		m.Antenna = uint8(data[offset])
		offset++
	}
	if m.Present.DBAntennaSignal() {
		m.DBAntennaSignal = uint8(data[offset])
		offset++
	}
	if m.Present.DBAntennaNoise() {
		m.DBAntennaNoise = uint8(data[offset])
		offset++
	}
	if m.Present.RxFlags() {
		// TODO: Implement RxFlags
	}
	if m.Present.TxFlags() {
		// TODO: Implement TxFlags
	}
	if m.Present.RtsRetries() {
		// TODO: Implement RtsRetries
	}
	if m.Present.DataRetries() {
		// TODO: Implement DataRetries
	}
	if m.Present.EXT() {
		offset += align(offset, 4)
		// TODO: Implement EXT
		_ = data[offset : offset+4]
		offset += 4
	}

	if m.Flags.Datapad() {
		// frame has padding between 802.11 header and payload (to 32-bit boundary)
		offset += align(offset, 4)
	}

	m.BaseLayer = BaseLayer{Contents: data[:m.Length], Payload: data[m.Length:]}

	return nil
}

func (m *RadioTap) CanDecode() gopacket.LayerClass    { return LayerTypeRadioTap }
func (m *RadioTap) NextLayerType() gopacket.LayerType { return LayerTypeDot11 }
