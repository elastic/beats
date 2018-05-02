%%{
  machine syslog_rfc3164;

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
  month = ( "Jan" ("uary")? | "Feb" "ruary"? | "Mar" "ch"? | "Apr" "il"? | "Ma" "y"? | "Jun" "e"? | "Jul" "y"? | "Aug" "ust"? | "Sep" ("tember")? | "Oct" "ober"? | "Nov" "ember"? | "ec" "ember"?) >tok %month;

  # Match: " 5" and "10" as the day
  multiple_digits_day = (([12][0-9]) | ("3"[01]))>tok %day;
  single_digit_day = [1-9]>tok %day;
  day = (space? single_digit_day | multiple_digits_day);

  # Match: hh:mm:ss (24 hr format)
  hour = ([01][0-9]|"2"[0-3])>tok %hour;
  minute = ([0-5][0-9])>tok %minute;
  second = ([0-5][0-9])>tok %second;
  nanosecond = digit+;
  time = hour ":" minute ":" second ("." nanosecond)?;
  timestamp = month space day space time;

  hostname = [a-zA-Z0-9.-_:]+>tok %hostname;
  header = timestamp space hostname space;

  # MSG
  # https://tools.ietf.org/html/rfc3164#section-4.1.3
  program = (extend -space -brackets)+>tok %program;
  pid = digit+>tok %pid;
  syslogprog = program ("[" pid "]")? ":" space;
  message = any+>tok %message;
  msg = syslogprog? message>tok %message;

  main := (prio)? (header msg | timestamp space message | message);

}%%
