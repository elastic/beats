// Copyright 2019 Tamás Gulácsi
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LIENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR ONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package sid

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	errors "golang.org/x/xerrors"
)

// Statement can Parse and Print Oracle connection descriptor (DESRIPTION=(ADDRESS=...)) format.
// It can be used to parse or build a SID.
//
// See https://docs.oracle.com/cd/B28359_01/network.111/b28317/tnsnames.htm#NETRF271
type Statement struct {
	Name, Value string
	Statements  []Statement
}

func (cs Statement) String() string {
	var buf strings.Builder
	cs.Print(&buf, "\n", "  ")
	return buf.String()
}
func (cs Statement) Print(w io.Writer, prefix, indent string) {
	fmt.Fprintf(w, "%s(%s=%s", prefix, cs.Name, cs.Value)
	if cs.Value == "" {
		for _, s := range cs.Statements {
			s.Print(w, prefix+indent, indent)
		}
	}
	io.WriteString(w, ")")
}

func ParseConnDescription(s string) (Statement, error) {
	var cs Statement
	_, err := cs.Parse(s)
	return cs, err
}
func (cs *Statement) Parse(s string) (string, error) {
	ltrim := func(s string) string { return strings.TrimLeftFunc(s, unicode.IsSpace) }
	s = ltrim(s)
	if s == "" || s[0] != '(' {
		return s, nil
	}
	i := strings.IndexByte(s[1:], '=') + 1
	if i <= 0 || strings.Contains(s[1:i], ")") {
		return s, errors.Errorf("no = after ( in %q", s)
	}
	cs.Name = s[1:i]
	s = ltrim(s[i+1:])

	if s == "" {
		return s, nil
	}
	if s[0] != '(' {
		if i = strings.IndexByte(s, ')'); i < 0 || strings.Contains(s[1:i], "(") {
			return s, errors.Errorf("no ) after = in %q", s)
		}
		cs.Value = s[:i]
		s = ltrim(s[i+1:])
		return s, nil
	}

	for s != "" && s[0] == '(' {
		var sub Statement
		var err error
		if s, err = sub.Parse(s); err != nil {
			return s, err
		}
		if sub.Name == "" {
			break
		}
		cs.Statements = append(cs.Statements, sub)
	}
	s = ltrim(s)
	if s != "" && s[0] == ')' {
		s = ltrim(s[1:])
	}
	return s, nil
}

type DescriptionList struct {
	Options       ListOptions
	Descriptions  []Description
	TypeOfService string
}

func (cd DescriptionList) Print(w io.Writer, prefix, indent string) {
	io.WriteString(w, prefix+"(DESCRIPTION_LIST=")
	cd.Options.Print(w, prefix, indent)
	for _, d := range cd.Descriptions {
		d.Print(w, prefix, indent)
	}
	if cd.TypeOfService != "" {
		fmt.Fprintf(w, "%s(TYPE_OF_SERVICE=%s)", prefix, cd.TypeOfService)
	}
	io.WriteString(w, ")")
}
func (cd *DescriptionList) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "DESCRIPTION_LIST" {
		ss = ss[0].Statements
	}
	cd.TypeOfService = ""
	if err := cd.Options.Parse(ss); err != nil {
		return err
	}
	cd.Descriptions = cd.Descriptions[:0]
	for _, s := range ss {
		switch s.Name {
		case "DESCRIPTION":
			var d Description
			if err := d.Parse(s.Statements); err != nil {
				return err
			}
			cd.Descriptions = append(cd.Descriptions, d)
		case "TYPE_OF_SERVICE":
			cd.TypeOfService = s.Value
		}
	}
	return cd.Options.Parse(ss)
}

type Description struct {
	TCPKeepAlive  bool
	SDU           int
	Bufs          BufSizes
	Options       ListOptions
	Addresses     []Address
	AddressList   AddressList
	ConnectData   ConnectData
	TypeOfService string
	Security      Security
}

