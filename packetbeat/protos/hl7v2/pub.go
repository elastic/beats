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

	hl7message := "request"

	var hl7segments []string
	var hl7fieldseperator string
	//var hl7componentseperator string
	//var hl7subcomponentseperator string
	//var hl7fieldrepeatseperator string
	//var hl7escapecharacter string

	for i := 0; i < 2; i++ {
		hl7data := map[string]interface{}{}
		// Split message into segments
		if hl7message == "request" {
			hl7segments = strings.Split(string(requ.content), pub.NewLineChars)
		}
		if hl7message == "response" {
			hl7segments = strings.Split(string(resp.content), pub.NewLineChars)
		}
		// Loop through hl7segments
		for hl7segment := range hl7segments {

			// Prevent error when reading blank lines.
			if strings.TrimRight(hl7segments[hl7segment], "\r\n") == "" {
				continue
			}

			hl7segmentheader := hl7segments[hl7segment][0:3]
			debugf("Processing segment: %s", hl7segmentheader)
			// If segment matches
			if strings.EqualFold(pub.SegmentSelectionMode, "Include") && pub.segmentsmap[hl7segmentheader] || strings.EqualFold(pub.SegmentSelectionMode, "Exclude") && !pub.segmentsmap[hl7segmentheader] {
				debugf("Segment %s matched.", hl7segmentheader)
				// If MSH get our encoding characters
				if strings.EqualFold(hl7segmentheader, "MSH") {
					hl7fieldseperator = string(hl7segments[hl7segment][3])
					//hl7componentseperator = string(hl7segments[hl7segment][4])
					//hl7subcomponentseperator = string(hl7segments[hl7segment][5])
					//hl7fieldrepeatseperator = string(hl7segments[hl7segment][6])
					//hl7escapecharacter = string(hl7segments[hl7segment][7])
				}
				if pub.FieldSelectionMode != "" {
					debugf("FieldSelectionMode: %s", pub.FieldSelectionMode)
					// Split hl7segment into hl7fields
					hl7fields := strings.Split(hl7segments[hl7segment], hl7fieldseperator)
					for hl7field := range hl7fields {
						hl7fieldnumber := strconv.Itoa(hl7field)
						// Increment field numbers if this is an MSH value
						if strings.EqualFold(hl7segmentheader, "MSH") {
							hl7fieldnumber = strconv.Itoa(hl7field + 1)
						}
						hl7fieldname := strings.Join([]string{hl7segmentheader, "-", hl7fieldnumber}, "")
						debugf("Processing field: %s", hl7fieldname)
						hl7fieldvalue := hl7fields[hl7field]
						// If this is MSH-1 change hl7fieldvalue to the sperator character
						if strings.EqualFold(hl7fieldname, "MSH-1") {
							hl7fieldvalue = hl7fieldseperator
						}
						// If field matches
						if strings.EqualFold(pub.FieldSelectionMode, "Include") && pub.fieldsmap[hl7fieldname] || strings.EqualFold(pub.FieldSelectionMode, "Exclude") && !pub.fieldsmap[hl7fieldname] {
							debugf("Field %s matched.", hl7fieldname)

                            // To be added once get fields.yml down to component level
							// If selected split field into components
							/*if pub.ComponentSelectionMode != "" {
								debugf("componentsmap: %s", pub.componentsmap)
								debugf("ComponentSelectionMode: %s", pub.ComponentSelectionMode)
								hl7fieldcomponents := strings.Split(hl7fields[hl7field], hl7componentseperator)
								for hl7fieldcomponent := range hl7fieldcomponents {
									hl7fieldcomponentnumber := strconv.Itoa(hl7fieldcomponent + 1)
									hl7fieldcomponentname := strings.Join([]string{hl7fieldname, "-", hl7fieldcomponentnumber}, "")
									hl7fieldcomponentvalue := hl7fieldcomponents[hl7fieldcomponent]
									debugf("Processing component: %s", hl7fieldcomponentname)
									// If component matches
									if strings.EqualFold(pub.ComponentSelectionMode, "Include") && pub.componentsmap[hl7fieldcomponentname] || strings.EqualFold(pub.ComponentSelectionMode, "Exclude") && !pub.componentsmap[hl7fieldcomponentname] {
										debugf("Component %s matched.", hl7fieldcomponentname)
										// Add component if not empty
										if hl7fieldcomponentvalue != "" {
											hl7data[hl7fieldcomponentname] = hl7fieldcomponentvalue
											debugf("Added component %s with value %s", hl7fieldcomponentname, hl7fieldcomponentvalue)
										}
									}
								}
							} else {*/
							    // Add to field if not empty
							    if hl7fieldvalue != "" {
								    hl7data[hl7fieldname] = hl7fieldvalue
								    debugf("Added field %s with value %s", hl7fieldname, hl7fieldvalue)
							    }
                            //}
						}
					}
				} else {
                    // Add segment if not empty
                    if hl7segments[hl7segment] != "" {
                        hl7data[hl7segmentheader] = hl7segments[hl7segment]
					    debugf("Added segment %s with value %s", hl7segmentheader, hl7segments[hl7segment])
                    }
                }
			}
		}
		fields["hl7v2"].(common.MapStr)[hl7message] = hl7data
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
