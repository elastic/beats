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

var map_getActionName = {
	keyvaluepairs: {
		"0": constant("Allowed Message"),
		"1": constant("Aborted Message"),
		"10": constant("Attachments Stubbed"),
		"2": constant("Blocked Message"),
		"3": constant("Quarantined Message"),
		"4": constant("Tagged Message"),
		"5": dup21,
		"6": constant("Per-User Quarantined Message"),
		"7": constant("Whitelisted Message"),
		"8": constant("Encrypted Message"),
		"9": constant("Redirected Message"),
	},
};

var map_getActionNameForSend = {
	keyvaluepairs: {
		"1": constant("Delivered Message"),
		"2": constant("Rejected Message"),
		"3": dup21,
		"4": constant("Expired Message"),
	},
};

var map_getReasonName = {
	keyvaluepairs: {
		"1": constant("Virus"),
		"11": constant("Client IP"),
		"12": constant("Recipient Address"),
		"13": constant("No Valid Recipients"),
		"14": constant("Domain Not Found"),
		"15": constant("Sender Address"),
		"17": constant("Need Fully Qualified Recipient"),
		"18": constant("Need Fully Qualified Sender"),
		"19": constant("Unsupported Command"),
		"2": constant("Banned Attachment"),
		"20": constant("MAIL FROM Syntax Error"),
		"21": constant("Bad Address Syntax"),
		"22": constant("RCPT TO Syntax Error"),
		"23": constant("Send EHLO/HELO First"),
		"24": constant("Need MAIL Command"),
		"25": constant("Nested MAIL Command"),
		"27": constant("EHLO/HELO Syntax Error"),
		"3": constant("RBL Match"),
		"30": constant("Mail Protocol Violation"),
		"31": constant("Score"),
		"34": constant("Header Filter Match"),
		"35": constant("Sender Block/Accept"),
		"36": constant("Recipient Block/Accept"),
		"37": constant("Body Filter Match"),
		"38": constant("Message Size Bypass"),
		"39": constant("Intention Analysis Match"),
		"4": constant("Rate Control"),
		"40": constant("SPF/Caller-ID"),
		"41": constant("Client Host Rejected"),
		"44": constant("Authentication Not Enabled"),
		"45": constant("Allowed Message Size Exceeded"),
		"46": constant("Too Many Recipients"),
		"47": constant("Need RCPT Command"),
		"48": constant("DATA Syntax Error"),
		"49": constant("Internal Error"),
		"5": constant("Too Many Message In Session"),
		"50": constant("Too Many Hops"),
		"51": constant("Mail Protocol Error"),
		"55": constant("Invalid Parameter Syntax"),
		"56": constant("STARTTLS Syntax Error"),
		"57": constant("TLS Already Active"),
		"58": constant("Too Many Errors"),
		"59": constant("Need STARTTLS First"),
		"6": constant("Timeout Exceeded"),
		"60": constant("Spam Fingerprint Found"),
		"61": constant("Barracuda Reputation Whitelist"),
		"62": constant("Barracuda Reputation Blocklist"),
		"63": constant("DomainKeys"),
		"64": constant("Recipient Verification Unavailable"),
		"65": constant("Realtime Intent"),
		"66": constant("Client Reverse DNS"),
		"67": constant("Email Registry"),
		"68": constant("Invalid Bounce"),
		"69": constant("Intent - Adult"),
		"7": constant("No Such Domain"),
		"70": constant("Intent - Political"),
		"71": constant("Multi-Level Intent"),
		"72": constant("Attachment Limit Exceeded"),
		"73": constant("System Busy"),
		"74": constant("BRTS Intent"),
		"75": constant("Per Domain Recipient"),
		"76": constant("Per Domain Sender"),
		"77": constant("Per Domain Client IP"),
		"78": constant("Sender Spoofed"),
		"79": constant("Attachment Content"),
		"8": constant("No Such User"),
		"80": constant("Outlook Add-in"),
		"82": constant("Barracuda IP/Domain Reputation"),
		"83": constant("Authentication Failure"),
		"85": constant("Attachment Size"),
		"86": constant("Virus detected by Extended Malware Protection"),
		"87": constant("Extended Malware Protection engine is busy"),
		"88": constant("A message was categorized for Email Category"),
		"89": constant("Macro Blocked"),
		"9": constant("Subject Filter Match"),
	},
};

