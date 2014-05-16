package main

import (
    "encoding/hex"
    "testing"
    //"fmt"
)

func TestPgsqlParser_simpleRequest(t *testing.T) {

    data := []byte(
        "510000001a53454c454354202a2046524f4d20466f6f6261723b00")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &PgsqlStream{tcpStream: nil, data: message, message: new(PgsqlMessage)}

    ok, complete := pgsqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if !stream.message.IsRequest {
        t.Error("Failed to parse postgres request")
    }
    if stream.message.Query != "SELECT * FROM Foobar;" {
        t.Error("Failed to parse query")
    }

}

func TestPgsqlParser_dataResponse(t *testing.T) {

    data := []byte(
        "5400000033000269640000008fc40001000000170004ffffffff000076616c75650000008fc4000200000019ffffffffffff0000" +
        "44000000130002000000013100000004746f746f" +
        "440000001500020000000133000000066d617274696e" +
        "440000001300020000000134000000046a65616e" +
        "430000000b53454c45435400" +
        "5a0000000549")

    message, err := hex.DecodeString(string(data))
    if err != nil {
        t.Error("Failed to decode hex string")
    }

    stream := &PgsqlStream{tcpStream: nil, data: message, message: new(PgsqlMessage)}

    ok, complete := pgsqlMessageParser(stream)

    if !ok {
        t.Error("Parsing returned error")
    }
    if !complete {
        t.Error("Expecting a complete message")
    }
    if stream.message.IsRequest {
        t.Error("Failed to parse postgres response")
    }
    if !stream.message.IsOK || stream.message.IsError {
        t.Error("Failed to parse postgres response")
    }
    if stream.message.NumberOfFields != 2 {
        t.Error("Failed to parse the number of field")
    }
    if stream.message.NumberOfRows != 3 {
        t.Error("Failed to parse the number of rows")
    }

}


