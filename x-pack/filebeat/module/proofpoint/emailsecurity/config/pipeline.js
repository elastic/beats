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

var dup1 = // "Pattern{Constant('info'), Field(p0,false)}"
match("HEADER#0:0024/1_0", "nwparser.p0", "info%{p0}");

var dup2 = // "Pattern{Constant('rprt'), Field(p0,false)}"
match("HEADER#0:0024/1_1", "nwparser.p0", "rprt%{p0}");

var dup3 = // "Pattern{Constant('warn'), Field(p0,false)}"
match("HEADER#0:0024/1_2", "nwparser.p0", "warn%{p0}");

var dup4 = // "Pattern{Constant('err'), Field(p0,false)}"
match("HEADER#0:0024/1_3", "nwparser.p0", "err%{p0}");

var dup5 = // "Pattern{Constant('note'), Field(p0,false)}"
match("HEADER#0:0024/1_4", "nwparser.p0", "note%{p0}");

var dup6 = call({
	dest: "nwparser.messageid",
	fn: STRCAT,
	args: [
		field("msgIdPart1"),
		constant("_"),
		field("msgIdPart2"),
	],
});

var dup7 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hinstance"),
		constant("["),
		field("hfld2"),
		constant("]: "),
		field("severity"),
		constant(" mod="),
		field("msgIdPart1"),
		constant(" "),
		field("p0"),
	],
});

var dup8 = setc("eventcategory","1207010000");

var dup9 = setf("msg","$MSG");

var dup10 = setc("eventcategory","1207020100");

var dup11 = setc("eventcategory","1207020000");

var dup12 = setc("dclass_counter1_string","No of attachments:");

var dup13 = setc("dclass_counter2_string","No of recipients:");

var dup14 = // "Pattern{Field(hostip,true), Constant(' sampling='), Field(fld19,false)}"
match("MESSAGE#11:mail_env_from:ofrom/1_0", "nwparser.p0", "%{hostip->} sampling=%{fld19}");

var dup15 = // "Pattern{Field(hostip,false)}"
match_copy("MESSAGE#11:mail_env_from:ofrom/1_1", "nwparser.p0", "hostip");

var dup16 = setc("eventcategory","1207030000");

var dup17 = setc("eventcategory","1207000000");

var dup18 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#25:session_judge/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} %{p0}");

var dup19 = // "Pattern{Constant('attachment='), Field(fld58,true), Constant(' file='), Field(fld1,true), Constant(' mod='), Field(p0,false)}"
match("MESSAGE#25:session_judge/1_0", "nwparser.p0", "attachment=%{fld58->} file=%{fld1->} mod=%{p0}");

var dup20 = // "Pattern{Constant('mod='), Field(p0,false)}"
match("MESSAGE#25:session_judge/1_1", "nwparser.p0", "mod=%{p0}");

var dup21 = call({
	dest: "nwparser.filename",
	fn: RMQ,
	args: [
		field("fld1"),
	],
});

var dup22 = setc("eventcategory","1207040200");

var dup23 = // "Pattern{Constant('vendor='), Field(fld36,true), Constant(' version="'), Field(component_version,false), Constant('" duration='), Field(p0,false)}"
match("MESSAGE#39:av_run:02/1_1", "nwparser.p0", "vendor=%{fld36->} version=\"%{component_version}\" duration=%{p0}");

var dup24 = // "Pattern{Field(duration_string,false)}"
match_copy("MESSAGE#39:av_run:02/2", "nwparser.p0", "duration_string");

var dup25 = setc("eventcategory","1003010000");

var dup26 = setc("eventcategory","1003000000");

var dup27 = setc("eventcategory","1207040000");

var dup28 = // "Pattern{Constant('['), Field(daddr,false), Constant('] ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_0", "nwparser.p0", "[%{daddr}] [%{daddr}],%{p0}");

var dup29 = // "Pattern{Constant('['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_1", "nwparser.p0", "[%{daddr}],%{p0}");

var dup30 = // "Pattern{Field(dhost,true), Constant(' ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_2", "nwparser.p0", "%{dhost->} [%{daddr}],%{p0}");

var dup31 = // "Pattern{Field(dhost,false), Constant(','), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_3", "nwparser.p0", "%{dhost},%{p0}");

var dup32 = // "Pattern{Field(,false), Constant('dsn='), Field(resultcode,false), Constant(', stat='), Field(info,false)}"
match("MESSAGE#98:queued-alert/4", "nwparser.p0", "%{}dsn=%{resultcode}, stat=%{info}");

var dup33 = // "Pattern{Constant('['), Field(daddr,false), Constant(']')}"
match("MESSAGE#99:queued-alert:01/1_1", "nwparser.p0", "[%{daddr}]");

var dup34 = // "Pattern{Field(dhost,true), Constant(' ['), Field(daddr,false), Constant(']')}"
match("MESSAGE#99:queued-alert:01/1_2", "nwparser.p0", "%{dhost->} [%{daddr}]");

var dup35 = // "Pattern{Field(dhost,false)}"
match_copy("MESSAGE#99:queued-alert:01/1_3", "nwparser.p0", "dhost");

var dup36 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup37 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: STARTTLS='), Field(fld1,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#100:queued-alert:02/0", "nwparser.payload", "%{agent}[%{process_id}]: STARTTLS=%{fld1}, relay=%{p0}");

var dup38 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#101:queued-VoltageEncrypt/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, pri=%{fld23}, relay=%{p0}");

var dup39 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld57,false), Constant(', nrcpts='), Field(fld58,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(fld69,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#120:queued-VoltageEncrypt:01/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld57}, nrcpts=%{fld58}, msgid=%{id}, proto=%{protocol}, daemon=%{fld69}, relay=%{p0}");

var dup40 = // "Pattern{Constant('['), Field(daddr,false), Constant('] ['), Field(daddr,false), Constant(']')}"
match("MESSAGE#120:queued-VoltageEncrypt:01/1_0", "nwparser.p0", "[%{daddr}] [%{daddr}]");

var dup41 = // "Pattern{Field(,false), Constant('field='), Field(fld2,false), Constant(', status='), Field(info,false)}"
match("MESSAGE#104:queued-default:02/2", "nwparser.p0", "%{}field=%{fld2}, status=%{info}");

var dup42 = // "Pattern{Field(,false), Constant('version='), Field(fld55,false), Constant(', verify='), Field(fld57,false), Constant(', cipher='), Field(fld58,false), Constant(', bits='), Field(fld59,false)}"
match("MESSAGE#105:queued-default:03/2", "nwparser.p0", "%{}version=%{fld55}, verify=%{fld57}, cipher=%{fld58}, bits=%{fld59}");

var dup43 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld57,false), Constant(', nrcpts='), Field(fld58,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(fld69,false), Constant(', tls_verify='), Field(fld70,false), Constant(', auth='), Field(fld71,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#116:queued-eurort:02/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld57}, nrcpts=%{fld58}, msgid=%{id}, proto=%{protocol}, daemon=%{fld69}, tls_verify=%{fld70}, auth=%{fld71}, relay=%{p0}");

var dup44 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#126:sendmail/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, pri=%{fld23}, relay=%{p0}");

var dup45 = linear_select([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]);

var dup46 = linear_select([
	dup14,
	dup15,
]);

var dup47 = linear_select([
	dup19,
	dup20,
]);

var dup48 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' vendor='), Field(fld36,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' signatures='), Field(fld94,false)}"
match("MESSAGE#43:av_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} vendor=%{fld36->} engine=%{fld49->} definitions=%{fld50->} signatures=%{fld94}", processor_chain([
	dup26,
	dup9,
]));

var dup49 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#48:access_run:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var dup50 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#49:access_run:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var dup51 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' action='), Field(action,true), Constant(' dict='), Field(fld37,true), Constant(' file='), Field(filename,false)}"
match("MESSAGE#51:access_refresh:01", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} action=%{action->} dict=%{fld37->} file=%{filename}", processor_chain([
	dup17,
	dup9,
]));

var dup52 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,false)}"
match("MESSAGE#52:access_load", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5}", processor_chain([
	dup17,
	dup9,
]));

var dup53 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,false)}"
match("MESSAGE#64:spam_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} engine=%{fld49->} definitions=%{fld50}", processor_chain([
	dup27,
	dup9,
]));

var dup54 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' version='), Field(fld55,false)}"
match("MESSAGE#71:zerohour_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} version=%{fld55}", processor_chain([
	dup17,
	dup9,
]));

var dup55 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' sig='), Field(fld60,false)}"
match("MESSAGE#82:cvtd:01", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} sig=%{fld60}", processor_chain([
	dup17,
	dup9,
]));

var dup56 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,false)}"
match("MESSAGE#83:cvtd", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type}", processor_chain([
	dup17,
	dup9,
]));

var dup57 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' addr='), Field(saddr,false)}"
match("MESSAGE#87:soap_listen", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} addr=%{saddr}", processor_chain([
	dup17,
	dup9,
]));

var dup58 = linear_select([
	dup28,
	dup29,
	dup30,
	dup31,
]);

var dup59 = linear_select([
	dup40,
	dup33,
	dup34,
	dup35,
]);

var dup60 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': timeout waiting for input from '), Field(fld11,true), Constant(' during server cmd read')}"
match("MESSAGE#106:queued-default:04", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: timeout waiting for input from %{fld11->} during server cmd read", processor_chain([
	dup17,
	dup9,
]));

var dup61 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(event_description,false)}"
match("MESSAGE#113:queued-reinject:06", "nwparser.payload", "%{agent}[%{process_id}]: %{event_description}", processor_chain([
	dup17,
	dup9,
]));

var dup62 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' pid='), Field(process_id,true), Constant(' '), Field(web_method,true), Constant(' /'), Field(info,false), Constant(': '), Field(resultcode,false)}"
match("MESSAGE#141:info:pid", "nwparser.payload", "%{fld0->} %{severity->} pid=%{process_id->} %{web_method->} /%{info}: %{resultcode}", processor_chain([
	dup17,
	dup9,
]));

