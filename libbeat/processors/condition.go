package processors

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/match"
	"github.com/elastic/beats/libbeat/logp"
)

type RangeValue struct {
	gte *float64
	gt  *float64
	lte *float64
	lt  *float64
}

type EqualsValue struct {
	Int  uint64
	Str  string
	Bool bool
}

type Condition struct {
	equals  map[string]EqualsValue
	matches struct {
		name    string
		filters map[string]match.Matcher
	}
	hasfields []string
	rangexp   map[string]RangeValue
	or        []Condition
	and       []Condition
	not       *Condition
}

type WhenProcessor struct {
	condition *Condition
	p         Processor
}

// ValuesMap provides a common interface to read fields for condition checking
type ValuesMap interface {
	// GetValue returns the given field from the map
	GetValue(string) (interface{}, error)
}

func NewConditional(
	ruleFactory Constructor,
) Constructor {
	return func(cfg *common.Config) (Processor, error) {
		rule, err := ruleFactory(cfg)
		if err != nil {
			return nil, err
		}

		return addCondition(cfg, rule)
	}
}

func NewCondition(config *ConditionConfig) (*Condition, error) {
	c := Condition{}

	if config == nil {
		// empty condition
		return nil, nil
	}

	var err error
	switch {
	case config.Equals != nil:
		err = c.setEquals(config.Equals)
	case config.Contains != nil:
		c.matches.name = "contains"
		c.matches.filters, err = compileMatches(config.Contains.fields, match.CompileString)
	case config.Regexp != nil:
		c.matches.name = "regexp"
		c.matches.filters, err = compileMatches(config.Regexp.fields, match.Compile)
	case config.Range != nil:
		err = c.setRange(config.Range)
	case config.HasFields != nil:
		c.hasfields = config.HasFields
	case len(config.OR) > 0:
		c.or, err = NewConditionList(config.OR)
	case len(config.AND) > 0:
		c.and, err = NewConditionList(config.AND)
	case config.NOT != nil:
		c.not, err = NewCondition(config.NOT)
	default:
		err = errors.New("missing condition")
	}
	if err != nil {
		return nil, err
	}

	logp.Debug("processors", "New condition %s", c)
	return &c, nil
}

func NewConditionList(config []ConditionConfig) ([]Condition, error) {
	out := make([]Condition, len(config))
	for i, condConfig := range config {
		cond, err := NewCondition(&condConfig)
		if err != nil {
			return nil, err
		}

		out[i] = *cond
	}
	return out, nil
}

func (c *Condition) setEquals(cfg *ConditionFields) error {
	c.equals = map[string]EqualsValue{}

	for field, value := range cfg.fields {
		uintValue, err := extractInt(value)
		if err == nil {
			c.equals[field] = EqualsValue{Int: uintValue}
			continue
		}

		sValue, err := extractString(value)
		if err == nil {
			c.equals[field] = EqualsValue{Str: sValue}
			continue
		}

		bValue, err := extractBool(value)
		if err == nil {
			c.equals[field] = EqualsValue{Bool: bValue}
			continue
		}

		return fmt.Errorf("unexpected type %T in equals condition", value)
	}

	return nil
}

func compileMatches(
	fields map[string]interface{},
	compile func(string) (match.Matcher, error),
) (map[string]match.Matcher, error) {
	if len(fields) == 0 {
		return nil, nil
	}

	out := map[string]match.Matcher{}
	for field, value := range fields {
		var err error

		switch v := value.(type) {
		case string:
			out[field], err = compile(v)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("unexpected type %T of %v", value, value)
		}
	}
	return out, nil
}

func (c *Condition) setRange(cfg *ConditionFields) error {
	c.rangexp = map[string]RangeValue{}

	updateRangeValue := func(key string, op string, value float64) error {

		field := strings.TrimSuffix(key, "."+op)
		_, exists := c.rangexp[field]
		if !exists {
			c.rangexp[field] = RangeValue{}
		}
		rv := c.rangexp[field]
		switch op {
		case "gte":
			rv.gte = &value
		case "gt":
			rv.gt = &value
		case "lt":
			rv.lt = &value
		case "lte":
			rv.lte = &value
		default:
			return fmt.Errorf("unexpected field %s", op)
		}
		c.rangexp[field] = rv
		return nil
	}

	for key, value := range cfg.fields {

		floatValue, err := extractFloat(value)
		if err != nil {
			return err
		}

		list := strings.Split(key, ".")
		err = updateRangeValue(key, list[len(list)-1], floatValue)
		if err != nil {
			return err
		}

	}

	return nil
}

func (c *Condition) Check(event ValuesMap) bool {
	if len(c.or) > 0 {
		return c.checkOR(event)
	}

	if len(c.and) > 0 {
		return c.checkAND(event)
	}

	if c.not != nil {
		return c.checkNOT(event)
	}

	return c.checkEquals(event) &&
		c.checkMatches(event) &&
		c.checkRange(event) &&
		c.checkHasFields(event)
}

func (c *Condition) checkOR(event ValuesMap) bool {
	for _, cond := range c.or {
		if cond.Check(event) {
			return true
		}
	}
	return false
}

func (c *Condition) checkAND(event ValuesMap) bool {
	for _, cond := range c.and {
		if !cond.Check(event) {
			return false
		}
	}
	return true
}

