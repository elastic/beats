package processors

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type RangeValue struct {
	gte *float64
	gt  *float64
	lte *float64
	lt  *float64
}

type EqualsValue struct {
	Int uint64
	Str string
}

type Condition struct {
	equals   map[string]EqualsValue
	contains map[string]string
	regexp   map[string]*regexp.Regexp
	rangexp  map[string]RangeValue
	or       []Condition
	and      []Condition
	not      *Condition
}

type WhenProcessor struct {
	condition *Condition
	p         Processor
}

func NewConditional(
	ruleFactory Constructor,
) Constructor {
	return func(cfg common.Config) (Processor, error) {
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

	if config.Equals != nil {
		if err := c.setEquals(config.Equals); err != nil {
			return nil, err
		}
	} else if config.Contains != nil {
		if err := c.setContains(config.Contains); err != nil {
			return nil, err
		}
	} else if config.Regexp != nil {
		if err := c.setRegexp(config.Regexp); err != nil {
			return nil, err
		}
	} else if config.Range != nil {
		if err := c.setRange(config.Range); err != nil {
			return nil, err
		}
	} else if len(config.OR) > 0 {
		for _, condConfig := range config.OR {
			cond, err := NewCondition(&condConfig)
			if err != nil {
				return nil, err
			}
			c.or = append(c.or, *cond)
		}
	} else if len(config.AND) > 0 {
		for _, condConfig := range config.AND {
			cond, err := NewCondition(&condConfig)
			if err != nil {
				return nil, err
			}
			c.and = append(c.and, *cond)
		}
	} else if config.NOT != nil {
		cond, err := NewCondition(config.NOT)
		if err != nil {
			return nil, err
		}
		c.not = cond
	} else {
		return nil, fmt.Errorf("missing condition")
	}

	logp.Debug("processors", "New condition %s", c)
	return &c, nil
}

func (c *Condition) setEquals(cfg *ConditionFields) error {

	c.equals = map[string]EqualsValue{}

	for field, value := range cfg.fields {
		uintValue, err := extractInt(value)
		if err == nil {
			c.equals[field] = EqualsValue{Int: uintValue}
		} else {
			sValue, err := extractString(value)
			if err != nil {
				return err
			}
			c.equals[field] = EqualsValue{Str: sValue}
		}
	}

	return nil
}

func (c *Condition) setContains(cfg *ConditionFields) error {

	c.contains = map[string]string{}

	for field, value := range cfg.fields {
		switch v := value.(type) {
		case string:
			c.contains[field] = v
		default:
			return fmt.Errorf("unexpected type %T of %v", value, value)
		}
	}

	return nil
}

func (c *Condition) setRegexp(cfg *ConditionFields) error {

	var err error

	c.regexp = map[string]*regexp.Regexp{}
	for field, value := range cfg.fields {
		switch v := value.(type) {
		case string:
			c.regexp[field], err = regexp.Compile(v)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unexpected type %T of %v", value, value)
		}
	}
	return nil
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

func (c *Condition) Check(event common.MapStr) bool {

	if len(c.or) > 0 {
		return c.checkOR(event)
	}

	if len(c.and) > 0 {
		return c.checkAND(event)
	}

	if c.not != nil {
		return c.checkNOT(event)
	}

	if !c.checkEquals(event) {
		return false
	}
	if !c.checkContains(event) {
		return false
	}
	if !c.checkRegexp(event) {
		return false
	}
	if !c.checkRange(event) {
		return false
	}

	return true
}

func (c *Condition) checkEquals(event common.MapStr) bool {

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
		} else {
			sValue, err := extractString(value)
			if err != nil {
				logp.Warn("unexpected type %T in equals condition as it accepts only integers and strings. ", value)
				return false
			}
			if sValue != equalValue.Str {
				return false
			}
		}
	}

	return true

}

func (c *Condition) checkContains(event common.MapStr) bool {
outer:
	for field, equalValue := range c.contains {
		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		switch value.(type) {
		case string:
			if !strings.Contains(value.(string), equalValue) {
				return false
			}
		case []string:
			for _, s := range value.([]string) {
				if strings.Contains(s, equalValue) {
					continue outer
				}
			}
			return false
		default:
			logp.Warn("unexpected type %T in contains condition as it accepts only strings.", value)
			return false
		}
	}

	return true
}

func (c *Condition) checkRegexp(event common.MapStr) bool {

	for field, equalValue := range c.regexp {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		sValue, err := extractString(value)
		if err != nil {
			logp.Warn("unexpected type %T in regexp condition as it accepts only strings. ", value)
			return false
		}
		if !equalValue.MatchString(sValue) {
			return false
		}
	}

	return true

}

func (c *Condition) checkRange(event common.MapStr) bool {

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

func (c *Condition) checkOR(event common.MapStr) bool {

	for _, cond := range c.or {
		if cond.Check(event) {
			return true
		}
	}
	return false
}

func (c *Condition) checkAND(event common.MapStr) bool {

	for _, cond := range c.and {
		if !cond.Check(event) {
			return false
		}
	}
	return true
}

func (c *Condition) checkNOT(event common.MapStr) bool {

	if c.not.Check(event) {
		return false
	}
	return true
}

func (c Condition) String() string {

	s := ""

	if len(c.equals) > 0 {
		s = s + fmt.Sprintf("equals: %v", c.equals)
	}
	if len(c.contains) > 0 {
		s = s + fmt.Sprintf("contains: %v", c.contains)
	}
	if len(c.regexp) > 0 {
		s = s + fmt.Sprintf("regexp: %v", c.regexp)
	}
	if len(c.rangexp) > 0 {
		s = s + fmt.Sprintf("range: %v", c.rangexp)
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

func (r *WhenProcessor) Run(event common.MapStr) (common.MapStr, error) {
	if !r.condition.Check(event) {
		return event, nil
	}
	return r.p.Run(event)
}

func (r *WhenProcessor) String() string {
	return fmt.Sprintf("%v, condition=%v", r.p.String(), r.condition.String())
}

func addCondition(
	cfg common.Config,
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