var dup63 = all_match({
	processors: [
		dup38,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var dup64 = all_match({
	processors: [
		dup39,
		dup59,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var dup65 = all_match({
	processors: [
		dup37,
		dup58,
		dup41,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var dup66 = all_match({
	processors: [
		dup37,
		dup58,
		dup42,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var dup67 = all_match({
	processors: [
		dup43,
		dup59,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var dup68 = all_match({
	processors: [
		dup44,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var hdr1 = // "Pattern{Field(hdate,false), Constant('T'), Field(htime,false), Constant('.'), Field(hfld1,true), Constant(' '), Field(hfld2,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld3,false), Constant(']: '), Field(p0,false)}"
match("HEADER#0:0024/0", "message", "%{hdate}T%{htime}.%{hfld1->} %{hfld2->} %{hinstance}[%{hfld3}]: %{p0}", processor_chain([
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld3"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var part1 = // "Pattern{Field(,false), Constant('s='), Field(hfld4,true), Constant(' cmd=send '), Field(p0,false)}"
match("HEADER#0:0024/2", "nwparser.p0", "%{}s=%{hfld4->} cmd=send %{p0}");

var all1 = all_match({
	processors: [
		hdr1,
		dup45,
		part1,
	],
	on_success: processor_chain([
		setc("header_id","0024"),
		setc("messageid","send"),
	]),
});

var hdr2 = // "Pattern{Field(hdate,false), Constant('T'), Field(htime,false), Constant('.'), Field(hfld1,true), Constant(' '), Field(hfld2,true), Constant(' '), Field(messageid,false), Constant('['), Field(hfld3,false), Constant(']: '), Field(p0,false)}"
match("HEADER#1:0023/0", "message", "%{hdate}T%{htime}.%{hfld1->} %{hfld2->} %{messageid}[%{hfld3}]: %{p0}");

var part2 = // "Pattern{Field(,true), Constant(' '), Field(payload,false)}"
match("HEADER#1:0023/2", "nwparser.p0", "%{} %{payload}");

var all2 = all_match({
	processors: [
		hdr2,
		dup45,
		part2,
	],
	on_success: processor_chain([
		setc("header_id","0023"),
	]),
});

var hdr3 = // "Pattern{Field(hdate,false), Constant('T'), Field(htime,false), Constant('.'), Field(hfld1,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(messageid,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(p0,false)}"
match("HEADER#2:0025", "message", "%{hdate}T%{htime}.%{hfld1->} %{hinstance->} %{messageid}[%{hfld2}]: %{p0}", processor_chain([
	setc("header_id","0025"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("["),
			field("hfld2"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr4 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hostname,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld4,false), Constant(']: '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' m='), Field(hfld2,true), Constant(' x='), Field(hfld3,true), Constant(' attachment='), Field(hfld7,true), Constant(' file='), Field(hfld5,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#3:0026", "message", "%{hmonth->} %{hday->} %{htime->} %{hostname->} %{hinstance}[%{hfld4}]: %{hseverity->} s=%{hfld1->} m=%{hfld2->} x=%{hfld3->} attachment=%{hfld7->} file=%{hfld5->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0026"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld4"),
			constant("]: "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" m="),
			field("hfld2"),
			constant(" x="),
			field("hfld3"),
			constant(" attachment="),
			field("hfld7"),
			constant(" file="),
			field("hfld5"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr5 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' m='), Field(hfld2,true), Constant(' x='), Field(hfld3,true), Constant(' attachment='), Field(hfld7,true), Constant(' file='), Field(hfld5,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#4:0003", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} m=%{hfld2->} x=%{hfld3->} attachment=%{hfld7->} file=%{hfld5->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0003"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" m="),
			field("hfld2"),
			constant(" x="),
			field("hfld3"),
			constant(" attachment="),
			field("hfld7"),
			constant(" file="),
			field("hfld5"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr6 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(hseverity,true), Constant(' s='), Field(hfld3,true), Constant(' m='), Field(hfld4,true), Constant(' x='), Field(hfld5,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#5:0015", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}[%{hfld2}]: %{hseverity->} s=%{hfld3->} m=%{hfld4->} x=%{hfld5->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0015"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld2"),
			constant("]: "),
			field("hseverity"),
			constant(" s="),
			field("hfld3"),
			constant(" m="),
			field("hfld4"),
			constant(" x="),
			field("hfld5"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr7 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(hseverity,true), Constant(' s='), Field(hfld3,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#6:0016", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}[%{hfld2}]: %{hseverity->} s=%{hfld3->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0016"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld2"),
			constant("]: "),
			field("hseverity"),
			constant(" s="),
			field("hfld3"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr8 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(severity,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' '), Field(p0,false)}"
match("HEADER#7:0017", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}[%{hfld2}]: %{severity->} mod=%{msgIdPart1->} %{p0}", processor_chain([
	setc("header_id","0017"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("msgIdPart1"),
			constant("_ttl"),
		],
	}),
	dup7,
]));

var hdr9 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant(': '), Field(hseverity,true), Constant(' s='), Field(hfld2,true), Constant(' m='), Field(hfld3,true), Constant(' x='), Field(hfld4,true), Constant(' cmd='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#8:0018", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}: %{hseverity->} s=%{hfld2->} m=%{hfld3->} x=%{hfld4->} cmd=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0018"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(": "),
			field("hseverity"),
			constant(" s="),
			field("hfld2"),
			constant(" m="),
			field("hfld3"),
			constant(" x="),
			field("hfld4"),
			constant(" cmd="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr10 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld2,true), Constant(' mod='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#9:0019", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance->} %{hseverity->} s=%{hfld2->} mod=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0019"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld2"),
			constant(" mod="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr11 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(hseverity,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' '), Field(msgIdPart2,false), Constant('='), Field(hfld3,true), Constant(' '), Field(p0,false)}"
match("HEADER#10:0020", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}[%{hfld2}]: %{hseverity->} mod=%{msgIdPart1->} %{msgIdPart2}=%{hfld3->} %{p0}", processor_chain([
	setc("header_id","0020"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld2"),
			constant("]: "),
			field("hseverity"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" "),
			field("msgIdPart2"),
			constant("="),
			field("hfld3"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr12 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld2,false), Constant(']: '), Field(severity,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' '), Field(p0,false)}"
match("HEADER#11:0021", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}[%{hfld2}]: %{severity->} mod=%{msgIdPart1->} %{p0}", processor_chain([
	setc("header_id","0021"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("msgIdPart1"),
			constant("_type"),
		],
	}),
	dup7,
]));

var hdr13 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld1,true), Constant(' '), Field(hinstance,false), Constant(': '), Field(hseverity,true), Constant(' s='), Field(hfld2,true), Constant(' m='), Field(hfld3,true), Constant(' x='), Field(hfld4,true), Constant(' '), Field(msgIdPart1,false), Constant('='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#12:0022", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1->} %{hinstance}: %{hseverity->} s=%{hfld2->} m=%{hfld3->} x=%{hfld4->} %{msgIdPart1}=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0022"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(": "),
			field("hseverity"),
			constant(" s="),
			field("hfld2"),
			constant(" m="),
			field("hfld3"),
			constant(" x="),
			field("hfld4"),
			constant(" "),
			field("msgIdPart1"),
			constant("="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr14 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' m='), Field(hfld2,true), Constant(' x='), Field(hfld3,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#13:0001", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} m=%{hfld2->} x=%{hfld3->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0001"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" m="),
			field("hfld2"),
			constant(" x="),
			field("hfld3"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr15 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' m='), Field(hfld2,true), Constant(' x='), Field(hfld3,true), Constant(' cmd='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#14:0008", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} m=%{hfld2->} x=%{hfld3->} cmd=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0008"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" m="),
			field("hfld2"),
			constant(" x="),
			field("hfld3"),
			constant(" cmd="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr16 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#15:0002", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0002"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr17 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' mod='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#16:0007", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} mod=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0007"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" mod="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr18 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' s='), Field(hfld1,true), Constant(' cmd='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#17:0012", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} s=%{hfld1->} cmd=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0012"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" s="),
			field("hfld1"),
			constant(" cmd="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr19 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' type='), Field(hfld5,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#18:0004", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} mod=%{msgIdPart1->} type=%{hfld5->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0004"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" type="),
			field("hfld5"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr20 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' pid='), Field(hfld5,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#19:0005", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} pid=%{hfld5->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0005"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" pid="),
			field("hfld5"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr21 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' mod='), Field(msgIdPart1,true), Constant(' cmd='), Field(msgIdPart2,true), Constant(' '), Field(p0,false)}"
match("HEADER#20:0006", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} mod=%{msgIdPart1->} cmd=%{msgIdPart2->} %{p0}", processor_chain([
	setc("header_id","0006"),
	dup6,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" mod="),
			field("msgIdPart1"),
			constant(" cmd="),
			field("msgIdPart2"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr22 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(hseverity,true), Constant(' mod='), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#21:0009", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{hseverity->} mod=%{messageid->} %{p0}", processor_chain([
	setc("header_id","0009"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("hseverity"),
			constant(" mod="),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr23 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hfld2,true), Constant(' '), Field(hinstance,false), Constant('['), Field(hfld1,false), Constant(']: '), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#22:0014", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld2->} %{hinstance}[%{hfld1}]: %{messageid->} %{p0}", processor_chain([
	setc("header_id","0014"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant("["),
			field("hfld1"),
			constant("]: "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr24 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(messageid,false), Constant('['), Field(hfld1,false), Constant(']: '), Field(p0,false)}"
match("HEADER#23:0013", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{messageid}[%{hfld1}]: %{p0}", processor_chain([
	setc("header_id","0013"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("messageid"),
			constant("["),
			field("hfld1"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr25 = // "Pattern{Field(hmonth,true), Constant(' '), Field(hday,true), Constant(' '), Field(htime,true), Constant(' '), Field(hinstance,true), Constant(' '), Field(messageid,true), Constant(' '), Field(p0,false)}"
match("HEADER#24:0011", "message", "%{hmonth->} %{hday->} %{htime->} %{hinstance->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0011"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hinstance"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr26 = // "Pattern{Field(messageid,false), Constant('['), Field(hfld1,false), Constant(']: '), Field(p0,false)}"
match("HEADER#25:0010", "message", "%{messageid}[%{hfld1}]: %{p0}", processor_chain([
	setc("header_id","0010"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("["),
			field("hfld1"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var select1 = linear_select([
	all1,
	all2,
	hdr3,
	hdr4,
	hdr5,
	hdr6,
	hdr7,
	hdr8,
	hdr9,
	hdr10,
	hdr11,
	hdr12,
	hdr13,
	hdr14,
	hdr15,
	hdr16,
	hdr17,
	hdr18,
	hdr19,
	hdr20,
	hdr21,
	hdr22,
	hdr23,
	hdr24,
	hdr25,
	hdr26,
]);

var part3 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' r='), Field(event_counter,true), Constant(' value='), Field(to,true), Constant(' verified='), Field(fld3,true), Constant(' routes='), Field(fld4,false)}"
match("MESSAGE#0:mail_env_rcpt", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} r=%{event_counter->} value=%{to->} verified=%{fld3->} routes=%{fld4}", processor_chain([
	dup8,
	dup9,
]));

var msg1 = msg("mail_env_rcpt", part3);

var part4 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' r='), Field(event_counter,true), Constant(' value='), Field(to,true), Constant(' verified='), Field(fld3,true), Constant(' routes='), Field(fld4,false)}"
match("MESSAGE#1:mail_env_rcpt:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} r=%{event_counter->} value=%{to->} verified=%{fld3->} routes=%{fld4}", processor_chain([
	dup8,
	dup9,
]));

var msg2 = msg("mail_env_rcpt:01", part4);

var select2 = linear_select([
	msg1,
	msg2,
]);

var part5 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' file='), Field(filename,true), Constant(' mime='), Field(content_type,true), Constant(' type='), Field(fld6,true), Constant(' omime='), Field(fld7,true), Constant(' oext='), Field(fld8,true), Constant(' corrupted='), Field(fld9,true), Constant(' protected='), Field(fld10,true), Constant(' size='), Field(bytes,true), Constant(' virtual='), Field(fld11,true), Constant(' a='), Field(fld12,false)}"
match("MESSAGE#2:mail_attachment", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} id=%{fld5->} file=%{filename->} mime=%{content_type->} type=%{fld6->} omime=%{fld7->} oext=%{fld8->} corrupted=%{fld9->} protected=%{fld10->} size=%{bytes->} virtual=%{fld11->} a=%{fld12}", processor_chain([
	dup10,
	dup9,
]));

var msg3 = msg("mail_attachment", part5);

var part6 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' file='), Field(filename,true), Constant(' mime='), Field(content_type,true), Constant(' type='), Field(fld6,true), Constant(' omime='), Field(fld7,true), Constant(' oext='), Field(fld8,true), Constant(' corrupted='), Field(fld9,true), Constant(' protected='), Field(fld10,true), Constant(' size='), Field(bytes,true), Constant(' virtual='), Field(fld11,true), Constant(' a='), Field(fld12,false)}"
match("MESSAGE#3:mail_attachment:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} id=%{fld5->} file=%{filename->} mime=%{content_type->} type=%{fld6->} omime=%{fld7->} oext=%{fld8->} corrupted=%{fld9->} protected=%{fld10->} size=%{bytes->} virtual=%{fld11->} a=%{fld12}", processor_chain([
	dup10,
	dup9,
]));

var msg4 = msg("mail_attachment:01", part6);

var part7 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' file='), Field(filename,true), Constant(' mime='), Field(content_type,true), Constant(' type='), Field(fld6,true), Constant(' omime='), Field(fld7,true), Constant(' oext='), Field(fld8,true), Constant(' corrupted='), Field(fld9,true), Constant(' protected='), Field(fld10,true), Constant(' size='), Field(bytes,true), Constant(' virtual='), Field(fld11,false)}"
match("MESSAGE#4:mail_attachment:02", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} id=%{fld5->} file=%{filename->} mime=%{content_type->} type=%{fld6->} omime=%{fld7->} oext=%{fld8->} corrupted=%{fld9->} protected=%{fld10->} size=%{bytes->} virtual=%{fld11}", processor_chain([
	dup10,
	dup9,
]));

var msg5 = msg("mail_attachment:02", part7);

var part8 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' file='), Field(filename,true), Constant(' mime='), Field(content_type,true), Constant(' type='), Field(fld6,true), Constant(' omime='), Field(fld7,true), Constant(' oext='), Field(fld8,true), Constant(' corrupted='), Field(fld9,true), Constant(' protected='), Field(fld10,true), Constant(' size='), Field(bytes,true), Constant(' virtual='), Field(fld11,false)}"
match("MESSAGE#5:mail_attachment:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} id=%{fld5->} file=%{filename->} mime=%{content_type->} type=%{fld6->} omime=%{fld7->} oext=%{fld8->} corrupted=%{fld9->} protected=%{fld10->} size=%{bytes->} virtual=%{fld11}", processor_chain([
	dup10,
	dup9,
]));

var msg6 = msg("mail_attachment:03", part8);

var select3 = linear_select([
	msg3,
	msg4,
	msg5,
	msg6,
]);

var part9 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject='), Field(subject,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' virusname='), Field(threat_name,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#6:mail_msg", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=%{subject->} spamscore=%{reputation_num->} virusname=%{threat_name->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup11,
	dup9,
	dup12,
	dup13,
]));

var msg7 = msg("mail_msg", part9);

var part10 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject='), Field(subject,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' virusname='), Field(threat_name,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#7:mail_msg:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=%{subject->} spamscore=%{reputation_num->} virusname=%{threat_name->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup11,
	dup9,
	dup12,
	dup13,
]));

var msg8 = msg("mail_msg:01", part10);

var part11 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject='), Field(subject,true), Constant(' virusname='), Field(threat_name,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#8:mail_msg:04", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=%{subject->} virusname=%{threat_name->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup11,
	dup9,
	dup12,
	dup13,
]));

var msg9 = msg("mail_msg:04", part11);

var part12 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject='), Field(subject,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#9:mail_msg:02", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=%{subject->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup11,
	dup9,
	dup12,
	dup13,
]));

var msg10 = msg("mail_msg:02", part12);

var part13 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject='), Field(subject,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#10:mail_msg:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=%{subject->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup11,
	dup9,
	dup12,
	dup13,
]));

var msg11 = msg("mail_msg:03", part13);

var select4 = linear_select([
	msg7,
	msg8,
	msg9,
	msg10,
	msg11,
]);

var part14 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(to,true), Constant(' ofrom='), Field(from,true), Constant(' qid='), Field(fld15,true), Constant(' tls='), Field(fld17,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' host='), Field(hostname,true), Constant(' ip='), Field(p0,false)}"
match("MESSAGE#11:mail_env_from:ofrom/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} value=%{to->} ofrom=%{from->} qid=%{fld15->} tls=%{fld17->} routes=%{fld4->} notroutes=%{fld18->} host=%{hostname->} ip=%{p0}");

var all3 = all_match({
	processors: [
		part14,
		dup46,
	],
	on_success: processor_chain([
		dup16,
		dup9,
	]),
});

var msg12 = msg("mail_env_from:ofrom", all3);

