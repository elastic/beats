package outputs

import (
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
)

type Writer interface {
	Write(Event common.MapStr) ([]byte, error)
}
type writerType string

func (t *writerType) Unpack(in interface{}) error {
	s, ok := in.(string)
	if !ok {
		return fmt.Errorf("writer type must be an identifier")
	}

	writerType, found := writerTypes[s]
	if !found {
		return fmt.Errorf("invalid writer type '%v'", s)
	}

	*t = writerType
	return nil
}

const (
	FormatStringWriterType writerType = "FormatStringWriter"
	JsonWriterType         writerType = "JsonWriter"
)

var (
	writerTypes = map[string]writerType{
		"":                   JsonWriterType,
		"JsonWriter":         JsonWriterType,
		"FormatStringWriter": FormatStringWriterType,
	}
)

// TLSConfig defines config file options for TLS clients.
type WriterConfig struct {
	Type   writerType                `config:"type"`
	Pretty bool                      `config:"pretty"`
	Format *fmtstr.EventFormatString `config:"format"`
}

type JsonWriter struct {
	Writer,
	Pretty bool
}

func CreateWriter(config WriterConfig) Writer {
	switch config.Type {
	case "FormatStringWriter":
		return NewFormatStringWriter(config.Format)
	case "JsonWriter":
	default:
		return NewJsonWriter(config.Pretty)
	}
	return NewJsonWriter(config.Pretty)
}

func NewJsonWriter(pretty bool) *JsonWriter {
	jsonWriter := new(JsonWriter)
	jsonWriter.Pretty = pretty
	return jsonWriter
}

func (j *JsonWriter) Write(event common.MapStr) ([]byte, error) {
	var err error
	var serializedEvent []byte

	if j.Pretty {
		serializedEvent, err = json.MarshalIndent(event, "", "  ")
	} else {
		serializedEvent, err = json.Marshal(event)
	}
	if err != nil {
		logp.Err("Fail to convert the event to JSON (%v): %#v", err, event)
	}

	return serializedEvent, err
}

type FormatStringWriter struct {
	Writer,
	Format *fmtstr.EventFormatString
}

func NewFormatStringWriter(format *fmtstr.EventFormatString) *FormatStringWriter {
	formattedWriter := new(FormatStringWriter)
	if format == nil {
		format = fmtstr.MustCompileEvent("%{[message]}")
	}
	formattedWriter.Format = format
	return formattedWriter
}

func (w *FormatStringWriter) Write(event common.MapStr) ([]byte, error) {

	serializedEvent, err := w.Format.RunBytes(event)
	if err != nil {
		logp.Err("Fail to format event (%v): %#v", err, event)
		return serializedEvent, err
	}

	return serializedEvent, err
}
