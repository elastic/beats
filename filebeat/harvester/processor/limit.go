package processor

// LimitProcessor sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type LimitProcessor struct {
	reader   LineProcessor
	maxBytes int
}

// NewLimitProcessor creates a new processor limiting the line length.
func NewLimitProcessor(in LineProcessor, maxBytes int) *LimitProcessor {
	return &LimitProcessor{reader: in, maxBytes: maxBytes}
}

// Next returns the next line.
func (p *LimitProcessor) Next() (Line, error) {
	line, err := p.reader.Next()
	if len(line.Content) > p.maxBytes {
		line.Content = line.Content[:p.maxBytes]
	}
	return line, err
}
