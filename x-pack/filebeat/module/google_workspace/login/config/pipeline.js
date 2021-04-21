// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var login = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.category", ["authentication"]);
        switch (evt.Get("event.action")) {
            case "login_failure":
                evt.AppendTo("event.category", "session");
                evt.Put("event.type", ["start"]);
                evt.Put("event.outcome", "failure");
                break;
            case "login_success":
                evt.AppendTo("event.category", "session");
                evt.Put("event.type", ["start"]);
                evt.Put("event.outcome", "success");
                break;
            case "logout":
                evt.AppendTo("event.category", "session");
                evt.Put("event.type", ["end"]);
                break;
            case "account_disabled_generic":
            case "account_disabled_spamming_through_relay":
            case "account_disabled_spamming":
            case "account_disabled_hijacked":
            case "account_disabled_password_leak":
                evt.Put("event.type", ["user", "change"]);
                break;
            case "gov_attack_warning":
            case "login_challenge":
            case "login_verification":
            case "suspicious_login":
            case "suspicious_login_less_secure_app":
            case "suspicious_programmatic_login":
                evt.Put("event.type", ["info"]);
                break;
        }
    };

    var getParamValue = function(param) {
        if (param.value) {
            return param.value;
        }
        if (param.multiValue) {
            return param.multiValue;
        }
    };

    var processParams = function(evt) {
        var params = evt.Get("json.events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        var prefixRegex = /^(login_)/;

        params.forEach(function(p){
            p.name = p.name.replace(prefixRegex, "");
            switch (p.name) {
                // According to https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login
                // this is a timestamp in microseconds
                case "timestamp":
                    var millis = p.intValue / 1000;
                    evt.Put("event.start", new Date(millis));
                    break;
                case "challenge_status":
                    if (p.value === "Challenge Passed") {
                        evt.Put("event.outcome", "success");
                    } else {
                        evt.Put("event.outcome", "failure");
                    }
                    break;
                case "is_second_factor":
                case "is_suspicious":
                    evt.Put("google_workspace.login."+p.name, p.boolValue);
                    break;
                // the rest of params are strings
                default:
                    evt.Put("google_workspace.login."+p.name, getParamValue(p));
            }
        });

        evt.Delete("json.events.parameters");
    };

    var addTargetUser = function(evt) {
        var affectedEmail = evt.Get("google_workspace.login.affected_email_address");
        if (affectedEmail) {
            evt.Put("user.target.email", affectedEmail);
            var data = affectedEmail.split("@");
            if (data.length !== 2) {
                return;
            }

            evt.Put("user.target.name", data[0]);
            evt.Put("user.target.domain", data[1]);
            evt.AppendTo("related.user", data[0]);
        }
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(processParams)
        .Add(addTargetUser)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return login.process(evt);
}
