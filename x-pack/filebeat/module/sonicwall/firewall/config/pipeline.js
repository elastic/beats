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

var dup10 = match("MESSAGE#14:14:01/2", "nwparser.p0", "%{} %{p0}");

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

var dup17 = setf("hostip","hhostip");

var dup18 = setf("id","hid");

var dup19 = setf("serial_number","hserial_number");

var dup20 = setf("category","hcategory");

var dup21 = setf("severity","hseverity");

var dup22 = setc("eventcategory","1805010000");

var dup23 = call({
	dest: "nwparser.msg",
	fn: RMQ,
	args: [
		field("msg"),
	],
});

var dup24 = setc("eventcategory","1302000000");

var dup25 = match("MESSAGE#38:29:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var dup26 = match("MESSAGE#38:29:01/1_1", "nwparser.p0", " %{saddr->} dst= %{p0}");

var dup27 = match("MESSAGE#38:29:01/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} ");

var dup28 = match("MESSAGE#38:29:01/3_1", "nwparser.p0", "%{daddr->} ");

var dup29 = setc("eventcategory","1401050100");

var dup30 = setc("eventcategory","1401030000");

var dup31 = match("MESSAGE#40:30:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} src=%{p0}");

var dup32 = setc("eventcategory","1301020000");

var dup33 = match("MESSAGE#49:33:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{p0}");

var dup34 = match("MESSAGE#54:36:01/2_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{p0}");

var dup35 = match("MESSAGE#54:36:01/2_1", "nwparser.p0", "%{saddr->} %{p0}");

var dup36 = match("MESSAGE#54:36:01/3", "nwparser.p0", "%{}dst= %{p0}");

var dup37 = date_time({
	dest: "event_time",
	args: ["date","time"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup38 = match("MESSAGE#55:36:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup39 = match("MESSAGE#55:36:02/1_1", "nwparser.p0", "%{saddr->} dst= %{p0}");

var dup40 = match("MESSAGE#57:37:01/1_1", "nwparser.p0", "n=%{fld1->} src=%{p0}");

var dup41 = match("MESSAGE#59:37:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto= %{p0}");

var dup42 = match("MESSAGE#59:37:03/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto= %{p0}");

var dup43 = match("MESSAGE#59:37:03/4", "nwparser.p0", "%{} %{protocol->} npcs=%{info}");

var dup44 = match("MESSAGE#62:38:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src= %{p0}");

var dup45 = match("MESSAGE#62:38:01/5_1", "nwparser.p0", "rule=%{rule->} ");

var dup46 = match("MESSAGE#63:38:02/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} type= %{p0}");

var dup47 = match("MESSAGE#63:38:02/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} type= %{p0}");

var dup48 = match("MESSAGE#64:38:03/0", "nwparser.payload", "msg=\"%{p0}");

var dup49 = match("MESSAGE#64:38:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var dup50 = match("MESSAGE#64:38:03/3_1", "nwparser.p0", "%{daddr->} srcMac=%{p0}");

var dup51 = setc("ec_subject","NetworkComm");

var dup52 = setc("ec_activity","Deny");

var dup53 = setc("ec_theme","Communication");

var dup54 = setf("msg","$MSG");

var dup55 = setc("action","dropped");

var dup56 = setc("eventcategory","1608010000");

var dup57 = setc("eventcategory","1302010000");

var dup58 = setc("eventcategory","1301000000");

var dup59 = setc("eventcategory","1001000000");

var dup60 = setc("eventcategory","1003030000");

var dup61 = setc("eventcategory","1003050000");

var dup62 = setc("eventcategory","1103000000");

var dup63 = setc("eventcategory","1603110000");

var dup64 = setc("eventcategory","1605020000");

var dup65 = match("MESSAGE#135:97:01/0", "nwparser.payload", "n=%{fld1->} src= %{p0}");

var dup66 = match("MESSAGE#135:97:01/7_1", "nwparser.p0", "dstname=%{name->} ");

var dup67 = match("MESSAGE#137:97:03/0", "nwparser.payload", "sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup68 = match("MESSAGE#140:97:06/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{p0}");

var dup69 = match("MESSAGE#140:97:06/1_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}dst=%{p0}");

var dup70 = setc("eventcategory","1801000000");

var dup71 = match("MESSAGE#145:98/2_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} %{p0}");

var dup72 = match("MESSAGE#145:98/3_0", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var dup73 = match("MESSAGE#147:98:01/4_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{p0}");

var dup74 = match("MESSAGE#147:98:01/4_2", "nwparser.p0", "%{saddr}dst=%{p0}");

var dup75 = match("MESSAGE#147:98:01/6_1", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} %{p0}");

var dup76 = match("MESSAGE#147:98:01/6_2", "nwparser.p0", " %{daddr->} %{p0}");

var dup77 = match("MESSAGE#148:98:06/5_2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{p0}");

var dup78 = match("MESSAGE#148:98:06/5_3", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var dup79 = match("MESSAGE#149:98:02/4", "nwparser.p0", "%{}proto=%{protocol}");

var dup80 = match("MESSAGE#154:427/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{p0}");

var dup81 = match("MESSAGE#155:428/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var dup82 = setf("id","hfld1");

var dup83 = setc("eventcategory","1001020309");

var dup84 = setc("eventcategory","1303000000");

var dup85 = setc("eventcategory","1801010100");

var dup86 = setc("eventcategory","1604010000");

var dup87 = setc("eventcategory","1002020000");

var dup88 = match("MESSAGE#240:171:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} npcs= %{p0}");

var dup89 = match("MESSAGE#240:171:03/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} npcs= %{p0}");

var dup90 = match("MESSAGE#240:171:03/4", "nwparser.p0", "%{} %{info}");

var dup91 = setc("eventcategory","1001010000");

var dup92 = match("MESSAGE#256:180:01/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} note= %{p0}");

var dup93 = match("MESSAGE#256:180:01/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} note= %{p0}");

var dup94 = match("MESSAGE#256:180:01/4", "nwparser.p0", "%{}\"%{fld3}\" npcs=%{info}");

var dup95 = match("MESSAGE#260:194/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} sport=%{sport->} dport=%{dport->} %{p0}");

var dup96 = match("MESSAGE#260:194/1_0", "nwparser.p0", "sent=%{sbytes->} ");

var dup97 = match("MESSAGE#260:194/1_1", "nwparser.p0", " rcvd=%{rbytes}");

var dup98 = match("MESSAGE#262:196/1_0", "nwparser.p0", "sent=%{sbytes->} cmd=%{p0}");

var dup99 = match("MESSAGE#262:196/2", "nwparser.p0", "%{method}");

var dup100 = setc("eventcategory","1401060000");

var dup101 = setc("eventcategory","1804000000");

var dup102 = match("MESSAGE#280:261:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var dup103 = setc("eventcategory","1401070000");

var dup104 = match("MESSAGE#283:273/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{p0}");

var dup105 = setc("eventcategory","1801030000");

var dup106 = setc("eventcategory","1402020300");

var dup107 = match("MESSAGE#302:401/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} %{p0}");

var dup108 = match("MESSAGE#302:401/1_1", "nwparser.p0", " %{space}");

var dup109 = setc("eventcategory","1402000000");

var dup110 = match("MESSAGE#313:446/3_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=\"%{p0}");

var dup111 = match("MESSAGE#313:446/3_1", "nwparser.p0", "%{protocol->} fw_action=\"%{p0}");

var dup112 = match("MESSAGE#313:446/4", "nwparser.p0", "%{action}\"");

var dup113 = setc("eventcategory","1803020000");

var dup114 = match("MESSAGE#318:522:01/4", "nwparser.p0", "%{}proto=%{protocol->} npcs=%{info}");

var dup115 = match("MESSAGE#321:524/5_0", "nwparser.p0", "proto=%{protocol->} ");

var dup116 = match("MESSAGE#330:537:01/0", "nwparser.payload", "msg=\"%{action}\" f=%{fld1->} n=%{fld2->} src= %{p0}");

var dup117 = match("MESSAGE#332:537:08/0_0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld51->} appName=\"%{application}\"n=%{p0}");

var dup118 = match("MESSAGE#332:537:08/0_1", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld51->} sess=\"%{fld4}\" n=%{p0}");

var dup119 = match("MESSAGE#332:537:08/0_2", "nwparser.payload", " msg=\"%{event_description}\" app=%{fld51}n=%{p0}");

var dup120 = match("MESSAGE#332:537:08/0_3", "nwparser.payload", "msg=\"%{event_description}\"n=%{p0}");

var dup121 = match("MESSAGE#332:537:08/1_0", "nwparser.p0", "%{fld1->} usr=\"%{username}\"src=%{p0}");

var dup122 = match("MESSAGE#332:537:08/1_1", "nwparser.p0", "%{fld1}src=%{p0}");

var dup123 = match("MESSAGE#332:537:08/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{p0}");

var dup124 = match("MESSAGE#332:537:08/5_0", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var dup125 = match("MESSAGE#332:537:08/5_1", "nwparser.p0", " proto=%{protocol->} sent=%{p0}");

var dup126 = match("MESSAGE#333:537:09/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{p0}");

var dup127 = match("MESSAGE#333:537:09/5_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} %{p0}");

var dup128 = match("MESSAGE#333:537:09/5_1", "nwparser.p0", "%{sbytes->} %{p0}");

var dup129 = match("MESSAGE#333:537:09/6_0", "nwparser.p0", " spkt=%{fld3->} cdur=%{fld7->} fw_action=\"%{action}\"");

var dup130 = match("MESSAGE#333:537:09/6_1", "nwparser.p0", "spkt=%{fld3->} rpkt=%{fld6->} cdur=%{fld7->} ");

var dup131 = match("MESSAGE#333:537:09/6_2", "nwparser.p0", "spkt=%{fld3->} cdur=%{fld7->} ");

var dup132 = match("MESSAGE#333:537:09/6_3", "nwparser.p0", " spkt=%{fld3}");

var dup133 = match("MESSAGE#336:537:04/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var dup134 = match("MESSAGE#336:537:04/3_2", "nwparser.p0", "%{daddr->} proto= %{p0}");

var dup135 = match("MESSAGE#338:537:10/1_0", "nwparser.p0", "%{fld2->} usr=\"%{username}\" %{p0}");

var dup136 = match("MESSAGE#338:537:10/1_1", "nwparser.p0", "%{fld2->} %{p0}");

var dup137 = match("MESSAGE#338:537:10/2", "nwparser.p0", "%{}src=%{p0}");

var dup138 = match("MESSAGE#338:537:10/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var dup139 = match("MESSAGE#338:537:10/3_1", "nwparser.p0", "%{saddr->} dst=%{p0}");

var dup140 = match("MESSAGE#338:537:10/6_0", "nwparser.p0", "npcs=%{info->} ");

var dup141 = match("MESSAGE#338:537:10/6_1", "nwparser.p0", "cdur=%{fld12->} ");

var dup142 = setc("event_description","Connection Closed");

var dup143 = setc("eventcategory","1801020000");

var dup144 = setc("ec_activity","Permit");

var dup145 = setc("action","allowed");

var dup146 = match("MESSAGE#355:598:01/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var dup147 = match("MESSAGE#361:606/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var dup148 = match("MESSAGE#361:606/1_1", "nwparser.p0", "%{daddr}:%{dport->} srcMac=%{p0}");

var dup149 = match("MESSAGE#361:606/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr}proto=%{p0}");

var dup150 = setc("eventcategory","1001030500");

var dup151 = match("MESSAGE#366:712:02/0", "nwparser.payload", "msg=\"%{action}\" %{p0}");

var dup152 = match("MESSAGE#366:712:02/1_0", "nwparser.p0", "app=%{fld21->} appName=\"%{application}\" n=%{fld1->} src=%{p0}");

var dup153 = match("MESSAGE#366:712:02/2", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var dup154 = match("MESSAGE#366:712:02/3_0", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var dup155 = match("MESSAGE#366:712:02/3_1", "nwparser.p0", "%{smacaddr->} proto=%{p0}");

var dup156 = match("MESSAGE#366:712:02/4_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=%{p0}");

var dup157 = match("MESSAGE#366:712:02/4_1", "nwparser.p0", "%{protocol->} fw_action=%{p0}");

var dup158 = match("MESSAGE#366:712:02/5", "nwparser.p0", "%{fld51}");

var dup159 = setc("eventcategory","1801010000");

var dup160 = match("MESSAGE#391:908/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{p0}");

var dup161 = match("MESSAGE#391:908/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var dup162 = setc("eventcategory","1003010000");

var dup163 = setc("eventcategory","1609000000");

var dup164 = setc("eventcategory","1204000000");

var dup165 = setc("eventcategory","1602000000");

var dup166 = match("MESSAGE#439:1199/2", "nwparser.p0", "%{} %{daddr}:%{dport}:%{dinterface->} npcs=%{info}");

var dup167 = setc("eventcategory","1803000000");

var dup168 = match("MESSAGE#444:1198/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var dup169 = match("MESSAGE#461:1220/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} note=%{p0}");

var dup170 = match("MESSAGE#461:1220/3_1", "nwparser.p0", "%{daddr}:%{dport->} note=%{p0}");

var dup171 = match("MESSAGE#461:1220/4", "nwparser.p0", "%{}\"%{info}\" fw_action=\"%{action}\"");

var dup172 = match("MESSAGE#471:1369/1_0", "nwparser.p0", "%{protocol}/%{fld3}fw_action=\"%{p0}");

var dup173 = match("MESSAGE#471:1369/1_1", "nwparser.p0", "%{protocol}fw_action=\"%{p0}");

var dup174 = linear_select([
	dup8,
	dup9,
]);

var dup175 = linear_select([
	dup15,
	dup16,
]);

var dup176 = match("MESSAGE#403:24:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var dup177 = linear_select([
	dup25,
	dup26,
]);

var dup178 = linear_select([
	dup27,
	dup28,
]);

var dup179 = linear_select([
	dup34,
	dup35,
]);

var dup180 = linear_select([
	dup25,
	dup39,
]);

var dup181 = linear_select([
	dup41,
	dup42,
]);

var dup182 = linear_select([
	dup46,
	dup47,
]);

var dup183 = linear_select([
	dup49,
	dup50,
]);

var dup184 = match("MESSAGE#116:82:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup62,
]));

var dup185 = match("MESSAGE#118:83:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup5,
]));

var dup186 = linear_select([
	dup71,
	dup75,
	dup76,
]);

var dup187 = linear_select([
	dup8,
	dup25,
]);

var dup188 = match("MESSAGE#168:111:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=%{shost}", processor_chain([
	dup1,
]));

var dup189 = linear_select([
	dup88,
	dup89,
]);

var dup190 = match("MESSAGE#253:178", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup5,
]));

var dup191 = linear_select([
	dup92,
	dup93,
]);

var dup192 = linear_select([
	dup96,
	dup97,
]);

var dup193 = match("MESSAGE#277:252", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup87,
]));

var dup194 = match("MESSAGE#293:355", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup87,
]));

var dup195 = match("MESSAGE#295:356", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup1,
]));

var dup196 = match("MESSAGE#298:358", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var dup197 = match("MESSAGE#414:371:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var dup198 = linear_select([
	dup66,
	dup108,
]);

var dup199 = linear_select([
	dup110,
	dup111,
]);

var dup200 = linear_select([
	dup115,
	dup45,
]);

var dup201 = linear_select([
	dup8,
	dup26,
]);

var dup202 = linear_select([
	dup8,
	dup25,
	dup39,
]);

var dup203 = linear_select([
	dup71,
	dup15,
	dup16,
]);

var dup204 = linear_select([
	dup121,
	dup122,
]);

var dup205 = linear_select([
	dup68,
	dup69,
	dup74,
]);

var dup206 = linear_select([
	dup127,
	dup128,
]);

var dup207 = linear_select([
	dup41,
	dup42,
	dup134,
]);

var dup208 = linear_select([
	dup135,
	dup136,
]);

var dup209 = linear_select([
	dup138,
	dup139,
]);

var dup210 = linear_select([
	dup140,
	dup141,
]);

var dup211 = linear_select([
	dup49,
	dup148,
]);

var dup212 = match("MESSAGE#365:710", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup150,
]));

var dup213 = linear_select([
	dup152,
	dup40,
]);

var dup214 = linear_select([
	dup154,
	dup155,
]);

var dup215 = linear_select([
	dup156,
	dup157,
]);

var dup216 = match("MESSAGE#375:766", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup5,
]));

var dup217 = match("MESSAGE#377:860:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{ntype}", processor_chain([
	dup5,
]));

var dup218 = match("MESSAGE#393:914", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{host->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{shost}", processor_chain([
	dup5,
	dup23,
]));

var dup219 = match("MESSAGE#399:994", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var dup220 = match("MESSAGE#406:1110", "nwparser.payload", "msg=\"%{msg}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
	dup23,
]));

var dup221 = match("MESSAGE#420:614", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup163,
	dup37,
]));

var dup222 = match("MESSAGE#454:654", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2}", processor_chain([
	dup1,
]));

var dup223 = linear_select([
	dup169,
	dup170,
]);

var dup224 = linear_select([
	dup172,
	dup173,
]);

var dup225 = match("MESSAGE#482:796", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var dup226 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var dup227 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup85,
	]),
});

var dup228 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup59,
	]),
});

var dup229 = all_match({
	processors: [
		dup95,
		dup192,
	],
	on_success: processor_chain([
		dup59,
	]),
});

var dup230 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup100,
	]),
});

var dup231 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup29,
	]),
});

var dup232 = all_match({
	processors: [
		dup102,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup103,
	]),
});

var dup233 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup106,
	]),
});

var dup234 = all_match({
	processors: [
		dup107,
		dup198,
	],
	on_success: processor_chain([
		dup87,
	]),
});

var dup235 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup109,
	]),
});

