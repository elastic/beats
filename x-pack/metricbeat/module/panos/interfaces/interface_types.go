// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package interfaces

import "encoding/xml"

type InterfaceResponse struct {
	XMLName xml.Name        `xml:"response"`
	Status  string          `xml:"status,attr"`
	Result  InterfaceResult `xml:"result"`
}

type InterfaceResult struct {
	HW    HW    `xml:"hw"`
	Ifnet Ifnet `xml:"ifnet"`
}

type HW struct {
	Entries []HWInterfaceEntry `xml:"entry"`
}

type HWInterfaceEntry struct {
	Name     string   `xml:"name"`
	ID       int      `xml:"id"`
	Type     int      `xml:"type"`
	MAC      string   `xml:"mac"`
	Speed    string   `xml:"speed"`
	Duplex   string   `xml:"duplex"`
	State    string   `xml:"state"`
	Mode     string   `xml:"mode"`
	ST       string   `xml:"st"`
	AEMember AEMember `xml:"ae_member"`
}

type IFInterfaceEntry struct {
	Name    string `xml:"name"`
	ID      int    `xml:"id"`
	Tag     int    `xml:"tag"`
	Vsys    int    `xml:"vsys"`
	Zone    string `xml:"zone"`
	Fwd     string `xml:"fwd"`
	IP_CIDR string `xml:"ip"`
	Addr    string `xml:"addr"`
	DynAddr string `xml:"dyn-addr"`
	Addr6   string `xml:"addr6"`
}

type AEMember struct {
	Members []string `xml:"member"`
}

type Ifnet struct {
	Entries []IFInterfaceEntry `xml:"entry"`
}

// HA Interfaces

type HAResponse struct {
	XMLName xml.Name `xml:"response"`
	Status  string   `xml:"status,attr"`
	Result  HAResult `xml:"result"`
}

type HAResult struct {
	Enabled string  `xml:"enabled"`
	Group   HAGroup `xml:"group"`
}

type HAGroup struct {
	Mode               string           `xml:"mode"`
	LocalInfo          HALocalInfo      `xml:"local-info"`
	PeerInfo           HAPeerInfo       `xml:"peer-info"`
	LinkMonitoring     HALinkMonitoring `xml:"link-monitoring"`
	PathMonitoring     HAPathMonitoring `xml:"path-monitoring"`
	RunningSync        string           `xml:"running-sync"`
	RunningSyncEnabled string           `xml:"running-sync-enabled"`
}

type HALocalInfo struct {
	Version            string        `xml:"version"`
	State              string        `xml:"state"`
	StateDuration      int           `xml:"state-duration"`
	MgmtIP             string        `xml:"mgmt-ip"`
	MgmtIPv6           string        `xml:"mgmt-ipv6"`
	Preemptive         string        `xml:"preemptive"`
	PromotionHold      int           `xml:"promotion-hold"`
	HelloInterval      int           `xml:"hello-interval"`
	HeartbeatInterval  int           `xml:"heartbeat-interval"`
	PreemptHold        int           `xml:"preempt-hold"`
	MonitorFailHoldup  int           `xml:"monitor-fail-holdup"`
	AddonMasterHoldup  int           `xml:"addon-master-holdup"`
	HA1EncryptImported string        `xml:"ha1-encrypt-imported"`
	Mode               string        `xml:"mode"`
	PlatformModel      string        `xml:"platform-model"`
	Priority           int           `xml:"priority"`
	MaxFlaps           int           `xml:"max-flaps"`
	PreemptFlapCnt     int           `xml:"preempt-flap-cnt"`
	NonfuncFlapCnt     int           `xml:"nonfunc-flap-cnt"`
	StateSync          string        `xml:"state-sync"`
	StateSyncType      string        `xml:"state-sync-type"`
	ActivePassive      ActivePassive `xml:"active-passive"`
	HA1IPAddr          string        `xml:"ha1-ipaddr"`
	HA1MACAddr         string        `xml:"ha1-macaddr"`
	HA1Port            string        `xml:"ha1-port"`
	HA1EncryptEnable   string        `xml:"ha1-encrypt-enable"`
	HA1LinkMonIntv     int           `xml:"ha1-link-mon-intv"`
	HA1BackupIPAddr    string        `xml:"ha1-backup-ipaddr"`
	HA1BackupMACAddr   string        `xml:"ha1-backup-macaddr"`
	HA1BackupPort      string        `xml:"ha1-backup-port"`
	HA1BackupGateway   string        `xml:"ha1-backup-gateway"`
	HA2IPAddr          string        `xml:"ha2-ipaddr"`
	HA2MACAddr         string        `xml:"ha2-macaddr"`
	HA2Port            string        `xml:"ha2-port"`
	BuildRel           string        `xml:"build-rel"`
	URLVersion         string        `xml:"url-version"`
	AppVersion         string        `xml:"app-version"`
	IoTVersion         string        `xml:"iot-version"`
	AVVersion          string        `xml:"av-version"`
	ThreatVersion      string        `xml:"threat-version"`
	VPNClientVersion   string        `xml:"vpnclient-version"`
	GPClientVersion    string        `xml:"gpclient-version"`
	DLP                string        `xml:"DLP"`
	BuildCompat        string        `xml:"build-compat"`
	URLCompat          string        `xml:"url-compat"`
	AppCompat          string        `xml:"app-compat"`
	IoTCompat          string        `xml:"iot-compat"`
	AVCompat           string        `xml:"av-compat"`
	ThreatCompat       string        `xml:"threat-compat"`
	VPNClientCompat    string        `xml:"vpnclient-compat"`
	GPClientCompat     string        `xml:"gpclient-compat"`
}

