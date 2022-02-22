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
        m.setVersion(data[tok:p])
    }

    action init_sd_data {
        m.structuredData = map[string]map[string]string{}
    }

    action init_sd_escapes {
        s.sdValueEscapes = nil
    }

    action set_param_name {
        s.sdParamName = data[tok:p]
    }

    action set_param_value {
        m.setDataValue(s.sdID, s.sdParamName, removeBytes(data[tok:p], s.sdValueEscapes, p))
    }

    action set_sd_id {
        s.sdID = data[tok:p]
        if _, ok := m.structuredData[s.sdID]; ok {
            err = ErrSDIDDuplicated
            fhold;
        } else {
            m.structuredData[s.sdID] = map[string]string{}
        }
    }

    action set_escape {
        s.sdValueEscapes = append(s.sdValueEscapes, p)
    }

    action err_version {
        err = ErrVersion
        fhold;
    }

    action err_app_name {
        err = ErrAppName
        fhold;
    }

    action err_proc_id {
        err = ErrProcID
        fhold;
    }

    action err_msg_id {
        err = ErrMsgID
        fhold;
    }

    action err_structured_data {
        err = ErrStructuredData
        fhold;
    }

    action err_sd_id {
        err = ErrSDID
        fhold;
    }

    action err_sd_param {
        err = ErrSDParam
        fhold;
    }

    nil_value = '-';

    version_range = ('1'..'9' . digit{0,2});
    version       = version_range >tok %set_version @err(err_version);

    escape_chars    = ('"' | ']' | bs);
    param_value_escape = (bs >set_escape escape_chars);
    sd_name         = (graph - ('=' | ']' | '"' | sp)){1,32};
    param_name      = sd_name >tok %set_param_name;
    param_value     = ((any - escape_chars) | param_value_escape)+ >tok %set_param_value;
    sd_param        = param_name '=' '"' param_value '"' >init_sd_escapes $err(err_sd_param);
    sd_id           = sd_name >tok %set_sd_id %err(err_sd_id) $err(err_sd_id);
    sd_element      = '[' sd_id (sp sd_param)* ']';
    structured_data = nil_value | sd_element+ >init_sd_data $err(err_structured_data);

    hostname_value = hostname_range >tok %set_hostname $err(err_hostname);
    hostname  = nil_value | hostname_value;

    app_name_range = graph{1,48};
    app_name_value = app_name_range >tok %set_app_name $err(err_app_name);
    app_name  = nil_value | app_name_value;

    proc_id_range = graph{1,128};
    proc_id_value = proc_id_range >tok %set_proc_id $err(err_proc_id);
    proc_id  = nil_value | proc_id_value;

    msg_id_range = graph{1,32};
    msg_id_value = msg_id_range >tok %set_msg_id $err(err_msg_id);
    msg_id  = nil_value | msg_id_value;

    timestamp = nil_value | timestamp_rfc3339;

    header    = pri version sp timestamp sp hostname sp app_name sp proc_id sp msg_id;

    msg = any* >tok %set_msg;
}%%
