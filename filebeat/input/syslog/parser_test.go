package main

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSyslog(t *testing.T) {
	tests := []struct {
		title  string
		log    []byte
		syslog SyslogMessage
	}{
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: SyslogMessage{
				Message:  []byte("'su root' failed for lonvick on /dev/pts/8"),
				Hostname: []byte("mymachine"),
				Priority: []byte("34"),
				Program:  []byte("su"),
				Pid:      []byte("230"),
				month:    []byte("Oct"),
				day:      []byte("11"),
				hour:     []byte("22"),
				minute:   []byte("14"),
				second:   []byte("15"),
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: SyslogMessage{
				Priority: []byte("34"),
				Message:  []byte("'su root' failed for lonvick on /dev/pts/8"),
				Hostname: []byte("mymachine"),
				Program:  []byte("su"),
				month:    []byte("Oct"),
				day:      []byte("11"),
				hour:     []byte("22"),
				minute:   []byte("14"),
				second:   []byte("15"),
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: SyslogMessage{
				Priority: []byte("34"),
				Message:  []byte("'su root' failed for lonvick on /dev/pts/8"),
				Hostname: []byte("mymachine"),
				Program:  []byte("postfix/smtpd"),
				Pid:      []byte("2000"),
				month:    []byte("Oct"),
				day:      []byte("11"),
				hour:     []byte("22"),
				minute:   []byte("14"),
				second:   []byte("15"),
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: SyslogMessage{
				Priority: []byte("34"),
				Message:  []byte("'su root' failed for lonvick on /dev/pts/8"),
				Hostname: []byte("wopr.mymachine.co"),
				Program:  []byte("postfix/smtpd"),
				Pid:      []byte("2000"),
				month:    []byte("Oct"),
				day:      []byte("11"),
				hour:     []byte("22"),
				minute:   []byte("14"),
				second:   []byte("15"),
			},
		},
		{
			log: []byte("<13>Feb 25 17:32:18 10.0.0.99 Use the Force!"),
			syslog: SyslogMessage{
				Message:  []byte("Use the Force!"),
				Hostname: []byte("10.0.0.99"),
				Priority: []byte("13"),
				month:    []byte("Feb"),
				day:      []byte("25"),
				hour:     []byte("17"),
				minute:   []byte("32"),
				second:   []byte("18"),
			},
		},
		{
			title: "Check relay + hostname alpha",
			log:   []byte("<13>Feb 25 17:32:18 wopr Use the Force!"),
			syslog: SyslogMessage{
				Message:  []byte("Use the Force!"),
				Hostname: []byte("wopr"),
				Priority: []byte("13"),
				month:    []byte("Feb"),
				day:      []byte("25"),
				hour:     []byte("17"),
				minute:   []byte("32"),
				second:   []byte("18"),
			},
		},
		{
			title: "Check relay + ipv6",
			log:   []byte("<13>Feb 25 17:32:18 2607:f0d0:1002:51::4 Use the Force!"),
			syslog: SyslogMessage{
				Message:  []byte("Use the Force!"),
				Hostname: []byte("2607:f0d0:1002:51::4"),
				Priority: []byte("13"),
				month:    []byte("Feb"),
				day:      []byte("25"),
				hour:     []byte("17"),
				minute:   []byte("32"),
				second:   []byte("18"),
			},
		},
		{
			title: "Check relay + ipv6",
			log:   []byte("<13>Feb 25 17:32:18 2607:f0d0:1002:0051:0000:0000:0000:0004 Use the Force!"),
			syslog: SyslogMessage{
				Message:  []byte("Use the Force!"),
				Hostname: []byte("2607:f0d0:1002:0051:0000:0000:0000:0004"),
				Priority: []byte("13"),
				month:    []byte("Feb"),
				day:      []byte("25"),
				hour:     []byte("17"),
				minute:   []byte("32"),
				second:   []byte("18"),
			},
		},
		{
			title: "Number inf the host",
			log:   []byte("<164>Oct 26 15:19:25 1.2.3.4 ASA1-2: Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
			syslog: SyslogMessage{
				Message:  []byte("Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
				Hostname: []byte("1.2.3.4"),
				Program:  []byte("ASA1-2"),
				Priority: []byte("164"),
				month:    []byte("Oct"),
				day:      []byte("26"),
				hour:     []byte("15"),
				minute:   []byte("19"),
				second:   []byte("25"),
			},
		},
		{
			log: []byte("<164>Oct 26 15:19:25 1.2.3.4 %ASA1-120: Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
			syslog: SyslogMessage{
				Message:  []byte("Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
				Hostname: []byte("1.2.3.4"),
				Program:  []byte("%ASA1-120"),
				Priority: []byte("164"),
				month:    []byte("Oct"),
				day:      []byte("26"),
				hour:     []byte("15"),
				minute:   []byte("19"),
				second:   []byte("25"),
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := &SyslogMessage{}
			Parse(test.log, l)
			assert.Equal(t, test.syslog.Message, l.Message)
			assert.Equal(t, test.syslog.Hostname, l.Hostname)
			assert.Equal(t, test.syslog.Priority, l.Priority)
			assert.Equal(t, test.syslog.Pid, l.Pid)
			assert.Equal(t, test.syslog.Program, l.Program)
			assert.Equal(t, test.syslog.month, l.month)
			assert.Equal(t, test.syslog.day, l.day)
			assert.Equal(t, test.syslog.hour, l.hour)
			assert.Equal(t, test.syslog.minute, l.minute)
			assert.Equal(t, test.syslog.second, l.second)
		})
	}
}

func TestDay(t *testing.T) {
	for d := 1; d <= 31; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct %2d 22:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := &SyslogMessage{}
			Parse([]byte(log), l)
			assert.Equal(t, []byte(strconv.Itoa(d)), l.day)
		})
	}
}

func TestHour(t *testing.T) {
	for d := 0; d <= 23; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 %02d:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := &SyslogMessage{}
			Parse([]byte(log), l)
			assert.Equal(t, []byte(fmt.Sprintf("%02d", d)), l.hour)
		})
	}
}

func TestMinute(t *testing.T) {
	for d := 0; d <= 59; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 10:%02d:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := &SyslogMessage{}
			Parse([]byte(log), l)
			assert.Equal(t, []byte(fmt.Sprintf("%02d", d)), l.minute)
		})
	}
}

func TestSecond(t *testing.T) {
	for d := 0; d <= 59; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 10:15:%02d mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := &SyslogMessage{}
			Parse([]byte(log), l)
			assert.Equal(t, []byte(fmt.Sprintf("%02d", d)), l.second)
		})
	}
}

func TestPriority(t *testing.T) {
	for d := 0; d <= 120; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<%d>Oct 11 10:15:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := &SyslogMessage{}
			Parse([]byte(log), l)
			assert.Equal(t, []byte(strconv.Itoa(d)), l.Priority)
		})
	}
}

func TestParserRe(t *testing.T) {
	t.SkipNow()
	l := &SyslogMessage{}
	log := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8")
	ParseRe(log, l)
	assert.Equal(t, []byte("mymachine su: 'su root' failed for lonvick on /dev/pts/8"), l.Message)
}

func BenchmarkParser(b *testing.B) {
	b.ReportAllocs()
	l := &SyslogMessage{}
	log := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8")
	for n := 0; n < b.N; n++ {
		Parse(log, l)
	}
}

func BenchmarkParserRegexp(b *testing.B) {
	b.ReportAllocs()
	l := &SyslogMessage{}
	log := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8")
	for n := 0; n < b.N; n++ {
		ParseRe(log, l)
	}
}
