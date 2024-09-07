// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vpn

// GlobalProtect sesssions
type GPSessionsResponse struct {
	Status string           `xml:"status,attr"`
	Result GPSessionsResult `xml:"result"`
}

type GPSessionsResult struct {
	Sessions []GPSession `xml:"entry"`
}

type GPSession struct {
	Domain               string `xml:"domain"`
	IsLocal              string `xml:"islocal"`
	Username             string `xml:"username"`
	PrimaryUsername      string `xml:"primary-username"`
	RegionForConfig      string `xml:"region-for-config"`
	SourceRegion         string `xml:"source-region"`
	Computer             string `xml:"computer"`
	Client               string `xml:"client"`
	VPNType              string `xml:"vpn-type"`
	HostID               string `xml:"host-id"`
	AppVersion           string `xml:"app-version"`
	VirtualIP            string `xml:"virtual-ip"`
	VirtualIPv6          string `xml:"virtual-ipv6"`
	PublicIP             string `xml:"public-ip"`
	PublicIPv6           string `xml:"public-ipv6"`
	TunnelType           string `xml:"tunnel-type"`
	PublicConnectionIPv6 string `xml:"public-connection-ipv6"`
	ClientIP             string `xml:"client-ip"`
	LoginTime            string `xml:"login-time"`
	LoginTimeUTC         string `xml:"login-time-utc"`
	Lifetime             string `xml:"lifetime"`
	RequestLogin         string `xml:"request-login"`
	RequestGetConfig     string `xml:"request-getconfig"`
	RequestSSLVPNConnect string `xml:"request-sslvpnconnect"`
}

// GlobalProtect gateway stats

type GPStatsResponse struct {
	Status string        `xml:"status,attr"`
	Result GPStatsResult `xml:"result"`
}

type GPStatsResult struct {
	Gateways           []GPGateway `xml:"Gateway"`
	TotalCurrentUsers  int         `xml:"TotalCurrentUsers"`
	TotalPreviousUsers int         `xml:"TotalPreviousUsers"`
}

type GPGateway struct {
	Name          string `xml:"name"`
	CurrentUsers  int    `xml:"CurrentUsers"`
	PreviousUsers int    `xml:"PreviousUsers"`
}

// IPSec tunnels

type TunnelsResponse struct {
	Status string        `xml:"status,attr"`
	Result TunnelsResult `xml:"result"`
}

type TunnelsResult struct {
	Entries []TunnelsEntry `xml:"entries>entry"`
	NTun    int            `xml:"ntun"`
}

type TunnelsEntry struct {
	ID        int    `xml:"id"`
	Name      string `xml:"name"`
	GW        string `xml:"gw"`
	TSiIP     string `xml:"TSi_ip"`
	TSiPrefix int    `xml:"TSi_prefix"`
	TSiProto  int    `xml:"TSi_proto"`
	TSiPort   int    `xml:"TSi_port"`
	TSrIP     string `xml:"TSr_ip"`
	TSrPrefix int    `xml:"TSr_prefix"`
	TSrProto  int    `xml:"TSr_proto"`
	TSrPort   int    `xml:"TSr_port"`
	Proto     string `xml:"proto"`
	Mode      string `xml:"mode"`
	DH        string `xml:"dh"`
	Enc       string `xml:"enc"`
	Hash      string `xml:"hash"`
	Life      int    `xml:"life"`
	KB        int    `xml:"kb"`
}