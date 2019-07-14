// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var security = (function () {
    var path = require("path");
    var processor = require("processor");
    var winlogbeat = require("winlogbeat");

    var addAuthSuccess = new processor.AddFields({
        fields: {
            "event.category": "authentication",
            "event.type": "authentication_success",
        },
        target: "",
    });

    var addAuthFailed = new processor.AddFields({
        fields: {
            "event.category": "authentication",
            "event.type": "authentication_failure",
        },
        target: "",
    });

    var convertAuthentication = new processor.Convert({
        fields: [
            {from: "winlog.event_data.TargetUserSid", to: "user.id"},
            {from: "winlog.event_data.TargetUserName", to: "user.name"},
            {from: "winlog.event_data.TargetDomainName", to: "user.domain"},
            {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
            {from: "winlog.event_data.ProcessName", to: "process.executable"},
            {from: "winlog.event_data.IpAddress", to: "source.ip", type: "ip"},
            {from: "winlog.event_data.IpPort", to: "source.port", type: "long"},
            {from: "winlog.event_data.WorkstationName", to: "source.domain"},
        ],
        mode: "rename",
        ignore_missing: true,
        fail_on_error: false,
    });

    var setProcessNameUsingExe = function(evt) {
        var name = evt.Get("process.name");
        if (name) {
            return;
        }
        var exe = evt.Get("process.executable");
        evt.Put("process.name", path.basename(exe));
    };

    var logonSuccess = new processor.Chain()
        .Add(addAuthSuccess)
        .Add(convertAuthentication)
        .Add(setProcessNameUsingExe)
        .Build();
    
    var logout = new processor.Chain()
        .Add(convertAuthentication)
        .Build();
        
    var logonFailed = new processor.Chain()
        .Add(addAuthFailed)
        .Add(convertAuthentication)
        .Add(setProcessNameUsingExe)
        .Build();

    return {
        // 4624 - An account was successfully logged on.
        4624: logonSuccess.Run,

        // 4625 - An account failed to log on.
        4625: logonFailed.Run,
        
        // 4634 - An account fwas logged off.
        4634: logout.Run,
        
        // 4648 - A logon was attempted using explicit credentials.
        4648: logonSuccess.Run,

        process: function(evt) {
            var event_id = evt.Get("winlog.event_id");
            var processor = this[event_id];
            if (processor === undefined) {
                return;
            }
            processor(evt);
        },
    };
})();

function process(evt) {
    return security.process(evt);
}
