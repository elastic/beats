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

var dup1 = setc("eventcategory","1502000000");

var dup2 = date_time({
	dest: "starttime",
	args: ["fld79","fld80","fld81","fld82"],
	fmts: [
		[dB,dD,dH,dc(":"),dU,dc(":"),dO,dW],
	],
});

var dup3 = setf("msg","$MSG");

var dup4 = date_time({
	dest: "starttime",
	args: ["fld79"],
	fmts: [
		[dW,dc("-"),dM,dc("-"),dD,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup5 = setc("eventcategory","1612000000");

var dup6 = setc("eventcategory","1401060000");

var dup7 = setc("ec_subject","User");

var dup8 = setc("ec_activity","Logon");

var dup9 = setc("ec_theme","Authentication");

var dup10 = setc("ec_outcome","Success");

var dup11 = setc("event_type","Login");

var dup12 = date_time({
	dest: "starttime",
	args: ["fld79","fld22","fld23","fld24"],
	fmts: [
		[dF,dR,dW,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup13 = setc("dclass_counter1_string","Affected Rows");

var dup14 = setc("eventcategory","1401030000");

var dup15 = setc("ec_outcome","Failure");

var dup16 = date_time({
	dest: "starttime",
	args: ["fld79"],
	fmts: [
		[dF,dR,dW,dH,dc(":"),dU,dc(":"),dO],
		[dW,dc("-"),dM,dc("-"),dD,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup17 = setc("eventcategory","1401070000");

var dup18 = setc("ec_activity","Logoff");

var dup19 = setc("event_type","Logout");

var dup20 = setc("event_type","Query");

var hdr1 = match("HEADER#0:0001", "message", "%IMPERVA-%{messageid},%{payload}", processor_chain([
	setc("header_id","0001"),
]));

var select1 = linear_select([
	hdr1,
]);

var part1 = match("MESSAGE#0:IMPERVA_ALERT:02", "nwparser.payload", "alert#=%{operation_id},event#=%{fld7},createTime=%{fld78->} %{fld79->} %{fld80->} %{fld81->} %{timezone->} %{fld82},updateTime=%{fld8},alertSev=%{severity},group=%{group},ruleName=\"%{rulename}\",evntDesc=\"%{event_description}\",category=%{category},disposition=%{disposition},eventType=%{event_type},proto=%{protocol},srcPort=%{sport},srcIP=%{saddr},dstPort=%{dport},dstIP=%{daddr},policyName=\"%{policyname}\",occurrences=%{event_counter},httpHost=%{web_host},webMethod=%{web_method},url=\"%{url}\",webQuery=\"%{web_query}\",soapAction=%{fld83},resultCode=%{resultcode},sessionID=%{sessionid},username=%{username},addUsername=%{fld84},responseTime=%{fld85},responseSize=%{fld86},direction=%{direction},dbUsername=%{fld87},queryGroup=%{fld88},application=\"%{application}\",srcHost=%{shost},osUsername=%{c_username},schemaName=%{owner},dbName=%{db_name},hdrName=%{fld92},action=\"%{action}\",errormsg=\"%{result}\"", processor_chain([
	dup1,
	dup2,
	dup3,
]));

var msg1 = msg("IMPERVA_ALERT:02", part1);

var part2 = match("MESSAGE#1:IMPERVA_ALERT", "nwparser.payload", "alert#=%{operation_id},event#=%{fld7},createTime=%{fld79},updateTime=%{fld80},alertSev=%{severity},group=%{group},ruleName=\"%{rulename}\",evntDesc=\"%{event_description}\",category=%{category},disposition=%{disposition},eventType=%{event_type},proto=%{protocol},srcPort=%{sport},srcIP=%{saddr},dstPort=%{dport},dstIP=%{daddr},policyName=\"%{policyname}\",occurrences=%{event_counter},httpHost=%{web_host},webMethod=%{web_method},url=\"%{url}\",webQuery=\"%{web_query}\",soapAction=%{fld83},resultCode=%{resultcode},sessionID=%{sessionid},username=%{username},addUsername=%{fld84},responseTime=%{fld85},responseSize=%{fld86},direction=%{direction},dbUsername=%{fld87},queryGroup=%{fld88},application=\"%{application}\",srcHost=%{shost},osUsername=%{c_username},schemaName=%{owner},dbName=%{db_name},hdrName=%{fld92},action=\"%{action}\",errormsg=\"%{result}\"", processor_chain([
	dup1,
	dup4,
	dup3,
]));

var msg2 = msg("IMPERVA_ALERT", part2);

var part3 = match("MESSAGE#2:IMPERVA_ALERT:03", "nwparser.payload", "alert#=%{operation_id},event#=%{fld7},createTime=%{fld78->} %{fld79->} %{fld80->} %{fld81->} %{timezone->} %{fld82},updateTime=%{fld8},alertSev=%{severity},group=%{group},ruleName=\"%{rulename}\",evntDesc=\"%{event_description}\",category=%{category},disposition=%{disposition},eventType=%{event_type},proto=%{protocol},srcPort=%{sport},srcIP=%{saddr},dstPort=%{dport},dstIP=%{daddr},policyName=\"%{policyname}\",occurrences=%{event_counter},httpHost=%{web_host},webMethod=%{web_method},url=\"%{url}\",webQuery=\"%{web_query}\",soapAction=%{fld83},resultCode=%{resultcode},sessionID=%{sessionid},username=%{username},addUsername=%{fld84},responseTime=%{fld85},responseSize=%{fld86},direction=%{direction},dbUsername=%{fld87},queryGroup=%{fld88},application=\"%{application}\",srcHost=%{shost},osUsername=%{c_username},schemaName=%{owner},dbName=%{db_name},hdrName=%{fld92},action=%{action}", processor_chain([
	dup1,
	dup2,
	dup3,
]));

var msg3 = msg("IMPERVA_ALERT:03", part3);

var part4 = match("MESSAGE#3:IMPERVA_ALERT:01", "nwparser.payload", "alert#=%{operation_id},event#=%{fld7},createTime=%{fld79},updateTime=%{fld80},alertSev=%{severity},group=%{group},ruleName=\"%{rulename}\",evntDesc=\"%{event_description}\",category=%{category},disposition=%{disposition},eventType=%{event_type},proto=%{protocol},srcPort=%{sport},srcIP=%{saddr},dstPort=%{dport},dstIP=%{daddr},policyName=\"%{policyname}\",occurrences=%{event_counter},httpHost=%{web_host},webMethod=%{web_method},url=\"%{url}\",webQuery=\"%{web_query}\",soapAction=%{fld83},resultCode=%{resultcode},sessionID=%{sessionid},username=%{username},addUsername=%{fld84},responseTime=%{fld85},responseSize=%{fld86},direction=%{direction},dbUsername=%{fld87},queryGroup=%{fld88},application=\"%{application}\",srcHost=%{shost},osUsername=%{c_username},schemaName=%{owner},dbName=%{db_name},hdrName=%{fld92},action=%{action}", processor_chain([
	dup1,
	dup4,
	dup3,
]));

var msg4 = msg("IMPERVA_ALERT:01", part4);

var part5 = match("MESSAGE#4:IMPERVA_EVENT:01", "nwparser.payload", "event#=%{fld77},createTime=%{fld78->} %{fld79->} %{fld80->} %{fld81->} %{timezone->} %{fld82},eventType=%{event_type},eventSev=%{severity},username=%{username},subsystem=%{fld7},message=\"%{event_description}\"", processor_chain([
	dup5,
	dup2,
	dup3,
]));

var msg5 = msg("IMPERVA_EVENT:01", part5);

var part6 = match("MESSAGE#5:IMPERVA_EVENT", "nwparser.payload", "event#=%{fld77},createTime=%{fld79},eventType=%{event_type},eventSev=%{severity},username=%{username},subsystem=%{fld7},message=\"%{event_description}\"", processor_chain([
	dup5,
	dup4,
	dup3,
]));

var msg6 = msg("IMPERVA_EVENT", part6);

var part7 = match("MESSAGE#6:IMPERVA_DATABASE_ACTIVITY:03", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Login,usrGroup=%{group},usrAuth=True,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
	dup3,
	dup13,
]));

var msg7 = msg("IMPERVA_DATABASE_ACTIVITY:03", part7);

var part8 = match("MESSAGE#7:IMPERVA_DATABASE_ACTIVITY:06", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Login,usrGroup=%{group},usrAuth=False,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup14,
	dup7,
	dup8,
	dup9,
	dup15,
	dup11,
	dup12,
	dup3,
	dup13,
]));

var msg8 = msg("IMPERVA_DATABASE_ACTIVITY:06", part8);

var part9 = match("MESSAGE#8:IMPERVA_DATABASE_ACTIVITY:01", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79},srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Login,usrGroup=%{group},usrAuth=True,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup16,
	dup3,
	dup13,
]));

var msg9 = msg("IMPERVA_DATABASE_ACTIVITY:01", part9);

var part10 = match("MESSAGE#9:IMPERVA_DATABASE_ACTIVITY:07", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79},srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Login,usrGroup=%{group},usrAuth=False,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup14,
	dup7,
	dup8,
	dup9,
	dup15,
	dup11,
	dup16,
	dup3,
	dup13,
]));

var msg10 = msg("IMPERVA_DATABASE_ACTIVITY:07", part10);

var part11 = match("MESSAGE#10:IMPERVA_DATABASE_ACTIVITY:04", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Logout,usrGroup=%{group},usrAuth=True,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup17,
	dup7,
	dup18,
	dup9,
	dup10,
	dup19,
	dup12,
	dup3,
	dup13,
]));

