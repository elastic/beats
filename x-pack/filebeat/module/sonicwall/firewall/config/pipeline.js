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

var dup2 = setc("eventcategory","1603090000");

var dup3 = setc("eventcategory","1605030000");

var dup4 = setc("eventcategory","1603060000");

var dup5 = setc("eventcategory","1603000000");

var dup6 = setc("eventcategory","1204020000");

var dup7 = match("MESSAGE#14:14:01/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var dup8 = match("MESSAGE#14:14:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst= %{p0}");

var dup9 = match("MESSAGE#14:14:01/1_1", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var dup10 = match("MESSAGE#14:14:01/2", "nwparser.p0", "%{daddr}:%{dport}:%{p0}");

var dup11 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup12 = setc("eventcategory","1502010000");

var dup13 = setc("eventcategory","1502020000");

var dup14 = setc("eventcategory","1002010000");

var dup15 = match("MESSAGE#28:23:01/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} %{p0}");

var dup16 = match("MESSAGE#28:23:01/1_1", "nwparser.p0", "%{daddr->} %{p0}");

var dup17 = match("MESSAGE#28:23:01/2", "nwparser.p0", "%{p0}");

var dup18 = setf("hostip","hhostip");

var dup19 = setf("id","hid");

var dup20 = setf("serial_number","hserial_number");

var dup21 = setf("category","hcategory");

var dup22 = setf("severity","hseverity");

var dup23 = setc("eventcategory","1805010000");

var dup24 = call({
	dest: "nwparser.msg",
	fn: RMQ,
	args: [
		field("msg"),
	],
});

var dup25 = setc("eventcategory","1302000000");

var dup26 = match("MESSAGE#38:29:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var dup27 = match("MESSAGE#38:29:01/1_1", "nwparser.p0", " %{saddr->} dst= %{p0}");

var dup28 = match("MESSAGE#38:29:01/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} ");

var dup29 = match("MESSAGE#38:29:01/2_1", "nwparser.p0", "%{daddr->} ");

var dup30 = setc("eventcategory","1401050100");

var dup31 = setc("eventcategory","1401030000");

var dup32 = match("MESSAGE#40:30:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} src=%{p0}");

var dup33 = setc("eventcategory","1301020000");

var dup34 = match("MESSAGE#49:33:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{p0}");

var dup35 = match("MESSAGE#52:35:01/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}");

var dup36 = match_copy("MESSAGE#52:35:01/2_1", "nwparser.p0", "daddr");

var dup37 = match("MESSAGE#54:36:01/1_0", "nwparser.p0", "app=%{fld51->} appName=\"%{application}\" n=%{p0}");

var dup38 = match("MESSAGE#54:36:01/1_1", "nwparser.p0", "n=%{p0}");

var dup39 = match("MESSAGE#54:36:01/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{p0}");

var dup40 = match("MESSAGE#54:36:01/3_1", "nwparser.p0", "%{saddr->} %{p0}");

var dup41 = match("MESSAGE#54:36:01/4", "nwparser.p0", "dst= %{p0}");

var dup42 = match("MESSAGE#54:36:01/7_1", "nwparser.p0", "rule=%{rule}");

var dup43 = match("MESSAGE#54:36:01/7_2", "nwparser.p0", "proto=%{protocol}");

var dup44 = date_time({
	dest: "event_time",
	args: ["date","time"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup45 = match("MESSAGE#55:36:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup46 = match("MESSAGE#55:36:02/1_1", "nwparser.p0", "%{saddr->} dst= %{p0}");

var dup47 = match_copy("MESSAGE#55:36:02/6", "nwparser.p0", "info");

var dup48 = match("MESSAGE#59:37:03/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} proto= %{p0}");

var dup49 = match("MESSAGE#59:37:03/3_1", "nwparser.p0", "%{dinterface->} proto= %{p0}");

var dup50 = match("MESSAGE#59:37:03/4", "nwparser.p0", "%{protocol->} npcs=%{info}");

var dup51 = match("MESSAGE#62:38:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src= %{p0}");

var dup52 = match("MESSAGE#63:38:02/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} type= %{p0}");

var dup53 = match("MESSAGE#63:38:02/3_1", "nwparser.p0", "%{dinterface->} type= %{p0}");

var dup54 = match("MESSAGE#64:38:03/0", "nwparser.payload", "msg=\"%{event_description}\"%{p0}");

var dup55 = match("MESSAGE#64:38:03/1_0", "nwparser.p0", " app=%{fld2->} appName=\"%{application}\"%{p0}");

var dup56 = match_copy("MESSAGE#64:38:03/1_1", "nwparser.p0", "p0");

var dup57 = match("MESSAGE#64:38:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var dup58 = match("MESSAGE#64:38:03/3_1", "nwparser.p0", "%{daddr->} srcMac=%{p0}");

var dup59 = setc("ec_subject","NetworkComm");

var dup60 = setc("ec_activity","Deny");

var dup61 = setc("ec_theme","Communication");

var dup62 = setf("msg","$MSG");

var dup63 = setc("action","dropped");

var dup64 = setc("eventcategory","1608010000");

var dup65 = setc("eventcategory","1302010000");

var dup66 = setc("eventcategory","1301000000");

var dup67 = setc("eventcategory","1001000000");

var dup68 = setc("eventcategory","1003030000");

var dup69 = setc("eventcategory","1003050000");

var dup70 = setc("eventcategory","1103000000");

var dup71 = setc("eventcategory","1603110000");

var dup72 = setc("eventcategory","1605020000");

var dup73 = match("MESSAGE#126:89:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{p0}");

var dup74 = match("MESSAGE#135:97:01/0", "nwparser.payload", "n=%{fld1->} src= %{p0}");

var dup75 = match("MESSAGE#135:97:01/6_0", "nwparser.p0", "result=%{result->} dstname=%{p0}");

var dup76 = match("MESSAGE#135:97:01/6_1", "nwparser.p0", "dstname=%{p0}");

var dup77 = match("MESSAGE#137:97:03/0", "nwparser.payload", "sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup78 = setc("eventcategory","1801000000");

var dup79 = match("MESSAGE#141:97:07/1_1", "nwparser.p0", "%{dinterface->} srcMac=%{p0}");

var dup80 = match("MESSAGE#147:98:01/6_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} %{p0}");

var dup81 = match("MESSAGE#147:98:01/7_4", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes}");

var dup82 = match("MESSAGE#148:98:06/0", "nwparser.payload", "msg=\"%{event_description}\" %{p0}");

var dup83 = match("MESSAGE#148:98:06/5_0", "nwparser.p0", "%{sinterface}:%{shost->} dst= %{p0}");

var dup84 = match("MESSAGE#148:98:06/5_1", "nwparser.p0", "%{sinterface->} dst= %{p0}");

var dup85 = match("MESSAGE#148:98:06/7_2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{p0}");

var dup86 = match("MESSAGE#148:98:06/9_3", "nwparser.p0", "sent=%{sbytes}");

var dup87 = match("MESSAGE#155:428/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var dup88 = setf("id","hfld1");

var dup89 = setc("eventcategory","1001020309");

var dup90 = setc("eventcategory","1303000000");

var dup91 = setc("eventcategory","1801010100");

var dup92 = setc("eventcategory","1604010000");

var dup93 = setc("eventcategory","1002020000");

var dup94 = match("MESSAGE#240:171:03/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} npcs= %{p0}");

var dup95 = match("MESSAGE#240:171:03/3_1", "nwparser.p0", "%{dinterface->} npcs= %{p0}");

var dup96 = match("MESSAGE#240:171:03/4", "nwparser.p0", "%{info}");

var dup97 = setc("eventcategory","1001010000");

var dup98 = match("MESSAGE#256:180:01/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} note= %{p0}");

var dup99 = match("MESSAGE#256:180:01/3_1", "nwparser.p0", "%{dinterface->} note= %{p0}");

var dup100 = match("MESSAGE#256:180:01/4", "nwparser.p0", "\"%{fld3}\" npcs=%{info}");

var dup101 = match("MESSAGE#260:194/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} sport=%{sport->} dport=%{dport->} %{p0}");

var dup102 = match("MESSAGE#260:194/1_1", "nwparser.p0", "rcvd=%{rbytes}");

var dup103 = match("MESSAGE#262:196/1_0", "nwparser.p0", "sent=%{sbytes->} cmd=%{p0}");

var dup104 = match("MESSAGE#262:196/1_1", "nwparser.p0", "rcvd=%{rbytes->} cmd=%{p0}");

var dup105 = match_copy("MESSAGE#262:196/2", "nwparser.p0", "method");

var dup106 = setc("eventcategory","1401060000");

var dup107 = setc("eventcategory","1804000000");

var dup108 = match("MESSAGE#280:261:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var dup109 = setc("eventcategory","1401070000");

var dup110 = match("MESSAGE#283:273/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{p0}");

var dup111 = setc("eventcategory","1801030000");

var dup112 = setc("eventcategory","1402020300");

var dup113 = match("MESSAGE#302:401/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} %{p0}");

var dup114 = match("MESSAGE#302:401/1_0", "nwparser.p0", "dstname=%{name}");

var dup115 = match_copy("MESSAGE#302:401/1_1", "nwparser.p0", "space");

var dup116 = setc("eventcategory","1402000000");

var dup117 = match("MESSAGE#313:446/3_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=\"%{p0}");

var dup118 = match("MESSAGE#313:446/3_1", "nwparser.p0", "%{protocol->} fw_action=\"%{p0}");

var dup119 = match("MESSAGE#313:446/4", "nwparser.p0", "%{action}\"");

var dup120 = setc("eventcategory","1803020000");

var dup121 = match("MESSAGE#318:522:01/4", "nwparser.p0", "proto=%{protocol->} npcs=%{info}");

var dup122 = match("MESSAGE#330:537:01/0", "nwparser.payload", "msg=\"%{action}\" f=%{fld1->} n=%{fld2->} src= %{p0}");

var dup123 = match_copy("MESSAGE#330:537:01/5_1", "nwparser.p0", "rbytes");

var dup124 = match("MESSAGE#332:537:08/1_0", "nwparser.p0", " app=%{fld51->} appName=\"%{application}\"n=%{p0}");

var dup125 = match("MESSAGE#332:537:08/1_1", "nwparser.p0", " app=%{fld51->} sess=\"%{fld4}\" n=%{p0}");

var dup126 = match("MESSAGE#332:537:08/1_2", "nwparser.p0", " app=%{fld51}n=%{p0}");

var dup127 = match("MESSAGE#332:537:08/2_0", "nwparser.p0", "%{fld1->} usr=\"%{username}\"src=%{p0}");

var dup128 = match("MESSAGE#332:537:08/2_1", "nwparser.p0", "%{fld1}src=%{p0}");

var dup129 = match("MESSAGE#332:537:08/6_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{p0}");

var dup130 = match("MESSAGE#332:537:08/6_1", "nwparser.p0", "%{sbytes->} spkt=%{p0}");

var dup131 = match("MESSAGE#332:537:08/7_1", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} cdur=%{fld7}");

var dup132 = match("MESSAGE#332:537:08/7_3", "nwparser.p0", "%{fld3->} cdur=%{fld7}");

var dup133 = match_copy("MESSAGE#332:537:08/7_4", "nwparser.p0", "fld3");

var dup134 = match("MESSAGE#336:537:04/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup135 = match("MESSAGE#336:537:04/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto= %{p0}");

var dup136 = match("MESSAGE#336:537:04/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto= %{p0}");

var dup137 = match("MESSAGE#336:537:04/3_2", "nwparser.p0", "%{daddr->} proto= %{p0}");

var dup138 = match("MESSAGE#338:537:10/1_0", "nwparser.p0", "usr=\"%{username}\" %{p0}");

var dup139 = match("MESSAGE#338:537:10/2", "nwparser.p0", "src=%{p0}");

var dup140 = match("MESSAGE#338:537:10/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var dup141 = match("MESSAGE#338:537:10/3_1", "nwparser.p0", "%{saddr->} dst=%{p0}");

var dup142 = match("MESSAGE#338:537:10/6_0", "nwparser.p0", "npcs=%{info}");

var dup143 = match("MESSAGE#338:537:10/6_1", "nwparser.p0", "cdur=%{fld12}");

var dup144 = setc("event_description","Connection Closed");

var dup145 = setc("eventcategory","1801020000");

var dup146 = setc("ec_activity","Permit");

var dup147 = setc("action","allowed");

var dup148 = match("MESSAGE#355:598:01/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{daddr}:%{dport}:%{p0}");

var dup149 = match("MESSAGE#361:606/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{p0}");

var dup150 = match("MESSAGE#361:606/1_0", "nwparser.p0", "%{dport}:%{dinterface->} srcMac=%{p0}");

var dup151 = match("MESSAGE#361:606/1_1", "nwparser.p0", "%{dport->} srcMac=%{p0}");

var dup152 = match("MESSAGE#361:606/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr}proto=%{p0}");

var dup153 = match("MESSAGE#362:608/4", "nwparser.p0", "%{daddr}:%{p0}");

var dup154 = match("MESSAGE#362:608/5_1", "nwparser.p0", "%{dport}:%{dinterface}");

var dup155 = match_copy("MESSAGE#362:608/5_2", "nwparser.p0", "dport");

var dup156 = setc("eventcategory","1001030500");

var dup157 = match("MESSAGE#366:712:02/0", "nwparser.payload", "msg=\"%{action}\" %{p0}");

var dup158 = match("MESSAGE#366:712:02/1_0", "nwparser.p0", "app=%{fld21->} appName=\"%{application}\" n=%{p0}");

var dup159 = match("MESSAGE#366:712:02/2", "nwparser.p0", "%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var dup160 = match("MESSAGE#366:712:02/3_0", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var dup161 = match("MESSAGE#366:712:02/3_1", "nwparser.p0", "%{smacaddr->} proto=%{p0}");

var dup162 = match("MESSAGE#366:712:02/4_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=%{p0}");

var dup163 = match("MESSAGE#366:712:02/4_1", "nwparser.p0", "%{protocol->} fw_action=%{p0}");

var dup164 = match_copy("MESSAGE#366:712:02/5", "nwparser.p0", "fld51");

var dup165 = setc("eventcategory","1801010000");

var dup166 = match("MESSAGE#391:908/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{p0}");

var dup167 = match("MESSAGE#391:908/1_1", "nwparser.p0", "%{sinterface->} dst=%{p0}");

var dup168 = match("MESSAGE#391:908/2", "nwparser.p0", "%{} %{daddr}:%{p0}");

var dup169 = match("MESSAGE#391:908/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var dup170 = setc("eventcategory","1003010000");

var dup171 = setc("eventcategory","1609000000");

var dup172 = setc("eventcategory","1204000000");

var dup173 = setc("eventcategory","1602000000");

var dup174 = match("MESSAGE#439:1199/2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} npcs=%{info}");

var dup175 = setc("eventcategory","1803000000");

var dup176 = match("MESSAGE#444:1198/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var dup177 = match("MESSAGE#461:1220/3_0", "nwparser.p0", "%{dport}:%{dinterface->} note=%{p0}");

var dup178 = match("MESSAGE#461:1220/3_1", "nwparser.p0", "%{dport->} note=%{p0}");

var dup179 = match("MESSAGE#461:1220/4", "nwparser.p0", "%{}\"%{info}\" fw_action=\"%{action}\"");

var dup180 = match("MESSAGE#471:1369/1_0", "nwparser.p0", "%{protocol}/%{fld3}fw_action=\"%{p0}");

var dup181 = match("MESSAGE#471:1369/1_1", "nwparser.p0", "%{protocol}fw_action=\"%{p0}");

var dup182 = linear_select([
	dup8,
	dup9,
]);

var dup183 = linear_select([
	dup15,
	dup16,
]);

var dup184 = match("MESSAGE#403:24:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var dup185 = linear_select([
	dup26,
	dup27,
]);

var dup186 = linear_select([
	dup28,
	dup29,
]);

var dup187 = linear_select([
	dup35,
	dup36,
]);

var dup188 = linear_select([
	dup37,
	dup38,
]);

var dup189 = linear_select([
	dup39,
	dup40,
]);

var dup190 = linear_select([
	dup26,
	dup46,
]);

var dup191 = linear_select([
	dup48,
	dup49,
]);

var dup192 = linear_select([
	dup52,
	dup53,
]);

var dup193 = linear_select([
	dup55,
	dup56,
]);

var dup194 = linear_select([
	dup57,
	dup58,
]);

var dup195 = match("MESSAGE#116:82:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup70,
]));

var dup196 = match("MESSAGE#118:83:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup5,
]));

var dup197 = linear_select([
	dup75,
	dup76,
]);

var dup198 = linear_select([
	dup83,
	dup84,
]);

var dup199 = match("MESSAGE#168:111:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=%{shost}", processor_chain([
	dup1,
]));

var dup200 = linear_select([
	dup94,
	dup95,
]);

var dup201 = match("MESSAGE#253:178", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup5,
]));

var dup202 = linear_select([
	dup98,
	dup99,
]);

var dup203 = linear_select([
	dup86,
	dup102,
]);

var dup204 = linear_select([
	dup103,
	dup104,
]);

var dup205 = match("MESSAGE#277:252", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup93,
]));

var dup206 = match("MESSAGE#293:355", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup93,
]));

var dup207 = match("MESSAGE#295:356", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup1,
]));

var dup208 = match("MESSAGE#298:358", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var dup209 = match("MESSAGE#414:371:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var dup210 = linear_select([
	dup114,
	dup115,
]);

var dup211 = linear_select([
	dup117,
	dup118,
]);

var dup212 = linear_select([
	dup43,
	dup42,
]);

var dup213 = linear_select([
	dup8,
	dup27,
]);

var dup214 = linear_select([
	dup8,
	dup26,
	dup46,
]);

var dup215 = linear_select([
	dup80,
	dup15,
	dup16,
]);

var dup216 = linear_select([
	dup124,
	dup125,
	dup126,
	dup38,
]);

var dup217 = linear_select([
	dup127,
	dup128,
]);

var dup218 = linear_select([
	dup129,
	dup130,
]);

var dup219 = linear_select([
	dup135,
	dup136,
	dup137,
]);

var dup220 = linear_select([
	dup138,
	dup56,
]);

var dup221 = linear_select([
	dup140,
	dup141,
]);

var dup222 = linear_select([
	dup142,
	dup143,
]);

var dup223 = linear_select([
	dup150,
	dup151,
]);

var dup224 = match("MESSAGE#365:710", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup156,
]));

var dup225 = linear_select([
	dup158,
	dup38,
]);

var dup226 = linear_select([
	dup160,
	dup161,
]);

var dup227 = linear_select([
	dup162,
	dup163,
]);

var dup228 = match("MESSAGE#375:766", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup5,
]));

var dup229 = match("MESSAGE#377:860:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{ntype}", processor_chain([
	dup5,
]));

var dup230 = match("MESSAGE#393:914", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{host->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{shost}", processor_chain([
	dup5,
	dup24,
]));

var dup231 = match("MESSAGE#399:994", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var dup232 = match("MESSAGE#406:1110", "nwparser.payload", "msg=\"%{msg}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
	dup24,
]));

var dup233 = match("MESSAGE#420:614", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup171,
	dup44,
]));

var dup234 = match("MESSAGE#454:654", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2}", processor_chain([
	dup1,
]));

