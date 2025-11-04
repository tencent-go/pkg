package wsx

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/util"
	"github.com/lxzan/gws"
	"github.com/sirupsen/logrus"
)

type Server interface {
	SetKeepaliveInterval(duration time.Duration)
	SetAuthorizer(func(request *http.Request, storage util.Storage) bool)
	OnConnected(func(conn Conn))
	OnDisconnected(func(conn Conn))
	RegisterSubscribableChannels(events ...EventChannel) //客戶端可訂閱的channel
	RegisterPublishableChannels(events ...EventChannel)  //客戶端可發佈的channel
	ListSubscribableChannels() []EventChannel
	ListPublishableChannels() []EventChannel
	Upgrade(res http.ResponseWriter, req *http.Request)
}

const (
	wrapperKey           = "wrapped"
	subscribeTopicsTopic = "subscribe_topics"
	errorTopic           = "error_notification"
)

func NewServer() Server {
	srv := &server{
		subscribableChannels: make(map[string]EventChannel),
		publishableChannels:  make(map[string]EventChannel),
	}

	var (
		subscribeTopicsChannel = NewEventChannel[SubscribeTopicsEvent](subscribeTopicsTopic)
		errorChannel           = NewEventChannel[ErrorEvent](errorTopic)
	)

	srv.RegisterPublishableChannels(subscribeTopicsChannel.WithPublisher(func(conn Conn, data SubscribeTopicsEvent) errx.Error {
		return srv.updateTopics(conn, data)
	}))
	srv.RegisterSubscribableChannels(subscribeTopicsChannel, errorChannel)
	srv.upgrader = gws.NewUpgrader(srv, nil)
	return srv
}

type server struct {
	keepaliveInterval    time.Duration
	authorize            func(r *http.Request, session util.Storage) bool
	upgrader             *gws.Upgrader
	onConnect            func(conn Conn)
	onDisconnect         func(conn Conn)
	subscribableChannels map[string]EventChannel
	publishableChannels  map[string]EventChannel
}

func (srv *server) OnOpen(socket *gws.Conn) {
	logrus.Debug("websocket connection opened")
	if srv.onConnect != nil {
		srv.onConnect(getWrappedConn(socket))
	}
}

func (srv *server) OnClose(socket *gws.Conn, err error) {
	logrus.WithError(err).Debug("websocket connection closed")
	c := getWrappedConn(socket)
	c.closed = true
	if srv.onDisconnect != nil {
		srv.onDisconnect(c)
	}
	c.Close()
	socket.Session().Delete(wrapperKey)
}

func (srv *server) OnPing(socket *gws.Conn, payload []byte) {
	logrus.Debugf("receive ping from %s", socket.RemoteAddr())
	if err := socket.WritePong(payload); err != nil {
		logrus.WithError(err).Error("pong failed")
		return
	}
}

func (srv *server) OnPong(socket *gws.Conn, payload []byte) {
	logrus.Debugf("received pong: %s", string(payload))
	if srv.keepaliveInterval > 0 {
		if err := socket.SetDeadline(time.Now().Add(srv.keepaliveInterval + time.Second*2)); err != nil {
			logrus.WithError(err).Error("set deadline failed")
			return
		}
	}
}

func (srv *server) updateTopics(conn Conn, ev SubscribeTopicsEvent) errx.Error {
	manager := conn.Subscriptions()
	manager.Retain(ev.Topics)
	for _, topic := range ev.Topics {
		if !manager.Exists(topic) {
			if sub, ok := srv.subscribableChannels[topic]; ok {
				manager.SetIfAbsent(topic, func() (Subscription, bool) {
					res, err := sub.Subscribe(conn)
					if err != nil {
						logrus.WithError(err).Error("subscribe failed")
						return nil, false
					}
					return res, true
				})
			}
		}
	}
	if manager.Exists(subscribeTopicsTopic) {
		return conn.Send(subscribeTopicsTopic, SubscribeTopicsEvent{
			Topics: manager.Topics(),
		})
	}
	return errx.Newf("no topic %s found", subscribeTopicsTopic)
}

