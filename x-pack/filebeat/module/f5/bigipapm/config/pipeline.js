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

var dup1 = setc("eventcategory","1801030000");

var dup2 = setf("msg","$MSG");

var dup3 = setc("eventcategory","1801020000");

var dup4 = match("MESSAGE#10:01490010/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{p0}");

var dup5 = setc("eventcategory","1801000000");

var dup6 = setc("eventcategory","1801010000");

var dup7 = setc("eventcategory","1502000000");

var dup8 = setc("eventcategory","1805010000");

var dup9 = setc("eventcategory","1803000000");

var dup10 = setc("eventcategory","1803030000");

var dup11 = setc("disposition"," Successful");

var dup12 = setc("dclass_counter1_string"," Logon Attempt");

var dup13 = setc("eventcategory","1204000000");

var dup14 = date_time({
	dest: "event_time",
	args: ["fld20"],
	fmts: [
		[dD,dc("/"),dB,dc("/"),dW,dc(":"),dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup15 = setc("eventcategory","1605000000");

var dup16 = setc("eventcategory","1612000000");

var dup17 = date_time({
	dest: "event_time",
	args: ["fld1","fld2","fld3"],
	fmts: [
		[dB,dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup18 = match("MESSAGE#0:01490502", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{event_description}", processor_chain([
	dup1,
	dup2,
]));

var dup19 = match("MESSAGE#58:crond:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: (%{username}) CMD (%{action})", processor_chain([
	dup15,
	dup2,
]));

var dup20 = match("MESSAGE#67:014d0001:02", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{info}", processor_chain([
	dup5,
	dup2,
]));

var hdr1 = match("HEADER#0:0001", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{hfld3}[%{hfld4}]: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant("["),
			field("hfld4"),
			constant("]: "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{hfld3}: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(": "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{hfld3}: [%{messageid}]%{p0}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(": ["),
			field("messageid"),
			constant("]"),
			field("p0"),
		],
	}),
]));

var hdr4 = match("HEADER#3:0004", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{messageid}[%{hfld3}]:%{p0}", processor_chain([
	setc("header_id","0004"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("messageid"),
			constant("["),
			field("hfld3"),
			constant("]:"),
			field("p0"),
		],
	}),
]));

var hdr5 = match("HEADER#4:0005", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{messageid}:%{p0}", processor_chain([
	setc("header_id","0005"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("messageid"),
			constant(":"),
			field("p0"),
		],
	}),
]));

var hdr6 = match("HEADER#5:0006", "message", "%{hmonth->} %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{hfld3}[%{hfld4}]: %{messageid->} /%{p0}", processor_chain([
	setc("header_id","0006"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant("["),
			field("hfld4"),
			constant("]: "),
			field("messageid"),
			constant(" /"),
			field("p0"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
	hdr5,
	hdr6,
]);

var msg1 = msg("01490502", dup18);

var part1 = match("MESSAGE#1:01490521", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Session statistics - bytes in:%{rbytes}, bytes out: %{sbytes}", processor_chain([
	dup3,
	dup2,
]));

var msg2 = msg("01490521", part1);

var part2 = match("MESSAGE#2:01490506", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Received User-Agent header: %{user_agent}", processor_chain([
	dup3,
	dup2,
]));

var msg3 = msg("01490506", part2);

var part3 = match("MESSAGE#3:01490113:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.server.network.name is %{fqdn}", processor_chain([
	dup3,
	dup2,
]));

var msg4 = msg("01490113:01", part3);

var part4 = match("MESSAGE#4:01490113:02", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.server.network.port is %{network_port}", processor_chain([
	dup3,
	dup2,
]));

var msg5 = msg("01490113:02", part4);

var part5 = match("MESSAGE#5:01490113:03", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.server.listener.name is %{service}", processor_chain([
	dup3,
	dup2,
]));

var msg6 = msg("01490113:03", part5);

var part6 = match("MESSAGE#6:01490113:04", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.server.network.protocol is %{network_service}", processor_chain([
	dup3,
	dup2,
]));