var map_getEventLegacyCategoryName = {
	keyvaluepairs: {
		"1207000000": constant("Content.Email"),
		"1207010000": constant("Content.Email.Delivery"),
		"1207010100": constant("Content.Email.Delivery.Success"),
		"1207010201": constant("Content.Email.Delivery.Error.Nondelivery Receipt"),
		"1207040100": constant("Content.Email.Spam.Suspect"),
		"1207040200": constant("Content.Email.Spam.Blocked"),
	},
	"default": constant("Other.Default"),
};

var map_getEventLegacyCategory = {
	keyvaluepairs: {
		"Aborted Message": dup23,
		"Allowed Message": dup22,
		"Attachments Stubbed": dup26,
		"Blocked Message": dup23,
		"Deferred Message": constant("1207010201"),
		"Delivered Message": dup22,
		"Encrypted Message": dup25,
		"Expired Message": dup25,
		"Per-User Quarantined Message": dup25,
		"Quarantined Message": dup24,
		"Redirected Message": dup26,
		"Rejected Message": dup23,
		"Tagged Message": dup24,
		"Whitelisted Message": dup22,
	},
	"default": constant("1901000000"),
};

var dup1 = match("MESSAGE#0:000001/1_0", "nwparser.p0", "%{fld3->} %{resultcode->} %{info}");

var dup2 = match_copy("MESSAGE#0:000001/1_1", "nwparser.p0", "info");

var dup3 = setc("eventcategory","1207010201");

var dup4 = setf("msg","$MSG");

var dup5 = setc("direction","inbound");

var dup6 = date_time({
	dest: "starttime",
	args: ["fld1"],
	fmts: [
		[dX],
	],
});

var dup7 = date_time({
	dest: "endtime",
	args: ["fld2"],
	fmts: [
		[dX],
	],
});

var dup8 = field("fld3");

var dup9 = field("resultcode");

var dup10 = field("disposition");

var dup11 = field("event_cat");

var dup12 = setc("action"," RECV");

var dup13 = setc("eventcategory","1207010000");

var dup14 = setc("direction","outbound");

var dup15 = match("MESSAGE#13:000003/1_0", "nwparser.p0", "SZ:%{fld9->} SUBJ:%{subject}");

var dup16 = setc("eventcategory","1207040000");

var dup17 = setc("eventcategory","1701020000");

var dup18 = setc("ec_subject","User");

var dup19 = setc("ec_activity","Logon");

var dup20 = setc("ec_theme","Authentication");

var dup21 = constant("Deferred Message");

var dup22 = constant("1207010100");

var dup23 = constant("1207040200");

var dup24 = constant("1207040100");

var dup25 = constant("1207010000");

var dup26 = constant("1207000000");

var dup27 = linear_select([
	dup1,
	dup2,
]);

var dup28 = lookup({
	dest: "nwparser.disposition",
	map: map_getActionName,
	key: dup8,
});

var dup29 = lookup({
	dest: "nwparser.result",
	map: map_getReasonName,
	key: dup9,
});

var dup30 = lookup({
	dest: "nwparser.event_cat",
	map: map_getEventLegacyCategory,
	key: dup10,
});

var dup31 = lookup({
	dest: "nwparser.event_cat_name",
	map: map_getEventLegacyCategoryName,
	key: dup11,
});

var dup32 = lookup({
	dest: "nwparser.disposition",
	map: map_getActionNameForSend,
	key: dup8,
});

var dup33 = linear_select([
	dup15,
	dup2,
]);

