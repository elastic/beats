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

var dup1 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hdata"),
		constant(": "),
		field("p0"),
	],
});

var dup2 = match("HEADER#1:0002/1_0", "nwparser.p0", "high %{p0}");

var dup3 = match("HEADER#1:0002/1_1", "nwparser.p0", "low %{p0}");

var dup4 = call({
	dest: "nwparser.messageid",
	fn: STRCAT,
	args: [
		field("msgIdPart1"),
		constant("_"),
		field("msgIdPart2"),
	],
});

var dup5 = match("HEADER#2:0008/2", "nwparser.p0", "%{} %{p0}");

var dup6 = match("HEADER#2:0008/3_0", "nwparser.p0", "jitter %{p0}");

var dup7 = match("HEADER#2:0008/3_1", "nwparser.p0", "loss %{p0}");

var dup8 = match("HEADER#2:0008/3_2", "nwparser.p0", "bps %{p0}");

var dup9 = match("HEADER#2:0008/3_3", "nwparser.p0", "pps %{p0}");

var dup10 = match("HEADER#3:0003/4", "nwparser.p0", "%{} %{msgIdPart1->} %{msgIdPart2->} %{p0}");

var dup11 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup12 = setc("eventcategory","1801010000");

var dup13 = setf("msg","$MSG");

var dup14 = date_time({
	dest: "starttime",
	args: ["fld15","fld16","fld17","fld18","fld19","fld20"],
	fmts: [
		[dW,dM,dD,dH,dT,dS],
	],
});

var dup15 = setc("eventcategory","1801020000");

var dup16 = date_time({
	dest: "endtime",
	args: ["fld15","fld16","fld17","fld18","fld19","fld20"],
	fmts: [
		[dW,dM,dD,dH,dT,dS],
	],
});

var dup17 = setc("eventcategory","1607000000");

var dup18 = setc("eventcategory","1605000000");

var dup19 = setc("eventcategory","1701000000");

var dup20 = setc("eventcategory","1603010000");

var dup21 = match("MESSAGE#19:mitigation:TMS_Start/1_0", "nwparser.p0", "%{fld21}, %{p0}");

var dup22 = match("MESSAGE#19:mitigation:TMS_Start/1_1", "nwparser.p0", ", %{p0}");

var dup23 = match("MESSAGE#19:mitigation:TMS_Start/2", "nwparser.p0", "leader %{parent_node}");

var dup24 = setc("eventcategory","1502020000");

var dup25 = setc("event_type","TMS mitigation");

var dup26 = setc("disposition","ongoing");

var dup27 = setc("disposition","done");

var dup28 = setc("event_type","Third party mitigation");

var dup29 = setc("event_type","Blackhole mitigation");

var dup30 = setc("event_type","Flowspec mitigation");

var dup31 = match("MESSAGE#39:anomaly:Resource_Info:01/1_0", "nwparser.p0", "%{fld21->} duration %{p0}");

var dup32 = match("MESSAGE#39:anomaly:Resource_Info:01/1_1", "nwparser.p0", "duration %{p0}");

var dup33 = match("MESSAGE#39:anomaly:Resource_Info:01/2", "nwparser.p0", "%{duration->} percent %{fld3->} rate %{fld4->} rateUnit %{fld5->} protocol %{protocol->} flags %{fld6->} url %{url}, %{info}");

var dup34 = setc("eventcategory","1002000000");

var dup35 = setc("signame","Bandwidth");

var dup36 = date_time({
	dest: "starttime",
	args: ["fld15","fld16","fld17","fld18","fld19","fld20"],
	fmts: [
		[dW,dM,dD,dN,dU,dO],
	],
});

var dup37 = match("MESSAGE#40:anomaly:Resource_Info:02/2", "nwparser.p0", "%{duration->} percent %{fld3->} rate %{fld4->} rateUnit %{fld5->} protocol %{protocol->} flags %{fld6->} url %{url}");

