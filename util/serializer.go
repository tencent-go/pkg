package util

import (
	"bytes"
	"net/url"
	"sync"

	"github.com/tencent-go/pkg/errx"
	"github.com/gorilla/schema"
	jsoniter "github.com/json-iterator/go"
	"github.com/vmihailenco/msgpack/v5"
)

func Json() Serializer {
	return json
}

var json = &jsonSerializer{jsoniter.Config{
	SortMapKeys: true,
}.Froze()}

type jsonSerializer struct {
	jsoniter.API
}

func (j *jsonSerializer) Unmarshal(data []byte, dst any) errx.Error {
	err := j.API.Unmarshal(data, dst)
	if err != nil {
		return errx.Wrap(err).WithMsgf("Failed to unmarshal JSON data").Err()
	}
	return nil
}

func (j *jsonSerializer) Marshal(src any) ([]byte, errx.Error) {
	data, err := j.API.Marshal(src)
	if err != nil {
		return nil, errx.Wrap(err).WithMsgf("Failed to marshal JSON data").Err()
	}
	return data, nil
}

type Serializer interface {
	Unmarshal(data []byte, dst any) errx.Error
	Marshal(src any) ([]byte, errx.Error)
}

type FormSerializer interface {
	Serializer
	Bind(src map[string][]string, dst any) errx.Error
	Extract(src any, dst map[string][]string) errx.Error
}

type formSerializer struct {
	*schema.Decoder
	*schema.Encoder
}

func (v *formSerializer) Unmarshal(data []byte, dst any) errx.Error {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return errx.Wrap(err).AppendMsgf("Failed to parse form data").Err()
	}

	err = v.Decode(dst, values)
	if err != nil {
		return errx.Wrap(err).AppendMsgf("Failed to decode form data").Err()
	}

	return nil
}

func (v *formSerializer) Marshal(src any) ([]byte, errx.Error) {
	values := make(map[string][]string)
	err := v.Encode(src, values)
	if err != nil {
		return nil, errx.Wrap(err).AppendMsgf("Failed to encode form data").Err()
	}

	// Convert map to URL encoded string
	params := url.Values{}
	for key, vals := range values {
		for _, val := range vals {
			params.Add(key, val)
		}
	}

	return []byte(params.Encode()), nil
}

func (v *formSerializer) Bind(src map[string][]string, dst any) errx.Error {
	err := v.Decode(dst, src)
	if err != nil {
		return errx.Wrap(err).AppendMsgf("Failed to bind form data").Err()
	}

	return nil
}

func (v *formSerializer) Extract(src any, dst map[string][]string) errx.Error {
	err := v.Encode(src, dst)
	if err != nil {
		return errx.Wrap(err).AppendMsgf("Failed to extract form data").Err()
	}

	return nil
}

var formSerializerInstance = sync.OnceValue(func() *formSerializer {
	d := schema.NewDecoder()
	d.SetAliasTag("form")
	e := schema.NewEncoder()
	e.SetAliasTag("form")
	return &formSerializer{d, e}
})

func Form() FormSerializer {
	return formSerializerInstance()
}

type msgpackSerializer struct{}

func (v *msgpackSerializer) Unmarshal(data []byte, dst any) errx.Error {
	reader := bytes.NewReader(data)
	decoder := msgpack.NewDecoder(reader)
	decoder.SetCustomStructTag("json")
	err := decoder.Decode(dst)
	if err != nil {
		return errx.Wrap(err).AppendMsgf("Failed to unmarshal msgpack data").Err()
	}
	return nil
}

func (v *msgpackSerializer) Marshal(src any) ([]byte, errx.Error) {
	var buf bytes.Buffer
	encoder := msgpack.NewEncoder(&buf)
	encoder.SetCustomStructTag("json")
	err := encoder.Encode(src)
	if err != nil {
		return nil, errx.Wrap(err).AppendMsgf("Failed to marshal msgpack data").Err()
	}
	return buf.Bytes(), nil
}

func Msgpack() Serializer {
	return &msgpackSerializer{}
}
