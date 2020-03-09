package service

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	dbusRaw "github.com/godbus/dbus"
	"github.com/pkg/errors"
)

type unitFetcher func(conn *dbus.Conn, states, patterns []string) ([]dbus.UnitStatus, error)

// instrospectForUnitMethods determines what methods are available via dbus for listing systemd units.
// We have a number of functions, some better than others, for getting and filtering unit lists.
// This will attempt to find the most optimal method, and move down to methods that require more work.
func instrospectForUnitMethods() (unitFetcher, error) {
	//setup a dbus connection
	conn, err := dbusRaw.SystemBusPrivate()
	if err != nil {
		return nil, errors.Wrap(err, "error getting connection to system bus")
	}

	auth := dbusRaw.AuthExternal(strconv.Itoa(os.Getuid()))
	err = conn.Auth([]dbusRaw.Auth{auth})
	if err != nil {
		return nil, errors.Wrap(err, "error authenticating")
	}

	err = conn.Hello()
	if err != nil {
		return nil, errors.Wrap(err, "error in Hello")
	}

	var props string

	//call "introspect" on the systemd1 path to see what ListUnit* methods are available
	obj := conn.Object("org.freedesktop.systemd1", dbusRaw.ObjectPath("/org/freedesktop/systemd1"))
	err = obj.Call("org.freedesktop.DBus.Introspectable.Introspect", 0).Store(&props)
	if err != nil {
		return nil, errors.Wrap(err, "error calling dbus")
	}

	unitMap, err := parseXMLAndReturnMethods(props)
	if err != nil {
		return nil, errors.Wrap(err, "error handling XML")
	}

	//return a function callback ordered by desirability
	if _, ok := unitMap["ListUnitsByPatterns"]; ok {
		return listUnitsByPatternWrapper, nil
	} else if _, ok := unitMap["ListUnitsFiltered"]; ok {
		return listUnitsFilteredWrapper, nil
	} else if _, ok := unitMap["ListUnits"]; ok {
		return listUnitsWrapper, nil
	}
	return nil, fmt.Errorf("No supported list Units function: %v", unitMap)

}

func parseXMLAndReturnMethods(str string) (map[string]bool, error) {

	type Method struct {
		Name string `xml:"name,attr"`
	}

	type Iface struct {
		Name   string   `xml:"name,attr"`
		Method []Method `xml:"method"`
	}

	type IntrospectData struct {
		XMLName   xml.Name `xml:"node"`
		Interface []Iface  `xml:"interface"`
	}

	methods := IntrospectData{}

	err := xml.Unmarshal([]byte(str), &methods)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshalling XML")
	}

	if len(methods.Interface) == 0 {
		return nil, errors.Wrap(err, "no methods found on introspect")
	}
	methodMap := make(map[string]bool)
	for _, iface := range methods.Interface {
		for _, method := range iface.Method {
			if strings.Contains(method.Name, "ListUnits") {
				methodMap[method.Name] = true
			}
		}
	}

	return methodMap, nil
}

// listUnitsByPatternWrapper is a bare wrapper for the unitFetcher type
func listUnitsByPatternWrapper(conn *dbus.Conn, states, patterns []string) ([]dbus.UnitStatus, error) {
	return conn.ListUnitsByPatterns(states, patterns)
}

//listUnitsFilteredWrapper wraps the dbus ListUnitsFiltered method
func listUnitsFilteredWrapper(conn *dbus.Conn, states, patterns []string) ([]dbus.UnitStatus, error) {
	units, err := conn.ListUnitsFiltered(states)
	if err != nil {
		return nil, errors.Wrap(err, "ListUnitsFiltered error")
	}

	return matchUnitPatterns(patterns, units)
}

// listUnitsWrapper wraps the dbus ListUnits method
func listUnitsWrapper(conn *dbus.Conn, states, patterns []string) ([]dbus.UnitStatus, error) {
	units, err := conn.ListUnits()
	if err != nil {
		return nil, errors.Wrap(err, "ListUnits error")
	}
	if len(patterns) > 0 {
		units, err = matchUnitPatterns(patterns, units)
		if err != nil {
			return nil, errors.Wrap(err, "error matching unit patterns")
		}
	}

	if len(states) > 0 {
		var finalUnits []dbus.UnitStatus
		for _, unit := range units {
			for _, state := range states {
				if unit.LoadState == state || unit.ActiveState == state || unit.SubState == state {
					finalUnits = append(finalUnits, unit)
					break
				}
			}
		}
		return finalUnits, nil
	}

	return units, nil
}

// matchUnitPatterns returns a list of units that match the pattern list.
// This algo, including filepath.Match, is designed to (somewhat) emulate the behavior of ListUnitsByPatterns, which uses `fnmatch`.
func matchUnitPatterns(patterns []string, units []dbus.UnitStatus) ([]dbus.UnitStatus, error) {
	var matchUnits []dbus.UnitStatus
	for _, unit := range units {
		for _, pattern := range patterns {
			match, err := filepath.Match(pattern, unit.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "error matching with pattern %s", pattern)
			}
			if match {
				matchUnits = append(matchUnits, unit)
				break
			}
		}
	}
	return matchUnits, nil
}
