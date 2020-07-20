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

var dup1 = match("HEADER#1:0022/0", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{p0}");

var dup2 = match("HEADER#1:0022/1_1", "nwparser.p0", "%{hpriority}][%{p0}");

var dup3 = match("HEADER#1:0022/1_2", "nwparser.p0", "%{hpriority}[%{p0}");

var dup4 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(": "),
		field("payload"),
	],
});

var dup5 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" "),
		field("payload"),
	],
});

var dup6 = match("HEADER#18:0034/0", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}]%{p0}");

var dup7 = match("HEADER#18:0034/1_0", "nwparser.p0", " [%{p0}");

var dup8 = match("HEADER#18:0034/1_1", "nwparser.p0", "[%{p0}");

var dup9 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(": "),
		field("hfld1"),
		constant(" "),
		field("payload"),
	],
});

var dup10 = call({
	dest: "nwparser.messageid",
	fn: STRCAT,
	args: [
		field("msgIdPart1"),
		constant("_"),
		field("msgIdPart2"),
	],
});

var dup11 = setc("eventcategory","1614000000");

var dup12 = setc("ec_activity","Scan");

var dup13 = setc("ec_theme","TEV");

var dup14 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup15 = setf("msg","$MSG");

var dup16 = setf("obj_name","hobj_name");

var dup17 = setc("obj_type","Asset");

var dup18 = setc("eventcategory","1614030000");

var dup19 = setc("ec_outcome","Error");

var dup20 = setc("eventcategory","1605000000");

var dup21 = setc("ec_activity","Start");

var dup22 = setc("ec_outcome","Success");

var dup23 = setc("eventcategory","1611000000");

var dup24 = setc("ec_activity","Stop");

var dup25 = setc("action","Shutting down");

var dup26 = setc("action","shutting down");

var dup27 = setc("ec_outcome","Failure");

var dup28 = match("MESSAGE#17:NSE:01/0", "nwparser.payload", "%{} %{p0}");

var dup29 = setf("fld17","hfld17");

var dup30 = setf("group_object","hsite");

var dup31 = setf("shost","hshost");

var dup32 = setf("sport","hsport");

var dup33 = setf("protocol","hprotocol");

var dup34 = setf("fld18","hinfo");

var dup35 = setc("ec_subject","Service");

var dup36 = setc("event_description","Nexpose is changing the database port number");

var dup37 = setc("event_state","DONE");

var dup38 = setc("event_description","Nexpose is executing data transfer process");

var dup39 = setc("event_description","Nexpose is installing the database");

var dup40 = match("MESSAGE#52:Scan:06/0", "nwparser.payload", "Scan: [ %{p0}");

var dup41 = match("MESSAGE#52:Scan:06/1_0", "nwparser.p0", "%{saddr}:%{sport->} %{p0}");

var dup42 = match("MESSAGE#52:Scan:06/1_1", "nwparser.p0", "%{saddr->} %{p0}");

var dup43 = setc("ec_outcome","Unknown");

var dup44 = setc("eventcategory","1701000000");

var dup45 = setc("ec_subject","User");

var dup46 = setc("ec_activity","Logon");

var dup47 = setc("ec_theme","Authentication");

var dup48 = setc("eventcategory","1401030000");

var dup49 = setc("ec_subject","NetworkComm");

var dup50 = setc("ec_subject","Group");

var dup51 = setc("ec_activity","Detect");

var dup52 = setc("ec_theme","Configuration");

var dup53 = setc("eventcategory","1801010000");

var dup54 = setf("obj_type","messageid");

var dup55 = setc("event_description","Cannot preload incremental pool with a connection");

var dup56 = setc("eventcategory","1605030000");

var dup57 = setc("ec_activity","Modify");

var dup58 = setc("action","Replaced conf values");

var dup59 = setc("service","fld1");

var dup60 = linear_select([
	dup7,
	dup8,
]);

var dup61 = match("MESSAGE#416:Nexpose:12", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var dup62 = match("MESSAGE#46:SPIDER", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var dup63 = linear_select([
	dup41,
	dup42,
]);

var dup64 = match("MESSAGE#93:Attempting", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var dup65 = match("MESSAGE#120:path", "nwparser.payload", "%{info}", processor_chain([
	dup20,
	dup15,
]));

var dup66 = match("MESSAGE#318:Loaded:01", "nwparser.payload", "%{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var dup67 = match("MESSAGE#236:Finished:03", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup15,
]));

var dup68 = match("MESSAGE#418:Mobile", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup25,
]));

var dup69 = match("MESSAGE#435:ConsoleProductInfoProvider", "nwparser.payload", "%{fld1->} %{action}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup59,
]));

var hdr1 = match("HEADER#0:0031", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] %{hfld39}[Thread: %{messageid}] [Started: %{hfld40}] [Duration: %{hfld41}] %{payload}", processor_chain([
	setc("header_id","0031"),
]));

var part1 = match("HEADER#1:0022/1_0", "nwparser.p0", "%{hpriority}] %{hfld39}[%{p0}");

var select1 = linear_select([
	part1,
	dup2,
	dup3,
]);

var part2 = match("HEADER#1:0022/2", "nwparser.p0", "Thread: %{hfld17}] %{messageid->} %{payload}");

var all1 = all_match({
	processors: [
		dup1,
		select1,
		part2,
	],
	on_success: processor_chain([
		setc("header_id","0022"),
	]),
});

var hdr2 = match("HEADER#2:0028", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] %{messageid}: %{payload}", processor_chain([
	setc("header_id","0028"),
	dup4,
]));

var hdr3 = match("HEADER#3:0017", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0017"),
	dup5,
]));

var hdr4 = match("HEADER#4:0024", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] %{hfld41->} %{messageid->} completed %{payload}", processor_chain([
	setc("header_id","0024"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" completed "),
			field("payload"),
		],
	}),
]));

var hdr5 = match("HEADER#5:0018", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] [%{hshost}:%{hsport}/%{hprotocol}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0018"),
	dup5,
]));

var hdr6 = match("HEADER#6:0029", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Silo ID: %{hfld22}] [Site: %{hsite}] [Site ID: %{hinfo}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0029"),
	dup5,
]));

var hdr7 = match("HEADER#7:0019", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] [%{hshost}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0019"),
	dup5,
]));

var hdr8 = match("HEADER#8:0020", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] [%{hshost}:%{hsport}/%{hprotocol}] [%{hinfo}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0020"),
	dup5,
]));

var hdr9 = match("HEADER#9:0021", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] [%{hshost}] [%{hinfo}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0021"),
	dup5,
]));

var hdr10 = match("HEADER#10:0023", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Site: %{hsite}] [%{hshost}] [%{hinfo}]: %{messageid->} %{payload}", processor_chain([
	setc("header_id","0023"),
	dup5,
]));

var hdr11 = match("HEADER#11:0036", "message", "%NEXPOSE-%{hfld49}: %{hfld1}: %{messageid->} %{hfld2->} %{payload}", processor_chain([
	setc("header_id","0036"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr12 = match("HEADER#12:0001", "message", "%NEXPOSE-%{hfld49}: %{messageid->} %{hdate}T%{htime->} [%{hobj_name}] %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr13 = match("HEADER#13:0037", "message", "%NEXPOSE-%{hfld49}: %{messageid->} %{hfld1->} '%{hfld2}' - %{hfld1->} %{payload}", processor_chain([
	setc("header_id","0037"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("hfld1"),
			constant(" '"),
			field("hfld2"),
			constant("' - "),
			field("hfld1"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr14 = match("HEADER#14:0002", "message", "%NEXPOSE-%{hfld49}: %{messageid->} %{hdate}T%{htime->} %{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr15 = match("HEADER#15:0003", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] (%{hfld41}) %{messageid->} %{payload}", processor_chain([
	setc("header_id","0003"),
	dup5,
]));

var hdr16 = match("HEADER#16:0030", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] %{messageid}: %{payload}", processor_chain([
	setc("header_id","0030"),
	dup4,
]));

var hdr17 = match("HEADER#17:0040", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}] [Thread: %{hfld17}] [Principal: %{username}] [%{messageid}: %{payload}", processor_chain([
	setc("header_id","0040"),
]));

var part3 = match("HEADER#18:0034/2", "nwparser.p0", "Thread: %{hfld17}] [%{hfld18}] [%{hfld19}] %{messageid->} %{hfld21->} %{payload}");

var all2 = all_match({
	processors: [
		dup6,
		dup60,
		part3,
	],
	on_success: processor_chain([
		setc("header_id","0034"),
	]),
});

var part4 = match("HEADER#19:0035/1_0", "nwparser.p0", "%{hpriority}] [%{p0}");

var select2 = linear_select([
	part4,
	dup2,
	dup3,
]);

var part5 = match("HEADER#19:0035/2", "nwparser.p0", "Thread: %{hfld17}] [%{hfld18}] %{messageid->} %{hfld21->} %{payload}");

var all3 = all_match({
	processors: [
		dup1,
		select2,
		part5,
	],
	on_success: processor_chain([
		setc("header_id","0035"),
	]),
});

var hdr18 = match("HEADER#20:0004", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{messageid->} %{payload}", processor_chain([
	setc("header_id","0004"),
	dup5,
]));

var part6 = match("HEADER#21:0032/2", "nwparser.p0", "Thread: %{hfld17}] [Silo ID: %{hfld18}] [Report: %{hobj_name}] [%{messageid->} Config ID: %{hfld19}] %{payload}");

var all4 = all_match({
	processors: [
		dup6,
		dup60,
		part6,
	],
	on_success: processor_chain([
		setc("header_id","0032"),
	]),
});

var hdr19 = match("HEADER#22:0038", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{messageid}: %{hfld1->} %{payload}", processor_chain([
	setc("header_id","0038"),
	dup9,
]));

var hdr20 = match("HEADER#23:0039", "message", "%NEXPOSE-%{hfld49}: %{messageid}: %{hfld1->} %{payload}", processor_chain([
	setc("header_id","0039"),
	dup9,
]));

var hdr21 = match("HEADER#24:0005", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{hfld48->} %{hfld41->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0005"),
	dup5,
]));

var hdr22 = match("HEADER#25:0006", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] [%{messageid}] %{payload}", processor_chain([
	setc("header_id","0006"),
]));

var part7 = match("HEADER#26:0033/2", "nwparser.p0", "Thread: %{hfld17}] [%{hfld18}] [%{hfld19}] [%{p0}");

var part8 = match("HEADER#26:0033/3_0", "nwparser.p0", "%{hfld20}] [%{hfld21}] [%{hfld22}] [%{hfld23}]%{p0}");

var part9 = match("HEADER#26:0033/3_1", "nwparser.p0", "%{hfld20}] [%{hfld21}]%{p0}");

var part10 = match("HEADER#26:0033/3_2", "nwparser.p0", "%{hfld20}]%{p0}");

var select3 = linear_select([
	part8,
	part9,
	part10,
]);

var part11 = match("HEADER#26:0033/4", "nwparser.p0", "%{} %{messageid->} %{hfld24->} %{payload}");

var all5 = all_match({
	processors: [
		dup6,
		dup60,
		part7,
		select3,
		part11,
	],
	on_success: processor_chain([
		setc("header_id","0033"),
	]),
});

var hdr23 = match("HEADER#27:0007", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0007"),
	dup5,
]));

var hdr24 = match("HEADER#28:0008", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] (%{messageid}) %{payload}", processor_chain([
	setc("header_id","0008"),
]));

var hdr25 = match("HEADER#29:0009", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{fld41->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0009"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("fld41"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr26 = match("HEADER#30:0010", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{messageid}: %{payload}", processor_chain([
	setc("header_id","0010"),
	dup4,
]));

var hdr27 = match("HEADER#31:0011", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} %{messageid}(%{hobj_name}): %{payload}", processor_chain([
	setc("header_id","0011"),
]));

var hdr28 = match("HEADER#32:0012", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} %{hfld41->} %{hfld42->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0012"),
	dup5,
]));

var hdr29 = match("HEADER#33:0013", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{hfld45->} (%{hfld46}) - %{msgIdPart1->} %{msgIdPart2->} %{msgIdPart3->} %{payload}", processor_chain([
	setc("header_id","0013"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("msgIdPart1"),
			constant("_"),
			field("msgIdPart2"),
			constant("_"),
			field("msgIdPart3"),
		],
	}),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld45"),
			constant(" ("),
			field("hfld46"),
			constant(") - "),
			field("msgIdPart1"),
			constant(" "),
			field("msgIdPart2"),
			constant(" "),
			field("msgIdPart3"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr30 = match("HEADER#34:0014", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{hfld45->} (%{hfld46}) - %{msgIdPart1->} %{msgIdPart2->} %{payload}", processor_chain([
	setc("header_id","0014"),
	dup10,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld45"),
			constant(" ("),
			field("hfld46"),
			constant(") - "),
			field("msgIdPart1"),
			constant(" "),
			field("msgIdPart2"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr31 = match("HEADER#35:0015", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{hfld45->} (%{hfld46}) - %{messageid->} %{payload}", processor_chain([
	setc("header_id","0015"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld45"),
			constant(" ("),
			field("hfld46"),
			constant(") - "),
			field("messageid"),
			constant(" "),
			field("payload"),
		],
	}),
]));

var hdr32 = match("HEADER#36:0016", "message", "%NEXPOSE-%{hfld49}: %{hfld40->} %{hdate}T%{htime->} [%{hobj_name}] %{hfld45->} (%{hfld46}) - %{msgIdPart1->} %{msgIdPart2}(U) %{payload}", processor_chain([
	setc("header_id","0016"),
	dup10,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld45"),
			constant(" ("),
			field("hfld46"),
			constant(") - "),
			field("msgIdPart1"),
			constant(" "),
			field("msgIdPart2"),
			constant("(U) "),
			field("payload"),
		],
	}),
]));

var hdr33 = match("HEADER#37:0026", "message", "%NEXPOSE-%{hfld49}: %{messageid->} Constructor threw %{payload}", processor_chain([
	setc("header_id","0026"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" Constructor threw "),
			field("payload"),
		],
	}),
]));

var hdr34 = match("HEADER#38:0027", "message", "%NEXPOSE-%{hfld49}: %{messageid->} Called method %{payload}", processor_chain([
	setc("header_id","0027"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" Called method "),
			field("payload"),
		],
	}),
]));