var dup38 = date_time({
	dest: "starttime",
	args: ["fld2","fld3"],
	fmts: [
		[dW,dc("-"),dM,dc("-"),dF,dZ],
	],
});

var dup39 = match("HEADER#0:0001/0", "message", "%{hmonth->} %{hday->} %{htime->} %{hdata}: %{p0}", processor_chain([
	dup1,
]));

var dup40 = linear_select([
	dup2,
	dup3,
]);

var dup41 = linear_select([
	dup6,
	dup7,
	dup8,
	dup9,
]);

var dup42 = match("MESSAGE#2:BGP:Down", "nwparser.payload", "%{protocol->} down for router %{node}, leader %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var dup43 = match("MESSAGE#3:BGP:Restored", "nwparser.payload", "%{protocol->} restored for router %{node}, leader %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup15,
	dup13,
	dup16,
]));

var dup44 = linear_select([
	dup21,
	dup22,
]);

var dup45 = linear_select([
	dup31,
	dup32,
]);

var part1 = match("HEADER#0:0001/1_0", "nwparser.p0", "TMS %{p0}");

var part2 = match("HEADER#0:0001/1_1", "nwparser.p0", "Third party %{p0}");

var part3 = match("HEADER#0:0001/1_2", "nwparser.p0", "Blackhole %{p0}");

var part4 = match("HEADER#0:0001/1_3", "nwparser.p0", "Flowspec %{p0}");

var select1 = linear_select([
	part1,
	part2,
	part3,
	part4,
]);

var part5 = match("HEADER#0:0001/2", "nwparser.p0", "%{} %{messageid->} %{p0}");

var all1 = all_match({
	processors: [
		dup39,
		select1,
		part5,
	],
	on_success: processor_chain([
		setc("header_id","0001"),
	]),
});

var part6 = match("HEADER#1:0002/2", "nwparser.p0", "%{}interface %{msgIdPart1->} %{msgIdPart2->} %{p0}");

var all2 = all_match({
	processors: [
		dup39,
		dup40,
		part6,
	],
	on_success: processor_chain([
		setc("header_id","0002"),
		dup4,
	]),
});

var part7 = match("HEADER#2:0008/4", "nwparser.p0", "%{} %{msgIdPart1->} %{hfld1->} for service %{p0}");

var all3 = all_match({
	processors: [
		dup39,
		dup40,
		dup5,
		dup41,
		part7,
	],
	on_success: processor_chain([
		setc("header_id","0008"),
		call({
			dest: "nwparser.messageid",
			fn: STRCAT,
			args: [
				constant("usage_"),
				field("msgIdPart1"),
			],
		}),
	]),
});

var all4 = all_match({
	processors: [
		dup39,
		dup40,
		dup5,
		dup41,
		dup10,
	],
	on_success: processor_chain([
		setc("header_id","0003"),
		dup4,
	]),
});

var part8 = match("HEADER#4:0004/1_2", "nwparser.p0", "High %{p0}");

var select2 = linear_select([
	dup2,
	dup3,
	part8,
]);

var all5 = all_match({
	processors: [
		dup39,
		select2,
		dup10,
	],
	on_success: processor_chain([
		setc("header_id","0004"),
		dup4,
	]),
});

var hdr1 = match("HEADER#5:0005", "message", "%{hmonth->} %{hday->} %{htime->} pfsp: The %{messageid->} %{p0}", processor_chain([
	setc("header_id","0005"),
	dup11,
]));

var hdr2 = match("HEADER#6:0006", "message", "%{hmonth->} %{hday->} %{htime->} pfsp: Alert %{messageid->} %{p0}", processor_chain([
	setc("header_id","0006"),
	dup11,
]));

var hdr3 = match("HEADER#7:0007", "message", "%{hmonth->} %{hday->} %{htime->} pfsp: %{messageid->} %{p0}", processor_chain([
	setc("header_id","0007"),
	dup11,
]));

