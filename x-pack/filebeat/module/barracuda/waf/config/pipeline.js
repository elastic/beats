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
		field("hfld7"),
		constant(" "),
		field("hfld8"),
		constant("."),
		field("hfld2"),
		constant(" "),
		field("hfld3"),
		constant(" "),
		field("hfld4"),
		constant(" "),
		field("hfld5"),
		constant(" "),
		field("hfld6"),
		constant(" "),
		field("messageid"),
		constant(" "),
		field("p0"),
	],
});

var dup2 = setc("messageid","BARRACUDA_GENRIC");

var dup3 = setc("eventcategory","1605000000");

var dup4 = setc("eventcategory","1613030000");

var dup5 = setc("event_description","STM: aps SetIpsLimitPolicy.");

var dup6 = setc("eventcategory","1603020000");

var dup7 = setc("eventcategory","1701000000");

var dup8 = date_time({
	dest: "event_time",
	args: ["hfld7","hfld8"],
	fmts: [
		[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup9 = setc("eventcategory","1401070000");

var dup10 = setc("eventcategory","1401000000");

var dup11 = setc("eventcategory","1201000000");

var dup12 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} %{p0}");

var dup13 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/1_0", "nwparser.p0", "\"[%{result}]\" %{p0}");

var dup14 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/1_1", "nwparser.p0", "[%{result}] %{p0}");

var dup15 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/2", "nwparser.p0", "%{web_method->} %{url->} %{protocol->} - %{stransaddr->} %{stransport->} %{web_referer}");

var dup16 = match("MESSAGE#85:CROSS_SITE_SCRIPTING_IN_PARAM:01/2", "nwparser.p0", "%{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}");

var dup17 = setc("eventcategory","1204000000");

var dup18 = match("MESSAGE#118:TR_Logs:01/1_0", "nwparser.p0", "%{stransport->} %{content_type}");

var dup19 = match_copy("MESSAGE#118:TR_Logs:01/1_1", "nwparser.p0", "stransport");

var dup20 = setf("msg_id","web_method");

var dup21 = setc("category","TR");

var dup22 = setc("vid","TR_Logs");

var dup23 = linear_select([
	dup13,
	dup14,
]);

var dup24 = match("MESSAGE#103:NO_DOMAIN_MATCH_IN_PROFILE", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} [%{result}] %{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}", processor_chain([
	dup17,
	dup8,
]));

var dup25 = linear_select([
	dup18,
	dup19,
]);

var dup26 = all_match({
	processors: [
		dup12,
		dup23,
		dup15,
	],
	on_success: processor_chain([
		dup11,
		dup8,
	]),
});

var dup27 = all_match({
	processors: [
		dup12,
		dup23,
		dup16,
	],
	on_success: processor_chain([
		dup11,
		dup8,
	]),
});

var hdr1 = match("HEADER#0:0001", "message", "%{messageid}:%{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("messageid"),
			constant(":"),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0005", "message", "time=%{hfld1->} %{hfld2->} %{timezone->} Unit=%{messageid->} %{payload}", processor_chain([
	setc("header_id","0005"),
]));

var hdr3 = match("HEADER#2:0003", "message", "%{hfld9->} %{hfld10->} %{hfld11->} %{hfld12->} %{hhost->} %{hfld7->} %{hfld8}.%{hfld2->} %{hfld3->} %{hfld4->} %{hfld5->} %{hfld6->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0003"),
	dup1,
]));

var hdr4 = match("HEADER#3:0002", "message", "%{hhost->} %{hfld7->} %{hfld8}.%{hfld2->} %{hfld3->} %{hfld4->} %{hfld5->} %{hfld6->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0002"),
	dup1,
]));

