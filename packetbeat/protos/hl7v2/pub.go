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
	namemappingmap         map[string]string
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
	}

	hl7message := "request"

	var hl7segments []string
	var hl7fieldseperator string
	var hl7componentseperator string
	//var hl7subcomponentseperator string
	//var hl7fieldrepeatseperator string
	//var hl7escapecharacter string

	for i := 0; i < 2; i++ {
		// Split message into segments
		if hl7message == "request" {
			hl7segments = strings.Split(string(requ.content), pub.NewLineChars)
		}
		if hl7message == "response" {
			hl7segments = strings.Split(string(resp.content), pub.NewLineChars)
		}
		// Loop through hl7segments
		for hl7segment := range hl7segments {
			hl7segmentheader := hl7segments[hl7segment][0:3]
			debugf("Processing segment: %s", hl7segmentheader)
			// If segment matches
			if pub.SegmentSelectionMode == "Include" && pub.segmentsmap[hl7segmentheader] || pub.SegmentSelectionMode == "Exclude" && !pub.segmentsmap[hl7segmentheader] {
				debugf("Segment %s matched.", hl7segmentheader)
				// If MSH get our encoding characters
				if hl7segmentheader == "MSH" {
					hl7fieldseperator = string(hl7segments[hl7segment][3])
					hl7componentseperator = string(hl7segments[hl7segment][4])
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
						if hl7segmentheader == "MSH" {
							hl7fieldnumber = strconv.Itoa(hl7field + 1)
						}
						hl7fieldname := strings.Join([]string{hl7segmentheader, "-", hl7fieldnumber}, "")
						debugf("Processing field: %s", hl7fieldname)
						hl7fieldvalue := hl7fields[hl7field]
						// If this is MSH-1 change hl7fieldvalue to the sperator character
						if hl7fieldname == "MSH-1" {
							hl7fieldvalue = hl7fieldseperator
						}
						// If field matches
						if pub.FieldSelectionMode == "Include" && pub.fieldsmap[hl7fieldname] || pub.FieldSelectionMode == "Exclude" && !pub.fieldsmap[hl7fieldname] {
							debugf("Field %s matched.", hl7fieldname)
							// If selected split field into components
							if pub.ComponentSelectionMode != "" {
								debugf("componentsmap: %s", pub.componentsmap)
								debugf("ComponentSelectionMode: %s", pub.ComponentSelectionMode)
								hl7fieldcomponents := strings.Split(hl7fields[hl7field], hl7componentseperator)
								for hl7fieldcomponent := range hl7fieldcomponents {
									hl7fieldcomponentnumber := strconv.Itoa(hl7fieldcomponent + 1)
									hl7fieldcomponentname := strings.Join([]string{hl7fieldname, "-", hl7fieldcomponentnumber}, "")
									hl7fieldcomponentvalue := hl7fieldcomponents[hl7fieldcomponent]
									debugf("Processing component: %s", hl7fieldcomponentname)
									// If component matches
									if pub.ComponentSelectionMode == "Include" && pub.componentsmap[hl7fieldcomponentname] || pub.ComponentSelectionMode == "Exclude" && !pub.componentsmap[hl7fieldcomponentname] {
										debugf("Component %s matched.", hl7fieldcomponentname)
										// Re-map componentname if configured
										if pub.namemappingmap[hl7fieldcomponentname] != "" {
											debugf("Component %s renamed to %s.", hl7fieldcomponentname, pub.namemappingmap[hl7fieldcomponentname])
											hl7fieldcomponentname = pub.namemappingmap[hl7fieldcomponentname]
										}
										// Add component if not empty
										if hl7fieldcomponentvalue != "" {
											fields[hl7fieldcomponentname] = hl7fieldcomponentvalue
										}
										debugf("Added component %s with value %s", hl7fieldcomponentname, hl7fieldcomponentvalue)
									}
								}
							} else {
								// Re-map fieldname if configured
								if pub.namemappingmap[hl7fieldname] != "" {
									debugf("Field %s renamed to %s.", hl7fieldname, pub.namemappingmap[hl7fieldname])
									hl7fieldname = pub.namemappingmap[hl7fieldname]
								}
								// Add to field if not empty
								if hl7fieldvalue != "" {
									fields[hl7fieldname] = hl7fieldvalue
								}
								debugf("Added field %s with value %s", hl7fieldname, hl7fieldvalue)
							}
						}
					}
				} else {
					fields[hl7segmentheader] = hl7segment
					debugf("Added segment %s with value %s", hl7segmentheader, hl7segment)
				}
			}
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
