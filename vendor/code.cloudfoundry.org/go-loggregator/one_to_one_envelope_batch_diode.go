package loggregator

import (
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	gendiodes "code.cloudfoundry.org/go-diodes"
)

// OneToOneEnvelopeBatch diode is optimized for a single writer and a single reader
type OneToOneEnvelopeBatch struct {
	d *gendiodes.Poller
}

// NewOneToOneEnvelopeBatch initializes a new one to one diode for envelope
// batches of a given size and alerter. The alerter is called whenever data is
// dropped with an integer representing the number of envelope batches that
// were dropped.
func NewOneToOneEnvelopeBatch(size int, alerter gendiodes.Alerter, opts ...gendiodes.PollerConfigOption) *OneToOneEnvelopeBatch {
	return &OneToOneEnvelopeBatch{
		d: gendiodes.NewPoller(gendiodes.NewOneToOne(size, alerter), opts...),
	}
}

// Set inserts the given V2 envelope into the diode.
func (d *OneToOneEnvelopeBatch) Set(data []*loggregator_v2.Envelope) {
	d.d.Set(gendiodes.GenericDataType(&data))
}

// TryNext returns the next envelope batch to be read from the diode. If the
// diode is empty it will return a nil envelope and false for the bool.
func (d *OneToOneEnvelopeBatch) TryNext() ([]*loggregator_v2.Envelope, bool) {
	data, ok := d.d.TryNext()
	if !ok {
		return nil, ok
	}

	return *(*[]*loggregator_v2.Envelope)(data), true
}

// Next will return the next envelope batch to be read from the diode. If the
// diode is empty this method will block until anenvelope is available to be
// read.
func (d *OneToOneEnvelopeBatch) Next() []*loggregator_v2.Envelope {
	data := d.d.Next()
	return *(*[]*loggregator_v2.Envelope)(data)
}
