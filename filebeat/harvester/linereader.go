package harvester

import (
	"golang.org/x/text/encoding"

	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/filebeat/input"
)

func createLineReader(
	in FileSource,
	codec encoding.Encoding,
	bufferSize int,
	maxBytes int,
	readerConfig logFileReaderConfig,
	jsonConfig *input.JSONConfig,
	mlrConfig *input.MultilineConfig,
	done chan struct{},
) (processor.LineProcessor, error) {
	var p processor.LineProcessor
	var err error

	fileReader, err := newLogFileReader(in, readerConfig, done)
	if err != nil {
		return nil, err
	}

	p, err = processor.NewLineSource(fileReader, codec, bufferSize)
	if err != nil {
		return nil, err
	}

	if jsonConfig != nil {
		p = processor.NewJSONProcessor(p, jsonConfig)
	}

	p = processor.NewStripNewline(p)
	if mlrConfig != nil {
		p, err = processor.NewMultiline(p, "\n", maxBytes, mlrConfig)
		if err != nil {
			return nil, err
		}
	}

	return processor.NewLimitProcessor(p, maxBytes), nil
}
