// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var ciscoIOS = (function() {
    var processor = require("processor");

    var newDissect = function(pattern) {
        return new processor.Dissect({
            "tokenizer": pattern,
            "field": "message",
            "target_prefix": "",
        }).Run;
    };

    var accessListMessagePatterns = {
        "IPACCESSLOGP": newDissect("list %{cisco.ios.access_list} %{event.outcome} " +
            "%{network.transport} %{source.address}(%{source.port}) -> " +
            "%{destination.address}(%{destination.port}), %{source.packets} packet"),

        "IPACCESSLOGDP": newDissect("list %{cisco.ios.access_list} %{event.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{icmp.type}/%{icmp.code}), %{source.packets} packet"),

        "IPACCESSLOGRP": newDissect("list %{cisco.ios.access_list} %{event.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),

        "IPACCESSLOGSP": newDissect("list %{cisco.ios.access_list} %{event.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{igmp.type}), %{source.packets} packet"),

        "IPACCESSLOGNP": newDissect("list %{cisco.ios.access_list} %{event.outcome} " +
            "%{network.iana_number} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),
    };
    // Add IPv6 log message patterns.
    accessListMessagePatterns.ACCESSLOGP = accessListMessagePatterns.IPACCESSLOGP;
    accessListMessagePatterns.ACCESSLOGSP = accessListMessagePatterns.IPACCESSLOGSP;
    accessListMessagePatterns.ACCESSLOGDP = accessListMessagePatterns.IPACCESSLOGDP;
    accessListMessagePatterns.ACCESSLOGNP = accessListMessagePatterns.IPACCESSLOGNP;


    var ciscoLogPatterns = {
        "SSH" : {
            "SSH2_UNEXPECTED_MSG": newDissect("Unexpected message type has arrived. Terminating the connection from %{source.address}"),
        },
        "SEC": accessListMessagePatterns,
        "FMANFP": accessListMessagePatterns,
        "SEC_LOGIN": {
            "LOGIN_SUCCESS": newDissect("Login Success [user: %{user.name}] [Source: %{source.address}] [localport: %{source.port}] at %{}"),
        },
        "SYS": {
            "LOGOUT": newDissect("User %{user.name} has exited tty session %{cisco.tty_session_id}(%{source.address})"),
            "CONFIG_I": newDissect("Configured from %{cisco.comfig_source} by %{user.name} on %{cisco.terminal} (%{source.address})"),
        },
        "LINK":  {
            "UPDOWN": newDissect("Interface %{interface.name}, changed state to %{interface.state}"),
        },
        "LINEPROTO": {
            "UPDOWN": newDissect("Line protocol on Interface %{source.interface.name}, changed state to %{source.interface.state}"),
        },
        "CDP": {
            "NATIVE_VLAN_MISMATCH": newDissect("Native VLAN mismatch discovered on %{source.interface.name} (%{source.interface.vlan}), with %{destination.address} %{destination.interface.name} (%{destination.interface.vlan})."),
            "DUPLEX_MISMATCH": newDissect("duplex mismatch discovered on %{source.interface.name} (%{source.interface.name}), with %{destination.address} %{destination.interface.name} (%{destination.interface.duplex})."),
        },
        "IP": {
            "DUPADDR": newDissect("Duplicate address %{source.address} on %{source.interface.name}, sourced by %{source.mac}")
        }
    }; 

    var setLogLevel = function(evt) {
        var severity = evt.Get("event.severity");

        var levelKeyword = "";
        switch (severity) {
            case 0:
                levelKeyword = "emergencies";
                break;
            case 1:
                levelKeyword = "alerts";
                break;
            case 2:
                levelKeyword = "critical";
                break;
            case 3:
                levelKeyword = "errors";
                break;
            case 4:
                levelKeyword = "warnings";
                break;
            case 5:
                levelKeyword = "notifications";
                break;
            case 6:
                levelKeyword = "informational";
                break;
            case 7:
                levelKeyword = "debugging";
                break;
            default:
                return;
        }

        evt.Put("log.level", levelKeyword);
    };

    var copyOriginalMessage = new processor.Convert({
        fields: [
            {from: "message", to: "log.original"},
        ],
        mode: "copy",
    });

    var parseSyslogFileHeader = new processor.Chain()
        .Dissect({
            tokenizer: "%{_tmp.ts->} %{+_tmp.ts} %{+_tmp.ts->} %{log.source.address} %{event.sequence}: %{_tmp.timestamp}: %{_tmp.message}",
            field: "message",
            target_prefix: "",
        })
        .Convert({
            fields: [
                {from: "_tmp.message", to: "message"},
            ],
            mode: "rename",
        })
        .Convert({
            fields: [
                {from: "event.sequence", type: "long"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            processor.Timestamp({
                field: "_tmp.timestamp",
                target_field: "@timestamp",
                timezone: evt.Get("event.timezone"),
                layouts: [
                    'Jan _2 15:04:05.999',
                    'Jan _2 15:04:05.999 MST',
                    'Jan _2 2020 15:04:05.999 MST',
                ],
                ignore_missing: true,
            }).Run(evt);
        })
        .Add(function(evt) {
            evt.Delete("_tmp");
        })
        .Build();

    var processMessage = new processor.Chain()
        // Parse the header of the message that is common to all messages.
        .Dissect({
            "tokenizer": "%{}%%{cisco.ios.facility}-%{_event_severity}-%{event.code}: %{_message}",
            "field": "message",
            "target_prefix": "",
        })
        .Add(function(evt) {
            evt.Delete("message");
            evt.Rename("_message", "message");
            evt.Delete("event.severity");
            evt.Rename("_event_severity", "event.severity");
        })
        .Convert({
            fields: [
                {from: "event.severity", type: "long"},
            ],
        })
        .Add(setLogLevel)
        
        .Add(function(evt) {
            
            var facility = evt.Get("cisco.ios.facility");
            if (!facility) {
                return;
            }
            var facility_patterns = ciscoLogPatterns[facility]
            // If no patterns for this facility then return
            if (!facility_patterns) {
                return;
            }else{
                var eventCode = evt.Get("event.code");
                if (!eventCode) {
                    return;
                }
                // Use a specific dissect pattern based on the event.code.
                var dissect = facility_patterns[eventCode];
                if (dissect) {
                    dissect(evt);
                    coerceNumbers(evt);
                    normalizeEventOutcome(evt);
                    setNetworkType(evt);
                    setRelatedIP(evt);
                    evt.AppendTo("tags", "cisco-ios-dissect");
                    return;
                }
    
                // Add Special Case for Access Logs so they can be treated like firewall rules
                if (eventCode =~ "ACCESS") {
                    evt.Put("event.category", "network_traffic");
                    evt.Put("event.type", "firewall");
                }
            }
        })
        .CommunityID()
        .Build();

    var coerceNumbers = new processor.Convert({
        fields: [
            {from: "destination.address", to: "destination.ip", type: "ip"},
            {from: "destination.port", type: "long"},
            {from: "source.address", to: "source.ip", type: "ip"},
            {from: "source.port", type: "long"},
            {from: "source.packets", type: "long"},
            {from: "source.packets", to: "network.packets", type: "long"},
            {from: "icmp.type", type: "long"},
            {from: "icmp.code", type: "long"},
            {from: "igmp.type", type: "long"},
        ],
        ignore_missing: true,
    }).Run;

    var normalizeEventOutcome = function(evt) {
        var outcome = evt.Get("event.outcome");
        if (!outcome) {
            return;
        }
        switch (outcome) {
            case "denied":
                evt.Put("event.outcome", "deny");
                break;
            case "permitted":
                evt.Put("event.outcome", "allow");
                break;
        }
    };

    var setNetworkType = function(event) {
        var ip = event.Get("source.ip");
        if (!ip) {
            return;
        }

        if (ip.indexOf(".") !== -1) {
            event.Put("network.type", "ipv4");
        } else {
            event.Put("network.type", "ipv6");
        }
    };

    var setRelatedIP = function(event) {
        var src_ip = event.Get("source.ip");
        if (src_ip) {
            event.AppendTo("related.ip", src_ip);
        }

        var dst_ip = event.Get("destination.ip");
        if (dst_ip) {
            event.AppendTo("related.ip", dst_ip);
        }
    };

    return {
        process: function(evt) {
            copyOriginalMessage.Run(evt);

            if (evt.Get("input.type") === "log") {
                parseSyslogFileHeader.Run(evt);
            }

            processMessage.Run(evt);
        },
    };
})();

function process(evt) {
    ciscoIOS.process(evt);
}
