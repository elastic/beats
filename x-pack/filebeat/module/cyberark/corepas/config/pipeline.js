//  Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
//  or more contributor license agreements. Licensed under the Elastic License;
//  you may not use this file except in compliance with the Elastic License.
var tvm = {
	pair_separator: ";",
	kv_separator: "=",
	open_quote: "\"",
	close_quote: "\"",
};

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

var dup1 = setc("eventcategory","1501040000");

var dup2 = setf("msg","$MSG");

var dup3 = setf("id","messageid");

var dup4 = setc("eventcategory","1605020000");

var dup5 = setc("eventcategory","1401030000");

var dup6 = setc("ec_subject","User");

var dup7 = setc("ec_activity","Logon");

var dup8 = setc("ec_theme","Authentication");

var dup9 = setc("ec_outcome","Failure");

var dup10 = setc("eventcategory","1401060000");

var dup11 = setc("ec_outcome","Success");

var dup12 = setc("eventcategory","1401070000");

var dup13 = setc("ec_activity","Logoff");

var dup14 = setc("ec_theme","Policy");

var dup15 = setc("eventcategory","1803000000");

var dup16 = setc("ec_subject","NetworkComm");

var dup17 = setc("ec_theme","Communication");

var dup18 = setc("ec_theme","AccessControl");

var dup19 = setc("eventcategory","1801000000");

var dup20 = setc("eventcategory","1801020000");

var dup21 = setc("eventcategory","1609000000");

var dup22 = setc("eventcategory","1603050000");

var dup23 = setc("eventcategory","1612010000");

var dup24 = date_time({
	dest: "event_time",
	args: ["hdatetime"],
	fmts: [
		[dW,dc("-"),dM,dc("-"),dD,dc("T"),dZ,dc("Z")],
	],
});

var dup25 = date_time({
	dest: "event_time",
	args: ["hmonth","hday","htime"],
	fmts: [
		[dB,dD,dZ],
	],
});

var dup26 = setc("eventcategory","1612000000");

var dup27 = setc("eventcategory","1303000000");

var dup28 = setc("ec_outcome","Error");

var dup29 = setc("ec_activity","Disable");

var dup30 = setc("eventcategory","1401050200");

var dup31 = match("MESSAGE#568:300:02/0", "nwparser.payload", "Version=%{p0}");

var dup32 = match("MESSAGE#568:300:02/1_0", "nwparser.p0", "\"%{version}\";Message=%{p0}");

var dup33 = match("MESSAGE#568:300:02/1_1", "nwparser.p0", "%{version};Message=%{p0}");

var dup34 = match("MESSAGE#568:300:02/2_0", "nwparser.p0", "\"%{action}\";Issuer=%{p0}");

var dup35 = match("MESSAGE#568:300:02/2_1", "nwparser.p0", "%{action};Issuer=%{p0}");

var dup36 = match("MESSAGE#568:300:02/3_0", "nwparser.p0", "\"%{username}\";Station=%{p0}");

var dup37 = match("MESSAGE#568:300:02/3_1", "nwparser.p0", "%{username};Station=%{p0}");

var dup38 = match("MESSAGE#568:300:02/4_0", "nwparser.p0", "\"%{hostip}\";File=%{p0}");

var dup39 = match("MESSAGE#568:300:02/4_1", "nwparser.p0", "%{hostip};File=%{p0}");

var dup40 = match("MESSAGE#568:300:02/5_0", "nwparser.p0", "\"%{filename}\";Safe=%{p0}");

var dup41 = match("MESSAGE#568:300:02/5_1", "nwparser.p0", "%{filename};Safe=%{p0}");

var dup42 = match("MESSAGE#568:300:02/6_0", "nwparser.p0", "\"%{group_object}\";Location=%{p0}");

var dup43 = match("MESSAGE#568:300:02/6_1", "nwparser.p0", "%{group_object};Location=%{p0}");

var dup44 = match("MESSAGE#568:300:02/7_0", "nwparser.p0", "\"%{directory}\";Category=%{p0}");

var dup45 = match("MESSAGE#568:300:02/7_1", "nwparser.p0", "%{directory};Category=%{p0}");

var dup46 = match("MESSAGE#568:300:02/8_0", "nwparser.p0", "\"%{category}\";RequestId=%{p0}");

var dup47 = match("MESSAGE#568:300:02/8_1", "nwparser.p0", "%{category};RequestId=%{p0}");

var dup48 = match("MESSAGE#568:300:02/9_0", "nwparser.p0", "\"%{id1}\";Reason=%{p0}");

var dup49 = match("MESSAGE#568:300:02/9_1", "nwparser.p0", "%{id1};Reason=%{p0}");

var dup50 = match("MESSAGE#568:300:02/10_0", "nwparser.p0", "\"%{event_description}\";Severity=%{p0}");

var dup51 = match("MESSAGE#568:300:02/10_1", "nwparser.p0", "%{event_description};Severity=%{p0}");

var dup52 = match("MESSAGE#568:300:02/11_0", "nwparser.p0", "\"%{severity}\";SourceUser=%{p0}");

var dup53 = match("MESSAGE#568:300:02/11_1", "nwparser.p0", "%{severity};SourceUser=%{p0}");

var dup54 = match("MESSAGE#568:300:02/12_0", "nwparser.p0", "\"%{group}\";TargetUser=%{p0}");

var dup55 = match("MESSAGE#568:300:02/12_1", "nwparser.p0", "%{group};TargetUser=%{p0}");

var dup56 = match("MESSAGE#568:300:02/13_0", "nwparser.p0", "\"%{uid}\";GatewayStation=%{p0}");

var dup57 = match("MESSAGE#568:300:02/13_1", "nwparser.p0", "%{uid};GatewayStation=%{p0}");

var dup58 = match("MESSAGE#568:300:02/14_0", "nwparser.p0", "\"%{saddr}\";TicketID=%{p0}");

var dup59 = match("MESSAGE#568:300:02/14_1", "nwparser.p0", "%{saddr};TicketID=%{p0}");

var dup60 = match("MESSAGE#568:300:02/15_0", "nwparser.p0", "\"%{operation_id}\";PolicyID=%{p0}");

var dup61 = match("MESSAGE#568:300:02/15_1", "nwparser.p0", "%{operation_id};PolicyID=%{p0}");

var dup62 = match("MESSAGE#568:300:02/16_0", "nwparser.p0", "\"%{policyname}\";UserName=%{p0}");

var dup63 = match("MESSAGE#568:300:02/16_1", "nwparser.p0", "%{policyname};UserName=%{p0}");

var dup64 = match("MESSAGE#568:300:02/17_0", "nwparser.p0", "\"%{fld11}\";LogonDomain=%{p0}");

var dup65 = match("MESSAGE#568:300:02/17_1", "nwparser.p0", "%{fld11};LogonDomain=%{p0}");

var dup66 = match("MESSAGE#568:300:02/18_0", "nwparser.p0", "\"%{domain}\";Address=%{p0}");

var dup67 = match("MESSAGE#568:300:02/18_1", "nwparser.p0", "%{domain};Address=%{p0}");

var dup68 = match("MESSAGE#568:300:02/19_0", "nwparser.p0", "\"%{fld14}\";CPMStatus=%{p0}");

var dup69 = match("MESSAGE#568:300:02/19_1", "nwparser.p0", "%{fld14};CPMStatus=%{p0}");

var dup70 = match("MESSAGE#568:300:02/20_0", "nwparser.p0", "\"%{disposition}\";Port=%{p0}");

var dup71 = match("MESSAGE#568:300:02/20_1", "nwparser.p0", "%{disposition};Port=%{p0}");

var dup72 = match("MESSAGE#568:300:02/21_0", "nwparser.p0", "\"%{dport}\";Database=%{p0}");

var dup73 = match("MESSAGE#568:300:02/21_1", "nwparser.p0", "%{dport};Database=%{p0}");

var dup74 = match("MESSAGE#568:300:02/22_0", "nwparser.p0", "\"%{db_name}\";DeviceType=%{p0}");

var dup75 = match("MESSAGE#568:300:02/22_1", "nwparser.p0", "%{db_name};DeviceType=%{p0}");

var dup76 = match("MESSAGE#568:300:02/23_0", "nwparser.p0", "\"%{obj_type}\";ExtraDetails=\"ApplicationType=%{p0}");

var dup77 = match("MESSAGE#568:300:02/23_1", "nwparser.p0", "%{obj_type};ExtraDetails=\"ApplicationType=%{p0}");

var dup78 = setc("eventcategory","1502000000");

var dup79 = setc("eventcategory","1402040100");

var dup80 = setc("ec_activity","Modify");

var dup81 = setc("ec_theme","Password");

var dup82 = setc("eventcategory","1608000000");

var dup83 = setc("eventcategory","1501000000");

var dup84 = setc("eventcategory","1206000000");

var dup85 = match("MESSAGE#621:411/1_0", "nwparser.p0", "\"%{version}\";%{p0}");

var dup86 = match("MESSAGE#621:411/1_1", "nwparser.p0", "%{version};%{p0}");

var dup87 = match("MESSAGE#621:411/2", "nwparser.p0", "Message=%{p0}");

var dup88 = match("MESSAGE#621:411/3_0", "nwparser.p0", "\"%{action}\";%{p0}");

var dup89 = match("MESSAGE#621:411/3_1", "nwparser.p0", "%{action};%{p0}");

var dup90 = match("MESSAGE#621:411/4", "nwparser.p0", "Issuer=%{p0}");

var dup91 = match("MESSAGE#621:411/5_0", "nwparser.p0", "\"%{username}\";%{p0}");

var dup92 = match("MESSAGE#621:411/5_1", "nwparser.p0", "%{username};%{p0}");

var dup93 = match("MESSAGE#621:411/6", "nwparser.p0", "Station=%{p0}");

var dup94 = match("MESSAGE#621:411/7_0", "nwparser.p0", "\"%{hostip}\";%{p0}");

var dup95 = match("MESSAGE#621:411/7_1", "nwparser.p0", "%{hostip};%{p0}");

var dup96 = match("MESSAGE#621:411/8", "nwparser.p0", "File=%{p0}");

var dup97 = match("MESSAGE#621:411/9_0", "nwparser.p0", "\"%{filename}\";%{p0}");

var dup98 = match("MESSAGE#621:411/9_1", "nwparser.p0", "%{filename};%{p0}");

var dup99 = match("MESSAGE#621:411/10", "nwparser.p0", "Safe=%{p0}");

var dup100 = match("MESSAGE#621:411/11_0", "nwparser.p0", "\"%{group_object}\";%{p0}");

var dup101 = match("MESSAGE#621:411/11_1", "nwparser.p0", "%{group_object};%{p0}");

var dup102 = match("MESSAGE#621:411/12", "nwparser.p0", "Location=%{p0}");

var dup103 = match("MESSAGE#621:411/13_0", "nwparser.p0", "\"%{directory}\";%{p0}");

var dup104 = match("MESSAGE#621:411/13_1", "nwparser.p0", "%{directory};%{p0}");

var dup105 = match("MESSAGE#621:411/14", "nwparser.p0", "Category=%{p0}");

var dup106 = match("MESSAGE#621:411/15_0", "nwparser.p0", "\"%{category}\";%{p0}");

var dup107 = match("MESSAGE#621:411/15_1", "nwparser.p0", "%{category};%{p0}");

var dup108 = match("MESSAGE#621:411/16", "nwparser.p0", "RequestId=%{p0}");

var dup109 = match("MESSAGE#621:411/17_0", "nwparser.p0", "\"%{id1}\";%{p0}");

var dup110 = match("MESSAGE#621:411/17_1", "nwparser.p0", "%{id1};%{p0}");

var dup111 = match("MESSAGE#621:411/18", "nwparser.p0", "Reason=%{p0}");

var dup112 = match("MESSAGE#621:411/19_0", "nwparser.p0", "\"%{event_description}\";%{p0}");

var dup113 = match("MESSAGE#621:411/19_1", "nwparser.p0", "%{event_description};%{p0}");

var dup114 = match("MESSAGE#621:411/20", "nwparser.p0", "Severity=%{p0}");

var dup115 = match("MESSAGE#621:411/21_0", "nwparser.p0", "\"%{severity}\";SourceUser=\"%{group}\";TargetUser=\"%{uid}\";%{p0}");

var dup116 = match("MESSAGE#621:411/21_1", "nwparser.p0", "%{severity};SourceUser=%{group};TargetUser=%{uid};%{p0}");

var dup117 = match("MESSAGE#621:411/21_2", "nwparser.p0", "\"%{severity}\";%{p0}");

var dup118 = match("MESSAGE#621:411/21_3", "nwparser.p0", "%{severity};%{p0}");

var dup119 = match("MESSAGE#621:411/22", "nwparser.p0", "GatewayStation=%{p0}");

var dup120 = match("MESSAGE#621:411/23_0", "nwparser.p0", "\"%{saddr}\";%{p0}");

var dup121 = match("MESSAGE#621:411/23_1", "nwparser.p0", "%{saddr};%{p0}");

var dup122 = match("MESSAGE#621:411/24", "nwparser.p0", "TicketID=%{p0}");

var dup123 = match("MESSAGE#621:411/25_0", "nwparser.p0", "\"%{operation_id}\";%{p0}");

var dup124 = match("MESSAGE#621:411/25_1", "nwparser.p0", "%{operation_id};%{p0}");

var dup125 = match("MESSAGE#621:411/26", "nwparser.p0", "PolicyID=%{p0}");

var dup126 = match("MESSAGE#621:411/27_0", "nwparser.p0", "\"%{policyname}\";%{p0}");

var dup127 = match("MESSAGE#621:411/27_1", "nwparser.p0", "%{policyname};%{p0}");

