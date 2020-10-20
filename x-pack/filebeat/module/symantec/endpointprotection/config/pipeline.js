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

var map_Protocol = {
	keyvaluepairs: {
		"1": constant("Others"),
		"2": constant("TCP"),
		"3": constant("UDP"),
		"4": constant("ICMP"),
	},
};

var map_Direction = {
	keyvaluepairs: {
		"0": constant("Unknown"),
		"1": constant("inbound"),
		"2": constant("outbound"),
	},
};

var map_Action = {
	keyvaluepairs: {
		"0": dup309,
		"1": constant("Block"),
		"2": constant("Ask"),
		"3": constant("Continue"),
		"4": constant("Terminate"),
	},
};

var map_Activity = {
	keyvaluepairs: {
		"0": dup309,
		"1": dup310,
		"3": dup309,
		"4": dup310,
	},
};

var dup1 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" "),
		field("data"),
		constant(".."),
		field("p0"),
	],
});

var dup2 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup3 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("messageid"),
		constant("."),
		field("fld2"),
		constant(" "),
		field("p0"),
	],
});

var dup4 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hfld1"),
		constant(". Traffic has been "),
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup5 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("fld1"),
		constant(" "),
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup6 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hname"),
		constant(","),
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup7 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("fld40"),
		constant(" "),
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup8 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hhost"),
		constant("^^"),
		field("p0"),
	],
});

var dup9 = match("MESSAGE#0:Active/1_0", "nwparser.p0", "%{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var dup10 = match_copy("MESSAGE#0:Active/1_1", "nwparser.p0", "domain");

var dup11 = setc("eventcategory","1001020100");

var dup12 = setf("hostname","hhost");

var dup13 = setf("shost","hshost");

var dup14 = setf("event_time_string","htime");

var dup15 = setf("msg","$MSG");

var dup16 = date_time({
	dest: "starttime",
	args: ["fld50","fld54"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup17 = date_time({
	dest: "endtime",
	args: ["fld16","fld19"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup18 = setc("event_description","Traffic from IP address blocked.");

var dup19 = setc("dclass_counter1_string","Occurences.");

var dup20 = setc("ec_subject","User");

var dup21 = setc("ec_theme","Authentication");

var dup22 = setc("ec_outcome","Success");

var dup23 = setf("username","husername");

var dup24 = setc("ec_activity","Logon");

var dup25 = setc("ec_outcome","Failure");

var dup26 = setc("eventcategory","1402040100");

var dup27 = setc("ec_activity","Delete");

var dup28 = setc("ec_theme","AccessControl");

var dup29 = setc("event_description","Password of System administrator has been changed.");

var dup30 = setc("ec_activity","Modify");

var dup31 = setc("ec_theme","Password");

var dup32 = setc("eventcategory","1001020305");

var dup33 = setc("event_description","Traffic has been allowed from this process.");

var dup34 = setc("direction","Inbound");

var dup35 = setc("direction","Outbound");

var dup36 = setc("eventcategory","1607000000");

var dup37 = setc("ec_activity","Deny");

var dup38 = setc("ec_theme","TEV");

var dup39 = setc("event_description","Traffic has been blocked for this application.");

var dup40 = date_time({
	dest: "event_time",
	args: ["hmonth","hday","hhour","hmin","hsec"],
	fmts: [
		[dB,dF,dN,dU,dO],
	],
});

var dup41 = date_time({
	dest: "starttime",
	args: ["fld50","fld52"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup42 = date_time({
	dest: "endtime",
	args: ["fld51","fld53"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup43 = setc("eventcategory","1605020000");

var dup44 = setc("ec_theme","ALM");

var dup45 = setc("ec_subject","SignatureDB");

var dup46 = setc("eventcategory","1103000000");

var dup47 = setc("dclass_counter1_string","Occurences");

var dup48 = setc("fld14","1");

var dup49 = field("fld14");

var dup50 = match("MESSAGE#15:Somebody:01/1_0", "nwparser.p0", "%{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var dup51 = setc("fld14","0");

var dup52 = setc("fld14","2");

var dup53 = setc("eventcategory","1605000000");

var dup54 = setc("event_description","Application and Device Control is ready.");

var dup55 = date_time({
	dest: "event_time",
	args: ["fld5"],
	fmts: [
		[dB,dF,dZ,dW],
	],
});

var dup56 = setc("ec_activity","Disable");

var dup57 = setc("event_description","Proactive Threat Protection has been disabled");

var dup58 = setc("event_description","Application has changed since the last time you opened it");

var dup59 = match("MESSAGE#27:Application:06/1_0", "nwparser.p0", "\"Intrusion URL: %{url}\",Intrusion Payload URL:%{fld25}");

var dup60 = match("MESSAGE#27:Application:06/1_1", "nwparser.p0", "Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var dup61 = match("MESSAGE#27:Application:06/1_2", "nwparser.p0", "Intrusion URL: %{url}");

var dup62 = setc("event_description","Traffic has been blocked from application.");

var dup63 = match("MESSAGE#31:scanning:01/1_0", "nwparser.p0", "%{url},Intrusion Payload URL:%{fld25}");

var dup64 = match_copy("MESSAGE#31:scanning:01/1_1", "nwparser.p0", "url");

var dup65 = setc("eventcategory","1401000000");

var dup66 = setc("event_description","Somebody is scanning your computer.");

var dup67 = match("MESSAGE#33:Informational/1_1", "nwparser.p0", "Domain:%{p0}");

var dup68 = setc("event_description","Informational: File Download Hash.");

var dup69 = setc("eventcategory","1001030000");

var dup70 = setc("event_description","Web Attack : Malvertisement Website Redirect");

var dup71 = match("MESSAGE#38:Web_Attack:16/1_1", "nwparser.p0", ":%{p0}");

var dup72 = setc("event_description","Web Attack: Mass Injection Website.");

var dup73 = setc("event_description","Fake App Attack: Misleading Application Website.");

var dup74 = setc("eventcategory","1603110000");

var dup75 = setc("event_description","The most recent Host Integrity content has not completed a download or cannot be authenticated.");

var dup76 = match("MESSAGE#307:process:12/1_0", "nwparser.p0", "\"%{p0}");

var dup77 = match_copy("MESSAGE#307:process:12/1_1", "nwparser.p0", "p0");

var dup78 = match("MESSAGE#307:process:12/4", "nwparser.p0", ",Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},%{p0}");

var dup79 = match("MESSAGE#307:process:12/5_0", "nwparser.p0", "Intrusion ID: %{fld33},Begin: %{p0}");

var dup80 = match("MESSAGE#307:process:12/5_1", "nwparser.p0", "%{fld33},Begin: %{p0}");

var dup81 = match("MESSAGE#307:process:12/6", "nwparser.p0", "%{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var dup82 = setc("result","Traffic has not been blocked from application.");

var dup83 = setc("result","Traffic has been blocked from application.");

var dup84 = setc("eventcategory","1002000000");

var dup85 = setc("event_description","Denial of Service 'Smurf' attack detected.");

var dup86 = setc("eventcategory","1603000000");

var dup87 = setf("hostip","hhostip");

var dup88 = setc("event_description","Host Integrity check passed");

var dup89 = setc("event_description","Host Integrity check failed.");

var dup90 = match("MESSAGE#21:Applied/1_0", "nwparser.p0", ",Event time:%{fld17->} %{fld18}");

var dup91 = match_copy("MESSAGE#21:Applied/1_1", "nwparser.p0", "");

var dup92 = setc("eventcategory","1702010000");

var dup93 = date_time({
	dest: "event_time",
	args: ["fld17","fld18"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup94 = setf("hostip","hhost");

var dup95 = setc("eventcategory","1701010000");

var dup96 = setc("ec_activity","Create");

var dup97 = setc("ec_theme","Configuration");

var dup98 = match("MESSAGE#23:blocked:01/1_0", "nwparser.p0", "\"Location: %{p0}");

var dup99 = match("MESSAGE#23:blocked:01/1_1", "nwparser.p0", "Location: %{p0}");

var dup100 = match("MESSAGE#52:blocked/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain}");

var dup101 = match("MESSAGE#190:Local::01/0_0", "nwparser.payload", "%{fld4},MD-5:%{fld5},Local:%{p0}");

var dup102 = match("MESSAGE#190:Local::01/0_1", "nwparser.payload", "Local:%{p0}");

var dup103 = setc("event_description","Active Response");

var dup104 = setc("dclass_counter1_string","Occurrences");

var dup105 = match("MESSAGE#192:Local:/1_0", "nwparser.p0", "Rule: %{rulename},Location: %{p0}");

var dup106 = match("MESSAGE#192:Local:/1_1", "nwparser.p0", " \"Rule: %{rulename}\",Location: %{p0}");

var dup107 = match("MESSAGE#192:Local:/2", "nwparser.p0", "%{fld11},User: %{username},%{p0}");

var dup108 = match("MESSAGE#192:Local:/3_0", "nwparser.p0", "Domain: %{domain},Action: %{action}");

var dup109 = match("MESSAGE#192:Local:/3_1", "nwparser.p0", " Domain: %{domain}");

var dup110 = setc("eventcategory","1003010000");

var dup111 = call({
	dest: "nwparser.sigid_string",
	fn: STRCAT,
	args: [
		field("fld28"),
		constant("CVE-"),
		field("cve"),
	],
});

var dup112 = match("MESSAGE#198:Local::04/1_0", "nwparser.p0", "\"Intrusion URL: %{url}\",Intrusion Payload URL:%{p0}");

var dup113 = match("MESSAGE#198:Local::04/1_1", "nwparser.p0", "Intrusion URL: %{url},Intrusion Payload URL:%{p0}");

var dup114 = match_copy("MESSAGE#198:Local::04/2", "nwparser.p0", "fld25");

var dup115 = setc("ec_subject","Virus");

var dup116 = setc("ec_activity","Detect");

var dup117 = match("MESSAGE#205:Local::07/0", "nwparser.payload", "%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{network_service},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var dup118 = match("MESSAGE#206:Local::19/0", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{network_service},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var dup119 = match("MESSAGE#209:Local::03/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain}");

var dup120 = setc("eventcategory","1801000000");

var dup121 = setc("eventcategory","1401010000");

var dup122 = setf("shost","hsource");

var dup123 = setc("event_description","File Read Begin.");

var dup124 = setc("ec_subject","File");

var dup125 = setc("action","Read");

var dup126 = setc("event_description","Create Process.");

var dup127 = setc("event_description","File Write.");

var dup128 = setc("action","Write");

var dup129 = setf("saddr","hsaddr");

var dup130 = setc("event_description","File Read.");

var dup131 = setc("action","Delete");

var dup132 = setf("process","filename");

var dup133 = setc("event_description","File Write Begin.");

var dup134 = date_time({
	dest: "starttime",
	args: ["fld2","fld3"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup135 = date_time({
	dest: "endtime",
	args: ["fld4","fld5"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup136 = setc("eventcategory","1701020000");

var dup137 = setf("domain","hdomain");

var dup138 = setc("event_description","The client has downloaded file successfully.");

var dup139 = match("MESSAGE#64:client:05/0", "nwparser.payload", "The client will block traffic from IP address %{fld14->} for the next %{duration_string->} (from %{fld13})%{p0}");

var dup140 = match("MESSAGE#64:client:05/1_0", "nwparser.p0", ".,%{p0}");

var dup141 = match("MESSAGE#64:client:05/1_1", "nwparser.p0", " . ,%{p0}");

var dup142 = setf("shost","hclient");

var dup143 = setc("event_description","The client will block traffic.");

var dup144 = setc("event_description","The client has successfully downloaded and applied a license file");

var dup145 = match("MESSAGE#70:Commercial/0", "nwparser.payload", "Commercial application detected,Computer name: %{p0}");

var dup146 = match("MESSAGE#70:Commercial/1_0", "nwparser.p0", "%{shost},IP Address: %{saddr},Detection type: %{p0}");

var dup147 = match("MESSAGE#70:Commercial/1_1", "nwparser.p0", "%{shost},Detection type: %{p0}");

var dup148 = match("MESSAGE#70:Commercial/2", "nwparser.p0", "%{severity},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var dup149 = match("MESSAGE#70:Commercial/3_0", "nwparser.p0", "\"%{filename}\",Actual action: %{p0}");

var dup150 = match("MESSAGE#70:Commercial/3_1", "nwparser.p0", "%{filename},Actual action: %{p0}");

var dup151 = match("MESSAGE#70:Commercial/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var dup152 = setf("threat_name","virusname");

var dup153 = date_time({
	dest: "recorded_time",
	args: ["fld19"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup154 = date_time({
	dest: "endtime",
	args: ["fld51"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup155 = setc("event_description","Commercial application detected");

var dup156 = setc("eventcategory","1701030000");

var dup157 = match("MESSAGE#76:Computer/0", "nwparser.payload", "IP Address: %{hostip},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{p0}");

var dup158 = setf("administrator","husername");

var dup159 = match("MESSAGE#78:Computer:03/1_0", "nwparser.p0", "\"%{filename}\",%{p0}");

var dup160 = match("MESSAGE#78:Computer:03/1_1", "nwparser.p0", "%{filename},%{p0}");

var dup161 = match("MESSAGE#79:Computer:02/2", "nwparser.p0", "%{severity},First Seen: %{fld55},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld13},Detection score:%{fld7},COH Engine Version: %{fld41},%{fld53},Permitted application reason: %{fld54},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},Risk Level: %{fld50},Detection Source: %{fld52},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var dup162 = setc("event_description","Security risk found");

var dup163 = date_time({
	dest: "event_time",
	args: ["fld5","fld6"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup164 = date_time({
	dest: "recorded_time",
	args: ["fld12"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup165 = setc("eventcategory","1701000000");

var dup166 = date_time({
	dest: "event_time",
	args: ["fld5","fld6"],
	fmts: [
		[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup167 = setc("event_description","Could not start service engine.");

var dup168 = setc("eventcategory","1603040000");

var dup169 = setc("event_description","Disconnected from Symantec Endpoint Protection Manager.");

var dup170 = setc("eventcategory","1402020200");

var dup171 = setc("eventcategory","1402020100");

var dup172 = setc("ec_activity","Enable");

var dup173 = setc("event_description","Failed to connect to the server.");

var dup174 = setc("eventcategory","1301000000");

var dup175 = setc("event_description","Failed to Login to Remote Site");

var dup176 = match("MESSAGE#250:Network:24/1_0", "nwparser.p0", "\"%{}");

var dup177 = setc("ec_subject","Group");

var dup178 = setc("ec_theme","UserGroup");

var dup179 = setc("eventcategory","1701070000");

var dup180 = setc("event_description","Host Integrity check is disabled.");

var dup181 = setc("event_description","Host Integrity failed but reported as pass");

var dup182 = match("MESSAGE#134:Host:09/1_1", "nwparser.p0", " Domain:%{p0}");

var dup183 = match("MESSAGE#135:Intrusion/1_0", "nwparser.p0", "is %{p0}");

var dup184 = setc("event_description","LiveUpdate");

var dup185 = setc("event_description","Submitting information to Symantec failed.");

var dup186 = match("MESSAGE#145:LiveUpdate:10/1_0", "nwparser.p0", ".,Event time:%{fld17->} %{fld18}");

var dup187 = setc("ec_outcome","Error");

var dup188 = setc("event_description","LiveUpdate encountered an error.");

var dup189 = setf("hostid","hhost");

var dup190 = setc("event_description","The latest SONAR Definitions update failed to load.");

var dup191 = match("MESSAGE#179:LiveUpdate:40/1_0", "nwparser.p0", "\",Event time:%{fld17->} %{fld18}");

var dup192 = date_time({
	dest: "event_time",
	args: ["fld5","fld6"],
	fmts: [
		[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO,dP],
	],
});

var dup193 = setc("event_description","Virus Found");

var dup194 = match("MESSAGE#432:Virus:02/1_1", "nwparser.p0", " %{p0}");

var dup195 = setc("event_description","Virus Definition File Update");

var dup196 = setf("event_description","hfld1");

var dup197 = match("MESSAGE#436:Virus:12/0", "nwparser.payload", "Virus found,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{p0}");

var dup198 = match("MESSAGE#436:Virus:12/1_0", "nwparser.p0", "\"%{fld1}\",Actual action: %{p0}");

var dup199 = match("MESSAGE#436:Virus:12/1_1", "nwparser.p0", "%{fld1},Actual action: %{p0}");

var dup200 = setc("event_description","Virus found");

var dup201 = match("MESSAGE#437:Virus:15/1_0", "nwparser.p0", "Intensive Protection Level: %{fld61},Certificate issuer: %{fld60},Certificate signer: %{fld62},Certificate thumbprint: %{fld63},Signing timestamp: %{fld64},Certificate serial number: %{fld65},Source: %{p0}");

var dup202 = match("MESSAGE#437:Virus:15/1_1", "nwparser.p0", "Source: %{p0}");

var dup203 = match("MESSAGE#438:Virus:13/3_0", "nwparser.p0", "\"Group: %{group}\",Server: %{p0}");

var dup204 = match("MESSAGE#438:Virus:13/3_1", "nwparser.p0", "Group: %{group},Server: %{p0}");

var dup205 = match("MESSAGE#438:Virus:13/4", "nwparser.p0", "%{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{p0}");

var dup206 = match("MESSAGE#438:Virus:13/5_0", "nwparser.p0", "%{filename_size},Category set: %{category},Category type: %{event_type}");

var dup207 = match_copy("MESSAGE#438:Virus:13/5_1", "nwparser.p0", "filename_size");

var dup208 = match("MESSAGE#440:Virus:14/0", "nwparser.payload", "Virus found,Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{p0}");

var dup209 = match("MESSAGE#441:Virus:05/1_0", "nwparser.p0", "\"%{info}\",Actual action: %{p0}");

var dup210 = match("MESSAGE#441:Virus:05/1_1", "nwparser.p0", "%{info},Actual action: %{p0}");

var dup211 = match("MESSAGE#218:Location/3_0", "nwparser.p0", "%{info},Event time:%{fld17->} %{fld18}");

var dup212 = match_copy("MESSAGE#218:Location/3_1", "nwparser.p0", "info");

var dup213 = setc("eventcategory","1701060000");

var dup214 = setc("event_description","Network Audit Search Unagented Hosts From NST Finished Abnormally.");

var dup215 = setc("event_description","Network Intrusion Prevention is malfunctioning");

var dup216 = match("MESSAGE#253:Network:27/1_0", "nwparser.p0", " by policy%{}");

var dup217 = setc("event_description","Generic Exploit Mitigation");

var dup218 = setc("event_description","No objects got swept.");

var dup219 = setc("event_description","Organization importing finished successfully.");

var dup220 = setc("event_description","Organization importing started.");

var dup221 = setc("event_description","Number of Group Update Providers");

var dup222 = setf("shost","hhostid");

var dup223 = setc("ec_theme","Policy");

var dup224 = setc("event_description","Policy has been added");

var dup225 = setc("event_description","Policy has been edited");

var dup226 = match("MESSAGE#296:Policy:deleted/1_0", "nwparser.p0", ",%{p0}");

var dup227 = setc("event_description","Potential risk found");

var dup228 = match("MESSAGE#298:Potential:02/0", "nwparser.payload", "Potential risk found,Computer name: %{p0}");

var dup229 = match("MESSAGE#299:Potential/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld20},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var dup230 = date_time({
	dest: "recorded_time",
	args: ["fld20"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup231 = match("MESSAGE#308:process:03/0", "nwparser.payload", "%{event_description}, process id: %{process_id->} Filename: %{filename->} The change was denied by user%{fld6}\"%{p0}");

var dup232 = setc("eventcategory","1606000000");

var dup233 = setc("event_description","Retry.");

var dup234 = setc("event_description","Successfully deleted the client install package");

var dup235 = setc("event_description","Risk Repair Failed");

var dup236 = setc("event_description","Risk Repaired");

var dup237 = setc("event_description","Scan Start/Stop");

var dup238 = setc("event_description","Scan Start");

var dup239 = setc("dclass_counter1_string","Infected Count.");

var dup240 = setc("dclass_counter2_string","Total File Count.");

var dup241 = setc("dclass_counter3_string","Threat Count.");

var dup242 = date_time({
	dest: "starttime",
	args: ["fld1"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup243 = setc("event_description","Scan");

var dup244 = setc("dclass_counter1_string","Infected");

var dup245 = setc("dclass_counter2_string","Files scanned");

var dup246 = setc("dclass_counter3_string","Threats");

var dup247 = setc("dclass_counter1_string","Risk Count.");

var dup248 = setc("dclass_counter2_string","Scan Count.");

var dup249 = match("MESSAGE#340:Scan:12/1_0", "nwparser.p0", "'%{context}',%{p0}");

var dup250 = match("MESSAGE#343:Security:03/0", "nwparser.payload", "Security risk found,Computer name: %{p0}");

var dup251 = match("MESSAGE#345:Security:05/0", "nwparser.payload", "Security risk found,IP Address: %{saddr},Computer name: %{shost},%{p0}");

var dup252 = match("MESSAGE#345:Security:05/7_0", "nwparser.p0", "%{filename_size},Category set: %{category},Category type: %{vendor_event_cat}");

var dup253 = setc("event_description","Compressed File");

var dup254 = setc("event_description","Stop serving as the Group Update Provider (proxy server).");

var dup255 = setc("event_description","Symantec AntiVirus Startup/Shutdown");

var dup256 = setc("eventcategory","1611000000");

var dup257 = setc("eventcategory","1610000000");

var dup258 = setc("event_description","services failed to start");

var dup259 = setc("eventcategory","1608010000");

var dup260 = match("MESSAGE#388:Symantec:26/0", "nwparser.payload", "Category: %{fld22},Symantec AntiVirus,%{p0}");

var dup261 = match("MESSAGE#388:Symantec:26/1_0", "nwparser.p0", "[Antivirus%{p0}");

var dup262 = match("MESSAGE#388:Symantec:26/1_1", "nwparser.p0", "\"[Antivirus%{p0}");

var dup263 = match("MESSAGE#389:Symantec:39/2", "nwparser.p0", "%{} %{p0}");

var dup264 = match("MESSAGE#389:Symantec:39/3_0", "nwparser.p0", "detection%{p0}");

var dup265 = match("MESSAGE#389:Symantec:39/3_1", "nwparser.p0", "advanced heuristic detection%{p0}");

var dup266 = match("MESSAGE#389:Symantec:39/5_0", "nwparser.p0", " Size (bytes): %{filename_size}.\",Event time:%{fld17->} %{fld18}");

var dup267 = match("MESSAGE#389:Symantec:39/5_2", "nwparser.p0", "Event time:%{fld17->} %{fld18}");

var dup268 = setc("ec_theme","Communication");

var dup269 = match("MESSAGE#410:Terminated/0_1", "nwparser.payload", ",%{p0}");

var dup270 = setc("event_description","Traffic from IP address is blocked.");

var dup271 = match("MESSAGE#416:Traffic:02/2", "nwparser.p0", "%{fld6},User: %{username},Domain: %{domain}");

var dup272 = setc("event_description","Unexpected server error.");

var dup273 = setc("event_description","Unsolicited incoming ARP reply detected.");

var dup274 = setc("event_description","Windows Version info.");

var dup275 = match("MESSAGE#455:Allowed:09/2_0", "nwparser.p0", "\"%{filename}\",User: %{p0}");

var dup276 = match("MESSAGE#455:Allowed:09/2_1", "nwparser.p0", "%{filename},User: %{p0}");

var dup277 = setc("event_description","File Write");

var dup278 = match("MESSAGE#457:Allowed:10/3_0", "nwparser.p0", "%{fld46},File size (%{fld10}): %{filename_size},Device ID: %{device}");

var dup279 = setc("event_description","File Delete");

var dup280 = setc("event_description","File Delete Begin.");

var dup281 = match("MESSAGE#505:Ping/0_0", "nwparser.payload", "\"\"%{action->} . Description: %{p0}");

var dup282 = match("MESSAGE#505:Ping/0_1", "nwparser.payload", "%{action->} . Description: %{p0}");

var dup283 = setc("dclass_counter1_string","Virus Count.");

var dup284 = date_time({
	dest: "event_time",
	args: ["fld1","fld2","fld3"],
	fmts: [
		[dG,dc("/"),dF,dc("/"),dY,dN,dc(":"),dU,dP],
	],
});

var dup285 = setc("event_description","Backup succeeded and finished.");

var dup286 = setc("event_description","Backup started.");

var dup287 = date_time({
	dest: "event_time",
	args: ["fld8"],
	fmts: [
		[dX],
	],
});

var dup288 = setc("ec_subject","Configuration");

var dup289 = setc("eventcategory","1801030000");

var dup290 = match("MESSAGE#639:303235080/1_0", "nwparser.p0", "%{event_description->} [name]:%{obj_name->} [class]:%{obj_type->} [guid]:%{hardware_id->} [deviceID]:%{info}^^%{p0}");

var dup291 = match("MESSAGE#639:303235080/1_1", "nwparser.p0", "%{event_description}. %{info}^^%{p0}");

var dup292 = match("MESSAGE#639:303235080/1_2", "nwparser.p0", "%{event_description}^^%{p0}");

var dup293 = match("MESSAGE#639:303235080/2", "nwparser.p0", "%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}");

var dup294 = setc("eventcategory","1803000000");

var dup295 = setc("ec_subject","NetworkComm");

var dup296 = field("fld17");

var dup297 = setc("event_description","Block all other IP traffic and log");

var dup298 = setc("rulename","Block all other IP traffic and log");

var dup299 = field("fld13");

var dup300 = date_time({
	dest: "starttime",
	args: ["fld15"],
	fmts: [
		[dX],
	],
});

var dup301 = date_time({
	dest: "endtime",
	args: ["fld16"],
	fmts: [
		[dX],
	],
});

var dup302 = setc("dclass_counter1_string","No. of attacks");

var dup303 = setc("event_description","Block Local File Sharing to external computers");

var dup304 = setc("event_description","Block all other traffic");

var dup305 = match("MESSAGE#674:238/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{p0}");

var dup306 = field("fld11");

var dup307 = setc("dclass_counter1_string","No. of events repeated");

var dup308 = setf("filename","parent_process");

var dup309 = constant("Allow");

var dup310 = constant("Deny");

var dup311 = linear_select([
	dup9,
	dup10,
]);

var dup312 = lookup({
	dest: "nwparser.direction",
	map: map_Direction,
	key: dup49,
});

var dup313 = linear_select([
	dup50,
	dup10,
]);

var dup314 = linear_select([
	dup59,
	dup60,
	dup61,
]);

var dup315 = linear_select([
	dup63,
	dup64,
]);

var dup316 = linear_select([
	dup76,
	dup77,
]);

var dup317 = linear_select([
	dup79,
	dup80,
]);

var dup318 = linear_select([
	dup90,
	dup91,
]);

var dup319 = linear_select([
	dup98,
	dup99,
]);

var dup320 = linear_select([
	dup101,
	dup102,
]);

var dup321 = linear_select([
	dup105,
	dup106,
]);

var dup322 = linear_select([
	dup108,
	dup109,
]);

var dup323 = linear_select([
	dup112,
	dup113,
]);

var dup324 = linear_select([
	dup140,
	dup141,
]);

var dup325 = linear_select([
	dup146,
	dup147,
]);

var dup326 = linear_select([
	dup149,
	dup150,
]);

var dup327 = linear_select([
	dup159,
	dup160,
]);

var dup328 = linear_select([
	dup198,
	dup199,
]);

var dup329 = linear_select([
	dup201,
	dup202,
]);

var dup330 = linear_select([
	dup203,
	dup204,
]);

var dup331 = linear_select([
	dup206,
	dup207,
]);

var dup332 = linear_select([
	dup209,
	dup210,
]);

var dup333 = linear_select([
	dup211,
	dup212,
]);

var dup334 = linear_select([
	dup216,
	dup91,
]);

var dup335 = linear_select([
	dup249,
	dup226,
]);

var dup336 = linear_select([
	dup252,
	dup207,
]);

var dup337 = linear_select([
	dup262,
	dup261,
]);

var dup338 = linear_select([
	dup264,
	dup265,
]);

var dup339 = linear_select([
	dup266,
	dup191,
	dup267,
	dup176,
	dup91,
]);

var dup340 = linear_select([
	dup275,
	dup276,
]);

var dup341 = linear_select([
	dup281,
	dup282,
]);

var dup342 = match("MESSAGE#524:1281", "nwparser.payload", "%{id}^^%{event_description}", processor_chain([
	dup53,
	dup15,
]));

var dup343 = match("MESSAGE#546:4868", "nwparser.payload", "%{id}^^%{event_description}", processor_chain([
	dup43,
	dup15,
]));

var dup344 = match("MESSAGE#549:302449153", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup43,
	dup15,
	dup287,
]));

var dup345 = match("MESSAGE#550:302449153:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup43,
	dup15,
	dup287,
]));

var dup346 = match("MESSAGE#553:302449155", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup74,
	dup15,
	dup287,
]));

var dup347 = match("MESSAGE#554:302449155:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup74,
	dup15,
	dup287,
]));

var dup348 = match("MESSAGE#585:302450432", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup168,
	dup15,
	dup287,
]));

var dup349 = match("MESSAGE#586:302450432:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup168,
	dup15,
	dup287,
]));

var dup350 = linear_select([
	dup290,
	dup291,
	dup292,
]);

var dup351 = lookup({
	dest: "nwparser.ec_activity",
	map: map_Activity,
	key: dup296,
});

var dup352 = lookup({
	dest: "nwparser.protocol",
	map: map_Protocol,
	key: dup299,
});

var dup353 = lookup({
	dest: "nwparser.protocol",
	map: map_Protocol,
	key: dup49,
});

var dup354 = lookup({
	dest: "nwparser.direction",
	map: map_Direction,
	key: dup299,
});

var dup355 = lookup({
	dest: "nwparser.action",
	map: map_Action,
	key: dup306,
});

var dup356 = lookup({
	dest: "nwparser.ec_activity",
	map: map_Activity,
	key: dup306,
});

var dup357 = match("MESSAGE#664:206", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var dup358 = match("MESSAGE#665:206:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var dup359 = match("MESSAGE#669:210", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup43,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var dup360 = match("MESSAGE#676:501", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{username}^^%{sdomain}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}^^%{fld31}^^%{filename_size}^^%{fld32}^^%{fld33}", processor_chain([
	dup43,
	dup15,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));

var dup361 = match("MESSAGE#677:501:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{username}^^%{sdomain}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}", processor_chain([
	dup43,
	dup15,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));

var hdr1 = match("HEADER#0:0001/0", "message", "%SYMANTECAV %{p0}");

var part1 = match("HEADER#0:0001/1_0", "nwparser.p0", "Delete %{p0}");

var part2 = match("HEADER#0:0001/1_1", "nwparser.p0", "Leave Alone %{p0}");

var part3 = match("HEADER#0:0001/1_2", "nwparser.p0", "Quarantine %{p0}");

var part4 = match("HEADER#0:0001/1_3", "nwparser.p0", "Undefined %{p0}");

var select1 = linear_select([
	part1,
	part2,
	part3,
	part4,
]);

var part5 = match("HEADER#0:0001/2", "nwparser.p0", "%{}..Alert: %{messageid->} %{data}..%{p0}", processor_chain([
	dup1,
]));

var all1 = all_match({
	processors: [
		hdr1,
		select1,
		part5,
	],
	on_success: processor_chain([
		setc("header_id","0001"),
	]),
});

var hdr2 = match("HEADER#1:0002", "message", "%SYMANTECAV Alert: %{messageid->} %{data}..%{p0}", processor_chain([
	setc("header_id","0002"),
	dup1,
]));

var hdr3 = match("HEADER#2:0003", "message", "%SYMANTECAV ..%{messageid->} %{data}..%{p0}", processor_chain([
	setc("header_id","0003"),
	dup1,
]));

var hdr4 = match("HEADER#3:0004", "message", "%SYMANTECAV %{hfld1->} ..%{messageid->} %{hfld2}.. %{p0}", processor_chain([
	setc("header_id","0004"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("hfld2"),
			constant(".. "),
			field("p0"),
		],
	}),
]));

var hdr5 = match("HEADER#4:0005", "message", "%SYMANTECAV %{hfld1->} %{messageid->} Found %{p0}", processor_chain([
	setc("header_id","0005"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" Found "),
			field("p0"),
		],
	}),
]));

var hdr6 = match("HEADER#5:0006", "message", "%SYMANTECAV %{messageid->} %{hfld1}..%{p0}", processor_chain([
	setc("header_id","0006"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(" "),
			field("hfld1"),
			constant(".."),
			field("p0"),
		],
	}),
]));

var hdr7 = match("HEADER#6:00081", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},Admin: %{husername},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00081"),
	dup2,
]));

var hdr8 = match("HEADER#7:0008", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},Admin: %{husername},%{messageid->} %{p0}", processor_chain([
	setc("header_id","0008"),
	dup2,
]));

var hdr9 = match("HEADER#8:00091", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","00091"),
	dup2,
]));

var hdr10 = match("HEADER#9:0009", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","0009"),
	dup2,
]));

var hdr11 = match("HEADER#10:00421", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},Admin: %{husername},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","00421"),
	dup2,
]));

var hdr12 = match("HEADER#11:0042", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},Admin: %{husername},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","0042"),
	dup2,
]));

var hdr13 = match("HEADER#12:99991", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},%{messageid->} %{p0}", processor_chain([
	setc("header_id","99991"),
	dup2,
]));

var hdr14 = match("HEADER#13:9999", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},Domain: %{hdomain},%{messageid->} %{p0}", processor_chain([
	setc("header_id","9999"),
	dup2,
]));

var hdr15 = match("HEADER#14:00101", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","00101"),
	dup2,
]));

var hdr16 = match("HEADER#15:0010", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","0010"),
	dup2,
]));

var hdr17 = match("HEADER#16:00111", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},%{messageid}.%{fld2->} %{p0}", processor_chain([
	setc("header_id","00111"),
	dup3,
]));

var hdr18 = match("HEADER#17:0011", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},%{messageid}.%{fld2->} %{p0}", processor_chain([
	setc("header_id","0011"),
	dup3,
]));

var hdr19 = match("HEADER#18:00121", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00121"),
	dup2,
]));

var hdr20 = match("HEADER#19:0012", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},%{messageid->} %{p0}", processor_chain([
	setc("header_id","0012"),
	dup2,
]));

var hdr21 = match("HEADER#20:11111", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},%{fld20->} %{fld21->} %{fld23->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","11111"),
	dup2,
]));

var hdr22 = match("HEADER#21:1111", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},%{fld20->} %{fld21->} %{fld23->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","1111"),
	dup2,
]));

var hdr23 = match("HEADER#22:13131", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},Category: %{hdata},%{hfld1},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","13131"),
	dup2,
]));

var hdr24 = match("HEADER#23:1313", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},Category: %{hdata},%{hfld1},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","1313"),
	dup2,
]));

var hdr25 = match("HEADER#24:00131", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},Category: %{hdata},%{hfld1},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00131"),
	dup2,
]));

var hdr26 = match("HEADER#25:0013", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},Category: %{hdata},%{hfld1},%{messageid->} %{p0}", processor_chain([
	setc("header_id","0013"),
	dup2,
]));

var hdr27 = match("HEADER#26:13142", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},SHA-256:%{checksum},MD-5:%{checksum},\"[SID: %{hfld1}] %{messageid->} %{p0}", processor_chain([
	setc("header_id","13142"),
	dup2,
]));

var hdr28 = match("HEADER#27:13141", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},\"[SID: %{hfld1}] %{messageid->} %{p0}", processor_chain([
	setc("header_id","13141"),
	dup2,
]));

var hdr29 = match("HEADER#28:1314", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},\"[SID: %{hfld1}] %{messageid->} %{p0}", processor_chain([
	setc("header_id","1314"),
	dup2,
]));

var hdr30 = match("HEADER#29:00141", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},[SID: %{hdata}] %{hfld1}. Traffic has been %{messageid->} %{p0}", processor_chain([
	setc("header_id","00141"),
	dup4,
]));

var hdr31 = match("HEADER#30:0014", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},[SID: %{hdata}] %{hfld1}. Traffic has been %{messageid->} %{p0}", processor_chain([
	setc("header_id","0014"),
	dup4,
]));

var hdr32 = match("HEADER#31:00161", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{messageid->} %{p0}", processor_chain([
	setc("header_id","00161"),
	dup2,
]));

var hdr33 = match("HEADER#32:0016", "message", "%{htime->} SymantecServer %{hhost}: %{messageid->} %{p0}", processor_chain([
	setc("header_id","0016"),
	dup2,
]));

var hdr34 = match("HEADER#33:29292", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},SHA-256:%{checksum},MD-5:%{checksum},%{fld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","29292"),
	dup5,
]));

var hdr35 = match("HEADER#34:29291", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{fld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","29291"),
	dup5,
]));

var hdr36 = match("HEADER#35:2929", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},%{fld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","2929"),
	dup5,
]));

var hdr37 = match("HEADER#36:00291", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{fld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","00291"),
	dup5,
]));

var hdr38 = match("HEADER#37:0029", "message", "%{htime->} SymantecServer %{hhost}: %{fld1->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0029"),
	dup5,
]));

var hdr39 = match("HEADER#38:00173", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhostip->} %{hhost->} SymantecServer: %{hshost},SHA-256:%{checksum},MD-5:%{checksum},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00173"),
	dup2,
]));

var hdr40 = match("HEADER#39:00172", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},SHA-256:%{checksum},MD-5:%{checksum},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00172"),
	dup2,
]));

var hdr41 = match("HEADER#40:00171", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00171"),
	dup2,
]));

var hdr42 = match("HEADER#41:0017", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},%{messageid->} %{p0}", processor_chain([
	setc("header_id","0017"),
	dup2,
]));

var hdr43 = match("HEADER#42:00151", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{hname},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00151"),
	dup6,
]));

var hdr44 = match("HEADER#43:0015", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},%{hname},%{messageid->} %{p0}", processor_chain([
	setc("header_id","0015"),
	dup6,
]));

var hdr45 = match("HEADER#44:0018", "message", "%SYMANTECAV Actual Name: %{hfld1->} ..Alert: %{messageid->} %{data}..%{p0}", processor_chain([
	setc("header_id","0018"),
	dup1,
]));

