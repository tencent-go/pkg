package types

import (
	"github.com/tencent-go/pkg/errx"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type (
	PlainPassword  string
	CipherPassword string
)

func (p PlainPassword) Encrypt() (*CipherPassword, errx.Error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(p), 10)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	res := CipherPassword(bytes)
	return &res, nil
}

func (c CipherPassword) Verify(p PlainPassword) errx.Error {
	err := bcrypt.CompareHashAndPassword([]byte(c), []byte(p))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errx.Validation.WithMsg("password mismatch").Err()
		}
		return errx.Wrap(err).Err()
	}
	return nil
}
