package util

import dto "github.com/prometheus/client_model/go"

// GetLabel returns desired label from the given metric, or "" if not present
func GetLabel(m *dto.Metric, label string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == label {
			return l.GetValue()
		}
	}
	return ""
}