var hdr4 = match("HEADER#8:0010", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1}: %{msgIdPart1->} %{msgIdPart2}: %{payload}", processor_chain([
	setc("header_id","0010"),
	dup4,
]));

var hdr5 = match("HEADER#9:0009", "message", "%{hmonth->} %{hday->} %{htime->} %{hfld1}: %{messageid}: %{payload}", processor_chain([
	setc("header_id","0009"),
]));

var select3 = linear_select([
	all1,
	all2,
	all3,
	all4,
	all5,
	hdr1,
	hdr2,
	hdr3,
	hdr4,
	hdr5,
]);

var part9 = match("MESSAGE#0:Flow:Down", "nwparser.payload", "Flow down for router %{node}, leader %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var msg1 = msg("Flow:Down", part9);

var part10 = match("MESSAGE#1:Flow:Restored", "nwparser.payload", "Flow restored for router %{node}, leader %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup15,
	dup13,
	dup16,
]));

var msg2 = msg("Flow:Restored", part10);

var select4 = linear_select([
	msg1,
	msg2,
]);

var msg3 = msg("BGP:Down", dup42);

var msg4 = msg("BGP:Restored", dup43);

var part11 = match("MESSAGE#4:BGP:Instability", "nwparser.payload", "%{protocol->} instability router %{node->} threshold %{fld25->} (%{fld1}) observed %{trigger_val->} (%{fld2})", processor_chain([
	dup17,
	dup13,
]));

var msg5 = msg("BGP:Instability", part11);

var part12 = match("MESSAGE#5:BGP:Instability_Ended", "nwparser.payload", "%{protocol->} Instability for router %{node->} ended", processor_chain([
	dup18,
	dup13,
]));

var msg6 = msg("BGP:Instability_Ended", part12);

var part13 = match("MESSAGE#6:BGP:Hijack", "nwparser.payload", "%{protocol->} Hijack local_prefix %{fld26->} router %{node->} bgp_prefix %{fld27->} bgp_attributes %{event_description}", processor_chain([
	setc("eventcategory","1002050000"),
	dup13,
]));

var msg7 = msg("BGP:Hijack", part13);

var part14 = match("MESSAGE#7:BGP:Hijack_Done", "nwparser.payload", "%{protocol->} Hijack for prefix %{fld26->} router %{node->} done", processor_chain([
	dup18,
	dup13,
]));

var msg8 = msg("BGP:Hijack_Done", part14);

var part15 = match("MESSAGE#8:BGP:Trap", "nwparser.payload", "%{protocol->} Trap %{node}: Prefix %{fld5->} %{fld6->} %{event_description}", processor_chain([
	dup19,
	dup13,
]));

var msg9 = msg("BGP:Trap", part15);

var select5 = linear_select([
	msg3,
	msg4,
	msg5,
	msg6,
	msg7,
	msg8,
	msg9,
]);

var part16 = match("MESSAGE#9:Device:Unreachable", "nwparser.payload", "Device %{node->} unreachable by controller %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var msg10 = msg("Device:Unreachable", part16);

var part17 = match("MESSAGE#10:Device:Reachable", "nwparser.payload", "Device %{node->} reachable again by controller %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup15,
	dup13,
	dup16,
]));

var msg11 = msg("Device:Reachable", part17);

var select6 = linear_select([
	msg10,
	msg11,
]);

var part18 = match("MESSAGE#11:Hardware:Failure", "nwparser.payload", "Hardware failure on %{node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} GMT: %{event_description}", processor_chain([
	dup20,
	dup13,
	dup14,
]));

var msg12 = msg("Hardware:Failure", part18);

var part19 = match("MESSAGE#12:Hardware:Failure_Done", "nwparser.payload", "Hardware failure on %{node->} done at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21->} GMT: %{event_description}", processor_chain([
	dup18,
	dup13,
	dup16,
]));

var msg13 = msg("Hardware:Failure_Done", part19);

