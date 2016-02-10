package fcgi

import (
    "time"
//    "fmt"
    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"
    "github.com/elastic/beats/packetbeat/protos"
    "github.com/elastic/beats/packetbeat/protos/tcp" // tcp.TcpDirectionOriginal n R
    "github.com/elastic/beats/packetbeat/publish"


)

//
// Define the required structs and types
//

    // The fastcgi plugin object

    type Fcgi struct {
        // config
        Ports               []int
        SendRequest         bool
        SendResponse        bool
        SplitCookie         bool
        HideKeywords        []string
        RedactAuthorization bool
        // parserConfig parserConfig
        transactionTimeout time.Duration
        results publish.Transactions
    }
    
    // Store bytes and bytes from a input or output
    // stream and  and whole parsed records with its info

    type stream struct {
        tcptuple *common.TcpTuple
        data []byte                 // Stream raw data
        parseOffset  uint32         // Bytes already parsed
        parseState   fcgiParseState // Control flags help
                                    // parsing long messages
        message *message            // Whole FCGI records with
                                    // a pointer for the next 
    }
    
    // A simple data structure with start and end
    // for message objects (list nodes)
    
    type messageList struct {
        head, tail *message
    }

    // Store a fcgi tcp connection data
    // with fcgiData (this is the private object that
    // is passed along tcp conversations)

    type fcgiData struct {
        Streams             map [uint8]*stream  // 
        ParsedRecordsCount  int                 //
        ParsedRecords       *messageList         //
    }
    
    // Interface call for fcgi protocol analyzer 

    func (fcgi *Fcgi) Init(test_mode bool, results publish.Transactions) error{
        logp.Info( "protos.fcgi.fcgi.Init: Init fcgi parse plugin.");
        return nil; 
    }

    func (fcgi *Fcgi) GetPorts() []int {
    
        return []int { 9000 }

    }

    // type Packet struct {
    //    Ts      time.Time 
    //    Tuple   common.IpPortTuple 
    //    Payload []byte 
    // }

    // tcptuple: Unique identifier for the TCP stream that the packet is part of.  tcptuple.Hashable()
    // dir flag: Direction in which the packet is flowing 
    //              tcp.TcpDirectionOriginal
    //              tcp.TcpDirectionReverse 
    // private:  Var to store state in the TCP stream. 
    //           it is the same per tcp stream so there are no x-talking
    //           all Parse() interface call, will be with the modified private value.

    func (fcgi *Fcgi) Parse(pkt *protos.Packet, tcptuple *common.TcpTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {
        
        // Debug
        //if dir == tcp.TcpDirectionOriginal {
        //    logp.Info("protos.fcgi.fcgi.Parse: %s:%d > %s:%d",tcptuple.Src_ip.String(),tcptuple.Src_port,tcptuple.Dst_ip.String(),tcptuple.Dst_port);
        //} else {
        //    logp.Info("protos.fcgi.fcgi.Parse: %s:%d < %s:%d",tcptuple.Src_ip.String(),tcptuple.Src_port,tcptuple.Dst_ip.String(),tcptuple.Dst_port);
        //}
       
        // try cast private to fcgiData
        priv_fcgiData, ok := private.(*fcgiData)

        // 2Do: Just create fcgiData with initialized streams constants
        //      and remove next stream checks * (if possible)

        if !ok {
            priv_fcgiData = &fcgiData{ Streams: make(map[uint8]*stream), ParsedRecordsCount: 0, ParsedRecords: &messageList{ head: nil, tail: nil} } 
        }
        if priv_fcgiData.Streams[dir] == nil {
            priv_fcgiData.Streams[dir] = &stream{ tcptuple: tcptuple, data: pkt.Payload }
        } else{
            priv_fcgiData.Streams[dir].data = append(priv_fcgiData.Streams[dir].data, pkt.Payload...)
        }
        priv_fcgiData, newRecordExists := tryGetRecord(priv_fcgiData, dir, pkt.Ts) 
        if newRecordExists {
            //logp.Info("Got a... New record!")
        }
        return priv_fcgiData
    }


    func (fcgi *Fcgi) ReceivedFin(tcptuple *common.TcpTuple, dir uint8, private protos.ProtocolData) protos.ProtocolData {

        // Debug
        //if dir == tcp.TcpDirectionOriginal {
        //    logp.Info("protos.fcgi.fcgi.ReceivedFin: %s:%d > %s:%d",tcptuple.Src_ip.String(),tcptuple.Src_port,tcptuple.Dst_ip.String(),tcptuple.Dst_port);
        //} else {
        //    logp.Info("protos.fcgi.fcgi.ReceivedFin: %s:%d < %s:%d",tcptuple.Src_ip.String(),tcptuple.Src_port,tcptuple.Dst_ip.String(),tcptuple.Dst_port);
        //}
        priv_fcgiData, ok := private.(*fcgiData)
        if ok && dir == tcp.TcpDirectionOriginal {

            /*rs_pointer  := priv_fcgiData.ParsedRecords.head
            rs_string   := ""
            for rs_pointer != nil {
                 
                rs_string += fmt.Sprint(rs_pointer.recordType) + ", "
                rs_pointer = rs_pointer.next
            } 


            logp.Info("protos.fcgi.fcgi.ReceivedFin: %s (%d)",rs_string, priv_fcgiData.ParsedRecordsCount);
            */
            priv_fcgiData = nil
        }
        return priv_fcgiData
    }

    // GapInStream is called when a gap of nbytes bytes is found in the stream (due
    // to packet loss).

    func (fcgi *Fcgi) GapInStream(tcptuple *common.TcpTuple, dir uint8, nbytes int,
                private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {
        logp.Info("protos.fcgi.fcgi.GapInStream: ")

        return private, true
    }

    func (fcgi *Fcgi) ConnectionTimeout() time.Duration {
        //logp.Info("Timeout! .\n");
        return time.Duration(60) * time.Second
    }
