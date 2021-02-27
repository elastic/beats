// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var userAccounts = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.type", ["change", "user"]);
        evt.Put("event.category", ["iam"]);
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return userAccounts.process(evt);
}
