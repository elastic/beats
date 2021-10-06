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

var dup1 = setc("eventcategory","1603000000");

var dup2 = setf("msg","$MSG");

var dup3 = setf("event_source","hfld19");

var dup4 = date_time({
	dest: "event_time",
	args: ["hfld14","hfld15","hfld16","hfld17"],
	fmts: [
		[dW,dB,dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup5 = setc("eventcategory","1401030000");

var dup6 = setc("event_description","Authentication failure for illegal user");

var dup7 = setc("event_description","Authentication failure for user");

var dup8 = setc("eventcategory","1605000000");

var dup9 = setc("eventcategory","1601000000");

var dup10 = setc("eventcategory","1304000000");

var dup11 = setc("ec_subject","User");

var dup12 = setc("ec_theme","Authentication");

var dup13 = setc("ec_activity","Logon");

var dup14 = setc("ec_outcome","Failure");

var dup15 = setc("eventcategory","1605020000");

var dup16 = setc("ec_activity","Modify");

var dup17 = setc("ec_outcome","Success");

var dup18 = setc("eventcategory","1402020200");

var dup19 = setc("eventcategory","1402020100");

var dup20 = setc("ec_activity","Delete");

var dup21 = match_copy("MESSAGE#24:SYSTEM_MSG:08/0_1", "nwparser.payload", "event_description");

var dup22 = setc("eventcategory","1701060000");

var dup23 = setc("eventcategory","1603030000");

var dup24 = setc("eventcategory","1701030000");

var dup25 = setc("event_description","Interface is down");

var dup26 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/1_0", "nwparser.p0", "rol%{p0}");

var dup27 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/1_1", "nwparser.p0", "ol%{p0}");

var dup28 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/2", "nwparser.p0", "%{}state changed to %{result}");

var dup29 = setc("eventcategory","1701010000");

var dup30 = setc("eventcategory","1701000000");

var dup31 = setc("eventcategory","1603040000");

var dup32 = setc("eventcategory","1603010000");

var dup33 = setc("eventcategory","1603110000");

var dup34 = setc("ec_subject","NetworkComm");

var dup35 = setc("ec_theme","Communication");

var dup36 = setc("eventcategory","1801020000");

var dup37 = setc("ec_activity","Enable");

var dup38 = setc("ec_theme","Configuration");

var dup39 = setc("action","update");

var dup40 = setc("event_description","enabled telnet");

var dup41 = setc("event_description","program update");

var dup42 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/0", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:%{p0}");

var dup43 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/2", "nwparser.p0", "%{result})");

var dup44 = setc("action","Update");

var dup45 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/0", "nwparser.payload", "S%{p0}");

var dup46 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/1_0", "nwparser.p0", "ource%{p0}");

var dup47 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/1_1", "nwparser.p0", "rc%{p0}");

var dup48 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/2", "nwparser.p0", "%{}IP: %{saddr}, D%{p0}");

var dup49 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/3_0", "nwparser.p0", "estination%{p0}");

var dup50 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/3_1", "nwparser.p0", "st%{p0}");

var dup51 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/4", "nwparser.p0", "%{}IP: %{daddr}, S%{p0}");

var dup52 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/6", "nwparser.p0", "%{}Port: %{sport}, D%{p0}");

var dup53 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/8", "nwparser.p0", "%{}Port: %{dport}, S%{p0}");

var dup54 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/9_0", "nwparser.p0", "ource Interface%{p0}");

var dup55 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/9_1", "nwparser.p0", "rc Intf%{p0}");

var dup56 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/10", "nwparser.p0", ": %{sinterface}, %{p0}");

var dup57 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/11_0", "nwparser.p0", "Protocol: %{p0}");

var dup58 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/11_1", "nwparser.p0", "protocol: %{p0}");

var dup59 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/12", "nwparser.p0", "\"%{protocol}\"(%{protocol_detail}),%{space->} Hit-count = %{dclass_counter1}");

var dup60 = setc("dclass_counter1_string","Hit Count");

var dup61 = setc("eventcategory","1603100000");

var dup62 = setc("eventcategory","1701020000");

var dup63 = setc("eventcategory","1801000000");

var dup64 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/0", "nwparser.payload", "%{action}: %{p0}");

var dup65 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/1_0", "nwparser.p0", "%{saddr}@%{terminal}: %{p0}");

var dup66 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/1_1", "nwparser.p0", "%{fld1->} %{p0}");

var dup67 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/3_0", "nwparser.p0", "(%{result})%{info}");

var dup68 = match_copy("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/3_1", "nwparser.p0", "info");

var dup69 = match("MESSAGE#238:IF_XCVR_WARNING/0", "nwparser.payload", "Interface %{interface}, %{p0}");

var dup70 = match("MESSAGE#238:IF_XCVR_WARNING/1_0", "nwparser.p0", "Low %{p0}");

var dup71 = match("MESSAGE#238:IF_XCVR_WARNING/1_1", "nwparser.p0", "High %{p0}");

var dup72 = setc("ec_outcome","Error");

var dup73 = setc("eventcategory","1703000000");

var dup74 = setc("obj_type","vPC");

var dup75 = setc("ec_subject","OS");

var dup76 = setc("ec_activity","Start");

var dup77 = setc("eventcategory","1801010000");

var dup78 = setc("ec_activity","Receive");

var dup79 = setc("ec_activity","Send");

var dup80 = setc("ec_activity","Create");

var dup81 = setc("event_description","Switchover completed.");

var dup82 = setc("event_description","Invalid user");

var dup83 = setc("eventcategory","1401000000");

var dup84 = setc("ec_subject","Service");

var dup85 = setc("event_description","Duplicate address Detected.");

var dup86 = match_copy("MESSAGE#0:LOG-7-SYSTEM_MSG", "nwparser.payload", "event_description", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var dup87 = match_copy("MESSAGE#32:NEIGHBOR_UPDATE_AUTOCOPY", "nwparser.payload", "event_description", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var dup88 = match("MESSAGE#35:IF_DOWN_ADMIN_DOWN", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var dup89 = match("MESSAGE#36:IF_DOWN_ADMIN_DOWN:01", "nwparser.payload", "%{fld43->} Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var dup90 = match("MESSAGE#37:IF_DOWN_CHANNEL_MEMBERSHIP_UPDATE_IN_PROGRESS", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var dup91 = match("MESSAGE#38:IF_DOWN_INTERFACE_REMOVED", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var dup92 = linear_select([
	dup26,
	dup27,
]);

var dup93 = match_copy("MESSAGE#58:IM_SEQ_ERROR", "nwparser.payload", "result", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var dup94 = match_copy("MESSAGE#88:PFM_VEM_REMOVE_NO_HB", "nwparser.payload", "event_description", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var dup95 = match("MESSAGE#108:IF_DOWN_INITIALIZING:01", "nwparser.payload", "%{fld43->} Interface %{interface->} is down (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var dup96 = match("MESSAGE#110:IF_DOWN_NONE:01", "nwparser.payload", "%{fld52->} Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup34,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
]));

var dup97 = match_copy("MESSAGE#123:PORT_PROFILE_CHANGE_VERIFY_REQ_FAILURE", "nwparser.payload", "event_description", processor_chain([
	dup33,
	dup2,
	dup3,
	dup4,
]));

var dup98 = linear_select([
	dup46,
	dup47,
]);

var dup99 = linear_select([
	dup49,
	dup50,
]);

var dup100 = linear_select([
	dup54,
	dup55,
]);

var dup101 = linear_select([
	dup57,
	dup58,
]);

var dup102 = match_copy("MESSAGE#214:NOHMS_DIAG_ERR_PS_FAIL", "nwparser.payload", "event_description", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var dup103 = linear_select([
	dup65,
	dup66,
]);

var dup104 = linear_select([
	dup67,
	dup68,
]);

var dup105 = match("MESSAGE#224:IF_SFP_WARNING", "nwparser.payload", "Interface %{interface}, %{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var dup106 = match("MESSAGE#225:IF_DOWN_TCP_MAX_RETRANSMIT", "nwparser.payload", "%{fld43->} Interface %{interface->} is down%{info}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var dup107 = linear_select([
	dup70,
	dup71,
]);

var dup108 = match("MESSAGE#239:IF_XCVR_WARNING:01", "nwparser.payload", "Interface %{interface}, %{event_description}", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
]));

var hdr1 = match("HEADER#0:0001", "message", ": %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{hfld18}: %%{hfld19}-%{hfld20}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = match("HEADER#1:0007", "message", "%{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{hfld18}: %%{hfld19}-%{hfld20}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0007"),
]));

var hdr3 = match("HEADER#2:0005", "message", "%{hfld4->} %{hfld5->} %{hfld6->} %{hfld7->} : %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %%{hfld19}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0005"),
]));

var hdr4 = match("HEADER#3:0002", "message", ": %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %%{hfld19}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr5 = match("HEADER#4:0012", "message", "%{fld13}: %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %%{hfld19}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0012"),
]));

var hdr6 = match("HEADER#5:0008", "message", "%{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %%{hfld19}-%{severity}-%{messageid}:%{payload}", processor_chain([
	setc("header_id","0008"),
]));

var hdr7 = match("HEADER#6:0011", "message", ": %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid}[%{hfld18}]:%{payload}", processor_chain([
	setc("header_id","0011"),
]));

var hdr8 = match("HEADER#7:0003", "message", ": %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid}:%{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr9 = match("HEADER#8:0004", "message", ": %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid->} %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var hdr10 = match("HEADER#9:0009", "message", "%{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid}:%{payload}", processor_chain([
	setc("header_id","0009"),
]));

var hdr11 = match("HEADER#10:0013", "message", "%{fld13}: %{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid->} %{payload}", processor_chain([
	setc("header_id","0013"),
]));

var hdr12 = match("HEADER#11:0010", "message", "%{hfld14->} %{hfld15->} %{hfld16->} %{hfld17->} %{timezone}: %{messageid->} %{payload}", processor_chain([
	setc("header_id","0010"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
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
]);

var msg1 = msg("LOG-7-SYSTEM_MSG", dup86);

var part1 = match("MESSAGE#1:SYSTEM_MSG", "nwparser.payload", "error: PAM: Authentication failure for illegal user %{username->} from %{saddr->} - %{agent}[%{process_id}]", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup6,
]));

var msg2 = msg("SYSTEM_MSG", part1);

var part2 = match("MESSAGE#2:SYSTEM_MSG:12", "nwparser.payload", "error: PAM: Authentication failure for illegal user %{username->} from %{shost}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup6,
]));

var msg3 = msg("SYSTEM_MSG:12", part2);

var part3 = match("MESSAGE#3:SYSTEM_MSG:01", "nwparser.payload", "error: PAM: Authentication failure for %{username->} from %{saddr->} - %{agent}[%{process_id}]", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup7,
]));

var msg4 = msg("SYSTEM_MSG:01", part3);

var part4 = match("MESSAGE#4:SYSTEM_MSG:11", "nwparser.payload", "error: PAM: Authentication failure for %{username->} from %{shost}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup7,
]));

var msg5 = msg("SYSTEM_MSG:11", part4);

var part5 = match("MESSAGE#5:SYSTEM_MSG:19/0", "nwparser.payload", "error: maximum authentication attempts exceeded for %{p0}");

var part6 = match("MESSAGE#5:SYSTEM_MSG:19/1_0", "nwparser.p0", "invalid user %{username->} from %{p0}");

var part7 = match("MESSAGE#5:SYSTEM_MSG:19/1_1", "nwparser.p0", "%{username->} from %{p0}");

var select2 = linear_select([
	part6,
	part7,
]);

var part8 = match("MESSAGE#5:SYSTEM_MSG:19/2", "nwparser.p0", "%{saddr->} port %{sport->} %{protocol->} - %{agent}[%{process_id}]");

var all1 = all_match({
	processors: [
		part5,
		select2,
		part8,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
	]),
});

var msg6 = msg("SYSTEM_MSG:19", all1);

var part9 = match("MESSAGE#6:SYSTEM_MSG:02", "nwparser.payload", "error:%{result}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var msg7 = msg("SYSTEM_MSG:02", part9);

var part10 = match("MESSAGE#7:SYSTEM_MSG:03/0_0", "nwparser.payload", "(pam_unix)%{p0}");

var part11 = match("MESSAGE#7:SYSTEM_MSG:03/0_1", "nwparser.payload", "pam_unix(%{fld1}:%{fld2}):%{p0}");

var select3 = linear_select([
	part10,
	part11,
]);

var part12 = match("MESSAGE#7:SYSTEM_MSG:03/1", "nwparser.p0", "%{}authentication failure; logname=%{fld20->} uid=%{fld21->} euid=%{fld22->} tty=%{terminal->} ruser=%{fld24->} rhost=%{p0}");

var part13 = match("MESSAGE#7:SYSTEM_MSG:03/2_0", "nwparser.p0", "%{fld25->} user=%{username->} - %{p0}");

var part14 = match("MESSAGE#7:SYSTEM_MSG:03/2_1", "nwparser.p0", "%{fld25->} - %{p0}");

var select4 = linear_select([
	part13,
	part14,
]);

var part15 = match_copy("MESSAGE#7:SYSTEM_MSG:03/3", "nwparser.p0", "agent");

var all2 = all_match({
	processors: [
		select3,
		part12,
		select4,
		part15,
	],
	on_success: processor_chain([
		dup5,
		dup2,
		dup3,
		dup4,
	]),
});

var msg8 = msg("SYSTEM_MSG:03", all2);

var part16 = match("MESSAGE#8:SYSTEM_MSG:04", "nwparser.payload", "(pam_unix) %{event_description}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
]));

var msg9 = msg("SYSTEM_MSG:04", part16);

var part17 = match("MESSAGE#9:SYSTEM_MSG:05/0", "nwparser.payload", "pam_aaa:Authentication failed f%{p0}");

var part18 = match("MESSAGE#9:SYSTEM_MSG:05/1_0", "nwparser.p0", "or user %{username->} from%{p0}");

var part19 = match("MESSAGE#9:SYSTEM_MSG:05/1_1", "nwparser.p0", "rom%{p0}");

var select5 = linear_select([
	part18,
	part19,
]);

var part20 = match("MESSAGE#9:SYSTEM_MSG:05/2", "nwparser.p0", "%{} %{saddr->} - %{agent}[%{process_id}]");

var all3 = all_match({
	processors: [
		part17,
		select5,
		part20,
	],
	on_success: processor_chain([
		dup5,
		dup2,
		dup3,
		dup4,
	]),
});

var msg10 = msg("SYSTEM_MSG:05", all3);

var part21 = match("MESSAGE#10:SYSTEM_MSG:06", "nwparser.payload", "FAILED LOGIN (%{fld20}) on %{fld21->} FOR %{username}, Authentication failure - login[%{process_id}]", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
]));

var msg11 = msg("SYSTEM_MSG:06", part21);

var part22 = match("MESSAGE#11:SYSTEM_MSG:07", "nwparser.payload", "fatal:%{event_description}", processor_chain([
	dup9,
	dup2,
	dup3,
	dup4,
]));

var msg12 = msg("SYSTEM_MSG:07", part22);

var part23 = match("MESSAGE#12:SYSTEM_MSG:09", "nwparser.payload", "%{fld1}: Host name is set %{hostname->} - kernel", processor_chain([
	dup9,
	dup2,
	dup3,
	dup4,
]));

var msg13 = msg("SYSTEM_MSG:09", part23);

var part24 = match("MESSAGE#13:SYSTEM_MSG:10", "nwparser.payload", "Unauthorized access by NFS client %{saddr}.", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
]));

var msg14 = msg("SYSTEM_MSG:10", part24);

var part25 = match("MESSAGE#14:SYSTEM_MSG:13", "nwparser.payload", "%{fld43->} : SNMP UDP authentication failed for %{saddr}.", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
]));

var msg15 = msg("SYSTEM_MSG:13", part25);

var part26 = match("MESSAGE#15:SYSTEM_MSG:14", "nwparser.payload", "%{fld43->} : Subsequent authentication success for user (%{username}) failed.", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
]));

var msg16 = msg("SYSTEM_MSG:14", part26);

var part27 = match("MESSAGE#16:SYSTEM_MSG:15", "nwparser.payload", "%{fld1->} : TTY=%{terminal->} ; PWD=%{directory->} ; USER=%{username->} ; COMMAND=%{param}", processor_chain([
	dup10,
	dup2,
	dup3,
	dup4,
	dup11,
	dup12,
]));

var msg17 = msg("SYSTEM_MSG:15", part27);

var part28 = match("MESSAGE#17:SYSTEM_MSG:16", "nwparser.payload", "Login failed for user %{username->} - %{agent}[%{process_id}]", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup11,
	dup13,
	dup12,
	dup14,
]));