var select7 = linear_select([
	msg12,
	msg13,
]);

var msg14 = msg("SNMP:Down", dup42);

var msg15 = msg("SNMP:Restored", dup43);

var select8 = linear_select([
	msg14,
	msg15,
]);

var part20 = match("MESSAGE#15:configuration", "nwparser.payload", "configuration was changed on leader %{parent_node->} to version %{version->} by %{administrator}", processor_chain([
	dup19,
	dup13,
	setc("event_description","Configuration changed"),
]));

var msg16 = msg("configuration", part20);

var part21 = match("MESSAGE#16:Autoclassification", "nwparser.payload", "Autoclassification was restarted on %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21->} by %{administrator}", processor_chain([
	dup19,
	dup13,
	setc("event_description","Autoclassification restarted"),
	dup14,
]));

var msg17 = msg("Autoclassification", part21);

var part22 = match("MESSAGE#17:GRE:Down", "nwparser.payload", "GRE tunnel down for destination %{daddr}, leader %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var msg18 = msg("GRE:Down", part22);

var part23 = match("MESSAGE#18:GRE:Restored", "nwparser.payload", "GRE tunnel restored for destination %{daddr}, leader %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	setc("eventcategory","1801020100"),
	dup13,
	dup16,
]));

var msg19 = msg("GRE:Restored", part23);

var select9 = linear_select([
	msg18,
	msg19,
]);

var part24 = match("MESSAGE#19:mitigation:TMS_Start/0", "nwparser.payload", "pfsp: TMS mitigation %{policyname->} started at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all6 = all_match({
	processors: [
		part24,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup25,
		dup26,
		dup14,
	]),
});

var msg20 = msg("mitigation:TMS_Start", all6);

var part25 = match("MESSAGE#20:mitigation:TMS_Stop/0", "nwparser.payload", "pfsp: TMS mitigation %{policyname->} stopped at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all7 = all_match({
	processors: [
		part25,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup25,
		dup27,
		dup16,
	]),
});

var msg21 = msg("mitigation:TMS_Stop", all7);

var part26 = match("MESSAGE#21:mitigation:Thirdparty_Start/0", "nwparser.payload", "pfsp: Third party mitigation %{node->} started at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all8 = all_match({
	processors: [
		part26,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup28,
		dup26,
		dup14,
	]),
});

var msg22 = msg("mitigation:Thirdparty_Start", all8);

var part27 = match("MESSAGE#22:mitigation:Thirdparty_Stop/0", "nwparser.payload", "pfsp: Third party mitigation %{node->} stopped at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all9 = all_match({
	processors: [
		part27,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup28,
		dup27,
	]),
});

var msg23 = msg("mitigation:Thirdparty_Stop", all9);

var part28 = match("MESSAGE#23:mitigation:Blackhole_Start/0", "nwparser.payload", "pfsp: Blackhole mitigation %{node->} started at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all10 = all_match({
	processors: [
		part28,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup29,
		dup26,
		dup14,
	]),
});

var msg24 = msg("mitigation:Blackhole_Start", all10);

var part29 = match("MESSAGE#24:mitigation:Blackhole_Stop/0", "nwparser.payload", "pfsp: Blackhole mitigation %{node->} stopped at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all11 = all_match({
	processors: [
		part29,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup29,
		dup27,
	]),
});

var msg25 = msg("mitigation:Blackhole_Stop", all11);

var part30 = match("MESSAGE#25:mitigation:Flowspec_Start/0", "nwparser.payload", "pfsp: Flowspec mitigation %{node->} started at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all12 = all_match({
	processors: [
		part30,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup30,
		dup26,
		dup14,
	]),
});

var msg26 = msg("mitigation:Flowspec_Start", all12);

var part31 = match("MESSAGE#26:mitigation:Flowspec_Stop/0", "nwparser.payload", "pfsp: Flowspec mitigation %{node->} stopped at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all13 = all_match({
	processors: [
		part31,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		dup30,
		dup27,
	]),
});