var part15 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(to,true), Constant(' ofrom='), Field(from,true), Constant(' qid='), Field(fld15,true), Constant(' tls='), Field(fld17,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' host='), Field(hostname,true), Constant(' ip='), Field(hostip,true), Constant(' sampling='), Field(fld19,false)}"
match("MESSAGE#12:mail_env_from:ofrom:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} value=%{to->} ofrom=%{from->} qid=%{fld15->} tls=%{fld17->} routes=%{fld4->} notroutes=%{fld18->} host=%{hostname->} ip=%{hostip->} sampling=%{fld19}", processor_chain([
	dup16,
	dup9,
]));

var msg13 = msg("mail_env_from:ofrom:01", part15);

var part16 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(from,true), Constant(' qid='), Field(fld15,true), Constant(' tls='), Field(fld17,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' host='), Field(hostname,true), Constant(' ip='), Field(p0,false)}"
match("MESSAGE#13:mail_env_from/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} value=%{from->} qid=%{fld15->} tls=%{fld17->} routes=%{fld4->} notroutes=%{fld18->} host=%{hostname->} ip=%{p0}");

var all4 = all_match({
	processors: [
		part16,
		dup46,
	],
	on_success: processor_chain([
		dup16,
		dup9,
	]),
});

var msg14 = msg("mail_env_from", all4);

var part17 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(from,true), Constant(' qid='), Field(fld15,true), Constant(' tls='), Field(fld17,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' host='), Field(hostname,true), Constant(' ip='), Field(hostip,true), Constant(' sampling='), Field(fld19,false)}"
match("MESSAGE#14:mail_env_from:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} value=%{from->} qid=%{fld15->} tls=%{fld17->} routes=%{fld4->} notroutes=%{fld18->} host=%{hostname->} ip=%{hostip->} sampling=%{fld19}", processor_chain([
	dup16,
	dup9,
]));

var msg15 = msg("mail_env_from:01", part17);

var select5 = linear_select([
	msg12,
	msg13,
	msg14,
	msg15,
]);

var part18 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(ddomain,true), Constant(' routes='), Field(fld4,false)}"
match("MESSAGE#15:mail_helo", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} value=%{ddomain->} routes=%{fld4}", processor_chain([
	dup17,
	dup9,
]));

var msg16 = msg("mail_helo", part18);

var part19 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' value='), Field(ddomain,true), Constant(' routes='), Field(fld4,false)}"
match("MESSAGE#16:mail_helo:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} value=%{ddomain->} routes=%{fld4}", processor_chain([
	dup17,
	dup9,
]));

var msg17 = msg("mail_helo:01", part19);

var select6 = linear_select([
	msg16,
	msg17,
]);

var part20 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' action='), Field(action,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#17:mail_continue-system-sendmail", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} action=%{action->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg18 = msg("mail_continue-system-sendmail", part20);

var part21 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' status='), Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#18:mail_release", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} status=%{result->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg19 = msg("mail_release", part21);

var part22 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#19:session_data/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} %{p0}");

var part23 = // "Pattern{Constant('rcpt_notroutes='), Field(fld20,true), Constant(' data_routes='), Field(fld21,false)}"
match("MESSAGE#19:session_data/1_0", "nwparser.p0", "rcpt_notroutes=%{fld20->} data_routes=%{fld21}");

var part24 = // "Pattern{Constant('rcpt='), Field(to,true), Constant(' suborg='), Field(fld22,false)}"
match("MESSAGE#19:session_data/1_1", "nwparser.p0", "rcpt=%{to->} suborg=%{fld22}");

var part25 = // "Pattern{Constant('from='), Field(from,true), Constant(' suborg='), Field(fld22,false)}"
match("MESSAGE#19:session_data/1_2", "nwparser.p0", "from=%{from->} suborg=%{fld22}");

var select7 = linear_select([
	part23,
	part24,
	part25,
]);

var all5 = all_match({
	processors: [
		part22,
		select7,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg20 = msg("session_data", all5);

var part26 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rcpt_notroutes='), Field(fld20,true), Constant(' data_routes='), Field(fld21,false)}"
match("MESSAGE#20:session_data:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rcpt_notroutes=%{fld20->} data_routes=%{fld21}", processor_chain([
	dup17,
	dup9,
]));

var msg21 = msg("session_data:01", part26);

var select8 = linear_select([
	msg20,
	msg21,
]);

var part27 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' folder='), Field(fld22,true), Constant(' pri='), Field(fld23,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#21:session_store", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} folder=%{fld22->} pri=%{fld23->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg22 = msg("session_store", part27);

var part28 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' folder='), Field(fld22,true), Constant(' pri='), Field(fld23,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#22:session_store:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} folder=%{fld22->} pri=%{fld23->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg23 = msg("session_store:01", part28);

var select9 = linear_select([
	msg22,
	msg23,
]);

var part29 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,false)}"
match("MESSAGE#23:session_headers", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} routes=%{fld4->} notroutes=%{fld18}", processor_chain([
	dup17,
	dup9,
]));

var msg24 = msg("session_headers", part29);

var part30 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,false)}"
match("MESSAGE#24:session_headers:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} routes=%{fld4->} notroutes=%{fld18}", processor_chain([
	dup17,
	dup9,
]));

var msg25 = msg("session_headers:01", part30);

var select10 = linear_select([
	msg24,
	msg25,
]);

var part31 = // "Pattern{Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,false)}"
match("MESSAGE#25:session_judge/2", "nwparser.p0", "%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename}");

var all6 = all_match({
	processors: [
		dup18,
		dup47,
		part31,
	],
	on_success: processor_chain([
		dup17,
		dup9,
		dup21,
	]),
});

var msg26 = msg("session_judge", all6);

var part32 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,false)}"
match("MESSAGE#26:session_judge:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename}", processor_chain([
	dup17,
	dup9,
]));

var msg27 = msg("session_judge:01", part32);

var select11 = linear_select([
	msg26,
	msg27,
]);

var part33 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' ip='), Field(hostip,true), Constant(' country='), Field(location_country,true), Constant(' lip='), Field(fld24,true), Constant(' prot='), Field(fld25,true), Constant(' hops_active='), Field(fld26,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' perlwait='), Field(fld27,false)}"
match("MESSAGE#27:session_connect", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} ip=%{hostip->} country=%{location_country->} lip=%{fld24->} prot=%{fld25->} hops_active=%{fld26->} routes=%{fld4->} notroutes=%{fld18->} perlwait=%{fld27}", processor_chain([
	dup17,
	dup9,
]));

var msg28 = msg("session_connect", part33);

var part34 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' ip='), Field(hostip,true), Constant(' country='), Field(location_country,true), Constant(' lip='), Field(fld24,true), Constant(' prot='), Field(fld25,true), Constant(' hops_active='), Field(fld26,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,true), Constant(' perlwait='), Field(fld27,false)}"
match("MESSAGE#28:session_connect:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} ip=%{hostip->} country=%{location_country->} lip=%{fld24->} prot=%{fld25->} hops_active=%{fld26->} routes=%{fld4->} notroutes=%{fld18->} perlwait=%{fld27}", processor_chain([
	dup17,
	dup9,
]));

var msg29 = msg("session_connect:01", part34);

var select12 = linear_select([
	msg28,
	msg29,
]);

var part35 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' host='), Field(hostname,true), Constant(' resolve='), Field(fld28,true), Constant(' reverse='), Field(fld13,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,false)}"
match("MESSAGE#29:session_resolve", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} host=%{hostname->} resolve=%{fld28->} reverse=%{fld13->} routes=%{fld4->} notroutes=%{fld18}", processor_chain([
	dup17,
	dup9,
]));

var msg30 = msg("session_resolve", part35);

var part36 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' host='), Field(hostname,true), Constant(' resolve='), Field(fld28,true), Constant(' reverse='), Field(fld13,true), Constant(' routes='), Field(fld4,true), Constant(' notroutes='), Field(fld18,false)}"
match("MESSAGE#30:session_resolve:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} host=%{hostname->} resolve=%{fld28->} reverse=%{fld13->} routes=%{fld4->} notroutes=%{fld18}", processor_chain([
	dup17,
	dup9,
]));

var msg31 = msg("session_resolve:01", part36);

var select13 = linear_select([
	msg30,
	msg31,
]);

var part37 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' ip='), Field(hostip,true), Constant(' rate='), Field(fld29,true), Constant(' crate='), Field(fld30,true), Constant(' limit='), Field(fld31,false)}"
match("MESSAGE#31:session_throttle", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} ip=%{hostip->} rate=%{fld29->} crate=%{fld30->} limit=%{fld31}", processor_chain([
	dup17,
	dup9,
]));

var msg32 = msg("session_throttle", part37);

var part38 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' ip='), Field(hostip,true), Constant(' rate='), Field(fld29,true), Constant(' crate='), Field(fld30,true), Constant(' limit='), Field(fld31,false)}"
match("MESSAGE#32:session_throttle:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} ip=%{hostip->} rate=%{fld29->} crate=%{fld30->} limit=%{fld31}", processor_chain([
	dup17,
	dup9,
]));

var msg33 = msg("session_throttle:01", part38);

var select14 = linear_select([
	msg32,
	msg33,
]);

var part39 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' rate='), Field(fld58,false)}"
match("MESSAGE#33:session_dispose", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} rate=%{fld58}", processor_chain([
	dup22,
	dup9,
]));

var msg34 = msg("session_dispose", part39);

var part40 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' rate='), Field(fld58,false)}"
match("MESSAGE#34:session_dispose:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} rate=%{fld58}", processor_chain([
	dup22,
	dup9,
]));

var msg35 = msg("session_dispose:01", part40);

var part41 = // "Pattern{Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,false)}"
match("MESSAGE#35:session_dispose:02/2", "nwparser.p0", "%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action}");

var all7 = all_match({
	processors: [
		dup18,
		dup47,
		part41,
	],
	on_success: processor_chain([
		dup22,
		dup9,
		dup21,
	]),
});

var msg36 = msg("session_dispose:02", all7);

var part42 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,false)}"
match("MESSAGE#36:session_dispose:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action}", processor_chain([
	dup22,
	dup9,
]));

var msg37 = msg("session_dispose:03", part42);

var select15 = linear_select([
	msg34,
	msg35,
	msg36,
	msg37,
]);

var part43 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' helo='), Field(fld32,true), Constant(' msgs='), Field(fld33,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#37:session_disconnect", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} helo=%{fld32->} msgs=%{fld33->} rcpts=%{dclass_counter2->} routes=%{fld4->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup17,
	dup9,
	dup13,
]));

var msg38 = msg("session_disconnect", part43);

var part44 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' helo='), Field(fld32,true), Constant(' msgs='), Field(fld33,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#38:session_disconnect:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} helo=%{fld32->} msgs=%{fld33->} rcpts=%{dclass_counter2->} routes=%{fld4->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup17,
	dup9,
	dup13,
]));

var msg39 = msg("session_disconnect:01", part44);

var select16 = linear_select([
	msg38,
	msg39,
]);

var part45 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' attachment='), Field(fld58,true), Constant(' file='), Field(fld1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' name='), Field(fld34,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#39:av_run:02/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} attachment=%{fld58->} file=%{fld1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} name=%{fld34->} %{p0}");

var part46 = // "Pattern{Constant('cleaned='), Field(fld35,true), Constant(' vendor='), Field(fld36,true), Constant(' duration='), Field(p0,false)}"
match("MESSAGE#39:av_run:02/1_0", "nwparser.p0", "cleaned=%{fld35->} vendor=%{fld36->} duration=%{p0}");

var part47 = // "Pattern{Constant('vendor='), Field(fld36,true), Constant(' duration='), Field(p0,false)}"
match("MESSAGE#39:av_run:02/1_2", "nwparser.p0", "vendor=%{fld36->} duration=%{p0}");

var select17 = linear_select([
	part46,
	dup23,
	part47,
]);

var all8 = all_match({
	processors: [
		part45,
		select17,
		dup24,
	],
	on_success: processor_chain([
		dup25,
		dup9,
		dup21,
	]),
});

var msg40 = msg("av_run:02", all8);

var part48 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' attachment='), Field(fld58,true), Constant(' file='), Field(filename,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' name='), Field(fld34,true), Constant(' cleaned='), Field(fld35,true), Constant(' vendor='), Field(fld36,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#40:av_run:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} attachment=%{fld58->} file=%{filename->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} name=%{fld34->} cleaned=%{fld35->} vendor=%{fld36->} duration=%{duration_string}", processor_chain([
	dup25,
	dup9,
]));

var msg41 = msg("av_run:03", part48);

var part49 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#41:av_run/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} %{p0}");

var part50 = // "Pattern{Constant('name='), Field(fld34,true), Constant(' cleaned='), Field(fld35,true), Constant(' vendor='), Field(fld36,true), Constant(' duration='), Field(p0,false)}"
match("MESSAGE#41:av_run/1_1", "nwparser.p0", "name=%{fld34->} cleaned=%{fld35->} vendor=%{fld36->} duration=%{p0}");

var part51 = // "Pattern{Constant('name='), Field(fld34,true), Constant(' vendor='), Field(fld36,true), Constant(' duration='), Field(p0,false)}"
match("MESSAGE#41:av_run/1_2", "nwparser.p0", "name=%{fld34->} vendor=%{fld36->} duration=%{p0}");

var select18 = linear_select([
	dup23,
	part50,
	part51,
]);

var all9 = all_match({
	processors: [
		part49,
		select18,
		dup24,
	],
	on_success: processor_chain([
		dup25,
		dup9,
	]),
});

var msg42 = msg("av_run", all9);

