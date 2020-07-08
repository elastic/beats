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

var map_getSeveritylevel = {
	keyvaluepairs: {
		"1": constant("Info"),
		"2": constant("Warning"),
		"3": constant("Error"),
		"4": constant("Critical"),
	},
};

var dup1 = setc("eventcategory","1609000000");

var dup2 = date_time({
	dest: "event_time",
	args: ["fld2","fld3"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup3 = field("fld6");

var dup4 = setc("eventcategory","1603000000");

var dup5 = setc("eventcategory","1612000000");

var dup6 = setc("eventcategory","1003010000");

var dup7 = setc("obj_type","Dangerous Object");

var dup8 = setc("eventcategory","1605000000");

var dup9 = setc("ec_subject","NetworkComm");

var dup10 = setc("ec_activity","Detect");

var dup11 = setc("ec_theme","TEV");

var dup12 = match("MESSAGE#51:HTTP:Object_Infected/0", "nwparser.payload", "%{fld11->} %{fld12->} %{fld13->} %{protocol->} %{p0}");

var dup13 = match("MESSAGE#51:HTTP:Object_Infected/1_0", "nwparser.p0", "object %{p0}");

var dup14 = match("MESSAGE#51:HTTP:Object_Infected/1_1", "nwparser.p0", "Object %{p0}");

var dup15 = match("MESSAGE#51:HTTP:Object_Infected/3_0", "nwparser.p0", "Client's %{p0}");

var dup16 = match("MESSAGE#51:HTTP:Object_Infected/3_1", "nwparser.p0", "client's %{p0}");

var dup17 = match("MESSAGE#51:HTTP:Object_Infected/4", "nwparser.p0", "%{}address: %{hostip})");

var dup18 = setf("msg","$MSG");

var dup19 = date_time({
	dest: "event_time",
	args: ["fld11","fld12","fld13"],
	fmts: [
		[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO,dP],
	],
});

var dup20 = setf("obj_type","protocol");

var dup21 = setc("eventcategory","1601020000");

var dup22 = lookup({
	dest: "nwparser.severity",
	map: map_getSeveritylevel,
	key: dup3,
});

var dup23 = linear_select([
	dup13,
	dup14,
]);

var dup24 = linear_select([
	dup15,
	dup16,
]);

var dup25 = match("MESSAGE#0:KLSRV_EVENT_HOSTS_NEW_DETECTED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var dup26 = match("MESSAGE#1:KLSRV_EVENT_HOSTS_NEW_DETECTED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var dup27 = match("MESSAGE#11:KLAUD_EV_OBJECTMODIFY:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{username}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var dup28 = match("MESSAGE#12:KLAUD_EV_OBJECTMODIFY", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{username}^^%{fld18}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var dup29 = match("MESSAGE#31:GNRL_EV_OBJECT_CURED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{virusname}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var dup30 = match("MESSAGE#42:KLEVP_GroupTaskSyncState:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var dup31 = match("MESSAGE#43:KLEVP_GroupTaskSyncState", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var dup32 = match("MESSAGE#46:KLSRV_EV_LICENSE_CHECK_90", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var dup33 = match("MESSAGE#58:000000ce", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup21,
	dup2,
	dup22,
]));

var dup34 = match("MESSAGE#63:000000db", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup8,
	dup2,
	dup22,
]));

