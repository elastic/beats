// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cef

type mappedField struct {
	Target string
	Type   DataType
}

// extensionMapping is a mapping of CEF key names to full field names and data
// types. This mapping was generated from tables contained in "Micro Focus
// Security ArcSight Common Event Format Version 25" dated September 28, 2017.
var extensionMapping = map[string]mappedField{
	"agt": {
		Target: "agentAddress",
		Type:   IPType,
	},
	"agentDnsDomain": {
		Target: "agentDnsDomain",
		Type:   StringType,
	},
	"ahost": {
		Target: "agentHostName",
		Type:   StringType,
	},
	"aid": {
		Target: "agentId",
		Type:   StringType,
	},
	"amac": {
		Target: "agentMacAddress",
		Type:   MACAddressType,
	},
	"agentNtDomain": {
		Target: "agentNtDomain",
		Type:   StringType,
	},
	"art": {
		Target: "agentReceiptTime",
		Type:   TimestampType,
	},
	"atz": {
		Target: "agentTimeZone",
		Type:   StringType,
	},
	"agentTranslatedAddress": {
		Target: "agentTranslatedAddress",
		Type:   IPType,
	},
	"agentTranslatedZoneExternalID": {
		Target: "agentTranslatedZoneExternalID",
		Type:   StringType,
	},
	"agentTranslatedZoneURI": {
		Target: "agentTranslatedZoneURI",
		Type:   StringType,
	},
	"at": {
		Target: "agentType",
		Type:   StringType,
	},
	"av": {
		Target: "agentVersion",
		Type:   StringType,
	},
	"agentZoneExternalID": {
		Target: "agentZoneExternalID",
		Type:   StringType,
	},
	"agentZoneURI": {
		Target: "agentZoneURI",
		Type:   StringType,
	},
	"app": {
		Target: "applicationProtocol",
		Type:   StringType,
	},
	"cnt": {
		Target: "baseEventCount",
		Type:   IntegerType,
	},
	"in": {
		Target: "bytesIn",
		Type:   IntegerType,
	},
	"out": {
		Target: "bytesOut",
		Type:   IntegerType,
	},
	"customerExternalID": {
		Target: "customerExternalID",
		Type:   StringType,
	},
	"customerURI": {
		Target: "customerURI",
		Type:   StringType,
	},
	"dst": {
		Target: "destinationAddress",
		Type:   IPType,
	},
	"destinationDnsDomain": {
		Target: "destinationDnsDomain",
		Type:   StringType,
	},
	"dlat": {
		Target: "destinationGeoLatitude",
		Type:   DoubleType,
	},
	"dlong": {
		Target: "destinationGeoLongitude",
		Type:   DoubleType,
	},
	"dhost": {
		Target: "destinationHostName",
		Type:   StringType,
	},
	"dmac": {
		Target: "destinationMacAddress",
		Type:   MACAddressType,
	},
	"dntdom": {
		Target: "destinationNtDomain",
		Type:   StringType,
	},
	"dpt": {
		Target: "destinationPort",
		Type:   IntegerType,
	},
	"dpid": {
		Target: "destinationProcessId",
		Type:   IntegerType,
	},
	"dproc": {
		Target: "destinationProcessName",
		Type:   StringType,
	},
	"destinationServiceName": {
		Target: "destinationServiceName",
		Type:   StringType,
	},
	"destinationTranslatedAddress": {
		Target: "destinationTranslatedAddress",
		Type:   IPType,
	},
	"destinationTranslatedPort": {
		Target: "destinationTranslatedPort",
		Type:   IntegerType,
	},
	"destinationTranslatedZoneExternalID": {
		Target: "destinationTranslatedZoneExternalID",
		Type:   StringType,
	},
	"destinationTranslatedZoneURI": {
		Target: "destinationTranslatedZoneURI",
		Type:   StringType,
	},
	"duid": {
		Target: "destinationUserId",
		Type:   StringType,
	},
	"duser": {
		Target: "destinationUserName",
		Type:   StringType,
	},
	"dpriv": {
		Target: "destinationUserPrivileges",
		Type:   StringType,
	},
	"destinationZoneExternalID": {
		Target: "destinationZoneExternalID",
		Type:   StringType,
	},
	"destinationZoneURI": {
		Target: "destinationZoneURI",
		Type:   StringType,
	},
	"act": {
		Target: "deviceAction",
		Type:   StringType,
	},
	"dvc": {
		Target: "deviceAddress",
		Type:   IPType,
	},
	"cfp1Label": {
		Target: "deviceCustomFloatingPoint1Label",
		Type:   StringType,
	},
	"cfp3Label": {
		Target: "deviceCustomFloatingPoint3Label",
		Type:   StringType,
	},
	"cfp4Label": {
		Target: "deviceCustomFloatingPoint4Label",
		Type:   StringType,
	},
	"deviceCustomDate1": {
		Target: "deviceCustomDate1",
		Type:   TimestampType,
	},
	"deviceCustomDate1Label": {
		Target: "deviceCustomDate1Label",
		Type:   StringType,
	},
	"deviceCustomDate2": {
		Target: "deviceCustomDate2",
		Type:   TimestampType,
	},
	"deviceCustomDate2Label": {
		Target: "deviceCustomDate2Label",
		Type:   StringType,
	},
	"cfp1": {
		Target: "deviceCustomFloatingPoint1",
		Type:   FloatType,
	},
	"cfp2": {
		Target: "deviceCustomFloatingPoint2",
		Type:   FloatType,
	},
	"cfp2Label": {
		Target: "deviceCustomFloatingPoint2Label",
		Type:   StringType,
	},
	"cfp3": {
		Target: "deviceCustomFloatingPoint3",
		Type:   FloatType,
	},
	"cfp4": {
		Target: "deviceCustomFloatingPoint4",
		Type:   FloatType,
	},
	"c6a1Label": {
		Target: "deviceCustomIPv6Address1Label",
		Type:   StringType,
	},
	"c6a4": {
		Target: "deviceCustomIPv6Address4",
		Type:   IPType,
	},
	"C6a4Label": {
		Target: "deviceCustomIPv6Address4Label",
		Type:   StringType,
	},
	"c6a1": {
		Target: "deviceCustomIPv6Address1",
		Type:   IPType,
	},
	"c6a3": {
		Target: "deviceCustomIPv6Address3",
		Type:   IPType,
	},
	"c6a3Label": {
		Target: "deviceCustomIPv6Address3Label",
		Type:   StringType,
	},
	"cn1": {
		Target: "deviceCustomNumber1",
		Type:   LongType,
	},
	"cn1Label": {
		Target: "deviceCustomNumber1Label",
		Type:   StringType,
	},
	"cn2": {
		Target: "DeviceCustomNumber2",
		Type:   LongType,
	},
	"cn2Label": {
		Target: "deviceCustomNumber2Label",
		Type:   StringType,
	},
	"cn3": {
		Target: "deviceCustomNumber3",
		Type:   LongType,
	},
	"cn3Label": {
		Target: "deviceCustomNumber3Label",
		Type:   StringType,
	},
	"cs1": {
		Target: "deviceCustomString1",
		Type:   StringType,
	},
	"cs1Label": {
		Target: "deviceCustomString1Label",
		Type:   StringType,
	},
	"cs2": {
		Target: "deviceCustomString2",
		Type:   StringType,
	},
	"cs2Label": {
		Target: "deviceCustomString2Label",
		Type:   StringType,
	},
	"cs3": {
		Target: "deviceCustomString3",
		Type:   StringType,
	},
	"cs3Label": {
		Target: "deviceCustomString3Label",
		Type:   StringType,
	},
	"cs4": {
		Target: "deviceCustomString4",
		Type:   StringType,
	},
	"cs4Label": {
		Target: "deviceCustomString4Label",
		Type:   StringType,
	},
	"cs5": {
		Target: "deviceCustomString5",
		Type:   StringType,
	},
	"cs5Label": {
		Target: "deviceCustomString5Label",
		Type:   StringType,
	},
	"cs6": {
		Target: "deviceCustomString6",
		Type:   StringType,
	},
	"cs6Label": {
		Target: "deviceCustomString6Label",
		Type:   StringType,
	},
	"deviceDirection": {
		Target: "deviceDirection",
		Type:   IntegerType,
	},
	"deviceDnsDomain": {
		Target: "deviceDnsDomain",
		Type:   StringType,
	},
	"cat": {
		Target: "deviceEventCategory",
		Type:   StringType,
	},
	"deviceExternalId": {
		Target: "deviceExternalId",
		Type:   StringType,
	},
	"deviceFacility": {
		Target: "deviceFacility",
		Type:   StringType,
	},
	"dvchost": {
		Target: "deviceHostName",
		Type:   StringType,
	},
	"deviceInboundInterface": {
		Target: "deviceInboundInterface",
		Type:   StringType,
	},
	"dvcmac": {
		Target: "deviceMacAddress",
		Type:   MACAddressType,
	},
	"deviceNtDomain": {
		Target: "deviceNtDomain",
		Type:   StringType,
	},
	"DeviceOutboundInterface": {
		Target: "deviceOutboundInterface",
		Type:   StringType,
	},
	"DevicePayloadId": {
		Target: "devicePayloadId",
		Type:   StringType,
	},
	"dvcpid": {
		Target: "deviceProcessId",
		Type:   IntegerType,
	},
	"deviceProcessName": {
		Target: "deviceProcessName",
		Type:   StringType,
	},
	"rt": {
		Target: "deviceReceiptTime",
		Type:   TimestampType,
	},
	"dtz": {
		Target: "deviceTimeZone",
		Type:   StringType,
	},
	"deviceTranslatedAddress": {
		Target: "deviceTranslatedAddress",
		Type:   IPType,
	},
	"deviceTranslatedZoneExternalID": {
		Target: "deviceTranslatedZoneExternalID",
		Type:   StringType,
	},
	"deviceTranslatedZoneURI": {
		Target: "deviceTranslatedZoneURI",
		Type:   StringType,
	},
	"deviceZoneExternalID": {
		Target: "deviceZoneExternalID",
		Type:   StringType,
	},
	"deviceZoneURI": {
		Target: "deviceZoneURI",
		Type:   StringType,
	},
	"end": {
		Target: "endTime",
		Type:   TimestampType,
	},
	"eventId": {
		Target: "eventId",
		Type:   LongType,
	},
	"outcome": {
		Target: "eventOutcome",
		Type:   StringType,
	},
	"externalId": {
		Target: "externalId",
		Type:   StringType,
	},
	"fileCreateTime": {
		Target: "fileCreateTime",
		Type:   TimestampType,
	},
	"fileHash": {
		Target: "fileHash",
		Type:   StringType,
	},
	"fileId": {
		Target: "fileId",
		Type:   StringType,
	},
	"fileModificationTime": {
		Target: "fileModificationTime",
		Type:   TimestampType,
	},
	"fname": {
		Target: "filename",
		Type:   StringType,
	},
	"filePath": {
		Target: "filePath",
		Type:   StringType,
	},
	"filePermission": {
		Target: "filePermission",
		Type:   StringType,
	},
	"fsize": {
		Target: "fileSize",
		Type:   IntegerType,
	},
	"fileType": {
		Target: "fileType",
		Type:   StringType,
	},
	"flexDate1": {
		Target: "flexDate1",
		Type:   TimestampType,
	},
	"flexDate1Label": {
		Target: "flexDate1Label",
		Type:   StringType,
	},
	"flexString1": {
		Target: "flexString1",
		Type:   StringType,
	},
	"flexString2": {
		Target: "flexString2",
		Type:   StringType,
	},
	"flexString1Label": {
		Target: "flexString1Label",
		Type:   StringType,
	},
	"flexString2Label": {
		Target: "flexString2Label",
		Type:   StringType,
	},
	"msg": {
		Target: "message",
		Type:   StringType,
	},
	"oldFileCreateTime": {
		Target: "oldFileCreateTime",
		Type:   TimestampType,
	},
	"oldFileHash": {
		Target: "oldFileHash",
		Type:   StringType,
	},
	"oldFileId": {
		Target: "oldFileId",
		Type:   StringType,
	},
	"oldFileModificationTime": {
		Target: "oldFileModificationTime",
		Type:   TimestampType,
	},
	"oldFileName": {
		Target: "oldFileName",
		Type:   StringType,
	},
	"oldFilePath": {
		Target: "oldFilePath",
		Type:   StringType,
	},
	"oldFilePermission": {
		Target: "oldFilePermission",
		Type:   StringType,
	},
	"oldFileSize": {
		Target: "oldFileSize",
		Type:   IntegerType,
	},
	"oldFileType": {
		Target: "oldFileType",
		Type:   StringType,
	},
	"rawEvent": {
		Target: "rawEvent",
		Type:   StringType,
	},
	"reason": {
		Target: "Reason",
		Type:   StringType,
	},
	"requestClientApplication": {
		Target: "requestClientApplication",
		Type:   StringType,
	},
	"requestContext": {
		Target: "requestContext",
		Type:   StringType,
	},
	"requestCookies": {
		Target: "requestCookies",
		Type:   StringType,
	},
	"requestMethod": {
		Target: "requestMethod",
		Type:   StringType,
	},
	"request": {
		Target: "requestUrl",
		Type:   StringType,
	},
	"src": {
		Target: "sourceAddress",
		Type:   IPType,
	},
	"sourceDnsDomain": {
		Target: "sourceDnsDomain",
		Type:   StringType,
	},
	"slat": {
		Target: "sourceGeoLatitude",
		Type:   DoubleType,
	},
	"slong": {
		Target: "sourceGeoLongitude",
		Type:   DoubleType,
	},
	"shost": {
		Target: "sourceHostName",
		Type:   StringType,
	},
	"smac": {
		Target: "sourceMacAddress",
		Type:   MACAddressType,
	},
	"sntdom": {
		Target: "sourceNtDomain",
		Type:   StringType,
	},
	"spt": {
		Target: "sourcePort",
		Type:   IntegerType,
	},
	"spid": {
		Target: "sourceProcessId",
		Type:   IntegerType,
	},
	"sproc": {
		Target: "sourceProcessName",
		Type:   StringType,
	},
	"sourceServiceName": {
		Target: "sourceServiceName",
		Type:   StringType,
	},
	"sourceTranslatedAddress": {
		Target: "sourceTranslatedAddress",
		Type:   IPType,
	},
	"sourceTranslatedPort": {
		Target: "sourceTranslatedPort",
		Type:   IntegerType,
	},
	"sourceTranslatedZoneExternalID": {
		Target: "sourceTranslatedZoneExternalID",
		Type:   StringType,
	},
	"sourceTranslatedZoneURI": {
		Target: "sourceTranslatedZoneURI",
		Type:   StringType,
	},
	"suid": {
		Target: "sourceUserId",
		Type:   StringType,
	},
	"suser": {
		Target: "sourceUserName",
		Type:   StringType,
	},
	"spriv": {
		Target: "sourceUserPrivileges",
		Type:   StringType,
	},
	"sourceZoneExternalID": {
		Target: "sourceZoneExternalID",
		Type:   StringType,
	},
	"sourceZoneURI": {
		Target: "sourceZoneURI",
		Type:   StringType,
	},
	"start": {
		Target: "startTime",
		Type:   TimestampType,
	},
	"proto": {
		Target: "transportProtocol",
		Type:   StringType,
	},
	"type": {
		Target: "type",
		Type:   IntegerType,
	},

	// This is an ArcSight categorization field that is commonly used, but its
	// short name is not contained in the documentation used for the above list.
	"catdt": {
		Target: "categoryDeviceType",
		Type:   StringType,
	},
	"mrt": {
		Target: "managerReceiptTime",
		Type:   TimestampType,
	},
}
