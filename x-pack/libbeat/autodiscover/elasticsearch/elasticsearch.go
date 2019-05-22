// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/libbeat/autodiscover/template"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func init() {
	autodiscover.Registry.AddProvider("elasticsearch", AutodiscoverBuilder)
}

type Provider struct {
	Watcher *ESWatcher
	bus     bus.Bus
}

func (p *Provider) String() string {
	return "ES Provider"
}

func (p *Provider) Start() {
	p.Watcher.Start()
}

func (p *Provider) Stop() {
	p.Watcher.Stop()
}

func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {

	fmt.Printf("DEFAULT\n")
	config := defaultConfig()
	fmt.Printf("UNPACKING\n")
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.HintsEnabled)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	watcher := ESWatcher{
		Query:     config.Query,
		Fields:    config.Fields,
		Index:     config.Index,
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		uuid:      uuid,
	}

	return &Provider{
		Watcher: &watcher,
		bus:     bus,
	}, nil
}

type ESWatcher struct {
	Query     map[string]interface{}
	Fields    []string
	Index     string
	bus       bus.Bus
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	templates template.Mapper
	uuid      uuid.UUID
}

func (esw *ESWatcher) Start() {
	seen := map[uint64]bool{}
	go func() {
		for {
			url := fmt.Sprintf("http://localhost:9200/%s/_search?size=0", esw.Index)

			sources := common.MapStr{}
			for _, f := range esw.Fields {
				sources[f] = common.MapStr{"terms": common.MapStr{"field": f}}
			}

			req := common.MapStr{
				"aggregations": common.MapStr{
					"group": common.MapStr{
						"composite": common.MapStr{
							"sources": sources,
						},
						"aggregations": common.MapStr{
							"doc": common.MapStr{"top_hits": common.MapStr{"size": 1}},
						},
					},
				},
			}
			jsonReq, err := json.Marshal(req)
			if err != nil {
				logp.Err("Could not encode request body", err)
				return
			}

			fmt.Printf("QUERY IS %s\n", string(jsonReq))
			resp, err := http.Post(url, "application/json", bytes.NewReader(jsonReq))
			if err != nil {
				logp.Err("Encountered error in ES autodisco %v", err)
				return
			}

			bodyBuf := new(bytes.Buffer)
			bodyBuf.ReadFrom(resp.Body)

			result := struct {
				Aggregations struct {
					Group struct {
						Buckets []struct {
							Key map[string]interface{}
							Doc struct {
								Hits struct {
									Hits []struct {
										Source map[string]interface{} `json:"_source"`
									} `json:"hits"`
								} `json:"hits"`
							} `json:"doc"`
						} `json:"buckets"`
					} `json:"group"`
				} `json:"aggregations"`
			}{}

			bodyBytes := bodyBuf.Bytes()
			err = json.Unmarshal(bodyBytes, &result)
			if err != nil {
				logp.Err("Could not decode JSON %v\n", err)
				return
			}

			buckets := result.Aggregations.Group.Buckets
			type hashAndDoc struct {
				hash uint64
				doc  map[string]interface{}
			}
			hashedDocs := []hashAndDoc{}
			for _, bucket := range buckets {
				hash, err := hashstructure.Hash(bucket, nil)
				if err != nil {
					logp.Err("could not hash bucket %s", err)
				}
				if !seen[hash] == true {
					hashedDoc := hashAndDoc{hash, bucket.Doc.Hits.Hits[0].Source}
					hashedDocs = append(hashedDocs, hashedDoc)
					fmt.Printf("Created hashed doc %v || %v", hashedDoc.hash, bucket.Doc.Hits.Hits)
					seen[hash] = true
				} else {
					fmt.Printf("Already seen hash %v", hash)
				}
			}

			fmt.Printf("RESP to %s IS %s >>\n %v\n", url, resp.Status, hashedDocs)

			for _, hashedDoc := range hashedDocs {
				event := bus.Event{
					"provider": esw.uuid,
					"id":       hashedDoc.hash,
					"start":    true,
					"doc":      hashedDoc.doc,
				}

				config := esw.templates.GetConfig(event)
				event["config"] = config

				fields := (*config[0]).GetFields()
				fmt.Printf("APPEND FIELDS %v\n\n", fields)
				fmt.Printf("DOC IS %v\n", hashedDoc.doc)
				esw.appenders.Append(event)
				esw.bus.Publish(event)
			}

			time.Sleep(time.Second)
		}
	}()
}

func (esw *ESWatcher) Stop() {

}
