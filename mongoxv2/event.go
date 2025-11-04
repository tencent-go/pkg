package mongox

import (
	"context"
	"encoding/base64"
	"path"
	"strings"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/etcdx"
	"github.com/tencent-go/pkg/keylocker"
	"github.com/tencent-go/pkg/types"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (c *collectionImpl[T]) Watch(ctx context.Context, pipeline interface{}, cb func(ctx ctxx.Context, ev ChangeEventWithDoc[T]) errx.Error, opts ...*ChangeStreamOptions) (func(), errx.Error) {
	var opt *ChangeStreamOptions
	if len(opts) > 0 {
		opt = opts[0]
	} else {
		opt = ChangeStream()
	}
	if opt.FullDocument == nil {
		opt.SetFullDocument(options.UpdateLookup)
	}
	var consumerName string
	if opt.ConsumerName != nil {
		consumerName = *opt.ConsumerName
	}
	var persistCursor bool
	if opt.PersistCursor != nil {
		persistCursor = *opt.PersistCursor
	}
	var resumeTokenKey string
	if persistCursor && consumerName != "" {
		resumeTokenKey = path.Join("/mongo-stream/resume-token", c.Database().Name(), c.Name(), consumerName)
	}
	if resumeTokenKey != "" {
		resumeToken, err := getResumeToken(ctx, resumeTokenKey)
		if err != nil {
			return nil, errx.Wrap(err).WithMsg("get resume token failed").Err()
		}
		if len(resumeToken) > 0 {
			opt.SetResumeAfter(resumeToken)
		}
	}
	if pipeline == nil {
		pipeline = mongo.Pipeline{}
	}
	stream, err := c.Collection.Watch(ctx, pipeline, opt.ChangeStreamOptions)
	if err != nil {
		return nil, errx.Wrap(err).WithMsg("Watch failed").Err()
	}
	watchingCtx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() { _ = stream.Close(context.Background()) }()
		if consumerName != "" {
			k := strings.Join([]string{"mongo-stream", c.Database().Name(), c.Name(), consumerName}, ":")
			l := keylocker.Etcd(k)
			l.Lock()
			defer l.Unlock()
		}
		for stream.Next(watchingCtx) {
			eventCtx := ctxx.WithMetadata(watchingCtx, ctxx.Metadata{Operator: consumerName, TraceID: types.ID(stream.ID())})
			var ev ChangeEventWithDoc[T]
			log := logrus.WithField("database", c.Database().Name()).WithField("collection", c.Name()).WithField("consumer", consumerName)
			if e := stream.Decode(&ev); e != nil {
				log.WithError(e).Panic("Mongodb watcher decode failed")
			}
			e := cb(eventCtx, ev)
			if e != nil {
				log.WithError(e).WithField("event", ev).Panic("Mongodb watcher process failed")
			} else {
				if resumeTokenKey != "" {
					if e = setResumeToken(ctx, resumeTokenKey, ev.ID); e != nil {
						log.WithError(e).Error("set resume token failed")
					}
				}
				if logrus.GetLevel() > logrus.InfoLevel {
					log.WithField("event", ev).Debug("receive mongodb watch event")
				} else {
					log.Info("receive mongodb watch event")
				}
			}
		}
	}()

	return cancel, nil
}

func getResumeToken(ctx context.Context, key string) (bson.Raw, errx.Error) {
	res, err := etcdx.DefaultClient().Get(ctx, key)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	if len(res.Kvs) == 0 {
		return nil, nil
	}
	data, err := base64.StdEncoding.DecodeString(string(res.Kvs[0].Value))
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return bson.Raw(data), nil
}

func setResumeToken(ctx context.Context, key string, raw bson.Raw) errx.Error {
	str := base64.StdEncoding.EncodeToString(raw)
	_, err := etcdx.DefaultClient().Put(ctx, key, str)
	return errx.Wrap(err).Err()
}

type ChangeEventWithDoc[T any] struct {
	ID            bson.Raw           `bson:"_id"`                         // 事件 ID
	OperationType ChangeEventType    `bson:"operationType"`               // 操作類型（例如 "insert", "update", "delete" 等）
	Namespace     NamespaceDetail    `bson:"ns"`                          // 命名空間，包含資料庫和集合名稱
	DocumentKey   bson.M             `bson:"documentKey"`                 // 被改變文檔的唯一標識符
	FullDocument  *T                 `bson:"fullDocument,omitempty"`      // 完整文檔（如果有）
	UpdateDesc    *UpdateDescription `bson:"updateDescription,omitempty"` // 更新描述，僅對更新操作存在
}

type NamespaceDetail struct {
	DB   string `bson:"db"`   // 資料庫名稱
	Coll string `bson:"coll"` // 集合名稱
}

type UpdateDescription struct {
	UpdatedFields map[string]interface{} `bson:"updatedFields"` // 更新的字段
	RemovedFields []string               `bson:"removedFields"` // 被移除的字段
}

type ChangeEventType string

const (
	ChangeEventTypeInsert       ChangeEventType = "insert"       // 插入一個新文件
	ChangeEventTypeUpdate       ChangeEventType = "update"       // 更新一個現有文件
	ChangeEventTypeReplace      ChangeEventType = "replace"      // 完全取代一個現有文件
	ChangeEventTypeDelete       ChangeEventType = "delete"       // 刪除一個文件
	ChangeEventTypeInvalidate   ChangeEventType = "invalidate"   // 指示 Change Stream 已失效（例如當集合被刪除時）
	ChangeEventTypeDrop         ChangeEventType = "drop"         // 集合被刪除
	ChangeEventTypeRename       ChangeEventType = "rename"       // 集合被重命名
	ChangeEventTypeDropDatabase ChangeEventType = "dropDatabase" // 資料庫被刪除
	ChangeEventTypeModify       ChangeEventType = "modify"       // 在分片集群的分片鍵修改時出現
)