var hdr35 = match("HEADER#39:0025", "message", "%NEXPOSE-%{hfld49}: %{hfld41->} %{hfld42->} %{messageid->} frames %{payload}", processor_chain([
	setc("header_id","0025"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" frames "),
			field("payload"),
		],
	}),
]));

var hdr36 = match("HEADER#40:9999", "message", "%NEXPOSE-%{hfld49}: %{payload}", processor_chain([
	setc("header_id","9999"),
	setc("messageid","NEXPOSE_GENERIC"),
]));

var select4 = linear_select([
	hdr1,
	all1,
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
	hdr13,
	hdr14,
	hdr15,
	hdr16,
	hdr17,
	all2,
	all3,
	hdr18,
	all4,
	hdr19,
	hdr20,
	hdr21,
	hdr22,
	all5,
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
	hdr35,
	hdr36,
]);

var part12 = match("MESSAGE#0:NOT_VULNERABLE_VERSION", "nwparser.payload", "%{signame->} - NOT VULNERABLE VERSION .", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg1 = msg("NOT_VULNERABLE_VERSION", part12);

var part13 = match("MESSAGE#1:VULNERABLE_VERSION", "nwparser.payload", "%{signame->} - VULNERABLE VERSION .", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg2 = msg("VULNERABLE_VERSION", part13);

var part14 = match("MESSAGE#2:NOT_VULNERABLE", "nwparser.payload", "%{signame->} - NOT VULNERABLE [UNIQUE ID: %{fld45}]", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg3 = msg("NOT_VULNERABLE", part14);

var part15 = match("MESSAGE#3:NOT_VULNERABLE:01", "nwparser.payload", "%{signame->} - NOT VULNERABLE(U) [UNIQUE ID: %{fld45}]", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg4 = msg("NOT_VULNERABLE:01", part15);

var part16 = match("MESSAGE#4:NOT_VULNERABLE:02", "nwparser.payload", "%{signame->} - NOT VULNERABLE .", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg5 = msg("NOT_VULNERABLE:02", part16);

var select5 = linear_select([
	msg3,
	msg4,
	msg5,
]);

var part17 = match("MESSAGE#5:VULNERABLE", "nwparser.payload", "%{signame->} - VULNERABLE [UNIQUE ID: %{fld45}]", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg6 = msg("VULNERABLE", part17);

var part18 = match("MESSAGE#6:VULNERABLE:01", "nwparser.payload", "%{signame->} - VULNERABLE .", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg7 = msg("VULNERABLE:01", part18);

var select6 = linear_select([
	msg6,
	msg7,
]);

var part19 = match("MESSAGE#7:ERROR", "nwparser.payload", "%{signame->} - ERROR [UNIQUE ID: %{fld45}] - %{context}", processor_chain([
	dup18,
	dup12,
	dup13,
	dup19,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg8 = msg("ERROR", part19);

var part20 = match("MESSAGE#8:ERROR:01", "nwparser.payload", "%{signame->} - ERROR - %{context}", processor_chain([
	dup18,
	dup12,
	dup13,
	dup19,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg9 = msg("ERROR:01", part20);

var select7 = linear_select([
	msg8,
	msg9,
]);

var part21 = match("MESSAGE#9:ExtMgr", "nwparser.payload", "Initialization successful.%{}", processor_chain([
	dup20,
	dup21,
	dup13,
	dup22,
	dup14,
	dup15,
	setc("event_description","Initialization successful"),
]));

var msg10 = msg("ExtMgr", part21);

var part22 = match("MESSAGE#10:ExtMgr:01", "nwparser.payload", "initializing...%{}", processor_chain([
	dup20,
	dup21,
	dup13,
	dup14,
	dup15,
	setc("event_description","initializing"),
]));

var msg11 = msg("ExtMgr:01", part22);

var part23 = match("MESSAGE#11:ExtMgr:02", "nwparser.payload", "Shutdown successful.%{}", processor_chain([
	dup23,
	dup24,
	dup13,
	dup22,
	dup14,
	dup15,
	setc("event_description","Shutdown successful."),
]));

var msg12 = msg("ExtMgr:02", part23);

var part24 = match("MESSAGE#12:ExtMgr:03", "nwparser.payload", "Shutting down...%{}", processor_chain([
	dup23,
	dup24,
	dup13,
	dup14,
	dup15,
	dup25,
]));

var msg13 = msg("ExtMgr:03", part24);

var select8 = linear_select([
	msg10,
	msg11,
	msg12,
	msg13,
]);

var part25 = match("MESSAGE#13:ScanMgr", "nwparser.payload", "Shutting down %{info}", processor_chain([
	dup20,
	dup24,
	dup13,
	dup14,
	dup15,
	dup25,
]));

var msg14 = msg("ScanMgr", part25);

var part26 = match("MESSAGE#14:ScanMgr:01", "nwparser.payload", "shutting down...%{}", processor_chain([
	dup23,
	dup24,
	dup13,
	dup14,
	dup15,
	dup26,
]));

var msg15 = msg("ScanMgr:01", part26);

var part27 = match("MESSAGE#15:ScanMgr:02", "nwparser.payload", "Scan %{fld30->} is being stopped.", processor_chain([
	dup20,
	dup12,
	dup13,
	dup27,
	dup14,
	dup15,
]));

var msg16 = msg("ScanMgr:02", part27);

var select9 = linear_select([
	msg14,
	msg15,
	msg16,
]);

var part28 = match("MESSAGE#16:NSE", "nwparser.payload", "Logging initialized %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Logging initialized"),
]));

var msg17 = msg("NSE", part28);

var part29 = match("MESSAGE#17:NSE:01/1_0", "nwparser.p0", "Initializing %{p0}");

var part30 = match("MESSAGE#17:NSE:01/1_1", "nwparser.p0", "initializing %{p0}");

var select10 = linear_select([
	part29,
	part30,
]);

var part31 = match("MESSAGE#17:NSE:01/2", "nwparser.p0", "%{} %{fld30}");

var all6 = all_match({
	processors: [
		dup28,
		select10,
		part31,
	],
	on_success: processor_chain([
		dup20,
		dup14,
		dup15,
		setc("action","Initializing"),
	]),
});

var msg18 = msg("NSE:01", all6);

var part32 = match("MESSAGE#18:NSE:02", "nwparser.payload", "shutting down %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup26,
]));

var msg19 = msg("NSE:02", part32);

var part33 = match("MESSAGE#19:NSE:03", "nwparser.payload", "NeXpose scan engine initialization completed.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","NeXpose scan engine initialization completed."),
]));

var msg20 = msg("NSE:03", part33);

var part34 = match("MESSAGE#20:NSE:04", "nwparser.payload", "disabling promiscuous on all devices...%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","disabling promiscuous on all devices"),
]));

var msg21 = msg("NSE:04", part34);

var part35 = match("MESSAGE#213:NSE:05", "nwparser.payload", "NSE connection failure%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg22 = msg("NSE:05", part35);

var part36 = match("MESSAGE#328:NSE:07", "nwparser.payload", "NSE DN is %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg23 = msg("NSE:07", part36);

var select11 = linear_select([
	msg17,
	msg18,
	msg19,
	msg20,
	msg21,
	msg22,
	msg23,
]);

var part37 = match("MESSAGE#21:Console", "nwparser.payload", "NSE Name: %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg24 = msg("Console", part37);

var part38 = match("MESSAGE#22:Console:01", "nwparser.payload", "NSE Identifier: %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg25 = msg("Console:01", part38);

var part39 = match("MESSAGE#23:Console:02", "nwparser.payload", "NSE version: %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg26 = msg("Console:02", part39);

var part40 = match("MESSAGE#24:Console:03", "nwparser.payload", "Last update: %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg27 = msg("Console:03", part40);

var part41 = match("MESSAGE#25:Console:04", "nwparser.payload", "VM version: %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg28 = msg("Console:04", part41);

var part42 = match("MESSAGE#26:Console:05", "nwparser.payload", "log rotation completed%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","log rotation completed"),
]));

var msg29 = msg("Console:05", part42);

var part43 = match("MESSAGE#27:Console:06", "nwparser.payload", "rotating logs...%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","rotating logs"),
]));

var msg30 = msg("Console:06", part43);

var select12 = linear_select([
	msg24,
	msg25,
	msg26,
	msg27,
	msg28,
	msg29,
	msg30,
]);

var part44 = match("MESSAGE#28:ProtocolFper", "nwparser.payload", "Loaded %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Loaded"),
]));

var msg31 = msg("ProtocolFper", part44);

var part45 = match("MESSAGE#29:Nexpose", "nwparser.payload", "Closing service: %{fld30}", processor_chain([
	dup20,
	dup35,
	dup24,
	dup14,
	dup15,
	dup16,
	dup17,
	setc("action","Closing service"),
]));

var msg32 = msg("Nexpose", part45);

var part46 = match("MESSAGE#30:Nexpose:01", "nwparser.payload", "Freeing %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
	setc("action","Freeing"),
]));

var msg33 = msg("Nexpose:01", part46);

var part47 = match("MESSAGE#31:Nexpose:02", "nwparser.payload", "starting %{fld30}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup16,
	dup17,
	setc("action","starting"),
]));

var msg34 = msg("Nexpose:02", part47);

var part48 = match("MESSAGE#32:Nexpose:03", "nwparser.payload", "%{fld31->} nodes completed, %{fld32->} active, %{fld33->} pending.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg35 = msg("Nexpose:03", part48);

var part49 = match("MESSAGE#373:Backup_completed", "nwparser.payload", "Nexpose system backup completed successfully in %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Backup completed"),
]));

var msg36 = msg("Backup_completed", part49);

var part50 = match("MESSAGE#408:Nexpose:04", "nwparser.payload", "Nexpose is changing the database port number from %{change_old->} to %{change_new}. DONE.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup36,
	dup37,
]));

var msg37 = msg("Nexpose:04", part50);

var part51 = match("MESSAGE#409:Nexpose:05", "nwparser.payload", "Nexpose is changing the database port number from %{change_old->} to %{change_new}.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup36,
]));

var msg38 = msg("Nexpose:05", part51);

var part52 = match("MESSAGE#410:Nexpose:06", "nwparser.payload", "Nexpose is executing the data transfer process from %{change_old->} to %{change_new->} DONE.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup38,
	dup37,
]));

var msg39 = msg("Nexpose:06", part52);

var part53 = match("MESSAGE#411:Nexpose:07", "nwparser.payload", "Nexpose is executing the data transfer process from %{change_old->} to %{change_new}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup38,
]));

var msg40 = msg("Nexpose:07", part53);

var part54 = match("MESSAGE#412:Nexpose:08", "nwparser.payload", "Nexpose is installing the %{db_name->} database. DONE.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup39,
	dup37,
]));

var msg41 = msg("Nexpose:08", part54);

var part55 = match("MESSAGE#413:Nexpose:09", "nwparser.payload", "Nexpose is installing the %{db_name->} database to %{directory->} using PostgreSQL binaries from package %{filename}.%{fld1}.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup39,
]));

var msg42 = msg("Nexpose:09", part55);

var part56 = match("MESSAGE#414:Nexpose:10", "nwparser.payload", "Nexpose is moving %{change_old->} to %{change_new}.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Nexpose is moving a directory"),
]));

var msg43 = msg("Nexpose:10", part56);

var part57 = match("MESSAGE#415:Nexpose:11", "nwparser.payload", "%{event_description->} DONE.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup37,
]));

var msg44 = msg("Nexpose:11", part57);

var msg45 = msg("Nexpose:12", dup61);

var select13 = linear_select([
	msg32,
	msg33,
	msg34,
	msg35,
	msg36,
	msg37,
	msg38,
	msg39,
	msg40,
	msg41,
	msg42,
	msg43,
	msg44,
	msg45,
]);

var part58 = match("MESSAGE#33:Shutting", "nwparser.payload", "Shutting down %{fld30}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup16,
	dup17,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	dup25,
]));

var msg46 = msg("Shutting", part58);

var part59 = match("MESSAGE#34:shutting:01", "nwparser.payload", "Interrupted, %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg47 = msg("shutting:01", part59);

var part60 = match("MESSAGE#35:shutting", "nwparser.payload", "shutting down %{fld30}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup16,
	dup17,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	dup26,
]));

var msg48 = msg("shutting", part60);

var part61 = match("MESSAGE#36:Shutdown", "nwparser.payload", "Shutdown successful.%{}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup16,
	dup17,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	dup25,
]));

var msg49 = msg("Shutdown", part61);

var part62 = match("MESSAGE#37:Security", "nwparser.payload", "Security Console shutting down.%{}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup29,
	dup25,
]));

var msg50 = msg("Security", part62);

var part63 = match("MESSAGE#261:Security:02", "nwparser.payload", "Security Console restarting from an auto-update%{}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg51 = msg("Security:02", part63);

var part64 = match("MESSAGE#296:Security:06", "nwparser.payload", "Started: %{fld1}] [Duration: %{fld2}] Security Console started", processor_chain([
	dup20,
	dup15,
]));

var msg52 = msg("Security:06", part64);

var part65 = match("MESSAGE#297:Security:03/0", "nwparser.payload", "%{}Security Console %{p0}");

var part66 = match("MESSAGE#297:Security:03/1_0", "nwparser.p0", "started %{}");

var part67 = match("MESSAGE#297:Security:03/1_1", "nwparser.p0", "web interface ready. %{info->} ");

var select14 = linear_select([
	part66,
	part67,
]);

var all7 = all_match({
	processors: [
		part65,
		select14,
	],
	on_success: processor_chain([
		dup20,
		dup15,
	]),
});

var msg53 = msg("Security:03", all7);

var part68 = match("MESSAGE#426:Security:04", "nwparser.payload", "Security Console is launching in Maintenance Mode. %{action}.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Security Console is launching in Maintenance Mode"),
]));

var msg54 = msg("Security:04", part68);

var part69 = match("MESSAGE#427:Security:05", "nwparser.payload", "Security Console update failed.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Security Console update failed"),
]));

var msg55 = msg("Security:05", part69);

var select15 = linear_select([
	msg50,
	msg51,
	msg52,
	msg53,
	msg54,
	msg55,
]);

var part70 = match("MESSAGE#38:Web", "nwparser.payload", "Web server stopped%{}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup16,
	dup17,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("action","Stopped"),
]));

var msg56 = msg("Web", part70);

