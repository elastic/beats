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

var dup1 = match("HEADER#3:0004/0", "message", "%{month->} %{day->} %{time->} %{p0}");

var dup2 = match("HEADER#3:0004/1_0", "nwparser.p0", "fpc0 %{p0}");

var dup3 = match("HEADER#3:0004/1_1", "nwparser.p0", "fpc1 %{p0}");

var dup4 = match("HEADER#3:0004/1_2", "nwparser.p0", "fpc2 %{p0}");

var dup5 = match("HEADER#3:0004/1_3", "nwparser.p0", "fpc3 %{p0}");

var dup6 = match("HEADER#3:0004/1_4", "nwparser.p0", "fpc4 %{p0}");

var dup7 = match("HEADER#3:0004/1_5", "nwparser.p0", "fpc5 %{p0}");

var dup8 = match("HEADER#3:0004/1_11", "nwparser.p0", "ssb %{p0}");

var dup9 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(": "),
		field("p0"),
	],
});

var dup10 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup11 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hfld2"),
		constant(" "),
		field("messageid"),
		constant(": "),
		field("p0"),
	],
});

var dup12 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hfld1"),
		constant("["),
		field("pid"),
		constant("]: "),
		field("messageid"),
		constant(": "),
		field("p0"),
	],
});

var dup13 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" ["),
		field("p0"),
	],
});

var dup14 = match("HEADER#15:0026.upd.a/1_0", "nwparser.p0", "RT_FLOW - %{p0}");

var dup15 = match("HEADER#15:0026.upd.a/1_1", "nwparser.p0", "junos-ssl-proxy - %{p0}");

var dup16 = match("HEADER#15:0026.upd.a/1_2", "nwparser.p0", "RT_APPQOS - %{p0}");

var dup17 = match("HEADER#15:0026.upd.a/1_3", "nwparser.p0", "%{hfld33->} - %{p0}");

var dup18 = match("HEADER#16:0026.upd.b/0", "message", "%{event_time->} %{hfld32->} %{hhostname->} %{p0}");

var dup19 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant("["),
		field("pid"),
		constant("]: "),
		field("p0"),
	],
});

var dup20 = setc("messageid","JUNOSROUTER_GENERIC");

var dup21 = setc("eventcategory","1605000000");

var dup22 = setf("msg","$MSG");

var dup23 = date_time({
	dest: "event_time",
	args: ["month","day","time"],
	fmts: [
		[dB,dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup24 = setf("hostname","hhost");

var dup25 = setc("event_description","AUDIT");

var dup26 = setc("event_description","CRON command");

var dup27 = setc("eventcategory","1801030000");

var dup28 = setc("eventcategory","1801020000");

var dup29 = setc("eventcategory","1605010000");

var dup30 = setc("eventcategory","1603000000");

var dup31 = setc("event_description","Process mode");

var dup32 = setc("event_description","NTP Server Unreachable");

var dup33 = setc("eventcategory","1401060000");

var dup34 = setc("ec_theme","Authentication");

var dup35 = setc("ec_subject","User");

var dup36 = setc("ec_activity","Logon");

var dup37 = setc("ec_outcome","Success");

var dup38 = setc("event_description","rpd proceeding");

var dup39 = match("MESSAGE#77:sshd:06/0", "nwparser.payload", "%{} %{p0}");

var dup40 = match("MESSAGE#77:sshd:06/1_0", "nwparser.p0", "%{process}[%{process_id}]: %{p0}");

var dup41 = match("MESSAGE#77:sshd:06/1_1", "nwparser.p0", "%{process}: %{p0}");

var dup42 = setc("eventcategory","1701010000");

var dup43 = setc("ec_outcome","Failure");

var dup44 = setc("eventcategory","1401030000");

var dup45 = match_copy("MESSAGE#72:Failed:05/1_2", "nwparser.p0", "p0");

var dup46 = setc("eventcategory","1803000000");

var dup47 = setc("event_type","VPN");

var dup48 = setc("eventcategory","1605020000");

var dup49 = setc("eventcategory","1602020000");

var dup50 = match("MESSAGE#114:ACCT_GETHOSTNAME_error/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{p0}");

var dup51 = setc("eventcategory","1603020000");

var dup52 = date_time({
	dest: "event_time",
	args: ["hfld32"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dc("T"),dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup53 = setc("ec_subject","NetworkComm");

var dup54 = setc("ec_activity","Create");

var dup55 = setc("ec_activity","Stop");

var dup56 = setc("event_description","Trap state change");

var dup57 = setc("event_description","peer NLRI mismatch");

var dup58 = setc("eventcategory","1605030000");

var dup59 = setc("eventcategory","1603010000");

var dup60 = setc("eventcategory","1606000000");

var dup61 = setf("hostname","hhostname");

var dup62 = date_time({
	dest: "event_time",
	args: ["hfld6"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dc("T"),dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup63 = setc("eventcategory","1401050200");

var dup64 = setc("event_description","Memory allocation failed during initialization for configuration load");

var dup65 = setc("event_description","unable to run in the background as a daemon");

var dup66 = setc("event_description","Another copy of this program is running");

var dup67 = setc("event_description","Unable to lock PID file");

var dup68 = setc("event_description","Unable to update process PID file");

var dup69 = setc("eventcategory","1301000000");

var dup70 = setc("event_description","Command stopped");

var dup71 = setc("event_description","Unable to create pipes for command");

var dup72 = setc("event_description","Command exited");

var dup73 = setc("eventcategory","1603050000");

var dup74 = setc("eventcategory","1801010000");

var dup75 = setc("event_description","Login failure");

var dup76 = match("MESSAGE#294:LOGIN_INFORMATION/3_0", "nwparser.p0", "User %{p0}");

var dup77 = match("MESSAGE#294:LOGIN_INFORMATION/3_1", "nwparser.p0", "user %{p0}");

var dup78 = setc("event_description","Unable to open file");

var dup79 = setc("event_description","SNMP index assigned changed");

var dup80 = setc("eventcategory","1302000000");

var dup81 = setc("eventcategory","1001020300");

var dup82 = setc("event_description","PFE FW SYSLOG_IP");

var dup83 = setc("event_description","process_mode");

var dup84 = setc("event_description","Logical interface collision");

var dup85 = setc("event_description","excessive runtime time during action of module");

var dup86 = setc("event_description","Reinitializing");

var dup87 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/0", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\"%{p0}");

var dup88 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/1_0", "nwparser.p0", " connection-tag=%{fld20->} service-name=\"%{p0}");

var dup89 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/1_1", "nwparser.p0", " service-name=\"%{p0}");

var dup90 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/3_0", "nwparser.p0", " nat-connection-tag=%{fld6->} src-nat-rule-type=%{fld20->} %{p0}");

var dup91 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/5_1", "nwparser.p0", "name=\"%{p0}");

var dup92 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/8", "nwparser.p0", "]%{}");

var dup93 = setc("eventcategory","1803010000");

var dup94 = setc("ec_activity","Deny");

var dup95 = match("MESSAGE#490:RT_FLOW_SESSION_DENY:03/0_0", "nwparser.payload", "%{process}: %{event_type}: session denied %{p0}");

var dup96 = match("MESSAGE#490:RT_FLOW_SESSION_DENY:03/0_1", "nwparser.payload", "%{event_type}: session denied %{p0}");

var dup97 = setc("event_description","session denied");

var dup98 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/0", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} reason=\"%{result}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\"%{p0}");

var dup99 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/2", "nwparser.p0", "%{service}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\"%{p0}");

var dup100 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/4", "nwparser.p0", "%{}src-nat-rule-name=\"%{rulename}\" dst-nat-rule-%{p0}");

var dup101 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/5_0", "nwparser.p0", "type=%{fld7->} dst-nat-rule-name=\"%{p0}");

var dup102 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/6", "nwparser.p0", "\"%{rule_template->} protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{sessionid}\" packets-from-client=\"%{packets}\" bytes-from-client=\"%{rbytes}\" packets-from-server=\"%{dclass_counter1}\" bytes-from-server=\"%{sbytes}\" elapsed-time=\"%{duration}\"%{p0}");

var dup103 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/7_0", "nwparser.p0", " application=\"%{fld6}\" nested-application=\"%{fld7}\" username=\"%{username}\" roles=\"%{fld15}\" packet-incoming-interface=\"%{dinterface}\" encrypted=%{fld16->} %{p0}");

var dup104 = setc("dclass_counter1_string","No.of packets from client");

var dup105 = setc("event_description","SNMPD AUTH FAILURE");

var dup106 = setc("event_description","send send-type (index1) failure");

var dup107 = setc("event_description","SNMP trap error");

var dup108 = setc("event_description","SNMP TRAP LINK DOWN");

var dup109 = setc("event_description","SNMP TRAP LINK UP");

var dup110 = setc("event_description","Login Failure");

var dup111 = match("MESSAGE#630:UI_CFG_AUDIT_OTHER:02/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' set: [%{action}] %{p0}");

var dup112 = match_copy("MESSAGE#630:UI_CFG_AUDIT_OTHER:02/1_1", "nwparser.p0", "space");

var dup113 = setc("eventcategory","1701020000");

var dup114 = match("MESSAGE#634:UI_CFG_AUDIT_SET:01/1_1", "nwparser.p0", "\u003c\u003c%{change_old}> %{p0}");

var dup115 = match("MESSAGE#634:UI_CFG_AUDIT_SET:01/2", "nwparser.p0", "-> \"%{change_new}\"");

var dup116 = setc("event_description","User set command");

var dup117 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' %{p0}");

var dup118 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/1_0", "nwparser.p0", "set %{p0}");

var dup119 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/1_1", "nwparser.p0", "replace %{p0}");

var dup120 = setc("event_description","User set groups to secret");

var dup121 = setc("event_description","UI CMDLINE READ LINE");

var dup122 = setc("event_description","User commit");

var dup123 = match("MESSAGE#675:UI_DAEMON_ACCEPT_FAILED/1_0", "nwparser.p0", "Network %{p0}");

var dup124 = match("MESSAGE#675:UI_DAEMON_ACCEPT_FAILED/1_1", "nwparser.p0", "Local %{p0}");

var dup125 = setc("eventcategory","1401070000");

var dup126 = setc("ec_activity","Logoff");

var dup127 = setc("event_description","Successful login");

var dup128 = setf("hostname","hostip");

var dup129 = setc("event_description","TACACS+ failure");

var dup130 = match("MESSAGE#755:node:05/0", "nwparser.payload", "%{hostname->} %{node->} %{p0}");

var dup131 = match("MESSAGE#755:node:05/1_0", "nwparser.p0", "partner%{p0}");

var dup132 = match("MESSAGE#755:node:05/1_1", "nwparser.p0", "actor%{p0}");

var dup133 = setc("eventcategory","1003010000");

var dup134 = setc("eventcategory","1901000000");

var dup135 = linear_select([
	dup14,
	dup15,
	dup16,
	dup17,
]);

var dup136 = match("HEADER#15:0026.upd.a/2", "nwparser.p0", "%{messageid->} [%{p0}", processor_chain([
	dup13,
]));

var dup137 = linear_select([
	dup40,
	dup41,
]);

var dup138 = match("MESSAGE#125:BFDD_TRAP_STATE_DOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: local discriminator: %{resultcode}, new state: %{result}", processor_chain([
	dup21,
	dup22,
	dup56,
	dup23,
]));

var dup139 = match("MESSAGE#214:DCD_MALLOC_FAILED_INIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Memory allocation failed during initialization for configuration load", processor_chain([
	dup51,
	dup22,
	dup64,
	dup23,
]));

var dup140 = match("MESSAGE#225:ECCD_DAEMONIZE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}, unable to run in the background as a daemon: %{result}", processor_chain([
	dup30,
	dup22,
	dup65,
	dup23,
]));

var dup141 = match("MESSAGE#226:ECCD_DUPLICATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Another copy of this program is running", processor_chain([
	dup30,
	dup22,
	dup66,
	dup23,
]));

var dup142 = match("MESSAGE#232:ECCD_PID_FILE_LOCK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to lock PID file: %{result}", processor_chain([
	dup30,
	dup22,
	dup67,
	dup23,
]));

var dup143 = match("MESSAGE#233:ECCD_PID_FILE_UPDATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to update process PID file: %{result}", processor_chain([
	dup30,
	dup22,
	dup68,
	dup23,
]));

var dup144 = match("MESSAGE#272:LIBJNX_EXEC_PIPE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to create pipes for command '%{action}': %{result}", processor_chain([
	dup30,
	dup22,
	dup71,
	dup23,
]));

var dup145 = linear_select([
	dup76,
	dup77,
]);

var dup146 = match("MESSAGE#310:MIB2D_IFD_IFINDEX_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: SNMP index assigned to %{uid->} changed from %{dclass_counter1->} to %{result}", processor_chain([
	dup30,
	dup22,
	dup79,
	dup23,
]));

var dup147 = match("MESSAGE#412:RPD_IFL_INDEXCOLLISION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Logical interface collision -- %{result}, %{info}", processor_chain([
	dup30,
	dup22,
	dup84,
	dup23,
]));

var dup148 = match("MESSAGE#466:RPD_SCHED_CALLBACK_LONGRUNTIME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: excessive runtime time during action of module", processor_chain([
	dup30,
	dup22,
	dup85,
	dup23,
]));

var dup149 = match("MESSAGE#482:RPD_TASK_REINIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reinitializing", processor_chain([
	dup21,
	dup22,
	dup86,
	dup23,
]));

var dup150 = linear_select([
	dup88,
	dup89,
]);

var dup151 = linear_select([
	dup90,
	dup45,
]);

var dup152 = linear_select([
	dup95,
	dup96,
]);

var dup153 = linear_select([
	dup101,
	dup91,
]);

var dup154 = match("MESSAGE#498:RT_SCREEN_TCP", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} attack-name=\"%{threat_name}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" source-zone-name=\"%{src_zone}\" interface-name=\"%{interface}\" action=\"%{action}\"]", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var dup155 = match("MESSAGE#527:SSL_PROXY_SSL_SESSION_ALLOW", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} logical-system-name=\"%{hostname}\" session-id=\"%{sessionid}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" profile-name=\"%{rulename}\" source-zone-name=\"%{src_zone}\" source-interface-name=\"%{sinterface}\" destination-zone-name=\"%{dst_zone}\" destination-interface-name=\"%{dinterface}\" message=\"%{info}\"]", processor_chain([
	dup27,
	dup22,
	dup52,
]));

var dup156 = linear_select([
	dup118,
	dup119,
]);

var dup157 = linear_select([
	dup123,
	dup124,
]);

var dup158 = match("MESSAGE#733:WEBFILTER_URL_PERMITTED", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=\"%{directory}\" username=\"%{username}\" roles=\"%{user_role}\"] WebFilter: ACTION=\"%{action}\" %{fld2}->%{fld3->} CATEGORY=\"%{category}\" REASON=\"%{fld4}\" PROFILE=\"%{fld6}\" URL=%{url->} OBJ=%{fld7->} USERNAME=%{fld8->} ROLES=%{fld9}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var dup159 = match_copy("MESSAGE#747:cli", "nwparser.payload", "fld12", processor_chain([
	dup48,
	dup47,
	dup23,
	dup22,
]));

var hdr1 = match("HEADER#0:0001", "message", "%{month->} %{day->} %{time->} %{messageid}: restart %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(": restart "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%{month->} %{day->} %{time->} %{messageid->} message repeated %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" message repeated "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "%{month->} %{day->} %{time->} ssb %{messageid}(%{hfld1}): %{p0}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("("),
			field("hfld1"),
			constant("): "),
			field("p0"),
		],
	}),
]));

var part1 = match("HEADER#3:0004/1_6", "nwparser.p0", "fpc6 %{p0}");

var part2 = match("HEADER#3:0004/1_7", "nwparser.p0", "fpc7 %{p0}");

var part3 = match("HEADER#3:0004/1_8", "nwparser.p0", "fpc8 %{p0}");

var part4 = match("HEADER#3:0004/1_9", "nwparser.p0", "fpc9 %{p0}");

var part5 = match("HEADER#3:0004/1_10", "nwparser.p0", "cfeb %{p0}");

var select1 = linear_select([
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	part1,
	part2,
	part3,
	part4,
	part5,
	dup8,
]);

var part6 = match("HEADER#3:0004/2", "nwparser.p0", "%{} %{messageid}: %{p0}", processor_chain([
	dup9,
]));

var all1 = all_match({
	processors: [
		dup1,
		select1,
		part6,
	],
	on_success: processor_chain([
		setc("header_id","0004"),
	]),
});

var select2 = linear_select([
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
]);

var part7 = match("HEADER#4:0005/2", "nwparser.p0", "%{} %{messageid->} %{p0}", processor_chain([
	dup10,
]));

var all2 = all_match({
	processors: [
		dup1,
		select2,
		part7,
	],
	on_success: processor_chain([
		setc("header_id","0005"),
	]),
});

var hdr4 = match("HEADER#5:0007", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhost}: %{hfld2}[%{hpid}]: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0007"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("["),
			field("hpid"),
			constant("]: "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr5 = match("HEADER#6:0008", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhost}: %{messageid}[%{hpid}]: %{p0}", processor_chain([
	setc("header_id","0008"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("["),
			field("hpid"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr6 = match("HEADER#7:0009", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhost}: %{hfld2->} IFP trace> %{messageid}: %{p0}", processor_chain([
	setc("header_id","0009"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant(" IFP trace> "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr7 = match("HEADER#8:0010", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhost}: %{hfld2->} %{messageid}: %{p0}", processor_chain([
	setc("header_id","0010"),
	dup11,
]));

var hdr8 = match("HEADER#9:0029", "message", "%{month->} %{day->} %{time->} %{hostip->} %{hfld1}[%{pid}]: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0029"),
	dup12,
]));

var hdr9 = match("HEADER#10:0015", "message", "%{month->} %{day->} %{time->} %{hfld1}[%{pid}]: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0015"),
	dup12,
]));

var hdr10 = match("HEADER#11:0011", "message", "%{month->} %{day->} %{time->} %{hfld2->} %{messageid}: %{p0}", processor_chain([
	setc("header_id","0011"),
	dup11,
]));

var hdr11 = match("HEADER#12:0027", "message", "%{month->} %{day->} %{time->} %{hhostname->} RT_FLOW: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0027"),
	dup9,
]));

var hdr12 = match("HEADER#13:0012", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhost}: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0012"),
	dup9,
]));

var hdr13 = match("HEADER#14:0013", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hfld32->} %{hhostname->} RT_FLOW - %{messageid->} [%{p0}", processor_chain([
	setc("header_id","0013"),
	dup13,
]));

var hdr14 = match("HEADER#15:0026.upd.a/0", "message", "%{hfld1->} %{event_time->} %{hfld32->} %{hhostname->} %{p0}");

var all3 = all_match({
	processors: [
		hdr14,
		dup135,
		dup136,
	],
	on_success: processor_chain([
		setc("header_id","0026.upd.a"),
	]),
});

var all4 = all_match({
	processors: [
		dup18,
		dup135,
		dup136,
	],
	on_success: processor_chain([
		setc("header_id","0026.upd.b"),
	]),
});

var all5 = all_match({
	processors: [
		dup18,
		dup135,
		dup136,
	],
	on_success: processor_chain([
		setc("header_id","0026"),
	]),
});

var hdr15 = match("HEADER#18:0014", "message", "%{month->} %{day->} %{time->} %{hfld1}[%{pid}]: %{messageid}[%{hpid}]: %{p0}", processor_chain([
	setc("header_id","0014"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant("["),
			field("pid"),
			constant("]: "),
			field("messageid"),
			constant("["),
			field("hpid"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr16 = match("HEADER#19:0016", "message", "%{month->} %{day->} %{time->} %{hfld1}: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0016"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant(": "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr17 = match("HEADER#20:0017", "message", "%{month->} %{day->} %{time->} %{hfld1}[%{pid}]: %{messageid->} %{p0}", processor_chain([
	setc("header_id","0017"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant("["),
			field("pid"),
			constant("]: "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr18 = match("HEADER#21:0018", "message", "%{month->} %{day->} %{time->} %{hhost}: %{messageid}[%{pid}]: %{p0}", processor_chain([
	setc("header_id","0018"),
	dup19,
]));

var hdr19 = match("HEADER#22:0028", "message", "%{month->} %{day->} %{time->} %{hhost->} %{messageid}[%{pid}]: %{p0}", processor_chain([
	setc("header_id","0028"),
	dup19,
]));

var hdr20 = match("HEADER#23:0019", "message", "%{month->} %{day->} %{time->} %{hhost}: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0019"),
	dup9,
]));

var hdr21 = match("HEADER#24:0020", "message", "%{month->} %{day->} %{time->} %{messageid}[%{pid}]: %{p0}", processor_chain([
	setc("header_id","0020"),
	dup19,
]));

var hdr22 = match("HEADER#25:0021", "message", "%{month->} %{day->} %{time->} /%{messageid}: %{p0}", processor_chain([
	setc("header_id","0021"),
	dup9,
]));

var hdr23 = match("HEADER#26:0022", "message", "%{month->} %{day->} %{time->} %{messageid}: %{p0}", processor_chain([
	setc("header_id","0022"),
	dup9,
]));

var hdr24 = match("HEADER#27:0023", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhostname}: %{messageid}[%{pid}]: %{p0}", processor_chain([
	setc("header_id","0023"),
	dup19,
]));

var hdr25 = match("HEADER#28:0024", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhostname}: %{messageid}: %{p0}", processor_chain([
	setc("header_id","0024"),
	dup9,
]));

var hdr26 = match("HEADER#29:0025", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhostname}: %{hfld2->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0025"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr27 = match("HEADER#30:0031", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhostname}: %{messageid->} %{p0}", processor_chain([
	setc("header_id","0031"),
	dup10,
]));

var hdr28 = match("HEADER#31:0032", "message", "%{month->} %{day->} %{time->} %{hostip->} (%{hfld1}) %{hfld2->} %{messageid}[%{pid}]: %{p0}", processor_chain([
	setc("header_id","0032"),
	dup19,
]));

var hdr29 = match("HEADER#32:0033", "message", "%{month->} %{day->} %{time->} %{hfld1->} %{hhostname->} %{messageid}: %{p0}", processor_chain([
	setc("header_id","0033"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant(" "),
			field("hhostname"),
			constant(" "),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr30 = match("HEADER#33:3336", "message", "%{month->} %{day->} %{time->} %{hhost->} %{process}[%{process_id}]: %{messageid}: %{payload}", processor_chain([
	setc("header_id","3336"),
]));

var hdr31 = match("HEADER#34:3339", "message", "%{month->} %{day->} %{time->} %{hhost->} %{process}[%{process_id}]: %{messageid->} %{payload}", processor_chain([
	setc("header_id","3339"),
]));

var hdr32 = match("HEADER#35:3337", "message", "%{month->} %{day->} %{time->} %{hhost->} %{messageid}: %{payload}", processor_chain([
	setc("header_id","3337"),
]));

var hdr33 = match("HEADER#36:3341", "message", "%{hfld1->} %{hfld6->} %{hhostname->} %{hfld2->} %{hfld3->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","3341"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr34 = match("HEADER#37:3338", "message", "%{month->} %{day->} %{time->} %{hhost->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","3338"),
]));

var hdr35 = match("HEADER#38:3340/0", "message", "%{month->} %{day->} %{time->} %{hhost->} node%{hfld1}.fpc%{p0}", processor_chain([
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hhost"),
			constant(" node"),
			field("hfld1"),
			constant(".fpc"),
			field("p0"),
		],
	}),
]));

var part8 = match("HEADER#38:3340/1_0", "nwparser.p0", "%{hfld2}.pic%{hfld3->} %{p0}");

var part9 = match("HEADER#38:3340/1_1", "nwparser.p0", "%{hfld2->} %{p0}");

var select3 = linear_select([
	part8,
	part9,
]);

var part10 = match("HEADER#38:3340/2", "nwparser.p0", "%{} %{p0}");

var all6 = all_match({
	processors: [
		hdr35,
		select3,
		part10,
	],
	on_success: processor_chain([
		setc("header_id","3340"),
		setc("messageid","node"),
	]),
});

var hdr36 = match("HEADER#39:9997/0_0", "message", "mgd[%{p0}");

var hdr37 = match("HEADER#39:9997/0_1", "message", "rpd[%{p0}");

var hdr38 = match("HEADER#39:9997/0_2", "message", "dcd[%{p0}");

var select4 = linear_select([
	hdr36,
	hdr37,
	hdr38,
]);

var part11 = match("HEADER#39:9997/1", "nwparser.p0", "%{process_id}]:%{payload}");

var all7 = all_match({
	processors: [
		select4,
		part11,
	],
	on_success: processor_chain([
		setc("header_id","9997"),
		dup20,
	]),
});

var hdr39 = match("HEADER#40:9995", "message", "%{month->} %{day->} %{time->} %{hhost->} %{hfld1->} %{hfld2->} %{messageid}[%{hfld3}]:%{p0}", processor_chain([
	setc("header_id","9995"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("["),
			field("hfld3"),
			constant("]:"),
			field("p0"),
		],
	}),
]));

var hdr40 = match("HEADER#41:9994", "message", "%{month->} %{day->} %{time->} %{hfld2->} %{hfld1->} qsfp %{p0}", processor_chain([
	setc("header_id","9994"),
	setc("messageid","qsfp"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant(" "),
			field("hfld1"),
			constant(" qsfp "),
			field("p0"),
		],
	}),
]));

var hdr41 = match("HEADER#42:9999", "message", "%{month->} %{day->} %{time->} %{hhost->} %{process}[%{process_id}]: %{hevent_type}: %{p0}", processor_chain([
	setc("header_id","9999"),
	dup20,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hevent_type"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr42 = match("HEADER#43:9998", "message", "%{month->} %{day->} %{time->} %{hfld2->} %{process}: %{p0}", processor_chain([
	setc("header_id","9998"),
	dup20,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant(" "),
			field("process"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var select5 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	all1,
	all2,
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
	all3,
	all4,
	all5,
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
	hdr27,
	hdr28,
	hdr29,
	hdr30,
	hdr31,
	hdr32,
	hdr33,
	hdr34,
	all6,
	all7,
	hdr39,
	hdr40,
	hdr41,
	hdr42,
]);

var part12 = match("MESSAGE#0:/usr/sbin/sshd", "nwparser.payload", "%{process}[%{process_id}]: %{agent}[%{id}]: exit status %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","sshd exit status"),
	dup23,
]));

var msg1 = msg("/usr/sbin/sshd", part12);

var part13 = match("MESSAGE#1:/usr/libexec/telnetd", "nwparser.payload", "%{process}[%{process_id}]: %{agent}[%{id}]: exit status %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","telnetd exit status"),
	dup23,
]));

var msg2 = msg("/usr/libexec/telnetd", part13);

var part14 = match("MESSAGE#2:alarmd", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: License color=%{severity}, class=%{device}, reason=%{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Alarm Set or Cleared"),
	dup23,
]));

var msg3 = msg("alarmd", part14);

var part15 = match("MESSAGE#3:bigd", "nwparser.payload", "%{process}: Node detected UP for %{node}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Node detected UP"),
	dup23,
]));

var msg4 = msg("bigd", part15);

var part16 = match("MESSAGE#4:bigd:01", "nwparser.payload", "%{process}: Monitor template id is %{id}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Monitor template id"),
	dup23,
]));

var msg5 = msg("bigd:01", part16);

var select6 = linear_select([
	msg4,
	msg5,
]);

var part17 = match("MESSAGE#5:bigpipe", "nwparser.payload", "%{process}: Loading the configuration file %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Loading configuration file"),
	dup23,
]));

var msg6 = msg("bigpipe", part17);

var part18 = match("MESSAGE#6:bigpipe:01", "nwparser.payload", "%{process}: Begin config install operation %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Begin config install operation"),
	dup23,
]));

var msg7 = msg("bigpipe:01", part18);

var part19 = match("MESSAGE#7:bigpipe:02", "nwparser.payload", "%{process}: AUDIT -- Action %{action->} User: %{username}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Audit"),
	dup23,
]));

var msg8 = msg("bigpipe:02", part19);

var select7 = linear_select([
	msg6,
	msg7,
	msg8,
]);

var part20 = match("MESSAGE#8:bigstart", "nwparser.payload", "%{process}: shutdown %{service}", processor_chain([
	dup21,
	dup22,
	setc("event_description","portal shutdown"),
	dup23,
]));

var msg9 = msg("bigstart", part20);

var part21 = match("MESSAGE#9:cgatool", "nwparser.payload", "%{process}: %{event_type}: generated address is %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","cga address genration"),
	dup23,
]));

var msg10 = msg("cgatool", part21);

var part22 = match("MESSAGE#10:chassisd:01", "nwparser.payload", "%{process}[%{process_id}]:%{fld12}", processor_chain([
	dup21,
	dup22,
	dup23,
	dup24,
]));

var msg11 = msg("chassisd:01", part22);

var part23 = match("MESSAGE#11:checkd", "nwparser.payload", "%{process}: AUDIT -- Action %{action->} User: %{username}", processor_chain([
	dup21,
	dup22,
	dup25,
	dup23,
]));

var msg12 = msg("checkd", part23);

var part24 = match("MESSAGE#12:checkd:01", "nwparser.payload", "%{process}: exiting", processor_chain([
	dup21,
	dup22,
	setc("event_description","checkd exiting"),
	dup23,
]));

var msg13 = msg("checkd:01", part24);

var select8 = linear_select([
	msg12,
	msg13,
]);

var part25 = match("MESSAGE#13:cosd", "nwparser.payload", "%{process}[%{process_id}]: link protection %{dclass_counter1->} for intf %{interface}", processor_chain([
	dup21,
	dup22,
	setc("event_description","link protection for interface"),
	dup23,
]));

var msg14 = msg("cosd", part25);

var part26 = match("MESSAGE#14:craftd", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}, %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","License expiration warning"),
	dup23,
]));

var msg15 = msg("craftd", part26);

var part27 = match("MESSAGE#15:CRON/0", "nwparser.payload", "%{process}[%{process_id}]: (%{username}) %{p0}");

var part28 = match("MESSAGE#15:CRON/1_0", "nwparser.p0", "CMD (%{result})");

var part29 = match("MESSAGE#15:CRON/1_1", "nwparser.p0", "cmd='%{result}'");

var select9 = linear_select([
	part28,
	part29,
]);

var all8 = all_match({
	processors: [
		part27,
		select9,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup26,
		dup23,
	]),
});

var msg16 = msg("CRON", all8);

var part30 = match("MESSAGE#16:Cmerror/0_0", "nwparser.payload", "%{hostname->} %{node}Cmerror: Level%{level}count increment %{dclass_counter1->} %{fld1}");

var part31 = match_copy("MESSAGE#16:Cmerror/0_1", "nwparser.payload", "fld2");

var select10 = linear_select([
	part30,
	part31,
]);

var all9 = all_match({
	processors: [
		select10,
	],
	on_success: processor_chain([
		dup21,
		dup23,
		dup22,
	]),
});

var msg17 = msg("Cmerror", all9);

var part32 = match("MESSAGE#17:cron", "nwparser.payload", "%{process}[%{process_id}]: (%{username}) %{action->} (%{filename})", processor_chain([
	dup21,
	dup22,
	setc("event_description","cron RELOAD"),
	dup23,
]));

var msg18 = msg("cron", part32);

var part33 = match("MESSAGE#18:CROND", "nwparser.payload", "%{process}[%{process_id}]: (%{username}) CMD (%{action})", processor_chain([
	dup21,
	dup22,
	dup23,
	dup24,
]));

var msg19 = msg("CROND", part33);

var part34 = match("MESSAGE#20:CROND:02", "nwparser.payload", "%{process}[%{process_id}]: pam_unix(crond:session): session closed for user %{username}", processor_chain([
	dup27,
	dup22,
	dup23,
	dup24,
]));

var msg20 = msg("CROND:02", part34);

var select11 = linear_select([
	msg19,
	msg20,
]);

var part35 = match("MESSAGE#19:crond:01", "nwparser.payload", "%{process}[%{process_id}]: pam_unix(crond:session): session opened for user %{username->} by (uid=%{uid})", processor_chain([
	dup28,
	dup22,
	dup23,
	dup24,
]));

var msg21 = msg("crond:01", part35);

var part36 = match("MESSAGE#21:dcd", "nwparser.payload", "%{process}[%{process_id}]: %{result->} Setting ignored, %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Setting ignored"),
	dup23,
]));

var msg22 = msg("dcd", part36);

var part37 = match("MESSAGE#22:EVENT/0", "nwparser.payload", "%{process}[%{process_id}]: EVENT %{event_type->} %{interface->} index %{resultcode->} %{p0}");

var part38 = match("MESSAGE#22:EVENT/1_0", "nwparser.p0", "%{saddr->} -> %{daddr->} \u003c\u003c%{p0}");

var part39 = match("MESSAGE#22:EVENT/1_1", "nwparser.p0", "\u003c\u003c%{p0}");

var select12 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#22:EVENT/2", "nwparser.p0", ">%{result}");

var all10 = all_match({
	processors: [
		part37,
		select12,
		part40,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","EVENT"),
		dup23,
	]),
});

var msg23 = msg("EVENT", all10);

var part41 = match("MESSAGE#23:ftpd", "nwparser.payload", "%{process}[%{process_id}]: connection from %{saddr->} (%{shost})", processor_chain([
	setc("eventcategory","1802000000"),
	dup22,
	setc("event_description","ftpd connection"),
	dup23,
]));

var msg24 = msg("ftpd", part41);

var part42 = match("MESSAGE#24:ha_rto_stats_handler", "nwparser.payload", "%{hostname->} %{node}ha_rto_stats_handler:%{fld12}", processor_chain([
	dup29,
	dup23,
	dup22,
]));

var msg25 = msg("ha_rto_stats_handler", part42);

var part43 = match("MESSAGE#25:hostinit", "nwparser.payload", "%{process}: %{obj_name->} -- LDAP Connection not bound correctly. %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","LDAP Connection not bound correctly"),
	dup23,
]));

var msg26 = msg("hostinit", part43);

var part44 = match("MESSAGE#26:ifinfo", "nwparser.payload", "%{process}: %{service}: PIC_INFO debug> Added entry - %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PIC_INFO debug - Added entry"),
	dup23,
]));

var msg27 = msg("ifinfo", part44);

var part45 = match("MESSAGE#27:ifinfo:01", "nwparser.payload", "%{process}: %{service}: PIC_INFO debug> Initializing spu listtype %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PIC_INFO debug Initializing spu"),
	dup23,
]));

var msg28 = msg("ifinfo:01", part45);

var part46 = match("MESSAGE#28:ifinfo:02", "nwparser.payload", "%{process}: %{service}: PIC_INFO debug> %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PIC_INFO debug delete from list"),
	dup23,
]));

var msg29 = msg("ifinfo:02", part46);

var select13 = linear_select([
	msg27,
	msg28,
	msg29,
]);

var part47 = match("MESSAGE#29:ifp_ifl_anydown_change_event", "nwparser.payload", "%{node->} %{action}> %{process}: IFL anydown change event: \"%{event_type}\"", processor_chain([
	dup21,
	dup22,
	setc("event_description","IFL anydown change event"),
	dup23,
]));

var msg30 = msg("ifp_ifl_anydown_change_event", part47);

var part48 = match("MESSAGE#30:ifp_ifl_config_event", "nwparser.payload", "%{node->} %{action}> %{process}: IFL config: \"%{filename}\"", processor_chain([
	dup21,
	dup22,
	setc("event_description","ifp ifl config_event"),
	dup23,
]));

var msg31 = msg("ifp_ifl_config_event", part48);

var part49 = match("MESSAGE#31:ifp_ifl_ext_chg", "nwparser.payload", "%{node->} %{process}: ifp ext piid %{parent_pid->} zone_id %{zone}", processor_chain([
	dup21,
	dup22,
	setc("event_description","ifp_ifl_ext_chg"),
	dup23,
]));

var msg32 = msg("ifp_ifl_ext_chg", part49);

var part50 = match("MESSAGE#32:inetd", "nwparser.payload", "%{process}[%{process_id}]: %{protocol->} from %{saddr->} exceeded counts/min (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","connection exceeded count limit"),
	dup23,
]));

var msg33 = msg("inetd", part50);

var part51 = match("MESSAGE#33:inetd:01", "nwparser.payload", "%{process}[%{process_id}]: %{agent}[%{id}]: exited, status %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","exited"),
	dup23,
]));

var msg34 = msg("inetd:01", part51);

var select14 = linear_select([
	msg33,
	msg34,
]);

var part52 = match("MESSAGE#34:init:04", "nwparser.payload", "%{process}: %{event_type->} current_mode=%{protocol}, requested_mode=%{result}, cmd=%{action}", processor_chain([
	dup21,
	dup22,
	dup31,
	dup23,
]));

var msg35 = msg("init:04", part52);

var part53 = match("MESSAGE#35:init", "nwparser.payload", "%{process}: %{event_type->} mode=%{protocol->} cmd=%{action->} master_mode=%{result}", processor_chain([
	dup21,
	dup22,
	dup31,
	dup23,
]));

var msg36 = msg("init", part53);

var part54 = match("MESSAGE#36:init:01", "nwparser.payload", "%{process}: failure target for routing set to %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","failure target for routing set"),
	dup23,
]));

var msg37 = msg("init:01", part54);

var part55 = match("MESSAGE#37:init:02", "nwparser.payload", "%{process}: ntp (PID %{child_pid}) started", processor_chain([
	dup21,
	dup22,
	setc("event_description","ntp started"),
	dup23,
]));

var msg38 = msg("init:02", part55);

var part56 = match("MESSAGE#38:init:03", "nwparser.payload", "%{process}: product mask %{info->} model %{dclass_counter1}", processor_chain([
	dup21,
	dup22,
	setc("event_description","product mask and model info"),
	dup23,
]));

var msg39 = msg("init:03", part56);

var select15 = linear_select([
	msg35,
	msg36,
	msg37,
	msg38,
	msg39,
]);

var part57 = match("MESSAGE#39:ipc_msg_write", "nwparser.payload", "%{node->} %{process}: IPC message type: %{event_type}, subtype: %{resultcode->} exceeds MTU, mtu %{dclass_counter1}, length %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IPC message exceeds MTU"),
	dup23,
]));

var msg40 = msg("ipc_msg_write", part57);

var part58 = match("MESSAGE#40:connection_established", "nwparser.payload", "%{process}: %{service}: conn established: listener idx=%{dclass_counter1->} tnpaddr=%{dclass_counter2}", processor_chain([
	dup28,
	dup22,
	setc("event_description","listener connection established"),
	dup23,
]));

var msg41 = msg("connection_established", part58);

var part59 = match("MESSAGE#41:connection_dropped/0", "nwparser.payload", "%{process}: %{p0}");

var part60 = match("MESSAGE#41:connection_dropped/1_0", "nwparser.p0", "%{result}, connection dropped - src %{saddr}:%{sport->} dest %{daddr}:%{dport}");

var part61 = match("MESSAGE#41:connection_dropped/1_1", "nwparser.p0", "%{result}: conn dropped: listener idx=%{dclass_counter1->} tnpaddr=%{dclass_counter2}");

var select16 = linear_select([
	part60,
	part61,
]);

var all11 = all_match({
	processors: [
		part59,
		select16,
	],
	on_success: processor_chain([
		dup27,
		dup22,
		setc("event_description","connection dropped"),
		dup23,
	]),
});

var msg42 = msg("connection_dropped", all11);

