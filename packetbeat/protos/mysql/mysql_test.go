// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build !integration

package mysql

import (
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/tcp"
	"github.com/elastic/beats/packetbeat/publish"
)

const serverPort = 3306

type eventStore struct {
	events []beat.Event
}

func (e *eventStore) publish(event beat.Event) {
	publish.MarshalPacketbeatFields(&event, nil)
	e.events = append(e.events, event)
}

func (e *eventStore) empty() bool {
	return len(e.events) == 0
}

func mysqlModForTests(store *eventStore) *mysqlPlugin {
	callback := func(beat.Event) {}
	if store != nil {
		callback = store.publish
	}

	var mysql mysqlPlugin
	config := defaultConfig
	config.Ports = []int{serverPort}
	mysql.init(callback, &config)
	return &mysql
}

func Test_parseStateNames(t *testing.T) {
	assert.Equal(t, "Start", mysqlStateStart.String())
	assert.Equal(t, "EatMessage", mysqlStateEatMessage.String())
	assert.Equal(t, "EatFields", mysqlStateEatFields.String())
	assert.Equal(t, "EatRows", mysqlStateEatRows.String())

	assert.NotNil(t, (mysqlStateMax - 1).String())
}

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
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage), isClient: true}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if !stream.message.isRequest {
		t.Errorf("Failed to parse MySQL request")
	}
	if stream.message.query != "INSERT INTO post (username, title, body, pub_date) VALUES ('Anonymous', 'test', 'test', '2013-07-22 18:44:17')" {
		t.Errorf("Failed to parse query")
	}

	if stream.message.size != 115 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}
func TestMySQLParser_OKResponse(t *testing.T) {
	data := []byte(
		"0700000100010401000000")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Errorf("Failed to parse MySQL response")
	}
	if !stream.message.isOK {
		t.Errorf("Failed to parse Response OK")
	}
	if stream.message.affectedRows != 1 {
		t.Errorf("Failed to parse affected rows")
	}
	if stream.message.insertID != 4 {
		t.Errorf("Failed to parse last INSERT id")
	}
	if stream.message.size != 11 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

func TestMySQLParser_errorResponse(t *testing.T) {
	data := []byte(
		"2e000001ff7a042334325330325461626c6520276d696e69747769742e706f737373742720646f65736e2774206578697374")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Errorf("Failed to parse MySQL response")
	}
	if stream.message.isOK {
		t.Errorf("Failed to parse MySQL error esponse")
	}

	if stream.message.size != 50 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

func TestMySQLParser_dataResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysqldetailed"))
	mysql := mysqlModForTests(nil)

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
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.isOK || stream.message.isError {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if stream.message.tables != "minitwit.post" {
		t.Errorf("Failed to get table name: %s", stream.message.tables)
	}
	if stream.message.numberOfFields != 5 {
		t.Errorf("Failed to get the number of fields")
	}
	if stream.message.numberOfRows != 4 {
		t.Errorf("Failed to get the number of rows")
	}

	// parse fields and rows
	raw := stream.data[stream.message.start:stream.message.end]
	if len(raw) == 0 {
		t.Errorf("Empty raw data")
	}
	fields, rows := mysql.parseMysqlResponse(raw)
	if len(fields) != stream.message.numberOfFields {
		t.Errorf("Failed to parse the fields")
	}
	if len(rows) != stream.message.numberOfRows {
		t.Errorf("Failed to parse the rows")
	}
	if stream.message.size != 528 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

func TestMySQLParser_simpleUpdateResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysqldetailed"))

	data := []byte("300000010001000100000028526f7773206d6174636865643a203120204368616e6765643a203120205761726e696e67733a2030")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.isOK || stream.message.isError {
		t.Errorf("Failed to true, true, parse MySQL Query response")
	}
	if stream.message.affectedRows != 1 {
		t.Errorf("Failed to get the number of affected rows")
	}
	if stream.message.size != 52 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

func TestMySQLParser_simpleUpdateResponseSplit(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	data1 := "300000010001000100000028526f7773206d6174636865"
	data2 := "643a203120204368616e6765643a"
	data3 := "203120205761726e696e67733a2030"

	message, err := hex.DecodeString(string(data1))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &mysqlStream{data: message, message: new(mysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	message, err = hex.DecodeString(data2)
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}
	stream.data = append(stream.data, message...)
	ok, complete = mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}

	if complete {
		t.Errorf("Not expecting a complete message yet")
	}

	message, err = hex.DecodeString(data3)
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}
	stream.data = append(stream.data, message...)
	ok, complete = mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.isRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.isOK || stream.message.isError {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if stream.message.affectedRows != 1 {
		t.Errorf("Failed to get the number of affected rows")
	}
	if stream.message.size != 52 {
		t.Errorf("Wrong message size %d", stream.message.size)
	}
}

func TestParseMySQL_simpleUpdateResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	mysql := mysqlModForTests(nil)
	data, err := hex.DecodeString("300000010001000100000028526f7773206d61746368" +
		"65643a203120204368616e6765643a203120205761726e696e67733a2030")
	if err != nil {
		t.Errorf("Failed to decode string")
	}
	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Errorf("Failed to get ts")
	}
	pkt := protos.Packet{
		Payload: data,
		Ts:      ts,
	}
	var tuple common.TCPTuple
	var private mysqlPrivateData

	var countHandleMysql = 0

	mysql.handleMysql = func(mysql *mysqlPlugin, m *mysqlMessage, tcp *common.TCPTuple,
		dir uint8, raw_msg []byte) {

		countHandleMysql++
	}

	mysql.Parse(&pkt, &tuple, tcp.TCPDirectionOriginal, private)

	if countHandleMysql != 1 {
		t.Errorf("handleMysql not called")
	}
}

