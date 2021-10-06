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

var dup1 = match("HEADER#1:006/0", "message", "%{month->} %{day->} %{time->} %{hhostname->} %{p0}");

var dup2 = setc("eventcategory","1401070000");

var dup3 = setc("ec_theme","Authentication");

var dup4 = setc("ec_subject","User");

var dup5 = setc("ec_activity","Logoff");

var dup6 = setc("ec_outcome","Success");

var dup7 = setf("msg","$MSG");

var dup8 = date_time({
	dest: "event_time",
	args: ["fld1","fld2"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup9 = setf("event_source","hhostname");

var dup10 = setc("eventcategory","1401060000");

var dup11 = setc("ec_activity","Logon");

var dup12 = setc("eventcategory","1609000000");

var dup13 = setc("eventcategory","1605000000");

var dup14 = setc("eventcategory","1401030000");

var dup15 = setc("ec_outcome","Failure");

var dup16 = setc("eventcategory","1603000000");

var dup17 = match("MESSAGE#19:dhcpd:18/0", "nwparser.payload", "%{} %{p0}");

var dup18 = match("MESSAGE#19:dhcpd:18/1_0", "nwparser.p0", "Added %{p0}");

var dup19 = match("MESSAGE#19:dhcpd:18/1_1", "nwparser.p0", "added %{p0}");

var dup20 = setc("action","DHCPDECLINE");

var dup21 = match("MESSAGE#25:dhcpd:03/1_0", "nwparser.p0", "(%{dhost}) via %{p0}");

var dup22 = match("MESSAGE#25:dhcpd:03/1_1", "nwparser.p0", "via %{p0}");

var dup23 = setc("action","DHCPRELEASE");

var dup24 = setc("action","DHCPDISCOVER");

var dup25 = match("MESSAGE#28:dhcpd:09/0", "nwparser.payload", "DHCPREQUEST for %{saddr->} from %{smacaddr->} %{p0}");

var dup26 = match("MESSAGE#28:dhcpd:09/1_0", "nwparser.p0", "(%{shost}) via %{p0}");

var dup27 = setc("action","DHCPREQUEST");

var dup28 = match("MESSAGE#31:dhcpd:11/2", "nwparser.p0", "%{interface}");

var dup29 = setc("event_description","unknown network segment");

var dup30 = date_time({
	dest: "event_time",
	args: ["month","day","time"],
	fmts: [
		[dB,dF,dZ],
	],
});

var dup31 = match("MESSAGE#38:dhcpd:14/2", "nwparser.p0", "%{interface->} relay %{fld1->} lease-duration %{duration}");

var dup32 = setc("action","DHCPACK");

var dup33 = match("MESSAGE#53:named:16/1_0", "nwparser.p0", "approved%{}");

var dup34 = match("MESSAGE#53:named:16/1_1", "nwparser.p0", "denied%{}");

var dup35 = setf("domain","zone");

var dup36 = match("MESSAGE#56:named:01/0", "nwparser.payload", "client %{saddr}#%{p0}");

var dup37 = match("MESSAGE#57:named:17/1_0", "nwparser.p0", "IN%{p0}");

var dup38 = match("MESSAGE#57:named:17/1_1", "nwparser.p0", "CH%{p0}");

var dup39 = match("MESSAGE#57:named:17/1_2", "nwparser.p0", "HS%{p0}");

var dup40 = match("MESSAGE#57:named:17/3_1", "nwparser.p0", "%{action->} at '%{p0}");

var dup41 = match("MESSAGE#57:named:17/4_0", "nwparser.p0", "%{hostip}.in-addr.arpa' %{p0}");

var dup42 = match("MESSAGE#57:named:17/5_0", "nwparser.p0", "%{dns_querytype->} \"%{fld3}\"");

var dup43 = match("MESSAGE#57:named:17/5_1", "nwparser.p0", "%{dns_querytype->} %{hostip}");

var dup44 = match_copy("MESSAGE#57:named:17/5_2", "nwparser.p0", "dns_querytype");

var dup45 = setc("event_description","updating zone");

var dup46 = match_copy("MESSAGE#60:named:19/2", "nwparser.p0", "event_description");

var dup47 = setf("domain","hostname");

var dup48 = match_copy("MESSAGE#66:named:25/1_1", "nwparser.p0", "result");

var dup49 = setc("eventcategory","1801010000");

var dup50 = setc("ec_activity","Request");

var dup51 = match("MESSAGE#67:named:63/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3}: %{severity}: client %{p0}");

var dup52 = match("MESSAGE#67:named:63/1_0", "nwparser.p0", "%{fld9->} %{p0}");

var dup53 = match("MESSAGE#67:named:63/1_1", "nwparser.p0", "%{p0}");

var dup54 = match("MESSAGE#74:named:10/1_3", "nwparser.p0", "%{sport}:%{p0}");

var dup55 = setc("action","Refused");

var dup56 = setf("dns_querytype","event_description");

var dup57 = setc("eventcategory","1901000000");

var dup58 = match("MESSAGE#83:named:24/0", "nwparser.payload", "client %{saddr}#%{sport->} (%{domain}): %{p0}");

var dup59 = setc("eventcategory","1801000000");

var dup60 = setf("zone","domain");

var dup61 = date_time({
	dest: "event_time",
	args: ["month","day","time"],
	fmts: [
		[dB,dD,dZ],
	],
});

var dup62 = setf("info","hdata");

var dup63 = setc("eventcategory","1301000000");

var dup64 = setc("eventcategory","1303000000");

var dup65 = match_copy("MESSAGE#7:httpd:06", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var dup66 = linear_select([
	dup18,
	dup19,
]);

var dup67 = linear_select([
	dup21,
	dup22,
]);

var dup68 = linear_select([
	dup26,
	dup22,
]);

var dup69 = match_copy("MESSAGE#204:dhcpd:37", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var dup70 = linear_select([
	dup33,
	dup34,
]);

var dup71 = linear_select([
	dup37,
	dup38,
	dup39,
]);

var dup72 = linear_select([
	dup42,
	dup43,
	dup44,
]);

var dup73 = linear_select([
	dup52,
	dup53,
]);

var dup74 = match_copy("MESSAGE#118:validate_dhcpd", "nwparser.payload", "event_description", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var dup75 = match("MESSAGE#134:openvpn-member:01", "nwparser.payload", "%{action->} : %{event_description->} (code=%{resultcode})", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var dup76 = match("MESSAGE#137:openvpn-member:04", "nwparser.payload", "%{severity}: %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var dup77 = match_copy("MESSAGE#225:syslog", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
	dup62,
]));

var hdr1 = match("HEADER#0:001", "message", "%{month->} %{day->} %{time->} %{hhostname->} %{messageid}[%{data}]: %{payload}", processor_chain([
	setc("header_id","001"),
]));

var part1 = match("HEADER#1:006/1_0", "nwparser.p0", "%{hhostip} %{messageid}[%{data}]: %{p0}");

var part2 = match("HEADER#1:006/1_1", "nwparser.p0", "%{hhostip} %{messageid}: %{p0}");

var select1 = linear_select([
	part1,
	part2,
]);

var part3 = match_copy("HEADER#1:006/2", "nwparser.p0", "payload");

var all1 = all_match({
	processors: [
		dup1,
		select1,
		part3,
	],
	on_success: processor_chain([
		setc("header_id","006"),
	]),
});

var hdr2 = match("HEADER#2:005", "message", "%{month->} %{day->} %{time->} %{hhostname->} %{hdata}: %{messageid->} %{payload}", processor_chain([
	setc("header_id","005"),
]));

var part4 = match("HEADER#3:002/1_0", "nwparser.p0", "-%{p0}");

var part5 = match_copy("HEADER#3:002/1_1", "nwparser.p0", "p0");

var select2 = linear_select([
	part4,
	part5,
]);

var part6 = match("HEADER#3:002/2", "nwparser.p0", ":%{messageid->} %{payload}");

var all2 = all_match({
	processors: [
		dup1,
		select2,
		part6,
	],
	on_success: processor_chain([
		setc("header_id","002"),
	]),
});

var hdr3 = match("HEADER#4:0003", "message", "%{messageid}[%{data}]: %{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr4 = match("HEADER#5:0004", "message", "%{messageid}: %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var hdr5 = match("HEADER#6:0005", "message", "%{month->} %{day->} %{time->} %{hhostname->} %{fld1->} |%{messageid->} |%{payload}", processor_chain([
	setc("header_id","0005"),
]));

var select3 = linear_select([
	hdr1,
	all1,
	hdr2,
	all2,
	hdr3,
	hdr4,
	hdr5,
]);

var part7 = match("MESSAGE#0:httpd", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Logout - - ip=%{saddr->} group=%{group->} trigger_event=%{event_description}", processor_chain([
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
]));

var msg1 = msg("httpd", part7);

var part8 = match("MESSAGE#1:httpd:01", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Login_Allowed - - to=%{fld4->} ip=%{saddr->} auth=%{authmethod->} group=%{group->} apparently_via=%{info}", processor_chain([
	dup10,
	dup3,
	dup4,
	dup11,
	dup6,
	dup7,
	dup8,
	dup9,
]));

var msg2 = msg("httpd:01", part8);

var part9 = match("MESSAGE#2:httpd:02", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Called - %{action->} message=%{info}", processor_chain([
	dup12,
	dup7,
	dup8,
	dup9,
]));

var msg3 = msg("httpd:02", part9);

var part10 = match("MESSAGE#3:httpd:03", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Created HostAddress %{hostip}: Set address=\"%{saddr}\",configure_for_dhcp=%{fld10},match_option=\"%{info}\",parent=%{context}", processor_chain([
	dup12,
	dup7,
	dup8,
	dup9,
]));

var msg4 = msg("httpd:03", part10);

var part11 = match("MESSAGE#4:httpd:04", "nwparser.payload", "%{shost}: %{fld1->} authentication for user %{username->} failed", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg5 = msg("httpd:04", part11);

var part12 = match("MESSAGE#5:httpd:05", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Called - %{event_description}", processor_chain([
	dup13,
	dup7,
	dup8,
	dup9,
]));

var msg6 = msg("httpd:05", part12);

var part13 = match("MESSAGE#6:httpd:07", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Login_Denied - - to=%{terminal->} ip=%{saddr->} info=%{info}", processor_chain([
	dup14,
	dup3,
	dup4,
	dup11,
	dup15,
	dup7,
	dup8,
	dup9,
]));

var msg7 = msg("httpd:07", part13);

var msg8 = msg("httpd:06", dup65);

var select4 = linear_select([
	msg1,
	msg2,
	msg3,
	msg4,
	msg5,
	msg6,
	msg7,
	msg8,
]);

var part14 = match("MESSAGE#8:in.tftpd:01", "nwparser.payload", "RRQ from %{saddr->} filename %{filename}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","RRQ from remote host"),
]));

var msg9 = msg("in.tftpd:01", part14);

var part15 = match("MESSAGE#9:in.tftpd:02", "nwparser.payload", "sending NAK (%{resultcode}, %{result}) to %{daddr}", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","sending NAK to remote host"),
]));

var msg10 = msg("in.tftpd:02", part15);

var part16 = match("MESSAGE#10:in.tftpd", "nwparser.payload", "connection refused from %{saddr}", processor_chain([
	setc("eventcategory","1801030000"),
	dup7,
	dup9,
]));

var msg11 = msg("in.tftpd", part16);

