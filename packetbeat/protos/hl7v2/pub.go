package hl7v2

import (
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

// Field struct
type Field struct {
	Field        int     `json:"field"`
	Repeat       int     `json:"repeat"`
	Component    int     `json:"component"`
	SubComponent int     `json:"subcomponent"`
	Text         string  `json:"text"`
	Numeric      float64 `json:"numeric,omitempty"`
	Date         string  `json:"date,omitempty"`
}

// Segment struct
type Segment struct {
	Line    int     `json:"line"`
	Segment string  `json:"segment"`
	Fields  []Field `json:"fields,omitempty"`
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

		// Map for our message
		messageMap := common.MapStr{}

		// Split message into segments
		if hl7message == "request" {
			hl7segments = strings.Split(string(requ.content), pub.NewLineChars)
		} else if hl7message == "response" {
			hl7segments = strings.Split(string(resp.content), pub.NewLineChars)
		} else {
			continue
		}

		// Array for our segments
		var segmentarray []Segment

		// Var for hl7linenumber
		hl7linenumber := 0

		// Loop through hl7segments
		for hl7segment := range hl7segments {

			// Array for our fields
			var fieldarray []Field

			// Increment hl7linenumber
			hl7linenumber = hl7segment + 1

			// Prevent error when reading blank lines.
			if strings.TrimSpace(hl7segments[hl7segment]) == "" {
				continue
			}

			// Set segment value
			hl7segmentvalue := hl7segments[hl7segment]
			//debugf("hl7segmentvalue: %v", hl7segmentvalue)

			// Set segment header
			hl7segmentheader := hl7segmentvalue[0:3]

			// Add hl7segmentheader to segmentsmap
			//segmentsmap[hl7segmentheader] = hl7segmentheader

			// If this is the MSH segment get our seperators
			if hl7segmentheader == "MSH" {
				hl7fieldseperator = string(hl7segments[hl7segment][3])
				//debugf("hl7fieldseperator: %s", hl7fieldseperator)
				hl7repeatseperator = string(hl7segments[hl7segment][5])
				//debugf("hl7repeatseperator: %s", hl7repeatseperator)
				hl7componentseperator = string(hl7segments[hl7segment][4])
				//debugf("hl7componentseperator: %s", hl7componentseperator)
				hl7subcomponentseperator = string(hl7segments[hl7segment][7])
				//debugf("hl7subcomponentseperator: %s", hl7subcomponentseperator)
			}

			// Split hl7segmentvalue into hl7fields
			hl7fields := strings.Split(hl7segmentvalue, hl7fieldseperator)

			// Loop through hl7fields
			for hl7field := range hl7fields {

				// If field header dont process
				if hl7field == 0 {
					//debugf("Not processing %v-%v.", hl7segmentheader, hl7field)
					continue
				}

				hl7fieldvalue := hl7fields[hl7field]
				//debugf("hl7fieldvalue: %v", hl7fieldvalue)
				hl7fieldnumber := hl7field
				//debugf("hl7fieldnumber: %v", hl7fieldnumber)

				// If this is the MSH segment increment the fieldnumber
				if hl7segmentheader == "MSH" {
					hl7fieldnumber++
				}

				// If MSH-2 (encoding chars) don't process
				if hl7segmentheader == "MSH" && hl7fieldnumber == 2 {
					//debugf("Not processing %v-%v.", hl7segmentheader, hl7fieldnumber)
					continue
				}

				// Log out core message info
				if hl7segmentheader == "MSH" {
					switch {
					case hl7fieldnumber == 3:
						messageMap["msh_sending_application"] = hl7fieldvalue
					case hl7fieldnumber == 4:
						messageMap["msh_sending_facility"] = hl7fieldvalue
					case hl7fieldnumber == 5:
						messageMap["msh_receiving_application"] = hl7fieldvalue
					case hl7fieldnumber == 6:
						messageMap["msh_receiving_facility"] = hl7fieldvalue
					case hl7fieldnumber == 7:
						messageMap["msh_datetime_of_message"] = hl7fieldvalue
					case hl7fieldnumber == 9:
						messageMap["msh_message_type"] = hl7fieldvalue
					case hl7fieldnumber == 10:
						messageMap["msh_message_control_id"] = hl7fieldvalue
					case hl7fieldnumber == 12:
						messageMap["msh_version_id"] = hl7fieldvalue
					default:
					}
				}

				if hl7segmentheader == "MSA" {
					switch {
					case hl7fieldnumber == 1:
						messageMap["msa_acknowledgement_code"] = hl7fieldvalue
					case hl7fieldnumber == 2:
						messageMap["msa_message_control_id"] = hl7fieldvalue
					case hl7fieldnumber == 3:
						messageMap["msa_text_message"] = hl7fieldvalue
					default:
					}
				}

				// Split hl7fieldvalue into hl7repeats
				hl7repeats := strings.Split(hl7fieldvalue, hl7repeatseperator)

				// Loop through hl7repeats
				for hl7repeat := range hl7repeats {
					hl7repeatvalue := hl7repeats[hl7repeat]
					//debugf("hl7repeatvalue: %v", hl7repeatvalue)
					hl7repeatnumber := hl7repeat + 1
					//debugf("hl7repeatnumber: %v", hl7repeatnumber)

					// Split hl7repeatvalue into hl7components
					hl7components := strings.Split(hl7repeatvalue, hl7componentseperator)

					// Loop through hl7components
					for hl7component := range hl7components {
						hl7componentvalue := hl7components[hl7component]
						//debugf("hl7componentvalue: %v", hl7componentvalue)
						hl7componentnumber := hl7component + 1
						//debugf("hl7componentnumber: %v", hl7componentnumber)

						// Split hl7componentvalue into hl7subcomponents
						hl7subcomponents := strings.Split(hl7componentvalue, hl7subcomponentseperator)

						// Loop through hl7subcomponents
						for hl7subcomponent := range hl7subcomponents {
							hl7subcomponentvalue := hl7subcomponents[hl7subcomponent]
							//debugf("hl7subcomponentvalue: %v", hl7subcomponentvalue)
							hl7subcomponentnumber := hl7subcomponent + 1
							//debugf("hl7subcomponentnumber: %v", hl7subcomponentnumber)

							// Add value to fieldarray
							if hl7subcomponentvalue != "" {
								if IsNumeric(hl7subcomponentvalue) {
									hl7subcomponentnumericvalue, _ := strconv.ParseFloat(hl7subcomponentvalue, 64)
									fieldarray = append(fieldarray, Field{hl7fieldnumber, hl7repeatnumber, hl7componentnumber, hl7subcomponentnumber, hl7subcomponentvalue, hl7subcomponentnumericvalue, ""})
								} else {
									fieldarray = append(fieldarray, Field{hl7fieldnumber, hl7repeatnumber, hl7componentnumber, hl7subcomponentnumber, hl7subcomponentvalue, 0, ""})
								}
							}
						}
						// End hl7subcomponents loop

					}
					// End hl7components loop

				}
				// End hl7repeats loop

			}
			// End hl7fields loop

			// Add fieldarray to segmentarray
			segmentarray = append(segmentarray, Segment{hl7linenumber, hl7segmentheader, fieldarray})

		}
		// End hl7segments loop

		// Add segmentarray to messageMap.items map
		messageMap["segments"] = segmentarray

		// Add messageMap to fields.hl7message map
		fields["hl7v2"].(common.MapStr)[hl7message] = messageMap

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
