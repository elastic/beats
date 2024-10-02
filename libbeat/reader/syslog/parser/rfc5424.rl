%%{
    # RFC 5424 Syslog Message Format.
    # https://tools.ietf.org/html/rfc5424

    machine rfc5424;

    action set_proc_id {
        m.setProcID(data[tok:p])
    }

    action set_app_name {
        m.setAppName(data[tok:p])
    }

    action set_msg_id {
        m.setMsgID(data[tok:p])
    }

    action set_version {
        if err := m.setVersion(data[tok:p]); err != nil {
            errs = multierr.Append(errs, &ValidationError{Err: err, Pos: tok+1})
        }
    }

    action set_sd_raw {
        m.setRawSDValue(data[tok:p])
    }

    action init_sd_escapes {
        s.sdValueEscapes = nil
    }

    action set_param_name {
        s.sdParamName = data[tok:p]
    }

    action set_param_value {
        if subMap, ok := structuredData[s.sdID].(map[string]interface{}); ok {
            subMap[s.sdParamName] = removeBytes(data[tok:p], s.sdValueEscapes, tok)
        }
    }

    action set_sd_id {
        s.sdID = data[tok:p]
        if _, ok := structuredData[s.sdID]; !ok {
            structuredData[s.sdID] = map[string]interface{}{}
        }
    }

    action set_escape {
        s.sdValueEscapes = append(s.sdValueEscapes, p)
    }

    nil_value = '-';

    version = graph+ > tok %set_version;

    escape_chars    = ('"' | ']' | bs);
    param_value_escape = (bs >set_escape escape_chars);
    sd_name         = (graph - ('=' | ']' | '"' | sp)){1,32};
    param_name      = sd_name >tok %set_param_name;
    param_value     = ((any - escape_chars) | param_value_escape)+ >tok %set_param_value;
    sd_param        = param_name '=' '"' param_value '"' >init_sd_escapes;
    sd_id           = sd_name >tok %set_sd_id;
    sd_element      = '[' sd_id (sp sd_param)* ']';
    structured_data = sd_element+;
    timestamp = nil_value | timestamp_rfc3339;

    hostname = nil_value | graph+ >tok %set_hostname;
    app_name = nil_value | graph+ >tok %set_app_name;
    proc_id  = nil_value | graph+ >tok %set_proc_id;
    msg_id   = nil_value | graph+ >tok %set_msg_id;

    header = priority version sp timestamp sp hostname sp app_name sp proc_id sp msg_id;

    sd_raw_escape = (bs | ']');
    sd_raw_values = ((bs ']') | (any - sd_raw_escape));
    sd_raw        = nil_value | ('[' sd_raw_values+ ']')+ >tok %set_sd_raw;

    msg = any* >tok %set_msg;
}%%