var select5 = linear_select([
	msg9,
	msg10,
	msg11,
]);

var part17 = match("MESSAGE#11:dhcpd:12/0", "nwparser.payload", "%{event_type}: received a REQUEST DHCP packet from relay-agent %{interface->} with a circuit-id of \"%{id}\" and remote-id of \"%{smacaddr}\" for %{hostip->} (%{dmacaddr}) lease time is %{p0}");

var part18 = match("MESSAGE#11:dhcpd:12/1_0", "nwparser.p0", "undefined %{p0}");

var part19 = match("MESSAGE#11:dhcpd:12/1_1", "nwparser.p0", "%{duration->} %{p0}");

var select6 = linear_select([
	part18,
	part19,
]);

var part20 = match("MESSAGE#11:dhcpd:12/2", "nwparser.p0", "seconds%{}");

var all3 = all_match({
	processors: [
		part17,
		select6,
		part20,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
		setc("event_description","received a REQUEST DHCP packet from relay-agent"),
	]),
});

var msg12 = msg("dhcpd:12", all3);

var part21 = match("MESSAGE#12:dhcpd:21", "nwparser.payload", "bind update on %{hostip->} from %{hostname}(%{fld1}) rejected: %{result}", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","bind update rejected"),
]));

var msg13 = msg("dhcpd:21", part21);

var part22 = match("MESSAGE#13:dhcpd:10", "nwparser.payload", "Unable to add forward map from %{shost->} %{fld1}to %{daddr}: %{result}", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","Unable to add forward map"),
]));

var msg14 = msg("dhcpd:10", part22);

var part23 = match("MESSAGE#14:dhcpd:13", "nwparser.payload", "Average %{fld1->} dynamic DNS update latency: %{result->} micro seconds", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Average dynamic DNS update latency"),
]));

var msg15 = msg("dhcpd:13", part23);

var part24 = match("MESSAGE#15:dhcpd:15", "nwparser.payload", "Dynamic DNS update timeout count in last %{info->} minutes: %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Dynamic DNS update timeout count"),
]));

var msg16 = msg("dhcpd:15", part24);

var part25 = match("MESSAGE#16:dhcpd:22", "nwparser.payload", "Removed forward map from %{shost->} %{fld1}to %{daddr}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Removed forward map"),
]));

var msg17 = msg("dhcpd:22", part25);

var part26 = match("MESSAGE#17:dhcpd:25", "nwparser.payload", "Removed reverse map on %{hostname}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Removed reverse map"),
]));

var msg18 = msg("dhcpd:25", part26);

var part27 = match("MESSAGE#18:dhcpd:06", "nwparser.payload", "received shutdown -/-/ %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","received shutdown"),
]));

var msg19 = msg("dhcpd:06", part27);

var part28 = match("MESSAGE#19:dhcpd:18/2", "nwparser.p0", "new forward map from %{hostname->} %{space->} %{daddr}");

var all4 = all_match({
	processors: [
		dup17,
		dup66,
		part28,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		setc("event_description","Added new forward map"),
	]),
});

var msg20 = msg("dhcpd:18", all4);

var part29 = match("MESSAGE#20:dhcpd:19/2", "nwparser.p0", "reverse map from %{hostname->} %{space->} %{daddr}");

var all5 = all_match({
	processors: [
		dup17,
		dup66,
		part29,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		setc("event_description","added reverse map"),
	]),
});

var msg21 = msg("dhcpd:19", all5);

var part30 = match("MESSAGE#21:dhcpd", "nwparser.payload", "Abandoning IP address %{hostip}: declined", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","Abandoning IP declined"),
]));

var msg22 = msg("dhcpd", part30);

var part31 = match("MESSAGE#22:dhcpd:30", "nwparser.payload", "Abandoning IP address %{hostip}: pinged before offer", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","Abandoning IP pinged before offer"),
]));

var msg23 = msg("dhcpd:30", part31);

var part32 = match("MESSAGE#23:dhcpd:01", "nwparser.payload", "DHCPDECLINE of %{saddr->} from %{smacaddr->} (%{shost}) via %{interface}: %{info}", processor_chain([
	dup16,
	dup7,
	dup9,
	dup20,
]));

var msg24 = msg("dhcpd:01", part32);

var part33 = match("MESSAGE#24:dhcpd:02", "nwparser.payload", "DHCPDECLINE of %{saddr->} from %{smacaddr->} via %{interface}: %{info}", processor_chain([
	dup16,
	dup7,
	dup9,
	dup20,
]));

var msg25 = msg("dhcpd:02", part33);

var part34 = match("MESSAGE#25:dhcpd:03/0", "nwparser.payload", "DHCPRELEASE of %{saddr->} from %{dmacaddr->} %{p0}");

var part35 = match("MESSAGE#25:dhcpd:03/2", "nwparser.p0", "%{interface->} (%{info})");

var all6 = all_match({
	processors: [
		part34,
		dup67,
		part35,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup23,
	]),
});

var msg26 = msg("dhcpd:03", all6);

var part36 = match("MESSAGE#26:dhcpd:04", "nwparser.payload", "DHCPDISCOVER from %{smacaddr->} via %{interface}: network %{mask}: %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup24,
]));

var msg27 = msg("dhcpd:04", part36);

var part37 = match("MESSAGE#27:dhcpd:07/0", "nwparser.payload", "DHCPREQUEST for %{saddr->} %{p0}");

var part38 = match("MESSAGE#27:dhcpd:07/1_0", "nwparser.p0", "(%{shost}) from %{p0}");

var part39 = match("MESSAGE#27:dhcpd:07/1_1", "nwparser.p0", "from %{p0}");

var select7 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#27:dhcpd:07/2", "nwparser.p0", "%{smacaddr->} (%{hostname}) via %{interface}: ignored (%{result})");

var all7 = all_match({
	processors: [
		part37,
		select7,
		part40,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
		setc("action","DHCPREQUEST ignored"),
	]),
});

var msg28 = msg("dhcpd:07", all7);

var part41 = match("MESSAGE#28:dhcpd:09/2", "nwparser.p0", "%{interface}: wrong network");

var all8 = all_match({
	processors: [
		dup25,
		dup68,
		part41,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup27,
		setc("result","wrong network"),
	]),
});

var msg29 = msg("dhcpd:09", all8);

var part42 = match("MESSAGE#29:dhcpd:26/2", "nwparser.p0", "%{interface}: lease %{hostip->} unavailable");

var all9 = all_match({
	processors: [
		dup25,
		dup68,
		part42,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
		dup27,
		setc("result","lease unavailable"),
	]),
});

var msg30 = msg("dhcpd:26", all9);

var part43 = match("MESSAGE#30:dhcpd:08", "nwparser.payload", "DHCPREQUEST for %{saddr->} (%{shost}) from %{smacaddr->} (%{hostname}) via %{interface}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup27,
]));

var msg31 = msg("dhcpd:08", part43);

var all10 = all_match({
	processors: [
		dup25,
		dup68,
		dup28,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup27,
	]),
});

var msg32 = msg("dhcpd:11", all10);

var part44 = match("MESSAGE#32:dhcpd:31", "nwparser.payload", "DHCPRELEASE from %{smacaddr->} via %{saddr}: unknown network segment", processor_chain([
	dup13,
	dup7,
	dup9,
	dup23,
	dup29,
]));

var msg33 = msg("dhcpd:31", part44);

var part45 = match("MESSAGE#33:dhcpd:32", "nwparser.payload", "BOOTREQUEST from %{smacaddr->} via %{saddr}: %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("action","BOOTREQUEST"),
	dup30,
]));

var msg34 = msg("dhcpd:32", part45);

var part46 = match("MESSAGE#34:dhcpd:33", "nwparser.payload", "Reclaiming abandoned lease %{saddr}.", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Reclaiming abandoned lease"),
]));

var msg35 = msg("dhcpd:33", part46);

var part47 = match("MESSAGE#35:dhcpd:34/0", "nwparser.payload", "balanc%{p0}");

var part48 = match("MESSAGE#35:dhcpd:34/1_0", "nwparser.p0", "ed%{p0}");

var part49 = match("MESSAGE#35:dhcpd:34/1_1", "nwparser.p0", "ing%{p0}");

var select8 = linear_select([
	part48,
	part49,
]);

var part50 = match("MESSAGE#35:dhcpd:34/2", "nwparser.p0", "%{}pool %{fld1->} %{saddr}/%{sport->} total %{fld2->} free %{fld3->} backup %{fld4->} lts %{fld5->} max-%{fld6->} %{p0}");

var part51 = match("MESSAGE#35:dhcpd:34/3_0", "nwparser.p0", "(+/-)%{fld7}(%{info})");

var part52 = match("MESSAGE#35:dhcpd:34/3_1", "nwparser.p0", "(+/-)%{fld7}");

var part53 = match_copy("MESSAGE#35:dhcpd:34/3_2", "nwparser.p0", "fld7");

var select9 = linear_select([
	part51,
	part52,
	part53,
]);

var all11 = all_match({
	processors: [
		part47,
		select8,
		part50,
		select9,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg36 = msg("dhcpd:34", all11);

var part54 = match("MESSAGE#36:dhcpd:35", "nwparser.payload", "Unable to add reverse map from %{shost->} to %{dhost}: REFUSED", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description"," Unable to add reverse map"),
]));

var msg37 = msg("dhcpd:35", part54);

var part55 = match("MESSAGE#37:dhcpd:36", "nwparser.payload", "Forward map from %{shost->} %{fld2}to %{daddr->} FAILED: %{fld1}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description"," Forward map failed"),
]));

var msg38 = msg("dhcpd:36", part55);

var part56 = match("MESSAGE#38:dhcpd:14/0", "nwparser.payload", "DHCPACK on %{saddr->} to %{dmacaddr->} %{p0}");

var all12 = all_match({
	processors: [
		part56,
		dup67,
		dup31,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup32,
	]),
});

var msg39 = msg("dhcpd:14", all12);

var part57 = match("MESSAGE#39:dhcpd:24/0", "nwparser.payload", "DHCPOFFER on %{saddr->} to %{p0}");

var part58 = match("MESSAGE#39:dhcpd:24/1_0", "nwparser.p0", "\"%{dmacaddr}\" (%{dhost}) via %{p0}");

var part59 = match("MESSAGE#39:dhcpd:24/1_1", "nwparser.p0", "%{dmacaddr->} (%{dhost}) via %{p0}");

var part60 = match("MESSAGE#39:dhcpd:24/1_2", "nwparser.p0", "%{dmacaddr->} via %{p0}");

var select10 = linear_select([
	part58,
	part59,
	part60,
]);

var all13 = all_match({
	processors: [
		part57,
		select10,
		dup31,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		setc("action","DHCPOFFER"),
	]),
});

var msg40 = msg("dhcpd:24", all13);

var part61 = match("MESSAGE#40:dhcpd:17", "nwparser.payload", "DHCPNAK on %{saddr->} to %{dmacaddr->} via %{interface}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("action","DHCPNAK"),
]));

var msg41 = msg("dhcpd:17", part61);

var part62 = match("MESSAGE#41:dhcpd:05/0", "nwparser.payload", "DHCPDISCOVER from %{smacaddr->} %{p0}");

var all14 = all_match({
	processors: [
		part62,
		dup68,
		dup28,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup24,
	]),
});

var msg42 = msg("dhcpd:05", all14);

