package spool

import (
	"bytes"
	"fmt"
	"time"

	"github.com/elastic/go-structform"
	"github.com/elastic/go-structform/cborl"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
	"github.com/elastic/go-structform/ubjson"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

type encoder struct {
	buf    bytes.Buffer
	folder *gotype.Iterator
	codec  codecID
}

type decoder struct {
	buf []byte

	json     *json.Parser
	cborl    *cborl.Parser
	ubjson   *ubjson.Parser
	unfolder *gotype.Unfolder
}

type codecID uint8

type entry struct {
	Timestamp int64
	Flags     uint8
	Meta      common.MapStr
	Fields    common.MapStr
}

const (
	// Note: Never change order. Codec IDs must be not change in the future. Only
	//       adding new IDs is allowed.
	codecUnknown codecID = iota
	codecJSON
	codecUBJSON
	codecCBORL

	flagGuaranteed uint8 = 1 << 0
)

func newEncoder(codec codecID) (*encoder, error) {
	switch codec {
	case codecJSON, codecCBORL, codecUBJSON:
		break
	default:
		return nil, fmt.Errorf("unknown codec type '%v'", codec)
	}

	e := &encoder{codec: codec}
	e.reset()
	return e, nil
}

func (e *encoder) reset() {
	e.folder = nil

	var visitor structform.Visitor
	switch e.codec {
	case codecJSON:
		visitor = json.NewVisitor(&e.buf)
	case codecCBORL:
		visitor = cborl.NewVisitor(&e.buf)
	case codecUBJSON:
		visitor = ubjson.NewVisitor(&e.buf)
	default:
		panic("no codec configured")
	}

	folder, err := gotype.NewIterator(visitor)
	if err != nil {
		panic(err)
	}

	e.folder = folder
}

func (e *encoder) encode(event *publisher.Event) ([]byte, error) {
	e.buf.Reset()
	e.buf.WriteByte(byte(e.codec))

	var flags uint8
	if (event.Flags & publisher.GuaranteedSend) == publisher.GuaranteedSend {
		flags = flagGuaranteed
	}

	err := e.folder.Fold(entry{
		Timestamp: event.Content.Timestamp.UTC().UnixNano(),
		Flags:     flags,
		Meta:      event.Content.Meta,
		Fields:    event.Content.Fields,
	})
	if err != nil {
		e.reset()
		return nil, err
	}

	return e.buf.Bytes(), nil
}

func newDecoder() *decoder {
	d := &decoder{}
	d.reset()
	return d
}

func (d *decoder) reset() {
	unfolder, err := gotype.NewUnfolder(nil)
	if err != nil {
		panic(err) // can not happen
	}

	d.unfolder = unfolder
	d.json = json.NewParser(unfolder)
	d.cborl = cborl.NewParser(unfolder)
	d.ubjson = ubjson.NewParser(unfolder)
}

// Buffer prepares the read buffer to hold the next event of n bytes.
func (d *decoder) Buffer(n int) []byte {
	if cap(d.buf) > n {
		d.buf = d.buf[:n]
	} else {
		d.buf = make([]byte, n)
	}
	return d.buf
}

func (d *decoder) Decode() (publisher.Event, error) {
	var (
		to       entry
		err      error
		codec    = codecID(d.buf[0])
		contents = d.buf[1:]
	)

	d.unfolder.SetTarget(&to)
	switch codec {
	case codecJSON:
		err = d.json.Parse(contents)
	case codecUBJSON:
		err = d.ubjson.Parse(contents)
	case codecCBORL:
		err = d.cborl.Parse(contents)
	default:
		return publisher.Event{}, fmt.Errorf("unknown codec type '%v'", codec)
	}

	if err != nil {
		d.reset() // reset parser just in case
		return publisher.Event{}, err
	}

	var flags publisher.EventFlags
	if (to.Flags & flagGuaranteed) != 0 {
		flags |= publisher.GuaranteedSend
	}

	return publisher.Event{
		Flags: flags,
		Content: beat.Event{
			Timestamp: time.Unix(0, to.Timestamp),
			Fields:    to.Fields,
			Meta:      to.Meta,
		},
	}, nil
}
