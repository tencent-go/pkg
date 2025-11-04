package types

import (
	"github.com/tencent-go/pkg/errx"
	"net"
)

type IP string

func (i IP) Validate() errx.Error {
	if net.ParseIP(string(i)) == nil {
		return errx.Validation.WithMsg("invalid ip address").Err()
	}
	return nil
}

type IPGeolocation struct {
	Country string `json:"country,omitempty"`
	Region  string `json:"region,omitempty"`
	City    string `json:"city,omitempty"`
}