var dup235 = linear_select([
	dup177,
	dup178,
]);

var dup236 = linear_select([
	dup180,
	dup181,
]);

var dup237 = match("MESSAGE#482:796", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var dup238 = all_match({
	processors: [
		dup32,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup31,
	]),
});

var dup239 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup91,
	]),
});

var dup240 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup67,
	]),
});

var dup241 = all_match({
	processors: [
		dup101,
		dup203,
	],
	on_success: processor_chain([
		dup67,
	]),
});

var dup242 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup106,
	]),
});

var dup243 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup31,
	]),
});

var dup244 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var dup245 = all_match({
	processors: [
		dup108,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup109,
	]),
});

var dup246 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup112,
	]),
});

var dup247 = all_match({
	processors: [
		dup113,
		dup210,
	],
	on_success: processor_chain([
		dup93,
	]),
});

var dup248 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup116,
	]),
});

var dup249 = all_match({
	processors: [
		dup51,
		dup189,
		dup41,
		dup187,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var dup250 = all_match({
	processors: [
		dup73,
		dup185,
		dup183,
		dup43,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var dup251 = all_match({
	processors: [
		dup157,
		dup225,
		dup159,
		dup226,
		dup227,
		dup164,
	],
	on_success: processor_chain([
		dup156,
		dup59,
		dup60,
		dup61,
		dup62,
		dup44,
		dup63,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var dup252 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup202,
		dup100,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var dup253 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "id=%{hfld1->} sn=%{hserial_number->} time=\"%{date->} %{time}\" fw=%{hhostip->} pri=%{hseverity->} c=%{hcategory->} m=%{messageid->} %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = match("HEADER#1:0002", "message", "id=%{hfld1->} sn=%{hserial_number->} time=\"%{date->} %{time}\" fw=%{hhostip->} pri=%{hseverity->} %{messageid}= %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("= "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "id=%{hfld1->} sn=%{hserial_number->} time=\"%{hdate->} %{htime}\" fw=%{hhostip->} pri=%{hseverity->} c=%{hcategory->} m=%{messageid->} %{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr4 = match("HEADER#3:0004", "message", "%{hfld20->} id=%{hfld1->} sn=%{hserial_number->} time=\"%{hdate->} %{htime}\" fw=%{hhostip->} pri=%{hseverity->} c=%{hcategory->} m=%{messageid->} %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
]);

var part1 = match("MESSAGE#0:4", "nwparser.payload", "SonicWALL activated%{}", processor_chain([
	dup1,
]));

var msg1 = msg("4", part1);

var part2 = match("MESSAGE#1:5", "nwparser.payload", "Log Cleared%{}", processor_chain([
	dup1,
]));

var msg2 = msg("5", part2);

var part3 = match("MESSAGE#2:5:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	dup1,
]));

var msg3 = msg("5:01", part3);

var select2 = linear_select([
	msg2,
	msg3,
]);

var part4 = match("MESSAGE#3:6", "nwparser.payload", "Log successfully sent via email%{}", processor_chain([
	dup1,
]));

var msg4 = msg("6", part4);

var part5 = match("MESSAGE#4:6:01", "nwparser.payload", "msg=\"Log successfully sent via email\" n=%{fld1}", processor_chain([
	dup1,
]));

var msg5 = msg("6:01", part5);

var select3 = linear_select([
	msg4,
	msg5,
]);

var part6 = match("MESSAGE#5:7", "nwparser.payload", "Log full; deactivating SonicWALL%{}", processor_chain([
	dup2,
]));

var msg6 = msg("7", part6);

var part7 = match("MESSAGE#6:8", "nwparser.payload", "New Filter list loaded%{}", processor_chain([
	dup3,
]));

var msg7 = msg("8", part7);

var part8 = match("MESSAGE#7:9", "nwparser.payload", "No new Filter list available%{}", processor_chain([
	dup4,
]));

var msg8 = msg("9", part8);

var part9 = match("MESSAGE#8:10", "nwparser.payload", "Problem loading the Filter list; check Filter settings%{}", processor_chain([
	dup4,
]));

var msg9 = msg("10", part9);

var part10 = match("MESSAGE#9:11", "nwparser.payload", "Problem loading the Filter list; check your DNS server%{}", processor_chain([
	dup4,
]));

var msg10 = msg("11", part10);

var part11 = match("MESSAGE#10:12", "nwparser.payload", "Problem sending log email; check log settings%{}", processor_chain([
	dup5,
]));

var msg11 = msg("12", part11);

var part12 = match("MESSAGE#11:12:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	dup5,
]));

var msg12 = msg("12:01", part12);

var select4 = linear_select([
	msg11,
	msg12,
]);

var part13 = match("MESSAGE#12:13", "nwparser.payload", "Restarting SonicWALL; dumping log to email%{}", processor_chain([
	dup1,
]));

var msg13 = msg("13", part13);

var part14 = match("MESSAGE#13:14/0_0", "nwparser.payload", "msg=\"Web site access denied\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} dstname=%{dhost->} arg=%{fld2->} code=%{icmpcode}");

var part15 = match("MESSAGE#13:14/0_1", "nwparser.payload", "Web site blocked%{}");

var select5 = linear_select([
	part14,
	part15,
]);

var all1 = all_match({
	processors: [
		select5,
	],
	on_success: processor_chain([
		dup6,
		setc("action","Web site access denied"),
	]),
});

var msg14 = msg("14", all1);

var part16 = match("MESSAGE#14:14:01/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} code= %{p0}");

var part17 = match("MESSAGE#14:14:01/3_1", "nwparser.p0", "%{dinterface->} code= %{p0}");

var select6 = linear_select([
	part16,
	part17,
]);

var part18 = match("MESSAGE#14:14:01/4", "nwparser.p0", "%{fld3->} Category=%{fld4->} npcs=%{info}");

var all2 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		select6,
		part18,
	],
	on_success: processor_chain([
		dup6,
	]),
});

var msg15 = msg("14:01", all2);

var part19 = match("MESSAGE#15:14:02", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{name->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg16 = msg("14:02", part19);

var part20 = match("MESSAGE#16:14:03", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg17 = msg("14:03", part20);

var part21 = match("MESSAGE#17:14:04", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{name->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg18 = msg("14:04", part21);

var part22 = match("MESSAGE#18:14:05", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr}dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{dhost->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg19 = msg("14:05", part22);

var select7 = linear_select([
	msg14,
	msg15,
	msg16,
	msg17,
	msg18,
	msg19,
]);

var part23 = match("MESSAGE#19:15", "nwparser.payload", "Newsgroup blocked%{}", processor_chain([
	dup12,
]));

var msg20 = msg("15", part23);

var part24 = match("MESSAGE#20:16", "nwparser.payload", "Web site accessed%{}", processor_chain([
	dup13,
]));

var msg21 = msg("16", part24);

var part25 = match("MESSAGE#21:17", "nwparser.payload", "Newsgroup accessed%{}", processor_chain([
	dup13,
]));

var msg22 = msg("17", part25);

var part26 = match("MESSAGE#22:18", "nwparser.payload", "ActiveX blocked%{}", processor_chain([
	dup12,
]));

var msg23 = msg("18", part26);

var part27 = match("MESSAGE#23:19", "nwparser.payload", "Java blocked%{}", processor_chain([
	dup12,
]));

var msg24 = msg("19", part27);

var part28 = match("MESSAGE#24:20", "nwparser.payload", "ActiveX or Java archive blocked%{}", processor_chain([
	dup12,
]));

var msg25 = msg("20", part28);

var part29 = match("MESSAGE#25:21", "nwparser.payload", "Cookie removed%{}", processor_chain([
	dup1,
]));

var msg26 = msg("21", part29);

var part30 = match("MESSAGE#26:22", "nwparser.payload", "Ping of death blocked%{}", processor_chain([
	dup14,
]));

var msg27 = msg("22", part30);

var part31 = match("MESSAGE#27:23", "nwparser.payload", "IP spoof detected%{}", processor_chain([
	dup14,
]));

var msg28 = msg("23", part31);

var part32 = match("MESSAGE#28:23:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part33 = match("MESSAGE#28:23:01/3_0", "nwparser.p0", "- MAC address: %{p0}");

var part34 = match("MESSAGE#28:23:01/3_1", "nwparser.p0", "mac= %{p0}");

var select8 = linear_select([
	part33,
	part34,
]);

var part35 = match("MESSAGE#28:23:01/4", "nwparser.p0", "%{smacaddr}");

var all3 = all_match({
	processors: [
		part32,
		dup183,
		dup17,
		select8,
		part35,
	],
	on_success: processor_chain([
		dup14,
	]),
});

var msg29 = msg("23:01", all3);

var part36 = match("MESSAGE#29:23:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} - MAC address: %{smacaddr}", processor_chain([
	dup14,
]));

var msg30 = msg("23:02", part36);

var part37 = match("MESSAGE#30:23:03/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{daddr}:%{dport}:%{p0}");

var part38 = match("MESSAGE#30:23:03/1_0", "nwparser.p0", "%{dinterface}:%{dhost->} srcMac= %{p0}");

var part39 = match("MESSAGE#30:23:03/1_1", "nwparser.p0", "%{dinterface->} srcMac= %{p0}");

var select9 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#30:23:03/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol}");

var all4 = all_match({
	processors: [
		part37,
		select9,
		part40,
	],
	on_success: processor_chain([
		dup14,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg31 = msg("23:03", all4);

var select10 = linear_select([
	msg28,
	msg29,
	msg30,
	msg31,
]);

var part41 = match("MESSAGE#31:24", "nwparser.payload", "Illegal LAN address in use%{}", processor_chain([
	dup23,
]));

var msg32 = msg("24", part41);

var msg33 = msg("24:01", dup184);

var select11 = linear_select([
	msg32,
	msg33,
]);

var part42 = match("MESSAGE#32:25", "nwparser.payload", "Possible SYN flood attack%{}", processor_chain([
	dup14,
]));

var msg34 = msg("25", part42);

var part43 = match("MESSAGE#33:26", "nwparser.payload", "Probable SYN flood attack%{}", processor_chain([
	dup14,
]));

var msg35 = msg("26", part43);

var part44 = match("MESSAGE#34:27", "nwparser.payload", "Land Attack Dropped%{}", processor_chain([
	dup14,
]));

var msg36 = msg("27", part44);

var part45 = match("MESSAGE#35:28", "nwparser.payload", "Fragmented Packet Dropped%{}", processor_chain([
	dup14,
]));

var msg37 = msg("28", part45);

var part46 = match("MESSAGE#36:28:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup14,
]));

var msg38 = msg("28:01", part46);

var select12 = linear_select([
	msg37,
	msg38,
]);

var part47 = match("MESSAGE#37:29", "nwparser.payload", "Successful administrator login%{}", processor_chain([
	dup25,
]));

var msg39 = msg("29", part47);

var part48 = match("MESSAGE#38:29:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var all5 = all_match({
	processors: [
		part48,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var msg40 = msg("29:01", all5);

var select13 = linear_select([
	msg39,
	msg40,
]);

var part49 = match("MESSAGE#39:30", "nwparser.payload", "Administrator login failed - incorrect password%{}", processor_chain([
	dup31,
]));

var msg41 = msg("30", part49);

var msg42 = msg("30:01", dup238);

var select14 = linear_select([
	msg41,
	msg42,
]);

var part50 = match("MESSAGE#41:31", "nwparser.payload", "Successful user login%{}", processor_chain([
	dup25,
]));

var msg43 = msg("31", part50);

var all6 = all_match({
	processors: [
		dup32,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup25,
	]),
});

var msg44 = msg("31:01", all6);

var part51 = match("MESSAGE#43:31:02", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup25,
	dup11,
]));

var msg45 = msg("31:02", part51);

var part52 = match("MESSAGE#44:31:03", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration}n=%{fld1}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}proto=%{protocol}note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup25,
	dup11,
]));

var msg46 = msg("31:03", part52);

var part53 = match("MESSAGE#45:31:04", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup25,
	dup11,
]));

var msg47 = msg("31:04", part53);

var select15 = linear_select([
	msg43,
	msg44,
	msg45,
	msg46,
	msg47,
]);

var part54 = match("MESSAGE#46:32", "nwparser.payload", "User login failed - incorrect password%{}", processor_chain([
	dup31,
]));

var msg48 = msg("32", part54);

var msg49 = msg("32:01", dup238);

var select16 = linear_select([
	msg48,
	msg49,
]);

var part55 = match("MESSAGE#48:33", "nwparser.payload", "Unknown user attempted to log in%{}", processor_chain([
	dup33,
]));

var msg50 = msg("33", part55);

var all7 = all_match({
	processors: [
		dup34,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup31,
	]),
});

var msg51 = msg("33:01", all7);

var select17 = linear_select([
	msg50,
	msg51,
]);

var part56 = match("MESSAGE#50:34", "nwparser.payload", "Login screen timed out%{}", processor_chain([
	dup5,
]));

var msg52 = msg("34", part56);

var part57 = match("MESSAGE#51:35", "nwparser.payload", "Attempted administrator login from WAN%{}", processor_chain([
	setc("eventcategory","1401040000"),
]));

var msg53 = msg("35", part57);

var all8 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1401050200"),
	]),
});

var msg54 = msg("35:01", all8);

var select18 = linear_select([
	msg53,
	msg54,
]);

var part58 = match("MESSAGE#53:36", "nwparser.payload", "TCP connection dropped%{}", processor_chain([
	dup5,
]));

var msg55 = msg("36", part58);

var part59 = match("MESSAGE#54:36:01/0", "nwparser.payload", "msg=\"%{msg}\" %{p0}");

var part60 = match("MESSAGE#54:36:01/2", "nwparser.p0", "%{fld1->} src= %{p0}");