var msg7 = msg("01490113:04", part6);

var part7 = match("MESSAGE#7:01490113:05", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.user.agent is %{info}", processor_chain([
	dup3,
	dup2,
]));

var msg8 = msg("01490113:05", part7);

var part8 = match("MESSAGE#8:01490113:06", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.user.clientip is %{saddr}", processor_chain([
	dup3,
	dup2,
]));

var msg9 = msg("01490113:06", part8);

var part9 = match("MESSAGE#9:01490113", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: session.%{info}", processor_chain([
	dup3,
	dup2,
]));

var msg10 = msg("01490113", part9);

var select2 = linear_select([
	msg4,
	msg5,
	msg6,
	msg7,
	msg8,
	msg9,
	msg10,
]);

var part10 = match("MESSAGE#10:01490010/1_0", "nwparser.p0", "%{fld10}:%{fld11}:%{sessionid}: Username '%{p0}");

var part11 = match("MESSAGE#10:01490010/1_1", "nwparser.p0", "%{sessionid}: Username '%{p0}");

var select3 = linear_select([
	part10,
	part11,
]);

var part12 = match("MESSAGE#10:01490010/2", "nwparser.p0", "%{username}'");

var all1 = all_match({
	processors: [
		dup4,
		select3,
		part12,
	],
	on_success: processor_chain([
		setc("eventcategory","1401000000"),
		dup2,
	]),
});

var msg11 = msg("01490010", all1);

var part13 = match("MESSAGE#11:01490009", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: ACL '%{policyname}' assigned", processor_chain([
	setc("eventcategory","1501020000"),
	dup2,
]));

var msg12 = msg("01490009", part13);

var part14 = match("MESSAGE#12:01490102", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Access policy result: %{result}", processor_chain([
	setc("eventcategory","1501000000"),
	dup2,
]));

var msg13 = msg("01490102", part14);

var part15 = match("MESSAGE#13:01490000:02", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{authmethod->} authentication for user %{username->} using config %{fld8}", processor_chain([
	dup5,
	dup2,
]));

var msg14 = msg("01490000:02", part15);

var part16 = match("MESSAGE#14:01490000:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: found HTTP %{resultcode->} in response header", processor_chain([
	dup6,
	dup2,
]));

var msg15 = msg("01490000:01", part16);

var part17 = match("MESSAGE#15:01490000", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{filename->} func: \"%{action}\" line: %{fld8->} Msg: %{result}", processor_chain([
	dup5,
	dup2,
]));

var msg16 = msg("01490000", part17);

var part18 = match("MESSAGE#16:01490000:03", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{event_description}", processor_chain([
	dup5,
	dup2,
]));

var msg17 = msg("01490000:03", part18);

var select4 = linear_select([
	msg14,
	msg15,
	msg16,
	msg17,
]);

var part19 = match("MESSAGE#17:01490004", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{fld8}: Executed agent '%{application}', return value %{resultcode}", processor_chain([
	dup5,
	dup2,
]));

var msg18 = msg("01490004", part19);

var part20 = match("MESSAGE#18:01490500/1_0", "nwparser.p0", "%{fld10}:%{fld11}:%{sessionid}: New session from client IP %{p0}");

var part21 = match("MESSAGE#18:01490500/1_1", "nwparser.p0", "%{sessionid}: New session from client IP %{p0}");

var select5 = linear_select([
	part20,
	part21,
]);

var part22 = match("MESSAGE#18:01490500/2", "nwparser.p0", "%{saddr->} (ST=%{location_state}/CC=%{location_country}/C=%{location_city}) at VIP %{p0}");

var part23 = match("MESSAGE#18:01490500/3_0", "nwparser.p0", "%{daddr->} Listener %{fld8->} (Reputation=%{category})");

var part24 = match("MESSAGE#18:01490500/3_1", "nwparser.p0", "%{daddr->} Listener %{fld8}");