var dup128 = match("MESSAGE#621:411/28", "nwparser.p0", "UserName=%{p0}");

var dup129 = match("MESSAGE#621:411/29_0", "nwparser.p0", "\"%{c_username}\";%{p0}");

var dup130 = match("MESSAGE#621:411/29_1", "nwparser.p0", "%{c_username};%{p0}");

var dup131 = match("MESSAGE#621:411/30", "nwparser.p0", "LogonDomain=%{p0}");

var dup132 = match("MESSAGE#621:411/31_0", "nwparser.p0", "\"%{domain}\";%{p0}");

var dup133 = match("MESSAGE#621:411/31_1", "nwparser.p0", "%{domain};%{p0}");

var dup134 = match("MESSAGE#621:411/32", "nwparser.p0", "Address=%{p0}");

var dup135 = match("MESSAGE#621:411/33_0", "nwparser.p0", "\"%{dhost}\";%{p0}");

var dup136 = match("MESSAGE#621:411/33_1", "nwparser.p0", "%{dhost};%{p0}");

var dup137 = match("MESSAGE#621:411/34", "nwparser.p0", "CPMStatus=%{p0}");

var dup138 = match("MESSAGE#621:411/35_0", "nwparser.p0", "\"%{disposition}\";%{p0}");

var dup139 = match("MESSAGE#621:411/35_1", "nwparser.p0", "%{disposition};%{p0}");

var dup140 = match("MESSAGE#621:411/36", "nwparser.p0", "Port=%{p0}");

var dup141 = match("MESSAGE#621:411/37_0", "nwparser.p0", "\"%{dport}\";%{p0}");

var dup142 = match("MESSAGE#621:411/37_1", "nwparser.p0", "%{dport};%{p0}");

var dup143 = match("MESSAGE#621:411/38", "nwparser.p0", "Database=%{p0}");

var dup144 = match("MESSAGE#621:411/39_0", "nwparser.p0", "\"%{db_name}\";%{p0}");

var dup145 = match("MESSAGE#621:411/39_1", "nwparser.p0", "%{db_name};%{p0}");

var dup146 = match("MESSAGE#621:411/40", "nwparser.p0", "DeviceType=%{p0}");

var dup147 = match("MESSAGE#621:411/41_0", "nwparser.p0", "\"%{obj_type}\";%{p0}");

var dup148 = match("MESSAGE#621:411/41_1", "nwparser.p0", "%{obj_type};%{p0}");

var dup149 = match("MESSAGE#621:411/42", "nwparser.p0", "ExtraDetails=%{p0}");

var dup150 = match("MESSAGE#621:411/43_1", "nwparser.p0", "%{info};");

var dup151 = tagval("MESSAGE#0:1:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup1,
	dup2,
	dup3,
]));

var dup152 = match("MESSAGE#1:1", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup1,
	dup2,
]));

var dup153 = tagval("MESSAGE#2:2:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup4,
	dup2,
	dup3,
]));

var dup154 = match("MESSAGE#3:2", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup4,
	dup2,
]));

var dup155 = tagval("MESSAGE#6:4:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
	dup3,
]));

var dup156 = match("MESSAGE#7:4", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
]));

var dup157 = tagval("MESSAGE#20:13:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup16,
	dup17,
	dup9,
	dup2,
	dup3,
]));

var dup158 = match("MESSAGE#21:13", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup16,
	dup17,
	dup9,
	dup2,
]));

var dup159 = tagval("MESSAGE#26:16:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup19,
	dup2,
	dup3,
]));

var dup160 = match("MESSAGE#27:16", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup19,
	dup2,
]));

var dup161 = tagval("MESSAGE#30:18:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup2,
	dup3,
]));

var dup162 = match("MESSAGE#31:18", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup2,
]));

var dup163 = tagval("MESSAGE#38:22:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup21,
	dup2,
	dup3,
]));

var dup164 = match("MESSAGE#39:22", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup21,
	dup2,
]));

var dup165 = tagval("MESSAGE#70:38:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup23,
	dup2,
	dup3,
]));

var dup166 = match("MESSAGE#71:38", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup23,
	dup2,
]));

var dup167 = tagval("MESSAGE#116:61:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup20,
	dup2,
	dup3,
]));

var dup168 = match("MESSAGE#117:61", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup20,
	dup2,
]));

var dup169 = tagval("MESSAGE#126:66:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup26,
	dup2,
	dup3,
]));

var dup170 = match("MESSAGE#127:66", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup26,
	dup2,
]));

var dup171 = tagval("MESSAGE#190:98:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup26,
	dup2,
	dup3,
	dup24,
	dup25,
]));

var dup172 = linear_select([
	dup32,
	dup33,
]);

var dup173 = linear_select([
	dup34,
	dup35,
]);

var dup174 = linear_select([
	dup36,
	dup37,
]);

var dup175 = linear_select([
	dup38,
	dup39,
]);

var dup176 = linear_select([
	dup40,
	dup41,
]);

var dup177 = linear_select([
	dup42,
	dup43,
]);

var dup178 = linear_select([
	dup44,
	dup45,
]);

var dup179 = linear_select([
	dup46,
	dup47,
]);

var dup180 = linear_select([
	dup48,
	dup49,
]);

var dup181 = linear_select([
	dup50,
	dup51,
]);

var dup182 = linear_select([
	dup52,
	dup53,
]);

var dup183 = linear_select([
	dup54,
	dup55,
]);

var dup184 = linear_select([
	dup56,
	dup57,
]);

var dup185 = linear_select([
	dup58,
	dup59,
]);

var dup186 = linear_select([
	dup60,
	dup61,
]);

var dup187 = linear_select([
	dup62,
	dup63,
]);

var dup188 = linear_select([
	dup64,
	dup65,
]);

var dup189 = linear_select([
	dup66,
	dup67,
]);

var dup190 = linear_select([
	dup68,
	dup69,
]);

var dup191 = linear_select([
	dup70,
	dup71,
]);

var dup192 = linear_select([
	dup72,
	dup73,
]);

var dup193 = linear_select([
	dup74,
	dup75,
]);

var dup194 = linear_select([
	dup76,
	dup77,
]);

var dup195 = tagval("MESSAGE#591:317:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup79,
	dup80,
	dup81,
	dup2,
	dup3,
]));

var dup196 = match("MESSAGE#592:317", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup79,
	dup80,
	dup81,
	dup2,
]));

var dup197 = tagval("MESSAGE#595:355:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup82,
	dup2,
	dup3,
]));

var dup198 = match("MESSAGE#596:355", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup82,
	dup2,
]));

var dup199 = tagval("MESSAGE#599:357:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup83,
	dup2,
	dup3,
]));

var dup200 = match("MESSAGE#600:357", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup83,
	dup2,
]));

var dup201 = match("MESSAGE#617:372", "nwparser.payload", "Version=%{version};Message=%{action};Issuer=%{username};Station=%{hostip};File=%{filename};Safe=%{group_object};Location=%{directory};Category=%{category};RequestId=%{id1};Reason=%{event_description};Severity=%{severity};GatewayStation=%{saddr};TicketID=%{operation_id};PolicyID=%{policyname};UserName=%{c_username};LogonDomain=%{domain};Address=%{dhost};CPMStatus=%{disposition};Port=\"%{dport}\";Database=%{db_name};DeviceType=%{obj_type};ExtraDetails=%{info};", processor_chain([
	dup4,
	dup2,
	dup3,
]));

var dup202 = linear_select([
	dup85,
	dup86,
]);

var dup203 = linear_select([
	dup88,
	dup89,
]);

var dup204 = linear_select([
	dup91,
	dup92,
]);

var dup205 = linear_select([
	dup94,
	dup95,
]);

var dup206 = linear_select([
	dup97,
	dup98,
]);

var dup207 = linear_select([
	dup100,
	dup101,
]);

var dup208 = linear_select([
	dup103,
	dup104,
]);

var dup209 = linear_select([
	dup106,
	dup107,
]);

var dup210 = linear_select([
	dup109,
	dup110,
]);

var dup211 = linear_select([
	dup112,
	dup113,
]);

var dup212 = linear_select([
	dup115,
	dup116,
	dup117,
	dup118,
]);

var dup213 = linear_select([
	dup120,
	dup121,
]);

var dup214 = linear_select([
	dup123,
	dup124,
]);

var dup215 = linear_select([
	dup126,
	dup127,
]);

var dup216 = linear_select([
	dup129,
	dup130,
]);

var dup217 = linear_select([
	dup132,
	dup133,
]);

var dup218 = linear_select([
	dup135,
	dup136,
]);

var dup219 = linear_select([
	dup138,
	dup139,
]);

var dup220 = linear_select([
	dup141,
	dup142,
]);

var dup221 = linear_select([
	dup144,
	dup145,
]);

var dup222 = linear_select([
	dup147,
	dup148,
]);

var hdr1 = match("HEADER#0:0001", "message", "%{hmonth->} %{hday->} %{htime->} %{hproduct->} ProductName=\"%{hdevice}\",ProductAccount=\"%{hfld1}\",ProductProcess=\"%{process}\",EventId=\"%{messageid}\", %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdevice"),
			constant("\",ProductAccount=\""),
			field("hfld1"),
			constant("\",ProductProcess=\""),
			field("process"),
			constant("\",EventId=\""),
			field("messageid"),
			constant("\", "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0005", "message", "%{hfld1->} %{hdatetime->} %{hproduct->} ProductName=\"%{hdevice}\",ProductAccount=\"%{hfld4}\",ProductProcess=\"%{process}\",EventId=\"%{messageid}\", %{p0}", processor_chain([
	setc("header_id","0005"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hdevice"),
			constant("\",ProductAccount=\""),
			field("hfld4"),
			constant("\",ProductProcess=\""),
			field("process"),
			constant("\",EventId=\""),
			field("messageid"),
			constant("\", "),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0002", "message", "%{hmonth->} %{hday->} %{htime->} %{hproduct->} %CYBERARK: MessageID=\"%{messageid}\";%{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr4 = match("HEADER#3:0003", "message", "%{hfld1->} %{hdatetime->} %{hostname->} %CYBERARK: MessageID=\"%{messageid}\";%{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr5 = match("HEADER#4:0004", "message", "%CYBERARK: MessageID=\"%{messageid}\";%{payload}", processor_chain([
	setc("header_id","0004"),
]));

var hdr6 = match("HEADER#5:0006", "message", "%{hdatetime->} %{hostname->} %CYBERARK: MessageID=\"%{messageid}\";%{payload}", processor_chain([
	setc("header_id","0006"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
	hdr5,
	hdr6,
]);

var msg1 = msg("1:01", dup151);

var msg2 = msg("1", dup152);

var select2 = linear_select([
	msg1,
	msg2,
]);

var msg3 = msg("2:01", dup153);

var msg4 = msg("2", dup154);

var select3 = linear_select([
	msg3,
	msg4,
]);

var msg5 = msg("3:01", dup151);

var msg6 = msg("3", dup152);

var select4 = linear_select([
	msg5,
	msg6,
]);

var msg7 = msg("4:01", dup155);

var msg8 = msg("4", dup156);

var select5 = linear_select([
	msg7,
	msg8,
]);

var part1 = tagval("MESSAGE#8:7:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup10,
	dup6,
	dup7,
	dup8,
	dup11,
	dup2,
	dup3,
]));

var msg9 = msg("7:01", part1);

var part2 = match("MESSAGE#9:7", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup10,
	dup6,
	dup7,
	dup8,
	dup11,
	dup2,
]));

var msg10 = msg("7", part2);

var select6 = linear_select([
	msg9,
	msg10,
]);

var part3 = tagval("MESSAGE#10:8:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup12,
	dup6,
	dup13,
	dup8,
	dup11,
	dup2,
	dup3,
]));

var msg11 = msg("8:01", part3);

var part4 = match("MESSAGE#11:8", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup12,
	dup6,
	dup13,
	dup8,
	dup11,
	dup2,
]));

var msg12 = msg("8", part4);

var select7 = linear_select([
	msg11,
	msg12,
]);

var part5 = tagval("MESSAGE#12:9:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup1,
	dup14,
	dup9,
	dup2,
	dup3,
]));

var msg13 = msg("9:01", part5);

var part6 = match("MESSAGE#13:9", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup1,
	dup14,
	dup9,
	dup2,
]));

var msg14 = msg("9", part6);

var select8 = linear_select([
	msg13,
	msg14,
]);

var msg15 = msg("10:01", dup151);

var msg16 = msg("10", dup152);

var select9 = linear_select([
	msg15,
	msg16,
]);

var msg17 = msg("11:01", dup151);

var msg18 = msg("11", dup152);

var select10 = linear_select([
	msg17,
	msg18,
]);

var msg19 = msg("12:01", dup151);

var msg20 = msg("12", dup152);

var select11 = linear_select([
	msg19,
	msg20,
]);

var msg21 = msg("13:01", dup157);

var msg22 = msg("13", dup158);

var select12 = linear_select([
	msg21,
	msg22,
]);

var msg23 = msg("14:01", dup157);

var msg24 = msg("14", dup158);

var select13 = linear_select([
	msg23,
	msg24,
]);

var part7 = tagval("MESSAGE#24:15:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup18,
	dup9,
	dup2,
	dup3,
]));

var msg25 = msg("15:01", part7);

var part8 = match("MESSAGE#25:15", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup18,
	dup9,
	dup2,
]));

var msg26 = msg("15", part8);

var select14 = linear_select([
	msg25,
	msg26,
]);

var msg27 = msg("16:01", dup159);

var msg28 = msg("16", dup160);

var select15 = linear_select([
	msg27,
	msg28,
]);

var msg29 = msg("17:01", dup151);

var msg30 = msg("17", dup152);

