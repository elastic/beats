//  Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
//  or more contributor license agreements. Licensed under the Elastic License;
//  you may not use this file except in compliance with the Elastic License.
var tvm = {
	pair_separator: " ",
	kv_separator: "=",
	open_quote: "\"",
	close_quote: "\"",
};

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

var map_getEventLegacyCategoryName = {
	keyvaluepairs: {
		"1301000000": constant("Auth.Failures"),
		"1302000000": constant("Auth.Successful"),
		"1401070000": constant("User.Activity.Logoff"),
	},
	"default": constant("Other.Default"),
};

var map_getEventLegacyCategory = {
	keyvaluepairs: {
		"Accepted": dup16,
		"Failed": constant("1301000000"),
		"login accepted": dup16,
		"timed out": constant("1401070000"),
	},
	"default": constant("1901000000"),
};

var map_getProtocolName = {
	keyvaluepairs: {
		"0": dup17,
		"1": dup18,
		"17": dup20,
		"21": dup21,
		"3": constant("GGP"),
		"6": dup19,
		"HOPOPT": dup17,
		"icmp": dup18,
		"prm": dup21,
		"tcp": dup19,
		"udp": dup20,
	},
};

var dup1 = setc("messageid","generic_fortinetmgr");

var dup2 = setc("messageid","generic_fortinetmgr_1");

var dup3 = setc("eventcategory","1803000000");

var dup4 = setf("msg","$MSG");

var dup5 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup6 = setf("event_source","hdevice");

var dup7 = setf("hardware_id","hfld1");

var dup8 = setf("event_type","hlog_type");

var dup9 = setf("category","hfld3");

var dup10 = setf("severity","hseverity");

var dup11 = setc("eventcategory","1605000000");

var dup12 = field("event_cat");

var dup13 = setc("eventcategory","1801000000");

var dup14 = call({
	dest: "nwparser.bytes",
	fn: CALC,
	args: [
		field("sbytes"),
		constant("+"),
		field("rbytes"),
	],
});

var dup15 = field("fld6");

var dup16 = constant("1302000000");

var dup17 = constant("HOPOPT");

var dup18 = constant("ICMP");

var dup19 = constant("TCP");

var dup20 = constant("UDP");

var dup21 = constant("PRM");

var dup22 = lookup({
	dest: "nwparser.event_cat_name",
	map: map_getEventLegacyCategoryName,
	key: dup12,
});

var dup23 = lookup({
	dest: "nwparser.protocol",
	map: map_getProtocolName,
	key: dup15,
});

var hdr1 = match("HEADER#0:0001", "message", "date=%{hdate->} time=%{htime->} devname=%{hdevice->} device_id=%{hfld1->} log_id=%{id->} type=%{hfld2->} subtype=%{hfld3->} pri=%{hseverity->} %{payload}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("_fortinetmgr"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "logver=%{hfld1->} date=%{hdate->} time=%{htime->} log_id=%{id->} %{payload}", processor_chain([
	setc("header_id","0002"),
	dup1,
]));

var hdr3 = match("HEADER#2:0003", "message", "date=%{hdate->} time=%{htime->} logver=%{fld1->} %{payload}", processor_chain([
	setc("header_id","0003"),
	dup1,
]));

var hdr4 = match("HEADER#3:0004", "message", "logver=%{hfld1->} dtime=%{hdatetime->} devid=%{hfld2->} devname=%{hdevice->} %{payload}", processor_chain([
	setc("header_id","0004"),
	dup2,
]));