var part52 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' name='), Field(fld34,true), Constant(' cleaned='), Field(fld35,true), Constant(' vendor='), Field(fld36,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#42:av_run:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} name=%{fld34->} cleaned=%{fld35->} vendor=%{fld36->} duration=%{duration_string}", processor_chain([
	dup25,
	dup9,
]));

var msg43 = msg("av_run:01", part52);

var select19 = linear_select([
	msg40,
	msg41,
	msg42,
	msg43,
]);

var msg44 = msg("av_refresh", dup48);

var msg45 = msg("av_init", dup48);

var part53 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,false)}"
match("MESSAGE#45:av_load", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5}", processor_chain([
	dup26,
	dup9,
]));

var msg46 = msg("av_load", part53);

var part54 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' attachment='), Field(fld58,true), Constant(' file='), Field(filename,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#46:access_run:02", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} attachment=%{fld58->} file=%{filename->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg47 = msg("access_run:02", part54);

var part55 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' attachment='), Field(fld58,true), Constant(' file='), Field(filename,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#47:access_run:04", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} attachment=%{fld58->} file=%{filename->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg48 = msg("access_run:04", part55);

var msg49 = msg("access_run:03", dup49);

var msg50 = msg("access_run:01", dup50);

var select20 = linear_select([
	msg47,
	msg48,
	msg49,
	msg50,
]);

var part56 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' action='), Field(action,true), Constant(' dict='), Field(fld37,true), Constant(' file='), Field(filename,false)}"
match("MESSAGE#50:access_refresh", "nwparser.payload", "%{fld0->} %{severity->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} action=%{action->} dict=%{fld37->} file=%{filename}", processor_chain([
	dup17,
	dup9,
]));

var msg51 = msg("access_refresh", part56);

var msg52 = msg("access_refresh:01", dup51);

var select21 = linear_select([
	msg51,
	msg52,
]);

var msg53 = msg("access_load", dup52);

var msg54 = msg("regulation_init", dup51);

var msg55 = msg("regulation_refresh", dup51);

var part57 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} policy=%{fld38->} score=%{fld39->} spamscore=%{reputation_num->} %{p0}");

var part58 = // "Pattern{Constant('ipscore='), Field(fld40,true), Constant(' suspectscore='), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/1_0", "nwparser.p0", "ipscore=%{fld40->} suspectscore=%{p0}");

var part59 = // "Pattern{Constant('suspectscore='), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/1_1", "nwparser.p0", "suspectscore=%{p0}");

var select22 = linear_select([
	part58,
	part59,
]);

var part60 = // "Pattern{Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/2", "nwparser.p0", "%{fld41->} phishscore=%{fld42->} %{p0}");

var part61 = // "Pattern{Constant('bulkscore='), Field(fld43,true), Constant(' adultscore='), Field(fld44,true), Constant(' classifier='), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/3_0", "nwparser.p0", "bulkscore=%{fld43->} adultscore=%{fld44->} classifier=%{p0}");

var part62 = // "Pattern{Constant('adultscore='), Field(fld44,true), Constant(' bulkscore='), Field(fld43,true), Constant(' classifier='), Field(p0,false)}"
match("MESSAGE#55:spam_run:rule/3_1", "nwparser.p0", "adultscore=%{fld44->} bulkscore=%{fld43->} classifier=%{p0}");

var select23 = linear_select([
	part61,
	part62,
]);

var part63 = // "Pattern{Field(fld45,true), Constant(' adjust='), Field(fld46,true), Constant(' reason='), Field(fld47,true), Constant(' scancount='), Field(fld48,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' raw='), Field(fld51,true), Constant(' tests='), Field(fld52,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#55:spam_run:rule/4", "nwparser.p0", "%{fld45->} adjust=%{fld46->} reason=%{fld47->} scancount=%{fld48->} engine=%{fld49->} definitions=%{fld50->} raw=%{fld51->} tests=%{fld52->} duration=%{duration_string}");

var all10 = all_match({
	processors: [
		part57,
		select22,
		part60,
		select23,
		part63,
	],
	on_success: processor_chain([
		dup27,
		dup9,
	]),
});

var msg56 = msg("spam_run:rule", all10);

var part64 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' ipscore='), Field(fld40,true), Constant(' suspectscore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' bulkscore='), Field(fld43,true), Constant(' adultscore='), Field(fld44,true), Constant(' classifier='), Field(fld45,true), Constant(' adjust='), Field(fld46,true), Constant(' reason='), Field(fld47,true), Constant(' scancount='), Field(fld48,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' raw='), Field(fld51,true), Constant(' tests='), Field(fld52,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#56:spam_run:rule_02", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} policy=%{fld38->} score=%{fld39->} spamscore=%{reputation_num->} ipscore=%{fld40->} suspectscore=%{fld41->} phishscore=%{fld42->} bulkscore=%{fld43->} adultscore=%{fld44->} classifier=%{fld45->} adjust=%{fld46->} reason=%{fld47->} scancount=%{fld48->} engine=%{fld49->} definitions=%{fld50->} raw=%{fld51->} tests=%{fld52->} duration=%{duration_string}", processor_chain([
	dup27,
	dup9,
]));

var msg57 = msg("spam_run:rule_02", part64);

var part65 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' ndrscore='), Field(fld57,true), Constant(' ipscore='), Field(fld40,true), Constant(' suspectscore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' bulkscore='), Field(fld43,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' adjustscore='), Field(fld58,true), Constant(' adultscore='), Field(fld44,true), Constant(' classifier='), Field(fld45,true), Constant(' adjust='), Field(fld46,true), Constant(' reason='), Field(fld47,true), Constant(' scancount='), Field(fld48,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' raw='), Field(fld51,true), Constant(' tests='), Field(fld52,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#57:spam_run:rule_03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} policy=%{fld38->} score=%{fld39->} ndrscore=%{fld57->} ipscore=%{fld40->} suspectscore=%{fld41->} phishscore=%{fld42->} bulkscore=%{fld43->} spamscore=%{reputation_num->} adjustscore=%{fld58->} adultscore=%{fld44->} classifier=%{fld45->} adjust=%{fld46->} reason=%{fld47->} scancount=%{fld48->} engine=%{fld49->} definitions=%{fld50->} raw=%{fld51->} tests=%{fld52->} duration=%{duration_string}", processor_chain([
	dup27,
	dup9,
]));

var msg58 = msg("spam_run:rule_03", part65);

var part66 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' kscore.is_bulkscore='), Field(fld57,true), Constant(' kscore.compositescore='), Field(fld40,true), Constant(' circleOfTrustscore='), Field(fld41,true), Constant(' compositescore='), Field(fld42,true), Constant(' urlsuspect_oldscore='), Field(fld43,true), Constant(' suspectscore='), Field(reputation_num,true), Constant(' recipient_domain_to_sender_totalscore='), Field(fld58,true), Constant(' phishscore='), Field(fld44,true), Constant(' bulkscore='), Field(fld45,true), Constant(' kscore.is_spamscore='), Field(fld46,true), Constant(' recipient_to_sender_totalscore='), Field(fld47,true), Constant(' recipient_domain_to_sender_domain_totalscore='), Field(fld48,true), Constant(' rbsscore='), Field(fld49,true), Constant(' spamscore='), Field(fld50,true), Constant(' recipient_to_sender_domain_totalscore='), Field(fld51,true), Constant(' urlsuspectscore='), Field(fld52,true), Constant(' '), Field(fld53,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#58:spam_run:rule_04", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} policy=%{fld38->} score=%{fld39->} kscore.is_bulkscore=%{fld57->} kscore.compositescore=%{fld40->} circleOfTrustscore=%{fld41->} compositescore=%{fld42->} urlsuspect_oldscore=%{fld43->} suspectscore=%{reputation_num->} recipient_domain_to_sender_totalscore=%{fld58->} phishscore=%{fld44->} bulkscore=%{fld45->} kscore.is_spamscore=%{fld46->} recipient_to_sender_totalscore=%{fld47->} recipient_domain_to_sender_domain_totalscore=%{fld48->} rbsscore=%{fld49->} spamscore=%{fld50->} recipient_to_sender_domain_totalscore=%{fld51->} urlsuspectscore=%{fld52->} %{fld53->} duration=%{duration_string}", processor_chain([
	dup27,
	dup9,
]));

var msg59 = msg("spam_run:rule_04", part66);

var part67 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' ndrscore='), Field(fld53,true), Constant(' suspectscore='), Field(fld40,true), Constant(' malwarescore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' bulkscore='), Field(fld43,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' adjustscore='), Field(fld54,true), Constant(' adultscore='), Field(fld44,true), Constant(' classifier='), Field(fld45,true), Constant(' adjust='), Field(fld46,true), Constant(' reason='), Field(fld47,true), Constant(' scancount='), Field(fld48,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' raw='), Field(fld51,true), Constant(' tests='), Field(fld52,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#59:spam_run:rule_05", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} policy=%{fld38->} score=%{fld39->} ndrscore=%{fld53->} suspectscore=%{fld40->} malwarescore=%{fld41->} phishscore=%{fld42->} bulkscore=%{fld43->} spamscore=%{reputation_num->} adjustscore=%{fld54->} adultscore=%{fld44->} classifier=%{fld45->} adjust=%{fld46->} reason=%{fld47->} scancount=%{fld48->} engine=%{fld49->} definitions=%{fld50->} raw=%{fld51->} tests=%{fld52->} duration=%{duration_string}", processor_chain([
	dup27,
	dup9,
]));

var msg60 = msg("spam_run:rule_05", part67);

var part68 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' mod='), Field(agent,true), Constant(' total_uri_count='), Field(dclass_counter1,true), Constant(' uris_excluded_from_report_info='), Field(dclass_counter2,false)}"
match("MESSAGE#60:spam_run:rule_06", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} mod=%{agent->} total_uri_count=%{dclass_counter1->} uris_excluded_from_report_info=%{dclass_counter2}", processor_chain([
	dup27,
	dup9,
]));

var msg61 = msg("spam_run:rule_06", part68);

var part69 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' action='), Field(action,true), Constant(' score='), Field(fld39,true), Constant(' submsgadjust='), Field(fld53,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' ipscore='), Field(fld40,true), Constant(' suspectscore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' bulkscore='), Field(fld43,true), Constant(' adultscore='), Field(fld44,true), Constant(' tests='), Field(fld52,false)}"
match("MESSAGE#61:spam_run:action_01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} action=%{action->} score=%{fld39->} submsgadjust=%{fld53->} spamscore=%{reputation_num->} ipscore=%{fld40->} suspectscore=%{fld41->} phishscore=%{fld42->} bulkscore=%{fld43->} adultscore=%{fld44->} tests=%{fld52}", processor_chain([
	dup27,
	dup9,
]));

var msg62 = msg("spam_run:action_01", part69);

var part70 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' action='), Field(action,true), Constant(' score='), Field(fld39,true), Constant(' submsgadjust='), Field(fld53,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' ipscore='), Field(fld40,true), Constant(' suspectscore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' bulkscore='), Field(fld43,true), Constant(' adultscore='), Field(fld44,true), Constant(' tests='), Field(fld52,false)}"
match("MESSAGE#62:spam_run:action", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} action=%{action->} score=%{fld39->} submsgadjust=%{fld53->} spamscore=%{reputation_num->} ipscore=%{fld40->} suspectscore=%{fld41->} phishscore=%{fld42->} bulkscore=%{fld43->} adultscore=%{fld44->} tests=%{fld52}", processor_chain([
	dup27,
	dup9,
]));

var msg63 = msg("spam_run:action", part70);

var part71 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' action='), Field(action,true), Constant(' num_domains='), Field(fld53,true), Constant(' num_domains_to_lookup='), Field(fld40,false)}"
match("MESSAGE#63:spam_run:action_02", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} action=%{action->} num_domains=%{fld53->} num_domains_to_lookup=%{fld40}", processor_chain([
	dup27,
	dup9,
]));

var msg64 = msg("spam_run:action_02", part71);

var select24 = linear_select([
	msg56,
	msg57,
	msg58,
	msg59,
	msg60,
	msg61,
	msg62,
	msg63,
	msg64,
]);

var msg65 = msg("spam_refresh", dup53);

var msg66 = msg("spam_init", dup53);

var part72 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,false)}"
match("MESSAGE#66:spam_load", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5}", processor_chain([
	dup27,
	dup9,
]));

var msg67 = msg("spam_load", part72);

var part73 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' policy='), Field(fld38,true), Constant(' address='), Field(fld54,false)}"
match("MESSAGE#67:batv_run", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} policy=%{fld38->} address=%{fld54}", processor_chain([
	dup17,
	dup9,
]));

var msg68 = msg("batv_run", part73);

var part74 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' policy='), Field(fld38,true), Constant(' address='), Field(fld54,false)}"
match("MESSAGE#68:batv_run:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} policy=%{fld38->} address=%{fld54}", processor_chain([
	dup17,
	dup9,
]));

var msg69 = msg("batv_run:01", part74);

var msg70 = msg("batv_run:02", dup49);

var msg71 = msg("batv_run:03", dup50);

var select25 = linear_select([
	msg68,
	msg69,
	msg70,
	msg71,
]);

var msg72 = msg("zerohour_refresh", dup54);

var msg73 = msg("zerohour_init", dup54);

var msg74 = msg("zerohour_load", dup52);

