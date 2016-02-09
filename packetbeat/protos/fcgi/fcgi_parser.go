package fcgi

    import  (   "time" 
//                "github.com/elastic/beats/libbeat/logp"
//                "github.com/elastic/beats/packetbeat/protos/tcp"    // tcp.TcpDirectionOriginal n R
//                "encoding/hex"                                      // Dump object
            )

type fcgiParseState uint8

const (
    FCGI_START_PARSE fcgiParseState = iota  // initial stream state
    FCGI_AWT_RECORD_DATA                    // waiting record data to finish
)
type fcgiProtoAppRecord uint8

const (
    FCGI_BEGIN_REQUEST uint32 = iota + 1
    FCGI_ABORT_REQUEST
    FCGI_END_REQUEST
    FCGI_PARAMS
    FCGI_STDIN 
    FCGI_STDOUT
    FCGI_STDERR
    FCGI_DATA
    FCGI_GET_VALUES
    FCGI_GET_VALUES_RESULT
    FCGI_UNKNOWN_TYPE
)
var fcgiProtoAppRecordNames = []string{
    " ",
    "fcgi_begin_request",
    "fcgi_abort_request",
    "fcgi_end_request",
    "fcgi_params",
    "fcgi_stdin",
    "fcgi_stdout",
    "fcgi_stderr",
    "fcgi_data",
    "fcgi_get_values",
    "fcgi_get_values_result",
    "fcgi_unknown_type",
}


    type message struct {
       
        // Net info
        Ts               time.Time
        Direction    uint8

        // FCGI record info
        recordType          uint32
        recordContentLength uint32

        //headerOffset     int
        //bodyOffset       int
        //version          version
        //connection       common.NetString
        // chunkedLength    int
        // chunkedBody      []byte
        // IsRequest    bool
        //TCPTuple     common.TcpTuple
        //CmdlineTuple *common.CmdlineTuple
        //Request Info
        //RequestURI   common.NetString
        //Method       common.NetString
        //StatusCode   uint16
        //StatusPhrase common.NetString
        //RealIP       common.NetString

        // Http Headers
        //ContentLength    int
        //ContentType      common.NetString
        //TransferEncoding common.NetString
        //Headers          map[string]common.NetString
        //Size             uint64

        //Raw Data
        Raw []byte

        //Notes []string

        //Timing
        //start int
        //end   int

        // Data structure pointer
        next *message
    }

    // called by Parse() on every new message

    func tryGetRecord(priv_fcgiData *fcgiData, dir uint8) (*fcgiData,bool) {

        // Working data, bytes already parsed
        recCount        := 0
        newRecordExists := false        
        data            := priv_fcgiData.Streams[dir].data;
        parseOffset     := priv_fcgiData.Streams[dir].parseOffset
        parseState      := priv_fcgiData.Streams[dir].parseState
        datasize        := uint32(len(data))


        

        //logp.Info("protos.fcgi.fcgi_parser: datasize: %d parseOffset: %d moving: %d bytes",datasize,parseOffset,(datasize-parseOffset) ) 
        //priv_fcgiData.Streams[dir].parseOffset = datasize
        if parseState == FCGI_START_PARSE {

            stop := false
            // At least we need a whole record
            for !stop && parseOffset + 8 <= datasize {
                // read 8 bytes
                recordVersion           := uint32(data[parseOffset + 0])
                recordType              := uint32(data[parseOffset + 1])
                //recordRequestId         := uint32(data[parseOffset + 2]) << 8 + uint32(data[parseOffset + 3])
                recordContentLength     := uint32(data[parseOffset + 4]) << 8 + uint32(data[parseOffset + 5])       
                recordPaddingLength     := uint32(data[parseOffset + 6])
                recordReservedByte      := uint32(data[parseOffset + 7])

                parseOffset += 8

                // Error or bogus message check and data ignore if so

                if ( recordVersion > 1 || recordReservedByte != 0 || recordType > FCGI_UNKNOWN_TYPE ){
                    parseOffset = datasize
                }else{

                        // to read Data      gt  what i have to read for this record
                    if datasize - parseOffset >= recordContentLength + recordPaddingLength {
                        //directionStringPrint := "<" // Asume reverse 
                        //if( dir == tcp.TcpDirectionOriginal ) { directionStringPrint = ">" } 
                        //logp.Info("protos.fcgi.fcgi_parser: Parsing record type: %s(%d) id=(%d) %s (parseOffset:%d)", 
                        //        fcgiProtoAppRecordNames[recordType],recordType,recordRequestId,directionStringPrint,parseOffset)
                        newRecordExists = true;
                        recCount++;
                        if( priv_fcgiData.ParsedRecords.head == nil ){
                            priv_fcgiData.ParsedRecords.head = &message{recordType: recordType, next: nil}
                            priv_fcgiData.ParsedRecords.tail = priv_fcgiData.ParsedRecords.head
                        }else {
                            priv_fcgiData.ParsedRecords.tail.next = &message{   recordType: recordType, 
                                                                                next: nil,
                                                                                Direction: dir}
                            priv_fcgiData.ParsedRecords.tail = priv_fcgiData.ParsedRecords.tail.next
                        }
                        //logp.Info("\n%s",hex.Dump(data[parseOffset:parseOffset+recordContentLength]))
                        // Lets do some record message parsing after 
                        parseOffset += recordContentLength + recordPaddingLength    
                    }else {
                        // we need more data, so let's stop parsing here, 
                        // and "unread" the header
                        // 2Do and WARN: This may cause an infinite loop on incomplete data
                        //               malformed packet or half conversation read
                        //logp.Info("protos.fcgi.fcgi_parser: More data needed got: %d, need: %d ",datasize - parseOffset,recordContentLength + recordPaddingLength)
                        parseOffset -= 8
                        stop = true
                    }
                }
            }
            //logp.Info("protos.fcgi.fcgi_parser: v=%d, type=%d, id=%d, len=%d, pad_len=%d, resv_0=%d",recordVersion, recordType, recordRequestId, recordContentLength, recordPaddingLength, recordReservedByte)
            //priv_fcgiData.Streams[dir].parseState = FCGI_AWT_RECORD_DATA

        } else {
            // get read bytes to go
        }

        priv_fcgiData.Streams[dir].parseOffset = parseOffset
        priv_fcgiData.ParsedRecordsCount += recCount;
        return priv_fcgiData, newRecordExists;
    }

