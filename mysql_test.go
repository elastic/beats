package main

import (
    "encoding/hex"
    "testing"
    //"fmt"
    //"log/syslog"
)

func TestMySQLParser_simpleRequest(t *testing.T) {

    data := []byte(
        "6f00000003494e5345525420494e544f20706f737" +
            "42028757365726e616d652c207469746c652c2062" +
            "6f64792c207075625f64617465292056414c55455" +
            "3202827416e6f6e796d6f7573272c202774657374" +
            "272c202774657374272c2027323031332d30372d3" +
            "2322031383a34343a31372729")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

    ok, complete := mysqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if !stream.message.IsRequest {
        t.Error("Failed to parse MySQL request")
    }
    if stream.message.Query != "INSERT INTO post (username, title, body, pub_date) VALUES ('Anonymous', 'test', 'test', '2013-07-22 18:44:17')" {
        t.Error("Failed to parse query")
    }

}
func TestMySQLParser_simpleOKResponse(t *testing.T) {

    data := []byte(
        "0700000100010401000000")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

    ok, complete := mysqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if stream.message.IsRequest {
        t.Error("Failed to parse MySQL response")
    }
    if !stream.message.IsOK {
        t.Error("Failed to parse Response OK")
    }
    if stream.message.AffectedRows != 1 {
        t.Error("Failed to parse affected rows")
    }
    if stream.message.InsertId != 4 {
        t.Error("Failed to parse last INSERT id")
    }
}

func TestMySQLParser_simpleResponse(t *testing.T) {
    //LogInit(syslog.LOG_DEBUG, "" /*toSyslog*/, false, []string{"mysqldetailed"})

    data := []byte(
        "0100000105" +
            "2f00000203646566086d696e697477697404706f737404706f737407706f73745f69640269640c3f000b000000030342000000" +
            "3b00000303646566086d696e697477697404706f737404706f73740d706f73745f757365726e616d6508757365726e616d650c2100f0000000fd0000000000" +
            "3500000403646566086d696e697477697404706f737404706f73740a706f73745f7469746c65057469746c650c2100f0000000fd0000000000" +
            "3300000503646566086d696e697477697404706f737404706f737409706f73745f626f647904626f64790c2100fdff0200fc1000000000" +
            "3b00000603646566086d696e697477697404706f737404706f73740d706f73745f7075625f64617465087075625f646174650c3f00130000000c8000000000" +
            "05000007fe00002100" +
            "2e000008013109416e6f6e796d6f75730474657374086461736461730d0a13323031332d30372d32322031373a33343a3032" +
            "46000009013209416e6f6e796d6f757312506f737465617a6120544f444f206c6973741270656e7472752063756d706172617475726913323031332d30372d32322031383a32393a3330" +
            "2a00000a013309416e6f6e796d6f75730454657374047465737413323031332d30372d32322031383a33323a3130" +
            "2a00000b013409416e6f6e796d6f75730474657374047465737413323031332d30372d32322031383a34343a3137" +
            "0500000cfe00002100")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

    ok, complete := mysqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if stream.message.IsRequest {
        t.Error("Failed to parse MySQL Query response")
    }
    if !stream.message.IsOK || stream.message.IsError {
        t.Error("Failed to parse MySQL Query response")
    }
    if stream.message.Size != 528 {
        t.Error("Failed to get the size of the message")
    }
    if stream.message.Tables != "minitwit.post" {
        t.Error("Failed to get table name: %s", stream.message.Tables)
    }
    if stream.message.NumberOfFields != 5 {
        t.Error("Failed to get the number of fields")
    }
    if stream.message.NumberOfRows != 4 {
        t.Error("Failed to get the number of rows")
    }
    if stream.message.Size != 528 {
        t.Error("failed to get the size of the response")
    }
}
func TestMySQLParser_simpleUpdateResponse(t *testing.T) {
    //LogInit(syslog.LOG_DEBUG, "" /*toSyslog*/, false, []string{"mysqldetailed"})

    data := []byte("300000010001000100000028526f7773206d6174636865643a203120204368616e6765643a203120205761726e696e67733a2030")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

    ok, complete := mysqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if stream.message.IsRequest {
        t.Error("Failed to parse MySQL Query response")
    }
    if !stream.message.IsOK || stream.message.IsError {
        t.Error("Failed to parse MySQL Query response")
    }
    if stream.message.AffectedRows != 1 {
        t.Error("Failed to get the number of affected rows")
    }
}
