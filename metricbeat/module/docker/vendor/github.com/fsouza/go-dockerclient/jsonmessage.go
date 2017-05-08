package docker

// NOTE: This is a modified copy of the file https://github.com/moby/moby/tree/master/pkg/jsonmessage
// 	     This heavily reduces the number of packages which are a dependency.

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/jsonlog"
)

// JSONError wraps a concrete Code and Message, `Code` is
// is an integer error code, `Message` is the error message.
type JSONError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *JSONError) Error() string {
	return e.Message
}

// JSONProgress describes a Progress. terminalFd is the fd of the current terminal,
// Start is the initial value for the operation. Current is the current status and
// value of the progress made towards Total. Total is the end value describing when
// we made 100% progress for an operation.
type JSONProgress struct {
	terminalFd uintptr
	Current    int64 `json:"current,omitempty"`
	Total      int64 `json:"total,omitempty"`
	Start      int64 `json:"start,omitempty"`
	// If true, don't show xB/yB
	HideCounts bool `json:"hidecounts,omitempty"`
}

func (p *JSONProgress) String() string {
	return ""
}

// JSONMessage defines a message struct. It describes
// the created time, where it from, status, ID of the
// message. It's used for docker events.
type JSONMessage struct {
	Stream          string        `json:"stream,omitempty"`
	Status          string        `json:"status,omitempty"`
	Progress        *JSONProgress `json:"progressDetail,omitempty"`
	ProgressMessage string        `json:"progress,omitempty"` //deprecated
	ID              string        `json:"id,omitempty"`
	From            string        `json:"from,omitempty"`
	Time            int64         `json:"time,omitempty"`
	TimeNano        int64         `json:"timeNano,omitempty"`
	Error           *JSONError    `json:"errorDetail,omitempty"`
	ErrorMessage    string        `json:"error,omitempty"` //deprecated
	// Aux contains out-of-band data, such as digests for push signing.
	Aux *json.RawMessage `json:"aux,omitempty"`
}

/* Satisfied by gotty.TermInfo as well as noTermInfo from below */
type termInfo interface {
	Parse(attr string, params ...interface{}) (string, error)
}

type noTermInfo struct{} // canary used when no terminfo.

func (ti *noTermInfo) Parse(attr string, params ...interface{}) (string, error) {
	return "", fmt.Errorf("noTermInfo")
}

func clearLine(out io.Writer, ti termInfo) {
	// el2 (clear whole line) is not exposed by terminfo.

	// First clear line from beginning to cursor
	if attr, err := ti.Parse("el1"); err == nil {
		fmt.Fprintf(out, "%s", attr)
	} else {
		fmt.Fprintf(out, "\x1b[1K")
	}
	// Then clear line from cursor to end
	if attr, err := ti.Parse("el"); err == nil {
		fmt.Fprintf(out, "%s", attr)
	} else {
		fmt.Fprintf(out, "\x1b[K")
	}
}

func cursorUp(out io.Writer, ti termInfo, l int) {
	if l == 0 { // Should never be the case, but be tolerant
		return
	}
	if attr, err := ti.Parse("cuu", l); err == nil {
		fmt.Fprintf(out, "%s", attr)
	} else {
		fmt.Fprintf(out, "\x1b[%dA", l)
	}
}

func cursorDown(out io.Writer, ti termInfo, l int) {
	if l == 0 { // Should never be the case, but be tolerant
		return
	}
	if attr, err := ti.Parse("cud", l); err == nil {
		fmt.Fprintf(out, "%s", attr)
	} else {
		fmt.Fprintf(out, "\x1b[%dB", l)
	}
}

// Display displays the JSONMessage to `out`. `termInfo` is non-nil if `out`
// is a terminal. If this is the case, it will erase the entire current line
// when displaying the progressbar.
func (jm *JSONMessage) Display(out io.Writer, termInfo termInfo) error {
	if jm.Error != nil {
		if jm.Error.Code == 401 {
			return fmt.Errorf("Authentication is required.")
		}
		return jm.Error
	}
	var endl string
	if termInfo != nil && jm.Stream == "" && jm.Progress != nil {
		clearLine(out, termInfo)
		endl = "\r"
		fmt.Fprintf(out, endl)
	} else if jm.Progress != nil && jm.Progress.String() != "" { //disable progressbar in non-terminal
		return nil
	}
	if jm.TimeNano != 0 {
		fmt.Fprintf(out, "%s ", time.Unix(0, jm.TimeNano).Format(jsonlog.RFC3339NanoFixed))
	} else if jm.Time != 0 {
		fmt.Fprintf(out, "%s ", time.Unix(jm.Time, 0).Format(jsonlog.RFC3339NanoFixed))
	}
	if jm.ID != "" {
		fmt.Fprintf(out, "%s: ", jm.ID)
	}
	if jm.From != "" {
		fmt.Fprintf(out, "(from %s) ", jm.From)
	}
	if jm.Progress != nil && termInfo != nil {
		fmt.Fprintf(out, "%s %s%s", jm.Status, jm.Progress.String(), endl)
	} else if jm.ProgressMessage != "" { //deprecated
		fmt.Fprintf(out, "%s %s%s", jm.Status, jm.ProgressMessage, endl)
	} else if jm.Stream != "" {
		fmt.Fprintf(out, "%s%s", jm.Stream, endl)
	} else {
		fmt.Fprintf(out, "%s%s\n", jm.Status, endl)
	}
	return nil
}

// DisplayJSONMessagesStream displays a json message stream from `in` to `out`, `isTerminal`
// describes if `out` is a terminal. If this is the case, it will print `\n` at the end of
// each line and move the cursor while displaying.
func DisplayJSONMessagesStream(in io.Reader, out io.Writer, terminalFd uintptr, isTerminal bool, auxCallback func(*json.RawMessage)) error {
	var (
		dec = json.NewDecoder(in)
		ids = make(map[string]int)
	)

	var termInfo termInfo

	for {
		diff := 0
		var jm JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if jm.Aux != nil {
			if auxCallback != nil {
				auxCallback(jm.Aux)
			}
			continue
		}

		if jm.Progress != nil {
			jm.Progress.terminalFd = terminalFd
		}
		if jm.ID != "" && (jm.Progress != nil || jm.ProgressMessage != "") {
			line, ok := ids[jm.ID]
			if !ok {
				// NOTE: This approach of using len(id) to
				// figure out the number of lines of history
				// only works as long as we clear the history
				// when we output something that's not
				// accounted for in the map, such as a line
				// with no ID.
				line = len(ids)
				ids[jm.ID] = line
				if termInfo != nil {
					fmt.Fprintf(out, "\n")
				}
			}
			diff = len(ids) - line
			if termInfo != nil {
				cursorUp(out, termInfo, diff)
			}
		} else {
			// When outputting something that isn't progress
			// output, clear the history of previous lines. We
			// don't want progress entries from some previous
			// operation to be updated (for example, pull -a
			// with multiple tags).
			ids = make(map[string]int)
		}
		err := jm.Display(out, termInfo)
		if jm.ID != "" && termInfo != nil {
			cursorDown(out, termInfo, diff)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

type stream interface {
	io.Writer
	FD() uintptr
	IsTerminal() bool
}