var select16 = linear_select([
	msg29,
	msg30,
]);

var msg31 = msg("18:01", dup161);

var msg32 = msg("18", dup162);

var select17 = linear_select([
	msg31,
	msg32,
]);

var part9 = tagval("MESSAGE#32:19:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup20,
	dup16,
	dup11,
	dup2,
	dup3,
]));

var msg33 = msg("19:01", part9);

var part10 = match("MESSAGE#33:19", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup20,
	dup16,
	dup11,
	dup2,
]));

var msg34 = msg("19", part10);

var select18 = linear_select([
	msg33,
	msg34,
]);

var part11 = tagval("MESSAGE#34:20:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup19,
	dup16,
	dup2,
	dup3,
]));

var msg35 = msg("20:01", part11);

var part12 = match("MESSAGE#35:20", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup19,
	dup16,
	dup2,
]));

var msg36 = msg("20", part12);

var select19 = linear_select([
	msg35,
	msg36,
]);

var part13 = tagval("MESSAGE#36:21:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup16,
	dup9,
	dup2,
	dup3,
]));

var msg37 = msg("21:01", part13);

var part14 = match("MESSAGE#37:21", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup16,
	dup9,
	dup2,
]));

var msg38 = msg("21", part14);

var select20 = linear_select([
	msg37,
	msg38,
]);

var msg39 = msg("22:01", dup163);

var msg40 = msg("22", dup164);

var select21 = linear_select([
	msg39,
	msg40,
]);

var part15 = tagval("MESSAGE#40:23:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup22,
	dup2,
	dup3,
]));

var msg41 = msg("23:01", part15);

var part16 = match("MESSAGE#41:23", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup22,
	dup2,
]));

var msg42 = msg("23", part16);

var select22 = linear_select([
	msg41,
	msg42,
]);

var msg43 = msg("24:01", dup163);

var msg44 = msg("24", dup164);

var select23 = linear_select([
	msg43,
	msg44,
]);

var msg45 = msg("25:01", dup151);

var msg46 = msg("25", dup152);

var select24 = linear_select([
	msg45,
	msg46,
]);

var msg47 = msg("26:01", dup151);

var msg48 = msg("26", dup152);

var select25 = linear_select([
	msg47,
	msg48,
]);

var msg49 = msg("27:01", dup151);

var msg50 = msg("27", dup152);

var select26 = linear_select([
	msg49,
	msg50,
]);

var msg51 = msg("28:01", dup163);

var msg52 = msg("28", dup164);

var select27 = linear_select([
	msg51,
	msg52,
]);

var msg53 = msg("29:01", dup151);

var msg54 = msg("29", dup152);

var select28 = linear_select([
	msg53,
	msg54,
]);

var msg55 = msg("30:01", dup151);

var msg56 = msg("30", dup152);

var select29 = linear_select([
	msg55,
	msg56,
]);

var msg57 = msg("31:01", dup163);

var msg58 = msg("31", dup164);

var select30 = linear_select([
	msg57,
	msg58,
]);

var msg59 = msg("32:01", dup163);

var msg60 = msg("32", dup164);

var select31 = linear_select([
	msg59,
	msg60,
]);

var msg61 = msg("33:01", dup163);

var msg62 = msg("33", dup164);

var select32 = linear_select([
	msg61,
	msg62,
]);

var msg63 = msg("34:01", dup151);

var msg64 = msg("34", dup152);

var select33 = linear_select([
	msg63,
	msg64,
]);

var msg65 = msg("35:01", dup151);

var msg66 = msg("35", dup152);

var select34 = linear_select([
	msg65,
	msg66,
]);

var msg67 = msg("36:01", dup163);

var msg68 = msg("36", dup164);

var select35 = linear_select([
	msg67,
	msg68,
]);

var msg69 = msg("37:01", dup163);

var msg70 = msg("37", dup164);

var select36 = linear_select([
	msg69,
	msg70,
]);

var msg71 = msg("38:01", dup165);

var msg72 = msg("38", dup166);

var select37 = linear_select([
	msg71,
	msg72,
]);

var msg73 = msg("39:01", dup163);

var msg74 = msg("39", dup164);

var select38 = linear_select([
	msg73,
	msg74,
]);

var msg75 = msg("40:01", dup151);

var msg76 = msg("40", dup152);

var select39 = linear_select([
	msg75,
	msg76,
]);

var msg77 = msg("41:01", dup151);

var msg78 = msg("41", dup152);

var select40 = linear_select([
	msg77,
	msg78,
]);

var msg79 = msg("42:01", dup151);

var msg80 = msg("42", dup152);

var select41 = linear_select([
	msg79,
	msg80,
]);

var msg81 = msg("43:01", dup151);

var msg82 = msg("43", dup152);

var select42 = linear_select([
	msg81,
	msg82,
]);

var msg83 = msg("44:01", dup151);

var msg84 = msg("44", dup152);

var select43 = linear_select([
	msg83,
	msg84,
]);

var msg85 = msg("45:01", dup151);

var msg86 = msg("45", dup152);

var select44 = linear_select([
	msg85,
	msg86,
]);

var msg87 = msg("46:01", dup151);

var msg88 = msg("46", dup152);

var select45 = linear_select([
	msg87,
	msg88,
]);

var msg89 = msg("47:01", dup151);

var msg90 = msg("47", dup152);

var select46 = linear_select([
	msg89,
	msg90,
]);

var msg91 = msg("48:01", dup151);

var msg92 = msg("48", dup152);

var select47 = linear_select([
	msg91,
	msg92,
]);

var msg93 = msg("49:01", dup151);

var msg94 = msg("49", dup152);

var select48 = linear_select([
	msg93,
	msg94,
]);

var part17 = tagval("MESSAGE#94:50:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup21,
	dup2,
	dup3,
	dup24,
	dup25,
]));

var msg95 = msg("50:01", part17);

var msg96 = msg("50", dup164);

var select49 = linear_select([
	msg95,
	msg96,
]);

var msg97 = msg("51:01", dup163);

var msg98 = msg("51", dup164);

var select50 = linear_select([
	msg97,
	msg98,
]);

var msg99 = msg("52:01", dup163);

var msg100 = msg("52", dup164);

var select51 = linear_select([
	msg99,
	msg100,
]);

var msg101 = msg("53:01", dup151);

var msg102 = msg("53", dup152);

var select52 = linear_select([
	msg101,
	msg102,
]);

var msg103 = msg("54:01", dup151);

var msg104 = msg("54", dup152);

var select53 = linear_select([
	msg103,
	msg104,
]);

var msg105 = msg("55:01", dup151);

var msg106 = msg("55", dup152);

var select54 = linear_select([
	msg105,
	msg106,
]);

var msg107 = msg("56:01", dup151);

var msg108 = msg("56", dup152);

var select55 = linear_select([
	msg107,
	msg108,
]);

var msg109 = msg("57:01", dup165);

var msg110 = msg("57", dup166);

var select56 = linear_select([
	msg109,
	msg110,
]);

var msg111 = msg("58:01", dup163);

var msg112 = msg("58", dup164);

var select57 = linear_select([
	msg111,
	msg112,
]);

var msg113 = msg("59:01", dup163);

var msg114 = msg("59", dup164);

var select58 = linear_select([
	msg113,
	msg114,
]);

var msg115 = msg("60:01", dup165);

var msg116 = msg("60", dup166);

var select59 = linear_select([
	msg115,
	msg116,
]);

var msg117 = msg("61:01", dup167);

var msg118 = msg("61", dup168);

var select60 = linear_select([
	msg117,
	msg118,
]);

var msg119 = msg("62:01", dup163);

var msg120 = msg("62", dup164);

var select61 = linear_select([
	msg119,
	msg120,
]);

var msg121 = msg("63:01", dup151);

var msg122 = msg("63", dup152);

var select62 = linear_select([
	msg121,
	msg122,
]);

var msg123 = msg("64:01", dup167);

var msg124 = msg("64", dup168);

var select63 = linear_select([
	msg123,
	msg124,
]);

var msg125 = msg("65:01", dup151);

var msg126 = msg("65", dup152);

var select64 = linear_select([
	msg125,
	msg126,
]);

var msg127 = msg("66:01", dup169);

var msg128 = msg("66", dup170);

var select65 = linear_select([
	msg127,
	msg128,
]);

var msg129 = msg("67:01", dup169);

var msg130 = msg("67", dup170);

var select66 = linear_select([
	msg129,
	msg130,
]);

var msg131 = msg("68:01", dup169);

var msg132 = msg("68", dup170);

var select67 = linear_select([
	msg131,
	msg132,
]);

var msg133 = msg("69:01", dup169);

var msg134 = msg("69", dup170);

var select68 = linear_select([
	msg133,
	msg134,
]);

var msg135 = msg("70:01", dup151);

var msg136 = msg("70", dup152);

var select69 = linear_select([
	msg135,
	msg136,
]);

var msg137 = msg("71:01", dup169);

var msg138 = msg("71", dup170);

var select70 = linear_select([
	msg137,
	msg138,
]);

var msg139 = msg("72:01", dup151);

var msg140 = msg("72", dup152);

var select71 = linear_select([
	msg139,
	msg140,
]);

var msg141 = msg("73:01", dup169);

var msg142 = msg("73", dup170);

var select72 = linear_select([
	msg141,
	msg142,
]);

var msg143 = msg("74:01", dup151);

var msg144 = msg("74", dup152);

var select73 = linear_select([
	msg143,
	msg144,
]);

var msg145 = msg("75:01", dup169);

var msg146 = msg("75", dup170);

var select74 = linear_select([
	msg145,
	msg146,
]);

var msg147 = msg("76:01", dup151);

var msg148 = msg("76", dup152);

var select75 = linear_select([
	msg147,
	msg148,
]);

var msg149 = msg("77:01", dup151);

var msg150 = msg("77", dup152);

var select76 = linear_select([
	msg149,
	msg150,
]);

var msg151 = msg("78:01", dup151);

var msg152 = msg("78", dup152);

var select77 = linear_select([
	msg151,
	msg152,
]);

var msg153 = msg("79:01", dup169);

var msg154 = msg("79", dup170);

var select78 = linear_select([
	msg153,
	msg154,
]);

var msg155 = msg("80:01", dup169);

var msg156 = msg("80", dup170);

var select79 = linear_select([
	msg155,
	msg156,
]);

var msg157 = msg("81:01", dup167);

var msg158 = msg("81", dup168);

var select80 = linear_select([
	msg157,
	msg158,
]);

var msg159 = msg("82:01", dup151);

var msg160 = msg("82", dup152);

var select81 = linear_select([
	msg159,
	msg160,
]);

var msg161 = msg("83:01", dup169);

var msg162 = msg("83", dup170);

var select82 = linear_select([
	msg161,
	msg162,
]);

var msg163 = msg("84:01", dup169);

var msg164 = msg("84", dup170);

var select83 = linear_select([
	msg163,
	msg164,
]);

var msg165 = msg("85:01", dup151);

var msg166 = msg("85", dup152);

var select84 = linear_select([
	msg165,
	msg166,
]);

var msg167 = msg("86:01", dup159);

var msg168 = msg("86", dup160);

var select85 = linear_select([
	msg167,
	msg168,
]);

var msg169 = msg("87:01", dup151);

var msg170 = msg("87", dup152);

var select86 = linear_select([
	msg169,
	msg170,
]);

var msg171 = msg("88:01", dup169);

var msg172 = msg("88", dup170);

var select87 = linear_select([
	msg171,
	msg172,
]);

var msg173 = msg("89:01", dup151);

var msg174 = msg("89", dup152);

var select88 = linear_select([
	msg173,
	msg174,
]);

var msg175 = msg("90:01", dup151);

var msg176 = msg("90", dup152);

var select89 = linear_select([
	msg175,
	msg176,
]);

var msg177 = msg("91:01", dup151);

var msg178 = msg("91", dup152);

var select90 = linear_select([
	msg177,
	msg178,
]);

var msg179 = msg("92:01", dup151);

var msg180 = msg("92", dup152);

var select91 = linear_select([
	msg179,
	msg180,
]);

var msg181 = msg("93:01", dup151);

var msg182 = msg("93", dup152);

var select92 = linear_select([
	msg181,
	msg182,
]);

var msg183 = msg("94:01", dup169);

var msg184 = msg("94", dup170);

var select93 = linear_select([
	msg183,
	msg184,
]);

var msg185 = msg("95:01", dup169);

var msg186 = msg("95", dup170);

var select94 = linear_select([
	msg185,
	msg186,
]);

var msg187 = msg("96:01", dup151);

var msg188 = msg("96", dup152);

var select95 = linear_select([
	msg187,
	msg188,
]);

var msg189 = msg("97:01", dup151);

var msg190 = msg("97", dup152);

var select96 = linear_select([
	msg189,
	msg190,
]);

var msg191 = msg("98:01", dup171);

var msg192 = msg("98", dup170);

var select97 = linear_select([
	msg191,
	msg192,
]);

var msg193 = msg("99:01", dup171);

var msg194 = msg("99", dup170);

var select98 = linear_select([
	msg193,
	msg194,
]);

var msg195 = msg("100:01", dup151);

var msg196 = msg("100", dup152);

var select99 = linear_select([
	msg195,
	msg196,
]);

var msg197 = msg("101:01", dup151);

var msg198 = msg("101", dup152);

var select100 = linear_select([
	msg197,
	msg198,
]);

var msg199 = msg("102:01", dup155);

var msg200 = msg("102", dup156);

var select101 = linear_select([
	msg199,
	msg200,
]);

var part18 = tagval("MESSAGE#200:103:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup27,
	dup6,
	dup7,
	dup8,
	dup28,
	dup2,
	dup3,
]));

