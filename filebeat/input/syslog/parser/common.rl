%%{
  machine common;
  action tok {
    tok = p
  }

  action priority {
    event.SetPriority(data[tok:p])
  }

  action message {
    event.SetMessage(data[tok:p])
  }

  action month {
    event.SetMonth(data[tok:p])
  }

  action year{
    event.SetYear(data[tok:p])
  }

  action month_numeric {
    event.SetMonthNumeric(data[tok:p])
  }

  action day {
    event.SetDay(data[tok:p])
  }

  action hour {
    event.SetHour(data[tok:p])
  }

  action minute {
    event.SetMinute(data[tok:p])
  }

  action second {
    event.SetSecond(data[tok:p])
  }

  action nanosecond{
    event.SetNanosecond(data[tok:p])
  }


  action init_data{
    event.data = EventData{}
  }

  action init_sd_param{
    state.sd_value_bs = []int{}
  }

  action set_sd_param_name{
    state.sd_param_name = string(data[tok:p])
  }

  action set_sd_param_value{
    event.SetData(state.sd_id, state.sd_param_name, data, tok, p, state.sd_value_bs)
 }

  action set_sd_id{
    state.sd_id = string(data[tok:p])
    if _, ok := event.data[ state.sd_id ]; ok {
		fhold;
	} else {
		event.data[state.sd_id] = map[string]string{}
	}
  }

  action set_bs{
    state.sd_value_bs = append(state.sd_value_bs, p)
  }
  # NOTES: This allow to bail out of obvious non valid
  # hostname, this might not be ideal in all situation, but
  # when this happen we just go to the catch all case and at least
  # extract the message
  action lookahead_duplicates{
    if p-1 > 0 {
      for _, b := range noDuplicates {
        if data[p] == b && data[p-1] == b {
          p = tok -1
          fgoto catch_all;
        }
      }
    }
  }

  action hostname {
    event.SetHostname(data[tok:p])
  }

  action program {
    event.SetProgram(data[tok:p])
  }

  action pid {
    event.SetPid(data[tok:p])
  }

  action timezone {
    event.SetTimeZone(data[tok:p])
  }

  action sequence {
    event.SetSequence(data[tok:p])
  }

  action version{
    event.SetVersion(data[tok:p])
  }

  action app_name{
    event.SetAppName(data[tok:p])
  }

  action proc_id {
    event.SetProcID(data[tok:p])
  }

  action msg_id {
    event.SetMsgID(data[tok:p])
  }

  SP = ' ';

  # backslash "\"
  BS = 0x5C;

  NIL_VALUE             = "-";
  PRINT_US_ASCII        =  0x21..0x7E;
  NONZERO_DIGIT         = [1-9];

  # OCTET                 = 0x00..0xFF;
  OCTET                 = any;
  BOM                   = 0xEF 0xBB 0xBF;
  UTF_8_STRING          = OCTET*;

}%%