var part63 = match("MESSAGE#42:dhcpd:16", "nwparser.payload", "DHCPACK to %{daddr->} (%{dmacaddr}) via %{interface}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup32,
]));

var msg43 = msg("dhcpd:16", part63);

var part64 = match("MESSAGE#43:dhcpd:20", "nwparser.payload", "DHCPINFORM from %{saddr->} via %{interface}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("action","DHCPINFORM"),
]));

var msg44 = msg("dhcpd:20", part64);

var part65 = match("MESSAGE#44:dhcpd:23", "nwparser.payload", "DHCPEXPIRE on %{saddr->} to %{dmacaddr}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("action","DHCPEXPIRE"),
]));

var msg45 = msg("dhcpd:23", part65);

var part66 = match("MESSAGE#45:dhcpd:28", "nwparser.payload", "uid lease %{hostip->} for client %{smacaddr->} is duplicate on %{mask}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg46 = msg("dhcpd:28", part66);

var part67 = match("MESSAGE#46:dhcpd:29", "nwparser.payload", "Attempt to add forward map \"%{shost}\" (and reverse map \"%{dhost}\") for %{saddr->} abandoned because of non-retryable failure: %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg47 = msg("dhcpd:29", part67);

var part68 = match("MESSAGE#191:dhcpd:39", "nwparser.payload", "NOT FREE/BACKUP lease%{hostip}End Time%{fld1->} Bind-State %{change_old->} Next-Bind-State %{change_new}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg48 = msg("dhcpd:39", part68);

var part69 = match("MESSAGE#192:dhcpd:41", "nwparser.payload", "RELEASE on%{saddr}to%{dmacaddr}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg49 = msg("dhcpd:41", part69);

var part70 = match("MESSAGE#193:dhcpd:42", "nwparser.payload", "r-l-e:%{hostip},%{result},%{fld1},%{macaddr},%{fld3},%{fld4},%{fld5},%{info}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg50 = msg("dhcpd:42", part70);

var part71 = match("MESSAGE#194:dhcpd:43", "nwparser.payload", "failover peer%{fld1}:%{dclass_counter1}leases added to send queue from pool%{fld3->} %{hostip}/%{network_port}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("dclass_counter1_string","count of leases"),
	dup30,
]));

var msg51 = msg("dhcpd:43", part71);

var part72 = match("MESSAGE#195:dhcpd:44", "nwparser.payload", "DHCPDECLINE from%{macaddr}via%{hostip}: unknown network segment", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	dup29,
]));

var msg52 = msg("dhcpd:44", part72);

var part73 = match("MESSAGE#196:dhcpd:45", "nwparser.payload", "Reverse map update for%{hostip}abandoned because of non-retryable failure:%{disposition}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg53 = msg("dhcpd:45", part73);

var part74 = match("MESSAGE#197:dhcpd:46", "nwparser.payload", "Reclaiming REQUESTed abandoned IP address%{saddr}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Reclaiming REQUESTed abandoned IP address"),
]));

var msg54 = msg("dhcpd:46", part74);

var part75 = match("MESSAGE#198:dhcpd:47/0", "nwparser.payload", "%{hostip}: removing client association (%{action})%{p0}");

var part76 = match("MESSAGE#198:dhcpd:47/1_0", "nwparser.p0", "uid=%{fld1}hw=%{p0}");

var part77 = match("MESSAGE#198:dhcpd:47/1_1", "nwparser.p0", "hw=%{p0}");

var select11 = linear_select([
	part76,
	part77,
]);

var part78 = match_copy("MESSAGE#198:dhcpd:47/2", "nwparser.p0", "macaddr");

var all15 = all_match({
	processors: [
		part75,
		select11,
		part78,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg55 = msg("dhcpd:47", all15);

var part79 = match("MESSAGE#199:dhcpd:48", "nwparser.payload", "Lease conflict at %{hostip}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg56 = msg("dhcpd:48", part79);

var part80 = match("MESSAGE#200:dhcpd:49", "nwparser.payload", "ICMP Echo reply while lease %{hostip->} valid.", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("protocol","ICMP"),
]));

var msg57 = msg("dhcpd:49", part80);

var part81 = match("MESSAGE#201:dhcpd:50", "nwparser.payload", "Lease state %{result}. Not abandoning %{hostip}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg58 = msg("dhcpd:50", part81);

var part82 = match("MESSAGE#202:dhcpd:51/0_0", "nwparser.payload", "Addition%{p0}");

var part83 = match("MESSAGE#202:dhcpd:51/0_1", "nwparser.payload", "Removal%{p0}");

var select12 = linear_select([
	part82,
	part83,
]);

var part84 = match("MESSAGE#202:dhcpd:51/1", "nwparser.p0", "%{}of %{p0}");

var part85 = match("MESSAGE#202:dhcpd:51/2_0", "nwparser.p0", "forward%{p0}");

var part86 = match("MESSAGE#202:dhcpd:51/2_1", "nwparser.p0", "reverse%{p0}");

var select13 = linear_select([
	part85,
	part86,
]);

var part87 = match("MESSAGE#202:dhcpd:51/3", "nwparser.p0", "%{}map for %{hostip->} deferred");

var all16 = all_match({
	processors: [
		select12,
		part84,
		select13,
		part87,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
		setc("disposition","deferred"),
	]),
});

var msg59 = msg("dhcpd:51", all16);

var part88 = match("MESSAGE#203:dhcpd:52", "nwparser.payload", "Hostname%{change_old}replaced by%{hostname}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg60 = msg("dhcpd:52", part88);

var msg61 = msg("dhcpd:37", dup69);

var select14 = linear_select([
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
	msg26,
	msg27,
	msg28,
	msg29,
	msg30,
	msg31,
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
	msg46,
	msg47,
	msg48,
	msg49,
	msg50,
	msg51,
	msg52,
	msg53,
	msg54,
	msg55,
	msg56,
	msg57,
	msg58,
	msg59,
	msg60,
	msg61,
]);

var part89 = match("MESSAGE#47:ntpd:05", "nwparser.payload", "system event '%{event_type}' (%{fld1}) status '%{result}' (%{fld2})", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","system event status"),
]));

var msg62 = msg("ntpd:05", part89);

var part90 = match("MESSAGE#48:ntpd:04", "nwparser.payload", "frequency initialized %{result->} from %{filename}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","frequency initialized from file"),
]));

var msg63 = msg("ntpd:04", part90);

var part91 = match("MESSAGE#49:ntpd:03", "nwparser.payload", "ntpd exiting on signal %{dclass_counter1}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","ntpd exiting on signal"),
]));

var msg64 = msg("ntpd:03", part91);

var part92 = match("MESSAGE#50:ntpd", "nwparser.payload", "time slew %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","time slew duraion"),
]));

var msg65 = msg("ntpd", part92);

var part93 = match("MESSAGE#51:ntpd:01", "nwparser.payload", "%{process}: signal %{dclass_counter1->} had flags %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","signal had flags"),
]));

var msg66 = msg("ntpd:01", part93);

var msg67 = msg("ntpd:02", dup65);

var select15 = linear_select([
	msg62,
	msg63,
	msg64,
	msg65,
	msg66,
	msg67,
]);

var part94 = match("MESSAGE#53:named:16/0", "nwparser.payload", "client %{saddr}#%{sport}:%{fld1}: update '%{zone}' %{p0}");

var all17 = all_match({
	processors: [
		part94,
		dup70,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
	]),
});

var msg68 = msg("named:16", all17);

var part95 = match("MESSAGE#54:named/0", "nwparser.payload", "client %{saddr}#%{sport}: update '%{zone}/IN' %{p0}");

var all18 = all_match({
	processors: [
		part95,
		dup70,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
		dup35,
	]),
});

var msg69 = msg("named", all18);

var part96 = match("MESSAGE#55:named:12/0", "nwparser.payload", "client %{saddr}#%{sport}/key dhcp_updater_default: signer \"%{owner}\" %{p0}");

var all19 = all_match({
	processors: [
		part96,
		dup70,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
	]),
});

var msg70 = msg("named:12", all19);

var part97 = match("MESSAGE#56:named:01/1_0", "nwparser.p0", "%{sport}/%{fld1}: signer \"%{p0}");

var part98 = match("MESSAGE#56:named:01/1_1", "nwparser.p0", "%{sport}: signer \"%{p0}");

var select16 = linear_select([
	part97,
	part98,
]);

var part99 = match("MESSAGE#56:named:01/2", "nwparser.p0", "%{owner}\" %{p0}");

var all20 = all_match({
	processors: [
		dup36,
		select16,
		part99,
		dup70,
	],
	on_success: processor_chain([
		dup16,
		dup7,
		dup9,
	]),
});

var msg71 = msg("named:01", all20);

var part100 = match("MESSAGE#57:named:17/0", "nwparser.payload", "client %{saddr}#%{sport}/%{fld1}: updating zone '%{zone}/%{p0}");

var part101 = match("MESSAGE#57:named:17/2", "nwparser.p0", "': %{p0}");

var part102 = match("MESSAGE#57:named:17/3_0", "nwparser.p0", "%{fld2}: %{action->} at '%{p0}");

var select17 = linear_select([
	part102,
	dup40,
]);

var part103 = match("MESSAGE#57:named:17/4_1", "nwparser.p0", "%{hostname}' %{p0}");

var select18 = linear_select([
	dup41,
	part103,
]);

var all21 = all_match({
	processors: [
		part100,
		dup71,
		part101,
		select17,
		select18,
		dup72,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup45,
		dup35,
	]),
});

var msg72 = msg("named:17", all21);

var part104 = match("MESSAGE#58:named:18/0", "nwparser.payload", "client %{saddr}#%{sport}:%{fld1}: updating zone '%{zone}': %{p0}");

var part105 = match("MESSAGE#58:named:18/1_0", "nwparser.p0", "adding %{p0}");

var part106 = match("MESSAGE#58:named:18/1_1", "nwparser.p0", "deleting%{p0}");

var select19 = linear_select([
	part105,
	part106,
]);

var part107 = match("MESSAGE#58:named:18/2", "nwparser.p0", "%{} %{info->} at '%{hostname}'");

var all22 = all_match({
	processors: [
		part104,
		select19,
		part107,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg73 = msg("named:18", all22);

var part108 = match("MESSAGE#59:named:02/0", "nwparser.payload", "client %{saddr}#%{sport}: updating zone '%{zone}/%{p0}");

var part109 = match("MESSAGE#59:named:02/2", "nwparser.p0", "':%{p0}");

var part110 = match("MESSAGE#59:named:02/3_0", "nwparser.p0", "%{fld1}: %{action->} at '%{p0}");

var select20 = linear_select([
	part110,
	dup40,
]);

var part111 = match("MESSAGE#59:named:02/4_1", "nwparser.p0", "%{hostip}' %{p0}");

var select21 = linear_select([
	dup41,
	part111,
]);

var all23 = all_match({
	processors: [
		part108,
		dup71,
		part109,
		select20,
		select21,
		dup72,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup45,
		dup35,
	]),
});

var msg74 = msg("named:02", all23);

var part112 = match("MESSAGE#60:named:19/0", "nwparser.payload", "client %{saddr}#%{sport}/%{fld1}: updating zone '%{zone}': update %{disposition}: %{p0}");

var part113 = match("MESSAGE#60:named:19/1_0", "nwparser.p0", "%{hostname}/%{dns_querytype}: %{p0}");

var part114 = match("MESSAGE#60:named:19/1_1", "nwparser.p0", "%{hostname}: %{p0}");

var select22 = linear_select([
	part113,
	part114,
]);

var all24 = all_match({
	processors: [
		part112,
		select22,
		dup46,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup47,
	]),
});

var msg75 = msg("named:19", all24);

var part115 = match("MESSAGE#61:named:03", "nwparser.payload", "client %{saddr}#%{sport}: updating zone '%{zone}': update %{disposition}: %{hostname}: %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg76 = msg("named:03", part115);

var part116 = match("MESSAGE#62:named:11", "nwparser.payload", "zone %{zone}: notify from %{saddr}#%{sport}: zone is up to date", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","notify zone is up to date"),
]));

