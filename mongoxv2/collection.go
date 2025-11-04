package mongox

import (
	"context"
	"errors"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection[T any] interface {
	Raw() *mongo.Collection
	WithOptions(opts ...*options.CollectionOptions) Collection[T]
	GetByID(ctx context.Context, id any, opts ...*options.FindOneOptions) (*T, errx.Error)
	GetOne(ctx context.Context, filter any, opts ...*options.FindOneOptions) (*T, errx.Error)
	GetList(ctx context.Context, filter any, opts ...*options.FindOptions) ([]T, errx.Error)
	Create(ctx context.Context, data *T, opts ...*options.InsertOneOptions) errx.Error
	UpdateByID(ctx context.Context, data *T, opts ...*UpdateOptions) errx.Error
	CreateOrUpdateByID(ctx context.Context, data *T, opts ...*UpdateOptions) (bool, errx.Error)
	GetAndCreateOrUpdateByID(ctx context.Context, data *T, opts ...*FindOneAndUpdateOptions) errx.Error
	DeleteByID(ctx context.Context, id any, opts ...*options.DeleteOptions) errx.Error
	CountDocuments(ctx context.Context, filter any, opts ...*options.CountOptions) (*int64, errx.Error)
	Watch(ctx context.Context, pipeline interface{}, cb func(ctx ctxx.Context, ev ChangeEventWithDoc[T]) errx.Error, opts ...*ChangeStreamOptions) (func(), errx.Error)
}

type collectionImpl[T any] struct {
	*mongo.Collection
	*EntityConfig[T]
}

func (c *collectionImpl[T]) Raw() *mongo.Collection {
	return c.Collection
}

func (c *collectionImpl[T]) WithOptions(opts ...*options.CollectionOptions) Collection[T] {
	newColl := c.Collection.Database().Collection(c.Name(), opts...)
	return &collectionImpl[T]{Collection: newColl, EntityConfig: c.EntityConfig}
}

func (c *collectionImpl[T]) GetByID(ctx context.Context, id any, opts ...*options.FindOneOptions) (*T, errx.Error) {
	return c.GetOne(ctx, bson.M{"_id": id}, opts...)
}

func (c *collectionImpl[T]) GetOne(ctx context.Context, filter any, opts ...*options.FindOneOptions) (*T, errx.Error) {
	res := c.FindOne(ctx, filter, opts...)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, mongo.ErrNilDocument) {
			return nil, errx.Wrap(err).WithType(errx.TypeNotFound).AppendMsg("data not found").Err()
		}
		return nil, errx.Wrap(err).AppendMsg("get one failed").Err()
	}
	var t T
	if err := res.Decode(&t); err != nil {
		return nil, errx.Wrap(err).AppendMsg("decode failed").Err()
	}
	return &t, nil
}

func (c *collectionImpl[T]) GetList(ctx context.Context, filter any, opts ...*options.FindOptions) ([]T, errx.Error) {
	if filter == nil {
		filter = bson.M{}
	}
	if len(opts) == 0 {
		opts = append(opts, options.Find().SetSort(defaultSort).SetLimit(defaultLimit))
	}
	var list []T
	cur, err := c.Find(ctx, filter, opts...)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || errors.Is(err, mongo.ErrNilDocument) {
			return list, nil
		}
		return nil, errx.Wrap(err).AppendMsg("find failed").Err()
	}
	defer func() { _ = cur.Close(ctx) }()

	if err = cur.All(ctx, &list); err != nil {
		return nil, errx.Wrap(err).AppendMsg("decode failed").Err()
	}
	return list, nil
}

