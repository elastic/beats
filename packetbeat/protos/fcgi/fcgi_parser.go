package fcgi

    import  (   "time" 
//                "bytes"
                "github.com/elastic/beats/libbeat/logp"
//                "github.com/elastic/beats/packetbeat/protos/tcp"    // tcp.TcpDirectionOriginal n R
//                "encoding/hex"                                      // Dump object
            )

type fcgiParseState uint8

const (
    FCGI_START_PARSE fcgiParseState = iota  // initial stream state
    FCGI_AWT_RECORD_DATA                    // waiting record data to finish
)
type fcgiProtoAppRecord uint8
// FCGI protocol defined constants
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
// Record type to name
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
// Parameters from Recrod type FCGI_PARAMS
// that should be stored and reported
var fcgiParamsRecordStoreParams = map[string]int{
    "SCRIPT_NAME"           : 1,
    "REQUEST_URI"           : 1,
    "HTTP_HOST"             : 1,
    "QUERY_STRING"          : 1,
    "REQUEST_METHOD"        : 1,
    "HTTP_CONTENT_LENGTH"   : 1,
    //"" : 1,
}

// DataStructure for storing records, includes
// a pointer to next record in order to store 
// them as a FIFO list.
type message struct {
   
    // Net info
    Ts                  time.Time
    Direction           uint8

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
    //Raw []byte

    //Notes []string

    //Timing
    //start int
    //end   int

    // Data structure pointer
    next *message
}

    ///////////////////////////////////////////////////////////////////////////////////////
    // tryGetRecord: Called by Parse() on every new message trys to get whole records    //
    //               and add them into the priv data.                                    //
    // Argument:     Private object, packet flow dir, packet time                        //
    // Returns:      Private modified object, boolean flag (may not be needed)           //
    ///////////////////////////////////////////////////////////////////////////////////////

    func tryGetRecord(priv_fcgiData *fcgiData, dir uint8, Ts time.Time) (*fcgiData,bool) {

        // Working data, bytes already parsed
        recCount        := 0
        newRecordExists := false        
        data            := priv_fcgiData.Streams[dir].data;
        parseOffset     := priv_fcgiData.Streams[dir].parseOffset
        parseState      := priv_fcgiData.Streams[dir].parseState
        datasize        := uint32(len(data))

        // 2Do: Del this line or setup on debug mode
        //logp.Info("protos.fcgi.fcgi_parser: datasize: %d parseOffset: %d moving: %d bytes",datasize,parseOffset,(datasize-parseOffset) ) 
       
        // 2Do: Del this condition or improve for better flow control
        if parseState == FCGI_START_PARSE {
            stop := false
            // At least we need a whole record so we check for 8 bytes
            for !stop && parseOffset + 8 <= datasize {
                // read 8 bytes
                recordVersion           := uint32(data[parseOffset + 0])
                recordType              := uint32(data[parseOffset + 1])
                //recordRequestId         := uint32(data[parseOffset + 2]) << 8 + uint32(data[parseOffset + 3])
                recordContentLength     := uint32(data[parseOffset + 4]) << 8 + uint32(data[parseOffset + 5])       
                recordPaddingLength     := uint32(data[parseOffset + 6])
                recordReservedByte      := uint32(data[parseOffset + 7])

                parseOffset += 8

                // Error or bogus message header byte check
                if ( recordVersion > 1 || recordReservedByte != 0 || recordType > FCGI_UNKNOWN_TYPE ){
                    parseOffset = datasize  // Ignore all trailing data as we can't parse in a correct way
                    //2Do: Emit a WARN to logs
                }else{

                    // Wait until the record has all its data
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
                            // 2Do: Create a Separate function for each record type
                            if ( recordType == FCGI_PARAMS  && recordContentLength > 0 ) {
                                paramsParseOffset := parseOffset
                                // 2Do: remove this string after a data structure is used
                                printstring := ""

                                for paramsParseOffset < recordContentLength {
                                    var (
                                    nameLength   uint32 = 0
                                    valueLength  uint32 = 0
                                    )

                                    if ( data[ paramsParseOffset ] >> 7 == 1 ){
                                        nameLength =    uint32(data[paramsParseOffset] & 0x7f) << 24 + 
                                                        uint32(data[paramsParseOffset+1]) << 16 + 
                                                        uint32(data[paramsParseOffset+2]) << 8 + 
                                                        uint32(data[paramsParseOffset+3]) 
                                        paramsParseOffset += 4 
                                    }else{
                                        nameLength = uint32(data[paramsParseOffset])
                                        paramsParseOffset++
                                    }

                                    if ( data [paramsParseOffset] >> 7 == 1 ){
                                        valueLength =   uint32(data[paramsParseOffset] & 0x7f) << 24 + 
                                                        uint32(data[paramsParseOffset+1]) << 16 + 
                                                        uint32(data[paramsParseOffset+2]) << 8 + 
                                                        uint32(data[paramsParseOffset+3])
                                        paramsParseOffset += 4

                                    }else{
                                        valueLength =   uint32(data[paramsParseOffset])
                                        paramsParseOffset++
                                    }
                                    key := string(data[paramsParseOffset:paramsParseOffset+nameLength])
                                    val := string(data[paramsParseOffset+nameLength:paramsParseOffset+nameLength+valueLength])

                                    // 2Do: Store interesting params into a data structure

                                    if  gotKey, gotVal1 := fcgiParamsRecordStoreParams[key]; gotVal1 {
                                        gotKey += 1; // Just any operation to compile
                                        printstring+= key + " " + val  + "\n"
                                    }

                                    //logp.Info("name len:%d  value len: %d \n%s",nameLength,valueLength,hex.Dump(data[parseOffset:parseOffset+recordContentLength]))

                                    //logp.Info("key: %s value: %s",string(data[paramsParseOffset:paramsParseOffset+nameLength]),
                                    //string(data[paramsParseOffset+nameLength:paramsParseOffset+nameLength+valueLength]))
                                    
                                    paramsParseOffset += nameLength + valueLength

                                }// end parsing parameters cycle

                                logp.Info("protos.fcgi.tryGetRecord: FCGI_PARAMS\n%s",printstring)
                            }// end if record == FCGI_PARAMS


                            priv_fcgiData.ParsedRecords.tail.next = &message{   recordType: recordType, 
                                                                                recordContentLength: recordContentLength,
                                                                                next: nil,
                                                                                Direction: dir,
                                                                                Ts: Ts, }
                            priv_fcgiData.ParsedRecords.tail = priv_fcgiData.ParsedRecords.tail.next
                        }
                        //logp.Info("\n%s",hex.Dump(data[parseOffset:parseOffset+recordContentLength]))
                        // Lets do some record message parsing after 
                        parseOffset += recordContentLength + recordPaddingLength    
                    }else { 
                        // Record header demands more data, stop parsing, "unread" the header
                        // and wait for next Parse() call
                        parseOffset -= 8
                        stop = true
                    }// End complete-data B4 parsing check
                }// End basic header checks
            }// End parsing cycle (end data or stop flag)

            //2Do: Del this lines or put them for debugging 
            //logp.Info("protos.fcgi.fcgi_parser: v=%d, type=%d, id=%d, len=%d, pad_len=%d, resv_0=%d",recordVersion, recordType, recordRequestId, recordContentLength, recordPaddingLength, recordReservedByte)
            //priv_fcgiData.Streams[dir].parseState = FCGI_AWT_RECORD_DATA

        } else { // End for parseState check (may be disappearing)
            // get read bytes to go
        }

        // Update parsing data for stream
        priv_fcgiData.Streams[dir].parseOffset = parseOffset
        priv_fcgiData.ParsedRecordsCount += recCount;
        return priv_fcgiData, newRecordExists;
    }

