%%{
  machine syslog_rfc5424;

  NONZERO_DIGIT = '1'..'9';

  # Syslog Message Format
  # https://tools.ietf.org/html/rfc5424#section-6

  # comment
  NILVALUE = "-";

  # PRI:  range 0 .. 191
  PRIVAL  = (('1' ('9' ('0' | '1'){,1} | '0'..'8' ('0'..'9'){,1}){,1})| ('2'..'9' ('0'..'9'){,1}) | ('0'))>tok %priority;
  PRI     =  "<" PRIVAL ">";

  VERSION = (NONZERO_DIGIT digit{0,2})>tok %version;

  # timestamp
  DATE_FULLYEAR   = digit{4}>tok %year;
  DATE_MONTH      = (("0"[1-9]) | ("1"[0-2]))>tok %month_numeric;
  DATE_MDAY       = (([12][0-9]) | ("3"[01]))>tok %day;
  FULL_DATE       = DATE_FULLYEAR "-" DATE_MONTH "-" DATE_MDAY;

  TIME_HOUR       = ([01][0-9] | "2"[0-3])>tok %hour;
  TIME_MINUTE     = ([0-5][0-9])>tok %minute;
  TIME_SECOND     = ([0-5][0-9])>tok %second;
  TIME_SECFRAC    = '.' digit{1,6}>tok %nanosecond;
  TIME_NUMOFFSET  = ('+' | '-') TIME_HOUR ':' TIME_MINUTE;
  TIME_OFFSET     = 'Z' | TIME_NUMOFFSET >tok %timezone;
  PARTIAL_TIME    = TIME_HOUR ":" TIME_MINUTE ":" TIME_SECOND  TIME_SECFRAC?;
  FULL_TIME       = PARTIAL_TIME TIME_OFFSET;

  TIMESTAMP       = NILVALUE | (FULL_DATE "T" FULL_TIME);



  message = any+>tok %message;

  main := PRI VERSION SP TIMESTAMP "T" message;

}%%