var msg201 = msg("103:01", part18);

var part19 = match("MESSAGE#201:103", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup27,
	dup6,
	dup7,
	dup8,
	dup28,
	dup2,
]));

var msg202 = msg("103", part19);

var select102 = linear_select([
	msg201,
	msg202,
]);

var part20 = tagval("MESSAGE#202:104:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup27,
	dup6,
	dup29,
	dup2,
	dup3,
]));

var msg203 = msg("104:01", part20);

var part21 = match("MESSAGE#203:104", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup27,
	dup6,
	dup29,
	dup2,
]));

var msg204 = msg("104", part21);

var select103 = linear_select([
	msg203,
	msg204,
]);

var msg205 = msg("105:01", dup169);

var msg206 = msg("105", dup170);

var select104 = linear_select([
	msg205,
	msg206,
]);

var msg207 = msg("106:01", dup169);

var msg208 = msg("106", dup170);

var select105 = linear_select([
	msg207,
	msg208,
]);

var msg209 = msg("107:01", dup169);

var msg210 = msg("107", dup170);

var select106 = linear_select([
	msg209,
	msg210,
]);

var msg211 = msg("108:01", dup169);

var msg212 = msg("108", dup170);

var select107 = linear_select([
	msg211,
	msg212,
]);

var msg213 = msg("109:01", dup169);

var msg214 = msg("109", dup170);

var select108 = linear_select([
	msg213,
	msg214,
]);

var msg215 = msg("110:01", dup151);

var msg216 = msg("110", dup152);

var select109 = linear_select([
	msg215,
	msg216,
]);

var msg217 = msg("111:01", dup169);

var msg218 = msg("111", dup170);

var select110 = linear_select([
	msg217,
	msg218,
]);

var msg219 = msg("112:01", dup169);

var msg220 = msg("112", dup170);

var select111 = linear_select([
	msg219,
	msg220,
]);

var msg221 = msg("114:01", dup169);

var msg222 = msg("114", dup170);

var select112 = linear_select([
	msg221,
	msg222,
]);

var msg223 = msg("115:01", dup169);

var msg224 = msg("115", dup170);

var select113 = linear_select([
	msg223,
	msg224,
]);

var msg225 = msg("116:01", dup151);

var msg226 = msg("116", dup152);

var select114 = linear_select([
	msg225,
	msg226,
]);

var msg227 = msg("117:01", dup151);

var msg228 = msg("117", dup152);

var select115 = linear_select([
	msg227,
	msg228,
]);

var msg229 = msg("118:01", dup169);

var msg230 = msg("118", dup170);

var select116 = linear_select([
	msg229,
	msg230,
]);

var msg231 = msg("119:01", dup169);

var msg232 = msg("119", dup170);

var select117 = linear_select([
	msg231,
	msg232,
]);

var msg233 = msg("120:01", dup169);

var msg234 = msg("120", dup170);

var select118 = linear_select([
	msg233,
	msg234,
]);

var msg235 = msg("121:01", dup169);

var msg236 = msg("121", dup170);

var select119 = linear_select([
	msg235,
	msg236,
]);

var msg237 = msg("122:01", dup169);

var msg238 = msg("122", dup170);

var select120 = linear_select([
	msg237,
	msg238,
]);

var msg239 = msg("123:01", dup169);

var msg240 = msg("123", dup170);

var select121 = linear_select([
	msg239,
	msg240,
]);

var msg241 = msg("124:01", dup169);

var msg242 = msg("124", dup170);

var select122 = linear_select([
	msg241,
	msg242,
]);

var msg243 = msg("125:01", dup169);

var msg244 = msg("125", dup170);

var select123 = linear_select([
	msg243,
	msg244,
]);

var msg245 = msg("126:01", dup169);

var msg246 = msg("126", dup170);

var select124 = linear_select([
	msg245,
	msg246,
]);

var msg247 = msg("127:01", dup169);

var msg248 = msg("127", dup170);

var select125 = linear_select([
	msg247,
	msg248,
]);

var msg249 = msg("128:01", dup169);

var msg250 = msg("128", dup170);

var select126 = linear_select([
	msg249,
	msg250,
]);

var msg251 = msg("129:01", dup169);

var msg252 = msg("129", dup170);

var select127 = linear_select([
	msg251,
	msg252,
]);

var msg253 = msg("130:01", dup169);

var msg254 = msg("130", dup170);

var select128 = linear_select([
	msg253,
	msg254,
]);

var msg255 = msg("131:01", dup151);

var msg256 = msg("131", dup152);

var select129 = linear_select([
	msg255,
	msg256,
]);

var msg257 = msg("132:01", dup151);

var msg258 = msg("132", dup152);

var select130 = linear_select([
	msg257,
	msg258,
]);

var msg259 = msg("133:01", dup151);

var msg260 = msg("133", dup152);

var select131 = linear_select([
	msg259,
	msg260,
]);

var part22 = tagval("MESSAGE#260:134:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup30,
	dup2,
	dup3,
]));

var msg261 = msg("134:01", part22);

var part23 = match("MESSAGE#261:134", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup30,
	dup2,
]));

var msg262 = msg("134", part23);

var select132 = linear_select([
	msg261,
	msg262,
]);

var msg263 = msg("135:01", dup151);

var msg264 = msg("135", dup152);

var select133 = linear_select([
	msg263,
	msg264,
]);

var msg265 = msg("136:01", dup169);

var msg266 = msg("136", dup170);

var select134 = linear_select([
	msg265,
	msg266,
]);

var msg267 = msg("137:01", dup169);

var msg268 = msg("137", dup170);

var select135 = linear_select([
	msg267,
	msg268,
]);

var msg269 = msg("138:01", dup169);

var msg270 = msg("138", dup170);

var select136 = linear_select([
	msg269,
	msg270,
]);

var msg271 = msg("139:01", dup169);

var msg272 = msg("139", dup170);

var select137 = linear_select([
	msg271,
	msg272,
]);

var msg273 = msg("140:01", dup169);

var msg274 = msg("140", dup170);

var select138 = linear_select([
	msg273,
	msg274,
]);

var msg275 = msg("141:01", dup169);

var msg276 = msg("141", dup170);

var select139 = linear_select([
	msg275,
	msg276,
]);

var msg277 = msg("142:01", dup169);

var msg278 = msg("142", dup170);

var select140 = linear_select([
	msg277,
	msg278,
]);

var msg279 = msg("143:01", dup169);

var msg280 = msg("143", dup170);

var select141 = linear_select([
	msg279,
	msg280,
]);

var msg281 = msg("144:01", dup169);

var msg282 = msg("144", dup170);

var select142 = linear_select([
	msg281,
	msg282,
]);

var msg283 = msg("145:01", dup169);

var msg284 = msg("145", dup170);

var select143 = linear_select([
	msg283,
	msg284,
]);

var msg285 = msg("146:01", dup151);

var msg286 = msg("146", dup152);

var select144 = linear_select([
	msg285,
	msg286,
]);

var msg287 = msg("147:01", dup151);

var msg288 = msg("147", dup152);

var select145 = linear_select([
	msg287,
	msg288,
]);

var msg289 = msg("148:01", dup151);

var msg290 = msg("148", dup152);

var select146 = linear_select([
	msg289,
	msg290,
]);

var msg291 = msg("149:01", dup151);

var msg292 = msg("149", dup152);

var select147 = linear_select([
	msg291,
	msg292,
]);

var msg293 = msg("150:01", dup151);

var msg294 = msg("150", dup152);

var select148 = linear_select([
	msg293,
	msg294,
]);

var msg295 = msg("152:01", dup151);

var msg296 = msg("152", dup152);

var select149 = linear_select([
	msg295,
	msg296,
]);

var msg297 = msg("153:01", dup151);

var msg298 = msg("153", dup152);

var select150 = linear_select([
	msg297,
	msg298,
]);

var msg299 = msg("154:01", dup151);

var msg300 = msg("154", dup152);

var select151 = linear_select([
	msg299,
	msg300,
]);

var msg301 = msg("155:01", dup151);

var msg302 = msg("155", dup152);

var select152 = linear_select([
	msg301,
	msg302,
]);

var msg303 = msg("156:01", dup151);

var msg304 = msg("156", dup152);

var select153 = linear_select([
	msg303,
	msg304,
]);

var msg305 = msg("157:01", dup151);

var msg306 = msg("157", dup152);

var select154 = linear_select([
	msg305,
	msg306,
]);

var msg307 = msg("158:01", dup151);

var msg308 = msg("158", dup152);

var select155 = linear_select([
	msg307,
	msg308,
]);

var msg309 = msg("159:01", dup151);

var msg310 = msg("159", dup152);

var select156 = linear_select([
	msg309,
	msg310,
]);

var msg311 = msg("160:01", dup151);

var msg312 = msg("160", dup152);

var select157 = linear_select([
	msg311,
	msg312,
]);

var msg313 = msg("161:01", dup151);

var msg314 = msg("161", dup152);

var select158 = linear_select([
	msg313,
	msg314,
]);

var msg315 = msg("162:01", dup151);

var msg316 = msg("162", dup152);

var select159 = linear_select([
	msg315,
	msg316,
]);

var msg317 = msg("163:01", dup151);

var msg318 = msg("163", dup152);

var select160 = linear_select([
	msg317,
	msg318,
]);

var msg319 = msg("164:01", dup151);

var msg320 = msg("164", dup152);

var select161 = linear_select([
	msg319,
	msg320,
]);

var msg321 = msg("165:01", dup151);

var msg322 = msg("165", dup152);

var select162 = linear_select([
	msg321,
	msg322,
]);

var msg323 = msg("166:01", dup151);

var msg324 = msg("166", dup152);

var select163 = linear_select([
	msg323,
	msg324,
]);

var msg325 = msg("167:01", dup151);

var msg326 = msg("167", dup152);

var select164 = linear_select([
	msg325,
	msg326,
]);

var msg327 = msg("168:01", dup151);

var msg328 = msg("168", dup152);

var select165 = linear_select([
	msg327,
	msg328,
]);

var msg329 = msg("169:01", dup151);

var msg330 = msg("169", dup152);

var select166 = linear_select([
	msg329,
	msg330,
]);

var msg331 = msg("170:01", dup169);

var msg332 = msg("170", dup170);

var select167 = linear_select([
	msg331,
	msg332,
]);

var msg333 = msg("171:01", dup151);

var msg334 = msg("171", dup152);

var select168 = linear_select([
	msg333,
	msg334,
]);

var msg335 = msg("172:01", dup169);

var msg336 = msg("172", dup170);

var select169 = linear_select([
	msg335,
	msg336,
]);

var msg337 = msg("173:01", dup151);

var msg338 = msg("173", dup152);

var select170 = linear_select([
	msg337,
	msg338,
]);

var msg339 = msg("174:01", dup151);

var msg340 = msg("174", dup152);

var select171 = linear_select([
	msg339,
	msg340,
]);

var msg341 = msg("175:01", dup151);

var msg342 = msg("175", dup152);

var select172 = linear_select([
	msg341,
	msg342,
]);

var msg343 = msg("176:01", dup151);

var msg344 = msg("176", dup152);

var select173 = linear_select([
	msg343,
	msg344,
]);

var msg345 = msg("177:01", dup151);

var msg346 = msg("177", dup152);

var select174 = linear_select([
	msg345,
	msg346,
]);

var msg347 = msg("178:01", dup151);

var msg348 = msg("178", dup152);

var select175 = linear_select([
	msg347,
	msg348,
]);

var msg349 = msg("179:01", dup169);

var msg350 = msg("179", dup170);

var select176 = linear_select([
	msg349,
	msg350,
]);

var msg351 = msg("180:01", dup169);

var msg352 = msg("180", dup170);

var select177 = linear_select([
	msg351,
	msg352,
]);

var msg353 = msg("181:01", dup169);

var msg354 = msg("181", dup170);

var select178 = linear_select([
	msg353,
	msg354,
]);

var msg355 = msg("182:01", dup169);

var msg356 = msg("182", dup170);

var select179 = linear_select([
	msg355,
	msg356,
]);

var msg357 = msg("183:01", dup169);

var msg358 = msg("183", dup170);

var select180 = linear_select([
	msg357,
	msg358,
]);

var msg359 = msg("184:01", dup169);

var msg360 = msg("184", dup170);

var select181 = linear_select([
	msg359,
	msg360,
]);

var msg361 = msg("185:01", dup169);

var msg362 = msg("185", dup170);

var select182 = linear_select([
	msg361,
	msg362,
]);

var msg363 = msg("186:01", dup151);

var msg364 = msg("186", dup152);

var select183 = linear_select([
	msg363,
	msg364,
]);

var msg365 = msg("187:01", dup169);

var msg366 = msg("187", dup170);

var select184 = linear_select([
	msg365,
	msg366,
]);

var msg367 = msg("188:01", dup169);

var msg368 = msg("188", dup170);

var select185 = linear_select([
	msg367,
	msg368,
]);

var msg369 = msg("189:01", dup169);

var msg370 = msg("189", dup170);

var select186 = linear_select([
	msg369,
	msg370,
]);

var msg371 = msg("191:01", dup151);

var msg372 = msg("191", dup152);

var select187 = linear_select([
	msg371,
	msg372,
]);

var msg373 = msg("192:01", dup169);

var msg374 = msg("192", dup170);

var select188 = linear_select([
	msg373,
	msg374,
]);

var msg375 = msg("193:01", dup151);

var msg376 = msg("193", dup152);

var select189 = linear_select([
	msg375,
	msg376,
]);

var msg377 = msg("194:01", dup169);

var msg378 = msg("194", dup170);

var select190 = linear_select([
	msg377,
	msg378,
]);

var msg379 = msg("195:01", dup169);