var hdr46 = match("HEADER#45:0021", "message", "%SYMANTECAV %{hfld1->} %{hfld2->} %{messageid->} %{hfld3->} %{p0}", processor_chain([
	setc("header_id","0021"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("hfld3"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr47 = match("HEADER#46:0022", "message", "%SYMANTECAV %{hfld1->} %{hfld2->} %{hfld3->} %{messageid->} %{hfld4->} %{p0}", processor_chain([
	setc("header_id","0022"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant(" "),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("hfld4"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr48 = match("HEADER#47:00191", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},Category: %{hdata},%{hfld1},%{fld40->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","00191"),
	dup7,
]));

var hdr49 = match("HEADER#48:0019", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},Category: %{hdata},%{hfld1},%{fld40->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0019"),
	dup7,
]));

var hdr50 = match("HEADER#49:00201", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hurl},Server: %{hhostid},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","00201"),
	dup2,
]));

var hdr51 = match("HEADER#50:0020", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hurl},Server: %{hhostid},The %{messageid->} %{p0}", processor_chain([
	setc("header_id","0020"),
	dup2,
]));

var hdr52 = match("HEADER#51:00231", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","00231"),
	dup2,
]));

var hdr53 = match("HEADER#52:0023", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},\"%{messageid->} %{p0}", processor_chain([
	setc("header_id","0023"),
	dup2,
]));

var hdr54 = match("HEADER#53:00241", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{messageid},%{payload}", processor_chain([
	setc("header_id","00241"),
]));

var hdr55 = match("HEADER#54:0024", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},%{messageid},%{payload}", processor_chain([
	setc("header_id","0024"),
]));

var hdr56 = match("HEADER#55:00261", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},\"%{haction->} \"\"\"\"%{messageid->} of Death\"\" %{payload}", processor_chain([
	setc("header_id","00261"),
]));

var hdr57 = match("HEADER#56:0026", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},\"%{haction->} \"\"\"\"%{messageid->} of Death\"\" %{payload}", processor_chain([
	setc("header_id","0026"),
]));

var hdr58 = match("HEADER#57:00371", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},\"%{haction->} \"\"%{messageid->} of Death\"\" %{payload}", processor_chain([
	setc("header_id","00371"),
]));

var hdr59 = match("HEADER#58:0037", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},\"%{haction->} \"\"%{messageid->} of Death\"\" %{payload}", processor_chain([
	setc("header_id","0037"),
]));

var hdr60 = match("HEADER#59:00271", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: Site: %{hsite},%{messageid}: %{p0}", processor_chain([
	setc("header_id","00271"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hhost"),
			constant(" SymantecServer: Site: "),
			field("hsite"),
			constant(","),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr61 = match("HEADER#60:0027", "message", "%{htime->} SymantecServer %{hhost}: Site: %{hsite},%{messageid}: %{p0}", processor_chain([
	setc("header_id","0027"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hhost"),
			constant(": Site: "),
			field("hsite"),
			constant(","),
			field("messageid"),
			constant(": "),
			field("p0"),
		],
	}),
]));

var hdr62 = match("HEADER#61:00301", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{messageid}: %{payload}", processor_chain([
	setc("header_id","00301"),
]));

var hdr63 = match("HEADER#62:0030", "message", "%{htime->} SymantecServer %{hhost}: %{hshost},%{messageid}: %{payload}", processor_chain([
	setc("header_id","0030"),
]));

var hdr64 = match("HEADER#63:00242", "message", "%{hmonth->} %{hday->} %{hhour}:%{hmin}:%{hsec->} %{hhost->} SymantecServer: %{hshost},%{hsaddr},%{messageid},%{payload}", processor_chain([
	setc("header_id","00242"),
]));

var hdr65 = match("HEADER#64:00243", "message", "%{htime->} %{hhost->} SymantecServer: %{hshost},%{hsaddr},%{hfld1},%{messageid->} %{p0}", processor_chain([
	setc("header_id","00243"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld1"),
			constant(","),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr66 = match("HEADER#65:00244", "message", "%{htime->} %{hhost->} SymantecServer: %{hshost},%{hsaddr},%{messageid},%{payload}", processor_chain([
	setc("header_id","00244"),
]));

var hdr67 = match("HEADER#66:0031", "message", "%SymantecEP: %{messageid}^^%{hhost}^^%{p0}", processor_chain([
	setc("header_id","0031"),
	dup8,
]));

var hdr68 = match("HEADER#67:0032", "message", "%SymantecEP-%{hevent}: %{hdomain}^^%{hlevel}^^%{fld1}^^%{messageid->} %{p0}", processor_chain([
	setc("header_id","0032"),
	dup2,
]));

var hdr69 = match("HEADER#68:0040", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{hfld5}^^%{hfld6}^^%{hfld7}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0040"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("hfld5"),
			constant("^^"),
			field("hfld6"),
			constant("^^"),
			field("hfld7"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr70 = match("HEADER#69:0033", "message", "%SymantecEP-%{hevent}: %{hdomain}^^%{hlevel}^^%{fld1}^^%{messageid}.%{fld2->} %{p0}", processor_chain([
	setc("header_id","0033"),
	dup3,
]));

var hdr71 = match("HEADER#70:0034", "message", "%SymantecEP-%{hevent}: %{hdomain}^^%{hlevel}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0034"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr72 = match("HEADER#71:0035", "message", "%SymantecEP-%{hfld1}: %{messageid}^^%{hhost}^^%{p0}", processor_chain([
	setc("header_id","0035"),
	dup8,
]));

var hdr73 = match("HEADER#72:0038", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{hfld5}^^%{hfld6}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0038"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("hfld5"),
			constant("^^"),
			field("hfld6"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr74 = match("HEADER#73:0041", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0041"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr75 = match("HEADER#74:0043", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{hfld7}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0043"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("hfld7"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr76 = match("HEADER#75:0039", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{hfld5}^^%{hfld6}^^%{hfld7}^^%{hfld8}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0039"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("hfld5"),
			constant("^^"),
			field("hfld6"),
			constant("^^"),
			field("hfld7"),
			constant("^^"),
			field("hfld8"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr77 = match("HEADER#76:0044", "message", "%SymantecEP-%{hfld1}: %{hfld2}^^%{hfld3}^^%{hfld4}^^%{hfld7}^^%{hfld8}^^%{messageid}^^%{p0}", processor_chain([
	setc("header_id","0044"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld2"),
			constant("^^"),
			field("hfld3"),
			constant("^^"),
			field("hfld4"),
			constant("^^"),
			field("hfld7"),
			constant("^^"),
			field("hfld8"),
			constant("^^"),
			field("messageid"),
			constant("^^"),
			field("p0"),
		],
	}),
]));

var hdr78 = match("HEADER#77:0045", "message", "%NICWIN-4-%{msgIdPart1}_%{msgIdPart2}_Symantec: %{payload}", processor_chain([
	setc("header_id","0045"),
	call({
		dest: "nwparser.messageid",
		fn: STRCAT,
		args: [
			field("msgIdPart1"),
			constant("_"),
			field("msgIdPart2"),
		],
	}),
]));

var hdr79 = match("HEADER#78:0046", "message", "%NICWIN-4-%{messageid}_%{hfld2}_Symantec AntiVirus: %{payload}", processor_chain([
	setc("header_id","0046"),
]));

var select2 = linear_select([
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
	hdr18,
	hdr19,
	hdr20,
	hdr21,
	hdr22,
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
	hdr37,
	hdr38,
	hdr39,
	hdr40,
	hdr41,
	hdr42,
	hdr43,
	hdr44,
	hdr45,
	hdr46,
	hdr47,
	hdr48,
	hdr49,
	hdr50,
	hdr51,
	hdr52,
	hdr53,
	hdr54,
	hdr55,
	hdr56,
	hdr57,
	hdr58,
	hdr59,
	hdr60,
	hdr61,
	hdr62,
	hdr63,
	hdr64,
	hdr65,
	hdr66,
	hdr67,
	hdr68,
	hdr69,
	hdr70,
	hdr71,
	hdr72,
	hdr73,
	hdr74,
	hdr75,
	hdr76,
	hdr77,
	hdr78,
	hdr79,
]);

var part6 = match("MESSAGE#0:Active/0", "nwparser.payload", "Active Response that started at %{fld1->} is disengaged. The traffic from IP address %{hostip->} was blocked for %{fld2->} second(s).,Local: %{saddr},Local: %{fld7},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},%{protocol},%{direction},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username}, Domain: %{p0}");

var all2 = all_match({
	processors: [
		part6,
		dup311,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup16,
		dup17,
		dup18,
		dup19,
	]),
});

var msg1 = msg("Active", all2);

var part7 = match("MESSAGE#1:Active:01/0", "nwparser.payload", "Active Response that started at %{fld1->} is disengaged. The traffic from IP address %{hostip->} was blocked for %{duration->} second(s). ,Local: %{saddr},Local: %{fld7},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},%{protocol},%{direction},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username}, Domain: %{p0}");

var all3 = all_match({
	processors: [
		part7,
		dup311,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup15,
		dup16,
		dup17,
		dup18,
		dup19,
	]),
});

var msg2 = msg("Active:01", all3);

var select3 = linear_select([
	msg1,
	msg2,
]);

var part8 = match("MESSAGE#2:Administrator", "nwparser.payload", "Administrator logout%{}", processor_chain([
	setc("eventcategory","1401070000"),
	dup12,
	dup13,
	dup20,
	setc("ec_activity","Logoff"),
	dup21,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Administrator logout."),
]));

var msg3 = msg("Administrator", part8);

var part9 = match("MESSAGE#3:Administrator:01", "nwparser.payload", "Administrator%{space}log on failed", processor_chain([
	setc("eventcategory","1401030000"),
	dup12,
	dup13,
	dup20,
	dup24,
	dup21,
	dup25,
	dup14,
	dup15,
	dup23,
	setc("event_description","Administrator log on failed."),
]));

var msg4 = msg("Administrator:01", part9);

var part10 = match("MESSAGE#4:Administrator:02", "nwparser.payload", "Administrator%{space}log on succeeded", processor_chain([
	setc("eventcategory","1401060000"),
	dup12,
	dup13,
	dup20,
	dup24,
	dup21,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Administrator log on succeeded."),
]));

var msg5 = msg("Administrator:02", part10);

var select4 = linear_select([
	msg3,
	msg4,
	msg5,
]);

var part11 = match("MESSAGE#5:Administrator:03", "nwparser.payload", "password of System administrator '%{username}' has been changed.", processor_chain([
	dup26,
	dup12,
	dup13,
	dup20,
	dup27,
	dup28,
	dup22,
	dup14,
	dup15,
	dup23,
	dup29,
]));

var msg6 = msg("Administrator:03", part11);

var part12 = match("MESSAGE#290:password", "nwparser.payload", "password of administrator \"%{c_username}\" was changed", processor_chain([
	dup26,
	dup12,
	dup13,
	dup20,
	dup30,
	dup31,
	dup22,
	dup14,
	dup15,
	setc("event_description","Password of administrator changed."),
	dup23,
]));

var msg7 = msg("password", part12);

var part13 = match("MESSAGE#291:password:01", "nwparser.payload", "password of System administrator \"%{c_username}\" has been changed", processor_chain([
	dup26,
	dup12,
	dup13,
	dup20,
	dup30,
	dup31,
	dup22,
	dup14,
	dup15,
	dup29,
	dup23,
]));

var msg8 = msg("password:01", part13);

var select5 = linear_select([
	msg6,
	msg7,
	msg8,
]);

var part14 = match("MESSAGE#6:allowed", "nwparser.payload", "%{fld6->} detected. Traffic has been allowed from this application: %{fld1},Local: %{daddr},Local: %{fld7},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID:%{fld23},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain}", processor_chain([
	dup32,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
	dup33,
	dup19,
	dup34,
]));

var msg9 = msg("allowed", part14);

var part15 = match("MESSAGE#7:allowed:11", "nwparser.payload", "%{fld6->} detected. Traffic has been allowed from this application: %{fld1},Local: %{saddr},Local: %{fld7},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID:%{fld23},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain}", processor_chain([
	dup32,
	dup12,
	dup13,
	dup14,
	dup15,
	dup16,
	dup17,
	dup33,
	dup19,
	dup35,
]));

var msg10 = msg("allowed:11", part15);

var select6 = linear_select([
	msg9,
	msg10,
]);

var part16 = match("MESSAGE#8:Malicious", "nwparser.payload", "Malicious Site: Malicious Web Site, Domain, or URL (%{fld11}) attack blocked. Traffic has been blocked for this application: %{fld12}\",Local: %{daddr},Local: %{fld7},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID:%{fld23},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld39},User: %{username},Domain: %{domain},!ExternalLoggingTask.localport! %{dport},!ExternalLoggingTask.remoteport! %{sport},!ExternalLoggingTask.cidssignid! %{sigid},\"!ExternalLoggingTask.strcidssignid! %{sigid_string}\",!ExternalLoggingTask.cidssignsubid! %{sigid1},!ExternalLoggingTask.intrusionurl! %{url},!ExternalLoggingTask.intrusionpayloadurl! %{fld33}", processor_chain([
	dup36,
	dup12,
	dup13,
	dup37,
	dup38,
	dup14,
	dup15,
	dup16,
	dup17,
	dup39,
	dup19,
	dup34,
]));

var msg11 = msg("Malicious", part16);

var part17 = match("MESSAGE#9:Malicious:01", "nwparser.payload", "Malicious Site: Malicious Web Site, Domain, or URL (%{fld11}) attack blocked. Traffic has been blocked for this application: %{fld12}\",Local: %{saddr},Local: %{fld7},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID:%{fld23},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld39},User: %{username},Domain: %{domain},!ExternalLoggingTask.localport! %{sport},!ExternalLoggingTask.remoteport! %{dport},!ExternalLoggingTask.cidssignid! %{sigid},\"!ExternalLoggingTask.strcidssignid! %{sigid_string}\",!ExternalLoggingTask.cidssignsubid! %{sigid1},!ExternalLoggingTask.intrusionurl! %{url},!ExternalLoggingTask.intrusionpayloadurl! %{fld33}", processor_chain([
	dup36,
	dup12,
	dup13,
	dup37,
	dup38,
	dup14,
	dup15,
	dup16,
	dup17,
	dup39,
	dup19,
	dup35,
]));

var msg12 = msg("Malicious:01", part17);

var part18 = match("MESSAGE#10:Malicious:02/0", "nwparser.payload", "Malicious Site: Malicious Web Site, Domain, or URL (%{fld11}) attack blocked. Traffic has been blocked for this application: %{fld12}\",Local: %{saddr},Local: %{fld7},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Inbound,%{p0}");

var part19 = match("MESSAGE#10:Malicious:02/1_0", "nwparser.p0", "%{protocol},Intrusion ID:%{fld23},Begin: %{p0}");

var part20 = match("MESSAGE#10:Malicious:02/1_1", "nwparser.p0", "%{protocol},Begin: %{p0}");

var select7 = linear_select([
	part19,
	part20,
]);

var part21 = match("MESSAGE#10:Malicious:02/2", "nwparser.p0", "%{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld39},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},\"CIDS Signature string: %{sigid_string}\",CIDS Signature SubID: %{fld29},Intrusion URL:%{fld24},Intrusion Payload URL:%{fld25}");

var all4 = all_match({
	processors: [
		part18,
		select7,
		part21,
	],
	on_success: processor_chain([
		dup36,
		dup12,
		dup13,
		dup40,
		dup41,
		dup42,
		dup15,
		dup19,
		dup34,
		setc("event_description","Malicious Site: Malicious Web Site, Domain, or URL attcak blocked"),
	]),
});

var msg13 = msg("Malicious:02", all4);

var select8 = linear_select([
	msg11,
	msg12,
	msg13,
]);

var part22 = match("MESSAGE#11:Antivirus", "nwparser.payload", "%{product->} definitions %{info->} failed to update.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup44,
	dup45,
	dup30,
	dup25,
	dup14,
	dup15,
	setc("event_description","Product definition failed to update."),
]));

var msg14 = msg("Antivirus", part22);

var part23 = match("MESSAGE#12:Antivirus:01", "nwparser.payload", "%{product->} definitions %{info->} is up-to-date.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Product definitions are up-to-date."),
]));

var msg15 = msg("Antivirus:01", part23);

var part24 = match("MESSAGE#13:Antivirus:02", "nwparser.payload", "%{product->} definitions %{info->} was successfully updated.", processor_chain([
	dup43,
	dup44,
	dup45,
	dup30,
	dup22,
	dup15,
	setc("event_description","Product definitions was successfully updated."),
]));

var msg16 = msg("Antivirus:02", part24);

var select9 = linear_select([
	msg14,
	msg15,
	msg16,
]);

var part25 = match("MESSAGE#14:Somebody/0", "nwparser.payload", "%{event_description}\",Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},1,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username}, Domain: %{p0}");

var all5 = all_match({
	processors: [
		part25,
		dup311,
	],
	on_success: processor_chain([
		dup46,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup47,
		dup48,
		dup312,
		dup14,
	]),
});

var msg17 = msg("Somebody", all5);

var part26 = match("MESSAGE#15:Somebody:01/0", "nwparser.payload", "%{event_description}\",Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},0,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username}, Domain: %{p0}");

var all6 = all_match({
	processors: [
		part26,
		dup313,
	],
	on_success: processor_chain([
		dup46,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup47,
		dup51,
		dup312,
		dup14,
	]),
});

var msg18 = msg("Somebody:01", all6);

var part27 = match("MESSAGE#16:Somebody:02/0", "nwparser.payload", "%{event_description}\",Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},2,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username}, Domain: %{p0}");

var all7 = all_match({
	processors: [
		part27,
		dup313,
	],
	on_success: processor_chain([
		dup46,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup47,
		dup52,
		dup312,
		dup14,
	]),
});

var msg19 = msg("Somebody:02", all7);

var select10 = linear_select([
	msg17,
	msg18,
	msg19,
]);

var part28 = match("MESSAGE#17:Application/0", "nwparser.payload", "%{fld44},Application and Device Control is ready,%{fld8},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{fld4},%{fld5},%{fld6},%{fld7},User: %{username},Domain: %{p0}");

var part29 = match("MESSAGE#17:Application/1_0", "nwparser.p0", "%{domain},Action Type:%{fld46},File size (%{fld10}): %{filename_size},Device ID: %{device}");

var select11 = linear_select([
	part29,
	dup10,
]);

var all8 = all_match({
	processors: [
		part28,
		select11,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup41,
		dup42,
		dup15,
		dup54,
	]),
});

var msg20 = msg("Application", all8);

var part30 = match("MESSAGE#18:Application:01", "nwparser.payload", "%{fld44},Application and Device Control engine is not verified,%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{fld4},%{fld5},%{fld6},%{fld7},User: %{username},Domain: %{domain}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	setc("event_description","Application and Device Control engine is not verified."),
]));

var msg21 = msg("Application:01", part30);

var part31 = match("MESSAGE#19:Application:02", "nwparser.payload", "%{fld44}Blocked,[%{fld5}] %{event_description->} - Caller MD5=%{fld6},Create Process,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld45}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup37,
	dup14,
	dup41,
	dup42,
	dup15,
]));

var msg22 = msg("Application:02", part31);

var part32 = match("MESSAGE#683:Application:03", "nwparser.payload", "Application,rn=%{fld1->} cid=%{fld2->} eid=%{fld3},%{fld4->} %{fld5},%{fld6},Symantec AntiVirus,%{hostname},Classic,%{shost},%{event_description},, Scan Complete: Risks: %{fld7->} Scanned: %{fld8->} Omitted: %{fld9->} Trusted Files Skipped: %{fld10}", processor_chain([
	dup43,
	dup15,
	dup55,
]));

var msg23 = msg("Application:03", part32);

var part33 = match("MESSAGE#684:Application:04", "nwparser.payload", "Application,rn=%{fld1->} cid=%{fld2->} eid=%{fld3},%{fld4->} %{fld5},%{fld6},Symantec AntiVirus,%{hostname},Classic,%{shost},%{event_description},, %{info}.", processor_chain([
	dup43,
	dup15,
	dup55,
]));

var msg24 = msg("Application:04", part33);

var part34 = match("MESSAGE#685:Application:05", "nwparser.payload", "Application,rn=%{fld1->} cid=%{fld2->} eid=%{fld3},%{fld4->} %{fld5},%{fld6},Symantec AntiVirus,%{hostname},Classic,%{shost},%{fld22},,%{space}Proactive Threat Protection has been disabled", processor_chain([
	dup43,
	dup56,
	dup15,
	dup57,
	dup55,
]));

var msg25 = msg("Application:05", part34);

var select12 = linear_select([
	msg20,
	msg21,
	msg22,
	msg23,
	msg24,
	msg25,
]);

var part35 = match("MESSAGE#20:Application:07", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},\"Application has changed since the last time you opened it, process id:%{process_id->} Filename: %{fld8->} The change was denied by user.\",Local: %{daddr},Local: %{fld12},Remote: %{fld15},Remote: %{saddr},Remote: %{fld11},Inbound,%{protocol},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup53,
	dup34,
	dup58,
	dup12,
	dup13,
	dup41,
	dup42,
	dup15,
	dup54,
	dup47,
]));

var msg26 = msg("Application:07", part35);

var part36 = match("MESSAGE#27:Application:06/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},\"Application has changed since the last time you opened it, process id: %{process_id->} Filename: %{filename->} %{fld1}\",Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all9 = all_match({
	processors: [
		part36,
		dup314,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup35,
		dup58,
	]),
});

var msg27 = msg("Application:06", all9);

var part37 = match("MESSAGE#28:REMEDIATION/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},REMEDIATION WAS NEEDED - %{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Unknown,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all10 = all_match({
	processors: [
		part37,
		dup314,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
	]),
});

var msg28 = msg("REMEDIATION", all10);

var part38 = match("MESSAGE#29:blocked:06/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all11 = all_match({
	processors: [
		part38,
		dup314,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup34,
	]),
});

var msg29 = msg("blocked:06", all11);

var part39 = match("MESSAGE#30:blocked:16/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all12 = all_match({
	processors: [
		part39,
		dup314,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup35,
		dup40,
	]),
});

var msg30 = msg("blocked:16", all12);

var part40 = match("MESSAGE#31:scanning:01/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum->} ,\"Somebody is scanning your computer. Your computer's TCP ports: %{fld60}, %{fld61}, %{fld62}, %{fld63->} and %{fld64->} have been scanned from %{fld65}.\",Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all13 = all_match({
	processors: [
		part40,
		dup315,
	],
	on_success: processor_chain([
		dup65,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup66,
		dup35,
	]),
});

var msg31 = msg("scanning:01", all13);

var part41 = match("MESSAGE#32:scanning/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum->} ,\"Somebody is scanning your computer. Your computer's TCP ports: %{fld60}, %{fld61}, %{fld62}, %{fld63->} and %{fld64->} have been scanned from %{fld65}.\",Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all14 = all_match({
	processors: [
		part41,
		dup315,
	],
	on_success: processor_chain([
		dup65,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup66,
		dup34,
	]),
});

var msg32 = msg("scanning", all14);

var part42 = match("MESSAGE#33:Informational/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum->} ,Informational: File Download Hash,Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},%{p0}");

var part43 = match("MESSAGE#33:Informational/1_0", "nwparser.p0", " Domain: %{p0}");

var select13 = linear_select([
	part43,
	dup67,
]);

var part44 = match("MESSAGE#33:Informational/2", "nwparser.p0", "%{} %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all15 = all_match({
	processors: [
		part42,
		select13,
		part44,
		dup315,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup68,
		dup34,
	]),
});

var msg33 = msg("Informational", all15);

var part45 = match("MESSAGE#34:Informational:01/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum->} ,Informational: File Download Hash,Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all16 = all_match({
	processors: [
		part45,
		dup315,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup68,
		dup35,
	]),
});

var msg34 = msg("Informational:01", all16);

var part46 = match("MESSAGE#35:SHA-256::01", "nwparser.payload", "%{shost}, SHA-256:%{checksum},MD-5:%{checksum},CCD Notification: REMEDIATION NOT REQUIRED,Local: %{saddr},Local: %{fld1},Remote:%{fld2},Remote: %{daddr},Remote: %{fld3},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application:%{fld6},Location: %{fld7},User: %{username}, Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string:%{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{url},Intrusion Payload URL:", processor_chain([
	dup53,
	dup12,
	dup13,
	dup40,
	dup41,
	dup42,
	dup15,
	dup19,
	setc("event_description","CCD Notification: REMEDIATION NOT REQUIRED"),
	setc("direction","Unknown"),
]));

var msg35 = msg("SHA-256::01", part46);

var part47 = match("MESSAGE#36:Web_Attack/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum->} ,Web Attack : Malvertisement Website Redirect %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all17 = all_match({
	processors: [
		part47,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup70,
		dup34,
	]),
});

var msg36 = msg("Web_Attack", all17);

var part48 = match("MESSAGE#37:Web_Attack:13/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Web Attack: Fake Flash Player Download %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all18 = all_match({
	processors: [
		part48,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		setc("event_description","Web Attack : Fake Flash Player Download"),
		dup34,
	]),
});

var msg37 = msg("Web_Attack:13", all18);

var part49 = match("MESSAGE#38:Web_Attack:16/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] Web Attack%{p0}");

var part50 = match("MESSAGE#38:Web_Attack:16/1_0", "nwparser.p0", " : %{p0}");

var select14 = linear_select([
	part50,
	dup71,
]);

var part51 = match("MESSAGE#38:Web_Attack:16/2", "nwparser.p0", "%{}JSCoinminer Download %{fld21->} attack blocked. Traffic has been blocked for this application: %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,OTHERS,,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},\"Intrusion URL: %{url}\",Intrusion Payload URL:%{fld25}");

var all19 = all_match({
	processors: [
		part49,
		select14,
		part51,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		setc("event_description","JSCoinminer Download attack blocked."),
		dup34,
	]),
});

var msg38 = msg("Web_Attack:16", all19);

var part52 = match("MESSAGE#39:Web_Attack:03", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum->} ,[SID: %{fld26}] Web Attack: Apache Struts2 devMode OGNL Execution attack detected but not blocked. %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},\"Intrusion URL: %{url}\",Intrusion Payload URL:%{fld25}", processor_chain([
	dup69,
	dup12,
	dup13,
	dup40,
	dup16,
	dup17,
	dup15,
	dup19,
	setc("event_description","Web Attack: Apache Struts2 devMode OGNL Execution attack detected but not blocked."),
	dup35,
]));

var msg39 = msg("Web_Attack:03", part52);

var part53 = match("MESSAGE#40:Web_Attack:15", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] Web Attack : Malvertisement Website Redirect %{fld2->} attack blocked. Traffic has been blocked for this application: %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,OTHERS,,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},\"Intrusion URL: %{url}\",Intrusion Payload URL:%{fld25}", processor_chain([
	dup69,
	dup12,
	dup13,
	dup40,
	dup16,
	dup17,
	dup15,
	dup19,
	setc("event_description","Malvertisement Website Redirect "),
	dup34,
]));

var msg40 = msg("Web_Attack:15", part53);

var part54 = match("MESSAGE#41:Web_Attack:11/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum->} ,Web Attack : Malvertisement Website Redirect %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all20 = all_match({
	processors: [
		part54,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup70,
		dup35,
	]),
});

var msg41 = msg("Web_Attack:11", all20);

var part55 = match("MESSAGE#42:Web_Attack:01/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum->} ,Web Attack: Mass Injection Website %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all21 = all_match({
	processors: [
		part55,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup72,
		dup34,
	]),
});

var msg42 = msg("Web_Attack:01", all21);

var part56 = match("MESSAGE#43:Web_Attack:12/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum->} ,Web Attack: Mass Injection Website %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all22 = all_match({
	processors: [
		part56,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup72,
		dup35,
	]),
});

var msg43 = msg("Web_Attack:12", all22);

var part57 = match("MESSAGE#44:Web_Attack:14/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Web Attack: Mass Injection Website %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all23 = all_match({
	processors: [
		part57,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		setc("event_description","Web Attack : Mass Injection Website"),
		dup34,
	]),
});

var msg44 = msg("Web_Attack:14", all23);

var part58 = match("MESSAGE#45:Web_Attack:17/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Web Attack : Malvertisement Website Redirect %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all24 = all_match({
	processors: [
		part58,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		setc("event_description","Web Attack: Malvertisement Website Redirect."),
		dup34,
	]),
});

var msg45 = msg("Web_Attack:17", all24);

var part59 = match("MESSAGE#46:Web_Attack:18/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Web Attack: Fake Tech Support Website %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all25 = all_match({
	processors: [
		part59,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		setc("event_description","Web Attack: Fake Tech Support Website"),
		dup34,
	]),
});

var msg46 = msg("Web_Attack:18", all25);

var part60 = match("MESSAGE#47:App_Attack/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum},Fake App Attack: Misleading Application Website%{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all26 = all_match({
	processors: [
		part60,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup73,
		dup35,
	]),
});

var msg47 = msg("App_Attack", all26);

var part61 = match("MESSAGE#48:App_Attack:02/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Fake App Attack: Misleading Application Website%{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all27 = all_match({
	processors: [
		part61,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup73,
		dup35,
	]),
});

var msg48 = msg("App_Attack:02", all27);

var part62 = match("MESSAGE#49:App_Attack:01/0", "nwparser.payload", "%{fld3}, SHA-256:%{checksum},MD-5:%{checksum->} ,Fake App Attack: Misleading Application Website%{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all28 = all_match({
	processors: [
		part62,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup73,
		dup34,
	]),
});

var msg49 = msg("App_Attack:01", all28);

var part63 = match("MESSAGE#50:Host_Integrity/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},The most recent Host Integrity content has not completed a download or cannot be authenticated.,Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Unknown,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all29 = all_match({
	processors: [
		part63,
		dup315,
	],
	on_success: processor_chain([
		dup74,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup75,
	]),
});

var msg50 = msg("Host_Integrity", all29);

var part64 = match("MESSAGE#307:process:12/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},\"%{p0}");

var part65 = match("MESSAGE#307:process:12/2", "nwparser.p0", "%{event_description}, process id: %{process_id->} Filename: %{filename->} The change was allowed by profile%{fld6}\"%{p0}");

var all30 = all_match({
	processors: [
		part64,
		dup316,
		part65,
		dup316,
		dup78,
		dup317,
		dup81,
		dup315,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
		dup40,
	]),
});

var msg51 = msg("process:12", all30);

var part66 = match("MESSAGE#461:Audit:01/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description}attack detected but not blocked. Application path:%{fld27},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all31 = all_match({
	processors: [
		part66,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup82,
		dup19,
		dup34,
	]),
});

var msg52 = msg("Audit:01", all31);

var part67 = match("MESSAGE#462:Audit:11/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description}attack detected but not blocked. Application path:%{fld27},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all32 = all_match({
	processors: [
		part67,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup82,
		dup19,
		dup35,
		dup40,
	]),
});

var msg53 = msg("Audit:11", all32);

var part68 = match("MESSAGE#463:Audit:02/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description}. Traffic has been blocked for this application:%{fld27},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},%{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all33 = all_match({
	processors: [
		part68,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup83,
		dup19,
		dup34,
	]),
});

var msg54 = msg("Audit:02", all33);

var part69 = match("MESSAGE#464:Audit:12/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld26}] %{category}: %{event_description}. Traffic has been blocked for this application:%{fld27},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},%{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all34 = all_match({
	processors: [
		part69,
		dup315,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup83,
		dup19,
		dup35,
		dup40,
	]),
});

var msg55 = msg("Audit:12", all34);

var part70 = match("MESSAGE#507:Attack:03/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld111}] %{category}:%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all35 = all_match({
	processors: [
		part70,
		dup314,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
	]),
});

var msg56 = msg("Attack:03", all35);

var part71 = match("MESSAGE#508:Attack:02/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},[SID: %{fld111}] %{category}:%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all36 = all_match({
	processors: [
		part71,
		dup314,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup35,
	]),
});

var msg57 = msg("Attack:02", all36);

var part72 = match("MESSAGE#710:Auto-block/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Auto-Block Event,Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all37 = all_match({
	processors: [
		part72,
		dup314,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
	]),
});

var msg58 = msg("Auto-block", all37);

var part73 = match("MESSAGE#711:Denial/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Denial of Service 'Smurf' attack detected. Description: %{info},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all38 = all_match({
	processors: [
		part73,
		dup314,
	],
	on_success: processor_chain([
		dup84,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup35,
		dup85,
	]),
});

var msg59 = msg("Denial", all38);

var part74 = match("MESSAGE#712:Denial:01/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Denial of Service 'Smurf' attack detected. Description: %{info},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all39 = all_match({
	processors: [
		part74,
		dup314,
	],
	on_success: processor_chain([
		dup84,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
		dup85,
	]),
});

var msg60 = msg("Denial:01", all39);

var part75 = match("MESSAGE#713:Denial:02/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},'Denial of Service ''Smurf'' attack detected. Description: %{info}',Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all40 = all_match({
	processors: [
		part75,
		dup314,
	],
	on_success: processor_chain([
		dup84,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup35,
		dup85,
	]),
});

var msg61 = msg("Denial:02", all40);

var part76 = match("MESSAGE#714:Denial:03/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},'Denial of Service ''Smurf'' attack detected. Description: %{info}',Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all41 = all_match({
	processors: [
		part76,
		dup314,
	],
	on_success: processor_chain([
		dup84,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
		dup85,
	]),
});

var msg62 = msg("Denial:03", all41);

var part77 = match("MESSAGE#715:Host:18", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},'Host Integrity check passed%{space}Requirement: %{fld11->} passed ',Local: %{saddr},Local: %{fld3},Remote: %{fld41},Remote: %{daddr},Remote: %{fld55},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup86,
	dup87,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup88,
	dup19,
]));

var msg63 = msg("Host:18", part77);

var part78 = match("MESSAGE#716:Host:19", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},'Host Integrity check failed Requirement: ''%{fld11}'' passed Requirement: ''%{fld12}'' failed ',Local: %{saddr},Local: %{fld3},Remote: %{fld41},Remote: %{daddr},Remote: %{fld55},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup25,
	dup14,
	dup15,
	dup89,
	dup19,
]));

var msg64 = msg("Host:19", part78);

var part79 = match("MESSAGE#719:DLP_version", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},DLP version is latest,Local: %{saddr},Local: %{fld3},Remote: %{fld41},Remote: %{daddr},Remote: %{fld55},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup40,
	dup41,
	dup42,
	dup15,
	dup19,
	dup34,
	setc("event_description","DLP version is latest"),
]));

var msg65 = msg("DLP_version", part79);

var part80 = match("MESSAGE#720:Brute_force/0", "nwparser.payload", "SHA-256:%{checksum},MD-5:%{checksum},Brute force remote login,Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld27},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all42 = all_match({
	processors: [
		part80,
		dup314,
	],
	on_success: processor_chain([
		setc("eventcategory","1101010000"),
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
		setc("event_description","Brute force remote login"),
	]),
});

var msg66 = msg("Brute_force", all42);

var select15 = linear_select([
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
	msg62,
	msg63,
	msg64,
	msg65,
	msg66,
]);

var part81 = match("MESSAGE#21:Applied/0", "nwparser.payload", "Applied new policy with %{info}successfully.%{p0}");

var all43 = all_match({
	processors: [
		part81,
		dup318,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Applied new policy successfully."),
	]),
});

var msg67 = msg("Applied", all43);

var part82 = match("MESSAGE#700:Smc:04", "nwparser.payload", "Applied new profile with serial number %{fld23->} successfully.", processor_chain([
	dup53,
	dup94,
	dup13,
	dup14,
	dup15,
	setc("event_description","Applied new profile successfully."),
]));

var msg68 = msg("Smc:04", part82);

var select16 = linear_select([
	msg67,
	msg68,
]);

var part83 = match("MESSAGE#22:Add", "nwparser.payload", "Add shared policy upon system install,LiveUpdate Settings policy%{}", processor_chain([
	dup95,
	dup12,
	dup13,
	dup96,
	dup97,
	dup14,
	dup15,
	dup23,
	setc("event_description","Add shared policy upon system install,LiveUpdate Settings policy."),
]));

var msg69 = msg("Add", part83);

var part84 = match("MESSAGE#23:blocked:01/0", "nwparser.payload", "System Infected: %{threat_name->} detected. Traffic has been blocked from this application: %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld15},Remote: %{daddr},Remote: %{fld51},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part85 = match("MESSAGE#23:blocked:01/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}");

var all44 = all_match({
	processors: [
		part84,
		dup319,
		part85,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup35,
	]),
});

var msg70 = msg("blocked:01", all44);

var part86 = match("MESSAGE#24:blocked:12/0", "nwparser.payload", "System Infected: %{threat_name->} detected. Traffic has been blocked from this application: %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld15},Remote: %{saddr},Remote: %{fld51},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part87 = match("MESSAGE#24:blocked:12/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}");

var all45 = all_match({
	processors: [
		part86,
		dup319,
		part87,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup34,
	]),
});

var msg71 = msg("blocked:12", all45);

var part88 = match("MESSAGE#25:blocked:05/0", "nwparser.payload", "%{fld28->} detected. Traffic has been blocked from this application: %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{daddr},Remote: %{fld15},Remote: %{fld51},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part89 = match("MESSAGE#25:blocked:05/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var all46 = all_match({
	processors: [
		part88,
		dup319,
		part89,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup35,
	]),
});

var msg72 = msg("blocked:05", all46);

var part90 = match("MESSAGE#26:blocked:15/0", "nwparser.payload", "%{fld28->} detected. Traffic has been blocked from this application: %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{saddr},Remote: %{fld15},Remote: %{fld51},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part91 = match("MESSAGE#26:blocked:15/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var all47 = all_match({
	processors: [
		part90,
		dup319,
		part91,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup34,
	]),
});

var msg73 = msg("blocked:15", all47);

var part92 = match("MESSAGE#52:blocked/0", "nwparser.payload", "%{fld28->} detected. Traffic has been blocked from this application: %{fld1},Local: %{saddr},Local: %{fld12},Remote: %{fld15},Remote: %{daddr},Remote: %{fld51},Outbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var all48 = all_match({
	processors: [
		part92,
		dup319,
		dup100,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup35,
	]),
});

var msg74 = msg("blocked", all48);

