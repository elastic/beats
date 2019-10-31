package dhcpv4

// values from http://www.networksorcery.com/enp/protocol/dhcp.htm and
// http://www.networksorcery.com/enp/protocol/bootp/options.htm

// MessageType represents the possible DHCP message types - DISCOVER, OFFER, etc
type MessageType byte

// DHCP message types
const (
	MessageTypeDiscover MessageType = 1
	MessageTypeOffer    MessageType = 2
	MessageTypeRequest  MessageType = 3
	MessageTypeDecline  MessageType = 4
	MessageTypeAck      MessageType = 5
	MessageTypeNak      MessageType = 6
	MessageTypeRelease  MessageType = 7
	MessageTypeInform   MessageType = 8
)

// MessageTypeToString maps DHCP message types to human-readable strings.
var MessageTypeToString = map[MessageType]string{
	MessageTypeDiscover: "DISCOVER",
	MessageTypeOffer:    "OFFER",
	MessageTypeRequest:  "REQUEST",
	MessageTypeDecline:  "DECLINE",
	MessageTypeAck:      "ACK",
	MessageTypeNak:      "NAK",
	MessageTypeRelease:  "RELEASE",
	MessageTypeInform:   "INFORM",
}

// OpcodeType represents a DHCPv4 opcode.
type OpcodeType uint8

// constants that represent valid values for OpcodeType
const (
	OpcodeBootRequest OpcodeType = 1
	OpcodeBootReply   OpcodeType = 2
)

// OpcodeToString maps an OpcodeType to its mnemonic name
var OpcodeToString = map[OpcodeType]string{
	OpcodeBootRequest: "BootRequest",
	OpcodeBootReply:   "BootReply",
}

