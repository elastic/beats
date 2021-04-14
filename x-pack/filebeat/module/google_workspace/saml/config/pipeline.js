// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var saml = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.type", ["start"]);
        evt.Put("event.category", ["authentication", "session"]);
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
        var params = evt.Get("json.events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        var prefixRegex = /^(saml_)/;

        params.forEach(function(p){
            p.name = p.name.replace(prefixRegex, "");

            // all saml event parameters are strings.
            // for this reason we know for sure they are in the 'value' field.
            // https://developers.google.com/admin-sdk/reports/v1/appendix/activity/saml
            evt.Put("google_workspace.saml."+p.name, p.value);
        });

        evt.Delete("json.events.parameters");
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(processParams)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return saml.process(evt);
}
