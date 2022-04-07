// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

%%{
    machine cef;

    # Actions to execute while executing state machine.
    action mark {
        mark = p
    }
    action mark_slash {
        mark_slash = p
    }
    action mark_escape {
        state.pushEscape(mark_slash, p)
    }
    action version {
        e.Version, _ = strconv.Atoi(data[mark:p])
    }
    action device_vendor {
        e.DeviceVendor = replaceEscapes(data[mark:p], mark, state.escapes)
        state.reset()
    }
    action device_product {
        e.DeviceProduct = replaceEscapes(data[mark:p], mark, state.escapes)
        state.reset()
    }
    action device_version {
        e.DeviceVersion = replaceEscapes(data[mark:p], mark, state.escapes)
        state.reset()
    }
    action device_event_class_id {
        e.DeviceEventClassID = replaceEscapes(data[mark:p], mark, state.escapes)
        state.reset()
    }
    action name {
        e.Name = replaceEscapes(data[mark:p], mark, state.escapes)
        state.reset()
    }
    action severity {
        e.Severity = data[mark:p]
    }
    action complete_header {
        complete = true
    }
    action incomplete_header {
        mark = p
        state.reset()
    }
    action extension_key {
        // A new extension key marks the end of the last extension value.
        if len(state.key) != 0 && state.valueStart < mark {
            // We should not be here, but purge the escapes and handle them.
            e.pushExtension(state.key, replaceEscapes(data[state.valueStart:mark-1], state.valueStart, state.escapes))
            state.reset()
        }
        state.key = data[mark:p]
    }
    action extension_value_start {
        if len(state.escapes) != 0 { // See ragel comment below.
            e.pushExtension(state.key, replaceEscapes(data[state.valueStart:state.valueEnd], state.valueStart, state.escapes))
            state.reset()
        }
        state.valueStart = p;
        state.valueEnd = p
    }
    action extension_value_mark {
        state.valueEnd = p+1
    }
    action extension_eof {
        // Reaching the EOF marks the end of the final extension value.
        if len(state.key) != 0 && state.valueStart < state.valueEnd {
            e.pushExtension(state.key, replaceEscapes(data[state.valueStart:state.valueEnd], state.valueStart, state.escapes))
            state.reset()
        }
    }
    action extension_err {
        recoveredErrs = append(recoveredErrs, fmt.Errorf("malformed value for %s at pos %d", state.key, p+1))
        fhold; fnext gobble_extension;
    }
    action recover_next_extension {
        state.reset()
        // Resume processing at p, the start of the next extension key.
        p = mark;
        fnext extensions;
    }

    # Note for extension_value_start
    #
    # There is a conditional execution there that depends only on the length of state.escapes.
    # This is explained by the following:
    #
    # // If we are here we must have been in a syntactically incorrect extension that failed to
    # // satisfy the machine and so did not trigger extension_eof and consume the escapes.
    # // This consumes them so we can move on.
    #
    # This comment is placed here because it causes confusion to go tool cover if it is placed
    # in the expected location due to //line directives emitted by ragel.
    # See https://go.dev/issue/35781 for a related issue. When that is resolved the Go comment
    # section of this should be moved to the line following `if len(state.escapes) != 0 {` in
    # extension_value_start and the remainder of this comment can then be deleted.
}%%
