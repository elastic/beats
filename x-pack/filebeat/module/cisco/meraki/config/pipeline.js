//  Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
//  or more contributor license agreements. Licensed under the Elastic License;
//  you may not use this file except in compliance with the Elastic License.
var tvm = {
	pair_separator: " ",
	kv_separator: "=",
	open_quote: "'",
	close_quote: "'",
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

var map_actionType = {
	keyvaluepairs: {
		"0": dup19,
		"1": constant("Deny"),
		"allow": dup19,
	},
};

var dup1 = match("HEADER#0:0003/0", "message", "%{hfld1->} %{hfld2}.%{hfld3->} %{p0}");

var dup2 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hfld4"),
		constant("_appliance "),
		field("p0"),
	],
});

var dup3 = call({
	dest: "nwparser.payload",
	fn: STRCAT,
	args: [
		field("hfld4"),
		constant(" "),
		field("p0"),
	],
});

var dup4 = match_copy("MESSAGE#0:flows/2_1", "nwparser.p0", "p0");

var dup5 = setc("eventcategory","1605020000");

var dup6 = setf("msg","$MSG");

var dup7 = setc("event_source","appliance");

var dup8 = setf("sensor","node");

var dup9 = date_time({
	dest: "event_time",
	args: ["hfld2"],
	fmts: [
		[dX],
	],
});

var dup10 = match_copy("MESSAGE#1:flows:01/1_2", "nwparser.p0", "");

var dup11 = match("MESSAGE#10:ids-alerts:01/1_0", "nwparser.p0", "dhost=%{dmacaddr->} direction=%{p0}");

var dup12 = match("MESSAGE#10:ids-alerts:01/1_1", "nwparser.p0", "shost=%{smacaddr->} direction=%{p0}");

var dup13 = match("MESSAGE#10:ids-alerts:01/2", "nwparser.p0", "%{direction->} protocol=%{protocol->} src=%{p0}");

var dup14 = match_copy("MESSAGE#10:ids-alerts:01/4", "nwparser.p0", "signame");

var dup15 = setc("eventcategory","1607000000");

var dup16 = setc("event_type","ids-alerts");

var dup17 = date_time({
	dest: "event_time",
	args: ["fld3"],
	fmts: [
		[dX],
	],
});

var dup18 = setc("event_type","security_event");

var dup19 = constant("Allow");

var dup20 = match("HEADER#0:0003/1_0", "nwparser.p0", "%{hfld4}_appliance %{p0}", processor_chain([
	dup2,
]));

var dup21 = match("HEADER#0:0003/1_1", "nwparser.p0", "%{hfld4->} %{p0}", processor_chain([
	dup3,
]));

var dup22 = linear_select([
	dup11,
	dup12,
]);

var dup23 = linear_select([
	dup20,
	dup21,
]);

var part1 = match("HEADER#0:0003/2", "nwparser.p0", "urls %{p0}");

var all1 = all_match({
	processors: [
		dup1,
		dup23,
		part1,
	],
	on_success: processor_chain([
		setc("header_id","0003"),
		setc("messageid","urls"),
	]),
});

var part2 = match("HEADER#1:0002/1_0", "nwparser.p0", "%{node}_appliance events %{p0}");

var part3 = match("HEADER#1:0002/1_1", "nwparser.p0", "%{node->} events %{p0}");

var select1 = linear_select([
	part2,
	part3,
]);

var part4 = match_copy("HEADER#1:0002/2", "nwparser.p0", "payload");

var all2 = all_match({
	processors: [
		dup1,
		select1,
		part4,
	],
	on_success: processor_chain([
		setc("header_id","0002"),
		setc("messageid","events"),
	]),
});

var part5 = match("HEADER#2:0001/2", "nwparser.p0", "%{messageid->} %{p0}");

var all3 = all_match({
	processors: [
		dup1,
		dup23,
		part5,
	],
	on_success: processor_chain([
		setc("header_id","0001"),
	]),
});

var part6 = match("HEADER#3:0005/1_0", "nwparser.p0", "%{hfld4}_appliance %{p0}");

var part7 = match("HEADER#3:0005/1_1", "nwparser.p0", "%{hfld4->} %{p0}");

var select2 = linear_select([
	part6,
	part7,
]);

