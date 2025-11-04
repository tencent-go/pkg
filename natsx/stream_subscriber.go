package natsx

import (
	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/types"
	"github.com/tencent-go/pkg/util"
	"context"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sirupsen/logrus"
	"time"
)

type StreamSubscriber[T any] interface {
	Subscribe(callback func(ctx StreamMessageContext, payload T) errx.Error, opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, errx.Error)
}

type StreamMessageContext interface {
	jetstream.Msg
	ctxx.Context
}

type streamSubscriber[T any] struct {
	consumer jetstream.Consumer
	timeout  time.Duration
}

func newStreamSubscriber[T any](jc jetstream.Consumer, timeout time.Duration) StreamSubscriber[T] {
	return &streamSubscriber[T]{consumer: jc, timeout: timeout}
}

func (s *streamSubscriber[T]) Subscribe(callback func(ctx StreamMessageContext, payload T) errx.Error, opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, errx.Error) {
	res, e := s.consumer.Consume(s.newHandler(callback), opts...)
	if e != nil {
		return nil, errx.Wrap(e).Err()
	}
	return res, nil
}

func (s *streamSubscriber[T]) newHandler(callback func(ctx StreamMessageContext, payload T) errx.Error) jetstream.MessageHandler {
	ackWait := s.consumer.CachedInfo().Config.AckWait
	backoff := s.consumer.CachedInfo().Config.BackOff
	return func(msg jetstream.Msg) {
		startTime := time.Now()
		headers := msg.Headers()
		log := logrus.WithField("subject", msg.Subject())
		traceId, err := types.NewIDFromString(headers.Get("traceId"))
		if err != nil {
			log.WithError(err).Error("invalid traceId")
		}
		metadata, e := msg.Metadata()
		if e != nil {
			log.WithError(e).Error("failed to get metadata")
			return
		}
		log = log.WithField("consumer", metadata.Consumer).WithField("stream", metadata.Stream)
		msgTimeout := getMessageTimeout(ackWait, backoff, metadata.NumDelivered)
		l := types.Locale(msg.Headers().Get("locale"))
		_ctx := context.Background()
		if s.timeout > 0 {
			c, cancel := context.WithTimeout(_ctx, s.timeout)
			defer cancel()
			_ctx = c
		} else {
			c, cancel := context.WithCancel(_ctx)
			defer cancel()
			_ctx = c
		}
		if s.timeout == 0 && s.timeout > msgTimeout && msgTimeout > 2*time.Second {
			t := time.NewTicker(msgTimeout - time.Second)
			go func() {
				for {
					select {
					case <-t.C:
						if e := msg.InProgress(); e != nil {
							log.WithError(e).Error("Renewal failed")
							return
						}
					case <-_ctx.Done():
						return
					}
				}
			}()
		}
		ctx := ctxx.WithMetadata(_ctx, ctxx.Metadata{
			TraceID:  traceId,
			Operator: headers.Get("operator"),
			Caller:   headers.Get("caller"),
			Locale:   l,
		})

		log = log.WithContext(ctx)
		data := msg.Data()

		defer func() {
			log = log.WithField("duration", time.Since(startTime).String())
			if err != nil {
				log.WithError(err).Error("process event failed")
				if e = msg.InProgress(); e != nil {
					log.WithError(e).Error("failed to in-progress")
				}
			} else {
				log.Info("process event successful")
				if e = msg.Ack(); e != nil {
					log.WithError(e).Error("failed to ack")
				}
			}
		}()
		if logrus.GetLevel() >= logrus.DebugLevel {
			log = log.WithField("message", string(data))
		}
		payload := new(T)
		if err = util.Json().Unmarshal(data, payload); err != nil {
			return
		}
		mCtx := &streamMessageContext{
			Msg:     msg,
			Context: ctx,
		}
		err = callback(mCtx, *payload)
	}
}

type streamMessageContext struct {
	ctxx.Context
	jetstream.Msg
}

func getMessageTimeout(ackWait time.Duration, backoff []time.Duration, numDelivered uint64) time.Duration {
	if len(backoff) > 0 {
		l := len(backoff)
		num := int(numDelivered)
		if num >= l {
			ackWait = backoff[l-1]
		} else {
			ackWait = backoff[num-1]
		}
	} else {
		if ackWait == 0 {
			ackWait = 30 * time.Second
		}
	}
	return ackWait
}