var dup236 = all_match({
	processors: [
		dup44,
		dup179,
		dup36,
		dup178,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var dup237 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup175,
		dup79,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var dup238 = all_match({
	processors: [
		dup151,
		dup213,
		dup153,
		dup214,
		dup215,
		dup158,
	],
	on_success: processor_chain([
		dup150,
		dup51,
		dup52,
		dup53,
		dup54,
		dup37,
		dup55,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var dup239 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup191,
		dup94,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var dup240 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "id=%{hfld1->} sn=%{hserial_number->} time=\"%{date->} %{time}\" fw=%{hhostip->} pri=%{hseverity->} c=%{hcategory->} m=%{messageid->} %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = match("HEADER#1:0002", "message", "id=%{hfld1->} sn=%{hserial_number->} time=\"%{date->} %{time}\" fw=%{hhostip->} pri=%{hseverity->} %{messageid}= %{payload}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("= "),
			field("payload"),
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

var part14 = match("MESSAGE#13:14/0", "nwparser.payload", "%{} %{p0}");

var part15 = match("MESSAGE#13:14/1_0", "nwparser.p0", "msg=\"Web site access denied\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} dstname=%{dhost->} arg=%{fld2->} code=%{icmpcode->} ");

var part16 = match("MESSAGE#13:14/1_1", "nwparser.p0", "Web site blocked %{}");

var select5 = linear_select([
	part15,
	part16,
]);

var all1 = all_match({
	processors: [
		part14,
		select5,
	],
	on_success: processor_chain([
		dup6,
		setc("action","Web site access denied"),
	]),
});

var msg14 = msg("14", all1);

var part17 = match("MESSAGE#14:14:01/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} code= %{p0}");

var part18 = match("MESSAGE#14:14:01/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} code= %{p0}");

var select6 = linear_select([
	part17,
	part18,
]);

var part19 = match("MESSAGE#14:14:01/4", "nwparser.p0", "%{} %{fld3->} Category=%{fld4->} npcs=%{info}");

var all2 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		select6,
		part19,
	],
	on_success: processor_chain([
		dup6,
	]),
});

var msg15 = msg("14:01", all2);

var part20 = match("MESSAGE#15:14:02", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{name->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg16 = msg("14:02", part20);

var part21 = match("MESSAGE#16:14:03", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg17 = msg("14:03", part21);

var part22 = match("MESSAGE#17:14:04", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{name->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg18 = msg("14:04", part22);

var part23 = match("MESSAGE#18:14:05", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} sess=\"%{fld2}\" n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr}dstMac=%{dmacaddr->} proto=%{protocol->} dstname=%{dhost->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup6,
	dup11,
]));

var msg19 = msg("14:05", part23);

var select7 = linear_select([
	msg14,
	msg15,
	msg16,
	msg17,
	msg18,
	msg19,
]);

var part24 = match("MESSAGE#19:15", "nwparser.payload", "Newsgroup blocked%{}", processor_chain([
	dup12,
]));

var msg20 = msg("15", part24);

var part25 = match("MESSAGE#20:16", "nwparser.payload", "Web site accessed%{}", processor_chain([
	dup13,
]));

var msg21 = msg("16", part25);

var part26 = match("MESSAGE#21:17", "nwparser.payload", "Newsgroup accessed%{}", processor_chain([
	dup13,
]));

var msg22 = msg("17", part26);

var part27 = match("MESSAGE#22:18", "nwparser.payload", "ActiveX blocked%{}", processor_chain([
	dup12,
]));

var msg23 = msg("18", part27);

var part28 = match("MESSAGE#23:19", "nwparser.payload", "Java blocked%{}", processor_chain([
	dup12,
]));

var msg24 = msg("19", part28);

var part29 = match("MESSAGE#24:20", "nwparser.payload", "ActiveX or Java archive blocked%{}", processor_chain([
	dup12,
]));

var msg25 = msg("20", part29);

var part30 = match("MESSAGE#25:21", "nwparser.payload", "Cookie removed%{}", processor_chain([
	dup1,
]));

var msg26 = msg("21", part30);

var part31 = match("MESSAGE#26:22", "nwparser.payload", "Ping of death blocked%{}", processor_chain([
	dup14,
]));

var msg27 = msg("22", part31);

var part32 = match("MESSAGE#27:23", "nwparser.payload", "IP spoof detected%{}", processor_chain([
	dup14,
]));

var msg28 = msg("23", part32);

var part33 = match("MESSAGE#28:23:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part34 = match("MESSAGE#28:23:01/3_0", "nwparser.p0", "- MAC address: %{p0}");

var part35 = match("MESSAGE#28:23:01/3_1", "nwparser.p0", "mac= %{p0}");

var select8 = linear_select([
	part34,
	part35,
]);

var part36 = match("MESSAGE#28:23:01/4", "nwparser.p0", "%{} %{smacaddr}");

var all3 = all_match({
	processors: [
		part33,
		dup175,
		dup10,
		select8,
		part36,
	],
	on_success: processor_chain([
		dup14,
	]),
});

var msg29 = msg("23:01", all3);

var part37 = match("MESSAGE#29:23:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} - MAC address: %{smacaddr}", processor_chain([
	dup14,
]));

var msg30 = msg("23:02", part37);

var part38 = match("MESSAGE#30:23:03/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part39 = match("MESSAGE#30:23:03/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac= %{p0}");

var part40 = match("MESSAGE#30:23:03/1_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} srcMac= %{p0}");

var select9 = linear_select([
	part39,
	part40,
]);

var part41 = match("MESSAGE#30:23:03/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol}");

var all4 = all_match({
	processors: [
		part38,
		select9,
		part41,
	],
	on_success: processor_chain([
		dup14,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg31 = msg("23:03", all4);

var select10 = linear_select([
	msg28,
	msg29,
	msg30,
	msg31,
]);

var part42 = match("MESSAGE#31:24", "nwparser.payload", "Illegal LAN address in use%{}", processor_chain([
	dup22,
]));

var msg32 = msg("24", part42);

var msg33 = msg("24:01", dup176);

var select11 = linear_select([
	msg32,
	msg33,
]);

var part43 = match("MESSAGE#32:25", "nwparser.payload", "Possible SYN flood attack%{}", processor_chain([
	dup14,
]));

var msg34 = msg("25", part43);

var part44 = match("MESSAGE#33:26", "nwparser.payload", "Probable SYN flood attack%{}", processor_chain([
	dup14,
]));

var msg35 = msg("26", part44);

var part45 = match("MESSAGE#34:27", "nwparser.payload", "Land Attack Dropped%{}", processor_chain([
	dup14,
]));

var msg36 = msg("27", part45);

var part46 = match("MESSAGE#35:28", "nwparser.payload", "Fragmented Packet Dropped%{}", processor_chain([
	dup14,
]));

var msg37 = msg("28", part46);

var part47 = match("MESSAGE#36:28:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup14,
]));

var msg38 = msg("28:01", part47);

var select12 = linear_select([
	msg37,
	msg38,
]);

var part48 = match("MESSAGE#37:29", "nwparser.payload", "Successful administrator login%{}", processor_chain([
	dup24,
]));

var msg39 = msg("29", part48);

var part49 = match("MESSAGE#38:29:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var all5 = all_match({
	processors: [
		part49,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup29,
	]),
});

var msg40 = msg("29:01", all5);

var select13 = linear_select([
	msg39,
	msg40,
]);

var part50 = match("MESSAGE#39:30", "nwparser.payload", "Administrator login failed - incorrect password%{}", processor_chain([
	dup30,
]));

var msg41 = msg("30", part50);

var msg42 = msg("30:01", dup226);

var select14 = linear_select([
	msg41,
	msg42,
]);

var part51 = match("MESSAGE#41:31", "nwparser.payload", "Successful user login%{}", processor_chain([
	dup24,
]));

var msg43 = msg("31", part51);

var all6 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup24,
	]),
});

var msg44 = msg("31:01", all6);

var part52 = match("MESSAGE#43:31:02", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup24,
	dup11,
]));

var msg45 = msg("31:02", part52);

var part53 = match("MESSAGE#44:31:03", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration}n=%{fld1}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}proto=%{protocol}note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup24,
	dup11,
]));

var msg46 = msg("31:03", part53);

var part54 = match("MESSAGE#45:31:04", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup24,
	dup11,
]));

var msg47 = msg("31:04", part54);

var select15 = linear_select([
	msg43,
	msg44,
	msg45,
	msg46,
	msg47,
]);

var part55 = match("MESSAGE#46:32", "nwparser.payload", "User login failed - incorrect password%{}", processor_chain([
	dup30,
]));

var msg48 = msg("32", part55);

var msg49 = msg("32:01", dup226);

var select16 = linear_select([
	msg48,
	msg49,
]);

var part56 = match("MESSAGE#48:33", "nwparser.payload", "Unknown user attempted to log in%{}", processor_chain([
	dup32,
]));

var msg50 = msg("33", part56);

var all7 = all_match({
	processors: [
		dup33,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var msg51 = msg("33:01", all7);

var select17 = linear_select([
	msg50,
	msg51,
]);

var part57 = match("MESSAGE#50:34", "nwparser.payload", "Login screen timed out%{}", processor_chain([
	dup5,
]));

var msg52 = msg("34", part57);

var part58 = match("MESSAGE#51:35", "nwparser.payload", "Attempted administrator login from WAN%{}", processor_chain([
	setc("eventcategory","1401040000"),
]));

var msg53 = msg("35", part58);

var part59 = match("MESSAGE#52:35:01/3_1", "nwparser.p0", "%{daddr}");

var select18 = linear_select([
	dup27,
	part59,
]);

var all8 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		select18,
	],
	on_success: processor_chain([
		setc("eventcategory","1401050200"),
	]),
});

var msg54 = msg("35:01", all8);

var select19 = linear_select([
	msg53,
	msg54,
]);

var part60 = match("MESSAGE#53:36", "nwparser.payload", "TCP connection dropped%{}", processor_chain([
	dup5,
]));

var msg55 = msg("36", part60);

var part61 = match("MESSAGE#54:36:01/0", "nwparser.payload", "msg=\"%{msg}\" %{p0}");

var part62 = match("MESSAGE#54:36:01/1_0", "nwparser.p0", "app=%{fld51->} appName=\"%{application}\" n=%{fld1->} src= %{p0}");

var part63 = match("MESSAGE#54:36:01/1_1", "nwparser.p0", "n=%{fld1->} src= %{p0}");

var select20 = linear_select([
	part62,
	part63,
]);

var part64 = match("MESSAGE#54:36:01/6_0", "nwparser.p0", "srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\" ");

var part65 = match("MESSAGE#54:36:01/6_1", "nwparser.p0", " rule=%{rule->} ");

var part66 = match("MESSAGE#54:36:01/6_2", "nwparser.p0", " proto=%{protocol->} ");

var select21 = linear_select([
	part64,
	part65,
	part66,
]);

var all9 = all_match({
	processors: [
		part61,
		select20,
		dup179,
		dup36,
		dup175,
		dup10,
		select21,
	],
	on_success: processor_chain([
		dup5,
		dup37,
	]),
});

var msg56 = msg("36:01", all9);

var part67 = match("MESSAGE#55:36:02/5_0", "nwparser.p0", "rule=%{rule->} %{p0}");

var part68 = match("MESSAGE#55:36:02/5_1", "nwparser.p0", "proto=%{protocol->} %{p0}");

var select22 = linear_select([
	part67,
	part68,
]);

var part69 = match("MESSAGE#55:36:02/6", "nwparser.p0", "%{}npcs=%{info}");

var all10 = all_match({
	processors: [
		dup38,
		dup180,
		dup10,
		dup175,
		dup10,
		select22,
		part69,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg57 = msg("36:02", all10);

var select23 = linear_select([
	msg55,
	msg56,
	msg57,
]);

var part70 = match("MESSAGE#56:37", "nwparser.payload", "UDP packet dropped%{}", processor_chain([
	dup5,
]));

var msg58 = msg("37", part70);

var part71 = match("MESSAGE#57:37:01/0", "nwparser.payload", "msg=\"UDP packet dropped\" %{p0}");

var part72 = match("MESSAGE#57:37:01/1_0", "nwparser.p0", "app=%{fld51->} appName=\"%{application}\" n=%{fld1->} src=%{p0}");

var select24 = linear_select([
	part72,
	dup40,
]);

var part73 = match("MESSAGE#57:37:01/2", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{p0}");

var part74 = match("MESSAGE#57:37:01/3_0", "nwparser.p0", "%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} %{p0}");

var part75 = match("MESSAGE#57:37:01/3_1", "nwparser.p0", "%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} %{p0}");

var part76 = match("MESSAGE#57:37:01/3_2", "nwparser.p0", "%{dport}:%{dinterface->} %{p0}");

var select25 = linear_select([
	part74,
	part75,
	part76,
]);

var part77 = match("MESSAGE#57:37:01/4_0", "nwparser.p0", "proto=%{protocol->} fw_action=\"%{fld3}\" ");

var part78 = match("MESSAGE#57:37:01/4_1", "nwparser.p0", " rule=%{rule}");

var select26 = linear_select([
	part77,
	part78,
]);

var all11 = all_match({
	processors: [
		part71,
		select24,
		part73,
		select25,
		select26,
	],
	on_success: processor_chain([
		dup5,
		dup37,
	]),
});

var msg59 = msg("37:01", all11);

var part79 = match("MESSAGE#58:37:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} rule=%{rule}", processor_chain([
	dup5,
]));

var msg60 = msg("37:02", part79);

var all12 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup181,
		dup43,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg61 = msg("37:03", all12);

var part80 = match("MESSAGE#60:37:04", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup11,
]));

var msg62 = msg("37:04", part80);

var select27 = linear_select([
	msg58,
	msg59,
	msg60,
	msg61,
	msg62,
]);

var part81 = match("MESSAGE#61:38", "nwparser.payload", "ICMP packet dropped%{}", processor_chain([
	dup5,
]));

var msg63 = msg("38", part81);

var part82 = match("MESSAGE#62:38:01/5_0", "nwparser.p0", "type=%{type->} code=%{code->} ");

var select28 = linear_select([
	part82,
	dup45,
]);

var all13 = all_match({
	processors: [
		dup44,
		dup179,
		dup36,
		dup175,
		dup10,
		select28,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg64 = msg("38:01", all13);

var part83 = match("MESSAGE#63:38:02/4", "nwparser.p0", "%{} %{fld3->} icmpCode=%{fld4->} npcs=%{info}");

var all14 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup182,
		part83,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg65 = msg("38:02", all14);

var part84 = match("MESSAGE#64:38:03/1_0", "nwparser.p0", "%{event_description}\" app=%{fld2->} appName=\"%{application}\"%{p0}");

var part85 = match("MESSAGE#64:38:03/1_1", "nwparser.p0", "%{event_description}\"%{p0}");

var select29 = linear_select([
	part84,
	part85,
]);

var part86 = match("MESSAGE#64:38:03/2", "nwparser.p0", "%{}n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part87 = match("MESSAGE#64:38:03/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"");

var all15 = all_match({
	processors: [
		dup48,
		select29,
		part86,
		dup183,
		part87,
	],
	on_success: processor_chain([
		dup5,
		dup11,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg66 = msg("38:03", all15);

var select30 = linear_select([
	msg63,
	msg64,
	msg65,
	msg66,
]);

var part88 = match("MESSAGE#65:39", "nwparser.payload", "PPTP packet dropped%{}", processor_chain([
	dup5,
]));

var msg67 = msg("39", part88);

var part89 = match("MESSAGE#66:40", "nwparser.payload", "IPSec packet dropped%{}", processor_chain([
	dup5,
]));

var msg68 = msg("40", part89);

var part90 = match("MESSAGE#67:41:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} note=\"IP Protocol: %{dclass_counter1}\"", processor_chain([
	dup5,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg69 = msg("41:01", part90);

var part91 = match("MESSAGE#68:41:02", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport}:%{sinterface->} dst=%{dtransaddr}:%{dtransport}::%{dinterface}", processor_chain([
	dup5,
]));

var msg70 = msg("41:02", part91);

var part92 = match("MESSAGE#69:41:03", "nwparser.payload", "Unknown protocol dropped%{}", processor_chain([
	dup5,
]));

var msg71 = msg("41:03", part92);

var select31 = linear_select([
	msg69,
	msg70,
	msg71,
]);

var part93 = match("MESSAGE#70:42", "nwparser.payload", "IPSec packet dropped; waiting for pending IPSec connection%{}", processor_chain([
	dup5,
]));

var msg72 = msg("42", part93);

var part94 = match("MESSAGE#71:43", "nwparser.payload", "IPSec connection interrupt%{}", processor_chain([
	dup5,
]));

var msg73 = msg("43", part94);

var part95 = match("MESSAGE#72:44", "nwparser.payload", "NAT could not remap incoming packet%{}", processor_chain([
	dup5,
]));

var msg74 = msg("44", part95);

var part96 = match("MESSAGE#73:45", "nwparser.payload", "ARP timeout%{}", processor_chain([
	dup5,
]));

var msg75 = msg("45", part96);

var part97 = match("MESSAGE#74:45:01", "nwparser.payload", "msg=\"ARP timeout\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup5,
]));

var msg76 = msg("45:01", part97);

var part98 = match("MESSAGE#75:45:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr->} dst=%{daddr->} npcs=%{info}", processor_chain([
	dup5,
]));

var msg77 = msg("45:02", part98);

var select32 = linear_select([
	msg75,
	msg76,
	msg77,
]);

var part99 = match("MESSAGE#76:46:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} proto=%{protocol}/%{fld4}", processor_chain([
	dup5,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg78 = msg("46:01", part99);

var part100 = match("MESSAGE#77:46:02", "nwparser.payload", "msg=\"Broadcast packet dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup5,
]));

var msg79 = msg("46:02", part100);

var part101 = match("MESSAGE#78:46", "nwparser.payload", "Broadcast packet dropped%{}", processor_chain([
	dup5,
]));

var msg80 = msg("46", part101);

var part102 = match("MESSAGE#79:46:03/0", "nwparser.payload", "msg=\"Broadcast packet dropped\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var all16 = all_match({
	processors: [
		part102,
		dup174,
		dup10,
		dup181,
		dup43,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg81 = msg("46:03", all16);

var select33 = linear_select([
	msg78,
	msg79,
	msg80,
	msg81,
]);

var part103 = match("MESSAGE#80:47", "nwparser.payload", "No ICMP redirect sent%{}", processor_chain([
	dup5,
]));

var msg82 = msg("47", part103);

var part104 = match("MESSAGE#81:48", "nwparser.payload", "Out-of-order command packet dropped%{}", processor_chain([
	dup5,
]));

var msg83 = msg("48", part104);

var part105 = match("MESSAGE#82:49", "nwparser.payload", "Failure to add data channel%{}", processor_chain([
	dup5,
]));

var msg84 = msg("49", part105);

var part106 = match("MESSAGE#83:50", "nwparser.payload", "RealAudio decode failure%{}", processor_chain([
	dup5,
]));

var msg85 = msg("50", part106);

var part107 = match("MESSAGE#84:51", "nwparser.payload", "Duplicate packet dropped%{}", processor_chain([
	dup5,
]));

var msg86 = msg("51", part107);

var part108 = match("MESSAGE#85:52", "nwparser.payload", "No HOST tag found in HTTP request%{}", processor_chain([
	dup5,
]));

var msg87 = msg("52", part108);

var part109 = match("MESSAGE#86:53", "nwparser.payload", "The cache is full; too many open connections; some will be dropped%{}", processor_chain([
	dup2,
]));

var msg88 = msg("53", part109);

var part110 = match("MESSAGE#87:58", "nwparser.payload", "License exceeded: Connection dropped because too many IP addresses are in use on your LAN%{}", processor_chain([
	dup56,
]));

var msg89 = msg("58", part110);

var part111 = match("MESSAGE#88:60", "nwparser.payload", "Access to Proxy Server Blocked%{}", processor_chain([
	dup12,
]));

var msg90 = msg("60", part111);

var part112 = match("MESSAGE#89:61", "nwparser.payload", "Diagnostic Code E%{}", processor_chain([
	dup1,
]));

var msg91 = msg("61", part112);

var part113 = match("MESSAGE#90:62", "nwparser.payload", "Dynamic IPSec client connected%{}", processor_chain([
	dup57,
]));

var msg92 = msg("62", part113);

var part114 = match("MESSAGE#91:63", "nwparser.payload", "IPSec packet too big%{}", processor_chain([
	dup58,
]));

var msg93 = msg("63", part114);

var part115 = match("MESSAGE#92:63:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup58,
]));

var msg94 = msg("63:01", part115);

var select34 = linear_select([
	msg93,
	msg94,
]);

var part116 = match("MESSAGE#93:64", "nwparser.payload", "Diagnostic Code D%{}", processor_chain([
	dup1,
]));

var msg95 = msg("64", part116);

var part117 = match("MESSAGE#94:65", "nwparser.payload", "Illegal IPSec SPI%{}", processor_chain([
	dup58,
]));

var msg96 = msg("65", part117);

var part118 = match("MESSAGE#95:66", "nwparser.payload", "Unknown IPSec SPI%{}", processor_chain([
	dup58,
]));

var msg97 = msg("66", part118);

var part119 = match("MESSAGE#96:67", "nwparser.payload", "IPSec Authentication Failed%{}", processor_chain([
	dup58,
]));

var msg98 = msg("67", part119);

var all17 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup58,
	]),
});

var msg99 = msg("67:01", all17);

var select35 = linear_select([
	msg98,
	msg99,
]);

var part120 = match("MESSAGE#98:68", "nwparser.payload", "IPSec Decryption Failed%{}", processor_chain([
	dup58,
]));

var msg100 = msg("68", part120);

var part121 = match("MESSAGE#99:69", "nwparser.payload", "Incompatible IPSec Security Association%{}", processor_chain([
	dup58,
]));

var msg101 = msg("69", part121);

var part122 = match("MESSAGE#100:70", "nwparser.payload", "IPSec packet from illegal host%{}", processor_chain([
	dup58,
]));

var msg102 = msg("70", part122);

var part123 = match("MESSAGE#101:70:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} %{p0}");

var part124 = match("MESSAGE#101:70:01/1_0", "nwparser.p0", "dst=%{daddr->} ");

var part125 = match("MESSAGE#101:70:01/1_1", "nwparser.p0", " dstname=%{name}");

var select36 = linear_select([
	part124,
	part125,
]);

var all18 = all_match({
	processors: [
		part123,
		select36,
	],
	on_success: processor_chain([
		dup58,
	]),
});

var msg103 = msg("70:01", all18);

var select37 = linear_select([
	msg102,
	msg103,
]);

var part126 = match("MESSAGE#102:72", "nwparser.payload", "NetBus Attack Dropped%{}", processor_chain([
	dup59,
]));

var msg104 = msg("72", part126);

var part127 = match("MESSAGE#103:72:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup59,
]));

var msg105 = msg("72:01", part127);

var select38 = linear_select([
	msg104,
	msg105,
]);

var part128 = match("MESSAGE#104:73", "nwparser.payload", "Back Orifice Attack Dropped%{}", processor_chain([
	dup60,
]));

var msg106 = msg("73", part128);

var part129 = match("MESSAGE#105:74", "nwparser.payload", "Net Spy Attack Dropped%{}", processor_chain([
	dup61,
]));

var msg107 = msg("74", part129);

var part130 = match("MESSAGE#106:75", "nwparser.payload", "Sub Seven Attack Dropped%{}", processor_chain([
	dup60,
]));

var msg108 = msg("75", part130);

var part131 = match("MESSAGE#107:76", "nwparser.payload", "Ripper Attack Dropped%{}", processor_chain([
	dup59,
]));

var msg109 = msg("76", part131);

var part132 = match("MESSAGE#108:77", "nwparser.payload", "Striker Attack Dropped%{}", processor_chain([
	dup59,
]));

var msg110 = msg("77", part132);

var part133 = match("MESSAGE#109:78", "nwparser.payload", "Senna Spy Attack Dropped%{}", processor_chain([
	dup61,
]));

var msg111 = msg("78", part133);

var part134 = match("MESSAGE#110:79", "nwparser.payload", "Priority Attack Dropped%{}", processor_chain([
	dup59,
]));

var msg112 = msg("79", part134);

var part135 = match("MESSAGE#111:80", "nwparser.payload", "Ini Killer Attack Dropped%{}", processor_chain([
	dup59,
]));

var msg113 = msg("80", part135);

var part136 = match("MESSAGE#112:81", "nwparser.payload", "Smurf Amplification Attack Dropped%{}", processor_chain([
	dup14,
]));

var msg114 = msg("81", part136);

var part137 = match("MESSAGE#113:82", "nwparser.payload", "Possible Port Scan%{}", processor_chain([
	dup62,
]));

var msg115 = msg("82", part137);

var part138 = match("MESSAGE#114:82:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{info}\"", processor_chain([
	dup62,
]));

var msg116 = msg("82:02", part138);

var part139 = match("MESSAGE#115:82:03", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{fld3}\" npcs=%{info}", processor_chain([
	dup62,
]));

var msg117 = msg("82:03", part139);

var msg118 = msg("82:01", dup184);

var select39 = linear_select([
	msg115,
	msg116,
	msg117,
	msg118,
]);

var part140 = match("MESSAGE#117:83", "nwparser.payload", "Probable Port Scan%{}", processor_chain([
	dup62,
]));

var msg119 = msg("83", part140);

var msg120 = msg("83:01", dup185);

var part141 = match("MESSAGE#119:83:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{fld3}\" npcs=%{info}", processor_chain([
	dup5,
]));

var msg121 = msg("83:02", part141);

var select40 = linear_select([
	msg119,
	msg120,
	msg121,
]);

var part142 = match("MESSAGE#120:84/0_0", "nwparser.payload", "msg=\"Failed to resolve name\" n=%{fld1->} dstname=%{dhost}");

var part143 = match("MESSAGE#120:84/0_1", "nwparser.payload", "Failed to resolve name%{}");

var select41 = linear_select([
	part142,
	part143,
]);

var all19 = all_match({
	processors: [
		select41,
	],
	on_success: processor_chain([
		dup63,
		setc("action","Failed to resolve name"),
	]),
});

var msg122 = msg("84", all19);

var part144 = match("MESSAGE#121:87", "nwparser.payload", "IKE Responder: Accepting IPSec proposal%{}", processor_chain([
	dup64,
]));

var msg123 = msg("87", part144);

var part145 = match("MESSAGE#122:87:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup64,
]));

var msg124 = msg("87:01", part145);

var select42 = linear_select([
	msg123,
	msg124,
]);

var part146 = match("MESSAGE#123:88", "nwparser.payload", "IKE Responder: IPSec proposal not acceptable%{}", processor_chain([
	dup58,
]));

var msg125 = msg("88", part146);

var part147 = match("MESSAGE#124:88:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup58,
]));

