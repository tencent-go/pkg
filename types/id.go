package types

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/tencent-go/pkg/errx"
	"github.com/bwmarrin/snowflake"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

var node *snowflake.Node

func SetIDNodeByString(str string) {
	h := fnv.New32a()
	h.Write([]byte(str))
	n, err := snowflake.NewNode(int64(h.Sum32()) % 1024)
	if err != nil {
		panic(err)
	}
	node = n
}

func init() {
	n, err := snowflake.NewNode(0)
	if err != nil {
		panic(err)
	}
	node = n
}

type ID int64

func (m ID) String() string {
	return strconv.FormatInt(int64(m), 10)
}

func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

// UnmarshalJSON 將 JSON 字符串反序列化為 ID
func (id *ID) UnmarshalJSON(data []byte) error {
	// 移除引號
	str := string(data)
	if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") {
		str = str[1 : len(str)-1]
	}
	if len(str) == 0 {
		*id = 0
		return nil
	}
	parsed, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}
	*id = ID(parsed)
	return nil
}

func (id ID) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bson.MarshalValue(int64(id))
}

// 实现 bson.Unmarshaler 接口
func (id *ID) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	var tmp int64
	err := bson.UnmarshalValue(t, data, &tmp)
	if err != nil {
		return err
	}
	*id = ID(tmp)
	return nil
}
func (id ID) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.EncodeString(strconv.FormatInt(int64(id), 10))
}

func (id *ID) DecodeMsgpack(dec *msgpack.Decoder) error {
	code, err := dec.PeekCode()
	if err != nil {
		return err
	}

	if (code >= 0xa0 && code <= 0xbf) || code == 0xd9 || code == 0xda || code == 0xdb {
		s, err := dec.DecodeString()
		if err != nil {
			return err
		}
		if s == "" {
			*id = 0
			return nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*id = ID(v)
		return nil
	}

	v, err := dec.DecodeInt64()
	if err != nil {
		return err
	}
	*id = ID(v)
	return nil
}

func NewID() ID {
	if node == nil {
		panic("NewID() called with nil node")
	}
	return ID(node.Generate())
}

func NewIDFromString(str string) (ID, errx.Error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, errx.Wrap(err).Err()
	}
	return ID(i), nil
}

const EmptyID = ID(0)
