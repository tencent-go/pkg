package validation

import (
	"regexp"
	"strings"

	"github.com/tencent-go/pkg/errx"
)

type Rule struct {
	Required *bool
	Label    *string
	Min      *string
	Max      *string
	Pattern  *regexp.Regexp
	Dive     *Rule
	MapKey   *Rule
}

func (o *Rule) SetRequired(required bool) *Rule {
	o.Required = &required
	return o
}

func (o *Rule) SetLabel(label string) *Rule {
	o.Label = &label
	return o
}

func (o *Rule) SetMin(min string) *Rule {
	o.Min = &min
	return o
}

func (o *Rule) SetMax(max string) *Rule {
	o.Max = &max
	return o
}

func (o *Rule) SetPattern(pattern *regexp.Regexp) *Rule {
	o.Pattern = pattern
	return o
}

func (o *Rule) SetDive(dive *Rule) *Rule {
	o.Dive = dive
	return o
}

func (o *Rule) SetMapKey(mapKey *Rule) *Rule {
	o.MapKey = mapKey
	return o
}

var re1 = regexp.MustCompile("^[a-z]+$")
var re2 = regexp.MustCompile("^[a-z]+,")
var re3 = regexp.MustCompile("^([a-z]+)='(.+)'(?:,[a-z]+)?")
var re4 = regexp.MustCompile("^([a-z]+)=(.+)$")
var re4Option = regexp.MustCompile(`,([a-z]+)`)

func (o *Rule) Parse(tag string) errx.Error {
	waiting := strings.ReplaceAll(tag, " ", "")
	for len(waiting) > 0 {
		var key, value string
		{
			if re1.MatchString(waiting) {
				key = waiting
				waiting = ""
			} else if idxs := re2.FindStringIndex(waiting); idxs != nil {
				key = waiting[:idxs[1]-1]
				waiting = waiting[idxs[1]:]
			} else if idxs = re3.FindStringSubmatchIndex(waiting); idxs != nil {
				key = waiting[:idxs[3]]
				value = waiting[idxs[4]:idxs[5]]
				waiting = strings.TrimLeft(waiting[idxs[5]+1:], ",")
			} else if kvMatched := re4.FindStringSubmatch(waiting); kvMatched != nil {
				key = kvMatched[1]
				value = kvMatched[2]
				if commaIdx := re4Option.FindStringIndex(value); commaIdx != nil {
					waiting = value[commaIdx[0]+1:]
					value = value[:commaIdx[0]]
				} else {
					waiting = ""
				}
			} else {
				return errx.Newf("Invalid validate tag:%s, unknown key:%s", tag, waiting)
			}
			if key == "keys" {
				idx := strings.Index(waiting, ",endkeys")
				if idx == -1 {
					value = waiting
					waiting = ""
				} else {
					value = waiting[:idx]
					waiting = strings.TrimLeft(waiting[idx+len(",endkeys"):], ",")
				}
			}
		}
		switch key {
		case "required":
			switch value {
			case "":
				o.SetRequired(true)
			case "true":
				o.SetRequired(true)
			case "false":
				o.SetRequired(false)
			}
		case "label":
			o.SetLabel(value)
		case "min":
			o.SetMin(value)
		case "max":
			o.SetMax(value)
		case "pattern":
			p, err := regexp.Compile(value)
			if err != nil {
				return errx.Newf("Invalid validate tag:%s, pattern:%s", tag, value)
			}
			o.SetPattern(p)
		case "dive":
			r := &Rule{}
			if len(waiting) > 0 {
				if err := r.Parse(waiting); err != nil {
					return err
				}
			}
			o.SetDive(r)
			return nil
		case "keys":
			r := &Rule{}
			if err := r.Parse(value); err != nil {
				return err
			}
			o.SetMapKey(r)
		default:
			return errx.Newf("Invalid validate tag:%s, unknown key:%s", tag, key)
		}
	}
	return nil
}
