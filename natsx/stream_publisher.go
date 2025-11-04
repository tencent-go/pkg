package natsx

import (
	"github.com/tencent-go/pkg/ctxx"
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/util"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type StreamPublisher[T any] interface {
	Publish(ctx ctxx.Context, msg T, opts ...jetstream.PublishOpt) (*jetstream.PubAck, errx.Error)
	PublishAsync(ctx ctxx.Context, msg T, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, errx.Error)
}

type streamPublisher[T any] struct {
	subject string
	js      jetstream.JetStream
}

func newStreamPublisher[T any](js jetstream.JetStream, subject string, args ...string) (StreamPublisher[T], errx.Error) {
	if js == nil {
		return nil, errx.Newf("stream is nil")
	}
	subject, missingPlaceholders := replaceSubjectPlaceholders(subject, args...)
	if len(missingPlaceholders) > 0 {
		return nil, errx.Newf("missing placeholders: %v", missingPlaceholders)
	}
	return &streamPublisher[T]{subject: subject, js: js}, nil
}

func (p *streamPublisher[T]) Publish(ctx ctxx.Context, msg T, opts ...jetstream.PublishOpt) (*jetstream.PubAck, errx.Error) {
	data, err := util.Json().Marshal(msg)
	if err != nil {
		return nil, err
	}
	natsMsg := &nats.Msg{
		Subject: p.subject,
		Data:    data,
		Header:  newNatsHeader(ctx),
	}
	opts = append(opts, jetstream.WithRetryAttempts(50))
	var msgId string
	if msgIdGetter, ok := any(msg).(MsgIdGetter); ok {
		msgId = msgIdGetter.MsgID()
		if msgId != "" {
			natsMsg.Header.Set("Nats-Msg-Id", msgId)
		}
	}
	res, e := p.js.PublishMsg(ctx, natsMsg, opts...)
	if e != nil {
		if msgId != "" && res != nil && res.Duplicate {
			return res, nil
		}
		return nil, errx.Wrap(e).Err()
	}
	return res, nil
}

func (p *streamPublisher[T]) PublishAsync(ctx ctxx.Context, msg T, opts ...jetstream.PublishOpt) (jetstream.PubAckFuture, errx.Error) {
	data, err := util.Json().Marshal(msg)
	if err != nil {
		return nil, err
	}
	natsMsg := &nats.Msg{
		Subject: p.subject,
		Data:    data,
		Header:  newNatsHeader(ctx),
	}
	msgId, ok := any(msg).(MsgIdGetter)
	if ok {
		natsMsg.Header.Set("Nats-Msg-Id", msgId.MsgID())
	}
	res, e := p.js.PublishMsgAsync(natsMsg, opts...)
	if e != nil {
		return nil, errx.Wrap(e).Err()
	}
	return res, nil
}