var part8 = match("HEADER#3:0005/2", "nwparser.p0", "%{} %{hfld5->} %{hfld6->} %{messageid->} %{p0}", processor_chain([
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hfld6"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var all4 = all_match({
	processors: [
		dup1,
		select2,
		part8,
	],
	on_success: processor_chain([
		setc("header_id","0005"),
	]),
});

var hdr1 = match("HEADER#4:0004", "message", "%{hfld1->} %{hfld2}.%{hfld3->} %{hfld4}_%{space->} %{messageid->} %{payload}", processor_chain([
	setc("header_id","0004"),
]));

var select3 = linear_select([
	all1,
	all2,
	all3,
	all4,
	hdr1,
]);

var part9 = match("MESSAGE#0:flows/0_0", "nwparser.payload", "%{node}_appliance %{p0}");

var part10 = match("MESSAGE#0:flows/0_1", "nwparser.payload", "%{node->} %{p0}");

var select4 = linear_select([
	part9,
	part10,
]);

var part11 = match("MESSAGE#0:flows/1", "nwparser.p0", "flows src=%{saddr->} dst=%{daddr->} %{p0}");

var part12 = match("MESSAGE#0:flows/2_0", "nwparser.p0", "mac=%{dmacaddr->} %{p0}");

var select5 = linear_select([
	part12,
	dup4,
]);

var part13 = match("MESSAGE#0:flows/3", "nwparser.p0", "protocol=%{protocol->} %{p0}");

var part14 = match("MESSAGE#0:flows/4_0", "nwparser.p0", "sport=%{sport->} dport=%{dport->} %{p0}");

var part15 = match("MESSAGE#0:flows/4_1", "nwparser.p0", "type=%{event_type->} %{p0}");

var select6 = linear_select([
	part14,
	part15,
	dup4,
]);

var part16 = match("MESSAGE#0:flows/5", "nwparser.p0", "pattern: %{fld21->} %{info}");

var all5 = all_match({
	processors: [
		select4,
		part11,
		select5,
		part13,
		select6,
		part16,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		lookup({
			dest: "nwparser.action",
			map: map_actionType,
			key: field("fld21"),
		}),
		dup7,
		dup8,
		dup9,
	]),
});

var msg1 = msg("flows", all5);

var part17 = match("MESSAGE#1:flows:01/0", "nwparser.payload", "%{node->} flows %{action->} src=%{saddr->} dst=%{daddr->} mac=%{smacaddr->} protocol=%{protocol->} %{p0}");

var part18 = match("MESSAGE#1:flows:01/1_0", "nwparser.p0", "sport=%{sport->} dport=%{dport->} ");

var part19 = match("MESSAGE#1:flows:01/1_1", "nwparser.p0", "type=%{event_type->} ");

var select7 = linear_select([
	part18,
	part19,
	dup10,
]);

var all6 = all_match({
	processors: [
		part17,
		select7,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup8,
		dup9,
	]),
});

var msg2 = msg("flows:01", all6);

var part20 = match("MESSAGE#2:flows:02", "nwparser.payload", "%{node->} flows %{action}", processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
]));

var msg3 = msg("flows:02", part20);

var select8 = linear_select([
	msg1,
	msg2,
	msg3,
]);

var part21 = match("MESSAGE#3:urls/0_0", "nwparser.payload", "%{node}_appliance urls src=%{p0}");

var part22 = match("MESSAGE#3:urls/0_1", "nwparser.payload", "%{node->} urls src=%{p0}");

var part23 = match("MESSAGE#3:urls/0_2", "nwparser.payload", "src=%{p0}");

var select9 = linear_select([
	part21,
	part22,
	part23,
]);

var part24 = match("MESSAGE#3:urls/1", "nwparser.p0", "%{sport}:%{saddr->} dst=%{daddr}:%{dport->} mac=%{macaddr->} %{p0}");

var part25 = match("MESSAGE#3:urls/2_0", "nwparser.p0", "agent='%{user_agent}' request: %{p0}");

var part26 = match("MESSAGE#3:urls/2_1", "nwparser.p0", "agent=%{user_agent->} request: %{p0}");

var part27 = match("MESSAGE#3:urls/2_2", "nwparser.p0", "request: %{p0}");

var select10 = linear_select([
	part25,
	part26,
	part27,
]);

var part28 = match("MESSAGE#3:urls/3", "nwparser.p0", "%{} %{web_method}%{url}");

var all7 = all_match({
	processors: [
		select9,
		part24,
		select10,
		part28,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup8,
		dup9,
	]),
});

var msg4 = msg("urls", all7);

var part29 = match("MESSAGE#4:events/0", "nwparser.payload", "dhcp lease of ip %{saddr->} from server mac %{smacaddr->} for client mac %{p0}");

