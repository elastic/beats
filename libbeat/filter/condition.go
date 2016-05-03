package filter

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Condition struct {
	Equals   map[string]EqualsValue
	Contains map[string]string
	Regexp   map[string]*regexp.Regexp
	Range    map[string]RangeValue
}

func NewCondition(config ConditionConfig) (*Condition, error) {

	c := Condition{}
	c.Equals = map[string]EqualsValue{}
	c.Contains = map[string]string{}
	c.Regexp = map[string]*regexp.Regexp{}
	c.Range = map[string]RangeValue{}

	if err := c.AddEquals(config.Equals); err != nil {
		return nil, err
	}
	if err := c.AddContains(config.Contains); err != nil {
		return nil, err
	}
	if err := c.AddRegexp(config.Regexp); err != nil {
		return nil, err
	}
	if err := c.AddRange(config.Range); err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *Condition) AddEquals(equals map[string]string) error {

	for field, value := range equals {

		i, err := strconv.Atoi(value)
		if err == nil {
			c.Equals[field] = EqualsValue{Int: i}
		} else {
			c.Equals[field] = EqualsValue{Str: value}
		}
	}
	return nil
}

func (c *Condition) AddContains(contains map[string]string) error {

	c.Contains = contains
	return nil
}

func (c *Condition) AddRegexp(r map[string]string) error {

	for field, value := range r {
		reg, err := regexp.CompilePOSIX(value)
		if err != nil {
			return err
		}
		c.Regexp[field] = reg
	}
	return nil
}

func (c *Condition) AddRange(config map[string]RangeValue) error {

	for field, rangeValue := range config {
		c.Range[field] = rangeValue
	}
	return nil
}

func (c *Condition) Check(event common.MapStr) bool {

	if !c.CheckEquals(event) {
		return false
	}
	if !c.CheckContains(event) {
		return false
	}
	if !c.CheckRegexp(event) {
		return false
	}
	if !c.CheckRange(event) {
		return false
	}

	return true
}

func (c *Condition) CheckEquals(event common.MapStr) bool {

	for field, equalValue := range c.Equals {

		value, err := event.GetValue(field)
		if err != nil {
			logp.Debug("filter", "unavailable field %s: %v", field, err)
			return false
		}

		switch value.(type) {
		case uint8, uint16, uint32, uint64, int8, int16, int32, int64, int, uint:
			return value == equalValue.Int
		case string:
			return value == equalValue.Str
		default:
			logp.Warn("unexpected type %T in equals condition as it accepts only integers and strings. ", value)
			return false
		}

	}

	return true

}

func (c *Condition) CheckContains(event common.MapStr) bool {

	for field, equalValue := range c.Contains {

		value, err := event.GetValue(field)
		if err != nil {
			logp.Debug("filter", "unavailable field %s: %v", field, err)
			return false
		}

		switch value.(type) {
		case string:
			return strings.Contains(value.(string), equalValue)
		default:
			logp.Warn("unexpected type %T in contains condition as it accepts only strings. ", value)
			return false
		}

	}

	return true

}

func (c *Condition) CheckRegexp(event common.MapStr) bool {

	for field, equalValue := range c.Regexp {

		value, err := event.GetValue(field)
		if err != nil {
			logp.Debug("filter", "unavailable field %s: %v", field, err)
			return false
		}

		switch value.(type) {
		case string:
			if !equalValue.MatchString(value.(string)) {
				return false
			}
		default:
			logp.Warn("unexpected type %T in regexp condition as it accepts only strings. ", value)
			return false
		}

	}

	return true

}

func (c *Condition) CheckRange(event common.MapStr) bool {

	for field, rangeValue := range c.Range {

		value, err := event.GetValue(field)
		if err != nil {
			logp.Debug("filter", "unavailable field %s: %v", field, err)
			return false
		}

		switch value.(type) {
		case int, int8, int16, int32, int64:
			int_value := reflect.ValueOf(value).Int()

			if rangeValue.Gte != nil {
				if int_value < int64(*rangeValue.Gte) {
					return false
				}
			}
			if rangeValue.Gt != nil {
				if int_value <= int64(*rangeValue.Gt) {
					return false
				}
			}
			if rangeValue.Lte != nil {
				if int_value > int64(*rangeValue.Lte) {
					return false
				}
			}
			if rangeValue.Lt != nil {
				if int_value >= int64(*rangeValue.Lt) {
					return false
				}
			}

		case uint, uint8, uint16, uint32, uint64:
			uint_value := reflect.ValueOf(value).Uint()

			if rangeValue.Gte != nil {
				if uint_value < uint64(*rangeValue.Gte) {
					return false
				}
			}
			if rangeValue.Gt != nil {
				if uint_value <= uint64(*rangeValue.Gt) {
					return false
				}
			}
			if rangeValue.Lte != nil {
				if uint_value > uint64(*rangeValue.Lte) {
					return false
				}
			}
			if rangeValue.Lt != nil {
				if uint_value >= uint64(*rangeValue.Lt) {
					return false
				}
			}

		case float64, float32:
			float_value := reflect.ValueOf(value).Float()

			if rangeValue.Gte != nil {
				if float_value < *rangeValue.Gte {
					return false
				}
			}
			if rangeValue.Gt != nil {
				if float_value <= *rangeValue.Gt {
					return false
				}
			}
			if rangeValue.Lte != nil {
				if float_value > *rangeValue.Lte {
					return false
				}
			}
			if rangeValue.Lt != nil {
				if float_value >= *rangeValue.Lt {
					return false
				}
			}

		default:
			logp.Warn("unexpected type %T in range condition as it accepts only strings. ", value)
			return false
		}

	}
	return true
}

func (c *Condition) String() string {

	s := ""

	if len(c.Equals) > 0 {
		s = s + fmt.Sprintf("equals: %v", c.Equals)
	}
	if len(c.Contains) > 0 {
		s = s + fmt.Sprintf("contains: %v", c.Contains)
	}
	if len(c.Regexp) > 0 {
		s = s + fmt.Sprintf("regexp: %v", c.Regexp)
	}
	if len(c.Range) > 0 {
		s = s + fmt.Sprintf("range: %v", c.Range)
	}
	return s
}

func (r RangeValue) String() string {

	s := ""
	if r.Gte != nil {
		s = s + fmt.Sprintf(">= %v", *r.Gte)
	}

	if r.Gt != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("> %v", *r.Gt)
	}

	if r.Lte != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("<= %v", *r.Lte)
	}
	if r.Lt != nil {
		if len(s) > 0 {
			s = s + " and "
		}
		s = s + fmt.Sprintf("< %v", *r.Lt)
	}
	return s
}

func (e EqualsValue) String() string {

	if len(e.Str) > 0 {
		return e.Str
	}
	return strconv.Itoa(e.Int)
}
