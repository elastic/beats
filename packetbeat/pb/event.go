// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package pb

import (
	"net"
	"reflect"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/flowhash"
	"github.com/elastic/ecs/code/go/ecs"
)

// FieldsKey is the key under which a *pb.Fields value may be stored in a
// beat.Event. The Packetbeat publisher will marshal those fields into the
// event at publish time.
const FieldsKey = "_packetbeat"

// Network direction values.
const (
	Inbound  = "inbound"
	Outbound = "outbound"
	Internal = "internal"
)

// Fields contains common fields used in Packetbeat events. Protocol
// implementations can publish a Fields pointer in a beat.Event and it will
// be marshaled into the event following the ECS schema where applicable.
//
// If client and server are nil then they will be populated with the source and
// destination values, respectively. Other fields like event.duration and
// and network.bytes are automatically computed at publish time.
type Fields struct {
	Source      *ecs.Source      `ecs:"source"`
	Destination *ecs.Destination `ecs:"destination"`
	Client      *ecs.Client      `ecs:"client"`
	Server      *ecs.Server      `ecs:"server"`
	Network     ecs.Network      `ecs:"network"`
	Event       ecs.Event        `ecs:"event"`

	SourceProcess      *ecs.Process `ecs:"source.process"`
	DestinationProcess *ecs.Process `ecs:"destination.process"`
	Process            *ecs.Process `ecs:"process"`

	Error struct {
		Message []string
	}

	ICMPType uint8 // ICMP message type for use in computing network.community_id.
	ICMPCode uint8 // ICMP message code for use in computing network.community_id.
}

// NewFields returns a new Fields value.
func NewFields() *Fields {
	return &Fields{
		Event: ecs.Event{
			Duration: -1,
			Kind:     "event",
			Category: "network_traffic",
		},
	}
}

// NewBeatEvent creates a new beat.Event populated with a Fields value and
// returns both.
func NewBeatEvent(timestamp time.Time) (beat.Event, *Fields) {
	pbf := NewFields()
	return beat.Event{
		Timestamp: timestamp,
		Fields: common.MapStr{
			FieldsKey: pbf,
		},
	}, pbf
}

// GetFields returns a pointer to a Fields object if one is stored within the
// given MapStr. It returns nil and no error if no Fields value is present.
func GetFields(m common.MapStr) (*Fields, error) {
	v, found := m[FieldsKey]
	if !found {
		return nil, nil
	}

	fields, ok := v.(*Fields)
	if !ok {
		return nil, errors.Errorf("%v must be a *types.Fields, but is %T", FieldsKey, fields)
	}
	return fields, nil
}

// SetSource populates the source fields with the endpoint data.
func (f *Fields) SetSource(endpoint *common.Endpoint) {
	if f.Source == nil {
		f.Source = &ecs.Source{}
	}
	f.Source.IP = endpoint.IP
	f.Source.Port = int64(endpoint.Port)
	f.Source.Domain = endpoint.Domain

	if endpoint.PID > 0 {
		f.SourceProcess = makeProcess(&endpoint.Process)
	}
}

// SetDestination populates the destination fields with the endpoint data.
func (f *Fields) SetDestination(endpoint *common.Endpoint) {
	if f.Destination == nil {
		f.Destination = &ecs.Destination{}
	}
	f.Destination.IP = endpoint.IP
	f.Destination.Port = int64(endpoint.Port)
	f.Destination.Domain = endpoint.Domain

	if endpoint.PID > 0 {
		f.DestinationProcess = makeProcess(&endpoint.Process)
	}
}

func makeProcess(p *common.Process) *ecs.Process {
	return &ecs.Process{
		Name:             p.Name,
		Args:             p.Args,
		Executable:       p.Exe,
		PID:              int64(p.PID),
		PPID:             int64(p.PPID),
		Start:            p.StartTime,
		WorkingDirectory: p.CWD,
	}
}

