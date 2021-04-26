// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function VPCFlow(keep_original_message, internalNetworks) {
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
        evt.Delete("labels");
    };

    var categorizeEvent = new processor.AddFields({
        target: "event",
        fields: {
            kind: "event",
            category: "network",
            type: "connection",
        },
    });


    var saveMetadata = new processor.Convert({
        fields: [
            {from: "json.logName", to: "log.logger"},
            {from: "json.insertId", to: "event.id"},
        ],
        ignore_missing: true
    });

    // Use the LogEntry object's timestamp. VPC flow logs are structured so the
    // LogEntry includes a jsonPayload field.
    // https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
    var convertLogEntry = new processor.Convert({
        fields: [
            {from: "json.jsonPayload", to: "json"},
        ],
        mode: "rename",
    });

    // The LogEntry's jsonPayload is moved to the json field. The jsonPayload
    // contains the structured VPC flow log fields.
    // https://cloud.google.com/vpc/docs/using-flow-logs#record_format
    var convertJsonPayload = new processor.Convert({
        fields: [
            {from: "json.connection.dest_ip", to: "destination.address"},
            {from: "json.connection.dest_port", to: "destination.port", type: "long"},
            {from: "json.connection.protocol", to: "network.iana_number", type: "string"},
            {from: "json.connection.src_ip", to: "source.address"},
            {from: "json.connection.src_port", to: "source.port", type: "long"},

            {from: "json.src_instance.vm_name", to: "source.domain"},
            {from: "json.dest_instance.vm_name", to: "destination.domain"},

            {from: "json.bytes_sent", to: "source.bytes", type: "long"},
            {from: "json.packets_sent", to: "source.packets", type: "long"},

            {from: "json.start_time", to: "event.start"},
            {from: "json.end_time", to: "event.end"},

            {from: "json.dest_location.asn", to: "destination.as.number", type: "long"},
            {from: "json.dest_location.continent", to: "destination.geo.continent_name"},
            {from: "json.dest_location.country", to: "destination.geo.country_name"},
            {from: "json.dest_location.region", to: "destination.geo.region_name"},
            {from: "json.dest_location.city", to: "destination.geo.city_name"},

            {from: "json.src_location.asn", to: "source.as.number", type: "long"},
            {from: "json.src_location.continent", to: "source.geo.continent_name"},
            {from: "json.src_location.country", to: "source.geo.country_name"},
            {from: "json.src_location.region", to: "source.geo.region_name"},
            {from: "json.src_location.city", to: "source.geo.city_name"},

            {from: "json.dest_instance", to: "gcp.destination.instance"},
            {from: "json.dest_vpc", to: "gcp.destination.vpc"},
            {from: "json.src_instance", to: "gcp.source.instance"},
            {from: "json.src_vpc", to: "gcp.source.vpc"},

            {from: "json.rtt_msec", to: "json.rtt.ms", type: "long"},
            {from: "json", to: "gcp.vpcflow"},
        ],
        mode: "rename",
        ignore_missing: true,
    });

    // Delete emtpy object's whose fields have been renamed leaving them childless.
    var dropEmptyObjects = function (evt) {
        evt.Delete("gcp.vpcflow.connection");
        evt.Delete("gcp.vpcflow.dest_location");
        evt.Delete("gcp.vpcflow.src_location");
    };

    // Copy the source/destination.address to source/destination.ip if they are
    // valid IP addresses.
    var copyAddressFields = new processor.Convert({
        fields: [
            {from: "source.address", to: "source.ip", type: "ip"},
            {from: "destination.address", to: "destination.ip", type: "ip"},
        ],
        fail_on_error: false,
    });

    var setCloudFromDestInstance = new processor.Convert({
        fields: [
            {from: "gcp.destination.instance.project_id", to: "cloud.project.id"},
            {from: "gcp.destination.instance.vm_name", to: "cloud.instance.name"},
            {from: "gcp.destination.instance.region", to: "cloud.region"},
            {from: "gcp.destination.instance.zone", to: "cloud.availability_zone"},
            {from: "gcp.destination.vpc.subnetwork_name", to: "network.name"},
        ],
        ignore_missing: true,
    });

    var setCloudFromSrcInstance = new processor.Convert({
        fields: [
            {from: "gcp.source.instance.project_id", to: "cloud.project.id"},
            {from: "gcp.source.instance.vm_name", to: "cloud.instance.name"},
            {from: "gcp.source.instance.region", to: "cloud.region"},
            {from: "gcp.source.instance.zone", to: "cloud.availability_zone"},
            {from: "gcp.source.vpc.subnetwork_name", to: "network.name"},
        ],
        ignore_missing: true,
    });

    // Set the cloud metadata fields based on the instance that reported the
    // event.
    var setCloudMetadata = function(evt) {
        var reporter = evt.Get("gcp.vpcflow.reporter");

        if (reporter === "DEST") {
            setCloudFromDestInstance.Run(evt);
        } else if (reporter === "SRC") {
            setCloudFromSrcInstance.Run(evt);
        }
    };

    var communityId = new processor.CommunityID({
        fields: {
            transport: "network.iana_number",
        }
    });

    // VPC flows are unidirectional so we only have to worry about copy the
    // source.bytes/packets over to network.bytes/packets.
    var setNetworkBytesPackets = new processor.Convert({
        fields: [
            {from: "source.bytes", to: "network.bytes"},
            {from: "source.packets", to: "network.packets"},
        ],
        ignore_missing: true,
    });

    // VPC flow logs are reported for TCP and UDP traffic only so handle these
    // protocols' IANA numbers.
    var setNetworkTransport = function(event) {
        var ianaNumber = event.Get("network.iana_number");
        switch (ianaNumber) {
            case "6":
                event.Put("network.transport", "tcp");
                break;
            case "17":
                event.Put("network.transport", "udp");
                break;
        }
    };

    var setNetworkDirection = function(event) {
        var srcInstance = event.Get("gcp.source.instance");
        var destInstance = event.Get("gcp.destination.instance");
        var direction = "unknown";

        if (srcInstance && destInstance) {
            direction = "internal";
        } else if (srcInstance) {
            direction = "outbound";
        } else if (destInstance) {
            direction = "inbound";
        }
        event.Put("network.direction", direction);
    };

    var setNetworkType = function(event) {
        var ip = event.Get("source.ip");
        if (!ip) {
            return;
        }

        if (ip.indexOf(".") !== -1) {
            event.Put("network.type", "ipv4");
        } else {
            event.Put("network.type", "ipv6");
        }
    };

    var setRelatedIP = function(event) {
        event.AppendTo("related.ip", event.Get("source.ip"));
        event.AppendTo("related.ip", event.Get("destination.ip"));
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(saveOriginalMessage)
        .Add(dropPubSubFields)
        .Add(categorizeEvent)
        .Add(saveMetadata)
        .Add(convertLogEntry)
        .Add(convertJsonPayload)
        .Add(dropEmptyObjects)
        .Add(copyAddressFields)
        .Add(setCloudMetadata)
        .Add(communityId)
        .Add(setNetworkBytesPackets)
        .Add(setNetworkTransport)
        .Add(setNetworkDirection)
        .Add(setNetworkType)
        .Add(setRelatedIP);

    if (internalNetworks) {
        pipeline = pipeline.AddNetworkDirection({
            source: "source.ip",
            destination: "destination.ip",
            target: "network.direction",
            internal_networks: internalNetworks,
        })
    }

    return {
        process: pipeline.Build().Run,
    };
}

var vpcflow;

// Register params from configuration.
function register(params) {
    vpcflow = new VPCFlow(params.keep_original_message, params.internal_networks);
}

function process(evt) {
    return vpcflow.process(evt);
}
