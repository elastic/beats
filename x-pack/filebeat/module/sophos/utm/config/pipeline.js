//  Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
//  or more contributor license agreements. Licensed under the Elastic License;
//  you may not use this file except in compliance with the Elastic License.
var tvm = {
	pair_separator: " ",
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

var map_getEventLegacyCategory = {
	keyvaluepairs: {
		"Authentication successful": constant("1302000000"),
		"File extension warned and proceeded": dup43,
		"ICMP flood detected": dup44,
		"Packet accepted": dup43,
		"Packet dropped": dup42,
		"Packet dropped (GEOIP)": dup42,
		"Packet logged": dup43,
		"SYN flood detected": dup44,
		"UDP flood detected": dup44,
		"checking if admin is enabled": constant("1304000000"),
		"http access": constant("1204000000"),
		"portscan detected": constant("1001030300"),
		"web request blocked": dup42,
		"web request blocked, forbidden application detected": dup42,
		"web request blocked, forbidden category detected": dup42,
		"web request blocked, forbidden file extension detected": dup42,
		"web request blocked, forbidden url detected": dup42,
	},
	"default": constant("1901000000"),
};

var map_getEventLegacyCategoryName = {
	keyvaluepairs: {
		"1001030300": constant("Attacks.Access.Informational.Network Based"),
		"1002010000": constant("Attacks.Denial of Service.Bandwidth consumption"),
		"1204000000": constant("Content.Web Traffic"),
		"1302000000": constant("Auth.Successful"),
		"1304000000": constant("Auth.General"),
		"1801000000": constant("Network.Connections"),
		"1803000000": constant("Network.Denied Connections"),
	},
	"default": constant("Other.Default"),
};

var dup1 = setc("eventcategory","1701000000");

var dup2 = setf("msg","$MSG");

var dup3 = date_time({
	dest: "event_time",
	args: ["hfld1"],
	fmts: [
		[dW,dc(":"),dG,dc(":"),dF,dc("-"),dH,dc(":"),dU,dc(":"),dS],
	],
});

var dup4 = setc("eventcategory","1703000000");

var dup5 = setc("eventcategory","1606000000");

var dup6 = setc("eventcategory","1701060000");

var dup7 = setc("eventcategory","1610000000");

var dup8 = setc("eventcategory","1805000000");

var dup9 = setc("action","loaded");

var dup10 = setc("eventcategory","1603000000");

var dup11 = date_time({
	dest: "event_time",
	args: ["hfld1"],
	fmts: [
		[dW,dc(":"),dG,dc(":"),dF,dc("-"),dH,dc(":"),dU,dc(":"),dO],
	],
});

var dup12 = setc("eventcategory","1605000000");

var dup13 = setc("eventcategory","1901000000");

var dup14 = field("event_description");

var dup15 = field("event_cat");

var dup16 = setc("eventcategory","1611000000");

var dup17 = setc("eventcategory","1702000000");

var dup18 = setc("comments","server certificate has a different hostname from actual hostname");

var dup19 = // "Pattern{Field(p0,false)}"
match_copy("MESSAGE#44:reverseproxy:07/1_0", "nwparser.p0", "p0");

var dup20 = setc("eventcategory","1603060000");

var dup21 = setc("eventcategory","1502020000");

var dup22 = setc("comments","No signature on cookie");

var dup23 = setc("eventcategory","1502010000");

var dup24 = setc("eventcategory","1803000000");

var dup25 = setc("eventcategory","1801010000");

var dup26 = setc("eventcategory","1603110000");

var dup27 = setc("eventcategory","1003010000");

var dup28 = setc("event_id","AH01095");

var dup29 = setc("result","Virus daemon connection problem");

var dup30 = setc("eventcategory","1801030000");

var dup31 = setc("event_id","AH01114");

var dup32 = setc("result","Backend connection failed");

var dup33 = setc("eventcategory","1613010000");

var dup34 = setc("eventcategory","1613030000");

var dup35 = setc("event_description","pluto:initiating Main Mode");

var dup36 = setc("event_description","pluto: No response to our first IKE message");

var dup37 = setc("event_description","pluto: starting keying attempt of an unlimited number");

var dup38 = setc("event_description","xl2tpd:xl2tpd Software copyright.");

var dup39 = setc("eventcategory","1207010200");

var dup40 = setc("event_description","exim:connection service message.");

var dup41 = setc("eventcategory","1303000000");

var dup42 = constant("1803000000");

var dup43 = constant("1801000000");

var dup44 = constant("1002010000");

var dup45 = lookup({
	dest: "nwparser.event_cat",
	map: map_getEventLegacyCategory,
	key: dup14,
});

var dup46 = lookup({
	dest: "nwparser.event_cat_name",
	map: map_getEventLegacyCategoryName,
	key: dup15,
});

var hdr1 = // "Pattern{Field(hfld1,true), Constant(' '), Field(messageid,false), Constant('['), Field(process_id,false), Constant(']: '), Field(payload,false)}"
match("HEADER#0:0001", "message", "%{hfld1->} %{messageid}[%{process_id}]: %{payload}", processor_chain([
	setc("header_id","0001"),
]));

var hdr2 = // "Pattern{Field(hfld1,true), Constant(' '), Field(hostname,true), Constant(' '), Field(messageid,false), Constant('['), Field(process_id,false), Constant(']: '), Field(payload,false)}"
match("HEADER#1:0002", "message", "%{hfld1->} %{hostname->} %{messageid}[%{process_id}]: %{payload}", processor_chain([
	setc("header_id","0002"),
]));

var hdr3 = // "Pattern{Field(hfld1,true), Constant(' '), Field(hostname,true), Constant(' reverseproxy: '), Field(payload,false)}"
match("HEADER#2:0003", "message", "%{hfld1->} %{hostname->} reverseproxy: %{payload}", processor_chain([
	setc("header_id","0003"),
	setc("messageid","reverseproxy"),
]));

var hdr4 = // "Pattern{Field(hfld1,true), Constant(' '), Field(hostname,true), Constant(' '), Field(messageid,false), Constant(': '), Field(payload,false)}"
match("HEADER#3:0005", "message", "%{hfld1->} %{hostname->} %{messageid}: %{payload}", processor_chain([
	setc("header_id","0005"),
]));

var hdr5 = // "Pattern{Field(hfld1,true), Constant(' '), Field(id,false), Constant('['), Field(process_id,false), Constant(']: '), Field(payload,false)}"
match("HEADER#4:0004", "message", "%{hfld1->} %{id}[%{process_id}]: %{payload}", processor_chain([
	setc("header_id","0004"),
	setc("messageid","astarosg_TVM"),
]));

var hdr6 = // "Pattern{Constant('device="'), Field(product,false), Constant('" date='), Field(hdate,true), Constant(' time='), Field(htime,true), Constant(' timezone="'), Field(timezone,false), Constant('" device_name="'), Field(device,false), Constant('" device_id='), Field(hardware_id,true), Constant(' log_id='), Field(id,true), Constant(' '), Field(payload,false)}"
match("HEADER#5:0006", "message", "device=\"%{product}\" date=%{hdate->} time=%{htime->} timezone=\"%{timezone}\" device_name=\"%{device}\" device_id=%{hardware_id->} log_id=%{id->} %{payload}", processor_chain([
	setc("header_id","0006"),
	setc("messageid","Sophos_Firewall"),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
	hdr4,
	hdr5,
	hdr6,
]);

var part1 = // "Pattern{Constant('received control channel command ''), Field(action,false), Constant(''')}"
match("MESSAGE#0:named:01", "nwparser.payload", "received control channel command '%{action}'", processor_chain([
	dup1,
	dup2,
	dup3,
]));

var msg1 = msg("named:01", part1);

var part2 = // "Pattern{Constant('flushing caches in all views '), Field(disposition,false)}"
match("MESSAGE#1:named:02", "nwparser.payload", "flushing caches in all views %{disposition}", processor_chain([
	dup1,
	dup2,
	dup3,
]));

var msg2 = msg("named:02", part2);

var part3 = // "Pattern{Constant('error ('), Field(result,false), Constant(') resolving ''), Field(dhost,false), Constant('': '), Field(daddr,false), Constant('#'), Field(dport,false)}"
match("MESSAGE#2:named:03", "nwparser.payload", "error (%{result}) resolving '%{dhost}': %{daddr}#%{dport}", processor_chain([
	dup4,
	dup2,
	dup3,
]));

var msg3 = msg("named:03", part3);

var part4 = // "Pattern{Constant('received '), Field(action,true), Constant(' signal to '), Field(fld3,false)}"
match("MESSAGE#3:named:04", "nwparser.payload", "received %{action->} signal to %{fld3}", processor_chain([
	dup5,
	dup2,
	dup3,
]));

var msg4 = msg("named:04", part4);

var part5 = // "Pattern{Constant('loading configuration from ''), Field(filename,false), Constant(''')}"
match("MESSAGE#4:named:05", "nwparser.payload", "loading configuration from '%{filename}'", processor_chain([
	dup6,
	dup2,
	dup3,
]));

var msg5 = msg("named:05", part5);

var part6 = // "Pattern{Constant('no '), Field(protocol,true), Constant(' interfaces found')}"
match("MESSAGE#5:named:06", "nwparser.payload", "no %{protocol->} interfaces found", processor_chain([
	setc("eventcategory","1804000000"),
	dup2,
	dup3,
]));

var msg6 = msg("named:06", part6);

var part7 = // "Pattern{Constant('sizing zone task pool based on '), Field(fld3,true), Constant(' zones')}"
match("MESSAGE#6:named:07", "nwparser.payload", "sizing zone task pool based on %{fld3->} zones", processor_chain([
	dup7,
	dup2,
	dup3,
]));

var msg7 = msg("named:07", part7);

var part8 = // "Pattern{Constant('automatic empty zone: view '), Field(fld3,false), Constant(': '), Field(dns_ptr_record,false)}"
match("MESSAGE#7:named:08", "nwparser.payload", "automatic empty zone: view %{fld3}: %{dns_ptr_record}", processor_chain([
	dup8,
	dup2,
	dup3,
]));

var msg8 = msg("named:08", part8);

var part9 = // "Pattern{Constant('reloading '), Field(obj_type,true), Constant(' '), Field(disposition,false)}"
match("MESSAGE#8:named:09", "nwparser.payload", "reloading %{obj_type->} %{disposition}", processor_chain([
	dup7,
	dup2,
	dup3,
	setc("action","reloading"),
]));

var msg9 = msg("named:09", part9);

var part10 = // "Pattern{Constant('zone '), Field(dhost,false), Constant('/'), Field(fld3,false), Constant(': loaded serial '), Field(operation_id,false)}"
match("MESSAGE#9:named:10", "nwparser.payload", "zone %{dhost}/%{fld3}: loaded serial %{operation_id}", processor_chain([
	dup7,
	dup9,
	dup2,
	dup3,
]));

var msg10 = msg("named:10", part10);

var part11 = // "Pattern{Constant('all zones loaded'), Field(,false)}"
match("MESSAGE#10:named:11", "nwparser.payload", "all zones loaded%{}", processor_chain([
	dup7,
	dup9,
	dup2,
	dup3,
	setc("action","all zones loaded"),
]));

var msg11 = msg("named:11", part11);

var part12 = // "Pattern{Constant('running'), Field(,false)}"
match("MESSAGE#11:named:12", "nwparser.payload", "running%{}", processor_chain([
	dup7,
	setc("disposition","running"),
	dup2,
	dup3,
	setc("action","running"),
]));

var msg12 = msg("named:12", part12);

var part13 = // "Pattern{Constant('using built-in root key for view '), Field(fld3,false)}"
match("MESSAGE#12:named:13", "nwparser.payload", "using built-in root key for view %{fld3}", processor_chain([
	dup7,
	setc("context","built-in root key"),
	dup2,
	dup3,
]));

var msg13 = msg("named:13", part13);

var part14 = // "Pattern{Constant('zone '), Field(dns_ptr_record,false), Constant('/'), Field(fld3,false), Constant(': ('), Field(username,false), Constant(') '), Field(action,false)}"
match("MESSAGE#13:named:14", "nwparser.payload", "zone %{dns_ptr_record}/%{fld3}: (%{username}) %{action}", processor_chain([
	dup8,
	dup2,
	dup3,
]));

var msg14 = msg("named:14", part14);

var part15 = // "Pattern{Constant('too many timeouts resolving ''), Field(fld3,false), Constant('' ('), Field(fld4,false), Constant('): disabling EDNS')}"
match("MESSAGE#14:named:15", "nwparser.payload", "too many timeouts resolving '%{fld3}' (%{fld4}): disabling EDNS", processor_chain([
	dup10,
	setc("event_description","named:too many timeouts resolving DNS."),
	dup11,
	dup2,
]));

var msg15 = msg("named:15", part15);

var part16 = // "Pattern{Constant('FORMERR resolving ''), Field(hostname,false), Constant('': '), Field(saddr,false), Constant('#'), Field(fld3,false)}"
match("MESSAGE#15:named:16", "nwparser.payload", "FORMERR resolving '%{hostname}': %{saddr}#%{fld3}", processor_chain([
	dup10,
	setc("event_description","named:FORMERR resolving DNS."),
	dup11,
	dup2,
]));

var msg16 = msg("named:16", part16);

var part17 = // "Pattern{Constant('unexpected RCODE (SERVFAIL) resolving ''), Field(hostname,false), Constant('': '), Field(saddr,false), Constant('#'), Field(fld3,false)}"
match("MESSAGE#16:named:17", "nwparser.payload", "unexpected RCODE (SERVFAIL) resolving '%{hostname}': %{saddr}#%{fld3}", processor_chain([
	dup10,
	setc("event_description","named:unexpected RCODE (SERVFAIL) resolving DNS."),
	dup11,
	dup2,
]));

var msg17 = msg("named:17", part17);

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
	msg13,
	msg14,
	msg15,
	msg16,
	msg17,
]);

