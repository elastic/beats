package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var (
	Client HTTPClient
)

func init() {
	Client = &http.Client{}
}

type Output struct {
	Hosts    []string `config:"hosts"`
	Username string   `config:"username"`
	Password string   `config:"password"`
}

type SyntheticServicePayload struct {
	Monitors []map[string]interface{} `json:"monitors"`
	Output   Output                   `json:"output"`
}

type SyntheticService struct {
	config               config.Config
	monitorReloader      *cfgfile.Reloader
	servicePushTicker    *time.Ticker
	servicePushWait      sync.WaitGroup
	serviceRunnerFactory *MonitorRunnerFactory
}

func NewSyntheticService(c config.Config, monReload *cfgfile.Reloader, sr *MonitorRunnerFactory) *SyntheticService {
	return &SyntheticService{
		config:               c,
		monitorReloader:      monReload,
		servicePushTicker:    nil,
		servicePushWait:      sync.WaitGroup{},
		serviceRunnerFactory: sr,
	}
}

func (service *SyntheticService) Run(b *beat.Beat) error {
	logp.Info("Starting run via synthetics service. This is an experimental feature and may be changed or removed in the future!")

	validationErr := service.validateMonitorsSchedule()
	if validationErr != nil {
		return validationErr
	}

	serviceManifest, sErr := service.getSyntheticServiceManifest()
	if sErr != nil {
		return sErr
	}

	serviceLocations := serviceManifest.Locations

	pushInterval := 30 * time.Second
	service.servicePushTicker = time.NewTicker(pushInterval)

	output := Output{}
	err := b.Config.Output.Config().Unpack(&output)
	if err != nil {
		logp.Info("Unable to parse output param")
		return err
	}

	// first we need to push at start, and then ticker will take over
	for locationKey, serviceLocation := range serviceLocations {
		service.servicePushWait.Add(1)
		go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output)
	}
	go service.schedulePushConfig(serviceLocations, output)
	if service.config.ConfigMonitors.Enabled(){
		go service.scheduleReloadPushConfig(serviceLocations, output)
	}
	return nil

}

func (service *SyntheticService) Wait() {
	service.servicePushWait.Wait()
}

func (service *SyntheticService) Stop() {
	service.servicePushTicker.Stop()
}

func (service *SyntheticService) schedulePushConfig(serviceLocations map[string]config.ServiceLocation, output Output) {
	service.servicePushWait.Add(1)
	for {
		select {
		case <-service.servicePushTicker.C:
			for locationKey, serviceLocation := range serviceLocations {
				service.servicePushWait.Add(1)
				// first we need to do at start, and then ticker will take over
				go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output)
			}

			defer service.servicePushWait.Done()
		}
	}
}

func (service *SyntheticService) scheduleReloadPushConfig(serviceLocations map[string]config.ServiceLocation, output Output) {
	reloadPushTicker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-service.serviceRunnerFactory.Update:
			if reloadPushTicker == nil {
				reloadPushTicker = time.NewTicker(1 * time.Second)
			} else {
				reloadPushTicker.Reset(1 * time.Second)
			}

		case <-reloadPushTicker.C:
			for locationKey, serviceLocation := range serviceLocations {
				service.servicePushWait.Add(1)
				// first we need to do at start, and then ticker will take over
				go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output)
			}
			reloadPushTicker.Stop()
		}
	}
}

func (service *SyntheticService) getSyntheticServiceManifest() (config.ServiceManifest, error) {
	serviceCfg := service.config.Service
	var err error

	if serviceCfg.Username == "" {
		err = errors.New("synthetic service username is required for authentication")
	}

	if serviceCfg.Password == "" {
		err = errors.New("synthetic service password is required for authentication")
	}

	if serviceCfg.ManifestURL == "" {
		err = errors.New("synthetic service manifest url is required")
	}

	if err != nil {
		return config.ServiceManifest{}, err
	}

	req, err := http.NewRequest("GET", serviceCfg.ManifestURL, nil)

	resp, err := Client.Do(req)

	if err != nil {
		return config.ServiceManifest{}, err
	}

	serviceManifest := config.ServiceManifest{}

	read, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(read, &serviceManifest)

	return serviceManifest, err

}

func (service *SyntheticService) validateMonitorsSchedule() error {
	for _, m := range service.config.Monitors {
		monitorFields, _ := stdfields.ConfigToStdMonitorFields(m)
		monitorSchedule, _ := schedule.ParseSchedule(monitorFields.ScheduleStr)
		if monitorSchedule.Seconds() < 60 {
			return errors.New("schedule can't be less than 1 minute while using synthetics service")
		}
	}
	return nil
}

func (service *SyntheticService) pushConfigsToSyntheticsService(locationKey string, serviceLocation config.ServiceLocation, output Output) {
	defer service.servicePushWait.Done()

	payload := SyntheticServicePayload{Output: output}

	addToPayload := func(monCfg *common.Config) {
		monitorFields, _ := stdfields.ConfigToStdMonitorFields(monCfg)
		if locationInServiceLocation(locationKey, monitorFields.ServiceLocations) {
			target := map[string]interface{}{}
			err := monCfg.Unpack(target)
			if err != nil {
				logp.Info("error unpacking monitor plugin config")
				return
			}
			payload.Monitors = append(payload.Monitors, target)
		}
	}
	monitorsById := map[string]*common.Config{}
	for _, monCfg := range service.config.Monitors {
		monitorFields, _ := stdfields.ConfigToStdMonitorFields(monCfg)
		monitorsById[monitorFields.ID] = monCfg
	}

	if service.config.ConfigMonitors.Enabled() {
		for monId, monitor := range service.serviceRunnerFactory.GetMonitorsById() {
			monitorsById[monId] = monitor
		}
	}

	for _, monitor := range monitorsById {
		addToPayload(monitor)
	}

	if len(payload.Monitors) == 0 {
		logp.Info("No monitor found for service: %s, to push configuration.", serviceLocation.Geo.Name)
		return
	}

	serviceCfg := service.config.Service

	jsonValue, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/cronjob", serviceLocation.Url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(serviceCfg.Username, serviceCfg.Password)

	resp, err := Client.Do(req)
	if err != nil {
		logp.Info("Failed to push configurations to the synthetics service: %s for %d monitors",
			serviceLocation.Geo.Name, len(payload.Monitors))
		logp.Error(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logp.Info("Failed to push configurations to the synthetics service: %s for %d monitors",
				serviceLocation.Geo.Name, len(payload.Monitors))
			logp.Error(err)
		}
		bodyString := string(bodyBytes)

		if bodyString == "success" {
			logp.Info("Successfully pushed configurations to the synthetics service: %s for %d monitors",
				serviceLocation.Geo.Name, len(payload.Monitors))
		}
	}

}

func locationInServiceLocation(location string, locationsList []string) bool {
	for _, b := range locationsList {
		if b == location {
			return true
		}
	}
	return false
}