var msg77 = msg("named:11", part116);

var part117 = match("MESSAGE#63:named:13", "nwparser.payload", "zone %{zone}: notify from %{saddr}#%{sport}: %{action}, %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg78 = msg("named:13", part117);

var part118 = match("MESSAGE#64:named:14", "nwparser.payload", "zone %{zone}: refresh: retry limit for master %{saddr}#%{sport->} exceeded (%{action})", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg79 = msg("named:14", part118);

var part119 = match("MESSAGE#65:named:15", "nwparser.payload", "zone %{zone}: refresh: failure trying master %{saddr}#%{sport->} (source ::#0): %{action}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg80 = msg("named:15", part119);

var part120 = match("MESSAGE#66:named:25/0", "nwparser.payload", "DNS format error from %{saddr}#%{sport->} resolving %{domain}/%{dns_querytype->} for client %{daddr}#%{dport}: %{p0}");

var part121 = match("MESSAGE#66:named:25/1_0", "nwparser.p0", "%{error}--%{result}");

var select23 = linear_select([
	part121,
	dup48,
]);

var all25 = all_match({
	processors: [
		part120,
		select23,
	],
	on_success: processor_chain([
		dup49,
		dup50,
		dup15,
		dup7,
		dup9,
		setc("event_description","DNS format error"),
		dup30,
	]),
});

var msg81 = msg("named:25", all25);

var part122 = match("MESSAGE#67:named:63/2", "nwparser.p0", "#%{saddr->} %{sport->} (#%{fld5}): query: %{domain->} %{fld4->} (%{daddr})");

var all26 = all_match({
	processors: [
		dup51,
		dup73,
		part122,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg82 = msg("named:63", all26);

var part123 = match("MESSAGE#68:named:72/0", "nwparser.payload", "client %{saddr}#%{sport->} (%{fld1}): %{p0}");

var part124 = match("MESSAGE#68:named:72/1_0", "nwparser.p0", "view%{fld3}: query:%{p0}");

var part125 = match("MESSAGE#68:named:72/1_1", "nwparser.p0", "query:%{p0}");

var select24 = linear_select([
	part124,
	part125,
]);

var part126 = match("MESSAGE#68:named:72/2", "nwparser.p0", "%{} %{domain->} %{fld2->} %{dns_querytype->} %{context->} (%{daddr})");

var all27 = all_match({
	processors: [
		part123,
		select24,
		part126,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg83 = msg("named:72", all27);

var part127 = match("MESSAGE#69:named:28", "nwparser.payload", "%{action->} (%{saddr}#%{sport}) %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg84 = msg("named:28", part127);

var part128 = match("MESSAGE#70:named:71/0", "nwparser.payload", "transfer of '%{zone}' from %{saddr}#%{sport}: failed %{p0}");

var part129 = match("MESSAGE#70:named:71/1_0", "nwparser.p0", "to connect: %{p0}");

var part130 = match("MESSAGE#70:named:71/1_1", "nwparser.p0", "while receiving responses: %{p0}");

var select25 = linear_select([
	part129,
	part130,
]);

var all28 = all_match({
	processors: [
		part128,
		select25,
		dup48,
	],
	on_success: processor_chain([
		dup49,
		dup7,
		dup9,
		dup30,
		setc("event_description","failed"),
	]),
});

var msg85 = msg("named:71", all28);

var part131 = match("MESSAGE#71:named:70/0", "nwparser.payload", "transfer of '%{zone}' from %{saddr}#%{sport}: %{p0}");

var part132 = match("MESSAGE#71:named:70/1_0", "nwparser.p0", "connected using %{daddr}#%{dport}");

var select26 = linear_select([
	part132,
	dup46,
]);

var all29 = all_match({
	processors: [
		part131,
		select26,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg86 = msg("named:70", all29);

var part133 = match("MESSAGE#72:named:40/0", "nwparser.payload", "%{fld1->} client %{saddr}#%{sport}: %{p0}");

var part134 = match("MESSAGE#72:named:40/1_0", "nwparser.p0", "view %{fld2}: %{protocol}: query: %{p0}");

var part135 = match("MESSAGE#72:named:40/1_1", "nwparser.p0", "%{protocol}: query: %{p0}");

var select27 = linear_select([
	part134,
	part135,
]);

var part136 = match("MESSAGE#72:named:40/2", "nwparser.p0", "%{domain->} %{fld3->} %{dns_querytype->} response:%{result->} %{p0}");

var part137 = match("MESSAGE#72:named:40/3_0", "nwparser.p0", "%{context->} %{dns.resptext}");

var part138 = match_copy("MESSAGE#72:named:40/3_1", "nwparser.p0", "context");

var select28 = linear_select([
	part137,
	part138,
]);

var all30 = all_match({
	processors: [
		part133,
		select27,
		part136,
		select28,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg87 = msg("named:40", all30);

var part139 = match("MESSAGE#73:named:05", "nwparser.payload", "zone '%{zone}' %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg88 = msg("named:05", part139);

var part140 = match("MESSAGE#74:named:10/1_0", "nwparser.p0", "%{sport->} %{fld22}/%{fld21}:%{p0}");

var part141 = match("MESSAGE#74:named:10/1_1", "nwparser.p0", "%{sport}/%{fld21}:%{p0}");

var part142 = match("MESSAGE#74:named:10/1_2", "nwparser.p0", "%{sport->} (%{fld21}): %{p0}");

var select29 = linear_select([
	part140,
	part141,
	part142,
	dup54,
]);

var part143 = match("MESSAGE#74:named:10/2", "nwparser.p0", "%{}query: %{domain->} %{info->} (%{daddr})");

var all31 = all_match({
	processors: [
		dup36,
		select29,
		part143,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		setc("event_description","dns query"),
	]),
});

var msg89 = msg("named:10", all31);

var part144 = match("MESSAGE#75:named:29", "nwparser.payload", "client %{saddr}#%{sport}: %{fld1}: received notify for zone '%{zone}'", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","received notify for zone"),
]));

var msg90 = msg("named:29", part144);

var part145 = match("MESSAGE#76:named:08", "nwparser.payload", "client %{saddr}#%{sport}: received notify for zone '%{zone}'", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","client received notify for zone"),
]));

var msg91 = msg("named:08", part145);

var part146 = match("MESSAGE#77:named:09", "nwparser.payload", "client %{saddr}#%{sport}: update forwarding '%{zone}' denied", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","client update forwarding for zone denied"),
]));

var msg92 = msg("named:09", part146);

var part147 = match("MESSAGE#78:named:76/0", "nwparser.payload", "zone %{zone}: ZRQ appl%{p0}");

var part148 = match("MESSAGE#78:named:76/1_0", "nwparser.p0", "ied%{p0}");

var part149 = match("MESSAGE#78:named:76/1_1", "nwparser.p0", "ying%{p0}");

var select30 = linear_select([
	part148,
	part149,
]);

var part150 = match("MESSAGE#78:named:76/2", "nwparser.p0", "%{}transaction %{p0}");

var part151 = match("MESSAGE#78:named:76/3_0", "nwparser.p0", "%{operation_id->} with SOA serial %{serial_number}. Zone version is now %{version}.");

var part152 = match("MESSAGE#78:named:76/3_1", "nwparser.p0", "%{fld1}.");

var select31 = linear_select([
	part151,
	part152,
]);

var all32 = all_match({
	processors: [
		part147,
		select30,
		part150,
		select31,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg93 = msg("named:76", all32);

var part153 = match("MESSAGE#79:named:75", "nwparser.payload", "zone %{zone}: ZRQ applied %{action->} for '%{fld1}': %{fld2->} %{fld3->} %{dns_querytype->} %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg94 = msg("named:75", part153);

var part154 = match("MESSAGE#80:named:06/0", "nwparser.payload", "zone%{p0}");

var part155 = match("MESSAGE#80:named:06/1_0", "nwparser.p0", "_%{fld1}: %{p0}");

var part156 = match("MESSAGE#80:named:06/1_1", "nwparser.p0", " %{zone}: %{p0}");

var select32 = linear_select([
	part155,
	part156,
]);

var all33 = all_match({
	processors: [
		part154,
		select32,
		dup46,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg95 = msg("named:06", all33);

var part157 = match("MESSAGE#81:named:20", "nwparser.payload", "REFUSED unexpected RCODE resolving '%{saddr}.in-addr.arpa/%{event_description}/IN': %{daddr}#%{dport}", processor_chain([
	dup13,
	dup50,
	dup15,
	dup7,
	dup9,
	dup55,
	dup30,
	dup56,
]));

var msg96 = msg("named:20", part157);

var part158 = match("MESSAGE#82:named:49/0", "nwparser.payload", "REFUSED unexpected RCODE resolving '%{zone}/%{dns_querytype}/IN': %{p0}");

var part159 = match("MESSAGE#82:named:49/1_0", "nwparser.p0", "%{daddr}#%{dport}");

var part160 = match_copy("MESSAGE#82:named:49/1_1", "nwparser.p0", "fld1");

var select33 = linear_select([
	part159,
	part160,
]);

var all34 = all_match({
	processors: [
		part158,
		select33,
	],
	on_success: processor_chain([
		dup57,
		dup50,
		dup15,
		dup7,
		dup9,
		dup55,
		dup30,
		dup35,
	]),
});

var msg97 = msg("named:49", all34);

var part161 = match("MESSAGE#83:named:24/1_0", "nwparser.p0", "%{fld2}: zone transfer%{p0}");

var part162 = match("MESSAGE#83:named:24/1_1", "nwparser.p0", "zone transfer%{p0}");

var select34 = linear_select([
	part161,
	part162,
]);

var part163 = match("MESSAGE#83:named:24/2", "nwparser.p0", "%{}'%{zone}' %{action}");

var all35 = all_match({
	processors: [
		dup58,
		select34,
		part163,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg98 = msg("named:24", all35);

var part164 = match("MESSAGE#84:named:26/1_0", "nwparser.p0", "%{fld2}: no more recursive clients %{p0}");

var part165 = match("MESSAGE#84:named:26/1_1", "nwparser.p0", "no more recursive clients%{p0}");

var select35 = linear_select([
	part164,
	part165,
]);

var part166 = match("MESSAGE#84:named:26/2", "nwparser.p0", "%{}(%{fld3}) %{info}");

var all36 = all_match({
	processors: [
		dup58,
		select35,
		part166,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg99 = msg("named:26", all36);

var part167 = match("MESSAGE#85:named:27/1_0", "nwparser.p0", "%{fld2->} : %{fld3->} response from Internet for %{p0}");

var part168 = match("MESSAGE#85:named:27/1_1", "nwparser.p0", "%{fld3->} response from Internet for %{p0}");

var select36 = linear_select([
	part167,
	part168,
]);

var part169 = match_copy("MESSAGE#85:named:27/2", "nwparser.p0", "fld4");

var all37 = all_match({
	processors: [
		dup58,
		select36,
		part169,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg100 = msg("named:27", all37);

var part170 = match("MESSAGE#86:named:38/2", "nwparser.p0", "#%{saddr->} %{p0}");

var part171 = match("MESSAGE#86:named:38/3_0", "nwparser.p0", "%{sport}#%{fld5->} (%{fld6}):%{p0}");

var part172 = match("MESSAGE#86:named:38/3_1", "nwparser.p0", "%{sport->} (%{fld5}):%{p0}");

var select37 = linear_select([
	part171,
	part172,
	dup54,
]);

var part173 = match("MESSAGE#86:named:38/4", "nwparser.p0", "%{}query%{p0}");

var part174 = match("MESSAGE#86:named:38/5_0", "nwparser.p0", " (%{fld7}) '%{domain}/%{fld4}' %{result}");

var part175 = match("MESSAGE#86:named:38/5_1", "nwparser.p0", ": %{domain->} %{fld4->} (%{daddr})");

var select38 = linear_select([
	part174,
	part175,
]);

var all38 = all_match({
	processors: [
		dup51,
		dup73,
		part170,
		select37,
		part173,
		select38,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg101 = msg("named:38", all38);

var part176 = match("MESSAGE#87:named:39", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3}: %{severity}: error (%{result}) resolving '%{saddr}.in-addr.arpa/%{event_description}/IN': %{daddr}#%{dport}", processor_chain([
	dup13,
	dup50,
	dup15,
	dup7,
	dup9,
	dup55,
]));

var msg102 = msg("named:39", part176);

var part177 = match("MESSAGE#88:named:46", "nwparser.payload", "%{event_description}: Authorization denied for the operation (%{fld4}): %{fld5->} (data=\"%{hostip}\", source=\"%{hostname}\")", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg103 = msg("named:46", part177);

var part178 = match("MESSAGE#89:named:64", "nwparser.payload", "client %{saddr}#%{sport}/%{fld1}: updating zone '%{zone}': deleting %{info->} at %{hostname->} %{dns_querytype}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg104 = msg("named:64", part178);

var part179 = match("MESSAGE#90:named:45", "nwparser.payload", "client %{saddr}#%{sport}: updating zone '%{zone}': deleting %{info->} at %{hostname->} %{dns_querytype}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup47,
]));

var msg105 = msg("named:45", part179);

var part180 = match("MESSAGE#91:named:44/0", "nwparser.payload", "client %{saddr}#%{sport}/key dhcp_updater_default: updating zone '%{p0}");

var part181 = match("MESSAGE#91:named:44/1_0", "nwparser.p0", "%{domain}/IN'%{p0}");

var part182 = match("MESSAGE#91:named:44/1_1", "nwparser.p0", "%{domain}'%{p0}");

var select39 = linear_select([
	part181,
	part182,
]);

var part183 = match("MESSAGE#91:named:44/2", "nwparser.p0", ": %{p0}");

var part184 = match("MESSAGE#91:named:44/3_0", "nwparser.p0", "deleting an RR at %{daddr}.in-addr.arpa");

var part185 = match("MESSAGE#91:named:44/3_1", "nwparser.p0", "deleting an RR at %{daddr}.%{fld6}");

var part186 = match_copy("MESSAGE#91:named:44/3_2", "nwparser.p0", "fld5");

var select40 = linear_select([
	part184,
	part185,
	part186,
]);

var all39 = all_match({
	processors: [
		part180,
		select39,
		part183,
		select40,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg106 = msg("named:44", all39);

var part187 = match("MESSAGE#92:named:43", "nwparser.payload", "client %{saddr}#%{sport->} (%{domain}): query (%{fld3}) '%{fld4}/%{dns_querytype}/IN' %{result}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg107 = msg("named:43", part187);

var part188 = match("MESSAGE#93:named:42", "nwparser.payload", "%{result->} resolving '%{saddr}.in-addr.arpa/%{event_description}/IN': %{daddr}#%{dport}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup56,
]));

var msg108 = msg("named:42", part188);

var part189 = match("MESSAGE#94:named:41", "nwparser.payload", "%{fld1}: unable to find root NS '%{domain}'", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg109 = msg("named:41", part189);

var part190 = match("MESSAGE#95:named:47", "nwparser.payload", "client %{saddr}#%{sport}: updating zone '%{zone}': update %{disposition}: %{event_description}", processor_chain([
	setc("eventcategory","1502000000"),
	dup7,
	dup9,
]));

var msg110 = msg("named:47", part190);

var part191 = match("MESSAGE#96:named:48", "nwparser.payload", "client %{saddr}#%{sport->} (%{hostname}): query '%{zone}' %{result}", processor_chain([
	dup57,
	dup7,
	dup9,
	dup30,
]));

var msg111 = msg("named:48", part191);

var part192 = match("MESSAGE#97:named:62", "nwparser.payload", "client %{saddr}#%{sport}/%{fld1->} (%{hostname}): transfer of '%{zone}': %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg112 = msg("named:62", part192);

var part193 = match("MESSAGE#98:named:53", "nwparser.payload", "client %{saddr}#%{sport->} (%{hostname}): transfer of '%{zone}': %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg113 = msg("named:53", part193);

var part194 = match("MESSAGE#99:named:77", "nwparser.payload", "client %{saddr}#%{sport->} (%{domain}): query failed (%{error}) for %{fld1}/IN/%{dns_querytype->} at %{filename}:%{fld2}", processor_chain([
	dup49,
	dup7,
	dup9,
	setc("event_description"," query failed"),
]));

var msg114 = msg("named:77", part194);

var part195 = match("MESSAGE#100:named:52", "nwparser.payload", "client %{saddr}#%{sport->} (%{hostname}): %{info}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup47,
]));

var msg115 = msg("named:52", part195);

var part196 = match("MESSAGE#101:named:50", "nwparser.payload", "%{fld1}: %{domain}/%{dns_querytype->} (%{saddr}) %{info}", processor_chain([
	dup59,
	dup7,
	dup9,
]));

var msg116 = msg("named:50", part196);

var part197 = match("MESSAGE#102:named:51", "nwparser.payload", "%{fld1}: %{fld2}: REFUSED", processor_chain([
	dup57,
	dup7,
	dup9,
	dup50,
	dup15,
	dup55,
]));

var msg117 = msg("named:51", part197);

var part198 = match("MESSAGE#103:named:54", "nwparser.payload", "%{hostip}#%{network_port}: GSS-TSIG authentication failed:%{event_description}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup3,
	dup15,
	dup30,
]));

var msg118 = msg("named:54", part198);

var part199 = match("MESSAGE#104:named:55/0", "nwparser.payload", "success resolving '%{domain}/%{dns_querytype}' (in '%{fld1}'?) %{p0}");

var part200 = match("MESSAGE#104:named:55/1_0", "nwparser.p0", "after disabling EDNS%{}");

var part201 = match_copy("MESSAGE#104:named:55/1_1", "nwparser.p0", "fld2");

var select41 = linear_select([
	part200,
	part201,
]);

var all40 = all_match({
	processors: [
		part199,
		select41,
	],
	on_success: processor_chain([
		dup59,
		dup7,
		dup9,
		dup6,
		dup30,
		dup60,
	]),
});

var msg119 = msg("named:55", all40);

var part202 = match("MESSAGE#105:named:56", "nwparser.payload", "SERVFAIL unexpected RCODE resolving '%{domain}/%{dns_querytype}/IN':%{hostip}#%{network_port}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup50,
	dup15,
	dup30,
	dup60,
]));

var msg120 = msg("named:56", part202);

var part203 = match("MESSAGE#106:named:57", "nwparser.payload", "FORMERR resolving '%{domain}/%{dns_querytype}/IN':%{hostip}#%{network_port}", processor_chain([
	dup59,
	dup7,
	dup9,
	setc("ec_outcome","Error"),
	dup30,
	dup60,
]));

var msg121 = msg("named:57", part203);

var part204 = match("MESSAGE#107:named:04/0", "nwparser.payload", "%{action->} on %{p0}");

var part205 = match("MESSAGE#107:named:04/1_0", "nwparser.p0", "IPv4 interface %{sinterface}, %{saddr}#%{p0}");

var part206 = match("MESSAGE#107:named:04/1_1", "nwparser.p0", "%{saddr}#%{p0}");

var select42 = linear_select([
	part205,
	part206,
]);

var part207 = match_copy("MESSAGE#107:named:04/2", "nwparser.p0", "sport");

var all41 = all_match({
	processors: [
		part204,
		select42,
		part207,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg122 = msg("named:04", all41);

var part208 = match("MESSAGE#108:named:58", "nwparser.payload", "lame server resolving '%{domain}' (in '%{fld2}'?):%{hostip}#%{network_port}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup30,
	dup60,
]));

var msg123 = msg("named:58", part208);

var part209 = match("MESSAGE#109:named:59", "nwparser.payload", "exceeded max queries resolving '%{domain}/%{dns_querytype}'", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	dup60,
]));

var msg124 = msg("named:59", part209);

var part210 = match("MESSAGE#110:named:60", "nwparser.payload", "skipping nameserver '%{hostname}' because it is a CNAME, while resolving '%{domain}/%{dns_querytype}'", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	dup60,
	setc("event_description","skipping nameserver because it is a CNAME"),
]));

var msg125 = msg("named:60", part210);

var part211 = match("MESSAGE#111:named:61", "nwparser.payload", "loading configuration from '%{filename}'", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg126 = msg("named:61", part211);

var part212 = match("MESSAGE#112:named:73", "nwparser.payload", "fetch: %{zone}/%{dns_querytype}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	dup35,
]));

var msg127 = msg("named:73", part212);

var part213 = match("MESSAGE#113:named:74", "nwparser.payload", "decrement_reference: delete from rbt: %{fld1->} %{domain}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg128 = msg("named:74", part213);

var part214 = match("MESSAGE#114:named:07/0_0", "nwparser.payload", "client %{saddr}#%{sport->} (%{hostname}): view %{fld2}: query: %{web_query}");

var part215 = match_copy("MESSAGE#114:named:07/0_1", "nwparser.payload", "event_description");

var select43 = linear_select([
	part214,
	part215,
]);

var all42 = all_match({
	processors: [
		select43,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
		dup30,
	]),
});

var msg129 = msg("named:07", all42);

var select44 = linear_select([
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
	msg83,
	msg84,
	msg85,
	msg86,
	msg87,
	msg88,
	msg89,
	msg90,
	msg91,
	msg92,
	msg93,
	msg94,
	msg95,
	msg96,
	msg97,
	msg98,
	msg99,
	msg100,
	msg101,
	msg102,
	msg103,
	msg104,
	msg105,
	msg106,
	msg107,
	msg108,
	msg109,
	msg110,
	msg111,
	msg112,
	msg113,
	msg114,
	msg115,
	msg116,
	msg117,
	msg118,
	msg119,
	msg120,
	msg121,
	msg122,
	msg123,
	msg124,
	msg125,
	msg126,
	msg127,
	msg128,
	msg129,
]);

var part216 = match("MESSAGE#115:pidof:01", "nwparser.payload", "can't read sid from %{agent}", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","can't read sid"),
]));

var msg130 = msg("pidof:01", part216);

var part217 = match("MESSAGE#116:pidof", "nwparser.payload", "can't get program name from %{agent}", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg131 = msg("pidof", part217);

var select45 = linear_select([
	msg130,
	msg131,
]);

var part218 = match("MESSAGE#117:validate_dhcpd:01", "nwparser.payload", "Configured local-address not available as source address for DNS updates. %{result}", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","Configured local-address not available as source address for DNS updates"),
]));

var msg132 = msg("validate_dhcpd:01", part218);

var msg133 = msg("validate_dhcpd", dup74);

var select46 = linear_select([
	msg132,
	msg133,
]);

var msg134 = msg("syslog-ng", dup65);

var part219 = match("MESSAGE#120:kernel", "nwparser.payload", "Linux version %{version->} (%{from}) (%{fld1}) %{fld2}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg135 = msg("kernel", part219);

var msg136 = msg("kernel:01", dup65);

var select47 = linear_select([
	msg135,
	msg136,
]);

var msg137 = msg("radiusd", dup65);

var part220 = match("MESSAGE#123:rc", "nwparser.payload", "executing %{agent->} start", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg138 = msg("rc", part220);

var msg139 = msg("rc3", dup65);

var part221 = match("MESSAGE#125:rcsysinit", "nwparser.payload", "fsck from %{version}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg140 = msg("rcsysinit", part221);

var msg141 = msg("rcsysinit:01", dup65);

var select48 = linear_select([
	msg140,
	msg141,
]);

var part222 = match("MESSAGE#126:watchdog", "nwparser.payload", "opened %{filename}, with timeout = %{duration->} secs", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg142 = msg("watchdog", part222);

var part223 = match("MESSAGE#127:watchdog:01", "nwparser.payload", "%{action}, pid = %{process_id}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg143 = msg("watchdog:01", part223);

var part224 = match("MESSAGE#128:watchdog:02", "nwparser.payload", "received %{fld1}, cancelling softdog and exiting...", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg144 = msg("watchdog:02", part224);

var part225 = match("MESSAGE#129:watchdog:03", "nwparser.payload", "%{filename->} could not be opened, errno = %{resultcode}", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg145 = msg("watchdog:03", part225);

var msg146 = msg("watchdog:04", dup65);

var select49 = linear_select([
	msg142,
	msg143,
	msg144,
	msg145,
	msg146,
]);

var msg147 = msg("init", dup65);

var part226 = match("MESSAGE#131:logger", "nwparser.payload", "%{action}: %{saddr}/%{mask->} to %{interface}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg148 = msg("logger", part226);

var msg149 = msg("logger:01", dup65);

var select50 = linear_select([
	msg148,
	msg149,
]);

var part227 = match("MESSAGE#133:openvpn-member", "nwparser.payload", "read %{protocol->} [%{info}] %{event_description->} (code=%{resultcode})", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg150 = msg("openvpn-member", part227);

var msg151 = msg("openvpn-member:01", dup75);

var part228 = match("MESSAGE#135:openvpn-member:02", "nwparser.payload", "Options error: %{event_description}", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg152 = msg("openvpn-member:02", part228);

var part229 = match("MESSAGE#136:openvpn-member:03", "nwparser.payload", "OpenVPN %{version->} [%{protocol}] [%{fld2}] %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg153 = msg("openvpn-member:03", part229);

var msg154 = msg("openvpn-member:04", dup76);

var msg155 = msg("openvpn-member:05", dup65);

var select51 = linear_select([
	msg150,
	msg151,
	msg152,
	msg153,
	msg154,
	msg155,
]);

var part230 = match("MESSAGE#139:sshd", "nwparser.payload", "Server listening on %{hostip->} port %{network_port}.", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg156 = msg("sshd", part230);

var part231 = match("MESSAGE#140:sshd:01/0", "nwparser.payload", "Accepted password for %{p0}");

var part232 = match("MESSAGE#140:sshd:01/1_0", "nwparser.p0", "root from %{p0}");

var part233 = match("MESSAGE#140:sshd:01/1_1", "nwparser.p0", "%{username->} from %{p0}");

var select52 = linear_select([
	part232,
	part233,
]);

var part234 = match("MESSAGE#140:sshd:01/2", "nwparser.p0", "%{saddr->} port %{sport->} %{protocol}");

var all43 = all_match({
	processors: [
		part231,
		select52,
		part234,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg157 = msg("sshd:01", all43);

var part235 = match("MESSAGE#141:sshd:02", "nwparser.payload", "Connection closed by %{hostip}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg158 = msg("sshd:02", part235);

var part236 = match("MESSAGE#142:sshd:03", "nwparser.payload", "%{severity}: Bind to port %{network_port->} on %{hostip->} %{result}: %{event_description}", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg159 = msg("sshd:03", part236);

var part237 = match("MESSAGE#143:sshd:04", "nwparser.payload", "%{severity}: Cannot bind any address.", processor_chain([
	setc("eventcategory","1601000000"),
	dup7,
	dup9,
]));

var msg160 = msg("sshd:04", part237);

var part238 = match("MESSAGE#144:sshd:05", "nwparser.payload", "%{action}: logout() %{result}", processor_chain([
	dup2,
	dup3,
	dup5,
	dup15,
	dup7,
	dup9,
	setc("event_description","logout"),
]));

var msg161 = msg("sshd:05", part238);

var part239 = match("MESSAGE#145:sshd:06", "nwparser.payload", "Did not receive identification string from %{saddr}", processor_chain([
	dup16,
	dup7,
	setc("result","no identification string"),
	setc("event_description","Did not receive identification string from peer"),
]));

var msg162 = msg("sshd:06", part239);

var part240 = match("MESSAGE#146:sshd:07", "nwparser.payload", "Sleep 60 seconds for slowing down ssh login%{}", processor_chain([
	dup13,
	dup7,
	setc("result","slowing down ssh login"),
	setc("event_description","Sleep 60 seconds"),
]));

var msg163 = msg("sshd:07", part240);

var part241 = match("MESSAGE#147:sshd:08", "nwparser.payload", "%{authmethod->} authentication succeeded for user %{username}", processor_chain([
	setc("eventcategory","1302010300"),
	dup7,
	setc("event_description","authentication succeeded"),
	dup9,
	dup61,
]));

var msg164 = msg("sshd:08", part241);

var part242 = match("MESSAGE#148:sshd:09", "nwparser.payload", "User group = %{group}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","User group"),
	dup61,
]));

var msg165 = msg("sshd:09", part242);

var part243 = match("MESSAGE#149:sshd:10", "nwparser.payload", "Bad protocol version identification '%{protocol_detail}' from %{saddr}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Bad protocol version identification"),
	dup61,
]));

var msg166 = msg("sshd:10", part243);

var select53 = linear_select([
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
]);

var part244 = match("MESSAGE#150:openvpn-master", "nwparser.payload", "OpenVPN %{version->} [%{protocol}] [%{fld1}] %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg167 = msg("openvpn-master", part244);

var part245 = match("MESSAGE#151:openvpn-master:01", "nwparser.payload", "read %{protocol->} [%{info}]: %{event_description->} (code=%{resultcode})", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg168 = msg("openvpn-master:01", part245);

var msg169 = msg("openvpn-master:02", dup75);

var part246 = match("MESSAGE#153:openvpn-master:03", "nwparser.payload", "%{saddr}:%{sport->} TLS Error: TLS handshake failed", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg170 = msg("openvpn-master:03", part246);

var part247 = match("MESSAGE#154:openvpn-master:04", "nwparser.payload", "%{fld1}/%{saddr}:%{sport->} [%{fld2}] %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg171 = msg("openvpn-master:04", part247);

var part248 = match("MESSAGE#155:openvpn-master:05", "nwparser.payload", "%{saddr}:%{sport->} [%{fld1}] %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg172 = msg("openvpn-master:05", part248);

var msg173 = msg("openvpn-master:06", dup76);

var msg174 = msg("openvpn-master:07", dup65);

var select54 = linear_select([
	msg167,
	msg168,
	msg169,
	msg170,
	msg171,
	msg172,
	msg173,
	msg174,
]);

var part249 = match("MESSAGE#158:INFOBLOX-Grid", "nwparser.payload", "Grid member at %{saddr->} %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg175 = msg("INFOBLOX-Grid", part249);

var part250 = match("MESSAGE#159:INFOBLOX-Grid:02/0_0", "nwparser.payload", "Started%{p0}");

var part251 = match("MESSAGE#159:INFOBLOX-Grid:02/0_1", "nwparser.payload", "Completed%{p0}");

var select55 = linear_select([
	part250,
	part251,
]);

var part252 = match("MESSAGE#159:INFOBLOX-Grid:02/1", "nwparser.p0", "%{}distribution on member with IP address %{saddr}");

var all44 = all_match({
	processors: [
		select55,
		part252,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg176 = msg("INFOBLOX-Grid:02", all44);

var part253 = match("MESSAGE#160:INFOBLOX-Grid:03", "nwparser.payload", "Upgrade Complete%{}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Upgrade Complete"),
]));

var msg177 = msg("INFOBLOX-Grid:03", part253);

var part254 = match("MESSAGE#161:INFOBLOX-Grid:04", "nwparser.payload", "Upgrade to %{fld1}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg178 = msg("INFOBLOX-Grid:04", part254);

var select56 = linear_select([
	msg175,
	msg176,
	msg177,
	msg178,
]);

var part255 = match("MESSAGE#162:db_jnld", "nwparser.payload", "Grid member at %{saddr->} is online.", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg179 = msg("db_jnld", part255);

var part256 = match("MESSAGE#219:db_jnld:01/0", "nwparser.payload", "Resolved conflict for replicated delete of %{p0}");

var part257 = match("MESSAGE#219:db_jnld:01/1_0", "nwparser.p0", "PTR %{p0}");

var part258 = match("MESSAGE#219:db_jnld:01/1_1", "nwparser.p0", "TXT %{p0}");

var part259 = match("MESSAGE#219:db_jnld:01/1_2", "nwparser.p0", "A %{p0}");

var part260 = match("MESSAGE#219:db_jnld:01/1_3", "nwparser.p0", "CNAME %{p0}");

var part261 = match("MESSAGE#219:db_jnld:01/1_4", "nwparser.p0", "SRV %{p0}");

var select57 = linear_select([
	part257,
	part258,
	part259,
	part260,
	part261,
]);

var part262 = match("MESSAGE#219:db_jnld:01/2", "nwparser.p0", "\"%{fld1}\" in zone \"%{zone}\"");

var all45 = all_match({
	processors: [
		part256,
		select57,
		part262,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg180 = msg("db_jnld:01", all45);

var select58 = linear_select([
	msg179,
	msg180,
]);

var part263 = match("MESSAGE#163:sSMTP/0", "nwparser.payload", "Sent mail for %{to->} (%{fld1}) %{p0}");

var part264 = match("MESSAGE#163:sSMTP/1_0", "nwparser.p0", "uid=%{uid->} username=%{username->} outbytes=%{sbytes}");

var part265 = match_copy("MESSAGE#163:sSMTP/1_1", "nwparser.p0", "space");

var select59 = linear_select([
	part264,
	part265,
]);

var all46 = all_match({
	processors: [
		part263,
		select59,
	],
	on_success: processor_chain([
		dup13,
		dup7,
		dup9,
	]),
});

var msg181 = msg("sSMTP", all46);

var part266 = match("MESSAGE#164:sSMTP:02", "nwparser.payload", "Cannot open %{hostname}:%{network_port}", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg182 = msg("sSMTP:02", part266);

var part267 = match("MESSAGE#165:sSMTP:03", "nwparser.payload", "Unable to locate %{hostname}.", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var msg183 = msg("sSMTP:03", part267);

var msg184 = msg("sSMTP:04", dup74);

var select60 = linear_select([
	msg181,
	msg182,
	msg183,
	msg184,
]);

var part268 = match("MESSAGE#167:scheduled_backups", "nwparser.payload", "Backup to %{device->} was successful - Backup file %{filename}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg185 = msg("scheduled_backups", part268);

var part269 = match("MESSAGE#168:scheduled_ftp_backups", "nwparser.payload", "Scheduled backup to the %{device->} was successful - Backup file %{filename}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Scheduled backup to the FTP server was successful"),
]));

var msg186 = msg("scheduled_ftp_backups", part269);

var part270 = match("MESSAGE#169:failed_scheduled_ftp_backups", "nwparser.payload", "Scheduled backup to the %{device->} failed - %{result}.", processor_chain([
	dup16,
	dup7,
	dup9,
	setc("event_description","Scheduled backup to the FTP server failed"),
]));

var msg187 = msg("failed_scheduled_ftp_backups", part270);

var select61 = linear_select([
	msg186,
	msg187,
]);

var part271 = match("MESSAGE#170:scheduled_scp_backups", "nwparser.payload", "Scheduled backup to the %{device->} was successful - Backup file %{filename}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Scheduled backup to the SCP server was successful"),
]));

var msg188 = msg("scheduled_scp_backups", part271);

var part272 = match("MESSAGE#171:python", "nwparser.payload", "%{action->} even though zone '%{zone}' in view '%{fld1}' is locked.", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg189 = msg("python", part272);

var part273 = match("MESSAGE#172:python:01", "nwparser.payload", "%{action->} (algorithm=%{fld1}, key tag=%{fld2}, key size=%{fld3}): '%{hostname}' in view '%{fld4}'.", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg190 = msg("python:01", part273);

var part274 = match("MESSAGE#173:python:02", "nwparser.payload", "%{action}: '%{hostname}' in view '%{fld1}'.", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg191 = msg("python:02", part274);

var part275 = match("MESSAGE#174:python:03", "nwparser.payload", "%{action}: FQDN='%{domain}', ADDRESS='%{saddr}', View='%{fld1}'", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg192 = msg("python:03", part275);

var part276 = match("MESSAGE#175:python:04", "nwparser.payload", "%{action}: FQDN='%{domain}', View='%{fld1}'", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg193 = msg("python:04", part276);

var part277 = match("MESSAGE#176:python:05", "nwparser.payload", "%{fld1}: %{fld2}.%{fld3->} [%{username}]: Populated %{zone->} %{hostname->} DnsView=%{fld4}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg194 = msg("python:05", part277);

var msg195 = msg("python:06", dup65);

var select62 = linear_select([
	msg189,
	msg190,
	msg191,
	msg192,
	msg193,
	msg194,
	msg195,
]);

var part278 = match("MESSAGE#178:monitor", "nwparser.payload", "Type: %{protocol}, State: %{event_state}, Event: %{event_description}.", processor_chain([
	dup12,
	dup7,
	dup9,
]));

var msg196 = msg("monitor", part278);

var part279 = match("MESSAGE#179:snmptrapd", "nwparser.payload", "NET-SNMP version %{version->} %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg197 = msg("snmptrapd", part279);

var part280 = match("MESSAGE#180:snmptrapd:01", "nwparser.payload", "lock in %{fld1->} sleeps more than %{duration->} milliseconds in %{fld2}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg198 = msg("snmptrapd:01", part280);

var msg199 = msg("snmptrapd:02", dup65);

var select63 = linear_select([
	msg197,
	msg198,
	msg199,
]);

var part281 = match("MESSAGE#182:ntpdate", "nwparser.payload", "adjust time server %{saddr->} offset %{duration->} sec", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg200 = msg("ntpdate", part281);

var msg201 = msg("ntpdate:01", dup74);

var select64 = linear_select([
	msg200,
	msg201,
]);

var msg202 = msg("phonehome", dup65);

var part282 = match("MESSAGE#185:purge_scheduled_tasks", "nwparser.payload", "Scheduled tasks have been purged%{}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg203 = msg("purge_scheduled_tasks", part282);

var part283 = match("MESSAGE#186:serial_console:04", "nwparser.payload", "%{fld20->} %{fld21}.%{fld22->} [%{domain}]: Login_Denied - - to=%{terminal->} apparently_via=%{info->} ip=%{saddr->} error=%{result}", processor_chain([
	dup14,
	dup3,
	dup4,
	dup11,
	dup15,
	dup7,
	date_time({
		dest: "event_time",
		args: ["fld20","fld21"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
		],
	}),
	dup9,
	setc("event_description","Login Denied"),
]));

var msg204 = msg("serial_console:04", part283);

var part284 = match("MESSAGE#187:serial_console:03", "nwparser.payload", "No authentication methods succeeded for user %{username}", processor_chain([
	dup14,
	dup3,
	dup4,
	dup11,
	dup15,
	dup7,
	dup9,
	setc("event_description","No authentication methods succeeded for user"),
]));

var msg205 = msg("serial_console:03", part284);

var part285 = match("MESSAGE#188:serial_console", "nwparser.payload", "%{fld1->} %{fld2}.%{fld3->} [%{username}]: Login_Allowed - - to=%{terminal->} apparently_via=%{info->} auth=%{authmethod->} group=%{group}", processor_chain([
	dup10,
	dup3,
	dup4,
	dup11,
	dup6,
	dup7,
	dup8,
	dup9,
]));

var msg206 = msg("serial_console", part285);

var part286 = match("MESSAGE#189:serial_console:01", "nwparser.payload", "RADIUS authentication succeeded for user %{username}", processor_chain([
	setc("eventcategory","1302010100"),
	dup3,
	dup4,
	dup11,
	dup6,
	dup7,
	dup9,
	setc("event_description","RADIUS authentication succeeded for user"),
]));

var msg207 = msg("serial_console:01", part286);

var part287 = match("MESSAGE#190:serial_console:02", "nwparser.payload", "User group = %{group}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","User group identification"),
]));

var msg208 = msg("serial_console:02", part287);

var part288 = match("MESSAGE#205:serial_console:05", "nwparser.payload", "%{fld1->} [%{username}]: rebooted the system", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","system reboot"),
]));

var msg209 = msg("serial_console:05", part288);

var part289 = match("MESSAGE#214:serial_console:06", "nwparser.payload", "Local authentication succeeded for user %{username}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Local authentication succeeded for user"),
]));

var msg210 = msg("serial_console:06", part289);

var select65 = linear_select([
	msg204,
	msg205,
	msg206,
	msg207,
	msg208,
	msg209,
	msg210,
]);

var msg211 = msg("rc6", dup65);

var msg212 = msg("acpid", dup65);

var msg213 = msg("diskcheck", dup65);

var part290 = match("MESSAGE#210:debug_mount", "nwparser.payload", "mount %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg214 = msg("debug_mount", part290);

var msg215 = msg("smart_check_io", dup65);

var msg216 = msg("speedstep_control", dup65);

var part291 = match("MESSAGE#215:controld", "nwparser.payload", "Distribution Started%{}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Distribution Started"),
]));

var msg217 = msg("controld", part291);

var part292 = match("MESSAGE#216:controld:02", "nwparser.payload", "Distribution Complete%{}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","Distribution Complete"),
]));

var msg218 = msg("controld:02", part292);

var select66 = linear_select([
	msg217,
	msg218,
]);

var part293 = match("MESSAGE#217:shutdown", "nwparser.payload", "shutting down for system reboot%{}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","shutting down for system reboot"),
]));

var msg219 = msg("shutdown", part293);

var part294 = match("MESSAGE#218:ntpd_initres", "nwparser.payload", "ntpd exiting on signal 15%{}", processor_chain([
	dup13,
	dup7,
	dup9,
	setc("event_description","ntpd exiting"),
]));

var msg220 = msg("ntpd_initres", part294);

var part295 = match("MESSAGE#220:rsyncd", "nwparser.payload", "name lookup failed for %{saddr}: %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg221 = msg("rsyncd", part295);

var part296 = match("MESSAGE#221:rsyncd:01", "nwparser.payload", "connect from %{shost->} (%{saddr})", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg222 = msg("rsyncd:01", part296);

var part297 = match("MESSAGE#222:rsyncd:02", "nwparser.payload", "rsync on %{filename->} from %{shost->} (%{saddr})", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg223 = msg("rsyncd:02", part297);

var part298 = match("MESSAGE#223:rsyncd:03", "nwparser.payload", "sent %{sbytes->} bytes received %{rbytes->} bytes total size %{fld1}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var msg224 = msg("rsyncd:03", part298);

var part299 = match("MESSAGE#224:rsyncd:04", "nwparser.payload", "building file list%{}", processor_chain([
	dup13,
	dup7,
	setc("event_description","building file list"),
	dup9,
]));

var msg225 = msg("rsyncd:04", part299);

var select67 = linear_select([
	msg221,
	msg222,
	msg223,
	msg224,
	msg225,
]);

var msg226 = msg("syslog", dup77);

var msg227 = msg("restarting", dup77);

var part300 = match_copy("MESSAGE#227:ipmievd", "nwparser.payload", "fld1", processor_chain([
	dup13,
	dup7,
	dup9,
	dup62,
]));

var msg228 = msg("ipmievd", part300);

var part301 = match("MESSAGE#228:netauto_discovery", "nwparser.payload", "%{agent}: Processing path%{fld1}, vnid [%{fld2}]", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
]));

var msg229 = msg("netauto_discovery", part301);

var part302 = match("MESSAGE#229:netauto_discovery:01", "nwparser.payload", "%{agent}:%{fld1}(%{fld2})%{hostip}/%{fld3}:%{product}ver%{version->} device does not answer to lldpRem OID requests, skipping LLDP Neighbors poll", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
	setc("event_description","device does not answer to lldpRem OID requests, skipping LLDP Neighbors poll"),
]));

var msg230 = msg("netauto_discovery:01", part302);

var part303 = match("MESSAGE#230:netauto_discovery:02", "nwparser.payload", "%{agent}:%{space}Static address already set with IP:%{hostip}, Processing%{fld1}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
]));

var msg231 = msg("netauto_discovery:02", part303);

var part304 = match("MESSAGE#231:netauto_discovery:03", "nwparser.payload", "%{agent}:%{fld1}(%{fld2})%{hostip}/%{fld3}: SNMP Credentials: Failed to authenticate", processor_chain([
	dup63,
	dup7,
	dup9,
	dup61,
	dup15,
]));

var msg232 = msg("netauto_discovery:03", part304);

var select68 = linear_select([
	msg229,
	msg230,
	msg231,
	msg232,
]);

var part305 = match("MESSAGE#232:netauto_core:01", "nwparser.payload", "%{agent}: Attempting CLI on device%{device}with interface not in table, ip%{hostip}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
]));

var msg233 = msg("netauto_core:01", part305);

var part306 = match("MESSAGE#233:netauto_core", "nwparser.payload", "netautoctl:%{event_description}", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
]));

var msg234 = msg("netauto_core", part306);

var select69 = linear_select([
	msg233,
	msg234,
]);

var part307 = match_copy("MESSAGE#234:captured_dns_uploader", "nwparser.payload", "event_description", processor_chain([
	dup49,
	dup7,
	dup9,
	dup61,
	dup15,
]));

var msg235 = msg("captured_dns_uploader", part307);

var part308 = match("MESSAGE#235:DIS", "nwparser.payload", "%{fld1}:%{fld2}: Device%{device}/%{hostip}login failure%{result}", processor_chain([
	dup63,
	dup7,
	dup9,
	dup61,
	dup11,
	dup15,
]));

var msg236 = msg("DIS", part308);

var part309 = match("MESSAGE#236:DIS:01", "nwparser.payload", "%{fld2}: %{fld3}: Attempting discover-now for %{hostip->} on %{fld4}, using session ID", processor_chain([
	dup59,
	dup7,
	dup9,
	dup61,
]));

var msg237 = msg("DIS:01", part309);

var select70 = linear_select([
	msg236,
	msg237,
]);

var part310 = match_copy("MESSAGE#237:ErrorMsg", "nwparser.payload", "result", processor_chain([
	dup64,
	dup7,
	dup9,
	dup61,
]));

var msg238 = msg("ErrorMsg", part310);

var part311 = match("MESSAGE#238:tacacs_acct", "nwparser.payload", "%{fld1}: Server %{daddr->} port %{dport}: %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup61,
]));

var msg239 = msg("tacacs_acct", part311);

var part312 = match("MESSAGE#239:tacacs_acct:01", "nwparser.payload", "%{fld1}: Accounting request failed. %{fld2}Server is %{daddr}, port is %{dport}.", processor_chain([
	dup64,
	dup7,
	dup9,
	dup61,
	setc("event_description","Accounting request failed."),
]));

var msg240 = msg("tacacs_acct:01", part312);

var part313 = match("MESSAGE#240:tacacs_acct:02", "nwparser.payload", "%{fld1}: Read %{fld2->} bytes from server %{daddr->} port %{dport}, expecting %{fld3}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup61,
]));

var msg241 = msg("tacacs_acct:02", part313);

var select71 = linear_select([
	msg239,
	msg240,
	msg241,
]);

var part314 = match("MESSAGE#241:dhcpdv6", "nwparser.payload", "Relay-forward message from %{saddr_v6->} port %{sport}, link address %{fld1}, peer address %{daddr_v6}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Relay-forward message"),
]));

var msg242 = msg("dhcpdv6", part314);

var part315 = match("MESSAGE#242:dhcpdv6:01", "nwparser.payload", "Encapsulated Solicit message from %{saddr_v6->} port %{sport->} from client DUID %{fld1}, transaction ID %{id}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Encapsulated Solicit message"),
]));

var msg243 = msg("dhcpdv6:01", part315);

var part316 = match("MESSAGE#243:dhcpdv6:02", "nwparser.payload", "Client %{fld1}, IP '%{fld2}': No addresses available for this interface", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","IP unknown - No addresses available for this interface"),
]));

var msg244 = msg("dhcpdv6:02", part316);

var part317 = match("MESSAGE#244:dhcpdv6:03", "nwparser.payload", "Encapsulating Advertise message to send to %{saddr_v6->} port %{sport}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Encapsulating Advertise message"),
]));

var msg245 = msg("dhcpdv6:03", part317);

var part318 = match("MESSAGE#245:dhcpdv6:04", "nwparser.payload", "Sending Relay-reply message to %{saddr_v6->} port %{sport}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Sending Relay-reply message"),
]));

var msg246 = msg("dhcpdv6:04", part318);

var part319 = match("MESSAGE#246:dhcpdv6:05", "nwparser.payload", "Encapsulated Information-request message from %{saddr_v6->} port %{sport}, transaction ID %{id}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Encapsulated Information-request message"),
]));

var msg247 = msg("dhcpdv6:05", part319);

var part320 = match("MESSAGE#247:dhcpdv6:06", "nwparser.payload", "Encapsulating Reply message to send to %{saddr_v6->} port %{sport}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Encapsulating Reply message"),
]));

var msg248 = msg("dhcpdv6:06", part320);

var part321 = match("MESSAGE#248:dhcpdv6:07", "nwparser.payload", "Encapsulated Renew message from %{saddr_v6->} port %{sport->} from client DUID %{fld1}, transaction ID %{id}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","Encapsulated Renew message"),
]));