var hdr5 = match("HEADER#4:0009", "message", "%{hhost->} %{hfld7->} %{hfld8}.%{hfld2->} %{hfld3->} TR %{hfld5->} %{hfld6->} %{hfld8->} %{p0}", processor_chain([
	setc("header_id","0009"),
	dup2,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld7"),
			constant(" "),
			field("hfld8"),
			constant("."),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(" TR "),
			field("hfld5"),
			constant(" "),
			field("hfld6"),
			constant(" "),
			field("hfld8"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr6 = match("HEADER#5:0007", "message", "%{hhost->} %{hfld7->} %{hfld8}.%{hfld2->} %{hfld3->} AUDIT %{hfld5->} %{hfld6->} %{hfld8->} %{p0}", processor_chain([
	setc("header_id","0007"),
	dup2,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld7"),
			constant(" "),
			field("hfld8"),
			constant("."),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(" AUDIT "),
			field("hfld5"),
			constant(" "),
			field("hfld6"),
			constant(" "),
			field("hfld8"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr7 = match("HEADER#6:0008", "message", "%{hhost->} %{hfld7->} %{hfld8}.%{hfld2->} %{hfld3->} WF %{hfld5->} %{hfld6->} %{hfld8->} %{p0}", processor_chain([
	setc("header_id","0008"),
	dup2,
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld7"),
			constant(" "),
			field("hfld8"),
			constant("."),
			field("hfld2"),
			constant(" "),
			field("hfld3"),
			constant(" WF "),
			field("hfld5"),
			constant(" "),
			field("hfld6"),
			constant(" "),
			field("hfld8"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr8 = match("HEADER#7:0006", "message", "%{hmonth->} %{hday->} %{htime->} BARRACUDAWAF %{hhost->} %{hdate->} %{htime->} %{htimezone->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0006"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hhost"),
			constant(" "),
			field("hdate"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("htimezone"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr9 = match("HEADER#8:0004", "message", "%{hfld9->} %{hfld10->} %{hfld11->} %{hhost->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0004"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld10"),
			constant(" "),
			field("hfld11"),
			constant(" "),
			field("hhost"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
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
	hdr9,
]);

var part1 = match("MESSAGE#0:UPDATE", "nwparser.payload", "UPDATE: [ALERT:%{fld3}] New attack definition version %{version->} is available", processor_chain([
	setc("eventcategory","1502030000"),
	setc("event_description","UPDATE: ALERT New attack definition version is available"),
]));

var msg1 = msg("UPDATE", part1);

var part2 = match("MESSAGE#1:STM:01", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} [ALERT:%{id}] Server %{daddr}:%{dport->} is disabled by out of band monitor ( new mode out_of_service_all ) Reason:%{result}", processor_chain([
	setc("eventcategory","1603000000"),
	setc("event_description","STM: LB Server disabled by out of band monitor"),
]));

var msg2 = msg("STM:01", part2);

var part3 = match("MESSAGE#2:STM:02", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} Server %{saddr->} is created.", processor_chain([
	dup3,
	setc("event_description","STM: LB Server created."),
]));

var msg3 = msg("STM:02", part3);

var part4 = match("MESSAGE#3:STM:03", "nwparser.payload", "STM: SSKey-%{fld1->} %{fld2->} Cookie Encryption Key has already expired", processor_chain([
	setc("eventcategory","1613030100"),
	setc("event_description","STM: SSKEY Cookie Encryption Key has already expired."),
]));

var msg4 = msg("STM:03", part4);

var part5 = match("MESSAGE#4:STM:04", "nwparser.payload", "STM: FAILOVE-%{fld1->} %{fld2->} Module CookieKey registered with Stateful Failover module.", processor_chain([
	dup4,
	setc("event_description","STM:FAILOVE Module CookieKey registered with Stateful Failover module."),
]));

var msg5 = msg("STM:04", part5);

var part6 = match("MESSAGE#5:STM:05", "nwparser.payload", "STM: FEHCMON-%{fld1->} %{fld2->} FEHC Monitor Module initialized.", processor_chain([
	dup3,
	setc("event_description","STM:FECHMON FEHC Monitor Module initialized."),
]));

var msg6 = msg("STM:05", part6);

var part7 = match("MESSAGE#6:STM:06", "nwparser.payload", "STM: FAILOVE-%{fld1->} %{fld2->} Stateful Failover Module initialized.", processor_chain([
	dup3,
	setc("event_description","STM: FAILOVE Stateful Failover Module initialized."),
]));

var msg7 = msg("STM:06", part7);

var part8 = match("MESSAGE#7:STM:07", "nwparser.payload", "STM: SERVICE-%{fld1->} %{fld3->} [%{fld2}] New Service (ID %{fld4}) Created at %{saddr}:%{sport}", processor_chain([
	dup3,
	setc("event_description","STM: SERVICE New Service created."),
]));

var msg8 = msg("STM:07", part8);

var part9 = match("MESSAGE#8:STM:08", "nwparser.payload", "STM: SSL-%{fld1->} %{fld2->} Ssl Initialization", processor_chain([
	dup4,
	setc("event_description","STM: SSL Initialization."),
]));

var msg9 = msg("STM:08", part9);

var part10 = match("MESSAGE#9:STM:09", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} LookupServerCtx = %{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB-LookupServerCtx."),
]));

var msg10 = msg("STM:09", part10);

var part11 = match("MESSAGE#10:STM:10", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} ParamProtectionClonePatterns: Old:%{change_old}, New:%{change_new}, PatternsNode:%{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps ParamProtectionClonePatterns values changed."),
]));

var msg11 = msg("STM:10", part11);

var part12 = match("MESSAGE#11:STM:11", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} %{obj_name->} SapCtx %{fld3}, SapId %{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps SapCtx log."),
]));

var msg12 = msg("STM:11", part12);

var part13 = match("MESSAGE#12:STM:12", "nwparser.payload", "STM: CACHE-%{fld1->} %{fld2->} %{obj_name->} SapCtx %{fld3}, SapId %{fld4}, Return Code %{result}", processor_chain([
	dup3,
	setc("event_description","STM: CACHE SapCtx log."),
]));

var msg13 = msg("STM:12", part13);

var part14 = match("MESSAGE#13:STM:13", "nwparser.payload", "STM: FTPSVC-%{fld1->} %{fld2->} Ftp proxy initialized %{info}", processor_chain([
	dup3,
	setc("event_description","STM: FTPSVC Ftp proxy initialized."),
]));

var msg14 = msg("STM:13", part14);

var part15 = match("MESSAGE#14:STM:14", "nwparser.payload", "STM: STM-%{fld1->} %{fld2->} Secure Traffic Manager Initialization complete: %{info}", processor_chain([
	dup3,
	setc("event_description","STM: STM Secure Traffic Manager Initialization complete."),
]));

var msg15 = msg("STM:14", part15);

var part16 = match("MESSAGE#15:STM:15", "nwparser.payload", "STM: COOKIE-%{fld1->} %{fld2->} %{obj_name->} = %{info}", processor_chain([
	dup3,
	setc("event_description","STM: COOKIE Cookie parameters set."),
]));

var msg16 = msg("STM:15", part16);

var part17 = match("MESSAGE#16:STM:16", "nwparser.payload", "STM: WebLog-%{fld1->} %{fld2->} %{obj_name}: SapCtx=%{fld3},SapId=%{fld4}, %{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: WebLog Set Sap variable."),
]));

