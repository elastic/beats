package sniffer

import (
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

type Dumper struct {
	writer *pcapgo.Writer
	file   *os.File
}

func newPcapDumper(
	filename string,
	lt layers.LinkType,
	snaplen uint32,
) (*Dumper, error) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	perm := os.FileMode(0666)
	file, err := os.OpenFile(filename, flags, perm)
	if err != nil {
		return nil, err
	}

	writer := pcapgo.NewWriter(file)
	if err := writer.WriteFileHeader(snaplen, lt); err != nil {
		file.Close()
		return nil, err
	}

	return &Dumper{file: file, writer: writer}, nil
}

func (d *Dumper) Close() {
	d.file.Close()
}

func (d *Dumper) WritePacket(ci gopacket.CaptureInfo, data []byte) error {
	return d.writer.WritePacket(ci, data)
}
