// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var crowdstrikeFalcon = (function () {
    var processor = require("processor");

    var convertUnderscore = function (text) {
        return text.split(/(?=[A-Z])/).join('_').toLowerCase();
    };

    var convertToMSEpoch = function (evt, field) {
        var timestamp = evt.Get(field);
        if (timestamp) {
            if (timestamp < 100000000000) { // check if we have a seconds timestamp, this is roughly 1973 in MS
                evt.Put(field, timestamp * 1000);
            }
            (new processor.Timestamp({
                field: field,
                target_field: field,
                timezone: "UTC",
                layouts: ["UNIX_MS"]
            })).Run(evt);
        }
    };

    var normalizeProcess = function (evt) {
        var commandLine = evt.Get("crowdstrike.event.CommandLine")
        if (commandLine && commandLine.trim() !== "") {
            var args = commandLine.split(' ').filter(function (arg) {
                return arg !== "";
            });
            var executable = args[0]

            evt.Put("process.command_line", commandLine)
            evt.Put("process.args", args)
            evt.Put("process.executable", executable)
        }
    }

    var normalizeSourceDestination = function (evt) {
        var localAddress = evt.Get("crowdstrike.event.LocalAddress");
        var localPort = evt.Get("crowdstrike.event.LocalPort");
        var remoteAddress = evt.Get("crowdstrike.event.RemoteAddress");
        var remotePort = evt.Get("crowdstrike.event.RemotePort");
        if (evt.Get("crowdstrike.event.ConnectionDirection") === "1") {
            evt.Put("network.direction", "inbound")
            evt.Put("source.ip", remoteAddress)
            evt.Put("source.port", remotePort)
            evt.Put("destination.ip", localAddress)
            evt.Put("destination.port", localPort)
        } else {
            evt.Put("network.direction", "outbound")
            evt.Put("destination.ip", remoteAddress)
            evt.Put("destination.port", remotePort)
            evt.Put("source.ip", localAddress)
            evt.Put("source.port", localPort)
        }
    }

    var normalizeEventAction = function (evt) {
        var eventType = evt.Get("crowdstrike.metadata.eventType")
        evt.Put("event.action", convertUnderscore(eventType))
    }

    var normalizeUsername = function (evt) {
        var username = evt.Get("crowdstrike.event.UserName")
        if (!username || username === "") {
            username = evt.Get("crowdstrike.event.UserId")
        }
        if (username && username !== "") {
            evt.Put("user.name", username)
            if (username.split('@').length == 2) {
                evt.Put("user.email", username)
            }
        }
    }

    // DetectionSummaryEvent
    var convertDetectionSummaryEvent = new processor.Chain()
        .AddFields({
            fields: {
                kind: "alert",
                category: ["malware"],
                type: ["info"],
                dataset: "crowdstrike.falcon_endpoint",
            },
            target: "event",
        })
        .AddFields({
            fields: {
                type: "falcon",
            },
            target: "agent",
        })
        .Convert({
            fields: [{
                    from: "crowdstrike.event.LocalIP",
                    to: "source.ip",
                    type: "ip"
                }, {
                    from: "crowdstrike.event.ProcessId",
                    to: "process.pid"
                }, {
                    from: "crowdstrike.event.ParentImageFileName",
                    to: "process.parent.executable"
                }, {
                    from: "crowdstrike.event.ParentCommandLine",
                    to: "process.parent.command_line"
                }, {
                    from: "crowdstrike.event.PatternDispositionDescription",
                    to: "event.action",
                }, {
                    from: "crowdstrike.event.FalconHostLink",
                    to: "event.url",
                }, {
                    from: "crowdstrike.event.Severity",
                    to: "event.severity",
                }, {
                    from: "crowdstrike.event.DetectDescription",
                    to: "message",
                }, {
                    from: "crowdstrike.event.FileName",
                    to: "process.name",
                }, {
                    from: "crowdstrike.event.UserName",
                    to: "user.name",
                },
                {
                    from: "crowdstrike.event.MachineDomain",
                    to: "user.domain",
                },
                {
                    from: "crowdstrike.event.SensorId",
                    to: "agent.id",
                },
                {
                    from: "crowdstrike.event.ComputerName",
                    to: "host.name",
                },
                {
                    from: "crowdstrike.event.SHA256String",
                    to: "file.hash.sha256",
                },
                {
                    from: "crowdstrike.event.MD5String",
                    to: "file.hash.md5",
                },
                {
                    from: "crowdstrike.event.SHA1String",
                    to: "file.hash.sha1",
                },
                {
                    from: "crowdstrike.event.DetectName",
                    to: "rule.name",
                },
                {
                    from: "crowdstrike.event.DetectDescription",
                    to: "rule.description",
                }
            ],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(function (evt) {
            var tactic = evt.Get("crowdstrike.event.Tactic").toLowerCase()
            var technique = evt.Get("crowdstrike.event.Technique").toLowerCase()
            evt.Put("threat.technique.name", technique)
            evt.Put("threat.tactic.name", tactic)
        })
        .Add(normalizeProcess)
        .Build()

    // IncidentSummaryEvent
    var convertIncidentSummaryEvent = new processor.Chain()
        .AddFields({
            fields: {
                kind: "alert",
                category: ["malware"],
                type: ["info"],
                action: "incident",
                dataset: "crowdstrike.falcon_endpoint",
            },
            target: "event",
        })
        .AddFields({
            fields: {
                type: "falcon",
            },
            target: "agent",
        })
        .Convert({
            fields: [{
                from: "crowdstrike.event.FalconHostLink",
                to: "event.url",
            }],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(function (evt) {
            evt.Put("message", "Incident score " + evt.Get("crowdstrike.event.FineScore"))
        })
        .Add(normalizeProcess)
        .Build()

    // UserActivityAuditEvent
    var convertUserActivityAuditEvent = new processor.Chain()
        .AddFields({
            fields: {
                category: ["iam"],
                type: ["change"],
                dataset: "crowdstrike.falcon_audit",
            },
            target: "event",
        })
        .Convert({
            fields: [{
                from: "crowdstrike.event.OperationName",
                to: "message",
            }, {
                from: "crowdstrike.event.UserIp",
                to: "source.ip",
                type: "ip"
            }],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(normalizeUsername)
        .Add(normalizeEventAction)
        .Build()

    // AuthActivityAuditEvent
    var convertAuthActivityAuditEvent = new processor.Chain()
        .AddFields({
            fields: {
                category: ["authentication"],
                type: ["change"],
                dataset: "crowdstrike.falcon_audit",
            },
            target: "event",
        })
        .Convert({
            fields: [{
                from: "crowdstrike.event.ServiceName",
                to: "message",
            }, {
                from: "crowdstrike.event.UserIp",
                to: "source.ip",
                type: "ip"
            }],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(normalizeUsername)
        .Add(function (evt) {
            evt.Put("event.action", convertUnderscore(evt.Get("crowdstrike.event.OperationName")))
        })
        .Build()

    // FirewallMatchEvent
    var convertFirewallMatchEvent = new processor.Chain()
        .AddFields({
            fields: {
                category: ["network"],
                type: ["start", "connection"],
                outcome: ["unknown"],
                dataset: "crowdstrike.falcon_endpoint",
            },
            target: "event",
        })
        .Convert({
            fields: [{
                    from: "crowdstrike.event.Ipv",
                    to: "network.type",
                }, {
                    from: "crowdstrike.event.PID",
                    to: "process.pid",
                },
                {
                    from: "crowdstrike.event.RuleId",
                    to: "rule.id"
                },
                {
                    from: "crowdstrike.event.RuleName",
                    to: "rule.name"
                },
                {
                    from: "crowdstrike.event.RuleGroupName",
                    to: "rule.ruleset"
                },
                {
                    from: "crowdstrike.event.RuleDescription",
                    to: "rule.description"
                },
                {
                    from: "crowdstrike.event.RuleFamilyID",
                    to: "rule.category"
                },
                {
                    from: "crowdstrike.event.HostName",
                    to: "host.name"
                },
                {
                    from: "crowdstrike.event.Ipv",
                    to: "network.type",
                },
                {
                    from: "crowdstrike.event.EventType",
                    to: "event.code",
                }
            ],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(function (evt) {
            evt.Put("message", "Firewall Rule '" + evt.Get("crowdstrike.event.RuleName") + "' triggered")
        })
        .Add(normalizeEventAction)
        .Add(normalizeProcess)
        .Add(normalizeSourceDestination)
        .Build();

    // RemoteResponseSessionStartEvent
    var convertRemoteResponseSessionStartEvent = new processor.Chain()
        .AddFields({
            fields: {
                type: ["start"],
                dataset: "crowdstrike.falcon_audit",
            },
            target: "event",
        })
        .AddFields({
            fields: {
                message: "Remote response session started",
            },
            target: "",
        })
        .Convert({
            fields: [{
                from: "crowdstrike.event.HostnameField",
                to: "host.name",
            }],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(normalizeUsername)
        .Add(normalizeEventAction)
        .Build()


    // RemoteResponseSessionEndEvent
    var convertRemoteResponseSessionEndEvent = new processor.Chain()
        .AddFields({
            fields: {
                type: ["end"],
                dataset: "crowdstrike.falcon_audit",
            },
            target: "event",
        })
        .AddFields({
            fields: {
                message: "Remote response session ended",
            },
            target: "",
        })
        .Convert({
            fields: [{
                from: "crowdstrike.event.HostnameField",
                to: "host.name",
            }],
            mode: "copy",
            ignore_missing: true,
            fail_on_error: false
        })
        .Add(normalizeUsername)
        .Add(normalizeEventAction)
        .Build()

    return {
        process: new processor.Chain()
            .DecodeJSONFields({
                fields: ["message"],
                target: "crowdstrike",
                process_array: true,
                max_depth: 8
            })
            .Add(function (evt) {
                convertToMSEpoch(evt, "crowdstrike.event.ProcessStartTime")
                convertToMSEpoch(evt, "crowdstrike.event.ProcessEndTime")
                convertToMSEpoch(evt, "crowdstrike.event.IncidentStartTime")
                convertToMSEpoch(evt, "crowdstrike.event.IncidentEndTime")
                convertToMSEpoch(evt, "crowdstrike.event.StartTimestamp")
                convertToMSEpoch(evt, "crowdstrike.event.EndTimestamp")
                convertToMSEpoch(evt, "crowdstrike.event.UTCTimestamp")
                convertToMSEpoch(evt, "crowdstrike.metadata.eventCreationTime")
            })
            .Add(function (evt) {
                evt.Delete("message");
                evt.Delete("host.name");
            })
            .Convert({
                fields: [{
                    from: "crowdstrike.metadata.eventCreationTime",
                    to: "@timestamp",
                }],
                mode: "copy",
                ignore_missing: false,
                fail_on_error: true
            })
            .Add(function (evt) {
                var eventType = evt.Get("crowdstrike.metadata.eventType")
                var outcome = evt.Get("crowdstrike.event.Success")

                evt.Put("event.kind", "event")

                if (outcome === true) {
                    evt.Put("event.outcome", "success")
                } else if (outcome === false) {
                    evt.Put("event.outcome", "failure")
                } else {
                    evt.Put("event.outcome", "unknown")
                }

                switch (eventType) {
                    case "DetectionSummaryEvent":
                        convertDetectionSummaryEvent.Run(evt)
                        break;

                    case "IncidentSummaryEvent":
                        convertIncidentSummaryEvent.Run(evt)
                        break;

                    case "UserActivityAuditEvent":
                        convertUserActivityAuditEvent.Run(evt)
                        break;

                    case "FirewallMatchEvent":
                        convertFirewallMatchEvent.Run(evt)
                        break;

                    case "AuthActivityAuditEvent":
                        convertAuthActivityAuditEvent.Run(evt)
                        break;

                    case "RemoteResponseSessionStartEvent":
                        convertRemoteResponseSessionStartEvent.Run(evt);
                        break;

                    case "RemoteResponseSessionEndEvent":
                        convertRemoteResponseSessionEndEvent.Run(evt);
                        break;

                    default:
                        break;
                }
            })
            .Build()
            .Run,
    };
})();

function process(evt) {
    crowdstrikeFalcon.process(evt);
}
