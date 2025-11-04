package types

import (
	"github.com/tencent-go/pkg/errx"
	"strings"
)

type RealName struct {
	FirstName  string `json:"firstName" bson:"firstName"`
	MiddleName string `json:"middleName,omitempty" bson:"middleName,omitempty"`
	LastName   string `json:"lastName,omitempty" bson:"lastName,omitempty"`
}

func (r RealName) Validate() errx.Error {
	if r.FirstName == "" {
		return errx.Validation.WithMsg("First name is required").Err()
	}
	return nil
}

func (r RealName) FullName() string {
	return r.FirstName + " " + r.MiddleName + " " + r.LastName
}

func (r RealName) Equal(t RealName) bool {
	return r.FirstName == t.FirstName && r.MiddleName == t.MiddleName && r.LastName == t.LastName
}

func NewRealNameFromFullName(fullName string) RealName {
	parts := strings.Fields(fullName)
	var firstName, middleName, lastName string
	if len(parts) == 2 {
		firstName = parts[0]
		lastName = parts[1]
	} else if len(parts) == 3 {
		firstName = parts[0]
		middleName = parts[1]
		lastName = parts[2]
	}
	return RealName{
		FirstName:  firstName,
		MiddleName: middleName,
		LastName:   lastName,
	}
}