var part75 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' count='), Field(fld2,true), Constant(' name='), Field(fld34,true), Constant(' init_time='), Field(fld3,true), Constant(' init_virusthreat='), Field(fld4,true), Constant(' virusthreat='), Field(fld5,true), Constant(' virusthreatid='), Field(fld6,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#74:zerohour_run", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} count=%{fld2->} name=%{fld34->} init_time=%{fld3->} init_virusthreat=%{fld4->} virusthreat=%{fld5->} virusthreatid=%{fld6->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg75 = msg("zerohour_run", part75);

var part76 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' count='), Field(fld2,true), Constant(' name='), Field(fld34,true), Constant(' init_time='), Field(fld3,true), Constant(' init_virusthreat='), Field(fld4,true), Constant(' virusthreat='), Field(fld5,true), Constant(' virusthreatid='), Field(fld6,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#75:zerohour_run:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} count=%{fld2->} name=%{fld34->} init_time=%{fld3->} init_virusthreat=%{fld4->} virusthreat=%{fld5->} virusthreatid=%{fld6->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg76 = msg("zerohour_run:01", part76);

var select26 = linear_select([
	msg75,
	msg76,
]);

var part77 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#76:service_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg77 = msg("service_refresh", part77);

var part78 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#77:perl_clone", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} id=%{fld5->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg78 = msg("perl_clone", part78);

var part79 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' cset='), Field(fld56,true), Constant(' name='), Field(fld34,true), Constant(' status='), Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#78:cvt_convert", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} cset=%{fld56->} name=%{fld34->} status=%{result->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg79 = msg("cvt_convert", part79);

var part80 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' cset='), Field(fld56,true), Constant(' name='), Field(fld34,true), Constant(' status='), Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#79:cvt_convert:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} cset=%{fld56->} name=%{fld34->} status=%{result->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg80 = msg("cvt_convert:01", part80);

var part81 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' pid='), Field(process_id,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' cset='), Field(fld56,true), Constant(' name='), Field(fld34,true), Constant(' status='), Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#80:cvt_convert:02", "nwparser.payload", "%{fld0->} %{severity->} pid=%{process_id->} mod=%{agent->} cmd=%{obj_type->} cset=%{fld56->} name=%{fld34->} status=%{result->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg81 = msg("cvt_convert:02", part81);

var select27 = linear_select([
	msg79,
	msg80,
	msg81,
]);

var part82 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' pid='), Field(process_id,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' name='), Field(fld34,true), Constant(' status='), Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#81:cvt_detect", "nwparser.payload", "%{fld0->} %{severity->} pid=%{process_id->} mod=%{agent->} cmd=%{obj_type->} name=%{fld34->} status=%{result->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg82 = msg("cvt_detect", part82);

var msg83 = msg("cvtd:01", dup55);

var msg84 = msg("cvtd", dup56);

var select28 = linear_select([
	msg83,
	msg84,
]);

var part83 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' pid='), Field(fld5,true), Constant(' mod='), Field(agent,true), Constant(' encrypted='), Field(fld6,false)}"
match("MESSAGE#84:cvtd_encrypted", "nwparser.payload", "%{fld0->} %{severity->} pid=%{fld5->} mod=%{agent->} encrypted=%{fld6}", processor_chain([
	dup17,
	dup9,
]));

var msg85 = msg("cvtd_encrypted", part83);

var msg86 = msg("filter:01", dup55);

var msg87 = msg("filter", dup56);

var select29 = linear_select([
	msg86,
	msg87,
]);

var msg88 = msg("soap_listen", dup57);

var msg89 = msg("http_listen", dup57);

var part84 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' '), Field(event_description,false)}"
match("MESSAGE#89:mltr", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} %{event_description}", processor_chain([
	dup17,
	dup9,
]));

var msg90 = msg("mltr", part84);

var msg91 = msg("milter_listen", dup57);

var msg92 = msg("smtpsrv_load", dup52);

var msg93 = msg("smtpsrv_listen", dup57);

var part85 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#93:smtpsrv_run", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg94 = msg("smtpsrv_run", part85);

var part86 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#94:smtpsrv/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} %{p0}");

var part87 = // "Pattern{Field(result,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#94:smtpsrv/1_0", "nwparser.p0", "%{result->} err=%{fld58}");

var part88 = // "Pattern{Field(result,false)}"
match_copy("MESSAGE#94:smtpsrv/1_1", "nwparser.p0", "result");

var select30 = linear_select([
	part87,
	part88,
]);

var all11 = all_match({
	processors: [
		part86,
		select30,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg95 = msg("smtpsrv", all11);

var part89 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' cmd='), Field(obj_type,true), Constant(' profile='), Field(fld52,true), Constant(' qid='), Field(fld15,true), Constant(' rcpts='), Field(to,false)}"
match("MESSAGE#95:send", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} cmd=%{obj_type->} profile=%{fld52->} qid=%{fld15->} rcpts=%{to}", processor_chain([
	dup17,
	dup9,
]));

var msg96 = msg("send", part89);

var part90 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' cmd='), Field(obj_type,true), Constant(' profile='), Field(fld52,true), Constant(' qid='), Field(fld15,true), Constant(' rcpts='), Field(to,false)}"
match("MESSAGE#96:send:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} cmd=%{obj_type->} profile=%{fld52->} qid=%{fld15->} rcpts=%{to}", processor_chain([
	dup17,
	dup9,
]));

var msg97 = msg("send:01", part90);

var part91 = // "Pattern{Field(fld0,false), Constant(': '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' cmd='), Field(obj_type,true), Constant(' rcpt='), Field(to,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#97:send:02", "nwparser.payload", "%{fld0}: %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} cmd=%{obj_type->} rcpt=%{to->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg98 = msg("send:02", part91);

var select31 = linear_select([
	msg96,
	msg97,
	msg98,
]);

var part92 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(p0,false)}"
match("MESSAGE#98:queued-alert/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{p0}");

var part93 = // "Pattern{Field(fld55,true), Constant(' tls_verify='), Field(fld70,false), Constant(', pri='), Field(p0,false)}"
match("MESSAGE#98:queued-alert/1_0", "nwparser.p0", "%{fld55->} tls_verify=%{fld70}, pri=%{p0}");

var part94 = // "Pattern{Field(fld55,false), Constant(', pri='), Field(p0,false)}"
match("MESSAGE#98:queued-alert/1_1", "nwparser.p0", "%{fld55}, pri=%{p0}");

var select32 = linear_select([
	part93,
	part94,
]);

var part95 = // "Pattern{Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#98:queued-alert/2", "nwparser.p0", "%{fld23}, relay=%{p0}");

var all12 = all_match({
	processors: [
		part92,
		select32,
		part95,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg99 = msg("queued-alert", all12);

var part96 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld57,false), Constant(', nrcpts='), Field(fld58,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(fld69,false), Constant(', tls_verify='), Field(fld70,false), Constant(', auth='), Field(authmethod,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#99:queued-alert:01/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld57}, nrcpts=%{fld58}, msgid=%{id}, proto=%{protocol}, daemon=%{fld69}, tls_verify=%{fld70}, auth=%{authmethod}, relay=%{p0}");

var part97 = // "Pattern{Constant('['), Field(fld50,false), Constant('] ['), Field(daddr,false), Constant(']')}"
match("MESSAGE#99:queued-alert:01/1_0", "nwparser.p0", "[%{fld50}] [%{daddr}]");

var select33 = linear_select([
	part97,
	dup33,
	dup34,
	dup35,
]);

var all13 = all_match({
	processors: [
		part96,
		select33,
	],
	on_success: processor_chain([
		dup17,
		dup9,
		dup36,
	]),
});

var msg100 = msg("queued-alert:01", all13);

var part98 = // "Pattern{Constant('['), Field(fld50,false), Constant('] ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#100:queued-alert:02/1_0", "nwparser.p0", "[%{fld50}] [%{daddr}],%{p0}");

var select34 = linear_select([
	part98,
	dup29,
	dup30,
	dup31,
]);

var part99 = // "Pattern{Field(,false), Constant('version='), Field(version,false), Constant(', verify='), Field(fld57,false), Constant(', cipher='), Field(s_cipher,false), Constant(', bits='), Field(fld59,false)}"
match("MESSAGE#100:queued-alert:02/2", "nwparser.p0", "%{}version=%{version}, verify=%{fld57}, cipher=%{s_cipher}, bits=%{fld59}");

var all14 = all_match({
	processors: [
		dup37,
		select34,
		part99,
	],
	on_success: processor_chain([
		dup17,
		dup9,
		dup36,
	]),
});

var msg101 = msg("queued-alert:02", all14);

var select35 = linear_select([
	msg99,
	msg100,
	msg101,
]);

var msg102 = msg("queued-VoltageEncrypt", dup63);

var msg103 = msg("queued-VoltageEncrypt:01", dup64);

var select36 = linear_select([
	msg102,
	msg103,
]);

var msg104 = msg("queued-default", dup63);

var msg105 = msg("queued-default:01", dup64);

var msg106 = msg("queued-default:02", dup65);

var msg107 = msg("queued-default:03", dup66);

var msg108 = msg("queued-default:04", dup60);

var select37 = linear_select([
	msg104,
	msg105,
	msg106,
	msg107,
	msg108,
]);

var msg109 = msg("queued-reinject", dup63);

var msg110 = msg("queued-reinject:01", dup64);

var msg111 = msg("queued-reinject:02", dup65);

var msg112 = msg("queued-reinject:03", dup66);

var part100 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': maxrcpts='), Field(fld56,false), Constant(', rcpts='), Field(fld57,false), Constant(', count='), Field(fld58,false), Constant(', ids='), Field(fld59,false)}"
match("MESSAGE#111:queued-reinject:05", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: maxrcpts=%{fld56}, rcpts=%{fld57}, count=%{fld58}, ids=%{fld59}", processor_chain([
	dup17,
	dup9,
]));

var msg113 = msg("queued-reinject:05", part100);

var msg114 = msg("queued-reinject:04", dup60);

var msg115 = msg("queued-reinject:06", dup61);

var select38 = linear_select([
	msg109,
	msg110,
	msg111,
	msg112,
	msg113,
	msg114,
	msg115,
]);

var part101 = // "Pattern{Field(,false), Constant('version='), Field(version,false), Constant(', verify='), Field(disposition,false), Constant(', cipher='), Field(fld58,false), Constant(', bits='), Field(fld59,false)}"
match("MESSAGE#114:queued-eurort/2", "nwparser.p0", "%{}version=%{version}, verify=%{disposition}, cipher=%{fld58}, bits=%{fld59}");

var all15 = all_match({
	processors: [
		dup37,
		dup58,
		part101,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg116 = msg("queued-eurort", all15);

var msg117 = msg("queued-eurort:01", dup63);

var msg118 = msg("queued-eurort:02", dup67);

var msg119 = msg("queued-eurort:03", dup60);

var select39 = linear_select([
	msg116,
	msg117,
	msg118,
	msg119,
]);

var msg120 = msg("queued-vdedc2v5", dup63);

var msg121 = msg("queued-vdedc2v5:01", dup67);

var select40 = linear_select([
	msg120,
	msg121,
]);

var msg122 = msg("sm-msp-queue", dup66);

var part102 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: starting daemon ('), Field(fld7,false), Constant('): '), Field(fld6,false)}"
match("MESSAGE#122:sm-msp-queue:01", "nwparser.payload", "%{agent}[%{process_id}]: starting daemon (%{fld7}): %{fld6}", processor_chain([
	setc("eventcategory","1605000000"),
	dup9,
]));

var msg123 = msg("sm-msp-queue:01", part102);

var part103 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': to='), Field(to,false), Constant(', ctladdr='), Field(fld13,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#123:sm-msp-queue:02/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: to=%{to}, ctladdr=%{fld13}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, pri=%{fld23}, relay=%{p0}");

var all16 = all_match({
	processors: [
		part103,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg124 = msg("sm-msp-queue:02", all16);

var select41 = linear_select([
	msg122,
	msg123,
	msg124,
]);

var part104 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', tls_verify='), Field(fld24,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#124:sendmail:15/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, tls_verify=%{fld24}, pri=%{fld23}, relay=%{p0}");

var part105 = // "Pattern{Field(dhost,false), Constant('. ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#124:sendmail:15/1_1", "nwparser.p0", "%{dhost}. [%{daddr}],%{p0}");

var part106 = // "Pattern{Field(dhost,false), Constant('.,'), Field(p0,false)}"
match("MESSAGE#124:sendmail:15/1_2", "nwparser.p0", "%{dhost}.,%{p0}");

var select42 = linear_select([
	dup28,
	part105,
	part106,
]);

var all17 = all_match({
	processors: [
		part104,
		select42,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg125 = msg("sendmail:15", all17);

var part107 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld54,false), Constant(', nrcpts='), Field(fld55,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(p0,false)}"
match("MESSAGE#125:sendmail:14/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld54}, nrcpts=%{fld55}, msgid=%{id}, proto=%{protocol}, daemon=%{p0}");

var part108 = // "Pattern{Field(fld69,false), Constant(', tls_verify='), Field(fld70,false), Constant(', auth='), Field(authmethod,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#125:sendmail:14/1_0", "nwparser.p0", "%{fld69}, tls_verify=%{fld70}, auth=%{authmethod}, relay=%{p0}");

var part109 = // "Pattern{Field(fld69,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#125:sendmail:14/1_1", "nwparser.p0", "%{fld69}, relay=%{p0}");

var select43 = linear_select([
	part108,
	part109,
]);

var all18 = all_match({
	processors: [
		part107,
		select43,
		dup59,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg126 = msg("sendmail:14", all18);

var msg127 = msg("sendmail", dup68);

var part110 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': available mech='), Field(fld2,false), Constant(', allowed mech='), Field(fld3,false)}"
match("MESSAGE#127:sendmail:01", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: available mech=%{fld2}, allowed mech=%{fld3}", processor_chain([
	dup17,
	dup9,
]));

var msg128 = msg("sendmail:01", part110);

var part111 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': milter='), Field(fld2,false), Constant(', action='), Field(action,false), Constant(', reject='), Field(fld3,false)}"
match("MESSAGE#128:sendmail:02", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: milter=%{fld2}, action=%{action}, reject=%{fld3}", processor_chain([
	dup17,
	dup9,
]));

var msg129 = msg("sendmail:02", part111);

var part112 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(': '), Field(fld57,false), Constant(': host='), Field(hostname,false), Constant(', addr='), Field(saddr,false), Constant(', reject='), Field(fld3,false)}"
match("MESSAGE#129:sendmail:03", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}: %{fld57}: host=%{hostname}, addr=%{saddr}, reject=%{fld3}", processor_chain([
	dup17,
	dup9,
]));