func (d Description) Print(w io.Writer, prefix, indent string) {
	if d.IsZero() {
		return
	}
	io.WriteString(w, prefix+"(DESCRIPTION=")
	if d.TCPKeepAlive {
		io.WriteString(w, prefix+"(ENABLE=broken)")
	}
	if d.SDU != 0 {
		fmt.Fprintf(w, prefix+"(SDU=%d)", d.SDU)
	}
	d.Bufs.Print(w, prefix, indent)
	d.Options.Print(w, prefix, indent)
	for _, a := range d.Addresses {
		a.Print(w, prefix, indent)
	}
	d.AddressList.Print(w, prefix, indent)
	d.ConnectData.Print(w, prefix, indent)
	if d.TypeOfService != "" {
		fmt.Fprintf(w, "%s(TYPE_OF_SERVICE=%s)", prefix, d.TypeOfService)
	}
	d.Security.Print(w, prefix, indent)
	io.WriteString(w, ")")
}
func (d Description) IsZero() bool {
	return !d.TCPKeepAlive && d.SDU == 0 && d.Bufs.IsZero() && d.Options.IsZero() && len(d.Addresses) == 0 && d.AddressList.IsZero() && d.ConnectData.IsZero() && d.TypeOfService == "" && d.Security.IsZero()
}
func (d *Description) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "DESCRIPTION" {
		ss = ss[0].Statements
	}
	d.TCPKeepAlive, d.SDU = false, 0
	for _, s := range ss {
		switch s.Name {
		case "ADDRESS":
			var a Address
			if err := a.Parse(s.Statements); err != nil {
				return err
			}
			if !a.IsZero() {
				d.Addresses = append(d.Addresses, a)
			}
		case "ADDRESS_LIST":
			if err := d.AddressList.Parse(s.Statements); err != nil {
				return err
			}
		case "CONNECT_DATA":
			if err := d.ConnectData.Parse(s.Statements); err != nil {
				return err
			}
		case "ENABLE":
			d.TCPKeepAlive = d.TCPKeepAlive || s.Value == "broken"
		case "SDU":
			var err error
			if d.SDU, err = strconv.Atoi(s.Value); err != nil {
				return err
			}
		case "SECURITY":
			if err := d.Security.Parse(s.Statements); err != nil {
				return err
			}
		}
	}
	if err := d.Bufs.Parse(ss); err != nil {
		return err
	}
	if err := d.Options.Parse(ss); err != nil {
		return err
	}
	return nil
}

type Address struct {
	Protocol, Host string
	Port           int
	BufSizes
}

func (a Address) Print(w io.Writer, prefix, indent string) {
	if a.IsZero() {
		return
	}
	io.WriteString(w, prefix+"(ADDRESS=")
	if a.Protocol != "" {
		fmt.Fprintf(w, "%s(PROTOCOL=%s)", prefix, a.Protocol)
	}
	if a.Host != "" {
		fmt.Fprintf(w, "%s(HOST=%s)", prefix, a.Host)
	}
	if a.Port != 0 {
		fmt.Fprintf(w, "%s(PORT=%d)", prefix, a.Port)
	}
	a.BufSizes.Print(w, prefix, indent)
	io.WriteString(w, ")")
}
func (a Address) IsZero() bool {
	return a.Protocol == "" && a.Host == "" && a.Port == 0 && a.BufSizes.IsZero()
}
func (a *Address) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "ADDRESS" {
		ss = ss[0].Statements
	}
	for _, s := range ss {
		switch s.Name {
		case "PROTOCOL":
			a.Protocol = s.Value
		case "HOST":
			a.Host = s.Value
		case "PORT":
			i, err := strconv.Atoi(s.Value)
			if err != nil {
				return err
			}
			a.Port = i
		}
	}
	return a.BufSizes.Parse(ss)
}

type BufSizes struct {
	RecvBufSize, SendBufSize int
}

