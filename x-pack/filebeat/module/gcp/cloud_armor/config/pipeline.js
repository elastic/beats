// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function CloudArmor(keep_original_message) {
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
            },
            {
                from: "json.resource.labels.backend_service_name",
                to: "cloud.backend.name",
                type: "string"
            }
        ],
        ignore_missing: true,
        fail_on_error: false,
    });

    // Keep httpRequest metadata in message
    var convertHttpRequest = new processor.Convert({
        fields: [
            {
                from: "json.httpRequest.requestMethod",
                to: "gcp.cloud_armor.http_request.method",
                type: "string"
            },
            {
                from: "json.httpRequest.requestUrl",
                to: "gcp.cloud_armor.http_request.url",
                type: "string"
            },
            {
                from: "json.httpRequest.requestSize",
                to: "gcp.cloud_armor.http_request.request_size",
                type: "string"
            },
            {
                from: "json.httpRequest.status",
                to: "gcp.cloud_armor.http_request.status_code",
                type: "integer"
            },
            {
                from: "json.httpRequest.responseSize",
                to: "gcp.cloud_armor.http_request.response_size",
                type: "string"
            },
            {
                from: "json.httpRequest.userAgent",
                to: "gcp.cloud_armor.http_request.user_agent",
                type: "string"
            },
            {
                from: "json.httpRequest.remoteIp",
                to: "gcp.cloud_armor.http_request.remote_ip",
                type: "ip"
            },
            {
                from: "json.httpRequest.serverIp",
                to: "gcp.cloud_armor.http_request.server_ip",
                type: "ip"
            },
            {
                from: "json.httpRequest.referer",
                to: "gcp.cloud_armor.http_request.referer",
                type: "string"
            },
        ],
        mode: "rename",
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
                to: "gcp.cloud_armor.type",
                type: "string"
            },
            // enforcedSecurityPolicy
            {
                from: "json.enforcedSecurityPolicy.configuredAction",
                to: "gcp.cloud_armor.enforced_security_policy.action",
                type: "string"
            },
            {
                from: "json.enforcedSecurityPolicy.outcome",
                to: "gcp.cloud_armor.enforced_security_policy.outcome",
                type: "string"
            },
            {
                from: "json.enforcedSecurityPolicy.preconfiguredExprIds",
                to: "gcp.cloud_armor.enforced_security_policy.signature_ids",
                // Type is a string array.
            },
            {
                from: "json.enforcedSecurityPolicy.priority",
                to: "gcp.cloud_armor.enforced_security_policy.priority",
                type: "integer"
            },
            {
                from: "json.enforcedSecurityPolicy.name",
                to: "gcp.cloud_armor.enforced_security_policy.name",
                type: "string"
            },
            // previewSecurityPolicy
            {
                from: "json.previewSecurityPolicy.configuredAction",
                to: "gcp.cloud_armor.preview_security_policy.action",
                type: "string"
            },
            {
                from: "json.previewSecurityPolicy.outcome",
                to: "gcp.cloud_armor.preview_security_policy.outcome",
                type: "string"
            },
            {
                from: "json.previewSecurityPolicy.preconfiguredExprIds",
                to: "gcp.cloud_armor.preview_security_policy.signature_ids",
                // Type is a string array.
            },
            {
                from: "json.previewSecurityPolicy.priority",
                to: "gcp.cloud_armor.preview_security_policy.priority",
                type: "integer"
            },
            {
                from: "json.previewSecurityPolicy.name",
                to: "gcp.cloud_armor.preview_security_policy.name",
                type: "string"
            },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    // Copy some fields
    var copyFields = new processor.Convert({
        fields: [
            {
                from: "gcp.cloud_armor.http_request.remote_ip",
                to: "source.ip",
                type: "ip"
            },
            {
                from: "cloud.backend.name",
                to: "service.name",
                type: "string"
            },
            {
                from: "gcp.cloud_armor.http_request.user_agent",
                to: "user_agent.original",
                type: "string"
            },
        ],
        ignore_missing: true,
        fail_on_error: false,
    });

    // Drop extra fields
    var dropExtraFields = function(evt) {
        evt.Delete("json");
    };

    // Convert some misc values into a slightly nicer format for analysis
    var convertValues = function(evt) {
        // 1. Convert backend name to remove identifier
        var name = evt.Get("cloud.backend.name");
        if (name !== null) {
            var arr = name.split("-");
            evt.Put("cloud.backend.name", arr.slice(2,arr.length-2).join("-"));
        }

        // 2. Aggregate signature_ids in common field
        var preview_ids = evt.Get("gcp.cloud_armor.preview_security_policy.signature_ids");
        var enforced_ids = evt.Get("gcp.cloud_armor.enforced_security_policy.signature_ids");

        if (preview_ids !== null && preview_ids.length > 0) {
            evt.Put("gcp.cloud_armor.signature_ids", preview_ids);
        }
        if (enforced_ids !== null && enforced_ids.length > 0) {
            evt.Put("gcp.cloud_armor.signature_ids", enforced_ids);
        }

        // 3. Convert signature_ids into common threat type
        var ids = evt.Get("gcp.cloud_armor.signature_ids");
        if (ids !== null && ids.length > 0) {
            var threats = [];
            ids.forEach(function (item) {
                var arr = item.split("-");
                var rule = arr[arr.length -1];
                switch (rule) {
                    case "sqli":
                        threats.push("SQLi");
                        break;
                    case "xss":
                        threats.push("XSS");
                        break;
                    case "lfi":
                        threats.push("LFI");
                        break;
                    case "rfi":
                        threats.push("RFI");
                        break;
                    case "rce":
                        threats.push("RCE");
                        break;
                    case "scannerdetection":
                        threats.push("Scanner detection")
                        break;
                    case "cve":
                        if (item === "owasp-crs-v030001-id044228-cve") {
                            threats.push("Log4shell")
                        } else {
                            threats.push("CVE")
                        }
                        break;
                }
            });
            evt.Put("gcp.cloud_armor.threats", threats);
        }
    }

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(saveOriginalMessage)
        .Add(dropPubSubFields)
        .Add(saveMetadata)
        .Add(setCloudMetadata)
        .Add(convertHttpRequest)
        .Add(convertLogEntry)
        .Add(convertJsonPayload)
        .Add(copyFields)
        .Add(convertValues)
        .Add(dropExtraFields)
        .Build();

    return {
        process: pipeline.Run,
    };
}

var pipeline;

// Register params from configuration.
function register(params) {
    pipeline = new CloudArmor(params.keep_original_message);
}

function process(evt) {
    return pipeline.process(evt);
}
