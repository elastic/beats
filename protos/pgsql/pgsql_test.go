package pgsql

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/publisher"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/packetbeat/protos"
)

func PgsqlModForTests() *Pgsql {
	var pgsql Pgsql
	results := publisher.ChanClient{make(chan common.MapStr, 10)}
	pgsql.Init(true, results)
	return &pgsql
}

// Test parsing a request with a single query
func TestPgsqlParser_simpleRequest(t *testing.T) {
	pgsql := PgsqlModForTests()

	data := []byte(
		"510000001a53454c454354202a2046524f4d20466f6f6261723b00")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

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
	if stream.message.Size != 27 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

// Test parsing a response with data attached
func TestPgsqlParser_dataResponse(t *testing.T) {

	pgsql := PgsqlModForTests()
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

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

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

	if stream.message.Size != 126 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

// Test parsing a pgsql response
func TestPgsqlParser_response(t *testing.T) {

	pgsql := PgsqlModForTests()
	data := []byte(
		"54000000420003610000004009000100000413ffffffffffff0000620000004009000200000413ffffffffffff0000630000004009000300000413ffffffffffff0000" +
			"440000001b0003000000036d6561000000036d6562000000036d6563" +
			"440000001e0003000000046d656131000000046d656231000000046d656331" +
			"440000001e0003000000046d656132000000046d656232000000046d656332" +
			"440000001e0003000000046d656133000000046d656233000000046d656333" +
			"430000000d53454c454354203400" +
			"5a0000000549")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

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
	if stream.message.NumberOfFields != 3 {
		t.Error("Failed to parse the number of field")
	}
	if stream.message.NumberOfRows != 4 {
		t.Error("Failed to parse the number of rows")
	}

	if stream.message.Size != 202 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

// Test parsing an incomplete pgsql response
func TestPgsqlParser_incomplete_response(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"pgsql", "pgsqldetailed"})
	}
	pgsql := PgsqlModForTests()

	data := []byte(
		"54000000420003610000004009000100000413ffffffffffff0000620000004009000200000413ffffffffffff0000630000004009000300000413ffffffffffff0000" +
			"440000001b0003000000036d6561000000036d6562000000036d6563" +
			"440000001e0003000000046d656131000000046d656231000000046d656331" +
			"440000001e0003000000046d")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if complete {
		t.Error("Expecting an incomplete message")
	}

}

// Test 3 responses in a row
func TestPgsqlParser_threeResponses(t *testing.T) {

	pgsql := PgsqlModForTests()

	data, err := hex.DecodeString(
		"5300000017446174655374796c650049534f2c204d445900430000000853455400430000000853455400540000005700036f696400000004eefffe0000001a0004ffffffff0000656e636f64696e6700000000000000000000130040ffffffff00006461746c6173747379736f696400000004ee00090000001a0004ffffffff0000440000002000030000000531313836350000000455544638000000053131383537430000000d53454c4543542031005a0000000549")
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Error("Failed to get ts")
	}
	pkt := protos.Packet{
		Payload: data,
		Ts:      ts,
	}
	var tuple common.TcpTuple
	var private pgsqlPrivateData
	var count_handlePgsql = 0

	pgsql.handlePgsql = func(pgsql *Pgsql, m *PgsqlMessage, tcptuple *common.TcpTuple,
		dir uint8, raw_msg []byte) {

		count_handlePgsql += 1
	}

	pgsql.Parse(&pkt, &tuple, 1, private)

	if count_handlePgsql != 3 {
		t.Error("handlePgsql not called three times")
	}

}

// Test parsing an error response
func TestPgsqlParser_errorResponse(t *testing.T) {
	pgsql := PgsqlModForTests()
	data := []byte(
		"4500000088534552524f5200433235503032004d63757272656e74207472616e73616374696f6e2069732061626f727465642c20636f6d6d616e64732069676e6f72656420756e74696c20656e64206f66207472616e73616374696f6e20626c6f636b0046706f7374677265732e63004c3932310052657865635f73696d706c655f71756572790000")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if !ok {
		t.Error("Parsing returned error")
	}
	if !complete {
		t.Error("Expecting a complete message")
	}

	if stream.message.IsRequest {
		t.Error("Failed to parse postgres response")
	}
	if !stream.message.IsError {
		t.Error("Failed to parse error response")
	}
	if stream.message.ErrorSeverity != "ERROR" {
		t.Error("Failed to parse severity")
	}
	if stream.message.ErrorCode != "25P02" {
		t.Error("Failed to parse error code")
	}
	if stream.message.ErrorInfo != "current transaction is aborted, commands ignored until end of transaction block" {
		t.Error("Failed to parse error message")
	}
	if stream.message.Size != 137 {
		t.Errorf("Wrong message size %d", stream.message.Size)
	}
}

// Test parsing an error response
func TestPgsqlParser_invalidMessage(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"pgsql", "pgsqldetailed"})
	}
	pgsql := PgsqlModForTests()
	data := []byte(
		"4300000002")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Error("Failed to decode hex string")
	}

	stream := &PgsqlStream{data: message, message: new(PgsqlMessage)}

	ok, complete := pgsql.pgsqlMessageParser(stream)

	if ok {
		t.Error("Parsing returned success instead of error")
	}
	if complete {
		t.Error("Expecting a non complete message")
	}
}

func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: 6512, Dst_port: 5432,
	}
	t.ComputeHashebles()
	return t
}

// Helper function to read from the Publisher Queue
func expectTransaction(t *testing.T, pgsql *Pgsql) common.MapStr {
	client := pgsql.results.(publisher.ChanClient)
	select {
	case trans := <-client.Channel:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

// Test that loss of data during the response (but not at the beginning)
// don't cause the whole transaction to be dropped.
func Test_gap_in_response(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"pgsql", "pgsqldetailed"})
	}

	pgsql := PgsqlModForTests()

	// request and response from tests/pcaps/pgsql_request_response.pcap
	// select * from test
	req_data, err := hex.DecodeString(
		"510000001873656c656374202a20" +
			"66726f6d20746573743b00")
	assert.Nil(t, err)

	// response is incomplete
	resp_data, err := hex.DecodeString(
		"5400000042000361000000410900" +
			"0100000413ffffffffffff0000620000" +
			"004009000200000413ffffffffffff00" +
			"00630000004009000300000413ffffff" +
			"ffffff0000440000001b000300000003" +
			"6d6561000000036d6562000000036d65" +
			"63440000001e0003000000046d656131" +
			"000000046d656231000000046d656331" +
			"440000001e0003000000046d65613200")
	assert.Nil(t, err)

	tcptuple := testTcpTuple()
	req := protos.Packet{Payload: req_data}
	resp := protos.Packet{Payload: resp_data}

	private := protos.ProtocolData(new(pgsqlPrivateData))

	private = pgsql.Parse(&req, tcptuple, 0, private)
	private = pgsql.Parse(&resp, tcptuple, 1, private)

	logp.Debug("pgsql", "Now sending gap..")

	private, drop := pgsql.GapInStream(tcptuple, 1, 10, private)
	assert.Equal(t, true, drop)

	trans := expectTransaction(t, pgsql)
	assert.NotNil(t, trans)
	assert.Equal(t, trans["notes"], []string{"Packet loss while capturing the response"})
}
