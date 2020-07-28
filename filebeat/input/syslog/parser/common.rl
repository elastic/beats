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

  NIL_VALUE = "-";
  PRINT_US_ASCII =  0x21..0x7E;
  NONZERO_DIGIT = [1-9];

  # UTF8 rfc3629
  utf8tail = 0x80..0xBF;
  utf81 = 0x00..0x7F;
  utf82 = 0xC2..0xDF utf8tail;
  utf83 = 0xE0 0xA0..0xBF utf8tail | 0xE1..0xEC utf8tail{2} | 0xED 0x80..0x9F utf8tail | 0xEE..0xEF utf8tail{2};
  utf84 = 0xF0 0x90..0xBF utf8tail{2} | 0xF1..0xF3 utf8tail{3} | 0xF4 0x80..0x8F utf8tail{2};
  utf8char = utf81 | utf82 | utf83 | utf84;


}%%