var dup35 = match("MESSAGE#77:KLSRV_EV_LICENSE_SRV_LIMITED_MODE", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var hdr1 = match("HEADER#0:0001", "message", "%kasperskyav: %{hfld1}^^%{hrecorded_time}^^%{messageid}^^%{payload}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant("^^"),
			field("hrecorded_time"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("payload"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%kasperskyav-%{hlevel}: %{hdate->} %{htime->} %{hfld1->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0002"),
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
			field("payload"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
]);

var msg1 = msg("KLSRV_EVENT_HOSTS_NEW_DETECTED:01", dup25);

var msg2 = msg("KLSRV_EVENT_HOSTS_NEW_DETECTED", dup26);

var select2 = linear_select([
	msg1,
	msg2,
]);

var msg3 = msg("KLSRV_EVENT_HOSTS_NOT_VISIBLE", dup26);

var msg4 = msg("KLSRV_HOST_STATUS_WARNING:01", dup25);

var msg5 = msg("KLSRV_HOST_STATUS_WARNING", dup26);

var select3 = linear_select([
	msg4,
	msg5,
]);

var part1 = match("MESSAGE#5:KLSRV_RUNTIME_ERROR", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup4,
	dup2,
	dup22,
]));

var msg6 = msg("KLSRV_RUNTIME_ERROR", part1);

var msg7 = msg("KLSRV_HOST_STATUS_CRITICAL:01", dup25);

var msg8 = msg("KLSRV_HOST_STATUS_CRITICAL", dup26);

var select4 = linear_select([
	msg7,
	msg8,
]);

var msg9 = msg("KLSRV_HOST_MOVED_WITH_RULE_EX", dup26);

var msg10 = msg("KLSRV_HOST_OUT_CONTROL", dup26);

var msg11 = msg("KLSRV_INVISIBLE_HOSTS_REMOVED", dup26);

var msg12 = msg("KLAUD_EV_OBJECTMODIFY:01", dup27);

var msg13 = msg("KLAUD_EV_OBJECTMODIFY", dup28);

var select5 = linear_select([
	msg12,
	msg13,
]);

var msg14 = msg("KLAUD_EV_TASK_STATE_CHANGED:01", dup27);

var msg15 = msg("KLAUD_EV_TASK_STATE_CHANGED", dup28);

var select6 = linear_select([
	msg14,
	msg15,
]);

var msg16 = msg("KLAUD_EV_ADMGROUP_CHANGED:01", dup27);

var msg17 = msg("KLAUD_EV_ADMGROUP_CHANGED", dup28);

var select7 = linear_select([
	msg16,
	msg17,
]);

var msg18 = msg("KLAUD_EV_SERVERCONNECT:01", dup27);

var msg19 = msg("KLAUD_EV_SERVERCONNECT", dup28);

var select8 = linear_select([
	msg18,
	msg19,
]);

var msg20 = msg("00010009", dup26);

var msg21 = msg("00010013", dup26);

var msg22 = msg("00020006", dup26);

var msg23 = msg("00020007", dup26);

var msg24 = msg("00020008", dup26);

var msg25 = msg("00030006", dup26);

var msg26 = msg("00030015", dup26);

var msg27 = msg("00040007", dup26);

var msg28 = msg("00040008", dup26);

var part2 = match("MESSAGE#28:GNRL_EV_SUSPICIOUS_OBJECT_FOUND:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{fld18}^^%{virusname}^^%{username}^^%{fld19}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var msg29 = msg("GNRL_EV_SUSPICIOUS_OBJECT_FOUND:01", part2);

var part3 = match("MESSAGE#29:GNRL_EV_SUSPICIOUS_OBJECT_FOUND", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{fld18}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var msg30 = msg("GNRL_EV_SUSPICIOUS_OBJECT_FOUND", part3);

var select9 = linear_select([
	msg29,
	msg30,
]);

var part4 = match("MESSAGE#30:GNRL_EV_OBJECT_CURED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{virusname}^^%{username}^^%{fld18}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var msg31 = msg("GNRL_EV_OBJECT_CURED:01", part4);

var msg32 = msg("GNRL_EV_OBJECT_CURED", dup29);

var select10 = linear_select([
	msg31,
	msg32,
]);

var part5 = match("MESSAGE#32:GNRL_EV_OBJECT_NOTCURED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup8,
	dup2,
	dup7,
	dup22,
]));

var msg33 = msg("GNRL_EV_OBJECT_NOTCURED:01", part5);

var msg34 = msg("GNRL_EV_OBJECT_NOTCURED", dup29);

var select11 = linear_select([
	msg33,
	msg34,
]);

var part6 = match("MESSAGE#34:GNRL_EV_OBJECT_DELETED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^^^%{virusname}^^%{username}^^%{fld18}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var msg35 = msg("GNRL_EV_OBJECT_DELETED:01", part6);

var msg36 = msg("GNRL_EV_OBJECT_DELETED", dup29);

var select12 = linear_select([
	msg35,
	msg36,
]);

var part7 = match("MESSAGE#36:GNRL_EV_VIRUS_FOUND:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^Virus '%{fld7}' detected in message from '%{from}' to '%{to}'.^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{virusname}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
	setc("event_description","Virus detected in email message"),
]));

var msg37 = msg("GNRL_EV_VIRUS_FOUND:01", part7);

var part8 = match("MESSAGE#37:GNRL_EV_VIRUS_FOUND:03", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{fld18}^^%{virusname}^^%{username}^^%{fld22}", processor_chain([
	dup8,
	dup2,
	dup7,
	dup22,
]));