var msg27 = msg("mitigation:Flowspec_Stop", all13);

var select10 = linear_select([
	msg20,
	msg21,
	msg22,
	msg23,
	msg24,
	msg25,
	msg26,
	msg27,
]);

var part32 = match("MESSAGE#27:TMS:Fault_Cleared", "nwparser.payload", "TMS '%{event_description}' fault for resource '%{resource}' on TMS %{node->} cleared", processor_chain([
	dup18,
	dup13,
	setc("event_type","Fault Cleared"),
]));

var msg28 = msg("TMS:Fault_Cleared", part32);

var part33 = match("MESSAGE#28:TMS:Fault", "nwparser.payload", "TMS '%{event_description}' fault for resource '%{resource}' on TMS %{node}", processor_chain([
	dup20,
	dup13,
	setc("event_type","Fault Occured"),
]));

var msg29 = msg("TMS:Fault", part33);

var select11 = linear_select([
	msg28,
	msg29,
]);

var part34 = match("MESSAGE#29:usage_alert:Interface", "nwparser.payload", "pfsp: %{trigger_desc->} interface usage alert %{fld1->} for router %{node->} interface \"%{interface}\" speed %{fld2->} threshold %{fld25->} observed %{trigger_val->} pct %{fld3}", processor_chain([
	dup17,
	dup13,
]));

var msg30 = msg("usage_alert:Interface", part34);

var part35 = match("MESSAGE#30:usage_alert:Interface_Done", "nwparser.payload", "pfsp: %{trigger_desc->} interface usage alert %{fld1->} done for router %{node->} interface \"%{interface}\"", processor_chain([
	dup18,
	dup13,
]));

var msg31 = msg("usage_alert:Interface_Done", part35);

var part36 = match("MESSAGE#31:usage_alert:Fingerprint_Threshold", "nwparser.payload", "pfsp: %{trigger_desc->} usage alert %{fld1->} for fingerprint %{policyname->} threshold %{fld25->} observed %{trigger_val}", processor_chain([
	dup17,
	dup13,
]));

var msg32 = msg("usage_alert:Fingerprint_Threshold", part36);

var part37 = match("MESSAGE#32:usage_alert:Fingerprint_Threshold_Done", "nwparser.payload", "pfsp: %{trigger_desc->} usage alert %{fld1->} for fingerprint %{policyname->} done", processor_chain([
	dup18,
	dup13,
]));

var msg33 = msg("usage_alert:Fingerprint_Threshold_Done", part37);

var part38 = match("MESSAGE#33:usage_alert:Service_Threshold", "nwparser.payload", "pfsp: %{trigger_desc->} %{fld1->} usage alert %{fld2->} for service %{service}, %{application->} threshold %{fld25->} observed %{trigger_val}", processor_chain([
	dup17,
	dup13,
]));

var msg34 = msg("usage_alert:Service_Threshold", part38);

var part39 = match("MESSAGE#34:usage_alert:Service_Threshold_Done", "nwparser.payload", "pfsp: %{trigger_desc->} %{fld1->} alert %{fld2->} for service %{service->} done", processor_chain([
	dup18,
	dup13,
]));

var msg35 = msg("usage_alert:Service_Threshold_Done", part39);

var part40 = match("MESSAGE#35:usage_alert:ManagedObject_Threshold", "nwparser.payload", "pfsp: %{trigger_desc->} usage alert %{fld1->} for %{category->} %{fld2->} threshold %{fld25->} observed %{trigger_val}", processor_chain([
	dup17,
	dup13,
]));

var msg36 = msg("usage_alert:ManagedObject_Threshold", part40);

var part41 = match("MESSAGE#36:usage_alert:ManagedObject_Threshold_Done", "nwparser.payload", "pfsp: %{trigger_desc->} usage alert %{fld1->} for %{fld3->} %{fld4->} done", processor_chain([
	dup18,
	dup13,
]));

