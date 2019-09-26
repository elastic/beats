// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var threat = (function () {
    var processor = require("processor");

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    var parseTimestamp = function (evt) {
        var secs = evt.Get("json.timestamp");
        evt.Delete("json.timestamp");
        evt.Put("@timestamp", new Date(secs * 1000));
    };

    var categorizeEvent = new processor.AddFields({
        target: "event",
        fields: {
            category: "threat-intel",
            type: "indicator",
        },
    });

    var setThreatFeedField = function (evt) {
        evt.Put("threat.threat_indicator.feed", "misp");
    };

    var convertFields = new processor.Convert({
        fields: [
            { from: "json.Event.info", to: "threat.threat_indicator.description" },
            { from: "json.Event.uuid", to: "threat.threat_indicator.id" },
        ],
        mode: "rename",
        ignore_missing: true,
    });

    var setAttackPattern = function (evt) {
        var indicator_type = evt.Get("json.type");
        var attackPattern;
        var arr;
        var filename;
        var v = evt.Get("json.value");
        switch (indicator_type) {
            case "ip-dst":
                attackPattern = '[destination.ip = ' + '"' + v + '"' + ']';
                break;
            case "ip-src":
                attackPattern = '[' + 'source.ip = ' + '"' + v + '"' + ']';
                break;
            case "filename":
                attackPattern = '[' + 'file.path = ' + '"' + v + '"' + ']';
                break;
            case "hostname":
                attackPattern = '[' + 'source.domain = ' + '"' + v + '"' + ' OR destination.domain = ' + '"' + v + '"' + ']';
                break;
            case "sha512":
                attackPattern = '[' + 'file.sha512 = ' + '"' + v + '"' + ']';
                break;
            case "sha256":
                attackPattern = '[' + 'file.sha256 = ' + '"' + v + '"' + ']';
                break;
            case "md5":
                attackPattern = '[' + 'file.md5 = ' + '"' + v + '"' + ']';
                break;
            case "sha1":
                attackPattern = '[' + 'file.sha1 = ' + '"' + v + '"' + ']';
                break;
            case "link":
                attackPattern = '[' + 'url.full = ' + '"' + v + '"' + ']';
                break;
            case "url":
                attackPattern = '[' + 'url.full = ' + '"' + v + '"' + ']';
                break;
            case "domain":
                attackPattern = '[' + 'dns.question.name = ' + '"' + v + '"' + ' OR zeek.dns.query = ' + '"' + v + '"' + ' OR url.domain = ' + '"' + v + '"' + ']';
                break;
            case "domain|ip":
                arr = v.split("|");
                if (arr.length == 2) {
                    var domain = arr[0];
                    var ip = arr[1];
                    attackPattern = '[' + '(' + 'dns.question.name = ' + '"' + domain + '"' + ' OR zeek.dns.query = ' + '"' + domain + '"' + ' OR url.domain = ' + '"' + domain + '"' + ')' +
                        ' AND ' + '(' + 'source.ip = ' + '"' + ip + '"' + ' OR destination.ip = ' + '"' + ip + '"' + ')' + ']';
                }
                break;
            case "filename|sha256":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var sha256 = arr[1];
                    attackPattern = '[' + 'file.sha256 = ' + '"' + sha256 + '"' + ' AND file.path = ' + '"' + filename + '"' + ']';
                }
                break;
            case "filename|sha1":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var sha1 = arr[1];
                    attackPattern = '[' + 'file.sha1 = ' + '"' + sha1 + '"' + ' AND file.path = ' + '"' + filename + '"' + ']';
                }
                break;
            case "filename|md5":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var md5 = arr[1];
                    attackPattern = '[' + 'file.md5 = ' + '"' + md5 + '"' + ' AND file.path = ' + '"' + filename + '"' + ']';
                }
                break;
        }
        if (attackPattern == undefined) {
            evt.Put("error.message", 'Unsupported type: ' + indicator_type)
        }
        evt.Put("threat.threat_indicator.attack_pattern", attackPattern);
        
    }

    var dropOldFields = function (evt) {
        evt.Delete("message");
        evt.Delete("json");
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(categorizeEvent)
        .Add(setThreatFeedField)
        .Add(convertFields)
        .Add(setAttackPattern)
        .Add(dropOldFields)
        .Build();

    return {
        process: pipeline.Run,
    };
})();

function process(evt) {
    return threat.process(evt);
}