func (c *collectionImpl[T]) Create(ctx context.Context, entity *T, opts ...*options.InsertOneOptions) errx.Error {
	b := bson.M{}
	if err := c.BsonParser(entity, &b, false); err != nil {
		return err
	}
	var needSetID bool
	{
		id, ok := b["_id"]
		if !ok || isZeroID(id, c.IDType) {
			needSetID = true
			b["_id"] = createID(c.IDType)
		}
	}
	if c.CreatedAtBsonField != "" {
		if t, ok := b[c.CreatedAtBsonField]; !ok || isZeroTime(t, c.TimeType) {
			b[c.CreatedAtBsonField] = createCurrentTime(c.TimeType)
		}
	}
	if c.UpdatedAtBsonField != "" {
		if t, ok := b[c.UpdatedAtBsonField]; !ok || isZeroTime(t, c.TimeType) {
			b[c.UpdatedAtBsonField] = createCurrentTime(c.TimeType)
		}
	}
	if c.VersionBsonField != "" {
		b[c.VersionBsonField] = int64(1)
	}
	res, err := c.InsertOne(ctx, b, opts...)
	if err != nil {
		return errx.Wrap(err).AppendMsg("create failed").Err()
	}
	if needSetID && c.IdSetter != nil {
		c.IdSetter(entity, res.InsertedID)
	}
	if c.CreatedAtBsonField != "" && c.CreatedAtSetter != nil {
		c.CreatedAtSetter(entity, b[c.CreatedAtBsonField])
	}
	if c.UpdatedAtBsonField != "" && c.UpdatedAtSetter != nil {
		c.UpdatedAtSetter(entity, b[c.UpdatedAtBsonField])
	}
	if c.VersionBsonField != "" && c.VersionSetter != nil {
		c.VersionSetter(entity, 1)
	}
	return nil
}

func (c *collectionImpl[T]) UpdateByID(ctx context.Context, data *T, expandedOpts ...*UpdateOptions) errx.Error {
	var opt *UpdateOptions
	if len(expandedOpts) > 0 {
		opt = expandedOpts[0]
	} else {
		opt = Update()
	}
	if opt.IgnoreZeroValue == nil {
		opt.SetIgnoreZeroValue(true)
	}
	ignoreZeroValue := *opt.IgnoreZeroValue
	if opt.OptimisticLock == nil {
		opt.SetOptimisticLock(true)
	}
	optimisticLock := *opt.OptimisticLock
	if opt.Upsert != nil && *opt.Upsert {
		return errx.New("UpdateByID: upsert is not supported")
	}
	var filter, set = bson.M{}, bson.M{}
	if err := c.BsonParser(data, &set, ignoreZeroValue); err != nil {
		return err
	}

	// id
	{
		id, ok := set["_id"]
		if !ok || isZeroID(id, c.IDType) {
			return errx.New("UpdateByID: ID is empty")
		}
		delete(set, "_id")
		filter["_id"] = id
	}

	// version
	if c.VersionBsonField != "" {
		version, exists := set[c.VersionBsonField]
		if !exists {
			if optimisticLock {
				return errx.New("UpdateByID with optimistic lock: the version is missing.")
			}
		} else {
			v, ok := version.(int64)
			if !ok {
				return errx.New("UpdateByID: the version must be int64.")
			}
			if optimisticLock {
				filter[c.VersionBsonField] = v
				v++
				set[c.VersionBsonField] = v
			}
		}
	}

	// updatedAt
	if c.UpdatedAtBsonField != "" {
		set[c.UpdatedAtBsonField] = createCurrentTime(c.TimeType)
	}

	res, err := c.UpdateOne(ctx, filter, bson.M{"$set": set}, opt.UpdateOptions)
	if err != nil {
		return errx.Wrap(err).AppendMsg("UpdateByID failed").Err()
	}
	if res.MatchedCount == 0 {
		return errx.Conflict.WithMsg("UpdateByID failed due to optimistic lock conflict.").Err()
	}
	if c.VersionBsonField != "" && c.VersionSetter != nil {
		c.VersionSetter(data, set[c.VersionBsonField].(int64))
	}
	if c.UpdatedAtBsonField != "" && c.UpdatedAtSetter != nil {
		c.UpdatedAtSetter(data, set[c.UpdatedAtBsonField])
	}
	return nil
}