var part30 = match("MESSAGE#4:events/1_0", "nwparser.p0", "%{dmacaddr->} with hostname %{hostname->} from router %{p0}");

var part31 = match("MESSAGE#4:events/1_1", "nwparser.p0", "%{dmacaddr->} from router %{p0}");

var select11 = linear_select([
	part30,
	part31,
]);

var part32 = match("MESSAGE#4:events/2", "nwparser.p0", "%{hostip->} on subnet %{mask->} with dns %{dns_a_record}");

var all8 = all_match({
	processors: [
		part29,
		select11,
		part32,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		dup7,
		dup8,
		dup9,
	]),
});

var msg5 = msg("events", all8);

var part33 = match("MESSAGE#5:events:02/0", "nwparser.payload", "content_filtering_block url='%{url}' category0='%{category}' server='%{daddr}:%{dport}'%{p0}");

var part34 = match("MESSAGE#5:events:02/1_0", "nwparser.p0", " client_mac='%{dmacaddr}'");

var select12 = linear_select([
	part34,
	dup10,
]);

var all9 = all_match({
	processors: [
		part33,
		select12,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		setc("event_description","content_filtering_block"),
		dup8,
		dup9,
	]),
});

var msg6 = msg("events:02", all9);

var part35 = tagval("MESSAGE#6:events:01", "nwparser.payload", tvm, {
	"aid": "fld1",
	"arp_resp": "fld2",
	"arp_src": "fld3",
	"auth_neg_dur": "fld4",
	"auth_neg_failed": "fld5",
	"category0": "category",
	"channel": "fld6",
	"client_ip": "daddr",
	"client_mac": "dmacaddr",
	"connectivity": "fld28",
	"dhcp_ip": "fld23",
	"dhcp_lease_completed": "fld22",
	"dhcp_resp": "fld26",
	"dhcp_server": "fld24",
	"dhcp_server_mac": "fld25",
	"dns_req_rtt": "fld7",
	"dns_resp": "fld8",
	"dns_server": "fld9",
	"duration": "duration",
	"full_conn": "fld11",
	"http_resp": "fld21",
	"identity": "fld12",
	"instigator": "fld20",
	"ip_resp": "fld13",
	"ip_src": "saddr",
	"is_8021x": "fld15",
	"is_wpa": "fld16",
	"last_auth_ago": "fld17",
	"radio": "fld18",
	"reason": "fld19",
	"rssi": "dclass_ratio1",
	"server": "daddr",
	"type": "event_type",
	"url": "url",
	"vap": "fld22",
	"vpn_type": "fld27",
}, processor_chain([
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
]));

var msg7 = msg("events:01", part35);

var part36 = match("MESSAGE#7:events:03", "nwparser.payload", "IDS: %{info}", processor_chain([
	dup5,
	dup6,
	setc("event_description","events IDS"),
	dup8,
	dup9,
]));

var msg8 = msg("events:03", part36);

var part37 = match("MESSAGE#8:events:04/0", "nwparser.payload", "dhcp %{p0}");

var part38 = match("MESSAGE#8:events:04/1_0", "nwparser.p0", "no offers%{p0}");

var part39 = match("MESSAGE#8:events:04/1_1", "nwparser.p0", "release%{p0}");

var select13 = linear_select([
	part38,
	part39,
]);

var part40 = match("MESSAGE#8:events:04/2", "nwparser.p0", "%{}for mac %{macaddr}");

var all10 = all_match({
	processors: [
		part37,
		select13,
		part40,
	],
	on_success: processor_chain([
		dup5,
		dup6,
		setc("event_description","events DHCP"),
		dup8,
		dup9,
	]),
});

var msg9 = msg("events:04", all10);

var part41 = match("MESSAGE#9:events:05", "nwparser.payload", "MAC %{macaddr->} and MAC %{macaddr->} both claim IP: %{saddr}", processor_chain([
	dup5,
	dup6,
	setc("event_description"," events MAC"),
	dup8,
	dup9,
]));

var msg10 = msg("events:05", part41);

var select14 = linear_select([
	msg5,
	msg6,
	msg7,
	msg8,
	msg9,
	msg10,
]);

var part42 = match("MESSAGE#10:ids-alerts:01/0", "nwparser.payload", "%{node->} ids-alerts signature=%{fld1->} priority=%{fld2->} timestamp=%{fld3}.%{fld4->} %{p0}");

var part43 = match("MESSAGE#10:ids-alerts:01/3_0", "nwparser.p0", "%{saddr}:%{sport->} dst=%{daddr}:%{dport->} message: %{p0}");

