package consumer

type EnvelopeFilter int

const (
	LogMessages EnvelopeFilter = iota
	Metrics
	allEnvelopes
)

func (f EnvelopeFilter) queryStringParam() string {
	switch f {
	case LogMessages:
		return "filter-type=logs"
	case Metrics:
		return "filter-type=metrics"
	default:
		return ""
	}
}