var msg249 = msg("dhcpdv6:07", part321);

var part322 = match("MESSAGE#249:dhcpdv6:08", "nwparser.payload", "Reply NA: address %{saddr_v6->} to client with duid %{fld1->} iaid = %{fld2->} static", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var msg250 = msg("dhcpdv6:08", part322);

var msg251 = msg("dhcpdv6:09", dup69);

var select72 = linear_select([
	msg242,
	msg243,
	msg244,
	msg245,
	msg246,
	msg247,
	msg248,
	msg249,
	msg250,
	msg251,
]);

var msg252 = msg("debug", dup69);

var part323 = match("MESSAGE#252:cloud_api", "nwparser.payload", "proxying request to %{hostname}(%{hostip}) %{web_method->} %{url->} %{protocol->} %{info}", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
	setc("event_description","proxying request"),
]));

var msg253 = msg("cloud_api", part323);

var chain1 = processor_chain([
	select3,
	msgid_select({
		"DIS": select70,
		"ErrorMsg": msg238,
		"INFOBLOX-Grid": select56,
		"acpid": msg212,
		"captured_dns_uploader": msg235,
		"cloud_api": msg253,
		"controld": select66,
		"db_jnld": select58,
		"debug": msg252,
		"debug_mount": msg214,
		"dhcpd": select14,
		"dhcpdv6": select72,
		"diskcheck": msg213,
		"httpd": select4,
		"in.tftpd": select5,
		"init": msg147,
		"ipmievd": msg228,
		"kernel": select47,
		"logger": select50,
		"monitor": msg196,
		"named": select44,
		"netauto_core": select69,
		"netauto_discovery": select68,
		"ntpd": select15,
		"ntpd_initres": msg220,
		"ntpdate": select64,
		"openvpn-master": select54,
		"openvpn-member": select51,
		"phonehome": msg202,
		"pidof": select45,
		"purge_scheduled_tasks": msg203,
		"python": select62,
		"radiusd": msg137,
		"rc": msg138,
		"rc3": msg139,
		"rc6": msg211,
		"rcsysinit": select48,
		"restarting": msg227,
		"rsyncd": select67,
		"sSMTP": select60,
		"scheduled_backups": msg185,
		"scheduled_ftp_backups": select61,
		"scheduled_scp_backups": msg188,
		"serial_console": select65,
		"shutdown": msg219,
		"smart_check_io": msg215,
		"snmptrapd": select63,
		"speedstep_control": msg216,
		"sshd": select53,
		"syslog": msg226,
		"syslog-ng": msg134,
		"tacacs_acct": select71,
		"validate_dhcpd": select46,
		"watchdog": select49,
	}),
]);

