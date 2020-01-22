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
        ],
        ignore_missing: true
    });

    var setCloudMetadata = new processor.Convert({
        fields: [
            {from: "json.resource.labels.project_id", to: "cloud.project.id"},
        ],
        ignore_missing: true
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
    var convertProtoPayload = new processor.Convert({
        fields: [
            {from: "json.@type", to: "json.type"},

            {from: "json.authenticationInfo.principalEmail", to: "json.authenticationInfo.principal_email"},
            {from: "json.authenticationInfo.authoritySelector", to: "json.authenticationInfo.authority_selector"},
            {from: "json.authenticationInfo", to: "json.authentication_info"},

            {from: "json.authorizationInfo", to: "json.authorization_info"},

            {from: "json.methodName", to: "json.method_name"},

            {from: "json.numResponseItems", to: "json.num_response_items", type: "long"},

            {from: "json.request.@type", to: "json.request.proto_name"},
            {from: "json.request.filter", to: "json.request.filter"},
            {from: "json.request.name", to: "json.request.name"},
            {from: "json.request.resourceName", to: "json.request.resource_name"},

            {from: "json.requestMetadata.callerIp", to: "json.requestMetadata.caller_ip", type: "ip"},
            {from: "json.requestMetadata.callerSuppliedUserAgent", to: "json.requestMetadata.caller_supplied_user_agent"},
            {from: "json.requestMetadata", to: "json.request_metadata"},

            {from: "json.resourceName", to: "json.resource_name"},

            {from: "json.resourceLocation.currentLocations", to: "json.resourceLocation.current_locations"},
            {from: "json.resourceLocation", to: "json.resource_location"},

            {from: "json.serviceName", to: "json.service_name"},

            {from: "json", to: "googlecloud.audit"},
        ],
        mode: "rename",
        ignore_missing: true,
    });

    // Copy some fields
    var copyFields = new processor.Convert({
        fields: [
            {from: "googlecloud.audit.request_metadata.caller_ip", to: "source.ip"},
            {from: "googlecloud.audit.authentication_info.principal_email", to: "user.email"},
            {from: "googlecloud.audit.service_name", to: "service.name"},
            {from: "googlecloud.audit.request_metadata.caller_supplied_user_agent", to: "user_agent.original"},
        ],
        fail_on_error: false,
    });

    // Drop extra fields
    var dropExtraFields = function(evt) {
        evt.Delete("json");
        evt.Delete("googlecloud.audit.request_metadata.requestAttributes");
        evt.Delete("googlecloud.audit.request_metadata.destinationAttributes");
    };

    // Rename nested fields 
    var RenameNestedFields = function(evt) {
        var arr = evt.Get("googlecloud.audit.authorization_info");
        for (var i = 0; i < arr.length; i++) {
          arr[i].resource_attributes = arr[i].resourceAttributes;
          delete arr[i].resourceAttributes;
        }
    };

    // Set event.outcome based on authenticatio_info and status 
    var setEventOutcome = function(evt) {
        if (evt.Get("googlecloud.audit.status.code") == null) {
            var authorization_info = evt.Get("googlecloud.audit.authorization_info");
            if (authorization_info.length == 1) {
                if (authorization_info[0].granted == null) {
                    evt.Put("event.outcome", "unknown");
                } else if (authorization_info[0].granted == true) {
                    evt.Put("event.outcome", "success");
                } else {
                    evt.Put("event.outcome", "failure");
                }
            } else {
                evt.Put("event.outcome", "unknown");
            } 
        } else if (evt.Get("googlecloud.audit.status.code") == 0) {
           evt.Put("event.outcome", "success");
        } else {
           evt.Put("event.outcome", "failure");
        }
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
        .Add(RenameNestedFields)
        .Add(setEventOutcome)
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
