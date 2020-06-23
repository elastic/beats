// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function GSuiteSAML() {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.type", "access");
        evt.Put("event.category", "authentication");
    };

    var setEventOutcome = function(evt) {
        switch (evt.Get("event.action")) {
            case "login_failure":
                evt.Put("event.outcome", "failure");
                break;
            case "login_success":
                evt.Put("event.outcome", "success");
                break;
        }
    };

    var processParams = function(evt) {
        var params = evt.Get("events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        params.forEach(function(p){
            // all saml event parameters are strings.
            // for this reason we know for sure they are in the 'value' field.
            // https://developers.google.com/admin-sdk/reports/v1/appendix/activity/saml
            evt.Set(p.name, p.value);
        });

        evt.Delete("events.parameters");
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(setEventOutcome)
        .Add(processParams)
        .Build();

    return {
        process: pipeline.Run,
    };
}

var gsuite;

function register() {
    gsuite = new GSuiteSAML();
}

function process(evt) {
    return gsuite.process(evt);
}
