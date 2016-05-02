package input

import "os"

// Contains statistic about file when it was last seen by the prospector
type FileStat struct {
	Fileinfo      os.FileInfo /* the file info */
	Offset        chan int64  /* the harvester will send an event with its offset when it closes */
	LastIteration uint32      /* int number of the last iterations in which we saw this file */
}

func NewFileStat(fi os.FileInfo, lastIteration uint32) *FileStat {
	fs := &FileStat{
		Fileinfo:      fi,
		Offset:        make(chan int64, 1),
		LastIteration: lastIteration,
	}
	return fs
}

func (fs *FileStat) Finished() bool {
	return len(fs.Offset) != 0
}

// Ignore forgets about the previous harvester results and let it continue on the old
// file - start a new channel to use with the new harvester.
func (fs *FileStat) Ignore() {
	fs.Offset = make(chan int64, 1)
}

func (fs *FileStat) Continue(old *FileStat) {
	if old != nil {
		fs.Offset = old.Offset
	}
}

func (fs *FileStat) Skip(newOffset int64) {
	fs.Offset <- newOffset
}
