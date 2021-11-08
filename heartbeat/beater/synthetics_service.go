package beater

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"io/ioutil"
	"net/http"
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

type ServicePayload struct {
	Monitors []map[string]interface{} `json:"monitors"`
	Output   Output                   `json:"output"`
}

func (bt *Heartbeat) runViaSyntheticsService(b *beat.Beat) error {
	logp.Info("Starting run via synthetics service. This is an experimental feature and may be changed or removed in the future!")

	validationErr := bt.validateMonitorsSchedule()
	if validationErr != nil {
		return validationErr
	}

	serviceManifest, sErr := bt.getSyntheticServiceManifest()
	if sErr != nil {
		return sErr
	}

	serviceLocations := serviceManifest.Locations

	pushInterval := 20 * time.Minute
	bt.servicePushTicker = time.NewTicker(pushInterval)

	output := Output{}
	err := b.Config.Output.Config().Unpack(&output)
	if err != nil {
		logp.Info("Unable to parse output param")
		return err
	}

	// first we need to push at start, and then ticker will take over
	for locationKey, serviceLocation := range serviceLocations {
		bt.servicePushWait.Add(1)
		go bt.pushConfigsToSyntheticsService(locationKey, serviceLocation, output)
	}
	go bt.schedulePushConfig(serviceLocations, output)
	return nil

}

func (bt *Heartbeat) schedulePushConfig(serviceLocations map[string]config.ServiceLocation, output Output) {
	bt.servicePushWait.Add(1)
	for {
		select {
		case <-bt.servicePushTicker.C:
			for locationKey, serviceLocation := range serviceLocations {
				bt.servicePushWait.Add(1)
				// first we need to do at start, and then ticker will take over
				go bt.pushConfigsToSyntheticsService(locationKey, serviceLocation, output)
			}

			defer bt.servicePushWait.Done()
		case <-bt.done:
			bt.servicePushTicker.Stop()
		}
	}
}

func (bt *Heartbeat) getSyntheticServiceManifest() (config.ServiceManifest, error) {
	service := bt.config.Service
	var err error

	if service.Username == "" {
		err = errors.New("synthetic service username is required for authentication")
	}

	if service.Password == "" {
		err = errors.New("synthetic service password is required for authentication")
	}

	if service.ManifestURL == "" {
		err = errors.New("synthetic service manifest url is required")
	}

	if err != nil {
		return config.ServiceManifest{}, err
	}

	req, err := http.NewRequest("GET", service.ManifestURL, nil)

	resp, err := Client.Do(req)

	if err != nil {
		return config.ServiceManifest{}, err
	}

	serviceManifest := config.ServiceManifest{}



	read, err := ioutil.ReadAll(resp.Body)


	err = json.Unmarshal(read, &serviceManifest)

	return serviceManifest, err

}

func (bt *Heartbeat) validateMonitorsSchedule() error {
	for _, m := range bt.config.Monitors {
		monitorFields, _ := stdfields.ConfigToStdMonitorFields(m)
		monitorSchedule, _ := schedule.ParseSchedule(monitorFields.ScheduleStr)
		if monitorSchedule.Seconds() < 60 {
			return errors.New("schedule can't be less than 1 minute while using synthetics service")
		}
	}
	return nil
}

func (bt *Heartbeat) pushConfigsToSyntheticsService(locationKey string, serviceLocation config.ServiceLocation, output Output) {
	defer bt.servicePushWait.Done()

	payload := ServicePayload{Output: output}

	for _, monitor := range bt.config.Monitors {
		monitorFields, _ := stdfields.ConfigToStdMonitorFields(monitor)
		if locationInServiceLocation(locationKey, monitorFields.ServiceLocations) {
			target := map[string]interface{}{}
			err := monitor.Unpack(target)
			if err != nil {
				logp.Info("error unpacking monitor plugin config")
				return
			}
			payload.Monitors = append(payload.Monitors, target)
		}
	}

	if len(payload.Monitors) == 0 {
		logp.Info("No monitor found for service: %s, to push configuration.", serviceLocation.Geo.Name)
		return
	}

	service := bt.config.Service

	jsonValue, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://%s/cronjob", serviceLocation.Url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")

	req.SetBasicAuth(service.Username, service.Password)

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
