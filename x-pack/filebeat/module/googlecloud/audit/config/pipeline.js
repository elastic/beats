// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function Audit(keep_original_message) {
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
    // https://cloud.google.com/logging/docs/reference/v2/rest/v2/MonitoredResource
    var setCloudMetadata = new processor.Convert({
        fields: [
            {
                from: "json.resource.labels.project_id",
                to: "cloud.project.id",
                type: "string"
            },
            {
                from: "json.resource.labels.instance_id",
                to: "cloud.instance.id",
                type: "string"
            }
        ],
        ignore_missing: true,
        fail_on_error: false,
    });

    // The log includes a protoPayload field.
    // https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
    var convertLogEntry = new processor.Convert({
        fields: [
            {from: "json.protoPayload", to: "json"},
        ],
        mode: "rename",
    });

    // The LogEntry's protoPayload is moved to the json field. The protoPayload
    // contains the structured audit log fields.
    // https://cloud.google.com/logging/docs/reference/audit/auditlog/rest/Shared.Types/AuditLog
    var convertProtoPayload = new processor.Convert({
        fields: [
            {
                from: "json.@type",
                to: "googlecloud.audit.type",
                type: "string"
            },
            {
                from: "json.authenticationInfo.principalEmail",
                to: "googlecloud.audit.authentication_info.principal_email",
                type: "string"
            },
            {
                from: "json.authenticationInfo.authoritySelector",
                to: "googlecloud.audit.authentication_info.authority_selector",
                type: "string"
            },
            {
                from: "json.authorizationInfo",
                to: "googlecloud.audit.authorization_info"
                // Type is an array of objects.
            },
            {
                from: "json.methodName",
                to: "googlecloud.audit.method_name",
                type: "string",
            },
            {
                from: "json.numResponseItems",
                to: "googlecloud.audit.num_response_items",
                type: "long"
            },
            {
                from: "json.request.@type",
                to: "googlecloud.audit.request.proto_name",
                type: "string"
            },
            // The values in the request object will depend on the proto type.
            // So be very careful about making any assumptions about data shape.
            {
                from: "json.request.filter",
                to: "googlecloud.audit.request.filter",
                type: "string"
            },
            {
                from: "json.request.name",
                to: "googlecloud.audit.request.name",
                type: "string"
            },
            {
                from: "json.request.resourceName",
                to: "googlecloud.audit.request.resource_name",
                type: "string"
            },
            {
                from: "json.requestMetadata.callerIp",
                to: "googlecloud.audit.request_metadata.caller_ip",
                type: "ip"
            },
            {
                from: "json.requestMetadata.callerSuppliedUserAgent",
                to: "googlecloud.audit.request_metadata.caller_supplied_user_agent",
                type: "string",
            },
            {
                from: "json.response.@type",
                to: "googlecloud.audit.response.proto_name",
                type: "string"
            },
            // The values in the response object will depend on the proto type.
            // So be very careful about making any assumptions about data shape.
            {
                from: "json.response.status",
                to: "googlecloud.audit.response.status",
                type: "string"
            },
            {
                from: "json.response.details.group",
                to: "googlecloud.audit.response.details.group",
                type: "string"
            },
            {
                from: "json.response.details.kind",
                to: "googlecloud.audit.response.details.kind",
                type: "string"
            },
            {
                from: "json.response.details.name",
                to: "googlecloud.audit.response.details.name",
                type: "string"
            },
            {
                from: "json.response.details.uid",
                to: "googlecloud.audit.response.details.uid",
                type: "string",
            },
            {
                from: "json.resourceName",
                to: "googlecloud.audit.resource_name",
                type: "string",
            },
            {
                from: "json.resourceLocation.currentLocations",
                to: "googlecloud.audit.resource_location.current_locations"
                // Type is a string array.
            },
            {
                from: "json.serviceName",
                to: "googlecloud.audit.service_name",
                type: "string",
            },
            {
                from: "json.status.code",
                to: "googlecloud.audit.status.code",
                type: "integer",
            },
            {
                from: "json.status.message",
                to: "googlecloud.audit.status.message",
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
                from: "googlecloud.audit.request_metadata.caller_ip",
                to: "source.ip",
                type: "ip"
            },
            {
                from: "googlecloud.audit.authentication_info.principal_email",
                to: "user.email",
                type: "string"
            },
            {
                from: "googlecloud.audit.service_name",
                to: "service.name",
                type: "string"
            },
            {
                from: "googlecloud.audit.request_metadata.caller_supplied_user_agent",
                to: "user_agent.original",
                type: "string"
            },
            {
                from: "googlecloud.audit.method_name",
                to: "event.action",
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

    // Rename nested fields.
    var renameNestedFields = function(evt) {
        var arr = evt.Get("googlecloud.audit.authorization_info");
        if (Array.isArray(arr)) {
            for (var i = 0; i < arr.length; i++) {
                if (arr[i].resourceAttributes) {
                    // Convert to snake_case.
                    arr[i].resource_attributes = arr[i].resourceAttributes;
                    delete arr[i].resourceAttributes;
                }
            }
        }
    };

    // Set ECS categorization fields.
    var setECSCategorization = function(evt) {
        evt.Put("event.kind", "event");

        // google.rpc.Code value for OK is 0.
        if (evt.Get("googlecloud.audit.status.code") === 0) {
            evt.Put("event.outcome", "success");
            return;
        }

        // Try to use authorization_info.granted when there was no status code.
        if (evt.Get("googlecloud.audit.status.code") == null) {
            var authorization_info = evt.Get("googlecloud.audit.authorization_info");
            if (Array.isArray(authorization_info) && authorization_info.length === 1) {
                if (authorization_info[0].granted === true) {
                    evt.Put("event.outcome", "success");
                } else if (authorization_info[0].granted === false) {
                    evt.Put("event.outcome", "failure");
                }
                return
            }

            evt.Put("event.outcome", "unknown");
            return;
        }

        evt.Put("event.outcome", "failure");
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(saveOriginalMessage)
        .Add(dropPubSubFields)
        .Add(saveMetadata)
        .Add(setCloudMetadata)
        .Add(convertLogEntry)
        .Add(convertProtoPayload)
        .Add(copyFields)
        .Add(dropExtraFields)
        .Add(renameNestedFields)
        .Add(setECSCategorization)
        .Build();

    return {
        process: pipeline.Run,
    };
}

var audit;

// Register params from configuration.
function register(params) {
    audit = new Audit(params.keep_original_message);
}

function process(evt) {
    return audit.process(evt);
}