var part62 = match("MESSAGE#42:kernel", "nwparser.payload", "%{process}: %{interface}: Asserting SONET alarm(s) %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Asserting SONET alarm(s)"),
	dup23,
]));

var msg43 = msg("kernel", part62);

var part63 = match("MESSAGE#43:kernel:01", "nwparser.payload", "%{process}: %{interface->} down: %{result}.", processor_chain([
	dup21,
	dup22,
	setc("event_description","interface down"),
	dup23,
]));

var msg44 = msg("kernel:01", part63);

var part64 = match("MESSAGE#44:kernel:02", "nwparser.payload", "%{process}: %{interface}: loopback suspected; %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","loopback suspected om interface"),
	dup23,
]));

var msg45 = msg("kernel:02", part64);

var part65 = match("MESSAGE#45:kernel:03", "nwparser.payload", "%{process}: %{service}: soreceive() error %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","soreceive error"),
	dup23,
]));

var msg46 = msg("kernel:03", part65);

var part66 = match("MESSAGE#46:kernel:04", "nwparser.payload", "%{process}: %{service->} !VALID(state 4)->%{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","pfe_peer_alloc state 4"),
	dup23,
]));

var msg47 = msg("kernel:04", part66);

var part67 = match("MESSAGE#47:kernel:05", "nwparser.payload", "%{fld1->} %{hostip->} (%{fld2}) %{fld3->} %{process}[%{process_id}]: NTP Server %{result}", processor_chain([
	dup21,
	dup22,
	dup32,
	dup23,
]));

var msg48 = msg("kernel:05", part67);

var part68 = match("MESSAGE#48:kernel:06", "nwparser.payload", "%{fld1->} %{hostip->} %{process}[%{process_id}]: NTP Server %{result}", processor_chain([
	dup21,
	dup22,
	dup32,
	dup23,
]));

var msg49 = msg("kernel:06", part68);

var select17 = linear_select([
	msg41,
	msg42,
	msg43,
	msg44,
	msg45,
	msg46,
	msg47,
	msg48,
	msg49,
]);

var part69 = match("MESSAGE#49:successful_login", "nwparser.payload", "%{process}: login from %{saddr->} on %{interface->} as %{username}", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	setc("event_description","successful user login"),
	dup23,
]));

var msg50 = msg("successful_login", part69);

var part70 = match("MESSAGE#50:login_attempt", "nwparser.payload", "%{process}: Login attempt for user %{username->} from host %{hostip}", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup22,
	setc("event_description","user login attempt"),
	dup23,
]));

var msg51 = msg("login_attempt", part70);

var part71 = match("MESSAGE#51:login", "nwparser.payload", "%{process}: PAM module %{dclass_counter1->} returned: %{space}[%{resultcode}]%{result}", processor_chain([
	dup33,
	dup34,
	dup37,
	dup22,
	setc("event_description","PAM module return from login"),
	dup23,
]));

var msg52 = msg("login", part71);

var select18 = linear_select([
	msg50,
	msg51,
	msg52,
]);

var part72 = match("MESSAGE#52:lsys_ssam_handler", "nwparser.payload", "%{node->} %{process}: processing lsys root-logical-system %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","processing lsys root-logical-system"),
	dup23,
]));

var msg53 = msg("lsys_ssam_handler", part72);

var part73 = match("MESSAGE#53:mcsn", "nwparser.payload", "%{process}[%{process_id}]: Removing mif from group [%{group}] %{space->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Removing mif  from group"),
	dup23,
]));

var msg54 = msg("mcsn", part73);

var part74 = match("MESSAGE#54:mrvl_dfw_log_effuse_status", "nwparser.payload", "%{process}: Firewall rows could not be redirected on device %{device}.", processor_chain([
	dup30,
	dup22,
	setc("event_description","Firewall rows could not be redirected on device"),
	dup23,
]));

var msg55 = msg("mrvl_dfw_log_effuse_status", part74);

var part75 = match("MESSAGE#55:MRVL-L2", "nwparser.payload", "%{process}:%{action}(),%{process_id}:MFilter (%{filter}) already exists", processor_chain([
	dup30,
	dup22,
	setc("event_description","mfilter already exists for add"),
	dup23,
]));

var msg56 = msg("MRVL-L2", part75);

var part76 = match("MESSAGE#56:profile_ssam_handler", "nwparser.payload", "%{node->} %{process}: processing profile SP-root %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","processing profile SP-root"),
	dup23,
]));

var msg57 = msg("profile_ssam_handler", part76);

var part77 = match("MESSAGE#57:pst_nat_binding_set_profile", "nwparser.payload", "%{node->} %{process}: %{event_source}: can't get resource bucket %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","can't get resource bucket"),
	dup23,
]));

var msg58 = msg("pst_nat_binding_set_profile", part77);

var part78 = match("MESSAGE#58:task_reconfigure", "nwparser.payload", "%{process}[%{process_id}]: task_reconfigure %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","reinitializing done"),
	dup23,
]));

var msg59 = msg("task_reconfigure", part78);

var part79 = match("MESSAGE#59:tnetd/0_0", "nwparser.payload", "%{process}[%{process_id}]:%{service}[%{fld1}]: exit status%{resultcode}");

var part80 = match_copy("MESSAGE#59:tnetd/0_1", "nwparser.payload", "fld3");

var select19 = linear_select([
	part79,
	part80,
]);

var all12 = all_match({
	processors: [
		select19,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup23,
		dup24,
	]),
});

var msg60 = msg("tnetd", all12);

var part81 = match("MESSAGE#60:PFEMAN", "nwparser.payload", "%{process}: Session manager active", processor_chain([
	dup21,
	dup22,
	setc("event_description","Session manager active"),
	dup23,
]));

var msg61 = msg("PFEMAN", part81);

var part82 = match("MESSAGE#61:mgd", "nwparser.payload", "%{process}[%{process_id}]: Could not send message to %{service}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Could not send message to service"),
	dup23,
]));

var msg62 = msg("mgd", part82);

var part83 = match("MESSAGE#62:Resolve", "nwparser.payload", "Resolve request came for an address matching on Wrong nh nh:%{result}, %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Resolve request came for an address matching on Wrong nh"),
	dup23,
]));

var msg63 = msg("Resolve", part83);

var part84 = match("MESSAGE#63:respawn", "nwparser.payload", "%{process}: %{service->} exited with status = %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","service exited with status"),
	dup23,
]));

var msg64 = msg("respawn", part84);

var part85 = match("MESSAGE#64:root", "nwparser.payload", "%{process}: %{node}: This system does not have 3-DNS or Link Controller enabled", processor_chain([
	dup30,
	dup22,
	setc("event_description","system does not have 3-DNS or Link Controller enabled"),
	dup23,
]));

var msg65 = msg("root", part85);

var part86 = match("MESSAGE#65:rpd", "nwparser.payload", "%{process}[%{process_id}]: Received %{result->} for intf device %{interface}; mc_ae_id %{dclass_counter1}, status %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Received data for interface"),
	dup23,
]));

var msg66 = msg("rpd", part86);

var part87 = match("MESSAGE#66:rpd:01", "nwparser.payload", "%{process}[%{process_id}]: RSVP neighbor %{daddr->} up on interface %{interface}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RSVP neighbor up on interface "),
	dup23,
]));

var msg67 = msg("rpd:01", part87);

var part88 = match("MESSAGE#67:rpd:02", "nwparser.payload", "%{process}[%{process_id}]: %{saddr->} (%{shost}): reseting pending active connection", processor_chain([
	dup21,
	dup22,
	setc("event_description","reseting pending active connection"),
	dup23,
]));

var msg68 = msg("rpd:02", part88);

var part89 = match("MESSAGE#68:rpd_proceeding", "nwparser.payload", "%{process}: proceeding. %{param}", processor_chain([
	dup21,
	dup22,
	dup38,
	dup23,
]));

var msg69 = msg("rpd_proceeding", part89);

var select20 = linear_select([
	msg66,
	msg67,
	msg68,
	msg69,
]);

var part90 = match("MESSAGE#69:rshd", "nwparser.payload", "%{process}[%{process_id}]: %{username->} as root: cmd='%{action}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","user issuing command as root"),
	dup23,
]));

var msg70 = msg("rshd", part90);

var part91 = match("MESSAGE#70:sfd", "nwparser.payload", "%{process}: Waiting on accept", processor_chain([
	dup21,
	dup22,
	setc("event_description","sfd waiting on accept"),
	dup23,
]));

var msg71 = msg("sfd", part91);

var part92 = match("MESSAGE#71:sshd", "nwparser.payload", "%{process}[%{process_id}]: Accepted password for %{username->} from %{saddr->} port %{sport->} %{protocol}", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	setc("event_description","Accepted password"),
	dup23,
]));

var msg72 = msg("sshd", part92);

var part93 = match("MESSAGE#73:sshd:02", "nwparser.payload", "%{process}[%{process_id}]: Received disconnect from %{shost}: %{fld1}: %{result}", processor_chain([
	dup27,
	dup22,
	setc("event_description","Received disconnect"),
	dup23,
]));

var msg73 = msg("sshd:02", part93);

var part94 = match("MESSAGE#74:sshd:03", "nwparser.payload", "%{process}[%{process_id}]: Did not receive identification string from %{saddr}", processor_chain([
	dup30,
	dup22,
	setc("result","no identification string"),
	setc("event_description","Did not receive identification string from peer"),
	dup23,
]));

var msg74 = msg("sshd:03", part94);

var part95 = match("MESSAGE#75:sshd:04", "nwparser.payload", "%{process}[%{process_id}]: Could not write ident string to %{dhost}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Could not write ident string"),
	dup23,
]));

var msg75 = msg("sshd:04", part95);

var part96 = match("MESSAGE#76:sshd:05", "nwparser.payload", "%{process}[%{process_id}]: subsystem request for netconf", processor_chain([
	dup21,
	dup22,
	setc("event_description","subsystem request for netconf"),
	dup23,
]));

var msg76 = msg("sshd:05", part96);

var part97 = match("MESSAGE#77:sshd:06/2", "nwparser.p0", "sendmsg to %{saddr}(%{shost}).%{sport}: %{info}");

var all13 = all_match({
	processors: [
		dup39,
		dup137,
		part97,
	],
	on_success: processor_chain([
		dup29,
		dup22,
		setc("event_description","send message stats"),
		dup23,
	]),
});

var msg77 = msg("sshd:06", all13);

var part98 = match("MESSAGE#78:sshd:07/2", "nwparser.p0", "Added radius server %{saddr}(%{shost})");

var all14 = all_match({
	processors: [
		dup39,
		dup137,
		part98,
	],
	on_success: processor_chain([
		dup42,
		setc("ec_theme","Configuration"),
		setc("ec_activity","Modify"),
		dup37,
		dup22,
		setc("event_description","Added radius server"),
		dup23,
	]),
});

var msg78 = msg("sshd:07", all14);

var part99 = match("MESSAGE#79:sshd:08", "nwparser.payload", "%{process}[%{process_id}]: %{result}: %{space->} [%{resultcode}]authentication error", processor_chain([
	setc("eventcategory","1301020000"),
	dup34,
	dup43,
	dup22,
	setc("event_description","authentication error"),
	dup23,
]));

var msg79 = msg("sshd:08", part99);

var part100 = match("MESSAGE#80:sshd:09", "nwparser.payload", "%{process}[%{process_id}]: unrecognized attribute in %{policyname}: %{change_attribute}", processor_chain([
	dup30,
	dup22,
	setc("event_description","unrecognized attribute in policy"),
	dup23,
]));

var msg80 = msg("sshd:09", part100);

var part101 = match("MESSAGE#81:sshd:10", "nwparser.payload", "%{process}: PAM module %{dclass_counter1->} returned: %{space}[%{resultcode}]%{result}", processor_chain([
	dup44,
	dup34,
	dup43,
	dup22,
	setc("event_description","PAM module return from sshd"),
	dup23,
]));

var msg81 = msg("sshd:10", part101);

var part102 = match("MESSAGE#82:sshd:11", "nwparser.payload", "%{process}: PAM authentication chain returned: %{space}[%{resultcode}]%{result}", processor_chain([
	dup44,
	dup34,
	dup43,
	dup22,
	setc("event_description","PAM authentication chain return"),
	dup23,
]));

var msg82 = msg("sshd:11", part102);

var part103 = match("MESSAGE#83:sshd:12", "nwparser.payload", "%{process}: %{severity}: can't get client address: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","can't get client address"),
	dup23,
]));

var msg83 = msg("sshd:12", part103);

var part104 = match("MESSAGE#84:sshd:13", "nwparser.payload", "%{process}: auth server unresponsive", processor_chain([
	dup30,
	dup22,
	setc("event_description","auth server unresponsive"),
	dup23,
]));

var msg84 = msg("sshd:13", part104);

var part105 = match("MESSAGE#85:sshd:14", "nwparser.payload", "%{process}: %{service}: No valid RADIUS responses received", processor_chain([
	dup30,
	dup22,
	setc("event_description","No valid RADIUS responses received"),
	dup23,
]));

var msg85 = msg("sshd:14", part105);

var part106 = match("MESSAGE#86:sshd:15", "nwparser.payload", "%{process}: Moving to next server: %{saddr}(%{shost}).%{sport}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Moving to next server"),
	dup23,
]));

var msg86 = msg("sshd:15", part106);

var part107 = match("MESSAGE#87:sshd:16", "nwparser.payload", "%{fld1->} sshd: SSHD_LOGIN_FAILED: Login failed for user '%{username}' from host '%{hostip}'.", processor_chain([
	dup44,
	dup34,
	dup43,
	dup22,
	setc("event_description","Login failed for user"),
	dup23,
]));

var msg87 = msg("sshd:16", part107);

var select21 = linear_select([
	msg72,
	msg73,
	msg74,
	msg75,
	msg76,
	msg77,
	msg78,
	msg79,
	msg80,
	msg81,
	msg82,
	msg83,
	msg84,
	msg85,
	msg86,
	msg87,
]);

var part108 = match("MESSAGE#72:Failed:05/0", "nwparser.payload", "%{process}[%{process_id}]: Failed password for %{p0}");

var part109 = match("MESSAGE#72:Failed:05/1_0", "nwparser.p0", "illegal user %{p0}");

var part110 = match("MESSAGE#72:Failed:05/1_1", "nwparser.p0", "invalid user %{p0}");

var select22 = linear_select([
	part109,
	part110,
	dup45,
]);

var part111 = match("MESSAGE#72:Failed:05/2", "nwparser.p0", "%{username->} from %{saddr->} port %{sport->} %{protocol}");

var all15 = all_match({
	processors: [
		part108,
		select22,
		part111,
	],
	on_success: processor_chain([
		dup44,
		dup34,
		dup35,
		dup36,
		dup43,
		dup22,
		setc("event_description","authentication failure"),
		dup23,
	]),
});

var msg88 = msg("Failed:05", all15);

var part112 = match("MESSAGE#746:Failed/0", "nwparser.payload", "%{hostname->} %{process}[%{process_id}]: Failed to resolve ipv%{p0}");

var part113 = match("MESSAGE#746:Failed/1_0", "nwparser.p0", "4%{p0}");

var part114 = match("MESSAGE#746:Failed/1_1", "nwparser.p0", "6%{p0}");

var select23 = linear_select([
	part113,
	part114,
]);

var part115 = match("MESSAGE#746:Failed/2", "nwparser.p0", "%{}addresses for domain name %{sdomain}");

var all16 = all_match({
	processors: [
		part112,
		select23,
		part115,
	],
	on_success: processor_chain([
		dup46,
		dup47,
		dup23,
		dup22,
	]),
});

var msg89 = msg("Failed", all16);

var part116 = match("MESSAGE#767:Failed:01", "nwparser.payload", "%{hostname->} %{process}[%{process_id}]: %{fld1}", processor_chain([
	dup46,
	dup23,
	dup22,
]));

var msg90 = msg("Failed:01", part116);

var part117 = match("MESSAGE#768:Failed:02/0_0", "nwparser.payload", "%{fld1->} to create a route if table for Multiservice");

var part118 = match_copy("MESSAGE#768:Failed:02/0_1", "nwparser.payload", "fld10");

var select24 = linear_select([
	part117,
	part118,
]);

var all17 = all_match({
	processors: [
		select24,
	],
	on_success: processor_chain([
		dup46,
		dup23,
		dup22,
		setf("hostname","hfld1"),
	]),
});

var msg91 = msg("Failed:02", all17);

var select25 = linear_select([
	msg88,
	msg89,
	msg90,
	msg91,
]);

var part119 = match("MESSAGE#88:syslogd", "nwparser.payload", "%{process}: restart", processor_chain([
	dup21,
	dup22,
	setc("event_description","syslog daemon restart"),
	dup23,
]));

var msg92 = msg("syslogd", part119);

var part120 = match("MESSAGE#89:ucd-snmp", "nwparser.payload", "%{process}[%{process_id}]: AUDIT -- Action %{action->} User: %{username}", processor_chain([
	dup21,
	dup22,
	dup25,
	dup23,
]));

var msg93 = msg("ucd-snmp", part120);

var part121 = match("MESSAGE#90:ucd-snmp:01", "nwparser.payload", "%{process}[%{process_id}]: Received TERM or STOP signal %{space->} %{result}.", processor_chain([
	dup21,
	dup22,
	setc("event_description","Received TERM or STOP signal"),
	dup23,
]));

var msg94 = msg("ucd-snmp:01", part121);

var select26 = linear_select([
	msg93,
	msg94,
]);

var part122 = match("MESSAGE#91:usp_ipc_client_reconnect", "nwparser.payload", "%{node->} %{process}: failed to connect to the server: %{result->} (%{resultcode})", processor_chain([
	dup27,
	dup22,
	setc("event_description","failed to connect to the server"),
	dup23,
]));

var msg95 = msg("usp_ipc_client_reconnect", part122);

var part123 = match("MESSAGE#92:usp_trace_ipc_disconnect", "nwparser.payload", "%{node->} %{process}:Trace client disconnected. %{result}", processor_chain([
	dup27,
	dup22,
	setc("event_description","Trace client disconnected"),
	dup23,
]));

var msg96 = msg("usp_trace_ipc_disconnect", part123);

var part124 = match("MESSAGE#93:usp_trace_ipc_reconnect", "nwparser.payload", "%{node->} %{process}:USP trace client cannot reconnect to server", processor_chain([
	dup30,
	dup22,
	setc("event_description","USP trace client cannot reconnect to server"),
	dup23,
]));

var msg97 = msg("usp_trace_ipc_reconnect", part124);

var part125 = match("MESSAGE#94:uspinfo", "nwparser.payload", "%{process}: flow_print_session_summary_output received %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","flow_print_session_summary_output received"),
	dup23,
]));

var msg98 = msg("uspinfo", part125);

var part126 = match("MESSAGE#95:Version", "nwparser.payload", "Version %{version->} by builder on %{event_time_string}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Version build date"),
	dup23,
]));

var msg99 = msg("Version", part126);

var part127 = match("MESSAGE#96:xntpd", "nwparser.payload", "%{process}[%{process_id}]: frequency initialized %{result->} from %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","frequency initialized from file"),
	dup23,
]));

var msg100 = msg("xntpd", part127);

var part128 = match("MESSAGE#97:xntpd:01", "nwparser.payload", "%{process}[%{process_id}]: ntpd %{version->} %{event_time_string->} (%{resultcode})", processor_chain([
	dup21,
	dup22,
	setc("event_description","nptd version build"),
	dup23,
]));

var msg101 = msg("xntpd:01", part128);

var part129 = match("MESSAGE#98:xntpd:02", "nwparser.payload", "%{process}: kernel time sync enabled %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","kernel time sync enabled"),
	dup23,
]));

var msg102 = msg("xntpd:02", part129);

var part130 = match("MESSAGE#99:xntpd:03", "nwparser.payload", "%{process}[%{process_id}]: NTP Server %{result}", processor_chain([
	dup21,
	dup22,
	dup32,
	dup23,
]));

var msg103 = msg("xntpd:03", part130);

var select27 = linear_select([
	msg100,
	msg101,
	msg102,
	msg103,
]);

var part131 = match("MESSAGE#100:last", "nwparser.payload", "last message repeated %{dclass_counter1->} times", processor_chain([
	dup21,
	dup22,
	setc("event_description","last message repeated"),
	dup23,
]));

var msg104 = msg("last", part131);

var part132 = match("MESSAGE#739:last:01", "nwparser.payload", "message repeated %{dclass_counter1->} times", processor_chain([
	dup48,
	dup47,
	dup23,
	dup22,
	dup24,
]));

var msg105 = msg("last:01", part132);

var select28 = linear_select([
	msg104,
	msg105,
]);

var part133 = match("MESSAGE#101:BCHIP", "nwparser.payload", "%{process->} %{device}: cannot write ucode mask reg", processor_chain([
	dup30,
	dup22,
	setc("event_description","cannot write ucode mask reg"),
	dup23,
]));

var msg106 = msg("BCHIP", part133);

var part134 = match("MESSAGE#102:CM", "nwparser.payload", "%{process}(%{fld1}): Slot %{device}: On-line", processor_chain([
	dup21,
	dup22,
	setc("event_description","Slot on-line"),
	dup23,
]));

var msg107 = msg("CM", part134);

var part135 = match("MESSAGE#103:COS", "nwparser.payload", "%{process}: Received FC->Q map, %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Received FC Q map"),
	dup23,
]));

var msg108 = msg("COS", part135);

var part136 = match("MESSAGE#104:COSFPC", "nwparser.payload", "%{process}: ifd %{resultcode}: %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","ifd error"),
	dup23,
]));

var msg109 = msg("COSFPC", part136);

var part137 = match("MESSAGE#105:COSMAN", "nwparser.payload", "%{process}: %{service}: delete class_to_ifl table %{dclass_counter1}, ifl %{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","delete class to ifl link"),
	dup23,
]));

var msg110 = msg("COSMAN", part137);

var part138 = match("MESSAGE#106:RDP", "nwparser.payload", "%{process}: Keepalive timeout for rdp.(%{interface}).(%{device}) (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","Keepalive timeout"),
	dup23,
]));

var msg111 = msg("RDP", part138);

var part139 = match("MESSAGE#107:SNTPD", "nwparser.payload", "%{process}: Initial time of day set", processor_chain([
	dup30,
	dup22,
	setc("event_description","Initial time of day set"),
	dup23,
]));

var msg112 = msg("SNTPD", part139);

var part140 = match("MESSAGE#108:SSB", "nwparser.payload", "%{process}(%{fld1}): Slot %{device}, serial number S/N %{serial_number}.", processor_chain([
	dup21,
	dup22,
	setc("event_description","Slot serial number"),
	dup23,
]));

var msg113 = msg("SSB", part140);

var part141 = match("MESSAGE#109:ACCT_ACCOUNTING_FERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unexpected error %{result->} from file %{filename}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unexpected error"),
	dup23,
]));

var msg114 = msg("ACCT_ACCOUNTING_FERROR", part141);

var part142 = match("MESSAGE#110:ACCT_ACCOUNTING_FOPEN_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to open file %{filename}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Failed to open file"),
	dup23,
]));

var msg115 = msg("ACCT_ACCOUNTING_FOPEN_ERROR", part142);

var part143 = match("MESSAGE#111:ACCT_ACCOUNTING_SMALL_FILE_SIZE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: File %{filename->} size (%{dclass_counter1}) is smaller than record size (%{dclass_counter2})", processor_chain([
	dup49,
	dup22,
	setc("event_description","File size mismatch"),
	dup23,
]));

var msg116 = msg("ACCT_ACCOUNTING_SMALL_FILE_SIZE", part143);

var part144 = match("MESSAGE#112:ACCT_BAD_RECORD_FORMAT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Invalid statistics record: %{result}", processor_chain([
	dup49,
	dup22,
	setc("event_description","Invalid statistics record"),
	dup23,
]));

var msg117 = msg("ACCT_BAD_RECORD_FORMAT", part144);

var part145 = match("MESSAGE#113:ACCT_CU_RTSLIB_error", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{filename->} getting class usage statistics for interface %{interface}: %{result}", processor_chain([
	dup49,
	dup22,
	setc("event_description","Class usage statistics error for interface"),
	dup23,
]));

var msg118 = msg("ACCT_CU_RTSLIB_error", part145);

var part146 = match("MESSAGE#114:ACCT_GETHOSTNAME_error/1_0", "nwparser.p0", "Error %{resultcode->} trying %{p0}");

var part147 = match("MESSAGE#114:ACCT_GETHOSTNAME_error/1_1", "nwparser.p0", "trying %{p0}");

var select29 = linear_select([
	part146,
	part147,
]);

var part148 = match("MESSAGE#114:ACCT_GETHOSTNAME_error/2", "nwparser.p0", "to get hostname%{}");

var all18 = all_match({
	processors: [
		dup50,
		select29,
		part148,
	],
	on_success: processor_chain([
		dup49,
		dup22,
		setc("event_description","error trying to get hostname"),
		dup23,
	]),
});

var msg119 = msg("ACCT_GETHOSTNAME_error", all18);

var part149 = match("MESSAGE#115:ACCT_MALLOC_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Memory allocation failed while reallocating %{obj_name}", processor_chain([
	dup51,
	dup22,
	setc("event_description","Memory allocation failure"),
	dup23,
]));

var msg120 = msg("ACCT_MALLOC_FAILURE", part149);

var part150 = match("MESSAGE#116:ACCT_UNDEFINED_COUNTER_NAME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{filename->} in accounting profile %{dclass_counter1->} is not defined in a firewall using this filter profile", processor_chain([
	dup30,
	dup22,
	setc("event_description","Accounting profile counter not defined in firewall"),
	dup23,
]));

var msg121 = msg("ACCT_UNDEFINED_COUNTER_NAME", part150);

var part151 = match("MESSAGE#117:ACCT_XFER_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type->} %{result}: %{disposition}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ACCT_XFER_FAILED"),
	dup23,
]));

var msg122 = msg("ACCT_XFER_FAILED", part151);

var part152 = match("MESSAGE#118:ACCT_XFER_POPEN_FAIL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type->} %{result}: in invoking command command to transfer file %{filename}", processor_chain([
	dup30,
	dup22,
	setc("event_description","POPEN FAIL invoking command command to transfer file"),
	dup23,
]));

var msg123 = msg("ACCT_XFER_POPEN_FAIL", part152);

var part153 = match("MESSAGE#119:APPQOS_LOG_EVENT", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} timestamp=\"%{result}\" message-type=\"%{info}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-name=\"%{protocol}\" application-name=\"%{application}\" rule-set-name=\"%{rule_group}\" rule-name=\"%{rulename}\" action=\"%{action}\" argument=\"%{fld2}\" argument1=\"%{fld3}\"]", processor_chain([
	dup28,
	dup22,
	dup52,
]));

var msg124 = msg("APPQOS_LOG_EVENT", part153);

var part154 = match("MESSAGE#120:APPTRACK_SESSION_CREATE", "nwparser.payload", "%{event_type}: AppTrack session created %{saddr}/%{sport}->%{daddr}/%{dport->} %{service->} %{protocol->} %{fld11->} %{hostip}/%{network_port}->%{dtransaddr}/%{dtransport->} %{rulename->} %{rule_template->} %{fld12->} %{policyname->} %{src_zone->} %{dst_zone->} %{sessionid->} %{username->} %{fld10}", processor_chain([
	dup28,
	dup53,
	dup54,
	dup22,
	setc("result","AppTrack session created"),
	dup23,
]));

var msg125 = msg("APPTRACK_SESSION_CREATE", part154);

var part155 = match("MESSAGE#121:APPTRACK_SESSION_CLOSE", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} reason=\"%{result}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" service-name=\"%{service}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" src-nat-rule-name=\"%{rulename}\" dst-nat-rule-name=\"%{rule_template}\" protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{sessionid}\" packets-from-client=\"%{packets}\" bytes-from-client=\"%{rbytes}\" packets-from-server=\"%{dclass_counter1}\" bytes-from-server=\"%{sbytes}\" elapsed-time=\"%{duration}\"]", processor_chain([
	dup28,
	dup53,
	dup55,
	dup22,
	dup52,
]));

var msg126 = msg("APPTRACK_SESSION_CLOSE", part155);

var part156 = match("MESSAGE#122:APPTRACK_SESSION_CLOSE:01", "nwparser.payload", "%{event_type}: %{result}: %{saddr}/%{sport}->%{daddr}/%{dport->} %{service->} %{protocol->} %{fld11->} %{hostip}/%{network_port}->%{dtransaddr}/%{dtransport->} %{rulename->} %{rule_template->} %{fld12->} %{policyname->} %{src_zone->} %{dst_zone->} %{sessionid->} %{packets}(%{rbytes}) %{dclass_counter1}(%{sbytes}) %{duration->} %{username->} %{fld10}", processor_chain([
	dup28,
	dup53,
	dup55,
	dup22,
	dup23,
]));

var msg127 = msg("APPTRACK_SESSION_CLOSE:01", part156);

var select30 = linear_select([
	msg126,
	msg127,
]);

var part157 = match("MESSAGE#123:APPTRACK_SESSION_VOL_UPDATE", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" service-name=\"%{service}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" src-nat-rule-name=\"%{rulename}\" dst-nat-rule-name=\"%{rule_template}\" protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{sessionid}\" packets-from-client=\"%{packets}\" bytes-from-client=\"%{rbytes}\" packets-from-server=\"%{dclass_counter1}\" bytes-from-server=\"%{sbytes}\" elapsed-time=\"%{duration}\"]", processor_chain([
	dup28,
	dup53,
	dup22,
	dup52,
]));

var msg128 = msg("APPTRACK_SESSION_VOL_UPDATE", part157);

var part158 = match("MESSAGE#124:APPTRACK_SESSION_VOL_UPDATE:01", "nwparser.payload", "%{event_type}: %{result}: %{saddr}/%{sport}->%{daddr}/%{dport->} %{service->} %{protocol->} %{fld11->} %{hostip}/%{network_port}->%{dtransaddr}/%{dtransport->} %{rulename->} %{rule_template->} %{fld12->} %{policyname->} %{src_zone->} %{dst_zone->} %{sessionid->} %{packets}(%{rbytes}) %{dclass_counter1}(%{sbytes}) %{duration->} %{username->} %{fld10}", processor_chain([
	dup28,
	dup53,
	dup22,
	dup23,
]));

var msg129 = msg("APPTRACK_SESSION_VOL_UPDATE:01", part158);

var select31 = linear_select([
	msg128,
	msg129,
]);

var msg130 = msg("BFDD_TRAP_STATE_DOWN", dup138);

var msg131 = msg("BFDD_TRAP_STATE_UP", dup138);

var part159 = match("MESSAGE#127:bgp_connect_start", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: connect %{saddr->} (%{shost}): %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","bgp connect error"),
	dup23,
]));

var msg132 = msg("bgp_connect_start", part159);

var part160 = match("MESSAGE#128:bgp_event", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: peer %{daddr->} (%{dhost}) old state %{change_old->} event %{action->} new state %{change_new}", processor_chain([
	dup21,
	dup22,
	setc("event_description","bgp peer state change"),
	dup23,
]));

var msg133 = msg("bgp_event", part160);

var part161 = match("MESSAGE#129:bgp_listen_accept", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Connection attempt from unconfigured neighbor: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Connection attempt from unconfigured neighbor"),
	dup23,
]));

var msg134 = msg("bgp_listen_accept", part161);

var part162 = match("MESSAGE#130:bgp_listen_reset", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","bgp reset"),
	dup23,
]));

var msg135 = msg("bgp_listen_reset", part162);

var part163 = match("MESSAGE#131:bgp_nexthop_sanity", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: peer %{daddr->} (%{dhost}) next hop %{saddr->} local, %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","peer next hop local"),
	dup23,
]));

var msg136 = msg("bgp_nexthop_sanity", part163);

var part164 = match("MESSAGE#132:bgp_process_caps", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: NOTIFICATION sent to %{daddr->} (%{dhost}): code %{severity->} (%{action}) subcode %{version->} (%{result}) value %{disposition}", processor_chain([
	dup30,
	dup22,
	setc("event_description","code RED error NOTIFICATION sent"),
	dup23,
]));

var msg137 = msg("bgp_process_caps", part164);

var part165 = match("MESSAGE#133:bgp_process_caps:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: mismatch NLRI with %{hostip->} (%{hostname}): peer: %{daddr->} us: %{saddr}", processor_chain([
	dup30,
	dup22,
	dup57,
	dup23,
]));

var msg138 = msg("bgp_process_caps:01", part165);

var select32 = linear_select([
	msg137,
	msg138,
]);

var part166 = match("MESSAGE#134:bgp_pp_recv", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: dropping %{daddr->} (%{dhost}), %{info->} (%{protocol})", processor_chain([
	dup30,
	dup22,
	setc("event_description","connection collision"),
	setc("result","dropping connection to peer"),
	dup23,
]));

var msg139 = msg("bgp_pp_recv", part166);

var part167 = match("MESSAGE#135:bgp_pp_recv:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: peer %{daddr->} (%{dhost}): received unexpected EOF", processor_chain([
	dup30,
	dup22,
	setc("event_description","peer received unexpected EOF"),
	dup23,
]));

var msg140 = msg("bgp_pp_recv:01", part167);

var select33 = linear_select([
	msg139,
	msg140,
]);

var part168 = match("MESSAGE#136:bgp_send", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: sending %{sbytes->} bytes to %{daddr->} (%{dhost}) blocked (%{disposition}): %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","bgp send blocked error"),
	dup23,
]));

var msg141 = msg("bgp_send", part168);

var part169 = match("MESSAGE#137:bgp_traffic_timeout", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: NOTIFICATION sent to %{daddr->} (%{dhost}): code %{resultcode->} (%{action}), Reason: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","bgp timeout NOTIFICATION sent"),
	dup23,
]));

var msg142 = msg("bgp_traffic_timeout", part169);

var part170 = match("MESSAGE#138:BOOTPD_ARG_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Ignoring unknown option %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","boot argument error"),
	dup23,
]));

var msg143 = msg("BOOTPD_ARG_ERR", part170);

var part171 = match("MESSAGE#139:BOOTPD_BAD_ID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unexpected ID %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","boot unexpected Id value"),
	dup23,
]));

var msg144 = msg("BOOTPD_BAD_ID", part171);

var part172 = match("MESSAGE#140:BOOTPD_BOOTSTRING", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Boot string: %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Invalid boot string"),
	dup23,
]));

var msg145 = msg("BOOTPD_BOOTSTRING", part172);

var part173 = match("MESSAGE#141:BOOTPD_CONFIG_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Problems with configuration file '%{filename}', %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","configuration file error"),
	dup23,
]));

var msg146 = msg("BOOTPD_CONFIG_ERR", part173);

var part174 = match("MESSAGE#142:BOOTPD_CONF_OPEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to open configuration file '%{filename}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to open configuration file"),
	dup23,
]));

var msg147 = msg("BOOTPD_CONF_OPEN", part174);

var part175 = match("MESSAGE#143:BOOTPD_DUP_REV", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Duplicate revision: %{version}", processor_chain([
	dup30,
	dup22,
	setc("event_description","boot - Duplicate revision"),
	dup23,
]));

var msg148 = msg("BOOTPD_DUP_REV", part175);

var part176 = match("MESSAGE#144:BOOTPD_DUP_SLOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Duplicate slot default: %{ssid}", processor_chain([
	dup30,
	dup22,
	setc("event_description","boot - duplicate slot"),
	dup23,
]));

var msg149 = msg("BOOTPD_DUP_SLOT", part176);

var part177 = match("MESSAGE#145:BOOTPD_MODEL_CHK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unexpected ID %{id->} for model %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unexpected ID for model"),
	dup23,
]));

var msg150 = msg("BOOTPD_MODEL_CHK", part177);

var part178 = match("MESSAGE#146:BOOTPD_MODEL_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unsupported model %{dclass_counter1}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unsupported model"),
	dup23,
]));

var msg151 = msg("BOOTPD_MODEL_ERR", part178);

var part179 = match("MESSAGE#147:BOOTPD_NEW_CONF", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: New configuration installed", processor_chain([
	dup21,
	dup22,
	setc("event_description","New configuration installed"),
	dup23,
]));

var msg152 = msg("BOOTPD_NEW_CONF", part179);

var part180 = match("MESSAGE#148:BOOTPD_NO_BOOTSTRING", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No boot string found for type %{filename}", processor_chain([
	dup30,
	dup22,
	setc("event_description","No boot string found"),
	dup23,
]));

var msg153 = msg("BOOTPD_NO_BOOTSTRING", part180);

var part181 = match("MESSAGE#149:BOOTPD_NO_CONFIG", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No configuration file '%{filename}', %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","No configuration file found"),
	dup23,
]));

var msg154 = msg("BOOTPD_NO_CONFIG", part181);

var part182 = match("MESSAGE#150:BOOTPD_PARSE_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{filename}: number parse errors on SIGHUP", processor_chain([
	dup30,
	dup22,
	setc("event_description","parse errors on SIGHUP"),
	dup23,
]));

var msg155 = msg("BOOTPD_PARSE_ERR", part182);

var part183 = match("MESSAGE#151:BOOTPD_REPARSE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reparsing configuration file '%{filename}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","Reparsing configuration file"),
	dup23,
]));

var msg156 = msg("BOOTPD_REPARSE", part183);

var part184 = match("MESSAGE#152:BOOTPD_SELECT_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: select: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","select error"),
	dup23,
]));

var msg157 = msg("BOOTPD_SELECT_ERR", part184);

var part185 = match("MESSAGE#153:BOOTPD_TIMEOUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Timeout %{result->} unreasonable", processor_chain([
	dup30,
	dup22,
	setc("event_description","timeout unreasonable"),
	dup23,
]));

var msg158 = msg("BOOTPD_TIMEOUT", part185);

var part186 = match("MESSAGE#154:BOOTPD_VERSION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Version: %{version->} built by builder on %{event_time_string}", processor_chain([
	dup21,
	dup22,
	setc("event_description","boot version built"),
	dup23,
]));

var msg159 = msg("BOOTPD_VERSION", part186);

var part187 = match("MESSAGE#155:CHASSISD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type->} %{version->} built by builder on %{event_time_string}", processor_chain([
	dup58,
	dup22,
	setc("event_description","CHASSISD release built"),
	dup23,
]));

var msg160 = msg("CHASSISD", part187);

var part188 = match("MESSAGE#156:CHASSISD_ARGUMENT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unknown option %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD Unknown option"),
	dup23,
]));

var msg161 = msg("CHASSISD_ARGUMENT_ERROR", part188);

var part189 = match("MESSAGE#157:CHASSISD_BLOWERS_SPEED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Fans and impellers are now running at normal speed", processor_chain([
	dup21,
	dup22,
	setc("event_description","Fans and impellers are now running at normal speed"),
	dup23,
]));

var msg162 = msg("CHASSISD_BLOWERS_SPEED", part189);

var part190 = match("MESSAGE#158:CHASSISD_BLOWERS_SPEED_FULL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Fans and impellers being set to full speed [%{result}]", processor_chain([
	dup21,
	dup22,
	setc("event_description","Fans and impellers being set to full speed"),
	dup23,
]));

var msg163 = msg("CHASSISD_BLOWERS_SPEED_FULL", part190);

var part191 = match("MESSAGE#159:CHASSISD_CB_READ", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result->} reading midplane ID EEPROM, %{dclass_counter1->} %{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","reading midplane ID EEPROM"),
	dup23,
]));

var msg164 = msg("CHASSISD_CB_READ", part191);

