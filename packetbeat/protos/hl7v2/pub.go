package hl7v2

import (
	"strings"
	"strconv"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/protos"
)

// Transaction Publisher.
type transPub struct {
	sendRequest				bool
	sendResponse			bool
	NewLineChars			string
	SegmentSelectionMode	string
	FieldSelectionMode		string
	segmentsmap		map[string]bool
	fieldsmap		map[string]bool
	fieldmappingmap			map[string]string
	results protos.Reporter
}

func (pub *transPub) onTransaction(requ, resp *message) error {
	if pub.results == nil {
		return nil
	}

	pub.results(pub.createEvent(requ, resp))
	return nil
}

func (pub *transPub) createEvent(requ, resp *message) beat.Event {
	
	status := common.OK_STATUS
	if resp.failed {
		status = common.ERROR_STATUS
	}

	// resp_time in milliseconds
	responseTime := int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6)

	src := &common.Endpoint{
		IP:   requ.Tuple.SrcIP.String(),
		Port: requ.Tuple.SrcPort,
		Proc: string(requ.CmdlineTuple.Src),
	}
	dst := &common.Endpoint{
		IP:   requ.Tuple.DstIP.String(),
		Port: requ.Tuple.DstPort,
		Proc: string(requ.CmdlineTuple.Dst),
	}

	fields := common.MapStr {
		"type":         "hl7v2",
		"status":       status,
		"responsetime": responseTime,
		"bytes_in":     requ.Size,
		"bytes_out":    resp.Size,
		"src":          src,
		"dst":          dst,
	}
	
	thisMessage := "request"
	var segments []string
	for i := 0; i < 2; i++ {
		if thisMessage == "request" {
			segments = strings.Split(string(requ.content), pub.NewLineChars)
		}
		if thisMessage == "response" {
			segments = strings.Split(string(resp.content), pub.NewLineChars)
		}
		// Loop through segments
		for segment := range segments {
			segmentheader := segments[segment][0:3]
			if (pub.SegmentSelectionMode == "Include" && pub.segmentsmap[segmentheader]) || (pub.SegmentSelectionMode == "Exclude" && !(pub.segmentsmap[segmentheader])) {
				// Field seperator
				hl7fieldseperator := string(segments[segment][3])
				// Split segment into fields
				hl7fields := strings.Split(segments[segment], hl7fieldseperator)
				for field := range hl7fields {
					fieldnumber := strconv.Itoa(field)
					// Increment field numbers if this is an MSH value
					if segmentheader == "MSH" {
						fieldnumber = strconv.Itoa(field + 1)
					}
					fieldname := strings.Join([]string{segmentheader, "-", fieldnumber}, "")
					if (pub.FieldSelectionMode == "Include" && pub.fieldsmap[fieldname]) || (pub.FieldSelectionMode == "Exclude" && !(pub.fieldsmap[fieldname])) {
						fieldvalue := hl7fields[field]
						// If this is MSH-1 change fieldvalue to the sperator character
						if fieldname == "MSH-1" {
							fieldvalue = hl7fieldseperator
						}
						// Re-map fieldname if configured
						if pub.fieldmappingmap[fieldname] != "" {
							fieldname = pub.fieldmappingmap[fieldname]
						}
						// Add to fields map if not empty
						if fieldvalue != "" {
							fields[fieldname] = fieldvalue
						}
					}
				}
			}
		}
		thisMessage = "response"
	}
	
	// add processing notes/errors to event
	if len(requ.Notes)+len(resp.Notes) > 0 {
		fields["notes"] = append(requ.Notes, resp.Notes...)
	}

	if pub.sendRequest {
		fields["request"] = requ.content
	}
	if pub.sendResponse {
		fields["response"] = resp.content
	}

	return beat.Event{
		Timestamp: requ.Ts,
		Fields: fields,
	}
}