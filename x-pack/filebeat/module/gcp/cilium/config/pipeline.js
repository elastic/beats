// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function Cilium(keep_original_message) {
    var processor = require("processor");

    // The pub/sub input writes the Stackdriver LogEntry object into the message
    // field. The message needs decoded as JSON.
    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    // Set @timetamp the LogEntry's timestamp.
    var parseTimestamp = new processor.Timestamp({
        field: "json.timestamp",
        timezone: "UTC",
        layouts: ["2006-01-02T15:04:05.999999999Z07:00"],
        tests: ["2019-06-14T03:50:10.845445834Z"],
        ignore_missing: true,
    });

    var saveOriginalMessage = function(evt) {};
    if (keep_original_message) {
        saveOriginalMessage = new processor.Convert({
            fields: [
                {from: "message", to: "event.original"}
            ],
            mode: "rename"
        });
    }

    var dropPubSubFields = function(evt) {
        evt.Delete("message");
    };

    var saveMetadata = new processor.Convert({
        fields: [
            {from: "json.logName", to: "log.logger"},
            {from: "json.insertId", to: "event.id"},
        ],
        ignore_missing: true
    });

    // Use the monitored resource type's labels to set the cloud metadata.
    // The labels can vary based on the resource.type.
    var setCloudMetadata = new processor.Convert({
        fields: [
            {
                from: "json.resource.labels.project_id",
                to: "cloud.project.id",
                type: "string"
            }
        ],
        ignore_missing: true,
        fail_on_error: false,
    });

    // The log includes a jsonPayload field.
    // https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
    var convertLogEntry = new processor.Convert({
        fields: [
            {from: "json.jsonPayload", to: "json"},
        ],
        mode: "rename",
    });

    var convertJsonPayload = new processor.Convert({
        fields: [
            {
                from: "json.@type",
                to: "gcp.cilium.type",
                type: "string"
            },
            {
                from: "json.flow.IP.destination",
                to: "gcp.cilium.destination.ip",
                type: "ip"
            },
            {
                from: "json.flow.IP.source",
                to: "gcp.cilium.source.ip",
                type: "ip"
            },
            {
                from: "json.flow.destination_names",
                to: "gcp.cilium.destination.hosts",
                // Type is a string array
            },
            {
                from: "json.flow.source.namespace",
                to: "gcp.cilium.source.namespace",
                type: "string"
            },
            {
                from: "json.flow.source.pod_name",
                to: "gcp.cilium.source.pod",
                type: "string"
            },
            {
                from: "json.flow.traffic_direction",
                to: "gcp.cilium.direction",
                type: "string"
            },
            {
                from: "json.flow.verdict",
                to: "gcp.cilium.verdict",
                type: "string"
            },
            {
                from: "json.resource.labels.cluster_name",
                to: "cloud.cluster.name",
                type: "string"
            }
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    // Drop extra fields
    var dropExtraFields = function(evt) {
        evt.Delete("json");
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(saveOriginalMessage)
        .Add(dropPubSubFields)
        .Add(saveMetadata)
        .Add(setCloudMetadata)
        .Add(convertLogEntry)
        .Add(convertJsonPayload)
        .Add(dropExtraFields)
        .Build();

    return {
        process: pipeline.Run,
    };
}

var pipeline;

// Register params from configuration.
function register(params) {
    pipeline = new Cilium(params.keep_original_message);
}

function process(evt) {
    return pipeline.process(evt);
}
