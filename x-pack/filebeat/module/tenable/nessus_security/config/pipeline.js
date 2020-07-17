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

var dup1 = setc("eventcategory","1605000000");

var dup2 = setf("msg","$MSG");

var dup3 = setc("eventcategory","1801020000");

var dup4 = date_time({
	dest: "event_time",
	args: ["hfld21","hfld22","hfld23","hfld24"],
	fmts: [
		[dB,dF,dN,dc(":"),dU,dc(":"),dO,dW],
	],
});

var dup5 = setc("eventcategory","1614000000");

var dup6 = setc("action","scan started");

var dup7 = setc("eventcategory","1609000000");

var dup8 = setc("action","started");

var dup9 = setc("eventcategory","1401030000");

var dup10 = setc("action","login failure");

var dup11 = setc("ec_outcome","Failure");

var dup12 = setc("ec_activity","Scan");

var dup13 = setc("eventcategory","1607000000");

var dup14 = match("MESSAGE#10:Total", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var dup15 = match("MESSAGE#12:started", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup8,
]));

var dup16 = match("MESSAGE#45:Could", "nwparser.payload", "%{event_description}", processor_chain([
	dup13,
	dup2,
	dup4,
]));

var hdr1 = match("HEADER#0:0001", "message", "%{hfld1->} %NESSUSVS-%{messageid}: %{payload}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(": "),
			field("payload"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%NESSUSVS-%{hfld49}: [%{hfld20->} %{hfld21->} %{hfld22->} %{hfld23->} %{hfld24}][%{hfld2}.%{hfld3}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "%NESSUSVS-%{hfld49}: [%{hfld20->} %{hfld21->} %{hfld22->} %{hfld23->} %{hfld24}][%{hfld2}.%{hfld3}] %{hfld4}: %{messageid->} %{payload}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld4"),
			constant(": "),
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr4 = match("HEADER#3:0004", "message", "%NESSUSVS-%{hfld49}: [%{hfld20->} %{hfld21->} %{hfld22->} %{hfld23->} %{hfld24}][%{hfld2}.%{hfld3}] %{hfld4}: %{hfld5->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0004"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld4"),
			constant(": "),
			field("hfld5"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr5 = match("HEADER#4:0005", "message", "%NESSUSVS-%{hfld49}: [%{hfld20->} %{hfld21->} %{hfld22->} %{hfld23->} %{hfld24}][%{hfld2}.%{hfld3}] %{hfld4->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0005"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld4"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr6 = match("HEADER#5:0006", "message", "%NESSUSVS-%{hfld49}: [%{hfld20->} %{hfld21->} %{hfld22->} %{hfld23->} %{hfld24}][%{hfld2}.%{hfld3}] %{hfld4->} (%{messageid->} %{hfld5}) %{hfld6->} %{payload}", processor_chain([
	setc("header_id","0006"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld4"),
			constant(" ("),
			field("messageid"),
			constant(" "),
			field("hfld5"),
			constant(") "),
			field("hfld6"),
			constant(" "),
			field("payload"),
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

var part1 = match("MESSAGE#0:REPORTITEM", "nwparser.payload", "%{fld1}:Hostname=%{hostname}^^Host_ip=%{hostip}^^FQDN=%{fqdn}^^Port=%{network_port}^^OS=%{os}^^MAC_address=%{macaddr}^^Host_start=%{fld30}^^Host_end=%{fld31}^^Severity=%{severity}^^Risk_factor=%{risk}^^Service_name=%{service}^^Protocol=%{protocol}^^Vulnerability_refs=%{vuln_ref}^^CVSS_base_score=%{risk_num}^^CVSS_vector=%{fld32}^^PluginID=%{rule}^^Plugin_name=%{rulename}^^Plugin Family=%{rule_group}^^Synopsis=%{event_description}", processor_chain([
	dup1,
	dup2,
]));

var msg1 = msg("REPORTITEM", part1);

var part2 = match("MESSAGE#1:REPORTITEM:01", "nwparser.payload", "%{fld1}:Hostname=%{hostname}^^Host_ip=%{hostip}^^FQDN=%{fqdn}^^Port=%{network_port}^^OS=%{os}^^MAC_address=%{macaddr}^^%{event_description}", processor_chain([
	dup1,
	dup2,
]));

var msg2 = msg("REPORTITEM:01", part2);

var select2 = linear_select([
	msg1,
	msg2,
]);

var part3 = match("MESSAGE#2:connection", "nwparser.payload", "connection from %{hostip}", processor_chain([
	dup3,
	dup2,
	dup4,
	setc("action","connecting"),
]));

var msg3 = msg("connection", part3);

var part4 = match("MESSAGE#3:Deleting", "nwparser.payload", "Deleting user %{username}", processor_chain([
	dup3,
	setc("ec_subject","User"),
	setc("ec_activity","Delete"),
	dup2,
	dup4,
	setc("action","Deleting"),
]));

var msg4 = msg("Deleting", part4);

var part5 = match("MESSAGE#4:Finished", "nwparser.payload", "Finished testing %{hostip}. %{fld5}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","Finished testing"),
]));

var msg5 = msg("Finished", part5);

var part6 = match("MESSAGE#5:Finished:01", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","Finished"),
]));

var msg6 = msg("Finished:01", part6);

var select3 = linear_select([
	msg5,
	msg6,
]);

var part7 = match("MESSAGE#6:finished", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","finished"),
]));

