package sip

import (
    "fmt"
    "strings"
    "strconv"

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
    ports               []int
    includeRawMessage   bool
    includeHeaders      bool
    includeBody         bool
    parseDetail         bool
    useDefaultHeaders   bool
    headersToParseAsURI []string
    headersToParseAsInt []string
    parseSet            map[string]int

    results protos.Reporter // Channel where results are pushed.
}

func (sip *sipPlugin) init(results protos.Reporter, config *sipConfig) error {
    sip.setFromConfig(config)

    if sip.parseDetail{
        sip.initDetailOption()
    }

    sip.results = results

    return nil
}

func (sip *sipPlugin) initDetailOption(){
    // Detail of headers
    sip.parseSet = make(map[string]int)

    if sip.useDefaultHeaders{
        sip.parseSet["from"                ] = SIP_DETAIL_NAME_ADDR
        sip.parseSet["to"                  ] = SIP_DETAIL_NAME_ADDR
        sip.parseSet["contact"             ] = SIP_DETAIL_NAME_ADDR
        sip.parseSet["record-route"        ] = SIP_DETAIL_NAME_ADDR
        sip.parseSet["p-asserted-identity" ] = SIP_DETAIL_NAME_ADDR
        sip.parseSet["p-preferred-identity"] = SIP_DETAIL_NAME_ADDR
    }
    for _,header := range sip.headersToParseAsURI{
        header=strings.ToLower(strings.TrimSpace(header))
        sip.parseSet[header] = SIP_DETAIL_NAME_ADDR
    }
    sip.parseSet["cseq"                ] = SIP_DETAIL_INT_METHOD
    sip.parseSet["rack"                ] = SIP_DETAIL_INT_INT_METHOD
    if sip.useDefaultHeaders{
        sip.parseSet["rseq"                ] = SIP_DETAIL_INT
        sip.parseSet["content-length"      ] = SIP_DETAIL_INT
        sip.parseSet["max-forwards"        ] = SIP_DETAIL_INT
        sip.parseSet["expires"             ] = SIP_DETAIL_INT
        sip.parseSet["session-expires"     ] = SIP_DETAIL_INT
        sip.parseSet["min-se"              ] = SIP_DETAIL_INT
    }
    for _,header := range sip.headersToParseAsInt{
        header=strings.ToLower(strings.TrimSpace(header))
        sip.parseSet[header] = SIP_DETAIL_INT
    }
}

