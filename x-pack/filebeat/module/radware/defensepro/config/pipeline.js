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

var dup1 = match("MESSAGE#0:Intrusions:01/0", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} %{id->} %{category->} \"%{event_type}\" %{protocol->} %{p0}");

var dup2 = match("MESSAGE#0:Intrusions:01/1_0", "nwparser.p0", "%{saddr}:%{p0}");

var dup3 = match("MESSAGE#0:Intrusions:01/1_1", "nwparser.p0", "%{saddr->} %{p0}");

var dup4 = match("MESSAGE#0:Intrusions:01/2", "nwparser.p0", "%{sport->} %{p0}");

var dup5 = match("MESSAGE#0:Intrusions:01/3_0", "nwparser.p0", "%{daddr}:%{p0}");

var dup6 = match("MESSAGE#0:Intrusions:01/3_1", "nwparser.p0", "%{daddr->} %{p0}");

var dup7 = match("MESSAGE#0:Intrusions:01/4", "nwparser.p0", "%{dport->} %{interface->} %{context->} \"%{policyname}\" %{event_state->} %{packets->} %{dclass_counter1->} %{vlan->} %{fld15->} %{fld16->} %{risk->} %{p0}");

var dup8 = match("MESSAGE#0:Intrusions:01/5_0", "nwparser.p0", "%{action->} %{sigid_string}");

var dup9 = match_copy("MESSAGE#0:Intrusions:01/5_1", "nwparser.p0", "action");

var dup10 = setc("eventcategory","1001000000");

var dup11 = setc("ec_theme","TEV");

var dup12 = setf("msg","$MSG");

var dup13 = date_time({
	dest: "event_time",
	args: ["fld1","fld2"],
	fmts: [
		[dF,dc("-"),dG,dc("-"),dW,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup14 = setc("dclass_counter1_string","Bandwidth in Kbps");

var dup15 = match("MESSAGE#1:Intrusions:02/0", "nwparser.payload", "%{id->} %{category->} \\\"%{event_type}\\\" %{protocol->} %{p0}");

var dup16 = match("MESSAGE#1:Intrusions:02/4", "nwparser.p0", "%{dport->} %{interface->} %{context->} \\\"%{policyname}\\\" %{event_state->} %{packets->} %{dclass_counter1->} %{fld1->} %{risk->} %{action->} %{vlan->} %{fld15->} %{fld16->} %{direction}");

var dup17 = setc("eventcategory","1002000000");

var dup18 = setc("ec_subject","NetworkComm");

var dup19 = setc("ec_activity","Scan");

var dup20 = setc("eventcategory","1401000000");

var dup21 = setc("ec_subject","User");

var dup22 = setc("ec_theme","ALM");

var dup23 = setc("ec_activity","Modify");

var dup24 = setc("ec_theme","Configuration");

var dup25 = setc("eventcategory","1612000000");

var dup26 = match("MESSAGE#22:Login:04/1_0", "nwparser.p0", "for user%{p0}");

var dup27 = match("MESSAGE#22:Login:04/1_1", "nwparser.p0", "user%{p0}");

var dup28 = match("MESSAGE#22:Login:04/2", "nwparser.p0", "%{} %{username->} via %{network_service->} (IP: %{saddr})%{p0}");

var dup29 = match("MESSAGE#22:Login:04/3_0", "nwparser.p0", ": %{result}");

var dup30 = match_copy("MESSAGE#22:Login:04/3_1", "nwparser.p0", "result");

var dup31 = setc("eventcategory","1401030000");

var dup32 = setc("ec_activity","Logon");

var dup33 = setc("ec_theme","Authentication");

var dup34 = setc("ec_outcome","Failure");

var dup35 = setc("event_description","Login Failed");

var dup36 = setc("ec_outcome","Error");

var dup37 = setc("eventcategory","1603000000");

var dup38 = setc("ec_theme","AccessControl");

var dup39 = setc("eventcategory","1401060000");

var dup40 = setc("ec_outcome","Success");

var dup41 = setc("event_description","User logged in");

var dup42 = linear_select([
	dup2,
	dup3,
]);

var dup43 = linear_select([
	dup5,
	dup6,
]);

var dup44 = linear_select([
	dup8,
	dup9,
]);

var dup45 = linear_select([
	dup26,
	dup27,
]);

var dup46 = linear_select([
	dup29,
	dup30,
]);

var dup47 = all_match({
	processors: [
		dup1,
		dup42,
		dup4,
		dup43,
		dup7,
		dup44,
	],
	on_success: processor_chain([
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
	]),
});

var dup48 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup10,
		dup11,
		dup12,
		dup14,
	]),
});

