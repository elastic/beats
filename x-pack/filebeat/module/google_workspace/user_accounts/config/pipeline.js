// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var userAccounts = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.type", ["change", "user"]);
        evt.Put("event.category", ["iam"]);
    };

    var getParamValue = function(param) {
        if (param.value) {
            return param.value;
        }
        if (param.multiValue) {
            return param.multiValue;
        }
        if (param.intValue !== null) {
            return param.intValue;
        }
    };

    var flattenParams = function(evt) {
        var params = evt.Get("json.events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        params.forEach(function(p){
            evt.Put("google_workspace.user_accounts."+p.name, getParamValue(p));
        });

        evt.Delete("json.events.parameters");
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(flattenParams)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return userAccounts.process(evt);
}
