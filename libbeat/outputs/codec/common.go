package codec

import (
	"time"

	"github.com/elastic/go-structform"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/dtfmt"
)

func MakeTimestampEncoder() func(*time.Time, structform.ExtVisitor) error {
	formatter, err := dtfmt.NewFormatter("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'")
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 0, formatter.EstimateSize())
	return func(t *time.Time, v structform.ExtVisitor) error {
		tmp, err := formatter.AppendTo(buf, (*t).UTC())
		if err != nil {
			return err
		}

		buf = tmp[:0]
		return v.OnStringRef(tmp)
	}
}

func MakeBCTimestampEncoder() func(*common.Time, structform.ExtVisitor) error {
	enc := MakeTimestampEncoder()
	return func(t *common.Time, v structform.ExtVisitor) error {
		return enc((*time.Time)(t), v)
	}
}
