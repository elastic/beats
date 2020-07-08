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
		field("messageid"),
		constant(": "),
		field("payload"),
	],
});

var dup2 = setc("eventcategory","1401040000");

var dup3 = setf("msg","$MSG");

var dup4 = setf("username","husername");

var dup5 = setc("ec_subject","User");

var dup6 = setc("ec_activity","Logoff");

var dup7 = setc("eventcategory","1801020000");

var dup8 = setc("eventcategory","1605000000");

var dup9 = setc("ec_subject","Service");

var dup10 = setc("eventcategory","1801030000");

var dup11 = setc("eventcategory","1603110000");

var dup12 = setc("ec_subject","NetworkComm");

var dup13 = setc("ec_theme","Communication");

var dup14 = setc("ec_activity","Logon");

var dup15 = setc("ec_theme","Authentication");

var dup16 = setc("eventcategory","1401030000");

var dup17 = setc("ec_outcome","Failure");

var dup18 = setc("eventcategory","1501000000");

var dup19 = setc("eventcategory","1401000000");

var dup20 = setc("eventcategory","1603060000");

var hdr1 = match("HEADER#0:0005", "message", "%{hmonth->} %{hday->} %{htime->} %{hhost->} %{messageid}[%{hfld1}]: [%{husername}] [%{hfld2}] %{payload}", processor_chain([
	setc("header_id","0005"),
]));

var hdr2 = match("HEADER#1:0006", "message", "%{hmonth->} %{hday->} %{htime->} %{hhost->} %{messageid}[%{hfld1}]: [%{husername}] %{payload}", processor_chain([
	setc("header_id","0006"),
]));

var hdr3 = match("HEADER#2:0007", "message", "%{hmonth->} %{hday->} %{htime->} %{hhost->} %{messageid}[%{hfld1}]: %{payload}", processor_chain([
	setc("header_id","0007"),
]));

var hdr4 = match("HEADER#3:0008", "message", "%{hmonth->} %{hday->} %{htime->} %{hhost->} %{messageid}: %{payload}", processor_chain([
	setc("header_id","0008"),
	dup1,
]));

var hdr5 = match("HEADER#4:0001", "message", "%{messageid}[%{hfld1}]: [%{husername}] [%{hfld2}] %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr6 = match("HEADER#5:0002", "message", "%{messageid}[%{hfld1}]: [%{husername}] %{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr7 = match("HEADER#6:0003", "message", "%{messageid}[%{hfld1}]: %{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr8 = match("HEADER#7:0004", "message", "%{messageid}: %{payload}", processor_chain([
	setc("header_id","0004"),
	dup1,
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
]);

var part1 = match("MESSAGE#0:firepass:01", "nwparser.payload", "Entered %{fld2}", processor_chain([
	dup2,
	dup3,
	dup4,
]));

var msg1 = msg("firepass:01", part1);

var part2 = match("MESSAGE#1:firepass:02", "nwparser.payload", "Logged out%{}", processor_chain([
	setc("eventcategory","1401070000"),
	dup5,
	dup6,
	dup3,
	dup4,
]));

var msg2 = msg("firepass:02", part2);

var part3 = match("MESSAGE#2:firepass:03", "nwparser.payload", "Finished using %{fld2}", processor_chain([
	dup2,
	dup3,
	dup4,
]));

var msg3 = msg("firepass:03", part3);

var part4 = match("MESSAGE#3:firepass:04", "nwparser.payload", "Open %{fld2->} to Remote Host:%{dhost}", processor_chain([
	dup7,
	dup3,
	dup4,
]));

var msg4 = msg("firepass:04", part4);

var part5 = match("MESSAGE#4:firepass:05", "nwparser.payload", "param %{fld1->} = %{fld2}", processor_chain([
	setc("eventcategory","1701020000"),
	dup3,
	dup4,
]));

var msg5 = msg("firepass:05", part5);

var part6 = match("MESSAGE#5:firepass:06", "nwparser.payload", "Access menu %{fld2}", processor_chain([
	dup2,
	dup3,
	dup4,
]));

var msg6 = msg("firepass:06", part6);

var part7 = match("MESSAGE#6:firepass:07", "nwparser.payload", "Accessing %{url}", processor_chain([
	dup2,
	dup3,
	dup4,
]));