var part93 = match("MESSAGE#53:blocked:11/0", "nwparser.payload", "%{fld28->} detected. Traffic has been blocked from this application: %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld15},Remote: %{saddr},Remote: %{fld51},Inbound,%{protocol},Intrusion ID: %{fld52},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var all49 = all_match({
	processors: [
		part93,
		dup319,
		dup100,
	],
	on_success: processor_chain([
		dup32,
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		dup62,
		dup19,
		dup34,
	]),
});

var msg75 = msg("blocked:11", all49);

var select17 = linear_select([
	msg70,
	msg71,
	msg72,
	msg73,
	msg74,
	msg75,
]);

var part94 = match("MESSAGE#51:Host_Integrity:01/0", "nwparser.payload", "The most recent Host Integrity content has not completed a download or cannot be authenticated.,Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Unknown,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all50 = all_match({
	processors: [
		part94,
		dup315,
	],
	on_success: processor_chain([
		dup74,
		dup12,
		dup13,
		dup40,
		dup16,
		dup17,
		dup15,
		dup19,
		dup75,
	]),
});

var msg76 = msg("Host_Integrity:01", all50);

var part95 = match("MESSAGE#190:Local::01/1", "nwparser.p0", "%{} %{daddr},Local: %{dport},Local: %{fld12},Remote: %{saddr},Remote: %{fld13},Remote: %{sport},Remote: %{fld15},%{protocol},Inbound,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Rule: %{rulename},Location: %{fld11},User: %{username},Domain: %{domain},Action: %{action}");

var all51 = all_match({
	processors: [
		dup320,
		part95,
	],
	on_success: processor_chain([
		dup86,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup103,
		dup104,
		dup34,
	]),
});

var msg77 = msg("Local::01", all51);

var part96 = match("MESSAGE#191:Local::13/1", "nwparser.p0", "%{} %{saddr},Local: %{sport},Local: %{fld12},Remote: %{daddr},Remote: %{fld13},Remote: %{dport},Remote: %{fld15},%{protocol},Outbound,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Rule: %{rulename},Location: %{fld11},User: %{username},Domain: %{domain},Action: %{action}");

var all52 = all_match({
	processors: [
		dup320,
		part96,
	],
	on_success: processor_chain([
		dup86,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup103,
		dup104,
		dup35,
	]),
});

var msg78 = msg("Local::13", all52);

var part97 = match("MESSAGE#192:Local:/0", "nwparser.payload", "Local: %{saddr},Local: %{sport},Local: %{fld12},Remote: %{daddr},Remote: %{fld13},Remote: %{dport},Remote: %{fld15},%{protocol},Outbound,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},%{p0}");

var all53 = all_match({
	processors: [
		part97,
		dup321,
		dup107,
		dup322,
	],
	on_success: processor_chain([
		dup86,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup103,
		dup104,
		dup35,
	]),
});

var msg79 = msg("Local:", all53);

var part98 = match("MESSAGE#193:Local:11/0", "nwparser.payload", "Local: %{daddr},Local: %{dport},Local: %{fld12},Remote: %{saddr},Remote: %{fld13},Remote: %{sport},Remote: %{fld15},%{protocol},Inbound,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},%{p0}");

var all54 = all_match({
	processors: [
		part98,
		dup321,
		dup107,
		dup322,
	],
	on_success: processor_chain([
		dup86,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup103,
		dup104,
		dup34,
	]),
});

var msg80 = msg("Local:11", all54);

var part99 = match("MESSAGE#194:Local::09", "nwparser.payload", "[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string->} CVE-%{cve},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup110,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup111,
	dup34,
]));

var msg81 = msg("Local::09", part99);

var part100 = match("MESSAGE#195:Local::20", "nwparser.payload", "[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string->} CVE-%{cve},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup110,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup111,
	dup35,
]));

var msg82 = msg("Local::20", part100);

var part101 = match("MESSAGE#196:Local::08", "nwparser.payload", "[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup110,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup34,
]));

var msg83 = msg("Local::08", part101);

var part102 = match("MESSAGE#197:Local::18", "nwparser.payload", "[SID: %{fld26}] %{category}: %{event_description->} Traffic has been blocked for this application:%{fld27},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup110,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup35,
]));

var msg84 = msg("Local::18", part102);

var part103 = match("MESSAGE#198:Local::04/0", "nwparser.payload", "%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all55 = all_match({
	processors: [
		part103,
		dup323,
		dup114,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup34,
	]),
});

var msg85 = msg("Local::04", all55);

var part104 = match("MESSAGE#199:Local::17/0", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},%{p0}");

var all56 = all_match({
	processors: [
		part104,
		dup323,
		dup114,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup35,
	]),
});

var msg86 = msg("Local::17", all56);

var part105 = match("MESSAGE#200:Local::06", "nwparser.payload", "%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol}Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},!ExternalLoggingTask.localport! %{dport},!ExternalLoggingTask.remoteport! %{sport},!ExternalLoggingTask.cidssignid! %{sigid},!ExternalLoggingTask.strcidssignid! %{sigid_string},!ExternalLoggingTask.cidssignsubid! %{sigid1},!ExternalLoggingTask.intrusionurl! %{url},!ExternalLoggingTask.intrusionpayloadurl! %{fld23}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup34,
]));

var msg87 = msg("Local::06", part105);

var part106 = match("MESSAGE#201:Local::16", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},!ExternalLoggingTask.localport! %{sport},!ExternalLoggingTask.remoteport! %{dport},!ExternalLoggingTask.cidssignid! %{sigid},!ExternalLoggingTask.strcidssignid! %{sigid_string},!ExternalLoggingTask.cidssignsubid! %{sigid1},!ExternalLoggingTask.intrusionurl! %{url},!ExternalLoggingTask.intrusionpayloadurl! %{fld23}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup35,
]));

var msg88 = msg("Local::16", part106);

var part107 = match("MESSAGE#202:Local::02", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},%{protocol},0,Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup51,
	dup312,
]));

var msg89 = msg("Local::02", part107);

var part108 = match("MESSAGE#203:Local::22", "nwparser.payload", "%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},%{protocol},1,Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup48,
	dup312,
]));

var msg90 = msg("Local::22", part108);

var part109 = match("MESSAGE#204:Local::23", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},%{protocol},2,Intrusion ID: %{fld33},Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup16,
	dup17,
	dup14,
	dup104,
	dup52,
	dup312,
]));

var msg91 = msg("Local::23", part109);

var part110 = match("MESSAGE#205:Local::07/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string}: %{fld22->} CVE-%{cve->} %{fld26},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var all57 = all_match({
	processors: [
		dup117,
		dup319,
		part110,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup34,
	]),
});

var msg92 = msg("Local::07", all57);

var part111 = match("MESSAGE#206:Local::19/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string}: %{fld22->} CVE-%{cve->} %{fld26},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var all58 = all_match({
	processors: [
		dup118,
		dup319,
		part111,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup35,
	]),
});

var msg93 = msg("Local::19", all58);

var part112 = match("MESSAGE#207:Local::05/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}");

var all59 = all_match({
	processors: [
		dup117,
		dup319,
		part112,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup34,
	]),
});

var msg94 = msg("Local::05", all59);

var part113 = match("MESSAGE#208:Local::15/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}");

var all60 = all_match({
	processors: [
		dup118,
		dup319,
		part113,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup35,
	]),
});

var msg95 = msg("Local::15", all60);

var all61 = all_match({
	processors: [
		dup117,
		dup319,
		dup119,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup34,
	]),
});

var msg96 = msg("Local::03", all61);

var all62 = all_match({
	processors: [
		dup118,
		dup319,
		dup119,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup115,
		dup116,
		dup38,
		dup15,
		dup16,
		dup17,
		dup14,
		dup104,
		dup35,
	]),
});

var msg97 = msg("Local::14", all62);

var part114 = match("MESSAGE#211:Local::10", "nwparser.payload", "Local: %{daddr},Local: %{dport},Remote: %{saddr},Remote: %{fld13},Remote: %{sport},Inbound,Application: %{application},Action: %{action}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup40,
	dup103,
	dup34,
]));

var msg98 = msg("Local::10", part114);

var part115 = match("MESSAGE#212:Local::21", "nwparser.payload", "Local: %{saddr},Local: %{sport},Remote: %{daddr},Remote: %{fld13},Remote: %{dport},Outbound,Application: %{application},Action: %{action}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup103,
	dup35,
	dup40,
]));

var msg99 = msg("Local::21", part115);

var part116 = match("MESSAGE#213:Local::24", "nwparser.payload", "Event Description: %{event_description},Local: %{daddr},Local Host MAC: %{dmacaddr},Remote Host Name: %{fld3},Remote Host IP: %{saddr},Remote Host MAC: %{smacaddr},Inbound,%{protocol},Intrusion ID: 0,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port: %{dport},Remote Port: %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL: %{fld12},SHA-256: %{checksum},MD-5: %{checksum}", processor_chain([
	dup120,
	dup12,
	dup13,
	dup15,
	dup34,
	dup40,
]));

var msg100 = msg("Local::24", part116);

var part117 = match("MESSAGE#214:Local::25", "nwparser.payload", "Event Description: %{event_description},Local: %{saddr},Local Host MAC: %{smacaddr},Remote Host Name: %{fld3},Remote Host IP: %{daddr},Remote Host MAC: %{dmacaddr},Outbound,%{protocol},Intrusion ID: 0,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port: %{sport},Remote Port: %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL: %{fld12},SHA-256: %{checksum},MD-5: %{checksum}", processor_chain([
	dup36,
	dup12,
	dup13,
	dup15,
	dup35,
	dup40,
]));

var msg101 = msg("Local::25", part117);

var part118 = match("MESSAGE#215:Local::26", "nwparser.payload", "Event Description: %{event_description->} [Volume]: %{disk_volume->} [Model]: %{product->} [Access]: %{accesses},Local: %{saddr},Local Host MAC: %{smacaddr},Remote Host Name: %{fld3},Remote Host IP: %{daddr},Remote Host MAC: %{dmacaddr},%{direction},%{fld2},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port: %{sport},Remote Port: %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL: %{fld12},SHA-256: %{checksum},MD-5: %{checksum}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup15,
	dup40,
]));

var msg102 = msg("Local::26", part118);

var select18 = linear_select([
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
]);

var part119 = match("MESSAGE#54:Blocked:13/0", "nwparser.payload", "Blocked Attack: Memory Heap Spray attack against %{fld1},Local: %{daddr},Local: %{fld12},Remote: %{fld15},Remote: %{saddr},Remote: %{fld51},Inbound,%{protocol},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld2},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all63 = all_match({
	processors: [
		part119,
		dup315,
	],
	on_success: processor_chain([
		setc("eventcategory","1001020300"),
		dup12,
		dup13,
		dup14,
		dup16,
		dup17,
		dup15,
		setc("event_description","Attack: Memory Heap Spray attack"),
		dup19,
		dup34,
	]),
});

var msg103 = msg("Blocked:13", all63);

var part120 = match("MESSAGE#483:File:01", "nwparser.payload", "\"%{fld23},\",File Read,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
	dup123,
	dup124,
	dup125,
]));

var msg104 = msg("File:01", part120);

var part121 = match("MESSAGE#484:File:11", "nwparser.payload", "\"%{info}\",Create Process,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld1},%{process},%{fld3},%{fld4},%{application},User: %{username},Domain: %{domain},Action Type:%{fld6},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup126,
]));

var msg105 = msg("File:11", part121);

var part122 = match("MESSAGE#485:File:02", "nwparser.payload", "\"%{info}\",Create Process,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld1},%{process},%{fld3},%{fld4},%{application},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup126,
]));

var msg106 = msg("File:02", part122);

var part123 = match("MESSAGE#486:File:03", "nwparser.payload", "%{fld1},File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup127,
	dup124,
	dup128,
]));

var msg107 = msg("File:03", part123);

var part124 = match("MESSAGE#487:Blocked:04", "nwparser.payload", "%{info}.%{fld1},File Read,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain},Action Type:%{fld46},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup129,
	dup14,
	dup41,
	dup42,
	dup122,
	dup130,
	dup124,
	dup125,
]));

var msg108 = msg("Blocked:04", part124);

var part125 = match("MESSAGE#488:File:05", "nwparser.payload", "%{fld1},File Read,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain},Action Type:%{fld46}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup130,
	dup124,
	dup125,
]));

var msg109 = msg("File:05", part125);

var part126 = match("MESSAGE#489:File:04", "nwparser.payload", "\"%{fld23}\",,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},,%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
	dup123,
]));

var msg110 = msg("File:04", part126);

var part127 = match("MESSAGE#490:File:06", "nwparser.payload", "%{fld1},File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},\"Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup127,
	dup124,
	dup128,
]));

var msg111 = msg("File:06", part127);

var part128 = match("MESSAGE#491:File:07", "nwparser.payload", "'%{fld23}',,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},,%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
	dup123,
]));

var msg112 = msg("File:07", part128);

var part129 = match("MESSAGE#492:File:12", "nwparser.payload", "%{fld23},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{process_id},%{process},%{fld4},,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld6},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup41,
	dup42,
	dup15,
]));

var msg113 = msg("File:12", part129);

var part130 = match("MESSAGE#493:File:08", "nwparser.payload", "%{fld1},%{fld7},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},\"Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup127,
]));

var msg114 = msg("File:08", part130);

var part131 = match("MESSAGE#494:File:09", "nwparser.payload", "%{fld1},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain},Action Type:%{fld6},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	setc("event_description","File Delete."),
	dup124,
	dup131,
]));

var msg115 = msg("File:09", part131);

var part132 = match("MESSAGE#496:Blocked", "nwparser.payload", "Unauthorized NT call rejected by protection driver.,%{fld22},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},%{fld23},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup132,
	dup122,
	setc("event_description","Unauthorized NT call rejected by protection driver."),
]));

var msg116 = msg("Blocked", part132);

var part133 = match("MESSAGE#497:Blocked:01", "nwparser.payload", ",Create Process,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld4},%{process},%{fld5},%{fld6},%{info},User: %{username},Domain: %{domain},Action Type: %{fld8}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
]));

var msg117 = msg("Blocked:01", part133);

var part134 = match("MESSAGE#498:Blocked:02", "nwparser.payload", "%{fld5->} - Caller MD5=%{fld6},Registry Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup133,
]));

var msg118 = msg("Blocked:02", part134);

var part135 = match("MESSAGE#499:Blocked:03/0_0", "nwparser.payload", "%{fld21->} - Caller MD5=%{fld22},Create Process%{p0}");

var part136 = match("MESSAGE#499:Blocked:03/0_1", "nwparser.payload", "%{fld23},Load Dll%{p0}");

var select19 = linear_select([
	part135,
	part136,
]);

var part137 = match("MESSAGE#499:Blocked:03/1", "nwparser.p0", ",Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld24},%{process},%{fld25},%{fld26},%{filename},User: %{username},Domain: %{domain},Action Type: %{fld8},File size (bytes):%{filename_size},Device ID:%{device}");

var all64 = all_match({
	processors: [
		select19,
		part137,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup122,
		setc("event_description","Block from loading other DLLs/processes."),
	]),
});

var msg119 = msg("Blocked:03", all64);

var part138 = match("MESSAGE#500:Blocked:05", "nwparser.payload", "%{event_description->} - Caller MD5=%{checksum},%{fld1},Begin: %{fld2->} %{fld3},End: %{fld4->} %{fld5},Rule: %{rulename},%{process_id},%{process},%{fld6},%{fld7},%{fld8},User: %{username},Domain: %{sdomain},Action Type: %{fld9},File size (%{fld10}): %{filename_size},Device ID:", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup15,
	dup134,
	dup135,
]));

var msg120 = msg("Blocked:05", part138);

var part139 = match("MESSAGE#501:Blocked:06", "nwparser.payload", "[%{id}] %{event_description->} - %{fld11},%{fld1},Begin: %{fld2->} %{fld3},End: %{fld4->} %{fld5},Rule: %{rulename},%{process_id},%{process},%{fld6},%{fld7},%{fld8},User: %{username},Domain: %{domain},Action Type: %{fld9},File size (%{fld10}): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup40,
	dup15,
	dup134,
	dup135,
]));

var msg121 = msg("Blocked:06", part139);

var part140 = match("MESSAGE#502:Blocked:07", "nwparser.payload", "[%{id}] %{event_description},%{fld1},Begin: %{fld2->} %{fld3},End: %{fld4->} %{fld5},Rule: %{rulename},%{process_id},%{process},%{fld6},%{fld7},%{fld8},User: %{username},Domain: %{domain},Action Type: %{fld9},File size (%{fld10}): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup15,
	dup134,
	dup135,
]));

var msg122 = msg("Blocked:07", part140);

var part141 = match("MESSAGE#504:Blocked:09/0_0", "nwparser.payload", "%{fld11->} - Target MD5=%{fld6->} - Target Arguments=%{fld7}/service'%{fld33->} ,Create Process,Begin: %{p0}");

var part142 = match("MESSAGE#504:Blocked:09/0_1", "nwparser.payload", "%{fld11->} - Target MD5=%{fld6->} - Target Arguments=%{fld7}chrome-extension:%{fld99}'%{fld33->} ,Create Process,Begin: %{p0}");

var part143 = match("MESSAGE#504:Blocked:09/0_2", "nwparser.payload", "%{fld11->} - Target MD5=%{fld6->} - Target Arguments=%{fld7}-ServerName:%{hostid}'%{fld33->} ,Create Process,Begin: %{p0}");

var part144 = match("MESSAGE#504:Blocked:09/0_3", "nwparser.payload", "- Target MD5=%{fld6->} - Target Arguments=%{fld7}-ServerName:%{hostid}' ,Create Process,Begin: %{p0}");

var part145 = match("MESSAGE#504:Blocked:09/0_4", "nwparser.payload", "%{fld11->} - Target MD5=%{fld6->} - Target Arguments=%{fld7->} ,Create Process,Begin: %{p0}");

var part146 = match("MESSAGE#504:Blocked:09/0_5", "nwparser.payload", "- Target MD5=%{fld6},Create Process,Begin: %{p0}");

var select20 = linear_select([
	part141,
	part142,
	part143,
	part144,
	part145,
	part146,
]);

var part147 = match("MESSAGE#504:Blocked:09/1", "nwparser.p0", "%{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44->} ,File size (%{fld10}):%{filename_size},Device ID: %{device}");

var all65 = all_match({
	processors: [
		select20,
		part147,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup129,
		dup14,
		dup41,
		dup42,
		dup15,
	]),
});

var msg123 = msg("Blocked:09", all65);

var select21 = linear_select([
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
]);

var part148 = match("MESSAGE#55:Changed/0", "nwparser.payload", "Changed value '%{change_attribute}' from '%{change_old}' to '%{change_new}'%{p0}");

var all66 = all_match({
	processors: [
		part148,
		dup318,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		dup30,
		dup97,
		dup22,
		dup14,
		dup137,
		setc("event_description","Changed value"),
		dup15,
		dup93,
	]),
});

var msg124 = msg("Changed", all66);

var part149 = match("MESSAGE#56:Cleaned", "nwparser.payload", "Cleaned up %{dclass_counter1->} LiveUpdate downloaded content", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Cleaned up downloaded content."),
	setc("dclass_counter1_string","Number of Virus Cleaned."),
]));

var msg125 = msg("Cleaned", part149);

var part150 = match("MESSAGE#57:Client", "nwparser.payload", "Client has downloaded the issued Command,%{username}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup137,
	dup15,
	setc("event_description","Client has downloaded the issued command."),
]));

var msg126 = msg("Client", part150);

var part151 = match("MESSAGE#58:Client:01/0_0", "nwparser.payload", "%{event_description}, type SymDelta version%{version->} filesize%{filename_size}.\",Event time:%{fld17->} %{fld18}");

var part152 = match("MESSAGE#58:Client:01/0_1", "nwparser.payload", "%{event_description}, type full version%{version->} filesize%{filename_size}.\",Event time:%{fld17->} %{fld18}");

var part153 = match_copy("MESSAGE#58:Client:01/0_2", "nwparser.payload", "event_description");

var select22 = linear_select([
	part151,
	part152,
	part153,
]);

var all67 = all_match({
	processors: [
		select22,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
	]),
});

var msg127 = msg("Client:01", all67);

var select23 = linear_select([
	msg126,
	msg127,
]);

var part154 = match("MESSAGE#59:client/0", "nwparser.payload", "client has downloaded the %{p0}");

var part155 = match("MESSAGE#59:client/1_0", "nwparser.p0", "content package%{p0}");

var part156 = match("MESSAGE#59:client/1_1", "nwparser.p0", "policy%{p0}");

var part157 = match("MESSAGE#59:client/1_2", "nwparser.p0", "Intrusion Prevention policy%{p0}");

var select24 = linear_select([
	part155,
	part156,
	part157,
]);

var part158 = match("MESSAGE#59:client/2", "nwparser.p0", "%{}successfully,%{shost},%{username},%{group}");

var all68 = all_match({
	processors: [
		part154,
		select24,
		part158,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","The client has downloaded the policy successfully."),
	]),
});

var msg128 = msg("client", all68);

var part159 = match("MESSAGE#60:client:01", "nwparser.payload", "client has reconnected with the management server,%{shost},%{username},%{group}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The client has reconnected with the management server."),
]));

var msg129 = msg("client:01", part159);

var part160 = match("MESSAGE#61:client:02", "nwparser.payload", "client has downloaded %{filename->} successfully,%{shost},%{username},%{group}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup138,
]));

var msg130 = msg("client:02", part160);

var part161 = match("MESSAGE#62:client:03", "nwparser.payload", "client registered with the management server successfully,%{shost},%{username},%{group}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The client registered with the management server successfully"),
]));

var msg131 = msg("client:03", part161);

var part162 = match("MESSAGE#63:client:04", "nwparser.payload", "client has downloaded %{filename},%{shost},%{username},%{group}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup138,
]));

var msg132 = msg("client:04", part162);

var part163 = match("MESSAGE#64:client:05/2", "nwparser.p0", "Local: %{daddr},Local: %{fld1},Remote: %{fld25},Remote: %{saddr},Remote: %{fld3},Inbound,%{fld5},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld12}");

var all69 = all_match({
	processors: [
		dup139,
		dup324,
		part163,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup142,
		dup19,
		dup143,
		dup34,
	]),
});

var msg133 = msg("client:05", all69);

var part164 = match("MESSAGE#65:client:15/2", "nwparser.p0", "Local: %{saddr},Local: %{fld1},Remote: %{fld25},Remote: %{daddr},Remote: %{fld3},Outbound,%{fld5},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld12}");

var all70 = all_match({
	processors: [
		dup139,
		dup324,
		part164,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup142,
		dup19,
		dup143,
		dup35,
	]),
});

var msg134 = msg("client:15", all70);

var part165 = match("MESSAGE#66:client:06", "nwparser.payload", "client computer has been added to the group,%{shost},%{username},%{group}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Client computer has been added to the group."),
]));

var msg135 = msg("client:06", part165);

var part166 = match("MESSAGE#67:client:07", "nwparser.payload", "client computer has been renamed,%{shost},%{username},%{sdomain}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The client computer has been renamed"),
]));

var msg136 = msg("client:07", part166);

var part167 = match("MESSAGE#68:client:08", "nwparser.payload", "The client does not have a paid license. The current license cannot be used to obtain a client authentication token.,Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The client does not have a paid license"),
]));

var msg137 = msg("client:08", part167);

var part168 = match("MESSAGE#69:client:09", "nwparser.payload", "The client has successfully downloaded and applied a license from the server.,Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	date_time({
		dest: "event_time",
		args: ["event_time_string"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
		],
	}),
	dup15,
	setc("event_description","The client has successfully downloaded and applied a license from the server"),
]));

var msg138 = msg("client:09", part168);

var part169 = match("MESSAGE#693:SYLINK:01/0", "nwparser.payload", "The client opted to download a full definitions package for AV definitions from the management server or GUP %{p0}");

var part170 = match("MESSAGE#693:SYLINK:01/1_0", "nwparser.p0", "because LiveUpdate had no AV updates available%{p0}");

var part171 = match("MESSAGE#693:SYLINK:01/1_1", "nwparser.p0", "rather than download a large package from LiveUpdate%{p0}");

var select25 = linear_select([
	part170,
	part171,
]);

var part172 = match("MESSAGE#693:SYLINK:01/2", "nwparser.p0", ".%{p0}");

var all71 = all_match({
	processors: [
		part169,
		select25,
		part172,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup15,
		dup93,
		setc("event_description","The client opted to download a full definitions package for AV definitions from the management server or GUP"),
	]),
});

var msg139 = msg("SYLINK:01", all71);

var part173 = match("MESSAGE#694:SYLINK:02", "nwparser.payload", "The client opted to download an update for AV definitions from LiveUpdate rather than download a full definitions package from the management server or GUP.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","The client opted to download an update for AV definitions from LiveUpdate"),
]));

var msg140 = msg("SYLINK:02", part173);

var part174 = match("MESSAGE#695:SYLINK:04", "nwparser.payload", "The client has obtained an invalid license file (%{filename}) from the server.,Event time:%{fld17->} %{fld18}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup15,
	dup93,
	setc("event_description","The client has obtained an invalid license file from the server."),
]));

var msg141 = msg("SYLINK:04", part174);

var part175 = match("MESSAGE#697:Smc", "nwparser.payload", "The client has successfully downloaded a license file (%{filename}) from the server.", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The client has successfully downloaded a license file"),
]));

var msg142 = msg("Smc", part175);

var part176 = match("MESSAGE#698:Smc:01/0", "nwparser.payload", "The client has successfully downloaded and applied a license file (%{filename}) from the server.%{p0}");

var all72 = all_match({
	processors: [
		part176,
		dup318,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		dup144,
	]),
});

var msg143 = msg("Smc:01", all72);

var part177 = match("MESSAGE#701:Smc:05/0", "nwparser.payload", "\"The client has successfully downloaded and applied a license file (%{filename}, Serial: %{serial_number}) from the server.\"%{p0}");

var all73 = all_match({
	processors: [
		part177,
		dup318,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		dup144,
	]),
});

var msg144 = msg("Smc:05", all73);

var select26 = linear_select([
	msg128,
	msg129,
	msg130,
	msg131,
	msg132,
	msg133,
	msg134,
	msg135,
	msg136,
	msg137,
	msg138,
	msg139,
	msg140,
	msg141,
	msg142,
	msg143,
	msg144,
]);

var all74 = all_match({
	processors: [
		dup145,
		dup325,
		dup148,
		dup326,
		dup151,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup155,
		dup15,
		dup19,
	]),
});

var msg145 = msg("Commercial", all74);

var part178 = match("MESSAGE#71:Commercial:02/2_0", "nwparser.p0", "%{severity},First Seen: %{fld50},Application name: %{p0}");

var part179 = match("MESSAGE#71:Commercial:02/2_1", "nwparser.p0", "%{severity},Application name: %{p0}");

var select27 = linear_select([
	part178,
	part179,
]);

var part180 = match("MESSAGE#71:Commercial:02/3", "nwparser.p0", "%{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},COH Engine Version: %{fld41},Detection Submissions No,Permitted application reason: %{fld42},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},Risk Level: %{fld50},Detection Source: %{fld52},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var all75 = all_match({
	processors: [
		dup145,
		dup325,
		select27,
		part180,
		dup326,
		dup151,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup155,
		dup15,
		dup19,
	]),
});

var msg146 = msg("Commercial:02", all75);

var part181 = match("MESSAGE#72:Commercial:01/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},\"Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all76 = all_match({
	processors: [
		dup145,
		dup325,
		dup148,
		dup326,
		part181,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup155,
		dup15,
		dup19,
	]),
});

var msg147 = msg("Commercial:01", all76);

var select28 = linear_select([
	msg145,
	msg146,
	msg147,
]);

var part182 = match("MESSAGE#73:Computer:deleted", "nwparser.payload", "Computer has been deleted%{}", processor_chain([
	dup156,
	dup12,
	dup13,
	dup27,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Computer has been deleted."),
]));

var msg148 = msg("Computer:deleted", part182);

var part183 = match("MESSAGE#74:Computer:moved", "nwparser.payload", "Computer has been moved%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Computer has been moved."),
]));

var msg149 = msg("Computer:moved", part183);

var part184 = match("MESSAGE#75:Computer:propertieschanged", "nwparser.payload", "Computer properties have been changed%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Computer properties have been changed."),
]));

var msg150 = msg("Computer:propertieschanged", part184);

var part185 = match("MESSAGE#76:Computer/1_0", "nwparser.p0", "\"%{filename}\",\"%{p0}");

var part186 = match("MESSAGE#76:Computer/1_1", "nwparser.p0", "%{filename},\"%{p0}");

var select29 = linear_select([
	part185,
	part186,
]);

var part187 = match("MESSAGE#76:Computer/2", "nwparser.p0", "%{fld1}\",Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Last update time: %{fld52},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},First Seen: %{fld50},Sensitivity: %{fld58},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size}");

var all77 = all_match({
	processors: [
		dup157,
		select29,
		part187,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup132,
		dup14,
		dup15,
		dup158,
	]),
});

var msg151 = msg("Computer", all77);

var part188 = match("MESSAGE#77:Computer:01/1_0", "nwparser.p0", "\"%{filename}\",'%{p0}");

var part189 = match("MESSAGE#77:Computer:01/1_1", "nwparser.p0", "%{filename},'%{p0}");

var select30 = linear_select([
	part188,
	part189,
]);

var part190 = match("MESSAGE#77:Computer:01/2", "nwparser.p0", "%{fld1}',Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Last update time: %{fld52},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},First Seen: %{fld50},Sensitivity: %{fld58},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size},Category set: %{category},Category type: %{event_type}");

var all78 = all_match({
	processors: [
		dup157,
		select30,
		part190,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup132,
		dup14,
		dup15,
		dup158,
	]),
});

var msg152 = msg("Computer:01", all78);

var part191 = match("MESSAGE#78:Computer:03/0", "nwparser.payload", "IP Address: %{hostip},Computer name: %{shost},Intensive Protection Level: %{fld55},Certificate issuer: %{cert_subject},Certificate signer: %{fld68},Certificate thumbprint: %{fld57},Signing timestamp: %{fld69},Certificate serial number: %{cert.serial},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{p0}");

var part192 = match("MESSAGE#78:Computer:03/2", "nwparser.p0", "%{fld1},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Last update time: %{fld52},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},%{fld67},First Seen: %{fld50},Sensitivity: %{fld58},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size},Category set: %{category},Category type: %{event_type},Location:%{fld65}");

var all79 = all_match({
	processors: [
		part191,
		dup327,
		part192,
	],
	on_success: processor_chain([
		setc("eventcategory","1003000000"),
		dup12,
		dup132,
		dup15,
		dup93,
		dup47,
	]),
});

var msg153 = msg("Computer:03", all79);

var part193 = match("MESSAGE#79:Computer:02/0", "nwparser.payload", "Computer name: %{p0}");

var all80 = all_match({
	processors: [
		part193,
		dup325,
		dup161,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup132,
		dup152,
		dup162,
		dup163,
		dup164,
		dup154,
		dup15,
		dup19,
	]),
});

var msg154 = msg("Computer:02", all80);

var select31 = linear_select([
	msg148,
	msg149,
	msg150,
	msg151,
	msg152,
	msg153,
	msg154,
]);

var part194 = match("MESSAGE#80:Configuration", "nwparser.payload", "Configuration Change..Computer: %{shost}..Date: %{fld5}..Time: %{fld6}..Description: %{event_description->} ..Severity: %{severity}..Source: %{product}", processor_chain([
	dup165,
	dup166,
	dup15,
]));

var msg155 = msg("Configuration", part194);

var part195 = match("MESSAGE#81:Configuration:01", "nwparser.payload", "Configuration Change..%{shost}..%{fld5}........%{severity}..%{product}..%{fld6->} %{fld7}..", processor_chain([
	dup165,
	dup166,
	setc("event_description","Configuration Change"),
	dup15,
]));

var msg156 = msg("Configuration:01", part195);

var part196 = match("MESSAGE#82:Configuration:02", "nwparser.payload", "Configuration Change..Computer: %{shost}..Date: %{fld5}..Description: %{event_description}..Time: %{fld6->} %{fld7}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup165,
	dup166,
	dup15,
]));

var msg157 = msg("Configuration:02", part196);

var select32 = linear_select([
	msg155,
	msg156,
	msg157,
]);

var part197 = match("MESSAGE#83:Connected/0", "nwparser.payload", "Connected to Symantec Endpoint Protection Manager %{p0}");

var part198 = match("MESSAGE#83:Connected/1_0", "nwparser.p0", "%{fld11->} ,Event time: %{fld17->} %{fld18}");

var part199 = match("MESSAGE#83:Connected/1_1", "nwparser.p0", "%{fld11}");

var select33 = linear_select([
	part198,
	part199,
]);

var all81 = all_match({
	processors: [
		part197,
		select33,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup93,
		dup15,
		setc("event_description","Connected to Symantec Endpoint Protection Manager"),
	]),
});

var msg158 = msg("Connected", all81);

var part200 = match("MESSAGE#686:Connected:01", "nwparser.payload", "Connected to Management Server %{hostip}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Connected to Management Server"),
]));

var msg159 = msg("Connected:01", part200);

var select34 = linear_select([
	msg158,
	msg159,
]);

var part201 = match("MESSAGE#84:Connection", "nwparser.payload", "Connection reset%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Connection reset."),
]));

var msg160 = msg("Connection", part201);

var part202 = match("MESSAGE#85:Could", "nwparser.payload", "Could %{space}not start Service Engine err=%{resultcode}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup14,
	dup15,
	dup167,
]));

var msg161 = msg("Could", part202);

var part203 = match("MESSAGE#86:Could:01", "nwparser.payload", "Could not scan %{dclass_counter1->} files inside %{directory->} due to extraction errors encountered by the Decomposer Engines.", processor_chain([
	dup86,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("dclass_counter1_string","Number of Files"),
	dup167,
]));

var msg162 = msg("Could:01", part203);

var select35 = linear_select([
	msg161,
	msg162,
]);

var part204 = match("MESSAGE#87:Create", "nwparser.payload", "Create trident engine failed.%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Create trident engine failed."),
]));

var msg163 = msg("Create", part204);

var part205 = match("MESSAGE#88:Database", "nwparser.payload", "Database Maintenance Finished Successfully%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Database Maintenance Finished Successfully"),
]));

var msg164 = msg("Database", part205);

var part206 = match("MESSAGE#89:Database:01", "nwparser.payload", "Database maintenance started.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","Database maintenance started."),
]));

var msg165 = msg("Database:01", part206);

var part207 = match("MESSAGE#90:Database:02", "nwparser.payload", "Database maintenance finished successfully.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","Database maintenance finished successfully."),
]));

var msg166 = msg("Database:02", part207);

var part208 = match("MESSAGE#91:Database:03", "nwparser.payload", "Database properties are changed%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","Database properties are changed"),
]));

var msg167 = msg("Database:03", part208);

var select36 = linear_select([
	msg164,
	msg165,
	msg166,
	msg167,
]);

var part209 = match("MESSAGE#92:Disconnected", "nwparser.payload", "Disconnected from Symantec Endpoint Protection Manager. --- server address : %{hostid}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup169,
]));

var msg168 = msg("Disconnected", part209);

var part210 = match("MESSAGE#93:Disconnected:01/0", "nwparser.payload", "Disconnected from Symantec Endpoint Protection Manager (%{hostip})%{p0}");

var all82 = all_match({
	processors: [
		part210,
		dup318,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		dup30,
		dup97,
		dup14,
		dup15,
		dup93,
		dup169,
	]),
});

var msg169 = msg("Disconnected:01", all82);

var part211 = match("MESSAGE#687:Disconnected:02", "nwparser.payload", "Disconnected to Management Server %{hostip}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Disconnected to Management Server"),
]));

var msg170 = msg("Disconnected:02", part211);

var select37 = linear_select([
	msg168,
	msg169,
	msg170,
]);

var part212 = match_copy("MESSAGE#94:Decomposer", "nwparser.payload", "event_description", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg171 = msg("Decomposer", part212);

var part213 = match("MESSAGE#95:Domain:added", "nwparser.payload", "Domain \"%{domain}\" was added", processor_chain([
	dup95,
	dup12,
	dup13,
	dup96,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Domain was added."),
]));

var msg172 = msg("Domain:added", part213);

var part214 = match("MESSAGE#96:Domain:renamed", "nwparser.payload", "Domain \"%{change_old}\" was renamed to \"%{change_new}\"", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Domain was renamed."),
	setc("change_attribute","domain name"),
]));

var msg173 = msg("Domain:renamed", part214);

var part215 = match("MESSAGE#97:Domain:deleted", "nwparser.payload", "Domain \"%{domain}\" was deleted!", processor_chain([
	dup156,
	dup12,
	dup13,
	dup27,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Domain was deleted."),
]));

var msg174 = msg("Domain:deleted", part215);

var part216 = match("MESSAGE#98:Domain:administratoradded", "nwparser.payload", "Domain administrator \"%{username}\" was added", processor_chain([
	dup170,
	dup12,
	dup13,
	dup20,
	dup96,
	dup28,
	dup22,
	dup14,
	dup15,
	dup158,
	setc("event_description","Domain administrator was added."),
]));

var msg175 = msg("Domain:administratoradded", part216);

var part217 = match("MESSAGE#99:Domain:administratordeleted", "nwparser.payload", "Domain administrator \"%{username}\" was deleted", processor_chain([
	dup171,
	dup12,
	dup13,
	dup20,
	dup27,
	dup28,
	dup22,
	dup14,
	dup15,
	dup158,
	setc("event_description","Domain administrator deleted."),
]));

var msg176 = msg("Domain:administratordeleted", part217);

var part218 = match("MESSAGE#100:Domain:disabled", "nwparser.payload", "Domain \"%{domain}\" was disabled", processor_chain([
	dup136,
	dup12,
	dup13,
	dup56,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Domain disabled"),
]));

var msg177 = msg("Domain:disabled", part218);

var part219 = match("MESSAGE#101:Domain:enabled", "nwparser.payload", "Domain \"%{domain}\" was enabled", processor_chain([
	dup136,
	dup12,
	dup13,
	dup172,
	dup97,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Domain enabled"),
]));

var msg178 = msg("Domain:enabled", part219);

var select38 = linear_select([
	msg172,
	msg173,
	msg174,
	msg175,
	msg176,
	msg177,
	msg178,
]);

