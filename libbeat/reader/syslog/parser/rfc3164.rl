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
    hostname  = hostname_range >tok %set_hostname $err(err_hostname);

    tag           = (print -- [ :\[]){1,32} >tok %set_tag;
    content_value = print+ >tok %set_content;
    content       = '[' content_value ']';
    msg           = (tag content? ':' sp)? any+ >tok %set_msg;
}%%