// DHCPv4 Options
const (
	OptionPad                                        OptionCode = 0
	OptionSubnetMask                                 OptionCode = 1
	OptionTimeOffset                                 OptionCode = 2
	OptionRouter                                     OptionCode = 3
	OptionTimeServer                                 OptionCode = 4
	OptionNameServer                                 OptionCode = 5
	OptionDomainNameServer                           OptionCode = 6
	OptionLogServer                                  OptionCode = 7
	OptionQuoteServer                                OptionCode = 8
	OptionLPRServer                                  OptionCode = 9
	OptionImpressServer                              OptionCode = 10
	OptionResourceLocationServer                     OptionCode = 11
	OptionHostName                                   OptionCode = 12
	OptionBootFileSize                               OptionCode = 13
	OptionMeritDumpFile                              OptionCode = 14
	OptionDomainName                                 OptionCode = 15
	OptionSwapServer                                 OptionCode = 16
	OptionRootPath                                   OptionCode = 17
	OptionExtensionsPath                             OptionCode = 18
	OptionIPForwarding                               OptionCode = 19
	OptionNonLocalSourceRouting                      OptionCode = 20
	OptionPolicyFilter                               OptionCode = 21
	OptionMaximumDatagramAssemblySize                OptionCode = 22
	OptionDefaultIPTTL                               OptionCode = 23
	OptionPathMTUAgingTimeout                        OptionCode = 24
	OptionPathMTUPlateauTable                        OptionCode = 25
	OptionInterfaceMTU                               OptionCode = 26
	OptionAllSubnetsAreLocal                         OptionCode = 27
	OptionBroadcastAddress                           OptionCode = 28
	OptionPerformMaskDiscovery                       OptionCode = 29
	OptionMaskSupplier                               OptionCode = 30
	OptionPerformRouterDiscovery                     OptionCode = 31
	OptionRouterSolicitationAddress                  OptionCode = 32
	OptionStaticRoutingTable                         OptionCode = 33
	OptionTrailerEncapsulation                       OptionCode = 34
	OptionArpCacheTimeout                            OptionCode = 35
	OptionEthernetEncapsulation                      OptionCode = 36
	OptionDefaulTCPTTL                               OptionCode = 37
	OptionTCPKeepaliveInterval                       OptionCode = 38
	OptionTCPKeepaliveGarbage                        OptionCode = 39
	OptionNetworkInformationServiceDomain            OptionCode = 40
	OptionNetworkInformationServers                  OptionCode = 41
	OptionNTPServers                                 OptionCode = 42
	OptionVendorSpecificInformation                  OptionCode = 43
	OptionNetBIOSOverTCPIPNameServer                 OptionCode = 44
	OptionNetBIOSOverTCPIPDatagramDistributionServer OptionCode = 45
	OptionNetBIOSOverTCPIPNodeType                   OptionCode = 46
	OptionNetBIOSOverTCPIPScope                      OptionCode = 47
	OptionXWindowSystemFontServer                    OptionCode = 48
	OptionXWindowSystemDisplayManger                 OptionCode = 49
	OptionRequestedIPAddress                         OptionCode = 50
	OptionIPAddressLeaseTime                         OptionCode = 51
	OptionOptionOverload                             OptionCode = 52
	OptionDHCPMessageType                            OptionCode = 53
	OptionServerIdentifier                           OptionCode = 54
	OptionParameterRequestList                       OptionCode = 55
	OptionMessage                                    OptionCode = 56
	OptionMaximumDHCPMessageSize                     OptionCode = 57
	OptionRenewTimeValue                             OptionCode = 58
	OptionRebindingTimeValue                         OptionCode = 59
	OptionClassIdentifier                            OptionCode = 60
	OptionClientIdentifier                           OptionCode = 61
	OptionNetWareIPDomainName                        OptionCode = 62
	OptionNetWareIPInformation                       OptionCode = 63
	OptionNetworkInformationServicePlusDomain        OptionCode = 64
	OptionNetworkInformationServicePlusServers       OptionCode = 65
	OptionTFTPServerName                             OptionCode = 66
	OptionBootfileName                               OptionCode = 67
	OptionMobileIPHomeAgent                          OptionCode = 68
	OptionSimpleMailTransportProtocolServer          OptionCode = 69
	OptionPostOfficeProtocolServer                   OptionCode = 70
	OptionNetworkNewsTransportProtocolServer         OptionCode = 71
	OptionDefaultWorldWideWebServer                  OptionCode = 72
	OptionDefaultFingerServer                        OptionCode = 73
	OptionDefaultInternetRelayChatServer             OptionCode = 74
	OptionStreetTalkServer                           OptionCode = 75
	OptionStreetTalkDirectoryAssistanceServer        OptionCode = 76
	OptionUserClassInformation                       OptionCode = 77
	OptionSLPDirectoryAgent                          OptionCode = 78
	OptionSLPServiceScope                            OptionCode = 79
	OptionRapidCommit                                OptionCode = 80
	OptionFQDN                                       OptionCode = 81
	OptionRelayAgentInformation                      OptionCode = 82
	OptionInternetStorageNameService                 OptionCode = 83
	// Option 84 returned in RFC 3679
	OptionNDSServers                       OptionCode = 85
	OptionNDSTreeName                      OptionCode = 86
	OptionNDSContext                       OptionCode = 87
	OptionBCMCSControllerDomainNameList    OptionCode = 88
	OptionBCMCSControllerIPv4AddressList   OptionCode = 89
	OptionAuthentication                   OptionCode = 90
	OptionClientLastTransactionTime        OptionCode = 91
	OptionAssociatedIP                     OptionCode = 92
	OptionClientSystemArchitectureType     OptionCode = 93
	OptionClientNetworkInterfaceIdentifier OptionCode = 94
	OptionLDAP                             OptionCode = 95
	// Option 96 returned in RFC 3679
	OptionClientMachineIdentifier     OptionCode = 97
	OptionOpenGroupUserAuthentication OptionCode = 98
	OptionGeoConfCivic                OptionCode = 99
	OptionIEEE10031TZString           OptionCode = 100
	OptionReferenceToTZDatabase       OptionCode = 101
	// Options 102-111 returned in RFC 3679
	OptionNetInfoParentServerAddress OptionCode = 112
	OptionNetInfoParentServerTag     OptionCode = 113
	OptionURL                        OptionCode = 114
	// Option 115 returned in RFC 3679
	OptionAutoConfigure                   OptionCode = 116
	OptionNameServiceSearch               OptionCode = 117
	OptionSubnetSelection                 OptionCode = 118
	OptionDNSDomainSearchList             OptionCode = 119
	OptionSIPServersDHCPOption            OptionCode = 120
	OptionClasslessStaticRouteOption      OptionCode = 121
	OptionCCC                             OptionCode = 122
	OptionGeoConf                         OptionCode = 123
	OptionVendorIdentifyingVendorClass    OptionCode = 124
	OptionVendorIdentifyingVendorSpecific OptionCode = 125
	// Options 126-127 returned in RFC 3679
	OptionTFTPServerIPAddress                   OptionCode = 128
	OptionCallServerIPAddress                   OptionCode = 129
	OptionDiscriminationString                  OptionCode = 130
	OptionRemoteStatisticsServerIPAddress       OptionCode = 131
	Option8021PVLANID                           OptionCode = 132
	Option8021QL2Priority                       OptionCode = 133
	OptionDiffservCodePoint                     OptionCode = 134
	OptionHTTPProxyForPhoneSpecificApplications OptionCode = 135
	OptionPANAAuthenticationAgent               OptionCode = 136
	OptionLoSTServer                            OptionCode = 137
	OptionCAPWAPAccessControllerAddresses       OptionCode = 138
	OptionOPTIONIPv4AddressMoS                  OptionCode = 139
	OptionOPTIONIPv4FQDNMoS                     OptionCode = 140
	OptionSIPUAConfigurationServiceDomains      OptionCode = 141
	OptionOPTIONIPv4AddressANDSF                OptionCode = 142
	OptionOPTIONIPv6AddressANDSF                OptionCode = 143
	// Options 144-149 returned in RFC 3679
	OptionTFTPServerAddress OptionCode = 150
	OptionStatusCode        OptionCode = 151
	OptionBaseTime          OptionCode = 152
	OptionStartTimeOfState  OptionCode = 153
	OptionQueryStartTime    OptionCode = 154
	OptionQueryEndTime      OptionCode = 155
	OptionDHCPState         OptionCode = 156
	OptionDataSource        OptionCode = 157
	// Options 158-174 returned in RFC 3679
	OptionEtherboot                        OptionCode = 175
	OptionIPTelephone                      OptionCode = 176
	OptionEtherbootPacketCableAndCableHome OptionCode = 177
	// Options 178-207 returned in RFC 3679
	OptionPXELinuxMagicString  OptionCode = 208
	OptionPXELinuxConfigFile   OptionCode = 209
	OptionPXELinuxPathPrefix   OptionCode = 210
	OptionPXELinuxRebootTime   OptionCode = 211
	OptionOPTION6RD            OptionCode = 212
	OptionOPTIONv4AccessDomain OptionCode = 213
	// Options 214-219 returned in RFC 3679
	OptionSubnetAllocation        OptionCode = 220
	OptionVirtualSubnetAllocation OptionCode = 221
	// Options 222-223 returned in RFC 3679
	// Options 224-254 are reserved for private use
	OptionEnd OptionCode = 255
)