var msg130 = msg("sendmail:03", part112);

var part113 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': Milter '), Field(action,false), Constant(': '), Field(fld2,false), Constant(': '), Field(fld3,false), Constant(': vendor='), Field(fld36,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' signatures='), Field(fld94,false)}"
match("MESSAGE#130:sendmail:08", "nwparser.payload", "%{fld10->} %{agent}[%{process_id}]: %{fld1}: Milter %{action}: %{fld2}: %{fld3}: vendor=%{fld36->} engine=%{fld49->} definitions=%{fld50->} signatures=%{fld94}", processor_chain([
	dup17,
	dup9,
]));

var msg131 = msg("sendmail:08", part113);

var part114 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': Milter '), Field(action,false), Constant(': '), Field(fld2,false), Constant(': '), Field(fld3,false), Constant(': rule='), Field(rulename,true), Constant(' policy='), Field(fld38,true), Constant(' score='), Field(fld39,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' suspectscore='), Field(fld41,true), Constant(' phishscore='), Field(fld42,true), Constant(' adultscore='), Field(fld44,true), Constant(' bulkscore='), Field(fld43,true), Constant(' classifier='), Field(fld45,true), Constant(' adjust='), Field(fld46,true), Constant(' reason='), Field(fld47,true), Constant(' scancount='), Field(fld48,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,false)}"
match("MESSAGE#131:sendmail:09", "nwparser.payload", "%{fld10->} %{agent}[%{process_id}]: %{fld1}: Milter %{action}: %{fld2}: %{fld3}: rule=%{rulename->} policy=%{fld38->} score=%{fld39->} spamscore=%{reputation_num->} suspectscore=%{fld41->} phishscore=%{fld42->} adultscore=%{fld44->} bulkscore=%{fld43->} classifier=%{fld45->} adjust=%{fld46->} reason=%{fld47->} scancount=%{fld48->} engine=%{fld49->} definitions=%{fld50}", processor_chain([
	dup17,
	dup9,
]));

var msg132 = msg("sendmail:09", part114);

var part115 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': Milter '), Field(action,false), Constant(': rcpt'), Field(p0,false)}"
match("MESSAGE#132:sendmail:10/0", "nwparser.payload", "%{fld10->} %{agent}[%{process_id}]: %{fld1}: Milter %{action}: rcpt%{p0}");

var part116 = // "Pattern{Constant(': '), Field(p0,false)}"
match("MESSAGE#132:sendmail:10/1_0", "nwparser.p0", ": %{p0}");

var part117 = // "Pattern{Field(p0,false)}"
match_copy("MESSAGE#132:sendmail:10/1_1", "nwparser.p0", "p0");

var select44 = linear_select([
	part116,
	part117,
]);

var part118 = // "Pattern{Field(,true), Constant(' '), Field(fld2,false)}"
match("MESSAGE#132:sendmail:10/2", "nwparser.p0", "%{} %{fld2}");

var all19 = all_match({
	processors: [
		part115,
		select44,
		part118,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg133 = msg("sendmail:10", all19);

var part119 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant('['), Field(process_id,false), Constant(']: STARTTLS='), Field(fld1,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#133:sendmail:11/0", "nwparser.payload", "%{fld10->} %{agent}[%{process_id}]: STARTTLS=%{fld1}, relay=%{p0}");

var all20 = all_match({
	processors: [
		part119,
		dup58,
		dup42,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg134 = msg("sendmail:11", all20);

var part120 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': SYSERR('), Field(fld2,false), Constant('): '), Field(action,false), Constant(': '), Field(event_description,true), Constant(' from '), Field(from,false), Constant(', from='), Field(fld3,false)}"
match("MESSAGE#134:sendmail:12", "nwparser.payload", "%{fld10->} %{agent}[%{process_id}]: %{fld1}: SYSERR(%{fld2}): %{action}: %{event_description->} from %{from}, from=%{fld3}", processor_chain([
	dup17,
	dup9,
]));

var msg135 = msg("sendmail:12", part120);

var part121 = // "Pattern{Field(fld10,true), Constant(' '), Field(agent,false), Constant(']'), Field(p0,false)}"
match("MESSAGE#135:sendmail:13/0_0", "nwparser.payload", "%{fld10->} %{agent}]%{p0}");

var part122 = // "Pattern{Field(agent,false), Constant(']'), Field(p0,false)}"
match("MESSAGE#135:sendmail:13/0_1", "nwparser.payload", "%{agent}]%{p0}");

var select45 = linear_select([
	part121,
	part122,
]);

var part123 = // "Pattern{Field(process_id,false), Constant('[: '), Field(fld1,false), Constant(': SYSERR('), Field(fld2,false), Constant('): '), Field(action,false), Constant(': '), Field(event_description,true), Constant(' file '), Field(filename,false), Constant(': '), Field(fld3,false)}"
match("MESSAGE#135:sendmail:13/1", "nwparser.p0", "%{process_id}[: %{fld1}: SYSERR(%{fld2}): %{action}: %{event_description->} file %{filename}: %{fld3}");

var all21 = all_match({
	processors: [
		select45,
		part123,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg136 = msg("sendmail:13", all21);

var part124 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(': '), Field(fld57,false), Constant(':'), Field(event_description,false)}"
match("MESSAGE#136:sendmail:04", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}: %{fld57}:%{event_description}", processor_chain([
	dup17,
	dup9,
]));

var msg137 = msg("sendmail:04", part124);

var part125 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(':'), Field(event_description,false)}"
match("MESSAGE#137:sendmail:05", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}:%{event_description}", processor_chain([
	dup17,
	dup9,
]));

var msg138 = msg("sendmail:05", part125);

var part126 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: AUTH='), Field(authmethod,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#169:sendmail:06/0", "nwparser.payload", "%{agent}[%{process_id}]: AUTH=%{authmethod}, relay=%{p0}");

var part127 = // "Pattern{Field(,false), Constant('authid='), Field(uid,false), Constant(', mech='), Field(scheme,false), Constant(', bits='), Field(fld59,false)}"
match("MESSAGE#169:sendmail:06/2", "nwparser.p0", "%{}authid=%{uid}, mech=%{scheme}, bits=%{fld59}");

var all22 = all_match({
	processors: [
		part126,
		dup58,
		part127,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg139 = msg("sendmail:06", all22);

var msg140 = msg("sendmail:07", dup61);

var select46 = linear_select([
	msg125,
	msg126,
	msg127,
	msg128,
	msg129,
	msg130,
	msg131,
	msg132,
	msg133,
	msg134,
	msg135,
	msg136,
	msg137,
	msg138,
	msg139,
	msg140,
]);

var part128 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' pid='), Field(process_id,true), Constant(' status='), Field(fld29,false)}"
match("MESSAGE#138:info:eid_pid_status", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} pid=%{process_id->} status=%{fld29}", processor_chain([
	dup17,
	dup9,
]));

var msg141 = msg("info:eid_pid_status", part128);

var part129 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' status='), Field(fld29,false)}"
match("MESSAGE#139:info:eid_status", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} status=%{fld29}", processor_chain([
	dup17,
	dup9,
]));

var msg142 = msg("info:eid_status", part129);

var part130 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' '), Field(info,false)}"
match("MESSAGE#140:info:eid", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} %{info}", processor_chain([
	dup17,
	dup9,
]));

var msg143 = msg("info:eid", part130);

var msg144 = msg("info:pid", dup62);

var part131 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(p0,false)}"
match("MESSAGE#143:info/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{p0}");

var part132 = // "Pattern{Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' ofrom='), Field(from,false)}"
match("MESSAGE#143:info/1_0", "nwparser.p0", "%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} ofrom=%{from}");

var part133 = // "Pattern{Field(sessionid1,true), Constant(' status='), Field(info,true), Constant(' restquery_stage='), Field(fld3,false)}"
match("MESSAGE#143:info/1_1", "nwparser.p0", "%{sessionid1->} status=%{info->} restquery_stage=%{fld3}");

var part134 = // "Pattern{Field(sessionid1,false)}"
match_copy("MESSAGE#143:info/1_2", "nwparser.p0", "sessionid1");

var select47 = linear_select([
	part132,
	part133,
	part134,
]);

var all23 = all_match({
	processors: [
		part131,
		select47,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg145 = msg("info", all23);

var part135 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' sys='), Field(fld1,true), Constant(' evt='), Field(action,true), Constant(' active='), Field(fld2,true), Constant(' expires='), Field(fld3,true), Constant(' msg='), Field(event_description,false)}"
match("MESSAGE#144:info:02", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} sys=%{fld1->} evt=%{action->} active=%{fld2->} expires=%{fld3->} msg=%{event_description}", processor_chain([
	dup17,
	dup9,
]));

var msg146 = msg("info:02", part135);

var part136 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' server='), Field(saddr,true), Constant(' elapsed='), Field(duration_string,true), Constant(' avgtime='), Field(fld2,true), Constant(' qname='), Field(fld3,true), Constant(' qtype='), Field(fld4,false)}"
match("MESSAGE#145:info:03", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} server=%{saddr->} elapsed=%{duration_string->} avgtime=%{fld2->} qname=%{fld3->} qtype=%{fld4}", processor_chain([
	dup17,
	dup9,
]));

var msg147 = msg("info:03", part136);

var part137 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' '), Field(web_method,true), Constant(' /'), Field(info,false), Constant(': '), Field(resultcode,false)}"
match("MESSAGE#146:info:01", "nwparser.payload", "%{fld0->} %{severity->} %{web_method->} /%{info}: %{resultcode}", processor_chain([
	dup17,
	dup9,
]));

var msg148 = msg("info:01", part137);

var part138 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' sys='), Field(fld1,true), Constant(' evt='), Field(p0,false)}"
match("MESSAGE#147:info:04/0", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} sys=%{fld1->} evt=%{p0}");

var part139 = // "Pattern{Field(action,true), Constant(' msg='), Field(event_description,false)}"
match("MESSAGE#147:info:04/1_0", "nwparser.p0", "%{action->} msg=%{event_description}");

var part140 = // "Pattern{Field(action,false)}"
match_copy("MESSAGE#147:info:04/1_1", "nwparser.p0", "action");

var select48 = linear_select([
	part139,
	part140,
]);

var all24 = all_match({
	processors: [
		part138,
		select48,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg149 = msg("info:04", all24);

var part141 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#148:info:05/0", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} %{p0}");

var part142 = // "Pattern{Constant('type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,false)}"
match("MESSAGE#148:info:05/1_0", "nwparser.p0", "type=%{fld6->} cmd=%{obj_type->} id=%{fld5}");

var part143 = // "Pattern{Constant('cmd='), Field(obj_type,false)}"
match("MESSAGE#148:info:05/1_1", "nwparser.p0", "cmd=%{obj_type}");

var select49 = linear_select([
	part142,
	part143,
]);

var all25 = all_match({
	processors: [
		part141,
		select49,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg150 = msg("info:05", all25);

var select50 = linear_select([
	msg141,
	msg142,
	msg143,
	msg144,
	msg145,
	msg146,
	msg147,
	msg148,
	msg149,
	msg150,
]);

var msg151 = msg("note:pid", dup62);

var part144 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' module='), Field(agent,true), Constant(' action='), Field(action,true), Constant(' size='), Field(bytes,false)}"
match("MESSAGE#149:note:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} module=%{agent->} action=%{action->} size=%{bytes}", processor_chain([
	dup17,
	dup9,
]));

var msg152 = msg("note:01", part144);

var select51 = linear_select([
	msg151,
	msg152,
]);

var part145 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' secprofile_name='), Field(fld3,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#150:rprt", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} secprofile_name=%{fld3->} rcpts=%{dclass_counter2->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var msg153 = msg("rprt", part145);

var part146 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' module='), Field(agent,true), Constant(' age='), Field(fld6,true), Constant(' limit='), Field(fld31,false)}"
match("MESSAGE#151:err", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} module=%{agent->} age=%{fld6->} limit=%{fld31}", processor_chain([
	dup17,
	dup9,
]));

var msg154 = msg("err", part146);

var part147 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' eid='), Field(fld4,true), Constant(' result='), Field(result,false)}"
match("MESSAGE#152:warn", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} eid=%{fld4->} result=%{result}", processor_chain([
	dup17,
	dup9,
]));

var msg155 = msg("warn", part147);

var part148 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' status="'), Field(event_state,true), Constant(' file: '), Field(filename,false), Constant('"')}"
match("MESSAGE#153:warn:01", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} status=\"%{event_state->} file: %{filename}\"", processor_chain([
	dup17,
	dup9,
]));

var msg156 = msg("warn:01", part148);

var part149 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' eid='), Field(fld4,true), Constant(' status="'), Field(event_state,true), Constant(' file '), Field(filename,true), Constant(' does not contain enough (or correct) info. Fix this or remove the file."')}"
match("MESSAGE#154:warn:02", "nwparser.payload", "%{fld0->} %{severity->} eid=%{fld4->} status=\"%{event_state->} file %{filename->} does not contain enough (or correct) info. Fix this or remove the file.\"", processor_chain([
	dup17,
	dup9,
	setc("event_description","does not contain enough (or correct) info. Fix this or remove the file"),
]));

