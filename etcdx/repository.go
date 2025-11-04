package etcdx

import (
	"context"
	"encoding/json"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/env"

	"github.com/tencent-go/pkg/errx"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

var config = env.BaseConfigReaderBuilder.Build().Read()

type format int

const (
	formatYaml format = iota
	formatJson
	formatToml
	formatString
	formatInt
)

const repositoryKeyPrefix = "/repository"

type RepositoryBuilder[T any] interface {
	WithClient(c *clientv3.Client) RepositoryBuilder[T]
	WithKey(key string) RepositoryBuilder[T]
	WithFormatJson() RepositoryBuilder[T]
	WithFormatYaml() RepositoryBuilder[T]
	WithWatch() RepositoryBuilder[T]
	WithDefaultData(data T) RepositoryBuilder[T]
	Build() Repository[T]
}

type Repository[T any] interface {
	Get() T
	Set(data T) errx.Error
	OnChange(func(data T)) func()
}

type options[T any] struct {
	client      *clientv3.Client
	format      format
	key         string
	watch       bool
	defaultData *T
}
type repositoryBuilder[T any] struct {
	options[T]
	repository *repository[T]
}

func (r *repositoryBuilder[T]) WithClient(c *clientv3.Client) RepositoryBuilder[T] {
	o := r.options
	if c != nil {
		o.client = c
	}
	return &repositoryBuilder[T]{
		options: o,
	}
}

func (r *repositoryBuilder[T]) WithKey(key string) RepositoryBuilder[T] {
	o := r.options
	if key != "" {
		o.key = key
	}
	return &repositoryBuilder[T]{
		options: o,
	}
}

func (r *repositoryBuilder[T]) WithFormatJson() RepositoryBuilder[T] {
	o := r.options
	o.format = formatJson
	return &repositoryBuilder[T]{
		options: o,
	}
}

func (r *repositoryBuilder[T]) WithFormatYaml() RepositoryBuilder[T] {
	o := r.options
	o.format = formatYaml
	return &repositoryBuilder[T]{
		options: o,
	}
}

func (r *repositoryBuilder[T]) WithWatch() RepositoryBuilder[T] {
	o := r.options
	o.watch = true
	return &repositoryBuilder[T]{
		options: o,
	}
}

func (r *repositoryBuilder[T]) WithDefaultData(data T) RepositoryBuilder[T] {
	o := r.options
	o.defaultData = &data
	return &repositoryBuilder[T]{
		options: o,
	}
}

func sanitizeTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	name := t.Name()
	if name != "" {
		return name
	}
	name = t.String()
	if name != "" {
		return name
	}
	return "anonymous"
}

func generateEtcdKey[T any](prefix string) string {
	var v T
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	pkgPath := t.PkgPath()
	pathParts := strings.Split(pkgPath, "/")
	module := pathParts[len(pathParts)-2]
	pkg := pathParts[len(pathParts)-1]
	typeFull := sanitizeTypeName(t)
	return path.Join(prefix, module, pkg, typeFull)
}

func (r *repositoryBuilder[T]) Build() Repository[T] {
	if r.repository != nil {
		return r.repository
	}
	if r.key == "" {
		r.key = generateEtcdKey[T](repositoryKeyPrefix)

	} else {
		r.key = path.Join(repositoryKeyPrefix, r.key)
	}
	if r.client == nil {
		r.client = DefaultClient()
	}
	repo := &repository[T]{
		options: r.options,
		watcher: make(map[string]func(T)),
	}
	if r.defaultData != nil && config.Initializing {
		ctx := ctxx.WithMetadata(context.Background(), ctxx.Metadata{Operator: "initiator"})
		key := path.Join("/initialize/", r.key)
		if key != "" {
			res, err := DefaultClient().Get(ctx, key)
			if err != nil {
				logrus.Fatalf("failed to get etcd key %s: %v", key, err)
			}
			if res.Count == 0 {
				if err := repo.Set(*r.defaultData); err != nil {
					logrus.WithError(err).Errorf("set default data for repository key %s failed", r.key)
					return repo
				}
			}
			defer func() {
				if r := recover(); r != nil {
					logrus.Fatalf("panic on init %s: %v", key, r)
					return
				}
				if key != "" {
					v := time.Now().Format(time.RFC3339)
					_, err := DefaultClient().Put(context.Background(), key, v)
					if err != nil {
						logrus.Errorf("failed to put etcd key %s: %v", key, err)
					}
				}
			}()
		}

	}
	r.repository = repo
	return repo
}