var msg38 = msg("GNRL_EV_VIRUS_FOUND:03", part8);

var msg39 = msg("GNRL_EV_VIRUS_FOUND:02", dup29);

var select13 = linear_select([
	msg37,
	msg38,
	msg39,
]);

var part9 = match("MESSAGE#39:GNRL_EV_VIRUS_OUTBREAK", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup6,
	dup2,
	dup22,
]));

var msg40 = msg("GNRL_EV_VIRUS_OUTBREAK", part9);

var part10 = match("MESSAGE#40:GNRL_EV_ATTACK_DETECTED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{threat_name}^^%{protocol}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup5,
	dup9,
	dup10,
	dup11,
	dup2,
	dup22,
]));

var msg41 = msg("GNRL_EV_ATTACK_DETECTED:01", part10);

var part11 = match("MESSAGE#41:GNRL_EV_ATTACK_DETECTED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup6,
	dup9,
	dup10,
	dup11,
	dup2,
	dup22,
]));

var msg42 = msg("GNRL_EV_ATTACK_DETECTED", part11);

var select14 = linear_select([
	msg41,
	msg42,
]);

var msg43 = msg("KLEVP_GroupTaskSyncState:01", dup30);

var msg44 = msg("KLEVP_GroupTaskSyncState", dup31);

var select15 = linear_select([
	msg43,
	msg44,
]);

var msg45 = msg("KLPRCI_TaskState:01", dup30);

var msg46 = msg("KLPRCI_TaskState", dup31);

var select16 = linear_select([
	msg45,
	msg46,
]);

var msg47 = msg("KLSRV_EV_LICENSE_CHECK_90", dup32);

var msg48 = msg("KLNAG_EV_INV_APP_UNINSTALLED", dup32);

var msg49 = msg("KLNAG_EV_DEVICE_ARRIVAL", dup32);

var msg50 = msg("KLNAG_EV_DEVICE_REMOVE", dup32);

var msg51 = msg("FSEE_AKPLUGIN_CRITICAL_PATCHES_AVAILABLE", dup31);

var part12 = match("MESSAGE#51:HTTP:Object_Infected/2", "nwparser.p0", "%{}'%{obj_name}' is infected with '%{virusname}'(Database date: %{fld14}, %{p0}");

var all1 = all_match({
	processors: [
		dup12,
		dup23,
		part12,
		dup24,
		dup17,
	],
	on_success: processor_chain([
		dup6,
		dup18,
		dup19,
		dup20,
	]),
});

var msg52 = msg("HTTP:Object_Infected", all1);

var part13 = match("MESSAGE#52:HTTP:Object_Scanning_Error/2", "nwparser.p0", "%{}'%{obj_name}' scanning resulted in an error (Database date: %{fld14}, %{p0}");

var all2 = all_match({
	processors: [
		dup12,
		dup23,
		part13,
		dup24,
		dup17,
	],
	on_success: processor_chain([
		dup4,
		dup18,
		dup19,
		dup20,
	]),
});

var msg53 = msg("HTTP:Object_Scanning_Error", all2);

var part14 = match("MESSAGE#53:HTTP:Object_Scanned_And_Clean/2", "nwparser.p0", "%{}'%{obj_name}' has been scanned and flagged as clean(Database date: %{fld14}, %{p0}");

var all3 = all_match({
	processors: [
		dup12,
		dup23,
		part14,
		dup24,
		dup17,
	],
	on_success: processor_chain([
		dup8,
		dup18,
		dup19,
		dup20,
	]),
});

var msg54 = msg("HTTP:Object_Scanned_And_Clean", all3);

var part15 = match("MESSAGE#54:HTTP:Object_Not_Scanned_01/2", "nwparser.p0", "%{}'%{obj_name}' has not been scanned as defined by the policy as %{policyname->} %{fld17->} ( %{p0}");

