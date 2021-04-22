package mongodbatlas

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type mongoDbParser struct {
	publisher cursor.Publisher
	reader    *bufio.Reader
	writer    io.Writer
	log       *logp.Logger
	gz *gzip.Reader
}

func NewParser(publisher cursor.Publisher, log *logp.Logger) *mongoDbParser {
	r, w := io.Pipe()
	fmt.Printf("GZIP init")
	gz, err := gzip.NewReader(r)
	fmt.Printf("GZIP reader done")
	if err != nil {
		fmt.Printf("GZIP FAIL, %v", err)
	}
	reader := bufio.NewReader(gz)
	return &mongoDbParser{
		publisher: publisher,
		reader:    reader,
		writer:    w,
		log:       log,
		gz : gz,
	}
}

func (parser *mongoDbParser) Writer() io.Writer {
	return parser.writer
}

func (parser *mongoDbParser) Close() error {
	parser.log.Debugf("Closing pipe")
	parser.gz.Close()
	err := parser.Publish()
	return err
}

func (parser *mongoDbParser) Publish() error {
	parser.log.Debugf("Do some publish!")
	for {
		line, _, err := parser.reader.ReadLine()
		os.Stdout.Write(line)
		if err != nil {
			return err
		}
	}
	return nil
}
