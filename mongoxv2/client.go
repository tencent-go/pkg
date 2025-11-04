package mongox

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/tencent-go/pkg/env"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/shutdown"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetDefaultClient() *mongo.Client {
	return defaultClient()
}

var defaultClient = sync.OnceValue(func() *mongo.Client {
	client, err := NewClientWithEnvConfig()
	if err != nil {
		logrus.Fatalf("create mongodb connection failed: %v", err)
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		defer logrus.Infoln("mongodb connection closed")
		return client.Disconnect(ctx)
	}, true)
	return client
})

type Config struct {
	MongoURI      string `env:"MONGO_URI"`
	DefaultDBName string `env:"MONGO_DEFAULT_DB_NAME" default:"default"`
}

var configReader = env.NewReaderBuilder[Config]().Build()

func NewClientWithEnvConfig(opts ...*options.ClientOptions) (*mongo.Client, errx.Error) {
	arr := []*options.ClientOptions{options.Client().ApplyURI(configReader.Read().MongoURI)}
	opts = append(arr, opts...)
	client, err := NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewClient(opts ...*options.ClientOptions) (*mongo.Client, errx.Error) {
	r := bson.NewRegistry()
	{
		t := reflect.TypeOf(decimal.Decimal{})
		d := &decimalCodec{}
		r.RegisterTypeDecoder(t, d)
		r.RegisterTypeEncoder(t, d)
	}
	arr := []*options.ClientOptions{options.Client().SetRegistry(r)}
	opts = append(arr, opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, opts...)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return client, nil
}

type decimalCodec struct{}

func (dc *decimalCodec) EncodeValue(ctx bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return bsoncodec.ValueEncoderError{Name: "DecimalCodec.EncodeValue", Types: []reflect.Type{reflect.TypeOf(decimal.Decimal{})}, Received: val}
	}
	d, ok := val.Interface().(decimal.Decimal)
	if !ok {
		return bsoncodec.ValueEncoderError{Name: "DecimalCodec.EncodeValue", Types: []reflect.Type{reflect.TypeOf(decimal.Decimal{})}, Received: val}
	}

	dec128, err := primitive.ParseDecimal128(d.String())
	if err != nil {
		return err
	}

	return vw.WriteDecimal128(dec128)
}

func (dc *decimalCodec) DecodeValue(ctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if !val.CanSet() || val.Kind() != reflect.Struct {
		return bsoncodec.ValueDecoderError{Name: "DecimalCodec.DecodeValue", Types: []reflect.Type{reflect.TypeOf(decimal.Decimal{})}, Received: val}
	}
	switch vr.Type() {
	case bson.TypeDouble:
		if d, err := vr.ReadDouble(); err == nil {
			val.Set(reflect.ValueOf(decimal.NewFromFloat(d)))
		}
	case bson.TypeInt32:
		if i32, err := vr.ReadInt32(); err == nil {
			val.Set(reflect.ValueOf(decimal.NewFromInt(int64(i32))))
		}
	case bson.TypeInt64:
		if i64, err := vr.ReadInt64(); err == nil {
			val.Set(reflect.ValueOf(decimal.NewFromInt(i64)))
		}
	case bson.TypeDecimal128:
		if dec128, err := vr.ReadDecimal128(); err == nil {
			dec, err := decimal.NewFromString(dec128.String())
			if err != nil {
				return err
			}
			val.Set(reflect.ValueOf(dec))
		}
	case bson.TypeString:
		if str, err := vr.ReadString(); err == nil {
			dec, err := decimal.NewFromString(str)
			if err != nil {
				return err
			}
			val.Set(reflect.ValueOf(dec))
		}
	default:
		return bsoncodec.ValueDecoderError{Name: "DecimalCodec.DecodeValue", Types: []reflect.Type{reflect.TypeOf(decimal.Decimal{})}, Received: val}
	}
	return nil
}
