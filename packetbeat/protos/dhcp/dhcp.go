package dhcp

import (
	"bytes"
	"encoding/hex"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"net"
)

var hardwareTypes = map[int]string{
	1:  "ethernet",
	6:  "ieee_802",
	7:  "arcnet",
	11: "localtalk",
	12: "localnet",
	14: "smds",
	15: "frame_relay",
	16: "atm",
	17: "hdlc",
	18: "fibre_channel",
	19: "atm",
	20: "serial_line",
}

var messageTypes = []string{
	"DHCPDISCOVER",
	"DHCPOFFER",
	"DHCPREQUEST",
	"DHCPDECLINE",
	"DHCPACK",
	"DHCPNAK",
	"DHCPRELEASE",
	"DHCPINFORM",
}

type dhcpPlugin struct {
	ports   []int
	results protos.Reporter
}

func (dhcp *dhcpPlugin) init(results protos.Reporter, config *dhcpConfig) {
	dhcp.ports = config.Ports
	dhcp.results = results
}

func (dhcp *dhcpPlugin) GetPorts() []int {
	return dhcp.ports
}

func payloadToRows(payload []byte) [][]byte {
	result := make([][]byte, 0)
	var idx = 0
	for idx < len(payload) {
		result = append(result, []byte{payload[idx], payload[idx+1], payload[idx+2], payload[idx+3]})
		idx += 4
	}
	return result
}

func bytesToIPv4(bytes []byte) string {
	ip := net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
	str := ip.String()
	if str == "0.0.0.0" {
		return ""
	}
	return str
}

func bytesToHardwareAddress(bytes []byte, length int) string {
	return net.HardwareAddr(bytes[0:length]).String()
}

func combineRows(rows [][]byte, start int, length int) []byte {
	combined := make([]byte, 0)
	end := start + length
	var idx = start
	for idx < end {
		combined = append(combined, rows[idx]...)
		idx++
	}
	return combined
}

func trimNullBytes(b []byte) []byte {
	index := bytes.Index(b, []byte{0})
	return b[0:index]
}

func extractOptions(payload []byte) []byte {
	result := make([]byte, 0)
	var idx = 240
	for idx < len(payload) {
		result = append(result, payload[idx])
		idx++
	}
	return result
}

func parseOptions(options []byte) map[int][]byte {
	result := make(map[int][]byte)
	var remaining = 0
	var option = 0
	var body = make([]byte, 0)
	var idx = 0
	for idx < len(options) {
		if remaining == 0 {
			if option != 0 {
				result[option] = body
				body = make([]byte, 0)
			}
			option = int(options[idx])
			if option == 0 || option == 255 {
				idx = len(options)
			} else {
				idx++
				remaining = int(options[idx])
			}
		} else {
			body = append(body, options[idx])
			remaining--
		}
		idx++
	}
	return result
}

func (dhcp *dhcpPlugin) parsePacket(pkt *protos.Packet) beat.Event {
	dhcpFields := make(map[string]interface{})
	rows := payloadToRows(pkt.Payload)
	dhcpFields["transaction_id"] = hex.EncodeToString(rows[1])
	dhcpFields["client_ip"] = bytesToIPv4(rows[3])
	dhcpFields["assigned_ip"] = bytesToIPv4(rows[4])
	dhcpFields["server_ip"] = bytesToIPv4(rows[5])
	dhcpFields["gateway_ip"] = bytesToIPv4(rows[6])
	hwaddr := combineRows(rows, 7, 4)
	dhcpFields["client_hwaddr"] = bytesToHardwareAddress(hwaddr, int(pkt.Payload[2]))
	serverName := trimNullBytes(combineRows(rows, 11, 16))
	dhcpFields["server_name"] = string(serverName)
	dhcpFields["op_code"] = int(pkt.Payload[0])
	dhcpFields["hops"] = int(pkt.Payload[3])
	dhcpFields["hardware_type"] = hardwareTypes[int(pkt.Payload[1])]
	options := extractOptions(pkt.Payload)
	parsedOptions := parseOptions(options)
	if parsedOptions[53] != nil {
		dhcpFields["message_type"] = messageTypes[int(parsedOptions[53][0])-1]
	}
	if parsedOptions[54] != nil {
		dhcpFields["server_identifier"] = bytesToIPv4(parsedOptions[54])
	}
	if parsedOptions[1] != nil {
		dhcpFields["subnet_mask"] = bytesToIPv4(parsedOptions[1])
	}
	event := beat.Event{
		Timestamp: pkt.Ts,
		Fields: map[string]interface{}{
			"transport":   "udp",
			"ip":          pkt.Tuple.DstIP.String(),
			"client_ip":   pkt.Tuple.SrcIP.String(),
			"port":        pkt.Tuple.DstPort,
			"client_port": pkt.Tuple.SrcPort,
			"type":        "dhcp",
			"dhcp":        dhcpFields,
		},
	}
	return event
}

func (dhcp *dhcpPlugin) ParseUDP(pkt *protos.Packet) {
	event := dhcp.parsePacket(pkt)
	dhcp.results(event)
}

func init() {
	protos.Register("dhcp", New)
}

// New dhcpPlugin is created.
func New(
	testMode bool,
	results protos.Reporter,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &dhcpPlugin{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			logp.Err("Error unpacking configuration: %s", err)
			return nil, err
		}
	}
	p.init(results, &config)
	return p, nil
}
