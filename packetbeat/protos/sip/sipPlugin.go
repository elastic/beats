package sip

import (
    "fmt"

    "github.com/elastic/beats/libbeat/beat"
    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"

    "github.com/elastic/beats/packetbeat/procs"
    "github.com/elastic/beats/packetbeat/protos"
)

/**
 ******************************************************************
 * sipPlugin
 ******************************************************************
 **/
type sipPlugin struct {
    // Configuration data.
    ports              []int

    results protos.Reporter // Channel where results are pushed.
}

func (sip *sipPlugin) init(results protos.Reporter, config *sipConfig) error {
    sip.setFromConfig(config)

    sip.results = results

    return nil
}

// Set config values sip ports.
func (sip *sipPlugin) setFromConfig(config *sipConfig) error {
    sip.ports                 = config.Ports
    return nil
}

// Getter : instance Ports int slice
func (sip *sipPlugin) GetPorts() []int {
    return sip.ports
}

// publishMessage to reshape the sipMessage for in order to pushing with json.
func (sip *sipPlugin) publishMessage(msg *sipMessage) {
    if sip.results == nil {
        return
    }

    debugf("Publishing SIP Message. %s", msg.String())

    timestamp := msg.ts
    fields := common.MapStr{}
    fields["sip.unixtimenano"] = timestamp.UnixNano()
    fields["type"] = "sip"
    fields["sip.transport"] = msg.transport.String()
    fields["sip.raw"] = string(msg.raw)
    fields["sip.src"] = fmt.Sprintf("%s:%d",msg.tuple.SrcIP,msg.tuple.SrcPort)
    fields["sip.dst"] = fmt.Sprintf("%s:%d",msg.tuple.DstIP,msg.tuple.DstPort)

    if msg.isRequest {
        fields["sip.method"     ] = fmt.Sprintf("%s",msg.method)
        fields["sip.request-uri"] = fmt.Sprintf("%s",msg.requestUri)
    }else{
        fields["sip.status-code"  ] = int(msg.statusCode)
        fields["sip.status-phrase"] = fmt.Sprintf("%s",msg.statusPhrase)
    }

    fields["sip.from"   ] = fmt.Sprintf("%s",msg.from)
    fields["sip.to"     ] = fmt.Sprintf("%s",msg.to)
    fields["sip.cseq"   ] = fmt.Sprintf("%s",msg.cseq)
    fields["sip.call-id"] = fmt.Sprintf("%s",msg.callid)

    sipHeaders := common.MapStr{}
    fields["sip.headers"] = sipHeaders

    if msg.headers != nil{
        for header,lines := range *(msg.headers){
            sipHeaders[header] = lines
        }
    }

    sipBody := common.MapStr{}
    fields["sip.body"] = sipBody

    if msg.body !=nil{
        for content,keyval := range (msg.body){
            contetMap := common.MapStr{}
            sipBody[content] = contetMap
            for key,val_lines := range *keyval{
                contetMap[key] = val_lines
            }
        }
    }

    if msg.notes != nil{
        fields["sip.notes"] = fmt.Sprintf("%s",msg.notes)
    }

    sip.results(beat.Event{
        Timestamp: timestamp,
        Fields:    fields,
    })
}

// createSIPMessage a byte array into a SIP struct. If an error occurs
// then the returned sip pointer will be nil. This method recovers from panics
// and is concurrency-safe.
func (sip *sipPlugin) createSIPMessage(transp transport, rawData []byte) (msg *sipMessage, err error) {
    // Recover from any panics that occur while parsing a packet.
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r)
        }
    }()

    // create and initialized pakcet raw message and transport type.
    msg = &sipMessage{}
    msg.transport=transp
    msg.raw = rawData

    // offset values are initialized to -1
    msg.hdr_start    =-1
    msg.hdr_len      =-1
    msg.bdy_start    =-1
    msg.contentlength=-1

    msg.isIncompletedHdrMsg = false
    msg.isIncompletedBdyMsg = false

    return msg, nil
}

func (sip *sipPlugin) ParseUDP(pkt *protos.Packet) {

    defer logp.Recover("Sip ParseUdp")
    packetSize := len(pkt.Payload)

    debugf("Parsing packet addressed with %s of length %d.", pkt.Tuple.String(), packetSize)

    var sipMsg *sipMessage
    var err error

    debugf("New sip message: %s %s",&pkt.Tuple,transportUDP)

    // create new SIP Message
    sipMsg, err = sip.createSIPMessage(transportUDP, pkt.Payload)

    if err != nil{
        // ignore this message
        debugf("error %s\n",err)
        return
    }

    sipMsg.ts   =pkt.Ts
    sipMsg.tuple=pkt.Tuple
    sipMsg.cmdlineTuple=procs.ProcWatcher.FindProcessesTuple(&pkt.Tuple)

    // parse sip headers.
    // if the message was malformed, the message will be rejected
    parseHeaderErr:=sipMsg.parseSIPHeader()
    if parseHeaderErr != nil{
        debugf("error %s\n",parseHeaderErr)
        return
    }

    switch sipMsg.getMessageStatus(){
    case SIP_STATUS_REJECTED:
        return
    // In case the message was incompleted at header or body,
    // the message was added error notes and published.
    case SIP_STATUS_HEADER_RECEIVING, SIP_STATUS_BODY_RECEIVING:
        debugf("Incompleted message")
        sipMsg.notes = append(sipMsg.notes,common.NetString(fmt.Sprintf("Incompleted message")))

    // In case the message received completely, publishing the message.
    case SIP_STATUS_RECEIVED:
        err := sipMsg.parseSIPBody()
        if err != nil{
            sipMsg.notes = append(sipMsg.notes,common.NetString(fmt.Sprintf("%s",err)))
        }
    }
    sip.publishMessage(sipMsg)
}

