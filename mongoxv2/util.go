package mongox

import (
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/page"
	"github.com/tencent-go/pkg/types"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"strings"
	"time"
)

func Transaction[T any](ctx context.Context, callback func(sc mongo.SessionContext) (*T, errx.Error), opts ...*options.TransactionOptions) (*T, errx.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	session, err := GetDefaultClient().StartSession()
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	defer session.EndSession(ctx)
	res, err := session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return callback(sc)
	}, opts...)
	if err != nil {
		return nil, err.(errx.Error)
	}
	return res.(*T), nil
}

func GetPaginationOptions(pagination *page.Pagination, sortOptions ...page.Sortable) *options.FindOptions {
	if pagination == nil {
		return nil
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Current <= 0 {
		pagination.Current = 1
	}
	findOptions := options.Find()
	findOptions.SetSkip((pagination.Current - 1) * pagination.PageSize)
	findOptions.SetLimit(pagination.PageSize)
	sort := defaultSort
	if len(sortOptions) > 0 {
		s := sortOptions[0]
		if s.GetSort() != "" {
			d := 1
			if s.GetDirection() == page.DirectionDesc {
				d = -1
			}
			sort = bson.M{s.GetSort(): d}
		}
	}
	findOptions.SetSort(sort)
	return findOptions
}

func isZeroID(id any, idType IDType) bool {
	return reflect.ValueOf(id).IsZero()
}

func createID(idType IDType) any {
	switch idType {
	case IDTypeUUID:
		return uuid.New().String()
	case IDTypeSnowflake:
		return types.NewID()
	default:
		return nil
	}
}

func isZeroTime(t any, timeType TimeType) bool {
	return reflect.ValueOf(t).IsZero()
}

func createCurrentTime(timeType TimeType) any {
	switch timeType {
	case TimeTypeRFC3339:
		return time.Now()
	case TimeTypeRFC3339String:
		return time.Now().Format(time.RFC3339)
	case TimeTypeUnixMilli:
		return time.Now().UnixMilli()
	case TimeTypeUnixSec:
		return time.Now().Unix()
	default:
		return nil
	}
}

type basicField struct {
	path []string
	typ  reflect.Type
}

func fillDefaultCollectionConfig[T any](c *EntityConfig[T]) {
	typ := reflect.TypeOf(*(new(T)))
	basicFields := getBasicFields(typ)
	if c.BsonParser == nil {
		c.BsonParser = newDefaultBsonParser[T]()
	}
	if c.IDType == 0 || c.IdSetter == nil {
		idField, ok := basicFields["_id"]
		if !ok {
			logrus.Fatalf("%s.%s id field not found", typ.PkgPath(), typ.Name())
		}
		if c.IDType == 0 {
			c.IDType = getIDType(idField.typ)
		}
		if c.IdSetter == nil {
			c.IdSetter = newDefaultSetter[T](idField.path)
		}
	}

	if c.CreatedAtBsonField == "" || c.CreatedAtSetter == nil {
		if f, ok := basicFields["createdAt"]; ok {
			if c.TimeType == 0 {
				c.TimeType = getTimeType(f.typ)
			}
			if c.CreatedAtBsonField == "" {
				c.CreatedAtBsonField = "createdAt"
			}
			if c.CreatedAtSetter == nil {
				c.CreatedAtSetter = newDefaultSetter[T](f.path)
			}
		}
	}

	if c.UpdatedAtBsonField == "" || c.UpdatedAtSetter == nil {
		if f, ok := basicFields["updatedAt"]; ok {
			if c.TimeType == 0 {
				c.TimeType = getTimeType(f.typ)
			}
			if c.UpdatedAtBsonField == "" {
				c.UpdatedAtBsonField = "updatedAt"
			}
			if c.UpdatedAtSetter == nil {
				c.UpdatedAtSetter = newDefaultSetter[T](f.path)
			}
		}
	}

	if c.VersionBsonField == "" || c.VersionSetter == nil {
		if f, ok := basicFields["version"]; ok {
			if c.VersionBsonField == "" {
				c.VersionBsonField = "version"
			}
			if c.VersionSetter == nil {
				c.VersionSetter = newDefaultInt64Setter[T](f.path)
			}
		}
	}
}

func newDefaultInt64Setter[T any](path []string) func(obj *T, value int64) {
	return func(obj *T, value int64) {
		if obj == nil {
			return
		}
		target := reflect.ValueOf(obj).Elem()
		for _, s := range path {
			target = target.FieldByName(s)
			for target.Kind() == reflect.Ptr {
				if target.IsNil() {
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}
		}
		target.SetInt(value)
	}
}

func newDefaultSetter[T any](path []string) func(obj *T, value any) {
	return func(obj *T, value any) {
		if obj == nil {
			return
		}
		target := reflect.ValueOf(obj).Elem()
		for _, s := range path {
			target = target.FieldByName(s)
			for target.Kind() == reflect.Ptr {
				if target.IsNil() {
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}
		}
		if !target.CanSet() {
			t := reflect.TypeOf(*obj)
			logrus.Fatalf("can not set %s.%s field %v", t.PkgPath(), t.Name(), strings.Join(path, "."))
		}
		v := reflect.ValueOf(value)
		if v.Type().ConvertibleTo(target.Type()) {
			target.Set(v.Convert(target.Type()))
			return
		}
		logrus.Fatalf("can not convert %v to %v", v.Type(), target.Type())
	}
}

func getTimeType(t reflect.Type) TimeType {
	if reflect.TypeOf(time.Time{}) == t {
		return TimeTypeRFC3339
	}
	switch t.Kind() {
	case reflect.String:
		return TimeTypeRFC3339String
	case reflect.Int64:
		return TimeTypeUnixMilli
	default:
		return TimeTypeUnixSec
	}
}

func getIDType(t reflect.Type) IDType {
	switch t.Kind() {
	case reflect.String:
		return IDTypeUUID
	default:
		return IDTypeSnowflake
	}
}

func getBasicFields(typ reflect.Type, path ...string) map[string]basicField {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		logrus.Panicf("obj %s must be a struct", typ.String())
	}
	res := make(map[string]basicField)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := field.Name
		if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
			elements := strings.Split(bsonTag, ",")
			if len(elements) > 0 {
				n := elements[0]
				if n == "-" {
					continue
				}
				if n != "" {
					name = n
				}
				for _, opt := range elements[1:] {
					if opt == "inline" {
						//递归
						for k, v := range getBasicFields(field.Type, append(path, field.Name)...) {
							res[k] = v
						}
						continue
					}
				}
			}
		}
		switch name {
		case "createdAt", "updatedAt", "version", "_id":
			res[name] = basicField{path: append(path, field.Name), typ: field.Type}
			continue
		}
	}
	return res
}