var msg380 = msg("195", dup170);

var select191 = linear_select([
	msg379,
	msg380,
]);

var msg381 = msg("196:01", dup151);

var msg382 = msg("196", dup152);

var select192 = linear_select([
	msg381,
	msg382,
]);

var msg383 = msg("197:01", dup151);

var msg384 = msg("197", dup152);

var select193 = linear_select([
	msg383,
	msg384,
]);

var msg385 = msg("198:01", dup169);

var msg386 = msg("198", dup170);

var select194 = linear_select([
	msg385,
	msg386,
]);

var msg387 = msg("199:01", dup169);

var msg388 = msg("199", dup170);

var select195 = linear_select([
	msg387,
	msg388,
]);

var msg389 = msg("200:01", dup169);

var msg390 = msg("200", dup170);

var select196 = linear_select([
	msg389,
	msg390,
]);

var msg391 = msg("201:01", dup169);

var msg392 = msg("201", dup170);

var select197 = linear_select([
	msg391,
	msg392,
]);

var msg393 = msg("202:01", dup169);

var msg394 = msg("202", dup170);

var select198 = linear_select([
	msg393,
	msg394,
]);

var msg395 = msg("203:01", dup169);

var msg396 = msg("203", dup170);

var select199 = linear_select([
	msg395,
	msg396,
]);

var msg397 = msg("204:01", dup151);

var msg398 = msg("204", dup152);

var select200 = linear_select([
	msg397,
	msg398,
]);

var msg399 = msg("205:01", dup151);

var msg400 = msg("205", dup152);

var select201 = linear_select([
	msg399,
	msg400,
]);

var msg401 = msg("206:01", dup151);

var msg402 = msg("206", dup152);

var select202 = linear_select([
	msg401,
	msg402,
]);

var msg403 = msg("207:01", dup151);

var msg404 = msg("207", dup152);

var select203 = linear_select([
	msg403,
	msg404,
]);

var msg405 = msg("208:01", dup151);

var msg406 = msg("208", dup152);

var select204 = linear_select([
	msg405,
	msg406,
]);

var msg407 = msg("209:01", dup169);

var msg408 = msg("209", dup170);

var select205 = linear_select([
	msg407,
	msg408,
]);

var msg409 = msg("211:01", dup169);

var msg410 = msg("211", dup170);

var select206 = linear_select([
	msg409,
	msg410,
]);

var msg411 = msg("212:01", dup169);

var msg412 = msg("212", dup170);

var select207 = linear_select([
	msg411,
	msg412,
]);

var msg413 = msg("213:01", dup169);

var msg414 = msg("213", dup170);

var select208 = linear_select([
	msg413,
	msg414,
]);

var msg415 = msg("214:01", dup151);

var msg416 = msg("214", dup152);

var select209 = linear_select([
	msg415,
	msg416,
]);

var msg417 = msg("215:01", dup151);

var msg418 = msg("215", dup152);

var select210 = linear_select([
	msg417,
	msg418,
]);

var msg419 = msg("216:01", dup151);

var msg420 = msg("216", dup152);

var select211 = linear_select([
	msg419,
	msg420,
]);

var msg421 = msg("217:01", dup169);

var msg422 = msg("217", dup170);

var select212 = linear_select([
	msg421,
	msg422,
]);

var msg423 = msg("218:01", dup169);

var msg424 = msg("218", dup170);

var select213 = linear_select([
	msg423,
	msg424,
]);

var msg425 = msg("219:01", dup169);

var msg426 = msg("219", dup170);

var select214 = linear_select([
	msg425,
	msg426,
]);

var msg427 = msg("220:01", dup169);

var msg428 = msg("220", dup170);

var select215 = linear_select([
	msg427,
	msg428,
]);

var msg429 = msg("221:01", dup169);

var msg430 = msg("221", dup170);

var select216 = linear_select([
	msg429,
	msg430,
]);

var msg431 = msg("222:01", dup151);

var msg432 = msg("222", dup152);

var select217 = linear_select([
	msg431,
	msg432,
]);

var msg433 = msg("223:01", dup169);

var msg434 = msg("223", dup170);

var select218 = linear_select([
	msg433,
	msg434,
]);

var msg435 = msg("224:01", dup169);

var msg436 = msg("224", dup170);

var select219 = linear_select([
	msg435,
	msg436,
]);

var msg437 = msg("229:01", dup169);

var msg438 = msg("229", dup170);

var select220 = linear_select([
	msg437,
	msg438,
]);

var msg439 = msg("230:01", dup151);

var msg440 = msg("230", dup152);

var select221 = linear_select([
	msg439,
	msg440,
]);

var msg441 = msg("231:01", dup151);

var msg442 = msg("231", dup152);

var select222 = linear_select([
	msg441,
	msg442,
]);

var msg443 = msg("232:01", dup151);

var msg444 = msg("232", dup152);

var select223 = linear_select([
	msg443,
	msg444,
]);

var msg445 = msg("233:01", dup151);

var msg446 = msg("233", dup152);

var select224 = linear_select([
	msg445,
	msg446,
]);

var msg447 = msg("236:01", dup153);

var msg448 = msg("236", dup154);

var select225 = linear_select([
	msg447,
	msg448,
]);

var msg449 = msg("237:01", dup169);

var msg450 = msg("237", dup170);

var select226 = linear_select([
	msg449,
	msg450,
]);

var msg451 = msg("238:01", dup151);

var msg452 = msg("238", dup152);

var select227 = linear_select([
	msg451,
	msg452,
]);

var msg453 = msg("239:01", dup169);

var msg454 = msg("239", dup170);

var select228 = linear_select([
	msg453,
	msg454,
]);

var msg455 = msg("240:01", dup169);

var msg456 = msg("240", dup170);

var select229 = linear_select([
	msg455,
	msg456,
]);

var msg457 = msg("241:01", dup169);

var msg458 = msg("241", dup170);

var select230 = linear_select([
	msg457,
	msg458,
]);

var msg459 = msg("243:01", dup151);

var msg460 = msg("243", dup152);

var select231 = linear_select([
	msg459,
	msg460,
]);

var msg461 = msg("244:01", dup151);

var msg462 = msg("244", dup152);

var select232 = linear_select([
	msg461,
	msg462,
]);

var msg463 = msg("246:01", dup169);

var msg464 = msg("246", dup170);

var select233 = linear_select([
	msg463,
	msg464,
]);

var msg465 = msg("247:01", dup169);

var msg466 = msg("247", dup170);

var select234 = linear_select([
	msg465,
	msg466,
]);

var msg467 = msg("248:01", dup151);

var msg468 = msg("248", dup152);

var select235 = linear_select([
	msg467,
	msg468,
]);

var msg469 = msg("249:01", dup151);

var msg470 = msg("249", dup152);

var select236 = linear_select([
	msg469,
	msg470,
]);

var msg471 = msg("250:01", dup151);

var msg472 = msg("250", dup152);

var select237 = linear_select([
	msg471,
	msg472,
]);

var msg473 = msg("251:01", dup169);

var msg474 = msg("251", dup170);

var select238 = linear_select([
	msg473,
	msg474,
]);

var msg475 = msg("252:01", dup169);

var msg476 = msg("252", dup170);

var select239 = linear_select([
	msg475,
	msg476,
]);

var msg477 = msg("253:01", dup151);

var msg478 = msg("253", dup152);

var select240 = linear_select([
	msg477,
	msg478,
]);

var msg479 = msg("254:01", dup169);

var msg480 = msg("254", dup170);

var select241 = linear_select([
	msg479,
	msg480,
]);

var msg481 = msg("255:01", dup151);

var msg482 = msg("255", dup152);

var select242 = linear_select([
	msg481,
	msg482,
]);

var msg483 = msg("256:01", dup169);

var msg484 = msg("256", dup170);

var select243 = linear_select([
	msg483,
	msg484,
]);

var msg485 = msg("257:01", dup169);

var msg486 = msg("257", dup170);

var select244 = linear_select([
	msg485,
	msg486,
]);

var msg487 = msg("259:01", dup169);

var msg488 = msg("259", dup170);

var select245 = linear_select([
	msg487,
	msg488,
]);

var msg489 = msg("260:01", dup151);

var msg490 = msg("260", dup152);

var select246 = linear_select([
	msg489,
	msg490,
]);

var msg491 = msg("261:01", dup151);

var msg492 = msg("261", dup152);

var select247 = linear_select([
	msg491,
	msg492,
]);

var msg493 = msg("262:01", dup151);

var msg494 = msg("262", dup152);

var select248 = linear_select([
	msg493,
	msg494,
]);

var msg495 = msg("263:01", dup151);

var msg496 = msg("263", dup152);

var select249 = linear_select([
	msg495,
	msg496,
]);

var msg497 = msg("264:01", dup169);

var msg498 = msg("264", dup170);

var select250 = linear_select([
	msg497,
	msg498,
]);

var msg499 = msg("265:01", dup169);

var msg500 = msg("265", dup170);

var select251 = linear_select([
	msg499,
	msg500,
]);

var msg501 = msg("266:01", dup169);

var msg502 = msg("266", dup170);

var select252 = linear_select([
	msg501,
	msg502,
]);

var msg503 = msg("267:01", dup169);

var msg504 = msg("267", dup170);

var select253 = linear_select([
	msg503,
	msg504,
]);

var msg505 = msg("268:01", dup169);

var msg506 = msg("268", dup170);

var select254 = linear_select([
	msg505,
	msg506,
]);

var msg507 = msg("269:01", dup151);

var msg508 = msg("269", dup152);

var select255 = linear_select([
	msg507,
	msg508,
]);

var msg509 = msg("270:01", dup169);

var msg510 = msg("270", dup170);

var select256 = linear_select([
	msg509,
	msg510,
]);

var msg511 = msg("271:01", dup151);

var msg512 = msg("271", dup152);

var select257 = linear_select([
	msg511,
	msg512,
]);

var msg513 = msg("272:01", dup169);

var msg514 = msg("272", dup170);

var select258 = linear_select([
	msg513,
	msg514,
]);

var msg515 = msg("273:01", dup169);

var msg516 = msg("273", dup170);

var select259 = linear_select([
	msg515,
	msg516,
]);

var msg517 = msg("274:01", dup169);

var msg518 = msg("274", dup170);

var select260 = linear_select([
	msg517,
	msg518,
]);

var msg519 = msg("275:01", dup169);

var msg520 = msg("275", dup170);

var select261 = linear_select([
	msg519,
	msg520,
]);

var msg521 = msg("276:01", dup169);

var msg522 = msg("276", dup170);

var select262 = linear_select([
	msg521,
	msg522,
]);

var msg523 = msg("277:01", dup169);

var msg524 = msg("277", dup170);

var select263 = linear_select([
	msg523,
	msg524,
]);

var msg525 = msg("278:01", dup169);

var msg526 = msg("278", dup170);

var select264 = linear_select([
	msg525,
	msg526,
]);

var msg527 = msg("279:01", dup169);

var msg528 = msg("279", dup170);

var select265 = linear_select([
	msg527,
	msg528,
]);

var msg529 = msg("280:01", dup151);

var msg530 = msg("280", dup152);

var select266 = linear_select([
	msg529,
	msg530,
]);

var msg531 = msg("281:01", dup151);

var msg532 = msg("281", dup152);

var select267 = linear_select([
	msg531,
	msg532,
]);

var msg533 = msg("282:01", dup169);

var msg534 = msg("282", dup170);

var select268 = linear_select([
	msg533,
	msg534,
]);

var msg535 = msg("283:01", dup169);

var msg536 = msg("283", dup170);

var select269 = linear_select([
	msg535,
	msg536,
]);

var msg537 = msg("284:01", dup151);

var msg538 = msg("284", dup152);

var select270 = linear_select([
	msg537,
	msg538,
]);

var msg539 = msg("285:01", dup159);

var msg540 = msg("285", dup160);

var select271 = linear_select([
	msg539,
	msg540,
]);

var msg541 = msg("286:01", dup169);

var msg542 = msg("286", dup170);

var select272 = linear_select([
	msg541,
	msg542,
]);

var msg543 = msg("287:01", dup169);

var msg544 = msg("287", dup170);

var select273 = linear_select([
	msg543,
	msg544,
]);

var msg545 = msg("288:01", dup169);

var msg546 = msg("288", dup170);

var select274 = linear_select([
	msg545,
	msg546,
]);

var msg547 = msg("289:01", dup169);

var msg548 = msg("289", dup170);

var select275 = linear_select([
	msg547,
	msg548,
]);

var msg549 = msg("290:01", dup169);

var msg550 = msg("290", dup170);

var select276 = linear_select([
	msg549,
	msg550,
]);

var msg551 = msg("291:01", dup169);

var msg552 = msg("291", dup170);

var select277 = linear_select([
	msg551,
	msg552,
]);

var msg553 = msg("292:01", dup169);

var msg554 = msg("292", dup170);

var select278 = linear_select([
	msg553,
	msg554,
]);

var msg555 = msg("293:01", dup169);

var msg556 = msg("293", dup170);

var select279 = linear_select([
	msg555,
	msg556,
]);

var msg557 = msg("294:01", dup169);

var msg558 = msg("294", dup170);

var select280 = linear_select([
	msg557,
	msg558,
]);

var msg559 = msg("295:01", dup169);

var msg560 = msg("295", dup170);

var select281 = linear_select([
	msg559,
	msg560,
]);

var msg561 = msg("296:01", dup169);

var msg562 = msg("296", dup170);

var select282 = linear_select([
	msg561,
	msg562,
]);

var msg563 = msg("297:01", dup151);

var msg564 = msg("297", dup152);

var select283 = linear_select([
	msg563,
	msg564,
]);

