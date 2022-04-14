%%{
    machine common;

    action tok {
        tok = p
    }

    action set_priority {
        m.setPriority(data[tok:p])
    }

    action set_timestamp_rfc3339 {
        m.setTimestampRFC3339(data[tok:p])
    }

    action set_timestamp_bsd {
        m.setTimestampBSD(data[tok:p], loc)
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

    action err_priority {
        err = multierr.Append(ErrPriority, err)
        fhold;
    }

    action err_timestamp {
        err = multierr.Append(ErrTimestamp, err)
        fhold;
    }

    action err_hostname {
        err = multierr.Append(ErrHostname, err)
        fhold;
    }

    sp = ' ';  # space
    dq = '"';  # double quote
    bs = 0x5C; # backslash

    # Date/Time patterns
    year        = digit{4};
    month       = ('0' . '1'..'9' | '1' . '0'..'2');
    month_str   = ('Jan' | 'Feb' | 'Mar' | 'Apr' | 'May' | 'Jun' | 'Jul' | 'Aug' | 'Sep' | 'Oct' | 'Nov' | 'Dec');
    day         = ('0' . '1'..'9' | '1'..'2' . '0'..'9' | '3' . '0'..'1');
    day_nopad   = (sp . '1'..'9' | '1'..'2' . '0'..'9' | '3' . '0'..'1');
    hour        = ('0'..'1' . '0'..'9' | '2' . '0'..'3');
    minute      = ('0'..'5' . '0'..'9');
    second      = ('0'..'5' . '0'..'9');
    ts_hhmmss   = hour ':' minute ':' second;
    ts_yyyymmdd = year '-' month '-' day;
    ts_offset   = 'Z' | ('+' | '-') hour ':' minute;

    # Priority
    pri_range = ('1' ('9' ('0' | '1')? | '0'..'8' ('0'..'9')?)?) | ('2'..'9' ('0'..'9')?) | '0';
    pri       = ('<' pri_range >tok %set_priority '>') @err(err_priority);

    # Timestamp
    timestamp_rfc3339 = (ts_yyyymmdd 'T' ts_hhmmss ('.' digit{1,6})? ts_offset) >tok %set_timestamp_rfc3339 $err(err_timestamp);
    timestamp_bsd     = (month_str . sp . day_nopad . sp . ts_hhmmss) >tok %set_timestamp_bsd $err(err_timestamp);

    # Hostname
    hostname_range    = graph{1,255};
}%%
