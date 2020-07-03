// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var gsuite = (function () {
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

    var convertFields = new processor.Convert({
        fields: [
            { from: "message", to: "event.original" },
            { from: "json.events.name", to: "event.action" },
            { from: "json.id.applicationName", to: "event.provider" },
            { from: "json.id.uniqueQualifier", to: "event.id", type: "string" },
            { from: "json.actor.email", to: "client.user.email" },
            { from: "json.actor.profileId", to: "client.user.id", type: "string" },
            { from: "json.ipAddress", to: "client.ip", type: "ip" },
            { from: "json.kind", to: "gsuite.kind" },
            { from: "json.id.customerId", to: "organization.id", type: "string" },
            { from: "json.actor.callerType", to: "gsuite.actor.type" },
            { from: "json.actor.key", to: "gsuite.actor.key" },
            { from: "json.ownerDomain", to: "gsuite.organization.domain" },
            { from: "json.events.type", to: "gsuite.event.type" },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    var completeUserData = function(evt) {
        var email = evt.Get("client.user.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.Put("client.user.name", data[0]);
        evt.Put("client.user.domain", data[1]);
    };

    var copyFields = function(evt) {
        var ip = evt.Get("client.ip");
        if (ip) {
            evt.Put("related.ip", [ip]);
        }
        var userName = evt.Get("client.user.name");
        if (userName) {
            evt.Put("related.user", [userName]);
        }
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(convertFields)
        .Add(completeUserData)
        .Add(copyFields)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return gsuite.process(evt);
}
