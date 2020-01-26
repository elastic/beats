// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package add_external_metadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/user"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	jsprocessor "github.com/elastic/beats/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("add_external_metadata", New)
	jsprocessor.RegisterPlugin("AddExternalMetadata", New)
}

type addHostMetadata struct {
	// lastUpdate struct {
	// 	time.Time
	// 	sync.Mutex
	// }
	fields    common.MapStr
	shared    bool
	overwrite bool
}

type addFields struct {
	fields common.MapStr
}

const (
	processorName = "add_external_metadata"
)

// FieldsKey is the default target key for the add_fields processor.
const FieldsKey = "fields"

// New constructs a new fetch_data processor.
func New(c *common.Config) (processors.Processor, error) {
	config := struct {
		Fields common.MapStr `config:"fields" validate:"required"`
		Target *string       `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the fetch_data configuration: %s", err)
	}
	data, err := postMessageData()
	if err != nil {
		p := &addFields{fields: common.MapStr{
			"stationCity":    "null",
			"stationCountry": "null",
		}}

		return p, nil

	}
	stationCity := data.City
	stationCountry := data.Country

	p := &addFields{fields: common.MapStr{
		"stationCity":    stationCity,
		"stationCountry": stationCountry,
	}}

	return p, nil
}

func (af *addFields) Run(event *beat.Event) (*beat.Event, error) {
	fields := af.fields
	// fields["TESTFROMFETCHDATA"] = "ben"
	// if af.shared {
	// 	fields = fields.Clone()
	// }
	event.Fields.DeepUpdate(fields)
	// } else {
	// 	event.Fields.DeepUpdateNoOverwrite(fields)
	// }
	return event, nil
}

func (af *addFields) String() string {
	s, _ := json.Marshal(af.fields)
	return fmt.Sprintf("add_fields=%s", s)
}

// api request to get json data
type nameData struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

func postMessageData() (*nameData, error) {
	url := "http://localhost:3001/api/host/"
	fmt.Println("URL:>", url)

	// getting username of enduser
	stationName := getStationName()

	var resData nameData
	var jsonStr = []byte(`{"name":"` + stationName + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("Problem With Request To Server ", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Problem with Client policy, or failure to speak to HTTP", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Could not read response Body", err)
		return nil, err
	}

	err2 := json.Unmarshal([]byte(body), &resData)
	if err != nil {
		log.Println("Problem with Json Parsing", err2)
		return nil, err
	}
	// fmt.Println(resData)
	return &resData, nil
}

// getting username of enduser
func getStationName() string {
	user, err := user.Current()
	if err != nil {
		log.Println("Could Not Get UserName Details ", err)
	}
	stationName := strings.Split(user.Username, `\`)[0]
	return stationName
}
