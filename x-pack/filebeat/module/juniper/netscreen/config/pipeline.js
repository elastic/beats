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

var map_dir2SumType = {
	keyvaluepairs: {
		"0": constant("2"),
		"1": constant("3"),
	},
	"default": constant("2"),
};

var map_dir2Addr = {
	keyvaluepairs: {
		"0": field("saddr"),
		"1": field("daddr"),
	},
	"default": field("saddr"),
};

var map_dir2Port = {
	keyvaluepairs: {
		"0": field("sport"),
		"1": field("dport"),
	},
	"default": field("sport"),
};

var dup1 = setc("eventcategory","1701000000");

var dup2 = setf("hardware_id","hfld2");

var dup3 = setf("vsys","hvsys");

var dup4 = setf("msg","$MSG");

var dup5 = setf("severity","hseverity");

var dup6 = match("MESSAGE#2:00001:02/0", "nwparser.payload", "Address %{group_object->} for %{p0}");

var dup7 = match("MESSAGE#2:00001:02/1_1", "nwparser.p0", "domain address %{domain->} in zone %{p0}");

var dup8 = match("MESSAGE#4:00001:04/3_0", "nwparser.p0", " (%{fld1})");

var dup9 = date_time({
	dest: "event_time",
	args: ["fld1"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup10 = match("MESSAGE#5:00001:05/1_0", "nwparser.p0", "(%{fld1})");

var dup11 = match_copy("MESSAGE#5:00001:05/1_1", "nwparser.p0", "fld1");

var dup12 = match("MESSAGE#8:00001:08/0", "nwparser.payload", "Address %{p0}");

var dup13 = match("MESSAGE#8:00001:08/1_0", "nwparser.p0", "MIP(%{interface}) %{p0}");

var dup14 = match("MESSAGE#8:00001:08/1_1", "nwparser.p0", "%{group_object->} %{p0}");

var dup15 = match("MESSAGE#8:00001:08/3_0", "nwparser.p0", "admin %{p0}");

var dup16 = match_copy("MESSAGE#8:00001:08/3_1", "nwparser.p0", "p0");

var dup17 = setc("eventcategory","1502000000");

var dup18 = setc("eventcategory","1703000000");

var dup19 = setc("eventcategory","1603000000");

var dup20 = match("MESSAGE#25:00002:20/1_1", "nwparser.p0", "from host %{saddr->} ");

var dup21 = match_copy("MESSAGE#25:00002:20/1_2", "nwparser.p0", "");

var dup22 = setc("eventcategory","1502050000");

var dup23 = match("MESSAGE#26:00002:21/1", "nwparser.p0", "%{p0}");

var dup24 = match("MESSAGE#26:00002:21/2_0", "nwparser.p0", "password %{p0}");

var dup25 = match("MESSAGE#26:00002:21/2_1", "nwparser.p0", "name %{p0}");

var dup26 = match_copy("MESSAGE#27:00002:22/1_2", "nwparser.p0", "administrator");

var dup27 = setc("eventcategory","1801010000");

var dup28 = setc("eventcategory","1401060000");

var dup29 = setc("ec_subject","User");

var dup30 = setc("ec_activity","Logon");

var dup31 = setc("ec_theme","Authentication");

var dup32 = setc("ec_outcome","Success");

var dup33 = setc("eventcategory","1401070000");

var dup34 = setc("ec_activity","Logoff");

var dup35 = setc("eventcategory","1303000000");

var dup36 = match_copy("MESSAGE#42:00002:38/1_1", "nwparser.p0", "disposition");

var dup37 = setc("eventcategory","1402020200");

var dup38 = setc("ec_theme","UserGroup");

var dup39 = setc("ec_outcome","Error");

var dup40 = match("MESSAGE#46:00002:42/1_1", "nwparser.p0", "via %{p0}");

var dup41 = match("MESSAGE#46:00002:42/4", "nwparser.p0", "%{fld1})");

var dup42 = setc("eventcategory","1402020300");

var dup43 = setc("ec_activity","Modify");

var dup44 = setc("eventcategory","1605000000");

var dup45 = match("MESSAGE#52:00002:48/3_1", "nwparser.p0", "%{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{p0}");

var dup46 = match("MESSAGE#53:00002:52/3_0", "nwparser.p0", "admin %{administrator->} via %{p0}");

var dup47 = match("MESSAGE#53:00002:52/3_2", "nwparser.p0", "%{username->} via %{p0}");

var dup48 = match("MESSAGE#53:00002:52/4_0", "nwparser.p0", "NSRP Peer . (%{p0}");

var dup49 = match("MESSAGE#55:00002:54/2", "nwparser.p0", ". (%{fld1})");

var dup50 = setc("eventcategory","1701020000");

var dup51 = setc("ec_theme","Configuration");

var dup52 = match("MESSAGE#56:00002/1_1", "nwparser.p0", "changed%{p0}");

var dup53 = setc("eventcategory","1301000000");

var dup54 = setc("ec_outcome","Failure");

var dup55 = match("MESSAGE#61:00003:05/0", "nwparser.payload", "The %{p0}");

var dup56 = match("MESSAGE#66:00004:04/1_0", "nwparser.p0", "interface%{p0}");

var dup57 = match("MESSAGE#66:00004:04/1_1", "nwparser.p0", "Interface%{p0}");

var dup58 = setc("eventcategory","1001000000");

var dup59 = setc("dclass_counter1_string","Number of times the attack occurred");

var dup60 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$OUT"),
		field("saddr"),
		field("daddr"),
	],
});

var dup61 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$OUT"),
		field("saddr"),
		field("daddr"),
		field("sport"),
		field("dport"),
	],
});

var dup62 = setc("eventcategory","1608010000");

var dup63 = match("MESSAGE#76:00004:14/0", "nwparser.payload", "DNS entries have been %{p0}");

var dup64 = match("MESSAGE#79:00004:17/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{p0}");

var dup65 = match("MESSAGE#79:00004:17/1_0", "nwparser.p0", "%{zone}, %{p0}");

var dup66 = match("MESSAGE#79:00004:17/1_1", "nwparser.p0", "%{zone->} %{p0}");

var dup67 = match("MESSAGE#79:00004:17/2", "nwparser.p0", "int %{interface}).%{space}Occurred %{dclass_counter1->} times. (%{fld1})");

var dup68 = match("MESSAGE#83:00005:03/1_0", "nwparser.p0", "%{dport},%{p0}");

var dup69 = match("MESSAGE#83:00005:03/1_1", "nwparser.p0", "%{dport->} %{p0}");

var dup70 = match("MESSAGE#83:00005:03/2", "nwparser.p0", "%{space}using protocol %{p0}");

var dup71 = match("MESSAGE#83:00005:03/3_0", "nwparser.p0", "%{protocol},%{p0}");

var dup72 = match("MESSAGE#83:00005:03/3_1", "nwparser.p0", "%{protocol->} %{p0}");

var dup73 = match("MESSAGE#83:00005:03/5_1", "nwparser.p0", ". %{p0}");

var dup74 = match("MESSAGE#86:00005:06/0_0", "nwparser.payload", "%{fld2}: SYN %{p0}");

var dup75 = match("MESSAGE#86:00005:06/0_1", "nwparser.payload", "SYN %{p0}");

var dup76 = match("MESSAGE#87:00005:07/1_2", "nwparser.p0", "timeout value %{p0}");

var dup77 = match("MESSAGE#88:00005:08/2_0", "nwparser.p0", "destination %{p0}");

var dup78 = match("MESSAGE#88:00005:08/2_1", "nwparser.p0", "source %{p0}");

var dup79 = match("MESSAGE#97:00005:17/0", "nwparser.payload", "A %{p0}");

var dup80 = match("MESSAGE#98:00005:18/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var dup81 = match("MESSAGE#98:00005:18/1_0", "nwparser.p0", ", int %{p0}");

var dup82 = match("MESSAGE#98:00005:18/1_1", "nwparser.p0", "int %{p0}");

var dup83 = match("MESSAGE#98:00005:18/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times. (%{fld1})");

var dup84 = setc("eventcategory","1002020000");

var dup85 = setc("eventcategory","1002000000");

var dup86 = setc("eventcategory","1603110000");

var dup87 = match("MESSAGE#111:00007:04/0", "nwparser.payload", "HA %{p0}");

var dup88 = match("MESSAGE#111:00007:04/1_0", "nwparser.p0", "encryption %{p0}");

var dup89 = match("MESSAGE#111:00007:04/1_1", "nwparser.p0", "authentication %{p0}");

var dup90 = match("MESSAGE#111:00007:04/3_1", "nwparser.p0", "key %{p0}");

var dup91 = setc("eventcategory","1613040200");

var dup92 = match("MESSAGE#118:00007:11/1_0", "nwparser.p0", "disabled%{}");

var dup93 = match("MESSAGE#118:00007:11/1_1", "nwparser.p0", "set to %{trigger_val}");

var dup94 = match("MESSAGE#127:00007:21/1_0", "nwparser.p0", "up%{}");

var dup95 = match("MESSAGE#127:00007:21/1_1", "nwparser.p0", "down%{}");

var dup96 = match("MESSAGE#139:00007:33/2_1", "nwparser.p0", " %{p0}");

var dup97 = setc("eventcategory","1613050200");

var dup98 = match("MESSAGE#143:00007:37/1_0", "nwparser.p0", "set%{}");

var dup99 = match("MESSAGE#143:00007:37/1_1", "nwparser.p0", "unset%{}");

var dup100 = match("MESSAGE#144:00007:38/1_0", "nwparser.p0", "undefined %{p0}");

var dup101 = match("MESSAGE#144:00007:38/1_1", "nwparser.p0", "set %{p0}");

var dup102 = match("MESSAGE#144:00007:38/1_2", "nwparser.p0", "active %{p0}");

var dup103 = match("MESSAGE#144:00007:38/2", "nwparser.p0", "to %{p0}");

var dup104 = match("MESSAGE#157:00007:51/1_0", "nwparser.p0", "created %{p0}");

var dup105 = match("MESSAGE#157:00007:51/3_0", "nwparser.p0", ", %{p0}");

var dup106 = match("MESSAGE#157:00007:51/5_0", "nwparser.p0", "is %{p0}");

var dup107 = match("MESSAGE#157:00007:51/5_1", "nwparser.p0", "was %{p0}");

var dup108 = match("MESSAGE#157:00007:51/6", "nwparser.p0", "%{fld2}");

var dup109 = match("MESSAGE#163:00007:57/1_0", "nwparser.p0", "threshold %{p0}");

var dup110 = match("MESSAGE#163:00007:57/1_1", "nwparser.p0", "interval %{p0}");

var dup111 = match("MESSAGE#163:00007:57/3_0", "nwparser.p0", "of %{p0}");

var dup112 = match("MESSAGE#163:00007:57/3_1", "nwparser.p0", "that %{p0}");

var dup113 = match("MESSAGE#170:00007:64/0_0", "nwparser.payload", "Zone %{p0}");

var dup114 = match("MESSAGE#170:00007:64/0_1", "nwparser.payload", "Interface %{p0}");

var dup115 = match("MESSAGE#172:00007:66/2_1", "nwparser.p0", "n %{p0}");

var dup116 = match("MESSAGE#174:00007:68/4", "nwparser.p0", ".%{}");

var dup117 = setc("eventcategory","1603090000");

var dup118 = match("MESSAGE#195:00009:06/1", "nwparser.p0", "for %{p0}");

var dup119 = match("MESSAGE#195:00009:06/2_0", "nwparser.p0", "the %{p0}");

var dup120 = match("MESSAGE#195:00009:06/4_0", "nwparser.p0", "removed %{p0}");

var dup121 = setc("eventcategory","1603030000");

var dup122 = match("MESSAGE#202:00009:14/2_0", "nwparser.p0", "interface %{p0}");

var dup123 = match("MESSAGE#202:00009:14/2_1", "nwparser.p0", "the interface %{p0}");

var dup124 = match_copy("MESSAGE#202:00009:14/4_1", "nwparser.p0", "interface");

var dup125 = match("MESSAGE#203:00009:15/1_1", "nwparser.p0", "s %{p0}");

var dup126 = match("MESSAGE#203:00009:15/2", "nwparser.p0", "on interface %{interface->} %{p0}");

var dup127 = match("MESSAGE#203:00009:15/3_0", "nwparser.p0", "has been %{p0}");

var dup128 = match("MESSAGE#203:00009:15/4", "nwparser.p0", "%{disposition}.");

var dup129 = match("MESSAGE#204:00009:16/3_0", "nwparser.p0", "removed from %{p0}");

var dup130 = match("MESSAGE#204:00009:16/3_1", "nwparser.p0", "added to %{p0}");

var dup131 = match("MESSAGE#210:00009:21/2", "nwparser.p0", "%{interface}). Occurred %{dclass_counter1->} times. (%{fld1})");

var dup132 = match("MESSAGE#219:00010:03/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone->} %{p0}");

var dup133 = match("MESSAGE#224:00011:04/1_1", "nwparser.p0", "Interface %{p0}");

var dup134 = match("MESSAGE#233:00011:14/1_0", "nwparser.p0", "set to %{fld2}");

var dup135 = match("MESSAGE#237:00011:18/4_1", "nwparser.p0", "gateway %{p0}");

var dup136 = match("MESSAGE#238:00011:19/6", "nwparser.p0", "%{} %{disposition}");

var dup137 = match("MESSAGE#274:00015:02/1_1", "nwparser.p0", "port number %{p0}");

var dup138 = match("MESSAGE#274:00015:02/2", "nwparser.p0", "has been %{disposition}");

var dup139 = match("MESSAGE#276:00015:04/1_0", "nwparser.p0", "IP %{p0}");

var dup140 = match("MESSAGE#276:00015:04/1_1", "nwparser.p0", "port %{p0}");

var dup141 = setc("eventcategory","1702030000");

var dup142 = match("MESSAGE#284:00015:12/3_0", "nwparser.p0", "up %{p0}");

var dup143 = match("MESSAGE#284:00015:12/3_1", "nwparser.p0", "down %{p0}");

var dup144 = setc("eventcategory","1601000000");

var dup145 = match("MESSAGE#294:00015:22/2_0", "nwparser.p0", "(%{fld1}) ");

var dup146 = date_time({
	dest: "event_time",
	args: ["fld2"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup147 = setc("eventcategory","1103000000");

var dup148 = setc("ec_subject","NetworkComm");

var dup149 = setc("ec_activity","Scan");

var dup150 = setc("ec_theme","TEV");

var dup151 = setc("eventcategory","1103010000");

var dup152 = match("MESSAGE#317:00017:01/2_0", "nwparser.p0", ": %{p0}");

var dup153 = match("MESSAGE#320:00017:04/0", "nwparser.payload", "IP %{p0}");

var dup154 = match("MESSAGE#320:00017:04/1_0", "nwparser.p0", "address pool %{p0}");

var dup155 = match("MESSAGE#320:00017:04/1_1", "nwparser.p0", "pool %{p0}");

var dup156 = match("MESSAGE#326:00017:10/1_0", "nwparser.p0", "enabled %{p0}");

var dup157 = match("MESSAGE#326:00017:10/1_1", "nwparser.p0", "disabled %{p0}");

var dup158 = match("MESSAGE#332:00017:15/1_0", "nwparser.p0", "AH %{p0}");

var dup159 = match("MESSAGE#332:00017:15/1_1", "nwparser.p0", "ESP %{p0}");

var dup160 = match("MESSAGE#354:00018:11/0", "nwparser.payload", "%{} %{p0}");

var dup161 = match("MESSAGE#356:00018:32/0_0", "nwparser.payload", "Source%{p0}");

var dup162 = match("MESSAGE#356:00018:32/0_1", "nwparser.payload", "Destination%{p0}");

var dup163 = match("MESSAGE#356:00018:32/2_0", "nwparser.p0", "from %{p0}");

var dup164 = match("MESSAGE#356:00018:32/3", "nwparser.p0", "policy ID %{policy_id->} by admin %{administrator->} via NSRP Peer . (%{fld1})");

var dup165 = match("MESSAGE#375:00019:01/0", "nwparser.payload", "Attempt to enable %{p0}");

var dup166 = match("MESSAGE#375:00019:01/1_0", "nwparser.p0", "traffic logging via syslog %{p0}");

var dup167 = match("MESSAGE#375:00019:01/1_1", "nwparser.p0", "syslog %{p0}");

var dup168 = match("MESSAGE#378:00019:04/0", "nwparser.payload", "Syslog %{p0}");

var dup169 = match("MESSAGE#378:00019:04/1_0", "nwparser.p0", "host %{p0}");

var dup170 = match("MESSAGE#378:00019:04/3_1", "nwparser.p0", "domain name %{p0}");

var dup171 = match("MESSAGE#378:00019:04/4", "nwparser.p0", "has been changed to %{fld2}");

var dup172 = match("MESSAGE#380:00019:06/1_0", "nwparser.p0", "security facility %{p0}");

var dup173 = match("MESSAGE#380:00019:06/1_1", "nwparser.p0", "facility %{p0}");

var dup174 = match("MESSAGE#380:00019:06/3_0", "nwparser.p0", "local0%{}");

var dup175 = match("MESSAGE#380:00019:06/3_1", "nwparser.p0", "local1%{}");

var dup176 = match("MESSAGE#380:00019:06/3_2", "nwparser.p0", "local2%{}");

var dup177 = match("MESSAGE#380:00019:06/3_3", "nwparser.p0", "local3%{}");

var dup178 = match("MESSAGE#380:00019:06/3_4", "nwparser.p0", "local4%{}");

var dup179 = match("MESSAGE#380:00019:06/3_5", "nwparser.p0", "local5%{}");

var dup180 = match("MESSAGE#380:00019:06/3_6", "nwparser.p0", "local6%{}");

var dup181 = match("MESSAGE#380:00019:06/3_7", "nwparser.p0", "local7%{}");

var dup182 = match("MESSAGE#380:00019:06/3_8", "nwparser.p0", "auth/sec%{}");

var dup183 = match("MESSAGE#384:00019:10/0", "nwparser.payload", "%{fld2->} %{p0}");

var dup184 = setc("eventcategory","1603020000");

var dup185 = setc("eventcategory","1803000000");

var dup186 = match("MESSAGE#405:00022/0", "nwparser.payload", "All %{p0}");

var dup187 = setc("eventcategory","1603010000");

var dup188 = setc("eventcategory","1603100000");

var dup189 = match("MESSAGE#414:00022:09/1_0", "nwparser.p0", "primary %{p0}");

var dup190 = match("MESSAGE#414:00022:09/1_1", "nwparser.p0", "secondary %{p0}");

var dup191 = match("MESSAGE#414:00022:09/3_0", "nwparser.p0", "t %{p0}");

var dup192 = match("MESSAGE#414:00022:09/3_1", "nwparser.p0", "w %{p0}");

var dup193 = match("MESSAGE#423:00024/1", "nwparser.p0", "server %{p0}");

var dup194 = match("MESSAGE#426:00024:03/1_0", "nwparser.p0", "has %{p0}");

var dup195 = match("MESSAGE#434:00026:01/0", "nwparser.payload", "SCS%{p0}");

var dup196 = match("MESSAGE#434:00026:01/3_0", "nwparser.p0", "bound to %{p0}");

var dup197 = match("MESSAGE#434:00026:01/3_1", "nwparser.p0", "unbound from %{p0}");

var dup198 = setc("eventcategory","1801030000");

var dup199 = setc("eventcategory","1302010200");

var dup200 = match("MESSAGE#441:00026:08/1_1", "nwparser.p0", "PKA RSA %{p0}");

var dup201 = match("MESSAGE#443:00026:10/3_1", "nwparser.p0", "unbind %{p0}");

var dup202 = match("MESSAGE#443:00026:10/4", "nwparser.p0", "PKA key %{p0}");

var dup203 = setc("eventcategory","1304000000");

var dup204 = match("MESSAGE#446:00027/0", "nwparser.payload", "Multiple login failures %{p0}");

var dup205 = match("MESSAGE#446:00027/1_0", "nwparser.p0", "occurred for %{p0}");

var dup206 = setc("eventcategory","1401030000");

var dup207 = match("MESSAGE#451:00027:05/5_0", "nwparser.p0", "aborted%{}");

var dup208 = match("MESSAGE#451:00027:05/5_1", "nwparser.p0", "performed%{}");

var dup209 = setc("eventcategory","1605020000");

var dup210 = match("MESSAGE#466:00029:03/0", "nwparser.payload", "IP pool of DHCP server on %{p0}");

var dup211 = setc("ec_subject","Certificate");

var dup212 = match("MESSAGE#492:00030:17/1_0", "nwparser.p0", "certificate %{p0}");

var dup213 = match("MESSAGE#492:00030:17/1_1", "nwparser.p0", "CRL %{p0}");

var dup214 = match("MESSAGE#493:00030:40/1_0", "nwparser.p0", "auto %{p0}");

var dup215 = match("MESSAGE#508:00030:55/1_0", "nwparser.p0", "RSA %{p0}");

var dup216 = match("MESSAGE#508:00030:55/1_1", "nwparser.p0", "DSA %{p0}");

var dup217 = match("MESSAGE#508:00030:55/2", "nwparser.p0", "key pair.%{}");

var dup218 = setc("ec_subject","CryptoKey");

var dup219 = setc("ec_subject","Configuration");

var dup220 = setc("ec_activity","Request");

var dup221 = match("MESSAGE#539:00030:86/0", "nwparser.payload", "FIPS test for %{p0}");

var dup222 = match("MESSAGE#539:00030:86/1_0", "nwparser.p0", "ECDSA %{p0}");

var dup223 = setc("eventcategory","1612000000");

var dup224 = match("MESSAGE#543:00031:02/1_0", "nwparser.p0", "yes %{p0}");

var dup225 = match("MESSAGE#543:00031:02/1_1", "nwparser.p0", "no %{p0}");

var dup226 = match("MESSAGE#545:00031:04/1_1", "nwparser.p0", "location %{p0}");

var dup227 = match("MESSAGE#548:00031:05/2", "nwparser.p0", "%{} %{interface}");

var dup228 = match("MESSAGE#549:00031:06/0", "nwparser.payload", "arp re%{p0}");

var dup229 = match("MESSAGE#549:00031:06/1_1", "nwparser.p0", "q %{p0}");

var dup230 = match("MESSAGE#549:00031:06/1_2", "nwparser.p0", "ply %{p0}");

var dup231 = match("MESSAGE#549:00031:06/9_0", "nwparser.p0", "%{interface->} (%{fld1})");

var dup232 = setc("eventcategory","1201000000");

var dup233 = match("MESSAGE#561:00033/0_0", "nwparser.payload", "Global PRO %{p0}");

var dup234 = match("MESSAGE#561:00033/0_1", "nwparser.payload", "%{fld3->} %{p0}");

var dup235 = match("MESSAGE#569:00033:08/0", "nwparser.payload", "NACN Policy Manager %{p0}");

var dup236 = match("MESSAGE#569:00033:08/1_0", "nwparser.p0", "1 %{p0}");

var dup237 = match("MESSAGE#569:00033:08/1_1", "nwparser.p0", "2 %{p0}");

var dup238 = match("MESSAGE#571:00033:10/3_1", "nwparser.p0", "unset %{p0}");

var dup239 = match("MESSAGE#581:00033:21/0", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var dup240 = setc("eventcategory","1401000000");

var dup241 = match("MESSAGE#586:00034:01/2_1", "nwparser.p0", "SSH %{p0}");

var dup242 = match("MESSAGE#588:00034:03/0_0", "nwparser.payload", "SCS: NetScreen %{p0}");

var dup243 = match("MESSAGE#588:00034:03/0_1", "nwparser.payload", "NetScreen %{p0}");

var dup244 = match("MESSAGE#595:00034:10/0", "nwparser.payload", "S%{p0}");

var dup245 = match("MESSAGE#595:00034:10/1_0", "nwparser.p0", "CS: SSH%{p0}");

var dup246 = match("MESSAGE#595:00034:10/1_1", "nwparser.p0", "SH%{p0}");

var dup247 = match("MESSAGE#596:00034:12/3_0", "nwparser.p0", "the root system %{p0}");

var dup248 = match("MESSAGE#596:00034:12/3_1", "nwparser.p0", "vsys %{fld2->} %{p0}");

var dup249 = match("MESSAGE#599:00034:18/1_0", "nwparser.p0", "CS: SSH %{p0}");

var dup250 = match("MESSAGE#599:00034:18/1_1", "nwparser.p0", "SH %{p0}");

var dup251 = match("MESSAGE#630:00035:06/1_0", "nwparser.p0", "a %{p0}");

var dup252 = match("MESSAGE#630:00035:06/1_1", "nwparser.p0", "ert %{p0}");

var dup253 = match("MESSAGE#633:00035:09/0", "nwparser.payload", "SSL %{p0}");

var dup254 = setc("eventcategory","1608000000");

var dup255 = match("MESSAGE#644:00037:01/1_0", "nwparser.p0", "id: %{p0}");

var dup256 = match("MESSAGE#644:00037:01/1_1", "nwparser.p0", "ID %{p0}");

var dup257 = match("MESSAGE#659:00044/1_0", "nwparser.p0", "permit %{p0}");

var dup258 = match("MESSAGE#675:00055/0", "nwparser.payload", "IGMP %{p0}");

var dup259 = match("MESSAGE#677:00055:02/0", "nwparser.payload", "IGMP will %{p0}");

var dup260 = match("MESSAGE#677:00055:02/1_0", "nwparser.p0", "not do %{p0}");

var dup261 = match("MESSAGE#677:00055:02/1_1", "nwparser.p0", "do %{p0}");

var dup262 = match("MESSAGE#689:00059/1_1", "nwparser.p0", "shut down %{p0}");

var dup263 = match("MESSAGE#707:00070/0", "nwparser.payload", "NSRP: %{p0}");

var dup264 = match("MESSAGE#707:00070/1_0", "nwparser.p0", "Unit %{p0}");

var dup265 = match("MESSAGE#707:00070/1_1", "nwparser.p0", "local unit= %{p0}");

var dup266 = match("MESSAGE#707:00070/2", "nwparser.p0", "%{fld2->} of VSD group %{group->} %{info}");

var dup267 = match("MESSAGE#708:00070:01/0", "nwparser.payload", "The local device %{fld2->} in the Virtual Sec%{p0}");

var dup268 = match("MESSAGE#708:00070:01/1_0", "nwparser.p0", "ruity%{p0}");

var dup269 = match("MESSAGE#708:00070:01/1_1", "nwparser.p0", "urity%{p0}");

var dup270 = match("MESSAGE#713:00072:01/2", "nwparser.p0", "%{}Device group %{group->} changed state");

var dup271 = match("MESSAGE#717:00075/2", "nwparser.p0", "%{fld2->} of VSD group %{group->} %{info}");

var dup272 = setc("eventcategory","1805010000");

var dup273 = setc("eventcategory","1805000000");

var dup274 = date_time({
	dest: "starttime",
	args: ["fld2"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup275 = call({
	dest: "nwparser.bytes",
	fn: CALC,
	args: [
		field("sbytes"),
		constant("+"),
		field("rbytes"),
	],
});

var dup276 = setc("action","Deny");

var dup277 = setc("disposition","Deny");

var dup278 = setc("direction","outgoing");

var dup279 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$IN"),
		field("saddr"),
		field("daddr"),
		field("sport"),
		field("dport"),
	],
});

var dup280 = setc("direction","incoming");

var dup281 = setc("eventcategory","1801000000");

var dup282 = setf("action","disposition");

var dup283 = match("MESSAGE#748:00257:19/0", "nwparser.payload", "start_time=%{p0}");

var dup284 = match("MESSAGE#748:00257:19/1_0", "nwparser.p0", "\\\"%{fld2}\\\"%{p0}");

var dup285 = match("MESSAGE#748:00257:19/1_1", "nwparser.p0", " \"%{fld2}\" %{p0}");

var dup286 = match_copy("MESSAGE#756:00257:10/1_1", "nwparser.p0", "daddr");

var dup287 = match("MESSAGE#760:00259/0_0", "nwparser.payload", "Admin %{p0}");

var dup288 = match("MESSAGE#760:00259/0_1", "nwparser.payload", "Vsys admin %{p0}");

var dup289 = match("MESSAGE#760:00259/2_1", "nwparser.p0", "Telnet %{p0}");

var dup290 = setc("eventcategory","1401050200");

var dup291 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$IN"),
		field("daddr"),
		field("saddr"),
	],
});

var dup292 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$IN"),
		field("daddr"),
		field("saddr"),
		field("dport"),
		field("sport"),
	],
});

var dup293 = match("MESSAGE#777:00406/2", "nwparser.p0", "%{interface}). Occurred %{dclass_counter1->} times.");

var dup294 = match("MESSAGE#790:00423/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times.");

var dup295 = match("MESSAGE#793:00430/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times.%{p0}");

var dup296 = match("MESSAGE#795:00431/0", "nwparser.payload", "%{obj_type->} %{disposition}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var dup297 = setc("eventcategory","1204000000");

var dup298 = match("MESSAGE#797:00433/0", "nwparser.payload", "%{signame->} %{disposition}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var dup299 = match("MESSAGE#804:00437:01/0", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{p0}");

var dup300 = match("MESSAGE#817:00511:01/1_0", "nwparser.p0", "%{administrator->} (%{fld1})");

var dup301 = setc("eventcategory","1801020000");

var dup302 = setc("disposition","failed");

var dup303 = match("MESSAGE#835:00515:04/2_1", "nwparser.p0", "ut %{p0}");

var dup304 = match("MESSAGE#835:00515:04/4_0", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport}");

var dup305 = match("MESSAGE#837:00515:05/1_0", "nwparser.p0", "user %{p0}");

var dup306 = match("MESSAGE#837:00515:05/5_0", "nwparser.p0", "the %{logon_type}");

var dup307 = match("MESSAGE#869:00519:01/1_0", "nwparser.p0", "WebAuth user %{p0}");

var dup308 = match("MESSAGE#876:00520:02/1_1", "nwparser.p0", "backup1 %{p0}");

var dup309 = match("MESSAGE#876:00520:02/1_2", "nwparser.p0", "backup2 %{p0}");

var dup310 = match("MESSAGE#890:00524:13/1_0", "nwparser.p0", ",%{p0}");

var dup311 = match("MESSAGE#901:00527/1_0", "nwparser.p0", "assigned %{p0}");

var dup312 = match("MESSAGE#901:00527/3_0", "nwparser.p0", "assigned to %{p0}");

var dup313 = setc("eventcategory","1803020000");

var dup314 = setc("eventcategory","1613030000");

var dup315 = match("MESSAGE#927:00528:15/1_0", "nwparser.p0", "'%{administrator}' %{p0}");

var dup316 = match("MESSAGE#930:00528:18/0", "nwparser.payload", "SSH: P%{p0}");

var dup317 = match("MESSAGE#930:00528:18/1_0", "nwparser.p0", "KA %{p0}");

var dup318 = match("MESSAGE#930:00528:18/1_1", "nwparser.p0", "assword %{p0}");

var dup319 = match("MESSAGE#930:00528:18/3_0", "nwparser.p0", "\\'%{administrator}\\' %{p0}");

var dup320 = match("MESSAGE#930:00528:18/4", "nwparser.p0", "at host %{saddr}");

var dup321 = match("MESSAGE#932:00528:19/0", "nwparser.payload", "%{}S%{p0}");

var dup322 = match("MESSAGE#932:00528:19/1_0", "nwparser.p0", "CS %{p0}");

var dup323 = setc("event_description","Cannot connect to NSM server");

var dup324 = setc("eventcategory","1603040000");

var dup325 = match("MESSAGE#1060:00553/2", "nwparser.p0", "from server.ini file.%{}");

var dup326 = match("MESSAGE#1064:00553:04/1_0", "nwparser.p0", "pattern %{p0}");

var dup327 = match("MESSAGE#1064:00553:04/1_1", "nwparser.p0", "server.ini %{p0}");

var dup328 = match("MESSAGE#1068:00553:08/2", "nwparser.p0", "file.%{}");

var dup329 = match("MESSAGE#1087:00554:04/1_1", "nwparser.p0", "AV pattern %{p0}");

var dup330 = match("MESSAGE#1116:00556:14/1_0", "nwparser.p0", "added into %{p0}");

var dup331 = match("MESSAGE#1157:00767:11/1_0", "nwparser.p0", "loader %{p0}");

var dup332 = call({
	dest: "nwparser.inout",
	fn: DIRCHK,
	args: [
		field("$OUT"),
		field("daddr"),
		field("saddr"),
		field("dport"),
		field("sport"),
	],
});

var dup333 = linear_select([
	dup10,
	dup11,
]);

var dup334 = match("MESSAGE#7:00001:07", "nwparser.payload", "Policy ID=%{policy_id->} Rate=%{fld2->} exceeds threshold", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var dup335 = linear_select([
	dup13,
	dup14,
]);

var dup336 = linear_select([
	dup15,
	dup16,
]);

var dup337 = linear_select([
	dup56,
	dup57,
]);

var dup338 = linear_select([
	dup65,
	dup66,
]);

var dup339 = linear_select([
	dup68,
	dup69,
]);

var dup340 = linear_select([
	dup71,
	dup72,
]);

var dup341 = match("MESSAGE#84:00005:04", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol->} (%{interface})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var dup342 = linear_select([
	dup74,
	dup75,
]);

var dup343 = linear_select([
	dup81,
	dup82,
]);

var dup344 = linear_select([
	dup24,
	dup90,
]);

var dup345 = linear_select([
	dup94,
	dup95,
]);

var dup346 = linear_select([
	dup98,
	dup99,
]);

var dup347 = linear_select([
	dup100,
	dup101,
	dup102,
]);

var dup348 = linear_select([
	dup113,
	dup114,
]);

var dup349 = linear_select([
	dup111,
	dup16,
]);

var dup350 = linear_select([
	dup127,
	dup107,
]);

var dup351 = linear_select([
	dup8,
	dup21,
]);

var dup352 = linear_select([
	dup122,
	dup133,
]);

var dup353 = linear_select([
	dup142,
	dup143,
]);

var dup354 = linear_select([
	dup145,
	dup21,
]);

var dup355 = linear_select([
	dup127,
	dup106,
]);

var dup356 = linear_select([
	dup152,
	dup96,
]);

var dup357 = linear_select([
	dup154,
	dup155,
]);

var dup358 = linear_select([
	dup156,
	dup157,
]);

var dup359 = linear_select([
	dup99,
	dup134,
]);

var dup360 = linear_select([
	dup158,
	dup159,
]);

var dup361 = linear_select([
	dup161,
	dup162,
]);

var dup362 = linear_select([
	dup163,
	dup103,
]);

var dup363 = linear_select([
	dup162,
	dup161,
]);

var dup364 = linear_select([
	dup46,
	dup47,
]);

var dup365 = linear_select([
	dup166,
	dup167,
]);

var dup366 = linear_select([
	dup172,
	dup173,
]);

var dup367 = linear_select([
	dup174,
	dup175,
	dup176,
	dup177,
	dup178,
	dup179,
	dup180,
	dup181,
	dup182,
]);

var dup368 = linear_select([
	dup49,
	dup21,
]);

var dup369 = linear_select([
	dup189,
	dup190,
]);

var dup370 = linear_select([
	dup96,
	dup152,
]);

var dup371 = linear_select([
	dup196,
	dup197,
]);

var dup372 = linear_select([
	dup24,
	dup200,
]);

var dup373 = linear_select([
	dup103,
	dup163,
]);

var dup374 = linear_select([
	dup205,
	dup118,
]);

var dup375 = match("MESSAGE#477:00030:02", "nwparser.payload", "%{change_attribute->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var dup376 = linear_select([
	dup212,
	dup213,
]);

var dup377 = linear_select([
	dup215,
	dup216,
]);

var dup378 = linear_select([
	dup222,
	dup215,
]);

var dup379 = linear_select([
	dup224,
	dup225,
]);

var dup380 = linear_select([
	dup231,
	dup124,
]);

var dup381 = linear_select([
	dup229,
	dup230,
]);

var dup382 = linear_select([
	dup233,
	dup234,
]);

var dup383 = linear_select([
	dup236,
	dup237,
]);

var dup384 = linear_select([
	dup242,
	dup243,
]);

var dup385 = linear_select([
	dup245,
	dup246,
]);

var dup386 = linear_select([
	dup247,
	dup248,
]);

var dup387 = linear_select([
	dup249,
	dup250,
]);

var dup388 = linear_select([
	dup251,
	dup252,
]);

var dup389 = linear_select([
	dup260,
	dup261,
]);

var dup390 = linear_select([
	dup264,
	dup265,
]);

var dup391 = linear_select([
	dup268,
	dup269,
]);

var dup392 = match("MESSAGE#716:00074", "nwparser.payload", "The local device %{fld2->} in the Virtual Security Device group %{group->} %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var dup393 = linear_select([
	dup284,
	dup285,
]);

var dup394 = linear_select([
	dup287,
	dup288,
]);

var dup395 = match("MESSAGE#799:00435", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup59,
	dup4,
	dup5,
	dup3,
	dup60,
]));

var dup396 = match("MESSAGE#814:00442", "nwparser.payload", "%{signame->} From %{saddr->} to zone %{zone}, proto %{protocol->} (int %{interface}). Occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup4,
	dup59,
	dup5,
	dup9,
	dup2,
	dup3,
	dup60,
]));

var dup397 = linear_select([
	dup300,
	dup26,
]);

var dup398 = linear_select([
	dup115,
	dup303,
]);

var dup399 = linear_select([
	dup125,
	dup96,
]);

var dup400 = linear_select([
	dup189,
	dup308,
	dup309,
]);

var dup401 = linear_select([
	dup310,
	dup16,
]);

var dup402 = linear_select([
	dup317,
	dup318,
]);

var dup403 = linear_select([
	dup319,
	dup315,
]);

var dup404 = linear_select([
	dup322,
	dup250,
]);

var dup405 = linear_select([
	dup327,
	dup329,
]);

var dup406 = linear_select([
	dup330,
	dup129,
]);

var dup407 = match("MESSAGE#1196:01269:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup60,
	dup282,
]));

var dup408 = match("MESSAGE#1197:01269:02", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup277,
	dup60,
]));

var dup409 = match("MESSAGE#1198:01269:03", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup60,
	dup282,
]));

var dup410 = match("MESSAGE#1203:23184", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup277,
	dup61,
]));

var dup411 = all_match({
	processors: [
		dup263,
		dup390,
		dup266,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var dup412 = all_match({
	processors: [
		dup267,
		dup391,
		dup270,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var dup413 = all_match({
	processors: [
		dup80,
		dup343,
		dup293,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var dup414 = all_match({
	processors: [
		dup296,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup297,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup61,
	]),
});

var dup415 = all_match({
	processors: [
		dup298,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup297,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup61,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "%{hfld1}: NetScreen device_id=%{hfld2->} [No Name]system-%{hseverity}-%{messageid}(%{hfld3}): %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = match("HEADER#1:0003", "message", "%{hfld1}: NetScreen device_id=%{hfld2->} [%{hvsys}]system-%{hseverity}-%{messageid}(%{hfld3}): %{payload}", processor_chain([
	setc("header_id","0003"),
]));

var hdr3 = match("HEADER#2:0004", "message", "%{hfld1}: NetScreen device_id=%{hfld2->} system-%{hseverity}-%{messageid}(%{hfld3}): %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var hdr4 = match("HEADER#3:0002/0", "message", "%{hfld1}: NetScreen device_id=%{hfld2->} %{p0}");

var part1 = match("HEADER#3:0002/1_0", "nwparser.p0", "[No Name]system%{p0}");

var part2 = match("HEADER#3:0002/1_1", "nwparser.p0", "[%{hvsys}]system%{p0}");

var part3 = match("HEADER#3:0002/1_2", "nwparser.p0", "system%{p0}");

var select1 = linear_select([
	part1,
	part2,
	part3,
]);

var part4 = match("HEADER#3:0002/2", "nwparser.p0", "-%{hseverity}-%{messageid}: %{payload}");

var all1 = all_match({
	processors: [
		hdr4,
		select1,
		part4,
	],
	on_success: processor_chain([
		setc("header_id","0002"),
	]),
});

var select2 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	all1,
]);

var part5 = match("MESSAGE#0:00001", "nwparser.payload", "%{zone->} address %{interface->} with ip address %{hostip->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1 = msg("00001", part5);

var part6 = match("MESSAGE#1:00001:01", "nwparser.payload", "%{zone->} address %{interface->} with domain name %{domain->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg2 = msg("00001:01", part6);

var part7 = match("MESSAGE#2:00001:02/1_0", "nwparser.p0", "ip address %{hostip->} in zone %{p0}");

var select3 = linear_select([
	part7,
	dup7,
]);

var part8 = match("MESSAGE#2:00001:02/2", "nwparser.p0", "%{zone->} has been %{disposition}");

var all2 = all_match({
	processors: [
		dup6,
		select3,
		part8,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg3 = msg("00001:02", all2);

var part9 = match("MESSAGE#3:00001:03", "nwparser.payload", "arp entry %{hostip->} interface changed!", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg4 = msg("00001:03", part9);

var part10 = match("MESSAGE#4:00001:04/1_0", "nwparser.p0", "IP address %{hostip->} in zone %{p0}");

var select4 = linear_select([
	part10,
	dup7,
]);

var part11 = match("MESSAGE#4:00001:04/2", "nwparser.p0", "%{zone->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} session%{p0}");

var part12 = match("MESSAGE#4:00001:04/3_1", "nwparser.p0", ".%{fld1}");

var select5 = linear_select([
	dup8,
	part12,
]);

var all3 = all_match({
	processors: [
		dup6,
		select4,
		part11,
		select5,
	],
	on_success: processor_chain([
		dup1,
		dup9,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg5 = msg("00001:04", all3);

var part13 = match("MESSAGE#5:00001:05/0", "nwparser.payload", "%{fld2}: Address %{group_object->} for ip address %{hostip->} in zone %{zone->} has been %{disposition->} from host %{saddr->} session %{p0}");

var all4 = all_match({
	processors: [
		part13,
		dup333,
	],
	on_success: processor_chain([
		dup1,
		dup9,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg6 = msg("00001:05", all4);

var part14 = match("MESSAGE#6:00001:06", "nwparser.payload", "Address group %{group_object->} %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg7 = msg("00001:06", part14);

var msg8 = msg("00001:07", dup334);

var part15 = match("MESSAGE#8:00001:08/2", "nwparser.p0", "for IP address %{hostip}/%{mask->} in zone %{zone->} has been %{disposition->} by %{p0}");

var part16 = match("MESSAGE#8:00001:08/4", "nwparser.p0", "%{} %{username}via NSRP Peer session. (%{fld1})");

var all5 = all_match({
	processors: [
		dup12,
		dup335,
		part15,
		dup336,
		part16,
	],
	on_success: processor_chain([
		dup1,
		dup9,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg9 = msg("00001:08", all5);

var part17 = match("MESSAGE#9:00001:09/2", "nwparser.p0", "for IP address %{hostip}/%{mask->} in zone %{zone->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr}:%{sport->} session. (%{fld1})");

var all6 = all_match({
	processors: [
		dup12,
		dup335,
		part17,
	],
	on_success: processor_chain([
		dup1,
		dup9,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg10 = msg("00001:09", all6);

var select6 = linear_select([
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
]);

var part18 = match("MESSAGE#10:00002:03", "nwparser.payload", "Admin user %{administrator->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg11 = msg("00002:03", part18);

var part19 = match("MESSAGE#11:00002:04", "nwparser.payload", "E-mail address %{user_address->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg12 = msg("00002:04", part19);

var part20 = match("MESSAGE#12:00002:05", "nwparser.payload", "E-mail notification has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg13 = msg("00002:05", part20);

var part21 = match("MESSAGE#13:00002:06", "nwparser.payload", "Inclusion of traffic logs with e-mail notification of event alarms has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg14 = msg("00002:06", part21);

var part22 = match("MESSAGE#14:00002:07", "nwparser.payload", "LCD display has been %{action->} and the LCD control keys have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg15 = msg("00002:07", part22);

var part23 = match("MESSAGE#15:00002:55", "nwparser.payload", "HTTP component blocking for %{fld2->} is %{disposition->} on zone %{zone->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg16 = msg("00002:55", part23);

var part24 = match("MESSAGE#16:00002:08", "nwparser.payload", "LCD display has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg17 = msg("00002:08", part24);

var part25 = match("MESSAGE#17:00002:09", "nwparser.payload", "LCD control keys have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg18 = msg("00002:09", part25);

var part26 = match("MESSAGE#18:00002:10", "nwparser.payload", "Mail server %{hostip->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg19 = msg("00002:10", part26);

var part27 = match("MESSAGE#19:00002:11", "nwparser.payload", "Management restriction for %{hostip->} %{fld2->} has been %{disposition}", processor_chain([
	dup17,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg20 = msg("00002:11", part27);

var part28 = match("MESSAGE#20:00002:12", "nwparser.payload", "%{change_attribute->} has been restored from %{change_old->} to default port %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg21 = msg("00002:12", part28);

var part29 = match("MESSAGE#21:00002:15", "nwparser.payload", "System configuration has been %{disposition}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg22 = msg("00002:15", part29);

var msg23 = msg("00002:17", dup334);

var part30 = match("MESSAGE#23:00002:18/0", "nwparser.payload", "Unexpected error from e%{p0}");

var part31 = match("MESSAGE#23:00002:18/1_0", "nwparser.p0", "-mail %{p0}");

var part32 = match("MESSAGE#23:00002:18/1_1", "nwparser.p0", "mail %{p0}");

var select7 = linear_select([
	part31,
	part32,
]);

var part33 = match("MESSAGE#23:00002:18/2", "nwparser.p0", "server(%{fld2}):");

var all7 = all_match({
	processors: [
		part30,
		select7,
		part33,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg24 = msg("00002:18", all7);

var part34 = match("MESSAGE#24:00002:19", "nwparser.payload", "Web Admin %{change_attribute->} value has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg25 = msg("00002:19", part34);

var part35 = match("MESSAGE#25:00002:20/0", "nwparser.payload", "Root admin password restriction of minimum %{fld2->} characters has been %{disposition->} by admin %{administrator->} %{p0}");

var part36 = match("MESSAGE#25:00002:20/1_0", "nwparser.p0", "from Console %{}");

var select8 = linear_select([
	part36,
	dup20,
	dup21,
]);

var all8 = all_match({
	processors: [
		part35,
		select8,
	],
	on_success: processor_chain([
		dup22,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg26 = msg("00002:20", all8);

var part37 = match("MESSAGE#26:00002:21/0_0", "nwparser.payload", "Root admin %{p0}");

var part38 = match("MESSAGE#26:00002:21/0_1", "nwparser.payload", "%{fld2->} admin %{p0}");

var select9 = linear_select([
	part37,
	part38,
]);

var select10 = linear_select([
	dup24,
	dup25,
]);

var part39 = match("MESSAGE#26:00002:21/3", "nwparser.p0", "has been changed by admin %{administrator}");

var all9 = all_match({
	processors: [
		select9,
		dup23,
		select10,
		part39,
	],
	on_success: processor_chain([
		dup22,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg27 = msg("00002:21", all9);

var part40 = match("MESSAGE#27:00002:22/0", "nwparser.payload", "%{change_attribute->} from %{protocol->} before administrative session disconnects has been changed from %{change_old->} to %{change_new->} by admin %{p0}");

var part41 = match("MESSAGE#27:00002:22/1_0", "nwparser.p0", "%{administrator->} from Console");

var part42 = match("MESSAGE#27:00002:22/1_1", "nwparser.p0", "%{administrator->} from host %{saddr}");

var select11 = linear_select([
	part41,
	part42,
	dup26,
]);

var all10 = all_match({
	processors: [
		part40,
		select11,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg28 = msg("00002:22", all10);

var part43 = match("MESSAGE#28:00002:23/0", "nwparser.payload", "Root admin access restriction through console only has been %{disposition->} by admin %{administrator->} %{p0}");

var part44 = match("MESSAGE#28:00002:23/1_1", "nwparser.p0", "from Console%{}");

var select12 = linear_select([
	dup20,
	part44,
	dup21,
]);

var all11 = all_match({
	processors: [
		part43,
		select12,
	],
	on_success: processor_chain([
		dup22,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg29 = msg("00002:23", all11);

var part45 = match("MESSAGE#29:00002:24/0", "nwparser.payload", "Admin access restriction of %{protocol->} administration through tunnel only has been %{disposition->} by admin %{administrator->} from %{p0}");

var part46 = match("MESSAGE#29:00002:24/1_0", "nwparser.p0", "host %{saddr}");

var part47 = match("MESSAGE#29:00002:24/1_1", "nwparser.p0", "Console%{}");

var select13 = linear_select([
	part46,
	part47,
]);

var all12 = all_match({
	processors: [
		part45,
		select13,
	],
	on_success: processor_chain([
		dup22,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg30 = msg("00002:24", all12);

var part48 = match("MESSAGE#30:00002:25", "nwparser.payload", "Admin AUTH: Local instance of an %{change_attribute->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	setc("eventcategory","1402000000"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg31 = msg("00002:25", part48);

var part49 = match("MESSAGE#31:00002:26", "nwparser.payload", "Cannot connect to e-mail server %{hostip}.", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg32 = msg("00002:26", part49);

var part50 = match("MESSAGE#32:00002:27", "nwparser.payload", "Mail server is not configured.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg33 = msg("00002:27", part50);

var part51 = match("MESSAGE#33:00002:28", "nwparser.payload", "Mail recipients were not configured.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg34 = msg("00002:28", part51);

var part52 = match("MESSAGE#34:00002:29", "nwparser.payload", "Single use password restriction for read-write administrators has been %{disposition->} by admin %{administrator}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg35 = msg("00002:29", part52);

var part53 = match("MESSAGE#35:00002:30", "nwparser.payload", "Admin user \"%{administrator}\" logged in for %{logon_type}(%{network_service}) management (port %{network_port}) from %{saddr}:%{sport}", processor_chain([
	dup28,
	dup29,
	dup30,
	dup31,
	dup32,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg36 = msg("00002:30", part53);

var part54 = match("MESSAGE#36:00002:41", "nwparser.payload", "Admin user \"%{administrator}\" logged out for %{logon_type}(%{network_service}) management (port %{network_port}) from %{saddr}:%{sport}", processor_chain([
	dup33,
	dup29,
	dup34,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg37 = msg("00002:41", part54);

var part55 = match("MESSAGE#37:00002:31", "nwparser.payload", "Admin user \"%{administrator}\" login attempt for %{logon_type->} %{space->} (%{network_service}) management (port %{network_port}) from %{saddr}:%{sport->} %{disposition}", processor_chain([
	dup35,
	dup29,
	dup30,
	dup31,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg38 = msg("00002:31", part55);

var part56 = match("MESSAGE#38:00002:32/0_0", "nwparser.payload", "E-mail notification %{p0}");

var part57 = match("MESSAGE#38:00002:32/0_1", "nwparser.payload", "Transparent virutal %{p0}");

var select14 = linear_select([
	part56,
	part57,
]);

var part58 = match("MESSAGE#38:00002:32/1", "nwparser.p0", "wire mode has been %{disposition}");

var all13 = all_match({
	processors: [
		select14,
		part58,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg39 = msg("00002:32", all13);

var part59 = match("MESSAGE#39:00002:35", "nwparser.payload", "Malicious URL %{url->} has been %{disposition->} for zone %{zone}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg40 = msg("00002:35", part59);

var part60 = match("MESSAGE#40:00002:36/0", "nwparser.payload", "Bypass%{p0}");

var part61 = match("MESSAGE#40:00002:36/1_0", "nwparser.p0", "-others-IPSec %{p0}");

var part62 = match("MESSAGE#40:00002:36/1_1", "nwparser.p0", " non-IP traffic %{p0}");

var select15 = linear_select([
	part61,
	part62,
]);

var part63 = match("MESSAGE#40:00002:36/2", "nwparser.p0", "option has been %{disposition}");

var all14 = all_match({
	processors: [
		part60,
		select15,
		part63,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg41 = msg("00002:36", all14);

var part64 = match("MESSAGE#41:00002:37/0", "nwparser.payload", "Logging of %{p0}");

var part65 = match("MESSAGE#41:00002:37/1_0", "nwparser.p0", "dropped %{p0}");

var part66 = match("MESSAGE#41:00002:37/1_1", "nwparser.p0", "IKE %{p0}");

var part67 = match("MESSAGE#41:00002:37/1_2", "nwparser.p0", "SNMP %{p0}");

var part68 = match("MESSAGE#41:00002:37/1_3", "nwparser.p0", "ICMP %{p0}");

var select16 = linear_select([
	part65,
	part66,
	part67,
	part68,
]);

var part69 = match("MESSAGE#41:00002:37/2", "nwparser.p0", "traffic to self has been %{disposition}");

var all15 = all_match({
	processors: [
		part64,
		select16,
		part69,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg42 = msg("00002:37", all15);

var part70 = match("MESSAGE#42:00002:38/0", "nwparser.payload", "Logging of dropped traffic to self (excluding multicast) has been %{p0}");

var part71 = match("MESSAGE#42:00002:38/1_0", "nwparser.p0", "%{disposition->} on %{zone}");

var select17 = linear_select([
	part71,
	dup36,
]);

var all16 = all_match({
	processors: [
		part70,
		select17,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg43 = msg("00002:38", all16);

var part72 = match("MESSAGE#43:00002:39", "nwparser.payload", "Traffic shaping is %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg44 = msg("00002:39", part72);

var part73 = match("MESSAGE#44:00002:40", "nwparser.payload", "Admin account created for '%{username}' by %{administrator->} via %{logon_type->} from host %{saddr->} (%{fld1})", processor_chain([
	dup37,
	dup29,
	setc("ec_activity","Create"),
	dup38,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg45 = msg("00002:40", part73);

var part74 = match("MESSAGE#45:00002:44", "nwparser.payload", "ADMIN AUTH: Privilege requested for unknown user %{username}. Possible HA syncronization problem.", processor_chain([
	dup35,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg46 = msg("00002:44", part74);

var part75 = match("MESSAGE#46:00002:42/0", "nwparser.payload", "%{change_attribute->} for account '%{change_old}' has been %{disposition->} to '%{change_new}' %{p0}");

var part76 = match("MESSAGE#46:00002:42/1_0", "nwparser.p0", "by %{administrator->} via %{p0}");

var select18 = linear_select([
	part76,
	dup40,
]);

var part77 = match("MESSAGE#46:00002:42/2", "nwparser.p0", "%{logon_type->} from host %{p0}");

var part78 = match("MESSAGE#46:00002:42/3_0", "nwparser.p0", "%{saddr->} to %{daddr}:%{dport->} (%{p0}");

var part79 = match("MESSAGE#46:00002:42/3_1", "nwparser.p0", "%{saddr}:%{sport->} (%{p0}");

var select19 = linear_select([
	part78,
	part79,
]);

var all17 = all_match({
	processors: [
		part75,
		select18,
		part77,
		select19,
		dup41,
	],
	on_success: processor_chain([
		dup42,
		dup43,
		dup38,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg47 = msg("00002:42", all17);

var part80 = match("MESSAGE#47:00002:43/0", "nwparser.payload", "Admin account %{disposition->} for %{p0}");

var part81 = match("MESSAGE#47:00002:43/1_0", "nwparser.p0", "'%{username}'%{p0}");

var part82 = match("MESSAGE#47:00002:43/1_1", "nwparser.p0", "\"%{username}\"%{p0}");

var select20 = linear_select([
	part81,
	part82,
]);

var part83 = match("MESSAGE#47:00002:43/2", "nwparser.p0", "%{}by %{administrator->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})");

var all18 = all_match({
	processors: [
		part80,
		select20,
		part83,
	],
	on_success: processor_chain([
		dup42,
		dup29,
		dup43,
		dup38,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg48 = msg("00002:43", all18);

var part84 = match("MESSAGE#48:00002:50", "nwparser.payload", "Admin account %{disposition->} for \"%{username}\" by %{administrator->} via %{logon_type->} from host %{saddr}:%{sport->} (%{fld1})", processor_chain([
	dup42,
	dup29,
	dup43,
	dup38,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg49 = msg("00002:50", part84);

var part85 = match("MESSAGE#49:00002:51", "nwparser.payload", "Admin account %{disposition->} for \"%{username}\" by %{administrator->} %{fld2->} via %{logon_type->} (%{fld1})", processor_chain([
	dup42,
	dup29,
	dup43,
	dup38,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg50 = msg("00002:51", part85);

var part86 = match("MESSAGE#50:00002:45", "nwparser.payload", "Extraneous exit is issued by %{username->} via %{logon_type->} from host %{saddr}:%{sport->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg51 = msg("00002:45", part86);

var part87 = match("MESSAGE#51:00002:47/0_0", "nwparser.payload", "Ping of Death attack protection %{p0}");

var part88 = match("MESSAGE#51:00002:47/0_1", "nwparser.payload", "Src Route IP option filtering %{p0}");

var part89 = match("MESSAGE#51:00002:47/0_2", "nwparser.payload", "Teardrop attack protection %{p0}");

var part90 = match("MESSAGE#51:00002:47/0_3", "nwparser.payload", "Land attack protection %{p0}");

var part91 = match("MESSAGE#51:00002:47/0_4", "nwparser.payload", "SYN flood protection %{p0}");

var select21 = linear_select([
	part87,
	part88,
	part89,
	part90,
	part91,
]);

var part92 = match("MESSAGE#51:00002:47/1", "nwparser.p0", "is %{disposition->} on zone %{zone->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})");

var all19 = all_match({
	processors: [
		select21,
		part92,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg52 = msg("00002:47", all19);

var part93 = match("MESSAGE#52:00002:48/0", "nwparser.payload", "Dropping pkts if not %{p0}");

var part94 = match("MESSAGE#52:00002:48/1_0", "nwparser.p0", "exactly same with incoming if %{p0}");

var part95 = match("MESSAGE#52:00002:48/1_1", "nwparser.p0", "in route table %{p0}");

var select22 = linear_select([
	part94,
	part95,
]);

var part96 = match("MESSAGE#52:00002:48/2", "nwparser.p0", "(IP spoof protection) is %{disposition->} on zone %{zone->} by %{username->} via %{p0}");

var part97 = match("MESSAGE#52:00002:48/3_0", "nwparser.p0", "NSRP Peer. (%{p0}");

var select23 = linear_select([
	part97,
	dup45,
]);

var all20 = all_match({
	processors: [
		part93,
		select22,
		part96,
		select23,
		dup41,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg53 = msg("00002:48", all20);

var part98 = match("MESSAGE#53:00002:52/0", "nwparser.payload", "%{signame->} %{p0}");

var part99 = match("MESSAGE#53:00002:52/1_0", "nwparser.p0", "protection%{p0}");

var part100 = match("MESSAGE#53:00002:52/1_1", "nwparser.p0", "limiting%{p0}");

var part101 = match("MESSAGE#53:00002:52/1_2", "nwparser.p0", "detection%{p0}");

var part102 = match("MESSAGE#53:00002:52/1_3", "nwparser.p0", "filtering %{p0}");

var select24 = linear_select([
	part99,
	part100,
	part101,
	part102,
]);

var part103 = match("MESSAGE#53:00002:52/2", "nwparser.p0", "%{}is %{disposition->} on zone %{zone->} by %{p0}");

var part104 = match("MESSAGE#53:00002:52/3_1", "nwparser.p0", "admin via %{p0}");

var select25 = linear_select([
	dup46,
	part104,
	dup47,
]);

var select26 = linear_select([
	dup48,
	dup45,
]);

var all21 = all_match({
	processors: [
		part98,
		select24,
		part103,
		select25,
		select26,
		dup41,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg54 = msg("00002:52", all21);

var part105 = match("MESSAGE#54:00002:53", "nwparser.payload", "Admin password for account \"%{username}\" has been %{disposition->} by %{administrator->} via %{logon_type->} (%{fld1})", processor_chain([
	dup42,
	dup43,
	dup38,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg55 = msg("00002:53", part105);

var part106 = match("MESSAGE#55:00002:54/0", "nwparser.payload", "Traffic shaping clearing DSCP selector is turned O%{p0}");

var part107 = match("MESSAGE#55:00002:54/1_0", "nwparser.p0", "FF%{p0}");

var part108 = match("MESSAGE#55:00002:54/1_1", "nwparser.p0", "N%{p0}");

var select27 = linear_select([
	part107,
	part108,
]);

var all22 = all_match({
	processors: [
		part106,
		select27,
		dup49,
	],
	on_success: processor_chain([
		dup50,
		dup43,
		dup51,
		dup2,
		dup3,
		dup4,
		dup5,
		dup9,
	]),
});

var msg56 = msg("00002:54", all22);

var part109 = match("MESSAGE#56:00002/0", "nwparser.payload", "%{change_attribute->} %{p0}");

var part110 = match("MESSAGE#56:00002/1_0", "nwparser.p0", "has been changed%{p0}");

var select28 = linear_select([
	part110,
	dup52,
]);

var part111 = match("MESSAGE#56:00002/2", "nwparser.p0", "%{}from %{change_old->} to %{change_new}");

var all23 = all_match({
	processors: [
		part109,
		select28,
		part111,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg57 = msg("00002", all23);

var part112 = match("MESSAGE#1215:00002:56", "nwparser.payload", "Admin user \"%{administrator}\" login attempt for %{logon_type}(%{network_service}) management (port %{network_port}) from %{saddr}:%{sport->} failed. (%{fld1})", processor_chain([
	dup53,
	dup9,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg58 = msg("00002:56", part112);

var select29 = linear_select([
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
]);

var part113 = match("MESSAGE#57:00003", "nwparser.payload", "Multiple authentication failures have been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}", processor_chain([
	dup53,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg59 = msg("00003", part113);

var part114 = match("MESSAGE#58:00003:01", "nwparser.payload", "Multiple authentication failures have been detected!%{}", processor_chain([
	dup53,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg60 = msg("00003:01", part114);

var part115 = match("MESSAGE#59:00003:02", "nwparser.payload", "The console debug buffer has been %{disposition}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg61 = msg("00003:02", part115);

var part116 = match("MESSAGE#60:00003:03", "nwparser.payload", "%{change_attribute->} changed from %{change_old->} to %{change_new}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg62 = msg("00003:03", part116);

var part117 = match("MESSAGE#61:00003:05/1_0", "nwparser.p0", "serial%{p0}");

var part118 = match("MESSAGE#61:00003:05/1_1", "nwparser.p0", "local%{p0}");

var select30 = linear_select([
	part117,
	part118,
]);

var part119 = match("MESSAGE#61:00003:05/2", "nwparser.p0", "%{}console has been %{disposition->} by admin %{administrator}.");

var all24 = all_match({
	processors: [
		dup55,
		select30,
		part119,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg63 = msg("00003:05", all24);

var select31 = linear_select([
	msg59,
	msg60,
	msg61,
	msg62,
	msg63,
]);

var part120 = match("MESSAGE#62:00004", "nwparser.payload", "%{info}DNS server IP has been changed", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg64 = msg("00004", part120);

var part121 = match("MESSAGE#63:00004:01", "nwparser.payload", "DNS cache table has been %{disposition}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg65 = msg("00004:01", part121);

var part122 = match("MESSAGE#64:00004:02", "nwparser.payload", "Daily DNS lookup has been %{disposition}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg66 = msg("00004:02", part122);

var part123 = match("MESSAGE#65:00004:03", "nwparser.payload", "Daily DNS lookup time has been %{disposition}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg67 = msg("00004:03", part123);

var part124 = match("MESSAGE#66:00004:04/0", "nwparser.payload", "%{signame->} has been detected! From %{saddr->} to %{daddr->} using protocol %{protocol->} on %{p0}");

var part125 = match("MESSAGE#66:00004:04/2", "nwparser.p0", "%{} %{interface->} %{space}The attack occurred %{dclass_counter1->} times");

var all25 = all_match({
	processors: [
		part124,
		dup337,
		part125,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup4,
		dup5,
		dup59,
		dup3,
		dup60,
	]),
});

var msg68 = msg("00004:04", all25);

var part126 = match("MESSAGE#67:00004:05", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol}", processor_chain([
	dup58,
	dup2,
	dup4,
	dup5,
	dup3,
	dup61,
]));

var msg69 = msg("00004:05", part126);

var part127 = match("MESSAGE#68:00004:06", "nwparser.payload", "DNS lookup time has been changed to start at %{fld2}:%{fld3->} with an interval of %{fld4}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg70 = msg("00004:06", part127);

var part128 = match("MESSAGE#69:00004:07", "nwparser.payload", "DNS cache table entries have been refreshed as result of external event.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg71 = msg("00004:07", part128);

var part129 = match("MESSAGE#70:00004:08", "nwparser.payload", "DNS Proxy module has been %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg72 = msg("00004:08", part129);

var part130 = match("MESSAGE#71:00004:09", "nwparser.payload", "DNS Proxy module has more concurrent client requests than allowed.%{}", processor_chain([
	dup62,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg73 = msg("00004:09", part130);

var part131 = match("MESSAGE#72:00004:10", "nwparser.payload", "DNS Proxy server select table entries exceeded maximum limit.%{}", processor_chain([
	dup62,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg74 = msg("00004:10", part131);

var part132 = match("MESSAGE#73:00004:11", "nwparser.payload", "Proxy server select table added with domain %{domain}, interface %{interface}, primary-ip %{fld2}, secondary-ip %{fld3}, tertiary-ip %{fld4}, failover %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg75 = msg("00004:11", part132);

var part133 = match("MESSAGE#74:00004:12", "nwparser.payload", "DNS Proxy server select table entry %{disposition->} with domain %{domain}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg76 = msg("00004:12", part133);

var part134 = match("MESSAGE#75:00004:13", "nwparser.payload", "DDNS server %{domain->} returned incorrect ip %{fld2}, local-ip should be %{fld3}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg77 = msg("00004:13", part134);

var part135 = match("MESSAGE#76:00004:14/1_0", "nwparser.p0", "automatically refreshed %{p0}");

var part136 = match("MESSAGE#76:00004:14/1_1", "nwparser.p0", "refreshed by HA %{p0}");

var select32 = linear_select([
	part135,
	part136,
]);

var all26 = all_match({
	processors: [
		dup63,
		select32,
		dup49,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg78 = msg("00004:14", all26);

var part137 = match("MESSAGE#77:00004:15", "nwparser.payload", "DNS entries have been refreshed as result of DNS server address change. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg79 = msg("00004:15", part137);

var part138 = match("MESSAGE#78:00004:16", "nwparser.payload", "DNS entries have been manually refreshed. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg80 = msg("00004:16", part138);

var all27 = all_match({
	processors: [
		dup64,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup4,
		dup59,
		dup9,
		dup5,
		dup3,
		dup60,
	]),
});

var msg81 = msg("00004:17", all27);

var select33 = linear_select([
	msg64,
	msg65,
	msg66,
	msg67,
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
]);

var part139 = match("MESSAGE#80:00005", "nwparser.payload", "%{signame->} alarm threshold from the same source has been changed to %{trigger_val}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg82 = msg("00005", part139);

var part140 = match("MESSAGE#81:00005:01", "nwparser.payload", "Logging of %{fld2->} traffic to self has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg83 = msg("00005:01", part140);

var part141 = match("MESSAGE#82:00005:02", "nwparser.payload", "SYN flood %{fld2->} has been changed to %{fld3}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg84 = msg("00005:02", part141);

var part142 = match("MESSAGE#83:00005:03/0", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{p0}");

var part143 = match("MESSAGE#83:00005:03/4", "nwparser.p0", "%{fld99}interface %{interface->} %{p0}");

var part144 = match("MESSAGE#83:00005:03/5_0", "nwparser.p0", "in zone %{zone}. %{p0}");

var select34 = linear_select([
	part144,
	dup73,
]);

var part145 = match("MESSAGE#83:00005:03/6", "nwparser.p0", "%{space}The attack occurred %{dclass_counter1->} times");

var all28 = all_match({
	processors: [
		part142,
		dup339,
		dup70,
		dup340,
		part143,
		select34,
		part145,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup3,
		dup4,
		dup5,
		dup59,
		dup61,
	]),
});

var msg85 = msg("00005:03", all28);

var msg86 = msg("00005:04", dup341);

var part146 = match("MESSAGE#85:00005:05", "nwparser.payload", "SYN flood drop pak in %{fld2->} mode when receiving unknown dst mac has been %{disposition->} on %{zone}.", processor_chain([
	setc("eventcategory","1001020100"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg87 = msg("00005:05", part146);

var part147 = match("MESSAGE#86:00005:06/1", "nwparser.p0", "flood timeout has been set to %{trigger_val->} on %{zone}.");

var all29 = all_match({
	processors: [
		dup342,
		part147,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg88 = msg("00005:06", all29);

var part148 = match("MESSAGE#87:00005:07/0", "nwparser.payload", "SYN flood %{p0}");

var part149 = match("MESSAGE#87:00005:07/1_0", "nwparser.p0", "alarm threshold %{p0}");

var part150 = match("MESSAGE#87:00005:07/1_1", "nwparser.p0", "packet queue size %{p0}");

var part151 = match("MESSAGE#87:00005:07/1_3", "nwparser.p0", "attack threshold %{p0}");

var part152 = match("MESSAGE#87:00005:07/1_4", "nwparser.p0", "same source IP threshold %{p0}");

var select35 = linear_select([
	part149,
	part150,
	dup76,
	part151,
	part152,
]);

var part153 = match("MESSAGE#87:00005:07/2", "nwparser.p0", "is set to %{trigger_val}.");

var all30 = all_match({
	processors: [
		part148,
		select35,
		part153,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg89 = msg("00005:07", all30);

var part154 = match("MESSAGE#88:00005:08/1", "nwparser.p0", "flood same %{p0}");

var select36 = linear_select([
	dup77,
	dup78,
]);

var part155 = match("MESSAGE#88:00005:08/3", "nwparser.p0", "ip threshold has been set to %{trigger_val->} on %{zone}.");

var all31 = all_match({
	processors: [
		dup342,
		part154,
		select36,
		part155,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg90 = msg("00005:08", all31);

var part156 = match("MESSAGE#89:00005:09", "nwparser.payload", "Screen service %{service->} is %{disposition->} on interface %{interface}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg91 = msg("00005:09", part156);

var part157 = match("MESSAGE#90:00005:10", "nwparser.payload", "Screen service %{service->} is %{disposition->} on %{zone}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg92 = msg("00005:10", part157);

var part158 = match("MESSAGE#91:00005:11/0", "nwparser.payload", "The SYN flood %{p0}");

var part159 = match("MESSAGE#91:00005:11/1_0", "nwparser.p0", "alarm threshold%{}");

var part160 = match("MESSAGE#91:00005:11/1_1", "nwparser.p0", "packet queue size%{}");

var part161 = match("MESSAGE#91:00005:11/1_2", "nwparser.p0", "timeout value%{}");

var part162 = match("MESSAGE#91:00005:11/1_3", "nwparser.p0", "attack threshold%{}");

var part163 = match("MESSAGE#91:00005:11/1_4", "nwparser.p0", "same source IP%{}");

var select37 = linear_select([
	part159,
	part160,
	part161,
	part162,
	part163,
]);

var all32 = all_match({
	processors: [
		part158,
		select37,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg93 = msg("00005:11", all32);

var part164 = match("MESSAGE#92:00005:12", "nwparser.payload", "The SYN-ACK-ACK proxy threshold value has been set to %{trigger_val->} on %{interface}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg94 = msg("00005:12", part164);

var part165 = match("MESSAGE#93:00005:13", "nwparser.payload", "The session limit threshold has been set to %{trigger_val->} on %{zone}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg95 = msg("00005:13", part165);

var part166 = match("MESSAGE#94:00005:14", "nwparser.payload", "syn proxy drop packet with unknown mac!%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg96 = msg("00005:14", part166);

var part167 = match("MESSAGE#95:00005:15", "nwparser.payload", "%{signame->} alarm threshold has been changed to %{trigger_val}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg97 = msg("00005:15", part167);

var part168 = match("MESSAGE#96:00005:16", "nwparser.payload", "%{signame->} threshold has been set to %{trigger_val->} on %{zone}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg98 = msg("00005:16", part168);

var part169 = match("MESSAGE#97:00005:17/1_0", "nwparser.p0", "destination-based %{p0}");

var part170 = match("MESSAGE#97:00005:17/1_1", "nwparser.p0", "source-based %{p0}");

var select38 = linear_select([
	part169,
	part170,
]);

var part171 = match("MESSAGE#97:00005:17/2", "nwparser.p0", "session-limit threshold has been set at %{trigger_val->} in zone %{zone}.");

var all33 = all_match({
	processors: [
		dup79,
		select38,
		part171,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg99 = msg("00005:17", all33);

var all34 = all_match({
	processors: [
		dup80,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup84,
		dup2,
		dup59,
		dup9,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg100 = msg("00005:18", all34);

var part172 = match("MESSAGE#99:00005:19", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup84,
	dup2,
	dup3,
	dup4,
	dup5,
	dup59,
	dup61,
]));

var msg101 = msg("00005:19", part172);

var part173 = match("MESSAGE#100:00005:20", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone->} int %{interface}).%{space->} Occurred %{fld2->} times. (%{fld1})\u003c\u003c%{fld6}>", processor_chain([
	dup84,
	dup9,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg102 = msg("00005:20", part173);

var select39 = linear_select([
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

var part174 = match("MESSAGE#101:00006", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup85,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup61,
]));

var msg103 = msg("00006", part174);

var part175 = match("MESSAGE#102:00006:01", "nwparser.payload", "Hostname set to \"%{hostname}\"", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg104 = msg("00006:01", part175);

var part176 = match("MESSAGE#103:00006:02", "nwparser.payload", "Domain set to %{domain}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg105 = msg("00006:02", part176);

var part177 = match("MESSAGE#104:00006:03", "nwparser.payload", "An optional ScreenOS feature has been activated via a software key.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg106 = msg("00006:03", part177);

var part178 = match("MESSAGE#105:00006:04/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{p0}");

var all35 = all_match({
	processors: [
		part178,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup84,
		dup2,
		dup59,
		dup9,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg107 = msg("00006:04", all35);

var all36 = all_match({
	processors: [
		dup64,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup84,
		dup2,
		dup59,
		dup9,
		dup3,
		dup4,
		dup5,
		dup60,
	]),
});

var msg108 = msg("00006:05", all36);

var select40 = linear_select([
	msg103,
	msg104,
	msg105,
	msg106,
	msg107,
	msg108,
]);

var part179 = match("MESSAGE#107:00007", "nwparser.payload", "HA cluster ID has been changed to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg109 = msg("00007", part179);

var part180 = match("MESSAGE#108:00007:01", "nwparser.payload", "%{change_attribute->} of the local NetScreen device has changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg110 = msg("00007:01", part180);

var part181 = match("MESSAGE#109:00007:02/0", "nwparser.payload", "HA state of the local device has changed to backup because a device with a %{p0}");

var part182 = match("MESSAGE#109:00007:02/1_0", "nwparser.p0", "higher priority has been detected%{}");

var part183 = match("MESSAGE#109:00007:02/1_1", "nwparser.p0", "lower MAC value has been detected%{}");

var select41 = linear_select([
	part182,
	part183,
]);

var all37 = all_match({
	processors: [
		part181,
		select41,
	],
	on_success: processor_chain([
		dup86,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg111 = msg("00007:02", all37);

var part184 = match("MESSAGE#110:00007:03", "nwparser.payload", "HA state of the local device has changed to init because IP tracking has failed%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg112 = msg("00007:03", part184);

var select42 = linear_select([
	dup88,
	dup89,
]);

var part185 = match("MESSAGE#111:00007:04/4", "nwparser.p0", "has been changed%{}");

var all38 = all_match({
	processors: [
		dup87,
		select42,
		dup23,
		dup344,
		part185,
	],
	on_success: processor_chain([
		dup91,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg113 = msg("00007:04", all38);

var part186 = match("MESSAGE#112:00007:05", "nwparser.payload", "HA: Local NetScreen device has been elected backup because a master already exists%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg114 = msg("00007:05", part186);

var part187 = match("MESSAGE#113:00007:06", "nwparser.payload", "HA: Local NetScreen device has been elected backup because its MAC value is higher than those of other devices in the cluster%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg115 = msg("00007:06", part187);

var part188 = match("MESSAGE#114:00007:07", "nwparser.payload", "HA: Local NetScreen device has been elected backup because its priority value is higher than those of other devices in the cluster%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg116 = msg("00007:07", part188);

var part189 = match("MESSAGE#115:00007:08", "nwparser.payload", "HA: Local device has been elected master because no other master exists%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg117 = msg("00007:08", part189);

var part190 = match("MESSAGE#116:00007:09", "nwparser.payload", "HA: Local device priority has been changed to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg118 = msg("00007:09", part190);

var part191 = match("MESSAGE#117:00007:10", "nwparser.payload", "HA: Previous master has promoted the local NetScreen device to master%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg119 = msg("00007:10", part191);

var part192 = match("MESSAGE#118:00007:11/0", "nwparser.payload", "IP tracking device failover threshold has been %{p0}");

var select43 = linear_select([
	dup92,
	dup93,
]);

var all39 = all_match({
	processors: [
		part192,
		select43,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg120 = msg("00007:11", all39);

var part193 = match("MESSAGE#119:00007:12", "nwparser.payload", "IP tracking has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg121 = msg("00007:12", part193);

var part194 = match("MESSAGE#120:00007:13", "nwparser.payload", "IP tracking to %{hostip->} with interval %{fld2->} threshold %{trigger_val->} weight %{fld4->} interface %{interface->} method %{fld5->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg122 = msg("00007:13", part194);

var part195 = match("MESSAGE#121:00007:14", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr->} using protocol %{protocol->} on zone %{zone->} interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup85,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup60,
]));

var msg123 = msg("00007:14", part195);

var part196 = match("MESSAGE#122:00007:15", "nwparser.payload", "Primary HA interface has been changed to %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg124 = msg("00007:15", part196);

var part197 = match("MESSAGE#123:00007:16", "nwparser.payload", "Reporting of HA configuration and status changes to NetScreen-Global Manager has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg125 = msg("00007:16", part197);

var part198 = match("MESSAGE#124:00007:17", "nwparser.payload", "Tracked IP %{hostip->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg126 = msg("00007:17", part198);

var part199 = match("MESSAGE#125:00007:18/0", "nwparser.payload", "Tracked IP %{hostip->} options have been changed from int %{fld2->} thr %{fld3->} wgt %{fld4->} inf %{fld5->} %{p0}");

var part200 = match("MESSAGE#125:00007:18/1_0", "nwparser.p0", "ping %{p0}");

var part201 = match("MESSAGE#125:00007:18/1_1", "nwparser.p0", "ARP %{p0}");

var select44 = linear_select([
	part200,
	part201,
]);

var part202 = match("MESSAGE#125:00007:18/2", "nwparser.p0", "to %{fld6->} %{p0}");

var part203 = match("MESSAGE#125:00007:18/3_0", "nwparser.p0", "ping%{}");

var part204 = match("MESSAGE#125:00007:18/3_1", "nwparser.p0", "ARP%{}");

var select45 = linear_select([
	part203,
	part204,
]);

var all40 = all_match({
	processors: [
		part199,
		select44,
		part202,
		select45,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg127 = msg("00007:18", all40);

var part205 = match("MESSAGE#126:00007:20", "nwparser.payload", "Change %{change_attribute->} path from %{change_old->} to %{change_new}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg128 = msg("00007:20", part205);

var part206 = match("MESSAGE#127:00007:21/0", "nwparser.payload", "HA Slave is %{p0}");

var all41 = all_match({
	processors: [
		part206,
		dup345,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg129 = msg("00007:21", all41);

var part207 = match("MESSAGE#128:00007:22", "nwparser.payload", "HA change group id to %{groupid}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg130 = msg("00007:22", part207);

var part208 = match("MESSAGE#129:00007:23", "nwparser.payload", "HA change priority to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg131 = msg("00007:23", part208);

var part209 = match("MESSAGE#130:00007:24", "nwparser.payload", "HA change state to init%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg132 = msg("00007:24", part209);

var part210 = match("MESSAGE#131:00007:25", "nwparser.payload", "HA: Change state to initial state.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg133 = msg("00007:25", part210);

var part211 = match("MESSAGE#132:00007:26/0", "nwparser.payload", "HA: Change state to slave for %{p0}");

var part212 = match("MESSAGE#132:00007:26/1_0", "nwparser.p0", "tracking ip failed%{}");

var part213 = match("MESSAGE#132:00007:26/1_1", "nwparser.p0", "linkdown%{}");

var select46 = linear_select([
	part212,
	part213,
]);

var all42 = all_match({
	processors: [
		part211,
		select46,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg134 = msg("00007:26", all42);

var part214 = match("MESSAGE#133:00007:27", "nwparser.payload", "HA: Change to master command issued from original master to change state%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg135 = msg("00007:27", part214);

var part215 = match("MESSAGE#134:00007:28", "nwparser.payload", "HA: Elected master no other master%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg136 = msg("00007:28", part215);

var part216 = match("MESSAGE#135:00007:29/0", "nwparser.payload", "HA: Elected slave %{p0}");

var part217 = match("MESSAGE#135:00007:29/1_0", "nwparser.p0", "lower priority%{}");

var part218 = match("MESSAGE#135:00007:29/1_1", "nwparser.p0", "MAC value is larger%{}");

var part219 = match("MESSAGE#135:00007:29/1_2", "nwparser.p0", "master already exists%{}");

var part220 = match("MESSAGE#135:00007:29/1_3", "nwparser.p0", "detect new master with higher priority%{}");

var part221 = match("MESSAGE#135:00007:29/1_4", "nwparser.p0", "detect new master with smaller MAC value%{}");

var select47 = linear_select([
	part217,
	part218,
	part219,
	part220,
	part221,
]);

var all43 = all_match({
	processors: [
		part216,
		select47,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg137 = msg("00007:29", all43);

var part222 = match("MESSAGE#136:00007:30", "nwparser.payload", "HA: Promoted master command issued from original master to change state%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg138 = msg("00007:30", part222);

var part223 = match("MESSAGE#137:00007:31/0", "nwparser.payload", "HA: ha link %{p0}");

var all44 = all_match({
	processors: [
		part223,
		dup345,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg139 = msg("00007:31", all44);

var part224 = match("MESSAGE#138:00007:32/0", "nwparser.payload", "NSRP %{fld2->} %{p0}");

var select48 = linear_select([
	dup89,
	dup88,
]);

var part225 = match("MESSAGE#138:00007:32/4", "nwparser.p0", "changed.%{}");

var all45 = all_match({
	processors: [
		part224,
		select48,
		dup23,
		dup344,
		part225,
	],
	on_success: processor_chain([
		dup91,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg140 = msg("00007:32", all45);

var part226 = match("MESSAGE#139:00007:33/0_0", "nwparser.payload", "NSRP: VSD %{p0}");

var part227 = match("MESSAGE#139:00007:33/0_1", "nwparser.payload", "Virtual Security Device group %{p0}");

var select49 = linear_select([
	part226,
	part227,
]);

var part228 = match("MESSAGE#139:00007:33/1", "nwparser.p0", "%{fld2->} change%{p0}");

var part229 = match("MESSAGE#139:00007:33/2_0", "nwparser.p0", "d %{p0}");

var select50 = linear_select([
	part229,
	dup96,
]);

var part230 = match("MESSAGE#139:00007:33/3", "nwparser.p0", "to %{fld3->} mode.");

var all46 = all_match({
	processors: [
		select49,
		part228,
		select50,
		part230,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg141 = msg("00007:33", all46);

var part231 = match("MESSAGE#140:00007:34", "nwparser.payload", "NSRP: message %{fld2->} dropped: invalid encryption password.", processor_chain([
	dup97,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg142 = msg("00007:34", part231);

var part232 = match("MESSAGE#141:00007:35", "nwparser.payload", "NSRP: nsrp interface change to %{interface}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg143 = msg("00007:35", part232);

var part233 = match("MESSAGE#142:00007:36", "nwparser.payload", "RTO mirror group id=%{groupid->} direction= %{direction->} local unit=%{fld3->} duplicate from unit=%{fld4}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg144 = msg("00007:36", part233);

var part234 = match("MESSAGE#143:00007:37/0", "nwparser.payload", "RTO mirror group id=%{groupid->} direction= %{direction->} is %{p0}");

var all47 = all_match({
	processors: [
		part234,
		dup346,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg145 = msg("00007:37", all47);

var part235 = match("MESSAGE#144:00007:38/0", "nwparser.payload", "RTO mirror group id=%{groupid->} direction= %{direction->} peer=%{fld3->} from %{p0}");

var part236 = match("MESSAGE#144:00007:38/4", "nwparser.p0", "state %{p0}");

var part237 = match("MESSAGE#144:00007:38/5_0", "nwparser.p0", "missed heartbeat%{}");

var part238 = match("MESSAGE#144:00007:38/5_1", "nwparser.p0", "group detached%{}");

var select51 = linear_select([
	part237,
	part238,
]);

var all48 = all_match({
	processors: [
		part235,
		dup347,
		dup103,
		dup347,
		part236,
		select51,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg146 = msg("00007:38", all48);

var part239 = match("MESSAGE#145:00007:39/0", "nwparser.payload", "RTO mirror group id=%{groupid->} is %{p0}");

var all49 = all_match({
	processors: [
		part239,
		dup346,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg147 = msg("00007:39", all49);

var part240 = match("MESSAGE#146:00007:40", "nwparser.payload", "Remove pathname %{fld2->} (ifnum=%{fld3}) as secondary HA path", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg148 = msg("00007:40", part240);

var part241 = match("MESSAGE#147:00007:41", "nwparser.payload", "Session sync ended by unit=%{fld2}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg149 = msg("00007:41", part241);

var part242 = match("MESSAGE#148:00007:42", "nwparser.payload", "Set secondary HA path to %{fld2->} (ifnum=%{fld3})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg150 = msg("00007:42", part242);

var part243 = match("MESSAGE#149:00007:43", "nwparser.payload", "VSD %{change_attribute->} changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg151 = msg("00007:43", part243);

var part244 = match("MESSAGE#150:00007:44", "nwparser.payload", "vsd group id=%{groupid->} is %{disposition->} total number=%{fld3}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg152 = msg("00007:44", part244);

var part245 = match("MESSAGE#151:00007:45", "nwparser.payload", "vsd group %{group->} local unit %{change_attribute->} changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg153 = msg("00007:45", part245);

var part246 = match("MESSAGE#152:00007:46", "nwparser.payload", "%{signame->} has been detected! From %{saddr->} to %{daddr->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup85,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup60,
]));

var msg154 = msg("00007:46", part246);

var part247 = match("MESSAGE#153:00007:47", "nwparser.payload", "The HA channel changed to interface %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg155 = msg("00007:47", part247);

var part248 = match("MESSAGE#154:00007:48", "nwparser.payload", "Message %{fld2->} was dropped because it contained an invalid encryption password.", processor_chain([
	dup97,
	dup2,
	dup3,
	dup4,
	setc("disposition","dropped"),
	setc("result","Invalid encryption Password"),
]));

var msg156 = msg("00007:48", part248);

var part249 = match("MESSAGE#155:00007:49", "nwparser.payload", "The %{change_attribute->} of all Virtual Security Device groups changed from %{change_old->} to %{change_new}", processor_chain([
	setc("eventcategory","1604000000"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg157 = msg("00007:49", part249);

var part250 = match("MESSAGE#156:00007:50/0", "nwparser.payload", "Device %{fld2->} %{p0}");

var part251 = match("MESSAGE#156:00007:50/1_0", "nwparser.p0", "has joined %{p0}");

var part252 = match("MESSAGE#156:00007:50/1_1", "nwparser.p0", "quit current %{p0}");

var select52 = linear_select([
	part251,
	part252,
]);

var part253 = match("MESSAGE#156:00007:50/2", "nwparser.p0", "NSRP cluster %{fld3}");

var all50 = all_match({
	processors: [
		part250,
		select52,
		part253,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg158 = msg("00007:50", all50);

var part254 = match("MESSAGE#157:00007:51/0", "nwparser.payload", "Virtual Security Device group %{group->} was %{p0}");

var part255 = match("MESSAGE#157:00007:51/1_1", "nwparser.p0", "deleted %{p0}");

var select53 = linear_select([
	dup104,
	part255,
]);

var select54 = linear_select([
	dup105,
	dup73,
]);

var part256 = match("MESSAGE#157:00007:51/4", "nwparser.p0", "The total number of members in the group %{p0}");

var select55 = linear_select([
	dup106,
	dup107,
]);

var all51 = all_match({
	processors: [
		part254,
		select53,
		dup23,
		select54,
		part256,
		select55,
		dup108,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg159 = msg("00007:51", all51);

var part257 = match("MESSAGE#158:00007:52", "nwparser.payload", "Virtual Security Device group %{group->} %{change_attribute->} changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg160 = msg("00007:52", part257);

var part258 = match("MESSAGE#159:00007:53", "nwparser.payload", "The secondary HA path of the devices was set to interface %{interface->} with ifnum %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg161 = msg("00007:53", part258);

var part259 = match("MESSAGE#160:00007:54", "nwparser.payload", "The %{change_attribute->} of the devices changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg162 = msg("00007:54", part259);

var part260 = match("MESSAGE#161:00007:55", "nwparser.payload", "The interface %{interface->} with ifnum %{fld2->} was removed from the secondary HA path of the devices.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg163 = msg("00007:55", part260);

var part261 = match("MESSAGE#162:00007:56", "nwparser.payload", "The probe that detects the status of High Availability link %{fld2->} was %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg164 = msg("00007:56", part261);

var select56 = linear_select([
	dup109,
	dup110,
]);

var select57 = linear_select([
	dup111,
	dup112,
]);

var part262 = match("MESSAGE#163:00007:57/4", "nwparser.p0", "the probe detecting the status of High Availability link %{fld2->} was set to %{fld3}");

var all52 = all_match({
	processors: [
		dup55,
		select56,
		dup23,
		select57,
		part262,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg165 = msg("00007:57", all52);

var part263 = match("MESSAGE#164:00007:58", "nwparser.payload", "A request by device %{fld2->} for session synchronization(s) was accepted.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg166 = msg("00007:58", part263);

var part264 = match("MESSAGE#165:00007:59", "nwparser.payload", "The current session synchronization by device %{fld2->} completed.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg167 = msg("00007:59", part264);

var part265 = match("MESSAGE#166:00007:60", "nwparser.payload", "Run Time Object mirror group %{group->} direction was set to %{direction}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg168 = msg("00007:60", part265);

var part266 = match("MESSAGE#167:00007:61", "nwparser.payload", "Run Time Object mirror group %{group->} was set.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg169 = msg("00007:61", part266);

var part267 = match("MESSAGE#168:00007:62", "nwparser.payload", "Run Time Object mirror group %{group->} with direction %{direction->} was unset.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg170 = msg("00007:62", part267);

var part268 = match("MESSAGE#169:00007:63", "nwparser.payload", "RTO mirror group %{group->} was unset.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg171 = msg("00007:63", part268);

var part269 = match("MESSAGE#170:00007:64/1", "nwparser.p0", "%{fld2->} was removed from the monitoring list %{p0}");

var part270 = match("MESSAGE#170:00007:64/3", "nwparser.p0", "%{fld3}");

var all53 = all_match({
	processors: [
		dup348,
		part269,
		dup349,
		part270,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg172 = msg("00007:64", all53);

var part271 = match("MESSAGE#171:00007:65/1", "nwparser.p0", "%{fld2->} with weight %{fld3->} was added%{p0}");

var part272 = match("MESSAGE#171:00007:65/2_0", "nwparser.p0", " to or updated on %{p0}");

var part273 = match("MESSAGE#171:00007:65/2_1", "nwparser.p0", "/updated to %{p0}");

var select58 = linear_select([
	part272,
	part273,
]);

var part274 = match("MESSAGE#171:00007:65/3", "nwparser.p0", "the monitoring list %{p0}");

var part275 = match("MESSAGE#171:00007:65/5", "nwparser.p0", "%{fld4}");

var all54 = all_match({
	processors: [
		dup348,
		part271,
		select58,
		part274,
		dup349,
		part275,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg173 = msg("00007:65", all54);

var part276 = match("MESSAGE#172:00007:66/0_0", "nwparser.payload", "The monitoring %{p0}");

var part277 = match("MESSAGE#172:00007:66/0_1", "nwparser.payload", "Monitoring %{p0}");

var select59 = linear_select([
	part276,
	part277,
]);

var part278 = match("MESSAGE#172:00007:66/1", "nwparser.p0", "threshold was modified to %{trigger_val->} o%{p0}");

var part279 = match("MESSAGE#172:00007:66/2_0", "nwparser.p0", "f %{p0}");

var select60 = linear_select([
	part279,
	dup115,
]);

var all55 = all_match({
	processors: [
		select59,
		part278,
		select60,
		dup108,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg174 = msg("00007:66", all55);

var part280 = match("MESSAGE#173:00007:67", "nwparser.payload", "NSRP data forwarding %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg175 = msg("00007:67", part280);

var part281 = match("MESSAGE#174:00007:68/0", "nwparser.payload", "NSRP b%{p0}");

var part282 = match("MESSAGE#174:00007:68/1_0", "nwparser.p0", "lack %{p0}");

var part283 = match("MESSAGE#174:00007:68/1_1", "nwparser.p0", "ack %{p0}");

var select61 = linear_select([
	part282,
	part283,
]);

var part284 = match("MESSAGE#174:00007:68/2", "nwparser.p0", "hole prevention %{disposition}. Master(s) of Virtual Security Device groups %{p0}");

var part285 = match("MESSAGE#174:00007:68/3_0", "nwparser.p0", "may not exist %{p0}");

var part286 = match("MESSAGE#174:00007:68/3_1", "nwparser.p0", "always exists %{p0}");

var select62 = linear_select([
	part285,
	part286,
]);

var all56 = all_match({
	processors: [
		part281,
		select61,
		part284,
		select62,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg176 = msg("00007:68", all56);

var part287 = match("MESSAGE#175:00007:69", "nwparser.payload", "NSRP Run Time Object synchronization between devices was %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg177 = msg("00007:69", part287);

var part288 = match("MESSAGE#176:00007:70", "nwparser.payload", "The NSRP encryption key was changed.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg178 = msg("00007:70", part288);

var part289 = match("MESSAGE#177:00007:71", "nwparser.payload", "NSRP transparent Active-Active mode was %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg179 = msg("00007:71", part289);

var part290 = match("MESSAGE#178:00007:72", "nwparser.payload", "NSRP: nsrp link probe enable on %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg180 = msg("00007:72", part290);

var select63 = linear_select([
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
	msg145,
	msg146,
	msg147,
	msg148,
	msg149,
	msg150,
	msg151,
	msg152,
	msg153,
	msg154,
	msg155,
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
	msg167,
	msg168,
	msg169,
	msg170,
	msg171,
	msg172,
	msg173,
	msg174,
	msg175,
	msg176,
	msg177,
	msg178,
	msg179,
	msg180,
]);

var part291 = match("MESSAGE#179:00008", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup61,
]));

var msg181 = msg("00008", part291);

var msg182 = msg("00008:01", dup341);

var part292 = match("MESSAGE#181:00008:02", "nwparser.payload", "NTP settings have been changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg183 = msg("00008:02", part292);

var part293 = match("MESSAGE#182:00008:03", "nwparser.payload", "The system clock has been updated through NTP%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg184 = msg("00008:03", part293);

var part294 = match("MESSAGE#183:00008:04/0", "nwparser.payload", "System clock %{p0}");

var part295 = match("MESSAGE#183:00008:04/1_0", "nwparser.p0", "configurations have been%{p0}");

var part296 = match("MESSAGE#183:00008:04/1_1", "nwparser.p0", "was%{p0}");

var part297 = match("MESSAGE#183:00008:04/1_2", "nwparser.p0", "is%{p0}");

var select64 = linear_select([
	part295,
	part296,
	part297,
]);

var part298 = match("MESSAGE#183:00008:04/2", "nwparser.p0", "%{}changed%{p0}");

var part299 = match("MESSAGE#183:00008:04/3_0", "nwparser.p0", " by admin %{administrator}");

var part300 = match("MESSAGE#183:00008:04/3_1", "nwparser.p0", " by %{username->} (%{fld1})");

var part301 = match("MESSAGE#183:00008:04/3_2", "nwparser.p0", " by %{username}");

var part302 = match("MESSAGE#183:00008:04/3_3", "nwparser.p0", " manually.%{}");

var part303 = match("MESSAGE#183:00008:04/3_4", "nwparser.p0", " manually%{}");

var select65 = linear_select([
	part299,
	part300,
	part301,
	part302,
	part303,
	dup21,
]);

var all57 = all_match({
	processors: [
		part294,
		select64,
		part298,
		select65,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
		dup9,
	]),
});

var msg185 = msg("00008:04", all57);

var part304 = match("MESSAGE#184:00008:05", "nwparser.payload", "failed to get clock through NTP%{}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg186 = msg("00008:05", part304);

var part305 = match("MESSAGE#185:00008:06", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup59,
	dup61,
]));

var msg187 = msg("00008:06", part305);

var part306 = match("MESSAGE#186:00008:07", "nwparser.payload", "%{signame->} has been detected! From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup59,
	dup60,
]));

var msg188 = msg("00008:07", part306);

var part307 = match("MESSAGE#187:00008:08", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup59,
	dup60,
]));

var msg189 = msg("00008:08", part307);

var part308 = match("MESSAGE#188:00008:09", "nwparser.payload", "system clock is changed manually%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg190 = msg("00008:09", part308);

var part309 = match("MESSAGE#189:00008:10/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol}(zone %{p0}");

var all58 = all_match({
	processors: [
		part309,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup9,
		dup60,
	]),
});

var msg191 = msg("00008:10", all58);

var select66 = linear_select([
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
]);

var part310 = match("MESSAGE#190:00009", "nwparser.payload", "802.1Q VLAN trunking for the interface %{interface->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg192 = msg("00009", part310);

var part311 = match("MESSAGE#191:00009:01", "nwparser.payload", "802.1Q VLAN tag %{fld1->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg193 = msg("00009:01", part311);

var part312 = match("MESSAGE#192:00009:02", "nwparser.payload", "DHCP on the interface %{interface->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg194 = msg("00009:02", part312);

var part313 = match("MESSAGE#193:00009:03", "nwparser.payload", "%{change_attribute->} for interface %{interface->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg195 = msg("00009:03", part313);

var part314 = match("MESSAGE#194:00009:05", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg196 = msg("00009:05", part314);

var part315 = match("MESSAGE#195:00009:06/0_0", "nwparser.payload", "%{fld2}: The 802.1Q tag %{p0}");

var part316 = match("MESSAGE#195:00009:06/0_1", "nwparser.payload", "The 802.1Q tag %{p0}");

var select67 = linear_select([
	part315,
	part316,
]);

var select68 = linear_select([
	dup119,
	dup16,
]);

var part317 = match("MESSAGE#195:00009:06/3", "nwparser.p0", "interface %{interface->} has been %{p0}");

var part318 = match("MESSAGE#195:00009:06/4_1", "nwparser.p0", "changed to %{p0}");

var select69 = linear_select([
	dup120,
	part318,
]);

var part319 = match("MESSAGE#195:00009:06/6_0", "nwparser.p0", "%{info->} from host %{saddr}");

var part320 = match_copy("MESSAGE#195:00009:06/6_1", "nwparser.p0", "info");

var select70 = linear_select([
	part319,
	part320,
]);

var all59 = all_match({
	processors: [
		select67,
		dup118,
		select68,
		part317,
		select69,
		dup23,
		select70,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg197 = msg("00009:06", all59);

var part321 = match("MESSAGE#196:00009:07/0", "nwparser.payload", "Maximum bandwidth %{fld2->} on %{p0}");

var part322 = match("MESSAGE#196:00009:07/2", "nwparser.p0", "%{} %{interface->} is less than t%{p0}");

var part323 = match("MESSAGE#196:00009:07/3_0", "nwparser.p0", "he total %{p0}");

var part324 = match("MESSAGE#196:00009:07/3_1", "nwparser.p0", "otal %{p0}");

var select71 = linear_select([
	part323,
	part324,
]);

var part325 = match("MESSAGE#196:00009:07/4", "nwparser.p0", "guaranteed bandwidth %{fld3}");

var all60 = all_match({
	processors: [
		part321,
		dup337,
		part322,
		select71,
		part325,
	],
	on_success: processor_chain([
		dup121,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg198 = msg("00009:07", all60);

var part326 = match("MESSAGE#197:00009:09", "nwparser.payload", "The configured bandwidth setting on the interface %{interface->} has been changed to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg199 = msg("00009:09", part326);

var part327 = match("MESSAGE#198:00009:10/0", "nwparser.payload", "The operational mode for the interface %{interface->} has been changed to %{p0}");

var part328 = match("MESSAGE#198:00009:10/1_0", "nwparser.p0", "Route%{}");

var part329 = match("MESSAGE#198:00009:10/1_1", "nwparser.p0", "NAT%{}");

var select72 = linear_select([
	part328,
	part329,
]);

var all61 = all_match({
	processors: [
		part327,
		select72,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg200 = msg("00009:10", all61);

var part330 = match("MESSAGE#199:00009:11/0_0", "nwparser.payload", "%{fld1}: VLAN %{p0}");

var part331 = match("MESSAGE#199:00009:11/0_1", "nwparser.payload", "VLAN %{p0}");

var select73 = linear_select([
	part330,
	part331,
]);

var part332 = match("MESSAGE#199:00009:11/1", "nwparser.p0", "tag %{fld2->} has been %{disposition}");

var all62 = all_match({
	processors: [
		select73,
		part332,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg201 = msg("00009:11", all62);

var part333 = match("MESSAGE#200:00009:12", "nwparser.payload", "DHCP client has been %{disposition->} on interface %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg202 = msg("00009:12", part333);

var part334 = match("MESSAGE#201:00009:13", "nwparser.payload", "DHCP relay agent settings on %{interface->} have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg203 = msg("00009:13", part334);

var part335 = match("MESSAGE#202:00009:14/0_0", "nwparser.payload", "Global-PRO has been %{p0}");

var part336 = match("MESSAGE#202:00009:14/0_1", "nwparser.payload", "Global PRO has been %{p0}");

var part337 = match("MESSAGE#202:00009:14/0_2", "nwparser.payload", "DNS proxy was %{p0}");

var select74 = linear_select([
	part335,
	part336,
	part337,
]);

var part338 = match("MESSAGE#202:00009:14/1", "nwparser.p0", "%{disposition->} on %{p0}");

var select75 = linear_select([
	dup122,
	dup123,
]);

var part339 = match("MESSAGE#202:00009:14/4_0", "nwparser.p0", "%{interface->} (%{fld2})");

var select76 = linear_select([
	part339,
	dup124,
]);

var all63 = all_match({
	processors: [
		select74,
		part338,
		select75,
		dup23,
		select76,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg204 = msg("00009:14", all63);

var part340 = match("MESSAGE#203:00009:15/0", "nwparser.payload", "Route between secondary IP%{p0}");

var part341 = match("MESSAGE#203:00009:15/1_0", "nwparser.p0", " addresses %{p0}");

var select77 = linear_select([
	part341,
	dup125,
]);

var all64 = all_match({
	processors: [
		part340,
		select77,
		dup126,
		dup350,
		dup128,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg205 = msg("00009:15", all64);

var part342 = match("MESSAGE#204:00009:16/0", "nwparser.payload", "Secondary IP address %{hostip}/%{mask->} %{p0}");

var part343 = match("MESSAGE#204:00009:16/3_2", "nwparser.p0", "deleted from %{p0}");

var select78 = linear_select([
	dup129,
	dup130,
	part343,
]);

var part344 = match("MESSAGE#204:00009:16/4", "nwparser.p0", "interface %{interface}.");

var all65 = all_match({
	processors: [
		part342,
		dup350,
		dup23,
		select78,
		part344,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg206 = msg("00009:16", all65);

var part345 = match("MESSAGE#205:00009:17/0", "nwparser.payload", "Secondary IP address %{p0}");

var part346 = match("MESSAGE#205:00009:17/1_0", "nwparser.p0", "%{hostip}/%{mask->} was added to interface %{p0}");

var part347 = match("MESSAGE#205:00009:17/1_1", "nwparser.p0", "%{hostip->} was added to interface %{p0}");

var select79 = linear_select([
	part346,
	part347,
]);

var part348 = match("MESSAGE#205:00009:17/2", "nwparser.p0", "%{interface}.");

var all66 = all_match({
	processors: [
		part345,
		select79,
		part348,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg207 = msg("00009:17", all66);

var part349 = match("MESSAGE#206:00009:18", "nwparser.payload", "The configured bandwidth on the interface %{interface->} has been changed to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg208 = msg("00009:18", part349);

var part350 = match("MESSAGE#207:00009:19", "nwparser.payload", "interface %{interface->} with IP %{hostip->} %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg209 = msg("00009:19", part350);

var part351 = match("MESSAGE#208:00009:27", "nwparser.payload", "interface %{interface->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg210 = msg("00009:27", part351);

var part352 = match("MESSAGE#209:00009:20/0_0", "nwparser.payload", "%{fld2}: %{service->} has been %{p0}");

var part353 = match("MESSAGE#209:00009:20/0_1", "nwparser.payload", "%{service->} has been %{p0}");

var select80 = linear_select([
	part352,
	part353,
]);

var part354 = match("MESSAGE#209:00009:20/1", "nwparser.p0", "%{disposition->} on interface %{interface->} %{p0}");

var part355 = match("MESSAGE#209:00009:20/2_0", "nwparser.p0", "by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}");

var part356 = match("MESSAGE#209:00009:20/2_1", "nwparser.p0", "by %{username->} via %{logon_type->} from host %{saddr}:%{sport}");

var part357 = match("MESSAGE#209:00009:20/2_2", "nwparser.p0", "by %{username->} via %{logon_type->} from host %{saddr}");

var part358 = match("MESSAGE#209:00009:20/2_3", "nwparser.p0", "from host %{saddr->} (%{fld1})");

var select81 = linear_select([
	part355,
	part356,
	part357,
	part358,
]);

var all67 = all_match({
	processors: [
		select80,
		part354,
		select81,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg211 = msg("00009:20", all67);

var part359 = match("MESSAGE#210:00009:21/0", "nwparser.payload", "Source Route IP option! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone->} %{p0}");

var all68 = all_match({
	processors: [
		part359,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup9,
		dup60,
	]),
});

var msg212 = msg("00009:21", all68);

var part360 = match("MESSAGE#211:00009:22", "nwparser.payload", "MTU for interface %{interface->} has been changed to %{fld2->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg213 = msg("00009:22", part360);

var part361 = match("MESSAGE#212:00009:23", "nwparser.payload", "Secondary IP address %{hostip->} has been added to interface %{interface->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup9,
	dup3,
	dup4,
	dup5,
]));

var msg214 = msg("00009:23", part361);

var part362 = match("MESSAGE#213:00009:24/0", "nwparser.payload", "Web has been enabled on interface %{interface->} by admin %{administrator->} via %{p0}");

var part363 = match("MESSAGE#213:00009:24/1_0", "nwparser.p0", "%{logon_type->} %{space}(%{p0}");

var part364 = match("MESSAGE#213:00009:24/1_1", "nwparser.p0", "%{logon_type}. (%{p0}");

var select82 = linear_select([
	part363,
	part364,
]);

var part365 = match("MESSAGE#213:00009:24/2", "nwparser.p0", ")%{fld1}");

var all69 = all_match({
	processors: [
		part362,
		select82,
		part365,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup9,
		dup3,
		dup4,
		dup5,
	]),
});

var msg215 = msg("00009:24", all69);

var part366 = match("MESSAGE#214:00009:25", "nwparser.payload", "Web has been enabled on interface %{interface->} by %{username->} via %{logon_type}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup9,
	dup3,
	dup4,
	dup5,
]));

var msg216 = msg("00009:25", part366);

var part367 = match("MESSAGE#215:00009:26/0", "nwparser.payload", "%{protocol->} has been %{disposition->} on interface %{interface->} by %{username->} via NSRP Peer . %{p0}");

var all70 = all_match({
	processors: [
		part367,
		dup333,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup9,
		dup3,
		dup4,
		dup5,
	]),
});

var msg217 = msg("00009:26", all70);

var select83 = linear_select([
	msg192,
	msg193,
	msg194,
	msg195,
	msg196,
	msg197,
	msg198,
	msg199,
	msg200,
	msg201,
	msg202,
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
	msg214,
	msg215,
	msg216,
	msg217,
]);

var part368 = match("MESSAGE#216:00010/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport->} %{p0}");

var part369 = match("MESSAGE#216:00010/1_0", "nwparser.p0", "using protocol %{p0}");

var part370 = match("MESSAGE#216:00010/1_1", "nwparser.p0", "proto %{p0}");

var select84 = linear_select([
	part369,
	part370,
]);

var part371 = match("MESSAGE#216:00010/2", "nwparser.p0", "%{protocol->} %{p0}");

var part372 = match("MESSAGE#216:00010/3_0", "nwparser.p0", "( zone %{zone}, int %{interface}) %{p0}");

var part373 = match("MESSAGE#216:00010/3_1", "nwparser.p0", "zone %{zone->} int %{interface}) %{p0}");

var select85 = linear_select([
	part372,
	part373,
	dup126,
]);

var part374 = match("MESSAGE#216:00010/4", "nwparser.p0", ".%{space}The attack occurred %{dclass_counter1->} times%{p0}");

var all71 = all_match({
	processors: [
		part368,
		select84,
		part371,
		select85,
		part374,
		dup351,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup4,
		dup59,
		dup5,
		dup9,
		dup3,
		dup61,
	]),
});

var msg218 = msg("00010", all71);

var part375 = match("MESSAGE#217:00010:01", "nwparser.payload", "MIP %{hostip}/%{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg219 = msg("00010:01", part375);

var part376 = match("MESSAGE#218:00010:02", "nwparser.payload", "Mapped IP %{hostip->} %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg220 = msg("00010:02", part376);

var all72 = all_match({
	processors: [
		dup132,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup4,
		dup5,
		dup9,
		dup3,
		dup60,
	]),
});

var msg221 = msg("00010:03", all72);

var select86 = linear_select([
	msg218,
	msg219,
	msg220,
	msg221,
]);

var part377 = match("MESSAGE#220:00011", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg222 = msg("00011", part377);

var part378 = match("MESSAGE#221:00011:01/0", "nwparser.payload", "Route to %{daddr}/%{fld2->} [ %{p0}");

var select87 = linear_select([
	dup57,
	dup56,
]);

var part379 = match("MESSAGE#221:00011:01/2", "nwparser.p0", "%{} %{interface->} gateway %{fld3->} ] has been %{disposition}");

var all73 = all_match({
	processors: [
		part378,
		select87,
		part379,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg223 = msg("00011:01", all73);

var part380 = match("MESSAGE#222:00011:02", "nwparser.payload", "%{signame->} from %{saddr->} to %{daddr->} protocol %{protocol->} (%{fld2})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg224 = msg("00011:02", part380);

var part381 = match("MESSAGE#223:00011:03/0", "nwparser.payload", "An %{p0}");

var part382 = match("MESSAGE#223:00011:03/1_0", "nwparser.p0", "import %{p0}");

var part383 = match("MESSAGE#223:00011:03/1_1", "nwparser.p0", "export %{p0}");

var select88 = linear_select([
	part382,
	part383,
]);

var part384 = match("MESSAGE#223:00011:03/2", "nwparser.p0", "rule in virtual router %{node->} to virtual router %{fld4->} with %{p0}");

var part385 = match("MESSAGE#223:00011:03/3_0", "nwparser.p0", "route-map %{fld3->} and protocol %{protocol->} has been %{p0}");

var part386 = match("MESSAGE#223:00011:03/3_1", "nwparser.p0", "IP-prefix %{hostip}/%{interface->} has been %{p0}");

var select89 = linear_select([
	part385,
	part386,
]);

var all74 = all_match({
	processors: [
		part381,
		select88,
		part384,
		select89,
		dup36,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg225 = msg("00011:03", all74);

var part387 = match("MESSAGE#224:00011:04/0", "nwparser.payload", "A route in virtual router %{node->} that has IP address %{hostip}/%{fld2->} through %{p0}");

var part388 = match("MESSAGE#224:00011:04/2", "nwparser.p0", "%{interface->} and gateway %{fld3->} with metric %{fld4->} has been %{disposition}");

var all75 = all_match({
	processors: [
		part387,
		dup352,
		part388,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg226 = msg("00011:04", all75);

var part389 = match("MESSAGE#225:00011:05/1_0", "nwparser.p0", "sharable virtual router using name%{p0}");

var part390 = match("MESSAGE#225:00011:05/1_1", "nwparser.p0", "virtual router with name%{p0}");

var select90 = linear_select([
	part389,
	part390,
]);

var part391 = match("MESSAGE#225:00011:05/2", "nwparser.p0", "%{} %{node->} and id %{fld2->} has been %{disposition}");

var all76 = all_match({
	processors: [
		dup79,
		select90,
		part391,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg227 = msg("00011:05", all76);

var part392 = match("MESSAGE#226:00011:07", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup4,
	dup5,
	dup59,
	dup3,
	dup60,
]));

var msg228 = msg("00011:07", part392);

var part393 = match("MESSAGE#227:00011:08", "nwparser.payload", "Route(s) in virtual router %{node->} with an IP address %{hostip->} and gateway %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg229 = msg("00011:08", part393);

var part394 = match("MESSAGE#228:00011:09", "nwparser.payload", "The auto-route-export feature in virtual router %{node->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg230 = msg("00011:09", part394);

var part395 = match("MESSAGE#229:00011:10", "nwparser.payload", "The maximum number of routes that can be created in virtual router %{node->} is %{fld2}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg231 = msg("00011:10", part395);

var part396 = match("MESSAGE#230:00011:11", "nwparser.payload", "The maximum routes limit in virtual router %{node->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg232 = msg("00011:11", part396);

var part397 = match("MESSAGE#231:00011:12", "nwparser.payload", "The router-id of virtual router %{node->} used by OSPF BGP routing instances id has been uninitialized", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg233 = msg("00011:12", part397);

var part398 = match("MESSAGE#232:00011:13", "nwparser.payload", "The router-id that can be used by OSPF BGP routing instances in virtual router %{node->} has been set to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg234 = msg("00011:13", part398);

var part399 = match("MESSAGE#233:00011:14/0", "nwparser.payload", "The routing preference for protocol %{protocol->} in virtual router %{node->} has been %{p0}");

var part400 = match("MESSAGE#233:00011:14/1_1", "nwparser.p0", "reset%{}");

var select91 = linear_select([
	dup134,
	part400,
]);

var all77 = all_match({
	processors: [
		part399,
		select91,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg235 = msg("00011:14", all77);

var part401 = match("MESSAGE#234:00011:15", "nwparser.payload", "The system default-route in virtual router %{node->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg236 = msg("00011:15", part401);

var part402 = match("MESSAGE#235:00011:16", "nwparser.payload", "The system default-route through virtual router %{node->} has been added in virtual router %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg237 = msg("00011:16", part402);

var part403 = match("MESSAGE#236:00011:17/0", "nwparser.payload", "The virtual router %{node->} has been made %{p0}");

var part404 = match("MESSAGE#236:00011:17/1_0", "nwparser.p0", "sharable%{}");

var part405 = match("MESSAGE#236:00011:17/1_1", "nwparser.p0", "unsharable%{}");

var part406 = match("MESSAGE#236:00011:17/1_2", "nwparser.p0", "default virtual router for virtual system %{fld2}");

var select92 = linear_select([
	part404,
	part405,
	part406,
]);

var all78 = all_match({
	processors: [
		part403,
		select92,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg238 = msg("00011:17", all78);

var part407 = match("MESSAGE#237:00011:18/0_0", "nwparser.payload", "Source route(s) %{p0}");

var part408 = match("MESSAGE#237:00011:18/0_1", "nwparser.payload", "A source route %{p0}");

var select93 = linear_select([
	part407,
	part408,
]);

var part409 = match("MESSAGE#237:00011:18/1", "nwparser.p0", "in virtual router %{node->} %{p0}");

var part410 = match("MESSAGE#237:00011:18/2_0", "nwparser.p0", "with route addresses of %{p0}");

var part411 = match("MESSAGE#237:00011:18/2_1", "nwparser.p0", "that has IP address %{p0}");

var select94 = linear_select([
	part410,
	part411,
]);

var part412 = match("MESSAGE#237:00011:18/3", "nwparser.p0", "%{hostip}/%{fld2->} through interface %{interface->} and %{p0}");

var part413 = match("MESSAGE#237:00011:18/4_0", "nwparser.p0", "a default gateway address %{p0}");

var select95 = linear_select([
	part413,
	dup135,
]);

var part414 = match("MESSAGE#237:00011:18/5", "nwparser.p0", "%{fld3->} with metric %{fld4->} %{p0}");

var all79 = all_match({
	processors: [
		select93,
		part409,
		select94,
		part412,
		select95,
		part414,
		dup350,
		dup128,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg239 = msg("00011:18", all79);

var part415 = match("MESSAGE#238:00011:19/0", "nwparser.payload", "Source Route(s) in virtual router %{node->} with %{p0}");

var part416 = match("MESSAGE#238:00011:19/1_0", "nwparser.p0", "route addresses of %{p0}");

var part417 = match("MESSAGE#238:00011:19/1_1", "nwparser.p0", "an IP address %{p0}");

var select96 = linear_select([
	part416,
	part417,
]);

var part418 = match("MESSAGE#238:00011:19/2", "nwparser.p0", "%{hostip}/%{fld3->} and %{p0}");

var part419 = match("MESSAGE#238:00011:19/3_0", "nwparser.p0", "a default gateway address of %{p0}");

var select97 = linear_select([
	part419,
	dup135,
]);

var part420 = match("MESSAGE#238:00011:19/4", "nwparser.p0", "%{fld4->} %{p0}");

var part421 = match("MESSAGE#238:00011:19/5_1", "nwparser.p0", "has been%{p0}");

var select98 = linear_select([
	dup107,
	part421,
]);

var all80 = all_match({
	processors: [
		part415,
		select96,
		part418,
		select97,
		part420,
		select98,
		dup136,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg240 = msg("00011:19", all80);

var part422 = match("MESSAGE#239:00011:20/0_0", "nwparser.payload", "%{fld2}: A %{p0}");

var select99 = linear_select([
	part422,
	dup79,
]);

var part423 = match("MESSAGE#239:00011:20/1", "nwparser.p0", "route has been created in virtual router \"%{node}\"%{space}with an IP address %{hostip->} and next-hop as virtual router \"%{fld3}\"");

var all81 = all_match({
	processors: [
		select99,
		part423,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg241 = msg("00011:20", all81);

var part424 = match("MESSAGE#240:00011:21", "nwparser.payload", "SIBR route(s) in virtual router %{node->} for interface %{interface->} with an IP address %{hostip->} and gateway %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg242 = msg("00011:21", part424);

var part425 = match("MESSAGE#241:00011:22", "nwparser.payload", "SIBR route in virtual router %{node->} for interface %{interface->} that has IP address %{hostip->} through interface %{fld3->} and gateway %{fld4->} with metric %{fld5->} was %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg243 = msg("00011:22", part425);

var all82 = all_match({
	processors: [
		dup132,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup9,
		dup3,
		dup4,
		dup5,
		call({
			dest: "nwparser.inout",
			fn: DIRCHK,
			args: [
				field("$IN"),
				field("saddr"),
				field("daddr"),
			],
		}),
	]),
});

var msg244 = msg("00011:23", all82);

var part426 = match("MESSAGE#243:00011:24", "nwparser.payload", "Route in virtual router \"%{node}\" that has IP address %{hostip->} through interface %{interface->} and gateway %{fld2->} with metric %{fld3->} %{disposition}. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg245 = msg("00011:24", part426);

var part427 = match("MESSAGE#244:00011:25", "nwparser.payload", "Route(s) in virtual router \"%{node}\" with an IP address %{hostip}/%{fld2->} and gateway %{fld3->} %{disposition}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg246 = msg("00011:25", part427);

var part428 = match("MESSAGE#245:00011:26", "nwparser.payload", "Route in virtual router \"%{node}\" with IP address %{hostip}/%{fld2->} and next-hop as virtual router \"%{fld3}\" created. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg247 = msg("00011:26", part428);

var select100 = linear_select([
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
]);

var part429 = match("MESSAGE#246:00012:02", "nwparser.payload", "Service group %{group->} comments have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg248 = msg("00012:02", part429);

var part430 = match("MESSAGE#247:00012:03", "nwparser.payload", "Service group %{change_old->} %{change_attribute->} has been changed to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg249 = msg("00012:03", part430);

var part431 = match("MESSAGE#248:00012:04", "nwparser.payload", "%{fld2->} Service group %{group->} has %{disposition->} member %{username->} from host %{saddr}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg250 = msg("00012:04", part431);

var part432 = match("MESSAGE#249:00012:05", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol->} (%{fld2}) (%{fld3})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg251 = msg("00012:05", part432);

var part433 = match("MESSAGE#250:00012:06", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup59,
	dup61,
]));

var msg252 = msg("00012:06", part433);

var part434 = match("MESSAGE#251:00012:07", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
	dup59,
]));

var msg253 = msg("00012:07", part434);

var part435 = match("MESSAGE#252:00012:08", "nwparser.payload", "%{fld2}: Service %{service->} has been %{disposition->} from host %{saddr->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg254 = msg("00012:08", part435);

var all83 = all_match({
	processors: [
		dup80,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup9,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg255 = msg("00012:09", all83);

var all84 = all_match({
	processors: [
		dup132,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup9,
		dup59,
		dup3,
		dup4,
		dup5,
		dup60,
	]),
});

var msg256 = msg("00012:10", all84);

var part436 = match("MESSAGE#255:00012:11", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup9,
	dup61,
]));

var msg257 = msg("00012:11", part436);

var part437 = match("MESSAGE#256:00012:12", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol->} (%{zone}) %{info->} (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg258 = msg("00012:12", part437);

var part438 = match("MESSAGE#257:00012", "nwparser.payload", "Service group %{group->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg259 = msg("00012", part438);

var part439 = match("MESSAGE#258:00012:01", "nwparser.payload", "Service %{service->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg260 = msg("00012:01", part439);

var select101 = linear_select([
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
]);

var part440 = match("MESSAGE#259:00013", "nwparser.payload", "Global Manager error in decoding bytes has been detected%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg261 = msg("00013", part440);

var part441 = match("MESSAGE#260:00013:01", "nwparser.payload", "Intruder has attempted to connect to the NetScreen-Global Manager port! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} at interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
	setc("signame","An Attempt to connect to NetScreen-Global Manager Port."),
]));

var msg262 = msg("00013:01", part441);

var part442 = match("MESSAGE#261:00013:02", "nwparser.payload", "URL Filtering %{fld2->} has been changed to %{fld3}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg263 = msg("00013:02", part442);

var part443 = match("MESSAGE#262:00013:03", "nwparser.payload", "Web Filtering has been %{disposition->} (%{fld1})", processor_chain([
	dup50,
	dup43,
	dup51,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg264 = msg("00013:03", part443);

var select102 = linear_select([
	msg261,
	msg262,
	msg263,
	msg264,
]);

var part444 = match("MESSAGE#263:00014", "nwparser.payload", "%{change_attribute->} in minutes has changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg265 = msg("00014", part444);

var part445 = match("MESSAGE#264:00014:01/0", "nwparser.payload", "The group member %{username->} has been %{disposition->} %{p0}");

var part446 = match("MESSAGE#264:00014:01/1_0", "nwparser.p0", "to a group%{}");

var part447 = match("MESSAGE#264:00014:01/1_1", "nwparser.p0", "from a group%{}");

var select103 = linear_select([
	part446,
	part447,
]);

var all85 = all_match({
	processors: [
		part445,
		select103,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg266 = msg("00014:01", all85);

var part448 = match("MESSAGE#265:00014:02", "nwparser.payload", "The user group %{group->} has been %{disposition->} by %{username}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg267 = msg("00014:02", part448);

var part449 = match("MESSAGE#266:00014:03", "nwparser.payload", "The user %{username->} has been %{disposition->} by %{administrator}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg268 = msg("00014:03", part449);

var part450 = match("MESSAGE#267:00014:04", "nwparser.payload", "Communication error with %{hostname->} server { %{hostip->} }: SrvErr (%{fld2}), SockErr (%{fld3}), Valid (%{fld4}),Connected (%{fld5})", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg269 = msg("00014:04", part450);

var part451 = match("MESSAGE#268:00014:05", "nwparser.payload", "System clock configurations have been %{disposition->} by admin %{administrator}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg270 = msg("00014:05", part451);

var part452 = match("MESSAGE#269:00014:06", "nwparser.payload", "System clock is %{disposition->} manually.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg271 = msg("00014:06", part452);

var part453 = match("MESSAGE#270:00014:07", "nwparser.payload", "System up time is %{disposition->} by %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg272 = msg("00014:07", part453);

var part454 = match("MESSAGE#271:00014:08", "nwparser.payload", "Communication error with %{hostname->} server[%{hostip}]: SrvErr(%{fld2}),SockErr(%{fld3}),Valid(%{fld4}),Connected(%{fld5}) (%{fld1})", processor_chain([
	dup27,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg273 = msg("00014:08", part454);

var select104 = linear_select([
	msg265,
	msg266,
	msg267,
	msg268,
	msg269,
	msg270,
	msg271,
	msg272,
	msg273,
]);

var part455 = match("MESSAGE#272:00015", "nwparser.payload", "Authentication type has been changed to %{authmethod}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg274 = msg("00015", part455);

var part456 = match("MESSAGE#273:00015:01", "nwparser.payload", "IP tracking to %{daddr->} has %{disposition}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg275 = msg("00015:01", part456);

var part457 = match("MESSAGE#274:00015:02/0", "nwparser.payload", "LDAP %{p0}");

var part458 = match("MESSAGE#274:00015:02/1_0", "nwparser.p0", "server name %{p0}");

var part459 = match("MESSAGE#274:00015:02/1_2", "nwparser.p0", "distinguished name %{p0}");

var part460 = match("MESSAGE#274:00015:02/1_3", "nwparser.p0", "common name %{p0}");

var select105 = linear_select([
	part458,
	dup137,
	part459,
	part460,
]);

var all86 = all_match({
	processors: [
		part457,
		select105,
		dup138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg276 = msg("00015:02", all86);

var part461 = match("MESSAGE#275:00015:03", "nwparser.payload", "Primary HA link has gone down. Local NetScreen device has begun using the secondary HA link%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg277 = msg("00015:03", part461);

var part462 = match("MESSAGE#276:00015:04/0", "nwparser.payload", "RADIUS server %{p0}");

var part463 = match("MESSAGE#276:00015:04/1_2", "nwparser.p0", "secret %{p0}");

var select106 = linear_select([
	dup139,
	dup140,
	part463,
]);

var all87 = all_match({
	processors: [
		part462,
		select106,
		dup138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg278 = msg("00015:04", all87);

var part464 = match("MESSAGE#277:00015:05/0", "nwparser.payload", "SecurID %{p0}");

var part465 = match("MESSAGE#277:00015:05/1_0", "nwparser.p0", "authentication port %{p0}");

var part466 = match("MESSAGE#277:00015:05/1_1", "nwparser.p0", "duress mode %{p0}");

var part467 = match("MESSAGE#277:00015:05/1_3", "nwparser.p0", "number of retries value %{p0}");

var select107 = linear_select([
	part465,
	part466,
	dup76,
	part467,
]);

var all88 = all_match({
	processors: [
		part464,
		select107,
		dup138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg279 = msg("00015:05", all88);

var part468 = match("MESSAGE#278:00015:06/0_0", "nwparser.payload", "Master %{p0}");

var part469 = match("MESSAGE#278:00015:06/0_1", "nwparser.payload", "Backup %{p0}");

var select108 = linear_select([
	part468,
	part469,
]);

var part470 = match("MESSAGE#278:00015:06/1", "nwparser.p0", "SecurID server IP address has been %{disposition}");

var all89 = all_match({
	processors: [
		select108,
		part470,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg280 = msg("00015:06", all89);

var part471 = match("MESSAGE#279:00015:07", "nwparser.payload", "HA change from slave to master%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg281 = msg("00015:07", part471);

var part472 = match("MESSAGE#280:00015:08", "nwparser.payload", "inconsistent configuration between master and slave%{}", processor_chain([
	dup141,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg282 = msg("00015:08", part472);

var part473 = match("MESSAGE#281:00015:09/0_0", "nwparser.payload", "configuration %{p0}");

var part474 = match("MESSAGE#281:00015:09/0_1", "nwparser.payload", "Configuration %{p0}");

var select109 = linear_select([
	part473,
	part474,
]);

var part475 = match("MESSAGE#281:00015:09/1", "nwparser.p0", "out of sync between local unit and remote unit%{}");

var all90 = all_match({
	processors: [
		select109,
		part475,
	],
	on_success: processor_chain([
		dup141,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg283 = msg("00015:09", all90);

var part476 = match("MESSAGE#282:00015:10", "nwparser.payload", "HA control channel change to %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg284 = msg("00015:10", part476);

var part477 = match("MESSAGE#283:00015:11", "nwparser.payload", "HA data channel change to %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg285 = msg("00015:11", part477);

var part478 = match("MESSAGE#284:00015:12/1_0", "nwparser.p0", "control %{p0}");

var part479 = match("MESSAGE#284:00015:12/1_1", "nwparser.p0", "data %{p0}");

var select110 = linear_select([
	part478,
	part479,
]);

var part480 = match("MESSAGE#284:00015:12/2", "nwparser.p0", "channel moved from link %{p0}");

var part481 = match("MESSAGE#284:00015:12/6", "nwparser.p0", "(%{interface})");

var all91 = all_match({
	processors: [
		dup87,
		select110,
		part480,
		dup353,
		dup103,
		dup353,
		part481,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg286 = msg("00015:12", all91);

var part482 = match("MESSAGE#285:00015:13", "nwparser.payload", "HA: Slave is down%{}", processor_chain([
	dup144,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg287 = msg("00015:13", part482);

var part483 = match("MESSAGE#286:00015:14/0", "nwparser.payload", "NSRP link %{p0}");

var all92 = all_match({
	processors: [
		part483,
		dup353,
		dup116,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg288 = msg("00015:14", all92);

var part484 = match("MESSAGE#287:00015:15", "nwparser.payload", "no HA %{fld2->} channel available (%{fld3->} used by other channel)", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg289 = msg("00015:15", part484);

var part485 = match("MESSAGE#288:00015:16", "nwparser.payload", "The NSRP configuration is out of synchronization between the local device and the peer device.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg290 = msg("00015:16", part485);

var part486 = match("MESSAGE#289:00015:17", "nwparser.payload", "NSRP %{change_attribute->} %{change_old->} changed to link channel %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg291 = msg("00015:17", part486);

var part487 = match("MESSAGE#290:00015:18", "nwparser.payload", "RTO mirror group %{group->} with direction %{direction->} on peer device %{fld2->} changed from %{fld3->} to %{fld4->} state.", processor_chain([
	dup121,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("change_attribute","RTO mirror group"),
]));

var msg292 = msg("00015:18", part487);

var part488 = match("MESSAGE#291:00015:19", "nwparser.payload", "RTO mirror group %{group->} with direction %{direction->} on local device %{fld2}, detected a duplicate direction on the peer device %{fld3}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg293 = msg("00015:19", part488);

var part489 = match("MESSAGE#292:00015:20", "nwparser.payload", "RTO mirror group %{group->} with direction %{direction->} changed on the local device from %{fld2->} to up state, it had peer device %{fld3}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg294 = msg("00015:20", part489);

var part490 = match("MESSAGE#293:00015:21/0", "nwparser.payload", "Peer device %{fld2->} %{p0}");

var part491 = match("MESSAGE#293:00015:21/1_0", "nwparser.p0", "disappeared %{p0}");

var part492 = match("MESSAGE#293:00015:21/1_1", "nwparser.p0", "was discovered %{p0}");

var select111 = linear_select([
	part491,
	part492,
]);

var all93 = all_match({
	processors: [
		part490,
		select111,
		dup116,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg295 = msg("00015:21", all93);

var part493 = match("MESSAGE#294:00015:22/0_0", "nwparser.payload", "The local %{p0}");

var part494 = match("MESSAGE#294:00015:22/0_1", "nwparser.payload", "The peer %{p0}");

var part495 = match("MESSAGE#294:00015:22/0_2", "nwparser.payload", "Peer %{p0}");

var select112 = linear_select([
	part493,
	part494,
	part495,
]);

var part496 = match("MESSAGE#294:00015:22/1", "nwparser.p0", "device %{fld2->} in the Virtual Security Device group %{group->} changed %{change_attribute->} from %{change_old->} to %{change_new->} %{p0}");

var all94 = all_match({
	processors: [
		select112,
		part496,
		dup354,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg296 = msg("00015:22", all94);

var part497 = match("MESSAGE#295:00015:23", "nwparser.payload", "WebAuth is set to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg297 = msg("00015:23", part497);

var part498 = match("MESSAGE#296:00015:24", "nwparser.payload", "Default firewall authentication server has been changed to %{hostname}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg298 = msg("00015:24", part498);

var part499 = match("MESSAGE#297:00015:25", "nwparser.payload", "Admin user %{administrator->} attempted to verify the encrypted password %{fld2}. Verification was successful", processor_chain([
	setc("eventcategory","1613050100"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg299 = msg("00015:25", part499);

var part500 = match("MESSAGE#298:00015:29", "nwparser.payload", "Admin user %{administrator->} attempted to verify the encrypted password %{fld2}. Verification failed", processor_chain([
	dup97,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg300 = msg("00015:29", part500);

var part501 = match("MESSAGE#299:00015:26/0", "nwparser.payload", "unit %{fld2->} just dis%{p0}");

var part502 = match("MESSAGE#299:00015:26/1_0", "nwparser.p0", "appeared%{}");

var part503 = match("MESSAGE#299:00015:26/1_1", "nwparser.p0", "covered%{}");

var select113 = linear_select([
	part502,
	part503,
]);

var all95 = all_match({
	processors: [
		part501,
		select113,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg301 = msg("00015:26", all95);

var part504 = match("MESSAGE#300:00015:33", "nwparser.payload", "NSRP: HA data channel change to %{interface}. (%{fld2})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
	dup146,
]));

var msg302 = msg("00015:33", part504);

var part505 = match("MESSAGE#301:00015:27", "nwparser.payload", "NSRP: %{fld2}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg303 = msg("00015:27", part505);

var part506 = match("MESSAGE#302:00015:28", "nwparser.payload", "Auth server %{hostname->} RADIUS retry timeout has been set to default of %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg304 = msg("00015:28", part506);

var part507 = match("MESSAGE#303:00015:30/0", "nwparser.payload", "Number of RADIUS retries for auth server %{hostname->} %{p0}");

var part508 = match("MESSAGE#303:00015:30/2", "nwparser.p0", "set to %{fld2->} (%{fld1})");

var all96 = all_match({
	processors: [
		part507,
		dup355,
		part508,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg305 = msg("00015:30", all96);

var part509 = match("MESSAGE#304:00015:31", "nwparser.payload", "Forced timeout for Auth server %{hostname->} is unset to its default value, %{info->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg306 = msg("00015:31", part509);

var part510 = match("MESSAGE#305:00015:32", "nwparser.payload", "Accounting port of server RADIUS is set to %{network_port}. (%{fld1})", processor_chain([
	dup50,
	dup43,
	dup51,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg307 = msg("00015:32", part510);

var select114 = linear_select([
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
	msg286,
	msg287,
	msg288,
	msg289,
	msg290,
	msg291,
	msg292,
	msg293,
	msg294,
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
]);

var part511 = match("MESSAGE#306:00016", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup147,
	dup148,
	dup149,
	dup150,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg308 = msg("00016", part511);

var part512 = match("MESSAGE#307:00016:01", "nwparser.payload", "Address VIP (%{fld2}) for %{fld3->} has been %{disposition}.", processor_chain([
	dup1,
	dup148,
	dup149,
	dup150,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg309 = msg("00016:01", part512);

var part513 = match("MESSAGE#308:00016:02", "nwparser.payload", "VIP (%{fld2}) has been %{disposition}", processor_chain([
	dup1,
	dup148,
	dup149,
	dup150,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg310 = msg("00016:02", part513);

var part514 = match("MESSAGE#309:00016:03", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol->} (%{fld2})", processor_chain([
	dup147,
	dup148,
	dup149,
	dup150,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg311 = msg("00016:03", part514);

var part515 = match("MESSAGE#310:00016:05", "nwparser.payload", "VIP multi-port was %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg312 = msg("00016:05", part515);

var part516 = match("MESSAGE#311:00016:06", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup147,
	dup148,
	dup149,
	dup150,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg313 = msg("00016:06", part516);

var part517 = match("MESSAGE#312:00016:07/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} ( zone %{p0}");

var all97 = all_match({
	processors: [
		part517,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup147,
		dup148,
		dup149,
		dup150,
		dup2,
		dup9,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg314 = msg("00016:07", all97);

var part518 = match("MESSAGE#313:00016:08", "nwparser.payload", "VIP (%{fld2}:%{fld3->} HTTP %{fld4}) Modify by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	setc("eventcategory","1001020305"),
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg315 = msg("00016:08", part518);

var part519 = match("MESSAGE#314:00016:09", "nwparser.payload", "VIP (%{fld2}:%{fld3->} HTTP %{fld4}) New by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	setc("eventcategory","1001030305"),
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg316 = msg("00016:09", part519);

var select115 = linear_select([
	msg308,
	msg309,
	msg310,
	msg311,
	msg312,
	msg313,
	msg314,
	msg315,
	msg316,
]);

var part520 = match("MESSAGE#315:00017", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup151,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
]));

var msg317 = msg("00017", part520);

var part521 = match("MESSAGE#316:00017:23/0", "nwparser.payload", "Gateway %{fld2->} at %{fld3->} in %{fld5->} mode with ID %{p0}");

var part522 = match("MESSAGE#316:00017:23/1_0", "nwparser.p0", "[%{fld4}] %{p0}");

var part523 = match("MESSAGE#316:00017:23/1_1", "nwparser.p0", "%{fld4->} %{p0}");

var select116 = linear_select([
	part522,
	part523,
]);

var part524 = match("MESSAGE#316:00017:23/2", "nwparser.p0", "has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} %{fld}");

var all98 = all_match({
	processors: [
		part521,
		select116,
		part524,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg318 = msg("00017:23", all98);

var part525 = match("MESSAGE#317:00017:01/0_0", "nwparser.payload", "%{fld1}: Gateway %{p0}");

var part526 = match("MESSAGE#317:00017:01/0_1", "nwparser.payload", "Gateway %{p0}");

var select117 = linear_select([
	part525,
	part526,
]);

var part527 = match("MESSAGE#317:00017:01/1", "nwparser.p0", "%{fld2->} at %{fld3->} in %{fld5->} mode with ID%{p0}");

var part528 = match("MESSAGE#317:00017:01/3", "nwparser.p0", "%{fld4->} has been %{disposition}");

var all99 = all_match({
	processors: [
		select117,
		part527,
		dup356,
		part528,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg319 = msg("00017:01", all99);

var part529 = match("MESSAGE#318:00017:02", "nwparser.payload", "IKE %{hostip}: Gateway settings have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg320 = msg("00017:02", part529);

var part530 = match("MESSAGE#319:00017:03", "nwparser.payload", "IKE key %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg321 = msg("00017:03", part530);

var part531 = match("MESSAGE#320:00017:04/2", "nwparser.p0", "%{group_object->} with range %{fld2->} has been %{disposition}");

var all100 = all_match({
	processors: [
		dup153,
		dup357,
		part531,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg322 = msg("00017:04", all100);

var part532 = match("MESSAGE#321:00017:05", "nwparser.payload", "IPSec NAT-T for VPN %{group->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg323 = msg("00017:05", part532);

var part533 = match("MESSAGE#322:00017:06/0", "nwparser.payload", "The DF-BIT for VPN %{group->} has been set to %{p0}");

var part534 = match("MESSAGE#322:00017:06/1_0", "nwparser.p0", "clear %{p0}");

var part535 = match("MESSAGE#322:00017:06/1_2", "nwparser.p0", "copy %{p0}");

var select118 = linear_select([
	part534,
	dup101,
	part535,
]);

var all101 = all_match({
	processors: [
		part533,
		select118,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg324 = msg("00017:06", all101);

var part536 = match("MESSAGE#323:00017:07/0", "nwparser.payload", "The DF-BIT for VPN %{group->} has been %{p0}");

var part537 = match("MESSAGE#323:00017:07/1_0", "nwparser.p0", "clear%{}");

var part538 = match("MESSAGE#323:00017:07/1_1", "nwparser.p0", "cleared%{}");

var part539 = match("MESSAGE#323:00017:07/1_3", "nwparser.p0", "copy%{}");

var part540 = match("MESSAGE#323:00017:07/1_4", "nwparser.p0", "copied%{}");

var select119 = linear_select([
	part537,
	part538,
	dup98,
	part539,
	part540,
]);

var all102 = all_match({
	processors: [
		part536,
		select119,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg325 = msg("00017:07", all102);

var part541 = match("MESSAGE#324:00017:08", "nwparser.payload", "VPN %{group->} with gateway %{fld2->} and SPI %{fld3}/%{fld4->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg326 = msg("00017:08", part541);

var part542 = match("MESSAGE#325:00017:09/0_0", "nwparser.payload", "%{fld1}: VPN %{p0}");

var part543 = match("MESSAGE#325:00017:09/0_1", "nwparser.payload", "VPN %{p0}");

var select120 = linear_select([
	part542,
	part543,
]);

var part544 = match("MESSAGE#325:00017:09/1", "nwparser.p0", "%{group->} with gateway %{fld2->} %{p0}");

var part545 = match("MESSAGE#325:00017:09/2_0", "nwparser.p0", "no-rekey %{p0}");

var part546 = match("MESSAGE#325:00017:09/2_1", "nwparser.p0", "rekey, %{p0}");

var part547 = match("MESSAGE#325:00017:09/2_2", "nwparser.p0", "rekey %{p0}");

var select121 = linear_select([
	part545,
	part546,
	part547,
]);

var part548 = match("MESSAGE#325:00017:09/3", "nwparser.p0", "and p2-proposal %{fld3->} has been %{p0}");

var part549 = match("MESSAGE#325:00017:09/4_0", "nwparser.p0", "%{disposition->} from peer unit");

var part550 = match("MESSAGE#325:00017:09/4_1", "nwparser.p0", "%{disposition->} from host %{saddr}");

var select122 = linear_select([
	part549,
	part550,
	dup36,
]);

var all103 = all_match({
	processors: [
		select120,
		part544,
		select121,
		part548,
		select122,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg327 = msg("00017:09", all103);

var part551 = match("MESSAGE#326:00017:10/0", "nwparser.payload", "VPN monitoring for VPN %{group->} has been %{disposition}. Src IF %{sinterface->} dst IP %{daddr->} with rekeying %{p0}");

var all104 = all_match({
	processors: [
		part551,
		dup358,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg328 = msg("00017:10", all104);

var part552 = match("MESSAGE#327:00017:11", "nwparser.payload", "VPN monitoring for VPN %{group->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg329 = msg("00017:11", part552);

var part553 = match("MESSAGE#328:00017:12/0", "nwparser.payload", "VPN monitoring %{p0}");

var part554 = match("MESSAGE#328:00017:12/1_2", "nwparser.p0", "frequency %{p0}");

var select123 = linear_select([
	dup109,
	dup110,
	part554,
]);

var all105 = all_match({
	processors: [
		part553,
		select123,
		dup127,
		dup359,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg330 = msg("00017:12", all105);

var part555 = match("MESSAGE#329:00017:26", "nwparser.payload", "VPN %{group->} with gateway %{fld2->} and P2 proposal %{fld3->} has been added by %{username->} via %{logon_type->} from host %{saddr}:%{sport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg331 = msg("00017:26", part555);

var part556 = match("MESSAGE#330:00017:13", "nwparser.payload", "No IP pool has been assigned. You cannot allocate an IP address.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg332 = msg("00017:13", part556);

var part557 = match("MESSAGE#331:00017:14", "nwparser.payload", "P1 proposal %{fld2->} with %{protocol_detail}, DH group %{group}, ESP %{encryption_type}, auth %{authmethod}, and lifetime %{fld3->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup9,
	dup5,
]));

var msg333 = msg("00017:14", part557);

var part558 = match("MESSAGE#332:00017:15/0", "nwparser.payload", "P2 proposal %{fld2->} with DH group %{group->} %{p0}");

var part559 = match("MESSAGE#332:00017:15/2", "nwparser.p0", "%{encryption_type->} auth %{authmethod->} and lifetime (%{fld3}) (%{fld4}) has been %{disposition}.");

var all106 = all_match({
	processors: [
		part558,
		dup360,
		part559,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg334 = msg("00017:15", all106);

var part560 = match("MESSAGE#333:00017:31/0", "nwparser.payload", "P1 proposal %{fld2->} with %{protocol_detail->} DH group %{group->} %{p0}");

var part561 = match("MESSAGE#333:00017:31/2", "nwparser.p0", "%{encryption_type->} auth %{authmethod->} and lifetime %{fld3->} has been %{disposition}.");

var all107 = all_match({
	processors: [
		part560,
		dup360,
		part561,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg335 = msg("00017:31", all107);

var part562 = match("MESSAGE#334:00017:16/0", "nwparser.payload", "vpnmonitor interval is %{p0}");

var all108 = all_match({
	processors: [
		part562,
		dup359,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg336 = msg("00017:16", all108);

var part563 = match("MESSAGE#335:00017:17/0", "nwparser.payload", "vpnmonitor threshold is %{p0}");

var select124 = linear_select([
	dup99,
	dup93,
]);

var all109 = all_match({
	processors: [
		part563,
		select124,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg337 = msg("00017:17", all109);

var part564 = match("MESSAGE#336:00017:18/2", "nwparser.p0", "%{group_object->} with range %{fld2->} was %{disposition}");

var all110 = all_match({
	processors: [
		dup153,
		dup357,
		part564,
	],
	on_success: processor_chain([
		dup50,
		dup43,
		dup51,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg338 = msg("00017:18", all110);

var part565 = match("MESSAGE#337:00017:19/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at %{p0}");

var part566 = match("MESSAGE#337:00017:19/2", "nwparser.p0", "%{} %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times");

var all111 = all_match({
	processors: [
		part565,
		dup337,
		part566,
	],
	on_success: processor_chain([
		dup151,
		dup2,
		dup3,
		dup59,
		dup4,
		dup5,
	]),
});

var msg339 = msg("00017:19", all111);

var all112 = all_match({
	processors: [
		dup64,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup151,
		dup2,
		dup9,
		dup59,
		dup3,
		dup4,
		dup5,
	]),
});

var msg340 = msg("00017:20", all112);

var part567 = match("MESSAGE#339:00017:21", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup151,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
]));

var msg341 = msg("00017:21", part567);

var part568 = match("MESSAGE#340:00017:22", "nwparser.payload", "VPN %{group->} with gateway %{fld2->} and P2 proposal %{fld3->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg342 = msg("00017:22", part568);

var part569 = match("MESSAGE#341:00017:24", "nwparser.payload", "VPN \"%{group}\" has been bound to tunnel interface %{interface}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg343 = msg("00017:24", part569);

var part570 = match("MESSAGE#342:00017:25", "nwparser.payload", "VPN %{group->} with gateway %{fld2->} and P2 proposal standard has been added by admin %{administrator->} via NSRP Peer (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg344 = msg("00017:25", part570);

var part571 = match("MESSAGE#343:00017:28", "nwparser.payload", "P2 proposal %{fld2->} with DH group %{group}, ESP, enc %{encryption_type}, auth %{authmethod}, and lifetime %{fld3->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg345 = msg("00017:28", part571);

var part572 = match("MESSAGE#344:00017:29", "nwparser.payload", "L2TP \"%{fld2}\", all-L2TP-users secret \"%{fld3}\" keepalive %{fld4->} has been %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg346 = msg("00017:29", part572);

var select125 = linear_select([
	msg317,
	msg318,
	msg319,
	msg320,
	msg321,
	msg322,
	msg323,
	msg324,
	msg325,
	msg326,
	msg327,
	msg328,
	msg329,
	msg330,
	msg331,
	msg332,
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
	msg346,
]);

var part573 = match("MESSAGE#345:00018", "nwparser.payload", "Positions of policies %{fld2->} and %{fld3->} have been exchanged", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg347 = msg("00018", part573);

var part574 = match("MESSAGE#346:00018:01", "nwparser.payload", "Deny Policy Alarm%{}", processor_chain([
	setc("eventcategory","1502010000"),
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg348 = msg("00018:01", part574);

var part575 = match("MESSAGE#347:00018:02", "nwparser.payload", "Device%{quote}s %{change_attribute->} has been changed from %{change_old->} to %{change_new->} by admin %{administrator}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg349 = msg("00018:02", part575);

var part576 = match("MESSAGE#348:00018:04", "nwparser.payload", "%{fld2->} Policy (%{policy_id}, %{info->} ) was %{disposition->} from host %{saddr->} by admin %{administrator->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg350 = msg("00018:04", part576);

var part577 = match("MESSAGE#349:00018:16", "nwparser.payload", "%{fld2->} Policy (%{policy_id}, %{info->} ) was %{disposition->} by admin %{administrator->} via NSRP Peer", processor_chain([
	dup17,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg351 = msg("00018:16", part577);

var part578 = match("MESSAGE#350:00018:06/0", "nwparser.payload", "%{fld2->} Policy %{policy_id->} has been moved %{p0}");

var part579 = match("MESSAGE#350:00018:06/1_0", "nwparser.p0", "before %{p0}");

var part580 = match("MESSAGE#350:00018:06/1_1", "nwparser.p0", "after %{p0}");

var select126 = linear_select([
	part579,
	part580,
]);

var part581 = match("MESSAGE#350:00018:06/2", "nwparser.p0", "%{fld3->} by admin %{administrator}");

var all113 = all_match({
	processors: [
		part578,
		select126,
		part581,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg352 = msg("00018:06", all113);

var part582 = match("MESSAGE#351:00018:08", "nwparser.payload", "Policy %{policy_id->} application was modified to %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg353 = msg("00018:08", part582);

var part583 = match("MESSAGE#352:00018:09", "nwparser.payload", "Policy (%{policy_id}, %{info}) was %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup17,
	dup3,
	dup2,
	dup9,
	dup4,
	dup5,
]));

var msg354 = msg("00018:09", part583);

var part584 = match("MESSAGE#353:00018:10/0", "nwparser.payload", "Policy (%{policy_id}, %{info}) was %{p0}");

var part585 = match("MESSAGE#353:00018:10/1_0", "nwparser.p0", "%{disposition->} from peer unit by %{p0}");

var part586 = match("MESSAGE#353:00018:10/1_1", "nwparser.p0", "%{disposition->} by %{p0}");

var select127 = linear_select([
	part585,
	part586,
]);

var part587 = match("MESSAGE#353:00018:10/2", "nwparser.p0", "%{username->} via %{interface->} from host %{saddr->} (%{fld1})");

var all114 = all_match({
	processors: [
		part584,
		select127,
		part587,
	],
	on_success: processor_chain([
		dup17,
		dup3,
		dup2,
		dup9,
		dup4,
		dup5,
	]),
});

var msg355 = msg("00018:10", all114);

var part588 = match("MESSAGE#354:00018:11/1_0", "nwparser.p0", "Service %{service->} was %{p0}");

var part589 = match("MESSAGE#354:00018:11/1_1", "nwparser.p0", "Attack group %{signame->} was %{p0}");

var select128 = linear_select([
	part588,
	part589,
]);

var part590 = match("MESSAGE#354:00018:11/2", "nwparser.p0", "%{disposition->} to policy ID %{policy_id->} by %{username->} via %{logon_type->} from host %{saddr->} %{p0}");

var part591 = match("MESSAGE#354:00018:11/3_0", "nwparser.p0", "to %{daddr}:%{dport}. %{p0}");

var select129 = linear_select([
	part591,
	dup16,
]);

var all115 = all_match({
	processors: [
		dup160,
		select128,
		part590,
		select129,
		dup10,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg356 = msg("00018:11", all115);

var part592 = match("MESSAGE#355:00018:12/0", "nwparser.payload", "In policy %{policy_id}, the %{p0}");

var part593 = match("MESSAGE#355:00018:12/1_0", "nwparser.p0", "application %{p0}");

var part594 = match("MESSAGE#355:00018:12/1_1", "nwparser.p0", "attack severity %{p0}");

var part595 = match("MESSAGE#355:00018:12/1_2", "nwparser.p0", "DI attack component %{p0}");

var select130 = linear_select([
	part593,
	part594,
	part595,
]);

var part596 = match("MESSAGE#355:00018:12/2", "nwparser.p0", "was modified by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})");

var all116 = all_match({
	processors: [
		part592,
		select130,
		part596,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg357 = msg("00018:12", all116);

var part597 = match("MESSAGE#356:00018:32/1", "nwparser.p0", "%{}address %{dhost}(%{daddr}) was %{disposition->} %{p0}");

var all117 = all_match({
	processors: [
		dup361,
		part597,
		dup362,
		dup164,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg358 = msg("00018:32", all117);

var part598 = match("MESSAGE#357:00018:22/1", "nwparser.p0", "%{}address %{dhost->} was %{disposition->} %{p0}");

var all118 = all_match({
	processors: [
		dup361,
		part598,
		dup362,
		dup164,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg359 = msg("00018:22", all118);

var part599 = match("MESSAGE#358:00018:15/0", "nwparser.payload", "%{agent->} was %{disposition->} from policy %{policy_id->} %{p0}");

var select131 = linear_select([
	dup78,
	dup77,
]);

var part600 = match("MESSAGE#358:00018:15/2", "nwparser.p0", "address by admin %{administrator->} via NSRP Peer");

var all119 = all_match({
	processors: [
		part599,
		select131,
		part600,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg360 = msg("00018:15", all119);

var part601 = match("MESSAGE#359:00018:14/0", "nwparser.payload", "%{agent->} was %{disposition->} %{p0}");

var part602 = match("MESSAGE#359:00018:14/1_0", "nwparser.p0", "to%{p0}");

var part603 = match("MESSAGE#359:00018:14/1_1", "nwparser.p0", "from%{p0}");

var select132 = linear_select([
	part602,
	part603,
]);

var part604 = match("MESSAGE#359:00018:14/2", "nwparser.p0", "%{}policy %{policy_id->} %{p0}");

var part605 = match("MESSAGE#359:00018:14/3_0", "nwparser.p0", "service %{p0}");

var part606 = match("MESSAGE#359:00018:14/3_1", "nwparser.p0", "source address %{p0}");

var part607 = match("MESSAGE#359:00018:14/3_2", "nwparser.p0", "destination address %{p0}");

var select133 = linear_select([
	part605,
	part606,
	part607,
]);

var part608 = match("MESSAGE#359:00018:14/4", "nwparser.p0", "by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})");

var all120 = all_match({
	processors: [
		part601,
		select132,
		part604,
		select133,
		part608,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg361 = msg("00018:14", all120);

var part609 = match("MESSAGE#360:00018:29", "nwparser.payload", "Service %{service->} was %{disposition->} to policy ID %{policy_id->} by admin %{administrator->} via NSRP Peer . (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg362 = msg("00018:29", part609);

var part610 = match("MESSAGE#361:00018:07", "nwparser.payload", "%{agent->} was added to policy %{policy_id->} %{rule_group->} by admin %{administrator->} via NSRP Peer %{space->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg363 = msg("00018:07", part610);

var part611 = match("MESSAGE#362:00018:18", "nwparser.payload", "Service %{service->} was %{disposition->} to policy ID %{policy_id->} by %{username->} via %{logon_type->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg364 = msg("00018:18", part611);

var part612 = match("MESSAGE#363:00018:17", "nwparser.payload", "AntiSpam ns-profile was %{disposition->} from policy ID %{policy_id->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg365 = msg("00018:17", part612);

var part613 = match("MESSAGE#364:00018:19", "nwparser.payload", "Source address Info %{info->} was %{disposition->} to policy ID %{policy_id->} by %{username->} via %{logon_type->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg366 = msg("00018:19", part613);

var part614 = match("MESSAGE#365:00018:23/0_0", "nwparser.payload", "Destination %{p0}");

var part615 = match("MESSAGE#365:00018:23/0_1", "nwparser.payload", "Source %{p0}");

var select134 = linear_select([
	part614,
	part615,
]);

var part616 = match("MESSAGE#365:00018:23/1", "nwparser.p0", "address %{info->} was added to policy ID %{policy_id->} by %{username->} via %{logon_type->} %{p0}");

var part617 = match("MESSAGE#365:00018:23/2_0", "nwparser.p0", "from host %{p0}");

var select135 = linear_select([
	part617,
	dup103,
]);

var part618 = match("MESSAGE#365:00018:23/4_0", "nwparser.p0", "%{saddr->} to %{daddr->} %{p0}");

var part619 = match("MESSAGE#365:00018:23/4_1", "nwparser.p0", "%{daddr->} %{p0}");

var select136 = linear_select([
	part618,
	part619,
]);

var part620 = match("MESSAGE#365:00018:23/5", "nwparser.p0", "%{dport}:(%{fld1})");

var all121 = all_match({
	processors: [
		select134,
		part616,
		select135,
		dup23,
		select136,
		part620,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg367 = msg("00018:23", all121);

var part621 = match("MESSAGE#366:00018:21", "nwparser.payload", "Service %{service->} was deleted from policy ID %{policy_id->} by %{username->} via %{logon_type->} from host %{saddr}:%{sport}. (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg368 = msg("00018:21", part621);

var part622 = match("MESSAGE#367:00018:24", "nwparser.payload", "Policy (%{policyname}) was %{disposition->} by %{username->} via %{logon_type->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg369 = msg("00018:24", part622);

var part623 = match("MESSAGE#368:00018:25/1", "nwparser.p0", "%{}address %{info->} was added to policy ID %{policy_id->} by %{username->} via %{logon_type->} from host %{saddr}. (%{fld1})");

var all122 = all_match({
	processors: [
		dup363,
		part623,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg370 = msg("00018:25", all122);

var part624 = match("MESSAGE#369:00018:30/1", "nwparser.p0", "%{}address %{info->} was deleted from policy ID %{policy_id->} by %{username->} via %{logon_type->} from host %{saddr}. (%{fld1})");

var all123 = all_match({
	processors: [
		dup363,
		part624,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg371 = msg("00018:30", all123);

var part625 = match("MESSAGE#370:00018:26/0", "nwparser.payload", "In policy %{policy_id}, the application was modified to %{disposition->} by %{p0}");

var part626 = match("MESSAGE#370:00018:26/2_1", "nwparser.p0", "%{logon_type->} from host %{saddr}. (%{p0}");

var select137 = linear_select([
	dup48,
	part626,
]);

var all124 = all_match({
	processors: [
		part625,
		dup364,
		select137,
		dup41,
	],
	on_success: processor_chain([
		dup17,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg372 = msg("00018:26", all124);

var part627 = match("MESSAGE#371:00018:27", "nwparser.payload", "In policy %{policy_id}, the DI attack component was modified by %{username->} via %{logon_type->} from host %{saddr}:%{sport}. (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg373 = msg("00018:27", part627);

var part628 = match("MESSAGE#372:00018:28", "nwparser.payload", "In policy %{policyname}, the DI attack component was modified by admin %{administrator->} via %{logon_type}. (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup4,
	dup5,
	dup9,
	setc("info","the DI attack component was modified"),
]));

var msg374 = msg("00018:28", part628);

var part629 = match("MESSAGE#373:00018:03", "nwparser.payload", "Policy (%{policy_id}, %{info}) was %{disposition}", processor_chain([
	dup17,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg375 = msg("00018:03", part629);

var part630 = match("MESSAGE#1213:00018:31", "nwparser.payload", "In policy %{policy_id}, the option %{fld2->} was %{disposition}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg376 = msg("00018:31", part630);

var select138 = linear_select([
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
	msg359,
	msg360,
	msg361,
	msg362,
	msg363,
	msg364,
	msg365,
	msg366,
	msg367,
	msg368,
	msg369,
	msg370,
	msg371,
	msg372,
	msg373,
	msg374,
	msg375,
	msg376,
]);

var part631 = match("MESSAGE#374:00019", "nwparser.payload", "Attempt to enable WebTrends has %{disposition->} because WebTrends settings have not yet been configured", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg377 = msg("00019", part631);

var part632 = match("MESSAGE#375:00019:01/2", "nwparser.p0", "has %{disposition->} because syslog settings have not yet been configured");

var all125 = all_match({
	processors: [
		dup165,
		dup365,
		part632,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg378 = msg("00019:01", all125);

var part633 = match("MESSAGE#376:00019:02/0", "nwparser.payload", "Socket cannot be assigned for %{p0}");

var part634 = match("MESSAGE#376:00019:02/1_0", "nwparser.p0", "WebTrends%{}");

var part635 = match("MESSAGE#376:00019:02/1_1", "nwparser.p0", "syslog%{}");

var select139 = linear_select([
	part634,
	part635,
]);

var all126 = all_match({
	processors: [
		part633,
		select139,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg379 = msg("00019:02", all126);

var part636 = match("MESSAGE#377:00019:03", "nwparser.payload", "Syslog VPN encryption has been %{disposition}", processor_chain([
	dup91,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg380 = msg("00019:03", part636);

var select140 = linear_select([
	dup169,
	dup78,
]);

var select141 = linear_select([
	dup139,
	dup170,
	dup137,
	dup122,
]);

var all127 = all_match({
	processors: [
		dup168,
		select140,
		dup23,
		select141,
		dup171,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg381 = msg("00019:04", all127);

var part637 = match("MESSAGE#379:00019:05/0", "nwparser.payload", "Syslog message level has been changed to %{p0}");

var part638 = match("MESSAGE#379:00019:05/1_0", "nwparser.p0", "debug%{}");

var part639 = match("MESSAGE#379:00019:05/1_1", "nwparser.p0", "information%{}");

var part640 = match("MESSAGE#379:00019:05/1_2", "nwparser.p0", "notification%{}");

var part641 = match("MESSAGE#379:00019:05/1_3", "nwparser.p0", "warning%{}");

var part642 = match("MESSAGE#379:00019:05/1_4", "nwparser.p0", "error%{}");

var part643 = match("MESSAGE#379:00019:05/1_5", "nwparser.p0", "critical%{}");

var part644 = match("MESSAGE#379:00019:05/1_6", "nwparser.p0", "alert%{}");

var part645 = match("MESSAGE#379:00019:05/1_7", "nwparser.p0", "emergency%{}");

var select142 = linear_select([
	part638,
	part639,
	part640,
	part641,
	part642,
	part643,
	part644,
	part645,
]);

var all128 = all_match({
	processors: [
		part637,
		select142,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg382 = msg("00019:05", all128);

var part646 = match("MESSAGE#380:00019:06/2", "nwparser.p0", "has been changed to %{p0}");

var all129 = all_match({
	processors: [
		dup168,
		dup366,
		part646,
		dup367,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg383 = msg("00019:06", all129);

var part647 = match("MESSAGE#381:00019:07", "nwparser.payload", "WebTrends VPN encryption has been %{disposition}", processor_chain([
	dup91,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg384 = msg("00019:07", part647);

var part648 = match("MESSAGE#382:00019:08", "nwparser.payload", "WebTrends has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg385 = msg("00019:08", part648);

var part649 = match("MESSAGE#383:00019:09/0", "nwparser.payload", "WebTrends host %{p0}");

var select143 = linear_select([
	dup139,
	dup170,
	dup137,
]);

var all130 = all_match({
	processors: [
		part649,
		select143,
		dup171,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg386 = msg("00019:09", all130);

var part650 = match("MESSAGE#384:00019:10/1_0", "nwparser.p0", "Traffic logging via syslog %{p0}");

var part651 = match("MESSAGE#384:00019:10/1_1", "nwparser.p0", "Syslog %{p0}");

var select144 = linear_select([
	part650,
	part651,
]);

var all131 = all_match({
	processors: [
		dup183,
		select144,
		dup138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg387 = msg("00019:10", all131);

var part652 = match("MESSAGE#385:00019:11/2", "nwparser.p0", "has %{disposition->} because there is no syslog server defined");

var all132 = all_match({
	processors: [
		dup165,
		dup365,
		part652,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg388 = msg("00019:11", all132);

var part653 = match("MESSAGE#386:00019:12", "nwparser.payload", "Removing all syslog servers%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg389 = msg("00019:12", part653);

var part654 = match("MESSAGE#387:00019:13/0", "nwparser.payload", "Syslog server %{hostip->} %{p0}");

var select145 = linear_select([
	dup107,
	dup106,
]);

var part655 = match("MESSAGE#387:00019:13/2", "nwparser.p0", "%{disposition}");

var all133 = all_match({
	processors: [
		part654,
		select145,
		part655,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg390 = msg("00019:13", all133);

var part656 = match("MESSAGE#388:00019:14/2", "nwparser.p0", "for %{hostip->} has been changed to %{p0}");

var all134 = all_match({
	processors: [
		dup168,
		dup366,
		part656,
		dup367,
	],
	on_success: processor_chain([
		dup50,
		dup43,
		dup51,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg391 = msg("00019:14", all134);

var part657 = match("MESSAGE#389:00019:15", "nwparser.payload", "Syslog cannot connect to the TCP server %{hostip}; the connection is closed.", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg392 = msg("00019:15", part657);

var part658 = match("MESSAGE#390:00019:16", "nwparser.payload", "All syslog servers were removed.%{}", processor_chain([
	setc("eventcategory","1701030000"),
	setc("ec_activity","Delete"),
	dup51,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg393 = msg("00019:16", part658);

var part659 = match("MESSAGE#391:00019:17", "nwparser.payload", "Syslog server %{hostip->} host port number has been changed to %{network_port->} %{fld5}", processor_chain([
	dup50,
	dup43,
	dup51,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg394 = msg("00019:17", part659);

var part660 = match("MESSAGE#392:00019:18/0", "nwparser.payload", "Traffic logging %{p0}");

var part661 = match("MESSAGE#392:00019:18/1_0", "nwparser.p0", "via syslog %{p0}");

var part662 = match("MESSAGE#392:00019:18/1_1", "nwparser.p0", "for syslog server %{hostip->} %{p0}");

var select146 = linear_select([
	part661,
	part662,
]);

var all135 = all_match({
	processors: [
		part660,
		select146,
		dup138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg395 = msg("00019:18", all135);

var part663 = match("MESSAGE#393:00019:19", "nwparser.payload", "Transport protocol for syslog server %{hostip->} was changed to udp", processor_chain([
	dup50,
	dup43,
	dup51,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg396 = msg("00019:19", part663);

var part664 = match("MESSAGE#394:00019:20", "nwparser.payload", "The traffic/IDP syslog is enabled on backup device by netscreen via web from host %{saddr->} to %{daddr}:%{dport}. (%{fld1})", processor_chain([
	dup50,
	dup43,
	dup51,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg397 = msg("00019:20", part664);

var select147 = linear_select([
	msg377,
	msg378,
	msg379,
	msg380,
	msg381,
	msg382,
	msg383,
	msg384,
	msg385,
	msg386,
	msg387,
	msg388,
	msg389,
	msg390,
	msg391,
	msg392,
	msg393,
	msg394,
	msg395,
	msg396,
	msg397,
]);

var part665 = match("MESSAGE#395:00020", "nwparser.payload", "Schedule %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg398 = msg("00020", part665);

var part666 = match("MESSAGE#396:00020:01/0", "nwparser.payload", "System memory is low %{p0}");

var part667 = match("MESSAGE#396:00020:01/1_1", "nwparser.p0", "( %{p0}");

var select148 = linear_select([
	dup152,
	part667,
]);

var part668 = match("MESSAGE#396:00020:01/2", "nwparser.p0", "%{fld2->} bytes allocated out of %{p0}");

var part669 = match("MESSAGE#396:00020:01/3_0", "nwparser.p0", "total %{fld3->} bytes");

var part670 = match("MESSAGE#396:00020:01/3_1", "nwparser.p0", "%{fld4->} bytes total");

var select149 = linear_select([
	part669,
	part670,
]);

var all136 = all_match({
	processors: [
		part666,
		select148,
		part668,
		select149,
	],
	on_success: processor_chain([
		dup184,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg399 = msg("00020:01", all136);

var part671 = match("MESSAGE#397:00020:02", "nwparser.payload", "System memory is low (%{fld2->} allocated out of %{fld3->} ) %{fld4->} times in %{fld5}", processor_chain([
	dup184,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg400 = msg("00020:02", part671);

var select150 = linear_select([
	msg398,
	msg399,
	msg400,
]);

var part672 = match("MESSAGE#398:00021", "nwparser.payload", "DIP %{fld2->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg401 = msg("00021", part672);

var part673 = match("MESSAGE#399:00021:01", "nwparser.payload", "IP pool %{fld2->} with range %{info->} has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg402 = msg("00021:01", part673);

var part674 = match("MESSAGE#400:00021:02", "nwparser.payload", "DNS server is not configured%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg403 = msg("00021:02", part674);

var part675 = match("MESSAGE#401:00021:03", "nwparser.payload", "Connection refused by the DNS server%{}", processor_chain([
	dup185,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg404 = msg("00021:03", part675);

var part676 = match("MESSAGE#402:00021:04", "nwparser.payload", "Unknown DNS error%{}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg405 = msg("00021:04", part676);

var part677 = match("MESSAGE#403:00021:05", "nwparser.payload", "DIP port-translatation stickiness was %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg406 = msg("00021:05", part677);

var part678 = match("MESSAGE#404:00021:06", "nwparser.payload", "DIP port-translation stickiness was %{disposition->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup9,
	setc("info","DIP port-translation stickiness was modified"),
]));

var msg407 = msg("00021:06", part678);

var select151 = linear_select([
	msg401,
	msg402,
	msg403,
	msg404,
	msg405,
	msg406,
	msg407,
]);

var part679 = match("MESSAGE#405:00022/1_0", "nwparser.p0", "power supplies %{p0}");

var part680 = match("MESSAGE#405:00022/1_1", "nwparser.p0", "fans %{p0}");

var select152 = linear_select([
	part679,
	part680,
]);

var part681 = match("MESSAGE#405:00022/2", "nwparser.p0", "are %{fld2->} functioning properly");

var all137 = all_match({
	processors: [
		dup186,
		select152,
		part681,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg408 = msg("00022", all137);

var part682 = match("MESSAGE#406:00022:01/0_0", "nwparser.payload", "At least one power supply %{p0}");

var part683 = match("MESSAGE#406:00022:01/0_1", "nwparser.payload", "The power supply %{fld2->} %{p0}");

var part684 = match("MESSAGE#406:00022:01/0_2", "nwparser.payload", "At least one fan %{p0}");

var select153 = linear_select([
	part682,
	part683,
	part684,
]);

var part685 = match("MESSAGE#406:00022:01/1", "nwparser.p0", "is not functioning properly%{p0}");

var all138 = all_match({
	processors: [
		select153,
		part685,
		dup368,
	],
	on_success: processor_chain([
		dup187,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg409 = msg("00022:01", all138);

var part686 = match("MESSAGE#407:00022:02", "nwparser.payload", "Global Manager VPN management tunnel has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg410 = msg("00022:02", part686);

var part687 = match("MESSAGE#408:00022:03", "nwparser.payload", "Global Manager domain name has been defined as %{domain}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg411 = msg("00022:03", part687);

var part688 = match("MESSAGE#409:00022:04/0", "nwparser.payload", "Reporting of the %{p0}");

var part689 = match("MESSAGE#409:00022:04/1_0", "nwparser.p0", "network activities %{p0}");

var part690 = match("MESSAGE#409:00022:04/1_1", "nwparser.p0", "device resources %{p0}");

var part691 = match("MESSAGE#409:00022:04/1_2", "nwparser.p0", "event logs %{p0}");

var part692 = match("MESSAGE#409:00022:04/1_3", "nwparser.p0", "summary logs %{p0}");

var select154 = linear_select([
	part689,
	part690,
	part691,
	part692,
]);

var part693 = match("MESSAGE#409:00022:04/2", "nwparser.p0", "to Global Manager has been %{disposition}");

var all139 = all_match({
	processors: [
		part688,
		select154,
		part693,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg412 = msg("00022:04", all139);

var part694 = match("MESSAGE#410:00022:05", "nwparser.payload", "Global Manager has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg413 = msg("00022:05", part694);

var part695 = match("MESSAGE#411:00022:06/0", "nwparser.payload", "Global Manager %{p0}");

var part696 = match("MESSAGE#411:00022:06/1_0", "nwparser.p0", "report %{p0}");

var part697 = match("MESSAGE#411:00022:06/1_1", "nwparser.p0", "listen %{p0}");

var select155 = linear_select([
	part696,
	part697,
]);

var part698 = match("MESSAGE#411:00022:06/2", "nwparser.p0", "port has been set to %{interface}");

var all140 = all_match({
	processors: [
		part695,
		select155,
		part698,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg414 = msg("00022:06", all140);

var part699 = match("MESSAGE#412:00022:07", "nwparser.payload", "The Global Manager keep-alive value has been changed to %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg415 = msg("00022:07", part699);

var part700 = match("MESSAGE#413:00022:08/0_0", "nwparser.payload", "System temperature %{p0}");

var part701 = match("MESSAGE#413:00022:08/0_1", "nwparser.payload", "System's temperature: %{p0}");

var part702 = match("MESSAGE#413:00022:08/0_2", "nwparser.payload", "The system temperature %{p0}");

var select156 = linear_select([
	part700,
	part701,
	part702,
]);

var part703 = match("MESSAGE#413:00022:08/1", "nwparser.p0", "(%{fld2->} C%{p0}");

var part704 = match("MESSAGE#413:00022:08/2_0", "nwparser.p0", "entigrade, %{p0}");

var select157 = linear_select([
	part704,
	dup96,
]);

var part705 = match("MESSAGE#413:00022:08/3", "nwparser.p0", "%{fld3->} F%{p0}");

var part706 = match("MESSAGE#413:00022:08/4_0", "nwparser.p0", "ahrenheit %{p0}");

var select158 = linear_select([
	part706,
	dup96,
]);

var part707 = match("MESSAGE#413:00022:08/5", "nwparser.p0", ") is too high%{}");

var all141 = all_match({
	processors: [
		select156,
		part703,
		select157,
		part705,
		select158,
		part707,
	],
	on_success: processor_chain([
		dup188,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg416 = msg("00022:08", all141);

var part708 = match("MESSAGE#414:00022:09/2", "nwparser.p0", "power supply is no%{p0}");

var select159 = linear_select([
	dup191,
	dup192,
]);

var part709 = match("MESSAGE#414:00022:09/4", "nwparser.p0", "functioning properly%{}");

var all142 = all_match({
	processors: [
		dup55,
		dup369,
		part708,
		select159,
		part709,
	],
	on_success: processor_chain([
		dup188,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg417 = msg("00022:09", all142);

var part710 = match("MESSAGE#415:00022:10/0", "nwparser.payload", "The NetScreen device was unable to upgrade the file system%{p0}");

var part711 = match("MESSAGE#415:00022:10/1_0", "nwparser.p0", " due to an internal conflict%{}");

var part712 = match("MESSAGE#415:00022:10/1_1", "nwparser.p0", ", but the old file system is intact%{}");

var select160 = linear_select([
	part711,
	part712,
]);

var all143 = all_match({
	processors: [
		part710,
		select160,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg418 = msg("00022:10", all143);

var part713 = match("MESSAGE#416:00022:11/0", "nwparser.payload", "The NetScreen device was unable to upgrade %{p0}");

var part714 = match("MESSAGE#416:00022:11/1_0", "nwparser.p0", "due to an internal conflict%{}");

var part715 = match("MESSAGE#416:00022:11/1_1", "nwparser.p0", "the loader, but the loader is intact%{}");

var select161 = linear_select([
	part714,
	part715,
]);

var all144 = all_match({
	processors: [
		part713,
		select161,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg419 = msg("00022:11", all144);

var part716 = match("MESSAGE#417:00022:12/0", "nwparser.payload", "Battery is no%{p0}");

var select162 = linear_select([
	dup192,
	dup191,
]);

var part717 = match("MESSAGE#417:00022:12/2", "nwparser.p0", "functioning properly.%{}");

var all145 = all_match({
	processors: [
		part716,
		select162,
		part717,
	],
	on_success: processor_chain([
		dup188,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg420 = msg("00022:12", all145);

var part718 = match("MESSAGE#418:00022:13", "nwparser.payload", "System's temperature (%{fld2->} Centigrade, %{fld3->} Fahrenheit) is OK now.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg421 = msg("00022:13", part718);

var part719 = match("MESSAGE#419:00022:14", "nwparser.payload", "The power supply %{fld2->} is functioning properly. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg422 = msg("00022:14", part719);

var select163 = linear_select([
	msg408,
	msg409,
	msg410,
	msg411,
	msg412,
	msg413,
	msg414,
	msg415,
	msg416,
	msg417,
	msg418,
	msg419,
	msg420,
	msg421,
	msg422,
]);

var part720 = match("MESSAGE#420:00023", "nwparser.payload", "VIP server %{hostip->} is not responding", processor_chain([
	dup187,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg423 = msg("00023", part720);

var part721 = match("MESSAGE#421:00023:01", "nwparser.payload", "VIP/load balance server %{hostip->} cannot be contacted", processor_chain([
	dup187,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg424 = msg("00023:01", part721);

var part722 = match("MESSAGE#422:00023:02", "nwparser.payload", "VIP server %{hostip->} cannot be contacted", processor_chain([
	dup187,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg425 = msg("00023:02", part722);

var select164 = linear_select([
	msg423,
	msg424,
	msg425,
]);

var part723 = match("MESSAGE#423:00024/0_0", "nwparser.payload", "The DHCP %{p0}");

var part724 = match("MESSAGE#423:00024/0_1", "nwparser.payload", " DHCP %{p0}");

var select165 = linear_select([
	part723,
	part724,
]);

var part725 = match("MESSAGE#423:00024/2_0", "nwparser.p0", "IP address pool has %{p0}");

var part726 = match("MESSAGE#423:00024/2_1", "nwparser.p0", "options have been %{p0}");

var select166 = linear_select([
	part725,
	part726,
]);

var all146 = all_match({
	processors: [
		select165,
		dup193,
		select166,
		dup52,
		dup368,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg426 = msg("00024", all146);

var part727 = match("MESSAGE#424:00024:01/0_0", "nwparser.payload", "Traffic log %{p0}");

var part728 = match("MESSAGE#424:00024:01/0_1", "nwparser.payload", "Alarm log %{p0}");

var part729 = match("MESSAGE#424:00024:01/0_2", "nwparser.payload", "Event log %{p0}");

var part730 = match("MESSAGE#424:00024:01/0_3", "nwparser.payload", "Self log %{p0}");

var part731 = match("MESSAGE#424:00024:01/0_4", "nwparser.payload", "Asset Recovery log %{p0}");

var select167 = linear_select([
	part727,
	part728,
	part729,
	part730,
	part731,
]);

var part732 = match("MESSAGE#424:00024:01/1", "nwparser.p0", "has overflowed%{}");

var all147 = all_match({
	processors: [
		select167,
		part732,
	],
	on_success: processor_chain([
		dup117,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg427 = msg("00024:01", all147);

var part733 = match("MESSAGE#425:00024:02/0", "nwparser.payload", "DHCP relay agent settings on %{fld2->} %{p0}");

var part734 = match("MESSAGE#425:00024:02/1_0", "nwparser.p0", "are %{p0}");

var part735 = match("MESSAGE#425:00024:02/1_1", "nwparser.p0", "have been %{p0}");

var select168 = linear_select([
	part734,
	part735,
]);

var part736 = match("MESSAGE#425:00024:02/2", "nwparser.p0", "%{disposition->} (%{fld1})");

var all148 = all_match({
	processors: [
		part733,
		select168,
		part736,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg428 = msg("00024:02", all148);

var part737 = match("MESSAGE#426:00024:03/0", "nwparser.payload", "DHCP server IP address pool %{p0}");

var select169 = linear_select([
	dup194,
	dup106,
]);

var part738 = match("MESSAGE#426:00024:03/2", "nwparser.p0", "changed. (%{fld1})");

var all149 = all_match({
	processors: [
		part737,
		select169,
		part738,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg429 = msg("00024:03", all149);

var select170 = linear_select([
	msg426,
	msg427,
	msg428,
	msg429,
]);

var part739 = match("MESSAGE#427:00025", "nwparser.payload", "The DHCP server IP address pool has changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg430 = msg("00025", part739);

var part740 = match("MESSAGE#428:00025:01", "nwparser.payload", "PKI: The current device %{disposition->} to save the certificate authority configuration.", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg431 = msg("00025:01", part740);

var part741 = match("MESSAGE#429:00025:02", "nwparser.payload", "%{disposition->} to send the X509 request file via e-mail", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg432 = msg("00025:02", part741);

var part742 = match("MESSAGE#430:00025:03", "nwparser.payload", "%{disposition->} to save the CA configuration", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg433 = msg("00025:03", part742);

var part743 = match("MESSAGE#431:00025:04", "nwparser.payload", "Cannot load more X509 certificates. The %{result}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg434 = msg("00025:04", part743);

var select171 = linear_select([
	msg430,
	msg431,
	msg432,
	msg433,
	msg434,
]);

var part744 = match("MESSAGE#432:00026", "nwparser.payload", "%{signame->} have been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg435 = msg("00026", part744);

var part745 = match("MESSAGE#433:00026:13", "nwparser.payload", "%{signame->} have been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on interface %{interface}", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg436 = msg("00026:13", part745);

var part746 = match("MESSAGE#434:00026:01/2", "nwparser.p0", "PKA key has been %{p0}");

var part747 = match("MESSAGE#434:00026:01/4", "nwparser.p0", "admin user %{administrator}. (Key ID = %{fld2})");

var all150 = all_match({
	processors: [
		dup195,
		dup370,
		part746,
		dup371,
		part747,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg437 = msg("00026:01", all150);

var part748 = match("MESSAGE#435:00026:02/1_0", "nwparser.p0", ": SCS %{p0}");

var select172 = linear_select([
	part748,
	dup96,
]);

var part749 = match("MESSAGE#435:00026:02/2", "nwparser.p0", "has been %{disposition->} for %{p0}");

var part750 = match("MESSAGE#435:00026:02/3_0", "nwparser.p0", "root system %{p0}");

var part751 = match("MESSAGE#435:00026:02/3_1", "nwparser.p0", "%{interface->} %{p0}");

var select173 = linear_select([
	part750,
	part751,
]);

var all151 = all_match({
	processors: [
		dup195,
		select172,
		part749,
		select173,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg438 = msg("00026:02", all151);

var part752 = match("MESSAGE#436:00026:03/2", "nwparser.p0", "%{change_attribute->} has been changed from %{change_old->} to %{change_new}");

var all152 = all_match({
	processors: [
		dup195,
		dup370,
		part752,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg439 = msg("00026:03", all152);

var part753 = match("MESSAGE#437:00026:04", "nwparser.payload", "SCS: Connection has been terminated for admin user %{administrator->} at %{hostip}:%{network_port}", processor_chain([
	dup198,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg440 = msg("00026:04", part753);

var part754 = match("MESSAGE#438:00026:05", "nwparser.payload", "SCS: Host client has requested NO cipher from %{interface}", processor_chain([
	dup198,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg441 = msg("00026:05", part754);

var part755 = match("MESSAGE#439:00026:06", "nwparser.payload", "SCS: SSH user %{username->} has been authenticated using PKA RSA from %{saddr}:%{sport}. (key-ID=%{fld2}", processor_chain([
	dup199,
	dup29,
	dup30,
	dup31,
	dup32,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg442 = msg("00026:06", part755);

var part756 = match("MESSAGE#440:00026:07", "nwparser.payload", "SCS: SSH user %{username->} has been authenticated using password from %{saddr}:%{sport}.", processor_chain([
	dup199,
	dup29,
	dup30,
	dup31,
	dup32,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg443 = msg("00026:07", part756);

var part757 = match("MESSAGE#441:00026:08/0", "nwparser.payload", "SSH user %{username->} has been authenticated using %{p0}");

var part758 = match("MESSAGE#441:00026:08/2", "nwparser.p0", "from %{saddr}:%{sport->} [ with key ID %{fld2->} ]");

var all153 = all_match({
	processors: [
		part757,
		dup372,
		part758,
	],
	on_success: processor_chain([
		dup199,
		dup29,
		dup30,
		dup31,
		dup32,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg444 = msg("00026:08", all153);

var part759 = match("MESSAGE#442:00026:09", "nwparser.payload", "IPSec tunnel on int %{interface->} with tunnel ID %{fld2->} received a packet with a bad SPI.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg445 = msg("00026:09", part759);

var part760 = match("MESSAGE#443:00026:10/0", "nwparser.payload", "SSH: %{p0}");

var part761 = match("MESSAGE#443:00026:10/1_0", "nwparser.p0", "Failed %{p0}");

var part762 = match("MESSAGE#443:00026:10/1_1", "nwparser.p0", "Attempt %{p0}");

var select174 = linear_select([
	part761,
	part762,
]);

var part763 = match("MESSAGE#443:00026:10/3_0", "nwparser.p0", "bind duplicate %{p0}");

var select175 = linear_select([
	part763,
	dup201,
]);

var part764 = match("MESSAGE#443:00026:10/6", "nwparser.p0", "admin user '%{administrator}' (Key ID %{fld2})");

var all154 = all_match({
	processors: [
		part760,
		select174,
		dup103,
		select175,
		dup202,
		dup373,
		part764,
	],
	on_success: processor_chain([
		dup203,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg446 = msg("00026:10", all154);

var part765 = match("MESSAGE#444:00026:11", "nwparser.payload", "SSH: Maximum number of PKA keys (%{fld2}) has been bound to user '%{username}' Key not bound. (Key ID %{fld3})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg447 = msg("00026:11", part765);

var part766 = match("MESSAGE#445:00026:12", "nwparser.payload", "IKE %{fld2}: Missing heartbeats have exceeded the threshold. All Phase 1 and 2 SAs have been removed", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg448 = msg("00026:12", part766);

var select176 = linear_select([
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
]);

var part767 = match("MESSAGE#446:00027/2", "nwparser.p0", "user %{username->} from %{p0}");

var part768 = match("MESSAGE#446:00027/3_0", "nwparser.p0", "IP address %{saddr}:%{sport}");

var part769 = match("MESSAGE#446:00027/3_1", "nwparser.p0", "%{saddr}:%{sport}");

var part770 = match("MESSAGE#446:00027/3_2", "nwparser.p0", "console%{}");

var select177 = linear_select([
	part768,
	part769,
	part770,
]);

var all155 = all_match({
	processors: [
		dup204,
		dup374,
		part767,
		select177,
	],
	on_success: processor_chain([
		dup206,
		dup30,
		dup31,
		dup54,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg449 = msg("00027", all155);

var part771 = match("MESSAGE#447:00027:01", "nwparser.payload", "%{change_attribute->} has been restored from %{change_old->} to default port %{change_new}. %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg450 = msg("00027:01", part771);

var part772 = match("MESSAGE#448:00027:02", "nwparser.payload", "%{change_attribute->} has been restored from %{change_old->} to %{change_new}. %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg451 = msg("00027:02", part772);

var part773 = match("MESSAGE#449:00027:03", "nwparser.payload", "%{change_attribute->} has been changed from %{change_old->} to port %{change_new}. %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg452 = msg("00027:03", part773);

var part774 = match("MESSAGE#450:00027:04", "nwparser.payload", "%{change_attribute->} has been changed from %{change_old->} to port %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg453 = msg("00027:04", part774);

var part775 = match("MESSAGE#451:00027:05/0", "nwparser.payload", "ScreenOS %{version->} %{p0}");

var part776 = match("MESSAGE#451:00027:05/1_0", "nwparser.p0", "Serial %{p0}");

var part777 = match("MESSAGE#451:00027:05/1_1", "nwparser.p0", "serial %{p0}");

var select178 = linear_select([
	part776,
	part777,
]);

var part778 = match("MESSAGE#451:00027:05/2", "nwparser.p0", "# %{fld2}: Asset recovery %{p0}");

var part779 = match("MESSAGE#451:00027:05/3_0", "nwparser.p0", "performed %{p0}");

var select179 = linear_select([
	part779,
	dup127,
]);

var select180 = linear_select([
	dup207,
	dup208,
]);

var all156 = all_match({
	processors: [
		part775,
		select178,
		part778,
		select179,
		dup23,
		select180,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg454 = msg("00027:05", all156);

var part780 = match("MESSAGE#452:00027:06/0", "nwparser.payload", "Device Reset (Asset Recovery) has been %{p0}");

var select181 = linear_select([
	dup208,
	dup207,
]);

var all157 = all_match({
	processors: [
		part780,
		select181,
	],
	on_success: processor_chain([
		setc("eventcategory","1606000000"),
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg455 = msg("00027:06", all157);

var part781 = match("MESSAGE#453:00027:07", "nwparser.payload", "%{change_attribute->} has been changed from %{change_old->} to %{change_new}. %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg456 = msg("00027:07", part781);

var part782 = match("MESSAGE#454:00027:08", "nwparser.payload", "System configuration has been erased%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg457 = msg("00027:08", part782);

var part783 = match("MESSAGE#455:00027:09", "nwparser.payload", "License key %{fld2->} is due to expire in %{fld3}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg458 = msg("00027:09", part783);

var part784 = match("MESSAGE#456:00027:10", "nwparser.payload", "License key %{fld2->} has expired.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg459 = msg("00027:10", part784);

var part785 = match("MESSAGE#457:00027:11", "nwparser.payload", "License key %{fld2->} expired after 30-day grace period.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg460 = msg("00027:11", part785);

var part786 = match("MESSAGE#458:00027:12/0", "nwparser.payload", "Request to retrieve license key failed to reach %{p0}");

var part787 = match("MESSAGE#458:00027:12/1_0", "nwparser.p0", "the server %{p0}");

var select182 = linear_select([
	part787,
	dup193,
]);

var part788 = match("MESSAGE#458:00027:12/2", "nwparser.p0", "by %{fld2}. Server url: %{url}");

var all158 = all_match({
	processors: [
		part786,
		select182,
		part788,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg461 = msg("00027:12", all158);

var part789 = match("MESSAGE#459:00027:13/2", "nwparser.p0", "user %{username}");

var all159 = all_match({
	processors: [
		dup204,
		dup374,
		part789,
	],
	on_success: processor_chain([
		dup206,
		dup30,
		dup31,
		dup54,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg462 = msg("00027:13", all159);

var part790 = match("MESSAGE#460:00027:14/0", "nwparser.payload", "Configuration Erasure Process %{p0}");

var part791 = match("MESSAGE#460:00027:14/1_0", "nwparser.p0", "has been initiated %{p0}");

var part792 = match("MESSAGE#460:00027:14/1_1", "nwparser.p0", "aborted %{p0}");

var select183 = linear_select([
	part791,
	part792,
]);

var part793 = match("MESSAGE#460:00027:14/2", "nwparser.p0", ".%{space}(%{fld1})");

var all160 = all_match({
	processors: [
		part790,
		select183,
		part793,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg463 = msg("00027:14", all160);

var part794 = match("MESSAGE#461:00027:15", "nwparser.payload", "Waiting for 2nd confirmation. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg464 = msg("00027:15", part794);

var part795 = match("MESSAGE#1220:00027:16", "nwparser.payload", "Admin %{fld3->} policy id %{policy_id->} name \"%{fld2->} has been re-enabled by NetScreen system after being locked due to excessive failed login attempts (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg465 = msg("00027:16", part795);

var part796 = match("MESSAGE#1225:00027:17", "nwparser.payload", "Admin %{username->} is locked and will be unlocked after %{duration->} minutes (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg466 = msg("00027:17", part796);

var part797 = match("MESSAGE#1226:00027:18", "nwparser.payload", "Login attempt by admin %{username->} from %{saddr->} is refused as this account is locked (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg467 = msg("00027:18", part797);

var part798 = match("MESSAGE#1227:00027:19", "nwparser.payload", "Admin %{username->} has been re-enabled by NetScreen system after being locked due to excessive failed login attempts (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg468 = msg("00027:19", part798);

var select184 = linear_select([
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
]);

var part799 = match("MESSAGE#462:00028/0_0", "nwparser.payload", "An Intruder%{p0}");

var part800 = match("MESSAGE#462:00028/0_1", "nwparser.payload", "Intruder%{p0}");

var part801 = match("MESSAGE#462:00028/0_2", "nwparser.payload", "An intruter%{p0}");

var select185 = linear_select([
	part799,
	part800,
	part801,
]);

var part802 = match("MESSAGE#462:00028/1", "nwparser.p0", "%{}has attempted to connect to the NetScreen-Global PRO port! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} at interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times");

var all161 = all_match({
	processors: [
		select185,
		part802,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
		setc("signame","Attempt to Connect to the NetScreen-Global Port"),
	]),
});

var msg469 = msg("00028", all161);

var part803 = match("MESSAGE#463:00029", "nwparser.payload", "DNS has been refreshed%{}", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg470 = msg("00029", part803);

var part804 = match("MESSAGE#464:00029:01", "nwparser.payload", "DHCP file write: out of memory.%{}", processor_chain([
	dup184,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg471 = msg("00029:01", part804);

var part805 = match("MESSAGE#465:00029:02/0", "nwparser.payload", "The DHCP process cannot open file %{fld2->} to %{p0}");

var part806 = match("MESSAGE#465:00029:02/1_0", "nwparser.p0", "read %{p0}");

var part807 = match("MESSAGE#465:00029:02/1_1", "nwparser.p0", "write %{p0}");

var select186 = linear_select([
	part806,
	part807,
]);

var part808 = match("MESSAGE#465:00029:02/2", "nwparser.p0", "data.%{}");

var all162 = all_match({
	processors: [
		part805,
		select186,
		part808,
	],
	on_success: processor_chain([
		dup117,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg472 = msg("00029:02", all162);

var part809 = match("MESSAGE#466:00029:03/2", "nwparser.p0", "%{} %{interface->} is full. Unable to %{p0}");

var part810 = match("MESSAGE#466:00029:03/3_0", "nwparser.p0", "commit %{p0}");

var part811 = match("MESSAGE#466:00029:03/3_1", "nwparser.p0", "offer %{p0}");

var select187 = linear_select([
	part810,
	part811,
]);

var part812 = match("MESSAGE#466:00029:03/4", "nwparser.p0", "IP address to client at %{fld2}");

var all163 = all_match({
	processors: [
		dup210,
		dup337,
		part809,
		select187,
		part812,
	],
	on_success: processor_chain([
		dup117,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg473 = msg("00029:03", all163);

var part813 = match("MESSAGE#467:00029:04", "nwparser.payload", "DHCP server set to OFF on %{interface->} (another server found on %{hostip}).", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg474 = msg("00029:04", part813);

var select188 = linear_select([
	msg470,
	msg471,
	msg472,
	msg473,
	msg474,
]);

var part814 = match("MESSAGE#468:00030", "nwparser.payload", "CA configuration is invalid%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg475 = msg("00030", part814);

var part815 = match("MESSAGE#469:00030:01/0", "nwparser.payload", "DSS checking of CRLs has been changed from %{p0}");

var part816 = match("MESSAGE#469:00030:01/1_0", "nwparser.p0", "0 to 1%{}");

var part817 = match("MESSAGE#469:00030:01/1_1", "nwparser.p0", "1 to 0%{}");

var select189 = linear_select([
	part816,
	part817,
]);

var all164 = all_match({
	processors: [
		part815,
		select189,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg476 = msg("00030:01", all164);

var part818 = match("MESSAGE#470:00030:05", "nwparser.payload", "For the X509 certificate %{change_attribute->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg477 = msg("00030:05", part818);

var part819 = match("MESSAGE#471:00030:06", "nwparser.payload", "In the X509 certificate request the %{fld2->} field has been changed from %{fld3}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg478 = msg("00030:06", part819);

var part820 = match("MESSAGE#472:00030:07", "nwparser.payload", "RA X509 certificate cannot be loaded%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg479 = msg("00030:07", part820);

var part821 = match("MESSAGE#473:00030:10", "nwparser.payload", "Self-signed X509 certificate cannot be generated%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg480 = msg("00030:10", part821);

var part822 = match("MESSAGE#474:00030:12", "nwparser.payload", "The public key for ScreenOS image has successfully been updated%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg481 = msg("00030:12", part822);

var part823 = match("MESSAGE#475:00030:13/0", "nwparser.payload", "The public key used for ScreenOS image authentication cannot be %{p0}");

var part824 = match("MESSAGE#475:00030:13/1_0", "nwparser.p0", "decoded%{}");

var part825 = match("MESSAGE#475:00030:13/1_1", "nwparser.p0", "loaded%{}");

var select190 = linear_select([
	part824,
	part825,
]);

var all165 = all_match({
	processors: [
		part823,
		select190,
	],
	on_success: processor_chain([
		dup35,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg482 = msg("00030:13", all165);

var part826 = match("MESSAGE#476:00030:14/1_0", "nwparser.p0", "CA IDENT %{p0}");

var part827 = match("MESSAGE#476:00030:14/1_1", "nwparser.p0", "Challenge password %{p0}");

var part828 = match("MESSAGE#476:00030:14/1_2", "nwparser.p0", "CA CGI URL %{p0}");

var part829 = match("MESSAGE#476:00030:14/1_3", "nwparser.p0", "RA CGI URL %{p0}");

var select191 = linear_select([
	part826,
	part827,
	part828,
	part829,
]);

var part830 = match("MESSAGE#476:00030:14/2", "nwparser.p0", "for SCEP %{p0}");

var part831 = match("MESSAGE#476:00030:14/3_0", "nwparser.p0", "requests %{p0}");

var select192 = linear_select([
	part831,
	dup16,
]);

var part832 = match("MESSAGE#476:00030:14/4", "nwparser.p0", "has been changed from %{change_old->} to %{change_new}");

var all166 = all_match({
	processors: [
		dup55,
		select191,
		part830,
		select192,
		part832,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg483 = msg("00030:14", all166);

var msg484 = msg("00030:02", dup375);

var part833 = match("MESSAGE#478:00030:15", "nwparser.payload", "X509 certificate for ScreenOS image authentication is invalid%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg485 = msg("00030:15", part833);

var part834 = match("MESSAGE#479:00030:16", "nwparser.payload", "X509 certificate has been deleted%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg486 = msg("00030:16", part834);

var part835 = match("MESSAGE#480:00030:18", "nwparser.payload", "PKI CRL: no revoke info accept per config DN %{interface}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg487 = msg("00030:18", part835);

var part836 = match("MESSAGE#481:00030:19/0", "nwparser.payload", "PKI: A configurable item %{change_attribute->} %{p0}");

var part837 = match("MESSAGE#481:00030:19/1_0", "nwparser.p0", "mode %{p0}");

var part838 = match("MESSAGE#481:00030:19/1_1", "nwparser.p0", "field%{p0}");

var select193 = linear_select([
	part837,
	part838,
]);

var part839 = match("MESSAGE#481:00030:19/2", "nwparser.p0", "%{}has changed from %{change_old->} to %{change_new}");

var all167 = all_match({
	processors: [
		part836,
		select193,
		part839,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg488 = msg("00030:19", all167);

var part840 = match("MESSAGE#482:00030:30", "nwparser.payload", "PKI: NSRP cold sync start for total of %{fld2->} items.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg489 = msg("00030:30", part840);

var part841 = match("MESSAGE#483:00030:31", "nwparser.payload", "PKI: NSRP sync received cold sync item %{fld2->} out of order expect %{fld3->} of %{fld4}.", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg490 = msg("00030:31", part841);

var part842 = match("MESSAGE#484:00030:32", "nwparser.payload", "PKI: NSRP sync received cold sync item %{fld2->} without first item.", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg491 = msg("00030:32", part842);

var part843 = match("MESSAGE#485:00030:33", "nwparser.payload", "PKI: NSRP sync received normal item during cold sync.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg492 = msg("00030:33", part843);

var part844 = match("MESSAGE#486:00030:34", "nwparser.payload", "PKI: The CRL %{policy_id->} is deleted.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg493 = msg("00030:34", part844);

var part845 = match("MESSAGE#487:00030:35", "nwparser.payload", "PKI: The NSRP high availability synchronization %{fld2->} failed.", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg494 = msg("00030:35", part845);

var part846 = match("MESSAGE#488:00030:36", "nwparser.payload", "PKI: The %{change_attribute->} has changed from %{change_old->} to %{change_new}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg495 = msg("00030:36", part846);

var part847 = match("MESSAGE#489:00030:37", "nwparser.payload", "PKI: The X.509 certificate for the ScreenOS image authentication is invalid.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg496 = msg("00030:37", part847);

var part848 = match("MESSAGE#490:00030:38", "nwparser.payload", "PKI: The X.509 local certificate cannot be sync to vsd member.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg497 = msg("00030:38", part848);

var part849 = match("MESSAGE#491:00030:39/0", "nwparser.payload", "PKI: The X.509 certificate %{p0}");

var part850 = match("MESSAGE#491:00030:39/1_0", "nwparser.p0", "revocation list %{p0}");

var select194 = linear_select([
	part850,
	dup16,
]);

var part851 = match("MESSAGE#491:00030:39/2", "nwparser.p0", "cannot be loaded during NSRP synchronization.%{}");

var all168 = all_match({
	processors: [
		part849,
		select194,
		part851,
	],
	on_success: processor_chain([
		dup35,
		dup211,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg498 = msg("00030:39", all168);

var part852 = match("MESSAGE#492:00030:17/0", "nwparser.payload", "X509 %{p0}");

var part853 = match("MESSAGE#492:00030:17/2", "nwparser.p0", "cannot be loaded%{}");

var all169 = all_match({
	processors: [
		part852,
		dup376,
		part853,
	],
	on_success: processor_chain([
		dup35,
		dup211,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg499 = msg("00030:17", all169);

var part854 = match("MESSAGE#493:00030:40/0", "nwparser.payload", "PKI: The certificate %{fld2->} will expire %{p0}");

var part855 = match("MESSAGE#493:00030:40/1_1", "nwparser.p0", "please %{p0}");

var select195 = linear_select([
	dup214,
	part855,
]);

var part856 = match("MESSAGE#493:00030:40/2", "nwparser.p0", "renew.%{}");

var all170 = all_match({
	processors: [
		part854,
		select195,
		part856,
	],
	on_success: processor_chain([
		dup35,
		dup211,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg500 = msg("00030:40", all170);

var part857 = match("MESSAGE#494:00030:41", "nwparser.payload", "PKI: The certificate revocation list has expired issued by certificate authority %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg501 = msg("00030:41", part857);

var part858 = match("MESSAGE#495:00030:42", "nwparser.payload", "PKI: The configuration content of certificate authority %{fld2->} is not valid.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg502 = msg("00030:42", part858);

var part859 = match("MESSAGE#496:00030:43", "nwparser.payload", "PKI: The device cannot allocate this object id number %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg503 = msg("00030:43", part859);

var part860 = match("MESSAGE#497:00030:44", "nwparser.payload", "PKI: The device cannot extract the X.509 certificate revocation list [ (CRL) ].%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg504 = msg("00030:44", part860);

var part861 = match("MESSAGE#498:00030:45", "nwparser.payload", "PKI: The device cannot find the PKI object %{fld2->} during cold sync.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg505 = msg("00030:45", part861);

var part862 = match("MESSAGE#499:00030:46", "nwparser.payload", "PKI: The device cannot load X.509 certificate onto the device certificate %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg506 = msg("00030:46", part862);

var part863 = match("MESSAGE#500:00030:47", "nwparser.payload", "PKI: The device cannot load a certificate pending SCEP completion.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg507 = msg("00030:47", part863);

var part864 = match("MESSAGE#501:00030:48", "nwparser.payload", "PKI: The device cannot load an X.509 certificate revocation list (CRL).%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg508 = msg("00030:48", part864);

var part865 = match("MESSAGE#502:00030:49", "nwparser.payload", "PKI: The device cannot load the CA certificate received through SCEP.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg509 = msg("00030:49", part865);

var part866 = match("MESSAGE#503:00030:50", "nwparser.payload", "PKI: The device cannot load the X.509 certificate revocation list (CRL) from the file.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg510 = msg("00030:50", part866);

var part867 = match("MESSAGE#504:00030:51", "nwparser.payload", "PKI: The device cannot load the X.509 local certificate received through SCEP.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg511 = msg("00030:51", part867);

var part868 = match("MESSAGE#505:00030:52", "nwparser.payload", "PKI: The device cannot load the X.509 %{product->} during boot.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg512 = msg("00030:52", part868);

var part869 = match("MESSAGE#506:00030:53", "nwparser.payload", "PKI: The device cannot load the X.509 certificate file.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg513 = msg("00030:53", part869);

var part870 = match("MESSAGE#507:00030:54", "nwparser.payload", "PKI: The device completed the coldsync of the PKI object at %{fld2->} attempt.", processor_chain([
	dup44,
	dup211,
	dup31,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg514 = msg("00030:54", part870);

var part871 = match("MESSAGE#508:00030:55/0", "nwparser.payload", "PKI: The device could not generate %{p0}");

var all171 = all_match({
	processors: [
		part871,
		dup377,
		dup217,
	],
	on_success: processor_chain([
		dup35,
		dup211,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg515 = msg("00030:55", all171);

var part872 = match("MESSAGE#509:00030:56", "nwparser.payload", "PKI: The device detected an invalid RSA key.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg516 = msg("00030:56", part872);

var part873 = match("MESSAGE#510:00030:57", "nwparser.payload", "PKI: The device detected an invalid digital signature algorithm (DSA) key.%{}", processor_chain([
	dup35,
	dup218,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg517 = msg("00030:57", part873);

var part874 = match("MESSAGE#511:00030:58", "nwparser.payload", "PKI: The device failed to coldsync the PKI object at %{fld2->} attempt.", processor_chain([
	dup86,
	dup218,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg518 = msg("00030:58", part874);

var part875 = match("MESSAGE#512:00030:59", "nwparser.payload", "PKI: The device failed to decode the public key of the image%{quote}s signer certificate.", processor_chain([
	dup35,
	dup218,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg519 = msg("00030:59", part875);

var part876 = match("MESSAGE#513:00030:60", "nwparser.payload", "PKI: The device failed to install the RSA key.%{}", processor_chain([
	dup35,
	dup218,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg520 = msg("00030:60", part876);

var part877 = match("MESSAGE#514:00030:61", "nwparser.payload", "PKI: The device failed to retrieve the pending certificate %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg521 = msg("00030:61", part877);

var part878 = match("MESSAGE#515:00030:62", "nwparser.payload", "PKI: The device failed to save the certificate authority related configuration.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg522 = msg("00030:62", part878);

var part879 = match("MESSAGE#516:00030:63", "nwparser.payload", "PKI: The device failed to store the authority configuration.%{}", processor_chain([
	dup18,
	dup219,
	dup51,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg523 = msg("00030:63", part879);

var part880 = match("MESSAGE#517:00030:64", "nwparser.payload", "PKI: The device failed to synchronize new DSA/RSA key pair to NSRP peer.%{}", processor_chain([
	dup18,
	dup218,
	dup51,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg524 = msg("00030:64", part880);

var part881 = match("MESSAGE#518:00030:65", "nwparser.payload", "PKI: The device failed to synchronize DSA/RSA key pair to NSRP peer.%{}", processor_chain([
	dup18,
	dup218,
	dup51,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg525 = msg("00030:65", part881);

var part882 = match("MESSAGE#519:00030:66", "nwparser.payload", "PKI: The device has detected an invalid X.509 object attribute %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg526 = msg("00030:66", part882);

var part883 = match("MESSAGE#520:00030:67", "nwparser.payload", "PKI: The device has detected invalid X.509 object content.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg527 = msg("00030:67", part883);

var part884 = match("MESSAGE#521:00030:68", "nwparser.payload", "PKI: The device has failed to load an invalid X.509 object.%{}", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg528 = msg("00030:68", part884);

var part885 = match("MESSAGE#522:00030:69", "nwparser.payload", "PKI: The device is loading the version 0 PKI data.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg529 = msg("00030:69", part885);

var part886 = match("MESSAGE#523:00030:70/0", "nwparser.payload", "PKI: The device successfully generated a new %{p0}");

var all172 = all_match({
	processors: [
		part886,
		dup377,
		dup217,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg530 = msg("00030:70", all172);

var part887 = match("MESSAGE#524:00030:71", "nwparser.payload", "PKI: The public key of image%{quote}s signer has been loaded successfully, for future image authentication.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg531 = msg("00030:71", part887);

var part888 = match("MESSAGE#525:00030:72", "nwparser.payload", "PKI: The signature of the image%{quote}s signer certificate cannot be verified.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg532 = msg("00030:72", part888);

var part889 = match("MESSAGE#526:00030:73/0", "nwparser.payload", "PKI: The %{p0}");

var part890 = match("MESSAGE#526:00030:73/1_0", "nwparser.p0", "file name %{p0}");

var part891 = match("MESSAGE#526:00030:73/1_1", "nwparser.p0", "friendly name of a certificate %{p0}");

var part892 = match("MESSAGE#526:00030:73/1_2", "nwparser.p0", "vsys name %{p0}");

var select196 = linear_select([
	part890,
	part891,
	part892,
]);

var part893 = match("MESSAGE#526:00030:73/2", "nwparser.p0", "is too long %{fld2->} to do NSRP synchronization allowed %{fld3}.");

var all173 = all_match({
	processors: [
		part889,
		select196,
		part893,
	],
	on_success: processor_chain([
		dup35,
		dup211,
		dup31,
		dup39,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg533 = msg("00030:73", all173);

var part894 = match("MESSAGE#527:00030:74", "nwparser.payload", "PKI: Upgrade from earlier version save to file.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg534 = msg("00030:74", part894);

var part895 = match("MESSAGE#528:00030:75", "nwparser.payload", "PKI: X.509 certificate has been deleted distinguished name %{username}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg535 = msg("00030:75", part895);

var part896 = match("MESSAGE#529:00030:76/0", "nwparser.payload", "PKI: X.509 %{p0}");

var part897 = match("MESSAGE#529:00030:76/2", "nwparser.p0", "file has been loaded successfully filename %{fld2}.");

var all174 = all_match({
	processors: [
		part896,
		dup376,
		part897,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg536 = msg("00030:76", all174);

var part898 = match("MESSAGE#530:00030:77", "nwparser.payload", "PKI: failed to install DSA key.%{}", processor_chain([
	dup18,
	dup218,
	dup51,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg537 = msg("00030:77", part898);

var part899 = match("MESSAGE#531:00030:78", "nwparser.payload", "PKI: no FQDN available when requesting certificate.%{}", processor_chain([
	dup35,
	dup211,
	dup220,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg538 = msg("00030:78", part899);

var part900 = match("MESSAGE#532:00030:79", "nwparser.payload", "PKI: no cert revocation check per config DN %{username}.", processor_chain([
	dup35,
	dup211,
	dup220,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg539 = msg("00030:79", part900);

var part901 = match("MESSAGE#533:00030:80", "nwparser.payload", "PKI: no nsrp sync for pre 2.5 objects.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg540 = msg("00030:80", part901);

var part902 = match("MESSAGE#534:00030:81", "nwparser.payload", "X509 certificate with subject name %{fld2->} is deleted.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg541 = msg("00030:81", part902);

var part903 = match("MESSAGE#535:00030:82", "nwparser.payload", "create new authcfg for CA %{fld2}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg542 = msg("00030:82", part903);

var part904 = match("MESSAGE#536:00030:83", "nwparser.payload", "loadCert: Cannot acquire authcfg for this CA cert %{fld2}.", processor_chain([
	dup35,
	dup211,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg543 = msg("00030:83", part904);

var part905 = match("MESSAGE#537:00030:84", "nwparser.payload", "upgrade to 4.0 copy authcfg from global.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg544 = msg("00030:84", part905);

var part906 = match("MESSAGE#538:00030:85", "nwparser.payload", "System CPU utilization is high (%{fld2->} alarm threshold: %{trigger_val}) %{info}", processor_chain([
	setc("eventcategory","1603080000"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg545 = msg("00030:85", part906);

var part907 = match("MESSAGE#539:00030:86/2", "nwparser.p0", "Pair-wise invoked by started after key generation. (%{fld1})");

var all175 = all_match({
	processors: [
		dup221,
		dup378,
		part907,
	],
	on_success: processor_chain([
		dup223,
		dup2,
		dup4,
		dup5,
		dup9,
	]),
});

var msg546 = msg("00030:86", all175);

var part908 = match("MESSAGE#1214:00030:87", "nwparser.payload", "SYSTEM CPU utilization is high (%{fld2->} > %{fld3->} ) %{fld4->} times in %{fld5->} minute (%{fld1})\u003c\u003c%{fld6}>", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
	dup9,
]));

var msg547 = msg("00030:87", part908);

var part909 = match("MESSAGE#1217:00030:88/2", "nwparser.p0", "Pair-wise invoked by passed. (%{fld1})\u003c\u003c%{fld6}>");

var all176 = all_match({
	processors: [
		dup221,
		dup378,
		part909,
	],
	on_success: processor_chain([
		dup223,
		dup2,
		dup4,
		dup5,
		dup9,
	]),
});

var msg548 = msg("00030:88", all176);

var select197 = linear_select([
	msg475,
	msg476,
	msg477,
	msg478,
	msg479,
	msg480,
	msg481,
	msg482,
	msg483,
	msg484,
	msg485,
	msg486,
	msg487,
	msg488,
	msg489,
	msg490,
	msg491,
	msg492,
	msg493,
	msg494,
	msg495,
	msg496,
	msg497,
	msg498,
	msg499,
	msg500,
	msg501,
	msg502,
	msg503,
	msg504,
	msg505,
	msg506,
	msg507,
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
	msg519,
	msg520,
	msg521,
	msg522,
	msg523,
	msg524,
	msg525,
	msg526,
	msg527,
	msg528,
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
	msg540,
	msg541,
	msg542,
	msg543,
	msg544,
	msg545,
	msg546,
	msg547,
	msg548,
]);

var part910 = match("MESSAGE#540:00031:13", "nwparser.payload", "ARP detected IP conflict: IP address %{hostip->} changed from %{sinterface->} to interface %{dinterface->} (%{fld1})", processor_chain([
	dup121,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg549 = msg("00031:13", part910);

var part911 = match("MESSAGE#541:00031", "nwparser.payload", "SNMP AuthenTraps have been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg550 = msg("00031", part911);

var part912 = match("MESSAGE#542:00031:01", "nwparser.payload", "SNMP VPN has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg551 = msg("00031:01", part912);

var part913 = match("MESSAGE#543:00031:02/0", "nwparser.payload", "SNMP community %{fld2->} attributes-write access %{p0}");

var part914 = match("MESSAGE#543:00031:02/2", "nwparser.p0", "; receive traps %{p0}");

var part915 = match("MESSAGE#543:00031:02/4", "nwparser.p0", "; receive traffic alarms %{p0}");

var part916 = match("MESSAGE#543:00031:02/6", "nwparser.p0", "-have been modified%{}");

var all177 = all_match({
	processors: [
		part913,
		dup379,
		part914,
		dup379,
		part915,
		dup379,
		part916,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg552 = msg("00031:02", all177);

var part917 = match("MESSAGE#544:00031:03/0", "nwparser.payload", "%{fld2->} SNMP host %{hostip->} has been %{p0}");

var select198 = linear_select([
	dup130,
	dup129,
]);

var part918 = match("MESSAGE#544:00031:03/2", "nwparser.p0", "SNMP community %{fld3}");

var all178 = all_match({
	processors: [
		part917,
		select198,
		part918,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg553 = msg("00031:03", all178);

var part919 = match("MESSAGE#545:00031:04/0", "nwparser.payload", "SNMP %{p0}");

var part920 = match("MESSAGE#545:00031:04/1_0", "nwparser.p0", "contact %{p0}");

var select199 = linear_select([
	part920,
	dup226,
]);

var part921 = match("MESSAGE#545:00031:04/2", "nwparser.p0", "description has been modified%{}");

var all179 = all_match({
	processors: [
		part919,
		select199,
		part921,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg554 = msg("00031:04", all179);

var part922 = match("MESSAGE#546:00031:11/0", "nwparser.payload", "SNMP system %{p0}");

var select200 = linear_select([
	dup226,
	dup25,
]);

var part923 = match("MESSAGE#546:00031:11/2", "nwparser.p0", "has been changed to %{fld2}. (%{fld1})");

var all180 = all_match({
	processors: [
		part922,
		select200,
		part923,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg555 = msg("00031:11", all180);

var part924 = match("MESSAGE#547:00031:08/0", "nwparser.payload", "%{fld2}: SNMP community name \"%{fld3}\" %{p0}");

var part925 = match("MESSAGE#547:00031:08/1_0", "nwparser.p0", "attributes -- %{p0}");

var part926 = match("MESSAGE#547:00031:08/1_1", "nwparser.p0", "-- %{p0}");

var select201 = linear_select([
	part925,
	part926,
]);

var part927 = match("MESSAGE#547:00031:08/2", "nwparser.p0", "write access, %{p0}");

var part928 = match("MESSAGE#547:00031:08/4", "nwparser.p0", "; receive traps, %{p0}");

var part929 = match("MESSAGE#547:00031:08/6", "nwparser.p0", "; receive traffic alarms, %{p0}");

var part930 = match("MESSAGE#547:00031:08/8", "nwparser.p0", "-%{p0}");

var part931 = match("MESSAGE#547:00031:08/9_0", "nwparser.p0", "- %{p0}");

var select202 = linear_select([
	part931,
	dup96,
]);

var part932 = match("MESSAGE#547:00031:08/10", "nwparser.p0", "have been modified%{}");

var all181 = all_match({
	processors: [
		part924,
		select201,
		part927,
		dup379,
		part928,
		dup379,
		part929,
		dup379,
		part930,
		select202,
		part932,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg556 = msg("00031:08", all181);

var part933 = match("MESSAGE#548:00031:05/0", "nwparser.payload", "Detect IP conflict (%{fld2}) on %{p0}");

var all182 = all_match({
	processors: [
		part933,
		dup337,
		dup227,
	],
	on_success: processor_chain([
		dup121,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg557 = msg("00031:05", all182);

var part934 = match("MESSAGE#549:00031:06/1_0", "nwparser.p0", "q, %{p0}");

var select203 = linear_select([
	part934,
	dup229,
	dup230,
]);

var part935 = match("MESSAGE#549:00031:06/2", "nwparser.p0", "detect IP conflict ( %{hostip->} )%{p0}");

var select204 = linear_select([
	dup105,
	dup96,
]);

var part936 = match("MESSAGE#549:00031:06/4", "nwparser.p0", "mac%{p0}");

var part937 = match("MESSAGE#549:00031:06/6", "nwparser.p0", "%{macaddr->} on %{p0}");

var all183 = all_match({
	processors: [
		dup228,
		select203,
		part935,
		select204,
		part936,
		dup356,
		part937,
		dup352,
		dup23,
		dup380,
	],
	on_success: processor_chain([
		dup121,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg558 = msg("00031:06", all183);

var part938 = match("MESSAGE#550:00031:07/2", "nwparser.p0", "detects a duplicate virtual security device group master IP address %{hostip}, MAC address %{macaddr->} on %{p0}");

var all184 = all_match({
	processors: [
		dup228,
		dup381,
		part938,
		dup337,
		dup227,
	],
	on_success: processor_chain([
		dup121,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg559 = msg("00031:07", all184);

var part939 = match("MESSAGE#551:00031:09/2", "nwparser.p0", "detected an IP conflict (IP %{hostip}, MAC %{macaddr}) on interface %{p0}");

var all185 = all_match({
	processors: [
		dup228,
		dup381,
		part939,
		dup380,
	],
	on_success: processor_chain([
		dup121,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg560 = msg("00031:09", all185);

var part940 = match("MESSAGE#552:00031:10", "nwparser.payload", "%{fld2}: SNMP community \"%{fld3}\" has been moved. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg561 = msg("00031:10", part940);

var part941 = match("MESSAGE#553:00031:12", "nwparser.payload", "%{fld2->} system contact has been changed to %{fld3}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg562 = msg("00031:12", part941);

var select205 = linear_select([
	msg549,
	msg550,
	msg551,
	msg552,
	msg553,
	msg554,
	msg555,
	msg556,
	msg557,
	msg558,
	msg559,
	msg560,
	msg561,
	msg562,
]);

var part942 = match("MESSAGE#554:00032", "nwparser.payload", "%{signame->} has been detected and blocked! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup232,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg563 = msg("00032", part942);

var part943 = match("MESSAGE#555:00032:01", "nwparser.payload", "%{signame->} has been detected and blocked! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}", processor_chain([
	dup232,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg564 = msg("00032:01", part943);

var part944 = match("MESSAGE#556:00032:03/0", "nwparser.payload", "Vsys %{fld2->} has been %{p0}");

var part945 = match("MESSAGE#556:00032:03/1_0", "nwparser.p0", "changed to %{fld3}");

var part946 = match("MESSAGE#556:00032:03/1_1", "nwparser.p0", "created%{}");

var part947 = match("MESSAGE#556:00032:03/1_2", "nwparser.p0", "deleted%{}");

var part948 = match("MESSAGE#556:00032:03/1_3", "nwparser.p0", "removed%{}");

var select206 = linear_select([
	part945,
	part946,
	part947,
	part948,
]);

var all186 = all_match({
	processors: [
		part944,
		select206,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg565 = msg("00032:03", all186);

var part949 = match("MESSAGE#557:00032:04", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} on interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup232,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup61,
]));

var msg566 = msg("00032:04", part949);

var part950 = match("MESSAGE#558:00032:05", "nwparser.payload", "%{change_attribute->} for vsys %{fld2->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg567 = msg("00032:05", part950);

var msg568 = msg("00032:02", dup375);

var select207 = linear_select([
	msg563,
	msg564,
	msg565,
	msg566,
	msg567,
	msg568,
]);

var part951 = match("MESSAGE#560:00033:25", "nwparser.payload", "NSM has been %{disposition}. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	setc("agent","NSM"),
]));

var msg569 = msg("00033:25", part951);

var part952 = match("MESSAGE#561:00033/1", "nwparser.p0", "timeout value has been %{p0}");

var part953 = match("MESSAGE#561:00033/2_1", "nwparser.p0", "returned%{p0}");

var select208 = linear_select([
	dup52,
	part953,
]);

var part954 = match("MESSAGE#561:00033/3", "nwparser.p0", "%{}to %{fld2}");

var all187 = all_match({
	processors: [
		dup382,
		part952,
		select208,
		part954,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg570 = msg("00033", all187);

var part955 = match("MESSAGE#562:00033:03/1_0", "nwparser.p0", "Global PRO %{p0}");

var part956 = match("MESSAGE#562:00033:03/1_1", "nwparser.p0", "%{fld3->} %{p0}");

var select209 = linear_select([
	part955,
	part956,
]);

var part957 = match("MESSAGE#562:00033:03/4", "nwparser.p0", "host has been set to %{fld4}");

var all188 = all_match({
	processors: [
		dup160,
		select209,
		dup23,
		dup369,
		part957,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg571 = msg("00033:03", all188);

var part958 = match("MESSAGE#563:00033:02/3", "nwparser.p0", "host has been %{disposition}");

var all189 = all_match({
	processors: [
		dup382,
		dup23,
		dup369,
		part958,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg572 = msg("00033:02", all189);

var part959 = match("MESSAGE#564:00033:04", "nwparser.payload", "Reporting of %{fld2->} to %{fld3->} has been %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg573 = msg("00033:04", part959);

var part960 = match("MESSAGE#565:00033:05", "nwparser.payload", "Global PRO has been %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg574 = msg("00033:05", part960);

var part961 = match("MESSAGE#566:00033:06", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at interface %{interface}. The attack occurred %{dclass_counter1->} times", processor_chain([
	dup27,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup61,
]));

var msg575 = msg("00033:06", part961);

var part962 = match("MESSAGE#567:00033:01", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at interface %{interface}. The threshold was exceeded %{dclass_counter1->} times", processor_chain([
	dup27,
	dup2,
	dup3,
	setc("dclass_counter1_string","Number of times the threshold was exceeded"),
	dup4,
	dup5,
	dup61,
]));

var msg576 = msg("00033:01", part962);

var part963 = match("MESSAGE#568:00033:07", "nwparser.payload", "User-defined service %{service->} has been %{disposition->} from %{fld2->} distribution", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg577 = msg("00033:07", part963);

var part964 = match("MESSAGE#569:00033:08/2", "nwparser.p0", "?s CA certificate field has not been specified.%{}");

var all190 = all_match({
	processors: [
		dup235,
		dup383,
		part964,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg578 = msg("00033:08", all190);

var part965 = match("MESSAGE#570:00033:09/2", "nwparser.p0", "?s Cert-Subject field has not been specified.%{}");

var all191 = all_match({
	processors: [
		dup235,
		dup383,
		part965,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg579 = msg("00033:09", all191);

var part966 = match("MESSAGE#571:00033:10/2", "nwparser.p0", "?s host field has been %{p0}");

var part967 = match("MESSAGE#571:00033:10/3_0", "nwparser.p0", "set to %{fld2->} %{p0}");

var select210 = linear_select([
	part967,
	dup238,
]);

var all192 = all_match({
	processors: [
		dup235,
		dup383,
		part966,
		select210,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg580 = msg("00033:10", all192);

var part968 = match("MESSAGE#572:00033:11/2", "nwparser.p0", "?s outgoing interface used to report NACN to Policy Manager %{p0}");

var part969 = match("MESSAGE#572:00033:11/4", "nwparser.p0", "has not been specified.%{}");

var all193 = all_match({
	processors: [
		dup235,
		dup383,
		part968,
		dup383,
		part969,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg581 = msg("00033:11", all193);

var part970 = match("MESSAGE#573:00033:12/2", "nwparser.p0", "?s password field has been %{p0}");

var select211 = linear_select([
	dup101,
	dup238,
]);

var all194 = all_match({
	processors: [
		dup235,
		dup383,
		part970,
		select211,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg582 = msg("00033:12", all194);

var part971 = match("MESSAGE#574:00033:13/2", "nwparser.p0", "?s policy-domain field has been %{p0}");

var part972 = match("MESSAGE#574:00033:13/3_0", "nwparser.p0", "unset .%{}");

var part973 = match("MESSAGE#574:00033:13/3_1", "nwparser.p0", "set to %{domain}.");

var select212 = linear_select([
	part972,
	part973,
]);

var all195 = all_match({
	processors: [
		dup235,
		dup383,
		part971,
		select212,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg583 = msg("00033:13", all195);

var part974 = match("MESSAGE#575:00033:14/2", "nwparser.p0", "?s CA certificate field has been set to %{fld2}.");

var all196 = all_match({
	processors: [
		dup235,
		dup383,
		part974,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg584 = msg("00033:14", all196);

var part975 = match("MESSAGE#576:00033:15/2", "nwparser.p0", "?s Cert-Subject field has been set to %{fld2}.");

var all197 = all_match({
	processors: [
		dup235,
		dup383,
		part975,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg585 = msg("00033:15", all197);

var part976 = match("MESSAGE#577:00033:16/2", "nwparser.p0", "?s outgoing-interface field has been set to %{interface}.");

var all198 = all_match({
	processors: [
		dup235,
		dup383,
		part976,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg586 = msg("00033:16", all198);

var part977 = match("MESSAGE#578:00033:17/2", "nwparser.p0", "?s port field has been %{p0}");

var part978 = match("MESSAGE#578:00033:17/3_0", "nwparser.p0", "set to %{network_port->} %{p0}");

var part979 = match("MESSAGE#578:00033:17/3_1", "nwparser.p0", "reset to the default value %{p0}");

var select213 = linear_select([
	part978,
	part979,
]);

var all199 = all_match({
	processors: [
		dup235,
		dup383,
		part977,
		select213,
		dup116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg587 = msg("00033:17", all199);

var part980 = match("MESSAGE#579:00033:19/0", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{p0}");

var part981 = match("MESSAGE#579:00033:19/4", "nwparser.p0", "%{fld99}arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} time.");

var all200 = all_match({
	processors: [
		part980,
		dup339,
		dup70,
		dup340,
		part981,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup4,
		dup5,
		dup3,
		dup59,
		dup61,
	]),
});

var msg588 = msg("00033:19", all200);

var part982 = match("MESSAGE#580:00033:20", "nwparser.payload", "%{signame}! From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} time.", processor_chain([
	dup27,
	dup2,
	dup4,
	dup5,
	dup3,
	dup59,
	dup60,
]));

var msg589 = msg("00033:20", part982);

var all201 = all_match({
	processors: [
		dup239,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup9,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg590 = msg("00033:21", all201);

var part983 = match("MESSAGE#582:00033:22/0", "nwparser.payload", "%{signame}! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone->} %{p0}");

var all202 = all_match({
	processors: [
		part983,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup9,
		dup59,
		dup3,
		dup4,
		dup5,
		dup60,
	]),
});

var msg591 = msg("00033:22", all202);

var part984 = match("MESSAGE#583:00033:23", "nwparser.payload", "NSM primary server with name %{hostname->} was set: addr %{hostip}, port %{network_port}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg592 = msg("00033:23", part984);

var part985 = match("MESSAGE#584:00033:24", "nwparser.payload", "session threshold From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.%{info}. (%{fld1})", processor_chain([
	setc("eventcategory","1001030500"),
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg593 = msg("00033:24", part985);

var select214 = linear_select([
	msg569,
	msg570,
	msg571,
	msg572,
	msg573,
	msg574,
	msg575,
	msg576,
	msg577,
	msg578,
	msg579,
	msg580,
	msg581,
	msg582,
	msg583,
	msg584,
	msg585,
	msg586,
	msg587,
	msg588,
	msg589,
	msg590,
	msg591,
	msg592,
	msg593,
]);

var part986 = match("MESSAGE#585:00034/0_0", "nwparser.payload", "SCS: Failed %{p0}");

var part987 = match("MESSAGE#585:00034/0_1", "nwparser.payload", "Failed %{p0}");

var select215 = linear_select([
	part986,
	part987,
]);

var part988 = match("MESSAGE#585:00034/2_0", "nwparser.p0", "bind %{p0}");

var part989 = match("MESSAGE#585:00034/2_2", "nwparser.p0", "retrieve %{p0}");

var select216 = linear_select([
	part988,
	dup201,
	part989,
]);

var select217 = linear_select([
	dup196,
	dup103,
	dup163,
]);

var part990 = match("MESSAGE#585:00034/5", "nwparser.p0", "SSH user %{username}. (Key ID=%{fld2})");

var all203 = all_match({
	processors: [
		select215,
		dup103,
		select216,
		dup202,
		select217,
		part990,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg594 = msg("00034", all203);

var part991 = match("MESSAGE#586:00034:01/0_0", "nwparser.payload", "SCS: Incompatible %{p0}");

var part992 = match("MESSAGE#586:00034:01/0_1", "nwparser.payload", "Incompatible %{p0}");

var select218 = linear_select([
	part991,
	part992,
]);

var part993 = match("MESSAGE#586:00034:01/1", "nwparser.p0", "SSH version %{version->} has been received from %{p0}");

var part994 = match("MESSAGE#586:00034:01/2_0", "nwparser.p0", "the SSH %{p0}");

var select219 = linear_select([
	part994,
	dup241,
]);

var part995 = match("MESSAGE#586:00034:01/3", "nwparser.p0", "client at %{saddr}:%{sport}");

var all204 = all_match({
	processors: [
		select218,
		part993,
		select219,
		part995,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg595 = msg("00034:01", all204);

var part996 = match("MESSAGE#587:00034:02", "nwparser.payload", "Maximum number of SCS sessions %{fld2->} has been reached. Connection request from SSH user %{username->} at %{saddr}:%{sport->} has been %{disposition}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg596 = msg("00034:02", part996);

var part997 = match("MESSAGE#588:00034:03/1", "nwparser.p0", "device failed to authenticate the SSH client at %{saddr}:%{sport}");

var all205 = all_match({
	processors: [
		dup384,
		part997,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg597 = msg("00034:03", all205);

var part998 = match("MESSAGE#589:00034:04", "nwparser.payload", "SCS: NetScreen device failed to generate a PKA RSA challenge for SSH user %{username->} at %{saddr}:%{sport}. (Key ID=%{fld2})", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg598 = msg("00034:04", part998);

var part999 = match("MESSAGE#590:00034:05", "nwparser.payload", "NetScreen device failed to generate a PKA RSA challenge for SSH user %{username}. (Key ID=%{fld2})", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg599 = msg("00034:05", part999);

var part1000 = match("MESSAGE#591:00034:06/1", "nwparser.p0", "device failed to %{p0}");

var part1001 = match("MESSAGE#591:00034:06/2_0", "nwparser.p0", "identify itself %{p0}");

var part1002 = match("MESSAGE#591:00034:06/2_1", "nwparser.p0", "send the identification string %{p0}");

var select220 = linear_select([
	part1001,
	part1002,
]);

var part1003 = match("MESSAGE#591:00034:06/3", "nwparser.p0", "to the SSH client at %{saddr}:%{sport}");

var all206 = all_match({
	processors: [
		dup384,
		part1000,
		select220,
		part1003,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg600 = msg("00034:06", all206);

var part1004 = match("MESSAGE#592:00034:07", "nwparser.payload", "SCS connection has been terminated for admin user %{username->} at %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg601 = msg("00034:07", part1004);

var part1005 = match("MESSAGE#593:00034:08", "nwparser.payload", "SCS: SCS has been %{disposition->} for %{username->} with %{fld2->} existing PKA keys already bound to %{fld3->} SSH users.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg602 = msg("00034:08", part1005);

var part1006 = match("MESSAGE#594:00034:09", "nwparser.payload", "SCS has been %{disposition->} for %{username->} with %{fld2->} PKA keys already bound to %{fld3->} SSH users", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg603 = msg("00034:09", part1006);

var part1007 = match("MESSAGE#595:00034:10/2", "nwparser.p0", "%{}client at %{saddr->} has attempted to make an SCS connection to %{p0}");

var part1008 = match("MESSAGE#595:00034:10/4", "nwparser.p0", "%{interface->} %{p0}");

var part1009 = match("MESSAGE#595:00034:10/5_0", "nwparser.p0", "with%{p0}");

var part1010 = match("MESSAGE#595:00034:10/5_1", "nwparser.p0", "at%{p0}");

var select221 = linear_select([
	part1009,
	part1010,
]);

var part1011 = match("MESSAGE#595:00034:10/6", "nwparser.p0", "%{}IP %{hostip->} but %{disposition->} because %{result}");

var all207 = all_match({
	processors: [
		dup244,
		dup385,
		part1007,
		dup352,
		part1008,
		select221,
		part1011,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg604 = msg("00034:10", all207);

var part1012 = match("MESSAGE#596:00034:12/2", "nwparser.p0", "%{}client at %{saddr}:%{sport->} has attempted to make an SCS connection to %{p0}");

var part1013 = match("MESSAGE#596:00034:12/4", "nwparser.p0", "but %{disposition->} because %{result}");

var all208 = all_match({
	processors: [
		dup244,
		dup385,
		part1012,
		dup386,
		part1013,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg605 = msg("00034:12", all208);

var part1014 = match("MESSAGE#597:00034:11/2", "nwparser.p0", "%{}client at %{saddr}:%{sport->} has %{disposition->} to make an SCS connection to %{p0}");

var part1015 = match("MESSAGE#597:00034:11/4", "nwparser.p0", "because %{result}");

var all209 = all_match({
	processors: [
		dup244,
		dup385,
		part1014,
		dup386,
		part1015,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg606 = msg("00034:11", all209);

var part1016 = match("MESSAGE#598:00034:15", "nwparser.payload", "SSH client at %{saddr}:%{sport->} has %{disposition->} to make an SCS connection because %{result}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg607 = msg("00034:15", part1016);

var part1017 = match("MESSAGE#599:00034:18/2", "nwparser.p0", "user %{username->} at %{saddr}:%{sport->} cannot log in via SCS to %{service->} using the shared %{interface->} interface because %{result}");

var all210 = all_match({
	processors: [
		dup244,
		dup387,
		part1017,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg608 = msg("00034:18", all210);

var part1018 = match("MESSAGE#600:00034:20/2", "nwparser.p0", "user %{username->} at %{saddr}:%{sport->} has %{disposition->} the PKA RSA challenge");

var all211 = all_match({
	processors: [
		dup244,
		dup387,
		part1018,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg609 = msg("00034:20", all211);

var part1019 = match("MESSAGE#601:00034:21/2", "nwparser.p0", "user %{username->} at %{saddr}:%{sport->} has requested %{p0}");

var part1020 = match("MESSAGE#601:00034:21/4", "nwparser.p0", "authentication which is not %{p0}");

var part1021 = match("MESSAGE#601:00034:21/5_0", "nwparser.p0", "supported %{p0}");

var select222 = linear_select([
	part1021,
	dup156,
]);

var part1022 = match("MESSAGE#601:00034:21/6", "nwparser.p0", "for that %{p0}");

var part1023 = match("MESSAGE#601:00034:21/7_0", "nwparser.p0", "client%{}");

var part1024 = match("MESSAGE#601:00034:21/7_1", "nwparser.p0", "user%{}");

var select223 = linear_select([
	part1023,
	part1024,
]);

var all212 = all_match({
	processors: [
		dup244,
		dup387,
		part1019,
		dup372,
		part1020,
		select222,
		part1022,
		select223,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg610 = msg("00034:21", all212);

var part1025 = match("MESSAGE#602:00034:22", "nwparser.payload", "SSH user %{username->} at %{saddr}:%{sport->} has unsuccessfully attempted to log in via SCS to vsys %{fld2->} using the shared untrusted interface", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg611 = msg("00034:22", part1025);

var part1026 = match("MESSAGE#603:00034:23/1_0", "nwparser.p0", "SCS: Unable %{p0}");

var part1027 = match("MESSAGE#603:00034:23/1_1", "nwparser.p0", "Unable %{p0}");

var select224 = linear_select([
	part1026,
	part1027,
]);

var part1028 = match("MESSAGE#603:00034:23/2", "nwparser.p0", "to validate cookie from the SSH client at %{saddr}:%{sport}");

var all213 = all_match({
	processors: [
		dup160,
		select224,
		part1028,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg612 = msg("00034:23", all213);

var part1029 = match("MESSAGE#604:00034:24", "nwparser.payload", "AC %{username->} is advertising URL %{fld2}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg613 = msg("00034:24", part1029);

var part1030 = match("MESSAGE#605:00034:25", "nwparser.payload", "Message from AC %{username}: %{fld2}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg614 = msg("00034:25", part1030);

var part1031 = match("MESSAGE#606:00034:26", "nwparser.payload", "PPPoE Settings changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg615 = msg("00034:26", part1031);

var part1032 = match("MESSAGE#607:00034:27", "nwparser.payload", "PPPoE is %{disposition->} on %{interface->} interface", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg616 = msg("00034:27", part1032);

var part1033 = match("MESSAGE#608:00034:28", "nwparser.payload", "PPPoE%{quote}s session closed by AC", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg617 = msg("00034:28", part1033);

var part1034 = match("MESSAGE#609:00034:29", "nwparser.payload", "SCS: Disabled for %{username}. Attempted connection %{disposition->} from %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg618 = msg("00034:29", part1034);

var part1035 = match("MESSAGE#610:00034:30", "nwparser.payload", "SCS: %{disposition->} to remove PKA key removed.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg619 = msg("00034:30", part1035);

var part1036 = match("MESSAGE#611:00034:31", "nwparser.payload", "SCS: %{disposition->} to retrieve host key", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg620 = msg("00034:31", part1036);

var part1037 = match("MESSAGE#612:00034:32", "nwparser.payload", "SCS: %{disposition->} to send identification string to client host at %{saddr}:%{sport}.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg621 = msg("00034:32", part1037);

var part1038 = match("MESSAGE#613:00034:33", "nwparser.payload", "SCS: Max %{fld2->} sessions reached unabel to accept connection : %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg622 = msg("00034:33", part1038);

var part1039 = match("MESSAGE#614:00034:34", "nwparser.payload", "SCS: Maximum number for SCS sessions %{fld2->} has been reached. Connection request from SSH user at %{saddr}:%{sport->} has been %{disposition}.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg623 = msg("00034:34", part1039);

var part1040 = match("MESSAGE#615:00034:35", "nwparser.payload", "SCS: SSH user %{username->} at %{saddr}:%{sport->} has unsuccessfully attempted to log in via SCS to %{service->} using the shared untrusted interface because SCS is disabled on that interface.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg624 = msg("00034:35", part1040);

var part1041 = match("MESSAGE#616:00034:36", "nwparser.payload", "SCS: Unsupported cipher type %{fld2->} requested from: %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg625 = msg("00034:36", part1041);

var part1042 = match("MESSAGE#617:00034:37", "nwparser.payload", "The Point-to-Point Protocol over Ethernet (PPPoE) protocol settings changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg626 = msg("00034:37", part1042);

var part1043 = match("MESSAGE#618:00034:38", "nwparser.payload", "SSH: %{disposition->} to retreive PKA key bound to SSH user %{username->} (Key ID %{fld2})", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg627 = msg("00034:38", part1043);

var part1044 = match("MESSAGE#619:00034:39", "nwparser.payload", "SSH: Error processing packet from host %{saddr->} (Code %{fld2})", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg628 = msg("00034:39", part1044);

var part1045 = match("MESSAGE#620:00034:40", "nwparser.payload", "SSH: Device failed to send initialization string to client at %{saddr}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg629 = msg("00034:40", part1045);

var part1046 = match("MESSAGE#621:00034:41/0", "nwparser.payload", "SCP: Admin user '%{administrator}' attempted to transfer file %{p0}");

var part1047 = match("MESSAGE#621:00034:41/2", "nwparser.p0", "the device with insufficient privilege.%{}");

var all214 = all_match({
	processors: [
		part1046,
		dup373,
		part1047,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg630 = msg("00034:41", all214);

var part1048 = match("MESSAGE#622:00034:42", "nwparser.payload", "SSH: Maximum number of SSH sessions (%{fld2}) exceeded. Connection request from SSH user %{username->} at %{saddr->} denied.", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg631 = msg("00034:42", part1048);

var part1049 = match("MESSAGE#623:00034:43", "nwparser.payload", "Ethernet driver ran out of rx bd (port %{network_port})", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg632 = msg("00034:43", part1049);

var part1050 = match("MESSAGE#1224:00034:44", "nwparser.payload", "Potential replay attack detected on SSH connection initiated from %{saddr}:%{sport->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg633 = msg("00034:44", part1050);

var select225 = linear_select([
	msg594,
	msg595,
	msg596,
	msg597,
	msg598,
	msg599,
	msg600,
	msg601,
	msg602,
	msg603,
	msg604,
	msg605,
	msg606,
	msg607,
	msg608,
	msg609,
	msg610,
	msg611,
	msg612,
	msg613,
	msg614,
	msg615,
	msg616,
	msg617,
	msg618,
	msg619,
	msg620,
	msg621,
	msg622,
	msg623,
	msg624,
	msg625,
	msg626,
	msg627,
	msg628,
	msg629,
	msg630,
	msg631,
	msg632,
	msg633,
]);

var part1051 = match("MESSAGE#624:00035", "nwparser.payload", "PKI Verify Error: %{resultcode}:%{result}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg634 = msg("00035", part1051);

var part1052 = match("MESSAGE#625:00035:01", "nwparser.payload", "SSL - Error MessageID in incoming mail - %{fld2}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg635 = msg("00035:01", part1052);

var part1053 = match("MESSAGE#626:00035:02", "nwparser.payload", "SSL - cipher type %{fld2->} is not allowed in export or firewall only system", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg636 = msg("00035:02", part1053);

var part1054 = match("MESSAGE#627:00035:03", "nwparser.payload", "SSL CA changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg637 = msg("00035:03", part1054);

var part1055 = match("MESSAGE#628:00035:04/0", "nwparser.payload", "SSL Error when retrieve local c%{p0}");

var part1056 = match("MESSAGE#628:00035:04/1_0", "nwparser.p0", "a(verify) %{p0}");

var part1057 = match("MESSAGE#628:00035:04/1_1", "nwparser.p0", "ert(verify) %{p0}");

var part1058 = match("MESSAGE#628:00035:04/1_2", "nwparser.p0", "ert(all) %{p0}");

var select226 = linear_select([
	part1056,
	part1057,
	part1058,
]);

var part1059 = match("MESSAGE#628:00035:04/2", "nwparser.p0", ": %{fld2}");

var all215 = all_match({
	processors: [
		part1055,
		select226,
		part1059,
	],
	on_success: processor_chain([
		dup117,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg638 = msg("00035:04", all215);

var part1060 = match("MESSAGE#629:00035:05", "nwparser.payload", "SSL No ssl context. Not ready for connections.%{}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg639 = msg("00035:05", part1060);

var part1061 = match("MESSAGE#630:00035:06/0", "nwparser.payload", "SSL c%{p0}");

var part1062 = match("MESSAGE#630:00035:06/2", "nwparser.p0", "changed to none%{}");

var all216 = all_match({
	processors: [
		part1061,
		dup388,
		part1062,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg640 = msg("00035:06", all216);

var part1063 = match("MESSAGE#631:00035:07", "nwparser.payload", "SSL cert subject mismatch: %{fld2->} recieved %{fld3->} is expected", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg641 = msg("00035:07", part1063);

var part1064 = match("MESSAGE#632:00035:08", "nwparser.payload", "SSL certificate changed%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg642 = msg("00035:08", part1064);

var part1065 = match("MESSAGE#633:00035:09/1_0", "nwparser.p0", "enabled%{}");

var select227 = linear_select([
	part1065,
	dup92,
]);

var all217 = all_match({
	processors: [
		dup253,
		select227,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg643 = msg("00035:09", all217);

var part1066 = match("MESSAGE#634:00035:10/0", "nwparser.payload", "SSL memory allocation fails in process_c%{p0}");

var part1067 = match("MESSAGE#634:00035:10/1_0", "nwparser.p0", "a()%{}");

var part1068 = match("MESSAGE#634:00035:10/1_1", "nwparser.p0", "ert()%{}");

var select228 = linear_select([
	part1067,
	part1068,
]);

var all218 = all_match({
	processors: [
		part1066,
		select228,
	],
	on_success: processor_chain([
		dup184,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg644 = msg("00035:10", all218);

var part1069 = match("MESSAGE#635:00035:11/0", "nwparser.payload", "SSL no ssl c%{p0}");

var part1070 = match("MESSAGE#635:00035:11/1_0", "nwparser.p0", "a%{}");

var part1071 = match("MESSAGE#635:00035:11/1_1", "nwparser.p0", "ert%{}");

var select229 = linear_select([
	part1070,
	part1071,
]);

var all219 = all_match({
	processors: [
		part1069,
		select229,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg645 = msg("00035:11", all219);

var part1072 = match("MESSAGE#636:00035:12/0", "nwparser.payload", "SSL set c%{p0}");

var part1073 = match("MESSAGE#636:00035:12/2", "nwparser.p0", "id is invalid %{fld2}");

var all220 = all_match({
	processors: [
		part1072,
		dup388,
		part1073,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg646 = msg("00035:12", all220);

var part1074 = match("MESSAGE#637:00035:13/1_1", "nwparser.p0", "verify %{p0}");

var select230 = linear_select([
	dup101,
	part1074,
]);

var part1075 = match("MESSAGE#637:00035:13/2", "nwparser.p0", "cert failed. Key type is not RSA%{}");

var all221 = all_match({
	processors: [
		dup253,
		select230,
		part1075,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg647 = msg("00035:13", all221);

var part1076 = match("MESSAGE#638:00035:14", "nwparser.payload", "SSL ssl context init failed%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg648 = msg("00035:14", part1076);

var part1077 = match("MESSAGE#639:00035:15/0", "nwparser.payload", "%{change_attribute->} has been changed %{p0}");

var part1078 = match("MESSAGE#639:00035:15/1_0", "nwparser.p0", "from %{change_old->} to %{change_new}");

var part1079 = match("MESSAGE#639:00035:15/1_1", "nwparser.p0", "to %{fld2}");

var select231 = linear_select([
	part1078,
	part1079,
]);

var all222 = all_match({
	processors: [
		part1077,
		select231,
	],
	on_success: processor_chain([
		dup184,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg649 = msg("00035:15", all222);

var part1080 = match("MESSAGE#640:00035:16", "nwparser.payload", "web SSL certificate changed to by %{username->} via web from host %{saddr->} to %{daddr}:%{dport->} %{fld5}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg650 = msg("00035:16", part1080);

var select232 = linear_select([
	msg634,
	msg635,
	msg636,
	msg637,
	msg638,
	msg639,
	msg640,
	msg641,
	msg642,
	msg643,
	msg644,
	msg645,
	msg646,
	msg647,
	msg648,
	msg649,
	msg650,
]);

var part1081 = match("MESSAGE#641:00036", "nwparser.payload", "An optional ScreenOS feature has been activated via a software key%{}", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg651 = msg("00036", part1081);

var part1082 = match("MESSAGE#642:00036:01/0", "nwparser.payload", "%{fld2->} license keys were updated successfully by %{p0}");

var part1083 = match("MESSAGE#642:00036:01/1_1", "nwparser.p0", "manual %{p0}");

var select233 = linear_select([
	dup214,
	part1083,
]);

var part1084 = match("MESSAGE#642:00036:01/2", "nwparser.p0", "retrieval%{}");

var all223 = all_match({
	processors: [
		part1082,
		select233,
		part1084,
	],
	on_success: processor_chain([
		dup254,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg652 = msg("00036:01", all223);

var select234 = linear_select([
	msg651,
	msg652,
]);

var part1085 = match("MESSAGE#643:00037/0", "nwparser.payload", "Intra-zone block for zone %{zone->} was set to o%{p0}");

var part1086 = match("MESSAGE#643:00037/1_0", "nwparser.p0", "n%{}");

var part1087 = match("MESSAGE#643:00037/1_1", "nwparser.p0", "ff%{}");

var select235 = linear_select([
	part1086,
	part1087,
]);

var all224 = all_match({
	processors: [
		part1085,
		select235,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg653 = msg("00037", all224);

var part1088 = match("MESSAGE#644:00037:01/0", "nwparser.payload", "New zone %{zone->} ( %{p0}");

var select236 = linear_select([
	dup255,
	dup256,
]);

var part1089 = match("MESSAGE#644:00037:01/2", "nwparser.p0", "%{fld2}) was created.%{p0}");

var all225 = all_match({
	processors: [
		part1088,
		select236,
		part1089,
		dup351,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg654 = msg("00037:01", all225);

var part1090 = match("MESSAGE#645:00037:02", "nwparser.payload", "Tunnel zone %{src_zone->} was bound to out zone %{dst_zone}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg655 = msg("00037:02", part1090);

var part1091 = match("MESSAGE#646:00037:03/1_0", "nwparser.p0", "was was %{p0}");

var part1092 = match("MESSAGE#646:00037:03/1_1", "nwparser.p0", "%{zone->} was %{p0}");

var select237 = linear_select([
	part1091,
	part1092,
]);

var part1093 = match("MESSAGE#646:00037:03/3", "nwparser.p0", "virtual router %{p0}");

var part1094 = match("MESSAGE#646:00037:03/4_0", "nwparser.p0", "%{node->} (%{fld1})");

var part1095 = match("MESSAGE#646:00037:03/4_1", "nwparser.p0", "%{node}.");

var select238 = linear_select([
	part1094,
	part1095,
]);

var all226 = all_match({
	processors: [
		dup113,
		select237,
		dup371,
		part1093,
		select238,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg656 = msg("00037:03", all226);

var part1096 = match("MESSAGE#647:00037:04", "nwparser.payload", "Zone %{zone->} was changed to non-shared.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg657 = msg("00037:04", part1096);

var part1097 = match("MESSAGE#648:00037:05/0", "nwparser.payload", "Zone %{zone->} ( %{p0}");

var select239 = linear_select([
	dup256,
	dup255,
]);

var part1098 = match("MESSAGE#648:00037:05/2", "nwparser.p0", "%{fld2}) was deleted. %{p0}");

var part1099 = match_copy("MESSAGE#648:00037:05/3_1", "nwparser.p0", "space");

var select240 = linear_select([
	dup10,
	part1099,
]);

var all227 = all_match({
	processors: [
		part1097,
		select239,
		part1098,
		select240,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg658 = msg("00037:05", all227);

var part1100 = match("MESSAGE#649:00037:06", "nwparser.payload", "IP/TCP reassembly for ALG was %{disposition->} on zone %{zone}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg659 = msg("00037:06", part1100);

var select241 = linear_select([
	msg653,
	msg654,
	msg655,
	msg656,
	msg657,
	msg658,
	msg659,
]);

var part1101 = match("MESSAGE#650:00038/0", "nwparser.payload", "OSPF routing instance in vrouter %{p0}");

var part1102 = match("MESSAGE#650:00038/1_0", "nwparser.p0", "%{node->} is %{p0}");

var part1103 = match("MESSAGE#650:00038/1_1", "nwparser.p0", "%{node->} %{p0}");

var select242 = linear_select([
	part1102,
	part1103,
]);

var all228 = all_match({
	processors: [
		part1101,
		select242,
		dup36,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg660 = msg("00038", all228);

var part1104 = match("MESSAGE#651:00039", "nwparser.payload", "BGP instance name created for vr %{node}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg661 = msg("00039", part1104);

var part1105 = match("MESSAGE#652:00040/0_0", "nwparser.payload", "Low watermark%{p0}");

var part1106 = match("MESSAGE#652:00040/0_1", "nwparser.payload", "High watermark%{p0}");

var select243 = linear_select([
	part1105,
	part1106,
]);

var part1107 = match("MESSAGE#652:00040/1", "nwparser.p0", "%{}for early aging has been changed to the default %{fld2}");

var all229 = all_match({
	processors: [
		select243,
		part1107,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg662 = msg("00040", all229);

var part1108 = match("MESSAGE#653:00040:01", "nwparser.payload", "VPN '%{group}' from %{daddr->} is %{disposition->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg663 = msg("00040:01", part1108);

var select244 = linear_select([
	msg662,
	msg663,
]);

var part1109 = match("MESSAGE#654:00041", "nwparser.payload", "A route-map name in virtual router %{node->} has been removed", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg664 = msg("00041", part1109);

var part1110 = match("MESSAGE#655:00041:01", "nwparser.payload", "VPN '%{group}' from %{daddr->} is %{disposition->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg665 = msg("00041:01", part1110);

var select245 = linear_select([
	msg664,
	msg665,
]);

var part1111 = match("MESSAGE#656:00042", "nwparser.payload", "Replay packet detected on IPSec tunnel on %{interface->} with tunnel ID %{fld2}! From %{saddr->} to %{daddr}/%{dport}, %{info->} (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg666 = msg("00042", part1111);

var part1112 = match("MESSAGE#657:00042:01", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup9,
	dup4,
	dup5,
	dup60,
]));

var msg667 = msg("00042:01", part1112);

var select246 = linear_select([
	msg666,
	msg667,
]);

var part1113 = match("MESSAGE#658:00043", "nwparser.payload", "Receive StopCCN_msg, remove l2tp tunnel (%{fld2}-%{fld3}), Result code %{resultcode->} (%{result}). (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg668 = msg("00043", part1113);

var part1114 = match("MESSAGE#659:00044/0", "nwparser.payload", "access list %{listnum->} sequence number %{fld3->} %{p0}");

var part1115 = match("MESSAGE#659:00044/1_1", "nwparser.p0", "deny %{p0}");

var select247 = linear_select([
	dup257,
	part1115,
]);

var part1116 = match("MESSAGE#659:00044/2", "nwparser.p0", "ip %{hostip}/%{mask->} %{disposition->} in vrouter %{node}");

var all230 = all_match({
	processors: [
		part1114,
		select247,
		part1116,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg669 = msg("00044", all230);

var part1117 = match("MESSAGE#660:00044:01", "nwparser.payload", "access list %{listnum->} %{disposition->} in vrouter %{node}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg670 = msg("00044:01", part1117);

var select248 = linear_select([
	msg669,
	msg670,
]);

var part1118 = match("MESSAGE#661:00045", "nwparser.payload", "RIP instance in virtual router %{node->} was %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg671 = msg("00045", part1118);

var part1119 = match("MESSAGE#662:00047/1_0", "nwparser.p0", "remove %{p0}");

var part1120 = match("MESSAGE#662:00047/1_1", "nwparser.p0", "add %{p0}");

var select249 = linear_select([
	part1119,
	part1120,
]);

var part1121 = match("MESSAGE#662:00047/2", "nwparser.p0", "multicast policy from %{src_zone->} %{fld4->} to %{dst_zone->} %{fld3->} (%{fld1})");

var all231 = all_match({
	processors: [
		dup183,
		select249,
		part1121,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg672 = msg("00047", all231);

var part1122 = match("MESSAGE#663:00048/0", "nwparser.payload", "Access list entry %{listnum->} with %{p0}");

var part1123 = match("MESSAGE#663:00048/1_0", "nwparser.p0", "a sequence %{p0}");

var part1124 = match("MESSAGE#663:00048/1_1", "nwparser.p0", "sequence %{p0}");

var select250 = linear_select([
	part1123,
	part1124,
]);

var part1125 = match("MESSAGE#663:00048/2", "nwparser.p0", "number %{fld2->} %{p0}");

var part1126 = match("MESSAGE#663:00048/3_0", "nwparser.p0", "with an action of %{p0}");

var select251 = linear_select([
	part1126,
	dup112,
]);

var part1127 = match("MESSAGE#663:00048/5_0", "nwparser.p0", "with an IP %{p0}");

var select252 = linear_select([
	part1127,
	dup139,
]);

var part1128 = match("MESSAGE#663:00048/6", "nwparser.p0", "address %{p0}");

var part1129 = match("MESSAGE#663:00048/7_0", "nwparser.p0", "and subnetwork mask of %{p0}");

var select253 = linear_select([
	part1129,
	dup16,
]);

var part1130 = match("MESSAGE#663:00048/8", "nwparser.p0", "%{} %{fld3}was %{p0}");

var part1131 = match("MESSAGE#663:00048/9_0", "nwparser.p0", "created on %{p0}");

var select254 = linear_select([
	part1131,
	dup129,
]);

var part1132 = match("MESSAGE#663:00048/10", "nwparser.p0", "virtual router %{node->} (%{fld1})");

var all232 = all_match({
	processors: [
		part1122,
		select250,
		part1125,
		select251,
		dup257,
		select252,
		part1128,
		select253,
		part1130,
		select254,
		part1132,
	],
	on_success: processor_chain([
		setc("eventcategory","1501000000"),
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg673 = msg("00048", all232);

var part1133 = match("MESSAGE#664:00048:01/0", "nwparser.payload", "Route %{p0}");

var part1134 = match("MESSAGE#664:00048:01/1_0", "nwparser.p0", "map entry %{p0}");

var part1135 = match("MESSAGE#664:00048:01/1_1", "nwparser.p0", "entry %{p0}");

var select255 = linear_select([
	part1134,
	part1135,
]);

var part1136 = match("MESSAGE#664:00048:01/2", "nwparser.p0", "with sequence number %{fld2->} in route map binck-ospf%{p0}");

var part1137 = match("MESSAGE#664:00048:01/3_0", "nwparser.p0", " in %{p0}");

var select256 = linear_select([
	part1137,
	dup105,
]);

var part1138 = match("MESSAGE#664:00048:01/4", "nwparser.p0", "virtual router %{node->} was %{disposition->} (%{fld1})");

var all233 = all_match({
	processors: [
		part1133,
		select255,
		part1136,
		select256,
		part1138,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg674 = msg("00048:01", all233);

var part1139 = match("MESSAGE#665:00048:02", "nwparser.payload", "%{space}set match interface %{interface->} (%{fld1})", processor_chain([
	dup209,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg675 = msg("00048:02", part1139);

var select257 = linear_select([
	msg673,
	msg674,
	msg675,
]);

var part1140 = match("MESSAGE#666:00049", "nwparser.payload", "Route-lookup preference changed to %{fld8->} (%{fld2}) => %{fld3->} (%{fld4}) => %{fld5->} (%{fld6}) in virtual router (%{node})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg676 = msg("00049", part1140);

var part1141 = match("MESSAGE#667:00049:01", "nwparser.payload", "SIBR routing %{disposition->} in virtual router %{node}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg677 = msg("00049:01", part1141);

var part1142 = match("MESSAGE#668:00049:02", "nwparser.payload", "A virtual router with name %{node->} and ID %{fld2->} has been removed", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg678 = msg("00049:02", part1142);

var part1143 = match("MESSAGE#669:00049:03", "nwparser.payload", "The router-id of virtual router \"%{node}\" used by OSPF, BGP routing instances id has been uninitialized. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg679 = msg("00049:03", part1143);

var part1144 = match("MESSAGE#670:00049:04", "nwparser.payload", "The system default-route through virtual router \"%{node}\" has been added in virtual router \"%{fld4}\" (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg680 = msg("00049:04", part1144);

var part1145 = match("MESSAGE#671:00049:05", "nwparser.payload", "Subnetwork conflict checking for interfaces in virtual router (%{node}) has been enabled. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg681 = msg("00049:05", part1145);

var select258 = linear_select([
	msg676,
	msg677,
	msg678,
	msg679,
	msg680,
	msg681,
]);

var part1146 = match("MESSAGE#672:00050", "nwparser.payload", "Track IP enabled (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg682 = msg("00050", part1146);

var part1147 = match("MESSAGE#673:00051", "nwparser.payload", "Session utilization has reached %{fld2}, which is %{fld3->} of the system capacity!", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg683 = msg("00051", part1147);

var part1148 = match("MESSAGE#674:00052", "nwparser.payload", "AV: Suspicious client %{saddr}:%{sport}->%{daddr}:%{dport->} used %{fld2->} percent of AV resources, which exceeded the max of %{fld3->} percent.", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg684 = msg("00052", part1148);

var part1149 = match("MESSAGE#675:00055/1_1", "nwparser.p0", "router %{p0}");

var select259 = linear_select([
	dup169,
	part1149,
]);

var part1150 = match("MESSAGE#675:00055/2", "nwparser.p0", "instance was %{disposition->} on interface %{interface}.");

var all234 = all_match({
	processors: [
		dup258,
		select259,
		part1150,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg685 = msg("00055", all234);

var part1151 = match("MESSAGE#676:00055:01/1_0", "nwparser.p0", "proxy %{p0}");

var part1152 = match("MESSAGE#676:00055:01/1_1", "nwparser.p0", "function %{p0}");

var select260 = linear_select([
	part1151,
	part1152,
]);

var part1153 = match("MESSAGE#676:00055:01/2", "nwparser.p0", "was %{disposition->} on interface %{interface}.");

var all235 = all_match({
	processors: [
		dup258,
		select260,
		part1153,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg686 = msg("00055:01", all235);

var part1154 = match("MESSAGE#677:00055:02/2", "nwparser.p0", "same subnet check on interface %{interface}.");

var all236 = all_match({
	processors: [
		dup259,
		dup389,
		part1154,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg687 = msg("00055:02", all236);

var part1155 = match("MESSAGE#678:00055:03/2", "nwparser.p0", "router alert IP option check on interface %{interface}.");

var all237 = all_match({
	processors: [
		dup259,
		dup389,
		part1155,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg688 = msg("00055:03", all237);

var part1156 = match("MESSAGE#679:00055:04", "nwparser.payload", "IGMP version was changed to %{version->} on interface %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg689 = msg("00055:04", part1156);

var part1157 = match("MESSAGE#680:00055:05/0", "nwparser.payload", "IGMP query %{p0}");

var part1158 = match("MESSAGE#680:00055:05/1_1", "nwparser.p0", "max response time %{p0}");

var select261 = linear_select([
	dup110,
	part1158,
]);

var part1159 = match("MESSAGE#680:00055:05/2", "nwparser.p0", "was changed to %{fld2->} on interface %{interface}");

var all238 = all_match({
	processors: [
		part1157,
		select261,
		part1159,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg690 = msg("00055:05", all238);

var part1160 = match("MESSAGE#681:00055:06/0", "nwparser.payload", "IGMP l%{p0}");

var part1161 = match("MESSAGE#681:00055:06/1_0", "nwparser.p0", "eave %{p0}");

var part1162 = match("MESSAGE#681:00055:06/1_1", "nwparser.p0", "ast member query %{p0}");

var select262 = linear_select([
	part1161,
	part1162,
]);

var part1163 = match("MESSAGE#681:00055:06/2", "nwparser.p0", "interval was changed to %{fld2->} on interface %{interface}.");

var all239 = all_match({
	processors: [
		part1160,
		select262,
		part1163,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg691 = msg("00055:06", all239);

var part1164 = match("MESSAGE#682:00055:07/1_0", "nwparser.p0", "routers %{p0}");

var part1165 = match("MESSAGE#682:00055:07/1_1", "nwparser.p0", "hosts %{p0}");

var part1166 = match("MESSAGE#682:00055:07/1_2", "nwparser.p0", "groups %{p0}");

var select263 = linear_select([
	part1164,
	part1165,
	part1166,
]);

var part1167 = match("MESSAGE#682:00055:07/2", "nwparser.p0", "accept list ID was changed to %{fld2->} on interface %{interface}.");

var all240 = all_match({
	processors: [
		dup258,
		select263,
		part1167,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg692 = msg("00055:07", all240);

var part1168 = match("MESSAGE#683:00055:08/1_0", "nwparser.p0", "all groups %{p0}");

var part1169 = match("MESSAGE#683:00055:08/1_1", "nwparser.p0", "group %{p0}");

var select264 = linear_select([
	part1168,
	part1169,
]);

var part1170 = match("MESSAGE#683:00055:08/2", "nwparser.p0", "%{group->} static flag was %{disposition->} on interface %{interface}.");

var all241 = all_match({
	processors: [
		dup258,
		select264,
		part1170,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg693 = msg("00055:08", all241);

var part1171 = match("MESSAGE#684:00055:09", "nwparser.payload", "IGMP static group %{group->} was added on interface %{interface}", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg694 = msg("00055:09", part1171);

var part1172 = match("MESSAGE#685:00055:10", "nwparser.payload", "IGMP proxy always is %{disposition->} on interface %{interface}.", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg695 = msg("00055:10", part1172);

var select265 = linear_select([
	msg685,
	msg686,
	msg687,
	msg688,
	msg689,
	msg690,
	msg691,
	msg692,
	msg693,
	msg694,
	msg695,
]);

var part1173 = match("MESSAGE#686:00056", "nwparser.payload", "Remove multicast policy from %{src_zone->} %{saddr->} to %{dst_zone->} %{daddr}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg696 = msg("00056", part1173);

var part1174 = match("MESSAGE#687:00057", "nwparser.payload", "%{fld2}: static multicast route src=%{saddr}, grp=%{group->} input ifp = %{sinterface->} output ifp = %{dinterface->} added", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg697 = msg("00057", part1174);

var part1175 = match("MESSAGE#688:00058", "nwparser.payload", "PIMSM protocol configured on interface %{interface}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg698 = msg("00058", part1175);

var part1176 = match("MESSAGE#689:00059/0", "nwparser.payload", "DDNS module is %{p0}");

var part1177 = match("MESSAGE#689:00059/1_0", "nwparser.p0", "initialized %{p0}");

var select266 = linear_select([
	part1177,
	dup262,
	dup157,
	dup156,
]);

var all242 = all_match({
	processors: [
		part1176,
		select266,
		dup116,
	],
	on_success: processor_chain([
		dup209,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg699 = msg("00059", all242);

var part1178 = match("MESSAGE#690:00059:02/0", "nwparser.payload", "DDNS entry with id %{fld2->} is configured with server type \"%{fld3}\" name \"%{hostname}\" refresh-interval %{fld5->} hours minimum update interval %{fld6->} minutes with %{p0}");

var part1179 = match("MESSAGE#690:00059:02/1_0", "nwparser.p0", "secure %{p0}");

var part1180 = match("MESSAGE#690:00059:02/1_1", "nwparser.p0", "clear-text %{p0}");

var select267 = linear_select([
	part1179,
	part1180,
]);

var part1181 = match("MESSAGE#690:00059:02/2", "nwparser.p0", "secure connection.%{}");

var all243 = all_match({
	processors: [
		part1178,
		select267,
		part1181,
	],
	on_success: processor_chain([
		dup209,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg700 = msg("00059:02", all243);

var part1182 = match("MESSAGE#691:00059:03", "nwparser.payload", "DDNS entry with id %{fld2->} is configured with user name \"%{username}\" agent \"%{fld3}\"", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg701 = msg("00059:03", part1182);

var part1183 = match("MESSAGE#692:00059:04", "nwparser.payload", "DDNS entry with id %{fld2->} is configured with interface \"%{interface}\" host-name \"%{hostname}\"", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg702 = msg("00059:04", part1183);

var part1184 = match("MESSAGE#693:00059:05/0_0", "nwparser.payload", "Hostname %{p0}");

var part1185 = match("MESSAGE#693:00059:05/0_1", "nwparser.payload", "Source interface %{p0}");

var part1186 = match("MESSAGE#693:00059:05/0_2", "nwparser.payload", "Username and password %{p0}");

var part1187 = match("MESSAGE#693:00059:05/0_3", "nwparser.payload", "Server %{p0}");

var select268 = linear_select([
	part1184,
	part1185,
	part1186,
	part1187,
]);

var part1188 = match("MESSAGE#693:00059:05/1", "nwparser.p0", "of DDNS entry with id %{fld2->} is cleared.");

var all244 = all_match({
	processors: [
		select268,
		part1188,
	],
	on_success: processor_chain([
		dup209,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg703 = msg("00059:05", all244);

var part1189 = match("MESSAGE#694:00059:06", "nwparser.payload", "Agent of DDNS entry with id %{fld2->} is reset to its default value.", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg704 = msg("00059:06", part1189);

var part1190 = match("MESSAGE#695:00059:07", "nwparser.payload", "Updates for DDNS entry with id %{fld2->} are set to be sent in secure (%{protocol}) mode.", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg705 = msg("00059:07", part1190);

var part1191 = match("MESSAGE#696:00059:08/0_0", "nwparser.payload", "Refresh %{p0}");

var part1192 = match("MESSAGE#696:00059:08/0_1", "nwparser.payload", "Minimum update %{p0}");

var select269 = linear_select([
	part1191,
	part1192,
]);

var part1193 = match("MESSAGE#696:00059:08/1", "nwparser.p0", "interval of DDNS entry with id %{fld2->} is set to default value (%{fld3}).");

var all245 = all_match({
	processors: [
		select269,
		part1193,
	],
	on_success: processor_chain([
		dup209,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg706 = msg("00059:08", all245);

var part1194 = match("MESSAGE#697:00059:09/1_0", "nwparser.p0", "No-Change %{p0}");

var part1195 = match("MESSAGE#697:00059:09/1_1", "nwparser.p0", "Error %{p0}");

var select270 = linear_select([
	part1194,
	part1195,
]);

var part1196 = match("MESSAGE#697:00059:09/2", "nwparser.p0", "response received for DDNS entry update for id %{fld2->} user \"%{username}\" domain \"%{domain}\" server type \" d%{p0}");

var part1197 = match("MESSAGE#697:00059:09/3_1", "nwparser.p0", "yndns %{p0}");

var select271 = linear_select([
	dup261,
	part1197,
]);

var part1198 = match("MESSAGE#697:00059:09/4", "nwparser.p0", "\", server name \"%{hostname}\"");

var all246 = all_match({
	processors: [
		dup160,
		select270,
		part1196,
		select271,
		part1198,
	],
	on_success: processor_chain([
		dup209,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg707 = msg("00059:09", all246);

var part1199 = match("MESSAGE#698:00059:01", "nwparser.payload", "DDNS entry with id %{fld2->} is %{disposition}.", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg708 = msg("00059:01", part1199);

var select272 = linear_select([
	msg699,
	msg700,
	msg701,
	msg702,
	msg703,
	msg704,
	msg705,
	msg706,
	msg707,
	msg708,
]);

var part1200 = match("MESSAGE#699:00062:01", "nwparser.payload", "Track IP IP address %{hostip->} failed. (%{event_time_string})", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("event_description","Track IP failed"),
]));

var msg709 = msg("00062:01", part1200);

var part1201 = match("MESSAGE#700:00062:02", "nwparser.payload", "Track IP failure reached threshold. (%{event_time_string})", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("event_description","Track IP failure reached threshold"),
]));

var msg710 = msg("00062:02", part1201);

var part1202 = match("MESSAGE#701:00062:03", "nwparser.payload", "Track IP IP address %{hostip->} succeeded. (%{event_time_string})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("event_description","Track IP succeeded"),
]));

var msg711 = msg("00062:03", part1202);

var part1203 = match("MESSAGE#702:00062", "nwparser.payload", "HA linkdown%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg712 = msg("00062", part1203);

var select273 = linear_select([
	msg709,
	msg710,
	msg711,
	msg712,
]);

var part1204 = match("MESSAGE#703:00063", "nwparser.payload", "nsrp track-ip ip %{hostip->} %{disposition}!", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg713 = msg("00063", part1204);

var part1205 = match("MESSAGE#704:00064", "nwparser.payload", "Can not create track-ip list%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg714 = msg("00064", part1205);

var part1206 = match("MESSAGE#705:00064:01", "nwparser.payload", "track ip fail reaches threshold system may fail over!%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg715 = msg("00064:01", part1206);

var part1207 = match("MESSAGE#706:00064:02", "nwparser.payload", "Anti-Spam is detached from policy ID %{policy_id}. (%{fld1})", processor_chain([
	dup17,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg716 = msg("00064:02", part1207);

var select274 = linear_select([
	msg714,
	msg715,
	msg716,
]);

var msg717 = msg("00070", dup411);

var part1208 = match("MESSAGE#708:00070:01/2", "nwparser.p0", "%{}Device group %{group->} changed state from %{fld3->} to %{p0}");

var part1209 = match("MESSAGE#708:00070:01/3_0", "nwparser.p0", "Init%{}");

var part1210 = match("MESSAGE#708:00070:01/3_1", "nwparser.p0", "init. (%{fld1})");

var select275 = linear_select([
	part1209,
	part1210,
]);

var all247 = all_match({
	processors: [
		dup267,
		dup391,
		part1208,
		select275,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg718 = msg("00070:01", all247);

var part1211 = match("MESSAGE#709:00070:02", "nwparser.payload", "NSRP: nsrp control channel change to %{interface}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg719 = msg("00070:02", part1211);

var select276 = linear_select([
	msg717,
	msg718,
	msg719,
]);

var msg720 = msg("00071", dup411);

var part1212 = match("MESSAGE#711:00071:01", "nwparser.payload", "The local device %{fld1->} in the Virtual Security Device group %{group->} changed state", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg721 = msg("00071:01", part1212);

var select277 = linear_select([
	msg720,
	msg721,
]);

var msg722 = msg("00072", dup411);

var msg723 = msg("00072:01", dup412);

var select278 = linear_select([
	msg722,
	msg723,
]);

var msg724 = msg("00073", dup411);

var msg725 = msg("00073:01", dup412);

var select279 = linear_select([
	msg724,
	msg725,
]);

var msg726 = msg("00074", dup392);

var all248 = all_match({
	processors: [
		dup263,
		dup390,
		dup271,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg727 = msg("00075", all248);

var part1213 = match("MESSAGE#718:00075:02", "nwparser.payload", "The local device %{hardware_id->} in the Virtual Security Device group %{group->} changed state from %{event_state->} to inoperable. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	setc("event_description","local device in the Virtual Security Device group changed state to inoperable"),
]));

var msg728 = msg("00075:02", part1213);

var part1214 = match("MESSAGE#719:00075:01", "nwparser.payload", "The local device %{hardware_id->} in the Virtual Security Device group %{group->} %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg729 = msg("00075:01", part1214);

var select280 = linear_select([
	msg727,
	msg728,
	msg729,
]);

var msg730 = msg("00076", dup392);

var part1215 = match("MESSAGE#721:00076:01/2", "nwparser.p0", "%{fld2->} of VSD group %{group->} send 2nd path request to unit=%{fld3}");

var all249 = all_match({
	processors: [
		dup263,
		dup390,
		part1215,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg731 = msg("00076:01", all249);

var select281 = linear_select([
	msg730,
	msg731,
]);

var part1216 = match("MESSAGE#722:00077", "nwparser.payload", "HA link disconnect. Begin to use second path of HA%{}", processor_chain([
	dup144,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg732 = msg("00077", part1216);

var all250 = all_match({
	processors: [
		dup263,
		dup390,
		dup271,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg733 = msg("00077:01", all250);

var part1217 = match("MESSAGE#724:00077:02", "nwparser.payload", "The local device %{fld2->} in the Virtual Security Device group %{group}", processor_chain([
	setc("eventcategory","1607000000"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg734 = msg("00077:02", part1217);

var select282 = linear_select([
	msg732,
	msg733,
	msg734,
]);

var part1218 = match("MESSAGE#725:00084", "nwparser.payload", "RTSYNC: NSRP route synchronization is %{disposition}", processor_chain([
	dup272,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg735 = msg("00084", part1218);

var part1219 = match("MESSAGE#726:00090/0_0", "nwparser.payload", "Failover %{p0}");

var part1220 = match("MESSAGE#726:00090/0_1", "nwparser.payload", "Recovery %{p0}");

var select283 = linear_select([
	part1219,
	part1220,
]);

var part1221 = match("MESSAGE#726:00090/3", "nwparser.p0", "untrust interface occurred.%{}");

var all251 = all_match({
	processors: [
		select283,
		dup103,
		dup369,
		part1221,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg736 = msg("00090", all251);

var part1222 = match("MESSAGE#727:00200", "nwparser.payload", "A new route cannot be added to the device because the maximum number of system route entries %{fld2->} has been exceeded", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg737 = msg("00200", part1222);

var part1223 = match("MESSAGE#728:00201", "nwparser.payload", "A route %{hostip}/%{fld2->} cannot be added to the virtual router %{node->} because the number of route entries in the virtual router exceeds the maximum number of routes %{fld3->} allowed", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg738 = msg("00201", part1223);

var part1224 = match("MESSAGE#729:00202", "nwparser.payload", "%{fld2->} hello-packet flood from neighbor (ip = %{hostip->} router-id = %{fld3}) on interface %{interface->} packet is dropped", processor_chain([
	dup272,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg739 = msg("00202", part1224);

var part1225 = match("MESSAGE#730:00203", "nwparser.payload", "%{fld2->} lsa flood on interface %{interface->} has dropped a packet.", processor_chain([
	dup272,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg740 = msg("00203", part1225);

var part1226 = match("MESSAGE#731:00206/0", "nwparser.payload", "The total number of redistributed routes into %{p0}");

var part1227 = match("MESSAGE#731:00206/1_0", "nwparser.p0", "BGP %{p0}");

var part1228 = match("MESSAGE#731:00206/1_1", "nwparser.p0", "OSPF %{p0}");

var select284 = linear_select([
	part1227,
	part1228,
]);

var part1229 = match("MESSAGE#731:00206/2", "nwparser.p0", "in vrouter %{node->} exceeded system limit (%{fld2})");

var all252 = all_match({
	processors: [
		part1226,
		select284,
		part1229,
	],
	on_success: processor_chain([
		dup272,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg741 = msg("00206", all252);

var part1230 = match("MESSAGE#732:00206:01/0", "nwparser.payload", "LSA flood in OSPF with router-id %{fld2->} on %{p0}");

var part1231 = match("MESSAGE#732:00206:01/2", "nwparser.p0", "%{interface->} forced the interface to drop a packet.");

var all253 = all_match({
	processors: [
		part1230,
		dup352,
		part1231,
	],
	on_success: processor_chain([
		dup273,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg742 = msg("00206:01", all253);

var part1232 = match("MESSAGE#733:00206:02/0", "nwparser.payload", "OSPF instance with router-id %{fld3->} received a Hello packet flood from neighbor (IP address %{hostip}, router ID %{fld2}) on %{p0}");

var part1233 = match("MESSAGE#733:00206:02/2", "nwparser.p0", "%{interface->} forcing the interface to drop the packet.");

var all254 = all_match({
	processors: [
		part1232,
		dup352,
		part1233,
	],
	on_success: processor_chain([
		dup273,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg743 = msg("00206:02", all254);

var part1234 = match("MESSAGE#734:00206:03", "nwparser.payload", "Link State Advertisement Id %{fld2}, router ID %{fld3}, type %{fld4->} cannot be deleted from the real-time database in area %{fld5}", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg744 = msg("00206:03", part1234);

var part1235 = match("MESSAGE#735:00206:04", "nwparser.payload", "Reject second OSPF neighbor (%{fld2}) on interface (%{interface}) since it_s configured as point-to-point interface", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg745 = msg("00206:04", part1235);

var select285 = linear_select([
	msg741,
	msg742,
	msg743,
	msg744,
	msg745,
]);

var part1236 = match("MESSAGE#736:00207", "nwparser.payload", "System wide RIP route limit exceeded, RIP route dropped.%{}", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg746 = msg("00207", part1236);

var part1237 = match("MESSAGE#737:00207:01", "nwparser.payload", "%{fld2->} RIP routes dropped from last system wide RIP route limit exceed.", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg747 = msg("00207:01", part1237);

var part1238 = match("MESSAGE#738:00207:02", "nwparser.payload", "RIP database size limit exceeded for %{fld2}, RIP route dropped.", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg748 = msg("00207:02", part1238);

var part1239 = match("MESSAGE#739:00207:03", "nwparser.payload", "%{fld2->} RIP routes dropped from the last database size exceed in vr %{fld3}.", processor_chain([
	dup273,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg749 = msg("00207:03", part1239);

var select286 = linear_select([
	msg746,
	msg747,
	msg748,
	msg749,
]);

var part1240 = match("MESSAGE#740:00257", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=outgoing action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} translated ip=%{stransaddr->} port=%{stransport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup276,
	dup277,
	dup278,
]));

var msg750 = msg("00257", part1240);

var part1241 = match("MESSAGE#741:00257:14", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=incoming action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} translated ip=%{dtransaddr->} port=%{dtransport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup279,
	dup276,
	dup277,
	dup280,
]));

var msg751 = msg("00257:14", part1241);

var part1242 = match("MESSAGE#742:00257:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=outgoing action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} translated ip=%{stransaddr->} port=%{stransport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup282,
	dup278,
]));

var msg752 = msg("00257:01", part1242);

var part1243 = match("MESSAGE#743:00257:15", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=incoming action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} translated ip=%{dtransaddr->} port=%{dtransport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup279,
	dup282,
	dup280,
]));

var msg753 = msg("00257:15", part1243);

var part1244 = match("MESSAGE#744:00257:02", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup276,
	dup277,
]));

var msg754 = msg("00257:02", part1244);

var part1245 = match("MESSAGE#745:00257:03", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup282,
]));

var msg755 = msg("00257:03", part1245);

var part1246 = match("MESSAGE#746:00257:04", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} src-xlated ip=%{stransaddr->} port=%{stransport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup276,
	dup277,
]));

var msg756 = msg("00257:04", part1246);

var part1247 = match("MESSAGE#747:00257:05", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} src-xlated ip=%{stransaddr->} port=%{stransport->} dst-xlated ip=%{dtransaddr->} port=%{dtransport->} session_id=%{sessionid->} reason=%{result}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup282,
]));

var msg757 = msg("00257:05", part1247);

var part1248 = match("MESSAGE#748:00257:19/2", "nwparser.p0", "%{}duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype->} icmp code=%{icmpcode->} src-xlated ip=%{stransaddr->} dst-xlated ip=%{dtransaddr->} session_id=%{sessionid->} reason=%{result}");

var all255 = all_match({
	processors: [
		dup283,
		dup393,
		part1248,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup60,
		dup282,
	]),
});

var msg758 = msg("00257:19", all255);

var part1249 = match("MESSAGE#749:00257:16/2", "nwparser.p0", "%{}duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype->} src-xlated ip=%{stransaddr->} dst-xlated ip=%{dtransaddr->} session_id=%{sessionid}");

var all256 = all_match({
	processors: [
		dup283,
		dup393,
		part1249,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup60,
		dup282,
	]),
});

var msg759 = msg("00257:16", all256);

var part1250 = match("MESSAGE#750:00257:17/2", "nwparser.p0", "%{}duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} src-xlated ip=%{stransaddr->} port=%{stransport->} dst-xlated ip=%{dtransaddr->} port=%{dtransport->} session_id=%{sessionid}");

var all257 = all_match({
	processors: [
		dup283,
		dup393,
		part1250,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup61,
		dup282,
	]),
});

var msg760 = msg("00257:17", all257);

var part1251 = match("MESSAGE#751:00257:18/2", "nwparser.p0", "%{}duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} src-xlated ip=%{stransaddr->} port=%{stransport->} session_id=%{sessionid}");

var all258 = all_match({
	processors: [
		dup283,
		dup393,
		part1251,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup61,
		dup282,
	]),
});

var msg761 = msg("00257:18", all258);

var part1252 = match("MESSAGE#752:00257:06/0", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{p0}");

var part1253 = match("MESSAGE#752:00257:06/1_0", "nwparser.p0", "%{dport->} session_id=%{sessionid}");

var part1254 = match_copy("MESSAGE#752:00257:06/1_1", "nwparser.p0", "dport");

var select287 = linear_select([
	part1253,
	part1254,
]);

var all259 = all_match({
	processors: [
		part1252,
		select287,
	],
	on_success: processor_chain([
		dup185,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup61,
		dup276,
		dup277,
	]),
});

var msg762 = msg("00257:06", all259);

var part1255 = match("MESSAGE#753:00257:07", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup61,
	dup282,
]));

var msg763 = msg("00257:07", part1255);

var part1256 = match("MESSAGE#754:00257:08", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} tcp=%{icmptype}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup60,
	dup276,
	dup277,
]));

var msg764 = msg("00257:08", part1256);

var part1257 = match("MESSAGE#755:00257:09/0", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{p0}");

var part1258 = match("MESSAGE#755:00257:09/1_0", "nwparser.p0", "%{icmptype->} icmp code=%{icmpcode->} session_id=%{sessionid->} reason=%{result}");

var part1259 = match("MESSAGE#755:00257:09/1_1", "nwparser.p0", "%{icmptype->} session_id=%{sessionid}");

var part1260 = match_copy("MESSAGE#755:00257:09/1_2", "nwparser.p0", "icmptype");

var select288 = linear_select([
	part1258,
	part1259,
	part1260,
]);

var all260 = all_match({
	processors: [
		part1257,
		select288,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup60,
		dup282,
	]),
});

var msg765 = msg("00257:09", all260);

var part1261 = match("MESSAGE#756:00257:10/0", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{p0}");

var part1262 = match("MESSAGE#756:00257:10/1_0", "nwparser.p0", "%{daddr->} session_id=%{sessionid}");

var select289 = linear_select([
	part1262,
	dup286,
]);

var all261 = all_match({
	processors: [
		part1261,
		select289,
	],
	on_success: processor_chain([
		dup185,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup60,
		dup276,
		dup277,
	]),
});

var msg766 = msg("00257:10", all261);

var part1263 = match("MESSAGE#757:00257:11/0", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{p0}");

var part1264 = match("MESSAGE#757:00257:11/1_0", "nwparser.p0", "%{daddr->} session_id=%{sessionid->} reason=%{result}");

var select290 = linear_select([
	part1264,
	dup286,
]);

var all262 = all_match({
	processors: [
		part1263,
		select290,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		dup274,
		dup275,
		dup60,
		dup282,
	]),
});

var msg767 = msg("00257:11", all262);

var part1265 = match("MESSAGE#758:00257:12", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} type=%{fld3}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup274,
	dup275,
	dup60,
	dup282,
]));

var msg768 = msg("00257:12", part1265);

var part1266 = match("MESSAGE#759:00257:13", "nwparser.payload", "start_time=\"%{fld2}", processor_chain([
	dup281,
	dup2,
	dup3,
	dup274,
	dup4,
	dup5,
]));

var msg769 = msg("00257:13", part1266);

var select291 = linear_select([
	msg750,
	msg751,
	msg752,
	msg753,
	msg754,
	msg755,
	msg756,
	msg757,
	msg758,
	msg759,
	msg760,
	msg761,
	msg762,
	msg763,
	msg764,
	msg765,
	msg766,
	msg767,
	msg768,
	msg769,
]);

var part1267 = match("MESSAGE#760:00259/1", "nwparser.p0", "user %{username->} has logged on via %{p0}");

var part1268 = match("MESSAGE#760:00259/2_0", "nwparser.p0", "the console %{p0}");

var select292 = linear_select([
	part1268,
	dup289,
	dup241,
]);

var part1269 = match("MESSAGE#760:00259/3", "nwparser.p0", "from %{saddr}:%{sport}");

var all263 = all_match({
	processors: [
		dup394,
		part1267,
		select292,
		part1269,
	],
	on_success: processor_chain([
		dup28,
		dup29,
		dup30,
		dup31,
		dup32,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg770 = msg("00259", all263);

var part1270 = match("MESSAGE#761:00259:07/1", "nwparser.p0", "user %{administrator->} has logged out via %{logon_type->} from %{saddr}:%{sport}");

var all264 = all_match({
	processors: [
		dup394,
		part1270,
	],
	on_success: processor_chain([
		dup33,
		dup29,
		dup34,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg771 = msg("00259:07", all264);

var part1271 = match("MESSAGE#762:00259:01", "nwparser.payload", "Management session via %{logon_type->} from %{saddr}:%{sport->} for [vsys] admin %{administrator->} has timed out", processor_chain([
	dup290,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg772 = msg("00259:01", part1271);

var part1272 = match("MESSAGE#763:00259:02", "nwparser.payload", "Management session via %{logon_type->} for [ vsys ] admin %{administrator->} has timed out", processor_chain([
	dup290,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg773 = msg("00259:02", part1272);

var part1273 = match("MESSAGE#764:00259:03", "nwparser.payload", "Login attempt to system by admin %{administrator->} via the %{logon_type->} has failed", processor_chain([
	dup206,
	dup29,
	dup30,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg774 = msg("00259:03", part1273);

var part1274 = match("MESSAGE#765:00259:04", "nwparser.payload", "Login attempt to system by admin %{administrator->} via %{logon_type->} from %{saddr}:%{sport->} has failed", processor_chain([
	dup206,
	dup29,
	dup30,
	dup31,
	dup54,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg775 = msg("00259:04", part1274);

var part1275 = match("MESSAGE#766:00259:05/0", "nwparser.payload", "Admin user %{administrator->} has been forced to log out of the %{p0}");

var part1276 = match("MESSAGE#766:00259:05/1_2", "nwparser.p0", "Web %{p0}");

var select293 = linear_select([
	dup241,
	dup289,
	part1276,
]);

var part1277 = match("MESSAGE#766:00259:05/2", "nwparser.p0", "session on host %{daddr}:%{dport}");

var all265 = all_match({
	processors: [
		part1275,
		select293,
		part1277,
	],
	on_success: processor_chain([
		dup290,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg776 = msg("00259:05", all265);

var part1278 = match("MESSAGE#767:00259:06", "nwparser.payload", "Admin user %{administrator->} has been forced to log out of the serial console session.", processor_chain([
	dup290,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg777 = msg("00259:06", part1278);

var select294 = linear_select([
	msg770,
	msg771,
	msg772,
	msg773,
	msg774,
	msg775,
	msg776,
	msg777,
]);

var part1279 = match("MESSAGE#768:00262", "nwparser.payload", "Admin user %{administrator->} has been rejected via the %{logon_type->} server at %{hostip}", processor_chain([
	dup290,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg778 = msg("00262", part1279);

var part1280 = match("MESSAGE#769:00263", "nwparser.payload", "Admin user %{administrator->} has been accepted via the %{logon_type->} server at %{hostip}", processor_chain([
	setc("eventcategory","1401050100"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg779 = msg("00263", part1280);

var part1281 = match("MESSAGE#770:00400/0_0", "nwparser.payload", "ActiveX control %{p0}");

var part1282 = match("MESSAGE#770:00400/0_1", "nwparser.payload", "JAVA applet %{p0}");

var part1283 = match("MESSAGE#770:00400/0_2", "nwparser.payload", "EXE file %{p0}");

var part1284 = match("MESSAGE#770:00400/0_3", "nwparser.payload", "ZIP file %{p0}");

var select295 = linear_select([
	part1281,
	part1282,
	part1283,
	part1284,
]);

var part1285 = match("MESSAGE#770:00400/1", "nwparser.p0", "has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at interface %{dinterface->} in zone %{dst_zone}. %{info}");

var all266 = all_match({
	processors: [
		select295,
		part1285,
	],
	on_success: processor_chain([
		setc("eventcategory","1003000000"),
		dup2,
		dup4,
		dup5,
		dup3,
		dup61,
	]),
});

var msg780 = msg("00400", all266);

var part1286 = match("MESSAGE#771:00401", "nwparser.payload", "%{signame}! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone}, int %{interface}). %{info}", processor_chain([
	dup85,
	dup2,
	dup4,
	dup5,
	dup3,
	dup291,
]));

var msg781 = msg("00401", part1286);

var part1287 = match("MESSAGE#772:00402", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone}, int %{interface}). %{info}", processor_chain([
	dup85,
	dup2,
	dup4,
	dup5,
	dup3,
	dup292,
]));

var msg782 = msg("00402", part1287);

var part1288 = match("MESSAGE#773:00402:01/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at %{p0}");

var part1289 = match("MESSAGE#773:00402:01/2", "nwparser.p0", "%{} %{interface->} in zone %{zone}. %{info}");

var all267 = all_match({
	processors: [
		part1288,
		dup337,
		part1289,
	],
	on_success: processor_chain([
		dup85,
		dup2,
		dup4,
		dup5,
		dup3,
		dup292,
	]),
});

var msg783 = msg("00402:01", all267);

var select296 = linear_select([
	msg782,
	msg783,
]);

var part1290 = match("MESSAGE#774:00403", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone}, int %{interface}). %{info}", processor_chain([
	dup85,
	dup2,
	dup4,
	dup5,
	dup3,
	dup291,
]));

var msg784 = msg("00403", part1290);

var part1291 = match("MESSAGE#775:00404", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone}, int %{interface}). %{info}", processor_chain([
	dup147,
	dup148,
	dup149,
	dup150,
	dup2,
	dup4,
	dup5,
	dup3,
	dup292,
]));

var msg785 = msg("00404", part1291);

var part1292 = match("MESSAGE#776:00405", "nwparser.payload", "%{signame}! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone}, int %{interface}). %{info}", processor_chain([
	dup147,
	dup2,
	dup4,
	dup5,
	dup3,
	dup291,
]));

var msg786 = msg("00405", part1292);

var msg787 = msg("00406", dup413);

var msg788 = msg("00407", dup413);

var msg789 = msg("00408", dup413);

var all268 = all_match({
	processors: [
		dup132,
		dup343,
		dup293,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup60,
	]),
});

var msg790 = msg("00409", all268);

var msg791 = msg("00410", dup413);

var part1293 = match("MESSAGE#782:00410:01", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup60,
]));

var msg792 = msg("00410:01", part1293);

var select297 = linear_select([
	msg791,
	msg792,
]);

var part1294 = match("MESSAGE#783:00411/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto TCP (zone %{zone->} %{p0}");

var all269 = all_match({
	processors: [
		part1294,
		dup343,
		dup293,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg793 = msg("00411", all269);

var part1295 = match("MESSAGE#784:00413/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at %{p0}");

var part1296 = match("MESSAGE#784:00413/2", "nwparser.p0", "%{} %{interface}.%{space}The attack occurred %{dclass_counter1->} times");

var all270 = all_match({
	processors: [
		part1295,
		dup337,
		part1296,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var msg794 = msg("00413", all270);

var part1297 = match("MESSAGE#785:00413:01/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol}(zone %{group->} %{p0}");

var all271 = all_match({
	processors: [
		part1297,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup9,
		dup61,
	]),
});

var msg795 = msg("00413:01", all271);

var part1298 = match("MESSAGE#786:00413:02", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup59,
	dup5,
	dup9,
]));

var msg796 = msg("00413:02", part1298);

var select298 = linear_select([
	msg794,
	msg795,
	msg796,
]);

var part1299 = match("MESSAGE#787:00414", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone}, int %{interface}). Occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup9,
]));

var msg797 = msg("00414", part1299);

var part1300 = match("MESSAGE#788:00414:01", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup59,
	dup4,
	dup5,
	dup9,
]));

var msg798 = msg("00414:01", part1300);

var select299 = linear_select([
	msg797,
	msg798,
]);

var part1301 = match("MESSAGE#789:00415", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg799 = msg("00415", part1301);

var all272 = all_match({
	processors: [
		dup132,
		dup343,
		dup294,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup60,
	]),
});

var msg800 = msg("00423", all272);

var all273 = all_match({
	processors: [
		dup80,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup60,
	]),
});

var msg801 = msg("00429", all273);

var all274 = all_match({
	processors: [
		dup132,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup60,
	]),
});

var msg802 = msg("00429:01", all274);

var select300 = linear_select([
	msg801,
	msg802,
]);

var all275 = all_match({
	processors: [
		dup80,
		dup343,
		dup295,
		dup351,
	],
	on_success: processor_chain([
		dup85,
		dup2,
		dup59,
		dup3,
		dup9,
		dup4,
		dup5,
		dup61,
	]),
});

var msg803 = msg("00430", all275);

var all276 = all_match({
	processors: [
		dup132,
		dup343,
		dup295,
		dup351,
	],
	on_success: processor_chain([
		dup85,
		dup2,
		dup59,
		dup3,
		dup9,
		dup4,
		dup5,
		dup60,
	]),
});

var msg804 = msg("00430:01", all276);

var select301 = linear_select([
	msg803,
	msg804,
]);

var msg805 = msg("00431", dup414);

var msg806 = msg("00432", dup414);

var msg807 = msg("00433", dup415);

var msg808 = msg("00434", dup415);

var msg809 = msg("00435", dup395);

var all277 = all_match({
	processors: [
		dup132,
		dup343,
		dup294,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup4,
		dup59,
		dup5,
		dup3,
		dup60,
	]),
});

var msg810 = msg("00435:01", all277);

var select302 = linear_select([
	msg809,
	msg810,
]);

var msg811 = msg("00436", dup395);

var all278 = all_match({
	processors: [
		dup64,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup9,
		dup4,
		dup5,
		dup3,
		dup60,
	]),
});

var msg812 = msg("00436:01", all278);

var select303 = linear_select([
	msg811,
	msg812,
]);

var part1302 = match("MESSAGE#803:00437", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg813 = msg("00437", part1302);

var all279 = all_match({
	processors: [
		dup299,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
		dup9,
	]),
});

var msg814 = msg("00437:01", all279);

var part1303 = match("MESSAGE#805:00437:02", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup61,
	dup9,
]));

var msg815 = msg("00437:02", part1303);

var select304 = linear_select([
	msg813,
	msg814,
	msg815,
]);

var part1304 = match("MESSAGE#806:00438", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport->} using protocol %{protocol->} and arriving at interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg816 = msg("00438", part1304);

var part1305 = match("MESSAGE#807:00438:01", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, on zone %{zone->} interface %{interface}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var msg817 = msg("00438:01", part1305);

var all280 = all_match({
	processors: [
		dup299,
		dup338,
		dup67,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup9,
		dup61,
	]),
});

var msg818 = msg("00438:02", all280);

var select305 = linear_select([
	msg816,
	msg817,
	msg818,
]);

var part1306 = match("MESSAGE#809:00440", "nwparser.payload", "%{signame->} has been detected! From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup4,
	dup5,
	dup9,
	dup60,
]));

var msg819 = msg("00440", part1306);

var part1307 = match("MESSAGE#810:00440:02", "nwparser.payload", "%{signame->} has been detected! From %{saddr}:%{sport->} to %{daddr}:%{dport}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup59,
	dup4,
	dup5,
	dup3,
	dup61,
]));

var msg820 = msg("00440:02", part1307);

var all281 = all_match({
	processors: [
		dup239,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup4,
		dup5,
		dup3,
		dup9,
		dup61,
	]),
});

var msg821 = msg("00440:01", all281);

var part1308 = match("MESSAGE#812:00440:03/0", "nwparser.payload", "Fragmented traffic! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{group->} %{p0}");

var all282 = all_match({
	processors: [
		part1308,
		dup343,
		dup83,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup4,
		dup5,
		dup3,
		dup9,
		dup60,
	]),
});

var msg822 = msg("00440:03", all282);

var select306 = linear_select([
	msg819,
	msg820,
	msg821,
	msg822,
]);

var part1309 = match("MESSAGE#813:00441", "nwparser.payload", "%{signame->} id=%{fld2}! From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone}). Occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup4,
	dup59,
	dup5,
	dup9,
	dup2,
	dup3,
	dup60,
]));

var msg823 = msg("00441", part1309);

var msg824 = msg("00442", dup396);

var msg825 = msg("00443", dup396);

var part1310 = match("MESSAGE#816:00511", "nwparser.payload", "admin %{administrator->} issued command %{fld2->} to redirect output.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg826 = msg("00511", part1310);

var part1311 = match("MESSAGE#817:00511:01/0", "nwparser.payload", "All System Config saved by admin %{p0}");

var all283 = all_match({
	processors: [
		part1311,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg827 = msg("00511:01", all283);

var part1312 = match("MESSAGE#818:00511:02", "nwparser.payload", "All logged events or alarms are cleared by admin %{administrator}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg828 = msg("00511:02", part1312);

var part1313 = match("MESSAGE#819:00511:03/0", "nwparser.payload", "Get new software from flash to slot (file: %{fld2}) by admin %{p0}");

var all284 = all_match({
	processors: [
		part1313,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg829 = msg("00511:03", all284);

var part1314 = match("MESSAGE#820:00511:04/0", "nwparser.payload", "Get new software from %{hostip->} (file: %{fld2}) to slot (file: %{fld3}) by admin %{p0}");

var all285 = all_match({
	processors: [
		part1314,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg830 = msg("00511:04", all285);

var part1315 = match("MESSAGE#821:00511:05/0", "nwparser.payload", "Get new software to %{hostip->} (file: %{fld2}) by admin %{p0}");

var all286 = all_match({
	processors: [
		part1315,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg831 = msg("00511:05", all286);

var part1316 = match("MESSAGE#822:00511:06/0", "nwparser.payload", "Log setting is modified by admin %{p0}");

var all287 = all_match({
	processors: [
		part1316,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg832 = msg("00511:06", all287);

var part1317 = match("MESSAGE#823:00511:07/0", "nwparser.payload", "Save configuration to %{hostip->} (file: %{fld2}) by admin %{p0}");

var all288 = all_match({
	processors: [
		part1317,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg833 = msg("00511:07", all288);

var part1318 = match("MESSAGE#824:00511:08/0", "nwparser.payload", "Save new software from slot (file: %{fld2}) to flash by admin %{p0}");

var all289 = all_match({
	processors: [
		part1318,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg834 = msg("00511:08", all289);

var part1319 = match("MESSAGE#825:00511:09/0", "nwparser.payload", "Save new software from %{hostip->} (file: %{result}) to flash by admin %{p0}");

var all290 = all_match({
	processors: [
		part1319,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg835 = msg("00511:09", all290);

var part1320 = match("MESSAGE#826:00511:10/0", "nwparser.payload", "System Config from flash to slot - %{fld2->} by admin %{p0}");

var all291 = all_match({
	processors: [
		part1320,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg836 = msg("00511:10", all291);

var part1321 = match("MESSAGE#827:00511:11/0", "nwparser.payload", "System Config load from %{hostip->} (file %{fld2}) to slot - %{fld3->} by admin %{p0}");

var all292 = all_match({
	processors: [
		part1321,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg837 = msg("00511:11", all292);

var part1322 = match("MESSAGE#828:00511:12/0", "nwparser.payload", "System Config load from %{hostip->} (file %{fld2}) by admin %{p0}");

var all293 = all_match({
	processors: [
		part1322,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg838 = msg("00511:12", all293);

var part1323 = match("MESSAGE#829:00511:13/0", "nwparser.payload", "The system configuration was loaded from the slot by admin %{p0}");

var all294 = all_match({
	processors: [
		part1323,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg839 = msg("00511:13", all294);

var part1324 = match("MESSAGE#830:00511:14", "nwparser.payload", "FIPS: Attempt to set RADIUS shared secret with invalid length %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg840 = msg("00511:14", part1324);

var select307 = linear_select([
	msg826,
	msg827,
	msg828,
	msg829,
	msg830,
	msg831,
	msg832,
	msg833,
	msg834,
	msg835,
	msg836,
	msg837,
	msg838,
	msg839,
	msg840,
]);

var part1325 = match("MESSAGE#831:00513/0", "nwparser.payload", "The physical state of %{p0}");

var part1326 = match("MESSAGE#831:00513/1_1", "nwparser.p0", "the Interface %{p0}");

var select308 = linear_select([
	dup123,
	part1326,
	dup122,
]);

var part1327 = match("MESSAGE#831:00513/2", "nwparser.p0", "%{interface->} has changed to %{p0}");

var part1328 = match("MESSAGE#831:00513/3_0", "nwparser.p0", "%{result}. (%{fld1})");

var part1329 = match_copy("MESSAGE#831:00513/3_1", "nwparser.p0", "result");

var select309 = linear_select([
	part1328,
	part1329,
]);

var all295 = all_match({
	processors: [
		part1325,
		select308,
		part1327,
		select309,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
		dup9,
	]),
});

var msg841 = msg("00513", all295);

var part1330 = match("MESSAGE#832:00515/0_0", "nwparser.payload", "Vsys Admin %{p0}");

var select310 = linear_select([
	part1330,
	dup287,
]);

var part1331 = match("MESSAGE#832:00515/1", "nwparser.p0", "%{administrator->} has logged on via the %{logon_type->} ( HTTP%{p0}");

var part1332 = match("MESSAGE#832:00515/2_1", "nwparser.p0", "S%{p0}");

var select311 = linear_select([
	dup96,
	part1332,
]);

var part1333 = match("MESSAGE#832:00515/3", "nwparser.p0", "%{}) to port %{interface->} from %{saddr}:%{sport}");

var all296 = all_match({
	processors: [
		select310,
		part1331,
		select311,
		part1333,
	],
	on_success: processor_chain([
		dup301,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg842 = msg("00515", all296);

var part1334 = match("MESSAGE#833:00515:01/0", "nwparser.payload", "Login attempt to system by admin %{administrator->} via %{p0}");

var part1335 = match("MESSAGE#833:00515:01/1_0", "nwparser.p0", "the %{logon_type->} has failed %{p0}");

var part1336 = match("MESSAGE#833:00515:01/1_1", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport->} has failed %{p0}");

var select312 = linear_select([
	part1335,
	part1336,
]);

var part1337 = match_copy("MESSAGE#833:00515:01/2", "nwparser.p0", "fld2");

var all297 = all_match({
	processors: [
		part1334,
		select312,
		part1337,
	],
	on_success: processor_chain([
		dup206,
		dup29,
		dup30,
		dup31,
		dup54,
		dup2,
		dup4,
		dup5,
		dup302,
		dup3,
	]),
});

var msg843 = msg("00515:01", all297);

var part1338 = match("MESSAGE#834:00515:02/0", "nwparser.payload", "Management session via %{p0}");

var part1339 = match("MESSAGE#834:00515:02/1_0", "nwparser.p0", "the %{logon_type->} for %{p0}");

var part1340 = match("MESSAGE#834:00515:02/1_1", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport->} for %{p0}");

var select313 = linear_select([
	part1339,
	part1340,
]);

var part1341 = match("MESSAGE#834:00515:02/2_0", "nwparser.p0", "[vsys] admin %{p0}");

var part1342 = match("MESSAGE#834:00515:02/2_1", "nwparser.p0", "vsys admin %{p0}");

var select314 = linear_select([
	part1341,
	part1342,
	dup15,
]);

var part1343 = match("MESSAGE#834:00515:02/3", "nwparser.p0", "%{administrator->} has timed out");

var all298 = all_match({
	processors: [
		part1338,
		select313,
		select314,
		part1343,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg844 = msg("00515:02", all298);

var part1344 = match("MESSAGE#835:00515:04/0_0", "nwparser.payload", "[Vsys] %{p0}");

var part1345 = match("MESSAGE#835:00515:04/0_1", "nwparser.payload", "Vsys %{p0}");

var select315 = linear_select([
	part1344,
	part1345,
]);

var part1346 = match("MESSAGE#835:00515:04/1", "nwparser.p0", "Admin %{administrator->} has logged o%{p0}");

var part1347 = match_copy("MESSAGE#835:00515:04/4_1", "nwparser.p0", "logon_type");

var select316 = linear_select([
	dup304,
	part1347,
]);

var all299 = all_match({
	processors: [
		select315,
		part1346,
		dup398,
		dup40,
		select316,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg845 = msg("00515:04", all299);

var part1348 = match("MESSAGE#836:00515:06", "nwparser.payload", "Admin User %{administrator->} has logged on via %{logon_type->} from %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg846 = msg("00515:06", part1348);

var part1349 = match("MESSAGE#837:00515:05/0", "nwparser.payload", "%{}Admin %{p0}");

var select317 = linear_select([
	dup305,
	dup16,
]);

var part1350 = match("MESSAGE#837:00515:05/2", "nwparser.p0", "%{administrator->} has logged o%{p0}");

var part1351 = match("MESSAGE#837:00515:05/5_1", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport->} (%{fld2})");

var select318 = linear_select([
	dup306,
	part1351,
	dup304,
]);

var all300 = all_match({
	processors: [
		part1349,
		select317,
		part1350,
		dup398,
		dup40,
		select318,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg847 = msg("00515:05", all300);

var part1352 = match("MESSAGE#838:00515:07", "nwparser.payload", "Admin user %{administrator->} login attempt for %{logon_type}(http) management (port %{network_port}) from %{saddr}:%{sport->} %{disposition}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg848 = msg("00515:07", part1352);

var part1353 = match("MESSAGE#839:00515:08/0", "nwparser.payload", "%{fld2->} Admin User \"%{administrator}\" logged in for %{logon_type}(http%{p0}");

var part1354 = match("MESSAGE#839:00515:08/1_0", "nwparser.p0", ") %{p0}");

var part1355 = match("MESSAGE#839:00515:08/1_1", "nwparser.p0", "s) %{p0}");

var select319 = linear_select([
	part1354,
	part1355,
]);

var part1356 = match("MESSAGE#839:00515:08/2", "nwparser.p0", "management (port %{network_port}) from %{saddr}:%{sport}");

var all301 = all_match({
	processors: [
		part1353,
		select319,
		part1356,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg849 = msg("00515:08", all301);

var part1357 = match("MESSAGE#840:00515:09", "nwparser.payload", "User %{username->} telnet management session from (%{saddr}:%{sport}) timed out", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg850 = msg("00515:09", part1357);

var part1358 = match("MESSAGE#841:00515:10", "nwparser.payload", "User %{username->} logged out of telnet session from %{saddr}:%{sport}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg851 = msg("00515:10", part1358);

var part1359 = match("MESSAGE#842:00515:11", "nwparser.payload", "The session limit threshold has been set to %{trigger_val->} on zone %{zone}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg852 = msg("00515:11", part1359);

var part1360 = match("MESSAGE#843:00515:12/0", "nwparser.payload", "[ Vsys ] Admin User \"%{administrator}\" logged in for Web( http%{p0}");

var part1361 = match("MESSAGE#843:00515:12/2", "nwparser.p0", ") management (port %{network_port})");

var all302 = all_match({
	processors: [
		part1360,
		dup399,
		part1361,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg853 = msg("00515:12", all302);

var select320 = linear_select([
	dup288,
	dup287,
]);

var part1362 = match("MESSAGE#844:00515:13/1", "nwparser.p0", "user %{administrator->} has logged o%{p0}");

var select321 = linear_select([
	dup306,
	dup304,
]);

var all303 = all_match({
	processors: [
		select320,
		part1362,
		dup398,
		dup40,
		select321,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg854 = msg("00515:13", all303);

var part1363 = match("MESSAGE#845:00515:14/0_0", "nwparser.payload", "Admin user %{administrator->} has been forced to log o%{p0}");

var part1364 = match("MESSAGE#845:00515:14/0_1", "nwparser.payload", "%{username->} %{fld1->} has been forced to log o%{p0}");

var select322 = linear_select([
	part1363,
	part1364,
]);

var part1365 = match("MESSAGE#845:00515:14/2", "nwparser.p0", "of the %{p0}");

var part1366 = match("MESSAGE#845:00515:14/3_0", "nwparser.p0", "serial %{logon_type->} session.");

var part1367 = match("MESSAGE#845:00515:14/3_1", "nwparser.p0", "%{logon_type->} session on host %{hostip}:%{network_port->} (%{event_time})");

var part1368 = match("MESSAGE#845:00515:14/3_2", "nwparser.p0", "%{logon_type->} session on host %{hostip}:%{network_port}");

var select323 = linear_select([
	part1366,
	part1367,
	part1368,
]);

var all304 = all_match({
	processors: [
		select322,
		dup398,
		part1365,
		select323,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg855 = msg("00515:14", all304);

var part1369 = match("MESSAGE#846:00515:15/0", "nwparser.payload", "%{fld2}: Admin User %{administrator->} has logged o%{p0}");

var part1370 = match("MESSAGE#846:00515:15/3_0", "nwparser.p0", "the %{logon_type->} (%{p0}");

var part1371 = match("MESSAGE#846:00515:15/3_1", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport->} (%{p0}");

var select324 = linear_select([
	part1370,
	part1371,
]);

var all305 = all_match({
	processors: [
		part1369,
		dup398,
		dup40,
		select324,
		dup41,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg856 = msg("00515:15", all305);

var part1372 = match("MESSAGE#847:00515:16/0_0", "nwparser.payload", "%{fld2}: Admin %{p0}");

var select325 = linear_select([
	part1372,
	dup287,
]);

var part1373 = match("MESSAGE#847:00515:16/1", "nwparser.p0", "user %{administrator->} attempt access to %{url->} illegal from %{logon_type}( http%{p0}");

var part1374 = match("MESSAGE#847:00515:16/3", "nwparser.p0", ") management (port %{network_port}) from %{saddr}:%{sport}. (%{fld1})");

var all306 = all_match({
	processors: [
		select325,
		part1373,
		dup399,
		part1374,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg857 = msg("00515:16", all306);

var part1375 = match("MESSAGE#848:00515:17/0", "nwparser.payload", "Admin user \"%{administrator}\" logged out for %{logon_type}(%{p0}");

var part1376 = match("MESSAGE#848:00515:17/1_0", "nwparser.p0", "https %{p0}");

var part1377 = match("MESSAGE#848:00515:17/1_1", "nwparser.p0", " http %{p0}");

var select326 = linear_select([
	part1376,
	part1377,
]);

var part1378 = match("MESSAGE#848:00515:17/2", "nwparser.p0", ") management (port %{network_port}) from %{saddr}:%{sport}");

var all307 = all_match({
	processors: [
		part1375,
		select326,
		part1378,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg858 = msg("00515:17", all307);

var part1379 = match("MESSAGE#849:00515:18", "nwparser.payload", "Admin user %{administrator->} login attempt for %{logon_type}(https) management (port %{network_port}) from %{saddr}:%{sport->} %{disposition}. (%{fld1})", processor_chain([
	dup240,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg859 = msg("00515:18", part1379);

var part1380 = match("MESSAGE#850:00515:19/0", "nwparser.payload", "Vsys admin user %{administrator->} logged on via %{p0}");

var part1381 = match("MESSAGE#850:00515:19/1_0", "nwparser.p0", "%{logon_type->} from remote IP address %{saddr->} using port %{sport}. (%{p0}");

var part1382 = match("MESSAGE#850:00515:19/1_1", "nwparser.p0", "the console. (%{p0}");

var select327 = linear_select([
	part1381,
	part1382,
]);

var all308 = all_match({
	processors: [
		part1380,
		select327,
		dup41,
	],
	on_success: processor_chain([
		dup240,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg860 = msg("00515:19", all308);

var part1383 = match("MESSAGE#851:00515:20", "nwparser.payload", "netscreen: Management session via SCS from %{saddr}:%{sport->} for admin netscreen has timed out (%{fld1})", processor_chain([
	dup240,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg861 = msg("00515:20", part1383);

var select328 = linear_select([
	msg842,
	msg843,
	msg844,
	msg845,
	msg846,
	msg847,
	msg848,
	msg849,
	msg850,
	msg851,
	msg852,
	msg853,
	msg854,
	msg855,
	msg856,
	msg857,
	msg858,
	msg859,
	msg860,
	msg861,
]);

var part1384 = match("MESSAGE#852:00518", "nwparser.payload", "Admin user %{administrator->} %{fld1}at %{saddr->} has been %{disposition->} via the %{logon_type->} server at %{hostip}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg862 = msg("00518", part1384);

var part1385 = match("MESSAGE#853:00518:17", "nwparser.payload", "Admin user %{administrator->} has been %{disposition->} via the %{logon_type->} server at %{hostip}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg863 = msg("00518:17", part1385);

var part1386 = match("MESSAGE#854:00518:01", "nwparser.payload", "Local authentication for WebAuth user %{username->} was %{disposition}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg864 = msg("00518:01", part1386);

var part1387 = match("MESSAGE#855:00518:02", "nwparser.payload", "Local authentication for user %{username->} was %{disposition}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg865 = msg("00518:02", part1387);

var part1388 = match("MESSAGE#856:00518:03", "nwparser.payload", "User %{username->} at %{saddr->} must enter \"Next Code\" for SecurID %{hostip}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg866 = msg("00518:03", part1388);

var part1389 = match("MESSAGE#857:00518:04", "nwparser.payload", "WebAuth user %{username->} at %{saddr->} has been %{disposition->} via the %{logon_type->} server at %{hostip}", processor_chain([
	dup203,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg867 = msg("00518:04", part1389);

var part1390 = match("MESSAGE#858:00518:05", "nwparser.payload", "User %{username->} at %{saddr->} has been challenged via the %{authmethod->} server at %{hostip->} (Rejected since challenge is not supported for %{logon_type})", processor_chain([
	dup203,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg868 = msg("00518:05", part1390);

var part1391 = match("MESSAGE#859:00518:06", "nwparser.payload", "Error in authentication for WebAuth user %{username}", processor_chain([
	dup35,
	dup29,
	dup31,
	dup54,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg869 = msg("00518:06", part1391);

var part1392 = match("MESSAGE#860:00518:07/0", "nwparser.payload", "Authentication for user %{username->} was denied (long %{p0}");

var part1393 = match("MESSAGE#860:00518:07/1_1", "nwparser.p0", "username %{p0}");

var select329 = linear_select([
	dup24,
	part1393,
]);

var part1394 = match("MESSAGE#860:00518:07/2", "nwparser.p0", ")%{}");

var all309 = all_match({
	processors: [
		part1392,
		select329,
		part1394,
	],
	on_success: processor_chain([
		dup53,
		dup29,
		dup31,
		dup54,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg870 = msg("00518:07", all309);

var part1395 = match("MESSAGE#861:00518:08", "nwparser.payload", "User %{username->} at %{saddr->} %{authmethod->} authentication attempt has timed out", processor_chain([
	dup35,
	dup29,
	dup31,
	dup39,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg871 = msg("00518:08", part1395);

var part1396 = match("MESSAGE#862:00518:09", "nwparser.payload", "User %{username->} at %{saddr->} has been %{disposition->} via the %{logon_type->} server at %{hostip}", processor_chain([
	dup203,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg872 = msg("00518:09", part1396);

var part1397 = match("MESSAGE#863:00518:10", "nwparser.payload", "Admin user \"%{administrator}\" login attempt for %{logon_type->} (%{network_service}) management (port %{network_port}) from %{saddr}:%{sport->} failed due to %{result}. (%{fld1})", processor_chain([
	dup206,
	dup29,
	dup30,
	dup31,
	dup54,
	dup2,
	dup4,
	dup9,
	dup5,
	dup3,
	dup302,
]));

var msg873 = msg("00518:10", part1397);

var part1398 = match("MESSAGE#864:00518:11/0", "nwparser.payload", "ADM: Local admin authentication failed for login name %{p0}");

var part1399 = match("MESSAGE#864:00518:11/1_0", "nwparser.p0", "'%{username}': %{p0}");

var part1400 = match("MESSAGE#864:00518:11/1_1", "nwparser.p0", "%{username}: %{p0}");

var select330 = linear_select([
	part1399,
	part1400,
]);

var part1401 = match("MESSAGE#864:00518:11/2", "nwparser.p0", "%{result->} (%{fld1})");

var all310 = all_match({
	processors: [
		part1398,
		select330,
		part1401,
	],
	on_success: processor_chain([
		dup206,
		dup29,
		dup30,
		dup31,
		dup54,
		dup2,
		dup9,
		dup4,
		dup5,
		dup3,
	]),
});

var msg874 = msg("00518:11", all310);

var part1402 = match("MESSAGE#865:00518:12", "nwparser.payload", "Admin user \"%{administrator}\" login attempt for %{logon_type}(%{network_service}) management (port %{network_port}) from %{saddr}:%{sport->} %{disposition}. (%{fld1})", processor_chain([
	dup240,
	dup2,
	dup4,
	dup9,
	dup5,
	dup3,
]));

var msg875 = msg("00518:12", part1402);

var part1403 = match("MESSAGE#866:00518:13", "nwparser.payload", "User %{username->} at %{saddr->} is rejected by the Radius server at %{hostip}. (%{fld1})", processor_chain([
	dup290,
	dup2,
	dup3,
	dup4,
	dup9,
	dup5,
]));

var msg876 = msg("00518:13", part1403);

var part1404 = match("MESSAGE#867:00518:14", "nwparser.payload", "%{fld2}: Admin user has been rejected via the Radius server at %{hostip->} (%{fld1})", processor_chain([
	dup290,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg877 = msg("00518:14", part1404);

var select331 = linear_select([
	msg862,
	msg863,
	msg864,
	msg865,
	msg866,
	msg867,
	msg868,
	msg869,
	msg870,
	msg871,
	msg872,
	msg873,
	msg874,
	msg875,
	msg876,
	msg877,
]);

var part1405 = match("MESSAGE#868:00519/0", "nwparser.payload", "Admin user %{administrator->} %{p0}");

var part1406 = match("MESSAGE#868:00519/1_1", "nwparser.p0", "of group %{group->} at %{saddr->} has %{p0}");

var part1407 = match("MESSAGE#868:00519/1_2", "nwparser.p0", "%{group->} at %{saddr->} has %{p0}");

var select332 = linear_select([
	dup194,
	part1406,
	part1407,
]);

var part1408 = match("MESSAGE#868:00519/2", "nwparser.p0", "been %{disposition->} via the %{logon_type->} server %{p0}");

var part1409 = match("MESSAGE#868:00519/3_0", "nwparser.p0", "at %{p0}");

var select333 = linear_select([
	part1409,
	dup16,
]);

var part1410 = match("MESSAGE#868:00519/4", "nwparser.p0", "%{hostip}");

var all311 = all_match({
	processors: [
		part1405,
		select332,
		part1408,
		select333,
		part1410,
	],
	on_success: processor_chain([
		dup203,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg878 = msg("00519", all311);

var part1411 = match("MESSAGE#869:00519:01/0", "nwparser.payload", "Local authentication for %{p0}");

var select334 = linear_select([
	dup307,
	dup305,
]);

var part1412 = match("MESSAGE#869:00519:01/2", "nwparser.p0", "%{username->} was %{disposition}");

var all312 = all_match({
	processors: [
		part1411,
		select334,
		part1412,
	],
	on_success: processor_chain([
		dup203,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg879 = msg("00519:01", all312);

var part1413 = match("MESSAGE#870:00519:02/1_1", "nwparser.p0", "User %{p0}");

var select335 = linear_select([
	dup307,
	part1413,
]);

var part1414 = match("MESSAGE#870:00519:02/2", "nwparser.p0", "%{username->} at %{saddr->} has been %{disposition->} via the %{logon_type->} server at %{hostip}");

var all313 = all_match({
	processors: [
		dup160,
		select335,
		part1414,
	],
	on_success: processor_chain([
		dup203,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg880 = msg("00519:02", all313);

var part1415 = match("MESSAGE#871:00519:03", "nwparser.payload", "Admin user \"%{administrator}\" logged in for %{logon_type}(%{network_service}) management (port %{network_port}) from %{saddr}:%{sport->} %{fld4}", processor_chain([
	dup240,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg881 = msg("00519:03", part1415);

var part1416 = match("MESSAGE#872:00519:04", "nwparser.payload", "ADM: Local admin authentication successful for login name %{username->} (%{fld1})", processor_chain([
	dup240,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg882 = msg("00519:04", part1416);

var part1417 = match("MESSAGE#873:00519:05", "nwparser.payload", "%{fld2}Admin user %{administrator->} has been accepted via the Radius server at %{hostip}(%{fld1})", processor_chain([
	dup240,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg883 = msg("00519:05", part1417);

var select336 = linear_select([
	msg878,
	msg879,
	msg880,
	msg881,
	msg882,
	msg883,
]);

var part1418 = match("MESSAGE#874:00520", "nwparser.payload", "%{hostname->} user authentication attempt has timed out", processor_chain([
	dup35,
	dup31,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg884 = msg("00520", part1418);

var part1419 = match("MESSAGE#875:00520:01/0", "nwparser.payload", "User %{username->} at %{hostip->} %{p0}");

var part1420 = match("MESSAGE#875:00520:01/1_0", "nwparser.p0", "RADIUS %{p0}");

var part1421 = match("MESSAGE#875:00520:01/1_1", "nwparser.p0", "SecurID %{p0}");

var part1422 = match("MESSAGE#875:00520:01/1_2", "nwparser.p0", "LDAP %{p0}");

var part1423 = match("MESSAGE#875:00520:01/1_3", "nwparser.p0", "Local %{p0}");

var select337 = linear_select([
	part1420,
	part1421,
	part1422,
	part1423,
]);

var part1424 = match("MESSAGE#875:00520:01/2", "nwparser.p0", "authentication attempt has timed out%{}");

var all314 = all_match({
	processors: [
		part1419,
		select337,
		part1424,
	],
	on_success: processor_chain([
		dup35,
		dup31,
		dup39,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg885 = msg("00520:01", all314);

var part1425 = match("MESSAGE#876:00520:02/0", "nwparser.payload", "Trying %{p0}");

var part1426 = match("MESSAGE#876:00520:02/2", "nwparser.p0", "server %{fld2}");

var all315 = all_match({
	processors: [
		part1425,
		dup400,
		part1426,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg886 = msg("00520:02", all315);

var part1427 = match("MESSAGE#877:00520:03/1_0", "nwparser.p0", "Primary %{p0}");

var part1428 = match("MESSAGE#877:00520:03/1_1", "nwparser.p0", "Backup1 %{p0}");

var part1429 = match("MESSAGE#877:00520:03/1_2", "nwparser.p0", "Backup2 %{p0}");

var select338 = linear_select([
	part1427,
	part1428,
	part1429,
]);

var part1430 = match("MESSAGE#877:00520:03/2", "nwparser.p0", "%{fld2}, %{p0}");

var part1431 = match("MESSAGE#877:00520:03/4", "nwparser.p0", "%{fld3}, and %{p0}");

var part1432 = match("MESSAGE#877:00520:03/6", "nwparser.p0", "%{fld4->} servers failed");

var all316 = all_match({
	processors: [
		dup160,
		select338,
		part1430,
		dup400,
		part1431,
		dup400,
		part1432,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg887 = msg("00520:03", all316);

var part1433 = match("MESSAGE#878:00520:04", "nwparser.payload", "Trying %{fld2->} Server %{hostip->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg888 = msg("00520:04", part1433);

var part1434 = match("MESSAGE#1221:00520:05", "nwparser.payload", "Active Server Switchover: New requests for %{fld31->} server will try %{fld32->} from now on. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg889 = msg("00520:05", part1434);

var select339 = linear_select([
	msg884,
	msg885,
	msg886,
	msg887,
	msg888,
	msg889,
]);

var part1435 = match("MESSAGE#879:00521", "nwparser.payload", "Can't connect to E-mail server %{hostip}", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg890 = msg("00521", part1435);

var part1436 = match("MESSAGE#880:00522", "nwparser.payload", "HA link state has %{fld2}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg891 = msg("00522", part1436);

var part1437 = match("MESSAGE#881:00523", "nwparser.payload", "URL filtering received an error from %{fld2->} (error %{resultcode}).", processor_chain([
	dup232,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg892 = msg("00523", part1437);

var part1438 = match("MESSAGE#882:00524", "nwparser.payload", "NetScreen device at %{hostip}:%{network_port->} has responded successfully to SNMP request from %{saddr}:%{sport}", processor_chain([
	dup209,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg893 = msg("00524", part1438);

var part1439 = match("MESSAGE#883:00524:02", "nwparser.payload", "SNMP request from an unknown SNMP community public at %{hostip}:%{network_port->} has been received. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg894 = msg("00524:02", part1439);

var part1440 = match("MESSAGE#884:00524:03", "nwparser.payload", "SNMP: NetScreen device has responded successfully to the SNMP request from %{saddr}:%{sport}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg895 = msg("00524:03", part1440);

var part1441 = match("MESSAGE#885:00524:04", "nwparser.payload", "SNMP request from an unknown SNMP community admin at %{hostip}:%{network_port->} has been received. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg896 = msg("00524:04", part1441);

var part1442 = match("MESSAGE#886:00524:05", "nwparser.payload", "SNMP request from an unknown SNMP community %{fld2->} at %{hostip}:%{network_port->} has been received. (%{fld1})", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg897 = msg("00524:05", part1442);

var part1443 = match("MESSAGE#887:00524:06", "nwparser.payload", "SNMP request has been received from an unknown host in SNMP community %{fld2->} at %{hostip}:%{network_port}. (%{fld1})", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg898 = msg("00524:06", part1443);

var part1444 = match("MESSAGE#888:00524:12", "nwparser.payload", "SNMP request from an unknown SNMP community %{fld2->} at %{saddr}:%{sport->} to %{daddr}:%{dport->} has been received", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg899 = msg("00524:12", part1444);

var part1445 = match("MESSAGE#889:00524:14", "nwparser.payload", "SNMP request from %{saddr}:%{sport->} has been received, but the SNMP version type is incorrect. (%{fld1})", processor_chain([
	dup19,
	dup2,
	dup4,
	setc("result","the SNMP version type is incorrect"),
	dup5,
	dup9,
]));

var msg900 = msg("00524:14", part1445);

var part1446 = match("MESSAGE#890:00524:13/0", "nwparser.payload", "SNMP request has been received%{p0}");

var part1447 = match("MESSAGE#890:00524:13/2", "nwparser.p0", "%{}but %{result}");

var all317 = all_match({
	processors: [
		part1446,
		dup401,
		part1447,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup4,
		dup5,
	]),
});

var msg901 = msg("00524:13", all317);

var part1448 = match("MESSAGE#891:00524:07", "nwparser.payload", "Response to SNMP request from %{saddr}:%{sport->} to %{daddr}:%{dport->} has %{disposition->} due to %{result}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg902 = msg("00524:07", part1448);

var part1449 = match("MESSAGE#892:00524:08", "nwparser.payload", "SNMP community %{fld2->} cannot be added because %{result}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg903 = msg("00524:08", part1449);

var part1450 = match("MESSAGE#893:00524:09", "nwparser.payload", "SNMP host %{hostip->} cannot be added to community %{fld2->} because of %{result}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg904 = msg("00524:09", part1450);

var part1451 = match("MESSAGE#894:00524:10", "nwparser.payload", "SNMP host %{hostip->} cannot be added because %{result}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg905 = msg("00524:10", part1451);

var part1452 = match("MESSAGE#895:00524:11", "nwparser.payload", "SNMP host %{hostip->} cannot be removed from community %{fld2->} because %{result}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
]));

var msg906 = msg("00524:11", part1452);

var part1453 = match("MESSAGE#1222:00524:16", "nwparser.payload", "SNMP user/community %{fld34->} doesn't exist. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg907 = msg("00524:16", part1453);

var select340 = linear_select([
	msg893,
	msg894,
	msg895,
	msg896,
	msg897,
	msg898,
	msg899,
	msg900,
	msg901,
	msg902,
	msg903,
	msg904,
	msg905,
	msg906,
	msg907,
]);

var part1454 = match("MESSAGE#896:00525", "nwparser.payload", "The new PIN for user %{username->} at %{hostip->} has been %{disposition->} by SecurID %{fld2}", processor_chain([
	dup203,
	setc("ec_subject","Password"),
	dup38,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg908 = msg("00525", part1454);

var part1455 = match("MESSAGE#897:00525:01", "nwparser.payload", "User %{username->} at %{hostip->} has selected a system-generated PIN for authentication with SecurID %{fld2}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg909 = msg("00525:01", part1455);

var part1456 = match("MESSAGE#898:00525:02", "nwparser.payload", "User %{username->} at %{hostip->} must enter the \"new PIN\" for SecurID %{fld2}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg910 = msg("00525:02", part1456);

var part1457 = match("MESSAGE#899:00525:03", "nwparser.payload", "User %{username->} at %{hostip->} must make a \"New PIN\" choice for SecurID %{fld2}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg911 = msg("00525:03", part1457);

var select341 = linear_select([
	msg908,
	msg909,
	msg910,
	msg911,
]);

var part1458 = match("MESSAGE#900:00526", "nwparser.payload", "The user limit has been exceeded and %{hostip->} cannot be added", processor_chain([
	dup37,
	dup219,
	dup38,
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg912 = msg("00526", part1458);

var part1459 = match("MESSAGE#901:00527/0", "nwparser.payload", "A DHCP-%{p0}");

var part1460 = match("MESSAGE#901:00527/1_1", "nwparser.p0", " assigned %{p0}");

var select342 = linear_select([
	dup311,
	part1460,
]);

var part1461 = match("MESSAGE#901:00527/2", "nwparser.p0", "IP address %{hostip->} has been %{p0}");

var part1462 = match("MESSAGE#901:00527/3_1", "nwparser.p0", "freed from %{p0}");

var part1463 = match("MESSAGE#901:00527/3_2", "nwparser.p0", "freed %{p0}");

var select343 = linear_select([
	dup312,
	part1462,
	part1463,
]);

var all318 = all_match({
	processors: [
		part1459,
		select342,
		part1461,
		select343,
		dup108,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg913 = msg("00527", all318);

var part1464 = match("MESSAGE#902:00527:01", "nwparser.payload", "A DHCP-assigned IP address has been manually released%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg914 = msg("00527:01", part1464);

var part1465 = match("MESSAGE#903:00527:02/0", "nwparser.payload", "DHCP server has %{p0}");

var part1466 = match("MESSAGE#903:00527:02/1_1", "nwparser.p0", "released %{p0}");

var part1467 = match("MESSAGE#903:00527:02/1_2", "nwparser.p0", "assigned or released %{p0}");

var select344 = linear_select([
	dup311,
	part1466,
	part1467,
]);

var part1468 = match("MESSAGE#903:00527:02/2", "nwparser.p0", "an IP address%{}");

var all319 = all_match({
	processors: [
		part1465,
		select344,
		part1468,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg915 = msg("00527:02", all319);

var part1469 = match("MESSAGE#904:00527:03", "nwparser.payload", "MAC address %{macaddr->} has detected an IP conflict and has declined address %{hostip}", processor_chain([
	dup272,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg916 = msg("00527:03", part1469);

var part1470 = match("MESSAGE#905:00527:04", "nwparser.payload", "One or more DHCP-assigned IP addresses have been manually released.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg917 = msg("00527:04", part1470);

var part1471 = match("MESSAGE#906:00527:05/2", "nwparser.p0", "%{} %{interface->} is more than %{fld2->} allocated.");

var all320 = all_match({
	processors: [
		dup210,
		dup337,
		part1471,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg918 = msg("00527:05", all320);

var part1472 = match("MESSAGE#907:00527:06/0", "nwparser.payload", "IP address %{hostip->} %{p0}");

var select345 = linear_select([
	dup106,
	dup127,
]);

var part1473 = match("MESSAGE#907:00527:06/3_1", "nwparser.p0", "released from %{p0}");

var select346 = linear_select([
	dup312,
	part1473,
]);

var part1474 = match("MESSAGE#907:00527:06/4", "nwparser.p0", "%{fld2->} (%{fld1})");

var all321 = all_match({
	processors: [
		part1472,
		select345,
		dup23,
		select346,
		part1474,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg919 = msg("00527:06", all321);

var part1475 = match("MESSAGE#908:00527:07", "nwparser.payload", "One or more IP addresses have expired. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg920 = msg("00527:07", part1475);

var part1476 = match("MESSAGE#909:00527:08", "nwparser.payload", "DHCP server on interface %{interface->} received %{protocol_detail->} from %{smacaddr->} requesting out-of-scope IP address %{hostip}/%{mask->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg921 = msg("00527:08", part1476);

var part1477 = match("MESSAGE#910:00527:09/0", "nwparser.payload", "MAC address %{macaddr->} has %{disposition->} %{p0}");

var part1478 = match("MESSAGE#910:00527:09/1_0", "nwparser.p0", "address %{hostip->} (%{p0}");

var part1479 = match("MESSAGE#910:00527:09/1_1", "nwparser.p0", "%{hostip->} (%{p0}");

var select347 = linear_select([
	part1478,
	part1479,
]);

var all322 = all_match({
	processors: [
		part1477,
		select347,
		dup41,
	],
	on_success: processor_chain([
		dup272,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg922 = msg("00527:09", all322);

var part1480 = match("MESSAGE#911:00527:10", "nwparser.payload", "One or more IP addresses are expired. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg923 = msg("00527:10", part1480);

var select348 = linear_select([
	msg913,
	msg914,
	msg915,
	msg916,
	msg917,
	msg918,
	msg919,
	msg920,
	msg921,
	msg922,
	msg923,
]);

var part1481 = match("MESSAGE#912:00528", "nwparser.payload", "SCS: User '%{username}' authenticated using password :", processor_chain([
	setc("eventcategory","1302010000"),
	dup29,
	dup31,
	dup32,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg924 = msg("00528", part1481);

var part1482 = match("MESSAGE#913:00528:01", "nwparser.payload", "SCS: Connection terminated for user %{username->} from", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg925 = msg("00528:01", part1482);

var part1483 = match("MESSAGE#914:00528:02", "nwparser.payload", "SCS: Disabled for all root/vsys on device. Client host attempting connection to interface '%{interface}' with address %{hostip->} from %{saddr}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg926 = msg("00528:02", part1483);

var part1484 = match("MESSAGE#915:00528:03", "nwparser.payload", "SSH: NetScreen device %{disposition->} to identify itself to the SSH client at %{hostip}", processor_chain([
	dup203,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg927 = msg("00528:03", part1484);

var part1485 = match("MESSAGE#916:00528:04", "nwparser.payload", "SSH: Incompatible SSH version string has been received from SSH client at %{hostip}", processor_chain([
	dup203,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg928 = msg("00528:04", part1485);

var part1486 = match("MESSAGE#917:00528:05", "nwparser.payload", "SSH: %{disposition->} to send identification string to client host at %{hostip}", processor_chain([
	dup203,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg929 = msg("00528:05", part1486);

var part1487 = match("MESSAGE#918:00528:06", "nwparser.payload", "SSH: Client at %{saddr->} attempted to connect with invalid version string.", processor_chain([
	dup313,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("result","invalid version string"),
]));

var msg930 = msg("00528:06", part1487);

var part1488 = match("MESSAGE#919:00528:07/0", "nwparser.payload", "SSH: %{disposition->} to negotiate %{p0}");

var part1489 = match("MESSAGE#919:00528:07/1_1", "nwparser.p0", "MAC %{p0}");

var part1490 = match("MESSAGE#919:00528:07/1_2", "nwparser.p0", "key exchange %{p0}");

var part1491 = match("MESSAGE#919:00528:07/1_3", "nwparser.p0", "host key %{p0}");

var select349 = linear_select([
	dup88,
	part1489,
	part1490,
	part1491,
]);

var part1492 = match("MESSAGE#919:00528:07/2", "nwparser.p0", "algorithm with host %{hostip}");

var all323 = all_match({
	processors: [
		part1488,
		select349,
		part1492,
	],
	on_success: processor_chain([
		dup314,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg931 = msg("00528:07", all323);

var part1493 = match("MESSAGE#920:00528:08", "nwparser.payload", "SSH: Unsupported cipher type %{fld2->} requested from %{saddr}", processor_chain([
	dup314,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg932 = msg("00528:08", part1493);

var part1494 = match("MESSAGE#921:00528:09", "nwparser.payload", "SSH: Host client has requested NO cipher from %{saddr}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg933 = msg("00528:09", part1494);

var part1495 = match("MESSAGE#922:00528:10", "nwparser.payload", "SSH: Disabled for '%{vsys}'. Attempted connection %{disposition->} from %{saddr}:%{sport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg934 = msg("00528:10", part1495);

var part1496 = match("MESSAGE#923:00528:11", "nwparser.payload", "SSH: Disabled for %{fld2->} Attempted connection %{disposition->} from %{saddr}:%{sport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg935 = msg("00528:11", part1496);

var part1497 = match("MESSAGE#924:00528:12", "nwparser.payload", "SSH: SSH user %{username->} at %{saddr->} tried unsuccessfully to log in to %{vsys->} using the shared untrusted interface. SSH disabled on that interface.", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	setc("disposition","disabled"),
]));

var msg936 = msg("00528:12", part1497);

var part1498 = match("MESSAGE#925:00528:13/0", "nwparser.payload", "SSH: SSH client at %{saddr->} tried unsuccessfully to %{p0}");

var part1499 = match("MESSAGE#925:00528:13/1_0", "nwparser.p0", "make %{p0}");

var part1500 = match("MESSAGE#925:00528:13/1_1", "nwparser.p0", "establish %{p0}");

var select350 = linear_select([
	part1499,
	part1500,
]);

var part1501 = match("MESSAGE#925:00528:13/2", "nwparser.p0", "an SSH connection to %{p0}");

var part1502 = match("MESSAGE#925:00528:13/4", "nwparser.p0", "%{} %{interface->} with IP %{hostip->} SSH %{p0}");

var part1503 = match("MESSAGE#925:00528:13/5_0", "nwparser.p0", "not enabled %{p0}");

var select351 = linear_select([
	part1503,
	dup157,
]);

var part1504 = match("MESSAGE#925:00528:13/6", "nwparser.p0", "on that interface.%{}");

var all324 = all_match({
	processors: [
		part1498,
		select350,
		part1501,
		dup337,
		part1502,
		select351,
		part1504,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg937 = msg("00528:13", all324);

var part1505 = match("MESSAGE#926:00528:14", "nwparser.payload", "SSH: SSH client %{saddr->} unsuccessfully attempted to make an SSH connection to %{vsys->} SSH was not completely initialized for that system.", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg938 = msg("00528:14", part1505);

var part1506 = match("MESSAGE#927:00528:15/0", "nwparser.payload", "SSH: Admin user %{p0}");

var part1507 = match("MESSAGE#927:00528:15/1_1", "nwparser.p0", "%{administrator->} %{p0}");

var select352 = linear_select([
	dup315,
	part1507,
]);

var part1508 = match("MESSAGE#927:00528:15/2", "nwparser.p0", "at host %{saddr->} requested unsupported %{p0}");

var part1509 = match("MESSAGE#927:00528:15/3_0", "nwparser.p0", "PKA algorithm %{p0}");

var part1510 = match("MESSAGE#927:00528:15/3_1", "nwparser.p0", "authentication method %{p0}");

var select353 = linear_select([
	part1509,
	part1510,
]);

var all325 = all_match({
	processors: [
		part1506,
		select352,
		part1508,
		select353,
		dup108,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg939 = msg("00528:15", all325);

var part1511 = match("MESSAGE#928:00528:16", "nwparser.payload", "SCP: Admin '%{administrator}' at host %{saddr->} executed invalid scp command: '%{fld2}'", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg940 = msg("00528:16", part1511);

var part1512 = match("MESSAGE#929:00528:17", "nwparser.payload", "SCP: Disabled for '%{username}'. Attempted file transfer failed from host %{saddr}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg941 = msg("00528:17", part1512);

var part1513 = match("MESSAGE#930:00528:18/2", "nwparser.p0", "authentication successful for admin user %{p0}");

var all326 = all_match({
	processors: [
		dup316,
		dup402,
		part1513,
		dup403,
		dup320,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		setc("disposition","successful"),
		setc("event_description","authentication successful for admin user"),
	]),
});

var msg942 = msg("00528:18", all326);

var part1514 = match("MESSAGE#931:00528:26/2", "nwparser.p0", "authentication failed for admin user %{p0}");

var all327 = all_match({
	processors: [
		dup316,
		dup402,
		part1514,
		dup403,
		dup320,
	],
	on_success: processor_chain([
		dup206,
		dup29,
		dup31,
		dup54,
		dup2,
		dup4,
		dup5,
		dup302,
		dup3,
		setc("event_description","authentication failed for admin user"),
	]),
});

var msg943 = msg("00528:26", all327);

var part1515 = match("MESSAGE#932:00528:19/2", "nwparser.p0", ": SSH user %{username->} has been %{disposition->} using password from %{saddr}:%{sport}");

var all328 = all_match({
	processors: [
		dup321,
		dup404,
		part1515,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg944 = msg("00528:19", all328);

var part1516 = match("MESSAGE#933:00528:20/2", "nwparser.p0", ": Connection has been %{disposition->} for admin user %{administrator->} at %{saddr}:%{sport}");

var all329 = all_match({
	processors: [
		dup321,
		dup404,
		part1516,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg945 = msg("00528:20", all329);

var part1517 = match("MESSAGE#934:00528:21", "nwparser.payload", "SCS: SSH user %{username->} at %{saddr}:%{sport->} has requested PKA RSA authentication, which is not supported for that client.", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg946 = msg("00528:21", part1517);

var part1518 = match("MESSAGE#935:00528:22/0", "nwparser.payload", "SCS: SSH client at %{saddr->} has attempted to make an SCS connection to %{p0}");

var part1519 = match("MESSAGE#935:00528:22/2", "nwparser.p0", "%{} %{interface->} with IP %{hostip->} but %{disposition->} because SCS is not enabled for that interface.");

var all330 = all_match({
	processors: [
		part1518,
		dup337,
		part1519,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
		setc("result","SCS is not enabled for that interface"),
	]),
});

var msg947 = msg("00528:22", all330);

var part1520 = match("MESSAGE#936:00528:23", "nwparser.payload", "SCS: SSH client at %{saddr}:%{sport->} has %{disposition->} to make an SCS connection to vsys %{vsys->} because SCS cannot generate the host and server keys before timing out.", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	setc("result","SCS cannot generate the host and server keys before timing out"),
]));

var msg948 = msg("00528:23", part1520);

var part1521 = match("MESSAGE#937:00528:24", "nwparser.payload", "SSH: %{change_attribute->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup281,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg949 = msg("00528:24", part1521);

var part1522 = match("MESSAGE#938:00528:25/0", "nwparser.payload", "SSH: Admin %{p0}");

var part1523 = match("MESSAGE#938:00528:25/2", "nwparser.p0", "at host %{saddr->} attempted to be authenticated with no authentication methods enabled.");

var all331 = all_match({
	processors: [
		part1522,
		dup403,
		part1523,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup3,
	]),
});

var msg950 = msg("00528:25", all331);

var select354 = linear_select([
	msg924,
	msg925,
	msg926,
	msg927,
	msg928,
	msg929,
	msg930,
	msg931,
	msg932,
	msg933,
	msg934,
	msg935,
	msg936,
	msg937,
	msg938,
	msg939,
	msg940,
	msg941,
	msg942,
	msg943,
	msg944,
	msg945,
	msg946,
	msg947,
	msg948,
	msg949,
	msg950,
]);

var part1524 = match("MESSAGE#939:00529/1_0", "nwparser.p0", "manually %{p0}");

var part1525 = match("MESSAGE#939:00529/1_1", "nwparser.p0", "automatically %{p0}");

var select355 = linear_select([
	part1524,
	part1525,
]);

var part1526 = match("MESSAGE#939:00529/2", "nwparser.p0", "refreshed%{}");

var all332 = all_match({
	processors: [
		dup63,
		select355,
		part1526,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg951 = msg("00529", all332);

var part1527 = match("MESSAGE#940:00529:01/0", "nwparser.payload", "DNS entries have been refreshed by %{p0}");

var part1528 = match("MESSAGE#940:00529:01/1_0", "nwparser.p0", "state change%{}");

var part1529 = match("MESSAGE#940:00529:01/1_1", "nwparser.p0", "HA%{}");

var select356 = linear_select([
	part1528,
	part1529,
]);

var all333 = all_match({
	processors: [
		part1527,
		select356,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg952 = msg("00529:01", all333);

var select357 = linear_select([
	msg951,
	msg952,
]);

var part1530 = match("MESSAGE#941:00530", "nwparser.payload", "An IP conflict has been detected and the DHCP client has declined address %{hostip}", processor_chain([
	dup272,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg953 = msg("00530", part1530);

var part1531 = match("MESSAGE#942:00530:01/0", "nwparser.payload", "DHCP client IP %{hostip->} for the %{p0}");

var part1532 = match("MESSAGE#942:00530:01/2", "nwparser.p0", "%{} %{interface->} has been manually released");

var all334 = all_match({
	processors: [
		part1531,
		dup337,
		part1532,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg954 = msg("00530:01", all334);

var part1533 = match("MESSAGE#943:00530:02", "nwparser.payload", "DHCP client is unable to get an IP address for the %{interface->} interface", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg955 = msg("00530:02", part1533);

var part1534 = match("MESSAGE#944:00530:03", "nwparser.payload", "DHCP client lease for %{hostip->} has expired", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg956 = msg("00530:03", part1534);

var part1535 = match("MESSAGE#945:00530:04", "nwparser.payload", "DHCP server %{hostip->} has assigned the untrust Interface %{interface->} with lease %{fld2}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg957 = msg("00530:04", part1535);

var part1536 = match("MESSAGE#946:00530:05", "nwparser.payload", "DHCP server %{hostip->} has assigned the %{interface->} interface %{fld2->} with lease %{fld3}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg958 = msg("00530:05", part1536);

var part1537 = match("MESSAGE#947:00530:06", "nwparser.payload", "DHCP client is unable to get IP address for the untrust interface.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg959 = msg("00530:06", part1537);

var select358 = linear_select([
	msg953,
	msg954,
	msg955,
	msg956,
	msg957,
	msg958,
	msg959,
]);

var part1538 = match("MESSAGE#948:00531/0", "nwparser.payload", "System clock configurations have been changed by admin %{p0}");

var all335 = all_match({
	processors: [
		part1538,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg960 = msg("00531", all335);

var part1539 = match("MESSAGE#949:00531:01", "nwparser.payload", "failed to get clock through NTP%{}", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg961 = msg("00531:01", part1539);

var part1540 = match("MESSAGE#950:00531:02", "nwparser.payload", "The system clock has been updated through NTP.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg962 = msg("00531:02", part1540);

var part1541 = match("MESSAGE#951:00531:03/0", "nwparser.payload", "The system clock was updated from %{type->} NTP server type %{hostname->} with a%{p0}");

var part1542 = match("MESSAGE#951:00531:03/1_0", "nwparser.p0", " ms %{p0}");

var select359 = linear_select([
	part1542,
	dup115,
]);

var part1543 = match("MESSAGE#951:00531:03/2", "nwparser.p0", "adjustment of %{fld3}. Authentication was %{fld4}. Update mode was %{p0}");

var part1544 = match("MESSAGE#951:00531:03/3_0", "nwparser.p0", "%{fld5}(%{fld2})");

var part1545 = match_copy("MESSAGE#951:00531:03/3_1", "nwparser.p0", "fld5");

var select360 = linear_select([
	part1544,
	part1545,
]);

var all336 = all_match({
	processors: [
		part1541,
		select359,
		part1543,
		select360,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
		dup146,
	]),
});

var msg963 = msg("00531:03", all336);

var part1546 = match("MESSAGE#952:00531:04/0", "nwparser.payload", "The NetScreen device is attempting to contact the %{p0}");

var part1547 = match("MESSAGE#952:00531:04/1_0", "nwparser.p0", "primary backup %{p0}");

var part1548 = match("MESSAGE#952:00531:04/1_1", "nwparser.p0", "secondary backup %{p0}");

var select361 = linear_select([
	part1547,
	part1548,
	dup189,
]);

var part1549 = match("MESSAGE#952:00531:04/2", "nwparser.p0", "NTP server %{hostname}");

var all337 = all_match({
	processors: [
		part1546,
		select361,
		part1549,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg964 = msg("00531:04", all337);

var part1550 = match("MESSAGE#953:00531:05", "nwparser.payload", "No NTP server could be contacted. (%{fld1})", processor_chain([
	dup86,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg965 = msg("00531:05", part1550);

var part1551 = match("MESSAGE#954:00531:06", "nwparser.payload", "Network Time Protocol adjustment of %{fld2->} from NTP server %{hostname->} exceeds the allowed adjustment of %{fld3}. (%{fld1})", processor_chain([
	dup86,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg966 = msg("00531:06", part1551);

var part1552 = match("MESSAGE#955:00531:07", "nwparser.payload", "No acceptable time could be obtained from any NTP server. (%{fld1})", processor_chain([
	dup86,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg967 = msg("00531:07", part1552);

var part1553 = match("MESSAGE#956:00531:08", "nwparser.payload", "Administrator %{administrator->} changed the %{change_attribute->} from %{change_old->} to %{change_new->} (by %{fld3->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport}) (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg968 = msg("00531:08", part1553);

var part1554 = match("MESSAGE#957:00531:09", "nwparser.payload", "Network Time Protocol settings changed. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg969 = msg("00531:09", part1554);

var part1555 = match("MESSAGE#958:00531:10", "nwparser.payload", "NTP server is %{disposition->} on interface %{interface->} (%{fld1})", processor_chain([
	dup86,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg970 = msg("00531:10", part1555);

var part1556 = match("MESSAGE#959:00531:11", "nwparser.payload", "The system clock will be changed from %{change_old->} to %{change_new->} received from primary NTP server %{hostip->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	setc("event_description","system clock changed based on receive from primary NTP server"),
]));

var msg971 = msg("00531:11", part1556);

var part1557 = match("MESSAGE#1223:00531:12", "nwparser.payload", "%{fld35->} NTP server %{saddr->} could not be contacted. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg972 = msg("00531:12", part1557);

var select362 = linear_select([
	msg960,
	msg961,
	msg962,
	msg963,
	msg964,
	msg965,
	msg966,
	msg967,
	msg968,
	msg969,
	msg970,
	msg971,
	msg972,
]);

var part1558 = match("MESSAGE#960:00533", "nwparser.payload", "VIP server %{hostip->} is now responding", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg973 = msg("00533", part1558);

var part1559 = match("MESSAGE#961:00534", "nwparser.payload", "%{fld2->} has been cleared", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg974 = msg("00534", part1559);

var part1560 = match("MESSAGE#962:00535", "nwparser.payload", "Cannot find the CA certificate with distinguished name %{fld2}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg975 = msg("00535", part1560);

var part1561 = match("MESSAGE#963:00535:01", "nwparser.payload", "Distinguished name %{dn->} in the X509 certificate request is %{disposition}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg976 = msg("00535:01", part1561);

var part1562 = match("MESSAGE#964:00535:02", "nwparser.payload", "Local certificate with distinguished name %{dn->} is %{disposition}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg977 = msg("00535:02", part1562);

var part1563 = match("MESSAGE#965:00535:03", "nwparser.payload", "PKCS #7 data cannot be decapsulated%{}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg978 = msg("00535:03", part1563);

var part1564 = match("MESSAGE#966:00535:04", "nwparser.payload", "SCEP_FAILURE message has been received from the CA%{}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("result","SCEP_FAILURE message"),
]));

var msg979 = msg("00535:04", part1564);

var part1565 = match("MESSAGE#967:00535:05", "nwparser.payload", "PKI error message has been received: %{result}", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg980 = msg("00535:05", part1565);

var part1566 = match("MESSAGE#968:00535:06", "nwparser.payload", "PKI: Saved CA configuration (CA cert subject name %{dn}). (%{event_time_string})", processor_chain([
	dup314,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("event_description","Saved CA configuration - cert subject name"),
]));

var msg981 = msg("00535:06", part1566);

var select363 = linear_select([
	msg975,
	msg976,
	msg977,
	msg978,
	msg979,
	msg980,
	msg981,
]);

var part1567 = match("MESSAGE#969:00536:49/0", "nwparser.payload", "IKE %{hostip->} %{p0}");

var part1568 = match("MESSAGE#969:00536:49/1_0", "nwparser.p0", "Phase 2 msg ID %{sessionid}: %{disposition}. %{p0}");

var part1569 = match("MESSAGE#969:00536:49/1_1", "nwparser.p0", "Phase 1: %{disposition->} %{p0}");

var part1570 = match("MESSAGE#969:00536:49/1_2", "nwparser.p0", "phase 2:%{disposition}. %{p0}");

var part1571 = match("MESSAGE#969:00536:49/1_3", "nwparser.p0", "phase 1:%{disposition}. %{p0}");

var select364 = linear_select([
	part1568,
	part1569,
	part1570,
	part1571,
]);

var all338 = all_match({
	processors: [
		part1567,
		select364,
		dup10,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup9,
		dup3,
		dup4,
		dup5,
	]),
});

var msg982 = msg("00536:49", all338);

var part1572 = match("MESSAGE#970:00536", "nwparser.payload", "UDP packets have been received from %{saddr}/%{sport->} at interface %{interface->} at %{daddr}/%{dport}", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup3,
	dup61,
]));

var msg983 = msg("00536", part1572);

var part1573 = match("MESSAGE#971:00536:01", "nwparser.payload", "Attempt to set tunnel (%{fld2}) without IP address at both end points! Check outgoing interface.", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg984 = msg("00536:01", part1573);

var part1574 = match("MESSAGE#972:00536:02", "nwparser.payload", "Gateway %{fld2->} at %{hostip->} in %{fld4->} mode with ID: %{fld3->} has been %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg985 = msg("00536:02", part1574);

var part1575 = match("MESSAGE#973:00536:03", "nwparser.payload", "IKE gateway %{fld2->} has been %{disposition}. %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg986 = msg("00536:03", part1575);

var part1576 = match("MESSAGE#974:00536:04", "nwparser.payload", "VPN monitoring for VPN %{group->} has deactivated the SA with ID %{fld2}.", processor_chain([
	setc("eventcategory","1801010100"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg987 = msg("00536:04", part1576);

var part1577 = match("MESSAGE#975:00536:05", "nwparser.payload", "VPN ID number cannot be assigned%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg988 = msg("00536:05", part1577);

var part1578 = match("MESSAGE#976:00536:06", "nwparser.payload", "Local gateway IP address has changed to %{fld2}. VPNs cannot terminate at an interface with IP %{hostip}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg989 = msg("00536:06", part1578);

var part1579 = match("MESSAGE#977:00536:07", "nwparser.payload", "Local gateway IP address has changed from %{change_old->} to another setting", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg990 = msg("00536:07", part1579);

var part1580 = match("MESSAGE#978:00536:08", "nwparser.payload", "IKE %{hostip}: Sent initial contact notification message", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg991 = msg("00536:08", part1580);

var part1581 = match("MESSAGE#979:00536:09", "nwparser.payload", "IKE %{hostip}: Sent initial contact notification", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg992 = msg("00536:09", part1581);

var part1582 = match("MESSAGE#980:00536:10", "nwparser.payload", "IKE %{hostip}: Responded to a packet with a bad SPI after rebooting", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg993 = msg("00536:10", part1582);

var part1583 = match("MESSAGE#981:00536:11", "nwparser.payload", "IKE %{hostip}: Removed Phase 2 SAs after receiving a notification message", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg994 = msg("00536:11", part1583);

var part1584 = match("MESSAGE#982:00536:12", "nwparser.payload", "IKE %{hostip}: Rejected first Phase 1 packet from an unrecognized source", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg995 = msg("00536:12", part1584);

var part1585 = match("MESSAGE#983:00536:13", "nwparser.payload", "IKE %{hostip}: Rejected an initial Phase 1 packet from an unrecognized peer gateway", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg996 = msg("00536:13", part1585);

var part1586 = match("MESSAGE#984:00536:14/0", "nwparser.payload", "IKE %{hostip}: Received initial contact notification and removed Phase %{p0}");

var part1587 = match("MESSAGE#984:00536:14/2", "nwparser.p0", "SAs%{}");

var all339 = all_match({
	processors: [
		part1586,
		dup383,
		part1587,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg997 = msg("00536:14", all339);

var part1588 = match("MESSAGE#985:00536:50", "nwparser.payload", "IKE %{hostip}: Received a notification message for %{disposition}. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup9,
	dup3,
	dup4,
	dup5,
]));

var msg998 = msg("00536:50", part1588);

var part1589 = match("MESSAGE#986:00536:15", "nwparser.payload", "IKE %{hostip}: Received incorrect ID payload: IP address %{fld2->} instead of IP address %{fld3}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg999 = msg("00536:15", part1589);

var part1590 = match("MESSAGE#987:00536:16", "nwparser.payload", "IKE %{hostip}: Phase 2 negotiation request is already in the task list", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1000 = msg("00536:16", part1590);

var part1591 = match("MESSAGE#988:00536:17", "nwparser.payload", "IKE %{hostip}: Heartbeats have been lost %{fld2->} times", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1001 = msg("00536:17", part1591);

var part1592 = match("MESSAGE#989:00536:18", "nwparser.payload", "IKE %{hostip}: Dropped peer packet because no policy uses the peer configuration", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1002 = msg("00536:18", part1592);

var part1593 = match("MESSAGE#990:00536:19", "nwparser.payload", "IKE %{hostip}: Dropped packet because remote gateway OK is not used in any VPN tunnel configurations", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1003 = msg("00536:19", part1593);

var part1594 = match("MESSAGE#991:00536:20", "nwparser.payload", "IKE %{hostip}: Added the initial contact task to the task list", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1004 = msg("00536:20", part1594);

var part1595 = match("MESSAGE#992:00536:21", "nwparser.payload", "IKE %{hostip}: Added Phase 2 session tasks to the task list", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1005 = msg("00536:21", part1595);

var part1596 = match("MESSAGE#993:00536:22", "nwparser.payload", "IKE %{hostip->} Phase 1 : %{disposition->} proposals from peer. Negotiations failed", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("result","Negotiations failed"),
]));

var msg1006 = msg("00536:22", part1596);

var part1597 = match("MESSAGE#994:00536:23", "nwparser.payload", "IKE %{hostip->} Phase 1 : Aborted negotiations because the time limit has elapsed", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("result","The time limit has elapsed"),
	setc("disposition","Aborted"),
]));

var msg1007 = msg("00536:23", part1597);

var part1598 = match("MESSAGE#995:00536:24", "nwparser.payload", "IKE %{hostip->} Phase 2: Received a message but did not check a policy because id-mode is set to IP or policy-checking is disabled", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1008 = msg("00536:24", part1598);

var part1599 = match("MESSAGE#996:00536:25", "nwparser.payload", "IKE %{hostip->} Phase 2: Received DH group %{fld2->} instead of expected group %{fld3->} for PFS", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1009 = msg("00536:25", part1599);

var part1600 = match("MESSAGE#997:00536:26", "nwparser.payload", "IKE %{hostip->} Phase 2: No policy exists for the proxy ID received: local ID %{fld2->} remote ID %{fld3}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1010 = msg("00536:26", part1600);

var part1601 = match("MESSAGE#998:00536:27", "nwparser.payload", "IKE %{hostip->} Phase 1: RSA private key is needed to sign packets", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1011 = msg("00536:27", part1601);

var part1602 = match("MESSAGE#999:00536:28", "nwparser.payload", "IKE %{hostip->} Phase 1: Aggressive mode negotiations have %{disposition}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1012 = msg("00536:28", part1602);

var part1603 = match("MESSAGE#1000:00536:29", "nwparser.payload", "IKE %{hostip->} Phase 1: Vendor ID payload indicates that the peer does not support NAT-T", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1013 = msg("00536:29", part1603);

var part1604 = match("MESSAGE#1001:00536:30", "nwparser.payload", "IKE %{hostip->} Phase 1: Retransmission limit has been reached", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1014 = msg("00536:30", part1604);

var part1605 = match("MESSAGE#1002:00536:31", "nwparser.payload", "IKE %{hostip->} Phase 1: Received an invalid RSA signature", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1015 = msg("00536:31", part1605);

var part1606 = match("MESSAGE#1003:00536:32", "nwparser.payload", "IKE %{hostip->} Phase 1: Received an incorrect public key authentication method", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1016 = msg("00536:32", part1606);

var part1607 = match("MESSAGE#1004:00536:33", "nwparser.payload", "IKE %{hostip->} Phase 1: No private key exists to sign packets", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1017 = msg("00536:33", part1607);

var part1608 = match("MESSAGE#1005:00536:34", "nwparser.payload", "IKE %{hostip->} Phase 1: Main mode packet has arrived with ID type IP address but no user configuration was found for that ID", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1018 = msg("00536:34", part1608);

var part1609 = match("MESSAGE#1006:00536:35", "nwparser.payload", "IKE %{hostip->} Phase 1: IKE initiator has detected NAT in front of the local device", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1019 = msg("00536:35", part1609);

var part1610 = match("MESSAGE#1007:00536:36/0", "nwparser.payload", "IKE %{hostip->} Phase 1: Discarded a second initial packet%{p0}");

var part1611 = match("MESSAGE#1007:00536:36/2", "nwparser.p0", "%{}which arrived within %{fld2->} after the first");

var all340 = all_match({
	processors: [
		part1610,
		dup401,
		part1611,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1020 = msg("00536:36", all340);

var part1612 = match("MESSAGE#1008:00536:37", "nwparser.payload", "IKE %{hostip->} Phase 1: Completed Aggressive mode negotiations with a %{fld2->} lifetime", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1021 = msg("00536:37", part1612);

var part1613 = match("MESSAGE#1009:00536:38", "nwparser.payload", "IKE %{hostip->} Phase 1: Certificate received has a subject name that does not match the ID payload", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1022 = msg("00536:38", part1613);

var part1614 = match("MESSAGE#1010:00536:39", "nwparser.payload", "IKE %{hostip->} Phase 1: Certificate received has a different IP address %{fld2->} than expected", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1023 = msg("00536:39", part1614);

var part1615 = match("MESSAGE#1011:00536:40", "nwparser.payload", "IKE %{hostip->} Phase 1: Cannot use a preshared key because the peer%{quote}s gateway has a dynamic IP address and negotiations are in Main mode", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1024 = msg("00536:40", part1615);

var part1616 = match("MESSAGE#1012:00536:47", "nwparser.payload", "IKE %{hostip->} Phase 1: Initiated negotiations in Aggressive mode", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1025 = msg("00536:47", part1616);

var part1617 = match("MESSAGE#1013:00536:41", "nwparser.payload", "IKE %{hostip->} Phase 1: Cannot verify RSA signature", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1026 = msg("00536:41", part1617);

var part1618 = match("MESSAGE#1014:00536:42", "nwparser.payload", "IKE %{hostip->} Phase 1: Initiated Main mode negotiations", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1027 = msg("00536:42", part1618);

var part1619 = match("MESSAGE#1015:00536:43", "nwparser.payload", "IKE %{hostip->} Phase 2: Initiated negotiations", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1028 = msg("00536:43", part1619);

var part1620 = match("MESSAGE#1016:00536:44", "nwparser.payload", "IKE %{hostip}: Changed heartbeat interval to %{fld2}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1029 = msg("00536:44", part1620);

var part1621 = match("MESSAGE#1017:00536:45", "nwparser.payload", "IKE %{hostip}: Heartbeats have been %{disposition->} because %{result}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1030 = msg("00536:45", part1621);

var part1622 = match("MESSAGE#1018:00536:48", "nwparser.payload", "Received an IKE packet on %{interface->} from %{saddr}:%{sport->} to %{daddr}:%{dport}/%{fld1}. Cookies: %{ike_cookie1}, %{ike_cookie2}. (%{event_time_string})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
	setc("event_description","Received an IKE packet on interface"),
]));

var msg1031 = msg("00536:48", part1622);

var part1623 = match("MESSAGE#1019:00536:46", "nwparser.payload", "IKE %{hostip}: Received a bad SPI", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1032 = msg("00536:46", part1623);

var select365 = linear_select([
	msg982,
	msg983,
	msg984,
	msg985,
	msg986,
	msg987,
	msg988,
	msg989,
	msg990,
	msg991,
	msg992,
	msg993,
	msg994,
	msg995,
	msg996,
	msg997,
	msg998,
	msg999,
	msg1000,
	msg1001,
	msg1002,
	msg1003,
	msg1004,
	msg1005,
	msg1006,
	msg1007,
	msg1008,
	msg1009,
	msg1010,
	msg1011,
	msg1012,
	msg1013,
	msg1014,
	msg1015,
	msg1016,
	msg1017,
	msg1018,
	msg1019,
	msg1020,
	msg1021,
	msg1022,
	msg1023,
	msg1024,
	msg1025,
	msg1026,
	msg1027,
	msg1028,
	msg1029,
	msg1030,
	msg1031,
	msg1032,
]);

var part1624 = match("MESSAGE#1020:00537", "nwparser.payload", "PPPoE %{disposition->} to establish a session: %{info}", processor_chain([
	dup18,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg1033 = msg("00537", part1624);

var part1625 = match("MESSAGE#1021:00537:01", "nwparser.payload", "PPPoE session shuts down: %{result}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1034 = msg("00537:01", part1625);

var part1626 = match("MESSAGE#1022:00537:02", "nwparser.payload", "The Point-to-Point over Ethernet (PPPoE) connection failed to establish a session: %{result}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1035 = msg("00537:02", part1626);

var part1627 = match("MESSAGE#1023:00537:03", "nwparser.payload", "PPPoE session has successfully established%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1036 = msg("00537:03", part1627);

var select366 = linear_select([
	msg1033,
	msg1034,
	msg1035,
	msg1036,
]);

var part1628 = match("MESSAGE#1024:00538/0", "nwparser.payload", "NACN failed to register to Policy Manager %{fld2->} because %{p0}");

var select367 = linear_select([
	dup111,
	dup119,
]);

var part1629 = match("MESSAGE#1024:00538/2", "nwparser.p0", "%{result}");

var all341 = all_match({
	processors: [
		part1628,
		select367,
		part1629,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1037 = msg("00538", all341);

var part1630 = match("MESSAGE#1025:00538:01", "nwparser.payload", "NACN successfully registered to Policy Manager %{fld2}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1038 = msg("00538:01", part1630);

var part1631 = match("MESSAGE#1026:00538:02", "nwparser.payload", "The NACN protocol has started for Policy Manager %{fld2->} on hostname %{hostname->} IP address %{hostip->} port %{network_port}.", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1039 = msg("00538:02", part1631);

var part1632 = match("MESSAGE#1027:00538:03", "nwparser.payload", "Cannot connect to NSM Server at %{hostip->} (%{fld2->} connect attempt(s)) %{fld3}", processor_chain([
	dup19,
	dup2,
	dup4,
	dup5,
	dup3,
]));

var msg1040 = msg("00538:03", part1632);

var part1633 = match("MESSAGE#1028:00538:04", "nwparser.payload", "Device is not known to Global PRO data collector at %{hostip}", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1041 = msg("00538:04", part1633);

var part1634 = match("MESSAGE#1029:00538:05/0", "nwparser.payload", "Lost %{p0}");

var part1635 = match("MESSAGE#1029:00538:05/1_0", "nwparser.p0", "socket connection%{p0}");

var part1636 = match("MESSAGE#1029:00538:05/1_1", "nwparser.p0", "connection%{p0}");

var select368 = linear_select([
	part1635,
	part1636,
]);

var part1637 = match("MESSAGE#1029:00538:05/2", "nwparser.p0", "%{}to Global PRO data collector at %{hostip}");

var all342 = all_match({
	processors: [
		part1634,
		select368,
		part1637,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1042 = msg("00538:05", all342);

var part1638 = match("MESSAGE#1030:00538:06/0", "nwparser.payload", "Device has connected to the Global PRO%{p0}");

var part1639 = match("MESSAGE#1030:00538:06/1_0", "nwparser.p0", " %{fld2->} primary data collector at %{p0}");

var part1640 = match("MESSAGE#1030:00538:06/1_1", "nwparser.p0", " primary data collector at %{p0}");

var select369 = linear_select([
	part1639,
	part1640,
]);

var part1641 = match_copy("MESSAGE#1030:00538:06/2", "nwparser.p0", "hostip");

var all343 = all_match({
	processors: [
		part1638,
		select369,
		part1641,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1043 = msg("00538:06", all343);

var part1642 = match("MESSAGE#1031:00538:07/0", "nwparser.payload", "Connection to Global PRO data collector at %{hostip->} has%{p0}");

var part1643 = match("MESSAGE#1031:00538:07/1_0", "nwparser.p0", " been%{p0}");

var select370 = linear_select([
	part1643,
	dup16,
]);

var all344 = all_match({
	processors: [
		part1642,
		select370,
		dup136,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1044 = msg("00538:07", all344);

var part1644 = match("MESSAGE#1032:00538:08", "nwparser.payload", "Cannot connect to Global PRO data collector at %{hostip}", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1045 = msg("00538:08", part1644);

var part1645 = match("MESSAGE#1033:00538:09", "nwparser.payload", "NSM: Connected to NSM server at %{hostip->} (%{info}) (%{fld1})", processor_chain([
	dup301,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	setc("event_description","Connected to NSM server"),
]));

var msg1046 = msg("00538:09", part1645);

var part1646 = match("MESSAGE#1034:00538:10/0", "nwparser.payload", "NSM: Connection to NSM server at %{hostip->} is down. Reason: %{resultcode}, %{result->} (%{p0}");

var part1647 = match("MESSAGE#1034:00538:10/1_0", "nwparser.p0", "%{info}) (%{fld1})");

var select371 = linear_select([
	part1647,
	dup41,
]);

var all345 = all_match({
	processors: [
		part1646,
		select371,
	],
	on_success: processor_chain([
		dup198,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
		setc("event_description","Connection to NSM server is down"),
	]),
});

var msg1047 = msg("00538:10", all345);

var part1648 = match("MESSAGE#1035:00538:11", "nwparser.payload", "NSM: Cannot connect to NSM server at %{hostip}. Reason: %{resultcode}, %{result->} (%{info}) (%{fld2->} connect attempt(s)) (%{fld1})", processor_chain([
	dup198,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	dup323,
]));

var msg1048 = msg("00538:11", part1648);

var part1649 = match("MESSAGE#1036:00538:12", "nwparser.payload", "NSM: Cannot connect to NSM server at %{hostip}. Reason: %{resultcode}, %{result->} (%{info}) (%{fld1})", processor_chain([
	dup198,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	dup323,
]));

var msg1049 = msg("00538:12", part1649);

var part1650 = match("MESSAGE#1037:00538:13", "nwparser.payload", "NSM: Sent 2B message (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	setc("event_description","Sent 2B message"),
]));

var msg1050 = msg("00538:13", part1650);

var select372 = linear_select([
	msg1037,
	msg1038,
	msg1039,
	msg1040,
	msg1041,
	msg1042,
	msg1043,
	msg1044,
	msg1045,
	msg1046,
	msg1047,
	msg1048,
	msg1049,
	msg1050,
]);

var part1651 = match("MESSAGE#1038:00539", "nwparser.payload", "No IP address in L2TP IP pool for user %{username}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1051 = msg("00539", part1651);

var part1652 = match("MESSAGE#1039:00539:01", "nwparser.payload", "No L2TP IP pool for user %{username}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1052 = msg("00539:01", part1652);

var part1653 = match("MESSAGE#1040:00539:02", "nwparser.payload", "Cannot allocate IP addr from Pool %{group_object->} for user %{username}", processor_chain([
	dup117,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1053 = msg("00539:02", part1653);

var part1654 = match("MESSAGE#1041:00539:03", "nwparser.payload", "Dialup HDLC PPP failed to establish a session: %{fld2}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1054 = msg("00539:03", part1654);

var part1655 = match("MESSAGE#1042:00539:04", "nwparser.payload", "Dialup HDLC PPP session has successfully established.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1055 = msg("00539:04", part1655);

var part1656 = match("MESSAGE#1043:00539:05", "nwparser.payload", "No IP Pool has been assigned. You cannot allocate an IP address%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1056 = msg("00539:05", part1656);

var part1657 = match("MESSAGE#1044:00539:06", "nwparser.payload", "PPP settings changed.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1057 = msg("00539:06", part1657);

var select373 = linear_select([
	msg1051,
	msg1052,
	msg1053,
	msg1054,
	msg1055,
	msg1056,
	msg1057,
]);

var part1658 = match("MESSAGE#1045:00541", "nwparser.payload", "ScreenOS %{fld2->} serial # %{serial_number}: Asset recovery has been %{disposition}", processor_chain([
	dup324,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1058 = msg("00541", part1658);

var part1659 = match("MESSAGE#1216:00541:01", "nwparser.payload", "Neighbor router ID - %{fld2->} IP address - %{hostip->} changed its state to %{change_new}. (%{fld1})", processor_chain([
	dup273,
	dup9,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1059 = msg("00541:01", part1659);

var part1660 = match("MESSAGE#1218:00541:02", "nwparser.payload", "The system killed OSPF neighbor because the current router could not see itself in the hello packet. Neighbor changed state from %{change_old->} to %{change_new->} state, (neighbor router-id 1%{fld2}, ip-address %{hostip}). (%{fld1})", processor_chain([
	dup273,
	dup9,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1060 = msg("00541:02", part1660);

var part1661 = match("MESSAGE#1219:00541:03/0", "nwparser.payload", "LSA in following area aged out: LSA area ID %{fld3}, LSA ID %{fld4}, router ID %{fld2}, type %{fld7->} in OSPF. (%{fld1})%{p0}");

var part1662 = match("MESSAGE#1219:00541:03/1_0", "nwparser.p0", "\u003c\u003c%{fld16}>");

var select374 = linear_select([
	part1662,
	dup21,
]);

var all346 = all_match({
	processors: [
		part1661,
		select374,
	],
	on_success: processor_chain([
		dup44,
		dup9,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1061 = msg("00541:03", all346);

var select375 = linear_select([
	msg1058,
	msg1059,
	msg1060,
	msg1061,
]);

var part1663 = match("MESSAGE#1046:00542", "nwparser.payload", "BGP of vr: %{node}, prefix adding: %{fld2}, ribin overflow %{fld3->} times (max rib-in %{fld4})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1062 = msg("00542", part1663);

var part1664 = match("MESSAGE#1047:00543/0", "nwparser.payload", "Access for %{p0}");

var part1665 = match("MESSAGE#1047:00543/1_0", "nwparser.p0", "WebAuth firewall %{p0}");

var part1666 = match("MESSAGE#1047:00543/1_1", "nwparser.p0", "firewall %{p0}");

var select376 = linear_select([
	part1665,
	part1666,
]);

var part1667 = match("MESSAGE#1047:00543/2", "nwparser.p0", "user %{username->} %{space}at %{hostip->} (accepted at %{fld2->} for duration %{duration->} via the %{logon_type}) %{p0}");

var part1668 = match("MESSAGE#1047:00543/3_0", "nwparser.p0", "by policy id %{policy_id->} is %{p0}");

var select377 = linear_select([
	part1668,
	dup106,
]);

var part1669 = match("MESSAGE#1047:00543/4", "nwparser.p0", "now over (%{fld1})");

var all347 = all_match({
	processors: [
		part1664,
		select376,
		part1667,
		select377,
		part1669,
	],
	on_success: processor_chain([
		dup281,
		dup2,
		dup4,
		dup5,
		dup9,
		dup3,
	]),
});

var msg1063 = msg("00543", all347);

var part1670 = match("MESSAGE#1048:00544", "nwparser.payload", "User %{username->} [ of group %{group->} ] at %{hostip->} has been challenged by the RADIUS server at %{daddr}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup3,
	dup60,
	setc("action","RADIUS server challenge"),
]));

var msg1064 = msg("00544", part1670);

var part1671 = match("MESSAGE#1049:00546", "nwparser.payload", "delete-route-> trust-vr: %{fld2}", processor_chain([
	dup281,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1065 = msg("00546", part1671);

var part1672 = match("MESSAGE#1050:00547", "nwparser.payload", "AV: Content from %{saddr}:%{sport}->%{daddr}:%{dport->} was not scanned because max content size was exceeded.", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup3,
	dup61,
]));

var msg1066 = msg("00547", part1672);

var part1673 = match("MESSAGE#1051:00547:01", "nwparser.payload", "AV: Content from %{saddr}:%{sport}->%{daddr}:%{dport->} was not scanned due to a scan engine error or constraint.", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup3,
	dup61,
]));

var msg1067 = msg("00547:01", part1673);

var part1674 = match("MESSAGE#1052:00547:02", "nwparser.payload", "AV object scan-mgr data has been %{disposition}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1068 = msg("00547:02", part1674);

var part1675 = match("MESSAGE#1053:00547:03/0", "nwparser.payload", "AV: Content from %{location_desc}, http url: %{url}, is passed %{p0}");

var part1676 = match("MESSAGE#1053:00547:03/1_0", "nwparser.p0", "due to %{p0}");

var part1677 = match("MESSAGE#1053:00547:03/1_1", "nwparser.p0", "because %{p0}");

var select378 = linear_select([
	part1676,
	part1677,
]);

var part1678 = match("MESSAGE#1053:00547:03/2", "nwparser.p0", "%{result}. (%{event_time_string})");

var all348 = all_match({
	processors: [
		part1675,
		select378,
		part1678,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
		setc("event_description","Content is bypassed for connection"),
	]),
});

var msg1069 = msg("00547:03", all348);

var select379 = linear_select([
	msg1066,
	msg1067,
	msg1068,
	msg1069,
]);

var part1679 = match("MESSAGE#1054:00549", "nwparser.payload", "add-route-> untrust-vr: %{fld2}", processor_chain([
	dup281,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1070 = msg("00549", part1679);

var part1680 = match("MESSAGE#1055:00551", "nwparser.payload", "Error %{resultcode->} occurred during configlet file processing.", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1071 = msg("00551", part1680);

var part1681 = match("MESSAGE#1056:00551:01", "nwparser.payload", "Error %{resultcode->} occurred, causing failure to establish secure management with Management System.", processor_chain([
	dup86,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1072 = msg("00551:01", part1681);

var part1682 = match("MESSAGE#1057:00551:02/0", "nwparser.payload", "Configlet file %{p0}");

var part1683 = match("MESSAGE#1057:00551:02/1_0", "nwparser.p0", "decryption %{p0}");

var select380 = linear_select([
	part1683,
	dup89,
]);

var all349 = all_match({
	processors: [
		part1682,
		select380,
		dup128,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1073 = msg("00551:02", all349);

var part1684 = match("MESSAGE#1058:00551:03", "nwparser.payload", "Rapid Deployment cannot start because gateway has undergone configuration changes. (%{fld1})", processor_chain([
	dup18,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1074 = msg("00551:03", part1684);

var part1685 = match("MESSAGE#1059:00551:04", "nwparser.payload", "Secure management established successfully with remote server. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1075 = msg("00551:04", part1685);

var select381 = linear_select([
	msg1071,
	msg1072,
	msg1073,
	msg1074,
	msg1075,
]);

var part1686 = match("MESSAGE#1060:00553/0", "nwparser.payload", "SCAN-MGR: Failed to get %{p0}");

var part1687 = match("MESSAGE#1060:00553/1_0", "nwparser.p0", "AltServer %{p0}");

var part1688 = match("MESSAGE#1060:00553/1_1", "nwparser.p0", "Version %{p0}");

var part1689 = match("MESSAGE#1060:00553/1_2", "nwparser.p0", "Path_GateLockCE %{p0}");

var select382 = linear_select([
	part1687,
	part1688,
	part1689,
]);

var all350 = all_match({
	processors: [
		part1686,
		select382,
		dup325,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1076 = msg("00553", all350);

var part1690 = match("MESSAGE#1061:00553:01", "nwparser.payload", "SCAN-MGR: Zero pattern size from server.ini.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1077 = msg("00553:01", part1690);

var part1691 = match("MESSAGE#1062:00553:02", "nwparser.payload", "SCAN-MGR: Pattern size from server.ini is too large: %{bytes->} (bytes).", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1078 = msg("00553:02", part1691);

var part1692 = match("MESSAGE#1063:00553:03", "nwparser.payload", "SCAN-MGR: Pattern URL from server.ini is too long: %{fld2}; max is %{fld3}.", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1079 = msg("00553:03", part1692);

var part1693 = match("MESSAGE#1064:00553:04/0", "nwparser.payload", "SCAN-MGR: Failed to retrieve %{p0}");

var select383 = linear_select([
	dup326,
	dup327,
]);

var part1694 = match("MESSAGE#1064:00553:04/2", "nwparser.p0", "file: %{fld2}; http status code: %{resultcode}.");

var all351 = all_match({
	processors: [
		part1693,
		select383,
		part1694,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1080 = msg("00553:04", all351);

var part1695 = match("MESSAGE#1065:00553:05", "nwparser.payload", "SCAN-MGR: Failed to write pattern into a RAM file.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1081 = msg("00553:05", part1695);

var part1696 = match("MESSAGE#1066:00553:06", "nwparser.payload", "SCAN-MGR: Check Pattern File failed: code from VSAPI: %{resultcode}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1082 = msg("00553:06", part1696);

var part1697 = match("MESSAGE#1067:00553:07", "nwparser.payload", "SCAN-MGR: Failed to write pattern into flash.%{}", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1083 = msg("00553:07", part1697);

var part1698 = match("MESSAGE#1068:00553:08/0", "nwparser.payload", "SCAN-MGR: Internal error while setting up for retrieving %{p0}");

var select384 = linear_select([
	dup327,
	dup326,
]);

var all352 = all_match({
	processors: [
		part1698,
		select384,
		dup328,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1084 = msg("00553:08", all352);

var part1699 = match("MESSAGE#1069:00553:09", "nwparser.payload", "SCAN-MGR: %{fld2->} %{disposition}: Err: %{resultcode}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1085 = msg("00553:09", part1699);

var part1700 = match("MESSAGE#1070:00553:10", "nwparser.payload", "SCAN-MGR: TMIntCPVSInit %{disposition->} due to %{result}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1086 = msg("00553:10", part1700);

var part1701 = match("MESSAGE#1071:00553:11", "nwparser.payload", "SCAN-MGR: Attempted Pattern Creation Date(%{fld2}) is after AV Key Expiration date(%{fld3}).", processor_chain([
	dup18,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1087 = msg("00553:11", part1701);

var part1702 = match("MESSAGE#1072:00553:12", "nwparser.payload", "SCAN-MGR: TMIntSetDecompressLayer %{disposition}: Layer: %{fld2}, Err: %{resultcode}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1088 = msg("00553:12", part1702);

var part1703 = match("MESSAGE#1073:00553:13", "nwparser.payload", "SCAN-MGR: TMIntSetExtractFileSizeLimit %{disposition}: Limit: %{fld2}, Err: %{resultcode}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1089 = msg("00553:13", part1703);

var part1704 = match("MESSAGE#1074:00553:14", "nwparser.payload", "SCAN-MGR: TMIntScanFile %{disposition}: ret: %{fld2}; cpapiErrCode: %{resultcode}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1090 = msg("00553:14", part1704);

var part1705 = match("MESSAGE#1075:00553:15", "nwparser.payload", "SCAN-MGR: VSAPI resource usage error. Left usage: %{fld2}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1091 = msg("00553:15", part1705);

var part1706 = match("MESSAGE#1076:00553:16", "nwparser.payload", "SCAN-MGR: Set decompress layer to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1092 = msg("00553:16", part1706);

var part1707 = match("MESSAGE#1077:00553:17", "nwparser.payload", "SCAN-MGR: Set maximum content size to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1093 = msg("00553:17", part1707);

var part1708 = match("MESSAGE#1078:00553:18", "nwparser.payload", "SCAN-MGR: Set maximum number of concurrent messages to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1094 = msg("00553:18", part1708);

var part1709 = match("MESSAGE#1079:00553:19", "nwparser.payload", "SCAN-MGR: Set drop if maximum number of concurrent messages exceeds max to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1095 = msg("00553:19", part1709);

var part1710 = match("MESSAGE#1080:00553:20", "nwparser.payload", "SCAN-MGR: Set Pattern URL to %{fld2}; update interval is %{fld3}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1096 = msg("00553:20", part1710);

var part1711 = match("MESSAGE#1081:00553:21", "nwparser.payload", "SCAN-MGR: Unset Pattern URL; Pattern will not be updated automatically.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1097 = msg("00553:21", part1711);

var part1712 = match("MESSAGE#1082:00553:22", "nwparser.payload", "SCAN-MGR: New pattern updated: version: %{version}, size: %{bytes->} (bytes).", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1098 = msg("00553:22", part1712);

var select385 = linear_select([
	msg1076,
	msg1077,
	msg1078,
	msg1079,
	msg1080,
	msg1081,
	msg1082,
	msg1083,
	msg1084,
	msg1085,
	msg1086,
	msg1087,
	msg1088,
	msg1089,
	msg1090,
	msg1091,
	msg1092,
	msg1093,
	msg1094,
	msg1095,
	msg1096,
	msg1097,
	msg1098,
]);

var part1713 = match("MESSAGE#1083:00554/0", "nwparser.payload", "SCAN-MGR: Cannot get %{p0}");

var part1714 = match("MESSAGE#1083:00554/1_0", "nwparser.p0", "AltServer info %{p0}");

var part1715 = match("MESSAGE#1083:00554/1_1", "nwparser.p0", "Version number %{p0}");

var part1716 = match("MESSAGE#1083:00554/1_2", "nwparser.p0", "Path_GateLockCE info %{p0}");

var select386 = linear_select([
	part1714,
	part1715,
	part1716,
]);

var all353 = all_match({
	processors: [
		part1713,
		select386,
		dup325,
	],
	on_success: processor_chain([
		dup144,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1099 = msg("00554", all353);

var part1717 = match("MESSAGE#1084:00554:01", "nwparser.payload", "SCAN-MGR: Per server.ini file, the AV pattern file size is zero.%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1100 = msg("00554:01", part1717);

var part1718 = match("MESSAGE#1085:00554:02", "nwparser.payload", "SCAN-MGR: AV pattern file size is too large (%{bytes->} bytes).", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1101 = msg("00554:02", part1718);

var part1719 = match("MESSAGE#1086:00554:03", "nwparser.payload", "SCAN-MGR: Alternate AV pattern file server URL is too long: %{bytes->} bytes. Max: %{fld2->} bytes.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1102 = msg("00554:03", part1719);

var part1720 = match("MESSAGE#1087:00554:04/0", "nwparser.payload", "SCAN-MGR: Cannot retrieve %{p0}");

var part1721 = match("MESSAGE#1087:00554:04/2", "nwparser.p0", "file from %{hostip}:%{network_port}. HTTP status code: %{fld2}.");

var all354 = all_match({
	processors: [
		part1720,
		dup405,
		part1721,
	],
	on_success: processor_chain([
		dup144,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1103 = msg("00554:04", all354);

var part1722 = match("MESSAGE#1088:00554:05/0", "nwparser.payload", "SCAN-MGR: Cannot write AV pattern file to %{p0}");

var part1723 = match("MESSAGE#1088:00554:05/1_0", "nwparser.p0", "RAM %{p0}");

var part1724 = match("MESSAGE#1088:00554:05/1_1", "nwparser.p0", "flash %{p0}");

var select387 = linear_select([
	part1723,
	part1724,
]);

var all355 = all_match({
	processors: [
		part1722,
		select387,
		dup116,
	],
	on_success: processor_chain([
		dup144,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1104 = msg("00554:05", all355);

var part1725 = match("MESSAGE#1089:00554:06", "nwparser.payload", "SCAN-MGR: Cannot check AV pattern file. VSAPI code: %{fld2}", processor_chain([
	dup144,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1105 = msg("00554:06", part1725);

var part1726 = match("MESSAGE#1090:00554:07/0", "nwparser.payload", "SCAN-MGR: Internal error occurred while retrieving %{p0}");

var all356 = all_match({
	processors: [
		part1726,
		dup405,
		dup328,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1106 = msg("00554:07", all356);

var part1727 = match("MESSAGE#1091:00554:08/0", "nwparser.payload", "SCAN-MGR: Internal error occurred when calling this function: %{fld2}. %{fld3->} %{p0}");

var part1728 = match("MESSAGE#1091:00554:08/1_0", "nwparser.p0", "Error: %{resultcode->} %{p0}");

var part1729 = match("MESSAGE#1091:00554:08/1_1", "nwparser.p0", "Returned a NULL VSC handler %{p0}");

var part1730 = match("MESSAGE#1091:00554:08/1_2", "nwparser.p0", "cpapiErrCode: %{resultcode->} %{p0}");

var select388 = linear_select([
	part1728,
	part1729,
	part1730,
]);

var all357 = all_match({
	processors: [
		part1727,
		select388,
		dup116,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1107 = msg("00554:08", all357);

var part1731 = match("MESSAGE#1092:00554:09", "nwparser.payload", "SCAN-MGR: Number of decompression layers has been set to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1108 = msg("00554:09", part1731);

var part1732 = match("MESSAGE#1093:00554:10", "nwparser.payload", "SCAN-MGR: Maximum content size has been set to %{fld2->} KB.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1109 = msg("00554:10", part1732);

var part1733 = match("MESSAGE#1094:00554:11", "nwparser.payload", "SCAN-MGR: Maximum number of concurrent messages has been set to %{fld2}.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1110 = msg("00554:11", part1733);

var part1734 = match("MESSAGE#1095:00554:12/0", "nwparser.payload", "SCAN-MGR: Fail mode has been set to %{p0}");

var part1735 = match("MESSAGE#1095:00554:12/1_0", "nwparser.p0", "drop %{p0}");

var part1736 = match("MESSAGE#1095:00554:12/1_1", "nwparser.p0", "pass %{p0}");

var select389 = linear_select([
	part1735,
	part1736,
]);

var part1737 = match("MESSAGE#1095:00554:12/2", "nwparser.p0", "unexamined traffic if %{p0}");

var part1738 = match("MESSAGE#1095:00554:12/3_0", "nwparser.p0", "content size %{p0}");

var part1739 = match("MESSAGE#1095:00554:12/3_1", "nwparser.p0", "number of concurrent messages %{p0}");

var select390 = linear_select([
	part1738,
	part1739,
]);

var part1740 = match("MESSAGE#1095:00554:12/4", "nwparser.p0", "exceeds max.%{}");

var all358 = all_match({
	processors: [
		part1734,
		select389,
		part1737,
		select390,
		part1740,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1111 = msg("00554:12", all358);

var part1741 = match("MESSAGE#1096:00554:13", "nwparser.payload", "SCAN-MGR: URL for AV pattern update server has been set to %{fld2}, and the update interval to %{fld3->} minutes.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1112 = msg("00554:13", part1741);

var part1742 = match("MESSAGE#1097:00554:14", "nwparser.payload", "SCAN-MGR: URL for AV pattern update server has been unset, and the update interval returned to its default.%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1113 = msg("00554:14", part1742);

var part1743 = match("MESSAGE#1098:00554:15", "nwparser.payload", "SCAN-MGR: New AV pattern file has been updated. Version: %{version}; size: %{bytes->} bytes.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1114 = msg("00554:15", part1743);

var part1744 = match("MESSAGE#1099:00554:16", "nwparser.payload", "SCAN-MGR: AV client has exceeded its resource allotment. Remaining available resources: %{fld2}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1115 = msg("00554:16", part1744);

var part1745 = match("MESSAGE#1100:00554:17", "nwparser.payload", "SCAN-MGR: Attempted to load AV pattern file created %{fld2->} after the AV subscription expired. (Exp: %{fld3})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1116 = msg("00554:17", part1745);

var select391 = linear_select([
	msg1099,
	msg1100,
	msg1101,
	msg1102,
	msg1103,
	msg1104,
	msg1105,
	msg1106,
	msg1107,
	msg1108,
	msg1109,
	msg1110,
	msg1111,
	msg1112,
	msg1113,
	msg1114,
	msg1115,
	msg1116,
]);

var part1746 = match("MESSAGE#1101:00555", "nwparser.payload", "Vrouter %{node->} PIMSM cannot process non-multicast address %{hostip}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1117 = msg("00555", part1746);

var part1747 = match("MESSAGE#1102:00556", "nwparser.payload", "UF-MGR: Failed to process a request. Reason: %{result}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1118 = msg("00556", part1747);

var part1748 = match("MESSAGE#1103:00556:01", "nwparser.payload", "UF-MGR: Failed to abort a transaction. Reason: %{result}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1119 = msg("00556:01", part1748);

var part1749 = match("MESSAGE#1104:00556:02/0", "nwparser.payload", "UF-MGR: UF %{p0}");

var part1750 = match("MESSAGE#1104:00556:02/1_0", "nwparser.p0", "K%{p0}");

var part1751 = match("MESSAGE#1104:00556:02/1_1", "nwparser.p0", "k%{p0}");

var select392 = linear_select([
	part1750,
	part1751,
]);

var part1752 = match("MESSAGE#1104:00556:02/2", "nwparser.p0", "ey %{p0}");

var part1753 = match("MESSAGE#1104:00556:02/3_0", "nwparser.p0", "Expired%{p0}");

var part1754 = match("MESSAGE#1104:00556:02/3_1", "nwparser.p0", "expired%{p0}");

var select393 = linear_select([
	part1753,
	part1754,
]);

var part1755 = match("MESSAGE#1104:00556:02/4", "nwparser.p0", "%{}(expiration date: %{fld2}; current date: %{fld3}).");

var all359 = all_match({
	processors: [
		part1749,
		select392,
		part1752,
		select393,
		part1755,
	],
	on_success: processor_chain([
		dup254,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1120 = msg("00556:02", all359);

var part1756 = match("MESSAGE#1105:00556:03/0", "nwparser.payload", "UF-MGR: Failed to %{p0}");

var part1757 = match("MESSAGE#1105:00556:03/1_0", "nwparser.p0", "enable %{p0}");

var part1758 = match("MESSAGE#1105:00556:03/1_1", "nwparser.p0", "disable %{p0}");

var select394 = linear_select([
	part1757,
	part1758,
]);

var part1759 = match("MESSAGE#1105:00556:03/2", "nwparser.p0", "cache.%{}");

var all360 = all_match({
	processors: [
		part1756,
		select394,
		part1759,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1121 = msg("00556:03", all360);

var part1760 = match("MESSAGE#1106:00556:04", "nwparser.payload", "UF-MGR: Internal Error: %{resultcode}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1122 = msg("00556:04", part1760);

var part1761 = match("MESSAGE#1107:00556:05", "nwparser.payload", "UF-MGR: Cache size changed to %{fld2}(K).", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1123 = msg("00556:05", part1761);

var part1762 = match("MESSAGE#1108:00556:06", "nwparser.payload", "UF-MGR: Cache timeout changes to %{fld2->} (hours).", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1124 = msg("00556:06", part1762);

var part1763 = match("MESSAGE#1109:00556:07", "nwparser.payload", "UF-MGR: Category update interval changed to %{fld2->} (weeks).", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1125 = msg("00556:07", part1763);

var part1764 = match("MESSAGE#1110:00556:08/0", "nwparser.payload", "UF-MGR: Cache %{p0}");

var all361 = all_match({
	processors: [
		part1764,
		dup358,
		dup116,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1126 = msg("00556:08", all361);

var part1765 = match("MESSAGE#1111:00556:09", "nwparser.payload", "UF-MGR: URL BLOCKED: ip_addr (%{fld2}) -> ip_addr (%{fld3}), %{fld4->} action: %{disposition}, category: %{fld5}, reason %{result}", processor_chain([
	dup232,
	dup2,
	dup3,
	dup4,
	dup5,
	dup282,
]));

var msg1127 = msg("00556:09", part1765);

var part1766 = match("MESSAGE#1112:00556:10", "nwparser.payload", "UF-MGR: URL FILTER ERR: ip_addr (%{fld2}) -> ip_addr (%{fld3}), host: %{fld5->} page: %{fld4->} code: %{resultcode->} reason: %{result}.", processor_chain([
	dup232,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1128 = msg("00556:10", part1766);

var part1767 = match("MESSAGE#1113:00556:11", "nwparser.payload", "UF-MGR: Primary CPA server changed to %{fld2}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1129 = msg("00556:11", part1767);

var part1768 = match("MESSAGE#1114:00556:12/0", "nwparser.payload", "UF-MGR: %{fld2->} CPA server %{p0}");

var select395 = linear_select([
	dup140,
	dup169,
]);

var part1769 = match("MESSAGE#1114:00556:12/2", "nwparser.p0", "changed to %{fld3}.");

var all362 = all_match({
	processors: [
		part1768,
		select395,
		part1769,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1130 = msg("00556:12", all362);

var part1770 = match("MESSAGE#1115:00556:13", "nwparser.payload", "UF-MGR: SurfControl URL filtering %{disposition}.", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1131 = msg("00556:13", part1770);

var part1771 = match("MESSAGE#1116:00556:14/0", "nwparser.payload", "UF-MGR: The url %{url->} was %{p0}");

var part1772 = match("MESSAGE#1116:00556:14/2", "nwparser.p0", "category %{fld2}.");

var all363 = all_match({
	processors: [
		part1771,
		dup406,
		part1772,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1132 = msg("00556:14", all363);

var part1773 = match("MESSAGE#1117:00556:15/0", "nwparser.payload", "UF-MGR: The category %{fld2->} was %{p0}");

var part1774 = match("MESSAGE#1117:00556:15/2", "nwparser.p0", "profile %{fld3->} with action %{disposition}.");

var all364 = all_match({
	processors: [
		part1773,
		dup406,
		part1774,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
		dup282,
	]),
});

var msg1133 = msg("00556:15", all364);

var part1775 = match("MESSAGE#1118:00556:16/0", "nwparser.payload", "UF-MGR: The %{p0}");

var part1776 = match("MESSAGE#1118:00556:16/1_0", "nwparser.p0", "profile %{p0}");

var part1777 = match("MESSAGE#1118:00556:16/1_1", "nwparser.p0", "category %{p0}");

var select396 = linear_select([
	part1776,
	part1777,
]);

var part1778 = match("MESSAGE#1118:00556:16/2", "nwparser.p0", "%{fld2->} was %{p0}");

var select397 = linear_select([
	dup104,
	dup120,
]);

var all365 = all_match({
	processors: [
		part1775,
		select396,
		part1778,
		select397,
		dup116,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1134 = msg("00556:16", all365);

var part1779 = match("MESSAGE#1119:00556:17/0", "nwparser.payload", "UF-MGR: The category %{fld2->} was set in profile %{profile->} as the %{p0}");

var part1780 = match("MESSAGE#1119:00556:17/1_0", "nwparser.p0", "black %{p0}");

var part1781 = match("MESSAGE#1119:00556:17/1_1", "nwparser.p0", "white %{p0}");

var select398 = linear_select([
	part1780,
	part1781,
]);

var part1782 = match("MESSAGE#1119:00556:17/2", "nwparser.p0", "list.%{}");

var all366 = all_match({
	processors: [
		part1779,
		select398,
		part1782,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1135 = msg("00556:17", all366);

var part1783 = match("MESSAGE#1120:00556:18/0", "nwparser.payload", "UF-MGR: The action for %{fld2->} in profile %{profile->} was %{p0}");

var part1784 = match("MESSAGE#1120:00556:18/1_1", "nwparser.p0", "changed %{p0}");

var select399 = linear_select([
	dup101,
	part1784,
]);

var part1785 = match("MESSAGE#1120:00556:18/2", "nwparser.p0", "to %{fld3}.");

var all367 = all_match({
	processors: [
		part1783,
		select399,
		part1785,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1136 = msg("00556:18", all367);

var part1786 = match("MESSAGE#1121:00556:20/0", "nwparser.payload", "UF-MGR: The category list from the CPA server %{p0}");

var part1787 = match("MESSAGE#1121:00556:20/2", "nwparser.p0", "updated on%{p0}");

var select400 = linear_select([
	dup103,
	dup96,
]);

var part1788 = match("MESSAGE#1121:00556:20/4", "nwparser.p0", "the device.%{}");

var all368 = all_match({
	processors: [
		part1786,
		dup355,
		part1787,
		select400,
		part1788,
	],
	on_success: processor_chain([
		dup19,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1137 = msg("00556:20", all368);

var part1789 = match("MESSAGE#1122:00556:21", "nwparser.payload", "UF-MGR: URL BLOCKED: %{saddr}(%{sport})->%{daddr}(%{dport}), %{fld2->} action: %{disposition}, category: %{category}, reason: %{result->} (%{fld1})", processor_chain([
	dup232,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
	dup282,
]));

var msg1138 = msg("00556:21", part1789);

var part1790 = match("MESSAGE#1123:00556:22", "nwparser.payload", "UF-MGR: URL BLOCKED: %{saddr}(%{sport})->%{daddr}(%{dport}), %{fld2->} (%{fld1})", processor_chain([
	dup232,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1139 = msg("00556:22", part1790);

var select401 = linear_select([
	msg1118,
	msg1119,
	msg1120,
	msg1121,
	msg1122,
	msg1123,
	msg1124,
	msg1125,
	msg1126,
	msg1127,
	msg1128,
	msg1129,
	msg1130,
	msg1131,
	msg1132,
	msg1133,
	msg1134,
	msg1135,
	msg1136,
	msg1137,
	msg1138,
	msg1139,
]);

var part1791 = match("MESSAGE#1124:00572", "nwparser.payload", "PPP LCP on interface %{interface->} is %{fld2}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1140 = msg("00572", part1791);

var part1792 = match("MESSAGE#1125:00572:01", "nwparser.payload", "PPP authentication state on interface %{interface}: %{result}. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1141 = msg("00572:01", part1792);

var part1793 = match("MESSAGE#1126:00572:03", "nwparser.payload", "PPP on interface %{interface->} is %{disposition->} by receiving Terminate-Request. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1142 = msg("00572:03", part1793);

var select402 = linear_select([
	msg1140,
	msg1141,
	msg1142,
]);

var part1794 = match("MESSAGE#1127:00615", "nwparser.payload", "PBR policy \"%{policyname}\" rebuilding lookup tree for virtual router \"%{node}\". (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg1143 = msg("00615", part1794);

var part1795 = match("MESSAGE#1128:00615:01", "nwparser.payload", "PBR policy \"%{policyname}\" lookup tree rebuilt successfully in virtual router \"%{node}\". (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg1144 = msg("00615:01", part1795);

var select403 = linear_select([
	msg1143,
	msg1144,
]);

var part1796 = match("MESSAGE#1129:00601", "nwparser.payload", "%{signame->} attack! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol}, through policy %{policyname}. Occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup9,
	dup4,
	dup5,
	dup61,
]));

var msg1145 = msg("00601", part1796);

var part1797 = match("MESSAGE#1130:00601:01", "nwparser.payload", "%{signame->} has been detected from %{saddr}/%{sport->} to %{daddr}/%{dport->} through policy %{policyname->} %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup59,
	dup3,
	dup9,
	dup4,
	dup5,
	dup61,
]));

var msg1146 = msg("00601:01", part1797);

var part1798 = match("MESSAGE#1131:00601:18", "nwparser.payload", "Error in initializing multicast.%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1147 = msg("00601:18", part1798);

var select404 = linear_select([
	msg1145,
	msg1146,
	msg1147,
]);

var part1799 = match("MESSAGE#1132:00602", "nwparser.payload", "PIMSM Error in initializing interface state change%{}", processor_chain([
	dup19,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1148 = msg("00602", part1799);

var part1800 = match("MESSAGE#1133:00612/0", "nwparser.payload", "Switch event: the status of ethernet port %{fld2->} changed to link %{p0}");

var part1801 = match("MESSAGE#1133:00612/2", "nwparser.p0", ", duplex %{p0}");

var part1802 = match("MESSAGE#1133:00612/3_0", "nwparser.p0", "full %{p0}");

var part1803 = match("MESSAGE#1133:00612/3_1", "nwparser.p0", "half %{p0}");

var select405 = linear_select([
	part1802,
	part1803,
]);

var part1804 = match("MESSAGE#1133:00612/4", "nwparser.p0", ", speed 10%{p0}");

var part1805 = match("MESSAGE#1133:00612/5_0", "nwparser.p0", "0 %{p0}");

var select406 = linear_select([
	part1805,
	dup96,
]);

var part1806 = match("MESSAGE#1133:00612/6", "nwparser.p0", "M. (%{fld1})");

var all369 = all_match({
	processors: [
		part1800,
		dup353,
		part1801,
		select405,
		part1804,
		select406,
		part1806,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1149 = msg("00612", all369);

var part1807 = match("MESSAGE#1134:00620", "nwparser.payload", "RTSYNC: Event posted to send all the DRP routes to backup device. (%{fld1})", processor_chain([
	dup272,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1150 = msg("00620", part1807);

var part1808 = match("MESSAGE#1135:00620:01/0", "nwparser.payload", "RTSYNC: %{p0}");

var part1809 = match("MESSAGE#1135:00620:01/1_0", "nwparser.p0", "Serviced%{p0}");

var part1810 = match("MESSAGE#1135:00620:01/1_1", "nwparser.p0", "Recieved%{p0}");

var select407 = linear_select([
	part1809,
	part1810,
]);

var part1811 = match("MESSAGE#1135:00620:01/2", "nwparser.p0", "%{}coldstart request for route synchronization from NSRP peer. (%{fld1})");

var all370 = all_match({
	processors: [
		part1808,
		select407,
		part1811,
	],
	on_success: processor_chain([
		dup272,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1151 = msg("00620:01", all370);

var part1812 = match("MESSAGE#1136:00620:02", "nwparser.payload", "RTSYNC: Started timer to purge all the DRP backup routes - %{fld2->} (%{fld1})", processor_chain([
	dup272,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1152 = msg("00620:02", part1812);

var part1813 = match("MESSAGE#1137:00620:03", "nwparser.payload", "RTSYNC: Event posted to purge backup routes in all vrouters. (%{fld1})", processor_chain([
	dup272,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1153 = msg("00620:03", part1813);

var part1814 = match("MESSAGE#1138:00620:04", "nwparser.payload", "RTSYNC: Timer to purge the DRP backup routes is stopped. (%{fld1})", processor_chain([
	dup272,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1154 = msg("00620:04", part1814);

var select408 = linear_select([
	msg1150,
	msg1151,
	msg1152,
	msg1153,
	msg1154,
]);

var part1815 = match("MESSAGE#1139:00622", "nwparser.payload", "NHRP : NHRP instance in virtual router %{node->} is created. (%{fld1})", processor_chain([
	dup273,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1155 = msg("00622", part1815);

var part1816 = match("MESSAGE#1140:00625/0", "nwparser.payload", "Session (id %{sessionid->} src-ip %{saddr->} dst-ip %{daddr->} dst port %{dport}) route is %{p0}");

var part1817 = match("MESSAGE#1140:00625/1_0", "nwparser.p0", "invalid%{p0}");

var part1818 = match("MESSAGE#1140:00625/1_1", "nwparser.p0", "valid%{p0}");

var select409 = linear_select([
	part1817,
	part1818,
]);

var all371 = all_match({
	processors: [
		part1816,
		select409,
		dup49,
	],
	on_success: processor_chain([
		dup273,
		dup2,
		dup4,
		dup5,
		dup9,
	]),
});

var msg1156 = msg("00625", all371);

var part1819 = match("MESSAGE#1141:00628/0", "nwparser.payload", "audit log queue %{p0}");

var part1820 = match("MESSAGE#1141:00628/1_0", "nwparser.p0", "Traffic Log %{p0}");

var part1821 = match("MESSAGE#1141:00628/1_1", "nwparser.p0", "Event Alarm Log %{p0}");

var part1822 = match("MESSAGE#1141:00628/1_2", "nwparser.p0", "Event Log %{p0}");

var select410 = linear_select([
	part1820,
	part1821,
	part1822,
]);

var part1823 = match("MESSAGE#1141:00628/2", "nwparser.p0", "is overwritten (%{fld1})");

var all372 = all_match({
	processors: [
		part1819,
		select410,
		part1823,
	],
	on_success: processor_chain([
		dup223,
		dup2,
		dup4,
		dup5,
		dup9,
	]),
});

var msg1157 = msg("00628", all372);

var part1824 = match("MESSAGE#1142:00767:50", "nwparser.payload", "Log setting was modified to %{disposition->} %{fld2->} level by admin %{administrator->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup9,
	dup282,
]));

var msg1158 = msg("00767:50", part1824);

var part1825 = match("MESSAGE#1143:00767:51", "nwparser.payload", "Attack CS:Man in Middle is created by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} by admin %{administrator->} (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg1159 = msg("00767:51", part1825);

var part1826 = match("MESSAGE#1144:00767:52", "nwparser.payload", "Attack group %{group->} is created by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} by admin %{administrator->} (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg1160 = msg("00767:52", part1826);

var part1827 = match("MESSAGE#1145:00767:53", "nwparser.payload", "Attack CS:Man in Middle is added to attack group %{group->} by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} by admin %{administrator->} (%{fld1})", processor_chain([
	dup58,
	dup2,
	dup4,
	dup5,
	dup9,
]));

var msg1161 = msg("00767:53", part1827);

var part1828 = match("MESSAGE#1146:00767", "nwparser.payload", "Cannot contact the SecurID server%{}", processor_chain([
	dup27,
	setc("ec_theme","Communication"),
	dup39,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1162 = msg("00767", part1828);

var part1829 = match("MESSAGE#1147:00767:01/0", "nwparser.payload", "System auto-config of file %{fld2->} from TFTP server %{hostip->} has %{p0}");

var part1830 = match("MESSAGE#1147:00767:01/1_0", "nwparser.p0", "been loaded successfully%{}");

var part1831 = match("MESSAGE#1147:00767:01/1_1", "nwparser.p0", "failed%{}");

var select411 = linear_select([
	part1830,
	part1831,
]);

var all373 = all_match({
	processors: [
		part1829,
		select411,
	],
	on_success: processor_chain([
		dup44,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1163 = msg("00767:01", all373);

var part1832 = match("MESSAGE#1148:00767:02", "nwparser.payload", "netscreen: System Config saved from host %{saddr}", processor_chain([
	setc("eventcategory","1702000000"),
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1164 = msg("00767:02", part1832);

var part1833 = match("MESSAGE#1149:00767:03", "nwparser.payload", "System Config saved to filename %{filename}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1165 = msg("00767:03", part1833);

var part1834 = match("MESSAGE#1150:00767:04", "nwparser.payload", "System is operational.%{}", processor_chain([
	dup44,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1166 = msg("00767:04", part1834);

var part1835 = match("MESSAGE#1151:00767:05", "nwparser.payload", "The device cannot contact the SecurID server%{}", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1167 = msg("00767:05", part1835);

var part1836 = match("MESSAGE#1152:00767:06", "nwparser.payload", "The device cannot send data to the SecurID server%{}", processor_chain([
	dup27,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1168 = msg("00767:06", part1836);

var part1837 = match("MESSAGE#1153:00767:07", "nwparser.payload", "The system configuration was saved from peer unit by admin%{}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1169 = msg("00767:07", part1837);

var part1838 = match("MESSAGE#1154:00767:08/0", "nwparser.payload", "The system configuration was saved by admin %{p0}");

var all374 = all_match({
	processors: [
		part1838,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1170 = msg("00767:08", all374);

var part1839 = match("MESSAGE#1155:00767:09/0", "nwparser.payload", "traffic shaping is turned O%{p0}");

var part1840 = match("MESSAGE#1155:00767:09/1_0", "nwparser.p0", "N%{}");

var part1841 = match("MESSAGE#1155:00767:09/1_1", "nwparser.p0", "FF%{}");

var select412 = linear_select([
	part1840,
	part1841,
]);

var all375 = all_match({
	processors: [
		part1839,
		select412,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1171 = msg("00767:09", all375);

var part1842 = match("MESSAGE#1156:00767:10/0", "nwparser.payload", "The system configuration was saved from host %{saddr->} by admin %{p0}");

var all376 = all_match({
	processors: [
		part1842,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1172 = msg("00767:10", all376);

var part1843 = match("MESSAGE#1157:00767:11/0", "nwparser.payload", "Fatal error. The NetScreen device was unable to upgrade the %{p0}");

var part1844 = match("MESSAGE#1157:00767:11/1_1", "nwparser.p0", "file system %{p0}");

var select413 = linear_select([
	dup331,
	part1844,
]);

var part1845 = match("MESSAGE#1157:00767:11/2", "nwparser.p0", ", and the %{p0}");

var part1846 = match("MESSAGE#1157:00767:11/3_1", "nwparser.p0", "old file system %{p0}");

var select414 = linear_select([
	dup331,
	part1846,
]);

var part1847 = match("MESSAGE#1157:00767:11/4", "nwparser.p0", "is damaged.%{}");

var all377 = all_match({
	processors: [
		part1843,
		select413,
		part1845,
		select414,
		part1847,
	],
	on_success: processor_chain([
		dup18,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1173 = msg("00767:11", all377);

var part1848 = match("MESSAGE#1158:00767:12", "nwparser.payload", "System configuration saved by %{username->} via %{logon_type->} from host %{saddr->} to %{daddr}:%{dport->} by %{fld2->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1174 = msg("00767:12", part1848);

var part1849 = match("MESSAGE#1159:00767:13/0", "nwparser.payload", "%{fld2}Environment variable %{fld3->} is changed to %{fld4->} by admin %{p0}");

var all378 = all_match({
	processors: [
		part1849,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1175 = msg("00767:13", all378);

var part1850 = match("MESSAGE#1160:00767:14/0", "nwparser.payload", "System was %{p0}");

var part1851 = match("MESSAGE#1160:00767:14/1_0", "nwparser.p0", "reset %{p0}");

var select415 = linear_select([
	part1851,
	dup262,
]);

var part1852 = match("MESSAGE#1160:00767:14/2", "nwparser.p0", "at %{fld2->} by %{p0}");

var part1853 = match("MESSAGE#1160:00767:14/3_0", "nwparser.p0", "admin %{administrator}");

var part1854 = match_copy("MESSAGE#1160:00767:14/3_1", "nwparser.p0", "username");

var select416 = linear_select([
	part1853,
	part1854,
]);

var all379 = all_match({
	processors: [
		part1850,
		select415,
		part1852,
		select416,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1176 = msg("00767:14", all379);

var part1855 = match("MESSAGE#1161:00767:15/1_0", "nwparser.p0", "System %{p0}");

var part1856 = match("MESSAGE#1161:00767:15/1_1", "nwparser.p0", "Event %{p0}");

var part1857 = match("MESSAGE#1161:00767:15/1_2", "nwparser.p0", "Traffic %{p0}");

var select417 = linear_select([
	part1855,
	part1856,
	part1857,
]);

var part1858 = match("MESSAGE#1161:00767:15/2", "nwparser.p0", "log was reviewed by %{p0}");

var part1859 = match("MESSAGE#1161:00767:15/4", "nwparser.p0", "%{} %{username}.");

var all380 = all_match({
	processors: [
		dup183,
		select417,
		part1858,
		dup336,
		part1859,
	],
	on_success: processor_chain([
		dup223,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1177 = msg("00767:15", all380);

var part1860 = match("MESSAGE#1162:00767:16", "nwparser.payload", "%{fld2->} Admin %{administrator->} issued command %{info->} to redirect output.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1178 = msg("00767:16", part1860);

var part1861 = match("MESSAGE#1163:00767:17/0", "nwparser.payload", "%{fld2->} Save new software from %{fld3->} to flash by admin %{p0}");

var all381 = all_match({
	processors: [
		part1861,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1179 = msg("00767:17", all381);

var part1862 = match("MESSAGE#1164:00767:18", "nwparser.payload", "Attack database version %{version->} has been %{fld2->} saved to flash.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1180 = msg("00767:18", part1862);

var part1863 = match("MESSAGE#1165:00767:19", "nwparser.payload", "Attack database version %{version->} was rejected because the authentication check failed.", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1181 = msg("00767:19", part1863);

var part1864 = match("MESSAGE#1166:00767:20", "nwparser.payload", "The dictionary file version of the RADIUS server %{hostname->} does not match %{fld2}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1182 = msg("00767:20", part1864);

var part1865 = match("MESSAGE#1167:00767:21", "nwparser.payload", "Session (%{fld2->} %{fld3}, %{fld4}) cleared %{fld5}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1183 = msg("00767:21", part1865);

var part1866 = match("MESSAGE#1168:00767:22/0", "nwparser.payload", "The system configuration was not saved %{p0}");

var part1867 = match("MESSAGE#1168:00767:22/1_0", "nwparser.p0", "%{fld2->} by admin %{administrator->} via NSRP Peer %{p0}");

var part1868 = match("MESSAGE#1168:00767:22/1_1", "nwparser.p0", "%{fld2->} %{p0}");

var select418 = linear_select([
	part1867,
	part1868,
]);

var part1869 = match("MESSAGE#1168:00767:22/2", "nwparser.p0", "by administrator %{fld3}. %{p0}");

var part1870 = match("MESSAGE#1168:00767:22/3_0", "nwparser.p0", "It was locked %{p0}");

var part1871 = match("MESSAGE#1168:00767:22/3_1", "nwparser.p0", "Locked %{p0}");

var select419 = linear_select([
	part1870,
	part1871,
]);

var part1872 = match("MESSAGE#1168:00767:22/4", "nwparser.p0", "by administrator %{fld4->} %{p0}");

var all382 = all_match({
	processors: [
		part1866,
		select418,
		part1869,
		select419,
		part1872,
		dup354,
	],
	on_success: processor_chain([
		dup50,
		dup43,
		dup51,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1184 = msg("00767:22", all382);

var part1873 = match("MESSAGE#1169:00767:23", "nwparser.payload", "Save new software from slot filename %{filename->} to flash memory by administrator %{administrator}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var msg1185 = msg("00767:23", part1873);

var part1874 = match("MESSAGE#1170:00767:25/0", "nwparser.payload", "System configuration saved by %{username->} via %{logon_type->} from %{p0}");

var select420 = linear_select([
	dup169,
	dup16,
]);

var part1875 = match("MESSAGE#1170:00767:25/3_0", "nwparser.p0", "%{saddr}:%{sport->} by %{p0}");

var part1876 = match("MESSAGE#1170:00767:25/3_1", "nwparser.p0", "%{saddr->} by %{p0}");

var select421 = linear_select([
	part1875,
	part1876,
]);

var all383 = all_match({
	processors: [
		part1874,
		select420,
		dup23,
		select421,
		dup108,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var msg1186 = msg("00767:25", all383);

var part1877 = match("MESSAGE#1171:00767:26/0", "nwparser.payload", "Lock configuration %{p0}");

var part1878 = match("MESSAGE#1171:00767:26/1_0", "nwparser.p0", "started%{p0}");

var part1879 = match("MESSAGE#1171:00767:26/1_1", "nwparser.p0", "ended%{p0}");

var select422 = linear_select([
	part1878,
	part1879,
]);

var part1880 = match("MESSAGE#1171:00767:26/2", "nwparser.p0", "%{}by task %{p0}");

var part1881 = match("MESSAGE#1171:00767:26/3_0", "nwparser.p0", "%{fld3}, with a timeout value of %{fld2}");

var part1882 = match("MESSAGE#1171:00767:26/3_1", "nwparser.p0", "%{fld2->} (%{fld1})");

var select423 = linear_select([
	part1881,
	part1882,
]);

var all384 = all_match({
	processors: [
		part1877,
		select422,
		part1880,
		select423,
	],
	on_success: processor_chain([
		dup50,
		dup43,
		dup51,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1187 = msg("00767:26", all384);

var part1883 = match("MESSAGE#1172:00767:27/0", "nwparser.payload", "Environment variable %{fld2->} changed to %{p0}");

var part1884 = match("MESSAGE#1172:00767:27/1_0", "nwparser.p0", "%{fld3->} by %{username->} (%{fld1})");

var part1885 = match_copy("MESSAGE#1172:00767:27/1_1", "nwparser.p0", "fld3");

var select424 = linear_select([
	part1884,
	part1885,
]);

var all385 = all_match({
	processors: [
		part1883,
		select424,
	],
	on_success: processor_chain([
		dup223,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1188 = msg("00767:27", all385);

var part1886 = match("MESSAGE#1173:00767:28", "nwparser.payload", "The system configuration was loaded from IP address %{hostip->} under filename %{filename->} by administrator by admin %{administrator->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1189 = msg("00767:28", part1886);

var part1887 = match("MESSAGE#1174:00767:29", "nwparser.payload", "Save configuration to IP address %{hostip->} under filename %{filename->} by administrator by admin %{administrator->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1190 = msg("00767:29", part1887);

var part1888 = match("MESSAGE#1175:00767:30", "nwparser.payload", "%{fld2}: The system configuration was saved from host %{saddr->} by admin %{administrator->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1191 = msg("00767:30", part1888);

var part1889 = match("MESSAGE#1176:00767:31/1_0", "nwparser.p0", "logged events or alarms %{p0}");

var part1890 = match("MESSAGE#1176:00767:31/1_1", "nwparser.p0", "traffic logs %{p0}");

var select425 = linear_select([
	part1889,
	part1890,
]);

var part1891 = match("MESSAGE#1176:00767:31/2", "nwparser.p0", "were cleared by admin %{p0}");

var all386 = all_match({
	processors: [
		dup186,
		select425,
		part1891,
		dup397,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1192 = msg("00767:31", all386);

var part1892 = match("MESSAGE#1177:00767:32/0", "nwparser.payload", "SIP parser error %{p0}");

var part1893 = match("MESSAGE#1177:00767:32/1_0", "nwparser.p0", "SIP-field%{p0}");

var part1894 = match("MESSAGE#1177:00767:32/1_1", "nwparser.p0", "Message%{p0}");

var select426 = linear_select([
	part1893,
	part1894,
]);

var part1895 = match("MESSAGE#1177:00767:32/2", "nwparser.p0", ": %{result}(%{fld1})");

var all387 = all_match({
	processors: [
		part1892,
		select426,
		part1895,
	],
	on_success: processor_chain([
		dup27,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1193 = msg("00767:32", all387);

var part1896 = match("MESSAGE#1178:00767:33", "nwparser.payload", "Daylight Saving Time has started. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1194 = msg("00767:33", part1896);

var part1897 = match("MESSAGE#1179:00767:34", "nwparser.payload", "NetScreen devices do not support multiple IP addresses %{hostip->} or ports %{network_port->} in SIP headers RESPONSE (%{fld1})", processor_chain([
	dup313,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1195 = msg("00767:34", part1897);

var part1898 = match("MESSAGE#1180:00767:35", "nwparser.payload", "Environment variable %{fld2->} set to %{fld3->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1196 = msg("00767:35", part1898);

var part1899 = match("MESSAGE#1181:00767:36", "nwparser.payload", "System configuration saved from %{fld2->} by %{username->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1197 = msg("00767:36", part1899);

var part1900 = match("MESSAGE#1182:00767:37", "nwparser.payload", "Trial keys are available to download to enable advanced features. %{space->} To find out, please visit %{url->} (%{fld1})", processor_chain([
	dup254,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1198 = msg("00767:37", part1900);

var part1901 = match("MESSAGE#1183:00767:38", "nwparser.payload", "Log buffer was full and remaining messages were sent to external destination. %{fld2->} packets were dropped. (%{fld1})", processor_chain([
	setc("eventcategory","1602000000"),
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1199 = msg("00767:38", part1901);

var part1902 = match("MESSAGE#1184:00767:39/0", "nwparser.payload", "Cannot %{p0}");

var part1903 = match("MESSAGE#1184:00767:39/1_0", "nwparser.p0", "download %{p0}");

var part1904 = match("MESSAGE#1184:00767:39/1_1", "nwparser.p0", "parse %{p0}");

var select427 = linear_select([
	part1903,
	part1904,
]);

var part1905 = match("MESSAGE#1184:00767:39/2", "nwparser.p0", "attack database %{p0}");

var part1906 = match("MESSAGE#1184:00767:39/3_0", "nwparser.p0", "from %{url->} (%{result}). %{p0}");

var part1907 = match("MESSAGE#1184:00767:39/3_1", "nwparser.p0", "%{fld2->} %{p0}");

var select428 = linear_select([
	part1906,
	part1907,
]);

var all388 = all_match({
	processors: [
		part1902,
		select427,
		part1905,
		select428,
		dup10,
	],
	on_success: processor_chain([
		dup324,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1200 = msg("00767:39", all388);

var part1908 = match("MESSAGE#1185:00767:40", "nwparser.payload", "Deep Inspection update key is %{disposition}. (%{fld1})", processor_chain([
	dup62,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1201 = msg("00767:40", part1908);

var part1909 = match("MESSAGE#1186:00767:42", "nwparser.payload", "System configuration saved by %{username->} via %{logon_type->} to %{daddr}:%{dport->} by %{fld2->} (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1202 = msg("00767:42", part1909);

var part1910 = match("MESSAGE#1187:00767:43", "nwparser.payload", "Daylight Saving Time ended. (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1203 = msg("00767:43", part1910);

var part1911 = match("MESSAGE#1188:00767:44", "nwparser.payload", "New GMT zone ahead or behind by %{fld2->} (%{fld1})", processor_chain([
	dup44,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1204 = msg("00767:44", part1911);

var part1912 = match("MESSAGE#1189:00767:45", "nwparser.payload", "Attack database version %{version->} is saved to flash. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1205 = msg("00767:45", part1912);

var part1913 = match("MESSAGE#1190:00767:46", "nwparser.payload", "System configuration saved by netscreen via %{logon_type->} by netscreen. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1206 = msg("00767:46", part1913);

var part1914 = match("MESSAGE#1191:00767:47", "nwparser.payload", "User %{username->} belongs to a different group in the RADIUS server than that allowed in the device. (%{fld1})", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup9,
]));

var msg1207 = msg("00767:47", part1914);

var part1915 = match("MESSAGE#1192:00767:24/0", "nwparser.payload", "System configuration saved by %{p0}");

var part1916 = match("MESSAGE#1192:00767:24/2", "nwparser.p0", "%{logon_type->} by %{fld2->} (%{fld1})");

var all389 = all_match({
	processors: [
		part1915,
		dup364,
		part1916,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup9,
		dup4,
		dup5,
	]),
});

var msg1208 = msg("00767:24", all389);

var part1917 = match("MESSAGE#1193:00767:48", "nwparser.payload", "HA: Synchronization file(s) hidden file end with c sent to backup device in cluster. (%{fld1})", processor_chain([
	dup272,
	dup2,
	dup3,
	dup9,
	dup4,
	dup5,
]));

var msg1209 = msg("00767:48", part1917);

var part1918 = match("MESSAGE#1194:00767:49/0", "nwparser.payload", "%{fld2->} turn o%{p0}");

var part1919 = match("MESSAGE#1194:00767:49/1_0", "nwparser.p0", "n%{p0}");

var part1920 = match("MESSAGE#1194:00767:49/1_1", "nwparser.p0", "ff%{p0}");

var select429 = linear_select([
	part1919,
	part1920,
]);

var part1921 = match("MESSAGE#1194:00767:49/2", "nwparser.p0", "%{}debug switch for %{fld3->} (%{fld1})");

var all390 = all_match({
	processors: [
		part1918,
		select429,
		part1921,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup4,
		dup5,
		dup9,
	]),
});

var msg1210 = msg("00767:49", all390);

var select430 = linear_select([
	msg1158,
	msg1159,
	msg1160,
	msg1161,
	msg1162,
	msg1163,
	msg1164,
	msg1165,
	msg1166,
	msg1167,
	msg1168,
	msg1169,
	msg1170,
	msg1171,
	msg1172,
	msg1173,
	msg1174,
	msg1175,
	msg1176,
	msg1177,
	msg1178,
	msg1179,
	msg1180,
	msg1181,
	msg1182,
	msg1183,
	msg1184,
	msg1185,
	msg1186,
	msg1187,
	msg1188,
	msg1189,
	msg1190,
	msg1191,
	msg1192,
	msg1193,
	msg1194,
	msg1195,
	msg1196,
	msg1197,
	msg1198,
	msg1199,
	msg1200,
	msg1201,
	msg1202,
	msg1203,
	msg1204,
	msg1205,
	msg1206,
	msg1207,
	msg1208,
	msg1209,
	msg1210,
]);

var part1922 = match("MESSAGE#1195:01269", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup277,
	dup3,
	dup275,
	dup60,
]));

var msg1211 = msg("01269", part1922);

var msg1212 = msg("01269:01", dup407);

var msg1213 = msg("01269:02", dup408);

var msg1214 = msg("01269:03", dup409);

var select431 = linear_select([
	msg1211,
	msg1212,
	msg1213,
	msg1214,
]);

var part1923 = match("MESSAGE#1199:17852", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup276,
	dup277,
	dup275,
	dup332,
]));

var msg1215 = msg("17852", part1923);

var part1924 = match("MESSAGE#1200:17852:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup332,
	dup282,
]));

var msg1216 = msg("17852:01", part1924);

var part1925 = match("MESSAGE#1201:17852:02", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup277,
	dup61,
]));

var msg1217 = msg("17852:02", part1925);

var part1926 = match("MESSAGE#1202:17852:03", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup332,
	dup282,
]));

var msg1218 = msg("17852:03", part1926);

var select432 = linear_select([
	msg1215,
	msg1216,
	msg1217,
	msg1218,
]);

var msg1219 = msg("23184", dup410);

var part1927 = match("MESSAGE#1204:23184:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol->} direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup61,
	dup282,
]));

var msg1220 = msg("23184:01", part1927);

var part1928 = match("MESSAGE#1205:23184:02", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup276,
	dup277,
	dup275,
	dup61,
]));

var msg1221 = msg("23184:02", part1928);

var part1929 = match("MESSAGE#1206:23184:03", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup332,
	dup282,
]));

var msg1222 = msg("23184:03", part1929);

var select433 = linear_select([
	msg1219,
	msg1220,
	msg1221,
	msg1222,
]);

var msg1223 = msg("27052", dup410);

var part1930 = match("MESSAGE#1208:27052:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol}direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup61,
	dup282,
]));

var msg1224 = msg("27052:01", part1930);

var select434 = linear_select([
	msg1223,
	msg1224,
]);

var part1931 = match("MESSAGE#1209:39568", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup277,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup60,
]));

var msg1225 = msg("39568", part1931);

var msg1226 = msg("39568:01", dup407);

var msg1227 = msg("39568:02", dup408);

var msg1228 = msg("39568:03", dup409);

var select435 = linear_select([
	msg1225,
	msg1226,
	msg1227,
	msg1228,
]);

var chain1 = processor_chain([
	select2,
	msgid_select({
		"00001": select6,
		"00002": select29,
		"00003": select31,
		"00004": select33,
		"00005": select39,
		"00006": select40,
		"00007": select63,
		"00008": select66,
		"00009": select83,
		"00010": select86,
		"00011": select100,
		"00012": select101,
		"00013": select102,
		"00014": select104,
		"00015": select114,
		"00016": select115,
		"00017": select125,
		"00018": select138,
		"00019": select147,
		"00020": select150,
		"00021": select151,
		"00022": select163,
		"00023": select164,
		"00024": select170,
		"00025": select171,
		"00026": select176,
		"00027": select184,
		"00028": msg469,
		"00029": select188,
		"00030": select197,
		"00031": select205,
		"00032": select207,
		"00033": select214,
		"00034": select225,
		"00035": select232,
		"00036": select234,
		"00037": select241,
		"00038": msg660,
		"00039": msg661,
		"00040": select244,
		"00041": select245,
		"00042": select246,
		"00043": msg668,
		"00044": select248,
		"00045": msg671,
		"00047": msg672,
		"00048": select257,
		"00049": select258,
		"00050": msg682,
		"00051": msg683,
		"00052": msg684,
		"00055": select265,
		"00056": msg696,
		"00057": msg697,
		"00058": msg698,
		"00059": select272,
		"00062": select273,
		"00063": msg713,
		"00064": select274,
		"00070": select276,
		"00071": select277,
		"00072": select278,
		"00073": select279,
		"00074": msg726,
		"00075": select280,
		"00076": select281,
		"00077": select282,
		"00084": msg735,
		"00090": msg736,
		"00200": msg737,
		"00201": msg738,
		"00202": msg739,
		"00203": msg740,
		"00206": select285,
		"00207": select286,
		"00257": select291,
		"00259": select294,
		"00262": msg778,
		"00263": msg779,
		"00400": msg780,
		"00401": msg781,
		"00402": select296,
		"00403": msg784,
		"00404": msg785,
		"00405": msg786,
		"00406": msg787,
		"00407": msg788,
		"00408": msg789,
		"00409": msg790,
		"00410": select297,
		"00411": msg793,
		"00413": select298,
		"00414": select299,
		"00415": msg799,
		"00423": msg800,
		"00429": select300,
		"00430": select301,
		"00431": msg805,
		"00432": msg806,
		"00433": msg807,
		"00434": msg808,
		"00435": select302,
		"00436": select303,
		"00437": select304,
		"00438": select305,
		"00440": select306,
		"00441": msg823,
		"00442": msg824,
		"00443": msg825,
		"00511": select307,
		"00513": msg841,
		"00515": select328,
		"00518": select331,
		"00519": select336,
		"00520": select339,
		"00521": msg890,
		"00522": msg891,
		"00523": msg892,
		"00524": select340,
		"00525": select341,
		"00526": msg912,
		"00527": select348,
		"00528": select354,
		"00529": select357,
		"00530": select358,
		"00531": select362,
		"00533": msg973,
		"00534": msg974,
		"00535": select363,
		"00536": select365,
		"00537": select366,
		"00538": select372,
		"00539": select373,
		"00541": select375,
		"00542": msg1062,
		"00543": msg1063,
		"00544": msg1064,
		"00546": msg1065,
		"00547": select379,
		"00549": msg1070,
		"00551": select381,
		"00553": select385,
		"00554": select391,
		"00555": msg1117,
		"00556": select401,
		"00572": select402,
		"00601": select404,
		"00602": msg1148,
		"00612": msg1149,
		"00615": select403,
		"00620": select408,
		"00622": msg1155,
		"00625": msg1156,
		"00628": msg1157,
		"00767": select430,
		"01269": select431,
		"17852": select432,
		"23184": select433,
		"27052": select434,
		"39568": select435,
	}),
]);

var part1932 = match("MESSAGE#2:00001:02/0", "nwparser.payload", "Address %{group_object->} for %{p0}");

var part1933 = match("MESSAGE#2:00001:02/1_1", "nwparser.p0", "domain address %{domain->} in zone %{p0}");

var part1934 = match("MESSAGE#4:00001:04/3_0", "nwparser.p0", " (%{fld1})");

var part1935 = match("MESSAGE#5:00001:05/1_0", "nwparser.p0", "(%{fld1})");

var part1936 = match_copy("MESSAGE#5:00001:05/1_1", "nwparser.p0", "fld1");

var part1937 = match("MESSAGE#8:00001:08/0", "nwparser.payload", "Address %{p0}");

var part1938 = match("MESSAGE#8:00001:08/1_0", "nwparser.p0", "MIP(%{interface}) %{p0}");

var part1939 = match("MESSAGE#8:00001:08/1_1", "nwparser.p0", "%{group_object->} %{p0}");

var part1940 = match("MESSAGE#8:00001:08/3_0", "nwparser.p0", "admin %{p0}");

var part1941 = match_copy("MESSAGE#8:00001:08/3_1", "nwparser.p0", "p0");

var part1942 = match("MESSAGE#25:00002:20/1_1", "nwparser.p0", "from host %{saddr->} ");

var part1943 = match_copy("MESSAGE#25:00002:20/1_2", "nwparser.p0", "");

var part1944 = match("MESSAGE#26:00002:21/1", "nwparser.p0", "%{p0}");

var part1945 = match("MESSAGE#26:00002:21/2_0", "nwparser.p0", "password %{p0}");

var part1946 = match("MESSAGE#26:00002:21/2_1", "nwparser.p0", "name %{p0}");

var part1947 = match_copy("MESSAGE#27:00002:22/1_2", "nwparser.p0", "administrator");

var part1948 = match_copy("MESSAGE#42:00002:38/1_1", "nwparser.p0", "disposition");

var part1949 = match("MESSAGE#46:00002:42/1_1", "nwparser.p0", "via %{p0}");

var part1950 = match("MESSAGE#46:00002:42/4", "nwparser.p0", "%{fld1})");

var part1951 = match("MESSAGE#52:00002:48/3_1", "nwparser.p0", "%{logon_type->} from host %{saddr->} to %{daddr}:%{dport}. (%{p0}");

var part1952 = match("MESSAGE#53:00002:52/3_0", "nwparser.p0", "admin %{administrator->} via %{p0}");

var part1953 = match("MESSAGE#53:00002:52/3_2", "nwparser.p0", "%{username->} via %{p0}");

var part1954 = match("MESSAGE#53:00002:52/4_0", "nwparser.p0", "NSRP Peer . (%{p0}");

var part1955 = match("MESSAGE#55:00002:54/2", "nwparser.p0", ". (%{fld1})");

var part1956 = match("MESSAGE#56:00002/1_1", "nwparser.p0", "changed%{p0}");

var part1957 = match("MESSAGE#61:00003:05/0", "nwparser.payload", "The %{p0}");

var part1958 = match("MESSAGE#66:00004:04/1_0", "nwparser.p0", "interface%{p0}");

var part1959 = match("MESSAGE#66:00004:04/1_1", "nwparser.p0", "Interface%{p0}");

var part1960 = match("MESSAGE#76:00004:14/0", "nwparser.payload", "DNS entries have been %{p0}");

var part1961 = match("MESSAGE#79:00004:17/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{p0}");

var part1962 = match("MESSAGE#79:00004:17/1_0", "nwparser.p0", "%{zone}, %{p0}");

var part1963 = match("MESSAGE#79:00004:17/1_1", "nwparser.p0", "%{zone->} %{p0}");

var part1964 = match("MESSAGE#79:00004:17/2", "nwparser.p0", "int %{interface}).%{space}Occurred %{dclass_counter1->} times. (%{fld1})");

var part1965 = match("MESSAGE#83:00005:03/1_0", "nwparser.p0", "%{dport},%{p0}");

var part1966 = match("MESSAGE#83:00005:03/1_1", "nwparser.p0", "%{dport->} %{p0}");

var part1967 = match("MESSAGE#83:00005:03/2", "nwparser.p0", "%{space}using protocol %{p0}");

var part1968 = match("MESSAGE#83:00005:03/3_0", "nwparser.p0", "%{protocol},%{p0}");

var part1969 = match("MESSAGE#83:00005:03/3_1", "nwparser.p0", "%{protocol->} %{p0}");

var part1970 = match("MESSAGE#83:00005:03/5_1", "nwparser.p0", ". %{p0}");

var part1971 = match("MESSAGE#86:00005:06/0_0", "nwparser.payload", "%{fld2}: SYN %{p0}");

var part1972 = match("MESSAGE#86:00005:06/0_1", "nwparser.payload", "SYN %{p0}");

var part1973 = match("MESSAGE#87:00005:07/1_2", "nwparser.p0", "timeout value %{p0}");

var part1974 = match("MESSAGE#88:00005:08/2_0", "nwparser.p0", "destination %{p0}");

var part1975 = match("MESSAGE#88:00005:08/2_1", "nwparser.p0", "source %{p0}");

var part1976 = match("MESSAGE#97:00005:17/0", "nwparser.payload", "A %{p0}");

var part1977 = match("MESSAGE#98:00005:18/0", "nwparser.payload", "%{signame->} From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var part1978 = match("MESSAGE#98:00005:18/1_0", "nwparser.p0", ", int %{p0}");

var part1979 = match("MESSAGE#98:00005:18/1_1", "nwparser.p0", "int %{p0}");

var part1980 = match("MESSAGE#98:00005:18/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times. (%{fld1})");

var part1981 = match("MESSAGE#111:00007:04/0", "nwparser.payload", "HA %{p0}");

var part1982 = match("MESSAGE#111:00007:04/1_0", "nwparser.p0", "encryption %{p0}");

var part1983 = match("MESSAGE#111:00007:04/1_1", "nwparser.p0", "authentication %{p0}");

var part1984 = match("MESSAGE#111:00007:04/3_1", "nwparser.p0", "key %{p0}");

var part1985 = match("MESSAGE#118:00007:11/1_0", "nwparser.p0", "disabled%{}");

var part1986 = match("MESSAGE#118:00007:11/1_1", "nwparser.p0", "set to %{trigger_val}");

var part1987 = match("MESSAGE#127:00007:21/1_0", "nwparser.p0", "up%{}");

var part1988 = match("MESSAGE#127:00007:21/1_1", "nwparser.p0", "down%{}");

var part1989 = match("MESSAGE#139:00007:33/2_1", "nwparser.p0", " %{p0}");

var part1990 = match("MESSAGE#143:00007:37/1_0", "nwparser.p0", "set%{}");

var part1991 = match("MESSAGE#143:00007:37/1_1", "nwparser.p0", "unset%{}");

var part1992 = match("MESSAGE#144:00007:38/1_0", "nwparser.p0", "undefined %{p0}");

var part1993 = match("MESSAGE#144:00007:38/1_1", "nwparser.p0", "set %{p0}");

var part1994 = match("MESSAGE#144:00007:38/1_2", "nwparser.p0", "active %{p0}");

var part1995 = match("MESSAGE#144:00007:38/2", "nwparser.p0", "to %{p0}");

var part1996 = match("MESSAGE#157:00007:51/1_0", "nwparser.p0", "created %{p0}");

var part1997 = match("MESSAGE#157:00007:51/3_0", "nwparser.p0", ", %{p0}");

var part1998 = match("MESSAGE#157:00007:51/5_0", "nwparser.p0", "is %{p0}");

var part1999 = match("MESSAGE#157:00007:51/5_1", "nwparser.p0", "was %{p0}");

var part2000 = match("MESSAGE#157:00007:51/6", "nwparser.p0", "%{fld2}");

var part2001 = match("MESSAGE#163:00007:57/1_0", "nwparser.p0", "threshold %{p0}");

var part2002 = match("MESSAGE#163:00007:57/1_1", "nwparser.p0", "interval %{p0}");

var part2003 = match("MESSAGE#163:00007:57/3_0", "nwparser.p0", "of %{p0}");

var part2004 = match("MESSAGE#163:00007:57/3_1", "nwparser.p0", "that %{p0}");

var part2005 = match("MESSAGE#170:00007:64/0_0", "nwparser.payload", "Zone %{p0}");

var part2006 = match("MESSAGE#170:00007:64/0_1", "nwparser.payload", "Interface %{p0}");

var part2007 = match("MESSAGE#172:00007:66/2_1", "nwparser.p0", "n %{p0}");

var part2008 = match("MESSAGE#174:00007:68/4", "nwparser.p0", ".%{}");

var part2009 = match("MESSAGE#195:00009:06/1", "nwparser.p0", "for %{p0}");

var part2010 = match("MESSAGE#195:00009:06/2_0", "nwparser.p0", "the %{p0}");

var part2011 = match("MESSAGE#195:00009:06/4_0", "nwparser.p0", "removed %{p0}");

var part2012 = match("MESSAGE#202:00009:14/2_0", "nwparser.p0", "interface %{p0}");

var part2013 = match("MESSAGE#202:00009:14/2_1", "nwparser.p0", "the interface %{p0}");

var part2014 = match_copy("MESSAGE#202:00009:14/4_1", "nwparser.p0", "interface");

var part2015 = match("MESSAGE#203:00009:15/1_1", "nwparser.p0", "s %{p0}");

var part2016 = match("MESSAGE#203:00009:15/2", "nwparser.p0", "on interface %{interface->} %{p0}");

var part2017 = match("MESSAGE#203:00009:15/3_0", "nwparser.p0", "has been %{p0}");

var part2018 = match("MESSAGE#203:00009:15/4", "nwparser.p0", "%{disposition}.");

var part2019 = match("MESSAGE#204:00009:16/3_0", "nwparser.p0", "removed from %{p0}");

var part2020 = match("MESSAGE#204:00009:16/3_1", "nwparser.p0", "added to %{p0}");

var part2021 = match("MESSAGE#210:00009:21/2", "nwparser.p0", "%{interface}). Occurred %{dclass_counter1->} times. (%{fld1})");

var part2022 = match("MESSAGE#219:00010:03/0", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, proto %{protocol->} (zone %{zone->} %{p0}");

var part2023 = match("MESSAGE#224:00011:04/1_1", "nwparser.p0", "Interface %{p0}");

var part2024 = match("MESSAGE#233:00011:14/1_0", "nwparser.p0", "set to %{fld2}");

var part2025 = match("MESSAGE#237:00011:18/4_1", "nwparser.p0", "gateway %{p0}");

var part2026 = match("MESSAGE#238:00011:19/6", "nwparser.p0", "%{} %{disposition}");

var part2027 = match("MESSAGE#274:00015:02/1_1", "nwparser.p0", "port number %{p0}");

var part2028 = match("MESSAGE#274:00015:02/2", "nwparser.p0", "has been %{disposition}");

var part2029 = match("MESSAGE#276:00015:04/1_0", "nwparser.p0", "IP %{p0}");

var part2030 = match("MESSAGE#276:00015:04/1_1", "nwparser.p0", "port %{p0}");

var part2031 = match("MESSAGE#284:00015:12/3_0", "nwparser.p0", "up %{p0}");

var part2032 = match("MESSAGE#284:00015:12/3_1", "nwparser.p0", "down %{p0}");

var part2033 = match("MESSAGE#294:00015:22/2_0", "nwparser.p0", "(%{fld1}) ");

var part2034 = match("MESSAGE#317:00017:01/2_0", "nwparser.p0", ": %{p0}");

var part2035 = match("MESSAGE#320:00017:04/0", "nwparser.payload", "IP %{p0}");

var part2036 = match("MESSAGE#320:00017:04/1_0", "nwparser.p0", "address pool %{p0}");

var part2037 = match("MESSAGE#320:00017:04/1_1", "nwparser.p0", "pool %{p0}");

var part2038 = match("MESSAGE#326:00017:10/1_0", "nwparser.p0", "enabled %{p0}");

var part2039 = match("MESSAGE#326:00017:10/1_1", "nwparser.p0", "disabled %{p0}");

var part2040 = match("MESSAGE#332:00017:15/1_0", "nwparser.p0", "AH %{p0}");

var part2041 = match("MESSAGE#332:00017:15/1_1", "nwparser.p0", "ESP %{p0}");

var part2042 = match("MESSAGE#354:00018:11/0", "nwparser.payload", "%{} %{p0}");

var part2043 = match("MESSAGE#356:00018:32/0_0", "nwparser.payload", "Source%{p0}");

var part2044 = match("MESSAGE#356:00018:32/0_1", "nwparser.payload", "Destination%{p0}");

var part2045 = match("MESSAGE#356:00018:32/2_0", "nwparser.p0", "from %{p0}");

var part2046 = match("MESSAGE#356:00018:32/3", "nwparser.p0", "policy ID %{policy_id->} by admin %{administrator->} via NSRP Peer . (%{fld1})");

var part2047 = match("MESSAGE#375:00019:01/0", "nwparser.payload", "Attempt to enable %{p0}");

var part2048 = match("MESSAGE#375:00019:01/1_0", "nwparser.p0", "traffic logging via syslog %{p0}");

var part2049 = match("MESSAGE#375:00019:01/1_1", "nwparser.p0", "syslog %{p0}");

var part2050 = match("MESSAGE#378:00019:04/0", "nwparser.payload", "Syslog %{p0}");

var part2051 = match("MESSAGE#378:00019:04/1_0", "nwparser.p0", "host %{p0}");

var part2052 = match("MESSAGE#378:00019:04/3_1", "nwparser.p0", "domain name %{p0}");

var part2053 = match("MESSAGE#378:00019:04/4", "nwparser.p0", "has been changed to %{fld2}");

var part2054 = match("MESSAGE#380:00019:06/1_0", "nwparser.p0", "security facility %{p0}");

var part2055 = match("MESSAGE#380:00019:06/1_1", "nwparser.p0", "facility %{p0}");

var part2056 = match("MESSAGE#380:00019:06/3_0", "nwparser.p0", "local0%{}");

var part2057 = match("MESSAGE#380:00019:06/3_1", "nwparser.p0", "local1%{}");

var part2058 = match("MESSAGE#380:00019:06/3_2", "nwparser.p0", "local2%{}");

var part2059 = match("MESSAGE#380:00019:06/3_3", "nwparser.p0", "local3%{}");

var part2060 = match("MESSAGE#380:00019:06/3_4", "nwparser.p0", "local4%{}");

var part2061 = match("MESSAGE#380:00019:06/3_5", "nwparser.p0", "local5%{}");

var part2062 = match("MESSAGE#380:00019:06/3_6", "nwparser.p0", "local6%{}");

var part2063 = match("MESSAGE#380:00019:06/3_7", "nwparser.p0", "local7%{}");

var part2064 = match("MESSAGE#380:00019:06/3_8", "nwparser.p0", "auth/sec%{}");

var part2065 = match("MESSAGE#384:00019:10/0", "nwparser.payload", "%{fld2->} %{p0}");

var part2066 = match("MESSAGE#405:00022/0", "nwparser.payload", "All %{p0}");

var part2067 = match("MESSAGE#414:00022:09/1_0", "nwparser.p0", "primary %{p0}");

var part2068 = match("MESSAGE#414:00022:09/1_1", "nwparser.p0", "secondary %{p0}");

var part2069 = match("MESSAGE#414:00022:09/3_0", "nwparser.p0", "t %{p0}");

var part2070 = match("MESSAGE#414:00022:09/3_1", "nwparser.p0", "w %{p0}");

var part2071 = match("MESSAGE#423:00024/1", "nwparser.p0", "server %{p0}");

var part2072 = match("MESSAGE#426:00024:03/1_0", "nwparser.p0", "has %{p0}");

var part2073 = match("MESSAGE#434:00026:01/0", "nwparser.payload", "SCS%{p0}");

var part2074 = match("MESSAGE#434:00026:01/3_0", "nwparser.p0", "bound to %{p0}");

var part2075 = match("MESSAGE#434:00026:01/3_1", "nwparser.p0", "unbound from %{p0}");

var part2076 = match("MESSAGE#441:00026:08/1_1", "nwparser.p0", "PKA RSA %{p0}");

var part2077 = match("MESSAGE#443:00026:10/3_1", "nwparser.p0", "unbind %{p0}");

var part2078 = match("MESSAGE#443:00026:10/4", "nwparser.p0", "PKA key %{p0}");

var part2079 = match("MESSAGE#446:00027/0", "nwparser.payload", "Multiple login failures %{p0}");

var part2080 = match("MESSAGE#446:00027/1_0", "nwparser.p0", "occurred for %{p0}");

var part2081 = match("MESSAGE#451:00027:05/5_0", "nwparser.p0", "aborted%{}");

var part2082 = match("MESSAGE#451:00027:05/5_1", "nwparser.p0", "performed%{}");

var part2083 = match("MESSAGE#466:00029:03/0", "nwparser.payload", "IP pool of DHCP server on %{p0}");

var part2084 = match("MESSAGE#492:00030:17/1_0", "nwparser.p0", "certificate %{p0}");

var part2085 = match("MESSAGE#492:00030:17/1_1", "nwparser.p0", "CRL %{p0}");

var part2086 = match("MESSAGE#493:00030:40/1_0", "nwparser.p0", "auto %{p0}");

var part2087 = match("MESSAGE#508:00030:55/1_0", "nwparser.p0", "RSA %{p0}");

var part2088 = match("MESSAGE#508:00030:55/1_1", "nwparser.p0", "DSA %{p0}");

var part2089 = match("MESSAGE#508:00030:55/2", "nwparser.p0", "key pair.%{}");

var part2090 = match("MESSAGE#539:00030:86/0", "nwparser.payload", "FIPS test for %{p0}");

var part2091 = match("MESSAGE#539:00030:86/1_0", "nwparser.p0", "ECDSA %{p0}");

var part2092 = match("MESSAGE#543:00031:02/1_0", "nwparser.p0", "yes %{p0}");

var part2093 = match("MESSAGE#543:00031:02/1_1", "nwparser.p0", "no %{p0}");

var part2094 = match("MESSAGE#545:00031:04/1_1", "nwparser.p0", "location %{p0}");

var part2095 = match("MESSAGE#548:00031:05/2", "nwparser.p0", "%{} %{interface}");

var part2096 = match("MESSAGE#549:00031:06/0", "nwparser.payload", "arp re%{p0}");

var part2097 = match("MESSAGE#549:00031:06/1_1", "nwparser.p0", "q %{p0}");

var part2098 = match("MESSAGE#549:00031:06/1_2", "nwparser.p0", "ply %{p0}");

var part2099 = match("MESSAGE#549:00031:06/9_0", "nwparser.p0", "%{interface->} (%{fld1})");

var part2100 = match("MESSAGE#561:00033/0_0", "nwparser.payload", "Global PRO %{p0}");

var part2101 = match("MESSAGE#561:00033/0_1", "nwparser.payload", "%{fld3->} %{p0}");

var part2102 = match("MESSAGE#569:00033:08/0", "nwparser.payload", "NACN Policy Manager %{p0}");

var part2103 = match("MESSAGE#569:00033:08/1_0", "nwparser.p0", "1 %{p0}");

var part2104 = match("MESSAGE#569:00033:08/1_1", "nwparser.p0", "2 %{p0}");

var part2105 = match("MESSAGE#571:00033:10/3_1", "nwparser.p0", "unset %{p0}");

var part2106 = match("MESSAGE#581:00033:21/0", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var part2107 = match("MESSAGE#586:00034:01/2_1", "nwparser.p0", "SSH %{p0}");

var part2108 = match("MESSAGE#588:00034:03/0_0", "nwparser.payload", "SCS: NetScreen %{p0}");

var part2109 = match("MESSAGE#588:00034:03/0_1", "nwparser.payload", "NetScreen %{p0}");

var part2110 = match("MESSAGE#595:00034:10/0", "nwparser.payload", "S%{p0}");

var part2111 = match("MESSAGE#595:00034:10/1_0", "nwparser.p0", "CS: SSH%{p0}");

var part2112 = match("MESSAGE#595:00034:10/1_1", "nwparser.p0", "SH%{p0}");

var part2113 = match("MESSAGE#596:00034:12/3_0", "nwparser.p0", "the root system %{p0}");

var part2114 = match("MESSAGE#596:00034:12/3_1", "nwparser.p0", "vsys %{fld2->} %{p0}");

var part2115 = match("MESSAGE#599:00034:18/1_0", "nwparser.p0", "CS: SSH %{p0}");

var part2116 = match("MESSAGE#599:00034:18/1_1", "nwparser.p0", "SH %{p0}");

var part2117 = match("MESSAGE#630:00035:06/1_0", "nwparser.p0", "a %{p0}");

var part2118 = match("MESSAGE#630:00035:06/1_1", "nwparser.p0", "ert %{p0}");

var part2119 = match("MESSAGE#633:00035:09/0", "nwparser.payload", "SSL %{p0}");

var part2120 = match("MESSAGE#644:00037:01/1_0", "nwparser.p0", "id: %{p0}");

var part2121 = match("MESSAGE#644:00037:01/1_1", "nwparser.p0", "ID %{p0}");

var part2122 = match("MESSAGE#659:00044/1_0", "nwparser.p0", "permit %{p0}");

var part2123 = match("MESSAGE#675:00055/0", "nwparser.payload", "IGMP %{p0}");

var part2124 = match("MESSAGE#677:00055:02/0", "nwparser.payload", "IGMP will %{p0}");

var part2125 = match("MESSAGE#677:00055:02/1_0", "nwparser.p0", "not do %{p0}");

var part2126 = match("MESSAGE#677:00055:02/1_1", "nwparser.p0", "do %{p0}");

var part2127 = match("MESSAGE#689:00059/1_1", "nwparser.p0", "shut down %{p0}");

var part2128 = match("MESSAGE#707:00070/0", "nwparser.payload", "NSRP: %{p0}");

var part2129 = match("MESSAGE#707:00070/1_0", "nwparser.p0", "Unit %{p0}");

var part2130 = match("MESSAGE#707:00070/1_1", "nwparser.p0", "local unit= %{p0}");

var part2131 = match("MESSAGE#707:00070/2", "nwparser.p0", "%{fld2->} of VSD group %{group->} %{info}");

var part2132 = match("MESSAGE#708:00070:01/0", "nwparser.payload", "The local device %{fld2->} in the Virtual Sec%{p0}");

var part2133 = match("MESSAGE#708:00070:01/1_0", "nwparser.p0", "ruity%{p0}");

var part2134 = match("MESSAGE#708:00070:01/1_1", "nwparser.p0", "urity%{p0}");

var part2135 = match("MESSAGE#713:00072:01/2", "nwparser.p0", "%{}Device group %{group->} changed state");

var part2136 = match("MESSAGE#717:00075/2", "nwparser.p0", "%{fld2->} of VSD group %{group->} %{info}");

var part2137 = match("MESSAGE#748:00257:19/0", "nwparser.payload", "start_time=%{p0}");

var part2138 = match("MESSAGE#748:00257:19/1_0", "nwparser.p0", "\\\"%{fld2}\\\"%{p0}");

var part2139 = match("MESSAGE#748:00257:19/1_1", "nwparser.p0", " \"%{fld2}\" %{p0}");

var part2140 = match_copy("MESSAGE#756:00257:10/1_1", "nwparser.p0", "daddr");

var part2141 = match("MESSAGE#760:00259/0_0", "nwparser.payload", "Admin %{p0}");

var part2142 = match("MESSAGE#760:00259/0_1", "nwparser.payload", "Vsys admin %{p0}");

var part2143 = match("MESSAGE#760:00259/2_1", "nwparser.p0", "Telnet %{p0}");

var part2144 = match("MESSAGE#777:00406/2", "nwparser.p0", "%{interface}). Occurred %{dclass_counter1->} times.");

var part2145 = match("MESSAGE#790:00423/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times.");

var part2146 = match("MESSAGE#793:00430/2", "nwparser.p0", "%{interface}).%{space}Occurred %{dclass_counter1->} times.%{p0}");

var part2147 = match("MESSAGE#795:00431/0", "nwparser.payload", "%{obj_type->} %{disposition}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var part2148 = match("MESSAGE#797:00433/0", "nwparser.payload", "%{signame->} %{disposition}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{zone->} %{p0}");

var part2149 = match("MESSAGE#804:00437:01/0", "nwparser.payload", "%{signame}! From %{saddr}:%{sport->} to %{daddr}:%{dport}, proto %{protocol->} (zone %{p0}");

var part2150 = match("MESSAGE#817:00511:01/1_0", "nwparser.p0", "%{administrator->} (%{fld1})");

var part2151 = match("MESSAGE#835:00515:04/2_1", "nwparser.p0", "ut %{p0}");

var part2152 = match("MESSAGE#835:00515:04/4_0", "nwparser.p0", "%{logon_type->} from %{saddr}:%{sport}");

var part2153 = match("MESSAGE#837:00515:05/1_0", "nwparser.p0", "user %{p0}");

var part2154 = match("MESSAGE#837:00515:05/5_0", "nwparser.p0", "the %{logon_type}");

var part2155 = match("MESSAGE#869:00519:01/1_0", "nwparser.p0", "WebAuth user %{p0}");

var part2156 = match("MESSAGE#876:00520:02/1_1", "nwparser.p0", "backup1 %{p0}");

var part2157 = match("MESSAGE#876:00520:02/1_2", "nwparser.p0", "backup2 %{p0}");

var part2158 = match("MESSAGE#890:00524:13/1_0", "nwparser.p0", ",%{p0}");

var part2159 = match("MESSAGE#901:00527/1_0", "nwparser.p0", "assigned %{p0}");

var part2160 = match("MESSAGE#901:00527/3_0", "nwparser.p0", "assigned to %{p0}");

var part2161 = match("MESSAGE#927:00528:15/1_0", "nwparser.p0", "'%{administrator}' %{p0}");

var part2162 = match("MESSAGE#930:00528:18/0", "nwparser.payload", "SSH: P%{p0}");

var part2163 = match("MESSAGE#930:00528:18/1_0", "nwparser.p0", "KA %{p0}");

var part2164 = match("MESSAGE#930:00528:18/1_1", "nwparser.p0", "assword %{p0}");

var part2165 = match("MESSAGE#930:00528:18/3_0", "nwparser.p0", "\\'%{administrator}\\' %{p0}");

var part2166 = match("MESSAGE#930:00528:18/4", "nwparser.p0", "at host %{saddr}");

var part2167 = match("MESSAGE#932:00528:19/0", "nwparser.payload", "%{}S%{p0}");

var part2168 = match("MESSAGE#932:00528:19/1_0", "nwparser.p0", "CS %{p0}");

var part2169 = match("MESSAGE#1060:00553/2", "nwparser.p0", "from server.ini file.%{}");

var part2170 = match("MESSAGE#1064:00553:04/1_0", "nwparser.p0", "pattern %{p0}");

var part2171 = match("MESSAGE#1064:00553:04/1_1", "nwparser.p0", "server.ini %{p0}");

var part2172 = match("MESSAGE#1068:00553:08/2", "nwparser.p0", "file.%{}");

var part2173 = match("MESSAGE#1087:00554:04/1_1", "nwparser.p0", "AV pattern %{p0}");

var part2174 = match("MESSAGE#1116:00556:14/1_0", "nwparser.p0", "added into %{p0}");

var part2175 = match("MESSAGE#1157:00767:11/1_0", "nwparser.p0", "loader %{p0}");

var select436 = linear_select([
	dup10,
	dup11,
]);

var part2176 = match("MESSAGE#7:00001:07", "nwparser.payload", "Policy ID=%{policy_id->} Rate=%{fld2->} exceeds threshold", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var select437 = linear_select([
	dup13,
	dup14,
]);

var select438 = linear_select([
	dup15,
	dup16,
]);

var select439 = linear_select([
	dup56,
	dup57,
]);

var select440 = linear_select([
	dup65,
	dup66,
]);

var select441 = linear_select([
	dup68,
	dup69,
]);

var select442 = linear_select([
	dup71,
	dup72,
]);

var part2177 = match("MESSAGE#84:00005:04", "nwparser.payload", "%{signame->} from %{saddr}/%{sport->} to %{daddr}/%{dport->} protocol %{protocol->} (%{interface})", processor_chain([
	dup58,
	dup2,
	dup3,
	dup4,
	dup5,
	dup61,
]));

var select443 = linear_select([
	dup74,
	dup75,
]);

var select444 = linear_select([
	dup81,
	dup82,
]);

var select445 = linear_select([
	dup24,
	dup90,
]);

var select446 = linear_select([
	dup94,
	dup95,
]);

var select447 = linear_select([
	dup98,
	dup99,
]);

var select448 = linear_select([
	dup100,
	dup101,
	dup102,
]);

var select449 = linear_select([
	dup113,
	dup114,
]);

var select450 = linear_select([
	dup111,
	dup16,
]);

var select451 = linear_select([
	dup127,
	dup107,
]);

var select452 = linear_select([
	dup8,
	dup21,
]);

var select453 = linear_select([
	dup122,
	dup133,
]);

var select454 = linear_select([
	dup142,
	dup143,
]);

var select455 = linear_select([
	dup145,
	dup21,
]);

var select456 = linear_select([
	dup127,
	dup106,
]);

var select457 = linear_select([
	dup152,
	dup96,
]);

var select458 = linear_select([
	dup154,
	dup155,
]);

var select459 = linear_select([
	dup156,
	dup157,
]);

var select460 = linear_select([
	dup99,
	dup134,
]);

var select461 = linear_select([
	dup158,
	dup159,
]);

var select462 = linear_select([
	dup161,
	dup162,
]);

var select463 = linear_select([
	dup163,
	dup103,
]);

var select464 = linear_select([
	dup162,
	dup161,
]);

var select465 = linear_select([
	dup46,
	dup47,
]);

var select466 = linear_select([
	dup166,
	dup167,
]);

var select467 = linear_select([
	dup172,
	dup173,
]);

var select468 = linear_select([
	dup174,
	dup175,
	dup176,
	dup177,
	dup178,
	dup179,
	dup180,
	dup181,
	dup182,
]);

var select469 = linear_select([
	dup49,
	dup21,
]);

var select470 = linear_select([
	dup189,
	dup190,
]);

var select471 = linear_select([
	dup96,
	dup152,
]);

var select472 = linear_select([
	dup196,
	dup197,
]);

var select473 = linear_select([
	dup24,
	dup200,
]);

var select474 = linear_select([
	dup103,
	dup163,
]);

var select475 = linear_select([
	dup205,
	dup118,
]);

var part2178 = match("MESSAGE#477:00030:02", "nwparser.payload", "%{change_attribute->} has been changed from %{change_old->} to %{change_new}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var select476 = linear_select([
	dup212,
	dup213,
]);

var select477 = linear_select([
	dup215,
	dup216,
]);

var select478 = linear_select([
	dup222,
	dup215,
]);

var select479 = linear_select([
	dup224,
	dup225,
]);

var select480 = linear_select([
	dup231,
	dup124,
]);

var select481 = linear_select([
	dup229,
	dup230,
]);

var select482 = linear_select([
	dup233,
	dup234,
]);

var select483 = linear_select([
	dup236,
	dup237,
]);

var select484 = linear_select([
	dup242,
	dup243,
]);

var select485 = linear_select([
	dup245,
	dup246,
]);

var select486 = linear_select([
	dup247,
	dup248,
]);

var select487 = linear_select([
	dup249,
	dup250,
]);

var select488 = linear_select([
	dup251,
	dup252,
]);

var select489 = linear_select([
	dup260,
	dup261,
]);

var select490 = linear_select([
	dup264,
	dup265,
]);

var select491 = linear_select([
	dup268,
	dup269,
]);

var part2179 = match("MESSAGE#716:00074", "nwparser.payload", "The local device %{fld2->} in the Virtual Security Device group %{group->} %{info}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
]));

var select492 = linear_select([
	dup284,
	dup285,
]);

var select493 = linear_select([
	dup287,
	dup288,
]);

var part2180 = match("MESSAGE#799:00435", "nwparser.payload", "%{signame->} From %{saddr->} to %{daddr}, using protocol %{protocol}, and arriving at interface %{dinterface->} in zone %{dst_zone}.%{space}The attack occurred %{dclass_counter1->} times.", processor_chain([
	dup58,
	dup2,
	dup59,
	dup4,
	dup5,
	dup3,
	dup60,
]));

var part2181 = match("MESSAGE#814:00442", "nwparser.payload", "%{signame->} From %{saddr->} to zone %{zone}, proto %{protocol->} (int %{interface}). Occurred %{dclass_counter1->} times. (%{fld1})", processor_chain([
	dup58,
	dup4,
	dup59,
	dup5,
	dup9,
	dup2,
	dup3,
	dup60,
]));

var select494 = linear_select([
	dup300,
	dup26,
]);

var select495 = linear_select([
	dup115,
	dup303,
]);

var select496 = linear_select([
	dup125,
	dup96,
]);

var select497 = linear_select([
	dup189,
	dup308,
	dup309,
]);

var select498 = linear_select([
	dup310,
	dup16,
]);

var select499 = linear_select([
	dup317,
	dup318,
]);

var select500 = linear_select([
	dup319,
	dup315,
]);

var select501 = linear_select([
	dup322,
	dup250,
]);

var select502 = linear_select([
	dup327,
	dup329,
]);

var select503 = linear_select([
	dup330,
	dup129,
]);

var part2182 = match("MESSAGE#1196:01269:01", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} direction=%{direction->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup60,
	dup282,
]));

var part2183 = match("MESSAGE#1197:01269:02", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup277,
	dup60,
]));

var part2184 = match("MESSAGE#1198:01269:03", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} proto=%{protocol->} src zone=%{src_zone->} dst zone=%{dst_zone->} action=%{disposition->} sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} icmp type=%{icmptype}", processor_chain([
	dup281,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup60,
	dup282,
]));

var part2185 = match("MESSAGE#1203:23184", "nwparser.payload", "start_time=\"%{fld2}\" duration=%{duration->} policy_id=%{policy_id->} service=%{service->} (%{fld3}) proto=%{protocol->} direction=%{direction->} action=Deny sent=%{sbytes->} rcvd=%{rbytes->} src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport}", processor_chain([
	dup185,
	dup2,
	dup4,
	dup5,
	dup274,
	dup3,
	dup275,
	dup276,
	dup277,
	dup61,
]));

var all391 = all_match({
	processors: [
		dup263,
		dup390,
		dup266,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var all392 = all_match({
	processors: [
		dup267,
		dup391,
		dup270,
	],
	on_success: processor_chain([
		dup1,
		dup2,
		dup3,
		dup4,
		dup5,
	]),
});

var all393 = all_match({
	processors: [
		dup80,
		dup343,
		dup293,
	],
	on_success: processor_chain([
		dup58,
		dup2,
		dup59,
		dup3,
		dup4,
		dup5,
		dup61,
	]),
});

var all394 = all_match({
	processors: [
		dup296,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup297,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup61,
	]),
});

var all395 = all_match({
	processors: [
		dup298,
		dup343,
		dup131,
	],
	on_success: processor_chain([
		dup297,
		dup2,
		dup3,
		dup9,
		dup59,
		dup4,
		dup5,
		dup61,
	]),
});