var hdr1 = match("HEADER#0:0001", "message", "%{messageid}[%{hfld14}]: %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("["),
			field("hfld14"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%{hfld1}/%{messageid}[%{hfld14}]: %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant("/"),
			field("messageid"),
			constant("["),
			field("hfld14"),
			constant("]: "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0003", "message", "%{messageid}: %{p0}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
]);

var part1 = match("MESSAGE#0:000001/0", "nwparser.payload", "inbound/pass1[%{fld14}]: %{username}[%{saddr}] %{id->} %{fld1->} %{fld2->} RECV %{from->} %{to->} %{p0}");

var all1 = all_match({
	processors: [
		part1,
		dup27,
	],
	on_success: processor_chain([
		dup3,
		dup4,
		dup5,
		dup6,
		dup7,
		dup28,
		dup29,
		dup30,
		dup31,
		dup12,
	]),
});

var msg1 = msg("000001", all1);

var part2 = match("MESSAGE#1:inbound/pass1/0", "nwparser.payload", "inbound/pass1: %{web_domain}[%{saddr}] %{id->} %{fld1->} %{fld2->} SCAN %{fld4->} %{from->} %{to->} %{fld5->} %{fld3->} %{resultcode->} %{p0}");

var part3 = match("MESSAGE#1:inbound/pass1/1_0", "nwparser.p0", "%{fld6->} SZ:%{fld8->} SUBJ:%{subject}");

var part4 = match("MESSAGE#1:inbound/pass1/1_1", "nwparser.p0", "%{domain->} %{info}");

var select2 = linear_select([
	part3,
	part4,
]);

var all2 = all_match({
	processors: [
		part2,
		select2,
	],
	on_success: processor_chain([
		dup3,
		dup4,
		dup5,
		dup6,
		dup7,
		dup28,
		dup29,
		dup30,
		dup31,
		setc("action"," SCAN"),
	]),
});

var msg2 = msg("inbound/pass1", all2);

var part5 = match("MESSAGE#2:inbound/pass1:01/0", "nwparser.payload", "inbound/pass1:%{web_domain}[%{saddr}] %{id->} %{fld1->} %{fld2->} RECV %{from->} %{to->} %{p0}");

var all3 = all_match({
	processors: [
		part5,
		dup27,
	],
	on_success: processor_chain([
		dup3,
		dup4,
		dup5,
		dup6,
		dup7,
		dup28,
		dup29,
		dup30,
		dup31,
		dup12,
	]),
});

var msg3 = msg("inbound/pass1:01", all3);

var select3 = linear_select([
	msg1,
	msg2,
	msg3,
]);

var part6 = match("MESSAGE#3:000002/0", "nwparser.payload", "outbound/smtp[%{fld14}]: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{p0}");

var part7 = match("MESSAGE#3:000002/1_0", "nwparser.p0", "%{fld4->} %{fld3->} %{sessionid->} %{resultcode->} %{info}");

var select4 = linear_select([
	part7,
	dup2,
]);

var all4 = all_match({
	processors: [
		part6,
		select4,
	],
	on_success: processor_chain([
		dup13,
		dup4,
		dup14,
		dup32,
		dup30,
		dup31,
	]),
});

var msg4 = msg("000002", all4);

var part8 = match("MESSAGE#4:outbound/smtp/0", "nwparser.payload", "outbound/smtp: %{saddr->} %{fld5->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{resultcode->} %{p0}");

var part9 = match("MESSAGE#4:outbound/smtp/1_0", "nwparser.p0", "%{fld8->} \u003c\u003c%{from}> %{p0}");

var part10 = match("MESSAGE#4:outbound/smtp/1_1", "nwparser.p0", "\u003c\u003c%{from}>%{p0}");

var select5 = linear_select([
	part9,
	part10,
]);

var part11 = match("MESSAGE#4:outbound/smtp/2", "nwparser.p0", "%{} %{p0}");

var part12 = match("MESSAGE#4:outbound/smtp/3_0", "nwparser.p0", "[InternalId=%{id}, Hostname=%{hostname}] %{event_description->} #to#%{ddomain}");

var part13 = match("MESSAGE#4:outbound/smtp/3_1", "nwparser.p0", "[InternalId=%{id}] %{event_description->} #to#%{daddr}");

var part14 = match("MESSAGE#4:outbound/smtp/3_2", "nwparser.p0", "[InternalId=%{id}, Hostname=%{hostname}] %{info}");

var part15 = match("MESSAGE#4:outbound/smtp/3_3", "nwparser.p0", "%{event_description->} #to#%{ddomain}[%{daddr}]:%{dport}");

var part16 = match("MESSAGE#4:outbound/smtp/3_4", "nwparser.p0", "%{event_description->} #to#%{ddomain}");

var select6 = linear_select([
	part12,
	part13,
	part14,
	part15,
	part16,
]);

var all5 = all_match({
	processors: [
		part8,
		select5,
		part11,
		select6,
	],
	on_success: processor_chain([
		dup13,
		dup4,
		dup14,
		dup32,
		dup30,
		dup31,
	]),
});

var msg5 = msg("outbound/smtp", all5);

var part17 = match("MESSAGE#5:000009/0", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{resultcode->} %{p0}");

var part18 = match("MESSAGE#5:000009/1_0", "nwparser.p0", "%{fld8->} ok%{p0}");

var part19 = match("MESSAGE#5:000009/1_1", "nwparser.p0", "ok%{p0}");

var select7 = linear_select([
	part18,
	part19,
]);

var part20 = match("MESSAGE#5:000009/2", "nwparser.p0", "%{fld9->} Message %{fld10->} accepted #to#%{ddomain}[%{daddr}]:%{dport}");

var all6 = all_match({
	processors: [
		part17,
		select7,
		part20,
	],
	on_success: processor_chain([
		dup13,
		dup4,
		dup14,
		dup32,
		dup30,
		dup31,
	]),
});

var msg6 = msg("000009", all6);

var part21 = match("MESSAGE#6:outbound/smtp:01", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{resultcode->} Message accepted for delivery #to#%{ddomain}[%{daddr}]:%{dport}", processor_chain([
	dup13,
	dup4,
	dup14,
	setc("result"," Message accepted for delivery"),
	dup32,
	dup30,
	dup31,
]));

var msg7 = msg("outbound/smtp:01", part21);

var part22 = match("MESSAGE#7:outbound/smtp:02", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} conversation with %{fld5}[%{fld6}] timed out while sending %{fld7->} #to#%{ddomain}[%{daddr}]:%{dport}", processor_chain([
	dup13,
	dup4,
	dup14,
	dup32,
	dup30,
	dup31,
]));