var part18 = // "Pattern{Constant('Integrated HTTP-Proxy '), Field(version,false)}"
match("MESSAGE#17:httpproxy:09", "nwparser.payload", "Integrated HTTP-Proxy %{version}", processor_chain([
	dup12,
	setc("event_description","httpproxy:Integrated HTTP-Proxy."),
	dup11,
	dup2,
]));

var msg18 = msg("httpproxy:09", part18);

var part19 = // "Pattern{Constant('['), Field(fld2,false), Constant('] parse_address ('), Field(fld3,false), Constant(') getaddrinfo: passthrough.fw-notify.net: Name or service not known')}"
match("MESSAGE#18:httpproxy:10", "nwparser.payload", "[%{fld2}] parse_address (%{fld3}) getaddrinfo: passthrough.fw-notify.net: Name or service not known", processor_chain([
	dup10,
	setc("event_description","httpproxy:Name or service not known."),
	dup11,
	dup2,
]));

var msg19 = msg("httpproxy:10", part19);

var part20 = // "Pattern{Constant('['), Field(fld2,false), Constant('] confd_config_filter ('), Field(fld3,false), Constant(') failed to resolve passthrough.fw-notify.net, using '), Field(saddr,false)}"
match("MESSAGE#19:httpproxy:11", "nwparser.payload", "[%{fld2}] confd_config_filter (%{fld3}) failed to resolve passthrough.fw-notify.net, using %{saddr}", processor_chain([
	dup10,
	setc("event_description","httpproxy:failed to resolve passthrough."),
	dup11,
	dup2,
]));

var msg20 = msg("httpproxy:11", part20);

var part21 = // "Pattern{Constant('['), Field(fld2,false), Constant('] ssl_log_errors ('), Field(fld3,false), Constant(') '), Field(fld4,false), Constant('ssl handshake failure'), Field(fld5,false)}"
match("MESSAGE#20:httpproxy:12", "nwparser.payload", "[%{fld2}] ssl_log_errors (%{fld3}) %{fld4}ssl handshake failure%{fld5}", processor_chain([
	dup10,
	setc("event_description","httpproxy:ssl handshake failure."),
	dup11,
	dup2,
]));

var msg21 = msg("httpproxy:12", part21);

var part22 = // "Pattern{Constant('['), Field(fld2,false), Constant('] sc_decrypt ('), Field(fld3,false), Constant(') EVP_DecryptFinal failed')}"
match("MESSAGE#21:httpproxy:13", "nwparser.payload", "[%{fld2}] sc_decrypt (%{fld3}) EVP_DecryptFinal failed", processor_chain([
	dup10,
	setc("event_description","httpproxy:EVP_DecryptFinal failed."),
	dup11,
	dup2,
]));

var msg22 = msg("httpproxy:13", part22);

var part23 = // "Pattern{Constant('['), Field(fld2,false), Constant('] sc_server_cmd ('), Field(fld3,false), Constant(') decrypt failed')}"
match("MESSAGE#22:httpproxy:14", "nwparser.payload", "[%{fld2}] sc_server_cmd (%{fld3}) decrypt failed", processor_chain([
	dup10,
	setc("event_description","httpproxy:decrypt failed."),
	dup11,
	dup2,
]));

var msg23 = msg("httpproxy:14", part23);

var part24 = // "Pattern{Constant('['), Field(fld2,false), Constant('] clamav_reload ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#23:httpproxy:15", "nwparser.payload", "[%{fld2}] clamav_reload (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:reloading av pattern"),
	dup11,
	dup2,
]));

var msg24 = msg("httpproxy:15", part24);

var part25 = // "Pattern{Constant('['), Field(fld2,false), Constant('] sc_check_servers ('), Field(fld3,false), Constant(') server ''), Field(hostname,false), Constant('' access time: '), Field(fld4,false)}"
match("MESSAGE#24:httpproxy:16", "nwparser.payload", "[%{fld2}] sc_check_servers (%{fld3}) server '%{hostname}' access time: %{fld4}", processor_chain([
	dup12,
	setc("event_description","httpproxy:sc_check_servers.Server checked."),
	dup11,
	dup2,
]));

var msg25 = msg("httpproxy:16", part25);

var part26 = // "Pattern{Constant('['), Field(fld2,false), Constant('] main ('), Field(fld3,false), Constant(') shutdown finished, exiting')}"
match("MESSAGE#25:httpproxy:17", "nwparser.payload", "[%{fld2}] main (%{fld3}) shutdown finished, exiting", processor_chain([
	dup12,
	setc("event_description","httpproxy:shutdown finished, exiting."),
	dup11,
	dup2,
]));

var msg26 = msg("httpproxy:17", part26);

var part27 = // "Pattern{Constant('['), Field(fld2,false), Constant('] main ('), Field(fld3,false), Constant(') reading configuration')}"
match("MESSAGE#26:httpproxy:18", "nwparser.payload", "[%{fld2}] main (%{fld3}) reading configuration", processor_chain([
	dup12,
	setc("event_description","httpproxy:"),
	dup11,
	dup2,
]));

var msg27 = msg("httpproxy:18", part27);

var part28 = // "Pattern{Constant('['), Field(fld2,false), Constant('] main ('), Field(fld3,false), Constant(') reading profiles')}"
match("MESSAGE#27:httpproxy:19", "nwparser.payload", "[%{fld2}] main (%{fld3}) reading profiles", processor_chain([
	dup12,
	setc("event_description","httpproxy:reading profiles"),
	dup11,
	dup2,
]));

var msg28 = msg("httpproxy:19", part28);

var part29 = // "Pattern{Constant('['), Field(fld2,false), Constant('] main ('), Field(fld3,false), Constant(') finished startup')}"
match("MESSAGE#28:httpproxy:20", "nwparser.payload", "[%{fld2}] main (%{fld3}) finished startup", processor_chain([
	dup12,
	setc("event_description","httpproxy:finished startup"),
	dup11,
	dup2,
]));

var msg29 = msg("httpproxy:20", part29);

var part30 = // "Pattern{Constant('['), Field(fld2,false), Constant('] read_request_headers ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#29:httpproxy:21", "nwparser.payload", "[%{fld2}] read_request_headers (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:read_request_headers related message."),
	dup11,
	dup2,
]));

var msg30 = msg("httpproxy:21", part30);

var part31 = // "Pattern{Constant('['), Field(fld2,false), Constant('] epoll_loop ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#30:httpproxy:22", "nwparser.payload", "[%{fld2}] epoll_loop (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:epoll_loop related message."),
	dup11,
	dup2,
]));

var msg31 = msg("httpproxy:22", part31);

var part32 = // "Pattern{Constant('['), Field(fld2,false), Constant('] scan_exit ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#31:httpproxy:23", "nwparser.payload", "[%{fld2}] scan_exit (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:scan_exit related message."),
	dup11,
	dup2,
]));

var msg32 = msg("httpproxy:23", part32);

var part33 = // "Pattern{Constant('['), Field(fld2,false), Constant('] epoll_exit ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#32:httpproxy:24", "nwparser.payload", "[%{fld2}] epoll_exit (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:epoll_exit related message."),
	dup11,
	dup2,
]));

var msg33 = msg("httpproxy:24", part33);

var part34 = // "Pattern{Constant('['), Field(fld2,false), Constant('] disk_cache_exit ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#33:httpproxy:25", "nwparser.payload", "[%{fld2}] disk_cache_exit (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:disk_cache_exit related message."),
	dup11,
	dup2,
]));

var msg34 = msg("httpproxy:25", part34);

var part35 = // "Pattern{Constant('['), Field(fld2,false), Constant('] disk_cache_zap ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#34:httpproxy:26", "nwparser.payload", "[%{fld2}] disk_cache_zap (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:disk_cache_zap related message."),
	dup11,
	dup2,
]));

var msg35 = msg("httpproxy:26", part35);

var part36 = // "Pattern{Constant('['), Field(fld2,false), Constant('] scanner_init ('), Field(fld3,false), Constant(') '), Field(info,false)}"
match("MESSAGE#35:httpproxy:27", "nwparser.payload", "[%{fld2}] scanner_init (%{fld3}) %{info}", processor_chain([
	dup12,
	setc("event_description","httpproxy:scanner_init related message."),
	dup11,
	dup2,
]));

var msg36 = msg("httpproxy:27", part36);

var part37 = tagval("MESSAGE#36:httpproxy:01", "nwparser.payload", tvm, {
	"action": "action",
	"ad_domain": "fld1",
	"app-id": "fld18",
	"application": "fld17",
	"auth": "fld10",
	"authtime": "fld4",
	"avscantime": "fld7",
	"cached": "fld2",
	"category": "policy_id",
	"categoryname": "info",
	"cattime": "fld6",
	"content-type": "content_type",
	"device": "fld9",
	"dnstime": "fld5",
	"dstip": "daddr",
	"error": "result",
	"exceptions": "fld12",
	"extension": "fld13",
	"file": "filename",
	"filename": "filename",
	"filteraction": "fld3",
	"fullreqtime": "fld8",
	"function": "action",
	"group": "group",
	"id": "rule",
	"line": "fld14",
	"message": "context",
	"method": "web_method",
	"name": "event_description",
	"profile": "policyname",
	"reason": "rule_group",
	"referer": "web_referer",
	"reputation": "fld16",
	"request": "connectionid",
	"severity": "severity",
	"size": "rbytes",
	"srcip": "saddr",
	"statuscode": "resultcode",
	"sub": "network_service",
	"sys": "vsys",
	"time": "fld15",
	"ua": "fld11",
	"url": "url",
	"user": "username",
}, processor_chain([
	dup13,
	dup11,
	dup2,
	dup45,
	dup46,
]));

var msg37 = msg("httpproxy:01", part37);

var select3 = linear_select([
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
]);

var part38 = // "Pattern{Constant('T='), Field(fld3,true), Constant(' ------ 1 - [exit] '), Field(action,false), Constant(': '), Field(disposition,false)}"
match("MESSAGE#37:URID:01", "nwparser.payload", "T=%{fld3->} ------ 1 - [exit] %{action}: %{disposition}", processor_chain([
	dup16,
	dup2,
	dup3,
]));

var msg38 = msg("URID:01", part38);

var part39 = tagval("MESSAGE#38:ulogd:01", "nwparser.payload", tvm, {
	"action": "action",
	"code": "fld30",
	"dstip": "daddr",
	"dstmac": "dmacaddr",
	"dstport": "dport",
	"fwrule": "policy_id",
	"id": "rule",
	"info": "context",
	"initf": "sinterface",
	"length": "fld25",
	"name": "event_description",
	"outitf": "dinterface",
	"prec": "fld27",
	"proto": "fld24",
	"seq": "fld23",
	"severity": "severity",
	"srcip": "saddr",
	"srcmac": "smacaddr",
	"srcport": "sport",
	"sub": "network_service",
	"sys": "vsys",
	"tcpflags": "fld29",
	"tos": "fld26",
	"ttl": "fld28",
	"type": "fld31",
}, processor_chain([
	dup13,
	setc("ec_subject","NetworkComm"),
	setc("ec_activity","Scan"),
	setc("ec_theme","TEV"),
	dup11,
	dup2,
	dup45,
	dup46,
]));

var msg39 = msg("ulogd:01", part39);

var part40 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] ModSecurity for Apache/'), Field(fld5,true), Constant(' ('), Field(fld6,false), Constant(') configured.')}"
match("MESSAGE#39:reverseproxy:01", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] ModSecurity for Apache/%{fld5->} (%{fld6}) configured.", processor_chain([
	dup6,
	setc("disposition","configured"),
	dup2,
	dup3,
]));

