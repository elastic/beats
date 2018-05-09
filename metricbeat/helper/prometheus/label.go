package prometheus

// LabelMap defines the mapping from Prometheus label to a Metricbeat field
type LabelMap interface {
	// GetField returns the resulting field name
	GetField() string

	// IsKey returns true if the label is a key label
	IsKey() bool
}

// Label maps a Prometheus label to a Metricbeat field
func Label(field string) LabelMap {
	return &commonLabel{
		field: field,
		key:   false,
	}
}

// KeyLabel maps a Prometheus label to a Metricbeat field. The label is flagged as key.
// Metrics with the same tuple of key labels will be grouped in the same event.
func KeyLabel(field string) LabelMap {
	return &commonLabel{
		field: field,
		key:   true,
	}
}

type commonLabel struct {
	field string
	key   bool
}

// GetField returns the resulting field name
func (l *commonLabel) GetField() string {
	return l.field
}

// IsKey returns true if the label is a key label
func (l *commonLabel) IsKey() bool {
	return l.key
}