var part192 = match("MESSAGE#160:CHASSISD_COMMAND_ACK_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{device->} online ack code %{dclass_counter1->} - - %{result}, %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD COMMAND ACK ERROR"),
	dup23,
]));

var msg165 = msg("CHASSISD_COMMAND_ACK_ERROR", part192);

var part193 = match("MESSAGE#161:CHASSISD_COMMAND_ACK_SF_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{disposition->} - %{result}, code %{resultcode}, SFM %{dclass_counter1}, FPC %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD COMMAND ACK SF ERROR"),
	dup23,
]));

var msg166 = msg("CHASSISD_COMMAND_ACK_SF_ERROR", part193);

var part194 = match("MESSAGE#162:CHASSISD_CONCAT_MODE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Cannot set no-concatenated mode for FPC %{dclass_counter2->} PIC %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Cannot set no-concatenated mode for FPC"),
	dup23,
]));

var msg167 = msg("CHASSISD_CONCAT_MODE_ERROR", part194);

var part195 = match("MESSAGE#163:CHASSISD_CONFIG_INIT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Problems with configuration file %{filename}; %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CONFIG File Problem"),
	dup23,
]));

var msg168 = msg("CHASSISD_CONFIG_INIT_ERROR", part195);

var part196 = match("MESSAGE#164:CHASSISD_CONFIG_WARNING", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{filename}: %{result}, FPC %{dclass_counter2->} %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD CONFIG WARNING"),
	dup23,
]));

var msg169 = msg("CHASSISD_CONFIG_WARNING", part196);

var part197 = match("MESSAGE#165:CHASSISD_EXISTS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: chassisd already running; %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","chassisd already running"),
	dup23,
]));

var msg170 = msg("CHASSISD_EXISTS", part197);

var part198 = match("MESSAGE#166:CHASSISD_EXISTS_TERM_OTHER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Killing existing chassisd and exiting", processor_chain([
	dup21,
	dup22,
	setc("event_description","Killing existing chassisd and exiting"),
	dup23,
]));

var msg171 = msg("CHASSISD_EXISTS_TERM_OTHER", part198);

var part199 = match("MESSAGE#167:CHASSISD_FILE_OPEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: File open: %{filename}, error: %{resultcode->} - - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","file open error"),
	dup23,
]));

var msg172 = msg("CHASSISD_FILE_OPEN", part199);

var part200 = match("MESSAGE#168:CHASSISD_FILE_STAT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: File stat: %{filename}, error: %{resultcode->} - - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD file statistics error"),
	dup23,
]));

var msg173 = msg("CHASSISD_FILE_STAT", part200);

var part201 = match("MESSAGE#169:CHASSISD_FRU_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD received restart EVENT"),
	dup23,
]));

var msg174 = msg("CHASSISD_FRU_EVENT", part201);

var part202 = match("MESSAGE#170:CHASSISD_FRU_IPC_WRITE_ERROR_EXT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} FRU %{filename}#%{resultcode}, %{result->} %{dclass_counter1}, %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD restart WRITE_ERROR"),
	dup23,
]));

var msg175 = msg("CHASSISD_FRU_IPC_WRITE_ERROR_EXT", part202);

var part203 = match("MESSAGE#171:CHASSISD_FRU_STEP_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{filename->} %{resultcode->} at step %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD FRU STEP ERROR"),
	dup23,
]));

var msg176 = msg("CHASSISD_FRU_STEP_ERROR", part203);

var part204 = match("MESSAGE#172:CHASSISD_GETTIMEOFDAY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unexpected error from gettimeofday: %{resultcode->} - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unexpected error from gettimeofday"),
	dup23,
]));

var msg177 = msg("CHASSISD_GETTIMEOFDAY", part204);

var part205 = match("MESSAGE#173:CHASSISD_HOST_TEMP_READ", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result->} reading host temperature sensor", processor_chain([
	dup21,
	dup22,
	setc("event_description","reading host temperature sensor"),
	dup23,
]));

var msg178 = msg("CHASSISD_HOST_TEMP_READ", part205);

var part206 = match("MESSAGE#174:CHASSISD_IFDEV_DETACH_ALL_PSEUDO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}(%{disposition})", processor_chain([
	dup21,
	dup22,
	setc("event_description","detaching all pseudo devices"),
	dup23,
]));

var msg179 = msg("CHASSISD_IFDEV_DETACH_ALL_PSEUDO", part206);

var part207 = match("MESSAGE#175:CHASSISD_IFDEV_DETACH_FPC", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}(%{resultcode})", processor_chain([
	dup21,
	dup22,
	setc("event_description","CHASSISD IFDEV DETACH FPC"),
	dup23,
]));

var msg180 = msg("CHASSISD_IFDEV_DETACH_FPC", part207);

var part208 = match("MESSAGE#176:CHASSISD_IFDEV_DETACH_PIC", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}(%{resultcode})", processor_chain([
	dup21,
	dup22,
	setc("event_description","CHASSISD IFDEV DETACH PIC"),
	dup23,
]));

var msg181 = msg("CHASSISD_IFDEV_DETACH_PIC", part208);

var part209 = match("MESSAGE#177:CHASSISD_IFDEV_DETACH_PSEUDO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}(%{disposition})", processor_chain([
	dup21,
	dup22,
	setc("event_description","CHASSISD IFDEV DETACH PSEUDO"),
	dup23,
]));

var msg182 = msg("CHASSISD_IFDEV_DETACH_PSEUDO", part209);

var part210 = match("MESSAGE#178:CHASSISD_IFDEV_DETACH_TLV_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD IFDEV DETACH TLV ERROR"),
	dup23,
]));

var msg183 = msg("CHASSISD_IFDEV_DETACH_TLV_ERROR", part210);

var part211 = match("MESSAGE#179:CHASSISD_IFDEV_GET_BY_INDEX_FAIL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: rtslib_ifdm_get_by_index failed: %{resultcode->} - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","rtslib_ifdm_get_by_index failed"),
	dup23,
]));

var msg184 = msg("CHASSISD_IFDEV_GET_BY_INDEX_FAIL", part211);

var part212 = match("MESSAGE#180:CHASSISD_IPC_MSG_QFULL_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}: type = %{dclass_counter1}, subtype = %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Message Queue full"),
	dup23,
]));

var msg185 = msg("CHASSISD_IPC_MSG_QFULL_ERROR", part212);

var part213 = match("MESSAGE#181:CHASSISD_IPC_UNEXPECTED_RECV", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Received unexpected message from %{service}: type = %{dclass_counter1}, subtype = %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Received unexpected message"),
	dup23,
]));

var msg186 = msg("CHASSISD_IPC_UNEXPECTED_RECV", part213);

var part214 = match("MESSAGE#182:CHASSISD_IPC_WRITE_ERR_NO_PIPE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: FRU has no connection pipe %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","FRU has no connection pipe"),
	dup23,
]));

var msg187 = msg("CHASSISD_IPC_WRITE_ERR_NO_PIPE", part214);

var part215 = match("MESSAGE#183:CHASSISD_IPC_WRITE_ERR_NULL_ARGS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: FRU has no connection arguments %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","FRU has no connection arguments"),
	dup23,
]));

var msg188 = msg("CHASSISD_IPC_WRITE_ERR_NULL_ARGS", part215);

var part216 = match("MESSAGE#184:CHASSISD_MAC_ADDRESS_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: chassisd MAC address allocation error", processor_chain([
	dup30,
	dup22,
	setc("event_description","chassisd MAC address allocation error"),
	dup23,
]));

var msg189 = msg("CHASSISD_MAC_ADDRESS_ERROR", part216);

var part217 = match("MESSAGE#185:CHASSISD_MAC_DEFAULT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Using default MAC address base", processor_chain([
	dup21,
	dup22,
	setc("event_description","Using default MAC address base"),
	dup23,
]));

var msg190 = msg("CHASSISD_MAC_DEFAULT", part217);

var part218 = match("MESSAGE#186:CHASSISD_MBUS_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} %{resultcode}: management bus failed sanity test", processor_chain([
	dup30,
	dup22,
	setc("event_description","management bus failed sanity test"),
	dup23,
]));

var msg191 = msg("CHASSISD_MBUS_ERROR", part218);

var part219 = match("MESSAGE#187:CHASSISD_PARSE_COMPLETE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Using new configuration", processor_chain([
	dup21,
	dup22,
	setc("event_description","Using new configuration"),
	dup23,
]));

var msg192 = msg("CHASSISD_PARSE_COMPLETE", part219);

var part220 = match("MESSAGE#188:CHASSISD_PARSE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{resultcode->} %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHASSISD PARSE ERROR"),
	dup23,
]));

var msg193 = msg("CHASSISD_PARSE_ERROR", part220);

var part221 = match("MESSAGE#189:CHASSISD_PARSE_INIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Parsing configuration file '%{filename}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","Parsing configuration file"),
	dup23,
]));

var msg194 = msg("CHASSISD_PARSE_INIT", part221);

var part222 = match("MESSAGE#190:CHASSISD_PIDFILE_OPEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to open PID file '%{filename}': %{result->} %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to open PID file"),
	dup23,
]));

var msg195 = msg("CHASSISD_PIDFILE_OPEN", part222);

var part223 = match("MESSAGE#191:CHASSISD_PIPE_WRITE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Pipe error: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Pipe error"),
	dup23,
]));

var msg196 = msg("CHASSISD_PIPE_WRITE_ERROR", part223);

var part224 = match("MESSAGE#192:CHASSISD_POWER_CHECK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{device->} %{dclass_counter1->} not powering up", processor_chain([
	dup59,
	dup22,
	setc("event_description","device not powering up"),
	dup23,
]));

var msg197 = msg("CHASSISD_POWER_CHECK", part224);

var part225 = match("MESSAGE#193:CHASSISD_RECONNECT_SUCCESSFUL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Successfully reconnected on soft restart", processor_chain([
	dup21,
	dup22,
	setc("event_description","Successful reconnect on soft restart"),
	dup23,
]));

var msg198 = msg("CHASSISD_RECONNECT_SUCCESSFUL", part225);

var part226 = match("MESSAGE#194:CHASSISD_RELEASE_MASTERSHIP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Release mastership notification", processor_chain([
	dup21,
	dup22,
	setc("event_description","Release mastership notification"),
	dup23,
]));

var msg199 = msg("CHASSISD_RELEASE_MASTERSHIP", part226);

var part227 = match("MESSAGE#195:CHASSISD_RE_INIT_INVALID_RE_SLOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: re_init: re %{resultcode}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","re_init Invalid RE slot"),
	dup23,
]));

var msg200 = msg("CHASSISD_RE_INIT_INVALID_RE_SLOT", part227);

var part228 = match("MESSAGE#196:CHASSISD_ROOT_MOUNT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to determine the mount point for root directory: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to determine mount point for root directory"),
	dup23,
]));

var msg201 = msg("CHASSISD_ROOT_MOUNT_ERROR", part228);

var part229 = match("MESSAGE#197:CHASSISD_RTS_SEQ_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifmsg sequence gap %{resultcode->} - - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ifmsg sequence gap"),
	dup23,
]));

var msg202 = msg("CHASSISD_RTS_SEQ_ERROR", part229);

var part230 = match("MESSAGE#198:CHASSISD_SBOARD_VERSION_MISMATCH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Version mismatch: %{info}", processor_chain([
	setc("eventcategory","1603040000"),
	dup22,
	setc("event_description","Version mismatch"),
	dup23,
]));

var msg203 = msg("CHASSISD_SBOARD_VERSION_MISMATCH", part230);

var part231 = match("MESSAGE#199:CHASSISD_SERIAL_ID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Serial ID read error: %{resultcode->} - - %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Serial ID read error"),
	dup23,
]));

var msg204 = msg("CHASSISD_SERIAL_ID", part231);

var part232 = match("MESSAGE#200:CHASSISD_SMB_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: fpga download not complete: val %{resultcode}, %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","fpga download not complete"),
	dup23,
]));

var msg205 = msg("CHASSISD_SMB_ERROR", part232);

var part233 = match("MESSAGE#201:CHASSISD_SNMP_TRAP6", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: SNMP trap generated: %{result->} (%{info})", processor_chain([
	dup58,
	dup22,
	setc("event_description","SNMP Trap6 generated"),
	dup23,
]));

var msg206 = msg("CHASSISD_SNMP_TRAP6", part233);

var part234 = match("MESSAGE#202:CHASSISD_SNMP_TRAP7", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: SNMP trap: %{result}: %{info}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP Trap7 generated"),
	dup23,
]));

var msg207 = msg("CHASSISD_SNMP_TRAP7", part234);

var part235 = match("MESSAGE#203:CHASSISD_SNMP_TRAP10", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: SNMP trap: %{result}: %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP trap - FRU power on"),
	dup23,
]));

var msg208 = msg("CHASSISD_SNMP_TRAP10", part235);

var part236 = match("MESSAGE#204:CHASSISD_TERM_SIGNAL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Received SIGTERM request, %{result}", processor_chain([
	dup60,
	dup22,
	setc("event_description","Received SIGTERM request"),
	dup23,
]));

var msg209 = msg("CHASSISD_TERM_SIGNAL", part236);

var part237 = match("MESSAGE#205:CHASSISD_TRACE_PIC_OFFLINE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Taking PIC offline - - FPC slot %{dclass_counter1}, PIC slot %{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Taking PIC offline"),
	dup23,
]));

var msg210 = msg("CHASSISD_TRACE_PIC_OFFLINE", part237);

var part238 = match("MESSAGE#206:CHASSISD_UNEXPECTED_EXIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} returned %{resultcode}: %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","UNEXPECTED EXIT"),
	dup23,
]));

var msg211 = msg("CHASSISD_UNEXPECTED_EXIT", part238);

var part239 = match("MESSAGE#207:CHASSISD_UNSUPPORTED_MODEL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Model %{dclass_counter1->} unsupported with this version of chassisd", processor_chain([
	dup59,
	dup22,
	setc("event_description","Model number unsupported with this version of chassisd"),
	dup23,
]));

var msg212 = msg("CHASSISD_UNSUPPORTED_MODEL", part239);

var part240 = match("MESSAGE#208:CHASSISD_VERSION_MISMATCH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Version mismatch: %{info}", processor_chain([
	dup59,
	dup22,
	setc("event_description","Chassisd Version mismatch"),
	dup23,
]));

var msg213 = msg("CHASSISD_VERSION_MISMATCH", part240);

var part241 = match("MESSAGE#209:CHASSISD_HIGH_TEMP_CONDITION", "nwparser.payload", "%{process->} %{process_id->} %{event_type->} [junos@%{obj_name->} temperature=\"%{fld2}\" message=\"%{info}\"]", processor_chain([
	dup59,
	dup22,
	setc("event_description","CHASSISD HIGH TEMP CONDITION"),
	dup61,
	dup62,
]));

var msg214 = msg("CHASSISD_HIGH_TEMP_CONDITION", part241);

var part242 = match("MESSAGE#210:clean_process", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: process %{agent->} RESTART mode %{event_state->} new master=%{obj_name->} old failover=%{change_old->} new failover = %{change_new}", processor_chain([
	dup21,
	dup22,
	setc("event_description","process RESTART mode"),
	dup23,
]));

var msg215 = msg("clean_process", part242);

var part243 = match("MESSAGE#211:CM_JAVA", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Chassis %{group->} Linklocal MAC:%{macaddr}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Chassis Linklocal to MAC"),
	dup23,
]));

var msg216 = msg("CM_JAVA", part243);

var part244 = match("MESSAGE#212:DCD_AS_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","DCD must be run as root"),
	dup23,
]));

var msg217 = msg("DCD_AS_ROOT", part244);

var part245 = match("MESSAGE#213:DCD_FILTER_LIB_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Filter library initialization failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","Filter library initialization failed"),
	dup23,
]));

var msg218 = msg("DCD_FILTER_LIB_ERROR", part245);

var msg219 = msg("DCD_MALLOC_FAILED_INIT", dup139);

var part246 = match("MESSAGE#215:DCD_PARSE_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: errors while parsing configuration file", processor_chain([
	dup30,
	dup22,
	setc("event_description","errors while parsing configuration file"),
	dup23,
]));

var msg220 = msg("DCD_PARSE_EMERGENCY", part246);

var part247 = match("MESSAGE#216:DCD_PARSE_FILTER_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: errors while parsing filter index file", processor_chain([
	dup30,
	dup22,
	setc("event_description","errors while parsing filter index file"),
	dup23,
]));

var msg221 = msg("DCD_PARSE_FILTER_EMERGENCY", part247);

var part248 = match("MESSAGE#217:DCD_PARSE_MINI_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: errors while parsing configuration overlay", processor_chain([
	dup30,
	dup22,
	setc("event_description","errors while parsing configuration overlay"),
	dup23,
]));

var msg222 = msg("DCD_PARSE_MINI_EMERGENCY", part248);

var part249 = match("MESSAGE#218:DCD_PARSE_STATE_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: An unhandled state was encountered during interface parsing", processor_chain([
	dup30,
	dup22,
	setc("event_description","unhandled state was encountered during interface parsing"),
	dup23,
]));

var msg223 = msg("DCD_PARSE_STATE_EMERGENCY", part249);

var part250 = match("MESSAGE#219:DCD_POLICER_PARSE_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: errors while parsing policer indexfile", processor_chain([
	dup30,
	dup22,
	setc("event_description","errors while parsing policer indexfile"),
	dup23,
]));

var msg224 = msg("DCD_POLICER_PARSE_EMERGENCY", part250);

var part251 = match("MESSAGE#220:DCD_PULL_LOG_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to pull file %{filename->} after %{dclass_counter1->} retries last error=%{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Failed to pull file"),
	dup23,
]));

var msg225 = msg("DCD_PULL_LOG_FAILURE", part251);

var part252 = match("MESSAGE#221:DFWD_ARGUMENT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","DFWD ARGUMENT ERROR"),
	dup23,
]));

var msg226 = msg("DFWD_ARGUMENT_ERROR", part252);

var msg227 = msg("DFWD_MALLOC_FAILED_INIT", dup139);

var part253 = match("MESSAGE#223:DFWD_PARSE_FILTER_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} encountered errors while parsing filter index file", processor_chain([
	dup30,
	dup22,
	setc("event_description","errors encountered while parsing filter index file"),
	dup23,
]));

var msg228 = msg("DFWD_PARSE_FILTER_EMERGENCY", part253);

var part254 = match("MESSAGE#224:DFWD_PARSE_STATE_EMERGENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} encountered unhandled state while parsing interface", processor_chain([
	dup30,
	dup22,
	setc("event_description","encountered unhandled state while parsing interface"),
	dup23,
]));

var msg229 = msg("DFWD_PARSE_STATE_EMERGENCY", part254);

var msg230 = msg("ECCD_DAEMONIZE_FAILED", dup140);

var msg231 = msg("ECCD_DUPLICATE", dup141);

var part255 = match("MESSAGE#227:ECCD_LOOP_EXIT_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MainLoop return value: %{disposition}, error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ECCD LOOP EXIT FAILURE"),
	dup23,
]));

var msg232 = msg("ECCD_LOOP_EXIT_FAILURE", part255);

var part256 = match("MESSAGE#228:ECCD_NOT_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","ECCD Must be run as root"),
	dup23,
]));

var msg233 = msg("ECCD_NOT_ROOT", part256);

var part257 = match("MESSAGE#229:ECCD_PCI_FILE_OPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: open() failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ECCD PCI FILE OPEN FAILED"),
	dup23,
]));

var msg234 = msg("ECCD_PCI_FILE_OPEN_FAILED", part257);

var part258 = match("MESSAGE#230:ECCD_PCI_READ_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PCI read failure"),
	dup23,
]));

var msg235 = msg("ECCD_PCI_READ_FAILED", part258);

var part259 = match("MESSAGE#231:ECCD_PCI_WRITE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PCI write failure"),
	dup23,
]));

var msg236 = msg("ECCD_PCI_WRITE_FAILED", part259);

var msg237 = msg("ECCD_PID_FILE_LOCK", dup142);

var msg238 = msg("ECCD_PID_FILE_UPDATE", dup143);

var part260 = match("MESSAGE#234:ECCD_TRACE_FILE_OPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ECCD TRACE FILE OPEN FAILURE"),
	dup23,
]));

var msg239 = msg("ECCD_TRACE_FILE_OPEN_FAILED", part260);

var part261 = match("MESSAGE#235:ECCD_usage", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}: %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","ECCD Usage"),
	dup23,
]));

var msg240 = msg("ECCD_usage", part261);

var part262 = match("MESSAGE#236:EVENTD_AUDIT_SHOW", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User %{username->} viewed security audit log with arguments: %{param}", processor_chain([
	dup21,
	dup22,
	setc("event_description","User viewed security audit log with arguments"),
	dup23,
]));

var msg241 = msg("EVENTD_AUDIT_SHOW", part262);

var part263 = match("MESSAGE#237:FLOW_REASSEMBLE_SUCCEED", "nwparser.payload", "%{event_type}: Packet merged source %{saddr->} destination %{daddr->} ipid %{fld11->} succeed", processor_chain([
	dup21,
	dup22,
	dup23,
]));

var msg242 = msg("FLOW_REASSEMBLE_SUCCEED", part263);

var part264 = match("MESSAGE#238:FSAD_CHANGE_FILE_OWNER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to change owner of file `%{filename}' to user %{username}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to change owner of file"),
	dup23,
]));

var msg243 = msg("FSAD_CHANGE_FILE_OWNER", part264);

var part265 = match("MESSAGE#239:FSAD_CONFIG_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","FSAD CONFIG ERROR"),
	dup23,
]));

var msg244 = msg("FSAD_CONFIG_ERROR", part265);

var part266 = match("MESSAGE#240:FSAD_CONNTIMEDOUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Connection timed out to the client (%{shost}, %{saddr}) having request type %{obj_type}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Connection timed out to client"),
	dup23,
]));

var msg245 = msg("FSAD_CONNTIMEDOUT", part266);

var part267 = match("MESSAGE#241:FSAD_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","FSAD_FAILED"),
	dup23,
]));

var msg246 = msg("FSAD_FAILED", part267);

var part268 = match("MESSAGE#242:FSAD_FETCHTIMEDOUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Fetch to server %{hostname->} for file `%{filename}' timed out", processor_chain([
	dup30,
	dup22,
	setc("event_description","Fetch to server to get file timed out"),
	dup23,
]));

var msg247 = msg("FSAD_FETCHTIMEDOUT", part268);

var part269 = match("MESSAGE#243:FSAD_FILE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: fn failed for file `%{filename}' with error message %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","fn failed for file"),
	dup23,
]));

var msg248 = msg("FSAD_FILE_FAILED", part269);

var part270 = match("MESSAGE#244:FSAD_FILE_REMOVE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to remove file `%{filename}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to remove file"),
	dup23,
]));

var msg249 = msg("FSAD_FILE_REMOVE", part270);

var part271 = match("MESSAGE#245:FSAD_FILE_RENAME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to rename file `%{filename}' to `%{resultcode}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to rename file"),
	dup23,
]));

var msg250 = msg("FSAD_FILE_RENAME", part271);

var part272 = match("MESSAGE#246:FSAD_FILE_STAT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} failed for file pathname %{filename}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","stat failed for file"),
	dup23,
]));

var msg251 = msg("FSAD_FILE_STAT", part272);

var part273 = match("MESSAGE#247:FSAD_FILE_SYNC", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to sync file %{filename}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to sync file"),
	dup23,
]));

var msg252 = msg("FSAD_FILE_SYNC", part273);

var part274 = match("MESSAGE#248:FSAD_MAXCONN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Upper limit reached in fsad for handling connections", processor_chain([
	dup30,
	dup22,
	setc("event_description","Upper limit reached in fsad"),
	dup23,
]));

var msg253 = msg("FSAD_MAXCONN", part274);

var part275 = match("MESSAGE#249:FSAD_MEMORYALLOC_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service->} failed in the function %{action->} (%{resultcode})", processor_chain([
	dup51,
	dup22,
	setc("event_description","FSAD MEMORYALLOC FAILED"),
	dup23,
]));

var msg254 = msg("FSAD_MEMORYALLOC_FAILED", part275);

var part276 = match("MESSAGE#250:FSAD_NOT_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","FSAD must be run as root"),
	dup23,
]));

var msg255 = msg("FSAD_NOT_ROOT", part276);

var part277 = match("MESSAGE#251:FSAD_PARENT_DIRECTORY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: invalid directory: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","invalid directory"),
	dup23,
]));

var msg256 = msg("FSAD_PARENT_DIRECTORY", part277);

var part278 = match("MESSAGE#252:FSAD_PATH_IS_DIRECTORY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: File path cannot be a directory (%{filename})", processor_chain([
	dup30,
	dup22,
	setc("event_description","File path cannot be a directory"),
	dup23,
]));

var msg257 = msg("FSAD_PATH_IS_DIRECTORY", part278);

var part279 = match("MESSAGE#253:FSAD_PATH_IS_SPECIAL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Not a regular file (%{filename})", processor_chain([
	dup30,
	dup22,
	setc("event_description","Not a regular file"),
	dup23,
]));

var msg258 = msg("FSAD_PATH_IS_SPECIAL", part279);

var part280 = match("MESSAGE#254:FSAD_RECVERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: fsad received error message from client having request type %{obj_type->} at (%{saddr}, %{sport})", processor_chain([
	dup30,
	dup22,
	setc("event_description","fsad received error message from client"),
	dup23,
]));

var msg259 = msg("FSAD_RECVERROR", part280);

var part281 = match("MESSAGE#255:FSAD_TERMINATED_CONNECTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Open file %{filename}` closed due to %{result}", processor_chain([
	dup27,
	dup22,
	setc("event_description","FSAD TERMINATED CONNECTION"),
	dup23,
]));

var msg260 = msg("FSAD_TERMINATED_CONNECTION", part281);

var part282 = match("MESSAGE#256:FSAD_TERMINATING_SIGNAL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Received terminating %{resultcode}; %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Received terminating signal"),
	dup23,
]));

var msg261 = msg("FSAD_TERMINATING_SIGNAL", part282);

var part283 = match("MESSAGE#257:FSAD_TRACEOPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Open operation on trace file `%{filename}' returned error %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Open operation on trace file failed"),
	dup23,
]));

var msg262 = msg("FSAD_TRACEOPEN_FAILED", part283);

var part284 = match("MESSAGE#258:FSAD_USAGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Incorrect usage, %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Incorrect FSAD usage"),
	dup23,
]));

var msg263 = msg("FSAD_USAGE", part284);

var part285 = match("MESSAGE#259:GGSN_ALARM_TRAP_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","GGSN ALARM TRAP FAILED"),
	dup23,
]));

var msg264 = msg("GGSN_ALARM_TRAP_FAILED", part285);

var part286 = match("MESSAGE#260:GGSN_ALARM_TRAP_SEND", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","GGSN ALARM TRAP SEND FAILED"),
	dup23,
]));

var msg265 = msg("GGSN_ALARM_TRAP_SEND", part286);

var part287 = match("MESSAGE#261:GGSN_TRAP_SEND", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unknown trap request type %{obj_type}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unknown trap request type"),
	dup23,
]));

var msg266 = msg("GGSN_TRAP_SEND", part287);

var part288 = match("MESSAGE#262:JADE_AUTH_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Authorization failed: %{result}", processor_chain([
	dup69,
	dup34,
	setc("ec_subject","Service"),
	dup43,
	dup22,
	setc("event_description","Authorization failed"),
	dup23,
]));

var msg267 = msg("JADE_AUTH_ERROR", part288);

var part289 = match("MESSAGE#263:JADE_EXEC_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: CLI %{resultcode->} %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","JADE EXEC ERROR"),
	dup23,
]));

var msg268 = msg("JADE_EXEC_ERROR", part289);

var part290 = match("MESSAGE#264:JADE_NO_LOCAL_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Local user %{username->} does not exist", processor_chain([
	dup30,
	dup22,
	setc("event_description","Local user does not exist"),
	dup23,
]));

var msg269 = msg("JADE_NO_LOCAL_USER", part290);

var part291 = match("MESSAGE#265:JADE_PAM_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","JADE PAM error"),
	dup23,
]));

var msg270 = msg("JADE_PAM_ERROR", part291);

var part292 = match("MESSAGE#266:JADE_PAM_NO_LOCAL_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to get local username from PAM: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to get local username from PAM"),
	dup23,
]));

var msg271 = msg("JADE_PAM_NO_LOCAL_USER", part292);

var part293 = match("MESSAGE#267:KERN_ARP_ADDR_CHANGE", "nwparser.payload", "%{process}: %{event_type}: arp info overwritten for %{saddr->} from %{smacaddr->} to %{dmacaddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","arp info overwritten"),
	dup23,
]));

var msg272 = msg("KERN_ARP_ADDR_CHANGE", part293);

var part294 = match("MESSAGE#268:KMD_PM_SA_ESTABLISHED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Local gateway: %{gateway}, Remote gateway: %{fld1}, Local ID:%{fld2}, Remote ID:%{fld3}, Direction:%{fld4}, SPI:%{fld5}", processor_chain([
	dup30,
	dup22,
	setc("event_description","security association has been established"),
	dup23,
]));

var msg273 = msg("KMD_PM_SA_ESTABLISHED", part294);

var part295 = match("MESSAGE#269:L2CPD_TASK_REINIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reinitialized", processor_chain([
	dup21,
	dup22,
	setc("event_description","Task Reinitialized"),
	dup61,
	dup23,
]));

var msg274 = msg("L2CPD_TASK_REINIT", part295);

var part296 = match("MESSAGE#270:LIBJNX_EXEC_EXITED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command stopped: PID %{child_pid}, signal='%{obj_type}' %{result}, command '%{action}'", processor_chain([
	dup21,
	dup22,
	dup70,
	dup23,
]));

var msg275 = msg("LIBJNX_EXEC_EXITED", part296);

var part297 = match("MESSAGE#271:LIBJNX_EXEC_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child exec failed for command '%{action}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Child exec failed for command"),
	dup23,
]));

var msg276 = msg("LIBJNX_EXEC_FAILED", part297);

var msg277 = msg("LIBJNX_EXEC_PIPE", dup144);

var part298 = match("MESSAGE#273:LIBJNX_EXEC_SIGNALED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command received signal: PID %{child_pid}, signal %{result}, command '%{action}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Command received signal"),
	dup23,
]));

var msg278 = msg("LIBJNX_EXEC_SIGNALED", part298);

var part299 = match("MESSAGE#274:LIBJNX_EXEC_WEXIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command exited: PID %{child_pid}, status %{result}, command '%{action}'", processor_chain([
	dup21,
	dup22,
	dup72,
	dup23,
]));

var msg279 = msg("LIBJNX_EXEC_WEXIT", part299);

var part300 = match("MESSAGE#275:LIBJNX_FILE_COPY_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: copy_file_to_transfer_dir failed to copy from source to destination", processor_chain([
	dup73,
	dup22,
	setc("event_description","copy_file_to_transfer_dir failed to copy"),
	dup23,
]));

var msg280 = msg("LIBJNX_FILE_COPY_FAILED", part300);

var part301 = match("MESSAGE#276:LIBJNX_PRIV_LOWER_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to lower privilege level: %{result}", processor_chain([
	dup73,
	dup22,
	setc("event_description","Unable to lower privilege level"),
	dup23,
]));

var msg281 = msg("LIBJNX_PRIV_LOWER_FAILED", part301);

var part302 = match("MESSAGE#277:LIBJNX_PRIV_RAISE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to raise privilege level: %{result}", processor_chain([
	dup73,
	dup22,
	setc("event_description","Unable to raise privilege level"),
	dup23,
]));

var msg282 = msg("LIBJNX_PRIV_RAISE_FAILED", part302);

var part303 = match("MESSAGE#278:LIBJNX_REPLICATE_RCP_EXEC_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup73,
	dup22,
	setc("event_description","rcp failed"),
	dup23,
]));

var msg283 = msg("LIBJNX_REPLICATE_RCP_EXEC_FAILED", part303);

var part304 = match("MESSAGE#279:LIBJNX_ROTATE_COMPRESS_EXEC_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{resultcode->} %{dclass_counter1->} -f %{action}: %{result}", processor_chain([
	dup73,
	dup22,
	setc("event_description","ROTATE COMPRESS EXEC FAILED"),
	dup23,
]));

var msg284 = msg("LIBJNX_ROTATE_COMPRESS_EXEC_FAILED", part304);

var part305 = match("MESSAGE#280:LIBSERVICED_CLIENT_CONNECTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Client connection error: %{result}", processor_chain([
	dup74,
	dup22,
	setc("event_description","Client connection error"),
	dup23,
]));

var msg285 = msg("LIBSERVICED_CLIENT_CONNECTION", part305);

var part306 = match("MESSAGE#281:LIBSERVICED_OUTBOUND_REQUEST", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Outbound request failed for command [%{action}]: %{result}", processor_chain([
	dup73,
	dup22,
	setc("event_description","Outbound request failed for command"),
	dup23,
]));

var msg286 = msg("LIBSERVICED_OUTBOUND_REQUEST", part306);

var part307 = match("MESSAGE#282:LIBSERVICED_SNMP_LOST_CONNECTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Connection closed while receiving from client %{dclass_counter1}", processor_chain([
	dup27,
	dup22,
	setc("event_description","Connection closed while receiving from client"),
	dup23,
]));

var msg287 = msg("LIBSERVICED_SNMP_LOST_CONNECTION", part307);

var part308 = match("MESSAGE#283:LIBSERVICED_SOCKET_BIND", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{resultcode}: unable to bind socket %{ssid}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","unable to bind socket"),
	dup23,
]));

var msg288 = msg("LIBSERVICED_SOCKET_BIND", part308);

var part309 = match("MESSAGE#284:LIBSERVICED_SOCKET_PRIVATIZE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to attach socket %{ssid->} to management routing instance: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to attach socket to management routing instance"),
	dup23,
]));

var msg289 = msg("LIBSERVICED_SOCKET_PRIVATIZE", part309);

var part310 = match("MESSAGE#285:LICENSE_EXPIRED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","LICENSE EXPIRED"),
	dup23,
]));

var msg290 = msg("LICENSE_EXPIRED", part310);

var part311 = match("MESSAGE#286:LICENSE_EXPIRED_KEY_DELETED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: License key \"%{filename}\" has expired.", processor_chain([
	dup21,
	dup22,
	setc("event_description","License key has expired"),
	dup23,
]));

var msg291 = msg("LICENSE_EXPIRED_KEY_DELETED", part311);

var part312 = match("MESSAGE#287:LICENSE_NEARING_EXPIRY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: License for feature %{disposition->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","License key expiration soon"),
	dup23,
]));

var msg292 = msg("LICENSE_NEARING_EXPIRY", part312);

var part313 = match("MESSAGE#288:LOGIN_ABORTED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Client aborted login", processor_chain([
	dup30,
	dup22,
	setc("event_description","client aborted login"),
	dup23,
]));

var msg293 = msg("LOGIN_ABORTED", part313);

var part314 = match("MESSAGE#289:LOGIN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Login failed for user %{username->} from host %{dhost}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	dup23,
]));

var msg294 = msg("LOGIN_FAILED", part314);

var part315 = match("MESSAGE#290:LOGIN_FAILED_INCORRECT_PASSWORD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Incorrect password for user %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Incorrect password for user"),
	dup23,
]));

var msg295 = msg("LOGIN_FAILED_INCORRECT_PASSWORD", part315);

var part316 = match("MESSAGE#291:LOGIN_FAILED_SET_CONTEXT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to set context for user %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Failed to set context for user"),
	dup23,
]));

var msg296 = msg("LOGIN_FAILED_SET_CONTEXT", part316);

var part317 = match("MESSAGE#292:LOGIN_FAILED_SET_LOGIN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to set login ID for user %{username}: %{dhost}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Failed to set login ID for user"),
	dup23,
]));

var msg297 = msg("LOGIN_FAILED_SET_LOGIN", part317);

var part318 = match("MESSAGE#293:LOGIN_HOSTNAME_UNRESOLVED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to resolve hostname %{dhost}: %{info}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Unable to resolve hostname"),
	dup23,
]));

var msg298 = msg("LOGIN_HOSTNAME_UNRESOLVED", part318);

var part319 = match("MESSAGE#294:LOGIN_INFORMATION/2", "nwparser.p0", "%{event_type}: %{p0}");

var part320 = match("MESSAGE#294:LOGIN_INFORMATION/4", "nwparser.p0", "%{username->} logged in from host %{dhost->} on %{p0}");

var part321 = match("MESSAGE#294:LOGIN_INFORMATION/5_0", "nwparser.p0", "device %{p0}");

var select34 = linear_select([
	part321,
	dup45,
]);

var part322 = match("MESSAGE#294:LOGIN_INFORMATION/6", "nwparser.p0", "%{terminal}");

var all19 = all_match({
	processors: [
		dup39,
		dup137,
		part319,
		dup145,
		part320,
		select34,
		part322,
	],
	on_success: processor_chain([
		dup33,
		dup34,
		dup35,
		dup36,
		dup37,
		dup22,
		setc("event_description","Successful Login"),
		dup23,
	]),
});

var msg299 = msg("LOGIN_INFORMATION", all19);

var part323 = match("MESSAGE#295:LOGIN_INVALID_LOCAL_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No entry in local password file for user %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","No entry in local password file for user"),
	dup23,
]));

var msg300 = msg("LOGIN_INVALID_LOCAL_USER", part323);

var part324 = match("MESSAGE#296:LOGIN_MALFORMED_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Invalid username: %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Invalid username"),
	dup23,
]));

var msg301 = msg("LOGIN_MALFORMED_USER", part324);

var part325 = match("MESSAGE#297:LOGIN_PAM_AUTHENTICATION_ERROR/1_0", "nwparser.p0", "PAM authentication error for user %{p0}");

var part326 = match("MESSAGE#297:LOGIN_PAM_AUTHENTICATION_ERROR/1_1", "nwparser.p0", "Failed password for user %{p0}");

var select35 = linear_select([
	part325,
	part326,
]);

var part327 = match("MESSAGE#297:LOGIN_PAM_AUTHENTICATION_ERROR/2", "nwparser.p0", "%{username}");

var all20 = all_match({
	processors: [
		dup50,
		select35,
		part327,
	],
	on_success: processor_chain([
		dup44,
		dup34,
		dup35,
		dup36,
		dup43,
		dup22,
		dup75,
		setc("result","PAM authentication error for user"),
		dup23,
	]),
});

var msg302 = msg("LOGIN_PAM_AUTHENTICATION_ERROR", all20);

var part328 = match("MESSAGE#298:LOGIN_PAM_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failure while authenticating user %{username}: %{dhost}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	setc("event_description","PAM authentication failure"),
	setc("result","Failure while authenticating user"),
	dup23,
]));

var msg303 = msg("LOGIN_PAM_ERROR", part328);

var part329 = match("MESSAGE#299:LOGIN_PAM_MAX_RETRIES", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Too many retries while authenticating user %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Too many retries while authenticating user"),
	dup23,
]));

var msg304 = msg("LOGIN_PAM_MAX_RETRIES", part329);

var part330 = match("MESSAGE#300:LOGIN_PAM_NONLOCAL_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User %{username->} authenticated but has no local login ID", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","User authenticated but has no local login ID"),
	dup23,
]));

var msg305 = msg("LOGIN_PAM_NONLOCAL_USER", part330);

var part331 = match("MESSAGE#301:LOGIN_PAM_STOP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to end PAM session: %{info}", processor_chain([
	setc("eventcategory","1303000000"),
	dup34,
	dup43,
	dup22,
	setc("event_description","Failed to end PAM session"),
	dup23,
]));