var msg40 = msg("reverseproxy:01", part40);

var part41 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] ModSecurity: '), Field(fld5,true), Constant(' compiled version="'), Field(fld6,false), Constant('"; loaded version="'), Field(fld7,false), Constant('"')}"
match("MESSAGE#40:reverseproxy:02", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] ModSecurity: %{fld5->} compiled version=\"%{fld6}\"; loaded version=\"%{fld7}\"", processor_chain([
	dup17,
	dup2,
	dup3,
]));

var msg41 = msg("reverseproxy:02", part41);

var part42 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] ModSecurity: '), Field(fld5,true), Constant(' compiled version="'), Field(fld6,false), Constant('"')}"
match("MESSAGE#41:reverseproxy:03", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] ModSecurity: %{fld5->} compiled version=\"%{fld6}\"", processor_chain([
	dup17,
	dup2,
	dup3,
]));

var msg42 = msg("reverseproxy:03", part42);

var part43 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] '), Field(fld5,true), Constant(' configured -- '), Field(disposition,true), Constant(' normal operations')}"
match("MESSAGE#42:reverseproxy:04", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] %{fld5->} configured -- %{disposition->} normal operations", processor_chain([
	dup17,
	setc("event_id","AH00292"),
	dup2,
	dup3,
]));

var msg43 = msg("reverseproxy:04", part43);

var part44 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] ['), Field(fld5,false), Constant('] Hostname in '), Field(network_service,true), Constant(' request ('), Field(fld6,false), Constant(') does not match the server name ('), Field(ddomain,false), Constant(')')}"
match("MESSAGE#43:reverseproxy:06", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [%{fld5}] Hostname in %{network_service->} request (%{fld6}) does not match the server name (%{ddomain})", processor_chain([
	setc("eventcategory","1805010000"),
	dup18,
	dup2,
	dup3,
]));

var msg44 = msg("reverseproxy:06", part44);

