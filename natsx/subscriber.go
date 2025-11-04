package natsx

import (
	"context"
	"time"

	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type Subscriber[T any] interface {
	Subscribe(callback func(ctx NatsMessageContext, payload T) errx.Error) (*nats.Subscription, errx.Error)
	SubscribeSync() (*nats.Subscription, errx.Error)
	QueueSubscribe(queue string, callback func(ctx NatsMessageContext, payload T) errx.Error) (*nats.Subscription, errx.Error)
	QueueSubscribeSync(queue string) (*nats.Subscription, errx.Error)
}

type NatsMessageContext interface {
	ctxx.Context
	Subject() string
	Reply() string
	Headers() nats.Header
	Data() []byte
	Sub() *nats.Subscription
}

func newSubscriber[T any](nc *nats.Conn, subject string) *subscriber[T] {
	return &subscriber[T]{subject: subject, nc: nc}
}

type subscriber[T any] struct {
	subject string
	nc      *nats.Conn
}

func (s *subscriber[T]) Subscribe(callback func(ctx NatsMessageContext, payload T) errx.Error) (*nats.Subscription, errx.Error) {
	sub, e := s.nc.Subscribe(s.subject, s.newHandler(callback))
	if e != nil {
		return nil, errx.Wrap(e).Err()
	}
	return sub, nil
}

func (s *subscriber[T]) SubscribeSync() (*nats.Subscription, errx.Error) {
	res, err := s.nc.SubscribeSync(s.subject)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return res, nil
}

func (s *subscriber[T]) QueueSubscribe(queue string, callback func(ctx NatsMessageContext, payload T) errx.Error) (*nats.Subscription, errx.Error) {
	sub, e := s.nc.QueueSubscribe(s.subject, queue, s.newHandler(callback))
	if e != nil {
		return nil, errx.Wrap(e).Err()
	}
	return sub, nil
}

func (s *subscriber[T]) QueueSubscribeSync(queue string) (*nats.Subscription, errx.Error) {
	res, err := s.nc.QueueSubscribeSync(s.subject, queue)
	if err != nil {
		return nil, errx.Wrap(err).Err()
	}
	return res, nil
}

func (s *subscriber[T]) newHandler(callback func(ctx NatsMessageContext, payload T) errx.Error) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		startTime := time.Now()
		headers := msg.Header
		traceId, e := types.NewIDFromString(headers.Get("traceId"))
		if e != nil {
			logrus.WithError(e).Error("invalid traceId")
		}
		l := types.Locale(headers.Get("locale"))
		ctx := ctxx.WithMetadata(context.Background(), ctxx.Metadata{
			TraceID:  traceId,
			Operator: headers.Get("operator"),
			Caller:   headers.Get("caller"),
			Locale:   l,
		})
		log := logrus.WithContext(ctx).WithField("subject", s.subject)
		data := msg.Data
		if logrus.GetLevel() >= logrus.DebugLevel {
			log = log.WithField("message", string(data))
		}
		payload := new(T)
		if err := util.Json().Unmarshal(data, payload); e != nil {
			log.WithError(err).Error("unmarshal payload failed")
			return
		}
		msgCtx := &natsMessageContext{
			Context: ctx,
			msg:     msg,
		}
		err := callback(msgCtx, *payload)
		log = log.WithField("duration", time.Since(startTime).String())
		if err != nil {
			log.WithError(err).Error("handle event failed")
		} else {
			log.Info("handle event successful")
		}
	}
}

type natsMessageContext struct {
	ctxx.Context
	msg *nats.Msg
}

func (n *natsMessageContext) Reply() string {
	return n.msg.Reply
}

func (n *natsMessageContext) Headers() nats.Header {
	return n.msg.Header
}

func (n *natsMessageContext) Data() []byte {
	return n.msg.Data
}

func (n *natsMessageContext) Sub() *nats.Subscription {
	return n.msg.Sub
}

func (n *natsMessageContext) Subject() string {
	return n.msg.Subject
}
