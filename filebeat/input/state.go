package input

type FileState struct {
	Source      string `json:"source,omitempty"`
	Offset      int64  `json:"offset,omitempty"`
	FileStateOS *FileStateOS
}
