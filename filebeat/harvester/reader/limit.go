package reader

// LimitProcessor sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type Limit struct {
	reader   Reader
	maxBytes int
}

// NewLimit creates a new reader limiting the line length.
func NewLimit(r Reader, maxBytes int) *Limit {
	return &Limit{reader: r, maxBytes: maxBytes}
}

// Next returns the next line.
func (p *Limit) Next() (Message, error) {
	message, err := p.reader.Next()
	if len(message.Content) > p.maxBytes {
		message.Content = message.Content[:p.maxBytes]
	}
	return message, err
}
