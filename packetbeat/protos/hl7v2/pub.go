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

// SubComponent struct
type SubComponent struct {
	ID      int     `json:"id"`
	Value   string  `json:"value,omitempty"`
	Numeric float64 `json:"numeric,omitempty"`
}

// Component struct
type Component struct {
	ID           int            `json:"id"`
	SubComponent []SubComponent `json:"subcomponent,omitempty"`
}

// Repeat struct
type Repeat struct {
	ID        int         `json:"id"`
	Component []Component `json:"component,omitempty"`
}

// Field struct
type Field struct {
	ID     int      `json:"id"`
	Repeat []Repeat `json:"repeat,omitempty"`
}

// Segment struct
type Segment struct {
	ID    string  `json:"id"`
	Field []Field `json:"field,omitempty"`
}

// Message struct
type Message struct {
	Segment []Segment `json:"segment"`
}

// IsNumeric checks if a string is a numeric value
func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
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

	var hl7segments []string

	// Default field seperator
	hl7fieldseperator := "|"

	// Default repeat seperator
	hl7repeatseperator := "~"

	// Default component seperator
	hl7componentseperator := "^"

	// Default subcomponent seperator
	hl7subcomponentseperator := "&"

	// Loop through request and response
	for i := 0; i < 2; i++ {

		// Split message into segments
		if hl7message == "request" {
			hl7segments = strings.Split(string(requ.content), pub.NewLineChars)
		} else if hl7message == "response" {
			hl7segments = strings.Split(string(resp.content), pub.NewLineChars)
		} else {
			continue
		}

		// Array for our segment values
		var segmentarray []Segment
		// Loop through hl7segments
		for hl7segment := range hl7segments {

			// Prevent error when reading blank lines.
			if strings.TrimSpace(hl7segments[hl7segment]) == "" {
				continue
			}

			hl7segmentvalue := hl7segments[hl7segment]
			debugf("hl7segmentvalue: %v", hl7segmentvalue)
			hl7segmentnumber := hl7segment + 1
			debugf("hl7segmentnumber: %v", hl7segmentnumber)

			// Set segment header
			hl7segmentheader := hl7segmentvalue[0:3]

			// If this is the MSH segment get our seperators
			if hl7segmentheader == "MSH" {
				hl7fieldseperator = string(hl7segments[hl7segment][3])
				debugf("hl7fieldseperator: %s", hl7fieldseperator)
				hl7repeatseperator = string(hl7segments[hl7segment][5])
				debugf("hl7repeatseperator: %s", hl7repeatseperator)
				hl7componentseperator = string(hl7segments[hl7segment][4])
				debugf("hl7componentseperator: %s", hl7componentseperator)
				hl7subcomponentseperator = string(hl7segments[hl7segment][7])
				debugf("hl7subcomponentseperator: %s", hl7subcomponentseperator)
			}

			// Split hl7segmentvalue into hl7fields
			hl7fields := strings.Split(hl7segmentvalue, hl7fieldseperator)

			// Array for our field values
			var fieldarray []Field
			// Loop through hl7fields
			for hl7field := range hl7fields {

				// If field header dont process
				if hl7field == 0 {
					debugf("Not processing %v-%v.", hl7segmentheader, hl7field)
					continue
				}

				hl7fieldvalue := hl7fields[hl7field]
				debugf("hl7fieldvalue: %v", hl7fieldvalue)
				hl7fieldnumber := hl7field + 1
				debugf("hl7fieldnumber: %v", hl7fieldnumber)

				// If MSH-1 or MSH-2 don't process
				if hl7segmentheader == "MSH" && (hl7fieldnumber == 1 || hl7fieldnumber == 2) {
					debugf("Not processing %v-%v.", hl7segmentheader, hl7fieldnumber)
					continue
				}

				// Split hl7fieldvalue into hl7repeats
				hl7repeats := strings.Split(hl7fieldvalue, hl7repeatseperator)

				// Array for our repeat values
				var repeatarray []Repeat
				// Loop through hl7repeats
				for hl7repeat := range hl7repeats {
					hl7repeatvalue := hl7repeats[hl7repeat]
					debugf("hl7repeatvalue: %v", hl7repeatvalue)
					hl7repeatnumber := hl7repeat + 1
					debugf("hl7repeatnumber: %v", hl7repeatnumber)

					// Split hl7repeatvalue into hl7components
					hl7components := strings.Split(hl7repeatvalue, hl7componentseperator)

					// Array for our component values
					var componentarray []Component
					// Loop through hl7components
					for hl7component := range hl7components {
						hl7componentvalue := hl7components[hl7component]
						debugf("hl7componentvalue: %v", hl7componentvalue)
						hl7componentnumber := hl7component + 1
						debugf("hl7componentnumber: %v", hl7componentnumber)

						// Split hl7componentvalue into hl7subcomponents
						hl7subcomponents := strings.Split(hl7componentvalue, hl7subcomponentseperator)

						// Array for our subcomponent values
						var subcomponentarray []SubComponent
						// Loop through hl7subcomponents
						for hl7subcomponent := range hl7subcomponents {
							hl7subcomponentvalue := hl7subcomponents[hl7subcomponent]
							debugf("hl7subcomponentvalue: %v", hl7subcomponentvalue)
							hl7subcomponentnumber := hl7subcomponent + 1
							debugf("hl7subcomponentnumber: %v", hl7subcomponentnumber)

							// Add value to subcomponentarray
							if hl7subcomponentvalue != "" {
								if IsNumeric(hl7subcomponentvalue) {
									hl7subcomponentnumericvalue, _ := strconv.ParseFloat(hl7subcomponentvalue, 64)
									subcomponentarray = append(subcomponentarray, SubComponent{hl7subcomponentnumber, hl7subcomponentvalue, hl7subcomponentnumericvalue})
								} else {
									subcomponentarray = append(subcomponentarray, SubComponent{hl7subcomponentnumber, hl7subcomponentvalue, 0})
								}
							}
						}
						// End hl7subcomponents loop

						// Add subcomponentarray to componentarray
						if len(subcomponentarray) != 0 {
							componentarray = append(componentarray, Component{hl7componentnumber, subcomponentarray})
						}

					}
					// End hl7components loop

					// Add componentarray to repeatarray
					if len(componentarray) != 0 {
						repeatarray = append(repeatarray, Repeat{hl7repeatnumber, componentarray})
					}

				}
				// End hl7repeats loop

				// Add repeatarray to fieldarray
				if len(repeatarray) != 0 {
					fieldarray = append(fieldarray, Field{hl7fieldnumber, repeatarray})
				}

			}
			// End hl7fields loop

			// Add fieldarray to segmentarray
			if len(fieldarray) != 0 {
				segmentarray = append(segmentarray, Segment{hl7segmentheader, fieldarray})
			}

		}
		// End hl7segments loop

		// Add Message to fields.hl7message map
		fields["hl7v2"].(common.MapStr)[hl7message] = Message{segmentarray}

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
