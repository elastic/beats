package hl7v2

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type hl7v2Config struct {
	config.ProtocolCommon  `config:",inline"`
	NewLineChars           string              `config:"newline_chars"`
	SegmentSelectionMode   string              `config:"segment_selection_mode"`
	Segments               []string            `config:"segments"`
	FieldSelectionMode     string              `config:"field_selection_mode"`
	Fields                 []string            `config:"fields"`
	FieldMappingMap        []map[string]string `config:"field_name_mapping"`
	ComponentSelectionMode string              `config:"component_selection_mode"`
	Components             []string            `config:"components"`
}

var (
	defaultConfig = hl7v2Config{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
	}
)

func (c *hl7v2Config) Validate() error {
	return nil
}