var msg17 = msg("STM:16", part17);

var part18 = match("MESSAGE#17:STM:17", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} AddIpsPatternGroup SapCtx : %{fld3}, grp_id : %{fld4}, type : %{fld5->} grp: %{info}", processor_chain([
	dup3,
	setc("event_description","STM: aps Set AddIpsPatternGroup."),
]));

var msg18 = msg("STM:17", part18);

var part19 = match("MESSAGE#18:STM:18", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} AddPCInfoKeyWordMeta: Info:%{fld3}, Table:%{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps AddPCInfoKeyWordMeta."),
]));

var msg19 = msg("STM:18", part19);

var part20 = match("MESSAGE#19:STM:19", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} AddParamClass: %{fld3}: KeyWords:%{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps AddParamClass."),
]));

var msg20 = msg("STM:19", part20);

var part21 = match("MESSAGE#20:STM:20", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetParamClassPatternsAndDFA: Ctx:%{fld3}, type:%{fld4}, dfaId %{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: aps AddParamClassPatternsAndDFA."),
]));

var msg21 = msg("STM:20", part21);

var part22 = match("MESSAGE#21:STM:21", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} ParamClassClonePatternsInfo: Old:%{fld3}, New:%{fld4}, PatternsNode:%{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: aps AddParamClassClonePatternsInfo."),
]));

var msg22 = msg("STM:21", part22);

var part23 = match("MESSAGE#22:STM:22", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsLogIntrusionOn SapCtx %{fld3}, Return Code %{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps SetIpsLogIntrusionOn."),
]));

var msg23 = msg("STM:22", part23);

var part24 = match("MESSAGE#23:STM:23", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} AddIpsCloakFilterRespHeader [%{fld3}] Ret %{fld4}, SapCtx %{fld5}, sapId %{fld6}", processor_chain([
	dup3,
	setc("event_description","STM: aps AddIpsCloakFilterRespHeader."),
]));

var msg24 = msg("STM:23", part24);

var part25 = match("MESSAGE#24:STM:24", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsTheftPolicy SapCtx %{fld3}, Policy %{fld4}, Return %{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: aps SetIpsTheftPolicy."),
]));

var msg25 = msg("STM:24", part25);

var part26 = match("MESSAGE#25:STM:25", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsTheftPolicyDfa SapCtx %{fld3}, Policy %{fld4}, mode %{fld5}, bytes %{fld6}, Return %{fld7}", processor_chain([
	dup3,
	setc("event_description","STM: aps SetIpsTheftPolicyDfa."),
]));

var msg26 = msg("STM:25", part26);

var part27 = match("MESSAGE#26:STM:26", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsLimitPolicy Return Code %{fld3}", processor_chain([
	dup3,
	dup5,
]));

var msg27 = msg("STM:26", part27);

var part28 = match("MESSAGE#27:STM:27", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} CreateRC: RC Add policy Success", processor_chain([
	dup3,
	setc("event_description","STM: aps CreateRC: RC Add policy Success."),
]));

var msg28 = msg("STM:27", part28);

var part29 = match("MESSAGE#28:STM:28", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} SetSap%{info}=%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB Set Sap command."),
]));

var msg29 = msg("STM:28", part29);

var part30 = match("MESSAGE#29:STM:29", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} SetServer%{info}=%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB Set Server command."),
]));

var msg30 = msg("STM:29", part30);

var part31 = match("MESSAGE#30:STM:30", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} AddServer%{info}=%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB Add Server command."),
]));

var msg31 = msg("STM:30", part31);

var part32 = match("MESSAGE#31:STM:31", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} CreateServer =%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB Create Server command."),
]));

var msg32 = msg("STM:31", part32);

var part33 = match("MESSAGE#32:STM:32", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} EnableServer =%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB Enable Server command."),
]));

var msg33 = msg("STM:32", part33);

var part34 = match("MESSAGE#33:STM:33", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} ActiveServerOutOfBandMonitorAttr =%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB ActiveServerOutOfBandMonitorAttr command."),
]));

var msg34 = msg("STM:33", part34);

var part35 = match("MESSAGE#34:STM:34", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} BindServerToSap =%{fld3}", processor_chain([
	dup3,
	setc("event_description","STM: LB BindServerToSap command."),
]));

var msg35 = msg("STM:34", part35);

var part36 = match("MESSAGE#35:STM:35", "nwparser.payload", "STM: LB-%{fld1->} %{fld2->} [ALERT:%{fld3}] Server %{saddr}:%{sport->} is enabled by out of band monitor. Reason:out of band monitor", processor_chain([
	dup3,
	setc("event_description","STM: LB Server is enabled by out of band monitor Reason out of band monitor"),
]));

var msg36 = msg("STM:35", part36);

var part37 = match("MESSAGE#36:STM:36", "nwparser.payload", "STM: SERVICE-%{fld1->} %{fld2->} [%{saddr}:%{sport}] Service Started %{fld3}:%{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: SERVICE Server service started command."),
]));

var msg37 = msg("STM:36", part37);

