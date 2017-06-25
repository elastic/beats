package logstash

/*
// Event describes the event strucutre for events
// (in-)directly send to logstash
type Event struct {
	Timestamp time.Time     `struct:"@timestamp"`
	Meta      Meta          `struct:"@metadata"`
	Fields    common.MapStr `struct:",inline"`
}

// Meta defines common event metadata to be stored in '@metadata'
type Meta struct {
	Beat   string                 `struct:"beat"`
	Type   string                 `struct:"type"`
	Fields map[string]interface{} `struct:",inline"`
}

func MakeEvent(index string, event *beat.Event) Event {
	return Event{
		Timestamp: event.Timestamp,
		Meta: Meta{
			Beat:   index,
			Type:   "doc",
			Fields: event.Meta,
		},
		Fields: event.Fields,
	}
}
*/