var msg126 = msg("88:01", part147);

var select43 = linear_select([
	msg125,
	msg126,
]);

var part148 = match("MESSAGE#125:89", "nwparser.payload", "IKE negotiation complete. Adding IPSec SA%{}", processor_chain([
	dup64,
]));

var msg127 = msg("89", part148);

var part149 = match("MESSAGE#126:89:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} %{p0}");

var part150 = match("MESSAGE#126:89:01/1_0", "nwparser.p0", "src=%{saddr}:::%{sinterface->} dst=%{daddr}:::%{dinterface->} ");

var part151 = match("MESSAGE#126:89:01/1_1", "nwparser.p0", " src=%{saddr->} dst=%{daddr->} dstname=%{name}");

var select44 = linear_select([
	part150,
	part151,
]);

var all20 = all_match({
	processors: [
		part149,
		select44,
	],
	on_success: processor_chain([
		dup64,
	]),
});

var msg128 = msg("89:01", all20);

var select45 = linear_select([
	msg127,
	msg128,
]);

var part152 = match("MESSAGE#127:90", "nwparser.payload", "Starting IKE negotiation%{}", processor_chain([
	dup64,
]));

var msg129 = msg("90", part152);

var part153 = match("MESSAGE#128:91", "nwparser.payload", "Deleting IPSec SA for destination%{}", processor_chain([
	dup64,
]));

var msg130 = msg("91", part153);

var part154 = match("MESSAGE#129:92", "nwparser.payload", "Deleting IPSec SA%{}", processor_chain([
	dup64,
]));

var msg131 = msg("92", part154);

var part155 = match("MESSAGE#130:93", "nwparser.payload", "Diagnostic Code A%{}", processor_chain([
	dup1,
]));

var msg132 = msg("93", part155);

var part156 = match("MESSAGE#131:94", "nwparser.payload", "Diagnostic Code B%{}", processor_chain([
	dup1,
]));

var msg133 = msg("94", part156);

var part157 = match("MESSAGE#132:95", "nwparser.payload", "Diagnostic Code C%{}", processor_chain([
	dup1,
]));

var msg134 = msg("95", part157);

var part158 = match("MESSAGE#133:96", "nwparser.payload", "Status%{}", processor_chain([
	dup1,
]));

var msg135 = msg("96", part158);

var part159 = match("MESSAGE#134:97", "nwparser.payload", "Web site hit%{}", processor_chain([
	dup1,
]));

var msg136 = msg("97", part159);

var part160 = match("MESSAGE#135:97:01/4", "nwparser.p0", "%{}proto=%{protocol->} op=%{fld->} %{p0}");

var part161 = match("MESSAGE#135:97:01/5_0", "nwparser.p0", "rcvd=%{rbytes->} %{p0}");

var part162 = match("MESSAGE#135:97:01/5_1", "nwparser.p0", "sent=%{sbytes->} %{p0}");

var select46 = linear_select([
	part161,
	part162,
]);

var part163 = match("MESSAGE#135:97:01/7_0", "nwparser.p0", "result=%{result->} dstname=%{name->} ");

var select47 = linear_select([
	part163,
	dup66,
]);

