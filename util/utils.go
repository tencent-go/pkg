package util

import (
	"bytes"
	"encoding/gob"
	"regexp"

	"github.com/tencent-go/pkg/errx"
)

func IsMilliSecond(t int64) bool {
	return t >= 1e12 && t < 1e13
}

func DeepCopy[T any](src, dst T) errx.Error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)

	if err := enc.Encode(src); err != nil {
		return errx.Wrap(err).WithMsg("encode failed").Err()
	}
	if err := dec.Decode(dst); err != nil {
		return errx.Wrap(err).WithMsg("decode failed").Err()
	}
	return nil
}

var PlaceholderRegex = regexp.MustCompile(`\{([^{}]+)\}`)