var all4 = all_match({
	processors: [
		dup12,
		dup23,
		part15,
		dup24,
		dup17,
	],
	on_success: processor_chain([
		dup8,
		dup18,
		dup19,
		dup20,
	]),
});

var msg55 = msg("HTTP:Object_Not_Scanned_01", all4);

var part16 = match("MESSAGE#55:HTTP:Object_Not_Scanned_02/2", "nwparser.p0", "%{}'%{obj_name}' has not been scanned as defined by the policy ( %{p0}");

var all5 = all_match({
	processors: [
		dup12,
		dup23,
		part16,
		dup24,
		dup17,
	],
	on_success: processor_chain([
		dup8,
		dup18,
		dup19,
		dup20,
	]),
});

var msg56 = msg("HTTP:Object_Not_Scanned_02", all5);

var part17 = match("MESSAGE#57:HTTP:01/2", "nwparser.p0", "%{}'%{obj_name}");

var all6 = all_match({
	processors: [
		dup12,
		dup23,
		part17,
	],
	on_success: processor_chain([
		dup8,
		dup18,
		dup19,
		dup20,
	]),
});

var msg57 = msg("HTTP:01", all6);

var select17 = linear_select([
	msg52,
	msg53,
	msg54,
	msg55,
	msg56,
	msg57,
]);

var msg58 = msg("KLSRV_EV_LICENSE_CHECK_MORE_110", dup30);

var msg59 = msg("000000ce", dup33);

var msg60 = msg("000000d4", dup33);

var msg61 = msg("000000d5", dup25);

var msg62 = msg("000000d8", dup25);

var msg63 = msg("000000da", dup25);

var msg64 = msg("000000db", dup34);

var msg65 = msg("000000d6", dup25);

var msg66 = msg("000000de", dup34);

var part18 = match("MESSAGE#66:000000e1", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	setc("eventcategory","1606000000"),
	dup2,
	dup22,
]));

var msg67 = msg("000000e1", part18);

var msg68 = msg("0000012f", dup25);

var msg69 = msg("00000134", dup34);

var msg70 = msg("00000143", dup34);

var msg71 = msg("00000141", dup25);

var msg72 = msg("00000353", dup25);

var msg73 = msg("00000354", dup25);

var msg74 = msg("000003fb", dup34);

var msg75 = msg("000003fd", dup25);

var msg76 = msg("000000cc", dup25);

