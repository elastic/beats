// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var crowdstrikeFalcon = (function() {
    var processor = require("processor");

    var convertUnderscore = function(text) {
        return text.split(/(?=[A-Z])/).join('_').toLowerCase(); 
    };

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "crowdstrike",
        process_array: true,
        max_depth: 8
    });

    var dropFields = function(evt) {
        evt.Delete("message");
        evt.Delete("host.name");
    };
  
    var setFields = function (evt) {
        evt.Put("agent.name", "falcon");
    };

    var convertFields = new processor.Convert({
        fields: [
            // DetectionSummaryEvent
            { from: "crowdstrike.event.LocalIP", to: "source.ip", type: "ip" },
            { from: "crowdstrike.event.ProcessId", to: "process.pid" },
            // UserActivityAuditEvent and AuthActivityAuditEvent
            { from: "crowdstrike.event.UserIp", to: "source.ip", type: "ip" },
        ],
        mode: "copy",
        ignore_missing: true,
        ignore_failure: true
    });

    var parseTimestamp = new processor.Timestamp({
        field: "crowdstrike.metadata.eventCreationTime",
        target_field: "@timestamp",
        timezone: "UTC",
        layouts: ["UNIX_MS"],
        ignore_missing: false,
    });

    var processEvent = function(evt) {
        var eventType = evt.Get("crowdstrike.metadata.eventType")
        var outcome = evt.Get("crowdstrike.event.Success")

        evt.Put("event.kind", "event")

        if (outcome === true) {
            evt.Put("event.outcome", "success")
        }
        else if (outcome === false) {
            evt.Put("event.outcome", "failure")
        }
        else {
            evt.Put("event.outcome", "unknown")
        }

        switch (eventType) {
            case "DetectionSummaryEvent":
                var tactic = evt.Get("crowdstrike.event.Tactic").toLowerCase()
                var technique = evt.Get("crowdstrike.event.Technique").toLowerCase()
                evt.Put("threat.technique.name", technique) 
                evt.Put("threat.tactic.name", tactic)

                evt.Put("event.action", evt.Get("crowdstrike.event.PatternDispositionDescription"))
                evt.Put("event.kind", "alert")
                evt.Put("event.type", ["info"])
                evt.Put("event.category", ["malware"])
                evt.Put("event.url", evt.Get("crowdstrike.event.FalconHostLink"))
                evt.Put("event.dataset", "crowdstrike.falcon_endpoint")

                evt.Put("event.severity", evt.Get("crowdstrike.event.Severity"))
                evt.Put("message", evt.Get("crowdstrike.event.DetectDescription"))
                evt.Put("process.name", evt.Get("crowdstrike.event.FileName"))

                var command_line = evt.Get("crowdstrike.event.CommandLine")
                var args = command_line.split(' ')
                var executable = args[0]

                evt.Put("process.command_line", command_line)
                evt.Put("process.args", args)
                evt.Put("process.executable", executable)

                evt.Put("user.name", evt.Get("crowdstrike.event.UserName"))
                evt.Put("user.domain", evt.Get("crowdstrike.event.MachineDomain"))
                evt.Put("agent.id", evt.Get("crowdstrike.event.SensorId"))
                evt.Put("host.name", evt.Get("crowdstrike.event.ComputerName"))
                evt.Put("agent.type", "falcon")
                evt.Put("file.hash.sha256", evt.Get("crowdstrike.event.SHA256String"))
                evt.Put("file.hash.md5", evt.Get("crowdstrike.event.MD5String"))
                evt.Put("rule.name", evt.Get("crowdstrike.event.DetectName"))
                evt.Put("rule.description", evt.Get("crowdstrike.event.DetectDescription"))

                break;

            case "IncidentSummaryEvent":
                evt.Put("event.kind", "alert")
                evt.Put("event.type", ["info"])
                evt.Put("event.category", ["malware"])
                evt.Put("event.action", "incident")
                evt.Put("event.url", evt.Get("crowdstrike.event.FalconHostLink"))
                evt.Put("event.dataset", "crowdstrike.falcon_endpoint")

                evt.Put("message", "Incident score " + evt.Get("crowdstrike.event.FineScore"))

                break;

            case "UserActivityAuditEvent":
                var userid = evt.Get("crowdstrike.event.UserId")
                evt.Put("user.name", userid)
                if (userid.split('@').length == 2) {
                    evt.Put("user.email", userid)
                }

                evt.Put("message", evt.Get("crowdstrike.event.OperationName"))
                evt.Put("event.action", convertUnderscore(eventType))
                evt.Put("event.type", ["change"])
                evt.Put("event.category", ["iam"])
                evt.Put("event.dataset", "crowdstrike.falcon_audit")

                break;

            case "AuthActivityAuditEvent":
                var userid = evt.Get("crowdstrike.event.UserId")
                evt.Put("user.name", userid)
                if (userid.split('@').length == 2) {
                    evt.Put("user.email", userid)
                }

                evt.Put("message", evt.Get("crowdstrike.event.ServiceName"))
                evt.Put("event.action", convertUnderscore(evt.Get("crowdstrike.event.OperationName")))
                evt.Put("event.type", ["change"])
                evt.Put("event.category", ["authentication"])
                evt.Put("event.dataset", "crowdstrike.falcon_audit")

                break;

            case "RemoteResponseSessionStartEvent":
            case "RemoteResponseSessionEndEvent":
                var username = evt.Get("crowdstrike.event.UserName")
                evt.Put("user.name", username)
                if (username.split('@').length == 2) {
                    evt.Put("user.email", username)
                }

                evt.Put("host.name", evt.Get("crowdstrike.event.HostnameField"))
                evt.Put("event.action", convertUnderscore(eventType))
                evt.Put("event.dataset", "crowdstrike.falcon_audit")

                if (eventType == "RemoteResponseSessionStartEvent") {
                    evt.Put("event.type", ["start"])
                    evt.Put("message", "Remote response session started")
                } else {
                    evt.Put("event.type", ["end"])
                    evt.Put("message", "Remote response session ended")
                }

                break;

            default:
                break;
        }
    } 

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(dropFields)
        .Add(convertFields)
        .Add(processEvent)
        .Build();

    return {
        process: pipeline.Run,
    };
})();

function process(evt) {
    crowdstrikeFalcon.process(evt);
}
