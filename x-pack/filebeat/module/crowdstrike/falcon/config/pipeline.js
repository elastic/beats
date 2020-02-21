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
            { from: "crowdstrike.event.LocalIP", to: "source.ip" },
            { from: "crowdstrike.event.ProcessId", to: "process.pid" },

            // UserActivityAuditEvent and AuthActivityAuditEvent
            { from: "crowdstrike.event.UserIp", to: "source.ip" },
        ],
        mode: "copy",
        ignore_missing: true,
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

        switch (eventType) {
            case "DetectionSummaryEvent":
                var tactic = evt.Get("crowdstrike.event.Tactic").toLowerCase()
                var technique = evt.Get("crowdstrike.event.Technique").toLowerCase()
                  
                evt.Put("event.action", tactic + "_" + technique)
                evt.Put("event.kind", "alert")
                evt.Put("event.category", "malware")
                evt.Put("message", evt.Get("crowdstrike.event.DetectDescription"))
                evt.Put("process.name", evt.Get("crowdstrike.event.FileName"))
                evt.Put("process.executable", evt.Get("crowdstrike.event.CommandLine"))
                evt.Put("user.name", evt.Get("crowdstrike.event.UserName"))
                evt.Put("user.domain", evt.Get("crowdstrike.event.MachineDomain"))
                evt.Put("rule.reference", evt.Get("crowdstrike.event.FalconHostLink"))
                evt.Put("agent.id", evt.Get("crowdstrike.event.SensorId"))
                evt.Put("host.name", evt.Get("crowdstrike.event.ComputerName"))

                break;

            case "UserActivityAuditEvent":

                evt.Put("user.name", evt.Get("crowdstrike.event.UserId"))
                evt.Put("message", evt.Get("crowdstrike.event.OperationName"))
                evt.Put("event.action", convertUnderscore(eventType))
                evt.Put("event.kind", "event")
                evt.Put("host.name", "")

                break;

            case "AuthActivityAuditEvent":

                evt.Put("user.name", evt.Get("crowdstrike.event.UserId"))
                evt.Put("message", evt.Get("crowdstrike.event.ServiceName"))
                evt.Put("event.action", convertUnderscore(evt.Get("crowdstrike.event.OperationName")))
                evt.Put("host.name", "")
 
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