var dup49 = all_match({
	processors: [
		dup1,
		dup42,
		dup4,
		dup43,
		dup7,
		dup44,
	],
	on_success: processor_chain([
		dup17,
		dup11,
		dup12,
		dup13,
		dup14,
	]),
});

var dup50 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup17,
		dup11,
		dup12,
		dup14,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "%DefensePro %{hfld1->} %{hfld2->} %{hfld3->} %{messageid->} \\\"%{hfld4}\\\" %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld3"),
			constant(" "),
			field("messageid"),
			constant(" \\\""),
			field("hfld4"),
			constant("\\\" "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%DefensePro %{messageid->} %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "DefensePro: %{hdate->} %{htime->} %{hfld1->} %{hfld2->} %{messageid->} \"%{hfld3}\" %{p0}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("messageid"),
			constant(" \""),
			field("hfld3"),
			constant("\" "),
			field("p0"),
		],
	}),
]));

var hdr4 = match("HEADER#3:0004", "message", "DefensePro: %{hdate->} %{htime->} %{hfld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0004"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hfld1"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
]);

var msg1 = msg("Intrusions:01", dup47);

var msg2 = msg("Intrusions:02", dup48);

var select2 = linear_select([
	msg1,
	msg2,
]);

var msg3 = msg("SynFlood:01", dup49);

var msg4 = msg("Behavioral-DoS:01", dup49);

var msg5 = msg("Behavioral-DoS:02", dup50);

var select3 = linear_select([
	msg4,
	msg5,
]);

var all1 = all_match({
	processors: [
		dup1,
		dup42,
		dup4,
		dup43,
		dup7,
		dup44,
	],
	on_success: processor_chain([
		dup10,
		dup18,
		dup19,
		dup11,
		dup12,
		dup13,
		dup14,
	]),
});

var msg6 = msg("Anti-Scanning:01", all1);

var all2 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup10,
		dup18,
		dup19,
		dup11,
		dup12,
		dup14,
	]),
});

var msg7 = msg("Anti-Scanning:02", all2);

var select4 = linear_select([
	msg6,
	msg7,
]);

var msg8 = msg("DoS:01", dup49);

var all3 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup17,
		dup18,
		dup19,
		dup11,
		dup12,
		dup14,
	]),
});

var msg9 = msg("DoS:02", all3);

var select5 = linear_select([
	msg8,
	msg9,
]);

var msg10 = msg("Cracking-Protection:01", dup47);

var msg11 = msg("Cracking-Protection:02", dup48);

var select6 = linear_select([
	msg10,
	msg11,
]);

var msg12 = msg("Anomalies:01", dup49);

var msg13 = msg("Anomalies:02", dup50);

var select7 = linear_select([
	msg12,
	msg13,
]);

var msg14 = msg("HttpFlood:01", dup49);

var msg15 = msg("HttpFlood:02", dup50);

var select8 = linear_select([
	msg14,
	msg15,
]);

var part1 = match("MESSAGE#15:COMMAND:", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} COMMAND: \"%{action}\" by user %{username->} via %{network_service}, source IP %{saddr}", processor_chain([
	dup20,
	dup21,
	setc("ec_activity","Execute"),
	dup22,
	dup12,
	dup13,
]));

var msg16 = msg("COMMAND:", part1);

var part2 = match("MESSAGE#16:Configuration:01", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} %{event_description->} set %{change_new}, Old Values: %{change_old}, ACTION: %{action->} by user %{username->} via %{network_service->} source IP %{saddr}", processor_chain([
	dup20,
	dup21,
	dup23,
	dup24,
	dup12,
	dup13,
]));

var msg17 = msg("Configuration:01", part2);

var part3 = match("MESSAGE#17:Configuration:02", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} %{event_description}, ACTION: %{action->} by user %{username->} via %{network_service->} source IP %{saddr}", processor_chain([
	dup20,
	dup21,
	dup24,
	dup12,
	dup13,
]));

var msg18 = msg("Configuration:02", part3);

