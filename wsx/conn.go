package wsx

import (
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/util"
	"github.com/lxzan/gws"
	"github.com/sirupsen/logrus"
	"net"
	"sort"
	"sync"
)

type Conn interface {
	Send(topic string, data any) errx.Error
	AsyncSend(topic string, data any, callback func(err errx.Error))
	RemoteAddr() net.Addr
	Subscriptions() SubscriptionManager
	Storage() util.Storage
	Close()
}

type connWrapper struct {
	*gws.Conn
	subscriptions subscriptionManager
	storage       sync.Map
	closed        bool
}

func (c *connWrapper) Storage() util.Storage {
	return &c.storage
}

func (c *connWrapper) Subscriptions() SubscriptionManager {
	return &c.subscriptions
}

func (c *connWrapper) Send(topic string, data any) errx.Error {
	m := sendMsgWrapper{
		Topic: topic,
		Data:  data,
	}
	d, err := util.Msgpack().Marshal(m)
	if err != nil {
		return err
	}
	if e := c.WriteMessage(gws.OpcodeBinary, d); e != nil {
		return errx.Wrap(e).Err()
	}
	return nil
}

func (c *connWrapper) AsyncSend(topic string, data any, callback func(err errx.Error)) {
	m := sendMsgWrapper{
		Topic: topic,
		Data:  data,
	}
	d, err := util.Msgpack().Marshal(m)
	if err != nil {
		if callback != nil {
			callback(err)
		}
	}
	c.WriteAsync(gws.OpcodeBinary, d, func(err error) {
		if callback != nil {
			callback(errx.Wrap(err).Err())
		}
	})
}

func (c *connWrapper) Close() {
	if !c.closed {
		c.closed = true
		if e := c.WriteClose(1000, nil); e != nil {
			logrus.WithError(e).Error("write close failed")
		}
	}
	c.subscriptions.ClearAll()
}

type SubscriptionManager interface {
	SetIfAbsent(topic string, factory func() (Subscription, bool))
	Topics() []string
	Remove(topic string)
	Exists(topic string) bool
	Retain(topics []string)
	ClearAll()
}

type subscriptionManager struct {
	mu            sync.RWMutex
	subscriptions map[string]Subscription
}

func (s *subscriptionManager) SetIfAbsent(topic string, factory func() (Subscription, bool)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subscriptions == nil {
		s.subscriptions = make(map[string]Subscription)
	}
	if _, ok := s.subscriptions[topic]; ok {
		return
	}
	if factory != nil {
		res, ok := factory()
		if ok {
			s.subscriptions[topic] = res
		}
	}
}

func (s *subscriptionManager) Exists(topic string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.subscriptions == nil {
		return false
	}
	_, ok := s.subscriptions[topic]
	return ok
}

func (s *subscriptionManager) Remove(topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.subscriptions[topic]
	if !ok {
		return
	}
	v.Unsubscribe()
	delete(s.subscriptions, topic)
}

func (s *subscriptionManager) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subscriptions == nil {
		return
	}
	for _, sub := range s.subscriptions {
		sub.Unsubscribe()
	}
	s.subscriptions = nil
}

func (s *subscriptionManager) Retain(topics []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}
	var shouldUnsubscribeList []Subscription
	for topic, sub := range s.subscriptions {
		if !topicMap[topic] {
			if sub != nil {
				shouldUnsubscribeList = append(shouldUnsubscribeList, sub)
			}
			delete(s.subscriptions, topic)
		}
	}
	if len(shouldUnsubscribeList) > 0 {
		go func(list []Subscription) {
			for _, sub := range list {
				sub.Unsubscribe()
			}
		}(shouldUnsubscribeList)
	}
}

func (s *subscriptionManager) Topics() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	topics := make([]string, 0, len(s.subscriptions))
	for topic := range s.subscriptions {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics
}
