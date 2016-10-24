package diskio

import (
	"testing"
	"time"
)

var blkioService BLkioService
var oldBlkioRaw = make([]BlkioRaw, 3)
var newBlkioRaw = make([]BlkioRaw, 3)

func TestWritePs(t *testing.T) {
	oldWritePs := []uint64{220, 951, 0}
	newWritePs := []uint64{120, 2951, 0}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].writes = oldWritePs[index]
		newBlkioRaw[index].writes = newWritePs[index]
	}
	writePsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 0},
		{oldBlkioRaw[1], newBlkioRaw[1], 1000},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range writePsTest {
		out := blkioService.getWritePs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getWritePs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}
func TestReadPS(t *testing.T) {
	oldReasPs := []uint64{0, 951, 235}
	newReadPs := []uint64{120, 3951, 62}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].reads = oldReasPs[index]
		newBlkioRaw[index].reads = newReadPs[index]
	}
	readPsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 60},
		{oldBlkioRaw[1], newBlkioRaw[1], 1500},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range readPsTest {
		out := blkioService.getReadPs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getReadPs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}
func TestBlkioTotal(t *testing.T) {
	oldTotal := []uint64{40, 1954, 235}
	newTotal := []uint64{120, 1964, 62}
	for index := range oldBlkioRaw {
		setTime(index)
		oldBlkioRaw[index].totals = oldTotal[index]
		newBlkioRaw[index].totals = newTotal[index]
	}
	totalPsTest := []struct {
		givenOld BlkioRaw
		givenNew BlkioRaw
		expected float64
	}{
		{oldBlkioRaw[0], newBlkioRaw[0], 40},
		{oldBlkioRaw[1], newBlkioRaw[1], 5},
		{oldBlkioRaw[2], newBlkioRaw[2], 0},
	}
	for _, tt := range totalPsTest {
		out := blkioService.getTotalPs(&tt.givenOld, &tt.givenNew)
		if out != tt.expected {
			t.Errorf("getTotalPs(%v,%v) => %v, want %v", tt.givenOld, tt.givenNew, out, tt.expected)
		}
	}
}
func setTime(index int) {
	oldBlkioRaw[index].Time = time.Now()
	newBlkioRaw[index].Time = oldBlkioRaw[index].Time.Add(time.Duration(2000000000))
}