var part38 = match("MESSAGE#37:STM:37", "nwparser.payload", "STM: RespPage-%{fld1->} %{fld2->} CreateRP: Response Page %{fld3->} created successfully", processor_chain([
	dup3,
	setc("event_description","STM: RespPage Response Page created successfully."),
]));

var msg38 = msg("STM:37", part38);

var part39 = match("MESSAGE#38:STM:38", "nwparser.payload", "STM: WATRewr-%{fld1->} %{fld2->} AddWATReqRewriteRule AclName [%{fld3}] Ret %{fld4->} SapCtx %{fld5}, SapId %{fld6}", processor_chain([
	dup3,
	setc("event_description","STM: AddWATReqRewriteRule AclName."),
]));

var msg39 = msg("STM:38", part39);

var part40 = match("MESSAGE#39:STM:39", "nwparser.payload", "STM: WATRewr-%{fld1->} %{fld2->} SetWATReqRewriteRuleNameWithKe AclName [%{fld3}] Ret %{fld4->} SapCtx %{fld5}, SapId %{fld6}", processor_chain([
	dup3,
	setc("event_description","STM: SetWATReqRewriteRuleNameWithKe AclName."),
]));

var msg40 = msg("STM:39", part40);

var part41 = match("MESSAGE#40:STM:40", "nwparser.payload", "STM: WATRewr-%{fld1->} %{fld2->} SetWATReqRewritePolicyOn - %{fld6->} Ret %{fld3->} SapCtx %{fld4}, SapId %{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: SetWATReqRewritePolicyOn."),
]));

var msg41 = msg("STM:40", part41);

var part42 = match("MESSAGE#41:STM:41", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsOn SapCtx %{fld3}, Return Code %{fld4}", processor_chain([
	dup3,
	setc("event_description","STM: aps SetIpsOn."),
]));

var msg42 = msg("STM:41", part42);

var part43 = match("MESSAGE#42:STM:42", "nwparser.payload", "STM: aps-%{fld1->} %{fld2->} SetIpsLimitPolicyOn Return Code %{fld3}", processor_chain([
	dup3,
	dup5,
]));

var msg43 = msg("STM:42", part43);

var part44 = match("MESSAGE#43:STM:43", "nwparser.payload", "STM: WATRewr-%{fld1->} %{fld2->} SetWATRespRewritePolicyOn - %{fld6->} Ret %{fld3->} SapCtx %{fld4}, SapId %{fld5}", processor_chain([
	dup3,
	setc("event_description","STM: SetWATRespRewritePolicyOn."),
]));

var msg44 = msg("STM:43", part44);

var select2 = linear_select([
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
]);

var part45 = match("MESSAGE#44:STM_WRAPPER:01", "nwparser.payload", "STM_WRAPPER: command(--digest) execution status = %{info}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: command execution status."),
]));

var msg45 = msg("STM_WRAPPER:01", part45);

var part46 = match("MESSAGE#45:STM_WRAPPER:02", "nwparser.payload", "STM_WRAPPER: [ALERT:%{fld1}] Configuration size is %{fld2->} which exceeds the %{fld3->} safe limit. Please check your configuration.", processor_chain([
	dup6,
	setc("event_description","STM_WRAPPER: ALERT Configuration size exceeds the safe memory limit."),
]));

var msg46 = msg("STM_WRAPPER:02", part46);

var part47 = match("MESSAGE#46:STM_WRAPPER:03", "nwparser.payload", "STM_WRAPPER: Committing UI configuration.%{}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: Committing UI configuration."),
]));

var msg47 = msg("STM_WRAPPER:03", part47);

var part48 = match("MESSAGE#47:STM_WRAPPER:04", "nwparser.payload", "STM_WRAPPER: Successfully stopped STM.%{}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: Successfully stopped STM."),
]));

var msg48 = msg("STM_WRAPPER:04", part48);

var part49 = match("MESSAGE#48:STM_WRAPPER:05", "nwparser.payload", "STM_WRAPPER: Successfully initialized STM.%{}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: Successfully initialized STM."),
]));

var msg49 = msg("STM_WRAPPER:05", part49);

var part50 = match("MESSAGE#49:STM_WRAPPER:06", "nwparser.payload", "STM_WRAPPER: Initializing STM.%{}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: Initializing STM."),
]));

var msg50 = msg("STM_WRAPPER:06", part50);

var part51 = match("MESSAGE#50:STM_WRAPPER:07", "nwparser.payload", "STM_WRAPPER: Rolling back the current database transaction. Configuration digest failed.%{}", processor_chain([
	dup3,
	setc("event_description","STM_WRAPPER: Rolling back the current database transaction. Configuration digest failed."),
]));

var msg51 = msg("STM_WRAPPER:07", part51);

var select3 = linear_select([
	msg45,
	msg46,
	msg47,
	msg48,
	msg49,
	msg50,
	msg51,
]);

var part52 = match("MESSAGE#51:CONFIG_AGENT:01", "nwparser.payload", "CONFIG_AGENT: %{fld1->} RPC Name =%{fld2}, RPC Result: %{fld3}", processor_chain([
	dup3,
	setc("event_description","CONFIG_AGENT: RPC information."),
]));

var msg52 = msg("CONFIG_AGENT:01", part52);

var part53 = match("MESSAGE#52:CONFIG_AGENT:02", "nwparser.payload", "CONFIG_AGENT: %{fld1->} %{fld2->} Received put-tree command", processor_chain([
	dup3,
	setc("event_description","CONFIG_AGENT:Received put-tree command."),
]));