var part71 = match("MESSAGE#304:Web:02", "nwparser.payload", "Web %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg57 = msg("Web:02", part71);

var select16 = linear_select([
	msg56,
	msg57,
]);

var part72 = match("MESSAGE#39:Done", "nwparser.payload", "Done shutting down.%{}", processor_chain([
	dup23,
	dup14,
	dup15,
	dup16,
	dup17,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	dup26,
]));

var msg58 = msg("Done", part72);

var part73 = match("MESSAGE#282:Done:02", "nwparser.payload", "Done with statistics generation [Started: %{fld1}] [Duration: %{fld2}].", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg59 = msg("Done:02", part73);

var select17 = linear_select([
	msg58,
	msg59,
]);

var part74 = match("MESSAGE#40:Queueing:01", "nwparser.payload", "Queueing %{protocol->} port scan", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg60 = msg("Queueing:01", part74);

var part75 = match("MESSAGE#41:Queueing", "nwparser.payload", "Queueing %{fld30}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
	setc("action","Queueing"),
]));

var msg61 = msg("Queueing", part75);

var select18 = linear_select([
	msg60,
	msg61,
]);

var part76 = match("MESSAGE#42:Performing/0", "nwparser.payload", "Performing %{p0}");

var part77 = match("MESSAGE#42:Performing/1_0", "nwparser.p0", "form %{p0}");

var part78 = match("MESSAGE#42:Performing/1_1", "nwparser.p0", "query %{p0}");

var select19 = linear_select([
	part77,
	part78,
]);

var part79 = match("MESSAGE#42:Performing/2", "nwparser.p0", "%{}injection against %{info}");

var all8 = all_match({
	processors: [
		part76,
		select19,
		part79,
	],
	on_success: processor_chain([
		dup20,
		dup12,
		dup13,
		dup14,
		dup15,
		dup16,
		dup17,
		setc("action","Performing injection"),
	]),
});

var msg62 = msg("Performing", all8);

var part80 = match("MESSAGE#43:Performing:01", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg63 = msg("Performing:01", part80);

var select20 = linear_select([
	msg62,
	msg63,
]);

var part81 = match("MESSAGE#44:Trying", "nwparser.payload", "Trying %{fld30->} injection %{fld31}", processor_chain([
	dup20,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
	setc("action","Trying injection"),
]));

var msg64 = msg("Trying", part81);

var part82 = match("MESSAGE#45:Rewrote", "nwparser.payload", "Rewrote to %{url}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg65 = msg("Rewrote", part82);

var msg66 = msg("SPIDER", dup62);

var msg67 = msg("Preparing", dup62);

var part83 = match("MESSAGE#48:Scan", "nwparser.payload", "Scan started by: \"%{username}\" %{fld34}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("action","scan started"),
]));

var msg68 = msg("Scan", part83);

var part84 = match("MESSAGE#49:Scan:01", "nwparser.payload", "Scan [%{fld35}] completed in %{fld36}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	setc("action","scan completed"),
]));

var msg69 = msg("Scan:01", part84);