type ActivePassive struct {
	PassiveLinkState    string `xml:"passive-link-state"`
	MonitorFailHolddown int    `xml:"monitor-fail-holddown"`
}

type HAPeerInfo struct {
	ConnHA1          ConnHA1       `xml:"conn-ha1"`
	ConnHA1Backup    ConnHA1Backup `xml:"conn-ha1-backup"`
	ConnHA2          ConnHA2       `xml:"conn-ha2"`
	ConnStatus       string        `xml:"conn-status"`
	Version          string        `xml:"version"`
	State            string        `xml:"state"`
	StateDuration    int           `xml:"state-duration"`
	LastErrorReason  string        `xml:"last-error-reason"`
	LastErrorState   string        `xml:"last-error-state"`
	Preemptive       string        `xml:"preemptive"`
	Mode             string        `xml:"mode"`
	PlatformModel    string        `xml:"platform-model"`
	VMLicense        string        `xml:"vm-license"`
	Priority         int           `xml:"priority"`
	MgmtIP           string        `xml:"mgmt-ip"`
	MgmtIPv6         string        `xml:"mgmt-ipv6"`
	HA1IPAddr        string        `xml:"ha1-ipaddr"`
	HA1MACAddr       string        `xml:"ha1-macaddr"`
	HA1BackupIPAddr  string        `xml:"ha1-backup-ipaddr"`
	HA1BackupMACAddr string        `xml:"ha1-backup-macaddr"`
	HA2IPAddr        string        `xml:"ha2-ipaddr"`
	HA2MACAddr       string        `xml:"ha2-macaddr"`
	BuildRel         string        `xml:"build-rel"`
	URLVersion       string        `xml:"url-version"`
	AppVersion       string        `xml:"app-version"`
	IoTVersion       string        `xml:"iot-version"`
	AVVersion        string        `xml:"av-version"`
	ThreatVersion    string        `xml:"threat-version"`
	VPNClientVersion string        `xml:"vpnclient-version"`
	GPClientVersion  string        `xml:"gpclient-version"`
	DLP              string        `xml:"DLP"`
}

type ConnHA1 struct {
	Status  string `xml:"conn-status"`
	Primary string `xml:"conn-primary"`
	Desc    string `xml:"conn-desc"`
}

type ConnHA1Backup struct {
	Status string `xml:"conn-status"`
	Desc   string `xml:"conn-desc"`
}

type ConnHA2 struct {
	Primary   string `xml:"conn-primary"`
	KAEnabled string `xml:"conn-ka-enbled"`
	Desc      string `xml:"conn-desc"`
	Status    string `xml:"conn-status"`
}

type HALinkMonitoring struct {
	Enabled          string         `xml:"enabled"`
	FailureCondition string         `xml:"failure-condition"`
	Groups           []HAGroupEntry `xml:"groups>entry"`
}

type HAGroupEntry struct {
	Name             string             `xml:"name"`
	Enabled          string             `xml:"enabled"`
	FailureCondition string             `xml:"failure-condition"`
	Interface        []HAInterfaceEntry `xml:"interface>entry"`
}

type HAInterfaceEntry struct {
	Name   string `xml:"name"`
	Status string `xml:"status"`
}

type HAPathMonitoring struct {
	Enabled          string `xml:"enabled"`
	FailureCondition string `xml:"failure-condition"`
	VirtualWire      string `xml:"virtual-wire"`
	VLAN             string `xml:"vlan"`
	VirtualRouter    string `xml:"virtual-router"`
}