var msg18 = msg("SYSTEM_MSG:16", part28);

var part29 = match("MESSAGE#18:SYSTEM_MSG:17/0", "nwparser.payload", "NTP: Peer %{hostip->} %{p0}");

var part30 = match("MESSAGE#18:SYSTEM_MSG:17/1_0", "nwparser.p0", "with stratum %{fld1->} selected - %{p0}");

var part31 = match("MESSAGE#18:SYSTEM_MSG:17/1_1", "nwparser.p0", "is %{disposition->} - %{p0}");

var select6 = linear_select([
	part30,
	part31,
]);

var part32 = match("MESSAGE#18:SYSTEM_MSG:17/2", "nwparser.p0", "%{agent}[%{process_id}]");

var all4 = all_match({
	processors: [
		part29,
		select6,
		part32,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg19 = msg("SYSTEM_MSG:17", all4);

var part33 = match("MESSAGE#19:SYSTEM_MSG:20", "nwparser.payload", "New user added with username %{username->} - %{agent}", processor_chain([
	dup10,
	dup2,
	dup3,
	dup4,
	dup12,
]));

var msg20 = msg("SYSTEM_MSG:20", part33);

var part34 = match("MESSAGE#20:SYSTEM_MSG:21", "nwparser.payload", "pam_unix(%{fld1}:%{fld2}): password changed for %{username->} - %{agent}", processor_chain([
	dup10,
	dup2,
	dup3,
	dup4,
	setc("ec_subject","Password"),
	dup16,
	dup12,
	dup17,
]));

var msg21 = msg("SYSTEM_MSG:21", part34);

var part35 = match("MESSAGE#21:SYSTEM_MSG:22", "nwparser.payload", "pam_unix(%{fld1}:%{fld2}): check pass; user %{username->} - %{agent}", processor_chain([
	dup10,
	dup2,
	dup3,
	dup4,
	dup12,
]));

var msg22 = msg("SYSTEM_MSG:22", part35);

var part36 = match("MESSAGE#22:SYSTEM_MSG:23", "nwparser.payload", "new user: name=%{username}, uid=%{uid}, gid=%{fld1}, home=%{directory}, shell=%{fld2->} - %{agent}[%{process_id}]", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup11,
]));

var msg23 = msg("SYSTEM_MSG:23", part36);

var part37 = match("MESSAGE#23:SYSTEM_MSG:24/0", "nwparser.payload", "delete user %{p0}");

var part38 = match("MESSAGE#23:SYSTEM_MSG:24/1_0", "nwparser.p0", "`%{p0}");

var part39 = match("MESSAGE#23:SYSTEM_MSG:24/1_1", "nwparser.p0", "'%{p0}");

var select7 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#23:SYSTEM_MSG:24/2", "nwparser.p0", "'%{username->} - %{agent}[%{process_id}]");

var all5 = all_match({
	processors: [
		part37,
		select7,
		part40,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup11,
		dup20,
		dup17,
	]),
});

var msg24 = msg("SYSTEM_MSG:24", all5);

var part41 = match("MESSAGE#24:SYSTEM_MSG:08/0_0", "nwparser.payload", "%{event_description->} - %{agent}");

var select8 = linear_select([
	part41,
	dup21,
]);

var all6 = all_match({
	processors: [
		select8,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg25 = msg("SYSTEM_MSG:08", all6);

var select9 = linear_select([
	msg2,
	msg3,
	msg4,
	msg5,
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
	msg19,
	msg20,
	msg21,
	msg22,
	msg23,
	msg24,
	msg25,
]);

var part42 = match("MESSAGE#25:VDC_HOSTNAME_CHANGE", "nwparser.payload", "%{fld1->} hostname changed to %{hostname}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg26 = msg("VDC_HOSTNAME_CHANGE", part42);

var part43 = match("MESSAGE#26:POLICY_ACTIVATE_EVENT", "nwparser.payload", "Policy %{policyname->} is activated by profile %{username}", processor_chain([
	dup22,
	dup2,
	dup3,
	dup4,
	setc("action","activated"),
	setc("event_description","Policy is activated by profile"),
]));

var msg27 = msg("POLICY_ACTIVATE_EVENT", part43);

var part44 = match("MESSAGE#27:POLICY_COMMIT_EVENT", "nwparser.payload", "Commit operation %{disposition}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg28 = msg("POLICY_COMMIT_EVENT", part44);

var part45 = match("MESSAGE#28:POLICY_DEACTIVATE_EVENT", "nwparser.payload", "Policy %{policyname->} is de-activated by last referring profile %{username}", processor_chain([
	setc("eventcategory","1701070000"),
	dup2,
	dup3,
	dup4,
	setc("action","de-activated"),
	setc("event_description","Policy is de-activated by last referring profile"),
]));

var msg29 = msg("POLICY_DEACTIVATE_EVENT", part45);

var part46 = match("MESSAGE#29:POLICY_LOOKUP_EVENT:01", "nwparser.payload", "policy=%{policyname->} rule=%{rulename->} action=%{action->} direction=%{direction->} src.net.ip-address=%{saddr->} src.net.port=%{sport->} dst.net.ip-address=%{daddr->} dst.net.port=%{dport->} net.protocol=%{protocol->} net.ethertype=%{fld2->} dst.zone.name=%{dst_zone->} src.zone.name=%{src_zone}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg30 = msg("POLICY_LOOKUP_EVENT:01", part46);

var part47 = match("MESSAGE#30:POLICY_LOOKUP_EVENT", "nwparser.payload", "policy=%{policyname->} rule=%{rulename->} action=%{action->} direction=%{direction->} src.net.ip-address=%{saddr->} src.net.port=%{sport->} dst.net.ip-address=%{daddr->} dst.net.port=%{dport->} net.protocol=%{protocol->} net.ethertype=%{fld2}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg31 = msg("POLICY_LOOKUP_EVENT", part47);

var part48 = match("MESSAGE#31:POLICY_LOOKUP_EVENT:02", "nwparser.payload", "policy=%{policyname->} rule=%{rulename->} action=%{action->} direction=%{direction->} net.ethertype=%{fld2}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg32 = msg("POLICY_LOOKUP_EVENT:02", part48);

var select10 = linear_select([
	msg30,
	msg31,
	msg32,
]);

var msg33 = msg("NEIGHBOR_UPDATE_AUTOCOPY", dup87);

var msg34 = msg("MTSERROR", dup86);

var part49 = match("MESSAGE#34:IF_DOWN_ERROR_DISABLED", "nwparser.payload", "Interface %{interface->} is down (Error disabled. Reason:%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg35 = msg("IF_DOWN_ERROR_DISABLED", part49);

var msg36 = msg("IF_DOWN_ADMIN_DOWN", dup88);

var msg37 = msg("IF_DOWN_ADMIN_DOWN:01", dup89);

var select11 = linear_select([
	msg36,
	msg37,
]);

var msg38 = msg("IF_DOWN_CHANNEL_MEMBERSHIP_UPDATE_IN_PROGRESS", dup90);

var msg39 = msg("IF_DOWN_INTERFACE_REMOVED", dup91);

var part50 = match("MESSAGE#39:IF_DOWN_LINK_FAILURE", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
	dup25,
]));

var msg40 = msg("IF_DOWN_LINK_FAILURE", part50);

var msg41 = msg("IF_DOWN_LINK_FAILURE:01", dup89);

var select12 = linear_select([
	msg40,
	msg41,
]);

var msg42 = msg("IF_DOWN_MODULE_REMOVED", dup91);

var msg43 = msg("IF_DOWN_PORT_CHANNEL_MEMBERS_DOWN", dup88);

var part51 = match("MESSAGE#43:IF_DUPLEX", "nwparser.payload", "Interface %{interface}, operational duplex mode changed to %{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface duplex mode changed"),
]));

var msg44 = msg("IF_DUPLEX", part51);

var part52 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/0", "nwparser.payload", "Interface %{interface}, operational Receive Flow Cont%{p0}");

var all7 = all_match({
	processors: [
		part52,
		dup92,
		dup28,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		setc("event_description","Interface operational Receive Flow Control state changed"),
	]),
});

var msg45 = msg("IF_RX_FLOW_CONTROL", all7);

var part53 = match_copy("MESSAGE#45:IF_SEQ_ERROR", "nwparser.payload", "result", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg46 = msg("IF_SEQ_ERROR", part53);

var part54 = match("MESSAGE#46:IF_TX_FLOW_CONTROL/0", "nwparser.payload", "Interface %{interface}, operational Transmit Flow Cont%{p0}");

var all8 = all_match({
	processors: [
		part54,
		dup92,
		dup28,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		setc("event_description","Interface operational Transmit Flow Control state changed"),
	]),
});

var msg47 = msg("IF_TX_FLOW_CONTROL", all8);

var part55 = match("MESSAGE#47:IF_UP", "nwparser.payload", "%{fld43->} Interface %{sinterface->} is up in mode %{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface is up in mode"),
]));

var msg48 = msg("IF_UP", part55);

var part56 = match("MESSAGE#48:IF_UP:01", "nwparser.payload", "Interface %{sinterface->} is up", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface is up"),
]));

var msg49 = msg("IF_UP:01", part56);

var select13 = linear_select([
	msg48,
	msg49,
]);

var part57 = match("MESSAGE#49:SPEED", "nwparser.payload", "Interface %{interface}, operational speed changed to %{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface operational speed changed"),
]));

var msg50 = msg("SPEED", part57);

var part58 = match("MESSAGE#50:CREATED", "nwparser.payload", "%{group_object->} created", processor_chain([
	dup29,
	dup2,
	dup3,
	dup4,
]));

var msg51 = msg("CREATED", part58);

var part59 = match("MESSAGE#51:FOP_CHANGED", "nwparser.payload", "%{group_object}: first operational port changed from %{change_old->} to %{change_new}", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
]));

var msg52 = msg("FOP_CHANGED", part59);