var part220 = match("MESSAGE#102:Failed", "nwparser.payload", "Failed to connect to the server. %{action}. ErrorCode: %{resultcode}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	dup173,
]));

var msg179 = msg("Failed", part220);

var part221 = match("MESSAGE#103:Failed:01/0", "nwparser.payload", "Failed to contact server for more than %{p0}");

var part222 = match("MESSAGE#103:Failed:01/1_0", "nwparser.p0", "%{fld1->} times.,Event time:%{fld17->} %{fld18}");

var part223 = match("MESSAGE#103:Failed:01/1_1", "nwparser.p0", "%{fld1->} times.");

var select39 = linear_select([
	part222,
	part223,
]);

var all83 = all_match({
	processors: [
		part221,
		select39,
	],
	on_success: processor_chain([
		dup74,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		dup173,
	]),
});

var msg180 = msg("Failed:01", all83);

var part224 = match("MESSAGE#104:Failed:02", "nwparser.payload", "Failed to disable Windows firewall%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Failed to disable Windows firewall."),
]));

var msg181 = msg("Failed:02", part224);

var part225 = match("MESSAGE#105:Failed:03", "nwparser.payload", "Failed to install teefer driver%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Failed to install teefer driver."),
]));

var msg182 = msg("Failed:03", part225);

var part226 = match("MESSAGE#106:Failed:04", "nwparser.payload", "Failed to connect to %{fld22}. Make sure the server can ping or resolve this domain. ErrorCode: %{resultcode}", processor_chain([
	dup168,
	dup14,
	dup15,
	setc("event_description","Failed to connect."),
]));

var msg183 = msg("Failed:04", part226);

var part227 = match("MESSAGE#107:Failed:05", "nwparser.payload", "Failed to download new client upgrade package from the management server. New Version: %{version->} Package size: %{filename_size->} bytes. Package url: %{url}", processor_chain([
	dup168,
	dup12,
	dup13,
	setc("ec_subject","Agent"),
	dup97,
	dup25,
	dup14,
	dup15,
	setc("event_description","Failed to download new client upgrade package from the management server."),
]));

var msg184 = msg("Failed:05", part227);

var part228 = match("MESSAGE#108:Failed:06", "nwparser.payload", "Failed to import server policy.%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup97,
	dup25,
	dup14,
	dup15,
	setc("event_description","Failed to import server policy."),
]));

var msg185 = msg("Failed:06", part228);

var part229 = match("MESSAGE#109:Failed:07", "nwparser.payload", "Failed to load plugin:%{filename}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup97,
	dup25,
	dup14,
	dup15,
	setc("event_description","Failed to load plugin"),
]));

var msg186 = msg("Failed:07", part229);

var part230 = match("MESSAGE#110:Failed:08", "nwparser.payload", "Failed to clean up LiveUpdate downloaded content%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup97,
	dup25,
	dup14,
	dup15,
	setc("event_description","Failed to clean up LiveUpdate downloaded content"),
]));

var msg187 = msg("Failed:08", part230);

var part231 = match("MESSAGE#111:Failed:09", "nwparser.payload", "Failed to Login to Remote Site [%{node}] Failed to connect to the server. Make sure that the server is running and your session has not timed out. If you can reach the server but cannot log on, make sure that you provided the correct parameters. If you are experiencing network issues, contact your system administrator.", processor_chain([
	dup174,
	dup12,
	dup13,
	dup97,
	dup25,
	dup14,
	dup15,
	dup175,
]));

var msg188 = msg("Failed:09", part231);

var part232 = match("MESSAGE#112:Failed:10", "nwparser.payload", "Failed to Login to Remote Site [%{node}] Replication partnership has been deleted from remote site.", processor_chain([
	dup174,
	dup12,
	dup13,
	dup97,
	dup25,
	dup14,
	dup15,
	dup175,
]));

var msg189 = msg("Failed:10", part232);

var part233 = match("MESSAGE#113:Failed:11", "nwparser.payload", "Failed to import new policy.,Event time: %{event_time_string}", processor_chain([
	setc("eventcategory","1601000000"),
	dup12,
	dup13,
	dup15,
	setc("event_description","Failed to import new policy."),
]));

var msg190 = msg("Failed:11", part233);

var part234 = match("MESSAGE#250:Network:24/0", "nwparser.payload", "Failed to set a custom action for IPS signature %{sigid->} (errcode=0x%{resultcode}). Most probably, this IPS signature was removed from the IPS content.%{p0}");

var select40 = linear_select([
	dup176,
	dup91,
]);

var all84 = all_match({
	processors: [
		part234,
		select40,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","Failed to set a custom action for IPS signature"),
	]),
});

var msg191 = msg("Network:24", all84);

var part235 = match("MESSAGE#696:SYLINK:03", "nwparser.payload", "Failed to connect to all GUPs, now trying to connect SEPM\"%{}", processor_chain([
	dup74,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Failed to connect to all GUPs."),
]));

var msg192 = msg("SYLINK:03", part235);

var select41 = linear_select([
	msg179,
	msg180,
	msg181,
	msg182,
	msg183,
	msg184,
	msg185,
	msg186,
	msg187,
	msg188,
	msg189,
	msg190,
	msg191,
	msg192,
]);

var part236 = match("MESSAGE#114:Firewall", "nwparser.payload", "Firewall driver failed to %{info}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Firewall driver failed."),
]));

var msg193 = msg("Firewall", part236);

var part237 = match("MESSAGE#115:Firewall:01", "nwparser.payload", "Firewall is enabled,Event time: %{event_time_string}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Firewall is enabled"),
]));

var msg194 = msg("Firewall:01", part237);

var part238 = match("MESSAGE#116:Firewall:02", "nwparser.payload", "Firewall is disabled by policy,Event time: %{event_time_string}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Firewall is disabled by policy"),
]));

var msg195 = msg("Firewall:02", part238);

var part239 = match("MESSAGE#117:Firewall:03", "nwparser.payload", "Firewall is disabled,Event time: %{event_time_string}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Firewall is disabled"),
]));

var msg196 = msg("Firewall:03", part239);

var select42 = linear_select([
	msg193,
	msg194,
	msg195,
	msg196,
]);

var part240 = match("MESSAGE#118:Group:created", "nwparser.payload", "Group has been created%{}", processor_chain([
	dup95,
	dup12,
	dup13,
	dup177,
	dup96,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group has been created"),
]));

var msg197 = msg("Group:created", part240);

var part241 = match("MESSAGE#119:Group:deleted", "nwparser.payload", "Group has been deleted%{}", processor_chain([
	dup156,
	dup12,
	dup13,
	dup177,
	dup27,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group has been deleted"),
]));

var msg198 = msg("Group:deleted", part241);

var part242 = match("MESSAGE#120:Group:deleted_01", "nwparser.payload", "Group '%{group}' was deleted", processor_chain([
	dup156,
	dup12,
	dup13,
	dup177,
	dup27,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group was deleted"),
]));

var msg199 = msg("Group:deleted_01", part242);

var part243 = match("MESSAGE#121:Group:moved", "nwparser.payload", "Group has been moved%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup177,
	dup30,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group has been moved"),
]));

var msg200 = msg("Group:moved", part243);

var part244 = match("MESSAGE#122:Group:renamed", "nwparser.payload", "Group has been renamed%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup177,
	dup30,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group has been renamed"),
]));

var msg201 = msg("Group:renamed", part244);

var part245 = match("MESSAGE#123:Group:added", "nwparser.payload", "Group '%{group}' was added", processor_chain([
	dup95,
	dup12,
	dup13,
	dup177,
	dup30,
	dup178,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","Group was added"),
]));

var msg202 = msg("Group:added", part245);

var select43 = linear_select([
	msg197,
	msg198,
	msg199,
	msg200,
	msg201,
	msg202,
]);

var part246 = match("MESSAGE#124:Host", "nwparser.payload", "Host Integrity check is disabled. %{info->} by the %{username}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup56,
	dup97,
	dup22,
	dup14,
	dup15,
	dup180,
]));

var msg203 = msg("Host", part246);

var part247 = match("MESSAGE#125:Host:01", "nwparser.payload", "%{info->} up-to-date", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Component is up-to-date"),
]));

var msg204 = msg("Host:01", part247);

var part248 = match("MESSAGE#126:Host:02", "nwparser.payload", "Host Integrity check failed Requirement: \"%{fld11}\" passed Requirement: \"%{fld12}\" failed Requirement: \"%{fld13}\" passed Requirement: \"%{fld14}\" passed %{fld44},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup25,
	dup14,
	dup15,
	dup89,
]));

var msg205 = msg("Host:02", part248);

var part249 = match("MESSAGE#127:Host:05", "nwparser.payload", "Host Integrity failed but reported as pass Requirement: \"%{fld11}\" passed Requirement: \"%{fld12}\" passed Requirement: \"%{fld13}\" passed Requirement: \"%{fld14}\" failed %{fld44},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup25,
	dup14,
	dup15,
	dup181,
]));

var msg206 = msg("Host:05", part249);

var part250 = match("MESSAGE#128:Host:06", "nwparser.payload", "Host Integrity failed but reported as pass Requirement: \"%{fld11}\" %{fld18->} Requirement: \"%{fld12}\" %{fld17->} Requirement: \"%{fld13}\" %{fld16->} Requirement: \"%{fld14}\" %{fld15->} %{fld44},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup25,
	dup14,
	dup15,
	dup181,
]));

var msg207 = msg("Host:06", part250);

var part251 = match("MESSAGE#129:Host:04", "nwparser.payload", "Host Integrity check failed %{result},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup25,
	dup14,
	dup15,
	setc("event_description","Host Integrity check failed"),
]));

var msg208 = msg("Host:04", part251);

var part252 = match("MESSAGE#130:Host:03", "nwparser.payload", "Host Integrity check passed Requirement: \"%{fld11}\" passed Requirement: \"%{fld12}\" passed Requirement: \"%{fld13}\" passed Requirement: \"%{fld14}\" passed %{fld44},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup88,
]));

var msg209 = msg("Host:03", part252);

var part253 = match("MESSAGE#132:Host:07", "nwparser.payload", "Host Integrity check passed%{space}Requirement: '%{fld11}' passed %{fld12},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup87,
	dup12,
	dup13,
	dup22,
	dup14,
	dup15,
	dup88,
]));

var msg210 = msg("Host:07", part253);

var part254 = match("MESSAGE#133:Host:08/0_0", "nwparser.payload", "%{shost}, Host Integrity check passed %{p0}");

var part255 = match("MESSAGE#133:Host:08/0_1", "nwparser.payload", "Host Integrity check passed%{p0}");

var select44 = linear_select([
	part254,
	part255,
]);

var part256 = match("MESSAGE#133:Host:08/1", "nwparser.p0", "%{},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{url},Intrusion Payload URL:%{fld25}");

var all85 = all_match({
	processors: [
		select44,
		part256,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup22,
		dup15,
		dup88,
		dup40,
		dup41,
		dup42,
		dup47,
	]),
});

var msg211 = msg("Host:08", all85);

var part257 = match("MESSAGE#134:Host:09/0", "nwparser.payload", "%{shost}, Host Integrity check pass.%{info},Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},%{p0}");

var select45 = linear_select([
	dup67,
	dup182,
]);

var part258 = match("MESSAGE#134:Host:09/2", "nwparser.p0", "%{} %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{url},Intrusion Payload URL:%{fld25}");

var all86 = all_match({
	processors: [
		part257,
		select45,
		part258,
	],
	on_success: processor_chain([
		dup179,
		dup12,
		dup15,
		dup40,
		dup41,
		dup42,
		dup47,
	]),
});

var msg212 = msg("Host:09", all86);

var part259 = match("MESSAGE#702:Smc:06", "nwparser.payload", "Host Integrity check is disabled. Only do Host Integrity checking when connected to the Symantec Endpoint Protection Manager is checked.,Event time: %{fld17->} %{fld18}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	dup93,
	dup180,
]));

var msg213 = msg("Smc:06", part259);

var select46 = linear_select([
	msg203,
	msg204,
	msg205,
	msg206,
	msg207,
	msg208,
	msg209,
	msg210,
	msg211,
	msg212,
	msg213,
]);

var part260 = match("MESSAGE#131:??:", "nwparser.payload", "%{fld31->} ??????????????? ??: \"%{fld11}\"?? ??: \"%{fld12}\"?? ??: \"%{fld13}\"?? ??: \"%{fld14}\"??,??????????? ,Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld25}", processor_chain([
	dup179,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg214 = msg("??:", part260);

var part261 = match("MESSAGE#135:Intrusion/0", "nwparser.payload", "%{info->} %{p0}");

var part262 = match("MESSAGE#135:Intrusion/1_1", "nwparser.p0", "was %{p0}");

var select47 = linear_select([
	dup183,
	part262,
]);

var part263 = match("MESSAGE#135:Intrusion/2", "nwparser.p0", "%{action}");

var all87 = all_match({
	processors: [
		part261,
		select47,
		part263,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		setc("event_description","Intrusion Prevention signatures is up-to-date."),
		dup15,
	]),
});

var msg215 = msg("Intrusion", all87);

var part264 = match("MESSAGE#136:Intrusion:01", "nwparser.payload", "%{info->} failed to update", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	setc("event_description"," Failed to update Signature"),
	dup15,
]));

var msg216 = msg("Intrusion:01", part264);

var select48 = linear_select([
	msg215,
	msg216,
]);

var part265 = match("MESSAGE#137:Invalid", "nwparser.payload", "Invalid log record:%{info}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Invalid log record"),
]));

var msg217 = msg("Invalid", part265);

var part266 = match("MESSAGE#138:Limited", "nwparser.payload", "Limited Administrator administrator \"%{change_old}\" was renamed to \"%{change_new}\"", processor_chain([
	setc("eventcategory","1402020300"),
	dup12,
	dup13,
	dup30,
	dup22,
	dup14,
	dup15,
	setc("event_description","Limited Administrator renamed"),
	dup23,
	setc("change_attribute","limited administrator username."),
]));

var msg218 = msg("Limited", part266);

var part267 = match("MESSAGE#139:LiveUpdate:08", "nwparser.payload", "LiveUpdate will start next on %{info->} on %{product}", processor_chain([
	dup43,
	dup15,
	dup184,
]));

var msg219 = msg("LiveUpdate:08", part267);

var part268 = match("MESSAGE#140:LiveUpdate:01", "nwparser.payload", "LiveUpdate %{info->} on %{product}\"", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup184,
]));

var msg220 = msg("LiveUpdate:01", part268);

var part269 = match("MESSAGE#141:LiveUpdate", "nwparser.payload", "LiveUpdate failed.%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","LiveUpdate failed."),
]));

var msg221 = msg("LiveUpdate", part269);

var part270 = match("MESSAGE#142:LiveUpdate:04", "nwparser.payload", "LiveUpdate encountered one or more errors. Return code = %{resultcode}", processor_chain([
	dup168,
	dup15,
	setc("event_description","LiveUpdate encountered one or more errors"),
]));

var msg222 = msg("LiveUpdate:04", part270);

var part271 = match("MESSAGE#143:LiveUpdate:02", "nwparser.payload", "LiveUpdate succeeded%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","LiveUpdate succeeded"),
]));

var msg223 = msg("LiveUpdate:02", part271);

var part272 = match("MESSAGE#144:LiveUpdate:09", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,[LiveUpdate error submission] Submitting information to Symantec failed.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup185,
]));

var msg224 = msg("LiveUpdate:09", part272);

var part273 = match("MESSAGE#145:LiveUpdate:10/0", "nwparser.payload", "LiveUpdate encountered an error: Failed to connect to the LiveUpdate server (%{resultcode})%{p0}");

var select49 = linear_select([
	dup186,
	dup91,
]);

var all88 = all_match({
	processors: [
		part273,
		select49,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Failed to connect to the LiveUpdate server"),
	]),
});

var msg225 = msg("LiveUpdate:10", all88);

var part274 = match("MESSAGE#146:LiveUpdate:11/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,\"An update for %{application->} failed to install. Error: %{resultcode}, DuResult:%{fld23}.\"%{p0}");

var all89 = all_match({
	processors: [
		part274,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","An update failed to install"),
	]),
});

var msg226 = msg("LiveUpdate:11", all89);

var part275 = match("MESSAGE#147:LiveUpdate:12", "nwparser.payload", "LiveUpdate re-run triggered by the download of content catalog.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","LiveUpdate re-run triggered by the download of content catalog."),
]));

var msg227 = msg("LiveUpdate:12", part275);

var part276 = match("MESSAGE#148:LiveUpdate:13", "nwparser.payload", "LiveUpdate cannot be run because all licenses have expired.%{}", processor_chain([
	dup43,
	dup14,
	dup15,
	setc("event_description","LiveUpdate cannot be run because all licenses have expired."),
]));

var msg228 = msg("LiveUpdate:13", part276);

var part277 = match("MESSAGE#149:LiveUpdate::05", "nwparser.payload", "LiveUpdate started.%{}", processor_chain([
	dup43,
	dup15,
	setc("action","LiveUpdate started."),
]));

var msg229 = msg("LiveUpdate::05", part277);

var part278 = match("MESSAGE#150:LiveUpdate::06", "nwparser.payload", "LiveUpdate retry started.%{}", processor_chain([
	dup43,
	dup15,
	setc("action","LiveUpdate retry started."),
]));

var msg230 = msg("LiveUpdate::06", part278);

var part279 = match("MESSAGE#151:LiveUpdate::07", "nwparser.payload", "LiveUpdate retry succeeded.%{}", processor_chain([
	dup43,
	dup15,
	setc("action","LiveUpdate retry succeeded."),
]));

var msg231 = msg("LiveUpdate::07", part279);

var part280 = match("MESSAGE#152:LiveUpdate::08", "nwparser.payload", "LiveUpdate retry failed. Will try again.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("action","LiveUpdate retry failed."),
]));

var msg232 = msg("LiveUpdate::08", part280);

var part281 = match("MESSAGE#153:LiveUpdate:14", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Centralized Reputation Settings from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","An update for Centralized Reputation Settings from LiveUpdate failed to install."),
]));

var msg233 = msg("LiveUpdate:14", part281);

var part282 = match("MESSAGE#154:LiveUpdate:15", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Intrusion Prevention Signatures (hub) from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Intrusion Prevention Signatures (hub) from LiveUpdate failed to install."),
]));

var msg234 = msg("LiveUpdate:15", part282);

var part283 = match("MESSAGE#155:LiveUpdate:16", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Intrusion Prevention Signatures from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Intrusion Prevention Signatures from LiveUpdate failed to install."),
]));

var msg235 = msg("LiveUpdate:16", part283);

var part284 = match("MESSAGE#156:LiveUpdate:17", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Revocation Data from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","An update for Revocation Data from LiveUpdate failed to install."),
]));

var msg236 = msg("LiveUpdate:17", part284);

var part285 = match("MESSAGE#157:LiveUpdate:18/0", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for SONAR Definitions from LiveUpdate failed to install. Error:%{result}(%{resultcode})%{p0}");

var all90 = all_match({
	processors: [
		part285,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","An update for SONAR Definitions from LiveUpdate failed to install."),
	]),
});

var msg237 = msg("LiveUpdate:18", all90);

var part286 = match("MESSAGE#158:LiveUpdate:19/0", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Symantec Whitelist from LiveUpdate failed to install. Error:%{result}(%{resultcode})%{p0}");

var all91 = all_match({
	processors: [
		part286,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","An update for Symantec Whitelist from LiveUpdate failed to install."),
	]),
});

var msg238 = msg("LiveUpdate:19", all91);

var part287 = match("MESSAGE#159:LiveUpdate:20", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Virus and Spyware Definitions Win32 (hub) from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Virus and Spyware Definitions Win32 (hub) from LiveUpdate failed to install."),
]));

var msg239 = msg("LiveUpdate:20", part287);

var part288 = match("MESSAGE#160:LiveUpdate:21", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Virus and Spyware Definitions Win32 from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Virus and Spyware Definitions Win32 from LiveUpdate failed to install."),
]));

var msg240 = msg("LiveUpdate:21", part288);

var part289 = match("MESSAGE#161:LiveUpdate:22", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Virus and Spyware Definitions Win64 (hub) from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup12,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Virus and Spyware Definitions Win64 (hub) from LiveUpdate failed to install."),
]));

var msg241 = msg("LiveUpdate:22", part289);

var part290 = match("MESSAGE#162:LiveUpdate:23", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,An update for Virus and Spyware Definitions Win64 from LiveUpdate failed to install. Error:%{result}(%{resultcode})", processor_chain([
	dup43,
	dup94,
	dup13,
	dup45,
	dup38,
	dup25,
	dup14,
	dup15,
	setc("event_description","An update for Virus and Spyware Definitions Win64 from LiveUpdate failed to install."),
]));

var msg242 = msg("LiveUpdate:23", part290);

var part291 = match("MESSAGE#163:LiveUpdate:24/0", "nwparser.payload", "LiveUpdate encountered an error: %{result->} (%{resultcode}).%{p0}");

var all92 = all_match({
	processors: [
		part291,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup38,
		dup187,
		dup14,
		dup15,
		dup93,
		dup188,
	]),
});

var msg243 = msg("LiveUpdate:24", all92);

var part292 = match("MESSAGE#164:LiveUpdate:25", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest Revocation Data update failed to load. The component has no valid content and will not function correctly until it is updated.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The latest Revocation Data update failed to load. The component has no valid content and will not function correctly until it is updated."),
]));

var msg244 = msg("LiveUpdate:25", part292);

var part293 = match("MESSAGE#165:LiveUpdate:26", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest Symantec Whitelist update failed to load. The component has no valid content and will not function correctly until it is updated.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The latest Symantec Whitelist update failed to load. The component has no valid content and will not function correctly until it is updated."),
]));

var msg245 = msg("LiveUpdate:26", part293);

var part294 = match("MESSAGE#166:LiveUpdate:27/0", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,A LiveUpdate session encountered errors. %{fld1->} update(s) were available. %{fld2->} update(s) installed successfully. %{fld3->} update(s) failed to install.%{p0}");

var all93 = all_match({
	processors: [
		part294,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","LiveUpdate session encountered errors"),
	]),
});

var msg246 = msg("LiveUpdate:27", all93);

var part295 = match("MESSAGE#167:LiveUpdate:28/0", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest Revocation Data update failed to load. The component will continue to use its previous content.%{p0}");

var all94 = all_match({
	processors: [
		part295,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","The latest Revocation Data update failed to load."),
	]),
});

var msg247 = msg("LiveUpdate:28", all94);

var part296 = match("MESSAGE#168:LiveUpdate:29", "nwparser.payload", "%{fld11}: Impossible de se connecter au serveur LiveUpdate %{fld12}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","LiveUpdate a rencontrï¿½ une erreur"),
]));

var msg248 = msg("LiveUpdate:29", part296);

var part297 = match("MESSAGE#169:LiveUpdate:30/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,An update for %{application->} was successfully installed.%{space}The new sequence number is %{fld23}.%{p0}");

var part298 = match("MESSAGE#169:LiveUpdate:30/1_0", "nwparser.p0", "%{space}Content was downloaded from %{url->} (%{sport}).,Event time:%{fld17->} %{fld18}");

var part299 = match("MESSAGE#169:LiveUpdate:30/1_1", "nwparser.p0", "%{space}Content was downloaded from %{url->} (%{sport}).");

var select50 = linear_select([
	part298,
	part299,
	dup90,
	dup91,
]);

var all95 = all_match({
	processors: [
		part297,
		select50,
	],
	on_success: processor_chain([
		dup43,
		dup189,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","An update from LiveUpdate Manager installed successfully"),
	]),
});

var msg249 = msg("LiveUpdate:30", all95);

var part300 = match("MESSAGE#170:LiveUpdate:31/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest %{application->} update failed to load. The component will continue to use its previous content.%{p0}");

var all96 = all_match({
	processors: [
		part300,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup189,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","The latest update from LiveUpdate Manager failed to load."),
	]),
});

var msg250 = msg("LiveUpdate:31", all96);

var part301 = match("MESSAGE#171:LiveUpdate:32", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,Scheduled LiveUpdate switched to %{change_new}.", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	setc("event_description","Scheduled LiveUpdate interval switched."),
]));

var msg251 = msg("LiveUpdate:32", part301);

var part302 = match("MESSAGE#172:LiveUpdate:33/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,An update for %{application->} from LiveUpdate failed to install. Error: %{result}(%{resultcode})%{p0}");

var all97 = all_match({
	processors: [
		part302,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","An update from LiveUpdate Manager failed to install."),
	]),
});

var msg252 = msg("LiveUpdate:33", all97);

var part303 = match("MESSAGE#173:LiveUpdate:34", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,An update for %{application->} from Intelligent Updater was already installed.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","An update from Intelligent Updater already installed."),
]));

var msg253 = msg("LiveUpdate:34", part303);

var part304 = match("MESSAGE#174:LiveUpdate:35/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,%{p0}");

var part305 = match("MESSAGE#174:LiveUpdate:35/1_0", "nwparser.p0", "A %{p0}");

var part306 = match("MESSAGE#174:LiveUpdate:35/1_1", "nwparser.p0", " The%{p0}");

var select51 = linear_select([
	part305,
	part306,
]);

var part307 = match("MESSAGE#174:LiveUpdate:35/2", "nwparser.p0", "%{}LiveUpdate session %{p0}");

var part308 = match("MESSAGE#174:LiveUpdate:35/3_1", "nwparser.p0", "was%{p0}");

var select52 = linear_select([
	dup183,
	part308,
]);

var part309 = match("MESSAGE#174:LiveUpdate:35/4", "nwparser.p0", "%{}cancelled.");

var all98 = all_match({
	processors: [
		part304,
		select51,
		part307,
		select52,
		part309,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","A LiveUpdate session from LiveUpdate Manager was cancelled."),
	]),
});

var msg254 = msg("LiveUpdate:35", all98);

var part310 = match("MESSAGE#175:LiveUpdate:36/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,\"A LiveUpdate session is already running, so the scheduled LiveUpdate was skipped.\"%{p0}");

var all99 = all_match({
	processors: [
		part310,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","A LiveUpdate session from LiveUpdate Manager is running, LiveUpdate skipped."),
	]),
});

var msg255 = msg("LiveUpdate:36", all99);

var part311 = match("MESSAGE#176:LiveUpdate:37", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,Scheduled LiveUpdate keep trying to connect to Server for %{fld23->} times.", processor_chain([
	dup43,
	dup94,
	dup13,
	dup14,
	dup15,
	setc("event_description","LiveUpdate is trying to connect to Server."),
]));

var msg256 = msg("LiveUpdate:37", part311);

var part312 = match("MESSAGE#177:LiveUpdate:38/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,A LiveUpdate session ran successfully. %{p0}");

var part313 = match("MESSAGE#177:LiveUpdate:38/1_0", "nwparser.p0", "%{fld23},Event time:%{fld17->} %{fld18}");

var part314 = match_copy("MESSAGE#177:LiveUpdate:38/1_1", "nwparser.p0", "fld23");

var select53 = linear_select([
	part313,
	part314,
]);

var all100 = all_match({
	processors: [
		part312,
		select53,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup93,
		dup15,
		setc("event_description","A LiveUpdate session from LiveUpdate Manager ran successfully."),
	]),
});

var msg257 = msg("LiveUpdate:38", all100);

var part315 = match("MESSAGE#178:LiveUpdate:39/0", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,[LiveUpdate error submission] Information submitted to Symantec.%{p0}");

var all101 = all_match({
	processors: [
		part315,
		dup318,
	],
	on_success: processor_chain([
		dup168,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","LiveUpdate error submission to Symantec."),
	]),
});

var msg258 = msg("LiveUpdate:39", all101);

var part316 = match("MESSAGE#180:LiveUpdate:41", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest Submission Control Thresholds update failed to load. The component has no valid content and will not function correctly until it is updated.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The latest Submission Control Thresholds update failed to load."),
]));

var msg259 = msg("LiveUpdate:41", part316);

var part317 = match("MESSAGE#181:LiveUpdate:42", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest SONAR Definitions update failed to load. The component has no valid content and will not function correctly until it is updated.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup190,
]));

var msg260 = msg("LiveUpdate:42", part317);

var part318 = match("MESSAGE#182:LiveUpdate:43", "nwparser.payload", "Category: %{fld11},LiveUpdate Manager,The latest Endpoint Detection and Response update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup190,
]));

var msg261 = msg("LiveUpdate:43", part318);

var part319 = match("MESSAGE#183:LiveUpdate:44", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,\"[LiveUpdate error submission] Submitting information to Symantec failed. Network error : '%{result}'%{fld23}\",Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup185,
]));

var msg262 = msg("LiveUpdate:44", part319);

var part320 = match("MESSAGE#184:LiveUpdate:45", "nwparser.payload", "LiveUpdate encountered an error.,Event time: %{fld17->} %{fld18}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	dup188,
	dup93,
]));

var msg263 = msg("LiveUpdate:45", part320);

var part321 = match("MESSAGE#185:LiveUpdate:46", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest AP Portal List update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","The latest AP Portal List update failed to load."),
]));

var msg264 = msg("LiveUpdate:46", part321);

var part322 = match("MESSAGE#186:LiveUpdate:47", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest Centralized Reputation Settings update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","The latest Centralized Reputation Settings update failed to load."),
]));

var msg265 = msg("LiveUpdate:47", part322);

var part323 = match("MESSAGE#187:LiveUpdate:48", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest Power Eraser Definitions update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","The latest Power Eraser Definitions update failed to load."),
]));

var msg266 = msg("LiveUpdate:48", part323);

var part324 = match("MESSAGE#188:LiveUpdate:49", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest Common Network Transport Library and Configuration update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","The latest Common Network Transport Library and Configuration update failed to load."),
]));

var msg267 = msg("LiveUpdate:49", part324);

var part325 = match("MESSAGE#189:LiveUpdate:50", "nwparser.payload", "Category: %{fld22},LiveUpdate Manager,The latest Extended File Attributes and Signatures update failed to load. The component has no valid content and will not function correctly until it is updated.,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","The latest Extended File Attributes and Signatures update failed to load."),
]));

var msg268 = msg("LiveUpdate:50", part325);

var select54 = linear_select([
	msg219,
	msg220,
	msg221,
	msg222,
	msg223,
	msg224,
	msg225,
	msg226,
	msg227,
	msg228,
	msg229,
	msg230,
	msg231,
	msg232,
	msg233,
	msg234,
	msg235,
	msg236,
	msg237,
	msg238,
	msg239,
	msg240,
	msg241,
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
	msg252,
	msg253,
	msg254,
	msg255,
	msg256,
	msg257,
	msg258,
	msg259,
	msg260,
	msg261,
	msg262,
	msg263,
	msg264,
	msg265,
	msg266,
	msg267,
	msg268,
]);

var part326 = match("MESSAGE#179:LiveUpdate:40/0", "nwparser.payload", "Virus and Spyware Definitions were updated recently, so the scheduled LiveUpdate was skipped.%{p0}");

var select55 = linear_select([
	dup191,
	dup91,
]);

var all102 = all_match({
	processors: [
		part326,
		select55,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","The scheduled LiveUpdate from LiveUpdate Manager was skipped."),
	]),
});

var msg269 = msg("LiveUpdate:40", all102);

var part327 = match("MESSAGE#430:Virus", "nwparser.payload", "Virus Found..Computer: %{shost}..Date: %{fld5}..Time: %{fld6}..Virus Name: %{virusname}..Path: %{filename}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup192,
	dup15,
	dup193,
]));

var msg270 = msg("Virus", part327);

var part328 = match("MESSAGE#431:Virus:01", "nwparser.payload", "Virus Found..Computer: %{shost}..Date: %{fld5}..Time: %{fld6}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup110,
	dup192,
	dup15,
	dup193,
]));

var msg271 = msg("Virus:01", part328);

var part329 = match("MESSAGE#432:Virus:02/0", "nwparser.payload", "Virus Definition File Update..%{fld4}..%{fld5}..Update to computer %{shost->} of virus definition file %{fld6->} failed. Status %{fld7->} ..%{p0}");

var part330 = match("MESSAGE#432:Virus:02/1_0", "nwparser.p0", ". %{p0}");

var select56 = linear_select([
	part330,
	dup194,
]);

var part331 = match("MESSAGE#432:Virus:02/2", "nwparser.p0", "%{severity}..%{product}..%{fld8}");

var all103 = all_match({
	processors: [
		part329,
		select56,
		part331,
	],
	on_success: processor_chain([
		dup43,
		dup44,
		dup45,
		dup30,
		dup25,
		date_time({
			dest: "event_time",
			args: ["fld5","fld8"],
			fmts: [
				[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO,dP],
			],
		}),
		dup15,
		dup195,
	]),
});

var msg272 = msg("Virus:02", all103);

var part332 = match("MESSAGE#433:Virus:03", "nwparser.payload", "Virus Definition File Update..%{shost}..%{fld5}..%{severity}..%{product}..%{fld6}", processor_chain([
	dup43,
	dup44,
	dup45,
	dup30,
	dup22,
	dup192,
	dup15,
	dup195,
]));

var msg273 = msg("Virus:03", part332);

var part333 = match("MESSAGE#434:Virus:09", "nwparser.payload", "Virus Found..%{shost}..%{fld5}..%{filename}.....%{info}..%{action}....%{severity}..%{product}..%{fld6}..%{username}..%{virusname}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup192,
	dup15,
	dup196,
]));

var msg274 = msg("Virus:09", part333);

var part334 = match("MESSAGE#435:Virus:04", "nwparser.payload", "Virus Found..%{fld12}..%{fld5}..%{filename}..%{info}..%{action}..%{severity}..%{product}..%{fld6}..%{username}..%{virusname}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup192,
	dup15,
	dup196,
]));

var msg275 = msg("Virus:04", part334);

var part335 = match("MESSAGE#436:Virus:12/2", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},0,Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size}");

var all104 = all_match({
	processors: [
		dup197,
		dup328,
		part335,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg276 = msg("Virus:12", all104);

var part336 = match("MESSAGE#437:Virus:15/0", "nwparser.payload", "Virus found,IP Address: %{saddr},Computer name: %{shost},%{p0}");

var part337 = match("MESSAGE#437:Virus:15/2", "nwparser.p0", "%{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{p0}");

var part338 = match("MESSAGE#437:Virus:15/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{url},Web domain: %{fld45},Downloaded by: %{filename},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size},Category set: %{category},Category type: %{event_type}");

var all105 = all_match({
	processors: [
		part336,
		dup329,
		part337,
		dup328,
		part338,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg277 = msg("Virus:15", all105);

var part339 = match("MESSAGE#438:Virus:13/2", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},%{p0}");

var all106 = all_match({
	processors: [
		dup197,
		dup328,
		part339,
		dup330,
		dup205,
		dup331,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg278 = msg("Virus:13", all106);

var part340 = match("MESSAGE#439:Virus:10/2", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31}");

var all107 = all_match({
	processors: [
		dup197,
		dup328,
		part340,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg279 = msg("Virus:10", all107);

var part341 = match("MESSAGE#440:Virus:14/1_0", "nwparser.p0", "\"%{fld22}\",Actual action: %{p0}");

var part342 = match("MESSAGE#440:Virus:14/1_1", "nwparser.p0", "%{fld22},Actual action: %{p0}");

var select57 = linear_select([
	part341,
	part342,
]);

var part343 = match("MESSAGE#440:Virus:14/2", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},First Seen: %{fld50},Sensitivity: %{fld58},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size}");

var all108 = all_match({
	processors: [
		dup208,
		select57,
		part343,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg280 = msg("Virus:14", all108);

var all109 = all_match({
	processors: [
		dup208,
		dup332,
		dup151,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg281 = msg("Virus:05", all109);

var part344 = match("MESSAGE#442:Virus:11/2", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},\"Group: %{group}\",Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all110 = all_match({
	processors: [
		dup208,
		dup332,
		part344,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup15,
		dup47,
		dup200,
	]),
});

var msg282 = msg("Virus:11", all110);

var part345 = match("MESSAGE#443:Virus:06/0", "nwparser.payload", "Virus Found..Computer: %{shost}..%{p0}");

var part346 = match("MESSAGE#443:Virus:06/1_0", "nwparser.p0", "Date: %{fld5}..File Path:%{p0}");

var part347 = match("MESSAGE#443:Virus:06/1_1", "nwparser.p0", "%{fld5}..File Path:%{p0}");

var select58 = linear_select([
	part346,
	part347,
]);

var part348 = match("MESSAGE#443:Virus:06/2", "nwparser.p0", "%{filename}..%{info}..Requested Action:%{action}..Severity:%{severity}..Source:%{product}..Time:%{fld6}..User:%{username}");

var all111 = all_match({
	processors: [
		part345,
		select58,
		part348,
	],
	on_success: processor_chain([
		dup110,
		dup115,
		dup116,
		dup38,
		dup192,
		dup15,
		dup196,
	]),
});

var msg283 = msg("Virus:06", all111);

var part349 = match("MESSAGE#444:Virus:07", "nwparser.payload", "%{fld1->} Virus Found %{shost->} %{fld5->} %{filename->} Forward from %{info->} %{action->} %{severity->} %{product->} Edition %{version->} %{virusname}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup15,
	dup132,
	dup196,
]));

var msg284 = msg("Virus:07", part349);

var part350 = match("MESSAGE#445:Virus:08", "nwparser.payload", "%{product->} definitions %{info}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Product successfully updated."),
]));

var msg285 = msg("Virus:08", part350);

var select59 = linear_select([
	msg269,
	msg270,
	msg271,
	msg272,
	msg273,
	msg274,
	msg275,
	msg276,
	msg277,
	msg278,
	msg279,
	msg280,
	msg281,
	msg282,
	msg283,
	msg284,
	msg285,
]);

var part351 = match("MESSAGE#216:Local:01", "nwparser.payload", "%{shost}, Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup53,
	dup12,
	dup15,
	dup40,
]));

var msg286 = msg("Local:01", part351);

var part352 = match("MESSAGE#217:Local:02", "nwparser.payload", "Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup15,
	dup40,
]));

var msg287 = msg("Local:02", part352);

var select60 = linear_select([
	msg286,
	msg287,
]);

var part353 = match("MESSAGE#218:Location/0", "nwparser.payload", "Location has been %{p0}");

var part354 = match("MESSAGE#218:Location/1_0", "nwparser.p0", "changed %{p0}");

var part355 = match("MESSAGE#218:Location/1_1", "nwparser.p0", "switched%{p0}");

var select61 = linear_select([
	part354,
	part355,
]);

var part356 = match("MESSAGE#218:Location/2", "nwparser.p0", "%{}to %{p0}");

var all112 = all_match({
	processors: [
		part353,
		select61,
		part356,
		dup333,
	],
	on_success: processor_chain([
		dup136,
		dup94,
		dup13,
		dup30,
		dup97,
		dup22,
		dup14,
		dup15,
		dup93,
		setc("event_description","Location has been changed or switched"),
	]),
});

var msg288 = msg("Location", all112);

var part357 = match_copy("MESSAGE#219:LUALL", "nwparser.payload", "event_description", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
]));

var msg289 = msg("LUALL", part357);

var part358 = match("MESSAGE#220:Management", "nwparser.payload", "Management server started up successfully%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Management server started up successfully."),
]));