var part61 = match("MESSAGE#54:36:01/7_0", "nwparser.p0", "srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"");

var select19 = linear_select([
	part61,
	dup42,
	dup43,
]);

var all9 = all_match({
	processors: [
		part59,
		dup188,
		part60,
		dup189,
		dup41,
		dup183,
		dup17,
		select19,
	],
	on_success: processor_chain([
		dup5,
		dup44,
	]),
});

var msg56 = msg("36:01", all9);

var part62 = match("MESSAGE#55:36:02/5_0", "nwparser.p0", "rule=%{rule->} npcs=%{p0}");

var part63 = match("MESSAGE#55:36:02/5_1", "nwparser.p0", "proto=%{protocol->} npcs=%{p0}");

var select20 = linear_select([
	part62,
	part63,
]);

var all10 = all_match({
	processors: [
		dup45,
		dup190,
		dup17,
		dup183,
		dup17,
		select20,
		dup47,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg57 = msg("36:02", all10);

var select21 = linear_select([
	msg55,
	msg56,
	msg57,
]);

var part64 = match("MESSAGE#56:37", "nwparser.payload", "UDP packet dropped%{}", processor_chain([
	dup5,
]));

var msg58 = msg("37", part64);

var part65 = match("MESSAGE#57:37:01/0", "nwparser.payload", "msg=\"UDP packet dropped\" %{p0}");

var part66 = match("MESSAGE#57:37:01/2", "nwparser.p0", "%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{p0}");

var part67 = match("MESSAGE#57:37:01/3_0", "nwparser.p0", "%{dport}proto=%{protocol->} fw_action=\"%{fld3}\"");

var part68 = match("MESSAGE#57:37:01/3_1", "nwparser.p0", "%{dport}rule=%{rule}");

var select22 = linear_select([
	part67,
	part68,
]);

var all11 = all_match({
	processors: [
		part65,
		dup188,
		part66,
		select22,
	],
	on_success: processor_chain([
		dup5,
		dup44,
	]),
});

var msg59 = msg("37:01", all11);

var part69 = match("MESSAGE#58:37:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} rule=%{rule}", processor_chain([
	dup5,
]));

var msg60 = msg("37:02", part69);

var all12 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup191,
		dup50,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg61 = msg("37:03", all12);

var part70 = match("MESSAGE#60:37:04", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup11,
]));

var msg62 = msg("37:04", part70);

var select23 = linear_select([
	msg58,
	msg59,
	msg60,
	msg61,
	msg62,
]);

var part71 = match("MESSAGE#61:38", "nwparser.payload", "ICMP packet dropped%{}", processor_chain([
	dup5,
]));

var msg63 = msg("38", part71);

var part72 = match("MESSAGE#62:38:01/5_0", "nwparser.p0", "type=%{type->} code=%{code}");

var select24 = linear_select([
	part72,
	dup42,
]);

var all13 = all_match({
	processors: [
		dup51,
		dup189,
		dup41,
		dup183,
		dup17,
		select24,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg64 = msg("38:01", all13);

var part73 = match("MESSAGE#63:38:02/4", "nwparser.p0", "%{fld3->} icmpCode=%{fld4->} npcs=%{info}");

var all14 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup192,
		part73,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg65 = msg("38:02", all14);

var part74 = match("MESSAGE#64:38:03/2", "nwparser.p0", "%{}n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part75 = match("MESSAGE#64:38:03/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"");

var all15 = all_match({
	processors: [
		dup54,
		dup193,
		part74,
		dup194,
		part75,
	],
	on_success: processor_chain([
		dup5,
		dup11,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg66 = msg("38:03", all15);

var select25 = linear_select([
	msg63,
	msg64,
	msg65,
	msg66,
]);

var part76 = match("MESSAGE#65:39", "nwparser.payload", "PPTP packet dropped%{}", processor_chain([
	dup5,
]));

var msg67 = msg("39", part76);

var part77 = match("MESSAGE#66:40", "nwparser.payload", "IPSec packet dropped%{}", processor_chain([
	dup5,
]));

var msg68 = msg("40", part77);

var part78 = match("MESSAGE#67:41:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} note=\"IP Protocol: %{dclass_counter1}\"", processor_chain([
	dup5,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg69 = msg("41:01", part78);

var part79 = match("MESSAGE#68:41:02", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport}:%{sinterface->} dst=%{dtransaddr}:%{dtransport}::%{dinterface}", processor_chain([
	dup5,
]));

var msg70 = msg("41:02", part79);

var part80 = match("MESSAGE#69:41:03", "nwparser.payload", "Unknown protocol dropped%{}", processor_chain([
	dup5,
]));

var msg71 = msg("41:03", part80);

var select26 = linear_select([
	msg69,
	msg70,
	msg71,
]);

var part81 = match("MESSAGE#70:42", "nwparser.payload", "IPSec packet dropped; waiting for pending IPSec connection%{}", processor_chain([
	dup5,
]));

var msg72 = msg("42", part81);

var part82 = match("MESSAGE#71:43", "nwparser.payload", "IPSec connection interrupt%{}", processor_chain([
	dup5,
]));

var msg73 = msg("43", part82);

var part83 = match("MESSAGE#72:44", "nwparser.payload", "NAT could not remap incoming packet%{}", processor_chain([
	dup5,
]));

var msg74 = msg("44", part83);

var part84 = match("MESSAGE#73:45", "nwparser.payload", "ARP timeout%{}", processor_chain([
	dup5,
]));

var msg75 = msg("45", part84);

var part85 = match("MESSAGE#74:45:01", "nwparser.payload", "msg=\"ARP timeout\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup5,
]));

var msg76 = msg("45:01", part85);

var part86 = match("MESSAGE#75:45:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr->} dst=%{daddr->} npcs=%{info}", processor_chain([
	dup5,
]));

var msg77 = msg("45:02", part86);

var select27 = linear_select([
	msg75,
	msg76,
	msg77,
]);

var part87 = match("MESSAGE#76:46:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} proto=%{protocol}/%{fld4}", processor_chain([
	dup5,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg78 = msg("46:01", part87);

var part88 = match("MESSAGE#77:46:02", "nwparser.payload", "msg=\"Broadcast packet dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup5,
]));

var msg79 = msg("46:02", part88);

var part89 = match("MESSAGE#78:46", "nwparser.payload", "Broadcast packet dropped%{}", processor_chain([
	dup5,
]));

var msg80 = msg("46", part89);

var part90 = match("MESSAGE#79:46:03/0", "nwparser.payload", "msg=\"Broadcast packet dropped\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var all16 = all_match({
	processors: [
		part90,
		dup182,
		dup10,
		dup191,
		dup50,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg81 = msg("46:03", all16);

var select28 = linear_select([
	msg78,
	msg79,
	msg80,
	msg81,
]);

var part91 = match("MESSAGE#80:47", "nwparser.payload", "No ICMP redirect sent%{}", processor_chain([
	dup5,
]));

var msg82 = msg("47", part91);

var part92 = match("MESSAGE#81:48", "nwparser.payload", "Out-of-order command packet dropped%{}", processor_chain([
	dup5,
]));

var msg83 = msg("48", part92);

var part93 = match("MESSAGE#82:49", "nwparser.payload", "Failure to add data channel%{}", processor_chain([
	dup5,
]));

var msg84 = msg("49", part93);

var part94 = match("MESSAGE#83:50", "nwparser.payload", "RealAudio decode failure%{}", processor_chain([
	dup5,
]));

var msg85 = msg("50", part94);

var part95 = match("MESSAGE#84:51", "nwparser.payload", "Duplicate packet dropped%{}", processor_chain([
	dup5,
]));

var msg86 = msg("51", part95);

var part96 = match("MESSAGE#85:52", "nwparser.payload", "No HOST tag found in HTTP request%{}", processor_chain([
	dup5,
]));

var msg87 = msg("52", part96);

var part97 = match("MESSAGE#86:53", "nwparser.payload", "The cache is full; too many open connections; some will be dropped%{}", processor_chain([
	dup2,
]));

var msg88 = msg("53", part97);

var part98 = match("MESSAGE#87:58", "nwparser.payload", "License exceeded: Connection dropped because too many IP addresses are in use on your LAN%{}", processor_chain([
	dup64,
]));

var msg89 = msg("58", part98);

var part99 = match("MESSAGE#88:60", "nwparser.payload", "Access to Proxy Server Blocked%{}", processor_chain([
	dup12,
]));

var msg90 = msg("60", part99);

var part100 = match("MESSAGE#89:61", "nwparser.payload", "Diagnostic Code E%{}", processor_chain([
	dup1,
]));

var msg91 = msg("61", part100);

var part101 = match("MESSAGE#90:62", "nwparser.payload", "Dynamic IPSec client connected%{}", processor_chain([
	dup65,
]));

var msg92 = msg("62", part101);

var part102 = match("MESSAGE#91:63", "nwparser.payload", "IPSec packet too big%{}", processor_chain([
	dup66,
]));

var msg93 = msg("63", part102);

var part103 = match("MESSAGE#92:63:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup66,
]));

var msg94 = msg("63:01", part103);

var select29 = linear_select([
	msg93,
	msg94,
]);

var part104 = match("MESSAGE#93:64", "nwparser.payload", "Diagnostic Code D%{}", processor_chain([
	dup1,
]));

var msg95 = msg("64", part104);

var part105 = match("MESSAGE#94:65", "nwparser.payload", "Illegal IPSec SPI%{}", processor_chain([
	dup66,
]));

var msg96 = msg("65", part105);

var part106 = match("MESSAGE#95:66", "nwparser.payload", "Unknown IPSec SPI%{}", processor_chain([
	dup66,
]));

var msg97 = msg("66", part106);

var part107 = match("MESSAGE#96:67", "nwparser.payload", "IPSec Authentication Failed%{}", processor_chain([
	dup66,
]));

var msg98 = msg("67", part107);

var all17 = all_match({
	processors: [
		dup32,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup66,
	]),
});

var msg99 = msg("67:01", all17);

var select30 = linear_select([
	msg98,
	msg99,
]);

var part108 = match("MESSAGE#98:68", "nwparser.payload", "IPSec Decryption Failed%{}", processor_chain([
	dup66,
]));

var msg100 = msg("68", part108);

var part109 = match("MESSAGE#99:69", "nwparser.payload", "Incompatible IPSec Security Association%{}", processor_chain([
	dup66,
]));

var msg101 = msg("69", part109);

var part110 = match("MESSAGE#100:70", "nwparser.payload", "IPSec packet from illegal host%{}", processor_chain([
	dup66,
]));

var msg102 = msg("70", part110);

var part111 = match("MESSAGE#101:70:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst%{p0}");

var part112 = match("MESSAGE#101:70:01/1_0", "nwparser.p0", "=%{daddr}");

var part113 = match("MESSAGE#101:70:01/1_1", "nwparser.p0", "name=%{name}");

var select31 = linear_select([
	part112,
	part113,
]);

var all18 = all_match({
	processors: [
		part111,
		select31,
	],
	on_success: processor_chain([
		dup66,
	]),
});

var msg103 = msg("70:01", all18);

var select32 = linear_select([
	msg102,
	msg103,
]);

var part114 = match("MESSAGE#102:72", "nwparser.payload", "NetBus Attack Dropped%{}", processor_chain([
	dup67,
]));

var msg104 = msg("72", part114);

var part115 = match("MESSAGE#103:72:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup67,
]));

var msg105 = msg("72:01", part115);

var select33 = linear_select([
	msg104,
	msg105,
]);

var part116 = match("MESSAGE#104:73", "nwparser.payload", "Back Orifice Attack Dropped%{}", processor_chain([
	dup68,
]));

var msg106 = msg("73", part116);

var part117 = match("MESSAGE#105:74", "nwparser.payload", "Net Spy Attack Dropped%{}", processor_chain([
	dup69,
]));

var msg107 = msg("74", part117);

var part118 = match("MESSAGE#106:75", "nwparser.payload", "Sub Seven Attack Dropped%{}", processor_chain([
	dup68,
]));

var msg108 = msg("75", part118);

var part119 = match("MESSAGE#107:76", "nwparser.payload", "Ripper Attack Dropped%{}", processor_chain([
	dup67,
]));

var msg109 = msg("76", part119);

var part120 = match("MESSAGE#108:77", "nwparser.payload", "Striker Attack Dropped%{}", processor_chain([
	dup67,
]));

var msg110 = msg("77", part120);

var part121 = match("MESSAGE#109:78", "nwparser.payload", "Senna Spy Attack Dropped%{}", processor_chain([
	dup69,
]));

var msg111 = msg("78", part121);

var part122 = match("MESSAGE#110:79", "nwparser.payload", "Priority Attack Dropped%{}", processor_chain([
	dup67,
]));

var msg112 = msg("79", part122);

var part123 = match("MESSAGE#111:80", "nwparser.payload", "Ini Killer Attack Dropped%{}", processor_chain([
	dup67,
]));

var msg113 = msg("80", part123);

var part124 = match("MESSAGE#112:81", "nwparser.payload", "Smurf Amplification Attack Dropped%{}", processor_chain([
	dup14,
]));

var msg114 = msg("81", part124);

var part125 = match("MESSAGE#113:82", "nwparser.payload", "Possible Port Scan%{}", processor_chain([
	dup70,
]));

var msg115 = msg("82", part125);

var part126 = match("MESSAGE#114:82:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{info}\"", processor_chain([
	dup70,
]));

var msg116 = msg("82:02", part126);

var part127 = match("MESSAGE#115:82:03", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{fld3}\" npcs=%{info}", processor_chain([
	dup70,
]));

var msg117 = msg("82:03", part127);

var msg118 = msg("82:01", dup195);

var select34 = linear_select([
	msg115,
	msg116,
	msg117,
	msg118,
]);

var part128 = match("MESSAGE#117:83", "nwparser.payload", "Probable Port Scan%{}", processor_chain([
	dup70,
]));

var msg119 = msg("83", part128);

var msg120 = msg("83:01", dup196);

var part129 = match("MESSAGE#119:83:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{fld3}\" npcs=%{info}", processor_chain([
	dup5,
]));

var msg121 = msg("83:02", part129);

var select35 = linear_select([
	msg119,
	msg120,
	msg121,
]);

var part130 = match("MESSAGE#120:84/0_0", "nwparser.payload", "msg=\"Failed to resolve name\" n=%{fld1->} dstname=%{dhost}");

var part131 = match("MESSAGE#120:84/0_1", "nwparser.payload", "Failed to resolve name%{}");

var select36 = linear_select([
	part130,
	part131,
]);

var all19 = all_match({
	processors: [
		select36,
	],
	on_success: processor_chain([
		dup71,
		setc("action","Failed to resolve name"),
	]),
});

var msg122 = msg("84", all19);

var part132 = match("MESSAGE#121:87", "nwparser.payload", "IKE Responder: Accepting IPSec proposal%{}", processor_chain([
	dup72,
]));

var msg123 = msg("87", part132);

var part133 = match("MESSAGE#122:87:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup72,
]));

var msg124 = msg("87:01", part133);

var select37 = linear_select([
	msg123,
	msg124,
]);

var part134 = match("MESSAGE#123:88", "nwparser.payload", "IKE Responder: IPSec proposal not acceptable%{}", processor_chain([
	dup66,
]));

var msg125 = msg("88", part134);

var part135 = match("MESSAGE#124:88:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup66,
]));

var msg126 = msg("88:01", part135);

var select38 = linear_select([
	msg125,
	msg126,
]);

var part136 = match("MESSAGE#125:89", "nwparser.payload", "IKE negotiation complete. Adding IPSec SA%{}", processor_chain([
	dup72,
]));

var msg127 = msg("89", part136);

var part137 = match("MESSAGE#126:89:01/1_0", "nwparser.p0", "%{saddr}:::%{sinterface->} dst=%{daddr}:::%{dinterface}");

var part138 = match("MESSAGE#126:89:01/1_1", "nwparser.p0", "%{saddr->} dst=%{daddr->} dstname=%{name}");

var select39 = linear_select([
	part137,
	part138,
]);

var all20 = all_match({
	processors: [
		dup73,
		select39,
	],
	on_success: processor_chain([
		dup72,
	]),
});

var msg128 = msg("89:01", all20);

var select40 = linear_select([
	msg127,
	msg128,
]);

var part139 = match("MESSAGE#127:90", "nwparser.payload", "Starting IKE negotiation%{}", processor_chain([
	dup72,
]));

var msg129 = msg("90", part139);

var part140 = match("MESSAGE#128:91", "nwparser.payload", "Deleting IPSec SA for destination%{}", processor_chain([
	dup72,
]));

var msg130 = msg("91", part140);

var part141 = match("MESSAGE#129:92", "nwparser.payload", "Deleting IPSec SA%{}", processor_chain([
	dup72,
]));

var msg131 = msg("92", part141);

var part142 = match("MESSAGE#130:93", "nwparser.payload", "Diagnostic Code A%{}", processor_chain([
	dup1,
]));

var msg132 = msg("93", part142);

var part143 = match("MESSAGE#131:94", "nwparser.payload", "Diagnostic Code B%{}", processor_chain([
	dup1,
]));

var msg133 = msg("94", part143);

var part144 = match("MESSAGE#132:95", "nwparser.payload", "Diagnostic Code C%{}", processor_chain([
	dup1,
]));

var msg134 = msg("95", part144);

var part145 = match("MESSAGE#133:96", "nwparser.payload", "Status%{}", processor_chain([
	dup1,
]));

var msg135 = msg("96", part145);

var part146 = match("MESSAGE#134:97", "nwparser.payload", "Web site hit%{}", processor_chain([
	dup1,
]));

var msg136 = msg("97", part146);

var part147 = match("MESSAGE#135:97:01/4", "nwparser.p0", "proto=%{protocol->} op=%{fld->} %{p0}");

var part148 = match("MESSAGE#135:97:01/5_0", "nwparser.p0", "rcvd=%{rbytes->} %{p0}");

var part149 = match("MESSAGE#135:97:01/5_1", "nwparser.p0", "sent=%{sbytes->} %{p0}");

var select41 = linear_select([
	part148,
	part149,
]);

var part150 = match_copy("MESSAGE#135:97:01/7", "nwparser.p0", "name");

var all21 = all_match({
	processors: [
		dup74,
		dup189,
		dup41,
		dup183,
		part147,
		select41,
		dup197,
		part150,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg137 = msg("97:01", all21);

var part151 = match("MESSAGE#136:97:02/4", "nwparser.p0", "proto=%{protocol->} op=%{fld->} result=%{result}");

var all22 = all_match({
	processors: [
		dup74,
		dup189,
		dup41,
		dup183,
		part151,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg138 = msg("97:02", all22);

var part152 = match("MESSAGE#137:97:03/4", "nwparser.p0", "proto=%{protocol->} op=%{fld3->} sent=%{sbytes->} rcvd=%{rbytes->} %{p0}");

var part153 = match("MESSAGE#137:97:03/6", "nwparser.p0", "%{} %{name}arg=%{fld4->} code=%{fld5->} Category=\"%{category}\" npcs=%{info}");

var all23 = all_match({
	processors: [
		dup77,
		dup189,
		dup41,
		dup183,
		part152,
		dup197,
		part153,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg139 = msg("97:03", all23);

var part154 = match("MESSAGE#138:97:04/4", "nwparser.p0", "proto=%{protocol->} op=%{fld3->} %{p0}");

var part155 = match("MESSAGE#138:97:04/6", "nwparser.p0", "%{}arg= %{name}%{fld4->} code=%{fld5->} Category=\"%{category}\" npcs=%{info}");

var all24 = all_match({
	processors: [
		dup77,
		dup189,
		dup41,
		dup183,
		part154,
		dup197,
		part155,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg140 = msg("97:04", all24);

var part156 = match("MESSAGE#139:97:05/4", "nwparser.p0", "proto=%{protocol->} op=%{fld2->} dstname=%{name->} arg=%{fld3->} code=%{fld4->} Category=%{category}");

var all25 = all_match({
	processors: [
		dup74,
		dup189,
		dup41,
		dup183,
		part156,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg141 = msg("97:05", all25);

var part157 = match("MESSAGE#140:97:06/0", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{p0}");

var part158 = match("MESSAGE#140:97:06/1_0", "nwparser.p0", "%{sinterface}:%{shost}dst=%{p0}");

var part159 = match("MESSAGE#140:97:06/1_1", "nwparser.p0", "%{sinterface}dst=%{p0}");

var select42 = linear_select([
	part158,
	part159,
]);

var part160 = match("MESSAGE#140:97:06/2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"");

var all26 = all_match({
	processors: [
		part157,
		select42,
		part160,
	],
	on_success: processor_chain([
		dup78,
		dup11,
	]),
});

var msg142 = msg("97:06", all26);

var part161 = match("MESSAGE#141:97:07/0", "nwparser.payload", "app=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{p0}");

var part162 = match("MESSAGE#141:97:07/1_0", "nwparser.p0", "%{dinterface}:%{fld3->} srcMac=%{p0}");

var select43 = linear_select([
	part162,
	dup79,
]);

var part163 = match("MESSAGE#141:97:07/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} dstname=%{dhost->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"");

var all27 = all_match({
	processors: [
		part161,
		select43,
		part163,
	],
	on_success: processor_chain([
		dup78,
		dup11,
	]),
});

var msg143 = msg("97:07", all27);

var part164 = match("MESSAGE#142:97:08", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup11,
]));

var msg144 = msg("97:08", part164);

var part165 = match("MESSAGE#143:97:09", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup11,
]));

var msg145 = msg("97:09", part165);

var part166 = match("MESSAGE#144:97:10", "nwparser.payload", "app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup11,
]));

var msg146 = msg("97:10", part166);

var select44 = linear_select([
	msg136,
	msg137,
	msg138,
	msg139,
	msg140,
	msg141,
	msg142,
	msg143,
	msg144,
	msg145,
	msg146,
]);

var part167 = match("MESSAGE#145:98/2", "nwparser.p0", "%{}n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{p0}");

var part168 = match("MESSAGE#145:98/3_0", "nwparser.p0", "%{dinterface} %{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var part169 = match("MESSAGE#145:98/3_1", "nwparser.p0", "%{dinterface} %{protocol->} sent=%{sbytes}");

var part170 = match("MESSAGE#145:98/3_2", "nwparser.p0", "%{dinterface} %{protocol}");

var select45 = linear_select([
	part168,
	part169,
	part170,
]);

var all28 = all_match({
	processors: [
		dup54,
		dup193,
		part167,
		select45,
	],
	on_success: processor_chain([
		dup78,
		dup59,
		setc("ec_activity","Stop"),
		dup61,
		dup62,
		dup11,
		setc("action","Opened"),
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg147 = msg("98", all28);

var part171 = match("MESSAGE#146:98:07", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} dstMac=%{dmacaddr->} proto=%{protocol}/%{fld4->} sent=%{sbytes->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg148 = msg("98:07", part171);

var part172 = match("MESSAGE#147:98:01/0", "nwparser.payload", "msg=\"%{msg}\"%{p0}");

var part173 = match("MESSAGE#147:98:01/1_0", "nwparser.p0", " app=%{fld2->} sess=\"%{fld3}\"%{p0}");

var select46 = linear_select([
	part173,
	dup56,
]);

var part174 = match("MESSAGE#147:98:01/2", "nwparser.p0", "%{}n=%{p0}");

var part175 = match("MESSAGE#147:98:01/3_0", "nwparser.p0", "%{fld1->} usr=%{username->} src=%{p0}");

var part176 = match("MESSAGE#147:98:01/3_1", "nwparser.p0", "%{fld1->} src=%{p0}");

var select47 = linear_select([
	part175,
	part176,
]);

var part177 = match("MESSAGE#147:98:01/4_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{p0}");

var part178 = match("MESSAGE#147:98:01/4_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}dst=%{p0}");

var part179 = match("MESSAGE#147:98:01/4_2", "nwparser.p0", "%{saddr}dst=%{p0}");

var select48 = linear_select([
	part177,
	part178,
	part179,
]);

var part180 = match("MESSAGE#147:98:01/5", "nwparser.p0", "%{} %{p0}");

var part181 = match("MESSAGE#147:98:01/6_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} %{p0}");

var part182 = match("MESSAGE#147:98:01/6_2", "nwparser.p0", "%{daddr->} %{p0}");

var select49 = linear_select([
	dup80,
	part181,
	part182,
]);

var part183 = match("MESSAGE#147:98:01/7_0", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var part184 = match("MESSAGE#147:98:01/7_1", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes}");

var part185 = match("MESSAGE#147:98:01/7_2", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} rule=\"%{rulename}\" fw_action=\"%{action}\"");

var part186 = match("MESSAGE#147:98:01/7_3", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var select50 = linear_select([
	part183,
	part184,
	part185,
	part186,
	dup81,
	dup43,
]);

var all29 = all_match({
	processors: [
		part172,
		select46,
		part174,
		select47,
		select48,
		part180,
		select49,
		select50,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg149 = msg("98:01", all29);

var part187 = match("MESSAGE#148:98:06/1_0", "nwparser.p0", "app=%{fld2->} appName=\"%{application}\" n=%{p0}");

var part188 = match("MESSAGE#148:98:06/1_1", "nwparser.p0", "app=%{fld2->} n=%{p0}");

var part189 = match("MESSAGE#148:98:06/1_2", "nwparser.p0", "sess=%{fld2->} n=%{p0}");

var select51 = linear_select([
	part187,
	part188,
	part189,
]);

var part190 = match("MESSAGE#148:98:06/2", "nwparser.p0", "%{fld1->} %{p0}");

var part191 = match("MESSAGE#148:98:06/3_0", "nwparser.p0", "usr=%{username->} %{p0}");

var select52 = linear_select([
	part191,
	dup56,
]);

var part192 = match("MESSAGE#148:98:06/4", "nwparser.p0", "src= %{saddr}:%{sport}:%{p0}");

var part193 = match("MESSAGE#148:98:06/7_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{dmacaddr->} proto=%{p0}");

var part194 = match("MESSAGE#148:98:06/7_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} dstMac=%{dmacaddr->} proto=%{p0}");

var part195 = match("MESSAGE#148:98:06/7_3", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var select53 = linear_select([
	part193,
	part194,
	dup85,
	part195,
]);

var part196 = match("MESSAGE#148:98:06/8", "nwparser.p0", "%{protocol->} %{p0}");

var part197 = match("MESSAGE#148:98:06/9_0", "nwparser.p0", "sent=%{sbytes->} rule=\"%{rulename}\" fw_action=\"%{action}\"");

var part198 = match("MESSAGE#148:98:06/9_1", "nwparser.p0", "sent=%{sbytes->} rule=\"%{rulename}\" fw_action=%{action}");

var part199 = match("MESSAGE#148:98:06/9_2", "nwparser.p0", "sent=%{sbytes->} fw_action=\"%{action}\"");

var part200 = match("MESSAGE#148:98:06/9_4", "nwparser.p0", "fw_action=\"%{action}\"");

var select54 = linear_select([
	part197,
	part198,
	part199,
	dup86,
	part200,
]);

var all30 = all_match({
	processors: [
		dup82,
		select51,
		part190,
		select52,
		part192,
		dup198,
		dup17,
		select53,
		part196,
		select54,
	],
	on_success: processor_chain([
		dup78,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg150 = msg("98:06", all30);

var part201 = match("MESSAGE#149:98:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} usr=%{username->} src=%{p0}");

var all31 = all_match({
	processors: [
		part201,
		dup185,
		dup183,
		dup43,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg151 = msg("98:02", all31);

var part202 = match("MESSAGE#150:98:03/0_0", "nwparser.payload", "Connection%{}");

var part203 = match("MESSAGE#150:98:03/0_1", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}");

var select55 = linear_select([
	part202,
	part203,
]);

var all32 = all_match({
	processors: [
		select55,
	],
	on_success: processor_chain([
		dup1,
		dup44,
	]),
});

var msg152 = msg("98:03", all32);

var part204 = match("MESSAGE#151:98:04/3", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} vpnpolicy=\"%{policyname}\" npcs=%{info}");

var all33 = all_match({
	processors: [
		dup7,
		dup185,
		dup183,
		part204,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg153 = msg("98:04", all33);

var part205 = match("MESSAGE#152:98:05/3", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} npcs=%{info}");

var all34 = all_match({
	processors: [
		dup7,
		dup185,
		dup183,
		part205,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg154 = msg("98:05", all34);

var select56 = linear_select([
	msg147,
	msg148,
	msg149,
	msg150,
	msg151,
	msg152,
	msg153,
	msg154,
]);

var part206 = match("MESSAGE#153:986", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup31,
	dup11,
]));

var msg155 = msg("986", part206);

var part207 = match("MESSAGE#154:427/3", "nwparser.p0", "note=\"%{event_description}\"");

var all35 = all_match({
	processors: [
		dup73,
		dup185,
		dup183,
		part207,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg156 = msg("427", all35);

var part208 = match("MESSAGE#155:428/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"");

var all36 = all_match({
	processors: [
		dup87,
		dup194,
		part208,
	],
	on_success: processor_chain([
		dup23,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg157 = msg("428", all36);

var part209 = match("MESSAGE#156:99", "nwparser.payload", "Retransmitting DHCP DISCOVER.%{}", processor_chain([
	dup72,
]));

var msg158 = msg("99", part209);

var part210 = match("MESSAGE#157:100", "nwparser.payload", "Retransmitting DHCP REQUEST (Requesting).%{}", processor_chain([
	dup72,
]));

var msg159 = msg("100", part210);

var part211 = match("MESSAGE#158:101", "nwparser.payload", "Retransmitting DHCP REQUEST (Renewing).%{}", processor_chain([
	dup72,
]));

var msg160 = msg("101", part211);

var part212 = match("MESSAGE#159:102", "nwparser.payload", "Retransmitting DHCP REQUEST (Rebinding).%{}", processor_chain([
	dup72,
]));

var msg161 = msg("102", part212);

var part213 = match("MESSAGE#160:103", "nwparser.payload", "Retransmitting DHCP REQUEST (Rebooting).%{}", processor_chain([
	dup72,
]));

var msg162 = msg("103", part213);

var part214 = match("MESSAGE#161:104", "nwparser.payload", "Retransmitting DHCP REQUEST (Verifying).%{}", processor_chain([
	dup72,
]));

var msg163 = msg("104", part214);

var part215 = match("MESSAGE#162:105", "nwparser.payload", "Sending DHCP DISCOVER.%{}", processor_chain([
	dup72,
]));

var msg164 = msg("105", part215);

var part216 = match("MESSAGE#163:106", "nwparser.payload", "DHCP Server not available. Did not get any DHCP OFFER.%{}", processor_chain([
	dup71,
]));

var msg165 = msg("106", part216);

var part217 = match("MESSAGE#164:107", "nwparser.payload", "Got DHCP OFFER. Selecting.%{}", processor_chain([
	dup72,
]));

var msg166 = msg("107", part217);

var part218 = match("MESSAGE#165:108", "nwparser.payload", "Sending DHCP REQUEST.%{}", processor_chain([
	dup72,
]));

var msg167 = msg("108", part218);

var part219 = match("MESSAGE#166:109", "nwparser.payload", "DHCP Client did not get DHCP ACK.%{}", processor_chain([
	dup71,
]));

var msg168 = msg("109", part219);

var part220 = match("MESSAGE#167:110", "nwparser.payload", "DHCP Client got NACK.%{}", processor_chain([
	dup72,
]));

var msg169 = msg("110", part220);

var msg170 = msg("111:01", dup199);

var part221 = match("MESSAGE#169:111", "nwparser.payload", "DHCP Client got ACK from server.%{}", processor_chain([
	dup72,
]));

var msg171 = msg("111", part221);

var select57 = linear_select([
	msg170,
	msg171,
]);

var part222 = match("MESSAGE#170:112", "nwparser.payload", "DHCP Client is declining address offered by the server.%{}", processor_chain([
	dup72,
]));

var msg172 = msg("112", part222);

var part223 = match("MESSAGE#171:113", "nwparser.payload", "DHCP Client sending REQUEST and going to REBIND state.%{}", processor_chain([
	dup72,
]));

var msg173 = msg("113", part223);

var part224 = match("MESSAGE#172:114", "nwparser.payload", "DHCP Client sending REQUEST and going to RENEW state.%{}", processor_chain([
	dup72,
]));

var msg174 = msg("114", part224);

var msg175 = msg("115:01", dup199);

var part225 = match("MESSAGE#174:115", "nwparser.payload", "Sending DHCP REQUEST (Renewing).%{}", processor_chain([
	dup72,
]));

var msg176 = msg("115", part225);

var select58 = linear_select([
	msg175,
	msg176,
]);

var part226 = match("MESSAGE#175:116", "nwparser.payload", "Sending DHCP REQUEST (Rebinding).%{}", processor_chain([
	dup72,
]));

var msg177 = msg("116", part226);

var part227 = match("MESSAGE#176:117", "nwparser.payload", "Sending DHCP REQUEST (Rebooting).%{}", processor_chain([
	dup72,
]));

var msg178 = msg("117", part227);

var part228 = match("MESSAGE#177:118", "nwparser.payload", "Sending DHCP REQUEST (Verifying).%{}", processor_chain([
	dup72,
]));

var msg179 = msg("118", part228);

var part229 = match("MESSAGE#178:119", "nwparser.payload", "DHCP Client failed to verify and lease has expired. Go to INIT state.%{}", processor_chain([
	dup71,
]));

var msg180 = msg("119", part229);

var part230 = match("MESSAGE#179:120", "nwparser.payload", "DHCP Client failed to verify and lease is still valid. Go to BOUND state.%{}", processor_chain([
	dup71,
]));

var msg181 = msg("120", part230);

var part231 = match("MESSAGE#180:121", "nwparser.payload", "DHCP Client got a new IP address lease.%{}", processor_chain([
	dup72,
]));

var msg182 = msg("121", part231);

var part232 = match("MESSAGE#181:122", "nwparser.payload", "Access attempt from host without Anti-Virus agent installed%{}", processor_chain([
	dup71,
]));

var msg183 = msg("122", part232);

var part233 = match("MESSAGE#182:123", "nwparser.payload", "Anti-Virus agent out-of-date on host%{}", processor_chain([
	dup71,
]));

var msg184 = msg("123", part233);

var part234 = match("MESSAGE#183:124", "nwparser.payload", "Received AV Alert: %s%{}", processor_chain([
	dup72,
]));

var msg185 = msg("124", part234);

var part235 = match("MESSAGE#184:125", "nwparser.payload", "Unused AV log entry.%{}", processor_chain([
	dup72,
]));

var msg186 = msg("125", part235);

var part236 = match("MESSAGE#185:1254", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup89,
	dup11,
]));

var msg187 = msg("1254", part236);

var part237 = match("MESSAGE#186:1256", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg188 = msg("1256", part237);

var part238 = match("MESSAGE#187:1257", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup89,
	dup11,
]));

var msg189 = msg("1257", part238);

var part239 = match("MESSAGE#188:126", "nwparser.payload", "Starting PPPoE discovery%{}", processor_chain([
	dup72,
]));

var msg190 = msg("126", part239);

var part240 = match("MESSAGE#189:127", "nwparser.payload", "PPPoE LCP Link Up%{}", processor_chain([
	dup72,
]));

var msg191 = msg("127", part240);

var part241 = match("MESSAGE#190:128", "nwparser.payload", "PPPoE LCP Link Down%{}", processor_chain([
	dup5,
]));

var msg192 = msg("128", part241);

var part242 = match("MESSAGE#191:129", "nwparser.payload", "PPPoE terminated%{}", processor_chain([
	dup5,
]));

var msg193 = msg("129", part242);

var part243 = match("MESSAGE#192:130", "nwparser.payload", "PPPoE Network Connected%{}", processor_chain([
	dup1,
]));

var msg194 = msg("130", part243);

var part244 = match("MESSAGE#193:131", "nwparser.payload", "PPPoE Network Disconnected%{}", processor_chain([
	dup1,
]));

var msg195 = msg("131", part244);

var part245 = match("MESSAGE#194:132", "nwparser.payload", "PPPoE discovery process complete%{}", processor_chain([
	dup1,
]));

var msg196 = msg("132", part245);

var part246 = match("MESSAGE#195:133", "nwparser.payload", "PPPoE starting CHAP Authentication%{}", processor_chain([
	dup1,
]));

var msg197 = msg("133", part246);

var part247 = match("MESSAGE#196:134", "nwparser.payload", "PPPoE starting PAP Authentication%{}", processor_chain([
	dup1,
]));

var msg198 = msg("134", part247);

var part248 = match("MESSAGE#197:135", "nwparser.payload", "PPPoE CHAP Authentication Failed%{}", processor_chain([
	dup90,
]));

var msg199 = msg("135", part248);

var part249 = match("MESSAGE#198:136", "nwparser.payload", "PPPoE PAP Authentication Failed%{}", processor_chain([
	dup90,
]));

var msg200 = msg("136", part249);

var part250 = match("MESSAGE#199:137", "nwparser.payload", "Wan IP Changed%{}", processor_chain([
	dup3,
]));

var msg201 = msg("137", part250);

var part251 = match("MESSAGE#200:138", "nwparser.payload", "XAUTH Succeeded%{}", processor_chain([
	dup3,
]));

var msg202 = msg("138", part251);

var part252 = match("MESSAGE#201:139", "nwparser.payload", "XAUTH Failed%{}", processor_chain([
	dup5,
]));

var msg203 = msg("139", part252);

var all37 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1801020100"),
	]),
});

var msg204 = msg("139:01", all37);

var select59 = linear_select([
	msg203,
	msg204,
]);

var msg205 = msg("140", dup239);

var msg206 = msg("141", dup239);

var part253 = match("MESSAGE#205:142", "nwparser.payload", "Primary firewall has transitioned to Active%{}", processor_chain([
	dup1,
]));

var msg207 = msg("142", part253);

var part254 = match("MESSAGE#206:143", "nwparser.payload", "Backup firewall has transitioned to Active%{}", processor_chain([
	dup1,
]));

var msg208 = msg("143", part254);

var part255 = match("MESSAGE#207:1431", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=::%{sinterface->} dstV6=%{daddr_v6->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup11,
]));

var msg209 = msg("1431", part255);

var part256 = match("MESSAGE#208:144", "nwparser.payload", "Primary firewall has transitioned to Idle%{}", processor_chain([
	dup1,
]));

var msg210 = msg("144", part256);

var part257 = match("MESSAGE#209:145", "nwparser.payload", "Backup firewall has transitioned to Idle%{}", processor_chain([
	dup1,
]));

var msg211 = msg("145", part257);

var part258 = match("MESSAGE#210:146", "nwparser.payload", "Primary missed heartbeats from Active Backup: Primary going Active%{}", processor_chain([
	dup92,
]));

var msg212 = msg("146", part258);

var part259 = match("MESSAGE#211:147", "nwparser.payload", "Backup missed heartbeats from Active Primary: Backup going Active%{}", processor_chain([
	dup92,
]));

var msg213 = msg("147", part259);

var part260 = match("MESSAGE#212:148", "nwparser.payload", "Primary received error signal from Active Backup: Primary going Active%{}", processor_chain([
	dup1,
]));

var msg214 = msg("148", part260);

var part261 = match("MESSAGE#213:1480", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	setc("eventcategory","1204010000"),
	dup11,
]));

var msg215 = msg("1480", part261);

var part262 = match("MESSAGE#214:149", "nwparser.payload", "Backup received error signal from Active Primary: Backup going Active%{}", processor_chain([
	dup1,
]));

var msg216 = msg("149", part262);

var part263 = match("MESSAGE#215:150", "nwparser.payload", "Backup firewall being preempted by Primary%{}", processor_chain([
	dup1,
]));

var msg217 = msg("150", part263);

var part264 = match("MESSAGE#216:151", "nwparser.payload", "Primary firewall preempting Backup%{}", processor_chain([
	dup1,
]));

var msg218 = msg("151", part264);

var part265 = match("MESSAGE#217:152", "nwparser.payload", "Active Backup detects Active Primary: Backup rebooting%{}", processor_chain([
	dup1,
]));

var msg219 = msg("152", part265);

var part266 = match("MESSAGE#218:153", "nwparser.payload", "Imported HA hardware ID did not match this firewall%{}", processor_chain([
	setc("eventcategory","1603010000"),
]));

var msg220 = msg("153", part266);

var part267 = match("MESSAGE#219:154", "nwparser.payload", "Received AV Alert: Your SonicWALL Network Anti-Virus subscription has expired. %s%{}", processor_chain([
	dup64,
]));

var msg221 = msg("154", part267);

var part268 = match("MESSAGE#220:155", "nwparser.payload", "Primary received heartbeat from wrong source%{}", processor_chain([
	dup92,
]));

var msg222 = msg("155", part268);

var part269 = match("MESSAGE#221:156", "nwparser.payload", "Backup received heartbeat from wrong source%{}", processor_chain([
	dup92,
]));

var msg223 = msg("156", part269);

var part270 = match("MESSAGE#222:157:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup1,
]));

var msg224 = msg("157:01", part270);

var part271 = match("MESSAGE#223:157", "nwparser.payload", "HA packet processing error%{}", processor_chain([
	dup5,
]));

var msg225 = msg("157", part271);

var select60 = linear_select([
	msg224,
	msg225,
]);

var part272 = match("MESSAGE#224:158", "nwparser.payload", "Heartbeat received from incompatible source%{}", processor_chain([
	dup92,
]));

var msg226 = msg("158", part272);

var part273 = match("MESSAGE#225:159", "nwparser.payload", "Diagnostic Code F%{}", processor_chain([
	dup5,
]));

var msg227 = msg("159", part273);

var part274 = match("MESSAGE#226:160", "nwparser.payload", "Forbidden E-mail attachment altered%{}", processor_chain([
	setc("eventcategory","1203000000"),
]));

var msg228 = msg("160", part274);

var part275 = match("MESSAGE#227:161", "nwparser.payload", "PPPoE PAP Authentication success.%{}", processor_chain([
	dup65,
]));

var msg229 = msg("161", part275);

var part276 = match("MESSAGE#228:162", "nwparser.payload", "PPPoE PAP Authentication Failed. Please verify PPPoE username and password%{}", processor_chain([
	dup33,
]));

var msg230 = msg("162", part276);

var part277 = match("MESSAGE#229:163", "nwparser.payload", "Disconnecting PPPoE due to traffic timeout%{}", processor_chain([
	dup5,
]));

var msg231 = msg("163", part277);

var part278 = match("MESSAGE#230:164", "nwparser.payload", "No response from ISP Disconnecting PPPoE.%{}", processor_chain([
	dup5,
]));

var msg232 = msg("164", part278);

var part279 = match("MESSAGE#231:165", "nwparser.payload", "Backup going Active in preempt mode after reboot%{}", processor_chain([
	dup1,
]));

var msg233 = msg("165", part279);

var part280 = match("MESSAGE#232:166", "nwparser.payload", "Denied TCP connection from LAN%{}", processor_chain([
	dup12,
]));

var msg234 = msg("166", part280);

var part281 = match("MESSAGE#233:167", "nwparser.payload", "Denied UDP packet from LAN%{}", processor_chain([
	dup12,
]));

var msg235 = msg("167", part281);

var part282 = match("MESSAGE#234:168", "nwparser.payload", "Denied ICMP packet from LAN%{}", processor_chain([
	dup12,
]));

var msg236 = msg("168", part282);

var part283 = match("MESSAGE#235:169", "nwparser.payload", "Firewall access from LAN%{}", processor_chain([
	dup1,
]));

var msg237 = msg("169", part283);

var part284 = match("MESSAGE#236:170", "nwparser.payload", "Received a path MTU icmp message from router/gateway%{}", processor_chain([
	dup1,
]));

var msg238 = msg("170", part284);

var part285 = match("MESSAGE#237:171", "nwparser.payload", "Probable TCP FIN scan%{}", processor_chain([
	dup70,
]));

var msg239 = msg("171", part285);

var part286 = match("MESSAGE#238:171:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup93,
]));

var msg240 = msg("171:01", part286);

var part287 = match("MESSAGE#239:171:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}:%{dport}", processor_chain([
	dup93,
]));

var msg241 = msg("171:02", part287);

var part288 = match("MESSAGE#240:171:03/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld1}\" sess=%{fld2->} n=%{fld3->} src=%{p0}");

var all38 = all_match({
	processors: [
		part288,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup93,
	]),
});

var msg242 = msg("171:03", all38);

var select61 = linear_select([
	msg239,
	msg240,
	msg241,
	msg242,
]);

var part289 = match("MESSAGE#241:172", "nwparser.payload", "Probable TCP XMAS scan%{}", processor_chain([
	dup70,
]));

var msg243 = msg("172", part289);

var part290 = match("MESSAGE#242:172:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	dup70,
]));

var msg244 = msg("172:01", part290);

var select62 = linear_select([
	msg243,
	msg244,
]);

var part291 = match("MESSAGE#243:173", "nwparser.payload", "Probable TCP NULL scan%{}", processor_chain([
	dup70,
]));

var msg245 = msg("173", part291);

var part292 = match("MESSAGE#244:174", "nwparser.payload", "IPSEC Replay Detected%{}", processor_chain([
	dup67,
]));

var msg246 = msg("174", part292);

var all39 = all_match({
	processors: [
		dup73,
		dup185,
		dup183,
		dup43,
	],
	on_success: processor_chain([
		dup67,
	]),
});

var msg247 = msg("174:01", all39);

var all40 = all_match({
	processors: [
		dup51,
		dup189,
		dup41,
		dup187,
	],
	on_success: processor_chain([
		dup12,
	]),
});

var msg248 = msg("174:02", all40);

var all41 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup191,
		dup50,
	],
	on_success: processor_chain([
		dup12,
	]),
});

var msg249 = msg("174:03", all41);

var select63 = linear_select([
	msg246,
	msg247,
	msg248,
	msg249,
]);

var part293 = match("MESSAGE#248:175", "nwparser.payload", "TCP FIN packet dropped%{}", processor_chain([
	dup67,
]));

var msg250 = msg("175", part293);

var part294 = match("MESSAGE#249:175:01", "nwparser.payload", "msg=\"ICMP packet from LAN dropped\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} type=%{type}", processor_chain([
	dup67,
]));

var msg251 = msg("175:01", part294);

var part295 = match("MESSAGE#250:175:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr->} dst=%{daddr->} type=%{type->} icmpCode=%{fld3->} npcs=%{info}", processor_chain([
	dup67,
]));

var msg252 = msg("175:02", part295);

var select64 = linear_select([
	msg250,
	msg251,
	msg252,
]);

var part296 = match("MESSAGE#251:176", "nwparser.payload", "Fraudulent Microsoft Certificate Blocked%{}", processor_chain([
	dup93,
]));

var msg253 = msg("176", part296);

var msg254 = msg("177", dup196);

var msg255 = msg("178", dup201);

var msg256 = msg("179", dup196);

var all42 = all_match({
	processors: [
		dup34,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup97,
	]),
});

var msg257 = msg("180", all42);

var all43 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup202,
		dup100,
	],
	on_success: processor_chain([
		dup97,
	]),
});

var msg258 = msg("180:01", all43);

var select65 = linear_select([
	msg257,
	msg258,
]);

var msg259 = msg("181", dup195);

var all44 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup70,
	]),
});

var msg260 = msg("181:01", all44);

var select66 = linear_select([
	msg259,
	msg260,
]);

var msg261 = msg("193", dup240);

var msg262 = msg("194", dup241);

var msg263 = msg("195", dup241);

var part297 = match("MESSAGE#262:196/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{fld2->} dst=%{daddr}:%{fld3->} sport=%{sport->} dport=%{dport->} %{p0}");

var all45 = all_match({
	processors: [
		part297,
		dup204,
		dup105,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg264 = msg("196", all45);

var all46 = all_match({
	processors: [
		dup101,
		dup204,
		dup105,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg265 = msg("196:01", all46);

var select67 = linear_select([
	msg264,
	msg265,
]);

var msg266 = msg("199", dup242);

var msg267 = msg("200", dup243);

var part298 = match("MESSAGE#266:235:02", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} usr=%{username->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup30,
]));

var msg268 = msg("235:02", part298);

var part299 = match("MESSAGE#267:235/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} usr=%{username->} src=%{p0}");

var all47 = all_match({
	processors: [
		part299,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var msg269 = msg("235", all47);

var msg270 = msg("235:01", dup244);

var select68 = linear_select([
	msg268,
	msg269,
	msg270,
]);

var msg271 = msg("236", dup244);

var msg272 = msg("237", dup242);

var msg273 = msg("238", dup242);

var part300 = match("MESSAGE#272:239", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr->} dst=%{dtransaddr}", processor_chain([
	dup107,
]));

var msg274 = msg("239", part300);

var part301 = match("MESSAGE#273:240", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr->} dst=%{dtransaddr}", processor_chain([
	dup107,
]));

var msg275 = msg("240", part301);

var part302 = match("MESSAGE#274:241", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup78,
]));

var msg276 = msg("241", part302);

var part303 = match("MESSAGE#275:241:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup78,
]));

var msg277 = msg("241:01", part303);

var select69 = linear_select([
	msg276,
	msg277,
]);

var part304 = match("MESSAGE#276:242/1_0", "nwparser.p0", "%{saddr}:%{sport}:: %{p0}");

var part305 = match("MESSAGE#276:242/1_1", "nwparser.p0", "%{saddr}:%{sport->} %{p0}");

var select70 = linear_select([
	part304,
	part305,
	dup40,
]);

var part306 = match("MESSAGE#276:242/3_0", "nwparser.p0", "%{daddr}:%{dport}::");

var part307 = match("MESSAGE#276:242/3_1", "nwparser.p0", "%{daddr}:%{dport}");

var select71 = linear_select([
	part306,
	part307,
	dup36,
]);

var all48 = all_match({
	processors: [
		dup51,
		select70,
		dup41,
		select71,
	],
	on_success: processor_chain([
		dup78,
	]),
});

var msg278 = msg("242", all48);

var msg279 = msg("252", dup205);

var msg280 = msg("255", dup205);

var msg281 = msg("257", dup205);

var msg282 = msg("261:01", dup245);

var msg283 = msg("261", dup205);

var select72 = linear_select([
	msg282,
	msg283,
]);

var msg284 = msg("262", dup245);

var all49 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg285 = msg("273", all49);

var msg286 = msg("328", dup246);

var msg287 = msg("329", dup243);

var msg288 = msg("346", dup205);

var msg289 = msg("350", dup205);

var msg290 = msg("351", dup205);

var msg291 = msg("352", dup205);

var msg292 = msg("353:01", dup201);

var part308 = match("MESSAGE#291:353", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr->} dst=%{dtransaddr->} dstname=%{shost->} lifeSeconds=%{misc}\"", processor_chain([
	dup5,
]));

var msg293 = msg("353", part308);

var select73 = linear_select([
	msg292,
	msg293,
]);

var part309 = match("MESSAGE#292:354", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=\"%{shost->} lifeSeconds=%{misc}\"", processor_chain([
	dup1,
]));

var msg294 = msg("354", part309);

var msg295 = msg("355", dup206);

var msg296 = msg("355:01", dup205);

var select74 = linear_select([
	msg295,
	msg296,
]);

var msg297 = msg("356", dup207);

var part310 = match("MESSAGE#296:357", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport->} dstname=%{name}", processor_chain([
	dup93,
]));

var msg298 = msg("357", part310);

var part311 = match("MESSAGE#297:357:01", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup93,
]));

var msg299 = msg("357:01", part311);

var select75 = linear_select([
	msg298,
	msg299,
]);

var msg300 = msg("358", dup208);

var part312 = match("MESSAGE#299:371", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr->} dst=%{dtransaddr->} dstname=%{shost}", processor_chain([
	setc("eventcategory","1503000000"),
]));

var msg301 = msg("371", part312);

var msg302 = msg("371:01", dup209);

var select76 = linear_select([
	msg301,
	msg302,
]);

var msg303 = msg("372", dup205);

var msg304 = msg("373", dup207);

var msg305 = msg("401", dup247);

var msg306 = msg("402", dup247);

var msg307 = msg("406", dup208);

var part313 = match("MESSAGE#305:413", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var msg308 = msg("413", part313);

var msg309 = msg("414", dup205);

var msg310 = msg("438", dup248);

var msg311 = msg("439", dup248);

var all50 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1501020000"),
	]),
});

