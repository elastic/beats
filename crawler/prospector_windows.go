package crawler

import (
	"os"
)

// TODO: Implement
func (p *Prospector) getPreviousFile(file string, info os.FileInfo) string {
	// Can we detect if a file was renamed on Windows?
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
	return ""
}