var msg290 = msg("Management", part358);

var part359 = match("MESSAGE#221:Management:01", "nwparser.payload", "Management server shut down gracefully%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Management server shut down gracefully"),
]));

var msg291 = msg("Management:01", part359);

var part360 = match("MESSAGE#222:Management:02", "nwparser.payload", "Management Server has detected and ignored one or more duplicate entries.Please check the following entries in your directory server:%{fld12}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Management Server has detected and ignored one or more duplicate entries."),
]));

var msg292 = msg("Management:02", part360);

var select62 = linear_select([
	msg290,
	msg291,
	msg292,
]);

var part361 = match("MESSAGE#223:management", "nwparser.payload", "management server received the client log successfully,%{shost},%{username},%{group}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The management server received the client log successfully."),
]));

var msg293 = msg("management", part361);

var part362 = match("MESSAGE#224:management:01", "nwparser.payload", "management server received a report that the client computer changed its hardware identity,%{shost},%{username},%{group}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The management server received a report that the client computer changed its hardware identity."),
]));

var msg294 = msg("management:01", part362);

var select63 = linear_select([
	msg293,
	msg294,
]);

var part363 = match("MESSAGE#225:Network/0", "nwparser.payload", "Network Threat Protection --%{p0}");

var part364 = match("MESSAGE#225:Network/1_0", "nwparser.p0", "-- Engine version%{p0}");

var part365 = match("MESSAGE#225:Network/1_1", "nwparser.p0", " Engine version%{p0}");

var select64 = linear_select([
	part364,
	part365,
]);

var part366 = match("MESSAGE#225:Network/2", "nwparser.p0", "%{}: %{version->} Windows Version info: Operating System: %{os->} Network info:%{info}");

var all113 = all_match({
	processors: [
		part363,
		select64,
		part366,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","Network Threat Protection information."),
	]),
});

var msg295 = msg("Network", all113);

var part367 = match("MESSAGE#226:Network:01", "nwparser.payload", "Network Threat Protection has been activated%{}", processor_chain([
	dup213,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	setc("event_description","Network Threat Protection has been activated"),
]));

var msg296 = msg("Network:01", part367);

var part368 = match("MESSAGE#227:Network:02/0", "nwparser.payload", "Network Threat Protection applied a new IPS %{p0}");

var part369 = match("MESSAGE#227:Network:02/1_0", "nwparser.p0", "Library%{p0}");

var part370 = match("MESSAGE#227:Network:02/1_1", "nwparser.p0", "library%{p0}");

var select65 = linear_select([
	part369,
	part370,
]);

var part371 = match("MESSAGE#227:Network:02/2", "nwparser.p0", "%{}.");

var all114 = all_match({
	processors: [
		part368,
		select65,
		part371,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","Network Threat Protection applied a new IPS Library."),
	]),
});

var msg297 = msg("Network:02", all114);

var part372 = match("MESSAGE#228:Network:03", "nwparser.payload", "The Network Threat Protection already has the newest policy.%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The Network Threat Protection already has the newest policy."),
]));

var msg298 = msg("Network:03", part372);

var part373 = match("MESSAGE#229:Network:04", "nwparser.payload", "The Network Threat Protection is unable to download the newest policy from the Symantec Endpoint Protection Manager.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The Network Threat Protection is unable to download the newest policy from the Symantec Endpoint Protection Manager."),
]));

var msg299 = msg("Network:04", part373);

var part374 = match("MESSAGE#230:Network:05", "nwparser.payload", "Network Threat Protection's firewall and Intrusion Prevention features are disabled%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Threat Protection's firewall and Intrusion Prevention features are disabled"),
]));

var msg300 = msg("Network:05", part374);

var part375 = match("MESSAGE#231:Network:06", "nwparser.payload", "The Network Threat Protection is unable to communicate with the Symantec Endpoint Protection Manager.%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","The Network Threat Protection is unable to communicate with the Symantec Endpoint Protection Manager."),
]));

var msg301 = msg("Network:06", part375);

var part376 = match("MESSAGE#232:Network:07", "nwparser.payload", "Network Audit Search Unagented Hosts Started%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Audit Search Unagented Hosts Started."),
]));

var msg302 = msg("Network:07", part376);

var part377 = match("MESSAGE#233:Network:08", "nwparser.payload", "Network Audit Search Unagented Hosts From NST Finished Abnormally%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	dup214,
]));

var msg303 = msg("Network:08", part377);

var part378 = match("MESSAGE#234:Network:09", "nwparser.payload", "Network Audit Search Unagented Hosts From NST Finished Normally%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	dup214,
]));

var msg304 = msg("Network:09", part378);

var part379 = match("MESSAGE#235:Network:10", "nwparser.payload", "Network Audit Client Remote Pushing Install Started%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Audit Client Remote Pushing Install Started."),
]));

var msg305 = msg("Network:10", part379);

var part380 = match("MESSAGE#236:Network:11", "nwparser.payload", "Network Audit Client Remote Pushing Install Finished Normally%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Audit Client Remote Pushing Install Finished Normally."),
]));

var msg306 = msg("Network:11", part380);

var part381 = match("MESSAGE#237:Network:12", "nwparser.payload", "Network Intrusion Prevention is malfunctioning, %{result}\"", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	dup215,
]));

var msg307 = msg("Network:12", part381);

var part382 = match("MESSAGE#238:Network:13", "nwparser.payload", "Category: %{fld11},Network Intrusion Protection Sys,Browser Intrusion Prevention is malfunctioning. Browser type: %{obj_name}.Try to update the signatures Browser path: %{filename}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	setc("event_description","Browser Intrusion Prevention is malfunctioning."),
]));

var msg308 = msg("Network:13", part382);

var part383 = match("MESSAGE#241:Network:16", "nwparser.payload", "Network Intrusion Prevention and Browser Intrusion Prevention are malfunctioning because their content is not installed. The IPS content is going to be installed automatically%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	setc("event_description","Network Intrusion Prevention and Browser Intrusion Prevention are malfunctioning because their content is not installed."),
]));

var msg309 = msg("Network:16", part383);

var part384 = match("MESSAGE#242:Network:17", "nwparser.payload", "Network Intrusion Prevention is malfunctioning%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	dup215,
]));

var msg310 = msg("Network:17", part384);

var part385 = match("MESSAGE#243:Network:18", "nwparser.payload", "Network Intrusion Prevention is not protecting machine because its driver was unloaded%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Intrusion Prevention is not protecting machine because its driver was unloaded"),
]));

var msg311 = msg("Network:18", part385);

var part386 = match("MESSAGE#244:Network:19", "nwparser.payload", "Network Threat Protection's firewall is disabled by policy%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Threat Protection's firewall is disabled"),
]));

var msg312 = msg("Network:19", part386);

var part387 = match("MESSAGE#246:Network:21", "nwparser.payload", "%{service->} has been restored and %{result}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Intrusion Prevention has been restored"),
]));

var msg313 = msg("Network:21", part387);

var part388 = match("MESSAGE#247:Network:33", "nwparser.payload", "%{service->} is not protecting machine because its driver was disabled,Event time: %{event_time_string}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","Network Intrusion Prevention is not protecting machine because its driver was disabled"),
]));

var msg314 = msg("Network:33", part388);

var part389 = match("MESSAGE#251:Network:25/0", "nwparser.payload", "Network Threat Protection's firewall is enabled%{p0}");

var all115 = all_match({
	processors: [
		part389,
		dup318,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Network Threat Protection's firewall is enabled"),
	]),
});

var msg315 = msg("Network:25", all115);

var part390 = match("MESSAGE#253:Network:27/0", "nwparser.payload", "Network Intrusion Prevention disabled%{p0}");

var all116 = all_match({
	processors: [
		part390,
		dup334,
	],
	on_success: processor_chain([
		dup92,
		dup94,
		dup13,
		dup14,
		dup15,
		setc("event_description","Network Intrusion Prevention disabled"),
	]),
});

var msg316 = msg("Network:27", all116);

var part391 = match("MESSAGE#254:Network:28/0", "nwparser.payload", "Network Intrusion Prevention enabled%{p0}");

var all117 = all_match({
	processors: [
		part391,
		dup318,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Network Intrusion Prevention enabled"),
	]),
});

var msg317 = msg("Network:28", all117);

var part392 = match("MESSAGE#257:Network:30", "nwparser.payload", "Network Audit Client Remote Pushing Install Finished Abnormally in Pusing Stage%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Network Audit Client Remote Pushing Install Finished Abnormally in Pusing Stage"),
]));

var msg318 = msg("Network:30", part392);

var select66 = linear_select([
	msg295,
	msg296,
	msg297,
	msg298,
	msg299,
	msg300,
	msg301,
	msg302,
	msg303,
	msg304,
	msg305,
	msg306,
	msg307,
	msg308,
	msg309,
	msg310,
	msg311,
	msg312,
	msg313,
	msg314,
	msg315,
	msg316,
	msg317,
	msg318,
]);

var part393 = match("MESSAGE#239:Network:14", "nwparser.payload", "Firefox Browser Intrusion Prevention is malfunctioning%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	setc("event_description","Firefox Browser Intrusion Prevention is malfunctioning."),
]));

var msg319 = msg("Network:14", part393);

var part394 = match("MESSAGE#245:Network:20/0", "nwparser.payload", "Firefox Browser Intrusion Prevention disabled%{p0}");

var all118 = all_match({
	processors: [
		part394,
		dup334,
	],
	on_success: processor_chain([
		dup92,
		dup94,
		dup13,
		dup14,
		dup15,
		setc("event_description","Firefox Browser Intrusion Prevention disabled"),
	]),
});

var msg320 = msg("Network:20", all118);

var part395 = match("MESSAGE#252:Network:26/0", "nwparser.payload", "Firefox Browser Intrusion Prevention enabled%{p0}");

var all119 = all_match({
	processors: [
		part395,
		dup318,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Firefox Browser Intrusion Prevention enabled"),
	]),
});

var msg321 = msg("Network:26", all119);

var select67 = linear_select([
	msg319,
	msg320,
	msg321,
]);

var part396 = match("MESSAGE#240:Network:15", "nwparser.payload", "Internet Explorer Browser Intrusion Prevention is malfunctioning%{}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup187,
	dup14,
	dup15,
	setc("event_description","Internet Explorer Browser Intrusion Prevention is malfunctioning."),
]));

var msg322 = msg("Network:15", part396);

var part397 = match("MESSAGE#248:Network:22/0", "nwparser.payload", "Internet Explorer Browser Intrusion Prevention enabled%{p0}");

var all120 = all_match({
	processors: [
		part397,
		dup318,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Internet Explorer Browser Intrusion Prevention enabled"),
	]),
});

var msg323 = msg("Network:22", all120);

var part398 = match("MESSAGE#249:Network:23/0", "nwparser.payload", "Internet Explorer Browser Intrusion Prevention disabled%{p0}");

var all121 = all_match({
	processors: [
		part398,
		dup334,
	],
	on_success: processor_chain([
		dup92,
		dup94,
		dup13,
		dup14,
		dup15,
		setc("event_description","Internet Explorer Browser Intrusion Prevention disabled"),
	]),
});

var msg324 = msg("Network:23", all121);

var select68 = linear_select([
	msg322,
	msg323,
	msg324,
]);

var part399 = match("MESSAGE#255:Network:29/0", "nwparser.payload", "Generic Exploit Mitigation %{p0}");

var part400 = match("MESSAGE#255:Network:29/1_0", "nwparser.p0", "enabled%{p0}");

var part401 = match("MESSAGE#255:Network:29/1_1", "nwparser.p0", "disabled%{p0}");

var part402 = match("MESSAGE#255:Network:29/1_2", "nwparser.p0", "is malfunctioning%{p0}");

var select69 = linear_select([
	part400,
	part401,
	part402,
]);

var part403 = match("MESSAGE#255:Network:29/2", "nwparser.p0", ",Event time: %{event_time_string}");

var all122 = all_match({
	processors: [
		part399,
		select69,
		part403,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup217,
	]),
});

var msg325 = msg("Network:29", all122);

var part404 = match("MESSAGE#256:Network:31", "nwparser.payload", "Category: %{fld22},Generic Exploit Mitigation Syste,Already running process (PID:%{process_id}) '%{process}' is affected by a change to the application rules.,Event time: %{event_time_string}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	dup217,
]));

var msg326 = msg("Network:31", part404);

var select70 = linear_select([
	msg325,
	msg326,
]);

var part405 = match("MESSAGE#258:Network:32", "nwparser.payload", "%{event_description},Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg327 = msg("Network:32", part405);

var part406 = match("MESSAGE#259:New/0", "nwparser.payload", "New virus definition file loaded. Version: %{p0}");

var part407 = match("MESSAGE#259:New/1_0", "nwparser.p0", "%{version},Event time:%{fld17->} %{fld18}");

var part408 = match_copy("MESSAGE#259:New/1_1", "nwparser.p0", "version");

var select71 = linear_select([
	part407,
	part408,
]);

var all123 = all_match({
	processors: [
		part406,
		select71,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup44,
		dup45,
		dup30,
		dup22,
		dup14,
		dup15,
		dup93,
		setc("event_description","New virus definition file loaded."),
	]),
});

var msg328 = msg("New", all123);

var part409 = match("MESSAGE#260:New:01/0", "nwparser.payload", "New Value '%{change_attribute}' = '%{change_new}'%{p0}");

var all124 = all_match({
	processors: [
		part409,
		dup318,
	],
	on_success: processor_chain([
		dup95,
		dup12,
		dup13,
		dup30,
		dup97,
		dup22,
		dup14,
		dup137,
		dup15,
		dup93,
		setc("event_description","New value"),
	]),
});

var msg329 = msg("New:01", all124);

var part410 = match("MESSAGE#261:New:02", "nwparser.payload", "New AgentGUID = %{fld22}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup137,
	dup15,
	setc("event_description","New AgentGUID"),
]));

var msg330 = msg("New:02", part410);

var part411 = match("MESSAGE#262:New:03", "nwparser.payload", "New policy has been imported.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup137,
	dup15,
	setc("event_description","New policy has been imported."),
]));

var msg331 = msg("New:03", part411);

var part412 = match("MESSAGE#263:New:04/0", "nwparser.payload", "New content update failed to download from the management server. Remote file path: %{p0}");

var part413 = match("MESSAGE#263:New:04/1_0", "nwparser.p0", "%{url},Event time: %{event_time_string}");

var select72 = linear_select([
	part413,
	dup64,
]);

var all125 = all_match({
	processors: [
		part412,
		select72,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup137,
		dup15,
		setc("event_description","New content update failed to download from the management server"),
	]),
});

var msg332 = msg("New:04", all125);

var part414 = match("MESSAGE#264:New:05", "nwparser.payload", "New content update failed to download from Group Update Provider. Remote file path: %{url}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup137,
	dup15,
	setc("event_description","New content update failed to download from Group Update Provider"),
]));

var msg333 = msg("New:05", part414);

var select73 = linear_select([
	msg328,
	msg329,
	msg330,
	msg331,
	msg332,
	msg333,
]);

var part415 = match("MESSAGE#265:No", "nwparser.payload", "No %{virusname->} virus found events got swept.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup152,
	dup14,
	dup15,
	setc("event_description","No virus found events got swept."),
]));

var msg334 = msg("No", part415);

var part416 = match("MESSAGE#266:No:01", "nwparser.payload", "No clients got swept.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","No clients got swept."),
]));

var msg335 = msg("No:01", part416);

var part417 = match("MESSAGE#267:No:02", "nwparser.payload", "No objects got swept.%{}", processor_chain([
	dup43,
	dup15,
	dup218,
]));

var msg336 = msg("No:02", part417);

var part418 = match("MESSAGE#268:No:06", "nwparser.payload", "No clients got swept [Domain: %{sdomain}].", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	dup218,
]));

var msg337 = msg("No:06", part418);

var part419 = match("MESSAGE#269:No:03", "nwparser.payload", "No old risk events got swept.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","No old risk events got swept."),
]));

var msg338 = msg("No:03", part419);

var part420 = match("MESSAGE#270:No:04", "nwparser.payload", "No physical files got swept.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","No physical files got swept."),
]));

var msg339 = msg("No:04", part420);

var part421 = match("MESSAGE#271:No:05", "nwparser.payload", "No risk events from deleted clients got swept.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","No risk events from deleted clients got swept."),
]));

var msg340 = msg("No:05", part421);

var part422 = match("MESSAGE#272:No:07", "nwparser.payload", "No updates found for %{application}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","No updates found."),
]));

var msg341 = msg("No:07", part422);

var select74 = linear_select([
	msg334,
	msg335,
	msg336,
	msg337,
	msg338,
	msg339,
	msg340,
	msg341,
]);

var part423 = match("MESSAGE#273:Organization:03", "nwparser.payload", "Organization Unit or Container importing finished successfully%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Organization Unit or Container importing finished successfully"),
]));

var msg342 = msg("Organization:03", part423);

var part424 = match("MESSAGE#274:Organization:02", "nwparser.payload", "Organization Unit or Container importing started%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Organization Unit or Container importing started."),
]));

var msg343 = msg("Organization:02", part424);

var part425 = match("MESSAGE#275:Organization:01", "nwparser.payload", "Organization importing finished successfully%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	dup219,
]));

var msg344 = msg("Organization:01", part425);

var part426 = match("MESSAGE#276:Organization", "nwparser.payload", "Organization importing started%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	dup220,
]));

var msg345 = msg("Organization", part426);

var select75 = linear_select([
	msg342,
	msg343,
	msg344,
	msg345,
]);

var part427 = match("MESSAGE#277:Number:01", "nwparser.payload", "Number of %{virusname->} virus found events swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup152,
	dup15,
	setc("event_description","Number of virus found events swept."),
	setc("dclass_counter1_string","Virus found events swept count."),
]));

var msg346 = msg("Number:01", part427);

var part428 = match("MESSAGE#278:Number", "nwparser.payload", "Number of virus definition records swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Number of virus definition records swept."),
	setc("dclass_counter1_string","Virus definition records swept."),
]));

var msg347 = msg("Number", part428);

var part429 = match("MESSAGE#279:Number:02", "nwparser.payload", "Number of scan events swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup15,
	setc("event_description","Number of scan events swept."),
	setc("dclass_counter1_string","scan events swept"),
]));

var msg348 = msg("Number:02", part429);

var part430 = match("MESSAGE#280:Number:04", "nwparser.payload", "Number of clients swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of clients swept."),
	setc("dclass_counter1_string","clients swept"),
]));

var msg349 = msg("Number:04", part430);

var part431 = match("MESSAGE#281:Number:05", "nwparser.payload", "Number of old risk events swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of old risk events swept."),
	setc("dclass_counter1_string","old risk events swept"),
]));

var msg350 = msg("Number:05", part431);

var part432 = match("MESSAGE#282:Number:06", "nwparser.payload", "Number of unacknowledged notifications swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of unacknowledged notification swept."),
	setc("dclass_counter1_string","unacknowledged notifications swept"),
]));

var msg351 = msg("Number:06", part432);

var part433 = match("MESSAGE#283:Number:07", "nwparser.payload", "Number of objects swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of objects swept."),
	setc("dclass_counter1_string","Number of objects swept"),
]));

var msg352 = msg("Number:07", part433);

var part434 = match("MESSAGE#284:Number:08", "nwparser.payload", "Number of risk events from deleted clients swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of risk events swept."),
	setc("dclass_counter1_string","Deleted clients swept"),
]));

var msg353 = msg("Number:08", part434);

var part435 = match("MESSAGE#285:Number:09", "nwparser.payload", "Number of old risk events compressed: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of old risk events compressed."),
	setc("dclass_counter1_string","old risk events compressed"),
]));

var msg354 = msg("Number:09", part435);

var part436 = match("MESSAGE#286:Number:10", "nwparser.payload", "Number of compressed risk events swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Number of compressed risk events swept."),
	setc("dclass_counter1_string","compressed risk events swept"),
]));

var msg355 = msg("Number:10", part436);

var part437 = match("MESSAGE#287:Number:11/0", "nwparser.payload", "Number of %{info->} in the policy: %{p0}");

var part438 = match("MESSAGE#287:Number:11/1_0", "nwparser.p0", "%{dclass_counter1},Event time:%{fld17->} %{fld18}");

var part439 = match_copy("MESSAGE#287:Number:11/1_1", "nwparser.p0", "dclass_counter1");

var select76 = linear_select([
	part438,
	part439,
]);

var all126 = all_match({
	processors: [
		part437,
		select76,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup221,
		setc("dclass_counter1_string","Group Update Providers"),
		dup93,
	]),
});

var msg356 = msg("Number:11", all126);

var part440 = match("MESSAGE#288:Number:12", "nwparser.payload", "Number of physical files swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	dup221,
	setc("dclass_counter1_string","Number of physical files swept"),
]));

var msg357 = msg("Number:12", part440);

var part441 = match("MESSAGE#289:Number:13", "nwparser.payload", "Number of %{fld1->} swept: %{dclass_counter1}", processor_chain([
	dup43,
	dup15,
	dup12,
	dup222,
	setc("a","Number of "),
	call({
		dest: "nwparser.event_description",
		fn: STRCAT,
		args: [
			constant("a"),
			field("fld1"),
			constant("\t"),
			field("swept."),
		],
	}),
	call({
		dest: "nwparser.dclass_counter1_string",
		fn: STRCAT,
		args: [
			field("fld1"),
			constant("\t"),
			field("swept"),
		],
	}),
]));

var msg358 = msg("Number:13", part441);

var select77 = linear_select([
	msg346,
	msg347,
	msg348,
	msg349,
	msg350,
	msg351,
	msg352,
	msg353,
	msg354,
	msg355,
	msg356,
	msg357,
	msg358,
]);

var part442 = match("MESSAGE#292:Policy:added", "nwparser.payload", "Policy has been added,%{info}", processor_chain([
	dup95,
	dup12,
	dup13,
	dup96,
	dup223,
	dup22,
	dup14,
	dup15,
	dup23,
	dup224,
]));

var msg359 = msg("Policy:added", part442);

var part443 = match("MESSAGE#293:Policy:added_01", "nwparser.payload", "Policy has been added:%{info}", processor_chain([
	dup95,
	dup12,
	dup13,
	dup96,
	dup223,
	dup22,
	dup14,
	dup15,
	dup23,
	dup224,
]));

var msg360 = msg("Policy:added_01", part443);

var part444 = match("MESSAGE#294:Policy:edited", "nwparser.payload", "Policy has been edited,%{info}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup223,
	dup22,
	dup14,
	dup15,
	dup23,
	dup225,
]));

var msg361 = msg("Policy:edited", part444);

var part445 = match("MESSAGE#295:Policy:edited_01", "nwparser.payload", "Policy has been edited:%{info},%{fld1}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup223,
	dup22,
	dup14,
	dup15,
	dup23,
	dup225,
]));

var msg362 = msg("Policy:edited_01", part445);

var part446 = match("MESSAGE#296:Policy:deleted/0", "nwparser.payload", "Policy has been deleted%{p0}");

var select78 = linear_select([
	dup226,
	dup71,
]);

var all127 = all_match({
	processors: [
		part446,
		select78,
		dup212,
	],
	on_success: processor_chain([
		dup156,
		dup12,
		dup13,
		dup27,
		dup223,
		dup22,
		dup14,
		dup15,
		dup23,
		setc("event_description","Policy has been deleted"),
	]),
});

var msg363 = msg("Policy:deleted", all127);

var select79 = linear_select([
	msg359,
	msg360,
	msg361,
	msg362,
	msg363,
]);

var part447 = match("MESSAGE#297:Potential:03", "nwparser.payload", "Potential risk found,IP Address: %{saddr},Computer name: %{shost},Intensive Protection Level: %{fld61},Certificate issuer: %{fld60},Certificate signer: %{fld62},Certificate thumbprint: %{fld63},Signing timestamp: %{fld64},Certificate serial number: %{fld65},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld1},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Last update time: %{fld53},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld100},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size},Category set: %{category},Category type: %{vendor_event_cat},Location:%{fld55}", processor_chain([
	dup110,
	dup12,
	dup152,
	dup132,
	dup93,
	dup153,
	dup154,
	dup227,
	dup15,
	dup19,
]));

var msg364 = msg("Potential:03", part447);

var part448 = match("MESSAGE#298:Potential:02/2", "nwparser.p0", "%{severity},First Seen:%{fld55},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},COH Engine Version: %{fld41},Detection Submissions No,Permitted application reason: %{fld42},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},Risk Level: %{fld50},Detection Source: %{fld52},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{p0}");

var part449 = match("MESSAGE#298:Potential:02/4", "nwparser.p0", "%{fld1},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all128 = all_match({
	processors: [
		dup228,
		dup325,
		part448,
		dup327,
		part449,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup132,
		dup93,
		dup153,
		dup154,
		dup227,
		dup15,
		dup19,
	]),
});

var msg365 = msg("Potential:02", all128);

var part450 = match("MESSAGE#299:Potential/2", "nwparser.p0", "%{fld23},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var all129 = all_match({
	processors: [
		dup228,
		dup325,
		part450,
		dup326,
		dup229,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup132,
		dup230,
		dup154,
		dup15,
		dup227,
		dup19,
	]),
});

var msg366 = msg("Potential", all129);

var part451 = match("MESSAGE#300:Potential:01/0", "nwparser.payload", "Potential risk found,Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var all130 = all_match({
	processors: [
		part451,
		dup326,
		dup229,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup93,
		dup230,
		dup154,
		dup132,
		dup15,
		dup227,
		dup19,
	]),
});

var msg367 = msg("Potential:01", all130);

var select80 = linear_select([
	msg364,
	msg365,
	msg366,
	msg367,
]);

var part452 = match("MESSAGE#301:Previous", "nwparser.payload", "Previous virus definition file loaded. Version: %{version}", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Previous virus definition file loaded."),
]));

var msg368 = msg("Previous", part452);

var part453 = match("MESSAGE#302:Proactive", "nwparser.payload", "Proactive Threat Scan %{info->} failed to update.", processor_chain([
	setc("eventcategory","1703020000"),
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Proactive Threat Scan failed to update."),
]));

var msg369 = msg("Proactive", part453);

var part454 = match("MESSAGE#303:Proactive:01", "nwparser.payload", "Proactive Threat Scan whitelist %{info->} is up-to-date.", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Proactive Threat Scan whitelist is up-to-date."),
]));

var msg370 = msg("Proactive:01", part454);

var part455 = match("MESSAGE#399:Symantec:38/0", "nwparser.payload", "Proactive Threat Protection has been enabled%{p0}");

var all131 = all_match({
	processors: [
		part455,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","Proactive Threat Protection has been enabled"),
	]),
});

var msg371 = msg("Symantec:38", all131);

var part456 = match("MESSAGE#400:Symantec:42", "nwparser.payload", "Proactive Threat Protection has been disabled%{}", processor_chain([
	dup43,
	dup56,
	dup12,
	dup13,
	dup15,
	dup57,
]));

var msg372 = msg("Symantec:42", part456);

var select81 = linear_select([
	msg369,
	msg370,
	msg371,
	msg372,
]);

var part457 = match("MESSAGE#304:process", "nwparser.payload", "process %{process->} can not lock the process status table. The process status has been locked by the server %{info->} since %{fld50}.", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Cannot lock process status table since it has been locked by server."),
]));

var msg373 = msg("process", part457);

var part458 = match("MESSAGE#305:process:01", "nwparser.payload", "\"Application has changed since the last time you opened it, process id: %{process_id->} Filename: %{filename->} The change was allowed by profile.\",Local: %{saddr},Local: %{fld1},Remote: %{fld25},Remote: %{daddr},Remote: %{fld3},Outbound,%{protocol},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld12}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup40,
	dup41,
	dup42,
	dup15,
	dup142,
	dup19,
	dup35,
]));

var msg374 = msg("process:01", part458);

var part459 = match("MESSAGE#306:process:11", "nwparser.payload", "\"Application has changed since the last time you opened it, process id: %{process_id->} Filename: %{filename->} The change was allowed by profile.\",Local: %{daddr},Local: %{fld1},Remote: %{fld25},Remote: %{saddr},Remote: %{fld3},Inbound,%{protocol},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld12}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup41,
	dup42,
	dup15,
	dup142,
	dup19,
	dup34,
	dup40,
]));

var msg375 = msg("process:11", part459);

var part460 = match("MESSAGE#308:process:03/2", "nwparser.p0", ",Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{protocol},%{p0}");

var part461 = match("MESSAGE#308:process:03/4", "nwparser.p0", "%{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var all132 = all_match({
	processors: [
		dup231,
		dup316,
		part460,
		dup317,
		part461,
		dup315,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup19,
		dup35,
		dup40,
	]),
});

var msg376 = msg("process:03", all132);

var all133 = all_match({
	processors: [
		dup231,
		dup316,
		dup78,
		dup317,
		dup81,
		dup315,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup16,
		dup17,
		dup15,
		dup19,
		dup34,
		dup40,
	]),
});

var msg377 = msg("process:13", all133);

var select82 = linear_select([
	msg373,
	msg374,
	msg375,
	msg376,
	msg377,
]);

var part462 = match("MESSAGE#310:properties/0", "nwparser.payload", "properties of domain %{p0}");

var part463 = match("MESSAGE#310:properties/1_0", "nwparser.p0", "\"%{domain}\"%{p0}");

var part464 = match("MESSAGE#310:properties/1_1", "nwparser.p0", "'%{domain}'%{p0}");

var select83 = linear_select([
	part463,
	part464,
]);

var part465 = match("MESSAGE#310:properties/2", "nwparser.p0", "%{}were changed");

var all134 = all_match({
	processors: [
		part462,
		select83,
		part465,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		dup30,
		dup97,
		dup22,
		dup14,
		dup15,
		setc("event_description","The properties of domain were changed"),
	]),
});

var msg378 = msg("properties", all134);

var part466 = match("MESSAGE#311:properties:01/0", "nwparser.payload", "properties for system administrator %{p0}");

var part467 = match("MESSAGE#311:properties:01/1_0", "nwparser.p0", "\"%{c_username}\"%{p0}");

var part468 = match("MESSAGE#311:properties:01/1_1", "nwparser.p0", "'%{c_username}'%{p0}");

var select84 = linear_select([
	part467,
	part468,
]);

var part469 = match("MESSAGE#311:properties:01/2", "nwparser.p0", "%{}have been changed");

var all135 = all_match({
	processors: [
		part466,
		select84,
		part469,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		dup30,
		dup97,
		dup22,
		dup14,
		dup15,
		setc("event_description","The properties of system administrator have been changed"),
	]),
});

var msg379 = msg("properties:01", all135);

var select85 = linear_select([
	msg378,
	msg379,
]);

var part470 = match("MESSAGE#312:PTS", "nwparser.payload", "PTS has generated an error: code %{resultcode}: description: %{info}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","PTS has generated an error"),
]));

var msg380 = msg("PTS", part470);

var part471 = match("MESSAGE#313:Received/0", "nwparser.payload", "Received a new policy with %{p0}");

var part472 = match("MESSAGE#313:Received/1_0", "nwparser.p0", "%{info},Event time: %{fld17->} %{fld18}");

var select86 = linear_select([
	part472,
	dup212,
]);

var all136 = all_match({
	processors: [
		part471,
		select86,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Received a new policy."),
	]),
});

var msg381 = msg("Received", all136);

var part473 = match("MESSAGE#699:Smc:03", "nwparser.payload", "Received a new profile with serial number %{fld23->} from Symantec Endpoint Protection Manager.", processor_chain([
	dup53,
	dup94,
	dup13,
	dup14,
	dup15,
	setc("event_description","Received a new profile from Symantec Endpoint Protection Manager."),
]));

var msg382 = msg("Smc:03", part473);

var select87 = linear_select([
	msg381,
	msg382,
]);

var part474 = match("MESSAGE#314:Reconfiguring", "nwparser.payload", "Reconfiguring Symantec Management Client....%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	setc("event_description","Reconfiguring Symantec Management Client."),
]));

var msg383 = msg("Reconfiguring", part474);

var part475 = match("MESSAGE#315:Reconnected/0", "nwparser.payload", "Reconnected to server after server was unreacheable.%{p0}");

var all137 = all_match({
	processors: [
		part475,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Reconnected to server after server was unreachable."),
	]),
});

var msg384 = msg("Reconnected", all137);

var part476 = match("MESSAGE#316:restart/0", "nwparser.payload", "Please restart your computer to enable %{info->} changes.%{p0}");

var all138 = all_match({
	processors: [
		part476,
		dup318,
	],
	on_success: processor_chain([
		dup232,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Please restart your computer to enable changes."),
	]),
});

var msg385 = msg("restart", all138);

var part477 = match("MESSAGE#317:Retry", "nwparser.payload", "Retry %{info}\"", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup233,
]));

var msg386 = msg("Retry", part477);

var part478 = match("MESSAGE#318:Retry:01", "nwparser.payload", "Retry timestamp is equal or over the next schedule time, switching to regular schedule run.%{}", processor_chain([
	dup43,
	dup15,
	setc("action","Retry timestamp is equal or over the next schedule time, switching to regular schedule run."),
]));

var msg387 = msg("Retry:01", part478);

var part479 = match("MESSAGE#319:Retry:02", "nwparser.payload", "Retry timestamp is over the maximum retry window, switching to regular schedule run.%{}", processor_chain([
	dup43,
	dup233,
	dup15,
]));

var msg388 = msg("Retry:02", part479);

var select88 = linear_select([
	msg386,
	msg387,
	msg388,
]);

var part480 = match("MESSAGE#320:Successfully", "nwparser.payload", "Successfully downloaded the %{application->} security definitions from LiveUpdate. The security definitions are now available for deployment.", processor_chain([
	dup43,
	setc("event_description","Successfully Downloaded."),
	dup15,
]));

var msg389 = msg("Successfully", part480);

var part481 = match("MESSAGE#321:Successfully:01", "nwparser.payload", "Successfully deleted the client install package '%{info}'.", processor_chain([
	dup43,
	dup234,
	dup15,
]));

var msg390 = msg("Successfully:01", part481);

var part482 = match("MESSAGE#322:Successfully:02", "nwparser.payload", "Successfully imported the Symantec Endpoint Protection version %{version->} for %{fld3->} package during the server upgrade. This package is now available for deployment.", processor_chain([
	dup43,
	dup234,
	dup15,
]));

var msg391 = msg("Successfully:02", part482);

var select89 = linear_select([
	msg389,
	msg390,
	msg391,
]);

var part483 = match("MESSAGE#323:Risk:01", "nwparser.payload", "Risk Repair Failed..Computer: %{shost}..Date: %{fld5}..Time: %{fld6->} %{fld7->} ..Severity: %{severity}..Source: %{product}", processor_chain([
	dup110,
	dup166,
	dup15,
	dup235,
]));

var msg392 = msg("Risk:01", part483);

var part484 = match("MESSAGE#324:Risk:02", "nwparser.payload", "Risk Repair Failed..%{shost}..%{fld5}..%{filename}..%{info}..%{action}..%{severity}..%{product}..%{fld6->} %{fld7}..%{username}..%{virusname}", processor_chain([
	dup110,
	dup152,
	dup166,
	dup15,
	dup235,
]));

var msg393 = msg("Risk:02", part484);

var part485 = match("MESSAGE#325:Risk:03", "nwparser.payload", "Risk Repaired..Computer: %{shost}..Date: %{fld5}..Time: %{fld6->} %{fld7}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup110,
	dup166,
	dup15,
	dup236,
]));

var msg394 = msg("Risk:03", part485);

var part486 = match("MESSAGE#326:Risk:04", "nwparser.payload", "Risk Repaired..%{shost}..%{fld5}..%{filename}..%{info}..%{action}..%{severity}..%{product}..%{fld6->} %{fld7}..%{username}..%{virusname}", processor_chain([
	dup110,
	dup152,
	dup166,
	dup15,
	dup236,
]));

var msg395 = msg("Risk:04", part486);

var part487 = match("MESSAGE#327:Risk:05/0", "nwparser.payload", "Risk sample submitted to Symantec,Computer name: %{p0}");

var part488 = match("MESSAGE#327:Risk:05/2", "nwparser.p0", "%{event_type},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var part489 = match("MESSAGE#327:Risk:05/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld16->} %{fld17},Inserted: %{fld20},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all139 = all_match({
	processors: [
		part487,
		dup325,
		part488,
		dup326,
		part489,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup132,
		date_time({
			dest: "event_time",
			args: ["fld16","fld17"],
			fmts: [
				[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
			],
		}),
		dup230,
		dup154,
		dup15,
		dup19,
		setc("event_description","Risk sample submitted to Symantec."),
	]),
});

var msg396 = msg("Risk:05", all139);

var select90 = linear_select([
	msg392,
	msg393,
	msg394,
	msg395,
	msg396,
]);

var part490 = match("MESSAGE#328:Scan", "nwparser.payload", "Scan Start/Stop..%{shost}..%{fld5}..%{filename}..%{info}..%{fld22}..%{severity}..%{product}..%{fld6}..%{username}..%{virusname}", processor_chain([
	dup43,
	dup152,
	dup166,
	dup15,
	dup237,
]));

var msg397 = msg("Scan", part490);

var part491 = match("MESSAGE#329:Scan:01", "nwparser.payload", "Scan Start/Stop..%{shost}..%{fld5}..%{info}..%{severity}..%{product}..%{fld6}..%{username}", processor_chain([
	dup43,
	dup166,
	dup15,
	dup237,
]));

var msg398 = msg("Scan:01", part491);

