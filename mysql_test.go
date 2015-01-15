package main

import (
	"encoding/hex"
	"testing"
	//"fmt"
	"time"
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
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if !stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL request")
	}
	if stream.message.Query != "INSERT INTO post (username, title, body, pub_date) VALUES ('Anonymous', 'test', 'test', '2013-07-22 18:44:17')" {
		t.Errorf("Failed to parse query")
	}

}
func TestMySQLParser_OKResponse(t *testing.T) {

	data := []byte(
		"0700000100010401000000")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL response")
	}
	if !stream.message.IsOK {
		t.Errorf("Failed to parse Response OK")
	}
	if stream.message.AffectedRows != 1 {
		t.Errorf("Failed to parse affected rows")
	}
	if stream.message.InsertId != 4 {
		t.Errorf("Failed to parse last INSERT id")
	}
}

func TestMySQLParser_errorResponse(t *testing.T) {

	data := []byte(
		"2e000001ff7a042334325330325461626c6520276d696e69747769742e706f737373742720646f65736e2774206578697374")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL response")
	}
	if stream.message.IsOK {
		t.Errorf("Failed to parse MySQL error esponse")
	}

}

func TestMySQLParser_dataResponse(t *testing.T) {
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
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.IsOK || stream.message.IsError {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if stream.message.Size != 528 {
		t.Errorf("Failed to get the size of the message")
	}
	if stream.message.Tables != "minitwit.post" {
		t.Errorf("Failed to get table name: %s", stream.message.Tables)
	}
	if stream.message.NumberOfFields != 5 {
		t.Errorf("Failed to get the number of fields")
	}
	if stream.message.NumberOfRows != 4 {
		t.Errorf("Failed to get the number of rows")
	}
	if stream.message.Size != 528 {
		t.Errorf("failed to get the size of the response")
	}

	// parse fields and rows
	raw := stream.data[stream.message.start:stream.message.end]
	if len(raw) == 0 {
		t.Errorf("Empty raw data")
	}
	fields, rows := parseMysqlResponse(raw)
	if len(fields) != stream.message.NumberOfFields {
		t.Errorf("Failed to parse the fields")
	}
	if len(rows) != stream.message.NumberOfRows {
		t.Errorf("Failed to parse the rows")
	}
}

func TestMySQLParser_simpleUpdateResponse(t *testing.T) {
	//LogInit(syslog.LOG_DEBUG, "" /*toSyslog*/, false, []string{"mysqldetailed"})

	data := []byte("300000010001000100000028526f7773206d6174636865643a203120204368616e6765643a203120205761726e696e67733a2030")

	message, err := hex.DecodeString(string(data))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

	ok, complete := mysqlMessageParser(stream)

	if !ok {
		t.Errorf("Parsing returned error")
	}
	if !complete {
		t.Errorf("Expecting a complete message")
	}
	if stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.IsOK || stream.message.IsError {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if stream.message.AffectedRows != 1 {
		t.Errorf("Failed to get the number of affected rows")
	}
}

func TestMySQLParser_simpleUpdateResponseSplit(t *testing.T) {
	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"mysql", "mysqldetailed"})
	}

	data1 := "300000010001000100000028526f7773206d6174636865"
	data2 := "643a203120204368616e6765643a"
	data3 := "203120205761726e696e67733a2030"

	message, err := hex.DecodeString(string(data1))
	if err != nil {
		t.Errorf("Failed to decode hex string")
	}

	stream := &MysqlStream{tcpStream: nil, data: message, message: new(MysqlMessage)}

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
	if stream.message.IsRequest {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if !stream.message.IsOK || stream.message.IsError {
		t.Errorf("Failed to parse MySQL Query response")
	}
	if stream.message.AffectedRows != 1 {
		t.Errorf("Failed to get the number of affected rows")
	}
}

func TestParseMySQL_simpleUpdateResponse(t *testing.T) {
	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"mysql", "mysqldetailed"})
	}

	data, err := hex.DecodeString("300000010001000100000028526f7773206d61746368" +
		"65643a203120204368616e6765643a203120205761726e696e67733a2030")
	if err != nil {
		t.Errorf("Failed to decode string")
	}
	ts, err := time.Parse(time.RFC3339, "2000-12-26T01:15:06+04:20")
	if err != nil {
		t.Errorf("Failed to get ts")
	}
	pkt := Packet{
		payload: data,
		ts:      ts,
	}
	tcp := TcpStream{
		mysqlData: [2]*MysqlStream{nil, nil},
	}

	var count_handleMysql = 0

	handleMysql = func(m *MysqlMessage, tcp *TcpStream,
		dir uint8, raw_msg []byte) {

		count_handleMysql += 1
	}

	ParseMysql(&pkt, &tcp, 1)

	if count_handleMysql != 1 {
		t.Errorf("handleMysql not called")
	}
}

// Test parsing three OK responses in the same packet
func TestParseMySQL_threeResponses(t *testing.T) {
	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"mysql", "mysqldetailed"})
	}

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
	pkt := Packet{
		payload: data,
		ts:      ts,
	}
	tcp := TcpStream{
		mysqlData: [2]*MysqlStream{nil, nil},
	}

	var count_handleMysql = 0

	old_handleMysql := handleMysql
	defer func() {
		handleMysql = old_handleMysql
	}()
	handleMysql = func(m *MysqlMessage, tcp *TcpStream,
		dir uint8, raw_msg []byte) {

		count_handleMysql += 1
	}

	ParseMysql(&pkt, &tcp, 1)

	if count_handleMysql != 3 {
		t.Errorf("handleMysql not called three times")
	}
}

// Test parsing one response split in two packets
func TestParseMySQL_splitResponse(t *testing.T) {
	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"mysql", "mysqldetailed"})
	}

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
	pkt := Packet{
		payload: data,
		ts:      ts,
	}
	tcp := TcpStream{
		mysqlData: [2]*MysqlStream{nil, nil},
	}

	var count_handleMysql = 0

	old_handleMysql := handleMysql
	defer func() {
		handleMysql = old_handleMysql
	}()
	handleMysql = func(m *MysqlMessage, tcp *TcpStream,
		dir uint8, raw_msg []byte) {

		count_handleMysql += 1
	}

	ParseMysql(&pkt, &tcp, 1)
	if count_handleMysql != 0 {
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

	pkt = Packet{
		payload: data,
		ts:      ts,
	}

	ParseMysql(&pkt, &tcp, 1)
	if count_handleMysql != 1 {
		t.Errorf("handleMysql not called on the second run")
	}
}
