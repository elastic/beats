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
  month = ( "Jan" ("uary")? | "Feb" "ruary"? | "Mar" "ch"? | "Apr" "il"? | "Ma" "y"? | "Jun" "e"? | "Jul" "y"? | "Aug" "ust"? | "Sep" ("tember")? | "Oct" "ober"? | "Nov" "ember"? | "Dec" "ember"?) >tok %month;

  # Match: " 5" and "10" as the day
  multiple_digits_day = (([12][0-9]) | ("3"[01]))>tok %day;
  single_digit_day = [1-9]>tok %day;
  day = (space? single_digit_day | multiple_digits_day);

  # Match: hh:mm:ss (24 hr format)
  hour = ([01][0-9]|"2"[0-3])>tok %hour;
  minute = ([0-5][0-9])>tok %minute;
  second = ([0-5][0-9])>tok %second;
  nanosecond = digit+>tok %nanosecond;
  time = hour ":" minute ":" second ("." nanosecond)?;
  offset_marker = "Z" | "z";
  offset_direction = "-" | "+";
  offset_hour = digit{2};
  offset_minute = digit{2};
  timezone = (offset_marker | offset_marker? offset_direction offset_hour (":"? offset_minute)?)>tok %timezone;

  # Some BSD style actually uses rfc3339 formatted date.
  year = digit{4}>tok %year;
  month_numeric = digit{2}>tok %month_numeric;
  day_two_digits = ([0-3][0-9])>tok %day;

  # common timestamp format
  timestamp_rfc3164 = month space day space time;
  time_separator = "T" | "t";
  timestamp_rfc3339 = year "-" month_numeric "-" day_two_digits (time_separator | space) time timezone?;
  timestamp = (timestamp_rfc3339 | timestamp_rfc3164) ":"?;

  hostname = ([a-zA-Z0-9\.\-_:]*([a-zA-Z0-9] | "::"))+>tok $lookahead_duplicates %hostname;
  hostVars = (hostname ":") | hostname;
  header = timestamp space hostVars ":"? space;

  # MSG
  # https://tools.ietf.org/html/rfc3164#section-4.1.3
  program = (extend -space -brackets)+>tok %program;
  pid = digit+>tok %pid;
  syslogprog = program ("[" pid "]")? ":" space;
  message = any+>tok %message;
  msg = syslogprog? message>tok %message;
  sequence = digit+ ":" space>tok %sequence;

  main := (prio)?(sequence)? (header msg | timestamp space message | message);
  catch_all := message;

}%%
