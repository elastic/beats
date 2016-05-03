package filter

type ConditionConfig struct {
	Equals   map[string]string     `config:"equals"`
	Contains map[string]string     `config:"contains"`
	Regexp   map[string]string     `config:"regexp"`
	Range    map[string]RangeValue `config:"range"`
}

type RangeValue struct {
	Gte *float64 `config:"gte"`
	Gt  *float64 `config:"gt"`
	Lte *float64 `config:"lte"`
	Lt  *float64 `config:"lt"`
}

type EqualsValue struct {
	Int int
	Str string
}

type DropFieldsConfig struct {
	Fields          []string `config:"fields"`
	ConditionConfig `config:",inline"`
}

type IncludeFieldsConfig struct {
	Fields          []string `config:"fields"`
	ConditionConfig `config:",inline"`
}

type FilterConfig struct {
	DropFields    *DropFieldsConfig    `config:"drop_fields"`
	IncludeFields *IncludeFieldsConfig `config:"include_fields"`
}

// fields that should be always exported
var MandatoryExportedFields = []string{"@timestamp", "type"}
