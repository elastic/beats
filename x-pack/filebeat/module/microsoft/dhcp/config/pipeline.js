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

var dup1 = match("MESSAGE#0:00/1_0", "nwparser.p0", "%{smacaddr},%{username},%{sessionid},%{fld3},%{fld4},%{fld5},%{fld7},%{fld8},%{vendor_event_cat},%{fld10},%{fld11},%{fld13}");

var dup2 = match("MESSAGE#0:00/1_1", "nwparser.p0", "%{smacaddr},%{username},%{fld2},%{fld3},%{fld4},%{fld5}");

var dup3 = match("MESSAGE#0:00/1_2", "nwparser.p0", "%{smacaddr},");

var dup4 = match("MESSAGE#0:00/1_3", "nwparser.p0", "%{smacaddr},%{fld6}");

var dup5 = match_copy("MESSAGE#0:00/1_4", "nwparser.p0", "smacaddr");

var dup6 = setc("eventcategory","1605020000");

var dup7 = setc("ec_activity","Start");

var dup8 = setc("ec_theme","Communication");

var dup9 = setf("msg","$MSG");

var dup10 = date_time({
	dest: "event_time",
	args: ["fld12","fld1"],
	fmts: [
		[dG,dc("/"),dF,dc("/"),dY,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup11 = setc("ec_activity","Stop");

var dup12 = setc("eventcategory","1605000000");

var dup13 = setc("eventcategory","1603040000");

var dup14 = setc("ec_activity","Delete");

var dup15 = setc("eventcategory","1603000000");

var dup16 = setc("ec_theme","Configuration");

var dup17 = setc("ec_outcome","Failure");

var dup18 = setc("ec_outcome","Success");

var dup19 = setc("eventcategory","1801010000");

var dup20 = setc("eventcategory","1302000000");

var dup21 = setc("ec_theme","AccessControl");

var dup22 = setc("eventcategory","1301000000");

var dup23 = setc("eventcategory","1611000000");

var dup24 = setc("ec_subject","Service");

var dup25 = linear_select([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]);

var hdr1 = match("HEADER#0:0001", "message", "%MSDHCP-%{hlevel}-%{messageid}: %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var select1 = linear_select([
	hdr1,
]);

var part1 = match("MESSAGE#0:00/0", "nwparser.payload", "00,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all1 = all_match({
	processors: [
		part1,
		dup25,
	],
	on_success: processor_chain([
		dup6,
		dup7,
		dup8,
		dup9,
		dup10,
	]),
});

var msg1 = msg("00", all1);

var part2 = match("MESSAGE#1:01/0", "nwparser.payload", "01,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all2 = all_match({
	processors: [
		part2,
		dup25,
	],
	on_success: processor_chain([
		dup6,
		dup11,
		dup8,
		dup9,
		dup10,
	]),
});

var msg2 = msg("01", all2);

var part3 = match("MESSAGE#2:02/0", "nwparser.payload", "02,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all3 = all_match({
	processors: [
		part3,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg3 = msg("02", all3);

var part4 = match("MESSAGE#3:10/0", "nwparser.payload", "10,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all4 = all_match({
	processors: [
		part4,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg4 = msg("10", all4);

var part5 = match("MESSAGE#4:11/0", "nwparser.payload", "11,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all5 = all_match({
	processors: [
		part5,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		setc("ec_activity","Restore"),
		dup8,
		dup9,
		dup10,
	]),
});

var msg5 = msg("11", all5);

var part6 = match("MESSAGE#5:12/0", "nwparser.payload", "12,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all6 = all_match({
	processors: [
		part6,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg6 = msg("12", all6);

var part7 = match("MESSAGE#6:13/0", "nwparser.payload", "13,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all7 = all_match({
	processors: [
		part7,
		dup25,
	],
	on_success: processor_chain([
		dup13,
		dup9,
		dup10,
	]),
});

var msg7 = msg("13", all7);

var part8 = match("MESSAGE#7:14/0", "nwparser.payload", "14,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all8 = all_match({
	processors: [
		part8,
		dup25,
	],
	on_success: processor_chain([
		dup13,
		dup9,
		dup10,
	]),
});

var msg8 = msg("14", all8);

var part9 = match("MESSAGE#8:15/0", "nwparser.payload", "15,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all9 = all_match({
	processors: [
		part9,
		dup25,
	],
	on_success: processor_chain([
		dup13,
		dup9,
		dup10,
	]),
});

var msg9 = msg("15", all9);

var part10 = match("MESSAGE#9:16/0", "nwparser.payload", "16,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all10 = all_match({
	processors: [
		part10,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup14,
		dup8,
		dup9,
		dup10,
	]),
});