var msg565 = msg("298:01", dup151);

var msg566 = msg("298", dup152);

var select284 = linear_select([
	msg565,
	msg566,
]);

var msg567 = msg("299:01", dup169);

var msg568 = msg("299", dup170);

var select285 = linear_select([
	msg567,
	msg568,
]);

var part24 = match("MESSAGE#568:300:02/24", "nwparser.p0", "%{application};DstHost=%{dhost};Protocol=%{protocol};PSMID=%{fld10};SessionID=%{sessionid};SrcHost=%{shost};User=%{c_username};\"");

var all1 = all_match({
	processors: [
		dup31,
		dup172,
		dup173,
		dup174,
		dup175,
		dup176,
		dup177,
		dup178,
		dup179,
		dup180,
		dup181,
		dup182,
		dup183,
		dup184,
		dup185,
		dup186,
		dup187,
		dup188,
		dup189,
		dup190,
		dup191,
		dup192,
		dup193,
		dup194,
		part24,
	],
	on_success: processor_chain([
		dup4,
		dup2,
		dup3,
		dup24,
	]),
});

var msg569 = msg("300:02", all1);

var part25 = tagval("MESSAGE#569:300:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup4,
	dup2,
	dup3,
	dup24,
]));

var msg570 = msg("300:01", part25);

var msg571 = msg("300", dup154);

var select286 = linear_select([
	msg569,
	msg570,
	msg571,
]);

var msg572 = msg("301:01", dup163);

var msg573 = msg("301", dup164);

var select287 = linear_select([
	msg572,
	msg573,
]);

var part26 = match("MESSAGE#573:302:02/24", "nwparser.p0", "%{application};DstHost=%{dhost};Protocol=%{protocol};PSMID=%{fld12};SessionDuration=%{duration_string};SessionID=%{sessionid};SrcHost=%{shost};User=%{c_username};\"");

var all2 = all_match({
	processors: [
		dup31,
		dup172,
		dup173,
		dup174,
		dup175,
		dup176,
		dup177,
		dup178,
		dup179,
		dup180,
		dup181,
		dup182,
		dup183,
		dup184,
		dup185,
		dup186,
		dup187,
		dup188,
		dup189,
		dup190,
		dup191,
		dup192,
		dup193,
		dup194,
		part26,
	],
	on_success: processor_chain([
		dup21,
		dup2,
		dup3,
		dup24,
	]),
});

var msg574 = msg("302:02", all2);

var msg575 = msg("302:01", dup163);

var msg576 = msg("302", dup164);

var select288 = linear_select([
	msg574,
	msg575,
	msg576,
]);

var msg577 = msg("303:01", dup163);

var msg578 = msg("303", dup164);

var select289 = linear_select([
	msg577,
	msg578,
]);

var part27 = match("MESSAGE#578:304:02/23_0", "nwparser.p0", "\"%{obj_type}\";ExtraDetails=\"DstHost=%{p0}");

var part28 = match("MESSAGE#578:304:02/23_1", "nwparser.p0", "%{obj_type};ExtraDetails=\"DstHost=%{p0}");

var select290 = linear_select([
	part27,
	part28,
]);

var part29 = match("MESSAGE#578:304:02/24", "nwparser.p0", "%{dhost};Protocol=%{protocol};PSMID=%{fld10};SessionDuration=%{duration_string};SessionID=%{sessionid};SrcHost=%{shost};User=%{c_username};\"");

var all3 = all_match({
	processors: [
		dup31,
		dup172,
		dup173,
		dup174,
		dup175,
		dup176,
		dup177,
		dup178,
		dup179,
		dup180,
		dup181,
		dup182,
		dup183,
		dup184,
		dup185,
		dup186,
		dup187,
		dup188,
		dup189,
		dup190,
		dup191,
		dup192,
		dup193,
		select290,
		part29,
	],
	on_success: processor_chain([
		dup26,
		dup2,
		dup3,
		dup24,
	]),
});

var msg579 = msg("304:02", all3);

var msg580 = msg("304:01", dup169);

var msg581 = msg("304", dup170);

var select291 = linear_select([
	msg579,
	msg580,
	msg581,
]);

var msg582 = msg("305:01", dup169);

var msg583 = msg("305", dup170);

var select292 = linear_select([
	msg582,
	msg583,
]);

var msg584 = msg("306:01", dup151);

var msg585 = msg("306", dup152);

var select293 = linear_select([
	msg584,
	msg585,
]);

var msg586 = msg("307:01", dup151);

var msg587 = msg("307", dup152);

var select294 = linear_select([
	msg586,
	msg587,
]);

var part30 = tagval("MESSAGE#587:308:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup78,
	dup2,
	dup3,
]));

var msg588 = msg("308:01", part30);

var part31 = match("MESSAGE#588:308", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup78,
	dup2,
]));

var msg589 = msg("308", part31);

var select295 = linear_select([
	msg588,
	msg589,
]);

var part32 = tagval("MESSAGE#589:309:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup10,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
	dup3,
]));

var msg590 = msg("309:01", part32);

var part33 = match("MESSAGE#590:309", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup10,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
]));

var msg591 = msg("309", part33);

var select296 = linear_select([
	msg590,
	msg591,
]);

var msg592 = msg("317:01", dup195);

var msg593 = msg("317", dup196);

var select297 = linear_select([
	msg592,
	msg593,
]);

var msg594 = msg("316:01", dup195);

var msg595 = msg("316", dup196);

var select298 = linear_select([
	msg594,
	msg595,
]);

var msg596 = msg("355:01", dup197);

var msg597 = msg("355", dup198);

var select299 = linear_select([
	msg596,
	msg597,
]);

var msg598 = msg("356:01", dup197);

var msg599 = msg("356", dup198);

var select300 = linear_select([
	msg598,
	msg599,
]);

var msg600 = msg("357:01", dup199);

var msg601 = msg("357", dup200);

var select301 = linear_select([
	msg600,
	msg601,
]);

var msg602 = msg("358:01", dup199);

var msg603 = msg("358", dup200);

var select302 = linear_select([
	msg602,
	msg603,
]);

var part34 = tagval("MESSAGE#603:190:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup84,
	dup2,
	dup3,
]));

var msg604 = msg("190:01", part34);

var part35 = match("MESSAGE#604:190", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup84,
	dup2,
]));

var msg605 = msg("190", part35);

var select303 = linear_select([
	msg604,
	msg605,
]);

var msg606 = msg("5:01", dup161);

var msg607 = msg("5", dup162);

var select304 = linear_select([
	msg606,
	msg607,
]);

var msg608 = msg("310:01", dup153);

var msg609 = msg("310", dup154);

var select305 = linear_select([
	msg608,
	msg609,
]);

var msg610 = msg("311:01", dup153);

var msg611 = msg("311", dup154);

var select306 = linear_select([
	msg610,
	msg611,
]);

var msg612 = msg("312:01", dup153);

var msg613 = msg("312", dup154);

var select307 = linear_select([
	msg612,
	msg613,
]);

var msg614 = msg("313:01", dup153);

var msg615 = msg("313", dup154);

var select308 = linear_select([
	msg614,
	msg615,
]);

var msg616 = msg("359:01", dup153);

var msg617 = msg("359", dup154);

var select309 = linear_select([
	msg616,
	msg617,
]);

var msg618 = msg("372", dup201);

var msg619 = msg("374", dup201);

var msg620 = msg("376", dup201);

var part36 = match("MESSAGE#620:411:01/17_0", "nwparser.p0", "\"%{fld89}\";LogonDomain=%{p0}");

var part37 = match("MESSAGE#620:411:01/17_1", "nwparser.p0", "%{fld89};LogonDomain=%{p0}");

var select310 = linear_select([
	part36,
	part37,
]);

var part38 = match("MESSAGE#620:411:01/23_0", "nwparser.p0", "\"%{obj_type}\";ExtraDetails=\"Command=%{p0}");

var part39 = match("MESSAGE#620:411:01/23_1", "nwparser.p0", "%{obj_type};ExtraDetails=\"Command=%{p0}");

var select311 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#620:411:01/24", "nwparser.p0", "%{param};ConnectionComponentId=%{fld67};DstHost=%{dhost};Protocol=%{protocol};PSMID=%{fld11};RDPOffset=%{fld12};SessionID=%{sessionid};SrcHost=%{shost};User=%{c_username};VIDOffset=%{fld13};");

var all4 = all_match({
	processors: [
		dup31,
		dup172,
		dup173,
		dup174,
		dup175,
		dup176,
		dup177,
		dup178,
		dup179,
		dup180,
		dup181,
		dup182,
		dup183,
		dup184,
		dup185,
		dup186,
		dup187,
		select310,
		dup189,
		dup190,
		dup191,
		dup192,
		dup193,
		select311,
		part40,
	],
	on_success: processor_chain([
		dup4,
		dup2,
		dup3,
		dup24,
	]),
});

var msg621 = msg("411:01", all4);

var part41 = match("MESSAGE#621:411/43_0", "nwparser.p0", "\"Command=%{param};ConnectionComponentId=%{fld1};DstHost=%{fld2};ProcessId=%{process_id};ProcessName=%{process};Protocol=%{protocol};PSMID=%{fld3};RDPOffset=%{fld4};SessionID=%{sessionid};SrcHost=%{shost};User=%{fld5};VIDOffset=%{fld6};\"");

var select312 = linear_select([
	part41,
	dup150,
]);

var all5 = all_match({
	processors: [
		dup31,
		dup202,
		dup87,
		dup203,
		dup90,
		dup204,
		dup93,
		dup205,
		dup96,
		dup206,
		dup99,
		dup207,
		dup102,
		dup208,
		dup105,
		dup209,
		dup108,
		dup210,
		dup111,
		dup211,
		dup114,
		dup212,
		dup119,
		dup213,
		dup122,
		dup214,
		dup125,
		dup215,
		dup128,
		dup216,
		dup131,
		dup217,
		dup134,
		dup218,
		dup137,
		dup219,
		dup140,
		dup220,
		dup143,
		dup221,
		dup146,
		dup222,
		dup149,
		select312,
	],
	on_success: processor_chain([
		dup4,
		dup2,
		dup3,
	]),
});

var msg622 = msg("411", all5);

var select313 = linear_select([
	msg621,
	msg622,
]);

var part42 = match("MESSAGE#622:385", "nwparser.payload", "Version=%{version};Message=%{action};Issuer=%{username};Station=%{hostip};File=%{filename};Safe=%{group_object};Location=\"%{directory}\";Category=%{category};RequestId=%{id1};Reason=%{event_description};Severity=%{severity};GatewayStation=%{saddr};TicketID=%{operation_id};PolicyID=%{policyname};UserName=%{c_username};LogonDomain=%{domain};Address=%{dhost};CPMStatus=%{disposition};Port=\"%{dport}\";Database=%{db_name};DeviceType=%{obj_type};ExtraDetails=%{info}", processor_chain([
	dup4,
	dup2,
	dup3,
]));

var msg623 = msg("385", part42);

var part43 = match("MESSAGE#623:361/43_0", "nwparser.p0", "\"Command=%{param};ConnectionComponentId=%{fld1};DstHost=%{fld2};Protocol=%{protocol};PSMID=%{fld3};SessionID=%{sessionid};SrcHost=%{shost};SSHOffset=%{fld4};User=%{fld5};VIDOffset=%{fld6};\"");

var select314 = linear_select([
	part43,
	dup150,
]);

var all6 = all_match({
	processors: [
		dup31,
		dup202,
		dup87,
		dup203,
		dup90,
		dup204,
		dup93,
		dup205,
		dup96,
		dup206,
		dup99,
		dup207,
		dup102,
		dup208,
		dup105,
		dup209,
		dup108,
		dup210,
		dup111,
		dup211,
		dup114,
		dup212,
		dup119,
		dup213,
		dup122,
		dup214,
		dup125,
		dup215,
		dup128,
		dup216,
		dup131,
		dup217,
		dup134,
		dup218,
		dup137,
		dup219,
		dup140,
		dup220,
		dup143,
		dup221,
		dup146,
		dup222,
		dup149,
		select314,
	],
	on_success: processor_chain([
		dup4,
		dup2,
		dup3,
	]),
});

var msg624 = msg("361", all6);

var part44 = match("MESSAGE#624:412/43_0", "nwparser.p0", "\"Command=%{param};ConnectionComponentId=%{fld1};DstHost=%{fld2};Protocol=%{protocol};PSMID=%{fld3};SessionID=%{sessionid};SrcHost=%{shost};TXTOffset=%{fld4};User=%{fld5};VIDOffset=%{fld6};\"");

var select315 = linear_select([
	part44,
	dup150,
]);

var all7 = all_match({
	processors: [
		dup31,
		dup202,
		dup87,
		dup203,
		dup90,
		dup204,
		dup93,
		dup205,
		dup96,
		dup206,
		dup99,
		dup207,
		dup102,
		dup208,
		dup105,
		dup209,
		dup108,
		dup210,
		dup111,
		dup211,
		dup114,
		dup212,
		dup119,
		dup213,
		dup122,
		dup214,
		dup125,
		dup215,
		dup128,
		dup216,
		dup131,
		dup217,
		dup134,
		dup218,
		dup137,
		dup219,
		dup140,
		dup220,
		dup143,
		dup221,
		dup146,
		dup222,
		dup149,
		select315,
	],
	on_success: processor_chain([
		dup4,
		dup2,
		dup3,
	]),
});

var msg625 = msg("412", all7);

var msg626 = msg("378", dup153);

var msg627 = msg("321", dup153);

var msg628 = msg("322", dup153);

var msg629 = msg("323", dup153);

var msg630 = msg("318", dup153);