var part45 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00297: '), Field(action,true), Constant(' received. Doing'), Field(p0,false)}"
match("MESSAGE#44:reverseproxy:07/0", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00297: %{action->} received. Doing%{p0}");

var select4 = linear_select([
	dup19,
]);

var part46 = // "Pattern{Field(,false), Constant('graceful '), Field(disposition,false)}"
match("MESSAGE#44:reverseproxy:07/2", "nwparser.p0", "%{}graceful %{disposition}");

var all1 = all_match({
	processors: [
		part45,
		select4,
		part46,
	],
	on_success: processor_chain([
		dup5,
		setc("event_id","AH00297"),
		dup2,
		dup3,
	]),
});

var msg45 = msg("reverseproxy:07", all1);

var part47 = // "Pattern{Constant('AH00112: Warning: DocumentRoot ['), Field(web_root,false), Constant('] does not exist')}"
match("MESSAGE#45:reverseproxy:08", "nwparser.payload", "AH00112: Warning: DocumentRoot [%{web_root}] does not exist", processor_chain([
	dup4,
	setc("event_id","AH00112"),
	dup2,
	dup3,
]));

var msg46 = msg("reverseproxy:08", part47);

var part48 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00094: Command line: ''), Field(web_root,false), Constant(''')}"
match("MESSAGE#46:reverseproxy:09", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00094: Command line: '%{web_root}'", processor_chain([
	setc("eventcategory","1605010000"),
	setc("event_id","AH00094"),
	dup2,
	dup3,
]));

var msg47 = msg("reverseproxy:09", part48);

var part49 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00291: long lost child came home! (pid '), Field(fld5,false), Constant(')')}"
match("MESSAGE#47:reverseproxy:10", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00291: long lost child came home! (pid %{fld5})", processor_chain([
	dup12,
	setc("event_id","AH00291"),
	dup2,
	dup3,
]));

var msg48 = msg("reverseproxy:10", part49);

var part50 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH02572: Failed to configure at least one certificate and key for '), Field(fld5,false), Constant(':'), Field(fld6,false)}"
match("MESSAGE#48:reverseproxy:11", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH02572: Failed to configure at least one certificate and key for %{fld5}:%{fld6}", processor_chain([
	dup20,
	setc("event_id","AH02572"),
	dup2,
	dup3,
]));

var msg49 = msg("reverseproxy:11", part50);

var part51 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] SSL Library Error: error:'), Field(resultcode,false), Constant(':'), Field(result,false)}"
match("MESSAGE#49:reverseproxy:12", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] SSL Library Error: error:%{resultcode}:%{result}", processor_chain([
	dup20,
	setc("context","SSL Library Error"),
	dup2,
	dup3,
]));

var msg50 = msg("reverseproxy:12", part51);

var part52 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH02312: Fatal error initialising mod_ssl, '), Field(disposition,false), Constant('.')}"
match("MESSAGE#50:reverseproxy:13", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH02312: Fatal error initialising mod_ssl, %{disposition}.", processor_chain([
	dup20,
	setc("result","Fatal error"),
	setc("event_id","AH02312"),
	dup2,
	dup3,
]));

var msg51 = msg("reverseproxy:13", part52);

var part53 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00020: Configuration Failed, '), Field(disposition,false)}"
match("MESSAGE#51:reverseproxy:14", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00020: Configuration Failed, %{disposition}", processor_chain([
	dup20,
	setc("result","Configuration Failed"),
	setc("event_id","AH00020"),
	dup2,
	dup3,
]));

var msg52 = msg("reverseproxy:14", part53);

var part54 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00098: pid file '), Field(filename,true), Constant(' overwritten -- Unclean shutdown of previous Apache run?')}"
match("MESSAGE#52:reverseproxy:15", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00098: pid file %{filename->} overwritten -- Unclean shutdown of previous Apache run?", processor_chain([
	setc("eventcategory","1609000000"),
	setc("context","Unclean shutdown"),
	setc("event_id","AH00098"),
	dup2,
	dup3,
]));

var msg53 = msg("reverseproxy:15", part54);

var part55 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00295: caught '), Field(action,false), Constant(', '), Field(disposition,false)}"
match("MESSAGE#53:reverseproxy:16", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00295: caught %{action}, %{disposition}", processor_chain([
	dup16,
	setc("event_id","AH00295"),
	dup2,
	dup3,
]));

var msg54 = msg("reverseproxy:16", part55);

var part56 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(result,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ModSecurity: Warning. '), Field(rulename,true), Constant(' [file "'), Field(filename,false), Constant('"] [line "'), Field(fld5,false), Constant('"] [id "'), Field(rule,false), Constant('"]'), Field(p0,false)}"
match("MESSAGE#54:reverseproxy:17/0", "nwparser.payload", "[%{fld3}] [%{event_log}:%{result}] [pid %{process_id}:%{fld4}] [client %{gateway}] ModSecurity: Warning. %{rulename->} [file \"%{filename}\"] [line \"%{fld5}\"] [id \"%{rule}\"]%{p0}");

var part57 = // "Pattern{Constant(' [rev "'), Field(fld6,false), Constant('"]'), Field(p0,false)}"
match("MESSAGE#54:reverseproxy:17/1_0", "nwparser.p0", " [rev \"%{fld6}\"]%{p0}");

var select5 = linear_select([
	part57,
	dup19,
]);

var part58 = // "Pattern{Field(,false), Constant('[msg "'), Field(comments,false), Constant('"] [data "'), Field(daddr,false), Constant('"] [severity "'), Field(severity,false), Constant('"] [ver "'), Field(policyname,false), Constant('"] [maturity "'), Field(fld7,false), Constant('"] [accuracy "'), Field(fld8,false), Constant('"] '), Field(context,true), Constant(' [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#54:reverseproxy:17/2", "nwparser.p0", "%{}[msg \"%{comments}\"] [data \"%{daddr}\"] [severity \"%{severity}\"] [ver \"%{policyname}\"] [maturity \"%{fld7}\"] [accuracy \"%{fld8}\"] %{context->} [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]");

var all2 = all_match({
	processors: [
		part56,
		select5,
		part58,
	],
	on_success: processor_chain([
		dup21,
		dup2,
		dup3,
	]),
});

var msg55 = msg("reverseproxy:17", all2);

var part59 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] No signature found, cookie: '), Field(fld5,false)}"
match("MESSAGE#55:reverseproxy:18", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] No signature found, cookie: %{fld5}", processor_chain([
	dup4,
	dup22,
	dup2,
	dup3,
]));

var msg56 = msg("reverseproxy:18", part59);

var part60 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] '), Field(disposition,true), Constant(' ''), Field(fld5,false), Constant('' from request due to missing/invalid signature')}"
match("MESSAGE#56:reverseproxy:19", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] %{disposition->} '%{fld5}' from request due to missing/invalid signature", processor_chain([
	dup23,
	dup22,
	dup2,
	dup3,
]));

var msg57 = msg("reverseproxy:19", part60);

var part61 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ModSecurity: Warning. '), Field(rulename,true), Constant(' [file "'), Field(filename,false), Constant('"] [line "'), Field(fld5,false), Constant('"] [id "'), Field(rule,false), Constant('"] [msg "'), Field(comments,false), Constant('"] [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#57:reverseproxy:20", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] ModSecurity: Warning. %{rulename->} [file \"%{filename}\"] [line \"%{fld5}\"] [id \"%{rule}\"] [msg \"%{comments}\"] [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]", processor_chain([
	dup21,
	dup2,
	dup3,
]));

var msg58 = msg("reverseproxy:20", part61);

var part62 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH01909: '), Field(daddr,false), Constant(':'), Field(dport,false), Constant(':'), Field(fld5,true), Constant(' server certificate does NOT include an ID which matches the server name')}"
match("MESSAGE#58:reverseproxy:21", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH01909: %{daddr}:%{dport}:%{fld5->} server certificate does NOT include an ID which matches the server name", processor_chain([
	dup20,
	dup18,
	setc("event_id","AH01909"),
	dup2,
	dup3,
]));

var msg59 = msg("reverseproxy:21", part62);

var part63 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH01915: Init: ('), Field(daddr,false), Constant(':'), Field(dport,false), Constant(') You configured '), Field(network_service,false), Constant('('), Field(fld5,false), Constant(') on the '), Field(fld6,false), Constant('('), Field(fld7,false), Constant(') port!')}"
match("MESSAGE#59:reverseproxy:22", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH01915: Init: (%{daddr}:%{dport}) You configured %{network_service}(%{fld5}) on the %{fld6}(%{fld7}) port!", processor_chain([
	dup20,
	setc("comments","Invalid port configuration"),
	dup2,
	dup3,
]));

var msg60 = msg("reverseproxy:22", part63);

var part64 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ModSecurity: Rule '), Field(rulename,true), Constant(' [id "'), Field(rule,false), Constant('"][file "'), Field(filename,false), Constant('"][line "'), Field(fld5,false), Constant('"] - Execution error - PCRE limits exceeded ('), Field(fld6,false), Constant('): ('), Field(fld7,false), Constant('). [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#60:reverseproxy:23", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] ModSecurity: Rule %{rulename->} [id \"%{rule}\"][file \"%{filename}\"][line \"%{fld5}\"] - Execution error - PCRE limits exceeded (%{fld6}): (%{fld7}). [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]", processor_chain([
	dup21,
	dup2,
	dup3,
]));

var msg61 = msg("reverseproxy:23", part64);

var part65 = // "Pattern{Constant('rManage\\x22,\\x22manageLiveSystemSettings\\x22,\\x22accessViewJobs\\x22,\\x22exportList\\..."] [ver "'), Field(policyname,false), Constant('"] [maturity "'), Field(fld3,false), Constant('"] [accuracy "'), Field(fld4,false), Constant('"] '), Field(context,true), Constant(' [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#61:reverseproxy:24", "nwparser.payload", "rManage\\\\x22,\\\\x22manageLiveSystemSettings\\\\x22,\\\\x22accessViewJobs\\\\x22,\\\\x22exportList\\\\...\"] [ver \"%{policyname}\"] [maturity \"%{fld3}\"] [accuracy \"%{fld4}\"] %{context->} [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]", processor_chain([
	dup21,
	dup2,
	dup3,
]));

var msg62 = msg("reverseproxy:24", part65);

var part66 = // "Pattern{Constant('ARGS:userPermissions: [\\x22dashletAccessAlertingRecentAlertsPanel\\x22,\\x22dashletAccessAlerterTopAlertsDashlet\\x22,\\x22accessViewRules\\x22,\\x22deployLiveResources\\x22,\\x22vi..."] [severity [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#62:reverseproxy:25", "nwparser.payload", "ARGS:userPermissions: [\\\\x22dashletAccessAlertingRecentAlertsPanel\\\\x22,\\\\x22dashletAccessAlerterTopAlertsDashlet\\\\x22,\\\\x22accessViewRules\\\\x22,\\\\x22deployLiveResources\\\\x22,\\\\x22vi...\"] [severity [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]", processor_chain([
	dup21,
	dup2,
	dup3,
]));

var msg63 = msg("reverseproxy:25", part66);

var part67 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ModSecurity: '), Field(disposition,true), Constant(' with code '), Field(resultcode,true), Constant(' ('), Field(fld5,false), Constant('). '), Field(rulename,true), Constant(' [file "'), Field(filename,false), Constant('"] [line "'), Field(fld6,false), Constant('"] [id "'), Field(rule,false), Constant('"]'), Field(p0,false)}"
match("MESSAGE#63:reverseproxy:26/0", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] ModSecurity: %{disposition->} with code %{resultcode->} (%{fld5}). %{rulename->} [file \"%{filename}\"] [line \"%{fld6}\"] [id \"%{rule}\"]%{p0}");

var part68 = // "Pattern{Constant(' [rev "'), Field(fld7,false), Constant('"]'), Field(p0,false)}"
match("MESSAGE#63:reverseproxy:26/1_0", "nwparser.p0", " [rev \"%{fld7}\"]%{p0}");

var select6 = linear_select([
	part68,
	dup19,
]);

var part69 = // "Pattern{Field(,false), Constant('[msg "'), Field(comments,false), Constant('"] [data "Last Matched Data: '), Field(p0,false)}"
match("MESSAGE#63:reverseproxy:26/2", "nwparser.p0", "%{}[msg \"%{comments}\"] [data \"Last Matched Data: %{p0}");

var part70 = // "Pattern{Field(daddr,false), Constant(':'), Field(dport,false), Constant('"] [hostname "'), Field(p0,false)}"
match("MESSAGE#63:reverseproxy:26/3_0", "nwparser.p0", "%{daddr}:%{dport}\"] [hostname \"%{p0}");

var part71 = // "Pattern{Field(daddr,false), Constant('"] [hostname "'), Field(p0,false)}"
match("MESSAGE#63:reverseproxy:26/3_1", "nwparser.p0", "%{daddr}\"] [hostname \"%{p0}");

var select7 = linear_select([
	part70,
	part71,
]);

var part72 = // "Pattern{Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]')}"
match("MESSAGE#63:reverseproxy:26/4", "nwparser.p0", "%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]");

var all3 = all_match({
	processors: [
		part67,
		select6,
		part69,
		select7,
		part72,
	],
	on_success: processor_chain([
		dup24,
		dup2,
		dup3,
	]),
});

var msg64 = msg("reverseproxy:26", all3);

var part73 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] '), Field(disposition,true), Constant(' while reading reply from cssd, referer: '), Field(web_referer,false)}"
match("MESSAGE#64:reverseproxy:27", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] %{disposition->} while reading reply from cssd, referer: %{web_referer}", processor_chain([
	dup25,
	dup2,
	dup3,
]));

var msg65 = msg("reverseproxy:27", part73);

var part74 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] virus daemon error found in request '), Field(web_root,false), Constant(', referer: '), Field(web_referer,false)}"
match("MESSAGE#65:reverseproxy:28", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] virus daemon error found in request %{web_root}, referer: %{web_referer}", processor_chain([
	dup26,
	setc("result","virus daemon error"),
	dup2,
	dup3,
]));

var msg66 = msg("reverseproxy:28", part74);

var part75 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] mod_avscan_input_filter: virus found, referer: '), Field(web_referer,false)}"
match("MESSAGE#66:reverseproxy:29", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] mod_avscan_input_filter: virus found, referer: %{web_referer}", processor_chain([
	dup27,
	setc("result","virus found"),
	dup2,
	dup3,
]));

var msg67 = msg("reverseproxy:29", part75);

var part76 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] (13)'), Field(result,false), Constant(': [client '), Field(gateway,false), Constant('] AH01095: prefetch request body failed to '), Field(saddr,false), Constant(':'), Field(sport,true), Constant(' ('), Field(fld5,false), Constant(') from '), Field(fld6,true), Constant(' (), referer: '), Field(web_referer,false)}"
match("MESSAGE#67:reverseproxy:30", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] (13)%{result}: [client %{gateway}] AH01095: prefetch request body failed to %{saddr}:%{sport->} (%{fld5}) from %{fld6->} (), referer: %{web_referer}", processor_chain([
	dup24,
	dup28,
	dup2,
	dup3,
]));

var msg68 = msg("reverseproxy:30", part76);

var part77 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] cannot read reply: Operation now in progress (115), referer: '), Field(web_referer,false)}"
match("MESSAGE#68:reverseproxy:31", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] cannot read reply: Operation now in progress (115), referer: %{web_referer}", processor_chain([
	dup25,
	setc("result","Cannot read reply"),
	dup2,
	dup3,
]));

var msg69 = msg("reverseproxy:31", part77);

var part78 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] cannot connect: '), Field(result,true), Constant(' (111), referer: '), Field(web_referer,false)}"
match("MESSAGE#69:reverseproxy:32", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] cannot connect: %{result->} (111), referer: %{web_referer}", processor_chain([
	dup25,
	dup2,
	dup3,
]));

var msg70 = msg("reverseproxy:32", part78);

var part79 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] cannot connect: '), Field(result,true), Constant(' (111)')}"
match("MESSAGE#70:reverseproxy:33", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] cannot connect: %{result->} (111)", processor_chain([
	dup25,
	dup2,
	dup3,
]));

var msg71 = msg("reverseproxy:33", part79);

var part80 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] virus daemon connection problem found in request '), Field(url,false), Constant(', referer: '), Field(web_referer,false)}"
match("MESSAGE#71:reverseproxy:34", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] virus daemon connection problem found in request %{url}, referer: %{web_referer}", processor_chain([
	dup26,
	dup29,
	dup2,
	dup3,
]));

var msg72 = msg("reverseproxy:34", part80);

var part81 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] virus daemon connection problem found in request '), Field(url,false)}"
match("MESSAGE#72:reverseproxy:35", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] virus daemon connection problem found in request %{url}", processor_chain([
	dup26,
	dup29,
	dup2,
	dup3,
]));

var msg73 = msg("reverseproxy:35", part81);

var part82 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] mod_avscan_input_filter: virus found')}"
match("MESSAGE#73:reverseproxy:36", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] mod_avscan_input_filter: virus found", processor_chain([
	dup27,
	setc("result","Virus found"),
	dup2,
	dup3,
]));

var msg74 = msg("reverseproxy:36", part82);

var part83 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] (13)'), Field(result,false), Constant(': [client '), Field(gateway,false), Constant('] AH01095: prefetch request body failed to '), Field(saddr,false), Constant(':'), Field(sport,true), Constant(' ('), Field(fld5,false), Constant(') from '), Field(fld6,true), Constant(' ()')}"
match("MESSAGE#74:reverseproxy:37", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] (13)%{result}: [client %{gateway}] AH01095: prefetch request body failed to %{saddr}:%{sport->} (%{fld5}) from %{fld6->} ()", processor_chain([
	dup24,
	dup28,
	dup2,
	dup3,
]));

var msg75 = msg("reverseproxy:37", part83);

var part84 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] Invalid signature, cookie: JSESSIONID')}"
match("MESSAGE#75:reverseproxy:38", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] Invalid signature, cookie: JSESSIONID", processor_chain([
	dup25,
	dup2,
	dup3,
]));

var msg76 = msg("reverseproxy:38", part84);

var part85 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] Form validation failed: Received unhardened form data, referer: '), Field(web_referer,false)}"
match("MESSAGE#76:reverseproxy:39", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] Form validation failed: Received unhardened form data, referer: %{web_referer}", processor_chain([
	dup23,
	setc("result","Form validation failed"),
	dup2,
	dup3,
]));

var msg77 = msg("reverseproxy:39", part85);

var part86 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] sending trickle failed: 103')}"
match("MESSAGE#77:reverseproxy:40", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] sending trickle failed: 103", processor_chain([
	dup25,
	setc("result","Sending trickle failed"),
	dup2,
	dup3,
]));

var msg78 = msg("reverseproxy:40", part86);

var part87 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] client requesting '), Field(web_root,true), Constant(' has '), Field(disposition,false)}"
match("MESSAGE#78:reverseproxy:41", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] client requesting %{web_root->} has %{disposition}", processor_chain([
	dup30,
	dup2,
	dup3,
]));

var msg79 = msg("reverseproxy:41", part87);

var part88 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] mod_avscan_check_file_single_part() called with parameter filename='), Field(filename,false)}"
match("MESSAGE#79:reverseproxy:42", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] mod_avscan_check_file_single_part() called with parameter filename=%{filename}", processor_chain([
	setc("eventcategory","1603050000"),
	dup2,
	dup3,
]));

var msg80 = msg("reverseproxy:42", part88);

var part89 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] (70007)The '), Field(disposition,true), Constant(' specified has expired: [client '), Field(gateway,false), Constant('] AH01110: error reading response')}"
match("MESSAGE#80:reverseproxy:43", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] (70007)The %{disposition->} specified has expired: [client %{gateway}] AH01110: error reading response", processor_chain([
	dup30,
	setc("event_id","AH01110"),
	setc("result","Error reading response"),
	dup2,
	dup3,
]));

var msg81 = msg("reverseproxy:43", part89);

var part90 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] (22)'), Field(result,false), Constant(': [client '), Field(gateway,false), Constant('] No form context found when parsing '), Field(fld5,true), Constant(' tag, referer: '), Field(web_referer,false)}"
match("MESSAGE#81:reverseproxy:44", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] (22)%{result}: [client %{gateway}] No form context found when parsing %{fld5->} tag, referer: %{web_referer}", processor_chain([
	setc("eventcategory","1601020000"),
	setc("result","No form context found"),
	dup2,
	dup3,
]));

var msg82 = msg("reverseproxy:44", part90);

var part91 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] (111)'), Field(result,false), Constant(': AH00957: '), Field(network_service,false), Constant(': attempt to connect to '), Field(daddr,false), Constant(':'), Field(dport,true), Constant(' ('), Field(fld5,false), Constant(') failed')}"
match("MESSAGE#82:reverseproxy:45", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] (111)%{result}: AH00957: %{network_service}: attempt to connect to %{daddr}:%{dport->} (%{fld5}) failed", processor_chain([
	dup25,
	setc("event_id","AH00957"),
	dup2,
	dup3,
]));

var msg83 = msg("reverseproxy:45", part91);

var part92 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] AH00959: ap_proxy_connect_backend disabling worker for ('), Field(daddr,false), Constant(') for '), Field(processing_time,false), Constant('s')}"
match("MESSAGE#83:reverseproxy:46", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] AH00959: ap_proxy_connect_backend disabling worker for (%{daddr}) for %{processing_time}s", processor_chain([
	dup16,
	setc("event_id","AH00959"),
	setc("result","disabling worker"),
	dup2,
	dup3,
]));

var msg84 = msg("reverseproxy:46", part92);

var part93 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ['), Field(fld5,false), Constant('] not all the file sent to the client: '), Field(fld6,false), Constant(', referer: '), Field(web_referer,false)}"
match("MESSAGE#84:reverseproxy:47", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] [%{fld5}] not all the file sent to the client: %{fld6}, referer: %{web_referer}", processor_chain([
	setc("eventcategory","1801000000"),
	setc("context","Not all file sent to client"),
	dup2,
	dup3,
]));

var msg85 = msg("reverseproxy:47", part93);

var part94 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] AH01114: '), Field(network_service,false), Constant(': failed to make connection to backend: '), Field(daddr,false), Constant(', referer: '), Field(web_referer,false)}"
match("MESSAGE#85:reverseproxy:48", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] AH01114: %{network_service}: failed to make connection to backend: %{daddr}, referer: %{web_referer}", processor_chain([
	dup25,
	dup31,
	dup32,
	dup2,
	dup3,
]));

var msg86 = msg("reverseproxy:48", part94);

var part95 = // "Pattern{Constant('['), Field(fld3,false), Constant('] ['), Field(event_log,false), Constant(':'), Field(severity,false), Constant('] [pid '), Field(process_id,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] AH01114: '), Field(network_service,false), Constant(': failed to make connection to backend: '), Field(daddr,false)}"
match("MESSAGE#86:reverseproxy:49", "nwparser.payload", "[%{fld3}] [%{event_log}:%{severity}] [pid %{process_id}:%{fld4}] [client %{gateway}] AH01114: %{network_service}: failed to make connection to backend: %{daddr}", processor_chain([
	dup25,
	dup31,
	dup32,
	dup2,
	dup3,
]));

var msg87 = msg("reverseproxy:49", part95);

var part96 = tagval("MESSAGE#87:reverseproxy:05", "nwparser.payload", tvm, {
	"cookie": "web_cookie",
	"exceptions": "policy_waiver",
	"extra": "info",
	"host": "dhost",
	"id": "policy_id",
	"localip": "fld3",
	"method": "web_method",
	"reason": "comments",
	"referer": "web_referer",
	"server": "daddr",
	"set-cookie": "fld5",
	"size": "fld4",
	"srcip": "saddr",
	"statuscode": "resultcode",
	"time": "processing_time",
	"url": "web_root",
	"user": "username",
}, processor_chain([
	setc("eventcategory","1802000000"),
	dup2,
	dup3,
]));

var msg88 = msg("reverseproxy:05", part96);

var select8 = linear_select([
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
	msg82,
	msg83,
	msg84,
	msg85,
	msg86,
	msg87,
	msg88,
]);

var part97 = tagval("MESSAGE#88:confd-sync", "nwparser.payload", tvm, {
	"id": "fld5",
	"name": "event_description",
	"severity": "severity",
	"sub": "service",
	"sys": "fld2",
}, processor_chain([
	dup1,
	dup11,
	dup2,
]));

var msg89 = msg("confd-sync", part97);

var part98 = tagval("MESSAGE#89:confd:01", "nwparser.payload", tvm, {
	"account": "logon_id",
	"attributes": "obj_name",
	"class": "group_object",
	"client": "fld3",
	"count": "fld4",
	"facility": "logon_type",
	"id": "fld1",
	"name": "event_description",
	"node": "node",
	"object": "fld6",
	"severity": "severity",
	"srcip": "saddr",
	"storage": "directory",
	"sub": "service",
	"sys": "fld2",
	"type": "obj_type",
	"user": "username",
	"version": "version",
}, processor_chain([
	dup1,
	dup11,
	dup2,
]));

var msg90 = msg("confd:01", part98);

var part99 = // "Pattern{Constant('Frox started'), Field(,false)}"
match("MESSAGE#90:frox", "nwparser.payload", "Frox started%{}", processor_chain([
	dup12,
	setc("event_description","frox:FTP Proxy Frox started."),
	dup11,
	dup2,
]));

var msg91 = msg("frox", part99);

var part100 = // "Pattern{Constant('Listening on '), Field(saddr,false), Constant(':'), Field(sport,false)}"
match("MESSAGE#91:frox:01", "nwparser.payload", "Listening on %{saddr}:%{sport}", processor_chain([
	dup12,
	setc("event_description","frox:FTP Proxy listening on port."),
	dup11,
	dup2,
]));

var msg92 = msg("frox:01", part100);

var part101 = // "Pattern{Constant('Dropped privileges'), Field(,false)}"
match("MESSAGE#92:frox:02", "nwparser.payload", "Dropped privileges%{}", processor_chain([
	dup12,
	setc("event_description","frox:FTP Proxy dropped priveleges."),
	dup11,
	dup2,
]));

var msg93 = msg("frox:02", part101);

var select9 = linear_select([
	msg91,
	msg92,
	msg93,
]);

var part102 = // "Pattern{Constant('Classifier configuration reloaded successfully'), Field(,false)}"
match("MESSAGE#93:afcd", "nwparser.payload", "Classifier configuration reloaded successfully%{}", processor_chain([
	dup12,
	setc("event_description","afcd: IM/P2P Classifier configuration reloaded successfully."),
	dup11,
	dup2,
]));

var msg94 = msg("afcd", part102);

var part103 = // "Pattern{Constant('Starting strongSwan '), Field(fld2,true), Constant(' IPsec [starter]...')}"
match("MESSAGE#94:ipsec_starter", "nwparser.payload", "Starting strongSwan %{fld2->} IPsec [starter]...", processor_chain([
	dup12,
	setc("event_description","ipsec_starter: Starting strongSwan 4.2.3 IPsec [starter]..."),
	dup11,
	dup2,
]));

var msg95 = msg("ipsec_starter", part103);

var part104 = // "Pattern{Constant('IP address or index of physical interface changed -> reinit of ipsec interface'), Field(,false)}"
match("MESSAGE#95:ipsec_starter:01", "nwparser.payload", "IP address or index of physical interface changed -> reinit of ipsec interface%{}", processor_chain([
	dup12,
	setc("event_description","ipsec_starter: IP address or index of physical interface changed."),
	dup11,
	dup2,
]));

var msg96 = msg("ipsec_starter:01", part104);

var select10 = linear_select([
	msg95,
	msg96,
]);

var part105 = // "Pattern{Constant('Starting Pluto ('), Field(info,false), Constant(')')}"
match("MESSAGE#96:pluto", "nwparser.payload", "Starting Pluto (%{info})", processor_chain([
	dup12,
	setc("event_description","pluto: Starting Pluto."),
	dup11,
	dup2,
]));

var msg97 = msg("pluto", part105);

var part106 = // "Pattern{Constant('including NAT-Traversal patch ('), Field(info,false), Constant(')')}"
match("MESSAGE#97:pluto:01", "nwparser.payload", "including NAT-Traversal patch (%{info})", processor_chain([
	dup12,
	setc("event_description","pluto: including NAT-Traversal patch."),
	dup11,
	dup2,
]));

var msg98 = msg("pluto:01", part106);

var part107 = // "Pattern{Constant('ike_alg: Activating '), Field(info,true), Constant(' encryption: Ok')}"
match("MESSAGE#98:pluto:02", "nwparser.payload", "ike_alg: Activating %{info->} encryption: Ok", processor_chain([
	dup33,
	setc("event_description","pluto: Activating encryption algorithm."),
	dup11,
	dup2,
]));

var msg99 = msg("pluto:02", part107);

var part108 = // "Pattern{Constant('ike_alg: Activating '), Field(info,true), Constant(' hash: Ok')}"
match("MESSAGE#99:pluto:03", "nwparser.payload", "ike_alg: Activating %{info->} hash: Ok", processor_chain([
	dup33,
	setc("event_description","pluto: Activating hash algorithm."),
	dup11,
	dup2,
]));

var msg100 = msg("pluto:03", part108);

var part109 = // "Pattern{Constant('Testing registered IKE encryption algorithms:'), Field(,false)}"
match("MESSAGE#100:pluto:04", "nwparser.payload", "Testing registered IKE encryption algorithms:%{}", processor_chain([
	dup12,
	setc("event_description","pluto: Testing registered IKE encryption algorithms"),
	dup11,
	dup2,
]));

var msg101 = msg("pluto:04", part109);

var part110 = // "Pattern{Field(info,true), Constant(' self-test not available')}"
match("MESSAGE#101:pluto:05", "nwparser.payload", "%{info->} self-test not available", processor_chain([
	dup12,
	setc("event_description","pluto: Algorithm self-test not available."),
	dup11,
	dup2,
]));

var msg102 = msg("pluto:05", part110);

var part111 = // "Pattern{Field(info,true), Constant(' self-test passed')}"
match("MESSAGE#102:pluto:06", "nwparser.payload", "%{info->} self-test passed", processor_chain([
	dup12,
	setc("event_description","pluto: Algorithm self-test passed."),
	dup11,
	dup2,
]));

var msg103 = msg("pluto:06", part111);

var part112 = // "Pattern{Constant('Using KLIPS IPsec interface code'), Field(,false)}"
match("MESSAGE#103:pluto:07", "nwparser.payload", "Using KLIPS IPsec interface code%{}", processor_chain([
	dup12,
	setc("event_description","pluto: Using KLIPS IPsec interface code"),
	dup11,
	dup2,
]));

var msg104 = msg("pluto:07", part112);

var part113 = // "Pattern{Constant('adding interface '), Field(interface,true), Constant(' '), Field(saddr,false), Constant(':'), Field(sport,false)}"
match("MESSAGE#104:pluto:08", "nwparser.payload", "adding interface %{interface->} %{saddr}:%{sport}", processor_chain([
	dup12,
	setc("event_description","pluto: adding interface"),
	dup11,
	dup2,
]));

var msg105 = msg("pluto:08", part113);

var part114 = // "Pattern{Constant('loading secrets from "'), Field(filename,false), Constant('"')}"
match("MESSAGE#105:pluto:09", "nwparser.payload", "loading secrets from \"%{filename}\"", processor_chain([
	dup34,
	setc("event_description","pluto: loading secrets"),
	dup11,
	dup2,
]));

var msg106 = msg("pluto:09", part114);

var part115 = // "Pattern{Constant('loaded private key file ''), Field(filename,false), Constant('' ('), Field(filename_size,true), Constant(' bytes)')}"
match("MESSAGE#106:pluto:10", "nwparser.payload", "loaded private key file '%{filename}' (%{filename_size->} bytes)", processor_chain([
	dup34,
	setc("event_description","pluto: loaded private key file"),
	dup11,
	dup2,
]));

var msg107 = msg("pluto:10", part115);

var part116 = // "Pattern{Constant('added connection description "'), Field(fld2,false), Constant('"')}"
match("MESSAGE#107:pluto:11", "nwparser.payload", "added connection description \"%{fld2}\"", processor_chain([
	dup12,
	setc("event_description","pluto: added connection description"),
	dup11,
	dup2,
]));

var msg108 = msg("pluto:11", part116);

var part117 = // "Pattern{Constant('"'), Field(fld2,false), Constant('" #'), Field(fld3,false), Constant(': initiating Main Mode')}"
match("MESSAGE#108:pluto:12", "nwparser.payload", "\"%{fld2}\" #%{fld3}: initiating Main Mode", processor_chain([
	dup12,
	dup35,
	dup11,
	dup2,
]));

var msg109 = msg("pluto:12", part117);

var part118 = // "Pattern{Constant('"'), Field(fld2,false), Constant('" #'), Field(fld3,false), Constant(': max number of retransmissions ('), Field(fld4,false), Constant(') reached STATE_MAIN_I1. No response (or no acceptable response) to our first IKE message')}"
match("MESSAGE#109:pluto:13", "nwparser.payload", "\"%{fld2}\" #%{fld3}: max number of retransmissions (%{fld4}) reached STATE_MAIN_I1. No response (or no acceptable response) to our first IKE message", processor_chain([
	dup10,
	dup36,
	dup11,
	dup2,
]));

var msg110 = msg("pluto:13", part118);

var part119 = // "Pattern{Constant('"'), Field(fld2,false), Constant('" #'), Field(fld3,false), Constant(': starting keying attempt '), Field(fld4,true), Constant(' of an unlimited number')}"
match("MESSAGE#110:pluto:14", "nwparser.payload", "\"%{fld2}\" #%{fld3}: starting keying attempt %{fld4->} of an unlimited number", processor_chain([
	dup12,
	dup37,
	dup11,
	dup2,
]));

var msg111 = msg("pluto:14", part119);

var part120 = // "Pattern{Constant('forgetting secrets'), Field(,false)}"
match("MESSAGE#111:pluto:15", "nwparser.payload", "forgetting secrets%{}", processor_chain([
	dup12,
	setc("event_description","pluto:forgetting secrets"),
	dup11,
	dup2,
]));

var msg112 = msg("pluto:15", part120);

var part121 = // "Pattern{Constant('Changing to directory ''), Field(directory,false), Constant(''')}"
match("MESSAGE#112:pluto:17", "nwparser.payload", "Changing to directory '%{directory}'", processor_chain([
	dup12,
	setc("event_description","pluto:Changing to directory"),
	dup11,
	dup2,
]));