var part44 = match("MESSAGE#10:ids-alerts:01/3_1", "nwparser.p0", "%{saddr->} dst=%{daddr->} message: %{p0}");

var select15 = linear_select([
	part43,
	part44,
]);

var all11 = all_match({
	processors: [
		part42,
		dup22,
		dup13,
		select15,
		dup14,
	],
	on_success: processor_chain([
		dup15,
		dup6,
		dup16,
		dup8,
		dup17,
	]),
});

var msg11 = msg("ids-alerts:01", all11);

var part45 = match("MESSAGE#11:ids-alerts:03", "nwparser.payload", "%{node->} ids-alerts signature=%{fld1->} priority=%{fld2->} timestamp=%{fld3}.%{fld4}direction=%{direction->} protocol=%{protocol->} src=%{saddr}:%{sport}", processor_chain([
	dup15,
	dup6,
	dup16,
	dup8,
	dup17,
]));

var msg12 = msg("ids-alerts:03", part45);

var part46 = match("MESSAGE#12:ids-alerts:02", "nwparser.payload", "%{node->} ids-alerts signature=%{fld1->} priority=%{fld2->} timestamp=%{fld3}.%{fld4}protocol=%{protocol->} src=%{saddr->} dst=%{daddr}message: %{signame}", processor_chain([
	dup15,
	dup6,
	dup16,
	dup8,
	dup17,
]));

var msg13 = msg("ids-alerts:02", part46);

var select16 = linear_select([
	msg11,
	msg12,
	msg13,
]);

var part47 = match("MESSAGE#13:security_event", "nwparser.payload", "%{node}security_event %{event_description->} url=%{url->} src=%{saddr}:%{sport->} dst=%{daddr}:%{dport->} mac=%{smacaddr->} name=%{fld10->} sha256=%{fld11->} disposition=%{disposition->} action=%{action}", processor_chain([
	dup5,
	dup6,
	dup18,
	dup8,
	dup9,
]));

var msg14 = msg("security_event", part47);

var part48 = match("MESSAGE#14:security_event:01/0", "nwparser.payload", "%{node->} security_event %{event_description->} signature=%{fld1->} priority=%{fld2->} timestamp=%{fld3}.%{fld4->} %{p0}");

var part49 = match("MESSAGE#14:security_event:01/3_0", "nwparser.p0", "%{saddr}:%{sport->} dst=%{daddr}:%{dport->} message:%{p0}");

var part50 = match("MESSAGE#14:security_event:01/3_1", "nwparser.p0", "%{saddr->} dst=%{daddr->} message:%{p0}");

var select17 = linear_select([
	part49,
	part50,
]);

var all12 = all_match({
	processors: [
		part48,
		dup22,
		dup13,
		select17,
		dup14,
	],
	on_success: processor_chain([
		dup15,
		dup6,
		dup18,
		dup8,
		dup17,
	]),
});

var msg15 = msg("security_event:01", all12);

var select18 = linear_select([
	msg14,
	msg15,
]);

var chain1 = processor_chain([
	select3,
	msgid_select({
		"events": select14,
		"flows": select8,
		"ids-alerts": select16,
		"security_event": select18,
		"urls": msg4,
	}),
]);

var hdr2 = match("HEADER#0:0003/0", "message", "%{hfld1->} %{hfld2}.%{hfld3->} %{p0}");

var part51 = match_copy("MESSAGE#0:flows/2_1", "nwparser.p0", "p0");

var part52 = match_copy("MESSAGE#1:flows:01/1_2", "nwparser.p0", "");

var part53 = match("MESSAGE#10:ids-alerts:01/1_0", "nwparser.p0", "dhost=%{dmacaddr->} direction=%{p0}");

var part54 = match("MESSAGE#10:ids-alerts:01/1_1", "nwparser.p0", "shost=%{smacaddr->} direction=%{p0}");

var part55 = match("MESSAGE#10:ids-alerts:01/2", "nwparser.p0", "%{direction->} protocol=%{protocol->} src=%{p0}");

var part56 = match_copy("MESSAGE#10:ids-alerts:01/4", "nwparser.p0", "signame");

var part57 = match("HEADER#0:0003/1_0", "nwparser.p0", "%{hfld4}_appliance %{p0}", processor_chain([
	dup2,
]));

var part58 = match("HEADER#0:0003/1_1", "nwparser.p0", "%{hfld4->} %{p0}", processor_chain([
	dup3,
]));

var select19 = linear_select([
	dup11,
	dup12,
]);

var select20 = linear_select([
	dup20,
	dup21,
]);