var part60 = match("MESSAGE#52:PORT_DOWN", "nwparser.payload", "%{group_object}: %{interface->} is down", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg53 = msg("PORT_DOWN", part60);

var part61 = match("MESSAGE#53:PORT_UP", "nwparser.payload", "%{group_object}: %{interface->} is up", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg54 = msg("PORT_UP", part61);

var part62 = match("MESSAGE#54:SUBGROUP_ID_PORT_ADDED", "nwparser.payload", "Interface %{interface->} is added to %{group_object->} with subgroup id %{fld20}", processor_chain([
	dup29,
	dup2,
	dup3,
	dup4,
]));

var msg55 = msg("SUBGROUP_ID_PORT_ADDED", part62);

var part63 = match("MESSAGE#55:SUBGROUP_ID_PORT_REMOVED", "nwparser.payload", "Interface %{interface->} is removed from %{group_object->} with subgroup id %{fld20}", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg56 = msg("SUBGROUP_ID_PORT_REMOVED", part63);

var msg57 = msg("MTS_DROP", dup87);

var msg58 = msg("SYSLOG_LOG_WARNING", dup87);

var msg59 = msg("IM_SEQ_ERROR", dup93);

var msg60 = msg("ADDON_IMG_DNLD_COMPLETE", dup87);

var msg61 = msg("ADDON_IMG_DNLD_STARTED", dup87);

var msg62 = msg("ADDON_IMG_DNLD_SUCCESSFUL", dup87);

var msg63 = msg("IMG_DNLD_COMPLETE", dup87);

var msg64 = msg("IMG_DNLD_STARTED", dup87);

var part64 = match_copy("MESSAGE#64:PORT_SOFTWARE_FAILURE", "nwparser.payload", "result", processor_chain([
	dup31,
	dup2,
	dup3,
	dup4,
]));

var msg65 = msg("PORT_SOFTWARE_FAILURE", part64);

var msg66 = msg("MSM_CRIT", dup93);

var part65 = match("MESSAGE#66:LOG_CMP_AAA_FAILURE", "nwparser.payload", "Authentication failed for a login from %{shost->} (%{result})", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup7,
]));

var msg67 = msg("LOG_CMP_AAA_FAILURE", part65);

var msg68 = msg("LOG_LIC_N1K_EXPIRY_WARNING", dup87);

var part66 = match("MESSAGE#68:MOD_FAIL", "nwparser.payload", "Initialization of module %{fld20->} (serial: %{serial_number}) failed", processor_chain([
	dup32,
	dup2,
	dup3,
	dup4,
]));

var msg69 = msg("MOD_FAIL", part66);

var part67 = match("MESSAGE#69:MOD_MAJORSWFAIL", "nwparser.payload", "Module %{fld20->} (serial: %{serial_number}) reported a critical failure in service %{fld22}", processor_chain([
	dup33,
	dup2,
	dup3,
	dup4,
]));

var msg70 = msg("MOD_MAJORSWFAIL", part67);

var part68 = match("MESSAGE#70:MOD_SRG_NOT_COMPATIBLE", "nwparser.payload", "Module %{fld20->} (serial: %{serial_number}) firmware is not compatible with supervisor, downloading new image", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg71 = msg("MOD_SRG_NOT_COMPATIBLE", part68);

var part69 = match("MESSAGE#71:MOD_WARNING:01", "nwparser.payload", "Module %{fld20->} (serial: %{serial_number}) reported warnings on %{info->} due to %{result->} in device %{fld23->} (device error %{fld22})", processor_chain([
	dup32,
	dup2,
	dup3,
	dup4,
]));

var msg72 = msg("MOD_WARNING:01", part69);

var part70 = match("MESSAGE#72:MOD_WARNING", "nwparser.payload", "Module %{fld20->} (serial: %{serial_number}) reported warning %{info->} due to %{result->} in device %{fld23->} (device error %{fld22})", processor_chain([
	dup32,
	dup2,
	dup3,
	dup4,
]));

var msg73 = msg("MOD_WARNING", part70);

var select14 = linear_select([
	msg72,
	msg73,
]);

var part71 = match("MESSAGE#73:ACTIVE_SUP_OK", "nwparser.payload", "Supervisor %{fld20->} is active (serial: %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg74 = msg("ACTIVE_SUP_OK", part71);

var part72 = match("MESSAGE#74:MOD_OK", "nwparser.payload", "Module %{fld20->} is online (serial: %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg75 = msg("MOD_OK", part72);

var part73 = match("MESSAGE#75:MOD_RESTART", "nwparser.payload", "Module %{fld20->} is restarting after image download", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg76 = msg("MOD_RESTART", part73);

var part74 = match("MESSAGE#76:DISPUTE_CLEARED", "nwparser.payload", "Dispute resolved for port %{portname->} on %{vlan}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("event_description","Dispute resolved for port on VLAN"),
]));

var msg77 = msg("DISPUTE_CLEARED", part74);

var part75 = match("MESSAGE#77:DISPUTE_DETECTED", "nwparser.payload", "Dispute detected on port %{portname->} on %{vlan}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("event_description","Dispute detected on port on VLAN"),
]));

var msg78 = msg("DISPUTE_DETECTED", part75);

var msg79 = msg("DOMAIN_CFG_SYNC_DONE", dup87);

var msg80 = msg("CHASSIS_CLKMODOK", dup87);

var msg81 = msg("CHASSIS_CLKSRC", dup87);

var msg82 = msg("FAN_OK", dup87);

var part76 = match("MESSAGE#82:MOD_DETECT", "nwparser.payload", "Module %{fld19->} detected (Serial number %{serial_number}) Module-Type %{fld20->} Model %{fld21}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg83 = msg("MOD_DETECT", part76);

var part77 = match("MESSAGE#83:MOD_PWRDN", "nwparser.payload", "Module %{fld19->} powered down (Serial number %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg84 = msg("MOD_PWRDN", part77);

var part78 = match("MESSAGE#84:MOD_PWRUP", "nwparser.payload", "Module %{fld19->} powered up (Serial number %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg85 = msg("MOD_PWRUP", part78);

var part79 = match("MESSAGE#85:MOD_REMOVE", "nwparser.payload", "Module %{fld19->} removed (Serial number %{serial_number})", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg86 = msg("MOD_REMOVE", part79);

var msg87 = msg("PFM_MODULE_POWER_ON", dup87);

var msg88 = msg("PFM_SYSTEM_RESET", dup87);

var msg89 = msg("PFM_VEM_REMOVE_NO_HB", dup94);

var msg90 = msg("PFM_VEM_REMOVE_RESET", dup94);

var msg91 = msg("PFM_VEM_REMOVE_STATE_CONFLICT", dup94);

var msg92 = msg("PFM_VEM_REMOVE_TWO_ACT_VSM", dup94);

var msg93 = msg("PFM_VEM_UNLICENSED", dup87);

var msg94 = msg("PS_FANOK", dup87);

var part80 = match("MESSAGE#94:PS_OK", "nwparser.payload", "Power supply %{fld19->} ok (Serial number %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg95 = msg("PS_OK", part80);

var part81 = match_copy("MESSAGE#95:MOD_BRINGUP_MULTI_LIMIT", "nwparser.payload", "event_description", processor_chain([
	dup31,
	dup2,
	dup3,
	dup4,
]));

var msg96 = msg("MOD_BRINGUP_MULTI_LIMIT", part81);

var part82 = match("MESSAGE#96:FAN_DETECT", "nwparser.payload", "Fan module %{fld19->} (Serial number %{serial_number}) %{fld20->} detected", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg97 = msg("FAN_DETECT", part82);

var msg98 = msg("MOD_STATUS", dup87);

var part83 = match("MESSAGE#98:PEER_VPC_CFGD_VLANS_CHANGED", "nwparser.payload", "Peer vPC %{obj_name->} configured vlans changed", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Peer vPC configured vlans changed"),
]));

var msg99 = msg("PEER_VPC_CFGD_VLANS_CHANGED", part83);

var part84 = match("MESSAGE#99:PEER_VPC_DELETED", "nwparser.payload", "Peer vPC %{obj_name->} deleted", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg100 = msg("PEER_VPC_DELETED", part84);

var msg101 = msg("PFM_VEM_DETECTED", dup87);

var part85 = match("MESSAGE#101:PS_FOUND", "nwparser.payload", "Power supply %{fld19->} found (Serial number %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg102 = msg("PS_FOUND", part85);

var part86 = match("MESSAGE#102:PS_STATUS/0_0", "nwparser.payload", "PowerSupply %{fld1->} current-status is %{disposition}");

var select15 = linear_select([
	part86,
	dup21,
]);

var all9 = all_match({
	processors: [
		select15,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg103 = msg("PS_STATUS", all9);

var part87 = match("MESSAGE#103:PS_CAPACITY_CHANGE:01", "nwparser.payload", "Power supply %{fld1->} changed its capacity. possibly due to On/Off or power cable removal/insertion (Serial number %{serial_number})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg104 = msg("PS_CAPACITY_CHANGE:01", part87);

var msg105 = msg("PS_CAPACITY_CHANGE", dup87);

var select16 = linear_select([
	msg104,
	msg105,
]);

var msg106 = msg("IF_DOWN_FCOT_NOT_PRESENT", dup88);

var msg107 = msg("IF_DOWN_FCOT_NOT_PRESENT:01", dup89);

var select17 = linear_select([
	msg106,
	msg107,
]);

var msg108 = msg("IF_DOWN_INITIALIZING", dup90);

var msg109 = msg("IF_DOWN_INITIALIZING:01", dup95);

var select18 = linear_select([
	msg108,
	msg109,
]);

var part88 = match("MESSAGE#109:IF_DOWN_NONE", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup34,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
]));

var msg110 = msg("IF_DOWN_NONE", part88);

var msg111 = msg("IF_DOWN_NONE:01", dup96);

var select19 = linear_select([
	msg110,
	msg111,
]);

var msg112 = msg("IF_DOWN_NOS_RCVD", dup88);

var msg113 = msg("IF_DOWN_NOS_RCVD:01", dup89);

var select20 = linear_select([
	msg112,
	msg113,
]);

var msg114 = msg("IF_DOWN_OFFLINE", dup88);

var msg115 = msg("IF_DOWN_OLS_RCVD", dup88);

var part89 = match("MESSAGE#115:IF_DOWN_SOFTWARE_FAILURE", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup31,
	dup2,
	dup3,
	dup4,
]));

var msg116 = msg("IF_DOWN_SOFTWARE_FAILURE", part89);

var msg117 = msg("IF_DOWN_SRC_PORT_NOT_BOUND", dup90);

var part90 = match("MESSAGE#117:IF_TRUNK_DOWN", "nwparser.payload", "Interface %{interface}, vsan %{fld20->} is down (%{info})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg118 = msg("IF_TRUNK_DOWN", part90);

var part91 = match("MESSAGE#118:IF_TRUNK_DOWN:01", "nwparser.payload", "Interface %{interface}, vlan %{vlan->} down", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg119 = msg("IF_TRUNK_DOWN:01", part91);

var part92 = match("MESSAGE#119:IF_TRUNK_DOWN:02", "nwparser.payload", "%{fld43->} Interface %{interface}, vsan %{vlan->} is down %{info}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg120 = msg("IF_TRUNK_DOWN:02", part92);

var select21 = linear_select([
	msg118,
	msg119,
	msg120,
]);

var part93 = match("MESSAGE#120:IF_TRUNK_UP", "nwparser.payload", "Interface %{interface}, vsan %{fld20->} is up", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg121 = msg("IF_TRUNK_UP", part93);

var part94 = match("MESSAGE#121:IF_TRUNK_UP:01", "nwparser.payload", "Interface %{interface}, vlan %{vlan->} up", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg122 = msg("IF_TRUNK_UP:01", part94);

var part95 = match("MESSAGE#122:IF_TRUNK_UP:02", "nwparser.payload", "%{fld43->} Interface %{interface}, vsan %{vlan->} is up %{info}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg123 = msg("IF_TRUNK_UP:02", part95);

var select22 = linear_select([
	msg121,
	msg122,
	msg123,
]);

var msg124 = msg("PORT_PROFILE_CHANGE_VERIFY_REQ_FAILURE", dup97);

var part96 = match("MESSAGE#124:IF_PORTPROFILE_ATTACHED", "nwparser.payload", "Interface %{interface->} is inheriting port-profile %{fld20}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg125 = msg("IF_PORTPROFILE_ATTACHED", part96);

var msg126 = msg("STANDBY_SUP_OK", dup87);

var part97 = match("MESSAGE#126:STM_LOOP_DETECT", "nwparser.payload", "Loops detected in the network among ports %{portname->} and %{info->} vlan %{vlan->} - %{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Loops detected in the network among ports"),
]));

var msg127 = msg("STM_LOOP_DETECT", part97);

var part98 = match("MESSAGE#127:SYNC_COMPLETE", "nwparser.payload", "Sync completed.%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg128 = msg("SYNC_COMPLETE", part98);

var msg129 = msg("PVLAN_PPM_PORT_CONFIG_FAILED", dup97);

var msg130 = msg("MESG", dup87);

var part99 = match("MESSAGE#130:ERR_MSG", "nwparser.payload", "ERROR:%{result}", processor_chain([
	dup33,
	dup2,
	dup3,
	dup4,
]));

var msg131 = msg("ERR_MSG", part99);

var msg132 = msg("RM_VICPP_RECREATE_ERROR", dup97);

var part100 = match("MESSAGE#132:CFGWRITE_ABORTED_LOCK", "nwparser.payload", "Unable to lock the configuration (error-id %{resultcode}). Aborting configuration copy.", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg133 = msg("CFGWRITE_ABORTED_LOCK", part100);

var part101 = match("MESSAGE#133:CFGWRITE_FAILED", "nwparser.payload", "Configuration copy failed (error-id %{resultcode}).", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg134 = msg("CFGWRITE_FAILED", part101);

var msg135 = msg("CFGWRITE_ABORTED", dup87);

var msg136 = msg("CFGWRITE_DONE", dup87);

var part102 = match("MESSAGE#136:CFGWRITE_STARTED/0_0", "nwparser.payload", "%{event_description->} (PID %{process_id}).");

var select23 = linear_select([
	part102,
	dup21,
]);

var all10 = all_match({
	processors: [
		select23,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg137 = msg("CFGWRITE_STARTED", all10);

var msg138 = msg("IF_ATTACHED", dup87);

var msg139 = msg("IF_DELETE_AUTO", dup94);

var part103 = match("MESSAGE#139:IF_DETACHED", "nwparser.payload", "Interface %{interface->} is detached", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg140 = msg("IF_DETACHED", part103);

var msg141 = msg("IF_DETACHED_MODULE_REMOVED", dup94);

var msg142 = msg("IF_DOWN_INACTIVE", dup88);

var msg143 = msg("IF_DOWN_NON_PARTICIPATING", dup88);

var part104 = match("MESSAGE#143:IF_DOWN_VEM_UNLICENSED", "nwparser.payload", "Interface %{interface->} is down", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg144 = msg("IF_DOWN_VEM_UNLICENSED", part104);

var part105 = match("MESSAGE#144:CONN_CONNECT", "nwparser.payload", "Connection %{hostname->} connected to the vCenter Server.", processor_chain([
	dup36,
	dup2,
	dup3,
	dup4,
]));

var msg145 = msg("CONN_CONNECT", part105);

var part106 = match("MESSAGE#145:CONN_DISCONNECT", "nwparser.payload", "Connection %{hostname->} disconnected from the vCenter Server.", processor_chain([
	setc("eventcategory","1801030000"),
	dup2,
	dup3,
	dup4,
]));

var msg146 = msg("CONN_DISCONNECT", part106);

var part107 = match("MESSAGE#146:DVPG_CREATE", "nwparser.payload", "created port-group %{info->} on the vCenter Server.", processor_chain([
	dup29,
	dup2,
	dup3,
	dup4,
]));

var msg147 = msg("DVPG_CREATE", part107);

var part108 = match("MESSAGE#147:DVPG_DELETE", "nwparser.payload", "deleted port-group %{info->} from the vCenter Server.", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg148 = msg("DVPG_DELETE", part108);

var msg149 = msg("DVS_HOSTMEMBER_INFO", dup87);

var part109 = match("MESSAGE#149:DVS_NAME_CHANGE", "nwparser.payload", "Changed dvswitch name to %{info->} on the vCenter Server.", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg150 = msg("DVS_NAME_CHANGE", part109);

var msg151 = msg("VMS_PPM_SYNC_COMPLETE", dup87);

var part110 = match("MESSAGE#151:VPC_DELETED", "nwparser.payload", "vPC %{obj_name->} is deleted", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg152 = msg("VPC_DELETED", part110);

var part111 = match("MESSAGE#152:VPC_UP", "nwparser.payload", "vPC %{obj_name->} is up", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("event_description","VPC is up"),
]));

var msg153 = msg("VPC_UP", part111);

var part112 = match("MESSAGE#153:VSHD_SYSLOG_CONFIG_I/0", "nwparser.payload", "Configured from vty by %{username->} on %{p0}");

var part113 = match("MESSAGE#153:VSHD_SYSLOG_CONFIG_I/1_0", "nwparser.p0", "%{saddr}@%{terminal}");

var part114 = match_copy("MESSAGE#153:VSHD_SYSLOG_CONFIG_I/1_1", "nwparser.p0", "saddr");

var select24 = linear_select([
	part113,
	part114,
]);

var all11 = all_match({
	processors: [
		part112,
		select24,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg154 = msg("VSHD_SYSLOG_CONFIG_I", all11);

var part115 = match("MESSAGE#154:VSHD_SYSLOG_CONFIG_I:01", "nwparser.payload", "Configuring console from %{fld43->} %{saddr}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg155 = msg("VSHD_SYSLOG_CONFIG_I:01", part115);

var select25 = linear_select([
	msg154,
	msg155,
]);

var part116 = match("MESSAGE#155:AAA_ACCOUNTING_MESSAGE:18", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:%{event_description}; feature %{protocol->} (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg156 = msg("AAA_ACCOUNTING_MESSAGE:18", part116);

var part117 = match("MESSAGE#156:AAA_ACCOUNTING_MESSAGE:17", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:enabled telnet", processor_chain([
	dup22,
	dup37,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	dup39,
	dup40,
]));

var msg157 = msg("AAA_ACCOUNTING_MESSAGE:17", part117);

var part118 = match("MESSAGE#157:AAA_ACCOUNTING_MESSAGE", "nwparser.payload", "start:%{saddr}@%{application}:%{username}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","program start"),
]));

var msg158 = msg("AAA_ACCOUNTING_MESSAGE", part118);

var part119 = match("MESSAGE#158:AAA_ACCOUNTING_MESSAGE:08", "nwparser.payload", "start:snmp_%{fld43}_%{saddr}:%{username}:", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg159 = msg("AAA_ACCOUNTING_MESSAGE:08", part119);

var part120 = match("MESSAGE#159:AAA_ACCOUNTING_MESSAGE:03", "nwparser.payload", "start:%{saddr}(%{terminal}):%{username}:", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg160 = msg("AAA_ACCOUNTING_MESSAGE:03", part120);

var part121 = match("MESSAGE#160:AAA_ACCOUNTING_MESSAGE:19", "nwparser.payload", "start:%{fld40}:%{username}:", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg161 = msg("AAA_ACCOUNTING_MESSAGE:19", part121);

var part122 = match("MESSAGE#161:AAA_ACCOUNTING_MESSAGE:22", "nwparser.payload", "update:::added user %{username}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
]));

var msg162 = msg("AAA_ACCOUNTING_MESSAGE:22", part122);

var part123 = match("MESSAGE#162:AAA_ACCOUNTING_MESSAGE:23", "nwparser.payload", "update:::%{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg163 = msg("AAA_ACCOUNTING_MESSAGE:23", part123);

var part124 = match("MESSAGE#163:AAA_ACCOUNTING_MESSAGE:11", "nwparser.payload", "update:snmp_%{fld43}_%{saddr}:%{username}:target (name:%{dhost->} address:%{daddr}:%{dport}) deleted", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg164 = msg("AAA_ACCOUNTING_MESSAGE:11", part124);

var part125 = match("MESSAGE#164:AAA_ACCOUNTING_MESSAGE:12", "nwparser.payload", "update:snmp_%{fld43}_%{saddr}:%{username}:target (name:%{dhost->} address:%{daddr}:%{dport->} timeout:%{fld44->} retry:%{fld45->} tagList:trap params:%{fld46}) added", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg165 = msg("AAA_ACCOUNTING_MESSAGE:12", part125);

var part126 = match("MESSAGE#165:AAA_ACCOUNTING_MESSAGE:13", "nwparser.payload", "update:snmp_%{fld43}_%{saddr}:%{username}:Interface %{interface->} state updated to up", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg166 = msg("AAA_ACCOUNTING_MESSAGE:13", part126);

var part127 = match("MESSAGE#166:AAA_ACCOUNTING_MESSAGE:14", "nwparser.payload", "update:snmp_%{fld43}_%{saddr}:%{username}:Interface %{interface->} state updated to down", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg167 = msg("AAA_ACCOUNTING_MESSAGE:14", part127);

var part128 = match("MESSAGE#167:AAA_ACCOUNTING_MESSAGE:15", "nwparser.payload", "update:snmp_%{fld43}_%{saddr}:%{username}:Performing configuration copy.", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg168 = msg("AAA_ACCOUNTING_MESSAGE:15", part128);

var part129 = match("MESSAGE#168:AAA_ACCOUNTING_MESSAGE:16", "nwparser.payload", "update:%{saddr}@%{application}:%{username}:terminal length %{dclass_counter1->} (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	dup41,
]));

var msg169 = msg("AAA_ACCOUNTING_MESSAGE:16", part129);

var part130 = match("MESSAGE#169:AAA_ACCOUNTING_MESSAGE:04", "nwparser.payload", "update:%{saddr}(%{fld3}):%{username}:terminal length %{fld5}:%{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg170 = msg("AAA_ACCOUNTING_MESSAGE:04", part130);

var part131 = match("MESSAGE#170:AAA_ACCOUNTING_MESSAGE:01", "nwparser.payload", "update:%{saddr}@%{terminal}:%{application}:terminal width %{dclass_counter1->} (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	dup41,
]));

var msg171 = msg("AAA_ACCOUNTING_MESSAGE:01", part131);

var part132 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/1_0", "nwparser.p0", "configure terminal ; ntp source-interface %{sinterface->} (%{p0}");

var part133 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/1_1", "nwparser.p0", "show ntp statistics peer ipaddr %{hostip->} (%{p0}");

var select26 = linear_select([
	part132,
	part133,
]);

var all12 = all_match({
	processors: [
		dup42,
		select26,
		dup43,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		dup44,
	]),
});

var msg172 = msg("AAA_ACCOUNTING_MESSAGE:27", all12);

var part134 = match("MESSAGE#172:AAA_ACCOUNTING_MESSAGE:28/1_0", "nwparser.p0", "clock set %{event_time_string->} (%{p0}");

var part135 = match("MESSAGE#172:AAA_ACCOUNTING_MESSAGE:28/1_1", "nwparser.p0", "show logging last %{fld1->} (%{p0}");

var select27 = linear_select([
	part134,
	part135,
]);

var all13 = all_match({
	processors: [
		dup42,
		select27,
		dup43,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		dup44,
	]),
});

var msg173 = msg("AAA_ACCOUNTING_MESSAGE:28", all13);

var part136 = match("MESSAGE#173:AAA_ACCOUNTING_MESSAGE:20", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:%{info->} (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg174 = msg("AAA_ACCOUNTING_MESSAGE:20", part136);

var part137 = match("MESSAGE#174:AAA_ACCOUNTING_MESSAGE:30", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:added user %{c_username}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup11,
	dup17,
	setc("event_description","Added user"),
	dup44,
]));

var msg175 = msg("AAA_ACCOUNTING_MESSAGE:30", part137);

var part138 = match("MESSAGE#175:AAA_ACCOUNTING_MESSAGE:29", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:deleted user %{c_username}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup11,
	dup17,
	setc("event_description","Deleted user"),
	dup44,
]));

var msg176 = msg("AAA_ACCOUNTING_MESSAGE:29", part138);

var part139 = match("MESSAGE#176:AAA_ACCOUNTING_MESSAGE:21", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:%{info}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg177 = msg("AAA_ACCOUNTING_MESSAGE:21", part139);

var part140 = match("MESSAGE#177:AAA_ACCOUNTING_MESSAGE:07", "nwparser.payload", "update:%{saddr}(%{fld3}):%{username}:terminal width %{dclass_counter1}:%{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg178 = msg("AAA_ACCOUNTING_MESSAGE:07", part140);

var part141 = match("MESSAGE#178:AAA_ACCOUNTING_MESSAGE:05", "nwparser.payload", "update:%{saddr}(%{fld3}):%{username}:terminal session-timeout %{fld5}:%{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg179 = msg("AAA_ACCOUNTING_MESSAGE:05", part141);

var part142 = match("MESSAGE#179:AAA_ACCOUNTING_MESSAGE:10", "nwparser.payload", "update:%{saddr}(%{fld3}):%{username}:copy %{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg180 = msg("AAA_ACCOUNTING_MESSAGE:10", part142);

var part143 = match("MESSAGE#180:AAA_ACCOUNTING_MESSAGE:24", "nwparser.payload", "update:%{terminal}:%{username}: %{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg181 = msg("AAA_ACCOUNTING_MESSAGE:24", part143);

var part144 = match("MESSAGE#181:AAA_ACCOUNTING_MESSAGE:06", "nwparser.payload", "stop:%{saddr}(%{fld3}):%{username}:shell terminated", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg182 = msg("AAA_ACCOUNTING_MESSAGE:06", part144);

var part145 = match("MESSAGE#182:AAA_ACCOUNTING_MESSAGE:02", "nwparser.payload", "stop:%{saddr}@%{terminal}:%{username}:shell %{result}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","shell terminated"),
]));

var msg183 = msg("AAA_ACCOUNTING_MESSAGE:02", part145);

var part146 = match("MESSAGE#183:AAA_ACCOUNTING_MESSAGE:25", "nwparser.payload", "stop:%{saddr}@%{terminal}:%{username}:%{fld40}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg184 = msg("AAA_ACCOUNTING_MESSAGE:25", part146);

var part147 = match("MESSAGE#184:AAA_ACCOUNTING_MESSAGE:09", "nwparser.payload", "stop:snmp_%{fld43}_%{saddr}:%{username}:", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg185 = msg("AAA_ACCOUNTING_MESSAGE:09", part147);

var part148 = match("MESSAGE#185:AAA_ACCOUNTING_MESSAGE:26", "nwparser.payload", "stop:%{terminal}:%{username}:", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg186 = msg("AAA_ACCOUNTING_MESSAGE:26", part148);

var select28 = linear_select([
	msg156,
	msg157,
	msg158,
	msg159,
	msg160,
	msg161,
	msg162,
	msg163,
	msg164,
	msg165,
	msg166,
	msg167,
	msg168,
	msg169,
	msg170,
	msg171,
	msg172,
	msg173,
	msg174,
	msg175,
	msg176,
	msg177,
	msg178,
	msg179,
	msg180,
	msg181,
	msg182,
	msg183,
	msg184,
	msg185,
	msg186,
]);

var all14 = all_match({
	processors: [
		dup45,
		dup98,
		dup48,
		dup99,
		dup51,
		dup98,
		dup52,
		dup99,
		dup53,
		dup100,
		dup56,
		dup101,
		dup59,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		setc("event_description","ACL Log Flow Interval"),
		dup60,
	]),
});

var msg187 = msg("ACLLOG_FLOW_INTERVAL", all14);

var part149 = match("MESSAGE#187:ACLLOG_MAXFLOW_REACHED", "nwparser.payload", "Maximum limit %{fld3->} reached for number of flows", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg188 = msg("ACLLOG_MAXFLOW_REACHED", part149);

var all15 = all_match({
	processors: [
		dup45,
		dup98,
		dup48,
		dup99,
		dup51,
		dup98,
		dup52,
		dup99,
		dup53,
		dup100,
		dup56,
		dup101,
		dup59,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
		setc("event_description","ACL Lof New Flow"),
		dup60,
	]),
});

var msg189 = msg("ACLLOG_NEW_FLOW", all15);

var part150 = match("MESSAGE#189:DUP_VADDR_SRC_IP", "nwparser.payload", "%{process->} [%{process_id}] Source address of packet received from %{smacaddr->} on %{vlan}(%{interface}) is duplicate of local virtual ip, %{saddr}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","Source address of packet received on vlan is duplicate of local virtual ip"),
]));

var msg190 = msg("DUP_VADDR_SRC_IP", part150);

var part151 = match("MESSAGE#190:IF_ERROR_VLANS_REMOVED", "nwparser.payload", "VLANs %{vlan->} on Interface %{sinterface->} are removed from suspended state.", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg191 = msg("IF_ERROR_VLANS_REMOVED", part151);

var part152 = match("MESSAGE#191:IF_ERROR_VLANS_SUSPENDED", "nwparser.payload", "VLANs %{vlan->} on Interface %{sinterface->} are being suspended. (Reason: %{info})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg192 = msg("IF_ERROR_VLANS_SUSPENDED", part152);

var part153 = match("MESSAGE#192:IF_DOWN_CFG_CHANGE", "nwparser.payload", "Interface %{sinterface->} is down(%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg193 = msg("IF_DOWN_CFG_CHANGE", part153);

var part154 = match("MESSAGE#193:PFM_CLOCK_CHANGE", "nwparser.payload", "Clock setting has been changed on the system. Please be aware that clock changes will force a recheckout of all existing VEM licenses. During this recheckout procedure, licensed VEMs which are offline will lose their licenses.%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg194 = msg("PFM_CLOCK_CHANGE", part154);

var part155 = match("MESSAGE#194:SYNC_FAILURE_STANDBY_RESET", "nwparser.payload", "Failure in syncing messages to standby for vdc %{fld3->} causing standby to reset.", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg195 = msg("SYNC_FAILURE_STANDBY_RESET", part155);

var part156 = match("MESSAGE#195:snmpd", "nwparser.payload", "snmp_pss_snapshot : Copying local engine DB PSS file to url%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg196 = msg("snmpd", part156);

var part157 = match("MESSAGE#196:snmpd:01", "nwparser.payload", "SNMPD_SYSLOG_CONFIG_I: Configuration update from %{fld43}_%{saddr->} %{info}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg197 = msg("snmpd:01", part157);

var select29 = linear_select([
	msg196,
	msg197,
]);

var part158 = match("MESSAGE#197:CFGWRITE_USER_ABORT", "nwparser.payload", "Configuration copy aborted by the user.%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg198 = msg("CFGWRITE_USER_ABORT", part158);

var msg199 = msg("IF_DOWN_BIT_ERR_RT_THRES_EXCEEDED", dup95);

var part159 = match("MESSAGE#199:last", "nwparser.payload", "message repeated %{dclass_counter1->} time", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","last message repeated number of times."),
	setc("dclass_counter1_string","Number of times repeated"),
]));

var msg200 = msg("last", part159);

var part160 = match("MESSAGE#200:SERVICE_CRASHED", "nwparser.payload", "Service %{service->} (PID %{parent_pid}) hasn't caught signal %{fld43->} (%{result}).", processor_chain([
	dup32,
	dup2,
	dup3,
	dup4,
]));

var msg201 = msg("SERVICE_CRASHED", part160);

var part161 = match("MESSAGE#201:SERVICELOST", "nwparser.payload", "Service %{service->} lost on WCCP Client %{saddr}", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
	setc("event_description","Service lost on WCCP Client"),
]));

var msg202 = msg("SERVICELOST", part161);

var part162 = match("MESSAGE#202:IF_BRINGUP_ALLOWED_FCOT_CHECKSUM_ERR", "nwparser.payload", "Interface %{interface->} is allowed to come up even with SFP checksum error", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg203 = msg("IF_BRINGUP_ALLOWED_FCOT_CHECKSUM_ERR", part162);

var part163 = match("MESSAGE#203:PS_FAIL/0", "nwparser.payload", "Power supply %{fld43->} failed or shut%{p0}");

var part164 = match("MESSAGE#203:PS_FAIL/1_0", "nwparser.p0", " down %{p0}");

var part165 = match("MESSAGE#203:PS_FAIL/1_1", "nwparser.p0", "down %{p0}");

var select30 = linear_select([
	part164,
	part165,
]);

var part166 = match("MESSAGE#203:PS_FAIL/2", "nwparser.p0", "(Serial number %{serial_number})");

var all16 = all_match({
	processors: [
		part163,
		select30,
		part166,
	],
	on_success: processor_chain([
		dup23,
		dup2,
		dup3,
		dup4,
	]),
});

var msg204 = msg("PS_FAIL", all16);

var msg205 = msg("INFORMATION", dup87);

var msg206 = msg("EVENT", dup87);

var part167 = match("MESSAGE#206:NATIVE_VLAN_MISMATCH", "nwparser.payload", "Native VLAN mismatch discovered on %{interface}, with %{fld23}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg207 = msg("NATIVE_VLAN_MISMATCH", part167);

var part168 = match("MESSAGE#207:NEIGHBOR_ADDED", "nwparser.payload", "Device %{fld22->} discovered of type %{fld23->} with port %{fld24->} on incoming port %{interface->} with ip addr %{fld25->} and mgmt ip %{hostip}", processor_chain([
	dup29,
	dup2,
	dup3,
	dup4,
]));

var msg208 = msg("NEIGHBOR_ADDED", part168);

var part169 = match("MESSAGE#208:NEIGHBOR_REMOVED", "nwparser.payload", "CDP Neighbor %{fld22->} on port %{interface->} has been removed", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg209 = msg("NEIGHBOR_REMOVED", part169);

var part170 = match("MESSAGE#209:IF_BANDWIDTH_CHANGE", "nwparser.payload", "Interface %{interface},%{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var msg210 = msg("IF_BANDWIDTH_CHANGE", part170);

var part171 = match("MESSAGE#210:IF_DOWN_PARENT_ADMIN_DOWN", "nwparser.payload", "Interface %{interface->} is down (Parent interface down)", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg211 = msg("IF_DOWN_PARENT_ADMIN_DOWN", part171);

var part172 = match("MESSAGE#211:PORT_INDIVIDUAL_DOWN", "nwparser.payload", "individual port %{interface->} is down", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg212 = msg("PORT_INDIVIDUAL_DOWN", part172);

var part173 = match("MESSAGE#212:PORT_SUSPENDED", "nwparser.payload", "%{fld22}: %{interface->} is suspended", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg213 = msg("PORT_SUSPENDED", part173);

var part174 = match("MESSAGE#213:FEX_PORT_STATUS_NOTI", "nwparser.payload", "Uplink-ID %{fld22->} of Fex %{fld23->} that is connected with %{interface->} changed its status from %{change_old->} to %{change_new}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("change_attribute","status"),
]));

var msg214 = msg("FEX_PORT_STATUS_NOTI", part174);

var msg215 = msg("NOHMS_DIAG_ERR_PS_FAIL", dup102);

var msg216 = msg("NOHMS_DIAG_ERR_PS_RECOVERED", dup87);

var msg217 = msg("ADJCHANGE", dup87);

var part175 = match("MESSAGE#217:PORT_ADDED", "nwparser.payload", "Interface %{interface}, added to VLAN%{vlan->} with role %{fld22}, state %{disposition}, %{info}", processor_chain([
	dup29,
	dup2,
	dup3,
	dup4,
]));

var msg218 = msg("PORT_ADDED", part175);

var part176 = match("MESSAGE#218:PORT_DELETED", "nwparser.payload", "Interface %{interface}, removed from VLAN%{vlan}", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var msg219 = msg("PORT_DELETED", part176);

var part177 = match("MESSAGE#219:PORT_ROLE", "nwparser.payload", "Port %{interface->} instance VLAN%{vlan->} role changed to %{fld22}", processor_chain([
	dup62,
	dup2,
	dup3,
	dup4,
]));

var msg220 = msg("PORT_ROLE", part177);

var part178 = match("MESSAGE#220:PORT_STATE", "nwparser.payload", "Port %{interface->} instance VLAN%{vlan->} moving from %{change_old->} to %{change_new}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("change_attribute","Port state"),
]));

var msg221 = msg("PORT_STATE", part178);

var part179 = match("MESSAGE#221:TACACS_ACCOUNTING_MESSAGE", "nwparser.payload", "update: %{saddr}@%{terminal}: %{username}: %{event_description}; feature %{protocol->} (%{result}) %{info}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var msg222 = msg("TACACS_ACCOUNTING_MESSAGE", part179);

var part180 = match("MESSAGE#222:TACACS_ACCOUNTING_MESSAGE:01", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}: enabled telnet", processor_chain([
	dup22,
	dup37,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	dup39,
	dup40,
]));

var msg223 = msg("TACACS_ACCOUNTING_MESSAGE:01", part180);

var part181 = match("MESSAGE#368:TACACS_ACCOUNTING_MESSAGE:04", "nwparser.payload", "%{action}: %{saddr}@%{terminal}: %{username}: configure terminal ; ntp source-interface %{sinterface->} (%{result})%{info}", processor_chain([
	dup63,
	dup2,
	dup4,
]));

var msg224 = msg("TACACS_ACCOUNTING_MESSAGE:04", part181);

var part182 = match("MESSAGE#369:TACACS_ACCOUNTING_MESSAGE:05/0", "nwparser.payload", "%{action}: %{saddr}@%{terminal}: %{username}: show %{p0}");

var part183 = match("MESSAGE#369:TACACS_ACCOUNTING_MESSAGE:05/1_0", "nwparser.p0", "ntp statistics peer ipaddr %{hostip->} (%{p0}");

var part184 = match("MESSAGE#369:TACACS_ACCOUNTING_MESSAGE:05/1_1", "nwparser.p0", "logging last %{fld3->} (%{p0}");

var select31 = linear_select([
	part183,
	part184,
]);

var part185 = match("MESSAGE#369:TACACS_ACCOUNTING_MESSAGE:05/2", "nwparser.p0", "%{result})%{info}");

var all17 = all_match({
	processors: [
		part182,
		select31,
		part185,
	],
	on_success: processor_chain([
		dup63,
		dup2,
		dup4,
	]),
});

var msg225 = msg("TACACS_ACCOUNTING_MESSAGE:05", all17);

var part186 = match("MESSAGE#370:TACACS_ACCOUNTING_MESSAGE:06", "nwparser.payload", "%{action}: %{saddr}@%{terminal}: %{username}: clock set %{event_time_string->} (%{result})%{info}", processor_chain([
	dup63,
	dup2,
	dup4,
]));

var msg226 = msg("TACACS_ACCOUNTING_MESSAGE:06", part186);

var part187 = match("MESSAGE#371:TACACS_ACCOUNTING_MESSAGE:08", "nwparser.payload", "%{action}: %{saddr}@%{terminal}: %{username}: Performing configuration copy. %{info}", processor_chain([
	dup63,
	dup2,
	dup4,
	setc("event_description","Performing configuration copy"),
]));

var msg227 = msg("TACACS_ACCOUNTING_MESSAGE:08", part187);

var part188 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/2", "nwparser.p0", "%{username}: shell terminated because of session timeout %{p0}");

var all18 = all_match({
	processors: [
		dup64,
		dup103,
		part188,
		dup104,
	],
	on_success: processor_chain([
		dup63,
		dup2,
		dup4,
		setc("event_description","shell terminated because of session timeout"),
	]),
});

var msg228 = msg("TACACS_ACCOUNTING_MESSAGE:09", all18);

var part189 = match("MESSAGE#373:TACACS_ACCOUNTING_MESSAGE:07/2", "nwparser.p0", "%{username}: %{event_description->} %{p0}");

var all19 = all_match({
	processors: [
		dup64,
		dup103,
		part189,
		dup104,
	],
	on_success: processor_chain([
		dup63,
		dup2,
		dup4,
	]),
});

var msg229 = msg("TACACS_ACCOUNTING_MESSAGE:07", all19);

var select32 = linear_select([
	msg222,
	msg223,
	msg224,
	msg225,
	msg226,
	msg227,
	msg228,
	msg229,
]);

var msg230 = msg("TACACS_ERROR_MESSAGE", dup102);

var msg231 = msg("IF_SFP_WARNING", dup105);

var msg232 = msg("IF_DOWN_TCP_MAX_RETRANSMIT", dup106);

var msg233 = msg("FCIP_PEER_CAVIUM", dup87);

var msg234 = msg("IF_DOWN_PEER_CLOSE", dup106);

var msg235 = msg("IF_DOWN_PEER_RESET", dup106);

var part190 = match("MESSAGE#229:INTF_CONSISTENCY_FAILED", "nwparser.payload", "In domain %{domain}, VPC %{obj_name->} configuration is not consistent (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","configuration is not consistent in domain"),
]));

var msg236 = msg("INTF_CONSISTENCY_FAILED", part190);

var part191 = match("MESSAGE#230:INTF_CONSISTENCY_SUCCESS", "nwparser.payload", "In domain %{domain}, vPC %{obj_name->} configuration is consistent", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("event_description","configuration is consistent in domain"),
]));

var msg237 = msg("INTF_CONSISTENCY_SUCCESS", part191);

var msg238 = msg("INTF_COUNTERS_CLEARED", dup105);

var msg239 = msg("IF_HARDWARE", dup105);

var part192 = match_copy("MESSAGE#233:HEARTBEAT_FAILURE", "nwparser.payload", "event_description", processor_chain([
	setc("eventcategory","1604010000"),
	dup2,
	dup3,
	dup4,
]));

var msg240 = msg("HEARTBEAT_FAILURE", part192);

var msg241 = msg("SYSMGR_AUTOCOLLECT_TECH_SUPPORT_LOG", dup87);

var msg242 = msg("PFM_FAN_FLTR_STATUS", dup87);

var msg243 = msg("MOUNT", dup87);

var msg244 = msg("LOG_CMP_UP", dup87);

var part193 = match("MESSAGE#238:IF_XCVR_WARNING/2", "nwparser.p0", "Temperature Warning cleared%{}");

var all20 = all_match({
	processors: [
		dup69,
		dup107,
		part193,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg245 = msg("IF_XCVR_WARNING", all20);

var msg246 = msg("IF_XCVR_WARNING:01", dup108);

var select33 = linear_select([
	msg245,
	msg246,
]);

var part194 = match("MESSAGE#240:IF_XCVR_ALARM/2", "nwparser.p0", "Temperature Alarm cleared%{}");

var all21 = all_match({
	processors: [
		dup69,
		dup107,
		part194,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg247 = msg("IF_XCVR_ALARM", all21);

var msg248 = msg("IF_XCVR_ALARM:01", dup108);

var select34 = linear_select([
	msg247,
	msg248,
]);

var msg249 = msg("MEMORY_ALERT", dup87);

var msg250 = msg("MEMORY_ALERT_RECOVERED", dup87);

var part195 = match("MESSAGE#244:IF_SFP_ALARM/2", "nwparser.p0", "Rx Power Alarm cleared%{}");

var all22 = all_match({
	processors: [
		dup69,
		dup107,
		part195,
	],
	on_success: processor_chain([
		dup15,
		dup2,
		dup3,
		dup4,
	]),
});

var msg251 = msg("IF_SFP_ALARM", all22);

var msg252 = msg("IF_SFP_ALARM:01", dup108);

var select35 = linear_select([
	msg251,
	msg252,
]);

var part196 = match_copy("MESSAGE#246:NBRCHANGE_DUAL", "nwparser.payload", "event_description", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
]));

var msg253 = msg("NBRCHANGE_DUAL", part196);

var part197 = match("MESSAGE#247:SOHMS_DIAG_ERROR/0", "nwparser.payload", "%{} %{device->} %{p0}");

var part198 = match("MESSAGE#247:SOHMS_DIAG_ERROR/1_0", "nwparser.p0", "%{action}: System %{p0}");

var part199 = match("MESSAGE#247:SOHMS_DIAG_ERROR/1_1", "nwparser.p0", "System %{p0}");

var select36 = linear_select([
	part198,
	part199,
]);

var part200 = match("MESSAGE#247:SOHMS_DIAG_ERROR/2", "nwparser.p0", "minor alarm on fans in fan tray %{dclass_counter1}");

var all23 = all_match({
	processors: [
		part197,
		select36,
		part200,
	],
	on_success: processor_chain([
		dup61,
		dup38,
		dup72,
		dup2,
		dup3,
		dup4,
		setc("event_description","System minor alarm on fans in fan tray"),
	]),
});

var msg254 = msg("SOHMS_DIAG_ERROR", all23);

var part201 = match("MESSAGE#248:SOHMS_DIAG_ERROR:01", "nwparser.payload", "%{device->} System minor alarm on power supply %{fld42}: %{result}", processor_chain([
	dup61,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","FEX-System minor alarm on power supply."),
]));

var msg255 = msg("SOHMS_DIAG_ERROR:01", part201);

var part202 = match("MESSAGE#249:SOHMS_DIAG_ERROR:02", "nwparser.payload", "%{device}: %{event_description}", processor_chain([
	dup61,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg256 = msg("SOHMS_DIAG_ERROR:02", part202);

var select37 = linear_select([
	msg254,
	msg255,
	msg256,
]);

var part203 = match("MESSAGE#250:M2FIB_MAC_TBL_PRGMING", "nwparser.payload", "Failed to program the mac table on %{device->} for group: %{fld1}, (%{fld2->} (%{fld3}), %{fld4}, %{hostip}). Error: %{result}. %{info}", processor_chain([
	dup73,
	dup34,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","Failed to program the mac table"),
]));

var msg257 = msg("M2FIB_MAC_TBL_PRGMING", part203);

var part204 = match("MESSAGE#251:DELETE_STALE_USER_ACCOUNT", "nwparser.payload", "deleting expired user account:%{username}", processor_chain([
	dup19,
	dup11,
	dup20,
	setc("ec_theme","UserGroup"),
	dup2,
	dup3,
	dup4,
	setc("event_description","deleting expired user account"),
]));

var msg258 = msg("DELETE_STALE_USER_ACCOUNT", part204);

var part205 = match("MESSAGE#252:IF_ADMIN_UP", "nwparser.payload", "Interface %{interface->} is admin up", processor_chain([
	dup30,
	dup34,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface is admin up."),
]));

var msg259 = msg("IF_ADMIN_UP", part205);

var part206 = match("MESSAGE#253:VPC_CFGD", "nwparser.payload", "vPC %{obj_name->} is configured", processor_chain([
	dup30,
	dup34,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","vPC is configured"),
	dup74,
]));

var msg260 = msg("VPC_CFGD", part206);

var part207 = match("MESSAGE#254:MODULE_ONLINE", "nwparser.payload", "System Manager has received notification of %{info}", processor_chain([
	dup30,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","System Manager has received notification of local module becoming online."),
]));

var msg261 = msg("MODULE_ONLINE", part207);

var part208 = match("MESSAGE#255:BIOS_DAEMON_LC_PRI_BOOT", "nwparser.payload", "System booted from Primary BIOS Flash%{}", processor_chain([
	dup30,
	dup75,
	dup76,
	dup2,
	dup3,
	dup4,
	setc("event_description","System booted from Primary BIOS Flash"),
]));

var msg262 = msg("BIOS_DAEMON_LC_PRI_BOOT", part208);

var part209 = match("MESSAGE#256:PEER_VPC_DOWN", "nwparser.payload", "Peer %{obj_name->} is down ()", processor_chain([
	dup77,
	dup34,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","Peer vPC is down"),
	dup74,
]));

var msg263 = msg("PEER_VPC_DOWN", part209);

var part210 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/0", "nwparser.payload", "In domain %{domain}, %{p0}");

var part211 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/1_0", "nwparser.p0", "VPC%{p0}");

var part212 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/1_1", "nwparser.p0", "vPC%{p0}");

var select38 = linear_select([
	part211,
	part212,
]);

var part213 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/2", "nwparser.p0", "%{}peer%{p0}");

var part214 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/3_0", "nwparser.p0", "-keepalive%{p0}");

var part215 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/3_1", "nwparser.p0", " keep-alive%{p0}");

var select39 = linear_select([
	part214,
	part215,
]);

var part216 = match("MESSAGE#257:PEER_KEEP_ALIVE_RECV_INT_LATEST/4", "nwparser.p0", "%{}received on interface %{interface}");

var all24 = all_match({
	processors: [
		part210,
		select38,
		part213,
		select39,
		part216,
	],
	on_success: processor_chain([
		dup36,
		dup2,
		dup3,
		dup4,
		setc("event_description","In domain, VPC peer-keepalive received on interface"),
	]),
});

var msg264 = msg("PEER_KEEP_ALIVE_RECV_INT_LATEST", all24);

var part217 = match("MESSAGE#258:PEER_KEEP_ALIVE_RECV_SUCCESS", "nwparser.payload", "In domain %{domain}, vPC peer keep-alive receive is successful", processor_chain([
	dup36,
	dup34,
	dup78,
	dup35,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","In domain, vPC peer keep-alive receive is successful"),
]));

var msg265 = msg("PEER_KEEP_ALIVE_RECV_SUCCESS", part217);

var part218 = match("MESSAGE#259:PEER_KEEP_ALIVE_RECV_FAIL", "nwparser.payload", "In domain %{domain}, VPC peer keep-alive receive has failed", processor_chain([
	dup77,
	dup34,
	dup78,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
	setc("event_description","In domain, VPC peer keep-alive receive has failed"),
]));

var msg266 = msg("PEER_KEEP_ALIVE_RECV_FAIL", part218);

var part219 = match("MESSAGE#260:PEER_KEEP_ALIVE_SEND_INT_LATEST", "nwparser.payload", "In domain %{domain}, VPC peer-keepalive sent on interface %{interface}", processor_chain([
	dup36,
	dup34,
	dup79,
	dup35,
	dup2,
	dup3,
	dup4,
	setc("event_description","In domain, VPC peer-keepalive sent on interface"),
]));

var msg267 = msg("PEER_KEEP_ALIVE_SEND_INT_LATEST", part219);

var part220 = match("MESSAGE#261:PEER_KEEP_ALIVE_SEND_SUCCESS", "nwparser.payload", "In domain %{domain}, vPC peer keep-alive send is successful", processor_chain([
	dup36,
	dup34,
	dup79,
	dup35,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","In domain, vPC peer keep-alive send is successful"),
]));

var msg268 = msg("PEER_KEEP_ALIVE_SEND_SUCCESS", part220);

var part221 = match("MESSAGE#262:PEER_KEEP_ALIVE_STATUS", "nwparser.payload", "In domain %{domain}, peer keep-alive status changed to %{change_new}", processor_chain([
	dup30,
	dup34,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Peer keep-alive status changed."),
	setc("change_attribute","peer keep-alive status"),
]));

var msg269 = msg("PEER_KEEP_ALIVE_STATUS", part221);

var part222 = match("MESSAGE#263:EJECTOR_STAT_CHANGED", "nwparser.payload", "Ejectors' status in slot %{fld47->} has changed, %{info}", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Ejectors' status in slot has changed."),
]));

var msg270 = msg("EJECTOR_STAT_CHANGED", part222);

var part223 = match("MESSAGE#264:XBAR_DETECT", "nwparser.payload", "Xbar %{fld41->} detected (Serial number %{fld42})", processor_chain([
	dup29,
	setc("ec_activity","Detect"),
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Xbar detected"),
]));

var msg271 = msg("XBAR_DETECT", part223);

var part224 = match("MESSAGE#265:XBAR_PWRUP", "nwparser.payload", "Xbar %{fld41->} powered up (Serial number %{fld42})", processor_chain([
	dup15,
	dup75,
	dup76,
	dup2,
	dup3,
	dup4,
	setc("event_description","Xbar powered up"),
]));

var msg272 = msg("XBAR_PWRUP", part224);

var part225 = match("MESSAGE#266:XBAR_PWRDN", "nwparser.payload", "Xbar %{fld41->} powered down (Serial number %{fld42})", processor_chain([
	dup15,
	dup75,
	setc("ec_activity","Stop"),
	dup2,
	dup3,
	dup4,
	setc("event_description","Xbar powered down"),
]));

var msg273 = msg("XBAR_PWRDN", part225);

var part226 = match("MESSAGE#267:XBAR_OK", "nwparser.payload", "Xbar %{fld41->} is online (serial: %{fld42})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Xbar is online"),
]));

var msg274 = msg("XBAR_OK", part226);

var part227 = match("MESSAGE#268:VPC_ISSU_START", "nwparser.payload", "Peer vPC switch ISSU start, locking configuration%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Peer vPC switch ISSU start, locking configuration"),
]));

var msg275 = msg("VPC_ISSU_START", part227);

var part228 = match("MESSAGE#269:VPC_ISSU_END", "nwparser.payload", "Peer vPC switch ISSU end, unlocking configuration%{}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
	setc("event_description","Peer vPC switch ISSU end, unlocking configuration"),
]));

var msg276 = msg("VPC_ISSU_END", part228);

var part229 = match("MESSAGE#270:PORT_RANGE_ROLE", "nwparser.payload", "new_role=%{obj_name->} interface=%{interface->} mst=%{fld42}", processor_chain([
	dup62,
	dup2,
	dup3,
	dup4,
	setc("obj_type","new_role"),
]));

var msg277 = msg("PORT_RANGE_ROLE", part229);

var part230 = match("MESSAGE#271:PORT_RANGE_STATE", "nwparser.payload", "new_state=%{obj_name->} interface=%{interface->} mst=%{fld42}", processor_chain([
	dup62,
	dup2,
	dup3,
	dup4,
	setc("obj_type","new_state"),
]));

var msg278 = msg("PORT_RANGE_STATE", part230);

var part231 = match("MESSAGE#272:PORT_RANGE_DELETED", "nwparser.payload", "Interface %{interface->} removed from mst=%{fld42}", processor_chain([
	dup24,
	dup34,
	dup20,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface removed from MST."),
]));

var msg279 = msg("PORT_RANGE_DELETED", part231);

var part232 = match("MESSAGE#273:PORT_RANGE_ADDED", "nwparser.payload", "Interface %{interface->} added to mst=%{fld42->} with %{info}", processor_chain([
	dup29,
	dup34,
	dup80,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface added to MST."),
]));

var msg280 = msg("PORT_RANGE_ADDED", part232);

var part233 = match("MESSAGE#274:MST_PORT_BOUNDARY", "nwparser.payload", "Port %{portname->} removed as MST Boundary port", processor_chain([
	dup24,
	dup34,
	dup20,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Port removed as MST Boundary port"),
]));

var msg281 = msg("MST_PORT_BOUNDARY", part233);

var part234 = match("MESSAGE#275:PIXM_SYSLOG_MESSAGE_TYPE_CRIT", "nwparser.payload", "Non-transactional PIXM Error. Error Type: %{result}.%{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","Non-transactional PIXM Error"),
]));

var msg282 = msg("PIXM_SYSLOG_MESSAGE_TYPE_CRIT", part234);

var part235 = match("MESSAGE#276:IM_INTF_STATE", "nwparser.payload", "%{interface->} is %{obj_name->} in vdc %{fld43}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("obj_type"," Interface state"),
]));

var msg283 = msg("IM_INTF_STATE", part235);

var part236 = match("MESSAGE#277:VDC_STATE_CHANGE", "nwparser.payload", "vdc %{fld43->} state changed to %{obj_name}", processor_chain([
	dup62,
	dup34,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","VDC state changed."),
	setc("obj_type"," VDC state"),
]));

var msg284 = msg("VDC_STATE_CHANGE", part236);

var part237 = match("MESSAGE#278:SWITCHOVER_OVER", "nwparser.payload", "Switchover completed.%{}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	dup81,
]));

var msg285 = msg("SWITCHOVER_OVER", part237);

var part238 = match("MESSAGE#279:VDC_MODULETYPE", "nwparser.payload", "%{process}: Module type changed to %{obj_name}", processor_chain([
	dup62,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	dup81,
	setc("obj_type"," New Module type"),
]));

var msg286 = msg("VDC_MODULETYPE", part238);

var part239 = match("MESSAGE#280:HASEQNO_SYNC_FAILED", "nwparser.payload", "Unable to sync HA sequence number %{fld44->} for service \"%{service}\" (PID %{process_id}): %{result}.", processor_chain([
	dup77,
	dup34,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
	setc("event_description","Unable to sync HA sequence number for service"),
]));

var msg287 = msg("HASEQNO_SYNC_FAILED", part239);

var part240 = match("MESSAGE#281:MSG_SEND_FAILURE_STANDBY_RESET", "nwparser.payload", "Failure in sending message to standby causing standby to reset.%{}", processor_chain([
	dup1,
	dup34,
	dup79,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
	setc("event_description","Failure in sending message to standby causing standby to reset."),
]));

var msg288 = msg("MSG_SEND_FAILURE_STANDBY_RESET", part240);

var part241 = match("MESSAGE#282:MODULE_LOCK_FAILED", "nwparser.payload", "Failed to lock the local module to avoid reset (error-id %{resultcode}).", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","Failed to lock the local module to avoid reset"),
]));

var msg289 = msg("MODULE_LOCK_FAILED", part241);

var part242 = match("MESSAGE#283:L2FMC_NL_MTS_SEND_FAILURE", "nwparser.payload", "Failed to send Mac New Learns/Mac moves due to mts send failure errno %{resultcode}", processor_chain([
	dup1,
	dup34,
	dup79,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
	setc("event_description","Failed to send Mac New Learns/Mac moves due to mts send failure."),
]));

var msg290 = msg("L2FMC_NL_MTS_SEND_FAILURE", part242);

var part243 = match("MESSAGE#284:SERVER_ADDED", "nwparser.payload", "Server with Chassis ID %{id->} Port ID %{fld45->} management address %{fld46->} discovered on local port %{portname->} in vlan %{vlan->} %{info}", processor_chain([
	dup29,
	dup80,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Server discovered on local in vlan 0 with enabled capability Station"),
]));

var msg291 = msg("SERVER_ADDED", part243);

var part244 = match("MESSAGE#285:SERVER_REMOVED", "nwparser.payload", "Server with Chassis ID %{id->} Port ID %{fld45->} on local port %{portname->} has been removed", processor_chain([
	dup24,
	dup20,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Server on local port has been removed"),
]));

var msg292 = msg("SERVER_REMOVED", part244);

var part245 = match("MESSAGE#286:IF_DOWN_SUSPENDED_BY_SPEED", "nwparser.payload", "Interface %{interface->} is down %{info}", processor_chain([
	dup23,
	dup34,
	dup72,
	dup2,
	dup3,
	dup4,
	dup25,
]));

var msg293 = msg("IF_DOWN_SUSPENDED_BY_SPEED", part245);

var part246 = match("MESSAGE#287:PORT_INDIVIDUAL", "nwparser.payload", "port %{portname->} is operationally individual", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	setc("event_description","port is operationally individual"),
]));

var msg294 = msg("PORT_INDIVIDUAL", part246);

var part247 = match("MESSAGE#288:IF_DOWN_CHANNEL_ADMIN_DOWN", "nwparser.payload", "Interface %{interface->} is down %{info}", processor_chain([
	dup23,
	dup34,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	dup25,
]));

var msg295 = msg("IF_DOWN_CHANNEL_ADMIN_DOWN", part247);

var part248 = match("MESSAGE#289:IF_ERRDIS_RECOVERY", "nwparser.payload", "Interface %{interface->} is being recovered from error disabled state %{info}", processor_chain([
	dup22,
	dup2,
	dup3,
	dup4,
	setc("event_description","Interface is being recovered from error disabled state"),
]));

var msg296 = msg("IF_ERRDIS_RECOVERY", part248);

var part249 = match("MESSAGE#290:IF_NON_CISCO_TRANSCEIVER", "nwparser.payload", "Non-Cisco transceiver on interface %{interface->} is detected", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Non-Cisco transceiver on interface is detected"),
]));

var msg297 = msg("IF_NON_CISCO_TRANSCEIVER", part249);

var part250 = match("MESSAGE#291:ACTIVE_LOWER_MEM_THAN_STANDBY", "nwparser.payload", "Active supervisor in slot %{fld47->} is running with less memory than standby supervisor in slot %{fld48}.", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Active supervisor is running with less memory than standby supervisor."),
]));

var msg298 = msg("ACTIVE_LOWER_MEM_THAN_STANDBY", part250);

var part251 = match("MESSAGE#292:READCONF_STARTED", "nwparser.payload", "Configuration update started (PID %{process_id}).", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Configuration update started."),
]));

var msg299 = msg("READCONF_STARTED", part251);

var part252 = match("MESSAGE#293:SUP_POWERDOWN", "nwparser.payload", "Supervisor in slot %{fld47->} is running with less memory than active supervisor in slot %{fld48}", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Supervisor is running with less memory than active supervisor."),
]));

var msg300 = msg("SUP_POWERDOWN", part252);

var part253 = match("MESSAGE#294:LC_UPGRADE_START", "nwparser.payload", "Starting linecard upgrade%{}", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Starting linecard upgrade"),
]));

var msg301 = msg("LC_UPGRADE_START", part253);

var part254 = match("MESSAGE#295:LC_UPGRADE_REBOOT", "nwparser.payload", "Rebooting linecard as a part of upgrade%{}", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Rebooting linecard as a part of upgrade"),
]));