var part85 = match("MESSAGE#50:Scan:03", "nwparser.payload", "Scan for site %{fld11->} started by Schedule[%{info}].", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg70 = msg("Scan:03", part85);

var part86 = match("MESSAGE#51:Scan:04", "nwparser.payload", "Scan startup took %{fld24->} seconds", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg71 = msg("Scan:04", part86);

var part87 = match("MESSAGE#52:Scan:06/2", "nwparser.p0", "] %{fld12->} (%{info}) - VULNERABLE VERSION");

var all9 = all_match({
	processors: [
		dup40,
		dup63,
		part87,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg72 = msg("Scan:06", all9);

var part88 = match("MESSAGE#53:Scan:05/2", "nwparser.p0", "] %{fld12->} (%{info}) - VULNERABLE");

var all10 = all_match({
	processors: [
		dup40,
		dup63,
		part88,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg73 = msg("Scan:05", all10);

var part89 = match("MESSAGE#54:Scan:07/2", "nwparser.p0", "] %{fld12->} (%{info}) - NOT VULNERABLE VERSION");

var all11 = all_match({
	processors: [
		dup40,
		dup63,
		part89,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg74 = msg("Scan:07", all11);

var part90 = match("MESSAGE#55:Scan:09/2", "nwparser.p0", "] %{fld12->} (%{info}) - NOT VULNERABLE [UNIQUE ID: %{fld13}]");

var all12 = all_match({
	processors: [
		dup40,
		dup63,
		part90,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg75 = msg("Scan:09", all12);

var part91 = match("MESSAGE#56:Scan:08/2", "nwparser.p0", "] %{fld12->} (%{info}) - NOT VULNERABLE");

var all13 = all_match({
	processors: [
		dup40,
		dup63,
		part91,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg76 = msg("Scan:08", all13);

var part92 = match("MESSAGE#57:Scan:10", "nwparser.payload", "Scan for site %{fld12->} started by \"%{username}\".", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg77 = msg("Scan:10", part92);

var part93 = match("MESSAGE#58:Scan:11", "nwparser.payload", "Scan stopped: \"%{username}\"", processor_chain([
	dup18,
	dup12,
	dup13,
	dup14,
	dup15,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg78 = msg("Scan:11", part93);

var part94 = match("MESSAGE#59:Scan:12", "nwparser.payload", "Scan Engine shutting down...%{}", processor_chain([
	dup23,
	dup12,
	dup13,
	dup19,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg79 = msg("Scan:12", part94);

var part95 = match("MESSAGE#60:Scan:13", "nwparser.payload", "Scan ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] Scan synopsis inconsistency resolved.", processor_chain([
	dup11,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	setc("event_description","Scan synopsis inconsistency resolved"),
]));

var msg80 = msg("Scan:13", part95);

var part96 = match("MESSAGE#62:Scan:15/0", "nwparser.payload", "Silo ID: %{fld1}] [Scan ID: %{fld2}] Scan for site %{audit_object->} - %{p0}");

var part97 = match("MESSAGE#62:Scan:15/1_0", "nwparser.p0", "Non-Windows Systems Audit%{p0}");

var part98 = match("MESSAGE#62:Scan:15/1_1", "nwparser.p0", "Audit%{p0}");

var select21 = linear_select([
	part97,
	part98,
]);

var part99 = match("MESSAGE#62:Scan:15/2", "nwparser.p0", "%{}restored. %{info}");

var all14 = all_match({
	processors: [
		part96,
		select21,
		part99,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup22,
		dup14,
		dup15,
		setc("event_description","Scan for site restored"),
	]),
});

var msg81 = msg("Scan:15", all14);

var part100 = match("MESSAGE#63:Scan:02", "nwparser.payload", "%{event_description}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg82 = msg("Scan:02", part100);

var select22 = linear_select([
	msg68,
	msg69,
	msg70,
	msg71,
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
]);

var part101 = match("MESSAGE#61:Scan:14", "nwparser.payload", "Scan ID: %{fld1}] Inconsistency discovered for scan. %{info}", processor_chain([
	dup18,
	dup12,
	dup13,
	dup43,
	dup14,
	dup15,
	setc("event_description","Inconsistency discovered for scan"),
]));

var msg83 = msg("Scan:14", part101);

var part102 = match("MESSAGE#64:Site", "nwparser.payload", "Site saved.%{}", processor_chain([
	dup44,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg84 = msg("Site", part102);

var part103 = match("MESSAGE#65:Authenticated", "nwparser.payload", "Authenticated: %{username}", processor_chain([
	setc("eventcategory","1401060000"),
	dup45,
	dup46,
	dup47,
	dup22,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg85 = msg("Authenticated", part103);

var part104 = match("MESSAGE#66:Authentication", "nwparser.payload", "Authentication failed. Login information is missing.%{}", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg86 = msg("Authentication", part104);

var part105 = match("MESSAGE#67:Authentication:01", "nwparser.payload", "Authentication failed for %{username}: Access denied.", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg87 = msg("Authentication:01", part105);

var part106 = match("MESSAGE#68:Authentication:02", "nwparser.payload", "Authentication failed. User account may be invalid or disabled.%{}", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg88 = msg("Authentication:02", part106);

var part107 = match("MESSAGE#69:Authentication:03", "nwparser.payload", "%{info}", processor_chain([
	setc("eventcategory","1304000000"),
	dup45,
	dup46,
	dup47,
	dup14,
	dup15,
	dup16,
	dup29,
]));

var msg89 = msg("Authentication:03", part107);

var select23 = linear_select([
	msg86,
	msg87,
	msg88,
	msg89,
]);

var part108 = match("MESSAGE#70:User", "nwparser.payload", "User (%{username}) is over the limit (%{fld12}) for failed login attempts.", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg90 = msg("User", part108);

var part109 = match("MESSAGE#265:User:04", "nwparser.payload", "User name: %{username}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg91 = msg("User:04", part109);

var select24 = linear_select([
	msg90,
	msg91,
]);

var msg92 = msg("persistent-xss", dup61);

var part110 = match("MESSAGE#72:Adding:01", "nwparser.payload", "Adding user to datastore: %{username}", processor_chain([
	setc("eventcategory","1402020200"),
	dup45,
	setc("ec_activity","Create"),
	dup47,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("obj_type","User"),
]));

var msg93 = msg("Adding:01", part110);

var msg94 = msg("Adding", dup62);

var select25 = linear_select([
	msg93,
	msg94,
]);

var msg95 = msg("credentials", dup62);

var msg96 = msg("SPIDER-XSS", dup62);

var msg97 = msg("Processing", dup62);

var msg98 = msg("but", dup62);

var msg99 = msg("j_password", dup62);

var msg100 = msg("j_username", dup62);

var msg101 = msg("osspi_defaultTargetLocation", dup62);

var part111 = match("MESSAGE#81:spider-parse-robot-exclusions", "nwparser.payload", "spider-parse-robot-exclusions: %{fld40->} Malformed HTTP %{fld41}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg102 = msg("spider-parse-robot-exclusions", part111);

var msg103 = msg("Cataloged", dup62);

var msg104 = msg("Dumping", dup62);

var msg105 = msg("Form", dup62);

var msg106 = msg("Relaunching", dup62);

var msg107 = msg("main", dup62);

var msg108 = msg("SystemFingerprint", dup62);

var part112 = match("MESSAGE#88:Searching", "nwparser.payload", "Searching for %{service->} domain %{fld11}...", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg109 = msg("Searching", part112);

var msg110 = msg("TCPSocket", dup62);

var part113 = match("MESSAGE#90:connected", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup49,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg111 = msg("connected", part113);

var part114 = match("MESSAGE#91:Failed", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup49,
	dup27,
	dup14,
	dup15,
]));

var msg112 = msg("Failed", part114);

var part115 = match("MESSAGE#92:Attempting:01", "nwparser.payload", "Attempting to authenticate user %{username->} from %{saddr}.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg113 = msg("Attempting:01", part115);

var msg114 = msg("Attempting", dup64);

var select26 = linear_select([
	msg113,
	msg114,
]);

var part116 = match("MESSAGE#94:Recursively:01", "nwparser.payload", "Recursively listing files on %{service}[%{info}]", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg115 = msg("Recursively:01", part116);

var msg116 = msg("Recursively", dup62);

var select27 = linear_select([
	msg115,
	msg116,
]);

var msg117 = msg("building", dup62);

var msg118 = msg("Sending", dup62);

var msg119 = msg("sending", dup64);

var part117 = match("MESSAGE#99:creating", "nwparser.payload", "creating new connection to %{obj_name}", processor_chain([
	dup20,
	dup49,
	dup14,
	dup15,
	dup17,
]));

var msg120 = msg("creating", part117);

var part118 = match("MESSAGE#100:Trusted", "nwparser.payload", "Trusted MAC address checking is disabled%{}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg121 = msg("Trusted", part118);

var part119 = match("MESSAGE#101:signon_type", "nwparser.payload", "signon_type: %{fld40}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg122 = msg("signon_type", part119);

var msg123 = msg("list-user-directory", dup62);

var msg124 = msg("dcerpc-get-ms-blaster-codes", dup62);

var msg125 = msg("Could", dup62);

var part120 = match("MESSAGE#105:Asserting", "nwparser.payload", "Asserting software fingerprint name=%{obj_name}, version=%{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("obj_type","Software Fingerprint"),
]));

var msg126 = msg("Asserting", part120);

var part121 = match("MESSAGE#106:Asserting:01", "nwparser.payload", "Asserting run entry: %{service}: %{filename}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg127 = msg("Asserting:01", part121);

var part122 = match("MESSAGE#107:Asserting:02", "nwparser.payload", "Asserting network interface: %{sinterface->} with IP: %{saddr->} and netmask: %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg128 = msg("Asserting:02", part122);

var part123 = match("MESSAGE#108:Asserting:03", "nwparser.payload", "Asserting highest MDAC version of %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg129 = msg("Asserting:03", part123);

var msg130 = msg("Asserting:04", dup62);

var select28 = linear_select([
	msg126,
	msg127,
	msg128,
	msg129,
	msg130,
]);

var part124 = match("MESSAGE#110:Determining:01", "nwparser.payload", "Determining version of file %{filename->} (%{application})", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg131 = msg("Determining:01", part124);

var msg132 = msg("Determining", dup62);

var select29 = linear_select([
	msg131,
	msg132,
]);

var part125 = match("MESSAGE#112:Webmin", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup35,
	dup27,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg133 = msg("Webmin", part125);

var part126 = match("MESSAGE#113:Running:02", "nwparser.payload", "Running unresolved %{service}", processor_chain([
	dup20,
	dup35,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg134 = msg("Running:02", part126);

var part127 = match("MESSAGE#114:Running:01", "nwparser.payload", "Running %{protocol->} service %{service}", processor_chain([
	dup20,
	dup35,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg135 = msg("Running:01", part127);

var part128 = match("MESSAGE#115:Running", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup35,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg136 = msg("Running", part128);

var select30 = linear_select([
	msg134,
	msg135,
	msg136,
]);

var part129 = match("MESSAGE#116:path:/0_0", "nwparser.payload", "Service path:%{p0}");

var part130 = match("MESSAGE#116:path:/0_1", "nwparser.payload", "path:%{p0}");

var select31 = linear_select([
	part129,
	part130,
]);

var part131 = match("MESSAGE#116:path:/1", "nwparser.p0", "%{} %{filename}");

var all15 = all_match({
	processors: [
		select31,
		part131,
	],
	on_success: processor_chain([
		dup20,
		dup15,
	]),
});

var msg137 = msg("path:", all15);

var part132 = match("MESSAGE#117:path:01", "nwparser.payload", "Service path is insecure.%{}", processor_chain([
	dup20,
	dup15,
	setc("info","Service path is insecure."),
]));

var msg138 = msg("path:01", part132);

var part133 = match("MESSAGE#118:Service", "nwparser.payload", "Service %{service->} %{action->} on Provider: %{fld2}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg139 = msg("Service", part133);

var part134 = match("MESSAGE#119:ServiceFingerprint", "nwparser.payload", "Service running: %{event_description}", processor_chain([
	dup20,
	dup35,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg140 = msg("ServiceFingerprint", part134);

var msg141 = msg("path", dup65);

var select32 = linear_select([
	msg137,
	msg138,
	msg139,
	msg140,
	msg141,
]);

var msg142 = msg("using", dup61);

var part135 = match("MESSAGE#122:Found:01", "nwparser.payload", "Found group: CIFS Group %{group}", processor_chain([
	dup20,
	dup50,
	dup51,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg143 = msg("Found:01", part135);

var part136 = match("MESSAGE#123:Found:02", "nwparser.payload", "Found user: CIFS User %{username}", processor_chain([
	dup20,
	dup45,
	dup51,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg144 = msg("Found:02", part136);

var part137 = match("MESSAGE#124:Found:03", "nwparser.payload", "Found user %{username}", processor_chain([
	dup20,
	dup45,
	dup51,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg145 = msg("Found:03", part137);

var part138 = match("MESSAGE#125:Found:04", "nwparser.payload", "Found interface %{sinterface}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg146 = msg("Found:04", part138);

var part139 = match("MESSAGE#126:Found:05", "nwparser.payload", "Found DHCP-assigned WINS server: %{saddr}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg147 = msg("Found:05", part139);

var msg148 = msg("Found", dup62);

var select33 = linear_select([
	msg143,
	msg144,
	msg145,
	msg146,
	msg147,
	msg148,
]);

var part140 = match("MESSAGE#128:FTP", "nwparser.payload", "FTP name: %{fld40}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var msg149 = msg("FTP", part140);

var part141 = match("MESSAGE#129:Starting:02", "nwparser.payload", "Starting Office fingerprinting with dir %{directory}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg150 = msg("Starting:02", part141);

var part142 = match("MESSAGE#130:Starting:01", "nwparser.payload", "Starting scan against %{fld11->} (%{fld12}) with scan template: %{fld13}.", processor_chain([
	dup20,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg151 = msg("Starting:01", part142);

var msg152 = msg("Starting", dup62);

var select34 = linear_select([
	msg150,
	msg151,
	msg152,
]);

var msg153 = msg("loading", dup61);

var part143 = match("MESSAGE#133:trying", "nwparser.payload", "trying the next key: %{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg154 = msg("trying", part143);

var msg155 = msg("Retrieving", dup64);

var part144 = match("MESSAGE#135:Got", "nwparser.payload", "Got version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
]));

var msg156 = msg("Got", part144);

var msg157 = msg("unexpected", dup64);

var part145 = match("MESSAGE#137:checking:03", "nwparser.payload", "checking version of '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg158 = msg("checking:03", part145);

var part146 = match("MESSAGE#138:No", "nwparser.payload", "No closed UDP ports, IP fingerprinting may be less accurate%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg159 = msg("No", part146);

var part147 = match("MESSAGE#139:No:01", "nwparser.payload", "No credentials available%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg160 = msg("No:01", part147);

var part148 = match("MESSAGE#140:No:02", "nwparser.payload", "No access to %{directory->} with %{service}[%{info}]", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg161 = msg("No:02", part148);

var part149 = match("MESSAGE#141:No:03", "nwparser.payload", "No approved updates found for processing.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg162 = msg("No:03", part149);

var msg163 = msg("No:04", dup61);

var select35 = linear_select([
	msg159,
	msg160,
	msg161,
	msg162,
	msg163,
]);

var part150 = match("MESSAGE#142:Applying", "nwparser.payload", "Applying update ID %{fld12}.", processor_chain([
	dup44,
	dup52,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg164 = msg("Applying", part150);

var part151 = match("MESSAGE#143:Update", "nwparser.payload", "Update ID %{fld12->} applied successfully.", processor_chain([
	dup44,
	dup52,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg165 = msg("Update", part151);

var part152 = match("MESSAGE#227:Update:02", "nwparser.payload", "Update ID %{fld1}, for product ID %{id}, %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg166 = msg("Update:02", part152);

var msg167 = msg("Update:03", dup61);

var select36 = linear_select([
	msg165,
	msg166,
	msg167,
]);

var part153 = match("MESSAGE#144:Installing", "nwparser.payload", "Installing directory %{directory}.", processor_chain([
	dup20,
	dup52,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg168 = msg("Installing", part153);

var part154 = match("MESSAGE#145:Installing:01", "nwparser.payload", "Installing file, %{filename}.", processor_chain([
	dup20,
	dup52,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg169 = msg("Installing:01", part154);

var part155 = match("MESSAGE#405:Installing:02", "nwparser.payload", "Installing Postgres files into %{directory->} from %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Installing Postgres files"),
]));

var msg170 = msg("Installing:02", part155);

var select37 = linear_select([
	msg168,
	msg169,
	msg170,
]);

var part156 = match("MESSAGE#146:Resolving", "nwparser.payload", "Resolving additional DNS records%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg171 = msg("Resolving", part156);

var part157 = match("MESSAGE#147:DNS", "nwparser.payload", "DNS name: %{obj_name}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("obj_type","DNS"),
]));

var msg172 = msg("DNS", part157);

var part158 = match("MESSAGE#148:Scanning", "nwparser.payload", "Scanning %{fld23->} %{protocol->} ports", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg173 = msg("Scanning", part158);

var msg174 = msg("param:", dup64);

var part159 = match("MESSAGE#150:Windows", "nwparser.payload", "Windows %{obj_name->} dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg175 = msg("Windows", part159);

var part160 = match("MESSAGE#151:Windows:01", "nwparser.payload", "Windows Media Player version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg176 = msg("Windows:01", part160);

var msg177 = msg("Windows:02", dup61);

var select38 = linear_select([
	msg175,
	msg176,
	msg177,
]);

var msg178 = msg("Parsed", dup64);

var part161 = match("MESSAGE#153:JRE", "nwparser.payload", "JRE version %{version->} is installed", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg179 = msg("JRE", part161);

var msg180 = msg("Microsoft", dup64);

var part162 = match("MESSAGE#155:MDAC", "nwparser.payload", "MDAC version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg181 = msg("MDAC", part162);

var part163 = match("MESSAGE#156:Name", "nwparser.payload", "Name Server: %{saddr}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg182 = msg("Name", part163);

var msg183 = msg("Flash", dup64);

var msg184 = msg("Skipping", dup64);

var part164 = match("MESSAGE#159:Closing", "nwparser.payload", "Closing service: %{service->} (source: %{info})", processor_chain([
	dup20,
	dup35,
	dup24,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg185 = msg("Closing", part164);

var part165 = match("MESSAGE#238:Closing:03", "nwparser.payload", "Engine: %{fld1}] [Engine ID: %{fld3}] Closing connection to scan engine.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Closing connection to scan engine"),
]));

var msg186 = msg("Closing:03", part165);

var msg187 = msg("Closing:02", dup61);

var select39 = linear_select([
	msg185,
	msg186,
	msg187,
]);

var part166 = match("MESSAGE#160:key", "nwparser.payload", "key does not exist: %{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg188 = msg("key", part166);

var part167 = match("MESSAGE#161:Listing", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup50,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg189 = msg("Listing", part167);

var msg190 = msg("Getting", dup64);

var part168 = match("MESSAGE#163:Version:", "nwparser.payload", "Version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg191 = msg("Version:", part168);

var msg192 = msg("IE", dup64);

var part169 = match("MESSAGE#165:Completed", "nwparser.payload", "Completed %{protocol->} port scan (%{dclass_counter1->} open ports): %{fld11->} seconds", processor_chain([
	dup20,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","No. of Open ports"),
]));

var msg193 = msg("Completed", part169);

var part170 = match("MESSAGE#291:Completed:01", "nwparser.payload", "Completed %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg194 = msg("Completed:01", part170);

var part171 = match("MESSAGE#344:Completed:02", "nwparser.payload", "Scan ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] Completed computation of asset group synopses.", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Completed computation of asset group synopses"),
]));

var msg195 = msg("Completed:02", part171);

var part172 = match("MESSAGE#345:Completed:03", "nwparser.payload", "Scan ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] Completed computation of site synopsis.", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Completed computation of site synopsis"),
]));

var msg196 = msg("Completed:03", part172);

var part173 = match("MESSAGE#346:Completed:04", "nwparser.payload", "Scan ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] Completed recomputation of synopsis data.", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Completed recomputation of synopsis data"),
]));

var msg197 = msg("Completed:04", part173);

var part174 = match("MESSAGE#347:Completed:05", "nwparser.payload", "Scan ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] %{event_description}", processor_chain([
	dup18,
	dup12,
	dup13,
	dup43,
	dup14,
	dup15,
]));

var msg198 = msg("Completed:05", part174);

var part175 = match("MESSAGE#348:Completed:06", "nwparser.payload", "Started: %{fld2}T%{fld3}] [Duration: %{fld4}] %{event_description}", processor_chain([
	dup18,
	dup12,
	dup13,
	dup43,
	dup14,
	dup15,
]));

var msg199 = msg("Completed:06", part175);

var part176 = match("MESSAGE#460:Completed:07", "nwparser.payload", "%{fld1}] [%{fld2}] [%{fld3}] [%{fld4}] [Started: %{fld5}T%{fld6}] [Duration: %{fld7}] Completed purging sub-scan results.", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Completed purging sub-scan results"),
]));

var msg200 = msg("Completed:07", part176);

var part177 = match("MESSAGE#461:Completed:08", "nwparser.payload", "SiteID: %{fld1}] [Scan ID: %{fld2}] [Started: %{fld3}T%{fld4}] [Duration: %{fld5}] Completed computation of synopsis.", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Completed computation of synopsis"),
]));

var msg201 = msg("Completed:08", part177);

var select40 = linear_select([
	msg193,
	msg194,
	msg195,
	msg196,
	msg197,
	msg198,
	msg199,
	msg200,
	msg201,
]);

var part178 = match("MESSAGE#166:Retrieved", "nwparser.payload", "Retrieved XML version %{version->} for file %{filename}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg202 = msg("Retrieved", part178);

var part179 = match("MESSAGE#167:CIFS", "nwparser.payload", "CIFS Name Service name: %{service}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg203 = msg("CIFS", part179);

var msg204 = msg("Cached:", dup64);

var msg205 = msg("Enumerating", dup64);

var part180 = match("MESSAGE#170:Checking:01", "nwparser.payload", "Checking for approved updates.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg206 = msg("Checking:01", part180);

var msg207 = msg("Checking:02", dup64);

var select41 = linear_select([
	msg206,
	msg207,
]);

var part181 = match("MESSAGE#172:CSIDL_SYSTEMX86", "nwparser.payload", "CSIDL_SYSTEMX86 dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg208 = msg("CSIDL_SYSTEMX86", part181);

var part182 = match("MESSAGE#173:CSIDL_SYSTEM", "nwparser.payload", "CSIDL_SYSTEM dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg209 = msg("CSIDL_SYSTEM", part182);

var part183 = match("MESSAGE#174:office", "nwparser.payload", "office root dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg210 = msg("office", part183);

var part184 = match("MESSAGE#175:Exchange", "nwparser.payload", "Exchange root dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg211 = msg("Exchange", part184);

var part185 = match("MESSAGE#176:SQL", "nwparser.payload", "SQL Server root dir is: '%{directory}'", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg212 = msg("SQL", part185);

var part186 = match("MESSAGE#177:starting", "nwparser.payload", "starting %{service}", processor_chain([
	dup20,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg213 = msg("starting", part186);

var part187 = match("MESSAGE#178:Host", "nwparser.payload", "Host type (from MAC %{smacaddr}): %{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg214 = msg("Host", part187);

var part188 = match("MESSAGE#268:Host:01", "nwparser.payload", "Host Address: %{saddr}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg215 = msg("Host:01", part188);

var part189 = match("MESSAGE#269:Host:02", "nwparser.payload", "Host FQDN: %{fqdn}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg216 = msg("Host:02", part189);

var select42 = linear_select([
	msg214,
	msg215,
	msg216,
]);

var part190 = match("MESSAGE#179:Advertising", "nwparser.payload", "Advertising %{service->} service", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg217 = msg("Advertising", part190);

var part191 = match("MESSAGE#180:IP", "nwparser.payload", "IP fingerprint:%{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg218 = msg("IP", part191);

var part192 = match("MESSAGE#181:Updating:01", "nwparser.payload", "Updating file, %{filename}.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg219 = msg("Updating:01", part192);

var part193 = match("MESSAGE#182:Updating", "nwparser.payload", "Updating %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg220 = msg("Updating", part193);

var select43 = linear_select([
	msg219,
	msg220,
]);

var part194 = match("MESSAGE#183:Updated", "nwparser.payload", "Updated risk scores for %{dclass_counter1->} vulnerabilities in %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","Number of vulnerabilities"),
]));

var msg221 = msg("Updated", part194);

var part195 = match("MESSAGE#184:Updated:01", "nwparser.payload", "Updated risk scores for %{dclass_counter1->} assets in %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","Number of assets"),
]));

var msg222 = msg("Updated:01", part195);

var part196 = match("MESSAGE#185:Updated:02", "nwparser.payload", "Updated risk scores for %{dclass_counter1->} sites in %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","Number of sites"),
]));

var msg223 = msg("Updated:02", part196);

var part197 = match("MESSAGE#186:Updated:03", "nwparser.payload", "Updated risk scores for %{dclass_counter1->} groups in %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","Number of groups"),
]));

var msg224 = msg("Updated:03", part197);

var part198 = match("MESSAGE#260:Updated:04/0", "nwparser.payload", "Started: %{fld2}] [Duration: %{fld3}] Updated risk scores for %{fld1->} %{p0}");

var part199 = match("MESSAGE#260:Updated:04/1_0", "nwparser.p0", "vulnerabilities.%{}");

var part200 = match("MESSAGE#260:Updated:04/1_1", "nwparser.p0", "assets.%{}");

var part201 = match("MESSAGE#260:Updated:04/1_2", "nwparser.p0", "sites.%{}");

var part202 = match("MESSAGE#260:Updated:04/1_3", "nwparser.p0", "groups.%{}");

var select44 = linear_select([
	part199,
	part200,
	part201,
	part202,
]);

var all16 = all_match({
	processors: [
		part198,
		select44,
	],
	on_success: processor_chain([
		dup20,
		dup15,
	]),
});

var msg225 = msg("Updated:04", all16);

var part203 = match("MESSAGE#311:Updated:06/0", "nwparser.payload", "%{fld1}] [Started: %{fld2}] [Duration: %{fld3}] Updated %{p0}");

var part204 = match("MESSAGE#311:Updated:06/1_0", "nwparser.p0", "scan risk scores%{p0}");

var part205 = match("MESSAGE#311:Updated:06/1_1", "nwparser.p0", "risk scores for site%{p0}");

var select45 = linear_select([
	part204,
	part205,
]);

var part206 = match("MESSAGE#311:Updated:06/2", "nwparser.p0", ".%{}");

var all17 = all_match({
	processors: [
		part203,
		select45,
		part206,
	],
	on_success: processor_chain([
		dup11,
		dup14,
		dup15,
		setc("event_description","Updated risk scores"),
	]),
});

var msg226 = msg("Updated:06", all17);

var msg227 = msg("Updated:05", dup65);

var select46 = linear_select([
	msg221,
	msg222,
	msg223,
	msg224,
	msg225,
	msg226,
	msg227,
]);

var part207 = match("MESSAGE#187:Started", "nwparser.payload", "Started auto-update.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg228 = msg("Started", part207);

var msg229 = msg("Started:02", dup61);

var select47 = linear_select([
	msg228,
	msg229,
]);

var part208 = match("MESSAGE#188:Executing", "nwparser.payload", "Executing job JobID[%{info}] Risk and daily history updater for silo %{fld12}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg230 = msg("Executing", part208);

var part209 = match("MESSAGE#189:Executing:01", "nwparser.payload", "Executing job JobID[%{info}] Auto-update retriever", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg231 = msg("Executing:01", part209);

var part210 = match("MESSAGE#190:Executing:02", "nwparser.payload", "Executing job JobID[%{info}] %{fld1->} retention updater-default", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg232 = msg("Executing:02", part210);

var part211 = match("MESSAGE#191:Executing:04", "nwparser.payload", "Executing job JobID[%{info}] %{obj_type}: %{obj_name}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg233 = msg("Executing:04", part211);

var part212 = match("MESSAGE#326:Executing:03", "nwparser.payload", "Executing SQL: %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg234 = msg("Executing:03", part212);

var select48 = linear_select([
	msg230,
	msg231,
	msg232,
	msg233,
	msg234,
]);

var part213 = match("MESSAGE#192:A", "nwparser.payload", "A set of SSH administrative credentials have failed verification.%{}", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg235 = msg("A", part213);

var part214 = match("MESSAGE#193:Administrative:01", "nwparser.payload", "Administrative credentials failed (access denied).%{}", processor_chain([
	dup48,
	dup45,
	dup46,
	dup47,
	dup27,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg236 = msg("Administrative:01", part214);

var part215 = match("MESSAGE#194:Administrative", "nwparser.payload", "Administrative credentials for %{service->} will be used.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg237 = msg("Administrative", part215);

var select49 = linear_select([
	msg236,
	msg237,
]);

var part216 = match("MESSAGE#195:Initializing:01", "nwparser.payload", "Engine: %{fld1}] [Engine ID: %{fld2}] Initializing remote scan engine (%{dhost}).", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Initializing remote scan engine"),
]));

var msg238 = msg("Initializing:01", part216);

var part217 = match("MESSAGE#196:Initializing/1_0", "nwparser.p0", "Initializing %{service}.");

var part218 = match("MESSAGE#196:Initializing/1_1", "nwparser.p0", "Initializing JDBC drivers %{}");

var part219 = match("MESSAGE#196:Initializing/1_2", "nwparser.p0", "%{event_description}");

var select50 = linear_select([
	part217,
	part218,
	part219,
]);

var all18 = all_match({
	processors: [
		dup28,
		select50,
	],
	on_success: processor_chain([
		dup20,
		dup14,
		dup15,
		dup16,
		dup29,
		dup30,
		dup31,
		dup32,
		dup33,
		dup34,
	]),
});

var msg239 = msg("Initializing", all18);

var select51 = linear_select([
	msg238,
	msg239,
]);

var msg240 = msg("Creating", dup64);

var msg241 = msg("Loading", dup64);

var part220 = match("MESSAGE#199:Loaded", "nwparser.payload", "Loaded %{dclass_counter1->} policy checks for scan.", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","No. of policies"),
]));

var msg242 = msg("Loaded", part220);

var msg243 = msg("Loaded:01", dup66);

var select52 = linear_select([
	msg242,
	msg243,
]);

var part221 = match("MESSAGE#200:Finished", "nwparser.payload", "Finished locating %{dclass_counter1->} live nodes. [Started: %{fld11}] [Duration: %{fld12}]", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","No. of live nodes"),
]));

var msg244 = msg("Finished", part221);

var part222 = match("MESSAGE#201:Finished:01", "nwparser.payload", "Finished loading %{service}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg245 = msg("Finished:01", part222);

var part223 = match("MESSAGE#202:Finished:02", "nwparser.payload", "Finished resolving DNS records%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg246 = msg("Finished:02", part223);

var msg247 = msg("Finished:03", dup67);

var select53 = linear_select([
	msg244,
	msg245,
	msg246,
	msg247,
]);

var msg248 = msg("CheckProcessor:", dup64);

var msg249 = msg("Locating", dup64);

var part224 = match("MESSAGE#205:TCP", "nwparser.payload", "TCP port scanner is using: %{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg250 = msg("TCP", part224);

var part225 = match("MESSAGE#206:UDP", "nwparser.payload", "UDP port scanner is using: %{fld11}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg251 = msg("UDP", part225);

var part226 = match("MESSAGE#207:Queued", "nwparser.payload", "Queued live nodes for scanning: %{dclass_counter1}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
	setc("dclass_counter1_string","Live nodes"),
]));

var msg252 = msg("Queued", part226);

var msg253 = msg("Reading", dup64);

var msg254 = msg("Registering", dup64);

var part227 = match("MESSAGE#210:Registered", "nwparser.payload", "Registered session [%{fld12}] for IP [%{saddr}]", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg255 = msg("Registered", part227);

var part228 = match("MESSAGE#219:Registered:02", "nwparser.payload", "Registered session for principal name [%{username}] for IP [%{saddr}]", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg256 = msg("Registered:02", part228);

var select54 = linear_select([
	msg255,
	msg256,
]);

var part229 = match("MESSAGE#211:Seeing", "nwparser.payload", "Seeing if %{saddr->} is a valid network node", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var msg257 = msg("Seeing", part229);

var part230 = match("MESSAGE#212:Logging", "nwparser.payload", "Logging initialized. [Name = %{obj_name}] [Level = %{fld11}] [Timezone = %{fld12}]", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
]));

var msg258 = msg("Logging", part230);

var msg259 = msg("Firefox", dup64);

var msg260 = msg("nodes", dup64);

var msg261 = msg("common", dup67);

var msg262 = msg("jess.JessException:", dup67);

var part231 = match("MESSAGE#218:Successfully", "nwparser.payload", "Successfully %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg263 = msg("Successfully", part231);

var msg264 = msg("Establishing", dup61);

var msg265 = msg("Response", dup61);

var msg266 = msg("Auto-update", dup61);

var msg267 = msg("Approved:03", dup61);

var msg268 = msg("HHH000436:", dup61);

var msg269 = msg("Staged", dup61);

var msg270 = msg("Refreshing", dup61);

var msg271 = msg("Activation", dup61);

var msg272 = msg("Acknowledging", dup61);

var msg273 = msg("Acknowledged", dup61);

var msg274 = msg("Validating", dup61);

var msg275 = msg("Patching", dup61);

var msg276 = msg("JAR", dup61);

var msg277 = msg("Destroying", dup61);

var msg278 = msg("Invocation", dup61);

var msg279 = msg("Using", dup61);

var part232 = match("MESSAGE#243:Route:01", "nwparser.payload", "Route: %{fld1->} shutdown complete, %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg280 = msg("Route:01", part232);

var part233 = match("MESSAGE#244:Route:02", "nwparser.payload", "Route: %{fld1->} started and consuming from: %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg281 = msg("Route:02", part233);

var select55 = linear_select([
	msg280,
	msg281,
]);

var msg282 = msg("Deploying", dup61);

var msg283 = msg("Generating", dup61);

var msg284 = msg("Staging", dup61);

var msg285 = msg("Removing", dup61);

var msg286 = msg("At", dup61);

var msg287 = msg("An", dup61);

var msg288 = msg("The", dup61);

var msg289 = msg("Downloading", dup61);

var msg290 = msg("Downloaded", dup61);

var msg291 = msg("Restarting", dup61);

var msg292 = msg("Requested", dup61);

var part234 = match("MESSAGE#257:Freeing", "nwparser.payload", "Freeing session for principal name [%{username}]", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg293 = msg("Freeing", part234);

var part235 = match("MESSAGE#258:Freeing:01", "nwparser.payload", "Freeing %{dclass_counter1->} current sessions.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg294 = msg("Freeing:01", part235);

var select56 = linear_select([
	msg293,
	msg294,
]);

var part236 = match("MESSAGE#259:Kill", "nwparser.payload", "Kill session for principal name [%{username}]", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg295 = msg("Kill", part236);

var part237 = match("MESSAGE#262:Created:01", "nwparser.payload", "Created temporary directory %{filename}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg296 = msg("Created:01", part237);

var part238 = match("MESSAGE#331:Created:02", "nwparser.payload", "Created %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg297 = msg("Created:02", part238);

var select57 = linear_select([
	msg296,
	msg297,
]);

var part239 = match("MESSAGE#263:Product", "nwparser.payload", "Product Version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg298 = msg("Product", part239);

var part240 = match("MESSAGE#264:Current", "nwparser.payload", "Current directory: %{filename}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg299 = msg("Current", part240);

var part241 = match("MESSAGE#308:Current:01", "nwparser.payload", "Current DB_VERSION = %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg300 = msg("Current:01", part241);

var select58 = linear_select([
	msg299,
	msg300,
]);

var part242 = match("MESSAGE#266:Super", "nwparser.payload", "Super user: %{result}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg301 = msg("Super", part242);

var part243 = match("MESSAGE#267:Computer", "nwparser.payload", "Computer name: %{hostname}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg302 = msg("Computer", part243);

var part244 = match("MESSAGE#270:Operating", "nwparser.payload", "Operating system: %{os}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg303 = msg("Operating", part244);

var part245 = match("MESSAGE#271:CPU", "nwparser.payload", "CPU speed: %{fld1}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg304 = msg("CPU", part245);

var part246 = match("MESSAGE#272:Number", "nwparser.payload", "Number of CPUs: %{dclass_counter1}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg305 = msg("Number", part246);

var part247 = match("MESSAGE#273:Total", "nwparser.payload", "Total %{fld1}: %{fld2}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg306 = msg("Total", part247);

var part248 = match("MESSAGE#320:Total:02", "nwparser.payload", "Total %{dclass_counter1->} routes, of which %{dclass_counter2->} is started.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg307 = msg("Total:02", part248);

var select59 = linear_select([
	msg306,
	msg307,
]);

var part249 = match("MESSAGE#274:Available", "nwparser.payload", "Available %{fld1}: %{fld2}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg308 = msg("Available", part249);

var part250 = match("MESSAGE#275:Disk", "nwparser.payload", "Disk space used by %{fld1}: %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg309 = msg("Disk", part250);

var part251 = match("MESSAGE#276:JVM", "nwparser.payload", "JVM %{fld1}: %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg310 = msg("JVM", part251);

var part252 = match("MESSAGE#277:Pausing", "nwparser.payload", "Pausing ProtocolHandler [%{info}]", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg311 = msg("Pausing", part252);

var part253 = match("MESSAGE#278:Policy", "nwparser.payload", "Policy %{policyname->} replaces %{fld1}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg312 = msg("Policy", part253);

var part254 = match("MESSAGE#420:Policy:01", "nwparser.payload", "Policy benchmark %{policyname->} in %{info->} with hash %{fld1->} is not valid builtin content and will not load.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Policy benchmark is not valid builtin content and will not load"),
]));

var msg313 = msg("Policy:01", part254);

var select60 = linear_select([
	msg312,
	msg313,
]);

var part255 = match("MESSAGE#279:Bulk", "nwparser.payload", "Bulk %{action->} %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg314 = msg("Bulk", part255);

var part256 = match("MESSAGE#280:Importing", "nwparser.payload", "%{action->} %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg315 = msg("Importing", part256);

var part257 = match("MESSAGE#281:Imported", "nwparser.payload", "%{action->} %{dclass_counter1->} new categories, categorized %{fld1->} vulnerabilities and %{fld2->} tags.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg316 = msg("Imported", part257);

var msg317 = msg("Imported:01", dup65);

var select61 = linear_select([
	msg316,
	msg317,
]);

var part258 = match("MESSAGE#283:Compiling", "nwparser.payload", "Compiling %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg318 = msg("Compiling", part258);

var part259 = match("MESSAGE#284:Vulnerability", "nwparser.payload", "Vulnerability %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg319 = msg("Vulnerability", part259);

var part260 = match("MESSAGE#285:Truncating", "nwparser.payload", "Truncating %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg320 = msg("Truncating", part260);

var part261 = match("MESSAGE#286:Synchronizing", "nwparser.payload", "Synchronizing %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg321 = msg("Synchronizing", part261);

var part262 = match("MESSAGE#287:Parsing", "nwparser.payload", "Parsing %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg322 = msg("Parsing", part262);

var part263 = match("MESSAGE#288:Remapping", "nwparser.payload", "Remapping %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg323 = msg("Remapping", part263);

var part264 = match("MESSAGE#289:Remapped", "nwparser.payload", "Started: %{fld1}] [Duration: %{fld2}] Remapped %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg324 = msg("Remapped", part264);

var part265 = match("MESSAGE#290:Database", "nwparser.payload", "Started: %{fld1}] [Duration: %{fld2}] Database %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg325 = msg("Database", part265);

var part266 = match("MESSAGE#428:Database:01", "nwparser.payload", "Database %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg326 = msg("Database:01", part266);

var select62 = linear_select([
	msg325,
	msg326,
]);

var part267 = match("MESSAGE#292:Accepting", "nwparser.payload", "Accepting %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg327 = msg("Accepting", part267);

var part268 = match("MESSAGE#293:VERSION:03", "nwparser.payload", "VERSION %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg328 = msg("VERSION:03", part268);

var part269 = match("MESSAGE#294:Detected", "nwparser.payload", "Detected %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg329 = msg("Detected", part269);

var part270 = match("MESSAGE#295:Telling", "nwparser.payload", "Telling %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg330 = msg("Telling", part270);

var part271 = match("MESSAGE#298:Stopping", "nwparser.payload", "Stopping %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg331 = msg("Stopping", part271);

var part272 = match("MESSAGE#299:removing", "nwparser.payload", "removing %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg332 = msg("removing", part272);

var part273 = match("MESSAGE#300:Enabling", "nwparser.payload", "Enabling %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg333 = msg("Enabling", part273);

var part274 = match("MESSAGE#301:Granting", "nwparser.payload", "Granting %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg334 = msg("Granting", part274);

var part275 = match("MESSAGE#302:Version", "nwparser.payload", "Version %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg335 = msg("Version", part275);

var part276 = match("MESSAGE#303:Configuring", "nwparser.payload", "Configuring %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg336 = msg("Configuring", part276);

var part277 = match("MESSAGE#305:Scheduler", "nwparser.payload", "Scheduler %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg337 = msg("Scheduler", part277);

var part278 = match("MESSAGE#341:Scheduler:01", "nwparser.payload", "Silo: %{fld1}] [Started: %{fld2}] [Duration: %{fld3}] Scheduler started.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Scheduler started"),
]));

var msg338 = msg("Scheduler:01", part278);

var part279 = match("MESSAGE#429:Scheduler:02", "nwparser.payload", "%{fld1}: %{fld2}] Scheduler %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg339 = msg("Scheduler:02", part279);

var select63 = linear_select([
	msg337,
	msg338,
	msg339,
]);

var part280 = match("MESSAGE#306:PostgreSQL", "nwparser.payload", "PostgreSQL %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg340 = msg("PostgreSQL", part280);

var part281 = match("MESSAGE#307:Cleaning", "nwparser.payload", "Cleaning %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg341 = msg("Cleaning", part281);

var part282 = match("MESSAGE#462:Cleaning:01", "nwparser.payload", "%{fld1}] [%{fld2}] [%{fld3}] [%{fld4}] Cleaning up sub-scan results.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Cleaning up sub-scan results"),
]));

var msg342 = msg("Cleaning:01", part282);

var select64 = linear_select([
	msg341,
	msg342,
]);

var part283 = match("MESSAGE#309:Installed:01/0", "nwparser.payload", "Installed DB%{p0}");

var part284 = match("MESSAGE#309:Installed:01/1_0", "nwparser.p0", "_VERSION after upgrade%{p0}");

var part285 = match("MESSAGE#309:Installed:01/1_1", "nwparser.p0", " VERSION %{p0}");

var select65 = linear_select([
	part284,
	part285,
]);

var part286 = match("MESSAGE#309:Installed:01/2", "nwparser.p0", "%{}= %{version}");

var all19 = all_match({
	processors: [
		part283,
		select65,
		part286,
	],
	on_success: processor_chain([
		dup20,
		dup14,
		dup15,
	]),
});

var msg343 = msg("Installed:01", all19);

var part287 = match("MESSAGE#310:Inserted", "nwparser.payload", "Inserted %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg344 = msg("Inserted", part287);

var part288 = match("MESSAGE#313:Deleted", "nwparser.payload", "Started: %{fld1}] [Duration: %{fld2}] Deleted %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg345 = msg("Deleted", part288);

var msg346 = msg("Default", dup66);

var msg347 = msg("Apache", dup66);

var msg348 = msg("JMX", dup66);

var msg349 = msg("AllowUseOriginalMessage", dup66);

var part289 = match("MESSAGE#321:Initialized", "nwparser.payload", "Initialized PolicyCheckService with %{dclass_counter1->} benchmarks, containing %{fld1->} policies. The total check count is %{dclass_counter2}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg350 = msg("Initialized", part289);

var part290 = match("MESSAGE#322:Initialized:01", "nwparser.payload", "Initialized %{dclass_counter1->} policy benchmarks in total.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg351 = msg("Initialized:01", part290);

var part291 = match("MESSAGE#379:Initialized_Scheduler", "nwparser.payload", "Initialized Scheduler Signaller of type: %{obj_type->} %{obj_name}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Initialized Scheduler Signaller"),
]));

var msg352 = msg("Initialized_Scheduler", part291);

var select66 = linear_select([
	msg350,
	msg351,
	msg352,
]);

var msg353 = msg("Error", dup66);

var part292 = match("MESSAGE#324:Graceful", "nwparser.payload", "Graceful shutdown of %{dclass_counter1->} routes completed in %{dclass_counter2->} seconds", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg354 = msg("Graceful", part292);

var msg355 = msg("StreamCaching", dup61);

var msg356 = msg("Local", dup66);

var part293 = match("MESSAGE#329:DB_VERSION", "nwparser.payload", "DB_VERSION = %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg357 = msg("DB_VERSION", part293);

var part294 = match("MESSAGE#330:Populating", "nwparser.payload", "Populating %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg358 = msg("Populating", part294);

var part295 = match("MESSAGE#332:EventLog", "nwparser.payload", "EventLog %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg359 = msg("EventLog", part295);

var part296 = match("MESSAGE#333:Making", "nwparser.payload", "Making %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg360 = msg("Making", part296);

var part297 = match("MESSAGE#334:Setting", "nwparser.payload", "Setting %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg361 = msg("Setting", part297);

var part298 = match("MESSAGE#335:initdb", "nwparser.payload", "initdb %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg362 = msg("initdb", part298);

var part299 = match("MESSAGE#336:Verifying", "nwparser.payload", "Verifying %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg363 = msg("Verifying", part299);

var msg364 = msg("OS", dup66);

var part300 = match("MESSAGE#338:Benchmark", "nwparser.payload", "Benchmark %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg365 = msg("Benchmark", part300);

var part301 = match("MESSAGE#339:Report:01", "nwparser.payload", "Report Config ID: %{fld1}] [Started: %{fld2}T%{fld3}] [Duration: %{fld4}] %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup29,
	dup54,
	dup16,
]));

var msg366 = msg("Report:01", part301);

var part302 = match("MESSAGE#340:Report", "nwparser.payload", "Report Config ID: %{fld1}] %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup29,
	dup54,
	dup16,
]));

var msg367 = msg("Report", part302);

var select67 = linear_select([
	msg366,
	msg367,
]);

var part303 = match("MESSAGE#342:Cannot_preload", "nwparser.payload", "Engine ID: %{fld1}] [Engine Name: %{fld2}] Cannot preload incremental pool with a connection %{fld3}", processor_chain([
	dup53,
	dup14,
	dup15,
	dup55,
]));

var msg368 = msg("Cannot_preload", part303);

var part304 = match("MESSAGE#343:Cannot_preload:01", "nwparser.payload", "Cannot preload incremental pool with a connection%{fld3}", processor_chain([
	dup53,
	dup14,
	dup15,
	dup55,
]));

var msg369 = msg("Cannot_preload:01", part304);

var select68 = linear_select([
	msg368,
	msg369,
]);

var part305 = match("MESSAGE#349:ERROR:02", "nwparser.payload", "ERROR: syntax error at or near \"%{fld1}\"", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","Syntax error"),
]));

var msg370 = msg("ERROR:02", part305);

var part306 = match("MESSAGE#350:QuartzRepeaterBuilder", "nwparser.payload", "QuartzRepeaterBuilder failed to add schedule to ScanConfig: null%{}", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","QuartzRepeaterBuilder failed to add schedule"),
]));

var msg371 = msg("QuartzRepeaterBuilder", part306);

var part307 = match("MESSAGE#351:Backing_up", "nwparser.payload", "Backing up %{event_source}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Backing up"),
]));

var msg372 = msg("Backing_up", part307);

var part308 = match("MESSAGE#352:Not_configured", "nwparser.payload", "com.rapid.nexpose.scanpool.stateInterval is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid.nexpose.scanpool.stateInterval is not configured"),
]));

var msg373 = msg("Not_configured", part308);

var part309 = match("MESSAGE#353:Not_configured:01", "nwparser.payload", "com.rapid7.nexpose.comms.clientConnectionProvider.autoPairTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.comms.clientConnectionProvider.autoPairTimeout is not configured"),
]));

var msg374 = msg("Not_configured:01", part309);

var part310 = match("MESSAGE#354:Not_configured:02", "nwparser.payload", "com.rapid7.nexpose.comms.clientConnectionProvider.getConnectionTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.comms.clientConnectionProvider.getConnectionTimeout is not configured"),
]));

var msg375 = msg("Not_configured:02", part310);

var part311 = match("MESSAGE#355:Not_configured:03", "nwparser.payload", "com.rapid7.nexpose.datastore.connection.evictionThreadTime is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.datastore.connection.evictionThreadTime is not configured"),
]));

var msg376 = msg("Not_configured:03", part311);

var part312 = match("MESSAGE#356:Not_configured:04", "nwparser.payload", "com.rapid7.nexpose.datastore.eviction.connection.threadIdleTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.datastore.eviction.connection.threadIdleTimeout is not configured"),
]));

var msg377 = msg("Not_configured:04", part312);

var part313 = match("MESSAGE#357:Not_configured:05", "nwparser.payload", "com.rapid7.nexpose.nsc.dbcc is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.dbcc is not configured"),
]));

var msg378 = msg("Not_configured:05", part313);

var part314 = match("MESSAGE#358:Not_configured:06", "nwparser.payload", "com.rapid7.nexpose.nsc.scanExecutorService.maximumCorePoolSize is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scanExecutorService.maximumCorePoolSize is not configured"),
]));

var msg379 = msg("Not_configured:06", part314);

var part315 = match("MESSAGE#359:Not_configured:07", "nwparser.payload", "com.rapid7.nexpose.nsc.scanExecutorService.minimumCorePoolSize is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scanExecutorService.minimumCorePoolSize is not configured"),
]));

var msg380 = msg("Not_configured:07", part315);

var part316 = match("MESSAGE#360:Not_configured:08", "nwparser.payload", "com.rapid7.nexpose.nsc.scanExecutorService.monitorCorePoolSizeIncreaseOnSaturation is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scanExecutorService.monitorCorePoolSizeIncreaseOnSaturation is not configured"),
]));

var msg381 = msg("Not_configured:08", part316);

var part317 = match("MESSAGE#361:Not_configured:09", "nwparser.payload", "com.rapid7.nexpose.nsc.scanExecutorService.monitorEnabled is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scanExecutorService.monitorEnabled is not configured"),
]));

var msg382 = msg("Not_configured:09", part317);

var part318 = match("MESSAGE#362:Not_configured:10", "nwparser.payload", "com.rapid7.nexpose.nsc.scanExecutorService.monitorInterval is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scanExecutorService.monitorInterval is not configured"),
]));

var msg383 = msg("Not_configured:10", part318);

var part319 = match("MESSAGE#363:Not_configured:11", "nwparser.payload", "com.rapid7.nexpose.nse.nscClient.connectTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nse.nscClient.connectTimeout is not configured"),
]));

var msg384 = msg("Not_configured:11", part319);

var part320 = match("MESSAGE#364:Not_configured:12", "nwparser.payload", "com.rapid7.nexpose.nse.nscClient.readTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nse.nscClient.readTimeout is not configured"),
]));

var msg385 = msg("Not_configured:12", part320);

var part321 = match("MESSAGE#365:Not_configured:13", "nwparser.payload", "com.rapid7.nexpose.reportGenerator.assetCollectionUpdateTimeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.reportGenerator.assetCollectionUpdateTimeout is not configured"),
]));

var msg386 = msg("Not_configured:13", part321);

var part322 = match("MESSAGE#366:Not_configured:14", "nwparser.payload", "com.rapid7.nexpose.scan.consolidation.delay is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.consolidation.delay is not configured"),
]));

var msg387 = msg("Not_configured:14", part322);

var part323 = match("MESSAGE#367:Not_configured:15", "nwparser.payload", "com.rapid7.nexpose.scan.lifecyclemonitor.delay is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.lifecyclemonitor.delay is not configured"),
]));

var msg388 = msg("Not_configured:15", part323);

var part324 = match("MESSAGE#368:Not_configured:16", "nwparser.payload", "com.rapid7.nexpose.scan.usescanpool is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.usescanpool is not configured"),
]));

var msg389 = msg("Not_configured:16", part324);

var part325 = match("MESSAGE#369:Not_configured:17", "nwparser.payload", "com.rapid7.nsc.workflow.timeout is not configured - returning default value %{resultcode}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nsc.workflow.timeout is not configured"),
]));

var msg390 = msg("Not_configured:17", part325);

var part326 = match("MESSAGE#370:Delivered", "nwparser.payload", "Delivered mail to %{to}: %{fld1->} %{fld2->} %{mail_id->} [InternalId=%{fld3}] Queued mail for delivery", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("action","Queued mail for delivery"),
]));

var msg391 = msg("Delivered", part326);

var part327 = match("MESSAGE#371:Engine_update", "nwparser.payload", "Engine update thread pool shutting down.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Engine update thread pool shutting down"),
]));

var msg392 = msg("Engine_update", part327);

var part328 = match("MESSAGE#372:Freed_triggers", "nwparser.payload", "Freed %{fld1->} triggers from 'acquired' / 'blocked' state.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Freed triggers from 'acquired' / 'blocked' state"),
]));

var msg393 = msg("Freed_triggers", part328);

var part329 = match("MESSAGE#374:Upgrade_completed", "nwparser.payload", "PG Upgrade has completed succesfully%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Upgrade has completed succesfully"),
]));

var msg394 = msg("Upgrade_completed", part329);

var part330 = match("MESSAGE#375:PG", "nwparser.payload", "%{fld1}: %{process->} %{param}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg395 = msg("PG", part330);

var select69 = linear_select([
	msg394,
	msg395,
]);

var part331 = match("MESSAGE#376:DEFAULT_SCHEDULER", "nwparser.payload", "DEFAULT SCHEDULER: %{obj_name}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","DEFAULT SCHEDULER"),
]));

var msg396 = msg("DEFAULT_SCHEDULER", part331);

var part332 = match("MESSAGE#377:Context_loader", "nwparser.payload", "Context loader config file is jar:file:%{filename}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Context loader config file"),
]));

var msg397 = msg("Context_loader", part332);

var part333 = match("MESSAGE#378:Copied_file", "nwparser.payload", "Copied %{filename->} file from %{directory->} to %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Copied file"),
]));

var msg398 = msg("Copied_file", part333);

var part334 = match("MESSAGE#380:Java", "nwparser.payload", "Java HotSpot(TM) %{info}", processor_chain([
	dup20,
	dup15,
	setc("event_description","Console VM version"),
]));

var msg399 = msg("Java", part334);

var part335 = match("MESSAGE#381:Changing", "nwparser.payload", "Changing permissions of %{obj_type->} '%{obj_name}' to %{change_new}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Changing permissions"),
]));

var msg400 = msg("Changing", part335);

var part336 = match("MESSAGE#382:Changing:01", "nwparser.payload", "Changing the new database AUTH method to %{change_new}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Changing new database AUTH method"),
]));

var msg401 = msg("Changing:01", part336);

var select70 = linear_select([
	msg400,
	msg401,
]);

var part337 = match("MESSAGE#383:Job_execution", "nwparser.payload", "Job execution threads will use class loader of thread: %{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Job execution threads will use class loader"),
]));

var msg402 = msg("Job_execution", part337);

var part338 = match("MESSAGE#384:Initialized:02", "nwparser.payload", "JobStoreCMT initialized.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","JobStoreCMT initialized"),
]));

var msg403 = msg("Initialized:02", part338);

var part339 = match("MESSAGE#385:Initialized:03", "nwparser.payload", "Quartz scheduler '%{obj_name}' %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Quartz scheduler initialized"),
]));

var msg404 = msg("Initialized:03", part339);

var part340 = match("MESSAGE#386:Created:03", "nwparser.payload", "Quartz Scheduler %{version->} created.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Quartz Scheduler created."),
]));

var msg405 = msg("Created:03", part340);

var part341 = match("MESSAGE#387:Scheduler_version", "nwparser.payload", "Quartz scheduler version: %{version}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg406 = msg("Scheduler_version", part341);

var select71 = linear_select([
	msg404,
	msg405,
	msg406,
]);

var part342 = match("MESSAGE#388:Recovering", "nwparser.payload", "Recovering %{fld1->} %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Recovering jobs"),
]));

var msg407 = msg("Recovering", part342);

var part343 = match("MESSAGE#389:Recovery", "nwparser.payload", "Recovery complete.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Recovery"),
	setc("disposition","Complete"),
]));

var msg408 = msg("Recovery", part343);

var part344 = match("MESSAGE#390:Removed", "nwparser.payload", "Removed %{fld1->} 'complete' triggers.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Removed triggers"),
]));

var msg409 = msg("Removed", part344);

var part345 = match("MESSAGE#391:Removed:01", "nwparser.payload", "Removed %{fld1->} stale fired job entries.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Removed job entries"),
]));

var msg410 = msg("Removed:01", part345);

var select72 = linear_select([
	msg409,
	msg410,
]);

var part346 = match("MESSAGE#392:Restoring", "nwparser.payload", "%{action}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg411 = msg("Restoring", part346);

var part347 = match("MESSAGE#393:Upgrading", "nwparser.payload", "Upgrading database%{fld1}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Upgrading database"),
]));

var msg412 = msg("Upgrading", part347);

var part348 = match("MESSAGE#394:Exploits", "nwparser.payload", "Exploits are up to date.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Exploits are up to date"),
]));

var msg413 = msg("Exploits", part348);

var part349 = match("MESSAGE#395:Failure", "nwparser.payload", "Failure communicating with NSE @ %{dhost}:%{dport}.", processor_chain([
	dup53,
	dup49,
	dup27,
	dup14,
	dup15,
	setc("event_description","Failure communicating with NSE"),
]));

var msg414 = msg("Failure", part349);

var part350 = match("MESSAGE#396:Renamed", "nwparser.payload", "Renamed %{filename->} to %{info}", processor_chain([
	dup20,
	dup57,
	dup22,
	dup14,
	dup15,
]));

var msg415 = msg("Renamed", part350);

var part351 = match("MESSAGE#397:Reinitializing", "nwparser.payload", "Reinitializing web server for maintenance mode...%{}", processor_chain([
	dup20,
	dup57,
	dup22,
	dup14,
	dup15,
	setc("event_description","Reinitializing web server for maintenance mode"),
]));

var msg416 = msg("Reinitializing", part351);

var part352 = match("MESSAGE#398:Replaced", "nwparser.payload", "Replaced %{change_old->} values from %{filename->} file with new auth method: %{change_new}.", processor_chain([
	dup20,
	dup57,
	dup22,
	dup14,
	dup15,
	dup58,
]));

var msg417 = msg("Replaced", part352);

var part353 = match("MESSAGE#399:Replaced:01", "nwparser.payload", "Replaced %{change_old->} values from %{filename->} with new setting values", processor_chain([
	dup20,
	dup57,
	dup22,
	dup14,
	dup15,
	dup58,
]));

var msg418 = msg("Replaced:01", part353);

var select73 = linear_select([
	msg417,
	msg418,
]);

var part354 = match("MESSAGE#400:System", "nwparser.payload", "System is running low on memory: %{fld1}MB total (%{fld2}MB free)", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","System is running low on memory"),
]));

var msg419 = msg("System", part354);

var part355 = match("MESSAGE#401:System:01", "nwparser.payload", "%{info}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup30,
	dup31,
	dup32,
	dup33,
]));

var msg420 = msg("System:01", part355);

var select74 = linear_select([
	msg419,
	msg420,
]);

var part356 = match("MESSAGE#402:Analyzing", "nwparser.payload", "Analyzing the database.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Analyzing the database"),
]));

var msg421 = msg("Analyzing", part356);

var part357 = match("MESSAGE#403:Connection", "nwparser.payload", "Connection to the new database was successful. %{action}.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Connection to the new database was successful"),
]));

var msg422 = msg("Connection", part357);

var part358 = match("MESSAGE#404:Handling", "nwparser.payload", "Handling %{fld1->} trigger(s) that missed their scheduled fire-time.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Handling trigger(s) that missed their scheduled fire-time"),
]));

var msg423 = msg("Handling", part358);

var part359 = match("MESSAGE#406:LDAP", "nwparser.payload", "LDAP authentication requires resolution%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","LDAP authentication requires resolution"),
]));

var msg424 = msg("LDAP", part359);

var part360 = match("MESSAGE#407:Maintenance", "nwparser.payload", "Maintenance Task Started%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Maintenance Task Started"),
]));

var msg425 = msg("Maintenance", part360);

var msg426 = msg("Migration", dup61);

var msg427 = msg("Mobile", dup68);

var msg428 = msg("ConsoleScanImporter", dup68);

var part361 = match("MESSAGE#421:Postgres:01", "nwparser.payload", "%{event_description}. Cleaning up. %{directory}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Cleaning up"),
]));

var msg429 = msg("Postgres:01", part361);

var part362 = match("MESSAGE#422:Succesfully", "nwparser.payload", "Succesfully %{event_description->} to %{dport}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg430 = msg("Succesfully", part362);

var part363 = match("MESSAGE#423:Unzipped", "nwparser.payload", "%{action->} %{fld1->} bytes into %{directory}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg431 = msg("Unzipped", part363);

var part364 = match("MESSAGE#424:vacuumdb", "nwparser.payload", "%{process->} executed with a return value of %{resultcode}.", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg432 = msg("vacuumdb", part364);

var part365 = match("MESSAGE#425:Processed_vuln", "nwparser.payload", "Started: %{fld2}T%{fld3}] [Duration: %{fld4}] Processed vuln check types for %{fld5->} vuln checks.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Processed vuln check types"),
]));

var msg433 = msg("Processed_vuln", part365);

var part366 = match("MESSAGE#430:Reflections", "nwparser.payload", "Reflections %{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var msg434 = msg("Reflections", part366);

var part367 = match("MESSAGE#431:CorrelationAttributes", "nwparser.payload", "0.16: %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg435 = msg("CorrelationAttributes", part367);

var part368 = match("MESSAGE#432:CorrelationAttributes:01", "nwparser.payload", "0.49: %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg436 = msg("CorrelationAttributes:01", part368);

var part369 = match("MESSAGE#433:CorrelationAttributes:02", "nwparser.payload", "0.245: %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg437 = msg("CorrelationAttributes:02", part369);

var part370 = match("MESSAGE#434:CorrelationAttributes:03", "nwparser.payload", "0.325: %{info}", processor_chain([
	dup20,
	dup15,
]));

var msg438 = msg("CorrelationAttributes:03", part370);

var msg439 = msg("ConsoleProductInfoProvider", dup69);

var msg440 = msg("NSXAssetEventHandler", dup69);

var msg441 = msg("ProductNotificationService", dup69);

var msg442 = msg("AssetEventHandler", dup69);

var msg443 = msg("SiteEventHandler", dup69);

var msg444 = msg("UserEventHandler", dup69);

var msg445 = msg("VulnerabilityExceptionEventHandler", dup69);

var msg446 = msg("TagEventHandler", dup69);

var msg447 = msg("AssetGroupEventHandler", dup69);

var msg448 = msg("ScanEventHandler", dup69);

var part371 = match("MESSAGE#445:Not_configured:18", "nwparser.payload", "com.rapid7.nexpose.nsc.critical.task.executor.core.thread.pool.size is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.critical.task.executor.core.thread.pool.size is not configured"),
]));

var msg449 = msg("Not_configured:18", part371);

var part372 = match("MESSAGE#446:Not_configured:19", "nwparser.payload", "com.rapid7.nexpose.nsc.scan.multiengine.scanHaltTimeoutMilliSecond is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scan.multiengine.scanHaltTimeoutMilliSecond is not configured"),
]));

var msg450 = msg("Not_configured:19", part372);

var part373 = match("MESSAGE#447:Not_configured:20", "nwparser.payload", "com.rapid7.nexpose.nsc.scan.scan.event.monitor.poll.duration is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.scan.scan.event.monitor.poll.duration is not configured"),
]));

var msg451 = msg("Not_configured:20", part373);

var part374 = match("MESSAGE#448:Not_configured:21", "nwparser.payload", "com.rapid7.nexpose.nse.excludedFileSystems is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nse.excludedFileSystems is not configured"),
]));

