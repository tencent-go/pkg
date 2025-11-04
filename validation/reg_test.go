package validation

import (
	"regexp"
	"strings"
	"testing"
)

func TestRuleStringRegexp(t *testing.T) {
	var re2 = regexp.MustCompile("^[a-z]+,")

	t.Run("re2", func(t *testing.T) {
		const str = "skfjasdks,sakdlj,DK#$$#fkasjk23423"
		m := re2.FindStringIndex(str)
		if m == nil {
			t.Fatal("re2.FindStringIndex() returned nil")
		}
		if str[:m[1]-1] != "skfjasdks" {
			t.Error("re2.FindStringIndex() returned " + str[:m[1]-1])
		}
		if str[m[1]:] != "sakdlj,DK#$$#fkasjk23423" {
			t.Error("re2.FindStringIndex() returned " + str[:m[1]-1])
		}
	})

	var re3 = regexp.MustCompile("^([a-z]+)='(.+)'(?:,[a-z]+)?")

	t.Run("re3", func(t *testing.T) {
		cases := []struct {
			str, key, value, waiting string
		}{
			{"key='value',key2=value2,key3", "key", "value", "key2=value2,key3"},
			{"key='value'", "key", "value", ""},
			{"key='value,v2',waiting", "key", "value,v2", "waiting"},
		}
		for _, c := range cases {
			res := re3.FindStringSubmatch(c.str)
			if res == nil {
				t.Fatal("match failed")
			}
			key := res[1]
			value := res[2]
			if key != c.key {
				t.Error("key mismatch")
			}
			if value != c.value {
				t.Error("value mismatch")
			}
			waiting := c.str[len(key)+len(value)+3:]
			waiting = strings.TrimPrefix(waiting, ",")
			if waiting != c.waiting {
				t.Error("waiting mismatch")
			}
		}
		for _, c := range cases {
			res := re3.FindStringSubmatchIndex(c.str)
			if res == nil {
				t.Fatal("match failed")
			}
			key := c.str[:res[3]]
			value := c.str[res[4]:res[5]]
			if key != c.key {
				t.Error("key mismatch")
			}
			if value != c.value {
				t.Error("value mismatch")
			}
			waiting := c.str[res[5]+1:]
			waiting = strings.TrimPrefix(waiting, ",")
			if waiting != c.waiting {
				t.Error("waiting mismatch")
			}
		}
	})

	var re4 = regexp.MustCompile("^([a-z]+)=(.+)$")
	var re4Option = regexp.MustCompile(`,([a-z]+)`)
	t.Run("re4", func(t *testing.T) {
		cases := []struct {
			str, key, value, waiting string
		}{
			{"key=value,key2=value2,key3", "key", "value", "key2=value2,key3"},
			{"key=value", "key", "value", ""},
			{"key=value,v2,waiting", "key", "value", "v2,waiting"},
			{"key=value,&v2,waiting", "key", "value,&v2", "waiting"},
		}
		for _, c := range cases {
			res := re4.FindStringSubmatch(c.str)
			if res == nil {
				t.Fatal("match failed")
			}
			key := res[1]
			value := res[2]
			if key != c.key {
				t.Error("key mismatch")
			}
			var waiting string
			m := re4Option.FindStringIndex(value)
			if m != nil {
				waiting = value[m[0]+1:]
				value = value[:m[0]]
			}
			if value != c.value {
				t.Error("value mismatch")
			}
			if waiting != c.waiting {
				t.Error("waiting mismatch")
			}
		}
	})

}
