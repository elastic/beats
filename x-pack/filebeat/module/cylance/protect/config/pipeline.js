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

var map_getEventLegacyCategoryName = {
	keyvaluepairs: {
		"1003010000": constant("Attacks.Malicious Code.Virus"),
		"1401060000": constant("User.Activity.Successful Logins"),
		"1502000000": constant("Policies.Rules"),
		"1502030000": constant("Policies.Rules.Added"),
		"1600000000": constant("System"),
		"1609000000": constant("System.Alerts"),
		"1701000000": constant("Config.Changes"),
		"1804000000": constant("Network.Devices"),
		"1804010000": constant("Network.Devices.Additions"),
		"1804020000": constant("Network.Devices.Removals"),
	},
	"default": constant("Other.Default"),
};

var map_getEventLegacyCategory = {
	keyvaluepairs: {
		"Alert": constant("1609000000"),
		"Device Policy Assigned": constant("1502000000"),
		"Device Updated": constant("1804010000"),
		"DeviceEdit": dup20,
		"DeviceRemove": constant("1804020000"),
		"LoginSuccess": constant("1401060000"),
		"PolicyAdd": constant("1502030000"),
		"Registration": dup21,
		"SyslogSettingsSave": dup20,
		"SystemSecurity": constant("1600000000"),
		"ThreatUpdated": dup22,
		"ZoneAdd": dup20,
		"ZoneAddDevice": dup20,
		"fullaccess": dup21,
		"pechange": dup20,
		"threat_changed": dup22,
		"threat_found": dup22,
		"threat_quarantined": dup22,
	},
	"default": constant("1901000000"),
};

var dup1 = setc("messageid","CylancePROTECT");

var dup2 = match("MESSAGE#0:CylancePROTECT:01/0", "nwparser.payload", "%{fld13->} %{fld14->} %{p0}");

var dup3 = match("MESSAGE#0:CylancePROTECT:01/1_0", "nwparser.p0", "[%{fld2}] Event Type: AuditLog, Event Name: %{p0}");

var dup4 = match("MESSAGE#0:CylancePROTECT:01/1_1", "nwparser.p0", "%{fld5->} Event Type: AuditLog, Event Name: %{p0}");

var dup5 = setc("eventcategory","1901000000");

var dup6 = setc("vendor_event_cat"," AuditLog");