var msg302 = msg("LC_UPGRADE_REBOOT", part254);

var part255 = match("MESSAGE#296:RUNTIME_DB_RESTORE_STARTED", "nwparser.payload", "Runtime database controller started (PID %{process_id}).", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Runtime database controller started."),
]));

var msg303 = msg("RUNTIME_DB_RESTORE_STARTED", part255);

var part256 = match("MESSAGE#297:RUNTIME_DB_RESTORE_SUCCESS", "nwparser.payload", "Runtime database successfully restored.%{}", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Runtime database successfully restored."),
]));

var msg304 = msg("RUNTIME_DB_RESTORE_SUCCESS", part256);

var part257 = match("MESSAGE#298:LCM_MODULE_UPGRADE_START", "nwparser.payload", "Upgrade of module %{fld49->} started", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Upgrade of module started"),
]));

var msg305 = msg("LCM_MODULE_UPGRADE_START", part257);

var part258 = match("MESSAGE#299:LCM_MODULE_UPGRADE_END", "nwparser.payload", "Upgrade of module %{fld49->} ended", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Upgrade of module ended"),
]));

var msg306 = msg("LCM_MODULE_UPGRADE_END", part258);

var part259 = match("MESSAGE#300:FIPS_POST_INFO_MSG", "nwparser.payload", "Recieved insert for %{fld50}", processor_chain([
	dup63,
	dup34,
	dup78,
	dup35,
	dup2,
	dup3,
	dup4,
	setc("event_description","Recieved insert for lc mod"),
]));

