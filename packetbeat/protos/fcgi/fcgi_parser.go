package fcgi

    import  (   "time" 
                "github.com/elastic/beats/libbeat/logp"
            )

type fcgiParseState uint8

const (
    FCGI_START_PARSE fcgiParseState = iota  // initial stream state
    FCGI_AWT_RECORD_DATA                    // waiting record data to finish
)


    type parserState uint8

    const (
        stateStart parserState = iota
        stateFLine
        stateHeaders
        stateBody
        stateBodyChunkedStart
        stateBodyChunked
        stateBodyChunkedWaitFinalCRLF
    )


    type message struct {
        Ts               time.Time
        headerOffset     int
        bodyOffset       int
        //version          version
        //connection       common.NetString
        chunkedLength    int
        chunkedBody      []byte

        IsRequest    bool
        //TCPTuple     common.TcpTuple
        //CmdlineTuple *common.CmdlineTuple
        Direction    uint8

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
        start int
        end   int

        next *message
    }

    // called by Parse() on every new message

    func tryGetRecord(priv_fcgiData *fcgiData, dir uint8) (*fcgiData,bool) {

        // Working data, bytes already parsed
                
        data            := priv_fcgiData.Streams[dir].data;
        parseOffset     := priv_fcgiData.Streams[dir].parseOffset
        // parseState      := priv_fcgiData.Streams[dir].parseState
        datasize        := len(data)

        //logp.Info("protos.fcgi.fcgi_parser: datasize: %d parseOffset: %d moving: %d bytes",datasize,parseOffset,(datasize-parseOffset) ) 
        //priv_fcgiData.Streams[dir].parseOffset = datasize
        if parseOffset == FCGI_START_PARSE {
            // read 8 bytes
        } else {
            // get read bytes to go
        }

        return priv_fcgiData,false;
    }
