var msg306 = msg("LOGIN_PAM_STOP", part331);

var part332 = match("MESSAGE#302:LOGIN_PAM_USER_UNKNOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Attempt to authenticate unknown user %{username}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Attempt to authenticate unknown user"),
	dup23,
]));

var msg307 = msg("LOGIN_PAM_USER_UNKNOWN", part332);

var part333 = match("MESSAGE#303:LOGIN_PASSWORD_EXPIRED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Forcing change of expired password for user %{username}>", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Forcing change of expired password for user"),
	dup23,
]));

var msg308 = msg("LOGIN_PASSWORD_EXPIRED", part333);

var part334 = match("MESSAGE#304:LOGIN_REFUSED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Login of user %{username->} from host %{shost->} on %{terminal->} was refused: %{info}", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Login of user refused"),
	dup23,
]));

var msg309 = msg("LOGIN_REFUSED", part334);

var part335 = match("MESSAGE#305:LOGIN_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User %{username->} logged in as root from host %{shost->} on %{terminal}", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	setc("event_description","successful login as root"),
	setc("result","User logged in as root"),
	dup23,
]));

var msg310 = msg("LOGIN_ROOT", part335);

var part336 = match("MESSAGE#306:LOGIN_TIMED_OUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Login attempt timed out after %{dclass_counter1->} seconds", processor_chain([
	dup44,
	dup34,
	dup36,
	dup43,
	dup22,
	dup75,
	setc("result","Login attempt timed out"),
	dup23,
]));

var msg311 = msg("LOGIN_TIMED_OUT", part336);

var part337 = match("MESSAGE#307:MIB2D_ATM_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MIB2D ATM ERROR"),
	dup23,
]));

var msg312 = msg("MIB2D_ATM_ERROR", part337);

var part338 = match("MESSAGE#308:MIB2D_CONFIG_CHECK_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CONFIG CHECK FAILED"),
	dup23,
]));

var msg313 = msg("MIB2D_CONFIG_CHECK_FAILED", part338);

var part339 = match("MESSAGE#309:MIB2D_FILE_OPEN_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to open file '%{filename}': %{result}", processor_chain([
	dup30,
	dup22,
	dup78,
	dup23,
]));

var msg314 = msg("MIB2D_FILE_OPEN_FAILURE", part339);

var msg315 = msg("MIB2D_IFD_IFINDEX_FAILURE", dup146);

var msg316 = msg("MIB2D_IFL_IFINDEX_FAILURE", dup146);

var part340 = match("MESSAGE#312:MIB2D_INIT_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: mib2d initialization failure: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","mib2d initialization failure"),
	dup23,
]));

var msg317 = msg("MIB2D_INIT_FAILURE", part340);

var part341 = match("MESSAGE#313:MIB2D_KVM_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MIB2D KVM FAILURE"),
	dup23,
]));

var msg318 = msg("MIB2D_KVM_FAILURE", part341);

var part342 = match("MESSAGE#314:MIB2D_RTSLIB_READ_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: failed in %{dclass_counter1->} %{dclass_counter2->} index (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","MIB2D RTSLIB READ FAILURE"),
	dup23,
]));

var msg319 = msg("MIB2D_RTSLIB_READ_FAILURE", part342);

var part343 = match("MESSAGE#315:MIB2D_RTSLIB_SEQ_MISMATCH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: sequence mismatch (%{result}), %{action}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RTSLIB sequence mismatch"),
	dup23,
]));

var msg320 = msg("MIB2D_RTSLIB_SEQ_MISMATCH", part343);

var part344 = match("MESSAGE#316:MIB2D_SYSCTL_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MIB2D SYSCTL FAILURE"),
	dup23,
]));

var msg321 = msg("MIB2D_SYSCTL_FAILURE", part344);

var part345 = match("MESSAGE#317:MIB2D_TRAP_HEADER_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: trap_request_header failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","trap_request_header failed"),
	dup23,
]));

var msg322 = msg("MIB2D_TRAP_HEADER_FAILURE", part345);

var part346 = match("MESSAGE#318:MIB2D_TRAP_SEND_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{service}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MIB2D TRAP SEND FAILURE"),
	dup23,
]));

var msg323 = msg("MIB2D_TRAP_SEND_FAILURE", part346);

var part347 = match("MESSAGE#319:Multiuser", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: old requested_transition==%{change_new->} sighupped=%{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","user sighupped"),
	dup23,
]));

var msg324 = msg("Multiuser", part347);

var part348 = match("MESSAGE#320:NASD_AUTHENTICATION_CREATE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to allocate authentication handle: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to allocate authentication handle"),
	dup23,
]));

var msg325 = msg("NASD_AUTHENTICATION_CREATE_FAILED", part348);

var part349 = match("MESSAGE#321:NASD_CHAP_AUTHENTICATION_IN_PROGRESS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{interface}: received %{filename}, authentication already in progress", processor_chain([
	dup80,
	dup34,
	dup43,
	dup22,
	setc("event_description","authentication already in progress"),
	dup23,
]));

var msg326 = msg("NASD_CHAP_AUTHENTICATION_IN_PROGRESS", part349);

var part350 = match("MESSAGE#322:NASD_CHAP_GETHOSTNAME_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{interface}: unable to obtain hostname for outgoing CHAP message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","unable to obtain hostname for outgoing CHAP message"),
	dup23,
]));

var msg327 = msg("NASD_CHAP_GETHOSTNAME_FAILED", part350);

var part351 = match("MESSAGE#323:NASD_CHAP_INVALID_CHAP_IDENTIFIER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{interface}: received %{filename->} expected CHAP ID: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHAP INVALID_CHAP IDENTIFIER"),
	dup23,
]));

var msg328 = msg("NASD_CHAP_INVALID_CHAP_IDENTIFIER", part351);

var part352 = match("MESSAGE#324:NASD_CHAP_INVALID_OPCODE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{interface}.%{dclass_counter1}: invalid operation code received %{filename}, CHAP ID: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHAP INVALID OPCODE"),
	dup23,
]));

var msg329 = msg("NASD_CHAP_INVALID_OPCODE", part352);

var part353 = match("MESSAGE#325:NASD_CHAP_LOCAL_NAME_UNAVAILABLE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to determine value for '%{username}' in outgoing CHAP packet", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to determine value for username in outgoing CHAP packet"),
	dup23,
]));

var msg330 = msg("NASD_CHAP_LOCAL_NAME_UNAVAILABLE", part353);

var part354 = match("MESSAGE#326:NASD_CHAP_MESSAGE_UNEXPECTED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{interface}: received %{filename}", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHAP MESSAGE UNEXPECTED"),
	dup23,
]));

var msg331 = msg("NASD_CHAP_MESSAGE_UNEXPECTED", part354);

var part355 = match("MESSAGE#327:NASD_CHAP_REPLAY_ATTACK_DETECTED", "nwparser.payload", "%{process}[%{ssid}]: %{event_type}: %{interface}.%{dclass_counter1}: received %{filename->} %{result}.%{info}", processor_chain([
	dup81,
	dup22,
	setc("event_description","CHAP REPLAY ATTACK DETECTED"),
	dup23,
]));

var msg332 = msg("NASD_CHAP_REPLAY_ATTACK_DETECTED", part355);

var part356 = match("MESSAGE#328:NASD_CONFIG_GET_LAST_MODIFIED_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to determine last modified time of JUNOS configuration database: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to determine last modified time of JUNOS configuration database"),
	dup23,
]));

var msg333 = msg("NASD_CONFIG_GET_LAST_MODIFIED_FAILED", part356);

var msg334 = msg("NASD_DAEMONIZE_FAILED", dup140);

var part357 = match("MESSAGE#330:NASD_DB_ALLOC_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to allocate database object: %{filename}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to allocate database object"),
	dup23,
]));

var msg335 = msg("NASD_DB_ALLOC_FAILURE", part357);

var part358 = match("MESSAGE#331:NASD_DB_TABLE_CREATE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{filename}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","DB TABLE CREATE FAILURE"),
	dup23,
]));

var msg336 = msg("NASD_DB_TABLE_CREATE_FAILURE", part358);

var msg337 = msg("NASD_DUPLICATE", dup141);

var part359 = match("MESSAGE#333:NASD_EVLIB_CREATE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} with: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","EVLIB CREATE FAILURE"),
	dup23,
]));

var msg338 = msg("NASD_EVLIB_CREATE_FAILURE", part359);

var part360 = match("MESSAGE#334:NASD_EVLIB_EXIT_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} value: %{result}, error: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","EVLIB EXIT FAILURE"),
	dup23,
]));

var msg339 = msg("NASD_EVLIB_EXIT_FAILURE", part360);

var part361 = match("MESSAGE#335:NASD_LOCAL_CREATE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to allocate LOCAL module handle: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to allocate LOCAL module handle"),
	dup23,
]));

var msg340 = msg("NASD_LOCAL_CREATE_FAILED", part361);

var part362 = match("MESSAGE#336:NASD_NOT_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","NASD must be run as root"),
	dup23,
]));

var msg341 = msg("NASD_NOT_ROOT", part362);

var msg342 = msg("NASD_PID_FILE_LOCK", dup142);

var msg343 = msg("NASD_PID_FILE_UPDATE", dup143);

var part363 = match("MESSAGE#339:NASD_POST_CONFIGURE_EVENT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","POST CONFIGURE EVENT FAILED"),
	dup23,
]));

var msg344 = msg("NASD_POST_CONFIGURE_EVENT_FAILED", part363);

var part364 = match("MESSAGE#340:NASD_PPP_READ_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PPP READ FAILURE"),
	dup23,
]));

var msg345 = msg("NASD_PPP_READ_FAILURE", part364);

var part365 = match("MESSAGE#341:NASD_PPP_SEND_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to send message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to send message"),
	dup23,
]));

var msg346 = msg("NASD_PPP_SEND_FAILURE", part365);

var part366 = match("MESSAGE#342:NASD_PPP_SEND_PARTIAL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to send all of message: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to send all of message"),
	dup23,
]));

var msg347 = msg("NASD_PPP_SEND_PARTIAL", part366);

var part367 = match("MESSAGE#343:NASD_PPP_UNRECOGNIZED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unrecognized authentication protocol: %{protocol}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unrecognized authentication protocol"),
	dup23,
]));

var msg348 = msg("NASD_PPP_UNRECOGNIZED", part367);

var part368 = match("MESSAGE#344:NASD_RADIUS_ALLOCATE_PASSWORD_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} when allocating password for RADIUS: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS password allocation failure"),
	dup23,
]));

var msg349 = msg("NASD_RADIUS_ALLOCATE_PASSWORD_FAILED", part368);

var part369 = match("MESSAGE#345:NASD_RADIUS_CONFIG_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS CONFIG FAILED"),
	dup23,
]));

var msg350 = msg("NASD_RADIUS_CONFIG_FAILED", part369);

var part370 = match("MESSAGE#346:NASD_RADIUS_CREATE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to allocate RADIUS module handle: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to allocate RADIUS module handle"),
	dup23,
]));

var msg351 = msg("NASD_RADIUS_CREATE_FAILED", part370);

var part371 = match("MESSAGE#347:NASD_RADIUS_CREATE_REQUEST_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS CREATE REQUEST FAILED"),
	dup23,
]));

var msg352 = msg("NASD_RADIUS_CREATE_REQUEST_FAILED", part371);

var part372 = match("MESSAGE#348:NASD_RADIUS_GETHOSTNAME_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to obtain hostname for outgoing RADIUS message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to obtain hostname for outgoing RADIUS message"),
	dup23,
]));

var msg353 = msg("NASD_RADIUS_GETHOSTNAME_FAILED", part372);

var part373 = match("MESSAGE#349:NASD_RADIUS_MESSAGE_UNEXPECTED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unknown response from RADIUS server: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unknown response from RADIUS server"),
	dup23,
]));

var msg354 = msg("NASD_RADIUS_MESSAGE_UNEXPECTED", part373);

var part374 = match("MESSAGE#350:NASD_RADIUS_OPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS OPEN FAILED"),
	dup23,
]));

var msg355 = msg("NASD_RADIUS_OPEN_FAILED", part374);

var part375 = match("MESSAGE#351:NASD_RADIUS_SELECT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS SELECT FAILED"),
	dup23,
]));

var msg356 = msg("NASD_RADIUS_SELECT_FAILED", part375);

var part376 = match("MESSAGE#352:NASD_RADIUS_SET_TIMER_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RADIUS SET TIMER FAILED"),
	dup23,
]));

var msg357 = msg("NASD_RADIUS_SET_TIMER_FAILED", part376);

var part377 = match("MESSAGE#353:NASD_TRACE_FILE_OPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TRACE FILE OPEN FAILED"),
	dup23,
]));

var msg358 = msg("NASD_TRACE_FILE_OPEN_FAILED", part377);

var part378 = match("MESSAGE#354:NASD_usage", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}: %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","NASD Usage"),
	dup23,
]));

var msg359 = msg("NASD_usage", part378);

var part379 = match("MESSAGE#355:NOTICE", "nwparser.payload", "%{agent}: %{event_type}:%{action}: %{event_description}: The %{result}", processor_chain([
	dup21,
	dup22,
	dup23,
]));

var msg360 = msg("NOTICE", part379);

var part380 = match("MESSAGE#356:PFE_FW_SYSLOG_IP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: FW: %{smacaddr->} %{fld10->} %{protocol->} %{saddr->} %{daddr->} %{sport->} %{dport->} (%{packets->} packets)", processor_chain([
	dup21,
	dup22,
	dup82,
	dup23,
]));

var msg361 = msg("PFE_FW_SYSLOG_IP", part380);

var part381 = match("MESSAGE#357:PFE_FW_SYSLOG_IP:01", "nwparser.payload", "%{hostip->} %{hostname->} %{event_type}: FW: %{smacaddr->} %{fld10->} %{protocol->} %{saddr->} %{daddr->} %{sport->} %{dport->} (%{packets->} packets)", processor_chain([
	dup21,
	dup22,
	dup82,
	dup23,
]));

var msg362 = msg("PFE_FW_SYSLOG_IP:01", part381);

var select36 = linear_select([
	msg361,
	msg362,
]);

var part382 = match("MESSAGE#358:PFE_NH_RESOLVE_THROTTLED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Next-hop resolution requests from interface %{interface->} throttled", processor_chain([
	dup21,
	dup22,
	setc("event_description","Next-hop resolution requests throttled"),
	dup23,
]));

var msg363 = msg("PFE_NH_RESOLVE_THROTTLED", part382);

var part383 = match("MESSAGE#359:PING_TEST_COMPLETED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pingCtlOwnerIndex = %{dclass_counter1}, pingCtlTestName = %{obj_name}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PING TEST COMPLETED"),
	dup23,
]));

var msg364 = msg("PING_TEST_COMPLETED", part383);

var part384 = match("MESSAGE#360:PING_TEST_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pingCtlOwnerIndex = %{dclass_counter1}, pingCtlTestName = %{obj_name}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PING TEST FAILED"),
	dup23,
]));

var msg365 = msg("PING_TEST_FAILED", part384);

var part385 = match("MESSAGE#361:process_mode/2", "nwparser.p0", "%{p0}");

var part386 = match("MESSAGE#361:process_mode/3_0", "nwparser.p0", "%{event_type}: %{p0}");

var part387 = match("MESSAGE#361:process_mode/3_1", "nwparser.p0", "%{event_type->} %{p0}");

var select37 = linear_select([
	part386,
	part387,
]);

var part388 = match("MESSAGE#361:process_mode/4", "nwparser.p0", "mode=%{protocol->} cmd=%{action->} master_mode=%{result}");

var all21 = all_match({
	processors: [
		dup39,
		dup137,
		part385,
		select37,
		part388,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup83,
		dup23,
	]),
});

var msg366 = msg("process_mode", all21);

var part389 = match("MESSAGE#362:process_mode:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: current_mode=%{protocol}, requested_mode=%{result}, cmd=%{action}", processor_chain([
	dup21,
	dup22,
	dup83,
	dup23,
]));

var msg367 = msg("process_mode:01", part389);

var select38 = linear_select([
	msg366,
	msg367,
]);

var part390 = match("MESSAGE#363:PWC_EXIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} exiting with status %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","process exit with status"),
	dup23,
]));

var msg368 = msg("PWC_EXIT", part390);

var part391 = match("MESSAGE#364:PWC_HOLD_RELEASE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} released child %{child_pid->} from %{dclass_counter1->} state", processor_chain([
	dup21,
	dup22,
	setc("event_description","Process released child from state"),
	dup23,
]));

var msg369 = msg("PWC_HOLD_RELEASE", part391);

var part392 = match("MESSAGE#365:PWC_INVALID_RUNS_ARGUMENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}, not %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","invalid runs argument"),
	dup23,
]));

var msg370 = msg("PWC_INVALID_RUNS_ARGUMENT", part392);

var part393 = match("MESSAGE#366:PWC_INVALID_TIMEOUT_ARGUMENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","INVALID TIMEOUT ARGUMENT"),
	dup23,
]));

var msg371 = msg("PWC_INVALID_TIMEOUT_ARGUMENT", part393);

var part394 = match("MESSAGE#367:PWC_KILLED_BY_SIGNAL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pwc process %{agent->} received terminating signal", processor_chain([
	dup21,
	dup22,
	setc("event_description","pwc process received terminating signal"),
	dup23,
]));

var msg372 = msg("PWC_KILLED_BY_SIGNAL", part394);

var part395 = match("MESSAGE#368:PWC_KILL_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pwc is sending %{resultcode->} to child %{child_pid}", processor_chain([
	dup30,
	dup22,
	setc("event_description","pwc is sending kill event to child"),
	dup23,
]));

var msg373 = msg("PWC_KILL_EVENT", part395);

var part396 = match("MESSAGE#369:PWC_KILL_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to kill process %{child_pid}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to kill process"),
	dup23,
]));

var msg374 = msg("PWC_KILL_FAILED", part396);

var part397 = match("MESSAGE#370:PWC_KQUEUE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: kevent failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","kevent failed"),
	dup23,
]));

var msg375 = msg("PWC_KQUEUE_ERROR", part397);

var part398 = match("MESSAGE#371:PWC_KQUEUE_INIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to create kqueue: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to create kqueue"),
	dup23,
]));

var msg376 = msg("PWC_KQUEUE_INIT", part398);

var part399 = match("MESSAGE#372:PWC_KQUEUE_REGISTER_FILTER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to register kqueue filter: %{agent->} for purpose: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Failed to register kqueue filter"),
	dup23,
]));

var msg377 = msg("PWC_KQUEUE_REGISTER_FILTER", part399);

var part400 = match("MESSAGE#373:PWC_LOCKFILE_BAD_FORMAT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PID lock file has bad format: %{agent}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PID lock file has bad format"),
	dup23,
]));

var msg378 = msg("PWC_LOCKFILE_BAD_FORMAT", part400);

var part401 = match("MESSAGE#374:PWC_LOCKFILE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PID lock file had error: %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PID lock file error"),
	dup23,
]));

var msg379 = msg("PWC_LOCKFILE_ERROR", part401);

var part402 = match("MESSAGE#375:PWC_LOCKFILE_MISSING", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PID lock file not found: %{agent}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PID lock file not found"),
	dup23,
]));

var msg380 = msg("PWC_LOCKFILE_MISSING", part402);

var part403 = match("MESSAGE#376:PWC_LOCKFILE_NOT_LOCKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PID lock file not locked: %{agent}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PID lock file not locked"),
	dup23,
]));

var msg381 = msg("PWC_LOCKFILE_NOT_LOCKED", part403);

var part404 = match("MESSAGE#377:PWC_NO_PROCESS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No process specified", processor_chain([
	dup30,
	dup22,
	setc("event_description","No process specified for PWC"),
	dup23,
]));

var msg382 = msg("PWC_NO_PROCESS", part404);

var part405 = match("MESSAGE#378:PWC_PROCESS_EXIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pwc process %{agent->} child %{child_pid->} exited with status %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","pwc process exited with status"),
	dup23,
]));

var msg383 = msg("PWC_PROCESS_EXIT", part405);

var part406 = match("MESSAGE#379:PWC_PROCESS_FORCED_HOLD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} forcing hold down of child %{child_pid->} until signal", processor_chain([
	dup21,
	dup22,
	setc("event_description","Process forcing hold down of child until signalled"),
	dup23,
]));

var msg384 = msg("PWC_PROCESS_FORCED_HOLD", part406);

var part407 = match("MESSAGE#380:PWC_PROCESS_HOLD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} holding down child %{child_pid->} until signal", processor_chain([
	dup21,
	dup22,
	setc("event_description","Process holding down child until signalled"),
	dup23,
]));

var msg385 = msg("PWC_PROCESS_HOLD", part407);

var part408 = match("MESSAGE#381:PWC_PROCESS_HOLD_SKIPPED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} will not down child %{child_pid->} because of %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Process not holding down child"),
	dup23,
]));

var msg386 = msg("PWC_PROCESS_HOLD_SKIPPED", part408);

var part409 = match("MESSAGE#382:PWC_PROCESS_OPEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to create child process with pidpopen: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Failed to create child process with pidpopen"),
	dup23,
]));

var msg387 = msg("PWC_PROCESS_OPEN", part409);

var part410 = match("MESSAGE#383:PWC_PROCESS_TIMED_HOLD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process %{agent->} holding down child %{child_pid->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Process holding down child"),
	dup23,
]));

var msg388 = msg("PWC_PROCESS_TIMED_HOLD", part410);

var part411 = match("MESSAGE#384:PWC_PROCESS_TIMEOUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child timed out %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Child process timed out"),
	dup23,
]));

var msg389 = msg("PWC_PROCESS_TIMEOUT", part411);

var part412 = match("MESSAGE#385:PWC_SIGNAL_INIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: signal(%{agent}) failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","signal failure"),
	dup23,
]));

var msg390 = msg("PWC_SIGNAL_INIT", part412);

var part413 = match("MESSAGE#386:PWC_SOCKET_CONNECT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to connect socket to %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to connect socket to service"),
	dup23,
]));

var msg391 = msg("PWC_SOCKET_CONNECT", part413);

var part414 = match("MESSAGE#387:PWC_SOCKET_CREATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Failed to create socket: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Failed to create socket"),
	dup23,
]));

var msg392 = msg("PWC_SOCKET_CREATE", part414);

var part415 = match("MESSAGE#388:PWC_SOCKET_OPTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to set socket option %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to set socket option"),
	dup23,
]));

var msg393 = msg("PWC_SOCKET_OPTION", part415);

var part416 = match("MESSAGE#389:PWC_STDOUT_WRITE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Write to stdout failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Write to stdout failed"),
	dup23,
]));

var msg394 = msg("PWC_STDOUT_WRITE", part416);

var part417 = match("MESSAGE#390:PWC_SYSTEM_CALL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PWC SYSTEM CALL"),
	dup23,
]));

var msg395 = msg("PWC_SYSTEM_CALL", part417);

var part418 = match("MESSAGE#391:PWC_UNKNOWN_KILL_OPTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unknown kill option [%{agent}]", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unknown kill option"),
	dup23,
]));

var msg396 = msg("PWC_UNKNOWN_KILL_OPTION", part418);

var part419 = match("MESSAGE#392:RMOPD_ADDRESS_MULTICAST_INVALID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Multicast address is not allowed", processor_chain([
	dup30,
	dup22,
	setc("event_description","Multicast address not allowed"),
	dup23,
]));

var msg397 = msg("RMOPD_ADDRESS_MULTICAST_INVALID", part419);

var part420 = match("MESSAGE#393:RMOPD_ADDRESS_SOURCE_INVALID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Source address invalid: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RMOPD ADDRESS SOURCE INVALID"),
	dup23,
]));

var msg398 = msg("RMOPD_ADDRESS_SOURCE_INVALID", part420);

var part421 = match("MESSAGE#394:RMOPD_ADDRESS_STRING_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to convert numeric address to string: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to convert numeric address to string"),
	dup23,
]));

var msg399 = msg("RMOPD_ADDRESS_STRING_FAILURE", part421);

var part422 = match("MESSAGE#395:RMOPD_ADDRESS_TARGET_INVALID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: rmop_util_set_address status message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","rmop_util_set_address status message invalid"),
	dup23,
]));

var msg400 = msg("RMOPD_ADDRESS_TARGET_INVALID", part422);

var msg401 = msg("RMOPD_DUPLICATE", dup141);

var part423 = match("MESSAGE#397:RMOPD_ICMP_ADDRESS_TYPE_UNSUPPORTED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Only IPv4 source address is supported", processor_chain([
	dup30,
	dup22,
	setc("event_description","Only IPv4 source address is supported"),
	dup23,
]));

var msg402 = msg("RMOPD_ICMP_ADDRESS_TYPE_UNSUPPORTED", part423);

var part424 = match("MESSAGE#398:RMOPD_ICMP_SENDMSG_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{fld1}: No route to host", processor_chain([
	dup30,
	dup22,
	setc("event_description","No route to host"),
	dup23,
]));

var msg403 = msg("RMOPD_ICMP_SENDMSG_FAILURE", part424);

var part425 = match("MESSAGE#399:RMOPD_IFINDEX_NOT_ACTIVE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifindex: %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IFINDEX NOT ACTIVE"),
	dup23,
]));

var msg404 = msg("RMOPD_IFINDEX_NOT_ACTIVE", part425);

var part426 = match("MESSAGE#400:RMOPD_IFINDEX_NO_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No information for %{interface}, message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IFINDEX NO INFO"),
	dup23,
]));

var msg405 = msg("RMOPD_IFINDEX_NO_INFO", part426);

var part427 = match("MESSAGE#401:RMOPD_IFNAME_NOT_ACTIVE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifname: %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RMOPD IFNAME NOT ACTIVE"),
	dup23,
]));

var msg406 = msg("RMOPD_IFNAME_NOT_ACTIVE", part427);

var part428 = match("MESSAGE#402:RMOPD_IFNAME_NO_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No information for %{interface}, message: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IFNAME NO INFO"),
	dup23,
]));

var msg407 = msg("RMOPD_IFNAME_NO_INFO", part428);

var part429 = match("MESSAGE#403:RMOPD_NOT_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","RMOPD Must be run as root"),
	dup23,
]));

var msg408 = msg("RMOPD_NOT_ROOT", part429);

var part430 = match("MESSAGE#404:RMOPD_ROUTING_INSTANCE_NO_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No information for routing instance %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","No information for routing instance"),
	dup23,
]));

var msg409 = msg("RMOPD_ROUTING_INSTANCE_NO_INFO", part430);

var part431 = match("MESSAGE#405:RMOPD_TRACEROUTE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TRACEROUTE ERROR"),
	dup23,
]));

var msg410 = msg("RMOPD_TRACEROUTE_ERROR", part431);

var part432 = match("MESSAGE#406:RMOPD_usage", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}: %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RMOPD usage"),
	dup23,
]));

var msg411 = msg("RMOPD_usage", part432);

var part433 = match("MESSAGE#407:RPD_ABORT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} version built by builder on %{dclass_counter1}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD ABORT"),
	dup23,
]));

var msg412 = msg("RPD_ABORT", part433);

var part434 = match("MESSAGE#408:RPD_ACTIVE_TERMINATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Exiting with active tasks: %{agent}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD exiting with active tasks"),
	dup23,
]));

var msg413 = msg("RPD_ACTIVE_TERMINATE", part434);

var part435 = match("MESSAGE#409:RPD_ASSERT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Assertion failed %{resultcode}: file \"%{filename}\", line %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD Assertion failed"),
	dup23,
]));

var msg414 = msg("RPD_ASSERT", part435);

var part436 = match("MESSAGE#410:RPD_ASSERT_SOFT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Soft assertion failed %{resultcode}: file \"%{filename}\", line %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD Soft assertion failed"),
	dup23,
]));

var msg415 = msg("RPD_ASSERT_SOFT", part436);

var part437 = match("MESSAGE#411:RPD_EXIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} version built by builder on %{dclass_counter1}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RPD EXIT"),
	dup23,
]));

var msg416 = msg("RPD_EXIT", part437);

var msg417 = msg("RPD_IFL_INDEXCOLLISION", dup147);

var msg418 = msg("RPD_IFL_NAMECOLLISION", dup147);

var part438 = match("MESSAGE#414:RPD_ISIS_ADJDOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: IS-IS lost %{dclass_counter1->} adjacency to %{dclass_counter2->} on %{interface}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IS-IS lost adjacency"),
	dup23,
]));

var msg419 = msg("RPD_ISIS_ADJDOWN", part438);

var part439 = match("MESSAGE#415:RPD_ISIS_ADJUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: IS-IS new %{dclass_counter1->} adjacency to %{dclass_counter2->} %{interface}", processor_chain([
	dup21,
	dup22,
	setc("event_description","IS-IS new adjacency"),
	dup23,
]));

var msg420 = msg("RPD_ISIS_ADJUP", part439);

var part440 = match("MESSAGE#416:RPD_ISIS_ADJUPNOIP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: IS-IS new %{dclass_counter1->} adjacency to %{dclass_counter2->} %{interface->} without an address", processor_chain([
	dup30,
	dup22,
	setc("event_description","IS-IS new adjacency without an address"),
	dup23,
]));

var msg421 = msg("RPD_ISIS_ADJUPNOIP", part440);

var part441 = match("MESSAGE#417:RPD_ISIS_LSPCKSUM", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: IS-IS %{dclass_counter1->} LSP checksum error, interface %{interface}, LSP id %{id}, sequence %{dclass_counter2}, checksum %{resultcode}, lifetime %{fld2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","IS-IS LSP checksum error on iterface"),
	dup23,
]));

var msg422 = msg("RPD_ISIS_LSPCKSUM", part441);

var part442 = match("MESSAGE#418:RPD_ISIS_OVERLOAD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: IS-IS database overload", processor_chain([
	dup30,
	dup22,
	setc("event_description","IS-IS database overload"),
	dup23,
]));

var msg423 = msg("RPD_ISIS_OVERLOAD", part442);

var part443 = match("MESSAGE#419:RPD_KRT_AFUNSUPRT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{resultcode}: received %{agent->} message with unsupported address family %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","message with unsupported address family received"),
	dup23,
]));

var msg424 = msg("RPD_KRT_AFUNSUPRT", part443);

var part444 = match("MESSAGE#420:RPD_KRT_CCC_IFL_MODIFY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}, error", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD KRT CCC IFL MODIFY"),
	dup23,
]));

var msg425 = msg("RPD_KRT_CCC_IFL_MODIFY", part444);

var part445 = match("MESSAGE#421:RPD_KRT_DELETED_RTT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: received deleted routing table from the kernel for family %{dclass_counter1->} table ID %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","received deleted routing table from kernel"),
	dup23,
]));

var msg426 = msg("RPD_KRT_DELETED_RTT", part445);

var part446 = match("MESSAGE#422:RPD_KRT_IFA_GENERATION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifa generation mismatch -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ifa generation mismatch"),
	dup23,
]));

var msg427 = msg("RPD_KRT_IFA_GENERATION", part446);

var part447 = match("MESSAGE#423:RPD_KRT_IFDCHANGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} CHANGE for ifd %{interface->} failed, error \"%{result}\"", processor_chain([
	dup30,
	dup22,
	setc("event_description","CHANGE for ifd failed"),
	dup23,
]));

var msg428 = msg("RPD_KRT_IFDCHANGE", part447);

var part448 = match("MESSAGE#424:RPD_KRT_IFDEST_GET", "nwparser.payload", "%{process}[%{process_id}]: %{event_type->} SERVICE: %{service->} for ifd %{interface->} failed, error \"%{result}\"", processor_chain([
	dup30,
	dup22,
	setc("event_description","GET SERVICE failure on interface"),
	dup23,
]));

var msg429 = msg("RPD_KRT_IFDEST_GET", part448);

var part449 = match("MESSAGE#425:RPD_KRT_IFDGET", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} GET index for ifd interface failed, error \"%{result}\"", processor_chain([
	dup30,
	dup22,
	setc("event_description","GET index for ifd interface failed"),
	dup23,
]));

var msg430 = msg("RPD_KRT_IFDGET", part449);

var part450 = match("MESSAGE#426:RPD_KRT_IFD_GENERATION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifd %{dclass_counter1->} generation mismatch -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ifd generation mismatch"),
	dup23,
]));

var msg431 = msg("RPD_KRT_IFD_GENERATION", part450);

var part451 = match("MESSAGE#427:RPD_KRT_IFL_CELL_RELAY_MODE_INVALID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifl : %{agent}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","KRT IFL CELL RELAY MODE INVALID"),
	dup23,
]));

var msg432 = msg("RPD_KRT_IFL_CELL_RELAY_MODE_INVALID", part451);

var part452 = match("MESSAGE#428:RPD_KRT_IFL_CELL_RELAY_MODE_UNSPECIFIED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifl : %{agent}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","KRT IFL CELL RELAY MODE UNSPECIFIED"),
	dup23,
]));

var msg433 = msg("RPD_KRT_IFL_CELL_RELAY_MODE_UNSPECIFIED", part452);

var part453 = match("MESSAGE#429:RPD_KRT_IFL_GENERATION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifl %{interface->} generation mismatch -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ifl generation mismatch"),
	dup23,
]));

var msg434 = msg("RPD_KRT_IFL_GENERATION", part453);

var part454 = match("MESSAGE#430:RPD_KRT_KERNEL_BAD_ROUTE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: lost %{interface->} %{dclass_counter1->} for route %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","lost interface for route"),
	dup23,
]));

var msg435 = msg("RPD_KRT_KERNEL_BAD_ROUTE", part454);

var part455 = match("MESSAGE#431:RPD_KRT_NEXTHOP_OVERFLOW", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: number of next hops (%{dclass_counter1}) exceeded the maximum allowed (%{dclass_counter2}) -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","number of next hops exceeded the maximum"),
	dup23,
]));

var msg436 = msg("RPD_KRT_NEXTHOP_OVERFLOW", part455);

var part456 = match("MESSAGE#432:RPD_KRT_NOIFD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No device %{dclass_counter1->} for interface %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","No device for interface"),
	dup23,
]));

var msg437 = msg("RPD_KRT_NOIFD", part456);

var part457 = match("MESSAGE#433:RPD_KRT_UNKNOWN_RTT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: received routing table message for unknown table with kernel ID %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","received routing table message for unknown table"),
	dup23,
]));

var msg438 = msg("RPD_KRT_UNKNOWN_RTT", part457);

var part458 = match("MESSAGE#434:RPD_KRT_VERSION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Routing socket version mismatch (%{info}) -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Routing socket version mismatch"),
	dup23,
]));

var msg439 = msg("RPD_KRT_VERSION", part458);

var part459 = match("MESSAGE#435:RPD_KRT_VERSIONNONE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Routing socket message type %{agent}'s version is not supported by kernel, %{info->} -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Routing socket message type not supported by kernel"),
	dup23,
]));

var msg440 = msg("RPD_KRT_VERSIONNONE", part459);

var part460 = match("MESSAGE#436:RPD_KRT_VERSIONOLD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Routing socket message type %{agent}'s version is older than expected (%{info}) -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Routing socket message type version is older than expected"),
	dup23,
]));

var msg441 = msg("RPD_KRT_VERSIONOLD", part460);

var part461 = match("MESSAGE#437:RPD_LDP_INTF_BLOCKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Duplicate session ID detected from %{daddr}, interface %{interface}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Duplicate session ID detected"),
	dup23,
]));

var msg442 = msg("RPD_LDP_INTF_BLOCKED", part461);

var part462 = match("MESSAGE#438:RPD_LDP_INTF_UNBLOCKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: LDP interface %{interface->} is now %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","LDP interface now unblocked"),
	dup23,
]));

var msg443 = msg("RPD_LDP_INTF_UNBLOCKED", part462);

var part463 = match("MESSAGE#439:RPD_LDP_NBRDOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: LDP neighbor %{daddr->} (%{interface}) is %{result}", processor_chain([
	setc("eventcategory","1603030000"),
	dup22,
	setc("event_description","LDP neighbor down"),
	dup23,
]));

var msg444 = msg("RPD_LDP_NBRDOWN", part463);

var part464 = match("MESSAGE#440:RPD_LDP_NBRUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: LDP neighbor %{daddr->} (%{interface}) is %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","LDP neighbor up"),
	dup23,
]));

var msg445 = msg("RPD_LDP_NBRUP", part464);

var part465 = match("MESSAGE#441:RPD_LDP_SESSIONDOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: LDP session %{daddr->} is down, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","LDP session down"),
	dup23,
]));

var msg446 = msg("RPD_LDP_SESSIONDOWN", part465);

var part466 = match("MESSAGE#442:RPD_LDP_SESSIONUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: LDP session %{daddr->} is up", processor_chain([
	dup21,
	dup22,
	setc("event_description","LDP session up"),
	dup23,
]));

var msg447 = msg("RPD_LDP_SESSIONUP", part466);

var part467 = match("MESSAGE#443:RPD_LOCK_FLOCKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to obtain a lock on %{agent}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to obtain a lock"),
	dup23,
]));

var msg448 = msg("RPD_LOCK_FLOCKED", part467);

var part468 = match("MESSAGE#444:RPD_LOCK_LOCKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to obtain a lock on %{agent}, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to obtain service lock"),
	dup23,
]));

var msg449 = msg("RPD_LOCK_LOCKED", part468);

var part469 = match("MESSAGE#445:RPD_MPLS_LSP_CHANGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MPLS LSP %{interface->} %{result->} Route %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","MPLS LSP CHANGE"),
	dup23,
]));

var msg450 = msg("RPD_MPLS_LSP_CHANGE", part469);

var part470 = match("MESSAGE#446:RPD_MPLS_LSP_DOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MPLS LSP %{interface->} %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MPLS LSP DOWN"),
	dup23,
]));

var msg451 = msg("RPD_MPLS_LSP_DOWN", part470);

var part471 = match("MESSAGE#447:RPD_MPLS_LSP_SWITCH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MPLS LSP %{interface->} %{result}, Route %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","MPLS LSP SWITCH"),
	dup23,
]));

var msg452 = msg("RPD_MPLS_LSP_SWITCH", part471);

var part472 = match("MESSAGE#448:RPD_MPLS_LSP_UP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MPLS LSP %{interface->} %{result->} Route %{info}", processor_chain([
	dup21,
	dup22,
	setc("event_description","MPLS LSP UP"),
	dup23,
]));

var msg453 = msg("RPD_MPLS_LSP_UP", part472);

var part473 = match("MESSAGE#449:RPD_MSDP_PEER_DOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MSDP peer %{group->} %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","MSDP PEER DOWN"),
	dup23,
]));

var msg454 = msg("RPD_MSDP_PEER_DOWN", part473);

var part474 = match("MESSAGE#450:RPD_MSDP_PEER_UP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: MSDP peer %{group->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","MSDP PEER UP"),
	dup23,
]));

var msg455 = msg("RPD_MSDP_PEER_UP", part474);

var part475 = match("MESSAGE#451:RPD_OSPF_NBRDOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: OSPF neighbor %{daddr->} (%{interface}) %{disposition->} due to %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","OSPF neighbor down"),
	dup23,
]));

var msg456 = msg("RPD_OSPF_NBRDOWN", part475);

var part476 = match("MESSAGE#452:RPD_OSPF_NBRUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: OSPF neighbor %{daddr->} (%{interface}) %{disposition->} due to %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","OSPF neighbor up"),
	dup23,
]));

