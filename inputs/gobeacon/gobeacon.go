package gobeacon

import (
	"packetbeat/common"
	"packetbeat/config"
	"packetbeat/logp"
	"reflect"
	"strconv"
	"time"

	"github.com/gdamore/mangos"
	"github.com/gdamore/mangos/protocol/rep"
	"github.com/gdamore/mangos/transport/ipc"
	"github.com/gdamore/mangos/transport/tcp"
	"github.com/ugorji/go/codec"
)

type Config struct {
	ListenAddr string
	Tracker    string
}
type GoBeacon struct {
	Config

	events  chan common.MapStr
	isAlive bool
	conn    mangos.Socket
	mh      codec.MsgpackHandle
}

func (server *GoBeacon) decode(buf []byte) (map[string][]string, error) {

	doc := map[string][]string(nil)
	dec := codec.NewDecoderBytes(buf, &server.mh)
	err := dec.Decode(&doc)

	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Calculate delta between start and end
func delta(start string, end string) (int, error) {
	s, err := strconv.Atoi(start)
	if err != nil {
		return -1, err
	}
	e, err := strconv.Atoi(end)
	if err != nil {
		return -1, err
	}
	return e - s, nil
}

func boomerangMetrics(d map[string][]string) (common.MapStr, error) {
	defer logp.Recover("boomerangMetrics exception")

	ntDNS, _ := delta(d["nt_dns_st"][0], d["nt_dns_end"][0])                               // domainLookupEnd - domainLookupStart
	ntCon, _ := delta(d["nt_con_st"][0], d["nt_con_end"][0])                               // connectEnd - connectStart
	ntDomcontloaded, _ := delta(d["nt_domcontloaded_st"][0], d["nt_domcontloaded_end"][0]) // domContentLoadedEnd - domContentLoadedStart
	ntProcessed, _ := delta(d["nt_domcontloaded_st"][0], d["nt_domcomp"][0])               // domComplete - domContentLoadedStart
	ntNavtype := d["nt_nav_type"][0]                                                       //boolean
	roundtrip, _ := delta(d["rt.bstart"][0], d["rt.end"][0])
	page := d["r"][0]
	url := d["u"][0]

	metrics := common.MapStr{
		"type":         "RUM",
		"count":        1,
		"path":         url,
		"dnstime":      ntDNS,
		"connecttime":  ntCon,
		"responsetime": roundtrip,
	}

	metrics["rum"] = common.MapStr{
		"page":           page,
		"navigationtype": ntNavtype,
		"domloadtime":    ntDomcontloaded,
		"domprocesstime": ntProcessed,
	}

	return metrics, nil
}

func jsMetrics(d map[string][]string) (common.MapStr, error) {
	defer logp.Recover("jsMetrics exception")

	ntDNS, _ := delta(d["nt_dns_st"][0], d["nt_dns_end"][0])                               // domainLookupEnd - domainLookupStart
	ntCon, _ := delta(d["nt_con_st"][0], d["nt_con_end"][0])                               // connectEnd - connectStart
	ntDomcontloaded, _ := delta(d["nt_domcontloaded_st"][0], d["nt_domcontloaded_end"][0]) // domContentLoadedEnd - domContentLoadedStart
	ntProcessed, _ := delta(d["nt_domcontloaded_st"][0], d["nt_domcomp"][0])               // domComplete - domContentLoadedStart
	ntNavtype := d["nt_nav_type"][0]
	roundtrip, _ := delta(d["rt.bstart"][0], d["rt.end"][0])
	page := d["r"][0]
	url := d["u"][0]

	metrics := common.MapStr{
		"type":         "RUM",
		"count":        1,
		"path":         url,
		"dnstime":      ntDNS,
		"connecttime":  ntCon,
		"responsetime": roundtrip,
	}

	metrics["rum"] = common.MapStr{
		"page":           page,
		"navigationtype": ntNavtype,
		"domloadtime":    ntDomcontloaded,
		"domprocesstime": ntProcessed,
	}

	return metrics, nil
}

func (server *GoBeacon) Run() error {

	for server.isAlive {

		serverMsg, err := server.conn.RecvMsg()
		if err != nil {
			logp.Err("Server receive failed: %v", err)
			return err
		}
		d, err := server.decode(serverMsg.Body)
		if len(d) < 1 {
			logp.Err("Discarded message")
			continue
		}
		logp.Debug("gobeacon", "Read from socket: %s", d)

		var obj common.MapStr
		switch server.Config.Tracker {
		case "boomerang":
			obj, _ = boomerangMetrics(d)
		case "js":
			obj, _ = jsMetrics(d)
		}

		serverMsg.Body = []byte("OK")
		err = server.conn.SendMsg(serverMsg)
		if err != nil {
			logp.Err("Server send failed: %v", err)
			return err
		}

		err = obj.EnsureTimestampField(time.Now)
		if err != nil {
			logp.Err("Invalid timestamp field: %v", err)
			continue
		}

		server.events <- obj
	}
	return nil
}

func (server *GoBeacon) setFromConfig() error {
	var cfg Config

	if len(config.ConfigSingleton.GoBeacon.Listen_addr) > 0 {
		cfg.ListenAddr = config.ConfigSingleton.GoBeacon.Listen_addr
	} else {
		cfg.ListenAddr = "tcp://127.0.0.1:8000"
	}
	if len(config.ConfigSingleton.GoBeacon.Tracker) > 0 {
		cfg.Tracker = config.ConfigSingleton.GoBeacon.Tracker
	} else {
		cfg.Tracker = "boomerang"
	}

	server.Config = cfg
	return nil
}

func (server *GoBeacon) Init(test_mode bool, events chan common.MapStr) error {

	if !test_mode {
		err := server.setFromConfig()
		if err != nil {
			return err
		}
	}

	server.events = events

	var err error
	server.conn, err = rep.NewSocket()
	if err != nil {
		return err
	}
	server.conn.AddTransport(ipc.NewTransport())
	server.conn.AddTransport(tcp.NewTransport())

	err = server.conn.Listen(server.Config.ListenAddr)
	if err != nil {
		return err
	}

	server.mh.MapType = reflect.TypeOf(map[string][]string(nil))

	server.isAlive = true

	logp.Info("GoBeacon plugin listening on %s. Using tracker %s.", server.Config.ListenAddr, server.Config.Tracker)

	return nil
}

func (server *GoBeacon) Stop() error {
	server.isAlive = false
	return nil
}

func (server *GoBeacon) Close() error {
	return server.conn.Close()
}

func (server *GoBeacon) IsAlive() bool {
	return server.isAlive
}
