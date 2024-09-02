// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package globalprotect_sessions

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Sessions []Session `xml:"entry"`
}

type Session struct {
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