func (bs BufSizes) Print(w io.Writer, prefix, indent string) {
	if bs.RecvBufSize > 0 {
		fmt.Fprintf(w, "%s(RECV_BUF_SIZE=%d)", prefix, bs.RecvBufSize)
	}
	if bs.SendBufSize > 0 {
		fmt.Fprintf(w, "%s(SEND_BUF_SIZE=%d)", prefix, bs.SendBufSize)
	}
}
func (bs BufSizes) IsZero() bool { return bs.RecvBufSize > 0 && bs.SendBufSize > 0 }
func (bs *BufSizes) Parse(ss []Statement) error {
	for _, s := range ss {
		switch s.Name {
		case "RECV_BUF_SIZE", "SEND_BUF_SIZE":
			i, err := strconv.Atoi(s.Value)
			if err != nil {
				return err
			}
			if s.Name == "RECV_BUF_SIZE" {
				bs.RecvBufSize = i
			} else {
				bs.SendBufSize = i
			}
		}
	}
	return nil
}

type ListOptions struct {
	Failover, LoadBalance, SourceRoute bool
}

func (lo ListOptions) Print(w io.Writer, prefix, indent string) {
	if lo.Failover {
		io.WriteString(w, prefix+"(FAILOVER=on)")
	}
	if lo.LoadBalance {
		io.WriteString(w, prefix+"(LOAD_BALANE=on)")
	}
	if lo.SourceRoute {
		io.WriteString(w, prefix+"(SOURE_ROUTE=on)")
	}
}
func (lo ListOptions) IsZero() bool { return !lo.Failover && !lo.LoadBalance && !lo.SourceRoute }
func s2b(s string) bool             { return s == "on" || s == "yes" || s == "true" }
func (lo *ListOptions) Parse(ss []Statement) error {
	*lo = ListOptions{}
	for _, s := range ss {
		switch s.Name {
		case "FAILOVER":
			lo.Failover = s2b(s.Value)
		case "LOAD_BALANE":
			lo.LoadBalance = s2b(s.Value)
		case "SourceRoute":
			lo.SourceRoute = s2b(s.Value)
		}
	}
	return nil
}

type AddressList struct {
	Options   ListOptions
	Addresses []Address
}

func (al AddressList) Print(w io.Writer, prefix, indent string) {
	if al.IsZero() {
		return
	}
	io.WriteString(w, prefix+"(ADDRESS_LIST=")
	al.Options.Print(w, prefix, indent)
	for _, a := range al.Addresses {
		a.Print(w, prefix, indent)
	}
	io.WriteString(w, ")")
}
func (al AddressList) IsZero() bool { return al.Options.IsZero() && len(al.Addresses) == 0 }
func (al *AddressList) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "ADDRESS_LIST" {
		ss = ss[0].Statements
	}
	if err := al.Options.Parse(ss); err != nil {
		return err
	}
	al.Addresses = al.Addresses[:0]
	for _, s := range ss {
		switch s.Name {
		case "ADDRESS":
			var a Address
			if err := a.Parse(s.Statements); err != nil {
				return err
			}
			if !a.IsZero() {
				al.Addresses = append(al.Addresses, a)
			}
		}
	}
	return nil
}

type ConnectData struct {
	FailoverMode                          FailoverMode
	ServiceName, SID                      string
	GlobalName, InstanceName, RDBDatabase string
	Hs                                    bool
	Server                                ServiceHandler
}