var msg457 = msg("RPD_OSPF_NBRUP", part476);

var part477 = match("MESSAGE#453:RPD_OS_MEMHIGH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Using %{dclass_counter1->} KB of memory, %{info}", processor_chain([
	dup51,
	dup22,
	setc("event_description","OS MEMHIGH"),
	dup23,
]));

var msg458 = msg("RPD_OS_MEMHIGH", part477);

var part478 = match("MESSAGE#454:RPD_PIM_NBRDOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PIM neighbor %{daddr->} timeout interface %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","PIM neighbor down"),
	setc("result","timeout"),
	dup23,
]));

var msg459 = msg("RPD_PIM_NBRDOWN", part478);

var part479 = match("MESSAGE#455:RPD_PIM_NBRUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: PIM new neighbor %{daddr->} interface %{interface}", processor_chain([
	dup21,
	dup22,
	setc("event_description","PIM neighbor up"),
	dup23,
]));

var msg460 = msg("RPD_PIM_NBRUP", part479);

var part480 = match("MESSAGE#456:RPD_RDISC_CKSUM", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Bad checksum for router solicitation from %{saddr->} to %{daddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Bad checksum for router solicitation"),
	dup23,
]));

var msg461 = msg("RPD_RDISC_CKSUM", part480);

var part481 = match("MESSAGE#457:RPD_RDISC_NOMULTI", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Ignoring interface %{dclass_counter1->} on %{interface->} -- %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Ignoring interface"),
	dup23,
]));

var msg462 = msg("RPD_RDISC_NOMULTI", part481);

var part482 = match("MESSAGE#458:RPD_RDISC_NORECVIF", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to locate interface for router solicitation from %{saddr->} to %{daddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to locate interface for router"),
	dup23,
]));

var msg463 = msg("RPD_RDISC_NORECVIF", part482);

var part483 = match("MESSAGE#459:RPD_RDISC_SOLICITADDR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Expected multicast (%{dclass_counter1}) for router solicitation from %{saddr->} to %{daddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Expected multicast for router solicitation"),
	dup23,
]));

var msg464 = msg("RPD_RDISC_SOLICITADDR", part483);

var part484 = match("MESSAGE#460:RPD_RDISC_SOLICITICMP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Nonzero ICMP code (%{resultcode}) for router solicitation from %{saddr->} to %{daddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Nonzero ICMP code for router solicitation"),
	dup23,
]));

var msg465 = msg("RPD_RDISC_SOLICITICMP", part484);

var part485 = match("MESSAGE#461:RPD_RDISC_SOLICITLEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Insufficient length (%{dclass_counter1}) for router solicitation from %{saddr->} to %{daddr}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Insufficient length for router solicitation"),
	dup23,
]));

var msg466 = msg("RPD_RDISC_SOLICITLEN", part485);

var part486 = match("MESSAGE#462:RPD_RIP_AUTH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Update with invalid authentication from %{saddr->} (%{interface})", processor_chain([
	dup30,
	dup22,
	setc("event_description","RIP update with invalid authentication"),
	dup23,
]));

var msg467 = msg("RPD_RIP_AUTH", part486);

var part487 = match("MESSAGE#463:RPD_RIP_JOIN_BROADCAST", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to get broadcast address %{interface}; %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RIP - unable to get broadcast address"),
	dup23,
]));

var msg468 = msg("RPD_RIP_JOIN_BROADCAST", part487);

var part488 = match("MESSAGE#464:RPD_RIP_JOIN_MULTICAST", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to join multicast group %{interface}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RIP - Unable to join multicast group"),
	dup23,
]));

var msg469 = msg("RPD_RIP_JOIN_MULTICAST", part488);

var part489 = match("MESSAGE#465:RPD_RT_IFUP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: UP route for interface %{interface->} index %{dclass_counter1->} %{saddr}/%{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RIP interface up"),
	dup23,
]));

var msg470 = msg("RPD_RT_IFUP", part489);

var msg471 = msg("RPD_SCHED_CALLBACK_LONGRUNTIME", dup148);

var part490 = match("MESSAGE#467:RPD_SCHED_CUMULATIVE_LONGRUNTIME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: excessive runtime (%{result}) after action of module", processor_chain([
	dup30,
	dup22,
	setc("event_description","excessive runtime after action of module"),
	dup23,
]));

var msg472 = msg("RPD_SCHED_CUMULATIVE_LONGRUNTIME", part490);

var msg473 = msg("RPD_SCHED_MODULE_LONGRUNTIME", dup148);

var part491 = match("MESSAGE#469:RPD_SCHED_TASK_LONGRUNTIME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} ran for %{dclass_counter1}(%{dclass_counter2})", processor_chain([
	dup30,
	dup22,
	setc("event_description","task extended runtime"),
	dup23,
]));

var msg474 = msg("RPD_SCHED_TASK_LONGRUNTIME", part491);

var part492 = match("MESSAGE#470:RPD_SIGNAL_TERMINATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} termination signal received", processor_chain([
	dup30,
	dup22,
	setc("event_description","termination signal received for service"),
	dup23,
]));

var msg475 = msg("RPD_SIGNAL_TERMINATE", part492);

var part493 = match("MESSAGE#471:RPD_START", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Start %{dclass_counter1->} version version built %{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","version built"),
	dup23,
]));

var msg476 = msg("RPD_START", part493);

var part494 = match("MESSAGE#472:RPD_SYSTEM", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: detail: %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","system command"),
	dup23,
]));

var msg477 = msg("RPD_SYSTEM", part494);

var part495 = match("MESSAGE#473:RPD_TASK_BEGIN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Commencing routing updates, version %{dclass_counter1}, built %{dclass_counter2->} by builder", processor_chain([
	dup21,
	dup22,
	setc("event_description","Commencing routing updates"),
	dup23,
]));

var msg478 = msg("RPD_TASK_BEGIN", part495);

var part496 = match("MESSAGE#474:RPD_TASK_CHILDKILLED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{dclass_counter2->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","task killed by signal"),
	dup23,
]));

var msg479 = msg("RPD_TASK_CHILDKILLED", part496);

var part497 = match("MESSAGE#475:RPD_TASK_CHILDSTOPPED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{dclass_counter2->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","task stopped by signal"),
	dup23,
]));

var msg480 = msg("RPD_TASK_CHILDSTOPPED", part497);

var part498 = match("MESSAGE#476:RPD_TASK_FORK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to fork task: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to fork task"),
	dup23,
]));

var msg481 = msg("RPD_TASK_FORK", part498);

var part499 = match("MESSAGE#477:RPD_TASK_GETWD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: getwd: %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RPD TASK GETWD"),
	dup23,
]));

var msg482 = msg("RPD_TASK_GETWD", part499);

var part500 = match("MESSAGE#478:RPD_TASK_NOREINIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reinitialization not possible", processor_chain([
	dup30,
	dup22,
	setc("event_description","Reinitialization not possible"),
	dup23,
]));

var msg483 = msg("RPD_TASK_NOREINIT", part500);

var part501 = match("MESSAGE#479:RPD_TASK_PIDCLOSED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to close and remove %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to close and remove task"),
	dup23,
]));

var msg484 = msg("RPD_TASK_PIDCLOSED", part501);

var part502 = match("MESSAGE#480:RPD_TASK_PIDFLOCK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: flock(%{agent}, %{action}): %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RPD TASK PIDFLOCK"),
	dup23,
]));

var msg485 = msg("RPD_TASK_PIDFLOCK", part502);

var part503 = match("MESSAGE#481:RPD_TASK_PIDWRITE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to write %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to write"),
	dup23,
]));

var msg486 = msg("RPD_TASK_PIDWRITE", part503);

var msg487 = msg("RPD_TASK_REINIT", dup149);

var part504 = match("MESSAGE#483:RPD_TASK_SIGNALIGNORE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: sigaction(%{result}): %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","ignoring task signal"),
	dup23,
]));

var msg488 = msg("RPD_TASK_SIGNALIGNORE", part504);

var part505 = match("MESSAGE#484:RT_COS", "nwparser.payload", "%{process}: %{event_type}: COS IPC op %{dclass_counter1->} (%{agent}) failed, err %{resultcode->} (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","COS IPC op failed"),
	dup23,
]));

var msg489 = msg("RT_COS", part505);

var part506 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/2", "nwparser.p0", "%{fld5}\" nat-source-address=\"%{stransaddr}\" nat-source-port=\"%{stransport}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\"%{p0}");

var part507 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/4", "nwparser.p0", "%{}src-nat-rule-name=\"%{fld10}\" dst-nat-rule-%{p0}");

var part508 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/5_0", "nwparser.p0", "type=%{fld21->} dst-nat-rule-name=\"%{p0}");

var select39 = linear_select([
	part508,
	dup91,
]);

var part509 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/6", "nwparser.p0", "\"%{fld11->} protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{fld13}\" username=\"%{username}\" roles=\"%{fld15}\" packet-incoming-interface=\"%{dinterface}\"%{p0}");

var part510 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/7_0", "nwparser.p0", " application=\"%{fld6}\" nested-application=\"%{fld7}\" encrypted=%{fld8->} %{p0}");

var select40 = linear_select([
	part510,
	dup45,
]);

var all22 = all_match({
	processors: [
		dup87,
		dup150,
		part506,
		dup151,
		part507,
		select39,
		part509,
		select40,
		dup92,
	],
	on_success: processor_chain([
		dup28,
		dup53,
		dup54,
		dup22,
		dup52,
	]),
});

var msg490 = msg("RT_FLOW_SESSION_CREATE:02", all22);

var part511 = match("MESSAGE#486:RT_FLOW_SESSION_CREATE/1_0", "nwparser.p0", " service-name=\"%{service}\" nat-source-address=\"%{stransaddr}\" nat-source-port=\"%{stransport}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" src-nat-rule-type=\"%{fld20}\" src-nat-rule-name=\"%{rulename}\" dst-nat-rule-type=\"%{fld10}\" dst-nat-rule-name=\"%{rule_template}\"%{p0}");

var select41 = linear_select([
	part511,
	dup45,
]);

var part512 = match("MESSAGE#486:RT_FLOW_SESSION_CREATE/2", "nwparser.p0", "%{}protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\"%{p0}");

var part513 = match("MESSAGE#486:RT_FLOW_SESSION_CREATE/3_0", "nwparser.p0", " source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{sessionid}\" username=\"%{username}\" roles=\"%{fld50}\" packet-incoming-interface=\"%{dinterface}\" application=\"%{application}\" nested-application=\"%{fld7}\" encrypted=\"%{fld8}\"%{p0}");

var select42 = linear_select([
	part513,
	dup45,
]);

var all23 = all_match({
	processors: [
		dup87,
		select41,
		part512,
		select42,
		dup92,
	],
	on_success: processor_chain([
		dup28,
		dup53,
		dup54,
		dup22,
		dup52,
	]),
});

var msg491 = msg("RT_FLOW_SESSION_CREATE", all23);

var part514 = match("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/0_0", "nwparser.payload", "%{process}: %{event_type}: session created %{p0}");

var part515 = match("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/0_1", "nwparser.payload", "%{event_type}: session created %{p0}");

var select43 = linear_select([
	part514,
	part515,
]);

var part516 = match("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/1", "nwparser.p0", "%{saddr}/%{sport}->%{daddr}/%{dport->} %{fld20->} %{hostip}/%{network_port}->%{dtransaddr}/%{dtransport->} %{p0}");

var part517 = match("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/2_0", "nwparser.p0", "%{rulename->} %{rule_template->} %{fld12->} %{fld13->} %{fld14->} %{policyname->} %{src_zone->} %{dst_zone->} %{sessionid->} %{username}(%{fld10}) %{interface->} %{protocol->} %{fld15->} UNKNOWN UNKNOWN");

var part518 = match("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/2_1", "nwparser.p0", "%{rulename->} %{rule_template->} %{fld12->} %{fld13->} %{fld14->} %{policyname->} %{src_zone->} %{dst_zone->} %{sessionid->} %{username}(%{fld10}) %{interface->} %{fld15}");

var part519 = match_copy("MESSAGE#487:RT_FLOW_SESSION_CREATE:01/2_2", "nwparser.p0", "info");

var select44 = linear_select([
	part517,
	part518,
	part519,
]);

var all24 = all_match({
	processors: [
		select43,
		part516,
		select44,
	],
	on_success: processor_chain([
		dup28,
		dup53,
		dup54,
		dup22,
		setc("event_description","session created"),
		dup23,
	]),
});

var msg492 = msg("RT_FLOW_SESSION_CREATE:01", all24);

var select45 = linear_select([
	msg490,
	msg491,
	msg492,
]);

var part520 = match("MESSAGE#488:RT_FLOW_SESSION_DENY:02/2", "nwparser.p0", "%{fld5}\" protocol-id=\"%{protocol}\" icmp-type=\"%{obj_type}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" application=\"%{fld6}\" nested-application=\"%{fld7}\" username=\"%{username}\" roles=\"%{user_role}\" packet-incoming-interface=\"%{dinterface}\"%{p0}");

var part521 = match("MESSAGE#488:RT_FLOW_SESSION_DENY:02/3_0", "nwparser.p0", " encrypted=\"%{fld16}\" reason=\"%{result}\" src-vrf-grp=\"%{fld99}\" dst-vrf-grp=\"%{fld98}\"%{p0}");

var part522 = match("MESSAGE#488:RT_FLOW_SESSION_DENY:02/3_1", "nwparser.p0", " encrypted=%{fld16->} reason=\"%{result}\"%{p0}");

var select46 = linear_select([
	part521,
	part522,
	dup45,
]);

var all25 = all_match({
	processors: [
		dup87,
		dup150,
		part520,
		select46,
		dup92,
	],
	on_success: processor_chain([
		dup93,
		dup53,
		dup94,
		dup22,
		dup52,
	]),
});

var msg493 = msg("RT_FLOW_SESSION_DENY:02", all25);

var part523 = match("MESSAGE#489:RT_FLOW_SESSION_DENY", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-id=\"%{protocol}\" icmp-type=\"%{obj_type}\" policy-name=\"%{policyname}\"]", processor_chain([
	dup93,
	dup53,
	dup94,
	dup22,
	dup52,
]));

var msg494 = msg("RT_FLOW_SESSION_DENY", part523);

var part524 = match("MESSAGE#490:RT_FLOW_SESSION_DENY:03/1", "nwparser.p0", "%{saddr}/%{sport}->%{daddr}/%{dport->} %{fld20->} %{fld1->} %{result->} %{src_zone->} %{dst_zone->} HTTP %{info}");

var all26 = all_match({
	processors: [
		dup152,
		part524,
	],
	on_success: processor_chain([
		dup27,
		dup53,
		dup94,
		dup22,
		dup97,
		dup23,
	]),
});

var msg495 = msg("RT_FLOW_SESSION_DENY:03", all26);

var part525 = match("MESSAGE#491:RT_FLOW_SESSION_DENY:01/1", "nwparser.p0", "%{saddr}/%{sport}->%{daddr}/%{dport->} %{fld20->} %{fld1->} %{result->} %{src_zone->} %{dst_zone}");

var all27 = all_match({
	processors: [
		dup152,
		part525,
	],
	on_success: processor_chain([
		dup27,
		dup53,
		dup94,
		dup22,
		dup97,
		dup23,
	]),
});

var msg496 = msg("RT_FLOW_SESSION_DENY:01", all27);

var select47 = linear_select([
	msg493,
	msg494,
	msg495,
	msg496,
]);

var select48 = linear_select([
	dup103,
	dup45,
]);

var all28 = all_match({
	processors: [
		dup98,
		dup150,
		dup99,
		dup151,
		dup100,
		dup153,
		dup102,
		select48,
		dup92,
	],
	on_success: processor_chain([
		dup27,
		dup53,
		dup55,
		dup104,
		dup22,
		dup52,
	]),
});

var msg497 = msg("RT_FLOW_SESSION_CLOSE:01", all28);

var part526 = match("MESSAGE#493:RT_FLOW_SESSION_CLOSE", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} reason=\"%{result}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" inbound-packets=\"%{packets}\" inbound-bytes=\"%{rbytes}\" outbound-packets=\"%{dclass_counter1}\" outbound-bytes=\"%{sbytes}\" elapsed-time=\"%{duration}\"]", processor_chain([
	dup27,
	dup53,
	dup55,
	dup22,
	dup52,
]));

var msg498 = msg("RT_FLOW_SESSION_CLOSE", part526);

var part527 = match("MESSAGE#494:RT_FLOW_SESSION_CLOSE:02/0_0", "nwparser.payload", "%{process}: %{event_type}: session closed %{p0}");

var part528 = match("MESSAGE#494:RT_FLOW_SESSION_CLOSE:02/0_1", "nwparser.payload", "%{event_type}: session closed %{p0}");

var select49 = linear_select([
	part527,
	part528,
]);

var part529 = match("MESSAGE#494:RT_FLOW_SESSION_CLOSE:02/1", "nwparser.p0", "%{result}: %{saddr}/%{sport}->%{daddr}/%{dport->} %{fld20->} %{hostip}/%{network_port}->%{dtransaddr}/%{dtransport->} %{info}");

var all29 = all_match({
	processors: [
		select49,
		part529,
	],
	on_success: processor_chain([
		dup27,
		dup53,
		dup55,
		dup22,
		setc("event_description","session closed"),
		dup23,
	]),
});

var msg499 = msg("RT_FLOW_SESSION_CLOSE:02", all29);

var part530 = match("MESSAGE#495:RT_FLOW_SESSION_CLOSE:03/7_1", "nwparser.p0", " application=\"%{fld6}\" nested-application=\"%{fld7}\" username=\"%{username}\" roles=\"%{user_role}\" packet-incoming-interface=\"%{dinterface}\" %{p0}");

var select50 = linear_select([
	dup103,
	part530,
	dup45,
]);

var part531 = match("MESSAGE#495:RT_FLOW_SESSION_CLOSE:03/8", "nwparser.p0", "] session closed %{fld60}: %{fld51}/%{fld52}->%{fld53}/%{fld54->} %{fld55->} %{fld56}/%{fld57}->%{fld58}/%{fld59->} %{info}");

var all30 = all_match({
	processors: [
		dup98,
		dup150,
		dup99,
		dup151,
		dup100,
		dup153,
		dup102,
		select50,
		part531,
	],
	on_success: processor_chain([
		dup27,
		dup53,
		dup55,
		dup104,
		dup22,
		dup52,
		dup61,
	]),
});

var msg500 = msg("RT_FLOW_SESSION_CLOSE:03", all30);

var select51 = linear_select([
	msg497,
	msg498,
	msg499,
	msg500,
]);

var part532 = match("MESSAGE#496:RT_SCREEN_IP", "nwparser.payload", "%{process}: %{event_type}: Fragmented traffic! source:%{saddr}, destination: %{daddr}, protocol-id: %{protocol}, zone name: %{zone}, interface name: %{interface}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Fragmented traffic"),
	dup23,
]));

var msg501 = msg("RT_SCREEN_IP", part532);

var part533 = match("MESSAGE#497:RT_SCREEN_IP:01", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} attack-name=\"%{threat_name}\" source-address=\"%{saddr}\" destination-address=\"%{daddr}\" protocol-id=\"%{protocol}\" source-zone-name=\"%{src_zone}\" interface-name=\"%{interface}\" action=\"%{action}\"]", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg502 = msg("RT_SCREEN_IP:01", part533);

var select52 = linear_select([
	msg501,
	msg502,
]);

var msg503 = msg("RT_SCREEN_TCP", dup154);

var part534 = match("MESSAGE#499:RT_SCREEN_SESSION_LIMIT", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} attack-name=\"%{threat_name}\" message=\"%{info}\" ip-address=\"%{hostip}\" source-zone-name=\"%{src_zone}\" interface-name=\"%{interface}\" action=\"%{action}\"]", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg504 = msg("RT_SCREEN_SESSION_LIMIT", part534);

var msg505 = msg("RT_SCREEN_UDP", dup154);

var part535 = match("MESSAGE#501:SERVICED_CLIENT_CONNECT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: attempt to connect to interface failed with error: %{result}", processor_chain([
	dup27,
	dup22,
	setc("event_description","attempt to connect to interface failed"),
	dup23,
]));

var msg506 = msg("SERVICED_CLIENT_CONNECT", part535);

var part536 = match("MESSAGE#502:SERVICED_CLIENT_DISCONNECTED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: unexpected termination of connection to interface", processor_chain([
	dup27,
	dup22,
	setc("event_description","unexpected termination of connection"),
	dup23,
]));

var msg507 = msg("SERVICED_CLIENT_DISCONNECTED", part536);

var part537 = match("MESSAGE#503:SERVICED_CLIENT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: client interface connection failure: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","client interface connection failure"),
	dup23,
]));

var msg508 = msg("SERVICED_CLIENT_ERROR", part537);

var part538 = match("MESSAGE#504:SERVICED_COMMAND_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: remote command execution failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","remote command execution failed"),
	dup23,
]));

var msg509 = msg("SERVICED_COMMAND_FAILED", part538);

var part539 = match("MESSAGE#505:SERVICED_COMMIT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: client failed to commit configuration with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","client commit configuration failed"),
	dup23,
]));

var msg510 = msg("SERVICED_COMMIT_FAILED", part539);

var part540 = match("MESSAGE#506:SERVICED_CONFIGURATION_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: configuration process failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","configuration process failed"),
	dup23,
]));

var msg511 = msg("SERVICED_CONFIGURATION_FAILED", part540);

var part541 = match("MESSAGE#507:SERVICED_CONFIG_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SERVICED CONFIG ERROR"),
	dup23,
]));

var msg512 = msg("SERVICED_CONFIG_ERROR", part541);

var part542 = match("MESSAGE#508:SERVICED_CONFIG_FILE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{dclass_counter2->} failed to read path with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","service failed to read path"),
	dup23,
]));

var msg513 = msg("SERVICED_CONFIG_FILE", part542);

var part543 = match("MESSAGE#509:SERVICED_CONNECTION_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SERVICED CONNECTION ERROR"),
	dup23,
]));

var msg514 = msg("SERVICED_CONNECTION_ERROR", part543);

var part544 = match("MESSAGE#510:SERVICED_DISABLED_GGSN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: GGSN services disabled: object: %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","GGSN services disabled"),
	dup23,
]));

var msg515 = msg("SERVICED_DISABLED_GGSN", part544);

var msg516 = msg("SERVICED_DUPLICATE", dup141);

var part545 = match("MESSAGE#512:SERVICED_EVENT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: event function %{dclass_counter2->} failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","event function failed"),
	dup23,
]));

var msg517 = msg("SERVICED_EVENT_FAILED", part545);

var part546 = match("MESSAGE#513:SERVICED_INIT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: initialization failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","service initialization failed"),
	dup23,
]));

var msg518 = msg("SERVICED_INIT_FAILED", part546);

var part547 = match("MESSAGE#514:SERVICED_MALLOC_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: failed to allocate [%{dclass_counter2}] object [%{dclass_counter1->} bytes %{bytes}]: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","memory allocation failure"),
	dup23,
]));

var msg519 = msg("SERVICED_MALLOC_FAILURE", part547);

var part548 = match("MESSAGE#515:SERVICED_NETWORK_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{dclass_counter2->} had error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","NETWORK FAILURE"),
	dup23,
]));

var msg520 = msg("SERVICED_NETWORK_FAILURE", part548);

var part549 = match("MESSAGE#516:SERVICED_NOT_ROOT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Must be run as root", processor_chain([
	dup63,
	dup22,
	setc("event_description","SERVICED must be run as root"),
	dup23,
]));

var msg521 = msg("SERVICED_NOT_ROOT", part549);

var msg522 = msg("SERVICED_PID_FILE_LOCK", dup142);

var msg523 = msg("SERVICED_PID_FILE_UPDATE", dup143);

var part550 = match("MESSAGE#519:SERVICED_RTSOCK_SEQUENCE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: routing socket sequence error, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","routing socket sequence error"),
	dup23,
]));

var msg524 = msg("SERVICED_RTSOCK_SEQUENCE", part550);

var part551 = match("MESSAGE#520:SERVICED_SIGNAL_HANDLER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: set up of signal name handler failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","set up of signal name handler failed"),
	dup23,
]));

var msg525 = msg("SERVICED_SIGNAL_HANDLER", part551);

var part552 = match("MESSAGE#521:SERVICED_SOCKET_CREATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: socket create failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","socket create failed with error"),
	dup23,
]));

var msg526 = msg("SERVICED_SOCKET_CREATE", part552);

var part553 = match("MESSAGE#522:SERVICED_SOCKET_IO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: socket function %{dclass_counter2->} failed with error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","socket function failed"),
	dup23,
]));

var msg527 = msg("SERVICED_SOCKET_IO", part553);

var part554 = match("MESSAGE#523:SERVICED_SOCKET_OPTION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: unable to set socket option %{dclass_counter2}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","unable to set socket option"),
	dup23,
]));

var msg528 = msg("SERVICED_SOCKET_OPTION", part554);

var part555 = match("MESSAGE#524:SERVICED_STDLIB_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{dclass_counter2->} had error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","STDLIB FAILURE"),
	dup23,
]));

var msg529 = msg("SERVICED_STDLIB_FAILURE", part555);

var part556 = match("MESSAGE#525:SERVICED_USAGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Incorrect usage: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Incorrect service usage"),
	dup23,
]));

var msg530 = msg("SERVICED_USAGE", part556);

var part557 = match("MESSAGE#526:SERVICED_WORK_INCONSISTENCY", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: object has unexpected value %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","object has unexpected value"),
	dup23,
]));

var msg531 = msg("SERVICED_WORK_INCONSISTENCY", part557);

var msg532 = msg("SSL_PROXY_SSL_SESSION_ALLOW", dup155);

var msg533 = msg("SSL_PROXY_SSL_SESSION_DROP", dup155);

var msg534 = msg("SSL_PROXY_SESSION_IGNORE", dup155);

var part558 = match("MESSAGE#530:SNMP_NS_LOG_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: NET-SNMP version %{version->} AgentX subagent connected", processor_chain([
	dup21,
	dup22,
	setc("event_description","AgentX subagent connected"),
	dup61,
	dup23,
]));

var msg535 = msg("SNMP_NS_LOG_INFO", part558);

var part559 = match("MESSAGE#531:SNMP_SUBAGENT_IPC_REG_ROWS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ns_subagent_register_mibs: registering %{dclass_counter1->} rows", processor_chain([
	dup21,
	dup22,
	setc("event_description","ns_subagent registering rows"),
	dup61,
	dup23,
]));

var msg536 = msg("SNMP_SUBAGENT_IPC_REG_ROWS", part559);

var part560 = match("MESSAGE#532:SNMPD_ACCESS_GROUP_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result->} in %{dclass_counter1->} access group %{group}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD ACCESS GROUP ERROR"),
	dup23,
]));

var msg537 = msg("SNMPD_ACCESS_GROUP_ERROR", part560);

var part561 = match("MESSAGE#533:SNMPD_AUTH_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: unauthorized SNMP community from %{daddr->} to unknown community name (%{pool_name})", processor_chain([
	dup30,
	dup22,
	dup105,
	setc("result","unauthorized SNMP community to unknown community name"),
	dup23,
]));

var msg538 = msg("SNMPD_AUTH_FAILURE", part561);

var part562 = match("MESSAGE#534:SNMPD_AUTH_FAILURE:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: failed input interface authorization from %{daddr->} to unknown (%{pool_name})", processor_chain([
	dup30,
	dup22,
	dup105,
	setc("result","failed input interface authorization to unknown"),
	dup23,
]));

var msg539 = msg("SNMPD_AUTH_FAILURE:01", part562);

var part563 = match("MESSAGE#535:SNMPD_AUTH_FAILURE:02", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: unauthorized SNMP community from %{daddr->} to %{saddr->} (%{pool_name})", processor_chain([
	dup30,
	dup22,
	dup105,
	setc("result","unauthorized SNMP community "),
	dup23,
]));

var msg540 = msg("SNMPD_AUTH_FAILURE:02", part563);

var part564 = match("MESSAGE#595:SNMPD_AUTH_FAILURE:03", "nwparser.payload", "%{process->} %{process_id->} %{event_type->} [junos@%{obj_name->} function-name=\"%{fld1}\" message=\"%{info}\" source-address=\"%{saddr}\" destination-address=\"%{daddr}\" index1=\"%{fld4}\"]", processor_chain([
	dup30,
	dup22,
	dup105,
	dup61,
	dup62,
]));

var msg541 = msg("SNMPD_AUTH_FAILURE:03", part564);

var select53 = linear_select([
	msg538,
	msg539,
	msg540,
	msg541,
]);

var part565 = match("MESSAGE#536:SNMPD_AUTH_PRIVILEGES_EXCEEDED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{saddr}: request exceeded community privileges", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP request exceeded community privileges"),
	dup23,
]));

var msg542 = msg("SNMPD_AUTH_PRIVILEGES_EXCEEDED", part565);

var part566 = match("MESSAGE#537:SNMPD_AUTH_RESTRICTED_ADDRESS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: request from address %{daddr->} not allowed", processor_chain([
	dup48,
	dup22,
	setc("event_description","SNMPD AUTH RESTRICTED ADDRESS"),
	setc("result","request not allowed"),
	dup23,
]));

var msg543 = msg("SNMPD_AUTH_RESTRICTED_ADDRESS", part566);

var part567 = match("MESSAGE#538:SNMPD_AUTH_WRONG_PDU_TYPE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{saddr}: unauthorized SNMP PDU type: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","unauthorized SNMP PDU type"),
	dup23,
]));

var msg544 = msg("SNMPD_AUTH_WRONG_PDU_TYPE", part567);

var part568 = match("MESSAGE#539:SNMPD_CONFIG_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Configuration database has errors", processor_chain([
	dup30,
	dup22,
	setc("event_description","Configuration database has errors"),
	dup23,
]));

var msg545 = msg("SNMPD_CONFIG_ERROR", part568);

var part569 = match("MESSAGE#540:SNMPD_CONTEXT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result->} in %{dclass_counter1->} context %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD CONTEXT ERROR"),
	dup23,
]));

var msg546 = msg("SNMPD_CONTEXT_ERROR", part569);

var part570 = match("MESSAGE#541:SNMPD_ENGINE_FILE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{dclass_counter2}: operation: %{dclass_counter1->} %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD ENGINE FILE FAILURE"),
	dup23,
]));

var msg547 = msg("SNMPD_ENGINE_FILE_FAILURE", part570);

var part571 = match("MESSAGE#542:SNMPD_ENGINE_PROCESS_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: from-path: undecodable/unmatched subagent response", processor_chain([
	dup30,
	dup22,
	setc("event_description"," from-path - SNMP undecodable/unmatched subagent response"),
	dup23,
]));

var msg548 = msg("SNMPD_ENGINE_PROCESS_ERROR", part571);

var part572 = match("MESSAGE#543:SNMPD_FILE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: fopen %{dclass_counter2}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD FILE FAILURE"),
	dup23,
]));

var msg549 = msg("SNMPD_FILE_FAILURE", part572);

var part573 = match("MESSAGE#544:SNMPD_GROUP_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result->} in %{dclass_counter1->} group: '%{group}' user '%{username}' model '%{version}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD GROUP ERROR"),
	dup23,
]));

var msg550 = msg("SNMPD_GROUP_ERROR", part573);

var part574 = match("MESSAGE#545:SNMPD_INIT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: snmpd initialization failure: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","snmpd initialization failure"),
	dup23,
]));

var msg551 = msg("SNMPD_INIT_FAILED", part574);

var part575 = match("MESSAGE#546:SNMPD_LIBJUNIPER_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: system_default_inaddr: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","LIBJUNIPER FAILURE"),
	dup23,
]));

var msg552 = msg("SNMPD_LIBJUNIPER_FAILURE", part575);

var part576 = match("MESSAGE#547:SNMPD_LOOPBACK_ADDR_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","LOOPBACK ADDR ERROR"),
	dup23,
]));

var msg553 = msg("SNMPD_LOOPBACK_ADDR_ERROR", part576);

var part577 = match("MESSAGE#548:SNMPD_MEMORY_FREED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: called for freed - already freed", processor_chain([
	dup30,
	dup22,
	setc("event_description","duplicate memory free"),
	dup23,
]));

var msg554 = msg("SNMPD_MEMORY_FREED", part577);

var part578 = match("MESSAGE#549:SNMPD_RADIX_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: radix_add failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","radix_add failed"),
	dup23,
]));

var msg555 = msg("SNMPD_RADIX_FAILURE", part578);

var part579 = match("MESSAGE#550:SNMPD_RECEIVE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: receive %{dclass_counter1->} failure: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD RECEIVE FAILURE"),
	dup23,
]));

var msg556 = msg("SNMPD_RECEIVE_FAILURE", part579);

var part580 = match("MESSAGE#551:SNMPD_RMONFILE_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{dclass_counter2}: operation: %{dclass_counter1->} %{agent}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","RMONFILE FAILURE"),
	dup23,
]));

var msg557 = msg("SNMPD_RMONFILE_FAILURE", part580);

var part581 = match("MESSAGE#552:SNMPD_RMON_COOKIE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: Null cookie", processor_chain([
	dup30,
	dup22,
	setc("event_description","Null cookie"),
	dup23,
]));

var msg558 = msg("SNMPD_RMON_COOKIE", part581);

var part582 = match("MESSAGE#553:SNMPD_RMON_EVENTLOG", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","RMON EVENTLOG"),
	dup23,
]));

var msg559 = msg("SNMPD_RMON_EVENTLOG", part582);

var part583 = match("MESSAGE#554:SNMPD_RMON_IOERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: Received io error, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Received io error"),
	dup23,
]));

var msg560 = msg("SNMPD_RMON_IOERROR", part583);

var part584 = match("MESSAGE#555:SNMPD_RMON_MIBERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: internal Get request error: description, %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","internal Get request error"),
	dup23,
]));

var msg561 = msg("SNMPD_RMON_MIBERROR", part584);

var part585 = match("MESSAGE#556:SNMPD_RTSLIB_ASYNC_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: sequence mismatch %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","sequence mismatch"),
	dup23,
]));

var msg562 = msg("SNMPD_RTSLIB_ASYNC_EVENT", part585);

var part586 = match("MESSAGE#557:SNMPD_SEND_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: send send-type (index1) failure: %{result}", processor_chain([
	dup30,
	dup22,
	dup106,
	dup23,
]));

var msg563 = msg("SNMPD_SEND_FAILURE", part586);

var part587 = match("MESSAGE#558:SNMPD_SEND_FAILURE:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: send to (%{saddr}) failure: %{result}", processor_chain([
	dup30,
	dup22,
	dup106,
	dup23,
]));

var msg564 = msg("SNMPD_SEND_FAILURE:01", part587);

var select54 = linear_select([
	msg563,
	msg564,
]);

var part588 = match("MESSAGE#559:SNMPD_SOCKET_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: socket failure: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD SOCKET FAILURE"),
	dup23,
]));

var msg565 = msg("SNMPD_SOCKET_FAILURE", part588);

var part589 = match("MESSAGE#560:SNMPD_SUBAGENT_NO_BUFFERS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: No buffers available for subagent (%{agent})", processor_chain([
	dup30,
	dup22,
	setc("event_description","No buffers available for subagent"),
	dup23,
]));

var msg566 = msg("SNMPD_SUBAGENT_NO_BUFFERS", part589);

var part590 = match("MESSAGE#561:SNMPD_SUBAGENT_SEND_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Send to subagent failed (%{agent}): %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Send to subagent failed"),
	dup23,
]));

var msg567 = msg("SNMPD_SUBAGENT_SEND_FAILED", part590);

var part591 = match("MESSAGE#562:SNMPD_SYSLIB_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: system function '%{dclass_counter1}' failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","system function failed"),
	dup23,
]));

var msg568 = msg("SNMPD_SYSLIB_FAILURE", part591);

var part592 = match("MESSAGE#563:SNMPD_THROTTLE_QUEUE_DRAINED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: cleared all throttled traps", processor_chain([
	dup21,
	dup22,
	setc("event_description","cleared all throttled traps"),
	dup23,
]));

var msg569 = msg("SNMPD_THROTTLE_QUEUE_DRAINED", part592);

var part593 = match("MESSAGE#564:SNMPD_TRAP_COLD_START", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap: cold start", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP trap: cold start"),
	dup23,
]));

var msg570 = msg("SNMPD_TRAP_COLD_START", part593);

var part594 = match("MESSAGE#565:SNMPD_TRAP_GEN_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: %{resultcode->} (%{result})", processor_chain([
	dup30,
	dup22,
	dup107,
	dup23,
]));

var msg571 = msg("SNMPD_TRAP_GEN_FAILURE", part594);

var part595 = match("MESSAGE#566:SNMPD_TRAP_GEN_FAILURE2", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: %{dclass_counter2->} %{result}", processor_chain([
	dup30,
	dup22,
	dup107,
	dup23,
]));

var msg572 = msg("SNMPD_TRAP_GEN_FAILURE2", part595);

var part596 = match("MESSAGE#567:SNMPD_TRAP_INVALID_DATA", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: %{result->} (%{dclass_counter2}) received", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD TRAP INVALID DATA"),
	dup23,
]));

var msg573 = msg("SNMPD_TRAP_INVALID_DATA", part596);

var part597 = match("MESSAGE#568:SNMPD_TRAP_NOT_ENOUGH_VARBINDS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: %{info->} (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD TRAP ERROR"),
	dup23,
]));

var msg574 = msg("SNMPD_TRAP_NOT_ENOUGH_VARBINDS", part597);

var part598 = match("MESSAGE#569:SNMPD_TRAP_QUEUED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Adding trap to %{dclass_counter2->} to %{obj_name->} queue, %{dclass_counter1->} traps in queue", processor_chain([
	dup21,
	dup22,
	setc("event_description","Adding trap to queue"),
	dup23,
]));

var msg575 = msg("SNMPD_TRAP_QUEUED", part598);

var part599 = match("MESSAGE#570:SNMPD_TRAP_QUEUE_DRAINED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: traps queued to %{obj_name->} sent successfully", processor_chain([
	dup21,
	dup22,
	setc("event_description","traps queued - sent successfully"),
	dup23,
]));

var msg576 = msg("SNMPD_TRAP_QUEUE_DRAINED", part599);

var part600 = match("MESSAGE#571:SNMPD_TRAP_QUEUE_MAX_ATTEMPTS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: after %{dclass_counter1->} attempts, deleting %{dclass_counter2->} traps queued to %{obj_name}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD TRAP QUEUE MAX_ATTEMPTS - deleting some traps"),
	dup23,
]));

var msg577 = msg("SNMPD_TRAP_QUEUE_MAX_ATTEMPTS", part600);

var part601 = match("MESSAGE#572:SNMPD_TRAP_QUEUE_MAX_SIZE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: maximum queue size exceeded (%{dclass_counter1}), discarding trap to %{dclass_counter2->} from %{obj_name->} queue", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP TRAP maximum queue size exceeded"),
	dup23,
]));

var msg578 = msg("SNMPD_TRAP_QUEUE_MAX_SIZE", part601);

var part602 = match("MESSAGE#573:SNMPD_TRAP_THROTTLED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: traps throttled after %{dclass_counter1->} traps", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP traps throttled"),
	dup23,
]));

var msg579 = msg("SNMPD_TRAP_THROTTLED", part602);

var part603 = match("MESSAGE#574:SNMPD_TRAP_TYPE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: unknown trap type requested (%{obj_type->} )", processor_chain([
	dup30,
	dup22,
	setc("event_description","unknown SNMP trap type requested"),
	dup23,
]));