var part492 = match("MESSAGE#330:Scan:02", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld50->} %{fld52},End: %{fld51},%{disposition},Duration (seconds): %{duration_string},User1: %{username},User2: %{fld3},\"%{info}\",\"%{context}\",Command: Not a command scan (),Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{dclass_counter2},Omitted: %{fld4}Computer: %{shost},IP Address: %{saddr}Domain: %{domain}Group: %{group},Server: %{hostid}", processor_chain([
	dup43,
	dup12,
	dup14,
	dup238,
	dup41,
	dup15,
	dup239,
	dup240,
	dup241,
]));

var msg399 = msg("Scan:02", part492);

var part493 = match("MESSAGE#331:Scan:09", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld1},End: %{fld2},%{disposition},Duration (seconds): %{duration_string},User1: %{username},User2: %{fld3},%{fld22},,Command: %{fld4},Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{fld5},Omitted: %{fld21},Computer: %{shost},IP Address: %{saddr},\"Group: %{group},Server: %{hostid}", processor_chain([
	dup43,
	dup12,
	dup14,
	dup242,
	dup15,
	dup243,
	dup244,
	dup245,
	dup246,
]));

var msg400 = msg("Scan:09", part493);

var part494 = match("MESSAGE#332:Scan:03/0", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld50->} %{fld52},End: %{fld51},%{disposition},Duration (seconds): %{duration_string},User1: %{username},User2: %{fld22},%{info},Command: Not a command scan (),Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{dclass_counter2},Omitted: %{fld21}Computer: %{shost},IP Address: %{saddr},Domain: %{domain},%{p0}");

var part495 = match_copy("MESSAGE#332:Scan:03/2", "nwparser.p0", "hostid");

var all140 = all_match({
	processors: [
		part494,
		dup330,
		part495,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup14,
		dup41,
		dup15,
		dup243,
		setc("dclass_counter1_string","Infected Count"),
		setc("dclass_counter2_string","Total File Count"),
		setc("dclass_counter3_string","Total Threat Count"),
	]),
});

var msg401 = msg("Scan:03", all140);

var part496 = match("MESSAGE#333:Scan:08", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld1},End: %{fld2},%{disposition},Duration (seconds): %{duration_string},User1: %{username},User2: %{fld3},Files scanned: %{dclass_counter2},,Command: %{fld4},Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{fld5},Omitted: %{fld21},Computer: %{shost},IP Address: %{saddr},Domain: %{domain},Group: %{group},Server: %{hostid}", processor_chain([
	dup43,
	dup12,
	dup14,
	dup242,
	dup15,
	dup243,
	dup244,
	dup245,
	dup246,
]));

var msg402 = msg("Scan:08", part496);

var part497 = match("MESSAGE#334:Scan:04/0", "nwparser.payload", "Scan Delayed: Risks: %{dclass_counter1->} Scanned: %{dclass_counter2->} Files/Folders/Drives Omitted: %{p0}");

var part498 = match("MESSAGE#334:Scan:04/1_0", "nwparser.p0", "%{dclass_counter3->} Trusted Files Skipped: %{fld1}");

var part499 = match_copy("MESSAGE#334:Scan:04/1_1", "nwparser.p0", "dclass_counter3");

var select91 = linear_select([
	part498,
	part499,
]);

var all141 = all_match({
	processors: [
		part497,
		select91,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		dup30,
		dup97,
		dup14,
		dup15,
		setc("event_description","Scan Delayed."),
		dup247,
		dup248,
		setc("dclass_counter3_string","Omitted Count."),
	]),
});

var msg403 = msg("Scan:04", all141);

var part500 = match("MESSAGE#335:Scan:05", "nwparser.payload", "%{action}..Computer: %{shost}..Date: %{fld5}..Description: %{event_description}: Risks: %{dclass_counter1->} Scanned: %{dclass_counter2->} Files/Folders/Drives Omitted: %{dclass_counter3}..Time: %{fld6->} %{fld4}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup43,
	dup166,
	dup15,
	dup247,
	dup248,
	setc("dclass_counter3_string","Ommitted count."),
]));

var msg404 = msg("Scan:05", part500);

var part501 = match("MESSAGE#336:Scan:06", "nwparser.payload", "%{action}..Computer: %{shost}..Date: %{fld5}..Description: %{event_description}...Time: %{fld6->} %{fld4}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup43,
	dup166,
	dup15,
]));

var msg405 = msg("Scan:06", part501);

var part502 = match("MESSAGE#337:Scan:07", "nwparser.payload", "Scan started on all drives and all extensions.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Scan started on all drives and all extensions."),
]));

var msg406 = msg("Scan:07", part502);

var part503 = match("MESSAGE#338:Scan:11", "nwparser.payload", "Scan Suspended: %{info}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Scan Suspended."),
]));

var msg407 = msg("Scan:11", part503);

var part504 = match("MESSAGE#339:Scan:10", "nwparser.payload", "Scan resumed on all drives and all extensions.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Scan resumed on all drives and all extensions."),
]));

var msg408 = msg("Scan:10", part504);

var part505 = match("MESSAGE#340:Scan:12/0", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld50->} %{fld52},End: %{fld51},%{disposition},Duration (seconds): %{duration_string},User1: %{uid},User2: %{fld3},'%{info}',%{p0}");

var part506 = match("MESSAGE#340:Scan:12/2", "nwparser.p0", "Command: Update Content and Scan Active,Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{dclass_counter2},Omitted: %{fld4}Computer: %{shost},IP Address: %{saddr},Domain: %{domain},Group: %{group},Server: %{hostid}");

var all142 = all_match({
	processors: [
		part505,
		dup335,
		part506,
	],
	on_success: processor_chain([
		dup43,
		dup94,
		dup14,
		dup238,
		dup41,
		dup15,
		dup239,
		dup240,
		dup241,
	]),
});

var msg409 = msg("Scan:12", all142);

var part507 = match("MESSAGE#341:Scan:13/0", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld50->} %{fld52},End:%{fld51},%{disposition},Duration (seconds): %{duration_string},User1: %{uid},User2:%{fld3},'%{info}',%{p0}");

var part508 = match("MESSAGE#341:Scan:13/2", "nwparser.p0", "Command: Full Scan,Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{dclass_counter2},Omitted: %{fld4}Computer: %{shost},IP Address: %{saddr},Domain: %{domain},Group: %{group},Server: %{hostid}");

var all143 = all_match({
	processors: [
		part507,
		dup335,
		part508,
	],
	on_success: processor_chain([
		dup43,
		dup94,
		dup14,
		dup238,
		dup41,
		dup15,
		dup239,
		dup240,
		dup241,
	]),
});

var msg410 = msg("Scan:13", all143);

var part509 = match("MESSAGE#342:Scan:14/0", "nwparser.payload", "Scan ID: %{fld11},Begin: %{fld50->} %{fld52},End: %{fld51},%{disposition},Duration (seconds): %{duration_string},User1: %{username},User2: %{fld3},%{p0}");

var part510 = match("MESSAGE#342:Scan:14/2_0", "nwparser.p0", "%{info}\",\"%{p0}");

var part511 = match("MESSAGE#342:Scan:14/2_1", "nwparser.p0", "%{info},%{p0}");

var select92 = linear_select([
	part510,
	part511,
]);

var part512 = match("MESSAGE#342:Scan:14/3_0", "nwparser.p0", "%{context}\",%{p0}");

var part513 = match("MESSAGE#342:Scan:14/3_1", "nwparser.p0", "%{context},%{p0}");

var select93 = linear_select([
	part512,
	part513,
]);

var part514 = match("MESSAGE#342:Scan:14/4", "nwparser.p0", "Command: %{fld10},Threats: %{dclass_counter3},Infected: %{dclass_counter1},Total files: %{dclass_counter2},Omitted: %{fld4},Computer: %{shost},IP Address: %{saddr},Domain: %{domain},Group: %{group},Server: %{hostid}");

var all144 = all_match({
	processors: [
		part509,
		dup316,
		select92,
		select93,
		part514,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup14,
		setf("event_description","fld10"),
		dup41,
		dup15,
		dup239,
		dup240,
		dup241,
	]),
});

var msg411 = msg("Scan:14", all144);

var select94 = linear_select([
	msg397,
	msg398,
	msg399,
	msg400,
	msg401,
	msg402,
	msg403,
	msg404,
	msg405,
	msg406,
	msg407,
	msg408,
	msg409,
	msg410,
	msg411,
]);

var part515 = match("MESSAGE#343:Security:03/2", "nwparser.p0", "%{severity},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld13},Detection score:%{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all145 = all_match({
	processors: [
		dup250,
		dup325,
		part515,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup132,
		dup152,
		dup162,
		dup163,
		dup164,
		dup154,
		dup15,
		dup19,
	]),
});

var msg412 = msg("Security:03", all145);

var all146 = all_match({
	processors: [
		dup250,
		dup325,
		dup161,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup132,
		dup152,
		dup162,
		dup163,
		dup164,
		dup154,
		dup15,
		dup19,
	]),
});

var msg413 = msg("Security:06", all146);

var part516 = match("MESSAGE#345:Security:05/2", "nwparser.p0", "%{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},Cookie:%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Last update time: %{fld57},Domain: %{domain->} ,%{p0}");

var part517 = match("MESSAGE#345:Security:05/3_0", "nwparser.p0", "\" %{p0}");

var select95 = linear_select([
	part517,
	dup194,
]);

var part518 = match("MESSAGE#345:Security:05/4", "nwparser.p0", "Group: %{group->} %{p0}");

var part519 = match("MESSAGE#345:Security:05/5_0", "nwparser.p0", "\", %{p0}");

var part520 = match("MESSAGE#345:Security:05/5_1", "nwparser.p0", ", %{p0}");

var select96 = linear_select([
	part519,
	part520,
]);

var part521 = match("MESSAGE#345:Security:05/6", "nwparser.p0", "Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type}, File size (bytes): %{p0}");

var all147 = all_match({
	processors: [
		dup251,
		dup329,
		part516,
		select95,
		part518,
		select96,
		part521,
		dup336,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup132,
		dup162,
		dup163,
		dup164,
		dup154,
		dup15,
		dup19,
	]),
});

var msg414 = msg("Security:05", all147);

var part522 = match("MESSAGE#346:Security:04", "nwparser.payload", "Security risk found,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},0,Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{filename_size}", processor_chain([
	dup110,
	dup12,
	dup115,
	dup116,
	dup38,
	dup152,
	dup132,
	dup162,
	dup163,
	dup164,
	dup154,
	dup15,
	dup19,
]));

var msg415 = msg("Security:04", part522);

var part523 = match("MESSAGE#347:Security:07/2", "nwparser.p0", "%{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Last update time: %{fld57},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type}, File size (bytes): %{p0}");

var all148 = all_match({
	processors: [
		dup251,
		dup329,
		part523,
		dup336,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup152,
		dup132,
		dup162,
		dup163,
		dup164,
		dup154,
		dup15,
		dup19,
	]),
});

var msg416 = msg("Security:07", all148);

var part524 = match("MESSAGE#348:Security:13/0", "nwparser.payload", "Security risk found,Computer name: %{shost},%{p0}");

var part525 = match("MESSAGE#348:Security:13/1_0", "nwparser.p0", "Intensive Protection Level: %{fld61},Certificate issuer: %{fld60},Certificate signer: %{fld62},Certificate thumbprint: %{fld63},Signing timestamp: %{fld64},Certificate serial number: %{fld65},%{p0}");

var select97 = linear_select([
	part525,
	dup77,
]);

var part526 = match("MESSAGE#348:Security:13/2", "nwparser.p0", "IP Address: %{saddr},Detection type: %{severity},First Seen: %{fld1},Application name: %{application},Application type: %{obj_type},Application version:%{version->} ,Hash type: %{encryption_type},Application hash: %{checksum},Company name: %{fld3->} ,File size (bytes): %{filename_size},Sensitivity: %{fld4},Detection score: %{fld5},COH Engine Version: %{fld6},%{fld7},Permitted application reason: %{fld8},Disposition: %{result},Download site: %{fld10},Web domain:%{fld11->} ,Downloaded by: %{fld12},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld15},Risk Level: %{fld16},Risk type: %{fld17},Source: %{event_source},Risk name:%{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld18},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld19->} %{fld20},Inserted: %{fld21},End: %{fld22},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld23},Source IP: %{fld24}");

var all149 = all_match({
	processors: [
		part524,
		select97,
		part526,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup115,
		dup116,
		dup38,
		dup162,
		date_time({
			dest: "event_time",
			args: ["fld19","fld20"],
			fmts: [
				[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
			],
		}),
		date_time({
			dest: "recorded_time",
			args: ["fld21"],
			fmts: [
				[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
			],
		}),
		date_time({
			dest: "endtime",
			args: ["fld22"],
			fmts: [
				[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
			],
		}),
		dup15,
		dup19,
	]),
});

var msg417 = msg("Security:13", all149);

var part527 = match("MESSAGE#349:Security", "nwparser.payload", "Security risk found,Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}", processor_chain([
	dup110,
	dup12,
	dup115,
	dup116,
	dup38,
	dup152,
	dup162,
	dup132,
	dup163,
	dup164,
	dup154,
	dup15,
	dup19,
]));

var msg418 = msg("Security", part527);

var part528 = match("MESSAGE#350:Security:01", "nwparser.payload", "Security risk found,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},Cookie: %{fld1},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31}", processor_chain([
	dup110,
	dup12,
	dup115,
	dup116,
	dup38,
	dup152,
	dup163,
	dup164,
	dup154,
	dup15,
	dup47,
	dup162,
]));

var msg419 = msg("Security:01", part528);

var part529 = match("MESSAGE#351:Security:02", "nwparser.payload", "Security risk found,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31}", processor_chain([
	dup110,
	dup12,
	dup115,
	dup116,
	dup38,
	dup152,
	dup162,
	dup132,
	dup163,
	dup164,
	dup154,
	dup15,
	dup19,
]));

var msg420 = msg("Security:02", part529);

var select98 = linear_select([
	msg412,
	msg413,
	msg414,
	msg415,
	msg416,
	msg417,
	msg418,
	msg419,
	msg420,
]);

var part530 = match("MESSAGE#352:Compressed", "nwparser.payload", "Compressed File,Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}", processor_chain([
	dup110,
	dup12,
	dup152,
	dup163,
	dup164,
	dup132,
	dup154,
	dup15,
	dup253,
	dup19,
]));

var msg421 = msg("Compressed", part530);

var part531 = match("MESSAGE#353:Compressed:02/0", "nwparser.payload", "Compressed File,IP Address: %{saddr},Computer name: %{shost},%{p0}");

var part532 = match("MESSAGE#353:Compressed:02/2", "nwparser.p0", "%{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},%{p0}");

var all150 = all_match({
	processors: [
		part531,
		dup329,
		part532,
		dup330,
		dup205,
		dup331,
	],
	on_success: processor_chain([
		dup110,
		dup12,
		dup152,
		dup163,
		dup164,
		dup132,
		dup154,
		dup15,
		dup253,
		dup19,
	]),
});

var msg422 = msg("Compressed:02", all150);

var part533 = match("MESSAGE#354:Compressed:01", "nwparser.payload", "Compressed File,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{info},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31}", processor_chain([
	dup110,
	dup12,
	dup152,
	dup163,
	dup164,
	dup132,
	dup154,
	dup15,
	dup253,
	dup19,
]));

var msg423 = msg("Compressed:01", part533);

var select99 = linear_select([
	msg421,
	msg422,
	msg423,
]);

var part534 = match("MESSAGE#355:Stop", "nwparser.payload", "Stop serving as the Group Update Provider (proxy server)%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup254,
]));

var msg424 = msg("Stop", part534);

var part535 = match("MESSAGE#356:Stop:01", "nwparser.payload", "Stop Symantec Network Access Control client.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup254,
]));

var msg425 = msg("Stop:01", part535);

var part536 = match("MESSAGE#357:Stop:02", "nwparser.payload", "Stop using Group Update Provider (proxy server) @ %{saddr}:%{sport}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Stop using Group Update Provider (proxy server)."),
]));

var msg426 = msg("Stop:02", part536);

var select100 = linear_select([
	msg424,
	msg425,
	msg426,
]);

var part537 = match("MESSAGE#358:Stopping/0", "nwparser.payload", "Stopping Symantec Management Client....%{p0}");

var all151 = all_match({
	processors: [
		part537,
		dup318,
	],
	on_success: processor_chain([
		dup136,
		dup12,
		dup13,
		setc("ec_activity","Stop"),
		dup97,
		dup22,
		dup14,
		dup15,
		dup93,
		setc("event_description","Stopping Symantec Management Client"),
	]),
});

var msg427 = msg("Stopping", all151);

var part538 = match("MESSAGE#359:Submission", "nwparser.payload", "Submission Control signatures %{version->} is up-to-date.", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Submission Control signatures is up to date"),
]));

var msg428 = msg("Submission", part538);

var part539 = match("MESSAGE#360:Switched", "nwparser.payload", "Switched to server control.%{}", processor_chain([
	dup136,
	dup12,
	dup13,
	dup30,
	dup97,
	dup22,
	dup14,
	dup15,
	setc("event_description","Switched to server control."),
]));

var msg429 = msg("Switched", part539);

var part540 = match("MESSAGE#361:Symantec:18", "nwparser.payload", "Symantec Endpoint Protection Manager Content Catalog %{version->} is up-to-date.", processor_chain([
	dup86,
	dup15,
	setc("event_description","Symantec Endpoint Protection Manager Content Catalog is up to date."),
]));

var msg430 = msg("Symantec:18", part540);

var part541 = match("MESSAGE#362:Symantec:33", "nwparser.payload", "Symantec Endpoint Protection Manager could not update TruScan proactive threat scan commercial application list %{application}.", processor_chain([
	dup43,
	dup15,
	setc("event_description","Symantec Endpoint Protection Manager could not update TruScan proactive threat scan."),
]));

var msg431 = msg("Symantec:33", part541);

var part542 = match("MESSAGE#363:Symantec:17", "nwparser.payload", "Symantec Endpoint Protection %{application->} %{version->} (%{info}) is up-to-date.", processor_chain([
	dup86,
	dup15,
	setc("event_description","Symantec Endpoint Protection is up to date."),
]));

var msg432 = msg("Symantec:17", part542);

var part543 = match("MESSAGE#364:Symantec:20", "nwparser.payload", "Symantec Endpoint Protection %{application->} %{version->} (%{info}) failed to update.", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","Symantec Endpoint Protection failed to update."),
]));

var msg433 = msg("Symantec:20", part543);

var part544 = match("MESSAGE#365:Symantec:16/0", "nwparser.payload", "Symantec Endpoint Protection Microsoft Exchange E-mail Auto-Protect Disabled%{p0}");

var all152 = all_match({
	processors: [
		part544,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup93,
		setc("event_description","Symantec Endpoint Protection Microsoft Exchange E-mail Auto-Protect Disabled"),
	]),
});

var msg434 = msg("Symantec:16", all152);

var part545 = match("MESSAGE#366:Symantec:15", "nwparser.payload", "Symantec Network Access Control client started.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	setc("event_description","Symantec Network Access Control client started."),
]));

var msg435 = msg("Symantec:15", part545);

var part546 = match("MESSAGE#367:Symantec:11", "nwparser.payload", "Symantec Endpoint Protection Tamper Protection Disabled%{}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Tamper Protection disabled"),
]));

var msg436 = msg("Symantec:11", part546);

var part547 = match("MESSAGE#368:Symantec", "nwparser.payload", "Symantec AntiVirus Startup/Shutdown..Computer: %{shost}..Date: %{fld5}..Time: %{fld6}..Description: %{info}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup43,
	dup166,
	dup15,
	dup255,
]));

var msg437 = msg("Symantec", part547);

var part548 = match("MESSAGE#369:Symantec:01", "nwparser.payload", "Symantec AntiVirus Startup/Shutdown..%{shost}..%{fld5}........%{severity}..%{product}..%{fld6}", processor_chain([
	dup43,
	dup166,
	dup15,
	dup255,
]));

var msg438 = msg("Symantec:01", part548);

var part549 = match("MESSAGE#370:Symantec:02", "nwparser.payload", "Symantec AntiVirus Startup/Shutdown..%{shost}..%{fld5}..%{severity}..%{product}..%{fld6}", processor_chain([
	dup43,
	dup166,
	dup15,
	dup255,
]));

var msg439 = msg("Symantec:02", part549);

var part550 = match("MESSAGE#371:Symantec:03/0", "nwparser.payload", "Symantec Endpoint Protection Manager Content Catalog %{version->} %{p0}");

var part551 = match("MESSAGE#371:Symantec:03/1_0", "nwparser.p0", "is up-to-date %{p0}");

var part552 = match("MESSAGE#371:Symantec:03/1_1", "nwparser.p0", "was successfully updated %{p0}");

var select101 = linear_select([
	part551,
	part552,
]);

var part553 = match("MESSAGE#371:Symantec:03/2", "nwparser.p0", ".%{}");

var all153 = all_match({
	processors: [
		part550,
		select101,
		part553,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","Symantec Endpoint Protection Manager Content Catalog is up to date or successfully updated."),
	]),
});

var msg440 = msg("Symantec:03", all153);

var part554 = match("MESSAGE#372:Symantec:04/0", "nwparser.payload", "Symantec Endpoint Protection services shutdown was successful.%{p0}");

var all154 = all_match({
	processors: [
		part554,
		dup318,
	],
	on_success: processor_chain([
		dup256,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Symantec Endpoint Protection services shutdown was successful."),
	]),
});

var msg441 = msg("Symantec:04", all154);

var part555 = match("MESSAGE#373:Symantec:05/0", "nwparser.payload", "Symantec Endpoint Protection services startup was successful.%{p0}");

var all155 = all_match({
	processors: [
		part555,
		dup318,
	],
	on_success: processor_chain([
		dup257,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Symantec Endpoint Protection services startup was successful."),
	]),
});

var msg442 = msg("Symantec:05", all155);

var part556 = match("MESSAGE#374:Symantec:06/0", "nwparser.payload", "Symantec Management Client is stopped.%{p0}");

var all156 = all_match({
	processors: [
		part556,
		dup318,
	],
	on_success: processor_chain([
		dup256,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Symantec Management Client is stopped."),
	]),
});

var msg443 = msg("Symantec:06", all156);

var part557 = match("MESSAGE#375:Symantec:07/0", "nwparser.payload", "Symantec Management Client has been %{p0}");

var part558 = match("MESSAGE#375:Symantec:07/1_0", "nwparser.p0", "started%{p0}");

var part559 = match("MESSAGE#375:Symantec:07/1_1", "nwparser.p0", "activated%{p0}");

var select102 = linear_select([
	part558,
	part559,
]);

var part560 = match("MESSAGE#375:Symantec:07/2_1", "nwparser.p0", " .%{}");

var select103 = linear_select([
	dup186,
	part560,
]);

var all157 = all_match({
	processors: [
		part557,
		select102,
		select103,
	],
	on_success: processor_chain([
		dup257,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Symantec Management Client has been started or activated."),
	]),
});

var msg444 = msg("Symantec:07", all157);

var part561 = match("MESSAGE#376:Symantec:08", "nwparser.payload", "Symantec Management Client has been %{info}", processor_chain([
	dup257,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Symantec Management Client has been activated."),
]));

var msg445 = msg("Symantec:08", part561);

var part562 = match("MESSAGE#377:Symantec:09", "nwparser.payload", "Symantec Endpoint Protection Auto-Protect failed to load.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Symantec Endpoint Protection Auto-Protect failed to load."),
]));

var msg446 = msg("Symantec:09", part562);

var part563 = match("MESSAGE#378:Symantec:10/0", "nwparser.payload", "Symantec Endpoint Protection has determined that the virus definitions are missing on this computer. %{p0}");

var all158 = all_match({
	processors: [
		part563,
		dup333,
	],
	on_success: processor_chain([
		dup168,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","virus definitions are missing on this computer"),
	]),
});

var msg447 = msg("Symantec:10", all158);

var part564 = match("MESSAGE#379:Symantec:12", "nwparser.payload", "Symantec AntiVirus services startup was successful%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","services startup was successful"),
]));

var msg448 = msg("Symantec:12", part564);

var part565 = match("MESSAGE#380:Symantec:13", "nwparser.payload", "Symantec AntiVirus services shutdown was successful%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","services shutdown was successful"),
]));

var msg449 = msg("Symantec:13", part565);

var part566 = match("MESSAGE#381:Symantec:14", "nwparser.payload", "Symantec AntiVirus services failed to start. %{space->} (%{resultcode})", processor_chain([
	dup86,
	dup12,
	dup13,
	dup14,
	dup15,
	dup258,
]));

var msg450 = msg("Symantec:14", part566);

var part567 = match("MESSAGE#382:Symantec:19", "nwparser.payload", "Symantec Endpoint Protection services failed to start. %{space->} (%{resultcode})", processor_chain([
	dup86,
	dup12,
	dup13,
	dup14,
	dup15,
	dup258,
]));

var msg451 = msg("Symantec:19", part567);

var part568 = match("MESSAGE#383:Symantec:21", "nwparser.payload", "Symantec Endpoint Protection Manager server started with trial license.%{}", processor_chain([
	dup43,
	dup15,
	setc("event_description","Symantec Endpoint Protection Manager server started with trial license."),
]));

var msg452 = msg("Symantec:21", part568);

var part569 = match("MESSAGE#384:Symantec:22", "nwparser.payload", "Symantec trial license has expired.%{}", processor_chain([
	dup259,
	dup15,
	setc("event_description","Symantec trial license has expired."),
]));

var msg453 = msg("Symantec:22", part569);

var part570 = match("MESSAGE#385:Symantec:23", "nwparser.payload", "Category: %{fld22},Symantec Endpoint Protection,\"Reputation check timed out during unproven file evaluation, likely due to network delays.\"", processor_chain([
	dup259,
	dup12,
	dup13,
	dup15,
	setc("event_description","Reputation check timed out"),
]));

var msg454 = msg("Symantec:23", part570);

var part571 = match("MESSAGE#386:Symantec:24", "nwparser.payload", "Symantec Endpoint Protection Lotus Notes E-mail Auto-Protect Disabled%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Symantec Endpoint Protection Lotus Notes E-mail Auto-Protect Disabled"),
]));

var msg455 = msg("Symantec:24", part571);

var part572 = match("MESSAGE#387:Symantec:25", "nwparser.payload", "Category: %{fld22},Symantec AntiVirus,[Antivirus advanced heuristic detection submission] Submitting file to Symantec failed. File : '%{filename}'.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Submitting file to Symantec failed"),
]));

var msg456 = msg("Symantec:25", part572);

var select104 = linear_select([
	dup261,
	dup262,
]);

var part573 = match("MESSAGE#388:Symantec:26/2", "nwparser.p0", "%{}advanced heuristic detection submission] Submitting information to Symantec about file failed. File : '%{filename}'.%{p0}");

var part574 = match("MESSAGE#388:Symantec:26/3_0", "nwparser.p0", " Network error : '%{fld56}'.,Event time: %{fld17->} %{fld18}");

var select105 = linear_select([
	part574,
	dup176,
	dup91,
]);

var all159 = all_match({
	processors: [
		dup260,
		select104,
		part573,
		select105,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","Submitting information to Symantec about file failed"),
	]),
});

var msg457 = msg("Symantec:26", all159);

var part575 = match("MESSAGE#389:Symantec:39/4", "nwparser.p0", "%{}submission] Information submitted to Symantec about file. File : '%{filename}',%{p0}");

var all160 = all_match({
	processors: [
		dup260,
		dup337,
		dup263,
		dup338,
		part575,
		dup339,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","Information submitted to Symantec about file."),
	]),
});

var msg458 = msg("Symantec:39", all160);

var part576 = match("MESSAGE#390:Symantec:40/4", "nwparser.p0", "%{}submission] File submitted to Symantec for analysis. File : '%{filename}',%{p0}");

var all161 = all_match({
	processors: [
		dup260,
		dup337,
		dup263,
		dup338,
		part576,
		dup339,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","File submitted to Symantec for analysis."),
	]),
});

var msg459 = msg("Symantec:40", all161);

var part577 = match("MESSAGE#391:Symantec:27", "nwparser.payload", "Symantec Endpoint Protection Manager server started with paid license.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Symantec Endpoint Protection Manager server started with paid license."),
]));

var msg460 = msg("Symantec:27", part577);

var part578 = match("MESSAGE#392:Symantec:28", "nwparser.payload", "Uninstalling Symantec Management Client....%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Uninstalling Symantec Management Client"),
]));

var msg461 = msg("Symantec:28", part578);

var part579 = match("MESSAGE#393:Symantec:29", "nwparser.payload", "Category: 2,Symantec Endpoint Protection,SONAR has generated an error: code %{resultcode}: description: %{result}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup187,
	dup15,
	setc("event_description","SONAR has generated an error"),
]));

var msg462 = msg("Symantec:29", part579);

var part580 = match("MESSAGE#394:Symantec:30", "nwparser.payload", "Symantec Endpoint Protection cannot connect to Symantec Endpoint Protection Manager. %{result}.", processor_chain([
	dup43,
	dup12,
	dup13,
	dup268,
	dup187,
	dup15,
	setc("event_description","Symantec Endpoint Protection cannot connect to Symantec Endpoint Protection Manager."),
]));

var msg463 = msg("Symantec:30", part580);

var part581 = match("MESSAGE#395:Symantec:31", "nwparser.payload", "The Symantec Endpoint Protection is unable to communicate with the Symantec Endpoint Protection Manager.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup268,
	dup187,
	dup15,
	setc("event_description","The Symantec Endpoint Protection is unable to communicate with the Symantec Endpoint Protection Manager."),
]));

var msg464 = msg("Symantec:31", part581);

var part582 = match("MESSAGE#396:Symantec:32", "nwparser.payload", "The Symantec Endpoint Protection is unable to download the newest policy from the Symantec Endpoint Protection Manager.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","The Symantec Endpoint Protection is unable to download the newest policy from the Symantec Endpoint Protection Manager."),
]));

var msg465 = msg("Symantec:32", part582);

var part583 = match("MESSAGE#397:Symantec:36/0", "nwparser.payload", "Category: 2,Symantec Endpoint Protection,SymELAM Protection has been enabled%{p0}");

var all162 = all_match({
	processors: [
		part583,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","SymELAM Protection has been enabled"),
	]),
});

var msg466 = msg("Symantec:36", all162);

var part584 = match("MESSAGE#398:Symantec:37/0", "nwparser.payload", "Category: 2,Symantec Endpoint Protection,SONAR has been enabled%{p0}");

var all163 = all_match({
	processors: [
		part584,
		dup318,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup15,
		dup93,
		setc("event_description","SONAR has been enabled"),
	]),
});

var msg467 = msg("Symantec:37", all163);

var part585 = match("MESSAGE#401:Symantec:41", "nwparser.payload", "Category: %{fld22},Symantec Endpoint Protection,SONAR has been disabled", processor_chain([
	dup43,
	dup56,
	dup12,
	dup13,
	dup15,
	setc("event_description","SONAR has been disabled"),
]));

var msg468 = msg("Symantec:41", part585);

var part586 = match("MESSAGE#403:Symantec:44", "nwparser.payload", "Symantec Endpoint Protection Internet E-mail Auto-Protect Disabled,Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Symantec Endpoint Protection Internet E-mail Auto-Protect Disabled"),
]));

var msg469 = msg("Symantec:44", part586);

var part587 = match("MESSAGE#511:Server:02", "nwparser.payload", "Symantec Network Access Control is overdeployed%{}", processor_chain([
	dup86,
	dup12,
	dup222,
	dup14,
	dup15,
]));

var msg470 = msg("Server:02", part587);

var part588 = match("MESSAGE#513:Server:04", "nwparser.payload", "Symantec Endpoint Protection is overdeployed%{}", processor_chain([
	dup86,
	dup12,
	dup222,
	setc("event_description","Symantec Endpoint Protection is overdeployed"),
	dup40,
	dup15,
]));

var msg471 = msg("Server:04", part588);

var part589 = match("MESSAGE#688:Symantec:34", "nwparser.payload", "Symantec Endpoint Protection Manager could not update %{application}.", processor_chain([
	dup43,
	dup14,
	dup15,
	setc("event_description","Symantec Endpoint Protection Manager could not update."),
]));

var msg472 = msg("Symantec:34", part589);

var part590 = match("MESSAGE#689:Symantec:35/0_0", "nwparser.payload", "%{event_description}. File : %{filename}, Size (bytes): %{filename_size}.\",Event time:%{fld17->} %{fld18}");

var part591 = match("MESSAGE#689:Symantec:35/0_1", "nwparser.payload", "%{event_description}. File : %{filename},Event time:%{fld17->} %{fld18}");

var part592 = match("MESSAGE#689:Symantec:35/0_2", "nwparser.payload", "%{event_description}.,Event time:%{fld17->} %{fld18}");

var part593 = match("MESSAGE#689:Symantec:35/0_3", "nwparser.payload", "%{event_description}Operating System: %{os}Network info:%{info},Event time:%{fld17->} %{fld18}");

var part594 = match("MESSAGE#689:Symantec:35/0_4", "nwparser.payload", "%{event_description}.");

var select106 = linear_select([
	part590,
	part591,
	part592,
	part593,
	part594,
]);

var all164 = all_match({
	processors: [
		select106,
	],
	on_success: processor_chain([
		dup43,
		dup94,
		dup13,
		dup14,
		dup15,
		dup93,
	]),
});

var msg473 = msg("Symantec:35", all164);

var part595 = match("MESSAGE#690:Symantec:45", "nwparser.payload", "Category: %{fld22},Symantec Endpoint Protection,%{event_description},Event time:%{fld17->} %{fld18}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	dup93,
]));

var msg474 = msg("Symantec:45", part595);

var part596 = match_copy("MESSAGE#691:Server:05", "nwparser.payload", "event_description", processor_chain([
	dup53,
	dup12,
	dup222,
	dup40,
	dup15,
]));

var msg475 = msg("Server:05", part596);

var select107 = linear_select([
	msg430,
	msg431,
	msg432,
	msg433,
	msg434,
	msg435,
	msg436,
	msg437,
	msg438,
	msg439,
	msg440,
	msg441,
	msg442,
	msg443,
	msg444,
	msg445,
	msg446,
	msg447,
	msg448,
	msg449,
	msg450,
	msg451,
	msg452,
	msg453,
	msg454,
	msg455,
	msg456,
	msg457,
	msg458,
	msg459,
	msg460,
	msg461,
	msg462,
	msg463,
	msg464,
	msg465,
	msg466,
	msg467,
	msg468,
	msg469,
	msg470,
	msg471,
	msg472,
	msg473,
	msg474,
	msg475,
]);

var part597 = match("MESSAGE#402:Symantec:43", "nwparser.payload", "Suspicious Behavior Detection has been %{fld2},Event time: %{event_time_string}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("a","Suspicious Behavior Detection has been "),
	call({
		dest: "nwparser.event_description",
		fn: STRCAT,
		args: [
			constant("a"),
			field("fld2"),
		],
	}),
]));

var msg476 = msg("Symantec:43", part597);

var part598 = match("MESSAGE#404:System", "nwparser.payload", "System has been restarted %{info}.", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","System has been restarted"),
]));

var msg477 = msg("System", part598);

var part599 = match("MESSAGE#405:System:01", "nwparser.payload", "System client-server activity logs have been swept.%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","System client-server activity logs have been swept."),
]));

var msg478 = msg("System:01", part599);

var part600 = match("MESSAGE#406:System:02", "nwparser.payload", "System server activity logs have been swept.%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","System server activity logs have been swept."),
]));

var msg479 = msg("System:02", part600);

var part601 = match("MESSAGE#407:System:03", "nwparser.payload", "System administrative logs have been swept.%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","System administrative logs have been swept."),
]));

var msg480 = msg("System:03", part601);

var part602 = match("MESSAGE#408:System:04", "nwparser.payload", "System enforcer activity logs have been swept.%{}", processor_chain([
	dup53,
	dup14,
	dup15,
	setc("event_description","System enforcer activity logs have been swept."),
]));

var msg481 = msg("System:04", part602);

var part603 = match("MESSAGE#409:System:05", "nwparser.payload", "System administrator \"%{username}\" was added", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg482 = msg("System:05", part603);

var select108 = linear_select([
	msg477,
	msg478,
	msg479,
	msg480,
	msg481,
	msg482,
]);

var part604 = match("MESSAGE#410:Terminated/0_0", "nwparser.payload", "- Caller MD5=%{fld6},%{p0}");

var select109 = linear_select([
	part604,
	dup269,
]);

var part605 = match("MESSAGE#410:Terminated/1", "nwparser.p0", "%{action},Begin:%{fld50->} %{fld52},End:%{fld51->} %{fld53},Rule:%{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User:%{username},Domain:%{domain},Action Type:%{fld45},File size (bytes):%{filename_size},Device ID:%{device}");

var all165 = all_match({
	processors: [
		select109,
		part605,
	],
	on_success: processor_chain([
		dup36,
		dup12,
		dup13,
		dup129,
		dup37,
		dup14,
		dup41,
		dup42,
		dup15,
		setc("event_state","Terminated"),
	]),
});

var msg483 = msg("Terminated", all165);

var part606 = match("MESSAGE#411:Compliance/0", "nwparser.payload", "Compliance %{p0}");

var part607 = match("MESSAGE#411:Compliance/1_0", "nwparser.p0", "server %{p0}");

var part608 = match("MESSAGE#411:Compliance/1_1", "nwparser.p0", "client %{p0}");

var part609 = match("MESSAGE#411:Compliance/1_2", "nwparser.p0", "traffic %{p0}");

var part610 = match("MESSAGE#411:Compliance/1_3", "nwparser.p0", "criteria %{p0}");

var select110 = linear_select([
	part607,
	part608,
	part609,
	part610,
]);

var part611 = match("MESSAGE#411:Compliance/2", "nwparser.p0", "logs have been swept.%{}");

var all166 = all_match({
	processors: [
		part606,
		select110,
		part611,
	],
	on_success: processor_chain([
		dup53,
		dup14,
		dup15,
		setc("event_description","Compliance logs have been swept."),
	]),
});

var msg484 = msg("Compliance", all166);

var part612 = match("MESSAGE#412:Download", "nwparser.payload", "Download started.%{}", processor_chain([
	dup43,
	dup14,
	dup15,
	setc("event_description","Download started."),
]));

var msg485 = msg("Download", part612);

var part613 = match("MESSAGE#413:Traffic", "nwparser.payload", "Traffic from IP address %{hostip->} is blocked from %{fld14->} to %{fld15}.,Local: %{daddr},Local: %{fld16},Remote: %{fld17},Remote: %{saddr},Remote: %{fld18},Inbound,%{fld19},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld10},User: %{username},Domain: %{domain}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup270,
	dup19,
	dup34,
]));

