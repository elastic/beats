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
    action extension_key {
        // A new extension key marks the end of the last extension value.
        if len(state.key) > 0 && state.valueStart <= mark - 1 {
            e.pushExtension(state.key, replaceEscapes(data[state.valueStart:mark-1], state.valueStart, state.escapes))
            state.reset()
        }
        state.key = data[mark:p]
    }
    action extension_value_start {
        state.valueStart = p;
        state.valueEnd = p
    }
    action extension_value_mark {
        state.valueEnd = p+1
    }
    action extension_eof {
        // Reaching the EOF marks the end of the final extension value.
        if len(state.key) > 0 && state.valueStart <= state.valueEnd {
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
}%%
