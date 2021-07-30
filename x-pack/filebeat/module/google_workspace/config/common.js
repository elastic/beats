// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var googleWorkspace = (function () {
    var processor = require("processor");

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    var parseTimestamp = new processor.Timestamp({
        field: "json.id.time",
        timezone: "UTC",
        layouts: ["2006-01-02T15:04:05.999Z"],
        tests: ["2020-02-05T18:19:23.599Z"],
        ignore_missing: true,
    });

    var addID = new processor.Fingerprint({
        fields: [
            "json.id.time",
            "json.id.uniqueQualifier",
            "json.id.applicationName",
            "json.id.customerId",
        ],
        target_field: "@metadata.id",
        ignore_missing: true,
        fail_on_error: false,
    });

    var convertFields = new processor.Convert({
        fields: [
            { from: "message", to: "event.original" },
            { from: "json.events.name", to: "event.action" },
            { from: "json.id.applicationName", to: "event.provider" },
            { from: "json.id.uniqueQualifier", to: "event.id", type: "string" },
            { from: "json.actor.email", to: "source.user.email" },
            { from: "json.actor.profileId", to: "source.user.id", type: "string" },
            { from: "json.ipAddress", to: "source.ip", type: "ip" },
            { from: "json.kind", to: "google_workspace.kind" },
            { from: "json.id.customerId", to: "organization.id", type: "string" },
            { from: "json.actor.callerType", to: "google_workspace.actor.type" },
            { from: "json.actor.key", to: "google_workspace.actor.key" },
            { from: "json.ownerDomain", to: "google_workspace.organization.domain" },
            { from: "json.events.type", to: "google_workspace.event.type" },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    var completeUserData = function(evt) {
        var email = evt.Get("source.user.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.Put("user.id", evt.Get("source.user.id"));
        evt.Put("user.name", data[0]);
        evt.Put("source.user.name", data[0]);
        evt.Put("user.domain", data[1]);
        evt.Put("source.user.domain", data[1]);
    };

    var copyFields = function(evt) {
        var ip = evt.Get("source.ip");
        if (ip) {
            evt.Put("related.ip", [ip]);
        }
        var userName = evt.Get("source.user.name");
        if (userName) {
            evt.Put("related.user", [userName]);
        }
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(addID)
        .Add(convertFields)
        .Add(completeUserData)
        .Add(copyFields)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return googleWorkspace.process(evt);
}