var part4 = match("MESSAGE#18:Configuration:03", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Configuration File downloaded from device by user %{username->} via %{network_service}, source IP %{saddr}", processor_chain([
	dup20,
	dup21,
	dup24,
	dup12,
	setc("event_description","Configuration File downloaded"),
	dup13,
]));

var msg19 = msg("Configuration:03", part4);

var part5 = match("MESSAGE#19:Configuration:04", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Configuration Upload has been completed", processor_chain([
	dup25,
	dup24,
	dup12,
	setc("event_description","Configuration Upload has been completed"),
	dup13,
]));

var msg20 = msg("Configuration:04", part5);

var part6 = match("MESSAGE#20:Configuration:05", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Configuration Download has been completed", processor_chain([
	dup25,
	dup24,
	dup12,
	setc("event_description","Configuration Download has been completed"),
	dup13,
]));

var msg21 = msg("Configuration:05", part6);

var part7 = match("MESSAGE#21:Configuration:06", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Configuration file has been modified. Device may fail to load configuration file!", processor_chain([
	dup25,
	dup23,
	dup24,
	dup12,
	setc("event_description","Configuration file has been modified. Device may fail to load configuration file!"),
	dup13,
]));

var msg22 = msg("Configuration:06", part7);

var select9 = linear_select([
	msg17,
	msg18,
	msg19,
	msg20,
	msg21,
	msg22,
]);

var part8 = match("MESSAGE#22:Login:04/0", "nwparser.payload", "Login failed %{p0}");

var all4 = all_match({
	processors: [
		part8,
		dup45,
		dup28,
		dup46,
	],
	on_success: processor_chain([
		dup31,
		dup21,
		dup32,
		dup33,
		dup34,
		dup12,
		dup35,
	]),
});

var msg23 = msg("Login:04", all4);

var part9 = match("MESSAGE#23:Login:05", "nwparser.payload", "Login locked user %{username->} (IP: %{saddr}): %{result}", processor_chain([
	dup31,
	dup21,
	dup32,
	dup33,
	dup36,
	dup12,
	setc("event_description","Login Locked"),
]));

var msg24 = msg("Login:05", part9);

var part10 = match("MESSAGE#24:Login:01/0", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Login failed %{p0}");

var all5 = all_match({
	processors: [
		part10,
		dup45,
		dup28,
		dup46,
	],
	on_success: processor_chain([
		dup31,
		dup21,
		dup32,
		dup33,
		dup34,
		dup12,
		dup35,
		dup13,
	]),
});

var msg25 = msg("Login:01", all5);

var part11 = match("MESSAGE#25:Login:02", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Login failed via %{network_service->} (IP: %{saddr}): %{result}", processor_chain([
	dup31,
	dup21,
	dup32,
	dup33,
	dup34,
	dup12,
	dup35,
	dup13,
]));

var msg26 = msg("Login:02", part11);

var part12 = match("MESSAGE#26:Login:03", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Login locked user %{username->} (IP: %{saddr}): %{result}", processor_chain([
	dup31,
	dup21,
	dup32,
	dup33,
	dup36,
	dup12,
	dup35,
	dup13,
]));

var msg27 = msg("Login:03", part12);

var select10 = linear_select([
	msg23,
	msg24,
	msg25,
	msg26,
	msg27,
]);

var part13 = match("MESSAGE#27:Connection", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Connection to NTP server timed out", processor_chain([
	dup37,
	dup22,
	dup12,
	setc("event_description","Connection to NTP server timed out"),
	dup13,
]));

var msg28 = msg("Connection", part13);

var part14 = match("MESSAGE#28:Device", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Device was rebooted by user %{username->} via %{network_service}, source IP %{saddr}", processor_chain([
	dup20,
	dup21,
	dup22,
	dup12,
	setc("event_description","Device was rebooted"),
	dup13,
]));

var msg29 = msg("Device", part14);

var part15 = match("MESSAGE#29:Power", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Power supply fully operational", processor_chain([
	dup25,
	dup22,
	dup12,
	setc("event_description","Power supply fully operational"),
	dup13,
]));

var msg30 = msg("Power", part15);

var part16 = match("MESSAGE#30:Cold", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Cold Start", processor_chain([
	dup25,
	setc("ec_activity","Start"),
	dup22,
	dup12,
	setc("event_description","Cold Start"),
	dup13,
]));