var hdr6 = match("HEADER#1:006/0", "message", "%{month->} %{day->} %{time->} %{hhostname->} %{p0}");

var part324 = match("MESSAGE#19:dhcpd:18/0", "nwparser.payload", "%{} %{p0}");

var part325 = match("MESSAGE#19:dhcpd:18/1_0", "nwparser.p0", "Added %{p0}");

var part326 = match("MESSAGE#19:dhcpd:18/1_1", "nwparser.p0", "added %{p0}");

var part327 = match("MESSAGE#25:dhcpd:03/1_0", "nwparser.p0", "(%{dhost}) via %{p0}");

var part328 = match("MESSAGE#25:dhcpd:03/1_1", "nwparser.p0", "via %{p0}");

var part329 = match("MESSAGE#28:dhcpd:09/0", "nwparser.payload", "DHCPREQUEST for %{saddr->} from %{smacaddr->} %{p0}");

var part330 = match("MESSAGE#28:dhcpd:09/1_0", "nwparser.p0", "(%{shost}) via %{p0}");

var part331 = match("MESSAGE#31:dhcpd:11/2", "nwparser.p0", "%{interface}");

var part332 = match("MESSAGE#38:dhcpd:14/2", "nwparser.p0", "%{interface->} relay %{fld1->} lease-duration %{duration}");

