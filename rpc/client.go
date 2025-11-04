package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/etcdx"
	"github.com/tencent-go/pkg/shutdown"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/http2"
)

func getURL(o options) (string, errx.Error) {
	etcd := o.etcd
	if etcd == nil {
		etcd = defaultEtcd()
	}
	d, ok := discovers.Load(etcd)
	if !ok {
		d, _ = discovers.LoadOrLazyStore(etcd, func() *discover {
			return newDiscover(etcd)
		})
	}
	services, ok := d.find(o.path, o.serviceName)
	if !ok {
		return "", errx.Newf("rpc service %s not found", o.path)
	}
	config := configReader.Read()
	var url string
	for i := range services {
		s := services[i]
		if s.namespace == config.Namespace {
			url = fmt.Sprintf("http://%s:%d/%s/%s", s.serviceName, s.port, s.serviceName, s.path)
			break
		}
	}
	if url == "" {
		s := services[0]
		url = fmt.Sprintf("http://%s.%s.svc.%s:%d/%s/%s", s.serviceName, s.namespace, config.ServiceDomainSuffix, s.port, s.serviceName, s.path)
	}
	return url, nil
}

var defaultEtcd = sync.OnceValue(func() *clientv3.Client {
	c := etcdConfigReader.Read()
	if len(c.Endpoints) == 0 {
		return etcdx.DefaultClient()
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		str := strings.Join(c.Endpoints, ",")
		logrus.WithError(err).Panicf("connect rpc default etcd %s failed", str)
	}
	shutdown.OnShutdown(func(ctx context.Context) error {
		defer logrus.Infoln("rpc default etcd connection closed")
		return cli.Close()
	}, true)
	logrus.Infoln("rpc default etcd connected")
	return cli
})

func call[I, O any](o options, ctx ctxx.Context, cmd I) (*O, errx.Error) {
	if o.timeout == 0 {
		o.timeout = defaultClientTimeout
	}
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = ctxx.WithTimeout(ctx, o.timeout)
		defer cancel()
	}
	url, err := getURL(o)
	if err != nil {
		return nil, err
	}
	body, err := NewRequestBody(cmd)
	if err != nil {
		return nil, err
	}
	req, e := http.NewRequestWithContext(ctx, "POST", url, body)
	if e != nil {
		return nil, errx.Wrap(e).AppendMsgf("create request failed. target: %s", url).Err()
	}
	WriteRequestHeader(ctx, req)
	res, e := httpClient.Do(req)
	if e != nil {
		return nil, errx.Wrap(e).AppendMsg("request rpc method failed").Err()
	}
	return ParseResponse[O](res)
}

var discovers util.LazyMap[*clientv3.Client, *discover]

func newDiscover(etcd *clientv3.Client) *discover {
	d := &discover{}
	reload := func() {
		res, err := etcd.Get(context.Background(), etcdPrefix, clientv3.WithPrefix())
		if err != nil {
			logrus.WithError(err).Warn("rpc discover get etcd failed")
		}
		byPath := map[string][]*onlineService{}
		for _, kv := range res.Kvs {
			k := string(kv.Key)
			v := string(kv.Value)
			if k != "" && v != "" {
				s, ok := parseEtcdDataItem(k, v)
				if ok {
					byPath[s.path] = append(byPath[s.path], s)
				}
			}
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		d.onlineServicesByPath = byPath
	}
	reload()
	debouncer := util.NewDebouncer(reload, time.Second)
	go func() {
		for {
			ch := etcd.Watch(context.Background(), etcdPrefix, clientv3.WithPrefix())
			for range ch {
				debouncer.Trigger()
			}
		}
	}()
	return d
}

type discover struct {
	onlineServicesByPath map[string][]*onlineService
	mu                   sync.RWMutex
}

type onlineService struct {
	serviceName string
	port        int
	path        string
	namespace   string
}

func (d *discover) find(path string, serviceName string) ([]*onlineService, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	list, ok := d.onlineServicesByPath[path]
	if !ok {
		return nil, false
	}
	if serviceName != "" {
		var filtered []*onlineService
		for i := range list {
			item := list[i]
			if item.serviceName == serviceName {
				filtered = append(filtered, item)
			}
		}
		return filtered, len(filtered) > 0
	}
	return list, ok
}

func parseEtcdDataItem(key, value string) (*onlineService, bool) {
	keyParts := strings.Split(strings.Trim(key, "/"), "/")
	if len(keyParts) < 4 {
		return nil, false
	}
	keyParts = keyParts[1 : len(keyParts)-1]
	valueParts := strings.Split(value, ",")
	if len(valueParts) != 2 {
		return nil, false
	}
	serviceName := keyParts[0]
	path := strings.Join(keyParts[1:], "/")
	namespace := valueParts[0]
	port, err := strconv.Atoi(valueParts[1])
	if err != nil {
		return nil, false
	}
	return &onlineService{
		serviceName: serviceName,
		path:        path,
		namespace:   namespace,
		port:        port,
	}, true
}

var httpClient = &http.Client{
	Transport: &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{}
			return dialer.DialContext(ctx, network, addr)
		},
	},
}