var part25 = match_copy("MESSAGE#18:01490500/3_2", "nwparser.p0", "daddr");

var select6 = linear_select([
	part23,
	part24,
	part25,
]);

var all2 = all_match({
	processors: [
		dup4,
		select5,
		part22,
		select6,
	],
	on_success: processor_chain([
		dup3,
		dup2,
	]),
});

var msg19 = msg("01490500", all2);

var part26 = match("MESSAGE#19:01490005", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Following rule %{fld8->} from item %{fld9->} to ending %{fld10}", processor_chain([
	dup7,
	dup2,
]));

var msg20 = msg("01490005", part26);

var part27 = match("MESSAGE#20:01490006", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Following rule %{fld8->} from item '%{fld9}' to item '%{fld10}'", processor_chain([
	dup7,
	dup2,
]));

var msg21 = msg("01490006", part27);

var part28 = match("MESSAGE#21:01490007", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Session variable '%{change_attribute}' set to %{change_new}", processor_chain([
	dup7,
	dup2,
]));

var msg22 = msg("01490007", part28);

var part29 = match("MESSAGE#22:01490008", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Connectivity resource %{application->} assigned", processor_chain([
	dup3,
	dup2,
]));

var msg23 = msg("01490008", part29);

var part30 = match("MESSAGE#23:01490514", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{fld8}: Access encountered error: %{result}. File: %{filename}, Function: %{action}, Line: %{fld9}", processor_chain([
	dup6,
	dup2,
]));

var msg24 = msg("01490514", part30);

var part31 = match("MESSAGE#24:01490505", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{event_description}", processor_chain([
	dup5,
	dup2,
]));

var msg25 = msg("01490505", part31);

var msg26 = msg("01490501", dup18);

var msg27 = msg("01490520", dup18);

var part32 = match("MESSAGE#27:01490142", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{event_description}", processor_chain([
	setc("eventcategory","1609000000"),
	dup2,
]));

var msg28 = msg("01490142", part32);

var part33 = match("MESSAGE#28:01490504", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{fqdn->} can not be resolved.", processor_chain([
	dup8,
	dup2,
]));

var msg29 = msg("01490504", part33);

var part34 = match("MESSAGE#29:01490538", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{fld8}: Configuration snapshot deleted by Access.", processor_chain([
	dup8,
	dup2,
]));

var msg30 = msg("01490538", part34);

var part35 = match("MESSAGE#30:01490107:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD module: authentication with '%{fld8}' failed: Clients credentials have been revoked, principal name: %{username}@%{fqdn}. %{result->} %{fld9}", processor_chain([
	dup9,
	dup2,
]));

var msg31 = msg("01490107:01", part35);

var part36 = match("MESSAGE#31:01490107", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD module: authentication with '%{username}' failed in %{action}: %{result->} %{fld8}", processor_chain([
	dup9,
	dup2,
]));

var msg32 = msg("01490107", part36);

var part37 = match("MESSAGE#32:01490107:02/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD module: authentication with '%{username}' failed: %{p0}");

var part38 = match("MESSAGE#32:01490107:02/1_0", "nwparser.p0", "Client '%{fqdn}' not found in Kerberos database, principal name:%{fld10->} %{p0}");

var part39 = match("MESSAGE#32:01490107:02/1_1", "nwparser.p0", "%{result->} %{p0}");

var select7 = linear_select([
	part38,
	part39,
]);

var part40 = match_copy("MESSAGE#32:01490107:02/2", "nwparser.p0", "info");

var all3 = all_match({
	processors: [
		part37,
		select7,
		part40,
	],
	on_success: processor_chain([
		dup9,
		dup2,
	]),
});

var msg33 = msg("01490107:02", all3);

var select8 = linear_select([
	msg31,
	msg32,
	msg33,
]);

var part41 = match("MESSAGE#33:01490106", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD module: authentication with '%{username}' failed in %{action}: Preauthentication failed, principal name: %{fld8}. %{result->} %{fld9}", processor_chain([
	dup9,
	dup2,
]));