var part333 = match("MESSAGE#53:named:16/1_0", "nwparser.p0", "approved%{}");

var part334 = match("MESSAGE#53:named:16/1_1", "nwparser.p0", "denied%{}");

var part335 = match("MESSAGE#56:named:01/0", "nwparser.payload", "client %{saddr}#%{p0}");

var part336 = match("MESSAGE#57:named:17/1_0", "nwparser.p0", "IN%{p0}");

var part337 = match("MESSAGE#57:named:17/1_1", "nwparser.p0", "CH%{p0}");

var part338 = match("MESSAGE#57:named:17/1_2", "nwparser.p0", "HS%{p0}");

var part339 = match("MESSAGE#57:named:17/3_1", "nwparser.p0", "%{action->} at '%{p0}");

var part340 = match("MESSAGE#57:named:17/4_0", "nwparser.p0", "%{hostip}.in-addr.arpa' %{p0}");

var part341 = match("MESSAGE#57:named:17/5_0", "nwparser.p0", "%{dns_querytype->} \"%{fld3}\"");

var part342 = match("MESSAGE#57:named:17/5_1", "nwparser.p0", "%{dns_querytype->} %{hostip}");

var part343 = match_copy("MESSAGE#57:named:17/5_2", "nwparser.p0", "dns_querytype");