var dup7 = date_time({
	dest: "event_time",
	args: ["hdate","htime"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup8 = field("event_type");

var dup9 = field("event_cat");

var dup10 = match("MESSAGE#1:CylancePROTECT:02/2", "nwparser.p0", "%{event_type}, Message: %{p0}");

var dup11 = match("MESSAGE#8:CylancePROTECT:09/1_0", "nwparser.p0", "[%{fld2}] Event Type: ScriptControl, Event Name: %{p0}");

var dup12 = match("MESSAGE#8:CylancePROTECT:09/1_1", "nwparser.p0", "%{fld5->} Event Type: ScriptControl, Event Name: %{p0}");

var dup13 = match_copy("MESSAGE#8:CylancePROTECT:09/3_1", "nwparser.p0", "info");

var dup14 = match("MESSAGE#11:CylancePROTECT:15/1_0", "nwparser.p0", "[%{fld2}] Event Type: %{p0}");

var dup15 = match("MESSAGE#11:CylancePROTECT:15/1_1", "nwparser.p0", "%{fld5->} Event Type: %{p0}");

var dup16 = match("MESSAGE#13:CylancePROTECT:13/3_0", "nwparser.p0", "%{os->} Zone Names: %{info}");

var dup17 = match_copy("MESSAGE#13:CylancePROTECT:13/3_1", "nwparser.p0", "os");

var dup18 = date_time({
	dest: "event_time",
	args: ["hmonth","hdate","hhour","hmin","hsec"],
	fmts: [
		[dB,dF,dN,dU,dO],
	],
});

var dup19 = match("MESSAGE#22:CylancePROTECT:22/2_0", "nwparser.p0", "%{info}, Device Id: %{fld3}");

var dup20 = constant("1701000000");

var dup21 = constant("1804000000");

var dup22 = constant("1003010000");

var dup23 = linear_select([
	dup3,
	dup4,
]);

var dup24 = lookup({
	dest: "nwparser.event_cat",
	map: map_getEventLegacyCategory,
	key: dup8,
});

var dup25 = lookup({
	dest: "nwparser.event_cat_name",
	map: map_getEventLegacyCategoryName,
	key: dup9,
});

var dup26 = linear_select([
	dup11,
	dup12,
]);

var dup27 = linear_select([
	dup14,
	dup15,
]);

var dup28 = linear_select([
	dup16,
	dup17,
]);

var dup29 = linear_select([
	dup19,
	dup13,
]);

var hdr1 = match("HEADER#0:0001", "message", "%{hday}-%{hmonth}-%{hyear->} %{hhour}:%{hmin}:%{hsec->} %{hseverity->} %{hhost->} %{hfld2->} \u003c\u003c%{fld44}>%{hfld3->} %{hdate}T%{htime}.%{hfld4->} %{hostname->} CylancePROTECT %{payload}", processor_chain([
	setc("header_id","0001"),
	dup1,
]));

var hdr2 = match("HEADER#1:0002", "message", "%{hfld1->} %{hdate}T%{htime}.%{hfld2->} %{hostname->} CylancePROTECT %{payload}", processor_chain([
	setc("header_id","0002"),
	dup1,
]));

var hdr3 = match("HEADER#2:0004", "message", "%{hdate}T%{htime}.%{hfld2->} %{hostname->} CylancePROTECT %{payload}", processor_chain([
	setc("header_id","0004"),
	dup1,
]));

var hdr4 = match("HEADER#3:0003", "message", "%{hmonth->} %{hdate->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} CylancePROTECT Event Type:%{vendor_event_cat}, %{payload}", processor_chain([
	setc("header_id","0003"),
	dup1,
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
]);

var part1 = match("MESSAGE#0:CylancePROTECT:01/2", "nwparser.p0", "%{event_type}, Message: S%{p0}");

var part2 = match("MESSAGE#0:CylancePROTECT:01/3_0", "nwparser.p0", "ource: %{product}; SHA256: %{p0}");

var part3 = match("MESSAGE#0:CylancePROTECT:01/3_1", "nwparser.p0", "HA256: %{p0}");

var select2 = linear_select([
	part2,
	part3,
]);

var part4 = match("MESSAGE#0:CylancePROTECT:01/4", "nwparser.p0", "%{checksum}; %{p0}");

var part5 = match("MESSAGE#0:CylancePROTECT:01/5_0", "nwparser.p0", "Category: %{category}; Reason: %{p0}");

var part6 = match("MESSAGE#0:CylancePROTECT:01/5_1", "nwparser.p0", "Reason: %{p0}");

var select3 = linear_select([
	part5,
	part6,
]);

var part7 = match("MESSAGE#0:CylancePROTECT:01/6", "nwparser.p0", "%{result}, User: %{user_fname->} %{user_lname->} (%{mail_id})");

var all1 = all_match({
	processors: [
		dup2,
		dup23,
		part1,
		select2,
		part4,
		select3,
		part7,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg1 = msg("CylancePROTECT:01", all1);

var part8 = match("MESSAGE#1:CylancePROTECT:02/3_0", "nwparser.p0", "Device: %{node}; SHA256: %{p0}");

var part9 = match("MESSAGE#1:CylancePROTECT:02/3_1", "nwparser.p0", "Policy: %{policyname}; SHA256: %{p0}");

var select4 = linear_select([
	part8,
	part9,
]);

var part10 = match("MESSAGE#1:CylancePROTECT:02/4_0", "nwparser.p0", "%{checksum}; Category: %{category}, User: %{p0}");

var part11 = match("MESSAGE#1:CylancePROTECT:02/4_1", "nwparser.p0", "%{checksum}, User: %{p0}");

var select5 = linear_select([
	part10,
	part11,
]);

var part12 = match("MESSAGE#1:CylancePROTECT:02/5", "nwparser.p0", ")%{mail_id->} (%{user_lname->} %{user_fname}");

var all2 = all_match({
	processors: [
		dup2,
		dup23,
		dup10,
		select4,
		select5,
		part12,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg2 = msg("CylancePROTECT:02", all2);

var part13 = match("MESSAGE#2:CylancePROTECT:03/3_0", "nwparser.p0", "Devices: %{node},%{p0}");

var part14 = match("MESSAGE#2:CylancePROTECT:03/3_1", "nwparser.p0", "Device: %{node};%{p0}");

var part15 = match("MESSAGE#2:CylancePROTECT:03/3_2", "nwparser.p0", "Policy: %{policyname},%{p0}");

var select6 = linear_select([
	part13,
	part14,
	part15,
]);

var part16 = match("MESSAGE#2:CylancePROTECT:03/4", "nwparser.p0", "%{}User: %{user_fname->} %{user_lname->} (%{mail_id})");

var all3 = all_match({
	processors: [
		dup2,
		dup23,
		dup10,
		select6,
		part16,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg3 = msg("CylancePROTECT:03", all3);

var part17 = match("MESSAGE#3:CylancePROTECT:04/2", "nwparser.p0", "%{event_type}, Message: Zone: %{info}; Policy: %{policyname}; Value: %{fld3}, User: %{user_fname->} %{user_lname->} (%{mail_id})");

var all4 = all_match({
	processors: [
		dup2,
		dup23,
		part17,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg4 = msg("CylancePROTECT:04", all4);

var part18 = match("MESSAGE#4:CylancePROTECT:05/3_0", "nwparser.p0", "Policy Assigned:%{signame}; Devices: %{node->} , User: %{p0}");

var part19 = match("MESSAGE#4:CylancePROTECT:05/3_1", "nwparser.p0", "Provider: %{product}, Source IP: %{saddr}, User: %{p0}");

var part20 = match("MESSAGE#4:CylancePROTECT:05/3_2", "nwparser.p0", "%{info}, User: %{p0}");

var select7 = linear_select([
	part18,
	part19,
	part20,
]);

var part21 = match("MESSAGE#4:CylancePROTECT:05/4", "nwparser.p0", "%{user_fname->} %{user_lname->} (%{mail_id})");

var all5 = all_match({
	processors: [
		dup2,
		dup23,
		dup10,
		select7,
		part21,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg5 = msg("CylancePROTECT:05", all5);

var part22 = match("MESSAGE#5:CylancePROTECT:06/2", "nwparser.p0", "%{event_type}, Message: The Device: %{node->} was auto assigned to the Zone: IP Address: %{p0}");

var part23 = match("MESSAGE#5:CylancePROTECT:06/3_0", "nwparser.p0", "Fake Devices, User: %{p0}");

var part24 = match("MESSAGE#5:CylancePROTECT:06/3_1", "nwparser.p0", "%{saddr}, User: %{p0}");

var select8 = linear_select([
	part23,
	part24,
]);

var part25 = match("MESSAGE#5:CylancePROTECT:06/4_0", "nwparser.p0", "(%{p0}");

var part26 = match("MESSAGE#5:CylancePROTECT:06/4_1", "nwparser.p0", "%{user_fname->} %{user_lname->} (%{p0}");

var select9 = linear_select([
	part25,
	part26,
]);

var part27 = match("MESSAGE#5:CylancePROTECT:06/5", "nwparser.p0", ")%{mail_id}");

var all6 = all_match({
	processors: [
		dup2,
		dup23,
		part22,
		select8,
		select9,
		part27,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup24,
		dup25,
	]),
});

var msg6 = msg("CylancePROTECT:06", all6);

var part28 = match("MESSAGE#6:CylancePROTECT:07/1_0", "nwparser.p0", "[%{fld2}] Event Type: ExploitAttempt, Event Name: %{p0}");

var part29 = match("MESSAGE#6:CylancePROTECT:07/1_1", "nwparser.p0", "%{fld5->} Event Type: ExploitAttempt, Event Name: %{p0}");

var select10 = linear_select([
	part28,
	part29,
]);

var part30 = match("MESSAGE#6:CylancePROTECT:07/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, IP Address: (%{saddr}), Action: %{action}, Process ID: %{process_id}, Process Name: %{process}, User Name: %{username}, Violation Type: %{signame}, Zone Names: %{info}");

var all7 = all_match({
	processors: [
		dup2,
		select10,
		part30,
	],
	on_success: processor_chain([
		dup5,
		setc("vendor_event_cat"," ExploitAttempt"),
		dup7,
		dup24,
		dup25,
	]),
});

var msg7 = msg("CylancePROTECT:07", all7);

var part31 = match("MESSAGE#7:CylancePROTECT:08/1_0", "nwparser.p0", "[%{fld2}] Event Type: DeviceControl, Event Name: %{p0}");

var part32 = match("MESSAGE#7:CylancePROTECT:08/1_1", "nwparser.p0", "%{fld5->} Event Type: DeviceControl, Event Name: %{p0}");

var select11 = linear_select([
	part31,
	part32,
]);

var part33 = match("MESSAGE#7:CylancePROTECT:08/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, External Device Type: %{fld3}, External Device Vendor ID: %{fld18}, External Device Name: %{fld4}, External Device Product ID: %{fld17}, External Device Serial Number: %{serial_number}, Zone Names: %{info}");

var all8 = all_match({
	processors: [
		dup2,
		select11,
		part33,
	],
	on_success: processor_chain([
		dup5,
		setc("vendor_event_cat"," DeviceControl"),
		dup7,
		dup24,
		dup25,
	]),
});

var msg8 = msg("CylancePROTECT:08", all8);

var part34 = match("MESSAGE#8:CylancePROTECT:09/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, File Path: %{directory}, Interpreter: %{application}, Interpreter Version: %{version->} (%{fld3}), Zone Names: %{p0}");

var part35 = match("MESSAGE#8:CylancePROTECT:09/3_0", "nwparser.p0", "%{info}, User Name: %{username}");

var select12 = linear_select([
	part35,
	dup13,
]);

var all9 = all_match({
	processors: [
		dup2,
		dup26,
		part34,
		select12,
	],
	on_success: processor_chain([
		dup5,
		setc("vendor_event_cat"," ScriptControl"),
		dup7,
		dup24,
		dup25,
	]),
});

var msg9 = msg("CylancePROTECT:09", all9);

var part36 = match("MESSAGE#9:CylancePROTECT:10/1_0", "nwparser.p0", "[%{fld2}] Event Type: Threat, Event Name: %{p0}");

var part37 = match("MESSAGE#9:CylancePROTECT:10/1_1", "nwparser.p0", "%{fld4->} Event Type: Threat, Event Name: %{p0}");

var select13 = linear_select([
	part36,
	part37,
]);

var part38 = match("MESSAGE#9:CylancePROTECT:10/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, IP Address: (%{saddr}), File Name: %{filename}, Path: %{directory}, Drive Type: %{fld1}, SHA256: %{checksum}, MD5: %{fld3}, Status: %{event_state}, Cylance Score: %{reputation_num}, Found Date: %{fld5}, File Type: %{filetype}, Is Running: %{fld6}, Auto Run: %{fld7}, Detected By: %{fld8}, Zone Names: %{info}, Is Malware: %{fld10}, Is Unique To Cylance: %{fld11}, Threat Classification: %{sigtype}");

var all10 = all_match({
	processors: [
		dup2,
		select13,
		part38,
	],
	on_success: processor_chain([
		dup5,
		setc("vendor_event_cat"," Threat"),
		dup7,
		dup24,
		dup25,
	]),
});

var msg10 = msg("CylancePROTECT:10", all10);

var part39 = match("MESSAGE#10:CylancePROTECT:11/1_0", "nwparser.p0", "[%{fld2}] Event Type: AppControl, Event Name: %{p0}");

var part40 = match("MESSAGE#10:CylancePROTECT:11/1_1", "nwparser.p0", "%{fld5->} Event Type: AppControl, Event Name: %{p0}");

var select14 = linear_select([
	part39,
	part40,
]);

var part41 = match("MESSAGE#10:CylancePROTECT:11/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, IP Address: (%{saddr}), Action: %{action}, Action Type: %{fld3}, File Path: %{directory}, SHA256: %{checksum}, Zone Names: %{info}");

var all11 = all_match({
	processors: [
		dup2,
		select14,
		part41,
	],
	on_success: processor_chain([
		dup5,
		setc("vendor_event_cat"," AppControl"),
		dup24,
		dup25,
	]),
});

var msg11 = msg("CylancePROTECT:11", all11);

var part42 = match("MESSAGE#11:CylancePROTECT:15/2", "nwparser.p0", "%{vendor_event_cat}, Event Name: %{event_type}, Threat Class: %{sigtype}, Threat Subclass: %{fld7}, SHA256: %{checksum}, MD5: %{fld8}");

var all12 = all_match({
	processors: [
		dup2,
		dup27,
		part42,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg12 = msg("CylancePROTECT:15", all12);

var part43 = match("MESSAGE#12:CylancePROTECT:14/2", "nwparser.p0", "%{vendor_event_cat}, Event Name: %{event_type}, Device Names: (%{node}), Policy Name: %{policyname}, User: %{user_fname->} %{user_lname->} (%{mail_id})");

var all13 = all_match({
	processors: [
		dup2,
		dup27,
		part43,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg13 = msg("CylancePROTECT:14", all13);

var part44 = match("MESSAGE#13:CylancePROTECT:13/2", "nwparser.p0", "%{vendor_event_cat}, Event Name: %{event_type}, Device Name: %{node}, Agent Version: %{fld6}, IP Address: (%{saddr}, %{fld15}), MAC Address: (%{macaddr}, %{fld16}), Logged On Users: (%{username}), OS: %{p0}");

var all14 = all_match({
	processors: [
		dup2,
		dup27,
		part44,
		dup28,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg14 = msg("CylancePROTECT:13", all14);

var part45 = match("MESSAGE#14:CylancePROTECT:16/2", "nwparser.p0", "%{vendor_event_cat}, Event Name: %{event_type}, Device Name: %{node}, Agent Version: %{fld1}, IP Address: (%{saddr}), MAC Address: (%{macaddr}), Logged On Users: (%{username}), OS: %{p0}");

var all15 = all_match({
	processors: [
		dup2,
		dup27,
		part45,
		dup28,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg15 = msg("CylancePROTECT:16", all15);

var part46 = match("MESSAGE#15:CylancePROTECT:25/2", "nwparser.p0", "%{event_type}, Device Name: %{node}, File Path: %{directory}, Interpreter: %{application}, Interpreter Version: %{version}, Zone Names: %{info}, User Name: %{username}");

var all16 = all_match({
	processors: [
		dup2,
		dup26,
		part46,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg16 = msg("CylancePROTECT:25", all16);

var part47 = match("MESSAGE#16:CylancePROTECT:12/2", "nwparser.p0", "%{vendor_event_cat}, Event Name: %{event_type}, %{p0}");

var part48 = match("MESSAGE#16:CylancePROTECT:12/3_0", "nwparser.p0", "Device Name: %{node}, Zone Names:%{info}");

var part49 = match("MESSAGE#16:CylancePROTECT:12/3_1", "nwparser.p0", "Device Name: %{node}");

var part50 = match_copy("MESSAGE#16:CylancePROTECT:12/3_2", "nwparser.p0", "fld1");

var select15 = linear_select([
	part48,
	part49,
	part50,
]);

var all17 = all_match({
	processors: [
		dup2,
		dup27,
		part47,
		select15,
	],
	on_success: processor_chain([
		dup5,
		dup7,
		dup24,
		dup25,
	]),
});

var msg17 = msg("CylancePROTECT:12", all17);

var part51 = match("MESSAGE#17:CylancePROTECT:17/0", "nwparser.payload", "Event Name:%{event_type}, Device Name:%{node}, File Path:%{filename}, Interpreter:%{application}, Interpreter Version:%{version}, Zone Names:%{info}, User Name: %{p0}");

var part52 = match("MESSAGE#17:CylancePROTECT:17/1_0", "nwparser.p0", "%{username}, Device Id: %{fld3}, Policy Name: %{policyname}");

var part53 = match_copy("MESSAGE#17:CylancePROTECT:17/1_1", "nwparser.p0", "username");

var select16 = linear_select([
	part52,
	part53,
]);

var all18 = all_match({
	processors: [
		part51,
		select16,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg18 = msg("CylancePROTECT:17", all18);

var part54 = match("MESSAGE#18:CylancePROTECT:18", "nwparser.payload", "Event Name:%{event_type}, Device Name:%{node}, Agent Version:%{fld1}, IP Address: (%{saddr}), MAC Address: (%{macaddr}), Logged On Users: (%{username}), OS:%{os}, Zone Names:%{info}", processor_chain([
	dup5,
	dup18,
	dup24,
	dup25,
]));

var msg19 = msg("CylancePROTECT:18", part54);

var part55 = match("MESSAGE#19:CylancePROTECT:19/0", "nwparser.payload", "Event Name:%{event_type}, Device Name:%{node}, External Device Type:%{device}, External Device Vendor ID:%{fld2}, External Device Name:%{fld3}, External Device Product ID:%{fld4}, External Device Serial Number:%{serial_number}, Zone Names:%{p0}");

var part56 = match("MESSAGE#19:CylancePROTECT:19/1_0", "nwparser.p0", "%{info}, Device Id: %{fld5}, Policy Name: %{policyname}");

var select17 = linear_select([
	part56,
	dup13,
]);

var all19 = all_match({
	processors: [
		part55,
		select17,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg20 = msg("CylancePROTECT:19", all19);

var part57 = match("MESSAGE#20:CylancePROTECT:20/0", "nwparser.payload", "Event Name:%{event_type}, Message: %{p0}");

var part58 = match("MESSAGE#20:CylancePROTECT:20/1_0", "nwparser.p0", "The Device%{p0}");

var part59 = match("MESSAGE#20:CylancePROTECT:20/1_1", "nwparser.p0", "Device%{p0}");

var select18 = linear_select([
	part58,
	part59,
]);

var part60 = match("MESSAGE#20:CylancePROTECT:20/2", "nwparser.p0", ":%{node}was auto assigned to%{p0}");

var part61 = match("MESSAGE#20:CylancePROTECT:20/3_0", "nwparser.p0", " the%{p0}");

var part62 = match_copy("MESSAGE#20:CylancePROTECT:20/3_1", "nwparser.p0", "p0");

var select19 = linear_select([
	part61,
	part62,
]);

var part63 = match("MESSAGE#20:CylancePROTECT:20/4", "nwparser.p0", "%{}Zone:%{zone}, User:%{user_fname}");

var all20 = all_match({
	processors: [
		part57,
		select18,
		part60,
		select19,
		part63,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg21 = msg("CylancePROTECT:20", all20);

var part64 = match("MESSAGE#21:CylancePROTECT:21", "nwparser.payload", "Event Name:%{event_type}, Device Name:%{node}, IP Address: (%{saddr}), File Name:%{filename}, Path:%{directory}, Drive Type:%{fld1}, SHA256:%{checksum}, MD5:%{fld3}, Status:%{event_state}, Cylance Score:%{fld4}, Found Date:%{fld51}, File Type:%{fld6}, Is Running:%{fld7}, Auto Run:%{fld8}, Detected By:%{fld9}, Zone Names: (%{info}), Is Malware:%{fld10}, Is Unique To Cylance:%{fld11}, Threat Classification:%{sigtype}", processor_chain([
	dup5,
	dup18,
	dup24,
	dup25,
	date_time({
		dest: "effective_time",
		args: ["fld51"],
		fmts: [
			[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO,dQ],
		],
	}),
]));

var msg22 = msg("CylancePROTECT:21", part64);

var part65 = match("MESSAGE#22:CylancePROTECT:22/0", "nwparser.payload", "Event Name:%{p0}");

var part66 = match("MESSAGE#22:CylancePROTECT:22/1_0", "nwparser.p0", " %{event_type}, Device Name: %{device}, IP Address: (%{saddr}), Action: %{action}, Process ID: %{process_id}, Process Name: %{process}, User Name: %{username}, Violation Type: %{signame}, Zone Names:%{p0}");

var part67 = match("MESSAGE#22:CylancePROTECT:22/1_1", "nwparser.p0", "%{event_type}, Device Name:%{node}, Zone Names:%{p0}");

var select20 = linear_select([
	part66,
	part67,
]);

var all21 = all_match({
	processors: [
		part65,
		select20,
		dup29,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg23 = msg("CylancePROTECT:22", all21);

var part68 = match("MESSAGE#23:CylancePROTECT:23", "nwparser.payload", "Event Name:%{event_type}, Threat Class:%{sigtype}, Threat Subclass:%{fld1}, SHA256:%{checksum}, MD5:%{fld3}", processor_chain([
	dup5,
	dup18,
	dup24,
	dup25,
]));

var msg24 = msg("CylancePROTECT:23", part68);

var part69 = match("MESSAGE#24:CylancePROTECT:24/0", "nwparser.payload", "Event Name:%{event_type}, Message: Provider:%{fld3}, Source IP:%{saddr}, User: %{user_fname->} %{user_lname->} (%{mail_id})%{p0}");

var part70 = match("MESSAGE#24:CylancePROTECT:24/1_0", "nwparser.p0", "#015%{}");

var part71 = match_copy("MESSAGE#24:CylancePROTECT:24/1_1", "nwparser.p0", "");

var select21 = linear_select([
	part70,
	part71,
]);

var all22 = all_match({
	processors: [
		part69,
		select21,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg25 = msg("CylancePROTECT:24", all22);

var part72 = match("MESSAGE#25:CylancePROTECT:26/0", "nwparser.payload", "Event Name:%{event_type}, Device Message: Device: %{device}; Policy Changed: %{fld4->} to '%{policyname}', User: %{user_fname->} %{user_lname->} (%{mail_id}), Zone Names:%{p0}");

var all23 = all_match({
	processors: [
		part72,
		dup29,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg26 = msg("CylancePROTECT:26", all23);

var part73 = match("MESSAGE#26:CylancePROTECT:27/0", "nwparser.payload", "Event Name:%{event_type}, Device Message: Device: %{device}; Zones Removed: %{p0}");

var part74 = match("MESSAGE#26:CylancePROTECT:27/1_0", "nwparser.p0", "%{fld4}; Zones Added: %{fld5},%{p0}");

var part75 = match("MESSAGE#26:CylancePROTECT:27/1_1", "nwparser.p0", "%{fld4},%{p0}");

var select22 = linear_select([
	part74,
	part75,
]);

var part76 = match("MESSAGE#26:CylancePROTECT:27/2", "nwparser.p0", "%{}User: %{user_fname->} %{user_lname->} (%{mail_id}), Zone Names:%{p0}");

var part77 = match("MESSAGE#26:CylancePROTECT:27/3_0", "nwparser.p0", "%{info->} Device Id: %{fld3}");

var select23 = linear_select([
	part77,
	dup13,
]);

var all24 = all_match({
	processors: [
		part73,
		select22,
		part76,
		select23,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg27 = msg("CylancePROTECT:27", all24);

var part78 = match("MESSAGE#27:CylancePROTECT:28/0", "nwparser.payload", "Event Name:%{event_type}, Device Message: Device: %{device->} %{p0}");

var part79 = match("MESSAGE#27:CylancePROTECT:28/1_0", "nwparser.p0", "Agent Self Protection Level Changed: '%{change_old}' to '%{change_new}', User: %{p0}");

var part80 = match("MESSAGE#27:CylancePROTECT:28/1_1", "nwparser.p0", "User: %{p0}");

var select24 = linear_select([
	part79,
	part80,
]);

var part81 = match("MESSAGE#27:CylancePROTECT:28/2", "nwparser.p0", "),%{mail_id->} (%{user_lname->} %{user_fname->} Zone Names: %{info->} Device Id: %{fld3}");

var all25 = all_match({
	processors: [
		part78,
		select24,
		part81,
	],
	on_success: processor_chain([
		dup5,
		dup18,
		dup24,
		dup25,
	]),
});

var msg28 = msg("CylancePROTECT:28", all25);

var select25 = linear_select([
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
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"CylancePROTECT": select25,
	}),
]);

var part82 = match("MESSAGE#0:CylancePROTECT:01/0", "nwparser.payload", "%{fld13->} %{fld14->} %{p0}");

var part83 = match("MESSAGE#0:CylancePROTECT:01/1_0", "nwparser.p0", "[%{fld2}] Event Type: AuditLog, Event Name: %{p0}");

var part84 = match("MESSAGE#0:CylancePROTECT:01/1_1", "nwparser.p0", "%{fld5->} Event Type: AuditLog, Event Name: %{p0}");

var part85 = match("MESSAGE#1:CylancePROTECT:02/2", "nwparser.p0", "%{event_type}, Message: %{p0}");

var part86 = match("MESSAGE#8:CylancePROTECT:09/1_0", "nwparser.p0", "[%{fld2}] Event Type: ScriptControl, Event Name: %{p0}");

var part87 = match("MESSAGE#8:CylancePROTECT:09/1_1", "nwparser.p0", "%{fld5->} Event Type: ScriptControl, Event Name: %{p0}");

var part88 = match_copy("MESSAGE#8:CylancePROTECT:09/3_1", "nwparser.p0", "info");

var part89 = match("MESSAGE#11:CylancePROTECT:15/1_0", "nwparser.p0", "[%{fld2}] Event Type: %{p0}");

var part90 = match("MESSAGE#11:CylancePROTECT:15/1_1", "nwparser.p0", "%{fld5->} Event Type: %{p0}");

var part91 = match("MESSAGE#13:CylancePROTECT:13/3_0", "nwparser.p0", "%{os->} Zone Names: %{info}");

var part92 = match_copy("MESSAGE#13:CylancePROTECT:13/3_1", "nwparser.p0", "os");

var part93 = match("MESSAGE#22:CylancePROTECT:22/2_0", "nwparser.p0", "%{info}, Device Id: %{fld3}");

var select26 = linear_select([
	dup3,
	dup4,
]);

var select27 = linear_select([
	dup11,
	dup12,
]);

var select28 = linear_select([
	dup14,
	dup15,
]);

var select29 = linear_select([
	dup16,
	dup17,
]);

var select30 = linear_select([
	dup19,
	dup13,
]);
