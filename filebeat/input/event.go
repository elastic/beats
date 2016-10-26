package input

import (
	"fmt"
	"regexp"
	"strconv"
	s "strings"
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Event is sent to the output and must contain all relevant information
type Event struct {
	common.EventMetadata
	ReadTime        time.Time
	InputType       string
	DocumentType    string
	Bytes           int
	Text            *string
	JSONFields      common.MapStr
	JSONConfig      *reader.JSONConfig
	State           file.State
	AnnotationRegex string
	DateFormat      string
}

func NewEvent(state file.State) *Event {
	return &Event{
		State: state,
	}
}

func (f *Event) ToMapStr() common.MapStr {
	event := common.MapStr{
		common.EventMetadataKey: f.EventMetadata,
		"@timestamp":            common.Time(f.ReadTime),
		"source":                f.State.Source,
		"offset":                f.State.Offset, // Offset here is the offset before the starting char.
		"type":                  f.DocumentType,
		"input_type":            f.InputType,
	}

	if f.JSONConfig != nil && len(f.JSONFields) > 0 {
		mergeJSONFields(f, event)
	} else if f.Text != nil {
		event["message"] = *f.Text
	}

	logp.Debug("event", "ToMapStr-->DocumentType=%s", f.DocumentType)
	logp.Debug("event", "ToMapStr-->regex=%s", f.AnnotationRegex)

	if f.AnnotationRegex != "" {
		logp.Debug("event", "ToMapStr-->apply regular expression on text")

		//move compiling part to once per propsector v/s once for every event
		var myExp = regexp.MustCompile(f.AnnotationRegex)
		match := myExp.FindStringSubmatch(*f.Text)

		if match != nil {

			//for every named match from regular expression
			for i, name := range myExp.SubexpNames() {
				if i != 0 {
					if match[i] != "" {
						if s.HasSuffix(name, "_int") {
							i, err := strconv.Atoi(match[i])
							if err == nil {
								event[s.TrimSuffix(name, "_int")] = i
							} else {
								logp.Err("event", "Err converting %d to int", match[i])
							}
						} else if s.HasSuffix(name, "_date") {
							logp.Debug("event", "dateFormat : %s", f.DateFormat)

							//input format should come from yml file
							//const inputTime = "02/Jan/2006:15:04:05 -0700"
							//check if dateFormat field exists, if yes then consume it

							if f.DateFormat != "" {
								//convert : to . so millisecond can be taken
								str1 := s.Replace(f.DateFormat, ":", ".", 3)
								logp.Debug("event", "input timeformat  %s", str1)
								inputTime := s.Replace(match[i], ":", ".", 3)
								logp.Debug("event", "input time  %s", inputTime)
								t, err := time.Parse(str1, inputTime)
								if err != nil {
									logp.Err("event", "Err converting date %s", inputTime)
								} else {
									event[s.TrimSuffix(name, "_date")] = t.Format("2006-01-02T15:04:05.000-0700")
									logp.Debug("event", "after conversion %s", event[s.TrimSuffix(name, "_date")])
								}
							}
						} else if s.HasSuffix(name, "_long") {

							num, err := strconv.ParseInt(match[i], 10, 64)
							if err == nil {
								event[s.TrimSuffix(name, "_long")] = num
							} else {
								logp.Err("file.go", "Err converting %s to long", match[i])
							}
						} else if s.HasSuffix(name, "_float") {
							num, err := strconv.ParseFloat(match[i], 64)
							if err == nil {
								event[s.TrimSuffix(name, "_float")] = num
							} else {
								logp.Err("event", "Err converting %s to float", match[i])
							}
						} else {
							event[name] = match[i]
						}
						logp.Debug("event", "%s=%s", name, match[i])
					} //end of match[i] != ""
				} // end of if i !=0
			} // enf of forloop

		} else {
			logp.Err("no match for regular expression:%s=", f.AnnotationRegex)
			logp.Err("text being matched : %s=", *f.Text)
		}
	}

	return event
}

// HasData returns true if the event itself contains data
// Events without data are only state updates
func (e *Event) HasData() bool {
	return e.Bytes > 0
}

// mergeJSONFields writes the JSON fields in the event map,
// respecting the KeysUnderRoot and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func mergeJSONFields(f *Event, event common.MapStr) {

	// The message key might have been modified by multiline
	if len(f.JSONConfig.MessageKey) > 0 && f.Text != nil {
		f.JSONFields[f.JSONConfig.MessageKey] = *f.Text
	}

	if f.JSONConfig.KeysUnderRoot {
		for k, v := range f.JSONFields {
			if f.JSONConfig.OverwriteKeys {
				if k == "@timestamp" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite @timestamp because value is not string")
						event[reader.JsonErrorKey] = "@timestamp not overwritten (not string)"
						continue
					}

					// @timestamp must be of format RFC3339
					ts, err := time.Parse(time.RFC3339, vstr)
					if err != nil {
						logp.Err("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
						event[reader.JsonErrorKey] = fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr)
						continue
					}
					event[k] = common.Time(ts)
				} else if k == "type" {
					vstr, ok := v.(string)
					if !ok {
						logp.Err("JSON: Won't overwrite type because value is not string")
						event[reader.JsonErrorKey] = "type not overwritten (not string)"
						continue
					}
					if len(vstr) == 0 || vstr[0] == '_' {
						logp.Err("JSON: Won't overwrite type because value is empty or starts with an underscore")
						event[reader.JsonErrorKey] = fmt.Sprintf("type not overwritten (invalid value [%s])", vstr)
						continue
					}
					event[k] = vstr
				} else {
					event[k] = v
				}
			} else if _, exists := event[k]; !exists {
				event[k] = v
			}
		}
	} else {
		event["json"] = f.JSONFields
	}
}