var msg113 = msg("pluto:17", part121);

var part122 = // "Pattern{Constant('| *time to handle event'), Field(,false)}"
match("MESSAGE#113:pluto:18", "nwparser.payload", "| *time to handle event%{}", processor_chain([
	dup12,
	setc("event_description","pluto:*time to handle event"),
	dup11,
	dup2,
]));

var msg114 = msg("pluto:18", part122);

var part123 = // "Pattern{Constant('| *received kernel message'), Field(,false)}"
match("MESSAGE#114:pluto:19", "nwparser.payload", "| *received kernel message%{}", processor_chain([
	dup12,
	setc("event_description","pluto:*received kernel message"),
	dup11,
	dup2,
]));

var msg115 = msg("pluto:19", part123);

var part124 = // "Pattern{Constant('| rejected packet:'), Field(,false)}"
match("MESSAGE#115:pluto:20", "nwparser.payload", "| rejected packet:%{}", processor_chain([
	dup25,
	setc("event_description","pluto:rejected packet"),
	dup11,
	dup2,
]));

var msg116 = msg("pluto:20", part124);

var part125 = // "Pattern{Constant('| next event '), Field(event_type,true), Constant(' in '), Field(fld2,true), Constant(' seconds for #'), Field(fld3,false)}"
match("MESSAGE#116:pluto:21", "nwparser.payload", "| next event %{event_type->} in %{fld2->} seconds for #%{fld3}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg117 = msg("pluto:21", part125);

var part126 = // "Pattern{Constant('| next event '), Field(event_type,true), Constant(' in '), Field(fld2,true), Constant(' seconds')}"
match("MESSAGE#117:pluto:22", "nwparser.payload", "| next event %{event_type->} in %{fld2->} seconds", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg118 = msg("pluto:22", part126);

var part127 = // "Pattern{Constant('| inserting event '), Field(event_type,true), Constant(' in '), Field(fld2,true), Constant(' seconds for #'), Field(fld3,false)}"
match("MESSAGE#118:pluto:23", "nwparser.payload", "| inserting event %{event_type->} in %{fld2->} seconds for #%{fld3}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg119 = msg("pluto:23", part127);

var part128 = // "Pattern{Constant('| event after this is '), Field(event_type,true), Constant(' in '), Field(fld2,true), Constant(' seconds')}"
match("MESSAGE#119:pluto:24", "nwparser.payload", "| event after this is %{event_type->} in %{fld2->} seconds", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg120 = msg("pluto:24", part128);

var part129 = // "Pattern{Constant('| recent '), Field(action,true), Constant(' activity '), Field(fld2,true), Constant(' seconds ago, '), Field(info,false)}"
match("MESSAGE#120:pluto:25", "nwparser.payload", "| recent %{action->} activity %{fld2->} seconds ago, %{info}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg121 = msg("pluto:25", part129);

var part130 = // "Pattern{Constant('| *received '), Field(rbytes,true), Constant(' bytes from '), Field(saddr,false), Constant(':'), Field(sport,true), Constant(' on '), Field(dinterface,false)}"
match("MESSAGE#121:pluto:26", "nwparser.payload", "| *received %{rbytes->} bytes from %{saddr}:%{sport->} on %{dinterface}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg122 = msg("pluto:26", part130);

var part131 = // "Pattern{Constant('| received '), Field(action,true), Constant(' notification '), Field(msg,true), Constant(' with seqno = '), Field(fld2,false)}"
match("MESSAGE#122:pluto:27", "nwparser.payload", "| received %{action->} notification %{msg->} with seqno = %{fld2}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg123 = msg("pluto:27", part131);

var part132 = // "Pattern{Constant('| sent '), Field(action,true), Constant(' notification '), Field(msg,true), Constant(' with seqno = '), Field(fld2,false)}"
match("MESSAGE#123:pluto:28", "nwparser.payload", "| sent %{action->} notification %{msg->} with seqno = %{fld2}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg124 = msg("pluto:28", part132);

var part133 = // "Pattern{Constant('| inserting event '), Field(event_type,false), Constant(', timeout in '), Field(fld2,true), Constant(' seconds')}"
match("MESSAGE#124:pluto:29", "nwparser.payload", "| inserting event %{event_type}, timeout in %{fld2->} seconds", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg125 = msg("pluto:29", part133);

var part134 = // "Pattern{Constant('| handling event '), Field(event_type,true), Constant(' for '), Field(saddr,true), Constant(' "'), Field(fld2,false), Constant('" #'), Field(fld3,false)}"
match("MESSAGE#125:pluto:30", "nwparser.payload", "| handling event %{event_type->} for %{saddr->} \"%{fld2}\" #%{fld3}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg126 = msg("pluto:30", part134);

var part135 = // "Pattern{Constant('| '), Field(event_description,false)}"
match("MESSAGE#126:pluto:31", "nwparser.payload", "| %{event_description}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg127 = msg("pluto:31", part135);

var part136 = // "Pattern{Field(fld2,false), Constant(': asynchronous network error report on '), Field(interface,true), Constant(' for message to '), Field(daddr,true), Constant(' port '), Field(dport,false), Constant(', complainant '), Field(saddr,false), Constant(': Connection refused [errno '), Field(fld4,false), Constant(', origin ICMP type '), Field(icmptype,true), Constant(' code '), Field(icmpcode,true), Constant(' (not authenticated)]')}"
match("MESSAGE#127:pluto:32", "nwparser.payload", "%{fld2}: asynchronous network error report on %{interface->} for message to %{daddr->} port %{dport}, complainant %{saddr}: Connection refused [errno %{fld4}, origin ICMP type %{icmptype->} code %{icmpcode->} (not authenticated)]", processor_chain([
	dup12,
	setc("event_description","not authenticated"),
	dup11,
	dup2,
]));

var msg128 = msg("pluto:32", part136);

var part137 = // "Pattern{Constant('"'), Field(fld2,false), Constant('"['), Field(fld4,false), Constant('] '), Field(saddr,true), Constant(' #'), Field(fld3,false), Constant(': initiating Main Mode')}"
match("MESSAGE#128:pluto:33", "nwparser.payload", "\"%{fld2}\"[%{fld4}] %{saddr->} #%{fld3}: initiating Main Mode", processor_chain([
	dup12,
	dup35,
	dup11,
	dup2,
]));

var msg129 = msg("pluto:33", part137);

var part138 = // "Pattern{Constant('"'), Field(fld2,false), Constant('"['), Field(fld4,false), Constant('] '), Field(saddr,true), Constant(' #'), Field(fld3,false), Constant(': max number of retransmissions ('), Field(fld5,false), Constant(') reached STATE_MAIN_I1. No response (or no acceptable response) to our first IKE message')}"
match("MESSAGE#129:pluto:34", "nwparser.payload", "\"%{fld2}\"[%{fld4}] %{saddr->} #%{fld3}: max number of retransmissions (%{fld5}) reached STATE_MAIN_I1. No response (or no acceptable response) to our first IKE message", processor_chain([
	dup12,
	dup36,
	dup11,
	dup2,
]));

var msg130 = msg("pluto:34", part138);

var part139 = // "Pattern{Constant('"'), Field(fld2,false), Constant('"['), Field(fld4,false), Constant('] '), Field(saddr,true), Constant(' #'), Field(fld3,false), Constant(': starting keying attempt '), Field(fld5,true), Constant(' of an unlimited number')}"
match("MESSAGE#130:pluto:35", "nwparser.payload", "\"%{fld2}\"[%{fld4}] %{saddr->} #%{fld3}: starting keying attempt %{fld5->} of an unlimited number", processor_chain([
	dup12,
	dup37,
	dup11,
	dup2,
]));

var msg131 = msg("pluto:35", part139);

var select11 = linear_select([
	msg97,
	msg98,
	msg99,
	msg100,
	msg101,
	msg102,
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
	msg124,
	msg125,
	msg126,
	msg127,
	msg128,
	msg129,
	msg130,
	msg131,
]);

var part140 = // "Pattern{Constant('This binary does not support kernel L2TP.'), Field(,false)}"
match("MESSAGE#131:xl2tpd", "nwparser.payload", "This binary does not support kernel L2TP.%{}", processor_chain([
	setc("eventcategory","1607000000"),
	setc("event_description","xl2tpd:This binary does not support kernel L2TP."),
	dup11,
	dup2,
]));

var msg132 = msg("xl2tpd", part140);

var part141 = // "Pattern{Constant('xl2tpd version '), Field(version,true), Constant(' started on PID:'), Field(fld2,false)}"
match("MESSAGE#132:xl2tpd:01", "nwparser.payload", "xl2tpd version %{version->} started on PID:%{fld2}", processor_chain([
	dup12,
	setc("event_description","xl2tpd:xl2tpd started."),
	dup11,
	dup2,
]));

var msg133 = msg("xl2tpd:01", part141);

var part142 = // "Pattern{Constant('Written by '), Field(info,false)}"
match("MESSAGE#133:xl2tpd:02", "nwparser.payload", "Written by %{info}", processor_chain([
	dup12,
	dup38,
	dup11,
	dup2,
]));

var msg134 = msg("xl2tpd:02", part142);

var part143 = // "Pattern{Constant('Forked by '), Field(info,false)}"
match("MESSAGE#134:xl2tpd:03", "nwparser.payload", "Forked by %{info}", processor_chain([
	dup12,
	dup38,
	dup11,
	dup2,
]));

var msg135 = msg("xl2tpd:03", part143);

var part144 = // "Pattern{Constant('Inherited by '), Field(info,false)}"
match("MESSAGE#135:xl2tpd:04", "nwparser.payload", "Inherited by %{info}", processor_chain([
	dup12,
	dup38,
	dup11,
	dup2,
]));

var msg136 = msg("xl2tpd:04", part144);

var part145 = // "Pattern{Constant('Listening on IP address '), Field(saddr,false), Constant(', port '), Field(sport,false)}"
match("MESSAGE#136:xl2tpd:05", "nwparser.payload", "Listening on IP address %{saddr}, port %{sport}", processor_chain([
	dup12,
	dup38,
	dup11,
	dup2,
]));

var msg137 = msg("xl2tpd:05", part145);

var select12 = linear_select([
	msg132,
	msg133,
	msg134,
	msg135,
	msg136,
	msg137,
]);

var part146 = // "Pattern{Constant('Exiting'), Field(,false)}"
match("MESSAGE#137:barnyard:01", "nwparser.payload", "Exiting%{}", processor_chain([
	dup12,
	setc("event_description","barnyard: Exiting"),
	dup11,
	dup2,
]));

var msg138 = msg("barnyard:01", part146);

var part147 = // "Pattern{Constant('Initializing daemon mode'), Field(,false)}"
match("MESSAGE#138:barnyard:02", "nwparser.payload", "Initializing daemon mode%{}", processor_chain([
	dup12,
	setc("event_description","barnyard:Initializing daemon mode"),
	dup11,
	dup2,
]));

var msg139 = msg("barnyard:02", part147);

var part148 = // "Pattern{Constant('Opened spool file ''), Field(filename,false), Constant(''')}"
match("MESSAGE#139:barnyard:03", "nwparser.payload", "Opened spool file '%{filename}'", processor_chain([
	dup12,
	setc("event_description","barnyard:Opened spool file."),
	dup11,
	dup2,
]));

var msg140 = msg("barnyard:03", part148);

var part149 = // "Pattern{Constant('Waiting for new data'), Field(,false)}"
match("MESSAGE#140:barnyard:04", "nwparser.payload", "Waiting for new data%{}", processor_chain([
	dup12,
	setc("event_description","barnyard:Waiting for new data"),
	dup11,
	dup2,
]));

var msg141 = msg("barnyard:04", part149);

var select13 = linear_select([
	msg138,
	msg139,
	msg140,
	msg141,
]);

var part150 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' SMTP connection from localhost ('), Field(hostname,false), Constant(') ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' closed by QUIT')}"
match("MESSAGE#141:exim:01", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} SMTP connection from localhost (%{hostname}) [%{saddr}]:%{sport->} closed by QUIT", processor_chain([
	dup12,
	setc("event_description","exim:SMTP connection from localhost closed by QUIT"),
	dup11,
	dup2,
]));

var msg142 = msg("exim:01", part150);

var part151 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' ['), Field(saddr,false), Constant('] F=<<'), Field(from,false), Constant('> R=<<'), Field(to,false), Constant('> Accepted: '), Field(info,false)}"
match("MESSAGE#142:exim:02", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} [%{saddr}] F=\u003c\u003c%{from}> R=\u003c\u003c%{to}> Accepted: %{info}", processor_chain([
	setc("eventcategory","1207010000"),
	setc("event_description","exim:e-mail accepted from relay."),
	dup11,
	dup2,
]));

var msg143 = msg("exim:02", part151);

var part152 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' '), Field(fld8,true), Constant(' <<= '), Field(from,true), Constant(' H=localhost ('), Field(hostname,false), Constant(') ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' P='), Field(protocol,true), Constant(' S='), Field(fld9,true), Constant(' id='), Field(info,false)}"
match("MESSAGE#143:exim:03", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} %{fld8->} \u003c\u003c= %{from->} H=localhost (%{hostname}) [%{saddr}]:%{sport->} P=%{protocol->} S=%{fld9->} id=%{info}", processor_chain([
	setc("eventcategory","1207000000"),
	setc("event_description","exim: e-mail sent."),
	dup11,
	dup2,
]));