// Set config values sip ports and options.
func (sip *sipPlugin) setFromConfig(config *sipConfig) error {
    sip.ports               = config.Ports
    sip.includeRawMessage   = config.IncludeRawMessage
    sip.includeHeaders      = config.IncludeHeaders
    sip.includeBody         = config.IncludeBody  
    sip.parseDetail         = config.ParseDetail
    sip.useDefaultHeaders   = config.UseDefaultHeaders
    sip.headersToParseAsURI = config.HeadersToParseAsURI
    sip.headersToParseAsInt = config.HeadersToParseAsInt
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
    if sip.includeRawMessage{
        fields["sip.raw"] = string(msg.raw)
    }
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
    if sip.includeHeaders{
        fields["sip.headers"] = sipHeaders

        if msg.headers != nil{
            for header,lines := range *(msg.headers){
                sipHeaders[header] = lines
            }
        }
    }

    if sip.includeBody  {
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
    }

    if sip.parseDetail{
        var display_name, user_info, host, port string
        var addrparams, params []string
        var number int
        var err error

        // Detail of Request-URI
        if value, ok := fields["sip.request-uri"]; ok {
            user_info, host, port,addrparams = sip.parseDetailURI(value.(string))

            fields["sip.request-uri-user"    ]     = user_info
            number,err=strconv.Atoi(strings.TrimSpace(port))
            if err==nil{
                fields["sip.request-uri-port"]     = number
            }
            fields["sip.request-uri-host"]         = host
            if len(addrparams) > 0 { fields["sip.request-uri-params"] = addrparams }
        }

        for key,values := range sipHeaders{
            newval:=[]common.MapStr{}

            for _, header_s := range values.([]common.NetString){
                newobj:=common.MapStr{}
                newobj[ "raw" ] = header_s

                if mode, ok := sip.parseSet[key]; ok{
                    switch(mode){
                    case SIP_DETAIL_NAME_ADDR:
                        display_name, user_info, host, port,addrparams, params = sip.parseDetailNameAddr(fmt.Sprintf("%s",header_s))

                        number,err = strconv.Atoi(port)
                        if display_name != "" { newobj[ "display" ] = display_name }
                        if user_info != ""    { newobj[ "user"    ] = user_info    }
                        if host != ""         { newobj[ "host"    ] = host         }
                        if err  == nil        { newobj[ "port"    ] = number       }
                        if addrparams != nil && len(addrparams) > 0 { newobj["uri-params"] = addrparams }
                        if params     != nil && len(params)     > 0 { newobj["params"    ] = params     }

                    case SIP_DETAIL_INT:
                        number,err=strconv.Atoi(strings.TrimSpace(fmt.Sprintf("%s",header_s)))
                        if err==nil{
                            newobj["number"]=number
                        }

                    case SIP_DETAIL_INT_METHOD:
                        values:=strings.SplitN(fmt.Sprintf("%s",header_s)," ",2)
                        number,err=strconv.Atoi(strings.TrimSpace(values[0]))
                        if err==nil{
                            newobj["number"]=number
                        }
                        newobj["method"]=strings.TrimSpace(values[1])

                    case SIP_DETAIL_INT_INT_METHOD:
                        values:=strings.SplitN(fmt.Sprintf("%s",header_s)," ",3)
                        number,err=strconv.Atoi(strings.TrimSpace(values[0]))
                        if err==nil{
                            newobj["number1"]=number
                        }
                        number,err=strconv.Atoi(strings.TrimSpace(values[1]))
                        if err==nil{
                            newobj["number2"]=number
                        }
                        newobj["method"]=strings.TrimSpace(values[2])
                    }
                }
                newval=append(newval,newobj)
            }
            sipHeaders[key]=newval
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

func (sip *sipPlugin) parseDetailURI(addr string) (user_info string, host string, port string, params []string){
    var prevChar rune
    addr=strings.TrimSpace(addr)
    prevChar=' '
    pos:=-1
    ppos:=-1
    epos:=len(addr)
    in_v6:=false
    idx:=0
    for idx=0; idx<len(addr);idx++{
        curChar:=rune(addr[idx])

        if idx == 0{
            if(idx+4>=len(addr)){
                break
            }
            // sip/sips/tel-uri
            if addr[idx:idx+5] == "sips:"{
                idx+=4
            }else if addr[idx:idx+4] == "sip:"||addr[idx:idx+4] == "tel:"{
                idx+=3
            }else{
                break
            }
            pos=idx+1
        } else if curChar == '[' && prevChar != '\\'{
            in_v6 = true
        } else if curChar == ']' && prevChar != '\\'{
            in_v6 = false
        } else if curChar == ';' && prevChar != '\\'{
            if len(params) == 0{
                epos=idx
                params=strings.Split(addr[idx+1:],";")
            }
            //break
        } else {
            // select wich part
            switch(curChar){
            case '@':
                if host != ""{
                    pos=ppos
                    host=""
                }
                if len(params) > 0{
                    epos=len(addr)
                    params=params[:0] // clear slice
                }
                user_info=addr[pos:idx]
                ppos=pos
                pos=idx+1
            case ':':
                if ! in_v6{
                    host=addr[pos:idx]
                    ppos=pos
                    pos=idx+1
                }
            }
        }
        prevChar=curChar
    }
    if pos>0 && epos <= len(addr) && pos <= epos{
        if host==""{
            host=strings.TrimSpace(addr[pos:epos])
        }else{
            port=strings.TrimSpace(addr[pos:epos])
        }
    }

    return user_info, host, port , params
}

func (sip *sipPlugin) parseDetailNameAddr(addr string) (display_name string,user_info string, host string, port string, addrparams []string, params []string) {

    addr=strings.TrimSpace(addr)
    var prevChar rune
    prevChar=' '
    pos:=-1
    _=port
    in_addr:=false
    escaped:=false

    for idx:=0; idx<len(addr);idx++{
        curChar:=rune(addr[idx])
        // Display name
        if !in_addr && display_name == "" && user_info == "" && host == "" {
            if idx == 0 && idx+5 < len(addr){
                if addr[idx:idx+5] == "sips:" || addr[idx:idx+4] == "sip:" || addr[idx:idx+4] == "tel:"{
                    user_info,host,port,addrparams=sip.parseDetailURI(addr[idx:])
                    idx=len(addr)
                    break
                }
            }
            if idx == 0 && curChar != '<'{
                pos=idx
                if curChar== '"'{
                    pos++
                    escaped=true
                }
                continue
            }else if curChar == '"' && prevChar != '\\'{
                display_name=addr[pos:idx]
                pos=-1
            } else if escaped{
                prevChar=curChar
                continue
            }
        }
        if curChar == '<' && !in_addr && prevChar != '\\'{
            if display_name=="" && pos >=0{
                display_name=strings.TrimSpace(addr[pos:idx])
            }
            pos=idx+1
            for idx=idx+1; idx < len(addr); idx++{
                if rune(addr[idx]) == '>' && addr[idx-1] != '\\'{
                    user_info,host,port,addrparams=sip.parseDetailURI(addr[pos:idx])

                    for idx=idx+1; idx < len(addr); idx++{
                        if rune(addr[idx]) == ';'{
                            substr:=addr[idx+1:]
                            params=strings.Split(substr,";")
                            idx=len(addr)
                        }
                    }
                    break
                }
            }
        }

        prevChar=curChar
    }

    return display_name,user_info, host, port , addrparams, params
}

