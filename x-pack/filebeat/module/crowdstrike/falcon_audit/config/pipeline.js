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

    // Don't process non-audit related events handled by the endpoint fileset
    // This avoids generating duplicate events since both event types come from 
    // the same log file. Abort early to minimize CPU time.
    var cancelNonAuditEvents = function(evt) {
        var eventType = evt.Get("crowdstrike.metadata.eventType")
        switch (eventType) {
            case "DetectionSummaryEvent":
            case "IncidentSummaryEvent":
                evt.Cancel()
                break;
        }  
    };

    var dropFields = function(evt) {
        evt.Delete("message");
    };
  
    var setFields = function (evt) {
        evt.Put("agent.name", "falcon");
    };

    var convertFields = new processor.Convert({
        fields: [
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
        .Add(cancelNonAuditEvents)
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