var all21 = all_match({
	processors: [
		dup65,
		dup179,
		dup36,
		dup175,
		part160,
		select46,
		dup10,
		select47,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg137 = msg("97:01", all21);

var part164 = match("MESSAGE#136:97:02/4", "nwparser.p0", "%{}proto=%{protocol->} op=%{fld->} result=%{result}");

var all22 = all_match({
	processors: [
		dup65,
		dup179,
		dup36,
		dup175,
		part164,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg138 = msg("97:02", all22);

var part165 = match("MESSAGE#137:97:03/4", "nwparser.p0", "%{}proto=%{protocol->} op=%{fld3->} sent=%{sbytes->} rcvd=%{rbytes->} %{p0}");

var part166 = match("MESSAGE#137:97:03/5_0", "nwparser.p0", "result=%{result->} dstname=%{name->} %{p0}");

var part167 = match("MESSAGE#137:97:03/5_1", "nwparser.p0", "dstname=%{name->} %{p0}");

var select48 = linear_select([
	part166,
	part167,
]);

var part168 = match("MESSAGE#137:97:03/6", "nwparser.p0", "%{}arg=%{fld4->} code=%{fld5->} Category=\"%{category}\" npcs=%{info}");

var all23 = all_match({
	processors: [
		dup67,
		dup179,
		dup36,
		dup175,
		part165,
		select48,
		part168,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg139 = msg("97:03", all23);

var part169 = match("MESSAGE#138:97:04/4", "nwparser.p0", "%{}proto=%{protocol->} op=%{fld3->} %{p0}");

var part170 = match("MESSAGE#138:97:04/5_0", "nwparser.p0", "result=%{result->} dstname=%{name->} arg= %{p0}");

var part171 = match("MESSAGE#138:97:04/5_1", "nwparser.p0", "dstname=%{name->} arg= %{p0}");

var select49 = linear_select([
	part170,
	part171,
]);

var part172 = match("MESSAGE#138:97:04/6", "nwparser.p0", "%{} %{fld4->} code=%{fld5->} Category=\"%{category}\" npcs=%{info}");

var all24 = all_match({
	processors: [
		dup67,
		dup179,
		dup36,
		dup175,
		part169,
		select49,
		part172,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg140 = msg("97:04", all24);

var part173 = match("MESSAGE#139:97:05/4", "nwparser.p0", "%{}proto=%{protocol->} op=%{fld2->} dstname=%{name->} arg=%{fld3->} code=%{fld4->} Category=%{category}");

var all25 = all_match({
	processors: [
		dup65,
		dup179,
		dup36,
		dup175,
		part173,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg141 = msg("97:05", all25);

var part174 = match("MESSAGE#140:97:06/0", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{p0}");

var select50 = linear_select([
	dup68,
	dup69,
]);

var part175 = match("MESSAGE#140:97:06/2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"");

var all26 = all_match({
	processors: [
		part174,
		select50,
		part175,
	],
	on_success: processor_chain([
		dup70,
		dup11,
	]),
});

var msg142 = msg("97:06", all26);

var part176 = match("MESSAGE#141:97:07/0", "nwparser.payload", "app=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{p0}");

var part177 = match("MESSAGE#141:97:07/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{fld3->} srcMac=%{p0}");

var select51 = linear_select([
	part177,
	dup49,
]);

var part178 = match("MESSAGE#141:97:07/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} dstname=%{dhost->} arg=%{param->} code=%{resultcode->} Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"");

var all27 = all_match({
	processors: [
		part176,
		select51,
		part178,
	],
	on_success: processor_chain([
		dup70,
		dup11,
	]),
});

var msg143 = msg("97:07", all27);

var part179 = match("MESSAGE#142:97:08", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup11,
]));

var msg144 = msg("97:08", part179);

var part180 = match("MESSAGE#143:97:09", "nwparser.payload", "app=%{fld1}sess=\"%{fld2}\" n=%{fld3}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup11,
]));

var msg145 = msg("97:09", part180);

var part181 = match("MESSAGE#144:97:10", "nwparser.payload", "app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}dstname=%{dhost}arg=%{param}code=%{resultcode}Category=\"%{category}\" rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup11,
]));

var msg146 = msg("97:10", part181);

var select52 = linear_select([
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

var part182 = match("MESSAGE#145:98/0_0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld2->} appName=\"%{application}\"%{p0}");

var part183 = match("MESSAGE#145:98/0_1", "nwparser.payload", " msg=\"%{event_description}\"%{p0}");

var select53 = linear_select([
	part182,
	part183,
]);

var part184 = match("MESSAGE#145:98/1", "nwparser.p0", "%{}n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{p0}");

var part185 = match("MESSAGE#145:98/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} dstMac=%{dmacaddr->} %{p0}");

var select54 = linear_select([
	part185,
	dup71,
]);

var part186 = match("MESSAGE#145:98/3_1", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} ");

var part187 = match("MESSAGE#145:98/3_2", "nwparser.p0", " proto=%{protocol}");

var select55 = linear_select([
	dup72,
	part186,
	part187,
]);

var all28 = all_match({
	processors: [
		select53,
		part184,
		select54,
		select55,
	],
	on_success: processor_chain([
		dup70,
		dup51,
		setc("ec_activity","Stop"),
		dup53,
		dup54,
		dup11,
		setc("action","Opened"),
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg147 = msg("98", all28);

var part188 = match("MESSAGE#146:98:07", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} dstMac=%{dmacaddr->} proto=%{protocol}/%{fld4->} sent=%{sbytes->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg148 = msg("98:07", part188);

var part189 = match("MESSAGE#147:98:01/1_0", "nwparser.p0", "%{msg}\" app=%{fld2->} sess=\"%{fld3}\"%{p0}");

var part190 = match("MESSAGE#147:98:01/1_1", "nwparser.p0", "%{msg}\"%{p0}");

var select56 = linear_select([
	part189,
	part190,
]);

var part191 = match("MESSAGE#147:98:01/2", "nwparser.p0", "%{}n=%{p0}");

var part192 = match("MESSAGE#147:98:01/3_0", "nwparser.p0", "%{fld1->} usr=%{username->} src=%{p0}");

var part193 = match("MESSAGE#147:98:01/3_1", "nwparser.p0", "%{fld1->} src=%{p0}");

var select57 = linear_select([
	part192,
	part193,
]);

var select58 = linear_select([
	dup73,
	dup69,
	dup74,
]);

var part194 = match("MESSAGE#147:98:01/7_0", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var part195 = match("MESSAGE#147:98:01/7_1", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} ");

var part196 = match("MESSAGE#147:98:01/7_2", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} rule=\"%{rulename}\" fw_action=\"%{action}\"");

var part197 = match("MESSAGE#147:98:01/7_4", "nwparser.p0", " proto=%{protocol->} sent=%{sbytes}");

var part198 = match("MESSAGE#147:98:01/7_5", "nwparser.p0", "proto=%{protocol}");

var select59 = linear_select([
	part194,
	part195,
	part196,
	dup72,
	part197,
	part198,
]);

var all29 = all_match({
	processors: [
		dup48,
		select56,
		part191,
		select57,
		select58,
		dup10,
		dup186,
		select59,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg149 = msg("98:01", all29);

var part199 = match("MESSAGE#148:98:06/0_0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld2->} appName=\"%{application}\" %{p0}");

var part200 = match("MESSAGE#148:98:06/0_1", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld2->} %{p0}");

var part201 = match("MESSAGE#148:98:06/0_2", "nwparser.payload", " msg=\"%{event_description}\" sess=%{fld2->} %{p0}");

var select60 = linear_select([
	part199,
	part200,
	part201,
]);

var part202 = match("MESSAGE#148:98:06/1_0", "nwparser.p0", "n=%{fld1->} usr=%{username->} %{p0}");

var part203 = match("MESSAGE#148:98:06/1_1", "nwparser.p0", " n=%{fld1->} %{p0}");

var select61 = linear_select([
	part202,
	part203,
]);

var part204 = match("MESSAGE#148:98:06/2", "nwparser.p0", "%{}src= %{p0}");

var part205 = match("MESSAGE#148:98:06/5_0", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{dmacaddr->} proto=%{p0}");

var part206 = match("MESSAGE#148:98:06/5_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} dstMac=%{dmacaddr->} proto=%{p0}");

var select62 = linear_select([
	part205,
	part206,
	dup77,
	dup78,
]);

var part207 = match("MESSAGE#148:98:06/6", "nwparser.p0", "%{protocol->} %{p0}");

var part208 = match("MESSAGE#148:98:06/7_0", "nwparser.p0", "sent=%{sbytes->} rule=\"%{rulename}\" fw_action=\"%{action}\"");

var part209 = match("MESSAGE#148:98:06/7_1", "nwparser.p0", "sent=%{sbytes->} rule=\"%{rulename}\" fw_action=%{action}");

var part210 = match("MESSAGE#148:98:06/7_2", "nwparser.p0", "sent=%{sbytes->} fw_action=\"%{action}\"");

var part211 = match("MESSAGE#148:98:06/7_3", "nwparser.p0", "sent=%{sbytes}");

var part212 = match("MESSAGE#148:98:06/7_4", "nwparser.p0", "fw_action=\"%{action}\"");

var select63 = linear_select([
	part208,
	part209,
	part210,
	part211,
	part212,
]);

var all30 = all_match({
	processors: [
		select60,
		select61,
		part204,
		dup187,
		dup10,
		select62,
		part207,
		select63,
	],
	on_success: processor_chain([
		dup70,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg150 = msg("98:06", all30);

var part213 = match("MESSAGE#149:98:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} usr=%{username->} src=%{p0}");

var all31 = all_match({
	processors: [
		part213,
		dup177,
		dup10,
		dup175,
		dup79,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg151 = msg("98:02", all31);

var part214 = match("MESSAGE#150:98:03/0_0", "nwparser.payload", "Connection %{}");

var part215 = match("MESSAGE#150:98:03/0_1", "nwparser.payload", " msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} ");

var select64 = linear_select([
	part214,
	part215,
]);

var all32 = all_match({
	processors: [
		select64,
	],
	on_success: processor_chain([
		dup1,
		dup37,
	]),
});

var msg152 = msg("98:03", all32);

var part216 = match("MESSAGE#151:98:04/4", "nwparser.p0", "%{}proto=%{protocol->} sent=%{sbytes->} vpnpolicy=\"%{policyname}\" npcs=%{info}");

var all33 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup175,
		part216,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg153 = msg("98:04", all33);

var part217 = match("MESSAGE#152:98:05/4", "nwparser.p0", "%{}proto=%{protocol->} sent=%{sbytes->} npcs=%{info}");

var all34 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup175,
		part217,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg154 = msg("98:05", all34);

var select65 = linear_select([
	msg147,
	msg148,
	msg149,
	msg150,
	msg151,
	msg152,
	msg153,
	msg154,
]);

var part218 = match("MESSAGE#153:986", "nwparser.payload", "msg=\"%{msg}\" dur=%{duration->} n=%{fld1->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup30,
	dup11,
]));

var msg155 = msg("986", part218);

var part219 = match("MESSAGE#154:427/4", "nwparser.p0", "%{}note=\"%{event_description}\"");

var all35 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup175,
		part219,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg156 = msg("427", all35);

var part220 = match("MESSAGE#155:428/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"");

var all36 = all_match({
	processors: [
		dup81,
		dup183,
		part220,
	],
	on_success: processor_chain([
		dup22,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg157 = msg("428", all36);

var part221 = match("MESSAGE#156:99", "nwparser.payload", "Retransmitting DHCP DISCOVER.%{}", processor_chain([
	dup64,
]));

var msg158 = msg("99", part221);

var part222 = match("MESSAGE#157:100", "nwparser.payload", "Retransmitting DHCP REQUEST (Requesting).%{}", processor_chain([
	dup64,
]));

var msg159 = msg("100", part222);

var part223 = match("MESSAGE#158:101", "nwparser.payload", "Retransmitting DHCP REQUEST (Renewing).%{}", processor_chain([
	dup64,
]));

var msg160 = msg("101", part223);

var part224 = match("MESSAGE#159:102", "nwparser.payload", "Retransmitting DHCP REQUEST (Rebinding).%{}", processor_chain([
	dup64,
]));

var msg161 = msg("102", part224);

var part225 = match("MESSAGE#160:103", "nwparser.payload", "Retransmitting DHCP REQUEST (Rebooting).%{}", processor_chain([
	dup64,
]));

var msg162 = msg("103", part225);

var part226 = match("MESSAGE#161:104", "nwparser.payload", "Retransmitting DHCP REQUEST (Verifying).%{}", processor_chain([
	dup64,
]));

var msg163 = msg("104", part226);

var part227 = match("MESSAGE#162:105", "nwparser.payload", "Sending DHCP DISCOVER.%{}", processor_chain([
	dup64,
]));

var msg164 = msg("105", part227);

var part228 = match("MESSAGE#163:106", "nwparser.payload", "DHCP Server not available. Did not get any DHCP OFFER.%{}", processor_chain([
	dup63,
]));

var msg165 = msg("106", part228);

var part229 = match("MESSAGE#164:107", "nwparser.payload", "Got DHCP OFFER. Selecting.%{}", processor_chain([
	dup64,
]));

var msg166 = msg("107", part229);

var part230 = match("MESSAGE#165:108", "nwparser.payload", "Sending DHCP REQUEST.%{}", processor_chain([
	dup64,
]));

var msg167 = msg("108", part230);

var part231 = match("MESSAGE#166:109", "nwparser.payload", "DHCP Client did not get DHCP ACK.%{}", processor_chain([
	dup63,
]));

var msg168 = msg("109", part231);

var part232 = match("MESSAGE#167:110", "nwparser.payload", "DHCP Client got NACK.%{}", processor_chain([
	dup64,
]));

var msg169 = msg("110", part232);

var msg170 = msg("111:01", dup188);

var part233 = match("MESSAGE#169:111", "nwparser.payload", "DHCP Client got ACK from server.%{}", processor_chain([
	dup64,
]));

var msg171 = msg("111", part233);

var select66 = linear_select([
	msg170,
	msg171,
]);

var part234 = match("MESSAGE#170:112", "nwparser.payload", "DHCP Client is declining address offered by the server.%{}", processor_chain([
	dup64,
]));

var msg172 = msg("112", part234);

var part235 = match("MESSAGE#171:113", "nwparser.payload", "DHCP Client sending REQUEST and going to REBIND state.%{}", processor_chain([
	dup64,
]));

var msg173 = msg("113", part235);

var part236 = match("MESSAGE#172:114", "nwparser.payload", "DHCP Client sending REQUEST and going to RENEW state.%{}", processor_chain([
	dup64,
]));

var msg174 = msg("114", part236);

var msg175 = msg("115:01", dup188);

var part237 = match("MESSAGE#174:115", "nwparser.payload", "Sending DHCP REQUEST (Renewing).%{}", processor_chain([
	dup64,
]));

var msg176 = msg("115", part237);

var select67 = linear_select([
	msg175,
	msg176,
]);

var part238 = match("MESSAGE#175:116", "nwparser.payload", "Sending DHCP REQUEST (Rebinding).%{}", processor_chain([
	dup64,
]));

var msg177 = msg("116", part238);

var part239 = match("MESSAGE#176:117", "nwparser.payload", "Sending DHCP REQUEST (Rebooting).%{}", processor_chain([
	dup64,
]));

var msg178 = msg("117", part239);

var part240 = match("MESSAGE#177:118", "nwparser.payload", "Sending DHCP REQUEST (Verifying).%{}", processor_chain([
	dup64,
]));

var msg179 = msg("118", part240);

var part241 = match("MESSAGE#178:119", "nwparser.payload", "DHCP Client failed to verify and lease has expired. Go to INIT state.%{}", processor_chain([
	dup63,
]));

var msg180 = msg("119", part241);

var part242 = match("MESSAGE#179:120", "nwparser.payload", "DHCP Client failed to verify and lease is still valid. Go to BOUND state.%{}", processor_chain([
	dup63,
]));

var msg181 = msg("120", part242);

var part243 = match("MESSAGE#180:121", "nwparser.payload", "DHCP Client got a new IP address lease.%{}", processor_chain([
	dup64,
]));

var msg182 = msg("121", part243);

var part244 = match("MESSAGE#181:122", "nwparser.payload", "Access attempt from host without Anti-Virus agent installed%{}", processor_chain([
	dup63,
]));

var msg183 = msg("122", part244);

var part245 = match("MESSAGE#182:123", "nwparser.payload", "Anti-Virus agent out-of-date on host%{}", processor_chain([
	dup63,
]));

var msg184 = msg("123", part245);

var part246 = match("MESSAGE#183:124", "nwparser.payload", "Received AV Alert: %s%{}", processor_chain([
	dup64,
]));

var msg185 = msg("124", part246);

var part247 = match("MESSAGE#184:125", "nwparser.payload", "Unused AV log entry.%{}", processor_chain([
	dup64,
]));

var msg186 = msg("125", part247);

var part248 = match("MESSAGE#185:1254", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup83,
	dup11,
]));

var msg187 = msg("1254", part248);

var part249 = match("MESSAGE#186:1256", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg188 = msg("1256", part249);

var part250 = match("MESSAGE#187:1257", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup83,
	dup11,
]));

var msg189 = msg("1257", part250);

var part251 = match("MESSAGE#188:126", "nwparser.payload", "Starting PPPoE discovery%{}", processor_chain([
	dup64,
]));

var msg190 = msg("126", part251);

var part252 = match("MESSAGE#189:127", "nwparser.payload", "PPPoE LCP Link Up%{}", processor_chain([
	dup64,
]));

var msg191 = msg("127", part252);

var part253 = match("MESSAGE#190:128", "nwparser.payload", "PPPoE LCP Link Down%{}", processor_chain([
	dup5,
]));

var msg192 = msg("128", part253);

var part254 = match("MESSAGE#191:129", "nwparser.payload", "PPPoE terminated%{}", processor_chain([
	dup5,
]));

var msg193 = msg("129", part254);

var part255 = match("MESSAGE#192:130", "nwparser.payload", "PPPoE Network Connected%{}", processor_chain([
	dup1,
]));

var msg194 = msg("130", part255);

var part256 = match("MESSAGE#193:131", "nwparser.payload", "PPPoE Network Disconnected%{}", processor_chain([
	dup1,
]));

var msg195 = msg("131", part256);

var part257 = match("MESSAGE#194:132", "nwparser.payload", "PPPoE discovery process complete%{}", processor_chain([
	dup1,
]));

var msg196 = msg("132", part257);

var part258 = match("MESSAGE#195:133", "nwparser.payload", "PPPoE starting CHAP Authentication%{}", processor_chain([
	dup1,
]));

var msg197 = msg("133", part258);

var part259 = match("MESSAGE#196:134", "nwparser.payload", "PPPoE starting PAP Authentication%{}", processor_chain([
	dup1,
]));

var msg198 = msg("134", part259);

var part260 = match("MESSAGE#197:135", "nwparser.payload", "PPPoE CHAP Authentication Failed%{}", processor_chain([
	dup84,
]));

var msg199 = msg("135", part260);

var part261 = match("MESSAGE#198:136", "nwparser.payload", "PPPoE PAP Authentication Failed%{}", processor_chain([
	dup84,
]));

var msg200 = msg("136", part261);

var part262 = match("MESSAGE#199:137", "nwparser.payload", "Wan IP Changed%{}", processor_chain([
	dup3,
]));

var msg201 = msg("137", part262);

var part263 = match("MESSAGE#200:138", "nwparser.payload", "XAUTH Succeeded%{}", processor_chain([
	dup3,
]));

var msg202 = msg("138", part263);

var part264 = match("MESSAGE#201:139", "nwparser.payload", "XAUTH Failed%{}", processor_chain([
	dup5,
]));

var msg203 = msg("139", part264);

var all37 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1801020100"),
	]),
});

var msg204 = msg("139:01", all37);

var select68 = linear_select([
	msg203,
	msg204,
]);

var msg205 = msg("140", dup227);

var msg206 = msg("141", dup227);

var part265 = match("MESSAGE#205:142", "nwparser.payload", "Primary firewall has transitioned to Active%{}", processor_chain([
	dup1,
]));

var msg207 = msg("142", part265);

var part266 = match("MESSAGE#206:143", "nwparser.payload", "Backup firewall has transitioned to Active%{}", processor_chain([
	dup1,
]));

var msg208 = msg("143", part266);

var part267 = match("MESSAGE#207:1431", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=::%{sinterface->} dstV6=%{daddr_v6->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} type=%{icmptype->} icmpCode=%{icmpcode->} fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup11,
]));

var msg209 = msg("1431", part267);

var part268 = match("MESSAGE#208:144", "nwparser.payload", "Primary firewall has transitioned to Idle%{}", processor_chain([
	dup1,
]));

var msg210 = msg("144", part268);

var part269 = match("MESSAGE#209:145", "nwparser.payload", "Backup firewall has transitioned to Idle%{}", processor_chain([
	dup1,
]));

var msg211 = msg("145", part269);

var part270 = match("MESSAGE#210:146", "nwparser.payload", "Primary missed heartbeats from Active Backup: Primary going Active%{}", processor_chain([
	dup86,
]));

var msg212 = msg("146", part270);

var part271 = match("MESSAGE#211:147", "nwparser.payload", "Backup missed heartbeats from Active Primary: Backup going Active%{}", processor_chain([
	dup86,
]));

var msg213 = msg("147", part271);

var part272 = match("MESSAGE#212:148", "nwparser.payload", "Primary received error signal from Active Backup: Primary going Active%{}", processor_chain([
	dup1,
]));

var msg214 = msg("148", part272);

var part273 = match("MESSAGE#213:1480", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	setc("eventcategory","1204010000"),
	dup11,
]));

var msg215 = msg("1480", part273);

var part274 = match("MESSAGE#214:149", "nwparser.payload", "Backup received error signal from Active Primary: Backup going Active%{}", processor_chain([
	dup1,
]));

var msg216 = msg("149", part274);

var part275 = match("MESSAGE#215:150", "nwparser.payload", "Backup firewall being preempted by Primary%{}", processor_chain([
	dup1,
]));

var msg217 = msg("150", part275);

var part276 = match("MESSAGE#216:151", "nwparser.payload", "Primary firewall preempting Backup%{}", processor_chain([
	dup1,
]));

var msg218 = msg("151", part276);

var part277 = match("MESSAGE#217:152", "nwparser.payload", "Active Backup detects Active Primary: Backup rebooting%{}", processor_chain([
	dup1,
]));

var msg219 = msg("152", part277);

var part278 = match("MESSAGE#218:153", "nwparser.payload", "Imported HA hardware ID did not match this firewall%{}", processor_chain([
	setc("eventcategory","1603010000"),
]));

var msg220 = msg("153", part278);

var part279 = match("MESSAGE#219:154", "nwparser.payload", "Received AV Alert: Your SonicWALL Network Anti-Virus subscription has expired. %s%{}", processor_chain([
	dup56,
]));

var msg221 = msg("154", part279);

var part280 = match("MESSAGE#220:155", "nwparser.payload", "Primary received heartbeat from wrong source%{}", processor_chain([
	dup86,
]));

var msg222 = msg("155", part280);

var part281 = match("MESSAGE#221:156", "nwparser.payload", "Backup received heartbeat from wrong source%{}", processor_chain([
	dup86,
]));

var msg223 = msg("156", part281);

var part282 = match("MESSAGE#222:157:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup1,
]));

var msg224 = msg("157:01", part282);

var part283 = match("MESSAGE#223:157", "nwparser.payload", "HA packet processing error%{}", processor_chain([
	dup5,
]));

var msg225 = msg("157", part283);

var select69 = linear_select([
	msg224,
	msg225,
]);

var part284 = match("MESSAGE#224:158", "nwparser.payload", "Heartbeat received from incompatible source%{}", processor_chain([
	dup86,
]));

var msg226 = msg("158", part284);

var part285 = match("MESSAGE#225:159", "nwparser.payload", "Diagnostic Code F%{}", processor_chain([
	dup5,
]));

var msg227 = msg("159", part285);

var part286 = match("MESSAGE#226:160", "nwparser.payload", "Forbidden E-mail attachment altered%{}", processor_chain([
	setc("eventcategory","1203000000"),
]));

var msg228 = msg("160", part286);

var part287 = match("MESSAGE#227:161", "nwparser.payload", "PPPoE PAP Authentication success.%{}", processor_chain([
	dup57,
]));

var msg229 = msg("161", part287);

var part288 = match("MESSAGE#228:162", "nwparser.payload", "PPPoE PAP Authentication Failed. Please verify PPPoE username and password%{}", processor_chain([
	dup32,
]));

var msg230 = msg("162", part288);

var part289 = match("MESSAGE#229:163", "nwparser.payload", "Disconnecting PPPoE due to traffic timeout%{}", processor_chain([
	dup5,
]));

var msg231 = msg("163", part289);

var part290 = match("MESSAGE#230:164", "nwparser.payload", "No response from ISP Disconnecting PPPoE.%{}", processor_chain([
	dup5,
]));

var msg232 = msg("164", part290);

var part291 = match("MESSAGE#231:165", "nwparser.payload", "Backup going Active in preempt mode after reboot%{}", processor_chain([
	dup1,
]));

var msg233 = msg("165", part291);

var part292 = match("MESSAGE#232:166", "nwparser.payload", "Denied TCP connection from LAN%{}", processor_chain([
	dup12,
]));

var msg234 = msg("166", part292);

var part293 = match("MESSAGE#233:167", "nwparser.payload", "Denied UDP packet from LAN%{}", processor_chain([
	dup12,
]));

var msg235 = msg("167", part293);

var part294 = match("MESSAGE#234:168", "nwparser.payload", "Denied ICMP packet from LAN%{}", processor_chain([
	dup12,
]));

var msg236 = msg("168", part294);

var part295 = match("MESSAGE#235:169", "nwparser.payload", "Firewall access from LAN%{}", processor_chain([
	dup1,
]));

var msg237 = msg("169", part295);

var part296 = match("MESSAGE#236:170", "nwparser.payload", "Received a path MTU icmp message from router/gateway%{}", processor_chain([
	dup1,
]));

var msg238 = msg("170", part296);

var part297 = match("MESSAGE#237:171", "nwparser.payload", "Probable TCP FIN scan%{}", processor_chain([
	dup62,
]));

var msg239 = msg("171", part297);

var part298 = match("MESSAGE#238:171:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup87,
]));

var msg240 = msg("171:01", part298);

var part299 = match("MESSAGE#239:171:02", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}:%{dport}", processor_chain([
	dup87,
]));

var msg241 = msg("171:02", part299);

var part300 = match("MESSAGE#240:171:03/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld1}\" sess=%{fld2->} n=%{fld3->} src=%{p0}");

var all38 = all_match({
	processors: [
		part300,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup87,
	]),
});

var msg242 = msg("171:03", all38);

var select70 = linear_select([
	msg239,
	msg240,
	msg241,
	msg242,
]);

var part301 = match("MESSAGE#241:172", "nwparser.payload", "Probable TCP XMAS scan%{}", processor_chain([
	dup62,
]));

var msg243 = msg("172", part301);

var part302 = match("MESSAGE#242:172:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	dup62,
]));

var msg244 = msg("172:01", part302);

var select71 = linear_select([
	msg243,
	msg244,
]);

var part303 = match("MESSAGE#243:173", "nwparser.payload", "Probable TCP NULL scan%{}", processor_chain([
	dup62,
]));

var msg245 = msg("173", part303);

var part304 = match("MESSAGE#244:174", "nwparser.payload", "IPSEC Replay Detected%{}", processor_chain([
	dup59,
]));

var msg246 = msg("174", part304);

var all39 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup175,
		dup79,
	],
	on_success: processor_chain([
		dup59,
	]),
});

var msg247 = msg("174:01", all39);

var all40 = all_match({
	processors: [
		dup44,
		dup179,
		dup36,
		dup178,
	],
	on_success: processor_chain([
		dup12,
	]),
});

var msg248 = msg("174:02", all40);

var all41 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup181,
		dup43,
	],
	on_success: processor_chain([
		dup12,
	]),
});

var msg249 = msg("174:03", all41);

var select72 = linear_select([
	msg246,
	msg247,
	msg248,
	msg249,
]);

var part305 = match("MESSAGE#248:175", "nwparser.payload", "TCP FIN packet dropped%{}", processor_chain([
	dup59,
]));

var msg250 = msg("175", part305);

var part306 = match("MESSAGE#249:175:01", "nwparser.payload", "msg=\"ICMP packet from LAN dropped\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} type=%{type}", processor_chain([
	dup59,
]));

var msg251 = msg("175:01", part306);

var part307 = match("MESSAGE#250:175:02", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr->} dst=%{daddr->} type=%{type->} icmpCode=%{fld3->} npcs=%{info}", processor_chain([
	dup59,
]));

var msg252 = msg("175:02", part307);

var select73 = linear_select([
	msg250,
	msg251,
	msg252,
]);

var part308 = match("MESSAGE#251:176", "nwparser.payload", "Fraudulent Microsoft Certificate Blocked%{}", processor_chain([
	dup87,
]));

var msg253 = msg("176", part308);

var msg254 = msg("177", dup185);

var msg255 = msg("178", dup190);

var msg256 = msg("179", dup185);

var all42 = all_match({
	processors: [
		dup33,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup91,
	]),
});

var msg257 = msg("180", all42);

var all43 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup191,
		dup94,
	],
	on_success: processor_chain([
		dup91,
	]),
});

