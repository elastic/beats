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

var dup1 = setc("eventcategory","1204000000");

var dup2 = setc("ec_subject","NetworkComm");

var dup3 = setc("ec_activity","Request");

var dup4 = setc("ec_theme","ALM");

var dup5 = domain("web_ref_domain","web_referer");

var dup6 = root("web_root","web_referer");

var dup7 = query("web_ref_query","web_referer");

var dup8 = domain("web_domain","url");

var dup9 = domain("domain","url");

var dup10 = domain("web_host","url");

var dup11 = date_time({
	dest: "event_time",
	args: ["fld20"],
	fmts: [
		[dD,dc("/"),dB,dc("/"),dW,dc(":"),dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup12 = setf("msg","$MSG");

var dup13 = date_time({
	dest: "event_time",
	args: ["event_time_string"],
	fmts: [
		[dX],
	],
});

var dup14 = page("webpage","url");

var dup15 = match("MESSAGE#0:GET", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var dup16 = match("MESSAGE#19:GET:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));

var dup17 = match("MESSAGE#2:POST", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var dup18 = match("MESSAGE#21:POST:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));

var dup19 = match("MESSAGE#3:PUT", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var dup20 = match("MESSAGE#22:PUT:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));

var hdr1 = match("HEADER#0:0001", "message", "%{hsaddr->} %{hsport->} [%{fld20->} %{fld21}] \"%{messageid->} %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hsaddr"),
			constant(" "),
			field("hsport"),
			constant(" ["),
			field("fld20"),
			constant(" "),
			field("fld21"),
			constant("] \""),
			field("messageid"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0002", "message", "%{hevent_time_string->} %{hduration->} %{hsaddr->} %{haction}/%{hresultcode->} %{hsbytes->} %{messageid->} %{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hevent_time_string"),
			constant(" "),
			field("hduration"),
			constant(" "),
			field("hsaddr"),
			constant(" "),
			field("haction"),
			constant("/"),
			field("hresultcode"),
			constant(" "),
			field("hsbytes"),
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
]);

var msg1 = msg("GET", dup15);

var part1 = match("MESSAGE#18:GET:02", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{resultcode->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action->} %{daddr->} %{content_type->} %{duration}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var msg2 = msg("GET:02", part1);

var msg3 = msg("GET:01", dup16);

var select2 = linear_select([
	msg1,
	msg2,
	msg3,
]);

var msg4 = msg("HEAD", dup15);

var msg5 = msg("HEAD:01", dup16);

var select3 = linear_select([
	msg4,
	msg5,
]);

var msg6 = msg("POST", dup17);

var msg7 = msg("POST:01", dup18);

var select4 = linear_select([
	msg6,
	msg7,
]);

var msg8 = msg("PUT", dup19);

var msg9 = msg("PUT:01", dup20);

var select5 = linear_select([
	msg8,
	msg9,
]);

var msg10 = msg("DELETE", dup19);

var msg11 = msg("DELETE:01", dup20);

var select6 = linear_select([
	msg10,
	msg11,
]);

var msg12 = msg("TRACE", dup19);

var msg13 = msg("TRACE:01", dup20);

var select7 = linear_select([
	msg12,
	msg13,
]);

var msg14 = msg("OPTIONS", dup19);

var msg15 = msg("OPTIONS:01", dup20);

var select8 = linear_select([
	msg14,
	msg15,
]);

var msg16 = msg("CONNECT", dup17);

var msg17 = msg("CONNECT:01", dup18);

var select9 = linear_select([
	msg16,
	msg17,
]);

var msg18 = msg("ICP_QUERY", dup19);

var msg19 = msg("ICP_QUERY:01", dup20);

var select10 = linear_select([
	msg18,
	msg19,
]);

var msg20 = msg("PURGE", dup19);

var msg21 = msg("PURGE:01", dup20);

var select11 = linear_select([
	msg20,
	msg21,
]);

var msg22 = msg("PROPFIND", dup19);

var msg23 = msg("PROPFIND:01", dup20);

var select12 = linear_select([
	msg22,
	msg23,
]);

var msg24 = msg("PROPATCH", dup19);

var msg25 = msg("PROPATCH:01", dup20);

var select13 = linear_select([
	msg24,
	msg25,
]);

var msg26 = msg("MKOL", dup19);

var msg27 = msg("MKOL:01", dup20);

var select14 = linear_select([
	msg26,
	msg27,
]);

var msg28 = msg("COPY", dup19);

var msg29 = msg("COPY:01", dup20);

var select15 = linear_select([
	msg28,
	msg29,
]);

var msg30 = msg("MOVE", dup19);

var msg31 = msg("MOVE:01", dup20);

var select16 = linear_select([
	msg30,
	msg31,
]);

var msg32 = msg("LOCK", dup19);

var msg33 = msg("LOCK:01", dup20);

var select17 = linear_select([
	msg32,
	msg33,
]);

var msg34 = msg("UNLOCK", dup19);

var msg35 = msg("UNLOCK:01", dup20);

var select18 = linear_select([
	msg34,
	msg35,
]);

var msg36 = msg("NONE", dup19);

var msg37 = msg("NONE:01", dup20);

var select19 = linear_select([
	msg36,
	msg37,
]);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"CONNECT": select9,
		"COPY": select15,
		"DELETE": select6,
		"GET": select2,
		"HEAD": select3,
		"ICP_QUERY": select10,
		"LOCK": select17,
		"MKOL": select14,
		"MOVE": select16,
		"NONE": select19,
		"OPTIONS": select8,
		"POST": select4,
		"PROPATCH": select13,
		"PROPFIND": select12,
		"PURGE": select11,
		"PUT": select5,
		"TRACE": select7,
		"UNLOCK": select18,
	}),
]);

var part2 = match("MESSAGE#0:GET", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var part3 = match("MESSAGE#19:GET:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));

var part4 = match("MESSAGE#2:POST", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var part5 = match("MESSAGE#21:POST:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup2,
	dup4,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));

var part6 = match("MESSAGE#3:PUT", "nwparser.payload", "%{saddr->} %{sport->} [%{fld20->} %{fld21}] \"%{web_method->} %{url->} %{network_service}\" %{daddr->} %{fld1->} %{username->} \"%{webpage}\" %{resultcode->} %{content_type->} %{sbytes->} \"%{web_referer}\" \"%{user_agent}\" %{action}", processor_chain([
	dup1,
	dup5,
	dup6,
	dup7,
	dup8,
	dup9,
	dup10,
	dup11,
	dup12,
]));

var part7 = match("MESSAGE#22:PUT:01", "nwparser.payload", "%{event_time_string}.%{fld20->} %{duration->} %{saddr->} %{action}/%{resultcode->} %{sbytes->} %{web_method->} %{url->} %{username->} %{h_code}/%{daddr->} %{content_type}", processor_chain([
	dup1,
	dup13,
	dup8,
	dup9,
	dup10,
	dup14,
	dup12,
]));
