package crawler

import (
	"os"
)

// TODO: Implement
func (p *Prospector) isFileRenamed(file string, info os.FileInfo) string {
	// Can we detect if a file was renamed on Windows?
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
	return ""
}

func (c *Crawler) isFileRenamed(file string, info os.FileInfo) string {
	// Can we detect if a file was renamed on Windows?
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
	// TODO: Check http://stackoverflow.com/questions/562701/best-way-to-determine-if-two-path-reference-to-same-file-in-windows
	return ""
}
