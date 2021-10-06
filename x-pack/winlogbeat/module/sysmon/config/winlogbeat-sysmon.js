// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Polyfill for String startsWith.
if (!String.prototype.startsWith) {
    Object.defineProperty(String.prototype, "startsWith", {
        value: function (search, pos) {
            pos = !pos || pos < 0 ? 0 : +pos;
            return this.substring(pos, pos + search.length) === search;
        },
    });
}

var sysmon = (function () {
    var path = require("path");
    var processor = require("processor");
    var windows = require("windows");
    var net = require("net");

    // Windows error codes for DNS. This list was generated using
    // 'go run gen_dns_error_codes.go'.
    var dnsQueryStatusCodes = {
        "0": "SUCCESS",
        "5": "ERROR_ACCESS_DENIED",
        "8": "ERROR_NOT_ENOUGH_MEMORY",
        "13": "ERROR_INVALID_DATA",
        "14": "ERROR_OUTOFMEMORY",
        "123": "ERROR_INVALID_NAME",
        "1214": "ERROR_INVALID_NETNAME",
        "1223": "ERROR_CANCELLED",
        "1460": "ERROR_TIMEOUT",
        "4312": "ERROR_OBJECT_NOT_FOUND",
        "9001": "DNS_ERROR_RCODE_FORMAT_ERROR",
        "9002": "DNS_ERROR_RCODE_SERVER_FAILURE",
        "9003": "DNS_ERROR_RCODE_NAME_ERROR",
        "9004": "DNS_ERROR_RCODE_NOT_IMPLEMENTED",
        "9005": "DNS_ERROR_RCODE_REFUSED",
        "9006": "DNS_ERROR_RCODE_YXDOMAIN",
        "9007": "DNS_ERROR_RCODE_YXRRSET",
        "9008": "DNS_ERROR_RCODE_NXRRSET",
        "9009": "DNS_ERROR_RCODE_NOTAUTH",
        "9010": "DNS_ERROR_RCODE_NOTZONE",
        "9016": "DNS_ERROR_RCODE_BADSIG",
        "9017": "DNS_ERROR_RCODE_BADKEY",
        "9018": "DNS_ERROR_RCODE_BADTIME",
        "9101": "DNS_ERROR_KEYMASTER_REQUIRED",
        "9102": "DNS_ERROR_NOT_ALLOWED_ON_SIGNED_ZONE",
        "9103": "DNS_ERROR_NSEC3_INCOMPATIBLE_WITH_RSA_SHA1",
        "9104": "DNS_ERROR_NOT_ENOUGH_SIGNING_KEY_DESCRIPTORS",
        "9105": "DNS_ERROR_UNSUPPORTED_ALGORITHM",
        "9106": "DNS_ERROR_INVALID_KEY_SIZE",
        "9107": "DNS_ERROR_SIGNING_KEY_NOT_ACCESSIBLE",
        "9108": "DNS_ERROR_KSP_DOES_NOT_SUPPORT_PROTECTION",
        "9109": "DNS_ERROR_UNEXPECTED_DATA_PROTECTION_ERROR",
        "9110": "DNS_ERROR_UNEXPECTED_CNG_ERROR",
        "9111": "DNS_ERROR_UNKNOWN_SIGNING_PARAMETER_VERSION",
        "9112": "DNS_ERROR_KSP_NOT_ACCESSIBLE",
        "9113": "DNS_ERROR_TOO_MANY_SKDS",
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
        "9504": "DNS_ERROR_RCODE",
        "9505": "DNS_ERROR_UNSECURE_PACKET",
        "9506": "DNS_REQUEST_PENDING",
        "9551": "DNS_ERROR_INVALID_TYPE",
        "9552": "DNS_ERROR_INVALID_IP_ADDRESS",
        "9553": "DNS_ERROR_INVALID_PROPERTY",
        "9554": "DNS_ERROR_TRY_AGAIN_LATER",
        "9555": "DNS_ERROR_NOT_UNIQUE",
        "9556": "DNS_ERROR_NON_RFC_NAME",
        "9557": "DNS_STATUS_FQDN",
        "9558": "DNS_STATUS_DOTTED_NAME",
        "9559": "DNS_STATUS_SINGLE_PART_NAME",
        "9560": "DNS_ERROR_INVALID_NAME_CHAR",
        "9561": "DNS_ERROR_NUMERIC_NAME",
        "9562": "DNS_ERROR_NOT_ALLOWED_ON_ROOT_SERVER",
        "9563": "DNS_ERROR_NOT_ALLOWED_UNDER_DELEGATION",
        "9564": "DNS_ERROR_CANNOT_FIND_ROOT_HINTS",
        "9565": "DNS_ERROR_INCONSISTENT_ROOT_HINTS",
        "9566": "DNS_ERROR_DWORD_VALUE_TOO_SMALL",
        "9567": "DNS_ERROR_DWORD_VALUE_TOO_LARGE",
        "9568": "DNS_ERROR_BACKGROUND_LOADING",
        "9569": "DNS_ERROR_NOT_ALLOWED_ON_RODC",
        "9570": "DNS_ERROR_NOT_ALLOWED_UNDER_DNAME",
        "9571": "DNS_ERROR_DELEGATION_REQUIRED",
        "9572": "DNS_ERROR_INVALID_POLICY_TABLE",
        "9573": "DNS_ERROR_ADDRESS_REQUIRED",
        "9601": "DNS_ERROR_ZONE_DOES_NOT_EXIST",
        "9602": "DNS_ERROR_NO_ZONE_INFO",
        "9603": "DNS_ERROR_INVALID_ZONE_OPERATION",
        "9604": "DNS_ERROR_ZONE_CONFIGURATION_ERROR",
        "9605": "DNS_ERROR_ZONE_HAS_NO_SOA_RECORD",
        "9606": "DNS_ERROR_ZONE_HAS_NO_NS_RECORDS",
        "9607": "DNS_ERROR_ZONE_LOCKED",
        "9608": "DNS_ERROR_ZONE_CREATION_FAILED",
        "9609": "DNS_ERROR_ZONE_ALREADY_EXISTS",
        "9610": "DNS_ERROR_AUTOZONE_ALREADY_EXISTS",
        "9611": "DNS_ERROR_INVALID_ZONE_TYPE",
        "9612": "DNS_ERROR_SECONDARY_REQUIRES_MASTER_IP",
        "9613": "DNS_ERROR_ZONE_NOT_SECONDARY",
        "9614": "DNS_ERROR_NEED_SECONDARY_ADDRESSES",
        "9615": "DNS_ERROR_WINS_INIT_FAILED",
        "9616": "DNS_ERROR_NEED_WINS_SERVERS",
        "9617": "DNS_ERROR_NBSTAT_INIT_FAILED",
        "9618": "DNS_ERROR_SOA_DELETE_INVALID",
        "9619": "DNS_ERROR_FORWARDER_ALREADY_EXISTS",
        "9620": "DNS_ERROR_ZONE_REQUIRES_MASTER_IP",
        "9621": "DNS_ERROR_ZONE_IS_SHUTDOWN",
        "9622": "DNS_ERROR_ZONE_LOCKED_FOR_SIGNING",
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
        "9751": "DNS_INFO_AXFR_COMPLETE",
        "9752": "DNS_ERROR_AXFR",
        "9753": "DNS_INFO_ADDED_LOCAL_WINS",
        "9801": "DNS_STATUS_CONTINUE_NEEDED",
        "9851": "DNS_ERROR_NO_TCPIP",
        "9852": "DNS_ERROR_NO_DNS_SERVERS",
        "9901": "DNS_ERROR_DP_DOES_NOT_EXIST",
        "9902": "DNS_ERROR_DP_ALREADY_EXISTS",
        "9903": "DNS_ERROR_DP_NOT_ENLISTED",
        "9904": "DNS_ERROR_DP_ALREADY_ENLISTED",
        "9905": "DNS_ERROR_DP_NOT_AVAILABLE",
        "9906": "DNS_ERROR_DP_FSMO_ERROR",
        "9911": "DNS_ERROR_RRL_NOT_ENABLED",
        "9912": "DNS_ERROR_RRL_INVALID_WINDOW_SIZE",
        "9913": "DNS_ERROR_RRL_INVALID_IPV4_PREFIX",
        "9914": "DNS_ERROR_RRL_INVALID_IPV6_PREFIX",
        "9915": "DNS_ERROR_RRL_INVALID_TC_RATE",
        "9916": "DNS_ERROR_RRL_INVALID_LEAK_RATE",
        "9917": "DNS_ERROR_RRL_LEAK_RATE_LESSTHAN_TC_RATE",
        "9921": "DNS_ERROR_VIRTUALIZATION_INSTANCE_ALREADY_EXISTS",
        "9922": "DNS_ERROR_VIRTUALIZATION_INSTANCE_DOES_NOT_EXIST",
        "9923": "DNS_ERROR_VIRTUALIZATION_TREE_LOCKED",
        "9924": "DNS_ERROR_INVAILD_VIRTUALIZATION_INSTANCE_NAME",
        "9925": "DNS_ERROR_DEFAULT_VIRTUALIZATION_INSTANCE",
        "9951": "DNS_ERROR_ZONESCOPE_ALREADY_EXISTS",
        "9952": "DNS_ERROR_ZONESCOPE_DOES_NOT_EXIST",
        "9953": "DNS_ERROR_DEFAULT_ZONESCOPE",
        "9954": "DNS_ERROR_INVALID_ZONESCOPE_NAME",
        "9955": "DNS_ERROR_NOT_ALLOWED_WITH_ZONESCOPES",
        "9956": "DNS_ERROR_LOAD_ZONESCOPE_FAILED",
        "9957": "DNS_ERROR_ZONESCOPE_FILE_WRITEBACK_FAILED",
        "9958": "DNS_ERROR_INVALID_SCOPE_NAME",
        "9959": "DNS_ERROR_SCOPE_DOES_NOT_EXIST",
        "9960": "DNS_ERROR_DEFAULT_SCOPE",
        "9961": "DNS_ERROR_INVALID_SCOPE_OPERATION",
        "9962": "DNS_ERROR_SCOPE_LOCKED",
        "9963": "DNS_ERROR_SCOPE_ALREADY_EXISTS",
        "9971": "DNS_ERROR_POLICY_ALREADY_EXISTS",
        "9972": "DNS_ERROR_POLICY_DOES_NOT_EXIST",
        "9973": "DNS_ERROR_POLICY_INVALID_CRITERIA",
        "9974": "DNS_ERROR_POLICY_INVALID_SETTINGS",
        "9975": "DNS_ERROR_CLIENT_SUBNET_IS_ACCESSED",
        "9976": "DNS_ERROR_CLIENT_SUBNET_DOES_NOT_EXIST",
        "9977": "DNS_ERROR_CLIENT_SUBNET_ALREADY_EXISTS",
        "9978": "DNS_ERROR_SUBNET_DOES_NOT_EXIST",
        "9979": "DNS_ERROR_SUBNET_ALREADY_EXISTS",
        "9980": "DNS_ERROR_POLICY_LOCKED",
        "9981": "DNS_ERROR_POLICY_INVALID_WEIGHT",
        "9982": "DNS_ERROR_POLICY_INVALID_NAME",
        "9983": "DNS_ERROR_POLICY_MISSING_CRITERIA",
        "9984": "DNS_ERROR_INVALID_CLIENT_SUBNET_NAME",
        "9985": "DNS_ERROR_POLICY_PROCESSING_ORDER_INVALID",
        "9986": "DNS_ERROR_POLICY_SCOPE_MISSING",
        "9987": "DNS_ERROR_POLICY_SCOPE_NOT_ALLOWED",
        "9988": "DNS_ERROR_SERVERSCOPE_IS_REFERENCED",
        "9989": "DNS_ERROR_ZONESCOPE_IS_REFERENCED",
        "9990": "DNS_ERROR_POLICY_INVALID_CRITERIA_CLIENT_SUBNET",
        "9991": "DNS_ERROR_POLICY_INVALID_CRITERIA_TRANSPORT_PROTOCOL",
        "9992": "DNS_ERROR_POLICY_INVALID_CRITERIA_NETWORK_PROTOCOL",
        "9993": "DNS_ERROR_POLICY_INVALID_CRITERIA_INTERFACE",
        "9994": "DNS_ERROR_POLICY_INVALID_CRITERIA_FQDN",
        "9995": "DNS_ERROR_POLICY_INVALID_CRITERIA_QUERY_TYPE",
        "9996": "DNS_ERROR_POLICY_INVALID_CRITERIA_TIME_OF_DAY",
        "10054": "WSAECONNRESET",
        "10055": "WSAENOBUFS",
        "10060": "WSAETIMEDOUT",
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

    var setProcessNameUsingExe = function (evt) {
        setProcessNameFromPath(evt, "process.executable", "process.name");
    };

    var setParentProcessNameUsingExe = function (evt) {
        setProcessNameFromPath(
            evt,
            "process.parent.executable",
            "process.parent.name"
        );
    };

    var setProcessNameFromPath = function (evt, pathField, nameField) {
        var name = evt.Get(nameField);
        if (name) {
            return;
        }
        var exe = evt.Get(pathField);
        if (!exe) {
            return;
        }
        evt.Put(nameField, path.basename(exe));
    };

    var splitCommandLine = function (evt, source, target) {
        var commandLine = evt.Get(source);
        if (!commandLine) {
            return;
        }
        evt.Put(target, windows.splitCommandLine(commandLine));
    };

    var splitProcessArgs = function (evt) {
        splitCommandLine(evt, "process.command_line", "process.args");
    };

    var splitParentProcessArgs = function (evt) {
        splitCommandLine(
            evt,
            "process.parent.command_line",
            "process.parent.args"
        );
    };

    var addUser = function (evt) {
        var id = evt.Get("winlog.user.identifier");
        if (id) {
            evt.Put("user.id", id);
        }
        var userParts = evt.Get("winlog.event_data.User");
        if (!userParts) {
            return;
        }
        userParts = userParts.split("\\");
        if (userParts.length === 2) {
            evt.Put("user.domain", userParts[0]);
            evt.Put("user.name", userParts[1]);
            evt.AppendTo("related.user", userParts[1]);
            evt.Delete("winlog.event_data.User");
        }
    };

    var setRuleName = function (evt) {
        var ruleName = evt.Get("winlog.event_data.RuleName");
        evt.Delete("winlog.event_data.RuleName");

        if (!ruleName || ruleName === "-") {
            return;
        }

        evt.Put("rule.name", ruleName);
    };

    var addNetworkDirection = function (evt) {
        switch (evt.Get("winlog.event_data.Initiated")) {
            case "true":
                evt.Put("network.direction", "egress");
                break;
            case "false":
                evt.Put("network.direction", "ingress");
                break;
        }
        evt.Delete("winlog.event_data.Initiated");
    };

    var addNetworkType = function (evt) {
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

    var setRelatedIP = function (evt) {
        var sourceIP = evt.Get("source.ip");
        if (sourceIP) {
            evt.AppendTo("related.ip", sourceIP);
        }

        var destIP = evt.Get("destination.ip");
        if (destIP) {
            evt.AppendTo("related.ip", destIP);
        }
    };

    var getHashPath = function (namespace, hashKey) {
        if (hashKey === "imphash") {
            return namespace + ".pe.imphash";
        }

        return namespace + ".hash." + hashKey;
    };

    var emptyHashRegex = /^0*$/;

    var hashIsEmpty = function (value) {
        if (!value) {
            return true;
        }

        return emptyHashRegex.test(value);
    }

    // Adds hashes from the given hashField in the event to the 'hash' key
    // in the specified namespace. It also adds all the hashes to 'related.hash'.
    var addHashes = function (evt, namespace, hashField) {
        var hashes = evt.Get(hashField);
        if (!hashes) {
            return;
        }
        evt.Delete(hashField);
        hashes.split(",").forEach(function (hash) {
            var parts = hash.split("=");
            if (parts.length !== 2) {
                return;
            }

            var key = parts[0].toLowerCase();
            var value = parts[1].toLowerCase();

            if (hashIsEmpty(value)) {
                return;
            }

            var path = getHashPath(namespace, key);

            evt.Put(path, value);
            evt.AppendTo("related.hash", value);
        });
    };

    var splitFileHashes = function (evt) {
        addHashes(evt, "file", "winlog.event_data.Hashes");
    };

    var splitFileHash = function (evt) {
        addHashes(evt, "file", "winlog.event_data.Hash");
    };

    var splitProcessHashes = function (evt) {
        addHashes(evt, "process", "winlog.event_data.Hashes");
    };

    var removeEmptyEventData = function (evt) {
        var eventData = evt.Get("winlog.event_data");
        if (eventData && Object.keys(eventData).length === 0) {
            evt.Delete("winlog.event_data");
        }
    };

    var translateDnsQueryStatus = function (evt) {
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
    var splitDnsQueryResults = function (evt) {
        var results = evt.Get("winlog.event_data.QueryResults");
        if (!results) {
            return;
        }
        results = results.split(";");

        var answers = [];
        var ips = [];
        for (var i = 0; i < results.length; i++) {
            var answer = results[i];
            if (!answer) {
                continue;
            }

            if (answer.startsWith("type:")) {
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
                if (net.isIP(answer)) {
                    ips.push(answer);

                    // Synthesize record type based on IP address type.
                    var type = "A";
                    if (answer.indexOf(":") !== -1) {
                        type = "AAAA";
                    }
                    answers.push({
                        type: type,
                        data: answer,
                    });
                }
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

    var setAdditionalSignatureFields = function (evt) {
        var signed = evt.Get("winlog.event_data.Signed");
        if (!signed) {
            return;
        }
        evt.Put("file.code_signature.signed", true);
        var signatureStatus = evt.Get("winlog.event_data.SignatureStatus");
        evt.Put("file.code_signature.valid", signatureStatus === "Valid");
    };

    var setAdditionalFileFieldsFromPath = function (evt) {
        var filePath = evt.Get("file.path");
        if (!filePath) {
            return;
        }

        evt.Put("file.name", path.basename(filePath));
        evt.Put("file.directory", path.dirname(filePath));

        // path returns extensions with a preceding ., e.g.: .tmp, .png
        // according to ecs the expected format is without it, so we need to remove it.
        var ext = path.extname(filePath);
        if (!ext) {
            return;
        }

        if (ext.charAt(0) === ".") {
            ext = ext.substr(1);
        }
        evt.Put("file.extension", ext);
    };

    // https://docs.microsoft.com/en-us/windows/win32/sysinfo/registry-hives
    var commonRegistryHives = {
        HKEY_CLASSES_ROOT: "HKCR",
        HKCR: "HKCR",
        HKEY_CURRENT_CONFIG: "HKCC",
        HKCC: "HKCC",
        HKEY_CURRENT_USER: "HKCU",
        HKCU: "HKCU",
        HKEY_DYN_DATA: "HKDD",
        HKDD: "HKDD",
        HKEY_LOCAL_MACHINE: "HKLM",
        HKLM: "HKLM",
        HKEY_PERFORMANCE_DATA: "HKPD",
        HKPD: "HKPD",
        HKEY_USERS: "HKU",
        HKU: "HKU",
    };

    var qwordRegex = new RegExp(/QWORD \(((0x\d{8})-(0x\d{8}))\)/, "i");
    var dwordRegex = new RegExp(/DWORD \((0x\d{8})\)/, "i");

    var setRegistryFields = function (evt) {
        var path = evt.Get("winlog.event_data.TargetObject");
        if (!path) {
            return;
        }
        evt.Put("registry.path", path);
        var pathTokens = path.split("\\");
        var hive = commonRegistryHives[pathTokens[0]];
        if (hive) {
            evt.Put("registry.hive", hive);
            pathTokens.splice(0, 1);
            if (pathTokens.length > 0) {
                evt.Put("registry.key", pathTokens.join("\\"));
            }
        }
        var value = pathTokens[pathTokens.length - 1];
        evt.Put("registry.value", value);
        var data = evt.Get("winlog.event_data.Details");
        if (!data) {
            return;
        }
        // sysmon only returns details of a registry modification
        // if it's a qword or dword
        var dataType;
        var dataValue;
        var match = qwordRegex.exec(data);
        if (match && match.length > 0) {
            var parsedHighByte = parseInt(match[2]);
            var parsedLowByte = parseInt(match[3]);
            if (!isNaN(parsedHighByte) && !isNaN(parsedLowByte)) {
                dataValue = "" + ((parsedHighByte << 8) + parsedLowByte);
                dataType = "SZ_QWORD";
            }
        } else {
            match = dwordRegex.exec(data);
            if (match && match.length > 0) {
                var parsedValue = parseInt(match[1]);
                if (!isNaN(parsedValue)) {
                    dataType = "SZ_DWORD";
                    dataValue = "" + parsedValue;
                }
            }
        }
        if (dataType) {
            evt.Put("registry.data.strings", [dataValue]);
            evt.Put("registry.data.type", dataType);
        }
    };

    // Event ID 1 - Process Create.
    var event1 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["start", "process_start"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.CommandLine",
                    to: "process.command_line",
                },
                {
                    from: "winlog.event_data.CurrentDirectory",
                    to: "process.working_directory",
                },
                {
                    from: "winlog.event_data.ParentProcessGuid",
                    to: "process.parent.entity_id",
                },
                {
                    from: "winlog.event_data.ParentProcessId",
                    to: "process.parent.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.ParentImage",
                    to: "process.parent.executable",
                },
                {
                    from: "winlog.event_data.ParentCommandLine",
                    to: "process.parent.command_line",
                },
                {
                    from: "winlog.event_data.OriginalFileName",
                    to: "process.pe.original_file_name",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.Company",
                    to: "process.pe.company",
                },
                {
                    from: "winlog.event_data.Description",
                    to: "process.pe.description",
                },
                {
                    from: "winlog.event_data.FileVersion",
                    to: "process.pe.file_version",
                },
                {
                    from: "winlog.event_data.Product",
                    to: "process.pe.product",
                },
            ],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(splitProcessArgs)
        .Add(addUser)
        .Add(splitProcessHashes)
        .Add(setParentProcessNameUsingExe)
        .Add(splitParentProcessArgs)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 2 - File creation time changed.
    var event2 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.TargetFilename",
                    to: "file.path",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 3 - Network connection detected.
    var event3 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["network"],
                type: ["connection", "start", "protocol"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.Protocol",
                    to: "network.transport",
                },
                {
                    from: "winlog.event_data.SourceIp",
                    to: "source.ip",
                    type: "ip",
                },
                {
                    from: "winlog.event_data.SourceHostname",
                    to: "source.domain",
                    type: "string",
                },
                {
                    from: "winlog.event_data.SourcePort",
                    to: "source.port",
                    type: "long",
                },
                {
                    from: "winlog.event_data.DestinationIp",
                    to: "destination.ip",
                    type: "ip",
                },
                {
                    from: "winlog.event_data.DestinationHostname",
                    to: "destination.domain",
                    type: "string",
                },
                {
                    from: "winlog.event_data.DestinationPort",
                    to: "destination.port",
                    type: "long",
                },
                {
                    from: "winlog.event_data.DestinationPortName",
                    to: "network.protocol",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setRelatedIP)
        .Add(setProcessNameUsingExe)
        .Add(addUser)
        .Add(addNetworkDirection)
        .Add(addNetworkType)
        .CommunityID()
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 4 - Sysmon service state changed.
    var event4 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                from: "winlog.event_data.UtcTime",
                to: "@timestamp",
            }, ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 5 - Process terminated.
    var event5 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["end", "process_end"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 6 - Driver loaded.
    var event6 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["driver"],
                type: ["start"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ImageLoaded",
                    to: "file.path",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.Signature",
                    to: "file.code_signature.subject_name",
                },
                {
                    from: "winlog.event_data.SignatureStatus",
                    to: "file.code_signature.status",
                },
            ],
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setAdditionalSignatureFields)
        .Add(splitFileHashes)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 7 - Image loaded.
    var event7 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.ImageLoaded",
                    to: "file.path",
                },
                {
                    from: "winlog.event_data.OriginalFileName",
                    to: "file.pe.original_file_name",
                },

            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.Signature",
                    to: "file.code_signature.subject_name",
                },
                {
                    from: "winlog.event_data.SignatureStatus",
                    to: "file.code_signature.status",
                },
                {
                    from: "winlog.event_data.Company",
                    to: "file.pe.company",
                },
                {
                    from: "winlog.event_data.Description",
                    to: "file.pe.description",
                },
                {
                    from: "winlog.event_data.FileVersion",
                    to: "file.pe.file_version",
                },
                {
                    from: "winlog.event_data.Product",
                    to: "file.pe.product",
                },
            ],
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setAdditionalSignatureFields)
        .Add(setProcessNameUsingExe)
        .Add(splitFileHashes)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 8 - CreateRemoteThread detected.
    var event8 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.SourceProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.SourceProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.SourceImage",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 9 - RawAccessRead detected.
    var event9 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.Device",
                    to: "file.path",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 10 - Process accessed.
    var event10 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["access"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.SourceProcessGUID",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.SourceProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.SourceThreadId",
                    to: "process.thread.id",
                    type: "long",
                },
                {
                    from: "winlog.event_data.SourceImage",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 11 - File created.
    var event11 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"],
                type: ["creation"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.TargetFilename",
                    to: "file.path",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 12 - Registry object added or deleted.
    var event12 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["configuration", "registry"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setRegistryFields)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 13 - Registry value set.
    var event13 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["configuration", "registry"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setRegistryFields)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 14 - Registry object renamed.
    var event14 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["configuration", "registry"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setRegistryFields)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 15 - File stream created.
    var event15 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"],
                type: ["access"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.TargetFilename",
                    to: "file.path",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(setProcessNameUsingExe)
        .Add(splitFileHash)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 16 - Sysmon config state changed.
    var event16 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["configuration"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                from: "winlog.event_data.UtcTime",
                to: "@timestamp",
            }, ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 17 - Pipe Created.
    var event17 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"], // pipes are files
                type: ["creation"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.PipeName",
                    to: "file.name",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 18 - Pipe Connected.
    var event18 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"], // pipes are files
                type: ["access"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.PipeName",
                    to: "file.name",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 19 - WmiEventFilter activity detected.
    var event19 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                from: "winlog.event_data.UtcTime",
                to: "@timestamp",
            }, ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 20 - WmiEventConsumer activity detected.
    var event20 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.Destination",
                    to: "process.executable",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 21 - WmiEventConsumerToFilter activity detected.
    var event21 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                from: "winlog.event_data.UtcTime",
                to: "@timestamp",
            }, ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 22 - DNSEvent (DNS query).
    var event22 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["network"],
                type: ["connection", "protocol", "info"],
            },
            target: "event",
        })
        .AddFields({
            fields: {
                protocol: "dns",
            },
            target: "network",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.QueryName",
                    to: "dns.question.name",
                },
                {
                    from: "winlog.event_data.QueryStatus",
                    to: "sysmon.dns.status",
                },
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
            target_subdomain_field: "dns.question.subdomain",
            target_etld_field: "dns.question.top_level_domain",
        })
        .Add(setRuleName)
        .Add(translateDnsQueryStatus)
        .Add(splitDnsQueryResults)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 23 - FileDelete (A file delete was detected).
    var event23 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["file"], // pipes are files
                type: ["deletion"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.RuleName",
                    to: "rule.name",
                },
                {
                    from: "winlog.event_data.TargetFilename",
                    to: "file.path",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.Archived",
                    to: "sysmon.file.archived",
                    type: "boolean",
                },
                {
                    from: "winlog.event_data.IsExecutable",
                    to: "sysmon.file.is_executable",
                    type: "boolean",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(splitProcessHashes)
        .Add(setProcessNameUsingExe)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 24 - ClipboardChange (New content in the clipboard).
    var event24 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.Archived",
                    to: "sysmon.file.archived",
                    type: "boolean",
                },
                {
                    from: "winlog.event_data.IsExecutable",
                    to: "sysmon.file.is_executable",
                    type: "boolean",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(splitProcessHashes)
        .Add(setProcessNameUsingExe)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 25 - ProcessTampering (Process image change).
    var event25 = new processor.Chain()
        .Add(parseUtcTime)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["change"],
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ProcessGuid",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.ProcessId",
                    to: "process.pid",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Image",
                    to: "process.executable",
                },
                {
                    from: "winlog.event_data.Archived",
                    to: "sysmon.file.archived",
                    type: "boolean",
                },
                {
                    from: "winlog.event_data.IsExecutable",
                    to: "sysmon.file.is_executable",
                    type: "boolean",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setRuleName)
        .Add(addUser)
        .Add(splitProcessHashes)
        .Add(setProcessNameUsingExe)
        .Add(setAdditionalFileFieldsFromPath)
        .Add(removeEmptyEventData)
        .Build();

    // Event ID 255 - Error report.
    var event255 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [{
                    from: "winlog.event_data.UtcTime",
                    to: "@timestamp",
                },
                {
                    from: "winlog.event_data.ID",
                    to: "error.code",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    return {
        1: event1.Run,
        2: event2.Run,
        3: event3.Run,
        4: event4.Run,
        5: event5.Run,
        6: event6.Run,
        7: event7.Run,
        8: event8.Run,
        9: event9.Run,
        10: event10.Run,
        11: event11.Run,
        12: event12.Run,
        13: event13.Run,
        14: event14.Run,
        15: event15.Run,
        16: event16.Run,
        17: event17.Run,
        18: event18.Run,
        19: event19.Run,
        20: event20.Run,
        21: event21.Run,
        22: event22.Run,
        23: event23.Run,
        24: event24.Run,
        25: event25.Run,
        255: event255.Run,

        process: function (evt) {
            var event_id = evt.Get("winlog.event_id");
            var processor = this[event_id];
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
