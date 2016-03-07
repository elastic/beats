package filter

type DropFieldsConfig struct {
	Fields []string `config:"fields"`
}

type IncludeFieldsConfig struct {
	Fields []string `config:"fields"`
}

type FilterConfig struct {
	DropFields    *DropFieldsConfig    `config:"drop_fields"`
	IncludeFields *IncludeFieldsConfig `config:"include_fields"`
}

// fields that should be always exported
var MandatoryExportedFields = []string{"@timestamp", "type"}