var part19 = match("MESSAGE#76:000000e2", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{fld7}^^%{fld8}^^%{fld15}^^", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg77 = msg("000000e2", part19);

var msg78 = msg("KLSRV_EV_LICENSE_SRV_LIMITED_MODE", dup35);

var part20 = match("MESSAGE#78:KSNPROXY_STOPPED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{fld5}^^%{fld7}^^%{fld8}^^", processor_chain([
	setc("eventcategory","1801030000"),
	dup2,
	dup22,
]));

var msg79 = msg("KSNPROXY_STOPPED", part20);

var part21 = match("MESSAGE#79:KLSRV_UPD_BASES_UPDATED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{fld5}^^%{fld7}^^%{fld8}^^", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg80 = msg("KLSRV_UPD_BASES_UPDATED", part21);

var part22 = match("MESSAGE#80:FSEE_AKPLUGIN_OBJECT_NOT_PROCESSED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^Object not scanned. Reason: %{event_description->} Object name: %{filename}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg81 = msg("FSEE_AKPLUGIN_OBJECT_NOT_PROCESSED", part22);

var part23 = match("MESSAGE#81:KLNAG_EV_INV_APP_INSTALLED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{fld5}^^%{fld7}^^%{product}^^%{version}^^%{fld8}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg82 = msg("KLNAG_EV_INV_APP_INSTALLED", part23);

var part24 = match("MESSAGE#82:GNRL_EV_LICENSE_EXPIRATION", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info->} User: %{username->} Component: %{fld5}Result\\Description: %{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg83 = msg("GNRL_EV_LICENSE_EXPIRATION", part24);

var part25 = match("MESSAGE#83:KSNPROXY_STARTED_CON_CHK_FAILED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{fld5}^^%{fld7}^^%{fld8}^^", processor_chain([
	setc("eventcategory","1703000000"),
	dup2,
	dup22,
]));

var msg84 = msg("KSNPROXY_STARTED_CON_CHK_FAILED", part25);

var part26 = match("MESSAGE#84:000003f8", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_description}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^Event type:%{event_type->} Result: %{fld23->} Object: %{obj_name->} Object\\Path: %{url->} User:%{username->} Update ID: %{fld51}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg85 = msg("000003f8", part26);

var msg86 = msg("FSEE_AKPLUGIN_AVBASES_CORRUPTED", dup35);

var part27 = match("MESSAGE#86:GNRL_EV_OBJECT_BLOCKED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{fld19}^^%{virusname}^^%{username}^^%{fld18}", processor_chain([
	dup1,
	dup2,
	dup7,
	dup22,
]));

var msg87 = msg("GNRL_EV_OBJECT_BLOCKED", part27);

var part28 = match("MESSAGE#87:0000014d", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg88 = msg("0000014d", part28);

var part29 = match("MESSAGE#88:000003f7/0", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_description}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^Event type:%{event_type->} Result: %{result->} %{p0}");

var part30 = match("MESSAGE#88:000003f7/1_0", "nwparser.p0", "Object: %{obj_name->} Object\\Path: %{url->} User:%{username}(%{privilege})%{p0}");

var part31 = match("MESSAGE#88:000003f7/1_1", "nwparser.p0", "User:%{username}(%{privilege})%{p0}");

var select18 = linear_select([
	part30,
	part31,
]);

var part32 = match("MESSAGE#88:000003f7/2", "nwparser.p0", "%{}Release date: %{fld23}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}");

var all7 = all_match({
	processors: [
		part29,
		select18,
		part32,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup22,
	]),
});

var msg89 = msg("000003f7", all7);

var part33 = match("MESSAGE#89:FSEE_AKPLUGIN_OBJECT_NOT_ISOLATED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^Object not quarantined. Reason: %{event_description}^^%{context}^^%{product}^^%{version}^^%{filename}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var msg90 = msg("FSEE_AKPLUGIN_OBJECT_NOT_ISOLATED", part33);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"000000cc": msg76,
		"000000ce": msg59,
		"000000d4": msg60,
		"000000d5": msg61,
		"000000d6": msg65,
		"000000d8": msg62,
		"000000da": msg63,
		"000000db": msg64,
		"000000de": msg66,
		"000000e1": msg67,
		"000000e2": msg77,
		"0000012f": msg68,
		"00000134": msg69,
		"00000141": msg71,
		"00000143": msg70,
		"0000014d": msg88,
		"00000353": msg72,
		"00000354": msg73,
		"000003f7": msg89,
		"000003f8": msg85,
		"000003fb": msg74,
		"000003fd": msg75,
		"00010009": msg20,
		"00010013": msg21,
		"00020006": msg22,
		"00020007": msg23,
		"00020008": msg24,
		"00030006": msg25,
		"00030015": msg26,
		"00040007": msg27,
		"00040008": msg28,
		"FSEE_AKPLUGIN_AVBASES_CORRUPTED": msg86,
		"FSEE_AKPLUGIN_CRITICAL_PATCHES_AVAILABLE": msg51,
		"FSEE_AKPLUGIN_OBJECT_NOT_ISOLATED": msg90,
		"FSEE_AKPLUGIN_OBJECT_NOT_PROCESSED": msg81,
		"GNRL_EV_ATTACK_DETECTED": select14,
		"GNRL_EV_LICENSE_EXPIRATION": msg83,
		"GNRL_EV_OBJECT_BLOCKED": msg87,
		"GNRL_EV_OBJECT_CURED": select10,
		"GNRL_EV_OBJECT_DELETED": select12,
		"GNRL_EV_OBJECT_NOTCURED": select11,
		"GNRL_EV_SUSPICIOUS_OBJECT_FOUND": select9,
		"GNRL_EV_VIRUS_FOUND": select13,
		"GNRL_EV_VIRUS_OUTBREAK": msg40,
		"HTTP": select17,
		"KLAUD_EV_ADMGROUP_CHANGED": select7,
		"KLAUD_EV_OBJECTMODIFY": select5,
		"KLAUD_EV_SERVERCONNECT": select8,
		"KLAUD_EV_TASK_STATE_CHANGED": select6,
		"KLEVP_GroupTaskSyncState": select15,
		"KLNAG_EV_DEVICE_ARRIVAL": msg49,
		"KLNAG_EV_DEVICE_REMOVE": msg50,
		"KLNAG_EV_INV_APP_INSTALLED": msg82,
		"KLNAG_EV_INV_APP_UNINSTALLED": msg48,
		"KLPRCI_TaskState": select16,
		"KLSRV_EVENT_HOSTS_NEW_DETECTED": select2,
		"KLSRV_EVENT_HOSTS_NOT_VISIBLE": msg3,
		"KLSRV_EV_LICENSE_CHECK_90": msg47,
		"KLSRV_EV_LICENSE_CHECK_MORE_110": msg58,
		"KLSRV_EV_LICENSE_SRV_LIMITED_MODE": msg78,
		"KLSRV_HOST_MOVED_WITH_RULE_EX": msg9,
		"KLSRV_HOST_OUT_CONTROL": msg10,
		"KLSRV_HOST_STATUS_CRITICAL": select4,
		"KLSRV_HOST_STATUS_WARNING": select3,
		"KLSRV_INVISIBLE_HOSTS_REMOVED": msg11,
		"KLSRV_RUNTIME_ERROR": msg6,
		"KLSRV_UPD_BASES_UPDATED": msg80,
		"KSNPROXY_STARTED_CON_CHK_FAILED": msg84,
		"KSNPROXY_STOPPED": msg79,
	}),
]);

var part34 = match("MESSAGE#51:HTTP:Object_Infected/0", "nwparser.payload", "%{fld11->} %{fld12->} %{fld13->} %{protocol->} %{p0}");

var part35 = match("MESSAGE#51:HTTP:Object_Infected/1_0", "nwparser.p0", "object %{p0}");

var part36 = match("MESSAGE#51:HTTP:Object_Infected/1_1", "nwparser.p0", "Object %{p0}");

var part37 = match("MESSAGE#51:HTTP:Object_Infected/3_0", "nwparser.p0", "Client's %{p0}");

var part38 = match("MESSAGE#51:HTTP:Object_Infected/3_1", "nwparser.p0", "client's %{p0}");

var part39 = match("MESSAGE#51:HTTP:Object_Infected/4", "nwparser.p0", "%{}address: %{hostip})");

var select19 = linear_select([
	dup13,
	dup14,
]);

var select20 = linear_select([
	dup15,
	dup16,
]);

var part40 = match("MESSAGE#0:KLSRV_EVENT_HOSTS_NEW_DETECTED:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var part41 = match("MESSAGE#1:KLSRV_EVENT_HOSTS_NEW_DETECTED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup1,
	dup2,
	dup22,
]));

