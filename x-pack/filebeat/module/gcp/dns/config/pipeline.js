// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function DNS(keep_original_message) {
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
                {from: "message", to: "event.original"},
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
                from: "json.resource.labels.location",
                to: "cloud.region",
                type: "string"
            },
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

    // The jsonPayload contains the structured dns log fields.
    // https://cloud.google.com/dns/docs/monitoring#dns-log-record-format
    var convertJsonPayload = new processor.Convert({
        fields: [
            {
                from: "json.authAnswer",
                to: "gcp.dns.auth_answer",
                type: "boolean"
            },
            {
                from: "json.destinationIP",
                to: "gcp.dns.destination_ip",
                type: "ip"
            },
            {
                from: "json.egressError",
                to: "gcp.dns.egress_error",
                type: "ip"
            },
            {
                from: "json.protocol",
                to: "gcp.dns.protocol",
                type: "string"
            },
            {
                from: "json.queryName",
                to: "gcp.dns.query_name",
                type: "string"
            },
            {
                from: "json.queryType",
                to: "gcp.dns.query_type",
                type: "string"
            },
            {
                from: "json.rdata",
                to: "gcp.dns.rdata",
                type: "string"
            },
            {
                from: "json.responseCode",
                to: "gcp.dns.response_code",
                type: "string"
            },
            {
                from: "json.serverLatency",
                to: "gcp.dns.server_latency",
                type: "integer"
            },
            {
                from: "json.sourceIP",
                to: "gcp.dns.source_ip",
                type: "ip"
            },
            {
                from: "json.sourceNetwork",
                to: "gcp.dns.source_network",
                type: "string"
            },
            {
                from: "json.vmInstanceIdString",
                to: "gcp.dns.vm_instance_id",
                type: "string"
            },
            {
                from: "json.vmInstanceName",
                to: "gcp.dns.vm_instance_name",
                type: "string"
            },
            {
                from: "json.vmProjectId",
                to: "gcp.dns.vm_project_id",
                type: "string"
            },
            {
                from: "json.vmZoneName",
                to: "gcp.dns.vm_zone_name",
                type: "string"
            },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    // Copy some fields.
    var copyFields = new processor.Convert({
        fields: [
            {from: "gcp.dns.destination_ip", to: "destination.address"},
            {from: "gcp.dns.destination_ip", to: "destination.ip"},
            {from: "gcp.dns.protocol", to: "network.transport"},
            {from: "gcp.dns.query_type", to: "dns.question.type"},
            {from: "gcp.dns.response_code", to: "dns.response_code"},
            {from: "gcp.dns.source_ip", to: "source.address"},
            {from: "gcp.dns.source_ip", to: "source.ip"},
            {from: "gcp.dns.vm_instance_id", to: "cloud.instance.id"},
            {from: "gcp.dns.vm_zone_name", to: "cloud.availability_zone"},
        ],
        ignore_missing: true,
        fail_on_error: false,
    });

    // Drop extra fields.
    var dropExtraFields = function(evt) {
        evt.Delete("json");
    };

    // Convert query name.
    var convertQueryName = function(evt) {
        var query_name = evt.Get("gcp.dns.query_name");

        // Remove trailing fullstop.
        query_name = query_name.replace(/[.]$/, "")

        evt.Put("dns.question.name", query_name);
    }

    // Convert VM instance name.
    var convertVMInstanceName = function(evt) {
        var vm_instance_name = evt.Get("gcp.dns.vm_instance_name");

        // Remove preceding project.
        vm_instance_name = vm_instance_name.replace(/^.*[.]/, "")

        evt.Put("cloud.instance.name", vm_instance_name);
    }

    // The RData contains the DNS answer, truncated to 260 bytes.
    var convertRData = function(evt) {
        var rdata = evt.Get("gcp.dns.rdata");

        var dns_answers = [];
        var dns_resolved_ip = [];

        // Remove truncated answers.
        rdata = rdata.replace(/\n.*[.]{3}$/, "");

        // Process answers.
        rdata.split("\n").forEach(function(answer) {
            var answer_parts = answer.split("\t");

            // Assign answer parts.
            var name = answer_parts[0];
            var ttl = answer_parts[1];
            var cls = answer_parts[2];
            var type = answer_parts[3];
            var data = answer_parts[4];

            // Remove trailing fullstop.
            name = name.replace(/[.]$/, "")
            data = data.replace(/[.]$/, "")

            // Uppercase type.
            type = type.toUpperCase();

            dns_answers.push({
                "name": name,
                "ttl": ttl,
                "class": cls,
                "type": type,
                "data": data
            });

            if (type == "A" || type == "AAAA") {
                dns_resolved_ip.push(data);
            }
        });

        evt.Put("dns.answers", dns_answers);
        evt.Put("dns.resolved_ip", dns_resolved_ip);
    };

    // Set ECS categorization fields.
    var setECSCategorization = function(evt) {
        evt.Put("event.kind", "event");

        if (evt.Get("gcp.dns.response_code") == "NOERROR") {
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
        .Add(convertJsonPayload)
        .Add(copyFields)
        .Add(dropExtraFields)
        .Add(convertQueryName)
        .Add(convertVMInstanceName)
        .Add(convertRData)
        .Add(setECSCategorization)
        .Build();

    return {
        process: pipeline.Run,
    };
}

var dns;

// Register params from configuration.
function register(params) {
    dns = new DNS(params.keep_original_message);
}

function process(evt) {
    return dns.process(evt);
}
