// Copyright 2017 Marcus Heese
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code based on logic from journalctl unit filter. i.e. journalctl -u in
// the systemd source code.
// See: https://github.com/systemd/systemd/blob/master/src/journal/journalctl.c#L1410
// and https://github.com/systemd/systemd/blob/master/src/basic/unit-name.c

package reader

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/danwakefield/fnmatch" // port of c function fnmatch to pure go
)

const (
	unitNameMax      = 256
	globChars        = "*?["
	uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
	letters          = lowercaseLetters + uppercaseLetters
	digits           = "0123456789"
	validChars       = digits + letters + ":-_.\\"
	validCharsWithAt = "@" + validChars
	validCharsGlob   = validCharsWithAt + "[]!-*?"
)

var (
	systemUnits = []string{
		"_SYSTEMD_UNIT",
		"COREDUMP_UNIT",
		"UNIT",
		"OBJECT_SYSTEMD_UNIT",
		"_SYSTEMD_SLICE",
	}

	unitTypes = []string{
		".service",
		".socket",
		".target",
		".device",
		".mount",
		".automount",
		".swap",
		".target",
		".path",
		".timer",
		".snapshot",
		".slice",
		".scope",
	}
)

// Add units to monitor
func (r *Reader) addUnits() error {
	var patterns []string

	// add specific units to monitor if any
	for _, unit := range r.config.Units {
		unit, err := unitNameMangle(unit, ".service")
		if err != nil {
			return fmt.Errorf("filtering unit %s failed: %+v", unit, err)
		}

		if stringIsGlob(unit) {
			patterns = append(patterns, unit)
		} else {
			if err = r.addMatchesForUnit(unit); err != nil {
				return fmt.Errorf("filtering unit %s failed: %+v", unit, err)
			}
		}
	}

	// Now add glob pattern matches if/any
	if len(patterns) > 0 {
		var units []string
		units = r.getPossibleUnits(systemUnits, patterns)
		for _, unit := range units {
			if err := r.addMatchesForUnit(unit); err != nil {
				return fmt.Errorf("filtering unit %s failed: %+v", unit, err)
			}
		}
	}

	r.logger.Debugf("Added matcher expression to filter units %+v", r.config.Units)

	return nil
}

// See: https://github.com/systemd/systemd/blob/master/src/shared/logs-show.c#L1114
func (r *Reader) addMatchesForUnit(unit string) error {
	// Wrap AddMatch/AddDisjunction with function literal to avoid repeated checks against err.
	var err error
	AddMatch := func(s string) {
		if err == nil {
			err = r.journal.AddMatch(s)
		}
	}

	AddDisjunction := func() {
		if err == nil {
			err = r.journal.AddDisjunction()
		}
	}

	// Look for messages from the service itself
	AddMatch("_SYSTEMD_UNIT=" + unit)

	// Look for coredumps of the service
	AddDisjunction()
	AddMatch("MESSAGE_ID=fc2e22bc6ee647b6b90729ab34a250b1")
	AddMatch("_UID=0")
	AddMatch("COREDUMP_UNIT=" + unit)

	// Look for messages from PID 1 about this service
	AddDisjunction()
	AddMatch("_PID=1")
	AddMatch("UNIT=" + unit)

	// Look for messages from authorized daemons about this service
	AddDisjunction()
	AddMatch("_UID=0")
	AddMatch("OBJECT_SYSTEMD_UNIT=" + unit)

	// Show all messages belonging to a slice
	if err == nil && strings.HasSuffix(unit, ".slice") {
		AddDisjunction()
		AddMatch("_SYSTEMD_SLICE=" + unit)
	}

	AddDisjunction()
	return err
}

//  Convert a string to a unit name. /dev/blah is converted to dev-blah.device,
//  /blah/blah is converted to blah-blah.mount, anything else is left alone,
//  except that "suffix" is appended if a valid unit suffix is not present.

