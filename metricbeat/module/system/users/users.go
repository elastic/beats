package users

import (
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "users", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	totalUsersProfiles int
	totalLoggedUsers   int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system users metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:      base,
		totalUsersProfiles: 0,
		totalLoggedUsers:   0,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// get total users that have account.
	totalUsersProfiles := getTotalUsersNumber()
	// get total currently logged users
	totalLoggedUsers := getUsersLoggedIn()
	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"totalUsersProfiles": totalUsersProfiles,
			"totalLoggedUsers":   totalLoggedUsers,
		},
	})
	return nil
}

func getTotalUsersNumber() int {
	// set number of folder to remove from count(default,public,user)
	foldersToRemove := 3
	// get user folder
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	// get parent folder
	parent := filepath.Dir(usr.HomeDir)

	var usersDir []string
	files, err := ioutil.ReadDir(parent)
	if err != nil {
		log.Fatal(err, "could not read parent  folder")
	}

	for _, file := range files {
		if file.IsDir() {
			usersDir = append(usersDir, file.Name())
		}
	}
	return len(usersDir) - foldersToRemove
}

type key struct {
	Name      string
	ProcessID int
}

func getUsersLoggedIn() int {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()
	unknown, _ := oleutil.CreateObject("WbemScripting.SWbemLocator")
	defer unknown.Release()
	wmi, _ := unknown.QueryInterface(ole.IID_IDispatch)
	defer wmi.Release()
	serviceRaw, _ := oleutil.CallMethod(wmi, "ConnectServer")
	service := serviceRaw.ToIDispatch()
	defer service.Release()
	resultRaw, _ := oleutil.CallMethod(service, "ExecQuery", "SELECT * FROM Win32_Process WHERE Name = 'explorer.exe'")
	result := resultRaw.ToIDispatch()
	defer result.Release()
	countVar, _ := oleutil.GetProperty(result, "Count")
	count := int(countVar.Val)
	return count
}
