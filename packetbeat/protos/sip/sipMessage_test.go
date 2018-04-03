package sip

import (
    "testing"
    "fmt"

    "github.com/stretchr/testify/assert"

    "github.com/elastic/beats/libbeat/common"
)

func TestSeparatedStrings(t *testing.T) {
    msg := sipMessage{}
    var input_str string
    var separatedStrings *[]common.NetString

    input_str = "aaaa,bbbb,cccc,dddd"
    separatedStrings = msg.separateCsv(input_str)
    assert.Equal(t, "aaaa", fmt.Sprintf("%s",(*separatedStrings)[0]), "There should be [aaaa].")
    assert.Equal(t, "bbbb", fmt.Sprintf("%s",(*separatedStrings)[1]), "There should be [bbbb].")
    assert.Equal(t, "cccc", fmt.Sprintf("%s",(*separatedStrings)[2]), "There should be [cccc].")
    assert.Equal(t, "dddd", fmt.Sprintf("%s",(*separatedStrings)[3]), "There should be [dddd].")

    input_str = ",aaaa,\"bbbb,ccc\",dddd\\,eeee,\\\"ff,gg\\\","
    separatedStrings = msg.separateCsv(input_str)
    assert.Equal(t, ""            , fmt.Sprintf("%s",(*separatedStrings)[0]), "There should be blank.")
    assert.Equal(t, "aaaa"        , fmt.Sprintf("%s",(*separatedStrings)[1]), "There should be [aaaa].")
    assert.Equal(t, "\"bbbb,ccc\"", fmt.Sprintf("%s",(*separatedStrings)[2]), "There should be [\"bbbb,ccc\"].")
    assert.Equal(t, "dddd\\,eeee" , fmt.Sprintf("%s",(*separatedStrings)[3]), "There should be [dddd\\,eeee].")
    assert.Equal(t, "\\\"ff"      , fmt.Sprintf("%s",(*separatedStrings)[4]), "There should be [\\\"ff].")
    assert.Equal(t, "gg\\\""      , fmt.Sprintf("%s",(*separatedStrings)[5]), "There should be [gg\\\"].")
    assert.Equal(t, ""            , fmt.Sprintf("%s",(*separatedStrings)[6]), "There should be blank.")

    input_str = "aaaa,\"aaaaa,bbb"
    separatedStrings = msg.separateCsv(input_str)
    assert.Equal(t,"aaaa"       , fmt.Sprintf("%s",(*separatedStrings)[0]), "There should be [aaaa].")
    assert.Equal(t,"\"aaaaa,bbb", fmt.Sprintf("%s",(*separatedStrings)[1]), "There should be [\"aaaaa,bbb].")

    input_str = "aaaa,\"aaaaa,"
    separatedStrings = msg.separateCsv(input_str)
    assert.Equal(t,"aaaa"    , fmt.Sprintf("%s",(*separatedStrings)[0]), "There should be [aaaa].")
    assert.Equal(t,"\"aaaaa,", fmt.Sprintf("%s",(*separatedStrings)[1]), "There should be [\"aaaaa,].")
}

