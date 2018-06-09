package hl7v2

import (
	//"encoding/json"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/packetbeat/protos"
)

// Transaction Publisher.
type transPub struct {
	sendRequest            bool
	sendResponse           bool
	NewLineChars           string
	SegmentSelectionMode   string
	FieldSelectionMode     string
	ComponentSelectionMode string
	segmentsmap            map[string]bool
	fieldsmap              map[string]bool
	componentsmap          map[string]bool
	results                protos.Reporter
}

// Component struct
type Component struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

// Field struct
type Field struct {
	Id        string      `json:"id"`
	Component []Component `json:"component"`
}

// Segment struct
type Segment struct {
	Id    string  `json:"id"`
	Field []Field `json:"field"`
}

// Message struct
type Message struct {
	Segment []Segment `json:"segment"`
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

	fields := common.MapStr{
		"type":         "hl7v2",
		"status":       status,
		"responsetime": responseTime,
		"bytes_in":     requ.Size,
		"bytes_out":    resp.Size,
		"src":          src,
		"dst":          dst,
		"hl7v2":        common.MapStr{},
	}

	// Start with the request
	hl7message := "request"

	// Var for our segments array
	var hl7segments []string

	// Loop through request and response
	for i := 0; i < 2; i++ {

		// Default field seperator
		hl7fieldseperator := "|"

		// Default component seperator
		hl7componentseperator := "^"

		// Split message into segments
		if hl7message == "request" {
			hl7segments = strings.Split(string(requ.content), pub.NewLineChars)
		} else if hl7message == "response" {
			hl7segments = strings.Split(string(resp.content), pub.NewLineChars)
		} else {
			continue
		}

		// Slice for our segment fields
		var segmentslice []Segment

		// Loop through hl7segments
		for hl7segment := range hl7segments {

			// Prevent error when reading blank lines.
			if strings.TrimRight(hl7segments[hl7segment], "\r\n") == "" {
				continue
			}

			// Set segment header
			hl7segmentheader := hl7segments[hl7segment][0:3]

			// If this is the MSH segment get our encoding characters
			if strings.EqualFold(hl7segmentheader, "MSH") {
				hl7fieldseperator = string(hl7segments[hl7segment][3])
				hl7componentseperator = string(hl7segments[hl7segment][4])
			}

			// Split segment into fields
			hl7fields := strings.Split(hl7segments[hl7segment], hl7fieldseperator)

			// Slice for our field components
			var fieldslice []Field

			// Loop through fields
			for hl7field := range hl7fields {

				// Set field number
				hl7fieldnumber := strconv.Itoa(hl7field)

				// Increment field numbers if this is an MSH value
				if strings.EqualFold(hl7segmentheader, "MSH") {
					hl7fieldnumber = strconv.Itoa(hl7field + 1)
				}

				// Process if not hl7fieldnumber 0
				if hl7fieldnumber != "0" {

					// Split field into components
					hl7components := strings.Split(hl7fields[hl7field], hl7componentseperator)

					// Slice for our component values
					var componentslice []Component

					// Loop through components
					for hl7component := range hl7components {

						// Set component number
						hl7componentnumber := strconv.Itoa(hl7component + 1)

						// Set component value
						hl7componentvalue := strings.TrimSpace(hl7components[hl7component])

						// If this is MSH field 1, component 1 then set value to the field seperator
						if hl7segmentheader == "MSH" && hl7fieldnumber == "1" && hl7componentnumber == "1" {
							hl7componentvalue = hl7fieldseperator
						}

						// Add hl7componentvalue to componentslice if not empty
						if hl7componentvalue != "" {
							componentslice = append(componentslice, Component{"" + hl7componentnumber + "", "" + hl7componentvalue + ""})
						}
					}

					// Add componentslice to fieldslice
					if len(componentslice) != 0 {
						fieldslice = append(fieldslice, Field{"" + hl7fieldnumber + "", componentslice})
					}

				}

			}

			// Add fieldslice to segmentslice
			if len(fieldslice) != 0 {
				segmentslice = append(segmentslice, Segment{"" + hl7segmentheader + "", fieldslice})
			}

		}

		// Add Message to fields.hl7message map
		fields["hl7v2"].(common.MapStr)[hl7message] = Message{segmentslice}

		// Switch to response message
		hl7message = "response"
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
		Fields:    fields,
	}
}