var msg452 = msg("Not_configured:21", part374);

var part375 = match("MESSAGE#449:Not_configured:22", "nwparser.payload", "com.rapid7.nexpose.scan.logCPUMemoryToMemLog.enable is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.logCPUMemoryToMemLog.enable is not configured"),
]));

var msg453 = msg("Not_configured:22", part375);

var part376 = match("MESSAGE#450:Not_configured:23", "nwparser.payload", "com.rapid7.nexpose.scan.logMemory.interval is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.logMemory.interval is not configured"),
]));

var msg454 = msg("Not_configured:23", part376);

var part377 = match("MESSAGE#451:Not_configured:24", "nwparser.payload", "com.rapid7.nexpose.scan.monitor.numberSavedAssetDurations is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.scan.monitor.numberSavedAssetDurations is not configured"),
]));

var msg455 = msg("Not_configured:24", part377);

var part378 = match("MESSAGE#452:Not_configured:25", "nwparser.payload", "com.rapid7.scan.perTestDurationLogging is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.scan.perTestDurationLogging is not configured"),
]));

var msg456 = msg("Not_configured:25", part378);

var part379 = match("MESSAGE#453:Not_configured:26", "nwparser.payload", "com.rapid7.thread.threadPoolNonBlockingOpsProviderParallelism is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.thread.threadPoolNonBlockingOpsProviderParallelism is not configured"),
]));