var msg580 = msg("SNMPD_TRAP_TYPE_ERROR", part603);

var part604 = match("MESSAGE#575:SNMPD_TRAP_VARBIND_TYPE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: expecting %{dclass_counter1->} varbind to be VT_NUMBER (%{resultcode->} )", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD TRAP VARBIND TYPE ERROR"),
	dup23,
]));

var msg581 = msg("SNMPD_TRAP_VARBIND_TYPE_ERROR", part604);

var part605 = match("MESSAGE#576:SNMPD_TRAP_VERSION_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap error: invalid version signature (%{result})", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD TRAP ERROR - invalid version signature"),
	dup23,
]));

var msg582 = msg("SNMPD_TRAP_VERSION_ERROR", part605);

var part606 = match("MESSAGE#577:SNMPD_TRAP_WARM_START", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: SNMP trap: warm start", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMPD TRAP WARM START"),
	dup23,
]));

var msg583 = msg("SNMPD_TRAP_WARM_START", part606);

var part607 = match("MESSAGE#578:SNMPD_USER_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result->} in %{dclass_counter1->} user '%{username}' %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMPD USER ERROR"),
	dup23,
]));

var msg584 = msg("SNMPD_USER_ERROR", part607);

var part608 = match("MESSAGE#579:SNMPD_VIEW_DELETE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: deleting view %{dclass_counter2->} %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP deleting view"),
	dup23,
]));

var msg585 = msg("SNMPD_VIEW_DELETE", part608);

var part609 = match("MESSAGE#580:SNMPD_VIEW_INSTALL_DEFAULT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: %{result->} installing default %{dclass_counter1->} view %{dclass_counter2}", processor_chain([
	dup21,
	dup22,
	setc("event_description","installing default SNMP view"),
	dup23,
]));

var msg586 = msg("SNMPD_VIEW_INSTALL_DEFAULT", part609);

var part610 = match("MESSAGE#581:SNMPD_VIEW_OID_PARSE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: oid parsing failed for view %{dclass_counter2->} oid %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","oid parsing failed for SNMP view"),
	dup23,
]));

var msg587 = msg("SNMPD_VIEW_OID_PARSE", part610);

var part611 = match("MESSAGE#582:SNMP_GET_ERROR1", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} %{dclass_counter1->} failed for %{dclass_counter2->} : %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP_GET_ERROR 1"),
	dup23,
]));

var msg588 = msg("SNMP_GET_ERROR1", part611);

var part612 = match("MESSAGE#583:SNMP_GET_ERROR2", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} %{dclass_counter1->} failed for %{dclass_counter2->} : %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP GET ERROR 2"),
	dup23,
]));

var msg589 = msg("SNMP_GET_ERROR2", part612);

var part613 = match("MESSAGE#584:SNMP_GET_ERROR3", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} %{dclass_counter1->} failed for %{dclass_counter2->} : %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP GET ERROR 3"),
	dup23,
]));

var msg590 = msg("SNMP_GET_ERROR3", part613);

var part614 = match("MESSAGE#585:SNMP_GET_ERROR4", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent->} %{dclass_counter1->} failed for %{dclass_counter2->} : %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP GET ERROR 4"),
	dup23,
]));

var msg591 = msg("SNMP_GET_ERROR4", part614);

var part615 = match("MESSAGE#586:SNMP_RTSLIB_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: rtslib-error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP RTSLIB FAILURE"),
	dup23,
]));

var msg592 = msg("SNMP_RTSLIB_FAILURE", part615);

var part616 = match("MESSAGE#587:SNMP_TRAP_LINK_DOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifIndex %{dclass_counter1}, ifAdminStatus %{resultcode}, ifOperStatus %{result}, ifName %{interface}", processor_chain([
	dup30,
	dup22,
	dup108,
	dup23,
]));

var msg593 = msg("SNMP_TRAP_LINK_DOWN", part616);

var part617 = match("MESSAGE#596:SNMP_TRAP_LINK_DOWN:01", "nwparser.payload", "%{process->} %{process_id->} %{event_type->} [junos@%{obj_name->} snmp-interface-index=\"%{fld1}\" admin-status=\"%{fld3}\" operational-status=\"%{fld2}\" interface-name=\"%{interface}\"]", processor_chain([
	dup30,
	dup22,
	dup108,
	dup61,
	dup62,
]));

var msg594 = msg("SNMP_TRAP_LINK_DOWN:01", part617);

var select55 = linear_select([
	msg593,
	msg594,
]);

var part618 = match("MESSAGE#588:SNMP_TRAP_LINK_UP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: ifIndex %{dclass_counter1}, ifAdminStatus %{resultcode}, ifOperStatus %{result}, ifName %{interface}", processor_chain([
	dup21,
	dup22,
	dup109,
	dup23,
]));

var msg595 = msg("SNMP_TRAP_LINK_UP", part618);

var part619 = match("MESSAGE#597:SNMP_TRAP_LINK_UP:01", "nwparser.payload", "%{process->} %{process_id->} %{event_type->} [junos@%{obj_name->} snmp-interface-index=\"%{fld1}\" admin-status=\"%{fld3}\" operational-status=\"%{event_state}\" interface-name=\"%{interface}\"]", processor_chain([
	dup21,
	dup22,
	dup109,
	dup61,
	dup62,
]));

var msg596 = msg("SNMP_TRAP_LINK_UP:01", part619);

var select56 = linear_select([
	msg595,
	msg596,
]);

var part620 = match("MESSAGE#589:SNMP_TRAP_PING_PROBE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pingCtlOwnerIndex = %{dclass_counter1}, pingCtlTestName = %{obj_name}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP TRAP PING PROBE FAILED"),
	dup23,
]));

var msg597 = msg("SNMP_TRAP_PING_PROBE_FAILED", part620);

var part621 = match("MESSAGE#590:SNMP_TRAP_PING_TEST_COMPLETED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pingCtlOwnerIndex = %{dclass_counter1}, pingCtlTestName = %{obj_name}", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP TRAP PING TEST COMPLETED"),
	dup23,
]));

var msg598 = msg("SNMP_TRAP_PING_TEST_COMPLETED", part621);

var part622 = match("MESSAGE#591:SNMP_TRAP_PING_TEST_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: pingCtlOwnerIndex = %{dclass_counter1}, pingCtlTestName = %{obj_name}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP TRAP PING TEST FAILED"),
	dup23,
]));

var msg599 = msg("SNMP_TRAP_PING_TEST_FAILED", part622);

var part623 = match("MESSAGE#592:SNMP_TRAP_TRACE_ROUTE_PATH_CHANGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: traceRouteCtlOwnerIndex = %{dclass_counter1}, traceRouteCtlTestName = %{obj_name}", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP TRAP TRACE ROUTE PATH CHANGE"),
	dup23,
]));

var msg600 = msg("SNMP_TRAP_TRACE_ROUTE_PATH_CHANGE", part623);

var part624 = match("MESSAGE#593:SNMP_TRAP_TRACE_ROUTE_TEST_COMPLETED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: traceRouteCtlOwnerIndex = %{dclass_counter1}, traceRouteCtlTestName = %{obj_name}", processor_chain([
	dup21,
	dup22,
	setc("event_description","SNMP TRAP TRACE ROUTE TEST COMPLETED"),
	dup23,
]));

var msg601 = msg("SNMP_TRAP_TRACE_ROUTE_TEST_COMPLETED", part624);

var part625 = match("MESSAGE#594:SNMP_TRAP_TRACE_ROUTE_TEST_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: traceRouteCtlOwnerIndex = %{dclass_counter1}, traceRouteCtlTestName = %{obj_name}", processor_chain([
	dup30,
	dup22,
	setc("event_description","SNMP TRAP TRACE ROUTE TEST FAILED"),
	dup23,
]));

var msg602 = msg("SNMP_TRAP_TRACE_ROUTE_TEST_FAILED", part625);

var part626 = match("MESSAGE#598:SSHD_LOGIN_FAILED", "nwparser.payload", "%{process}: %{event_type}: Login failed for user '%{username}' from host '%{saddr}'", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup110,
	dup23,
]));

var msg603 = msg("SSHD_LOGIN_FAILED", part626);

var part627 = match("MESSAGE#599:SSHD_LOGIN_FAILED:01", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} username=\"%{username}\" source-address=\"%{saddr}\"]", processor_chain([
	dup44,
	dup34,
	dup35,
	dup36,
	dup43,
	dup22,
	dup110,
	dup61,
	dup52,
	setf("process","hfld33"),
]));

var msg604 = msg("SSHD_LOGIN_FAILED:01", part627);

var select57 = linear_select([
	msg603,
	msg604,
]);

var part628 = match("MESSAGE#600:task_connect", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: task %{agent->} addr %{daddr}+%{dport}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","task connect failure"),
	dup23,
]));

var msg605 = msg("task_connect", part628);

var msg606 = msg("TASK_TASK_REINIT", dup149);

var part629 = match("MESSAGE#602:TFTPD_AF_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unexpected address family %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unexpected address family"),
	dup23,
]));

var msg607 = msg("TFTPD_AF_ERR", part629);

var part630 = match("MESSAGE#603:TFTPD_BIND_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: bind: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD BIND ERROR"),
	dup23,
]));

var msg608 = msg("TFTPD_BIND_ERR", part630);

var part631 = match("MESSAGE#604:TFTPD_CONNECT_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: connect: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD CONNECT ERROR"),
	dup23,
]));

var msg609 = msg("TFTPD_CONNECT_ERR", part631);

var part632 = match("MESSAGE#605:TFTPD_CONNECT_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: TFTP %{protocol->} from address %{daddr->} port %{dport->} file %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","TFTPD CONNECT INFO"),
	dup23,
]));

var msg610 = msg("TFTPD_CONNECT_INFO", part632);

var part633 = match("MESSAGE#606:TFTPD_CREATE_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: check_space %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD CREATE ERROR"),
	dup23,
]));

var msg611 = msg("TFTPD_CREATE_ERR", part633);

var part634 = match("MESSAGE#607:TFTPD_FIO_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD FIO ERR"),
	dup23,
]));

var msg612 = msg("TFTPD_FIO_ERR", part634);

var part635 = match("MESSAGE#608:TFTPD_FORK_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: fork: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD FORK ERROR"),
	dup23,
]));

var msg613 = msg("TFTPD_FORK_ERR", part635);

var part636 = match("MESSAGE#609:TFTPD_NAK_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: nak error %{resultcode}, %{dclass_counter1}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD NAK ERROR"),
	dup23,
]));

var msg614 = msg("TFTPD_NAK_ERR", part636);

var part637 = match("MESSAGE#610:TFTPD_OPEN_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to open file '%{filename}', error: %{result}", processor_chain([
	dup30,
	dup22,
	dup78,
	dup23,
]));

var msg615 = msg("TFTPD_OPEN_ERR", part637);

var part638 = match("MESSAGE#611:TFTPD_RECVCOMPLETE_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Received %{dclass_counter1->} blocks of %{dclass_counter2->} size for file '%{filename}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","TFTPD RECVCOMPLETE INFO"),
	dup23,
]));

var msg616 = msg("TFTPD_RECVCOMPLETE_INFO", part638);

var part639 = match("MESSAGE#612:TFTPD_RECVFROM_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: recvfrom: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD RECVFROM ERROR"),
	dup23,
]));

var msg617 = msg("TFTPD_RECVFROM_ERR", part639);

var part640 = match("MESSAGE#613:TFTPD_RECV_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: recv: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD RECV ERROR"),
	dup23,
]));

var msg618 = msg("TFTPD_RECV_ERR", part640);

var part641 = match("MESSAGE#614:TFTPD_SENDCOMPLETE_INFO", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Sent %{dclass_counter1->} blocks of %{dclass_counter2->} and %{info->} for file '%{filename}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","TFTPD SENDCOMPLETE INFO"),
	dup23,
]));

var msg619 = msg("TFTPD_SENDCOMPLETE_INFO", part641);

var part642 = match("MESSAGE#615:TFTPD_SEND_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: send: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD SEND ERROR"),
	dup23,
]));

var msg620 = msg("TFTPD_SEND_ERR", part642);

var part643 = match("MESSAGE#616:TFTPD_SOCKET_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: socket: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD SOCKET ERROR"),
	dup23,
]));

var msg621 = msg("TFTPD_SOCKET_ERR", part643);

var part644 = match("MESSAGE#617:TFTPD_STATFS_ERR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: statfs %{agent}, error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","TFTPD STATFS ERROR"),
	dup23,
]));

var msg622 = msg("TFTPD_STATFS_ERR", part644);

var part645 = match("MESSAGE#618:TNP", "nwparser.payload", "%{process}: %{event_type}: adding neighbor %{dclass_counter1->} to interface %{interface}", processor_chain([
	dup21,
	dup22,
	setc("event_description","adding neighbor to interface"),
	dup23,
]));

var msg623 = msg("TNP", part645);

var part646 = match("MESSAGE#619:trace_on", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: tracing to %{fld33->} started", processor_chain([
	dup21,
	dup22,
	setc("event_description","tracing to file"),
	dup23,
	call({
		dest: "nwparser.filename",
		fn: RMQ,
		args: [
			field("fld33"),
		],
	}),
]));

var msg624 = msg("trace_on", part646);

var part647 = match("MESSAGE#620:trace_rotate", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: rotating %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","trace rotating file"),
	dup23,
]));

var msg625 = msg("trace_rotate", part647);

var part648 = match("MESSAGE#621:transfer-file", "nwparser.payload", "%{process}: %{event_type}: Transferred %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","transfered file"),
	dup23,
]));

var msg626 = msg("transfer-file", part648);

var part649 = match("MESSAGE#622:ttloop", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: peer died: %{result}: %{resultcode}", processor_chain([
	dup30,
	dup22,
	setc("event_description","ttloop - peer died"),
	dup23,
]));

var msg627 = msg("ttloop", part649);

var part650 = match("MESSAGE#623:UI_AUTH_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Authenticated user '%{username}' at permission level '%{privilege}'", processor_chain([
	dup80,
	dup34,
	dup35,
	dup37,
	dup22,
	setc("event_description","Authenticated user"),
	dup23,
]));

var msg628 = msg("UI_AUTH_EVENT", part650);

var part651 = match("MESSAGE#624:UI_AUTH_INVALID_CHALLENGE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Received invalid authentication challenge for user '%{username}': response", processor_chain([
	dup30,
	dup22,
	setc("event_description","Received invalid authentication challenge for user response"),
	dup23,
]));

var msg629 = msg("UI_AUTH_INVALID_CHALLENGE", part651);

var part652 = match("MESSAGE#625:UI_BOOTTIME_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to fetch boot time: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to fetch boot time"),
	dup23,
]));

var msg630 = msg("UI_BOOTTIME_FAILED", part652);

var part653 = match("MESSAGE#626:UI_CFG_AUDIT_NEW", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' %{dclass_counter2->} path unknown", processor_chain([
	dup30,
	dup22,
	setc("event_description","user path unknown"),
	dup23,
]));

var msg631 = msg("UI_CFG_AUDIT_NEW", part653);

var part654 = match("MESSAGE#627:UI_CFG_AUDIT_NEW:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' insert: [edit-config config %{filename->} security policies %{policyname}] %{info}", processor_chain([
	dup42,
	dup22,
	setc("event_description"," user Inserted Security Policies in config"),
	dup23,
]));

var msg632 = msg("UI_CFG_AUDIT_NEW:01", part654);

var select58 = linear_select([
	msg631,
	msg632,
]);

var part655 = match("MESSAGE#628:UI_CFG_AUDIT_OTHER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' delete: [%{filename}]", processor_chain([
	dup21,
	dup22,
	setc("event_description","User deleted file"),
	setc("action","delete"),
	dup23,
]));

var msg633 = msg("UI_CFG_AUDIT_OTHER", part655);

var part656 = match("MESSAGE#629:UI_CFG_AUDIT_OTHER:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' rollback: %{filename}", processor_chain([
	dup21,
	dup22,
	setc("event_description","User rollback file"),
	dup23,
]));

var msg634 = msg("UI_CFG_AUDIT_OTHER:01", part656);

var part657 = match("MESSAGE#630:UI_CFG_AUDIT_OTHER:02/1_0", "nwparser.p0", "\"%{info}\"");

var select59 = linear_select([
	part657,
	dup112,
]);

var all31 = all_match({
	processors: [
		dup111,
		select59,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","User set"),
		dup23,
	]),
});

var msg635 = msg("UI_CFG_AUDIT_OTHER:02", all31);

var part658 = match("MESSAGE#631:UI_CFG_AUDIT_OTHER:03", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' replace: [edit-config config %{filename->} applications %{info}]", processor_chain([
	dup21,
	dup22,
	setc("event_description","User config replace"),
	setc("action","replace"),
	dup23,
]));

var msg636 = msg("UI_CFG_AUDIT_OTHER:03", part658);

var part659 = match("MESSAGE#632:UI_CFG_AUDIT_OTHER:04", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' deactivate: [groups %{info}]", processor_chain([
	setc("eventcategory","1701070000"),
	dup22,
	setc("event_description","User deactivating group(s)"),
	setc("action","deactivate"),
	dup23,
]));

var msg637 = msg("UI_CFG_AUDIT_OTHER:04", part659);

var part660 = match("MESSAGE#633:UI_CFG_AUDIT_OTHER:05", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' update: %{filename}", processor_chain([
	dup113,
	dup22,
	setc("event_description","User updates config file"),
	setc("action","update"),
	dup23,
]));

var msg638 = msg("UI_CFG_AUDIT_OTHER:05", part660);

var select60 = linear_select([
	msg633,
	msg634,
	msg635,
	msg636,
	msg637,
	msg638,
]);

var part661 = match("MESSAGE#634:UI_CFG_AUDIT_SET:01/1_0", "nwparser.p0", "\"%{change_old}\" %{p0}");

var select61 = linear_select([
	part661,
	dup114,
]);

var all32 = all_match({
	processors: [
		dup111,
		select61,
		dup115,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup116,
		dup23,
	]),
});

var msg639 = msg("UI_CFG_AUDIT_SET:01", all32);

var part662 = match("MESSAGE#635:UI_CFG_AUDIT_SET:02/1_0", "nwparser.p0", "\"%{change_old->} %{p0}");

var select62 = linear_select([
	part662,
	dup114,
]);

var all33 = all_match({
	processors: [
		dup111,
		select62,
		dup115,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup116,
		dup23,
	]),
});

var msg640 = msg("UI_CFG_AUDIT_SET:02", all33);

var part663 = match("MESSAGE#636:UI_CFG_AUDIT_SET", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' replace: [edit-config config %{filename->} applications %{info}] \u003c\u003c%{disposition}> -> \"%{agent}\"", processor_chain([
	dup21,
	dup22,
	setc("event_description","User replace config application(s)"),
	dup23,
]));

var msg641 = msg("UI_CFG_AUDIT_SET", part663);

var select63 = linear_select([
	msg639,
	msg640,
	msg641,
]);

var part664 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/2", "nwparser.p0", ": [groups %{info->} secret]");

var all34 = all_match({
	processors: [
		dup117,
		dup156,
		part664,
	],
	on_success: processor_chain([
		dup113,
		dup22,
		dup120,
		dup23,
	]),
});

var msg642 = msg("UI_CFG_AUDIT_SET_SECRET:01", all34);

var part665 = match("MESSAGE#638:UI_CFG_AUDIT_SET_SECRET:02/2", "nwparser.p0", ": [%{info}]");

var all35 = all_match({
	processors: [
		dup117,
		dup156,
		part665,
	],
	on_success: processor_chain([
		dup113,
		dup22,
		dup120,
		dup23,
	]),
});

var msg643 = msg("UI_CFG_AUDIT_SET_SECRET:02", all35);

var part666 = match("MESSAGE#639:UI_CFG_AUDIT_SET_SECRET", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' %{dclass_counter2->} %{directory}", processor_chain([
	dup21,
	dup22,
	setc("event_description","UI CFG AUDIT SET SECRET"),
	dup23,
]));

var msg644 = msg("UI_CFG_AUDIT_SET_SECRET", part666);

var select64 = linear_select([
	msg642,
	msg643,
	msg644,
]);

var part667 = match("MESSAGE#640:UI_CHILD_ARGS_EXCEEDED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Too many arguments for child process '%{agent}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Too many arguments for child process"),
	dup23,
]));

var msg645 = msg("UI_CHILD_ARGS_EXCEEDED", part667);

var part668 = match("MESSAGE#641:UI_CHILD_CHANGE_USER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to switch to local user: %{username}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to switch to local user"),
	dup23,
]));

var msg646 = msg("UI_CHILD_CHANGE_USER", part668);

var part669 = match("MESSAGE#642:UI_CHILD_EXEC", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child exec failed for command '%{action}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Child exec failed"),
	dup23,
]));

var msg647 = msg("UI_CHILD_EXEC", part669);

var part670 = match("MESSAGE#643:UI_CHILD_EXITED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child exited: PID %{child_pid}, status %{result}, command '%{action}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Child exited"),
	dup23,
]));

var msg648 = msg("UI_CHILD_EXITED", part670);

var part671 = match("MESSAGE#644:UI_CHILD_FOPEN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to append to log '%{filename}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to append to log"),
	dup23,
]));

var msg649 = msg("UI_CHILD_FOPEN", part671);

var part672 = match("MESSAGE#645:UI_CHILD_PIPE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to create pipe for command '%{action}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to create pipe for command"),
	dup23,
]));

var msg650 = msg("UI_CHILD_PIPE_FAILED", part672);

var part673 = match("MESSAGE#646:UI_CHILD_SIGNALED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child received signal: PID %{child_pid}, signal %{result}: %{resultcode}, command='%{action}'", processor_chain([
	dup21,
	dup22,
	dup61,
	setc("event_description","Child received signal"),
	dup23,
]));

var msg651 = msg("UI_CHILD_SIGNALED", part673);

var part674 = match("MESSAGE#647:UI_CHILD_STOPPED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Child stopped: PID %{child_pid}, signal=%{resultcode->} command='%{action}')", processor_chain([
	dup21,
	dup22,
	setc("event_description","Child stopped"),
	dup23,
]));

var msg652 = msg("UI_CHILD_STOPPED", part674);

var part675 = match("MESSAGE#648:UI_CHILD_START", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Starting child '%{agent}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","Starting child"),
	dup23,
]));

var msg653 = msg("UI_CHILD_START", part675);

var part676 = match("MESSAGE#649:UI_CHILD_STATUS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Cleanup child '%{agent}', PID %{child_pid}, status %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Cleanup child"),
	dup23,
]));

var msg654 = msg("UI_CHILD_STATUS", part676);

var part677 = match("MESSAGE#650:UI_CHILD_WAITPID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: waitpid failed: PID %{child_pid}, rc %{dclass_counter2}, status %{resultcode}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","waitpid failed"),
	dup23,
]));

var msg655 = msg("UI_CHILD_WAITPID", part677);

var part678 = match("MESSAGE#651:UI_CLI_IDLE_TIMEOUT", "nwparser.payload", "%{event_type}: Idle timeout for user '%{username}' exceeded and %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Idle timeout for user exceeded"),
	dup23,
]));

var msg656 = msg("UI_CLI_IDLE_TIMEOUT", part678);

var part679 = match("MESSAGE#652:UI_CMDLINE_READ_LINE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}', command '%{action}'", processor_chain([
	dup21,
	dup22,
	dup121,
	dup23,
]));

var msg657 = msg("UI_CMDLINE_READ_LINE", part679);

var part680 = match("MESSAGE#653:UI_CMDSET_EXEC_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command execution failed for '%{agent}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Command execution failed"),
	dup23,
]));

var msg658 = msg("UI_CMDSET_EXEC_FAILED", part680);

var part681 = match("MESSAGE#654:UI_CMDSET_FORK_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to fork command '%{agent}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to fork command"),
	dup23,
]));

var msg659 = msg("UI_CMDSET_FORK_FAILED", part681);

var msg660 = msg("UI_CMDSET_PIPE_FAILED", dup144);

var part682 = match("MESSAGE#656:UI_CMDSET_STOPPED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command stopped: PID %{child_pid}, signal '%{resultcode}, command '%{action}'", processor_chain([
	dup30,
	dup22,
	dup70,
	dup23,
]));

var msg661 = msg("UI_CMDSET_STOPPED", part682);

var part683 = match("MESSAGE#657:UI_CMDSET_WEXITED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Command exited: PID %{child_pid}, status %{resultcode}, command '%{action}'", processor_chain([
	dup30,
	dup22,
	dup72,
	dup23,
]));

var msg662 = msg("UI_CMDSET_WEXITED", part683);

var part684 = match("MESSAGE#658:UI_CMD_AUTH_REGEX_INVALID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Invalid '%{action}' command authorization regular expression '%{agent}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Invalid regexp command"),
	dup23,
]));

var msg663 = msg("UI_CMD_AUTH_REGEX_INVALID", part684);

var part685 = match("MESSAGE#659:UI_COMMIT/1_0", "nwparser.p0", "requested '%{action}' operation (comment:%{info})");

var part686 = match("MESSAGE#659:UI_COMMIT/1_1", "nwparser.p0", "performed %{action}");

var select65 = linear_select([
	part685,
	part686,
]);

var all36 = all_match({
	processors: [
		dup117,
		select65,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup122,
		dup23,
	]),
});

var msg664 = msg("UI_COMMIT", all36);

var part687 = match("MESSAGE#660:UI_COMMIT_AT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' performed %{result}", processor_chain([
	dup21,
	dup22,
	dup122,
	dup23,
]));

var msg665 = msg("UI_COMMIT_AT", part687);

var part688 = match("MESSAGE#661:UI_COMMIT_AT_COMPLETED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: '%{agent}' was successful", processor_chain([
	dup21,
	dup22,
	setc("event_description","User commit successful"),
	dup23,
]));

var msg666 = msg("UI_COMMIT_AT_COMPLETED", part688);

var part689 = match("MESSAGE#662:UI_COMMIT_AT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{result}, %{info}", processor_chain([
	dup30,
	dup22,
	setc("event_description","User commit failed"),
	dup23,
]));

var msg667 = msg("UI_COMMIT_AT_FAILED", part689);

var part690 = match("MESSAGE#663:UI_COMMIT_COMPRESS_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to compress file %{filename}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to compress file"),
	dup23,
]));

var msg668 = msg("UI_COMMIT_COMPRESS_FAILED", part690);

var part691 = match("MESSAGE#664:UI_COMMIT_CONFIRMED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' performed '%{action}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","UI COMMIT CONFIRMED"),
	dup23,
]));

var msg669 = msg("UI_COMMIT_CONFIRMED", part691);

var part692 = match("MESSAGE#665:UI_COMMIT_CONFIRMED_REMINDER/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: '%{action}' must be confirmed within %{p0}");

var part693 = match("MESSAGE#665:UI_COMMIT_CONFIRMED_REMINDER/1_0", "nwparser.p0", "minutes %{dclass_counter1}");

var part694 = match("MESSAGE#665:UI_COMMIT_CONFIRMED_REMINDER/1_1", "nwparser.p0", "%{dclass_counter1->} minutes");

var select66 = linear_select([
	part693,
	part694,
]);

var all37 = all_match({
	processors: [
		part692,
		select66,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","COMMIT must be confirmed within # minutes"),
		dup23,
	]),
});

var msg670 = msg("UI_COMMIT_CONFIRMED_REMINDER", all37);

var part695 = match("MESSAGE#666:UI_COMMIT_CONFIRMED_TIMED/2", "nwparser.p0", "'%{username}' performed '%{action}'");

var all38 = all_match({
	processors: [
		dup50,
		dup145,
		part695,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","user performed commit confirm"),
		dup23,
	]),
});

var msg671 = msg("UI_COMMIT_CONFIRMED_TIMED", all38);

var part696 = match("MESSAGE#667:UI_COMMIT_EMPTY_CONTAINER", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Skipped empty object %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Skipped empty object"),
	dup23,
]));

var msg672 = msg("UI_COMMIT_EMPTY_CONTAINER", part696);

var part697 = match("MESSAGE#668:UI_COMMIT_NOT_CONFIRMED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Commit was not confirmed; %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","COMMIT NOT CONFIRMED"),
	dup23,
]));

var msg673 = msg("UI_COMMIT_NOT_CONFIRMED", part697);

var part698 = match("MESSAGE#669:UI_COMMIT_PROGRESS/1_0", "nwparser.p0", "commit %{p0}");

var part699 = match("MESSAGE#669:UI_COMMIT_PROGRESS/1_1", "nwparser.p0", "Commit operation in progress %{p0}");

var select67 = linear_select([
	part698,
	part699,
]);

var part700 = match("MESSAGE#669:UI_COMMIT_PROGRESS/2", "nwparser.p0", ": %{action}");

var all39 = all_match({
	processors: [
		dup50,
		select67,
		part700,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","Commit operation in progress"),
		dup23,
	]),
});

var msg674 = msg("UI_COMMIT_PROGRESS", all39);

var part701 = match("MESSAGE#670:UI_COMMIT_QUIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' performed %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","COMMIT QUIT"),
	dup23,
]));

var msg675 = msg("UI_COMMIT_QUIT", part701);

var part702 = match("MESSAGE#671:UI_COMMIT_ROLLBACK_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Automatic rollback failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","Automatic rollback failed"),
	dup23,
]));

var msg676 = msg("UI_COMMIT_ROLLBACK_FAILED", part702);

var part703 = match("MESSAGE#672:UI_COMMIT_SYNC", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' performed %{action}", processor_chain([
	dup21,
	dup22,
	setc("event_description","COMMIT SYNC"),
	dup23,
]));

var msg677 = msg("UI_COMMIT_SYNC", part703);

var part704 = match("MESSAGE#673:UI_COMMIT_SYNC_FORCE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: All logins to local configuration database were terminated because %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","All logins to local configuration database were terminated"),
	dup23,
]));

var msg678 = msg("UI_COMMIT_SYNC_FORCE", part704);

var part705 = match("MESSAGE#674:UI_CONFIGURATION_ERROR/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Process: %{agent}, path: %{p0}");

var part706 = match("MESSAGE#674:UI_CONFIGURATION_ERROR/1_0", "nwparser.p0", "[%{filename}], %{p0}");

var part707 = match("MESSAGE#674:UI_CONFIGURATION_ERROR/1_1", "nwparser.p0", "%{filename}, %{p0}");

var select68 = linear_select([
	part706,
	part707,
]);

var part708 = match("MESSAGE#674:UI_CONFIGURATION_ERROR/2", "nwparser.p0", "statement: %{info->} %{p0}");

var part709 = match("MESSAGE#674:UI_CONFIGURATION_ERROR/3_0", "nwparser.p0", ", error: %{result->} ");

var select69 = linear_select([
	part709,
	dup112,
]);

var all40 = all_match({
	processors: [
		part705,
		select68,
		part708,
		select69,
	],
	on_success: processor_chain([
		dup30,
		dup22,
		setc("event_description","CONFIGURATION ERROR"),
		dup23,
	]),
});

var msg679 = msg("UI_CONFIGURATION_ERROR", all40);

var part710 = match("MESSAGE#675:UI_DAEMON_ACCEPT_FAILED/2", "nwparser.p0", "socket connection accept failed: %{result}");

var all41 = all_match({
	processors: [
		dup50,
		dup157,
		part710,
	],
	on_success: processor_chain([
		dup30,
		dup22,
		setc("event_description","socket connection accept failed"),
		dup23,
	]),
});

var msg680 = msg("UI_DAEMON_ACCEPT_FAILED", all41);

var part711 = match("MESSAGE#676:UI_DAEMON_FORK_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to create session child: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to create session child"),
	dup23,
]));

var msg681 = msg("UI_DAEMON_FORK_FAILED", part711);

var part712 = match("MESSAGE#677:UI_DAEMON_SELECT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: select failed: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","DAEMON SELECT FAILED"),
	dup23,
]));

var msg682 = msg("UI_DAEMON_SELECT_FAILED", part712);

var part713 = match("MESSAGE#678:UI_DAEMON_SOCKET_FAILED/2", "nwparser.p0", "socket create failed: %{result}");

var all42 = all_match({
	processors: [
		dup50,
		dup157,
		part713,
	],
	on_success: processor_chain([
		dup30,
		dup22,
		setc("event_description","socket create failed"),
		dup23,
	]),
});

var msg683 = msg("UI_DAEMON_SOCKET_FAILED", all42);

var part714 = match("MESSAGE#679:UI_DBASE_ACCESS_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to reaccess database file '%{filename}', address %{interface}, size %{dclass_counter1}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to reaccess database file"),
	dup23,
]));

var msg684 = msg("UI_DBASE_ACCESS_FAILED", part714);

var part715 = match("MESSAGE#680:UI_DBASE_CHECKOUT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database '%{filename}' is out of data and needs to be rebuilt", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database is out of data"),
	dup23,
]));

var msg685 = msg("UI_DBASE_CHECKOUT_FAILED", part715);

var part716 = match("MESSAGE#681:UI_DBASE_EXTEND_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to extend database file '%{filename}' to size %{dclass_counter1}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to extend database file"),
	dup23,
]));

var msg686 = msg("UI_DBASE_EXTEND_FAILED", part716);

var part717 = match("MESSAGE#682:UI_DBASE_LOGIN_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' entering configuration mode", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	setc("event_description","User entering configuration mode"),
	dup23,
]));

var msg687 = msg("UI_DBASE_LOGIN_EVENT", part717);

var part718 = match("MESSAGE#683:UI_DBASE_LOGOUT_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' %{event_description}", processor_chain([
	dup125,
	dup34,
	dup35,
	dup126,
	dup37,
	dup22,
	setc("event_description","User exiting configuration mode"),
	dup23,
]));

var msg688 = msg("UI_DBASE_LOGOUT_EVENT", part718);

var part719 = match("MESSAGE#684:UI_DBASE_MISMATCH_EXTENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database header extent mismatch for file '%{agent}': expecting %{dclass_counter1}, got %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database header extent mismatch"),
	dup23,
]));

var msg689 = msg("UI_DBASE_MISMATCH_EXTENT", part719);

var part720 = match("MESSAGE#685:UI_DBASE_MISMATCH_MAJOR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database header major version number mismatch for file '%{filename}': expecting %{dclass_counter1}, got %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database header major version number mismatch"),
	dup23,
]));

var msg690 = msg("UI_DBASE_MISMATCH_MAJOR", part720);

var part721 = match("MESSAGE#686:UI_DBASE_MISMATCH_MINOR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database header minor version number mismatch for file '%{filename}': expecting %{dclass_counter1}, got %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database header minor version number mismatch"),
	dup23,
]));

var msg691 = msg("UI_DBASE_MISMATCH_MINOR", part721);

var part722 = match("MESSAGE#687:UI_DBASE_MISMATCH_SEQUENCE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database header sequence numbers mismatch for file '%{filename}'", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database header sequence numbers mismatch"),
	dup23,
]));

var msg692 = msg("UI_DBASE_MISMATCH_SEQUENCE", part722);

var part723 = match("MESSAGE#688:UI_DBASE_MISMATCH_SIZE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database header size mismatch for file '%{filename}': expecting %{dclass_counter1}, got %{dclass_counter2}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database header size mismatch"),
	dup23,
]));

var msg693 = msg("UI_DBASE_MISMATCH_SIZE", part723);

var part724 = match("MESSAGE#689:UI_DBASE_OPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Database open failed for file '%{filename}': %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Database open failed"),
	dup23,
]));

var msg694 = msg("UI_DBASE_OPEN_FAILED", part724);

var part725 = match("MESSAGE#690:UI_DBASE_REBUILD_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User %{username->} Automatic rebuild of the database '%{filename}' failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","DBASE REBUILD FAILED"),
	dup23,
]));

var msg695 = msg("UI_DBASE_REBUILD_FAILED", part725);

var part726 = match("MESSAGE#691:UI_DBASE_REBUILD_SCHEMA_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Automatic rebuild of the database failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","Automatic rebuild of the database failed"),
	dup23,
]));

var msg696 = msg("UI_DBASE_REBUILD_SCHEMA_FAILED", part726);

var part727 = match("MESSAGE#692:UI_DBASE_REBUILD_STARTED/1_1", "nwparser.p0", "Automatic %{p0}");

var select70 = linear_select([
	dup76,
	part727,
]);

var part728 = match("MESSAGE#692:UI_DBASE_REBUILD_STARTED/2", "nwparser.p0", "%{username->} rebuild/rollback of the database '%{filename}' started");

var all43 = all_match({
	processors: [
		dup50,
		select70,
		part728,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		setc("event_description","DBASE REBUILD STARTED"),
		dup23,
	]),
});

var msg697 = msg("UI_DBASE_REBUILD_STARTED", all43);

var part729 = match("MESSAGE#693:UI_DBASE_RECREATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' attempting database re-creation", processor_chain([
	dup21,
	dup22,
	setc("event_description","user attempting database re-creation"),
	dup23,
]));

var msg698 = msg("UI_DBASE_RECREATE", part729);

var part730 = match("MESSAGE#694:UI_DBASE_REOPEN_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reopen of the database failed", processor_chain([
	dup30,
	dup22,
	setc("event_description","Reopen of the database failed"),
	dup23,
]));

var msg699 = msg("UI_DBASE_REOPEN_FAILED", part730);

var part731 = match("MESSAGE#695:UI_DUPLICATE_UID", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Users %{username->} have the same UID %{uid}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Users have the same UID"),
	dup23,
]));

var msg700 = msg("UI_DUPLICATE_UID", part731);

var part732 = match("MESSAGE#696:UI_JUNOSCRIPT_CMD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' used JUNOScript client to run command '%{action}'", processor_chain([
	setc("eventcategory","1401050100"),
	dup22,
	setc("event_description","User used JUNOScript client to run command"),
	dup23,
]));

var msg701 = msg("UI_JUNOSCRIPT_CMD", part732);

var part733 = match("MESSAGE#697:UI_JUNOSCRIPT_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: JUNOScript error: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","JUNOScript error"),
	dup23,
]));

var msg702 = msg("UI_JUNOSCRIPT_ERROR", part733);

var part734 = match("MESSAGE#698:UI_LOAD_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' is performing a '%{action}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","User command"),
	dup23,
]));

var msg703 = msg("UI_LOAD_EVENT", part734);

var part735 = match("MESSAGE#699:UI_LOAD_JUNOS_DEFAULT_FILE_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Loading the default config from %{filename}", processor_chain([
	setc("eventcategory","1701040000"),
	dup22,
	setc("event_description","Loading default config from file"),
	dup23,
]));

var msg704 = msg("UI_LOAD_JUNOS_DEFAULT_FILE_EVENT", part735);

var part736 = match("MESSAGE#700:UI_LOGIN_EVENT:01", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' login, class '%{group}' [%{fld01}], %{info->} '%{saddr->} %{sport->} %{daddr->} %{dport}', client-mode '%{fld02}'", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	dup127,
	dup128,
	dup23,
]));

var msg705 = msg("UI_LOGIN_EVENT:01", part736);

var part737 = match("MESSAGE#701:UI_LOGIN_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' login, class '%{group}' %{info}", processor_chain([
	dup33,
	dup34,
	dup35,
	dup36,
	dup37,
	dup22,
	dup127,
	dup23,
]));

var msg706 = msg("UI_LOGIN_EVENT", part737);

var select71 = linear_select([
	msg705,
	msg706,
]);

var part738 = match("MESSAGE#702:UI_LOGOUT_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' logout", processor_chain([
	dup125,
	dup34,
	dup35,
	dup126,
	dup37,
	dup22,
	setc("event_description","User logout"),
	dup23,
]));