var msg31 = msg("Cold", part16);

var part17 = match("MESSAGE#31:Port/0", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Port %{interface->} %{p0}");

var part18 = match("MESSAGE#31:Port/1_0", "nwparser.p0", "Down%{}");

var part19 = match("MESSAGE#31:Port/1_1", "nwparser.p0", "Up%{}");

var select11 = linear_select([
	part18,
	part19,
]);

var all6 = all_match({
	processors: [
		part17,
		select11,
	],
	on_success: processor_chain([
		dup25,
		dup22,
		dup12,
		setc("event_description","Port Status Change"),
		dup13,
	]),
});

var msg32 = msg("Port", all6);

var part20 = match("MESSAGE#32:DefensePro", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} DefensePro was powered off", processor_chain([
	dup25,
	dup22,
	dup12,
	setc("event_description","DefensePro Powered off"),
	dup13,
]));

var msg33 = msg("DefensePro", part20);

var part21 = match("MESSAGE#33:Access:01/0", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} %{id->} %{category->} \"%{event_type}\" %{protocol->} %{saddr->} %{sport->} %{daddr->} %{dport->} %{interface->} %{context->} \"%{policyname}\" %{event_state->} %{packets->} %{dclass_counter1->} %{vlan->} %{fld15->} %{fld16->} %{risk->} %{p0}");

var all7 = all_match({
	processors: [
		part21,
		dup44,
	],
	on_success: processor_chain([
		dup37,
		dup38,
		dup12,
		dup13,
	]),
});

var msg34 = msg("Access:01", all7);

var part22 = match("MESSAGE#34:Access", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Access attempted by unauthorized NMS, Community: %{fld3}, IP: \"%{saddr}\"", processor_chain([
	dup37,
	dup38,
	dup12,
	setc("event_description","Access attempted by unauthorized NMS"),
	dup13,
]));

var msg35 = msg("Access", part22);

var select12 = linear_select([
	msg34,
	msg35,
]);

var part23 = match("MESSAGE#35:Please", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Please reboot the device for the latest changes to take effect", processor_chain([
	dup20,
	dup22,
	dup12,
	setc("event_description","Reboot required for latest changes"),
	dup13,
]));

var msg36 = msg("Please", part23);

var part24 = match("MESSAGE#36:User:01", "nwparser.payload", "User %{username->} logged in via %{network_service->} (IP: %{saddr})", processor_chain([
	dup39,
	dup21,
	dup32,
	dup33,
	dup40,
	dup12,
	dup41,
]));

var msg37 = msg("User:01", part24);

var part25 = match("MESSAGE#37:User", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} User %{username->} logged in via %{network_service->} (IP: %{saddr})", processor_chain([
	dup39,
	dup21,
	dup32,
	dup33,
	dup40,
	dup12,
	dup41,
	dup13,
]));

var msg38 = msg("User", part25);

var select13 = linear_select([
	msg37,
	msg38,
]);

var part26 = match("MESSAGE#38:Certificate", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Certificate named %{fld3->} expired on %{fld4->} %{fld5}", processor_chain([
	dup20,
	dup12,
	setc("event_description","Certificate expired"),
	dup13,
	date_time({
		dest: "endtime",
		args: ["fld5"],
		fmts: [
			[dB,dF,dH,dc(":"),dU,dc(":"),dO,dW],
		],
	}),
]));

var msg39 = msg("Certificate", part26);

var part27 = match("MESSAGE#39:Vision", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} Vision %{event_description->} by user %{username->} via %{network_service}, source IP %{saddr}", processor_chain([
	dup20,
	dup12,
	dup13,
]));

var msg40 = msg("Vision", part27);

var part28 = match("MESSAGE#40:Updating", "nwparser.payload", "Updating policy database%{fld1}", processor_chain([
	dup25,
	dup22,
	dup12,
	setc("event_description","Updating policy database"),
]));

var msg41 = msg("Updating", part28);

var part29 = match("MESSAGE#41:Policy", "nwparser.payload", "Policy database updated successfully.%{}", processor_chain([
	dup25,
	dup24,
	dup40,
	dup12,
	setc("event_description","Policy database updated successfully"),
]));