var msg53 = msg("CONFIG_AGENT:02", part53);

var part54 = match("MESSAGE#53:CONFIG_AGENT:03", "nwparser.payload", "CONFIG_AGENT: %{fld1->} %{fld2->} It is recommended to configure cookie_encryption_key_expiry atleast 7 days ahead of current time., %{fld3}", processor_chain([
	dup4,
	setc("event_description","It is recommended to configure cookie_encryption_key_expiry atleast 7 days ahead of current time."),
]));

var msg54 = msg("CONFIG_AGENT:03", part54);

var part55 = match("MESSAGE#54:CONFIG_AGENT:04", "nwparser.payload", "CONFIG_AGENT: %{fld1->} Initiating config_agent database commit phase.", processor_chain([
	dup3,
	setc("event_description","CONFIG_AGENT:Initiating config_agent database commit phase."),
]));

var msg55 = msg("CONFIG_AGENT:04", part55);

var part56 = match("MESSAGE#55:CONFIG_AGENT:05", "nwparser.payload", "CONFIG_AGENT: %{fld1->} %{fld2->} Update succeeded", processor_chain([
	dup3,
	setc("event_description","CONFIG_AGENT:Update succeded."),
]));

var msg56 = msg("CONFIG_AGENT:05", part56);

var part57 = match("MESSAGE#56:CONFIG_AGENT:06", "nwparser.payload", "CONFIG_AGENT: %{fld1->} %{fld2->} No rules, %{fld3}", processor_chain([
	dup3,
	setc("event_description","CONFIG_AGENT:No rules."),
]));

var msg57 = msg("CONFIG_AGENT:06", part57);

var select4 = linear_select([
	msg52,
	msg53,
	msg54,
	msg55,
	msg56,
	msg57,
]);

var part58 = match("MESSAGE#57:PROCMON:01", "nwparser.payload", "PROCMON: Started monitoring%{}", processor_chain([
	dup3,
	setc("event_description","PROCMON: Started monitoring"),
]));

var msg58 = msg("PROCMON:01", part58);

var part59 = match("MESSAGE#58:PROCMON:02", "nwparser.payload", "PROCMON: number of stm worker threads is%{info}", processor_chain([
	dup3,
	setc("event_description","PROCMON: number of stm worker threads"),
]));

var msg59 = msg("PROCMON:02", part59);

var part60 = match("MESSAGE#59:PROCMON:03", "nwparser.payload", "PROCMON: Monitoring links: %{interface}", processor_chain([
	dup3,
	setc("event_description","PROCMON: Monitoring links."),
]));

var msg60 = msg("PROCMON:03", part60);

var part61 = match("MESSAGE#60:PROCMON:04", "nwparser.payload", "PROCMON: [ALERT:%{fld1}] %{interface}: link is up", processor_chain([
	dup3,
	setc("event_description","PROCMON:Link is up."),
]));

var msg61 = msg("PROCMON:04", part61);

var part62 = match("MESSAGE#61:PROCMON:05", "nwparser.payload", "PROCMON: [ALERT:%{fld1}] Firmware storage exceeds %{info}", processor_chain([
	setc("eventcategory","1607000000"),
	setc("event_description","PROCMON:Firmware storage exceeding."),
]));

var msg62 = msg("PROCMON:05", part62);

var part63 = match("MESSAGE#62:PROCMON:06", "nwparser.payload", "PROCMON: [ALERT:%{fld1}] One of the RAID arrays is degrading.", processor_chain([
	dup6,
	setc("event_description","PROCMON:One of the RAID arrays is degrading."),
]));

var msg63 = msg("PROCMON:06", part63);

var select5 = linear_select([
	msg58,
	msg59,
	msg60,
	msg61,
	msg62,
	msg63,
]);

var part64 = match("MESSAGE#63:BYPASS:01", "nwparser.payload", "BYPASS: State set to normal: starting heartbeat.%{}", processor_chain([
	dup3,
	setc("event_description","BYPASS: State set to normal: starting heartbeat."),
]));

var msg64 = msg("BYPASS:01", part64);

var part65 = match("MESSAGE#64:BYPASS:02", "nwparser.payload", "BYPASS: Mode change: %{fld1},%{fld2}", processor_chain([
	dup3,
	setc("event_description","Mode change."),
]));

var msg65 = msg("BYPASS:02", part65);

var part66 = match("MESSAGE#65:BYPASS:03", "nwparser.payload", "BYPASS: Mode set to BYPASS (%{fld2}).", processor_chain([
	dup3,
	setc("event_description"," Mode set to BYPASS."),
]));

var msg66 = msg("BYPASS:03", part66);

var part67 = match("MESSAGE#66:BYPASS:04", "nwparser.payload", "BYPASS: Mode set to never bypass.%{}", processor_chain([
	dup3,
	setc("event_description"," Mode set to never BYPASS."),
]));

var msg67 = msg("BYPASS:04", part67);

var select6 = linear_select([
	msg64,
	msg65,
	msg66,
	msg67,
]);

var part68 = match("MESSAGE#67:INSTALL:01", "nwparser.payload", "INSTALL: Migrating configuration from %{fld2->} to %{fld3}", processor_chain([
	dup3,
	setc("event_description"," INSTALL: migrating configuration."),
]));

var msg68 = msg("INSTALL:01", part68);

