package status

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
        "fmt"
	"encoding/json"
)

const (
        measurement = "ceph"
        typeMon     = "monitor"
        typeOsd     = "osd"
        osdPrefix   = "ceph-osd"
        monPrefix   = "ceph-mon"
        sockSuffix  = "asok"
)

func eventsMapping(input string) []common.MapStr {


	myEvents := []common.MapStr{}

	data := make(map[string]interface{}) 
        err := json.Unmarshal([]byte(input), &data)
        if err != nil {
                logp.Err("An error occurred while parsing data for getting ceph status: %v", err)
        }

        eventsHealthmap,errHealthMap := decodeStatusHealth(data)
        if errHealthMap != nil {
                logp.Err("An error occurred while parsing data for getting ceph status health: %v", errHealthMap)
        }

	eventsOsdmap,errOsdMap := decodeStatusOsdmap(data)
	if errOsdMap != nil {
		logp.Err("An error occurred while parsing data for getting ceph status osd: %v", errOsdMap)
	}

//        eventsPgmap,errPgMap := decodeStatusPgmap(data)
//        if errPgMap != nil {
//                logp.Err("An error occurred while parsing data for getting ceph status pg: %v", errPgMap)
//        }

//	myEvents = append(myEvents,eventsPgmap...)

//        eventsPgmapState,errPgmapState := decodeStatusPgmapState(data)
//        if errPgmapState != nil {
//                logp.Err("An error occurred while parsing data for getting ceph status pg state: %v", errPgmapState)
//        }

//	myEvents = append(myEvents,eventsPgmapState...)

	myEvents = append(myEvents,eventsHealthmap,eventsOsdmap)
	return myEvents
}

func decodeStatusHealth(data map[string]interface{}) (common.MapStr,error) {

        healthmap, ok := data["health"].(map[string]interface{})
        if !ok {
                return nil,fmt.Errorf("WARNING - unable to decode health")
        }

        newevent := common.MapStr{
				"overall_status": healthmap["overall_status"].(string),
			}

	//health
       	health, ok := healthmap["health"].(map[string]interface{})
        if !ok {
                return nil,fmt.Errorf("WARNING - unable to decode health health")
        }
	
        err := decodeHealthServices(health,&newevent)
        if err != nil {
               return nil,err
        }

	//timechecks
	timechecks, ok := healthmap["timechecks"].(map[string]interface{})
        if !ok {
                return nil,fmt.Errorf("WARNING - unable to decode health health")
        }

        for tag, datapoints := range timechecks {
                event := common.MapStr{
                    "health.timechecks."+tag: datapoints,
                }
                newevent = common.MapStrUnion(newevent,event)
        }
	
	//detail


        return newevent,nil

}

func decodeHealthServices(health map[string]interface{},newevent *common.MapStr) (error) {
        for key, value := range health {
                switch value.(type) {
                case []interface{}:
                        if key == "health_services" {
                                for _, hs := range value.([]interface{}) {
                                        healthservicesmap, ok := hs.(map[string]interface{})
                                        if !ok {
                                                return fmt.Errorf("WARNING - unable to decode health health health_services")
                                        }

                                        err := decodeHealthServicesMons(healthservicesmap,newevent)
                                        if err != nil {
                                                return err
                                        }
                                }
                        }
                }
        }
	
	return nil

}

func decodeHealthServicesMons(healthservicesmap map[string]interface{},newevent *common.MapStr) (error) {
        for key, value := range healthservicesmap {
                switch value.(type) {
                case []interface{}: 
                        if key == "mons" {
                                for _, mons := range value.([]interface{}) {
                                        mons_map, ok := mons.(map[string]interface{})
                                        if !ok {
                                                return fmt.Errorf("WARNING - unable to decode health health health_services mon")
                                        }       
                                        
                                        for tag, datapoints := range mons_map{
                                                event := common.MapStr{
                                                        "health.mons."+tag: datapoints,
                                                }       
                                                *newevent = common.MapStrUnion(*newevent,event)
                                        }
                                }
                        }
                }
        }
        return nil
}


func decodeStatusOsdmap(data map[string]interface{}) (common.MapStr,error) {
	osdmap, ok := data["osdmap"].(map[string]interface{})
	if !ok {
		return nil,fmt.Errorf("WARNING %s - unable to decode osdmap", measurement)
	}
	fields, ok := osdmap["osdmap"].(map[string]interface{})
	if !ok {
		return nil,fmt.Errorf("WARNING %s - unable to decode osdmap", measurement)
	}

	newevent := common.MapStr{}

	for tag, datapoints := range fields {
              	event := common.MapStr{
                    "osdmap."+tag: datapoints,
              	}
		newevent = common.MapStrUnion(newevent,event)
        }

	return newevent,nil

}


func decodeStatusPgmap(data map[string]interface{}) ([]common.MapStr,error) {
	pgmap, ok := data["pgmap"].(map[string]interface{})
	if !ok {
		return nil,fmt.Errorf("WARNING %s - unable to decode pgmap", measurement)
	}
	fields := make(map[string]interface{})
	
	myEvents := []common.MapStr{}

	for key, value := range pgmap {
		switch value.(type) {
		case float64:
			event := common.MapStr{
        	            fields[key].(string) : value,
	                }

			myEvents = append(myEvents, event)
		}
	}

	return myEvents,nil
}

func decodeStatusPgmapState(data map[string]interface{}) ([]common.MapStr,error) {
	pgmap, ok := data["pgmap"].(map[string]interface{})
	if !ok {
		return nil,fmt.Errorf("WARNING %s - unable to decode pgmap", measurement)
	}
	fields := make(map[string]interface{})

	myEvents := []common.MapStr{}

	for key, value := range pgmap {
		switch value.(type) {
		case []interface{}:
			if key != "pgs_by_state" {
				continue
			}
			for _, state := range value.([]interface{}) {
				state_map, ok := state.(map[string]interface{})
				if !ok {
					return nil,fmt.Errorf("WARNING %s - unable to decode pg state", measurement)
				}
				state_name, ok := state_map["state_name"].(string)
				if !ok {
					return nil,fmt.Errorf("WARNING %s - unable to decode pg state name", measurement)
				}
				state_count, ok := state_map["count"].(float64)
				if !ok {
					return nil,fmt.Errorf("WARNING %s - unable to decode pg state count", measurement)
				}

				 event := common.MapStr{
						fields[state_name].(string) : state_count,
                        		}

				myEvents = append(myEvents, event)
			}
		}
	}
	return myEvents,nil
}

