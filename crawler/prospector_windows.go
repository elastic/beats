package crawler

// TODO: Implement
func (p *Prospector) isFileRenamed(file string, info os.FileInfo, missingfiles map[string]os.FileInfo) string {
	// Can we detect if a file was renamed on Windows?
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
	return ""
}

func (p *Prospector) isFileRenamedResumelist(file string, info os.FileInfo, initial map[string]*FileState) string {
	// Can we detect if a file was renamed on Windows?
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool?
	return ""
}
