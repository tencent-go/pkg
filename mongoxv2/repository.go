package mongox

import (
	"context"
	"reflect"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/initialization"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository[T any] interface {
	WithCollectionName(string) Repository[T]
	WithIndexes([]mongo.IndexModel) Repository[T]
	WithDatabase(string) Repository[T]
	WithClient(func() *mongo.Client) Repository[T]
	Collection(opts ...*options.CollectionOptions) Collection[T]
}

type EntityName interface {
	EntityName() string
}

func Repo[T any](opts ...Option[T]) Repository[T] {
	conf := &EntityConfig[T]{}
	for _, opt := range opts {
		opt(conf)
	}
	fillDefaultCollectionConfig(conf)
	return &repository[T]{config: conf}
}

type repository[T any] struct {
	dbName   string
	collName string
	indexes  []mongo.IndexModel
	cli      func() *mongo.Client
	config   *EntityConfig[T]
}

func (r *repository[T]) copy() *repository[T] {
	return &repository[T]{
		dbName:   r.dbName,
		collName: r.collName,
		indexes:  r.indexes,
		cli:      r.cli,
		config:   r.config,
	}
}

func (r *repository[T]) fillDefaults() *repository[T] {
	c := r.copy()
	if c.cli == nil {
		c.cli = GetDefaultClient
	}
	if c.dbName == "" {
		c.dbName = configReader.Read().DefaultDBName
	}
	if c.collName == "" {
		if en, ok := any(new(T)).(EntityName); ok {
			c.collName = en.EntityName()
		} else {
			t := reflect.TypeOf(*(new(T)))
			logrus.Fatalf("empty mongodb collection name: %s.%s", t.PkgPath(), t.Name())
		}
	}
	return c
}

func (r *repository[T]) WithCollectionName(s string) Repository[T] {
	c := r.copy()
	c.collName = s
	return c
}

func (r *repository[T]) WithClient(cli func() *mongo.Client) Repository[T] {
	c := r.copy()
	c.cli = cli
	return c
}

func (r *repository[T]) WithIndexes(models []mongo.IndexModel) Repository[T] {
	c := r.copy()
	c.indexes = models
	return c
}

func (r *repository[T]) WithDatabase(s string) Repository[T] {
	c := r.copy()
	c.dbName = s
	return c
}

func (r *repository[T]) Collection(opts ...*options.CollectionOptions) Collection[T] {
	repo := r.fillDefaults()
	c := repo.cli().Database(repo.dbName).Collection(repo.collName, opts...)
	if len(repo.indexes) > 0 {
		initialization.Register(func(ctx ctxx.Context) {
			resetIndexes(ctx, c, repo.indexes)
		}, "mongodb", r.dbName, r.collName)
		repo.indexes = nil
	}
	return &collectionImpl[T]{Collection: c, EntityConfig: r.config}
}

func resetIndexes(_ctx ctxx.Context, c *mongo.Collection, indexes []mongo.IndexModel) {
	ctx, cancel := context.WithTimeout(_ctx, 20*time.Minute)
	defer cancel()
	// 獲取現有的索引
	existingIndexesCursor, err := c.Indexes().List(ctx)
	if err != nil {
		logrus.Panicf("Failed to list indexes: %v", err)
	}

	var existingIndexes []bson.M
	if err = existingIndexesCursor.All(ctx, &existingIndexes); err != nil {
		logrus.Panicf("Failed to decode existing indexes: %v", err)
	}
	log := logrus.WithField("database", c.Database().Name()).WithField("collection", c.Name())

	logrus.Infof("processing database %s collection %s indexes init", c.Database().Name(), c.Name())
	// 刪除舊的索引
	removedCount := 0
	for _, idx := range existingIndexes {
		if name, ok := idx["name"].(string); ok && name != "_id_" {
			_, err = c.Indexes().DropOne(ctx, name)
			if err != nil {
				log.Panicf("Failed to drop index %s: %v", name, err)
			}
			removedCount++
		}
	}
	if len(indexes) == 0 {
		log.Infof("indexes init done, removed %d indexes", removedCount)
		return
	}
	_, err = c.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		log.Panicf("Failed to create indexes: %v", err)
	}
	log.Infof("indexes init done, removed %d indexes and created %d", removedCount, len(indexes))
}
