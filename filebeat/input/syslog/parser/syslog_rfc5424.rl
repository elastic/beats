%%{
  machine syslog_rfc5424;

  NONZERO_DIGIT = '1'..'9';

  # Syslog Message Format
  # https://tools.ietf.org/html/rfc5424#section-6

  # PRIVAL: 1*3DIGIT ; range 0 .. 191
  PRIVAL = (('1' ('9' ('0' | '1'){,1} | '0'..'8' ('0'..'9'){,1}){,1})| ('2'..'9' ('0'..'9'){,1}) | ('0'))>tok %priority;
  PRI =  "<" PRIVAL ">";

  VERSION = (NONZERO_DIGIT digit{0,2})>tok %version;

  message = any+>tok %message;

  main := PRI VERSION SP message;

}%%