var msg258 = msg("180:01", all43);

var select74 = linear_select([
	msg257,
	msg258,
]);

var msg259 = msg("181", dup184);

var all44 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup62,
	]),
});

var msg260 = msg("181:01", all44);

var select75 = linear_select([
	msg259,
	msg260,
]);

var msg261 = msg("193", dup228);

var msg262 = msg("194", dup229);

var msg263 = msg("195", dup229);

var part309 = match("MESSAGE#262:196/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{fld2->} dst=%{daddr}:%{fld3->} sport=%{sport->} dport=%{dport->} %{p0}");

var part310 = match("MESSAGE#262:196/1_1", "nwparser.p0", " rcvd=%{rbytes->} cmd=%{p0}");

var select76 = linear_select([
	dup98,
	part310,
]);

var all45 = all_match({
	processors: [
		part309,
		select76,
		dup99,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg264 = msg("196", all45);

var part311 = match("MESSAGE#263:196:01/1_1", "nwparser.p0", "rcvd=%{rbytes->} cmd=%{p0}");

var select77 = linear_select([
	dup98,
	part311,
]);

var all46 = all_match({
	processors: [
		dup95,
		select77,
		dup99,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg265 = msg("196:01", all46);

var select78 = linear_select([
	msg264,
	msg265,
]);

var msg266 = msg("199", dup230);

var msg267 = msg("200", dup226);

var part312 = match("MESSAGE#266:235:02", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} usr=%{username->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup29,
]));

var msg268 = msg("235:02", part312);

var part313 = match("MESSAGE#267:235/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} usr=%{username->} src=%{p0}");

var all47 = all_match({
	processors: [
		part313,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup29,
	]),
});

var msg269 = msg("235", all47);

var msg270 = msg("235:01", dup231);

var select79 = linear_select([
	msg268,
	msg269,
	msg270,
]);

var msg271 = msg("236", dup231);

var msg272 = msg("237", dup230);

var msg273 = msg("238", dup230);

var part314 = match("MESSAGE#272:239", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr->} dst=%{dtransaddr}", processor_chain([
	dup101,
]));

var msg274 = msg("239", part314);

var part315 = match("MESSAGE#273:240", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr->} dst=%{dtransaddr}", processor_chain([
	dup101,
]));

var msg275 = msg("240", part315);

var part316 = match("MESSAGE#274:241", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup70,
]));

var msg276 = msg("241", part316);

var part317 = match("MESSAGE#275:241:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup70,
]));

var msg277 = msg("241:01", part317);

var select80 = linear_select([
	msg276,
	msg277,
]);

var part318 = match("MESSAGE#276:242/1_0", "nwparser.p0", "%{saddr}:%{sport}:: %{p0}");

var part319 = match("MESSAGE#276:242/1_1", "nwparser.p0", "%{saddr}:%{sport->} %{p0}");

var select81 = linear_select([
	part318,
	part319,
	dup35,
]);

var part320 = match("MESSAGE#276:242/3_0", "nwparser.p0", "%{daddr}:%{dport}:: ");

var part321 = match("MESSAGE#276:242/3_1", "nwparser.p0", "%{daddr}:%{dport->} ");

var select82 = linear_select([
	part320,
	part321,
	dup28,
]);

var all48 = all_match({
	processors: [
		dup44,
		select81,
		dup36,
		select82,
	],
	on_success: processor_chain([
		dup70,
	]),
});

var msg278 = msg("242", all48);

var msg279 = msg("252", dup193);

var msg280 = msg("255", dup193);

var msg281 = msg("257", dup193);

var msg282 = msg("261:01", dup232);

var msg283 = msg("261", dup193);

var select83 = linear_select([
	msg282,
	msg283,
]);

var msg284 = msg("262", dup232);

var all49 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg285 = msg("273", all49);

var msg286 = msg("328", dup233);

var msg287 = msg("329", dup226);

var msg288 = msg("346", dup193);

var msg289 = msg("350", dup193);

var msg290 = msg("351", dup193);

var msg291 = msg("352", dup193);

var msg292 = msg("353:01", dup190);

var part322 = match("MESSAGE#291:353", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr->} dst=%{dtransaddr->} dstname=%{shost->} lifeSeconds=%{misc}\"", processor_chain([
	dup5,
]));

var msg293 = msg("353", part322);

var select84 = linear_select([
	msg292,
	msg293,
]);

var part323 = match("MESSAGE#292:354", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=\"%{shost->} lifeSeconds=%{misc}\"", processor_chain([
	dup1,
]));

var msg294 = msg("354", part323);

var msg295 = msg("355", dup194);

var msg296 = msg("355:01", dup193);

var select85 = linear_select([
	msg295,
	msg296,
]);

var msg297 = msg("356", dup195);

var part324 = match("MESSAGE#296:357", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport->} dstname=%{name}", processor_chain([
	dup87,
]));

var msg298 = msg("357", part324);

var part325 = match("MESSAGE#297:357:01", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup87,
]));

var msg299 = msg("357:01", part325);

var select86 = linear_select([
	msg298,
	msg299,
]);

var msg300 = msg("358", dup196);

var part326 = match("MESSAGE#299:371", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr->} dst=%{dtransaddr->} dstname=%{shost}", processor_chain([
	setc("eventcategory","1503000000"),
]));

var msg301 = msg("371", part326);

var msg302 = msg("371:01", dup197);

var select87 = linear_select([
	msg301,
	msg302,
]);

var msg303 = msg("372", dup193);

var msg304 = msg("373", dup195);

var msg305 = msg("401", dup234);

var msg306 = msg("402", dup234);

var msg307 = msg("406", dup196);

var part327 = match("MESSAGE#305:413", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var msg308 = msg("413", part327);

var msg309 = msg("414", dup193);

var msg310 = msg("438", dup235);

var msg311 = msg("439", dup235);

var all50 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1501020000"),
	]),
});

var msg312 = msg("440", all50);

var all51 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1502050000"),
	]),
});

var msg313 = msg("441", all51);

var part328 = match("MESSAGE#311:441:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1}", processor_chain([
	setc("eventcategory","1001020000"),
]));

var msg314 = msg("441:01", part328);

var select88 = linear_select([
	msg313,
	msg314,
]);

var all52 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1501030000"),
	]),
});

var msg315 = msg("442", all52);

var part329 = match("MESSAGE#313:446/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{p0}");

var part330 = match("MESSAGE#313:446/1_0", "nwparser.p0", "%{fld1->} appName=\"%{application}\" n=%{p0}");

var part331 = match("MESSAGE#313:446/1_1", "nwparser.p0", "%{fld1->} n=%{p0}");

var select89 = linear_select([
	part330,
	part331,
]);

var part332 = match("MESSAGE#313:446/2", "nwparser.p0", "%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var all53 = all_match({
	processors: [
		part329,
		select89,
		part332,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup59,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg316 = msg("446", all53);

var part333 = match("MESSAGE#314:477", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} note=\"MAC=%{smacaddr->} HostName:%{hostname}\"", processor_chain([
	dup113,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg317 = msg("477", part333);

var all54 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup29,
	]),
});

var msg318 = msg("509", all54);

var all55 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup103,
	]),
});

var msg319 = msg("520", all55);

var msg320 = msg("522", dup236);

var part334 = match("MESSAGE#318:522:01/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} srcV6=%{saddr_v6->} src= %{p0}");

var part335 = match("MESSAGE#318:522:01/2", "nwparser.p0", "%{}dstV6=%{daddr_v6->} dst= %{p0}");