var msg8 = msg("outbound/smtp:02", part22);

var part23 = match("MESSAGE#8:000010/0", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{fld7->} %{p0}");

var part24 = match("MESSAGE#8:000010/1_0", "nwparser.p0", "Ok %{fld9->} %{fld10->} - gsmtp #to#%{p0}");

var part25 = match("MESSAGE#8:000010/1_1", "nwparser.p0", "Ok: queued as %{fld9->} #to#%{p0}");

var part26 = match("MESSAGE#8:000010/1_2", "nwparser.p0", "ok %{fld9->} #to#%{p0}");

var part27 = match("MESSAGE#8:000010/1_3", "nwparser.p0", "Ok (%{fld9}) #to#%{p0}");

var part28 = match("MESSAGE#8:000010/1_4", "nwparser.p0", "OK %{fld9->} #to#%{p0}");

var part29 = match("MESSAGE#8:000010/1_5", "nwparser.p0", "%{fld9->} #to#%{p0}");

var select8 = linear_select([
	part24,
	part25,
	part26,
	part27,
	part28,
	part29,
]);

var part30 = match_copy("MESSAGE#8:000010/2", "nwparser.p0", "daddr");

var all7 = all_match({
	processors: [
		part23,
		select8,
		part30,
	],
	on_success: processor_chain([
		dup13,
		dup4,
		dup14,
		dup32,
		dup30,
		dup31,
	]),
});

var msg9 = msg("000010", all7);

var part31 = match("MESSAGE#9:000011", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} connect to %{ddomain}[%{daddr}]: %{event_description}", processor_chain([
	dup13,
	dup4,
	dup14,
	dup32,
	dup30,
	dup31,
]));

var msg10 = msg("000011", part31);

var part32 = match("MESSAGE#10:000012", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{fld7->} [%{ddomain}]: %{event_description}", processor_chain([
	dup13,
	dup4,
	dup14,
	dup32,
	dup30,
	dup31,
]));

var msg11 = msg("000012", part32);

var part33 = match("MESSAGE#11:000013", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{resultcode->} %{fld7->} \u003c\u003c%{from}>: %{event_description}", processor_chain([
	dup13,
	dup4,
	dup14,
	dup32,
	dup30,
	dup31,
]));

var msg12 = msg("000013", part33);

var part34 = match("MESSAGE#12:000014", "nwparser.payload", "outbound/smtp: %{saddr->} %{id->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{resultcode->} %{fld8->} %{event_description}", processor_chain([
	dup13,
	dup4,
	dup14,
	dup32,
	dup30,
	dup31,
]));

var msg13 = msg("000014", part34);

var select9 = linear_select([
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
]);

var part35 = match("MESSAGE#13:000003/0", "nwparser.payload", "scan[%{fld14}]: %{username}[%{saddr}] %{id->} %{fld1->} %{fld2->} %{action->} %{fld8->} %{from->} %{to->} %{fld4->} %{fld3->} %{resultcode->} %{fld7->} %{p0}");