var msg157 = msg("warn:02", part149);

var select52 = linear_select([
	msg155,
	msg156,
	msg157,
]);

var msg158 = msg("queued-aglife", dup68);

var msg159 = msg("pdr_run", dup50);

var part150 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' ttl='), Field(fld1,true), Constant(' reply="'), Field(p0,false)}"
match("MESSAGE#157:pdr_ttl/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} ttl=%{fld1->} reply=\"%{p0}");

var part151 = // "Pattern{Constant('\"'), Field(fld2,true), Constant(' rscore='), Field(fld3,false), Constant('\""')}"
match("MESSAGE#157:pdr_ttl/1_0", "nwparser.p0", "\\\"%{fld2->} rscore=%{fld3}\\\"\"");

var part152 = // "Pattern{Field(fld2,false), Constant('"')}"
match("MESSAGE#157:pdr_ttl/1_1", "nwparser.p0", "%{fld2}\"");

var select53 = linear_select([
	part151,
	part152,
]);

var all26 = all_match({
	processors: [
		part150,
		select53,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var msg160 = msg("pdr_ttl", all26);

var part153 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' signature='), Field(fld1,true), Constant(' identity='), Field(sigid_string,true), Constant(' host='), Field(hostname,true), Constant(' result='), Field(result,true), Constant(' result_detail='), Field(fld2,false)}"
match("MESSAGE#158:dkimv_run:signature", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} signature=%{fld1->} identity=%{sigid_string->} host=%{hostname->} result=%{result->} result_detail=%{fld2}", processor_chain([
	dup17,
	dup9,
]));

var msg161 = msg("dkimv_run:signature", part153);

var part154 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' status="'), Field(info,false), Constant(', '), Field(event_state,false), Constant('"')}"
match("MESSAGE#159:dkimv_run:status", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} status=\"%{info}, %{event_state}\"", processor_chain([
	dup17,
	dup9,
]));

var msg162 = msg("dkimv_run:status", part154);

var select54 = linear_select([
	msg161,
	msg162,
]);

var part155 = // "Pattern{Field(fld0,false), Constant(': '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' unexpected response type='), Field(fld1,false)}"
match("MESSAGE#160:dkimv_type", "nwparser.payload", "%{fld0}: %{severity->} mod=%{agent->} unexpected response type=%{fld1}", processor_chain([
	dup17,
	dup9,
	setc("result","unexpected response"),
]));

var msg163 = msg("dkimv_type", part155);

var part156 = // "Pattern{Field(fld0,false), Constant(': '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld1,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' publickey_cache_entries='), Field(fld6,false)}"
match("MESSAGE#161:dkimv_type:01", "nwparser.payload", "%{fld0}: %{severity->} mod=%{agent->} type=%{fld1->} cmd=%{obj_type->} id=%{fld5->} publickey_cache_entries=%{fld6}", processor_chain([
	dup17,
	dup9,
]));

var msg164 = msg("dkimv_type:01", part156);

var select55 = linear_select([
	msg163,
	msg164,
]);

var msg165 = msg("dmarc_run:rule", dup49);

var part157 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' result='), Field(result,true), Constant(' result_detail='), Field(fld2,false)}"
match("MESSAGE#163:dmarc_run:result", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} result=%{result->} result_detail=%{fld2}", processor_chain([
	dup17,
	dup9,
]));

var msg166 = msg("dmarc_run:result", part157);

var select56 = linear_select([
	msg165,
	msg166,
]);

var part158 = // "Pattern{Field(fld0,false), Constant(': '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld1,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' policy_cache_entries='), Field(fld6,false)}"
match("MESSAGE#164:dmarc_type", "nwparser.payload", "%{fld0}: %{severity->} mod=%{agent->} type=%{fld1->} cmd=%{obj_type->} id=%{fld5->} policy_cache_entries=%{fld6}", processor_chain([
	dup17,
	dup9,
]));

var msg167 = msg("dmarc_type", part158);

var msg168 = msg("spf_run:rule", dup49);

var part159 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' cmd='), Field(obj_type,true), Constant(' result='), Field(result,false)}"
match("MESSAGE#166:spf_run:cmd", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} cmd=%{obj_type->} result=%{result}", processor_chain([
	dup17,
	dup9,
]));

var msg169 = msg("spf_run:cmd", part159);

var select57 = linear_select([
	msg168,
	msg169,
]);

var part160 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' action='), Field(action,true), Constant(' score='), Field(fld39,true), Constant(' submsgadjust='), Field(fld53,true), Constant(' spamscore='), Field(reputation_num,true), Constant(' suspectscore='), Field(fld41,true), Constant(' malwarescore='), Field(fld49,true), Constant(' phishscore='), Field(fld42,true), Constant(' adultscore='), Field(fld44,true), Constant(' bulkscore='), Field(fld43,true), Constant(' tests='), Field(fld52,false)}"
match("MESSAGE#167:action_checksubmsg", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} action=%{action->} score=%{fld39->} submsgadjust=%{fld53->} spamscore=%{reputation_num->} suspectscore=%{fld41->} malwarescore=%{fld49->} phishscore=%{fld42->} adultscore=%{fld44->} bulkscore=%{fld43->} tests=%{fld52}", processor_chain([
	dup17,
	dup9,
]));

var msg170 = msg("action_checksubmsg", part160);

var part161 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' authscope='), Field(fld5,true), Constant(' err='), Field(fld58,false)}"
match("MESSAGE#168:rest_oauth", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} authscope=%{fld5->} err=%{fld58}", processor_chain([
	dup17,
	dup9,
]));

var msg171 = msg("rest_oauth", part161);

var part162 = // "Pattern{Constant('mod='), Field(agent,true), Constant(' type='), Field(fld1,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(id,true), Constant(' load smartid ccard')}"
match("MESSAGE#171:filter_instance1:01", "nwparser.payload", "mod=%{agent->} type=%{fld1->} cmd=%{obj_type->} id=%{id->} load smartid ccard", processor_chain([
	dup17,
	dup9,
	setc("event_description","load smartid ccard"),
	dup36,
]));

var msg172 = msg("filter_instance1:01", part162);

var part163 = // "Pattern{Constant('mod='), Field(agent,true), Constant(' type='), Field(fld1,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(id,true), Constant(' load smartid jcb')}"
match("MESSAGE#172:filter_instance1:02", "nwparser.payload", "mod=%{agent->} type=%{fld1->} cmd=%{obj_type->} id=%{id->} load smartid jcb", processor_chain([
	dup17,
	dup9,
	setc("event_description","load smartid jcb"),
	dup36,
]));

var msg173 = msg("filter_instance1:02", part163);

var part164 = // "Pattern{Constant('s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' attachments='), Field(dclass_counter1,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' size='), Field(bytes,true), Constant(' guid='), Field(fld14,true), Constant(' hdr_mid='), Field(id,true), Constant(' qid='), Field(fld15,true), Constant(' subject="'), Field(subject,false), Constant('" '), Field(p0,false)}"
match("MESSAGE#173:filter_instance1:03/0", "nwparser.payload", "s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} attachments=%{dclass_counter1->} rcpts=%{dclass_counter2->} routes=%{fld4->} size=%{bytes->} guid=%{fld14->} hdr_mid=%{id->} qid=%{fld15->} subject=\"%{subject}\" %{p0}");

var part165 = // "Pattern{Constant('spamscore='), Field(reputation_num,true), Constant(' virusname='), Field(threat_name,true), Constant(' duration='), Field(p0,false)}"
match("MESSAGE#173:filter_instance1:03/1_0", "nwparser.p0", "spamscore=%{reputation_num->} virusname=%{threat_name->} duration=%{p0}");

var part166 = // "Pattern{Constant('duration='), Field(p0,false)}"
match("MESSAGE#173:filter_instance1:03/1_1", "nwparser.p0", "duration=%{p0}");

var select58 = linear_select([
	part165,
	part166,
]);

var part167 = // "Pattern{Field(fld16,true), Constant(' elapsed='), Field(duration_string,false)}"
match("MESSAGE#173:filter_instance1:03/2", "nwparser.p0", "%{fld16->} elapsed=%{duration_string}");

var all27 = all_match({
	processors: [
		part164,
		select58,
		part167,
	],
	on_success: processor_chain([
		dup11,
		dup9,
		dup12,
		dup13,
		dup36,
	]),
});

var msg174 = msg("filter_instance1:03", all27);

var part168 = // "Pattern{Constant('s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' module='), Field(event_source,true), Constant(' rule='), Field(rulename,true), Constant(' action='), Field(action,true), Constant(' helo='), Field(fld32,true), Constant(' msgs='), Field(fld33,true), Constant(' rcpts='), Field(dclass_counter2,true), Constant(' routes='), Field(fld4,true), Constant(' duration='), Field(duration_string,true), Constant(' elapsed='), Field(fld16,false)}"
match("MESSAGE#174:filter_instance1:04", "nwparser.payload", "s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} module=%{event_source->} rule=%{rulename->} action=%{action->} helo=%{fld32->} msgs=%{fld33->} rcpts=%{dclass_counter2->} routes=%{fld4->} duration=%{duration_string->} elapsed=%{fld16}", processor_chain([
	dup17,
	dup9,
	dup13,
	dup36,
]));

var msg175 = msg("filter_instance1:04", part168);

var part169 = // "Pattern{Constant('s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' header.from="\"'), Field(info,false), Constant('\" '), Field(fld4,true), Constant(' <<'), Field(user_address,false), Constant('>"')}"
match("MESSAGE#175:filter_instance1:05", "nwparser.payload", "s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} header.from=\"\\\"%{info}\\\" %{fld4->} \u003c\u003c%{user_address}>\"", processor_chain([
	dup17,
	dup9,
	dup36,
]));

var msg176 = msg("filter_instance1:05", part169);

var part170 = tagval("MESSAGE#176:filter_instance1", "nwparser.payload", tvm, {
	"X-Proofpoint-Spam-Details": "fld71",
	"a": "fld12",
	"action": "action",
	"active": "fld2",
	"addr": "saddr",
	"adjust": "fld46",
	"adjustscore": "fld54",
	"adultscore": "fld44",
	"alert": "fld53",
	"attachments": "fld80",
	"avgtime": "fld2",
	"bulkscore": "fld43",
	"cipher": "s_cipher",
	"cipher_bits": "fld59",
	"classifier": "fld45",
	"cmd": "obj_type",
	"corrupted": "fld9",
	"country": "location_country",
	"data_notroutes": "fld32",
	"data_routes": "fld31",
	"definitions": "fld50",
	"delegate-for": "fld5",
	"dict": "fld87",
	"dkimresult": "fld65",
	"duration": "duration_string",
	"elapsed": "duration_string",
	"engine": "fld49",
	"evt": "action",
	"expires": "fld3",
	"file": "filename",
	"from": "from",
	"guid": "fld14",
	"hdr_mid": "id",
	"header-size": "bytes",
	"header.from": "fld40",
	"helo": "fld32",
	"hops-ip": "fld61",
	"hops_active": "fld26",
	"host": "hostname",
	"id": "id",
	"install_dir": "directory",
	"instance": "fld90",
	"ip": "hostip",
	"ksurl": "fld7",
	"lint": "fld33",
	"lip": "fld24",
	"m": "mail_id",
	"malwarescore": "fld41",
	"maxfd": "fld91",
	"method": "fld37",
	"mime": "content_type",
	"mlxlogscore": "fld95",
	"mlxscore": "fld94",
	"mod": "agent",
	"module": "event_source",
	"msg": "msg",
	"msgs": "fld76",
	"notroutes": "fld18",
	"num_domains": "fld53",
	"num_domains_to_lookup": "fld40",
	"oext": "fld8",
	"omime": "fld7",
	"perlwait": "fld27",
	"phishscore": "fld42",
	"pid": "process_id",
	"policy": "fld48",
	"policy_cache_entries": "fld6",
	"profile": "fld52",
	"prot": "fld25",
	"protected": "fld10",
	"publickey_cache_entries": "fld6",
	"qid": "fld15",
	"qname": "fld3",
	"qtype": "fld4",
	"query": "fld38",
	"r": "event_counter",
	"rcpt": "to",
	"rcpt_notroutes": "fld29",
	"rcpt_routes": "fld28",
	"rcpts": "fld59",
	"realm": "fld61",
	"reason": "fld47",
	"record": "fld39",
	"release": "fld92",
	"resolve": "fld28",
	"result": "result",
	"result_detail": "fld74",
	"result_record": "fld2",
	"reverse": "fld13",
	"rewritten": "fld17",
	"routes": "fld4",
	"rule": "rulename",
	"s": "sessionid",
	"scancount": "fld18",
	"score": "fld39",
	"server": "saddr",
	"sha256": "checksum",
	"sig": "fld60",
	"signatures": "fld94",
	"size": "bytes",
	"smtp.mailfrom": "fld44",
	"spamscore": "reputation_num",
	"spfresult": "fld68",
	"subject": "subject",
	"submsgadjust": "fld53",
	"suborg": "fld22",
	"suspectscore": "fld41",
	"sys": "fld1",
	"tests": "fld52",
	"threshold": "fld11",
	"tls": "fld60",
	"tls_version": "fld84",
	"type": "fld1",
	"uid": "uid",
	"user": "username",
	"value": "context",
	"vendor": "fld36",
	"verified": "fld3",
	"verify": "fld57",
	"version": "version",
	"virtual": "fld11",
	"virusname": "threat_name",
	"x": "sessionid1",
}, processor_chain([
	dup17,
	dup36,
]));

var msg177 = msg("filter_instance1", part170);

