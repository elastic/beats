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

var dup2 = date_time({
	dest: "event_time",
	args: ["fld7"],
	fmts: [
		[dD,dc("/"),dB,dc("/"),dW,dc(":"),dN,dc(":"),dU,dc(":"),dO],
	],
});

var dup3 = domain("web_ref_domain","web_referer");

var dup4 = domain("web_domain","web_host");

var dup5 = setf("fqdn","web_host");

var dup6 = setf("msg","$MSG");

var dup7 = match("MESSAGE#0:ABCD", "nwparser.payload", "%{saddr}||%{fld5}||%{username}||[%{fld7->} %{timezone}]||%{web_method}||%{web_host}||%{webpage}||%{web_query}||%{network_service}||%{resultcode}||%{sbytes}||%{web_referer}||%{user_agent}||%{web_cookie}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
]));

var hdr1 = match("HEADER#0:0001", "message", "%APACHETOMCAT-%{level}-%{messageid}: %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = match("HEADER#1:0002", "message", "%{hmonth->} %{hday->} %{htime->} %{hostname->} %APACHETOMCAT- %{messageid}: %{payload}", processor_chain([
	setc("header_id","0002"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
]);

var msg1 = msg("ABCD", dup7);

var msg2 = msg("BADMETHOD", dup7);

var msg3 = msg("BADMTHD", dup7);

var msg4 = msg("BDMTHD", dup7);

var msg5 = msg("INDEX", dup7);

var msg6 = msg("CFYZ", dup7);

var msg7 = msg("CONNECT", dup7);

var msg8 = msg("DELETE", dup7);

var msg9 = msg("DETECT_METHOD_TYPE", dup7);

var msg10 = msg("FGET", dup7);

var msg11 = msg("GET", dup7);

var msg12 = msg("get", dup7);

var msg13 = msg("HEAD", dup7);

var msg14 = msg("id", dup7);

var msg15 = msg("LOCK", dup7);

var msg16 = msg("MKCOL", dup7);

var msg17 = msg("NCIRCLE", dup7);

var msg18 = msg("OPTIONS", dup7);

var msg19 = msg("POST", dup7);

var msg20 = msg("PRONECT", dup7);

var msg21 = msg("PROPFIND", dup7);

var msg22 = msg("PUT", dup7);

var msg23 = msg("QUALYS", dup7);

var msg24 = msg("SEARCH", dup7);

var msg25 = msg("TRACK", dup7);

var msg26 = msg("TRACE", dup7);

var msg27 = msg("uGET", dup7);

var msg28 = msg("null", dup7);

var msg29 = msg("rndmmtd", dup7);

var msg30 = msg("RNDMMTD", dup7);

var msg31 = msg("asdf", dup7);

var msg32 = msg("DEBUG", dup7);

var msg33 = msg("COOK", dup7);

var msg34 = msg("nGET", dup7);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"ABCD": msg1,
		"BADMETHOD": msg2,
		"BADMTHD": msg3,
		"BDMTHD": msg4,
		"CFYZ": msg6,
		"CONNECT": msg7,
		"COOK": msg33,
		"DEBUG": msg32,
		"DELETE": msg8,
		"DETECT_METHOD_TYPE": msg9,
		"FGET": msg10,
		"GET": msg11,
		"HEAD": msg13,
		"INDEX": msg5,
		"LOCK": msg15,
		"MKCOL": msg16,
		"NCIRCLE": msg17,
		"OPTIONS": msg18,
		"POST": msg19,
		"PRONECT": msg20,
		"PROPFIND": msg21,
		"PUT": msg22,
		"QUALYS": msg23,
		"RNDMMTD": msg30,
		"SEARCH": msg24,
		"TRACE": msg26,
		"TRACK": msg25,
		"asdf": msg31,
		"get": msg12,
		"id": msg14,
		"nGET": msg34,
		"null": msg28,
		"rndmmtd": msg29,
		"uGET": msg27,
	}),
]);

var part1 = match("MESSAGE#0:ABCD", "nwparser.payload", "%{saddr}||%{fld5}||%{username}||[%{fld7->} %{timezone}]||%{web_method}||%{web_host}||%{webpage}||%{web_query}||%{network_service}||%{resultcode}||%{sbytes}||%{web_referer}||%{user_agent}||%{web_cookie}", processor_chain([
	dup1,
	dup2,
	dup3,
	dup4,
	dup5,
	dup6,
]));