var msg631 = msg("380", dup153);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"1": select2,
		"10": select9,
		"100": select99,
		"101": select100,
		"102": select101,
		"103": select102,
		"104": select103,
		"105": select104,
		"106": select105,
		"107": select106,
		"108": select107,
		"109": select108,
		"11": select10,
		"110": select109,
		"111": select110,
		"112": select111,
		"114": select112,
		"115": select113,
		"116": select114,
		"117": select115,
		"118": select116,
		"119": select117,
		"12": select11,
		"120": select118,
		"121": select119,
		"122": select120,
		"123": select121,
		"124": select122,
		"125": select123,
		"126": select124,
		"127": select125,
		"128": select126,
		"129": select127,
		"13": select12,
		"130": select128,
		"131": select129,
		"132": select130,
		"133": select131,
		"134": select132,
		"135": select133,
		"136": select134,
		"137": select135,
		"138": select136,
		"139": select137,
		"14": select13,
		"140": select138,
		"141": select139,
		"142": select140,
		"143": select141,
		"144": select142,
		"145": select143,
		"146": select144,
		"147": select145,
		"148": select146,
		"149": select147,
		"15": select14,
		"150": select148,
		"152": select149,
		"153": select150,
		"154": select151,
		"155": select152,
		"156": select153,
		"157": select154,
		"158": select155,
		"159": select156,
		"16": select15,
		"160": select157,
		"161": select158,
		"162": select159,
		"163": select160,
		"164": select161,
		"165": select162,
		"166": select163,
		"167": select164,
		"168": select165,
		"169": select166,
		"17": select16,
		"170": select167,
		"171": select168,
		"172": select169,
		"173": select170,
		"174": select171,
		"175": select172,
		"176": select173,
		"177": select174,
		"178": select175,
		"179": select176,
		"18": select17,
		"180": select177,
		"181": select178,
		"182": select179,
		"183": select180,
		"184": select181,
		"185": select182,
		"186": select183,
		"187": select184,
		"188": select185,
		"189": select186,
		"19": select18,
		"190": select303,
		"191": select187,
		"192": select188,
		"193": select189,
		"194": select190,
		"195": select191,
		"196": select192,
		"197": select193,
		"198": select194,
		"199": select195,
		"2": select3,
		"20": select19,
		"200": select196,
		"201": select197,
		"202": select198,
		"203": select199,
		"204": select200,
		"205": select201,
		"206": select202,
		"207": select203,
		"208": select204,
		"209": select205,
		"21": select20,
		"211": select206,
		"212": select207,
		"213": select208,
		"214": select209,
		"215": select210,
		"216": select211,
		"217": select212,
		"218": select213,
		"219": select214,
		"22": select21,
		"220": select215,
		"221": select216,
		"222": select217,
		"223": select218,
		"224": select219,
		"229": select220,
		"23": select22,
		"230": select221,
		"231": select222,
		"232": select223,
		"233": select224,
		"236": select225,
		"237": select226,
		"238": select227,
		"239": select228,
		"24": select23,
		"240": select229,
		"241": select230,
		"243": select231,
		"244": select232,
		"246": select233,
		"247": select234,
		"248": select235,
		"249": select236,
		"25": select24,
		"250": select237,
		"251": select238,
		"252": select239,
		"253": select240,
		"254": select241,
		"255": select242,
		"256": select243,
		"257": select244,
		"259": select245,
		"26": select25,
		"260": select246,
		"261": select247,
		"262": select248,
		"263": select249,
		"264": select250,
		"265": select251,
		"266": select252,
		"267": select253,
		"268": select254,
		"269": select255,
		"27": select26,
		"270": select256,
		"271": select257,
		"272": select258,
		"273": select259,
		"274": select260,
		"275": select261,
		"276": select262,
		"277": select263,
		"278": select264,
		"279": select265,
		"28": select27,
		"280": select266,
		"281": select267,
		"282": select268,
		"283": select269,
		"284": select270,
		"285": select271,
		"286": select272,
		"287": select273,
		"288": select274,
		"289": select275,
		"29": select28,
		"290": select276,
		"291": select277,
		"292": select278,
		"293": select279,
		"294": select280,
		"295": select281,
		"296": select282,
		"297": select283,
		"298": select284,
		"299": select285,
		"3": select4,
		"30": select29,
		"300": select286,
		"301": select287,
		"302": select288,
		"303": select289,
		"304": select291,
		"305": select292,
		"306": select293,
		"307": select294,
		"308": select295,
		"309": select296,
		"31": select30,
		"310": select305,
		"311": select306,
		"312": select307,
		"313": select308,
		"316": select298,
		"317": select297,
		"318": msg630,
		"32": select31,
		"321": msg627,
		"322": msg628,
		"323": msg629,
		"33": select32,
		"34": select33,
		"35": select34,
		"355": select299,
		"356": select300,
		"357": select301,
		"358": select302,
		"359": select309,
		"36": select35,
		"361": msg624,
		"37": select36,
		"372": msg618,
		"374": msg619,
		"376": msg620,
		"378": msg626,
		"38": select37,
		"380": msg631,
		"385": msg623,
		"39": select38,
		"4": select5,
		"40": select39,
		"41": select40,
		"411": select313,
		"412": msg625,
		"42": select41,
		"43": select42,
		"44": select43,
		"45": select44,
		"46": select45,
		"47": select46,
		"48": select47,
		"49": select48,
		"5": select304,
		"50": select49,
		"51": select50,
		"52": select51,
		"53": select52,
		"54": select53,
		"55": select54,
		"56": select55,
		"57": select56,
		"58": select57,
		"59": select58,
		"60": select59,
		"61": select60,
		"62": select61,
		"63": select62,
		"64": select63,
		"65": select64,
		"66": select65,
		"67": select66,
		"68": select67,
		"69": select68,
		"7": select6,
		"70": select69,
		"71": select70,
		"72": select71,
		"73": select72,
		"74": select73,
		"75": select74,
		"76": select75,
		"77": select76,
		"78": select77,
		"79": select78,
		"8": select7,
		"80": select79,
		"81": select80,
		"82": select81,
		"83": select82,
		"84": select83,
		"85": select84,
		"86": select85,
		"87": select86,
		"88": select87,
		"89": select88,
		"9": select8,
		"90": select89,
		"91": select90,
		"92": select91,
		"93": select92,
		"94": select93,
		"95": select94,
		"96": select95,
		"97": select96,
		"98": select97,
		"99": select98,
	}),
]);

var part45 = match("MESSAGE#568:300:02/0", "nwparser.payload", "Version=%{p0}");

var part46 = match("MESSAGE#568:300:02/1_0", "nwparser.p0", "\"%{version}\";Message=%{p0}");

var part47 = match("MESSAGE#568:300:02/1_1", "nwparser.p0", "%{version};Message=%{p0}");

var part48 = match("MESSAGE#568:300:02/2_0", "nwparser.p0", "\"%{action}\";Issuer=%{p0}");

var part49 = match("MESSAGE#568:300:02/2_1", "nwparser.p0", "%{action};Issuer=%{p0}");

var part50 = match("MESSAGE#568:300:02/3_0", "nwparser.p0", "\"%{username}\";Station=%{p0}");

var part51 = match("MESSAGE#568:300:02/3_1", "nwparser.p0", "%{username};Station=%{p0}");

var part52 = match("MESSAGE#568:300:02/4_0", "nwparser.p0", "\"%{hostip}\";File=%{p0}");

var part53 = match("MESSAGE#568:300:02/4_1", "nwparser.p0", "%{hostip};File=%{p0}");

var part54 = match("MESSAGE#568:300:02/5_0", "nwparser.p0", "\"%{filename}\";Safe=%{p0}");

var part55 = match("MESSAGE#568:300:02/5_1", "nwparser.p0", "%{filename};Safe=%{p0}");

var part56 = match("MESSAGE#568:300:02/6_0", "nwparser.p0", "\"%{group_object}\";Location=%{p0}");

var part57 = match("MESSAGE#568:300:02/6_1", "nwparser.p0", "%{group_object};Location=%{p0}");

var part58 = match("MESSAGE#568:300:02/7_0", "nwparser.p0", "\"%{directory}\";Category=%{p0}");

var part59 = match("MESSAGE#568:300:02/7_1", "nwparser.p0", "%{directory};Category=%{p0}");

var part60 = match("MESSAGE#568:300:02/8_0", "nwparser.p0", "\"%{category}\";RequestId=%{p0}");

var part61 = match("MESSAGE#568:300:02/8_1", "nwparser.p0", "%{category};RequestId=%{p0}");

var part62 = match("MESSAGE#568:300:02/9_0", "nwparser.p0", "\"%{id1}\";Reason=%{p0}");

var part63 = match("MESSAGE#568:300:02/9_1", "nwparser.p0", "%{id1};Reason=%{p0}");

var part64 = match("MESSAGE#568:300:02/10_0", "nwparser.p0", "\"%{event_description}\";Severity=%{p0}");

var part65 = match("MESSAGE#568:300:02/10_1", "nwparser.p0", "%{event_description};Severity=%{p0}");

var part66 = match("MESSAGE#568:300:02/11_0", "nwparser.p0", "\"%{severity}\";SourceUser=%{p0}");

var part67 = match("MESSAGE#568:300:02/11_1", "nwparser.p0", "%{severity};SourceUser=%{p0}");

var part68 = match("MESSAGE#568:300:02/12_0", "nwparser.p0", "\"%{group}\";TargetUser=%{p0}");

var part69 = match("MESSAGE#568:300:02/12_1", "nwparser.p0", "%{group};TargetUser=%{p0}");

var part70 = match("MESSAGE#568:300:02/13_0", "nwparser.p0", "\"%{uid}\";GatewayStation=%{p0}");

var part71 = match("MESSAGE#568:300:02/13_1", "nwparser.p0", "%{uid};GatewayStation=%{p0}");

var part72 = match("MESSAGE#568:300:02/14_0", "nwparser.p0", "\"%{saddr}\";TicketID=%{p0}");

var part73 = match("MESSAGE#568:300:02/14_1", "nwparser.p0", "%{saddr};TicketID=%{p0}");

var part74 = match("MESSAGE#568:300:02/15_0", "nwparser.p0", "\"%{operation_id}\";PolicyID=%{p0}");

var part75 = match("MESSAGE#568:300:02/15_1", "nwparser.p0", "%{operation_id};PolicyID=%{p0}");

var part76 = match("MESSAGE#568:300:02/16_0", "nwparser.p0", "\"%{policyname}\";UserName=%{p0}");

var part77 = match("MESSAGE#568:300:02/16_1", "nwparser.p0", "%{policyname};UserName=%{p0}");

var part78 = match("MESSAGE#568:300:02/17_0", "nwparser.p0", "\"%{fld11}\";LogonDomain=%{p0}");

var part79 = match("MESSAGE#568:300:02/17_1", "nwparser.p0", "%{fld11};LogonDomain=%{p0}");

var part80 = match("MESSAGE#568:300:02/18_0", "nwparser.p0", "\"%{domain}\";Address=%{p0}");

var part81 = match("MESSAGE#568:300:02/18_1", "nwparser.p0", "%{domain};Address=%{p0}");

var part82 = match("MESSAGE#568:300:02/19_0", "nwparser.p0", "\"%{fld14}\";CPMStatus=%{p0}");

var part83 = match("MESSAGE#568:300:02/19_1", "nwparser.p0", "%{fld14};CPMStatus=%{p0}");

var part84 = match("MESSAGE#568:300:02/20_0", "nwparser.p0", "\"%{disposition}\";Port=%{p0}");

var part85 = match("MESSAGE#568:300:02/20_1", "nwparser.p0", "%{disposition};Port=%{p0}");

var part86 = match("MESSAGE#568:300:02/21_0", "nwparser.p0", "\"%{dport}\";Database=%{p0}");

var part87 = match("MESSAGE#568:300:02/21_1", "nwparser.p0", "%{dport};Database=%{p0}");

var part88 = match("MESSAGE#568:300:02/22_0", "nwparser.p0", "\"%{db_name}\";DeviceType=%{p0}");

var part89 = match("MESSAGE#568:300:02/22_1", "nwparser.p0", "%{db_name};DeviceType=%{p0}");

var part90 = match("MESSAGE#568:300:02/23_0", "nwparser.p0", "\"%{obj_type}\";ExtraDetails=\"ApplicationType=%{p0}");

var part91 = match("MESSAGE#568:300:02/23_1", "nwparser.p0", "%{obj_type};ExtraDetails=\"ApplicationType=%{p0}");

var part92 = match("MESSAGE#621:411/1_0", "nwparser.p0", "\"%{version}\";%{p0}");

var part93 = match("MESSAGE#621:411/1_1", "nwparser.p0", "%{version};%{p0}");

var part94 = match("MESSAGE#621:411/2", "nwparser.p0", "Message=%{p0}");

var part95 = match("MESSAGE#621:411/3_0", "nwparser.p0", "\"%{action}\";%{p0}");

var part96 = match("MESSAGE#621:411/3_1", "nwparser.p0", "%{action};%{p0}");

var part97 = match("MESSAGE#621:411/4", "nwparser.p0", "Issuer=%{p0}");

var part98 = match("MESSAGE#621:411/5_0", "nwparser.p0", "\"%{username}\";%{p0}");

var part99 = match("MESSAGE#621:411/5_1", "nwparser.p0", "%{username};%{p0}");

var part100 = match("MESSAGE#621:411/6", "nwparser.p0", "Station=%{p0}");

var part101 = match("MESSAGE#621:411/7_0", "nwparser.p0", "\"%{hostip}\";%{p0}");

var part102 = match("MESSAGE#621:411/7_1", "nwparser.p0", "%{hostip};%{p0}");

