// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type dataType uint8

// List of dataTypes.
const (
	unset dataType = iota
	Integer
	Long
	Float
	Double
	String
	Boolean
	IP
	Timestamp
)

type ecsMode uint8

// List of modes.
const (
	copyMode ecsMode = iota
	renameMode
)

type mappedField struct {
	Target    string
	Type      dataType
	Translate func(in string) (interface{}, error)
}

var ecsKeyMapping = map[string]mappedField{
	"agentAddress": {
		Target: "observer.ip",
		Type:   IP,
	},
	"agentDnsDomain": {
		Target: "observer.hostname",
		Type:   String,
	},
	"agentHostName": {
		Target: "observer.hostname",
		Type:   String,
	},
	"agentId": {
		Target: "observer.serial_number",
		Type:   String,
	},
	"agentMacAddress": {
		Target: "observer.mac",
		Type:   String,
	},
	"agentReceiptTime": {
		Target: "event.created",
		Type:   Timestamp,
	},
	"agentType": {
		Target: "observer.type",
		Type:   String,
	},
	"agentVersion": {
		Target: "observer.version",
		Type:   String,
	},
	"applicationProtocol": {
		Target: "network.application",
		Type:   String,
	},
	"bytesIn": {
		Target: "source.bytes",
		Type:   Integer,
	},
	"bytesOut": {
		Target: "destination.bytes",
		Type:   Integer,
	},
	"customerExternalID": {
		Target: "organization.id",
		Type:   String,
	},
	"customerURI": {
		Target: "organization.name",
		Type:   String,
	},
	"destinationAddress": {
		Target: "destination.ip",
		Type:   IP,
	},
	"destinationDnsDomain": {
		Target: "destination.domain",
		Type:   String,
	},
	"destinationGeoLatitude": {
		Target: "destination.geo.location",
		Type:   Double,
	},
	"destinationGeoLongitude": {
		Target: "destination.geo.location",
		Type:   Double,
	},
	"destinationHostName": {
		Target: "destination.domain",
		Type:   String,
	},
	"destinationMacAddress": {
		Target: "destination.mac",
		Type:   String,
	},
	"destinationPort": {
		Target: "destination.port",
		Type:   Integer,
	},
	"destinationProcessId": {
		Target: "destination.process.pid",
		Type:   Integer,
	},
	"destinationProcessName": {
		Target: "destination.process.name",
		Type:   String,
	},
	"destinationServiceName": {
		Target: "service.name",
		Type:   String,
	},
	"destinationTranslatedAddress": {
		Target: "destination.nat.ip",
		Type:   IP,
	},
	"destinationTranslatedPort": {
		Target: "destination.nat.port",
		Type:   Integer,
	},
	"destinationUserId": {
		Target: "destination.user.id",
		Type:   String,
	},
	"destinationUserName": {
		Target: "destination.user.name",
		Type:   String,
	},
	"destinationUserPrivileges": {
		Target: "destination.user.group",
		Type:   String,
	},
	"deviceAction": {
		Target: "event.action",
		Type:   String,
	},
	"deviceAddress": {
		Target: "host.ip",
		Type:   IP,
	},
	"deviceDirection": {
		Target: "network.direction",
		Translate: func(in string) (interface{}, error) {
			switch in {
			case "0":
				return "inbound", nil
			case "1":
				return "outbound", nil
			default:
				return nil, errors.Errorf("deviceDirection must be 0 or 1")
			}
		},
	},
	"deviceExternalId": {
		Target: "host.id",
		Type:   String,
	},
	"deviceHostName": {
		Target: "host.hostname",
		Type:   String,
	},
	"deviceMacAddress": {
		Target: "host.mac",
		Type:   String,
	},
	"devicePayloadId": {
		Target: "event.id",
		Type:   String,
	},
	"deviceProcessId": {
		Target: "process.pid",
		Type:   Integer,
	},
	"deviceProcessName": {
		Target: "process.name",
		Type:   String,
	},
	"deviceReceiptTime": {
		Target: "@timestamp",
		Type:   Timestamp,
	},
	"deviceTimeZone": {
		Target: "event.timezone",
		Type:   String,
	},
	"endTime": {
		Target: "event.end",
		Type:   Timestamp,
	},
	"eventId": {
		Target: "event.id",
		Type:   Long,
	},
	"eventOutcome": {
		Target: "event.outcome",
		Type:   String,
	},
	"fileCreateTime": {
		Target: "file.created",
		Type:   Timestamp,
	},
	"fileId": {
		Target: "file.inode",
		Type:   String,
	},
	"fileModificationTime": {
		Target: "file.mtime",
		Type:   Timestamp,
	},
	"filename": {
		Target: "file.uid",
		Type:   String,
	},
	"filePath": {
		Target: "file.path",
		Type:   String,
	},
	"filePermission": {
		Target: "file.group",
		Type:   String,
	},
	"fileSize": {
		Target: "file.size",
		Type:   Integer,
	},
	"fileType": {
		Target: "file.type",
		Type:   String,
	},
	"message": {
		Target: "message",
		Type:   String,
	},
	"rawEvent": {
		Target: "event.original",
		Type:   String,
	},
	"requestClientApplication": {
		Target: "user_agent.original",
		Type:   String,
	},
	"requestContext": {
		Target: "http.request.referrer",
		Type:   String,
	},
	"requestMethod": {
		Target: "http.request.method",
		Type:   String,
	},
	"requestUrl": {
		Target: "url.original",
		Type:   String,
	},
	"sourceAddress": {
		Target: "source.ip",
		Type:   IP,
	},
	"sourceDnsDomain": {
		Target: "source.domain",
		Type:   String,
	},
	"sourceGeoLatitude": {
		Target: "source.geo.location",
		Type:   Double,
	},
	"sourceGeoLongitude": {
		Target: "source.geo.location",
		Type:   Double,
	},
	"sourceHostName": {
		Target: "source.domain",
		Type:   String,
	},
	"sourceMacAddress": {
		Target: "source.mac",
		Type:   String,
	},
	"sourcePort": {
		Target: "source.port",
		Type:   Integer,
	},
	"sourceProcessId": {
		Target: "source.process.pid",
		Type:   Integer,
	},
	"sourceProcessName": {
		Target: "source.process.name",
		Type:   String,
	},
	"sourceTranslatedAddress": {
		Target: "source.nat.ip",
		Type:   IP,
	},
	"sourceTranslatedPort": {
		Target: "source.nat.port",
		Type:   Integer,
	},
	"sourceUserId": {
		Target: "source.user.id",
		Type:   String,
	},
	"sourceUserName": {
		Target: "source.user.name",
		Type:   String,
	},
	"sourceUserPrivileges": {
		Target: "source.user.group",
		Type:   String,
	},
	"startTime": {
		Target: "event.start",
		Type:   Timestamp,
	},
	"transportProtocol": {
		Target: "network.transport",
		Type:   String,
	},
	"type": {
		Target: "event.kind",
		Type:   Integer,
	},
}