var hdr5 = match("HEADER#4:0005", "message", "logver=%{hfld1->} devname=\"%{hdevice}\" devid=\"%{hfld2}\" %{payload}", processor_chain([
	setc("header_id","0005"),
	dup2,
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
	hdr5,
]);

var part1 = match("MESSAGE#0:fortinetmgr:01", "nwparser.payload", "user=%{fld1->} adom=%{domain->} user=%{username->} ui=%{fld2->} action=%{action->} status=%{event_state->} msg=\"%{event_description}\"", processor_chain([
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
]));

var msg1 = msg("fortinetmgr:01", part1);

var part2 = match("MESSAGE#1:fortinetmgr", "nwparser.payload", "user=%{username->} adom=%{domain->} msg=\"%{event_description}\"", processor_chain([
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
]));

var msg2 = msg("fortinetmgr", part2);

var part3 = match("MESSAGE#2:fortinetmgr:04/0", "nwparser.payload", "user=\"%{username}\" userfrom=%{fld7->} msg=\"%{p0}");

var part4 = match("MESSAGE#2:fortinetmgr:04/1_0", "nwparser.p0", "User%{p0}");

var part5 = match("MESSAGE#2:fortinetmgr:04/1_1", "nwparser.p0", "user%{p0}");

var select2 = linear_select([
	part4,
	part5,
]);

var part6 = match("MESSAGE#2:fortinetmgr:04/2", "nwparser.p0", "%{}'%{fld3}' with profile '%{fld4}' %{fld5->} from %{fld6}(%{hostip})%{p0}");

var part7 = match("MESSAGE#2:fortinetmgr:04/3_0", "nwparser.p0", ".\"%{p0}");

var part8 = match("MESSAGE#2:fortinetmgr:04/3_1", "nwparser.p0", "\"%{p0}");

var select3 = linear_select([
	part7,
	part8,
]);

var part9 = match("MESSAGE#2:fortinetmgr:04/4", "nwparser.p0", "%{}adminprof=%{p0}");

var part10 = match("MESSAGE#2:fortinetmgr:04/5_0", "nwparser.p0", "%{fld2->} sid=%{sid->} user_type=\"%{profile}\"");

var part11 = match_copy("MESSAGE#2:fortinetmgr:04/5_1", "nwparser.p0", "fld2");

var select4 = linear_select([
	part10,
	part11,
]);

var all1 = all_match({
	processors: [
		part3,
		select2,
		part6,
		select3,
		part9,
		select4,
	],
	on_success: processor_chain([
		dup11,
		dup4,
		lookup({
			dest: "nwparser.event_cat",
			map: map_getEventLegacyCategory,
			key: field("fld5"),
		}),
		dup22,
		dup5,
		dup6,
		dup7,
		dup8,
		dup9,
		dup10,
	]),
});

var msg3 = msg("fortinetmgr:04", all1);

var part12 = match("MESSAGE#3:fortinetmgr:02", "nwparser.payload", "user=%{username->} userfrom=%{fld4->} msg=\"%{event_description}\" adminprof=%{fld2}", processor_chain([
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
]));

var msg4 = msg("fortinetmgr:02", part12);

var part13 = match("MESSAGE#4:fortinetmgr:03", "nwparser.payload", "user=\"%{username}\" msg=\"Login from ssh:%{fld1->} for %{fld2->} from %{saddr->} port %{sport}\" remote_ip=\"%{daddr}\" remote_port=%{dport->} valid=%{fld3->} authmsg=\"%{result}\" extrainfo=%{fld5}", processor_chain([
	dup11,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	lookup({
		dest: "nwparser.event_cat",
		map: map_getEventLegacyCategory,
		key: field("result"),
	}),
	dup22,
]));

var msg5 = msg("fortinetmgr:03", part13);

var part14 = match("MESSAGE#5:fortinetmgr:05/0", "nwparser.payload", "user=\"%{username}\" userfrom=\"%{fld1}\"msg=\"%{p0}");

var part15 = match("MESSAGE#5:fortinetmgr:05/1_0", "nwparser.p0", "dev=%{fld2},vdom=%{fld3},type=%{fld4},key=%{fld5},act=%{action},pkgname=%{fld7},allowaccess=%{fld8}\"%{p0}");

var part16 = match("MESSAGE#5:fortinetmgr:05/1_1", "nwparser.p0", "%{event_description}\"%{p0}");

var select5 = linear_select([
	part15,
	part16,
]);

var part17 = match("MESSAGE#5:fortinetmgr:05/2", "nwparser.p0", "%{domain}\" adom=\"");

var all2 = all_match({
	processors: [
		part14,
		select5,
		part17,
	],
	on_success: processor_chain([
		dup13,
		dup4,
		dup5,
		dup6,
		dup7,
		dup8,
		dup9,
		dup10,
	]),
});

var msg6 = msg("fortinetmgr:05", all2);

var part18 = tagval("MESSAGE#6:event_fortinetmgr_tvm", "nwparser.payload", tvm, {
	"action": "action",
	"adom": "domain",
	"desc": "event_description",
	"msg": "info",
	"session_id": "sessionid",
	"user": "username",
	"userfrom": "fld1",
}, processor_chain([
	dup11,
	dup4,
	dup5,
	dup6,
	dup7,
	setf("event_type","hfld2"),
	dup9,
	dup10,
]));

var msg7 = msg("event_fortinetmgr_tvm", part18);

var select6 = linear_select([
	msg1,
	msg2,
	msg3,
	msg4,
	msg5,
	msg6,
	msg7,
]);

var part19 = tagval("MESSAGE#7:generic_fortinetmgr", "nwparser.payload", tvm, {
	"action": "action",
	"adminprof": "fld13",
	"cat": "fcatnum",
	"catdesc": "filter",
	"cipher_suite": "fld24",
	"content_switch_name": "fld15",
	"craction": "fld9",
	"crlevel": "fld10",
	"crscore": "reputation_num",
	"dev_id": "fld100",
	"device_id": "hardware_id",
	"devid": "hardware_id",
	"devname": "event_source",
	"devtype": "fld7",
	"direction": "direction",
	"dst": "daddr",
	"dst_port": "dport",
	"dstintf": "dinterface",
	"dstip": "daddr",
	"dstport": "dport",
	"duration": "duration",
	"eventtype": "vendor_event_cat",
	"false_positive_mitigation": "fld17",
	"ftp_cmd": "fld23",
	"ftp_mode": "fld22",
	"history_threat_weight": "fld21",
	"hostname": "hostname",
	"http_agent": "agent",
	"http_host": "web_ref_domain",
	"http_method": "web_method",
	"http_refer": "web_referer",
	"http_session_id": "sessionid",
	"http_url": "web_query",
	"http_version": "fld19",
	"level": "severity",
	"log_id": "id",
	"logid": "id",
	"main_type": "fld37",
	"mastersrcmac": "fld8",
	"method": "fld12",
	"monitor_status": "fld18",
	"msg": "event_description",
	"msg_id": "fld25",
	"osname": "os",
	"osversion": "version",
	"policy": "policyname",
	"policyid": "policy_id",
	"poluuid": "fld5",
	"pri": "severity",
	"profile": "rulename",
	"proto": "fld6",
	"rcvdbyte": "rbytes",
	"reqtype": "fld11",
	"sentbyte": "sbytes",
	"server_pool_name": "fld16",
	"service": "network_service",
	"sessionid": "sessionid",
	"severity_level": "fld101",
	"signature_id": "sigid",
	"signature_subclass": "fld14",
	"src": "saddr",
	"src_port": "sport",
	"srccountry": "location_src",
	"srcintf": "sinterface",
	"srcip": "saddr",
	"srcmac": "smacaddr",
	"srcport": "sport",
	"sub_type": "category",
	"subtype": "category",
	"threat_level": "threat_val",
	"threat_weight": "fld20",
	"timezone": "timezone",
	"trandisp": "context",
	"trigger_policy": "fld39",
	"type": "event_type",
	"url": "url",
	"user": "username",
	"user_name": "username",
	"userfrom": "fld30",
	"vd": "vsys",
}, processor_chain([
	dup13,
	dup4,
	dup5,
	dup14,
	dup23,
]));

var msg8 = msg("generic_fortinetmgr", part19);

var part20 = tagval("MESSAGE#8:generic_fortinetmgr_1", "nwparser.payload", tvm, {
	"action": "action",
	"app": "obj_name",
	"appcat": "fld33",
	"craction": "fld9",
	"crlevel": "fld10",
	"crscore": "reputation_num",
	"date": "fld1",
	"dstcountry": "location_dst",
	"dstintf": "dinterface",
	"dstintfrole": "fld31",
	"dstip": "daddr",
	"dstport": "dport",
	"duration": "duration",
	"eventtime": "event_time_string",
	"level": "severity",
	"logid": "id",
	"logtime": "fld35",
	"policyid": "policy_id",
	"policytype": "fld34",
	"poluuid": "fld5",
	"proto": "fld6",
	"rcvdbyte": "rbytes",
	"sentbyte": "sbytes",
	"sentpkt": "fld15",
	"service": "network_service",
	"sessionid": "sessionid",
	"srccountry": "location_src",
	"srcintf": "sinterface",
	"srcintfrole": "fld30",
	"srcip": "saddr",
	"srcport": "sport",
	"subtype": "category",
	"time": "fld2",
	"trandisp": "context",
	"tranip": "dtransaddr",
	"tranport": "dtransport",
	"type": "event_type",
	"vd": "vsys",
}, processor_chain([
	dup13,
	dup4,
	date_time({
		dest: "event_time",
		args: ["fld1","fld2"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
		],
	}),
	dup6,
	setf("hardware_id","hfld2"),
	dup14,
	dup23,
]));

var msg9 = msg("generic_fortinetmgr_1", part20);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"event_fortinetmgr": select6,
		"generic_fortinetmgr": msg8,
		"generic_fortinetmgr_1": msg9,
	}),
]);