// Test parsing three OK responses in the same packet
func TestParseMySQL_threeResponses(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	mysql := mysqlModForTests(nil)

	data, err := hex.DecodeString(
		"0700000100000000000000" +
			// second message
			"0700000100000000000000" +
			// third message
			"0700000100000000000000")
	if err != nil {
		t.Errorf("Failed to decode string")
	}
	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Errorf("Failed to get ts")
	}
	pkt := protos.Packet{
		Payload: data,
		Ts:      ts,
	}
	var tuple common.TCPTuple
	var private mysqlPrivateData

	var countHandleMysql = 0

	mysql.handleMysql = func(mysql *mysqlPlugin, m *mysqlMessage, tcptuple *common.TCPTuple,
		dir uint8, raw_msg []byte) {

		countHandleMysql++
	}

	mysql.Parse(&pkt, &tuple, tcp.TCPDirectionOriginal, private)

	if countHandleMysql != 3 {
		t.Errorf("handleMysql not called three times")
	}
}

// Test parsing one response split in two packets
func TestParseMySQL_splitResponse(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	mysql := mysqlModForTests(nil)

	data, err := hex.DecodeString(
		"0100000105" +
			"2f00000203646566086d696e697477697404706f737404706f737407706f73745f69640269640c3f000b000000030342000000" +
			"3b00000303646566086d696e697477697404706f737404706f73740d706f73745f757365726e616d6508757365726e616d650c2100f0000000fd0000000000" +
			"3500000403646566086d696e697477697404706f737404706f73740a706f73745f7469746c65057469746c650c2100f0000000fd0000000000" +
			"3300000503646566086d696e697477697404706f737404706f737409706f73745f626f647904626f64790c2100fdff0200fc1000000000")

	if err != nil {
		t.Errorf("Failed to decode string")
	}
	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Errorf("Failed to get ts")
	}
	pkt := protos.Packet{
		Payload: data,
		Ts:      ts,
	}
	var tuple common.TCPTuple
	var private mysqlPrivateData

	var countHandleMysql = 0

	mysql.handleMysql = func(mysql *mysqlPlugin, m *mysqlMessage, tcptuple *common.TCPTuple,
		dir uint8, raw_msg []byte) {

		countHandleMysql++
	}

	private = mysql.Parse(&pkt, &tuple, tcp.TCPDirectionOriginal, private).(mysqlPrivateData)
	if countHandleMysql != 0 {
		t.Errorf("handleMysql called on first run")
	}

	// now second fragment

	data, err = hex.DecodeString(
		"3b00000603646566086d696e697477697404706f737404706f73740d706f73745f7075625f64617465087075625f646174650c3f00130000000c8000000000" +
			"05000007fe00002100" +
			"2e000008013109416e6f6e796d6f75730474657374086461736461730d0a13323031332d30372d32322031373a33343a3032" +
			"46000009013209416e6f6e796d6f757312506f737465617a6120544f444f206c6973741270656e7472752063756d706172617475726913323031332d30372d32322031383a32393a3330" +
			"2a00000a013309416e6f6e796d6f75730454657374047465737413323031332d30372d32322031383a33323a3130" +
			"2a00000b013409416e6f6e796d6f75730474657374047465737413323031332d30372d32322031383a34343a3137" +
			"0500000cfe00002100")
	if err != nil {
		t.Fatal(err)
	}

	pkt = protos.Packet{
		Payload: data,
		Ts:      ts,
	}

	mysql.Parse(&pkt, &tuple, 1, private)
	if countHandleMysql != 1 {
		t.Errorf("handleMysql not called on the second run")
	}
}

