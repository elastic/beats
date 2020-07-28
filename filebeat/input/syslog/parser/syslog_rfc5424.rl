%%{
  machine syslog_rfc5424;

  NONZERO_DIGIT = '1'..'9';

  # Syslog Message Format
  # https://tools.ietf.org/html/rfc5424#section-6

  # comment
  NIL_VALUE = "-";
  PRINT_US_ASCII =  0x21..0x7E;

  # PRI:  range 0 .. 191
  PRIVAL  = (('1' ('9' ('0' | '1'){,1}
             | '0'..'8' ('0'..'9'){,1}){,1})
             | ('2'..'9' ('0'..'9'){,1})
             | ('0'))   >tok %priority;
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
  TIME_NUMOFFSET  = ('+' | '-') ([0-5][0-9]) ':' ([0-5][0-9]);
  TIME_OFFSET     = 'Z' | TIME_NUMOFFSET >tok %timezone;
  PARTIAL_TIME    = TIME_HOUR ":" TIME_MINUTE ":" TIME_SECOND  TIME_SECFRAC?;
  FULL_TIME       = PARTIAL_TIME TIME_OFFSET;

  TIMESTAMP       = NIL_VALUE | (FULL_DATE "T" FULL_TIME);

  HOSTNAME      = NIL_VALUE | PRINT_US_ASCII{1,255}     >tok %hostname;
  APP_NAME      = NIL_VALUE | PRINT_US_ASCII{1,48}      >tok %app_name;
  PROCID        = NIL_VALUE | PRINT_US_ASCII{1,128}     >tok %proc_id;
  MSGID         = NIL_VALUE | PRINT_US_ASCII{1,32}      >tok %msg_id;



  message = any+>tok %message;
  HEADER          = PRI VERSION SP TIMESTAMP SP HOSTNAME
                    SP APP_NAME SP PROCID SP MSGID;
  main := HEADER SP message;

}%%