func toType(value string, typ dataType) (interface{}, error) {
	switch typ {
	case String:
		return value, nil
	case Long:
		return toLong(value)
	case Integer:
		return toInteger(value)
	case Float:
		return toFloat(value)
	case Double:
		return toDouble(value)
	case Boolean:
		return toBoolean(value)
	case IP:
		return toIP(value)
	default:
		panic("invalid data type")
	}
}

func toLong(v string) (int64, error) {
	return strconv.ParseInt(v, 0, 64)
}

func toInteger(v string) (int32, error) {
	i, err := strconv.ParseInt(v, 0, 32)
	return int32(i), err
}

func toFloat(v string) (float32, error) {
	f, err := strconv.ParseFloat(v, 32)
	return float32(f), err
}

func toDouble(v string) (float64, error) {
	f, err := strconv.ParseFloat(v, 64)
	return f, err
}

func toBoolean(v string) (bool, error) {
	return strconv.ParseBool(v)
}

func toIP(v string) (string, error) {
	// This is validating that the value is an IP.
	if net.ParseIP(v) != nil {
		return v, nil
	}
	return "", errors.New("value is not a valid IP address")
}

var timeLayouts = []string{
	// MMM dd HH:mm:ss.SSS zzz
	"Jan _2 15:04:05.000 MST",
	// MMM dd HH:mm:sss.SSS
	"Jan _2 15:04:05.000",
	// MMM dd HH:mm:ss zzz
	"Jan _2 15:04:05 MST",
	// MMM dd HH:mm:ss
	"Jan _2 15:04:05",
	// MMM dd yyyy HH:mm:ss.SSS zzz
	"Jan _2 2006 15:04:05.000 MST",
	// MMM dd yyyy HH:mm:ss.SSS
	"Jan _2 2006 15:04:05.000",
	// MMM dd yyyy HH:mm:ss zzz
	"Jan _2 2006 15:04:05 MST",
	// MMM dd yyyy HH:mm:ss
	"Jan _2 2006 15:04:05",
}

func toTimestamp(v string) (time.Time, error) {
	if unixMs, err := toLong(v); err == nil {
		return time.Unix(0, unixMs*int64(time.Millisecond)), nil
	}

	for _, layout := range timeLayouts {
		ts, err := time.ParseInLocation(layout, v, time.UTC)
		if err == nil {
			// Use current year if no year is zero.
			if ts.Year() == 0 {
				currentYear := time.Now().In(ts.Location()).Year()
				ts = ts.AddDate(currentYear, 0, 0)
			}

			return ts, nil
		}
	}

	return time.Time{}, errors.New("value is not a valid timestamp")
}
