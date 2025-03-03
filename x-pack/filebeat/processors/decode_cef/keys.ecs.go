// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"errors"
	"strings"

	"github.com/elastic/beats/v7/x-pack/filebeat/processors/decode_cef/cef"
)

type mappedField struct {
	// Target is the ECS target field for the mapped field.
	Target string

	// Translate is the mapping function required to translate
	// the CEF field data into an ECS-conformant format.
	// If Translate is nil, no translation is done.
	// Translate should not mutate the input and should
	// return an error if the input data cannot be correctly
	// mapped to ECS-formatted data for the target field.
	Translate func(in *cef.Field) (interface{}, error)
}

var ecsExtensionMapping = map[string]mappedField{
	"agentAddress":   {Target: "agent.ip"},
	"agentDnsDomain": {Target: "agent.name"},
	"agentHostName":  {Target: "agent.name"},
	"agentId":        {Target: "agent.id"},
	"agentMacAddress": {
		Target:    "agent.mac",
		Translate: ecsMAC,
	},
	"agentReceiptTime":        {Target: "event.created"},
	"agentType":               {Target: "agent.type"},
	"agentVersion":            {Target: "agent.version"},
	"applicationProtocol":     {Target: "network.application"},
	"bytesIn":                 {Target: "source.bytes"},
	"bytesOut":                {Target: "destination.bytes"},
	"customerExternalID":      {Target: "organization.id"},
	"customerURI":             {Target: "organization.name"},
	"destinationAddress":      {Target: "destination.ip"},
	"destinationDnsDomain":    {Target: "destination.domain"},
	"destinationGeoLatitude":  {Target: "destination.geo.location.lat"},
	"destinationGeoLongitude": {Target: "destination.geo.location.lon"},
	"destinationHostName":     {Target: "destination.domain"},
	"destinationMacAddress": {
		Target:    "destination.mac",
		Translate: ecsMAC,
	},
	"destinationPort":              {Target: "destination.port"},
	"destinationProcessId":         {Target: "destination.process.pid"},
	"destinationProcessName":       {Target: "destination.process.name"},
	"destinationServiceName":       {Target: "destination.service.name"},
	"destinationTranslatedAddress": {Target: "destination.nat.ip"},
	"destinationTranslatedPort":    {Target: "destination.nat.port"},
	"destinationUserId":            {Target: "destination.user.id"},
	"destinationUserName":          {Target: "destination.user.name"},
	"destinationUserPrivileges":    {Target: "destination.user.group.name"},
	"deviceAction":                 {Target: "event.action"},
	"deviceAddress": {
		Target: "observer.ip",
		Translate: func(in *cef.Field) (interface{}, error) {
			return []string{in.String}, nil
		},
	},
	"deviceDirection": {
		Target: "network.direction",
		Translate: func(in *cef.Field) (interface{}, error) {
			switch in.String {
			case "0":
				return "inbound", nil
			case "1":
				return "outbound", nil
			default:
				return nil, errors.New("deviceDirection must be 0 or 1")
			}
		},
	},
	"deviceDnsDomain": {Target: "observer.hostname"},
	"deviceHostName":  {Target: "observer.hostname"},
	"deviceMacAddress": {
		Target:    "observer.mac",
		Translate: ecsMAC,
	},
	"devicePayloadId":          {Target: "event.id"},
	"deviceProcessId":          {Target: "process.pid"},
	"deviceProcessName":        {Target: "process.name"},
	"deviceReceiptTime":        {Target: "@timestamp"},
	"deviceTimeZone":           {Target: "event.timezone"},
	"endTime":                  {Target: "event.end"},
	"eventId":                  {Target: "event.id"},
	"eventOutcome":             {Target: "event.outcome"},
	"fileCreateTime":           {Target: "file.created"},
	"fileId":                   {Target: "file.inode"},
	"fileModificationTime":     {Target: "file.mtime"},
	"filename":                 {Target: "file.name"},
	"filePath":                 {Target: "file.path"},
	"filePermission":           {Target: "file.group"},
	"fileSize":                 {Target: "file.size"},
	"fileType":                 {Target: "file.type"},
	"message":                  {Target: "message"},
	"requestClientApplication": {Target: "user_agent.original"},
	"requestContext": {
		Target: "http.request.referrer",
		Translate: func(in *cef.Field) (interface{}, error) {
			// Does the string look like URL?
			if strings.HasPrefix(in.String, "http") {
				return in.String, nil
			}
			return nil, nil
		},
	},
	"requestMethod":      {Target: "http.request.method"},
	"requestUrl":         {Target: "url.original"},
	"sourceAddress":      {Target: "source.ip"},
	"sourceDnsDomain":    {Target: "source.domain"},
	"sourceGeoLatitude":  {Target: "source.geo.location.lat"},
	"sourceGeoLongitude": {Target: "source.geo.location.lon"},
	"sourceHostName":     {Target: "source.domain"},
	"sourceMacAddress": {
		Target:    "source.mac",
		Translate: ecsMAC,
	},
	"sourcePort":              {Target: "source.port"},
	"sourceProcessId":         {Target: "source.process.pid"},
	"sourceProcessName":       {Target: "source.process.name"},
	"sourceServiceName":       {Target: "source.service.name"},
	"sourceTranslatedAddress": {Target: "source.nat.ip"},
	"sourceTranslatedPort":    {Target: "source.nat.port"},
	"sourceUserId":            {Target: "source.user.id"},
	"sourceUserName":          {Target: "source.user.name"},
	"sourceUserPrivileges":    {Target: "source.user.group.name"},
	"startTime":               {Target: "event.start"},
	"transportProtocol": {
		Target: "network.transport",
		Translate: func(in *cef.Field) (interface{}, error) {
			return strings.ToLower(in.String), nil
		},
	},
	"type": {Target: "event.kind"},
}

func ecsMAC(in *cef.Field) (interface{}, error) {
	return strings.ToUpper(strings.ReplaceAll(in.String, ":", "-")), nil
}
