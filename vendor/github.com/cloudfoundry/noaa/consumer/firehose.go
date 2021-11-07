package consumer

type firehose struct {
	subscriptionID string
	authToken      string
	retry          bool
	envelopeFilter EnvelopeFilter
}

type FirehoseOption func(*firehose)

func WithRetry(retry bool) FirehoseOption {
	return func(f *firehose) {
		f.retry = retry
	}
}
func WithEnvelopeFilter(filter EnvelopeFilter) FirehoseOption {
	return func(f *firehose) {
		f.envelopeFilter = filter
	}
}

func newFirehose(
	subID string,
	authToken string,
	opts ...FirehoseOption,
) *firehose {
	f := &firehose{
		subscriptionID: subID,
		authToken:      authToken,
		retry:          true,
		envelopeFilter: allEnvelopes,
	}

	for _, o := range opts {
		o(f)
	}

	return f
}

func (f *firehose) streamPath() string {
	return "/firehose/" + f.subscriptionID + "?" + f.envelopeFilter.queryStringParam()
}