func (srv *server) OnMessage(socket *gws.Conn, message *gws.Message) {
	if srv.keepaliveInterval > 0 {
		if err := socket.SetDeadline(time.Now().Add(srv.keepaliveInterval)); err != nil {
			logrus.WithError(err).Error("set deadline failed")
			return
		}
	}
	msg := &receiveMsgWrapper{}
	if err := util.Msgpack().Unmarshal(message.Bytes(), msg); err != nil {
		logrus.WithError(err).Error("unmarshal msg failed")
		return
	}
	conn := getWrappedConn(socket)
	channel, ok := srv.publishableChannels[msg.Topic]
	if !ok {
		errMsg := fmt.Sprintf("unknown topic: %s", msg.Topic)
		_ = conn.Send(errorTopic, ErrorEvent{Message: errMsg})
		return
	}
	if err := channel.Publish(conn, msg.Data); err != nil {
		var errMsg string
		if err.Type() != errx.TypeInternal {
			errMsg = fmt.Sprintf("topic: %s %s", msg.Topic, err.Error())
		} else {
			errMsg = fmt.Sprintf("process message failed, topic: %s", msg.Topic)
		}
		_ = conn.Send(errorTopic, ErrorEvent{Message: errMsg})
		return
	}
}

func (srv *server) Upgrade(res http.ResponseWriter, req *http.Request) {
	wrapped := &connWrapper{}
	if srv.authorize != nil {
		if !srv.authorize(req, wrapped.Storage()) {
			return
		}
	}
	conn, err := srv.upgrader.Upgrade(res, req)
	if err != nil {
		logrus.WithError(err).Error("upgrade failed")
		return
	}
	wrapped.Conn = conn
	conn.Session().Store(wrapperKey, wrapped)
	go func() {
		end := make(chan struct{})
		if srv.keepaliveInterval > 0 {
			ticker := time.NewTicker(srv.keepaliveInterval)
			defer ticker.Stop()
			go func() {
				for {
					select {
					case <-ticker.C:
						if e := conn.WritePing(nil); e != nil {
							logrus.Debugf("write ping failed: %v", e)
							if e = conn.WriteClose(1006, nil); e != nil {
								logrus.WithError(e).Error("write close failed")
							}
							return
						}
					case <-end:
						return
					}
				}
			}()
		}
		conn.ReadLoop()
		close(end)
	}()
}

func (srv *server) OnConnected(fn func(conn Conn)) {
	srv.onConnect = fn
}

func (srv *server) OnDisconnected(fn func(conn Conn)) {
	srv.onDisconnect = fn
}

func (srv *server) SetKeepaliveInterval(interval time.Duration) {
	srv.keepaliveInterval = interval
}

func (srv *server) SetAuthorizer(fn func(r *http.Request, session util.Storage) bool) {
	srv.authorize = fn
}

func (srv *server) RegisterSubscribableChannels(events ...EventChannel) {
	for _, event := range events {
		topic := event.Topic()
		if _, ok := srv.subscribableChannels[topic]; ok {
			logrus.Panicf("duplicate subscribable channel for topic: %s", topic)
		}
		srv.subscribableChannels[topic] = event
	}
}

func (srv *server) RegisterPublishableChannels(events ...EventChannel) {
	for _, event := range events {
		topic := event.Topic()
		if _, ok := srv.publishableChannels[topic]; ok {
			logrus.Panicf("duplicate publishable channel for topic: %s", topic)
		}
		srv.publishableChannels[topic] = event
	}
}

func (srv *server) ListSubscribableChannels() []EventChannel {
	names := make([]string, 0, len(srv.subscribableChannels))
	for name := range srv.subscribableChannels {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]EventChannel, len(names))
	for i, name := range names {
		result[i] = srv.subscribableChannels[name]
	}
	return result
}

func (srv *server) ListPublishableChannels() []EventChannel {
	names := make([]string, 0, len(srv.subscribableChannels))
	for name := range srv.publishableChannels {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]EventChannel, len(names))
	for i, name := range names {
		result[i] = srv.publishableChannels[name]
	}
	return result
}

func getWrappedConn(socket *gws.Conn) *connWrapper {
	conn, _ := socket.Session().Load(wrapperKey)
	return conn.(*connWrapper)
}