func TestParseSIPHeader(t *testing.T){ 
    var garbage []byte
    var err error
    var msg sipMessage
    // CRLF only messags
    garbage = []byte( "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n"  +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,"malformed packet", fmt.Sprintf("%s",err) , "There should be no error." )
    assert.Equal(t,-1                , msg.hdr_start         , "There should be no error." )
    assert.Equal(t,-1                , msg.hdr_len           , "There should be no error." )
    assert.Equal(t,-1                , msg.bdy_start         , "There should be no error." )
    assert.Equal(t,-1                , msg.contentlength     , "There should be no error." )

    // \r\n start and fragmented packet
    garbage = []byte( "\r\n"                        +
                      "\r\n"                        +
                      "\r\n"                        +
                      "\r\n"                        +
                      "SIP/2.0 200 OK\r\n"          +
                      "Via: testVia1,\r\n"          +
                      " testVia2, \r\n"             +
                      " testVia3,  testVia4\r\n"    +
                      "From: testFrom\r\n"          +
                      "To  \t :\t  testTo\t\t\r\n"  +
                      "Call-ID: testCall-ID\r\n"    +
                      "CSeq: testCSeq\r\n"          +
                      "Vi")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil            ,err ,"There should be no error." )
    assert.Equal(t, 8             ,msg.hdr_start     ,"There should be no error." )
    assert.Equal(t,len(garbage)-8 ,msg.hdr_len       ,"There should be no error." )
    assert.Equal(t,len(garbage)   ,msg.bdy_start     ,"There should be no error." )
    assert.Equal(t,0              ,msg.contentlength ,"There should be no error." )

    // no mandatory header
    garbage = []byte( "SIP/2.0 200 OK\r\n"          +
                      "Via: testVia1,\r\n"          +
                      " testVia2, \r\n"             +
                      " testVia3,  testVia4\r\n"    +
                      "Via: testVia5,testVia6\r\n"  +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil                     ,err        ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.to     ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.from   ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.cseq   ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.callid ,"There should be." )
    assert.Contains(t,msg.notes ,common.NetString("mandatory header [To] does not exist.")      ,"There should be contained." )
    assert.Contains(t,msg.notes ,common.NetString("mandatory header [From] does not exist.")    ,"There should be contained." )
    assert.Contains(t,msg.notes ,common.NetString("mandatory header [CSeq] does not exist.")    ,"There should be contained." )
    assert.Contains(t,msg.notes ,common.NetString("mandatory header [Call-ID] does not exist.") ,"There should be contained." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be  0." )
    assert.Equal(t, 89,msg.hdr_len        ,"There should be 89." )
    assert.Equal(t, 93,msg.bdy_start      ,"There should be 93." )
    assert.Equal(t,  0 ,msg.contentlength ,"There should be  0." ) 
    // status-line/request-line fault
    garbage = []byte( "HTTP/1.1 302 Found\r\n"                                       +
                      "Location: https://golang.org/\r\n"                            +
                      "Content-Type: text/html; charset=utf-8\r\n"                   +
                      "X-Cloud-Trace-Context: 8635c1565e2e6113d8600407750c9c4b\r\n"  +
                      "Date: Sun, 21 Jan 2018 07:32:51 GMT\r\n"                      +
                      "Server: Google Frontend\r\n"                                  +
                      "Content-Length: 42\r\n"                                       +
                      "\r\n"                                                         +
                      "<a href=\"https://golang.org/\">Found</a>.\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,"malformed packet(this is not sip message)", fmt.Sprintf("%s",err)   ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.to     ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.from   ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.cseq   ,"There should be." )
    assert.Equal(t,(common.NetString)(nil) ,msg.callid ,"There should be." )
    assert.Contains(t,msg.notes ,common.NetString("malformed packet. this is not sip message.")      ,"There should be contained." )

    // invalid status number(String)
    garbage = []byte( "SIP/2.0 200C NG\r\n"         +
                      "Via: testVia1\r\n"           +
                      "From: testFrom\r\n"          +
                      "To: testTo\t\t\r\n"          +
                      "Call-ID: testCall-ID\r\n"    +
                      "CSeq: testCSeq\r\n"          +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,uint16(999)                     ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("NG")          ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Contains(t,msg.notes ,common.NetString("invalid status-code 200C") ,"There should be contained." )

    // status phrase missing
    garbage = []byte( "SIP/2.0 200 \r\n"         +
                      "Via: testVia1\r\n"          +
                      "From: testFrom\r\n"          +
                      "To: testTo\t\t\r\n"  +
                      "Call-ID: testCall-ID\r\n"    +
                      "CSeq: testCSeq\r\n"          +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,uint16(200)                     ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("")            ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be  0." )
    assert.Equal(t, 95,msg.hdr_len        ,"There should be 95." )
    assert.Equal(t, 99,msg.bdy_start      ,"There should be 99." )
    assert.Equal(t,  0 ,msg.contentlength ,"There should be  0." )

    // status phrase missing (split error)
    garbage = []byte( "SIP/2.0 200\r\n"          +
                      "Via: testVia1\r\n"        +
                      "From: testFrom\r\n"       +
                      "To: testTo\t\t\r\n"       +
                      "Call-ID: testCall-ID\r\n" +
                      "CSeq: testCSeq\r\n"       +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,"malformed packet", fmt.Sprintf("%s",err)   ,"There should be." )
    assert.Contains(t,msg.notes ,common.NetString("start line parse error.") ,"There should be contained." )

    // Toomany SP at start line deliminater
    garbage = []byte( "SIP/2.0  183  Session Progress\r\n" +
                      "Via: testVia1,\r\n"               +
                      " testVia2, \r\n"                  +
                      " testVia3,  testVia4\r\n"         +
                      "From: testFrom\r\n"               +
                      "To  \t :\t  testTo\t\t\r\n"       +
                      "Call-ID: testCall-ID\r\n"         +
                      "CSeq: testCSeq\r\n"               +
                      "Via: testVia5,testVia6\r\n"       +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil                                        , err            ,"There should be no error." )
    assert.Equal(t,uint16(999)                               ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("183  Session Progress") ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be   0." )
    assert.Equal(t,179 ,msg.hdr_len       ,"There should be 179." )
    assert.Equal(t,183 ,msg.bdy_start     ,"There should be 183." )
    assert.Equal(t,  0 ,msg.contentlength ,"There should be   0." )

    // Toomany SP deliminater at start line
    garbage = []byte( "INVITE testRequstURI SIP/2.0\r\n" +
                      "Via: testVia1,\r\n"               +
                      " testVia2, \r\n"                  +
                      " testVia3,  testVia4\r\n"         +
                      "From: testFrom\r\n"               +
                      "To  \t :\t  testTo\t\t\r\n"       +
                      "Call-ID: testCall-ID\r\n"         +
                      "CSeq: testCSeq\r\n"               +
                      "Via: testVia5,testVia6\r\n"       +
                      "\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil                               , err             ,"There should be no error." )
    assert.Equal(t,uint16(0)                         ,msg.statusCode   ,"There should be nill." )
    assert.Equal(t,(common.NetString)(nil)           ,msg.statusPhrase ,"There should be nill." )
    assert.Equal(t,common.NetString("INVITE")        ,msg.method       ,"There should be INVITE." )
    assert.Equal(t,common.NetString("testRequstURI") ,msg.requestUri   ,"There should be testRequstURI." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be   0." )
    assert.Equal(t,177 ,msg.hdr_len       ,"There should be 177." )
    assert.Equal(t,181 ,msg.bdy_start     ,"There should be 181." )
    assert.Equal(t,  0 ,msg.contentlength ,"There should be   0." )

    // content-type and content-length missing
    garbage = []byte( "SIP/2.0 183 Session Progress\r\n" +
                      "Via: testVia1\r\n"                +
                      "From: testFrom\r\n"               +
                      "To:  testTo\t\t\r\n"              +
                      "Call-ID: testCall-ID\r\n"         +
                      "CSeq: testCSeq\r\n"               +
                      "\r\n"                             +
                      "v=0\r\n"                          +
                      "o=- 0 0 IN IP4 10.0.0.1\r\n"      +
                      "s=-\r\n"                          +
                      "c=IN IP4 10.0.0.1\r\n"            +
                      "t=0 0\r\n"                        +
                      "m=audio 5012 RTP/AVP 0\r\n"       +
                      "a=rtpmap:0 PCMU/8000\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil        , err                     , "There should be no error." )
    assert.Equal(t,uint16(183)                          ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("Session Progress") ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be   0." )
    assert.Equal(t,112 ,msg.hdr_len       ,"There should be 112." )
    assert.Equal(t,116 ,msg.bdy_start     ,"There should be 116." )
    assert.Equal(t,  0 ,msg.contentlength ,"There should be   0." )
    assert.Equal(t,(map[string]*map[string][]common.NetString)(nil) ,msg.body ,"There should be nill." )

    // content-length missing
    garbage = []byte( "SIP/2.0 183 Session Progress\r\n"  +
                      "Via: testVia1\r\n"                 +
                      "From: testFrom\r\n"                +
                      "To: testTo\t\t\r\n"                +
                      "Content-Type: application/sdp\r\n" +
                      "Call-ID: testCall-ID\r\n"          +
                      "CSeq: testCSeq\r\n"                +
                      "\r\n"                              +
                      "v=0\r\n"                           + //24-29  5 
                      "o=- 0 0 IN IP4 10.0.0.1\r\n"       + //24-49 25  30
                      "s=-\r\n"                           + //24-29  5  35
                      "c=IN IP4 10.0.0.1\r\n"             + //24-43 19  54
                      "t=0 0\r\n"                         + //24-31  7  61
                      "m=audio 5012 RTP/AVP 0\r\n"        + //24-48 24  85
                      "a=rtpmap:0 PCMU/8000\r\n")           //24-46 22 107
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil        , err                     , "There should be no error." )
    assert.Equal(t,uint16(183)                          ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("Session Progress") ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be   0." )
    assert.Equal(t,142 ,msg.hdr_len       ,"There should be 142." )
    assert.Equal(t,146 ,msg.bdy_start     ,"There should be 146." )
    assert.Equal(t,107 ,msg.contentlength ,"There should be 107." )
    assert.Equal(t,(map[string]*map[string][]common.NetString)(nil) ,msg.body ,"There should be nill." )

    // too large content-length actually byte length
    garbage = []byte( "SIP/2.0 183 Session Progress\r\n" +
                      "Via: testVia1,\r\n"               +
                      " testVia2, \r\n"                  +
                      " testVia3,  testVia4\r\n"         +
                      "From: testFrom\r\n"               +
                      "To  \t :\t  testTo\t\t\r\n"       +
                      "Call-ID: testCall-ID\r\n"         +
                      "CSeq: testCSeq\r\n"               +
                      "Via: testVia5,testVia6\r\n"       +
                      "Content-Type: application/sdp\r\n"+
                      "Content-length: 134\r\n"          +
                      "\r\n"                             +
                      "v=0\r\n"                          +
                      "o=- 0 0 IN IP4 10.0.0.1\r\n"      +
                      "s=-\r\n"                          +
                      "c=IN IP4 10.0.0.1\r\n"            +
                      "t=0 0\r\n"                        +
                      "m=audio 5012 RTP/AVP 0\r\n"       +
                      "a=rtpmap:0 PCMU/8000\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil        , err                     , "There should be no error." )
    assert.Equal(t,uint16(183)                          ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("Session Progress") ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be -1." )
    assert.Equal(t,229 ,msg.hdr_len       ,"There should be -1." )
    assert.Equal(t,233 ,msg.bdy_start     ,"There should be -1." )
    assert.Equal(t, -1 ,msg.contentlength ,"There should be -1." )
    assert.Equal(t,(map[string]*map[string][]common.NetString)(nil) ,msg.body ,"There should be nill." )

    // normal case response
    garbage = []byte( "SIP/2.0 183 Session Progress\r\n" +
                      "Via: testVia1,\r\n"               +
                      " testVia2, \r\n"                  +
                      " testVia3,  testVia4\r\n"         +
                      "From: testFrom\r\n"               +
                      "To  \t :\t  testTo\t\t\r\n"       +
                      "Call-ID: testCall-ID\r\n"         +
                      "CSeq: testCSeq\r\n"               +
                      "Via: testVia5,testVia6\r\n"       +
                      "Content-Type: application/sdp\r\n"+
                      "Content-length: 107\r\n"           +
                      "\r\n"                             +
                      "v=0\r\n"                          +
                      "o=- 0 0 IN IP4 10.0.0.1\r\n"      +
                      "s=-\r\n"                          +
                      "c=IN IP4 10.0.0.1\r\n"            +
                      "t=0 0\r\n"                        +
                      "m=audio 5012 RTP/AVP 0\r\n"       +
                      "a=rtpmap:0 PCMU/8000\r\n")
    msg = sipMessage{}
    msg.raw = garbage
    err = msg.parseSIPHeader()
    assert.Equal(t,nil        , err                     , "There should be no error." )
    assert.Equal(t,uint16(183)                          ,msg.statusCode   ,"There should be." )
    assert.Equal(t,common.NetString("Session Progress") ,msg.statusPhrase ,"There should be." )
    assert.Equal(t,common.NetString("testTo")      ,msg.to     ,"There should be." )
    assert.Equal(t,common.NetString("testFrom")    ,msg.from   ,"There should be." )
    assert.Equal(t,common.NetString("testCSeq")    ,msg.cseq   ,"There should be." )
    assert.Equal(t,common.NetString("testCall-ID") ,msg.callid ,"There should be." )
    assert.Equal(t,  0 ,msg.hdr_start     ,"There should be -1." )
    assert.Equal(t,229 ,msg.hdr_len       ,"There should be -1." )
    assert.Equal(t,233 ,msg.bdy_start     ,"There should be -1." )
    assert.Equal(t,107 ,msg.contentlength ,"There should be 107." )
    assert.Equal(t,(map[string]*map[string][]common.NetString)(nil) ,msg.body ,"There should be nill." )
}

func TestParseSIPHeaderToMap(t *testing.T){
    var garbage []byte
    firstline:="SIP/2.0 200 OK\r\n"
    header0  :="Via: testVia1,\r\n"
    header1  :=" testVia2, \r\n"
    header2  :=" testVia3,  testVia4\r\n"
    header3  :="From: testFrom\r\n"
    header4  :="To  \t :\t  testTo\t\t\r\n"
    header5  :="Call-ID: testCall-ID\r\n"
    header6  :="CSeq: testCSeq\r\n"
    header7  :="Via: testVia5,testVia6\r\n"
    garbage = []byte( firstline +
                      header0   +
                      header1   +
                      header2   +
                      header3   +
                      header4   +
                      header5   +
                      header6   +
                      header7   +
                      "\r\n")
    offset0:=0
    offset1:=offset0+len(firstline)
    offset2:=offset1+len(header0)
    offset3:=offset2+len(header1)
    offset4:=offset3+len(header2)
    offset5:=offset4+len(header3)
    offset6:=offset5+len(header4)
    offset7:=offset6+len(header5)
    offset8:=offset7+len(header6)
    cuts:=[]int{               offset0,                offset1,                offset2,
                               offset3,                offset4,                offset5,
                               offset6,                offset7,                offset8}
    cute:=[]int{      len(firstline)-2, offset1+len(header0)-2, offset2+len(header1)-2, 
                offset3+len(header2)-2, offset4+len(header3)-2, offset5+len(header4)-2,
                offset6+len(header5)-2, offset7+len(header6)-2, offset8+len(header7)-2}
    msg := sipMessage{}
    msg.raw = garbage
    headers, first_lines:=msg.parseSIPHeaderToMap(cuts,cute)

    assert.Equal(t,3        , len(first_lines)                 , "There should be." )
    assert.Equal(t,"SIP/2.0", fmt.Sprintf("%s",first_lines[0]) , "There should be." )
    assert.Equal(t,"200"    , fmt.Sprintf("%s",first_lines[1]) , "There should be." )
    assert.Equal(t,"OK"     , fmt.Sprintf("%s",first_lines[2]) , "There should be." )

    assert.Equal(t,5             , len(*headers)                              , "There should be." )
    assert.Equal(t,1             , len((*headers)["from"   ])                 , "There should be." )
    assert.Equal(t,1             , len((*headers)["to"     ])                 , "There should be." )
    assert.Equal(t,6             , len((*headers)["via"    ])                 , "There should be." )
    assert.Equal(t,1             , len((*headers)["cseq"   ])                 , "There should be." )
    assert.Equal(t,1             , len((*headers)["call-id"])                 , "There should be." )

    assert.Equal(t,"testFrom"    , fmt.Sprintf("%s",(*headers)["from"   ][0]) , "There should be." )
    assert.Equal(t,"testTo"      , fmt.Sprintf("%s",(*headers)["to"     ][0]) , "There should be." )
    assert.Equal(t,"testCSeq"    , fmt.Sprintf("%s",(*headers)["cseq"   ][0]) , "There should be." )
    assert.Equal(t,"testCall-ID" , fmt.Sprintf("%s",(*headers)["call-id"][0]) , "There should be." )
    assert.Equal(t,"testVia1"    , fmt.Sprintf("%s",(*headers)["via"    ][0]) , "There should be." )
    assert.Equal(t,"testVia2"    , fmt.Sprintf("%s",(*headers)["via"    ][1]) , "There should be." )
    assert.Equal(t,"testVia3"    , fmt.Sprintf("%s",(*headers)["via"    ][2]) , "There should be." )
    assert.Equal(t,"testVia4"    , fmt.Sprintf("%s",(*headers)["via"    ][3]) , "There should be." )
    assert.Equal(t,"testVia5"    , fmt.Sprintf("%s",(*headers)["via"    ][4]) , "There should be." )
    assert.Equal(t,"testVia6"    , fmt.Sprintf("%s",(*headers)["via"    ][5]) , "There should be." )
}
func TestParseSIPBody(t *testing.T) { 
    var err error
    var garbage []byte
    msg := sipMessage{}

    // check when msg.header == nil
    err=msg.parseSIPBody()
    assert.Equal(t,"headers is nill", fmt.Sprintf("%s",err), "headers should be nill.")

    // check when msg.contentlength == 0
    msg.headers = &map[string][]common.NetString{}
    err=msg.parseSIPBody()
    assert.Equal(t,nil, err, "shuld be no error")

    // check msg.header has not content-type header.
    msg.contentlength=30
    err=msg.parseSIPBody()
    assert.Equal(t,"no content-type header", fmt.Sprintf("%s",err), "header should not have content-type.")

    // check zero length content-type header array
    msg.headers = &map[string][]common.NetString{}
    (*msg.headers)["content-type"]=[]common.NetString{}
    err=msg.parseSIPBody()
    assert.Equal(t,nil, err, "shuld be no error")
    assert.Equal(t,0, len(msg.body), "shuld be no entity in msg.body")

    // check not supported content-type.
    // initialized
    msg = sipMessage{}
    msg.contentlength=30
    msg.headers = &map[string][]common.NetString{}
    array:=[]common.NetString{}
    array=(*msg.headers)["content-type"]
    array=append(array,common.NetString("application/unsupported"))
    (*msg.headers)["content-type"]=array
    err=msg.parseSIPBody()
    assert.Equal(t,"unsupported content-type", fmt.Sprintf("%s",err), "shuld be error")
    assert.Equal(t,"application/unsupported",fmt.Sprintf("%s",(*msg.headers)["content-type"][0]), "shuld hasve content-type")
    assert.Equal(t,0, len(msg.body), "shuld be no entity in msg.body")

    // check supported content-type, sdp.
    // initialized
    msg = sipMessage{}
    garbage = []byte( "v=0\r\n"                    +
                      "o=- 0 0 IN IP4 10.0.0.1\r\n"+
                      "s=-\r\n"                    +
                      "c=IN IP4 10.0.0.1\r\n"      +
                      "t=0 0\r\n"                  +
                      "m=audio 5012 RTP/AVP 0\r\n" +
                      "a=rtpmap:0 PCMU/8000\r\n")
    msg.headers = &map[string][]common.NetString{}
    array=(*msg.headers)["content-type"]
    array=append(array,common.NetString("application/sdp"))
    (*msg.headers)["content-type"]=array
    msg.raw=garbage
    msg.bdy_start=0
    msg.contentlength=len(garbage)
    err=msg.parseSIPBody()
    assert.Equal(t,nil, err, "shuld be no error")
    assert.Equal(t,1, len(msg.body), "shuld be one entity in msg.body")
}

func TestParseBody_SDP(t *testing.T) {
    var result  *map[string][]common.NetString
    var err     error
    var garbage []byte

    msg := sipMessage{}

    // nil
    result,err =msg.parseBody_SDP(garbage)
    assert.Equal(t,nil                    , err                                 , "error recived"    )
    assert.Equal(t,0                      , len(*result)                        , "There should be." )

    // malformed
    garbage = []byte( "\r\n123149afajbngohk;kdgj\r\najkavnaa:aaaa\r\n===a===")
    result,err =msg.parseBody_SDP(garbage)
    assert.Equal(t,nil                    , err                                 , "error recived"    )
    assert.Equal(t,1                      , len(*result)                        , "There should be." )
    assert.Equal(t,"==a==="               , fmt.Sprintf("%s",(*result)[""][0])  , "There should be." )

    garbage = []byte( "v=0\r\n"                         +
                      "o=- 0 0 IN IP4 10.0.0.1    \r\n" + // Trim spaces
                      "s=-\r\n"                         +
                      "c=IN IP4 10.0.0.1\r\n"           +
                      "t=0 0\r\n"                       +
                      "m=audio 5012 RTP/AVP 0 16\r\n"   +
                      "a=rtpmap:0 PCMU/8000\r\n"        + // Multiple
                      "a=rtpmap:16 G729/8000\r\n")

    result,err =msg.parseBody_SDP(garbage)
    assert.Equal(t,nil                    , err                                 , "error recived"    )

    assert.Equal(t,7                      , len(*result)                        , "There should be." )
    assert.Equal(t,1                      , len((*result)["v"])                 , "There should be." )
    assert.Equal(t,1                      , len((*result)["o"])                 , "There should be." )
    assert.Equal(t,1                      , len((*result)["c"])                 , "There should be." )
    assert.Equal(t,1                      , len((*result)["t"])                 , "There should be." )
    assert.Equal(t,2                      , len((*result)["a"])                 , "There should be." )
    assert.Equal(t,"0"                    , fmt.Sprintf("%s",(*result)["v"][0]) , "There should be." )
    assert.Equal(t,"- 0 0 IN IP4 10.0.0.1", fmt.Sprintf("%s",(*result)["o"][0]) , "There should be." )
    assert.Equal(t,"IN IP4 10.0.0.1"      , fmt.Sprintf("%s",(*result)["c"][0]) , "There should be." )
    assert.Equal(t,"0 0"                  , fmt.Sprintf("%s",(*result)["t"][0]) , "There should be." )
    assert.Equal(t,"rtpmap:0 PCMU/8000"   , fmt.Sprintf("%s",(*result)["a"][0]) , "There should be." )
    assert.Equal(t,"rtpmap:16 G729/8000"  , fmt.Sprintf("%s",(*result)["a"][1]) , "There should be." )
}

func TestGetMessageStatus(t *testing.T) {
    msg := sipMessage{}

    msg.hdr_start=30
    msg.hdr_len=-1
    msg.bdy_start=-1
    msg.contentlength=-1
    msg.isIncompletedHdrMsg=true
    msg.isIncompletedBdyMsg=false
    assert.Equal(t,SIP_STATUS_HEADER_RECEIVING    , msg.getMessageStatus()   , "There should be HEADER RECEIVING." )

    msg.hdr_start=30
    msg.hdr_len=50
    msg.bdy_start=54
    msg.contentlength=-1
    msg.isIncompletedHdrMsg=false
    msg.isIncompletedBdyMsg=true
    assert.Equal(t,SIP_STATUS_BODY_RECEIVING      , msg.getMessageStatus()   , "There should be BODY RECEIVING." )

    msg.hdr_start=30
    msg.hdr_len=50
    msg.bdy_start=54
    msg.contentlength=55
    msg.isIncompletedHdrMsg=false
    msg.isIncompletedBdyMsg=false
    assert.Equal(t,SIP_STATUS_RECEIVED      , msg.getMessageStatus()   , "There should be RECEIVED." )

    msg.hdr_start=30
    msg.hdr_len=50
    msg.bdy_start=54
    msg.contentlength=0
    msg.isIncompletedHdrMsg=false
    msg.isIncompletedBdyMsg=false
    assert.Equal(t,SIP_STATUS_RECEIVED      , msg.getMessageStatus()   , "There should be RECEIVED." )
}