var msg457 = msg("Not_configured:26", part379);

var part380 = match("MESSAGE#454:Not_configured:27", "nwparser.payload", "com.rapid7.nexpose.nsc.critical.task.executor.max.thread.pool.size is not configured - returning default value %{result}.", processor_chain([
	dup56,
	dup14,
	dup15,
	setc("event_description","com.rapid7.nexpose.nsc.critical.task.executor.max.thread.pool.size is not configured"),
]));

var msg458 = msg("Not_configured:27", part380);

var part381 = match("MESSAGE#455:Spring", "nwparser.payload", "%{process->} detected on classpath: [%{fld2}]", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","detected"),
]));

var msg459 = msg("Spring", part381);

var part382 = match("MESSAGE#456:Storing", "nwparser.payload", "%{fld1}] [%{fld2}] Storing scan details for %{event_type}.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Storing scan details"),
]));

var msg460 = msg("Storing", part382);

var part383 = match("MESSAGE#457:Clearing", "nwparser.payload", "Clearing object tracker after %{dclass_counter1->} hits and %{dclass_counter2->} misses.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","Clearing object tracker"),
]));

var msg461 = msg("Clearing", part383);

var part384 = match("MESSAGE#458:All", "nwparser.payload", "%{fld1}] [%{fld2}] All scan engines are up to date.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("result","All scan engines are up to date"),
]));

