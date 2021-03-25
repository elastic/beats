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
        "IPACCESSLOGP": newDissect("list %{cisco.ios.access_list} %{cisco.ios.outcome} " +
            "%{network.transport} %{source.address}(%{source.port}) -> " +
            "%{destination.address}(%{destination.port}), %{source.packets} packet"),

        "IPACCESSLOGDP": newDissect("list %{cisco.ios.access_list} %{cisco.ios.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{icmp.type}/%{icmp.code}), %{source.packets} packet"),

        "IPACCESSLOGRP": newDissect("list %{cisco.ios.access_list} %{cisco.ios.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),

        "IPACCESSLOGSP": newDissect("list %{cisco.ios.access_list} %{cisco.ios.outcome} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{igmp.type}), %{source.packets} packet"),

        "IPACCESSLOGNP": newDissect("list %{cisco.ios.access_list} %{cisco.ios.outcome} " +
            "%{network.iana_number} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),
    };
    // Add IPv6 log message patterns.
    accessListMessagePatterns.ACCESSLOGP = accessListMessagePatterns.IPACCESSLOGP;
    accessListMessagePatterns.ACCESSLOGSP = accessListMessagePatterns.IPACCESSLOGSP;
    accessListMessagePatterns.ACCESSLOGDP = accessListMessagePatterns.IPACCESSLOGDP;
    accessListMessagePatterns.ACCESSLOGNP = accessListMessagePatterns.IPACCESSLOGNP;

    var otherMessagePatterns = {
        "NOVALIDKEY": newDissect("%{_tmp.message}"),
        "CALL_PRESERVED": newDissect("%{_tmp.message}"),
        "LOGIN_SUCCESS": newDissect("%{cisco.ios.action} %{cisco.ios.outcome} [user: %{user.name}] [Source: %{client.address}] [localport: %{destination.port}] at %{}"),
        "LOGOUT": newDissect("User %{user.name} has %{cisco.ios.action} %{cisco.ios.session.type} session %{cisco.ios.session.id|integer}(%{client.address})"),
        "INVALID_RP_JOIN": newDissect("Received (%{cisco.ios.tree}, %{IP:cisco.ios.multicast.group}) %{cisco.ios.action} from %{client.address} for %{cisco.ios.outcome} RP %{IP:server.address}"),
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
                    'Jan _2 15:04:05 MST',
                ],
                ignore_missing: true,
            }).Run(evt);
        })
        .Add(function(evt) {
            evt.Delete("_tmp");
        })
        .Build();

    var processFrontMessage = new processor.Chain()
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
        .Build();

    var processAccessMessage = new processor.Chain()
        .Add(function(evt) {
            var eventCode = evt.Get("event.code");
            if (!eventCode) {
                return;
            }

            var dissect = accessListMessagePatterns[eventCode];
            if (dissect) {
                dissect(evt);
                coerceNumbers(evt);
                normalizeEventOutcome(evt);
                setNetworkType(evt);
                setRelatedIP(evt);
                setECSCategorization(evt);
                return;
            }
        })
        .CommunityID()
        .Build();

    var processOtherMessage = new processor.Chain()
        .Add(function(evt) {
            var eventCode = evt.Get("event.code");
            if (!eventCode) {
                return;
            }

            var dissect = otherMessagePatterns[eventCode];
            if (dissect) {
                dissect(evt);
                coerceNumbers(evt);
                normalizeEventOutcome(evt);
                setNetworkType(evt);
                setRelatedIP(evt);
                setECSCategorization(evt);
                return;
            }
        })
        .Add(function(evt) {
            evt.Delete("message");
            evt.Rename("_message", "message");
        })
        .Add(function(evt) {
            evt.Delete("_tmp");
        })
        .Build();

    var coerceNumbers = new processor.Convert({
        fields: [
            {from: "server.address", to: "destination.address", type: "ip", ignore_missing: true},
            {from: "server.ip", to: "destination.ip", type: "ip", ignore_missing: true},
            {from: "destination.address", to: "destination.ip", type: "ip", ignore_missing: true},
            {from: "destination.port", type: "long", ignore_missing: true},
            {from: "client.address", to: "source.address", type: "ip", ignore_missing: true},
            {from: "client.ip", to: "source.ip", type: "ip", ignore_missing: true},
            {from: "source.address", to: "source.ip", type: "ip", ignore_missing: true},
            {from: "source.port", type: "long", ignore_missing: true},
            {from: "source.packets", type: "long", ignore_missing: true},
            {from: "source.packets", to: "network.packets", type: "long", ignore_missing: true},
        ],
        ignore_missing: true,
    }).Run;

    var normalizeEventOutcome = function(evt) {
        var outcome = evt.Get("cisco.ios.outcome");
        if (outcome) {
            switch (outcome.toLowerCase()) {
                case "invalid":
                case "denied":
                    evt.AppendTo("event.type", "denied");
                    evt.Put("event.outcome", "failure");
                    break;
                case "success":
                case "permitted":
                    evt.AppendTo("event.type", "allowed");
                    evt.Put("event.outcome", "success");
                    break;
            }
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
        var source_ip = event.Get("source.ip");
        if (source_ip) {
            event.AppendTo("related.ip", event.Get("source.ip"));
        }
        var source_ip = event.Get("destination.ip");
        if (source_ip) {
            event.AppendTo("related.ip", event.Get("destination.ip"));
        }
    };

    var setECSCategorization = function(event) {
        event.Put("event.kind", "event");
        event.AppendTo("event.category", "network");
        event.AppendTo("event.category", "network_traffic");
        event.AppendTo("event.type", "connection");
        event.AppendTo("event.type", "firewall");
    };

    return {
        process: function(evt) {
            copyOriginalMessage.Run(evt);

            if (evt.Get("input.type") === "log") {
                parseSyslogFileHeader.Run(evt);
            }
            processFrontMessage.Run(evt);
            var eventCode = evt.Get("event.code");
            var dissectAccess = accessListMessagePatterns[eventCode];
            var dissectOther = otherMessagePatterns[eventCode];
            if (dissectAccess) {
                processAccessMessage.Run(evt);
            } else if (dissectOther) {
                processOtherMessage.Run(evt);
            }
        },
    };
})();

function process(evt) {
    ciscoIOS.process(evt);
}
