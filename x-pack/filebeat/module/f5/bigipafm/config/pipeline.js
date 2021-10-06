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

var map_getEventCategoryActivity = {
	keyvaluepairs: {
		"Accept": constant("Permit"),
		"Closed": constant("Disable"),
		"Drop": dup1,
		"Established": constant("Enable"),
		"Reject": dup1,
	},
};

var dup1 = constant("Deny");

var hdr1 = match("HEADER#0:0001", "message", "%{hfld1->} %{hfld2->} %{hhostname->} %{hfld3->} %{hfld4->} %{hfld5->} [F5@%{hfld6->} %{payload}", processor_chain([
	setc("header_id","0001"),
	setc("messageid","BIGIP_AFM"),
]));

var select1 = linear_select([
	hdr1,
]);

var part1 = tagval("MESSAGE#0:BIGIP_AFM", "nwparser.payload", tvm, {
	"acl_policy_name": "policyname",
	"acl_policy_type": "fld1",
	"acl_rule_name": "rulename",
	"action": "action",
	"bigip_mgmt_ip": "hostip",
	"context_name": "context",
	"context_type": "fld2",
	"date_time": "event_time_string",
	"dest_ip": "daddr",
	"dest_port": "dport",
	"device_product": "product",
	"device_vendor": "fld3",
	"device_version": "version",
	"drop_reason": "fld4",
	"dst_geo": "location_dst",
	"errdefs_msg_name": "event_type",
	"errdefs_msgno": "id",
	"flow_id": "fld5",
	"hostname": "hostname",
	"ip_protocol": "protocol",
	"partition_name": "fld6",
	"route_domain": "fld7",
	"sa_translation_pool": "fld8",
	"sa_translation_type": "fld9",
	"severity": "severity",
	"source_ip": "saddr",
	"source_port": "sport",
	"source_user": "username",
	"src_geo": "location_src",
	"translated_dest_ip": "dtransaddr",
	"translated_dest_port": "dtransport",
	"translated_ip_protocol": "fld10",
	"translated_route_domain": "fld11",
	"translated_source_ip": "stransaddr",
	"translated_source_port": "stransport",
	"translated_vlan": "fld12",
	"vlan": "vlan",
}, processor_chain([
	setc("eventcategory","1801000000"),
	setf("msg","$MSG"),
	date_time({
		dest: "event_time",
		args: ["event_time_string"],
		fmts: [
			[dB,dD,dW,dZ],
		],
	}),
	setc("ec_subject","NetworkComm"),
	setc("ec_theme","Communication"),
	lookup({
		dest: "nwparser.ec_activity",
		map: map_getEventCategoryActivity,
		key: field("action"),
	}),
	setf("obj_name","hfld6"),
]));

var msg1 = msg("BIGIP_AFM", part1);

var chain1 = processor_chain([
	select1,
	msgid_select({
		"BIGIP_AFM": msg1,
	}),
]);
