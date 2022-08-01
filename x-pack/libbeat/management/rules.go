package management

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type UnitInput struct {
	Id          string         `config:"id"`
	Type        string         `config:"type"`
	DataStream  UnitDataStream `config:"data_stream"`
	UseOutput   string         `config:"use_output"`
	Streams     []mapstr.M     `config:"streams"`
	renderedCfg []mapstr.M
}

type UnitDataStream struct {
	Namespace string `config:"namespace"`
	Dataset   string `config:"dataset"`
	Type      string `config:"type"`
}

func (unit *UnitInput) Init() {
	for i := 0; i < len(unit.Streams); i++ {
		unit.renderedCfg = append(unit.renderedCfg, make(mapstr.M))
	}
}

func (unit *UnitInput) InjectIndex(streamType string) error {
	for i := 0; i < len(unit.Streams); i++ {
		datastream, err := unit.Streams[i].GetValue("data_stream.dataset")
		if err != nil {
			return err
		}
		index := fmt.Sprintf("%s-%s-%s", streamType, datastream, unit.DataStream.Namespace)
		unit.renderedCfg[i].Put("index", index)
	}
	return nil
}

// var Metricbeat transpiler.RuleList = transpiler.RuleList{Rules: []transpiler.Rule{
// 	&transpiler.FixStreamRule{},
// 	&transpiler.InjectIndexRule{Type: "metrics"},
// 	&transpiler.InjectStreamProcessorRule{Type: "metrics", OnConflict: "insert_after"},
// 	&transpiler.RenameRule{From: "inputs", To: "inputsstreams"},
// 	&transpiler.MapRule{Path: "inputsstreams", Rules: []transpiler.Rule{
// 		&transpiler.CopyAllToListRule{To: "streams", OnConflict: "noop", Except: []string{"streams", "id", "enabled", "processors"}},
// 		&transpiler.CopyToListRule{Item: "processors", To: "streams", OnConflict: "insert_before"},
// 	}},
// 	&transpiler.ExtractListItemRule{Path: "inputsstreams", Item: "streams", To: "inputs"},
// 	&transpiler.FilterValuesWithRegexpRule{Key: "type", Re: regexp.MustCompile("^.+/metrics$"), Selector: "inputs"},
// 	&transpiler.FilterValuesRule{Selector: "inputs", Key: "enabled", Values: []interface{}{true}},
// 	&transpiler.MapRule{Path: "inputs", Rules: []transpiler.Rule{
// 		&transpiler.TranslateWithRegexpRule{Path: "type", Re: regexp.MustCompile("^(?P<type>.+)/metrics$"), With: "$type"},
// 		&transpiler.RenameRule{From: "type", To: "module"},
// 		&transpiler.MakeArrayRule{Item: "metricset", To: "metricsets"},
// 		&transpiler.RemoveKeyRule{Key: "metricset"},
// 		&transpiler.RemoveKeyRule{Key: "enabled"},
// 		&transpiler.RemoveKeyRule{Key: "data_stream"},
// 		&transpiler.RemoveKeyRule{Key: "data_stream.dataset"},
// 		&transpiler.RemoveKeyRule{Key: "data_stream.namespace"},
// 		&transpiler.RemoveKeyRule{Key: "use_output"},
// 	}},
// 	//&transpiler.InjectAgentInfoRule{},
// 	&transpiler.CopyRule{From: "inputs", To: "metricbeat"},
// 	&transpiler.RenameRule{From: "metricbeat.inputs", To: "modules"},
// 	&transpiler.FilterRule{Selectors: []transpiler.Selector{"metricbeat", "output", "keystore"}},
// 	//&transpiler.InjectHeadersRule{},
// }}
