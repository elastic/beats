// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package bgp_peers

type Response struct {
	Status string `xml:"status,attr"`
	Result Result `xml:"result"`
}

type Result struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Peer                 string         `xml:"peer,attr"`
	Vr                   string         `xml:"vr,attr"`
	PeerGroup            string         `xml:"peer-group"`
	PeerRouterID         string         `xml:"peer-router-id"`
	RemoteAS             int            `xml:"remote-as"`
	Status               string         `xml:"status"`
	StatusDuration       int            `xml:"status-duration"`
	PasswordSet          string         `xml:"password-set"`
	Passive              string         `xml:"passive"`
	MultiHopTTL          int            `xml:"multi-hop-ttl"`
	PeerAddress          string         `xml:"peer-address"`
	LocalAddress         string         `xml:"local-address"`
	ReflectorClient      string         `xml:"reflector-client"`
	SameConfederation    string         `xml:"same-confederation"`
	AggregateConfedAS    string         `xml:"aggregate-confed-as"`
	PeeringType          string         `xml:"peering-type"`
	ConnectRetryInterval int            `xml:"connect-retry-interval"`
	OpenDelay            int            `xml:"open-delay"`
	IdleHold             int            `xml:"idle-hold"`
	PrefixLimit          int            `xml:"prefix-limit"`
	Holdtime             int            `xml:"holdtime"`
	HoldtimeConfig       int            `xml:"holdtime-config"`
	Keepalive            int            `xml:"keepalive"`
	KeepaliveConfig      int            `xml:"keepalive-config"`
	MsgUpdateIn          int            `xml:"msg-update-in"`
	MsgUpdateOut         int            `xml:"msg-update-out"`
	MsgTotalIn           int            `xml:"msg-total-in"`
	MsgTotalOut          int            `xml:"msg-total-out"`
	LastUpdateAge        int            `xml:"last-update-age"`
	LastError            string         `xml:"last-error"`
	StatusFlapCounts     int            `xml:"status-flap-counts"`
	EstablishedCounts    int            `xml:"established-counts"`
	ORFEntryReceived     int            `xml:"ORF-entry-received"`
	NexthopSelf          string         `xml:"nexthop-self"`
	NexthopThirdparty    string         `xml:"nexthop-thirdparty"`
	NexthopPeer          string         `xml:"nexthop-peer"`
	Config               Config         `xml:"config"`
	PeerCapability       PeerCapability `xml:"peer-capability"`
	PrefixCounter        PrefixCounter  `xml:"prefix-counter"`
}

type Config struct {
	RemovePrivateAS string `xml:"remove-private-as"`
}

type PeerCapability struct {
	List []Capability `xml:"list"`
}

type Capability struct {
	Capability string `xml:"capability"`
	Value      string `xml:"value"`
}

type PrefixCounter struct {
	Entries []PrefixEntry `xml:"entry"`
}

type PrefixEntry struct {
	AfiSafi            string `xml:"afi-safi,attr"`
	IncomingTotal      int    `xml:"incoming-total"`
	IncomingAccepted   int    `xml:"incoming-accepted"`
	IncomingRejected   int    `xml:"incoming-rejected"`
	PolicyRejected     int    `xml:"policy-rejected"`
	OutgoingTotal      int    `xml:"outgoing-total"`
	OutgoingAdvertised int    `xml:"outgoing-advertised"`
}
