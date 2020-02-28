package iana

type HwTypeType uint8

const (
	_ HwTypeType = iota // skip 0
	HwTypeEthernet
	HwTypeExperimentalEthernet
	HwTypeAmateurRadioAX25
	HwTypeProteonTokenRing
	HwTypeChaos
	HwTypeIEEE802
	HwTypeARCNET
	HwTypeHyperchannel
	HwTypeLanstar
	HwTypeAutonet
	HwTypeLocalTalk
	HwTypeLocalNet
	HwTypeUltraLink
	HwTypeSMDS
	HwTypeFrameRelay
	HwTypeATM
	HwTypeHDLC
	HwTypeFibreChannel
	HwTypeATM2
	HwTypeSerialLine
	HwTypeATM3
	HwTypeMILSTD188220
	HwTypeMetricom
	HwTypeIEEE1394
	HwTypeMAPOS
	HwTypeTwinaxial
	HwTypeEUI64
	HwTypeHIPARP
	HwTypeISO7816
	HwTypeARPSec
	HwTypeIPsec
	HwTypeInfiniband
	HwTypeCAI
	HwTypeWiegandInterface
	HwTypePureIP
)

var HwTypeToString = map[HwTypeType]string{
	HwTypeEthernet:             "Ethernet",
	HwTypeExperimentalEthernet: "Experimental Ethernet",
	HwTypeAmateurRadioAX25:     "Amateur Radio AX.25",
	HwTypeProteonTokenRing:     "Proteon ProNET Token Ring",
	HwTypeChaos:                "Chaos",
	HwTypeIEEE802:              "IEEE 802",
	HwTypeARCNET:               "ARCNET",
	HwTypeHyperchannel:         "Hyperchannel",
	HwTypeLanstar:              "Lanstar",
	HwTypeAutonet:              "Autonet Short Address",
	HwTypeLocalTalk:            "LocalTalk",
	HwTypeLocalNet:             "LocalNet",
	HwTypeUltraLink:            "Ultra link",
	HwTypeSMDS:                 "SMDS",
	HwTypeFrameRelay:           "Frame Relay",
	HwTypeATM:                  "ATM",
	HwTypeHDLC:                 "HDLC",
	HwTypeFibreChannel:         "Fibre Channel",
	HwTypeATM2:                 "ATM 2",
	HwTypeSerialLine:           "Serial Line",
	HwTypeATM3:                 "ATM 3",
	HwTypeMILSTD188220:         "MIL-STD-188-220",
	HwTypeMetricom:             "Metricom",
	HwTypeIEEE1394:             "IEEE 1394.1995",
	HwTypeMAPOS:                "MAPOS",
	HwTypeTwinaxial:            "Twinaxial",
	HwTypeEUI64:                "EUI-64",
	HwTypeHIPARP:               "HIPARP",
	HwTypeISO7816:              "IP and ARP over ISO 7816-3",
	HwTypeARPSec:               "ARPSec",
	HwTypeIPsec:                "IPsec tunnel",
	HwTypeInfiniband:           "Infiniband",
	HwTypeCAI:                  "CAI, TIA-102 Project 125 Common Air Interface",
	HwTypeWiegandInterface:     "Wiegand Interface",
	HwTypePureIP:               "Pure IP",
}