var msg34 = msg("01490106", part41);

var part42 = match("MESSAGE#34:01490106:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD module: authentication with '%{username}' failed: Preauthentication failed, principal name: %{fld8}. %{result->} %{fld9}", processor_chain([
	dup9,
	dup2,
]));

var msg35 = msg("01490106:01", part42);

var select9 = linear_select([
	msg34,
	msg35,
]);

var part43 = match("MESSAGE#35:01490128", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Webtop %{application->} assigned", processor_chain([
	dup5,
	dup2,
]));

var msg36 = msg("01490128", part43);

var part44 = match("MESSAGE#36:01490101", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Access profile: %{fld8->} configuration has been applied. Newly active generation count is: %{dclass_counter1}", processor_chain([
	dup10,
	dup2,
	setc("dclass_counter1_string","Newly active generation count"),
]));

var msg37 = msg("01490101", part44);

var part45 = match("MESSAGE#37:01490103", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Retry Username '%{username}'", processor_chain([
	dup10,
	dup2,
]));

var msg38 = msg("01490103", part45);

var part46 = match("MESSAGE#38:01490115", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Following rule %{rulename->} from item %{fld9->} to terminalout %{fld10}", processor_chain([
	dup7,
	dup2,
]));

var msg39 = msg("01490115", part46);

var part47 = match("MESSAGE#39:01490017", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD agent: Auth (logon attempt:%{dclass_counter1}): authenticate with '%{username}' successful", processor_chain([
	dup7,
	dup2,
	dup11,
	dup12,
]));

var msg40 = msg("01490017", part47);

var part48 = match("MESSAGE#41:01490017:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD agent: Auth (logon attempt:%{dclass_counter1}): authenticate with '%{username}' failed", processor_chain([
	dup7,
	dup2,
	setc("disposition"," Failed"),
	dup12,
]));

var msg41 = msg("01490017:01", part48);

var select10 = linear_select([
	msg40,
	msg41,
]);

var part49 = match("MESSAGE#40:01490013", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD agent: Retrieving AAA server: %{fld8}", processor_chain([
	dup7,
	dup2,
]));

var msg42 = msg("01490013", part49);

var part50 = match("MESSAGE#42:01490019", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: AD agent: Query: query with '(sAMAccountName=%{username})' successful", processor_chain([
	dup7,
	dup2,
	dup11,
]));

var msg43 = msg("01490019", part50);

var part51 = match("MESSAGE#43:01490544", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Received client info - %{web_referer}", processor_chain([
	dup7,
	dup2,
]));

var msg44 = msg("01490544", part51);

var part52 = match("MESSAGE#44:01490511", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Initializing Access profile %{fld8->} with max concurrent user sessions limit: %{dclass_counter1}", processor_chain([
	dup7,
	dup2,
	setc("dclass_counter1_string"," Max Concurrent User Sessions Limit"),
]));

var msg45 = msg("01490511", part52);

var part53 = match("MESSAGE#45:014d0002", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: %{sessionid}: SSOv2 Logon succeeded, config %{fld8->} form %{fld9}", processor_chain([
	dup7,
	dup2,
	setc("disposition","Succeeded"),
]));

var msg46 = msg("014d0002", part53);

var part54 = match("MESSAGE#46:014d0002:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: %{sessionid}: SSOv2 Logon failed, config %{fld8->} form %{fld9}", processor_chain([
	dup7,
	dup2,
	setc("disposition","Failed"),
]));

var msg47 = msg("014d0002:01", part54);

var select11 = linear_select([
	msg46,
	msg47,
]);

var part55 = match("MESSAGE#47:01490079", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: %{sessionid}: Access policy '%{fld8}' configuration has changed.Access profile '%{fld9}' configuration changes need to be applied for the new configuration", processor_chain([
	dup7,
	dup2,
]));

var msg48 = msg("01490079", part55);