func (c *collectionImpl[T]) CreateOrUpdateByID(ctx context.Context, data *T, expandedOpts ...*UpdateOptions) (bool, errx.Error) {
	var opt *UpdateOptions
	if len(expandedOpts) > 0 {
		opt = expandedOpts[0]
	} else {
		opt = Update()
	}
	if opt.IgnoreZeroValue == nil {
		opt.SetIgnoreZeroValue(true)
	}
	ignoreZeroValue := *opt.IgnoreZeroValue
	opt.SetUpsert(true)
	var filter, set, setOnInsert = bson.M{}, bson.M{}, bson.M{}
	if err := c.BsonParser(data, &set, ignoreZeroValue); err != nil {
		return false, err
	}

	// id
	{
		id, ok := set["_id"]
		if !ok || isZeroID(id, c.IDType) {
			return false, errx.New("CreateOrUpdateByID: ID is empty")
		}
		delete(set, "_id")
		filter["_id"] = id
		setOnInsert["_id"] = id
	}

	// updatedAt
	if c.UpdatedAtBsonField != "" {
		set[c.UpdatedAtBsonField] = createCurrentTime(c.TimeType)
	}

	// createdAt
	if c.CreatedAtBsonField != "" {
		setOnInsert[c.CreatedAtBsonField] = createCurrentTime(c.TimeType)
		delete(set, c.CreatedAtBsonField)
	}

	update := bson.M{
		"$set":         set,
		"$setOnInsert": setOnInsert,
	}

	if c.VersionBsonField != "" {
		delete(set, c.VersionBsonField)
		update["$inc"] = bson.M{c.VersionBsonField: int64(1)}
	}

	res, err := c.UpdateOne(ctx, filter, update, opt.UpdateOptions)
	if err != nil {
		return false, errx.Wrap(err).AppendMsg("CreateOrUpdateByID failed").Err()
	}

	if res.ModifiedCount == 0 && res.UpsertedCount == 0 {
		return false, errx.New("CreateOrUpdateByID failed: no data is modified or created")
	}

	isNew := res.UpsertedCount > 0
	if isNew {
		if c.CreatedAtBsonField != "" && c.CreatedAtSetter != nil {
			c.CreatedAtSetter(data, setOnInsert[c.CreatedAtBsonField])
		}
		if c.VersionBsonField != "" && c.VersionSetter != nil {
			c.VersionSetter(data, int64(1))
		}
		if c.UpdatedAtBsonField != "" && c.UpdatedAtSetter != nil {
			c.UpdatedAtSetter(data, set[c.UpdatedAtBsonField])
		}
	} else {
		if c.UpdatedAtBsonField != "" && c.UpdatedAtSetter != nil {
			c.UpdatedAtSetter(data, set[c.UpdatedAtBsonField])
		}
	}
	return isNew, nil
}

func (c *collectionImpl[T]) GetAndCreateOrUpdateByID(ctx context.Context, data *T, expandedOpts ...*FindOneAndUpdateOptions) errx.Error {
	var otp *FindOneAndUpdateOptions
	if len(expandedOpts) > 0 {
		otp = expandedOpts[0]
	} else {
		otp = FindOneAndUpdate()
	}
	if otp.IgnoreZeroValue == nil {
		otp.SetIgnoreZeroValue(true)
	}
	ignoreZeroValue := *otp.IgnoreZeroValue
	otp.SetUpsert(true)

	var filter, set, setOnInsert = bson.M{}, bson.M{}, bson.M{}
	if err := c.BsonParser(data, &set, ignoreZeroValue); err != nil {
		return err
	}

	// id
	{
		id, ok := set["_id"]
		if !ok || isZeroID(id, c.IDType) {
			return errx.New("GetAndCreateOrUpdateByID: ID is empty")
		}
		delete(set, "_id")
		filter["_id"] = id
		setOnInsert["_id"] = id
	}

	// updatedAt
	if c.UpdatedAtBsonField != "" {
		set[c.UpdatedAtBsonField] = createCurrentTime(c.TimeType)
	}

	// createdAt
	if c.CreatedAtBsonField != "" {
		setOnInsert[c.CreatedAtBsonField] = createCurrentTime(c.TimeType)
		delete(set, c.CreatedAtBsonField)
	}

	update := bson.M{
		"$set":         set,
		"$setOnInsert": setOnInsert,
	}

	if c.VersionBsonField != "" {
		update["$inc"] = bson.M{c.VersionBsonField: int64(1)}
		delete(set, c.VersionBsonField)
	}

	err := c.Collection.FindOneAndUpdate(ctx, filter, update, otp.FindOneAndUpdateOptions).Decode(data)
	if err != nil {
		return errx.Wrap(err).AppendMsg("GetAndCreateOrUpdateByID failed").Err()
	}
	return nil
}

func (c *collectionImpl[T]) DeleteByID(ctx context.Context, id any, opts ...*options.DeleteOptions) errx.Error {
	if isZeroID(id, c.IDType) {
		return errx.New("DeleteByID: ID is empty")
	}
	_, err := c.DeleteOne(ctx, bson.M{"_id": id}, opts...)
	return errx.Wrap(err).Err()
}

func (c *collectionImpl[T]) CountDocuments(ctx context.Context, filter any, opts ...*options.CountOptions) (*int64, errx.Error) {
	if filter == nil {
		filter = bson.M{}
	}
	// 获取总记录数
	total, err := c.Collection.CountDocuments(ctx, filter, opts...)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return &total, nil
}