var part103 = match("MESSAGE#621:411/8", "nwparser.p0", "File=%{p0}");

var part104 = match("MESSAGE#621:411/9_0", "nwparser.p0", "\"%{filename}\";%{p0}");

var part105 = match("MESSAGE#621:411/9_1", "nwparser.p0", "%{filename};%{p0}");

var part106 = match("MESSAGE#621:411/10", "nwparser.p0", "Safe=%{p0}");

var part107 = match("MESSAGE#621:411/11_0", "nwparser.p0", "\"%{group_object}\";%{p0}");

var part108 = match("MESSAGE#621:411/11_1", "nwparser.p0", "%{group_object};%{p0}");

var part109 = match("MESSAGE#621:411/12", "nwparser.p0", "Location=%{p0}");

var part110 = match("MESSAGE#621:411/13_0", "nwparser.p0", "\"%{directory}\";%{p0}");

var part111 = match("MESSAGE#621:411/13_1", "nwparser.p0", "%{directory};%{p0}");

var part112 = match("MESSAGE#621:411/14", "nwparser.p0", "Category=%{p0}");

var part113 = match("MESSAGE#621:411/15_0", "nwparser.p0", "\"%{category}\";%{p0}");

var part114 = match("MESSAGE#621:411/15_1", "nwparser.p0", "%{category};%{p0}");

var part115 = match("MESSAGE#621:411/16", "nwparser.p0", "RequestId=%{p0}");

var part116 = match("MESSAGE#621:411/17_0", "nwparser.p0", "\"%{id1}\";%{p0}");

var part117 = match("MESSAGE#621:411/17_1", "nwparser.p0", "%{id1};%{p0}");

var part118 = match("MESSAGE#621:411/18", "nwparser.p0", "Reason=%{p0}");

var part119 = match("MESSAGE#621:411/19_0", "nwparser.p0", "\"%{event_description}\";%{p0}");

var part120 = match("MESSAGE#621:411/19_1", "nwparser.p0", "%{event_description};%{p0}");

var part121 = match("MESSAGE#621:411/20", "nwparser.p0", "Severity=%{p0}");

var part122 = match("MESSAGE#621:411/21_0", "nwparser.p0", "\"%{severity}\";SourceUser=\"%{group}\";TargetUser=\"%{uid}\";%{p0}");

var part123 = match("MESSAGE#621:411/21_1", "nwparser.p0", "%{severity};SourceUser=%{group};TargetUser=%{uid};%{p0}");

var part124 = match("MESSAGE#621:411/21_2", "nwparser.p0", "\"%{severity}\";%{p0}");

var part125 = match("MESSAGE#621:411/21_3", "nwparser.p0", "%{severity};%{p0}");

var part126 = match("MESSAGE#621:411/22", "nwparser.p0", "GatewayStation=%{p0}");

var part127 = match("MESSAGE#621:411/23_0", "nwparser.p0", "\"%{saddr}\";%{p0}");

var part128 = match("MESSAGE#621:411/23_1", "nwparser.p0", "%{saddr};%{p0}");

var part129 = match("MESSAGE#621:411/24", "nwparser.p0", "TicketID=%{p0}");

var part130 = match("MESSAGE#621:411/25_0", "nwparser.p0", "\"%{operation_id}\";%{p0}");

var part131 = match("MESSAGE#621:411/25_1", "nwparser.p0", "%{operation_id};%{p0}");

var part132 = match("MESSAGE#621:411/26", "nwparser.p0", "PolicyID=%{p0}");

var part133 = match("MESSAGE#621:411/27_0", "nwparser.p0", "\"%{policyname}\";%{p0}");

var part134 = match("MESSAGE#621:411/27_1", "nwparser.p0", "%{policyname};%{p0}");

var part135 = match("MESSAGE#621:411/28", "nwparser.p0", "UserName=%{p0}");

var part136 = match("MESSAGE#621:411/29_0", "nwparser.p0", "\"%{c_username}\";%{p0}");

var part137 = match("MESSAGE#621:411/29_1", "nwparser.p0", "%{c_username};%{p0}");

var part138 = match("MESSAGE#621:411/30", "nwparser.p0", "LogonDomain=%{p0}");

var part139 = match("MESSAGE#621:411/31_0", "nwparser.p0", "\"%{domain}\";%{p0}");

var part140 = match("MESSAGE#621:411/31_1", "nwparser.p0", "%{domain};%{p0}");

var part141 = match("MESSAGE#621:411/32", "nwparser.p0", "Address=%{p0}");

var part142 = match("MESSAGE#621:411/33_0", "nwparser.p0", "\"%{dhost}\";%{p0}");

var part143 = match("MESSAGE#621:411/33_1", "nwparser.p0", "%{dhost};%{p0}");

var part144 = match("MESSAGE#621:411/34", "nwparser.p0", "CPMStatus=%{p0}");

var part145 = match("MESSAGE#621:411/35_0", "nwparser.p0", "\"%{disposition}\";%{p0}");

var part146 = match("MESSAGE#621:411/35_1", "nwparser.p0", "%{disposition};%{p0}");

var part147 = match("MESSAGE#621:411/36", "nwparser.p0", "Port=%{p0}");

var part148 = match("MESSAGE#621:411/37_0", "nwparser.p0", "\"%{dport}\";%{p0}");

var part149 = match("MESSAGE#621:411/37_1", "nwparser.p0", "%{dport};%{p0}");

var part150 = match("MESSAGE#621:411/38", "nwparser.p0", "Database=%{p0}");

var part151 = match("MESSAGE#621:411/39_0", "nwparser.p0", "\"%{db_name}\";%{p0}");

var part152 = match("MESSAGE#621:411/39_1", "nwparser.p0", "%{db_name};%{p0}");

var part153 = match("MESSAGE#621:411/40", "nwparser.p0", "DeviceType=%{p0}");

var part154 = match("MESSAGE#621:411/41_0", "nwparser.p0", "\"%{obj_type}\";%{p0}");

var part155 = match("MESSAGE#621:411/41_1", "nwparser.p0", "%{obj_type};%{p0}");

var part156 = match("MESSAGE#621:411/42", "nwparser.p0", "ExtraDetails=%{p0}");

var part157 = match("MESSAGE#621:411/43_1", "nwparser.p0", "%{info};");

var part158 = tagval("MESSAGE#0:1:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup1,
	dup2,
	dup3,
]));

var part159 = match("MESSAGE#1:1", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup1,
	dup2,
]));

var part160 = tagval("MESSAGE#2:2:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup4,
	dup2,
	dup3,
]));

var part161 = match("MESSAGE#3:2", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup4,
	dup2,
]));

var part162 = tagval("MESSAGE#6:4:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
	dup3,
]));

var part163 = match("MESSAGE#7:4", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup2,
]));

var part164 = tagval("MESSAGE#20:13:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup16,
	dup17,
	dup9,
	dup2,
	dup3,
]));

var part165 = match("MESSAGE#21:13", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup16,
	dup17,
	dup9,
	dup2,
]));

var part166 = tagval("MESSAGE#26:16:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup19,
	dup2,
	dup3,
]));

var part167 = match("MESSAGE#27:16", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup19,
	dup2,
]));

var part168 = tagval("MESSAGE#30:18:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup15,
	dup2,
	dup3,
]));

var part169 = match("MESSAGE#31:18", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup15,
	dup2,
]));

var part170 = tagval("MESSAGE#38:22:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup21,
	dup2,
	dup3,
]));

var part171 = match("MESSAGE#39:22", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup21,
	dup2,
]));

var part172 = tagval("MESSAGE#70:38:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup23,
	dup2,
	dup3,
]));

var part173 = match("MESSAGE#71:38", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup23,
	dup2,
]));

var part174 = tagval("MESSAGE#116:61:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup20,
	dup2,
	dup3,
]));

var part175 = match("MESSAGE#117:61", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup20,
	dup2,
]));

var part176 = tagval("MESSAGE#126:66:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup26,
	dup2,
	dup3,
]));

var part177 = match("MESSAGE#127:66", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup26,
	dup2,
]));

var part178 = tagval("MESSAGE#190:98:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup26,
	dup2,
	dup3,
	dup24,
	dup25,
]));

var select316 = linear_select([
	dup32,
	dup33,
]);

var select317 = linear_select([
	dup34,
	dup35,
]);

var select318 = linear_select([
	dup36,
	dup37,
]);

var select319 = linear_select([
	dup38,
	dup39,
]);

var select320 = linear_select([
	dup40,
	dup41,
]);

var select321 = linear_select([
	dup42,
	dup43,
]);

var select322 = linear_select([
	dup44,
	dup45,
]);

var select323 = linear_select([
	dup46,
	dup47,
]);

var select324 = linear_select([
	dup48,
	dup49,
]);

var select325 = linear_select([
	dup50,
	dup51,
]);

var select326 = linear_select([
	dup52,
	dup53,
]);

var select327 = linear_select([
	dup54,
	dup55,
]);

var select328 = linear_select([
	dup56,
	dup57,
]);

var select329 = linear_select([
	dup58,
	dup59,
]);

var select330 = linear_select([
	dup60,
	dup61,
]);

var select331 = linear_select([
	dup62,
	dup63,
]);

var select332 = linear_select([
	dup64,
	dup65,
]);

var select333 = linear_select([
	dup66,
	dup67,
]);

var select334 = linear_select([
	dup68,
	dup69,
]);

var select335 = linear_select([
	dup70,
	dup71,
]);

var select336 = linear_select([
	dup72,
	dup73,
]);

var select337 = linear_select([
	dup74,
	dup75,
]);

var select338 = linear_select([
	dup76,
	dup77,
]);

var part179 = tagval("MESSAGE#591:317:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup79,
	dup80,
	dup81,
	dup2,
	dup3,
]));

var part180 = match("MESSAGE#592:317", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup79,
	dup80,
	dup81,
	dup2,
]));

var part181 = tagval("MESSAGE#595:355:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup82,
	dup2,
	dup3,
]));

var part182 = match("MESSAGE#596:355", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup82,
	dup2,
]));

var part183 = tagval("MESSAGE#599:357:01", "nwparser.payload", tvm, {
	"Address": "dhost",
	"CPMStatus": "disposition",
	"Category": "category",
	"Database": "db_name",
	"DeviceType": "obj_type",
	"ExtraDetails": "info",
	"File": "filename",
	"GatewayStation": "saddr",
	"Issuer": "username",
	"Location": "directory",
	"LogonDomain": "domain",
	"Message": "action",
	"PolicyID": "policyname",
	"Port": "dport",
	"Reason": "event_description",
	"RequestId": "id1",
	"Safe": "group_object",
	"Severity": "severity",
	"SourceUser": "group",
	"Station": "hostip",
	"TargetUser": "uid",
	"TicketID": "operation_id",
	"UserName": "c_username",
	"Version": "version",
}, processor_chain([
	dup83,
	dup2,
	dup3,
]));

var part184 = match("MESSAGE#600:357", "nwparser.payload", "%{product->} %{version}\",ProductAccount=\"%{service_account}\",ProductProcess=\"%{fld2}\",EventId=\"%{id}\",EventClass=\"%{fld3}\",EventSeverity=\"%{severity}\",EventMessage=\"%{action}\",ActingUserName=\"%{username}\",ActingAddress=\"%{hostip}\",ActionSourceUser=\"%{fld4}\",ActionTargetUser=\"%{c_username}\",ActionObject=\"%{filename}\",ActionSafe=\"%{group_object}\",ActionLocation=\"%{directory}\",ActionCategory=\"%{category}\",ActionRequestId=\"%{id1}\",ActionReason=\"%{event_description}\",ActionExtraDetails=\"%{info}\"", processor_chain([
	dup83,
	dup2,
]));

var part185 = match("MESSAGE#617:372", "nwparser.payload", "Version=%{version};Message=%{action};Issuer=%{username};Station=%{hostip};File=%{filename};Safe=%{group_object};Location=%{directory};Category=%{category};RequestId=%{id1};Reason=%{event_description};Severity=%{severity};GatewayStation=%{saddr};TicketID=%{operation_id};PolicyID=%{policyname};UserName=%{c_username};LogonDomain=%{domain};Address=%{dhost};CPMStatus=%{disposition};Port=\"%{dport}\";Database=%{db_name};DeviceType=%{obj_type};ExtraDetails=%{info};", processor_chain([
	dup4,
	dup2,
	dup3,
]));

var select339 = linear_select([
	dup85,
	dup86,
]);

var select340 = linear_select([
	dup88,
	dup89,
]);

var select341 = linear_select([
	dup91,
	dup92,
]);

var select342 = linear_select([
	dup94,
	dup95,
]);

var select343 = linear_select([
	dup97,
	dup98,
]);

var select344 = linear_select([
	dup100,
	dup101,
]);

var select345 = linear_select([
	dup103,
	dup104,
]);

var select346 = linear_select([
	dup106,
	dup107,
]);

var select347 = linear_select([
	dup109,
	dup110,
]);

var select348 = linear_select([
	dup112,
	dup113,
]);

var select349 = linear_select([
	dup115,
	dup116,
	dup117,
	dup118,
]);

var select350 = linear_select([
	dup120,
	dup121,
]);

var select351 = linear_select([
	dup123,
	dup124,
]);

var select352 = linear_select([
	dup126,
	dup127,
]);

var select353 = linear_select([
	dup129,
	dup130,
]);

var select354 = linear_select([
	dup132,
	dup133,
]);

var select355 = linear_select([
	dup135,
	dup136,
]);

var select356 = linear_select([
	dup138,
	dup139,
]);

var select357 = linear_select([
	dup141,
	dup142,
]);

var select358 = linear_select([
	dup144,
	dup145,
]);

var select359 = linear_select([
	dup147,
	dup148,
]);