var msg307 = msg("FIPS_POST_INFO_MSG", part259);

var part260 = match("MESSAGE#301:PEER_VPC_CFGD", "nwparser.payload", "peer vPC %{obj_name->} is configured", processor_chain([
	dup30,
	dup34,
	dup38,
	dup17,
	dup2,
	dup3,
	dup4,
	setc("event_description","peer vPC is configured"),
	dup74,
]));

var msg308 = msg("PEER_VPC_CFGD", part260);

var part261 = match("MESSAGE#302:SYN_COLL_DIS_EN", "nwparser.payload", "%{info}: Potential Interop issue on [%{interface}]: %{result}", processor_chain([
	dup73,
	dup34,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","Potential Interop issue on interface."),
]));

var msg309 = msg("SYN_COLL_DIS_EN", part261);

var part262 = match("MESSAGE#303:NOHMS_ENV_FEX_OFFLINE", "nwparser.payload", "%{device->} Off-line (Serial Number %{fld42})", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","FEX OFFLINE"),
]));

var msg310 = msg("NOHMS_ENV_FEX_OFFLINE", part262);

var part263 = match("MESSAGE#304:NOHMS_ENV_FEX_ONLINE", "nwparser.payload", "%{device->} On-line", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","FEX ONLINE"),
]));

