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

var dup1 = setc("eventcategory","1605020000");

var dup2 = setf("msg","$MSG");

var dup3 = setc("eventcategory","1803000000");

var dup4 = setc("ec_theme","ALM");

var dup5 = setc("ec_subject","NetworkComm");

var dup6 = setc("ec_outcome","Failure");

var dup7 = setc("action","deny");

var dup8 = setc("dclass_counter1_string","block_count");

var dup9 = match("MESSAGE#2:ms-wbt-server", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} proto=%{protocol->} service=%{network_service->} status=deny src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} server_app=%{fld12->} pid=%{process_id->} app_name=%{fld14->} traff_direct=%{direction->} block_count=%{dclass_counter1->} logon_user=%{username}@%{domain->} msg=%{result}", processor_chain([
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup2,
	dup8,
]));

var hdr1 = match("HEADER#0:0001", "message", "%{hmonth->} %{hday->} %{htime->} %{hhostname->} proto=%{hprotocol->} service=%{messageid->} status=%{haction->} src=%{hsaddr->} dst=%{hdaddr->} src_port=%{hsport->} dst_port=%{hdport->} %{p0}", processor_chain([
	setc("header_id","0001"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hday"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hhostname"),
			constant(" proto="),
			field("hprotocol"),
			constant(" service="),
			field("messageid"),
			constant(" status="),
			field("haction"),
			constant(" src="),
			field("hsaddr"),
			constant(" dst="),
			field("hdaddr"),
			constant(" src_port="),
			field("hsport"),
			constant(" dst_port="),
			field("hdport"),
			constant(" "),
			field("p0"),
		],
	}),
]));

var hdr2 = match("HEADER#1:0003", "message", "%{hmonth->} %{hday->} %{htime->} %{hhostname->} (%{messageid->} %{hfld5->} times in last %{hfld6}) %{hfld7->} %{hfld8}::%{p0}", processor_chain([
	setc("header_id","0003"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hday"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hhostname"),
			constant(" ("),
			field("messageid"),
			constant(" "),
			field("hfld5"),
			constant(" times in last "),
			field("hfld6"),
			constant(") "),
			field("hfld7"),
			constant(" "),
			field("hfld8"),
			constant("::"),
			field("p0"),
		],
	}),
]));

var hdr3 = match("HEADER#2:0002", "message", "%{hmonth->} %{hday->} %{htime->} %{hhostname->} %{messageid->} %{hfld5}::%{p0}", processor_chain([
	setc("header_id","0002"),
	call({
		dest: "nwparser.payload",
		fn: STRCAT,
		args: [
			field("hmonth"),
			constant(" "),
			field("hday"),
			constant(" "),
			field("htime"),
			constant(" "),
			field("hhostname"),
			constant(" "),
			field("messageid"),
			constant(" "),
			field("hfld5"),
			constant("::"),
			field("p0"),
		],
	}),
]));

var select1 = linear_select([
	hdr1,
	hdr2,
	hdr3,
]);

var part1 = match("MESSAGE#0:enter", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} enter %{info}", processor_chain([
	dup1,
	dup2,
]));

var msg1 = msg("enter", part1);

var part2 = match("MESSAGE#1:repeated", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} (repeated %{fld5->} times in last %{fld6}) enter %{info}", processor_chain([
	dup1,
	dup2,
]));

var msg2 = msg("repeated", part2);

var msg3 = msg("ms-wbt-server", dup9);

var msg4 = msg("http", dup9);

var msg5 = msg("https", dup9);

var msg6 = msg("smtp", dup9);

var msg7 = msg("pop3", dup9);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"enter": msg1,
		"http": msg4,
		"https": msg5,
		"ms-wbt-server": msg3,
		"pop3": msg7,
		"repeated": msg2,
		"smtp": msg6,
	}),
]);

var part3 = match("MESSAGE#2:ms-wbt-server", "nwparser.payload", "%{fld1->} %{fld2->} %{fld3->} %{hostname->} proto=%{protocol->} service=%{network_service->} status=deny src=%{saddr->} dst=%{daddr->} src_port=%{sport->} dst_port=%{dport->} server_app=%{fld12->} pid=%{process_id->} app_name=%{fld14->} traff_direct=%{direction->} block_count=%{dclass_counter1->} logon_user=%{username}@%{domain->} msg=%{result}", processor_chain([
	dup3,
	dup4,
	dup5,
	dup6,
	dup7,
	dup2,
	dup8,
]));