func testTCPTuple() *common.TCPTuple {
	t := &common.TCPTuple{
		IPLength: 4,
		BaseTuple: common.BaseTuple{
			SrcIP: net.IPv4(192, 168, 0, 1), DstIP: net.IPv4(192, 168, 0, 2),
			SrcPort: 6512, DstPort: serverPort,
		},
	}
	t.ComputeHashables()
	return t
}

// Helper function to read from the Publisher Queue
func expectTransaction(t *testing.T, e *eventStore) common.MapStr {
	if len(e.events) == 0 {
		t.Error("No transaction")
		return nil
	}

	event := e.events[0]
	e.events = e.events[1:]
	return event.Fields
}

// Test that loss of data during the response (but not at the beginning)
// don't cause the whole transaction to be dropped.
func Test_gap_in_response(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	store := &eventStore{}
	mysql := mysqlModForTests(store)

	// request and response from tests/pcaps/mysql_result_long.pcap
	// select * from test
	reqData, err := hex.DecodeString(
		"130000000373656c656374202a20" +
			"66726f6d2074657374")
	assert.Nil(t, err)
	respData, err := hex.DecodeString(
		"0100000103240000020364656604" +
			"74657374047465737404746573740161" +
			"01610c3f000b00000003000000000024" +
			"00000303646566047465737404746573" +
			"740474657374016201620c3f000b0000" +
			"00030000000000240000040364656604" +
			"74657374047465737404746573740163" +
			"01630c2100fd020000fd000000000005" +
			"000005fe000022000a00000601310131" +
			"0548656c6c6f0a000007013201320548" +
			"656c6c6f0601000801330133fcff004c" +
			"6f72656d20497073756d206973207369" +
			"6d706c792064756d6d79207465787420" +
			"6f6620746865207072696e74696e6720" +
			"616e64207479706573657474696e6720" +
			"696e6475737472792e204c6f72656d20")
	assert.Nil(t, err)

	tcptuple := testTCPTuple()
	req := protos.Packet{Payload: reqData}
	resp := protos.Packet{Payload: respData}

	private := protos.ProtocolData(new(mysqlPrivateData))

	private = mysql.Parse(&req, tcptuple, tcp.TCPDirectionOriginal, private)
	private = mysql.Parse(&resp, tcptuple, tcp.TCPDirectionReverse, private)

	logp.Debug("mysql", "Now sending gap..")

	_, drop := mysql.GapInStream(tcptuple, tcp.TCPDirectionReverse, 10, private)
	assert.Equal(t, true, drop)

	trans := expectTransaction(t, store)
	if m, err := trans.GetValue("error.message"); assert.NoError(t, err) {
		assert.Equal(t, m, "Packet loss while capturing the response")
	}
}