var msg7 = msg("firepass:07", part7);

var part8 = match("MESSAGE#7:firepass:08", "nwparser.payload", "Network Access: dialing Click to connect to Network Access%{}", processor_chain([
	setc("eventcategory","1801000000"),
	dup3,
	dup4,
]));

var msg8 = msg("firepass:08", part8);

var part9 = match("MESSAGE#8:firepass:09", "nwparser.payload", "FirePass service stopped on %{hostname}", processor_chain([
	dup8,
	dup9,
	setc("ec_activity","Stop"),
	dup3,
	dup4,
]));

var msg9 = msg("firepass:09", part9);

var part10 = match("MESSAGE#9:firepass:10", "nwparser.payload", "FirePass service started on %{hostname}", processor_chain([
	dup8,
	dup9,
	setc("ec_activity","Start"),
	dup3,
	dup4,
]));

var msg10 = msg("firepass:10", part10);

var part11 = match("MESSAGE#10:firepass:11", "nwparser.payload", "shutting down for system reboot%{}", processor_chain([
	setc("eventcategory","1606000000"),
	dup3,
	setc("event_description","shutting down for system reboot"),
]));

var msg11 = msg("firepass:11", part11);

var part12 = match("MESSAGE#11:firepass:12", "nwparser.payload", "%{event_description}", processor_chain([
	dup8,
	dup3,
]));

var msg12 = msg("firepass:12", part12);

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
]);

var part13 = match("MESSAGE#12:GarbageCollection:01", "nwparser.payload", "User: '%{username}' session expired due to inactivity. %{result}.", processor_chain([
	dup10,
	dup3,
]));

var msg13 = msg("GarbageCollection:01", part13);

var part14 = match("MESSAGE#13:GarbageCollection:02", "nwparser.payload", "User: '%{username}' session was terminated.", processor_chain([
	dup10,
	dup3,
]));

var msg14 = msg("GarbageCollection:02", part14);

var part15 = match("MESSAGE#14:GarbageCollection:03", "nwparser.payload", "session '%{sessionid}' is expired due to inactivity. %{result}.", processor_chain([
	dup10,
	dup3,
]));

var msg15 = msg("GarbageCollection:03", part15);

var part16 = match("MESSAGE#15:GarbageCollection:04", "nwparser.payload", "apache server is not running. start it%{}", processor_chain([
	dup8,
	dup3,
]));

var msg16 = msg("GarbageCollection:04", part16);

var part17 = match("MESSAGE#16:GarbageCollection:05", "nwparser.payload", "%{fld2->} already started with pid %{process_id}", processor_chain([
	dup8,
	dup3,
]));

var msg17 = msg("GarbageCollection:05", part17);

var part18 = match("MESSAGE#17:GarbageCollection:06", "nwparser.payload", "no servers defined for Radius Accounting%{}", processor_chain([
	dup11,
	dup3,
]));

var msg18 = msg("GarbageCollection:06", part18);

var part19 = match("MESSAGE#18:GarbageCollection:07", "nwparser.payload", "DHCP Agent is not running... Restarting it.%{}", processor_chain([
	dup11,
	dup3,
]));

var msg19 = msg("GarbageCollection:07", part19);

var part20 = match("MESSAGE#19:GarbageCollection:08", "nwparser.payload", "session '%{sessionid}' is terminated.", processor_chain([
	dup11,
	dup3,
]));

var msg20 = msg("GarbageCollection:08", part20);

var part21 = match("MESSAGE#20:GarbageCollection:09", "nwparser.payload", "can not connect to database %{fld1}", processor_chain([
	dup11,
	dup3,
	setc("event_description","can not connect to database"),
]));

var msg21 = msg("GarbageCollection:09", part21);

var part22 = match("MESSAGE#21:GarbageCollection:10", "nwparser.payload", "timeout happened. restarting %{fld1->} services", processor_chain([
	dup11,
	dup3,
	setc("event_description","timeout happened. restarting services"),
]));

var msg22 = msg("GarbageCollection:10", part22);

var select3 = linear_select([
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
]);

var part23 = match("MESSAGE#22:maintenance:01", "nwparser.payload", "Failed to upload backup file %{filename}. %{info->} Server returned:%{result}", processor_chain([
	dup11,
	dup3,
	dup4,
]));