var msg7 = msg("finished", part7);

var part8 = match("MESSAGE#7:user", "nwparser.payload", "user %{username->} : test complete", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","Test Complete"),
]));

var msg8 = msg("user", part8);

var part9 = match("MESSAGE#8:user:01", "nwparser.payload", "user %{username->} : testing %{hostname->} (%{hostip}) %{fld1}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","testing"),
]));

var msg9 = msg("user:01", part9);

var part10 = match("MESSAGE#21:user:02", "nwparser.payload", "user %{username->} starts a new scan. Target(s) : %{hostname}, %{info}", processor_chain([
	dup5,
	dup2,
	dup4,
	dup6,
]));

var msg10 = msg("user:02", part10);

var part11 = match("MESSAGE#26:user_launching", "nwparser.payload", "user %{username->} : launching %{rulename->} against %{url->} [%{process_id}]", processor_chain([
	setc("eventcategory","1401000000"),
	dup2,
	dup4,
	setc("event_description","User launched rule scan"),
]));

var msg11 = msg("user_launching", part11);

var part12 = match("MESSAGE#27:user_not_launching", "nwparser.payload", "user %{username->} : Not launching %{rulename->} against %{url->} %{reason}", processor_chain([
	dup7,
	dup2,
	dup4,
]));

var msg12 = msg("user_not_launching", part12);

var select4 = linear_select([
	msg8,
	msg9,
	msg10,
	msg11,
	msg12,
]);

var part13 = match("MESSAGE#9:Scan", "nwparser.payload", "Scan done: %{info}", processor_chain([
	dup5,
	dup2,
	dup4,
	setc("action","Scan complete"),
]));

var msg13 = msg("Scan", part13);

var msg14 = msg("Total", dup14);

var msg15 = msg("Task", dup14);

var msg16 = msg("started", dup15);

var part14 = match("MESSAGE#13:failed", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","failed"),
]));

var msg17 = msg("failed", part14);

var part15 = match("MESSAGE#14:Nessus", "nwparser.payload", "%{event_description->} (pid=%{process_id})", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var msg18 = msg("Nessus", part15);

var part16 = match("MESSAGE#15:Reloading", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","Reloading"),
]));

var msg19 = msg("Reloading", part16);

var part17 = match("MESSAGE#16:New", "nwparser.payload", "New connection timeout -- closing the socket%{}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","connection timeout"),
]));

var msg20 = msg("New", part17);

var part18 = match("MESSAGE#17:Invalid", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("action","Invalid"),
]));

var msg21 = msg("Invalid", part18);

var msg22 = msg("Client", dup14);

var msg23 = msg("auth_check_user", dup14);

var part19 = match("MESSAGE#20:bad", "nwparser.payload", "bad login attempt from %{hostip}", processor_chain([
	dup9,
	dup2,
	dup4,
	dup10,
]));

var msg24 = msg("bad", part19);

var msg25 = msg("Reducing", dup14);

var msg26 = msg("Redirecting", dup14);

var msg27 = msg("Missing", dup14);

var part20 = match("MESSAGE#25:User", "nwparser.payload", "User '%{username}' %{event_description}", processor_chain([
	setc("eventcategory","1401060000"),
	dup2,
	dup4,
]));

var msg28 = msg("User", part20);