var msg311 = msg("NOHMS_ENV_FEX_ONLINE", part263);

var part264 = match("MESSAGE#305:FEX_STATUS_online", "nwparser.payload", "%{device->} is online", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Fex is online"),
]));

var msg312 = msg("FEX_STATUS_online", part264);

var part265 = match("MESSAGE#306:FEX_STATUS_offline", "nwparser.payload", "%{device->} is offline", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","Fex is offline"),
]));

var msg313 = msg("FEX_STATUS_offline", part265);

var select40 = linear_select([
	msg312,
	msg313,
]);

var part266 = match("MESSAGE#307:PS_PWR_INPUT_MISSING", "nwparser.payload", "Power supply %{fld41->} present but all AC/DC inputs are not connected, power redundancy might be affected", processor_chain([
	dup73,
	dup38,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","Power supply present but all AC/DC inputs are not connected, power redundancy might be affected"),
]));

var msg314 = msg("PS_PWR_INPUT_MISSING", part266);

var part267 = match("MESSAGE#308:PS_RED_MODE_RESTORED", "nwparser.payload", "Power redundancy operational mode changed to %{change_new}", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Power redundancy operational mode changed."),
	setc("change_attribute","operational mode"),
]));

var msg315 = msg("PS_RED_MODE_RESTORED", part267);

var part268 = match("MESSAGE#309:MOD_PWRFAIL_EJECTORS_OPEN", "nwparser.payload", "All ejectors open, Module %{fld41->} will not be powered up (Serial number %{fld42})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","All ejectors open, Module will not be powered up."),
]));

var msg316 = msg("MOD_PWRFAIL_EJECTORS_OPEN", part268);

var part269 = match("MESSAGE#310:PINNING_CHANGED", "nwparser.payload", "%{device->} pinning information is changed", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
	setc("event_description","Fex pinning information is changed"),
]));

var msg317 = msg("PINNING_CHANGED", part269);

var part270 = match("MESSAGE#311:SATCTRL", "nwparser.payload", "%{device->} Module %{fld41}: Cold boot", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","FEX-100 Module -Cold boot"),
]));

var msg318 = msg("SATCTRL", part270);

var part271 = match("MESSAGE#312:DUP_REGISTER", "nwparser.payload", "%{fld51->} [%{fld52}] Client %{fld43->} register more than once with same pid%{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","Client register more than once with same pid"),
]));

var msg319 = msg("DUP_REGISTER", part271);

var part272 = match("MESSAGE#313:UNKNOWN_MTYPE", "nwparser.payload", "%{fld51->} [%{fld52}] Unknown mtype: %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	setc("event_description","Unknown mtype"),
]));

var msg320 = msg("UNKNOWN_MTYPE", part272);

var part273 = match("MESSAGE#314:SATCTRL_IMAGE", "nwparser.payload", "%{fld51->} %{event_description}", processor_chain([
	dup30,
	dup16,
	dup38,
	dup2,
	dup3,
	dup4,
]));

var msg321 = msg("SATCTRL_IMAGE", part273);

var part274 = match("MESSAGE#315:API_FAILED", "nwparser.payload", "%{fld51->} [%{fld52}] %{event_description}", processor_chain([
	dup1,
	setc("ec_subject","Process"),
	dup14,
	dup2,
	dup3,
	dup4,
]));

var msg322 = msg("API_FAILED", part274);

var part275 = match_copy("MESSAGE#316:SENSOR_MSG1", "nwparser.payload", "event_description", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
]));

var msg323 = msg("SENSOR_MSG1", part275);

var part276 = match("MESSAGE#317:API_INIT_SEM_CLEAR", "nwparser.payload", "%{fld51->} [%{fld52}] %{event_description}", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
]));

var msg324 = msg("API_INIT_SEM_CLEAR", part276);

var part277 = match("MESSAGE#318:VDC_ONLINE", "nwparser.payload", "vdc %{fld51->} has come online", processor_chain([
	dup30,
	dup2,
	dup3,
	dup4,
	setc("event_description","vdc has come online"),
]));

var msg325 = msg("VDC_ONLINE", part277);

var part278 = match("MESSAGE#319:LACP_SUSPEND_INDIVIDUAL", "nwparser.payload", "LACP port %{portname->} of port-channel %{interface->} not receiving any LACP BPDUs %{result}", processor_chain([
	dup77,
	dup34,
	dup78,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
	setc("event_description","LACP port of port-channel not receiving any LACP BPDUs."),
]));

var msg326 = msg("LACP_SUSPEND_INDIVIDUAL", part278);

var part279 = match("MESSAGE#320:dstats", "nwparser.payload", "%{process}: %{info}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
]));

var msg327 = msg("dstats", part279);

var part280 = match("MESSAGE#321:MSG_PORT_LOGGED_OUT", "nwparser.payload", "%{fld52->} [VSAN %{fld51}, Interface %{interface}: %{fld53->} Nx Port %{portname->} logged OUT.", processor_chain([
	dup77,
	dup34,
	setc("ec_activity","Logoff"),
	dup35,
	dup2,
	dup3,
	dup4,
]));

var msg328 = msg("MSG_PORT_LOGGED_OUT", part280);

var part281 = match("MESSAGE#322:MSG_PORT_LOGGED_IN", "nwparser.payload", "%{fld52->} [VSAN %{fld51}, Interface %{interface}: %{fld53->} Nx Port %{portname->} with FCID %{fld54->} logged IN.", processor_chain([
	dup77,
	dup34,
	dup13,
	dup35,
	dup2,
	dup3,
	dup4,
]));

var msg329 = msg("MSG_PORT_LOGGED_IN", part281);

var msg330 = msg("IF_DOWN_ELP_FAILURE_ISOLATION", dup96);

var part282 = match("MESSAGE#324:ZS_MERGE_FAILED", "nwparser.payload", "%{fld52->} Zone merge failure, isolating interface %{interface->} reason: %{result}:[%{resultcode}]", processor_chain([
	dup23,
	dup34,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
]));

var msg331 = msg("ZS_MERGE_FAILED", part282);

var msg332 = msg("IF_DOWN_ZONE_MERGE_FAILURE_ISOLATION", dup96);

var part283 = match("MESSAGE#326:MAC_MOVE_NOTIFICATION", "nwparser.payload", "Host %{hostname->} in vlan %{vlan->} is flapping between port %{change_old->} and port %{change_new}", processor_chain([
	dup23,
	dup34,
	dup35,
	dup2,
	dup3,
	dup4,
	setc("change_attribute","Port"),
]));

var msg333 = msg("MAC_MOVE_NOTIFICATION", part283);

var part284 = match("MESSAGE#327:zone", "nwparser.payload", "num_tlv greater than 1, %{result}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
]));

var msg334 = msg("zone", part284);