var msg144 = msg("exim:03", part152);

var part153 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' '), Field(fld8,true), Constant(' == '), Field(from,true), Constant(' R=dnslookup defer ('), Field(fld9,false), Constant('): host lookup did not complete')}"
match("MESSAGE#144:exim:04", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} %{fld8->} == %{from->} R=dnslookup defer (%{fld9}): host lookup did not complete", processor_chain([
	dup39,
	setc("event_description","exim: e-mail host lookup did not complete in DNS."),
	dup11,
	dup2,
]));

var msg145 = msg("exim:04", part153);

var part154 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' '), Field(fld8,true), Constant(' == '), Field(from,true), Constant(' routing defer ('), Field(fld9,false), Constant('): retry time not reached')}"
match("MESSAGE#145:exim:05", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} %{fld8->} == %{from->} routing defer (%{fld9}): retry time not reached", processor_chain([
	dup39,
	setc("event_description","exim: e-mail routing defer:retry time not reached."),
	dup11,
	dup2,
]));

var msg146 = msg("exim:05", part154);

var part155 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' exim '), Field(version,true), Constant(' daemon started: pid='), Field(fld8,false), Constant(', no queue runs, listening for SMTP on port '), Field(sport,true), Constant(' ('), Field(info,false), Constant(') port '), Field(fld9,true), Constant(' ('), Field(fld10,false), Constant(') and for SMTPS on port '), Field(fld11,true), Constant(' ('), Field(fld12,false), Constant(')')}"
match("MESSAGE#146:exim:06", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} exim %{version->} daemon started: pid=%{fld8}, no queue runs, listening for SMTP on port %{sport->} (%{info}) port %{fld9->} (%{fld10}) and for SMTPS on port %{fld11->} (%{fld12})", processor_chain([
	dup12,
	setc("event_description","exim: exim daemon started."),
	dup11,
	dup2,
]));

