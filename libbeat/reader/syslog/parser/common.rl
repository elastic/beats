%%{
    machine common;

    action tok {
        tok = p
    }

    action set_priority {
        if err := m.setPriority(data[tok:p]); err != nil {
            errs = multierr.Append(errs, &ValidationError{Err: err, Pos: tok+1})
        }
    }

    action set_timestamp_rfc3339 {
        if err := m.setTimestampRFC3339(data[tok:p]); err != nil {
            errs = multierr.Append(errs, &ValidationError{Err: err, Pos: tok+1})
        }
    }

    action set_timestamp_bsd {
        if err := m.setTimestampBSD(data[tok:p], loc); err != nil {
            errs = multierr.Append(errs, &ValidationError{Err: err, Pos: tok+1})
        }
    }

    action set_hostname {
        m.setHostname(data[tok:p])
    }

    action set_msg {
        m.setMsg(data[tok:p])
    }

    action set_process {
        m.setProcess(data[tok:p])
    }

    action set_pid {
        m.setPID(data[tok:p])
    }

    action err_eof {
        errs = multierr.Append(errs, &ParseError{Err: io.ErrUnexpectedEOF, Pos: p+1})
        fhold;
    }

    sp = ' ';  # space
    bs = 0x5C; # backslash

    priority_value    = graph+ >tok %set_priority;
    priority          = '<' priority_value '>';

    timestamp_bsd     = (alpha+ sp+ digit+ sp+ digit+ ':' digit+ ':' digit+) >tok %set_timestamp_bsd;
    timestamp_rfc3339  = (digit+ (alnum | [:.+\-]))+ >tok %set_timestamp_rfc3339;
}%%
