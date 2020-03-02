// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

function OktaSystem(keep_original_message) {
    var processor = require("processor");

    var decodeJson = new processor.DecodeJSONFields({
        fields: ["message"],
        target: "json",
    });

    var parseTimestamp = new processor.Timestamp({
        field: "json.published",
        timezone: "UTC",
        layouts: ["2006-01-02T15:04:05.999Z"],
        tests: ["2020-02-05T18:19:23.599Z"],
        ignore_missing: true,
    });

    var saveOriginalMessage = function(evt) {};
    if (keep_original_message) {
        saveOriginalMessage = new processor.Convert({
            fields: [
                {from: "message", to: "event.original"}
            ],
            mode: "rename"
        });
    }

    var dropOriginalMessage = function(evt) {
        evt.Delete("message");
    };

    var categorizeEvent = new processor.AddFields({
        target: "event",
        fields: {
            category: "authentication",
            kind: "event",
            type: "access",

        },
    });

    var convertFields = new processor.Convert({
        fields: [
            { from: "json.displayMessage", to: "okta.display_message" },
            { from: "json.eventType", to: "okta.event_type" },
            { from: "json.uuid", to: "okta.uuid" },
            { from: "json.actor.alternateId", to: "okta.actor.alternate_id" },
            { from: "json.actor.displayName", to: "okta.actor.display_name" },
            { from: "json.actor.id", to: "okta.actor.id" },
            { from: "json.actor.type", to: "okta.actor.type" },
            { from: "json.client.device", to: "okta.client.device" },
            { from: "json.client.geographicalContext.geolocation", to: "client.geo.location" },
            { from: "json.client.geographicalContext.city", to: "client.geo.city_name" },
            { from: "json.client.geographicalContext.state", to: "client.geo.region_name" },
            { from: "json.client.geographicalContext.country", to: "client.geo.country_name" },
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
            { from: "json.debugcontext.debugdata.devicefingerprint", to: "okta.debug_context.debug_data.device_fingerprint" },
            { from: "json.debugcontext.debugdata.requestid", to: "okta.debug_context.debug_data.request_id" },
            { from: "json.debugcontext.debugdata.requesturi", to: "okta.debug_context.debug_data.request_uri" },
            { from: "json.debugcontext.debugdata.threatsuspected", to: "okta.debug_context.debug_data.threat_suspected" },
            { from: "json.debugcontext.debugdata.url", to: "okta.debug_context.debug_data.url" },
            { from: "json.authenticationcontext.authenticationprovider", to: "okta.authentication_context.authentication_provider" },
            { from: "json.authenticationcontext.authenticationstep", to: "okta.authentication_context.authentication_step" },
            { from: "json.authenticationcontext.credentialprovider", to: "okta.authentication_context.credential_provider" },
            { from: "json.authenticationcontext.credentialtype", to: "okta.authentication_context.credential_type" },
            { from: "json.authenticationcontext.externalsessionid", to: "okta.authentication_context.external_session_id" },
            { from: "json.authenticationcontext.interface", to: "okta.authentication_context.authentication_provider" },
            { from: "json.authenticationcontext.issuer", to: "okta.authentication_context.issuer" },
            { from: "json.securitycontext.asnumber", to: "okta.security_context.as.number" },
            { from: "json.securitycontext.asorg", to: "okta.security_context.as.organization.name" },
            { from: "json.securitycontext.domain", to: "okta.security_context.domain" },
            { from: "json.securitycontext.isproxy", to: "okta.security_context.is_proxy" },
            { from: "json.securitycontext.isp", to: "okta.security_context.isp" },
        ],
        mode: "rename",
        ignore_missing: true,
    });

    var copyfields = new processor.convert({
        fields: [
            { from: "okta.client.user_agent.raw_user_agent", to: "user_agent.original" },
            { from: "okta.client.ip", to: "client.ip" },
            { from: "okta.client.ip", to: "source.ip" },
            { from: "okta.security_context.as", to: "client.as" },
            { from: "okta.security_context.domain", to: "client.domain" },
            { from: "okta.security_context.domain", to: "source.domain" },
            { from: "okta.outcome.result", to: "event.outcome" },
            { from: "okta.uuid", to: "event.id" },
            { from: "okta.event_type", to: "event.action" },
        ],
        fail_on_error: false,
    });
 
    // Update nested fields 
    var renameNestedFields = function(evt) {
        var arr = evt.Get("okta.target");
        if (arr != null) {
            for (var i = 0; i < arr.length; i++) {
                arr[i].alternate_id = arr[i].alternateId;
                arr[i].display_name = arr[i].displayName;
                delete arr[i].alternateId;
                delete arr[i].displayName;
                delete arr[i].detailEntry;
	    }
        }
    };

    // Set user info if actor type is User
    var setUserInfo = function(evt) {
        if (evt.Get("okta.actor.type") === "User") {
	    evt.Put("user.full_name", evt.Get("okta.actor.display_name"));
	    evt.Put("user.id", evt.Get("okta.actor.id"));
        }
    };


    // Drop extra fields
    var dropExtraFields = function(evt) {
        evt.Delete("json");
    };

    var pipeline = new processor.Chain()
        .Add(decodeJson)
        .Add(parseTimestamp)
        .Add(saveOriginalMessage)
        .Add(dropOriginalMessage)
        .Add(categorizeEvent)
        .Add(convertFields)
        .Add(copyFields)
        .Add(renameNestedFields)
        .Add(setUserInfo)
        .Add(dropExtraFields)
        .Build();

    return {
        process: pipeline.Run,
    };
};

var oktaSystem;

// Register params from configuration.
function register(params) {
    oktaSystem = new OktaSystem(params.keep_original_message);
}

function process(evt) {
    return oktaSystem.process(evt);
}
