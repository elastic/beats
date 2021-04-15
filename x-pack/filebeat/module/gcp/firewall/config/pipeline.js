// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var processor = require("processor");
var console   = require("console");

// makeMapper({from:field, to:field, default:value mappings:{orig: new, [...]}})
//
// Processor that sets _to_ field from a mapping of _from_ field's value.
function makeMapper(options) {
    return function (evt) {
        var key = evt.Get(options.from);
        var value = options.default;
        if (key in options.mappings) {
            value = options.mappings[key];
        }
        if (value != null) {
            evt.Put(options.to, value);
        }
    };
}

// makeConditional({condition:expr, result1:processor|expr, [...]})
//
// Processor that selects which processor to run depending on the result of
// evaluating a _condition_. Result can be boolean (if-else equivalent) or any
// other value (switch equivalent). Unspecified values are a no-op.
function makeConditional(options) {
    return function (evt) {
        var branch = options[options.condition(evt)] || function(evt){};
        return (typeof branch === "function" ? branch : branch.Run)(evt);
    };
}

// logEvent(msg)
//
// Processor that logs the current value of evt to console.debug.
function makeLogEvent(msg) {
    return function (evt) {
        console.debug(msg + " :" +  JSON.stringify(evt, null, 4));
    };
}

// PipelineBuilder to aid debugging of pipelines during development.
function PipelineBuilder(pipelineName, debug) {
    this.pipeline = new processor.Chain();
    this.add = function (processor) {
        this.pipeline = this.pipeline.Add(processor);
    };
    this.Add = function (name, processor) {
        this.add(processor);
        if (debug) {
            this.add(makeLogEvent("after " + pipelineName + "/" + name));
        }
    };
    this.Build = function () {
        if (debug) {
            this.add(makeLogEvent(pipelineName + "processing done"));
        }
        return this.pipeline.Build();
    };
    if (debug) {
        this.add(makeLogEvent(pipelineName + ": begin processing event"));
    }
}

