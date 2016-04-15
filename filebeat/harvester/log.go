package harvester

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/filebeat/config"
	"github.com/elastic/beats/filebeat/harvester/encoding"
	"github.com/elastic/beats/filebeat/harvester/processor"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
	"golang.org/x/text/transform"
)

const (
	defaultMaxBytes = 10 * (1 << 20) // 10MB
)

func createLineReader(
	in FileSource,
	codec encoding.Encoding,
	bufferSize int,
	maxBytes int,
	readerConfig logFileReaderConfig,
	jsonConfig *config.JSONConfig,
	mlrConfig *config.MultilineConfig,
) (processor.LineProcessor, error) {
	var p processor.LineProcessor
	var err error

	fileReader, err := newLogFileReader(in, readerConfig)
	if err != nil {
		return nil, err
	}

	p, err = processor.NewLineSource(fileReader, codec, bufferSize)
	if err != nil {
		return nil, err
	}

	if jsonConfig != nil {
		p = processor.NewJSONProcessor(p, jsonConfig)
	}

	if mlrConfig != nil {
		p, err = processor.NewMultiline(p, maxBytes, mlrConfig)
		if err != nil {
			return nil, err
		}

		return processor.NewStripNewline(p), nil
	}

	p = processor.NewStripNewline(p)
	return processor.NewLimitProcessor(p, maxBytes), nil
}


func (h *Harvester) processException(text string) int {

	// no text to test
	if len(text) < 4 {
		return 0
	}

	isExtension := h.CurrentException.Name != "" && (strings.Contains(text, "Caused by: ") || strings.HasPrefix(strings.Trim(text, " "), "at ") || strings.HasPrefix(text, "... "))

	// not a continuation, if we have an exception, log it.
	if !isExtension && h.CurrentException.Name != "" {

		h.CurrentEvent.CustomS["ex_cause"] = h.CurrentException.Cause.String() + ""

		h.SpoolerChan <- h.CurrentEvent // ship the new event downstream

		h.CurrentException = LogException{} // make a new exception
	}


	// this is a multiline exception
	if isExtension {

		var expLen = len(h.Config.ExceptionPackages)


		// find the file and package name for the exception 
		if h.CurrentException.Name == "" {
			re := regexp.MustCompile("([a-zA-Z0-9\\.]+)\\(([a-zA-Z0-9]+\\.javax?):(\\d+)\\)")
			reg_result := re.FindStringSubmatch(text)
			if reg_result != nil {
				h.CurrentEvent.CustomS["ex_package"] = reg_result[1]
				h.CurrentEvent.CustomS["ex_file"] = reg_result[2]
				i64, err := strconv.ParseInt(reg_result[3], 10, 32)
				if err == nil {
					h.CurrentEvent.CustomI3["ex_line"] = i64
				}
			}
		}

		// load in the call stack
		if h.CurrentException.NumCause < h.Config.ExceptionMaxStack {
			var matched bool = false
			// is this package in the list of packages to track?
			for i := 0; i < expLen; i++ {
				if strings.Contains(text, h.Config.ExceptionPackages[i]) {
					h.CurrentException.Cause.WriteString("," + text[3:len(text)])
					matched = true
					h.CurrentException.NumCause += 1
					break
				}
			}

			// it's not so replace with a .
			if !matched {
				h.CurrentException.Cause.WriteString(".")
			}
		} else {
			// no packages to track use a .
			h.CurrentException.Cause.WriteString(".")
		}

		return 2
	}

	// find exceptions ...
	if strings.Contains(text, "Exception: ") || strings.Contains(text, "Exception; ") {

		// make sure we have additional storage
		if h.CurrentEvent.CustomS == nil {
			h.CurrentEvent.CustomS = make(map[string]string)
		}
		if h.CurrentEvent.CustomI3 == nil {
			h.CurrentEvent.CustomI3 = make(map[string]int64)
		}

		// caprture exception name and package
		re := regexp.MustCompile("([a-z\\.]*)\\.([A-Z][a-zA-Z\\.]*Exception)[:;]\\s+(.*)")
		reg_result := re.FindStringSubmatch(text)
		if reg_result != nil {
			h.CurrentEvent.CustomS["ex_package"] = reg_result[1] + ""
			h.CurrentEvent.CustomS["ex_name"] = reg_result[2] + ""
			h.CurrentException.Name = reg_result[2] + ""
			h.CurrentEvent.CustomS["ex_description"] = reg_result[3] + ""
			h.CurrentException.NumCause = 0
		} else {
			logp.Warn("   ****** Exception REGEX DID NOT MATCH", text)
		}

		return 3
	}

	return 0
}