var msg37 = msg("usage_alert:ManagedObject_Threshold_Done", part41);

var select12 = linear_select([
	msg30,
	msg31,
	msg32,
	msg33,
	msg34,
	msg35,
	msg36,
	msg37,
]);

var part42 = match("MESSAGE#37:Test", "nwparser.payload", "Test syslog message%{}", processor_chain([
	dup18,
	dup13,
]));

var msg38 = msg("Test", part42);

var part43 = match("MESSAGE#38:script/0", "nwparser.payload", "script %{node->} ran at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all14 = all_match({
	processors: [
		part43,
		dup44,
		dup23,
	],
	on_success: processor_chain([
		dup24,
		dup13,
		setc("event_type","Script mitigation"),
		dup26,
		dup14,
	]),
});

var msg39 = msg("script", all14);

var part44 = match("MESSAGE#39:anomaly:Resource_Info:01/0", "nwparser.payload", "anomaly Bandwidth id %{event_id->} status %{disposition->} severity %{severity->} classification %{category->} impact %{fld10->} src %{daddr}/%{dport->} %{fld1->} dst %{saddr}/%{sport->} %{fld2->} start %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all15 = all_match({
	processors: [
		part44,
		dup45,
		dup33,
	],
	on_success: processor_chain([
		dup34,
		dup13,
		dup35,
		dup36,
	]),
});

var msg40 = msg("anomaly:Resource_Info:01", all15);

var part45 = match("MESSAGE#40:anomaly:Resource_Info:02/0", "nwparser.payload", "anomaly Bandwidth id %{event_id->} status %{disposition->} severity %{severity->} classification %{category->} src %{daddr}/%{dport->} %{fld1->} dst %{saddr}/%{sport->} %{fld2->} start %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all16 = all_match({
	processors: [
		part45,
		dup45,
		dup37,
	],
	on_success: processor_chain([
		dup34,
		dup13,
		dup35,
		dup36,
	]),
});

var msg41 = msg("anomaly:Resource_Info:02", all16);

var part46 = match("MESSAGE#41:anomaly:Resource_Info:03/0", "nwparser.payload", "anomaly %{signame->} id %{event_id->} status %{disposition->} severity %{severity->} classification %{category->} impact %{fld10->} src %{daddr}/%{dport->} %{fld1->} dst %{saddr}/%{sport->} %{fld2->} start %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all17 = all_match({
	processors: [
		part46,
		dup45,
		dup33,
	],
	on_success: processor_chain([
		dup34,
		dup13,
		dup36,
	]),
});

var msg42 = msg("anomaly:Resource_Info:03", all17);

var part47 = match("MESSAGE#42:anomaly:Resource_Info:04/0", "nwparser.payload", "anomaly %{signame->} id %{event_id->} status %{disposition->} severity %{severity->} classification %{category->} src %{daddr}/%{dport->} %{fld1->} dst %{saddr}/%{sport->} %{fld2->} start %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{p0}");

var all18 = all_match({
	processors: [
		part47,
		dup45,
		dup37,
	],
	on_success: processor_chain([
		dup34,
		dup13,
		dup36,
	]),
});

var msg43 = msg("anomaly:Resource_Info:04", all18);

var part48 = match("MESSAGE#43:anomaly:Router_Info:01", "nwparser.payload", "anomaly Bandwidth id %{sigid->} status %{disposition->} severity %{severity->} classification %{category->} router %{fld6->} router_name %{node->} interface %{fld4->} interface_name \"%{interface}\" %{fld5}", processor_chain([
	dup34,
	dup13,
	dup35,
]));

var msg44 = msg("anomaly:Router_Info:01", part48);

var part49 = match("MESSAGE#44:anomaly:Router_Info:02", "nwparser.payload", "anomaly %{signame->} id %{sigid->} status %{disposition->} severity %{severity->} classification %{category->} router %{fld6->} router_name %{node->} interface %{fld4->} interface_name \"%{interface}\" %{fld5}", processor_chain([
	dup34,
	dup13,
]));