var msg11 = msg("IMPERVA_DATABASE_ACTIVITY:04", part11);

var part12 = match("MESSAGE#11:IMPERVA_DATABASE_ACTIVITY:08", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Logout,usrGroup=%{group},usrAuth=False,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup17,
	dup7,
	dup18,
	dup9,
	dup15,
	dup19,
	dup12,
	dup3,
	dup13,
]));

var msg12 = msg("IMPERVA_DATABASE_ACTIVITY:08", part12);

var part13 = match("MESSAGE#12:IMPERVA_DATABASE_ACTIVITY:02", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79},srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Logout,usrGroup=%{group},usrAuth=True,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup17,
	dup7,
	dup18,
	dup9,
	dup10,
	dup19,
	dup4,
	dup3,
	dup13,
]));

var msg13 = msg("IMPERVA_DATABASE_ACTIVITY:02", part13);

var part14 = match("MESSAGE#13:IMPERVA_DATABASE_ACTIVITY:09", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79},srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Logout,usrGroup=%{group},usrAuth=False,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	dup17,
	dup7,
	dup18,
	dup9,
	dup15,
	dup19,
	dup4,
	dup3,
	dup13,
]));

var msg14 = msg("IMPERVA_DATABASE_ACTIVITY:09", part14);

var part15 = match("MESSAGE#14:IMPERVA_DATABASE_ACTIVITY:10", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Query,usrGroup=%{group},usrAuth=True,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86}", processor_chain([
	dup17,
	dup20,
	dup12,
	dup3,
	dup13,
]));