var all56 = all_match({
	processors: [
		part334,
		dup179,
		part335,
		dup175,
		dup114,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg321 = msg("522:01", all56);

var part336 = match("MESSAGE#319:522:02/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{shost->} dst= %{p0}");

var select90 = linear_select([
	part336,
	dup39,
]);

var all57 = all_match({
	processors: [
		dup38,
		select90,
		dup10,
		dup175,
		dup114,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg322 = msg("522:02", all57);

var select91 = linear_select([
	msg320,
	msg321,
	msg322,
]);

var msg323 = msg("523", dup236);

var all58 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup175,
		dup10,
		dup200,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg324 = msg("524", all58);

var part337 = match("MESSAGE#322:524:01/5_0", "nwparser.p0", "proto=%{protocol->} npcs= %{p0}");

var part338 = match("MESSAGE#322:524:01/5_1", "nwparser.p0", "rule=%{rule->} npcs= %{p0}");

var select92 = linear_select([
	part337,
	part338,
]);

var all59 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup175,
		dup10,
		select92,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg325 = msg("524:01", all59);

var part339 = match("MESSAGE#323:524:02/0", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol}rule=\"%{p0}");

var part340 = match("MESSAGE#323:524:02/1_0", "nwparser.p0", "%{rule}\" note=\"%{rulename}\"%{p0}");

var part341 = match("MESSAGE#323:524:02/1_1", "nwparser.p0", "%{rule}\"%{p0}");

var select93 = linear_select([
	part340,
	part341,
]);

var part342 = match("MESSAGE#323:524:02/2", "nwparser.p0", "%{}fw_action=\"%{action}\"");

var all60 = all_match({
	processors: [
		part339,
		select93,
		part342,
	],
	on_success: processor_chain([
		dup6,
		dup11,
	]),
});

var msg326 = msg("524:02", all60);

var select94 = linear_select([
	msg324,
	msg325,
	msg326,
]);

var msg327 = msg("526", dup237);

var part343 = match("MESSAGE#325:526:01/1_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{fld20->} dst= %{p0}");

var select95 = linear_select([
	dup25,
	part343,
	dup39,
]);

var part344 = match("MESSAGE#325:526:01/3_1", "nwparser.p0", " %{daddr->} ");

var select96 = linear_select([
	dup27,
	part344,
]);

var all61 = all_match({
	processors: [
		dup80,
		select95,
		dup10,
		select96,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg328 = msg("526:01", all61);

var all62 = all_match({
	processors: [
		dup7,
		dup201,
		dup10,
		dup175,
		dup114,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg329 = msg("526:02", all62);

var part345 = match("MESSAGE#327:526:03", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg330 = msg("526:03", part345);

var part346 = match("MESSAGE#328:526:04", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg331 = msg("526:04", part346);

var part347 = match("MESSAGE#329:526:05", "nwparser.payload", "msg=\"%{msg}\" app=%{fld1}n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup11,
]));

var msg332 = msg("526:05", part347);

var select97 = linear_select([
	msg327,
	msg328,
	msg329,
	msg330,
	msg331,
	msg332,
]);

var part348 = match("MESSAGE#330:537:01/4", "nwparser.p0", "%{}proto=%{protocol->} sent=%{sbytes->} rcvd=%{p0}");

var part349 = match("MESSAGE#330:537:01/5_0", "nwparser.p0", "%{rbytes->} vpnpolicy=%{fld3->} ");

var part350 = match("MESSAGE#330:537:01/5_1", "nwparser.p0", "%{rbytes->} ");

var select98 = linear_select([
	part349,
	part350,
]);

var all63 = all_match({
	processors: [
		dup116,
		dup202,
		dup10,
		dup203,
		part348,
		select98,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg333 = msg("537:01", all63);

var part351 = match("MESSAGE#331:537:02/4", "nwparser.p0", "%{}proto=%{protocol->} sent=%{sbytes}");

var all64 = all_match({
	processors: [
		dup116,
		dup202,
		dup10,
		dup203,
		part351,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg334 = msg("537:02", all64);

var select99 = linear_select([
	dup117,
	dup118,
	dup119,
	dup120,
]);

var part352 = match("MESSAGE#332:537:08/3_1", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part353 = match("MESSAGE#332:537:08/3_2", "nwparser.p0", " %{daddr}srcMac=%{p0}");

var select100 = linear_select([
	dup123,
	part352,
	part353,
]);

var part354 = match("MESSAGE#332:537:08/4", "nwparser.p0", "%{} %{smacaddr->} %{p0}");

var select101 = linear_select([
	dup124,
	dup125,
]);

var part355 = match("MESSAGE#332:537:08/6_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{p0}");

var part356 = match("MESSAGE#332:537:08/6_1", "nwparser.p0", "%{sbytes->} spkt=%{p0}");

var select102 = linear_select([
	part355,
	part356,
]);

var part357 = match("MESSAGE#332:537:08/7_0", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} cdur=%{fld7->} fw_action=\"%{action}\" ");

var part358 = match("MESSAGE#332:537:08/7_1", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} cdur=%{fld7->} ");

var part359 = match("MESSAGE#332:537:08/7_2", "nwparser.p0", "%{fld3->} rpkt=%{fld6->} fw_action=\"%{action}\" ");

var part360 = match("MESSAGE#332:537:08/7_3", "nwparser.p0", "%{fld3->} cdur=%{fld7->} ");

var part361 = match("MESSAGE#332:537:08/7_4", "nwparser.p0", "%{fld3}");

var select103 = linear_select([
	part357,
	part358,
	part359,
	part360,
	part361,
]);

var all65 = all_match({
	processors: [
		select99,
		dup204,
		dup205,
		select100,
		part354,
		select101,
		select102,
		select103,
	],
	on_success: processor_chain([
		dup105,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg335 = msg("537:08", all65);

var select104 = linear_select([
	dup118,
	dup117,
	dup119,
	dup120,
]);

var part362 = match("MESSAGE#333:537:09/3_1", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} dstMac=%{p0}");

var part363 = match("MESSAGE#333:537:09/3_2", "nwparser.p0", " %{daddr}dstMac=%{p0}");

var select105 = linear_select([
	dup126,
	part362,
	part363,
]);

var part364 = match("MESSAGE#333:537:09/4", "nwparser.p0", "%{} %{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var select106 = linear_select([
	dup129,
	dup130,
	dup131,
	dup132,
]);

var all66 = all_match({
	processors: [
		select104,
		dup204,
		dup205,
		select105,
		part364,
		dup206,
		select106,
	],
	on_success: processor_chain([
		dup105,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg336 = msg("537:09", all66);

var part365 = match("MESSAGE#334:537:07/0_1", "nwparser.payload", " msg=\"%{event_description}\" app=%{fld51->} sess=\"%{fld4}\" n=%{p0}");

var select107 = linear_select([
	dup117,
	part365,
	dup119,
	dup120,
]);

var part366 = match("MESSAGE#334:537:07/4_0", "nwparser.p0", "srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var part367 = match("MESSAGE#334:537:07/4_1", "nwparser.p0", " srcMac=%{smacaddr->} proto=%{protocol->} sent=%{p0}");

var select108 = linear_select([
	part366,
	part367,
	dup124,
	dup125,
]);

var part368 = match("MESSAGE#334:537:07/6_3", "nwparser.p0", " spkt=%{fld3->} fw_action=\"%{action}\"");

var select109 = linear_select([
	dup129,
	dup130,
	dup131,
	part368,
	dup132,
]);

var all67 = all_match({
	processors: [
		select107,
		dup204,
		dup205,
		dup186,
		select108,
		dup206,
		select109,
	],
	on_success: processor_chain([
		dup105,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg337 = msg("537:07", all67);

var part369 = match("MESSAGE#335:537/1_0", "nwparser.p0", "%{action}\" app=%{fld51->} appName=\"%{application}\"%{p0}");

var part370 = match("MESSAGE#335:537/1_1", "nwparser.p0", "%{action}\"%{p0}");

var select110 = linear_select([
	part369,
	part370,
]);

var part371 = match("MESSAGE#335:537/2", "nwparser.p0", "%{}n=%{fld1->} src= %{p0}");

var part372 = match("MESSAGE#335:537/4_0", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{protocol->} sent=%{p0}");

var part373 = match("MESSAGE#335:537/4_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}: proto=%{protocol->} sent=%{p0}");

var part374 = match("MESSAGE#335:537/4_2", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} sent=%{p0}");

var part375 = match("MESSAGE#335:537/4_3", "nwparser.p0", " %{daddr->} proto=%{protocol->} sent=%{p0}");

var select111 = linear_select([
	part372,
	part373,
	part374,
	part375,
]);

var part376 = match("MESSAGE#335:537/5_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} cdur=%{fld5->} fw_action=\"%{fld6}\"");

var part377 = match("MESSAGE#335:537/5_1", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} fw_action=\"%{fld5}\"");

var part378 = match("MESSAGE#335:537/5_2", "nwparser.p0", "%{sbytes->} spkt=%{fld3}fw_action=\"%{fld4}\"");

var part379 = match("MESSAGE#335:537/5_3", "nwparser.p0", "%{sbytes}rcvd=%{rbytes}");

var part380 = match("MESSAGE#335:537/5_4", "nwparser.p0", "%{sbytes}");

var select112 = linear_select([
	part376,
	part377,
	part378,
	part379,
	part380,
]);

var all68 = all_match({
	processors: [
		dup48,
		select110,
		part371,
		dup202,
		select111,
		select112,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg338 = msg("537", all68);

var part381 = match("MESSAGE#336:537:04/4", "nwparser.p0", "%{} %{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} cdur=%{fld5->} npcs=%{info}");

var all69 = all_match({
	processors: [
		dup133,
		dup180,
		dup10,
		dup207,
		part381,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg339 = msg("537:04", all69);

var part382 = match("MESSAGE#337:537:05/4", "nwparser.p0", "%{} %{protocol->} sent=%{sbytes->} spkt=%{fld3->} cdur=%{p0}");

var part383 = match("MESSAGE#337:537:05/5_0", "nwparser.p0", "%{fld4->} appcat=%{fld5->} appid=%{fld6->} npcs= %{p0}");

var part384 = match("MESSAGE#337:537:05/5_1", "nwparser.p0", "%{fld4->} npcs= %{p0}");

var select113 = linear_select([
	part383,
	part384,
]);

var all70 = all_match({
	processors: [
		dup133,
		dup180,
		dup10,
		dup207,
		part382,
		select113,
		dup90,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg340 = msg("537:05", all70);

var part385 = match("MESSAGE#338:537:10/0", "nwparser.payload", "msg=\"%{event_description}\" sess=%{fld1->} n=%{p0}");

var part386 = match("MESSAGE#338:537:10/4_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} dstMac=%{p0}");

var part387 = match("MESSAGE#338:537:10/4_2", "nwparser.p0", "%{daddr->} dstMac=%{p0}");

var select114 = linear_select([
	dup126,
	part386,
	part387,
]);

var part388 = match("MESSAGE#338:537:10/5", "nwparser.p0", "%{} %{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld10->} rpkt=%{fld11->} %{p0}");

var all71 = all_match({
	processors: [
		part385,
		dup208,
		dup137,
		dup209,
		select114,
		part388,
		dup210,
	],
	on_success: processor_chain([
		dup105,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg341 = msg("537:10", all71);

var part389 = match("MESSAGE#339:537:03/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{p0}");

var part390 = match("MESSAGE#339:537:03/4_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part391 = match("MESSAGE#339:537:03/4_2", "nwparser.p0", "%{daddr->} proto=%{p0}");

var select115 = linear_select([
	dup77,
	part390,
	part391,
]);

var part392 = match("MESSAGE#339:537:03/5", "nwparser.p0", "%{} %{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld10->} rpkt=%{fld11->} %{p0}");

var all72 = all_match({
	processors: [
		part389,
		dup208,
		dup137,
		dup209,
		select115,
		part392,
		dup210,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg342 = msg("537:03", all72);

var part393 = match("MESSAGE#340:537:06/4", "nwparser.p0", "%{} %{protocol->} sent=%{sbytes->} spkt=%{fld3->} npcs=%{info}");

var all73 = all_match({
	processors: [
		dup133,
		dup180,
		dup10,
		dup207,
		part393,
	],
	on_success: processor_chain([
		dup105,
	]),
});

var msg343 = msg("537:06", all73);

var part394 = match("MESSAGE#341:537:11", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2}usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}sent=%{sbytes}rcvd=%{rbytes}spkt=%{fld3}rpkt=%{fld4}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup105,
	dup54,
	dup11,
	dup142,
]));

var msg344 = msg("537:11", part394);

var part395 = match("MESSAGE#342:537:12", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes->} spkt=%{fld3->} rpkt=%{fld4->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup105,
	dup54,
	dup11,
	dup142,
]));

var msg345 = msg("537:12", part395);

var select116 = linear_select([
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

var msg346 = msg("538", dup228);

var msg347 = msg("549", dup226);

var msg348 = msg("557", dup226);

var all74 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1402020200"),
	]),
});

var msg349 = msg("558", all74);

var msg350 = msg("561", dup233);

var msg351 = msg("562", dup233);

var msg352 = msg("563", dup233);

var all75 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		setc("eventcategory","1402020400"),
	]),
});

var msg353 = msg("583", all75);

var part396 = match("MESSAGE#351:597:01", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} type=%{icmptype->} code=%{icmpcode}", processor_chain([
	dup143,
	dup51,
	dup144,
	dup53,
	dup54,
	dup11,
	dup145,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg354 = msg("597:01", part396);

var part397 = match("MESSAGE#352:597:02", "nwparser.payload", "msg=%{msg->} n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} type=%{icmptype->} code=%{icmpcode}", processor_chain([
	dup1,
]));

var msg355 = msg("597:02", part397);

var part398 = match("MESSAGE#353:597:03/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src= %{p0}");

var all76 = all_match({
	processors: [
		part398,
		dup187,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg356 = msg("597:03", all76);

var select117 = linear_select([
	msg354,
	msg355,
	msg356,
]);

var part399 = match("MESSAGE#354:598", "nwparser.payload", "msg=%{msg->} n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} type=%{type->} code=%{code}", processor_chain([
	dup1,
]));

var msg357 = msg("598", part399);

var part400 = match("MESSAGE#355:598:01/2", "nwparser.p0", "%{} %{type->} npcs=%{info}");

var all77 = all_match({
	processors: [
		dup146,
		dup182,
		part400,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg358 = msg("598:01", all77);

var all78 = all_match({
	processors: [
		dup146,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg359 = msg("598:02", all78);

var select118 = linear_select([
	msg357,
	msg358,
	msg359,
]);

var part401 = match("MESSAGE#357:602:01", "nwparser.payload", "msg=\"%{event_description}allowed\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} proto=%{protocol}/%{fld4}", processor_chain([
	dup143,
	dup51,
	dup144,
	dup53,
	dup54,
	dup11,
	dup145,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg360 = msg("602:01", part401);

var msg361 = msg("602:02", dup237);

var all79 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup175,
		dup79,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg362 = msg("602:03", all79);

var select119 = linear_select([
	msg360,
	msg361,
	msg362,
]);

var msg363 = msg("605", dup196);

var all80 = all_match({
	processors: [
		dup147,
		dup211,
		dup149,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup87,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg364 = msg("606", all80);

var part402 = match("MESSAGE#362:608/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} ipscat=%{ipscat->} ipspri=%{p0}");

var part403 = match("MESSAGE#362:608/1_0", "nwparser.p0", "%{fld66->} pktdatId=%{fld11->} n=%{p0}");

var part404 = match("MESSAGE#362:608/1_1", "nwparser.p0", "%{ipspri->} n=%{p0}");

var select120 = linear_select([
	part403,
	part404,
]);

var part405 = match("MESSAGE#362:608/2", "nwparser.p0", "%{fld1->} src=%{saddr}:%{p0}");

var part406 = match("MESSAGE#362:608/3_0", "nwparser.p0", "%{sport}:%{sinterface->} dst=%{p0}");

var part407 = match("MESSAGE#362:608/3_1", "nwparser.p0", "%{sport->} dst=%{p0}");

var select121 = linear_select([
	part406,
	part407,
]);

var part408 = match("MESSAGE#362:608/4", "nwparser.p0", "%{daddr}:%{p0}");

var part409 = match("MESSAGE#362:608/5_0", "nwparser.p0", "%{dport}:%{dinterface->} proto=%{protocol->} fw_action=\"%{fld2}\"");

var part410 = match("MESSAGE#362:608/5_1", "nwparser.p0", "%{dport}:%{dinterface}");

var part411 = match("MESSAGE#362:608/5_2", "nwparser.p0", "%{dport}");

var select122 = linear_select([
	part409,
	part410,
	part411,
]);

var all81 = all_match({
	processors: [
		part402,
		select120,
		part405,
		select121,
		part408,
		select122,
	],
	on_success: processor_chain([
		dup1,
		dup37,
	]),
});

var msg365 = msg("608", all81);

var msg366 = msg("616", dup194);

var msg367 = msg("658", dup190);

var msg368 = msg("710", dup212);

var msg369 = msg("712:02", dup238);

var msg370 = msg("712", dup212);

var all82 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup191,
		dup94,
	],
	on_success: processor_chain([
		dup150,
	]),
});

var msg371 = msg("712:01", all82);

var select123 = linear_select([
	msg369,
	msg370,
	msg371,
]);

var part412 = match("MESSAGE#369:713:01", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{fld2->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld3->} note=%{info}", processor_chain([
	dup5,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg372 = msg("713:01", part412);

var msg373 = msg("713:04", dup238);

var msg374 = msg("713:02", dup212);

var part413 = match("MESSAGE#372:713:03", "nwparser.payload", "msg=\"%{event_description}dropped\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=\"%{action}\" npcs=%{info}", processor_chain([
	dup5,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg375 = msg("713:03", part413);

var select124 = linear_select([
	msg372,
	msg373,
	msg374,
	msg375,
]);

var part414 = match("MESSAGE#373:760", "nwparser.payload", "msg=\"%{event_description}dropped\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} note=%{info}", processor_chain([
	dup113,
	dup51,
	dup52,
	dup53,
	dup54,
	dup11,
	dup55,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg376 = msg("760", part414);

var part415 = match("MESSAGE#374:760:01/0", "nwparser.payload", "msg=\"%{event_description}dropped\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part416 = match("MESSAGE#374:760:01/4", "nwparser.p0", "%{} %{action->} npcs=%{info}");

var all83 = all_match({
	processors: [
		part415,
		dup174,
		dup10,
		dup191,
		part416,
	],
	on_success: processor_chain([
		dup113,
		dup51,
		dup52,
		dup53,
		dup54,
		dup11,
		dup55,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg377 = msg("760:01", all83);

var select125 = linear_select([
	msg376,
	msg377,
]);

var msg378 = msg("766", dup216);

var msg379 = msg("860", dup216);

var msg380 = msg("860:01", dup217);

var select126 = linear_select([
	msg379,
	msg380,
]);

var part417 = match("MESSAGE#378:866/0", "nwparser.payload", "msg=\"%{msg}\" n=%{p0}");

var part418 = match("MESSAGE#378:866/1_0", "nwparser.p0", "%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\" ");

var part419 = match("MESSAGE#378:866/1_1", "nwparser.p0", "%{ntype->} ");

var select127 = linear_select([
	part418,
	part419,
]);

var all84 = all_match({
	processors: [
		part417,
		select127,
	],
	on_success: processor_chain([
		dup5,
		dup37,
	]),
});

var msg381 = msg("866", all84);

var msg382 = msg("866:01", dup217);

var select128 = linear_select([
	msg381,
	msg382,
]);

var msg383 = msg("867", dup216);

var msg384 = msg("867:01", dup217);

var select129 = linear_select([
	msg383,
	msg384,
]);

var part420 = match("MESSAGE#382:882", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol}", processor_chain([
	dup1,
]));

var msg385 = msg("882", part420);

var part421 = match("MESSAGE#383:882:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} npcs=%{info}", processor_chain([
	dup1,
]));

var msg386 = msg("882:01", part421);

var select130 = linear_select([
	msg385,
	msg386,
]);

var part422 = match("MESSAGE#384:888", "nwparser.payload", "msg=\"%{reason};%{action}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}", processor_chain([
	dup159,
]));

var msg387 = msg("888", part422);

var part423 = match("MESSAGE#385:888:01", "nwparser.payload", "msg=\"%{reason};%{action}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} note=%{fld3->} npcs=%{info}", processor_chain([
	dup159,
]));

var msg388 = msg("888:01", part423);

var select131 = linear_select([
	msg387,
	msg388,
]);

var all85 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup159,
	]),
});

var msg389 = msg("892", all85);

var msg390 = msg("904", dup216);

var msg391 = msg("905", dup216);

var msg392 = msg("906", dup216);

var msg393 = msg("907", dup216);

var select132 = linear_select([
	dup73,
	dup138,
]);

var all86 = all_match({
	processors: [
		dup160,
		select132,
		dup10,
		dup211,
		dup161,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup70,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg394 = msg("908", all86);

var msg395 = msg("909", dup216);

var msg396 = msg("914", dup218);

var part424 = match("MESSAGE#394:931", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup64,
]));

var msg397 = msg("931", part424);

var msg398 = msg("657", dup218);

var all87 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var msg399 = msg("657:01", all87);

var select133 = linear_select([
	msg398,
	msg399,
]);

var msg400 = msg("403", dup197);

var msg401 = msg("534", dup176);

var msg402 = msg("994", dup219);

var part425 = match("MESSAGE#400:243", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} proto=%{protocol}", processor_chain([
	dup1,
	dup23,
]));

var msg403 = msg("243", part425);

var msg404 = msg("995", dup176);

var part426 = match("MESSAGE#402:997", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface}:%{fld3->} dst=%{daddr}:%{dport}:%{dinterface}:%{fld4->} note=\"%{info}\"", processor_chain([
	dup1,
	dup51,
	dup53,
	dup54,
	dup11,
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg405 = msg("997", part426);

var msg406 = msg("998", dup219);

var part427 = match("MESSAGE#405:998:01", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld3->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup105,
	dup11,
]));

var msg407 = msg("998:01", part427);

var select134 = linear_select([
	msg406,
	msg407,
]);

var msg408 = msg("1110", dup220);

var msg409 = msg("565", dup220);

var part428 = match("MESSAGE#408:404", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup54,
]));

var msg410 = msg("404", part428);

var select135 = linear_select([
	dup148,
	dup50,
]);

var part429 = match("MESSAGE#409:267:01/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} note=\"%{fld3}\" fw_action=\"%{action}\"");

var all88 = all_match({
	processors: [
		dup81,
		select135,
		part429,
	],
	on_success: processor_chain([
		dup105,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg411 = msg("267:01", all88);

var part430 = match("MESSAGE#410:267", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}", processor_chain([
	dup1,
	dup54,
]));

var msg412 = msg("267", part430);

var select136 = linear_select([
	msg411,
	msg412,
]);

var part431 = match("MESSAGE#411:263", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} proto=%{protocol}", processor_chain([
	dup1,
	dup23,
]));

var msg413 = msg("263", part431);

var part432 = match("MESSAGE#412:264", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup103,
	dup11,
]));

var msg414 = msg("264", part432);

var msg415 = msg("412", dup197);

var part433 = match("MESSAGE#415:793", "nwparser.payload", "msg=\"%{msg}\" af_polid=%{fld1->} af_policy=\"%{fld2}\" af_type=\"%{fld3}\" af_service=\"%{fld4}\" af_action=\"%{fld5}\" n=%{fld6->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{shost->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{dhost}", processor_chain([
	dup1,
	dup23,
]));

var msg416 = msg("793", part433);

var part434 = match("MESSAGE#416:805", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} if=%{fld2->} ucastRx=%{fld3->} bcastRx=%{fld4->} bytesRx=%{rbytes->} ucastTx=%{fld5->} bcastTx=%{fld6->} bytesTx=%{sbytes}", processor_chain([
	dup1,
	dup23,
]));

var msg417 = msg("805", part434);

var part435 = match("MESSAGE#417:809", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup162,
	dup11,
]));

var msg418 = msg("809", part435);

var part436 = match("MESSAGE#418:809:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} fw_action=\"%{action}\"", processor_chain([
	dup162,
	dup11,
]));

var msg419 = msg("809:01", part436);

var select137 = linear_select([
	msg418,
	msg419,
]);

var msg420 = msg("935", dup218);

var msg421 = msg("614", dup221);

var part437 = match("MESSAGE#421:748/0", "nwparser.payload", "msg=\"%{event_description}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var all89 = all_match({
	processors: [
		part437,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup58,
		dup37,
	]),
});

var msg422 = msg("748", all89);

var part438 = match("MESSAGE#422:794/0", "nwparser.payload", "msg=\"%{event_description}\" sid=%{sid->} spycat=%{fld1->} spypri=%{fld2->} pktdatId=%{fld3->} n=%{fld4->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part439 = match("MESSAGE#422:794/1_0", "nwparser.p0", "%{protocol}/%{fld5->} fw_action=\"%{p0}");

var select138 = linear_select([
	part439,
	dup111,
]);

var all90 = all_match({
	processors: [
		part438,
		select138,
		dup112,
	],
	on_success: processor_chain([
		dup163,
		dup37,
	]),
});

var msg423 = msg("794", all90);

var msg424 = msg("1086", dup221);

var part440 = match("MESSAGE#424:1430", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup163,
	dup37,
]));

var msg425 = msg("1430", part440);

var msg426 = msg("1149", dup221);

var msg427 = msg("1159", dup221);

var part441 = match("MESSAGE#427:1195", "nwparser.payload", "n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup163,
	dup37,
]));

var msg428 = msg("1195", part441);

var part442 = match("MESSAGE#428:1195:01", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1}", processor_chain([
	dup163,
	dup37,
]));

var msg429 = msg("1195:01", part442);

var select139 = linear_select([
	msg428,
	msg429,
]);

var part443 = match("MESSAGE#429:1226", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup37,
]));

var msg430 = msg("1226", part443);

var part444 = match("MESSAGE#430:1222", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport->} note=\"%{fld3}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup37,
]));

var msg431 = msg("1222", part444);

var part445 = match("MESSAGE#431:1154", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{shost->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{dhost}", processor_chain([
	dup1,
	dup23,
]));

var msg432 = msg("1154", part445);

var part446 = match("MESSAGE#432:1154:01/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{p0}");

var all91 = all_match({
	processors: [
		part446,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
		dup23,
	]),
});

var msg433 = msg("1154:01", all91);

var part447 = match("MESSAGE#433:1154:02", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=\"%{fld1}\" appid%{fld2->} catid=%{fld3->} sess=\"%{fld4}\" n=%{fld5->} usr=\"%{username}\" src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup164,
	dup11,
]));

var msg434 = msg("1154:02", part447);

var part448 = match("MESSAGE#434:1154:03/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=\"%{fld1}\" appid=%{fld2->} catid=%{fld3->} n=%{fld4->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var select140 = linear_select([
	dup123,
	dup49,
]);

var part449 = match("MESSAGE#434:1154:03/2", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} rule=\"%{rule}\" fw_action=\"%{action}\"");

var all92 = all_match({
	processors: [
		part448,
		select140,
		part449,
	],
	on_success: processor_chain([
		dup164,
		dup11,
	]),
});

var msg435 = msg("1154:03", all92);

var select141 = linear_select([
	msg432,
	msg433,
	msg434,
	msg435,
]);

var part450 = match("MESSAGE#435:msg", "nwparser.payload", "msg=\"%{msg}\" src=%{stransaddr->} dst=%{dtransaddr->} %{result}", processor_chain([
	dup165,
]));

var msg436 = msg("msg", part450);

var part451 = match("MESSAGE#436:src", "nwparser.payload", "src=%{stransaddr->} dst=%{dtransaddr->} %{msg}", processor_chain([
	dup165,
]));

var msg437 = msg("src", part451);

var all93 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup175,
		dup10,
		dup200,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg438 = msg("1235", all93);

var part452 = match("MESSAGE#438:1197/4", "nwparser.p0", "%{}\"%{fld3->} Protocol:%{protocol}\" npcs=%{info}");

var all94 = all_match({
	processors: [
		dup7,
		dup177,
		dup10,
		dup191,
		part452,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg439 = msg("1197", all94);

var part453 = match("MESSAGE#439:1199/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3->} sess=%{fld1->} n=%{fld2->} src=%{p0}");

var all95 = all_match({
	processors: [
		part453,
		dup177,
		dup166,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg440 = msg("1199", all95);

var part454 = match("MESSAGE#440:1199:01", "nwparser.payload", "msg=\"Responder from country blocked: Responder IP:%{fld1}Country Name:%{location_country}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup167,
	dup11,
]));

var msg441 = msg("1199:01", part454);

var part455 = match("MESSAGE#441:1199:02", "nwparser.payload", "msg=\"Responder from country blocked: Responder IP:%{fld1}Country Name:%{location_country}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}rule=\"%{rule}\" fw_action=\"%{action}\"", processor_chain([
	dup167,
	dup11,
]));

var msg442 = msg("1199:02", part455);

var select142 = linear_select([
	msg440,
	msg441,
	msg442,
]);

var part456 = match("MESSAGE#442:1155/0", "nwparser.payload", "msg=\"%{msg}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} catid=%{fld3->} sess=%{fld4->} n=%{fld5->} src=%{p0}");

var all96 = all_match({
	processors: [
		part456,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg443 = msg("1155", all96);

var part457 = match("MESSAGE#443:1155:01", "nwparser.payload", "msg=\"%{action}\" sid=%{sid->} appcat=%{fld1->} appid=%{fld2->} n=%{fld3->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost}", processor_chain([
	dup105,
]));

var msg444 = msg("1155:01", part457);

var select143 = linear_select([
	msg443,
	msg444,
]);

var all97 = all_match({
	processors: [
		dup168,
		dup201,
		dup166,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg445 = msg("1198", all97);

var all98 = all_match({
	processors: [
		dup7,
		dup177,
		dup166,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg446 = msg("714", all98);

var msg447 = msg("709", dup239);

var msg448 = msg("1005", dup239);

var msg449 = msg("1003", dup239);

var msg450 = msg("1007", dup240);

var part458 = match("MESSAGE#450:1008", "nwparser.payload", "msg=\"%{msg}\" sess=\"%{fld1}\" dur=%{duration->} n=%{fld2->} usr=\"%{username}\" src=%{saddr}::%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} note=\"%{rulename}\" fw_action=\"%{action}\"", processor_chain([
	dup103,
	dup11,
]));

var msg451 = msg("1008", part458);

var msg452 = msg("708", dup240);

var all99 = all_match({
	processors: [
		dup168,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg453 = msg("1201", all99);

var msg454 = msg("1201:01", dup240);

var select144 = linear_select([
	msg453,
	msg454,
]);

var msg455 = msg("654", dup222);

var msg456 = msg("670", dup222);

var msg457 = msg("884", dup240);

var part459 = match("MESSAGE#457:1153", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{protocol->} rcvd=%{rbytes->} note=\"%{info}\"", processor_chain([
	dup1,
]));

var msg458 = msg("1153", part459);

var part460 = match("MESSAGE#458:1153:01/0_0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld1->} sess=%{fld2->} n=%{p0}");

var part461 = match("MESSAGE#458:1153:01/0_1", "nwparser.payload", " msg=\"%{event_description}\" sess=%{fld2->} n=%{p0}");

var part462 = match("MESSAGE#458:1153:01/0_2", "nwparser.payload", " msg=\"%{event_description}\" n=%{p0}");

var select145 = linear_select([
	part460,
	part461,
	part462,
]);

var part463 = match("MESSAGE#458:1153:01/1", "nwparser.p0", "%{fld3->} usr=\"%{username}\" src=%{p0}");

var part464 = match("MESSAGE#458:1153:01/2_0", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface}:%{shost->} dst= %{p0}");

var select146 = linear_select([
	part464,
	dup25,
]);

var part465 = match("MESSAGE#458:1153:01/4_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}srcMac= %{p0}");

var part466 = match("MESSAGE#458:1153:01/4_1", "nwparser.p0", "%{daddr}:%{dport}srcMac= %{p0}");

var part467 = match("MESSAGE#458:1153:01/4_2", "nwparser.p0", "%{daddr}srcMac= %{p0}");

var select147 = linear_select([
	part465,
	part466,
	part467,
]);

var part468 = match("MESSAGE#458:1153:01/5", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} %{p0}");

var part469 = match("MESSAGE#458:1153:01/6_0", "nwparser.p0", "sent=%{sbytes}rcvd=%{rbytes->} ");

var part470 = match("MESSAGE#458:1153:01/6_1", "nwparser.p0", "type=%{fld4->} icmpCode=%{fld5->} rcvd=%{rbytes->} ");

var part471 = match("MESSAGE#458:1153:01/6_2", "nwparser.p0", "rcvd=%{rbytes->} ");

var select148 = linear_select([
	part469,
	part470,
	part471,
]);

var all100 = all_match({
	processors: [
		select145,
		part463,
		select146,
		dup10,
		select147,
		part468,
		select148,
	],
	on_success: processor_chain([
		dup1,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg459 = msg("1153:01", all100);

var part472 = match("MESSAGE#459:1153:02/0", "nwparser.payload", "msg=\"%{event_description}\" %{p0}");

var part473 = match("MESSAGE#459:1153:02/1_0", "nwparser.p0", "app=%{fld1->} n=%{fld2->} src=%{p0}");

var part474 = match("MESSAGE#459:1153:02/1_1", "nwparser.p0", " n=%{fld2->} src=%{p0}");

var select149 = linear_select([
	part473,
	part474,
]);

var part475 = match("MESSAGE#459:1153:02/2", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{sbytes->} rcvd=%{rbytes}");

var all101 = all_match({
	processors: [
		part472,
		select149,
		part475,
	],
	on_success: processor_chain([
		dup1,
		dup11,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var msg460 = msg("1153:02", all101);

var select150 = linear_select([
	msg458,
	msg459,
	msg460,
]);

var part476 = match("MESSAGE#460:1107", "nwparser.payload", "msg=\"%{msg}\"%{space}n=%{fld1}", processor_chain([
	dup1,
]));

var msg461 = msg("1107", part476);

var part477 = match("MESSAGE#461:1220/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{p0}");

var part478 = match("MESSAGE#461:1220/1_0", "nwparser.p0", "%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part479 = match("MESSAGE#461:1220/1_1", "nwparser.p0", "%{fld2}src=%{saddr}:%{sport->} dst=%{p0}");

var select151 = linear_select([
	part478,
	part479,
]);

var all102 = all_match({
	processors: [
		part477,
		select151,
		dup10,
		dup223,
		dup171,
	],
	on_success: processor_chain([
		dup159,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg462 = msg("1220", all102);

var all103 = all_match({
	processors: [
		dup147,
		dup223,
		dup171,
	],
	on_success: processor_chain([
		dup159,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg463 = msg("1230", all103);

var part480 = match("MESSAGE#463:1231", "nwparser.payload", "msg=\"%{msg}\"%{space}n=%{fld1->} note=\"%{info}\"", processor_chain([
	dup1,
]));

var msg464 = msg("1231", part480);

var part481 = match("MESSAGE#464:1233", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup167,
	dup11,
]));

var msg465 = msg("1233", part481);

var part482 = match("MESSAGE#465:1079/0", "nwparser.payload", "msg=\"User%{username}log%{p0}");

var part483 = match("MESSAGE#465:1079/1_0", "nwparser.p0", "in%{p0}");

var part484 = match("MESSAGE#465:1079/1_1", "nwparser.p0", "out%{p0}");

var select152 = linear_select([
	part483,
	part484,
]);

var part485 = match("MESSAGE#465:1079/2", "nwparser.p0", "\"%{p0}");

var part486 = match("MESSAGE#465:1079/3_0", "nwparser.p0", "dur=%{duration->} %{space}n=%{fld1}");

var part487 = match("MESSAGE#465:1079/3_1", "nwparser.p0", "sess=\"%{fld2}\" n=%{fld1->} ");

var part488 = match("MESSAGE#465:1079/3_2", "nwparser.p0", "n=%{fld1}");

var select153 = linear_select([
	part486,
	part487,
	part488,
]);

var all104 = all_match({
	processors: [
		part482,
		select152,
		part485,
		select153,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg466 = msg("1079", all104);

var part489 = match("MESSAGE#466:1079:01", "nwparser.payload", "msg=\"Client%{username}is assigned IP:%{hostip}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
]));

var msg467 = msg("1079:01", part489);

var part490 = match("MESSAGE#467:1079:02", "nwparser.payload", "msg=\"destination for %{daddr->} is not allowed by access control\" n=%{fld2}", processor_chain([
	dup1,
	dup11,
	setc("event_description","destination is not allowed by access control"),
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg468 = msg("1079:02", part490);

var part491 = match("MESSAGE#468:1079:03", "nwparser.payload", "msg=\"SSLVPN Client %{username->} matched device profile Default Device Profile for Windows\" n=%{fld2}", processor_chain([
	dup1,
	dup11,
	setc("event_description","SSLVPN Client matched device profile Default Device Profile for Windows"),
	dup17,
	dup18,
	dup19,
	dup20,
	dup21,
]));

var msg469 = msg("1079:03", part491);

var select154 = linear_select([
	msg466,
	msg467,
	msg468,
	msg469,
]);

var part492 = match("MESSAGE#469:1080/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} usr=\"%{username}\" src= %{p0}");

var part493 = match("MESSAGE#469:1080/1_1", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var select155 = linear_select([
	dup73,
	part493,
]);

var select156 = linear_select([
	dup77,
	dup78,
]);

var part494 = match("MESSAGE#469:1080/4", "nwparser.p0", "%{} %{protocol}");

var all105 = all_match({
	processors: [
		part492,
		select155,
		dup10,
		select156,
		part494,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var msg470 = msg("1080", all105);

var part495 = match("MESSAGE#470:580", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{protocol->} note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg471 = msg("580", part495);

var part496 = match("MESSAGE#471:1369/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{p0}");

var all106 = all_match({
	processors: [
		part496,
		dup224,
		dup112,
	],
	on_success: processor_chain([
		dup70,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg472 = msg("1369", all106);

var all107 = all_match({
	processors: [
		dup147,
		dup211,
		dup149,
		dup224,
		dup112,
	],
	on_success: processor_chain([
		dup70,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg473 = msg("1370", all107);

var all108 = all_match({
	processors: [
		dup147,
		dup211,
		dup161,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup70,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg474 = msg("1371", all108);

var part497 = match("MESSAGE#474:1387/1_1", "nwparser.p0", "%{saddr}:%{sport}: dst=%{p0}");

var select157 = linear_select([
	dup138,
	part497,
]);

var all109 = all_match({
	processors: [
		dup160,
		select157,
		dup10,
		dup211,
		dup161,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup159,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg475 = msg("1387", all109);

var part498 = match("MESSAGE#475:1391/0", "nwparser.payload", "pktdatId=%{fld1}pktdatNum=\"%{fld2}\" pktdatEnc=\"%{fld3}\" n=%{fld4}src=%{p0}");

var part499 = match("MESSAGE#475:1391/1_1", "nwparser.p0", "%{saddr}:%{sport}dst=%{p0}");

var select158 = linear_select([
	dup69,
	part499,
]);

var part500 = match("MESSAGE#475:1391/2_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost}");

var part501 = match("MESSAGE#475:1391/2_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}");

var part502 = match("MESSAGE#475:1391/2_2", "nwparser.p0", "%{daddr}:%{dport}");

var select159 = linear_select([
	part500,
	part501,
	part502,
]);

var all110 = all_match({
	processors: [
		part498,
		select158,
		select159,
	],
	on_success: processor_chain([
		dup1,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg476 = msg("1391", all110);

var part503 = match("MESSAGE#476:1253", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld1}appName=\"%{application}\" n=%{fld2}src=%{saddr}:%{sport}:%{sinterface}dst=%{daddr}:%{dport}:%{dinterface}srcMac=%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg477 = msg("1253", part503);

var part504 = match("MESSAGE#477:1009", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2}note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup5,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg478 = msg("1009", part504);

var part505 = match("MESSAGE#478:910/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld2}appName=\"%{application}\" n=%{fld3}src=%{saddr}:%{sport}:%{sinterface}dst=%{p0}");

var part506 = match("MESSAGE#478:910/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost}srcMac=%{p0}");

var part507 = match("MESSAGE#478:910/1_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}srcMac=%{p0}");

var select160 = linear_select([
	part506,
	part507,
]);

var part508 = match("MESSAGE#478:910/2", "nwparser.p0", "%{smacaddr}dstMac=%{dmacaddr}proto=%{protocol}fw_action=\"%{action}\"");

var all111 = all_match({
	processors: [
		part505,
		select160,
		part508,
	],
	on_success: processor_chain([
		dup5,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg479 = msg("910", all111);

var part509 = match("MESSAGE#479:m:01", "nwparser.payload", "m=%{id1}msg=\"%{event_description}\" n=%{fld2}if=%{interface}ucastRx=%{fld3}bcastRx=%{fld4}bytesRx=%{rbytes}ucastTx=%{fld5}bcastTx=%{fld6}bytesTx=%{sbytes}", processor_chain([
	dup1,
	dup54,
	dup17,
	dup82,
	dup19,
	dup21,
	dup37,
]));

var msg480 = msg("m:01", part509);

var part510 = match("MESSAGE#480:1011", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1}note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg481 = msg("1011", part510);

var part511 = match("MESSAGE#481:609", "nwparser.payload", "msg=\"%{event_description}\" sid=%{sid->} ipscat=\"%{fld3}\" ipspri=%{fld4->} pktdatId=%{fld5->} n=%{fld6->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} proto=%{protocol->} fw_action=\"%{action}\"", processor_chain([
	dup164,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg482 = msg("609", part511);

var msg483 = msg("796", dup225);

var part512 = match("MESSAGE#483:880", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} note=\"%{info}\" fw_action=\"%{action}\"", processor_chain([
	dup70,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg484 = msg("880", part512);

var part513 = match("MESSAGE#484:1309", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup159,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var msg485 = msg("1309", part513);

var msg486 = msg("1310", dup225);

var part514 = match("MESSAGE#486:1232/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} note=\"%{p0}");

var part515 = match("MESSAGE#486:1232/1_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} note=\"%{p0}");

var select161 = linear_select([
	part514,
	part515,
]);

var part516 = match("MESSAGE#486:1232/2", "nwparser.p0", "%{info}\" fw_action=\"%{action}\"");

var all112 = all_match({
	processors: [
		dup81,
		select161,
		part516,
	],
	on_success: processor_chain([
		dup1,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
	]),
});

var msg487 = msg("1232", all112);

var part517 = match("MESSAGE#487:1447/0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld1->} appName=\"%{application}\" n=%{fld2->} srcV6=%{saddr_v6->} src=%{saddr}:%{sport}:%{sinterface->} dstV6=%{daddr_v6->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var all113 = all_match({
	processors: [
		part517,
		dup199,
		dup112,
	],
	on_success: processor_chain([
		dup159,
		dup54,
		dup17,
		dup82,
		dup19,
		dup20,
		dup21,
		dup37,
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
		"1079": select154,
		"108": msg167,
		"1080": msg470,
		"1086": msg424,
		"109": msg168,
		"11": msg10,
		"110": msg169,
		"1107": msg461,
		"111": select66,
		"1110": msg408,
		"112": msg172,
		"113": msg173,
		"114": msg174,
		"1149": msg426,
		"115": select67,
		"1153": select150,
		"1154": select141,
		"1155": select143,
		"1159": msg427,
		"116": msg177,
		"117": msg178,
		"118": msg179,
		"119": msg180,
		"1195": select139,
		"1197": msg439,
		"1198": msg445,
		"1199": select142,
		"12": select4,
		"120": msg181,
		"1201": select144,
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
		"139": select68,
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
		"157": select69,
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
		"171": select70,
		"172": select71,
		"173": msg245,
		"174": select72,
		"175": select73,
		"176": msg253,
		"177": msg254,
		"178": msg255,
		"179": msg256,
		"18": msg23,
		"180": select74,
		"181": select75,
		"19": msg24,
		"193": msg261,
		"194": msg262,
		"195": msg263,
		"196": select78,
		"199": msg266,
		"20": msg25,
		"200": msg267,
		"21": msg26,
		"22": msg27,
		"23": select10,
		"235": select79,
		"236": msg271,
		"237": msg272,
		"238": msg273,
		"239": msg274,
		"24": select11,
		"240": msg275,
		"241": select80,
		"242": msg278,
		"243": msg403,
		"25": msg34,
		"252": msg279,
		"255": msg280,
		"257": msg281,
		"26": msg35,
		"261": select83,
		"262": msg284,
		"263": msg413,
		"264": msg414,
		"267": select136,
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
		"35": select19,
		"350": msg289,
		"351": msg290,
		"352": msg291,
		"353": select84,
		"354": msg294,
		"355": select85,
		"356": msg297,
		"357": select86,
		"358": msg300,
		"36": select23,
		"37": select27,
		"371": select87,
		"372": msg303,
		"373": msg304,
		"38": select30,
		"39": msg67,
		"4": msg1,
		"40": msg68,
		"401": msg305,
		"402": msg306,
		"403": msg400,
		"404": msg410,
		"406": msg307,
		"41": select31,
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
		"441": select88,
		"442": msg315,
		"446": msg316,
		"45": select32,
		"46": select33,
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
		"522": select91,
		"523": msg323,
		"524": select94,
		"526": select97,
		"53": msg88,
		"534": msg401,
		"537": select116,
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
		"597": select117,
		"598": select118,
		"6": select3,
		"60": msg90,
		"602": select119,
		"605": msg363,
		"606": msg364,
		"608": msg365,
		"609": msg482,
		"61": msg91,
		"614": msg421,
		"616": msg366,
		"62": msg92,
		"63": select34,
		"64": msg95,
		"65": msg96,
		"654": msg455,
		"657": select133,
		"658": msg367,
		"66": msg97,
		"67": select35,
		"670": msg456,
		"68": msg100,
		"69": msg101,
		"7": msg6,
		"70": select37,
		"708": msg452,
		"709": msg447,
		"710": msg368,
		"712": select123,
		"713": select124,
		"714": msg446,
		"72": select38,
		"73": msg106,
		"74": msg107,
		"748": msg422,
		"75": msg108,
		"76": msg109,
		"760": select125,
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
		"809": select137,
		"81": msg114,
		"82": select39,
		"83": select40,
		"84": msg122,
		"860": select126,
		"866": select128,
		"867": select129,
		"87": select42,
		"88": select43,
		"880": msg484,
		"882": select130,
		"884": msg457,
		"888": select131,
		"89": select45,
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
		"97": select52,
		"98": select65,
		"986": msg155,
		"99": msg158,
		"994": msg402,
		"995": msg404,
		"997": msg405,
		"998": select134,
		"m": msg480,
		"msg": msg436,
		"src": msg437,
	}),
]);

var part518 = match("MESSAGE#14:14:01/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part519 = match("MESSAGE#14:14:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst= %{p0}");

var part520 = match("MESSAGE#14:14:01/1_1", "nwparser.p0", " %{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part521 = match("MESSAGE#14:14:01/2", "nwparser.p0", "%{} %{p0}");

var part522 = match("MESSAGE#28:23:01/1_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} %{p0}");

var part523 = match("MESSAGE#28:23:01/1_1", "nwparser.p0", "%{daddr->} %{p0}");

var part524 = match("MESSAGE#38:29:01/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part525 = match("MESSAGE#38:29:01/1_1", "nwparser.p0", " %{saddr->} dst= %{p0}");

var part526 = match("MESSAGE#38:29:01/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} ");

var part527 = match("MESSAGE#38:29:01/3_1", "nwparser.p0", "%{daddr->} ");

var part528 = match("MESSAGE#40:30:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld->} src=%{p0}");

var part529 = match("MESSAGE#49:33:01/0", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{p0}");

var part530 = match("MESSAGE#54:36:01/2_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} %{p0}");

var part531 = match("MESSAGE#54:36:01/2_1", "nwparser.p0", "%{saddr->} %{p0}");

var part532 = match("MESSAGE#54:36:01/3", "nwparser.p0", "%{}dst= %{p0}");

var part533 = match("MESSAGE#55:36:02/0", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part534 = match("MESSAGE#55:36:02/1_1", "nwparser.p0", "%{saddr->} dst= %{p0}");

var part535 = match("MESSAGE#57:37:01/1_1", "nwparser.p0", "n=%{fld1->} src=%{p0}");

var part536 = match("MESSAGE#59:37:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto= %{p0}");

var part537 = match("MESSAGE#59:37:03/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} proto= %{p0}");

var part538 = match("MESSAGE#59:37:03/4", "nwparser.p0", "%{} %{protocol->} npcs=%{info}");

var part539 = match("MESSAGE#62:38:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src= %{p0}");

var part540 = match("MESSAGE#62:38:01/5_1", "nwparser.p0", "rule=%{rule->} ");

var part541 = match("MESSAGE#63:38:02/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} type= %{p0}");

var part542 = match("MESSAGE#63:38:02/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} type= %{p0}");

var part543 = match("MESSAGE#64:38:03/0", "nwparser.payload", "msg=\"%{p0}");

var part544 = match("MESSAGE#64:38:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part545 = match("MESSAGE#64:38:03/3_1", "nwparser.p0", "%{daddr->} srcMac=%{p0}");

var part546 = match("MESSAGE#135:97:01/0", "nwparser.payload", "n=%{fld1->} src= %{p0}");

var part547 = match("MESSAGE#135:97:01/7_1", "nwparser.p0", "dstname=%{name->} ");

var part548 = match("MESSAGE#137:97:03/0", "nwparser.payload", "sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part549 = match("MESSAGE#140:97:06/1_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost}dst=%{p0}");

var part550 = match("MESSAGE#140:97:06/1_1", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}dst=%{p0}");

var part551 = match("MESSAGE#145:98/2_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} %{p0}");

var part552 = match("MESSAGE#145:98/3_0", "nwparser.p0", "proto=%{protocol->} sent=%{sbytes->} fw_action=\"%{action}\"");

var part553 = match("MESSAGE#147:98:01/4_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface}:%{shost->} dst=%{p0}");

var part554 = match("MESSAGE#147:98:01/4_2", "nwparser.p0", "%{saddr}dst=%{p0}");

var part555 = match("MESSAGE#147:98:01/6_1", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} %{p0}");

var part556 = match("MESSAGE#147:98:01/6_2", "nwparser.p0", " %{daddr->} %{p0}");

var part557 = match("MESSAGE#148:98:06/5_2", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} proto=%{p0}");

var part558 = match("MESSAGE#148:98:06/5_3", "nwparser.p0", " %{daddr}:%{dport}:%{dinterface->} proto=%{p0}");

var part559 = match("MESSAGE#149:98:02/4", "nwparser.p0", "%{}proto=%{protocol}");

var part560 = match("MESSAGE#154:427/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{p0}");

var part561 = match("MESSAGE#155:428/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part562 = match("MESSAGE#240:171:03/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} npcs= %{p0}");

var part563 = match("MESSAGE#240:171:03/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} npcs= %{p0}");

var part564 = match("MESSAGE#240:171:03/4", "nwparser.p0", "%{} %{info}");

var part565 = match("MESSAGE#256:180:01/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} note= %{p0}");

var part566 = match("MESSAGE#256:180:01/3_1", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} note= %{p0}");

var part567 = match("MESSAGE#256:180:01/4", "nwparser.p0", "%{}\"%{fld3}\" npcs=%{info}");

var part568 = match("MESSAGE#260:194/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} sport=%{sport->} dport=%{dport->} %{p0}");

var part569 = match("MESSAGE#260:194/1_0", "nwparser.p0", "sent=%{sbytes->} ");

var part570 = match("MESSAGE#260:194/1_1", "nwparser.p0", " rcvd=%{rbytes}");

var part571 = match("MESSAGE#262:196/1_0", "nwparser.p0", "sent=%{sbytes->} cmd=%{p0}");

var part572 = match("MESSAGE#262:196/2", "nwparser.p0", "%{method}");

var part573 = match("MESSAGE#280:261:01/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{p0}");

var part574 = match("MESSAGE#283:273/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld->} src=%{p0}");

var part575 = match("MESSAGE#302:401/0", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr->} %{p0}");

var part576 = match("MESSAGE#302:401/1_1", "nwparser.p0", " %{space}");

var part577 = match("MESSAGE#313:446/3_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=\"%{p0}");

var part578 = match("MESSAGE#313:446/3_1", "nwparser.p0", "%{protocol->} fw_action=\"%{p0}");

var part579 = match("MESSAGE#313:446/4", "nwparser.p0", "%{action}\"");

var part580 = match("MESSAGE#318:522:01/4", "nwparser.p0", "%{}proto=%{protocol->} npcs=%{info}");

var part581 = match("MESSAGE#321:524/5_0", "nwparser.p0", "proto=%{protocol->} ");

var part582 = match("MESSAGE#330:537:01/0", "nwparser.payload", "msg=\"%{action}\" f=%{fld1->} n=%{fld2->} src= %{p0}");

var part583 = match("MESSAGE#332:537:08/0_0", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld51->} appName=\"%{application}\"n=%{p0}");

var part584 = match("MESSAGE#332:537:08/0_1", "nwparser.payload", "msg=\"%{event_description}\" app=%{fld51->} sess=\"%{fld4}\" n=%{p0}");

var part585 = match("MESSAGE#332:537:08/0_2", "nwparser.payload", " msg=\"%{event_description}\" app=%{fld51}n=%{p0}");

var part586 = match("MESSAGE#332:537:08/0_3", "nwparser.payload", "msg=\"%{event_description}\"n=%{p0}");

var part587 = match("MESSAGE#332:537:08/1_0", "nwparser.p0", "%{fld1->} usr=\"%{username}\"src=%{p0}");

var part588 = match("MESSAGE#332:537:08/1_1", "nwparser.p0", "%{fld1}src=%{p0}");

var part589 = match("MESSAGE#332:537:08/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} srcMac=%{p0}");

var part590 = match("MESSAGE#332:537:08/5_0", "nwparser.p0", "dstMac=%{dmacaddr->} proto=%{protocol->} sent=%{p0}");

var part591 = match("MESSAGE#332:537:08/5_1", "nwparser.p0", " proto=%{protocol->} sent=%{p0}");

var part592 = match("MESSAGE#333:537:09/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface}:%{dhost->} dstMac=%{p0}");

var part593 = match("MESSAGE#333:537:09/5_0", "nwparser.p0", "%{sbytes->} rcvd=%{rbytes->} %{p0}");

var part594 = match("MESSAGE#333:537:09/5_1", "nwparser.p0", "%{sbytes->} %{p0}");

var part595 = match("MESSAGE#333:537:09/6_0", "nwparser.p0", " spkt=%{fld3->} cdur=%{fld7->} fw_action=\"%{action}\"");

var part596 = match("MESSAGE#333:537:09/6_1", "nwparser.p0", "spkt=%{fld3->} rpkt=%{fld6->} cdur=%{fld7->} ");

var part597 = match("MESSAGE#333:537:09/6_2", "nwparser.p0", "spkt=%{fld3->} cdur=%{fld7->} ");

var part598 = match("MESSAGE#333:537:09/6_3", "nwparser.p0", " spkt=%{fld3}");

var part599 = match("MESSAGE#336:537:04/0", "nwparser.payload", "msg=\"%{action}\" sess=%{fld1->} n=%{fld2->} src= %{p0}");

var part600 = match("MESSAGE#336:537:04/3_2", "nwparser.p0", "%{daddr->} proto= %{p0}");

var part601 = match("MESSAGE#338:537:10/1_0", "nwparser.p0", "%{fld2->} usr=\"%{username}\" %{p0}");

var part602 = match("MESSAGE#338:537:10/1_1", "nwparser.p0", "%{fld2->} %{p0}");

var part603 = match("MESSAGE#338:537:10/2", "nwparser.p0", "%{}src=%{p0}");

var part604 = match("MESSAGE#338:537:10/3_0", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part605 = match("MESSAGE#338:537:10/3_1", "nwparser.p0", "%{saddr->} dst=%{p0}");

var part606 = match("MESSAGE#338:537:10/6_0", "nwparser.p0", "npcs=%{info->} ");

var part607 = match("MESSAGE#338:537:10/6_1", "nwparser.p0", "cdur=%{fld12->} ");

var part608 = match("MESSAGE#355:598:01/0", "nwparser.payload", "msg=%{msg->} sess=%{fld1->} n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst= %{p0}");

var part609 = match("MESSAGE#361:606/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{p0}");

var part610 = match("MESSAGE#361:606/1_1", "nwparser.p0", "%{daddr}:%{dport->} srcMac=%{p0}");

var part611 = match("MESSAGE#361:606/2", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr}proto=%{p0}");

var part612 = match("MESSAGE#366:712:02/0", "nwparser.payload", "msg=\"%{action}\" %{p0}");

var part613 = match("MESSAGE#366:712:02/1_0", "nwparser.p0", "app=%{fld21->} appName=\"%{application}\" n=%{fld1->} src=%{p0}");

var part614 = match("MESSAGE#366:712:02/2", "nwparser.p0", "%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface->} srcMac=%{p0}");

var part615 = match("MESSAGE#366:712:02/3_0", "nwparser.p0", "%{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var part616 = match("MESSAGE#366:712:02/3_1", "nwparser.p0", "%{smacaddr->} proto=%{p0}");

var part617 = match("MESSAGE#366:712:02/4_0", "nwparser.p0", "%{protocol}/%{fld3->} fw_action=%{p0}");

var part618 = match("MESSAGE#366:712:02/4_1", "nwparser.p0", "%{protocol->} fw_action=%{p0}");

var part619 = match("MESSAGE#366:712:02/5", "nwparser.p0", "%{fld51}");

var part620 = match("MESSAGE#391:908/0", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld2->} src=%{p0}");

var part621 = match("MESSAGE#391:908/4", "nwparser.p0", "%{} %{smacaddr->} dstMac=%{dmacaddr->} proto=%{p0}");

var part622 = match("MESSAGE#439:1199/2", "nwparser.p0", "%{} %{daddr}:%{dport}:%{dinterface->} npcs=%{info}");

var part623 = match("MESSAGE#444:1198/0", "nwparser.payload", "msg=\"%{msg}\" note=\"%{fld3}\" sess=%{fld1->} n=%{fld2->} src=%{p0}");

var part624 = match("MESSAGE#461:1220/3_0", "nwparser.p0", "%{daddr}:%{dport}:%{dinterface->} note=%{p0}");

var part625 = match("MESSAGE#461:1220/3_1", "nwparser.p0", "%{daddr}:%{dport->} note=%{p0}");

var part626 = match("MESSAGE#461:1220/4", "nwparser.p0", "%{}\"%{info}\" fw_action=\"%{action}\"");

var part627 = match("MESSAGE#471:1369/1_0", "nwparser.p0", "%{protocol}/%{fld3}fw_action=\"%{p0}");

var part628 = match("MESSAGE#471:1369/1_1", "nwparser.p0", "%{protocol}fw_action=\"%{p0}");

var select162 = linear_select([
	dup8,
	dup9,
]);

var select163 = linear_select([
	dup15,
	dup16,
]);

var part629 = match("MESSAGE#403:24:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var select164 = linear_select([
	dup25,
	dup26,
]);

var select165 = linear_select([
	dup27,
	dup28,
]);

var select166 = linear_select([
	dup34,
	dup35,
]);

var select167 = linear_select([
	dup25,
	dup39,
]);

var select168 = linear_select([
	dup41,
	dup42,
]);

var select169 = linear_select([
	dup46,
	dup47,
]);

var select170 = linear_select([
	dup49,
	dup50,
]);

var part630 = match("MESSAGE#116:82:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup62,
]));

var part631 = match("MESSAGE#118:83:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr}:%{sport}:%{sinterface->} dst=%{daddr}:%{dport}:%{dinterface}", processor_chain([
	dup5,
]));

var select171 = linear_select([
	dup71,
	dup75,
	dup76,
]);

var select172 = linear_select([
	dup8,
	dup25,
]);

var part632 = match("MESSAGE#168:111:01", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} dstname=%{shost}", processor_chain([
	dup1,
]));

var select173 = linear_select([
	dup88,
	dup89,
]);

var part633 = match("MESSAGE#253:178", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup5,
]));

var select174 = linear_select([
	dup92,
	dup93,
]);

var select175 = linear_select([
	dup96,
	dup97,
]);

var part634 = match("MESSAGE#277:252", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{saddr->} dst=%{daddr}", processor_chain([
	dup87,
]));

var part635 = match("MESSAGE#293:355", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup87,
]));

var part636 = match("MESSAGE#295:356", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup1,
]));

var part637 = match("MESSAGE#298:358", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport}", processor_chain([
	dup1,
]));

var part638 = match("MESSAGE#414:371:01", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var select176 = linear_select([
	dup66,
	dup108,
]);

var select177 = linear_select([
	dup110,
	dup111,
]);

var select178 = linear_select([
	dup115,
	dup45,
]);

var select179 = linear_select([
	dup8,
	dup26,
]);

var select180 = linear_select([
	dup8,
	dup25,
	dup39,
]);

var select181 = linear_select([
	dup71,
	dup15,
	dup16,
]);

var select182 = linear_select([
	dup121,
	dup122,
]);

var select183 = linear_select([
	dup68,
	dup69,
	dup74,
]);

var select184 = linear_select([
	dup127,
	dup128,
]);

var select185 = linear_select([
	dup41,
	dup42,
	dup134,
]);

var select186 = linear_select([
	dup135,
	dup136,
]);

var select187 = linear_select([
	dup138,
	dup139,
]);

var select188 = linear_select([
	dup140,
	dup141,
]);

var select189 = linear_select([
	dup49,
	dup148,
]);

var part639 = match("MESSAGE#365:710", "nwparser.payload", "msg=\"%{action}\" n=%{fld1->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport}", processor_chain([
	dup150,
]));

var select190 = linear_select([
	dup152,
	dup40,
]);

var select191 = linear_select([
	dup154,
	dup155,
]);

var select192 = linear_select([
	dup156,
	dup157,
]);

var part640 = match("MESSAGE#375:766", "nwparser.payload", "msg=\"%{msg}\" n=%{ntype}", processor_chain([
	dup5,
]));

var part641 = match("MESSAGE#377:860:01", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{ntype}", processor_chain([
	dup5,
]));

var part642 = match("MESSAGE#393:914", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} src=%{stransaddr}:%{stransport}:%{sinterface}:%{host->} dst=%{dtransaddr}:%{dtransport}:%{dinterface}:%{shost}", processor_chain([
	dup5,
	dup23,
]));

var part643 = match("MESSAGE#399:994", "nwparser.payload", "msg=\"%{msg}\" n=%{fld1->} usr=%{username->} src=%{stransaddr}:%{stransport->} dst=%{dtransaddr}:%{dtransport->} note=\"%{event_description}\"", processor_chain([
	dup1,
	dup23,
]));

var part644 = match("MESSAGE#406:1110", "nwparser.payload", "msg=\"%{msg}\" %{space->} n=%{fld1}", processor_chain([
	dup1,
	dup23,
]));

var part645 = match("MESSAGE#420:614", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup163,
	dup37,
]));

var part646 = match("MESSAGE#454:654", "nwparser.payload", "msg=\"%{msg}\" sess=%{fld1->} n=%{fld2}", processor_chain([
	dup1,
]));

var select193 = linear_select([
	dup169,
	dup170,
]);

var select194 = linear_select([
	dup172,
	dup173,
]);

var part647 = match("MESSAGE#482:796", "nwparser.payload", "msg=\"%{event_description}\" n=%{fld1->} fw_action=\"%{action}\"", processor_chain([
	dup1,
	dup54,
	dup17,
	dup82,
	dup19,
	dup20,
	dup21,
	dup37,
]));

var all114 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup30,
	]),
});

var all115 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup85,
	]),
});

var all116 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup59,
	]),
});

var all117 = all_match({
	processors: [
		dup95,
		dup192,
	],
	on_success: processor_chain([
		dup59,
	]),
});

var all118 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup100,
	]),
});

var all119 = all_match({
	processors: [
		dup31,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup29,
	]),
});

var all120 = all_match({
	processors: [
		dup102,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup103,
	]),
});

var all121 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup106,
	]),
});

var all122 = all_match({
	processors: [
		dup107,
		dup198,
	],
	on_success: processor_chain([
		dup87,
	]),
});

var all123 = all_match({
	processors: [
		dup104,
		dup177,
		dup10,
		dup178,
	],
	on_success: processor_chain([
		dup109,
	]),
});

var all124 = all_match({
	processors: [
		dup44,
		dup179,
		dup36,
		dup178,
	],
	on_success: processor_chain([
		dup5,
	]),
});

var all125 = all_match({
	processors: [
		dup80,
		dup177,
		dup10,
		dup175,
		dup79,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var all126 = all_match({
	processors: [
		dup151,
		dup213,
		dup153,
		dup214,
		dup215,
		dup158,
	],
	on_success: processor_chain([
		dup150,
		dup51,
		dup52,
		dup53,
		dup54,
		dup37,
		dup55,
		dup17,
		dup18,
		dup19,
		dup20,
		dup21,
	]),
});

var all127 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup191,
		dup94,
	],
	on_success: processor_chain([
		dup1,
	]),
});

var all128 = all_match({
	processors: [
		dup7,
		dup174,
		dup10,
		dup189,
		dup90,
	],
	on_success: processor_chain([
		dup1,
	]),
});
