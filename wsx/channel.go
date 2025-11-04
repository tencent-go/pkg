package wsx

import (
	"reflect"

	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/natsx"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
)

type EventChannel interface {
	Topic() string
	MessageType() reflect.Type
	Subscribe(conn Conn) (Subscription, errx.Error)
	Publish(conn Conn, data msgpack.RawMessage) errx.Error
	Description() string
}

type EventChannelBuilder[T any] interface {
	EventChannel
	WithSubscriber(subscriber func(conn Conn, send func(message T) errx.Error) (unsubscribe func(), err errx.Error)) EventChannelBuilder[T]
	WithPublisher(publisher func(conn Conn, data T) errx.Error) EventChannelBuilder[T]
	WithDescription(description string) EventChannelBuilder[T]
}

type Subscription interface {
	Unsubscribe()
}

func NewEventChannel[T any](topic string) EventChannelBuilder[T] {
	return &eventChannelBuilder[T]{
		channelOption: channelOption{topic: topic},
	}
}

type channelOption struct {
	topic       string
	subscriber  func(conn Conn) (Subscription, errx.Error)
	publisher   func(conn Conn, data msgpack.RawMessage) errx.Error
	description string
}

type eventChannelBuilder[T any] struct {
	channelOption
}

func (e *eventChannelBuilder[T]) Subscribe(conn Conn) (Subscription, errx.Error) {
	if e.subscriber == nil {
		return nil, nil
	}
	return e.subscriber(conn)
}

func (e *eventChannelBuilder[T]) Publish(conn Conn, data msgpack.RawMessage) errx.Error {
	if e.publisher == nil {
		return errx.Newf("no handler set")
	}
	return e.publisher(conn, data)
}

func (e *eventChannelBuilder[T]) WithSubscriber(subscriber func(conn Conn, send func(message T) errx.Error) (unsubscribe func(), err errx.Error)) EventChannelBuilder[T] {
	o := e.channelOption
	o.subscriber = func(conn Conn) (Subscription, errx.Error) {
		unsubscribe, err := subscriber(conn, func(message T) errx.Error {
			return conn.Send(e.topic, message)
		})
		if err != nil {
			return nil, err
		}
		return &subscription{unsubscribe: unsubscribe}, nil
	}
	return &eventChannelBuilder[T]{o}
}

func (e *eventChannelBuilder[T]) WithPublisher(publisher func(conn Conn, data T) errx.Error) EventChannelBuilder[T] {
	o := e.channelOption
	o.publisher = func(conn Conn, rawData msgpack.RawMessage) errx.Error {
		var data T
		err := util.Msgpack().Unmarshal(rawData, &data)
		if err != nil {
			return err
		}
		return publisher(conn, data)
	}
	return &eventChannelBuilder[T]{o}
}

func (e *eventChannelBuilder[T]) WithDescription(description string) EventChannelBuilder[T] {
	o := e.channelOption
	o.description = description
	return &eventChannelBuilder[T]{o}
}

func (e *eventChannelBuilder[T]) Topic() string {
	return e.topic
}

func (e *eventChannelBuilder[T]) MessageType() reflect.Type {
	t := reflect.TypeOf(*(new(T)))
	return t
}

func (e *eventChannelBuilder[T]) Description() string {
	return e.description
}

type subscription struct {
	unsubscribe func()
}

func (s *subscription) Unsubscribe() {
	s.unsubscribe()
}

func ChannelWithSubscriberFromNats[T any](subject natsx.Subject[T], topic string) EventChannelBuilder[T] {
	return NewEventChannel[T](topic).WithSubscriber(func(conn Conn, send func(message T) errx.Error) (unsubscribe func(), err errx.Error) {
		sub, err := subject.Subscriber().Subscribe(func(ctx natsx.NatsMessageContext, payload T) errx.Error {
			return send(payload)
		})
		if err != nil {
			return nil, err
		}
		return func() {
			if e := sub.Unsubscribe(); e != nil {
				logrus.WithError(e).WithField("topic", topic).Error("failed to unsubscribe")
			}
		}, nil
	})
}