var msg15 = msg("IMPERVA_DATABASE_ACTIVITY:10", part15);

var part16 = match("MESSAGE#15:IMPERVA_DATABASE_ACTIVITY:11", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},,srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=Query,usrGroup=%{group},usrAuth=False,application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86}", processor_chain([
	dup17,
	dup20,
	dup12,
	dup3,
	dup13,
]));

var msg16 = msg("IMPERVA_DATABASE_ACTIVITY:11", part16);

var part17 = match("MESSAGE#16:IMPERVA_DATABASE_ACTIVITY:12", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79->} %{fld22->} %{fld23->} %{fld24},srvGroup=%{group_object},service=%{service},appName=%{fld81},event#=%{fld82},eventType=Login,usrGroup=%{group},usrAuth=%{fld99},application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result}", processor_chain([
	setc("eventcategory","1401050200"),
	dup20,
	dup12,
	dup3,
	dup13,
]));

var msg17 = msg("IMPERVA_DATABASE_ACTIVITY:12", part17);

var part18 = match("MESSAGE#17:IMPERVA_DATABASE_ACTIVITY", "nwparser.payload", "dstIP=%{daddr},dstPort=%{dport},dbUsername=%{username},srcIP=%{saddr},srcPort=%{sport},creatTime=%{fld79},srvGroup=%{group_object},service=%{fld88},appName=%{fld81},event#=%{fld82},eventType=%{event_type},usrGroup=%{group},usrAuth=%{fld83},application=\"%{application}\",osUsername=%{c_username},srcHost=%{shost},dbName=%{db_name},schemaName=%{owner},bindVar=%{fld86},sqlError=%{result},respSize=%{dclass_counter1},respTime=%{duration},affRows=%{fld87},action=\"%{action}\",rawQuery=\"%{info}\"", processor_chain([
	setc("eventcategory","1206000000"),
	dup4,
	dup3,
	dup13,
]));

var msg18 = msg("IMPERVA_DATABASE_ACTIVITY", part18);

var select2 = linear_select([
	msg1,
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
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"Imperva": select2,
	}),
]);
