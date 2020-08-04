//  Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
//  or more contributor license agreements. Licensed under the Elastic License;
//  you may not use this file except in compliance with the Elastic License.

function DeviceProcessor() {
	var builder = new processor.Chain();
	builder.Add(save_flags);
	builder.Add(strip_syslog_priority);
	builder.Add(chain1);
	builder.Add(populate_fields);
	builder.Add(restore_flags);
	var chain = builder.Build();
	return {
		process: chain.Run,
	}
}

var map_operationtype = {
	keyvaluepairs: {
		"0": constant("NONE"),
		"1": constant("Created"),
		"2": constant("Modified"),
		"3": constant("Removed"),
	},
	"default": constant("0"),
};

var map_AdminTaskType = {
	keyvaluepairs: {
		"0": constant("Application"),
		"1": constant("Application Isolation Environment"),
		"10": constant("Server Group"),
		"11": constant("User"),
		"12": constant("Policy"),
		"13": constant("Monitoring Profile"),
		"14": constant("Load Manager"),
		"15": constant("Virtual IP Farm Range"),
		"16": constant("Virtual IP Server Range"),
		"17": constant("Print Driver"),
		"18": constant("Database"),
		"19": constant("Zone"),
		"2": constant("AIE Application"),
		"4": constant("Farm"),
		"5": constant("File Type Association"),
		"6": constant("Folder"),
		"7": constant("Installation Manager Application"),
		"8": constant("Printer"),
		"9": constant("Server"),
	},
	"default": constant("0"),
};

var dup1 = setc("eventcategory","1612000000");

var dup2 = date_time({
	dest: "event_time",
	args: ["fld1","fld2"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup3 = match("MESSAGE#3:Broker_SDK", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3}^^%{event_type}^^%{saddr}^^%{event_description}^^%{application}", processor_chain([
	dup1,
	dup2,
]));

var hdr1 = match("HEADER#0:0001", "message", "%citrixxa: %{hdatetime}^^%{messageid}^^%{payload}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdatetime"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("payload"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%citrixxa: %{hdatetime}^^%{msgIdPart1->} %{msgIdPart2}^^%{payload}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("msgIdPart1"),
			constant("_"),
			field("msgIdPart2"),
		],
	}),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdatetime"),
			constant("^^"),
			field("msgIdPart1"),
			constant(" "),
			field("msgIdPart2"),
			constant("^^"),
			field("payload"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
]);

var part1 = match("MESSAGE#0:CONFIGINFO", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3}^^%{event_type}^^%{administrator}^^%{shost}^^%{hostname}^^%{operation_id}^^%{obj_type}^^%{obj_name}", processor_chain([
	dup1,
	dup2,
	lookup({
		dest: "nwparser.operation_id",
		map: map_operationtype,
		key: field("operation_id"),
	}),
	lookup({
		dest: "nwparser.obj_type",
		map: map_AdminTaskType,
		key: field("obj_type"),
	}),
]));

var msg1 = msg("CONFIGINFO", part1);

var part2 = match("MESSAGE#1:SESSIONINFO", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3}^^%{event_type}^^%{username}^^%{hostname}^^%{saddr}^^%{application}^^%{fld4->} %{fld5}.%{fld6}", processor_chain([
	dup1,
	date_time({
		dest: "starttime",
		args: ["fld1","fld2"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
		],
	}),
	date_time({
		dest: "endtime",
		args: ["fld4","fld5"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
		],
	}),
]));

var msg2 = msg("SESSIONINFO", part2);

var part3 = match("MESSAGE#2:APPINFO", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3}^^%{event_type}^^%{domain}^^%{group_object}^^%{hostname}^^%{application}", processor_chain([
	dup1,
	dup2,
]));

var msg3 = msg("APPINFO", part3);

var msg4 = msg("Broker_SDK", dup3);

var msg5 = msg("ConfigurationLogging", dup3);

var msg6 = msg("Monitor", dup3);

var msg7 = msg("Analytics", dup3);

var msg8 = msg("Storefront", dup3);

var msg9 = msg("Configuration", dup3);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"APPINFO": msg3,
		"Analytics": msg7,
		"Broker_SDK": msg4,
		"CONFIGINFO": msg1,
		"Configuration": msg9,
		"ConfigurationLogging": msg5,
		"Monitor": msg6,
		"SESSIONINFO": msg2,
		"Storefront": msg8,
	}),
]);

var part4 = match("MESSAGE#3:Broker_SDK", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3}^^%{event_type}^^%{saddr}^^%{event_description}^^%{application}", processor_chain([
	dup1,
	dup2,
]));
