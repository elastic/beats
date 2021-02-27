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

var map_getEventCategoryActivity = {
	keyvaluepairs: {
		"Allowed": constant("Permit"),
		"Blocked": constant("Deny"),
	},
};

var hdr1 = match("HEADER#0:0001", "message", "%{data->} ZSCALERNSS: time=%{hfld2->} %{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hyear}^^timezone=%{timezone}^^%{payload}", processor_chain([
	setc("header_id","0001"),
	setc("messageid","ZSCALERNSS_1"),
]));

var select1 = linear_select([
	hdr1,
]);

var part1 = match("MESSAGE#0:ZSCALERNSS_1", "nwparser.payload", "action=%{action}^^reason=%{result}^^hostname=%{hostname}^^protocol=%{protocol}^^serverip=%{daddr}^^url=%{url}^^urlcategory=%{filter}^^urlclass=%{info}^^dlpdictionaries=%{fld3}^^dlpengine=%{fld4}^^filetype=%{filetype}^^threatcategory=%{category}^^threatclass=%{vendor_event_cat}^^pagerisk=%{fld8}^^threatname=%{threat_name}^^clientpublicIP=%{fld9}^^ClientIP=%{saddr}^^location=%{fld11}^^refererURL=%{web_referer}^^useragent=%{user_agent}^^department=%{user_dept}^^user=%{username}^^event_id=%{id}^^clienttranstime=%{fld17}^^requestmethod=%{web_method}^^requestsize=%{sbytes}^^requestversion=%{fld20}^^status=%{resultcode}^^responsesize=%{rbytes}^^responseversion=%{fld23}^^transactionsize=%{bytes}", processor_chain([
	setc("eventcategory","1605000000"),
	setf("fqdn","hostname"),
	setf("msg","$MSG"),
	date_time({
		dest: "event_time",
		args: ["hmonth","hday","hyear","hhour","hmin","hsec"],
		fmts: [
			[dB,dF,dW,dN,dU,dO],
		],
	}),
	lookup({
		dest: "nwparser.ec_activity",
		map: map_getEventCategoryActivity,
		key: field("action"),
	}),
	setc("ec_theme","Communication"),
	setc("ec_subject","User"),
]));

var msg1 = msg("ZSCALERNSS_1", part1);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"ZSCALERNSS_1": msg1,
	}),
]);