var all8 = all_match({
	processors: [
		part35,
		dup33,
	],
	on_success: processor_chain([
		dup16,
		dup4,
		dup6,
		dup7,
		dup28,
		dup29,
		dup30,
		dup31,
	]),
});

var msg14 = msg("000003", all8);

var part36 = match("MESSAGE#14:scan/0", "nwparser.payload", "scan: %{web_domain}[%{saddr}] %{id->} %{fld1->} %{fld2->} %{action->} %{fld8->} %{from->} %{to->} %{fld4->} %{fld3->} %{resultcode->} %{fld7->} %{p0}");

var all9 = all_match({
	processors: [
		part36,
		dup33,
	],
	on_success: processor_chain([
		dup16,
		dup4,
		dup6,
		dup7,
		dup28,
		dup29,
		dup30,
		dup31,
	]),
});

var msg15 = msg("scan", all9);

var select10 = linear_select([
	msg14,
	msg15,
]);

var part37 = match("MESSAGE#15:000004", "nwparser.payload", "web: Ret Policy Summary (Del:%{fld1->} Kept:%{fld2})", processor_chain([
	dup17,
	dup4,
]));

var msg16 = msg("000004", part37);

var part38 = match("MESSAGE#16:000005", "nwparser.payload", "web: [%{saddr}] FAILED_LOGIN (%{username})", processor_chain([
	setc("eventcategory","1401030000"),
	dup18,
	dup19,
	dup20,
	setc("ec_outcome","Failure"),
	dup4,
	setc("action","FAILED_LOGIN"),
]));

var msg17 = msg("000005", part38);

var part39 = match("MESSAGE#17:000006", "nwparser.payload", "web: Retention violating accounts: %{fld1->} total", processor_chain([
	setc("eventcategory","1605000000"),
	dup4,
]));

var msg18 = msg("000006", part39);

var part40 = match("MESSAGE#18:000007", "nwparser.payload", "web: [%{saddr}] global CHANGE %{category->} (%{info})", processor_chain([
	dup17,
	dup4,
	setc("action","CHANGE"),
]));

var msg19 = msg("000007", part40);

var part41 = match("MESSAGE#19:000029", "nwparser.payload", "web: [%{saddr}] LOGOUT (%{username})", processor_chain([
	setc("eventcategory","1401070000"),
	dup18,
	setc("ec_activity","Logoff"),
	dup20,
	dup4,
	setc("action","LOGOUT"),
]));

var msg20 = msg("000029", part41);

var part42 = match("MESSAGE#20:000030", "nwparser.payload", "web: [%{saddr}] LOGIN (%{username})", processor_chain([
	setc("eventcategory","1401060000"),
	dup18,
	dup19,
	dup20,
	dup4,
	setc("action","LOGIN"),
]));

var msg21 = msg("000030", part42);

var select11 = linear_select([
	msg16,
	msg17,
	msg18,
	msg19,
	msg20,
	msg21,
]);

var part43 = match("MESSAGE#21:000008", "nwparser.payload", "notify/smtp[%{fld14}]: %{saddr->} %{fld1->} %{fld2->} %{action->} %{fld4->} %{fld3->} %{sessionid->} %{bytes->} %{version->} %{from->} %{info}", processor_chain([
	dup13,
	dup4,
	dup32,
	dup30,
	dup31,
]));

var msg22 = msg("000008", part43);

var part44 = match("MESSAGE#22:reports", "nwparser.payload", "reports: REPORTS (%{process}) queued as %{fld1}", processor_chain([
	dup16,
	dup4,
	setc("event_description","report queued"),
]));

var msg23 = msg("reports", part44);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"inbound/pass1": select3,
		"notify/smtp": msg22,
		"outbound/smtp": select9,
		"reports": msg23,
		"scan": select10,
		"web": select11,
	}),
]);

var part45 = match("MESSAGE#0:000001/1_0", "nwparser.p0", "%{fld3->} %{resultcode->} %{info}");

var part46 = match_copy("MESSAGE#0:000001/1_1", "nwparser.p0", "info");

var part47 = match("MESSAGE#13:000003/1_0", "nwparser.p0", "SZ:%{fld9->} SUBJ:%{subject}");

var select12 = linear_select([
	dup1,
	dup2,
]);

var select13 = linear_select([
	dup15,
	dup2,
]);