var msg10 = msg("16", all10);

var part11 = match("MESSAGE#10:17/0", "nwparser.payload", "17,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all11 = all_match({
	processors: [
		part11,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg11 = msg("17", all11);

var part12 = match("MESSAGE#11:18/0", "nwparser.payload", "18,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all12 = all_match({
	processors: [
		part12,
		dup25,
	],
	on_success: processor_chain([
		dup15,
		dup9,
		dup10,
	]),
});

var msg12 = msg("18", all12);

var part13 = match("MESSAGE#12:20/0", "nwparser.payload", "20,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all13 = all_match({
	processors: [
		part13,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg13 = msg("20", all13);

var part14 = match("MESSAGE#13:21/0", "nwparser.payload", "21,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all14 = all_match({
	processors: [
		part14,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg14 = msg("21", all14);

var part15 = match("MESSAGE#14:22/0", "nwparser.payload", "22,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all15 = all_match({
	processors: [
		part15,
		dup25,
	],
	on_success: processor_chain([
		dup15,
		dup9,
		dup10,
	]),
});

var msg15 = msg("22", all15);

var part16 = match("MESSAGE#15:23/0", "nwparser.payload", "23,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all16 = all_match({
	processors: [
		part16,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg16 = msg("23", all16);

var part17 = match("MESSAGE#16:24/0", "nwparser.payload", "24,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all17 = all_match({
	processors: [
		part17,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg17 = msg("24", all17);

var part18 = match("MESSAGE#17:25/0", "nwparser.payload", "25,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all18 = all_match({
	processors: [
		part18,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg18 = msg("25", all18);

var part19 = match("MESSAGE#18:30/0", "nwparser.payload", "30,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all19 = all_match({
	processors: [
		part19,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup16,
		dup8,
		dup9,
		dup10,
	]),
});

var msg19 = msg("30", all19);

var part20 = match("MESSAGE#19:31/0", "nwparser.payload", "31,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all20 = all_match({
	processors: [
		part20,
		dup25,
	],
	on_success: processor_chain([
		dup13,
		dup16,
		dup17,
		dup9,
		dup10,
	]),
});

var msg20 = msg("31", all20);

var part21 = match("MESSAGE#20:32/0", "nwparser.payload", "32,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all21 = all_match({
	processors: [
		part21,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup16,
		dup18,
		dup9,
		dup10,
	]),
});

var msg21 = msg("32", all21);

var part22 = match("MESSAGE#21:33/0", "nwparser.payload", "33,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all22 = all_match({
	processors: [
		part22,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup16,
		dup18,
		dup9,
		dup10,
	]),
});

var msg22 = msg("33", all22);

var part23 = match("MESSAGE#22:36/0", "nwparser.payload", "36,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all23 = all_match({
	processors: [
		part23,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg23 = msg("36", all23);

var part24 = match("MESSAGE#23:50/0", "nwparser.payload", "50,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all24 = all_match({
	processors: [
		part24,
		dup25,
	],
	on_success: processor_chain([
		dup19,
		dup9,
		dup10,
	]),
});

var msg24 = msg("50", all24);

var part25 = match("MESSAGE#24:51/0", "nwparser.payload", "51,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all25 = all_match({
	processors: [
		part25,
		dup25,
	],
	on_success: processor_chain([
		dup20,
		dup21,
		dup18,
		dup9,
		dup10,
	]),
});

var msg25 = msg("51", all25);

var part26 = match("MESSAGE#25:52/0", "nwparser.payload", "52,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all26 = all_match({
	processors: [
		part26,
		dup25,
	],
	on_success: processor_chain([
		setc("eventcategory","1701070000"),
		dup9,
		dup10,
	]),
});

var msg26 = msg("52", all26);

var part27 = match("MESSAGE#26:53/0", "nwparser.payload", "53,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all27 = all_match({
	processors: [
		part27,
		dup25,
	],
	on_success: processor_chain([
		setc("eventcategory","1304000000"),
		dup9,
		dup10,
	]),
});

var msg27 = msg("53", all27);

var part28 = match("MESSAGE#27:54/0", "nwparser.payload", "54,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all28 = all_match({
	processors: [
		part28,
		dup25,
	],
	on_success: processor_chain([
		dup22,
		dup21,
		dup17,
		dup9,
		dup10,
	]),
});

var msg28 = msg("54", all28);

var part29 = match("MESSAGE#28:55/0", "nwparser.payload", "55,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all29 = all_match({
	processors: [
		part29,
		dup25,
	],
	on_success: processor_chain([
		dup20,
		dup9,
		dup10,
	]),
});

var msg29 = msg("55", all29);

