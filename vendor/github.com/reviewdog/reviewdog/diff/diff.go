// Package diff provides a utility to parse unified diff.
// https://en.wikipedia.org/wiki/Diff_utility#Unified_format
package diff

// FileDiff represents a unified diff for a single file.
//
// Example:
//   --- oldname	2009-10-11 15:12:20.000000000 -0700
//   +++ newname	2009-10-11 15:12:30.000000000 -0700
type FileDiff struct {
	// the old path of the file
	PathOld string
	// the new path of the file
	PathNew string

	// the old timestamp (empty if not present)
	TimeOld string
	// the new timestamp (empty if not present)
	TimeNew string

	Hunks []*Hunk

	// extended header lines (e.g., git's "new mode <mode>", "rename from <path>", index fb14f33..c19311b 100644, etc.)
	Extended []string

	// TODO: we may want `\ No newline at end of file` information for both the
	// old and new file.
}

// Hunk represents change hunks that contain the line differences in the file.
//
// Example:
//   @@ -1,3 +1,4 @@ optional section heading
//    unchanged, contextual line
//   -deleted line
//   +added line
//   +added line
//    unchanged, contextual line
//
//  -1 -> the starting line number of the old file
//  3  -> the number of lines the change hunk applies to for the old file
//  +1 -> the starting line number of the new file
//  4  -> the number of lines the change hunk applies to for the new file
type Hunk struct {
	// the starting line number of the old file
	StartLineOld int
	// the number of lines the change hunk applies to for the old file
	LineLengthOld int

	// the starting line number of the new file
	StartLineNew int
	// the number of lines the change hunk applies to for the new file
	LineLengthNew int

	// optional section heading
	Section string

	// the body lines of the hunk
	Lines []*Line
}

// LineType represents the type of diff line.
type LineType int

const (
	// LineUnchanged represents unchanged, contextual line preceded by ' '
	LineUnchanged LineType = iota
	// LineAdded represents added line preceded by '+'
	LineAdded
	// LineDeleted represents deleted line preceded by '-'
	LineDeleted
)

// Line represents a diff line.
type Line struct {
	// type of this line
	Type LineType
	// the line content without a preceded character (' ', '+', '-')
	Content string

	// the line in the file to a position in the diff.
	// the number of lines down from the first "@@" hunk header in the file.
	// e.g. The line just below the "@@" line is position 1, the next line is
	// position 2, and so on. The position in the file's diff continues to
	// increase through lines of whitespace and additional hunks until a new file
	// is reached. It's equivalent to the `position` field of input for cooment
	// API of GitHub https://developer.github.com/v3/pulls/comments/#input
	LnumDiff int

	// the line number of the old file for LineUnchanged and LineDeleted
	// type. 0 for LineAdded type.
	LnumOld int

	// the line number of the new file for LineUnchanged and LineAdded type.
	// 0 for LineDeleted type.
	LnumNew int
}
