package outputs

type Observer interface {
	NewBatch(int)     // report new batch being processed with number of events
	Acked(int)        // report number of acked events
	Failed(int)       // report number of failed events
	Dropped(int)      // report number of dropped events
	Cancelled(int)    // report number of cancelled events
	WriteError(error) // report an I/O error on write
	WriteBytes(int)   // report number of bytes being written
	ReadError(error)  // report an I/O error on read
	ReadBytes(int)    // report number of bytes being read
}

type emptyObserver struct{}

var nilObserver = (*emptyObserver)(nil)

func (*emptyObserver) NewBatch(int)     {}
func (*emptyObserver) Acked(int)        {}
func (*emptyObserver) Failed(int)       {}
func (*emptyObserver) Dropped(int)      {}
func (*emptyObserver) Cancelled(int)    {}
func (*emptyObserver) WriteError(error) {}
func (*emptyObserver) WriteBytes(int)   {}
func (*emptyObserver) ReadError(error)  {}
func (*emptyObserver) ReadBytes(int)    {}
