// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

%%{
    machine cef;

    # Define what header characters are allowed.
    pipe = "|";
    escape = "\\";
    escape_pipe = escape pipe;
    backslash = "\\\\";
    header_escapes = (backslash | escape_pipe) >mark_slash %mark_escape;
    device_chars = header_escapes | (any -- pipe -- escape);
    severity_chars = ( alpha | digit | "-" );

    # Header fields.
    version = "CEF:" digit+ >mark %version;
    device_vendor = device_chars*;
    device_product = device_chars*;
    device_version = device_chars*;
    device_event_class_id = device_chars*;
    name = device_chars*;
    severity = severity_chars*;

    # Define what extension characters are allowed.
    equal = "=";
    escape_equal = escape equal;
    escape_newline = escape 'n';
    escape_carriage_return = escape 'r';
    extension_value_escapes = (escape_equal | backslash | escape_newline | escape_carriage_return) >mark_slash %mark_escape;
    # Only alnum is defined in the CEF spec. The other characters allow
    # non-conforming extension keys to be parsed.
    extension_key_start_chars = alnum | '_';
    extension_key_chars = extension_key_start_chars | '.' | ',' | '[' | ']';
    extension_key_pattern = extension_key_start_chars extension_key_chars*;
    extension_value_chars_nospace = extension_value_escapes | (any -- equal -- escape -- space);

    # Extension fields.
    extension_key = extension_key_pattern >mark %extension_key;
    extension_value = (space* extension_value_chars_nospace @extension_value_mark)* >extension_value_start $err(extension_err);
    extension = extension_key equal extension_value;
    extensions = " "* extension (space* " " extension)* space* %/extension_eof;

    # gobble_extension attempts recovery from a malformed value by trying to
    # advance to the next extension key and re-entering the main state machine.
    gobble_extension := any* (" " >mark) extension_key_pattern equal @recover_next_extension;
}%%