var part69 = match("MESSAGE#68:INSTALL:02", "nwparser.payload", "INSTALL: Loading the snapshot for %{fld2->} release.", processor_chain([
	dup3,
	setc("event_description"," INSTALL: Loading snapshot from previous version."),
]));

var msg69 = msg("INSTALL:02", part69);

var select7 = linear_select([
	msg68,
	msg69,
]);

var part70 = match("MESSAGE#69:eventmgr:01", "nwparser.payload", "eventmgr: Forwarding log messages to syslog host #%{fld3}, address=%{hostip}", processor_chain([
	dup3,
	setc("event_description","eventmgr: Forwarding log messages to syslog host"),
]));

var msg70 = msg("eventmgr:01", part70);

var part71 = match("MESSAGE#70:eventmgr:02", "nwparser.payload", "eventmgr: Event manager startup succeeded.%{}", processor_chain([
	dup3,
	setc("event_description","eventmgr: Event manager startup succeeded."),
]));

var msg71 = msg("eventmgr:02", part71);

var select8 = linear_select([
	msg70,
	msg71,
]);

var part72 = match("MESSAGE#71:CONFIG", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup7,
	setc("event_description"," Configuration changes made."),
	dup8,
]));

var msg72 = msg("CONFIG", part72);

var part73 = match("MESSAGE#72:LOGIN", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	setc("eventcategory","1401060000"),
	setc("event_description"," Login."),
	dup8,
]));

var msg73 = msg("LOGIN", part73);

var part74 = match("MESSAGE#73:SESSION_TIMEOUT", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup9,
	setc("event_description"," Session timeout."),
	dup8,
]));

var msg74 = msg("SESSION_TIMEOUT", part74);

var part75 = match("MESSAGE#74:LOGOUT", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup9,
	setc("ec_subject","User"),
	setc("ec_activity","Logoff"),
	setc("ec_theme","Authentication"),
	setc("ec_outcome","Success"),
	setc("event_description"," Logout."),
	dup8,
]));

var msg75 = msg("LOGOUT", part75);

var part76 = match("MESSAGE#75:UNSUCCESSFUL_LOGIN", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	setc("eventcategory","1401030000"),
	setc("event_description"," Unsuccessful login."),
	dup8,
]));

var msg76 = msg("UNSUCCESSFUL_LOGIN", part76);

var part77 = match("MESSAGE#76:TRANSPARENT_MODE", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," Operating in Transport Mode"),
	dup8,
]));

var msg77 = msg("TRANSPARENT_MODE", part77);

var part78 = match("MESSAGE#77:SUPPORT_TUNNEL_OPEN", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," Support Tunnel Opened"),
	dup8,
]));

var msg78 = msg("SUPPORT_TUNNEL_OPEN", part78);

var part79 = match("MESSAGE#78:FIRMWARE_UPDATE", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," Firmware Update"),
	dup8,
]));

var msg79 = msg("FIRMWARE_UPDATE", part79);

var part80 = match("MESSAGE#79:FIRMWARE_REVERT", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," Firmware Revert."),
	dup8,
]));

var msg80 = msg("FIRMWARE_REVERT", part80);

var part81 = match("MESSAGE#80:REBOOT", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," System Reboot."),
	dup8,
]));

var msg81 = msg("REBOOT", part81);

var part82 = match("MESSAGE#81:ROLLBACK", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup10,
	setc("event_description"," System ROLLBACK."),
	dup8,
]));

var msg82 = msg("ROLLBACK", part82);

var part83 = match("MESSAGE#82:HEADER_COUNT_EXCEEDED:01", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} \"[%{result}]\" %{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}", processor_chain([
	dup11,
	dup8,
]));

var msg83 = msg("HEADER_COUNT_EXCEEDED:01", part83);

var part84 = match("MESSAGE#83:HEADER_COUNT_EXCEEDED:02", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} [%{result}] %{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}", processor_chain([
	dup11,
	dup8,
]));

var msg84 = msg("HEADER_COUNT_EXCEEDED:02", part84);

var msg85 = msg("HEADER_COUNT_EXCEEDED", dup26);

var select9 = linear_select([
	msg83,
	msg84,
	msg85,
]);

var msg86 = msg("CROSS_SITE_SCRIPTING_IN_PARAM:01", dup27);

var msg87 = msg("CROSS_SITE_SCRIPTING_IN_PARAM", dup26);

var select10 = linear_select([
	msg86,
	msg87,
]);

var msg88 = msg("SQL_INJECTION_IN_URL:01", dup27);

var msg89 = msg("SQL_INJECTION_IN_URL", dup26);

var select11 = linear_select([
	msg88,
	msg89,
]);

var msg90 = msg("OS_CMD_INJECTION_IN_URL:01", dup27);

var msg91 = msg("OS_CMD_INJECTION_IN_URL", dup26);

var select12 = linear_select([
	msg90,
	msg91,
]);

var msg92 = msg("TILDE_IN_URL:01", dup27);

var msg93 = msg("TILDE_IN_URL", dup26);

var select13 = linear_select([
	msg92,
	msg93,
]);

var msg94 = msg("SQL_INJECTION_IN_PARAM:01", dup27);

var msg95 = msg("SQL_INJECTION_IN_PARAM", dup26);

var select14 = linear_select([
	msg94,
	msg95,
]);

var part85 = match("MESSAGE#95:OS_CMD_INJECTION_IN_PARAM:01/1_1", "nwparser.p0", "[%{result->} \"] %{p0}");

