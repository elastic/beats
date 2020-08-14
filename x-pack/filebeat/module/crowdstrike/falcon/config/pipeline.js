// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var crowdstrikeFalconProcessor = (function () {
    var processor = require("processor");

    // conversion helpers
    function convertUnderscore(text) {
        return text.split(/(?=[A-Z])/).join('_').toLowerCase();
    }

    function convertToMSEpoch(evt, field) {
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
    }

    function convertProcess(evt) {
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

    function convertSourceDestination(evt) {
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
        evt.AppendTo("related.ip", remoteAddress)
        evt.AppendTo("related.ip", localAddress)
    }

    function convertEventAction(evt) {
        evt.Put("event.action", convertUnderscore(evt.Get("crowdstrike.metadata.eventType")))
    }

    function convertUsername(evt) {
        var username = evt.Get("crowdstrike.event.UserName")
        if (!username || username === "") {
            username = evt.Get("crowdstrike.event.UserId")
        }
        if (username && username !== "") {
            evt.Put("user.name", username)
            if (username.split('@').length == 2) {
                evt.Put("user.email", username)
            }
            evt.AppendTo("related.user", username)
        }
    }

    // event processors by type
    var eventProcessors = {
        DetectionSummaryEvent: new processor.Chain()
            .AddFields({
                fields: {
                    "event.kind": "alert",
                    "event.category": ["malware"],
                    "event.type": ["info"],
                    "event.dataset": "crowdstrike.falcon_endpoint",
                    "agent.type": "falcon",
                },
                target: "",
            })
            .Convert({
                fields: [{
                        from: "crowdstrike.event.LocalIP",
                        to: "source.ip",
                        type: "ip"
                    }, {
                        from: "crowdstrike.event.LocalIP",
                        to: "related.ip",
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
                convertProcess(evt)
            })
            .Build(),

        IncidentSummaryEvent: new processor.Chain()
            .AddFields({
                fields: {
                    "event.kind": "alert",
                    "event.category": ["malware"],
                    "event.type": ["info"],
                    "event.action": "incident",
                    "event.dataset": "crowdstrike.falcon_endpoint",
                    "agent.type": "falcon",
                },
                target: "",
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
                convertProcess(evt)
            })
            .Build(),

        UserActivityAuditEvent: new processor.Chain()
            .AddFields({
                fields: {
                    kind: "event",
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
                }, {
                    from: "crowdstrike.event.UserIp",
                    to: "related.ip",
                    type: "ip"
                }],
                mode: "copy",
                ignore_missing: true,
                fail_on_error: false
            })
            .Add(convertUsername)
            .Add(convertEventAction)
            .Build(),

        AuthActivityAuditEvent: new processor.Chain()
            .AddFields({
                fields: {
                    kind: "event",
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
                }, {
                    from: "crowdstrike.event.UserIp",
                    to: "related.ip",
                    type: "ip"
                }],
                mode: "copy",
                ignore_missing: true,
                fail_on_error: false
            })
            .Add(function (evt) {
                evt.Put("event.action", convertUnderscore(evt.Get("crowdstrike.event.OperationName")))
                convertUsername(evt)
            })
            .Build(),

        FirewallMatchEvent: new processor.Chain()
            .AddFields({
                fields: {
                    kind: "event",
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
                convertEventAction(evt)
                convertProcess(evt)
                convertSourceDestination(evt)
            })
            .Build(),

        RemoteResponseSessionStartEvent: new processor.Chain()
            .AddFields({
                fields: {
                    "event.kind": "event",
                    "event.type": ["start"],
                    "event.dataset": "crowdstrike.falcon_audit",
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
            .Add(convertUsername)
            .Add(convertEventAction)
            .Build(),

        RemoteResponseSessionEndEvent: new processor.Chain()
            .AddFields({
                fields: {
                    "event.kind": "event",
                    "event.type": ["end"],
                    "event.dataset": "crowdstrike.falcon_audit",
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
            .Add(convertUsername)
            .Add(convertEventAction)
            .Build(),
    }

    // main processor
    return new processor.Chain()
        .DecodeJSONFields({
            fields: ["message"],
            target: "crowdstrike",
            process_array: true,
            max_depth: 8
        })
        .Add(function (evt) {
            evt.Delete("message");
            evt.Delete("host.name");

            convertToMSEpoch(evt, "crowdstrike.event.ProcessStartTime")
            convertToMSEpoch(evt, "crowdstrike.event.ProcessEndTime")
            convertToMSEpoch(evt, "crowdstrike.event.IncidentStartTime")
            convertToMSEpoch(evt, "crowdstrike.event.IncidentEndTime")
            convertToMSEpoch(evt, "crowdstrike.event.StartTimestamp")
            convertToMSEpoch(evt, "crowdstrike.event.EndTimestamp")
            convertToMSEpoch(evt, "crowdstrike.event.UTCTimestamp")
            convertToMSEpoch(evt, "crowdstrike.metadata.eventCreationTime")

            var outcome = evt.Get("crowdstrike.event.Success")
            if (outcome === true) {
                evt.Put("event.outcome", "success")
            } else if (outcome === false) {
                evt.Put("event.outcome", "failure")
            } else {
                evt.Put("event.outcome", "unknown")
            }

            var eventProcessor = eventProcessors[evt.Get("crowdstrike.metadata.eventType")]
            if (eventProcessor) {
                eventProcessor.Run(evt)
            }
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
        .Build()
        .Run
})();

function process(evt) {
    crowdstrikeFalconProcessor(evt);
}