var part285 = match("MESSAGE#328:ERROR", "nwparser.payload", "%{event_description}: %{info}", processor_chain([
	dup1,
	dup34,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg335 = msg("ERROR", part285);

var part286 = match("MESSAGE#329:INVAL_IP", "nwparser.payload", "%{agent->} [%{process_id}] Received packet with invalid destination IP address (%{daddr}) from %{smacaddr->} on %{interface}", processor_chain([
	dup77,
	dup34,
	dup78,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg336 = msg("INVAL_IP", part286);

var part287 = match("MESSAGE#330:SYSLOG_SL_MSG_WARNING", "nwparser.payload", "%{process}: message repeated %{dclass_counter1->} times in last %{duration}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var msg337 = msg("SYSLOG_SL_MSG_WARNING", part287);

var part288 = match("MESSAGE#331:DUPLEX_MISMATCH", "nwparser.payload", "Duplex mismatch discovered on %{interface}, with %{fld55}", processor_chain([
	dup77,
	dup34,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg338 = msg("DUPLEX_MISMATCH", part288);

var part289 = match("MESSAGE#332:NOHMS_DIAG_ERROR", "nwparser.payload", "Module %{fld20}: Runtime diag detected major event: Fabric port failure %{interface}", processor_chain([
	dup77,
	dup34,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg339 = msg("NOHMS_DIAG_ERROR", part289);

var part290 = match("MESSAGE#333:STM_LEARNING_RE_ENABLE", "nwparser.payload", "Re enabling dynamic learning on all interfaces%{}", processor_chain([
	dup15,
	dup34,
	dup35,
	dup2,
	dup3,
	dup4,
]));

var msg340 = msg("STM_LEARNING_RE_ENABLE", part290);

var part291 = match("MESSAGE#334:UDLD_PORT_DISABLED", "nwparser.payload", "UDLD disabled interface %{interface}, %{result}", processor_chain([
	dup77,
	dup34,
	dup35,
	dup72,
	dup2,
	dup3,
	dup4,
]));

var msg341 = msg("UDLD_PORT_DISABLED", part291);

var part292 = match("MESSAGE#335:ntpd", "nwparser.payload", "ntp:no servers reachable%{}", processor_chain([
	dup15,
	dup2,
	dup4,
]));

var msg342 = msg("ntpd", part292);

var part293 = match("MESSAGE#336:ntpd:01", "nwparser.payload", "ntp:event EVNT_UNREACH %{saddr}", processor_chain([
	dup15,
	dup2,
	dup4,
]));

var msg343 = msg("ntpd:01", part293);

var part294 = match("MESSAGE#337:ntpd:02", "nwparser.payload", "ntp:event EVNT_REACH %{saddr}", processor_chain([
	dup15,
	dup2,
	dup4,
]));

var msg344 = msg("ntpd:02", part294);

var part295 = match("MESSAGE#338:ntpd:03", "nwparser.payload", "ntp:synchronized to %{saddr}, stratum %{fld9}", processor_chain([
	dup15,
	dup2,
	dup4,
]));

var msg345 = msg("ntpd:03", part295);

var part296 = match("MESSAGE#339:ntpd:04", "nwparser.payload", "ntp:%{event_description}", processor_chain([
	dup15,
	dup2,
	dup4,
]));

var msg346 = msg("ntpd:04", part296);

var select41 = linear_select([
	msg342,
	msg343,
	msg344,
	msg345,
	msg346,
]);

var part297 = match_copy("MESSAGE#340:PFM_ALERT", "nwparser.payload", "event_description", processor_chain([
	dup9,
	dup2,
	dup3,
	dup4,
]));

var msg347 = msg("PFM_ALERT", part297);

var part298 = match("MESSAGE#341:SERVICEFOUND", "nwparser.payload", "Service %{service->} acquired on WCCP Client %{saddr}", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
	setc("event_description","Service acquired on WCCP Client"),
]));

var msg348 = msg("SERVICEFOUND", part298);

var part299 = match("MESSAGE#342:ROUTERFOUND", "nwparser.payload", "Service %{service->} acquired on WCCP Router %{saddr}", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
	setc("event_description","Service acquired on WCCP Router"),
]));

var msg349 = msg("ROUTERFOUND", part299);

var part300 = match("MESSAGE#343:%AUTHPRIV-3-SYSTEM_MSG", "nwparser.payload", "pam_aaa:Authentication failed from %{shost->} - %{agent}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	setc("event_description","Authentication failed"),
]));

var msg350 = msg("%AUTHPRIV-3-SYSTEM_MSG", part300);

var part301 = match("MESSAGE#344:%AUTHPRIV-5-SYSTEM_MSG", "nwparser.payload", "New user added with username %{username->} - %{agent}", processor_chain([
	dup18,
	dup2,
	dup12,
	dup3,
	dup4,
	setc("event_description","New user added"),
]));

var msg351 = msg("%AUTHPRIV-5-SYSTEM_MSG", part301);

var part302 = match("MESSAGE#345:%AUTHPRIV-6-SYSTEM_MSG:01", "nwparser.payload", "%{action}: %{service->} pid=%{process_id->} from=::ffff:%{saddr->} - %{agent}", processor_chain([
	dup10,
	dup2,
	dup12,
	dup3,
	dup4,
]));

var msg352 = msg("%AUTHPRIV-6-SYSTEM_MSG:01", part302);

var part303 = match("MESSAGE#346:%AUTHPRIV-6-SYSTEM_MSG", "nwparser.payload", "pam_unix(%{fld1}:session): session opened for user %{username->} by (uid=%{uid}) - %{agent}", processor_chain([
	dup10,
	dup2,
	dup12,
	dup3,
	dup4,
	setc("event_description","session opened for user"),
]));

var msg353 = msg("%AUTHPRIV-6-SYSTEM_MSG", part303);

var select42 = linear_select([
	msg352,
	msg353,
]);

var part304 = match("MESSAGE#347:%USER-3-SYSTEM_MSG", "nwparser.payload", "error: %{result}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
]));

var msg354 = msg("%USER-3-SYSTEM_MSG", part304);

var part305 = match("MESSAGE#348:%USER-6-SYSTEM_MSG", "nwparser.payload", "Invalid user %{username->} from %{saddr->} - %{agent}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup82,
]));

var msg355 = msg("%USER-6-SYSTEM_MSG", part305);

var part306 = match("MESSAGE#349:%USER-6-SYSTEM_MSG:01", "nwparser.payload", "input_userauth_request: invalid user %{username->} - %{agent}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	dup82,
]));

var msg356 = msg("%USER-6-SYSTEM_MSG:01", part306);

var part307 = match("MESSAGE#350:%USER-6-SYSTEM_MSG:02", "nwparser.payload", "Failed none for invalid user %{username->} from %{saddr->} port %{sport->} %{protocol->} - %{agent}", processor_chain([
	dup5,
	dup2,
	dup3,
	dup4,
	setc("event_description","Failed none for invalid user"),
]));

var msg357 = msg("%USER-6-SYSTEM_MSG:02", part307);

var part308 = match("MESSAGE#351:%USER-6-SYSTEM_MSG:03", "nwparser.payload", "Accepted password for %{username->} from %{saddr->} port %{sport->} %{protocol->} - %{agent}", processor_chain([
	dup83,
	dup2,
	dup3,
	dup4,
	setc("event_description","Accepted password for user"),
]));

var msg358 = msg("%USER-6-SYSTEM_MSG:03", part308);

var part309 = match("MESSAGE#352:%USER-6-SYSTEM_MSG:04", "nwparser.payload", "lastlog_openseek: Couldn't stat %{directory}: No such file or directory - %{agent}", processor_chain([
	dup83,
	dup2,
	dup3,
	dup4,
	setc("event_description","No such file or directory"),
]));

var msg359 = msg("%USER-6-SYSTEM_MSG:04", part309);

var part310 = match("MESSAGE#353:%USER-6-SYSTEM_MSG:05", "nwparser.payload", "Could not load host key: %{encryption_type->} - %{agent}", processor_chain([
	dup83,
	dup2,
	dup3,
	dup4,
	setc("event_description","Could not load host key"),
]));

var msg360 = msg("%USER-6-SYSTEM_MSG:05", part310);

var part311 = match("MESSAGE#354:%USER-6-SYSTEM_MSG:06", "nwparser.payload", "%{event_description->} - %{agent}", processor_chain([
	dup83,
	dup2,
	dup3,
	dup4,
]));

var msg361 = msg("%USER-6-SYSTEM_MSG:06", part311);

var select43 = linear_select([
	msg355,
	msg356,
	msg357,
	msg358,
	msg359,
	msg360,
	msg361,
]);

var part312 = match("MESSAGE#355:L2FM_MAC_FLAP_DISABLE_LEARN", "nwparser.payload", "Disabling learning in vlan %{vlan->} for %{duration}s due to too many mac moves", processor_chain([
	dup30,
	dup2,
	dup4,
	setc("ec_activity","Disable"),
]));

var msg362 = msg("L2FM_MAC_FLAP_DISABLE_LEARN", part312);

var part313 = match("MESSAGE#356:L2FM_MAC_FLAP_RE_ENABLE_LEARN", "nwparser.payload", "Re-enabling learning in vlan %{vlan}", processor_chain([
	dup30,
	dup2,
	dup4,
	dup37,
]));

var msg363 = msg("L2FM_MAC_FLAP_RE_ENABLE_LEARN", part313);

var part314 = match("MESSAGE#357:PS_ABSENT", "nwparser.payload", "Power supply %{fld1->} is %{disposition}, ps-redundancy might be affected", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var msg364 = msg("PS_ABSENT", part314);

var part315 = match("MESSAGE#358:PS_DETECT", "nwparser.payload", "Power supply %{fld1->} detected but %{disposition->} (Serial number %{serial_number})", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var msg365 = msg("PS_DETECT", part315);

var part316 = match("MESSAGE#359:SUBPROC_TERMINATED", "nwparser.payload", "\"System Manager (configuration controller)\" (PID %{process_id}) has finished with error code %{result->} (%{resultcode}).", processor_chain([
	dup1,
	dup2,
	dup4,
]));

var msg366 = msg("SUBPROC_TERMINATED", part316);

var part317 = match("MESSAGE#360:SUBPROC_SUCCESS_EXIT", "nwparser.payload", "\"%{service}\" (PID %{process_id}) has successfully exited with exit code %{result->} (%{resultcode}).", processor_chain([
	dup15,
	dup2,
	dup4,
	dup84,
	dup17,
]));

var msg367 = msg("SUBPROC_SUCCESS_EXIT", part317);

var part318 = match("MESSAGE#361:UPDOWN", "nwparser.payload", "Line Protocol on Interface vlan %{vlan}, changed state to %{disposition}", processor_chain([
	dup30,
	dup2,
	dup4,
]));

var msg368 = msg("UPDOWN", part318);

var part319 = match("MESSAGE#362:L2FM_MAC_MOVE2", "nwparser.payload", "Mac %{smacaddr->} in vlan %{vlan->} has moved between %{change_old->} to %{change_new}", processor_chain([
	dup30,
	dup2,
	dup4,
	setc("change_attribute","Interface"),
]));

var msg369 = msg("L2FM_MAC_MOVE2", part319);

var part320 = match("MESSAGE#363:PFM_PS_RED_MODE_CHG", "nwparser.payload", "Power redundancy configured mode changed to %{event_state}", processor_chain([
	dup30,
	dup2,
	dup4,
	dup38,
]));

var msg370 = msg("PFM_PS_RED_MODE_CHG", part320);

var part321 = match("MESSAGE#364:PS_RED_MODE_CHG", "nwparser.payload", "Power supply operational redundancy mode changed to %{event_state}", processor_chain([
	dup30,
	dup2,
	dup4,
	dup38,
]));

var msg371 = msg("PS_RED_MODE_CHG", part321);

var part322 = match("MESSAGE#365:INVAL_MAC", "nwparser.payload", "%{agent->} [%{process_id}] Received packet with invalid source MAC address (%{smacaddr}) from %{saddr->} on %{vlan}", processor_chain([
	dup63,
	dup2,
	dup4,
]));

var msg372 = msg("INVAL_MAC", part322);

var part323 = match("MESSAGE#366:SRVSTATE_CHANGED", "nwparser.payload", "State for service \"%{service}\" changed from %{change_old->} to %{change_new->} in vdc %{fld1}.", processor_chain([
	dup15,
	dup2,
	dup4,
	setc("change_attribute","Service status"),
]));

var msg373 = msg("SRVSTATE_CHANGED", part323);

var part324 = match_copy("MESSAGE#367:INFO", "nwparser.payload", "event_description", processor_chain([
	dup63,
	dup2,
	dup4,
]));

var msg374 = msg("INFO", part324);

var part325 = match("MESSAGE#374:SERVICE_STARTED", "nwparser.payload", "Service \"%{service}\" in vdc %{fld1->} started with PID(%{process_id}).", processor_chain([
	dup15,
	dup2,
	dup4,
	dup84,
	dup76,
	dup17,
]));

var msg375 = msg("SERVICE_STARTED", part325);

var part326 = match("MESSAGE#375:DUP_VADDR_SRCIP_PROBE", "nwparser.payload", "%{process->} [%{process_id}] Duplicate address Detected. Probe packet received from %{smacaddr->} on %{vlan->} with destination set to our local Virtual ip, %{saddr}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	dup85,
]));

var msg376 = msg("DUP_VADDR_SRCIP_PROBE", part326);

var part327 = match("MESSAGE#376:DUP_SRCIP_PROBE", "nwparser.payload", "%{process->} [%{process_id}] Duplicate address Detected. Probe packet received from %{smacaddr->} on %{vlan->} with destination set to our local ip, %{saddr}", processor_chain([
	dup8,
	dup2,
	dup3,
	dup4,
	dup85,
]));

var msg377 = msg("DUP_SRCIP_PROBE", part327);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"%AUTHPRIV-3-SYSTEM_MSG": msg350,
		"%AUTHPRIV-5-SYSTEM_MSG": msg351,
		"%AUTHPRIV-6-SYSTEM_MSG": select42,
		"%USER-3-SYSTEM_MSG": msg354,
		"%USER-6-SYSTEM_MSG": select43,
		"AAA_ACCOUNTING_MESSAGE": select28,
		"ACLLOG_FLOW_INTERVAL": msg187,
		"ACLLOG_MAXFLOW_REACHED": msg188,
		"ACLLOG_NEW_FLOW": msg189,
		"ACTIVE_LOWER_MEM_THAN_STANDBY": msg298,
		"ACTIVE_SUP_OK": msg74,
		"ADDON_IMG_DNLD_COMPLETE": msg60,
		"ADDON_IMG_DNLD_STARTED": msg61,
		"ADDON_IMG_DNLD_SUCCESSFUL": msg62,
		"ADJCHANGE": msg217,
		"API_FAILED": msg322,
		"API_INIT_SEM_CLEAR": msg324,
		"BIOS_DAEMON_LC_PRI_BOOT": msg262,
		"CFGWRITE_ABORTED": msg135,
		"CFGWRITE_ABORTED_LOCK": msg133,
		"CFGWRITE_DONE": msg136,
		"CFGWRITE_FAILED": msg134,
		"CFGWRITE_STARTED": msg137,
		"CFGWRITE_USER_ABORT": msg198,
		"CHASSIS_CLKMODOK": msg80,
		"CHASSIS_CLKSRC": msg81,
		"CONN_CONNECT": msg145,
		"CONN_DISCONNECT": msg146,
		"CREATED": msg51,
		"DELETE_STALE_USER_ACCOUNT": msg258,
		"DISPUTE_CLEARED": msg77,
		"DISPUTE_DETECTED": msg78,
		"DOMAIN_CFG_SYNC_DONE": msg79,
		"DUPLEX_MISMATCH": msg338,
		"DUP_REGISTER": msg319,
		"DUP_SRCIP_PROBE": msg377,
		"DUP_VADDR_SRCIP_PROBE": msg376,
		"DUP_VADDR_SRC_IP": msg190,
		"DVPG_CREATE": msg147,
		"DVPG_DELETE": msg148,
		"DVS_HOSTMEMBER_INFO": msg149,
		"DVS_NAME_CHANGE": msg150,
		"EJECTOR_STAT_CHANGED": msg270,
		"ERROR": msg335,
		"ERR_MSG": msg131,
		"EVENT": msg206,
		"FAN_DETECT": msg97,
		"FAN_OK": msg82,
		"FCIP_PEER_CAVIUM": msg233,
		"FEX_PORT_STATUS_NOTI": msg214,
		"FEX_STATUS": select40,
		"FIPS_POST_INFO_MSG": msg307,
		"FOP_CHANGED": msg52,
		"HASEQNO_SYNC_FAILED": msg287,
		"HEARTBEAT_FAILURE": msg240,
		"IF_ADMIN_UP": msg259,
		"IF_ATTACHED": msg138,
		"IF_BANDWIDTH_CHANGE": msg210,
		"IF_BRINGUP_ALLOWED_FCOT_CHECKSUM_ERR": msg203,
		"IF_DELETE_AUTO": msg139,
		"IF_DETACHED": msg140,
		"IF_DETACHED_MODULE_REMOVED": msg141,
		"IF_DOWN_ADMIN_DOWN": select11,
		"IF_DOWN_BIT_ERR_RT_THRES_EXCEEDED": msg199,
		"IF_DOWN_CFG_CHANGE": msg193,
		"IF_DOWN_CHANNEL_ADMIN_DOWN": msg295,
		"IF_DOWN_CHANNEL_MEMBERSHIP_UPDATE_IN_PROGRESS": msg38,
		"IF_DOWN_ELP_FAILURE_ISOLATION": msg330,
		"IF_DOWN_ERROR_DISABLED": msg35,
		"IF_DOWN_FCOT_NOT_PRESENT": select17,
		"IF_DOWN_INACTIVE": msg142,
		"IF_DOWN_INITIALIZING": select18,
		"IF_DOWN_INTERFACE_REMOVED": msg39,
		"IF_DOWN_LINK_FAILURE": select12,
		"IF_DOWN_MODULE_REMOVED": msg42,
		"IF_DOWN_NONE": select19,
		"IF_DOWN_NON_PARTICIPATING": msg143,
		"IF_DOWN_NOS_RCVD": select20,
		"IF_DOWN_OFFLINE": msg114,
		"IF_DOWN_OLS_RCVD": msg115,
		"IF_DOWN_PARENT_ADMIN_DOWN": msg211,
		"IF_DOWN_PEER_CLOSE": msg234,
		"IF_DOWN_PEER_RESET": msg235,
		"IF_DOWN_PORT_CHANNEL_MEMBERS_DOWN": msg43,
		"IF_DOWN_SOFTWARE_FAILURE": msg116,
		"IF_DOWN_SRC_PORT_NOT_BOUND": msg117,
		"IF_DOWN_SUSPENDED_BY_SPEED": msg293,
		"IF_DOWN_TCP_MAX_RETRANSMIT": msg232,
		"IF_DOWN_VEM_UNLICENSED": msg144,
		"IF_DOWN_ZONE_MERGE_FAILURE_ISOLATION": msg332,
		"IF_DUPLEX": msg44,
		"IF_ERRDIS_RECOVERY": msg296,
		"IF_ERROR_VLANS_REMOVED": msg191,
		"IF_ERROR_VLANS_SUSPENDED": msg192,
		"IF_HARDWARE": msg239,
		"IF_NON_CISCO_TRANSCEIVER": msg297,
		"IF_PORTPROFILE_ATTACHED": msg125,
		"IF_RX_FLOW_CONTROL": msg45,
		"IF_SEQ_ERROR": msg46,
		"IF_SFP_ALARM": select35,
		"IF_SFP_WARNING": msg231,
		"IF_TRUNK_DOWN": select21,
		"IF_TRUNK_UP": select22,
		"IF_TX_FLOW_CONTROL": msg47,
		"IF_UP": select13,
		"IF_XCVR_ALARM": select34,
		"IF_XCVR_WARNING": select33,
		"IMG_DNLD_COMPLETE": msg63,
		"IMG_DNLD_STARTED": msg64,
		"IM_INTF_STATE": msg283,
		"IM_SEQ_ERROR": msg59,
		"INFO": msg374,
		"INFORMATION": msg205,
		"INTF_CONSISTENCY_FAILED": msg236,
		"INTF_CONSISTENCY_SUCCESS": msg237,
		"INTF_COUNTERS_CLEARED": msg238,
		"INVAL_IP": msg336,
		"INVAL_MAC": msg372,
		"L2FMC_NL_MTS_SEND_FAILURE": msg290,
		"L2FM_MAC_FLAP_DISABLE_LEARN": msg362,
		"L2FM_MAC_FLAP_RE_ENABLE_LEARN": msg363,
		"L2FM_MAC_MOVE2": msg369,
		"LACP_SUSPEND_INDIVIDUAL": msg326,
		"LCM_MODULE_UPGRADE_END": msg306,
		"LCM_MODULE_UPGRADE_START": msg305,
		"LC_UPGRADE_REBOOT": msg302,
		"LC_UPGRADE_START": msg301,
		"LOG-7-SYSTEM_MSG": msg1,
		"LOG_CMP_AAA_FAILURE": msg67,
		"LOG_CMP_UP": msg244,
		"LOG_LIC_N1K_EXPIRY_WARNING": msg68,
		"M2FIB_MAC_TBL_PRGMING": msg257,
		"MAC_MOVE_NOTIFICATION": msg333,
		"MEMORY_ALERT": msg249,
		"MEMORY_ALERT_RECOVERED": msg250,
		"MESG": msg130,
		"MODULE_LOCK_FAILED": msg289,
		"MODULE_ONLINE": msg261,
		"MOD_BRINGUP_MULTI_LIMIT": msg96,
		"MOD_DETECT": msg83,
		"MOD_FAIL": msg69,
		"MOD_MAJORSWFAIL": msg70,
		"MOD_OK": msg75,
		"MOD_PWRDN": msg84,
		"MOD_PWRFAIL_EJECTORS_OPEN": msg316,
		"MOD_PWRUP": msg85,
		"MOD_REMOVE": msg86,
		"MOD_RESTART": msg76,
		"MOD_SRG_NOT_COMPATIBLE": msg71,
		"MOD_STATUS": msg98,
		"MOD_WARNING": select14,
		"MOUNT": msg243,
		"MSG_PORT_LOGGED_IN": msg329,
		"MSG_PORT_LOGGED_OUT": msg328,
		"MSG_SEND_FAILURE_STANDBY_RESET": msg288,
		"MSM_CRIT": msg66,
		"MST_PORT_BOUNDARY": msg281,
		"MTSERROR": msg34,
		"MTS_DROP": msg57,
		"NATIVE_VLAN_MISMATCH": msg207,
		"NBRCHANGE_DUAL": msg253,
		"NEIGHBOR_ADDED": msg208,
		"NEIGHBOR_REMOVED": msg209,
		"NEIGHBOR_UPDATE_AUTOCOPY": msg33,
		"NOHMS_DIAG_ERROR": msg339,
		"NOHMS_DIAG_ERR_PS_FAIL": msg215,
		"NOHMS_DIAG_ERR_PS_RECOVERED": msg216,
		"NOHMS_ENV_FEX_OFFLINE": msg310,
		"NOHMS_ENV_FEX_ONLINE": msg311,
		"PEER_KEEP_ALIVE_RECV_FAIL": msg266,
		"PEER_KEEP_ALIVE_RECV_INT_LATEST": msg264,
		"PEER_KEEP_ALIVE_RECV_SUCCESS": msg265,
		"PEER_KEEP_ALIVE_SEND_INT_LATEST": msg267,
		"PEER_KEEP_ALIVE_SEND_SUCCESS": msg268,
		"PEER_KEEP_ALIVE_STATUS": msg269,
		"PEER_VPC_CFGD": msg308,
		"PEER_VPC_CFGD_VLANS_CHANGED": msg99,
		"PEER_VPC_DELETED": msg100,
		"PEER_VPC_DOWN": msg263,
		"PFM_ALERT": msg347,
		"PFM_CLOCK_CHANGE": msg194,
		"PFM_FAN_FLTR_STATUS": msg242,
		"PFM_MODULE_POWER_ON": msg87,
		"PFM_PS_RED_MODE_CHG": msg370,
		"PFM_SYSTEM_RESET": msg88,
		"PFM_VEM_DETECTED": msg101,
		"PFM_VEM_REMOVE_NO_HB": msg89,
		"PFM_VEM_REMOVE_RESET": msg90,
		"PFM_VEM_REMOVE_STATE_CONFLICT": msg91,
		"PFM_VEM_REMOVE_TWO_ACT_VSM": msg92,
		"PFM_VEM_UNLICENSED": msg93,
		"PINNING_CHANGED": msg317,
		"PIXM_SYSLOG_MESSAGE_TYPE_CRIT": msg282,
		"POLICY_ACTIVATE_EVENT": msg27,
		"POLICY_COMMIT_EVENT": msg28,
		"POLICY_DEACTIVATE_EVENT": msg29,
		"POLICY_LOOKUP_EVENT": select10,
		"PORT_ADDED": msg218,
		"PORT_DELETED": msg219,
		"PORT_DOWN": msg53,
		"PORT_INDIVIDUAL": msg294,
		"PORT_INDIVIDUAL_DOWN": msg212,
		"PORT_PROFILE_CHANGE_VERIFY_REQ_FAILURE": msg124,
		"PORT_RANGE_ADDED": msg280,
		"PORT_RANGE_DELETED": msg279,
		"PORT_RANGE_ROLE": msg277,
		"PORT_RANGE_STATE": msg278,
		"PORT_ROLE": msg220,
		"PORT_SOFTWARE_FAILURE": msg65,
		"PORT_STATE": msg221,
		"PORT_SUSPENDED": msg213,
		"PORT_UP": msg54,
		"PS_ABSENT": msg364,
		"PS_CAPACITY_CHANGE": select16,
		"PS_DETECT": msg365,
		"PS_FAIL": msg204,
		"PS_FANOK": msg94,
		"PS_FOUND": msg102,
		"PS_OK": msg95,
		"PS_PWR_INPUT_MISSING": msg314,
		"PS_RED_MODE_CHG": msg371,
		"PS_RED_MODE_RESTORED": msg315,
		"PS_STATUS": msg103,
		"PVLAN_PPM_PORT_CONFIG_FAILED": msg129,
		"READCONF_STARTED": msg299,
		"RM_VICPP_RECREATE_ERROR": msg132,
		"ROUTERFOUND": msg349,
		"RUNTIME_DB_RESTORE_STARTED": msg303,
		"RUNTIME_DB_RESTORE_SUCCESS": msg304,
		"SATCTRL": msg318,
		"SATCTRL_IMAGE": msg321,
		"SENSOR_MSG1": msg323,
		"SERVER_ADDED": msg291,
		"SERVER_REMOVED": msg292,
		"SERVICEFOUND": msg348,
		"SERVICELOST": msg202,
		"SERVICE_CRASHED": msg201,
		"SERVICE_STARTED": msg375,
		"SOHMS_DIAG_ERROR": select37,
		"SPEED": msg50,
		"SRVSTATE_CHANGED": msg373,
		"STANDBY_SUP_OK": msg126,
		"STM_LEARNING_RE_ENABLE": msg340,
		"STM_LOOP_DETECT": msg127,
		"SUBGROUP_ID_PORT_ADDED": msg55,
		"SUBGROUP_ID_PORT_REMOVED": msg56,
		"SUBPROC_SUCCESS_EXIT": msg367,
		"SUBPROC_TERMINATED": msg366,
		"SUP_POWERDOWN": msg300,
		"SWITCHOVER_OVER": msg285,
		"SYNC_COMPLETE": msg128,
		"SYNC_FAILURE_STANDBY_RESET": msg195,
		"SYN_COLL_DIS_EN": msg309,
		"SYSLOG_LOG_WARNING": msg58,
		"SYSLOG_SL_MSG_WARNING": msg337,
		"SYSMGR_AUTOCOLLECT_TECH_SUPPORT_LOG": msg241,
		"SYSTEM_MSG": select9,
		"TACACS_ACCOUNTING_MESSAGE": select32,
		"TACACS_ERROR_MESSAGE": msg230,
		"UDLD_PORT_DISABLED": msg341,
		"UNKNOWN_MTYPE": msg320,
		"UPDOWN": msg368,
		"VDC_HOSTNAME_CHANGE": msg26,
		"VDC_MODULETYPE": msg286,
		"VDC_ONLINE": msg325,
		"VDC_STATE_CHANGE": msg284,
		"VMS_PPM_SYNC_COMPLETE": msg151,
		"VPC_CFGD": msg260,
		"VPC_DELETED": msg152,
		"VPC_ISSU_END": msg276,
		"VPC_ISSU_START": msg275,
		"VPC_UP": msg153,
		"VSHD_SYSLOG_CONFIG_I": select25,
		"XBAR_DETECT": msg271,
		"XBAR_OK": msg274,
		"XBAR_PWRDN": msg273,
		"XBAR_PWRUP": msg272,
		"ZS_MERGE_FAILED": msg331,
		"dstats": msg327,
		"last": msg200,
		"ntpd": select41,
		"snmpd": select29,
		"zone": msg334,
	}),
]);