var msg707 = msg("UI_LOGOUT_EVENT", part738);

var part739 = match("MESSAGE#703:UI_LOST_CONN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Lost connection to daemon %{agent}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Lost connection to daemon"),
	dup23,
]));

var msg708 = msg("UI_LOST_CONN", part739);

var part740 = match("MESSAGE#704:UI_MASTERSHIP_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action->} by '%{username}'", processor_chain([
	dup21,
	dup22,
	setc("event_description","MASTERSHIP EVENT"),
	dup23,
]));

var msg709 = msg("UI_MASTERSHIP_EVENT", part740);

var part741 = match("MESSAGE#705:UI_MGD_TERMINATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Terminating operation: exit status %{resultcode}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Terminating operation"),
	dup23,
]));

var msg710 = msg("UI_MGD_TERMINATE", part741);

var part742 = match("MESSAGE#706:UI_NETCONF_CMD", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' used NETCONF client to run command '%{action}'", processor_chain([
	dup29,
	dup22,
	setc("event_description","User used NETCONF client to run command"),
	dup23,
]));

var msg711 = msg("UI_NETCONF_CMD", part742);

var part743 = match("MESSAGE#707:UI_READ_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: read failed for peer %{hostname}: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","read failed for peer"),
	dup23,
]));

var msg712 = msg("UI_READ_FAILED", part743);

var part744 = match("MESSAGE#708:UI_READ_TIMEOUT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Timeout on read of peer %{hostname}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Timeout on read of peer"),
	dup23,
]));

var msg713 = msg("UI_READ_TIMEOUT", part744);

var part745 = match("MESSAGE#709:UI_REBOOT_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: System %{action->} by '%{username}'", processor_chain([
	dup60,
	dup22,
	setc("event_description","System reboot or halt"),
	dup23,
]));

var msg714 = msg("UI_REBOOT_EVENT", part745);

var part746 = match("MESSAGE#710:UI_RESTART_EVENT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: user '%{username}' restarting daemon %{service}", processor_chain([
	dup29,
	dup22,
	setc("event_description","user restarting daemon"),
	dup23,
]));

var msg715 = msg("UI_RESTART_EVENT", part746);

var part747 = match("MESSAGE#711:UI_SCHEMA_CHECKOUT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Schema is out of date and %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Schema is out of date"),
	dup23,
]));

var msg716 = msg("UI_SCHEMA_CHECKOUT_FAILED", part747);

var part748 = match("MESSAGE#712:UI_SCHEMA_MISMATCH_MAJOR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Schema major version mismatch for package %{filename->} %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Schema major version mismatch"),
	dup23,
]));

var msg717 = msg("UI_SCHEMA_MISMATCH_MAJOR", part748);

var part749 = match("MESSAGE#713:UI_SCHEMA_MISMATCH_MINOR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Schema minor version mismatch for package %{filename->} %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Schema minor version mismatch"),
	dup23,
]));

var msg718 = msg("UI_SCHEMA_MISMATCH_MINOR", part749);

var part750 = match("MESSAGE#714:UI_SCHEMA_MISMATCH_SEQUENCE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Schema header sequence numbers mismatch for package %{filename}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Schema header sequence numbers mismatch"),
	dup23,
]));

var msg719 = msg("UI_SCHEMA_MISMATCH_SEQUENCE", part750);

var part751 = match("MESSAGE#715:UI_SCHEMA_SEQUENCE_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Schema sequence number mismatch", processor_chain([
	dup30,
	dup22,
	setc("event_description","Schema sequence number mismatch"),
	dup23,
]));

var msg720 = msg("UI_SCHEMA_SEQUENCE_ERROR", part751);

var part752 = match("MESSAGE#716:UI_SYNC_OTHER_RE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Configuration synchronization with remote Routing Engine %{result}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Configuration synchronization with remote Routing Engine"),
	dup23,
]));

var msg721 = msg("UI_SYNC_OTHER_RE", part752);

var part753 = match("MESSAGE#717:UI_TACPLUS_ERROR", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: TACACS+ failure: %{result}", processor_chain([
	dup30,
	dup22,
	dup129,
	dup23,
]));

var msg722 = msg("UI_TACPLUS_ERROR", part753);

var part754 = match("MESSAGE#718:UI_VERSION_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to fetch system version: %{result}", processor_chain([
	dup30,
	dup22,
	setc("event_description","Unable to fetch system version"),
	dup23,
]));

var msg723 = msg("UI_VERSION_FAILED", part754);

var part755 = match("MESSAGE#719:UI_WRITE_RECONNECT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Re-establishing connection to peer %{hostname}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Re-establishing connection to peer"),
	dup23,
]));

var msg724 = msg("UI_WRITE_RECONNECT", part755);

var part756 = match("MESSAGE#720:VRRPD_NEWMASTER_TRAP", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Interface %{interface->} (local addr: %{saddr}) is now master for %{username}", processor_chain([
	dup21,
	dup22,
	setc("event_description","Interface new master for User"),
	dup23,
]));

var msg725 = msg("VRRPD_NEWMASTER_TRAP", part756);

var part757 = match("MESSAGE#721:WEB_AUTH_FAIL", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to authenticate %{obj_name->} (username %{c_username})", processor_chain([
	dup69,
	dup34,
	dup35,
	dup43,
	dup22,
	setc("event_description","Unable to authenticate client"),
	dup23,
]));

var msg726 = msg("WEB_AUTH_FAIL", part757);

var part758 = match("MESSAGE#722:WEB_AUTH_SUCCESS", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Authenticated %{agent->} client (username %{c_username})", processor_chain([
	dup80,
	dup34,
	dup35,
	dup37,
	dup22,
	setc("event_description","Authenticated client"),
	dup23,
]));

var msg727 = msg("WEB_AUTH_SUCCESS", part758);

var part759 = match("MESSAGE#723:WEB_INTERFACE_UNAUTH", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Web services request received from unauthorized interface %{interface}", processor_chain([
	setc("eventcategory","1001030300"),
	dup22,
	setc("event_description","web request from unauthorized interface"),
	dup23,
]));

var msg728 = msg("WEB_INTERFACE_UNAUTH", part759);

var part760 = match("MESSAGE#724:WEB_READ", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to read from client: %{result}", processor_chain([
	dup74,
	dup22,
	setc("event_description","Unable to read from client"),
	dup23,
]));

var msg729 = msg("WEB_READ", part760);

var part761 = match("MESSAGE#725:WEBFILTER_REQUEST_NOT_CHECKED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Error encountered: %{result}, failed to check request %{url}", processor_chain([
	setc("eventcategory","1204020100"),
	dup22,
	setc("event_description","failed to check web request"),
	dup23,
]));

var msg730 = msg("WEBFILTER_REQUEST_NOT_CHECKED", part761);

var part762 = match("MESSAGE#726:FLOW_REASSEMBLE_FAIL", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} source-address=\"%{saddr}\" destination-address=\"%{daddr}\" assembly-id=\"%{fld1}\"]", processor_chain([
	dup74,
	dup53,
	dup43,
	dup22,
	dup52,
]));

var msg731 = msg("FLOW_REASSEMBLE_FAIL", part762);

var part763 = match("MESSAGE#727:eswd", "nwparser.payload", "%{process}[%{process_id}]: Bridge Address: add %{macaddr}", processor_chain([
	dup29,
	dup22,
	setc("event_description","Bridge Address"),
	dup23,
]));

var msg732 = msg("eswd", part763);

var part764 = match("MESSAGE#728:eswd:01", "nwparser.payload", "%{process}[%{process_id}]: %{info}: STP state for interface %{interface->} context id %{id->} changed from %{fld3}", processor_chain([
	dup29,
	dup22,
	setc("event_description","ESWD STP State Change Info"),
	dup23,
]));

var msg733 = msg("eswd:01", part764);

var select72 = linear_select([
	msg732,
	msg733,
]);

var part765 = match("MESSAGE#729:/usr/sbin/cron", "nwparser.payload", "%{process}[%{process_id}]: (%{username}) CMD ( %{action})", processor_chain([
	dup29,
	dup22,
	dup26,
	dup23,
]));

var msg734 = msg("/usr/sbin/cron", part765);

var part766 = match("MESSAGE#730:chassism:02", "nwparser.payload", "%{process}[%{process_id}]: %{info}: ifd %{interface->} %{action}", processor_chain([
	dup29,
	dup22,
	setc("event_description","Link status change event"),
	dup23,
]));

var msg735 = msg("chassism:02", part766);

var part767 = match("MESSAGE#731:chassism:01", "nwparser.payload", "%{process}[%{process_id}]: %{info}: %{interface}, %{action}", processor_chain([
	dup29,
	dup22,
	setc("event_description","ifd process flaps"),
	dup23,
]));

var msg736 = msg("chassism:01", part767);

var part768 = match("MESSAGE#732:chassism", "nwparser.payload", "%{process}[%{process_id}]: %{info}: %{action}", processor_chain([
	dup29,
	dup22,
	setc("event_description","IFCM "),
	dup23,
]));

var msg737 = msg("chassism", part768);

var select73 = linear_select([
	msg735,
	msg736,
	msg737,
]);

var msg738 = msg("WEBFILTER_URL_PERMITTED", dup158);

