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

var dup1 = call({
	dest: "nwparser.messageid",
	fn: STRCAT,
	args: [
		field("msgIdPart1"),
		constant("_"),
		field("msgIdPart2"),
	],
});

var dup2 = match("MESSAGE#0:event_admin/0", "nwparser.payload", "user=%{username->} ui=%{p0}");

var dup3 = match("MESSAGE#0:event_admin/1_0", "nwparser.p0", "%{network_service}(%{saddr}) action=%{p0}");

var dup4 = match("MESSAGE#0:event_admin/1_1", "nwparser.p0", "%{network_service->} action=%{p0}");

var dup5 = match("MESSAGE#0:event_admin/3_0", "nwparser.p0", "\"%{event_description}\"");

var dup6 = match_copy("MESSAGE#0:event_admin/3_1", "nwparser.p0", "event_description");

var dup7 = setc("eventcategory","1401000000");

var dup8 = setf("msg","$MSG");

var dup9 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup10 = setf("hardware_id","hfld1");

var dup11 = setf("id","hfld2");

var dup12 = setf("id1","hfld3");

var dup13 = setf("event_type","msgIdPart1");

var dup14 = setf("category","msgIdPart2");

var dup15 = setf("severity","hseverity");

var dup16 = match("MESSAGE#1:event_pop3/2", "nwparser.p0", "%{action->} status=%{event_state->} msg=%{p0}");

var dup17 = setc("eventcategory","1602000000");

var dup18 = match("MESSAGE#5:event_smtp:01/0", "nwparser.payload", "user=%{username}ui=%{p0}");

var dup19 = match("MESSAGE#5:event_smtp:01/1_0", "nwparser.p0", "%{network_service}(%{hostip}) action=%{p0}");

var dup20 = match("MESSAGE#5:event_smtp:01/1_1", "nwparser.p0", "%{network_service}action=%{p0}");

var dup21 = match("MESSAGE#5:event_smtp:01/2", "nwparser.p0", "%{action}status=%{event_state}session_id=%{p0}");

var dup22 = match("MESSAGE#5:event_smtp:01/3_0", "nwparser.p0", "\"%{sessionid}\"msg=\"STARTTLS=%{p0}");

var dup23 = match("MESSAGE#5:event_smtp:01/3_1", "nwparser.p0", "%{sessionid}msg=\"STARTTLS=%{p0}");

var dup24 = match("MESSAGE#16:event_smtp/3_0", "nwparser.p0", "\"%{sessionid}\" msg=%{p0}");

var dup25 = match("MESSAGE#16:event_smtp/3_1", "nwparser.p0", "%{sessionid->} msg=%{p0}");

var dup26 = match("MESSAGE#20:virus/0", "nwparser.payload", "from=%{p0}");

var dup27 = match("MESSAGE#20:virus/1_0", "nwparser.p0", "\"%{from}\" to=%{p0}");

var dup28 = match("MESSAGE#20:virus/1_1", "nwparser.p0", "%{from->} to=%{p0}");

var dup29 = match("MESSAGE#20:virus/2_0", "nwparser.p0", "\"%{to}\" src=%{p0}");

var dup30 = match("MESSAGE#20:virus/2_1", "nwparser.p0", "%{to->} src=%{p0}");

var dup31 = match("MESSAGE#20:virus/3_0", "nwparser.p0", "\"%{saddr}\" session_id=%{p0}");

var dup32 = match("MESSAGE#20:virus/3_1", "nwparser.p0", "%{saddr->} session_id=%{p0}");

var dup33 = setc("eventcategory","1003010000");

var dup34 = setf("event_type","messageid");

var dup35 = match("MESSAGE#23:statistics/0", "nwparser.payload", "session_id=%{p0}");

var dup36 = match("MESSAGE#23:statistics/1_0", "nwparser.p0", "\"%{sessionid}\" from=%{p0}");

var dup37 = match("MESSAGE#23:statistics/1_1", "nwparser.p0", "%{sessionid->} from=%{p0}");

var dup38 = match("MESSAGE#23:statistics/2_0", "nwparser.p0", "\"%{from}\" mailer=%{p0}");

var dup39 = match("MESSAGE#23:statistics/2_1", "nwparser.p0", "%{from->} mailer=%{p0}");

var dup40 = match("MESSAGE#23:statistics/3_0", "nwparser.p0", "\"%{agent}\" client_name=\"%{p0}");

var dup41 = match("MESSAGE#23:statistics/3_1", "nwparser.p0", "%{agent->} client_name=\"%{p0}");

var dup42 = match("MESSAGE#23:statistics/4_0", "nwparser.p0", "%{fqdn->} [%{saddr}] (%{info})\"%{p0}");

var dup43 = match("MESSAGE#23:statistics/4_1", "nwparser.p0", "%{fqdn->} [%{saddr}]\"%{p0}");

var dup44 = match("MESSAGE#23:statistics/4_2", "nwparser.p0", "%{saddr}\"%{p0}");

var dup45 = match("MESSAGE#23:statistics/6_0", "nwparser.p0", "\"%{context}\" to=%{p0}");

var dup46 = match("MESSAGE#23:statistics/6_1", "nwparser.p0", "%{context->} to=%{p0}");

var dup47 = match("MESSAGE#23:statistics/7_0", "nwparser.p0", "\"%{to}\" direction=%{p0}");

var dup48 = match("MESSAGE#23:statistics/7_1", "nwparser.p0", "%{to->} direction=%{p0}");

var dup49 = match("MESSAGE#23:statistics/8_0", "nwparser.p0", "\"%{direction}\" message_length=%{p0}");