var part328 = match_copy("MESSAGE#24:SYSTEM_MSG:08/0_1", "nwparser.payload", "event_description");

var part329 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/1_0", "nwparser.p0", "rol%{p0}");

var part330 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/1_1", "nwparser.p0", "ol%{p0}");

var part331 = match("MESSAGE#44:IF_RX_FLOW_CONTROL/2", "nwparser.p0", "%{}state changed to %{result}");

var part332 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/0", "nwparser.payload", "update:%{saddr}@%{terminal}:%{username}:%{p0}");

var part333 = match("MESSAGE#171:AAA_ACCOUNTING_MESSAGE:27/2", "nwparser.p0", "%{result})");

var part334 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/0", "nwparser.payload", "S%{p0}");

var part335 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/1_0", "nwparser.p0", "ource%{p0}");

var part336 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/1_1", "nwparser.p0", "rc%{p0}");

var part337 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/2", "nwparser.p0", "%{}IP: %{saddr}, D%{p0}");

var part338 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/3_0", "nwparser.p0", "estination%{p0}");

var part339 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/3_1", "nwparser.p0", "st%{p0}");

var part340 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/4", "nwparser.p0", "%{}IP: %{daddr}, S%{p0}");

var part341 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/6", "nwparser.p0", "%{}Port: %{sport}, D%{p0}");

var part342 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/8", "nwparser.p0", "%{}Port: %{dport}, S%{p0}");

var part343 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/9_0", "nwparser.p0", "ource Interface%{p0}");

var part344 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/9_1", "nwparser.p0", "rc Intf%{p0}");

var part345 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/10", "nwparser.p0", ": %{sinterface}, %{p0}");

var part346 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/11_0", "nwparser.p0", "Protocol: %{p0}");

var part347 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/11_1", "nwparser.p0", "protocol: %{p0}");

var part348 = match("MESSAGE#186:ACLLOG_FLOW_INTERVAL/12", "nwparser.p0", "\"%{protocol}\"(%{protocol_detail}),%{space->} Hit-count = %{dclass_counter1}");

var part349 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/0", "nwparser.payload", "%{action}: %{p0}");

var part350 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/1_0", "nwparser.p0", "%{saddr}@%{terminal}: %{p0}");

var part351 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/1_1", "nwparser.p0", "%{fld1->} %{p0}");

var part352 = match("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/3_0", "nwparser.p0", "(%{result})%{info}");

var part353 = match_copy("MESSAGE#372:TACACS_ACCOUNTING_MESSAGE:09/3_1", "nwparser.p0", "info");

var part354 = match("MESSAGE#238:IF_XCVR_WARNING/0", "nwparser.payload", "Interface %{interface}, %{p0}");

var part355 = match("MESSAGE#238:IF_XCVR_WARNING/1_0", "nwparser.p0", "Low %{p0}");

var part356 = match("MESSAGE#238:IF_XCVR_WARNING/1_1", "nwparser.p0", "High %{p0}");

var part357 = match_copy("MESSAGE#0:LOG-7-SYSTEM_MSG", "nwparser.payload", "event_description", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var part358 = match_copy("MESSAGE#32:NEIGHBOR_UPDATE_AUTOCOPY", "nwparser.payload", "event_description", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var part359 = match("MESSAGE#35:IF_DOWN_ADMIN_DOWN", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var part360 = match("MESSAGE#36:IF_DOWN_ADMIN_DOWN:01", "nwparser.payload", "%{fld43->} Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var part361 = match("MESSAGE#37:IF_DOWN_CHANNEL_MEMBERSHIP_UPDATE_IN_PROGRESS", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var part362 = match("MESSAGE#38:IF_DOWN_INTERFACE_REMOVED", "nwparser.payload", "Interface %{interface->} is down (%{result})", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var select44 = linear_select([
	dup26,
	dup27,
]);

var part363 = match_copy("MESSAGE#58:IM_SEQ_ERROR", "nwparser.payload", "result", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
]));

var part364 = match_copy("MESSAGE#88:PFM_VEM_REMOVE_NO_HB", "nwparser.payload", "event_description", processor_chain([
	dup24,
	dup2,
	dup3,
	dup4,
]));

var part365 = match("MESSAGE#108:IF_DOWN_INITIALIZING:01", "nwparser.payload", "%{fld43->} Interface %{interface->} is down (%{result})", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var part366 = match("MESSAGE#110:IF_DOWN_NONE:01", "nwparser.payload", "%{fld52->} Interface %{interface->} is down (%{result})", processor_chain([
	dup23,
	dup34,
	dup35,
	dup14,
	dup2,
	dup3,
	dup4,
]));

var part367 = match_copy("MESSAGE#123:PORT_PROFILE_CHANGE_VERIFY_REQ_FAILURE", "nwparser.payload", "event_description", processor_chain([
	dup33,
	dup2,
	dup3,
	dup4,
]));

var select45 = linear_select([
	dup46,
	dup47,
]);

var select46 = linear_select([
	dup49,
	dup50,
]);

var select47 = linear_select([
	dup54,
	dup55,
]);

var select48 = linear_select([
	dup57,
	dup58,
]);

var part368 = match_copy("MESSAGE#214:NOHMS_DIAG_ERR_PS_FAIL", "nwparser.payload", "event_description", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var select49 = linear_select([
	dup65,
	dup66,
]);

var select50 = linear_select([
	dup67,
	dup68,
]);

var part369 = match("MESSAGE#224:IF_SFP_WARNING", "nwparser.payload", "Interface %{interface}, %{event_description}", processor_chain([
	dup15,
	dup2,
	dup3,
	dup4,
]));

var part370 = match("MESSAGE#225:IF_DOWN_TCP_MAX_RETRANSMIT", "nwparser.payload", "%{fld43->} Interface %{interface->} is down%{info}", processor_chain([
	dup23,
	dup2,
	dup3,
	dup4,
]));

var select51 = linear_select([
	dup70,
	dup71,
]);

var part371 = match("MESSAGE#239:IF_XCVR_WARNING:01", "nwparser.payload", "Interface %{interface}, %{event_description}", processor_chain([
	dup61,
	dup2,
	dup3,
	dup4,
]));