var part769 = match("MESSAGE#734:WEBFILTER_URL_PERMITTED:01", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=\"%{directory}\" username=\"%{username}\" roles=\"%{user_role}\"] WebFilter: ACTION=\"%{action}\" %{fld2}->%{fld3->} CATEGORY=\"%{category}\" REASON=\"%{fld4}\" PROFILE=\"%{fld6}\" URL=%{url->} OBJ=%{fld7}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg739 = msg("WEBFILTER_URL_PERMITTED:01", part769);

var part770 = match("MESSAGE#735:WEBFILTER_URL_PERMITTED:03", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=\"%{directory}\" username=\"%{username}\" roles=\"%{user_role}\"] WebFilter: ACTION=\"%{action}\" %{fld2}->%{fld3->} CATEGORY=\"%{category}\" REASON=%{fld4}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg740 = msg("WEBFILTER_URL_PERMITTED:03", part770);

var part771 = match("MESSAGE#736:WEBFILTER_URL_PERMITTED:02", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=%{url}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg741 = msg("WEBFILTER_URL_PERMITTED:02", part771);

var select74 = linear_select([
	msg738,
	msg739,
	msg740,
	msg741,
]);

var msg742 = msg("WEBFILTER_URL_BLOCKED", dup158);

var part772 = match("MESSAGE#738:WEBFILTER_URL_BLOCKED:01", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=\"%{directory}\" username=\"%{username}\" roles=\"%{user_role}\"] WebFilter: ACTION=\"%{action}\" %{fld2}->%{fld3->} CATEGORY=\"%{category}\" REASON=\"%{fld4}\" PROFILE=\"%{fld6}\" URL=%{url}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var msg743 = msg("WEBFILTER_URL_BLOCKED:01", part772);

var select75 = linear_select([
	msg742,
	msg743,
]);

var part773 = match("MESSAGE#740:SECINTEL_NETWORK_CONNECT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{id}: \u003c\u003c%{fld12}> Access url %{url->} on port %{network_port->} failed\u003c\u003c%{result}>.", processor_chain([
	dup46,
	dup47,
	dup23,
	dup22,
	dup128,
]));

var msg744 = msg("SECINTEL_NETWORK_CONNECT_FAILED", part773);

var part774 = match("MESSAGE#741:AAMWD_NETWORK_CONNECT_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{id}: \u003c\u003c%{fld12}> Access host %{hostname->} on ip %{hostip->} port %{network_port->} %{result}.", processor_chain([
	dup46,
	dup47,
	dup23,
]));

var msg745 = msg("AAMWD_NETWORK_CONNECT_FAILED", part774);

var part775 = match("MESSAGE#742:PKID_UNABLE_TO_GET_CRL", "nwparser.payload", "%{process}[%{process_id}]: %{id}: Failed to retrieve CRL from received file for %{node}", processor_chain([
	dup46,
	dup47,
	dup23,
	dup22,
	dup128,
]));

var msg746 = msg("PKID_UNABLE_TO_GET_CRL", part775);

var part776 = match("MESSAGE#743:SECINTEL_ERROR_OTHERS", "nwparser.payload", "%{process}[%{process_id}]: %{id}: \u003c\u003c%{fld12}> %{result}", processor_chain([
	dup46,
	dup47,
	dup23,
	dup22,
	dup128,
]));

var msg747 = msg("SECINTEL_ERROR_OTHERS", part776);

var part777 = match("MESSAGE#744:JSRPD_HA_CONTROL_LINK_UP", "nwparser.payload", "%{process}[%{process_id}]: %{id}: HA control link monitor status is marked up", processor_chain([
	dup48,
	dup47,
	dup23,
	dup22,
	dup128,
]));

var msg748 = msg("JSRPD_HA_CONTROL_LINK_UP", part777);

var part778 = match("MESSAGE#745:LACPD_TIMEOUT", "nwparser.payload", "%{process}[%{process_id}]: LACPD_TIMEOUT: %{sinterface}: %{event_description}", processor_chain([
	dup46,
	dup47,
	dup23,
	dup22,
	dup128,
]));

var msg749 = msg("LACPD_TIMEOUT", part778);

var msg750 = msg("cli", dup159);

var msg751 = msg("pfed", dup159);

var msg752 = msg("idpinfo", dup159);

var msg753 = msg("kmd", dup159);

var part779 = match("MESSAGE#751:node:01", "nwparser.payload", "%{hostname->} %{node->} Next-hop resolution requests from interface %{interface->} throttled", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg754 = msg("node:01", part779);

var part780 = match("MESSAGE#752:node:02", "nwparser.payload", "%{hostname->} %{node->} %{process}: Trying peer connection, status %{resultcode}, attempt %{fld1}", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg755 = msg("node:02", part780);

var part781 = match("MESSAGE#753:node:03", "nwparser.payload", "%{hostname->} %{node->} %{process}: trying master connection, status %{resultcode}, attempt %{fld1}", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg756 = msg("node:03", part781);

var part782 = match("MESSAGE#754:node:04", "nwparser.payload", "%{hostname->} %{node->} %{fld1->} key %{fld2->} %{fld3->} port priority %{fld6->} %{fld4->} port %{portname->} %{fld5->} state %{resultcode}", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg757 = msg("node:04", part782);

var select76 = linear_select([
	dup131,
	dup132,
]);

var part783 = match("MESSAGE#755:node:05/2", "nwparser.p0", "%{}sys priority %{fld4->} %{p0}");

var select77 = linear_select([
	dup132,
	dup131,
]);

var part784 = match("MESSAGE#755:node:05/4", "nwparser.p0", "%{}sys %{interface}");

var all44 = all_match({
	processors: [
		dup130,
		select76,
		part783,
		select77,
		part784,
	],
	on_success: processor_chain([
		dup21,
		dup23,
		dup22,
	]),
});

var msg758 = msg("node:05", all44);

var part785 = match("MESSAGE#756:node:06/1_0", "nwparser.p0", "dst mac %{dinterface}");

var part786 = match("MESSAGE#756:node:06/1_1", "nwparser.p0", "src mac %{sinterface->} ether type %{fld1}");

var select78 = linear_select([
	part785,
	part786,
]);

var all45 = all_match({
	processors: [
		dup130,
		select78,
	],
	on_success: processor_chain([
		dup21,
		dup23,
		dup22,
	]),
});

var msg759 = msg("node:06", all45);

var part787 = match("MESSAGE#757:node:07", "nwparser.payload", "%{hostname->} %{node->} %{process}: interface %{interface->} trigger reth_scan", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg760 = msg("node:07", part787);

var part788 = match("MESSAGE#758:node:08", "nwparser.payload", "%{hostname->} %{node->} %{process}: %{info}", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg761 = msg("node:08", part788);

var part789 = match("MESSAGE#759:node:09", "nwparser.payload", "%{hostname->} %{node->} %{fld1}", processor_chain([
	dup21,
	dup23,
	dup22,
]));

var msg762 = msg("node:09", part789);

var select79 = linear_select([
	msg754,
	msg755,
	msg756,
	msg757,
	msg758,
	msg759,
	msg760,
	msg761,
	msg762,
]);

var part790 = match("MESSAGE#760:(FPC:01", "nwparser.payload", "%{fld1}) %{node->} kernel: %{event_type}: deleting active remote neighbor entry %{fld2->} from interface %{interface}.", processor_chain([
	dup21,
	dup23,
	dup22,
	dup24,
]));

var msg763 = msg("(FPC:01", part790);

var part791 = match("MESSAGE#761:(FPC:02", "nwparser.payload", "%{fld1}) %{node->} kernel: %{event_type->} deleting nb %{fld2->} on ifd %{interface->} for cid %{fld3->} from active neighbor table", processor_chain([
	dup21,
	dup23,
	dup22,
	dup24,
]));

var msg764 = msg("(FPC:02", part791);

var part792 = match("MESSAGE#762:(FPC:03/0", "nwparser.payload", "%{fld1}) %{node->} kernel: %{event_type}: M%{p0}");

var part793 = match("MESSAGE#762:(FPC:03/1_0", "nwparser.p0", "DOWN %{p0}");

var part794 = match("MESSAGE#762:(FPC:03/1_1", "nwparser.p0", "UP %{p0}");

var select80 = linear_select([
	part793,
	part794,
]);

var part795 = match("MESSAGE#762:(FPC:03/2", "nwparser.p0", "received for interface %{interface}, member of %{fld4}");

var all46 = all_match({
	processors: [
		part792,
		select80,
		part795,
	],
	on_success: processor_chain([
		dup21,
		dup23,
		dup22,
		dup24,
	]),
});

var msg765 = msg("(FPC:03", all46);

var part796 = match("MESSAGE#763:(FPC:04", "nwparser.payload", "%{fld1}) %{node->} kernel: %{event_type}: ifd=%{interface}, ifd flags=%{fld2}", processor_chain([
	dup21,
	dup23,
	dup22,
	dup24,
]));

var msg766 = msg("(FPC:04", part796);

var part797 = match("MESSAGE#764:(FPC:05", "nwparser.payload", "%{fld1}) %{node->} kernel: rdp keepalive expired, connection dropped - src %{fld3}:%{fld2->} dest %{fld4}:%{fld5}", processor_chain([
	dup21,
	dup23,
	dup22,
	dup24,
]));

var msg767 = msg("(FPC:05", part797);

var part798 = match("MESSAGE#765:(FPC", "nwparser.payload", "%{fld1}) %{node->} %{fld10}", processor_chain([
	dup21,
	dup23,
	dup22,
	dup24,
]));

var msg768 = msg("(FPC", part798);

var select81 = linear_select([
	msg763,
	msg764,
	msg765,
	msg766,
	msg767,
	msg768,
]);

var part799 = match("MESSAGE#766:tnp.bootpd", "nwparser.payload", "%{process}[%{process_id}]:%{fld1}", processor_chain([
	dup48,
	dup23,
	dup22,
	dup24,
]));

var msg769 = msg("tnp.bootpd", part799);

var part800 = match("MESSAGE#769:AAMW_ACTION_LOG", "nwparser.payload", "%{event_type}[junos@%{fld32->} hostname=\"%{hostname}\" file-category=\"%{fld9}\" verdict-number=\"%{fld10}\" action=\"%{action}\" list-hit=\"%{fld19}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-id=\"%{protocol}\" application=\"%{fld6}\" nested-application=\"%{fld7}\" policy-name=\"%{policyname}\" username=\"%{username}\" roles=\"%{user_role}\" session-id-32=\"%{sessionid}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" url=\"%{url}\"] %{fld27}", processor_chain([
	dup48,
	dup52,
	dup22,
	dup61,
]));

var msg770 = msg("AAMW_ACTION_LOG", part800);

var part801 = match("MESSAGE#770:AAMW_HOST_INFECTED_EVENT_LOG", "nwparser.payload", "%{event_type}[junos@%{fld32->} timestamp=\"%{fld30}\" tenant-id=\"%{fld1}\" client-ip-str=\"%{hostip}\" hostname=\"%{hostname}\" status=\"%{fld13}\" policy-name=\"%{policyname}\" verdict-number=\"%{fld15}\" state=\"%{fld16}\" reason=\"%{result}\" message=\"%{info}\" %{fld3}", processor_chain([
	dup133,
	dup52,
	dup22,
	dup61,
]));

var msg771 = msg("AAMW_HOST_INFECTED_EVENT_LOG", part801);

var part802 = match("MESSAGE#771:AAMW_MALWARE_EVENT_LOG", "nwparser.payload", "%{event_type}[junos@%{fld32->} timestamp=\"%{fld30}\" tenant-id=\"%{fld1}\" sample-sha256=\"%{checksum}\" client-ip-str=\"%{hostip}\" verdict-number=\"%{fld26}\" malware-info=\"%{threat_name}\" username=\"%{username}\" hostname=\"%{hostname}\" %{fld3}", processor_chain([
	dup133,
	dup52,
	dup22,
]));

var msg772 = msg("AAMW_MALWARE_EVENT_LOG", part802);

var part803 = match("MESSAGE#772:IDP_ATTACK_LOG_EVENT", "nwparser.payload", "%{event_type}[junos@%{fld32->} epoch-time=\"%{fld1}\" message-type=\"%{info}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-name=\"%{protocol}\" service-name=\"%{service}\" application-name=\"%{application}\" rule-name=\"%{fld5}\" rulebase-name=\"%{rulename}\" policy-name=\"%{policyname}\" export-id=\"%{fld6}\" repeat-count=\"%{fld7}\" action=\"%{action}\" threat-severity=\"%{severity}\" attack-name=\"%{threat_name}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" elapsed-time=%{fld8->} inbound-bytes=\"%{rbytes}\" outbound-bytes=\"%{sbytes}\" inbound-packets=\"%{packets}\" outbound-packets=\"%{dclass_counter1}\" source-zone-name=\"%{src_zone}\" source-interface-name=\"%{sinterface}\" destination-zone-name=\"%{dst_zone}\" destination-interface-name=\"%{dinterface}\" packet-log-id=\"%{fld9}\" alert=\"%{fld19}\" username=\"%{username}\" roles=\"%{fld15}\" message=\"%{fld28}\" %{fld3}", processor_chain([
	dup81,
	dup52,
	dup22,
	dup61,
]));

var msg773 = msg("IDP_ATTACK_LOG_EVENT", part803);

var part804 = match("MESSAGE#773:RT_SCREEN_ICMP", "nwparser.payload", "%{event_type}[junos@%{fld32->} attack-name=\"%{threat_name}\" source-address=\"%{saddr}\" destination-address=\"%{daddr}\" source-zone-name=\"%{src_zone}\" interface-name=\"%{interface}\" action=\"%{action}\"] %{fld23}", processor_chain([
	dup81,
	dup52,
	dup22,
	dup61,
]));

var msg774 = msg("RT_SCREEN_ICMP", part804);

var part805 = match("MESSAGE#774:SECINTEL_ACTION_LOG", "nwparser.payload", "%{event_type}[junos@%{fld32->} category=\"%{fld1}\" sub-category=\"%{fld2}\" action=\"%{action}\" action-detail=\"%{fld4}\" http-host=\"%{fld17}\" threat-severity=\"%{severity}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" protocol-id=\"%{protocol}\" application=\"%{fld5}\" nested-application=\"%{fld6}\" feed-name=\"%{fld18}\" policy-name=\"%{policyname}\" profile-name=\"%{rulename}\" username=\"%{username}\" roles=\"%{user_role}\" session-id-32=\"%{sessionid}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\"]%{fld10}", processor_chain([
	dup46,
	dup52,
	dup22,
	dup61,
]));

var msg775 = msg("SECINTEL_ACTION_LOG", part805);

var part806 = match("MESSAGE#775:qsfp/0", "nwparser.payload", "%{hostname->} %{fld2->} %{p0}");

var part807 = match("MESSAGE#775:qsfp/1_0", "nwparser.p0", "%{fld3->} %{process}: qsfp-%{p0}");

var part808 = match("MESSAGE#775:qsfp/1_1", "nwparser.p0", "qsfp-%{p0}");

var select82 = linear_select([
	part807,
	part808,
]);

var part809 = match("MESSAGE#775:qsfp/2", "nwparser.p0", "%{}Chan# %{interface->} %{fld5}:%{event_description}");

var all47 = all_match({
	processors: [
		part806,
		select82,
		part809,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup23,
	]),
});

var msg776 = msg("qsfp", all47);

var part810 = match("MESSAGE#776:JUNOSROUTER_GENERIC:03", "nwparser.payload", "%{event_type}: User '%{username}', command '%{action}'", processor_chain([
	dup21,
	dup22,
	dup121,
	dup23,
]));

var msg777 = msg("JUNOSROUTER_GENERIC:03", part810);

var part811 = match("MESSAGE#777:JUNOSROUTER_GENERIC:04", "nwparser.payload", "%{event_type}: User '%{username}' %{fld1}", processor_chain([
	dup125,
	dup34,
	dup35,
	dup126,
	dup37,
	dup22,
	setc("event_description","LOGOUT"),
	dup23,
]));

var msg778 = msg("JUNOSROUTER_GENERIC:04", part811);

var part812 = match("MESSAGE#778:JUNOSROUTER_GENERIC:05", "nwparser.payload", "%{event_type}: TACACS+ failure: %{result}", processor_chain([
	dup30,
	dup22,
	dup129,
	dup23,
]));

var msg779 = msg("JUNOSROUTER_GENERIC:05", part812);

var part813 = match("MESSAGE#779:JUNOSROUTER_GENERIC:06", "nwparser.payload", "%{event_type}: mismatch NLRI with %{hostip->} (%{hostname}): peer: %{daddr->} us: %{saddr}", processor_chain([
	dup30,
	dup22,
	dup57,
	dup23,
]));

var msg780 = msg("JUNOSROUTER_GENERIC:06", part813);

var part814 = match("MESSAGE#780:JUNOSROUTER_GENERIC:07", "nwparser.payload", "%{event_type}: NOTIFICATION sent to %{daddr->} (%{dhost}): code %{resultcode->} (%{action}), Reason: %{result}", processor_chain([
	dup21,
	dup22,
	dup38,
	dup23,
]));

var msg781 = msg("JUNOSROUTER_GENERIC:07", part814);

var part815 = match("MESSAGE#781:JUNOSROUTER_GENERIC:08/0", "nwparser.payload", "%{event_type}: NOTIFICATION received from %{daddr->} (%{dhost}): code %{resultcode->} (%{action})%{p0}");

var part816 = match("MESSAGE#781:JUNOSROUTER_GENERIC:08/1_0", "nwparser.p0", ", socket buffer sndcc: %{fld1->} rcvcc: %{fld2->} TCP state: %{event_state}, snd_una: %{fld3->} snd_nxt: %{fld4->} snd_wnd: %{fld5->} rcv_nxt: %{fld6->} rcv_adv: %{fld7}, hold timer %{fld8}");

var part817 = match_copy("MESSAGE#781:JUNOSROUTER_GENERIC:08/1_1", "nwparser.p0", "");

var select83 = linear_select([
	part816,
	part817,
]);

var all48 = all_match({
	processors: [
		part815,
		select83,
	],
	on_success: processor_chain([
		dup21,
		dup22,
		dup38,
		dup23,
	]),
});

var msg782 = msg("JUNOSROUTER_GENERIC:08", all48);

var part818 = match("MESSAGE#782:JUNOSROUTER_GENERIC:09", "nwparser.payload", "%{event_type}: [edit interfaces%{interface}unit%{fld1}family inet address%{hostip}/%{network_port}] :%{event_description}:%{info}", processor_chain([
	dup21,
	dup22,
	dup23,
]));

var msg783 = msg("JUNOSROUTER_GENERIC:09", part818);

var part819 = match("MESSAGE#783:JUNOSROUTER_GENERIC:01", "nwparser.payload", "%{event_type->} Interface Monitor failed %{fld1}", processor_chain([
	dup134,
	dup23,
	dup22,
	setc("event_description","Interface Monitor failed "),
	dup24,
]));

var msg784 = msg("JUNOSROUTER_GENERIC:01", part819);

var part820 = match("MESSAGE#784:JUNOSROUTER_GENERIC:02", "nwparser.payload", "%{event_type->} Interface Monitor failure recovered %{fld1}", processor_chain([
	dup134,
	dup23,
	dup22,
	setc("event_description","Interface Monitor failure recovered"),
	dup24,
]));

var msg785 = msg("JUNOSROUTER_GENERIC:02", part820);

var part821 = match("MESSAGE#785:JUNOSROUTER_GENERIC", "nwparser.payload", "%{event_type->} %{fld1}", processor_chain([
	dup134,
	dup23,
	dup22,
	dup24,
]));

var msg786 = msg("JUNOSROUTER_GENERIC", part821);

var select84 = linear_select([
	msg777,
	msg778,
	msg779,
	msg780,
	msg781,
	msg782,
	msg783,
	msg784,
	msg785,
	msg786,
]);

var chain1 = processor_chain([
	select5,
	msgid_select({
		"(FPC": select81,
		"/usr/libexec/telnetd": msg2,
		"/usr/sbin/cron": msg734,
		"/usr/sbin/sshd": msg1,
		"AAMWD_NETWORK_CONNECT_FAILED": msg745,
		"AAMW_ACTION_LOG": msg770,
		"AAMW_HOST_INFECTED_EVENT_LOG": msg771,
		"AAMW_MALWARE_EVENT_LOG": msg772,
		"ACCT_ACCOUNTING_FERROR": msg114,
		"ACCT_ACCOUNTING_FOPEN_ERROR": msg115,
		"ACCT_ACCOUNTING_SMALL_FILE_SIZE": msg116,
		"ACCT_BAD_RECORD_FORMAT": msg117,
		"ACCT_CU_RTSLIB_error": msg118,
		"ACCT_GETHOSTNAME_error": msg119,
		"ACCT_MALLOC_FAILURE": msg120,
		"ACCT_UNDEFINED_COUNTER_NAME": msg121,
		"ACCT_XFER_FAILED": msg122,
		"ACCT_XFER_POPEN_FAIL": msg123,
		"APPQOS_LOG_EVENT": msg124,
		"APPTRACK_SESSION_CLOSE": select30,
		"APPTRACK_SESSION_CREATE": msg125,
		"APPTRACK_SESSION_VOL_UPDATE": select31,
		"BCHIP": msg106,
		"BFDD_TRAP_STATE_DOWN": msg130,
		"BFDD_TRAP_STATE_UP": msg131,
		"BOOTPD_ARG_ERR": msg143,
		"BOOTPD_BAD_ID": msg144,
		"BOOTPD_BOOTSTRING": msg145,
		"BOOTPD_CONFIG_ERR": msg146,
		"BOOTPD_CONF_OPEN": msg147,
		"BOOTPD_DUP_REV": msg148,
		"BOOTPD_DUP_SLOT": msg149,
		"BOOTPD_MODEL_CHK": msg150,
		"BOOTPD_MODEL_ERR": msg151,
		"BOOTPD_NEW_CONF": msg152,
		"BOOTPD_NO_BOOTSTRING": msg153,
		"BOOTPD_NO_CONFIG": msg154,
		"BOOTPD_PARSE_ERR": msg155,
		"BOOTPD_REPARSE": msg156,
		"BOOTPD_SELECT_ERR": msg157,
		"BOOTPD_TIMEOUT": msg158,
		"BOOTPD_VERSION": msg159,
		"CHASSISD": msg160,
		"CHASSISD_ARGUMENT_ERROR": msg161,
		"CHASSISD_BLOWERS_SPEED": msg162,
		"CHASSISD_BLOWERS_SPEED_FULL": msg163,
		"CHASSISD_CB_READ": msg164,
		"CHASSISD_COMMAND_ACK_ERROR": msg165,
		"CHASSISD_COMMAND_ACK_SF_ERROR": msg166,
		"CHASSISD_CONCAT_MODE_ERROR": msg167,
		"CHASSISD_CONFIG_INIT_ERROR": msg168,
		"CHASSISD_CONFIG_WARNING": msg169,
		"CHASSISD_EXISTS": msg170,
		"CHASSISD_EXISTS_TERM_OTHER": msg171,
		"CHASSISD_FILE_OPEN": msg172,
		"CHASSISD_FILE_STAT": msg173,
		"CHASSISD_FRU_EVENT": msg174,
		"CHASSISD_FRU_IPC_WRITE_ERROR_EXT": msg175,
		"CHASSISD_FRU_STEP_ERROR": msg176,
		"CHASSISD_GETTIMEOFDAY": msg177,
		"CHASSISD_HIGH_TEMP_CONDITION": msg214,
		"CHASSISD_HOST_TEMP_READ": msg178,
		"CHASSISD_IFDEV_DETACH_ALL_PSEUDO": msg179,
		"CHASSISD_IFDEV_DETACH_FPC": msg180,
		"CHASSISD_IFDEV_DETACH_PIC": msg181,
		"CHASSISD_IFDEV_DETACH_PSEUDO": msg182,
		"CHASSISD_IFDEV_DETACH_TLV_ERROR": msg183,
		"CHASSISD_IFDEV_GET_BY_INDEX_FAIL": msg184,
		"CHASSISD_IPC_MSG_QFULL_ERROR": msg185,
		"CHASSISD_IPC_UNEXPECTED_RECV": msg186,
		"CHASSISD_IPC_WRITE_ERR_NO_PIPE": msg187,
		"CHASSISD_IPC_WRITE_ERR_NULL_ARGS": msg188,
		"CHASSISD_MAC_ADDRESS_ERROR": msg189,
		"CHASSISD_MAC_DEFAULT": msg190,
		"CHASSISD_MBUS_ERROR": msg191,
		"CHASSISD_PARSE_COMPLETE": msg192,
		"CHASSISD_PARSE_ERROR": msg193,
		"CHASSISD_PARSE_INIT": msg194,
		"CHASSISD_PIDFILE_OPEN": msg195,
		"CHASSISD_PIPE_WRITE_ERROR": msg196,
		"CHASSISD_POWER_CHECK": msg197,
		"CHASSISD_RECONNECT_SUCCESSFUL": msg198,
		"CHASSISD_RELEASE_MASTERSHIP": msg199,
		"CHASSISD_RE_INIT_INVALID_RE_SLOT": msg200,
		"CHASSISD_ROOT_MOUNT_ERROR": msg201,
		"CHASSISD_RTS_SEQ_ERROR": msg202,
		"CHASSISD_SBOARD_VERSION_MISMATCH": msg203,
		"CHASSISD_SERIAL_ID": msg204,
		"CHASSISD_SMB_ERROR": msg205,
		"CHASSISD_SNMP_TRAP10": msg208,
		"CHASSISD_SNMP_TRAP6": msg206,
		"CHASSISD_SNMP_TRAP7": msg207,
		"CHASSISD_TERM_SIGNAL": msg209,
		"CHASSISD_TRACE_PIC_OFFLINE": msg210,
		"CHASSISD_UNEXPECTED_EXIT": msg211,
		"CHASSISD_UNSUPPORTED_MODEL": msg212,
		"CHASSISD_VERSION_MISMATCH": msg213,
		"CM": msg107,
		"CM_JAVA": msg216,
		"COS": msg108,
		"COSFPC": msg109,
		"COSMAN": msg110,
		"CRON": msg16,
		"CROND": select11,
		"Cmerror": msg17,
		"DCD_AS_ROOT": msg217,
		"DCD_FILTER_LIB_ERROR": msg218,
		"DCD_MALLOC_FAILED_INIT": msg219,
		"DCD_PARSE_EMERGENCY": msg220,
		"DCD_PARSE_FILTER_EMERGENCY": msg221,
		"DCD_PARSE_MINI_EMERGENCY": msg222,
		"DCD_PARSE_STATE_EMERGENCY": msg223,
		"DCD_POLICER_PARSE_EMERGENCY": msg224,
		"DCD_PULL_LOG_FAILURE": msg225,
		"DFWD_ARGUMENT_ERROR": msg226,
		"DFWD_MALLOC_FAILED_INIT": msg227,
		"DFWD_PARSE_FILTER_EMERGENCY": msg228,
		"DFWD_PARSE_STATE_EMERGENCY": msg229,
		"ECCD_DAEMONIZE_FAILED": msg230,
		"ECCD_DUPLICATE": msg231,
		"ECCD_LOOP_EXIT_FAILURE": msg232,
		"ECCD_NOT_ROOT": msg233,
		"ECCD_PCI_FILE_OPEN_FAILED": msg234,
		"ECCD_PCI_READ_FAILED": msg235,
		"ECCD_PCI_WRITE_FAILED": msg236,
		"ECCD_PID_FILE_LOCK": msg237,
		"ECCD_PID_FILE_UPDATE": msg238,
		"ECCD_TRACE_FILE_OPEN_FAILED": msg239,
		"ECCD_usage": msg240,
		"EVENT": msg23,
		"EVENTD_AUDIT_SHOW": msg241,
		"FLOW_REASSEMBLE_FAIL": msg731,
		"FLOW_REASSEMBLE_SUCCEED": msg242,
		"FSAD_CHANGE_FILE_OWNER": msg243,
		"FSAD_CONFIG_ERROR": msg244,
		"FSAD_CONNTIMEDOUT": msg245,
		"FSAD_FAILED": msg246,
		"FSAD_FETCHTIMEDOUT": msg247,
		"FSAD_FILE_FAILED": msg248,
		"FSAD_FILE_REMOVE": msg249,
		"FSAD_FILE_RENAME": msg250,
		"FSAD_FILE_STAT": msg251,
		"FSAD_FILE_SYNC": msg252,
		"FSAD_MAXCONN": msg253,
		"FSAD_MEMORYALLOC_FAILED": msg254,
		"FSAD_NOT_ROOT": msg255,
		"FSAD_PARENT_DIRECTORY": msg256,
		"FSAD_PATH_IS_DIRECTORY": msg257,
		"FSAD_PATH_IS_SPECIAL": msg258,
		"FSAD_RECVERROR": msg259,
		"FSAD_TERMINATED_CONNECTION": msg260,
		"FSAD_TERMINATING_SIGNAL": msg261,
		"FSAD_TRACEOPEN_FAILED": msg262,
		"FSAD_USAGE": msg263,
		"Failed": select25,
		"GGSN_ALARM_TRAP_FAILED": msg264,
		"GGSN_ALARM_TRAP_SEND": msg265,
		"GGSN_TRAP_SEND": msg266,
		"IDP_ATTACK_LOG_EVENT": msg773,
		"JADE_AUTH_ERROR": msg267,
		"JADE_EXEC_ERROR": msg268,
		"JADE_NO_LOCAL_USER": msg269,
		"JADE_PAM_ERROR": msg270,
		"JADE_PAM_NO_LOCAL_USER": msg271,
		"JSRPD_HA_CONTROL_LINK_UP": msg748,
		"JUNOSROUTER_GENERIC": select84,
		"KERN_ARP_ADDR_CHANGE": msg272,
		"KMD_PM_SA_ESTABLISHED": msg273,
		"L2CPD_TASK_REINIT": msg274,
		"LACPD_TIMEOUT": msg749,
		"LIBJNX_EXEC_EXITED": msg275,
		"LIBJNX_EXEC_FAILED": msg276,
		"LIBJNX_EXEC_PIPE": msg277,
		"LIBJNX_EXEC_SIGNALED": msg278,
		"LIBJNX_EXEC_WEXIT": msg279,
		"LIBJNX_FILE_COPY_FAILED": msg280,
		"LIBJNX_PRIV_LOWER_FAILED": msg281,
		"LIBJNX_PRIV_RAISE_FAILED": msg282,
		"LIBJNX_REPLICATE_RCP_EXEC_FAILED": msg283,
		"LIBJNX_ROTATE_COMPRESS_EXEC_FAILED": msg284,
		"LIBSERVICED_CLIENT_CONNECTION": msg285,
		"LIBSERVICED_OUTBOUND_REQUEST": msg286,
		"LIBSERVICED_SNMP_LOST_CONNECTION": msg287,
		"LIBSERVICED_SOCKET_BIND": msg288,
		"LIBSERVICED_SOCKET_PRIVATIZE": msg289,
		"LICENSE_EXPIRED": msg290,
		"LICENSE_EXPIRED_KEY_DELETED": msg291,
		"LICENSE_NEARING_EXPIRY": msg292,
		"LOGIN_ABORTED": msg293,
		"LOGIN_FAILED": msg294,
		"LOGIN_FAILED_INCORRECT_PASSWORD": msg295,
		"LOGIN_FAILED_SET_CONTEXT": msg296,
		"LOGIN_FAILED_SET_LOGIN": msg297,
		"LOGIN_HOSTNAME_UNRESOLVED": msg298,
		"LOGIN_INFORMATION": msg299,
		"LOGIN_INVALID_LOCAL_USER": msg300,
		"LOGIN_MALFORMED_USER": msg301,
		"LOGIN_PAM_AUTHENTICATION_ERROR": msg302,
		"LOGIN_PAM_ERROR": msg303,
		"LOGIN_PAM_MAX_RETRIES": msg304,
		"LOGIN_PAM_NONLOCAL_USER": msg305,
		"LOGIN_PAM_STOP": msg306,
		"LOGIN_PAM_USER_UNKNOWN": msg307,
		"LOGIN_PASSWORD_EXPIRED": msg308,
		"LOGIN_REFUSED": msg309,
		"LOGIN_ROOT": msg310,
		"LOGIN_TIMED_OUT": msg311,
		"MIB2D_ATM_ERROR": msg312,
		"MIB2D_CONFIG_CHECK_FAILED": msg313,
		"MIB2D_FILE_OPEN_FAILURE": msg314,
		"MIB2D_IFD_IFINDEX_FAILURE": msg315,
		"MIB2D_IFL_IFINDEX_FAILURE": msg316,
		"MIB2D_INIT_FAILURE": msg317,
		"MIB2D_KVM_FAILURE": msg318,
		"MIB2D_RTSLIB_READ_FAILURE": msg319,
		"MIB2D_RTSLIB_SEQ_MISMATCH": msg320,
		"MIB2D_SYSCTL_FAILURE": msg321,
		"MIB2D_TRAP_HEADER_FAILURE": msg322,
		"MIB2D_TRAP_SEND_FAILURE": msg323,
		"MRVL-L2": msg56,
		"Multiuser": msg324,
		"NASD_AUTHENTICATION_CREATE_FAILED": msg325,
		"NASD_CHAP_AUTHENTICATION_IN_PROGRESS": msg326,
		"NASD_CHAP_GETHOSTNAME_FAILED": msg327,
		"NASD_CHAP_INVALID_CHAP_IDENTIFIER": msg328,
		"NASD_CHAP_INVALID_OPCODE": msg329,
		"NASD_CHAP_LOCAL_NAME_UNAVAILABLE": msg330,
		"NASD_CHAP_MESSAGE_UNEXPECTED": msg331,
		"NASD_CHAP_REPLAY_ATTACK_DETECTED": msg332,
		"NASD_CONFIG_GET_LAST_MODIFIED_FAILED": msg333,
		"NASD_DAEMONIZE_FAILED": msg334,
		"NASD_DB_ALLOC_FAILURE": msg335,
		"NASD_DB_TABLE_CREATE_FAILURE": msg336,
		"NASD_DUPLICATE": msg337,
		"NASD_EVLIB_CREATE_FAILURE": msg338,
		"NASD_EVLIB_EXIT_FAILURE": msg339,
		"NASD_LOCAL_CREATE_FAILED": msg340,
		"NASD_NOT_ROOT": msg341,
		"NASD_PID_FILE_LOCK": msg342,
		"NASD_PID_FILE_UPDATE": msg343,
		"NASD_POST_CONFIGURE_EVENT_FAILED": msg344,
		"NASD_PPP_READ_FAILURE": msg345,
		"NASD_PPP_SEND_FAILURE": msg346,
		"NASD_PPP_SEND_PARTIAL": msg347,
		"NASD_PPP_UNRECOGNIZED": msg348,
		"NASD_RADIUS_ALLOCATE_PASSWORD_FAILED": msg349,
		"NASD_RADIUS_CONFIG_FAILED": msg350,
		"NASD_RADIUS_CREATE_FAILED": msg351,
		"NASD_RADIUS_CREATE_REQUEST_FAILED": msg352,
		"NASD_RADIUS_GETHOSTNAME_FAILED": msg353,
		"NASD_RADIUS_MESSAGE_UNEXPECTED": msg354,
		"NASD_RADIUS_OPEN_FAILED": msg355,
		"NASD_RADIUS_SELECT_FAILED": msg356,
		"NASD_RADIUS_SET_TIMER_FAILED": msg357,
		"NASD_TRACE_FILE_OPEN_FAILED": msg358,
		"NASD_usage": msg359,
		"NOTICE": msg360,
		"PFEMAN": msg61,
		"PFE_FW_SYSLOG_IP": select36,
		"PFE_NH_RESOLVE_THROTTLED": msg363,
		"PING_TEST_COMPLETED": msg364,
		"PING_TEST_FAILED": msg365,
		"PKID_UNABLE_TO_GET_CRL": msg746,
		"PWC_EXIT": msg368,
		"PWC_HOLD_RELEASE": msg369,
		"PWC_INVALID_RUNS_ARGUMENT": msg370,
		"PWC_INVALID_TIMEOUT_ARGUMENT": msg371,
		"PWC_KILLED_BY_SIGNAL": msg372,
		"PWC_KILL_EVENT": msg373,
		"PWC_KILL_FAILED": msg374,
		"PWC_KQUEUE_ERROR": msg375,
		"PWC_KQUEUE_INIT": msg376,
		"PWC_KQUEUE_REGISTER_FILTER": msg377,
		"PWC_LOCKFILE_BAD_FORMAT": msg378,
		"PWC_LOCKFILE_ERROR": msg379,
		"PWC_LOCKFILE_MISSING": msg380,
		"PWC_LOCKFILE_NOT_LOCKED": msg381,
		"PWC_NO_PROCESS": msg382,
		"PWC_PROCESS_EXIT": msg383,
		"PWC_PROCESS_FORCED_HOLD": msg384,
		"PWC_PROCESS_HOLD": msg385,
		"PWC_PROCESS_HOLD_SKIPPED": msg386,
		"PWC_PROCESS_OPEN": msg387,
		"PWC_PROCESS_TIMED_HOLD": msg388,
		"PWC_PROCESS_TIMEOUT": msg389,
		"PWC_SIGNAL_INIT": msg390,
		"PWC_SOCKET_CONNECT": msg391,
		"PWC_SOCKET_CREATE": msg392,
		"PWC_SOCKET_OPTION": msg393,
		"PWC_STDOUT_WRITE": msg394,
		"PWC_SYSTEM_CALL": msg395,
		"PWC_UNKNOWN_KILL_OPTION": msg396,
		"RDP": msg111,
		"RMOPD_ADDRESS_MULTICAST_INVALID": msg397,
		"RMOPD_ADDRESS_SOURCE_INVALID": msg398,
		"RMOPD_ADDRESS_STRING_FAILURE": msg399,
		"RMOPD_ADDRESS_TARGET_INVALID": msg400,
		"RMOPD_DUPLICATE": msg401,
		"RMOPD_ICMP_ADDRESS_TYPE_UNSUPPORTED": msg402,
		"RMOPD_ICMP_SENDMSG_FAILURE": msg403,
		"RMOPD_IFINDEX_NOT_ACTIVE": msg404,
		"RMOPD_IFINDEX_NO_INFO": msg405,
		"RMOPD_IFNAME_NOT_ACTIVE": msg406,
		"RMOPD_IFNAME_NO_INFO": msg407,
		"RMOPD_NOT_ROOT": msg408,
		"RMOPD_ROUTING_INSTANCE_NO_INFO": msg409,
		"RMOPD_TRACEROUTE_ERROR": msg410,
		"RMOPD_usage": msg411,
		"RPD_ABORT": msg412,
		"RPD_ACTIVE_TERMINATE": msg413,
		"RPD_ASSERT": msg414,
		"RPD_ASSERT_SOFT": msg415,
		"RPD_EXIT": msg416,
		"RPD_IFL_INDEXCOLLISION": msg417,
		"RPD_IFL_NAMECOLLISION": msg418,
		"RPD_ISIS_ADJDOWN": msg419,
		"RPD_ISIS_ADJUP": msg420,
		"RPD_ISIS_ADJUPNOIP": msg421,
		"RPD_ISIS_LSPCKSUM": msg422,
		"RPD_ISIS_OVERLOAD": msg423,
		"RPD_KRT_AFUNSUPRT": msg424,
		"RPD_KRT_CCC_IFL_MODIFY": msg425,
		"RPD_KRT_DELETED_RTT": msg426,
		"RPD_KRT_IFA_GENERATION": msg427,
		"RPD_KRT_IFDCHANGE": msg428,
		"RPD_KRT_IFDEST_GET": msg429,
		"RPD_KRT_IFDGET": msg430,
		"RPD_KRT_IFD_GENERATION": msg431,
		"RPD_KRT_IFL_CELL_RELAY_MODE_INVALID": msg432,
		"RPD_KRT_IFL_CELL_RELAY_MODE_UNSPECIFIED": msg433,
		"RPD_KRT_IFL_GENERATION": msg434,
		"RPD_KRT_KERNEL_BAD_ROUTE": msg435,
		"RPD_KRT_NEXTHOP_OVERFLOW": msg436,
		"RPD_KRT_NOIFD": msg437,
		"RPD_KRT_UNKNOWN_RTT": msg438,
		"RPD_KRT_VERSION": msg439,
		"RPD_KRT_VERSIONNONE": msg440,
		"RPD_KRT_VERSIONOLD": msg441,
		"RPD_LDP_INTF_BLOCKED": msg442,
		"RPD_LDP_INTF_UNBLOCKED": msg443,
		"RPD_LDP_NBRDOWN": msg444,
		"RPD_LDP_NBRUP": msg445,
		"RPD_LDP_SESSIONDOWN": msg446,
		"RPD_LDP_SESSIONUP": msg447,
		"RPD_LOCK_FLOCKED": msg448,
		"RPD_LOCK_LOCKED": msg449,
		"RPD_MPLS_LSP_CHANGE": msg450,
		"RPD_MPLS_LSP_DOWN": msg451,
		"RPD_MPLS_LSP_SWITCH": msg452,
		"RPD_MPLS_LSP_UP": msg453,
		"RPD_MSDP_PEER_DOWN": msg454,
		"RPD_MSDP_PEER_UP": msg455,
		"RPD_OSPF_NBRDOWN": msg456,
		"RPD_OSPF_NBRUP": msg457,
		"RPD_OS_MEMHIGH": msg458,
		"RPD_PIM_NBRDOWN": msg459,
		"RPD_PIM_NBRUP": msg460,
		"RPD_RDISC_CKSUM": msg461,
		"RPD_RDISC_NOMULTI": msg462,
		"RPD_RDISC_NORECVIF": msg463,
		"RPD_RDISC_SOLICITADDR": msg464,
		"RPD_RDISC_SOLICITICMP": msg465,
		"RPD_RDISC_SOLICITLEN": msg466,
		"RPD_RIP_AUTH": msg467,
		"RPD_RIP_JOIN_BROADCAST": msg468,
		"RPD_RIP_JOIN_MULTICAST": msg469,
		"RPD_RT_IFUP": msg470,
		"RPD_SCHED_CALLBACK_LONGRUNTIME": msg471,
		"RPD_SCHED_CUMULATIVE_LONGRUNTIME": msg472,
		"RPD_SCHED_MODULE_LONGRUNTIME": msg473,
		"RPD_SCHED_TASK_LONGRUNTIME": msg474,
		"RPD_SIGNAL_TERMINATE": msg475,
		"RPD_START": msg476,
		"RPD_SYSTEM": msg477,
		"RPD_TASK_BEGIN": msg478,
		"RPD_TASK_CHILDKILLED": msg479,
		"RPD_TASK_CHILDSTOPPED": msg480,
		"RPD_TASK_FORK": msg481,
		"RPD_TASK_GETWD": msg482,
		"RPD_TASK_NOREINIT": msg483,
		"RPD_TASK_PIDCLOSED": msg484,
		"RPD_TASK_PIDFLOCK": msg485,
		"RPD_TASK_PIDWRITE": msg486,
		"RPD_TASK_REINIT": msg487,
		"RPD_TASK_SIGNALIGNORE": msg488,
		"RT_COS": msg489,
		"RT_FLOW_SESSION_CLOSE": select51,
		"RT_FLOW_SESSION_CREATE": select45,
		"RT_FLOW_SESSION_DENY": select47,
		"RT_SCREEN_ICMP": msg774,
		"RT_SCREEN_IP": select52,
		"RT_SCREEN_SESSION_LIMIT": msg504,
		"RT_SCREEN_TCP": msg503,
		"RT_SCREEN_UDP": msg505,
		"Resolve": msg63,
		"SECINTEL_ACTION_LOG": msg775,
		"SECINTEL_ERROR_OTHERS": msg747,
		"SECINTEL_NETWORK_CONNECT_FAILED": msg744,
		"SERVICED_CLIENT_CONNECT": msg506,
		"SERVICED_CLIENT_DISCONNECTED": msg507,
		"SERVICED_CLIENT_ERROR": msg508,
		"SERVICED_COMMAND_FAILED": msg509,
		"SERVICED_COMMIT_FAILED": msg510,
		"SERVICED_CONFIGURATION_FAILED": msg511,
		"SERVICED_CONFIG_ERROR": msg512,
		"SERVICED_CONFIG_FILE": msg513,
		"SERVICED_CONNECTION_ERROR": msg514,
		"SERVICED_DISABLED_GGSN": msg515,
		"SERVICED_DUPLICATE": msg516,
		"SERVICED_EVENT_FAILED": msg517,
		"SERVICED_INIT_FAILED": msg518,
		"SERVICED_MALLOC_FAILURE": msg519,
		"SERVICED_NETWORK_FAILURE": msg520,
		"SERVICED_NOT_ROOT": msg521,
		"SERVICED_PID_FILE_LOCK": msg522,
		"SERVICED_PID_FILE_UPDATE": msg523,
		"SERVICED_RTSOCK_SEQUENCE": msg524,
		"SERVICED_SIGNAL_HANDLER": msg525,
		"SERVICED_SOCKET_CREATE": msg526,
		"SERVICED_SOCKET_IO": msg527,
		"SERVICED_SOCKET_OPTION": msg528,
		"SERVICED_STDLIB_FAILURE": msg529,
		"SERVICED_USAGE": msg530,
		"SERVICED_WORK_INCONSISTENCY": msg531,
		"SNMPD_ACCESS_GROUP_ERROR": msg537,
		"SNMPD_AUTH_FAILURE": select53,
		"SNMPD_AUTH_PRIVILEGES_EXCEEDED": msg542,
		"SNMPD_AUTH_RESTRICTED_ADDRESS": msg543,
		"SNMPD_AUTH_WRONG_PDU_TYPE": msg544,
		"SNMPD_CONFIG_ERROR": msg545,
		"SNMPD_CONTEXT_ERROR": msg546,
		"SNMPD_ENGINE_FILE_FAILURE": msg547,
		"SNMPD_ENGINE_PROCESS_ERROR": msg548,
		"SNMPD_FILE_FAILURE": msg549,
		"SNMPD_GROUP_ERROR": msg550,
		"SNMPD_INIT_FAILED": msg551,
		"SNMPD_LIBJUNIPER_FAILURE": msg552,
		"SNMPD_LOOPBACK_ADDR_ERROR": msg553,
		"SNMPD_MEMORY_FREED": msg554,
		"SNMPD_RADIX_FAILURE": msg555,
		"SNMPD_RECEIVE_FAILURE": msg556,
		"SNMPD_RMONFILE_FAILURE": msg557,
		"SNMPD_RMON_COOKIE": msg558,
		"SNMPD_RMON_EVENTLOG": msg559,
		"SNMPD_RMON_IOERROR": msg560,
		"SNMPD_RMON_MIBERROR": msg561,
		"SNMPD_RTSLIB_ASYNC_EVENT": msg562,
		"SNMPD_SEND_FAILURE": select54,
		"SNMPD_SOCKET_FAILURE": msg565,
		"SNMPD_SUBAGENT_NO_BUFFERS": msg566,
		"SNMPD_SUBAGENT_SEND_FAILED": msg567,
		"SNMPD_SYSLIB_FAILURE": msg568,
		"SNMPD_THROTTLE_QUEUE_DRAINED": msg569,
		"SNMPD_TRAP_COLD_START": msg570,
		"SNMPD_TRAP_GEN_FAILURE": msg571,
		"SNMPD_TRAP_GEN_FAILURE2": msg572,
		"SNMPD_TRAP_INVALID_DATA": msg573,
		"SNMPD_TRAP_NOT_ENOUGH_VARBINDS": msg574,
		"SNMPD_TRAP_QUEUED": msg575,
		"SNMPD_TRAP_QUEUE_DRAINED": msg576,
		"SNMPD_TRAP_QUEUE_MAX_ATTEMPTS": msg577,
		"SNMPD_TRAP_QUEUE_MAX_SIZE": msg578,
		"SNMPD_TRAP_THROTTLED": msg579,
		"SNMPD_TRAP_TYPE_ERROR": msg580,
		"SNMPD_TRAP_VARBIND_TYPE_ERROR": msg581,
		"SNMPD_TRAP_VERSION_ERROR": msg582,
		"SNMPD_TRAP_WARM_START": msg583,
		"SNMPD_USER_ERROR": msg584,
		"SNMPD_VIEW_DELETE": msg585,
		"SNMPD_VIEW_INSTALL_DEFAULT": msg586,
		"SNMPD_VIEW_OID_PARSE": msg587,
		"SNMP_GET_ERROR1": msg588,
		"SNMP_GET_ERROR2": msg589,
		"SNMP_GET_ERROR3": msg590,
		"SNMP_GET_ERROR4": msg591,
		"SNMP_NS_LOG_INFO": msg535,
		"SNMP_RTSLIB_FAILURE": msg592,
		"SNMP_SUBAGENT_IPC_REG_ROWS": msg536,
		"SNMP_TRAP_LINK_DOWN": select55,
		"SNMP_TRAP_LINK_UP": select56,
		"SNMP_TRAP_PING_PROBE_FAILED": msg597,
		"SNMP_TRAP_PING_TEST_COMPLETED": msg598,
		"SNMP_TRAP_PING_TEST_FAILED": msg599,
		"SNMP_TRAP_TRACE_ROUTE_PATH_CHANGE": msg600,
		"SNMP_TRAP_TRACE_ROUTE_TEST_COMPLETED": msg601,
		"SNMP_TRAP_TRACE_ROUTE_TEST_FAILED": msg602,
		"SNTPD": msg112,
		"SSB": msg113,
		"SSHD_LOGIN_FAILED": select57,
		"SSL_PROXY_SESSION_IGNORE": msg534,
		"SSL_PROXY_SSL_SESSION_ALLOW": msg532,
		"SSL_PROXY_SSL_SESSION_DROP": msg533,
		"TASK_TASK_REINIT": msg606,
		"TFTPD_AF_ERR": msg607,
		"TFTPD_BIND_ERR": msg608,
		"TFTPD_CONNECT_ERR": msg609,
		"TFTPD_CONNECT_INFO": msg610,
		"TFTPD_CREATE_ERR": msg611,
		"TFTPD_FIO_ERR": msg612,
		"TFTPD_FORK_ERR": msg613,
		"TFTPD_NAK_ERR": msg614,
		"TFTPD_OPEN_ERR": msg615,
		"TFTPD_RECVCOMPLETE_INFO": msg616,
		"TFTPD_RECVFROM_ERR": msg617,
		"TFTPD_RECV_ERR": msg618,
		"TFTPD_SENDCOMPLETE_INFO": msg619,
		"TFTPD_SEND_ERR": msg620,
		"TFTPD_SOCKET_ERR": msg621,
		"TFTPD_STATFS_ERR": msg622,
		"TNP": msg623,
		"UI_AUTH_EVENT": msg628,
		"UI_AUTH_INVALID_CHALLENGE": msg629,
		"UI_BOOTTIME_FAILED": msg630,
		"UI_CFG_AUDIT_NEW": select58,
		"UI_CFG_AUDIT_OTHER": select60,
		"UI_CFG_AUDIT_SET": select63,
		"UI_CFG_AUDIT_SET_SECRET": select64,
		"UI_CHILD_ARGS_EXCEEDED": msg645,
		"UI_CHILD_CHANGE_USER": msg646,
		"UI_CHILD_EXEC": msg647,
		"UI_CHILD_EXITED": msg648,
		"UI_CHILD_FOPEN": msg649,
		"UI_CHILD_PIPE_FAILED": msg650,
		"UI_CHILD_SIGNALED": msg651,
		"UI_CHILD_START": msg653,
		"UI_CHILD_STATUS": msg654,
		"UI_CHILD_STOPPED": msg652,
		"UI_CHILD_WAITPID": msg655,
		"UI_CLI_IDLE_TIMEOUT": msg656,
		"UI_CMDLINE_READ_LINE": msg657,
		"UI_CMDSET_EXEC_FAILED": msg658,
		"UI_CMDSET_FORK_FAILED": msg659,
		"UI_CMDSET_PIPE_FAILED": msg660,
		"UI_CMDSET_STOPPED": msg661,
		"UI_CMDSET_WEXITED": msg662,
		"UI_CMD_AUTH_REGEX_INVALID": msg663,
		"UI_COMMIT": msg664,
		"UI_COMMIT_AT": msg665,
		"UI_COMMIT_AT_COMPLETED": msg666,
		"UI_COMMIT_AT_FAILED": msg667,
		"UI_COMMIT_COMPRESS_FAILED": msg668,
		"UI_COMMIT_CONFIRMED": msg669,
		"UI_COMMIT_CONFIRMED_REMINDER": msg670,
		"UI_COMMIT_CONFIRMED_TIMED": msg671,
		"UI_COMMIT_EMPTY_CONTAINER": msg672,
		"UI_COMMIT_NOT_CONFIRMED": msg673,
		"UI_COMMIT_PROGRESS": msg674,
		"UI_COMMIT_QUIT": msg675,
		"UI_COMMIT_ROLLBACK_FAILED": msg676,
		"UI_COMMIT_SYNC": msg677,
		"UI_COMMIT_SYNC_FORCE": msg678,
		"UI_CONFIGURATION_ERROR": msg679,
		"UI_DAEMON_ACCEPT_FAILED": msg680,
		"UI_DAEMON_FORK_FAILED": msg681,
		"UI_DAEMON_SELECT_FAILED": msg682,
		"UI_DAEMON_SOCKET_FAILED": msg683,
		"UI_DBASE_ACCESS_FAILED": msg684,
		"UI_DBASE_CHECKOUT_FAILED": msg685,
		"UI_DBASE_EXTEND_FAILED": msg686,
		"UI_DBASE_LOGIN_EVENT": msg687,
		"UI_DBASE_LOGOUT_EVENT": msg688,
		"UI_DBASE_MISMATCH_EXTENT": msg689,
		"UI_DBASE_MISMATCH_MAJOR": msg690,
		"UI_DBASE_MISMATCH_MINOR": msg691,
		"UI_DBASE_MISMATCH_SEQUENCE": msg692,
		"UI_DBASE_MISMATCH_SIZE": msg693,
		"UI_DBASE_OPEN_FAILED": msg694,
		"UI_DBASE_REBUILD_FAILED": msg695,
		"UI_DBASE_REBUILD_SCHEMA_FAILED": msg696,
		"UI_DBASE_REBUILD_STARTED": msg697,
		"UI_DBASE_RECREATE": msg698,
		"UI_DBASE_REOPEN_FAILED": msg699,
		"UI_DUPLICATE_UID": msg700,
		"UI_JUNOSCRIPT_CMD": msg701,
		"UI_JUNOSCRIPT_ERROR": msg702,
		"UI_LOAD_EVENT": msg703,
		"UI_LOAD_JUNOS_DEFAULT_FILE_EVENT": msg704,
		"UI_LOGIN_EVENT": select71,
		"UI_LOGOUT_EVENT": msg707,
		"UI_LOST_CONN": msg708,
		"UI_MASTERSHIP_EVENT": msg709,
		"UI_MGD_TERMINATE": msg710,
		"UI_NETCONF_CMD": msg711,
		"UI_READ_FAILED": msg712,
		"UI_READ_TIMEOUT": msg713,
		"UI_REBOOT_EVENT": msg714,
		"UI_RESTART_EVENT": msg715,
		"UI_SCHEMA_CHECKOUT_FAILED": msg716,
		"UI_SCHEMA_MISMATCH_MAJOR": msg717,
		"UI_SCHEMA_MISMATCH_MINOR": msg718,
		"UI_SCHEMA_MISMATCH_SEQUENCE": msg719,
		"UI_SCHEMA_SEQUENCE_ERROR": msg720,
		"UI_SYNC_OTHER_RE": msg721,
		"UI_TACPLUS_ERROR": msg722,
		"UI_VERSION_FAILED": msg723,
		"UI_WRITE_RECONNECT": msg724,
		"VRRPD_NEWMASTER_TRAP": msg725,
		"Version": msg99,
		"WEBFILTER_REQUEST_NOT_CHECKED": msg730,
		"WEBFILTER_URL_BLOCKED": select75,
		"WEBFILTER_URL_PERMITTED": select74,
		"WEB_AUTH_FAIL": msg726,
		"WEB_AUTH_SUCCESS": msg727,
		"WEB_INTERFACE_UNAUTH": msg728,
		"WEB_READ": msg729,
		"alarmd": msg3,
		"bgp_connect_start": msg132,
		"bgp_event": msg133,
		"bgp_listen_accept": msg134,
		"bgp_listen_reset": msg135,
		"bgp_nexthop_sanity": msg136,
		"bgp_pp_recv": select33,
		"bgp_process_caps": select32,
		"bgp_send": msg141,
		"bgp_traffic_timeout": msg142,
		"bigd": select6,
		"bigpipe": select7,
		"bigstart": msg9,
		"cgatool": msg10,
		"chassisd": msg11,
		"chassism": select73,
		"checkd": select8,
		"clean_process": msg215,
		"cli": msg750,
		"cosd": msg14,
		"craftd": msg15,
		"cron": msg18,
		"crond": msg21,
		"dcd": msg22,
		"eswd": select72,
		"ftpd": msg24,
		"ha_rto_stats_handler": msg25,
		"hostinit": msg26,
		"idpinfo": msg752,
		"ifinfo": select13,
		"ifp_ifl_anydown_change_event": msg30,
		"ifp_ifl_config_event": msg31,
		"ifp_ifl_ext_chg": msg32,
		"inetd": select14,
		"init": select15,
		"ipc_msg_write": msg40,
		"kernel": select17,
		"kmd": msg753,
		"last": select28,
		"login": select18,
		"lsys_ssam_handler": msg53,
		"mcsn": msg54,
		"mgd": msg62,
		"mrvl_dfw_log_effuse_status": msg55,
		"node": select79,
		"pfed": msg751,
		"process_mode": select38,
		"profile_ssam_handler": msg57,
		"pst_nat_binding_set_profile": msg58,
		"qsfp": msg776,
		"respawn": msg64,
		"root": msg65,
		"rpd": select20,
		"rshd": msg70,
		"sfd": msg71,
		"sshd": select21,
		"syslogd": msg92,
		"task_connect": msg605,
		"task_reconfigure": msg59,
		"tnetd": msg60,
		"tnp.bootpd": msg769,
		"trace_on": msg624,
		"trace_rotate": msg625,
		"transfer-file": msg626,
		"ttloop": msg627,
		"ucd-snmp": select26,
		"usp_ipc_client_reconnect": msg95,
		"usp_trace_ipc_disconnect": msg96,
		"usp_trace_ipc_reconnect": msg97,
		"uspinfo": msg98,
		"xntpd": select27,
	}),
]);

var hdr43 = match("HEADER#3:0004/0", "message", "%{month->} %{day->} %{time->} %{p0}");

var part822 = match("HEADER#3:0004/1_0", "nwparser.p0", "fpc0 %{p0}");

var part823 = match("HEADER#3:0004/1_1", "nwparser.p0", "fpc1 %{p0}");

var part824 = match("HEADER#3:0004/1_2", "nwparser.p0", "fpc2 %{p0}");

var part825 = match("HEADER#3:0004/1_3", "nwparser.p0", "fpc3 %{p0}");

var part826 = match("HEADER#3:0004/1_4", "nwparser.p0", "fpc4 %{p0}");

var part827 = match("HEADER#3:0004/1_5", "nwparser.p0", "fpc5 %{p0}");

var part828 = match("HEADER#3:0004/1_11", "nwparser.p0", "ssb %{p0}");

var part829 = match("HEADER#15:0026.upd.a/1_0", "nwparser.p0", "RT_FLOW - %{p0}");

var part830 = match("HEADER#15:0026.upd.a/1_1", "nwparser.p0", "junos-ssl-proxy - %{p0}");

var part831 = match("HEADER#15:0026.upd.a/1_2", "nwparser.p0", "RT_APPQOS - %{p0}");

var part832 = match("HEADER#15:0026.upd.a/1_3", "nwparser.p0", "%{hfld33->} - %{p0}");

var hdr44 = match("HEADER#16:0026.upd.b/0", "message", "%{event_time->} %{hfld32->} %{hhostname->} %{p0}");

var part833 = match("MESSAGE#77:sshd:06/0", "nwparser.payload", "%{} %{p0}");

var part834 = match("MESSAGE#77:sshd:06/1_0", "nwparser.p0", "%{process}[%{process_id}]: %{p0}");

var part835 = match("MESSAGE#77:sshd:06/1_1", "nwparser.p0", "%{process}: %{p0}");

var part836 = match_copy("MESSAGE#72:Failed:05/1_2", "nwparser.p0", "p0");

var part837 = match("MESSAGE#114:ACCT_GETHOSTNAME_error/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{p0}");

var part838 = match("MESSAGE#294:LOGIN_INFORMATION/3_0", "nwparser.p0", "User %{p0}");

var part839 = match("MESSAGE#294:LOGIN_INFORMATION/3_1", "nwparser.p0", "user %{p0}");

var part840 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/0", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\"%{p0}");

var part841 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/1_0", "nwparser.p0", " connection-tag=%{fld20->} service-name=\"%{p0}");

var part842 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/1_1", "nwparser.p0", " service-name=\"%{p0}");

var part843 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/3_0", "nwparser.p0", " nat-connection-tag=%{fld6->} src-nat-rule-type=%{fld20->} %{p0}");

var part844 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/5_1", "nwparser.p0", "name=\"%{p0}");

var part845 = match("MESSAGE#485:RT_FLOW_SESSION_CREATE:02/8", "nwparser.p0", "]%{}");

var part846 = match("MESSAGE#490:RT_FLOW_SESSION_DENY:03/0_0", "nwparser.payload", "%{process}: %{event_type}: session denied %{p0}");

var part847 = match("MESSAGE#490:RT_FLOW_SESSION_DENY:03/0_1", "nwparser.payload", "%{event_type}: session denied %{p0}");

var part848 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/0", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} reason=\"%{result}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\"%{p0}");

var part849 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/2", "nwparser.p0", "%{service}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\"%{p0}");

var part850 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/4", "nwparser.p0", "%{}src-nat-rule-name=\"%{rulename}\" dst-nat-rule-%{p0}");

var part851 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/5_0", "nwparser.p0", "type=%{fld7->} dst-nat-rule-name=\"%{p0}");

var part852 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/6", "nwparser.p0", "\"%{rule_template->} protocol-id=\"%{protocol}\" policy-name=\"%{policyname}\" source-zone-name=\"%{src_zone}\" destination-zone-name=\"%{dst_zone}\" session-id-32=\"%{sessionid}\" packets-from-client=\"%{packets}\" bytes-from-client=\"%{rbytes}\" packets-from-server=\"%{dclass_counter1}\" bytes-from-server=\"%{sbytes}\" elapsed-time=\"%{duration}\"%{p0}");

var part853 = match("MESSAGE#492:RT_FLOW_SESSION_CLOSE:01/7_0", "nwparser.p0", " application=\"%{fld6}\" nested-application=\"%{fld7}\" username=\"%{username}\" roles=\"%{fld15}\" packet-incoming-interface=\"%{dinterface}\" encrypted=%{fld16->} %{p0}");

var part854 = match("MESSAGE#630:UI_CFG_AUDIT_OTHER:02/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' set: [%{action}] %{p0}");

var part855 = match_copy("MESSAGE#630:UI_CFG_AUDIT_OTHER:02/1_1", "nwparser.p0", "space");

var part856 = match("MESSAGE#634:UI_CFG_AUDIT_SET:01/1_1", "nwparser.p0", "\u003c\u003c%{change_old}> %{p0}");

var part857 = match("MESSAGE#634:UI_CFG_AUDIT_SET:01/2", "nwparser.p0", "-> \"%{change_new}\"");

var part858 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/0", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: User '%{username}' %{p0}");

var part859 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/1_0", "nwparser.p0", "set %{p0}");

var part860 = match("MESSAGE#637:UI_CFG_AUDIT_SET_SECRET:01/1_1", "nwparser.p0", "replace %{p0}");

var part861 = match("MESSAGE#675:UI_DAEMON_ACCEPT_FAILED/1_0", "nwparser.p0", "Network %{p0}");

var part862 = match("MESSAGE#675:UI_DAEMON_ACCEPT_FAILED/1_1", "nwparser.p0", "Local %{p0}");

var part863 = match("MESSAGE#755:node:05/0", "nwparser.payload", "%{hostname->} %{node->} %{p0}");

var part864 = match("MESSAGE#755:node:05/1_0", "nwparser.p0", "partner%{p0}");

var part865 = match("MESSAGE#755:node:05/1_1", "nwparser.p0", "actor%{p0}");

var select85 = linear_select([
	dup14,
	dup15,
	dup16,
	dup17,
]);

var part866 = match("HEADER#15:0026.upd.a/2", "nwparser.p0", "%{messageid->} [%{p0}", processor_chain([
	dup13,
]));

var select86 = linear_select([
	dup40,
	dup41,
]);

var part867 = match("MESSAGE#125:BFDD_TRAP_STATE_DOWN", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: local discriminator: %{resultcode}, new state: %{result}", processor_chain([
	dup21,
	dup22,
	dup56,
	dup23,
]));

var part868 = match("MESSAGE#214:DCD_MALLOC_FAILED_INIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Memory allocation failed during initialization for configuration load", processor_chain([
	dup51,
	dup22,
	dup64,
	dup23,
]));

var part869 = match("MESSAGE#225:ECCD_DAEMONIZE_FAILED", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{action}, unable to run in the background as a daemon: %{result}", processor_chain([
	dup30,
	dup22,
	dup65,
	dup23,
]));

var part870 = match("MESSAGE#226:ECCD_DUPLICATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Another copy of this program is running", processor_chain([
	dup30,
	dup22,
	dup66,
	dup23,
]));

var part871 = match("MESSAGE#232:ECCD_PID_FILE_LOCK", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to lock PID file: %{result}", processor_chain([
	dup30,
	dup22,
	dup67,
	dup23,
]));

var part872 = match("MESSAGE#233:ECCD_PID_FILE_UPDATE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to update process PID file: %{result}", processor_chain([
	dup30,
	dup22,
	dup68,
	dup23,
]));

var part873 = match("MESSAGE#272:LIBJNX_EXEC_PIPE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Unable to create pipes for command '%{action}': %{result}", processor_chain([
	dup30,
	dup22,
	dup71,
	dup23,
]));

var select87 = linear_select([
	dup76,
	dup77,
]);

var part874 = match("MESSAGE#310:MIB2D_IFD_IFINDEX_FAILURE", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: SNMP index assigned to %{uid->} changed from %{dclass_counter1->} to %{result}", processor_chain([
	dup30,
	dup22,
	dup79,
	dup23,
]));

var part875 = match("MESSAGE#412:RPD_IFL_INDEXCOLLISION", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Logical interface collision -- %{result}, %{info}", processor_chain([
	dup30,
	dup22,
	dup84,
	dup23,
]));

var part876 = match("MESSAGE#466:RPD_SCHED_CALLBACK_LONGRUNTIME", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: %{agent}: excessive runtime time during action of module", processor_chain([
	dup30,
	dup22,
	dup85,
	dup23,
]));

var part877 = match("MESSAGE#482:RPD_TASK_REINIT", "nwparser.payload", "%{process}[%{process_id}]: %{event_type}: Reinitializing", processor_chain([
	dup21,
	dup22,
	dup86,
	dup23,
]));

var select88 = linear_select([
	dup88,
	dup89,
]);

var select89 = linear_select([
	dup90,
	dup45,
]);

var select90 = linear_select([
	dup95,
	dup96,
]);

var select91 = linear_select([
	dup101,
	dup91,
]);

var part878 = match("MESSAGE#498:RT_SCREEN_TCP", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} attack-name=\"%{threat_name}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" source-zone-name=\"%{src_zone}\" interface-name=\"%{interface}\" action=\"%{action}\"]", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var part879 = match("MESSAGE#527:SSL_PROXY_SSL_SESSION_ALLOW", "nwparser.payload", "%{event_type->} [junos@%{obj_name->} logical-system-name=\"%{hostname}\" session-id=\"%{sessionid}\" source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" nat-source-address=\"%{hostip}\" nat-source-port=\"%{network_port}\" nat-destination-address=\"%{dtransaddr}\" nat-destination-port=\"%{dtransport}\" profile-name=\"%{rulename}\" source-zone-name=\"%{src_zone}\" source-interface-name=\"%{sinterface}\" destination-zone-name=\"%{dst_zone}\" destination-interface-name=\"%{dinterface}\" message=\"%{info}\"]", processor_chain([
	dup27,
	dup22,
	dup52,
]));

var select92 = linear_select([
	dup118,
	dup119,
]);

var select93 = linear_select([
	dup123,
	dup124,
]);

var part880 = match("MESSAGE#733:WEBFILTER_URL_PERMITTED", "nwparser.payload", "%{event_type->} [junos@%{fld21->} source-address=\"%{saddr}\" source-port=\"%{sport}\" destination-address=\"%{daddr}\" destination-port=\"%{dport}\" name=\"%{info}\" error-message=\"%{result}\" profile-name=\"%{profile}\" object-name=\"%{obj_name}\" pathname=\"%{directory}\" username=\"%{username}\" roles=\"%{user_role}\"] WebFilter: ACTION=\"%{action}\" %{fld2}->%{fld3->} CATEGORY=\"%{category}\" REASON=\"%{fld4}\" PROFILE=\"%{fld6}\" URL=%{url->} OBJ=%{fld7->} USERNAME=%{fld8->} ROLES=%{fld9}", processor_chain([
	dup30,
	dup22,
	dup52,
]));

var part881 = match_copy("MESSAGE#747:cli", "nwparser.payload", "fld12", processor_chain([
	dup48,
	dup47,
	dup23,
	dup22,
]));