var part21 = match("MESSAGE#32:User:01", "nwparser.payload", "User %{username->} starts a new scan (%{fld25})", processor_chain([
	dup5,
	dup2,
	dup4,
	dup6,
]));

var msg29 = msg("User:01", part21);

var select5 = linear_select([
	msg28,
	msg29,
]);

var part22 = match("MESSAGE#28:Plugins", "nwparser.payload", "%{event_description}, as %{reason}", processor_chain([
	dup1,
	dup11,
	dup2,
	dup4,
]));

var msg30 = msg("Plugins", part22);

var part23 = match("MESSAGE#29:process_finished", "nwparser.payload", "%{rulename->} (process %{process_id}) finished its job in %{duration->} seconds", processor_chain([
	dup1,
	dup12,
	setc("ec_outcome","Success"),
	dup2,
	dup4,
	setc("event_description","Rule scan finished"),
]));

var msg31 = msg("process_finished", part23);

var part24 = match("MESSAGE#30:process_notfinished_killed", "nwparser.payload", "%{rulename->} (pid %{process_id}) is slow to finish - killing it", processor_chain([
	dup7,
	dup12,
	dup11,
	dup2,
	dup4,
	setc("event_description","Rule scan killed due to slow response"),
]));

var msg32 = msg("process_notfinished_killed", part24);

var part25 = match("MESSAGE#31:TCP", "nwparser.payload", "%{fld1->} TCP sessions in parallel", processor_chain([
	dup1,
	dup2,
	dup4,
	setc("event_description","TCP sessions in parallel"),
]));

var msg33 = msg("TCP", part25);

var msg34 = msg("nessusd", dup14);

var msg35 = msg("installation", dup14);

var msg36 = msg("Running", dup14);

var msg37 = msg("started.", dup15);

var msg38 = msg("scanner", dup14);

var part26 = match("MESSAGE#38:Another", "nwparser.payload", "%{event_description->} (pid %{process_id})", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var msg39 = msg("Another", part26);

var part27 = match("MESSAGE#39:Bad", "nwparser.payload", "Bad login attempt for user '%{username}' %{info}", processor_chain([
	dup9,
	dup2,
	dup4,
	dup10,
]));

var msg40 = msg("Bad", part27);

var msg41 = msg("Full", dup14);

var msg42 = msg("System", dup14);

var msg43 = msg("Initial", dup14);

var part28 = match("MESSAGE#43:Adding", "nwparser.payload", "Adding new user '%{username}'", processor_chain([
	setc("eventcategory","1402020200"),
	dup2,
	dup4,
]));

var msg44 = msg("Adding", part28);

var part29 = match("MESSAGE#44:Granting", "nwparser.payload", "Granting admin privileges to user '%{username}'", processor_chain([
	setc("eventcategory","1402030000"),
	dup2,
	dup4,
]));

var msg45 = msg("Granting", part29);

var msg46 = msg("Could", dup16);

var msg47 = msg("depends", dup16);

var msg48 = msg("Converting", dup14);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"Adding": msg44,
		"Another": msg39,
		"Bad": msg40,
		"Client": msg22,
		"Converting": msg48,
		"Could": msg46,
		"Deleting": msg4,
		"Finished": select3,
		"Full": msg41,
		"Granting": msg45,
		"Initial": msg43,
		"Invalid": msg21,
		"Missing": msg27,
		"Nessus": msg18,
		"New": msg20,
		"Plugins": msg30,
		"REPORTITEM": select2,
		"Redirecting": msg26,
		"Reducing": msg25,
		"Reloading": msg19,
		"Running": msg36,
		"Scan": msg13,
		"System": msg42,
		"TCP": msg33,
		"Task": msg15,
		"Total": msg14,
		"User": select5,
		"auth_check_user": msg23,
		"bad": msg24,
		"connection": msg3,
		"depends": msg47,
		"failed": msg17,
		"finished": msg7,
		"installation": msg35,
		"nessusd": msg34,
		"pid": msg32,
		"process": msg31,
		"scanner": msg38,
		"started": msg16,
		"started.": msg37,
		"user": select4,
	}),
]);

var part30 = match("MESSAGE#10:Total", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var part31 = match("MESSAGE#12:started", "nwparser.payload", "%{event_description}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup8,
]));

var part32 = match("MESSAGE#45:Could", "nwparser.payload", "%{event_description}", processor_chain([
	dup13,
	dup2,
	dup4,
]));
