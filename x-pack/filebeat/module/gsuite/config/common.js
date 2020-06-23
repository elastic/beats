// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function GSuite(keep_original_message) {
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
            { from: "json.id.uniqueQualifier", to: "event.id" },
            { from: "json.actor.email", to: "client.user.email" },
            { from: "json.actor.profileId", to: "client.user.id" },
            { from: "json.ipAddress", to: "client.ip" },
            { from: "json.kind", to: "gsuite.kind" },
            { from: "json.actor.callerType", to: "gsuite.actor.type" },
            { from: "json.actor.key", to: "gsuite.actor.key" },
            { from: "json.ownerDomain", to: "gsuite.owner.domain" },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    var copyFields = new processor.Convert({
        fields: [
            { from: "client.ip", to: "related.ip" },
        ],
        ignore_missing: true,
        fail_on_error: false,
    });


    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(convertFields)
        .Add(copyFields)
        .Delete("json")
        .Build();

    return {
        process: pipeline.Run,
    };
}

var gsuite;

// Register params from configuration.
function register(params) {
    gsuite = new GSuite();
}

function process(evt) {
    return gsuite.process(evt);
}
