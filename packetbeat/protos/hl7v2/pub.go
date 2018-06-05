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

	// Set some vars
	var hl7segments []string

	// Loop through request and response
	for i := 0; i < 2; i++ {

		// Map to store this message
		hl7messagemap := common.MapStr{}

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

		// Loop through hl7segments
		for hl7segment := range hl7segments {

			// Prevent error when reading blank lines.
			if strings.TrimRight(hl7segments[hl7segment], "\r\n") == "" {
				continue
			}

			// Map to store this segment
			hl7segmentmap := common.MapStr{}

			// Set line number
			hl7linenumber := hl7segment + 1

			// Set segment header
			hl7segmentheader := hl7segments[hl7segment][0:3]
			debugf("Processing segment: %s", hl7segmentheader)

			// If segment selected
			if strings.EqualFold(pub.SegmentSelectionMode, "Include") && pub.segmentsmap[hl7segmentheader] || strings.EqualFold(pub.SegmentSelectionMode, "Exclude") && !pub.segmentsmap[hl7segmentheader] {
				debugf("Segment %s matched.", hl7segmentheader)

				// If this is the MSH segment get our encoding characters
				if strings.EqualFold(hl7segmentheader, "MSH") {
					hl7fieldseperator = string(hl7segments[hl7segment][3])
					hl7componentseperator = string(hl7segments[hl7segment][4])
				}

				// If selected split segment into fields
				if pub.FieldSelectionMode != "" {
					debugf("FieldSelectionMode: %s", pub.FieldSelectionMode)
					hl7fields := strings.Split(hl7segments[hl7segment], hl7fieldseperator)

					// Create map to store field
					hl7fieldmap := common.MapStr{}

					// Loop through fields
					for hl7field := range hl7fields {

						// Set field number
						hl7fieldnumber := strconv.Itoa(hl7field)

						// Increment field numbers if this is an MSH value
						if strings.EqualFold(hl7segmentheader, "MSH") {
							hl7fieldnumber = strconv.Itoa(hl7field + 1)
						}

						// Set field name
						hl7fieldname := strings.Join([]string{hl7segmentheader, ".", hl7fieldnumber}, "")
						debugf("Processing field: %s", hl7fieldname)

						// Set field value
						hl7fieldvalue := hl7fields[hl7field]

						// If this is MSH.1 change field value to the sperator character
						if strings.EqualFold(hl7fieldname, "MSH.1") {
							hl7fieldvalue = hl7fieldseperator
						}

						// If field selected
						if strings.EqualFold(pub.FieldSelectionMode, "Include") && pub.fieldsmap[hl7fieldname] || strings.EqualFold(pub.FieldSelectionMode, "Exclude") && !pub.fieldsmap[hl7fieldname] {
							debugf("Field %s matched.", hl7fieldname)

							// If selected split field into components
							if pub.ComponentSelectionMode != "" {
								hl7fieldcomponents := strings.Split(hl7fields[hl7field], hl7componentseperator)

								// Create map to store this component
								hl7fieldcomponentmap := common.MapStr{}

								// Loop through components
								for hl7fieldcomponent := range hl7fieldcomponents {

									// Set component number
									hl7fieldcomponentnumber := strconv.Itoa(hl7fieldcomponent + 1)

									// Set component name
									hl7fieldcomponentname := strings.Join([]string{hl7fieldname, ".", hl7fieldcomponentnumber}, "")

									// Set component value
									hl7fieldcomponentvalue := hl7fieldcomponents[hl7fieldcomponent]
									debugf("Processing component: %s", hl7fieldcomponentname)

									// If component selected
									if strings.EqualFold(pub.ComponentSelectionMode, "Include") && pub.componentsmap[hl7fieldcomponentname] || strings.EqualFold(pub.ComponentSelectionMode, "Exclude") && !pub.componentsmap[hl7fieldcomponentname] {
										debugf("Component %s matched.", hl7fieldcomponentname)

										// Add component to hl7fieldcomponentmap if not empty
										if hl7fieldcomponentvalue != "" {
											hl7fieldcomponentmap[hl7fieldcomponentnumber] = hl7fieldcomponentvalue
											debugf("Added component %s with value %s", hl7fieldcomponentname, hl7fieldcomponentvalue)
										}
									}
								}

								// Add hl7fieldcomponentmap to hl7fieldmap if not empty
								if len(hl7fieldcomponentmap) != 0 {
									hl7fieldmap[hl7fieldnumber] = hl7fieldcomponentmap
								}

							} else {
								// Add field to hl7fieldmap if not empty
								if len(hl7fieldvalue) != 0 {
									hl7fieldmap[hl7fieldnumber] = hl7fieldvalue
									debugf("Added field %s with value %s", hl7fieldname, hl7fieldvalue)
								}
							}
						}
					}

					// Add hl7fieldmap to hl7segments if not empty
					if len(hl7fieldmap) != 0 {
						hl7segmentmap[hl7segmentheader] = hl7fieldmap
					}

				} else {
					// Add segment to hl7segmentmap if not empty
					if hl7segments[hl7segment] != "" {
						hl7segmentmap[hl7segmentheader] = hl7segments[hl7segment]
						debugf("Added segment %s with value %s", hl7segmentheader, hl7segments[hl7segment])
					}
				}
			}

			// Add hl7segmentmap to hl7messagemap if not empty
			if len(hl7segmentmap) != 0 {
				hl7messagemap[strconv.Itoa(hl7linenumber)] = hl7segmentmap
			}
		}

		// Add hl7messagemap to fields if not empty
		if len(hl7messagemap) != 0 {
			fields["hl7v2"].(common.MapStr)[hl7message] = hl7messagemap
		}
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