func (cd ConnectData) Print(w io.Writer, prefix, indent string) {
	if cd.IsZero() {
		return
	}
	io.WriteString(w, prefix+"(CONNECT_DATA=")
	cd.FailoverMode.Print(w, prefix, indent)
	if cd.GlobalName != "" {
		fmt.Fprintf(w, "%s(GLOBAL_NAME=%s)", prefix, cd.GlobalName)
	}
	if cd.InstanceName != "" {
		fmt.Fprintf(w, "%s(INSTANCE_NAME=%s)", prefix, cd.InstanceName)
	}
	if cd.RDBDatabase != "" {
		fmt.Fprintf(w, "%s(RDB_DATABASE=%s)", prefix, cd.RDBDatabase)
	}
	if cd.ServiceName != "" {
		fmt.Fprintf(w, "%s(SERVICE_NAME=%s)", prefix, cd.ServiceName)
	}
	if cd.SID != "" {
		fmt.Fprintf(w, "%s(SID=%s)", prefix, cd.SID)
	}
	if cd.Hs {
		io.WriteString(w, prefix+"(HS=ok)")
	}
	if cd.Server != "" {
		fmt.Fprintf(w, "%s(SERVER=%s)", prefix, cd.Server)
	}
	io.WriteString(w, ")")
}
func (cd ConnectData) IsZero() bool {
	return cd.FailoverMode.IsZero() && cd.GlobalName == "" && cd.InstanceName == "" && cd.RDBDatabase == "" && cd.ServiceName == "" && cd.SID == "" && !cd.Hs && cd.Server == ""
}
func (cd *ConnectData) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "CONNECT_DATA" {
		ss = ss[0].Statements
	}
	cd.Hs = false
	for _, s := range ss {
		switch s.Name {
		case "FAILOVER_MODE":
			if err := cd.FailoverMode.Parse(s.Statements); err != nil {
				return err
			}
		case "GLOBAL_NAME":
			cd.GlobalName = s.Value
		case "INSTANCE_NAME":
			cd.InstanceName = s.Value
		case "RDB_DATABASE":
			cd.RDBDatabase = s.Value
		case "SERVICE_NAME":
			cd.ServiceName = s.Value
		case "SID":
			cd.SID = s.Value
		case "HS":
			cd.Hs = s.Value == "ok"
		case "SERVER":
			cd.Server = ServiceHandler(s.Value)
		}
	}
	return nil
}

type FailoverMode struct {
	Backup, Type, Method string
	Retry, Delay         int
}

func (fo FailoverMode) Print(w io.Writer, prefix, indent string) {
	if fo.IsZero() {
		return
	}
	io.WriteString(w, prefix+"(FAILOVER_MODE=")
	if fo.Backup != "" {
		fmt.Fprintf(w, "%s(BACKUP=%s)", prefix, fo.Backup)
	}
	if fo.Type != "" {
		fmt.Fprintf(w, "%s(TYPE=%s)", prefix, fo.Type)
	}
	if fo.Method != "" {
		fmt.Fprintf(w, "%s(METHOD=%s)", prefix, fo.Method)
	}
	if fo.Retry != 0 {
		fmt.Fprintf(w, "%s(RETRY=%d)", prefix, fo.Retry)
	}
	if fo.Delay != 0 {
		fmt.Fprintf(w, "%s(DELAY=%d)", prefix, fo.Delay)
	}
	io.WriteString(w, ")")
}
func (fo FailoverMode) IsZero() bool {
	return fo.Backup == "" && fo.Type == "" && fo.Method == "" && fo.Retry == 0 && fo.Delay == 0
}
func (fo *FailoverMode) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "FAILOVER_MODE" {
		ss = ss[0].Statements
	}
	for _, s := range ss {
		switch s.Name {
		case "BACKUP":
			fo.Backup = s.Value
		case "TYPE":
			fo.Type = s.Value
		case "METHOD":
			fo.Method = s.Value
		case "RETRY", "DELAY":
			i, err := strconv.Atoi(s.Value)
			if err != nil {
				return err
			}
			if s.Name == "RETRY" {
				fo.Retry = i
			} else {
				fo.Delay = i
			}
		}
	}
	return nil
}

type ServiceHandler string

const (
	Dedicated = ServiceHandler("dedicated")
	Shared    = ServiceHandler("shared")
	Pooled    = ServiceHandler("pooled")
)

type Security struct {
	SSLServerCertDN string
}

func (sec Security) Print(w io.Writer, prefix, indent string) {
	if sec.SSLServerCertDN != "" {
		fmt.Fprintf(w, "%s(SECURITY=(SSL_SERVER_CERT_DN=%s))", prefix, sec.SSLServerCertDN)
	}
}
func (sec Security) IsZero() bool { return sec.SSLServerCertDN == "" }
func (sec *Security) Parse(ss []Statement) error {
	if len(ss) == 1 && ss[0].Name == "SECURITY" {
		ss = ss[0].Statements
	}
	sec.SSLServerCertDN = ""
	for _, s := range ss {
		if s.Name == "SSL_SERVER_CERT_DN" {
			sec.SSLServerCertDN = s.Value
		}
	}
	return nil
}