var select15 = linear_select([
	dup13,
	part85,
	dup14,
]);

var all1 = all_match({
	processors: [
		dup12,
		select15,
		dup16,
	],
	on_success: processor_chain([
		dup11,
		dup8,
	]),
});

var msg96 = msg("OS_CMD_INJECTION_IN_PARAM:01", all1);

var msg97 = msg("OS_CMD_INJECTION_IN_PARAM", dup26);

var select16 = linear_select([
	msg96,
	msg97,
]);

var msg98 = msg("METHOD_NOT_ALLOWED:01", dup27);

var msg99 = msg("METHOD_NOT_ALLOWED", dup26);

var select17 = linear_select([
	msg98,
	msg99,
]);

var msg100 = msg("ERROR_RESPONSE_SUPPRESSED:01", dup27);

var msg101 = msg("ERROR_RESPONSE_SUPPRESSED", dup26);

var select18 = linear_select([
	msg100,
	msg101,
]);

var msg102 = msg("DENY_ACL_MATCHED:01", dup27);

var msg103 = msg("DENY_ACL_MATCHED", dup26);

var select19 = linear_select([
	msg102,
	msg103,
]);

var msg104 = msg("NO_DOMAIN_MATCH_IN_PROFILE", dup24);

var msg105 = msg("NO_URL_PROFILE_MATCH", dup24);

var msg106 = msg("UNRECOGNIZED_COOKIE", dup24);

var msg107 = msg("HEADER_VALUE_LENGTH_EXCEEDED", dup24);

var msg108 = msg("UNKNOWN_CONTENT_TYPE", dup24);

var msg109 = msg("INVALID_URL_ENCODING", dup24);

var msg110 = msg("INVALID_URL_CHARSET", dup24);

var msg111 = msg("CROSS_SITE_SCRIPTING_IN_URL:01", dup27);

var msg112 = msg("CROSS_SITE_SCRIPTING_IN_URL", dup26);

var select20 = linear_select([
	msg111,
	msg112,
]);

var msg113 = msg("SLASH_DOT_IN_URL:01", dup27);

var msg114 = msg("SLASH_DOT_IN_URL", dup26);

var select21 = linear_select([
	msg113,
	msg114,
]);

var part86 = match("MESSAGE#114:SYS", "nwparser.payload", "%{fld9->} %{fld10->} %{timezone->} %{fld11->} %{category->} %{event_type->} %{severity->} %{operation_id->} %{event_description}", processor_chain([
	dup3,
	date_time({
		dest: "event_time",
		args: ["hfld9","hfld10"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
		],
	}),
]));

var msg115 = msg("SYS", part86);

var part87 = match("MESSAGE#115:BARRACUDAWAF", "nwparser.payload", "Log=%{event_log->} Severity=%{severity->} Protocol=%{protocol->} SourceIP=%{saddr->} SourcePort=%{sport->} DestIP=%{daddr->} DestPort=%{dport->} Action=%{action->} AdminName=%{administrator->} Details=%{info}", processor_chain([
	dup17,
	date_time({
		dest: "event_time",
		args: ["hfld1","hfld2"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dO],
		],
	}),
]));

var msg116 = msg("BARRACUDAWAF", part87);

var part88 = match("MESSAGE#116:Audit_Logs", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} AUDIT %{operation_id->} %{administrator->} %{action->} %{content_type->} %{hostip->} %{fld8->} %{info->} %{obj_type->} %{fld11->} %{obj_name->} \"%{change_old}\" \"%{change_new}\"", processor_chain([
	dup7,
	dup8,
	setc("category","AUDIT"),
	setc("vid","Audit_Logs"),
]));

var msg117 = msg("Audit_Logs", part88);

var part89 = match("MESSAGE#117:WF", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} WF %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} [%{result}] %{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}", processor_chain([
	dup17,
	dup8,
	setc("category","WF"),
	setc("vid","WF"),
]));

var msg118 = msg("WF", part89);

var part90 = match("MESSAGE#118:TR_Logs:01/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} TR %{operation_id->} %{protocol->} %{web_method->} %{saddr->} %{sport->} %{daddr->} %{dport->} %{url->} %{cert_username->} %{logon_id->} %{web_host->} %{web_referer->} %{resultcode->} %{sbytes->} %{rbytes->} \"-\" \"-\" \"%{user_agent}\" %{stransaddr->} %{p0}");

var all2 = all_match({
	processors: [
		part90,
		dup25,
	],
	on_success: processor_chain([
		dup17,
		dup20,
		dup8,
		dup21,
		dup22,
	]),
});

var msg119 = msg("TR_Logs:01", all2);

var part91 = match("MESSAGE#119:TR_Logs:02/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} TR %{operation_id->} %{protocol->} %{web_method->} %{saddr->} %{sport->} %{daddr->} %{dport->} %{url->} %{cert_username->} %{logon_id->} %{web_host->} %{web_referer->} %{resultcode->} %{sbytes->} %{rbytes->} %{web_query->} \"-\" \"%{user_agent}\" %{stransaddr->} %{p0}");

var all3 = all_match({
	processors: [
		part91,
		dup25,
	],
	on_success: processor_chain([
		dup17,
		dup20,
		dup8,
		dup21,
		dup22,
	]),
});

var msg120 = msg("TR_Logs:02", all3);

