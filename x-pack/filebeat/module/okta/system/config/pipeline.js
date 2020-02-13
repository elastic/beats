// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var threat = (function () {
    var processor = require("processor");

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    var convertFields = new processor.Convert({
        fields: [
            { from: "json.displayMessage", to: "okta.display_message" },
            { from: "json.eventType", to: "okta.event_type" },
            { from: "json.published", to: "@timestamp" },
            { from: "json.uuid", to: "okta.uuid" },
            { from: "json.actor.alternateId", to: "okta.actor.alternate_id" },
            { from: "json.actor.displayName", to: "okta.actor.display_name" },
            { from: "json.actor.id", to: "okta.actor.id" },
            { from: "json.actor.type", to: "okta.actor.type" },
            { from: "json.client.device", to: "okta.client.device" },
            { from: "json.client.id", to: "okta.client.id" },
            { from: "json.client.ipAddress", to: "okta.client.ip" },
            { from: "json.client.userAgent.browser", to: "okta.client.user_agent.browser" },
            { from: "json.client.userAgent.os", to: "okta.client.user_agent.os" },
            { from: "json.client.userAgent.rawUserAgent", to: "okta.client.user_agent.raw_user_agent" },
            { from: "json.client.zone", to: "okta.client.zone" },
            { from: "json.outcome.reason", to: "okta.outcome.reason" },
            { from: "json.outcome.result", to: "okta.outcome.result" },
            { from: "json.target", to: "okta.target" },
            { from: "json.transaction.id", to: "okta.transaction.id" },
            { from: "json.transaction.type", to: "okta.transaction.type" },
            { from: "json.debugContext.debugData.deviceFingerprint", to: "okta.debug_context.debug_data.device_fingerprint" },
            { from: "json.debugContext.debugData.requestId", to: "okta.debug_context.debug_data.request_id" },
            { from: "json.debugContext.debugData.requestUri", to: "okta.debug_context.debug_data.request_uri" },
            { from: "json.debugContext.debugData.threatSuspected", to: "okta.debug_context.debug_data.threat_suspected" },
            { from: "json.debugContext.debugData.url", to: "okta.debug_context.debug_data.url" },
            { from: "json.authenticationContext.authenticationProvider", to: "okta.authentication_context.authentication_provider" },
            { from: "json.authenticationContext.credentialProvider", to: "okta.authentication_context.credential_provider" },
            { from: "json.authenticationContext.credentialType", to: "okta.authentication_context.credential_type" },
            { from: "json.authenticationContext.issuer", to: "okta.authentication_context.issuer" },
            { from: "json.authenticationContext.interface", to: "okta.authentication_context.authentication_provider" },
            { from: "json.authenticationContext.authenticationStep", to: "okta.authentication_context.authentication_step" },
            { from: "json.authenticationContext.externalSessionId", to: "okta.authentication_context.external_session_id" },
            { from: "json.securityContext.asNumber", to: "okta.security_context.as.number" },
            { from: "json.securityContext.asOrg", to: "okta.security_context.as.organization.name" },
            { from: "json.securityContext.isp", to: "okta.security_context.isp" },
            { from: "json.securityContext.domain", to: "okta.security_context.domain" },
            { from: "json.securityContext.isProxy", to: "okta.security_context.is_proxy" },
        ],
        mode: "rename",
        ignore_missing: true,
    });

    var copyFields = new processor.Convert({
        fields: [
            { from: "okta.client.user_agent.raw_user_agent", to: "user_agent.original" },
            { from: "okta.client.geographical_context.geolocation", to: "client.geo.location" },
            { from: "okta.client.geographical_context.city", to: "client.geo.city_name" },
            { from: "okta.client.geographical_context.state", to: "client.geo.region_name" },
            { from: "okta.client.geographical_context.country", to: "client.geo.country_name" },
            { from: "okta.client.ip", to: "client.ip" },
            { from: "okta.client.ip", to: "source.ip" },
            { from: "okta.security_context.as", to: "client.as" },
            { from: "okta.security_context.domain", to: "client.domain" },
            { from: "okta.security_context.domain", to: "source.domain" },
        ],
        fail_on_error: false,
    });
 
    // Drop extra fields
    var dropExtraFields = function(evt) {
        evt.Delete("json");
        evt.Delete("googlecloud.audit.request_metadata.requestAttributes");
        evt.Delete("googlecloud.audit.request_metadata.destinationAttributes");
    };

    // Update nested fields 
    var RenameNestedFields = function(evt) {
        var arr = evt.Get("okta.target");
        for (var i = 0; i < arr.length; i++) {
            arr[i].alternate_id = arr[i].alternateId;
            arr[i].display_name = arr[i].displayName;
            delete arr[i].alternateId;
            delete arr[i].displayName;
            delete arr[i].detailEntry;
        }
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
        .Add(decodeJson)
        .Add(categorizeEvent)
        .Add(setThreatFeedField)
        .Add(convertFields)
        .Add(setAttackPattern)
        .Build();

    return {
        process: pipeline.Run,
    };
})();

function process(evt) {
    return threat.process(evt);
}
