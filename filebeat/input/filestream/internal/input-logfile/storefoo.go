package input_logfile

import (
	inpFile "github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/common/file"
)

type TakeOverState struct {
	FileStateOS    file.StateOS
	IdentifierName string
	Key            string
	Meta           map[string]string
	Source         string

	// used in store.TakeOver
	logInpOffset int64
}

// newTakeOverState creates a TakeOverState populated by the logSt or filestreamSt.
// If filestreamSt is not nil, it is used, otherwise logSt is used.
// An error is only returned if filestreamSt.UnpackCursorMeta fails, the only
// reason for it to fail is if the data format in the store has changed.
func newTakeOverState(logSt inpFile.State, filestreamSt *resource) (TakeOverState, error) {
	st := TakeOverState{}

	if filestreamSt != nil {
		meta := struct {
			IdentifierName string `json:"identifier_name" struct:"identifier_name"`
			Source         string `json:"source" struct:"source"`
		}{}

		if err := filestreamSt.UnpackCursorMeta(&meta); err != nil {
			return st, err
		}

		st.Source = meta.Source
		st.IdentifierName = meta.IdentifierName
		st.Key = filestreamSt.key

		return st, nil
	}

	st.Source = logSt.Source
	st.Meta = logSt.Meta
	st.IdentifierName = logSt.IdentifierName
	st.FileStateOS = logSt.FileStateOS
	st.logInpOffset = logSt.Offset
	st.Key = logSt.Id

	return st, nil
}
