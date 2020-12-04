// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var powershell = (function () {
    var path = require("path");
    var processor = require("processor");
    var windows = require("windows");

    var normalizeCommonFieldNames = new processor.Convert({
        fields: [
            {
                from: "winlog.event_data.Engine Version",
                to: "winlog.event_data.EngineVersion",
            },
            {
                from: "winlog.event_data.Pipeline ID",
                to: "winlog.event_data.PipelineId",
            },
            {
                from: "winlog.event_data.Runspace ID",
                to: "winlog.event_data.RunspaceId",
            },
            {
                from: "winlog.event_data.Host Version",
                to: "winlog.event_data.HostVersion",
            },
            {
                from: "winlog.event_data.Script Name",
                to: "winlog.event_data.ScriptName",
            },
            {
                from: "winlog.event_data.Path",
                to: "winlog.event_data.ScriptName",
            },
            {
                from: "winlog.event_data.Command Path",
                to: "winlog.event_data.CommandPath",
            },
            {
                from: "winlog.event_data.Command Name",
                to: "winlog.event_data.CommandName",
            },
            {
                from: "winlog.event_data.Command Type",
                to: "winlog.event_data.CommandType",
            },
            {
                from: "winlog.event_data.User",
                to: "winlog.event_data.UserId",
            },
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    })

    // Builds a dissect tokenizer.
    //
    // - chunks:    number of chunks dissect needs to look for.
    // - delimiter: indicates what is the delimiter between chunks,
    //              in addition to `\n` which is already expected.
    // - sep:       separator between key value pairs.
    //
    // example:
    // For a string like "Foo=Bar\n\tBar=Baz", chunks: 2, delimiter: '\t', sep: '='
    var buildNewlineSpacedTokenizer = function (chunks, delimiter, sep) {
        var tokenizer = "";
        for (var i = 0; i < chunks; i++) {
            if (i !== 0) {
                tokenizer += "\n%{}";
            }
            tokenizer += delimiter+"%{*p"+i+"}"+sep+"%{&p"+i+"}";
        }
        return tokenizer;
    };

    var dissectField = function (fromField, targetPrefix, chunks, delimiter, sep) {
        return new processor.Dissect({
            field: fromField,
            target_prefix: targetPrefix,
            tokenizer: buildNewlineSpacedTokenizer(chunks, delimiter, sep),
            fail_on_error: false,
        });
    };

    // countChunksDelimitedBy will return the number of chunks contained in a field
    // that are delimited by the given delimiter.
    var countChunksDelimitedBy = function(evt, fromField, delimiter) {
        var str = evt.Get(fromField);
        if (!str) {
            return 0;
        }
        return str.split(delimiter).length-1;
    };

    var dissect4xxAnd600 = function (evt) {
        var delimiter = "\t";
        var chunks = countChunksDelimitedBy(evt, "winlog.event_data.param3", delimiter);

        dissectField("winlog.event_data.param3", "winlog.event_data", chunks, delimiter, "=").Run(evt);

        // these fields contain redundant information.
        evt.Delete("winlog.event_data.param1");
        evt.Delete("winlog.event_data.param2");
        evt.Delete("winlog.event_data.param3");
    };

    var dissect800Detail = function (evt) {
        var delimiter = "\t";
        var chunks = countChunksDelimitedBy(evt, "winlog.event_data.param2", delimiter);

        dissectField("winlog.event_data.param2", "winlog.event_data", chunks, "\t", "=").Run(evt);

        // these fields contain redundant information.
        evt.Delete("winlog.event_data.param1");
        evt.Delete("winlog.event_data.param2");
    };

    var dissect4103 = function (evt) {
        var delimiter = "        ";
        var chunks = countChunksDelimitedBy(evt, "winlog.event_data.ContextInfo", delimiter);

        dissectField("winlog.event_data.ContextInfo", "winlog.event_data", chunks, delimiter, " = ").Run(evt);

        // these fields contain redundant information.
        evt.Delete("winlog.event_data.ContextInfo");
        evt.Delete("winlog.event_data.Severity");
    };

    var addEngineVersion = function (evt) {
        var version = evt.Get("winlog.event_data.EngineVersion");
        evt.Delete("winlog.event_data.EngineVersion");
        if (!version) {
            return;
        }

        evt.Put("powershell.engine.version", version);
    };

    var addPipelineID = function (evt) {
        var id = evt.Get("winlog.event_data.PipelineId");
        evt.Delete("winlog.event_data.PipelineId");
        if (!id) {
            return;
        }

        evt.Put("powershell.pipeline_id", id);
    };

    var addRunspaceID = function (evt) {
        var id = evt.Get("winlog.event_data.RunspaceId");
        evt.Delete("winlog.event_data.RunspaceId");
        if (!id) {
            return;
        }

        evt.Put("powershell.runspace_id", id);
    };

    var addScriptBlockID = function (evt) {
        var id = evt.Get("winlog.event_data.ScriptBlockId");
        evt.Delete("winlog.event_data.ScriptBlockId");
        if (!id) {
            return;
        }

        evt.Put("powershell.file.script_block_id", id);
    };

    var addScriptBlockText = function (evt) {
        var text = evt.Get("winlog.event_data.ScriptBlockText");
        evt.Delete("winlog.event_data.ScriptBlockText");
        if (!text) {
            return;
        }

        evt.Put("powershell.file.script_block_text", text);
    };

    var splitCommandLine = function (evt, source, target) {
        var commandLine = evt.Get(source);
        if (!commandLine) {
            return;
        }
        evt.Put(target, windows.splitCommandLine(commandLine));
    };

    var addProcessArgs = function (evt) {
        splitCommandLine(evt, "process.command_line", "process.args");
        var args = evt.Get("process.args");
        if (args && args.length > 0) {
            evt.Put("process.args_count", args.length);
        }
    };

    var addExecutableVersion = function (evt) {
        var version = evt.Get("winlog.event_data.HostVersion");
        evt.Delete("winlog.event_data.HostVersion");
        if (!version) {
            return;
        }

        evt.Put("powershell.process.executable_version", version);
    };

    var addFileInfo = function (evt) {
        var scriptName = evt.Get("winlog.event_data.ScriptName");
        evt.Delete("winlog.event_data.ScriptName");
        if (!scriptName) {
            return;
        }

        evt.Put("file.path", scriptName);
        evt.Put("file.name", path.basename(scriptName));
        evt.Put("file.directory", path.dirname(scriptName));

        // path returns extensions with a preceding ., e.g.: .tmp, .png
        // according to ecs the expected format is without it, so we need to remove it.
        var ext = path.extname(scriptName);
        if (!ext) {
            return;
        }

        if (ext.charAt(0) === ".") {
            ext = ext.substr(1);
        }
        evt.Put("file.extension", ext);
    };

    var addCommandValue = function (evt) {
        var value = evt.Get("winlog.event_data.CommandLine")
        evt.Delete("winlog.event_data.CommandLine");
        if (!value) {
            return;
        }

        evt.Put("powershell.command.value", value.trim());
    };

    var addCommandPath = function (evt) {
        var commandPath = evt.Get("winlog.event_data.CommandPath");
        evt.Delete("winlog.event_data.CommandPath");
        if (!commandPath) {
            return;
        }

        evt.Put("powershell.command.path", commandPath);
    };

    var addCommandName = function (evt) {
        var commandName = evt.Get("winlog.event_data.CommandName");
        evt.Delete("winlog.event_data.CommandName");
        if (!commandName) {
            return;
        }

        evt.Put("powershell.command.name", commandName);
    };

    var addCommandType = function (evt) {
        var commandType = evt.Get("winlog.event_data.CommandType");
        evt.Delete("winlog.event_data.CommandType");
        if (!commandType) {
            return;
        }

        evt.Put("powershell.command.type", commandType);
    };

    var detailRegex = /^(.+)\((.+)\)\:\s*(.+)?$/;
    var parameterBindingRegex = /^.*name\=(.+);\s*value\=(.+)$/

    // Parses a command invocation detail raw line, and converts it to an object, based on its type.
    //
    // - for unexpectedly formatted ones: {value: "the raw line as it is"}
    // - for all:
    //      * related_command: describes to what command it is related to
    //      * value: the value for that detail line
    //      * type: the type of the detail line, i.e.: CommandInvocation, ParameterBinding, NonTerminatingError
    // - additionally, ParameterBinding adds a `name` field with the parameter name being bound.
    var parseRawDetail = function (raw) {
        var matches = detailRegex.exec(raw);
        if (!matches || matches.length !== 4) {
            return {value: raw};
        }

        if (matches[1] !== "ParameterBinding") {
            return {type: matches[1], related_command: matches[2], value: matches[3]};
        }

        var nameValMatches = parameterBindingRegex.exec(matches[3]);
        if (!nameValMatches || nameValMatches.length !== 3) {
            return {value: matches[3]};
        }

        return {
            type: matches[1],
            related_command: matches[2],
            name: nameValMatches[1],
            value: nameValMatches[2],
        };
    };

    var addCommandInvocationDetails = function (evt, from) {
        var rawDetails = evt.Get(from);
        if (!rawDetails) {
            return;
        }

        var details = [];
        rawDetails.split("\n").forEach(function (raw) {
            details.push(parseRawDetail(raw));
        });

        if (details.length === 0) {
            return;
        }

        evt.Delete(from);
        evt.Put("powershell.command.invocation_details", details);
    };

    var addCommandInvocationDetailsForEvent800 = function (evt) {
        addCommandInvocationDetails(evt, "winlog.event_data.param3");
    };

    var addCommandInvocationDetailsForEvent4103 = function (evt) {
        addCommandInvocationDetails(evt, "winlog.event_data.Payload");
    };

    var addUser = function (evt) {
        var userParts = evt.Get("winlog.event_data.UserId").split("\\");
        evt.Delete("winlog.event_data.UserId");
        if (userParts.length === 2) {
            evt.Delete("user");
            evt.Put("user.domain", userParts[0]);
            evt.Put("user.name", userParts[1]);
            evt.AppendTo("related.user", userParts[1]);
            evt.Delete("winlog.event_data.UserId");
        }
    };

    var addConnectedUser = function (evt) {
        var userParts = evt.Get("winlog.event_data.Connected User").split("\\");
        evt.Delete("winlog.event_data.Connected User");
        if (userParts.length === 2) {
            evt.Put("powershell.connected_user.domain", userParts[0]);
            evt.Put("powershell.connected_user.name", userParts[1]);
            evt.AppendTo("related.user", userParts[1]);
        }
    };

    var removeEmptyEventData = function (evt) {
        var eventData = evt.Get("winlog.event_data");
        if (eventData && Object.keys(eventData).length === 0) {
            evt.Delete("winlog.event_data");
        }
    };

    var event4xxAnd600Common = new processor.Chain()
        .Add(dissect4xxAnd600)
        .Convert({
            fields: [
                {
                    from: "winlog.event_data.SequenceNumber",
                    to: "event.sequence",
                    type: "long",
                },
                {
                    from: "winlog.event_data.NewEngineState",
                    to: "powershell.engine.new_state",
                },
                {
                    from: "winlog.event_data.PreviousEngineState",
                    to: "powershell.engine.previous_state",
                },
                {
                    from: "winlog.event_data.NewProviderState",
                    to: "powershell.provider.new_state",
                },
                {
                    from: "winlog.event_data.ProviderName",
                    to: "powershell.provider.name",
                },
                {
                    from: "winlog.event_data.HostId",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.HostApplication",
                    to: "process.command_line",
                },
                {
                    from: "winlog.event_data.HostName",
                    to: "process.title",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addEngineVersion)
        .Add(addPipelineID)
        .Add(addRunspaceID)
        .Add(addProcessArgs)
        .Add(addExecutableVersion)
        .Add(addFileInfo)
        .Add(addCommandValue)
        .Add(addCommandPath)
        .Add(addCommandName)
        .Add(addCommandType)
        .Add(removeEmptyEventData)
        .Build();

    var event400 = new processor.Chain()
        .AddFields({
            fields: {
                category: ["process"],
                type: ["start"],
            },
            target: "event",
        })
        .Add(event4xxAnd600Common)
        .Build()

    var event403 = new processor.Chain()
        .AddFields({
            fields: {
                category: ["process"],
                type: ["end"],
            },
            target: "event",
        })
        .Add(event4xxAnd600Common)
        .Build()

    var event600 = new processor.Chain()
        .AddFields({
            fields: {
                category: ["process"],
                type: ["info"],
            },
            target: "event",
        })
        .Add(event4xxAnd600Common)
        .Build()

    var event800 = new processor.Chain()
        .Add(dissect800Detail)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["info"],
            },
            target: "event",
        })
        .Convert({
            fields: [
                {
                    from: "winlog.event_data.SequenceNumber",
                    to: "event.sequence",
                    type: "long",
                },
                {
                    from: "winlog.event_data.HostId",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.HostApplication",
                    to: "process.command_line",
                },
                {
                    from: "winlog.event_data.HostName",
                    to: "process.title",
                },
                {
                    from: "winlog.event_data.DetailTotal",
                    to: "powershell.total",
                    type: "long",
                },
                {
                    from: "winlog.event_data.DetailSequence",
                    to: "powershell.sequence",
                    type: "long",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(addEngineVersion)
        .Add(addPipelineID)
        .Add(addRunspaceID)
        .Add(addProcessArgs)
        .Add(addExecutableVersion)
        .Add(addFileInfo)
        .Add(addCommandValue)
        .Add(addCommandPath)
        .Add(addCommandName)
        .Add(addCommandType)
        .Add(addUser)
        .Add(addCommandInvocationDetailsForEvent800)
        .Add(removeEmptyEventData)
        .Build();

    var event4103 = new processor.Chain()
        .Add(dissect4103)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["info"],
            },
            target: "event",
        })
        .Convert({
            fields: [
                {
                    from: "winlog.event_data.Sequence Number",
                    to: "event.sequence",
                    type: "long",
                },
                {
                    from: "winlog.event_data.Host ID",
                    to: "process.entity_id",
                },
                {
                    from: "winlog.event_data.Host Application",
                    to: "process.command_line",
                },
                {
                    from: "winlog.event_data.Host Name",
                    to: "process.title",
                },
                {
                    from: "winlog.event_data.Shell ID",
                    to: "powershell.id",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(normalizeCommonFieldNames)
        .Add(addEngineVersion)
        .Add(addPipelineID)
        .Add(addRunspaceID)
        .Add(addProcessArgs)
        .Add(addExecutableVersion)
        .Add(addFileInfo)
        .Add(addCommandValue)
        .Add(addCommandPath)
        .Add(addCommandName)
        .Add(addCommandType)
        .Add(addUser)
        .Add(addConnectedUser)
        .Add(addCommandInvocationDetailsForEvent4103)
        .Add(removeEmptyEventData)
        .Build();

    var event4104 = new processor.Chain()
        .AddFields({
            fields: {
                category: ["process"],
                type: ["info"],
            },
            target: "event",
        })
        .Convert({
            fields: [
                {
                    from: "winlog.event_data.MessageNumber",
                    to: "powershell.sequence",
                    type: "long",
                },
                {
                    from: "winlog.event_data.MessageTotal",
                    to: "powershell.total",
                    type: "long",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(normalizeCommonFieldNames)
        .Add(addFileInfo)
        .Add(addScriptBlockID)
        .Add(addScriptBlockText)
        .Add(removeEmptyEventData)
        .Build();

    var event4105And4106Common = new processor.Chain()
        .Add(addRunspaceID)
        .Add(addScriptBlockID)
        .Add(removeEmptyEventData)
        .Build();

    var event4105 = new processor.Chain()
        .Add(event4105And4106Common)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["start"],
            },
            target: "event",
        })
        .Build();

    var event4106 = new processor.Chain()
        .Add(event4105And4106Common)
        .AddFields({
            fields: {
                category: ["process"],
                type: ["end"],
            },
            target: "event",
        })
        .Build();

    return {
        400: event400.Run,
        403: event403.Run,
        600: event600.Run,
        800: event800.Run,
        4103: event4103.Run,
        4104: event4104.Run,
        4105: event4105.Run,
        4106: event4106.Run,

        process: function(evt) {
            var eventId = evt.Get("winlog.event_id");
            var processor = this[eventId];
            if (processor === undefined) {
                return;
            }
            evt.Put("event.module", "powershell");
            processor(evt);
        },
    };
})();

function process(evt) {
    return powershell.process(evt);
}