var part42 = match("MESSAGE#11:KLAUD_EV_OBJECTMODIFY:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{username}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var part43 = match("MESSAGE#12:KLAUD_EV_OBJECTMODIFY", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{username}^^%{fld18}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var part44 = match("MESSAGE#31:GNRL_EV_OBJECT_CURED", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{obj_name}^^%{fld17}^^%{virusname}", processor_chain([
	dup6,
	dup2,
	dup7,
	dup22,
]));

var part45 = match("MESSAGE#42:KLEVP_GroupTaskSyncState:01", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var part46 = match("MESSAGE#43:KLEVP_GroupTaskSyncState", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var part47 = match("MESSAGE#46:KLSRV_EV_LICENSE_CHECK_90", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}", processor_chain([
	dup5,
	dup2,
	dup22,
]));

var part48 = match("MESSAGE#58:000000ce", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup21,
	dup2,
	dup22,
]));

var part49 = match("MESSAGE#63:000000db", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^%{fld16}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld21}^^%{fld22}", processor_chain([
	dup8,
	dup2,
	dup22,
]));

var part50 = match("MESSAGE#77:KLSRV_EV_LICENSE_SRV_LIMITED_MODE", "nwparser.payload", "%{fld1}^^%{fld2->} %{fld3}.%{fld4}^^%{event_type}^^%{fld6}^^%{hostip}^^%{hostname}^^%{group_object}^^%{info}^^%{event_description}^^%{context}^^%{product}^^%{version}^^%{fld15}^^", processor_chain([
	dup1,
	dup2,
	dup22,
]));