var part344 = match_copy("MESSAGE#60:named:19/2", "nwparser.p0", "event_description");

var part345 = match_copy("MESSAGE#66:named:25/1_1", "nwparser.p0", "result");

var part346 = match("MESSAGE#67:named:63/0", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3}: %{severity}: client %{p0}");

var part347 = match("MESSAGE#67:named:63/1_0", "nwparser.p0", "%{fld9->} %{p0}");

var part348 = match("MESSAGE#67:named:63/1_1", "nwparser.p0", "%{p0}");

var part349 = match("MESSAGE#74:named:10/1_3", "nwparser.p0", "%{sport}:%{p0}");

var part350 = match("MESSAGE#83:named:24/0", "nwparser.payload", "client %{saddr}#%{sport->} (%{domain}): %{p0}");

var part351 = match_copy("MESSAGE#7:httpd:06", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var select73 = linear_select([
	dup18,
	dup19,
]);

var select74 = linear_select([
	dup21,
	dup22,
]);

var select75 = linear_select([
	dup26,
	dup22,
]);

var part352 = match_copy("MESSAGE#204:dhcpd:37", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
	dup30,
]));

var select76 = linear_select([
	dup33,
	dup34,
]);

var select77 = linear_select([
	dup37,
	dup38,
	dup39,
]);

var select78 = linear_select([
	dup42,
	dup43,
	dup44,
]);

var select79 = linear_select([
	dup52,
	dup53,
]);

var part353 = match_copy("MESSAGE#118:validate_dhcpd", "nwparser.payload", "event_description", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var part354 = match("MESSAGE#134:openvpn-member:01", "nwparser.payload", "%{action->} : %{event_description->} (code=%{resultcode})", processor_chain([
	dup16,
	dup7,
	dup9,
]));

var part355 = match("MESSAGE#137:openvpn-member:04", "nwparser.payload", "%{severity}: %{event_description}", processor_chain([
	dup13,
	dup7,
	dup9,
]));

var part356 = match_copy("MESSAGE#225:syslog", "nwparser.payload", "event_description", processor_chain([
	dup13,
	dup7,
	dup9,
	dup62,
]));
