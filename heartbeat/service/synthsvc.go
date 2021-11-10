package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
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
	Client = http.DefaultClient
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

type SyntheticsService struct {
	config               config.Config
	servicePushTicker    *time.Ticker
	servicePushWait      sync.WaitGroup
	ServiceRunnerFactory *MonitorRunnerFactory
}

func NewSyntheticService(c config.Config) *SyntheticsService {
	return &SyntheticsService{
		config:               c,
		servicePushTicker:    nil,
		servicePushWait:      sync.WaitGroup{},
		ServiceRunnerFactory: NewRunnerFactory(),
	}
}

func (service *SyntheticsService) Run(b *beat.Beat) error {
	logp.Info("Starting run via synthetics service. This is an experimental feature and may be changed or removed in the future!")

	vErr := service.validateMonitorsSchedule()
	if vErr != nil {
		return fmt.Errorf("error validating monitor schedule: %w", vErr)
	}

	svManifest, sErr := service.getSyntheticServiceManifest()
	if sErr != nil {
		return fmt.Errorf("failed to fetch service manifest file with err %w", sErr)
	}

	pushInterval := 30 * time.Second
	service.servicePushTicker = time.NewTicker(pushInterval)

	output := Output{}
	err := b.Config.Output.Config().Unpack(&output)
	if err != nil {
		return fmt.Errorf("unable to parse output param %w", err)
	}

	serviceLocations := svManifest.Locations
	// first we need to push at start, and then ticker will take over
	for locationKey, serviceLocation := range serviceLocations {
		service.servicePushWait.Add(1)
		go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output, time.Millisecond)
	}
	go service.schedulePushConfig(serviceLocations, output)
	if service.config.ConfigMonitors.Enabled() {
		go service.scheduleReloadPushConfig(serviceLocations, output)
	}
	return nil
}

func (service *SyntheticsService) Wait() {
	service.servicePushWait.Wait()
}

func (service *SyntheticsService) Stop() {
	service.servicePushTicker.Stop()
}

func (service *SyntheticsService) schedulePushConfig(serviceLocations map[string]config.ServiceLocation, output Output) {
	service.servicePushWait.Add(1)
	for {
		select {
		case <-service.servicePushTicker.C:
			for locationKey, serviceLocation := range serviceLocations {
				service.servicePushWait.Add(1)
				go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output, 0)
			}
		}
	}
}

func (service *SyntheticsService) scheduleReloadPushConfig(serviceLocations map[string]config.ServiceLocation, output Output) {
	reloadPushTicker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-service.ServiceRunnerFactory.Update:
			if reloadPushTicker == nil {
				reloadPushTicker = time.NewTicker(1 * time.Second)
			} else {
				reloadPushTicker.Reset(1 * time.Second)
			}

		case <-reloadPushTicker.C:
			for locationKey, serviceLocation := range serviceLocations {
				service.servicePushWait.Add(1)
				// first we need to do at start, and then ticker will take over
				go service.pushConfigsToSyntheticsService(locationKey, serviceLocation, output, 0)
			}
			reloadPushTicker.Stop()
		}
	}
}

func (service *SyntheticsService) getSyntheticServiceManifest() (config.ServiceManifest, error) {
	logp.Info("fetching manifest file to get service locations")
	var err error
	var manifest config.ServiceManifest

	serviceCfg := service.config.Service
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

	fetchManifestFile := func() (config.ServiceManifest, error) {
		req, err := http.NewRequest("GET", serviceCfg.ManifestURL, nil)
		resp, err := Client.Do(req)

		if err != nil {
			logp.Warn("failed to fetch manifest file with error: %s", err)
			return config.ServiceManifest{}, err
		}

		serviceManifest := config.ServiceManifest{}
		read, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			logp.Warn("failed to fetch manifest file with error: %s", err)
			return config.ServiceManifest{}, err
		}

		err = json.Unmarshal(read, &serviceManifest)
		return serviceManifest, err
	}

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			waitAfter := time.Second * time.Duration(attempt*2)
			logp.Info("retrying fetching manifest file after %.0f seconds", waitAfter.Seconds())
			time.Sleep(waitAfter)
		}
		manifest, err = fetchManifestFile()
		if err == nil {
			return manifest, nil
		}
	}

	return config.ServiceManifest{}, err
}

func (service *SyntheticsService) validateMonitorsSchedule() error {
	for _, m := range service.config.Monitors {
		monitorFields, err := stdfields.ConfigToStdMonitorFields(m)
		if err !=nil{
			logp.Error(err)
			return err
		}
		sched := monitorFields.Schedule
		if err !=nil{
			logp.Error(err)
			return err
		}
		next := sched.Next(time.Now())
		nextNext := sched.Next(next)
		duration := nextNext.Sub(next)
		if duration < time.Second*60 {
			return errors.New("schedule can't be less than 1 minute while using synthetics service")
		}
	}
	return nil
}

func (service *SyntheticsService) pushConfigsToSyntheticsService(locationKey string, serviceLocation config.ServiceLocation, output Output, retryInterval time.Duration) {
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
		for monId, monitor := range service.ServiceRunnerFactory.GetMonitorsById() {
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

	resp, err := postConfig(url, jsonValue, serviceCfg, retryInterval)
	if err != nil || resp == nil {
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

func postConfig(url string, jsonValue []byte, serviceCfg config.ServiceConfig, retryInterval time.Duration) (*http.Response, error) {
	var err error
	var resp *http.Response
	var retryTicker *time.Ticker

	executePush := func() (*http.Response, error) {
		req, errReq := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
		if errReq != nil {
			return nil, errReq
		}
		req.Header.Set("Content-Type", "application/json")

		req.SetBasicAuth(serviceCfg.Username, serviceCfg.Password)
		resp, err = Client.Do(req)

		if resp != nil && resp.StatusCode == http.StatusOK {
			return resp, err
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, errors.New("service not found at provided url")
		}
		return nil, err
	}

	resp, err =  executePush()
	if resp !=nil {
		return resp, err
	}

	retryAttempts := 1
	if retryInterval != 0{
		retryTicker = time.NewTicker(retryInterval)
	}else {
		retryTicker = time.NewTicker(time.Second * time.Duration(retryAttempts*10))
	}
	for  {
		select {
		case <- retryTicker.C:
			waitAfter := time.Second * time.Duration(retryAttempts*10)
			logp.Warn("failed pushing configs with error: %s", err)
			logp.Info("retrying pushing configuration after %.0f seconds", waitAfter.Seconds())
			resp, err =  executePush()
			if resp !=nil {
				return resp, err
			}
			retryAttempts++
			if retryAttempts == 3 {
				retryTicker.Stop()
				return resp, err
			}
			if retryInterval != 0{
				retryTicker.Reset(retryInterval)
			}else {
				retryTicker.Reset(waitAfter)
			}
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
