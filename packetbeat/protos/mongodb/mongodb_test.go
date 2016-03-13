// +build !integration

package mongodb

import (
	"encoding/hex"
	"net"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/publish"
	"github.com/stretchr/testify/assert"
)

// Helper function returning a Mongodb module that can be used
// in tests. It publishes the transactions in the results channel.
func MongodbModForTests() *Mongodb {
	var mongodb Mongodb
	results := &publish.ChanTransactions{make(chan common.MapStr, 10)}
	config := defaultConfig
	mongodb.init(results, &config)
	return &mongodb
}

// Helper function that returns an example TcpTuple
func testTcpTuple() *common.TcpTuple {
	t := &common.TcpTuple{
		Ip_length: 4,
		Src_ip:    net.IPv4(192, 168, 0, 1), Dst_ip: net.IPv4(192, 168, 0, 2),
		Src_port: 6512, Dst_port: 27017,
	}
	t.ComputeHashebles()
	return t
}

// Helper function to read from the results Queue. Raises
// an error if nothing is found in the queue.
func expectTransaction(t *testing.T, mongodb *Mongodb) common.MapStr {
	client := mongodb.results.(*publish.ChanTransactions)
	select {
	case trans := <-client.Channel:
		return trans
	default:
		t.Error("No transaction")
	}
	return nil
}

// Test simple request / response.
func TestSimpleFindLimit1(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"mongodb", "mongodbdetailed"})
	}

	mongodb := MongodbModForTests()

	// request and response from tests/pcaps/mongo_one_row.pcap
	req_data, err := hex.DecodeString(
		"320000000a000000ffffffffd4070000" +
			"00000000746573742e72667374617572" +
			"616e7473000000000001000000050000" +
			"0000")
	assert.Nil(t, err)
	resp_data, err := hex.DecodeString(
		"020200004a0000000a00000001000000" +
			"08000000000000000000000000000000" +
			"01000000de010000075f696400558beb" +
			"b45f075665d2ae862703616464726573" +
			"730069000000026275696c64696e6700" +
			"05000000313030370004636f6f726400" +
			"1b000000013000e6762ff7c97652c001" +
			"3100d5b14ae9996c4440000273747265" +
			"657400100000004d6f72726973205061" +
			"726b2041766500027a6970636f646500" +
			"060000003130343632000002626f726f" +
			"756768000600000042726f6e78000263" +
			"756973696e65000700000042616b6572" +
			"79000467726164657300eb0000000330" +
			"002b00000009646174650000703d8544" +
			"01000002677261646500020000004100" +
			"1073636f72650002000000000331002b" +
			"0000000964617465000044510a410100" +
			"00026772616465000200000041001073" +
			"636f72650006000000000332002b0000" +
			"00096461746500009cda693c01000002" +
			"6772616465000200000041001073636f" +
			"7265000a000000000333002b00000009" +
			"646174650000ccb8cd33010000026772" +
			"616465000200000041001073636f7265" +
			"0009000000000334002b000000096461" +
			"7465000014109d2e0100000267726164" +
			"65000200000042001073636f7265000e" +
			"0000000000026e616d6500160000004d" +
			"6f72726973205061726b2042616b6520" +
			"53686f70000272657374617572616e74" +
			"5f696400090000003330303735343435" +
			"0000")
	assert.Nil(t, err)

	tcptuple := testTcpTuple()
	req := protos.Packet{Payload: req_data}
	resp := protos.Packet{Payload: resp_data}

	private := protos.ProtocolData(new(mongodbConnectionData))

	private = mongodb.Parse(&req, tcptuple, 0, private)
	private = mongodb.Parse(&resp, tcptuple, 1, private)
	trans := expectTransaction(t, mongodb)

	assert.Equal(t, "OK", trans["status"])
	assert.Equal(t, "find", trans["method"])
	assert.Equal(t, "mongodb", trans["type"])

	logp.Debug("mongodb", "Trans: %v", trans)
}