var msg42 = msg("Policy", part29);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"Access": select12,
		"Anomalies": select7,
		"Anti-Scanning": select4,
		"Behavioral-DoS": select3,
		"COMMAND:": msg16,
		"Certificate": msg39,
		"Cold": msg31,
		"Configuration": select9,
		"Connection": msg28,
		"Cracking-Protection": select6,
		"DefensePro": msg33,
		"Device": msg29,
		"DoS": select5,
		"HttpFlood": select8,
		"Intrusions": select2,
		"Login": select10,
		"Please": msg36,
		"Policy": msg42,
		"Port": msg32,
		"Power": msg30,
		"SynFlood": msg3,
		"Updating": msg41,
		"User": select13,
		"Vision": msg40,
	}),
]);

var part30 = match("MESSAGE#0:Intrusions:01/0", "nwparser.payload", "%{fld1->} %{fld2->} %{severity->} %{id->} %{category->} \"%{event_type}\" %{protocol->} %{p0}");

var part31 = match("MESSAGE#0:Intrusions:01/1_0", "nwparser.p0", "%{saddr}:%{p0}");

var part32 = match("MESSAGE#0:Intrusions:01/1_1", "nwparser.p0", "%{saddr->} %{p0}");

var part33 = match("MESSAGE#0:Intrusions:01/2", "nwparser.p0", "%{sport->} %{p0}");

var part34 = match("MESSAGE#0:Intrusions:01/3_0", "nwparser.p0", "%{daddr}:%{p0}");

var part35 = match("MESSAGE#0:Intrusions:01/3_1", "nwparser.p0", "%{daddr->} %{p0}");

var part36 = match("MESSAGE#0:Intrusions:01/4", "nwparser.p0", "%{dport->} %{interface->} %{context->} \"%{policyname}\" %{event_state->} %{packets->} %{dclass_counter1->} %{vlan->} %{fld15->} %{fld16->} %{risk->} %{p0}");

var part37 = match("MESSAGE#0:Intrusions:01/5_0", "nwparser.p0", "%{action->} %{sigid_string}");

var part38 = match_copy("MESSAGE#0:Intrusions:01/5_1", "nwparser.p0", "action");

var part39 = match("MESSAGE#1:Intrusions:02/0", "nwparser.payload", "%{id->} %{category->} \\\"%{event_type}\\\" %{protocol->} %{p0}");

var part40 = match("MESSAGE#1:Intrusions:02/4", "nwparser.p0", "%{dport->} %{interface->} %{context->} \\\"%{policyname}\\\" %{event_state->} %{packets->} %{dclass_counter1->} %{fld1->} %{risk->} %{action->} %{vlan->} %{fld15->} %{fld16->} %{direction}");

var part41 = match("MESSAGE#22:Login:04/1_0", "nwparser.p0", "for user%{p0}");

var part42 = match("MESSAGE#22:Login:04/1_1", "nwparser.p0", "user%{p0}");

var part43 = match("MESSAGE#22:Login:04/2", "nwparser.p0", "%{} %{username->} via %{network_service->} (IP: %{saddr})%{p0}");

var part44 = match("MESSAGE#22:Login:04/3_0", "nwparser.p0", ": %{result}");

var part45 = match_copy("MESSAGE#22:Login:04/3_1", "nwparser.p0", "result");

var select14 = linear_select([
	dup2,
	dup3,
]);

var select15 = linear_select([
	dup5,
	dup6,
]);

var select16 = linear_select([
	dup8,
	dup9,
]);

var select17 = linear_select([
	dup26,
	dup27,
]);

var select18 = linear_select([
	dup29,
	dup30,
]);

var all8 = all_match({
	processors: [
		dup1,
		dup42,
		dup4,
		dup43,
		dup7,
		dup44,
	],
	on_success: processor_chain([
		dup10,
		dup11,
		dup12,
		dup13,
		dup14,
	]),
});

var all9 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup10,
		dup11,
		dup12,
		dup14,
	]),
});

var all10 = all_match({
	processors: [
		dup1,
		dup42,
		dup4,
		dup43,
		dup7,
		dup44,
	],
	on_success: processor_chain([
		dup17,
		dup11,
		dup12,
		dup13,
		dup14,
	]),
});

var all11 = all_match({
	processors: [
		dup15,
		dup42,
		dup4,
		dup43,
		dup16,
	],
	on_success: processor_chain([
		dup17,
		dup11,
		dup12,
		dup14,
	]),
});
