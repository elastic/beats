// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Polyfill for String startsWith.
if (!String.prototype.startsWith) {
    Object.defineProperty(String.prototype, 'startsWith', {
        value: function(search, pos) {
            pos = !pos || pos < 0 ? 0 : +pos;
            return this.substring(pos, pos + search.length) === search;
        }
    });
}

var sysmon = (function () {
    var path = require("path");
    var processor = require("processor");
    var winlogbeat = require("winlogbeat");

    // Windows error codes for DNS.
    // https://docs.microsoft.com/en-us/windows/win32/debug/system-error-codes--9000-11999-
    var dnsQueryStatusCodes = {
        "0": "SUCCESS",
        "2329": "DNS_ERROR_RCODE_FORMAT_ERROR",
        "2330": "DNS_ERROR_RCODE_NXRRSET",
        "2331": "DNS_ERROR_RCODE_NOTAUTH",
        "2332": "DNS_ERROR_RCODE_NOTZONE",
        "2338": "DNS_ERROR_RCODE_BADSIG",
        "2339": "DNS_ERROR_RCODE_BADKEY",
        "2390": "DNS_ERROR_NOT_ENOUGH_SIGNING_KEY_DESCRIPTORS",
        "2391": "DNS_ERROR_UNSUPPORTED_ALGORITHM",
        "2392": "DNS_ERROR_INVALID_KEY_SIZE",
        "2393": "DNS_ERROR_SIGNING_KEY_NOT_ACCESSIBLE",
        "2394": "DNS_ERROR_KSP_DOES_NOT_SUPPORT_PROTECTION",
        "2395": "DNS_ERROR_UNEXPECTED_DATA_PROTECTION_ERROR",
        "2396": "DNS_ERROR_UNEXPECTED_CNG_ERROR",
        "2397": "DNS_ERROR_UNKNOWN_SIGNING_PARAMETER_VERSION",
        "2398": "DNS_ERROR_KSP_NOT_ACCESSIBLE",
        "2399": "DNS_ERROR_TOO_MANY_SKDS",
        "2520": "DNS_ERROR_RCODE",
        "2521": "DNS_ERROR_UNSECURE_PACKET",
        "2522": "DNS_REQUEST_PENDING",
        "2550": "DNS_ERROR_INVALID_IP_ADDRESS",
        "2551": "DNS_ERROR_INVALID_PROPERTY",
        "2552": "DNS_ERROR_TRY_AGAIN_LATER",
        "2553": "DNS_ERROR_NOT_UNIQUE",
        "2554": "DNS_ERROR_NON_RFC_NAME",
        "2555": "DNS_STATUS_FQDN",
        "2556": "DNS_STATUS_DOTTED_NAME",
        "2557": "DNS_STATUS_SINGLE_PART_NAME",
        "2558": "DNS_ERROR_INVALID_NAME_CHAR",
        "2559": "DNS_ERROR_NUMERIC_NAME",
        "2560": "DNS_ERROR_BACKGROUND_LOADING",
        "2561": "DNS_ERROR_NOT_ALLOWED_ON_RODC",
        "2562": "DNS_ERROR_NOT_ALLOWED_UNDER_DNAME",
        "2563": "DNS_ERROR_DELEGATION_REQUIRED",
        "2564": "DNS_ERROR_INVALID_POLICY_TABLE",
        "2581": "DNS_ERROR_ZONE_DOES_NOT_EXIST",
        "2582": "DNS_ERROR_NO_ZONE_INFO",
        "2583": "DNS_ERROR_INVALID_ZONE_OPERATION",
        "2584": "DNS_ERROR_ZONE_CONFIGURATION_ERROR",
        "2585": "DNS_ERROR_ZONE_HAS_NO_SOA_RECORD",
        "2586": "DNS_ERROR_ZONE_HAS_NO_NS_RECORDS",
        "2587": "DNS_ERROR_ZONE_LOCKED",
        "2588": "DNS_ERROR_ZONE_CREATION_FAILED",
        "2589": "DNS_ERROR_ZONE_ALREADY_EXISTS",
        "2590": "DNS_ERROR_NEED_WINS_SERVERS",
        "2591": "DNS_ERROR_NBSTAT_INIT_FAILED",
        "2592": "DNS_ERROR_SOA_DELETE_INVALID",
        "2593": "DNS_ERROR_FORWARDER_ALREADY_EXISTS",
        "2594": "DNS_ERROR_ZONE_REQUIRES_MASTER_IP",
        "2595": "DNS_ERROR_ZONE_IS_SHUTDOWN",
        "2596": "DNS_ERROR_ZONE_LOCKED_FOR_SIGNING",
        "2617": "DNS_INFO_AXFR_COMPLETE",
        "2618": "DNS_ERROR_AXFR",
        "2619": "DNS_INFO_ADDED_LOCAL_WINS",
        "2649": "DNS_STATUS_CONTINUE_NEEDED",
        "9002": "DNS_ERROR_RCODE_SERVER_FAILURE",
        "9003": "DNS_ERROR_RCODE_NAME_ERROR",
        "9004": "DNS_ERROR_RCODE_NOT_IMPLEMENTED",
        "9005": "DNS_ERROR_RCODE_REFUSED",
        "9006": "DNS_ERROR_RCODE_YXDOMAIN",
        "9007": "DNS_ERROR_RCODE_YXRRSET",
        "9018": "DNS_ERROR_RCODE_BADTIME",
        "9101": "DNS_ERROR_KEYMASTER_REQUIRED",
        "9102": "DNS_ERROR_NOT_ALLOWED_ON_SIGNED_ZONE",
        "9103": "DNS_ERROR_NSEC3_INCOMPATIBLE_WITH_RSA_SHA1",
        "9114": "DNS_ERROR_INVALID_ROLLOVER_PERIOD",
        "9115": "DNS_ERROR_INVALID_INITIAL_ROLLOVER_OFFSET",
        "9116": "DNS_ERROR_ROLLOVER_IN_PROGRESS",
        "9117": "DNS_ERROR_STANDBY_KEY_NOT_PRESENT",
        "9118": "DNS_ERROR_NOT_ALLOWED_ON_ZSK",
        "9119": "DNS_ERROR_NOT_ALLOWED_ON_ACTIVE_SKD",
        "9120": "DNS_ERROR_ROLLOVER_ALREADY_QUEUED",
        "9121": "DNS_ERROR_NOT_ALLOWED_ON_UNSIGNED_ZONE",
        "9122": "DNS_ERROR_BAD_KEYMASTER",
        "9123": "DNS_ERROR_INVALID_SIGNATURE_VALIDITY_PERIOD",
        "9124": "DNS_ERROR_INVALID_NSEC3_ITERATION_COUNT",
        "9125": "DNS_ERROR_DNSSEC_IS_DISABLED",
        "9126": "DNS_ERROR_INVALID_XML",
        "9127": "DNS_ERROR_NO_VALID_TRUST_ANCHORS",
        "9128": "DNS_ERROR_ROLLOVER_NOT_POKEABLE",
        "9129": "DNS_ERROR_NSEC3_NAME_COLLISION",
        "9130": "DNS_ERROR_NSEC_INCOMPATIBLE_WITH_NSEC3_RSA_SHA1",
        "9501": "DNS_INFO_NO_RECORDS",
        "9502": "DNS_ERROR_BAD_PACKET",
        "9503": "DNS_ERROR_NO_PACKET",
        "9551": "DNS_ERROR_INVALID_TYPE",
        "9562": "DNS_ERROR_NOT_ALLOWED_ON_ROOT_SERVER",
        "9563": "DNS_ERROR_NOT_ALLOWED_UNDER_DELEGATION",
        "9564": "DNS_ERROR_CANNOT_FIND_ROOT_HINTS",
        "9565": "DNS_ERROR_INCONSISTENT_ROOT_HINTS",
        "9566": "DNS_ERROR_DWORD_VALUE_TOO_SMALL",
        "9567": "DNS_ERROR_DWORD_VALUE_TOO_LARGE",
        "9610": "DNS_ERROR_AUTOZONE_ALREADY_EXISTS",
        "9611": "DNS_ERROR_INVALID_ZONE_TYPE",
        "9612": "DNS_ERROR_SECONDARY_REQUIRES_MASTER_IP",
        "9613": "DNS_ERROR_ZONE_NOT_SECONDARY",
        "9614": "DNS_ERROR_NEED_SECONDARY_ADDRESSES",
        "9615": "DNS_ERROR_WINS_INIT_FAILED",
        "9651": "DNS_ERROR_PRIMARY_REQUIRES_DATAFILE",
        "9652": "DNS_ERROR_INVALID_DATAFILE_NAME",
        "9653": "DNS_ERROR_DATAFILE_OPEN_FAILURE",
        "9654": "DNS_ERROR_FILE_WRITEBACK_FAILED",
        "9655": "DNS_ERROR_DATAFILE_PARSING",
        "9701": "DNS_ERROR_RECORD_DOES_NOT_EXIST",
        "9702": "DNS_ERROR_RECORD_FORMAT",
        "9703": "DNS_ERROR_NODE_CREATION_FAILED",
        "9704": "DNS_ERROR_UNKNOWN_RECORD_TYPE",
        "9705": "DNS_ERROR_RECORD_TIMED_OUT",
        "9706": "DNS_ERROR_NAME_NOT_IN_ZONE",
        "9707": "DNS_ERROR_CNAME_LOOP",
        "9708": "DNS_ERROR_NODE_IS_CNAME",
        "9709": "DNS_ERROR_CNAME_COLLISION",
        "9710": "DNS_ERROR_RECORD_ONLY_AT_ZONE_ROOT",
        "9711": "DNS_ERROR_RECORD_ALREADY_EXISTS",
        "9712": "DNS_ERROR_SECONDARY_DATA",
        "9713": "DNS_ERROR_NO_CREATE_CACHE_DATA",
        "9714": "DNS_ERROR_NAME_DOES_NOT_EXIST",
        "9715": "DNS_WARNING_PTR_CREATE_FAILED",
        "9716": "DNS_WARNING_DOMAIN_UNDELETED",
        "9717": "DNS_ERROR_DS_UNAVAILABLE",
        "9718": "DNS_ERROR_DS_ZONE_ALREADY_EXISTS",
        "9719": "DNS_ERROR_NO_BOOTFILE_IF_DS_ZONE",
        "9720": "DNS_ERROR_NODE_IS_DNAME",
        "9721": "DNS_ERROR_DNAME_COLLISION",
        "9722": "DNS_ERROR_ALIAS_LOOP",
        "9851": "DNS_ERROR_NO_TCPIP",
        "9852": "DNS_ERROR_NO_DNS_SERVERS",
        "9901": "DNS_ERROR_DP_DOES_NOT_EXIST",
        "9902": "DNS_ERROR_DP_ALREADY_EXISTS",
        "9903": "DNS_ERROR_DP_NOT_ENLISTED",
        "9904": "DNS_ERROR_DP_ALREADY_ENLISTED",
        "9905": "DNS_ERROR_DP_NOT_AVAILABLE",
        "9906": "DNS_ERROR_DP_FSMO_ERROR",
    };

    // Windows DNS record type constants.
    // https://docs.microsoft.com/en-us/windows/win32/dns/dns-constants
    var dnsRecordTypes = {
        "1": "A",
        "2": "NS",
        "3": "MD",
        "4": "MF",
        "5": "CNAME",
        "6": "SOA",
        "7": "MB",
        "8": "MG",
        "9": "MR",
        "10": "NULL",
        "11": "WKS",
        "12": "PTR",
        "13": "HINFO",
        "14": "MINFO",
        "15": "MX",
        "16": "TXT",
        "17": "RP",
        "18": "AFSDB",
        "19": "X25",
        "20": "ISDN",
        "21": "RT",
        "22": "NSAP",
        "23": "NSAPPTR",
        "24": "SIG",
        "25": "KEY",
        "26": "PX",
        "27": "GPOS",
        "28": "AAAA",
        "29": "LOC",
        "30": "NXT",
        "31": "EID",
        "32": "NIMLOC",
        "33": "SRV",
        "34": "ATMA",
        "35": "NAPTR",
        "36": "KX",
        "37": "CERT",
        "38": "A6",
        "39": "DNAME",
        "40": "SINK",
        "41": "OPT",
        "43": "DS",
        "46": "RRSIG",
        "47": "NSEC",
        "48": "DNSKEY",
        "49": "DHCID",
        "100": "UINFO",
        "101": "UID",
        "102": "GID",
        "103": "UNSPEC",
        "248": "ADDRS",
        "249": "TKEY",
        "250": "TSIG",
        "251": "IXFR",
        "252": "AXFR",
        "253": "MAILB",
        "254": "MAILA",
        "255": "ANY",
        "65281": "WINS",
        "65282": "WINSR",
    };

    var setProcessNameUsingExe = function(evt) {
        setProcessNameFromPath(evt, "process.executable", "process.name");
    };

    var setParentProcessNameUsingExe = function(evt) {
        setProcessNameFromPath(evt, "process.parent.executable", "process.parent.name");
    };

    var setProcessNameFromPath = function(evt, pathField, nameField) {
        var name = evt.Get(nameField);
        if (name) {
            return;
        }
        var exe = evt.Get(pathField);
        evt.Put(nameField, path.basename(exe));
    };

    var splitCommandLine = function(evt, field) {
        var commandLine = evt.Get(field);
        if (!commandLine) {
            return;
        }
        evt.Put(field, winlogbeat.splitCommandLine(commandLine));
    };

    var splitProcessArgs = function(evt) {
        splitCommandLine(evt, "process.args");
    };

    var splitParentProcessArgs = function(evt) {
        splitCommandLine(evt, "process.parent.args");
    };

    var addUser = function(evt) {
        var userParts = evt.Get("winlog.event_data.User").split("\\");
        if (userParts.length === 2) {
            evt.Delete("user");
            evt.Put("user.domain", userParts[0]);
            evt.Put("user.name", userParts[1]);
            evt.Delete("winlog.event_data.User");
        }
    };

    var addNetworkDirection = function(evt) {
        switch (evt.Get("winlog.event_data.Initiated")) {
            case "true":
                evt.Put("network.direction", "outbound");
                break;
            case "false":
                evt.Put("network.direction", "inbound");
                break;
        }
        evt.Delete("winlog.event_data.Initiated");
    };

    var addNetworkType = function(evt) {
        switch (evt.Get("winlog.event_data.SourceIsIpv6")) {
            case "true":
                evt.Put("network.type", "ipv6");
                break;
            case "false":
                evt.Put("network.type", "ipv4");
                break;
        }
        evt.Delete("winlog.event_data.SourceIsIpv6");
        evt.Delete("winlog.event_data.DestinationIsIpv6");
    };

    var addHashes = function(evt, hashField) {
        var hashes = evt.Get(hashField);
        evt.Delete(hashField);
        hashes.split(",").forEach(function(hash){
            var parts = hash.split("=");
            if (parts.length !== 2) {
                return;
            }

            var key = parts[0].toLowerCase();
            var value = parts[1].toLowerCase();
            evt.Put("hash."+key, value);
        });
    };

    var splitHashes = function(evt) {
        addHashes(evt, "winlog.event_data.Hashes");
    };

    var splitHash = function(evt) {
        addHashes(evt, "winlog.event_data.Hash");
    };

    var removeEmptyEventData = function(evt) {
        var eventData = evt.Get("winlog.event_data");
        if (eventData && Object.keys(eventData).length === 0) {
            evt.Delete("winlog.event_data");
        }
    };

    var translateDnsQueryStatus = function(evt) {
        var statusCode = evt.Get("sysmon.dns.status");
        if (!statusCode) {
            return;
        }
        var statusName = dnsQueryStatusCodes[statusCode];
        if (statusName === undefined) {
            return;
        }
        evt.Put("sysmon.dns.status", statusName);
    };

    // Splits the QueryResults field that contains the DNS responses.
    // Example: "type:  5 f2.taboola.map.fastly.net;::ffff:151.101.66.2;::ffff:151.101.130.2;::ffff:151.101.194.2;::ffff:151.101.2.2;"
    var splitDnsQueryResults = function(evt) {
        var results = evt.Get("winlog.event_data.QueryResults");
        if (!results) {
            return;
        }
        results = results.split(';');

        var answers = [];
        var ips = [];
        for (var i = 0; i < results.length; i++) {
            var answer = results[i];
            if (!answer) {
                continue;
            }

            if (answer.startsWith('type:')) {
                var parts = answer.split(/\s+/);
                if (parts.length !== 3) {
                    throw "unexpected QueryResult format";
                }

                answers.push({
                    type: dnsRecordTypes[parts[1]],
                    data: parts[2],
                });
            } else {
                // Convert V4MAPPED addresses.
                answer = answer.replace("::ffff:", "");
                ips.push(answer);

                // Synthesize record type based on IP address type.
                var type = "A";
                if (answer.indexOf(":") !== -1) {
                    type = "AAAA";
                }
                answers.push({type: type, data: answer});
            }
        }

        if (answers.length > 0) {
            evt.Put("dns.answers", answers);
        }
        if (ips.length > 0) {
            evt.Put("dns.resolved_ip", ips);
        }
        evt.Delete("winlog.event_data.QueryResults");
    };

    var parseUtcTime = new processor.Timestamp({
        field: "winlog.event_data.UtcTime",
        target_field: "winlog.event_data.UtcTime",
        timezone: "UTC",
        layouts: ["2006-01-02 15:04:05.999"],
        tests: ["2019-06-26 21:19:43.237"],
        ignore_missing: true,
    });

    var event1 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            "fields": {
                "event.category": "process",
                "event.type:": "process_start",
            },
            "target": "",
        })
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.CommandLine", to: "process.args"},
                {from: "winlog.event_data.CurrentDirectory", to: "process.working_directory"},
                {from: "winlog.event_data.ParentProcessGuid", to: "process.parent.entity_id"},
                {from: "winlog.event_data.ParentProcessId", to: "process.parent.pid", type: "long"},
                {from: "winlog.event_data.ParentImage", to: "process.parent.executable"},
                {from: "winlog.event_data.ParentCommandLine", to: "process.parent.args"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitProcessArgs)
        .Add(addUser)
        .Add(splitHashes)
        .Add(setParentProcessNameUsingExe)
        .Add(splitParentProcessArgs)
        .Add(removeEmptyEventData)
        .Build();

    var event2 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event3 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.Protocol", to: "network.transport"},
                {from: "winlog.event_data.SourceIp", to: "source.ip", type: "ip"},
                {from: "winlog.event_data.SourceHostname", to: "source.domain", type: "string"},
                {from: "winlog.event_data.SourcePort", to: "source.port", type: "long"},
                {from: "winlog.event_data.DestinationIp", to: "destination.ip", type: "ip"},
                {from: "winlog.event_data.DestinationHostname", to: "destination.domain", type: "string"},
                {from: "winlog.event_data.DestinationPort", to: "destination.port", type: "long"},
                {from: "winlog.event_data.DestinationPortName", to: "network.protocol"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(addUser)
        .Add(addNetworkDirection)
        .Add(addNetworkType)
        .CommunityID()
        .Add(removeEmptyEventData)
        .Build();

    var event4 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    var event5 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            "fields": {
                "event.category": "process",
                "event.type:": "process_end",
            },
            "target": "",
        })
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event6 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ImageLoaded", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(splitHashes)
        .Add(removeEmptyEventData)
        .Build();

    var event7 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.ImageLoaded", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitHashes)
        .Add(removeEmptyEventData)
        .Build();

    var event8 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.SourceProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.SourceProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.SourceImage", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event9 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.Device", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event10 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.SourceProcessGUID", to: "process.entity_id"},
                {from: "winlog.event_data.SourceProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.SourceThreadId", to: "process.thread.id", type: "long"},
                {from: "winlog.event_data.SourceImage", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event11 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event12 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event13 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event14 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event15 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitHash)
        .Add(removeEmptyEventData)
        .Build();

    var event16 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    var event17 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.PipeName", to: "file.name"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event18 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.PipeName", to: "file.name"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event19 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    var event20 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.Destination", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event21 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    var event22 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.QueryName", to: "dns.question.name"},
                {from: "winlog.event_data.QueryStatus", to: "sysmon.dns.status"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .RegisteredDomain({
            ignore_failure: true,
            ignore_missing: true,
            field: "dns.question.name",
            target_field: "dns.question.registered_domain",
        })
        .Add(translateDnsQueryStatus)
        .Add(splitDnsQueryResults)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event255 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ID", to: "error.code"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    return {
        // Event ID 1 - Process Create.
        1: event1.Run,

        // Event ID 2 - File creation time changed.
        2: event2.Run,

        // Event ID 3 - Network connection detected.
        3: event3.Run,

        // Event ID 4 - Sysmon service state changed.
        4: event4.Run,

        // Event ID 5 - Process terminated.
        5: event5.Run,

        // Event ID 6 - Driver loaded.
        6: event6.Run,

        // Event ID 7 - Image loaded.
        7: event7.Run,

        // Event ID 8 - CreateRemoteThread detected.
        8: event8.Run,

        // Event ID 9 - RawAccessRead detected.
        9: event9.Run,

        // Event ID 10 - Process accessed.
        10: event10.Run,

        // Event ID 11 - File created.
        11: event11.Run,

        // Event ID 12 - Registry object added or deleted.
        12: event12.Run,

        // Event ID 13 - Registry value set.
        13: event13.Run,

        // Event ID 14 - Registry object renamed.
        14: event14.Run,

        // Event ID 15 - File stream created.
        15: event15.Run,

        // Event ID 16 - Sysmon config state changed.
        16: event16.Run,

        // Event ID 17 - Pipe Created.
        17: event17.Run,

        // Event ID 18 - Pipe Connected.
        18: event18.Run,

        // Event ID 19 - WmiEventFilter activity detected.
        19: event19.Run,

        // Event ID 20 - WmiEventConsumer activity detected.
        20: event20.Run,

        // Event ID 21 - WmiEventConsumerToFilter activity detected.
        21: event21.Run,

        // Event ID 22 - DNSEvent (DNS query).
        22: event22.Run,

        // Event ID 255 - Error report.
        255: event255.Run,

        process: function(evt) {
            var event_id = evt.Get("winlog.event_id");
            var processor= this[event_id];
            if (processor === undefined) {
                throw "unexpected sysmon event_id";
            }
            evt.Put("event.module", "sysmon");
            processor(evt);
        },
    };
})();

function process(evt) {
    return sysmon.process(evt);
}
