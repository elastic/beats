package gobeacon

import (
	"errors"
	"net/url"
	"packetbeat/common"
	"packetbeat/logp"
	"reflect"
	"testing"

	"github.com/gdamore/mangos/protocol/req"
	"github.com/gdamore/mangos/transport/all"
	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"
)

func sendMessage(url string, message *[]byte) error {

	requestSocket, err := req.NewSocket()
	if err != nil {
		return err
	}
	defer requestSocket.Close()
	all.AddTransports(requestSocket)

	if err = requestSocket.Dial(url); err != nil {
		return err
	}

	if err = requestSocket.Send(*message); err != nil {
		return err
	}

	var clientMsg []byte

	if clientMsg, err = requestSocket.Recv(); err != nil {
		return err
	}

	if string(clientMsg) != "OK" {
		return errors.New("Response not OK, requeued")
	}

	return nil

}

func encodeMessage(query url.Values) ([]byte, error) {
	var b []byte
	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string][]string(nil))

	enc := codec.NewEncoderBytes(&b, &mh)
	err := enc.Encode(query)
	if err != nil {
		return nil, err
	}
	return b, nil

}

func TestGoBeacon(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"gobeacon"})
	}

	events := make(chan common.MapStr)
	server := new(GoBeacon)

	server.Config = Config{
		ListenAddr: "tcp://127.0.0.1:9000",
		Tracker:    "boomerang",
	}
	err := server.Init(true, events)
	assert.Nil(t, err)

	ready := make(chan bool)

	go func() {
		ready <- true
		err := server.Run()
		assert.Nil(t, err, "Error: %v", err)
	}()

	// make sure the goroutine runs first
	<-ready

	// send a message
	v := url.Values{}
	v.Set("nt_nav_type", "1")
	v.Add("nt_domcontloaded_st", "1426674694041")
	v.Add("nt_domcontloaded_end", "1426674694041")
	v.Add("u", "http://localhost:8080/static/")
	v.Add("r", "")
	v.Add("nt_domcomp", "1426674694056")
	v.Add("nt_dns_st", "1426674694014")
	v.Add("nt_dns_end", "1426675597491")
	v.Add("rt.bstart", "1426674694048")
	v.Add("rt.end", "1426674694057")
	v.Add("nt_con_st", "1426675597491")
	v.Add("nt_con_end", "1426675597491")

	obj, err := encodeMessage(v)
	err = sendMessage(server.Config.ListenAddr, &obj)
	assert.Nil(t, err, "Error: %v", err)

	server.Stop()
}