var part30 = match("MESSAGE#29:56/0", "nwparser.payload", "56,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all30 = all_match({
	processors: [
		part30,
		dup25,
	],
	on_success: processor_chain([
		dup22,
		dup21,
		dup17,
		dup9,
		dup10,
	]),
});

var msg30 = msg("56", all30);

var part31 = match("MESSAGE#30:57/0", "nwparser.payload", "57,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all31 = all_match({
	processors: [
		part31,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg31 = msg("57", all31);

var part32 = match("MESSAGE#31:58/0", "nwparser.payload", "58,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all32 = all_match({
	processors: [
		part32,
		dup25,
	],
	on_success: processor_chain([
		dup19,
		dup8,
		dup17,
		dup9,
		dup10,
	]),
});

var msg32 = msg("58", all32);

var part33 = match("MESSAGE#32:59/0", "nwparser.payload", "59,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all33 = all_match({
	processors: [
		part33,
		dup25,
	],
	on_success: processor_chain([
		dup19,
		dup8,
		dup17,
		dup9,
		dup10,
	]),
});

var msg33 = msg("59", all33);

var part34 = match("MESSAGE#33:60/0", "nwparser.payload", "60,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all34 = all_match({
	processors: [
		part34,
		dup25,
	],
	on_success: processor_chain([
		dup15,
		dup9,
		dup10,
	]),
});

var msg34 = msg("60", all34);

var part35 = match("MESSAGE#34:61/0", "nwparser.payload", "61,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all35 = all_match({
	processors: [
		part35,
		dup25,
	],
	on_success: processor_chain([
		dup13,
		dup9,
		dup10,
	]),
});

var msg35 = msg("61", all35);

var part36 = match("MESSAGE#35:62/0", "nwparser.payload", "62,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all36 = all_match({
	processors: [
		part36,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg36 = msg("62", all36);

var part37 = match("MESSAGE#36:63/0", "nwparser.payload", "63,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all37 = all_match({
	processors: [
		part37,
		dup25,
	],
	on_success: processor_chain([
		dup12,
		dup9,
		dup10,
	]),
});

var msg37 = msg("63", all37);

var part38 = match("MESSAGE#37:64/0", "nwparser.payload", "64,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{p0}");

var all38 = all_match({
	processors: [
		part38,
		dup25,
	],
	on_success: processor_chain([
		setc("eventcategory","1703000000"),
		dup9,
		dup10,
	]),
});

var msg38 = msg("64", all38);

var part39 = match("MESSAGE#38:1103", "nwparser.payload", "1103,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg39 = msg("1103", part39);

var part40 = match("MESSAGE#39:1098", "nwparser.payload", "1098,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup19,
	dup8,
	dup17,
	dup9,
	dup10,
]));

var msg40 = msg("1098", part40);

var part41 = match("MESSAGE#40:11000", "nwparser.payload", "11000,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg41 = msg("11000", part41);

var part42 = match("MESSAGE#41:11001", "nwparser.payload", "11001,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg42 = msg("11001", part42);

var part43 = match("MESSAGE#42:11002", "nwparser.payload", "11002,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg43 = msg("11002", part43);

var part44 = match("MESSAGE#43:11003", "nwparser.payload", "11003,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg44 = msg("11003", part44);

var part45 = match("MESSAGE#44:11004", "nwparser.payload", "11004,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg45 = msg("11004", part45);

var part46 = match("MESSAGE#45:11005", "nwparser.payload", "11005,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg46 = msg("11005", part46);

var part47 = match("MESSAGE#46:11006", "nwparser.payload", "11006,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg47 = msg("11006", part47);

var part48 = match("MESSAGE#47:11007", "nwparser.payload", "11007,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg48 = msg("11007", part48);

var part49 = match("MESSAGE#48:11008", "nwparser.payload", "11008,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg49 = msg("11008", part49);

var part50 = match("MESSAGE#49:11009", "nwparser.payload", "11009,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup15,
	dup9,
	dup10,
]));

var msg50 = msg("11009", part50);

var part51 = match("MESSAGE#50:11010", "nwparser.payload", "11010,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup7,
	dup8,
	dup9,
	dup10,
]));

var msg51 = msg("11010", part51);

var part52 = match("MESSAGE#51:11011", "nwparser.payload", "11011,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup23,
	dup11,
	dup8,
	dup9,
	dup10,
]));

var msg52 = msg("11011", part52);

var part53 = match("MESSAGE#52:11012", "nwparser.payload", "11012,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup23,
	dup9,
	dup10,
]));

var msg53 = msg("11012", part53);

var part54 = match("MESSAGE#53:11013", "nwparser.payload", "11013,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg54 = msg("11013", part54);

var part55 = match("MESSAGE#54:11014", "nwparser.payload", "11014,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup15,
	dup9,
	dup10,
]));

