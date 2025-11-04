package mongox

import (
	"github.com/tencent-go/pkg/errx"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	envURIKey = "MONGODB_URI_BASE64"
	envDBName = "MONGODB_DB_NAME"
)

var (
	defaultSort        = bson.M{"_id": -1}
	defaultLimit int64 = 5000
)

type EntityConfig[T any] struct {
	IdSetter           func(*T, any)
	IDType             IDType
	TimeType           TimeType
	CreatedAtBsonField string
	CreatedAtSetter    func(*T, any)
	UpdatedAtBsonField string
	UpdatedAtSetter    func(*T, any)
	VersionBsonField   string
	VersionSetter      func(*T, int64)
	BsonParser         func(src *T, dst *bson.M, ignoreZeroValue bool) errx.Error //required
}

type Option[T any] func(*EntityConfig[T])

type IDType int

const (
	IDTypeAuto IDType = iota + 1
	IDTypeUUID
	IDTypeSnowflake
)

type TimeType int

const (
	TimeTypeRFC3339 TimeType = iota + 1 // ISO 8601
	TimeTypeRFC3339String
	TimeTypeUnixMilli
	TimeTypeUnixSec
)