// OptionCodeToString maps an OptionCode to its mnemonic name
var OptionCodeToString = map[OptionCode]string{
	OptionPad:                                        "Pad",
	OptionSubnetMask:                                 "Subnet Mask",
	OptionTimeOffset:                                 "Time Offset",
	OptionRouter:                                     "Router",
	OptionTimeServer:                                 "Time Server",
	OptionNameServer:                                 "Name Server",
	OptionDomainNameServer:                           "Domain Name Server",
	OptionLogServer:                                  "Log Server",
	OptionQuoteServer:                                "Quote Server",
	OptionLPRServer:                                  "LPR Server",
	OptionImpressServer:                              "Impress Server",
	OptionResourceLocationServer:                     "Resource Location Server",
	OptionHostName:                                   "Host Name",
	OptionBootFileSize:                               "Boot File Size",
	OptionMeritDumpFile:                              "Merit Dump File",
	OptionDomainName:                                 "Domain Name",
	OptionSwapServer:                                 "Swap Server",
	OptionRootPath:                                   "Root Path",
	OptionExtensionsPath:                             "Extensions Path",
	OptionIPForwarding:                               "IP Forwarding enable/disable",
	OptionNonLocalSourceRouting:                      "Non-local Source Routing enable/disable",
	OptionPolicyFilter:                               "Policy Filter",
	OptionMaximumDatagramAssemblySize:                "Maximum Datagram Reassembly Size",
	OptionDefaultIPTTL:                               "Default IP Time-to-live",
	OptionPathMTUAgingTimeout:                        "Path MTU Aging Timeout",
	OptionPathMTUPlateauTable:                        "Path MTU Plateau Table",
	OptionInterfaceMTU:                               "Interface MTU",
	OptionAllSubnetsAreLocal:                         "All Subnets Are Local",
	OptionBroadcastAddress:                           "Broadcast Address",
	OptionPerformMaskDiscovery:                       "Perform Mask Discovery",
	OptionMaskSupplier:                               "Mask Supplier",
	OptionPerformRouterDiscovery:                     "Perform Router Discovery",
	OptionRouterSolicitationAddress:                  "Router Solicitation Address",
	OptionStaticRoutingTable:                         "Static Routing Table",
	OptionTrailerEncapsulation:                       "Trailer Encapsulation",
	OptionArpCacheTimeout:                            "ARP Cache Timeout",
	OptionEthernetEncapsulation:                      "Ethernet Encapsulation",
	OptionDefaulTCPTTL:                               "Default TCP TTL",
	OptionTCPKeepaliveInterval:                       "TCP Keepalive Interval",
	OptionTCPKeepaliveGarbage:                        "TCP Keepalive Garbage",
	OptionNetworkInformationServiceDomain:            "Network Information Service Domain",
	OptionNetworkInformationServers:                  "Network Information Servers",
	OptionNTPServers:                                 "NTP Servers",
	OptionVendorSpecificInformation:                  "Vendor Specific Information",
	OptionNetBIOSOverTCPIPNameServer:                 "NetBIOS over TCP/IP Name Server",
	OptionNetBIOSOverTCPIPDatagramDistributionServer: "NetBIOS over TCP/IP Datagram Distribution Server",
	OptionNetBIOSOverTCPIPNodeType:                   "NetBIOS over TCP/IP Node Type",
	OptionNetBIOSOverTCPIPScope:                      "NetBIOS over TCP/IP Scope",
	OptionXWindowSystemFontServer:                    "X Window System Font Server",
	OptionXWindowSystemDisplayManger:                 "X Window System Display Manager",
	OptionRequestedIPAddress:                         "Requested IP Address",
	OptionIPAddressLeaseTime:                         "IP Addresses Lease Time",
	OptionOptionOverload:                             "Option Overload",
	OptionDHCPMessageType:                            "DHCP Message Type",
	OptionServerIdentifier:                           "Server Identifier",
	OptionParameterRequestList:                       "Parameter Request List",
	OptionMessage:                                    "Message",
	OptionMaximumDHCPMessageSize:                     "Maximum DHCP Message Size",
	OptionRenewTimeValue:                             "Renew Time Value",
	OptionRebindingTimeValue:                         "Rebinding Time Value",
	OptionClassIdentifier:                            "Class Identifier",
	OptionClientIdentifier:                           "Client identifier",
	OptionNetWareIPDomainName:                        "NetWare/IP Domain Name",
	OptionNetWareIPInformation:                       "NetWare/IP Information",
	OptionNetworkInformationServicePlusDomain:        "Network Information Service+ Domain",
	OptionNetworkInformationServicePlusServers:       "Network Information Service+ Servers",
	OptionTFTPServerName:                             "TFTP Server Name",
	OptionBootfileName:                               "Bootfile Name",
	OptionMobileIPHomeAgent:                          "Mobile IP Home Agent",
	OptionSimpleMailTransportProtocolServer:          "SMTP Server",
	OptionPostOfficeProtocolServer:                   "POP Server",
	OptionNetworkNewsTransportProtocolServer:         "NNTP Server",
	OptionDefaultWorldWideWebServer:                  "Default WWW Server",
	OptionDefaultFingerServer:                        "Default Finger Server",
	OptionDefaultInternetRelayChatServer:             "Default IRC Server",
	OptionStreetTalkServer:                           "StreetTalk Server",
	OptionStreetTalkDirectoryAssistanceServer:        "StreetTalk Directory Assistance Server",
	OptionUserClassInformation:                       "User Class Information",
	OptionSLPDirectoryAgent:                          "SLP DIrectory Agent",
	OptionSLPServiceScope:                            "SLP Service Scope",
	OptionRapidCommit:                                "Rapid Commit",
	OptionFQDN:                                       "FQDN",
	OptionRelayAgentInformation:                      "Relay Agent Information",
	OptionInternetStorageNameService:                 "Internet Storage Name Service",
	// Option 84 returned in RFC 3679
	OptionNDSServers:                       "NDS Servers",
	OptionNDSTreeName:                      "NDS Tree Name",
	OptionNDSContext:                       "NDS Context",
	OptionBCMCSControllerDomainNameList:    "BCMCS Controller Domain Name List",
	OptionBCMCSControllerIPv4AddressList:   "BCMCS Controller IPv4 Address List",
	OptionAuthentication:                   "Authentication",
	OptionClientLastTransactionTime:        "Client Last Transaction Time",
	OptionAssociatedIP:                     "Associated IP",
	OptionClientSystemArchitectureType:     "Client System Architecture Type",
	OptionClientNetworkInterfaceIdentifier: "Client Network Interface Identifier",
	OptionLDAP:                             "LDAP",
	// Option 96 returned in RFC 3679
	OptionClientMachineIdentifier:     "Client Machine Identifier",
	OptionOpenGroupUserAuthentication: "OpenGroup's User Authentication",
	OptionGeoConfCivic:                "GEOCONF_CIVIC",
	OptionIEEE10031TZString:           "IEEE 1003.1 TZ String",
	OptionReferenceToTZDatabase:       "Reference to the TZ Database",
	// Options 102-111 returned in RFC 3679
	OptionNetInfoParentServerAddress: "NetInfo Parent Server Address",
	OptionNetInfoParentServerTag:     "NetInfo Parent Server Tag",
	OptionURL:                        "URL",
	// Option 115 returned in RFC 3679
	OptionAutoConfigure:                   "Auto-Configure",
	OptionNameServiceSearch:               "Name Service Search",
	OptionSubnetSelection:                 "Subnet Selection",
	OptionDNSDomainSearchList:             "DNS Domain Search List",
	OptionSIPServersDHCPOption:            "SIP Servers DHCP Option",
	OptionClasslessStaticRouteOption:      "Classless Static Route Option",
	OptionCCC:                             "CCC, CableLabs Client Configuration",
	OptionGeoConf:                         "GeoConf",
	OptionVendorIdentifyingVendorClass:    "Vendor-Identifying Vendor Class",
	OptionVendorIdentifyingVendorSpecific: "Vendor-Identifying Vendor-Specific",
	// Options 126-127 returned in RFC 3679
	OptionTFTPServerIPAddress:                   "TFTP Server IP Address",
	OptionCallServerIPAddress:                   "Call Server IP Address",
	OptionDiscriminationString:                  "Discrimination String",
	OptionRemoteStatisticsServerIPAddress:       "RemoteStatistics Server IP Address",
	Option8021PVLANID:                           "802.1P VLAN ID",
	Option8021QL2Priority:                       "802.1Q L2 Priority",
	OptionDiffservCodePoint:                     "Diffserv Code Point",
	OptionHTTPProxyForPhoneSpecificApplications: "HTTP Proxy for phone-specific applications",
	OptionPANAAuthenticationAgent:               "PANA Authentication Agent",
	OptionLoSTServer:                            "LoST Server",
	OptionCAPWAPAccessControllerAddresses:       "CAPWAP Access Controller Addresses",
	OptionOPTIONIPv4AddressMoS:                  "OPTION-IPv4_Address-MoS",
	OptionOPTIONIPv4FQDNMoS:                     "OPTION-IPv4_FQDN-MoS",
	OptionSIPUAConfigurationServiceDomains:      "SIP UA Configuration Service Domains",
	OptionOPTIONIPv4AddressANDSF:                "OPTION-IPv4_Address-ANDSF",
	OptionOPTIONIPv6AddressANDSF:                "OPTION-IPv6_Address-ANDSF",
	// Options 144-149 returned in RFC 3679
	OptionTFTPServerAddress: "TFTP Server Address",
	OptionStatusCode:        "Status Code",
	OptionBaseTime:          "Base Time",
	OptionStartTimeOfState:  "Start Time of State",
	OptionQueryStartTime:    "Query Start Time",
	OptionQueryEndTime:      "Query End Time",
	OptionDHCPState:         "DHCP Staet",
	OptionDataSource:        "Data Source",
	// Options 158-174 returned in RFC 3679
	OptionEtherboot:                        "Etherboot",
	OptionIPTelephone:                      "IP Telephone",
	OptionEtherbootPacketCableAndCableHome: "Etherboot / PacketCable and CableHome",
	// Options 178-207 returned in RFC 3679
	OptionPXELinuxMagicString:  "PXELinux Magic String",
	OptionPXELinuxConfigFile:   "PXELinux Config File",
	OptionPXELinuxPathPrefix:   "PXELinux Path Prefix",
	OptionPXELinuxRebootTime:   "PXELinux Reboot Time",
	OptionOPTION6RD:            "OPTION_6RD",
	OptionOPTIONv4AccessDomain: "OPTION_V4_ACCESS_DOMAIN",
	// Options 214-219 returned in RFC 3679
	OptionSubnetAllocation:        "Subnet Allocation",
	OptionVirtualSubnetAllocation: "Virtual Subnet Selection",
	// Options 222-223 returned in RFC 3679
	// Options 224-254 are reserved for private use

	OptionEnd: "End",
}
