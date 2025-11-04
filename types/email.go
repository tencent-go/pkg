package types

import (
	"github.com/tencent-go/pkg/errx"
	"regexp"
	"strings"
)

type Email string

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (e Email) Validate() errx.Error {
	if emailRegex.MatchString(string(e)) {
		return nil
	}
	return errx.Validation.WithMsgf("Invalid email format %s", e).Err()
}

func (e Email) Mask() string {
	address := string(e)
	atIndex := strings.Index(address, "@")
	if atIndex == -1 {
		return address
	}

	localPart := address[:atIndex]
	domainPart := address[atIndex:]
	// 确保localPart至少有两个字符
	if len(localPart) > 2 {
		prefix := localPart[:1]
		suffix := localPart[len(localPart)-1:]
		middle := strings.Repeat("*", len(localPart)-2)
		localPart = prefix + middle + suffix
	}
	return localPart + domainPart
}