var msg462 = msg("All", part384);

var part385 = match("MESSAGE#459:New", "nwparser.payload", "New Provider %{audit_object->} discovered.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("action","New Provider discovered"),
]));

var msg463 = msg("New", part385);

var part386 = match("MESSAGE#463:Session", "nwparser.payload", "%{fld1}] [%{fld2}] [%{fld3}] Session created.", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Session created"),
]));

var msg464 = msg("Session", part386);

var part387 = match("MESSAGE#464:Debug", "nwparser.payload", "Debug logging is not enabled for this scan.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Debug logging is not enabled"),
]));

var msg465 = msg("Debug", part387);

var msg466 = msg("Debug:01", dup61);

var select75 = linear_select([
	msg465,
	msg466,
]);

var part388 = match("MESSAGE#466:ACES", "nwparser.payload", "ACES logging is not enabled.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","ACES logging is not enabled"),
]));

var msg467 = msg("ACES", part388);

var msg468 = msg("ACES:01", dup61);

var select76 = linear_select([
	msg467,
	msg468,
]);

var part389 = match("MESSAGE#468:Invulnerable", "nwparser.payload", "Invulnerable Data Storage is on.%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Invulnerable Data Storage is on"),
]));

var msg469 = msg("Invulnerable", part389);

var part390 = match("MESSAGE#469:Nmap", "nwparser.payload", "Nmap ARP Ping for local networks%{}", processor_chain([
	dup20,
	dup14,
	dup15,
	setc("event_description","Nmap ARP Ping for local networks"),
]));

var msg470 = msg("Nmap", part390);