var part56 = match("MESSAGE#48:01490165", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Access profile: %{fld8->} initialized with configuration snapshot catalog: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg49 = msg("01490165", part56);

var part57 = match("MESSAGE#49:01490166", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Current snapshot ID: %{fld8->} retrieved from session db for access profile: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg50 = msg("01490166", part57);

var part58 = match("MESSAGE#50:01490167", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Current snapshot ID: %{fld8->} updated inside session db for access profile: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg51 = msg("01490167", part58);

var part59 = match("MESSAGE#51:01490169", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Snapshot catalog entry: %{fld8->} added for access profile: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg52 = msg("01490169", part59);

var part60 = match("MESSAGE#52:0149016a", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Initiating snapshot creation: %{fld8->} for access profile: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg53 = msg("0149016a", part60);

var part61 = match("MESSAGE#53:0149016b", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: %{fld7}:%{fld6}: Completed snapshot creation: %{fld8->} for access profile: %{fld9}", processor_chain([
	dup7,
	dup2,
]));

var msg54 = msg("0149016b", part61);

var part62 = match("MESSAGE#54:ssl_acc/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: [%{event_type}] %{saddr->} - %{p0}");

var part63 = match("MESSAGE#54:ssl_acc/1_0", "nwparser.p0", "- %{p0}");

var part64 = match("MESSAGE#54:ssl_acc/1_1", "nwparser.p0", "%{username->} %{p0}");

var select12 = linear_select([
	part63,
	part64,
]);

var part65 = match("MESSAGE#54:ssl_acc/2", "nwparser.p0", "[%{fld20->} %{timezone}] \"%{url}\" %{resultcode->} %{rbytes}");

var all4 = all_match({
	processors: [
		part62,
		select12,
		part65,
	],
	on_success: processor_chain([
		dup13,
		dup14,
		dup2,
	]),
});

var msg55 = msg("ssl_acc", all4);

var part66 = match("MESSAGE#55:ssl_req", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: [%{event_type}]%{space}[%{fld20->} %{timezone}] %{saddr->} %{protocol->} %{encryption_type->} \"%{url}\" %{rbytes}", processor_chain([
	dup13,
	dup14,
	dup2,
]));

var msg56 = msg("ssl_req", part66);

var part67 = match("MESSAGE#56:acc", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}: [%{event_type}]%{space}[%{fld20->} %{timezone}] \"%{web_method->} %{url->} %{version}\" %{resultcode->} %{rbytes->} \"%{fld7}\" \"%{user_agent}\"", processor_chain([
	dup13,
	dup14,
	dup2,
]));

var msg57 = msg("acc", part67);

var part68 = match("MESSAGE#57:crond", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: %{username}(%{sessionid}): %{action}", processor_chain([
	dup15,
	dup2,
]));

var msg58 = msg("crond", part68);

var msg59 = msg("crond:01", dup19);

var part69 = match("MESSAGE#59:crond:02", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: (%{username}) %{info}", processor_chain([
	dup15,
	dup2,
]));

var msg60 = msg("crond:02", part69);

var select13 = linear_select([
	msg58,
	msg59,
	msg60,
]);

var part70 = match("MESSAGE#60:sSMTP", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: %{info}", processor_chain([
	setc("eventcategory","1207000000"),
	dup2,
]));

var msg61 = msg("sSMTP", part70);

var part71 = match("MESSAGE#61:01420002", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: %{fld5}: AUDIT - pid=%{parent_pid->} user=%{username->} folder=%{directory->} module=%{fld6->} status=%{result->} cmd_data=%{info}", processor_chain([
	dup16,
	dup2,
]));

var msg62 = msg("01420002", part71);

var part72 = match("MESSAGE#62:syslog-ng", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: %{info}", processor_chain([
	dup15,
	dup2,
]));

var msg63 = msg("syslog-ng", part72);

var part73 = match("MESSAGE#63:syslog-ng:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}: %{info}", processor_chain([
	dup15,
	dup2,
]));

var msg64 = msg("syslog-ng:01", part73);