// Test that loss of data during the request doesn't result in a
// published transaction.
func Test_gap_in_eat_message(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	mysql := mysqlModForTests(nil)

	// request from tests/pcaps/mysql_result_long.pcap
	// "select * from test". Last byte missing.
	reqData, err := hex.DecodeString(
		"130000000373656c656374202a20" +
			"66726f6d20746573")
	assert.Nil(t, err)

	stream := &mysqlStream{data: reqData, message: new(mysqlMessage), isClient: true}
	ok, complete := mysqlMessageParser(stream)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, complete)

	complete = mysql.messageGap(stream, 10)
	assert.Equal(t, false, complete)
}

func Test_read_length(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	var err error
	var length int

	_, err = readLength([]byte{}, 0)
	assert.NotNil(t, err)

	_, err = readLength([]byte{0x00, 0x00}, 0)
	assert.NotNil(t, err)

	length, err = readLength([]byte{0x01, 0x00, 0x00}, 0)
	assert.Nil(t, err)
	assert.Equal(t, length, 1)
}

func Test_parseMysqlResponse_invalid(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))

	mysql := mysqlModForTests(nil)

	tests := [][]byte{
		{},
		{0x00, 0x00},
		{0x00, 0x00, 0x00},
		{0x05, 0x00, 0x00},
		{0x05, 0x00, 0x00, 0x01},
		{0x05, 0x00, 0x00, 0x01, 0x01},
		{0x05, 0x00, 0x00, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0xff},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00},
		{0x05, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00,
			0x01, 0x00},
		{0x15, 0x00, 0x00, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00,
			0x01, 0x00, 0x01},
		{0x15, 0x00, 0x00, 0x01, 0x01, 0x05, 0x15, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00,
			0x01, 0x00, 0x01, 0x00},
	}

	for _, input := range tests {
		fields, rows := mysql.parseMysqlResponse(input)
		assert.Equal(t, []string{}, fields)
		assert.Equal(t, [][]string{}, rows)
	}

	tests = [][]byte{
		{0x15, 0x00, 0x00, 0x01, 0x01,
			0x0b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0xfe, 0x00, 0x01, //field
			0x01, 0x00, 0x00, 0x00, 0xfe, // EOF
		},
		{0x15, 0x00, 0x00, 0x01, 0x01,
			0x0b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0xfe, 0x00, 0x01, //field
			0x01, 0x00, 0x00, 0x00, 0xfe, // EOF
			0x00, 0x00,
		},
	}

	for _, input := range tests {
		fields, rows := mysql.parseMysqlResponse(input)
		assert.Equal(t, []string{""}, fields)
		assert.Equal(t, [][]string{}, rows)
	}
}

func Test_PreparedStatement(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("mysql", "mysqldetailed"))
	tcpTuple := testTCPTuple()
	results := &eventStore{}
	mysql := mysqlModForTests(results)

	send := func(dir uint8, data string) {
		rawData, err := hex.DecodeString(data)
		assert.Nil(t, err)
		packet := protos.Packet{Payload: rawData}

		var private protos.ProtocolData
		private = mysql.Parse(&packet, tcpTuple, dir, private)
	}

	send(tcp.TCPDirectionOriginal, "c00000001673656c6563742064697374696e637420636f756e742864697374696e63742070757263686173656465305f2e69642920617320636f6c5f305f305f2066726f6d2070757263686173655f64656d616e642070757263686173656465305f2077686572652070757263686173656465305f2e636861696e5f6d61737465723d3f20616e642070757263686173656465305f2e6372656174655f74696d653e3d3f20616e642070757263686173656465305f2e6372656174655f74696d653c3d3f")
	send(tcp.TCPDirectionReverse, "0c000001000b000000010003000000001700000203646566000000013f000c3f0000000000fd80000000001700000303646566000000013f000c3f0000000000fd80000000001700000403646566000000013f000c3f0000000000fd800000000005000005fe000001201e0000060364656600000008636f6c5f305f305f000c3f001500000008810000000005000007fe00000120")
	send(tcp.TCPDirectionOriginal, "33000000170b00000000010000000001fd000c000c000841313232343633380be107071c000000000000000be1070a1c173b3b00000000")
	send(tcp.TCPDirectionReverse, "01000001011e0000020364656600000008636f6c5f305f305f000c3f001500000008810000000005000003fe000001200a00000400000b0000000000000005000005fe00000120")
	assert.Len(t, results.events, 2)
}