var part391 = match("MESSAGE#470:Nmap:01", "nwparser.payload", "%{event_description}", processor_chain([
	setc("eventcategory","1801000000"),
	dup14,
	dup15,
]));

var msg471 = msg("Nmap:01", part391);

var select77 = linear_select([
	msg470,
	msg471,
]);

var part392 = match("MESSAGE#471:Cause/0_0", "nwparser.payload", "Authentication %{result->} for principal %{fld}] %{info}");

var part393 = match("MESSAGE#471:Cause/0_1", "nwparser.payload", " %{result}] %{info}");

var select78 = linear_select([
	part392,
	part393,
]);

var all20 = all_match({
	processors: [
		select78,
	],
	on_success: processor_chain([
		setc("eventcategory","1301000000"),
		dup14,
		dup15,
	]),
});

var msg472 = msg("Cause", all20);

var part394 = match("MESSAGE#472:NEXPOSE_GENERIC", "nwparser.payload", "%{fld1}", processor_chain([
	setc("eventcategory","1901000000"),
	dup15,
]));

var msg473 = msg("NEXPOSE_GENERIC", part394);

var chain1 = processor_chain([
	select4,
	msgid_select({
		"0.16": msg435,
		"0.245": msg437,
		"0.325": msg438,
		"0.49": msg436,
		"A": msg235,
		"ACES": select76,
		"Accepting": msg327,
		"Acknowledged": msg273,
		"Acknowledging": msg272,
		"Activation": msg271,
		"Adding": select25,
		"Administrative": select49,
		"Advertising": msg217,
		"All": msg462,
		"AllowUseOriginalMessage": msg349,
		"An": msg287,
		"Analyzing": msg421,
		"Apache": msg347,
		"Applying": msg164,
		"Approved": msg267,
		"Asserting": select28,
		"AssetEventHandler": msg442,
		"AssetGroupEventHandler": msg447,
		"At": msg286,
		"Attempting": select26,
		"Authenticated": msg85,
		"Authentication": select23,
		"Auto-update": msg266,
		"Available": msg308,
		"Backing": msg372,
		"Benchmark": msg365,
		"Bulk": msg314,
		"CIFS": msg203,
		"CPU": msg304,
		"CSIDL_SYSTEM": msg209,
		"CSIDL_SYSTEMX86": msg208,
		"Cached:": msg204,
		"Cannot": select68,
		"Cataloged": msg103,
		"Cause": msg472,
		"Changing": select70,
		"CheckProcessor:": msg248,
		"Checking": select41,
		"Cleaning": select64,
		"Clearing": msg461,
		"Closing": select39,
		"Compiling": msg318,
		"Completed": select40,
		"Computer": msg302,
		"Configuring": msg336,
		"Connection": msg422,
		"Console": select12,
		"ConsoleProductInfoProvider": msg439,
		"ConsoleScanImporter": msg428,
		"Context": msg397,
		"Copied": msg398,
		"Could": msg125,
		"Created": select57,
		"Creating": msg240,
		"Current": select58,
		"DB_VERSION": msg357,
		"DEFAULT": msg396,
		"DNS": msg172,
		"Database": select62,
		"Debug": select75,
		"Default": msg346,
		"Deleted": msg345,
		"Delivered": msg391,
		"Deploying": msg282,
		"Destroying": msg277,
		"Detected": msg329,
		"Determining": select29,
		"Disk": msg309,
		"Done": select17,
		"Downloaded": msg290,
		"Downloading": msg289,
		"Dumping": msg104,
		"ERROR": select7,
		"ERROR:": msg370,
		"Enabling": msg333,
		"Engine": msg392,
		"Enumerating": msg205,
		"Error": msg353,
		"Establishing": msg264,
		"EventLog": msg359,
		"Exchange": msg211,
		"Executing": select48,
		"Exploits": msg413,
		"ExtMgr": select8,
		"FTP": msg149,
		"Failed": msg112,
		"Failure": msg414,
		"Finished": select53,
		"Firefox": msg259,
		"Flash": msg183,
		"Form": msg105,
		"Found": select33,
		"Freed": msg393,
		"Freeing": select56,
		"Generating": msg283,
		"Getting": msg190,
		"Got": msg156,
		"Graceful": msg354,
		"Granting": msg334,
		"HHH000436:": msg268,
		"Handling": msg423,
		"Host": select42,
		"IE": msg192,
		"IP": msg218,
		"Imported": select61,
		"Importing": msg315,
		"Inconsistency": msg83,
		"Initialized": select66,
		"Initializing": select51,
		"Inserted": msg344,
		"Installed": msg343,
		"Installing": select37,
		"Interrupted,": msg47,
		"Invocation": msg278,
		"Invulnerable": msg469,
		"JAR": msg276,
		"JMX": msg348,
		"JRE": msg179,
		"JVM": msg310,
		"Java": msg399,
		"Job": msg402,
		"JobStoreCMT": msg403,
		"Kill": msg295,
		"LDAP": msg424,
		"Listing": msg189,
		"Loaded": select52,
		"Loading": msg241,
		"Local": msg356,
		"Locating": msg249,
		"Logging": msg258,
		"MDAC": msg181,
		"Maintenance": msg425,
		"Making": msg360,
		"Microsoft": msg180,
		"Migration": msg426,
		"Mobile": msg427,
		"NEXPOSE_GENERIC": msg473,
		"NOT_VULNERABLE": select5,
		"NOT_VULNERABLE_VERSION": msg1,
		"NSE": select11,
		"NSXAssetEventHandler": msg440,
		"Name": msg182,
		"New": msg463,
		"Nexpose": select13,
		"Nmap": select77,
		"No": select35,
		"Number": msg305,
		"OS": msg364,
		"Operating": msg303,
		"PG": select69,
		"Parsed": msg178,
		"Parsing": msg322,
		"Patching": msg275,
		"Pausing": msg311,
		"Performing": select20,
		"Policy": select60,
		"Populating": msg358,
		"PostgreSQL": msg340,
		"Postgres": msg429,
		"Preparing": msg67,
		"Processed": msg433,
		"Processing": msg97,
		"Product": msg298,
		"ProductNotificationService": msg441,
		"ProtocolFper": msg31,
		"Quartz": select71,
		"QuartzRepeaterBuilder": msg371,
		"Queued": msg252,
		"Queueing": select18,
		"Reading": msg253,
		"Recovering": msg407,
		"Recovery": msg408,
		"Recursively": select27,
		"Reflections": msg434,
		"Refreshing": msg270,
		"Registered": select54,
		"Registering": msg254,
		"Reinitializing": msg416,
		"Relaunching": msg106,
		"Remapped": msg324,
		"Remapping": msg323,
		"Removed": select72,
		"Removing": msg285,
		"Renamed": msg415,
		"Replaced": select73,
		"Report": select67,
		"Requested": msg292,
		"Resolving": msg171,
		"Response": msg265,
		"Restarting": msg291,
		"Restoring": msg411,
		"Retrieved": msg202,
		"Retrieving": msg155,
		"Rewrote": msg65,
		"Route:": select55,
		"Running": select30,
		"SPIDER": msg66,
		"SPIDER-XSS": msg96,
		"SQL": msg212,
		"Scan": select22,
		"ScanEventHandler": msg448,
		"ScanMgr": select9,
		"Scanning": msg173,
		"Scheduler": select63,
		"Searching": msg109,
		"Security": select15,
		"Seeing": msg257,
		"Sending": msg118,
		"Service": select32,
		"Session": msg464,
		"Setting": msg361,
		"Shutdown": msg49,
		"Shutting": msg46,
		"Site": msg84,
		"SiteEventHandler": msg443,
		"Skipping": msg184,
		"Spring": msg459,
		"Staged": msg269,
		"Staging": msg284,
		"Started": select47,
		"Starting": select34,
		"Stopping": msg331,
		"Storing": msg460,
		"StreamCaching": msg355,
		"Succesfully": msg430,
		"Successfully": msg263,
		"Super": msg301,
		"Synchronizing": msg321,
		"System": select74,
		"SystemFingerprint": msg108,
		"TCP": msg250,
		"TCPSocket": msg110,
		"TagEventHandler": msg446,
		"Telling": msg330,
		"The": msg288,
		"Total": select59,
		"Truncating": msg320,
		"Trusted": msg121,
		"Trying": msg64,
		"UDP": msg251,
		"Unzipped": msg431,
		"Update": select36,
		"Updated": select46,
		"Updating": select43,
		"Upgrading": msg412,
		"User": select24,
		"UserEventHandler": msg444,
		"Using": msg279,
		"VERSION": msg328,
		"VULNERABLE": select6,
		"VULNERABLE_VERSION": msg2,
		"Validating": msg274,
		"Verifying": msg363,
		"Version": msg335,
		"Version:": msg191,
		"Vulnerability": msg319,
		"VulnerabilityExceptionEventHandler": msg445,
		"Web": select16,
		"Webmin": msg133,
		"Windows": select38,
		"building": msg117,
		"but": msg98,
		"checking": msg158,
		"com.rapid.nexpose.scanpool.stateInterval": msg373,
		"com.rapid7.nexpose.comms.clientConnectionProvider.autoPairTimeout": msg374,
		"com.rapid7.nexpose.comms.clientConnectionProvider.getConnectionTimeout": msg375,
		"com.rapid7.nexpose.datastore.connection.evictionThreadTime": msg376,
		"com.rapid7.nexpose.datastore.eviction.connection.threadIdleTimeout": msg377,
		"com.rapid7.nexpose.nsc.critical.task.executor.core.thread.pool.size": msg449,
		"com.rapid7.nexpose.nsc.critical.task.executor.max.thread.pool.size": msg458,
		"com.rapid7.nexpose.nsc.dbcc": msg378,
		"com.rapid7.nexpose.nsc.scan.multiengine.scanHaltTimeoutMilliSecond": msg450,
		"com.rapid7.nexpose.nsc.scan.scan.event.monitor.poll.duration": msg451,
		"com.rapid7.nexpose.nsc.scanExecutorService.maximumCorePoolSize": msg379,
		"com.rapid7.nexpose.nsc.scanExecutorService.minimumCorePoolSize": msg380,
		"com.rapid7.nexpose.nsc.scanExecutorService.monitorCorePoolSizeIncreaseOnSaturation": msg381,
		"com.rapid7.nexpose.nsc.scanExecutorService.monitorEnabled": msg382,
		"com.rapid7.nexpose.nsc.scanExecutorService.monitorInterval": msg383,
		"com.rapid7.nexpose.nse.excludedFileSystems": msg452,
		"com.rapid7.nexpose.nse.nscClient.connectTimeout": msg384,
		"com.rapid7.nexpose.nse.nscClient.readTimeout": msg385,
		"com.rapid7.nexpose.reportGenerator.assetCollectionUpdateTimeout": msg386,
		"com.rapid7.nexpose.scan.consolidation.delay": msg387,
		"com.rapid7.nexpose.scan.lifecyclemonitor.delay": msg388,
		"com.rapid7.nexpose.scan.logCPUMemoryToMemLog.enable": msg453,
		"com.rapid7.nexpose.scan.logMemory.interval": msg454,
		"com.rapid7.nexpose.scan.monitor.numberSavedAssetDurations": msg455,
		"com.rapid7.nexpose.scan.usescanpool": msg389,
		"com.rapid7.nsc.workflow.timeout": msg390,
		"com.rapid7.scan.perTestDurationLogging": msg456,
		"com.rapid7.thread.threadPoolNonBlockingOpsProviderParallelism": msg457,
		"common": msg261,
		"connected": msg111,
		"creating": msg120,
		"credentials": msg95,
		"dcerpc-get-ms-blaster-codes": msg124,
		"initdb": msg362,
		"j_password": msg99,
		"j_username": msg100,
		"jess.JessException:": msg262,
		"key": msg188,
		"list-user-directory": msg123,
		"loading": msg153,
		"main": msg107,
		"nodes": msg260,
		"office": msg210,
		"osspi_defaultTargetLocation": msg101,
		"param:": msg174,
		"persistent-xss": msg92,
		"removing": msg332,
		"sending": msg119,
		"shutting": msg48,
		"signon_type": msg122,
		"spider-parse-robot-exclusions": msg102,
		"starting": msg213,
		"trying": msg154,
		"unexpected": msg157,
		"using": msg142,
		"vacuumdb": msg432,
	}),
]);

var hdr37 = match("HEADER#1:0022/0", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{p0}");

var part395 = match("HEADER#1:0022/1_1", "nwparser.p0", "%{hpriority}][%{p0}");

var part396 = match("HEADER#1:0022/1_2", "nwparser.p0", "%{hpriority}[%{p0}");

var hdr38 = match("HEADER#18:0034/0", "message", "%NEXPOSE-%{hfld49}: %{hdate}T%{htime->} [%{hpriority}]%{p0}");

var part397 = match("HEADER#18:0034/1_0", "nwparser.p0", " [%{p0}");

var part398 = match("HEADER#18:0034/1_1", "nwparser.p0", "[%{p0}");

var part399 = match("MESSAGE#17:NSE:01/0", "nwparser.payload", "%{} %{p0}");

var part400 = match("MESSAGE#52:Scan:06/0", "nwparser.payload", "Scan: [ %{p0}");

var part401 = match("MESSAGE#52:Scan:06/1_0", "nwparser.p0", "%{saddr}:%{sport->} %{p0}");

var part402 = match("MESSAGE#52:Scan:06/1_1", "nwparser.p0", "%{saddr->} %{p0}");

var select79 = linear_select([
	dup7,
	dup8,
]);

var part403 = match("MESSAGE#416:Nexpose:12", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var part404 = match("MESSAGE#46:SPIDER", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup17,
]));

var select80 = linear_select([
	dup41,
	dup42,
]);

var part405 = match("MESSAGE#93:Attempting", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup16,
	dup29,
	dup30,
	dup31,
	dup32,
	dup33,
	dup34,
]));

var part406 = match("MESSAGE#120:path", "nwparser.payload", "%{info}", processor_chain([
	dup20,
	dup15,
]));

var part407 = match("MESSAGE#318:Loaded:01", "nwparser.payload", "%{info}", processor_chain([
	dup20,
	dup14,
	dup15,
]));

var part408 = match("MESSAGE#236:Finished:03", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup15,
]));

var part409 = match("MESSAGE#418:Mobile", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup25,
]));

var part410 = match("MESSAGE#435:ConsoleProductInfoProvider", "nwparser.payload", "%{fld1->} %{action}", processor_chain([
	dup20,
	dup14,
	dup15,
	dup59,
]));