type valueField struct {
	path      []string
	omitempty bool
	bsonKey   string
}

func newDefaultBsonParser[T any]() func(obj *T, dst *bson.M, ignoreZeroValue bool) errx.Error {
	valueFields := getValueFields(reflect.TypeOf(*(new(T))))
	return func(obj *T, dst *bson.M, ignoreZeroValue bool) errx.Error {
		if *dst == nil {
			*dst = bson.M{}
		}
		m := *dst
		if obj == nil {
			return nil
		}
		value := reflect.ValueOf(*obj)
		for _, field := range valueFields {
			if len(field.path) == 0 {
				continue
			}
			v := value
			isNil := false
			for i, s := range field.path {
				if !v.IsValid() {
					isNil = true
					break
				}
				if i < len(field.path)-1 {
					for v.Kind() == reflect.Ptr {
						if v.IsNil() {
							isNil = true
							break
						}
						v = v.Elem()
					}
				}
				if !isNil {
					v = v.FieldByName(s)
				}
			}
			if isNil || !v.IsValid() || (v.IsZero() && field.omitempty && ignoreZeroValue) {
				continue
			}
			m[field.bsonKey] = v.Interface()
		}
		return nil
	}
}

func getValueFields(typ reflect.Type, path ...string) []valueField {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		logrus.Panic("obj must be a struct")
	}
	var res []valueField
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := field.Name
		omitempty := false
		inline := false
		if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
			elements := strings.Split(bsonTag, ",")
			if len(elements) > 0 {
				n := elements[0]
				if n == "-" {
					continue
				}
				if n != "" {
					name = n
				}
			}
			for _, opt := range elements[1:] {
				if opt == "inline" {
					inline = true
				}
				if opt == "omitempty" {
					omitempty = true
				}
			}
		}
		if inline {
			//递归
			res = append(res, getValueFields(field.Type, append(path, field.Name)...)...)
			continue
		} else {
			if name == "" {
				logrus.Warnf("%s.%s field %s has no bson key", typ.PkgPath(), typ.Name(), field.Name)
			}
		}
		res = append(res, valueField{path: append(path, field.Name), omitempty: omitempty, bsonKey: name})
	}
	return res
}