var msg486 = msg("Traffic", part613);

var part614 = match("MESSAGE#414:Traffic:11", "nwparser.payload", "Traffic from IP address %{hostip->} is blocked from %{fld14->} to %{fld15}.,Local: %{saddr},Local: %{fld16},Remote: %{fld17},Remote: %{daddr},Remote: %{fld18},Outbound,%{fld19},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld10},User: %{username},Domain: %{domain}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup270,
	dup19,
	dup35,
]));

var msg487 = msg("Traffic:11", part614);

var part615 = match("MESSAGE#415:Traffic:01", "nwparser.payload", "Traffic from IP address %{hostip->} is blocked from %{fld1->} to %{fld2}. ,Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},1,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup270,
	dup19,
]));

var msg488 = msg("Traffic:01", part615);

var part616 = match("MESSAGE#416:Traffic:02/0", "nwparser.payload", "Traffic from IP address %{hostip->} is blocked from %{fld1->} to %{fld2}. ,Local: %{daddr},Local: %{fld3},Remote: %{fld4},Remote: %{saddr},Remote: %{fld5},Inbound,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var all167 = all_match({
	processors: [
		part616,
		dup319,
		dup271,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup270,
		dup19,
		dup34,
	]),
});

var msg489 = msg("Traffic:02", all167);

var part617 = match("MESSAGE#417:Traffic:12/0", "nwparser.payload", "Traffic from IP address %{hostip->} is blocked from %{fld1->} to %{fld2}. ,Local: %{saddr},Local: %{fld3},Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Outbound,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var all168 = all_match({
	processors: [
		part617,
		dup319,
		dup271,
	],
	on_success: processor_chain([
		dup11,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup270,
		dup19,
		dup35,
	]),
});

var msg490 = msg("Traffic:12", all168);

var part618 = match("MESSAGE#717:Traffic:13", "nwparser.payload", "%{fld1->} Traffic Redirection disabled.,Event time: %{fld17->} %{fld18}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","Traffic Redirection disabled."),
	dup93,
]));

var msg491 = msg("Traffic:13", part618);

var part619 = match("MESSAGE#718:Traffic:14", "nwparser.payload", "%{fld1->} Traffic Redirection is malfunctioning.,Event time: %{fld17->} %{fld18}", processor_chain([
	dup86,
	dup12,
	dup13,
	dup15,
	setc("event_description","Traffic Redirection is malfunctioning."),
	dup93,
]));

var msg492 = msg("Traffic:14", part619);

var select111 = linear_select([
	msg486,
	msg487,
	msg488,
	msg489,
	msg490,
	msg491,
	msg492,
]);

var part620 = match("MESSAGE#418:TruScan", "nwparser.payload", "TruScan has generated an error: code %{resultcode}: description: %{info}", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","TruScan has generated an error"),
]));

var msg493 = msg("TruScan", part620);

var part621 = match("MESSAGE#419:TruScan:01/0", "nwparser.payload", "Forced TruScan proactive threat detected,Computer name: %{p0}");

var part622 = match("MESSAGE#419:TruScan:01/2", "nwparser.p0", "%{fld1},Application name: %{application},Application type: %{obj_type},Application version: %{version},Hash type: %{encryption_type},Application hash: %{checksum},Company name: %{fld13},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score: %{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},\"%{fld12}\",Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld15},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var all169 = all_match({
	processors: [
		part621,
		dup325,
		part622,
	],
	on_success: processor_chain([
		setc("eventcategory","1001030200"),
		dup12,
		dup152,
		dup93,
		date_time({
			dest: "recorded_time",
			args: ["fld15"],
			fmts: [
				[dW,dc("-"),dG,dc("-"),dF,dN,dc(":"),dU,dc(":"),dO],
			],
		}),
		dup132,
		dup154,
		dup15,
		setc("event_description"," TruScan proactive threat detected"),
		dup19,
	]),
});

var msg494 = msg("TruScan:01", all169);

var part623 = match("MESSAGE#420:TruScan:update/0", "nwparser.payload", "TruScan %{info->} %{p0}");

var part624 = match("MESSAGE#420:TruScan:update/1_0", "nwparser.p0", "was successfully updated%{}");

var part625 = match("MESSAGE#420:TruScan:update/1_1", "nwparser.p0", "is up-to-date%{}");

var select112 = linear_select([
	part624,
	part625,
]);

var all170 = all_match({
	processors: [
		part623,
		select112,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		setc("event_description","Truscan was successfully updated or is up-to-date."),
	]),
});

var msg495 = msg("TruScan:update", all170);

var part626 = match("MESSAGE#421:TruScan:updatefailed", "nwparser.payload", "TruScan %{info->} failed to update.", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Truscan failed to update."),
]));

var msg496 = msg("TruScan:updatefailed", part626);

var select113 = linear_select([
	msg493,
	msg494,
	msg495,
	msg496,
]);

var part627 = match("MESSAGE#422:Unexpected", "nwparser.payload", "Unexpected server error. ErrorCode: %{resultcode}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup272,
]));

var msg497 = msg("Unexpected", part627);

var part628 = match("MESSAGE#423:Unexpected:01", "nwparser.payload", "Unexpected server error.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	dup272,
]));

var msg498 = msg("Unexpected:01", part628);

var select114 = linear_select([
	msg497,
	msg498,
]);

var part629 = match("MESSAGE#424:Unsolicited", "nwparser.payload", "Unsolicited incoming ARP reply detected,%{info}\",Local: %{daddr},Local: %{fld16},Remote: %{fld17},Remote: %{saddr},Remote: %{fld18},Inbound,%{fld19},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld20},User: %{username},Domain: %{domain}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup273,
	dup19,
	dup34,
]));

var msg499 = msg("Unsolicited", part629);

var part630 = match("MESSAGE#425:Unsolicited:01", "nwparser.payload", "Unsolicited incoming ARP reply detected,%{info}\",Local: %{saddr},Local: %{fld16},Remote: %{fld17},Remote: %{daddr},Remote: %{fld18},Outbound,%{fld19},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld20},User: %{username},Domain: %{domain}", processor_chain([
	dup11,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup273,
	dup19,
	dup35,
]));

var msg500 = msg("Unsolicited:01", part630);

var select115 = linear_select([
	msg499,
	msg500,
]);

var part631 = match("MESSAGE#426:User/0", "nwparser.payload", "User is attempting to terminate Symantec Management Client%{p0}");

var part632 = match("MESSAGE#426:User/1_0", "nwparser.p0", "....,Event time:%{fld17->} %{fld18}");

var select116 = linear_select([
	part632,
	dup91,
]);

var all171 = all_match({
	processors: [
		part631,
		select116,
	],
	on_success: processor_chain([
		setc("eventcategory","1401040000"),
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","User is attempting to terminate Symantec Management Client."),
	]),
});

var msg501 = msg("User", all171);

var part633 = match("MESSAGE#427:User:01", "nwparser.payload", "%{fld44},User - Kernel Hook Error,%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{fld4},%{fld5},%{fld6},%{fld7},User: %{username},Domain: %{domain}", processor_chain([
	dup171,
	dup12,
	dup13,
	dup20,
	dup97,
	dup187,
	dup14,
	dup41,
	dup42,
	dup15,
	setc("event_description"," User - Kernel Hook Error"),
]));

var msg502 = msg("User:01", part633);

var part634 = match("MESSAGE#428:User:created", "nwparser.payload", "User has been created%{}", processor_chain([
	dup170,
	dup12,
	dup13,
	dup20,
	dup96,
	dup28,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","User has been created."),
]));

var msg503 = msg("User:created", part634);

var part635 = match("MESSAGE#429:User:deleted", "nwparser.payload", "User has been deleted%{}", processor_chain([
	dup171,
	dup12,
	dup13,
	dup20,
	dup27,
	dup28,
	dup22,
	dup14,
	dup15,
	dup23,
	setc("event_description","User has been deleted."),
]));

var msg504 = msg("User:deleted", part635);

var select117 = linear_select([
	msg501,
	msg502,
	msg503,
	msg504,
]);

var part636 = match("MESSAGE#446:Windows/0", "nwparser.payload", "Windows Version info: Operating System: %{os->} Network info:%{p0}");

var part637 = match("MESSAGE#446:Windows/1_0", "nwparser.p0", "%{info},Event time:%{fld17->} %{fld18}");

var select118 = linear_select([
	part637,
	dup212,
]);

var all172 = all_match({
	processors: [
		part636,
		select118,
	],
	on_success: processor_chain([
		dup92,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		dup274,
	]),
});

var msg505 = msg("Windows", all172);

var part638 = match("MESSAGE#447:Windows:01", "nwparser.payload", "Windows Host Integrity Content %{version->} was successfully updated.", processor_chain([
	dup92,
	dup12,
	dup13,
	dup14,
	dup15,
	dup274,
]));

var msg506 = msg("Windows:01", part638);

var select119 = linear_select([
	msg505,
	msg506,
]);

var part639 = match("MESSAGE#448:\"=======EXCEPTION:", "nwparser.payload", "\"=======EXCEPTION:%{event_description}\"", processor_chain([
	dup168,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg507 = msg("\"=======EXCEPTION:", part639);

var part640 = match("MESSAGE#449:Allowed:08", "nwparser.payload", "Sysfer exception: %{info},Sysfer exception,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},%{event_description},User: %{username},Domain: %{domain},Action Type:%{fld6},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup132,
	dup15,
]));

var msg508 = msg("Allowed:08", part640);

var part641 = match("MESSAGE#450:Allowed", "nwparser.payload", "Sysfer exception: %{info},Sysfer exception,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},%{event_description},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup132,
	dup15,
]));

var msg509 = msg("Allowed", part641);

var part642 = match("MESSAGE#451:Allowed:05", "nwparser.payload", "\"%{filename}\",%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld4},%{process},%{fld5},%{fld6},%{info},User: %{username},Domain: %{domain},Action Type: %{fld8}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
]));

var msg510 = msg("Allowed:05", part642);

var part643 = match("MESSAGE#452:Allowed:06", "nwparser.payload", "\"%{filename},%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld4},%{process},%{fld5},%{fld6},%{info},User: %{username},Domain: %{domain},Action Type: %{fld8}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
]));

var msg511 = msg("Allowed:06", part643);

var part644 = match("MESSAGE#453:Allowed:01", "nwparser.payload", "\"%{filename}\",%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld4},%{process},%{fld5},%{fld6},%{info},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
]));

var msg512 = msg("Allowed:01", part644);

var part645 = match("MESSAGE#454:Allowed:02/0", "nwparser.payload", "%{fld1},File Read,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},No Module Name,%{directory},User: %{username},Domain: %{p0}");

var part646 = match("MESSAGE#454:Allowed:02/1_0", "nwparser.p0", "%{domain},Action Type:%{fld45},File size (bytes):%{filename_size},Device ID:%{device}");

var select120 = linear_select([
	part646,
	dup10,
]);

var all173 = all_match({
	processors: [
		part645,
		select120,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup14,
		setc("event_description","File Read"),
		dup41,
		dup42,
		dup132,
		dup15,
		dup124,
		dup125,
	]),
});

var msg513 = msg("Allowed:02", all173);

var part647 = match("MESSAGE#455:Allowed:09/0_0", "nwparser.payload", "- Caller MD5=%{checksum},File Write,Begin: %{p0}");

var part648 = match("MESSAGE#455:Allowed:09/0_1", "nwparser.payload", "%{fld1},File Write,Begin: %{p0}");

var select121 = linear_select([
	part647,
	part648,
]);

var part649 = match("MESSAGE#455:Allowed:09/1", "nwparser.p0", "%{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{p0}");

var part650 = match("MESSAGE#455:Allowed:09/3", "nwparser.p0", "%{username},Domain: %{domain},Action Type:%{fld46},File size (%{fld10}): %{filename_size},Device ID: %{device}");

var all174 = all_match({
	processors: [
		select121,
		part649,
		dup340,
		part650,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup129,
		dup41,
		dup42,
		dup277,
		dup15,
		dup124,
		dup128,
	]),
});

var msg514 = msg("Allowed:09", all174);

var part651 = match("MESSAGE#456:Allowed:03", "nwparser.payload", "%{fld1},File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},No Module Name,%{directory},User: %{username},Domain: %{domain},Action Type:%{fld46}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup277,
	dup132,
	dup15,
	dup124,
	dup128,
]));

var msg515 = msg("Allowed:03", part651);

var part652 = match("MESSAGE#457:Allowed:10/0", "nwparser.payload", "- Caller MD5=%{checksum},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{p0}");

var part653 = match("MESSAGE#457:Allowed:10/2", "nwparser.p0", "User: %{username},Domain: %{domain},Action Type:%{p0}");

var part654 = match_copy("MESSAGE#457:Allowed:10/3_1", "nwparser.p0", "fld46");

var select122 = linear_select([
	dup278,
	part654,
]);

var all175 = all_match({
	processors: [
		part652,
		dup327,
		part653,
		select122,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup124,
		dup27,
		dup14,
		dup41,
		dup42,
		dup279,
		dup15,
		dup131,
	]),
});

var msg516 = msg("Allowed:10", all175);

var part655 = match("MESSAGE#458:Allowed:04", "nwparser.payload", "%{fld1},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{filename},%{fld4},No Module Name,%{directory},User: %{username},Domain: %{domain},Action Type:%{fld46}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup124,
	dup27,
	dup14,
	dup41,
	dup42,
	dup132,
	dup279,
	dup15,
	dup131,
]));

var msg517 = msg("Allowed:04", part655);

var part656 = match("MESSAGE#459:Allowed:07", "nwparser.payload", "%{filename},%{fld1},Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld4},%{process},%{fld5},%{fld6},%{info},User: %{username},Domain: %{domain},Action Type: %{fld8}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup122,
]));

var msg518 = msg("Allowed:07", part656);

var select123 = linear_select([
	msg508,
	msg509,
	msg510,
	msg511,
	msg512,
	msg513,
	msg514,
	msg515,
	msg516,
	msg517,
	msg518,
]);

var part657 = match("MESSAGE#460:Audit", "nwparser.payload", "Audit logs have been swept.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Audit logs have been swept."),
]));

var msg519 = msg("Audit", part657);

var part658 = match("MESSAGE#465:Category", "nwparser.payload", "%{fld24},%{fld1},FATAL: %{event_description}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg520 = msg("Category", part658);

var part659 = match("MESSAGE#466:Category:03/0", "nwparser.payload", "%{fld1},%{fld2},%{event_description->} Remote file path:%{p0}");

var part660 = match("MESSAGE#466:Category:03/1_0", "nwparser.p0", "%{url},Event time:%{fld17->} %{fld18}");

var select124 = linear_select([
	part660,
	dup64,
]);

var all176 = all_match({
	processors: [
		part659,
		select124,
	],
	on_success: processor_chain([
		dup43,
		fqdn("daddr","url"),
		port("dport","url"),
		dup12,
		dup13,
		dup14,
		dup93,
		dup15,
	]),
});

var msg521 = msg("Category:03", all176);

var part661 = match("MESSAGE#467:Category:02/0", "nwparser.payload", "%{fld1},%{fld2},Downloaded content from GUP %{daddr}: %{p0}");

var part662 = match("MESSAGE#467:Category:02/1_0", "nwparser.p0", "%{dport},Event time:%{fld17->} %{fld18}");

var part663 = match_copy("MESSAGE#467:Category:02/1_1", "nwparser.p0", "dport");

var select125 = linear_select([
	part662,
	part663,
]);

var all177 = all_match({
	processors: [
		part661,
		select125,
	],
	on_success: processor_chain([
		dup43,
		setc("event_description","Downloaded content from GUP"),
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
	]),
});

var msg522 = msg("Category:02", all177);

var part664 = match("MESSAGE#468:Category:01/0", "nwparser.payload", "%{fld1},%{fld2},%{p0}");

var part665 = match("MESSAGE#468:Category:01/1_0", "nwparser.p0", "%{event_description}. File : '%{filename}',\",Event time: %{fld17->} %{fld18}");

var part666 = match("MESSAGE#468:Category:01/1_1", "nwparser.p0", "%{event_description}Size (bytes): %{filename_size}.,Event time: %{fld17->} %{fld18}");

var part667 = match("MESSAGE#468:Category:01/1_2", "nwparser.p0", "%{event_description},Event time: %{fld17->} %{fld18}");

var part668 = match("MESSAGE#468:Category:01/1_3", "nwparser.p0", "%{event_description}. Size (bytes):%{filename_size}.");

var part669 = match("MESSAGE#468:Category:01/1_4", "nwparser.p0", "%{event_description}. %{space->} File : '%{filename}',\"");

var part670 = match("MESSAGE#468:Category:01/1_5", "nwparser.p0", "%{event_description}. %{space->} File : '%{filename}'");

var part671 = match_copy("MESSAGE#468:Category:01/1_6", "nwparser.p0", "event_description");

var select126 = linear_select([
	part665,
	part666,
	part667,
	part668,
	part669,
	part670,
	part671,
]);

var all178 = all_match({
	processors: [
		part664,
		select126,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup93,
		dup15,
	]),
});

var msg523 = msg("Category:01", all178);

var select127 = linear_select([
	msg520,
	msg521,
	msg522,
	msg523,
]);

var part672 = match("MESSAGE#469:Default", "nwparser.payload", "Default %{info}..Computer: %{shost}..Date: %{fld2}..Failed Alert Name: %{action}..Time: %{fld3->} %{fld1}..Severity: %{severity}..Source: %{product}", processor_chain([
	dup43,
	date_time({
		dest: "event_time",
		args: ["fld2","fld3","fld1"],
		fmts: [
			[dG,dc("/"),dF,dc("/"),dW,dN,dc(":"),dU,dc(":"),dO,dP],
		],
	}),
	setc("event_description","Default Alert"),
	dup15,
]));

var msg524 = msg("Default", part672);

var part673 = match("MESSAGE#470:Default:01", "nwparser.payload", "Default Group blocks new clients. The client cannot register with the Default Group.%{}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup15,
	setc("event_description","Default Group blocks new clients. The client cannot register with the Default Group."),
]));

var msg525 = msg("Default:01", part673);

var select128 = linear_select([
	msg524,
	msg525,
]);

var part674 = match("MESSAGE#471:Device:01", "nwparser.payload", "%{action}. %{info},Local: %{saddr},Local: %{fld1},Remote: %{fld25},Remote: %{daddr},Remote: %{fld3},%{direction},%{fld5},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{fld24},Intrusion Payload URL:%{fld12}", processor_chain([
	dup43,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup142,
	dup19,
]));

var msg526 = msg("Device:01", part674);

var part675 = match("MESSAGE#472:Device/0", "nwparser.payload", "%{action}. %{info},Local: %{saddr},Local: %{fld1},Remote: %{fld25},Remote: %{daddr},Remote: %{fld3},%{direction},%{fld5},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld8},%{p0}");

var part676 = match("MESSAGE#472:Device/1_0", "nwparser.p0", "\"User:%{username}\",Domain:%{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld26}");

var part677 = match("MESSAGE#472:Device/1_1", "nwparser.p0", " User: %{username},Domain: %{domain}");

var select129 = linear_select([
	part676,
	part677,
]);

var all179 = all_match({
	processors: [
		part675,
		select129,
	],
	on_success: processor_chain([
		dup43,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup142,
		dup19,
	]),
});

var msg527 = msg("Device", all179);

var select130 = linear_select([
	msg526,
	msg527,
]);

var part678 = match("MESSAGE#473:Email", "nwparser.payload", "Email sending failed%{}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
	setc("event_description","Email sending failed"),
]));

var msg528 = msg("Email", part678);

var part679 = match("MESSAGE#474:FileWrite:02/0", "nwparser.payload", "%{fld5->} - Caller MD5=%{checksum},File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{p0}");

var part680 = match("MESSAGE#474:FileWrite:02/2", "nwparser.p0", "%{username},Domain: %{domain},Action Type:%{p0}");

var part681 = match_copy("MESSAGE#474:FileWrite:02/3_1", "nwparser.p0", "fld44");

var select131 = linear_select([
	dup278,
	part681,
]);

var all180 = all_match({
	processors: [
		part679,
		dup340,
		part680,
		select131,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup133,
		dup124,
		dup128,
	]),
});

var msg529 = msg("FileWrite:02", all180);

var part682 = match("MESSAGE#475:FileWrite:01", "nwparser.payload", "[AC5-1.1] Log files written to Removable Media,File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup133,
	dup124,
	dup128,
]));

var msg530 = msg("FileWrite:01", part682);

var part683 = match("MESSAGE#476:FileWrite:03", "nwparser.payload", "%{fld5},File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup133,
	dup124,
	dup128,
]));

var msg531 = msg("FileWrite:03", part683);

var part684 = match("MESSAGE#477:FileWrite", "nwparser.payload", ",File Write,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup133,
	dup124,
	dup128,
]));

var msg532 = msg("FileWrite", part684);

var part685 = match("MESSAGE#478:FileDelete", "nwparser.payload", "[AC5-1.1] Log files written to Removable Media,File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup280,
	dup124,
	dup131,
]));

var msg533 = msg("FileDelete", part685);

var part686 = match("MESSAGE#479:Continue/0", "nwparser.payload", "%{info->} - Caller MD5=%{checksum},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{p0}");

var part687 = match("MESSAGE#479:Continue/2", "nwparser.p0", "%{username},Domain: %{domain},Action Type:%{fld44},File size (bytes): %{filename_size},Device ID: %{device}");

var all181 = all_match({
	processors: [
		part686,
		dup340,
		part687,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup129,
		dup14,
		dup41,
		dup42,
		dup15,
		dup280,
		dup124,
		dup131,
	]),
});

var msg534 = msg("Continue", all181);

var part688 = match("MESSAGE#480:FileDelete:01", "nwparser.payload", "%{fld5->} - Caller MD5=%{fld6},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup280,
	dup124,
	dup131,
]));

var msg535 = msg("FileDelete:01", part688);

var part689 = match("MESSAGE#481:FileDelete:02", "nwparser.payload", "%{fld5},File Delete,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},No Module Name,%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
	dup280,
	dup124,
	dup131,
]));

var msg536 = msg("FileDelete:02", part689);

var part690 = match("MESSAGE#482:System:06", "nwparser.payload", "%{fld5},System,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld6},%{filename},User: %{username},Domain: %{domain},Action Type:%{fld44}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup15,
]));

var msg537 = msg("System:06", part690);

var part691 = match("MESSAGE#495:File:10", "nwparser.payload", "%{fld1},File Read,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Rule: %{rulename},%{fld3},%{process},%{fld4},%{fld5},%{filename},User: %{username},Domain: %{domain},Action Type:%{fld6},File size (bytes): %{filename_size},Device ID: %{device}", processor_chain([
	dup121,
	dup12,
	dup13,
	dup14,
	dup41,
	dup42,
	dup122,
	dup130,
	dup124,
	dup125,
]));

var msg538 = msg("File:10", part691);

var part692 = match("MESSAGE#503:Blocked:08/0_0", "nwparser.payload", "%{fld11->} - Caller MD5=%{fld6},%{p0}");

var select132 = linear_select([
	part692,
	dup269,
]);

var part693 = match("MESSAGE#503:Blocked:08/1", "nwparser.p0", "%{action},Begin: %{fld2->} %{fld3},End: %{fld4->} %{fld5},Rule: %{rulename},%{process_id},%{process},%{fld6},%{fld7},%{fld8},User: %{username},Domain: %{domain},Action Type: %{fld9},File size (%{fld10}): %{filename_size},Device ID: %{device}");

var all182 = all_match({
	processors: [
		select132,
		part693,
	],
	on_success: processor_chain([
		dup121,
		dup12,
		dup13,
		dup129,
		dup15,
		dup134,
		dup135,
	]),
});

var msg539 = msg("Blocked:08", all182);

var select133 = linear_select([
	msg529,
	msg530,
	msg531,
	msg532,
	msg533,
	msg534,
	msg535,
	msg536,
	msg537,
	msg538,
	msg539,
]);

var part694 = match("MESSAGE#505:Ping/1", "nwparser.p0", "%{event_description}\",Local: %{daddr},Local: %{fld1},Remote: %{fld9},Remote: %{saddr},Remote: %{fld3},Inbound,%{protocol},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{fld7}");

var all183 = all_match({
	processors: [
		dup341,
		part694,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup122,
		dup19,
		dup34,
	]),
});

var msg540 = msg("Ping", all183);

var part695 = match("MESSAGE#506:Ping:01/1", "nwparser.p0", "%{event_description}\",Local: %{saddr},Local: %{fld1},Remote: %{fld9},Remote: %{daddr},Remote: %{fld3},Outbound,%{protocol},,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{fld7}");

var all184 = all_match({
	processors: [
		dup341,
		part695,
	],
	on_success: processor_chain([
		dup69,
		dup12,
		dup13,
		dup14,
		dup41,
		dup42,
		dup15,
		dup122,
		dup19,
		dup35,
	]),
});

var msg541 = msg("Ping:01", all184);

var select134 = linear_select([
	msg540,
	msg541,
]);

var part696 = match("MESSAGE#509:Server", "nwparser.payload", "%{fld1}: Site: %{fld2},Server: %{hostid},%{directory->} %{event_description}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg542 = msg("Server", part696);

var part697 = match("MESSAGE#510:Server:01", "nwparser.payload", "Server returned HTTP response code: %{resultcode->} for URL: %{url}", processor_chain([
	dup53,
	dup12,
	dup13,
	dup14,
	dup15,
]));

var msg543 = msg("Server:01", part697);

var part698 = match("MESSAGE#512:Server:03", "nwparser.payload", "Server security validation failed.%{}", processor_chain([
	dup174,
	dup94,
	setf("saddr","hhostid"),
	dup14,
	dup15,
]));

var msg544 = msg("Server:03", part698);

var select135 = linear_select([
	msg542,
	msg543,
	msg544,
]);

var part699 = match("MESSAGE#514:1", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup200,
	dup15,
	dup283,
]));

var msg545 = msg("1", part699);

var part700 = match("MESSAGE#515:2", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup162,
	dup15,
	dup283,
]));

var msg546 = msg("2", part700);

var part701 = match("MESSAGE#516:3", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","FW Violation Event"),
	dup15,
	dup283,
]));

var msg547 = msg("3", part701);

var part702 = match("MESSAGE#517:4", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","IDS Event"),
	dup15,
	dup283,
]));

var msg548 = msg("4", part702);

var part703 = match("MESSAGE#518:5", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","CAL Event"),
	dup15,
	dup283,
]));

var msg549 = msg("5", part703);

var part704 = match("MESSAGE#519:6", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","Forced Detection Event"),
	dup15,
	dup283,
]));

var msg550 = msg("6", part704);

var part705 = match("MESSAGE#520:7", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","Detection Whitelisted"),
	dup15,
	dup283,
]));

var msg551 = msg("7", part705);

var part706 = match("MESSAGE#521:8", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup227,
	dup15,
	dup283,
]));

var msg552 = msg("8", part706);

var part707 = match("MESSAGE#522:9", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	setc("event_description","Risk submitted"),
	dup15,
	dup283,
]));

var msg553 = msg("9", part707);

var part708 = match("MESSAGE#523:10", "nwparser.payload", "%{hostip}^^%{hostname}^^%{domain}^^%{username}^^%{shost}^^%{saddr}^^%{event_source}^^%{virusname}^^%{info}^^%{disposition}^^%{action}^^%{recorded_time}^^%{fld33}^^%{fld1}^^%{dclass_counter1}^^%{filename}^^%{fld2}", processor_chain([
	dup110,
	dup115,
	dup116,
	dup38,
	dup152,
	dup253,
	dup15,
	dup283,
]));

var msg554 = msg("10", part708);

var msg555 = msg("1281", dup342);

var msg556 = msg("257", dup342);

var msg557 = msg("259", dup342);

var part709 = match("MESSAGE#527:264", "nwparser.payload", "%{id}^^%{fld1->} %{fld2->} %{fld3->} Organization importing started", processor_chain([
	dup53,
	dup284,
	dup15,
	dup220,
]));

var msg558 = msg("264", part709);

var part710 = match("MESSAGE#528:265", "nwparser.payload", "%{id}^^%{fld1->} %{fld2->} %{fld3->} Organization importing finished successfully", processor_chain([
	dup53,
	dup284,
	dup15,
	dup219,
]));

var msg559 = msg("265", part710);

var msg560 = msg("273", dup342);

var part711 = match("MESSAGE#530:275", "nwparser.payload", "%{id}^^The process %{process->} can not lock the process status table. The process status has been locked by the server %{shost->} (%{fld22}) since %{recorded_time}.", processor_chain([
	dup53,
	dup15,
	setc("event_description","The process can not lock the process status table"),
]));

var msg561 = msg("275", part711);

var msg562 = msg("769", dup342);

var msg563 = msg("772", dup342);

var msg564 = msg("773", dup342);

var msg565 = msg("778", dup342);

var msg566 = msg("779", dup342);

var msg567 = msg("782", dup342);

var part712 = match("MESSAGE#537:1029", "nwparser.payload", "%{id}^^%{fld1->} %{fld2->} %{fld3->} Backup succeeded and finished at %{fld4->} %{fld5->} %{fld6}. The backup file resides at the following location on the server %{shost}: %{directory}", processor_chain([
	dup53,
	dup284,
	date_time({
		dest: "recorded_time",
		args: ["fld4","fld5","fld6"],
		fmts: [
			[dG,dc("/"),dF,dc("/"),dY,dN,dc(":"),dU,dP],
		],
	}),
	dup15,
	dup285,
]));

var msg568 = msg("1029", part712);

var part713 = match("MESSAGE#538:1029:01", "nwparser.payload", "%{id}^^Backup succeeded and finished. The backup file resides at the following location on the server %{shost}: %{directory}", processor_chain([
	dup53,
	dup15,
	dup285,
]));

var msg569 = msg("1029:01", part713);

var select136 = linear_select([
	msg568,
	msg569,
]);

var part714 = match("MESSAGE#539:1030", "nwparser.payload", "%{id}^^%{fld1->} %{fld2->} %{fld3->} Backup started", processor_chain([
	dup53,
	dup284,
	dup15,
	dup286,
]));

var msg570 = msg("1030", part714);

var part715 = match("MESSAGE#540:1030:01", "nwparser.payload", "%{id}^^Backup started", processor_chain([
	dup53,
	dup15,
	dup286,
]));

var msg571 = msg("1030:01", part715);

var select137 = linear_select([
	msg570,
	msg571,
]);

var msg572 = msg("4097", dup342);

var msg573 = msg("4353", dup342);

var msg574 = msg("5121", dup342);

var msg575 = msg("5122", dup342);

var part716 = match("MESSAGE#545:4609", "nwparser.payload", "%{id}^^Sending Email Failed for following email address [%{user_address}].", processor_chain([
	setc("eventcategory","1207010200"),
	setc("event_description","Sending Email Failed"),
	dup15,
]));

var msg576 = msg("4609", part716);

var msg577 = msg("4868", dup343);

var msg578 = msg("5377", dup343);

var msg579 = msg("5378", dup343);

var msg580 = msg("302449153", dup344);

var msg581 = msg("302449153:01", dup345);

var select138 = linear_select([
	msg580,
	msg581,
]);

var msg582 = msg("302449154", dup344);

var msg583 = msg("302449154:01", dup345);

var select139 = linear_select([
	msg582,
	msg583,
]);

var msg584 = msg("302449155", dup346);

var msg585 = msg("302449155:01", dup347);

var select140 = linear_select([
	msg584,
	msg585,
]);

var msg586 = msg("302449156", dup346);

var msg587 = msg("302449156:01", dup347);

var select141 = linear_select([
	msg586,
	msg587,
]);

var msg588 = msg("302449158", dup344);

var msg589 = msg("302449158:01", dup345);

var select142 = linear_select([
	msg588,
	msg589,
]);

var part717 = match("MESSAGE#559:302449166", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup165,
	dup15,
	dup287,
]));

var msg590 = msg("302449166", part717);

var part718 = match("MESSAGE#560:302449166:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup165,
	dup15,
	dup287,
]));

var msg591 = msg("302449166:01", part718);

var select143 = linear_select([
	msg590,
	msg591,
]);

var part719 = match("MESSAGE#561:302449168", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup136,
	dup288,
	dup56,
	dup22,
	dup15,
	dup287,
]));

var msg592 = msg("302449168", part719);

var part720 = match("MESSAGE#562:302449168:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup136,
	dup288,
	dup56,
	dup22,
	dup15,
	dup287,
]));

var msg593 = msg("302449168:01", part720);

var select144 = linear_select([
	msg592,
	msg593,
]);

var msg594 = msg("302449169", dup344);

var msg595 = msg("302449169:01", dup345);

var select145 = linear_select([
	msg594,
	msg595,
]);

var part721 = match("MESSAGE#565:302449176", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup213,
	dup288,
	dup172,
	dup22,
	dup15,
	dup287,
]));

var msg596 = msg("302449176", part721);

var part722 = match("MESSAGE#566:302449176:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup213,
	dup288,
	dup172,
	dup22,
	dup15,
	dup287,
]));

var msg597 = msg("302449176:01", part722);

var select146 = linear_select([
	msg596,
	msg597,
]);

var part723 = match("MESSAGE#567:302449178", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup256,
	dup15,
	dup287,
]));

var msg598 = msg("302449178", part723);

var part724 = match("MESSAGE#568:302449178:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup256,
	dup15,
	dup287,
]));

var msg599 = msg("302449178:01", part724);

var select147 = linear_select([
	msg598,
	msg599,
]);

var msg600 = msg("302449409", dup344);

var msg601 = msg("302449409:01", dup345);

var select148 = linear_select([
	msg600,
	msg601,
]);

var msg602 = msg("302449410", dup346);

var msg603 = msg("302449410:01", dup347);

var select149 = linear_select([
	msg602,
	msg603,
]);

var part725 = match("MESSAGE#573:302449412", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup289,
	dup15,
	dup287,
]));

var msg604 = msg("302449412", part725);

var part726 = match("MESSAGE#574:302449412:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup289,
	dup15,
	dup287,
]));

var msg605 = msg("302449412:01", part726);

var select150 = linear_select([
	msg604,
	msg605,
]);

var part727 = match("MESSAGE#575:302449413", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup232,
	dup15,
	dup287,
]));

var msg606 = msg("302449413", part727);

var part728 = match("MESSAGE#576:302449413:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup232,
	dup15,
	dup287,
]));

var msg607 = msg("302449413:01", part728);

var select151 = linear_select([
	msg606,
	msg607,
]);

var msg608 = msg("302449414", dup344);

var msg609 = msg("302449414:01", dup345);

var select152 = linear_select([
	msg608,
	msg609,
]);

var msg610 = msg("302449415", dup344);

var msg611 = msg("302449415:01", dup345);

var select153 = linear_select([
	msg610,
	msg611,
]);

var msg612 = msg("302449418", dup344);

var msg613 = msg("302449418:01", dup345);

var select154 = linear_select([
	msg612,
	msg613,
]);

var msg614 = msg("302449420", dup344);

var msg615 = msg("302449420:01", dup345);

var select155 = linear_select([
	msg614,
	msg615,
]);

var msg616 = msg("302450432", dup348);

var msg617 = msg("302450432:01", dup349);

var select156 = linear_select([
	msg616,
	msg617,
]);

var msg618 = msg("302450688", dup344);

var msg619 = msg("302450688:01", dup345);

var select157 = linear_select([
	msg618,
	msg619,
]);

var msg620 = msg("302450944", dup344);

var msg621 = msg("302450944:01", dup345);

var select158 = linear_select([
	msg620,
	msg621,
]);

var msg622 = msg("302452736", dup344);

var msg623 = msg("302452736:01", dup345);

var select159 = linear_select([
	msg622,
	msg623,
]);

var msg624 = msg("302452743", dup344);

var msg625 = msg("302452743:01", dup345);

var select160 = linear_select([
	msg624,
	msg625,
]);

var msg626 = msg("302452758", dup348);

var msg627 = msg("302452758:01", dup349);

var select161 = linear_select([
	msg626,
	msg627,
]);

var msg628 = msg("302452801", dup348);

var msg629 = msg("302452801:01", dup349);

var select162 = linear_select([
	msg628,
	msg629,
]);

var msg630 = msg("302452802", dup344);

var msg631 = msg("302452802:01", dup345);

var select163 = linear_select([
	msg630,
	msg631,
]);

var msg632 = msg("302452807", dup344);

var msg633 = msg("302452807:01", dup345);

var select164 = linear_select([
	msg632,
	msg633,
]);

var msg634 = msg("302452808", dup348);

var msg635 = msg("302452808:01", dup349);

var select165 = linear_select([
	msg634,
	msg635,
]);

var msg636 = msg("302452816", dup344);

var msg637 = msg("302452816:01", dup345);

var select166 = linear_select([
	msg636,
	msg637,
]);

var msg638 = msg("302452817", dup344);

var msg639 = msg("302452817:01", dup345);

var select167 = linear_select([
	msg638,
	msg639,
]);

var msg640 = msg("302452819", dup344);

var msg641 = msg("302452819:01", dup345);

var select168 = linear_select([
	msg640,
	msg641,
]);

var msg642 = msg("302710785", dup348);

var msg643 = msg("302710785:01", dup349);

var select169 = linear_select([
	msg642,
	msg643,
]);

var msg644 = msg("302710786", dup344);

var msg645 = msg("302710786:01", dup345);

var select170 = linear_select([
	msg644,
	msg645,
]);

var msg646 = msg("302710790", dup344);

var msg647 = msg("302710790:01", dup345);

var select171 = linear_select([
	msg646,
	msg647,
]);

var msg648 = msg("302710791", dup348);

var msg649 = msg("302710791:01", dup349);

var select172 = linear_select([
	msg648,
	msg649,
]);

var msg650 = msg("302776321", dup348);

var msg651 = msg("302776321:01", dup349);

