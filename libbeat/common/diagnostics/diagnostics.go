package diagnostics

// DiagnosticRunner is a sub-type of the runner interface found as cfgfile.Runner
// Runners can implement this if they want their client filesets/metricsets to expose the underlying DiagnosticSet interface
type DiagnosticRunner interface {
	ModuleDiagnostics() []DiagnosticSetup
}

// DiagnosticSet is an interface that a metricset/fileset can implement to provide additional Diagnostic data.
// A metricset can provide any number of diagnostic responses when requested.
type DiagnosticSet interface {
	// DiagnosticSetup returns metadata and a callback halder.
	// note that this can be called any time after a metricset has started, so implementors should not assume
	// the state of a metricset/fileset when this method is called.
	DiagnosticSetup() []DiagnosticSetup
}

// DiagnosticSetup contains the data needed to register a callback.
type DiagnosticSetup struct {
	// The name of this diagnostics data result.
	Name string
	// A brief description of the file.
	Description string
	// The filename that the requester should save the body as. This value must be unique for all other diagnostics in the metricset/fileset
	Filename string
	// MIME/ContentType. See https://www.iana.org/assignments/media-types/media-types.xhtml
	ContentType string
	//Callback is called when diagnostic data is actually requested by central management.
	// Callback does not return an error, and if one occours, it should be written out as the result.
	Callback func() []byte
}