var select14 = linear_select([
	msg63,
	msg64,
]);

var part74 = match("MESSAGE#64:auditd", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: %{info}", processor_chain([
	dup16,
	dup2,
]));

var msg65 = msg("auditd", part74);

var part75 = match("MESSAGE#65:014d0001", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: ssoMethod: %{authmethod->} usernameSource: %{fld9->} passwordSource: %{fld10->} ntlmdomain: %{c_domain}", processor_chain([
	dup5,
	dup2,
]));

var msg66 = msg("014d0001", part75);

var part76 = match("MESSAGE#66:014d0001:01/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: ctx: %{fld9}, %{p0}");

var part77 = match("MESSAGE#66:014d0001:01/1_0", "nwparser.p0", "SERVER %{p0}");

var part78 = match("MESSAGE#66:014d0001:01/1_1", "nwparser.p0", "CLIENT %{p0}");

var select15 = linear_select([
	part77,
	part78,
]);

var part79 = match("MESSAGE#66:014d0001:01/2", "nwparser.p0", ": %{info}");

var all5 = all_match({
	processors: [
		part76,
		select15,
		part79,
	],
	on_success: processor_chain([
		dup5,
		dup2,
	]),
});

var msg67 = msg("014d0001:01", all5);

var msg68 = msg("014d0001:02", dup20);

var select16 = linear_select([
	msg66,
	msg67,
	msg68,
]);

var msg69 = msg("014d0044", dup20);

var part80 = match("MESSAGE#69:01490549/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: Assigned PPP Dynamic IPv4: %{stransaddr->} Tunnel Type: %{group->} %{fld8->} Resource: %{rulename->} Client IP: %{p0}");

var part81 = match("MESSAGE#69:01490549/1_0", "nwparser.p0", "%{saddr->} - %{fld9}");

var part82 = match("MESSAGE#69:01490549/1_1", "nwparser.p0", "%{saddr}");

var select17 = linear_select([
	part81,
	part82,
]);

var all6 = all_match({
	processors: [
		part80,
		select17,
	],
	on_success: processor_chain([
		dup3,
		dup2,
	]),
});

var msg70 = msg("01490549", all6);

var part83 = match("MESSAGE#70:01490547", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: Access Profile %{rulename}: %{result->} for %{saddr}", processor_chain([
	dup3,
	dup2,
]));

var msg71 = msg("01490547", part83);

var part84 = match("MESSAGE#71:01490517", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{result}", processor_chain([
	dup3,
	dup2,
]));

var msg72 = msg("01490517", part84);

var part85 = match("MESSAGE#72:011f0005", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{result->} (Client side: vip=%{url->} profile=%{protocol->} pool=%{fld8->} client_ip=%{saddr})", processor_chain([
	dup3,
	dup2,
]));

var msg73 = msg("011f0005", part85);

var part86 = match("MESSAGE#73:014d0048", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7->} %{rulename->} \u003c\u003c%{event_description}>: APM_EVENT=%{action->} | %{username->} | %{fld8->} ***%{result}***", processor_chain([
	dup3,
	dup2,
]));

var msg74 = msg("014d0048", part86);

var part87 = match("MESSAGE#74:error", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: [%{fld7}] [client %{saddr}] %{result}: %{url}", processor_chain([
	dup3,
	dup2,
]));

var msg75 = msg("error", part87);

var msg76 = msg("CROND:03", dup19);

var part88 = match("MESSAGE#76:01260009", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]:%{fld7}:%{fld6}: Connection error:%{event_description}", processor_chain([
	dup6,
	dup2,
]));

var msg77 = msg("01260009", part88);

var part89 = match("MESSAGE#77:apmd:04", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} %{severity->} %{agent}[%{process_id}]: %{fld4->} /Common/home_agent_tca:Common:%{fld5}: %{fld6->} - Hostname: %{shost->} Type: %{fld7->} Version: %{version->} Platform: %{os->} CPU: %{fld8->} Mode:%{fld9}", processor_chain([
	dup15,
	dup2,
	dup17,
]));

