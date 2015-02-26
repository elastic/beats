package inputs

import "packetbeat/common"

type InputPlugin interface {
	Init(events chan common.MapStr)
	Run()
	Stop()
	Close()
}