var msg147 = msg("exim:06", part155);

var part156 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' Start queue run: pid='), Field(fld8,false)}"
match("MESSAGE#147:exim:07", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} Start queue run: pid=%{fld8}", processor_chain([
	dup12,
	setc("event_description","exim: Start queue run."),
	dup11,
	dup2,
]));

var msg148 = msg("exim:07", part156);

var part157 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' pid '), Field(fld8,false), Constant(': SIGHUP received: re-exec daemon')}"
match("MESSAGE#148:exim:08", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} pid %{fld8}: SIGHUP received: re-exec daemon", processor_chain([
	dup12,
	setc("event_description","exim: SIGHUP received: re-exec daemon."),
	dup11,
	dup2,
]));

var msg149 = msg("exim:08", part157);

var part158 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' SMTP connection from ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' '), Field(info,false)}"
match("MESSAGE#149:exim:09", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} SMTP connection from [%{saddr}]:%{sport->} %{info}", processor_chain([
	dup12,
	setc("event_description","exim: SMTP connection from host."),
	dup11,
	dup2,
]));

var msg150 = msg("exim:09", part158);

var part159 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' rejected EHLO from ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' '), Field(info,false)}"
match("MESSAGE#150:exim:10", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} rejected EHLO from [%{saddr}]:%{sport->} %{info}", processor_chain([
	dup12,
	setc("event_description","exim:rejected EHLO from host."),
	dup11,
	dup2,
]));

var msg151 = msg("exim:10", part159);

var part160 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' SMTP protocol synchronization error ('), Field(result,false), Constant('): '), Field(fld8,true), Constant(' H=['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' '), Field(info,false)}"
match("MESSAGE#151:exim:11", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} SMTP protocol synchronization error (%{result}): %{fld8->} H=[%{saddr}]:%{sport->} %{info}", processor_chain([
	dup12,
	setc("event_description","exim:SMTP protocol synchronization error rejected connection from host."),
	dup11,
	dup2,
]));

var msg152 = msg("exim:11", part160);

var part161 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' TLS error on connection from ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' '), Field(info,false)}"
match("MESSAGE#152:exim:12", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} TLS error on connection from [%{saddr}]:%{sport->} %{info}", processor_chain([
	dup12,
	setc("event_description","exim:TLS error on connection from host."),
	dup11,
	dup2,
]));

var msg153 = msg("exim:12", part161);

var part162 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' '), Field(fld10,true), Constant(' == '), Field(hostname,true), Constant(' R='), Field(fld8,true), Constant(' T='), Field(fld9,false), Constant(': '), Field(info,false)}"
match("MESSAGE#153:exim:13", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} %{fld10->} == %{hostname->} R=%{fld8->} T=%{fld9}: %{info}", processor_chain([
	dup12,
	dup40,
	dup11,
	dup2,
]));

var msg154 = msg("exim:13", part162);

var part163 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' '), Field(fld10,true), Constant(' '), Field(hostname,true), Constant(' ['), Field(saddr,false), Constant(']:'), Field(sport,true), Constant(' '), Field(info,false)}"
match("MESSAGE#154:exim:14", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} %{fld10->} %{hostname->} [%{saddr}]:%{sport->} %{info}", processor_chain([
	dup12,
	dup40,
	dup11,
	dup2,
]));

var msg155 = msg("exim:14", part163);

var part164 = // "Pattern{Field(fld2,false), Constant('-'), Field(fld3,false), Constant('-'), Field(fld4,true), Constant(' '), Field(fld5,false), Constant(':'), Field(fld6,false), Constant(':'), Field(fld7,true), Constant(' End queue run: '), Field(info,false)}"
match("MESSAGE#155:exim:15", "nwparser.payload", "%{fld2}-%{fld3}-%{fld4->} %{fld5}:%{fld6}:%{fld7->} End queue run: %{info}", processor_chain([
	dup12,
	dup40,
	dup11,
	dup2,
]));

var msg156 = msg("exim:15", part164);

var part165 = // "Pattern{Field(fld2,true), Constant(' '), Field(fld3,false)}"
match("MESSAGE#156:exim:16", "nwparser.payload", "%{fld2->} %{fld3}", processor_chain([
	dup12,
	dup11,
	dup2,
]));

var msg157 = msg("exim:16", part165);

var select14 = linear_select([
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
]);

var part166 = // "Pattern{Constant('QMGR['), Field(fld2,false), Constant(']: '), Field(fld3,true), Constant(' moved to work queue')}"
match("MESSAGE#157:smtpd:01", "nwparser.payload", "QMGR[%{fld2}]: %{fld3->} moved to work queue", processor_chain([
	dup12,
	setc("event_description","smtpd: Process moved to work queue."),
	dup11,
	dup2,
]));

var msg158 = msg("smtpd:01", part166);

var part167 = // "Pattern{Constant('SCANNER['), Field(fld3,false), Constant(']: id="1000" severity="'), Field(severity,false), Constant('" sys="'), Field(fld4,false), Constant('" sub="'), Field(service,false), Constant('" name="'), Field(event_description,false), Constant('" srcip="'), Field(saddr,false), Constant('" from="'), Field(from,false), Constant('" to="'), Field(to,false), Constant('" subject="'), Field(subject,false), Constant('" queueid="'), Field(fld5,false), Constant('" size="'), Field(rbytes,false), Constant('"')}"
match("MESSAGE#158:smtpd:02", "nwparser.payload", "SCANNER[%{fld3}]: id=\"1000\" severity=\"%{severity}\" sys=\"%{fld4}\" sub=\"%{service}\" name=\"%{event_description}\" srcip=\"%{saddr}\" from=\"%{from}\" to=\"%{to}\" subject=\"%{subject}\" queueid=\"%{fld5}\" size=\"%{rbytes}\"", processor_chain([
	setc("eventcategory","1207010100"),
	dup11,
	dup2,
]));

var msg159 = msg("smtpd:02", part167);

var part168 = // "Pattern{Constant('SCANNER['), Field(fld3,false), Constant(']: Nothing to do, exiting.')}"
match("MESSAGE#159:smtpd:03", "nwparser.payload", "SCANNER[%{fld3}]: Nothing to do, exiting.", processor_chain([
	dup12,
	setc("event_description","smtpd: SCANNER: Nothing to do,exiting."),
	dup11,
	dup2,
]));

var msg160 = msg("smtpd:03", part168);

var part169 = // "Pattern{Constant('MASTER['), Field(fld3,false), Constant(']: QR globally disabled, status two set to 'disabled'')}"
match("MESSAGE#160:smtpd:04", "nwparser.payload", "MASTER[%{fld3}]: QR globally disabled, status two set to 'disabled'", processor_chain([
	dup12,
	setc("event_description","smtpd: MASTER:QR globally disabled, status two set to disabled."),
	dup11,
	dup2,
]));

var msg161 = msg("smtpd:04", part169);

var part170 = // "Pattern{Constant('MASTER['), Field(fld3,false), Constant(']: QR globally disabled, status one set to 'disabled'')}"
match("MESSAGE#161:smtpd:07", "nwparser.payload", "MASTER[%{fld3}]: QR globally disabled, status one set to 'disabled'", processor_chain([
	dup12,
	setc("event_description","smtpd: MASTER:QR globally disabled, status one set to disabled."),
	dup11,
	dup2,
]));

var msg162 = msg("smtpd:07", part170);

var part171 = // "Pattern{Constant('MASTER['), Field(fld3,false), Constant(']: (Re-)loading configuration from Confd')}"
match("MESSAGE#162:smtpd:05", "nwparser.payload", "MASTER[%{fld3}]: (Re-)loading configuration from Confd", processor_chain([
	dup12,
	setc("event_description","smtpd: MASTER:(Re-)loading configuration from Confd."),
	dup11,
	dup2,
]));

var msg163 = msg("smtpd:05", part171);

var part172 = // "Pattern{Constant('MASTER['), Field(fld3,false), Constant(']: Sending QR one')}"
match("MESSAGE#163:smtpd:06", "nwparser.payload", "MASTER[%{fld3}]: Sending QR one", processor_chain([
	dup12,
	setc("event_description","smtpd: MASTER:Sending QR one."),
	dup11,
	dup2,
]));

var msg164 = msg("smtpd:06", part172);

var select15 = linear_select([
	msg158,
	msg159,
	msg160,
	msg161,
	msg162,
	msg163,
	msg164,
]);

var part173 = // "Pattern{Constant('Did not receive identification string from '), Field(fld18,false)}"
match("MESSAGE#164:sshd:01", "nwparser.payload", "Did not receive identification string from %{fld18}", processor_chain([
	dup10,
	setc("event_description","sshd: Did not receive identification string."),
	dup11,
	dup2,
]));

var msg165 = msg("sshd:01", part173);

var part174 = // "Pattern{Constant('Received SIGHUP; restarting.'), Field(,false)}"
match("MESSAGE#165:sshd:02", "nwparser.payload", "Received SIGHUP; restarting.%{}", processor_chain([
	dup12,
	setc("event_description","sshd:Received SIGHUP restarting."),
	dup11,
	dup2,
]));

var msg166 = msg("sshd:02", part174);

var part175 = // "Pattern{Constant('Server listening on '), Field(saddr,true), Constant(' port '), Field(sport,false), Constant('.')}"
match("MESSAGE#166:sshd:03", "nwparser.payload", "Server listening on %{saddr->} port %{sport}.", processor_chain([
	dup12,
	setc("event_description","sshd:Server listening; restarting."),
	dup11,
	dup2,
]));

var msg167 = msg("sshd:03", part175);

var part176 = // "Pattern{Constant('Invalid user admin from '), Field(fld18,false)}"
match("MESSAGE#167:sshd:04", "nwparser.payload", "Invalid user admin from %{fld18}", processor_chain([
	dup41,
	setc("event_description","sshd:Invalid user admin."),
	dup11,
	dup2,
]));

var msg168 = msg("sshd:04", part176);

var part177 = // "Pattern{Constant('Failed none for invalid user admin from '), Field(saddr,true), Constant(' port '), Field(sport,true), Constant(' '), Field(fld3,false)}"
match("MESSAGE#168:sshd:05", "nwparser.payload", "Failed none for invalid user admin from %{saddr->} port %{sport->} %{fld3}", processor_chain([
	dup41,
	setc("event_description","sshd:Failed none for invalid user admin."),
	dup11,
	dup2,
]));

var msg169 = msg("sshd:05", part177);

var part178 = // "Pattern{Constant('error: Could not get shadow information for NOUSER'), Field(,false)}"
match("MESSAGE#169:sshd:06", "nwparser.payload", "error: Could not get shadow information for NOUSER%{}", processor_chain([
	dup10,
	setc("event_description","sshd:error:Could not get shadow information for NOUSER"),
	dup11,
	dup2,
]));

var msg170 = msg("sshd:06", part178);

var part179 = // "Pattern{Constant('Failed password for root from '), Field(saddr,true), Constant(' port '), Field(sport,true), Constant(' '), Field(fld3,false)}"
match("MESSAGE#170:sshd:07", "nwparser.payload", "Failed password for root from %{saddr->} port %{sport->} %{fld3}", processor_chain([
	dup41,
	setc("event_description","sshd:Failed password for root."),
	dup11,
	dup2,
]));

var msg171 = msg("sshd:07", part179);

var part180 = // "Pattern{Constant('Accepted password for loginuser from '), Field(saddr,true), Constant(' port '), Field(sport,true), Constant(' '), Field(fld3,false)}"
match("MESSAGE#171:sshd:08", "nwparser.payload", "Accepted password for loginuser from %{saddr->} port %{sport->} %{fld3}", processor_chain([
	setc("eventcategory","1302000000"),
	setc("event_description","sshd:Accepted password for loginuser."),
	dup11,
	dup2,
]));

