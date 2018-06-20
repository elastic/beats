package syslog

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseSyslog(t *testing.T) {
	tests := []struct {
		title  string
		log    []byte
		syslog event
	}{
		{
			title: "no timezone in date",
			log:   []byte("<190>2018-06-19 02:13:38 super mon message"),
			syslog: event{
				priority: 190,
				message:  "mon message",
				hostname: "super",
				program:  "",
				month:    6,
				pid:      -1,
				day:      19,
				year:     2018,
				hour:     2,
				minute:   13,
				second:   38,
			},
		},
		{
			title: "no timezone in date with nanoseconds",
			log:   []byte("<190>2018-06-19 02:13:38.0004 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 4000,
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322-07:00 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.FixedZone("", int(-7*time.Hour)),
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322-0700 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.FixedZone("", int(-7*time.Hour)),
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322-0730 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.FixedZone("", int(-7*time.Hour)+int(-30*time.Minute)),
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322-07:10 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.FixedZone("", int(-7*time.Hour)+int(-10*time.Minute)),
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322-07 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.FixedZone("", int(-7*time.Hour)),
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322Z super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.UTC,
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322Z+0000 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.UTC,
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322Z+00:00 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.UTC,
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38.635322Z+00 super mon message"),
			syslog: event{
				priority:   190,
				message:    "mon message",
				hostname:   "super",
				program:    "",
				month:      6,
				pid:        -1,
				day:        19,
				year:       2018,
				hour:       2,
				minute:     13,
				second:     38,
				nanosecond: 6353220,
				loc:        time.UTC,
			},
		},
		{
			title: "time in ISO8601 format",
			log:   []byte("<190>2018-06-19T02:13:38Z+00 super mon message"),
			syslog: event{
				priority: 190,
				message:  "mon message",
				hostname: "super",
				program:  "",
				month:    6,
				pid:      -1,
				day:      19,
				year:     2018,
				hour:     2,
				minute:   13,
				second:   38,
				loc:      time.UTC,
			},
		},
		{
			title: "priority and timestamp defined as 2018-05-08T10:31:24 (rfc3339)",
			log:   []byte("<38>2018-05-08T10:31:24 localhost prg00000[1234]: seq: 0000000000, thread: 0000, runid: 1525768284, stamp: 2018-05-08T10:31:24 PADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPAD DPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADD"),
			syslog: event{
				priority: 38,
				message:  "seq: 0000000000, thread: 0000, runid: 1525768284, stamp: 2018-05-08T10:31:24 PADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPAD DPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADDPADD",
				hostname: "localhost",
				program:  "prg00000",
				pid:      1234,
				month:    5,
				day:      8,
				hour:     10,
				minute:   31,
				second:   24,
				year:     2016,
			},
		},
		{
			title: "timestamp defined as 2018-05-08T10:31:24 (rfc3339)",
			log:   []byte("2016-05-08T10:31:24 localhost prg00000[1234]: seq: 0000000000, thread: 0000, runid: 1525768284"),
			syslog: event{
				priority: -1,
				message:  "seq: 0000000000, thread: 0000, runid: 1525768284",
				hostname: "localhost",
				program:  "prg00000",
				pid:      1234,
				month:    5,
				day:      8,
				hour:     10,
				minute:   31,
				second:   24,
				year:     2016,
			},
		},
		{
			title: "timestamp with nanosecond defined as 2018-05-08T10:31:24.0004 (rfc3339)",
			log:   []byte("2016-05-08T10:31:24.0004 localhost prg00000[1234]: seq: 0000000000, thread: 0000, runid: 1525768284"),
			syslog: event{
				priority:   -1,
				message:    "seq: 0000000000, thread: 0000, runid: 1525768284",
				hostname:   "localhost",
				program:    "prg00000",
				pid:        1234,
				month:      5,
				day:        8,
				hour:       10,
				minute:     31,
				second:     24,
				year:       2016,
				nanosecond: 4000,
			},
		},
		{
			title: "message only",
			log:   []byte("--- last message repeated 1 time ---"),
			syslog: event{
				priority: -1,
				message:  "--- last message repeated 1 time ---",
				hostname: "",
				program:  "",
				pid:      -1,
				month:    -1,
				day:      -1,
				hour:     -1,
				minute:   -1,
				second:   -1,
			},
		},
		{
			title: "time and message only",
			log:   []byte("Oct 11 22:14:15 --- last message repeated 1 time ---"),
			syslog: event{
				priority: -1,
				message:  "--- last message repeated 1 time ---",
				hostname: "",
				program:  "",
				pid:      -1,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			title: "time with nanosecond",
			log:   []byte("Oct 11 22:14:15.0000005 --- last message repeated 1 time ---"),
			syslog: event{
				priority:   -1,
				message:    "--- last message repeated 1 time ---",
				hostname:   "",
				program:    "",
				pid:        -1,
				month:      10,
				day:        11,
				hour:       22,
				minute:     14,
				second:     15,
				nanosecond: 5,
			},
		},
		{
			title: "No priority defined",
			log:   []byte("Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority: -1,
				message:  "'su root' failed for lonvick on /dev/pts/8",
				hostname: "mymachine",
				program:  "su",
				pid:      230,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine su[230]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority: 34,
				message:  "'su root' failed for lonvick on /dev/pts/8",
				hostname: "mymachine",
				program:  "su",
				pid:      230,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15.57643 mymachine su: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority:   34,
				message:    "'su root' failed for lonvick on /dev/pts/8",
				hostname:   "mymachine",
				program:    "su",
				pid:        -1,
				month:      10,
				day:        11,
				hour:       22,
				minute:     14,
				second:     15,
				nanosecond: 5764300,
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority: 34,
				message:  "'su root' failed for lonvick on /dev/pts/8",
				hostname: "mymachine",
				program:  "su",
				pid:      -1,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority: 34,
				message:  "'su root' failed for lonvick on /dev/pts/8",
				hostname: "mymachine",
				program:  "postfix/smtpd",
				pid:      2000,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			log: []byte("<34>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8"),
			syslog: event{
				priority: 34,
				message:  "'su root' failed for lonvick on /dev/pts/8",
				hostname: "wopr.mymachine.co",
				program:  "postfix/smtpd",
				pid:      2000,
				month:    10,
				day:      11,
				hour:     22,
				minute:   14,
				second:   15,
			},
		},
		{
			log: []byte("<13>Feb 25 17:32:18 10.0.0.99 Use the Force!"),
			syslog: event{
				message:  "Use the Force!",
				hostname: "10.0.0.99",
				priority: 13,
				pid:      -1,
				month:    2,
				day:      25,
				hour:     17,
				minute:   32,
				second:   18,
			},
		},
		{
			title: "Check relay + hostname alpha",
			log:   []byte("<13>Feb 25 17:32:18 wopr Use the Force!"),
			syslog: event{
				message:  "Use the Force!",
				hostname: "wopr",
				priority: 13,
				pid:      -1,
				month:    2,
				day:      25,
				hour:     17,
				minute:   32,
				second:   18,
			},
		},
		{
			title: "Check relay + ipv6",
			log:   []byte("<13>Feb 25 17:32:18 2607:f0d0:1002:51::4 Use the Force!"),
			syslog: event{
				message:  "Use the Force!",
				hostname: "2607:f0d0:1002:51::4",
				priority: 13,
				pid:      -1,
				month:    2,
				day:      25,
				hour:     17,
				minute:   32,
				second:   18,
			},
		},
		{
			title: "Check relay + ipv6",
			log:   []byte("<13>Feb 25 17:32:18 2607:f0d0:1002:0051:0000:0000:0000:0004 Use the Force!"),
			syslog: event{
				message:  "Use the Force!",
				hostname: "2607:f0d0:1002:0051:0000:0000:0000:0004",
				priority: 13,
				pid:      -1,
				month:    2,
				day:      25,
				hour:     17,
				minute:   32,
				second:   18,
			},
		},
		{
			title: "Number inf the host",
			log:   []byte("<164>Oct 26 15:19:25 1.2.3.4 ASA1-2: Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
			syslog: event{
				message:  "Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]",
				hostname: "1.2.3.4",
				program:  "ASA1-2",
				priority: 164,
				pid:      -1,
				month:    10,
				day:      26,
				hour:     15,
				minute:   19,
				second:   25,
			},
		},
		{
			log: []byte("<164>Oct 26 15:19:25 1.2.3.4 %ASA1-120: Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]"),
			syslog: event{
				message:  "Deny udp src DRAC:10.1.2.3/43434 dst outside:192.168.0.1/53 by access-group \"acl_drac\" [0x0, 0x0]",
				hostname: "1.2.3.4",
				program:  "%ASA1-120",
				priority: 164,
				pid:      -1,
				month:    10,
				day:      26,
				hour:     15,
				minute:   19,
				second:   25,
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s:%s", test.title, string(test.log)), func(t *testing.T) {
			l := newEvent()
			Parse(test.log, l)
			assert.Equal(t, test.syslog.Message(), l.Message())
			assert.Equal(t, test.syslog.Hostname(), l.Hostname())
			assert.Equal(t, test.syslog.Priority(), l.Priority())
			assert.Equal(t, test.syslog.Pid(), l.Pid())
			assert.Equal(t, test.syslog.Program(), l.Program())
			assert.Equal(t, test.syslog.Month(), l.Month())
			assert.Equal(t, test.syslog.Day(), l.Day())
			assert.Equal(t, test.syslog.Hour(), l.Hour())
			assert.Equal(t, test.syslog.Minute(), l.Minute())
			assert.Equal(t, test.syslog.Second(), l.Second())
			assert.Equal(t, test.syslog.Nanosecond(), l.Nanosecond())
			assert.Equal(t, test.syslog.loc, l.loc)
		})
	}
}

func TestDay(t *testing.T) {
	for d := 1; d <= 31; d++ {
		t.Run(fmt.Sprintf("Day %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct %2d 22:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := newEvent()
			Parse([]byte(log), l)
			assert.Equal(t, d, l.Day())
		})
	}
}

func TestHour(t *testing.T) {
	for d := 0; d <= 23; d++ {
		t.Run(fmt.Sprintf("Hour %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 %02d:14:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := newEvent()
			Parse([]byte(log), l)
			assert.Equal(t, d, l.Hour())
		})
	}
}

func TestMinute(t *testing.T) {
	for d := 0; d <= 59; d++ {
		t.Run(fmt.Sprintf("Minute %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 10:%02d:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := newEvent()
			Parse([]byte(log), l)
			assert.Equal(t, d, l.Minute())
		})
	}
}

func TestSecond(t *testing.T) {
	for d := 0; d <= 59; d++ {
		t.Run(fmt.Sprintf("Second %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<34>Oct 11 10:15:%02d mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := newEvent()
			Parse([]byte(log), l)
			assert.Equal(t, d, l.Second())
		})
	}
}

func TestPriority(t *testing.T) {
	for d := 1; d <= 120; d++ {
		t.Run(fmt.Sprintf("Priority %d", d), func(t *testing.T) {
			log := fmt.Sprintf("<%d>Oct 11 10:15:15 mymachine postfix/smtpd[2000]: 'su root' failed for lonvick on /dev/pts/8", d)
			l := newEvent()
			Parse([]byte(log), l)
			assert.Equal(t, d, l.Priority())
		})
		return
	}
}

var e *event

func BenchmarkParser(b *testing.B) {
	b.ReportAllocs()
	l := newEvent()
	log := []byte("<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8")
	for n := 0; n < b.N; n++ {
		Parse(log, l)
		e = l
	}
}
