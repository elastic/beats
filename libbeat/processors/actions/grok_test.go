package actions

import (
	"testing"

	"regexp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

var field = "msg"
var testGrokConfig, _ = common.NewConfigFrom(map[string]interface{}{
	"field":    "msg",
	"patterns": []string{`(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z),(?P<client_ip>\d+\.\d+\.\d+\.\d+)?`},
})

func TestGrokMissingKey(t *testing.T) {
	input := common.MapStr{
		"datacenter": "watson",
	}

	actual := getGrokActualValue(t, testGrokConfig, input)

	expected := common.MapStr{
		"datacenter": "watson",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestGrokSimpleMessage(t *testing.T) {
	input := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
	}

	actual := getGrokActualValue(t, testGrokConfig, input)

	expected := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
		"timestamp":  "2012-03-04T22:33:01.003Z",
		"client_ip":  "127.0.0.1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestGrokIp(t *testing.T) {
	var testGrokConfigIP, _ = common.NewConfigFrom(map[string]interface{}{
		"field":    "msg",
		"patterns": []string{`%{TIMESTAMP_ISO8601:timestamp},%{IP:client_ip}?`},
		//"patterns": []string{`%{IP:client_ip}`},
	})

	input := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
	}

	actual := getGrokActualValue(t, testGrokConfigIP, input)

	expected := common.MapStr{
		"datacenter": "watson",
		"msg":        "2012-03-04T22:33:01.003Z,127.0.0.1",
		"timestamp":  "2012-03-04T22:33:01.003Z",
		"client_ip":  "127.0.0.1",
	}

	assert.Equal(t, expected.String(), actual.String())
}

func TestGrokPatterns(t *testing.T) {
	type matchCase struct {
		matchedString string
		matches       map[string]string
	}

	testsExpressions := map[string][]matchCase{
		",%{EMAILADDRESS:email},": []matchCase{
			matchCase{",ramon_garcia@myaddress.org,", map[string]string{"email": "ramon_garcia@myaddress.org"}},
			matchCase{",0200@amazon.com,", map[string]string{"email": "0200@amazon.com"}}},
		":%{HTTPDUSER:user}:": []matchCase{
			matchCase{":frobenius:", map[string]string{"user": "frobenius"}},
			matchCase{":frobenius@somedomain.org:", map[string]string{"user": "frobenius@somedomain.org"}},
		},
		":%{INT:val}:": []matchCase{
			matchCase{":132:", map[string]string{"val": "132"}},
			matchCase{":-2:", map[string]string{"val": "-2"}},
			matchCase{":+23:", map[string]string{"val": "+23"}},
		},
	}
	for pattern, testMatches := range testsExpressions {
		expandedPattern, err := grokExpandPattern(pattern, []string{}, map[string]string{})
		if err != nil {
			logp.Err("Error expanding Grok expression")
			t.Error(err)
		}
		reg, err := regexp.Compile(expandedPattern)
		if err != nil {
			logp.Err("Error expanding compiling expression")
			t.Error(err)
		}
		for _, matchCase := range testMatches {
			matches := reg.FindStringSubmatchIndex(matchCase.matchedString)
			if len(matches) == 0 {
				t.Error("Expected match but did not match", pattern, matchCase.matchedString)
				continue
			}
			subexps := reg.SubexpNames()
			matchMap := make(map[string]string)

			for i, subexp := range subexps {
				if len(subexp) > 0 {
					if matches[2*i] >= 0 {
						matchMap[subexp] = matchCase.matchedString[matches[2*i]:matches[2*i+1]]
					}
				}
			}
			assert.Equal(t, matchCase.matches, matchMap)
		}
	}
}

// tcpflags tcpsyn tcpack tcpwin icmptype icmpcode info path

func TestGrokWindowsFirewallLog1(t *testing.T) {
	input := common.MapStr{
		"datacenter": "watson",
		"msg":        "2015-11-22 04:14:00 DROP TCP 10.31.42.53 10.0.0.1 52209 359 52 S 3190407656 0 8192 - - - RECEIVE",
	}
	var testGrokWindowsFirewallLog, _ = common.NewConfigFrom(map[string]interface{}{
		"field": "msg",
		"patterns": []string{`%{TIMESTAMP_ISO8601:timestamp} %{WORD:action} %{WORD:protocol} (?:%{IP:source_ip}|[-]) (?:%{IP:destination_ip}|-) (?:%{INT:source_port}|-) (?:%{INT:destination_port}|-) ` +
			`(?:%{INT:size}|-) (?:-|%{WORD:tcp_flags}) (?:-|%{WORD:tcp_syn}) (?:-|%{WORD:tcp_ack}) (?:-|%{WORD:tcp_win}) (?:-|%{WORD:icmp_type}) (?:-|%{WORD:icmp_code}) (?:-|%{WORD:info}) (?:-|%{WORD:direction})`},
		//"patterns": []string{`%{IP:client_ip}`},
	})

	actual := getGrokActualValue(t, testGrokWindowsFirewallLog, input)
	expected := common.MapStr{
		"datacenter":       "watson",
		"msg":              "2015-11-22 04:14:00 DROP TCP 10.31.42.53 10.0.0.1 52209 359 52 S 3190407656 0 8192 - - - RECEIVE",
		"timestamp":        "2015-11-22 04:14:00",
		"action":           "DROP",
		"protocol":         "TCP",
		"source_ip":        "10.31.42.53",
		"destination_ip":   "10.0.0.1",
		"source_port":      "52209",
		"destination_port": "359",
		"size":             "52",
		"tcp_flags":        "S",
		"tcp_syn":          "3190407656",
		"tcp_ack":          "0",
		"tcp_win":          "8192",
		"direction":        "RECEIVE",
	}

	assert.Equal(t, expected.String(), actual.String())

}

func TestGrokLinuxFirewall(t *testing.T) {
	input := common.MapStr{
		"msg": "2017-10-12T20:03:28.64642+01:00 fumunchu kernel: [ 4587.744455] iptables rejected input" +
			" IN=eth0 OUT= MAC=5b:1d:87:11:a2:c3:9a:90:12:3a:4d:3f:54:64 " +
			"SRC=5.188.10.182 DST=192.168.32.1 LEN=60 TOS=0x00 PREC=0x00 TTL=47 ID=3322 DF PROTO=TCP SPT=78965" +
			" DPT=22 WINDOW=29200 RES=0x00 SYN URGP=0",
	}
	var testGrokLinuxFirewallLog, _ = common.NewConfigFrom(map[string]interface{}{
		"field": "msg",
		"patterns": []string{`%{TIMESTAMP_ISO8601:timestamp} %{WORD:host} %{WORD:subsystem}\s+\[\s*\d+.\d+\]\+iptables rejected input ` +
			`IN=(?:%{INTERFACE:in_interface}|-) ` +
			`OUT=(?:%{INTERFACE:out_interface}|-) MAC=%{MAC:dest_mac}:%{MAC:src_mac}:\d{2}:\d{2} ` +
			`SRC=%{IP:src_ip} DST=%{IP:src_ip} LEN=%{NUMBER:len} TOS=%{NUMBER:tos} PREC=%{BASE16NUM:prec} ` +
			`TTL=%{NUMBER:ttl} ID=%{NUMBER:id} %{WORD:flags} PROTO=%{WORD:proto} ` +
			`SPT=%{NUMBER:source_port} DPT=%{NUMBER:dest_port} WINDOW=%{NUMBER:window} ` +
			`RES=%{BASE16NUM:res} %{WORD:syn} URGP=%{NUMBER:urg}`,
			`%{TIMESTAMP_ISO8601:timestamp} %{WORD:host} %{WORD:subsystem}\s+\[\s*\d+.\d+\]\+iptables rejected input `,
		},
		"additional_pattern_definitions": map[string]string{
			"RSYSLOG_TIMESTAMP": `\d+-\d+-\d+T\d+:\d+:\d+.\d+(?:\+\d+:\d+)?`,
			"INTERFACE":         `\w+\d+`,
			"MAC":               `\d{2}:\d{2}:\d{2}:\d{2}:\d{2}:\d{2}`,
		},
		//"patterns": []string{`%{IP:client_ip}`},
	})

	actual := getGrokActualValue(t, testGrokLinuxFirewallLog, input)
	expected := common.MapStr{
		"timestamp":    "2017-10-12T20:03:28.64642+01:00",
		"host":         "fumanchu",
		"subsystem":    "kernel",
		"in_interface": "eth0",
		"dest_mac":     "5b:1d:87:11:a2:c3",
		"src_mac":      "9a:90:12:3a:4d:3f",
		"src_ip":       "5.188.10.182",
		"dst_ip":       "192.168.32.1",
		"len":          "60",
		"tos":          "0x00",
		"prec":         "0x00",
		"ttl":          "47",
		"id":           "3322",
		"flags":        "DF",
		"proto":        "TCP",
		"dest_port":    "22",
		"source_port":  "78965",
		"window":       "29200",
		"res":          "0x00",
		"syn":          "syn",
		"urg":          "0",
	}
	assert.Equal(t, expected.String(), actual.String())
}

func getGrokActualValue(t *testing.T, config *common.Config, input common.MapStr) common.MapStr {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	p, err := newGrok(*config)
	if err != nil {
		logp.Err("Error initializing Grok ")
		t.Fatal(err)
	}

	actual, err := p.Run(input)

	return actual
}
