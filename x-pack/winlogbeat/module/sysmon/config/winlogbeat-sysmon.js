// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var sysmon = (function () {
    var path = require("path");
    var processor = require("processor");
    var winlogbeat = require("winlogbeat");

    var setProcessNameUsingExe = function(evt) {
        setProcessNameFromPath(evt, "process.executable", "process.name");
    };

    var setParentProcessNameUsingExe = function(evt) {
        setProcessNameFromPath(evt, "process.parent.executable", "process.parent.name");
    };

    var setProcessNameFromPath = function(evt, pathField, nameField) {
        var name = evt.Get(nameField);
        if (name) {
            return;
        }
        var exe = evt.Get(pathField);
        evt.Put(nameField, path.basename(exe));
    };

    var splitCommandLine = function(evt, field) {
        var commandLine = evt.Get(field);
        if (!commandLine) {
            return;
        }
        evt.Put(field, winlogbeat.splitCommandLine(commandLine));
    };

    var splitProcessArgs = function(evt) {
        splitCommandLine(evt, "process.args");
    };

    var splitParentProcessArgs = function(evt) {
        splitCommandLine(evt, "process.parent.args");
    };

    var addUser = function(evt) {
        var userParts = evt.Get("winlog.event_data.User").split("\\");
        if (userParts.length === 2) {
            evt.Delete("user");
            evt.Put("user.domain", userParts[0]);
            evt.Put("user.name", userParts[1]);
            evt.Delete("winlog.event_data.User");
        }
    };

    var addNetworkDirection = function(evt) {
        switch (evt.Get("winlog.event_data.Initiated")) {
            case "true":
                evt.Put("network.direction", "outbound");
                break;
            case "false":
                evt.Put("network.direction", "inbound");
                break;
        }
        evt.Delete("winlog.event_data.Initiated");
    };

    var addNetworkType = function(evt) {
        switch (evt.Get("winlog.event_data.SourceIsIpv6")) {
            case "true":
                evt.Put("network.type", "ipv6");
                break;
            case "false":
                evt.Put("network.type", "ipv4");
                break;
        }
        evt.Delete("winlog.event_data.SourceIsIpv6");
        evt.Delete("winlog.event_data.DestinationIsIpv6");
    };

    var addHashes = function(evt, hashField) {
        var hashes = evt.Get(hashField);
        evt.Delete(hashField);
        hashes.split(",").forEach(function(hash){
            var parts = hash.split("=");
            if (parts.length !== 2) {
                return;
            }

            var key = parts[0].toLowerCase();
            var value = parts[1].toLowerCase();
            evt.Put("hash."+key, value);
        });
    };

    var splitHashes = function(evt) {
        addHashes(evt, "winlog.event_data.Hashes");
    };

    var splitHash = function(evt) {
        addHashes(evt, "winlog.event_data.Hash");
    };

    var removeEmptyEventData = function(evt) {
        var eventData = evt.Get("winlog.event_data");
        if (eventData && Object.keys(eventData).length === 0) {
            evt.Delete("winlog.event_data");
        }
    };

    var parseUtcTime = new processor.Timestamp({
        field: "winlog.event_data.UtcTime",
        target_field: "winlog.event_data.UtcTime",
        timezone: "UTC",
        layouts: ["2006-01-02 15:04:05.999"],
        tests: ["2019-06-26 21:19:43.237"],
        ignore_missing: true,
    });

    var event1 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.CommandLine", to: "process.args"},
                {from: "winlog.event_data.CurrentDirectory", to: "process.working_directory"},
                {from: "winlog.event_data.ParentProcessGuid", to: "process.parent.entity_id"},
                {from: "winlog.event_data.ParentProcessId", to: "process.parent.pid", type: "long"},
                {from: "winlog.event_data.ParentImage", to: "process.parent.executable"},
                {from: "winlog.event_data.ParentCommandLine", to: "process.parent.args"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitProcessArgs)
        .Add(addUser)
        .Add(splitHashes)
        .Add(setParentProcessNameUsingExe)
        .Add(splitParentProcessArgs)
        .Add(removeEmptyEventData)
        .Build();

    var event2 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event3 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.Protocol", to: "network.transport"},
                {from: "winlog.event_data.SourceIp", to: "source.ip", type: "ip"},
                {from: "winlog.event_data.SourceHostname", to: "source.domain", type: "string"},
                {from: "winlog.event_data.SourcePort", to: "source.port", type: "long"},
                {from: "winlog.event_data.DestinationIp", to: "destination.ip", type: "ip"},
                {from: "winlog.event_data.DestinationHostname", to: "destination.domain", type: "string"},
                {from: "winlog.event_data.DestinationPort", to: "destination.port", type: "long"},
                {from: "winlog.event_data.DestinationPortName", to: "network.protocol"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(addUser)
        .Add(addNetworkDirection)
        .Add(addNetworkType)
        .CommunityID()
        .Add(removeEmptyEventData)
        .Build();

    var event4 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    var event5 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event6 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ImageLoaded", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(splitHashes)
        .Add(removeEmptyEventData)
        .Build();

    var event7 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.ImageLoaded", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitHashes)
        .Add(removeEmptyEventData)
        .Build();

    var event8 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.SourceProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.SourceProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.SourceImage", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event9 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.Device", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event10 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.SourceProcessGUID", to: "process.entity_id"},
                {from: "winlog.event_data.SourceProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.SourceThreadId", to: "process.thread.id", type: "long"},
                {from: "winlog.event_data.SourceImage", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event11 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event12 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event13 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event14 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event15 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.Image", to: "process.executable"},
                {from: "winlog.event_data.TargetFilename", to: "file.path"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(splitHash)
        .Add(removeEmptyEventData)
        .Build();

    var event16 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    var event17 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.PipeName", to: "file.name"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event18 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ProcessGuid", to: "process.entity_id"},
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.PipeName", to: "file.name"},
                {from: "winlog.event_data.Image", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event19 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    var event20 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.Destination", to: "process.executable"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(setProcessNameUsingExe)
        .Add(removeEmptyEventData)
        .Build();

    var event21 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addUser)
        .Add(removeEmptyEventData)
        .Build();

    var event255 = new processor.Chain()
        .Add(parseUtcTime)
        .Convert({
            fields: [
                {from: "winlog.event_data.UtcTime", to: "@timestamp"},
                {from: "winlog.event_data.ID", to: "error.code"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(removeEmptyEventData)
        .Build();

    return {
        // Event ID 1 - Process Create.
        1: event1.Run,

        // Event ID 2 - File creation time changed.
        2: event2.Run,

        // Event ID 3 - Network connection detected.
        3: event3.Run,

        // Event ID 4 - Sysmon service state changed.
        4: event4.Run,

        // Event ID 5 - Process terminated.
        5: event5.Run,

        // Event ID 6 - Driver loaded.
        6: event6.Run,

        // Event ID 7 - Image loaded.
        7: event7.Run,

        // Event ID 8 - CreateRemoteThread detected.
        8: event8.Run,

        // Event ID 9 - RawAccessRead detected.
        9: event9.Run,

        // Event ID 10 - Process accessed.
        10: event10.Run,

        // Event ID 11 - File created.
        11: event11.Run,

        // Event ID 12 - Registry object added or deleted.
        12: event12.Run,

        // Event ID 13 - Registry value set.
        13: event13.Run,

        // Event ID 14 - Registry object renamed.
        14: event14.Run,

        // Event ID 15 - File stream created.
        15: event15.Run,

        // Event ID 16 - Sysmon config state changed.
        16: event16.Run,

        // Event ID 17 - Pipe Created.
        17: event17.Run,

        // Event ID 18 - Pipe Connected.
        18: event18.Run,

        // Event ID 19 - WmiEventFilter activity detected.
        19: event19.Run,

        // Event ID 20 - WmiEventConsumer activity detected.
        20: event20.Run,

        // Event ID 21 - WmiEventConsumerToFilter activity detected.
        21: event21.Run,

        // Event ID 255 - Error report.
        255: event255.Run,

        process: function(evt) {
            var event_id = evt.Get("winlog.event_id");
            var processor= this[event_id];
            if (processor === undefined) {
                throw "unexpected sysmon event_id";
            }
            processor(evt);
        },
    };
})();

function process(evt) {
    return sysmon.process(evt);
}
