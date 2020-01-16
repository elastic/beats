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
        "IPACCESSLOGP": newDissect("list %{cisco.ios.access_list} %{event.action} " +
            "%{network.transport} %{source.address}(%{source.port}) -> " +
            "%{destination.address}(%{destination.port}), %{source.packets} packet"),

        "IPACCESSLOGDP": newDissect("list %{cisco.ios.access_list} %{event.action} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{icmp.type}/%{icmp.code}), %{source.packets} packet"),

        "IPACCESSLOGRP": newDissect("list %{cisco.ios.access_list} %{event.action} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),

        "IPACCESSLOGSP": newDissect("list %{cisco.ios.access_list} %{event.action} " +
            "%{network.transport} %{source.address} -> " +
            "%{destination.address} (%{igmp.type}), %{source.packets} packet"),

        "IPACCESSLOGNP": newDissect("list %{cisco.ios.access_list} %{event.action} " +
            "%{network.iana_number} %{source.address} -> " +
            "%{destination.address}, %{source.packets} packet"),
    };
    // Add IPv6 log message patterns.
    accessListMessagePatterns.ACCESSLOGP = accessListMessagePatterns.IPACCESSLOGP;
    accessListMessagePatterns.ACCESSLOGSP = accessListMessagePatterns.IPACCESSLOGSP;
    accessListMessagePatterns.ACCESSLOGDP = accessListMessagePatterns.IPACCESSLOGDP;
    accessListMessagePatterns.ACCESSLOGNP = accessListMessagePatterns.IPACCESSLOGNP;

    var linkMessagePatterns = {
        "UPDOWN": newDissect("Interface %{interface.name}, %{event.action} to %{interface.status}")
    };

    var lineProtoMessagePatterns = {
        "UPDOWN": newDissect("Line protocol on Interface %{interface.name}, %{event.action} to %{interface.status}")
    };

    var ilPowerMessagePatterns = {
        "POWER_GRANTED": newDissect("Interface %{interface.name}: %{event.action}"),
        "IEEE_DISCONNECT": newDissect("Interface %{interface.name}: %{event.action}")
    };

    var portSecurityMessagePatterns = {
        "PSECURE_VIOLATION": newDissect("%{event.action} occurred, caused by MAC address %{source.mac} on port %{interface.name}")
    };

    var pmMessagePatterns = {
        "ERR_DISABLE": newDissect("%{event.action} on %{interface.name}, putting Gi2/0/12 in %{interface.state} status"),
        "ERR_RECOVER": newDissect("Attempting to %{event.action} on %{interface.name}")
    };

    var platformMessagePatterns = {
        "PS_FAIL": newDissect("Power supply %{cisco.ios.power_supply.number} %{event.action} (Serial number %{cisco.ios.power_supply.serialnumber})"),
        "PS_STATUS": newDissect("PowerSupply %{cisco.ios.power_supply.number} current-status is PS_%{cisco.ios.power_supply.status}")
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
        // Use a specific dissect pattern based on the cisco.event.facility and event.code.
        .Add(function(evt) {
            var facility = evt.Get("cisco.ios.facility");
            if (!facility) {
                return;
            }

            evt.Put("event.provider", facility);

            var eventCode = evt.Get("event.code");
            if (!eventCode) {
                return;
            }

            setLogSource(evt);

            var dissect = "";

            switch (facility) {
                case "SEC":
                    dissect = accessListMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        setNetworkType(evt);
                        setRelatedIP(evt);
                        switch (evt.Get('event.action')) {
                            case "permitted":
                                setCategorization(evt,"event","process","access","success");
                                break;
                            case "denied":
                                setCategorization(evt,"event","process","access","failure");
                                break;
                            default:
                                setCategorization(evt,"event","process","access","unknown");
                                break;
                        }
                        return;
                    }
                    break;
                case "LINK":
                    dissect = linkMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        setInterfaceProperties(evt);
                        if (evt.Get('interface.status') == "up") {
                            setCategorization(evt,"event","driver","access","success");
                        } else {
                            setCategorization(evt,"event","driver","change","failure");
                        }
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        return;
                    }
                    break;
                case "LINEPROTO":
                    dissect = lineProtoMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        setInterfaceProperties(evt);
                        if (evt.Get('interface.status') == "up") {
                            setCategorization(evt,"event","driver","change","success");
                        } else {
                            setCategorization(evt,"event","driver","change","failure");
                        }
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        return;
                    }
                    break;
                case "ILPOWER":
                    dissect = ilPowerMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        setInterfaceProperties(evt);
                        normalizeEventAction(evt);
                        if (evt.Get('event.code') == "POWER_GRANTED") {
                            setCategorization(evt,"event","driver","change","success");
                        } else {
                            setCategorization(evt,"event","driver","change","unknown");
                        }
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        return;
                    }
                    break;
                case "PORT_SECURITY":
                    dissect = portSecurityMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        normalizeInterfaceName(evt);
                        setInterfaceProperties(evt);
                        setCategorization(evt,"event","intrusion_detection","info","failure");
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        return;
                    }
                    break;
                case "PM":
                    dissect = pmMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        normalizeInterfaceName(evt);
                        setInterfaceProperties(evt);
                        if (evt.Get('event.code') == "ERR_RECOVER") {
                            setCategorization(evt,"event","intrusion_detection","info","success");
                        } else {
                            setCategorization(evt,"event","intrusion_detection","info","failure");
                        }
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        return;
                    }
                    break;
                case "PLATFORM":
                    dissect = platformMessagePatterns[eventCode];
                    if (dissect) {
                        dissect(evt);
                        setCategorization(evt,"event","host","info","unknown");
                        coerceNumbers(evt);
                        normalizeEventAction(evt);
                        switch (evt.Get('event.code')) {
                            case "PS_FAIL":
                                setCategorization(evt,"alert","host","change","failure");
                                break;
                            case "PS_STATUS":
                                if (evt.Get('cisco.ios.power_supply.status') == "OK") {
                                    setCategorization(evt,"event","host","info","success");
                                } else {
                                    setCategorization(evt,"event","host","info","failure");
                                }
                                break;
                        }
                        return;
                    }
                    break;
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
            {from: "vlan.id", type: "integer"},
            {from: "interface.slot", type: "integer"},
            {from: "interface.subslot", type: "integer"},
            {from: "interface.port", type: "integer"},
            {from: "log.source.ip", type: "ip"},
            {from: "log.source.port", type: "integer"},

        ],
        ignore_missing: true,
    }).Run;

    var normalizeInterfaceName = function(evt) {
        var ifName = evt.Get("interface.name");
        ifName = ifName.split('.').join('');
        var iSpace = ifName.indexOf(' ');
        if (iSpace > 0) {
            ifName = ifName.substring(0,iSpace);
        }
        evt.Put("interface.name", ifName);
    };

    var normalizeEventAction = function(evt) {
        var action = evt.Get("event.action");
        if (!action) {
            return;
        }

        var iCut = action.indexOf(' (');

        switch (true) {
            case (iCut > 0):
                evt.Put("eventAction", action.substring(0,iCut));
                break;

            case (action.search("Power granted") == 0):
                evt.Put("event.action", "Power granted");
                break;
        }
    };

    var setCategorization = function(evt, kind, category, type, outcome) {
        if (kind) {
            evt.Put("event.kind", kind);
        }
        if (category) {
            evt.Put("event.category", category);
        }
        if (type) {
            evt.Put("event.type", type);
        }
        if (outcome) {
            evt.Put("event.outcome", outcome);
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

    var setLogSource = function(evt) {
        var source_address = evt.Get('log.source.address').split(':');
        if (source_address.length == 2 ) {
            evt.Put("log.source.ip", source_address[0]);
            evt.Put("log.source.port", source_address[1]);
        }
    };

    var setRelatedIP = function(event) {
        event.AppendTo("related.ip", event.Get("source.ip"));
        event.AppendTo("related.ip", event.Get("destination.ip"));
    };

    var setInterfaceProperties = function(event) {
        var ifName = event.Get("interface.name");
        if (!ifName) {
            return;
        }

        switch (true) {
            case (ifName.search(/Gi\d/) == 0):
                ifName = ifName.replace("Gi","GigabitEthernet");
                event.Put("interface.name", ifName);
                break;
        }

        var iFirstDigit = ifName.search(/\d/);
        var ifType = ifName.substring(0,iFirstDigit);
        var ifNumbers = ifName.substring(iFirstDigit).split('/');

        switch (true) {
            case (ifType == "Vlan"):
                event.Put("interface.type", "vlan");
                event.Put("vlan.id", ifNumbers[0]);
                break;

            default:
                event.Put("interface.type", ifType);
                break;
        }

        switch (ifNumbers.length) {
            case 1:
                event.Put("interface.port", ifNumbers[0]);
                break;

            case 2:
                event.Put("interface.slot", ifNumbers[0]);
                event.Put("interface.port", ifNumbers[1]);
                break;

            case 3:
                event.Put("interface.slot", ifNumbers[0]);
                event.Put("interface.subslot", ifNumbers[1]);
                event.Put("interface.port", ifNumbers[2]);
                break;
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
