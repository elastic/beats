package crawler

import (
	"testing"
)

func TestOpen(t *testing.T) {

	h := Harvester{
		Path: "/var/log/",
		Offset: 0,
	}

	h.open()


}
