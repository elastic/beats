// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var threat = (function () {
    var processor = require("processor");

    var copyToOriginal = function (evt) {
        evt.Put("event.original", evt.Get("message"));
    };

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    var setID = function (evt) {
        evt.Put("@metadata._id", evt.Get("event.id"));
    };

    var categorizeEvent = new processor.AddFields({
        target: "event",
        fields: {
            kind: "event",
            category: "threat-intel",
            type: "indicator",
        },
    });

    var setThreatFeedField = function (evt) {
        evt.Put("misp.threat_indicator.feed", "misp");
    };

    var convertFields = new processor.Convert({
        fields: [
            { from: "json.Event.id", to: "rule.id" },
            { from: "json.Event.info", to: "misp.threat_indicator.description" },
            { from: "json.Event.info", to: "rule.description" },
            { from: "json.Event.uuid", to: "misp.threat_indicator.id" },
            { from: "json.Event.uuid", to: "rule.uuid" },
            { from: "json.category", to: "rule.category" },
            { from: "json.uuid", to: "event.id" },
        ],
        mode: "rename",
        ignore_missing: true,
    });

    // Copy tag names from MISP event to tags field.
    var copyTags = function (evt) {
        var mispTags = evt.Get("json.Tag");
        if (!mispTags) {
            return;
        }
        mispTags.forEach(function (tag) {
            if (tag.name) {
                evt.AppendTo("tags", tag.name);
            }
        });
    };

    var setAttackPattern = function (evt) {
        var indicator_type = evt.Get("json.type");
        var attackPattern;
        var attackPatternKQL;
        var arr;
        var ip;
        var filename;
        var v = evt.Get("json.value");
        evt.Put("message", v);
        evt.Put("misp.threat_indicator.type", indicator_type);
        switch (indicator_type) {
            case "AS":
                var asn;
                if (v.substring(0, 2) == "AS") {
                    asn = v.substring(2, v.length);
                } else {
                    asn = v;
                }
                attackPattern = '[' + 'source:as:number = ' + '\'' + asn + '\'' + ' OR destination:as:number = ' + '\'' + asn + '\'' + ']';
                attackPatternKQL = 'source.as.number: ' + asn + ' OR destination.as.number: ' + asn;
                break;
            case 'btc':
                attackPattern = '[' + 'bitcoin:address = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'bitcoin.address: ' + '"' + v + '"';
                break;
            case "domain":
                attackPattern = '[' + 'dns:question:name = ' + '\'' + v + '\'' + ' OR url:domain = ' + '\'' + v + '\'' + ' OR source:domain = ' + '\'' + v + '\'' + ' OR destination:domain = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'dns.question.name: ' + '"' + v + '"' + ' OR url.domain: ' + '"' + v + '"' + ' OR source.domain: ' + '"' + v + '"' + ' OR destination.domain: ' + '"' + v + '"';
                break;
            case "domain|ip":
                arr = v.split("|");
                if (arr.length == 2) {
                    var domain = arr[0];
                    ip = arr[1].split("/")[0];
                    attackPattern = '[' + '(' + 'dns:question:name = ' + '\'' + domain + '\'' + ' OR url:domain = ' + '\'' + domain + '\'' + ')' +
                        ' AND ' + '(' + 'source:ip = ' + '\'' + ip + '\'' + ' OR destination:ip = ' + '\'' + ip + '\'' + ')' + ']';
                    attackPatternKQL = '(' + 'dns.question.name :' + '"' + domain + '"' + ' OR url.domain: ' + '"' + domain + '"' + ')' + ' AND ' + '(' + 'source.ip: ' + '"' + ip + '"' + ' OR destination.ip: ' + '"' + ip + '"' + ')';
                }
                break;
            case 'email-src':
                attackPattern = '[' + 'user:email = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'user.email: ' + '"' + v + '"';
                evt.Put("user.email", v);
                break;
            case "filename":
                attackPattern = '[' + 'file:path = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'file.path: ' + '"' + v + '"';
                evt.Put("file.path", v);
                break;
            case "filename|md5":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var md5 = arr[1];
                    attackPattern = '[' + 'file:hash:md5 = ' + '\'' + md5 + '\'' + ' AND file:path = ' + '\'' + filename + '\'' + ']';
                    attackPatternKQL = 'file.hash.md5: ' + '"' + md5 + '"' + ' AND file.path: ' + '"' + filename + '"';
                    evt.Put("file.hash.md5", md5);
                    evt.Put("file.path", filename);
                }
                break;
            case "filename|sha1":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var sha1 = arr[1];
                    attackPattern = '[' + 'file:hash:sha1 = ' + '\'' + sha1 + '\'' + ' AND file:path = ' + '\'' + filename + '\'' + ']';
                    attackPatternKQL = 'file.hash.sha1: ' + '"' + sha1 + '"' + ' AND file.path: ' + '"' + filename + '"';
                    evt.Put("file.hash.sha1", sha1);
                    evt.Put("file.path", filename);
                }
                break;
            case "filename|sha256":
                arr = v.split("|");
                if (arr.length == 2) {
                    filename = arr[0];
                    var sha256 = arr[1];
                    attackPattern = '[' + 'file:hash:sha256 = ' + '\'' + sha256 + '\'' + ' AND file:path = ' + '\'' + filename + '\'' + ']';
                    attackPatternKQL = 'file.hash.sha256: ' + '"' + sha256 + '"' + ' AND file.path: ' + '"' + filename + '"';
                    evt.Put("file.hash.sha256", sha256);
                    evt.Put("file.path", filename);
                }
                break;
            case 'github-username':
                attackPattern = '[' + 'user:name = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'user.name: ' + '"' + v + '"';
                evt.Put("user.name", v);
                break;
            case "hostname":
                attackPattern = '[' + 'source:domain = ' + '\'' + v + '\'' + ' OR destination:domain = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'source.domain: ' + '"' + v + '"' + ' OR destination.domain: ' + '"' + v + '"';
                break;
            case "ip-dst":
                ip = v.split("/")[0];
                attackPattern = '[destination:ip = ' + '\'' + ip + '\'' + ']';
                attackPatternKQL = 'destination.ip: ' + '"' + ip + '"';
                evt.Put("destination.ip", ip);
                break;
            case "ip-dst|port":
                arr = v.split("|");
                if (arr.length == 2) {
                  attackPattern = '[destination:ip = ' + '\'' + arr[0] + '\'' + ' AND destination:port = ' + '\'' + arr[1] + '\'' + ']';
                  attackPatternKQL = 'destination.ip: ' + '"' + arr[0] + '"' + ' AND destination.port: ' + arr[1];
                  evt.Put("destination.ip", arr[0]);
                  evt.Put("destination.port", arr[1]);
                }
                break;
            case "ip-src":
                ip = v.split("/")[0];
                attackPattern = '[' + 'source:ip = ' + '\'' + ip + '\'' + ']';
                attackPatternKQL = 'source.ip: ' + '"' + ip + '"';
                evt.Put("source.ip", ip);
                break;
            case "link":
                attackPattern = '[' + 'url:full = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'url.full: ' + '"' + v + '"';
                evt.Put("url.full", v);
                break;
            case "md5":
                attackPattern = '[' + 'file:hash:md5 = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'file.hash.md5: ' + '"' + v + '"';
                evt.Put("file.hash.md5", v);
                break;
            case 'regkey':
                attackPattern = '[' + 'regkey = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'regkey: ' + '"' + v + '"';
                evt.Put("registry.key", v);
                break;
            case "sha1":
                attackPattern = '[' + 'file:hash:sha1 = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'file.hash.sha1: ' + '"' + v + '"';
                evt.Put("file.hash.sha1", v);
                break;
            case "sha256":
                attackPattern = '[' + 'file:hash:sha256 = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'file.hash.sha256: ' + '"' + v + '"';
                evt.Put("file.hash.sha256", v);
                break;
            case "sha512":
                attackPattern = '[' + 'file:hash:sha512 = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'file.hash.sha512: ' + '"' + v + '"';
                evt.Put("file.hash.sha512", v);
                break;
            case "url":
                attackPattern = '[' + 'url:full = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'url.full: ' + '"' + v + '"';
                evt.Put("url.full", v);
                break;
            case 'yara':
                attackPattern = '[' + 'yara:rule = ' + '\'' + v + '\'' + ']';
                attackPatternKQL = 'yara.rule: ' + '"' + v + '"';
                break;
        }
        if (attackPattern == undefined || attackPatternKQL == undefined) {
            evt.Put("error.message", 'Unsupported type: ' + indicator_type);
        }
        evt.Put("misp.threat_indicator.attack_pattern", attackPattern);
        evt.Put("misp.threat_indicator.attack_pattern_kql", attackPatternKQL);
    };

    var pipeline = new processor.Chain()
        .Add(copyToOriginal)
        .Add(decodeJson)
        .Add(categorizeEvent)
        .Add(setThreatFeedField)
        .Add(convertFields)
        .Add(setID)
        .Add(setAttackPattern)
        .Add(copyTags)
        .Build();

    return {
        process: pipeline.Run,
    };
})();

function process(evt) {
    return threat.process(evt);
}