var msg23 = msg("maintenance:01", part23);

var part24 = match("MESSAGE#23:maintenance:02", "nwparser.payload", "Logged out Sid = %{sessionid}", processor_chain([
	dup8,
	dup12,
	dup6,
	dup13,
	dup3,
	dup4,
]));

var msg24 = msg("maintenance:02", part24);

var part25 = match("MESSAGE#24:maintenance:03", "nwparser.payload", "Network Access: %{info}", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg25 = msg("maintenance:03", part25);

var part26 = match("MESSAGE#25:maintenance:04", "nwparser.payload", "Trying connect to %{fld2->} on %{fqdn}:%{network_port}", processor_chain([
	dup11,
	dup3,
	dup4,
]));

var msg26 = msg("maintenance:04", part26);

var part27 = match("MESSAGE#26:maintenance:05", "nwparser.payload", "%{info}", processor_chain([
	dup11,
	dup3,
	dup4,
]));

var msg27 = msg("maintenance:05", part27);

var select4 = linear_select([
	msg23,
	msg24,
	msg25,
	msg26,
	msg27,
]);

var part28 = match("MESSAGE#27:NetworkAccess:01", "nwparser.payload", "\u003c\u003c%{sessionid}> Open Network Access Connection using remote IP address %{daddr}", processor_chain([
	dup7,
	dup12,
	dup13,
	dup3,
	dup4,
]));

var msg28 = msg("NetworkAccess:01", part28);

var part29 = match("MESSAGE#28:NetworkAccess:02", "nwparser.payload", "\u003c\u003c%{sessionid}> Network Access Connection terminated", processor_chain([
	dup10,
	dup12,
	dup13,
	dup3,
	dup4,
]));

var msg29 = msg("NetworkAccess:02", part29);

var part30 = match("MESSAGE#29:NetworkAccess:03", "nwparser.payload", "\u003c\u003c%{sessionid}> Error - %{info}", processor_chain([
	setc("eventcategory","1801010000"),
	dup12,
	dup13,
	dup3,
	dup4,
]));

var msg30 = msg("NetworkAccess:03", part30);

var select5 = linear_select([
	msg28,
	msg29,
	msg30,
]);

var part31 = match("MESSAGE#30:security:01/0", "nwparser.payload", "User %{username->} logged on from %{p0}");

var part32 = match("MESSAGE#30:security:01/1_0", "nwparser.p0", "%{saddr->} to %{daddr->} Sid = %{sessionid->} ");

var part33 = match("MESSAGE#30:security:01/1_1", "nwparser.p0", "%{saddr->} Sid = %{sessionid->} ");

var part34 = match("MESSAGE#30:security:01/1_2", "nwparser.p0", "%{saddr->} ");

var select6 = linear_select([
	part32,
	part33,
	part34,
]);

var all1 = all_match({
	processors: [
		part31,
		select6,
	],
	on_success: processor_chain([
		setc("eventcategory","1401060000"),
		dup5,
		dup14,
		dup15,
		dup3,
	]),
});

var msg31 = msg("security:01", all1);

var part35 = match("MESSAGE#31:security:02/0", "nwparser.payload", "%{} %{p0}");

var part36 = match("MESSAGE#31:security:02/1_0", "nwparser.p0", "Invalid %{p0}");

var part37 = match("MESSAGE#31:security:02/1_1", "nwparser.p0", "Valid %{p0}");

var select7 = linear_select([
	part36,
	part37,
]);

var part38 = match("MESSAGE#31:security:02/2", "nwparser.p0", "%{}user %{username->} failed to log on from %{saddr}");

var all2 = all_match({
	processors: [
		part35,
		select7,
		part38,
	],
	on_success: processor_chain([
		dup16,
		dup5,
		dup14,
		dup15,
		dup17,
		dup3,
	]),
});

var msg32 = msg("security:02", all2);

var part39 = match("MESSAGE#32:security:03", "nwparser.payload", "Successful password update for user %{user_fullname}, username: %{username}", processor_chain([
	setc("eventcategory","1402040100"),
	setc("ec_activity","Modify"),
	setc("ec_theme","Password"),
	setc("ec_outcome","Success"),
	dup3,
]));

var msg33 = msg("security:03", part39);

var part40 = match("MESSAGE#33:security:04", "nwparser.payload", "Possible intrusion attempt! %{fld1->} consecutive authentication failures happened within %{fld2->} min. Last Source IP Address: %{saddr->} %{info}", processor_chain([
	dup16,
	dup14,
	dup15,
	dup17,
	dup3,
]));

var msg34 = msg("security:04", part40);

var part41 = match("MESSAGE#34:security:05", "nwparser.payload", "User [%{action}] logon from %{saddr}", processor_chain([
	dup18,
	dup5,
	dup14,
	dup15,
	setc("ec_outcome","Error"),
	dup3,
]));

var msg35 = msg("security:05", part41);

var part42 = match("MESSAGE#35:security:06", "nwparser.payload", "Non-administrator account %{username->} attempted to access admin account", processor_chain([
	dup18,
	dup5,
	dup14,
	setc("ec_theme","Policy"),
	dup17,
	dup3,
]));

var msg36 = msg("security:06", part42);

var part43 = match("MESSAGE#36:security:07", "nwparser.payload", "User %{username->} exceeded the allowed number of concurrent logons", processor_chain([
	dup16,
	dup5,
	dup14,
	dup15,
	dup17,
	dup3,
	setc("event_description","user exceeded the allowed number of concurrent logons"),
]));

var msg37 = msg("security:07", part43);

var part44 = match("MESSAGE#37:security:08", "nwparser.payload", "User %{username->} from %{saddr->} presented with challenge", processor_chain([
	dup19,
	dup5,
	dup3,
	setc("event_description","user presented with challenge"),
]));

var msg38 = msg("security:08", part44);

var part45 = match("MESSAGE#38:security:09", "nwparser.payload", "Possible intrusion attempt detected against account %{fld1->} from source IP address %{saddr->} for URI=[%{fld2}]%{info}", processor_chain([
	dup19,
	dup5,
	dup3,
	setc("event_description","Possible intrusion attempt detected"),
]));

var msg39 = msg("security:09", part45);

var select8 = linear_select([
	msg31,
	msg32,
	msg33,
	msg34,
	msg35,
	msg36,
	msg37,
	msg38,
	msg39,
]);

var part46 = match("MESSAGE#39:httpd", "nwparser.payload", "scr_monitor: %{fld1}", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg40 = msg("httpd", part46);

var part47 = match("MESSAGE#40:Miscellaneous:01", "nwparser.payload", "Purge logs: not started. Next purge scheduled time %{fld1->} is not exceeded", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg41 = msg("Miscellaneous:01", part47);

var part48 = match("MESSAGE#41:Miscellaneous:02", "nwparser.payload", "Purge logs: finished. Deleted %{fld1->} logon records", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg42 = msg("Miscellaneous:02", part48);

var part49 = match("MESSAGE#42:Miscellaneous:03", "nwparser.payload", "Purge logs: auto started%{}", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg43 = msg("Miscellaneous:03", part49);

var part50 = match("MESSAGE#43:Miscellaneous:04", "nwparser.payload", "Database error detected, dump: %{info}", processor_chain([
	setc("eventcategory","1603000000"),
	dup3,
	dup4,
]));

var msg44 = msg("Miscellaneous:04", part50);

var part51 = match("MESSAGE#44:Miscellaneous:05", "nwparser.payload", "Recovered database successfully%{}", processor_chain([
	dup8,
	dup3,
	dup4,
]));

var msg45 = msg("Miscellaneous:05", part51);

var select9 = linear_select([
	msg41,
	msg42,
	msg43,
	msg44,
	msg45,
]);

var part52 = match("MESSAGE#45:kernel:07", "nwparser.payload", "kernel: Marketing_resource:%{fld1->} SRC=%{saddr->} DST=%{daddr->} %{info->} PROTO=%{protocol->} SPT=%{sport->} DPT=%{dport->} %{fld3}", processor_chain([
	dup8,
	dup3,
]));

var msg46 = msg("kernel:07", part52);

var part53 = match("MESSAGE#46:kernel:01", "nwparser.payload", "kernel: Marketing_resource: %{info}", processor_chain([
	dup8,
	dup3,
]));

var msg47 = msg("kernel:01", part53);

var part54 = match("MESSAGE#47:kernel:02", "nwparser.payload", "kernel: CSLIP: %{info}", processor_chain([
	dup8,
	dup3,
]));

var msg48 = msg("kernel:02", part54);

var part55 = match("MESSAGE#48:kernel:03", "nwparser.payload", "kernel: PPP %{info}", processor_chain([
	dup8,
	dup3,
]));

var msg49 = msg("kernel:03", part55);

var part56 = match("MESSAGE#49:kernel:04", "nwparser.payload", "kernel: cdrom: open failed.%{}", processor_chain([
	dup8,
	dup3,
]));

var msg50 = msg("kernel:04", part56);

var part57 = match("MESSAGE#50:kernel:06", "nwparser.payload", "kernel: GlobalFilter:%{fld1->} SRC=%{saddr->} DST=%{daddr->} %{info->} PROTO=%{protocol->} SPT=%{sport->} DPT=%{dport->} %{fld3}", processor_chain([
	dup8,
	dup3,
]));

var msg51 = msg("kernel:06", part57);

var part58 = match("MESSAGE#51:kernel:05", "nwparser.payload", "kernel: %{info}", processor_chain([
	dup8,
	dup3,
]));

var msg52 = msg("kernel:05", part58);

var select10 = linear_select([
	msg46,
	msg47,
	msg48,
	msg49,
	msg50,
	msg51,
	msg52,
]);

var part59 = match("MESSAGE#52:sshd", "nwparser.payload", "Accepted publickey for %{username->} from %{saddr->} port %{sport->} %{fld2}", processor_chain([
	setc("eventcategory","1401050100"),
	dup3,
]));

var msg53 = msg("sshd", part59);

var part60 = match("MESSAGE#53:ntpd:01", "nwparser.payload", "frequency initialized %{fld1->} PPM from %{fld2}", processor_chain([
	dup8,
	dup3,
]));

var msg54 = msg("ntpd:01", part60);

var part61 = match("MESSAGE#54:ntpd:02", "nwparser.payload", "kernel time sync status %{resultcode}", processor_chain([
	dup8,
	dup3,
]));

var msg55 = msg("ntpd:02", part61);

var part62 = match("MESSAGE#55:ntpd:03", "nwparser.payload", "Listening on interface %{interface}, %{hostip}#%{network_port}", processor_chain([
	dup8,
	dup3,
]));

var msg56 = msg("ntpd:03", part62);

var part63 = match("MESSAGE#56:ntpd:04", "nwparser.payload", "precision = %{duration_string}", processor_chain([
	dup8,
	dup3,
]));

var msg57 = msg("ntpd:04", part63);

var part64 = match("MESSAGE#57:ntpd:05", "nwparser.payload", "ntpd %{info}", processor_chain([
	dup8,
	dup3,
]));

var msg58 = msg("ntpd:05", part64);

var select11 = linear_select([
	msg54,
	msg55,
	msg56,
	msg57,
	msg58,
]);

var part65 = match("MESSAGE#58:AppTunnel:01", "nwparser.payload", "\u003c\u003c%{sessionid}> %{fld2->} connection to %{dhost}(%{daddr}):%{dport->} terminated", processor_chain([
	dup10,
	dup12,
	dup13,
	dup3,
	dup4,
]));

var msg59 = msg("AppTunnel:01", part65);

var part66 = match("MESSAGE#59:AppTunnel:02", "nwparser.payload", "\u003c\u003c%{sessionid}> %{fld2->} connection to %{dhost}(%{daddr}):%{dport}", processor_chain([
	dup7,
	dup12,
	dup13,
	dup3,
	dup4,
]));

var msg60 = msg("AppTunnel:02", part66);

var part67 = match("MESSAGE#60:AppTunnel:03", "nwparser.payload", "\u003c\u003c%{sessionid}> Error - Connection timed out", processor_chain([
	dup7,
	dup12,
	dup13,
	dup17,
	dup3,
	dup4,
]));

var msg61 = msg("AppTunnel:03", part67);

var part68 = match("MESSAGE#61:AppTunnel:04", "nwparser.payload", "Connection to %{daddr->} port %{dport->} failed", processor_chain([
	dup7,
	dup12,
	dup13,
	dup17,
	dup3,
	dup4,
]));

var msg62 = msg("AppTunnel:04", part68);

var part69 = match("MESSAGE#62:AppTunnel:05", "nwparser.payload", "\u003c\u003c%{sessionid}> Error - Invalid session id", processor_chain([
	dup7,
	dup12,
	dup13,
	dup3,
]));

var msg63 = msg("AppTunnel:05", part69);

var select12 = linear_select([
	msg59,
	msg60,
	msg61,
	msg62,
	msg63,
]);

var part70 = match("MESSAGE#63:run-crons", "nwparser.payload", "%{fld2->} returned %{resultcode}", processor_chain([
	dup8,
	dup3,
]));

var msg64 = msg("run-crons", part70);

var part71 = match("MESSAGE#64:/USR/SBIN/CRON", "nwparser.payload", "(%{username}) CMD (%{action})", processor_chain([
	dup2,
	dup3,
]));

var msg65 = msg("/USR/SBIN/CRON", part71);

var part72 = match("MESSAGE#65:ntpdate", "nwparser.payload", "adjust time server %{daddr->} offset %{duration_string}", processor_chain([
	setc("eventcategory","1605030000"),
	dup3,
]));

var msg66 = msg("ntpdate", part72);

var part73 = match("MESSAGE#66:heartbeat", "nwparser.payload", "info: %{info}", processor_chain([
	setc("eventcategory","1604000000"),
	dup3,
]));

var msg67 = msg("heartbeat", part73);

var part74 = match("MESSAGE#67:mailer", "nwparser.payload", "Failed to send \\'%{subject}\\' to \\'%{to}\\'", processor_chain([
	setc("eventcategory","1207010200"),
	setc("ec_subject","Message"),
	setc("ec_activity","Send"),
	dup13,
	dup17,
	dup3,
]));

var msg68 = msg("mailer", part74);

var part75 = match("MESSAGE#68:EndpointSecurity/0", "nwparser.payload", "id[%{fld1}]: \"%{p0}");

var part76 = match("MESSAGE#68:EndpointSecurity/1_0", "nwparser.p0", "%{fld2->} - Connected%{p0}");

var part77 = match("MESSAGE#68:EndpointSecurity/1_1", "nwparser.p0", "Connected%{p0}");

var select13 = linear_select([
	part76,
	part77,
]);

var part78 = match("MESSAGE#68:EndpointSecurity/2", "nwparser.p0", "%{}from %{saddr->} %{info}\"");

var all3 = all_match({
	processors: [
		part75,
		select13,
		part78,
	],
	on_success: processor_chain([
		dup20,
		dup13,
		dup3,
	]),
});

var msg69 = msg("EndpointSecurity", all3);

var part79 = match("MESSAGE#69:EndpointSecurity:01", "nwparser.payload", "id[%{fld1}]: %{event_description}", processor_chain([
	dup20,
	dup13,
	dup3,
]));

var msg70 = msg("EndpointSecurity:01", part79);

var select14 = linear_select([
	msg69,
	msg70,
]);

var part80 = match("MESSAGE#70:snmp", "nwparser.payload", "SNMP handler started%{}", processor_chain([
	dup20,
	dup3,
	setc("event_description","SNMP handler started"),
	setc("action","started"),
	setc("protocol","SNMP"),
]));

var msg71 = msg("snmp", part80);

var part81 = match("MESSAGE#71:snmp:01", "nwparser.payload", "%{event_description}", processor_chain([
	dup20,
	dup3,
]));

var msg72 = msg("snmp:01", part81);

var select15 = linear_select([
	msg71,
	msg72,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"/USR/SBIN/CRON": msg65,
		"AppTunnel": select12,
		"EndpointSecurity": select14,
		"GarbageCollection": select3,
		"Miscellaneous": select9,
		"NetworkAccess": select5,
		"firepass": select2,
		"heartbeat": msg67,
		"httpd": msg40,
		"kernel": select10,
		"mailer": msg68,
		"maintenance": select4,
		"ntpd": select11,
		"ntpdate": msg66,
		"run-crons": msg64,
		"security": select8,
		"snmp": select15,
		"sshd": msg53,
	}),
]);
