// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"math/bits"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
)

type vpcFlowField struct {
	Name        string                                           // Name of the VPC flow field that is added to our events.
	Type        dataType                                         // Data type to convert the string into.
	Enrich      func(originalFields mapstr.M, value interface{}) // Optional enrichment function to add new derived fields into the 'target_field' namespace.
	ECSMappings []ecsFieldMapping                                // List of ECS fields to create or derive from this field.
}

type ecsFieldMapping struct {
	Target    string                                                         // ECS field target.
	Transform func(targetField string, value interface{}, event *beat.Event) // Optional transform to modify the value. If omitted the value is copied.
}

var nameToFieldMap map[string]vpcFlowField

func init() {
	nameToFieldMap = make(map[string]vpcFlowField, len(vpcFlowFields))
	for _, field := range vpcFlowFields {
		nameToFieldMap[field.Name] = field
	}
}

var vpcFlowFields = [...]vpcFlowField{
	{
		Name: "version",
		Type: integerType,
	},
	{
		Name: "account_id",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{Target: "cloud.account.id"},
		},
	},
	{
		Name: "interface_id",
		Type: stringType,
	},
	{
		Name: "srcaddr",
		Type: ipType,
		ECSMappings: []ecsFieldMapping{
			{Target: "source.address"},
			{Target: "source.ip"},
			{
				Target: "network.type",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					if ip := value.(string); strings.Contains(ip, ".") {
						event.PutValue(targetField, "ipv4") //nolint:errcheck // This can only fail if 'network' is not an object.
					} else {
						event.PutValue(targetField, "ipv6") //nolint:errcheck // This can only fail if 'network' is not an object.
					}
				},
			},
		},
	},
	{
		Name: "dstaddr",
		Type: ipType,
		ECSMappings: []ecsFieldMapping{
			{Target: "destination.address"},
			{Target: "destination.ip"},
		},
	},
	{
		Name: "srcport",
		Type: integerType,
		ECSMappings: []ecsFieldMapping{
			{Target: "source.port"},
		},
	},
	{
		Name: "dstport",
		Type: integerType,
		ECSMappings: []ecsFieldMapping{
			{Target: "destination.port"},
		},
	},
	{
		Name: "protocol",
		Type: integerType,
		ECSMappings: []ecsFieldMapping{
			{
				Target: "network.iana_number",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					protocol := value.(int32)
					event.PutValue(targetField, strconv.Itoa(int(protocol))) //nolint:errcheck // This can only fail if 'network' is not an object.
				},
			},
			{
				Target: "network.transport",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					var name string
					switch protocol := value.(int32); protocol {
					case 0:
						name = "hopopt"
					case 1:
						name = "icmp"
					case 2:
						name = "igmp"
					case 6:
						name = "tcp"
					case 8:
						name = "egp"
					case 17:
						name = "udp"
					case 47:
						name = "gre"
					case 50:
						name = "esp"
					case 58:
						name = "ipv6-icmp"
					case 112:
						name = "vrrp"
					case 132:
						name = "sctp"
					}

					if name != "" {
						event.PutValue(targetField, name) //nolint:errcheck // This can only fail if 'network' is not an object.
					}
				},
			},
		},
	},
	{
		Name: "packets",
		Type: longType,
		ECSMappings: []ecsFieldMapping{
			{Target: "source.packets"},
			{Target: "network.packets"},
		},
	},
	{
		Name: "bytes",
		Type: longType,
		ECSMappings: []ecsFieldMapping{
			{Target: "source.bytes"},
			{Target: "network.bytes"},
		},
	},
	{
		Name: "start",
		Type: timestampType,
		ECSMappings: []ecsFieldMapping{
			{Target: "event.start"},
		},
	},
	{
		Name: "end",
		Type: timestampType,
		ECSMappings: []ecsFieldMapping{
			{Target: "event.end"},
			{Target: "@timestamp"},
		},
	},
	{
		Name: "action",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{
				Target: "event.outcome",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					var outcome string

					switch s := value.(string); s {
					case "ACCEPT":
						outcome = "success"
					case "REJECT":
						outcome = "failure"
					}

					if outcome != "" {
						event.PutValue(targetField, outcome) //nolint:errcheck // This can only fail if 'event' is not an object.
					}
				},
			},
			{
				Target: "event.action",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					event.PutValue(targetField, strings.ToLower(value.(string))) //nolint:errcheck // This can only fail if 'event' is not an object.
				},
			},
			{
				Target: "event.type",
				Transform: func(targetField string, value interface{}, event *beat.Event) {
					var eventType string

					switch s := value.(string); s {
					case "ACCEPT":
						eventType = "allowed"
					case "REJECT":
						eventType = "denied"
					}

					if len(eventType) > 0 {
						// The processor always adds event.type: [connection] in ECS mode.
						v, _ := event.GetValue(targetField)
						if eventTypes, ok := v.([]string); ok {
							event.PutValue(targetField, append(eventTypes, eventType)) //nolint:errcheck // This can only fail if 'event' is not an object.
							return
						}

						event.PutValue(targetField, []string{eventType}) //nolint:errcheck // This can only fail if 'event' is not an object.
					}
				},
			},
		},
	},
	{Name: "log_status", Type: stringType},
	{Name: "vpc_id", Type: stringType},
	{Name: "subnet_id", Type: stringType},
	{
		Name: "instance_id",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{Target: "cloud.instance.id"},
		},
	},
	{
		Name: "tcp_flags",
		Type: integerType,
		Enrich: func(originalFields mapstr.M, value interface{}) {
			flag := value.(int32)
			flags := make([]string, 0, bits.OnesCount8(uint8(flag)))
			if flag&0x01 != 0 {
				flags = append(flags, "fin")
			}
			if flag&0x02 != 0 {
				flags = append(flags, "syn")
			}
			if flag&0x04 != 0 {
				flags = append(flags, "rst")
			}
			if flag&0x08 != 0 {
				flags = append(flags, "psh")
			}
			if flag&0x10 != 0 {
				flags = append(flags, "ack")
			}
			if flag&0x20 != 0 {
				flags = append(flags, "urg")
			}

			if len(flags) > 0 {
				originalFields["tcp_flags_array"] = flags
			}
		},
	},
	{Name: "type", Type: stringType},
	// TODO: Could these be used in some way to set source.nat.* and destination.nat.*.
	{Name: "pkt_srcaddr", Type: ipType},
	{Name: "pkt_dstaddr", Type: ipType},
	{
		Name: "region",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{Target: "cloud.region"},
		},
	},
	{
		Name: "az_id",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{Target: "cloud.availability_zone"},
		},
	},
	{Name: "sublocation_type", Type: stringType},
	{Name: "sublocation_id", Type: stringType},
	{Name: "pkt_src_aws_service", Type: stringType},
	{Name: "pkt_dst_aws_service", Type: stringType},
	{
		Name: "flow_direction",
		Type: stringType,
		ECSMappings: []ecsFieldMapping{
			{Target: "network.direction"},
		},
	},
	{Name: "traffic_path", Type: integerType},
}