var msg55 = msg("11014", part55);

var part56 = match("MESSAGE#55:11015", "nwparser.payload", "11015,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup15,
	dup9,
	dup10,
]));

var msg56 = msg("11015", part56);

var part57 = match("MESSAGE#56:11016", "nwparser.payload", "11016,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup15,
	dup14,
	dup16,
	dup9,
	dup10,
]));

var msg57 = msg("11016", part57);

var part58 = match("MESSAGE#57:11017", "nwparser.payload", "11017,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg58 = msg("11017", part58);

var part59 = match("MESSAGE#58:11018", "nwparser.payload", "11018,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg59 = msg("11018", part59);

var part60 = match("MESSAGE#59:11019", "nwparser.payload", "11019,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg60 = msg("11019", part60);

var part61 = match("MESSAGE#60:11020", "nwparser.payload", "11020,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg61 = msg("11020", part61);

var part62 = match("MESSAGE#61:11021", "nwparser.payload", "11021,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg62 = msg("11021", part62);

var part63 = match("MESSAGE#62:11023", "nwparser.payload", "11023,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup15,
	dup24,
	dup21,
	dup17,
	dup9,
	dup10,
]));

var msg63 = msg("11023", part63);

var part64 = match("MESSAGE#63:11024", "nwparser.payload", "11024,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup24,
	dup21,
	dup18,
	dup9,
	dup10,
]));

var msg64 = msg("11024", part64);

var part65 = match("MESSAGE#64:11025", "nwparser.payload", "11025,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg65 = msg("11025", part65);

var part66 = match("MESSAGE#65:11030", "nwparser.payload", "11030,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{fld3},%{fld4},%{fld5},%{fld6}", processor_chain([
	dup12,
	dup9,
	dup10,
]));

var msg66 = msg("11030", part66);

var part67 = match("MESSAGE#66:ID", "nwparser.payload", "ID,%{fld12},%{fld1},%{event_description},%{saddr},%{shost},%{smacaddr}", processor_chain([
	dup6,
	dup9,
	dup10,
]));

var msg67 = msg("ID", part67);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"00": msg1,
		"01": msg2,
		"02": msg3,
		"10": msg4,
		"1098": msg40,
		"11": msg5,
		"11000": msg41,
		"11001": msg42,
		"11002": msg43,
		"11003": msg44,
		"11004": msg45,
		"11005": msg46,
		"11006": msg47,
		"11007": msg48,
		"11008": msg49,
		"11009": msg50,
		"11010": msg51,
		"11011": msg52,
		"11012": msg53,
		"11013": msg54,
		"11014": msg55,
		"11015": msg56,
		"11016": msg57,
		"11017": msg58,
		"11018": msg59,
		"11019": msg60,
		"11020": msg61,
		"11021": msg62,
		"11023": msg63,
		"11024": msg64,
		"11025": msg65,
		"1103": msg39,
		"11030": msg66,
		"12": msg6,
		"13": msg7,
		"14": msg8,
		"15": msg9,
		"16": msg10,
		"17": msg11,
		"18": msg12,
		"20": msg13,
		"21": msg14,
		"22": msg15,
		"23": msg16,
		"24": msg17,
		"25": msg18,
		"30": msg19,
		"31": msg20,
		"32": msg21,
		"33": msg22,
		"36": msg23,
		"50": msg24,
		"51": msg25,
		"52": msg26,
		"53": msg27,
		"54": msg28,
		"55": msg29,
		"56": msg30,
		"57": msg31,
		"58": msg32,
		"59": msg33,
		"60": msg34,
		"61": msg35,
		"62": msg36,
		"63": msg37,
		"64": msg38,
		"ID": msg67,
	}),
]);

var part68 = match("MESSAGE#0:00/1_0", "nwparser.p0", "%{smacaddr},%{username},%{sessionid},%{fld3},%{fld4},%{fld5},%{fld7},%{fld8},%{vendor_event_cat},%{fld10},%{fld11},%{fld13}");

var part69 = match("MESSAGE#0:00/1_1", "nwparser.p0", "%{smacaddr},%{username},%{fld2},%{fld3},%{fld4},%{fld5}");

var part70 = match("MESSAGE#0:00/1_2", "nwparser.p0", "%{smacaddr},");

var part71 = match("MESSAGE#0:00/1_3", "nwparser.p0", "%{smacaddr},%{fld6}");

var part72 = match_copy("MESSAGE#0:00/1_4", "nwparser.p0", "smacaddr");

var select2 = linear_select([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]);