var msg172 = msg("sshd:08", part180);

var part181 = // "Pattern{Constant('subsystem request for sftp failed, subsystem not found'), Field(,false)}"
match("MESSAGE#172:sshd:09", "nwparser.payload", "subsystem request for sftp failed, subsystem not found%{}", processor_chain([
	dup10,
	setc("event_description","sshd:subsystem request for sftp failed,subsystem not found."),
	dup11,
	dup2,
]));

var msg173 = msg("sshd:09", part181);

var select16 = linear_select([
	msg165,
	msg166,
	msg167,
	msg168,
	msg169,
	msg170,
	msg171,
	msg172,
	msg173,
]);

var part182 = tagval("MESSAGE#173:aua:01", "nwparser.payload", tvm, {
	"caller": "fld4",
	"engine": "fld5",
	"id": "fld1",
	"name": "event_description",
	"severity": "severity",
	"srcip": "saddr",
	"sub": "service",
	"sys": "fld2",
	"user": "username",
}, processor_chain([
	dup13,
	dup11,
	dup2,
	dup45,
	dup46,
]));

var msg174 = msg("aua:01", part182);

var part183 = // "Pattern{Constant('created new negotiatorchild'), Field(,false)}"
match("MESSAGE#174:sockd:01", "nwparser.payload", "created new negotiatorchild%{}", processor_chain([
	dup12,
	setc("event_description","sockd: created new negotiatorchild."),
	dup11,
	dup2,
]));

var msg175 = msg("sockd:01", part183);

var part184 = // "Pattern{Constant('dante/server '), Field(version,true), Constant(' running')}"
match("MESSAGE#175:sockd:02", "nwparser.payload", "dante/server %{version->} running", processor_chain([
	dup12,
	setc("event_description","sockd:dante/server running."),
	dup11,
	dup2,
]));

var msg176 = msg("sockd:02", part184);

var part185 = // "Pattern{Constant('sockdexit(): terminating on signal '), Field(fld2,false)}"
match("MESSAGE#176:sockd:03", "nwparser.payload", "sockdexit(): terminating on signal %{fld2}", processor_chain([
	dup12,
	setc("event_description","sockd:sockdexit():terminating on signal."),
	dup11,
	dup2,
]));

var msg177 = msg("sockd:03", part185);

var select17 = linear_select([
	msg175,
	msg176,
	msg177,
]);

var part186 = // "Pattern{Constant('Master started'), Field(,false)}"
match("MESSAGE#177:pop3proxy", "nwparser.payload", "Master started%{}", processor_chain([
	dup12,
	setc("event_description","pop3proxy:Master started."),
	dup11,
	dup2,
]));

var msg178 = msg("pop3proxy", part186);

var part187 = tagval("MESSAGE#178:astarosg_TVM", "nwparser.payload", tvm, {
	"account": "logon_id",
	"action": "action",
	"ad_domain": "fld5",
	"app-id": "fld20",
	"application": "fld19",
	"attributes": "obj_name",
	"auth": "fld15",
	"authtime": "fld9",
	"avscantime": "fld12",
	"cached": "fld7",
	"caller": "fld30",
	"category": "policy_id",
	"categoryname": "info",
	"cattime": "fld11",
	"class": "group_object",
	"client": "fld3",
	"content-type": "content_type",
	"cookie": "web_cookie",
	"count": "fld4",
	"device": "fld14",
	"dnstime": "fld10",
	"dstip": "daddr",
	"dstmac": "dmacaddr",
	"dstport": "dport",
	"engine": "fld31",
	"error": "comments",
	"exceptions": "fld17",
	"extension": "web_extension",
	"extra": "info",
	"facility": "logon_type",
	"file": "filename",
	"filename": "filename",
	"filteraction": "policyname",
	"fullreqtime": "fld13",
	"function": "action",
	"fwrule": "policy_id",
	"group": "group",
	"host": "dhost",
	"id": "rule",
	"info": "context",
	"initf": "sinterface",
	"length": "fld25",
	"line": "fld22",
	"localip": "fld31",
	"message": "context",
	"method": "web_method",
	"name": "event_description",
	"node": "node",
	"object": "fld6",
	"outitf": "dinterface",
	"prec": "fld30",
	"profile": "owner",
	"proto": "fld24",
	"reason": "comments",
	"referer": "web_referer",
	"reputation": "fld18",
	"request": "fld8",
	"seq": "fld23",
	"server": "daddr",
	"set-cookie": "fld32",
	"severity": "severity",
	"size": "filename_size",
	"srcip": "saddr",
	"srcmac": "smacaddr",
	"srcport": "sport",
	"statuscode": "resultcode",
	"storage": "directory",
	"sub": "service",
	"sys": "vsys",
	"tcpflags": "fld29",
	"time": "fld21",
	"tos": "fld26",
	"ttl": "fld28",
	"type": "obj_type",
	"ua": "fld16",
	"url": "url",
	"user": "username",
	"version": "version",
}, processor_chain([
	dup12,
	dup11,
	dup2,
	dup45,
	dup46,
]));

var msg179 = msg("astarosg_TVM", part187);

var part188 = tagval("MESSAGE#179:httpd", "nwparser.payload", tvm, {
	"account": "logon_id",
	"action": "action",
	"ad_domain": "fld5",
	"app-id": "fld20",
	"application": "fld19",
	"attributes": "obj_name",
	"auth": "fld15",
	"authtime": "fld9",
	"avscantime": "fld12",
	"cached": "fld7",
	"caller": "fld30",
	"category": "policy_id",
	"categoryname": "info",
	"cattime": "fld11",
	"class": "group_object",
	"client": "fld3",
	"content-type": "content_type",
	"cookie": "web_cookie",
	"count": "fld4",
	"device": "fld14",
	"dnstime": "fld10",
	"dstip": "daddr",
	"dstmac": "dmacaddr",
	"dstport": "dport",
	"engine": "fld31",
	"error": "comments",
	"exceptions": "fld17",
	"extension": "web_extension",
	"extra": "info",
	"facility": "logon_type",
	"file": "filename",
	"filename": "filename",
	"filteraction": "policyname",
	"fullreqtime": "fld13",
	"function": "action",
	"fwrule": "policy_id",
	"group": "group",
	"host": "dhost",
	"id": "rule",
	"info": "context",
	"initf": "sinterface",
	"length": "fld25",
	"line": "fld22",
	"localip": "fld31",
	"message": "context",
	"method": "web_method",
	"name": "event_description",
	"node": "node",
	"object": "fld6",
	"outitf": "dinterface",
	"port": "network_port",
	"prec": "fld30",
	"profile": "owner",
	"proto": "fld24",
	"query": "web_query",
	"reason": "comments",
	"referer": "web_referer",
	"reputation": "fld18",
	"request": "fld8",
	"seq": "fld23",
	"server": "daddr",
	"set-cookie": "fld32",
	"severity": "severity",
	"size": "filename_size",
	"srcip": "saddr",
	"srcmac": "smacaddr",
	"srcport": "sport",
	"statuscode": "resultcode",
	"storage": "directory",
	"sub": "service",
	"sys": "vsys",
	"tcpflags": "fld29",
	"time": "fld21",
	"tos": "fld26",
	"ttl": "fld28",
	"type": "obj_type",
	"ua": "fld16",
	"uid": "uid",
	"url": "url",
	"user": "username",
	"version": "version",
}, processor_chain([
	dup12,
	dup11,
	dup2,
	dup45,
	dup46,
]));

var msg180 = msg("httpd", part188);

var part189 = // "Pattern{Constant('['), Field(event_log,false), Constant(':'), Field(result,false), Constant('] [pid '), Field(fld3,false), Constant(':'), Field(fld4,false), Constant('] [client '), Field(gateway,false), Constant('] ModSecurity: Warning. '), Field(rulename,true), Constant(' [file "'), Field(filename,false), Constant('"] [line "'), Field(fld5,false), Constant('"] [id "'), Field(rule,false), Constant('"] [rev "'), Field(fld2,false), Constant('"] [msg "'), Field(event_description,false), Constant('"] [severity "'), Field(severity,false), Constant('"] [ver "'), Field(version,false), Constant('"] [maturity "'), Field(fld22,false), Constant('"] [accuracy "'), Field(fld23,false), Constant('"] [tag "'), Field(fld24,false), Constant('"] [hostname "'), Field(dhost,false), Constant('"] [uri "'), Field(web_root,false), Constant('"] [unique_id "'), Field(operation_id,false), Constant('"]'), Field(fld25,false)}"
match("MESSAGE#180:httpd:01", "nwparser.payload", "[%{event_log}:%{result}] [pid %{fld3}:%{fld4}] [client %{gateway}] ModSecurity: Warning. %{rulename->} [file \"%{filename}\"] [line \"%{fld5}\"] [id \"%{rule}\"] [rev \"%{fld2}\"] [msg \"%{event_description}\"] [severity \"%{severity}\"] [ver \"%{version}\"] [maturity \"%{fld22}\"] [accuracy \"%{fld23}\"] [tag \"%{fld24}\"] [hostname \"%{dhost}\"] [uri \"%{web_root}\"] [unique_id \"%{operation_id}\"]%{fld25}", processor_chain([
	setc("eventcategory","1502000000"),
	dup2,
	dup3,
]));

var msg181 = msg("httpd:01", part189);

var select18 = linear_select([
	msg180,
	msg181,
]);

var part190 = tagval("MESSAGE#181:Sophos_Firewall", "nwparser.payload", tvm, {
	"activityname": "fld9",
	"appfilter_policy_id": "fld10",
	"application": "application",
	"application_category": "fld23",
	"application_risk": "risk_num",
	"application_technology": "fld11",
	"appresolvedby": "fld22",
	"category": "fld4",
	"category_type": "fld5",
	"connevent": "fld19",
	"connid": "connectionid",
	"contenttype": "content_type",
	"dir_disp": "fld18",
	"domain": "fqdn",
	"dst_country_code": "location_dst",
	"dst_ip": "daddr",
	"dst_port": "dport",
	"dstzone": "dst_zone",
	"dstzonetype": "fld17",
	"duration": "duration",
	"exceptions": "fld8",
	"fw_rule_id": "rule_uid",
	"hb_health": "fld21",
	"httpresponsecode": "fld7",
	"iap": "id1",
	"in_interface": "sinterface",
	"ips_policy_id": "policy_id",
	"log_component": "event_source",
	"log_subtype": "category",
	"log_type": "event_type",
	"message": "info",
	"out_interface": "dinterface",
	"override_token": "fld6",
	"policy_type": "fld23",
	"priority": "severity",
	"protocol": "protocol",
	"reason": "result",
	"recv_bytes": "rbytes",
	"recv_pkts": "fld15",
	"referer": "web_referer",
	"sent_bytes": "sbytes",
	"sent_pkts": "fld14",
	"src_country_code": "location_src",
	"src_ip": "saddr",
	"src_mac": "smacaddr",
	"src_port": "sport",
	"srczone": "src_zone",
	"srczonetype": "fld16",
	"status": "event_state",
	"status_code": "resultcode",
	"tran_dst_ip": "dtransaddr",
	"tran_dst_port": "dtransport",
	"tran_src_ip": "stransaddr",
	"tran_src_port": "stransport",
	"transactionid": "id2",
	"url": "url",
	"user_agent": "user_agent",
	"user_gp": "group",
	"user_name": "username",
	"vconnid": "fld20",
}, processor_chain([
	setc("eventcategory","1204000000"),
	dup2,
	date_time({
		dest: "event_time",
		args: ["hdate","htime"],
		fmts: [
			[dW,dc("-"),dG,dc("-"),dF,dH,dc(":"),dU,dc(":"),dS],
		],
	}),
]));

var msg182 = msg("Sophos_Firewall", part190);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"Sophos_Firewall": msg182,
		"URID": msg38,
		"afcd": msg94,
		"astarosg_TVM": msg179,
		"aua": msg174,
		"barnyard": select13,
		"confd": msg90,
		"confd-sync": msg89,
		"exim": select14,
		"frox": select9,
		"httpd": select18,
		"httpproxy": select3,
		"ipsec_starter": select10,
		"named": select2,
		"pluto": select11,
		"pop3proxy": msg178,
		"reverseproxy": select8,
		"smtpd": select15,
		"sockd": select17,
		"sshd": select16,
		"ulogd": msg39,
		"xl2tpd": select12,
	}),
]);

var part191 = // "Pattern{Field(p0,false)}"
match_copy("MESSAGE#44:reverseproxy:07/1_0", "nwparser.p0", "p0");