var select59 = linear_select([
	msg172,
	msg173,
	msg174,
	msg175,
	msg176,
	msg177,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"access_load": msg53,
		"access_refresh": select21,
		"access_run": select20,
		"action_checksubmsg": msg170,
		"av_init": msg45,
		"av_load": msg46,
		"av_refresh": msg44,
		"av_run": select19,
		"batv_run": select25,
		"cvt_convert": select27,
		"cvt_detect": msg82,
		"cvtd": select28,
		"cvtd_encrypted": msg85,
		"dkimv_run": select54,
		"dkimv_type": select55,
		"dmarc_run": select56,
		"dmarc_type": msg167,
		"err": msg154,
		"filter": select29,
		"filter_instance1": select59,
		"http_listen": msg89,
		"info": select50,
		"mail_attachment": select3,
		"mail_continue-system-sendmail": msg18,
		"mail_env_from": select5,
		"mail_env_rcpt": select2,
		"mail_helo": select6,
		"mail_msg": select4,
		"mail_release": msg19,
		"milter_listen": msg91,
		"mltr": msg90,
		"note": select51,
		"pdr_run": msg159,
		"pdr_ttl": msg160,
		"perl_clone": msg78,
		"queued-VoltageEncrypt": select36,
		"queued-aglife": msg158,
		"queued-alert": select35,
		"queued-default": select37,
		"queued-eurort": select39,
		"queued-reinject": select38,
		"queued-vdedc2v5": select40,
		"regulation_init": msg54,
		"regulation_refresh": msg55,
		"rest_oauth": msg171,
		"rprt": msg153,
		"send": select31,
		"sendmail": select46,
		"service_refresh": msg77,
		"session_connect": select12,
		"session_data": select8,
		"session_disconnect": select16,
		"session_dispose": select15,
		"session_headers": select10,
		"session_judge": select11,
		"session_resolve": select13,
		"session_store": select9,
		"session_throttle": select14,
		"sm-msp-queue": select41,
		"smtpsrv": msg95,
		"smtpsrv_listen": msg93,
		"smtpsrv_load": msg92,
		"smtpsrv_run": msg94,
		"soap_listen": msg88,
		"spam_init": msg66,
		"spam_load": msg67,
		"spam_refresh": msg65,
		"spam_run": select24,
		"spf_run": select57,
		"warn": select52,
		"zerohour_init": msg73,
		"zerohour_load": msg74,
		"zerohour_refresh": msg72,
		"zerohour_run": select26,
	}),
]);

var part171 = // "Pattern{Constant('info'), Field(p0,false)}"
match("HEADER#0:0024/1_0", "nwparser.p0", "info%{p0}");

var part172 = // "Pattern{Constant('rprt'), Field(p0,false)}"
match("HEADER#0:0024/1_1", "nwparser.p0", "rprt%{p0}");

var part173 = // "Pattern{Constant('warn'), Field(p0,false)}"
match("HEADER#0:0024/1_2", "nwparser.p0", "warn%{p0}");

var part174 = // "Pattern{Constant('err'), Field(p0,false)}"
match("HEADER#0:0024/1_3", "nwparser.p0", "err%{p0}");

var part175 = // "Pattern{Constant('note'), Field(p0,false)}"
match("HEADER#0:0024/1_4", "nwparser.p0", "note%{p0}");

var part176 = // "Pattern{Field(hostip,true), Constant(' sampling='), Field(fld19,false)}"
match("MESSAGE#11:mail_env_from:ofrom/1_0", "nwparser.p0", "%{hostip->} sampling=%{fld19}");

var part177 = // "Pattern{Field(hostip,false)}"
match_copy("MESSAGE#11:mail_env_from:ofrom/1_1", "nwparser.p0", "hostip");

var part178 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' '), Field(p0,false)}"
match("MESSAGE#25:session_judge/0", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} %{p0}");

var part179 = // "Pattern{Constant('attachment='), Field(fld58,true), Constant(' file='), Field(fld1,true), Constant(' mod='), Field(p0,false)}"
match("MESSAGE#25:session_judge/1_0", "nwparser.p0", "attachment=%{fld58->} file=%{fld1->} mod=%{p0}");

var part180 = // "Pattern{Constant('mod='), Field(p0,false)}"
match("MESSAGE#25:session_judge/1_1", "nwparser.p0", "mod=%{p0}");

var part181 = // "Pattern{Constant('vendor='), Field(fld36,true), Constant(' version="'), Field(component_version,false), Constant('" duration='), Field(p0,false)}"
match("MESSAGE#39:av_run:02/1_1", "nwparser.p0", "vendor=%{fld36->} version=\"%{component_version}\" duration=%{p0}");

var part182 = // "Pattern{Field(duration_string,false)}"
match_copy("MESSAGE#39:av_run:02/2", "nwparser.p0", "duration_string");

var part183 = // "Pattern{Constant('['), Field(daddr,false), Constant('] ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_0", "nwparser.p0", "[%{daddr}] [%{daddr}],%{p0}");

var part184 = // "Pattern{Constant('['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_1", "nwparser.p0", "[%{daddr}],%{p0}");

var part185 = // "Pattern{Field(dhost,true), Constant(' ['), Field(daddr,false), Constant('],'), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_2", "nwparser.p0", "%{dhost->} [%{daddr}],%{p0}");

var part186 = // "Pattern{Field(dhost,false), Constant(','), Field(p0,false)}"
match("MESSAGE#98:queued-alert/3_3", "nwparser.p0", "%{dhost},%{p0}");

var part187 = // "Pattern{Field(,false), Constant('dsn='), Field(resultcode,false), Constant(', stat='), Field(info,false)}"
match("MESSAGE#98:queued-alert/4", "nwparser.p0", "%{}dsn=%{resultcode}, stat=%{info}");

var part188 = // "Pattern{Constant('['), Field(daddr,false), Constant(']')}"
match("MESSAGE#99:queued-alert:01/1_1", "nwparser.p0", "[%{daddr}]");

var part189 = // "Pattern{Field(dhost,true), Constant(' ['), Field(daddr,false), Constant(']')}"
match("MESSAGE#99:queued-alert:01/1_2", "nwparser.p0", "%{dhost->} [%{daddr}]");

var part190 = // "Pattern{Field(dhost,false)}"
match_copy("MESSAGE#99:queued-alert:01/1_3", "nwparser.p0", "dhost");

var part191 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: STARTTLS='), Field(fld1,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#100:queued-alert:02/0", "nwparser.payload", "%{agent}[%{process_id}]: STARTTLS=%{fld1}, relay=%{p0}");

var part192 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld51,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#101:queued-VoltageEncrypt/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld51}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, pri=%{fld23}, relay=%{p0}");

var part193 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld57,false), Constant(', nrcpts='), Field(fld58,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(fld69,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#120:queued-VoltageEncrypt:01/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld57}, nrcpts=%{fld58}, msgid=%{id}, proto=%{protocol}, daemon=%{fld69}, relay=%{p0}");

var part194 = // "Pattern{Constant('['), Field(daddr,false), Constant('] ['), Field(daddr,false), Constant(']')}"
match("MESSAGE#120:queued-VoltageEncrypt:01/1_0", "nwparser.p0", "[%{daddr}] [%{daddr}]");

var part195 = // "Pattern{Field(,false), Constant('field='), Field(fld2,false), Constant(', status='), Field(info,false)}"
match("MESSAGE#104:queued-default:02/2", "nwparser.p0", "%{}field=%{fld2}, status=%{info}");

var part196 = // "Pattern{Field(,false), Constant('version='), Field(fld55,false), Constant(', verify='), Field(fld57,false), Constant(', cipher='), Field(fld58,false), Constant(', bits='), Field(fld59,false)}"
match("MESSAGE#105:queued-default:03/2", "nwparser.p0", "%{}version=%{fld55}, verify=%{fld57}, cipher=%{fld58}, bits=%{fld59}");

var part197 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': from='), Field(from,false), Constant(', size='), Field(bytes,false), Constant(', class='), Field(fld57,false), Constant(', nrcpts='), Field(fld58,false), Constant(', msgid='), Field(id,false), Constant(', proto='), Field(protocol,false), Constant(', daemon='), Field(fld69,false), Constant(', tls_verify='), Field(fld70,false), Constant(', auth='), Field(fld71,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#116:queued-eurort:02/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: from=%{from}, size=%{bytes}, class=%{fld57}, nrcpts=%{fld58}, msgid=%{id}, proto=%{protocol}, daemon=%{fld69}, tls_verify=%{fld70}, auth=%{fld71}, relay=%{p0}");

var part198 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': to='), Field(to,false), Constant(', delay='), Field(fld53,false), Constant(', xdelay='), Field(fld54,false), Constant(', mailer='), Field(fld55,false), Constant(', pri='), Field(fld23,false), Constant(', relay='), Field(p0,false)}"
match("MESSAGE#126:sendmail/0", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: to=%{to}, delay=%{fld53}, xdelay=%{fld54}, mailer=%{fld55}, pri=%{fld23}, relay=%{p0}");

var select60 = linear_select([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]);

var select61 = linear_select([
	dup14,
	dup15,
]);

var select62 = linear_select([
	dup19,
	dup20,
]);

var part199 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' vendor='), Field(fld36,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,true), Constant(' signatures='), Field(fld94,false)}"
match("MESSAGE#43:av_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} vendor=%{fld36->} engine=%{fld49->} definitions=%{fld50->} signatures=%{fld94}", processor_chain([
	dup26,
	dup9,
]));

var part200 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' m='), Field(mail_id,true), Constant(' x='), Field(sessionid1,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#48:access_run:03", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} m=%{mail_id->} x=%{sessionid1->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var part201 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' s='), Field(sessionid,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' rule='), Field(rulename,true), Constant(' duration='), Field(duration_string,false)}"
match("MESSAGE#49:access_run:01", "nwparser.payload", "%{fld0->} %{severity->} s=%{sessionid->} mod=%{agent->} cmd=%{obj_type->} rule=%{rulename->} duration=%{duration_string}", processor_chain([
	dup17,
	dup9,
]));

var part202 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' action='), Field(action,true), Constant(' dict='), Field(fld37,true), Constant(' file='), Field(filename,false)}"
match("MESSAGE#51:access_refresh:01", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} action=%{action->} dict=%{fld37->} file=%{filename}", processor_chain([
	dup17,
	dup9,
]));

var part203 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,false)}"
match("MESSAGE#52:access_load", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5}", processor_chain([
	dup17,
	dup9,
]));

var part204 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' engine='), Field(fld49,true), Constant(' definitions='), Field(fld50,false)}"
match("MESSAGE#64:spam_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} engine=%{fld49->} definitions=%{fld50}", processor_chain([
	dup27,
	dup9,
]));

var part205 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' type='), Field(fld6,true), Constant(' cmd='), Field(obj_type,true), Constant(' id='), Field(fld5,true), Constant(' version='), Field(fld55,false)}"
match("MESSAGE#71:zerohour_refresh", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} type=%{fld6->} cmd=%{obj_type->} id=%{fld5->} version=%{fld55}", processor_chain([
	dup17,
	dup9,
]));

var part206 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' sig='), Field(fld60,false)}"
match("MESSAGE#82:cvtd:01", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} sig=%{fld60}", processor_chain([
	dup17,
	dup9,
]));

var part207 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,false)}"
match("MESSAGE#83:cvtd", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type}", processor_chain([
	dup17,
	dup9,
]));

var part208 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' mod='), Field(agent,true), Constant(' cmd='), Field(obj_type,true), Constant(' addr='), Field(saddr,false)}"
match("MESSAGE#87:soap_listen", "nwparser.payload", "%{fld0->} %{severity->} mod=%{agent->} cmd=%{obj_type->} addr=%{saddr}", processor_chain([
	dup17,
	dup9,
]));

var select63 = linear_select([
	dup28,
	dup29,
	dup30,
	dup31,
]);

var select64 = linear_select([
	dup40,
	dup33,
	dup34,
	dup35,
]);

var part209 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(fld1,false), Constant(': timeout waiting for input from '), Field(fld11,true), Constant(' during server cmd read')}"
match("MESSAGE#106:queued-default:04", "nwparser.payload", "%{agent}[%{process_id}]: %{fld1}: timeout waiting for input from %{fld11->} during server cmd read", processor_chain([
	dup17,
	dup9,
]));

var part210 = // "Pattern{Field(agent,false), Constant('['), Field(process_id,false), Constant(']: '), Field(event_description,false)}"
match("MESSAGE#113:queued-reinject:06", "nwparser.payload", "%{agent}[%{process_id}]: %{event_description}", processor_chain([
	dup17,
	dup9,
]));

var part211 = // "Pattern{Field(fld0,true), Constant(' '), Field(severity,true), Constant(' pid='), Field(process_id,true), Constant(' '), Field(web_method,true), Constant(' /'), Field(info,false), Constant(': '), Field(resultcode,false)}"
match("MESSAGE#141:info:pid", "nwparser.payload", "%{fld0->} %{severity->} pid=%{process_id->} %{web_method->} /%{info}: %{resultcode}", processor_chain([
	dup17,
	dup9,
]));

var all28 = all_match({
	processors: [
		dup38,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var all29 = all_match({
	processors: [
		dup39,
		dup59,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var all30 = all_match({
	processors: [
		dup37,
		dup58,
		dup41,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var all31 = all_match({
	processors: [
		dup37,
		dup58,
		dup42,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var all32 = all_match({
	processors: [
		dup43,
		dup59,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});

var all33 = all_match({
	processors: [
		dup44,
		dup58,
		dup32,
	],
	on_success: processor_chain([
		dup17,
		dup9,
	]),
});