// ComputeValues computes derived values like network.bytes and writes them to f.
func (f *Fields) ComputeValues(localIPs []net.IP) error {
	var flow flowhash.Flow

	// network.bytes
	if f.Source != nil {
		flow.SourceIP = net.ParseIP(f.Source.IP)
		flow.SourcePort = uint16(f.Source.Port)
		f.Network.Bytes += f.Source.Bytes
	}
	if f.Destination != nil {
		flow.DestinationIP = net.ParseIP(f.Destination.IP)
		flow.DestinationPort = uint16(f.Destination.Port)
		f.Network.Bytes += f.Destination.Bytes
	}

	// network.community_id
	switch {
	case f.Network.Transport == "udp":
		flow.Protocol = 17
	case f.Network.Transport == "tcp":
		flow.Protocol = 6
	case f.Network.Transport == "icmp":
		flow.Protocol = 1
	case f.Network.Transport == "ipv6-icmp":
		flow.Protocol = 58
	}
	flow.ICMP.Type = f.ICMPType
	flow.ICMP.Code = f.ICMPCode
	if flow.Protocol > 0 && len(flow.SourceIP) > 0 && len(flow.DestinationIP) > 0 {
		f.Network.CommunityID = flowhash.CommunityID.Hash(flow)
	}

	// network.type
	if f.Network.Type == "" {
		if len(flow.SourceIP) > 0 {
			if flow.SourceIP.To4() != nil {
				f.Network.Type = "ipv4"
			} else {
				f.Network.Type = "ipv6"
			}
		} else if len(flow.DestinationIP) > 0 {
			if flow.DestinationIP.To4() != nil {
				f.Network.Type = "ipv4"
			} else {
				f.Network.Type = "ipv6"
			}
		}
	}

	// network.direction
	if len(localIPs) > 0 && f.Network.Direction == "" {
		if flow.SourceIP != nil {
			for _, ip := range localIPs {
				if flow.SourceIP.Equal(ip) {
					f.Network.Direction = Outbound
					break
				}
			}
		}
		if flow.DestinationIP != nil {
			for _, ip := range localIPs {
				if flow.DestinationIP.Equal(ip) {
					if f.Network.Direction == Outbound {
						f.Network.Direction = Internal
					} else {
						f.Network.Direction = Inbound
					}
					break
				}
			}
		}
	}

	// process (dest process will take priority)
	if f.DestinationProcess != nil {
		f.Process = f.DestinationProcess
	} else if f.SourceProcess != nil {
		f.Process = f.SourceProcess
	}

	// event.duration
	if f.Event.Duration == -1 && !f.Event.Start.IsZero() && !f.Event.End.IsZero() {
		if elapsed := f.Event.End.Sub(f.Event.Start); elapsed >= 0 {
			f.Event.Duration = elapsed
		}
	}

	// event.dataset
	if f.Event.Dataset == "" {
		f.Event.Dataset = f.Network.Protocol
	}

	// client
	if f.Client == nil && f.Source != nil {
		client := ecs.Client(*f.Source)
		f.Client = &client
	}

	// server
	if f.Server == nil && f.Destination != nil {
		server := ecs.Server(*f.Destination)
		f.Server = &server
	}

	return nil
}

// MarshalMapStr marshals the fields into MapStr. It returns an error if there
// is a problem writing the keys to the given map (like if an intermediate key
// exists and is not a map).
func (f *Fields) MarshalMapStr(m common.MapStr) error {
	typ := reflect.TypeOf(*f)
	val := reflect.ValueOf(*f)

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		tag := structField.Tag.Get("ecs")
		if tag == "" {
			continue
		}

		fieldValue := val.Field(i)
		if !fieldValue.IsValid() || isEmptyValue(fieldValue) {
			continue
		}

		if err := marshalStruct(m, tag, fieldValue); err != nil {
			return err
		}
	}

	if len(f.Error.Message) == 1 {
		m.Put("error.message", f.Error.Message[0])
	} else if len(f.Error.Message) > 1 {
		m.Put("error.message", f.Error.Message)
	}

	return nil
}

// MarshalStruct marshals any struct containing ecs or packetbeat tags into the
// given MapStr. Zero-value and nil fields are always omitted.
func MarshalStruct(m common.MapStr, key string, val interface{}) error {
	return marshalStruct(m, key, reflect.ValueOf(val))
}

func marshalStruct(m common.MapStr, key string, val reflect.Value) error {
	// Dereference pointers.
	if val.Type().Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}

		val = val.Elem()
	}

	// Ignore zero values.
	if !val.IsValid() {
		return nil
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		tag := getTag(structField)
		if tag == "" {
			break
		}

		fieldValue := val.Field(i)
		if !fieldValue.IsValid() || isEmptyValue(fieldValue) {
			continue
		}

		if _, err := m.Put(key+"."+tag, fieldValue.Interface()); err != nil {
			return err
		}
	}
	return nil
}

func getTag(f reflect.StructField) string {
	if tag := f.Tag.Get("ecs"); tag != "" {
		return tag
	}
	return f.Tag.Get("packetbeat")
}

// isEmptyValue returns true if the given value is empty.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int64:
		if duration, ok := v.Interface().(time.Duration); ok {
			return duration < 0
		}
		return v.Int() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	switch t := v.Interface().(type) {
	case time.Time:
		return t.IsZero()
	}
	return false
}