func (c *Condition) checkNOT(event ValuesMap) bool {
	if c.not.Check(event) {
		return false
	}
	return true
}

func (c *Condition) checkEquals(event ValuesMap) bool {
	for field, equalValue := range c.equals {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		intValue, err := extractInt(value)
		if err == nil {
			if intValue != equalValue.Int {
				return false
			}

			continue
		}

		sValue, err := extractString(value)
		if err == nil {
			if sValue != equalValue.Str {
				return false
			}

			continue
		}

		bValue, err := extractBool(value)
		if err == nil {
			if bValue != equalValue.Bool {
				return false
			}

			continue
		}

		logp.Err("unexpected type %T in equals condition as it accepts only integers, strings or bools. ", value)
		return false
	}

	return true
}

func (c *Condition) checkMatches(event ValuesMap) bool {
	matchers := c.matches.filters
	if matchers == nil {
		return true
	}

	for field, matcher := range matchers {
		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		switch v := value.(type) {
		case string:
			if !matcher.MatchString(v) {
				return false
			}

		case []string:
			if !matcher.MatchAnyString(v) {
				return false
			}

		default:
			str, err := extractString(value)
			if err != nil {
				logp.Warn("unexpected type %T in %v condition as it accepts only strings.", value, c.matches.name)
				return false
			}

			if !matcher.MatchString(str) {
				return false
			}
		}
	}

	return true
}

func (c *Condition) checkRange(event ValuesMap) bool {
	checkValue := func(value float64, rangeValue RangeValue) bool {

		if rangeValue.gte != nil {
			if value < *rangeValue.gte {
				return false
			}
		}
		if rangeValue.gt != nil {
			if value <= *rangeValue.gt {
				return false
			}
		}
		if rangeValue.lte != nil {
			if value > *rangeValue.lte {
				return false
			}
		}
		if rangeValue.lt != nil {
			if value >= *rangeValue.lt {
				return false
			}
		}
		return true
	}

	for field, rangeValue := range c.rangexp {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		switch value.(type) {
		case int, int8, int16, int32, int64:
			intValue := reflect.ValueOf(value).Int()

			if !checkValue(float64(intValue), rangeValue) {
				return false
			}

		case uint, uint8, uint16, uint32, uint64:
			uintValue := reflect.ValueOf(value).Uint()

			if !checkValue(float64(uintValue), rangeValue) {
				return false
			}

		case float64, float32, common.Float:
			floatValue := reflect.ValueOf(value).Float()

			if !checkValue(floatValue, rangeValue) {
				return false
			}

		default:
			logp.Warn("unexpected type %T in range condition. ", value)
			return false
		}

	}
	return true
}

func (c *Condition) checkHasFields(event ValuesMap) bool {
	for _, field := range c.hasfields {
		_, err := event.GetValue(field)
		if err != nil {
			return false
		}
	}
	return true
}

func (c Condition) String() string {
	s := ""

	if len(c.equals) > 0 {
		s = s + fmt.Sprintf("equals: %v", c.equals)
	}
	if len(c.matches.filters) > 0 {
		s = s + fmt.Sprintf("%v: %v", c.matches.name, c.matches.filters)
	}
	if len(c.rangexp) > 0 {
		s = s + fmt.Sprintf("range: %v", c.rangexp)
	}
	if len(c.hasfields) > 0 {
		s = s + fmt.Sprintf("has_fields: %v", c.hasfields)
	}
	if len(c.or) > 0 {
		for _, cond := range c.or {
			s = s + cond.String() + " or "
		}
		s = s[:len(s)-len(" or ")] //delete the last or
	}
	if len(c.and) > 0 {
		for _, cond := range c.and {
			s = s + cond.String() + " and "
		}
		s = s[:len(s)-len(" and ")] //delete the last and
	}
	if c.not != nil {
		s = s + "not " + c.not.String()
	}

	return s
}

func (r RangeValue) String() string {
	s := ""
	if r.gte != nil {
		s = s + fmt.Sprintf(">= %v", *r.gte)
	}

	if r.gt != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("> %v", *r.gt)
	}

	if r.lte != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("<= %v", *r.lte)
	}
	if r.lt != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("< %v", *r.lt)
	}
	return s
}

func (e EqualsValue) String() string {
	if len(e.Str) > 0 {
		return e.Str
	}
	return strconv.Itoa(int(e.Int))
}

func NewConditionRule(
	config ConditionConfig,
	p Processor,
) (Processor, error) {
	cond, err := NewCondition(&config)
	if err != nil {
		logp.Err("Failed to initialize lookup condition: %v", err)
		return nil, err
	}

	if cond == nil {
		return p, nil
	}
	return &WhenProcessor{cond, p}, nil
}

func (r *WhenProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if !r.condition.Check(event) {
		return event, nil
	}
	return r.p.Run(event)
}

func (r *WhenProcessor) String() string {
	return fmt.Sprintf("%v, condition=%v", r.p.String(), r.condition.String())
}

func addCondition(
	cfg *common.Config,
	p Processor,
) (Processor, error) {
	if !cfg.HasField("when") {
		return p, nil
	}
	sub, err := cfg.Child("when", -1)
	if err != nil {
		return nil, err
	}

	condConfig := ConditionConfig{}
	if err := sub.Unpack(&condConfig); err != nil {
		return nil, err
	}

	return NewConditionRule(condConfig, p)
}