var msg45 = msg("anomaly:Router_Info:02", part49);

var select13 = linear_select([
	msg40,
	msg41,
	msg42,
	msg43,
	msg44,
	msg45,
]);

var part50 = match("MESSAGE#45:Peakflow:Unreachable", "nwparser.payload", "Peakflow device %{node->} unreachable by %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var msg46 = msg("Peakflow:Unreachable", part50);

var part51 = match("MESSAGE#46:Peakflow:Reachable", "nwparser.payload", "Peakflow device %{node->} reachable again by %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup15,
	dup13,
	dup16,
]));

var msg47 = msg("Peakflow:Reachable", part51);

var select14 = linear_select([
	msg46,
	msg47,
]);

var part52 = match("MESSAGE#47:Host:Detection", "nwparser.payload", "Host Detection alert %{fld1}, start %{fld2->} %{fld3->} %{fld4}, duration %{duration}, stop %{fld5->} %{fld6->} %{fld7}, , importance %{severity}, managed_objects (%{fld8}), is now %{result}, (parent managed object %{fld9})", processor_chain([
	dup18,
	dup13,
	dup38,
	date_time({
		dest: "endtime",
		args: ["fld5","fld6"],
		fmts: [
			[dW,dc("-"),dM,dc("-"),dF,dZ],
		],
	}),
]));

var msg48 = msg("Host:Detection", part52);

var part53 = match("MESSAGE#48:Host:Detection:01", "nwparser.payload", "Host Detection alert %{fld1}, start %{fld2->} %{fld3->} %{fld4}, duration %{duration}, direction %{direction}, host %{saddr}, signatures (%{signame}), impact %{fld5}, importance %{severity}, managed_objects (%{fld6}), (parent managed object %{fld7})", processor_chain([
	dup18,
	dup13,
	dup38,
]));

var msg49 = msg("Host:Detection:01", part53);

var select15 = linear_select([
	msg48,
	msg49,
]);

var part54 = match("MESSAGE#49:Infrastructure", "nwparser.payload", "AIF license expiring cleared,URL: %{url}", processor_chain([
	dup18,
	dup13,
	setc("event_description","AIF license expiring cleared"),
]));

var msg50 = msg("Infrastructure", part54);

var part55 = match("MESSAGE#50:Infrastructure:02", "nwparser.payload", "Hardware sensor detected a critical state. System Fan%{fld1}:%{fld2}Triggering value:%{fld3},URL:%{url}", processor_chain([
	dup18,
	dup13,
	setc("event_description","Hardware sensor detected a critical state"),
]));

var msg51 = msg("Infrastructure:02", part55);

var part56 = match("MESSAGE#51:Infrastructure:01", "nwparser.payload", "AIF license expired cleared,URL: %{url}", processor_chain([
	dup18,
	dup13,
	setc("event_description","AIF license expired cleared"),
]));

var msg52 = msg("Infrastructure:01", part56);

var select16 = linear_select([
	msg50,
	msg51,
	msg52,
]);

var part57 = match("MESSAGE#52:Blocked_Host", "nwparser.payload", "Blocked host%{saddr}at%{fld1}by Blocked Countries using%{protocol}destination%{daddr},URL:%{url}", processor_chain([
	setc("eventcategory","1803000000"),
	dup13,
]));

var msg53 = msg("Blocked_Host", part57);

var part58 = match("MESSAGE#53:Change_Log", "nwparser.payload", "Username:%{username}, Subsystem:%{fld1}, Setting Type:%{fld2}, Message:%{fld3}", processor_chain([
	dup18,
	dup13,
]));

var msg54 = msg("Change_Log", part58);

var part59 = match("MESSAGE#54:Protection_Mode", "nwparser.payload", "Changed protection mode to active for protection group%{group},URL:%{url}", processor_chain([
	dup18,
	dup13,
	setc("event_description","Changed protection mode to active for protection group"),
]));

