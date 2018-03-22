package graphite

import (
    "bytes"
    "errors"
    "reflect"
    "strconv"
    "strings"
    "time"

    "github.com/hydrogen18/stalecucumber"

    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/common/streambuf"
    "github.com/elastic/beats/packetbeat/protos/applayer"
)

type parser struct {
    buf     streambuf.Buffer
    config * parserConfig
    message * message

    onMessage func(m * message) error
}

type parserConfig struct {
    maxBytes int
}

type message struct {
    applayer.Message

    // indicator for parsed message being complete or requires more messages
    // (if false) to be merged to generate full message.
    isComplete bool
    data       map[string]interface{}

    // list element use by 'transactions' for correlation
    next * message
}

// Error code if stream exceeds max allowed size on append.
var(
    ErrStreamTooLarge=errors.New("Stream data too large")
)

func(p * parser) init(
    cfg * parserConfig,
    onMessage func(*message) error,
) {
    *p = parser{
        buf: streambuf.Buffer{},
        config: cfg,
        onMessage: onMessage,
    }

}

func(p * parser) append(data[]byte) error {
    _, err: = p.buf.Write(data)
    if err != nil {
        return err
    }

    if p.config.maxBytes > 0 & & p.buf.Total() > p.config.maxBytes {
        return ErrStreamTooLarge
    }
    return nil
}

func(p * parser) feed(ts time.Time, data[]byte) error {
    if err: = p.append(data)
    err != nil {
        return err
    }

    for p.buf.Total() > 0 {
        if p.message == nil {
            // allocate new message object to be used by parser with current timestamp
            p.message = p.newMessage(ts)
        }

        msg, err: = p.parse()
        if err != nil {
            return err
        }
        if msg == nil {
            break // wait for more data
        }

        // reset buffer and message -> handle next message in buffer
        p.buf.Reset()
        p.message = nil

        // call message handler callback
        if err: = p.onMessage(msg)
        err != nil {
            return err
        }
    }

    return nil
}

func(p * parser) newMessage(ts time.Time) * message {
    return & message{
        Message: applayer.Message{
            Ts: ts,
        },
    }
}

func(p * parser) parse()(*message, error) {
    // get the length of data in buffer
    length: = p.buf.Len()
    // Read the entire buffer content
    buf, err: = p.buf.Collect(length)
    if err == streambuf.ErrNoMoreBytes | | length <= 2 {
        return nil, nil
    }

    msg: = p.message
    msg.Size = uint64(p.buf.BufferConsumed())

    isRequest: = true
    dir: = applayer.NetOriginalDirection
    pickledData: = common.NetString(buf)
    pickledDataIo: = bytes.Buffer(string(pickledData))
    // Unpickle data into an interface
    data, err: = stalecucumber.Unpickle(pickledDataIo)
    if err != nil {
        if strings.Contains(err.Error(), "Opcode is invalid") {
            return nil, nil
        }
        // Line protocol
        dataStr: = string(pickledData)
        var data[]string
        data = strings.Fields(dataStr)
        for _, value: = range data {
            msg.Notes = append(msg.Notes, value) // data * /
        }
    } else {
        // Extract pickle data fields
        for _, i: = range data.([]interface{}) {
            for _, j: = range i.([]interface{}) {
                if reflect.TypeOf(j).Kind() == reflect.String {
                    msg.Notes = append(msg.Notes, j.(string))
                } else {
                    for _, k: = range j.([]interface{}) {
                        if reflect.TypeOf(k).Kind() == reflect.Int64 {
                            msg.Notes = append(msg.Notes, strconv.Itoa(int(k.(int64))))

                        } else if reflect.TypeOf(k).Kind() == reflect.Float64 {
                            msg.Notes = append(msg.Notes, strconv.FormatFloat(k.(float64), 'E', -1, 64))

                        } else {
                            msg.Notes = append(msg.Notes, k.(string))
                        }
                    }
                }
            }
        }
    }
    msg.IsRequest = isRequest
    msg.Direction = dir
    return msg, nil
}