var msg78 = msg("apmd:04", part89);

var part90 = match("MESSAGE#78:apmd:03", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} %{severity->} %{agent}[%{process_id}]: %{fld4->} /Common/home_agent_tca:Common:%{fld5}: RADIUS module: parseResponse(): Access-Reject packet from host %{saddr}:%{sport->} %{fld7}", processor_chain([
	dup9,
	dup2,
	dup17,
]));

var msg79 = msg("apmd:03", part90);

var part91 = match("MESSAGE#79:apmd:02/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} %{severity->} %{agent}[%{process_id}]: %{fld4->} /Common/home_agent_tca:Common:%{fld5}: RADIUS module: authentication with '%{username}' failed: %{p0}");

var part92 = match("MESSAGE#79:apmd:02/1_0", "nwparser.p0", "%{fld6->} from host %{saddr}:%{sport->} %{fld7}");

var part93 = match("MESSAGE#79:apmd:02/1_1", "nwparser.p0", "%{fld8}");

var select18 = linear_select([
	part92,
	part93,
]);

var all7 = all_match({
	processors: [
		part91,
		select18,
	],
	on_success: processor_chain([
		dup9,
		dup2,
		dup17,
	]),
});

var msg80 = msg("apmd:02", all7);

var part94 = match("MESSAGE#80:apmd", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} %{severity->} %{agent}[%{process_id}]:%{info}", processor_chain([
	dup15,
	dup2,
	dup17,
]));

var msg81 = msg("apmd", part94);

var select19 = linear_select([
	msg78,
	msg79,
	msg80,
	msg81,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"011f0005": msg73,
		"01260009": msg77,
		"01420002": msg62,
		"01490000": select4,
		"01490004": msg18,
		"01490005": msg20,
		"01490006": msg21,
		"01490007": msg22,
		"01490008": msg23,
		"01490009": msg12,
		"01490010": msg11,
		"01490013": msg42,
		"01490017": select10,
		"01490019": msg43,
		"01490079": msg48,
		"01490101": msg37,
		"01490102": msg13,
		"01490103": msg38,
		"01490106": select9,
		"01490107": select8,
		"01490113": select2,
		"01490115": msg39,
		"01490128": msg36,
		"01490142": msg28,
		"01490165": msg49,
		"01490166": msg50,
		"01490167": msg51,
		"01490169": msg52,
		"0149016a": msg53,
		"0149016b": msg54,
		"01490500": msg19,
		"01490501": msg26,
		"01490502": msg1,
		"01490504": msg29,
		"01490505": msg25,
		"01490506": msg3,
		"01490511": msg45,
		"01490514": msg24,
		"01490517": msg72,
		"01490520": msg27,
		"01490521": msg2,
		"01490538": msg30,
		"01490544": msg44,
		"01490547": msg71,
		"01490549": msg70,
		"014d0001": select16,
		"014d0002": select11,
		"014d0044": msg69,
		"CROND": msg76,
		"Rule": msg74,
		"acc": msg57,
		"apmd": select19,
		"auditd": msg65,
		"crond": select13,
		"error": msg75,
		"sSMTP": msg61,
		"ssl_acc": msg55,
		"ssl_req": msg56,
		"syslog-ng": select14,
	}),
]);

var part95 = match("MESSAGE#10:01490010/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{p0}");

var part96 = match("MESSAGE#0:01490502", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{sessionid}: %{event_description}", processor_chain([
	dup1,
	dup2,
]));

var part97 = match("MESSAGE#58:crond:01", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{agent}[%{process_id}]: (%{username}) CMD (%{action})", processor_chain([
	dup15,
	dup2,
]));

var part98 = match("MESSAGE#67:014d0001:02", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{fld4->} %{severity->} %{fld5}[%{process_id}]: %{fld7}:%{fld6}: %{info}", processor_chain([
	dup5,
	dup2,
]));
