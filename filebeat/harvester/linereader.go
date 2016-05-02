package harvester

import (
	"golang.org/x/text/encoding"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/processor"
)

func createLineReader(
	in FileSource,
	codec encoding.Encoding,
	bufferSize int,
	maxBytes int,
	readerConfig logFileReaderConfig,
	jsonConfig *config.JSONConfig,
	mlrConfig *config.MultilineConfig,
) (processor.LineProcessor, error) {
	var p processor.LineProcessor
	var err error

	fileReader, err := newLogFileReader(in, readerConfig)
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
