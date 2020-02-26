// Pacakge rfc5424 is a library for parsing and serializing RFC-5424 structured
// syslog messages.
//
// Example usage:
//
//     m := rfc5424.Message{
//         Priority:  rfc5424.Daemon | rfc5424.Info,
//         Timestamp: time.Now(),
//         Hostname:  "myhostname",
//         AppName:   "someapp",
//         Message:   []byte("Hello, World!"),
//     }
//     m.AddDatum("foo@1234", "Revision", "1.2.3.4")
//     m.WriteTo(os.Stdout)
//
// Produces output like:
//
//     107 <7>1 2016-02-28T09:57:10.804642398-05:00 myhostname someapp - - [foo@1234 Revision="1.2.3.4"] Hello, World!
//
// You can also use the library to parse syslog messages:
//
//     m := rfc5424.Message{}
//     _, err := m.ReadFrom(os.Stdin)
//     fmt.Printf("%s\n", m.Message)
package rfc5424

import "time"

// Message represents a log message as defined by RFC-5424
// (https://tools.ietf.org/html/rfc5424)
type Message struct {
	Priority       Priority
	Timestamp      time.Time
	Hostname       string
	AppName        string
	ProcessID      string
	MessageID      string
	StructuredData []StructuredData
	Message        []byte
}

// SDParam represents parameters for structured data
type SDParam struct {
	Name  string
	Value string
}

// StructuredData represents structured data within a log message
type StructuredData struct {
	ID         string
	Parameters []SDParam
}

// AddDatum adds structured data to a log message
func (m *Message) AddDatum(ID string, Name string, Value string) {
	if m.StructuredData == nil {
		m.StructuredData = []StructuredData{}
	}
	for i, sd := range m.StructuredData {
		if sd.ID == ID {
			sd.Parameters = append(sd.Parameters, SDParam{Name: Name, Value: Value})
			m.StructuredData[i] = sd
			return
		}
	}

	m.StructuredData = append(m.StructuredData, StructuredData{
		ID: ID,
		Parameters: []SDParam{
			{
				Name:  Name,
				Value: Value,
			},
		},
	})
}