var msg55 = msg("Protection_Mode", part59);

var chain1 = processor_chain([
	select3,
	msgid_select({
		"Autoclassification": msg17,
		"BGP": select5,
		"Blocked_Host": msg53,
		"Change_Log": msg54,
		"Device": select6,
		"Flow": select4,
		"GRE": select9,
		"Hardware": select7,
		"Host": select15,
		"Infrastructure": select16,
		"Peakflow": select14,
		"Protection_Mode": msg55,
		"SNMP": select8,
		"TMS": select11,
		"Test": msg38,
		"anomaly": select13,
		"configuration": msg16,
		"mitigation": select10,
		"script": msg39,
		"usage_alert": select12,
	}),
]);

var part60 = match("HEADER#1:0002/1_0", "nwparser.p0", "high %{p0}");

var part61 = match("HEADER#1:0002/1_1", "nwparser.p0", "low %{p0}");

var part62 = match("HEADER#2:0008/2", "nwparser.p0", "%{} %{p0}");

var part63 = match("HEADER#2:0008/3_0", "nwparser.p0", "jitter %{p0}");

var part64 = match("HEADER#2:0008/3_1", "nwparser.p0", "loss %{p0}");

var part65 = match("HEADER#2:0008/3_2", "nwparser.p0", "bps %{p0}");

var part66 = match("HEADER#2:0008/3_3", "nwparser.p0", "pps %{p0}");

var part67 = match("HEADER#3:0003/4", "nwparser.p0", "%{} %{msgIdPart1->} %{msgIdPart2->} %{p0}");

var part68 = match("MESSAGE#19:mitigation:TMS_Start/1_0", "nwparser.p0", "%{fld21}, %{p0}");

var part69 = match("MESSAGE#19:mitigation:TMS_Start/1_1", "nwparser.p0", ", %{p0}");

var part70 = match("MESSAGE#19:mitigation:TMS_Start/2", "nwparser.p0", "leader %{parent_node}");

var part71 = match("MESSAGE#39:anomaly:Resource_Info:01/1_0", "nwparser.p0", "%{fld21->} duration %{p0}");

var part72 = match("MESSAGE#39:anomaly:Resource_Info:01/1_1", "nwparser.p0", "duration %{p0}");

var part73 = match("MESSAGE#39:anomaly:Resource_Info:01/2", "nwparser.p0", "%{duration->} percent %{fld3->} rate %{fld4->} rateUnit %{fld5->} protocol %{protocol->} flags %{fld6->} url %{url}, %{info}");

var part74 = match("MESSAGE#40:anomaly:Resource_Info:02/2", "nwparser.p0", "%{duration->} percent %{fld3->} rate %{fld4->} rateUnit %{fld5->} protocol %{protocol->} flags %{fld6->} url %{url}");

var hdr6 = match("HEADER#0:0001/0", "message", "%{hmonth->} %{hday->} %{htime->} %{hdata}: %{p0}", processor_chain([
	dup1,
]));

var select17 = linear_select([
	dup2,
	dup3,
]);

var select18 = linear_select([
	dup6,
	dup7,
	dup8,
	dup9,
]);

var part75 = match("MESSAGE#2:BGP:Down", "nwparser.payload", "%{protocol->} down for router %{node}, leader %{parent_node->} since %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup12,
	dup13,
	dup14,
]));

var part76 = match("MESSAGE#3:BGP:Restored", "nwparser.payload", "%{protocol->} restored for router %{node}, leader %{parent_node->} at %{fld15}-%{fld16}-%{fld17->} %{fld18}:%{fld19}:%{fld20->} %{fld21}", processor_chain([
	dup15,
	dup13,
	dup16,
]));

var select19 = linear_select([
	dup21,
	dup22,
]);

var select20 = linear_select([
	dup31,
	dup32,
]);