func NewRepositoryBuilder[T any]() RepositoryBuilder[T] {
	return &repositoryBuilder[T]{}
}

type repository[T any] struct {
	options   options[T]
	mu        sync.RWMutex
	watching  bool
	data      *T
	onceWatch sync.Once
	watcher   map[string]func(T)
}

func (r *repository[T]) Get() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var data = r.data
	if !r.watching {
		data = r.load()
	}
	if data == nil {
		logrus.Errorf("get defaultData from etcd failed: key=%s", r.options.key)
		return *(new(T))
	}
	return *data
}

func (r *repository[T]) Set(data T) errx.Error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.set(data)
}

func (r *repository[T]) OnChange(f func(T)) func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.startWatch()
	k := uuid.New().String()
	r.watcher[k] = f
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.watcher, k)
	}
}

func (r *repository[T]) startWatch() {
	r.onceWatch.Do(func() {
		r.options.watch = true
		r.watching = true
		r.data = r.load()
		go func() {
			watchChan := r.options.client.Watch(context.Background(), r.options.key)
			log := logrus.WithField("key", r.options.key)
			for watchResp := range watchChan {
				for _, ev := range watchResp.Events {
					if ev.Kv == nil {
						log.Error("kv empty")
						continue
					}
					if ev.Type != mvccpb.PUT {
						log.Error("not put")
						continue
					}
					l := log.WithFields(logrus.Fields{
						"version":         ev.Kv.Version,
						"create_revision": ev.Kv.CreateRevision,
						"mod_revision":    ev.Kv.ModRevision,
						"value":           string(ev.Kv.Value),
					})
					c, err := r.unmarshal(ev.Kv.Value)
					if err != nil {
						l.WithError(err).Error("defaultData unmarshal failed")
					}
					r.data = c
					l.Info("defaultData updated")
					r.mu.RLock()
					for _, f := range r.watcher {
						f(*c)
					}
					r.mu.RUnlock()
				}
			}
		}()
	})
}

func (r *repository[T]) load() *T {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	res, err := r.options.client.Get(ctx, r.options.key)
	if err != nil {
		logrus.WithError(err).Panicf("get config failed")
	}
	if len(res.Kvs) == 0 {
		return nil
	}
	data := res.Kvs[0].Value
	c, err := r.unmarshal(data)
	if err != nil {
		logrus.WithError(err).Panicf("value unmarshal failed: key %s", r.options.key)
	}
	return c
}

func (r *repository[T]) set(data T) errx.Error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	out, e := r.marshal(data)
	if e != nil {
		return e
	}
	if _, err := r.options.client.Put(ctx, r.options.key, string(out)); err != nil {
		return errx.Wrap(err).Err()
	}
	r.data = &data
	return nil
}

func (r *repository[T]) unmarshal(data []byte) (*T, errx.Error) {
	var c T
	switch r.options.format {
	case formatJson:
		err := json.Unmarshal(data, &c)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}
		return &c, nil
	case formatYaml:
		err := yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}
		return &c, nil
	default:
		return nil, errx.New("unknown format")
	}
}

func (r *repository[T]) marshal(data T) ([]byte, errx.Error) {
	switch r.options.format {
	case formatJson:
		res, err := json.Marshal(data)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}
		return res, nil
	case formatYaml:
		res, err := yaml.Marshal(data)
		if err != nil {
			return nil, errx.Wrap(err).Err()
		}
		return res, nil
	default:
		return nil, errx.New("unknown format")
	}
}
