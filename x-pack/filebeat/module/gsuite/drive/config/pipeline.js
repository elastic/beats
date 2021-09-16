// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var drive = (function () {
    var path = require("path");
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.category", ["file"]);
        switch (evt.Get("event.action")) {
            case "add_to_folder":
            case "edit":
            case "add_lock":
            case "move":
            case "remove_from_folder":
            case "rename":
            case "remove_lock":
            case "sheets_import_range":
                evt.Put("event.type", ["change"]);
                break;
            case "approval_canceled":
            case "approval_comment_added":
            case "approval_requested":
            case "approval_reviewer_responded":
            case "change_acl_editors":
            case "change_document_access_scope":
            case "change_document_visibility":
            case "shared_drive_membership_change":
            case "shared_drive_settings_change":
            case "sheets_import_range_access_change":
            case "change_user_access":
                evt.AppendTo("event.category", "iam");
                evt.AppendTo("event.category", "configuration");
                evt.Put("event.type", ["change"]);
                break;
            case "create":
            case "untrash":
            case "upload":
                evt.Put("event.type", ["creation"]);
                break;
            case "delete":
            case "trash":
                evt.Put("event.type", ["deletion"]);
                break;
            case "download":
            case "preview":
            case "print":
            case "view":
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
        if (param.boolValue !== null) {
            return param.boolValue;
        }
    };

    var flattenParams = function(evt) {
        var params = evt.Get("json.events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        params.forEach(function(p){
            evt.Put("gsuite.drive."+p.name, getParamValue(p));
        });

        evt.Delete("json.events.parameters");
    };

    var setFileInfo = function(evt) {
        var type = evt.Get("gsuite.drive.file.type");
        if (!type) {
            return;
        }

        switch (type) {
            case "folder":
            case "shared_drive":
                evt.Put("file.type", "dir");
                break;
            default:
                evt.Put("file.type", "file");
        }

        // path returns extensions with a preceding ., e.g.: .tmp, .png
        // according to ecs the expected format is without it, so we need to remove it.
        var ext = path.extname(evt.Get("file.name"));
        if (!ext) {
            return;
        }

        if (ext.charAt(0) === ".") {
            ext = ext.substr(1);
        }
        evt.Put("file.extension", ext);
    };

    var setOwnerInfo = function(evt) {
        var email = evt.Get("gsuite.drive.file.owner.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.Put("file.owner", data[0]);
        evt.AppendTo("related.user", data[0]);
    };

    var setTargetRelatedUser = function(evt) {
        var email = evt.Get("gsuite.drive.target");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.AppendTo("related.user", data[0]);
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(flattenParams)
        .Convert({
            fields: [
                {
                    from: "gsuite.drive.doc_id",
                    to: "gsuite.drive.file.id",
                },
                {
                    from: "gsuite.drive.doc_title",
                    to: "file.name",
                },
                {
                    from: "gsuite.drive.doc_type",
                    to: "gsuite.drive.file.type",
                },
                {
                    from: "gsuite.drive.owner",
                    to: "gsuite.drive.file.owner.email",
                },
                {
                    from: "gsuite.drive.owner_is_shared_drive",
                    to: "gsuite.drive.file.owner.is_shared_drive",
                },
                {
                    from: "gsuite.drive.new_settings_state",
                    to: "gsuite.drive.new_value",
                },
                {
                    from: "gsuite.drive.old_settings_state",
                    to: "gsuite.drive.old_value",
                },
                {
                    from: "gsuite.drive.target_user",
                    to: "gsuite.drive.target",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setFileInfo)
        .Add(setOwnerInfo)
        .Add(setTargetRelatedUser)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return drive.process(evt);
}
