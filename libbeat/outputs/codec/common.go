package codec

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	structform "github.com/urso/go-structform"
)

func TimestampEncoder(t *time.Time, v structform.ExtVisitor) error {
	content, err := common.Time(*t).MarshalJSON()
	if err != nil {
		return err
	}

	return v.OnStringRef(content[1 : len(content)-1])
}

func BcTimestampEncoder(t *common.Time, v structform.ExtVisitor) error {
	content, err := t.MarshalJSON()
	if err != nil {
		return err
	}
	return v.OnStringRef(content[1 : len(content)-1])
}