// Log harvester reads files line by line and sends events to the defined output
func (h *Harvester) Harvest() {
	defer func() {
		// On completion, push offset so we can continue where we left off if we relaunch on the same file
		if h.Stat != nil {
			h.Stat.Return <- h.Offset
		}

		logp.Debug("harvester", "Stopping harvester for file: %s", h.Path)

		// Make sure file is closed as soon as harvester exits
		// If file was never properly opened, it can't be closed
		if h.file != nil {
			h.file.Close()
			logp.Debug("harvester", "Stopping harvester, closing file: %s", h.Path)
		} else {
			logp.Debug("harvester", "Stopping harvester, NOT closing file as file info not available: %s", h.Path)
		}
	}()

	enc, err := h.open()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file opening error: %s", err)
		return
	}

	info, err := h.file.Stat()
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected file stat rror: %s", err)
		return
	}

	logp.Info("Harvester started for file: %s", h.Path)

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	config := h.Config
	readerConfig := logFileReaderConfig{
		forceClose:         config.ForceCloseFiles,
		closeOlder:         config.CloseOlderDuration,
		backoffDuration:    config.BackoffDuration,
		maxBackoffDuration: config.MaxBackoffDuration,
		backoffFactor:      config.BackoffFactor,
	}

	reader, err := createLineReader(
		h.file, enc, config.BufferSize, config.MaxBytes, readerConfig,
		config.JSON, config.Multiline)
	if err != nil {
		logp.Err("Stop Harvesting. Unexpected encoding line reader error: %s", err)
		return
	}

	Loop:
	for {
		// Partial lines return error and are only read on completion
		ts, text, bytesRead, jsonFields, err := readLine(reader)
		if err != nil {
			if err == errFileTruncate {
				seeker, ok := h.file.(io.Seeker)
				if !ok {
					logp.Err("can not seek source")
					return
				}

				logp.Info("File was truncated. Begin reading file from offset 0: %s", h.Path)

				h.Offset = 0
				seeker.Seek(h.Offset, os.SEEK_SET)
				continue
			}

			logp.Info("Read line error: %s", err)
			return
		}
		h.Lineno += 1

		if h.shouldExportLine(text) {

			// if we aren't working on a current exception
			if h.CurrentException.Name == "" {
				//fmt.Println("  create event.")
				h.CurrentEvent = &input.FileEvent{
					EventMetadata: h.Config.EventMetadata,
					ReadTime:      ts,
					Source:        &h.Path,
					InputType:     h.Config.InputType,
					DocumentType:  h.Config.DocumentType,
					Offset:        h.Offset,
					Bytes:         bytesRead,
					// TODO remove Text
					//Text:     &text,
					Fileinfo: &info,
					JSONFields:    jsonFields,
					JSONConfig:    h.Config.JSON,
				}
			}


			// have a regular expression to map fields
			if h.Config.RegexList != nil {
				m := 0
				x := 0
				h.CurrentEvent.CustomS = make(map[string]string)
				h.CurrentEvent.CustomI3 = make(map[string]int64)

				NextRegex:
				for x := 0; x < len(h.Config.RegexList); x++ {

					bigregex := ""
					for i := 0; i < len(h.Config.RegexList[x].Regex); i++ {
						prop := h.Config.RegexList[x].Regex[i]
						bigregex += prop.Regex
					}
					reg_result := h.RegexList[x].FindStringSubmatch(text)


					var i = -1
					for k := range reg_result {
						v := reg_result[k]

						if i >= 0 && i < len(h.Config.RegexList[x].Regex) {

							prop := h.Config.RegexList[x].Regex[i]
							if prop.Name != "ignore" {
								h.CurrentEvent.CustomS[prop.Name] = v
								logp.Debug("(%s) %s\n", prop.Name, v)

								// we will process "message" for exceptions
								if prop.Name == "message" {
									var exproc = h.processException(h.CurrentEvent.CustomS[prop.Name])
									// continuation of an exception, get the next line
									if exproc > 1 {
										continue Loop
									}
									// the exception is done, log this line
								}
							}
						}
						i++
					}


					if i > -1 {
						logp.Debug("harvester", " OOO-> " + "[" + h.Config.RegexList[x].Name + "] regex matched: " + bigregex + "|" + text)
						if h.Config.RegexList[x].IncludeMessage {
							h.CurrentEvent.Text = &text
						}
						break NextRegex
					} 					
					m++
				}
				if x >= len(h.Config.RegexList) {
					logp.Warn("harvester", " XXX-> line: " + strconv.FormatInt(h.Lineno, 10) + " all regexs " + strconv.Itoa(m) + " failed to match: " + text)
					h.CurrentEvent.CustomI3["unmatched"] = 1
					h.CurrentEvent.Text = &text
				} 
			} else { 
				// no regular expression field mapping so just process the text
				h.CurrentEvent.Text = &text
			}

			h.SpoolerChan <- h.CurrentEvent // ship the new event downstream
		}

		// Set Offset
		h.Offset += int64(bytesRead) // Update offset if complete line has been processed
	}
}

// shouldExportLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (h *Harvester) shouldExportLine(line string) bool {
	if len(h.IncludeLinesRegexp) > 0 {
		if !MatchAnyRegexps(h.IncludeLinesRegexp, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does not match any of the include patterns %s", line)
			return false
		}
	}
	if len(h.ExcludeLinesRegexp) > 0 {
		if MatchAnyRegexps(h.ExcludeLinesRegexp, line) {
			// drop line
			logp.Debug("harvester", "Drop line as it does match one of the exclude patterns%s", line)
			return false
		}
	}

	return true
}

// open does open the file given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() (encoding.Encoding, error) {
	// Special handling that "-" means to read from standard input
	if h.Config.InputType == config.StdinInputType {
		return h.openStdin()
	}
	return h.openFile()
}

func (h *Harvester) openStdin() (encoding.Encoding, error) {
	h.file = pipeSource{os.Stdin}
	return h.encoding(h.file)
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (h *Harvester) openFile() (encoding.Encoding, error) {
	var file *os.File
	var err error
	var encoding encoding.Encoding

	file, err = input.ReadOpen(h.Path)
	if err == nil {
		// Check we are not following a rabbit hole (symlinks, etc.)
		if !input.IsRegularFile(file) {
			return nil, errors.New("Given file is not a regular file.")
		}

		encoding, err = h.encoding(file)
		if err != nil {

			if err == transform.ErrShortSrc {
				logp.Info("Initialising encoding for '%v' failed due to file being too short", file)
			} else {
				logp.Err("Initialising encoding for '%v' failed: %v", file, err)
			}
			return nil, err
		}

	} else {
		logp.Err("Failed opening %s: %s", h.Path, err)
		return nil, err
	}

	// update file offset
	err = h.initFileOffset(file)
	if err != nil {
		return nil, err
	}

	// yay, open file
	h.file = fileSource{file}
	return encoding, nil
}

func (h *Harvester) initFileOffset(file *os.File) error {
	offset, err := file.Seek(0, os.SEEK_CUR)

	if h.Offset > 0 {
		// continue from last known offset

		logp.Debug("harvester",
			"harvest: %q position:%d (offset snapshot:%d)", h.Path, h.Offset, offset)
		_, err = file.Seek(h.Offset, os.SEEK_SET)
	} else if h.Config.TailFiles {
		// tail file if file is new and tail_files config is set

		logp.Debug("harvester",
			"harvest: (tailing) %q (offset snapshot:%d)", h.Path, offset)
		h.Offset, err = file.Seek(0, os.SEEK_END)

	} else {
		// get offset from file in case of encoding factory was
		// required to read some data.
		logp.Debug("harvester", "harvest: %q (offset snapshot:%d)", h.Path, offset)
		h.Offset = offset
	}

	return err
}
