package users

import (
	"fmt"
	"os"
	"strconv"

	"github.com/godbus/dbus"
	"github.com/pkg/errors"
)

// sessionInfo contains useful properties for a session
type sessionInfo struct {
	Remote     bool
	RemoteHost string
	Name       string
	Scope      string
	Service    string
	State      string
	Type       string
	Leader     uint32
}

// loginSession contains basic information on a login session
type loginSession struct {
	ID   uint64
	UID  uint32
	User string
	Seat string
	Path dbus.ObjectPath
}

// initDbusConnection initializes a connection to the dbus
func initDbusConnection() (*dbus.Conn, error) {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return nil, errors.Wrap(err, "error getting connection to system bus")
	}

	auth := dbus.AuthExternal(strconv.Itoa(os.Getuid()))

	err = conn.Auth([]dbus.Auth{auth})
	if err != nil {
		return nil, errors.Wrap(err, "error authenticating")
	}

	err = conn.Hello()
	if err != nil {
		return nil, errors.Wrap(err, "error in Hello")
	}

	return conn, nil
}

// getSessionProps returns info on a given session pointed to by path
func getSessionProps(conn *dbus.Conn, path dbus.ObjectPath) (sessionInfo, error) {
	busObj := conn.Object("org.freedesktop.login1", path)

	var props map[string]dbus.Variant

	err := busObj.Call("org.freedesktop.DBus.Properties.GetAll", 0, "").Store(&props)
	if err != nil {
		return sessionInfo{}, errors.Wrap(err, "error calling DBus")
	}

	if len(props) < 8 {
		return sessionInfo{}, fmt.Errorf("wrong number of fields in  info: %v", props)
	}

	remote, ok := props["Remote"].Value().(bool)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast remote to bool")
	}

	remoteHost, ok := props["RemoteHost"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast remote to string")
	}

	userName, ok := props["Name"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast username to string")
	}

	scope, ok := props["Scope"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast scope to string")
	}

	service, ok := props["Service"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast service to string")
	}

	state, ok := props["State"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast state to string")
	}

	sessionType, ok := props["Type"].Value().(string)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast type to string")
	}

	leader, ok := props["Leader"].Value().(uint32)
	if !ok {
		return sessionInfo{}, fmt.Errorf("failed to cast type to uint32")
	}

	session := sessionInfo{
		Remote:     remote,
		RemoteHost: remoteHost,
		Name:       userName,
		Scope:      scope,
		Service:    service,
		State:      state,
		Type:       sessionType,
		Leader:     leader,
	}

	return session, nil
}

// listSessions lists all sessions known to dbus
func listSessions(conn *dbus.Conn) ([]loginSession, error) {
	busObj := conn.Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1"))
	var props [][]dbus.Variant

	if err := busObj.Call("org.freedesktop.login1.Manager.ListSessions", 0).Store(&props); err != nil {
		return nil, errors.Wrap(err, "error calling dbus")
	}

	sessionList := make([]loginSession, len(props))
	for iter, session := range props {
		if len(session) < 5 {
			return nil, fmt.Errorf("wrong number of fields in session: %v", session)
		}
		idStr, ok := session[0].Value().(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast user ID to string")
		}

		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing ID to int")
		}

		uid, ok := session[1].Value().(uint32)
		if !ok {
			return nil, fmt.Errorf("failed to cast session uid to uint32")
		}
		user, ok := session[2].Value().(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast session user to string")
		}
		seat, ok := session[3].Value().(string)
		if !ok {
			return nil, fmt.Errorf("failed to cast session seat to string")
		}
		path, ok := session[4].Value().(dbus.ObjectPath)
		if !ok {
			return nil, fmt.Errorf("failed to cast session path to ObjectPath")
		}
		newSession := loginSession{ID: id,
			UID:  uid,
			User: user,
			Seat: seat,
			Path: path}
		sessionList[iter] = newSession
	}

	return sessionList, nil
}