var select173 = linear_select([
	msg650,
	msg651,
]);

var msg652 = msg("302776322", dup348);

var msg653 = msg("302776322:01", dup349);

var select174 = linear_select([
	msg652,
	msg653,
]);

var msg654 = msg("302776576", dup344);

var msg655 = msg("302776576:01", dup345);

var select175 = linear_select([
	msg654,
	msg655,
]);

var msg656 = msg("302776834", dup344);

var msg657 = msg("302776834:01", dup345);

var select176 = linear_select([
	msg656,
	msg657,
]);

var msg658 = msg("303077785", dup348);

var msg659 = msg("303077785:01", dup349);

var select177 = linear_select([
	msg658,
	msg659,
]);

var msg660 = msg("303169538", dup348);

var msg661 = msg("303169538:01", dup349);

var select178 = linear_select([
	msg660,
	msg661,
]);

var msg662 = msg("303235073", dup348);

var msg663 = msg("303235073:01", dup349);

var select179 = linear_select([
	msg662,
	msg663,
]);

var msg664 = msg("303235074", dup348);

var msg665 = msg("303235074:01", dup349);

var select180 = linear_select([
	msg664,
	msg665,
]);

var msg666 = msg("303235075", dup344);

var msg667 = msg("303235075:01", dup345);

var select181 = linear_select([
	msg666,
	msg667,
]);

var msg668 = msg("303235079", dup344);

var msg669 = msg("303235079:01", dup345);

var select182 = linear_select([
	msg668,
	msg669,
]);

var part729 = match("MESSAGE#639:303235080/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{p0}");

var all185 = all_match({
	processors: [
		part729,
		dup350,
		dup293,
	],
	on_success: processor_chain([
		dup43,
		dup15,
		dup287,
	]),
});

var msg670 = msg("303235080", all185);

var part730 = match("MESSAGE#640:303235080:01/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{p0}");

var all186 = all_match({
	processors: [
		part730,
		dup350,
		dup293,
	],
	on_success: processor_chain([
		dup43,
		dup15,
		dup287,
	]),
});

var msg671 = msg("303235080:01", all186);

var select183 = linear_select([
	msg670,
	msg671,
]);

var msg672 = msg("303235081", dup344);

var msg673 = msg("303235081:01", dup345);

var select184 = linear_select([
	msg672,
	msg673,
]);

var msg674 = msg("303235082", dup344);

var msg675 = msg("303235082:01", dup345);

var select185 = linear_select([
	msg674,
	msg675,
]);

var msg676 = msg("303235083", dup344);

var msg677 = msg("303235083:01", dup345);

var select186 = linear_select([
	msg676,
	msg677,
]);

var msg678 = msg("302452762", dup344);

var msg679 = msg("303235076", dup344);

var msg680 = msg("303235076:01", dup345);

var select187 = linear_select([
	msg679,
	msg680,
]);

var msg681 = msg("302448900", dup345);

var part731 = match("MESSAGE#651:301", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^Block all other IP traffic and log^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup351,
	dup268,
	dup297,
	dup298,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg682 = msg("301", part731);

var part732 = match("MESSAGE#652:301:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^Block all other IP traffic and log^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup351,
	dup268,
	dup297,
	dup298,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg683 = msg("301:01", part732);

var part733 = match("MESSAGE#653:301:02", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup120,
	dup295,
	dup268,
	dup351,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg684 = msg("301:02", part733);

var select188 = linear_select([
	msg682,
	msg683,
	msg684,
]);

var part734 = match("MESSAGE#654:302", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup303,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg685 = msg("302", part734);

var part735 = match("MESSAGE#655:302:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup303,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg686 = msg("302:01", part735);

var select189 = linear_select([
	msg685,
	msg686,
]);

var part736 = match("MESSAGE#656:306", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup297,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg687 = msg("306", part736);

var part737 = match("MESSAGE#657:306:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup297,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg688 = msg("306:01", part737);

var select190 = linear_select([
	msg687,
	msg688,
]);

var part738 = match("MESSAGE#658:307", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup304,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg689 = msg("307", part738);

var part739 = match("MESSAGE#659:307:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup304,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg690 = msg("307:01", part739);

var select191 = linear_select([
	msg689,
	msg690,
]);

var part740 = match("MESSAGE#660:308", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^Block all other IP traffic and log^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup297,
	dup298,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg691 = msg("308", part740);

var part741 = match("MESSAGE#661:308:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^Block all other IP traffic and log^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup297,
	dup298,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg692 = msg("308:01", part741);

var part742 = match("MESSAGE#662:308:02", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{id}^^%{saddr_v6}^^%{daddr_v6}^^%{saddr}^^%{daddr}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{sport}^^%{dport}^^%{fld14}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{fld17}^^%{rule}^^%{rulename}^^%{fld18}^^%{fld19}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup120,
	dup295,
	dup351,
	dup268,
	dup15,
	dup352,
	dup312,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg693 = msg("308:02", part742);

var select192 = linear_select([
	msg691,
	msg692,
	msg693,
]);

var part743 = match("MESSAGE#663:202", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup36,
	dup295,
	setc("ec_activity","Scan"),
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg694 = msg("202", part743);

var msg695 = msg("206", dup357);

var msg696 = msg("206:01", dup358);

var select193 = linear_select([
	msg695,
	msg696,
]);

var msg697 = msg("207", dup357);

var msg698 = msg("207:01", dup358);

var select194 = linear_select([
	msg697,
	msg698,
]);

var part744 = match("MESSAGE#668:208", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup36,
	dup295,
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg699 = msg("208", part744);

var msg700 = msg("210", dup359);

var part745 = match("MESSAGE#670:210:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup43,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg701 = msg("210:01", part745);

var select195 = linear_select([
	msg700,
	msg701,
]);

var msg702 = msg("211", dup357);

var msg703 = msg("211:01", dup358);

var select196 = linear_select([
	msg702,
	msg703,
]);

var msg704 = msg("221", dup359);

var part746 = match("MESSAGE#674:238/2", "nwparser.p0", "%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}");

var all187 = all_match({
	processors: [
		dup305,
		dup350,
		part746,
	],
	on_success: processor_chain([
		dup43,
		dup15,
		dup353,
		dup354,
		dup287,
		dup300,
		dup301,
		dup302,
	]),
});

var msg705 = msg("238", all187);

var part747 = match("MESSAGE#675:238:01/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{p0}");

var part748 = match("MESSAGE#675:238:01/2", "nwparser.p0", "%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}");

var all188 = all_match({
	processors: [
		part747,
		dup350,
		part748,
	],
	on_success: processor_chain([
		dup43,
		dup15,
		dup353,
		dup354,
		dup287,
		dup300,
		dup301,
		dup302,
	]),
});

var msg706 = msg("238:01", all188);

var select197 = linear_select([
	msg705,
	msg706,
]);

var msg707 = msg("501", dup360);

var msg708 = msg("501:01", dup361);

var select198 = linear_select([
	msg707,
	msg708,
]);

var part749 = match("MESSAGE#678:502", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{username}^^%{sdomain}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}^^%{fld31}^^%{filename_size}^^%{fld32}^^%{fld33}", processor_chain([
	dup43,
	dup15,
	dup356,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));

var msg709 = msg("502", part749);

var part750 = match("MESSAGE#679:502:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{username}^^%{sdomain}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}", processor_chain([
	dup43,
	dup15,
	dup356,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));

var msg710 = msg("502:01", part750);

var select199 = linear_select([
	msg709,
	msg710,
]);

var msg711 = msg("999", dup360);

var msg712 = msg("999:01", dup361);

var select200 = linear_select([
	msg711,
	msg712,
]);

var part751 = match("MESSAGE#682:Application_45", "nwparser.payload", "Application,rn=%{fld1->} cid=%{fld2->} eid=%{fld3},%{fld4->} %{fld5},%{fld6},Symantec AntiVirus,SYSTEM,Information,%{shost},%{event_description}. string-data=[ Scan type: %{event_type->} Event: %{result->} Security risk detected: %{directory->} File: %{filename->} Location: %{fld7->} Computer: %{fld8->} User: %{username->} Action taken:%{action->} Date found: %{fld9}]", processor_chain([
	dup43,
	dup15,
	dup55,
]));

var msg713 = msg("Application_45", part751);

var part752 = match("MESSAGE#692:SYLINK/0", "nwparser.payload", "Using Group Update Provider type: %{p0}");

var part753 = match("MESSAGE#692:SYLINK/1_0", "nwparser.p0", "Single Group Update Provider,Event time:%{fld17->} %{fld18}");

var part754 = match("MESSAGE#692:SYLINK/1_1", "nwparser.p0", "Multiple Group Update Providers,Event time:%{fld17->} %{fld18}");

var part755 = match("MESSAGE#692:SYLINK/1_2", "nwparser.p0", "Mapped Group Update Providers,Event time:%{fld17->} %{fld18}");

var part756 = match("MESSAGE#692:SYLINK/1_3", "nwparser.p0", "Single Group Update Provider%{}");

var part757 = match("MESSAGE#692:SYLINK/1_4", "nwparser.p0", "Multiple Group Update Providers%{}");

var part758 = match("MESSAGE#692:SYLINK/1_5", "nwparser.p0", "Mapped Group Update Providers%{}");

var select201 = linear_select([
	part753,
	part754,
	part755,
	part756,
	part757,
	part758,
]);

var all189 = all_match({
	processors: [
		part752,
		select201,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup14,
		dup15,
		dup93,
		setc("event_description","Using Group Update Provider."),
	]),
});

var msg714 = msg("SYLINK", all189);

var part759 = match("MESSAGE#703:242", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description->} [name]:%{obj_name->} [class]:%{obj_type->} [guid]:%{hardware_id->} [deviceID]:%{info}^^%{fld79}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup53,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg715 = msg("242", part759);

var part760 = match("MESSAGE#704:242:01/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description->} [%{p0}");

var part761 = match("MESSAGE#704:242:01/1_0", "nwparser.p0", "Device]: %{device->} [guid]: %{hardware_id->} [Volume]:%{p0}");

var part762 = match("MESSAGE#704:242:01/1_1", "nwparser.p0", "Volume]:%{p0}");

var select202 = linear_select([
	part761,
	part762,
]);

var part763 = match("MESSAGE#704:242:01/2", "nwparser.p0", "%{} %{disk_volume->} [Vendor]:%{devvendor->} [Model]: %{product->} [Access]: %{accesses}^^%{fld79}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}");

var all190 = all_match({
	processors: [
		part760,
		select202,
		part763,
	],
	on_success: processor_chain([
		dup53,
		dup15,
		dup353,
		dup354,
		dup287,
		dup300,
		dup301,
		dup302,
	]),
});

var msg716 = msg("242:01", all190);

var part764 = match("MESSAGE#705:242:02", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description->} [Volume]: %{disk_volume->} [Model]: %{product->} [Access]: %{accesses}^^%{fld79}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup53,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var msg717 = msg("242:02", part764);

var part765 = match("MESSAGE#706:242:03/1_0", "nwparser.p0", "%{event_description}. %{info->} [Access]: %{accesses}^^%{p0}");

var part766 = match("MESSAGE#706:242:03/1_1", "nwparser.p0", " %{event_description}. %{info}^^%{p0}");

var part767 = match("MESSAGE#706:242:03/1_2", "nwparser.p0", " %{event_description}^^%{p0}");

var select203 = linear_select([
	part765,
	part766,
	part767,
]);

var part768 = match("MESSAGE#706:242:03/2", "nwparser.p0", "%{fld79}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}");

var all191 = all_match({
	processors: [
		dup305,
		select203,
		part768,
	],
	on_success: processor_chain([
		dup53,
		dup15,
		dup353,
		dup354,
		dup287,
		dup300,
		dup301,
		dup302,
	]),
});

var msg718 = msg("242:03", all191);

var select204 = linear_select([
	msg715,
	msg716,
	msg717,
	msg718,
]);

var part769 = match("MESSAGE#707:303169540", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	setc("eventcategory","1801010000"),
	dup15,
	dup287,
]));

var msg719 = msg("303169540", part769);

var part770 = match("MESSAGE#708:Remote::01", "nwparser.payload", "%{shost}, Remote: %{fld4},Remote: %{daddr},Remote: %{fld5},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},Domain: %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL:%{url},Intrusion Payload URL:%{fld25}", processor_chain([
	dup53,
	dup12,
	dup15,
	dup40,
	dup41,
	dup42,
	dup47,
]));

var msg720 = msg("Remote::01", part770);

var part771 = match("MESSAGE#709:Notification::01/0_0", "nwparser.payload", "\"%{info}\",Local: %{p0}");

var part772 = match("MESSAGE#709:Notification::01/0_1", "nwparser.payload", "%{info},Local: %{p0}");

var select205 = linear_select([
	part771,
	part772,
]);

var part773 = match("MESSAGE#709:Notification::01/1", "nwparser.p0", "%{saddr},Local: %{fld1},Remote: %{fld9},Remote: %{daddr},Remote: %{fld3},Unknown,OTHERS,,Begin: %{fld50->} %{fld52},End: %{fld51->} %{fld53},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld6},User: %{username},%{p0}");

var select206 = linear_select([
	dup182,
	dup67,
]);

var part774 = match("MESSAGE#709:Notification::01/3", "nwparser.p0", "%{} %{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var all192 = all_match({
	processors: [
		select205,
		part773,
		select206,
		part774,
	],
	on_success: processor_chain([
		dup53,
		dup12,
		dup13,
		dup15,
		dup14,
		dup40,
		dup41,
		dup42,
		dup47,
	]),
});

var msg721 = msg("Notification::01", all192);

var chain1 = processor_chain([
	select2,
	msgid_select({
		"\"=======EXCEPTION:": msg507,
		"1": msg545,
		"10": msg554,
		"1029": select136,
		"1030": select137,
		"1281": msg555,
		"2": msg546,
		"202": msg694,
		"206": select193,
		"207": select194,
		"208": msg699,
		"210": select195,
		"211": select196,
		"221": msg704,
		"238": select197,
		"242": select204,
		"257": msg556,
		"259": msg557,
		"264": msg558,
		"265": msg559,
		"273": msg560,
		"275": msg561,
		"3": msg547,
		"301": select188,
		"302": select189,
		"302448900": msg681,
		"302449153": select138,
		"302449154": select139,
		"302449155": select140,
		"302449156": select141,
		"302449158": select142,
		"302449166": select143,
		"302449168": select144,
		"302449169": select145,
		"302449176": select146,
		"302449178": select147,
		"302449409": select148,
		"302449410": select149,
		"302449412": select150,
		"302449413": select151,
		"302449414": select152,
		"302449415": select153,
		"302449418": select154,
		"302449420": select155,
		"302450432": select156,
		"302450688": select157,
		"302450944": select158,
		"302452736": select159,
		"302452743": select160,
		"302452758": select161,
		"302452762": msg678,
		"302452801": select162,
		"302452802": select163,
		"302452807": select164,
		"302452808": select165,
		"302452816": select166,
		"302452817": select167,
		"302452819": select168,
		"302710785": select169,
		"302710786": select170,
		"302710790": select171,
		"302710791": select172,
		"302776321": select173,
		"302776322": select174,
		"302776576": select175,
		"302776834": select176,
		"303077785": select177,
		"303169538": select178,
		"303169540": msg719,
		"303235073": select179,
		"303235074": select180,
		"303235075": select181,
		"303235076": select187,
		"303235079": select182,
		"303235080": select183,
		"303235081": select184,
		"303235082": select185,
		"303235083": select186,
		"306": select190,
		"307": select191,
		"308": select192,
		"4": msg548,
		"4097": msg572,
		"4353": msg573,
		"4609": msg576,
		"4868": msg577,
		"5": msg549,
		"501": select198,
		"502": select199,
		"5121": msg574,
		"5122": msg575,
		"5377": msg578,
		"5378": msg579,
		"6": msg550,
		"7": msg551,
		"769": msg562,
		"772": msg563,
		"773": msg564,
		"778": msg565,
		"779": msg566,
		"782": msg567,
		"8": msg552,
		"9": msg553,
		"999": select200,
		"??:": msg214,
		"Active": select3,
		"Add": msg69,
		"Administrator": select4,
		"Allowed": select123,
		"Antivirus": select9,
		"Application": select12,
		"Application_45": msg713,
		"Applied": select16,
		"Audit": msg519,
		"Blocked": select21,
		"Category": select127,
		"Changed": msg124,
		"Cleaned": msg125,
		"Client": select23,
		"Commercial": select28,
		"Compliance": msg484,
		"Compressed": select99,
		"Computer": select31,
		"Configuration": select32,
		"Connected": select34,
		"Connection": msg160,
		"Continue": select133,
		"Could": select35,
		"Create": msg163,
		"Database": select36,
		"Decomposer": msg171,
		"Default": select128,
		"Device": select130,
		"Disconnected": select37,
		"Domain": select38,
		"Download": msg485,
		"Email": msg528,
		"Failed": select41,
		"Firefox": select67,
		"Firewall": select42,
		"Generic": select70,
		"Group": select43,
		"Host": select46,
		"Internet": select68,
		"Intrusion": select48,
		"Invalid": msg217,
		"LUALL": msg289,
		"Limited": msg218,
		"LiveUpdate": select54,
		"Local": select60,
		"Local:": select18,
		"Location": msg288,
		"Malicious": select8,
		"Management": select62,
		"Memory": msg327,
		"Network": select66,
		"New": select73,
		"No": select74,
		"Notification:": msg721,
		"Number": select77,
		"Organization": select75,
		"PTS": msg380,
		"Ping": select134,
		"Policy": select79,
		"Potential": select80,
		"Previous": msg368,
		"Proactive": select81,
		"Received": select87,
		"Reconfiguring": msg383,
		"Reconnected": msg384,
		"Remote:": msg720,
		"Retry": select88,
		"Risk": select90,
		"SHA-256:": select15,
		"Scan": select94,
		"Security": select98,
		"Server": select135,
		"Somebody": select10,
		"Stop": select100,
		"Stopping": msg427,
		"Submission": msg428,
		"Successfully": select89,
		"Suspicious": msg476,
		"Switched": msg429,
		"Symantec": select107,
		"System": select108,
		"Terminated": msg483,
		"Traffic": select111,
		"TruScan": select113,
		"Unexpected": select114,
		"Unsolicited": select115,
		"User": select117,
		"Using": msg714,
		"Virus": select59,
		"Windows": select119,
		"allowed": select6,
		"blocked": select17,
		"client": select26,
		"management": select63,
		"password": select5,
		"process": select82,
		"properties": select85,
		"restart": msg385,
	}),
]);

var part775 = match("MESSAGE#0:Active/1_0", "nwparser.p0", "%{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var part776 = match_copy("MESSAGE#0:Active/1_1", "nwparser.p0", "domain");

var part777 = match("MESSAGE#15:Somebody:01/1_0", "nwparser.p0", "%{domain},Local Port %{sport},Remote Port %{dport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var part778 = match("MESSAGE#27:Application:06/1_0", "nwparser.p0", "\"Intrusion URL: %{url}\",Intrusion Payload URL:%{fld25}");

var part779 = match("MESSAGE#27:Application:06/1_1", "nwparser.p0", "Intrusion URL: %{url},Intrusion Payload URL:%{fld25}");

var part780 = match("MESSAGE#27:Application:06/1_2", "nwparser.p0", "Intrusion URL: %{url}");

var part781 = match("MESSAGE#31:scanning:01/1_0", "nwparser.p0", "%{url},Intrusion Payload URL:%{fld25}");

var part782 = match_copy("MESSAGE#31:scanning:01/1_1", "nwparser.p0", "url");

var part783 = match("MESSAGE#33:Informational/1_1", "nwparser.p0", "Domain:%{p0}");

var part784 = match("MESSAGE#38:Web_Attack:16/1_1", "nwparser.p0", ":%{p0}");

var part785 = match("MESSAGE#307:process:12/1_0", "nwparser.p0", "\"%{p0}");

var part786 = match_copy("MESSAGE#307:process:12/1_1", "nwparser.p0", "p0");

var part787 = match("MESSAGE#307:process:12/4", "nwparser.p0", ",Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{protocol},%{p0}");

var part788 = match("MESSAGE#307:process:12/5_0", "nwparser.p0", "Intrusion ID: %{fld33},Begin: %{p0}");

var part789 = match("MESSAGE#307:process:12/5_1", "nwparser.p0", "%{fld33},Begin: %{p0}");

var part790 = match("MESSAGE#307:process:12/6", "nwparser.p0", "%{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application},Location: %{fld11},User: %{username},Domain: %{domain},Local Port %{dport},Remote Port %{sport},CIDS Signature ID: %{sigid},CIDS Signature string: %{sigid_string},CIDS Signature SubID: %{fld23},Intrusion URL: %{p0}");

var part791 = match("MESSAGE#21:Applied/1_0", "nwparser.p0", ",Event time:%{fld17->} %{fld18}");

var part792 = match_copy("MESSAGE#21:Applied/1_1", "nwparser.p0", "");

var part793 = match("MESSAGE#23:blocked:01/1_0", "nwparser.p0", "\"Location: %{p0}");

var part794 = match("MESSAGE#23:blocked:01/1_1", "nwparser.p0", "Location: %{p0}");

var part795 = match("MESSAGE#52:blocked/2", "nwparser.p0", "%{fld2},User: %{username},Domain: %{domain}");

var part796 = match("MESSAGE#190:Local::01/0_0", "nwparser.payload", "%{fld4},MD-5:%{fld5},Local:%{p0}");

var part797 = match("MESSAGE#190:Local::01/0_1", "nwparser.payload", "Local:%{p0}");

var part798 = match("MESSAGE#192:Local:/1_0", "nwparser.p0", "Rule: %{rulename},Location: %{p0}");

var part799 = match("MESSAGE#192:Local:/1_1", "nwparser.p0", " \"Rule: %{rulename}\",Location: %{p0}");

var part800 = match("MESSAGE#192:Local:/2", "nwparser.p0", "%{fld11},User: %{username},%{p0}");

var part801 = match("MESSAGE#192:Local:/3_0", "nwparser.p0", "Domain: %{domain},Action: %{action}");

var part802 = match("MESSAGE#192:Local:/3_1", "nwparser.p0", " Domain: %{domain}");

var part803 = match("MESSAGE#198:Local::04/1_0", "nwparser.p0", "\"Intrusion URL: %{url}\",Intrusion Payload URL:%{p0}");

var part804 = match("MESSAGE#198:Local::04/1_1", "nwparser.p0", "Intrusion URL: %{url},Intrusion Payload URL:%{p0}");

var part805 = match_copy("MESSAGE#198:Local::04/2", "nwparser.p0", "fld25");

var part806 = match("MESSAGE#205:Local::07/0", "nwparser.payload", "%{event_description},Local: %{daddr},Local: %{fld12},Remote: %{fld13},Remote: %{saddr},Remote: %{fld15},Inbound,%{network_service},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part807 = match("MESSAGE#206:Local::19/0", "nwparser.payload", "%{event_description},Local: %{saddr},Local: %{fld12},Remote: %{fld13},Remote: %{daddr},Remote: %{fld15},Outbound,%{network_service},,Begin: %{fld50->} %{fld54},End: %{fld16->} %{fld19},Occurrences: %{dclass_counter1},Application: %{application}, %{p0}");

var part808 = match("MESSAGE#209:Local::03/2", "nwparser.p0", "%{fld11},User: %{username},Domain: %{domain}");

var part809 = match("MESSAGE#64:client:05/0", "nwparser.payload", "The client will block traffic from IP address %{fld14->} for the next %{duration_string->} (from %{fld13})%{p0}");

var part810 = match("MESSAGE#64:client:05/1_0", "nwparser.p0", ".,%{p0}");

var part811 = match("MESSAGE#64:client:05/1_1", "nwparser.p0", " . ,%{p0}");

var part812 = match("MESSAGE#70:Commercial/0", "nwparser.payload", "Commercial application detected,Computer name: %{p0}");

var part813 = match("MESSAGE#70:Commercial/1_0", "nwparser.p0", "%{shost},IP Address: %{saddr},Detection type: %{p0}");

var part814 = match("MESSAGE#70:Commercial/1_1", "nwparser.p0", "%{shost},Detection type: %{p0}");

var part815 = match("MESSAGE#70:Commercial/2", "nwparser.p0", "%{severity},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld6},Detection score:%{fld7},Submission recommendation: %{fld8},Permitted application reason: %{fld9},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{fld1},%{p0}");

var part816 = match("MESSAGE#70:Commercial/3_0", "nwparser.p0", "\"%{filename}\",Actual action: %{p0}");

var part817 = match("MESSAGE#70:Commercial/3_1", "nwparser.p0", "%{filename},Actual action: %{p0}");

var part818 = match("MESSAGE#70:Commercial/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld19},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var part819 = match("MESSAGE#76:Computer/0", "nwparser.payload", "IP Address: %{hostip},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{p0}");

var part820 = match("MESSAGE#78:Computer:03/1_0", "nwparser.p0", "\"%{filename}\",%{p0}");

var part821 = match("MESSAGE#78:Computer:03/1_1", "nwparser.p0", "%{filename},%{p0}");

var part822 = match("MESSAGE#79:Computer:02/2", "nwparser.p0", "%{severity},First Seen: %{fld55},Application name: %{application},Application type: %{obj_type},Application version:%{version},Hash type:%{encryption_type},Application hash: %{checksum},Company name: %{fld11},File size (bytes): %{filename_size},Sensitivity: %{fld13},Detection score:%{fld7},COH Engine Version: %{fld41},%{fld53},Permitted application reason: %{fld54},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},Risk Level: %{fld50},Detection Source: %{fld52},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{fld22},Actual action: %{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld5->} %{fld6},Inserted:%{fld12},End:%{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var part823 = match("MESSAGE#250:Network:24/1_0", "nwparser.p0", "\"%{}");

var part824 = match("MESSAGE#134:Host:09/1_1", "nwparser.p0", " Domain:%{p0}");

var part825 = match("MESSAGE#135:Intrusion/1_0", "nwparser.p0", "is %{p0}");

var part826 = match("MESSAGE#145:LiveUpdate:10/1_0", "nwparser.p0", ".,Event time:%{fld17->} %{fld18}");

var part827 = match("MESSAGE#179:LiveUpdate:40/1_0", "nwparser.p0", "\",Event time:%{fld17->} %{fld18}");

var part828 = match("MESSAGE#432:Virus:02/1_1", "nwparser.p0", " %{p0}");

var part829 = match("MESSAGE#436:Virus:12/0", "nwparser.payload", "Virus found,IP Address: %{saddr},Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{p0}");

var part830 = match("MESSAGE#436:Virus:12/1_0", "nwparser.p0", "\"%{fld1}\",Actual action: %{p0}");

var part831 = match("MESSAGE#436:Virus:12/1_1", "nwparser.p0", "%{fld1},Actual action: %{p0}");

var part832 = match("MESSAGE#437:Virus:15/1_0", "nwparser.p0", "Intensive Protection Level: %{fld61},Certificate issuer: %{fld60},Certificate signer: %{fld62},Certificate thumbprint: %{fld63},Signing timestamp: %{fld64},Certificate serial number: %{fld65},Source: %{p0}");

var part833 = match("MESSAGE#437:Virus:15/1_1", "nwparser.p0", "Source: %{p0}");

var part834 = match("MESSAGE#438:Virus:13/3_0", "nwparser.p0", "\"Group: %{group}\",Server: %{p0}");

var part835 = match("MESSAGE#438:Virus:13/3_1", "nwparser.p0", "Group: %{group},Server: %{p0}");

var part836 = match("MESSAGE#438:Virus:13/4", "nwparser.p0", "%{hostid},User: %{username},Source computer: %{fld29},Source IP: %{fld31},Disposition: %{result},Download site: %{fld44},Web domain: %{fld45},Downloaded by: %{fld46},Prevalence: %{info},Confidence: %{context},URL Tracking Status: %{fld49},,First Seen: %{fld50},Sensitivity: %{fld52},%{fld56},Application hash: %{checksum},Hash type: %{encryption_type},Company name: %{fld54},Application name: %{application},Application version: %{version},Application type: %{obj_type},File size (bytes): %{p0}");

var part837 = match("MESSAGE#438:Virus:13/5_0", "nwparser.p0", "%{filename_size},Category set: %{category},Category type: %{event_type}");

var part838 = match_copy("MESSAGE#438:Virus:13/5_1", "nwparser.p0", "filename_size");

var part839 = match("MESSAGE#440:Virus:14/0", "nwparser.payload", "Virus found,Computer name: %{shost},Source: %{event_source},Risk name: %{virusname},Occurrences: %{dclass_counter1},%{filename},%{p0}");

var part840 = match("MESSAGE#441:Virus:05/1_0", "nwparser.p0", "\"%{info}\",Actual action: %{p0}");

var part841 = match("MESSAGE#441:Virus:05/1_1", "nwparser.p0", "%{info},Actual action: %{p0}");

var part842 = match("MESSAGE#218:Location/3_0", "nwparser.p0", "%{info},Event time:%{fld17->} %{fld18}");

var part843 = match_copy("MESSAGE#218:Location/3_1", "nwparser.p0", "info");

var part844 = match("MESSAGE#253:Network:27/1_0", "nwparser.p0", " by policy%{}");

var part845 = match("MESSAGE#296:Policy:deleted/1_0", "nwparser.p0", ",%{p0}");

var part846 = match("MESSAGE#298:Potential:02/0", "nwparser.payload", "Potential risk found,Computer name: %{p0}");

var part847 = match("MESSAGE#299:Potential/4", "nwparser.p0", "%{action},Requested action: %{disposition},Secondary action: %{event_state},Event time: %{fld17->} %{fld18},Inserted: %{fld20},End: %{fld51},Domain: %{domain},Group: %{group},Server: %{hostid},User: %{username},Source computer: %{fld29},Source IP: %{saddr}");

var part848 = match("MESSAGE#308:process:03/0", "nwparser.payload", "%{event_description}, process id: %{process_id->} Filename: %{filename->} The change was denied by user%{fld6}\"%{p0}");

var part849 = match("MESSAGE#340:Scan:12/1_0", "nwparser.p0", "'%{context}',%{p0}");

var part850 = match("MESSAGE#343:Security:03/0", "nwparser.payload", "Security risk found,Computer name: %{p0}");

var part851 = match("MESSAGE#345:Security:05/0", "nwparser.payload", "Security risk found,IP Address: %{saddr},Computer name: %{shost},%{p0}");

var part852 = match("MESSAGE#345:Security:05/7_0", "nwparser.p0", "%{filename_size},Category set: %{category},Category type: %{vendor_event_cat}");

var part853 = match("MESSAGE#388:Symantec:26/0", "nwparser.payload", "Category: %{fld22},Symantec AntiVirus,%{p0}");

var part854 = match("MESSAGE#388:Symantec:26/1_0", "nwparser.p0", "[Antivirus%{p0}");

var part855 = match("MESSAGE#388:Symantec:26/1_1", "nwparser.p0", "\"[Antivirus%{p0}");

var part856 = match("MESSAGE#389:Symantec:39/2", "nwparser.p0", "%{} %{p0}");

var part857 = match("MESSAGE#389:Symantec:39/3_0", "nwparser.p0", "detection%{p0}");

var part858 = match("MESSAGE#389:Symantec:39/3_1", "nwparser.p0", "advanced heuristic detection%{p0}");

var part859 = match("MESSAGE#389:Symantec:39/5_0", "nwparser.p0", " Size (bytes): %{filename_size}.\",Event time:%{fld17->} %{fld18}");

var part860 = match("MESSAGE#389:Symantec:39/5_2", "nwparser.p0", "Event time:%{fld17->} %{fld18}");

var part861 = match("MESSAGE#410:Terminated/0_1", "nwparser.payload", ",%{p0}");

var part862 = match("MESSAGE#416:Traffic:02/2", "nwparser.p0", "%{fld6},User: %{username},Domain: %{domain}");

var part863 = match("MESSAGE#455:Allowed:09/2_0", "nwparser.p0", "\"%{filename}\",User: %{p0}");

var part864 = match("MESSAGE#455:Allowed:09/2_1", "nwparser.p0", "%{filename},User: %{p0}");

var part865 = match("MESSAGE#457:Allowed:10/3_0", "nwparser.p0", "%{fld46},File size (%{fld10}): %{filename_size},Device ID: %{device}");

var part866 = match("MESSAGE#505:Ping/0_0", "nwparser.payload", "\"\"%{action->} . Description: %{p0}");

var part867 = match("MESSAGE#505:Ping/0_1", "nwparser.payload", "%{action->} . Description: %{p0}");

var part868 = match("MESSAGE#639:303235080/1_0", "nwparser.p0", "%{event_description->} [name]:%{obj_name->} [class]:%{obj_type->} [guid]:%{hardware_id->} [deviceID]:%{info}^^%{p0}");

var part869 = match("MESSAGE#639:303235080/1_1", "nwparser.p0", "%{event_description}. %{info}^^%{p0}");

var part870 = match("MESSAGE#639:303235080/1_2", "nwparser.p0", "%{event_description}^^%{p0}");

var part871 = match("MESSAGE#639:303235080/2", "nwparser.p0", "%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}");

var part872 = match("MESSAGE#674:238/0", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{p0}");

var select207 = linear_select([
	dup9,
	dup10,
]);

var select208 = linear_select([
	dup50,
	dup10,
]);

var select209 = linear_select([
	dup59,
	dup60,
	dup61,
]);

var select210 = linear_select([
	dup63,
	dup64,
]);

var select211 = linear_select([
	dup76,
	dup77,
]);

var select212 = linear_select([
	dup79,
	dup80,
]);

var select213 = linear_select([
	dup90,
	dup91,
]);

var select214 = linear_select([
	dup98,
	dup99,
]);

var select215 = linear_select([
	dup101,
	dup102,
]);

var select216 = linear_select([
	dup105,
	dup106,
]);

var select217 = linear_select([
	dup108,
	dup109,
]);

var select218 = linear_select([
	dup112,
	dup113,
]);

var select219 = linear_select([
	dup140,
	dup141,
]);

var select220 = linear_select([
	dup146,
	dup147,
]);

var select221 = linear_select([
	dup149,
	dup150,
]);

var select222 = linear_select([
	dup159,
	dup160,
]);

var select223 = linear_select([
	dup198,
	dup199,
]);

var select224 = linear_select([
	dup201,
	dup202,
]);

var select225 = linear_select([
	dup203,
	dup204,
]);

var select226 = linear_select([
	dup206,
	dup207,
]);

var select227 = linear_select([
	dup209,
	dup210,
]);

var select228 = linear_select([
	dup211,
	dup212,
]);

var select229 = linear_select([
	dup216,
	dup91,
]);

var select230 = linear_select([
	dup249,
	dup226,
]);

var select231 = linear_select([
	dup252,
	dup207,
]);

var select232 = linear_select([
	dup262,
	dup261,
]);

var select233 = linear_select([
	dup264,
	dup265,
]);

var select234 = linear_select([
	dup266,
	dup191,
	dup267,
	dup176,
	dup91,
]);

var select235 = linear_select([
	dup275,
	dup276,
]);

var select236 = linear_select([
	dup281,
	dup282,
]);

var part873 = match("MESSAGE#524:1281", "nwparser.payload", "%{id}^^%{event_description}", processor_chain([
	dup53,
	dup15,
]));

var part874 = match("MESSAGE#546:4868", "nwparser.payload", "%{id}^^%{event_description}", processor_chain([
	dup43,
	dup15,
]));

var part875 = match("MESSAGE#549:302449153", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup43,
	dup15,
	dup287,
]));

var part876 = match("MESSAGE#550:302449153:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup43,
	dup15,
	dup287,
]));

var part877 = match("MESSAGE#553:302449155", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup74,
	dup15,
	dup287,
]));

var part878 = match("MESSAGE#554:302449155:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup74,
	dup15,
	dup287,
]));

var part879 = match("MESSAGE#585:302450432", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup168,
	dup15,
	dup287,
]));

var part880 = match("MESSAGE#586:302450432:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{id}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{event_source}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup168,
	dup15,
	dup287,
]));

var select237 = linear_select([
	dup290,
	dup291,
	dup292,
]);

var part881 = match("MESSAGE#664:206", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var part882 = match("MESSAGE#665:206:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}", processor_chain([
	dup294,
	dup295,
	dup37,
	dup268,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var part883 = match("MESSAGE#669:210", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{id}^^%{saddr}^^%{daddr}^^%{smacaddr}^^%{dmacaddr}^^%{zone}^^%{username}^^%{sdomain}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{dhost}^^%{fld13}^^%{fld14}^^%{fld29}^^%{fld15}^^%{fld16}^^%{dclass_counter1}^^%{application}^^%{event_description}^^%{fld17}^^%{fld18}^^%{fld19}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{saddr_v6}^^%{daddr_v6}^^%{sport}^^%{dport}^^%{sigid}^^%{sigid_string}^^%{sigid1}^^%{url}^^%{web_referer}^^%{fld30}^^%{version}^^%{policy_id}", processor_chain([
	dup43,
	dup15,
	dup353,
	dup354,
	dup287,
	dup300,
	dup301,
	dup302,
]));

var part884 = match("MESSAGE#676:501", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{saddr}^^%{username}^^%{sdomain}^^%{hostname}^^%{group}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}^^%{fld31}^^%{filename_size}^^%{fld32}^^%{fld33}", processor_chain([
	dup43,
	dup15,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));

var part885 = match("MESSAGE#677:501:01", "nwparser.payload", "%{fld1}^^%{domain}^^%{fld3}^^%{id}^^%{username}^^%{sdomain}^^%{fld6}^^%{fld7}^^%{fld8}^^%{severity}^^%{fld9}^^%{fld10}^^%{shost}^^%{fld11}^^%{fld12}^^%{event_description}^^%{fld13}^^%{fld14}^^%{fld15}^^%{fld16}^^%{rule}^^%{rulename}^^%{parent_pid}^^%{parent_process}^^%{fld17}^^%{fld18}^^%{param}^^%{fld20}^^%{fld21}^^%{fld22}^^%{fld23}^^%{fld24}^^%{fld25}^^%{fld26}^^%{fld27}^^%{fld28}^^%{fld29}^^%{dclass_counter1}^^%{fld30}", processor_chain([
	dup43,
	dup15,
	dup355,
	dup287,
	dup300,
	dup301,
	dup307,
	dup308,
]));