//  If allowGlobs, globs characters are preserved. Otherwise, they are escaped.
func unitNameMangle(name, suffix string) (string, error) {
	// Can't be empty or begin with a dot
	if len(name) == 0 || name[0] == '.' {
		return "", errors.New("unit name can't be empty or begin with a dot")
	}

	if !unitSuffixIsValid(suffix) {
		return "", errors.New("unit name has an invalid suffix")
	}

	// already a fully valid unit name?
	if unitNameIsValid(name) {
		return name, nil
	}

	// Already a fully valid globbing expression? If so, no mangling is necessary either...
	if stringIsGlob(name) && inCharset(name, validCharsGlob) {
		return name, nil
	}

	if isDevicePath(name) {
		// chop off path and put .device on the end
		return path.Base(path.Clean(name)) + "device", nil
	}

	if pathIsAbsolute(name) {
		// chop path and put .mount on the end
		return path.Base(path.Clean(name)) + ".mount", nil
	}

	name = doEscapeMangle(name)

	// Append a suffix if it doesn't have any, but only if this is not a glob,
	// so that we can allow "foo.*" as a valid glob.
	if !stringIsGlob(name) && !strings.ContainsAny(name, ".") {
		return name + suffix, nil
	}

	return name, nil
}

// Mangle the unit name.
func doEscapeMangle(name string) string {
	var mangled string
	for _, r := range name {
		if r == '/' {
			mangled += "-"
		} else if !strings.ContainsRune(validChars, r) {
			mangled += "\\x" + strconv.FormatInt(int64(r), 16)
		} else {
			mangled += string(r)
		}
	}
	return mangled
}

// Check if this is a valid systemd unit name
func unitNameIsValid(name string) bool {
	if len(name) >= unitNameMax {
		return false
	}

	dot := strings.Index(name, ".")

	// Must have a dot (i.e. suffix)
	if dot == -1 {
		return false
	}

	suffix := name[dot:]

	// Must end with a valid suffix
	if !unitSuffixIsValid(suffix) {
		return false
	}

	// name must only consist of characters from validChars + "@"
	if !inCharset(name, validCharsWithAt) {
		return false
	}

	at := strings.Index(name, "@")

	// Can't start with '@'
	if at == 0 {
		return false
	}

	// Plain unit (not a template or instance) or a template or instance
	if at == -1 || at > 0 && dot >= at+1 {
		return true
	}

	return false
}

func (r *Reader) getPossibleUnits(fields, patterns []string) []string {
	var found []string
	var possibles []string

	for _, field := range fields {
		var vals, err = r.journal.GetUniqueValues(field)
		if err != nil {
			continue
		}

		// Split at '=' and check against all patterns (actually GetUniqueValues does the '=' split for us)
		possibles = append(possibles, vals...)
	}

	// filter whole possibles list against patterns and append matches to found list
	for _, possible := range possibles {
		for _, pattern := range patterns {
			if fnmatch.Match(pattern, possible, fnmatch.FNM_NOESCAPE) {
				found = append(found, possible)
				break
			}
		}
	}

	return found
}

// Check for valid unit name suffix
func unitSuffixIsValid(name string) bool {
	if len(name) == 0 {
		return false
	}

	if name[0] != '.' {
		return false
	}

	// Unit type from string
	for _, unit := range unitTypes {
		if strings.HasSuffix(name, unit) {
			return true
		}
	}

	return false
}

func inCharset(s, charset string) bool {
	for _, char := range s {
		if !strings.Contains(charset, string(char)) {
			return false
		}
	}
	return true
}

// Returns true on paths that refer to a device, either in sysfs or in /dev
func isDevicePath(path string) bool {
	return strings.HasPrefix(path, "/dev/") || strings.HasPrefix(path, "/sys/")
}

// Absolute paths begin with a slash
func pathIsAbsolute(path string) bool {
	return path[0] == '/'
}

// Return true if the provided string contains any glob chars
func stringIsGlob(name string) bool {
	return strings.ContainsAny(name, globChars)
}