var part92 = match("MESSAGE#120:TR_Logs:03/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} TR %{operation_id->} %{protocol->} %{web_method->} %{saddr->} %{sport->} %{daddr->} %{dport->} %{url->} %{cert_username->} %{logon_id->} %{web_host->} %{web_referer->} %{resultcode->} %{sbytes->} %{rbytes->} \"-\" %{web_cookie->} \"%{user_agent}\" %{stransaddr->} %{p0}");

var all4 = all_match({
	processors: [
		part92,
		dup25,
	],
	on_success: processor_chain([
		dup17,
		dup20,
		dup8,
		dup21,
		dup22,
	]),
});

var msg121 = msg("TR_Logs:03", all4);

var part93 = match("MESSAGE#121:TR_Logs/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} TR %{operation_id->} %{protocol->} %{web_method->} %{saddr->} %{sport->} %{daddr->} %{dport->} %{url->} %{cert_username->} %{logon_id->} %{web_host->} %{web_referer->} %{resultcode->} %{sbytes->} %{rbytes->} %{web_query->} %{web_cookie->} \"%{user_agent}\" %{stransaddr->} %{p0}");

var all5 = all_match({
	processors: [
		part93,
		dup25,
	],
	on_success: processor_chain([
		dup17,
		dup20,
		dup8,
		dup21,
		dup22,
	]),
});

var msg122 = msg("TR_Logs", all5);

var select22 = linear_select([
	msg117,
	msg118,
	msg119,
	msg120,
	msg121,
	msg122,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"BARRACUDAWAF": msg116,
		"BARRACUDA_GENRIC": select22,
		"BYPASS": select6,
		"CONFIG": msg72,
		"CONFIG_AGENT": select4,
		"CROSS_SITE_SCRIPTING_IN_PARAM": select10,
		"CROSS_SITE_SCRIPTING_IN_URL": select20,
		"DENY_ACL_MATCHED": select19,
		"ERROR_RESPONSE_SUPPRESSED": select18,
		"FIRMWARE_REVERT": msg80,
		"FIRMWARE_UPDATE": msg79,
		"HEADER_COUNT_EXCEEDED": select9,
		"HEADER_VALUE_LENGTH_EXCEEDED": msg107,
		"INSTALL": select7,
		"INVALID_URL_CHARSET": msg110,
		"INVALID_URL_ENCODING": msg109,
		"LOGIN": msg73,
		"LOGOUT": msg75,
		"METHOD_NOT_ALLOWED": select17,
		"NO_DOMAIN_MATCH_IN_PROFILE": msg104,
		"NO_URL_PROFILE_MATCH": msg105,
		"OS_CMD_INJECTION_IN_PARAM": select16,
		"OS_CMD_INJECTION_IN_URL": select12,
		"PROCMON": select5,
		"REBOOT": msg81,
		"ROLLBACK": msg82,
		"SESSION_TIMEOUT": msg74,
		"SLASH_DOT_IN_URL": select21,
		"SQL_INJECTION_IN_PARAM": select14,
		"SQL_INJECTION_IN_URL": select11,
		"STM": select2,
		"STM_WRAPPER": select3,
		"SUPPORT_TUNNEL_OPEN": msg78,
		"SYS": msg115,
		"TILDE_IN_URL": select13,
		"TRANSPARENT_MODE": msg77,
		"UNKNOWN_CONTENT_TYPE": msg108,
		"UNRECOGNIZED_COOKIE": msg106,
		"UNSUCCESSFUL_LOGIN": msg76,
		"UPDATE": msg1,
		"eventmgr": select8,
	}),
]);

var part94 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/0", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} %{p0}");

var part95 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/1_0", "nwparser.p0", "\"[%{result}]\" %{p0}");

var part96 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/1_1", "nwparser.p0", "[%{result}] %{p0}");

var part97 = match("MESSAGE#84:HEADER_COUNT_EXCEEDED/2", "nwparser.p0", "%{web_method->} %{url->} %{protocol->} - %{stransaddr->} %{stransport->} %{web_referer}");

var part98 = match("MESSAGE#85:CROSS_SITE_SCRIPTING_IN_PARAM:01/2", "nwparser.p0", "%{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}");

var part99 = match("MESSAGE#118:TR_Logs:01/1_0", "nwparser.p0", "%{stransport->} %{content_type}");

var part100 = match_copy("MESSAGE#118:TR_Logs:01/1_1", "nwparser.p0", "stransport");

var select23 = linear_select([
	dup13,
	dup14,
]);

var part101 = match("MESSAGE#103:NO_DOMAIN_MATCH_IN_PROFILE", "nwparser.payload", "%{fld88->} %{fld89->} %{timezone->} %{category->} %{operation_id->} %{severity->} %{event_type->} %{saddr->} %{sport->} %{rulename->} %{rule_group->} %{action->} %{context->} [%{result}] %{web_method->} %{url->} %{protocol->} \"%{user_agent}\" %{stransaddr->} %{stransport->} %{web_referer}", processor_chain([
	dup17,
	dup8,
]));

var select24 = linear_select([
	dup18,
	dup19,
]);

var all6 = all_match({
	processors: [
		dup12,
		dup23,
		dup15,
	],
	on_success: processor_chain([
		dup11,
		dup8,
	]),
});

var all7 = all_match({
	processors: [
		dup12,
		dup23,
		dup16,
	],
	on_success: processor_chain([
		dup11,
		dup8,
	]),
});