function FirewallProcessor(keep_original_message, debug, internalNetworks) {
    var builder = new PipelineBuilder("firewall", debug);

    // The pub/sub input writes the Stackdriver LogEntry object into the message
    // field. The message needs decoded as JSON.
    builder.Add("decodeJson", new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json"
    }));

    // Set @timestamp to the LogEntry's timestamp.
    builder.Add("parseTimestamp", new processor.Timestamp({
        field: "json.timestamp",
        timezone: "UTC",
        layouts: ["2006-01-02T15:04:05.999999999Z07:00"],
        tests: ["2019-06-14T03:50:10.845445834Z"],
        ignore_missing: true
    }));

    if (keep_original_message) {
        builder.Add("saveOriginalMessage", new processor.Convert({
            fields: [
                {from: "message", to: "event.original"}
            ],
            mode: "rename"
        }));
    }

    builder.Add("dropPubSubFields", function(evt) {
        evt.Delete("message");
        evt.Delete("labels");
    });

    builder.Add("categorizeEvent", new processor.AddFields({
        target: "event",
        fields: {
            kind: "event",
            category: "network",
            type: "connection",
            action: "firewall-rule"
        },
    }));

    builder.Add("saveMetadata", new processor.Convert({
        fields: [
            {from: "json.logName", to: "log.logger"},
            {from: "json.resource.labels.subnetwork_name", to: "network.name"},
            {from: "json.insertId", to: "event.id"}
        ],
        ignore_missing: true
    }));

    // Firewall logs are structured so the LogEntry includes a jsonPayload field.
    // https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry
    // The LogEntry's jsonPayload is moved to the json field. The jsonPayload
    // contains the structured VPC flow log fields.
    builder.Add("convertLogEntry", new processor.Convert({
        fields: [
            {from: "json.jsonPayload", to: "json"},
        ],
        mode: "rename"
    }));

    builder.Add("addType", function(evt) {
        var disp = evt.Get("json.disposition");
        if (disp != null) {
            evt.AppendTo("event.type", disp.toLowerCase());
        }
    });

    builder.Add("addDirection", makeMapper({
        from: "json.rule_details.direction",
        to: "network.direction",
        mappings: {
            INGRESS: "inbound",
            EGRESS: "outbound"
        },
        default: "unknown"
    }));

    builder.Add("conditionalRename", makeConditional({
       condition: function(evt) {
          return evt.Get("json.rule_details.direction");
       },
       EGRESS: processor.Convert({
            fields: [
                {from: "json.vpc", to: "json.src_vpc"},
                {from: "json.instance", to: "json.src_instance"},
                {from: "json.location", to: "json.src_location"},
                {from: "json.remote_vpc", to: "json.dest_vpc"},
                {from: "json.remote_instance", to: "json.dest_instance"},
                {from: "json.remote_location", to: "json.dest_location"}
            ],
            mode: "rename",
            fail_on_error: false,
            ignore_missing: true
        }),

        INGRESS: processor.Convert({
            fields: [
                {from: "json.vpc", to: "json.dest_vpc"},
                {from: "json.instance", to: "json.dest_instance"},
                {from: "json.location", to: "json.dest_location"},
                {from: "json.remote_vpc", to: "json.src_vpc"},
                {from: "json.remote_instance", to: "json.src_instance"},
                {from: "json.remote_location", to: "json.src_location"}
            ],
            mode: "rename",
            fail_on_error: false,
            ignore_missing: true
        })
    }));

    // Set network.iana_number from connection.protocol, converting it to long
    // and ignoring the failure if it's not numeric.
    builder.Add("ianaNumber", new processor.Convert({
        fields: [{
            from:  "json.connection.protocol",
            to: "network.iana_number",
            type: "long"
        }],
        fail_on_error: false
    }));

    // Set network.transport from iana_number. GCP Firewall only supports
    // logging of tcp and udp connections, added icmp just in case as it's the
    // other protocol supported by firewall rules.
    builder.Add("transportFromIANA", makeMapper({
        from: "network.iana_number",
        to: "network.transport",
        mappings: {
            1: "icmp",
            6: "tcp",
            17: "udp"
        }
    }));

    builder.Add("convertJsonPayload", new processor.Convert({
        fields: [
            {from: "json.connection.dest_ip", to: "destination.address"},
            {from: "json.connection.dest_port", to: "destination.port", type: "long"},
            {from: "json.connection.src_ip", to: "source.address"},
            {from: "json.connection.src_port", to: "source.port", type: "long"},

            {from: "json.src_instance.vm_name", to: "source.domain"},
            {from: "json.dest_instance.vm_name", to: "destination.domain"},

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
            {from: "json.rule_details.reference", to: "rule.name"},
            {from: "json", to: "gcp.firewall"},
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false
    }));

    // Delete emtpy object's whose fields have been renamed leaving them childless.
    builder.Add("dropEmptyObjects", function (evt) {
        evt.Delete("gcp.firewall.connection");
        evt.Delete("gcp.firewall.dest_location");
        evt.Delete("gcp.firewall.disposition");
        evt.Delete("gcp.firewall.src_location");
    });

    // Copy the source/destination.address to source/destination.ip if they are
    // valid IP addresses.
    builder.Add("copyAddressFields", new processor.Convert({
        fields: [
            {from: "source.address", to: "source.ip", type: "ip"},
            {from: "destination.address", to: "destination.ip", type: "ip"}
        ],
        fail_on_error: false
    }));

    builder.Add("setCloudMetadata", makeConditional({
       condition: function (evt) {
           return evt.Get("json.rule_details.direction");
       },
       EGRESS: new processor.Convert({
           fields: [
               {from: "gcp.source.instance.project_id", to: "cloud.project.id"},
               {from: "gcp.source.instance.vm_name", to: "cloud.instance.name"},
               {from: "gcp.source.instance.region", to: "cloud.region"},
               {from: "gcp.source.instance.zone", to: "cloud.availability_zone"},
               {from: "gcp.source.vpc.subnetwork_name", to: "network.name"}
           ],
           ignore_missing: true
       }),

       INGRESS: new processor.Convert({
           fields: [
               {from: "gcp.destination.instance.project_id", to: "cloud.project.id"},
               {from: "gcp.destination.instance.vm_name", to: "cloud.instance.name"},
               {from: "gcp.destination.instance.region", to: "cloud.region"},
               {from: "gcp.destination.instance.zone", to: "cloud.availability_zone"},
               {from: "gcp.destination.vpc.subnetwork_name", to: "network.name"},
           ],
           ignore_missing: true
       })
    }));

    builder.Add("communityId", new processor.CommunityID({
        fields: {
            transport: "network.iana_number"
        }
    }));

    builder.Add("setInternalDirection", function(event) {
        var srcInstance = event.Get("gcp.source.instance");
        var destInstance = event.Get("gcp.destination.instance");
        if (srcInstance && destInstance) {
            event.Put("network.direction", "internal");
        }
    });

    builder.Add("setNetworkType", function(event) {
        var ip = event.Get("source.ip");
        if (!ip) {
            return;
        }

        if (ip.indexOf(".") !== -1) {
            event.Put("network.type", "ipv4");
        } else {
            event.Put("network.type", "ipv6");
        }
    });

    builder.Add("setRelatedIP", function(event) {
        event.AppendTo("related.ip", event.Get("source.ip"));
        event.AppendTo("related.ip", event.Get("destination.ip"));
    });

    if (internalNetworks) {
        builder.Add("addNetworkDirection", processor.AddNetworkDirection({
            source: "source.ip",
            destination: "destination.ip",
            target: "network.direction",
            internal_networks: internalNetworks,
        }))
    }

    return {
        process: builder.Build().Run
    };
}

var firewall;

// Register params from configuration.
function register(params) {
    firewall = new FirewallProcessor(params.keep_original_message, params.debug, params.internal_networks);
}

function process(evt) {
    return firewall.process(evt);
}
