%%{
    # RFC 3164 Syslog Message Format.
    # https://tools.ietf.org/html/rfc3164

    machine rfc3164;

    action set_tag {
        m.setTag(data[tok:p])
    }

    action set_content {
        m.setContent(data[tok:p])
    }

    timestamp = (timestamp_rfc3339 | timestamp_bsd);
    hostname  = graph+ >tok %set_hostname;

    tag           = (print -- [ :\[])+ >tok %set_tag;
    content_value = digit+ >tok %set_content;
    content       = '[' content_value ']';
    msg           = (tag content? ':' sp)? any+ >tok %set_msg;
}%%