var dup50 = match("MESSAGE#23:statistics/8_1", "nwparser.p0", "%{direction->} message_length=%{p0}");

var dup51 = match("MESSAGE#23:statistics/9", "nwparser.p0", "%{fld4->} virus=%{p0}");

var dup52 = match("MESSAGE#23:statistics/10_0", "nwparser.p0", "\"%{virusname}\" disposition=%{p0}");

var dup53 = match("MESSAGE#23:statistics/10_1", "nwparser.p0", "%{virusname->} disposition=%{p0}");

var dup54 = match("MESSAGE#23:statistics/11_0", "nwparser.p0", "\"%{disposition}\" classifier=%{p0}");

var dup55 = match("MESSAGE#23:statistics/11_1", "nwparser.p0", "%{disposition->} classifier=%{p0}");

var dup56 = match("MESSAGE#23:statistics/12_0", "nwparser.p0", "\"%{filter}\" subject=%{p0}");

var dup57 = match("MESSAGE#23:statistics/12_1", "nwparser.p0", "%{filter->} subject=%{p0}");

var dup58 = match("MESSAGE#23:statistics/13_0", "nwparser.p0", "\"%{subject}\"");

var dup59 = match_copy("MESSAGE#23:statistics/13_1", "nwparser.p0", "subject");

var dup60 = setc("eventcategory","1207000000");

var dup61 = match("MESSAGE#24:statistics:01/5", "nwparser.p0", "%{}resolved=%{p0}");

var dup62 = setc("eventcategory","1207040000");

var dup63 = linear_select([
	dup3,
	dup4,
]);

var dup64 = linear_select([
	dup5,
	dup6,
]);

var dup65 = linear_select([
	dup19,
	dup20,
]);

var dup66 = linear_select([
	dup22,
	dup23,
]);

var dup67 = linear_select([
	dup3,
	dup20,
]);

var dup68 = linear_select([
	dup24,
	dup25,
]);

var dup69 = linear_select([
	dup27,
	dup28,
]);

var dup70 = linear_select([
	dup29,
	dup30,
]);

var dup71 = linear_select([
	dup36,
	dup37,
]);

var dup72 = linear_select([
	dup38,
	dup39,
]);

var dup73 = linear_select([
	dup40,
	dup41,
]);

var dup74 = linear_select([
	dup42,
	dup43,
	dup44,
]);

var dup75 = linear_select([
	dup45,
	dup46,
]);

var dup76 = linear_select([
	dup47,
	dup48,
]);

var dup77 = linear_select([
	dup49,
	dup50,
]);

var dup78 = linear_select([
	dup52,
	dup53,
]);

var dup79 = linear_select([
	dup54,
	dup55,
]);

var dup80 = linear_select([
	dup56,
	dup57,
]);

var dup81 = linear_select([
	dup58,
	dup59,
]);

var dup82 = all_match({
	processors: [
		dup2,
		dup63,
		dup16,
		dup64,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "date=%{hdate->} time=%{htime->} device_id=%{hfld1->} log_id=%{hfld2->} log_part=%{hfld3->} type=%{msgIdPart1->} subtype=%{msgIdPart2->} pri=%{hseverity->} %{payload}", processor_chain([
	setc("header_id","0001"),
	dup1,
]));

var hdr2 = match("HEADER#1:0002", "message", "date=%{hdate->} time=%{htime->} device_id=%{hfld1->} log_id=%{hfld2->} log_part=%{hfld3->} type=%{messageid->} pri=%{hseverity->} %{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr3 = match("HEADER#2:0003", "message", "date=%{hdate->} time=%{htime->} device_id=%{hfld1->} log_id=%{hfld2->} type=%{msgIdPart1->} subtype=%{msgIdPart2->} pri=%{hseverity->} %{payload}", processor_chain([
	setc("header_id","0003"),
	dup1,
]));

var hdr4 = match("HEADER#3:0004", "message", "date=%{hdate->} time=%{htime->} device_id=%{hfld1->} log_id=%{hfld2->} type=%{messageid->} pri=%{hseverity->} %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
]);

var part1 = match("MESSAGE#0:event_admin/2", "nwparser.p0", "%{action->} status=%{event_state->} reason=%{result->} msg=%{p0}");

