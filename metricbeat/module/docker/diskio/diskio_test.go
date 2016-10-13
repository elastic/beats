package diskio

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var blkioService BLkioService
var oldBlkioRaw BlkioRaw
var newBLkioRaw BlkioRaw

func TestWritePs(t *testing.T) {
	setTime()
	oldBlkioRaw.writes = 951
	newBLkioRaw.writes = 2951
	value := blkioService.getWritePs(&oldBlkioRaw, &newBLkioRaw)
	t.Logf("value : %v", value)
	assert.Equal(t, float64(1000), value)
}
func TestReadPS(t *testing.T) {
	setTime()
	oldBlkioRaw.reads = 995
	newBLkioRaw.reads = 1995
	value := blkioService.getReadPs(&oldBlkioRaw, &newBLkioRaw)
	assert.Equal(t, float64(500), value)
}
func TestBlkioTotal(t *testing.T) {
	setTime()
	oldBlkioRaw.totals = 1954
	newBLkioRaw.totals = 1964
	value := blkioService.getTotalPs(&oldBlkioRaw, &newBLkioRaw)
	assert.Equal(t, float64(5), value)
}
func setTime() {
	oldBlkioRaw.Time = time.Now()
	newBLkioRaw.Time = oldBlkioRaw.Time.Add(time.Duration(2000000000))
}