var msg312 = msg("440", all50);

var all51 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1502050000"),
	]),
});

var msg313 = msg("441", all51);

var part314 = match("MESSAGE#311:441:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	setc("eventcategory","1001020000"),
]));

var msg314 = msg("441:01", part314);

var select77 = linear_select([
	msg313,
	msg314,
]);

var all52 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1501030000"),
	]),
});

var msg315 = msg("442", all52);

var part315 = match("MESSAGE#313:446/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{p0}");

var part316 = match("MESSAGE#313:446/1_0", "nwparser.p0", "%{fld1->} appName=\"%{application}\" n=%{p0}");

var part317 = match("MESSAGE#313:446/1_1", "nwparser.p0", "%{fld1->} n=%{p0}");

var select78 = linear_select([
	part316,
	part317,
]);

var part318 = match("MESSAGE#313:446/2", "nwparser.p0", "%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var all53 = all_match({
	processors: [
		part315,
		select78,
		part318,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup67,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg316 = msg("446", all53);

var part319 = match("MESSAGE#314:477", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} note=\"MAC=%{smacaddr->} HostName:%{hostname}\"", processor_chain([
	dup120,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg317 = msg("477", part319);

var all54 = all_match({
	processors: [
		dup73,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var msg318 = msg("509", all54);

var all55 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup109,
	]),
});

var msg319 = msg("520", all55);

var msg320 = msg("522", dup249);

var part320 = match("MESSAGE#318:522:01/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} srcV6=%{saddr_v6->} src= %{p0}");

var part321 = match("MESSAGE#318:522:01/2", "nwparser.p0", "dstV6=%{daddr_v6->} dst= %{p0}");

var all56 = all_match({
	processors: [
		part320,
		dup189,
		part321,
		dup183,
		dup121,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg321 = msg("522:01", all56);

var part322 = match("MESSAGE#319:522:02/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{shost->} dst= %{p0}");

var select79 = linear_select([
	part322,
	dup46,
]);

var all57 = all_match({
	processors: [
		dup45,
		select79,
		dup17,
		dup183,
		dup121,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg322 = msg("522:02", all57);

var select80 = linear_select([
	msg320,
	msg321,
	msg322,
]);

var msg323 = msg("523", dup249);

var all58 = all_match({
	processors: [
		dup73,
		dup185,
		dup183,
		dup17,
		dup212,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg324 = msg("524", all58);

var part323 = match("MESSAGE#322:524:01/4_0", "nwparser.p0", "proto=%{protocol->} npcs= %{p0}");

var part324 = match("MESSAGE#322:524:01/4_1", "nwparser.p0", "rule=%{rule->} npcs= %{p0}");

var select81 = linear_select([
	part323,
	part324,
]);

var all59 = all_match({
	processors: [
		dup7,
		dup185,
		dup183,
		dup17,
		select81,
		dup47,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg325 = msg("524:01", all59);

var part325 = match("MESSAGE#323:524:02/0", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol}rule=\"%{rule}\"%{p0}");

var part326 = match("MESSAGE#323:524:02/1_0", "nwparser.p0", " note=\"%{rulename}\"%{p0}");

var select82 = linear_select([
	part326,
	dup56,
]);

var part327 = match("MESSAGE#323:524:02/2", "nwparser.p0", "%{}fw_action=\"%{action}\"");

var all60 = all_match({
	processors: [
		part325,
		select82,
		part327,
	],
	on_success: processor_chain([
		dup6,
		dup11,
	]),
});

var msg326 = msg("524:02", all60);

var select83 = linear_select([
	msg324,
	msg325,
	msg326,
]);

var msg327 = msg("526", dup250);

var part328 = match("MESSAGE#325:526:01/1_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{fld20->} dst= %{p0}");

var select84 = linear_select([
	dup26,
	part328,
	dup46,
]);

var part329 = match("MESSAGE#325:526:01/3_1", "nwparser.p0", "%{daddr}");

var select85 = linear_select([
	dup35,
	part329,
]);

var all61 = all_match({
	processors: [
		dup73,
		select84,
		dup17,
		select85,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg328 = msg("526:01", all61);

var all62 = all_match({
	processors: [
		dup7,
		dup213,
		dup183,
		dup121,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg329 = msg("526:02", all62);

var part330 = match("MESSAGE#327:526:03", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg330 = msg("526:03", part330);

var part331 = match("MESSAGE#328:526:04", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg331 = msg("526:04", part331);

var part332 = match("MESSAGE#329:526:05", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg332 = msg("526:05", part332);

var select86 = linear_select([
	msg327,
	msg328,
	msg329,
	msg330,
	msg331,
	msg332,
]);

var part333 = match("MESSAGE#330:537:01/4", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} rcvd=%{p0}");

var part334 = match("MESSAGE#330:537:01/5_0", "nwparser.p0", "%{rbytes->} vpnpolicy=%{fld3}");

var select87 = linear_select([
	part334,
	dup123,
]);

var all63 = all_match({
	processors: [
		dup122,
		dup214,
		dup17,
		dup215,
		part333,
		select87,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg333 = msg("537:01", all63);

var all64 = all_match({
	processors: [
		dup122,
		dup214,
		dup17,
		dup215,
		dup81,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg334 = msg("537:02", all64);

var part335 = match("MESSAGE#332:537:08/3_0", "nwparser.p0", "%{saddr} %{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{p0}");

var part336 = match("MESSAGE#332:537:08/3_1", "nwparser.p0", "%{saddr->} %{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part337 = match("MESSAGE#332:537:08/3_2", "nwparser.p0", "%{saddr->} %{daddr}srcMac=%{p0}");

var select88 = linear_select([
	part335,
	part336,
	part337,
]);

var part338 = match("MESSAGE#332:537:08/4", "nwparser.p0", "%{} %{smacaddr->} %{p0}");

var part339 = match("MESSAGE#332:537:08/5_0", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var part340 = match("MESSAGE#332:537:08/5_1", "nwparser.p0", "proto=%{protocol->} sent=%{p0}");

var select89 = linear_select([
	part339,
	part340,
]);

var part341 = match("MESSAGE#332:537:08/7_0", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} cdur=%{fld7->} fw_action=\"%{action}\"");

var part342 = match("MESSAGE#332:537:08/7_2", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} fw_action=\"%{action}\"");

var select90 = linear_select([
	part341,
	dup131,
	part342,
	dup132,
	dup133,
]);

var all65 = all_match({
	processors: [
		dup54,
		dup216,
		dup217,
		select88,
		part338,
		select89,
		dup218,
		select90,
	],
	on_success: processor_chain([
		dup111,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg335 = msg("537:08", all65);

var select91 = linear_select([
	dup125,
	dup124,
	dup126,
	dup38,
]);

var part343 = match("MESSAGE#333:537:09/3_0", "nwparser.p0", "%{saddr} %{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{p0}");

var part344 = match("MESSAGE#333:537:09/3_1", "nwparser.p0", "%{saddr->} %{daddr}:%{dport}:%{dinterface->} dstMac=%{p0}");

var part345 = match("MESSAGE#333:537:09/3_2", "nwparser.p0", "%{saddr->} %{daddr}dstMac=%{p0}");

var select92 = linear_select([
	part343,
	part344,
	part345,
]);

var part346 = match("MESSAGE#333:537:09/4", "nwparser.p0", "%{} %{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var part347 = match("MESSAGE#333:537:09/6_0", "nwparser.p0", "%{fld3->} cdur=%{fld7->} fw_action=\"%{action}\"");

var select93 = linear_select([
	part347,
	dup131,
	dup132,
	dup133,
]);

var all66 = all_match({
	processors: [
		dup54,
		select91,
		dup217,
		select92,
		part346,
		dup218,
		select93,
	],
	on_success: processor_chain([
		dup111,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg336 = msg("537:09", all66);

var part348 = match("MESSAGE#334:537:07/3_0", "nwparser.p0", "%{saddr} %{fld3->} cdur=%{fld7->} fw_action=\"%{action}\"");

var part349 = match("MESSAGE#334:537:07/3_1", "nwparser.p0", "%{saddr} %{fld3->} rpkt=%{fld6->} cdur=%{fld7}");

var part350 = match("MESSAGE#334:537:07/3_2", "nwparser.p0", "%{saddr} %{fld3->} cdur=%{fld7}");

var part351 = match("MESSAGE#334:537:07/3_3", "nwparser.p0", "%{saddr} %{fld3->} fw_action=\"%{action}\"");

var part352 = match("MESSAGE#334:537:07/3_4", "nwparser.p0", "%{saddr} %{fld3}");

var select94 = linear_select([
	part348,
	part349,
	part350,
	part351,
	part352,
]);

var all67 = all_match({
	processors: [
		dup54,
		dup216,
		dup217,
		select94,
	],
	on_success: processor_chain([
		dup111,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg337 = msg("537:07", all67);

var part353 = match("MESSAGE#335:537/0", "nwparser.payload", "msg=\"%{action}\"%{p0}");

var part354 = match("MESSAGE#335:537/1_0", "nwparser.p0", " app=%{fld51->} appName=\"%{application}\"%{p0}");

var select95 = linear_select([
	part354,
	dup56,
]);

var part355 = match("MESSAGE#335:537/2", "nwparser.p0", "%{}n=%{fld1->} src= %{p0}");

var part356 = match("MESSAGE#335:537/3_0", "nwparser.p0", "%{saddr}%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{p0}");

var part357 = match("MESSAGE#335:537/3_1", "nwparser.p0", "%{saddr} %{daddr}:%{dport}:%{dinterface}: proto=%{p0}");

var part358 = match("MESSAGE#335:537/3_2", "nwparser.p0", "%{saddr}%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part359 = match("MESSAGE#335:537/3_3", "nwparser.p0", "%{saddr}%{daddr->} proto=%{p0}");

var select96 = linear_select([
	part356,
	part357,
	part358,
	part359,
]);

var part360 = match("MESSAGE#335:537/4", "nwparser.p0", "%{protocol->} sent=%{p0}");

var part361 = match("MESSAGE#335:537/5_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} cdur=%{fld5->} fw_action=\"%{fld6}\"");

var part362 = match("MESSAGE#335:537/5_1", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} fw_action=\"%{fld5}\"");

var part363 = match("MESSAGE#335:537/5_2", "nwparser.p0", "%{sbytes->} spkt=%{fld3}fw_action=\"%{fld4}\"");

var part364 = match("MESSAGE#335:537/5_3", "nwparser.p0", "%{sbytes}rcvd=%{rbytes}");

var part365 = match_copy("MESSAGE#335:537/5_4", "nwparser.p0", "sbytes");

var select97 = linear_select([
	part361,
	part362,
	part363,
	part364,
	part365,
]);

var all68 = all_match({
	processors: [
		part353,
		select95,
		part355,
		select96,
		part360,
		select97,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg338 = msg("537", all68);

var part366 = match("MESSAGE#336:537:04/4", "nwparser.p0", "%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} cdur=%{fld5->} npcs=%{info}");

var all69 = all_match({
	processors: [
		dup134,
		dup190,
		dup17,
		dup219,
		part366,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg339 = msg("537:04", all69);

var part367 = match("MESSAGE#337:537:05/4", "nwparser.p0", "%{protocol->} sent=%{sbytes->} spkt=%{fld3->} cdur=%{fld4->} %{p0}");

var part368 = match("MESSAGE#337:537:05/5_0", "nwparser.p0", "appcat=%{fld5->} appid=%{fld6->} npcs= %{p0}");

var part369 = match("MESSAGE#337:537:05/5_1", "nwparser.p0", "npcs= %{p0}");

var select98 = linear_select([
	part368,
	part369,
]);

var all70 = all_match({
	processors: [
		dup134,
		dup190,
		dup17,
		dup219,
		part367,
		select98,
		dup96,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg340 = msg("537:05", all70);

var part370 = match("MESSAGE#338:537:10/0", "nwparser.payload", "msg=\"%{event_description}\" sess=%{fld1->} n=%{fld2->} %{p0}");

var part371 = match("MESSAGE#338:537:10/4_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{p0}");

var part372 = match("MESSAGE#338:537:10/4_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} dstMac=%{p0}");

var part373 = match("MESSAGE#338:537:10/4_2", "nwparser.p0", "%{daddr->} dstMac=%{p0}");

var select99 = linear_select([
	part371,
	part372,
	part373,
]);

var part374 = match("MESSAGE#338:537:10/5", "nwparser.p0", "%{} %{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld10->} rpkt=%{fld11->} %{p0}");

var all71 = all_match({
	processors: [
		part370,
		dup220,
		dup139,
		dup221,
		select99,
		part374,
		dup222,
	],
	on_success: processor_chain([
		dup111,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg341 = msg("537:10", all71);

var part375 = match("MESSAGE#339:537:03/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{fld2->} %{p0}");

var part376 = match("MESSAGE#339:537:03/4_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part377 = match("MESSAGE#339:537:03/4_2", "nwparser.p0", "%{daddr->} proto=%{p0}");

var select100 = linear_select([
	dup85,
	part376,
	part377,
]);

var part378 = match("MESSAGE#339:537:03/5", "nwparser.p0", "%{} %{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld10->} rpkt=%{fld11->} %{p0}");

var all72 = all_match({
	processors: [
		part375,
		dup220,
		dup139,
		dup221,
		select100,
		part378,
		dup222,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg342 = msg("537:03", all72);

var part379 = match("MESSAGE#340:537:06/4", "nwparser.p0", "%{protocol->} sent=%{sbytes->} spkt=%{fld3->} npcs=%{info}");

var all73 = all_match({
	processors: [
		dup134,
		dup190,
		dup17,
		dup219,
		part379,
	],
	on_success: processor_chain([
		dup111,
	]),
});

var msg343 = msg("537:06", all73);

var part380 = match("MESSAGE#341:537:11", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}spkt=%{fld3}rpkt=%{fld4}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup111,
	dup62,
	dup11,
	dup144,
]));

var msg344 = msg("537:11", part380);

var part381 = match("MESSAGE#342:537:12", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup111,
	dup62,
	dup11,
	dup144,
]));

var msg345 = msg("537:12", part381);

var select101 = linear_select([
	msg333,
	msg334,
	msg335,
	msg336,
	msg337,
	msg338,
	msg339,
	msg340,
	msg341,
	msg342,
	msg343,
	msg344,
	msg345,
]);

var msg346 = msg("538", dup240);

var msg347 = msg("549", dup243);

var msg348 = msg("557", dup243);

var all74 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1402020200"),
	]),
});

var msg349 = msg("558", all74);

var msg350 = msg("561", dup246);

var msg351 = msg("562", dup246);

var msg352 = msg("563", dup246);

var all75 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		setc("eventcategory","1402020400"),
	]),
});

var msg353 = msg("583", all75);

var part382 = match("MESSAGE#351:597:01", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} type=%{icmptype->} code=%{icmpcode}", processor_chain([
	dup145,
	dup59,
	dup146,
	dup61,
	dup62,
	dup11,
	dup147,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg354 = msg("597:01", part382);

var part383 = match("MESSAGE#352:597:02", "nwparser.payload", "msg=%{msg->} n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} type=%{icmptype->} code=%{icmpcode}", processor_chain([
	dup1,
]));

var msg355 = msg("597:02", part383);

var part384 = match("MESSAGE#353:597:03/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src= %{saddr}:%{sport}:%{p0}");

var part385 = match("MESSAGE#353:597:03/2", "nwparser.p0", "%{daddr}:%{dport}:%{p0}");

var all76 = all_match({
	processors: [
		part384,
		dup198,
		part385,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg356 = msg("597:03", all76);

var select102 = linear_select([
	msg354,
	msg355,
	msg356,
]);

var part386 = match("MESSAGE#354:598", "nwparser.payload", "msg=%{msg->} n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} type=%{type->} code=%{code}", processor_chain([
	dup1,
]));

var msg357 = msg("598", part386);

var part387 = match("MESSAGE#355:598:01/2", "nwparser.p0", "%{type->} npcs=%{info}");

var all77 = all_match({
	processors: [
		dup148,
		dup192,
		part387,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg358 = msg("598:01", all77);

var all78 = all_match({
	processors: [
		dup148,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg359 = msg("598:02", all78);

var select103 = linear_select([
	msg357,
	msg358,
	msg359,
]);

var part388 = match("MESSAGE#357:602:01", "nwparser.payload", "msg=\"%{event_description}allowed\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} proto=%{protocol}/%{fld4}", processor_chain([
	dup145,
	dup59,
	dup146,
	dup61,
	dup62,
	dup11,
	dup147,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg360 = msg("602:01", part388);

var msg361 = msg("602:02", dup250);

var all79 = all_match({
	processors: [
		dup7,
		dup185,
		dup183,
		dup43,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg362 = msg("602:03", all79);

var select104 = linear_select([
	msg360,
	msg361,
	msg362,
]);

var msg363 = msg("605", dup208);

var all80 = all_match({
	processors: [
		dup149,
		dup223,
		dup152,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup93,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg364 = msg("606", all80);

var part389 = match("MESSAGE#362:608/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} ipscat=%{ipscat->} ipspri=%{p0}");

var part390 = match("MESSAGE#362:608/1_0", "nwparser.p0", "%{fld66->} pktdatId=%{fld11->} n=%{p0}");

var part391 = match("MESSAGE#362:608/1_1", "nwparser.p0", "%{ipspri->} n=%{p0}");

var select105 = linear_select([
	part390,
	part391,
]);

var part392 = match("MESSAGE#362:608/2", "nwparser.p0", "%{fld1->} src=%{saddr}:%{p0}");

var part393 = match("MESSAGE#362:608/3_0", "nwparser.p0", "%{sport}:%{sinterface->} dst=%{p0}");

var part394 = match("MESSAGE#362:608/3_1", "nwparser.p0", "%{sport->} dst=%{p0}");

var select106 = linear_select([
	part393,
	part394,
]);

var part395 = match("MESSAGE#362:608/5_0", "nwparser.p0", "%{dport}:%{dinterface->} proto=%{protocol->} fw_action=\"%{fld2}\"");

var select107 = linear_select([
	part395,
	dup154,
	dup155,
]);

var all81 = all_match({
	processors: [
		part389,
		select105,
		part392,
		select106,
		dup153,
		select107,
	],
	on_success: processor_chain([
		dup1,
		dup44,
	]),
});

var msg365 = msg("608", all81);

var msg366 = msg("616", dup206);

var msg367 = msg("658", dup201);

var msg368 = msg("710", dup224);

var msg369 = msg("712:02", dup251);

var msg370 = msg("712", dup224);

var all82 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup202,
		dup100,
	],
	on_success: processor_chain([
		dup156,
	]),
});

var msg371 = msg("712:01", all82);

var select108 = linear_select([
	msg369,
	msg370,
	msg371,
]);

var part396 = match("MESSAGE#369:713:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} note=%{info}", processor_chain([
	dup5,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg372 = msg("713:01", part396);

var msg373 = msg("713:04", dup251);

var msg374 = msg("713:02", dup224);

var part397 = match("MESSAGE#372:713:03", "nwparser.payload", "msg=\"%{event_description}dropped\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{action}\" npcs=%{info}", processor_chain([
	dup5,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg375 = msg("713:03", part397);

var select109 = linear_select([
	msg372,
	msg373,
	msg374,
	msg375,
]);

var part398 = match("MESSAGE#373:760", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} note=%{info}", processor_chain([
	dup120,
	dup59,
	dup60,
	dup61,
	dup62,
	dup11,
	dup63,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg376 = msg("760", part398);

var part399 = match("MESSAGE#374:760:01/0", "nwparser.payload", "msg=\"%{event_description}dropped\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part400 = match("MESSAGE#374:760:01/4", "nwparser.p0", "%{action->} npcs=%{info}");

var all83 = all_match({
	processors: [
		part399,
		dup182,
		dup10,
		dup202,
		part400,
	],
	on_success: processor_chain([
		dup120,
		dup59,
		dup60,
		dup61,
		dup62,
		dup11,
		dup63,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg377 = msg("760:01", all83);

var select110 = linear_select([
	msg376,
	msg377,
]);

var msg378 = msg("766", dup228);

var msg379 = msg("860", dup228);

var msg380 = msg("860:01", dup229);

var select111 = linear_select([
	msg379,
	msg380,
]);

var part401 = match("MESSAGE#378:866/0", "nwparser.payload", "msg=\"%{msg}\" n=%{p0}");

var part402 = match("MESSAGE#378:866/1_0", "nwparser.p0", "%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"");

var part403 = match_copy("MESSAGE#378:866/1_1", "nwparser.p0", "ntype");

var select112 = linear_select([
	part402,
	part403,
]);

var all84 = all_match({
	processors: [
		part401,
		select112,
	],
	on_success: processor_chain([
		dup5,
		dup44,
	]),
});

var msg381 = msg("866", all84);

var msg382 = msg("866:01", dup229);

var select113 = linear_select([
	msg381,
	msg382,
]);

var msg383 = msg("867", dup228);

var msg384 = msg("867:01", dup229);

var select114 = linear_select([
	msg383,
	msg384,
]);

var part404 = match("MESSAGE#382:882", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup1,
]));

var msg385 = msg("882", part404);

var part405 = match("MESSAGE#383:882:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} npcs=%{info}", processor_chain([
	dup1,
]));

var msg386 = msg("882:01", part405);

var select115 = linear_select([
	msg385,
	msg386,
]);

var part406 = match("MESSAGE#384:888", "nwparser.payload", "msg=\"%{reason};%{action}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}", processor_chain([
	dup165,
]));

var msg387 = msg("888", part406);

var part407 = match("MESSAGE#385:888:01", "nwparser.payload", "msg=\"%{reason};%{action}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=%{fld3->} npcs=%{info}", processor_chain([
	dup165,
]));

var msg388 = msg("888:01", part407);

var select116 = linear_select([
	msg387,
	msg388,
]);

var all85 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup165,
	]),
});

var msg389 = msg("892", all85);

var msg390 = msg("904", dup228);

var msg391 = msg("905", dup228);

var msg392 = msg("906", dup228);

var msg393 = msg("907", dup228);

var part408 = match("MESSAGE#391:908/1_0", "nwparser.p0", "%{sinterface}:%{shost->} dst=%{p0}");

var select117 = linear_select([
	part408,
	dup167,
]);

var all86 = all_match({
	processors: [
		dup166,
		select117,
		dup168,
		dup223,
		dup169,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup78,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg394 = msg("908", all86);

var msg395 = msg("909", dup228);

var msg396 = msg("914", dup230);

var part409 = match("MESSAGE#394:931", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup72,
]));

var msg397 = msg("931", part409);

var msg398 = msg("657", dup230);

var all87 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg399 = msg("657:01", all87);

var select118 = linear_select([
	msg398,
	msg399,
]);

var msg400 = msg("403", dup209);

var msg401 = msg("534", dup184);

var msg402 = msg("994", dup231);

var part410 = match("MESSAGE#400:243", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} proto=%{protocol}", processor_chain([
	dup1,
	dup24,
]));

var msg403 = msg("243", part410);

var msg404 = msg("995", dup184);

var part411 = match("MESSAGE#402:997", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{fld3->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld4->} note=\"%{info}\"", processor_chain([
	dup1,
	dup59,
	dup61,
	dup62,
	dup11,
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg405 = msg("997", part411);

var msg406 = msg("998", dup231);

var part412 = match("MESSAGE#405:998:01", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup111,
	dup11,
]));

var msg407 = msg("998:01", part412);

var select119 = linear_select([
	msg406,
	msg407,
]);

var msg408 = msg("1110", dup232);

var msg409 = msg("565", dup232);

var part413 = match("MESSAGE#408:404", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup62,
]));

var msg410 = msg("404", part413);

var part414 = match("MESSAGE#409:267:01/1_0", "nwparser.p0", "%{daddr}:%{dport->} srcMac=%{p0}");

var select120 = linear_select([
	part414,
	dup58,
]);

var part415 = match("MESSAGE#409:267:01/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} note=\"%{fld3}\" fw_action=\"%{action}\"");

var all88 = all_match({
	processors: [
		dup87,
		select120,
		part415,
	],
	on_success: processor_chain([
		dup111,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg411 = msg("267:01", all88);

var part416 = match("MESSAGE#410:267", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}", processor_chain([
	dup1,
	dup62,
]));

var msg412 = msg("267", part416);

var select121 = linear_select([
	msg411,
	msg412,
]);

var part417 = match("MESSAGE#411:263", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} proto=%{protocol}", processor_chain([
	dup1,
	dup24,
]));

var msg413 = msg("263", part417);

var part418 = match("MESSAGE#412:264", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup109,
	dup11,
]));

var msg414 = msg("264", part418);

var msg415 = msg("412", dup209);

var part419 = match("MESSAGE#415:793", "nwparser.payload", "msg=\"%{msg}\" af_polid=%{fld1->} af_policy=\"%{fld2}\" af_type=\"%{fld3}\" af_service=\"%{fld4}\" af_action=\"%{fld5}\" n=%{fld6->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{shost->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{dhost}", processor_chain([
	dup1,
	dup24,
]));

var msg416 = msg("793", part419);

var part420 = match("MESSAGE#416:805", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} if=%{fld2->} ucastRx=%{fld3->} bcastRx=%{fld4->} bytesRx=%{rbytes->} ucastTx=%{fld5->} bcastTx=%{fld6->} bytesTx=%{sbytes}", processor_chain([
	dup1,
	dup24,
]));

var msg417 = msg("805", part420);

var part421 = match("MESSAGE#417:809", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup170,
	dup11,
]));

var msg418 = msg("809", part421);

var part422 = match("MESSAGE#418:809:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup170,
	dup11,
]));

var msg419 = msg("809:01", part422);

var select122 = linear_select([
	msg418,
	msg419,
]);

var msg420 = msg("935", dup230);

var msg421 = msg("614", dup233);

var part423 = match("MESSAGE#421:748/0", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var all89 = all_match({
	processors: [
		part423,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup66,
		dup44,
	]),
});

var msg422 = msg("748", all89);

var part424 = match("MESSAGE#422:794/0", "nwparser.payload", "msg=\"%{event_description}\" sid=%{sid->} spycat=%{fld1->} spypri=%{fld2->} pktdatId=%{fld3->} n=%{fld4->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part425 = match("MESSAGE#422:794/1_0", "nwparser.p0", "%{protocol}/%{fld5->} fw_action=\"%{p0}");

var select123 = linear_select([
	part425,
	dup118,
]);

var all90 = all_match({
	processors: [
		part424,
		select123,
		dup119,
	],
	on_success: processor_chain([
		dup171,
		dup44,
	]),
});

var msg423 = msg("794", all90);

var msg424 = msg("1086", dup233);

var part426 = match("MESSAGE#424:1430", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup171,
	dup44,
]));

var msg425 = msg("1430", part426);

var msg426 = msg("1149", dup233);

var msg427 = msg("1159", dup233);

var part427 = match("MESSAGE#427:1195", "nwparser.payload", "n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup171,
	dup44,
]));

var msg428 = msg("1195", part427);

var part428 = match("MESSAGE#428:1195:01", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1}", processor_chain([
	dup171,
	dup44,
]));

var msg429 = msg("1195:01", part428);

var select124 = linear_select([
	msg428,
	msg429,
]);

var part429 = match("MESSAGE#429:1226", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup44,
]));

var msg430 = msg("1226", part429);

var part430 = match("MESSAGE#430:1222", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport->} note=\"%{fld3}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup44,
]));

var msg431 = msg("1222", part430);

var part431 = match("MESSAGE#431:1154", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{shost->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{dhost}", processor_chain([
	dup1,
	dup24,
]));

var msg432 = msg("1154", part431);

var part432 = match("MESSAGE#432:1154:01/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{p0}");

var all91 = all_match({
	processors: [
		part432,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
		dup24,
	]),
});

var msg433 = msg("1154:01", all91);

var part433 = match("MESSAGE#433:1154:02", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=\"%{fld1}\" appid%{fld2->} catid=%{fld3->} sess=\"%{fld4}\" n=%{fld5->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup172,
	dup11,
]));

var msg434 = msg("1154:02", part433);

var part434 = match("MESSAGE#434:1154:03/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=\"%{fld1}\" appid=%{fld2->} catid=%{fld3->} n=%{fld4->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{p0}");

var part435 = match("MESSAGE#434:1154:03/1_0", "nwparser.p0", "%{dinterface}:%{dhost->} srcMac=%{p0}");

var select125 = linear_select([
	part435,
	dup79,
]);

var part436 = match("MESSAGE#434:1154:03/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"");

var all92 = all_match({
	processors: [
		part434,
		select125,
		part436,
	],
	on_success: processor_chain([
		dup172,
		dup11,
	]),
});

var msg435 = msg("1154:03", all92);

var select126 = linear_select([
	msg432,
	msg433,
	msg434,
	msg435,
]);

var part437 = match("MESSAGE#435:msg", "nwparser.payload", "msg=\"%{msg}\" src=%{stransaddr->} dst=%{dtransaddr->} %{result}", processor_chain([
	dup173,
]));

var msg436 = msg("msg", part437);

var part438 = match("MESSAGE#436:src", "nwparser.payload", "src=%{stransaddr->} dst=%{dtransaddr->} %{msg}", processor_chain([
	dup173,
]));

var msg437 = msg("src", part438);

var all93 = all_match({
	processors: [
		dup7,
		dup185,
		dup183,
		dup17,
		dup212,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg438 = msg("1235", all93);

var part439 = match("MESSAGE#438:1197/4", "nwparser.p0", "\"%{fld3->} Protocol:%{protocol}\" npcs=%{info}");

var all94 = all_match({
	processors: [
		dup7,
		dup185,
		dup10,
		dup202,
		part439,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg439 = msg("1197", all94);

var part440 = match("MESSAGE#439:1199/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3->} sess=%{fld1->} n=%{fld2->} src=%{p0}");

var all95 = all_match({
	processors: [
		part440,
		dup185,
		dup174,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg440 = msg("1199", all95);

var part441 = match("MESSAGE#440:1199:01", "nwparser.payload", "msg=\"Responder from country blocked: Responder IP:%{fld1}Country Name:%{location_country}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup175,
	dup11,
]));

var msg441 = msg("1199:01", part441);

var part442 = match("MESSAGE#441:1199:02", "nwparser.payload", "msg=\"Responder from country blocked: Responder IP:%{fld1}Country Name:%{location_country}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup175,
	dup11,
]));

var msg442 = msg("1199:02", part442);

var select127 = linear_select([
	msg440,
	msg441,
	msg442,
]);

var part443 = match("MESSAGE#442:1155/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} catid=%{fld3->} sess=%{fld4->} n=%{fld5->} src=%{p0}");

var all96 = all_match({
	processors: [
		part443,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg443 = msg("1155", all96);

var part444 = match("MESSAGE#443:1155:01", "nwparser.payload", "msg=\"%{action}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}", processor_chain([
	dup111,
]));

var msg444 = msg("1155:01", part444);

var select128 = linear_select([
	msg443,
	msg444,
]);

var all97 = all_match({
	processors: [
		dup176,
		dup213,
		dup174,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg445 = msg("1198", all97);

var all98 = all_match({
	processors: [
		dup7,
		dup185,
		dup174,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg446 = msg("714", all98);

var msg447 = msg("709", dup252);

var msg448 = msg("1005", dup252);

var msg449 = msg("1003", dup252);

var msg450 = msg("1007", dup253);

var part445 = match("MESSAGE#450:1008", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}::%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup109,
	dup11,
]));

var msg451 = msg("1008", part445);

var msg452 = msg("708", dup253);

var all99 = all_match({
	processors: [
		dup176,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg453 = msg("1201", all99);

var msg454 = msg("1201:01", dup253);

var select129 = linear_select([
	msg453,
	msg454,
]);

var msg455 = msg("654", dup234);

var msg456 = msg("670", dup234);

var msg457 = msg("884", dup253);

var part446 = match("MESSAGE#457:1153", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{protocol->} rcvd=%{rbytes->} note=\"%{info}\"", processor_chain([
	dup1,
]));

var msg458 = msg("1153", part446);

var part447 = match("MESSAGE#458:1153:01/1_0", "nwparser.p0", " app=%{fld1->} sess=%{fld2->} n=%{p0}");

var part448 = match("MESSAGE#458:1153:01/1_1", "nwparser.p0", " sess=%{fld2->} n=%{p0}");

var part449 = match("MESSAGE#458:1153:01/1_2", "nwparser.p0", " n=%{p0}");

var select130 = linear_select([
	part447,
	part448,
	part449,
]);

var part450 = match("MESSAGE#458:1153:01/2", "nwparser.p0", "%{fld3->} usr=\"%{username}\" src=%{p0}");

var part451 = match("MESSAGE#458:1153:01/3_0", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface}:%{shost->} dst= %{p0}");

var select131 = linear_select([
	part451,
	dup26,
]);

var part452 = match("MESSAGE#458:1153:01/4_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}srcMac= %{p0}");

var part453 = match("MESSAGE#458:1153:01/4_1", "nwparser.p0", "%{daddr}:%{dport}srcMac= %{p0}");

var part454 = match("MESSAGE#458:1153:01/4_2", "nwparser.p0", "%{daddr}srcMac= %{p0}");

var select132 = linear_select([
	part452,
	part453,
	part454,
]);

var part455 = match("MESSAGE#458:1153:01/5", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} %{p0}");

var part456 = match("MESSAGE#458:1153:01/6_0", "nwparser.p0", "sent=%{sbytes}rcvd=%{p0}");

var part457 = match("MESSAGE#458:1153:01/6_1", "nwparser.p0", "type=%{fld4->} icmpCode=%{fld5->} rcvd=%{p0}");

var part458 = match("MESSAGE#458:1153:01/6_2", "nwparser.p0", "rcvd=%{p0}");

var select133 = linear_select([
	part456,
	part457,
	part458,
]);

var all100 = all_match({
	processors: [
		dup54,
		select130,
		part450,
		select131,
		select132,
		part455,
		select133,
		dup123,
	],
	on_success: processor_chain([
		dup1,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg459 = msg("1153:01", all100);

var part459 = match("MESSAGE#459:1153:02/1_0", "nwparser.p0", "app=%{fld1->} n=%{fld2->} src=%{p0}");

var part460 = match("MESSAGE#459:1153:02/1_1", "nwparser.p0", "n=%{fld2->} src=%{p0}");

var select134 = linear_select([
	part459,
	part460,
]);

var part461 = match("MESSAGE#459:1153:02/2", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes}");

var all101 = all_match({
	processors: [
		dup82,
		select134,
		part461,
	],
	on_success: processor_chain([
		dup1,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var msg460 = msg("1153:02", all101);

var select135 = linear_select([
	msg458,
	msg459,
	msg460,
]);

var part462 = match("MESSAGE#460:1107", "nwparser.payload", "msg=\"%{msg}\"%{space}n=%{fld1}", processor_chain([
	dup1,
]));

var msg461 = msg("1107", part462);

var part463 = match("MESSAGE#461:1220/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{p0}");

var part464 = match("MESSAGE#461:1220/1_0", "nwparser.p0", "%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part465 = match("MESSAGE#461:1220/1_1", "nwparser.p0", "%{fld2}src=%{saddr}:%{sport->} dst= %{p0}");

var select136 = linear_select([
	part464,
	part465,
]);

var all102 = all_match({
	processors: [
		part463,
		select136,
		dup153,
		dup235,
		dup179,
	],
	on_success: processor_chain([
		dup165,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg462 = msg("1220", all102);

var all103 = all_match({
	processors: [
		dup149,
		dup235,
		dup179,
	],
	on_success: processor_chain([
		dup165,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg463 = msg("1230", all103);

var part466 = match("MESSAGE#463:1231", "nwparser.payload", "msg=\"%{msg}\"%{space}n=%{fld1->} note=\"%{info}\"", processor_chain([
	dup1,
]));

var msg464 = msg("1231", part466);

var part467 = match("MESSAGE#464:1233", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup175,
	dup11,
]));

var msg465 = msg("1233", part467);

var part468 = match("MESSAGE#465:1079/0", "nwparser.payload", "msg=\"User%{username}log%{p0}");

var part469 = match("MESSAGE#465:1079/1_0", "nwparser.p0", "in%{p0}");

var part470 = match("MESSAGE#465:1079/1_1", "nwparser.p0", "out%{p0}");

var select137 = linear_select([
	part469,
	part470,
]);

var part471 = match("MESSAGE#465:1079/2", "nwparser.p0", "\"%{p0}");

var part472 = match("MESSAGE#465:1079/3_0", "nwparser.p0", "dur=%{duration->} %{space}n=%{p0}");

var part473 = match("MESSAGE#465:1079/3_1", "nwparser.p0", "sess=\"%{fld2}\" n=%{p0}");

var select138 = linear_select([
	part472,
	part473,
	dup38,
]);

var part474 = match_copy("MESSAGE#465:1079/4", "nwparser.p0", "fld1");

var all104 = all_match({
	processors: [
		part468,
		select137,
		part471,
		select138,
		part474,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg466 = msg("1079", all104);

var part475 = match("MESSAGE#466:1079:01", "nwparser.payload", "msg=\"Client%{username}is assigned IP:%{hostip}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
]));

var msg467 = msg("1079:01", part475);

var part476 = match("MESSAGE#467:1079:02", "nwparser.payload", "msg=\"destination for %{daddr->} is not allowed by access control\" n=%{fld2}", processor_chain([
	dup1,
	dup11,
	setc("event_description","destination is not allowed by access control"),
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg468 = msg("1079:02", part476);

var part477 = match("MESSAGE#468:1079:03", "nwparser.payload", "msg=\"SSLVPN Client %{username->} matched device profile Default Device Profile for Windows\" n=%{fld2}", processor_chain([
	dup1,
	dup11,
	setc("event_description","SSLVPN Client matched device profile Default Device Profile for Windows"),
	dup18,
	dup19,
	dup20,
	dup21,
	dup22,
]));

var msg469 = msg("1079:03", part477);

var select139 = linear_select([
	msg466,
	msg467,
	msg468,
	msg469,
]);

var part478 = match("MESSAGE#469:1080/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} usr=\"%{username}\" src= %{p0}");

var part479 = match("MESSAGE#469:1080/1_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var select140 = linear_select([
	dup8,
	part479,
]);

var part480 = match("MESSAGE#469:1080/2_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto= %{p0}");

var select141 = linear_select([
	dup135,
	part480,
]);

var part481 = match_copy("MESSAGE#469:1080/3", "nwparser.p0", "protocol");

var all105 = all_match({
	processors: [
		part478,
		select140,
		select141,
		part481,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg470 = msg("1080", all105);

var part482 = match("MESSAGE#470:580", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg471 = msg("580", part482);

var part483 = match("MESSAGE#471:1369/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{p0}");

var all106 = all_match({
	processors: [
		part483,
		dup236,
		dup119,
	],
	on_success: processor_chain([
		dup78,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg472 = msg("1369", all106);

var all107 = all_match({
	processors: [
		dup149,
		dup223,
		dup152,
		dup236,
		dup119,
	],
	on_success: processor_chain([
		dup78,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg473 = msg("1370", all107);

var all108 = all_match({
	processors: [
		dup149,
		dup223,
		dup169,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup78,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg474 = msg("1371", all108);

var part484 = match("MESSAGE#474:1387/1_1", "nwparser.p0", " dst=%{p0}");

var select142 = linear_select([
	dup167,
	part484,
]);

var all109 = all_match({
	processors: [
		dup166,
		select142,
		dup168,
		dup223,
		dup169,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup165,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg475 = msg("1387", all109);

var part485 = match("MESSAGE#475:1391/0", "nwparser.payload", "pktdatId=%{fld1}pktdatNum=\"%{fld2}\" pktdatEnc=\"%{fld3}\" n=%{fld4}src=%{saddr}:%{p0}");

var part486 = match("MESSAGE#475:1391/1_0", "nwparser.p0", "%{sport}:%{sinterface}dst=%{p0}");

var part487 = match("MESSAGE#475:1391/1_1", "nwparser.p0", "%{sport}dst=%{p0}");

var select143 = linear_select([
	part486,
	part487,
]);

var part488 = match("MESSAGE#475:1391/3_0", "nwparser.p0", "%{dport}:%{dinterface}:%{dhost}");

var select144 = linear_select([
	part488,
	dup154,
	dup155,
]);

var all110 = all_match({
	processors: [
		part485,
		select143,
		dup153,
		select144,
	],
	on_success: processor_chain([
		dup1,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg476 = msg("1391", all110);

var part489 = match("MESSAGE#476:1253", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld1}appName=\"%{application}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg477 = msg("1253", part489);

var part490 = match("MESSAGE#477:1009", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2}note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg478 = msg("1009", part490);

var part491 = match("MESSAGE#478:910/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld2}appName=\"%{application}\" n=%{fld3}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{p0}");

var part492 = match("MESSAGE#478:910/1_0", "nwparser.p0", "%{dinterface}:%{dhost}srcMac=%{p0}");

var part493 = match("MESSAGE#478:910/1_1", "nwparser.p0", "%{dinterface}srcMac=%{p0}");

var select145 = linear_select([
	part492,
	part493,
]);

var part494 = match("MESSAGE#478:910/2", "nwparser.p0", "%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}fw_action=\"%{action}\"");

var all111 = all_match({
	processors: [
		part491,
		select145,
		part494,
	],
	on_success: processor_chain([
		dup5,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg479 = msg("910", all111);

var part495 = match("MESSAGE#479:m:01", "nwparser.payload", "m=%{id1}msg=\"%{event_description}\" n=%{fld2}if=%{interface}ucastRx=%{fld3}bcastRx=%{fld4}bytesRx=%{rbytes}ucastTx=%{fld5}bcastTx=%{fld6}bytesTx=%{sbytes}", processor_chain([
	dup1,
	dup62,
	dup18,
	dup88,
	dup20,
	dup22,
	dup44,
]));

var msg480 = msg("m:01", part495);

var part496 = match("MESSAGE#480:1011", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1}note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg481 = msg("1011", part496);

var part497 = match("MESSAGE#481:609", "nwparser.payload", "msg=\"%{event_description}\" sid=%{sid->} ipscat=\"%{fld3}\" ipspri=%{fld4->} pktdatId=%{fld5->} n=%{fld6->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup172,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg482 = msg("609", part497);

var msg483 = msg("796", dup237);

var part498 = match("MESSAGE#483:880", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup78,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg484 = msg("880", part498);

var part499 = match("MESSAGE#484:1309", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup165,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var msg485 = msg("1309", part499);

var msg486 = msg("1310", dup237);

var part500 = match("MESSAGE#486:1232/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{p0}");

var part501 = match("MESSAGE#486:1232/1_0", "nwparser.p0", "%{dinterface}:%{dhost->} note=\"%{p0}");

var part502 = match("MESSAGE#486:1232/1_1", "nwparser.p0", "%{dinterface->} note=\"%{p0}");

var select146 = linear_select([
	part501,
	part502,
]);

var part503 = match("MESSAGE#486:1232/2", "nwparser.p0", "%{info}\" fw_action=\"%{action}\"");

var all112 = all_match({
	processors: [
		part500,
		select146,
		part503,
	],
	on_success: processor_chain([
		dup1,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg487 = msg("1232", all112);

var part504 = match("MESSAGE#487:1447/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld1->} appName=\"%{application}\" n=%{fld2->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var all113 = all_match({
	processors: [
		part504,
		dup211,
		dup119,
	],
	on_success: processor_chain([
		dup165,
		dup62,
		dup18,
		dup88,
		dup20,
		dup21,
		dup22,
		dup44,
	]),
});

var msg488 = msg("1447", all113);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"10": msg9,
		"100": msg159,
		"1003": msg449,
		"1005": msg448,
		"1007": msg450,
		"1008": msg451,
		"1009": msg478,
		"101": msg160,
		"1011": msg481,
		"102": msg161,
		"103": msg162,
		"104": msg163,
		"105": msg164,
		"106": msg165,
		"107": msg166,
		"1079": select139,
		"108": msg167,
		"1080": msg470,
		"1086": msg424,
		"109": msg168,
		"11": msg10,
		"110": msg169,
		"1107": msg461,
		"111": select57,
		"1110": msg408,
		"112": msg172,
		"113": msg173,
		"114": msg174,
		"1149": msg426,
		"115": select58,
		"1153": select135,
		"1154": select126,
		"1155": select128,
		"1159": msg427,
		"116": msg177,
		"117": msg178,
		"118": msg179,
		"119": msg180,
		"1195": select124,
		"1197": msg439,
		"1198": msg445,
		"1199": select127,
		"12": select4,
		"120": msg181,
		"1201": select129,
		"121": msg182,
		"122": msg183,
		"1220": msg462,
		"1222": msg431,
		"1226": msg430,
		"123": msg184,
		"1230": msg463,
		"1231": msg464,
		"1232": msg487,
		"1233": msg465,
		"1235": msg438,
		"124": msg185,
		"125": msg186,
		"1253": msg477,
		"1254": msg187,
		"1256": msg188,
		"1257": msg189,
		"126": msg190,
		"127": msg191,
		"128": msg192,
		"129": msg193,
		"13": msg13,
		"130": msg194,
		"1309": msg485,
		"131": msg195,
		"1310": msg486,
		"132": msg196,
		"133": msg197,
		"134": msg198,
		"135": msg199,
		"136": msg200,
		"1369": msg472,
		"137": msg201,
		"1370": msg473,
		"1371": msg474,
		"138": msg202,
		"1387": msg475,
		"139": select59,
		"1391": msg476,
		"14": select7,
		"140": msg205,
		"141": msg206,
		"142": msg207,
		"143": msg208,
		"1430": msg425,
		"1431": msg209,
		"144": msg210,
		"1447": msg488,
		"145": msg211,
		"146": msg212,
		"147": msg213,
		"148": msg214,
		"1480": msg215,
		"149": msg216,
		"15": msg20,
		"150": msg217,
		"151": msg218,
		"152": msg219,
		"153": msg220,
		"154": msg221,
		"155": msg222,
		"156": msg223,
		"157": select60,
		"158": msg226,
		"159": msg227,
		"16": msg21,
		"160": msg228,
		"161": msg229,
		"162": msg230,
		"163": msg231,
		"164": msg232,
		"165": msg233,
		"166": msg234,
		"167": msg235,
		"168": msg236,
		"169": msg237,
		"17": msg22,
		"170": msg238,
		"171": select61,
		"172": select62,
		"173": msg245,
		"174": select63,
		"175": select64,
		"176": msg253,
		"177": msg254,
		"178": msg255,
		"179": msg256,
		"18": msg23,
		"180": select65,
		"181": select66,
		"19": msg24,
		"193": msg261,
		"194": msg262,
		"195": msg263,
		"196": select67,
		"199": msg266,
		"20": msg25,
		"200": msg267,
		"21": msg26,
		"22": msg27,
		"23": select10,
		"235": select68,
		"236": msg271,
		"237": msg272,
		"238": msg273,
		"239": msg274,
		"24": select11,
		"240": msg275,
		"241": select69,
		"242": msg278,
		"243": msg403,
		"25": msg34,
		"252": msg279,
		"255": msg280,
		"257": msg281,
		"26": msg35,
		"261": select72,
		"262": msg284,
		"263": msg413,
		"264": msg414,
		"267": select121,
		"27": msg36,
		"273": msg285,
		"28": select12,
		"29": select13,
		"30": select14,
		"31": select15,
		"32": select16,
		"328": msg286,
		"329": msg287,
		"33": select17,
		"34": msg52,
		"346": msg288,
		"35": select18,
		"350": msg289,
		"351": msg290,
		"352": msg291,
		"353": select73,
		"354": msg294,
		"355": select74,
		"356": msg297,
		"357": select75,
		"358": msg300,
		"36": select21,
		"37": select23,
		"371": select76,
		"372": msg303,
		"373": msg304,
		"38": select25,
		"39": msg67,
		"4": msg1,
		"40": msg68,
		"401": msg305,
		"402": msg306,
		"403": msg400,
		"404": msg410,
		"406": msg307,
		"41": select26,
		"412": msg415,
		"413": msg308,
		"414": msg309,
		"42": msg72,
		"427": msg156,
		"428": msg157,
		"43": msg73,
		"438": msg310,
		"439": msg311,
		"44": msg74,
		"440": msg312,
		"441": select77,
		"442": msg315,
		"446": msg316,
		"45": select27,
		"46": select28,
		"47": msg82,
		"477": msg317,
		"48": msg83,
		"49": msg84,
		"5": select2,
		"50": msg85,
		"509": msg318,
		"51": msg86,
		"52": msg87,
		"520": msg319,
		"522": select80,
		"523": msg323,
		"524": select83,
		"526": select86,
		"53": msg88,
		"534": msg401,
		"537": select101,
		"538": msg346,
		"549": msg347,
		"557": msg348,
		"558": msg349,
		"561": msg350,
		"562": msg351,
		"563": msg352,
		"565": msg409,
		"58": msg89,
		"580": msg471,
		"583": msg353,
		"597": select102,
		"598": select103,
		"6": select3,
		"60": msg90,
		"602": select104,
		"605": msg363,
		"606": msg364,
		"608": msg365,
		"609": msg482,
		"61": msg91,
		"614": msg421,
		"616": msg366,
		"62": msg92,
		"63": select29,
		"64": msg95,
		"65": msg96,
		"654": msg455,
		"657": select118,
		"658": msg367,
		"66": msg97,
		"67": select30,
		"670": msg456,
		"68": msg100,
		"69": msg101,
		"7": msg6,
		"70": select32,
		"708": msg452,
		"709": msg447,
		"710": msg368,
		"712": select108,
		"713": select109,
		"714": msg446,
		"72": select33,
		"73": msg106,
		"74": msg107,
		"748": msg422,
		"75": msg108,
		"76": msg109,
		"760": select110,
		"766": msg378,
		"77": msg110,
		"78": msg111,
		"79": msg112,
		"793": msg416,
		"794": msg423,
		"796": msg483,
		"8": msg7,
		"80": msg113,
		"805": msg417,
		"809": select122,
		"81": msg114,
		"82": select34,
		"83": select35,
		"84": msg122,
		"860": select111,
		"866": select113,
		"867": select114,
		"87": select37,
		"88": select38,
		"880": msg484,
		"882": select115,
		"884": msg457,
		"888": select116,
		"89": select40,
		"892": msg389,
		"9": msg8,
		"90": msg129,
		"904": msg390,
		"905": msg391,
		"906": msg392,
		"907": msg393,
		"908": msg394,
		"909": msg395,
		"91": msg130,
		"910": msg479,
		"914": msg396,
		"92": msg131,
		"93": msg132,
		"931": msg397,
		"935": msg420,
		"94": msg133,
		"95": msg134,
		"96": msg135,
		"97": select44,
		"98": select56,
		"986": msg155,
		"99": msg158,
		"994": msg402,
		"995": msg404,
		"997": msg405,
		"998": select119,
		"m": msg480,
		"msg": msg436,
		"src": msg437,
	}),
]);

var part505 = match("MESSAGE#14:14:01/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part506 = match("MESSAGE#14:14:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst= %{p0}");

var part507 = match("MESSAGE#14:14:01/1_1", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part508 = match("MESSAGE#14:14:01/2", "nwparser.p0", "%{daddr}:%{dport}:%{p0}");

var part509 = match("MESSAGE#28:23:01/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} %{p0}");

var part510 = match("MESSAGE#28:23:01/1_1", "nwparser.p0", "%{daddr->} %{p0}");

var part511 = match("MESSAGE#28:23:01/2", "nwparser.p0", "%{p0}");

var part512 = match("MESSAGE#38:29:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part513 = match("MESSAGE#38:29:01/1_1", "nwparser.p0", " %{saddr->} dst= %{p0}");

var part514 = match("MESSAGE#38:29:01/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} ");

var part515 = match("MESSAGE#38:29:01/2_1", "nwparser.p0", "%{daddr->} ");

var part516 = match("MESSAGE#40:30:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} src=%{p0}");

var part517 = match("MESSAGE#49:33:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{p0}");

var part518 = match("MESSAGE#52:35:01/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}");

var part519 = match_copy("MESSAGE#52:35:01/2_1", "nwparser.p0", "daddr");

var part520 = match("MESSAGE#54:36:01/1_0", "nwparser.p0", "app=%{fld51->} appName=\"%{application}\" n=%{p0}");

var part521 = match("MESSAGE#54:36:01/1_1", "nwparser.p0", "n=%{p0}");

var part522 = match("MESSAGE#54:36:01/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{p0}");

var part523 = match("MESSAGE#54:36:01/3_1", "nwparser.p0", "%{saddr->} %{p0}");

var part524 = match("MESSAGE#54:36:01/4", "nwparser.p0", "dst= %{p0}");

var part525 = match("MESSAGE#54:36:01/7_1", "nwparser.p0", "rule=%{rule}");

var part526 = match("MESSAGE#54:36:01/7_2", "nwparser.p0", "proto=%{protocol}");

var part527 = match("MESSAGE#55:36:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part528 = match("MESSAGE#55:36:02/1_1", "nwparser.p0", "%{saddr->} dst= %{p0}");

var part529 = match_copy("MESSAGE#55:36:02/6", "nwparser.p0", "info");

var part530 = match("MESSAGE#59:37:03/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} proto= %{p0}");

var part531 = match("MESSAGE#59:37:03/3_1", "nwparser.p0", "%{dinterface->} proto= %{p0}");

var part532 = match("MESSAGE#59:37:03/4", "nwparser.p0", "%{protocol->} npcs=%{info}");

var part533 = match("MESSAGE#62:38:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src= %{p0}");

var part534 = match("MESSAGE#63:38:02/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} type= %{p0}");

var part535 = match("MESSAGE#63:38:02/3_1", "nwparser.p0", "%{dinterface->} type= %{p0}");

var part536 = match("MESSAGE#64:38:03/0", "nwparser.payload", "msg=\"%{event_description}\"%{p0}");

var part537 = match("MESSAGE#64:38:03/1_0", "nwparser.p0", " app=%{fld2->} appName=\"%{application}\"%{p0}");

var part538 = match_copy("MESSAGE#64:38:03/1_1", "nwparser.p0", "p0");

var part539 = match("MESSAGE#64:38:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part540 = match("MESSAGE#64:38:03/3_1", "nwparser.p0", "%{daddr->} srcMac=%{p0}");

var part541 = match("MESSAGE#126:89:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{p0}");

var part542 = match("MESSAGE#135:97:01/0", "nwparser.payload", "n=%{fld1->} src= %{p0}");

var part543 = match("MESSAGE#135:97:01/6_0", "nwparser.p0", "result=%{result->} dstname=%{p0}");

var part544 = match("MESSAGE#135:97:01/6_1", "nwparser.p0", "dstname=%{p0}");

var part545 = match("MESSAGE#137:97:03/0", "nwparser.payload", "sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part546 = match("MESSAGE#141:97:07/1_1", "nwparser.p0", "%{dinterface->} srcMac=%{p0}");

var part547 = match("MESSAGE#147:98:01/6_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} %{p0}");

var part548 = match("MESSAGE#147:98:01/7_4", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes}");

var part549 = match("MESSAGE#148:98:06/0", "nwparser.payload", "msg=\"%{event_description}\" %{p0}");

var part550 = match("MESSAGE#148:98:06/5_0", "nwparser.p0", "%{sinterface}:%{shost->} dst= %{p0}");

var part551 = match("MESSAGE#148:98:06/5_1", "nwparser.p0", "%{sinterface->} dst= %{p0}");

var part552 = match("MESSAGE#148:98:06/7_2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{p0}");

var part553 = match("MESSAGE#148:98:06/9_3", "nwparser.p0", "sent=%{sbytes}");

var part554 = match("MESSAGE#155:428/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part555 = match("MESSAGE#240:171:03/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} npcs= %{p0}");

var part556 = match("MESSAGE#240:171:03/3_1", "nwparser.p0", "%{dinterface->} npcs= %{p0}");

var part557 = match("MESSAGE#240:171:03/4", "nwparser.p0", "%{info}");

var part558 = match("MESSAGE#256:180:01/3_0", "nwparser.p0", "%{dinterface}:%{dhost->} note= %{p0}");

var part559 = match("MESSAGE#256:180:01/3_1", "nwparser.p0", "%{dinterface->} note= %{p0}");

var part560 = match("MESSAGE#256:180:01/4", "nwparser.p0", "\"%{fld3}\" npcs=%{info}");

var part561 = match("MESSAGE#260:194/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} sport=%{sport->} dport=%{dport->} %{p0}");

var part562 = match("MESSAGE#260:194/1_1", "nwparser.p0", "rcvd=%{rbytes}");

var part563 = match("MESSAGE#262:196/1_0", "nwparser.p0", "sent=%{sbytes->} cmd=%{p0}");

var part564 = match("MESSAGE#262:196/1_1", "nwparser.p0", "rcvd=%{rbytes->} cmd=%{p0}");

var part565 = match_copy("MESSAGE#262:196/2", "nwparser.p0", "method");

var part566 = match("MESSAGE#280:261:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var part567 = match("MESSAGE#283:273/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{p0}");

var part568 = match("MESSAGE#302:401/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} %{p0}");

var part569 = match("MESSAGE#302:401/1_0", "nwparser.p0", "dstname=%{name}");

var part570 = match_copy("MESSAGE#302:401/1_1", "nwparser.p0", "space");

var part571 = match("MESSAGE#313:446/3_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=\"%{p0}");

var part572 = match("MESSAGE#313:446/3_1", "nwparser.p0", "%{protocol->} fw_action=\"%{p0}");

var part573 = match("MESSAGE#313:446/4", "nwparser.p0", "%{action}\"");

var part574 = match("MESSAGE#318:522:01/4", "nwparser.p0", "proto=%{protocol->} npcs=%{info}");

var part575 = match("MESSAGE#330:537:01/0", "nwparser.payload", "msg=\"%{action}\" f=%{fld1->} n=%{fld2->} src= %{p0}");

var part576 = match_copy("MESSAGE#330:537:01/5_1", "nwparser.p0", "rbytes");

var part577 = match("MESSAGE#332:537:08/1_0", "nwparser.p0", " app=%{fld51->} appName=\"%{application}\"n=%{p0}");

var part578 = match("MESSAGE#332:537:08/1_1", "nwparser.p0", " app=%{fld51->} sess=\"%{fld4}\" n=%{p0}");

var part579 = match("MESSAGE#332:537:08/1_2", "nwparser.p0", " app=%{fld51}n=%{p0}");

var part580 = match("MESSAGE#332:537:08/2_0", "nwparser.p0", "%{fld1->} usr=\"%{username}\"src=%{p0}");

var part581 = match("MESSAGE#332:537:08/2_1", "nwparser.p0", "%{fld1}src=%{p0}");

var part582 = match("MESSAGE#332:537:08/6_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{p0}");

var part583 = match("MESSAGE#332:537:08/6_1", "nwparser.p0", "%{sbytes->} spkt=%{p0}");

var part584 = match("MESSAGE#332:537:08/7_1", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} cdur=%{fld7}");

var part585 = match("MESSAGE#332:537:08/7_3", "nwparser.p0", "%{fld3->} cdur=%{fld7}");

var part586 = match_copy("MESSAGE#332:537:08/7_4", "nwparser.p0", "fld3");

var part587 = match("MESSAGE#336:537:04/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part588 = match("MESSAGE#336:537:04/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto= %{p0}");

var part589 = match("MESSAGE#336:537:04/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto= %{p0}");

var part590 = match("MESSAGE#336:537:04/3_2", "nwparser.p0", "%{daddr->} proto= %{p0}");

var part591 = match("MESSAGE#338:537:10/1_0", "nwparser.p0", "usr=\"%{username}\" %{p0}");

var part592 = match("MESSAGE#338:537:10/2", "nwparser.p0", "src=%{p0}");

var part593 = match("MESSAGE#338:537:10/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part594 = match("MESSAGE#338:537:10/3_1", "nwparser.p0", "%{saddr->} dst=%{p0}");

var part595 = match("MESSAGE#338:537:10/6_0", "nwparser.p0", "npcs=%{info}");

var part596 = match("MESSAGE#338:537:10/6_1", "nwparser.p0", "cdur=%{fld12}");

var part597 = match("MESSAGE#355:598:01/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{daddr}:%{dport}:%{p0}");

var part598 = match("MESSAGE#361:606/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{p0}");

var part599 = match("MESSAGE#361:606/1_0", "nwparser.p0", "%{dport}:%{dinterface->} srcMac=%{p0}");

var part600 = match("MESSAGE#361:606/1_1", "nwparser.p0", "%{dport->} srcMac=%{p0}");

var part601 = match("MESSAGE#361:606/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr}proto=%{p0}");

var part602 = match("MESSAGE#362:608/4", "nwparser.p0", "%{daddr}:%{p0}");

var part603 = match("MESSAGE#362:608/5_1", "nwparser.p0", "%{dport}:%{dinterface}");

var part604 = match_copy("MESSAGE#362:608/5_2", "nwparser.p0", "dport");

var part605 = match("MESSAGE#366:712:02/0", "nwparser.payload", "msg=\"%{action}\" %{p0}");

var part606 = match("MESSAGE#366:712:02/1_0", "nwparser.p0", "app=%{fld21->} appName=\"%{application}\" n=%{p0}");

var part607 = match("MESSAGE#366:712:02/2", "nwparser.p0", "%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part608 = match("MESSAGE#366:712:02/3_0", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var part609 = match("MESSAGE#366:712:02/3_1", "nwparser.p0", "%{smacaddr->} proto=%{p0}");

var part610 = match("MESSAGE#366:712:02/4_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=%{p0}");

var part611 = match("MESSAGE#366:712:02/4_1", "nwparser.p0", "%{protocol->} fw_action=%{p0}");

var part612 = match_copy("MESSAGE#366:712:02/5", "nwparser.p0", "fld51");

var part613 = match("MESSAGE#391:908/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{p0}");

var part614 = match("MESSAGE#391:908/1_1", "nwparser.p0", "%{sinterface->} dst=%{p0}");

var part615 = match("MESSAGE#391:908/2", "nwparser.p0", "%{} %{daddr}:%{p0}");

var part616 = match("MESSAGE#391:908/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var part617 = match("MESSAGE#439:1199/2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} npcs=%{info}");

var part618 = match("MESSAGE#444:1198/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part619 = match("MESSAGE#461:1220/3_0", "nwparser.p0", "%{dport}:%{dinterface->} note=%{p0}");

var part620 = match("MESSAGE#461:1220/3_1", "nwparser.p0", "%{dport->} note=%{p0}");

var part621 = match("MESSAGE#461:1220/4", "nwparser.p0", "%{}\"%{info}\" fw_action=\"%{action}\"");

var part622 = match("MESSAGE#471:1369/1_0", "nwparser.p0", "%{protocol}/%{fld3}fw_action=\"%{p0}");

var part623 = match("MESSAGE#471:1369/1_1", "nwparser.p0", "%{protocol}fw_action=\"%{p0}");

var select147 = linear_select([
	dup8,
	dup9,
]);

var select148 = linear_select([
	dup15,
	dup16,
]);

var part624 = match("MESSAGE#403:24:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var select149 = linear_select([
	dup26,
	dup27,
]);

var select150 = linear_select([
	dup28,
	dup29,
]);

var select151 = linear_select([
	dup35,
	dup36,
]);

var select152 = linear_select([
	dup37,
	dup38,
]);

var select153 = linear_select([
	dup39,
	dup40,
]);

var select154 = linear_select([
	dup26,
	dup46,
]);

var select155 = linear_select([
	dup48,
	dup49,
]);

var select156 = linear_select([
	dup52,
	dup53,
]);

var select157 = linear_select([
	dup55,
	dup56,
]);

var select158 = linear_select([
	dup57,
	dup58,
]);

var part625 = match("MESSAGE#116:82:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup70,
]));

var part626 = match("MESSAGE#118:83:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup5,
]));

var select159 = linear_select([
	dup75,
	dup76,
]);

var select160 = linear_select([
	dup83,
	dup84,
]);

var part627 = match("MESSAGE#168:111:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=%{shost}", processor_chain([
	dup1,
]));

var select161 = linear_select([
	dup94,
	dup95,
]);

var part628 = match("MESSAGE#253:178", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup5,
]));

var select162 = linear_select([
	dup98,
	dup99,
]);

var select163 = linear_select([
	dup86,
	dup102,
]);

var select164 = linear_select([
	dup103,
	dup104,
]);

var part629 = match("MESSAGE#277:252", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup93,
]));

var part630 = match("MESSAGE#293:355", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup93,
]));

var part631 = match("MESSAGE#295:356", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup1,
]));

var part632 = match("MESSAGE#298:358", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var part633 = match("MESSAGE#414:371:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var select165 = linear_select([
	dup114,
	dup115,
]);

var select166 = linear_select([
	dup117,
	dup118,
]);

var select167 = linear_select([
	dup43,
	dup42,
]);

var select168 = linear_select([
	dup8,
	dup27,
]);

var select169 = linear_select([
	dup8,
	dup26,
	dup46,
]);

var select170 = linear_select([
	dup80,
	dup15,
	dup16,
]);

var select171 = linear_select([
	dup124,
	dup125,
	dup126,
	dup38,
]);

var select172 = linear_select([
	dup127,
	dup128,
]);

var select173 = linear_select([
	dup129,
	dup130,
]);

var select174 = linear_select([
	dup135,
	dup136,
	dup137,
]);

var select175 = linear_select([
	dup138,
	dup56,
]);

var select176 = linear_select([
	dup140,
	dup141,
]);

var select177 = linear_select([
	dup142,
	dup143,
]);

var select178 = linear_select([
	dup150,
	dup151,
]);

var part634 = match("MESSAGE#365:710", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup156,
]));

var select179 = linear_select([
	dup158,
	dup38,
]);

var select180 = linear_select([
	dup160,
	dup161,
]);

var select181 = linear_select([
	dup162,
	dup163,
]);

var part635 = match("MESSAGE#375:766", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup5,
]));

var part636 = match("MESSAGE#377:860:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{ntype}", processor_chain([
	dup5,
]));

var part637 = match("MESSAGE#393:914", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{host->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{shost}", processor_chain([
	dup5,
	dup24,
]));

var part638 = match("MESSAGE#399:994", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup24,
]));

var part639 = match("MESSAGE#406:1110", "nwparser.payload", "msg=\"%{msg}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
	dup24,
]));

var part640 = match("MESSAGE#420:614", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup171,
	dup44,
]));

var part641 = match("MESSAGE#454:654", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2}", processor_chain([
	dup1,
]));

var select182 = linear_select([
	dup177,
	dup178,
]);

var select183 = linear_select([
	dup180,
	dup181,
]);

var part642 = match("MESSAGE#482:796", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup62,
	dup18,
	dup88,
	dup20,
	dup21,
	dup22,
	dup44,
]));

var all114 = all_match({
	processors: [
		dup32,
		dup185,
		dup186,
	],
	on_success: processor_chain([
		dup31,
	]),
});

var all115 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup91,
	]),
});

var all116 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup67,
	]),
});

var all117 = all_match({
	processors: [
		dup101,
		dup203,
	],
	on_success: processor_chain([
		dup67,
	]),
});

var all118 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup106,
	]),
});

var all119 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup31,
	]),
});

var all120 = all_match({
	processors: [
		dup32,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var all121 = all_match({
	processors: [
		dup108,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup109,
	]),
});

var all122 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup112,
	]),
});

var all123 = all_match({
	processors: [
		dup113,
		dup210,
	],
	on_success: processor_chain([
		dup93,
	]),
});

var all124 = all_match({
	processors: [
		dup110,
		dup185,
		dup187,
	],
	on_success: processor_chain([
		dup116,
	]),
});

var all125 = all_match({
	processors: [
		dup51,
		dup189,
		dup41,
		dup187,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var all126 = all_match({
	processors: [
		dup73,
		dup185,
		dup183,
		dup43,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var all127 = all_match({
	processors: [
		dup157,
		dup225,
		dup159,
		dup226,
		dup227,
		dup164,
	],
	on_success: processor_chain([
		dup156,
		dup59,
		dup60,
		dup61,
		dup62,
		dup44,
		dup63,
		dup18,
		dup19,
		dup20,
		dup21,
		dup22,
	]),
});

var all128 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup202,
		dup100,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var all129 = all_match({
	processors: [
		dup7,
		dup182,
		dup10,
		dup200,
		dup96,
	],
	on_success: processor_chain([
		dup1,
	]),
});