var all1 = all_match({
	processors: [
		dup2,
		dup63,
		part1,
		dup64,
	],
	on_success: processor_chain([
		dup7,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg1 = msg("event_admin", all1);

var msg2 = msg("event_pop3", dup82);

var all2 = all_match({
	processors: [
		dup2,
		dup63,
		dup16,
		dup64,
	],
	on_success: processor_chain([
		dup7,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg3 = msg("event_webmail", all2);

var msg4 = msg("event_system", dup82);

var msg5 = msg("event_imap", dup82);

var part2 = match("MESSAGE#5:event_smtp:01/4", "nwparser.p0", "%{fld1}, relay=%{p0}");

var part3 = match("MESSAGE#5:event_smtp:01/5_0", "nwparser.p0", "%{shost}[%{saddr}], version=%{p0}");

var part4 = match("MESSAGE#5:event_smtp:01/5_1", "nwparser.p0", "%{shost}, version=%{p0}");

var select2 = linear_select([
	part3,
	part4,
]);

var part5 = match("MESSAGE#5:event_smtp:01/6", "nwparser.p0", "%{version}, verify=%{fld2}, cipher=%{s_cipher}, bits=%{fld3}\"");

var all3 = all_match({
	processors: [
		dup18,
		dup65,
		dup21,
		dup66,
		part2,
		select2,
		part5,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg6 = msg("event_smtp:01", all3);

var part6 = match("MESSAGE#6:event_smtp:02/4", "nwparser.p0", "%{fld1}, cert-subject=%{cert_subject}, cert-issuer=%{fld2}, verifymsg=%{fld3}\"");

var all4 = all_match({
	processors: [
		dup18,
		dup65,
		dup21,
		dup66,
		part6,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg7 = msg("event_smtp:02", all4);

var part7 = match("MESSAGE#7:event_smtp:03/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"to=\u003c\u003c%{to}>, delay=%{fld1}, xdelay=%{fld2}, mailer=%{protocol}, pri=%{fld3}, relay=%{shost}[%{saddr}], dsn=%{fld4}, stat=%{fld5}\"");

var all5 = all_match({
	processors: [
		dup18,
		dup65,
		part7,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg8 = msg("event_smtp:03", all5);

var part8 = match("MESSAGE#8:event_smtp:04/0", "nwparser.payload", "user=%{username}ui=%{network_service}action=%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"from=\u003c\u003c%{from}>, size=%{bytes}, class=%{fld2}, nrcpts=%{p0}");

var part9 = match("MESSAGE#8:event_smtp:04/1_0", "nwparser.p0", "%{fld3}, msgid=\u003c\u003c%{fld4}>, proto=%{p0}");

var part10 = match("MESSAGE#8:event_smtp:04/1_1", "nwparser.p0", "%{fld3}, proto=%{p0}");

var select3 = linear_select([
	part9,
	part10,
]);

var part11 = match("MESSAGE#8:event_smtp:04/2", "nwparser.p0", "%{protocol}, daemon=%{process}, relay=%{p0}");

var part12 = match("MESSAGE#8:event_smtp:04/3_0", "nwparser.p0", "%{shost}[%{saddr}] (may be forged)\"");

var part13 = match("MESSAGE#8:event_smtp:04/3_1", "nwparser.p0", "%{shost}[%{saddr}]\"");

var part14 = match("MESSAGE#8:event_smtp:04/3_2", "nwparser.p0", "%{shost}\"");

var select4 = linear_select([
	part12,
	part13,
	part14,
]);

var all6 = all_match({
	processors: [
		part8,
		select3,
		part11,
		select4,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg9 = msg("event_smtp:04", all6);

var part15 = match("MESSAGE#9:event_smtp:05/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"Milter: to=\u003c\u003c%{to}>, reject=%{fld1}\"");

var all7 = all_match({
	processors: [
		dup18,
		dup67,
		part15,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg10 = msg("event_smtp:05", all7);

var part16 = match("MESSAGE#10:event_smtp:06/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"timeout waiting for input from%{p0}");

var part17 = match("MESSAGE#10:event_smtp:06/3_0", "nwparser.p0", "[%{saddr}]during server cmd%{p0}");

var part18 = match("MESSAGE#10:event_smtp:06/3_1", "nwparser.p0", "%{saddr}during server cmd%{p0}");

var select5 = linear_select([
	part17,
	part18,
]);

var part19 = match("MESSAGE#10:event_smtp:06/4", "nwparser.p0", "%{fld5}\"");

var all8 = all_match({
	processors: [
		dup18,
		dup65,
		part16,
		select5,
		part19,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg11 = msg("event_smtp:06", all8);

var part20 = match("MESSAGE#11:event_smtp:07/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"collect:%{fld1}timeout on connection from%{shost}, from=\u003c\u003c%{from}>\"");

var all9 = all_match({
	processors: [
		dup18,
		dup67,
		part20,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg12 = msg("event_smtp:07", all9);

var part21 = match("MESSAGE#12:event_smtp:08/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"DSN: to \u003c\u003c%{to}>; reason:%{result}; sessionid:%{fld5}\"");

var all10 = all_match({
	processors: [
		dup18,
		dup67,
		part21,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg13 = msg("event_smtp:08", all10);

var part22 = match("MESSAGE#13:event_smtp:09/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"lost input channel from%{shost}[%{saddr}] (may be forged) to SMTP_MTA after rcpt\"");

var all11 = all_match({
	processors: [
		dup18,
		dup65,
		part22,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg14 = msg("event_smtp:09", all11);

var part23 = match("MESSAGE#14:event_smtp:10/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" msg=\"%{shost}[%{saddr}]: possible SMTP attack: command=%{fld1}, count=%{dclass_counter1}\"");

var all12 = all_match({
	processors: [
		dup18,
		dup65,
		part23,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("dclass_counter1_string","count"),
	]),
});

var msg15 = msg("event_smtp:10", all12);

var part24 = match("MESSAGE#15:event_smtp:11/2", "nwparser.p0", "%{action}status=%{event_state}session_id=\"%{sessionid}\" log_part=%{id1->} msg=\"to=\u003c\u003c%{to}, delay=%{p0}");

var part25 = match("MESSAGE#15:event_smtp:11/3_0", "nwparser.p0", "%{fld1}, xdelay=%{fld2}, mailer=%{protocol}, pri=%{fld3}, relay=%{shost}\"");

var part26 = match("MESSAGE#15:event_smtp:11/3_1", "nwparser.p0", "%{fld1}, xdelay=%{fld2}, mailer=%{protocol}, pri=%{fld3}\"");

var part27 = match("MESSAGE#15:event_smtp:11/3_2", "nwparser.p0", "%{fld1}, xdelay=%{fld2}, mailer=%{protocol}\"");

var part28 = match("MESSAGE#15:event_smtp:11/3_3", "nwparser.p0", "%{fld1}\"");

var select6 = linear_select([
	part25,
	part26,
	part27,
	part28,
]);

var all13 = all_match({
	processors: [
		dup18,
		dup65,
		part24,
		select6,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg16 = msg("event_smtp:11", all13);

var part29 = match("MESSAGE#16:event_smtp/2", "nwparser.p0", "%{action->} status=%{event_state->} session_id=%{p0}");

var all14 = all_match({
	processors: [
		dup2,
		dup63,
		part29,
		dup68,
		dup64,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg17 = msg("event_smtp", all14);

var part30 = tagval("MESSAGE#17:event_smtp:12", "nwparser.payload", tvm, {
	"action": "action",
	"log_part": "id1",
	"msg": "info",
	"session_id": "sessionid",
	"status": "event_state",
	"ui": "network_service",
	"user": "username",
}, processor_chain([
	dup17,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg18 = msg("event_smtp:12", part30);

var select7 = linear_select([
	msg6,
	msg7,
	msg8,
	msg9,
	msg10,
	msg11,
	msg12,
	msg13,
	msg14,
	msg15,
	msg16,
	msg17,
	msg18,
]);

var part31 = match("MESSAGE#18:event_update/0", "nwparser.payload", "msg=%{p0}");

var all15 = all_match({
	processors: [
		part31,
		dup64,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg19 = msg("event_update", all15);

var part32 = match("MESSAGE#19:event_config/1_0", "nwparser.p0", "%{network_service}(%{saddr}) module=%{p0}");

var part33 = match("MESSAGE#19:event_config/1_1", "nwparser.p0", "%{network_service->} module=%{p0}");

var select8 = linear_select([
	part32,
	part33,
]);

var part34 = match("MESSAGE#19:event_config/2", "nwparser.p0", "%{fld1->} submodule=%{fld2->} msg=%{p0}");

var all16 = all_match({
	processors: [
		dup2,
		select8,
		part34,
		dup64,
	],
	on_success: processor_chain([
		setc("eventcategory","1701000000"),
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});

var msg20 = msg("event_config", all16);

var select9 = linear_select([
	dup31,
	dup32,
]);

var all17 = all_match({
	processors: [
		dup26,
		dup69,
		dup70,
		select9,
		dup68,
		dup64,
	],
	on_success: processor_chain([
		dup33,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg21 = msg("virus", all17);

var part35 = match("MESSAGE#21:virus_infected/2_0", "nwparser.p0", "\"%{to}\" client_name=\"%{p0}");

var part36 = match("MESSAGE#21:virus_infected/2_1", "nwparser.p0", "%{to->} client_name=\"%{p0}");

var select10 = linear_select([
	part35,
	part36,
]);

var part37 = match("MESSAGE#21:virus_infected/3", "nwparser.p0", "%{fqdn}\" client_ip=\"%{saddr}\" session_id=%{p0}");

var all18 = all_match({
	processors: [
		dup26,
		dup69,
		select10,
		part37,
		dup68,
		dup64,
	],
	on_success: processor_chain([
		dup33,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup15,
	]),
});

var msg22 = msg("virus_infected", all18);

var part38 = match("MESSAGE#22:virus_file-signature/0_0", "nwparser.payload", "from=\"%{from}\" to=%{p0}");

var part39 = match("MESSAGE#22:virus_file-signature/0_1", "nwparser.payload", "%{from->} to=%{p0}");

var select11 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#22:virus_file-signature/2_0", "nwparser.p0", "\"%{sdomain->} [%{saddr}]\" session_id=%{p0}");

var part41 = match("MESSAGE#22:virus_file-signature/2_1", "nwparser.p0", "%{sdomain->} [%{saddr}] session_id=%{p0}");

var part42 = match("MESSAGE#22:virus_file-signature/2_2", "nwparser.p0", "\"[%{saddr}]\" session_id=%{p0}");

var part43 = match("MESSAGE#22:virus_file-signature/2_3", "nwparser.p0", "[%{saddr}] session_id=%{p0}");

var select12 = linear_select([
	part40,
	part41,
	part42,
	part43,
	dup31,
	dup32,
]);

var part44 = match("MESSAGE#22:virus_file-signature/4_0", "nwparser.p0", "\"Attachment file (%{filename}) has sha1 hash value: %{checksum}\"");

var select13 = linear_select([
	part44,
	dup5,
	dup6,
]);

var all19 = all_match({
	processors: [
		select11,
		dup70,
		select12,
		dup68,
		select13,
	],
	on_success: processor_chain([
		dup33,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg23 = msg("virus_file-signature", all19);

var part45 = match("MESSAGE#23:statistics/5", "nwparser.p0", "%{}MSISDN=%{fld3->} resolved=%{p0}");

var all20 = all_match({
	processors: [
		dup35,
		dup71,
		dup72,
		dup73,
		dup74,
		part45,
		dup75,
		dup76,
		dup77,
		dup51,
		dup78,
		dup79,
		dup80,
		dup81,
	],
	on_success: processor_chain([
		dup60,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg24 = msg("statistics", all20);

var all21 = all_match({
	processors: [
		dup35,
		dup71,
		dup72,
		dup73,
		dup74,
		dup61,
		dup75,
		dup76,
		dup77,
		dup51,
		dup78,
		dup79,
		dup80,
		dup81,
	],
	on_success: processor_chain([
		dup60,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg25 = msg("statistics:01", all21);

var part46 = match("MESSAGE#25:statistics:02/4_0", "nwparser.p0", "\"%{direction}\" subject=%{p0}");

var part47 = match("MESSAGE#25:statistics:02/4_1", "nwparser.p0", "%{direction->} subject=%{p0}");

var select14 = linear_select([
	part46,
	part47,
]);

var part48 = match("MESSAGE#25:statistics:02/5_0", "nwparser.p0", "\"%{subject}\" classifier=%{p0}");

var part49 = match("MESSAGE#25:statistics:02/5_1", "nwparser.p0", "%{subject->} classifier=%{p0}");

var select15 = linear_select([
	part48,
	part49,
]);

var part50 = match("MESSAGE#25:statistics:02/6_0", "nwparser.p0", "\"%{filter}\" disposition=%{p0}");

var part51 = match("MESSAGE#25:statistics:02/6_1", "nwparser.p0", "%{filter->} disposition=%{p0}");

var select16 = linear_select([
	part50,
	part51,
]);

var part52 = match("MESSAGE#25:statistics:02/7_0", "nwparser.p0", "\"%{disposition}\" client_name=\"%{p0}");

var part53 = match("MESSAGE#25:statistics:02/7_1", "nwparser.p0", "%{disposition->} client_name=\"%{p0}");

var select17 = linear_select([
	part52,
	part53,
]);

var part54 = match("MESSAGE#25:statistics:02/10_0", "nwparser.p0", "\"%{context}\" virus=%{p0}");

var part55 = match("MESSAGE#25:statistics:02/10_1", "nwparser.p0", "%{context->} virus=%{p0}");

var select18 = linear_select([
	part54,
	part55,
]);

var part56 = match("MESSAGE#25:statistics:02/11_0", "nwparser.p0", "\"%{virusname}\" message_length=%{p0}");

var part57 = match("MESSAGE#25:statistics:02/11_1", "nwparser.p0", "%{virusname->} message_length=%{p0}");

var select19 = linear_select([
	part56,
	part57,
]);

var part58 = match_copy("MESSAGE#25:statistics:02/12", "nwparser.p0", "fld4");

var all22 = all_match({
	processors: [
		dup35,
		dup71,
		dup69,
		dup76,
		select14,
		select15,
		select16,
		select17,
		dup74,
		dup61,
		select18,
		select19,
		part58,
	],
	on_success: processor_chain([
		dup60,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg26 = msg("statistics:02", all22);

var part59 = match("MESSAGE#26:statistics:03/0", "nwparser.payload", "session_id=\"%{sessionid}\" client_name=\"%{p0}");

var part60 = match("MESSAGE#26:statistics:03/1_0", "nwparser.p0", "%{fqdn}[%{saddr}] (may be forged)\"%{p0}");

var part61 = match("MESSAGE#26:statistics:03/1_1", "nwparser.p0", "%{fqdn}[%{saddr}]\"%{p0}");

var part62 = match("MESSAGE#26:statistics:03/1_2", "nwparser.p0", "[%{saddr}]\"%{p0}");

var select20 = linear_select([
	part60,
	part61,
	part62,
]);

var part63 = match("MESSAGE#26:statistics:03/2", "nwparser.p0", "dst_ip=\"%{daddr}\" from=\"%{from}\" to=\"%{to}\"%{p0}");

var part64 = match("MESSAGE#26:statistics:03/3_0", "nwparser.p0", " polid=\"%{fld5}\" domain=\"%{domain}\" subject=\"%{subject}\" mailer=\"%{agent}\" resolved=\"%{context}\"%{p0}");

var part65 = match_copy("MESSAGE#26:statistics:03/3_1", "nwparser.p0", "p0");

var select21 = linear_select([
	part64,
	part65,
]);

var part66 = match("MESSAGE#26:statistics:03/4", "nwparser.p0", "%{}direction=\"%{direction}\" virus=\"%{virusname}\" disposition=\"%{disposition}\" classifier=\"%{filter}\" message_length=%{fld4}");

var all23 = all_match({
	processors: [
		part59,
		select20,
		part63,
		select21,
		part66,
	],
	on_success: processor_chain([
		dup60,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg27 = msg("statistics:03", all23);

var part67 = match("MESSAGE#27:statistics:04/1_0", "nwparser.p0", "\"%{sessionid}\" client_name=%{p0}");

var part68 = match("MESSAGE#27:statistics:04/1_1", "nwparser.p0", "%{sessionid->} client_name=%{p0}");

var select22 = linear_select([
	part67,
	part68,
]);

var part69 = match("MESSAGE#27:statistics:04/2_0", "nwparser.p0", "\"%{fqdn}[%{saddr}]\"dst_ip=%{p0}");

var part70 = match("MESSAGE#27:statistics:04/2_1", "nwparser.p0", "%{fqdn}[%{saddr}]dst_ip=%{p0}");

var part71 = match("MESSAGE#27:statistics:04/2_2", "nwparser.p0", "\"[%{saddr}]\"dst_ip=%{p0}");

var part72 = match("MESSAGE#27:statistics:04/2_3", "nwparser.p0", "[%{saddr}]dst_ip=%{p0}");

var part73 = match("MESSAGE#27:statistics:04/2_4", "nwparser.p0", "\"%{saddr}\"dst_ip=%{p0}");

var part74 = match("MESSAGE#27:statistics:04/2_5", "nwparser.p0", "%{saddr}dst_ip=%{p0}");

var select23 = linear_select([
	part69,
	part70,
	part71,
	part72,
	part73,
	part74,
]);

var part75 = match("MESSAGE#27:statistics:04/3_0", "nwparser.p0", "\"%{daddr}\" from=%{p0}");

var part76 = match("MESSAGE#27:statistics:04/3_1", "nwparser.p0", "%{daddr->} from=%{p0}");

var select24 = linear_select([
	part75,
	part76,
]);

var part77 = match("MESSAGE#27:statistics:04/4_0", "nwparser.p0", "\"%{from}\" hfrom=%{p0}");

var part78 = match("MESSAGE#27:statistics:04/4_1", "nwparser.p0", "%{from->} hfrom=%{p0}");

var select25 = linear_select([
	part77,
	part78,
]);

var part79 = match("MESSAGE#27:statistics:04/5_0", "nwparser.p0", "\"%{fld3}\" to=%{p0}");

var part80 = match("MESSAGE#27:statistics:04/5_1", "nwparser.p0", "%{fld3->} to=%{p0}");

var select26 = linear_select([
	part79,
	part80,
]);

var part81 = match("MESSAGE#27:statistics:04/6_0", "nwparser.p0", "\"%{to}\" polid=%{p0}");

var part82 = match("MESSAGE#27:statistics:04/6_1", "nwparser.p0", "%{to->} polid=%{p0}");

var select27 = linear_select([
	part81,
	part82,
]);

var part83 = match("MESSAGE#27:statistics:04/7_0", "nwparser.p0", "\"%{fld5}\" domain=%{p0}");

var part84 = match("MESSAGE#27:statistics:04/7_1", "nwparser.p0", "%{fld5->} domain=%{p0}");

var select28 = linear_select([
	part83,
	part84,
]);

var part85 = match("MESSAGE#27:statistics:04/8_0", "nwparser.p0", "\"%{domain}\" subject=%{p0}");

var part86 = match("MESSAGE#27:statistics:04/8_1", "nwparser.p0", "%{domain->} subject=%{p0}");

var select29 = linear_select([
	part85,
	part86,
]);

var part87 = match("MESSAGE#27:statistics:04/9_0", "nwparser.p0", "\"%{subject}\" mailer=%{p0}");

var part88 = match("MESSAGE#27:statistics:04/9_1", "nwparser.p0", "%{subject->} mailer=%{p0}");

var select30 = linear_select([
	part87,
	part88,
]);

var part89 = match("MESSAGE#27:statistics:04/10_0", "nwparser.p0", "\"%{agent}\" resolved=%{p0}");

var part90 = match("MESSAGE#27:statistics:04/10_1", "nwparser.p0", "%{agent->} resolved=%{p0}");

var select31 = linear_select([
	part89,
	part90,
]);

var part91 = match("MESSAGE#27:statistics:04/11_0", "nwparser.p0", "\"%{context}\" direction=%{p0}");

var part92 = match("MESSAGE#27:statistics:04/11_1", "nwparser.p0", "%{context->} direction=%{p0}");

var select32 = linear_select([
	part91,
	part92,
]);

var part93 = match("MESSAGE#27:statistics:04/12_0", "nwparser.p0", "\"%{direction}\" virus=%{p0}");

var part94 = match("MESSAGE#27:statistics:04/12_1", "nwparser.p0", "%{direction->} virus=%{p0}");

var select33 = linear_select([
	part93,
	part94,
]);

var part95 = match("MESSAGE#27:statistics:04/15_0", "nwparser.p0", "\"%{filter}\" message_length=%{p0}");

var part96 = match("MESSAGE#27:statistics:04/15_1", "nwparser.p0", "%{filter->} message_length=%{p0}");

var select34 = linear_select([
	part95,
	part96,
]);

var part97 = match("MESSAGE#27:statistics:04/16_0", "nwparser.p0", "\"%{fld6}\"");

var part98 = match_copy("MESSAGE#27:statistics:04/16_1", "nwparser.p0", "fld6");

var select35 = linear_select([
	part97,
	part98,
]);

var all24 = all_match({
	processors: [
		dup35,
		select22,
		select23,
		select24,
		select25,
		select26,
		select27,
		select28,
		select29,
		select30,
		select31,
		select32,
		select33,
		dup78,
		dup79,
		select34,
		select35,
	],
	on_success: processor_chain([
		dup60,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg28 = msg("statistics:04", all24);

var part99 = tagval("MESSAGE#28:statistics:05", "nwparser.payload", tvm, {
	"classifier": "filter",
	"client_ip": "saddr",
	"client_name": "fqdn",
	"direction": "direction",
	"disposition": "disposition",
	"domain": "domain",
	"dst_ip": "daddr",
	"from": "from",
	"hfrom": "fld3",
	"mailer": "agent",
	"message_length": "fld6",
	"polid": "fld5",
	"resolved": "context",
	"session_id": "sessionid",
	"src_type": "fld7",
	"subject": "subject",
	"to": "to",
	"virus": "virusname",
}, processor_chain([
	dup60,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
	dup34,
	dup15,
]));

var msg29 = msg("statistics:05", part99);

var select36 = linear_select([
	msg24,
	msg25,
	msg26,
	msg27,
	msg28,
	msg29,
]);

var part100 = match("MESSAGE#29:spam/1_0", "nwparser.p0", "\"%{sessionid}\" client_name=\"%{p0}");

var part101 = match("MESSAGE#29:spam/1_1", "nwparser.p0", "%{sessionid->} client_name=\"%{p0}");

var select37 = linear_select([
	part100,
	part101,
]);

var part102 = match("MESSAGE#29:spam/3", "nwparser.p0", "%{}from=%{p0}");

var part103 = match("MESSAGE#29:spam/5_0", "nwparser.p0", "\"%{to}\" subject=%{p0}");

var part104 = match("MESSAGE#29:spam/5_1", "nwparser.p0", "%{to->} subject=%{p0}");

var select38 = linear_select([
	part103,
	part104,
]);

var part105 = match("MESSAGE#29:spam/6_0", "nwparser.p0", "\"%{subject}\" msg=%{p0}");

var part106 = match("MESSAGE#29:spam/6_1", "nwparser.p0", "%{subject->} msg=%{p0}");

var select39 = linear_select([
	part105,
	part106,
]);

var all25 = all_match({
	processors: [
		dup35,
		select37,
		dup74,
		part102,
		dup69,
		select38,
		select39,
		dup64,
	],
	on_success: processor_chain([
		dup62,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg30 = msg("spam", all25);

var part107 = match("MESSAGE#30:spam:04", "nwparser.payload", "session_id=\"%{sessionid}\" client_name=\"%{fqdn->} [%{saddr}] (%{fld2})\" dst_ip=\"%{daddr}\" from=\"%{from}\" to=\"%{to}\" subject=\"%{subject}\" msg=\"%{event_description}\"", processor_chain([
	dup62,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
	dup34,
	dup15,
]));

var msg31 = msg("spam:04", part107);

var part108 = match("MESSAGE#31:spam:03/0", "nwparser.payload", "session_id=\"%{sessionid}\" client_name=%{p0}");

var part109 = match("MESSAGE#31:spam:03/1_0", "nwparser.p0", "\"%{fqdn->} [%{saddr}]\" %{p0}");

var part110 = match("MESSAGE#31:spam:03/1_1", "nwparser.p0", " \"%{fqdn}\" client_ip=\"%{saddr}\"%{p0}");

var select40 = linear_select([
	part109,
	part110,
]);

var part111 = match("MESSAGE#31:spam:03/2", "nwparser.p0", "%{}dst_ip=\"%{daddr}\" from=\"%{from}\" to=\"%{to}\" subject=\"%{subject}\" msg=\"%{event_description}\"");

var all26 = all_match({
	processors: [
		part108,
		select40,
		part111,
	],
	on_success: processor_chain([
		dup62,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg32 = msg("spam:03", all26);

var part112 = match("MESSAGE#32:spam:02", "nwparser.payload", "session_id=\"%{sessionid}\" from=\"%{from}\" to=\"%{to}\" subject=\"%{subject}\" msg=\"%{event_description}\"", processor_chain([
	dup62,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
	dup34,
	dup15,
]));

var msg33 = msg("spam:02", part112);

var part113 = match("MESSAGE#33:spam:01/3_0", "nwparser.p0", "\"%{to}\" msg=%{p0}");

var part114 = match("MESSAGE#33:spam:01/3_1", "nwparser.p0", "%{to->} msg=%{p0}");

var select41 = linear_select([
	part113,
	part114,
]);

var all27 = all_match({
	processors: [
		dup35,
		dup71,
		dup69,
		select41,
		dup64,
	],
	on_success: processor_chain([
		dup62,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup34,
		dup15,
	]),
});

var msg34 = msg("spam:01", all27);

var select42 = linear_select([
	msg30,
	msg31,
	msg32,
	msg33,
	msg34,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"event_admin": msg1,
		"event_config": msg20,
		"event_imap": msg5,
		"event_pop3": msg2,
		"event_smtp": select7,
		"event_system": msg4,
		"event_update": msg19,
		"event_webmail": msg3,
		"spam": select42,
		"statistics": select36,
		"virus": msg21,
		"virus_file-signature": msg23,
		"virus_infected": msg22,
	}),
]);

var part115 = match("MESSAGE#0:event_admin/0", "nwparser.payload", "user=%{username->} ui=%{p0}");

var part116 = match("MESSAGE#0:event_admin/1_0", "nwparser.p0", "%{network_service}(%{saddr}) action=%{p0}");

var part117 = match("MESSAGE#0:event_admin/1_1", "nwparser.p0", "%{network_service->} action=%{p0}");

var part118 = match("MESSAGE#0:event_admin/3_0", "nwparser.p0", "\"%{event_description}\"");

var part119 = match_copy("MESSAGE#0:event_admin/3_1", "nwparser.p0", "event_description");

var part120 = match("MESSAGE#1:event_pop3/2", "nwparser.p0", "%{action->} status=%{event_state->} msg=%{p0}");

var part121 = match("MESSAGE#5:event_smtp:01/0", "nwparser.payload", "user=%{username}ui=%{p0}");

var part122 = match("MESSAGE#5:event_smtp:01/1_0", "nwparser.p0", "%{network_service}(%{hostip}) action=%{p0}");

var part123 = match("MESSAGE#5:event_smtp:01/1_1", "nwparser.p0", "%{network_service}action=%{p0}");

var part124 = match("MESSAGE#5:event_smtp:01/2", "nwparser.p0", "%{action}status=%{event_state}session_id=%{p0}");

var part125 = match("MESSAGE#5:event_smtp:01/3_0", "nwparser.p0", "\"%{sessionid}\"msg=\"STARTTLS=%{p0}");

var part126 = match("MESSAGE#5:event_smtp:01/3_1", "nwparser.p0", "%{sessionid}msg=\"STARTTLS=%{p0}");

var part127 = match("MESSAGE#16:event_smtp/3_0", "nwparser.p0", "\"%{sessionid}\" msg=%{p0}");

var part128 = match("MESSAGE#16:event_smtp/3_1", "nwparser.p0", "%{sessionid->} msg=%{p0}");

var part129 = match("MESSAGE#20:virus/0", "nwparser.payload", "from=%{p0}");

var part130 = match("MESSAGE#20:virus/1_0", "nwparser.p0", "\"%{from}\" to=%{p0}");

var part131 = match("MESSAGE#20:virus/1_1", "nwparser.p0", "%{from->} to=%{p0}");

var part132 = match("MESSAGE#20:virus/2_0", "nwparser.p0", "\"%{to}\" src=%{p0}");

var part133 = match("MESSAGE#20:virus/2_1", "nwparser.p0", "%{to->} src=%{p0}");

var part134 = match("MESSAGE#20:virus/3_0", "nwparser.p0", "\"%{saddr}\" session_id=%{p0}");

var part135 = match("MESSAGE#20:virus/3_1", "nwparser.p0", "%{saddr->} session_id=%{p0}");

var part136 = match("MESSAGE#23:statistics/0", "nwparser.payload", "session_id=%{p0}");

var part137 = match("MESSAGE#23:statistics/1_0", "nwparser.p0", "\"%{sessionid}\" from=%{p0}");

var part138 = match("MESSAGE#23:statistics/1_1", "nwparser.p0", "%{sessionid->} from=%{p0}");

var part139 = match("MESSAGE#23:statistics/2_0", "nwparser.p0", "\"%{from}\" mailer=%{p0}");

var part140 = match("MESSAGE#23:statistics/2_1", "nwparser.p0", "%{from->} mailer=%{p0}");

var part141 = match("MESSAGE#23:statistics/3_0", "nwparser.p0", "\"%{agent}\" client_name=\"%{p0}");

var part142 = match("MESSAGE#23:statistics/3_1", "nwparser.p0", "%{agent->} client_name=\"%{p0}");

var part143 = match("MESSAGE#23:statistics/4_0", "nwparser.p0", "%{fqdn->} [%{saddr}] (%{info})\"%{p0}");

var part144 = match("MESSAGE#23:statistics/4_1", "nwparser.p0", "%{fqdn->} [%{saddr}]\"%{p0}");

var part145 = match("MESSAGE#23:statistics/4_2", "nwparser.p0", "%{saddr}\"%{p0}");

var part146 = match("MESSAGE#23:statistics/6_0", "nwparser.p0", "\"%{context}\" to=%{p0}");

var part147 = match("MESSAGE#23:statistics/6_1", "nwparser.p0", "%{context->} to=%{p0}");

var part148 = match("MESSAGE#23:statistics/7_0", "nwparser.p0", "\"%{to}\" direction=%{p0}");

var part149 = match("MESSAGE#23:statistics/7_1", "nwparser.p0", "%{to->} direction=%{p0}");

var part150 = match("MESSAGE#23:statistics/8_0", "nwparser.p0", "\"%{direction}\" message_length=%{p0}");

var part151 = match("MESSAGE#23:statistics/8_1", "nwparser.p0", "%{direction->} message_length=%{p0}");

var part152 = match("MESSAGE#23:statistics/9", "nwparser.p0", "%{fld4->} virus=%{p0}");

var part153 = match("MESSAGE#23:statistics/10_0", "nwparser.p0", "\"%{virusname}\" disposition=%{p0}");

var part154 = match("MESSAGE#23:statistics/10_1", "nwparser.p0", "%{virusname->} disposition=%{p0}");

var part155 = match("MESSAGE#23:statistics/11_0", "nwparser.p0", "\"%{disposition}\" classifier=%{p0}");

var part156 = match("MESSAGE#23:statistics/11_1", "nwparser.p0", "%{disposition->} classifier=%{p0}");

var part157 = match("MESSAGE#23:statistics/12_0", "nwparser.p0", "\"%{filter}\" subject=%{p0}");

var part158 = match("MESSAGE#23:statistics/12_1", "nwparser.p0", "%{filter->} subject=%{p0}");

var part159 = match("MESSAGE#23:statistics/13_0", "nwparser.p0", "\"%{subject}\"");

var part160 = match_copy("MESSAGE#23:statistics/13_1", "nwparser.p0", "subject");

var part161 = match("MESSAGE#24:statistics:01/5", "nwparser.p0", "%{}resolved=%{p0}");

var select43 = linear_select([
	dup3,
	dup4,
]);

var select44 = linear_select([
	dup5,
	dup6,
]);

var select45 = linear_select([
	dup19,
	dup20,
]);

var select46 = linear_select([
	dup22,
	dup23,
]);

var select47 = linear_select([
	dup3,
	dup20,
]);

var select48 = linear_select([
	dup24,
	dup25,
]);

var select49 = linear_select([
	dup27,
	dup28,
]);

var select50 = linear_select([
	dup29,
	dup30,
]);

var select51 = linear_select([
	dup36,
	dup37,
]);

var select52 = linear_select([
	dup38,
	dup39,
]);

var select53 = linear_select([
	dup40,
	dup41,
]);

var select54 = linear_select([
	dup42,
	dup43,
	dup44,
]);

var select55 = linear_select([
	dup45,
	dup46,
]);

var select56 = linear_select([
	dup47,
	dup48,
]);

var select57 = linear_select([
	dup49,
	dup50,
]);

var select58 = linear_select([
	dup52,
	dup53,
]);

var select59 = linear_select([
	dup54,
	dup55,
]);

var select60 = linear_select([
	dup56,
	dup57,
]);

var select61 = linear_select([
	dup58,
	dup59,
]);

var all28 = all_match({
	processors: [
		dup2,
		dup63,
		dup16,
		dup64,
	],
	on_success: processor_chain([
		dup17,
		dup8,
		dup9,
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
	]),
});
