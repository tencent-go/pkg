package types

import (
	"github.com/tencent-go/pkg/errx"
	"fmt"
	"regexp"
	"strings"
)

type PhoneNumber struct {
	CountryCode CountryCode `json:"countryCode" bson:"countryCode"`
	Number      string      `json:"number" bson:"number"`
}

type validate func(string) errx.Error

var (
	reChina       = regexp.MustCompile(`^1[3-9]\d{9}$`)
	rePhilippines = regexp.MustCompile(`^(9)\d{9}$`)
)

var validateMap = map[CountryCode]validate{
	CountryCodeChina: func(s string) errx.Error {
		if reChina.MatchString(s) {
			return nil
		}
		return errx.Validation.WithMsgf("Phone number %s format is incorrect", s).Err()
	},
	CountryCodePhilippines: func(s string) errx.Error {
		if rePhilippines.MatchString(s) {
			return nil
		}
		return errx.Validation.WithMsgf("Phone number %s format is incorrect", s).Err()
	},
}

func (v PhoneNumber) Validate() errx.Error {
	if !v.CountryCode.Enum().Contains(v.CountryCode) {
		return errx.Validation.WithMsgf("unsupport country code: %s", v.CountryCode).Err()
	}
	va, ok := validateMap[v.CountryCode]
	if !ok {
		return errx.Validation.WithMsgf("unsupport country code: %s", v.CountryCode).Err()
	}
	return va(v.Number)
}

func (v PhoneNumber) IsEmpty() bool {
	return v.CountryCode == "" || v.Number == ""
}

func (v PhoneNumber) Equal(v2 PhoneNumber) bool {
	return v.CountryCode == v2.CountryCode && v.Number == v2.Number
}

func (v PhoneNumber) Mask() string {
	number := v.Number
	if len(number) > 6 {
		prefix := number[:3]
		suffix := number[len(number)-2:]
		middle := strings.Repeat("*", len(number)-5)
		number = prefix + middle + suffix
	}
	return fmt.Sprintf("+%s %s", v.CountryCode, number)
}
