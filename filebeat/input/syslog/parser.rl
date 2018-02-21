package main

%%{
    machine syslog;
    write data;
    variable p p;
    variable pe pe;
}%%

// syslog
//<34>Oct 11 22:14:15 mymachine su: 'su root' failed for lonvick on /dev/pts/8
//<13>Feb  5 17:32:18 10.0.0.99 Use the BFG!
func Parse(data []byte, syslog *SyslogMessage) {
    var p, cs int
    pe := len(data)
    %% write init;

    tok := 0
    eof := len(data)
    %%{
      action tok {
        tok = p
      }

      action priority {
        syslog.Priority = data[tok:p]
      }

      action message {
        syslog.Message = data[tok:p]
      }

      action month {
        syslog.Month(data[tok:p])
      }

      action day {
        syslog.Day(data[tok:p])
      }

      action hour {
        syslog.Hour(data[tok:p])
      }

      action minute {
        syslog.Minute(data[tok:p])
      }

      action second {
        syslog.Second(data[tok:p])
      }

      action hostname {
        syslog.Hostname = data[tok:p]
      }

      action program {
        syslog.Program = data[tok:p]
      }

      action pid {
        syslog.Pid = data[tok:p]
      }

      # General
      brackets = "[" | "]";

			# Priority
			# Ref: https://tools.ietf.org/html/rfc3164#section-4.1.1
      # Match: "<123>"
      priority = digit{1,5}>tok %priority;
      prio =  "<" priority ">";

			# Header
      # Timestamp
			# https://tools.ietf.org/html/rfc3164#section-4.1.2
			# Match: "Jan" and "January"
      month = ([Jj] "an" ("uary")? | [Ff] "eb" "ruary"? | [Mm] "ar" "ch"? | [Aa] "pr" "il"? | [Mm] "a" "y"? | [Jj] "un" "e"? | [Jj] "ul" "y"? | [Aa] "ug" "ust"? | [Ss] "ep" ("tember")? | [Oo] "ct" "ober"? | [Nn] "ov" "ember"? | [Dd] "ec" "ember"?) >tok %month;

			# Match: " 5" and "10" as the day
      multiple_digits_day = (([12][0-9]) | ("3"[01]))>tok %day;
      single_digit_day = [1-9]>tok %day;
      day = (space? single_digit_day | multiple_digits_day);

			# Match: hh:mm:ss (24 hr format)
      hour = ([01][0-9]|"2"[0-3])>tok %hour;
      minute = ([0-5][0-9])>tok %minute;
      second = ([0-5][0-9])>tok %second;
      time = hour ":" minute ":" second;
      timestamp_syslog = month space day space time;

      timestamp = timestamp_syslog;

			# TODO(ph): should we enforce ipv4, ipv6 or hostname? I tend to be more relax
      hostname = [a-zA-Z0-9.-_:]+>tok %hostname;
      header = timestamp space hostname space;

			# MSG
			# https://tools.ietf.org/html/rfc3164#section-4.1.3
      program = (extend -space -brackets)+>tok %program;
      pid = digit+>tok %pid;
      syslogprog = program ("[" pid "]")? ": ";
      message = any+>tok %message;
      msg = syslogprog? message>tok %message;

      main := prio header msg;
      write exec;
    }%%
}