// Test simple request / response, where the response is split in
// 3 messages
func TestSimpleFindLimit1_split(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"mongodb", "mongodbdetailed"})
	}

	mongodb := MongodbModForTests()
	mongodb.SendRequest = true
	mongodb.SendResponse = true

	// request and response from tests/pcaps/mongo_one_row.pcap
	req_data, err := hex.DecodeString(
		"320000000a000000ffffffffd4070000" +
			"00000000746573742e72667374617572" +
			"616e7473000000000001000000050000" +
			"0000")
	assert.Nil(t, err)
	resp_data1, err := hex.DecodeString(
		"020200004a0000000a00000001000000" +
			"08000000000000000000000000000000" +
			"01000000de010000075f696400558beb" +
			"b45f075665d2ae862703616464726573" +
			"730069000000026275696c64696e6700" +
			"05000000313030370004636f6f726400" +
			"1b000000013000e6762ff7c97652c001" +
			"3100d5b14ae9996c4440000273747265" +
			"657400100000004d6f72726973205061")

	resp_data2, err := hex.DecodeString(
		"726b2041766500027a6970636f646500" +
			"060000003130343632000002626f726f" +
			"756768000600000042726f6e78000263" +
			"756973696e65000700000042616b6572" +
			"79000467726164657300eb0000000330" +
			"002b00000009646174650000703d8544" +
			"01000002677261646500020000004100" +
			"1073636f72650002000000000331002b" +
			"0000000964617465000044510a410100" +
			"00026772616465000200000041001073" +
			"636f72650006000000000332002b0000")

	resp_data3, err := hex.DecodeString(
		"00096461746500009cda693c01000002" +
			"6772616465000200000041001073636f" +
			"7265000a000000000333002b00000009" +
			"646174650000ccb8cd33010000026772" +
			"616465000200000041001073636f7265" +
			"0009000000000334002b000000096461" +
			"7465000014109d2e0100000267726164" +
			"65000200000042001073636f7265000e" +
			"0000000000026e616d6500160000004d" +
			"6f72726973205061726b2042616b6520" +
			"53686f70000272657374617572616e74" +
			"5f696400090000003330303735343435" +
			"0000")
	assert.Nil(t, err)

	tcptuple := testTcpTuple()
	req := protos.Packet{Payload: req_data}

	private := protos.ProtocolData(new(mongodbConnectionData))

	private = mongodb.Parse(&req, tcptuple, 0, private)

	resp1 := protos.Packet{Payload: resp_data1}
	private = mongodb.Parse(&resp1, tcptuple, 1, private)

	resp2 := protos.Packet{Payload: resp_data2}
	private = mongodb.Parse(&resp2, tcptuple, 1, private)

	resp3 := protos.Packet{Payload: resp_data3}
	private = mongodb.Parse(&resp3, tcptuple, 1, private)

	trans := expectTransaction(t, mongodb)

	assert.Equal(t, "OK", trans["status"])
	assert.Equal(t, "find", trans["method"])
	assert.Equal(t, "mongodb", trans["type"])

	logp.Debug("mongodb", "Trans: %v", trans)
}

func TestReconstructQuery(t *testing.T) {
	type io struct {
		Input  transaction
		Full   bool
		Output string
	}
	tests := []io{
		{
			Input: transaction{
				resource: "test.col",
				method:   "find",
				event: map[string]interface{}{
					"numberToSkip":   3,
					"numberToReturn": 2,
				},
				params: map[string]interface{}{
					"me": "you",
				},
			},
			Full:   true,
			Output: `test.col.find({"me":"you"}).skip(3).limit(2)`,
		},
		{
			Input: transaction{
				resource: "test.col",
				method:   "insert",
				params: map[string]interface{}{
					"documents": "you",
				},
			},
			Full:   true,
			Output: `test.col.insert({"documents":"you"})`,
		},
		{
			Input: transaction{
				resource: "test.col",
				method:   "insert",
				params: map[string]interface{}{
					"documents": "you",
				},
			},
			Full:   false,
			Output: `test.col.insert({})`,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.Output,
			reconstructQuery(&test.Input, test.Full))
	}
}

// max_docs option should be respected
func TestMaxDocs(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"mongodb", "mongodbdetailed"})
	}

	// more docs than configured
	trans := transaction{
		documents: []interface{}{
			1, 2, 3, 4, 5, 6, 7, 8,
		},
	}

	mongodb := MongodbModForTests()
	mongodb.SendResponse = true
	mongodb.MaxDocs = 3

	mongodb.publishTransaction(&trans)

	res := expectTransaction(t, mongodb)

	assert.Equal(t, "1\n2\n3\n[...]", res["response"])

	// exactly the same number of docs
	trans = transaction{
		documents: []interface{}{
			1, 2, 3,
		},
	}

	mongodb.publishTransaction(&trans)
	res = expectTransaction(t, mongodb)
	assert.Equal(t, "1\n2\n3", res["response"])

	// less docs
	trans = transaction{
		documents: []interface{}{
			1, 2,
		},
	}

	mongodb.publishTransaction(&trans)
	res = expectTransaction(t, mongodb)
	assert.Equal(t, "1\n2", res["response"])

	// unlimited
	trans = transaction{
		documents: []interface{}{
			1, 2, 3, 4,
		},
	}
	mongodb.MaxDocs = 0
	mongodb.publishTransaction(&trans)
	res = expectTransaction(t, mongodb)
	assert.Equal(t, "1\n2\n3\n4", res["response"])
}

func TestMaxDocSize(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"mongodb", "mongodbdetailed"})
	}

	// more docs than configured
	trans := transaction{
		documents: []interface{}{
			"1234567",
			"123",
			"12",
		},
	}

	mongodb := MongodbModForTests()
	mongodb.SendResponse = true
	mongodb.MaxDocLength = 5

	mongodb.publishTransaction(&trans)

	res := expectTransaction(t, mongodb)

	assert.Equal(t, "\"1234 ...\n\"123\"\n\"12\"", res["response"])
}
